package notifications

import (
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/events"
)

// ParseGen2Notification parses a Gen2+ WebSocket/MQTT notification into typed events.
//
// The notification should be a JSON-RPC 2.0 notification frame as sent by the device.
//
// For NotifyStatus: Returns one StatusChangeEvent per component in the notification.
// For NotifyFullStatus: Returns a single FullStatusEvent.
// For NotifyEvent: Returns one NotifyEvent per event in the notification.
//
// Example:
//
//	msg := []byte(`{"src":"shellyplus1-aabbcc","method":"NotifyStatus","params":{"switch:0":{"output":true}}}`)
//	evts, err := ParseGen2Notification("shellyplus1-aabbcc", msg)
//	if err == nil {
//	    for _, e := range evts {
//	        fmt.Printf("Event type: %s\n", e.Type())
//	    }
//	}
func ParseGen2Notification(deviceID string, data []byte) ([]events.Event, error) {
	var notif GenericNotification
	if err := json.Unmarshal(data, &notif); err != nil {
		return nil, fmt.Errorf("failed to parse notification: %w", err)
	}

	// Use src as device ID if not provided
	if deviceID == "" && notif.Src != "" {
		deviceID = notif.Src
	}

	switch notif.Method {
	case MethodNotifyStatus:
		return parseStatusNotification(deviceID, notif.Params)
	case MethodNotifyFullStatus:
		return parseFullStatusNotification(deviceID, notif.Params)
	case MethodNotifyEvent:
		return parseEventNotification(deviceID, notif.Params)
	default:
		return nil, fmt.Errorf("unknown notification method: %s", notif.Method)
	}
}

// parseStatusNotification parses NotifyStatus params into StatusChangeEvents.
func parseStatusNotification(deviceID string, params json.RawMessage) ([]events.Event, error) {
	var statusParams StatusNotificationParams
	if err := json.Unmarshal(params, &statusParams); err != nil {
		return nil, fmt.Errorf("failed to parse status params: %w", err)
	}

	evts := make([]events.Event, 0, len(statusParams))
	for component, status := range statusParams {
		evt := events.NewStatusChangeEvent(deviceID, component, status).
			WithSource(events.EventSourceWebSocket).
			WithDelta(status)
		evts = append(evts, evt)
	}

	return evts, nil
}

// parseFullStatusNotification parses NotifyFullStatus params into a FullStatusEvent.
func parseFullStatusNotification(deviceID string, params json.RawMessage) ([]events.Event, error) {
	evt := events.NewFullStatusEvent(deviceID, params).
		WithSource(events.EventSourceWebSocket)
	return []events.Event{evt}, nil
}

// parseEventNotification parses NotifyEvent params into NotifyEvents.
func parseEventNotification(deviceID string, params json.RawMessage) ([]events.Event, error) {
	var eventParams EventNotificationParams
	if err := json.Unmarshal(params, &eventParams); err != nil {
		return nil, fmt.Errorf("failed to parse event params: %w", err)
	}

	evts := make([]events.Event, 0, len(eventParams.Events))
	for _, e := range eventParams.Events {
		evt := events.NewNotifyEvent(deviceID, e.Component, e.Event).
			WithSource(events.EventSourceWebSocket).
			WithData(e.Data)
		evts = append(evts, evt)
	}

	return evts, nil
}

// ParseGen2NotificationJSON is a convenience function that parses from json.RawMessage.
func ParseGen2NotificationJSON(deviceID string, data json.RawMessage) ([]events.Event, error) {
	return ParseGen2Notification(deviceID, []byte(data))
}

// IsNotification checks if the data appears to be a notification message.
//
// Notifications have a "method" field with one of the NotifyXxx methods
// and do not have an "id" field (unlike RPC responses).
func IsNotification(data []byte) bool {
	var check struct {
		ID     *int   `json:"id"`
		Method string `json:"method"`
	}
	if err := json.Unmarshal(data, &check); err != nil {
		return false
	}
	// Notifications have method but no id
	if check.ID != nil {
		return false
	}
	switch NotificationMethod(check.Method) {
	case MethodNotifyStatus, MethodNotifyFullStatus, MethodNotifyEvent:
		return true
	default:
		return false
	}
}

// GetNotificationMethod extracts the notification method from raw data.
//
// Returns empty string if the data is not a valid notification.
func GetNotificationMethod(data []byte) NotificationMethod {
	var check struct {
		Method NotificationMethod `json:"method"`
	}
	if err := json.Unmarshal(data, &check); err != nil {
		return ""
	}
	return check.Method
}

// GetNotificationSource extracts the source device ID from raw notification data.
//
// Returns empty string if the data does not contain a source.
func GetNotificationSource(data []byte) string {
	var check struct {
		Src string `json:"src"`
	}
	if err := json.Unmarshal(data, &check); err != nil {
		return ""
	}
	return check.Src
}
