package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

// mockWebSocketConn is a mock WebSocket connection for testing.
type mockWebSocketConn struct {
	writeErr    error
	readErr     error
	closeCh     chan struct{}
	messages    [][]byte
	writtenMsgs [][]byte
	msgIndex    int
	readDelay   time.Duration
	mu          sync.Mutex
	closed      bool
}

func newMockWebSocketConn() *mockWebSocketConn {
	return &mockWebSocketConn{
		closeCh: make(chan struct{}),
	}
}

func (m *mockWebSocketConn) ReadMessage() (int, []byte, error) {
	m.mu.Lock()

	if m.closed {
		m.mu.Unlock()
		return 0, nil, ErrWebSocketClosed
	}

	if m.readErr != nil {
		err := m.readErr
		m.mu.Unlock()
		return 0, nil, err
	}

	if m.readDelay > 0 {
		closeCh := m.closeCh
		m.mu.Unlock()
		// Wait for delay or close
		select {
		case <-time.After(m.readDelay):
		case <-closeCh:
			return 0, nil, ErrWebSocketClosed
		}
		m.mu.Lock()
	}

	if m.closed {
		m.mu.Unlock()
		return 0, nil, ErrWebSocketClosed
	}

	if m.msgIndex >= len(m.messages) {
		m.mu.Unlock()
		return 0, nil, ErrWebSocketClosed
	}

	msg := m.messages[m.msgIndex]
	m.msgIndex++
	m.mu.Unlock()
	return 1, msg, nil
}

func (m *mockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrWebSocketClosed
	}

	if m.writeErr != nil {
		return m.writeErr
	}

	m.writtenMsgs = append(m.writtenMsgs, data)
	return nil
}

func (m *mockWebSocketConn) Close() error {
	m.mu.Lock()
	if !m.closed {
		m.closed = true
		if m.closeCh != nil {
			close(m.closeCh)
		}
	}
	m.mu.Unlock()
	return nil
}

func (m *mockWebSocketConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockWebSocketConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// mockWebSocketDialer is a mock WebSocket dialer for testing.
type mockWebSocketDialer struct {
	conn    WebSocketConn
	dialErr error
}

func (m *mockWebSocketDialer) Dial(ctx context.Context, url string, headers http.Header) (WebSocketConn, error) {
	if m.dialErr != nil {
		return nil, m.dialErr
	}
	return m.conn, nil
}

func TestNewWebSocket(t *testing.T) {
	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client)

	if ws == nil {
		t.Fatal("NewWebSocket returned nil")
	}
	if ws.client != client {
		t.Error("client mismatch")
	}
	if ws.reconnectInterval != 5*time.Second {
		t.Errorf("reconnectInterval = %v, want 5s", ws.reconnectInterval)
	}
	if ws.maxReconnectInterval != 5*time.Minute {
		t.Errorf("maxReconnectInterval = %v, want 5m", ws.maxReconnectInterval)
	}
}

func TestNewWebSocketWithOptions(t *testing.T) {
	client := NewClient()
	mockDialer := &mockWebSocketDialer{}

	ws := NewWebSocket(client,
		WithDialer(mockDialer),
		WithReconnectInterval(10*time.Second),
		WithMaxReconnectInterval(10*time.Minute),
		WithPingInterval(60*time.Second),
		WithReadTimeout(120*time.Second),
	)

	if ws.dialer != mockDialer {
		t.Error("WithDialer option not applied")
	}
	if ws.reconnectInterval != 10*time.Second {
		t.Errorf("reconnectInterval = %v, want 10s", ws.reconnectInterval)
	}
	if ws.maxReconnectInterval != 10*time.Minute {
		t.Errorf("maxReconnectInterval = %v, want 10m", ws.maxReconnectInterval)
	}
	if ws.pingInterval != 60*time.Second {
		t.Errorf("pingInterval = %v, want 60s", ws.pingInterval)
	}
	if ws.readTimeout != 120*time.Second {
		t.Errorf("readTimeout = %v, want 120s", ws.readTimeout)
	}
}

func TestWebSocketConnect(t *testing.T) {
	mockConn := &mockWebSocketConn{}
	mockDialer := &mockWebSocketDialer{conn: mockConn}

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client, WithDialer(mockDialer))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if !ws.IsConnected() {
		t.Error("IsConnected() = false, want true")
	}
}

func TestWebSocketConnectAlreadyConnected(t *testing.T) {
	mockConn := &mockWebSocketConn{}
	mockDialer := &mockWebSocketDialer{conn: mockConn}

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client, WithDialer(mockDialer))

	ctx := context.Background()
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("First Connect failed: %v", err)
	}

	// Second connect should be no-op
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Second Connect failed: %v", err)
	}
}

func TestWebSocketConnectNoDialer(t *testing.T) {
	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client)

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestWebSocketConnectDialError(t *testing.T) {
	dialErr := errors.New("connection refused")
	mockDialer := &mockWebSocketDialer{dialErr: dialErr}

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client, WithDialer(mockDialer))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestWebSocketClose(t *testing.T) {
	mockConn := &mockWebSocketConn{}
	mockDialer := &mockWebSocketDialer{conn: mockConn}

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client, WithDialer(mockDialer))

	ctx := context.Background()
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if err := ws.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if ws.IsConnected() {
		t.Error("IsConnected() = true after Close()")
	}

	if !mockConn.closed {
		t.Error("Connection not closed")
	}
}

func TestWebSocketCloseNotConnected(t *testing.T) {
	client := NewClient()
	ws := NewWebSocket(client)

	// Should not error when not connected
	if err := ws.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestWebSocketBuildURL(t *testing.T) {
	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client)

	url, err := ws.buildWebSocketURL()
	if err != nil {
		t.Fatalf("buildWebSocketURL failed: %v", err)
	}

	expected := "wss://shelly-49-eu.shelly.cloud:6113/shelly/wss/hk_sock?t=test-token"
	if url != expected {
		t.Errorf("URL = %v, want %v", url, expected)
	}
}

func TestWebSocketBuildURLNoBaseURL(t *testing.T) {
	client := NewClient(
		WithAccessToken("test-token"),
	)
	client.baseURL = ""

	ws := NewWebSocket(client)

	_, err := ws.buildWebSocketURL()
	if err != ErrNoUserAPIURL {
		t.Errorf("Expected ErrNoUserAPIURL, got %v", err)
	}
}

func TestWebSocketBuildURLNoToken(t *testing.T) {
	client := NewClient(
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client)

	_, err := ws.buildWebSocketURL()
	if err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got %v", err)
	}
}

func TestWebSocketHandleMessage(t *testing.T) {
	client := NewClient()
	ws := NewWebSocket(client)

	var deviceID string
	ws.OnDeviceOnline(func(id string) {
		deviceID = id
	})

	msg := WebSocketMessage{
		Event:    EventDeviceOnline,
		DeviceID: "device123",
	}

	data, _ := json.Marshal(msg)
	ws.handleMessage(data)

	if deviceID != "device123" {
		t.Errorf("deviceID = %v, want device123", deviceID)
	}
}

func TestWebSocketHandleInvalidMessage(t *testing.T) {
	client := NewClient()
	ws := NewWebSocket(client)

	var called bool
	ws.OnMessage(func(msg *WebSocketMessage) {
		called = true
	})

	// Should not panic on invalid JSON
	ws.handleMessage([]byte("not valid json"))

	if called {
		t.Error("Handler should not be called for invalid JSON")
	}
}

func TestWebSocketEventHandlers(t *testing.T) {
	client := NewClient()
	ws := NewWebSocket(client)

	var onlineCalled, offlineCalled, statusChangeCalled bool
	var notifyStatusCalled, notifyFullStatusCalled, notifyEventCalled bool
	var messageCalled bool

	ws.OnDeviceOnline(func(deviceID string) {
		onlineCalled = true
	})

	ws.OnDeviceOffline(func(deviceID string) {
		offlineCalled = true
	})

	ws.OnStatusChange(func(deviceID string, status json.RawMessage) {
		statusChangeCalled = true
	})

	ws.OnNotifyStatus(func(deviceID string, status json.RawMessage) {
		notifyStatusCalled = true
	})

	ws.OnNotifyFullStatus(func(deviceID string, status json.RawMessage) {
		notifyFullStatusCalled = true
	})

	ws.OnNotifyEvent(func(deviceID string, event json.RawMessage) {
		notifyEventCalled = true
	})

	ws.OnMessage(func(msg *WebSocketMessage) {
		messageCalled = true
	})

	// Dispatch all event types
	events := []string{
		EventDeviceOnline,
		EventDeviceOffline,
		EventDeviceStatusChange,
		EventNotifyStatus,
		EventNotifyFullStatus,
		EventNotifyEvent,
	}

	for _, event := range events {
		msg := WebSocketMessage{Event: event, DeviceID: "test"}
		data, _ := json.Marshal(msg)
		ws.handleMessage(data)
	}

	if !onlineCalled {
		t.Error("OnDeviceOnline not called")
	}
	if !offlineCalled {
		t.Error("OnDeviceOffline not called")
	}
	if !statusChangeCalled {
		t.Error("OnStatusChange not called")
	}
	if !notifyStatusCalled {
		t.Error("OnNotifyStatus not called")
	}
	if !notifyFullStatusCalled {
		t.Error("OnNotifyFullStatus not called")
	}
	if !notifyEventCalled {
		t.Error("OnNotifyEvent not called")
	}
	if !messageCalled {
		t.Error("OnMessage not called")
	}
}

func TestWebSocketSendMessage(t *testing.T) {
	mockConn := &mockWebSocketConn{}
	mockDialer := &mockWebSocketDialer{conn: mockConn}

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client, WithDialer(mockDialer))

	ctx := context.Background()
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	msg := map[string]string{"action": "ping"}
	if err := ws.SendMessage(ctx, msg); err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if len(mockConn.writtenMsgs) != 1 {
		t.Errorf("writtenMsgs count = %v, want 1", len(mockConn.writtenMsgs))
	}
}

func TestWebSocketSendMessageNotConnected(t *testing.T) {
	client := NewClient()
	ws := NewWebSocket(client)

	ctx := context.Background()
	err := ws.SendMessage(ctx, map[string]string{"action": "ping"})
	if err != ErrWebSocketNotConnected {
		t.Errorf("Expected ErrWebSocketNotConnected, got %v", err)
	}
}

func TestWebSocketSendMessageWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	mockConn := &mockWebSocketConn{writeErr: writeErr}
	mockDialer := &mockWebSocketDialer{conn: mockConn}

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client, WithDialer(mockDialer))

	ctx := context.Background()
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	err := ws.SendMessage(ctx, map[string]string{"action": "ping"})
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestClientGetWebSocketURL(t *testing.T) {
	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	url, err := client.GetWebSocketURL()
	if err != nil {
		t.Fatalf("GetWebSocketURL failed: %v", err)
	}

	expected := "wss://shelly-49-eu.shelly.cloud:6113/shelly/wss/hk_sock?t=test-token"
	if url != expected {
		t.Errorf("URL = %v, want %v", url, expected)
	}
}

func TestClientGetWebSocketURLNoBaseURL(t *testing.T) {
	client := NewClient(
		WithAccessToken("test-token"),
	)
	client.baseURL = ""

	_, err := client.GetWebSocketURL()
	if err != ErrNoUserAPIURL {
		t.Errorf("Expected ErrNoUserAPIURL, got %v", err)
	}
}

func TestClientGetWebSocketURLNoToken(t *testing.T) {
	client := NewClient(
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	_, err := client.GetWebSocketURL()
	if err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got %v", err)
	}
}

func TestExtractHostname(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "https with path",
			input: "https://example.com/path",
			want:  "example.com",
		},
		{
			name:  "https with port",
			input: "https://example.com:8080",
			want:  "example.com",
		},
		{
			name:  "http",
			input: "http://example.com",
			want:  "example.com",
		},
		{
			name:  "no protocol",
			input: "example.com/path",
			want:  "example.com",
		},
		{
			name:  "no protocol with port",
			input: "example.com:8080",
			want:  "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHostname(tt.input)
			if got != tt.want {
				t.Errorf("extractHostname() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebSocketReadLoop(t *testing.T) {
	msg := WebSocketMessage{
		Event:    EventDeviceOnline,
		DeviceID: "device123",
	}
	msgData, _ := json.Marshal(msg)

	mockConn := newMockWebSocketConn()
	mockConn.messages = [][]byte{msgData}
	mockDialer := &mockWebSocketDialer{conn: mockConn}

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client, WithDialer(mockDialer))

	var deviceID string
	ws.OnDeviceOnline(func(id string) {
		deviceID = id
	})

	ctx := context.Background()
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Read loop should process message and then return on EOF
	err := ws.readLoop(ctx)
	if !errors.Is(err, ErrWebSocketClosed) {
		t.Errorf("readLoop() error = %v, want ErrWebSocketClosed", err)
	}

	if deviceID != "device123" {
		t.Errorf("deviceID = %v, want device123", deviceID)
	}
}

func TestWebSocketReadLoopContextCancel(t *testing.T) {
	mockConn := newMockWebSocketConn()
	mockConn.readDelay = 5 * time.Second // Long delay
	mockConn.messages = [][]byte{{}}
	mockDialer := &mockWebSocketDialer{conn: mockConn}

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client, WithDialer(mockDialer))

	ctx, cancel := context.WithCancel(context.Background())
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Run readLoop in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- ws.readLoop(ctx)
	}()

	// Cancel context after short delay
	time.Sleep(50 * time.Millisecond)
	cancel()
	// Also close the connection to unblock ReadMessage
	mockConn.Close()

	// Wait for readLoop to return with timeout
	select {
	case err := <-errCh:
		// Either context.Canceled or ErrWebSocketClosed is acceptable
		if !errors.Is(err, context.Canceled) && !errors.Is(err, ErrWebSocketClosed) {
			t.Errorf("readLoop() error = %v, want context.Canceled or ErrWebSocketClosed", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("readLoop() did not return in time")
	}
}

func TestWebSocketListen(t *testing.T) {
	// Create a mock connection that will close after one message
	msg := WebSocketMessage{
		Event:    EventDeviceOnline,
		DeviceID: "device123",
	}
	msgData, _ := json.Marshal(msg)

	mockConn := newMockWebSocketConn()
	mockConn.messages = [][]byte{msgData}
	mockDialer := &mockWebSocketDialer{conn: mockConn}

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client, WithDialer(mockDialer))

	var deviceID string
	ws.OnDeviceOnline(func(id string) {
		deviceID = id
	})

	ctx, cancel := context.WithCancel(context.Background())

	// Connect first
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Run Listen in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- ws.Listen(ctx)
	}()

	// Wait a bit for message to be processed, then cancel
	time.Sleep(100 * time.Millisecond)
	cancel()
	mockConn.Close()

	// Wait for Listen to return
	select {
	case <-errCh:
		// Listen returned (either error or nil is fine)
	case <-time.After(2 * time.Second):
		t.Error("Listen() did not return in time")
	}

	if deviceID != "device123" {
		t.Errorf("deviceID = %v, want device123", deviceID)
	}
}

func TestWebSocketListenContextCancel(t *testing.T) {
	mockConn := newMockWebSocketConn()
	mockConn.readDelay = 10 * time.Second // Long delay
	mockConn.messages = [][]byte{{}}
	mockDialer := &mockWebSocketDialer{conn: mockConn}

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client, WithDialer(mockDialer))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Connect first
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Listen should exit when context is canceled
	errCh := make(chan error, 1)
	go func() {
		errCh <- ws.Listen(ctx)
	}()

	// Close connection to help unblock
	time.Sleep(50 * time.Millisecond)
	mockConn.Close()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			// Might also return nil if stopped via stopCh
			if !errors.Is(err, ErrWebSocketClosed) {
				t.Errorf("Listen() error = %v, want context error or ErrWebSocketClosed", err)
			}
		}
	case <-time.After(2 * time.Second):
		t.Error("Listen() did not return in time")
	}
}

func TestWebSocketListenStopChannel(t *testing.T) {
	mockConn := newMockWebSocketConn()
	mockConn.readDelay = 10 * time.Second
	mockConn.messages = [][]byte{{}}
	mockDialer := &mockWebSocketDialer{conn: mockConn}

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)

	ws := NewWebSocket(client, WithDialer(mockDialer))

	ctx := context.Background()

	// Connect first
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Run Listen in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- ws.Listen(ctx)
	}()

	// Close the WebSocket to trigger stop
	time.Sleep(50 * time.Millisecond)
	ws.Close()

	select {
	case err := <-errCh:
		// Should return nil when stopped via Close()
		if err != nil && !errors.Is(err, ErrWebSocketClosed) {
			t.Logf("Listen() returned error = %v (acceptable)", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Listen() did not return in time")
	}
}
