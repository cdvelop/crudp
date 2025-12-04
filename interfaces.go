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
