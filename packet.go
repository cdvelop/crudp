package crudp

import (
	. "github.com/cdvelop/tinystring"
)

// Packet represents both requests and responses of the protocol
type Packet struct {
	Action    byte     // action: 'c', 'r', 'u', 'd', 'e'
	HandlerID uint8    // shared index within the registration slice
	Message   string   // additional information (optional in requests, used in responses)
	Data      [][]byte // slice of encoded data, each []byte is a structure
}

// EncodePacket encodes a packet for a known handler
// This method creates a temporary CrudP instance to use its TinyBin instance
func EncodePacket(action byte, handlerID uint8, message string, data ...any) ([]byte, error) {
	// Create a temporary CrudP instance to use its TinyBin instance
	cp := New()
	return cp.EncodePacket(action, handlerID, message, data...)
}

// EncodePacket encodes a packet for a known handler using this CrudP's TinyBin instance
func (cp *CrudP) EncodePacket(action byte, handlerID uint8, message string, data ...any) ([]byte, error) {
	encoded := make([][]byte, 0, len(data))
	for _, item := range data {
		bytes, err := cp.tinyBin.Encode(item)
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

	return cp.tinyBin.Encode(packet)
}

// DecodePacket decodes a packet using a temporary TinyBin instance
func DecodePacket(data []byte, packet *Packet) error {
	// Create a temporary CrudP instance to use its TinyBin instance
	cp := New()
	return cp.DecodePacket(data, packet)
}

// DecodePacket decodes a packet using this CrudP's TinyBin instance
func (cp *CrudP) DecodePacket(data []byte, packet *Packet) error {
	return cp.tinyBin.Decode(data, packet)
}

// DecodeData decodes the packet data using a temporary TinyBin instance
func DecodeData(packet *Packet, index int, target any) error {
	if index >= len(packet.Data) {
		return Errf("index out of range")
	}
	// Create a temporary CrudP instance to use its TinyBin instance
	cp := New()
	return cp.DecodeData(packet, index, target)
}

// DecodeData decodes the packet data using this CrudP's TinyBin instance
func (cp *CrudP) DecodeData(packet *Packet, index int, target any) error {
	if index >= len(packet.Data) {
		return Errf("index out of range")
	}
	return cp.tinyBin.Decode(packet.Data[index], target)
}

// ProcessPacket automatically processes a packet and calls the corresponding handler
func (cp *CrudP) ProcessPacket(requestBytes []byte) ([]byte, error) {
	var packet Packet
	if err := cp.DecodePacket(requestBytes, &packet); err != nil {
		return cp.createErrorResponse("decode_error", err)
	}

	// Use TinyBin instance for decoding - maintain raw bytes for now
	// This preserves the original working design while using new API
	var decodedData []any
	for _, itemBytes := range packet.Data {
		// Pass raw bytes - handlers will decode with concrete types
		// This maintains compatibility with existing handlers
		decodedData = append(decodedData, itemBytes)
	}

	result, err := cp.callHandler(packet.HandlerID, packet.Action, decodedData...)
	if err != nil {
		return cp.createErrorResponse("handler_error", err)
	}

	var responseData []byte
	if bytes, ok := result.([]byte); ok {
		responseData = bytes
	} else {
		responseData, err = cp.tinyBin.Encode(result)
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

	return cp.tinyBin.Encode(responsePacket)
}
