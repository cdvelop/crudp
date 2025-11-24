package crudp

import (
	"context"
	"testing"
)

type User struct {
	ID    int
	Name  string
	Email string
}

func (u *User) Create(ctx context.Context, data ...any) (any, error) {
	created := make([]*User, 0, len(data))
	for _, item := range data {
		// item is concrete type (User), cast directly
		user := item.(*User)
		user.ID = 123
		created = append(created, user)
	}
	return created, nil
}

func (u *User) Read(ctx context.Context, data ...any) (any, error) {
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
	log := func(msg ...any) {
		// Removed debug logs for cleaner output
	}
	cp := New(log)
	if err := cp.RegisterHandler(&User{}); err != nil {
		t.Fatalf("Failed to load handlers: %v", err)
	}

	// Test Create operation
	createPacket, err := cp.EncodePacket('c', 0, "", &User{Name: "John", Email: "john@example.com"})
	if err != nil {
		t.Fatalf("Failed to encode create packet: %v", err)
	}

	response, err := cp.ProcessPacket(context.Background(), createPacket)
	if err != nil {
		t.Fatalf("Failed to process create packet: %v", err)
	}

	// Decode response
	var responsePacket Packet
	if err := cp.DecodePacket(response, &responsePacket); err != nil {
		t.Fatalf("Failed to decode response packet: %v", err)
	}

	if responsePacket.Action != 'c' {
		t.Errorf("Expected action 'c', got '%c'", responsePacket.Action)
	}

	if responsePacket.Message != "success" {
		t.Errorf("Expected success message, got '%s'", responsePacket.Message)
	}

	// Test Read operation
	readPacket, err := cp.EncodePacket('r', 0, "", &User{ID: 123})
	if err != nil {
		t.Fatalf("Failed to encode read packet: %v", err)
	}

	response, err = cp.ProcessPacket(context.Background(), readPacket)
	if err != nil {
		t.Fatalf("Failed to process read packet: %v", err)
	}

	// Decode response
	if err := cp.DecodePacket(response, &responsePacket); err != nil {
		t.Fatalf("Failed to decode response packet: %v", err)
	}

	if responsePacket.Action != 'r' {
		t.Errorf("Expected action 'r', got '%c'", responsePacket.Action)
	}
}
