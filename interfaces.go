package crudp

// Separate CRUD interfaces - handlers can implement only the ones they need
type Creator interface {
	Create(data ...any) (any, error)
}

type Reader interface {
	Read(data ...any) (any, error)
}

type Updater interface {
	Update(data ...any) (any, error)
}

type Deleter interface {
	Delete(data ...any) (any, error)
}
