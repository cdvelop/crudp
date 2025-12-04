# Step 3: Configuration System

> **Prerequisite:** [STEP_02_HANDLER_REGISTRATION.md](./STEP_02_HANDLER_REGISTRATION.md)  
> **Next:** [STEP_04_BROKER_BATCHING.md](./STEP_04_BROKER_BATCHING.md)

## Objective

- Create `Codec` interface for serialization abstraction
- Create `Config` struct (without Logger - configured by method)
- Create `New(cfg *Config)` and `NewDefault()`
- Adapt tinyjson as default codec
- Methods `SetLogger()` and `DisableLogger()`

---

## 3.1 Create Codec interface

**File:** `codec.go` (create new)

```go
package crudp

// Codec interface for serialization (replaces direct tinybin dependency)
type Codec interface {
    Encode(data any) ([]byte, error)
    Decode(data []byte, v any) error
}
```

---

## 3.2 Create Config struct

**File:** `config.go` (create new)

```go
package crudp

// UserProvider provides user identification for SSE routing
type UserProvider interface {
    GetUserID(ctx any) string
}

// Config contains CrudP configuration
// NOTE: Logger is NOT here - configured via SetLogger()
type Config struct {
    // Codec for serialization. Default: tinyjson.New()
    Codec Codec
    
    // UseBinary uses binary encoding. Default: false (JSON)
    UseBinary bool
    
    // APIEndpoint for batch requests. Default: "/api"
    APIEndpoint string
    
    // SSEEndpoint for event stream. Default: "/events"
    SSEEndpoint string
    
    // BatchWindow in milliseconds. Default: 50
    BatchWindow int
    
    // MaxRetries for failed requests. Default: 3
    MaxRetries int
    
    // RetryInterval base in ms. Default: 1000
    RetryInterval int
    
    // Port for HTTP server (server only). Default: ":6060"
    Port string
    
    // UserProvider for SSE routing (server only). Default: nil
    UserProvider UserProvider
    
    // ServerURL base (client only). Default: "" (same origin)
    ServerURL string
    
    // OnMessage callback for notifications (client only)
    OnMessage func(msgType uint8, message string)
}

// DefaultConfig returns configuration with default values
func DefaultConfig() *Config {
    return &Config{
        Codec:         nil, // Will assign tinyjson in New()
        UseBinary:     false,
        APIEndpoint:   "/api",
        SSEEndpoint:   "/events",
        BatchWindow:   50,
        MaxRetries:    3,
        RetryInterval: 1000,
        Port:          ":6060",
    }
}
```

---

## 3.3 Create tinyjson adapter

**File:** `codec_tinyjson.go` (create new)

```go
package crudp

import "github.com/cdvelop/tinyjson"

// tinyjsonCodec adapts TinyJSON to the Codec interface
type tinyjsonCodec struct {
    tj *tinyjson.TinyJSON
}

// getDefaultCodec returns the default codec (tinyjson)
func getDefaultCodec() Codec {
    return &tinyjsonCodec{
        tj: tinyjson.New(),
    }
}

func (c *tinyjsonCodec) Encode(data any) ([]byte, error) {
    return c.tj.Encode(data)
}

func (c *tinyjsonCodec) Decode(data []byte, v any) error {
    return c.tj.Decode(data, v)
}
```

---

## 3.4 Update CrudP struct and constructor

**File:** `crudp.go`

**Current code:**
```go
type CrudP struct {
    handlers    []actionHandler
    tinyBin     *tinybin.TinyBin
    log         func(msg ...any)
    apiEndpoint string
}

func New(args ...any) *CrudP {
    // ... variadic parsing
}
```

**Replace with:**
```go
package crudp

import (
    "context"
)

// actionHandler groups CRUD functions for a registration index
type actionHandler struct {
    name    string
    index   uint8
    handler any
    Create  func(context.Context, ...any) any
    Read    func(context.Context, ...any) any
    Update  func(context.Context, ...any) any
    Delete  func(context.Context, ...any) any
}

// CrudP handles automatic handler processing
// Uses slices instead of maps for TinyGo compatibility
type CrudP struct {
    config   *Config
    handlers []actionHandler
    codec    Codec
    log      func(...any) // Never nil - uses no-op by default
}

// noopLogger is the default logger that does nothing
func noopLogger(...any) {}

// New creates a new CrudP instance with configuration
func New(cfg *Config) *CrudP {
    if cfg == nil {
        cfg = DefaultConfig()
    }
    
    // Assign default codec if not provided
    codec := cfg.Codec
    if codec == nil {
        codec = getDefaultCodec()
    }
    
    return &CrudP{
        config: cfg,
        codec:  codec,
        log:    noopLogger, // Logger disabled by default
    }
}

// NewDefault creates CrudP with default configuration
func NewDefault() *CrudP {
    return New(nil)
}

// SetLogger configures a custom logging function
// Pass nil to restore no-op logger
func (cp *CrudP) SetLogger(logger func(...any)) {
    if logger == nil {
        cp.log = noopLogger
        return
    }
    cp.log = logger
}

// DisableLogger disables logging
func (cp *CrudP) DisableLogger() {
    cp.log = noopLogger
}

// Config returns the current configuration (read-only)
func (cp *CrudP) Config() *Config {
    return cp.config
}

// Codec returns the current codec
func (cp *CrudP) Codec() Codec {
    return cp.codec
}

// SetCodec allows changing the codec at runtime
func (cp *CrudP) SetCodec(codec Codec) {
    if codec != nil {
        cp.codec = codec
    }
}
```

---

## 3.5 Update tinyBin references to codec

**File:** `packet.go`

Find and replace all references:

```go
// BEFORE:
cp.tinyBin.Encode(...)
cp.tinyBin.Decode(...)

// AFTER:
cp.codec.Encode(...)
cp.codec.Decode(...)
```

**Example in EncodePacket:**
```go
func (cp *CrudP) EncodePacket(action byte, handlerID uint8, reqID string, data ...any) ([]byte, error) {
    encoded := make([][]byte, 0, len(data))
    for _, item := range data {
        bytes, err := cp.codec.Encode(item)  // Change tinyBin -> codec
        if err != nil {
            return nil, err
        }
        encoded = append(encoded, bytes)
    }

    packet := Packet{
        Action:    action,
        HandlerID: handlerID,
        ReqID:     reqID,
        Data:      encoded,
    }

    return cp.codec.Encode(packet)  // Change tinyBin -> codec
}
```

---

## 3.6 Remove tinybin import

**File:** `crudp.go`

Remove:
```go
import "github.com/cdvelop/tinybin"
```

**File:** `go.mod`

Run after changes:
```bash
go mod tidy
```

---

## 3.7 Tests

### File: `config_shared_test.go`

```go
package crudp_test

import (
    "testing"
    
    "github.com/cdvelop/crudp"
)

func ConfigDefaultsShared(t *testing.T) {
    t.Run("DefaultConfig Values", func(t *testing.T) {
        cfg := crudp.DefaultConfig()
        
        if cfg.APIEndpoint != "/api" {
            t.Errorf("expected /api, got %s", cfg.APIEndpoint)
        }
        if cfg.SSEEndpoint != "/events" {
            t.Errorf("expected /events, got %s", cfg.SSEEndpoint)
        }
        if cfg.BatchWindow != 50 {
            t.Errorf("expected 50, got %d", cfg.BatchWindow)
        }
        if cfg.MaxRetries != 3 {
            t.Errorf("expected 3, got %d", cfg.MaxRetries)
        }
        if cfg.Port != ":6060" {
            t.Errorf("expected :6060, got %s", cfg.Port)
        }
    })
}

func NewWithConfigShared(t *testing.T) {
    t.Run("Custom Config", func(t *testing.T) {
        cfg := &crudp.Config{
            APIEndpoint: "/api/v2",
            BatchWindow: 100,
            Port:        ":8080",
        }
        
        cp := crudp.New(cfg)
        
        if cp.Config().APIEndpoint != "/api/v2" {
            t.Error("config not applied")
        }
        if cp.Config().BatchWindow != 100 {
            t.Error("BatchWindow not applied")
        }
    })
    
    t.Run("Nil Config Uses Defaults", func(t *testing.T) {
        cp := crudp.New(nil)
        
        if cp.Config() == nil {
            t.Error("expected default config")
        }
        if cp.Config().APIEndpoint != "/api" {
            t.Error("expected default APIEndpoint")
        }
    })
    
    t.Run("NewDefault", func(t *testing.T) {
        cp := crudp.NewDefault()
        
        if cp.Codec() == nil {
            t.Error("expected default codec")
        }
        if cp.Config().APIEndpoint != "/api" {
            t.Error("expected default config")
        }
    })
}

func LoggerConfigShared(t *testing.T) {
    t.Run("Logger Disabled By Default", func(t *testing.T) {
        cp := crudp.NewDefault()
        
        // This should not cause panic
        // Internal logger is no-op
        cp.DisableLogger()
    })
    
    t.Run("SetLogger Custom", func(t *testing.T) {
        cp := crudp.NewDefault()
        
        var logged []any
        cp.SetLogger(func(args ...any) {
            logged = append(logged, args...)
        })
        
        // Register handler to trigger log
        err := cp.RegisterHandler(&testLogHandler{})
        if err != nil {
            t.Fatal(err)
        }
        
        if len(logged) == 0 {
            t.Error("expected log output")
        }
    })
    
    t.Run("SetLogger Nil Restores NoOp", func(t *testing.T) {
        cp := crudp.NewDefault()
        
        cp.SetLogger(func(args ...any) {
            // Custom logger
        })
        
        cp.SetLogger(nil) // Should restore no-op
        
        // Should not cause panic
        cp.DisableLogger()
    })
}

type testLogHandler struct{}

func (h *testLogHandler) Create(ctx any, data ...any) any {
    return "ok"
}

func CodecShared(t *testing.T) {
    t.Run("Default Codec EncodeDecode", func(t *testing.T) {
        cp := crudp.NewDefault()
        
        type testData struct {
            Name  string `json:"name"`
            Value int    `json:"value"`
        }
        
        original := testData{Name: "test", Value: 42}
        
        encoded, err := cp.Codec().Encode(original)
        if err != nil {
            t.Fatalf("encode error: %v", err)
        }
        
        var decoded testData
        if err := cp.Codec().Decode(encoded, &decoded); err != nil {
            t.Fatalf("decode error: %v", err)
        }
        
        if decoded.Name != original.Name || decoded.Value != original.Value {
            t.Errorf("decode mismatch: got %+v, want %+v", decoded, original)
        }
    })
    
    t.Run("SetCodec Custom", func(t *testing.T) {
        cp := crudp.NewDefault()
        originalCodec := cp.Codec()
        
        // Create custom mock codec
        mockCodec := &mockCodec{}
        cp.SetCodec(mockCodec)
        
        if cp.Codec() == originalCodec {
            t.Error("codec should have changed")
        }
    })
    
    t.Run("SetCodec Nil Ignored", func(t *testing.T) {
        cp := crudp.NewDefault()
        originalCodec := cp.Codec()
        
        cp.SetCodec(nil)
        
        if cp.Codec() != originalCodec {
            t.Error("nil codec should be ignored")
        }
    })
}

// Mock codec for tests
type mockCodec struct{}

func (m *mockCodec) Encode(data any) ([]byte, error) {
    return []byte("mock"), nil
}

func (m *mockCodec) Decode(data []byte, v any) error {
    return nil
}
```

### File: `config_stlib_test.go`

```go
//go:build !wasm

package crudp_test

import "testing"

func TestConfig_Stdlib(t *testing.T) {
    t.Run("Defaults", func(t *testing.T) {
        ConfigDefaultsShared(t)
    })
    
    t.Run("NewWithConfig", func(t *testing.T) {
        NewWithConfigShared(t)
    })
    
    t.Run("Logger", func(t *testing.T) {
        LoggerConfigShared(t)
    })
    
    t.Run("Codec", func(t *testing.T) {
        CodecShared(t)
    })
}
```

### File: `config_wasm_test.go`

```go
//go:build wasm

package crudp_test

import "testing"

func TestConfig_WASM(t *testing.T) {
    t.Run("Defaults", func(t *testing.T) {
        ConfigDefaultsShared(t)
    })
    
    t.Run("NewWithConfig", func(t *testing.T) {
        NewWithConfigShared(t)
    })
    
    t.Run("Logger", func(t *testing.T) {
        LoggerConfigShared(t)
    })
    
    t.Run("Codec", func(t *testing.T) {
        CodecShared(t)
    })
}
```

---

## 3.8 Verification

```bash
# Verify it compiles without tinybin
go build ./...

# Stdlib tests
go test -v -run TestConfig

# WASM tests
GOOS=js GOARCH=wasm go test -v -tags wasm -run TestConfig

# Clean dependencies
go mod tidy
```

---

## Notes

- **Logger by method:** Not in Config to avoid distributed nil checks. `SetLogger()` and `DisableLogger()` handle the state
- **No-op logger:** There's always a valid logger, never nil
- **Injectable Codec:** Allows using tinyjson, tinybin, or custom
- **Immutable Config:** `Config()` returns reference but should not be modified after `New()`

---

> **Next step:** [STEP_04_BROKER_BATCHING.md](./STEP_04_BROKER_BATCHING.md)
