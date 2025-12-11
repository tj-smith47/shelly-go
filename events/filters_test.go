package events

import (
	"testing"
)

func TestWithDeviceID(t *testing.T) {
	filter := WithDeviceID("device1")

	event1 := NewDeviceOnlineEvent("device1")
	event2 := NewDeviceOnlineEvent("device2")

	if !filter(event1) {
		t.Error("filter should match device1")
	}
	if filter(event2) {
		t.Error("filter should not match device2")
	}
}

func TestWithDeviceIDs(t *testing.T) {
	filter := WithDeviceIDs("device1", "device3")

	tests := []struct {
		deviceID string
		want     bool
	}{
		{"device1", true},
		{"device2", false},
		{"device3", true},
		{"device4", false},
	}

	for _, tt := range tests {
		t.Run(tt.deviceID, func(t *testing.T) {
			event := NewDeviceOnlineEvent(tt.deviceID)
			if got := filter(event); got != tt.want {
				t.Errorf("filter(%v) = %v, want %v", tt.deviceID, got, tt.want)
			}
		})
	}
}

func TestWithEventType(t *testing.T) {
	filter := WithEventType(EventTypeDeviceOnline)

	online := NewDeviceOnlineEvent("device1")
	offline := NewDeviceOfflineEvent("device1")

	if !filter(online) {
		t.Error("filter should match DeviceOnline")
	}
	if filter(offline) {
		t.Error("filter should not match DeviceOffline")
	}
}

func TestWithEventTypes(t *testing.T) {
	filter := WithEventTypes(EventTypeDeviceOnline, EventTypeDeviceOffline)

	online := NewDeviceOnlineEvent("device1")
	offline := NewDeviceOfflineEvent("device1")
	status := NewStatusChangeEvent("device1", "switch:0", nil)

	if !filter(online) {
		t.Error("filter should match DeviceOnline")
	}
	if !filter(offline) {
		t.Error("filter should match DeviceOffline")
	}
	if filter(status) {
		t.Error("filter should not match StatusChange")
	}
}

func TestWithSource(t *testing.T) {
	filter := WithSource(EventSourceCloud)

	cloudEvent := NewDeviceOnlineEvent("device1").WithSource(EventSourceCloud)
	localEvent := NewDeviceOnlineEvent("device1").WithSource(EventSourceLocal)

	if !filter(cloudEvent) {
		t.Error("filter should match cloud events")
	}
	if filter(localEvent) {
		t.Error("filter should not match local events")
	}
}

func TestWithSources(t *testing.T) {
	filter := WithSources(EventSourceLocal, EventSourceWebSocket)

	tests := []struct {
		source EventSource
		want   bool
	}{
		{EventSourceLocal, true},
		{EventSourceWebSocket, true},
		{EventSourceCloud, false},
		{EventSourceCoIoT, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.source), func(t *testing.T) {
			event := NewDeviceOnlineEvent("device1").WithSource(tt.source)
			if got := filter(event); got != tt.want {
				t.Errorf("filter(%v) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestWithComponentType(t *testing.T) {
	filter := WithComponentType("switch")

	tests := []struct {
		event     Event
		name      string
		wantMatch bool
	}{
		{name: "status switch:0", event: NewStatusChangeEvent("d1", "switch:0", nil), wantMatch: true},
		{name: "status switch:1", event: NewStatusChangeEvent("d1", "switch:1", nil), wantMatch: true},
		{name: "status cover:0", event: NewStatusChangeEvent("d1", "cover:0", nil), wantMatch: false},
		{name: "notify input:0", event: NewNotifyEvent("d1", "input:0", "push"), wantMatch: false},
		{name: "notify switch:0", event: NewNotifyEvent("d1", "switch:0", "event"), wantMatch: true},
		{name: "config switch:0", event: NewConfigChangeEvent("d1", "switch:0", nil), wantMatch: true},
		{name: "config light:0", event: NewConfigChangeEvent("d1", "light:0", nil), wantMatch: false},
		{name: "error switch:0", event: NewErrorEvent("d1", 1, "err").WithComponent("switch:0"), wantMatch: true},
		{name: "error input:0", event: NewErrorEvent("d1", 1, "err").WithComponent("input:0"), wantMatch: false},
		{name: "device online", event: NewDeviceOnlineEvent("d1"), wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filter(tt.event); got != tt.wantMatch {
				t.Errorf("filter() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestWithComponent(t *testing.T) {
	filter := WithComponent("switch:0")

	tests := []struct {
		event     Event
		name      string
		wantMatch bool
	}{
		{name: "status switch:0", event: NewStatusChangeEvent("d1", "switch:0", nil), wantMatch: true},
		{name: "status switch:1", event: NewStatusChangeEvent("d1", "switch:1", nil), wantMatch: false},
		{name: "notify switch:0", event: NewNotifyEvent("d1", "switch:0", "event"), wantMatch: true},
		{name: "config switch:0", event: NewConfigChangeEvent("d1", "switch:0", nil), wantMatch: true},
		{name: "error switch:0", event: NewErrorEvent("d1", 1, "err").WithComponent("switch:0"), wantMatch: true},
		{name: "error switch:1", event: NewErrorEvent("d1", 1, "err").WithComponent("switch:1"), wantMatch: false},
		{name: "device online", event: NewDeviceOnlineEvent("d1"), wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filter(tt.event); got != tt.wantMatch {
				t.Errorf("filter() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestWithInputEvent(t *testing.T) {
	filter := WithInputEvent(InputEventSinglePush)

	singlePush := NewNotifyEvent("d1", "input:0", InputEventSinglePush)
	doublePush := NewNotifyEvent("d1", "input:0", InputEventDoublePush)
	statusChange := NewStatusChangeEvent("d1", "input:0", nil)

	if !filter(singlePush) {
		t.Error("filter should match single_push")
	}
	if filter(doublePush) {
		t.Error("filter should not match double_push")
	}
	if filter(statusChange) {
		t.Error("filter should not match non-notify events")
	}
}

func TestAnd(t *testing.T) {
	filter := And(
		WithDeviceID("device1"),
		WithEventType(EventTypeStatusChange),
	)

	tests := []struct {
		event Event
		name  string
		want  bool
	}{
		{name: "device1 + status", event: NewStatusChangeEvent("device1", "switch:0", nil), want: true},
		{name: "device2 + status", event: NewStatusChangeEvent("device2", "switch:0", nil), want: false},
		{name: "device1 + online", event: NewDeviceOnlineEvent("device1"), want: false},
		{name: "device2 + online", event: NewDeviceOnlineEvent("device2"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filter(tt.event); got != tt.want {
				t.Errorf("filter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnd_Empty(t *testing.T) {
	filter := And()

	// Empty And should match everything
	if !filter(NewDeviceOnlineEvent("device1")) {
		t.Error("empty And should match all events")
	}
}

func TestOr(t *testing.T) {
	filter := Or(
		WithDeviceID("device1"),
		WithDeviceID("device2"),
	)

	tests := []struct {
		deviceID string
		want     bool
	}{
		{"device1", true},
		{"device2", true},
		{"device3", false},
	}

	for _, tt := range tests {
		t.Run(tt.deviceID, func(t *testing.T) {
			event := NewDeviceOnlineEvent(tt.deviceID)
			if got := filter(event); got != tt.want {
				t.Errorf("filter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOr_Empty(t *testing.T) {
	filter := Or()

	// Empty Or should match nothing
	if filter(NewDeviceOnlineEvent("device1")) {
		t.Error("empty Or should not match any events")
	}
}

func TestNot(t *testing.T) {
	filter := Not(WithDeviceID("device1"))

	device1 := NewDeviceOnlineEvent("device1")
	device2 := NewDeviceOnlineEvent("device2")

	if filter(device1) {
		t.Error("Not filter should not match device1")
	}
	if !filter(device2) {
		t.Error("Not filter should match device2")
	}
}

func TestStatusChange_Shorthand(t *testing.T) {
	filter := StatusChange()

	status := NewStatusChangeEvent("d1", "switch:0", nil)
	online := NewDeviceOnlineEvent("d1")

	if !filter(status) {
		t.Error("StatusChange() should match status change events")
	}
	if filter(online) {
		t.Error("StatusChange() should not match online events")
	}
}

func TestDeviceOnline_Shorthand(t *testing.T) {
	filter := DeviceOnline()

	online := NewDeviceOnlineEvent("d1")
	offline := NewDeviceOfflineEvent("d1")

	if !filter(online) {
		t.Error("DeviceOnline() should match online events")
	}
	if filter(offline) {
		t.Error("DeviceOnline() should not match offline events")
	}
}

func TestDeviceOffline_Shorthand(t *testing.T) {
	filter := DeviceOffline()

	online := NewDeviceOnlineEvent("d1")
	offline := NewDeviceOfflineEvent("d1")

	if filter(online) {
		t.Error("DeviceOffline() should not match online events")
	}
	if !filter(offline) {
		t.Error("DeviceOffline() should match offline events")
	}
}

func TestInputEvents_Shorthand(t *testing.T) {
	filter := InputEvents()

	notify := NewNotifyEvent("d1", "input:0", InputEventSinglePush)
	status := NewStatusChangeEvent("d1", "input:0", nil)

	if !filter(notify) {
		t.Error("InputEvents() should match notify events")
	}
	if filter(status) {
		t.Error("InputEvents() should not match status events")
	}
}

func TestErrors_Shorthand(t *testing.T) {
	filter := Errors()

	errorEvent := NewErrorEvent("d1", 500, "error")
	statusEvent := NewStatusChangeEvent("d1", "switch:0", nil)

	if !filter(errorEvent) {
		t.Error("Errors() should match error events")
	}
	if filter(statusEvent) {
		t.Error("Errors() should not match status events")
	}
}

func TestFromCloud_Shorthand(t *testing.T) {
	filter := FromCloud()

	cloudEvent := NewDeviceOnlineEvent("d1").WithSource(EventSourceCloud)
	localEvent := NewDeviceOnlineEvent("d1").WithSource(EventSourceLocal)

	if !filter(cloudEvent) {
		t.Error("FromCloud() should match cloud events")
	}
	if filter(localEvent) {
		t.Error("FromCloud() should not match local events")
	}
}

func TestFromLocal_Shorthand(t *testing.T) {
	filter := FromLocal()

	tests := []struct {
		source EventSource
		want   bool
	}{
		{EventSourceLocal, true},
		{EventSourceWebSocket, true},
		{EventSourceCoIoT, true},
		{EventSourceMQTT, true},
		{EventSourceCloud, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.source), func(t *testing.T) {
			event := NewDeviceOnlineEvent("d1").WithSource(tt.source)
			if got := filter(event); got != tt.want {
				t.Errorf("FromLocal() for %v = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestComplexFilter(t *testing.T) {
	// Match switch status changes from device1 or device2 that are not from cloud
	filter := And(
		Or(WithDeviceID("device1"), WithDeviceID("device2")),
		StatusChange(),
		WithComponentType("switch"),
		Not(FromCloud()),
	)

	tests := []struct {
		event Event
		name  string
		want  bool
	}{
		{
			name:  "device1 switch local",
			event: NewStatusChangeEvent("device1", "switch:0", nil).WithSource(EventSourceLocal),
			want:  true,
		},
		{
			name:  "device2 switch websocket",
			event: NewStatusChangeEvent("device2", "switch:1", nil).WithSource(EventSourceWebSocket),
			want:  true,
		},
		{
			name:  "device1 switch cloud",
			event: NewStatusChangeEvent("device1", "switch:0", nil).WithSource(EventSourceCloud),
			want:  false,
		},
		{
			name:  "device3 switch local",
			event: NewStatusChangeEvent("device3", "switch:0", nil).WithSource(EventSourceLocal),
			want:  false,
		},
		{
			name:  "device1 cover local",
			event: NewStatusChangeEvent("device1", "cover:0", nil).WithSource(EventSourceLocal),
			want:  false,
		},
		{
			name:  "device1 online local",
			event: NewDeviceOnlineEvent("device1").WithSource(EventSourceLocal),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filter(tt.event); got != tt.want {
				t.Errorf("filter() = %v, want %v", got, tt.want)
			}
		})
	}
}
