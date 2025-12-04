//go:build wasm

package crudp_test

import "testing"

func TestBroker_WASM(t *testing.T) {
    t.Run("Consolidation", func(t *testing.T) {
        BrokerConsolidationShared(t)
    })

    t.Run("Flush", func(t *testing.T) {
        BrokerFlushShared(t)
    })

    t.Run("Clear", func(t *testing.T) {
        BrokerClearShared(t)
    })

    t.Run("EnqueuePacket", func(t *testing.T) {
        EnqueuePacketShared(t)
    })
}
