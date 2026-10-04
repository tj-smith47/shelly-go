package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/types"
)

// Response represents a JSON-RPC 2.0 response.
//
// See: https://www.jsonrpc.org/specification#response_object
type Response struct {
	ID      any             `json:"id"`
	Error   *ErrorObject    `json:"error,omitempty"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// ErrorObject represents a JSON-RPC 2.0 error object.
type ErrorObject struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
	Code    int             `json:"code"`
}

// Error implements the error interface for ErrorObject.
func (e *ErrorObject) Error() string {
	if len(e.Data) > 0 {
		return fmt.Sprintf("RPC error %d: %s (data: %s)", e.Code, e.Message, string(e.Data))
	}
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// Unwrap returns the standard error type corresponding to the RPC error code.
func (e *ErrorObject) Unwrap() error {
	return types.MapErrorCode(e.Code)
}

// MarshalJSON encodes the response to JSON.
func (r *Response) MarshalJSON() ([]byte, error) {
	type Alias Response
	return json.Marshal((*Alias)(r))
}

// UnmarshalJSON decodes the response from JSON.
func (r *Response) UnmarshalJSON(data []byte) error {
	type Alias Response
	aux := (*Alias)(r)
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Shelly devices may omit the jsonrpc field, so we only validate
	// if it's present and non-empty
	if r.JSONRPC != "" && r.JSONRPC != JSONRPCVersion {
		return fmt.Errorf("invalid jsonrpc version: %s", r.JSONRPC)
	}

	return nil
}

// String returns a string representation of the response for debugging.
func (r *Response) String() string {
	if r.Error != nil {
		return fmt.Sprintf("Response{ID: %v, Error: %v}", r.ID, r.Error)
	}
	return fmt.Sprintf("Response{ID: %v, Result: %s}", r.ID, string(r.Result))
}

// IsError returns true if the response contains an error.
func (r *Response) IsError() bool {
	return r.Error != nil
}

// GetResult unmarshals the response result into the provided value.
// Returns an error if the response contains an RPC error.
func (r *Response) GetResult(v any) error {
	if r.Error != nil {
		return r.Error
	}

	if len(r.Result) == 0 {
		return nil
	}

	if err := json.Unmarshal(r.Result, v); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return nil
}

// GetError returns the error object if present, nil otherwise.
func (r *Response) GetError() error {
	if r.Error != nil {
		return r.Error
	}
	return nil
}

// Notification represents a JSON-RPC 2.0 notification (server-initiated message).
//
// Notifications are messages from the server that do not expect a response.
// They are used for asynchronous events like status updates or alerts.
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MarshalJSON encodes the notification to JSON.
func (n *Notification) MarshalJSON() ([]byte, error) {
	type Alias Notification
	return json.Marshal((*Alias)(n))
}

// UnmarshalJSON decodes the notification from JSON.
func (n *Notification) UnmarshalJSON(data []byte) error {
	type Alias Notification
	aux := (*Alias)(n)
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Shelly devices may omit the jsonrpc field, so we only validate
	// if it's present and non-empty
	if n.JSONRPC != "" && n.JSONRPC != JSONRPCVersion {
		return fmt.Errorf("invalid jsonrpc version: %s", n.JSONRPC)
	}
	if n.Method == "" {
		return fmt.Errorf("method is required")
	}

	return nil
}

// String returns a string representation of the notification for debugging.
func (n *Notification) String() string {
	return fmt.Sprintf("Notification{Method: %s, Params: %s}", n.Method, string(n.Params))
}

// GetParams unmarshals the notification params into the provided value.
func (n *Notification) GetParams(v any) error {
	if len(n.Params) == 0 {
		return nil
	}
	return json.Unmarshal(n.Params, v)
}

// ParseResponse parses a JSON-RPC response from raw JSON data.
func ParseResponse(data []byte) (*Response, error) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &resp, nil
}

// ParseNotification parses a JSON-RPC notification from raw JSON data.
func ParseNotification(data []byte) (*Notification, error) {
	var notif Notification
	if err := json.Unmarshal(data, &notif); err != nil {
		return nil, fmt.Errorf("failed to parse notification: %w", err)
	}
	return &notif, nil
}

// ParseMessage attempts to parse a JSON-RPC message from raw JSON data.
// It returns the parsed message as either a *Response or a *Notification.
// The caller can use type assertion to determine the message type.
func ParseMessage(data []byte) (any, error) {
	// Try to detect the message type by peeking at the JSON structure
	var peek struct {
		ID      any             `json:"id"`
		Error   *ErrorObject    `json:"error"`
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Result  json.RawMessage `json:"result"`
	}

	if err := json.Unmarshal(data, &peek); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	// If it has an ID and (Result or Error), it's a response
	if peek.ID != nil && (len(peek.Result) > 0 || peek.Error != nil) {
		return ParseResponse(data)
	}

	// If it has a Method but no ID, it's a notification
	if peek.Method != "" && peek.ID == nil {
		return ParseNotification(data)
	}

	return nil, fmt.Errorf("unknown message type")
}
