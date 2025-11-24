# CRUDP Limitations & Supported Data Types

## Compatible Data Types (Minimalist Approach)

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
- `time.Time`
- `any`, `chan`, `func`
- `complex64`, `complex128`
- `uintptr`, `unsafe.Pointer` (used only internally)
- Arrays (different from slices)
- Complex nested types beyond the supported scope

This focused approach ensures minimal code size while covering the most common data transfer operations including simple structs.