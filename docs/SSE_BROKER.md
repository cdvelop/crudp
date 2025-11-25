# SSE Broker: Bidirectional HTTP Communication

## Overview

SSE Broker enables bidirectional communication between client (WASM) and server using HTTP/SSE, replacing WebSocket dependency. CRUDP manages queuing, batching, and automatic reconnection on both sides.

**Key Features:**
- **Bidirectional:** Client initiates requests, server streams responses via SSE
- **Batch Accumulation:** Server waits configurable time to batch responses (mailman pattern)
- **User-Based Routing:** Responses routed per-user, per-role, or broadcast
- **Automatic Queueing:** Failed packets queue locally and retry on reconnect
- **Optional Async:** Handlers opt-in via `WaitingTime()` interface
- **Shared Logic:** Core broker code reused between client/server

---

## Architecture

### File Structure

```
crudp/
├── sse.broker.go           # Shared broker logic (queue, batching, user correlation)
├── sse.broker.client.go    # WASM client SSE EventSource handling
├── sse.broker.server.go    # Server SSE endpoint + batch accumulator
├── interfaces.go           # WaitingTime, UserProvider, KVStore interfaces
└── config.go               # Config with defaults
```

### [Communication Flow](diagrams/SSE_BROKER_FLOW.md)

---

## Core Interfaces

### 1. WaitingTime Interface (Optional)

See [WAITING_TIME_INTERFACE.md](WAITING_TIME_INTERFACE.md) for details.

### 2. UserProvider Interface (Required for SSE)

**File: `interfaces.go`**
```go
// UserProvider extracts user context from requests for SSE routing
type UserProvider interface {
    // GetUserID returns unique user identifier from context
    GetUserID(ctx context.Context) string
    
    // GetUserRole returns user role for role-based broadcasting
    // Examples: "admin", "user", "guest"
    GetUserRole(ctx context.Context) string
}
```

**Example Implementation:**
```go
type AuthMiddleware struct{}

func (a *AuthMiddleware) GetUserID(ctx context.Context) string {
    // Extract from JWT, session, etc.
    if userID, ok := ctx.Value("userID").(string); ok {
        return userID
    }
    return ""
}

func (a *AuthMiddleware) GetUserRole(ctx context.Context) string {
    if role, ok := ctx.Value("role").(string); ok {
        return role
    }
    return "guest"
}
```

### 3. KVStore Interface (Optional for Persistence)

See [DATABASE_CONFIG.md](DATABASE_CONFIG.md) for detailed database configuration and KVStore implementations.

### 4. Config Structure

**File: `config.go`**
```go
// Config holds CrudP configuration including SSE broker settings
type Config struct {
    // HTTP endpoint for binary protocol (default: "/api")
    APIEndpoint string
    
    // Server port (default: 6060 - GO GO)
    Port string
    
    // Batch accumulation window in milliseconds (default: 50ms)
    // Server waits this long to accumulate responses before sending
    BatchWindow int
    
    // Max retry attempts for failed requests (default: 3)
    MaxRetries int
    
    // Retry interval in milliseconds (default: 1000ms with exponential backoff)
    RetryInterval int
    
    // Logger function for logging messages (optional)
    Logger func(msg ...any)
    
    // Optional persistence layer
    // See DATABASE_CONFIG.md for implementations
    Store KVStore
    
    // Optional user provider for SSE routing
    UserProvider UserProvider
}

// DefaultConfig returns standard configuration
func DefaultConfig() *Config {
    return &Config{
        APIEndpoint:  "/api",
        Port:         ":6060",
        BatchWindow:  100,  // 100ms batch accumulation
        MaxRetries:   3,
        RetryInterval: 1000, // 1 second base
        Logger:       nil,   // No logging by default
    }
}
```

### 5. Broker Queue Structure (Private)

**File: `sse.broker.go`**
```go
// queuedPacket represents a packet waiting to be sent/processed
type queuedPacket struct {
    packet    Packet
    timestamp int64  // Unix timestamp for retry logic use github.com/cdvelop/unixid
    attempts  int    // Retry counter
    userID    string // For user-based routing
}

// brokerQueue manages packet queueing for offline/retry scenarios
// Private implementation detail - not exposed in public API
type brokerQueue struct {
    packets       []queuedPacket
    tinyBin       *tinybin.TinyBin
    batchWindow   int  // From Config.BatchWindow
    maxRetries    int  // From Config.MaxRetries
    retryInterval int  // From Config.RetryInterval
}

// Enqueue adds packet to local queue
func (bq *brokerQueue) Enqueue(packet Packet, userID string) error

// DequeueBatch retrieves next batch for sending (waits BatchWindow ms)
func (bq *brokerQueue) DequeueBatch(maxSize int) []queuedPacket

// RequeueFailed puts failed packets back with exponential backoff
func (bq *brokerQueue) RequeueFailed(reqIDs []string) error
```

### 6. SSE Routing Strategy (Dependency Injection)

See [EVENTS.md](EVENTS.md) for details.

---

## Design Decisions Summary

### A. Handler Detection
**Resolved:** `WaitingTime()` is optional. If not implemented or returns 0, handler uses synchronous HTTP. Checked at registration and cached in handler metadata.

### B. Batch Accumulation (Mailman Pattern)
**Resolved:** Server accumulates responses in `BatchWindow` (default 100ms). If multiple events arrive within window, they're batched into single SSE event containing `BatchResponse`. This reduces network overhead.

**Example:** 
- Event 1 arrives at T+0ms for User A
- Event 2 arrives at T+2ms for User B
- Server waits until T+100ms, then sends single BatchResponse with both results

### C. User-Based Routing
**Resolved:** Requires `UserProvider` interface injected via `Config`. Server correlates requests by UserID extracted from context. SSE responses can target:
1. **Specific User:** `RouteTarget{UserID: "user123"}`
2. **Role-Based:** `RouteTarget{Role: "admin"}` (all admins receive)
3. **Broadcast:** `RouteTarget{All: true}` (all connected users)

**Resolved:** Handlers use dependency injection for explicit routing. See "Routing Strategy" below.

### D. Reconnection Logic
**Resolved:** Exponential backoff with configurable `MaxRetries` and `RetryInterval` in `Config`. Default: 3 retries, 1s base interval (1s, 2s, 4s).

### E. SSE Connection Lifecycle
**Resolved:** 
- **Multiplexed Connection:** Single SSE connection per user (tubo maestro).
- **Efficient:** Handles all traffic for that user (chat, notifications, data) in one stream.
- **Routing:** Client router dispatches messages to correct handlers using `HandlerID` in packet.

### F. Queue Persistence
**Resolved:** Optional `KVStore` interface in `Config`. User provides implementation (localStorage for WASM, memory/DB for server). See [DATABASE_CONFIG.md](DATABASE_CONFIG.md) for details.

### G. ListenAndServe() Behavior
**Resolved:** 
- **Client (WASM):** Blocks with `select{}`, starts background SSE listeners
- **Server:** Blocks with `http.ListenAndServe()`, configurable port via `Config`

---

## Proposed API

### CrudP Constructor Changes

**File: `crudp.go`**
```go
// New creates CrudP instance with config
// Accepts: *Config (required)
// Logger is set via config.Logger
func New(config *Config) *CrudP {
    if config == nil {
        config = DefaultConfig()
    }
    
    return &CrudP{
        tinyBin: tinybin.New(),
        log:     config.Logger,
        config:  config,
        brokerQueue: &brokerQueue{
            tinyBin:       tinybin.New(),
            batchWindow:   config.BatchWindow,
            maxRetries:    config.MaxRetries,
            retryInterval: config.RetryInterval,
        },
    }
}
```

### Client Side (WASM)

**File: `web/client.go`**
```go
//go:build wasm
package main

import (
    "myProject/pkg/router"
    "github.com/cdvelop/crudp"
)

func main() {
    cp := router.NewRouter()
    
    // Blocks forever, starts SSE listeners in background
      if err := cp.ListenAndServe(); err != nil {
        panic(err)
    }
}
```

### Server Side

**File: `web/server.go`**
```go
//go:build !wasm
package main

import (
    "myProject/pkg/router"
)

func main() {
    
    cp := crudp.New(config)
    // Custom config
    cp.SetPort(":8080")
    cp.SetBatchWindow(10) // 10ms batching
    
    // Blocks forever, starts HTTP server
    if err := cp.ListenAndServe(); err != nil {
        panic(err)
    }
}
```

---

## Implementation Roadmap

### Phase 1: Core Broker (Base)
- [ ] Define interfaces: `AsyncHandler`, `UserProvider`, `KVStore`
- [ ] Implement `Config` with defaults
- [ ] Create `brokerQueue` with retry logic
- [ ] Update `New()` to accept `Config`

### Phase 2: Synchronous Flow (Baseline)
- [ ] Refactor `ProcessBatch()` to check `WaitingTime()`
- [ ] Keep existing HTTP POST response for WaitingTime=0
- [ ] Test backward compatibility

### Phase 3: SSE Server Implementation
- [ ] Create `sse.broker.server.go` with SSE endpoint
- [ ] Implement global batch accumulator (Ticker)
- [ ] Implement multiplexed SSE connection manager (1 per user)
- [ ] Implement dependency injection for routing (Notifier interface)

### Phase 4: SSE Client Implementation  
- [ ] Create `sse.broker.client.go` with EventSource
- [ ] Implement client-side queue with retry
- [ ] Handle SSE reconnection logic
- [ ] Add localStorage persistence (optional) - see [DATABASE_CONFIG.md](DATABASE_CONFIG.md)

### Phase 5: Routing & Broadcasting
- [ ] Implement `RouteTarget` logic in broker
- [ ] Add user-based filtering
- [ ] Add role-based broadcasting
- [ ] Add global broadcast

---
