//go:build !wasm

package crudp_test

import "testing"

func TestPacketResult_Stdlib(t *testing.T) {
	t.Run("MessageType", func(t *testing.T) {
		PacketResultMessageTypeShared(t)
	})

	t.Run("ActionConversion", func(t *testing.T) {
		ActionConversionShared(t)
	})
}
