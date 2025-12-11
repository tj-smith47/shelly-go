package rpc

import (
	"encoding/json"
	"testing"
)

func TestRequestBuilder_Build(t *testing.T) {
	tests := []struct {
		params  any
		name    string
		method  string
		wantErr bool
	}{
		{
			name:    "simple request",
			method:  "Switch.Set",
			params:  map[string]any{"id": 0, "on": true},
			wantErr: false,
		},
		{
			name:    "request without params",
			method:  "Switch.GetStatus",
			params:  nil,
			wantErr: false,
		},
		{
			name:    "request with struct params",
			method:  "Light.Set",
			params:  struct{ Brightness int }{Brightness: 50},
			wantErr: false,
		},
		{
			name:    "request with invalid params",
			method:  "Test",
			params:  make(chan int), // channels cannot be marshaled
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRequestBuilder()
			req, err := rb.Build(tt.method, tt.params)

			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if req.JSONRPC != "2.0" {
				t.Errorf("JSONRPC = %v, want 2.0", req.JSONRPC)
			}

			if req.Method != tt.method {
				t.Errorf("Method = %v, want %v", req.Method, tt.method)
			}

			if req.ID == nil {
				t.Error("ID should not be nil")
			}

			if tt.params != nil && len(req.Params) == 0 {
				t.Error("Params should not be empty")
			}

			if tt.params == nil && len(req.Params) != 0 {
				t.Error("Params should be empty")
			}
		})
	}
}

func TestRequestBuilder_BuildWithID(t *testing.T) {
	rb := NewRequestBuilder()
	testID := uint64(42)

	req, err := rb.BuildWithID(testID, "Test.Method", nil)
	if err != nil {
		t.Fatalf("BuildWithID() error = %v", err)
	}

	if req.ID != testID {
		t.Errorf("ID = %v, want %v", req.ID, testID)
	}
}

func TestRequestBuilder_BuildNotification(t *testing.T) {
	rb := NewRequestBuilder()

	req, err := rb.BuildNotification("NotifyStatus", map[string]any{"status": "ok"})
	if err != nil {
		t.Fatalf("BuildNotification() error = %v", err)
	}

	if req.ID != nil {
		t.Error("Notification should not have ID")
	}

	if !req.IsNotification() {
		t.Error("IsNotification() should return true")
	}
}

func TestRequestBuilder_BuildBatch(t *testing.T) {
	rb := NewRequestBuilder()

	requests := []BatchRequest{
		{Method: "Switch.GetStatus", Params: map[string]any{"id": 0}},
		{Method: "Switch.GetStatus", Params: map[string]any{"id": 1}},
		{Method: "Light.GetStatus", Params: map[string]any{"id": 0}},
	}

	batch, err := rb.BuildBatch(requests)
	if err != nil {
		t.Fatalf("BuildBatch() error = %v", err)
	}

	if len(batch) != len(requests) {
		t.Errorf("batch length = %v, want %v", len(batch), len(requests))
	}

	// Check that each request has a unique ID
	ids := make(map[any]bool)
	for _, req := range batch {
		if req.ID == nil {
			t.Error("batch request should have ID")
		}
		if ids[req.ID] {
			t.Errorf("duplicate ID %v", req.ID)
		}
		ids[req.ID] = true
	}
}

func TestRequestBuilder_IDManagement(t *testing.T) {
	rb := NewRequestBuilder()

	// First ID should be 1
	id1 := rb.NextID()
	if id1 != 1 {
		t.Errorf("first ID = %v, want 1", id1)
	}

	// Second ID should be 2
	id2 := rb.NextID()
	if id2 != 2 {
		t.Errorf("second ID = %v, want 2", id2)
	}

	// CurrentID should return the last used ID
	current := rb.CurrentID()
	if current != 2 {
		t.Errorf("CurrentID = %v, want 2", current)
	}

	// Reset should start from 1 again
	rb.ResetID()
	id3 := rb.NextID()
	if id3 != 1 {
		t.Errorf("ID after reset = %v, want 1", id3)
	}
}

func TestRequest_MarshalJSON(t *testing.T) {
	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "Switch.Set",
		Params:  json.RawMessage(`{"id":0,"on":true}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.JSONRPC != req.JSONRPC {
		t.Errorf("JSONRPC = %v, want %v", decoded.JSONRPC, req.JSONRPC)
	}

	if decoded.Method != req.Method {
		t.Errorf("Method = %v, want %v", decoded.Method, req.Method)
	}
}

func TestRequest_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid request",
			json:    `{"jsonrpc":"2.0","id":1,"method":"Test"}`,
			wantErr: false,
		},
		{
			name:    "invalid jsonrpc version",
			json:    `{"jsonrpc":"1.0","id":1,"method":"Test"}`,
			wantErr: true,
		},
		{
			name:    "missing method",
			json:    `{"jsonrpc":"2.0","id":1}`,
			wantErr: true,
		},
		{
			name:    "notification (no ID)",
			json:    `{"jsonrpc":"2.0","method":"Notify"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req Request
			err := json.Unmarshal([]byte(tt.json), &req)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequest_WithAuth(t *testing.T) {
	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "Test",
	}

	auth := &AuthData{
		Username: "admin",
		Password: "password",
	}

	result := req.WithAuth(auth)

	if result != req {
		t.Error("WithAuth should return the same request")
	}

	if req.Auth != auth {
		t.Error("Auth was not set correctly")
	}
}

func TestRequest_GetParams(t *testing.T) {
	tests := []struct {
		target  any
		name    string
		params  json.RawMessage
		wantErr bool
	}{
		{
			name:    "unmarshal map",
			params:  json.RawMessage(`{"id":0,"on":true}`),
			target:  &map[string]any{},
			wantErr: false,
		},
		{
			name:    "unmarshal struct",
			params:  json.RawMessage(`{"id":0,"on":true}`),
			target:  &struct{ ID int }{},
			wantErr: false,
		},
		{
			name:    "empty params",
			params:  nil,
			target:  &map[string]any{},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			params:  json.RawMessage(`{invalid}`),
			target:  &map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "Test",
				Params:  tt.params,
			}

			err := req.GetParams(tt.target)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequest_String(t *testing.T) {
	tests := []struct {
		name string
		req  *Request
		want string
	}{
		{
			name: "request with ID",
			req: &Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "Test",
			},
			want: "Request{ID: 1, Method: Test}",
		},
		{
			name: "notification",
			req: &Request{
				JSONRPC: "2.0",
				Method:  "Notify",
			},
			want: "Notification{Method: Notify}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequest_IsNotification(t *testing.T) {
	tests := []struct {
		req  *Request
		name string
		want bool
	}{
		{
			name: "request with ID",
			req:  &Request{ID: 1},
			want: false,
		},
		{
			name: "notification",
			req:  &Request{},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.IsNotification(); got != tt.want {
				t.Errorf("IsNotification() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewBatchRequest(t *testing.T) {
	method := "Switch.Set"
	params := map[string]any{"id": 0, "on": true}

	br := NewBatchRequest(method, params)

	if br.Method != method {
		t.Errorf("Method = %v, want %v", br.Method, method)
	}

	// Check that params is the same object (maps can't be compared directly)
	if br.Params == nil {
		t.Error("Params should not be nil")
	}
}

func TestAuthData_Fields(t *testing.T) {
	auth := &AuthData{
		Username:  "admin",
		Password:  "password",
		Realm:     "shelly",
		Nonce:     "abc123",
		CNonce:    "def456",
		NC:        1,
		Algorithm: "SHA-256",
		Response:  "hash123",
	}

	// Test that all fields are set correctly
	if auth.Username != "admin" {
		t.Errorf("Username = %v, want admin", auth.Username)
	}
	if auth.Password != "password" {
		t.Errorf("Password = %v, want password", auth.Password)
	}
	if auth.Realm != "shelly" {
		t.Errorf("Realm = %v, want shelly", auth.Realm)
	}
	if auth.Nonce != "abc123" {
		t.Errorf("Nonce = %v, want abc123", auth.Nonce)
	}
	if auth.CNonce != "def456" {
		t.Errorf("CNonce = %v, want def456", auth.CNonce)
	}
	if auth.NC != 1 {
		t.Errorf("NC = %v, want 1", auth.NC)
	}
	if auth.Algorithm != "SHA-256" {
		t.Errorf("Algorithm = %v, want SHA-256", auth.Algorithm)
	}
	if auth.Response != "hash123" {
		t.Errorf("Response = %v, want hash123", auth.Response)
	}
}

func TestRequestBuilder_ConcurrentAccess(t *testing.T) {
	rb := NewRequestBuilder()
	done := make(chan bool)

	// Start multiple goroutines building requests concurrently
	for i := 0; i < 100; i++ {
		go func() {
			_, err := rb.Build("Test", nil)
			if err != nil {
				t.Errorf("Build() error = %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	// All requests should have received unique IDs
	currentID := rb.CurrentID()
	if currentID != 100 {
		t.Errorf("CurrentID = %v, want 100", currentID)
	}
}
