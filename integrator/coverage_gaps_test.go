package integrator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// jwtWithExp constructs a minimal 3-part JWT whose middle section encodes
// {"exp": <unix>}.  ParseJWTClaims only base64-decodes the payload, so the
// header and signature fields are arbitrary strings.
func jwtWithExp(exp int64) string {
	claims, _ := json.Marshal(map[string]int64{"exp": exp})
	payload := base64.RawURLEncoding.EncodeToString(claims)
	return "header." + payload + ".sig"
}

// ---------------------------------------------------------------------------
// client.go: RefreshToken, Connect (error paths), Close, Authenticate errors
// ---------------------------------------------------------------------------

func TestClient_RefreshToken_NeedsRefresh(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := AuthResponse{
			IsOK: true,
			Data: &AuthData{
				Token:     "refreshed",
				ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// authData is nil → RefreshToken must call Authenticate.
	client := NewWithOptions("tag", "token", server.URL, nil)
	if err := client.RefreshToken(context.Background()); err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if !client.IsAuthenticated() {
		t.Error("client not authenticated after RefreshToken")
	}
}

func TestClient_RefreshToken_AlreadyValid(t *testing.T) {
	client := New("tag", "token")
	client.authData = &AuthData{
		Token:     "valid",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	// No HTTP server: must be a no-op.
	if err := client.RefreshToken(context.Background()); err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if client.authData.Token != "valid" {
		t.Error("token should be unchanged when still valid")
	}
}

func TestClient_Connect_NotAuthenticated(t *testing.T) {
	client := New("tag", "token")
	_, err := client.Connect(context.Background(), "host.example.com", nil)
	if err == nil {
		t.Fatal("Connect() expected error when not authenticated")
	}
}

func TestClient_Connect_ExpiredToken(t *testing.T) {
	client := New("tag", "token")
	client.authData = &AuthData{
		Token:     "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	_, err := client.Connect(context.Background(), "host.example.com", nil)
	if err == nil {
		t.Fatal("Connect() expected error with expired token")
	}
}

func TestClient_Close_NoConnections(t *testing.T) {
	client := New("tag", "token")
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestClient_Close_WithInjectedConnection(t *testing.T) {
	client := New("tag", "token")
	mockWS := &mockWSConnector{}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
		host:    "host1",
	}
	client.connections["host1"] = conn

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !mockWS.closed {
		t.Error("underlying WebSocket not closed after client.Close()")
	}
}

func TestClient_Authenticate_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	client := NewWithOptions("tag", "token", server.URL, nil)
	if err := client.Authenticate(context.Background()); err == nil {
		t.Fatal("Authenticate() expected error for malformed JSON response")
	}
}

func TestClient_Authenticate_AuthFailedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := AuthResponse{
			IsOK:   false,
			Errors: json.RawMessage(`"invalid integrator tag"`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithOptions("tag", "token", server.URL, nil)
	if err := client.Authenticate(context.Background()); err == nil {
		t.Fatal("Authenticate() expected error when IsOK=false")
	}
}

func TestClient_Authenticate_InvalidURL(t *testing.T) {
	// Invalid URL → http.NewRequestWithContext fails.
	client := NewWithOptions("tag", "token", "://bad", nil)
	if err := client.Authenticate(context.Background()); err == nil {
		t.Fatal("Authenticate() expected error for invalid URL")
	}
}

// ---------------------------------------------------------------------------
// connection.go: newConnection dial error, readLoop error/nil-ws/closed exits,
//
//	pingLoop closed-channel exit
//
// ---------------------------------------------------------------------------

func TestNewConnection_DialError(t *testing.T) {
	// A plain HTTP server rejects WebSocket upgrade → dial must fail.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not ws", http.StatusBadRequest)
	}))
	defer server.Close()

	host := server.Listener.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := newConnection(ctx, host, "token", DefaultConnectOptions())
	if err == nil {
		t.Fatal("newConnection() expected dial error against plain HTTP server")
	}
}

func TestConnection_readLoop_ErrorPath(t *testing.T) {
	errCh := make(chan error, 1)
	mockWS := &mockWSConnector{
		readFunc: func() (int, []byte, error) {
			return 0, nil, &wsTestError{"simulated read error"}
		},
	}

	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
	}
	conn.OnError(func(err error) {
		select {
		case errCh <- err:
		default:
		}
	})

	go conn.readLoop()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected non-nil error from readLoop on WS read failure")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("readLoop did not deliver error in time")
	}
}

func TestConnection_readLoop_ExitsOnClosedCh(t *testing.T) {
	conn := &Connection{
		closeCh: make(chan struct{}),
	}
	close(conn.closeCh)
	conn.closed = true

	done := make(chan struct{})
	go func() {
		conn.readLoop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("readLoop did not exit after closeCh closed")
	}
}

func TestConnection_readLoop_ExitsOnNilWS(t *testing.T) {
	conn := &Connection{
		ws:      nil,
		closeCh: make(chan struct{}),
	}

	done := make(chan struct{})
	go func() {
		conn.readLoop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("readLoop did not exit with nil ws")
	}
}

func TestConnection_pingLoop_ExitsOnClosedCh(t *testing.T) {
	conn := &Connection{closeCh: make(chan struct{})}
	close(conn.closeCh)

	done := make(chan struct{})
	go func() {
		conn.pingLoop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("pingLoop did not exit after closeCh closed")
	}
}

// wsTestError satisfies the error interface for simulated WS failures.
type wsTestError struct{ msg string }

func (e *wsTestError) Error() string { return e.msg }

// ---------------------------------------------------------------------------
// accounts.go: UnsubscribeDevice
// ---------------------------------------------------------------------------

// withTLSTransport temporarily replaces http.DefaultTransport with the TLS
// client from a httptest.Server so that UnsubscribeDevice's hardcoded
// http.DefaultClient can reach our test server.  It restores the original on
// cleanup.
func withTLSTransport(t *testing.T, server *httptest.Server) {
	t.Helper()
	orig := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = orig })
}

func TestClient_UnsubscribeDevice_Success(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %v, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		if r.Form.Get("id") != "dev123" {
			t.Errorf("id = %v, want dev123", r.Form.Get("id"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	withTLSTransport(t, server)

	client := New("tag", "token")
	client.authData = &AuthData{
		Token:     "valid",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}

	// Strip the leading "https://" from the test server URL to get just "host:port".
	addr := server.Listener.Addr().String()
	if err := client.UnsubscribeDevice(context.Background(), addr, "dev123"); err != nil {
		t.Fatalf("UnsubscribeDevice() error = %v", err)
	}
}

func TestClient_UnsubscribeDevice_NotAuthenticated(t *testing.T) {
	client := New("tag", "token")
	if err := client.UnsubscribeDevice(context.Background(), "host.example.com", "dev1"); err == nil {
		t.Fatal("expected error when not authenticated")
	}
}

func TestClient_UnsubscribeDevice_NonOKStatus(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	withTLSTransport(t, server)

	client := New("tag", "token")
	client.authData = &AuthData{
		Token:     "valid",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	addr := server.Listener.Addr().String()
	if err := client.UnsubscribeDevice(context.Background(), addr, "dev1"); err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

// ---------------------------------------------------------------------------
// fleet.go: Connect (already connected, error), ConnectAll
// ---------------------------------------------------------------------------

func TestFleetManager_Connect_AlreadyConnected(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	mockWS := &mockWSConnector{}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
		host:    "host1",
	}
	fm.connections["host1"] = conn

	got, err := fm.Connect(context.Background(), "host1", nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if got != conn {
		t.Error("expected the existing cached connection")
	}
}

func TestFleetManager_Connect_ErrorPath(t *testing.T) {
	client := New("tag", "token")
	client.authData = &AuthData{
		Token:     "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	fm := NewFleetManager(client)

	_, err := fm.Connect(context.Background(), "host.example.com", nil)
	if err == nil {
		t.Fatal("FleetManager.Connect() expected error with expired token")
	}
}

func TestFleetManager_ConnectAll_NoHosts(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	errs := fm.ConnectAll(context.Background(), nil)
	if len(errs) != 0 {
		t.Errorf("ConnectAll() errors = %v, want empty", errs)
	}
}

func TestFleetManager_ConnectAll_ErrorPerHost(t *testing.T) {
	client := New("tag", "token")
	client.authData = &AuthData{
		Token:     "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	fm := NewFleetManager(client)
	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", Host: "host1"})
	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev2", Host: "host2"})

	errs := fm.ConnectAll(context.Background(), nil)
	if len(errs) != 2 {
		t.Errorf("ConnectAll() error count = %d, want 2", len(errs))
	}
}

// ---------------------------------------------------------------------------
// fleet.go: SendBatchCommands success branch, ListGroups sort
// ---------------------------------------------------------------------------

func TestFleetManager_SendBatchCommands_SuccessBranch(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	mockWS := &mockWSConnector{
		writeFunc: func(_ int, _ []byte) error { return nil },
	}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
		host:    "host1",
	}
	fm.connections["host1"] = conn

	_ = fm.accounts.AddDevice("user1", &AccountDevice{
		DeviceID:     "dev1",
		AccessGroups: "01",
		Host:         "host1",
	})

	results := fm.SendBatchCommands(context.Background(), []BatchCommand{
		{DeviceID: "dev1", Action: "relay", Params: map[string]any{"turn": "on"}},
	})
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if !results[0].Success {
		t.Errorf("results[0].Success = false, want true (err: %s)", results[0].Error)
	}
}

func TestFleetManager_ListGroups_SortOrder(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	fm.CreateGroup("g2", "Zeta", nil)
	fm.CreateGroup("g1", "Alpha", nil)
	fm.CreateGroup("g3", "Mu", nil)

	groups := fm.ListGroups()
	if len(groups) != 3 {
		t.Fatalf("len(ListGroups()) = %d, want 3", len(groups))
	}
	if groups[0].Name != "Alpha" || groups[1].Name != "Mu" || groups[2].Name != "Zeta" {
		t.Errorf("groups not sorted: %v %v %v", groups[0].Name, groups[1].Name, groups[2].Name)
	}
}

// ---------------------------------------------------------------------------
// auth.go: autoRefreshLoop ctx.Done branch, VerifyCallbackToken parse error
// ---------------------------------------------------------------------------

func TestTokenManager_AutoRefreshLoop_ContextCancel(t *testing.T) {
	client := New("tag", "token")
	client.authData = &AuthData{
		Token:     "valid",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	tm := NewTokenManager(client)

	ctx, cancel := context.WithCancel(context.Background())
	tm.StartAutoRefresh(ctx)
	time.Sleep(10 * time.Millisecond)
	// Canceling the context exercises the ctx.Done case in autoRefreshLoop.
	cancel()
	// Allow the goroutine to observe the cancellation.
	time.Sleep(20 * time.Millisecond)
}

func TestCallbackTokenVerifier_VerifyCallbackToken_ParseError(t *testing.T) {
	v := &CallbackTokenVerifier{}
	_, err := v.VerifyCallbackToken("not.a.jwt")
	if err == nil {
		t.Fatal("VerifyCallbackToken() expected error for non-JWT string")
	}
}

func TestCallbackTokenVerifier_VerifyCallbackToken_ExpiredJWT(t *testing.T) {
	token := jwtWithExp(time.Now().Add(-1 * time.Hour).Unix())
	v := &CallbackTokenVerifier{}
	_, err := v.VerifyCallbackToken(token)
	if err == nil {
		t.Fatal("VerifyCallbackToken() expected error for expired token")
	}
}

func TestCallbackTokenVerifier_VerifyCallbackToken_Valid(t *testing.T) {
	token := jwtWithExp(time.Now().Add(1 * time.Hour).Unix())
	v := &CallbackTokenVerifier{}
	ct, err := v.VerifyCallbackToken(token)
	if err != nil {
		t.Fatalf("VerifyCallbackToken() error = %v", err)
	}
	if ct.Token != token {
		t.Errorf("Token = %v, want %v", ct.Token, token)
	}
}

// ---------------------------------------------------------------------------
// analytics.go: RecentErrors (limit ≤ 0 and limit > len), GetSummary (>3 peak hours)
// ---------------------------------------------------------------------------

func TestErrorTracker_RecentErrors_ZeroLimit(t *testing.T) {
	tracker := NewErrorTracker()
	tracker.RecordError("timeout", "msg1", "dev1", "host1")
	tracker.RecordError("conn", "msg2", "dev2", "host2")

	if got := tracker.RecentErrors(0); len(got) != 2 {
		t.Errorf("RecentErrors(0) = %d items, want 2", len(got))
	}
	if got := tracker.RecentErrors(-5); len(got) != 2 {
		t.Errorf("RecentErrors(-5) = %d items, want 2", len(got))
	}
}

func TestErrorTracker_RecentErrors_LimitExceedsLen(t *testing.T) {
	tracker := NewErrorTracker()
	tracker.RecordError("timeout", "msg", "dev1", "host1")

	if got := tracker.RecentErrors(100); len(got) != 1 {
		t.Errorf("RecentErrors(100) = %d items, want 1", len(got))
	}
}

func TestAnalytics_GetSummary_PeakHoursTruncated(t *testing.T) {
	a := NewAnalytics()
	// Record status changes across 5 distinct hours so len(peakHoursList) > 3.
	// We set the internal hourlyActivity map directly to avoid time-zone issues.
	a.devicePatterns.mu.Lock()
	for h := 0; h < 5; h++ {
		a.devicePatterns.hourlyActivity[h] = int64(h + 1)
	}
	a.devicePatterns.mu.Unlock()

	summary := a.GetSummary()
	if len(summary.DevicePatterns.PeakHours) > 3 {
		t.Errorf("PeakHours len = %d, want ≤ 3", len(summary.DevicePatterns.PeakHours))
	}
}

// ---------------------------------------------------------------------------
// provisioning.go: provisionDevice success path (no actions, with DelayAfter)
// ---------------------------------------------------------------------------

func TestProvisioningManager_ProvisionDevice_SuccessNoActions(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{
		DeviceID:     "dev1",
		DeviceType:   "SHSW-1",
		AccessGroups: "01",
		Host:         "host1",
	})

	mockWS := &mockWSConnector{
		writeFunc: func(_ int, _ []byte) error { return nil },
	}
	fm.connections["host1"] = &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
		host:    "host1",
	}

	template := pm.CreateTemplate("t1", "T", nil)
	template.DeviceTypes = []string{"SHSW-1"}
	template.Actions = []TemplateAction{} // no actions → success path

	_, _ = pm.CreateTask("task1", "T1", "t1", []string{"dev1"})
	if err := pm.ExecuteTask(context.Background(), "task1"); err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}

	task, _ := pm.GetTask("task1")
	if task.Status != TaskStatusCompleted {
		t.Errorf("Status = %v, want %v", task.Status, TaskStatusCompleted)
	}
	progress, _ := pm.GetProgress("task1")
	if progress.CompletedDevices != 1 {
		t.Errorf("CompletedDevices = %d, want 1", progress.CompletedDevices)
	}
}

func TestProvisioningManager_ProvisionDevice_DelayAfterBranch(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{
		DeviceID:     "dev1",
		DeviceType:   "SHSW-1",
		AccessGroups: "01",
		Host:         "host1",
	})
	fm.connections["host1"] = &Connection{
		ws:      &mockWSConnector{writeFunc: func(_ int, _ []byte) error { return nil }},
		closeCh: make(chan struct{}),
		host:    "host1",
	}

	template := pm.CreateTemplate("t1", "T", nil)
	template.DeviceTypes = []string{"SHSW-1"}
	// 1 ms delay exercises the `if action.DelayAfter > 0` branch.
	template.Actions = []TemplateAction{
		{Type: "relay", Params: map[string]any{"turn": "on"}, DelayAfter: time.Millisecond},
	}

	_, _ = pm.CreateTask("task1", "T1", "t1", []string{"dev1"})
	if err := pm.ExecuteTask(context.Background(), "task1"); err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}

	task, _ := pm.GetTask("task1")
	if task.Status != TaskStatusCompleted {
		t.Errorf("Status = %v, want %v", task.Status, TaskStatusCompleted)
	}
}
