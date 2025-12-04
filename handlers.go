package crudp

import (
	"context"
	"reflect"

	. "github.com/cdvelop/tinystring"
)

// getHandlerName gets the handler name
// Priority: 1) HandlerName() if implemented, 2) reflection + snake_case
func getHandlerName(handler any) string {
	// First try NamedHandler interface
	if named, ok := handler.(NamedHandler); ok {
		return named.HandlerName()
	}

	// Fallback: use reflection and convert to snake_case
	t := reflect.TypeOf(handler)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Use tinystring.SnakeLow for conversion
	// UserHandler -> user_handler
	// APIController -> api_controller
	return Convert(t.Name()).SnakeLow().String()
}

// RegisterHandler prepares the shared handler table between client and server
// Receives the real implementations that act as prototypes and handlers.
func (cp *CrudP) RegisterHandler(handlers ...any) error {
	cp.handlers = make([]actionHandler, len(handlers))

	for i, h := range handlers {
		if h == nil {
			return Errf("handler %d is nil", i)
		}

		// Get name (via interface or reflection)
		name := getHandlerName(h)

		cp.handlers[i] = actionHandler{
			name:    name,
			index:   uint8(i),
			handler: h,
		}

		cp.bind(uint8(i), h)

		if cp.log != nil {
			cp.log("registered handler:", name, "at index", i)
		}
	}

	return nil
}

// GetHandlerName returns the handler name by its ID
func (cp *CrudP) GetHandlerName(handlerID uint8) string {
	if int(handlerID) >= len(cp.handlers) {
		return ""
	}
	return cp.handlers[handlerID].name
}

// bind copies the CRUD functions without dynamic allocations
func (cp *CrudP) bind(index uint8, handler any) {
	if creator, ok := handler.(Creator); ok {
		cp.handlers[index].Create = creator.Create
	}
	if reader, ok := handler.(Reader); ok {
		cp.handlers[index].Read = reader.Read
	}
	if updater, ok := handler.(Updater); ok {
		cp.handlers[index].Update = updater.Update
	}
	if deleter, ok := handler.(Deleter); ok {
		cp.handlers[index].Delete = deleter.Delete
	}
}

// CallHandler searches and calls the handler directly by shared index
func (cp *CrudP) CallHandler(ctx context.Context, handlerID uint8, action byte, data ...any) (any, error) {
	if int(handlerID) >= len(cp.handlers) {
		return nil, Errf("no handler found for id: %d", handlerID)
	}

	handler := cp.handlers[handlerID]

	// Optional validation before executing
	if validator, ok := handler.handler.(Validator); ok {
		if err := validator.Validate(action, data...); err != nil {
			return nil, err
		}
	}

	// Check context canceled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	switch action {
	case 'c':
		if handler.Create != nil {
			return handler.Create(ctx, data...), nil
		}
	case 'r':
		if handler.Read != nil {
			return handler.Read(ctx, data...), nil
		}
	case 'u':
		if handler.Update != nil {
			return handler.Update(ctx, data...), nil
		}
	case 'd':
		if handler.Delete != nil {
			return handler.Delete(ctx, data...), nil
		}
	}

	return nil, Errf("action '%c' not implemented for handler: %s", action, handler.name)
}
// decodeWithKnownType decodes packet data using cached type information when available
// This is the key method that enables handlers to receive concrete types instead of raw bytes
func (cp *CrudP) decodeWithKnownType(packet *Packet, handlerID uint8) ([]any, error) {

	// Validate handlerID
	if int(handlerID) >= len(cp.handlers) {
		return nil, Errf("no handler found for id: %d", handlerID)
	}

	handler := cp.handlers[handlerID].handler
	if handler == nil {
		if cp.log != nil {
			cp.log("decodeWithKnownType: handler is nil, fallback to raw bytes")
		}
		return cp.decodeWithRawBytes(packet)
	}

	// Get the handler type to determine what concrete type to decode to
	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	var concreteType reflect.Type

	// If handler is a pointer (e.g., &User{}), get the element type (User)
	if handlerType.Kind() == reflect.Ptr {
		concreteType = handlerType.Elem()
	} else {
		// If handler is a value type, use it directly
		concreteType = handlerType
	}

	if concreteType == nil {
		// Fallback to raw bytes if we can't determine the type
		return cp.decodeWithRawBytes(packet)
	}

	decodedData := make([]any, 0, len(packet.Data))

	for _, itemBytes := range packet.Data {

		// NOTE: Using handler directly - this reuses the same instance
		// This is a known limitation. For production use, implement
		// a proper instance factory based on your specific types.
		targetPtr := handler

		// Decode bytes into the concrete type using codec
		if err := cp.codec.Decode(itemBytes, targetPtr); err != nil {
			return nil, err
		}

		decodedData = append(decodedData, targetPtr)
	}

	return decodedData, nil
}

// decodeWithRawBytes decodes packet data as raw bytes (current working method)
func (cp *CrudP) decodeWithRawBytes(packet *Packet) ([]any, error) {
	decodedData := make([]any, 0, len(packet.Data))
	for _, itemBytes := range packet.Data {
		// Pass raw bytes for handlers to decode with concrete types
		decodedData = append(decodedData, itemBytes)
	}
	return decodedData, nil
}
