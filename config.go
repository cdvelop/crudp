package crudp

import "context"

// UserProvider provides user identification for SSE routing
type UserProvider interface {
	GetUserID(ctx context.Context) string
}

// Config contains CrudP configuration
// NOTE: Logger is NOT here - configured via SetLogger()
type Config struct {
	// Codec for serialization. Default: tinyjson.New()
	Codec Codec

	// UseBinary uses binary encoding. Default: false (JSON)
	UseBinary bool

	// APIEndpoint for batch requests. Default: "/api"
	APIEndpoint string

	// SSEEndpoint for event stream. Default: "/events"
	SSEEndpoint string

	// BatchWindow in milliseconds. Default: 50
	BatchWindow int

	// MaxRetries for failed requests. Default: 3
	MaxRetries int

	// RetryInterval base in ms. Default: 1000
	RetryInterval int

	// Port for HTTP server (server only). Default: ":6060"
	Port string

	// UserProvider for SSE routing (server only). Default: nil
	UserProvider UserProvider

	// ServerURL base (client only). Default: "" (same origin)
	ServerURL string

	// OnMessage callback for notifications (client only)
	OnMessage func(msgType uint8, message string)
}

// DefaultConfig returns configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Codec:         nil, // Will assign tinyjson in New()
		UseBinary:     false,
		APIEndpoint:   "/api",
		SSEEndpoint:   "/events",
		BatchWindow:   50,
		MaxRetries:    3,
		RetryInterval: 1000,
		Port:          ":6060",
	}
}
