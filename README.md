# CRUDP

 Is a binary CRUD protocol supporting the four basic operations (Create, Read, Update, Delete) that evolves from a simple synchronous system to a batched asynchronous one, using HTTP as a tunnel for binary data and SSE for reactive responses. It supports batch/asynchronous processing and is designed for Local-First / Offline-First apps, enabling offline synchronization, batch processing, and decoupling between client and server while maintaining binary efficiency and TinyGo compatibility.

## Documentation

- [`docs/INITIAL_VISION.md`](docs/INITIAL_VISION.md): Initial vision and protocol requirements.
- [`docs/SSE.md`](docs/SSE.md): Server-Sent Events implementation and broker.
- [`docs/FILE_UPLOAD.md`](docs/FILE_UPLOAD.md): Manejo de archivos, enfoque "Upload & Reference".
- [`docs/HANDLER_REGISTER.md`](docs/HANDLER_REGISTER.md): Hybrid handler registration system and router design.
- [`docs/INTEGRATION_GUIDE.md`](docs/INTEGRATION_GUIDE.md): How to integrate CRUDP into a project (server + WASM client).
- [`docs/OPTIMIZATION_PLAN.md`](docs/OPTIMIZATION_PLAN.md): Planned and existing performance optimizations.
- [`docs/PACKAGE_STRUCTURE.md`](docs/PACKAGE_STRUCTURE.md): Package layout, TinyGo-friendly structure.
- [`docs/PERFORMANCE.md`](docs/PERFORMANCE.md): Benchmark and profiling results.
- [`docs/ROUTER_IMPLEMENTATION.md`](docs/ROUTER_IMPLEMENTATION.md): Router implementation details (server side).
- [`docs/LIMITATIONS.md`](docs/LIMITATIONS.md): Supported data types and limitations.



---
## [Contributing](https://github.com/cdvelop/cdvelop/blob/main/CONTRIBUTING.md)