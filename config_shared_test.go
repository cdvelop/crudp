package crudp_test

import (
	"context"
	"testing"

	"github.com/cdvelop/crudp"
)

func ConfigDefaultsShared(t *testing.T) {
	t.Run("DefaultConfig Values", func(t *testing.T) {
		cfg := crudp.DefaultConfig()

		if cfg.APIEndpoint != "/api" {
			t.Errorf("expected /api, got %s", cfg.APIEndpoint)
		}
		if cfg.SSEEndpoint != "/events" {
			t.Errorf("expected /events, got %s", cfg.SSEEndpoint)
		}
		if cfg.BatchWindow != 50 {
			t.Errorf("expected 50, got %d", cfg.BatchWindow)
		}
		if cfg.MaxRetries != 3 {
			t.Errorf("expected 3, got %d", cfg.MaxRetries)
		}
		if cfg.Port != ":6060" {
			t.Errorf("expected :6060, got %s", cfg.Port)
		}
	})
}

func NewWithConfigShared(t *testing.T) {
	t.Run("Custom Config", func(t *testing.T) {
		cfg := &crudp.Config{
			APIEndpoint: "/api/v2",
			BatchWindow: 100,
			Port:        ":8080",
		}

		cp := crudp.New(cfg)

		if cp.Config().APIEndpoint != "/api/v2" {
			t.Error("config not applied")
		}
		if cp.Config().BatchWindow != 100 {
			t.Error("BatchWindow not applied")
		}
	})

	t.Run("Nil Config Uses Defaults", func(t *testing.T) {
		cp := crudp.New(nil)

		if cp.Config() == nil {
			t.Error("expected default config")
		}
		if cp.Config().APIEndpoint != "/api" {
			t.Error("expected default APIEndpoint")
		}
	})

	t.Run("NewDefault", func(t *testing.T) {
		cp := crudp.NewDefault()

		if cp.Codec() == nil {
			t.Error("expected default codec")
		}
		if cp.Config().APIEndpoint != "/api" {
			t.Error("expected default config")
		}
	})
}

func LoggerConfigShared(t *testing.T) {
	t.Run("Logger Disabled By Default", func(t *testing.T) {
		cp := crudp.NewDefault()

		// This should not cause panic
		// Internal logger is no-op
		cp.DisableLogger()
	})

	t.Run("SetLogger Custom", func(t *testing.T) {
		cp := crudp.NewDefault()

		var logged []any
		cp.SetLogger(func(args ...any) {
			logged = append(logged, args...)
		})

		// Register handler to trigger log
		err := cp.RegisterHandler(&testLogHandler{})
		if err != nil {
			t.Fatal(err)
		}

		if len(logged) == 0 {
			t.Error("expected log output")
		}
	})

	t.Run("SetLogger Nil Restores NoOp", func(t *testing.T) {
		cp := crudp.NewDefault()

		cp.SetLogger(func(args ...any) {
			// Custom logger
		})

		cp.SetLogger(nil) // Should restore no-op

		// Should not cause panic
		cp.DisableLogger()
	})
}

type testLogHandler struct{}

func (h *testLogHandler) Create(ctx context.Context, data ...any) any {
	return "ok"
}

func CodecShared(t *testing.T) {
	t.Run("Default Codec EncodeDecode", func(t *testing.T) {
		cp := crudp.NewDefault()

		type testData struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		original := testData{Name: "test", Value: 42}

		encoded, err := cp.Codec().Encode(original)
		if err != nil {
			t.Fatalf("encode error: %v", err)
		}

		var decoded testData
		if err := cp.Codec().Decode(encoded, &decoded); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if decoded.Name != original.Name || decoded.Value != original.Value {
			t.Errorf("decode mismatch: got %+v, want %+v", decoded, original)
		}
	})

	t.Run("SetCodec Custom", func(t *testing.T) {
		cp := crudp.NewDefault()
		originalCodec := cp.Codec()

		// Create custom mock codec
		mockCodec := &mockCodec{}
		cp.SetCodec(mockCodec)

		if cp.Codec() == originalCodec {
			t.Error("codec should have changed")
		}
	})

	t.Run("SetCodec Nil Ignored", func(t *testing.T) {
		cp := crudp.NewDefault()
		originalCodec := cp.Codec()

		cp.SetCodec(nil)

		if cp.Codec() != originalCodec {
			t.Error("nil codec should be ignored")
		}
	})
}

// Mock codec for tests
type mockCodec struct{}

func (m *mockCodec) Encode(data any) ([]byte, error) {
	return []byte("mock"), nil
}

func (m *mockCodec) Decode(data []byte, v any) error {
	return nil
}
