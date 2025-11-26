# SSE Routing Strategy: Response Interface Pattern

## Overview

**Problem:** How does a handler send messages to multiple recipients without coupling to CRUDP?

**Solution:** Response Interface. Handlers return a `Response` interface that encapsulates data, broadcast targets, and errors. CRUDP interprets this to route messages via SSE. Handlers that need user context define their own `UserProvider` interface for dependency injection.

**Key Pattern:**
```go
// Response interface that handlers return
type Response interface {
    Response() (data any, broadcast []string, err error)
}

// Handler defines its own UserProvider interface
type UserProvider interface {
    GetUserID(ctx context.Context) string
    GetUserRole(ctx context.Context) string
}

type Handler struct {
    userProvider UserProvider // Injected dependency
}

// Handler implements CRUD methods returning []any
func (h *Handler) Create(ctx context.Context, data ...any) []any {
    var responses []any
    
    for _, item := range data {
        // Use injected UserProvider for flexible user identification
        userID := h.userProvider.GetUserID(ctx)
        
        // Process each item...
        
        // Return response for each item
        responses = append(responses, &MyResponse{data: result, broadcast: targets})
    }
    
    return responses
}

// Response struct example
type MyResponse struct {
    data      any
    broadcast []string
}

func (r *MyResponse) Response() (data any, broadcast []string, err error) {
    return r.data, r.broadcast, nil
}
```

**Benefits:**
- **Zero Coupling:** Handlers don't import CRUDP or know about framework details
- **Simple:** No dependency injection needed for basic handlers
- **Flexible:** Handlers define their own interfaces for needed dependencies
- **Type Safe:** Explicit Response interface
- **Configurable:** Custom UserProvider implementations allow flexible user identification

## Chat Handler Example

This example demonstrates how to implement a real-time chat handler using CRUDP's SSE Broker with the **Response Interface Pattern** to maintain complete decoupling between business logic and the communication framework.

The chat handler needs to:
1. **Receive messages** from users (via `Create` method) - can handle multiple messages in one call
2. **Determine routing** based on message type (private vs broadcast) for each message
3. **Return responses** with data and broadcast targets via the Response interface for each message

Handlers return `[]any` containing structs that implement the `Response` interface, allowing CRUDP to extract routing information.

### Handler Implementation
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

// Define UserProvider interface locally to avoid coupling with CRUDP
type UserProvider interface {
    GetUserID(ctx context.Context) string
    GetUserRole(ctx context.Context) string
}

// Response struct for chat messages
type ChatResponse struct {
    Data      any
    Broadcast []string // Empty slice for broadcast to all
    err      error
}

func (r *ChatResponse) Response() (data any, broadcast []string, err error) {
    return r.Data, r.Broadcast, r.err
}

type Handler struct {
    userProvider UserProvider // Injected dependency
}

// Constructor for dependency injection
func New(up UserProvider) *Handler {
    return &Handler{userProvider: up}
}

// Business Logic: Handle incoming messages (can process multiple in one call)
func (h *Handler) Create(ctx context.Context, data ...any) []any {
    var responses []any
    
    // Process all incoming messages
    for _, item := range data {
        // Extract sender using injected UserProvider
        senderID := h.userProvider.GetUserID(ctx)
        
        // Decode input data
        msg := item.(Message)
        msg.From = senderID
        
        // Business logic: validate, save to DB, etc.
        // ... (omitted for brevity)
        
        // Routing Logic: Return response with appropriate broadcast targets
        if msg.IsPrivate {
            // Send to specific user
            responses = append(responses, &ChatResponse{Data: msg, Broadcast: []string{msg.ToUserID}})
        } else {
            // Broadcast to all users (empty slice means broadcast)
            responses = append(responses, &ChatResponse{Data: msg, Broadcast: []string{}})
        }
    }
    
    return responses
}

// Async Configuration: Enable SSE for real-time chat
func (h *Handler) WaitingTime() int {
    return -1 // Permanent SSE connection for chat
}
```

## Module Registration

**File: `modules/modules.go`**
```go
package modules

import (
    "myProject/modules/chat"
)

// Init receives UserProvider implementation (CRUDP will provide this)
func Init(up chat.UserProvider) []any {
    return []any{
        chat.New(up), // Inject UserProvider for user-aware handlers
        &other.Handler{}, // No injection needed for handlers without user context
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
    "myProject/auth"
)

func NewRouter() any {
    cp := crudp.New()
    
    // Set UserProvider for context enrichment
    userProvider := &auth.AuthMiddleware{}
    cp.SetUserProvider(userProvider)
    
    // Register handlers - CRUDP's UserProvider implements chat.UserProvider
    cp.RegisterHandler(modules.Init(userProvider)...)
    
    return cp.Router()
}
```

## How It Works

1. **Client sends message(s):** POST `/api` with `HandlerID=chat`, `Action='c'`, `Data=[Message1, Message2, ...]`
2. **CRUDP enriches context:** Uses its `UserProvider` to add user context to context
3. **Server processes:** `chat.Create()` iterates through all messages, uses injected `UserProvider` to get `UserID`, processes each message
4. **Return responses:** Handler returns `[]any` with multiple `ChatResponse` structs (one per input message)
5. **CRUDP interprets:** Extracts data, broadcast targets, and error from each `Response()` method
6. **SSE delivery:** CRUDP queues messages for the appropriate recipients via SSE
7. **Client receives:** SSE events with the messages, routed to correct handlers

## Testing

**File: `modules/chat/chat_test.go`**
```go
package chat

import (
    "context"
    "testing"
)

// Mock UserProvider for testing
type MockUserProvider struct{}

func (m *MockUserProvider) GetUserID(ctx context.Context) string {
    return "user123"
}

func (m *MockUserProvider) GetUserRole(ctx context.Context) string {
    return "user"
}

func TestChatHandler(t *testing.T) {
    mockUP := &MockUserProvider{}
    handler := New(mockUP)
    
    ctx := context.Background()
    
    // Test multiple messages in one batch
    privateMsg := Message{ToUserID: "user456", Content: "Hello", IsPrivate: true}
    broadcastMsg := Message{Content: "Hello all", IsPrivate: false}
    
    responses := handler.Create(ctx, privateMsg, broadcastMsg)
    
    if len(responses) != 2 {
        t.Fatalf("Expected 2 responses, got %d", len(responses))
    }
    
    // Check first response (private message)
    resp1 := responses[0].(*ChatResponse)
    data1, broadcast1, err1 := resp1.Response()
    
    if err1 != nil {
        t.Error("Unexpected error in first response:", err1)
    }
    
    if len(broadcast1) != 1 || broadcast1[0] != "user456" {
        t.Error("Private message not sent to correct user")
    }
    
    // Check second response (broadcast message)
    resp2 := responses[1].(*ChatResponse)
    data2, broadcast2, err2 := resp2.Response()
    
    if err2 != nil {
        t.Error("Unexpected error in second response:", err2)
    }
    
    if len(broadcast2) != 0 {
        t.Error("Broadcast should have empty slice")
    }
}
```

## Related Documentation

- [USER_PROVIDER.md](USER_PROVIDER.md) - User context extraction for routing
- [SSE_BROKER.md](SSE_BROKER.md) - Main SSE Broker documentation
- [INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md) - Integration examples