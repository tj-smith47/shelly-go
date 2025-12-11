package transport

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewWebSocket(t *testing.T) {
	tests := []struct {
		name string
		url  string
		opts []Option
	}{
		{
			name: "basic WebSocket",
			url:  "ws://192.168.1.100/rpc",
		},
		{
			name: "with reconnect",
			url:  "ws://192.168.1.100/rpc",
			opts: []Option{WithReconnect(true)},
		},
		{
			name: "with ping interval",
			url:  "ws://192.168.1.100/rpc",
			opts: []Option{WithPingInterval(30)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := NewWebSocket(tt.url, tt.opts...)
			if ws == nil {
				t.Fatal("NewWebSocket() returned nil")
			}
			if ws.url != tt.url {
				t.Errorf("url = %v, want %v", ws.url, tt.url)
			}
		})
	}
}

func TestWebSocket_Call(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")
	_, err := ws.Call(context.Background(), "Switch.Set", nil)
	if err == nil {
		t.Error("Call() error = nil, want not implemented error")
	}
}

func TestWebSocket_Subscribe(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	handler := NotificationHandler(func(data json.RawMessage) {})
	err := ws.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() error = %v", err)
	}

	// Second subscribe should fail
	err = ws.Subscribe(handler)
	if err == nil {
		t.Error("second Subscribe() error = nil, want error")
	}
}

func TestWebSocket_Unsubscribe(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	handler := NotificationHandler(func(data json.RawMessage) {})
	ws.Subscribe(handler)

	err := ws.Unsubscribe()
	if err != nil {
		t.Errorf("Unsubscribe() error = %v", err)
	}

	// Should be able to subscribe again after unsubscribe
	err = ws.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() after Unsubscribe() error = %v", err)
	}
}

func TestWebSocket_State(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	if ws.State() != StateDisconnected {
		t.Errorf("initial State() = %v, want %v", ws.State(), StateDisconnected)
	}
}

func TestWebSocket_OnStateChange(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	called := false
	ws.OnStateChange(func(state ConnectionState) {
		called = true
	})

	ws.setState(StateConnected)

	if !called {
		t.Error("state change callback not called")
	}
}

func TestWebSocket_Close(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	err := ws.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if ws.State() != StateClosed {
		t.Errorf("State() after Close() = %v, want %v", ws.State(), StateClosed)
	}

	// Calling Close again should be safe
	err = ws.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestWebSocket_ClosedCall(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")
	ws.Close()

	_, err := ws.Call(context.Background(), "Switch.Set", nil)
	if err == nil {
		t.Error("Call() on closed WebSocket error = nil, want error")
	}
}
