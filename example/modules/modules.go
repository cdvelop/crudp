package modules

import (
	"github.com/cdvelop/crudp/example/modules/patient"
	"github.com/cdvelop/crudp/example/modules/user"
)

// Init returns all business modules
func Init() []any {
	return []any{
		&user.Handler{},
		&patient.Handler{},
	}
}
