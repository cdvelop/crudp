package router

import (
	"github.com/cdvelop/crudp"
	"github.com/cdvelop/crudp/example/modules"
)

func NewRouter() *crudp.CrudP {
	cp := crudp.New()

	// Get handlers from modules
	handlers := modules.Init()

	// Register handlers in CRUDP
	cp.RegisterHandler(handlers...)

	return cp
}
