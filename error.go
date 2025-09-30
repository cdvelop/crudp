package crudp

import (
	. "github.com/cdvelop/tinystring"
)

// createErrorResponse creates an efficient error response
func (cp *CrudP) createErrorResponse(message string, err error) ([]byte, error) {
	errorMsg := Errf("%s: %v", message, err).Error()
	packet := Packet{
		Action:    'e',
		HandlerID: 0,
		Message:   errorMsg,
		Data:      nil,
	}
	return cp.tinyBin.Encode(packet)
}
