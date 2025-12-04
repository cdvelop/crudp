package crudp

import (
	"context"

	"github.com/cdvelop/tinybin"
)

// actionHandler groups the CRUD functions for a record index
type actionHandler struct {
	Create func(context.Context, ...any) any
	Read   func(context.Context, ...any) any
	Update func(context.Context, ...any) any
	Delete func(context.Context, ...any) any

	// Store original handler for type analysis
	Handler any
}

// CrudP handles automatic processing of handlers
// Uses slices instead of maps for TinyGo compatibility
type CrudP struct {
	handlers []actionHandler  // Dynamic table of handlers shared by index
	tinyBin  *tinybin.TinyBin // TinyBin instance for encoding/decoding with caching

	log func(msg ...any) // Optional logging function

	apiEndpoint string // HTTP endpoint for binary protocol (default "/api")
}

// New creates a new CrudP instance
// Optional arguments: logging function func(msg ...any), apiEndpoint string
// eg: crudp.New(func(msg ...any) { log.Printf("CrudP: %v", msg) }, "/api/v1")
func New(args ...any) *CrudP {

	var log func(msg ...any) // default no logging
	var apiEndpoint = "/api" // default endpoint

	// Parse optional arguments
	for _, arg := range args {
		if lf, ok := arg.(func(msg ...any)); ok {
			log = lf
		}
		if ep, ok := arg.(string); ok {
			apiEndpoint = ep
		}
	}

	return &CrudP{
		tinyBin:     tinybin.New(), // Initialize TinyBin instance for caching and performance
		log:         log,
		apiEndpoint: apiEndpoint,
	}
}

// routeToSSE handles Server-Sent Events routing based on broadcast targets
func (cp *CrudP) routeToSSE(data any, broadcast []string, handlerID uint8) {
	// TODO: Implement SSE broker routing
	// For now, just log the routing information
	if cp.log != nil {
		cp.log("SSE Route: handlerID=%d, broadcast=%v, data=%v", handlerID, broadcast, data)
	}
}
