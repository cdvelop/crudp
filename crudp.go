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
	broker   *broker      // Add this field
}

// noopLogger is the default logger that does nothing
func noopLogger(...any) {}

// New creates a new CrudP instance with configuration
func New(cfg *Config) *CrudP {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	codec := cfg.Codec
	if codec == nil {
		codec = getDefaultCodec()
	}

	cp := &CrudP{
		config: cfg,
		codec:  codec,
		log:    noopLogger,
	}

	// Initialize broker
	cp.broker = newBroker(cfg, codec)

	return cp
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

// Broker returns the broker for advanced configuration
func (cp *CrudP) Broker() *broker {
	return cp.broker
}

// EnqueuePacket queues a packet for batch sending
func (cp *CrudP) EnqueuePacket(handlerID uint8, action byte, reqID string, data any) error {
	encoded, err := cp.codec.Encode(data)
	if err != nil {
		return err
	}
	cp.broker.Enqueue(handlerID, action, reqID, encoded)
	return nil
}

// routeToSSE encodes data and sends it to the appropriate SSE broadcast channels.
func (cp *CrudP) routeToSSE(data any, broadcast []string, handlerID uint8) {
	if cp.log != nil {
		cp.log("routeToSSE called for handler", handlerID, "with broadcast targets:", broadcast)
	}

	encodedData, err := cp.codec.Encode(data)
	if err != nil {
		if cp.log != nil {
			cp.log("routeToSSE encoding error:", err)
		}
		return
	}

	// In a real implementation, this would send the encodedData to the specified broadcast channels.
	// For now, we will just log the encoded data.
	if cp.log != nil {
		for _, channel := range broadcast {
			cp.log("Broadcasting to channel:", channel, "data:", string(encodedData))
		}
	}
}
