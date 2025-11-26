# Config Structure

## Overview

The `Config` struct holds all configuration options for CRUDP, including settings for the SSE broker, HTTP endpoints, batching, retries, and optional components like persistence and user providers.

## Structure Definition

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