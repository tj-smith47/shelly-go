package transport

import (
	"context"
	"encoding/json"
	"errors"
)

// RPCRequest defines the interface for JSON-RPC 2.0 requests.
// This interface allows the rpc package to pass Request objects to transports
// without creating circular imports.
type RPCRequest interface {
	// GetID returns the request ID (typically uint64 or nil for notifications).
	GetID() any

	// GetMethod returns the RPC method name (e.g., "Switch.Set") or REST path (e.g., "/relay/0").
	GetMethod() string

	// GetParams returns the pre-marshaled JSON parameters.
	GetParams() json.RawMessage

	// GetAuth returns the authentication data, or nil if not set.
	// The returned value is typically *rpc.AuthData but returned as any
	// to avoid circular imports.
	GetAuth() any

	// GetJSONRPC returns the JSON-RPC version string (typically "2.0").
	// For REST requests, this returns an empty string.
	GetJSONRPC() string

	// IsREST returns true if this is a Gen1 REST request (path-based, not RPC).
	IsREST() bool
}

// BatchRPCRequest is an optional interface for batch requests.
// Requests that implement this interface contain multiple RPC requests.
type BatchRPCRequest interface {
	RPCRequest
	// IsBatch returns true if this is a batch request.
	IsBatch() bool
}

// SimpleRequest is a basic request for Gen1 REST API calls.
// It implements RPCRequest interface for backward compatibility.
type SimpleRequest struct {
	Path string
}

// NewSimpleRequest creates a new Gen1 REST request.
func NewSimpleRequest(path string) *SimpleRequest {
	return &SimpleRequest{Path: path}
}

func (r *SimpleRequest) GetID() any            { return nil }
func (r *SimpleRequest) GetMethod() string     { return r.Path }
func (r *SimpleRequest) GetParams() json.RawMessage { return nil }
func (r *SimpleRequest) GetAuth() any          { return nil }
func (r *SimpleRequest) GetJSONRPC() string    { return "" }
func (r *SimpleRequest) IsREST() bool          { return true }

// Transport defines the interface for communicating with Shelly devices.
// Implementations handle different protocols (HTTP, WebSocket, MQTT, CoAP).
//
// Transport implementations must be safe for concurrent use.
type Transport interface {
	// Call executes an RPC request and returns the response.
	// The request contains all necessary information (method, params, auth, id).
	// Returns the raw JSON response or an error.
	Call(ctx context.Context, req RPCRequest) (json.RawMessage, error)

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
