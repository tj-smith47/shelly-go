package rpc

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
)

// Request represents a JSON-RPC 2.0 request.
//
// See: https://www.jsonrpc.org/specification#request_object
type Request struct {
	ID      any             `json:"id,omitempty"`
	Auth    *AuthData       `json:"auth,omitempty"`
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// AuthData contains authentication credentials for RPC requests.
type AuthData struct {
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	Realm     string `json:"realm,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
	CNonce    string `json:"cnonce,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
	Response  string `json:"response,omitempty"`
	NC        int    `json:"nc,omitempty"`
}

// RequestBuilder builds JSON-RPC requests with automatic ID management.
type RequestBuilder struct {
	idCounter atomic.Uint64
}

// NewRequestBuilder creates a new RequestBuilder with ID counter starting at 1.
func NewRequestBuilder() *RequestBuilder {
	rb := &RequestBuilder{}
	rb.idCounter.Store(0) // Will be incremented to 1 on first request
	return rb
}

// Build creates a new RPC request with the given method and params.
// The request ID is automatically assigned sequentially.
func (rb *RequestBuilder) Build(method string, params any) (*Request, error) {
	return rb.BuildWithID(rb.NextID(), method, params)
}

// BuildWithID creates a new RPC request with a specific ID.
// This is useful for batch requests or when you need to control the ID.
func (rb *RequestBuilder) BuildWithID(id any, method string, params any) (*Request, error) {
	req := &Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
	}

	// Encode params to JSON if provided
	if params != nil {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		req.Params = paramsJSON
	}

	return req, nil
}

// BuildNotification creates a JSON-RPC notification (request without ID).
// Notifications do not expect a response from the server.
func (rb *RequestBuilder) BuildNotification(method string, params any) (*Request, error) {
	req := &Request{
		JSONRPC: "2.0",
		Method:  method,
	}

	// Encode params to JSON if provided
	if params != nil {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		req.Params = paramsJSON
	}

	return req, nil
}

// BuildBatch creates multiple RPC requests with sequential IDs.
// Each request receives its own unique ID.
func (rb *RequestBuilder) BuildBatch(requests []BatchRequest) ([]*Request, error) {
	result := make([]*Request, 0, len(requests))

	for _, br := range requests {
		req, err := rb.Build(br.Method, br.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to build request for %s: %w", br.Method, err)
		}
		result = append(result, req)
	}

	return result, nil
}

// NextID returns the next sequential request ID.
func (rb *RequestBuilder) NextID() uint64 {
	return rb.idCounter.Add(1)
}

// CurrentID returns the last used request ID without incrementing.
func (rb *RequestBuilder) CurrentID() uint64 {
	return rb.idCounter.Load()
}

// ResetID resets the request ID counter to 0 (next ID will be 1).
// This is primarily useful for testing.
func (rb *RequestBuilder) ResetID() {
	rb.idCounter.Store(0)
}

// BatchRequest represents a single request in a batch operation.
type BatchRequest struct {
	Params any
	Method string
}

// NewBatchRequest creates a new BatchRequest with the given method and params.
func NewBatchRequest(method string, params any) BatchRequest {
	return BatchRequest{
		Method: method,
		Params: params,
	}
}

// MarshalJSON encodes the request to JSON.
func (r *Request) MarshalJSON() ([]byte, error) {
	// Use an anonymous struct to control JSON field order and omitempty behavior
	type Alias Request
	return json.Marshal((*Alias)(r))
}

// UnmarshalJSON decodes the request from JSON.
func (r *Request) UnmarshalJSON(data []byte) error {
	type Alias Request
	aux := (*Alias)(r)
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Validate required fields
	if r.JSONRPC != "2.0" {
		return fmt.Errorf("invalid jsonrpc version: %s", r.JSONRPC)
	}
	if r.Method == "" {
		return fmt.Errorf("method is required")
	}

	return nil
}

// String returns a string representation of the request for debugging.
func (r *Request) String() string {
	if r.ID != nil {
		return fmt.Sprintf("Request{ID: %v, Method: %s}", r.ID, r.Method)
	}
	return fmt.Sprintf("Notification{Method: %s}", r.Method)
}

// IsNotification returns true if this request is a notification (has no ID).
func (r *Request) IsNotification() bool {
	return r.ID == nil
}

// WithAuth adds authentication data to the request.
func (r *Request) WithAuth(auth *AuthData) *Request {
	r.Auth = auth
	return r
}

// GetParams unmarshals the request params into the provided value.
func (r *Request) GetParams(v any) error {
	if len(r.Params) == 0 {
		return nil
	}
	return json.Unmarshal(r.Params, v)
}
