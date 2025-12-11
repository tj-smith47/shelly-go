package notifications

import (
	"testing"

	"github.com/tj-smith47/shelly-go/events"
)

func TestParseGen2Notification_NotifyStatus(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus1-aabbcc",
		"dst": "client",
		"method": "NotifyStatus",
		"params": {
			"switch:0": {"output": true, "source": "button"}
		}
	}`)

	evts, err := ParseGen2Notification("", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(evts) != 1 {
		t.Fatalf("len(evts) = %d, want 1", len(evts))
	}

	evt, ok := evts[0].(*events.StatusChangeEvent)
	if !ok {
		t.Fatalf("expected StatusChangeEvent, got %T", evts[0])
	}

	if evt.Type() != events.EventTypeStatusChange {
		t.Errorf("Type() = %q, want %q", evt.Type(), events.EventTypeStatusChange)
	}
	if evt.DeviceID() != "shellyplus1-aabbcc" {
		t.Errorf("DeviceID() = %q, want %q", evt.DeviceID(), "shellyplus1-aabbcc")
	}
	if evt.Component != "switch:0" {
		t.Errorf("Component = %q, want %q", evt.Component, "switch:0")
	}
	if evt.Source() != events.EventSourceWebSocket {
		t.Errorf("Source() = %q, want %q", evt.Source(), events.EventSourceWebSocket)
	}
}

func TestParseGen2Notification_NotifyStatus_MultipleComponents(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus2pm-aabbcc",
		"method": "NotifyStatus",
		"params": {
			"switch:0": {"output": true},
			"switch:1": {"output": false}
		}
	}`)

	evts, err := ParseGen2Notification("", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(evts) != 2 {
		t.Fatalf("len(evts) = %d, want 2", len(evts))
	}

	// Verify both are StatusChangeEvents
	for _, evt := range evts {
		if evt.Type() != events.EventTypeStatusChange {
			t.Errorf("expected StatusChangeEvent, got %s", evt.Type())
		}
	}
}

func TestParseGen2Notification_NotifyStatus_WithDeviceID(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus1-aabbcc",
		"method": "NotifyStatus",
		"params": {
			"switch:0": {"output": true}
		}
	}`)

	// Override device ID
	evts, err := ParseGen2Notification("custom-device-id", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(evts) != 1 {
		t.Fatalf("len(evts) = %d, want 1", len(evts))
	}

	// Provided device ID takes precedence
	if evts[0].DeviceID() != "custom-device-id" {
		t.Errorf("DeviceID() = %q, want %q", evts[0].DeviceID(), "custom-device-id")
	}
}

func TestParseGen2Notification_NotifyFullStatus(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus1-aabbcc",
		"method": "NotifyFullStatus",
		"params": {
			"switch:0": {"id": 0, "output": true},
			"input:0": {"id": 0, "state": false},
			"sys": {"mac": "AABBCC112233"}
		}
	}`)

	evts, err := ParseGen2Notification("", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(evts) != 1 {
		t.Fatalf("len(evts) = %d, want 1", len(evts))
	}

	evt, ok := evts[0].(*events.FullStatusEvent)
	if !ok {
		t.Fatalf("expected FullStatusEvent, got %T", evts[0])
	}

	if evt.Type() != events.EventTypeFullStatus {
		t.Errorf("Type() = %q, want %q", evt.Type(), events.EventTypeFullStatus)
	}
	if evt.DeviceID() != "shellyplus1-aabbcc" {
		t.Errorf("DeviceID() = %q, want %q", evt.DeviceID(), "shellyplus1-aabbcc")
	}
	if len(evt.Status) == 0 {
		t.Error("Status should not be empty")
	}
}

func TestParseGen2Notification_NotifyEvent(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus1-aabbcc",
		"method": "NotifyEvent",
		"params": {
			"ts": 1704067200.5,
			"events": [
				{"component": "input:0", "id": 0, "event": "single_push"}
			]
		}
	}`)

	evts, err := ParseGen2Notification("", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(evts) != 1 {
		t.Fatalf("len(evts) = %d, want 1", len(evts))
	}

	evt, ok := evts[0].(*events.NotifyEvent)
	if !ok {
		t.Fatalf("expected NotifyEvent, got %T", evts[0])
	}

	if evt.Type() != events.EventTypeNotify {
		t.Errorf("Type() = %q, want %q", evt.Type(), events.EventTypeNotify)
	}
	if evt.DeviceID() != "shellyplus1-aabbcc" {
		t.Errorf("DeviceID() = %q, want %q", evt.DeviceID(), "shellyplus1-aabbcc")
	}
	if evt.Component != "input:0" {
		t.Errorf("Component = %q, want %q", evt.Component, "input:0")
	}
	if evt.Event != "single_push" {
		t.Errorf("Event = %q, want %q", evt.Event, "single_push")
	}
}

func TestParseGen2Notification_NotifyEvent_MultipleEvents(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus1-aabbcc",
		"method": "NotifyEvent",
		"params": {
			"ts": 1704067200.5,
			"events": [
				{"component": "input:0", "id": 0, "event": "btn_down"},
				{"component": "input:0", "id": 0, "event": "single_push"},
				{"component": "input:0", "id": 0, "event": "btn_up"}
			]
		}
	}`)

	evts, err := ParseGen2Notification("", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(evts) != 3 {
		t.Fatalf("len(evts) = %d, want 3", len(evts))
	}

	// Verify all are NotifyEvents
	expectedEvents := []string{"btn_down", "single_push", "btn_up"}
	for i, evt := range evts {
		ne, ok := evt.(*events.NotifyEvent)
		if !ok {
			t.Fatalf("event %d: expected NotifyEvent, got %T", i, evt)
		}
		if ne.Event != expectedEvents[i] {
			t.Errorf("event %d: Event = %q, want %q", i, ne.Event, expectedEvents[i])
		}
	}
}

func TestParseGen2Notification_InvalidJSON(t *testing.T) {
	data := []byte(`{invalid json`)

	_, err := ParseGen2Notification("device", data)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseGen2Notification_UnknownMethod(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus1-aabbcc",
		"method": "UnknownMethod",
		"params": {}
	}`)

	_, err := ParseGen2Notification("", data)
	if err == nil {
		t.Error("expected error for unknown method")
	}
}

func TestParseGen2Notification_InvalidStatusParams(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus1-aabbcc",
		"method": "NotifyStatus",
		"params": "invalid"
	}`)

	_, err := ParseGen2Notification("", data)
	if err == nil {
		t.Error("expected error for invalid status params")
	}
}

func TestParseGen2Notification_InvalidEventParams(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus1-aabbcc",
		"method": "NotifyEvent",
		"params": "invalid"
	}`)

	_, err := ParseGen2Notification("", data)
	if err == nil {
		t.Error("expected error for invalid event params")
	}
}

func TestParseGen2NotificationJSON(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus1-aabbcc",
		"method": "NotifyStatus",
		"params": {"switch:0": {"output": true}}
	}`)

	evts, err := ParseGen2NotificationJSON("", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(evts) != 1 {
		t.Fatalf("len(evts) = %d, want 1", len(evts))
	}
}

func TestIsNotification(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "NotifyStatus",
			data: []byte(`{"method": "NotifyStatus", "params": {}}`),
			want: true,
		},
		{
			name: "NotifyFullStatus",
			data: []byte(`{"method": "NotifyFullStatus", "params": {}}`),
			want: true,
		},
		{
			name: "NotifyEvent",
			data: []byte(`{"method": "NotifyEvent", "params": {}}`),
			want: true,
		},
		{
			name: "RPC response with id",
			data: []byte(`{"id": 1, "result": {}}`),
			want: false,
		},
		{
			name: "RPC request with id",
			data: []byte(`{"id": 1, "method": "Switch.GetStatus"}`),
			want: false,
		},
		{
			name: "Unknown method",
			data: []byte(`{"method": "Unknown.Method"}`),
			want: false,
		},
		{
			name: "Invalid JSON",
			data: []byte(`{invalid`),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotification(tt.data)
			if got != tt.want {
				t.Errorf("IsNotification() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNotificationMethod(t *testing.T) {
	tests := []struct {
		name string
		want NotificationMethod
		data []byte
	}{
		{
			name: "NotifyStatus",
			data: []byte(`{"method": "NotifyStatus"}`),
			want: MethodNotifyStatus,
		},
		{
			name: "NotifyFullStatus",
			data: []byte(`{"method": "NotifyFullStatus"}`),
			want: MethodNotifyFullStatus,
		},
		{
			name: "NotifyEvent",
			data: []byte(`{"method": "NotifyEvent"}`),
			want: MethodNotifyEvent,
		},
		{
			name: "Unknown method",
			data: []byte(`{"method": "Unknown"}`),
			want: "Unknown",
		},
		{
			name: "No method",
			data: []byte(`{}`),
			want: "",
		},
		{
			name: "Invalid JSON",
			data: []byte(`{invalid`),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetNotificationMethod(tt.data)
			if got != tt.want {
				t.Errorf("GetNotificationMethod() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetNotificationSource(t *testing.T) {
	tests := []struct {
		name string
		want string
		data []byte
	}{
		{
			name: "with src",
			data: []byte(`{"src": "shellyplus1-aabbcc"}`),
			want: "shellyplus1-aabbcc",
		},
		{
			name: "no src",
			data: []byte(`{"method": "NotifyStatus"}`),
			want: "",
		},
		{
			name: "invalid JSON",
			data: []byte(`{invalid`),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetNotificationSource(tt.data)
			if got != tt.want {
				t.Errorf("GetNotificationSource() = %q, want %q", got, tt.want)
			}
		})
	}
}
