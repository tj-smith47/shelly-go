package components

import (
	"context"
	"encoding/json"
	"testing"
)

// testComponentError tests that a component method properly handles RPC errors.
//
// This helper reduces boilerplate in error test cases by providing a standard
// way to verify that errors from the transport layer are properly propagated.
//
// Example:
//
//	func TestSwitch_GetConfig_Error(t *testing.T) {
//	    client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
//	    sw := NewSwitch(client, 0)
//	    testComponentError(t, "GetConfig", func() error {
//	        _, err := sw.GetConfig(context.Background())
//	        return err
//	    })
//	}
func testComponentError(t *testing.T, methodName string, fn func() error) {
	t.Helper()

	err := fn()
	if err == nil {
		t.Errorf("%s expected error, got nil", methodName)
	}
}

// testComponentInvalidJSON tests that a component method properly handles invalid JSON responses.
//
// This helper reduces boilerplate in JSON unmarshaling error test cases.
//
// Example:
//
//	func TestSwitch_GetConfig_InvalidJSON(t *testing.T) {
//	    client := rpc.NewClient(invalidJSONTransport())
//	    sw := NewSwitch(client, 0)
//	    testComponentInvalidJSON(t, "GetConfig", func() error {
//	        _, err := sw.GetConfig(context.Background())
//	        return err
//	    })
//	}
func testComponentInvalidJSON(t *testing.T, methodName string, fn func() error) {
	t.Helper()

	err := fn()
	if err == nil {
		t.Errorf("%s expected JSON unmarshaling error, got nil", methodName)
	}
}

// errorTransport returns a mockTransport that always returns the given error.
//
// This is useful for testing error handling in component methods.
//
// Example:
//
//	transport := errorTransport(errors.New("device unreachable"))
//	client := rpc.NewClient(transport)
func errorTransport(err error) *mockTransport {
	return &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, err
		},
	}
}

// invalidJSONTransport returns a mockTransport that returns invalid JSON.
//
// This is useful for testing JSON unmarshaling error handling.
//
// Example:
//
//	transport := invalidJSONTransport()
//	client := rpc.NewClient(transport)
func invalidJSONTransport() *mockTransport {
	return &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return json.RawMessage(`{invalid`), nil
		},
	}
}
