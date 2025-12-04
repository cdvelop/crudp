package crudp

import (
	"context"
	"testing"

	. "github.com/cdvelop/tinystring"
)

type User struct {
	ID    int
	Name  string
	Email string
}

func (u *User) Create(ctx context.Context, data ...any) any {
	created := make([]*User, 0, len(data))
	for _, item := range data {
		// item is concrete type (User), cast directly
		user := item.(*User)
		user.ID = 123
		created = append(created, user)
	}
	return created
}

func (u *User) Read(ctx context.Context, data ...any) any {
	results := make([]*User, 0, len(data))
	for _, item := range data {
		// item is concrete type (User), cast directly
		user := item.(*User)
		results = append(results, &User{ID: user.ID, Name: "Found " + user.Name, Email: user.Email})
	}
	return results
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
	userData, err := cp.tinyBin.Encode(&User{Name: "John", Email: "john@example.com"})
	if err != nil {
		t.Fatalf("Failed to encode user data: %v", err)
	}

	createPacket := Packet{
		Action:    'c',
		HandlerID: 0,
		ReqID:     "test-create",
		Data:      [][]byte{userData},
	}

	batchReq := BatchRequest{Packets: []Packet{createPacket}}
	batchBytes, err := cp.tinyBin.Encode(batchReq)
	if err != nil {
		t.Fatalf("Failed to encode batch request: %v", err)
	}

	response, err := cp.ProcessBatch(context.Background(), batchBytes)
	if err != nil {
		t.Fatalf("Failed to process batch: %v", err)
	}

	var batchResp BatchResponse
	if err := cp.tinyBin.Decode(response, &batchResp); err != nil {
		t.Fatalf("Failed to decode batch response: %v", err)
	}

	if len(batchResp.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(batchResp.Results))
	}

	result := batchResp.Results[0]
	if result.ReqID != "test-create" {
		t.Errorf("Expected ReqID 'test-create', got '%s'", result.ReqID)
	}

	if result.MessageType != uint8(Msg.Success) {
		t.Errorf("Expected success, got failure: %s", result.Message)
	}

	var createdUser []*User
	if err := cp.tinyBin.Decode(result.Data[0], &createdUser); err != nil {
		t.Fatalf("Failed to decode created user: %v", err)
	}
	if len(createdUser) != 1 {
		t.Fatalf("Expected 1 created user, got %d", len(createdUser))
	}
	if createdUser[0].ID != 123 {
		t.Errorf("Expected created user ID 123, got %d", createdUser[0].ID)
	}

	// Test Read operation
	readUserData, err := cp.tinyBin.Encode(&User{ID: 123, Name: "John"})
	if err != nil {
		t.Fatalf("Failed to encode read user data: %v", err)
	}

	readPacket := Packet{
		Action:    'r',
		HandlerID: 0,
		ReqID:     "test-read",
		Data:      [][]byte{readUserData},
	}

	batchReq2 := BatchRequest{Packets: []Packet{readPacket}}
	batchBytes2, err := cp.tinyBin.Encode(batchReq2)
	if err != nil {
		t.Fatalf("Failed to encode batch request 2: %v", err)
	}

	response2, err := cp.ProcessBatch(context.Background(), batchBytes2)
	if err != nil {
		t.Fatalf("Failed to process read batch: %v", err)
	}

	var batchResp2 BatchResponse
	if err := cp.tinyBin.Decode(response2, &batchResp2); err != nil {
		t.Fatalf("Failed to decode batch response 2: %v", err)
	}

	if len(batchResp2.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(batchResp2.Results))
	}

	result2 := batchResp2.Results[0]
	if result2.ReqID != "test-read" {
		t.Errorf("Expected ReqID 'test-read', got '%s'", result2.ReqID)
	}

	if result2.MessageType != uint8(Msg.Success) {
		t.Errorf("Expected success, got failure: %s", result2.Message)
	}

	var readUser []*User
	if err := cp.tinyBin.Decode(result2.Data[0], &readUser); err != nil {
		t.Fatalf("Failed to decode read user: %v", err)
	}
	if len(readUser) != 1 {
		t.Fatalf("Expected 1 read user, got %d", len(readUser))
	}
	if readUser[0].Name != "Found John" {
		t.Errorf("Expected read user name 'Found John', got '%s'", readUser[0].Name)
	}
}
