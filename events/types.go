package events

import (
	"encoding/json"
	"time"
)

// EventType identifies the type of event.
type EventType string

const (
	// EventTypeStatusChange indicates a component status change.
	EventTypeStatusChange EventType = "status_change"

	// EventTypeFullStatus indicates a full device status update.
	EventTypeFullStatus EventType = "full_status"

	// EventTypeNotify indicates an input/button event.
	EventTypeNotify EventType = "notify_event"

	// EventTypeDeviceOnline indicates a device came online.
	EventTypeDeviceOnline EventType = "device_online"

	// EventTypeDeviceOffline indicates a device went offline.
	EventTypeDeviceOffline EventType = "device_offline"

	// EventTypeUpdateAvailable indicates a firmware update is available.
	EventTypeUpdateAvailable EventType = "update_available"

	// EventTypeScript indicates a script output event.
	EventTypeScript EventType = "script"

	// EventTypeConfig indicates a configuration change.
	EventTypeConfig EventType = "config_change"

	// EventTypeError indicates an error event.
	EventTypeError EventType = "error"
)

// Event is the interface implemented by all event types.
type Event interface {
	// Type returns the event type.
	Type() EventType

	// DeviceID returns the device identifier.
	DeviceID() string

	// Timestamp returns when the event occurred.
	Timestamp() time.Time

	// Source returns the event source (local, cloud, etc.).
	Source() EventSource
}

// EventSource indicates where the event originated.
type EventSource string

const (
	// EventSourceLocal indicates the event came from a local device connection.
	EventSourceLocal EventSource = "local"

	// EventSourceCloud indicates the event came from the Shelly Cloud.
	EventSourceCloud EventSource = "cloud"

	// EventSourceCoIoT indicates the event came from Gen1 CoIoT multicast.
	EventSourceCoIoT EventSource = "coiot"

	// EventSourceWebSocket indicates the event came from a WebSocket connection.
	EventSourceWebSocket EventSource = "websocket"

	// EventSourceMQTT indicates the event came from MQTT.
	EventSourceMQTT EventSource = "mqtt"
)

// BaseEvent provides common fields for all events.
type BaseEvent struct {
	eventType EventType
	deviceID  string
	timestamp time.Time
	source    EventSource
}

// Type returns the event type.
func (e *BaseEvent) Type() EventType {
	return e.eventType
}

// DeviceID returns the device identifier.
func (e *BaseEvent) DeviceID() string {
	return e.deviceID
}

// Timestamp returns when the event occurred.
func (e *BaseEvent) Timestamp() time.Time {
	return e.timestamp
}

// Source returns the event source.
func (e *BaseEvent) Source() EventSource {
	return e.source
}

// StatusChangeEvent represents a component status change.
type StatusChangeEvent struct {
	BaseEvent

	// Component is the component that changed (e.g., "switch:0", "cover:0").
	Component string `json:"component"`

	// Status contains the component status data.
	Status json.RawMessage `json:"status"`

	// Delta contains only the changed fields (Gen2+).
	Delta json.RawMessage `json:"delta,omitempty"`
}

// NewStatusChangeEvent creates a new status change event.
func NewStatusChangeEvent(deviceID, component string, status json.RawMessage) *StatusChangeEvent {
	return &StatusChangeEvent{
		BaseEvent: BaseEvent{
			eventType: EventTypeStatusChange,
			deviceID:  deviceID,
			timestamp: time.Now(),
			source:    EventSourceLocal,
		},
		Component: component,
		Status:    status,
	}
}

// WithSource sets the event source.
func (e *StatusChangeEvent) WithSource(source EventSource) *StatusChangeEvent {
	e.source = source
	return e
}

// WithDelta sets the delta data.
func (e *StatusChangeEvent) WithDelta(delta json.RawMessage) *StatusChangeEvent {
	e.Delta = delta
	return e
}

// FullStatusEvent represents a full device status notification.
type FullStatusEvent struct {
	BaseEvent

	// Status contains the complete device status.
	Status json.RawMessage `json:"status"`
}

// NewFullStatusEvent creates a new full status event.
func NewFullStatusEvent(deviceID string, status json.RawMessage) *FullStatusEvent {
	return &FullStatusEvent{
		BaseEvent: BaseEvent{
			eventType: EventTypeFullStatus,
			deviceID:  deviceID,
			timestamp: time.Now(),
			source:    EventSourceLocal,
		},
		Status: status,
	}
}

// WithSource sets the event source.
func (e *FullStatusEvent) WithSource(source EventSource) *FullStatusEvent {
	e.source = source
	return e
}

// NotifyEvent represents an input/button event.
type NotifyEvent struct {
	BaseEvent

	// Component is the component that triggered the event (e.g., "input:0").
	Component string `json:"component"`

	// Event is the event name (e.g., "single_push", "double_push", "long_push").
	Event string `json:"event"`

	// Data contains additional event data.
	Data json.RawMessage `json:"data,omitempty"`
}

// Common input events.
const (
	InputEventSinglePush = "single_push"
	InputEventDoublePush = "double_push"
	InputEventTriplePush = "triple_push"
	InputEventLongPush   = "long_push"
	InputEventBtnDown    = "btn_down"
	InputEventBtnUp      = "btn_up"
)

// NewNotifyEvent creates a new notify event.
func NewNotifyEvent(deviceID, component, event string) *NotifyEvent {
	return &NotifyEvent{
		BaseEvent: BaseEvent{
			eventType: EventTypeNotify,
			deviceID:  deviceID,
			timestamp: time.Now(),
			source:    EventSourceLocal,
		},
		Component: component,
		Event:     event,
	}
}

// WithSource sets the event source.
func (e *NotifyEvent) WithSource(source EventSource) *NotifyEvent {
	e.source = source
	return e
}

// WithData sets the event data.
func (e *NotifyEvent) WithData(data json.RawMessage) *NotifyEvent {
	e.Data = data
	return e
}

// DeviceOnlineEvent indicates a device came online.
type DeviceOnlineEvent struct {
	BaseEvent

	// Address is the device's IP address.
	Address string `json:"address,omitempty"`
}

// NewDeviceOnlineEvent creates a new device online event.
func NewDeviceOnlineEvent(deviceID string) *DeviceOnlineEvent {
	return &DeviceOnlineEvent{
		BaseEvent: BaseEvent{
			eventType: EventTypeDeviceOnline,
			deviceID:  deviceID,
			timestamp: time.Now(),
			source:    EventSourceLocal,
		},
	}
}

// WithAddress sets the device address.
func (e *DeviceOnlineEvent) WithAddress(address string) *DeviceOnlineEvent {
	e.Address = address
	return e
}

// WithSource sets the event source.
func (e *DeviceOnlineEvent) WithSource(source EventSource) *DeviceOnlineEvent {
	e.source = source
	return e
}

// DeviceOfflineEvent indicates a device went offline.
type DeviceOfflineEvent struct {
	BaseEvent

	// Reason optionally describes why the device went offline.
	Reason string `json:"reason,omitempty"`
}

// NewDeviceOfflineEvent creates a new device offline event.
func NewDeviceOfflineEvent(deviceID string) *DeviceOfflineEvent {
	return &DeviceOfflineEvent{
		BaseEvent: BaseEvent{
			eventType: EventTypeDeviceOffline,
			deviceID:  deviceID,
			timestamp: time.Now(),
			source:    EventSourceLocal,
		},
	}
}

// WithReason sets the offline reason.
func (e *DeviceOfflineEvent) WithReason(reason string) *DeviceOfflineEvent {
	e.Reason = reason
	return e
}

// WithSource sets the event source.
func (e *DeviceOfflineEvent) WithSource(source EventSource) *DeviceOfflineEvent {
	e.source = source
	return e
}

// UpdateAvailableEvent indicates a firmware update is available.
type UpdateAvailableEvent struct {
	BaseEvent

	// CurrentVersion is the currently installed version.
	CurrentVersion string `json:"current_version"`

	// AvailableVersion is the version available for update.
	AvailableVersion string `json:"available_version"`

	// Stage is the update stage (stable, beta).
	Stage string `json:"stage,omitempty"`
}

// NewUpdateAvailableEvent creates a new update available event.
func NewUpdateAvailableEvent(deviceID, currentVersion, availableVersion string) *UpdateAvailableEvent {
	return &UpdateAvailableEvent{
		BaseEvent: BaseEvent{
			eventType: EventTypeUpdateAvailable,
			deviceID:  deviceID,
			timestamp: time.Now(),
			source:    EventSourceLocal,
		},
		CurrentVersion:   currentVersion,
		AvailableVersion: availableVersion,
	}
}

// WithStage sets the update stage.
func (e *UpdateAvailableEvent) WithStage(stage string) *UpdateAvailableEvent {
	e.Stage = stage
	return e
}

// WithSource sets the event source.
func (e *UpdateAvailableEvent) WithSource(source EventSource) *UpdateAvailableEvent {
	e.source = source
	return e
}

// ScriptEvent represents output from a device script.
type ScriptEvent struct {
	BaseEvent
	Output   string `json:"output"`
	ScriptID int    `json:"script_id"`
}

// NewScriptEvent creates a new script event.
func NewScriptEvent(deviceID string, scriptID int, output string) *ScriptEvent {
	return &ScriptEvent{
		BaseEvent: BaseEvent{
			eventType: EventTypeScript,
			deviceID:  deviceID,
			timestamp: time.Now(),
			source:    EventSourceLocal,
		},
		ScriptID: scriptID,
		Output:   output,
	}
}

// WithSource sets the event source.
func (e *ScriptEvent) WithSource(source EventSource) *ScriptEvent {
	e.source = source
	return e
}

// ConfigChangeEvent represents a configuration change.
type ConfigChangeEvent struct {
	BaseEvent

	// Component is the component that changed.
	Component string `json:"component"`

	// Config contains the new configuration.
	Config json.RawMessage `json:"config"`
}

// NewConfigChangeEvent creates a new config change event.
func NewConfigChangeEvent(deviceID, component string, config json.RawMessage) *ConfigChangeEvent {
	return &ConfigChangeEvent{
		BaseEvent: BaseEvent{
			eventType: EventTypeConfig,
			deviceID:  deviceID,
			timestamp: time.Now(),
			source:    EventSourceLocal,
		},
		Component: component,
		Config:    config,
	}
}

// WithSource sets the event source.
func (e *ConfigChangeEvent) WithSource(source EventSource) *ConfigChangeEvent {
	e.source = source
	return e
}

// ErrorEvent represents an error from a device.
type ErrorEvent struct {
	BaseEvent
	Message   string `json:"message"`
	Component string `json:"component,omitempty"`
	Code      int    `json:"code"`
}

// NewErrorEvent creates a new error event.
func NewErrorEvent(deviceID string, code int, message string) *ErrorEvent {
	return &ErrorEvent{
		BaseEvent: BaseEvent{
			eventType: EventTypeError,
			deviceID:  deviceID,
			timestamp: time.Now(),
			source:    EventSourceLocal,
		},
		Code:    code,
		Message: message,
	}
}

// WithComponent sets the error component.
func (e *ErrorEvent) WithComponent(component string) *ErrorEvent {
	e.Component = component
	return e
}

// WithSource sets the event source.
func (e *ErrorEvent) WithSource(source EventSource) *ErrorEvent {
	e.source = source
	return e
}
