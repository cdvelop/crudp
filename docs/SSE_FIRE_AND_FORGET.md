## Código patrón **"Fire and Forget"** (Dispara y Olvida) 
para el envío, y **"Reactive Listener"** para la respuesta. Utiliza `syscall/js` para aprovechar la API nativa del navegador (`EventSource` y `fetch`), lo cual es mucho más ligero que la librería estándar de Go completa.

### Estructura del Cliente

El cliente tiene tres responsabilidades:

1.  **Acumular peticiones** en una cola (Batch).
2.  **Sincronizar (POST)** enviando el lote binario.
3.  **Escuchar (SSE)**, decodificar la respuesta y ejecutar el *callback* correcto.

### 1. Definición del Cliente (`client.go`)

```go
package client

import (
	"syscall/js"
	"github.com/cdvelop/unixid"
	"github.com/cdvelop/tinybin"
)

// ResponseCallback define qué hacer cuando el servidor responda
type ResponseCallback func(success bool, message string, data []byte)

type Client struct {
	protocol    *crudp.CrudP
	apiEndpoint string
	sseEndpoint string
	
	// Cola de paquetes pendientes de enviar (Offline/Batching)
	queue []crudp.Packet
	
	// Mapa de "Promesas": ReqID -> Callback
	// Cuando llega el SSE, buscamos aquí a quién avisar
	pending map[string]ResponseCallback
	mu      sync.Mutex
	
	// Para generar IDs únicos
	idHandler *unixid.UnixID
}

func New(protocol *crudp.CrudP, baseURL string) *Client {
	// Para WASM, necesitas proporcionar un session handler
	// Ejemplo: idHandler, _ := unixid.NewUnixID(&sessionHandler{})
	// Donde sessionHandler implementa userSessionNumber() string
	idHandler, _ := unixid.NewUnixID() // Asume server-side o session configurado
	
	return &Client{
		protocol:    protocol,
		apiEndpoint: baseURL + "/sync",
		sseEndpoint: baseURL + "/events",
		pending:     make(map[string]ResponseCallback),
		queue:       make([]crudp.Packet, 0),
		idHandler:   idHandler,
	}
}
```

### 2. Generación y Encolado (`AddToBatch`)

Aquí es donde el componente UI llama para guardar datos. No envía nada por red todavía, solo encola.

```go
// Enqueue agrega una operación al lote actual
func (c *Client) Enqueue(handlerID uint8, action byte, callback ResponseCallback, data ...any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Generar ID de Correlación (ReqID)
	reqID := c.idHandler.GetNewID()

	// 2. Codificar los datos usando TinyBin (igual que el servidor)
	encodedData := make([][]byte, 0, len(data))
	for _, item := range data {
		b, err := tinybin.Encode(item)
		if err != nil {
			return err
		}
		encodedData = append(encodedData, b)
	}

	// 3. Crear el Paquete
	packet := crudp.Packet{
		ReqID:     reqID,
		HandlerID: handlerID,
		Action:    action,
		Data:      encodedData,
	}

	// 4. Guardar en la cola de envío
	c.queue = append(c.queue, packet)

	// 5. Registrar el callback para cuando vuelva la respuesta (futuro)
	if callback != nil {
		c.pending[reqID] = callback
	}

	return nil
}
```





### 5. Ejemplo de Uso (Cómo se ve en tu código de UI)

Así es como integrarías esto en tu aplicación TinyGo (ej. usando un framework UI o DOM directo).

- web/client.go (TinyGo)
```go
//go:build wasm
package main

import (
    "syscall/js"
    "github.com/cdvelop/crudp"
    "tu-proyecto/client"
)

// Estructura de datos compartida
type User struct {
    Name string
    Age  int
}

func main() {
    // 1. Inicializar protocolo y cliente
    proto := crudp.New()
    // proto.RegisterHandler(...) // En el cliente solo necesitamos los índices, o ni eso si usamos constantes
    
    apiClient := client.New(proto, "http://localhost:8080")

    // 2. Iniciar escucha en fondo
    go apiClient.ListenSSE()

    // 3. Simular Interacción de Usuario (Botón "Guardar")
    js.Global().Get("console").Call("log", "Usuario hace click en Guardar...")

    newUser := User{Name: "Juan", Age: 30}
    
    // Definimos qué hacer cuando el servidor confirme
    onSaved := func(success bool, msg string, data []byte) {
        if success {
            js.Global().Get("console").Call("log", "✅ Servidor confirmó:", msg, "ID guardado.")
            // Aquí podrías decodificar 'data' si el servidor devolvió el ID nuevo
        } else {
            js.Global().Get("console").Call("log", "❌ Error guardando:", msg)
        }
    }

    // 4. Encolar petición (HandlerID 0 = Users, Action 'c' = Create)
    err := apiClient.Enqueue(0, 'c', onSaved, newUser)
    if err != nil {
        panic(err)
    }

    // 5. Simular envío (esto podría ser automático cada 5s o al detectar red)
    js.Global().Get("console").Call("log", "Enviando lote...")
    apiClient.Sync()

    // Mantener vivo para el ejemplo
    select {}
}
```

### Puntos Clave de esta Implementación

1.  **Eficiencia de Memoria:** `map[string]ResponseCallback` es muy ligero. Solo guardamos punteros a funciones.
2.  **TinyGo Friendly:** Al usar `syscall/js` para SSE y `tinybin` para datos, el binario final (`.wasm`) se mantiene extremadamente pequeño.
3.  **Desacoplamiento Temporal:** `Enqueue` es instantáneo (la UI no se congela). `Sync` puede ocurrir 1 milisegundo después o 1 hora después (si estaba offline).
4.  **Manejo de Archivos:** Si usas la estrategia de archivos dentro del paquete (bytes), pasas el `[]byte` del archivo en `data...` dentro de `Enqueue`, y funciona transparente.