//go:build !wasm

package crudp_test

import (
	"testing"
	"github.com/cdvelop/crudp"
)

func TestPacketResult_Stdlib(t *testing.T) {
	cp := crudp.NewDefault()
	t.Run("MessageType", func(t *testing.T) {
		PacketResultMessageTypeShared(t)
	})

	t.Run("ActionConversion", func(t *testing.T) {
		ActionConversionShared(t)
	})

	t.Run("SSERouting", func(t *testing.T) {
		SSERoutingShared(t, cp)
	})
}
