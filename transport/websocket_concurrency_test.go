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

// newWsNotifyingServer answers every request and sends `notifications`
// notification frames as soon as the connection opens.
func newWsNotifyingServer(t *testing.T, notifications int) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for i := range notifications {
			frame, _ := json.Marshal(map[string]any{"method": "NotifyStatus", "params": map[string]any{"seq": i}})
			if err := conn.WriteMessage(websocket.TextMessage, frame); err != nil {
				return
			}
		}
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req map[string]any
			if json.Unmarshal(msg, &req) != nil {
				return
			}
			resp, _ := json.Marshal(map[string]any{"id": req["id"], "result": map[string]any{"ok": true}})
			if conn.WriteMessage(mt, resp) != nil {
				return
			}
		}
	}))
}

// A notification handler that makes an RPC call needs the read goroutine to
// stay free to receive the response.
func TestWebSocket_NotificationHandlerMayCall(t *testing.T) {
	svr := newWsNotifyingServer(t, 1)
	defer svr.Close()

	ws := NewWebSocket(svrWSURL(svr), WithReconnect(false))
	defer ws.Close()

	result := make(chan error, 1)
	if err := ws.Subscribe(func(json.RawMessage) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := ws.Call(ctx, newTestRPCRequest("Shelly.GetStatus", nil))
		result <- err
	}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("Call() from a notification handler error = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("notification handler never ran")
	}
}

// The handler appends to an unlocked slice, so the race detector fails this
// test if two notifications are ever delivered at once.
func TestWebSocket_NotificationsSerializedAndOrdered(t *testing.T) {
	const total = 50
	svr := newWsNotifyingServer(t, total)
	defer svr.Close()

	ws := NewWebSocket(svrWSURL(svr), WithReconnect(false))
	defer ws.Close()

	var seen []int
	done := make(chan struct{})
	if err := ws.Subscribe(func(raw json.RawMessage) {
		var frame struct {
			Params struct {
				Seq int `json:"seq"`
			} `json:"params"`
		}
		if err := json.Unmarshal(raw, &frame); err != nil {
			t.Errorf("notification did not parse: %v", err)
		}
		seen = append(seen, frame.Params.Seq)
		if len(seen) == total {
			close(done)
		}
	}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("not every notification was delivered")
	}
	for i, seq := range seen {
		if seq != i {
			t.Fatalf("notification %d arrived in position %d", seq, i)
		}
	}
}

// Close and a read failure both stop the same session; doing so at the same
// moment must neither panic nor leave the transport reconnecting.
func TestWebSocket_CloseRacesDisconnect(t *testing.T) {
	svr := newWsEchoServer(t)
	defer svr.Close()

	for range 20 {
		ws := NewWebSocket(svrWSURL(svr), WithReconnect(true), WithRetry(1, time.Millisecond))
		session := connectedSession(t, ws)

		var wg sync.WaitGroup
		wg.Go(func() { ws.handleDisconnect(session) })
		wg.Go(func() { ws.Close() })
		wg.Wait()

		if got := ws.State(); got != StateClosed {
			t.Fatalf("state = %v after Close, want closed", got)
		}
	}
}

// Close during the reconnect backoff ends the reconnect instead of letting it
// dial after the caller has let go of the transport.
func TestWebSocket_CloseStopsReconnectBackoff(t *testing.T) {
	svr := newWsEchoServer(t)
	defer svr.Close()

	ws := NewWebSocket(svrWSURL(svr), WithReconnect(true), WithRetry(3, time.Hour))
	session := connectedSession(t, ws)

	done := make(chan struct{})
	go func() {
		defer close(done)
		ws.handleDisconnect(session)
	}()
	<-done
	ws.Close()

	reconnectDone := make(chan struct{})
	go func() {
		defer close(reconnectDone)
		ws.reconnect()
	}()
	select {
	case <-reconnectDone:
	case <-time.After(5 * time.Second):
		t.Fatal("reconnect kept waiting after Close")
	}
	if got := ws.State(); got != StateClosed {
		t.Errorf("state = %v, want closed", got)
	}
}

// A reconnect that finds the connection already restored by a Call must
// report Connected, not stay on Reconnecting.
func TestWebSocket_ReconnectFindsRestoredConnection(t *testing.T) {
	svr := newWsEchoServer(t)
	defer svr.Close()

	ws := NewWebSocket(svrWSURL(svr), WithReconnect(false))
	defer ws.Close()
	connectedSession(t, ws)

	ws.setState(StateReconnecting)
	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if got := ws.State(); got != StateConnected {
		t.Errorf("state = %v, want connected", got)
	}
}

// newWsSilentServer accepts connections and reads requests but never answers.
// closeConns drops every open connection.
func newWsSilentServer(t *testing.T) (svr *httptest.Server, closeConns func()) {
	t.Helper()
	var mu sync.Mutex
	var conns []*websocket.Conn
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	svr = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		mu.Lock()
		conns = append(conns, conn)
		mu.Unlock()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	return svr, func() {
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range conns {
			conn.Close()
		}
	}
}

// callWhilePending starts a Call that the server will never answer and
// returns once the request is registered as pending.
func callWhilePending(t *testing.T, ws *WebSocket) <-chan error {
	t.Helper()
	result := make(chan error, 1)
	go func() {
		_, err := ws.Call(context.Background(), newTestRPCRequest("Shelly.GetStatus", nil))
		result <- err
	}()
	deadline := time.After(5 * time.Second)
	for {
		ws.pendingMu.Lock()
		pending := len(ws.pending)
		ws.pendingMu.Unlock()
		if pending > 0 {
			return result
		}
		select {
		case <-deadline:
			t.Fatal("Call never registered its request")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestWebSocket_CloseEndsWaitingCall(t *testing.T) {
	svr, _ := newWsSilentServer(t)
	defer svr.Close()

	ws := NewWebSocket(svrWSURL(svr), WithReconnect(false))
	result := callWhilePending(t, ws)
	ws.Close()

	select {
	case err := <-result:
		if err == nil {
			t.Error("Call() error = nil after Close")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Call kept waiting after Close")
	}
}

func TestWebSocket_DisconnectEndsWaitingCall(t *testing.T) {
	svr, closeConns := newWsSilentServer(t)
	defer svr.Close()

	ws := NewWebSocket(svrWSURL(svr), WithReconnect(false))
	defer ws.Close()
	result := callWhilePending(t, ws)
	closeConns()

	select {
	case err := <-result:
		if err == nil {
			t.Error("Call() error = nil after the connection dropped")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Call kept waiting after the connection dropped")
	}
}
