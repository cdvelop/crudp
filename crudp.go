package crudp

import (
	"github.com/cdvelop/tinybin"
)

// actionHandler groups the CRUD functions for a record index
type actionHandler struct {
	Create func(...any) (any, error)
	Read   func(...any) (any, error)
	Update func(...any) (any, error)
	Delete func(...any) (any, error)

	// Store original handler for type analysis
	Handler any
}

// CrudP handles automatic processing of handlers
// Uses slices instead of maps for TinyGo compatibility
type CrudP struct {
	handlers []actionHandler  // Dynamic table of handlers shared by index
	tinyBin  *tinybin.TinyBin // TinyBin instance for encoding/decoding with caching

	log func(msg ...any) // Optional logging function
}

// New creates a new CrudP instance
// Optional argument: logging function func(msg ...any)
// eg: crudp.New(func(msg ...any) { log.Printf("CrudP: %v", msg) })
func New(args ...any) *CrudP {

	var log func(msg ...any) // default no logging

	// Parse optional arguments
	for _, arg := range args {
		if lf, ok := arg.(func(msg ...any)); ok {
			log = lf
		}
	}

	return &CrudP{
		tinyBin: tinybin.New(), // Initialize TinyBin instance for caching and performance
		log:     log,
	}
}
