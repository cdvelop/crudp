# CRUDP - Binary CRUD Protocol

Simple binary protocol for Go structs with automatic type detection.


```go
package crudp

import (
	"github.com/cdvelop/tinybin"
	"github.com/cdvelop/tinyreflect"
)

// Interfaces CRUD separadas - los handlers pueden implementar solo las que necesiten
type Creator interface {
	Create(data ...any) ([]byte, error)
}

type Reader interface {
	Read(data ...any) ([]byte, error)
}

type Updater interface {
	Update(data ...any) ([]byte, error)
}

type Deleter interface {
	Delete(data ...any) ([]byte, error)
}

// CrudP maneja el registro y procesamiento automático de handlers
type CrudP struct {
	handlers map[uint32]map[byte]func(...any) ([]byte, error) // [StructID][Action] -> Handler
}

// New crea una nueva instancia de CrudP
func New() *CrudP {
	return &CrudP{
		handlers: make(map[uint32]map[byte]func(...any) ([]byte, error)),
	}
}

// Packet representa tanto solicitudes como respuestas del protocolo
type Packet struct {
	Action   byte      // acción: 'c' (create), 'r' (read), 'u' (update), 'd' (delete), 'e' (error)
	StructID uint32    // identificador único de la estructura (obtenido automáticamente de tinyreflect)
	Message  string    // información adicional (opcional en requests, usado en responses)
	Data     [][]byte  // slice de datos codificados, cada []byte es una estructura
}

// EncodePacket codifica un paquete con detección automática de tipo
func EncodePacket(action byte, message string, data ...any) ([]byte, error) {
	var structID uint32
	var encodedData [][]byte
	
	if len(data) > 0 && data[0] != nil {
		typ := tinyreflect.TypeOf(data[0])
		structID = typ.StructID()
		
		// Codificar cada estructura individualmente
		for _, item := range data {
			itemBytes, err := tinybin.Encode(item)
			if err != nil {
				return nil, err
			}
			encodedData = append(encodedData, itemBytes)
		}
	}
	
	packet := Packet{
		Action:   action,
		StructID: structID,
		Message:  message,
		Data:     encodedData,
	}
	
	return tinybin.Encode(packet)
}

// DecodePacket decodifica un paquete
func DecodePacket(data []byte, packet *Packet) error {
	return tinybin.Decode(data, packet)
}

// RegisterHandlers registra automáticamente los handlers para las estructuras dadas
func (cp *CrudP) RegisterHandlers(structType any, handler any) error {
	typ := tinyreflect.TypeOf(structType)
	structID := typ.StructID()
	
	if cp.handlers[structID] == nil {
		cp.handlers[structID] = make(map[byte]func(...any) ([]byte, error))
	}
	
	// Detectar y registrar métodos CRUD automáticamente del handler
	if creator, ok := handler.(Creator); ok {
		cp.handlers[structID]['c'] = creator.Create
	}
	if reader, ok := handler.(Reader); ok {
		cp.handlers[structID]['r'] = reader.Read
	}
	if updater, ok := handler.(Updater); ok {
		cp.handlers[structID]['u'] = updater.Update
	}
	if deleter, ok := handler.(Deleter); ok {
		cp.handlers[structID]['d'] = deleter.Delete
	}
	return nil
}

// ProcessPacket procesa automáticamente un packet y llama al handler correspondiente
func (cp *CrudP) ProcessPacket(requestBytes []byte) ([]byte, error) {
	var packet Packet
	if err := DecodePacket(requestBytes, &packet); err != nil {
		return cp.createErrorResponse("decode_error", err)
	}
	
	// Buscar handler por StructID y Action
	structHandlers, exists := cp.handlers[packet.StructID]
	if !exists {
		return cp.createErrorResponse("no_handler", fmt.Errorf("no handler for StructID: %d", packet.StructID))
	}
	
	handler, exists := structHandlers[packet.Action]
	if !exists {
		return cp.createErrorResponse("no_action", fmt.Errorf("action '%c' not supported for StructID: %d", packet.Action, packet.StructID))
	}
	
	// Decodificar datos para el handler
	var decodedData []any
	for _, itemBytes := range packet.Data {
		var item any
		if err := tinybin.Decode(itemBytes, &item); err != nil {
			return cp.createErrorResponse("data_decode_error", err)
		}
		decodedData = append(decodedData, item)
	}
	
	// Llamar al handler
	responseData, err := handler(decodedData...)
	if err != nil {
		return cp.createErrorResponse("handler_error", err)
	}
	
	// Crear respuesta exitosa
	responsePacket := Packet{
		Action:   packet.Action, // Misma acción que la request
		StructID: packet.StructID,
		Message:  "success",
		Data:     [][]byte{responseData}, // Response del handler
	}
	
	return tinybin.Encode(responsePacket)
}

// createErrorResponse crea una respuesta de error eficiente
func (cp *CrudP) createErrorResponse(message string, err error) ([]byte, error) {
	errorPacket := Packet{
		Action:   'e',
		StructID: 0,
		Message:  fmt.Sprintf("%s: %v", message, err),
		Data:     nil,
	}
	return tinybin.Encode(errorPacket)
}

// DecodeData decodifica los datos del paquete
func DecodeData(packet *Packet, index int, target any) error {
	if index >= len(packet.Data) {
		return errors.New("index out of range")
	}
	return tinybin.Decode(packet.Data[index], target)
}
```

## Ejemplo de Uso

```go
package main

import (
	"io"
	"net/http"
	
	"github.com/cdvelop/crudp"
)

type User struct {
	ID    int    
	Name  string 
	Email string 
}

// Implementa StructNamer para tinyreflect
func (User) StructName() string {
	return "user"
}

type Product struct {
	ID    int     
	Name  string  
	Price float64
}

// Implementa StructNamer para tinyreflect
func (Product) StructName() string {
	return "product"
}

// UserHandler implementa las operaciones CRUD que necesita
type UserHandler struct{}

func (uh *UserHandler) Create(data ...any) ([]byte, error) {
	var created []User
	for _, item := range data {
		user := item.(User) // Casting manual como esperabas
		// Lógica de creación: insertar en BD, validar, etc.
		user.ID = 123 // Ejemplo: asignar ID generado
		created = append(created, user)
	}
	return tinybin.Encode(created)
}

func (uh *UserHandler) Read(data ...any) ([]byte, error) {
	var results []User
	for _, item := range data {
		user := item.(User)
		// Lógica de búsqueda: SELECT * FROM users WHERE id = user.ID
		foundUser := User{ID: user.ID, Name: "Found: " + user.Name, Email: user.Email}
		results = append(results, foundUser)
	}
	return tinybin.Encode(results)
}

func main() {
	// 1. Setup del servidor (una vez al iniciar)
	cp := crudp.New()
	userHandler := &UserHandler{}
	
	// Registro automático: CRUDP detecta que UserHandler implementa Creator y Reader
	cp.RegisterHandlers(User{}, userHandler) // Asocia User struct con UserHandler
	
	// 2. Handler HTTP (esto se ejecuta por cada request)
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		// Leer request body (bytes del Packet)
		requestBytes, _ := io.ReadAll(r.Body)
		
		// CRUDP procesa automáticamente:
		// - Decodifica Packet
		// - Busca handler por StructID 
		// - Decodifica Data
		// - Llama al método correspondiente (Create/Read/Update/Delete)
		// - Codifica respuesta
		responseBytes, err := cp.ProcessPacket(requestBytes)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		
		// Enviar respuesta
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(responseBytes)
	})
	
	// 3. Cliente envía request
	// POST /api con Packet codificado:
	// Action: 'c' (create)  
	// StructID: [ID automático de User]
	// Data: [User{Name: "Juan", Email: "juan@test.com"}]
	//
	// CRUDP automáticamente:
	// 1. Decodifica el Packet
	// 2. Ve StructID de User y Action 'c'  
	// 3. Busca UserHandler.Create
	// 4. Decodifica Data como []User
	// 5. Llama userHandler.Create(user)
	// 6. Codifica respuesta y la retorna
}
	
	// Respuesta con mensaje - reutilizar los mismos datos
	responseBytes, err := crudp.EncodePacket('r', "Usuarios encontrados",
		User{ID: 1, Name: "Juan", Email: "juan@test.com"},
		User{ID: 2, Name: "Ana", Email: "ana@test.com"},
	)
	if err != nil {
		panic(err)
	}
	
	var response crudp.Packet
	if err := crudp.DecodePacket(responseBytes, &response); err != nil {
		panic(err)
	}
	
	// response.Action = 'r' (misma acción)
	// response.Message = "success"
	// response.Data contiene el resultado del handler
}
```

## Características Principales

CRUDP sigue la filosofía minimalista con:

- 🏆 **Binarios ultra-pequeños** - Cero dependencias extras
- ✅ **Compatibilidad TinyGo** - Sin problemas de compilación  
- 🎯 **Rendimiento predecible** - Sin asignaciones ocultas
- 🔧 **API mínima** - Solo operaciones esenciales
- 🔍 **Identificación única** - StructID garantiza identificación sin colisiones
- 💪 **Tipado fuerte** - Estructuras Go directas, no maps
- ⚡ **Eficiencia** - uint32 vs string, menor uso de memoria

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

- **🎯 Registro automático** - `RegisterHandlers` detecta interfaces implementadas
- **🔧 Interfaces flexibles** - Implementa solo Create, Read, Update o Delete según necesites
- **⚡ Procesamiento eficiente** - `ProcessPacket` maneja todo automáticamente
- **🛡️ Manejo de errores robusto** - Errores se convierten automáticamente en responses
- **🏆 Testeable** - Constructor `New()` permite testing aislado
- **🔄 Caching HTTP** - Instancia CrudP se puede cachear en handlers HTTP
- **💪 Cero allocaciones innecesarias** - Handlers generan responses directamente

### ⚠️ Consideraciones

- **Casting manual requerido** - Handlers deben hacer `item.(Type)` - Garantiza type safety
- **Registro explícito necesario** - Debes llamar `RegisterHandlers` - Control total sobre qué se registra  
- **Un handler por StructID** - Ultima registración sobrescribe - Diseño simple y predecible

## Sistema de Handlers Automáticos

### ✅ Ventajas del Diseño

- **🎯 Registro automático** - `RegisterHandlers` detecta interfaces implementadas
- **🔧 Interfaces flexibles** - Implementa solo Create, Read, Update o Delete según necesites
- **⚡ Procesamiento eficiente** - `ProcessPacket` maneja todo automáticamente
- **🛡️ Manejo de errores robusto** - Errores se convierten automáticamente en responses
- **🏆 Testeable** - Constructor `New()` permite testing aislado
- **🔄 Caching HTTP** - Instancia CrudP se puede cachear en handlers HTTP
- **💪 Cero allocaciones innecesarias** - Handlers generan responses directamente

### ⚠️ Consideraciones

- **Casting manual requerido** - Handlers deben hacer `item.(Type)` - Garantiza type safety
- **Registro explícito necesario** - Debes llamar `RegisterHandlers` - Control total sobre qué se registra  
- **Un handler por StructID** - Ultima registración sobrescribe - Diseño simple y predecible

## Por qué StructID en lugar de nombres

**StructID ofrece identificación superior:**

- **✅ Sin colisiones** - Hash único del runtime de Go garantiza identificación
- **✅ Automático** - No requiere implementar StructNamer ni interfaces manuales  
- **✅ Eficiente** - uint32 (4 bytes) vs strings (N bytes)
- **✅ Consistente** - Misma estructura = mismo ID independiente de inicialización
- **✅ Compatible TinyGo** - Usa información del runtime existente
- **❌ Nombres pueden** - Colisionar entre paquetes, tener typos, requerir interfaces

### Arquitectura

- **`tinybin`**: Codificación/decodificación binaria
- **`tinyreflect`**: Detección de tipos únicos con StructID
- **`crudp`**: Lógica del protocolo CRUD