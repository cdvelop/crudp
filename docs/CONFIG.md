# Configuration System

## Overview

CRUDP is configured using a `Config` struct passed to the `New()` constructor. A `NewDefault()` constructor is also available for convenience.

## The `Config` Struct

The `Config` struct allows you to customize various aspects of CRUDP's behavior.

**File: `config.go`**
```go
package crudp

// UserProvider provides user identification for SSE routing
type UserProvider interface {
    GetUserID(ctx any) string
}

// Config contains CrudP configuration
// NOTE: Logger is NOT here - configured via SetLogger()
type Config struct {
    // Codec for serialization. Default: tinyjson.New()
    Codec Codec

    // UseBinary uses binary encoding. Default: false (JSON)
    UseBinary bool

    // APIEndpoint for batch requests. Default: "/api"
    APIEndpoint string
    
    // SSEEndpoint for event stream. Default: "/events"
    SSEEndpoint string
    
    // BatchWindow in milliseconds. Default: 50
    BatchWindow int
    
    // MaxRetries for failed requests. Default: 3
    MaxRetries int
    
    // RetryInterval base in ms. Default: 1000
    RetryInterval int
    
    // Port for HTTP server (server only). Default: ":6060"
    Port string
    
    // UserProvider for SSE routing (server only). Default: nil
    UserProvider UserProvider

    // ServerURL base (client only). Default: "" (same origin)
    ServerURL string

    // OnMessage callback for notifications (client only)
    OnMessage func(msgType uint8, message string)
}

// DefaultConfig returns configuration with default values
func DefaultConfig() *Config {
    return &Config{
        Codec:         nil, // Will assign tinyjson in New()
        UseBinary:     false,
        APIEndpoint:   "/api",
        SSEEndpoint:   "/events",
        BatchWindow:   50,
        MaxRetries:    3,
        RetryInterval: 1000,
        Port:          ":6060",
    }
}
```

## The `Codec` Interface

CRUDP uses a `Codec` interface for serialization, allowing you to plug in different encoding/decoding implementations. The default codec is `tinyjson`.

**File: `codec.go`**
```go
package crudp

// Codec interface for serialization (replaces direct tinybin dependency)
type Codec interface {
    Encode(data any) ([]byte, error)
    Decode(data []byte, v any) error
}
```

## Constructors

### `New(cfg *Config)`

The primary constructor for `CrudP`. It takes a `*Config` struct as an argument. If `nil` is passed, it will use the default configuration.

### `NewDefault()`

A convenience constructor that creates a `CrudP` instance with the default configuration.

## Logging

Logging is configured via methods on the `CrudP` instance, not through the `Config` struct.

### `SetLogger(logger func(...any))`

Sets a custom logging function.

### `DisableLogger()`

Disables logging. This is the default behavior.
