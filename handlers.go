package crudp

import (
	"context"
	"reflect"

	. "github.com/cdvelop/tinystring"
)

// RegisterHandler prepares the shared handler table between client and server
// Receives the real implementations that act as prototypes and handlers.
func (cp *CrudP) RegisterHandler(handlers ...any) error {
	cp.handlers = make([]actionHandler, len(handlers))

	for index, handler := range handlers {
		if handler == nil {
			return Errf("handler %d is nil", index)
		}

		// Store original handler for type analysis
		cp.handlers[index].Handler = handler

		cp.bind(uint8(index), handler)
	}

	return nil
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

// callHandler searches and calls the handler directly by shared index
func (cp *CrudP) callHandler(ctx context.Context, handlerID uint8, action byte, data ...any) ([]any, error) {
	if int(handlerID) >= len(cp.handlers) {
		return nil, Errf("no handler found for id: %d", handlerID)
	}

	handler := cp.handlers[handlerID]

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

	return nil, Errf("action '%c' not implemented for handler id: %d", action, handlerID)
}

// decodeWithKnownType decodes packet data using cached type information when available
// This is the key method that enables handlers to receive concrete types instead of raw bytes
func (cp *CrudP) decodeWithKnownType(packet *Packet, handlerID uint8) ([]any, error) {

	// Validate handlerID
	if int(handlerID) >= len(cp.handlers) {
		return nil, Errf("no handler found for id: %d", handlerID)
	}

	handler := cp.handlers[handlerID].Handler
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

		// Decode bytes into the concrete type using TinyBin instance
		if err := cp.tinyBin.Decode(itemBytes, targetPtr); err != nil {
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
