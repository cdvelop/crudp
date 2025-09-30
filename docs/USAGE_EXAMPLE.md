# Usage Examples

Registration is declared **only once** and shared between client (TinyGo/WASM) and server.

```
app/
	modules/
		modules.go
		patientCare/
			handlers.go
		userRegister/
			handlers.go
	web/
		main.server.go
		main.wasm.go
```

### modules/modules.go — centralized registration (shared)

```go
package modules

import (
	"github.com/cdvelop/crudp"
	"github.com/your/app/modules/userRegister"
	"github.com/your/app/modules/patientCare"
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
```

### modules/userRegister/userRegister.go — implementations per module

```go
package userRegister

type User struct {
	ID    int
	Name  string
	Email string
}

func (u *User) Create(data ...any) (any, error) {
	created := make([]*User, 0, len(data))
	for _, item := range data {
		user := item.(*User)
		user.ID = 123
		created = append(created, user)
	}
	return created, nil
}

func (u *User) Read(data ...any) (any, error) {
	results := make([]*User, 0, len(data))
	for _, item := range data {
		user := item.(*User)
		results = append(results, &User{ID: user.ID, Name: "Found " + user.Name, Email: user.Email})
	}
	return results, nil
}
```

### modules/patientCare/patientCare.go — another module

```go
package patientCare

type Patient struct {
	ID   int
	Name string
	Age  int
}

func (p *Patient) Create(data ...any) (any, error) {
	// Specific implementation for patients
	return nil, nil
}

func (p *Patient) Read(data ...any) (any, error) {
	// Specific implementation for patients
	return nil, nil
}
```

### web/main.server.go — standard server

```go
//go:build !wasm

package main

import (
	"io"
	"net/http"

	"github.com/your/app/modules"
)

func main() {
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		payload, _ := io.ReadAll(r.Body)
		response, err := modules.Protocol.ProcessPacket(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(response)
	})

	http.ListenAndServe(":8080", nil)
}
```

### web/main.wasm.go — TinyGo/WebAssembly client with fetchgo

```go
//go:build wasm

package main

import (
	"github.com/cdvelop/crudp"
	"github.com/cdvelop/fetchgo"
	"github.com/your/app/modules"
	"github.com/your/app/modules/userRegister"
	"github.com/your/app/modules/patientCare"
)

// HTTP client using fetchgo for cross-platform compatibility
var httpClient = &fetchgo.Client{
	BaseURL: "http://localhost:8080",
	RequestType: fetchgo.RequestRaw, // Send raw bytes for crudp packets
}

func sendCreateUser(user *userRegister.User) error {
	// Encode the packet using crudp
	packet, err := crudp.EncodePacket('c', 0, "", user)
	if err != nil {
		return err
	}

	// Send using fetchgo
	httpClient.SendRequest("POST", "/api", packet, func(result any, err error) {
		if err != nil {
			// handle error
			return
		}

		if responseData, ok := result.([]byte); ok {
			// Process the crudp response packet
			// responseData contains the server response
		}
	})

	return nil
}

func readUsers(id int) error {
	// Encode the packet using crudp
	packet, err := crudp.EncodePacket('r', 0, "", &userRegister.User{ID: id})
	if err != nil {
		return err
	}

	// Send using fetchgo
	httpClient.SendRequest("POST", "/api", packet, func(result any, err error) {
		if err != nil {
			// handle error
			return
		}

		if responseData, ok := result.([]byte); ok {
			// Process the crudp response packet
			// responseData contains the server response
		}
	})

	return nil
}

func sendCreatePatient(patient *patientCare.Patient) error {
	// Encode the packet using crudp
	packet, err := crudp.EncodePacket('c', 1, "", patient)
	if err != nil {
		return err
	}

	// Send using fetchgo
	httpClient.SendRequest("POST", "/api", packet, func(result any, err error) {
		if err != nil {
			// handle error
			return
		}

		if responseData, ok := result.([]byte); ok {
			// Process the crudp response packet
			// responseData contains the server response
		}
	})

	return nil
}

// Alternative: Synchronous wrapper for easier usage
func sendCreateUserSync(user *userRegister.User) ([]byte, error) {
	packet, err := crudp.EncodePacket('c', 0, "", user)
	if err != nil {
		return nil, err
	}

	var result []byte
	var requestErr error

	done := make(chan struct{})

	httpClient.SendRequest("POST", "/api", packet, func(response any, err error) {
		if err != nil {
			requestErr = err
		} else if data, ok := response.([]byte); ok {
			result = data
		}
		done <- struct{}{}
	})

	<-done
	return result, requestErr
}
```

Both binaries share the **same instance** `modules.Protocol`. Indexes are assigned automatically by order: `&userRegister.User{}` = 0, `&patientCare.Patient{}` = 1, eliminating the need for manual constants.

**Benefits of using fetchgo:**
- **Cross-platform**: Same code works in WASM (browser) and standard Go environments
- **Unified API**: Consistent HTTP handling across different runtimes
- **Async by default**: Non-blocking requests with callback-based responses
- **Configurable**: Support for headers, timeouts, custom encoders, and more