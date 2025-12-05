# Handler Registration

## Core Concepts

Handlers are the core of a CRUDP application. They implement the business logic for your data models. Handlers are registered with a `CrudP` instance, which then dispatches incoming requests to the appropriate handler and method.

## CRUD Interfaces

Handlers implement one or more of the following interfaces to handle CRUD operations:

**File: `interfaces.go`**
```go
package crudp

import "context"

// Creator handles create operations.
type Creator interface {
    Create(ctx context.Context, data ...any) any
}

// Reader handles read operations.
type Reader interface {
    Read(ctx context.Context, data ...any) any
}

// Updater handles update operations.
type Updater interface {
    Update(ctx context.Context, data ...any) any
}

// Deleter handles delete operations.
type Deleter interface {
    Delete(ctx context.Context, data ...any) any
}
```

**Key Points:**

-   Each CRUD method now returns a single `any` value.
-   This `any` value can be a simple struct, a slice of structs, or a `Response` interface for more advanced scenarios like SSE broadcasting.

## Handler Naming

CRUDP automatically determines a handler's name, which is used to route requests. This can be done in two ways:

1.  **By Convention (Reflection):** If a handler does not explicitly provide a name, CRUDP will use reflection to get the type name of the handler struct and convert it to `snake_case`. For example, a `UserHandler` struct will be named `"user_handler"`.
2.  **Explicitly (NamedHandler):** A handler can implement the `NamedHandler` interface to provide a custom name.

**File: `interfaces.go`**
```go
// NamedHandler allows a handler to provide a custom name.
type NamedHandler interface {
    HandlerName() string
}
```

## Validation

CRUDP provides two optional interfaces for data validation:

-   `Validator`: For validating the entire data payload before a CRUD operation is executed.
-   `FieldValidator`: For validating individual fields, often used for UI feedback.

**File: `interfaces.go`**
```go
// Validator validates the entire data payload.
type Validator interface {
    Validate(action byte, data ...any) error
}

// FieldValidator validates a single field.
type FieldValidator interface {
    ValidateField(fieldName string, value string) error
}
```

If a handler implements the `Validator` interface, its `Validate` method will be called before the corresponding CRUD method is executed.

## `RegisterHandler`

The `RegisterHandler` method on the `CrudP` instance is used to register one or more handlers.

```go
func (cp *CrudP) RegisterHandler(handlers ...any) error
```

During registration, CRUDP will:
1.  Determine the handler's name.
2.  Bind the handler's `Create`, `Read`, `Update`, and `Delete` methods.
3.  Cache the handler for later use.
