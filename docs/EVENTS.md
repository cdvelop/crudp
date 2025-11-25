# SSE Routing Strategy: Dependency Injection for Events

## Overview

**Problem:** How does a handler send messages to multiple recipients without coupling to CRUDP?

**Solution:** Dependency Injection. Handlers define their own `Notifier` interface, and CRUDP provides the implementation at initialization.

**Key Pattern:**
```go
// Handler defines what it needs
type Notifier interface {
    SendToUser(userID string, data any) error
    Broadcast(data any) error
}

// Constructor receives implementation
func New(n Notifier) *Handler {
    return &Handler{notifier: n}
}

// Business logic uses injected dependency
func (h *Handler) Create(ctx context.Context, data ...any) (any, error) {
    // ... process data ...
    
    // Route using injected notifier
    if isPrivate {
        h.notifier.SendToUser(targetUser, result)
    } else {
        h.notifier.Broadcast(result)
    }
    
    return "sent", nil
}
```

**Benefits:**
- **Zero Coupling:** Handlers don't import CRUDP
- **Testable:** Mock the Notifier interface
- **Type Safe:** Explicit interfaces instead of strings
- **Flexible:** Handler controls routing logic

## Chat Handler Example

This example demonstrates how to implement a real-time chat handler using CRUDP's SSE Broker with **Dependency Injection** to maintain complete decoupling between business logic and the communication framework.

The chat handler needs to:
1. **Receive messages** from users (via `Create` method)
2. **Determine routing** based on message type (private vs broadcast)
3. **Send responses** to appropriate recipients via SSE

To achieve this without importing CRUDP, we use **Dependency Injection**: the handler defines what it needs (a `Notifier` interface), and the framework provides the implementation.

### Handler Implementation

**File: `modules/chat/chat.go`**
```go
package chat

import "context"

type Message struct {
    ID       string 
    From     string 
    ToUserID string  // Empty for broadcast
    Content  string 
    IsPrivate bool  
}

type Handler struct {
    notifier Notifier // Injected dependency
}

// Constructor for dependency injection
func New(n Notifier) *Handler {
    return &Handler{notifier: n}
}

// Business Logic: Handle incoming messages
func (h *Handler) Create(ctx context.Context, data ...any) (any, error) {
    // Extract sender from context (provided by UserProvider)
    senderID := ctx.Value("UserID").(string)
    
    // Decode input data
    msg := data[0].(Message)
    msg.From = senderID
    
    // Business logic: validate, save to DB, etc.
    // ... (omitted for brevity)
    
    // Routing Logic: Use injected Notifier to send to recipients
    if msg.IsPrivate {
        // Send to specific user
        return h.notifier.SendToUser(msg.ToUserID, msg)
    } else {
        // Broadcast to all users
        return h.notifier.Broadcast(msg)
    }
}

// Async Configuration: Enable SSE for real-time chat
func (h *Handler) WaitingTime() int {
    return -1 // Permanent SSE connection for chat
}
```

## Module Registration with Dependency Injection

**File: `modules/modules.go`**
```go
package modules

import (
    "myProject/modules/chat"
)

// Init receives the Notifier implementation (CRUDP will provide this)
func Init(notifier chat.Notifier) []any {
    return []any{
        chat.New(notifier), // Inject dependency
        // other handlers...
    }
}
```

## Router Wiring

**File: `pkg/router/router.go`**
```go
package router

import (
    "github.com/cdvelop/crudp"
    "myProject/modules"
)

func NewRouter() any {
    cp := crudp.New()
    
    // CRUDP implements chat.Notifier implicitly
    // Pass cp as the Notifier implementation
    cp.RegisterHandler(modules.Init(cp)...)
    
    return cp.Router()
}
```

## How It Works

1. **Client sends message:** POST `/api` with `HandlerID=chat`, `Action='c'`, `Data=[Message]`
2. **Server processes:** `chat.Create()` extracts `UserID` from context, processes message
3. **Routing decision:** Handler calls `notifier.SendToUser()` or `notifier.Broadcast()`
4. **SSE delivery:** CRUDP queues the message for the appropriate recipients via SSE
5. **Client receives:** SSE event with the message, routed to correct handler

## Testing

**File: `modules/chat/chat_test.go`**
```go
package chat

import (
    "context"
    "testing"
)

// Mock implementation for testing
type MockNotifier struct {
    SentToUser   []string
    Broadcasted  []any
}

func (m *MockNotifier) SendToUser(userID string, data any) error {
    m.SentToUser = append(m.SentToUser, userID)
    return nil
}

func (m *MockNotifier) Broadcast(data any) error {
    m.Broadcasted = append(m.Broadcasted, data)
    return nil
}

func TestChatHandler(t *testing.T) {
    mock := &MockNotifier{}
    handler := New(mock)
    
    ctx := context.WithValue(context.Background(), "UserID", "user123")
    
    // Test private message
    privateMsg := Message{ToUserID: "user456", Content: "Hello", IsPrivate: true}
    handler.Create(ctx, privateMsg)
    
    if len(mock.SentToUser) != 1 || mock.SentToUser[0] != "user456" {
        t.Error("Private message not sent to correct user")
    }
    
    // Test broadcast
    broadcastMsg := Message{Content: "Hello all", IsPrivate: false}
    handler.Create(ctx, broadcastMsg)
    
    if len(mock.Broadcasted) != 1 {
        t.Error("Broadcast message not sent")
    }
}
```

## Related Documentation

- [SSE_BROKER.md](SSE_BROKER.md) - Main SSE Broker documentation
- [INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md) - Integration examples