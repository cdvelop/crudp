package crudp

import (
	"context"
	"testing"
)

// BenchUser is a simple structure for benchmarks
type BenchUser struct {
	ID    uint32
	Name  string
	Email string
	Age   uint8
}

func (u *BenchUser) Create(data ...any) []any {
	created := make([]*BenchUser, 0, len(data))
	for _, item := range data {
		user := item.(*BenchUser)
		user.ID = 123
		created = append(created, user)
	}
	return []any{created}
}

func (u *BenchUser) Read(data ...any) (any, error) {
	results := make([]*BenchUser, 0, len(data))
	for _, item := range data {
		user := item.(*BenchUser)
		results = append(results, &BenchUser{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Age:   user.Age,
		})
	}
	return results, nil
}

func (u *BenchUser) Update(data ...any) (any, error) {
	updated := make([]*BenchUser, 0, len(data))
	for _, item := range data {
		user := item.(*BenchUser)
		user.Name = "Updated " + user.Name
		updated = append(updated, user)
	}
	return updated, nil
}

func (u *BenchUser) Delete(data ...any) (any, error) {
	return len(data), nil // Return count of deleted items
}

// Global variables to prevent compiler optimizations
var (
	globalCrudP    *CrudP
	globalPacket   []byte
	globalResponse []byte
	globalUser     = &BenchUser{
		ID:    1,
		Name:  "BenchUser",
		Email: "bench@example.com",
		Age:   25,
	}
)

// BenchmarkCrudP_Setup measures allocations for CRUDP initialization
func BenchmarkCrudP_Setup(b *testing.B) {
	var cp *CrudP

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cp = New()
		if err := cp.RegisterHandler(&BenchUser{}); err != nil {
			b.Fatalf("RegisterHandler failed: %v", err)
		}
	}

	globalCrudP = cp // Prevent optimization
}

// BenchmarkCrudP_EncodePacket measures allocations for packet encoding
func BenchmarkCrudP_EncodePacket(b *testing.B) {
	cp := New()
	if err := cp.RegisterHandler(&BenchUser{}); err != nil {
		b.Fatalf("RegisterHandler failed: %v", err)
	}

	var packet []byte
	var err error

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		packet, err = cp.EncodePacket('c', 0, "", globalUser)
		if err != nil {
			b.Fatalf("EncodePacket failed: %v", err)
		}
	}

	globalPacket = packet // Prevent optimization
}

// BenchmarkCrudP_ProcessPacket measures allocations for complete packet processing
func BenchmarkCrudP_ProcessPacket(b *testing.B) {
	cp := New()
	if err := cp.RegisterHandler(&BenchUser{}); err != nil {
		b.Fatalf("RegisterHandler failed: %v", err)
	}

	// Pre-encode a packet to process
	packet, err := cp.EncodePacket('c', 0, "", globalUser)
	if err != nil {
		b.Fatalf("Failed to create test packet: %v", err)
	}

	var response []byte

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		response, err = cp.ProcessPacket(context.Background(), packet)
		if err != nil {
			b.Fatalf("ProcessPacket failed: %v", err)
		}
	}

	globalResponse = response // Prevent optimization
}

// BenchmarkCrudP_FullCycle measures allocations for complete encode->process->decode cycle
func BenchmarkCrudP_FullCycle(b *testing.B) {
	cp := New()
	if err := cp.RegisterHandler(&BenchUser{}); err != nil {
		b.Fatalf("RegisterHandler failed: %v", err)
	}

	var packet []byte
	var response []byte
	var responsePacket Packet
	var err error

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Encode request
		packet, err = cp.EncodePacket('c', 0, "", globalUser)
		if err != nil {
			b.Fatalf("EncodePacket failed: %v", err)
		}

		// Process request
		response, err = cp.ProcessPacket(context.Background(), packet)
		if err != nil {
			b.Fatalf("ProcessPacket failed: %v", err)
		}

		// Decode response
		err = cp.DecodePacket(response, &responsePacket)
		if err != nil {
			b.Fatalf("DecodePacket failed: %v", err)
		}
	}

	globalResponse = response // Prevent optimization
}

// BenchmarkCrudP_MultipleUsers measures allocations with multiple users in one packet
func BenchmarkCrudP_MultipleUsers(b *testing.B) {
	cp := New()
	if err := cp.RegisterHandler(&BenchUser{}); err != nil {
		b.Fatalf("RegisterHandler failed: %v", err)
	}

	users := []*BenchUser{
		{ID: 1, Name: "User1", Email: "user1@example.com", Age: 20},
		{ID: 2, Name: "User2", Email: "user2@example.com", Age: 25},
		{ID: 3, Name: "User3", Email: "user3@example.com", Age: 30},
		{ID: 4, Name: "User4", Email: "user4@example.com", Age: 35},
		{ID: 5, Name: "User5", Email: "user5@example.com", Age: 40},
	}

	var response []byte

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create packet with multiple users
		packet, err := cp.EncodePacket('c', 0, "", users[0], users[1], users[2], users[3], users[4])
		if err != nil {
			b.Fatalf("EncodePacket failed: %v", err)
		}

		// Process packet
		response, err = cp.ProcessPacket(context.Background(), packet)
		if err != nil {
			b.Fatalf("ProcessPacket failed: %v", err)
		}
	}

	globalResponse = response // Prevent optimization
}

// BenchmarkCrudP_AllOperations measures allocations for all CRUD operations
func BenchmarkCrudP_AllOperations(b *testing.B) {
	cp := New()
	if err := cp.RegisterHandler(&BenchUser{}); err != nil {
		b.Fatalf("RegisterHandler failed: %v", err)
	}

	operations := []byte{'c', 'r', 'u', 'd'}
	var response []byte

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, op := range operations {
			// Encode packet for each operation
			packet, err := cp.EncodePacket(op, 0, "", globalUser)
			if err != nil {
				b.Fatalf("EncodePacket failed for operation %c: %v", op, err)
			}

			// Process packet
			response, err = cp.ProcessPacket(context.Background(), packet)
			if err != nil {
				b.Fatalf("ProcessPacket failed for operation %c: %v", op, err)
			}
		}
	}

	globalResponse = response // Prevent optimization
}

// BenchmarkCrudP_LargePayload measures allocations with larger string data
func BenchmarkCrudP_LargePayload(b *testing.B) {
	cp := New()
	if err := cp.RegisterHandler(&BenchUser{}); err != nil {
		b.Fatalf("RegisterHandler failed: %v", err)
	}

	// Create user with large strings
	largeUser := &BenchUser{
		ID:    1,
		Name:  "This is a very long name that simulates real-world data with lots of characters and information that might be stored in a typical user profile",
		Email: "very.long.email.address.that.simulates.real.world.usage@very.long.domain.name.example.com",
		Age:   25,
	}

	var response []byte

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		packet, err := cp.EncodePacket('c', 0, "", largeUser)
		if err != nil {
			b.Fatalf("EncodePacket failed: %v", err)
		}

		response, err = cp.ProcessPacket(context.Background(), packet)
		if err != nil {
			b.Fatalf("ProcessPacket failed: %v", err)
		}
	}

	globalResponse = response // Prevent optimization
}
