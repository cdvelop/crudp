# Initial Vision of the CRUDP Protocol

## Introduction

CRUDP is the core that registers and manages handlers for CRUD operations. Instead of limited manual registration, CRUDP provides a centralized API for modules to register their handlers dynamically.

CRUDP simplifies the creation of typed APIs, allowing both the client (`client.go`) and server (`server.go`) to know the handlers for consistency. Handlers can process batches of requests.

## Current Problem

Currently, traditional APIs process one request at a time, which limits efficiency in offline-first scenarios. For example, in the frontend (using Go WebAssembly/TinyGo), if there's a connection failure, the user can continue working and storing objects locally. When the connection returns, I need to send multiple operations in a single request to synchronize everything.

Previously, I used WebSockets with an internal local queue, but now I need it to be public and scalable. That's why I thought of using Server-Sent Events (SSE): each request is processed asynchronously, stored in a queue, and as they are processed, responses are sent using CRUDP on both frontend and backend. This makes the code reusable and testable.

## Proposed Solution

### Core Architecture

The radical change here is that the `Handler` no longer touches HTTP. The Router acts as a "translator" that extracts context from HTTP (Auth, UserID) and passes it to the Handler in an agnostic way.

- **Centralized Registration in CRUDP:** CRUDP provides the API to register handlers dynamically from modules, without direct coupling to HTTP.

- **Batch Processing:** Handlers receive N requests in a single call, optimizing performance.

- **Asynchrony with SSE:** Asynchronous notifications for results, using correlation IDs.

- **Efficient Binary Protocol:** Typed communication reusable on client and server.

### Clean Architecture with Context

To achieve clean architecture, the business **Handler** should not know that a package called `crudp` exists, nor should it depend on proprietary transport structures. We use Go's standard `context.Context` to pass necessary information like UserID and cancellation signals.

See [interfaces.go](../interfaces.go) for the complete interface definitions.


## Protocol Requirements

- Must be TinyGo-friendly (resource-efficient).

- Support for batch operations.

- Correlation IDs to track asynchronous responses.

- Transport decoupling (no direct dependency on HTTP in handlers).

