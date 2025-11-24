# SSE (Server-Sent Events) Implementation

This document consolidates all Server-Sent Events (SSE) related functionality in CRUDP, including the efficient broker, client synchronization, fire-and-forget patterns, reactive listeners, and file integration.

## 1. SSE Handler Eficiente (Server-Sent Events)

Para manejar esto en producción con Go, necesitamos un **Broker** concurrente que gestione las conexiones sin bloquearse y limpie recursos (evitando goroutines zombies).

**Diseño del Broker:**

1.  **Non-blocking send:** Si el cliente está lento, no bloqueamos al resto.
2.  **Keep-Alive:** Enviamos un "ping" periódico para mantener la conexión NAT abierta.
3.  **Context Aware:** Se desconecta automáticamente si el cliente cierra el navegador.

#### Archivo: `pkg/sse/broker.go`

```go
package sse

import (
	"fmt"
	"net/http"
	"sync"
	"time"
    "github.com/cdvelop/tinybin"
    // importa tu paquete crudp donde definiste BatchResponse
)

type Broker struct {
	// Mapa de canales por UserID.
    // Usamos un map de maps para permitir múltiples pestañas por usuario si fuera necesario,
    // o simplificamos a 1 conexión por usuario.
	clients map[string]chan []byte 
	mutex   sync.RWMutex
}

func NewBroker() *Broker {
	b := &Broker{
		clients: make(map[string]chan []byte),
	}
    // Iniciar rutina de limpieza o keep-alive global si se desea
	return b
}

// ServeHTTP maneja la conexión persistente (GET /events)
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Identificar usuario
	userID := r.Header.Get("X-User-ID") // O desde tu contexto/session
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Configurar Headers SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*") // Ajustar para prod

	// 3. Registrar Cliente
	messageChan := make(chan []byte, 10) // Buffer de 10 mensajes para evitar bloqueos
	b.mutex.Lock()
	b.clients[userID] = messageChan // OJO: Esto reemplaza la conexión anterior de ese usuario
	b.mutex.Unlock()

	// 4. Notificar conexión exitosa (opcional)
	fmt.Fprintf(w, "event: connected\ndata: %s\n\n", userID)
	w.(http.Flusher).Flush()

	// 5. Loop Principal (Bloquea hasta desconexión)
	defer func() {
		b.mutex.Lock()
		delete(b.clients, userID)
		close(messageChan)
		b.mutex.Unlock()
	}()

    // Timer para Heartbeat (ping) cada 30s para evitar timeouts de proxies
    heartbeat := time.NewTicker(30 * time.Second)
    defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
            // Cliente cerró la conexión
			return 

        case <-heartbeat.C:
            // Enviar comentario para mantener vivo
            fmt.Fprintf(w, ": keep-alive\n\n")
            w.(http.Flusher).Flush()

		case msg, open := <-messageChan:
			if !open {
				return
			}
            // Escribir el mensaje binario (codificado en base64 o raw si el cliente aguanta)
            // SSE es texto, así que lo ideal es enviar Base64 del PacketResult binario
            // O enviar JSON si prefieres debug. Asumiremos Base64 de TinyBin.
			fmt.Fprintf(w, "data: %s\n\n", string(msg)) 
			w.(http.Flusher).Flush()
		}
	}
}

// Send envía un mensaje a un usuario específico
func (b *Broker) Send(userID string, data any) {
	b.mutex.RLock()
	clientChan, ok := b.clients[userID]
	b.mutex.RUnlock()

	if !ok {
		return // Usuario no conectado, aquí podrías guardar en DB para "Notificaciones offline"
	}

    // Codificar a binario (TinyBin)
    encodedBytes, err := tinybin.Encode(data) 
    if err != nil {
        return 
    }

    // Enviar con select no bloqueante para proteger al servidor
	select {
	case clientChan <- encodedBytes:
	default:
		// El canal está lleno (cliente muy lento), descartamos o logueamos
        fmt.Println("Client buffer full, dropping message for", userID)
	}
}
```

- [BROKER_DIAGRAM.mmd](BROKER_DIAGRAM.mmd)

## 2. SSE Sincronización (`Sync`)

Esta función toma todo lo acumulado y lo envía en un solo POST binario. Ideal para llamar cuando vuelve la conexión a internet o periódicamente.

```go
// Sync envía todos los paquetes pendientes al servidor
func (c *Client) Sync(callback func(error)) {
	c.mu.Lock()
	if len(c.queue) == 0 {
		c.mu.Unlock()
		callback(nil) // Nada que enviar
		return
	}
	
	// Copiar y limpiar cola inmediatamente
	batchToSend := crudp.BatchRequest{Packets: c.queue}
	c.queue = make([]crudp.Packet, 0) 
	c.mu.Unlock()

	// 1. Codificar el Lote completo a Binario
	bodyBytes, err := tinybin.Encode(batchToSend)
	if err != nil {
		callback(fmt.Errorf("encode error: %v", err))
		return
	}

	// 2. Enviar POST usando fetchgo (unificado para servidor y navegador)
	fg := fetchgo.New()
	client := fg.NewClient(c.apiEndpoint, 5000) // Ajusta timeout según necesites
	client.SendBinary("POST", "", bodyBytes, func(resp []byte, err error) {
		if err != nil {
			// TODO: Lógica de reintento. Volver a meter los paquetes a la cola?
			callback(err)
			return
		}
		// Verificar respuesta si es necesario
		// if len(resp) > 0 { ... }
		callback(nil)
	})
}
```

## 3. Código patrón "Fire and Forget" (Dispara y Olvida) para el envío, y "Reactive Listener" para la respuesta

Utiliza `syscall/js` para aprovechar la API nativa del navegador (`EventSource` y `fetch`), lo cual es mucho más ligero que la librería estándar de Go completa.

### Estructura del Cliente

El cliente tiene tres responsabilidades:

1.  **Acumular peticiones** en una cola (Batch).
2.  **Sincronizar (POST)** enviando el lote binario.
3.  **Escuchar (SSE)**, decodificar la respuesta y ejecutar el *callback* correcto.

### 3.1 Definición del Cliente (`client.go`)

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

### 3.2 Generación y Encolado (`AddToBatch`)

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

### 3.3 Ejemplo de Uso (Cómo se ve en tu código de UI)

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

## 4. El Listener SSE (El corazón reactivo)

Este código usa la API JS del navegador para escuchar eventos. Es más robusto que intentar mantener un socket abierto con `net/http` en WASM.

```go
// ListenSSE inicia la conexión persistente para recibir respuestas
func (c *Client) ListenSSE() {
	// Usamos syscall/js para acceder a new EventSource() del navegador
	jsGlobal := js.Global()
	eventSource := jsGlobal.Get("EventSource").New(c.sseEndpoint)

	// Definir el handler para el evento "message" (o eventos personalizados)
	onMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// args[0] es el evento
		eventData := args[0].Get("data").String()

		// Procesar asíncronamente para no bloquear el UI thread de JS
		go c.handleIncomingMessage(eventData)
		return nil
	})

	eventSource.Call("addEventListener", "message", onMessage)
    
    // Mantener referencia para que no lo recolecte el GC (si fuera necesario cerrar después)
	// c.sseClose = func() { eventSource.Call("close"); onMessage.Release() }
    
    // Bloquear para siempre (o manejar en una goroutine principal)
    select {} 
}

func (c *Client) handleIncomingMessage(base64Data string) {
	// 1. Decodificar Base64 (SSE es texto) -> Binario
	binaryData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		js.Global().Get("console").Call("log", "SSE Decode B64 Error:", err.Error())
		return
	}

	// 2. Decodificar Estructura BatchResponse (TinyBin)
	var batchResp crudp.BatchResponse
	if err := tinybin.Decode(binaryData, &batchResp); err != nil {
		js.Global().Get("console").Call("log", "SSE Decode TinyBin Error:", err.Error())
		return
	}

	// 3. Conciliar Respuestas con Peticiones Pendientes
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, result := range batchResp.Results {
		// Buscamos si alguien está esperando esta respuesta
		if callback, exists := c.pending[result.ReqID]; exists {
			// Ejecutamos el callback (Actualizar UI, notificar usuario, etc.)
			// Nota: Cuidado con race conditions en UI, a veces hay que volver al main thread
			go callback(result.Success, result.Message, result.Data)
			
			// Ya fue procesado, borramos del mapa
			delete(c.pending, result.ReqID)
		}
	}
}
```

## 5. Integración de Archivos y SSE Broker Eficiente

### 5.1 Estrategia de Archivos en Protocolo Unificado

#### A. Archivos Pequeños/Medianos (Directo en el Batch)

Como `CRUDP` y `tinybin` soportan nativamente `[]byte`, puedes enviar la imagen o PDF directamente dentro del paquete.

  * **Ventaja:** Atómico. Si falla la subida, falla la creación del registro asociado.
  * **Implementación:** El Handler recibe `[]byte` como cualquier otro dato.

#### B. Archivos Grandes (Upload Asíncrono + Referencia)

Para no bloquear el hilo principal ni consumir toda la RAM en un Batch de 50MB:

1.  **Paso 1 (Batch):** El cliente envía una acción `Create` con metadatos (Nombre, Tamaño, Checksum) y recibe un `UploadID` (vía SSE).
2.  **Paso 2 (Stream):** El cliente sube el binario puro a un endpoint `/upload/{UploadID}`.
3.  **Paso 3 (Confirmación):** El servidor detecta el fin de la subida y dispara el Handler final de procesamiento.

### 5.2 El `Context` Actualizado (Soporte de Archivos)

Para que tus Handlers sigan siendo agnósticos (sin saber si el archivo vino por HTTP Multipart o por un []byte en memoria), abstraemos esto en el `Context`.

#### Archivo: `pkg/router/context.go`

```go
package router

import (
	"context"
	"io"
)

// File abstrae un archivo subido, sea en memoria o stream
type File struct {
	Name    string
	Size    int64
	Content []byte        // Para archivos pequeños (Enfoque A)
	Stream  io.ReadCloser // Para archivos grandes (Enfoque B)
}

type Context struct {
	Ctx      context.Context
	UserID   string
	UserRole string
	// Mapa de archivos adjuntos. Clave = nombre del campo o ID temporal
	Files    map[string]File 
}
```

#### Uso en un Handler (Ejemplo: `modules/documents/handler.go`)

```go
func (h *Handler) Upload(ctx router.Context, data ...any) (any, error) {
    // El handler no sabe de HTTP ni Multipart. 
    // Solo pide el archivo por su clave.
    
    docFile, ok := ctx.Files["contract_pdf"]
    if !ok {
         // Si no está en el mapa, quizás viene en el payload data (Enfoque A)
         if len(data) > 0 {
             if bytes, ok := data[0].([]byte); ok {
                 // Procesar bytes...
                 return "uploaded_from_bytes", nil
             }
         }
         return nil, errors.New("no file found")
    }
    
    // Procesar docFile.Content o docFile.Stream...
    return "uploaded_from_context", nil
}
```