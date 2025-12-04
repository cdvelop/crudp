//go:build wasm

package crudp_test

import (
	"testing"
)

func TestCrudP_ErrorHandling(t *testing.T) {
	CrudPErrorHandlingShared(t)
}
