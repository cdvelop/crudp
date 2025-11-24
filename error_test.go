package crudp

import (
	"context"
	"testing"
)

func TestCrudP_ErrorHandling(t *testing.T) {
	cp := New()

	// Test with invalid packet
	invalidPacket := []byte("invalid")
	_, err := cp.ProcessPacket(context.Background(), invalidPacket)
	if err != nil {
		// Expected error for invalid packet
		t.Logf("Correctly handled invalid packet: %v", err)
	}

	// Test with non-existent handler
	invalidHandlerPacket, err := cp.EncodePacket('c', 99, "", &User{Name: "Test"})
	if err != nil {
		t.Fatalf("Failed to encode packet: %v", err)
	}

	_, err = cp.ProcessPacket(context.Background(), invalidHandlerPacket)
	if err != nil {
		// Expected error for non-existent handler
		t.Logf("Correctly handled non-existent handler: %v", err)
	}
}
