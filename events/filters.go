package events

import "strings"

// Filter is a function that determines if an event should be processed.
type Filter func(Event) bool

// WithDeviceID creates a filter that matches events from a specific device.
func WithDeviceID(deviceID string) Filter {
	return func(e Event) bool {
		return e.DeviceID() == deviceID
	}
}

// WithDeviceIDs creates a filter that matches events from any of the specified devices.
func WithDeviceIDs(deviceIDs ...string) Filter {
	idSet := make(map[string]bool, len(deviceIDs))
	for _, id := range deviceIDs {
		idSet[id] = true
	}
	return func(e Event) bool {
		return idSet[e.DeviceID()]
	}
}

// WithEventType creates a filter that matches events of a specific type.
func WithEventType(eventType EventType) Filter {
	return func(e Event) bool {
		return e.Type() == eventType
	}
}

// WithEventTypes creates a filter that matches events of any of the specified types.
func WithEventTypes(eventTypes ...EventType) Filter {
	typeSet := make(map[EventType]bool, len(eventTypes))
	for _, t := range eventTypes {
		typeSet[t] = true
	}
	return func(e Event) bool {
		return typeSet[e.Type()]
	}
}

// WithSource creates a filter that matches events from a specific source.
func WithSource(source EventSource) Filter {
	return func(e Event) bool {
		return e.Source() == source
	}
}

// WithSources creates a filter that matches events from any of the specified sources.
func WithSources(sources ...EventSource) Filter {
	sourceSet := make(map[EventSource]bool, len(sources))
	for _, s := range sources {
		sourceSet[s] = true
	}
	return func(e Event) bool {
		return sourceSet[e.Source()]
	}
}

// WithComponentType creates a filter that matches events for a specific component type.
// Matches components like "switch:0", "switch:1" when componentType is "switch".
func WithComponentType(componentType string) Filter {
	return func(e Event) bool {
		switch evt := e.(type) {
		case *StatusChangeEvent:
			return strings.HasPrefix(evt.Component, componentType)
		case *NotifyEvent:
			return strings.HasPrefix(evt.Component, componentType)
		case *ConfigChangeEvent:
			return strings.HasPrefix(evt.Component, componentType)
		case *ErrorEvent:
			return strings.HasPrefix(evt.Component, componentType)
		default:
			return false
		}
	}
}

// WithComponent creates a filter that matches events for a specific component.
// Matches exact component identifiers like "switch:0".
func WithComponent(component string) Filter {
	return func(e Event) bool {
		switch evt := e.(type) {
		case *StatusChangeEvent:
			return evt.Component == component
		case *NotifyEvent:
			return evt.Component == component
		case *ConfigChangeEvent:
			return evt.Component == component
		case *ErrorEvent:
			return evt.Component == component
		default:
			return false
		}
	}
}

// WithInputEvent creates a filter that matches specific input events.
// Use with InputEventSinglePush, InputEventDoublePush, etc.
func WithInputEvent(event string) Filter {
	return func(e Event) bool {
		if evt, ok := e.(*NotifyEvent); ok {
			return evt.Event == event
		}
		return false
	}
}

// And combines multiple filters with AND logic.
// All filters must match for the event to be accepted.
func And(filters ...Filter) Filter {
	return func(e Event) bool {
		for _, f := range filters {
			if !f(e) {
				return false
			}
		}
		return true
	}
}

// Or combines multiple filters with OR logic.
// At least one filter must match for the event to be accepted.
func Or(filters ...Filter) Filter {
	return func(e Event) bool {
		for _, f := range filters {
			if f(e) {
				return true
			}
		}
		return false
	}
}

// Not negates a filter.
func Not(filter Filter) Filter {
	return func(e Event) bool {
		return !filter(e)
	}
}

// StatusChange is a shorthand filter for status change events.
func StatusChange() Filter {
	return WithEventType(EventTypeStatusChange)
}

// DeviceOnline is a shorthand filter for device online events.
func DeviceOnline() Filter {
	return WithEventType(EventTypeDeviceOnline)
}

// DeviceOffline is a shorthand filter for device offline events.
func DeviceOffline() Filter {
	return WithEventType(EventTypeDeviceOffline)
}

// InputEvents is a shorthand filter for notify events (input/button events).
func InputEvents() Filter {
	return WithEventType(EventTypeNotify)
}

// Errors is a shorthand filter for error events.
func Errors() Filter {
	return WithEventType(EventTypeError)
}

// FromCloud is a shorthand filter for events from the cloud.
func FromCloud() Filter {
	return WithSource(EventSourceCloud)
}

// FromLocal is a shorthand filter for events from local connections.
func FromLocal() Filter {
	return Or(
		WithSource(EventSourceLocal),
		WithSource(EventSourceWebSocket),
		WithSource(EventSourceCoIoT),
		WithSource(EventSourceMQTT),
	)
}
