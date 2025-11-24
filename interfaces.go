package crudp

import "context"

// Separate CRUD interfaces - handlers can implement only the ones they need
type Creator interface {
	Create(ctx context.Context, data ...any) (any, error)
}

type Reader interface {
	Read(ctx context.Context, data ...any) (any, error)
}

type Updater interface {
	Update(ctx context.Context, data ...any) (any, error)
}

type Deleter interface {
	Delete(ctx context.Context, data ...any) (any, error)
}
