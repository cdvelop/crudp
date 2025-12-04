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
