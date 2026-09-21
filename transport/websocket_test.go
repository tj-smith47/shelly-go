package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewWebSocket(t *testing.T) {
	tests := []struct {
		name string
		url  string
		opts []Option
	}{
		{
			name: "basic WebSocket",
			url:  "ws://192.168.1.100/rpc",
		},
		{
			name: "with reconnect",
			url:  "ws://192.168.1.100/rpc",
			opts: []Option{WithReconnect(true)},
		},
		{
			name: "with ping interval",
			url:  "ws://192.168.1.100/rpc",
			opts: []Option{WithPingInterval(30)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := NewWebSocket(tt.url, tt.opts...)
			if ws == nil {
				t.Fatal("NewWebSocket() returned nil")
			}
			if ws.url != tt.url {
				t.Errorf("url = %v, want %v", ws.url, tt.url)
			}
		})
	}
}

func TestWebSocket_Call(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc", WithTimeout(200*time.Millisecond))
	_, err := ws.Call(context.Background(), NewSimpleRequest("Switch.Set"))
	if err == nil {
		t.Error("Call() error = nil, want not implemented error")
	}
}

func TestWebSocket_Subscribe(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	handler := NotificationHandler(func(data json.RawMessage) {})
	err := ws.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() error = %v", err)
	}

	// Second subscribe should fail
	err = ws.Subscribe(handler)
	if err == nil {
		t.Error("second Subscribe() error = nil, want error")
	}
}

func TestWebSocket_Unsubscribe(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	handler := NotificationHandler(func(data json.RawMessage) {})
	ws.Subscribe(handler)

	err := ws.Unsubscribe()
	if err != nil {
		t.Errorf("Unsubscribe() error = %v", err)
	}

	// Should be able to subscribe again after unsubscribe
	err = ws.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() after Unsubscribe() error = %v", err)
	}
}

func TestWebSocket_State(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	if ws.State() != StateDisconnected {
		t.Errorf("initial State() = %v, want %v", ws.State(), StateDisconnected)
	}
}

func TestWebSocket_OnStateChange(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	called := false
	ws.OnStateChange(func(state ConnectionState) {
		called = true
	})

	ws.setState(StateConnected)

	if !called {
		t.Error("state change callback not called")
	}
}

func TestWebSocket_Close(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	err := ws.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if ws.State() != StateClosed {
		t.Errorf("State() after Close() = %v, want %v", ws.State(), StateClosed)
	}

	// Calling Close again should be safe
	err = ws.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestWebSocket_ClosedCall(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")
	ws.Close()

	_, err := ws.Call(context.Background(), NewSimpleRequest("Switch.Set"))
	if err == nil {
		t.Error("Call() on closed WebSocket error = nil, want error")
	}
}

func TestWebSocket_ConnectClosed(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")
	ws.Close()

	err := ws.Connect(context.Background())
	if err == nil {
		t.Error("Connect() on closed WebSocket should return error")
	}
}

func TestWebSocket_MultipleStateCallbacks(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	calls := make([]ConnectionState, 0)

	ws.OnStateChange(func(state ConnectionState) {
		calls = append(calls, state)
	})
	ws.OnStateChange(func(state ConnectionState) {
		calls = append(calls, state)
	})

	ws.setState(StateConnecting)

	if len(calls) != 2 {
		t.Errorf("expected 2 callbacks, got %d", len(calls))
	}
}

func TestWebSocket_handleMessage_Response(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	// Create a pending request
	respChan := make(chan *rpcResponse, 1)
	ws.pendingMu.Lock()
	ws.pending[1] = respChan
	ws.pendingMu.Unlock()

	// Simulate receiving a response
	message := []byte(`{"id":1,"result":{"output":true}}`)
	ws.handleMessage(message)

	// Check that response was received
	select {
	case resp := <-respChan:
		if resp == nil {
			t.Fatal("response is nil")
		}
		if resp.ID != 1 {
			t.Errorf("response ID = %d, want 1", resp.ID)
		}
	default:
		t.Error("response not received")
	}
}

func TestWebSocket_handleMessage_Notification(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	received := make(chan json.RawMessage, 1)
	ws.Subscribe(func(data json.RawMessage) {
		received <- data
	})

	// Simulate receiving a notification
	message := []byte(`{"method":"NotifyStatus","params":{"ts":1234}}`)
	ws.handleMessage(message)

	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Error("notification handler not called")
	}
}

func TestWebSocket_handleMessage_NotificationNoHandler(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	// Simulate receiving a notification without handler - should not panic
	message := []byte(`{"method":"NotifyStatus","params":{"ts":1234}}`)
	ws.handleMessage(message)
}

func TestWebSocket_handleMessage_InvalidJSON(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	// Should not panic on invalid JSON
	ws.handleMessage([]byte(`not json`))
}

func TestWebSocket_handleMessage_ResponseNoPending(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	// Response for non-existent request ID - should not panic
	message := []byte(`{"id":999,"result":{"output":true}}`)
	ws.handleMessage(message)
}

func TestBase64Encode(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		input []byte
	}{
		{
			name:  "empty",
			input: []byte{},
			want:  "",
		},
		{
			name:  "single byte",
			input: []byte{0x00},
			want:  "AA==",
		},
		{
			name:  "two bytes",
			input: []byte{0x00, 0x00},
			want:  "AAA=",
		},
		{
			name:  "three bytes",
			input: []byte{0x00, 0x00, 0x00},
			want:  "AAAA",
		},
		{
			name:  "hello",
			input: []byte("hello"),
			want:  "aGVsbG8=",
		},
		{
			name:  "user:pass",
			input: []byte("user:pass"),
			want:  "dXNlcjpwYXNz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := base64Encode(tt.input)
			if got != tt.want {
				t.Errorf("base64Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasicAuth(t *testing.T) {
	result := basicAuth("user", "pass")
	want := "dXNlcjpwYXNz"
	if result != want {
		t.Errorf("basicAuth() = %v, want %v", result, want)
	}
}

// connectedSession connects ws and returns the session it established.
func connectedSession(t *testing.T, ws *WebSocket) *wsSession {
	t.Helper()
	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	ws.connMu.Lock()
	defer ws.connMu.Unlock()
	return ws.session
}

func TestWebSocket_handleDisconnect(t *testing.T) {
	svr := newWsEchoServer(t)
	defer svr.Close()

	ws := NewWebSocket(svrWSURL(svr), WithReconnect(false))
	defer ws.Close()
	session := connectedSession(t, ws)

	// Create a pending request that should be canceled
	respChan := make(chan *rpcResponse, 1)
	ws.pendingMu.Lock()
	ws.pending[1] = respChan
	ws.pendingMu.Unlock()

	// Record state changes
	var mu sync.Mutex
	states := make([]ConnectionState, 0)
	ws.OnStateChange(func(state ConnectionState) {
		mu.Lock()
		states = append(states, state)
		mu.Unlock()
	})

	// Simulate disconnect
	ws.handleDisconnect(session)

	// Verify pending request was canceled
	ws.pendingMu.Lock()
	if len(ws.pending) != 0 {
		t.Errorf("pending requests not cleared: %d remaining", len(ws.pending))
	}
	ws.pendingMu.Unlock()

	// Verify state changed to disconnected
	mu.Lock()
	defer mu.Unlock()
	if len(states) == 0 || states[len(states)-1] != StateDisconnected {
		t.Error("state should be disconnected")
	}
}

func TestWebSocket_handleDisconnectWithReconnect(t *testing.T) {
	svr := newWsEchoServer(t)
	defer svr.Close()

	ws := NewWebSocket(svrWSURL(svr), WithReconnect(true), WithRetry(1, 10*time.Millisecond))
	session := connectedSession(t, ws)

	var mu sync.Mutex
	states := make([]ConnectionState, 0)
	ws.OnStateChange(func(state ConnectionState) {
		mu.Lock()
		states = append(states, state)
		mu.Unlock()
	})

	// Simulate disconnect - it will attempt to reconnect
	ws.handleDisconnect(session)

	// Give reconnect time to attempt
	time.Sleep(100 * time.Millisecond)

	// Close to stop reconnect attempts
	ws.Close()
}

func TestWebSocket_reconnect(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc", WithRetry(1, 10*time.Millisecond),
		WithTimeout(200*time.Millisecond))

	var mu sync.Mutex
	states := make([]ConnectionState, 0)
	ws.OnStateChange(func(state ConnectionState) {
		mu.Lock()
		states = append(states, state)
		mu.Unlock()
	})

	// Attempt reconnect - it will fail to connect
	ws.reconnect()

	// Should have attempted reconnection and ended up disconnected
	mu.Lock()
	stateCount := len(states)
	mu.Unlock()
	if stateCount == 0 {
		t.Error("state changes should have occurred")
	}
}

func TestWebSocket_reconnectWhenClosed(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc", WithRetry(3, 10*time.Millisecond))

	// Close the websocket first
	ws.Close()

	// Reconnect should return early
	ws.reconnect()

	// No panics should occur - reconnect should return early when closed
}

func TestWebSocket_handleMessageError(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	// Create a pending request
	respChan := make(chan *rpcResponse, 1)
	ws.pendingMu.Lock()
	ws.pending[1] = respChan
	ws.pendingMu.Unlock()

	// Simulate receiving an error response
	message := []byte(`{"id":1,"error":{"code":-1,"message":"method not found"}}`)
	ws.handleMessage(message)

	// Check that response was received
	select {
	case resp := <-respChan:
		if resp == nil {
			t.Fatal("response is nil")
		}
		if resp.Error == nil {
			t.Fatal("expected error in response")
		}
		if resp.Error.Code != -1 {
			t.Errorf("error code = %d, want -1", resp.Error.Code)
		}
	default:
		t.Error("response not received")
	}
}

func TestWebSocket_isClosed(t *testing.T) {
	ws := NewWebSocket("ws://192.168.1.100/rpc")

	// Initially not closed
	if ws.isClosed() {
		t.Error("isClosed() = true, want false initially")
	}

	// After closing
	ws.Close()
	if !ws.isClosed() {
		t.Error("isClosed() = false after close, want true")
	}
}

func TestWebSocket_pingLoopStops(t *testing.T) {
	svr := newWsEchoServer(t)
	defer svr.Close()

	ws := NewWebSocket(svrWSURL(svr), WithPingInterval(time.Hour), WithReconnect(false))
	session := connectedSession(t, ws)

	done := make(chan struct{})
	go func() {
		defer close(done)
		ws.pingLoop(session)
	}()

	// Close should stop the ping loop
	ws.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("pingLoop kept running after Close")
	}
}

// A read loop left over from an earlier connection must not tear down the
// connection that replaced it.
func TestWebSocket_StaleSessionLeavesSuccessorAlone(t *testing.T) {
	svr := newWsEchoServer(t)
	defer svr.Close()

	ws := NewWebSocket(svrWSURL(svr), WithReconnect(false))
	defer ws.Close()

	first := connectedSession(t, ws)
	ws.handleDisconnect(first)
	second := connectedSession(t, ws)
	if second == first {
		t.Fatal("Connect() reused a torn-down session")
	}

	ws.handleDisconnect(first)

	if ws.State() != StateConnected {
		t.Errorf("state = %v after a stale disconnect, want connected", ws.State())
	}
	if _, err := ws.Call(context.Background(), newTestRPCRequest("Shelly.GetStatus", nil)); err != nil {
		t.Errorf("Call() on the successor session error = %v", err)
	}
}

// Close has to close the connection itself: that is what ends the read
// goroutine, which otherwise blocks in ReadMessage for good.
func TestWebSocket_CloseClosesConnection(t *testing.T) {
	serverSawClose := make(chan struct{})
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				close(serverSawClose)
				return
			}
		}
	}))
	defer svr.Close()

	ws := NewWebSocket(svrWSURL(svr), WithReconnect(false))
	connectedSession(t, ws)
	if err := ws.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	select {
	case <-serverSawClose:
	case <-time.After(5 * time.Second):
		t.Fatal("server never saw the connection close")
	}
}
