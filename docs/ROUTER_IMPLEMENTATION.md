### 4. Implementación del Router (Server Side)

Este es el cerebro que conecta HTTP -\> CRUDP -\> SSE.

#### `router.go` (Simplificado)

```go
package crudp

import (
    "github.com/cdvelop/tinybin" 
    "net/http"
    "github.com/cdvelop/crudp"
)

// Simulación de canal de eventos SSE por usuario
var sseChannels = make(map[string]chan crudp.BatchResponse)

func (cp *CrudP) SyncHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Identificar usuario (Middleware ya validó auth)
    userID := r.Header.Get("X-User-ID")
    
    // 2. Leer Body (BatchRequest binario)
    // bodyBytes, _ := io.ReadAll(r.Body)
    // var batch crudp.BatchRequest
    // tinybin.Decode(bodyBytes, &batch)

    // 3. Crear Contexto Desacoplado usando context.WithValue
    ctx := context.WithValue(r.Context(), "user_id", userID)
    // Agregar más valores si es necesario, ej: context.WithValue(ctx, "user_role", role)

    // 4. Enviar a procesar (No bloquea la respuesta HTTP)
    go cp.processBatchAsync(ctx)

    w.WriteHeader(http.StatusAccepted) // 202
}

func (cp *CrudP) processBatchAsync(ctx context.Context,) {
    var response crudp.BatchResponse

    // Procesamos cada paquete individualmente (Fallo Parcial permitido)
    for _, packet := range batch.Packets {
        
        resultData, err := crudp.InvokeHandler(ctx, packet.HandlerID, packet.Action, packet.Data)
        
        res := crudp.PacketResult{
            ReqID: packet.ReqID, // CRUCIAL: Devolver el mismo ID
            Success: err == nil,
        }

        if err != nil {
            res.Message = err.Error()
        } else {
            // Codificar resultado a bytes si es necesario
            res.Data, _ = tinybin.Encode(resultData)
        }
        
        response.Results = append(response.Results, res)
    }

    // 5. Enviar respuesta al canal SSE del usuario
    userID := ctx.Value("user_id").(string)
    if ch, ok := sseChannels[userID]; ok {
        ch <- response
    }
}
```