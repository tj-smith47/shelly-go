package transport

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewCoAP(t *testing.T) {
	tests := []struct {
		name    string
		address string
		opts    []Option
	}{
		{
			name:    "basic CoAP",
			address: "192.168.1.100",
		},
		{
			name:    "multicast",
			address: "224.0.1.187",
			opts:    []Option{WithCoAPMulticast()},
		},
		{
			name:    "custom port",
			address: "192.168.1.100",
			opts:    []Option{WithCoAPPort(5684)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coap := NewCoAP(tt.address, tt.opts...)
			if coap == nil {
				t.Fatal("NewCoAP() returned nil")
			}
			if coap.address != tt.address {
				t.Errorf("address = %v, want %v", coap.address, tt.address)
			}
		})
	}
}

func TestCoAP_Call(t *testing.T) {
	coap := NewCoAP("192.168.1.100")
	_, err := coap.Call(context.Background(), "Switch.Set", nil)
	if err == nil {
		t.Error("Call() error = nil, want not implemented error")
	}
}

func TestCoAP_Subscribe(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	handler := NotificationHandler(func(data json.RawMessage) {})
	err := coap.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() error = %v", err)
	}

	// Second subscribe should fail
	err = coap.Subscribe(handler)
	if err == nil {
		t.Error("second Subscribe() error = nil, want error")
	}
}

func TestCoAP_Unsubscribe(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	handler := NotificationHandler(func(data json.RawMessage) {})
	coap.Subscribe(handler)

	err := coap.Unsubscribe()
	if err != nil {
		t.Errorf("Unsubscribe() error = %v", err)
	}

	// Should be able to subscribe again
	err = coap.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() after Unsubscribe() error = %v", err)
	}
}

func TestCoAP_State(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	if coap.State() != StateDisconnected {
		t.Errorf("initial State() = %v, want %v", coap.State(), StateDisconnected)
	}
}

func TestCoAP_OnStateChange(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	called := false
	coap.OnStateChange(func(state ConnectionState) {
		called = true
	})

	coap.setState(StateConnected)

	if !called {
		t.Error("state change callback not called")
	}
}

func TestCoAP_Close(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	err := coap.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if coap.State() != StateClosed {
		t.Errorf("State() after Close() = %v, want %v", coap.State(), StateClosed)
	}

	// Calling Close again should be safe
	err = coap.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestCoAP_ClosedCall(t *testing.T) {
	coap := NewCoAP("192.168.1.100")
	coap.Close()

	_, err := coap.Call(context.Background(), "Switch.Set", nil)
	if err == nil {
		t.Error("Call() on closed CoAP error = nil, want error")
	}
}

func TestCoAP_Address(t *testing.T) {
	coap := NewCoAP("192.168.1.100")
	if got := coap.Address(); got != "192.168.1.100" {
		t.Errorf("Address() = %v, want %v", got, "192.168.1.100")
	}
}

func TestCoAP_IsMulticast(t *testing.T) {
	// Non-multicast CoAP
	coap := NewCoAP("192.168.1.100")
	if coap.IsMulticast() {
		t.Error("IsMulticast() = true for unicast, want false")
	}

	// Multicast CoAP
	coap = NewCoAP("224.0.1.187", WithCoAPMulticast())
	if !coap.IsMulticast() {
		t.Error("IsMulticast() = false for multicast, want true")
	}
}

func TestCoAP_IsConnected(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	// Initially not connected (no conn set)
	if coap.IsConnected() {
		t.Error("IsConnected() = true initially, want false")
	}

	// After closing - still not connected
	coap.Close()
	if coap.IsConnected() {
		t.Error("IsConnected() = true after close, want false")
	}
}
