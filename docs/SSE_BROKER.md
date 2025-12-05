# SSE Broker and Packet Batching

## Overview

CRUDP includes a broker that handles the batching and consolidation of outgoing packets. This is particularly useful in client-side (WASM) applications to minimize the number of HTTP requests sent to the server.

## The Broker

The broker is an internal component of CRUDP that is automatically initialized when you create a new `CrudP` instance. It is responsible for:

-   **Queueing:** Packets are added to a queue instead of being sent immediately.
-   **Consolidation:** Packets with the same handler and action are merged into a single packet.
-   **Batching:** The broker waits for a configurable amount of time (the "batch window") to collect multiple packets before sending them as a single batch.
-   **Flushing:** When the batch window timer expires, the broker "flushes" the queue, sending all the packets in a single request.

## How it Works

1.  When you call a method that sends a packet (e.g., `EnqueuePacket`), the packet is added to the broker's queue.
2.  If a packet with the same handler and action already exists in the queue, the new packet's data is appended to the existing packet.
3.  A timer is started (or reset) for the duration of the `BatchWindow` (configured in the `Config` struct).
4.  When the timer expires, the broker sends all the packets in the queue as a single batch request.

## Configuration

The broker's behavior is controlled by the `BatchWindow` setting in the `Config` struct.

```go
// Config contains CrudP configuration
type Config struct {
    // ...

    // BatchWindow in milliseconds. Default: 50
    BatchWindow int
    
    // ...
}
```

## `tinytime` Dependency

The broker uses the `tinytime` library for its timer, which is compatible with WebAssembly.

## Manual Flushing

You can manually flush the broker's queue at any time by calling the `FlushNow()` method on the broker.

```go
// Get the broker from the CrudP instance
broker := cp.Broker()

// Force an immediate flush
broker.FlushNow()
```
