package crudp_test

import (
	"testing"

	"github.com/cdvelop/crudp"
	. "github.com/cdvelop/tinystring"
)

func PacketResultMessageTypeShared(t *testing.T) {
	t.Run("MessageType Success", func(t *testing.T) {
		pr := crudp.PacketResult{
			Packet:      crudp.Packet{Action: 'c', HandlerID: 0, ReqID: "test-1"},
			MessageType: uint8(Msg.Success),
			Message:     "Created",
		}

		if pr.MessageType != uint8(Msg.Success) {
			t.Errorf("expected MessageType %d, got %d", uint8(Msg.Success), pr.MessageType)
		}

		if pr.Action != 'c' {
			t.Errorf("expected Action 'c', got %c", pr.Action)
		}

		if pr.ReqID != "test-1" {
			t.Errorf("expected ReqID 'test-1', got %s", pr.ReqID)
		}
	})

	t.Run("MessageType Error", func(t *testing.T) {
		pr := crudp.PacketResult{
			Packet:      crudp.Packet{Action: 'r', HandlerID: 1, ReqID: "test-2"},
			MessageType: uint8(Msg.Error),
			Message:     "Not found",
		}

		if pr.MessageType != uint8(Msg.Error) {
			t.Errorf("expected MessageType %d, got %d", uint8(Msg.Error), pr.MessageType)
		}
	})

	t.Run("Multiple Data Responses", func(t *testing.T) {
		pr := crudp.PacketResult{
			Packet: crudp.Packet{
				Action:    'r',
				HandlerID: 0,
				ReqID:     "test-3",
				Data: [][]byte{
					[]byte(`{"id":1,"name":"Alice"}`),
					[]byte(`{"id":2,"name":"Bob"}`),
					[]byte(`{"id":3,"name":"Charlie"}`),
				},
			},
			MessageType: uint8(Msg.Success),
			Message:     "OK",
		}

		if len(pr.Data) != 3 {
			t.Errorf("expected 3 data items, got %d", len(pr.Data))
		}
	})
}

func ActionConversionShared(t *testing.T) {
	tests := []struct {
		method string
		action byte
	}{
		{"POST", 'c'},
		{"GET", 'r'},
		{"PUT", 'u'},
		{"DELETE", 'd'},
		{"INVALID", 0},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := crudp.MethodToAction(tt.method)
			if got != tt.action {
				t.Errorf("MethodToAction(%s) = %c, want %c", tt.method, got, tt.action)
			}

			if tt.action != 0 {
				gotMethod := crudp.ActionToMethod(tt.action)
				if gotMethod != tt.method {
					t.Errorf("ActionToMethod(%c) = %s, want %s", tt.action, gotMethod, tt.method)
				}
			}
		})
	}
}
