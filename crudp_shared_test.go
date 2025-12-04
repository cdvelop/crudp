package crudp_test

import (
	"context"
	"testing"

	"github.com/cdvelop/crudp"
	. "github.com/cdvelop/tinystring"
)

type User struct {
	ID    int
	Name  string
	Email string
}

func (u *User) Response() (any, []string, error) {
	return u, nil, nil
}

func (u *User) Create(ctx context.Context, data ...any) any {
	created := make([]crudp.Response, 0, len(data))
	for _, item := range data {
		// item is concrete type (User), cast directly
		user := item.(*User)
		user.ID = 123
		created = append(created, user)
	}
	return created
}

func (u *User) Read(ctx context.Context, data ...any) any {
	results := make([]crudp.Response, 0, len(data))
	for _, item := range data {
		// item is concrete type (User), cast directly
		user := item.(*User)
		results = append(results, &User{ID: user.ID, Name: "Found " + user.Name, Email: user.Email})
	}
	return results
}

func CrudPBasicFunctionalityShared(t *testing.T) {
	// Initialize CRUDP with handlers
	cp := crudp.NewDefault()
	if err := cp.RegisterHandler(&User{}); err != nil {
		t.Fatalf("Failed to load handlers: %v", err)
	}

	// Test Create operation
	userData, err := cp.Codec().Encode(&User{Name: "John", Email: "john@example.com"})
	if err != nil {
		t.Fatalf("Failed to encode user data: %v", err)
	}

	createPacket := crudp.Packet{
		Action:    'c',
		HandlerID: 0,
		ReqID:     "test-create",
		Data:      [][]byte{userData},
	}

	batchReq := crudp.BatchRequest{Packets: []crudp.Packet{createPacket}}
	batchBytes, err := cp.Codec().Encode(batchReq)
	if err != nil {
		t.Fatalf("Failed to encode batch request: %v", err)
	}

	response, err := cp.ProcessBatch(context.Background(), batchBytes)
	if err != nil {
		t.Fatalf("Failed to process batch: %v", err)
	}

	var batchResp crudp.BatchResponse
	if err := cp.Codec().Decode(response, &batchResp); err != nil {
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

	// Handler returns []Response, so each element in Data is a separate User object
	if len(result.Data) != 1 {
		t.Fatalf("Expected 1 data element, got %d", len(result.Data))
	}

	var createdUser User
	if err := cp.Codec().Decode(result.Data[0], &createdUser); err != nil {
		t.Fatalf("Failed to decode created user: %v", err)
	}
	if createdUser.ID != 123 {
		t.Errorf("Expected created user ID 123, got %d", createdUser.ID)
	}

	// Test Read operation
	readUserData, err := cp.Codec().Encode(&User{ID: 123, Name: "John"})
	if err != nil {
		t.Fatalf("Failed to encode read user data: %v", err)
	}

	readPacket := crudp.Packet{
		Action:    'r',
		HandlerID: 0,
		ReqID:     "test-read",
		Data:      [][]byte{readUserData},
	}

	batchReq2 := crudp.BatchRequest{Packets: []crudp.Packet{readPacket}}
	batchBytes2, err := cp.Codec().Encode(batchReq2)
	if err != nil {
		t.Fatalf("Failed to encode batch request 2: %v", err)
	}

	response2, err := cp.ProcessBatch(context.Background(), batchBytes2)
	if err != nil {
		t.Fatalf("Failed to process read batch: %v", err)
	}

	var batchResp2 crudp.BatchResponse
	if err := cp.Codec().Decode(response2, &batchResp2); err != nil {
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

	// Handler returns []Response, so each element in Data is a separate User object
	if len(result2.Data) != 1 {
		t.Fatalf("Expected 1 data element, got %d", len(result2.Data))
	}

	var readUser User
	if err := cp.Codec().Decode(result2.Data[0], &readUser); err != nil {
		t.Fatalf("Failed to decode read user: %v", err)
	}
	if readUser.Name != "Found John" {
		t.Errorf("Expected read user name 'Found John', got '%s'", readUser.Name)
	}
}
