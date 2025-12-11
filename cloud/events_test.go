package cloud

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestEventHandlers(t *testing.T) {
	handlers := NewEventHandlers()

	var onlineCalled, offlineCalled, statusChangeCalled bool
	var receivedDeviceID string

	handlers.OnDeviceOnline(func(deviceID string) {
		onlineCalled = true
		receivedDeviceID = deviceID
	})

	handlers.OnDeviceOffline(func(deviceID string) {
		offlineCalled = true
	})

	handlers.OnStatusChange(func(deviceID string, status json.RawMessage) {
		statusChangeCalled = true
	})

	// Test device online event
	handlers.Dispatch(&WebSocketMessage{
		Event:    EventDeviceOnline,
		DeviceID: "device123",
	})

	if !onlineCalled {
		t.Error("OnDeviceOnline handler not called")
	}
	if receivedDeviceID != "device123" {
		t.Errorf("deviceID = %v, want device123", receivedDeviceID)
	}

	// Test device offline event
	handlers.Dispatch(&WebSocketMessage{
		Event:    EventDeviceOffline,
		DeviceID: "device123",
	})

	if !offlineCalled {
		t.Error("OnDeviceOffline handler not called")
	}

	// Test status change event
	handlers.Dispatch(&WebSocketMessage{
		Event:    EventDeviceStatusChange,
		DeviceID: "device123",
		Status:   json.RawMessage(`{"output": true}`),
	})

	if !statusChangeCalled {
		t.Error("OnStatusChange handler not called")
	}
}

func TestEventHandlersGen2(t *testing.T) {
	handlers := NewEventHandlers()

	var notifyStatusCalled, notifyFullStatusCalled, notifyEventCalled bool
	var receivedStatus, receivedData json.RawMessage

	handlers.OnNotifyStatus(func(deviceID string, status json.RawMessage) {
		notifyStatusCalled = true
		receivedStatus = status
	})

	handlers.OnNotifyFullStatus(func(deviceID string, status json.RawMessage) {
		notifyFullStatusCalled = true
	})

	handlers.OnNotifyEvent(func(deviceID string, event json.RawMessage) {
		notifyEventCalled = true
		receivedData = event
	})

	// Test NotifyStatus
	handlers.Dispatch(&WebSocketMessage{
		Event:    EventNotifyStatus,
		DeviceID: "device123",
		Status:   json.RawMessage(`{"switch:0": {"output": true}}`),
	})

	if !notifyStatusCalled {
		t.Error("OnNotifyStatus handler not called")
	}
	if string(receivedStatus) != `{"switch:0": {"output": true}}` {
		t.Errorf("status = %v, want {\"switch:0\": {\"output\": true}}", string(receivedStatus))
	}

	// Test NotifyFullStatus
	handlers.Dispatch(&WebSocketMessage{
		Event:    EventNotifyFullStatus,
		DeviceID: "device123",
		Status:   json.RawMessage(`{}`),
	})

	if !notifyFullStatusCalled {
		t.Error("OnNotifyFullStatus handler not called")
	}

	// Test NotifyEvent
	handlers.Dispatch(&WebSocketMessage{
		Event:    EventNotifyEvent,
		DeviceID: "device123",
		Data:     json.RawMessage(`{"event": "single_push"}`),
	})

	if !notifyEventCalled {
		t.Error("OnNotifyEvent handler not called")
	}
	if string(receivedData) != `{"event": "single_push"}` {
		t.Errorf("data = %v, want {\"event\": \"single_push\"}", string(receivedData))
	}
}

func TestEventHandlersOnMessage(t *testing.T) {
	handlers := NewEventHandlers()

	var messageCalled bool
	var receivedMsg *WebSocketMessage

	handlers.OnMessage(func(msg *WebSocketMessage) {
		messageCalled = true
		receivedMsg = msg
	})

	msg := &WebSocketMessage{
		Event:    EventDeviceOnline,
		DeviceID: "device123",
	}

	handlers.Dispatch(msg)

	if !messageCalled {
		t.Error("OnMessage handler not called")
	}
	if receivedMsg != msg {
		t.Error("Received message mismatch")
	}
}

func TestEventHandlersClear(t *testing.T) {
	handlers := NewEventHandlers()

	var called bool
	handlers.OnDeviceOnline(func(deviceID string) {
		called = true
	})

	handlers.Clear()

	handlers.Dispatch(&WebSocketMessage{
		Event:    EventDeviceOnline,
		DeviceID: "device123",
	})

	if called {
		t.Error("Handler should not be called after Clear()")
	}
}

func TestEventHandlersMultipleHandlers(t *testing.T) {
	handlers := NewEventHandlers()

	var count int
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		handlers.OnDeviceOnline(func(deviceID string) {
			mu.Lock()
			count++
			mu.Unlock()
		})
	}

	handlers.Dispatch(&WebSocketMessage{
		Event:    EventDeviceOnline,
		DeviceID: "device123",
	})

	if count != 3 {
		t.Errorf("count = %v, want 3", count)
	}
}

func TestEventHandlersConcurrency(t *testing.T) {
	handlers := NewEventHandlers()

	var count int
	var mu sync.Mutex

	handlers.OnDeviceOnline(func(deviceID string) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	// Dispatch concurrently
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handlers.Dispatch(&WebSocketMessage{
				Event:    EventDeviceOnline,
				DeviceID: "device123",
			})
		}()
	}

	wg.Wait()

	if count != 100 {
		t.Errorf("count = %v, want 100", count)
	}
}

func TestParseSwitchStatus(t *testing.T) {
	data := json.RawMessage(`{
		"id": 0,
		"output": true,
		"source": "button",
		"timer_started_at": 1609459200.5,
		"timer_duration": 60.0
	}`)

	status, err := ParseSwitchStatus(data)
	if err != nil {
		t.Fatalf("ParseSwitchStatus failed: %v", err)
	}

	if status.ID != 0 {
		t.Errorf("ID = %v, want 0", status.ID)
	}
	if !status.Output {
		t.Error("Output = false, want true")
	}
	if status.Source != "button" {
		t.Errorf("Source = %v, want button", status.Source)
	}
	if status.TimerDuration != 60.0 {
		t.Errorf("TimerDuration = %v, want 60.0", status.TimerDuration)
	}
}

func TestParseSwitchStatusInvalid(t *testing.T) {
	data := json.RawMessage(`not valid json`)

	_, err := ParseSwitchStatus(data)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseCoverStatus(t *testing.T) {
	pos := 50
	targetPos := 100
	data := json.RawMessage(`{
		"id": 0,
		"state": "opening",
		"current_pos": 50,
		"target_pos": 100,
		"source": "cloud"
	}`)

	status, err := ParseCoverStatus(data)
	if err != nil {
		t.Fatalf("ParseCoverStatus failed: %v", err)
	}

	if status.ID != 0 {
		t.Errorf("ID = %v, want 0", status.ID)
	}
	if status.State != "opening" {
		t.Errorf("State = %v, want opening", status.State)
	}
	if status.CurrentPos == nil || *status.CurrentPos != pos {
		t.Errorf("CurrentPos = %v, want %v", status.CurrentPos, pos)
	}
	if status.TargetPos == nil || *status.TargetPos != targetPos {
		t.Errorf("TargetPos = %v, want %v", status.TargetPos, targetPos)
	}
}

func TestParseCoverStatusInvalid(t *testing.T) {
	data := json.RawMessage(`not valid json`)

	_, err := ParseCoverStatus(data)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseLightStatus(t *testing.T) {
	brightness := 75
	colorTemp := 4000
	data := json.RawMessage(`{
		"id": 0,
		"output": true,
		"brightness": 75,
		"rgb": {"r": 255, "g": 128, "b": 64},
		"white": 32,
		"color_temp": 4000,
		"source": "schedule"
	}`)

	status, err := ParseLightStatus(data)
	if err != nil {
		t.Fatalf("ParseLightStatus failed: %v", err)
	}

	if status.ID != 0 {
		t.Errorf("ID = %v, want 0", status.ID)
	}
	if !status.Output {
		t.Error("Output = false, want true")
	}
	if status.Brightness == nil || *status.Brightness != brightness {
		t.Errorf("Brightness = %v, want %v", status.Brightness, brightness)
	}
	if status.RGB == nil {
		t.Fatal("RGB is nil")
	}
	if status.RGB.Red != 255 {
		t.Errorf("RGB.Red = %v, want 255", status.RGB.Red)
	}
	if status.RGB.Green != 128 {
		t.Errorf("RGB.Green = %v, want 128", status.RGB.Green)
	}
	if status.RGB.Blue != 64 {
		t.Errorf("RGB.Blue = %v, want 64", status.RGB.Blue)
	}
	if status.ColorTemp == nil || *status.ColorTemp != colorTemp {
		t.Errorf("ColorTemp = %v, want %v", status.ColorTemp, colorTemp)
	}
}

func TestParseLightStatusInvalid(t *testing.T) {
	data := json.RawMessage(`not valid json`)

	_, err := ParseLightStatus(data)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseInputEvent(t *testing.T) {
	data := json.RawMessage(`{
		"id": 0,
		"event": "single_push",
		"state": true
	}`)

	event, err := ParseInputEvent(data)
	if err != nil {
		t.Fatalf("ParseInputEvent failed: %v", err)
	}

	if event.ID != 0 {
		t.Errorf("ID = %v, want 0", event.ID)
	}
	if event.Event != "single_push" {
		t.Errorf("Event = %v, want single_push", event.Event)
	}
	if !event.State {
		t.Error("State = false, want true")
	}
}

func TestParseInputEventInvalid(t *testing.T) {
	data := json.RawMessage(`not valid json`)

	_, err := ParseInputEvent(data)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestEventFilter(t *testing.T) {
	tests := []struct {
		filter *EventFilter
		msg    *WebSocketMessage
		name   string
		want   bool
	}{
		{
			name:   "empty filter matches all",
			filter: &EventFilter{},
			msg: &WebSocketMessage{
				Event:    EventDeviceOnline,
				DeviceID: "device123",
			},
			want: true,
		},
		{
			name: "device ID filter matches",
			filter: &EventFilter{
				DeviceIDs: []string{"device123", "device456"},
			},
			msg: &WebSocketMessage{
				Event:    EventDeviceOnline,
				DeviceID: "device123",
			},
			want: true,
		},
		{
			name: "device ID filter does not match",
			filter: &EventFilter{
				DeviceIDs: []string{"device456"},
			},
			msg: &WebSocketMessage{
				Event:    EventDeviceOnline,
				DeviceID: "device123",
			},
			want: false,
		},
		{
			name: "event type filter matches",
			filter: &EventFilter{
				EventTypes: []string{EventDeviceOnline, EventDeviceOffline},
			},
			msg: &WebSocketMessage{
				Event:    EventDeviceOnline,
				DeviceID: "device123",
			},
			want: true,
		},
		{
			name: "event type filter does not match",
			filter: &EventFilter{
				EventTypes: []string{EventNotifyStatus},
			},
			msg: &WebSocketMessage{
				Event:    EventDeviceOnline,
				DeviceID: "device123",
			},
			want: false,
		},
		{
			name: "combined filter matches",
			filter: &EventFilter{
				DeviceIDs:  []string{"device123"},
				EventTypes: []string{EventDeviceOnline},
			},
			msg: &WebSocketMessage{
				Event:    EventDeviceOnline,
				DeviceID: "device123",
			},
			want: true,
		},
		{
			name: "combined filter fails on device",
			filter: &EventFilter{
				DeviceIDs:  []string{"device456"},
				EventTypes: []string{EventDeviceOnline},
			},
			msg: &WebSocketMessage{
				Event:    EventDeviceOnline,
				DeviceID: "device123",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Matches(tt.msg)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilteredEventHandler(t *testing.T) {
	var called bool
	var receivedMsg *WebSocketMessage

	handler := &FilteredEventHandler{
		Filter: &EventFilter{
			DeviceIDs: []string{"device123"},
		},
		Handler: func(msg *WebSocketMessage) {
			called = true
			receivedMsg = msg
		},
	}

	// Should call handler
	msg1 := &WebSocketMessage{
		Event:    EventDeviceOnline,
		DeviceID: "device123",
	}
	handler.Handle(msg1)

	if !called {
		t.Error("Handler should be called for matching message")
	}
	if receivedMsg != msg1 {
		t.Error("Received message mismatch")
	}

	// Reset
	called = false
	receivedMsg = nil

	// Should not call handler
	msg2 := &WebSocketMessage{
		Event:    EventDeviceOnline,
		DeviceID: "device456",
	}
	handler.Handle(msg2)

	if called {
		t.Error("Handler should not be called for non-matching message")
	}
}

func TestFilteredEventHandlerNilFilter(t *testing.T) {
	var called bool

	handler := &FilteredEventHandler{
		Filter: nil,
		Handler: func(msg *WebSocketMessage) {
			called = true
		},
	}

	handler.Handle(&WebSocketMessage{
		Event:    EventDeviceOnline,
		DeviceID: "device123",
	})

	if !called {
		t.Error("Handler should be called when filter is nil")
	}
}

func TestDeviceEventJSON(t *testing.T) {
	data := `{
		"device_id": "device123",
		"event": "single_push",
		"channel": 0,
		"component": "input:0",
		"data": {"key": "value"},
		"ts": 1609459200
	}`

	var event DeviceEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if event.DeviceID != "device123" {
		t.Errorf("DeviceID = %v, want device123", event.DeviceID)
	}
	if event.Event != "single_push" {
		t.Errorf("Event = %v, want single_push", event.Event)
	}
	if event.Channel != 0 {
		t.Errorf("Channel = %v, want 0", event.Channel)
	}
	if event.Component != "input:0" {
		t.Errorf("Component = %v, want input:0", event.Component)
	}
	if event.Timestamp != 1609459200 {
		t.Errorf("Timestamp = %v, want 1609459200", event.Timestamp)
	}
}

func TestRGBValueJSON(t *testing.T) {
	data := `{"r": 255, "g": 128, "b": 64}`

	var rgb RGBValue
	if err := json.Unmarshal([]byte(data), &rgb); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if rgb.Red != 255 {
		t.Errorf("Red = %v, want 255", rgb.Red)
	}
	if rgb.Green != 128 {
		t.Errorf("Green = %v, want 128", rgb.Green)
	}
	if rgb.Blue != 64 {
		t.Errorf("Blue = %v, want 64", rgb.Blue)
	}
}
