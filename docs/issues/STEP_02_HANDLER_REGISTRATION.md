# Step 2: Handler Registration

> **Prerequisite:** [STEP_01_PACKET_STRUCTURE.md](./STEP_01_PACKET_STRUCTURE.md)  
> **Next:** [STEP_03_CONFIG_SYSTEM.md](./STEP_03_CONFIG_SYSTEM.md)

## Objective

- Automatic handler name via reflection (optional `HandlerName()` for override)
- CRUD interfaces return `any` instead of `[]any`
- Add `Validator` and `FieldValidator` interfaces
- Optional validation before executing handler

---

## 2.1 Update CRUD interfaces

**File:** `interfaces.go`

**Current code:**
```go
type Creator interface {
    Create(ctx context.Context, data ...any) []any
}
// ... same for Reader, Updater, Deleter
```

**Replace with:**
```go
package crudp

import "context"

// Response interface that handlers return for routing
// Handlers return `any` which can be:
// - A simple value for direct response
// - []Response for multiple broadcast
// - Individual Response for SSE routing
type Response interface {
    Response() (data any, broadcast []string, err error)
}

// Separate CRUD interfaces - handlers implement only what they need
// Return `any` which internally can be Response or []Response for broadcast
type Creator interface {
    Create(ctx context.Context, data ...any) any
}

type Reader interface {
    Read(ctx context.Context, data ...any) any
}

type Updater interface {
    Update(ctx context.Context, data ...any) any
}

type Deleter interface {
    Delete(ctx context.Context, data ...any) any
}

// NamedHandler allows override of automatic name (optional)
// If not implemented, reflection is used: TypeName -> snake_case
type NamedHandler interface {
    HandlerName() string
}

// Validator validates complete data before action (optional)
type Validator interface {
    Validate(action byte, data ...any) error
}

// FieldValidator validates individual fields for UI (optional)
type FieldValidator interface {
    ValidateField(fieldName string, value string) error
}
```

---

## 2.2 Update actionHandler struct

**File:** `crudp.go`

**Current code:**
```go
type actionHandler struct {
    Create  func(context.Context, ...any) []any
    Read    func(context.Context, ...any) []any
    Update  func(context.Context, ...any) []any
    Delete  func(context.Context, ...any) []any
    Handler any
}
```

**Replace with:**
```go
type actionHandler struct {
    name    string  // Handler name (snake_case)
    index   uint8   // Position in handlers slice
    handler any     // Original handler for type analysis
    Create  func(context.Context, ...any) any
    Read    func(context.Context, ...any) any
    Update  func(context.Context, ...any) any
    Delete  func(context.Context, ...any) any
}
```

---

## 2.3 Create getHandlerName function

**File:** `handlers.go`

Add at the beginning, after imports:

```go
import (
    "context"
    "reflect"
    
    . "github.com/cdvelop/tinystring"
)

// getHandlerName gets the handler name
// Priority: 1) HandlerName() if implemented, 2) reflection + snake_case
func getHandlerName(handler any) string {
    // First try NamedHandler interface
    if named, ok := handler.(NamedHandler); ok {
        return named.HandlerName()
    }
    
    // Fallback: use reflection and convert to snake_case
    t := reflect.TypeOf(handler)
    if t.Kind() == reflect.Ptr {
        t = t.Elem()
    }
    
    // Use tinystring.SnakeLow for conversion
    // UserHandler -> user_handler
    // APIController -> api_controller
    return NewConv(t.Name()).SnakeLow().String()
}
```

---

## 2.4 Update RegisterHandler

**File:** `handlers.go`

**Current code:**
```go
func (cp *CrudP) RegisterHandler(handlers ...any) error {
    cp.handlers = make([]actionHandler, len(handlers))

    for index, handler := range handlers {
        if handler == nil {
            return Errf("handler %d is nil", index)
        }

        cp.handlers[index].Handler = handler
        cp.bind(uint8(index), handler)
    }

    return nil
}
```

**Replace with:**
```go
func (cp *CrudP) RegisterHandler(handlers ...any) error {
    cp.handlers = make([]actionHandler, len(handlers))
    
    for i, h := range handlers {
        if h == nil {
            return Errf("handler %d is nil", i)
        }
        
        // Get name (via interface or reflection)
        name := getHandlerName(h)
        
        cp.handlers[i] = actionHandler{
            name:    name,
            index:   uint8(i),
            handler: h,
        }
        
        cp.bind(uint8(i), h)
        
        if cp.log != nil {
            cp.log("registered handler:", name, "at index", i)
        }
    }
    
    return nil
}

// GetHandlerName returns the handler name by its ID
func (cp *CrudP) GetHandlerName(handlerID uint8) string {
    if int(handlerID) >= len(cp.handlers) {
        return ""
    }
    return cp.handlers[handlerID].name
}
```

---

## 2.5 Update bind

**File:** `handlers.go`

Update to match new return type:

```go
// bind copies CRUD functions without dynamic allocations
func (cp *CrudP) bind(index uint8, handler any) {
    if creator, ok := handler.(Creator); ok {
        cp.handlers[index].Create = creator.Create
    }
    if reader, ok := handler.(Reader); ok {
        cp.handlers[index].Read = reader.Read
    }
    if updater, ok := handler.(Updater); ok {
        cp.handlers[index].Update = updater.Update
    }
    if deleter, ok := handler.(Deleter); ok {
        cp.handlers[index].Delete = deleter.Delete
    }
}
```

---

## 2.6 Update callHandler with validation

**File:** `handlers.go`

**Current code:**
```go
func (cp *CrudP) callHandler(ctx context.Context, handlerID uint8, action byte, data ...any) ([]any, error) {
```

**Replace with:**
```go
func (cp *CrudP) callHandler(ctx context.Context, handlerID uint8, action byte, data ...any) (any, error) {
    if int(handlerID) >= len(cp.handlers) {
        return nil, Errf("no handler found for id: %d", handlerID)
    }
    
    handler := cp.handlers[handlerID]
    
    // Optional validation before executing
    if validator, ok := handler.handler.(Validator); ok {
        if err := validator.Validate(action, data...); err != nil {
            return nil, err
        }
    }
    
    // Check context canceled
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    switch action {
    case 'c':
        if handler.Create != nil {
            return handler.Create(ctx, data...), nil
        }
    case 'r':
        if handler.Read != nil {
            return handler.Read(ctx, data...), nil
        }
    case 'u':
        if handler.Update != nil {
            return handler.Update(ctx, data...), nil
        }
    case 'd':
        if handler.Delete != nil {
            return handler.Delete(ctx, data...), nil
        }
    }
    
    return nil, Errf("action '%c' not implemented for handler: %s", action, handler.name)
}
```

---

## 2.7 Tests

### File: `handlers_shared_test.go`

```go
package crudp_test

import (
    "context"
    "testing"
    
    "github.com/cdvelop/crudp"
    . "github.com/cdvelop/tinystring"
)

// Test handler with explicit name
type explicitNameHandler struct{}

type ExplicitCreateResponse struct {
    Message string `json:"message"`
}

func (r ExplicitCreateResponse) Response() (data any, broadcast []string, err error) {
    return r, nil, nil
}

func (h *explicitNameHandler) HandlerName() string { return "my_custom_name" }
func (h *explicitNameHandler) Create(ctx context.Context, data ...any) any {
    return ExplicitCreateResponse{Message: "created"}
}

// Test handler without explicit name (uses reflection)
type UserController struct{}

type CreateResponse struct {
    ID     int    `json:"id"`
    Status string `json:"status"`
}

func (r CreateResponse) Response() (data any, broadcast []string, err error) {
    return r, nil, nil
}

type ReadResponse struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func (r ReadResponse) Response() (data any, broadcast []string, err error) {
    return r, nil, nil
}

func (h *UserController) Create(ctx context.Context, data ...any) any {
    return CreateResponse{ID: 1, Status: "created"}
}

func (h *UserController) Read(ctx context.Context, data ...any) any {
    return ReadResponse{ID: 1, Name: "test"}
}

// Handler with validation
type ValidatedHandler struct{}

type ValidatedCreateResponse struct {
    Message string `json:"message"`
}

func (r ValidatedCreateResponse) Response() (data any, broadcast []string, err error) {
    return r, nil, nil
}

func (h *ValidatedHandler) Create(ctx context.Context, data ...any) any {
    return ValidatedCreateResponse{Message: "validated_created"}
}

func (h *ValidatedHandler) Validate(action byte, data ...any) error {
    if len(data) == 0 {
        return Err("no data provided")
    }
    return nil
}

func (h *ValidatedHandler) ValidateField(fieldName, value string) error {
    if fieldName == "email" && value == "" {
        return Err("email is required")
    }
    return nil
}

// Shared tests
func HandlerRegistrationShared(t *testing.T, cp *crudp.CrudP) {
    t.Run("Explicit HandlerName", func(t *testing.T) {
        cp := crudp.NewDefault()
        err := cp.RegisterHandler(&explicitNameHandler{})
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        
        name := cp.GetHandlerName(0)
        if name != "my_custom_name" {
            t.Errorf("expected 'my_custom_name', got '%s'", name)
        }
    })
    
    t.Run("Reflection Name (snake_case)", func(t *testing.T) {
        cp := crudp.NewDefault()
        err := cp.RegisterHandler(&UserController{})
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        
        name := cp.GetHandlerName(0)
        if name != "user_controller" {
            t.Errorf("expected 'user_controller', got '%s'", name)
        }
    })
    
    t.Run("Nil Handler Error", func(t *testing.T) {
        cp := crudp.NewDefault()
        err := cp.RegisterHandler(nil)
        if err == nil {
            t.Error("expected error for nil handler")
        }
    })
    
    t.Run("Multiple Handlers", func(t *testing.T) {
        cp := crudp.NewDefault()
        err := cp.RegisterHandler(
            &explicitNameHandler{},
            &UserController{},
            &ValidatedHandler{},
        )
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        
        if cp.GetHandlerName(0) != "my_custom_name" {
            t.Error("handler 0 name mismatch")
        }
        if cp.GetHandlerName(1) != "user_controller" {
            t.Error("handler 1 name mismatch")
        }
        if cp.GetHandlerName(2) != "validated_handler" {
            t.Error("handler 2 name mismatch")
        }
    })
}

func HandlerValidationShared(t *testing.T, cp *crudp.CrudP) {
    t.Run("Validation Passes", func(t *testing.T) {
        cp := crudp.NewDefault()
        cp.RegisterHandler(&ValidatedHandler{})
        
        ctx := context.Background()
        result, err := cp.CallHandler(ctx, 0, 'c', "some data")
        if err != nil {
            t.Errorf("unexpected error: %v", err)
        }
        if resp, ok := result.(ValidatedCreateResponse); !ok || resp.Message != "validated_created" {
            t.Errorf("expected ValidatedCreateResponse with message 'validated_created', got %v", result)
        }
    })
    
    t.Run("Validation Fails", func(t *testing.T) {
        cp := crudp.NewDefault()
        cp.RegisterHandler(&ValidatedHandler{})
        
        ctx := context.Background()
        _, err := cp.CallHandler(ctx, 0, 'c') // No data
        if err == nil {
            t.Error("expected validation error")
        }
    })
    
    t.Run("Field Validation", func(t *testing.T) {
        h := &ValidatedHandler{}
        
        if err := h.ValidateField("email", ""); err == nil {
            t.Error("expected error for empty email")
        }
        
        if err := h.ValidateField("email", "test@example.com"); err != nil {
            t.Errorf("unexpected error: %v", err)
        }
        
        if err := h.ValidateField("name", ""); err != nil {
            t.Error("non-required field should pass")
        }
    })
}

func CRUDOperationsShared(t *testing.T, cp *crudp.CrudP) {
    t.Run("Create Operation", func(t *testing.T) {
        cp := crudp.NewDefault()
        cp.RegisterHandler(&UserController{})
        
        ctx := context.Background()
        result, err := cp.CallHandler(ctx, 0, 'c', map[string]any{"name": "test"})
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        
        if result == nil {
            t.Error("expected result, got nil")
        }
        if _, ok := result.(CreateResponse); !ok {
            t.Errorf("expected CreateResponse, got %T", result)
        }
    })
    
    t.Run("Read Operation", func(t *testing.T) {
        cp := crudp.NewDefault()
        cp.RegisterHandler(&UserController{})
        
        ctx := context.Background()
        result, err := cp.CallHandler(ctx, 0, 'r', 1)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        
        if result == nil {
            t.Error("expected result, got nil")
        }
        if _, ok := result.(ReadResponse); !ok {
            t.Errorf("expected ReadResponse, got %T", result)
        }
    })
    
    t.Run("Unimplemented Action", func(t *testing.T) {
        cp := crudp.NewDefault()
        cp.RegisterHandler(&UserController{}) // Only has Create and Read
        
        ctx := context.Background()
        _, err := cp.CallHandler(ctx, 0, 'd', 1) // Delete not implemented
        if err == nil {
            t.Error("expected error for unimplemented action")
        }
    })
    
    t.Run("Invalid Handler ID", func(t *testing.T) {
        cp := crudp.NewDefault()
        cp.RegisterHandler(&UserController{})
        
        ctx := context.Background()
        _, err := cp.CallHandler(ctx, 99, 'r', 1)
        if err == nil {
            t.Error("expected error for invalid handler ID")
        }
    })
}
```

### File: `handlers_stlib_test.go`

```go
//go:build !wasm

package crudp_test

import (
    "testing"
    
    "github.com/cdvelop/crudp"
)

func TestHandlers_Stdlib(t *testing.T) {
    cp := crudp.NewDefault()
    
    t.Run("Registration", func(t *testing.T) {
        HandlerRegistrationShared(t, cp)
    })
    
    t.Run("Validation", func(t *testing.T) {
        HandlerValidationShared(t, cp)
    })
    
    t.Run("CRUD", func(t *testing.T) {
        CRUDOperationsShared(t, cp)
    })
}
```

### File: `handlers_wasm_test.go`

```go
//go:build wasm

package crudp_test

import (
    "testing"
    
    "github.com/cdvelop/crudp"
)

func TestHandlers_WASM(t *testing.T) {
    cp := crudp.NewDefault()
    
    t.Run("Registration", func(t *testing.T) {
        HandlerRegistrationShared(t, cp)
    })
    
    t.Run("Validation", func(t *testing.T) {
        HandlerValidationShared(t, cp)
    })
    
    t.Run("CRUD", func(t *testing.T) {
        CRUDOperationsShared(t, cp)
    })
}
```

---

## 2.8 Verification

```bash
# Stdlib tests
go test -v -run TestHandlers

# WASM tests  
GOOS=js GOARCH=wasm go test -v -tags wasm -run TestHandlers
```

---

## Notes

- **Optional NamedHandler:** If not implemented, reflection + `SnakeLow()` from tinystring is used
- **Optional Validator:** Only called if the handler implements it
- **Context check:** `ctx.Done()` is checked before executing to support cancellation
- **Return `any`:** Handler can return `Response`, `[]Response`, or direct value

---

> **Next step:** [STEP_03_CONFIG_SYSTEM.md](./STEP_03_CONFIG_SYSTEM.md)
