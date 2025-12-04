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
		cp := crudp.New()
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
		cp := crudp.New()
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
		cp := crudp.New()
		err := cp.RegisterHandler(nil)
		if err == nil {
			t.Error("expected error for nil handler")
		}
	})

	t.Run("Multiple Handlers", func(t *testing.T) {
		cp := crudp.New()
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
		cp := crudp.New()
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
		cp := crudp.New()
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
		cp := crudp.New()
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
		cp := crudp.New()
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
		cp := crudp.New()
		cp.RegisterHandler(&UserController{}) // Only has Create and Read

		ctx := context.Background()
		_, err := cp.CallHandler(ctx, 0, 'd', 1) // Delete not implemented
		if err == nil {
			t.Error("expected error for unimplemented action")
		}
	})

	t.Run("Invalid Handler ID", func(t *testing.T) {
		cp := crudp.New()
		cp.RegisterHandler(&UserController{})

		ctx := context.Background()
		_, err := cp.CallHandler(ctx, 99, 'r', 1)
		if err == nil {
			t.Error("expected error for invalid handler ID")
		}
	})
}
