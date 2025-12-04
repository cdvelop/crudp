# CRUDP Implementation Prompt

> **For LLM:** Follow these steps in order. Each step is in a separate file.

## Context

You are implementing CRUDP, a JSON/binary CRUD protocol for isomorphic Go applications. The codebase uses:

- **TinyGo Compatibility:** No maps in hot paths, minimal allocations
- **Build tags:** `//go:build wasm` for frontend, `//go:build !wasm` for backend
- **Error handling:** Use `github.com/cdvelop/tinystring` (Err, Errf) - already imported with dot
- **No fmt:** Avoid fmt package for TinyGo compatibility
- **Current Codec:** tinybin (will be replaced with Codec interface, default: tinyjson)

## Current State Analysis

**File: `crudp.go`**
- Uses `tinybin.TinyBin` directly → needs Codec interface
- Has variadic `New(args ...any)` → needs `New(cfg *Config)`
- `actionHandler` struct exists → needs `name string` field

**File: `packet.go`**
- `PacketResult` has `Success bool` → needs `MessageType uint8`
- `Data []byte` → should be `Data [][]byte` for multiple responses
- Good: `Packet` structure is correct with `Data [][]byte`

**File: `handlers.go`**
- Good: `bind()` correctly binds CRUD interfaces
- Needs: `Validator` verification before calling handler
- `HandlerName()` will be optional with reflection fallback

**File: `interfaces.go`**
- Returns `[]any` → needs to return `any`
- Missing: `Validator`, `FieldValidator`, `Codec` interfaces
- **IMPORTANT:** The `Response` interface defines the response contract:
  ```go
  type Response interface {
      Response() (data any, broadcast []string, err error)
  }
  ```
  Handlers return `any` but internally can return multiple `Response` for broadcast.

---

## Implementation Order

Complete each step **completely** (code + tests) before moving to the next.

| Step | File | Description |
|------|---------|-------------|
| 1 | [STEP_01_PACKET_STRUCTURE.md](./STEP_01_PACKET_STRUCTURE.md) | Update PacketResult with MessageType and Data [][]byte - completado |
| 2 | [STEP_02_HANDLER_REGISTRATION.md](./STEP_02_HANDLER_REGISTRATION.md) | Handler name via reflection, Validator, return `any` - completado |
| 3 | [STEP_03_CONFIG_SYSTEM.md](./STEP_03_CONFIG_SYSTEM.md) | Config system with Codec interface (Logger only by method) |
| 4 | [STEP_04_BROKER_BATCHING.md](./STEP_04_BROKER_BATCHING.md) | Broker with consolidation and tinytime Timer |
| 5 | [STEP_05_UPDATE_EXISTING.md](./STEP_05_UPDATE_EXISTING.md) | Update existing code and tests |

---

## Important Notes

### 1. Import tinystring correctly
```go
import . "github.com/cdvelop/tinystring"
// Use: Err(), Errf(), Msg.Error, Msg.Success, etc.
```

### 2. MessageType from tinystring
```go
// In tinystring/messagetype.go:
var Msg = struct {
    Normal  MessageType  // 0
    Info    MessageType  // 1
    Error   MessageType  // 2
    Warning MessageType  // 3
    Success MessageType  // 4
}{0, 1, 2, 3, 4}
```

### 3. TinyJSON uses `TinyJSON` (not `TinyJson`)
```go
import "github.com/cdvelop/tinyjson"
tj := tinyjson.New() // Returns *TinyJSON
```

### 4. Test Pattern (like tinyjson)
Tests must work in both environments using separate files:

```
*_shared_test.go  - Shared logic (no build tags)
*_stlib_test.go   - Backend entry point (//go:build !wasm)
*_wasm_test.go    - WASM entry point (//go:build wasm)
```

**Example:**
```go
// packet_shared_test.go (no build tags)
package crudp_test

func PacketResultShared(t *testing.T, cp *crudp.CrudP) {
    // Test logic here
}

// packet_stlib_test.go
//go:build !wasm

package crudp_test

func TestPacketResult(t *testing.T) {
    cp := crudp.NewDefault()
    PacketResultShared(t, cp)
}

// packet_wasm_test.go
//go:build wasm

package crudp_test

func TestPacketResult(t *testing.T) {
    cp := crudp.NewDefault()
    PacketResultShared(t, cp)
}
```

### 5. Response Interface for Multiple Responses
```go
// Handlers return `any` which can be:
// - A simple value for direct response
// - []Response for multiple broadcast
// - Individual Response for SSE routing

type Response interface {
    Response() (data any, broadcast []string, err error)
}

// The Data [][]byte design allows multiple responses:
// - Each []byte in Data is an encoded response
// - Allows batch responses and broadcast to multiple destinations
```

### 6. Build tags where necessary
```go
//go:build wasm
//go:build !wasm
```

### 7. Handler Name via Reflection
Use `reflect.TypeOf(handler).Elem().Name()` and convert to snake_case with `tinystring.SnakeLow()`:
```go
import . "github.com/cdvelop/tinystring"

func getHandlerName(handler any) string {
    t := reflect.TypeOf(handler)
    if t.Kind() == reflect.Ptr {
        t = t.Elem()
    }
    return Conv(t.Name()).SnakeLow().String()
}
// &UserHandler{} -> "user_handler"
```

---

## Final Verification Checklist

- [ ] All tests pass: `go test ./...`
- [ ] WASM tests pass: `GOOS=js GOARCH=wasm go test -tags wasm`
- [ ] No compilation errors
- [ ] PacketResult uses MessageType instead of Success
- [ ] PacketResult.Data is [][]byte for multiple responses
- [ ] Handler name optional via reflection with SnakeLow
- [ ] CRUD methods return `any` (not `[]any`)
- [ ] Logger configurable only via SetLogger/DisableLogger method (not in Config)
- [ ] Config system with Codec injection works
- [ ] Broker consolidates packets by Handler+Action
- [ ] No maps used (except in tests)
- [ ] No fmt package used (use tinystring)
