# Advanced: HTTP Routes & Middleware System

**Prerequisites:** Read [INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md) first for basic CRUDP integration.

**Purpose:** This guide covers advanced features for handlers that need custom HTTP routes or global middleware, beyond the binary protocol.

## Core Principles

1. **Optional HTTP Routes:** Add custom endpoints (e.g., `/upload`, `/export`) via `HttpRouteProvider`
2. **Global Middleware:** Provide middleware that applies to ALL routes via `MiddlewareProvider`
3. **Centralized Security:** All routes are automatically wrapped with registered middleware

---

## 1. Optional HTTP Interfaces

**File: `env.router.go` (CRUDP package)**
```go
//go:build !wasm
package crudp

import "net/http"

// Optional: Add custom HTTP routes (e.g., /upload, /export)
type HttpRouteProvider interface {
    RegisterRoutes(mux *http.ServeMux)
}

// Optional: Provide global middleware (authentication, logging, etc.)
type MiddlewareProvider interface {
    Middleware(next http.Handler) http.Handler
}
```

**Note:** Binary protocol interfaces (`Creator`, `Reader`, etc.) are covered in [INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md).

## 2. CRUDP Router Implementation

**File: `env.router.go` (CRUDP package, server-only)**
```go
//go:build !wasm
package crudp

import (
    "net/http"
)

// BuildRouter creates the complete HTTP handler with routes and middleware
func (cp *CrudP) BuildRouter() http.Handler {
    mux := http.NewServeMux()
    
    // 1. Register CRUDP's binary protocol endpoint (configurable)
    mux.HandleFunc(cp.apiEndpoint, cp.handleBinaryProtocol)
    
    // 2. Collect all global middleware from handlers
    var globalMiddleware []func(http.Handler) http.Handler
    for _, h := range cp.handlers {
        if mwProvider, ok := h.Handler.(MiddlewareProvider); ok {
            globalMiddleware = append(globalMiddleware, mwProvider.Middleware)
        }
    }
    
    // 3. Let handlers register their custom HTTP routes
    for _, h := range cp.handlers {
        if routeProvider, ok := h.Handler.(HttpRouteProvider); ok {
            routeProvider.RegisterRoutes(mux)
        }
    }
    
    // 4. Wrap everything with global middleware (applied in registration order)
    handler := mux
    for _, mw := range globalMiddleware {
        handler = mw(handler)
    }
    
    return handler
}

// handleBinaryProtocol processes CRUDP binary requests
func (cp *CrudP) handleBinaryProtocol(w http.ResponseWriter, r *http.Request) {
    // ... existing ProcessPacket logic adapted to HTTP
}
```

## 3. Handler with HTTP Routes & Middleware

**File: `modules/users/back.server.go`** (server-only code)
```go
//go:build !wasm

package users

import (
    "net/http"
)

// Optional: Implement custom HTTP routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
    mux.Handle("POST /users/avatar", http.HandlerFunc(h.handleAvatarUpload))
    mux.Handle("GET /users/export", http.HandlerFunc(h.handleExport))
}

// Optional: Provide global middleware (e.g., authentication)
func (h *Handler) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Authentication logic applied to ALL routes
        if !isAuthenticated(r) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func (h *Handler) handleAvatarUpload(w http.ResponseWriter, r *http.Request) {
    // Custom HTTP logic
}

func (h *Handler) handleExport(w http.ResponseWriter, r *http.Request) {
    // Custom HTTP logic
}
```

## 3.1 Middleware-Only Handler

**File: `modules/logging/logging.go`**
```go
//go:build !wasm
package logging

import (
    "net/http"
    "time"
)

type Handler struct{
    log func(args ...any)
}

// This handler only provides middleware, no CRUD operations
func (h *Handler) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        h.log("Request: %s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
        h.log("Completed in %v", time.Since(start))
    })
}
```

## 3.2 File Upload Example

**See:** [FILE_UPLOAD.md](FILE_UPLOAD.md) for complete implementation using `HttpRouteProvider`.

---

## Key Considerations

- **Middleware Order:** Applied in registration order. Put authentication first.
- **Optional:** Only implement these interfaces when you need custom HTTP routes or middleware.
- **No Impact on Binary Protocol:** Handlers without these interfaces work normally via CRUDP's binary protocol.
- **Server & Client Setup:** See [INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md) for `NewRouter()` usage.


