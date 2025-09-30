# crudp

A simple binary CRUD protocol for Go structs with deterministic, shared handler registration.


## Main Features

CRUDP follows the minimalist philosophy with:

- 🏆 **Ultra-small binaries** - Zero extra dependencies
- ✅ **TinyGo compatibility** - No compilation issues  
- 🎯 **Predictable performance** - No hidden allocations
- 🔧 **Minimal API** - Only essential operations
- 🔍 **Deterministic identification** - Shared indexes guarantee the same handler on client and server
- 💪 **Strong typing** - Direct Go structures, no maps
- ⚡ **Efficiency** - Compact IDs (`uint8`) and pre-allocated table in slice, no dynamic maps


## Documentation

- [Implementation Details](docs/IMPLEMENTATION.md)
- [Usage Examples](docs/USAGE_EXAMPLE.md)


### Compatible Data Types (Minimalist Approach)

CRUDP **intentionally** supports only a minimal set of types to keep the binary size small:

**✅ Supported Types:**
- **Basic types**: `string`, `bool`
- **All numeric types**: `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`
- **All basic slices**: `[]string`, `[]bool`, `[]byte`, `[]int`, `[]int8`, `[]int16`, `[]int32`, `[]int64`, `[]uint`, `[]uint8`, `[]uint16`, `[]uint32`, `[]uint64`, `[]float32`, `[]float64`
- **Structs**: Only with supported field types
- **Slices of structs**: `[]struct{...}` where all fields are supported types
- **Maps**: `map[K]V` where K and V are only supported types
- **Slices of maps**: `[]map[K]V` where K and V are supported types
- **Pointers**: Only to the supported types above

**❌ Unsupported Types:**
- `any`, `chan`, `func`
- `complex64`, `complex128`
- `uintptr`, `unsafe.Pointer` (used only internally)
- Arrays (different from slices)
- Complex nested types beyond the supported scope

This focused approach ensures minimal code size while covering the most common data transfer operations including simple structs.

## Automatic Handlers System

### ✅ Design Advantages

- **🎯 Unified registration** - `LoadHandlers()` receives the real implementations that act as prototypes and handlers
- **🔧 Flexible interfaces** - Implement only Create, Read, Update or Delete as needed
- **⚡ Efficient processing** - `ProcessPacket` handles everything automatically
- **🛡️ Robust error handling** - Errors are automatically converted to responses
- **🏆 Testable** - `New()` constructor allows isolated testing
- **🔄 Shared instance** - The same `modules.Protocol` instance is used on client and server
- **💪 Zero duplication** - Structures are both prototypes and handlers, minimizing code

### ⚠️ Considerations

- **Mandatory shared instance** - Client and server must import `modules.Protocol` for indexes to match
- **Manual casting required** - Handlers must do `item.(*Type)`; type safety is under your control
- **Deterministic IDs** - `LoadHandlers()` assigns indexes by order; maintain consistency between builds

### 🎯 Decoupled Handlers (Return `any`)

- **🔧 No tinybin dependency** - Handlers don't need to import or know about tinybin
- **⚡ Less work** - Just return Go structures, CRUDP encodes automatically
- **🧪 Easy testing** - Handlers are tested independently without CRUDP
- **📦 Natural API** - `return users, nil` instead of `return tinybin.Encode(users)`
- **🔄 Flexibility** - If special control is needed, can return `[]byte` directly

### ⚡ TinyGo/WebAssembly Optimization

- **🏗️ Compact slices** - No maps; the table is sized once with shared prototypes
- **🎯 Direct calls** - `callHandler()` avoids function variable allocations
- **💾 Predictable memory** - Length controlled by `modules.AllHandlers`
- **🔍 O(n) search** - Efficient for 5-15 typical types in WebAssembly
- **✅ TinyGo compatible** - No problematic dynamic map features

## Why shared indexes instead of `StructID`

Abandoning `StructID` simplifies the architecture when you control registration in a single place.

### Advantages

- **Total symmetry** – The same registration slice compiles into the WASM binary and the native server.
- **Automatic indexes** – `uint8` values are assigned by declaration order, no need for manual constants.
- **No reflection** – No need for `tinyreflect`; compatible with TinyGo and restricted builds.
- **Fast error detection** – Any desynchronization is caught in tests that compare the shared table.

### Disadvantages and Mitigations

- **Order maintenance** – Changing the order in `Setup()` changes the automatic IDs. Keep registration in a single versioned package.
- **Explicit IDs** – Use direct numeric indexes (0, 1, 2...) in client code according to declaration order.
- **Coordinated migrations** – Client and server must update together when adding or reordering entries.

### Architecture

- **`tinybin`**: Compact binary encoding/decoding
- **`crudp`**: CRUD protocol logic, handler slices and error handling
- **`modules/modules.go`**: Single source of truth for shared `Protocol` and exported IDs
- **`modules/{module}/`**: Real implementations organized by business functionality






---
## [Contributing](https://github.com/cdvelop/cdvelop/blob/main/CONTRIBUTING.md)