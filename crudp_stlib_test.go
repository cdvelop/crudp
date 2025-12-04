//go:build !wasm

package crudp_test

import (
	"testing"
)

func TestCrudP_BasicFunctionality(t *testing.T) {
	CrudPBasicFunctionalityShared(t)
}
