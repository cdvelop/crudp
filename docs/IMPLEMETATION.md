# CRUDP - Binary CRUD Protocol

Simple binary protocol for Go structs with deterministic, shared handler registration.


```go
package crudp

import (
	"github.com/cdvelop/tinybin"
	. "github.com/cdvelop/tinystring"
)

// Interfaces CRUD separadas - los handlers pueden implementar solo las que necesiten
type Creator interface {
	Create(data ...any) (any, error)
}

type Reader interface {
	Read(data ...any) (any, error)
}

type Updater interface {
	Update(data ...any) (any, error)
}

type Deleter interface {
	Delete(data ...any) (any, error)
}

// Constante para máximo número de handlers (optimizado para WebAssembly)
const maxHandlers = 32

// ActionHandlers agrupa las funciones CRUD para un índice de registro
type ActionHandlers struct {
	Create func(...any) (any, error)
	Read   func(...any) (any, error)
	Update func(...any) (any, error)
	Delete func(...any) (any, error)
}

// CrudP maneja el procesamiento automático de handlers
// Usa arrays fijos en lugar de maps para compatibilidad con TinyGo
type CrudP struct {
	handlers [maxHandlers]ActionHandlers // Handlers cargados por índice
	count    uint8                      // Número de handlers registrados
}

// New crea una nueva instancia de CrudP
func New() *CrudP {
	return &CrudP{}
}

// Packet representa tanto solicitudes como respuestas del protocolo
type Packet struct {
	Action    byte     // acción: 'c', 'r', 'u', 'd', 'e'
	HandlerID uint8    // índice compartido dentro del slice de registro
	Message   string   // información adicional (opcional en requests, usado en responses)
	Data      [][]byte // slice de datos codificados, cada []byte es una estructura
}

// EncodePacket codifica un paquete para un handler ya conocido
func EncodePacket(action byte, handlerID uint8, message string, data ...any) ([]byte, error) {
	encoded := make([][]byte, 0, len(data))
	for _, item := range data {
		bytes, err := tinybin.Encode(item)
		if err != nil {
			return nil, err
		}
		encoded = append(encoded, bytes)
	}

	packet := Packet{
		Action:    action,
		HandlerID: handlerID,
		Message:   message,
		Data:      encoded,
	}

	return tinybin.Encode(packet)
}

// DecodePacket decodifica un paquete
func DecodePacket(data []byte, packet *Packet) error {
	return tinybin.Decode(data, packet)
}

// Load vincula los handlers compartidos con índices deterministas
// Espera pares prototype, handler dentro del slice recibido.
func (cp *CrudP) Load(registrations []any) error {
	if len(registrations)%2 != 0 {
		return Errf("registrations must be provided as pairs: prototype, handler")
	}

	count := len(registrations) / 2
	if count > maxHandlers {
		return Errf("maximum handler registrations exceeded: %d", maxHandlers)
	}

	for pair := 0; pair < count; pair++ {
		handler := registrations[pair*2+1]
		if handler == nil {
			return Errf("registration %d has no handler", pair)
		}
		cp.bind(uint8(pair), handler)
	}

	cp.count = uint8(count)
	return nil
}

// bind copia las funciones CRUD sin asignaciones dinámicas
func (cp *CrudP) bind(index uint8, handler any) {
	if creator, ok := handler.(Creator); ok {
		cp.handlers[index].Create = creator.Create
	}
	if reader, ok := handler.(Reader); ok {
		cp.handlers[index].Read = reader.Read
	}
	if updater, ok := handler.(Updater); ok {
		cp.handlers[index].Update = updater.Update
	}
	if deleter, ok := handler.(Deleter); ok {
		cp.handlers[index].Delete = deleter.Delete
	}
}

// ProcessPacket procesa automáticamente un packet y llama al handler correspondiente
func (cp *CrudP) ProcessPacket(requestBytes []byte) ([]byte, error) {
	var packet Packet
	if err := DecodePacket(requestBytes, &packet); err != nil {
		return cp.createErrorResponse("decode_error", err)
	}

	var decodedData []any
	for _, itemBytes := range packet.Data {
		var item any
		if err := tinybin.Decode(itemBytes, &item); err != nil {
			return cp.createErrorResponse("data_decode_error", err)
		}
		decodedData = append(decodedData, item)
	}

	result, err := cp.callHandler(packet.HandlerID, packet.Action, decodedData...)
	if err != nil {
		return cp.createErrorResponse("handler_error", err)
	}

	var responseData []byte
	if bytes, ok := result.([]byte); ok {
		responseData = bytes
	} else {
		responseData, err = tinybin.Encode(result)
		if err != nil {
			return cp.createErrorResponse("encode_error", err)
		}
	}

	responsePacket := Packet{
		Action:    packet.Action,
		HandlerID: packet.HandlerID,
		Message:   "success",
		Data:      [][]byte{responseData},
	}

	return tinybin.Encode(responsePacket)
}

// callHandler busca y llama directamente al handler por índice compartido
func (cp *CrudP) callHandler(handlerID uint8, action byte, data ...any) (any, error) {
	if handlerID >= cp.count {
		return nil, Errf("no handler found for id: %d", handlerID)
	}

	handler := cp.handlers[handlerID]

	switch action {
	case 'c':
		if handler.Create != nil {
			return handler.Create(data...)
		}
	case 'r':
		if handler.Read != nil {
			return handler.Read(data...)
		}
	case 'u':
		if handler.Update != nil {
			return handler.Update(data...)
		}
	case 'd':
		if handler.Delete != nil {
			return handler.Delete(data...)
		}
	}

	return nil, Errf("action '%c' not implemented for handler id: %d", action, handlerID)
}

// createErrorResponse crea una respuesta de error eficiente
func (cp *CrudP) createErrorResponse(message string, err error) ([]byte, error) {
	errorMsg := Errf("%s: %v", message, err).Error()
	packet := Packet{
		Action:    'e',
		HandlerID: 0,
		Message:   errorMsg,
		Data:      nil,
	}
	return tinybin.Encode(packet)
}

// DecodeData decodifica los datos del paquete
func DecodeData(packet *Packet, index int, target any) error {
	if index >= len(packet.Data) {
		return Errf("index out of range")
	}
	return tinybin.Decode(packet.Data[index], target)
}
```

## Ejemplo de Uso

La registración se declara **una sola vez** y se comparte entre cliente (TinyGo/WASM) y servidor.

```
app/
	register.go
	config.go
	main.server.go
	main.wasm.go
```

### register.go — registro centralizado

```go
package app

import "github.com/cdvelop/crudp"

type User struct {
	ID    int
	Name  string
	Email string
}

type Product struct {
	ID    int
	Name  string
	Price float64
}

type UserHandler struct{}

func (UserHandler) Create(data ...any) (any, error) {
	created := make([]User, 0, len(data))
	for _, item := range data {
		user := item.(User)
		user.ID = 123
		created = append(created, user)
	}
	return created, nil
}

func (UserHandler) Read(data ...any) (any, error) {
	results := make([]User, 0, len(data))
	for _, item := range data {
		user := item.(User)
		results = append(results, User{ID: user.ID, Name: "Found " + user.Name, Email: user.Email})
	}
	return results, nil
}

type ProductHandler struct{}

// ...implementaciones opcionales de CRUD...

// Pares: prototipo cero-valor seguido del handler correspondiente
var HandlersRegistration = []any{
	User{}, &UserHandler{},
	Product{}, &ProductHandler{},
}

const (
	HandlerUser uint8 = iota
	HandlerProduct
)
```

Los identificadores `HandlerUser` y `HandlerProduct` se derivan del orden en el slice. Si prefieres minimizar riesgos humanos, puedes generar este bloque con `//go:generate`.

La tabla `HandlersRegistration` alterna **prototipo cero-valor** seguido de **handler**. `Load()` recorre el slice en pasos de dos entradas y asigna los índices compartidos (`uint8`) de izquierda a derecha.

### config.go — inicialización compartida

```go
package app

import "github.com/cdvelop/crudp"

var Protocol = crudp.New()

func init() {
	if err := Protocol.Load(HandlersRegistration); err != nil {
		panic(err)
	}
}
```

### main.server.go — servidor estándar

```go
package main

import (
	"io"
	"net/http"

	"github.com/cdvelop/crudp"
	"github.com/your/app"
)

func main() {
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		payload, _ := io.ReadAll(r.Body)
		response, err := app.Protocol.ProcessPacket(payload)
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

### main.wasm.go — cliente TinyGo/WebAssembly

```go
package main

import (
	"github.com/cdvelop/crudp"
	"github.com/your/app"
)

func sendCreate(user app.User) ([]byte, error) {
	return crudp.EncodePacket('c', app.HandlerUser, "", user)
}

func readUsers(id int) ([]byte, error) {
	return crudp.EncodePacket('r', app.HandlerUser, "", app.User{ID: id})
}
```

Ambos binarios comparten el **mismo slice** `HandlersRegistration`, por lo que el índice `HandlerUser` siempre significa lo mismo, sin necesidad de `StructID` ni `StructName`.

## Características Principales

CRUDP sigue la filosofía minimalista con:

- 🏆 **Binarios ultra-pequeños** - Cero dependencias extras
- ✅ **Compatibilidad TinyGo** - Sin problemas de compilación  
- 🎯 **Rendimiento predecible** - Sin asignaciones ocultas
- 🔧 **API mínima** - Solo operaciones esenciales
- 🔍 **Identificación determinista** - Índices compartidos garantizan el mismo handler en cliente y servidor
- 💪 **Tipado fuerte** - Estructuras Go directas, no maps
- ⚡ **Eficiencia** - IDs compactos (`uint8`) y tabla fija sin maps dinámicos

### Tipos de Datos Compatibles (Enfoque Minimalista)

CRUDP **intencionalmente** soporta solo un conjunto mínimo de tipos para mantener el tamaño del binario pequeño:

**✅ Tipos Soportados:**
- **Tipos básicos**: `string`, `bool`
- **Todos los tipos numéricos**: `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`
- **Todos los slices básicos**: `[]string`, `[]bool`, `[]byte`, `[]int`, `[]int8`, `[]int16`, `[]int32`, `[]int64`, `[]uint`, `[]uint8`, `[]uint16`, `[]uint32`, `[]uint64`, `[]float32`, `[]float64`
- **Structs**: Solo con tipos de campos soportados
- **Slices de structs**: `[]struct{...}` donde todos los campos sean tipos soportados
- **Maps**: `map[K]V` donde K y V sean solo tipos soportados
- **Slices de maps**: `[]map[K]V` donde K y V sean tipos soportados
- **Punteros**: Solo a los tipos soportados arriba

**❌ Tipos No Soportados:**
- `any`, `chan`, `func`
- `complex64`, `complex128`
- `uintptr`, `unsafe.Pointer` (usados solo internamente)
- Arrays (diferentes a slices)
- Tipos complejos anidados más allá del alcance soportado

Este enfoque enfocado asegura un tamaño de código mínimo mientras cubre las operaciones de transferencia de datos más comunes incluyendo structs simples.

## Sistema de Handlers Automáticos

### ✅ Ventajas del Diseño

- **🎯 Registro centralizado** - `Load()` copia las interfaces desde un único slice compartido (cliente/servidor)
- **🔧 Interfaces flexibles** - Implementa solo Create, Read, Update o Delete según necesites
- **⚡ Procesamiento eficiente** - `ProcessPacket` maneja todo automáticamente
- **🛡️ Manejo de errores robusto** - Errores se convierten automáticamente en responses
- **🏆 Testeable** - Constructor `New()` permite testing aislado
- **🔄 Caching HTTP** - Instancia CrudP se puede cachear en handlers HTTP
- **💪 Cero allocaciones innecesarias** - Handlers generan responses directamente

### ⚠️ Consideraciones

- **Slice compartido obligatorio** - Cliente y servidor deben importar la misma tabla de registro para que los índices coincidan
- **Casting manual requerido** - Handlers deben hacer `item.(Type)`; la seguridad de tipos queda bajo tu control
- **IDs deterministas** - Una nueva versión del registro debe mantener el orden u ofrecer constantes generadas automáticamente

### 🎯 Handlers Desacoplados (Retornan `any`)

- **🔧 Sin dependencia de tinybin** - Handlers no necesitan importar ni conocer tinybin
- **⚡ Menos trabajo** - Solo retornan estructuras Go, CRUDP codifica automáticamente
- **🧪 Testing fácil** - Handlers se testean independientemente sin CRUDP
- **📦 API natural** - `return users, nil` en lugar de `return tinybin.Encode(users)`
- **🔄 Flexibilidad** - Si necesita control especial, puede retornar `[]byte` directamente

### ⚡ Optimización TinyGo/WebAssembly

- **🏗️ Arrays fijos** - Sin maps, usa `[32]ActionHandlers` para cero asignaciones
- **🎯 Llamadas directas** - `callHandler()` evita asignaciones de variables de función
- **💾 Memoria predecible** - Tamaño fijo conocido en tiempo de compilación
- **🔍 Búsqueda O(n)** - Eficiente para 5-15 tipos típicos en WebAssembly
- **✅ Compatible TinyGo** - Sin características problemáticas de maps dinámicos

## Por qué índices compartidos en lugar de `StructID`

Abandonar `StructID` simplifica la arquitectura cuando controlas el registro en un único lugar.

### Ventajas

- **Simetría total** – El mismo slice de registro se compila en el binario WASM y en el servidor nativo.
- **Constantes explícitas** – Los valores `uint8` son conocidos en tiempo de compilación; puedes exportarlos o generarlos.
- **Sin reflection** – No se requiere `tinyreflect`; compatible con TinyGo y builds restringidos.
- **Detección rápida de errores** – Cualquier des-sincronización se captura en tests que comparen la tabla compartida.

### Desventajas y mitigaciones

- **Mantenimiento del orden** – Cambiar el orden del slice cambia los IDs. Mantén el registro en un paquete único y versionado.
- **Generación de IDs** – Considera `//go:generate` para producir las constantes a partir del slice y evitar errores humanos.
- **Migraciones coordinadas** – Cliente y servidor deben actualizarse juntos cuando se agregan entradas.

### Arquitectura

- **`tinybin`**: Codificación/decodificación binaria compacta
- **`crudp`**: Lógica del protocolo CRUD, tablas fijas y manejo de errores
- **`app/register.go`**: Fuente única de verdad para `HandlersRegistration` y los IDs exportados