# Estructura de Paquetes (TinyGo Friendly)

Mantenemos la eficiencia binaria, pero envolvemos todo en un concepto de `Envelope` (Sobre) para lotes.

**Cambios Clave:**

1.  **`ReqID` (Correlation ID):** Esencial para asincronía. El cliente genera un ID (UnixID) y el servidor *debe* devolverlo en la respuesta SSE para que el cliente sepa qué solicitud se completó.
2.  **Batch Wrapper:** Todo envío es un array de paquetes.

```go
// crudp/packet.go

type Packet struct {
    Action    byte     // 'c', 'r', 'u', 'd'
    HandlerID uint8    // Índice en la tabla de handlers compartida
    ReqID     string   // ID único generado por el cliente (ej. UnixID)
    Data      [][]byte // Argumentos serializados
}

// BatchRequest es lo que se envía en el POST /sync
type BatchRequest struct {
    Packets []Packet
}

// BatchResponse es lo que se recibe por SSE
type BatchResponse struct {
    Results []PacketResult
}

type PacketResult struct {
    ReqID   string // Correlación con la petición original
    Success bool   // true/false
    Message string // Error o mensaje de éxito
    Data    []byte // Resultado codificado (si aplica)
}
```