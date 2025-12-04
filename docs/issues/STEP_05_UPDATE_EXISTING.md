# Step 5: Update Existing Code

> **Prerequisite:** [STEP_04_BROKER_BATCHING.md](./STEP_04_BROKER_BATCHING.md)  
> **Next:** Final verification

## Objective

- Update existing tests to use new API
- Update code that depends on previous changes
- Verify compilation and tests in both environments

---

## 5.1 Update processSinglePacket

**File:** `packet.go`

Update to use new structure:

```go
func (cp *CrudP) processSinglePacket(ctx context.Context, packet *Packet) (PacketResult, error) {
    pr := PacketResult{
        Packet: *packet, // Embed original packet
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
    
    // Process result by type
    if err := cp.encodeResultToPacket(&pr, result); err != nil {
        pr.MessageType = uint8(Msg.Error)
        pr.Message = err.Error()
        return pr, err
    }
    
    pr.MessageType = uint8(Msg.Success)
    pr.Message = "OK"
    return pr, nil
}

// encodeResultToPacket encodes result to Data [][]byte
func (cp *CrudP) encodeResultToPacket(pr *PacketResult, result any) error {
    if result == nil {
        return nil
    }
    
    // Case 1: Slice of Response for multiple broadcast
    if responses, ok := result.([]Response); ok {
        pr.Data = make([][]byte, 0, len(responses))
        for _, resp := range responses {
            data, broadcast, err := resp.Response()
            if err != nil {
                return err
            }
            
            // SSE routing if broadcast targets exist
            if len(broadcast) > 0 {
                cp.routeToSSE(data, broadcast, pr.HandlerID)
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
        data, broadcast, err := resp.Response()
        if err != nil {
            return err
        }
        
        if len(broadcast) > 0 {
            cp.routeToSSE(data, broadcast, pr.HandlerID)
        }
        
        encoded, err := cp.codec.Encode(data)
        if err != nil {
            return err
        }
        pr.Data = [][]byte{encoded}
        return nil
    }
    
    // Case 3: Direct value (no Response)
    encoded, err := cp.codec.Encode(result)
    if err != nil {
        return err
    }
    pr.Data = [][]byte{encoded}
    return nil
}
```

---

## 5.2 Update ProcessBatch

**File:** `packet.go`

```go
func (cp *CrudP) ProcessBatch(ctx context.Context, requestBytes []byte) ([]byte, error) {
    var batchReq BatchRequest
    if err := cp.codec.Decode(requestBytes, &batchReq); err != nil {
        return cp.createErrorBatchResponse("decode_error", err)
    }

    results := make([]PacketResult, 0, len(batchReq.Packets))

    for _, packet := range batchReq.Packets {
        result, err := cp.processSinglePacket(ctx, &packet)
        results = append(results, result)
        if err != nil {
            // Continue processing other packets even if one fails
            continue
        }
    }

    batchResp := BatchResponse{
        Results: results,
    }

    return cp.codec.Encode(batchResp)
}
```

---

## 5.3 Update createErrorBatchResponse

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

## 5.4 Update existing handlers in tests

**File:** Any test that uses handlers

Test handlers must be updated to:
1. Return `any` instead of `[]any`
2. Optionally implement `HandlerName()` (or use name by reflection)

**Example of updated handler:**

```go
// BEFORE:
type OldHandler struct{}

func (h *OldHandler) Create(ctx context.Context, data ...any) []any {
    return []any{"created", nil}
}

// AFTER:
type NewHandler struct{}

// HandlerName is OPTIONAL - if not implemented, uses reflection:
// "NewHandler" -> "new_handler"
func (h *NewHandler) HandlerName() string {
    return "my_handler" // Optional override
}

func (h *NewHandler) Create(ctx context.Context, data ...any) any {
    return "created" // Returns any, not []any
}
```

---

## 5.5 Update decodeWithKnownType

**File:** `handlers.go`

Ensure it uses `cp.codec` instead of `cp.tinyBin`:

```go
func (cp *CrudP) decodeWithKnownType(packet *Packet, handlerID uint8) ([]any, error) {
    // ... existing code ...
    
    // Replace all references:
    // BEFORE: cp.tinyBin.Decode(...)
    // AFTER: cp.codec.Decode(...)
}
```

---

## 5.6 Create test script

**File:** `test.sh` (create new)

```bash
#!/bin/bash

echo "=========================================="
echo "Running Stdlib Tests..."
echo "=========================================="
go test -v ./...

if [ $? -ne 0 ]; then
    echo "❌ Stdlib tests failed"
    exit 1
fi

echo ""
echo "=========================================="
echo "Running WASM Tests..."
echo "=========================================="

# Check if wasmbrowsertest is installed
if ! command -v wasmbrowsertest &> /dev/null; then
    echo "⚠️  wasmbrowsertest not found. Install it with:"
    echo "   go install github.com/agnivade/wasmbrowsertest@latest"
    echo "   export PATH=\$PATH:\$(go env GOPATH)/bin"
    exit 1
fi

# Run WASM tests
GOOS=js GOARCH=wasm go test -v -tags wasm 2>&1 | grep -v "ERROR: could not unmarshal"
WASM_EXIT_CODE=$?

if [ $WASM_EXIT_CODE -ne 0 ]; then
    echo ""
    echo "❌ WASM tests failed"
    exit 1
fi

echo ""
echo "✅ All tests passed!"
```

```bash
chmod +x test.sh
```

---

## 5.7 Update go.mod

```bash
# Add tinytime dependency if not exists
go get github.com/cdvelop/tinytime

# Add tinyjson dependency if not exists  
go get github.com/cdvelop/tinyjson

# Clean unused dependencies (tinybin)
go mod tidy
```

---

## 5.8 Complete Verification

### Compilation
```bash
# Backend
go build ./...

# WASM
GOOS=js GOARCH=wasm go build ./...
```

### Tests
```bash
# Run all tests
./test.sh

# Or manually:
go test -v ./...
GOOS=js GOARCH=wasm go test -v -tags wasm
```

### Checklist

- [ ] `go build ./...` no errors
- [ ] `GOOS=js GOARCH=wasm go build ./...` no errors
- [ ] `go test ./...` all pass
- [ ] `GOOS=js GOARCH=wasm go test -tags wasm` all pass
- [ ] No references to `tinybin`
- [ ] No references to `fmt` (use `tinystring`)
- [ ] All handlers use new API (`any` instead of `[]any`)
- [ ] `PacketResult` uses `MessageType` and `Data [][]byte`
- [ ] Logger configured only via method
- [ ] Broker works with consolidation

---

## 5.9 Existing Code Migration

If you have code using the old API, here's the migration guide:

### Constructor
```go
// BEFORE:
cp := crudp.New(logFunc, "/api/v2")

// AFTER:
cp := crudp.New(&crudp.Config{
    APIEndpoint: "/api/v2",
})
cp.SetLogger(logFunc)
```

### Handlers
```go
// BEFORE:
func (h *Handler) Create(ctx context.Context, data ...any) []any {
    return []any{result, nil}
}

// AFTER:
func (h *Handler) Create(ctx context.Context, data ...any) any {
    return result
}
```

### PacketResult
```go
// BEFORE:
if result.Success {
    // ...
}

// AFTER:
if result.MessageType == uint8(Msg.Success) {
    // ...
}

// BEFORE:
data := result.Data // []byte

// AFTER:
data := result.Data // [][]byte - use result.Data[0] for first response
```

---

## Summary of Modified Files

| File | Changes |
|---------|---------|
| `crudp.go` | New struct, `New(cfg)`, logger methods |
| `packet.go` | `PacketResult` with `MessageType` and `Data [][]byte` |
| `handlers.go` | `callHandler` returns `any`, validation, reflection name |
| `interfaces.go` | CRUD returns `any`, new interfaces |
| `config.go` | New file with `Config` |
| `codec.go` | New file with interface |
| `codec_tinyjson.go` | New file with adapter |
| `broker.go` | New file with batching |
| `actions.go` | New file with conversions |
| `test.sh` | New test script |

---

> **Implementation completed.** Run `./test.sh` to verify.
