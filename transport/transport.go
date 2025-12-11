package transport

import (
	"context"
	"encoding/json"
	"errors"
)

// Transport defines the interface for communicating with Shelly devices.
// Implementations handle different protocols (HTTP, WebSocket, MQTT, CoAP).
//
// Transport implementations must be safe for concurrent use.
type Transport interface {
	// Call executes a method call with the given parameters and returns the response.
	// For Gen2+ RPC: method is the RPC method name (e.g., "Switch.Set")
	// For Gen1 REST: method is the HTTP path (e.g., "/relay/0")
	//
	// The params can be nil, a struct, or a map[string]any.
	// Returns the raw JSON response or an error.
	Call(ctx context.Context, method string, params any) (json.RawMessage, error)

	// Close closes the transport connection and releases resources.
	// After Close is called, the transport cannot be used again.
	Close() error
}

// Subscriber is an optional interface for transports that support
// real-time notifications (WebSocket, MQTT, CoAP).
type Subscriber interface {
	Transport

	// Subscribe registers a handler for incoming notifications.
	// The handler will be called for each notification received.
	Subscribe(handler NotificationHandler) error

	// Unsubscribe removes the notification handler.
	Unsubscribe() error
}

// NotificationHandler is called when a notification is received.
// The data contains the raw JSON notification.
type NotificationHandler func(data json.RawMessage)

// ConnectionState represents the connection state for stateful transports.
type ConnectionState int

const (
	// StateDisconnected indicates the transport is not connected.
	StateDisconnected ConnectionState = iota

	// StateConnecting indicates the transport is establishing a connection.
	StateConnecting

	// StateConnected indicates the transport is connected and ready.
	StateConnected

	// StateReconnecting indicates the transport is attempting to reconnect.
	StateReconnecting

	// StateClosed indicates the transport has been closed.
	StateClosed
)

// String returns the string representation of the connection state.
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateReconnecting:
		return "Reconnecting"
	case StateClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// Stateful is an optional interface for transports that maintain a connection state.
type Stateful interface {
	// State returns the current connection state.
	State() ConnectionState

	// OnStateChange registers a callback for connection state changes.
	OnStateChange(callback func(ConnectionState))
}

// Connectable is an optional interface for transports that require explicit connection.
type Connectable interface {
	// Connect establishes the transport connection.
	Connect(ctx context.Context) error
}

// errHandlerAlreadyRegistered is returned when trying to subscribe with a handler already registered.
var errHandlerAlreadyRegistered = errors.New("notification handler already registered")
