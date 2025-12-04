//go:build !wasm

package crudp_test

import (
	"testing"

	"github.com/cdvelop/crudp"
)

func TestHandlers_Stdlib(t *testing.T) {
	cp := crudp.NewDefault()

	t.Run("Registration", func(t *testing.T) {
		HandlerRegistrationShared(t, cp)
	})

	t.Run("Validation", func(t *testing.T) {
		HandlerValidationShared(t, cp)
	})

	t.Run("CRUD", func(t *testing.T) {
		CRUDOperationsShared(t, cp)
	})
}
