package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tj-smith47/shelly-go/transport"
)

// CallHandler is a function that handles RPC calls and returns a response.
type CallHandler func(params any) (json.RawMessage, error)

// MockTransport is a mock implementation of transport.Transport for testing.
type MockTransport struct {
	handlers map[string]CallHandler
	calls    []RecordedCall
	mu       sync.RWMutex
	closed   bool
}

// RecordedCall records an RPC call made to the mock transport.
type RecordedCall struct {
	Params any
	Method string
}

// NewMockTransport creates a new mock transport.
func NewMockTransport() *MockTransport {
	return &MockTransport{
		handlers: make(map[string]CallHandler),
		calls:    make([]RecordedCall, 0),
	}
}

// OnCall registers a handler for a specific RPC method.
func (mt *MockTransport) OnCall(method string, handler CallHandler) *MockTransport {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.handlers[method] = handler
	return mt
}

// OnCallReturn registers a simple return value for a method.
func (mt *MockTransport) OnCallReturn(method string, response any, err error) *MockTransport {
	return mt.OnCall(method, func(params any) (json.RawMessage, error) {
		if err != nil {
			return nil, err
		}
		if response == nil {
			return json.RawMessage(`{}`), nil
		}
		data, e := json.Marshal(response)
		if e != nil {
			return nil, e
		}
		return data, nil
	})
}

// OnCallJSON registers a JSON string response for a method.
func (mt *MockTransport) OnCallJSON(method, jsonResponse string) *MockTransport {
	return mt.OnCall(method, func(params any) (json.RawMessage, error) {
		return json.RawMessage(jsonResponse), nil
	})
}

// OnCallError registers an error response for a method.
func (mt *MockTransport) OnCallError(method string, err error) *MockTransport {
	return mt.OnCall(method, func(params any) (json.RawMessage, error) {
		return nil, err
	})
}

// Call implements the transport.Transport interface.
func (mt *MockTransport) Call(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
	method := req.GetMethod()
	params := req.GetParams()

	mt.mu.Lock()
	if mt.closed {
		mt.mu.Unlock()
		return nil, fmt.Errorf("transport closed")
	}
	mt.calls = append(mt.calls, RecordedCall{Method: method, Params: params})
	handler, ok := mt.handlers[method]
	mt.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if ok {
		return handler(params)
	}

	// Try path matchers as fallback
	if resp, err, found := mt.callWithMatchers(method, params); found {
		return resp, err
	}

	return nil, fmt.Errorf("no handler registered for method: %s", method)
}

// Close marks the transport as closed.
func (mt *MockTransport) Close() error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.closed = true
	return nil
}

// Calls returns all recorded calls.
func (mt *MockTransport) Calls() []RecordedCall {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	result := make([]RecordedCall, len(mt.calls))
	copy(result, mt.calls)
	return result
}

// CallCount returns the number of calls made.
func (mt *MockTransport) CallCount() int {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return len(mt.calls)
}

// LastCall returns the most recent call, or nil if none.
func (mt *MockTransport) LastCall() *RecordedCall {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	if len(mt.calls) == 0 {
		return nil
	}
	call := mt.calls[len(mt.calls)-1]
	return &call
}

// Reset clears all recorded calls but keeps handlers.
func (mt *MockTransport) Reset() {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.calls = mt.calls[:0]
	mt.closed = false
}

// ClearHandlers removes all registered handlers.
func (mt *MockTransport) ClearHandlers() {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.handlers = make(map[string]CallHandler)
}

// WasCalled returns true if the method was called at least once.
func (mt *MockTransport) WasCalled(method string) bool {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	for _, call := range mt.calls {
		if call.Method == method {
			return true
		}
	}
	return false
}

// CallsFor returns all calls to a specific method.
func (mt *MockTransport) CallsFor(method string) []RecordedCall {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	var result []RecordedCall
	for _, call := range mt.calls {
		if call.Method == method {
			result = append(result, call)
		}
	}
	return result
}

// PathMatcher is a function that determines if a path matches.
type PathMatcher func(path string) bool

// matcherHandler wraps a PathMatcher with a CallHandler.
type matcherHandler struct {
	matcher PathMatcher
	handler CallHandler
}

// matchers holds registered path matchers.
var matchers []matcherHandler

// OnPathMatch registers a handler for paths matching a custom function.
// This allows matching paths with query parameters.
func (mt *MockTransport) OnPathMatch(matcher PathMatcher, handler CallHandler) *MockTransport {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	matchers = append(matchers, matcherHandler{matcher: matcher, handler: handler})
	return mt
}

// makeResponseHandler creates a CallHandler that returns the given response or error.
func makeResponseHandler(response any, err error) CallHandler {
	return func(_ any) (json.RawMessage, error) {
		if err != nil {
			return nil, err
		}
		if response == nil {
			return json.RawMessage(`{}`), nil
		}
		switch v := response.(type) {
		case json.RawMessage:
			return v, nil
		case string:
			return json.RawMessage(v), nil
		default:
			data, e := json.Marshal(response)
			if e != nil {
				return nil, e
			}
			return data, nil
		}
	}
}

// OnPathContains registers a handler for paths containing a substring.
func (mt *MockTransport) OnPathContains(substring string, response any, err error) *MockTransport {
	return mt.OnPathMatch(func(path string) bool {
		return contains(path, substring)
	}, makeResponseHandler(response, err))
}

// OnPathPrefix registers a handler for paths starting with a prefix.
func (mt *MockTransport) OnPathPrefix(prefix string, response any, err error) *MockTransport {
	return mt.OnPathMatch(func(path string) bool {
		return hasPrefix(path, prefix)
	}, makeResponseHandler(response, err))
}

// contains checks if s contains substr (to avoid import strings).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" || findSubstr(s, substr) >= 0)
}

// hasPrefix checks if s starts with prefix.
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// findSubstr returns index of substr in s, or -1.
func findSubstr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// ClearMatchers clears all path matchers.
func (mt *MockTransport) ClearMatchers() {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	matchers = nil
}

// callWithMatchers tries matchers if no exact handler found.
func (mt *MockTransport) callWithMatchers(method string, params any) (json.RawMessage, error, bool) {
	for _, mh := range matchers {
		if mh.matcher(method) {
			resp, err := mh.handler(params)
			return resp, err, true
		}
	}
	return nil, nil, false
}
