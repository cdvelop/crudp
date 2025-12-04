# Step 4: Broker with Batching

> **Prerequisite:** [STEP_03_CONFIG_SYSTEM.md](./STEP_03_CONFIG_SYSTEM.md)  
> **Next:** [STEP_05_UPDATE_EXISTING.md](./STEP_05_UPDATE_EXISTING.md)

## Objective

Create broker that:
- Consolidates packets by Handler+Action
- Uses `tinytime.Timer` for batch window (WASM compatible)
- Automatic flush after BatchWindow ms

---

## 4.1 tinytime dependency

**File:** `go.mod`

Add:
```
require github.com/cdvelop/tinytime v0.x.x
```

**tinytime.Timer interface:**
```go
// AfterFunc waits the specified milliseconds and calls f
// Returns Timer that can be canceled
AfterFunc(milliseconds int, f func()) Timer

type Timer interface {
    Stop() bool
}
```

---

## 4.2 Create broker

**File:** `broker.go` (create new)

```go
package crudp

import (
    "sync"
    
    "github.com/cdvelop/tinytime"
)

// broker handles batching of packets for efficient sending
type broker struct {
    mu          sync.Mutex
    queue       []Packet      // Queue of pending packets
    batchWindow int
    timer       tinytime.Timer
    tp          tinytime.TimeProvider
    codec       Codec
    onFlush     func([]byte) // Callback to send batch
}

// newBroker creates a new broker
func newBroker(cfg *Config, codec Codec) *broker {
    return &broker{
        queue:       make([]Packet, 0, 16), // Typical pre-alloc
        batchWindow: cfg.BatchWindow,
        tp:          tinytime.NewTimeProvider(),
        codec:       codec,
    }
}

// SetOnFlush configures the flush callback
func (b *broker) SetOnFlush(fn func([]byte)) {
    b.mu.Lock()
    b.onFlush = fn
    b.mu.Unlock()
}

// Enqueue adds a packet to the queue, consolidating by Handler+Action
func (b *broker) Enqueue(handlerID uint8, action byte, reqID string, data []byte) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    // Find existing packet with same handler+action to consolidate
    for i := range b.queue {
        p := &b.queue[i]
        if p.HandlerID == handlerID && p.Action == action {
            // Consolidate: add data to existing packet
            p.Data = append(p.Data, data)
            b.resetTimerLocked()
            return
        }
    }
    
    // New packet
    b.queue = append(b.queue, Packet{
        Action:    action,
        HandlerID: handlerID,
        ReqID:     reqID,
        Data:      [][]byte{data},
    })
    
    b.resetTimerLocked()
}

// resetTimerLocked resets the flush timer (must be called with lock)
func (b *broker) resetTimerLocked() {
    if b.timer != nil {
        b.timer.Stop()
    }
    b.timer = b.tp.AfterFunc(b.batchWindow, b.flush)
}

// flush sends all packets in queue
func (b *broker) flush() {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if len(b.queue) == 0 {
        return
    }
    
    // Build BatchRequest directly from queue
    batch := BatchRequest{Packets: b.queue}
    encoded, err := b.codec.Encode(batch)
    if err != nil {
        // Log error but don't panic
        return
    }
    
    // Clear queue (keep capacity)
    b.queue = b.queue[:0]
    
    // Send if callback exists
    if b.onFlush != nil {
        b.onFlush(encoded)
    }
}

// FlushNow forces an immediate flush (useful for testing or shutdown)
func (b *broker) FlushNow() {
    if b.timer != nil {
        b.timer.Stop()
    }
    b.flush()
}

// QueueLength returns the current queue size (for testing)
func (b *broker) QueueLength() int {
    b.mu.Lock()
    defer b.mu.Unlock()
    return len(b.queue)
}

// Clear cleans the queue without sending (for reset)
func (b *broker) Clear() {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if b.timer != nil {
        b.timer.Stop()
        b.timer = nil
    }
    b.queue = b.queue[:0]
}
```

---

## 4.3 Integrate broker in CrudP

**File:** `crudp.go`

Add broker field:
```go
type CrudP struct {
    config   *Config
    handlers []actionHandler
    codec    Codec
    log      func(...any)
    broker   *broker  // Add this field
}
```

Update constructor:
```go
func New(cfg *Config) *CrudP {
    if cfg == nil {
        cfg = DefaultConfig()
    }
    
    codec := cfg.Codec
    if codec == nil {
        codec = getDefaultCodec()
    }
    
    cp := &CrudP{
        config: cfg,
        codec:  codec,
        log:    noopLogger,
    }
    
    // Initialize broker
    cp.broker = newBroker(cfg, codec)
    
    return cp
}
```

Add access methods:
```go
// Broker returns the broker for advanced configuration
func (cp *CrudP) Broker() *broker {
    return cp.broker
}

// EnqueuePacket queues a packet for batch sending
func (cp *CrudP) EnqueuePacket(handlerID uint8, action byte, reqID string, data any) error {
    encoded, err := cp.codec.Encode(data)
    if err != nil {
        return err
    }
    cp.broker.Enqueue(handlerID, action, reqID, encoded)
    return nil
}
```

---

## 4.4 Tests

### File: `broker_shared_test.go`

```go
package crudp_test

import (
    "sync"
    "testing"
    "time"
    
    "github.com/cdvelop/crudp"
)

func BrokerConsolidationShared(t *testing.T) {
    t.Run("Consolidate Same Handler+Action", func(t *testing.T) {
        cfg := crudp.DefaultConfig()
        cfg.BatchWindow = 500 // 500ms to avoid flush during test
        
        cp := crudp.New(cfg)
        broker := cp.Broker()
        
        // Queue 3 items for same handler+action
        broker.Enqueue(0, 'c', "req1", []byte(`{"name":"A"}`))
        broker.Enqueue(0, 'c', "req2", []byte(`{"name":"B"}`))
        broker.Enqueue(0, 'c', "req3", []byte(`{"name":"C"}`))
        
        // Should consolidate into 1 packet
        if broker.QueueLength() != 1 {
            t.Errorf("expected 1 packet, got %d", broker.QueueLength())
        }
    })
    
    t.Run("Different Handlers Not Consolidated", func(t *testing.T) {
        cfg := crudp.DefaultConfig()
        cfg.BatchWindow = 500
        
        cp := crudp.New(cfg)
        broker := cp.Broker()
        
        // Queue items for different handlers
        broker.Enqueue(0, 'c', "req1", []byte(`{}`))
        broker.Enqueue(1, 'c', "req2", []byte(`{}`))
        broker.Enqueue(0, 'r', "req3", []byte(`{}`)) // Same handler, different action
        
        // Should have 3 separate packets
        if broker.QueueLength() != 3 {
            t.Errorf("expected 3 packets, got %d", broker.QueueLength())
        }
    })
    
    t.Run("Different Actions Not Consolidated", func(t *testing.T) {
        cfg := crudp.DefaultConfig()
        cfg.BatchWindow = 500
        
        cp := crudp.New(cfg)
        broker := cp.Broker()
        
        // Same handler, different actions
        broker.Enqueue(0, 'c', "req1", []byte(`{}`))
        broker.Enqueue(0, 'r', "req2", []byte(`{}`))
        broker.Enqueue(0, 'u', "req3", []byte(`{}`))
        broker.Enqueue(0, 'd', "req4", []byte(`{}`))
        
        if broker.QueueLength() != 4 {
            t.Errorf("expected 4 packets, got %d", broker.QueueLength())
        }
    })
}

func BrokerFlushShared(t *testing.T) {
    t.Run("Flush After BatchWindow", func(t *testing.T) {
        cfg := crudp.DefaultConfig()
        cfg.BatchWindow = 50 // 50ms
        
        cp := crudp.New(cfg)
        broker := cp.Broker()
        
        var flushed []byte
        var wg sync.WaitGroup
        wg.Add(1)
        
        broker.SetOnFlush(func(data []byte) {
            flushed = data
            wg.Done()
        })
        
        broker.Enqueue(0, 'c', "req1", []byte(`{"test":true}`))
        
        // Wait for flush (with timeout)
        done := make(chan bool)
        go func() {
            wg.Wait()
            done <- true
        }()
        
        select {
        case <-done:
            if flushed == nil {
                t.Error("expected flush data")
            }
        case <-time.After(200 * time.Millisecond):
            t.Error("timeout waiting for flush")
        }
        
        // Queue should be empty after flush
        if broker.QueueLength() != 0 {
            t.Error("queue should be empty after flush")
        }
    })
    
    t.Run("FlushNow Forces Immediate Flush", func(t *testing.T) {
        cfg := crudp.DefaultConfig()
        cfg.BatchWindow = 5000 // 5 seconds - should not trigger
        
        cp := crudp.New(cfg)
        broker := cp.Broker()
        
        var flushed bool
        broker.SetOnFlush(func(data []byte) {
            flushed = true
        })
        
        broker.Enqueue(0, 'c', "req1", []byte(`{}`))
        
        // Immediate flush
        broker.FlushNow()
        
        if !flushed {
            t.Error("expected immediate flush")
        }
        
        if broker.QueueLength() != 0 {
            t.Error("queue should be empty")
        }
    })
    
    t.Run("Empty Queue No Flush", func(t *testing.T) {
        cfg := crudp.DefaultConfig()
        cp := crudp.New(cfg)
        broker := cp.Broker()
        
        flushed := false
        broker.SetOnFlush(func(data []byte) {
            flushed = true
        })
        
        // Flush with empty queue
        broker.FlushNow()
        
        if flushed {
            t.Error("should not flush empty queue")
        }
    })
}

func BrokerClearShared(t *testing.T) {
    t.Run("Clear Removes All", func(t *testing.T) {
        cfg := crudp.DefaultConfig()
        cfg.BatchWindow = 5000
        
        cp := crudp.New(cfg)
        broker := cp.Broker()
        
        broker.Enqueue(0, 'c', "req1", []byte(`{}`))
        broker.Enqueue(1, 'r', "req2", []byte(`{}`))
        
        if broker.QueueLength() != 2 {
            t.Fatal("setup failed")
        }
        
        broker.Clear()
        
        if broker.QueueLength() != 0 {
            t.Error("queue should be empty after clear")
        }
    })
}

func EnqueuePacketShared(t *testing.T) {
    t.Run("EnqueuePacket Encodes Data", func(t *testing.T) {
        cfg := crudp.DefaultConfig()
        cfg.BatchWindow = 5000
        
        cp := crudp.New(cfg)
        
        type testData struct {
            Name string `json:"name"`
            ID   int    `json:"id"`
        }
        
        err := cp.EnqueuePacket(0, 'c', "req1", testData{Name: "test", ID: 1})
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        
        if cp.Broker().QueueLength() != 1 {
            t.Error("expected 1 packet in queue")
        }
    })
}
```

### File: `broker_stlib_test.go`

```go
//go:build !wasm

package crudp_test

import "testing"

func TestBroker_Stdlib(t *testing.T) {
    t.Run("Consolidation", func(t *testing.T) {
        BrokerConsolidationShared(t)
    })
    
    t.Run("Flush", func(t *testing.T) {
        BrokerFlushShared(t)
    })
    
    t.Run("Clear", func(t *testing.T) {
        BrokerClearShared(t)
    })
    
    t.Run("EnqueuePacket", func(t *testing.T) {
        EnqueuePacketShared(t)
    })
}
```

### File: `broker_wasm_test.go`

```go
//go:build wasm

package crudp_test

import "testing"

func TestBroker_WASM(t *testing.T) {
    t.Run("Consolidation", func(t *testing.T) {
        BrokerConsolidationShared(t)
    })
    
    t.Run("Flush", func(t *testing.T) {
        BrokerFlushShared(t)
    })
    
    t.Run("Clear", func(t *testing.T) {
        BrokerClearShared(t)
    })
    
    t.Run("EnqueuePacket", func(t *testing.T) {
        EnqueuePacketShared(t)
    })
}
```

---

## 4.5 Verification

```bash
# Stdlib tests
go test -v -run TestBroker

# WASM tests
GOOS=js GOARCH=wasm go test -v -tags wasm -run TestBroker
```

---

## Notes

- **tinytime.Timer:** Uses `AfterFunc` from tinytime that works in WASM
- **Consolidation:** Packets with same `HandlerID+Action` are grouped into one
- **Thread-safe:** Uses `sync.Mutex` for concurrent access (in backend)
- **Pre-allocation:** `queue` is initialized with capacity 16 to reduce allocations

---

> **Next step:** [STEP_05_UPDATE_EXISTING.md](./STEP_05_UPDATE_EXISTING.md)
