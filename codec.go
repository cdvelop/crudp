package crudp

// Codec interface for serialization (replaces direct tinybin dependency)
type Codec interface {
	Encode(data any) ([]byte, error)
	Decode(data []byte, v any) error
}
