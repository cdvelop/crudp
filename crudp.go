package crudp

import (
	"github.com/cdvelop/tinybin"
	"github.com/cdvelop/tinyreflect"
)

// ActionHandler groups the CRUD functions for a record index
type ActionHandler struct {
	Create func(...any) (any, error)
	Read   func(...any) (any, error)
	Update func(...any) (any, error)
	Delete func(...any) (any, error)

	// Type caching for performance with TinyBin
	Type *tinyreflect.Type

	// Store original handler for type analysis
	Handler any
}

// CrudP handles automatic processing of handlers
// Uses slices instead of maps for TinyGo compatibility
type CrudP struct {
	handlers []ActionHandler  // Dynamic table of handlers shared by index
	tinyBin  *tinybin.TinyBin // TinyBin instance for encoding/decoding with caching
}

// New creates a new CrudP instance
func New() *CrudP {
	return &CrudP{
		tinyBin: tinybin.New(), // Initialize TinyBin instance for caching and performance
	}
}
