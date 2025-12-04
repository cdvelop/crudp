# Step 1: Update Packet Structure

> **Prerequisite:** None  
> **Next:** [STEP_02_HANDLER_REGISTRATION.md](./STEP_02_HANDLER_REGISTRATION.md)

## Objective

Update `PacketResult` to use `MessageType uint8` instead of `Success bool`, and `Data [][]byte` to support multiple responses.

---

## 1.1 Update PacketResult

**File:** `packet.go`

**Current code:**
```go
type PacketResult struct {
    ReqID   string
    Success bool
    Message string
    Data    []byte
}
```

**Replace with:**
```go
type PacketResult struct {
    Packet              // Embed Packet complete for symmetry with BatchRequest
    MessageType uint8   // tinystring.MessageType (0=Normal, 1=Info, 2=Error, 3=Warning, 4=Success)
    Message     string  // Message for the user
}
```

**Justification:**
- `MessageType` allows more states than just success/error (Info, Warning, etc.)
- Embed `Packet` includes `Data [][]byte` for multiple responses
- Symmetry: `BatchRequest` has `[]Packet`, `BatchResponse` has `[]PacketResult` with Packet embedded

---

## 1.2 Create action conversion functions

**File:** `actions.go` (create new)

```go
package crudp

// methodToAction converts HTTP method to CRUD action byte
func methodToAction(method string) byte {
    switch method {
    case "POST":
        return 'c'
    case "GET":
        return 'r'
    case "PUT":
        return 'u'
    case "DELETE":
        return 'd'
    default:
        return 0
    }
}

// actionToMethod converts CRUD action byte to HTTP method
func actionToMethod(action byte) string {
    switch action {
    case 'c':
        return "POST"
    case 'r':
        return "GET"
    case 'u':
        return "PUT"
    case 'd':
        return "DELETE"
    default:
        return ""
    }
}
```

---

## 1.3 Update processSinglePacket

**File:** `packet.go`

Update to use `MessageType` instead of `Success`:

```go
func (cp *CrudP) processSinglePacket(ctx context.Context, packet *Packet) (PacketResult, error) {
    pr := PacketResult{
        Packet: *packet, // Embed original packet (includes Data [][]byte)
    }
    
    // Decode data with known types
    decodedData, err := cp.decodeWithKnownType(packet, packet.HandlerID)
    if err != nil {
        pr.MessageType = uint8(Msg.Error)
        pr.Message = err.Error()
        return pr, err
    }
    
    // Call handler
    result, err := cp.callHandler(ctx, packet.HandlerID, packet.Action, decodedData...)
    if err != nil {
        pr.MessageType = uint8(Msg.Error)
        pr.Message = err.Error()
        return pr, err
    }
    
    // Process result - can be multiple Response
    if err := cp.encodeResultToPacket(&pr, result); err != nil {
        pr.MessageType = uint8(Msg.Error)
        pr.Message = err.Error()
        return pr, err
    }
    
    pr.MessageType = uint8(Msg.Success)
    pr.Message = "OK"
    return pr, nil
}

// encodeResultToPacket encodes handler result to Data [][]byte
func (cp *CrudP) encodeResultToPacket(pr *PacketResult, result any) error {
    if result == nil {
        return nil
    }
    
    // Case 1: Slice of Response for multiple broadcast
    if responses, ok := result.([]Response); ok {
        pr.Data = make([][]byte, 0, len(responses))
        for _, resp := range responses {
            data, _, err := resp.Response()
            if err != nil {
                return err
            }
            encoded, err := cp.codec.Encode(data)
            if err != nil {
                return err
            }
            pr.Data = append(pr.Data, encoded)
        }
        return nil
    }
    
    // Case 2: Individual Response
    if resp, ok := result.(Response); ok {
        data, _, err := resp.Response()
        if err != nil {
            return err
        }
        encoded, err := cp.codec.Encode(data)
        if err != nil {
            return err
        }
        pr.Data = [][]byte{encoded}
        return nil
    }
    
    // Case 3: Direct value
    encoded, err := cp.codec.Encode(result)
    if err != nil {
        return err
    }
    pr.Data = [][]byte{encoded}
    return nil
}
```

---

## 1.4 Update createErrorBatchResponse

**File:** `packet.go`

```go
func (cp *CrudP) createErrorBatchResponse(reqID string, err error) ([]byte, error) {
    result := PacketResult{
        Packet:      Packet{ReqID: reqID},
        MessageType: uint8(Msg.Error),
        Message:     err.Error(),
    }
    
    return cp.codec.Encode(BatchResponse{Results: []PacketResult{result}})
}
```

---

## 1.5 Tests

### File: `packet_shared_test.go`

```go
package crudp_test

import (
    "testing"
    
    "github.com/cdvelop/crudp"
    . "github.com/cdvelop/tinystring"
)

func PacketResultMessageTypeShared(t *testing.T) {
    t.Run("MessageType Success", func(t *testing.T) {
        pr := crudp.PacketResult{
            Packet:      crudp.Packet{Action: 'c', HandlerID: 0, ReqID: "test-1"},
            MessageType: uint8(Msg.Success),
            Message:     "Created",
        }
        
        if pr.MessageType != uint8(Msg.Success) {
            t.Errorf("expected MessageType %d, got %d", uint8(Msg.Success), pr.MessageType)
        }
        
        if pr.Action != 'c' {
            t.Errorf("expected Action 'c', got %c", pr.Action)
        }
        
        if pr.ReqID != "test-1" {
            t.Errorf("expected ReqID 'test-1', got %s", pr.ReqID)
        }
    })
    
    t.Run("MessageType Error", func(t *testing.T) {
        pr := crudp.PacketResult{
            Packet:      crudp.Packet{Action: 'r', HandlerID: 1, ReqID: "test-2"},
            MessageType: uint8(Msg.Error),
            Message:     "Not found",
        }
        
        if pr.MessageType != uint8(Msg.Error) {
            t.Errorf("expected MessageType %d, got %d", uint8(Msg.Error), pr.MessageType)
        }
    })
    
    t.Run("Multiple Data Responses", func(t *testing.T) {
        pr := crudp.PacketResult{
            Packet: crudp.Packet{
                Action:    'r',
                HandlerID: 0,
                ReqID:     "test-3",
                Data: [][]byte{
                    []byte(`{"id":1,"name":"Alice"}`),
                    []byte(`{"id":2,"name":"Bob"}`),
                    []byte(`{"id":3,"name":"Charlie"}`),
                },
            },
            MessageType: uint8(Msg.Success),
            Message:     "OK",
        }
        
        if len(pr.Data) != 3 {
            t.Errorf("expected 3 data items, got %d", len(pr.Data))
        }
    })
}

func ActionConversionShared(t *testing.T) {
    tests := []struct {
        method string
        action byte
    }{
        {"POST", 'c'},
        {"GET", 'r'},
        {"PUT", 'u'},
        {"DELETE", 'd'},
        {"INVALID", 0},
    }
    
    for _, tt := range tests {
        t.Run(tt.method, func(t *testing.T) {
            got := crudp.MethodToAction(tt.method)
            if got != tt.action {
                t.Errorf("MethodToAction(%s) = %c, want %c", tt.method, got, tt.action)
            }
            
            if tt.action != 0 {
                gotMethod := crudp.ActionToMethod(tt.action)
                if gotMethod != tt.method {
                    t.Errorf("ActionToMethod(%c) = %s, want %s", tt.action, gotMethod, tt.method)
                }
            }
        })
    }
}
```

### File: `packet_stlib_test.go`

```go
//go:build !wasm

package crudp_test

import "testing"

func TestPacketResult_Stdlib(t *testing.T) {
    t.Run("MessageType", func(t *testing.T) {
        PacketResultMessageTypeShared(t)
    })
    
    t.Run("ActionConversion", func(t *testing.T) {
        ActionConversionShared(t)
    })
}
```

### File: `packet_wasm_test.go`

```go
//go:build wasm

package crudp_test

import "testing"

func TestPacketResult_WASM(t *testing.T) {
    t.Run("MessageType", func(t *testing.T) {
        PacketResultMessageTypeShared(t)
    })
    
    t.Run("ActionConversion", func(t *testing.T) {
        ActionConversionShared(t)
    })
}
```

---

## 1.6 Verification

```bash
# Stdlib tests
go test -v -run TestPacketResult

# WASM tests
GOOS=js GOARCH=wasm go test -v -tags wasm -run TestPacketResult
```

---

## Notes

- **Data [][]byte:** Allows multiple responses for broadcast to different users/groups
- **MessageType:** Uses values from `tinystring.Msg` for consistency in the ecosystem
- **Response interface:** Handlers can return `[]Response` for multiple broadcast, which is encoded to multiple entries in `Data`

---

> **Next step:** [STEP_02_HANDLER_REGISTRATION.md](./STEP_02_HANDLER_REGISTRATION.md)
