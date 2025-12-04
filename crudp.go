package crudp

import (
	"context"
)

// actionHandler groups CRUD functions for a registration index
type actionHandler struct {
	name    string
	index   uint8
	handler any
	Create  func(context.Context, ...any) any
	Read    func(context.Context, ...any) any
	Update  func(context.Context, ...any) any
	Delete  func(context.Context, ...any) any
}

// CrudP handles automatic handler processing
// Uses slices instead of maps for TinyGo compatibility
type CrudP struct {
	config   *Config
	handlers []actionHandler
	codec    Codec
	log      func(...any) // Never nil - uses no-op by default
}

// noopLogger is the default logger that does nothing
func noopLogger(...any) {}

// New creates a new CrudP instance with configuration
func New(cfg *Config) *CrudP {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Assign default codec if not provided
	codec := cfg.Codec
	if codec == nil {
		codec = getDefaultCodec()
	}

	return &CrudP{
		config: cfg,
		codec:  codec,
		log:    noopLogger, // Logger disabled by default
	}
}

// NewDefault creates CrudP with default configuration
func NewDefault() *CrudP {
	return New(nil)
}

// SetLogger configures a custom logging function
// Pass nil to restore no-op logger
func (cp *CrudP) SetLogger(logger func(...any)) {
	if logger == nil {
		cp.log = noopLogger
		return
	}
	cp.log = logger
}

// DisableLogger disables logging
func (cp *CrudP) DisableLogger() {
	cp.log = noopLogger
}

// Config returns the current configuration (read-only)
func (cp *CrudP) Config() *Config {
	return cp.config
}

// Codec returns the current codec
func (cp *CrudP) Codec() Codec {
	return cp.codec
}

// SetCodec allows changing the codec at runtime
func (cp *CrudP) SetCodec(codec Codec) {
	if codec != nil {
		cp.codec = codec
	}
}
