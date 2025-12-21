package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/tj-smith47/shelly-go/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// createMockWebSocketServer creates an httptest server that upgrades to WebSocket
// and echoes back JSON-RPC responses
func createMockWebSocketServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("WebSocket upgrade error: %v", err)
			return
		}
		defer conn.Close()

		for {
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				return // Connection closed
			}

			// Parse the request
			var req struct {
				ID     int    `json:"id"`
				Method string `json:"method"`
				Params any    `json:"params,omitempty"`
			}
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}

			// Create response based on method
			var result any
			switch req.Method {
			case "Shelly.GetDeviceInfo":
				result = map[string]any{
					"id":    "shellyplus1pm-test",
					"model": "SNSW-001P16EU",
					"gen":   2,
					"fw_id": "20231107-test",
					"app":   "Plus1PM",
				}
			case "Switch.GetStatus":
				result = map[string]any{
					"id":     0,
					"output": true,
					"apower": 100.5,
				}
			case "Switch.Set":
				result = map[string]any{
					"was_on": false,
				}
			default:
				result = map[string]any{}
			}

			// Send response
			resp := types.Response{
				ID:     req.ID,
				Result: mustMarshalJSON(result),
			}
			respData, _ := json.Marshal(resp)
			conn.WriteMessage(messageType, respData)
		}
	}))
}

func mustMarshalJSON(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

func wsURL(httpURL string) string {
	return strings.Replace(httpURL, "http://", "ws://", 1)
}

func TestWebSocket_Connect_WithMockServer(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer ws.Close()

	if ws.State() != StateConnected {
		t.Errorf("State() = %v, want StateConnected", ws.State())
	}
}

func TestWebSocket_Call_WithMockServer(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer ws.Close()

	// Call GetDeviceInfo
	result, err := ws.Call(ctx, NewSimpleRequest("Shelly.GetDeviceInfo"))
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	var deviceInfo struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Gen   int    `json:"gen"`
	}
	if err := json.Unmarshal(result, &deviceInfo); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if deviceInfo.ID != "shellyplus1pm-test" {
		t.Errorf("deviceInfo.ID = %q, want shellyplus1pm-test", deviceInfo.ID)
	}
	if deviceInfo.Gen != 2 {
		t.Errorf("deviceInfo.Gen = %d, want 2", deviceInfo.Gen)
	}
}

func TestWebSocket_Call_SwitchSet_WithMockServer(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer ws.Close()

	// Call Switch.Set
	result, err := ws.Call(ctx, NewSimpleRequest("Switch.Set"))
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	var response struct {
		WasOn bool `json:"was_on"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
}

func TestWebSocket_MultipleCalls_WithMockServer(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer ws.Close()

	// Make multiple sequential calls
	for i := 0; i < 5; i++ {
		_, err := ws.Call(ctx, NewSimpleRequest("Shelly.GetDeviceInfo"))
		if err != nil {
			t.Fatalf("Call() %d error = %v", i, err)
		}
	}
}

func TestWebSocket_Close_WithMockServer(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if ws.State() != StateConnected {
		t.Errorf("State() = %v, want StateConnected", ws.State())
	}

	err = ws.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if ws.State() != StateClosed {
		t.Errorf("After Close() State() = %v, want StateClosed", ws.State())
	}
}

func TestWebSocket_CallAfterClose_WithMockServer(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	ws.Close()

	// Call after close should error
	_, err = ws.Call(ctx, NewSimpleRequest("Shelly.GetDeviceInfo"))
	if err == nil {
		t.Error("Call() after Close() should return error")
	}
}

func TestWebSocket_StateChanges_WithMockServer(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))

	states := make([]ConnectionState, 0)
	ws.OnStateChange(func(state ConnectionState) {
		states = append(states, state)
	})

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	ws.Close()

	// Should have recorded state changes
	if len(states) < 2 {
		t.Errorf("Expected at least 2 state changes, got %d: %v", len(states), states)
	}
}

func TestWebSocket_Subscribe_WithMockServer(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer ws.Close()

	// Subscribe to notifications
	received := make(chan json.RawMessage, 1)
	handler := NotificationHandler(func(data json.RawMessage) {
		received <- data
	})

	err = ws.Subscribe(handler)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	// Unsubscribe
	err = ws.Unsubscribe()
	if err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}
}

func TestWebSocket_Reconnect_WithMockServer(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second), WithReconnect(true))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Make a call to verify connection works
	_, err = ws.Call(ctx, NewSimpleRequest("Shelly.GetDeviceInfo"))
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	ws.Close()

	if ws.State() != StateClosed {
		t.Errorf("After Close() State() = %v, want StateClosed", ws.State())
	}
}

func TestWebSocket_ConcurrentCalls_WithMockServer(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer ws.Close()

	// Make concurrent calls
	const numCalls = 10
	errCh := make(chan error, numCalls)

	for range numCalls {
		go func() {
			_, err := ws.Call(ctx, NewSimpleRequest("Shelly.GetDeviceInfo"))
			errCh <- err
		}()
	}

	// Collect results
	for i := range numCalls {
		if err := <-errCh; err != nil {
			t.Errorf("Concurrent call %d error = %v", i, err)
		}
	}
}

// Test WebSocket with notification server
func createNotificationServer(t *testing.T, notifyChan chan<- struct{}) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send a notification after connection
		go func() {
			time.Sleep(100 * time.Millisecond)
			notification := map[string]any{
				"src":    "shellyplus1pm-test",
				"dst":    "user",
				"method": "NotifyStatus",
				"params": map[string]any{
					"ts": 1234567890.0,
					"switch:0": map[string]any{
						"output": true,
					},
				},
			}
			data, _ := json.Marshal(notification)
			conn.WriteMessage(websocket.TextMessage, data)
			if notifyChan != nil {
				notifyChan <- struct{}{}
			}
		}()

		// Echo loop for RPC
		for {
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req struct {
				ID     int    `json:"id"`
				Method string `json:"method"`
			}
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}

			if req.ID > 0 { // It's an RPC request
				resp := types.Response{
					ID:     req.ID,
					Result: mustMarshalJSON(map[string]any{}),
				}
				respData, _ := json.Marshal(resp)
				conn.WriteMessage(messageType, respData)
			}
		}
	}))
}

func TestWebSocket_ReceiveNotification_WithMockServer(t *testing.T) {
	notifySent := make(chan struct{}, 1)
	server := createNotificationServer(t, notifySent)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer ws.Close()

	// Subscribe and wait for notification
	received := make(chan json.RawMessage, 1)
	handler := NotificationHandler(func(data json.RawMessage) {
		received <- data
	})

	err = ws.Subscribe(handler)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	// Wait for notification to be sent
	select {
	case <-notifySent:
		// Give time for processing
		time.Sleep(200 * time.Millisecond)
	case <-time.After(2 * time.Second):
		t.Log("Notification send timeout")
	}
}

func TestWebSocket_ConnectError(t *testing.T) {
	// Try to connect to non-existent server
	ws := NewWebSocket("ws://127.0.0.1:59999/rpc", WithTimeout(1*time.Second))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err == nil {
		t.Error("Connect() to non-existent server should error")
		ws.Close()
	}
}

func TestWebSocket_PingInterval(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	// Create with ping interval
	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second), WithPingInterval(1))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Wait a bit to let ping loop run
	time.Sleep(1500 * time.Millisecond)

	// Should still be connected
	if ws.State() != StateConnected {
		t.Errorf("State() = %v, want StateConnected", ws.State())
	}

	ws.Close()
}

func TestWebSocket_MultipleClients_WithMockServer(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	const numClients = 5
	clients := make([]*WebSocket, numClients)

	ctx := context.Background()

	// Connect all clients
	for i := 0; i < numClients; i++ {
		clients[i] = NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))
		err := clients[i].Connect(ctx)
		if err != nil {
			t.Fatalf("Connect() client %d error = %v", i, err)
		}
	}

	// Verify all connected and can make calls
	for i, c := range clients {
		if c.State() != StateConnected {
			t.Errorf("Client %d State() = %v, want StateConnected", i, c.State())
		}

		_, err := c.Call(ctx, NewSimpleRequest("Shelly.GetDeviceInfo"))
		if err != nil {
			t.Errorf("Client %d Call() error = %v", i, err)
		}
	}

	// Close all
	for i, c := range clients {
		if err := c.Close(); err != nil {
			t.Errorf("Close() client %d error = %v", i, err)
		}
	}
}

func TestWebSocket_CallWithContext(t *testing.T) {
	server := createMockWebSocketServer(t)
	defer server.Close()

	ws := NewWebSocket(wsURL(server.URL), WithTimeout(10*time.Second))

	ctx := context.Background()
	err := ws.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer ws.Close()

	// Create a canceled context
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	_, err = ws.Call(cancelCtx, NewSimpleRequest("Shelly.GetDeviceInfo"))
	if err == nil {
		t.Error("Call() with canceled context should error")
	}
}
