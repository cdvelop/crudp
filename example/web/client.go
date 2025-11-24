//go:build wasm

package main

import (
	"github.com/cdvelop/crudp/example/pkg/router"
)

func main() {
	// Get CRUDP client for WASM
	_ = router.NewRouter() // TODO: Implement WASM client logic

	select {}
}
