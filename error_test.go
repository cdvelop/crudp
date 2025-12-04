package crudp_test

import (
	"context"
	"testing"

	"github.com/cdvelop/crudp"
)

func TestCrudP_ErrorHandling(t *testing.T) {
	cp := crudp.NewDefault()

	// Test with invalid packet
	invalidPacket := []byte("invalid")
	_, err := cp.ProcessPacket(context.Background(), invalidPacket)
	if err == nil {
		t.Error("Expected error for invalid packet")
	}

	// Test with non-existent handler
	invalidHandlerPacket, err := cp.EncodePacket('c', 99, "", &User{Name: "Test"})
	if err != nil {
		t.Fatalf("Failed to encode packet: %v", err)
	}

	_, err = cp.ProcessPacket(context.Background(), invalidHandlerPacket)
	if err == nil {
		t.Error("Expected error for non-existent handler")
	}
}
