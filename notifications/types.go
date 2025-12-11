package notifications

import "encoding/json"

// NotificationMethod identifies the type of Gen2+ notification.
type NotificationMethod string

const (
	// MethodNotifyStatus indicates a component status change notification.
	// Params contain partial status updates that should be overlaid on existing status.
	MethodNotifyStatus NotificationMethod = "NotifyStatus"

	// MethodNotifyFullStatus indicates a full device status notification.
	// Params contain the complete status of all components.
	MethodNotifyFullStatus NotificationMethod = "NotifyFullStatus"

	// MethodNotifyEvent indicates an input/button event notification.
	// Params contain the event details (component, event type, timestamp).
	MethodNotifyEvent NotificationMethod = "NotifyEvent"
)

// GenericNotification represents a raw Gen2+ RPC notification frame.
//
// Gen2+ devices send notifications in JSON-RPC 2.0 format:
//
//	{
//	    "src": "shellyplus1-aabbcc",
//	    "dst": "client",
//	    "method": "NotifyStatus",
//	    "params": {"switch:0": {"output": true}}
//	}
type GenericNotification struct {
	// Src is the source device identifier.
	Src string `json:"src"`

	// Dst is the destination (usually "client" or specific target).
	Dst string `json:"dst,omitempty"`

	// Method is the notification type (NotifyStatus, NotifyFullStatus, NotifyEvent).
	Method NotificationMethod `json:"method"`

	// Params contains the notification payload.
	Params json.RawMessage `json:"params"`
}

// StatusNotificationParams represents parameters for NotifyStatus.
// The keys are component identifiers (e.g., "switch:0", "input:0").
// The values contain the changed status fields.
type StatusNotificationParams map[string]json.RawMessage

// EventNotificationParams represents parameters for NotifyEvent.
type EventNotificationParams struct {
	Events []EventNotificationEvent `json:"events"`
	Ts     float64                  `json:"ts"`
}

// EventNotificationEvent represents a single event within NotifyEvent.
type EventNotificationEvent struct {
	Component string          `json:"component"`
	Event     string          `json:"event"`
	Data      json.RawMessage `json:"data,omitempty"`
	ID        int             `json:"id"`
}

// FullStatusNotificationParams represents parameters for NotifyFullStatus.
// This is a map of component keys to their complete status objects.
type FullStatusNotificationParams map[string]json.RawMessage
