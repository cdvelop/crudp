//go:build !wasm

package crudp_test

import (
	"testing"
)

func TestHandlerInstanceReuse(t *testing.T) {
	HandlerInstanceReuseShared(t)
}

func TestHandlerInstanceReuse_KNOWN_LIMITATION(t *testing.T) {
	HandlerInstanceReuseKnownLimitationShared(t)
}

func TestConcurrentHandlerAccess(t *testing.T) {
	ConcurrentHandlerAccessShared(t)
}
