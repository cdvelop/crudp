package modules

import (
	"github.com/cdvelop/crudp"
	"github.com/cdvelop/crudp/example/modules/patientCare"
	"github.com/cdvelop/crudp/example/modules/userRegister"
)

// Shared instance between client and server
var Protocol = Setup(
	&userRegister.User{},
	&patientCare.Patient{},
)

// Setup initializes CRUDP with the real implementations
func Setup(handlers ...any) *crudp.CrudP {
	cp := crudp.New()
	if err := cp.LoadHandlers(handlers...); err != nil {
		panic(err)
	}
	return cp
}
