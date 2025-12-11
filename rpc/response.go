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

// BatchResponse represents a collection of RPC responses from a batch request.
type BatchResponse struct {
	Responses []*Response
}

// NewBatchResponse creates a new BatchResponse from a slice of responses.
func NewBatchResponse(responses []*Response) *BatchResponse {
	return &BatchResponse{
		Responses: responses,
	}
}

// MarshalJSON encodes the batch response to JSON.
func (br *BatchResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(br.Responses)
}

// UnmarshalJSON decodes the batch response from JSON.
func (br *BatchResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &br.Responses)
}

// Get returns the response at the given index.
// Returns nil if the index is out of bounds.
func (br *BatchResponse) Get(index int) *Response {
	if index < 0 || index >= len(br.Responses) {
		return nil
	}
	return br.Responses[index]
}

// GetByID returns the response with the given ID.
// Returns nil if no response with that ID is found.
func (br *BatchResponse) GetByID(id any) *Response {
	for _, resp := range br.Responses {
		if idsEqual(resp.ID, id) {
			return resp
		}
	}
	return nil
}

// idsEqual compares two IDs, handling type conversions from JSON unmarshaling.
// JSON numbers are unmarshaled as float64, but IDs may be generated as int, uint64, etc.
func idsEqual(a, b any) bool {
	if a == b {
		return true
	}

	// Convert both to float64 for comparison
	aFloat := toFloat64(a)
	bFloat := toFloat64(b)

	if aFloat != nil && bFloat != nil {
		return *aFloat == *bFloat
	}

	// Try string comparison as fallback
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	if aOk && bOk {
		return aStr == bStr
	}

	return false
}

// toFloat64 attempts to convert an any to float64.
// Returns nil if conversion is not possible.
func toFloat64(v any) *float64 {
	switch val := v.(type) {
	case float64:
		return &val
	case float32:
		f := float64(val)
		return &f
	case int:
		f := float64(val)
		return &f
	case int8:
		f := float64(val)
		return &f
	case int16:
		f := float64(val)
		return &f
	case int32:
		f := float64(val)
		return &f
	case int64:
		f := float64(val)
		return &f
	case uint:
		f := float64(val)
		return &f
	case uint8:
		f := float64(val)
		return &f
	case uint16:
		f := float64(val)
		return &f
	case uint32:
		f := float64(val)
		return &f
	case uint64:
		f := float64(val)
		return &f
	default:
		return nil
	}
}

// Len returns the number of responses in the batch.
func (br *BatchResponse) Len() int {
	return len(br.Responses)
}

// HasErrors returns true if any response in the batch contains an error.
func (br *BatchResponse) HasErrors() bool {
	for _, resp := range br.Responses {
		if resp.IsError() {
			return true
		}
	}
	return false
}

// Errors returns a slice of all errors in the batch response.
// Returns nil if there are no errors.
func (br *BatchResponse) Errors() []error {
	var errs []error
	for _, resp := range br.Responses {
		if resp.Error != nil {
			errs = append(errs, resp.Error)
		}
	}
	return errs
}

// String returns a string representation of the batch response for debugging.
func (br *BatchResponse) String() string {
	successCount := 0
	errorCount := 0
	for _, resp := range br.Responses {
		if resp.IsError() {
			errorCount++
		} else {
			successCount++
		}
	}
	return fmt.Sprintf("BatchResponse{Total: %d, Success: %d, Errors: %d}",
		len(br.Responses), successCount, errorCount)
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

// ParseBatchResponse parses a JSON-RPC batch response from raw JSON data.
func ParseBatchResponse(data []byte) (*BatchResponse, error) {
	var batch BatchResponse
	if err := json.Unmarshal(data, &batch); err != nil {
		return nil, fmt.Errorf("failed to parse batch response: %w", err)
	}
	return &batch, nil
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
// It returns the parsed message as one of: *Response, *BatchResponse, or *Notification.
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
		// Maybe it's a batch response (array)
		if batch, batchErr := ParseBatchResponse(data); batchErr == nil {
			return batch, nil
		}
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
