package crudp

// routeToSSE encodes data and sends it to the appropriate SSE broadcast channels.
func (cp *CrudP) routeToSSE(data any, broadcast []string, handlerID uint8) {
	cp.log("routeToSSE called for handler", handlerID, "with broadcast targets:", broadcast)

	encodedData, err := cp.codec.Encode(data)
	if err != nil {
		cp.log("routeToSSE encoding error:", err)
		return
	}

	// In a real implementation, this would send the encodedData to the specified broadcast channels.
	// For now, we will just log the encoded data.
	for _, channel := range broadcast {
		cp.log("Broadcasting to channel:", channel, "data:", string(encodedData))
	}
}
