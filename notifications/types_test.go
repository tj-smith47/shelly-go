package notifications

import (
	"encoding/json"
	"testing"
)

func TestNotificationMethod_Constants(t *testing.T) {
	tests := []struct {
		method NotificationMethod
		want   string
	}{
		{MethodNotifyStatus, "NotifyStatus"},
		{MethodNotifyFullStatus, "NotifyFullStatus"},
		{MethodNotifyEvent, "NotifyEvent"},
	}

	for _, tt := range tests {
		if string(tt.method) != tt.want {
			t.Errorf("method %v = %q, want %q", tt.method, tt.method, tt.want)
		}
	}
}

func TestGenericNotification_Unmarshal(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus1-aabbcc",
		"dst": "client",
		"method": "NotifyStatus",
		"params": {"switch:0": {"output": true}}
	}`)

	var notif GenericNotification
	err := json.Unmarshal(data, &notif)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if notif.Src != "shellyplus1-aabbcc" {
		t.Errorf("Src = %q, want %q", notif.Src, "shellyplus1-aabbcc")
	}
	if notif.Dst != "client" {
		t.Errorf("Dst = %q, want %q", notif.Dst, "client")
	}
	if notif.Method != MethodNotifyStatus {
		t.Errorf("Method = %q, want %q", notif.Method, MethodNotifyStatus)
	}
	if len(notif.Params) == 0 {
		t.Error("Params should not be empty")
	}
}

func TestGenericNotification_Unmarshal_NoDst(t *testing.T) {
	data := []byte(`{
		"src": "shellyplus1-aabbcc",
		"method": "NotifyEvent",
		"params": {}
	}`)

	var notif GenericNotification
	err := json.Unmarshal(data, &notif)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if notif.Dst != "" {
		t.Errorf("Dst = %q, want empty", notif.Dst)
	}
}

func TestStatusNotificationParams_Unmarshal(t *testing.T) {
	data := []byte(`{
		"switch:0": {"output": true, "source": "button"},
		"switch:1": {"output": false}
	}`)

	var params StatusNotificationParams
	err := json.Unmarshal(data, &params)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(params) != 2 {
		t.Errorf("len(params) = %d, want 2", len(params))
	}
	if _, ok := params["switch:0"]; !ok {
		t.Error("expected switch:0 in params")
	}
	if _, ok := params["switch:1"]; !ok {
		t.Error("expected switch:1 in params")
	}
}

func TestEventNotificationParams_Unmarshal(t *testing.T) {
	data := []byte(`{
		"ts": 1704067200.5,
		"events": [
			{"component": "input:0", "id": 0, "event": "single_push"},
			{"component": "input:0", "id": 0, "event": "btn_up"}
		]
	}`)

	var params EventNotificationParams
	err := json.Unmarshal(data, &params)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if params.Ts != 1704067200.5 {
		t.Errorf("Ts = %v, want 1704067200.5", params.Ts)
	}
	if len(params.Events) != 2 {
		t.Errorf("len(Events) = %d, want 2", len(params.Events))
	}
}

func TestEventNotificationEvent_Unmarshal(t *testing.T) {
	data := []byte(`{
		"component": "input:0",
		"id": 0,
		"event": "single_push",
		"data": {"cnt": 1}
	}`)

	var event EventNotificationEvent
	err := json.Unmarshal(data, &event)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if event.Component != "input:0" {
		t.Errorf("Component = %q, want %q", event.Component, "input:0")
	}
	if event.ID != 0 {
		t.Errorf("ID = %d, want 0", event.ID)
	}
	if event.Event != "single_push" {
		t.Errorf("Event = %q, want %q", event.Event, "single_push")
	}
	if len(event.Data) == 0 {
		t.Error("Data should not be empty")
	}
}

func TestEventNotificationEvent_NoData(t *testing.T) {
	data := []byte(`{
		"component": "input:1",
		"id": 1,
		"event": "btn_down"
	}`)

	var event EventNotificationEvent
	err := json.Unmarshal(data, &event)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if event.Component != "input:1" {
		t.Errorf("Component = %q, want %q", event.Component, "input:1")
	}
	if len(event.Data) != 0 {
		t.Errorf("Data should be empty, got %s", event.Data)
	}
}

func TestFullStatusNotificationParams_Unmarshal(t *testing.T) {
	data := []byte(`{
		"switch:0": {"id": 0, "output": true},
		"input:0": {"id": 0, "state": false},
		"sys": {"mac": "AABBCC112233"}
	}`)

	var params FullStatusNotificationParams
	err := json.Unmarshal(data, &params)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(params) != 3 {
		t.Errorf("len(params) = %d, want 3", len(params))
	}
	if _, ok := params["switch:0"]; !ok {
		t.Error("expected switch:0 in params")
	}
	if _, ok := params["input:0"]; !ok {
		t.Error("expected input:0 in params")
	}
	if _, ok := params["sys"]; !ok {
		t.Error("expected sys in params")
	}
}
