//go:build !wasm

package crudp_test

import "testing"

func TestConfig_Stdlib(t *testing.T) {
	t.Run("Defaults", func(t *testing.T) {
		ConfigDefaultsShared(t)
	})

	t.Run("NewWithConfig", func(t *testing.T) {
		NewWithConfigShared(t)
	})

	t.Run("Logger", func(t *testing.T) {
		LoggerConfigShared(t)
	})

	t.Run("Codec", func(t *testing.T) {
		CodecShared(t)
	})
}
