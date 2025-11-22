# Architectural Pattern: Interface-Based Modular System

**Objective:** Plug-and-play modular system with automatic route registration via Go interfaces. Modules are completely decoupled and testable in isolation.

---

## **Core Principles**

1.  **Inversion of Control:** Modules define their own dependencies via interfaces.
2.  **Automatic Discovery:** Router discovers and registers routes by checking interface implementations.
3.  **Global Middleware:** Modules publish middleware, router applies it to ALL routes (secure by default).
4.  **Isolation:** Each module can be tested independently with mocks.

---

## **1. Module Interfaces**

**File: `pkg/router/interfaces.go`**
```go
package router

import "net/http"

type Module interface {
    ModuleName() string // Returns URL path prefix (e.g., "persons")
}

type CRUDHandler interface {
    List(w http.ResponseWriter, r *http.Request)   // GET /<moduleName>
    Create(w http.ResponseWriter, r *http.Request) // POST /<moduleName>/new
    Get(w http.ResponseWriter, r *http.Request)    // GET /<moduleName>/{id}, /new, /{id}/edit
    Update(w http.ResponseWriter, r *http.Request) // POST /<moduleName>/{id}/edit
    Delete(w http.ResponseWriter, r *http.Request) // POST /<moduleName>/{id}/delete
}

type CustomHandler interface {
    RegisterCustomRoutes(mux *http.ServeMux, middleware []func(http.HandlerFunc) http.HandlerFunc)
}

type MiddlewareProvider interface {
    Middleware() []func(http.HandlerFunc) http.HandlerFunc
}
```

## **2. Router with Automatic Discovery**

**File: `pkg/router/router.go`**
```go
package router

import (
    "fmt"
    "net/http"
)

type Router struct {
    modules []Module
}

func NewRouter() *Router {
    return &Router{modules: make([]Module, 0)}
}

func (r *Router) RegisterModules(modules []any) {
    for _, m := range modules {
        if mod, ok := m.(Module); ok {
            r.modules = append(r.modules, mod)
        }
    }
}

func (r *Router) BuildMux() *http.ServeMux {
    mux := http.NewServeMux()
    
    // Collect ALL middleware from ALL modules (applied globally)
    var allMiddleware []func(http.HandlerFunc) http.HandlerFunc
    for _, mod := range r.modules {
        if mw, ok := mod.(MiddlewareProvider); ok {
            allMiddleware = append(allMiddleware, mw.Middleware()...)
        }
    }
    
    // Register routes with global middleware
    for _, mod := range r.modules {
        basePath := mod.ModuleName()
        
        // Register CRUD routes (all use same global middleware)
        if crud, ok := mod.(CRUDHandler); ok {
            mux.HandleFunc(fmt.Sprintf("GET /%s", basePath), chain(crud.List, allMiddleware...))
            mux.HandleFunc(fmt.Sprintf("GET /%s/new", basePath), chain(crud.Get, allMiddleware...))
            mux.HandleFunc(fmt.Sprintf("POST /%s/new", basePath), chain(crud.Create, allMiddleware...))
            mux.HandleFunc(fmt.Sprintf("GET /%s/{id}", basePath), chain(crud.Get, allMiddleware...))
            mux.HandleFunc(fmt.Sprintf("GET /%s/{id}/edit", basePath), chain(crud.Get, allMiddleware...))
            mux.HandleFunc(fmt.Sprintf("POST /%s/{id}/edit", basePath), chain(crud.Update, allMiddleware...))
            mux.HandleFunc(fmt.Sprintf("POST /%s/{id}/delete", basePath), chain(crud.Delete, allMiddleware...))
        }
        
        // Register custom routes (also use global middleware)
        if custom, ok := mod.(CustomHandler); ok {
            custom.RegisterCustomRoutes(mux, allMiddleware)
        }
    }
    
    return mux
}

func chain(handler http.HandlerFunc, middleware ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
    for i := len(middleware) - 1; i >= 0; i-- {
        handler = middleware[i](handler)
    }
    return handler
}
```

## **3. Module Implementation**

Each module defines its own interfaces and is completely isolated.

**File: `modules/person/person.go`**
```go
package person

import "net/http"

type IDGenerator interface {
    GetNewID() string
}

type DB interface {
    Create(any) error
    First(any, ...any) error
    Find(any, ...any) error
    Save(any) error
    Delete(any, ...any) error
}

type handler struct {
    db    DB
    idGen IDGenerator
}

func New(db any, idGen any) any {
    return &handler{
        db:    db.(DB),
        idGen: idGen.(IDGenerator),
    }
}

// Compile-time interface checks
var _ interface{ ModuleName() string } = (*handler)(nil)

func (h *handler) ModuleName() string { return "persons" }

func (h *handler) List(w http.ResponseWriter, r *http.Request)   { /* ... */ }
func (h *handler) Create(w http.ResponseWriter, r *http.Request) {
    person := Person{
        ID:   h.idGen.GetNewID(),  // Manual ID generation
        Name: r.FormValue("name"),
    }
    h.db.Create(&person)
}
func (h *handler) Get(w http.ResponseWriter, r *http.Request)    { /* ... */ }
func (h *handler) Update(w http.ResponseWriter, r *http.Request) { /* ... */ }
func (h *handler) Delete(w http.ResponseWriter, r *http.Request) { /* ... */ }
```

**With Middleware (applied globally to ALL routes):**
```go
func (h *handler) Middleware() []func(http.HandlerFunc) http.HandlerFunc {
    return []func(http.HandlerFunc) http.HandlerFunc{
        authMiddleware,    // Will be applied to ALL modules
        roleMiddleware,    // Will be applied to ALL modules
    }
}
```

## **4. Module Registry**

All modules are initialized in one place with their dependencies.

**File: `modules/modules.go`**
```go
package modules

import (
    "app-platform/modules/person"
    "app-platform/modules/patient"
)

func Init(db any) []any {

    idGen, err := unixid.NewUnixID()
    if err != nil {
        panic(err)
    }

    return []any{
        person.Add(db, idGen),
        patient.Add(db, idGen),
        // Add new modules here
    }
}
```

## **5. Main Assembly**

**File: `web/server.go`**
```go
package main

import (
    "net/http"
    "app-platform/pkg/database"
    "app-platform/pkg/router"
    "app-platform/modules"
)

func main() {
    db := database.Connect()
    
    r := router.NewRouter()
    r.RegisterModules(modules.Init(db))
    mux := r.BuildMux()
    
    http.ListenAndServe(":8080", mux)
}
```

## **6. Testing**

**File: `modules/person/person_test.go`**
```go
package person

import "testing"

type mockDB struct {
    data map[string]any
}

func (m *mockDB) Create(v any) error { /* ... */ }
func (m *mockDB) First(v any, conds ...any) error { /* ... */ }
// Implement only what person.DB requires

type mockIDGen struct{}

func (m *mockIDGen) GetNewID() string {
    return "1234567890123456789"
}

func TestPersonCreate(t *testing.T) {
    db := &mockDB{data: make(map[string]any)}
    idGen := &mockIDGen{}
    h := New(db, idGen)
    // Test in complete isolation
}
```