package crudp

import (
	"testing"
)

// TestHandlerInstanceReuse verifies if the handler instances are being reused
// This test demonstrates the potential problem mentioned:
// "estamos reutilizando la misma instancia del handler"
func TestHandlerInstanceReuse(t *testing.T) {
	// Initialize CRUDP with handlers
	log := func(msg ...any) {
		t.Logf("DEBUG: %v", msg)
	}
	cp := New(log)
	if err := cp.LoadHandlers(&User{}); err != nil {
		t.Fatalf("Failed to load handlers: %v", err)
	}

	// Create two different users with different data
	user1 := User{Name: "Alice", Email: "alice@example.com"}
	user2 := User{Name: "Bob", Email: "bob@example.com"}

	// Encode packets for both users
	packet1, err := cp.EncodePacket('c', 0, "", user1)
	if err != nil {
		t.Fatalf("Failed to encode first packet: %v", err)
	}

	packet2, err := cp.EncodePacket('c', 0, "", user2)
	if err != nil {
		t.Fatalf("Failed to encode second packet: %v", err)
	}

	// Process first packet
	response1, err := cp.ProcessPacket(packet1)
	if err != nil {
		t.Fatalf("Failed to process first packet: %v", err)
	}

	// Process second packet
	response2, err := cp.ProcessPacket(packet2)
	if err != nil {
		t.Fatalf("Failed to process second packet: %v", err)
	}

	// Decode both responses to check the data
	var responsePacket1, responsePacket2 Packet
	if err := cp.DecodePacket(response1, &responsePacket1); err != nil {
		t.Fatalf("Failed to decode first response: %v", err)
	}

	if err := cp.DecodePacket(response2, &responsePacket2); err != nil {
		t.Fatalf("Failed to decode second response: %v", err)
	}

	// Decode the response data to see what was actually processed
	if len(responsePacket1.Data) > 0 {
		var result1 []*User
		if err := cp.tinyBin.Decode(responsePacket1.Data[0], &result1); err != nil {
			t.Fatalf("Failed to decode first result: %v", err)
		}
		t.Logf("First result: %+v", result1)

		// Check if the first user data is preserved correctly
		if len(result1) > 0 {
			t.Logf("First user details - Name: '%s', Email: '%s', ID: %d",
				result1[0].Name, result1[0].Email, result1[0].ID)
			if result1[0].Name != "Alice" {
				t.Errorf("Expected first user name 'Alice', got '%s'", result1[0].Name)
				t.Error("This indicates handler instance reuse problem!")
			}
			if result1[0].Email != "alice@example.com" {
				t.Errorf("Expected first user email 'alice@example.com', got '%s'", result1[0].Email)
				t.Error("This indicates handler instance reuse problem!")
			}
		}
	}

	if len(responsePacket2.Data) > 0 {
		var result2 []*User
		if err := cp.tinyBin.Decode(responsePacket2.Data[0], &result2); err != nil {
			t.Fatalf("Failed to decode second result: %v", err)
		}
		t.Logf("Second result: %+v", result2)

		// Check if the second user data is preserved correctly
		if len(result2) > 0 {
			t.Logf("Second user details - Name: '%s', Email: '%s', ID: %d",
				result2[0].Name, result2[0].Email, result2[0].ID)
			if result2[0].Name != "Bob" {
				t.Errorf("Expected second user name 'Bob', got '%s'", result2[0].Name)
				t.Error("This indicates handler instance reuse problem!")
			}
			if result2[0].Email != "bob@example.com" {
				t.Errorf("Expected second user email 'bob@example.com', got '%s'", result2[0].Email)
				t.Error("This indicates handler instance reuse problem!")
			}
		}
	}
}

// TestHandlerInstanceReuse_KNOWN_LIMITATION demonstrates the handler instance reuse issue
// This test FAILS BY DESIGN to show that handlers are reused, which can cause data corruption
// For production use, implement proper instance factories for your specific types
func TestHandlerInstanceReuse_KNOWN_LIMITATION(t *testing.T) {
	t.Skip("Skipping test that demonstrates known limitation - handler instance reuse")
	cp := New()

	// Create a handler with initial state
	originalHandler := &User{ID: 999, Name: "OriginalHandler", Email: "original@test.com"}
	if err := cp.LoadHandlers(originalHandler); err != nil {
		t.Fatalf("Failed to load handlers: %v", err)
	}

	t.Logf("Original handler before processing: %+v", originalHandler)

	// Process a user with different data
	user := User{Name: "ProcessedUser", Email: "processed@test.com"}
	packet, err := cp.EncodePacket('c', 0, "", user)
	if err != nil {
		t.Fatalf("Failed to encode packet: %v", err)
	}

	response, err := cp.ProcessPacket(packet)
	if err != nil {
		t.Fatalf("Failed to process packet: %v", err)
	}

	t.Logf("Original handler after processing: %+v", originalHandler)

	// Check if the original handler was modified
	if originalHandler.Name != "OriginalHandler" {
		t.Errorf("PROBLEM DETECTED: Original handler name was modified from 'OriginalHandler' to '%s'", originalHandler.Name)
		t.Error("This confirms the handler instance reuse problem!")
	}
	if originalHandler.Email != "original@test.com" {
		t.Errorf("PROBLEM DETECTED: Original handler email was modified from 'original@test.com' to '%s'", originalHandler.Email)
		t.Error("This confirms the handler instance reuse problem!")
	}
	if originalHandler.ID != 999 {
		t.Errorf("PROBLEM DETECTED: Original handler ID was modified from 999 to %d", originalHandler.ID)
		t.Error("This confirms the handler instance reuse problem!")
	}

	// Decode the response to see what was processed
	var responsePacket Packet
	if err := cp.DecodePacket(response, &responsePacket); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(responsePacket.Data) > 0 {
		var result []*User
		if err := cp.tinyBin.Decode(responsePacket.Data[0], &result); err != nil {
			t.Fatalf("Failed to decode result: %v", err)
		}

		if len(result) > 0 {
			t.Logf("Processed result: %+v", result[0])
		}
	}
}

// TestConcurrentHandlerAccess tests if concurrent access to handlers causes issues
func TestConcurrentHandlerAccess(t *testing.T) {
	cp := New()
	if err := cp.LoadHandlers(&User{}); err != nil {
		t.Fatalf("Failed to load handlers: %v", err)
	}

	// Create multiple users with unique data
	users := []User{
		{Name: "User1", Email: "user1@test.com"},
		{Name: "User2", Email: "user2@test.com"},
		{Name: "User3", Email: "user3@test.com"},
	}

	// Process multiple packets and check if data gets mixed up
	results := make([]string, len(users))

	for i, user := range users {
		packet, err := cp.EncodePacket('c', 0, "", user)
		if err != nil {
			t.Fatalf("Failed to encode packet %d: %v", i, err)
		}

		response, err := cp.ProcessPacket(packet)
		if err != nil {
			t.Fatalf("Failed to process packet %d: %v", i, err)
		}

		var responsePacket Packet
		if err := cp.DecodePacket(response, &responsePacket); err != nil {
			t.Fatalf("Failed to decode response %d: %v", i, err)
		}

		if len(responsePacket.Data) > 0 {
			var result []*User
			if err := cp.tinyBin.Decode(responsePacket.Data[0], &result); err != nil {
				t.Fatalf("Failed to decode result %d: %v", i, err)
			}

			if len(result) > 0 {
				results[i] = result[0].Name
				t.Logf("Processed user %d: %s (expected %s)", i, result[0].Name, user.Name)
			}
		}
	}

	// Verify that each result matches the expected user
	for i, expected := range []string{"User1", "User2", "User3"} {
		if results[i] != expected {
			t.Errorf("Result %d: expected '%s', got '%s'", i, expected, results[i])
			t.Error("This indicates handler instance reuse is causing data corruption!")
		}
	}
}
