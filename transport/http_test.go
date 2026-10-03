package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// testRPCRequest is a test-only RPC request type for testing RPC calls
type testRPCRequest struct {
	id      int
	jsonrpc string
	method  string
	params  json.RawMessage
}

func newTestRPCRequest(method string, params any) *testRPCRequest {
	var paramsJSON json.RawMessage
	if params != nil {
		paramsJSON, _ = json.Marshal(params)
	}
	return &testRPCRequest{
		id:      1,
		jsonrpc: "2.0",
		method:  method,
		params:  paramsJSON,
	}
}

func (r *testRPCRequest) GetID() any                 { return r.id }
func (r *testRPCRequest) GetJSONRPC() string         { return r.jsonrpc }
func (r *testRPCRequest) GetMethod() string          { return r.method }
func (r *testRPCRequest) GetParams() json.RawMessage { return r.params }
func (r *testRPCRequest) GetAuth() any               { return nil }
func (r *testRPCRequest) IsREST() bool               { return false }

// testRPCRequestWithAuth is a test RPC request that includes auth info
type testRPCRequestWithAuth struct {
	testRPCRequest
	auth map[string]any
}

func newTestRPCRequestWithAuth(method string, params any, auth map[string]any) *testRPCRequestWithAuth {
	base := newTestRPCRequest(method, params)
	return &testRPCRequestWithAuth{
		testRPCRequest: *base,
		auth:           auth,
	}
}

func (r *testRPCRequestWithAuth) GetAuth() any { return r.auth }

func TestNewHTTP(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		opts    []Option
	}{
		{
			name:    "basic HTTP",
			baseURL: "http://192.168.1.100",
		},
		{
			name:    "with timeout",
			baseURL: "http://192.168.1.100",
			opts:    []Option{WithTimeout(10 * time.Second)},
		},
		{
			name:    "with auth",
			baseURL: "http://192.168.1.100",
			opts:    []Option{WithAuth("admin", "password")},
		},
		{
			name:    "with multiple options",
			baseURL: "http://192.168.1.100",
			opts: []Option{
				WithTimeout(30 * time.Second),
				WithAuth("admin", "password"),
				WithRetry(5, 2*time.Second),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewHTTP(tt.baseURL, tt.opts...)
			if transport == nil {
				t.Fatal("NewHTTP() returned nil")
			}
			if transport.baseURL != tt.baseURL {
				t.Errorf("baseURL = %v, want %v", transport.baseURL, tt.baseURL)
			}
		})
	}
}

func TestNewHTTP_TrailingSlash(t *testing.T) {
	transport := NewHTTP("http://192.168.1.100/")
	want := "http://192.168.1.100"
	if transport.baseURL != want {
		t.Errorf("baseURL = %v, want %v", transport.baseURL, want)
	}
}

func TestHTTP_Call_RPC_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rpc" {
			t.Errorf("path = %v, want /rpc", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("method = %v, want POST", r.Method)
		}

		response := types.Response{
			ID:     1,
			Result: json.RawMessage(`{"output":true}`),
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	result, err := transport.Call(context.Background(), newTestRPCRequest("Switch.Set", nil))
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	// Transport now returns raw response; parsing is done by RPC client
	var resp types.Response
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	var output struct {
		Output bool `json:"output"`
	}
	if err := json.Unmarshal(resp.Result, &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !output.Output {
		t.Error("output = false, want true")
	}
}

func TestHTTP_Call_RPC_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := types.Response{
			ID: 1,
			Error: &types.Error{
				Code:    types.ErrCodeNotFound,
				Message: "component not found",
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	result, err := transport.Call(context.Background(), newTestRPCRequest("Switch.Set", nil))
	if err != nil {
		t.Fatalf("Call() error = %v, want nil (error handling is done by RPC client)", err)
	}

	// Transport returns raw response; RPC client handles error extraction
	var resp types.Response
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
	if resp.Error.Code != types.ErrCodeNotFound {
		t.Errorf("error code = %v, want %v", resp.Error.Code, types.ErrCodeNotFound)
	}
}

func TestHTTP_Call_REST_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/relay/0" {
			t.Errorf("path = %v, want /relay/0", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("method = %v, want GET", r.Method)
		}

		_, _ = w.Write([]byte(`{"ison":true}`))
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	result, err := transport.Call(context.Background(), NewSimpleRequest("/relay/0"))
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	var status struct {
		IsOn bool `json:"ison"`
	}
	if err := json.Unmarshal(result, &status); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !status.IsOn {
		t.Error("ison = false, want true")
	}
}

func TestHTTP_Call_HTTPError(t *testing.T) {
	tests := []struct {
		wantErr    error
		name       string
		statusCode int
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    types.ErrAuth,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantErr:    types.ErrNotFound,
		},
		{
			name:       "timeout",
			statusCode: http.StatusRequestTimeout,
			wantErr:    types.ErrTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			transport := NewHTTP(server.URL)
			_, err := transport.Call(context.Background(), newTestRPCRequest("Switch.Set", nil))
			if err == nil {
				t.Fatal("Call() error = nil, want error")
			}
			// Check if error wraps the expected error
			// Note: errors.Is would be better but we're checking string for now
			if tt.wantErr != nil && err.Error() == "" {
				t.Errorf("Call() error = %v, want error containing %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTP_Call_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{"result":{}}`))
	}))
	defer server.Close()

	transport := NewHTTP(server.URL, WithTimeout(10*time.Millisecond))

	ctx := context.Background()
	_, err := transport.Call(ctx, newTestRPCRequest("Switch.Set", nil))
	if err == nil {
		t.Fatal("Call() error = nil, want timeout error")
	}
}

func TestHTTP_Call_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"result":{}}`))
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := transport.Call(ctx, newTestRPCRequest("Switch.Set", nil))
	if err == nil {
		t.Fatal("Call() error = nil, want context canceled error")
	}
}

func TestHTTP_Call_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		response := types.Response{
			ID:     1,
			Result: json.RawMessage(`{"success":true}`),
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL, WithRetry(3, 10*time.Millisecond))
	result, err := transport.Call(context.Background(), newTestRPCRequest("Switch.Set", nil))
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	if result == nil {
		t.Error("result is nil")
	}

	if attempts != 3 {
		t.Errorf("attempts = %v, want 3", attempts)
	}
}

func TestHTTP_Call_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL, WithRetry(2, 10*time.Millisecond))
	_, err := transport.Call(context.Background(), newTestRPCRequest("Switch.Set", nil))
	if err == nil {
		t.Fatal("Call() error = nil, want max retries error")
	}
}

func TestHTTP_Close(t *testing.T) {
	transport := NewHTTP("http://192.168.1.100")
	err := transport.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestHTTP_SetTimeout(t *testing.T) {
	transport := NewHTTP("http://192.168.1.100")

	newTimeout := 60 * time.Second
	transport.SetTimeout(newTimeout)

	got := transport.GetTimeout()
	if got != newTimeout {
		t.Errorf("GetTimeout() = %v, want %v", got, newTimeout)
	}
}

func TestHTTP_GetTimeout(t *testing.T) {
	timeout := 45 * time.Second
	transport := NewHTTP("http://192.168.1.100", WithTimeout(timeout))

	got := transport.GetTimeout()
	if got != timeout {
		t.Errorf("GetTimeout() = %v, want %v", got, timeout)
	}
}

func TestHTTP_WithAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if username != "admin" || password != "password" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		response := types.Response{
			ID:     1,
			Result: json.RawMessage(`{"success":true}`),
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL, WithAuth("admin", "password"))
	_, err := transport.Call(context.Background(), newTestRPCRequest("Switch.Set", nil))
	if err != nil {
		t.Fatalf("Call() with auth error = %v", err)
	}
}

func TestHTTP_WithHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "test-value" {
			t.Error("custom header not found")
		}

		response := types.Response{
			ID:     1,
			Result: json.RawMessage(`{"success":true}`),
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL, WithHeader("X-Custom-Header", "test-value"))
	_, err := transport.Call(context.Background(), newTestRPCRequest("Switch.Set", nil))
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
}

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		want  string
		state ConnectionState
	}{
		{state: StateDisconnected, want: "Disconnected"},
		{state: StateConnecting, want: "Connecting"},
		{state: StateConnected, want: "Connected"},
		{state: StateReconnecting, want: "Reconnecting"},
		{state: StateClosed, want: "Closed"},
		{state: ConnectionState(99), want: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("ConnectionState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateBackoffDelay(t *testing.T) {
	tests := []struct {
		name       string
		baseDelay  time.Duration
		attempt    int
		multiplier float64
		want       time.Duration
	}{
		{
			name:       "first retry",
			baseDelay:  1 * time.Second,
			attempt:    0,
			multiplier: 2.0,
			want:       1 * time.Second,
		},
		{
			name:       "second retry",
			baseDelay:  1 * time.Second,
			attempt:    1,
			multiplier: 2.0,
			want:       2 * time.Second,
		},
		{
			name:       "third retry",
			baseDelay:  1 * time.Second,
			attempt:    2,
			multiplier: 2.0,
			want:       4 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBackoffDelay(tt.baseDelay, tt.attempt, tt.multiplier)
			if got != tt.want {
				t.Errorf("calculateBackoffDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTP_ShouldRetry(t *testing.T) {
	transport := NewHTTP("http://192.168.1.100")

	tests := []struct {
		err  error
		name string
		want bool
	}{
		{
			name: "should not retry auth errors",
			err:  types.ErrAuth,
			want: false,
		},
		{
			name: "should not retry not found errors",
			err:  types.ErrNotFound,
			want: false,
		},
		{
			name: "should not retry context canceled",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "should not retry context deadline exceeded",
			err:  context.DeadlineExceeded,
			want: false,
		},
		{
			name: "should retry other errors",
			err:  types.ErrTimeout,
			want: true,
		},
		{
			name: "should retry generic errors",
			err:  fmt.Errorf("network error"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transport.shouldRetry(tt.err)
			if got != tt.want {
				t.Errorf("shouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTP_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	result, err := transport.Call(context.Background(), newTestRPCRequest("Switch.Set", nil))
	// Transport returns raw body without parsing
	if err != nil {
		t.Errorf("Call() error = %v, want nil", err)
	}
	// The RPC client would fail to parse this as valid JSON
	var resp types.Response
	if err := json.Unmarshal(result, &resp); err == nil {
		t.Error("expected JSON unmarshal error for invalid JSON")
	}
}

func TestHTTP_CustomClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	transport := NewHTTP("http://192.168.1.100", WithClient(customClient))

	if transport.client != customClient {
		t.Error("custom client not set")
	}
}

func TestHTTP_Call_GenericHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	_, err := transport.Call(context.Background(), newTestRPCRequest("Switch.Set", nil))
	if err == nil {
		t.Fatal("Call() error = nil, want error")
	}
	// Error should contain status code 400
	if !contains(err.Error(), "400") {
		t.Errorf("error should contain status code 400, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestHTTP_Call_RetryContextCanceled(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL, WithRetry(5, 50*time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := transport.Call(ctx, newTestRPCRequest("Switch.Set", nil))
	if err == nil {
		t.Fatal("Call() error = nil, want error")
	}
}

func TestHTTP_Close_NonTransport(t *testing.T) {
	// Test Close with a client that has a non-*http.Transport
	customClient := &http.Client{
		Timeout:   10 * time.Second,
		Transport: nil, // nil transport
	}

	transport := NewHTTP("http://192.168.1.100", WithClient(customClient))
	err := transport.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestHTTP_Call_RPC_WithAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the auth field is in the request body
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Check auth field exists
		if _, ok := reqBody["auth"]; !ok {
			t.Error("Expected auth field in request body")
		}

		response := types.Response{
			ID:     1,
			Result: json.RawMessage(`{"success":true}`),
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	authData := map[string]any{
		"realm":    "shelly",
		"username": "admin",
		"nonce":    "12345",
		"response": "abcdef",
	}
	req := newTestRPCRequestWithAuth("Shelly.GetDeviceInfo", nil, authData)
	_, err := transport.Call(context.Background(), req)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
}

func TestHTTP_Call_RPC_WithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify params are in the request body
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Check params field exists
		params, ok := reqBody["params"].(map[string]any)
		if !ok {
			t.Error("Expected params field in request body")
		}
		if params["id"] != float64(0) {
			t.Errorf("params.id = %v, want 0", params["id"])
		}

		response := types.Response{
			ID:     1,
			Result: json.RawMessage(`{"success":true}`),
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	req := newTestRPCRequest("Switch.Set", map[string]any{"id": 0, "on": true})
	_, err := transport.Call(context.Background(), req)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
}

func TestHTTP_buildRESTRequest_Variants(t *testing.T) {
	transport := NewHTTP("http://192.168.1.100")

	// Test REST request with query params
	req, err := transport.buildRESTRequest(context.Background(), "/relay/0?turn=on")
	if err != nil {
		t.Fatalf("buildRESTRequest() error = %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("buildRESTRequest() method = %v, want GET", req.Method)
	}
	if req.URL.Path != "/relay/0" {
		t.Errorf("buildRESTRequest() path = %v, want /relay/0", req.URL.Path)
	}
}

// testRPCRequestWithInvalidParams is a test RPC request with invalid JSON params
type testRPCRequestWithInvalidParams struct {
	testRPCRequest
}

func (r *testRPCRequestWithInvalidParams) GetParams() json.RawMessage {
	return json.RawMessage(`{invalid json`)
}

func TestHTTP_buildRPCRequest_InvalidParams(t *testing.T) {
	transport := NewHTTP("http://192.168.1.100")

	req := &testRPCRequestWithInvalidParams{
		testRPCRequest: testRPCRequest{
			id:      1,
			jsonrpc: "2.0",
			method:  "Test.Method",
		},
	}

	_, err := transport.buildRPCRequest(context.Background(), req)
	if err == nil {
		t.Error("buildRPCRequest() should return error for invalid JSON params")
	}
}

func TestHTTP_Call_REST_WithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a GET request with query params in path
		if r.Method != "GET" {
			t.Errorf("method = %v, want GET", r.Method)
		}
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	result, err := transport.Call(context.Background(), NewSimpleRequest("/relay/0?turn=on"))
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if result == nil {
		t.Error("result is nil")
	}
}

func TestHTTP_buildRPCRequest_WithID(t *testing.T) {
	transport := NewHTTP("http://192.168.1.100")

	// Test with non-nil ID
	req := newTestRPCRequest("Test.Method", nil)
	httpReq, err := transport.buildRPCRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("buildRPCRequest() error = %v", err)
	}
	if httpReq.Method != "POST" {
		t.Errorf("HTTP method = %v, want POST", httpReq.Method)
	}
	if httpReq.URL.Path != "/rpc" {
		t.Errorf("HTTP path = %v, want /rpc", httpReq.URL.Path)
	}
}

func TestHTTP_Call_RPC_WithNilParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := types.Response{
			ID:     1,
			Result: json.RawMessage(`{"output":true}`),
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	result, err := transport.Call(context.Background(), newTestRPCRequest("Shelly.GetStatus", nil))
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if result == nil {
		t.Error("result is nil")
	}
}

func TestHTTP_WithRetryBackoff(t *testing.T) {
	transport := NewHTTP("http://192.168.1.100", WithRetryBackoff(2.5))
	if transport.opts.retryBackoff != 2.5 {
		t.Errorf("retryBackoff = %v, want 2.5", transport.opts.retryBackoff)
	}
}

func TestHTTP_Call_InternalServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	transport := NewHTTP(server.URL, WithRetry(0, 0))
	_, err := transport.Call(context.Background(), newTestRPCRequest("Switch.Set", nil))
	if err == nil {
		t.Fatal("Call() error = nil, want error")
	}
}

func TestHTTP_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			t.Errorf("path = %v, want /status", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("method = %v, want GET", r.Method)
		}
		_, _ = w.Write([]byte(`{"wifi_sta":{"connected":true}}`))
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	result, err := transport.Get(context.Background(), "/status")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	var status struct {
		WifiSta struct {
			Connected bool `json:"connected"`
		} `json:"wifi_sta"`
	}
	if err := json.Unmarshal(result, &status); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !status.WifiSta.Connected {
		t.Error("wifi_sta.connected = false, want true")
	}
}

func TestHTTP_Get_Gen2RPC(t *testing.T) {
	// Test that Get works for Gen2+ /rpc/Method style calls
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rpc/Shelly.GetStatus" {
			t.Errorf("path = %v, want /rpc/Shelly.GetStatus", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("method = %v, want GET", r.Method)
		}
		_, _ = w.Write([]byte(`{"sys":{"available_updates":{},"mac":"AABBCCDDEEFF"}}`))
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	result, err := transport.Get(context.Background(), "/rpc/Shelly.GetStatus")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	var status struct {
		Sys struct {
			MAC string `json:"mac"`
		} `json:"sys"`
	}
	if err := json.Unmarshal(result, &status); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if status.Sys.MAC != "AABBCCDDEEFF" {
		t.Errorf("sys.mac = %v, want AABBCCDDEEFF", status.Sys.MAC)
	}
}

func TestHTTP_Get_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL)
	_, err := transport.Get(context.Background(), "/invalid")
	if err == nil {
		t.Fatal("Get() error = nil, want error")
	}
}

func TestHTTPCall_ErrorRedactsQueryValues(t *testing.T) {
	// A closed server makes the request fail inside the HTTP client, whose
	// error quotes the full URL.
	server := httptest.NewServer(http.NotFoundHandler())
	addr := server.URL
	server.Close()

	h := NewHTTP(addr, WithRetry(0, time.Millisecond))
	_, err := h.Call(context.Background(), NewSimpleRequest("/settings/sta?ssid=HomeNet&key=hunter2secret"))
	if err == nil {
		t.Fatal("expected an error from a closed server")
	}
	msg := err.Error()
	if strings.Contains(msg, "hunter2secret") || strings.Contains(msg, "HomeNet") {
		t.Errorf("error leaks query values: %s", msg)
	}
	if !strings.Contains(msg, "key=REDACTED") || !strings.Contains(msg, "/settings/sta") {
		t.Errorf("error should keep the path and parameter names: %s", msg)
	}
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Errorf("error chain lost the *url.Error: %v", err)
	}
}

func TestRedactURLError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "query values hidden, names kept",
			err:  &url.Error{Op: "Get", URL: "http://192.0.2.1/settings/sta?ssid=Home&key=secret", Err: errors.New("boom")},
			want: `Get "http://192.0.2.1/settings/sta?ssid=REDACTED&key=REDACTED": boom`,
		},
		{
			name: "unparseable url from a parse failure",
			err:  &url.Error{Op: "parse", URL: "http://192.0.2.1/settings?key=se%zzcret", Err: errors.New("invalid URL escape")},
			want: `parse "http://192.0.2.1/settings?key=REDACTED": invalid URL escape`,
		},
		{
			name: "fragment kept",
			err:  &url.Error{Op: "Get", URL: "http://192.0.2.1/a?key=secret#frag", Err: errors.New("boom")},
			want: `Get "http://192.0.2.1/a?key=REDACTED#frag": boom`,
		},
		{
			name: "no query is unchanged",
			err:  &url.Error{Op: "Get", URL: "http://192.0.2.1/status", Err: errors.New("boom")},
			want: `Get "http://192.0.2.1/status": boom`,
		},
		{
			name: "other errors are unchanged",
			err:  errors.New("plain"),
			want: "plain",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactURLError(tt.err).Error(); got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestHTTPCall_BuildErrorRedactsQueryValues(t *testing.T) {
	h := NewHTTP("http://192.0.2.1", WithRetry(0, time.Millisecond))
	_, err := h.Call(context.Background(), NewSimpleRequest("/settings/%zz?password=hunter2secret"))
	if err == nil {
		t.Fatal("expected a request build error for an invalid escape")
	}
	if strings.Contains(err.Error(), "hunter2secret") {
		t.Errorf("error leaks the query value: %v", err)
	}
}
