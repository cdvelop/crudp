# CRUDP Integration Guide

## Overview

This guide explains how to integrate CRUDP into a Go project for binary protocol communication between server and WASM client. CRUDP is used only in the router; business modules remain decoupled.

**For advanced features** (HTTP routes, middleware, file uploads), see [HANDLER_REGISTER.md](HANDLER_REGISTER.md).

## Project Structure

```
myProject/
├── modules/
│   ├── modules.go          # Handler registration
│   └── users/
│       └── users.go        # Business logic (implements CRUD interfaces)
├── pkg/
│   └── router/
│       └── router.go       # CRUDP initialization
├── web/
│   ├── client.go           # WASM entry point
│   └── server.go           # Server entry point
└── go.mod                  # Dependency: github.com/cdvelop/crudp
```

## Implementation Steps

### 1. Define Handler with CRUD Interfaces

**File: `modules/users/users.go`**
```go
package users

import "context"

type Handler struct{}

// Implement CRUDP interfaces (Creator, Reader, Updater, Deleter)
func (h *Handler) Create(ctx context.Context, data ...any) (any, error) {
    // Business logic
    return "created", nil
}

func (h *Handler) Read(ctx context.Context, data ...any) (any, error) {
    // Business logic
    return "user data", nil
}
```

**Available interfaces:** See [interfaces.go](../interfaces.go) for `Creator`, `Reader`, `Updater`, `Deleter`, `Lister`, etc.

### 2. Register Handlers

**File: `modules/modules.go`**
```go
package modules

import "myProject/modules/users"

func Init() []any {
    return []any{
        &users.Handler{},
        // Add more handlers...
    }
}
```

### 3. Initialize CRUDP Router

**File: `pkg/router/router.go`**
```go
package router

import (
    "github.com/cdvelop/crudp"
    "myProject/modules"
)

func NewRouter() any {
    cp := crudp.New()
    cp.RegisterHandler(modules.Init()...)
    return cp.Router() // Returns http.Handler (!wasm) or client (wasm)
}
```

### 4. Server Entry Point

**File: `web/server.go`**
```go
//go:build !wasm

package main

import (
    "net/http"
    "myProject/pkg/router"
)

func main() {
    handler := router.NewRouter().(http.Handler)
    http.ListenAndServe(":8080", handler)
}
```

### 5. Client Entry Point

**File: `web/client.go`**
```go
//go:build wasm

package main

import "myProject/pkg/router"

func main() {
    client := router.NewRouter()
    // Use client for CRUD operations
    select {} // Keep alive
}
```

## Key Principles

- **📦 Decoupling:** Modules don't import CRUDP; only return handlers
- **🔄 Build Tags:** `NewRouter()` returns different types based on `!wasm` vs `wasm`
- **⚡ Binary Protocol:** Efficient typed communication between client/server
- **🎯 Single Registration:** `RegisterHandler()` sets up everything

## Advanced Features

- **HTTP Routes & Middleware:** See [HANDLER_REGISTER.md](HANDLER_REGISTER.md)
- **File Uploads:** See [FILE_UPLOAD.md](FILE_UPLOAD.md)
- **Package Structure:** See [crudp_project_structure.md](crudp_project_structure.md)
