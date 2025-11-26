# WaitingTime Interface (Optional)

**File: `interfaces.go`**
```go
// AsyncHandler marks handlers that need async processing via SSE
// If not implemented, handler defaults to synchronous HTTP
type AsyncHandler interface {
    // WaitingTime returns batch accumulation window in milliseconds
    // Not implemented or 0 = synchronous HTTP response (default)
    // >0 = async via SSE with batch accumulation (e.g., 2000 for 2s batching)
    // -1 = permanent SSE connection (kept-alive for real-time updates)
    WaitingTime() int
}
```

**Example Handlers:**
```go
// Synchronous handler (no WaitingTime needed)
type UserHandler struct{}

func (h *UserHandler) Create(ctx context.Context, data ...any) []any {
    var responses []any
    
    for _, item := range data {
        // Process item
        responses = append(responses, "created")
    }
    
    return responses
}

// Async batch handler (accumulates responses for 2 seconds)
type NotificationHandler struct{}

func (h *NotificationHandler) WaitingTime() int {
    return 2000 // Wait 2s to batch notifications
}

// Permanent connection handler (real-time chat)
type ChatHandler struct{}

func (h *ChatHandler) WaitingTime() int {
    return -1 // Keep SSE alive
}
```