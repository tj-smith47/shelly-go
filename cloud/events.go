package cloud

import (
	"encoding/json"
	"sync"
)

// EventHandlers manages event handlers for WebSocket events.
type EventHandlers struct {
	// onDeviceOnline is called when a device comes online.
	onDeviceOnline []func(deviceID string)

	// onDeviceOffline is called when a device goes offline.
	onDeviceOffline []func(deviceID string)

	// onStatusChange is called when a device status changes.
	onStatusChange []func(deviceID string, status json.RawMessage)

	// onNotifyStatus is called for Gen2+ NotifyStatus events.
	onNotifyStatus []func(deviceID string, status json.RawMessage)

	// onNotifyFullStatus is called for Gen2+ NotifyFullStatus events.
	onNotifyFullStatus []func(deviceID string, status json.RawMessage)

	// onNotifyEvent is called for Gen2+ NotifyEvent events.
	onNotifyEvent []func(deviceID string, event json.RawMessage)

	// onMessage is called for all messages.
	onMessage []func(msg *WebSocketMessage)

	// mu protects handler registration.
	mu sync.RWMutex
}

// NewEventHandlers creates a new EventHandlers instance.
func NewEventHandlers() *EventHandlers {
	return &EventHandlers{}
}

// OnDeviceOnline registers a handler for device online events.
func (h *EventHandlers) OnDeviceOnline(handler func(deviceID string)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onDeviceOnline = append(h.onDeviceOnline, handler)
}

// OnDeviceOffline registers a handler for device offline events.
func (h *EventHandlers) OnDeviceOffline(handler func(deviceID string)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onDeviceOffline = append(h.onDeviceOffline, handler)
}

// OnStatusChange registers a handler for device status change events.
func (h *EventHandlers) OnStatusChange(handler func(deviceID string, status json.RawMessage)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onStatusChange = append(h.onStatusChange, handler)
}

// OnNotifyStatus registers a handler for Gen2+ NotifyStatus events.
func (h *EventHandlers) OnNotifyStatus(handler func(deviceID string, status json.RawMessage)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onNotifyStatus = append(h.onNotifyStatus, handler)
}

// OnNotifyFullStatus registers a handler for Gen2+ NotifyFullStatus events.
func (h *EventHandlers) OnNotifyFullStatus(handler func(deviceID string, status json.RawMessage)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onNotifyFullStatus = append(h.onNotifyFullStatus, handler)
}

// OnNotifyEvent registers a handler for Gen2+ NotifyEvent events.
func (h *EventHandlers) OnNotifyEvent(handler func(deviceID string, event json.RawMessage)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onNotifyEvent = append(h.onNotifyEvent, handler)
}

// OnMessage registers a handler for all messages.
func (h *EventHandlers) OnMessage(handler func(msg *WebSocketMessage)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onMessage = append(h.onMessage, handler)
}

// Dispatch dispatches a message to the appropriate handlers.
func (h *EventHandlers) Dispatch(msg *WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Call all message handlers
	for _, handler := range h.onMessage {
		handler(msg)
	}

	// Dispatch based on event type
	switch msg.Event {
	case EventDeviceOnline:
		for _, handler := range h.onDeviceOnline {
			handler(msg.DeviceID)
		}

	case EventDeviceOffline:
		for _, handler := range h.onDeviceOffline {
			handler(msg.DeviceID)
		}

	case EventDeviceStatusChange:
		for _, handler := range h.onStatusChange {
			handler(msg.DeviceID, msg.Status)
		}

	case EventNotifyStatus:
		for _, handler := range h.onNotifyStatus {
			handler(msg.DeviceID, msg.Status)
		}

	case EventNotifyFullStatus:
		for _, handler := range h.onNotifyFullStatus {
			handler(msg.DeviceID, msg.Status)
		}

	case EventNotifyEvent:
		for _, handler := range h.onNotifyEvent {
			handler(msg.DeviceID, msg.Data)
		}
	}
}

// Clear removes all registered handlers.
func (h *EventHandlers) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.onDeviceOnline = nil
	h.onDeviceOffline = nil
	h.onStatusChange = nil
	h.onNotifyStatus = nil
	h.onNotifyFullStatus = nil
	h.onNotifyEvent = nil
	h.onMessage = nil
}

// DeviceEvent represents a device event from the WebSocket.
type DeviceEvent struct {
	DeviceID  string          `json:"device_id"`
	Event     string          `json:"event"`
	Component string          `json:"component,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Channel   int             `json:"channel,omitempty"`
	Timestamp int64           `json:"ts,omitempty"`
}

// SwitchStatusEvent represents a switch status change event.
type SwitchStatusEvent struct {
	Source         string  `json:"source,omitempty"`
	ID             int     `json:"id"`
	TimerStartedAt float64 `json:"timer_started_at,omitempty"`
	TimerDuration  float64 `json:"timer_duration,omitempty"`
	Output         bool    `json:"output"`
}

// CoverStatusEvent represents a cover status change event.
type CoverStatusEvent struct {
	CurrentPos *int   `json:"current_pos,omitempty"`
	TargetPos  *int   `json:"target_pos,omitempty"`
	State      string `json:"state"`
	Source     string `json:"source,omitempty"`
	ID         int    `json:"id"`
}

// LightStatusEvent represents a light status change event.
type LightStatusEvent struct {
	Brightness *int      `json:"brightness,omitempty"`
	RGB        *RGBValue `json:"rgb,omitempty"`
	White      *int      `json:"white,omitempty"`
	ColorTemp  *int      `json:"color_temp,omitempty"`
	Source     string    `json:"source,omitempty"`
	ID         int       `json:"id"`
	Output     bool      `json:"output"`
}

// RGBValue represents RGB color values.
type RGBValue struct {
	// Red is the red channel (0-255).
	Red int `json:"r"`

	// Green is the green channel (0-255).
	Green int `json:"g"`

	// Blue is the blue channel (0-255).
	Blue int `json:"b"`
}

// InputEvent represents an input event.
type InputEvent struct {
	Event string `json:"event"`
	ID    int    `json:"id"`
	State bool   `json:"state,omitempty"`
}

// ParseSwitchStatus parses a switch status from raw JSON.
func ParseSwitchStatus(data json.RawMessage) (*SwitchStatusEvent, error) {
	var status SwitchStatusEvent
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// ParseCoverStatus parses a cover status from raw JSON.
func ParseCoverStatus(data json.RawMessage) (*CoverStatusEvent, error) {
	var status CoverStatusEvent
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// ParseLightStatus parses a light status from raw JSON.
func ParseLightStatus(data json.RawMessage) (*LightStatusEvent, error) {
	var status LightStatusEvent
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// ParseInputEvent parses an input event from raw JSON.
func ParseInputEvent(data json.RawMessage) (*InputEvent, error) {
	var event InputEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// EventFilter allows filtering events based on criteria.
type EventFilter struct {
	// DeviceIDs filters events by device ID.
	DeviceIDs []string

	// EventTypes filters events by event type.
	EventTypes []string

	// Components filters events by component name (Gen2+).
	Components []string
}

// Matches returns true if the message matches the filter criteria.
func (f *EventFilter) Matches(msg *WebSocketMessage) bool {
	// Check device ID filter
	if len(f.DeviceIDs) > 0 {
		found := false
		for _, id := range f.DeviceIDs {
			if id == msg.DeviceID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check event type filter
	if len(f.EventTypes) > 0 {
		found := false
		for _, et := range f.EventTypes {
			if et == msg.Event {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// FilteredEventHandler wraps a handler with a filter.
type FilteredEventHandler struct {
	// Filter is the event filter.
	Filter *EventFilter

	// Handler is the message handler.
	Handler func(msg *WebSocketMessage)
}

// Handle processes a message if it matches the filter.
func (h *FilteredEventHandler) Handle(msg *WebSocketMessage) {
	if h.Filter == nil || h.Filter.Matches(msg) {
		h.Handler(msg)
	}
}
