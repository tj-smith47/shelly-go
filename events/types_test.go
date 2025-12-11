package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBaseEvent(t *testing.T) {
	base := BaseEvent{
		eventType: EventTypeStatusChange,
		deviceID:  "shellyplus1-aabbcc",
		timestamp: time.Now(),
		source:    EventSourceLocal,
	}

	if base.Type() != EventTypeStatusChange {
		t.Errorf("Type() = %v, want %v", base.Type(), EventTypeStatusChange)
	}
	if base.DeviceID() != "shellyplus1-aabbcc" {
		t.Errorf("DeviceID() = %v, want %v", base.DeviceID(), "shellyplus1-aabbcc")
	}
	if base.Source() != EventSourceLocal {
		t.Errorf("Source() = %v, want %v", base.Source(), EventSourceLocal)
	}
	if base.Timestamp().IsZero() {
		t.Error("Timestamp() should not be zero")
	}
}

func TestNewStatusChangeEvent(t *testing.T) {
	status := json.RawMessage(`{"output":true}`)
	event := NewStatusChangeEvent("device1", "switch:0", status)

	if event.Type() != EventTypeStatusChange {
		t.Errorf("Type() = %v, want %v", event.Type(), EventTypeStatusChange)
	}
	if event.DeviceID() != "device1" {
		t.Errorf("DeviceID() = %v, want %v", event.DeviceID(), "device1")
	}
	if event.Component != "switch:0" {
		t.Errorf("Component = %v, want %v", event.Component, "switch:0")
	}
	if string(event.Status) != `{"output":true}` {
		t.Errorf("Status = %v, want %v", string(event.Status), `{"output":true}`)
	}
	if event.Source() != EventSourceLocal {
		t.Errorf("Source() = %v, want %v", event.Source(), EventSourceLocal)
	}
}

func TestStatusChangeEvent_WithSource(t *testing.T) {
	event := NewStatusChangeEvent("device1", "switch:0", nil).
		WithSource(EventSourceCloud)

	if event.Source() != EventSourceCloud {
		t.Errorf("Source() = %v, want %v", event.Source(), EventSourceCloud)
	}
}

func TestStatusChangeEvent_WithDelta(t *testing.T) {
	delta := json.RawMessage(`{"output":false}`)
	event := NewStatusChangeEvent("device1", "switch:0", nil).
		WithDelta(delta)

	if string(event.Delta) != `{"output":false}` {
		t.Errorf("Delta = %v, want %v", string(event.Delta), `{"output":false}`)
	}
}

func TestNewFullStatusEvent(t *testing.T) {
	status := json.RawMessage(`{"switch:0":{"output":true}}`)
	event := NewFullStatusEvent("device1", status)

	if event.Type() != EventTypeFullStatus {
		t.Errorf("Type() = %v, want %v", event.Type(), EventTypeFullStatus)
	}
	if event.DeviceID() != "device1" {
		t.Errorf("DeviceID() = %v, want %v", event.DeviceID(), "device1")
	}
	if string(event.Status) != `{"switch:0":{"output":true}}` {
		t.Errorf("Status = %v, want %v", string(event.Status), `{"switch:0":{"output":true}}`)
	}
}

func TestFullStatusEvent_WithSource(t *testing.T) {
	event := NewFullStatusEvent("device1", nil).
		WithSource(EventSourceWebSocket)

	if event.Source() != EventSourceWebSocket {
		t.Errorf("Source() = %v, want %v", event.Source(), EventSourceWebSocket)
	}
}

func TestNewNotifyEvent(t *testing.T) {
	event := NewNotifyEvent("device1", "input:0", InputEventSinglePush)

	if event.Type() != EventTypeNotify {
		t.Errorf("Type() = %v, want %v", event.Type(), EventTypeNotify)
	}
	if event.DeviceID() != "device1" {
		t.Errorf("DeviceID() = %v, want %v", event.DeviceID(), "device1")
	}
	if event.Component != "input:0" {
		t.Errorf("Component = %v, want %v", event.Component, "input:0")
	}
	if event.Event != InputEventSinglePush {
		t.Errorf("Event = %v, want %v", event.Event, InputEventSinglePush)
	}
}

func TestNotifyEvent_WithSource(t *testing.T) {
	event := NewNotifyEvent("device1", "input:0", InputEventDoublePush).
		WithSource(EventSourceCoIoT)

	if event.Source() != EventSourceCoIoT {
		t.Errorf("Source() = %v, want %v", event.Source(), EventSourceCoIoT)
	}
}

func TestNotifyEvent_WithData(t *testing.T) {
	data := json.RawMessage(`{"ts":1234567890}`)
	event := NewNotifyEvent("device1", "input:0", InputEventLongPush).
		WithData(data)

	if string(event.Data) != `{"ts":1234567890}` {
		t.Errorf("Data = %v, want %v", string(event.Data), `{"ts":1234567890}`)
	}
}

func TestNewDeviceOnlineEvent(t *testing.T) {
	event := NewDeviceOnlineEvent("device1")

	if event.Type() != EventTypeDeviceOnline {
		t.Errorf("Type() = %v, want %v", event.Type(), EventTypeDeviceOnline)
	}
	if event.DeviceID() != "device1" {
		t.Errorf("DeviceID() = %v, want %v", event.DeviceID(), "device1")
	}
}

func TestDeviceOnlineEvent_WithAddress(t *testing.T) {
	event := NewDeviceOnlineEvent("device1").
		WithAddress("192.168.1.100")

	if event.Address != "192.168.1.100" {
		t.Errorf("Address = %v, want %v", event.Address, "192.168.1.100")
	}
}

func TestDeviceOnlineEvent_WithSource(t *testing.T) {
	event := NewDeviceOnlineEvent("device1").
		WithSource(EventSourceCloud)

	if event.Source() != EventSourceCloud {
		t.Errorf("Source() = %v, want %v", event.Source(), EventSourceCloud)
	}
}

func TestNewDeviceOfflineEvent(t *testing.T) {
	event := NewDeviceOfflineEvent("device1")

	if event.Type() != EventTypeDeviceOffline {
		t.Errorf("Type() = %v, want %v", event.Type(), EventTypeDeviceOffline)
	}
	if event.DeviceID() != "device1" {
		t.Errorf("DeviceID() = %v, want %v", event.DeviceID(), "device1")
	}
}

func TestDeviceOfflineEvent_WithReason(t *testing.T) {
	event := NewDeviceOfflineEvent("device1").
		WithReason("connection timeout")

	if event.Reason != "connection timeout" {
		t.Errorf("Reason = %v, want %v", event.Reason, "connection timeout")
	}
}

func TestDeviceOfflineEvent_WithSource(t *testing.T) {
	event := NewDeviceOfflineEvent("device1").
		WithSource(EventSourceMQTT)

	if event.Source() != EventSourceMQTT {
		t.Errorf("Source() = %v, want %v", event.Source(), EventSourceMQTT)
	}
}

func TestNewUpdateAvailableEvent(t *testing.T) {
	event := NewUpdateAvailableEvent("device1", "1.0.0", "1.1.0")

	if event.Type() != EventTypeUpdateAvailable {
		t.Errorf("Type() = %v, want %v", event.Type(), EventTypeUpdateAvailable)
	}
	if event.DeviceID() != "device1" {
		t.Errorf("DeviceID() = %v, want %v", event.DeviceID(), "device1")
	}
	if event.CurrentVersion != "1.0.0" {
		t.Errorf("CurrentVersion = %v, want %v", event.CurrentVersion, "1.0.0")
	}
	if event.AvailableVersion != "1.1.0" {
		t.Errorf("AvailableVersion = %v, want %v", event.AvailableVersion, "1.1.0")
	}
}

func TestUpdateAvailableEvent_WithStage(t *testing.T) {
	event := NewUpdateAvailableEvent("device1", "1.0.0", "1.1.0").
		WithStage("beta")

	if event.Stage != "beta" {
		t.Errorf("Stage = %v, want %v", event.Stage, "beta")
	}
}

func TestUpdateAvailableEvent_WithSource(t *testing.T) {
	event := NewUpdateAvailableEvent("device1", "1.0.0", "1.1.0").
		WithSource(EventSourceCloud)

	if event.Source() != EventSourceCloud {
		t.Errorf("Source() = %v, want %v", event.Source(), EventSourceCloud)
	}
}

func TestNewScriptEvent(t *testing.T) {
	event := NewScriptEvent("device1", 1, "hello world")

	if event.Type() != EventTypeScript {
		t.Errorf("Type() = %v, want %v", event.Type(), EventTypeScript)
	}
	if event.DeviceID() != "device1" {
		t.Errorf("DeviceID() = %v, want %v", event.DeviceID(), "device1")
	}
	if event.ScriptID != 1 {
		t.Errorf("ScriptID = %v, want %v", event.ScriptID, 1)
	}
	if event.Output != "hello world" {
		t.Errorf("Output = %v, want %v", event.Output, "hello world")
	}
}

func TestScriptEvent_WithSource(t *testing.T) {
	event := NewScriptEvent("device1", 1, "test").
		WithSource(EventSourceWebSocket)

	if event.Source() != EventSourceWebSocket {
		t.Errorf("Source() = %v, want %v", event.Source(), EventSourceWebSocket)
	}
}

func TestNewConfigChangeEvent(t *testing.T) {
	config := json.RawMessage(`{"name":"Living Room"}`)
	event := NewConfigChangeEvent("device1", "switch:0", config)

	if event.Type() != EventTypeConfig {
		t.Errorf("Type() = %v, want %v", event.Type(), EventTypeConfig)
	}
	if event.DeviceID() != "device1" {
		t.Errorf("DeviceID() = %v, want %v", event.DeviceID(), "device1")
	}
	if event.Component != "switch:0" {
		t.Errorf("Component = %v, want %v", event.Component, "switch:0")
	}
	if string(event.Config) != `{"name":"Living Room"}` {
		t.Errorf("Config = %v, want %v", string(event.Config), `{"name":"Living Room"}`)
	}
}

func TestConfigChangeEvent_WithSource(t *testing.T) {
	event := NewConfigChangeEvent("device1", "switch:0", nil).
		WithSource(EventSourceLocal)

	if event.Source() != EventSourceLocal {
		t.Errorf("Source() = %v, want %v", event.Source(), EventSourceLocal)
	}
}

func TestNewErrorEvent(t *testing.T) {
	event := NewErrorEvent("device1", 404, "not found")

	if event.Type() != EventTypeError {
		t.Errorf("Type() = %v, want %v", event.Type(), EventTypeError)
	}
	if event.DeviceID() != "device1" {
		t.Errorf("DeviceID() = %v, want %v", event.DeviceID(), "device1")
	}
	if event.Code != 404 {
		t.Errorf("Code = %v, want %v", event.Code, 404)
	}
	if event.Message != "not found" {
		t.Errorf("Message = %v, want %v", event.Message, "not found")
	}
}

func TestErrorEvent_WithComponent(t *testing.T) {
	event := NewErrorEvent("device1", 500, "internal error").
		WithComponent("switch:0")

	if event.Component != "switch:0" {
		t.Errorf("Component = %v, want %v", event.Component, "switch:0")
	}
}

func TestErrorEvent_WithSource(t *testing.T) {
	event := NewErrorEvent("device1", 500, "error").
		WithSource(EventSourceCloud)

	if event.Source() != EventSourceCloud {
		t.Errorf("Source() = %v, want %v", event.Source(), EventSourceCloud)
	}
}

func TestInputEventConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"SinglePush", InputEventSinglePush, "single_push"},
		{"DoublePush", InputEventDoublePush, "double_push"},
		{"TriplePush", InputEventTriplePush, "triple_push"},
		{"LongPush", InputEventLongPush, "long_push"},
		{"BtnDown", InputEventBtnDown, "btn_down"},
		{"BtnUp", InputEventBtnUp, "btn_up"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("got %v, want %v", tt.constant, tt.expected)
			}
		})
	}
}

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant EventType
		expected string
	}{
		{"StatusChange", EventTypeStatusChange, "status_change"},
		{"FullStatus", EventTypeFullStatus, "full_status"},
		{"Notify", EventTypeNotify, "notify_event"},
		{"DeviceOnline", EventTypeDeviceOnline, "device_online"},
		{"DeviceOffline", EventTypeDeviceOffline, "device_offline"},
		{"UpdateAvailable", EventTypeUpdateAvailable, "update_available"},
		{"Script", EventTypeScript, "script"},
		{"Config", EventTypeConfig, "config_change"},
		{"Error", EventTypeError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("got %v, want %v", tt.constant, tt.expected)
			}
		})
	}
}

func TestEventSourceConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant EventSource
		expected string
	}{
		{"Local", EventSourceLocal, "local"},
		{"Cloud", EventSourceCloud, "cloud"},
		{"CoIoT", EventSourceCoIoT, "coiot"},
		{"WebSocket", EventSourceWebSocket, "websocket"},
		{"MQTT", EventSourceMQTT, "mqtt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("got %v, want %v", tt.constant, tt.expected)
			}
		})
	}
}
