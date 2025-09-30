# CRUDP Implementation Details

```go
package crudp

import (
	"github.com/cdvelop/tinybin"
	. "github.com/cdvelop/tinystring"
)

// Separate CRUD interfaces - handlers can implement only the ones they need
type Creator interface {
	Create(data ...any) (any, error)
}

type Reader interface {
	Read(data ...any) (any, error)
}

type Updater interface {
	Update(data ...any) (any, error)
}

type Deleter interface {
	Delete(data ...any) (any, error)
}

// ActionHandler groups the CRUD functions for a record index
type ActionHandler struct {
	Create func(...any) (any, error)
	Read   func(...any) (any, error)
	Update func(...any) (any, error)
	Delete func(...any) (any, error)
}

// CrudP handles automatic processing of handlers
// Uses slices instead of maps for TinyGo compatibility
type CrudP struct {
	handlers []ActionHandler // Dynamic table of handlers shared by index
}

// New creates a new CrudP instance
func New() *CrudP {
	return &CrudP{}
}

// Packet represents both requests and responses of the protocol
type Packet struct {
	Action    byte     // action: 'c', 'r', 'u', 'd', 'e'
	HandlerID uint8    // shared index within the registration slice
	Message   string   // additional information (optional in requests, used in responses)
	Data      [][]byte // slice of encoded data, each []byte is a structure
}

// EncodePacket encodes a packet for a known handler
func EncodePacket(action byte, handlerID uint8, message string, data ...any) ([]byte, error) {
	encoded := make([][]byte, 0, len(data))
	for _, item := range data {
		bytes, err := tinybin.Encode(item)
		if err != nil {
			return nil, err
		}
		encoded = append(encoded, bytes)
	}

	packet := Packet{
		Action:    action,
		HandlerID: handlerID,
		Message:   message,
		Data:      encoded,
	}

	return tinybin.Encode(packet)
}

// DecodePacket decodes a packet
func DecodePacket(data []byte, packet *Packet) error {
	return tinybin.Decode(data, packet)
}

// LoadHandlers prepares the shared handler table between client and server
// Receives the real implementations that act as prototypes and handlers.
func (cp *CrudP) LoadHandlers(handlers ...any) error {
	cp.handlers = make([]ActionHandler, len(handlers))
	
	for index, handler := range handlers {
		if handler == nil {
			return Errf("handler %d is nil", index)
		}
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

// ProcessPacket automatically processes a packet and calls the corresponding handler
func (cp *CrudP) ProcessPacket(requestBytes []byte) ([]byte, error) {
	var packet Packet
	if err := DecodePacket(requestBytes, &packet); err != nil {
		return cp.createErrorResponse("decode_error", err)
	}

	var decodedData []any
	for _, itemBytes := range packet.Data {
		var item any
		if err := tinybin.Decode(itemBytes, &item); err != nil {
			return cp.createErrorResponse("data_decode_error", err)
		}
		decodedData = append(decodedData, item)
	}

	result, err := cp.callHandler(packet.HandlerID, packet.Action, decodedData...)
	if err != nil {
		return cp.createErrorResponse("handler_error", err)
	}

	var responseData []byte
	if bytes, ok := result.([]byte); ok {
		responseData = bytes
	} else {
		responseData, err = tinybin.Encode(result)
		if err != nil {
			return cp.createErrorResponse("encode_error", err)
		}
	}

	responsePacket := Packet{
		Action:    packet.Action,
		HandlerID: packet.HandlerID,
		Message:   "success",
		Data:      [][]byte{responseData},
	}

	return tinybin.Encode(responsePacket)
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

// createErrorResponse creates an efficient error response
func (cp *CrudP) createErrorResponse(message string, err error) ([]byte, error) {
	errorMsg := Errf("%s: %v", message, err).Error()
	packet := Packet{
		Action:    'e',
		HandlerID: 0,
		Message:   errorMsg,
		Data:      nil,
	}
	return tinybin.Encode(packet)
}

// DecodeData decodes the packet data
func DecodeData(packet *Packet, index int, target any) error {
	if index >= len(packet.Data) {
		return Errf("index out of range")
	}
	return tinybin.Decode(packet.Data[index], target)
}