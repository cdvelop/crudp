#!/bin/bash

echo "=========================================="
echo "Running Stdlib Tests..."
echo "=========================================="
go test -v ./...

if [ $? -ne 0 ]; then
    echo "❌ Stdlib tests failed"
    exit 1
fi

echo ""
echo "=========================================="
echo "Running WASM Tests..."
echo "=========================================="

# Check if wasmbrowsertest is installed
if ! command -v wasmbrowsertest &> /dev/null; then
    echo "⚠️  wasmbrowsertest not found. Install it with:"
    echo "   go install github.com/agnivade/wasmbrowsertest@latest"
    echo "   export PATH=\$PATH:\$(go env GOPATH)/bin"
    exit 1
fi

# Run WASM tests
GOOS=js GOARCH=wasm go test -v -tags wasm
WASM_EXIT_CODE=$?

if [ $WASM_EXIT_CODE -ne 0 ]; then
    echo ""
    echo "❌ WASM tests failed"
    exit 1
fi

echo ""
echo "✅ All tests passed!"
