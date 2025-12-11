package transport

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewMQTT(t *testing.T) {
	tests := []struct {
		name     string
		broker   string
		deviceID string
		opts     []Option
	}{
		{
			name:     "basic MQTT",
			broker:   "tcp://192.168.1.10:1883",
			deviceID: "shellyplus1pm-abc123",
		},
		{
			name:     "with topic",
			broker:   "tcp://192.168.1.10:1883",
			deviceID: "shellyplus1pm-abc123",
			opts:     []Option{WithMQTTTopic("shellies/device-id")},
		},
		{
			name:     "with client ID and QoS",
			broker:   "tcp://192.168.1.10:1883",
			deviceID: "shellyplus1pm-abc123",
			opts: []Option{
				WithMQTTClientID("test-client"),
				WithMQTTQoS(1),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqtt := NewMQTT(tt.broker, tt.deviceID, tt.opts...)
			if mqtt == nil {
				t.Fatal("NewMQTT() returned nil")
			}
			if mqtt.broker != tt.broker {
				t.Errorf("broker = %v, want %v", mqtt.broker, tt.broker)
			}
			if mqtt.deviceID != tt.deviceID {
				t.Errorf("deviceID = %v, want %v", mqtt.deviceID, tt.deviceID)
			}
		})
	}
}

func TestMQTT_CallNotConnected(t *testing.T) {
	// Call on non-connected MQTT should attempt to connect and fail
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	cancel() // Cancel immediately to avoid waiting

	_, err := mqtt.Call(ctx, "Switch.Set", nil)
	if err == nil {
		t.Error("Call() error = nil, want error (should fail to connect)")
	}
}

func TestMQTT_Subscribe(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	handler := NotificationHandler(func(data json.RawMessage) {})
	err := mqtt.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() error = %v", err)
	}

	// Second subscribe should fail
	err = mqtt.Subscribe(handler)
	if err == nil {
		t.Error("second Subscribe() error = nil, want error")
	}
}

func TestMQTT_Unsubscribe(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	handler := NotificationHandler(func(data json.RawMessage) {})
	err := mqtt.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() error = %v", err)
	}

	err = mqtt.Unsubscribe()
	if err != nil {
		t.Errorf("Unsubscribe() error = %v", err)
	}

	// Should be able to subscribe again
	err = mqtt.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() after Unsubscribe() error = %v", err)
	}
}

func TestMQTT_State(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	if mqtt.State() != StateDisconnected {
		t.Errorf("initial State() = %v, want %v", mqtt.State(), StateDisconnected)
	}
}

func TestMQTT_OnStateChange(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	called := false
	mqtt.OnStateChange(func(state ConnectionState) {
		called = true
	})

	mqtt.setState(StateConnected)

	if !called {
		t.Error("state change callback not called")
	}
}

func TestMQTT_Close(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	err := mqtt.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if mqtt.State() != StateClosed {
		t.Errorf("State() after Close() = %v, want %v", mqtt.State(), StateClosed)
	}

	// Calling Close again should be safe
	err = mqtt.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestMQTT_ClosedCall(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")
	mqtt.Close()

	_, err := mqtt.Call(context.Background(), "Switch.Set", nil)
	if err == nil {
		t.Error("Call() on closed MQTT error = nil, want error")
	}
}

func TestMQTT_DeviceID(t *testing.T) {
	deviceID := "shellyplus1pm-abc123"
	mqtt := NewMQTT("tcp://192.168.1.10:1883", deviceID)

	if mqtt.DeviceID() != deviceID {
		t.Errorf("DeviceID() = %v, want %v", mqtt.DeviceID(), deviceID)
	}
}

func TestMQTT_IsConnected(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	if mqtt.IsConnected() {
		t.Error("IsConnected() = true, want false (not connected)")
	}
}
