//go:build !wasm

package crudp

import (
	"io"
	"net/http"
)

// Optional: Add custom HTTP routes (e.g., /upload, /export)
type HttpRouteProvider interface {
	RegisterRoutes(mux *http.ServeMux)
}

// Optional: Provide global middleware (authentication, logging, etc.)
type MiddlewareProvider interface {
	Middleware(next http.Handler) http.Handler
}

// BuildRouter creates the complete HTTP handler with routes and middleware
func (cp *CrudP) BuildRouter() http.Handler {
	mux := http.NewServeMux()

	// 1. Register CRUDP's binary protocol endpoint (configurable)
	mux.HandleFunc(cp.config.APIEndpoint, cp.handleBinaryProtocol)

	// 2. Collect all global middleware from handlers
	var globalMiddleware []func(http.Handler) http.Handler
	for _, h := range cp.handlers {
		if mwProvider, ok := h.handler.(MiddlewareProvider); ok {
			globalMiddleware = append(globalMiddleware, mwProvider.Middleware)
		}
	}

	// 3. Let handlers register their custom HTTP routes
	for _, h := range cp.handlers {
		if routeProvider, ok := h.handler.(HttpRouteProvider); ok {
			routeProvider.RegisterRoutes(mux)
		}
	}

	// 4. Wrap everything with global middleware (applied in registration order)
	handler := http.Handler(mux)
	for _, mw := range globalMiddleware {
		handler = mw(handler)
	}

	return handler
}

// handleBinaryProtocol processes CRUDP binary batch requests
func (cp *CrudP) handleBinaryProtocol(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	response, err := cp.ProcessBatch(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(response)
}
