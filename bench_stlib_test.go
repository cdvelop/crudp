//go:build !wasm

package crudp

import (
	"testing"
)

func BenchmarkCrudP_Setup(b *testing.B) {
	BenchmarkCrudPSetupShared(b)
}

func BenchmarkCrudP_EncodePacket(b *testing.B) {
	BenchmarkCrudPEncodePacketShared(b)
}

func BenchmarkCrudP_ProcessPacket(b *testing.B) {
	BenchmarkCrudPProcessPacketShared(b)
}

func BenchmarkCrudP_FullCycle(b *testing.B) {
	BenchmarkCrudPFullCycleShared(b)
}

func BenchmarkCrudP_MultipleUsers(b *testing.B) {
	BenchmarkCrudPMultipleUsersShared(b)
}

func BenchmarkCrudP_AllOperations(b *testing.B) {
	BenchmarkCrudPAllOperationsShared(b)
}

func BenchmarkCrudP_LargePayload(b *testing.B) {
	BenchmarkCrudPLargePayloadShared(b)
}
