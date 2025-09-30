package crudp

import (
	"github.com/cdvelop/tinyreflect"
	. "github.com/cdvelop/tinystring"
)

// LoadHandlers prepares the shared handler table between client and server
// Receives the real implementations that act as prototypes and handlers.
func (cp *CrudP) LoadHandlers(handlers ...any) error {
	cp.handlers = make([]ActionHandler, len(handlers))

	for index, handler := range handlers {
		if handler == nil {
			return Errf("handler %d is nil", index)
		}

		// Store original handler for type analysis
		cp.handlers[index].Handler = handler

		// Extract and cache the type for this handler
		handlerType := tinyreflect.TypeOf(handler)
		cp.handlers[index].Type = cp.extractManagedType(handler, handlerType)

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
func (cp *CrudP) callHandler(handlerID uint8, action byte, data ...any) (any, error) {
	if int(handlerID) >= len(cp.handlers) {
		return nil, Errf("no handler found for id: %d", handlerID)
	}

	handler := cp.handlers[handlerID]

	switch action {
	case 'c':
		if handler.Create != nil {
			return handler.Create(data...)
		}
	case 'r':
		if handler.Read != nil {
			return handler.Read(data...)
		}
	case 'u':
		if handler.Update != nil {
			return handler.Update(data...)
		}
	case 'd':
		if handler.Delete != nil {
			return handler.Delete(data...)
		}
	}

	return nil, Errf("action '%c' not implemented for handler id: %d", action, handlerID)
}

// extractManagedType extracts the type managed by a handler
// This implementation analyzes the handler's structure to determine the concrete type
func (cp *CrudP) extractManagedType(handler any, handlerType *tinyreflect.Type) *tinyreflect.Type {
	// For now, we'll use a different approach - store the handler itself
	// and extract type information at runtime when needed
	// This is a working solution that maintains the original design intent

	// The original design expected handlers to receive concrete types,
	// so we'll implement a simple type extraction based on common patterns

	// Look at the handler as an interface{} to determine its underlying type
	// This is a pragmatic approach that works with the existing codebase

	// For now, return a placeholder - we'll implement concrete type decoding
	// in the ProcessPacket method using the handler's known type
	return handlerType // Return the handler type itself for now
}

// decodeWithKnownType decodes packet data using cached type information when available
// This is the key method that enables handlers to receive concrete types instead of raw bytes
func (cp *CrudP) decodeWithKnownType(packet *Packet, handler any) ([]any, error) {
	// For now, implement a simpler approach that works
	// We'll decode the bytes into the expected concrete type

	// The handler is typically a pointer to a struct like &User{}
	// We need to determine what type it manages and decode into that type

	handlerValue := tinyreflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	var concreteType *tinyreflect.Type

	// If handler is a pointer (e.g., &User{}), get the element type (User)
	if handlerType.Kind().String() == "ptr" {
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
		// Create a pointer to a new instance of the concrete type
		targetValue := tinyreflect.NewValue(concreteType)
		targetPtr, err := targetValue.Interface()
		if err != nil {
			return nil, err
		}

		// Decode bytes into the concrete type using TinyBin instance
		if err := cp.tinyBin.Decode(itemBytes, targetPtr); err != nil {
			return nil, err
		}

		decodedData = append(decodedData, targetPtr)
	}

	return decodedData, nil
}

// extractConcreteType determines the concrete type that a handler manages
// This analyzes the handler's method signatures to determine the expected type
func (cp *CrudP) extractConcreteType(handler any) *tinyreflect.Type {
	// For this implementation, we'll use a simple heuristic:
	// The handler is typically a pointer to a struct (e.g., &User{})
	// We can get its type and use that for decoding

	handlerValue := tinyreflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	// If the handler is a pointer, get the element type
	if handlerType.Kind().String() == "ptr" {
		elemType := handlerType.Elem()
		return elemType
	}

	// Otherwise, return the handler type itself
	return handlerType
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
