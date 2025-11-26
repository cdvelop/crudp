package crudp

import "context"

// Response interface that handlers return for routing
type Response interface {
	Response() (data any, broadcast []string, err error)
}

// Separate CRUD interfaces - handlers can implement only the ones they need
type Creator interface {
	Create(ctx context.Context, data ...any) []any
}

type Reader interface {
	Read(ctx context.Context, data ...any) []any
}

type Updater interface {
	Update(ctx context.Context, data ...any) []any
}

type Deleter interface {
	Delete(ctx context.Context, data ...any) []any
}
