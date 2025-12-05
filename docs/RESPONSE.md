# Returning Responses

## The `any` Return Type

All CRUD methods (`Create`, `Read`, `Update`, `Delete`) now return a single `any` value. This provides a flexible way to return data from your handlers. The returned value can be:

-   A simple struct or a slice of structs.
-   A `Response` interface for more advanced scenarios, such as SSE broadcasting.
-   `nil` if no data needs to be returned.

## The `Response` Interface

For scenarios where you need to send a response to multiple clients (e.g., in a real-time application), you can return a value that implements the `Response` interface. This interface allows you to specify the data to be sent, as well as a list of broadcast targets.

**File: `interfaces.go`**
```go
package crudp

// Response is an interface that can be returned by handlers to provide
// additional information for routing and broadcasting.
type Response interface {
    Response() (data any, broadcast []string, err error)
}
```

### Multiple Responses

A handler can also return a slice of `Response` objects (`[]Response`), which is useful when a single operation needs to trigger multiple notifications to different clients.

## Example

Here's an example of a handler that returns a simple struct:

```go
package main

import "context"

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

type UserHandler struct{}

func (h *UserHandler) Read(ctx context.Context, data ...any) any {
    // In a real application, you would fetch the user from a database.
    return User{ID: 1, Name: "John Doe"}
}
```

And here's an example of a handler that returns a `Response` for broadcasting:

```go
package main

import "context"

type ChatMessage struct {
    From    string `json:"from"`
    Message string `json:"message"`
}

type ChatResponse struct {
    Data      any
    Broadcast []string
}

func (r *ChatResponse) Response() (data any, broadcast []string, err error) {
    return r.Data, r.Broadcast, nil
}

type ChatHandler struct{}

func (h *ChatHandler) Create(ctx context.Context, data ...any) any {
    // In a real application, you would save the message to a database.
    msg := data[0].(ChatMessage)
    
    // Broadcast the message to all connected clients.
    return &ChatResponse{
        Data:      msg,
        Broadcast: []string{}, // Empty slice means broadcast to all.
    }
}
```
