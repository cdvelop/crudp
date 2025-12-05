# Packet Structure

## Overview

CRUDP uses a packet-based system for communication between the client and server. There are two main packet types: `Packet` for requests and `PacketResult` for responses.

## The `Packet` Struct

The `Packet` struct represents a single request.

```go
type Packet struct {
    Action    byte
    HandlerID uint8
    ReqID     string
    Data      [][]byte
}
```

-   `Action`: The CRUD action to perform (`c`, `r`, `u`, `d`).
-   `HandlerID`: The ID of the handler to process the request.
-   `ReqID`: A unique ID for the request.
-   `Data`: The data for the request, encoded as a slice of byte slices.

## The `PacketResult` Struct

The `PacketResult` struct represents the result of a request.

```go
type PacketResult struct {
    Packet
    MessageType uint8
    Message     string
}
```

-   `Packet`: The original `Packet` is embedded in the result.
-   `MessageType`: A `uint8` indicating the type of the message (e.g., success, error, info). This uses the `MessageType` values from the `tinystring` library.
-   `Message`: A human-readable message.

## Batching

CRUDP supports batching of requests and responses. A `BatchRequest` is a slice of `Packet`s, and a `BatchResponse` is a slice of `PacketResult`s.

```go
type BatchRequest struct {
    Packets []Packet
}

type BatchResponse struct {
    Results []PacketResult
}
```
