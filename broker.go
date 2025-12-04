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
