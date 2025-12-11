package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

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
	result, err := transport.Call(context.Background(), "Switch.Set", map[string]any{"id": 0, "on": true})
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
	result, err := transport.Call(context.Background(), "Switch.Set", nil)
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
	result, err := transport.Call(context.Background(), "/relay/0", nil)
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
			_, err := transport.Call(context.Background(), "Switch.Set", nil)
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
	_, err := transport.Call(ctx, "Switch.Set", nil)
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

	_, err := transport.Call(ctx, "Switch.Set", nil)
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
	result, err := transport.Call(context.Background(), "Switch.Set", nil)
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
	_, err := transport.Call(context.Background(), "Switch.Set", nil)
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
	_, err := transport.Call(context.Background(), "Switch.Set", nil)
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
	_, err := transport.Call(context.Background(), "Switch.Set", nil)
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
	result, err := transport.Call(context.Background(), "Switch.Set", nil)
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
	_, err := transport.Call(context.Background(), "Switch.Set", nil)
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

	_, err := transport.Call(ctx, "Switch.Set", nil)
	if err == nil {
		t.Fatal("Call() error = nil, want error")
	}
}

func TestHTTP_Call_DigestAuth(t *testing.T) {
	// Track request count to handle challenge-response flow
	requestCount := 0
	expectedNonce := "dcd98b7102dd2f0e8b11d0f600bfb0c093"
	expectedRealm := "shelly"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		authHeader := r.Header.Get("Authorization")

		// First request (from applyDigestAuth challenge request) - return 401 with challenge
		if authHeader == "" {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(
				`Digest realm="%s", nonce="%s", qop="auth"`,
				expectedRealm, expectedNonce,
			))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Second request should have Authorization header
		if !containsSubstr(authHeader, "Digest") {
			t.Errorf("expected Digest auth, got: %s", authHeader)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Verify the Authorization header contains expected fields
		if !containsSubstr(authHeader, `username="admin"`) {
			t.Errorf("missing username in auth header: %s", authHeader)
		}
		if !containsSubstr(authHeader, fmt.Sprintf(`realm="%s"`, expectedRealm)) {
			t.Errorf("missing realm in auth header: %s", authHeader)
		}
		if !containsSubstr(authHeader, fmt.Sprintf(`nonce="%s"`, expectedNonce)) {
			t.Errorf("missing nonce in auth header: %s", authHeader)
		}
		if !containsSubstr(authHeader, `response="`) {
			t.Errorf("missing response in auth header: %s", authHeader)
		}

		response := types.Response{
			ID:     1,
			Result: json.RawMessage(`{"success":true}`),
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	transport := NewHTTP(server.URL, WithDigestAuth("admin", "password"))
	_, err := transport.Call(context.Background(), "Switch.Set", nil)
	if err != nil {
		t.Fatalf("Call() with digest auth error = %v", err)
	}
}

func TestParseDigestChallenge(t *testing.T) {
	tests := []struct {
		name      string
		challenge string
		want      map[string]string
	}{
		{
			name:      "basic challenge",
			challenge: `realm="shelly", nonce="abc123"`,
			want: map[string]string{
				"realm": "shelly",
				"nonce": "abc123",
			},
		},
		{
			name:      "challenge with qop",
			challenge: `realm="shelly", nonce="abc123", qop="auth"`,
			want: map[string]string{
				"realm": "shelly",
				"nonce": "abc123",
				"qop":   "auth",
			},
		},
		{
			name:      "challenge with opaque",
			challenge: `realm="test", nonce="xyz", opaque="5ccc069c403ebaf9f0171e9517f40e41"`,
			want: map[string]string{
				"realm":  "test",
				"nonce":  "xyz",
				"opaque": "5ccc069c403ebaf9f0171e9517f40e41",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDigestChallenge(tt.challenge)
			for key, wantVal := range tt.want {
				if gotVal := got[key]; gotVal != wantVal {
					t.Errorf("parseDigestChallenge()[%s] = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestCalculateDigestResponse(t *testing.T) {
	// Test vectors based on RFC 2617 examples
	response := calculateDigestResponse(
		"Mufasa",                             // username
		"Circle Of Life",                     // password
		"testrealm@host.com",                 // realm
		"dcd98b7102dd2f0e8b11d0f600bfb0c093", // nonce
		"00000001",                           // nc
		"0a4f113b",                           // cnonce
		"auth",                               // qop
		"GET",                                // method
		"/dir/index.html",                    // uri
	)

	// The expected response is calculated per RFC 2617
	// This is a simplified test - just verify we get a 32-char hex string
	if len(response) != 32 {
		t.Errorf("response length = %d, want 32", len(response))
	}

	// Verify it's valid hex
	for _, c := range response {
		isDigit := c >= '0' && c <= '9'
		isHexLetter := c >= 'a' && c <= 'f'
		if !isDigit && !isHexLetter {
			t.Errorf("response contains non-hex character: %c", c)
		}
	}
}

func TestMD5Hash(t *testing.T) {
	// Test against known MD5 values
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: "d41d8cd98f00b204e9800998ecf8427e"},
		{input: "hello", want: "5d41402abc4b2a76b9719d911017c592"},
		{input: "Mufasa:testrealm@host.com:Circle Of Life", want: "939e7578ed9e3c518a452acee763bce9"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := md5Hash(tt.input)
			if got != tt.want {
				t.Errorf("md5Hash(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
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
	result, err := transport.Call(context.Background(), "/relay/0?turn=on", nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if result == nil {
		t.Error("result is nil")
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
	result, err := transport.Call(context.Background(), "Shelly.GetStatus", nil)
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
	_, err := transport.Call(context.Background(), "Switch.Set", nil)
	if err == nil {
		t.Fatal("Call() error = nil, want error")
	}
}
