package integrator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	client := New("test-tag", "test-token")

	if client == nil {
		t.Fatal("New returned nil")
	}
	if client.integratorTag != "test-tag" {
		t.Errorf("integratorTag = %v, want test-tag", client.integratorTag)
	}
	if client.token != "test-token" {
		t.Errorf("token = %v, want test-token", client.token)
	}
	if client.apiURL != DefaultAPIURL {
		t.Errorf("apiURL = %v, want %v", client.apiURL, DefaultAPIURL)
	}
}

func TestNewWithOptions(t *testing.T) {
	customHTTP := &http.Client{Timeout: 60 * time.Second}
	client := NewWithOptions("tag", "token", "https://custom.api.com", customHTTP)

	if client.apiURL != "https://custom.api.com" {
		t.Errorf("apiURL = %v, want https://custom.api.com", client.apiURL)
	}
	if client.httpClient != customHTTP {
		t.Error("httpClient not set correctly")
	}
}

func TestClient_Authenticate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %v, want POST", r.Method)
		}
		if r.URL.Path != AuthEndpoint {
			t.Errorf("path = %v, want %v", r.URL.Path, AuthEndpoint)
		}

		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req.IntegratorTag != "test-tag" {
			t.Errorf("itg = %v, want test-tag", req.IntegratorTag)
		}

		resp := AuthResponse{
			IsOK: true,
			Data: &AuthData{
				Token:     "jwt-token-here",
				ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithOptions("test-tag", "test-token", server.URL, nil)

	err := client.Authenticate(context.Background())
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if !client.IsAuthenticated() {
		t.Error("IsAuthenticated() = false, want true")
	}

	token, err := client.GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	//nolint:gosec // G101: This is a test token, not a real credential
	if token != "jwt-token-here" {
		t.Errorf("token = %v, want jwt-token-here", token)
	}
}

func TestClient_Authenticate_Failed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := AuthResponse{
			IsOK:   false,
			Errors: json.RawMessage(`"invalid credentials"`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithOptions("bad-tag", "bad-token", server.URL, nil)

	err := client.Authenticate(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestClient_IsAuthenticated(t *testing.T) {
	client := New("tag", "token")

	// Not authenticated initially
	if client.IsAuthenticated() {
		t.Error("IsAuthenticated() = true, want false (no auth data)")
	}

	// Set expired token
	client.authData = &AuthData{
		Token:     "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	if client.IsAuthenticated() {
		t.Error("IsAuthenticated() = true, want false (expired)")
	}

	// Set valid token
	client.authData = &AuthData{
		Token:     "valid",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	if !client.IsAuthenticated() {
		t.Error("IsAuthenticated() = false, want true")
	}
}

func TestClient_GetToken_NotAuthenticated(t *testing.T) {
	client := New("tag", "token")

	_, err := client.GetToken()
	if err != ErrNotAuthenticated {
		t.Errorf("GetToken() error = %v, want ErrNotAuthenticated", err)
	}
}

func TestClient_GetToken_Expired(t *testing.T) {
	client := New("tag", "token")
	client.authData = &AuthData{
		Token:     "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}

	_, err := client.GetToken()
	if err != ErrTokenExpired {
		t.Errorf("GetToken() error = %v, want ErrTokenExpired", err)
	}
}

func TestClient_TokenExpiresAt(t *testing.T) {
	client := New("tag", "token")

	// Not authenticated
	_, err := client.TokenExpiresAt()
	if err != ErrNotAuthenticated {
		t.Errorf("TokenExpiresAt() error = %v, want ErrNotAuthenticated", err)
	}

	// With auth data
	expTime := time.Now().Add(24 * time.Hour)
	client.authData = &AuthData{
		Token:     "token",
		ExpiresAt: expTime.Unix(),
	}

	got, err := client.TokenExpiresAt()
	if err != nil {
		t.Fatalf("TokenExpiresAt() error = %v", err)
	}
	if got.Unix() != expTime.Unix() {
		t.Errorf("TokenExpiresAt() = %v, want %v", got, expTime)
	}
}

func TestClient_ActiveConnections(t *testing.T) {
	client := New("tag", "token")

	// Initially empty
	conns := client.ActiveConnections()
	if len(conns) != 0 {
		t.Errorf("ActiveConnections() len = %d, want 0", len(conns))
	}

	// Add a connection
	client.connections["host1"] = &Connection{host: "host1"}
	client.connections["host2"] = &Connection{host: "host2"}

	conns = client.ActiveConnections()
	if len(conns) != 2 {
		t.Errorf("ActiveConnections() len = %d, want 2", len(conns))
	}
}

func TestClient_GetConnection(t *testing.T) {
	client := New("tag", "token")
	conn := &Connection{host: "test-host"}
	client.connections["test-host"] = conn

	got := client.GetConnection("test-host")
	if got != conn {
		t.Error("GetConnection() returned wrong connection")
	}

	got = client.GetConnection("nonexistent")
	if got != nil {
		t.Error("GetConnection() should return nil for nonexistent host")
	}
}

func TestClient_Disconnect(t *testing.T) {
	client := New("tag", "token")
	conn := &Connection{host: "test-host", closeCh: make(chan struct{})}
	client.connections["test-host"] = conn

	err := client.Disconnect("test-host")
	if err != nil {
		t.Errorf("Disconnect() error = %v", err)
	}

	if _, ok := client.connections["test-host"]; ok {
		t.Error("connection should be removed after Disconnect()")
	}

	// Disconnect nonexistent host should not error
	err = client.Disconnect("nonexistent")
	if err != nil {
		t.Errorf("Disconnect() nonexistent error = %v", err)
	}
}

func TestClient_DisconnectAll(t *testing.T) {
	client := New("tag", "token")
	client.connections["host1"] = &Connection{host: "host1", closeCh: make(chan struct{})}
	client.connections["host2"] = &Connection{host: "host2", closeCh: make(chan struct{})}

	err := client.DisconnectAll()
	if err != nil {
		t.Errorf("DisconnectAll() error = %v", err)
	}

	if len(client.connections) != 0 {
		t.Errorf("connections len = %d, want 0", len(client.connections))
	}
}

func TestAuthData_ExpiresTime(t *testing.T) {
	now := time.Now()
	auth := &AuthData{
		Token:     "token",
		ExpiresAt: now.Unix(),
	}

	got := auth.ExpiresTime()
	if got.Unix() != now.Unix() {
		t.Errorf("ExpiresTime() = %v, want %v", got, now)
	}
}

func TestAuthData_IsExpired(t *testing.T) {
	// Expired
	auth := &AuthData{
		Token:     "token",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	if !auth.IsExpired() {
		t.Error("IsExpired() = false, want true")
	}

	// Not expired
	auth.ExpiresAt = time.Now().Add(1 * time.Hour).Unix()
	if auth.IsExpired() {
		t.Error("IsExpired() = true, want false")
	}
}

func TestWSMessage_GetDeviceID(t *testing.T) {
	// Using DeviceID field
	msg := WSMessage{DeviceID: "device1"}
	if msg.GetDeviceID() != "device1" {
		t.Errorf("GetDeviceID() = %v, want device1", msg.GetDeviceID())
	}

	// Using Device field as fallback
	msg = WSMessage{Device: "device2"}
	if msg.GetDeviceID() != "device2" {
		t.Errorf("GetDeviceID() = %v, want device2", msg.GetDeviceID())
	}

	// DeviceID takes precedence
	msg = WSMessage{DeviceID: "device1", Device: "device2"}
	if msg.GetDeviceID() != "device1" {
		t.Errorf("GetDeviceID() = %v, want device1", msg.GetDeviceID())
	}
}

func TestWSMessage_IsOnline(t *testing.T) {
	// Nil online
	msg := WSMessage{}
	if msg.IsOnline() {
		t.Error("IsOnline() = true, want false (nil)")
	}

	// Online = 0
	zero := 0
	msg = WSMessage{Online: &zero}
	if msg.IsOnline() {
		t.Error("IsOnline() = true, want false (0)")
	}

	// Online = 1
	one := 1
	msg = WSMessage{Online: &one}
	if !msg.IsOnline() {
		t.Error("IsOnline() = false, want true")
	}
}

func TestAccessGroup(t *testing.T) {
	tests := []struct {
		wantStr  string
		group    AccessGroup
		wantCtrl bool
	}{
		{group: AccessGroupReadOnly, wantStr: "read-only", wantCtrl: false},
		{group: AccessGroupControl, wantStr: "control", wantCtrl: true},
		{group: AccessGroup(0xFF), wantStr: "unknown", wantCtrl: true},
	}

	for _, tt := range tests {
		if got := tt.group.String(); got != tt.wantStr {
			t.Errorf("AccessGroup(%d).String() = %v, want %v", tt.group, got, tt.wantStr)
		}
		if got := tt.group.CanControl(); got != tt.wantCtrl {
			t.Errorf("AccessGroup(%d).CanControl() = %v, want %v", tt.group, got, tt.wantCtrl)
		}
	}
}

func TestDefaultConnectOptions(t *testing.T) {
	opts := DefaultConnectOptions()

	if opts.PingInterval != 30*time.Second {
		t.Errorf("PingInterval = %v, want 30s", opts.PingInterval)
	}
	if opts.ReadTimeout != 60*time.Second {
		t.Errorf("ReadTimeout = %v, want 60s", opts.ReadTimeout)
	}
	if opts.ReconnectDelay != 5*time.Second {
		t.Errorf("ReconnectDelay = %v, want 5s", opts.ReconnectDelay)
	}
	if opts.MaxReconnectAttempts != 0 {
		t.Errorf("MaxReconnectAttempts = %v, want 0", opts.MaxReconnectAttempts)
	}
}

func TestDefaultCloudServers(t *testing.T) {
	servers := DefaultCloudServers()

	if len(servers) == 0 {
		t.Error("DefaultCloudServers() returned empty list")
	}

	// Check first server
	if servers[0].Host == "" {
		t.Error("first server has empty host")
	}
	if servers[0].WSPort != 6113 {
		t.Errorf("first server WSPort = %d, want 6113", servers[0].WSPort)
	}
}
