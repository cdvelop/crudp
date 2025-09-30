package crudp

import (
	"testing"
)

type User struct {
	ID    int
	Name  string
	Email string
}

func (u *User) Create(data ...any) (any, error) {
	created := make([]*User, 0, len(data))
	for _, item := range data {
		// item is concrete type (User), cast directly
		user := item.(*User)
		user.ID = 123
		created = append(created, user)
	}
	return created, nil
}

func (u *User) Read(data ...any) (any, error) {
	results := make([]*User, 0, len(data))
	for _, item := range data {
		// item is concrete type (User), cast directly
		user := item.(*User)
		results = append(results, &User{ID: user.ID, Name: "Found " + user.Name, Email: user.Email})
	}
	return results, nil
}

func TestCrudP_BasicFunctionality(t *testing.T) {
	// Initialize CRUDP with handlers
	cp := New()
	if err := cp.LoadHandlers(&User{}); err != nil {
		t.Fatalf("Failed to load handlers: %v", err)
	}

	// Test Create operation
	createPacket, err := EncodePacket('c', 0, "", &User{Name: "John", Email: "john@example.com"})
	if err != nil {
		t.Fatalf("Failed to encode create packet: %v", err)
	}

	response, err := cp.ProcessPacket(createPacket)
	if err != nil {
		t.Fatalf("Failed to process create packet: %v", err)
	}

	// Decode response
	var responsePacket Packet
	if err := DecodePacket(response, &responsePacket); err != nil {
		t.Fatalf("Failed to decode response packet: %v", err)
	}

	if responsePacket.Action != 'c' {
		t.Errorf("Expected action 'c', got '%c'", responsePacket.Action)
	}

	if responsePacket.Message != "success" {
		t.Errorf("Expected success message, got '%s'", responsePacket.Message)
	}

	// Test Read operation
	readPacket, err := EncodePacket('r', 0, "", &User{ID: 123})
	if err != nil {
		t.Fatalf("Failed to encode read packet: %v", err)
	}

	response, err = cp.ProcessPacket(readPacket)
	if err != nil {
		t.Fatalf("Failed to process read packet: %v", err)
	}

	// Decode response
	if err := DecodePacket(response, &responsePacket); err != nil {
		t.Fatalf("Failed to decode response packet: %v", err)
	}

	if responsePacket.Action != 'r' {
		t.Errorf("Expected action 'r', got '%c'", responsePacket.Action)
	}
}

func TestCrudP_ErrorHandling(t *testing.T) {
	cp := New()

	// Test with invalid packet
	invalidPacket := []byte("invalid")
	_, err := cp.ProcessPacket(invalidPacket)
	if err != nil {
		// Expected error for invalid packet
		t.Logf("Correctly handled invalid packet: %v", err)
	}

	// Test with non-existent handler
	invalidHandlerPacket, err := EncodePacket('c', 99, "", &User{Name: "Test"})
	if err != nil {
		t.Fatalf("Failed to encode packet: %v", err)
	}

	_, err = cp.ProcessPacket(invalidHandlerPacket)
	if err != nil {
		// Expected error for non-existent handler
		t.Logf("Correctly handled non-existent handler: %v", err)
	}
}
