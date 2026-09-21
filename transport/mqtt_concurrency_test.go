package transport

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// droppableClient counts Disconnect calls and answers every published request
// through the transport's own response handler.
type droppableClient struct {
	transport   *MQTT
	disconnects atomic.Int32
	mockClient
}

func (c *droppableClient) Disconnect(uint) { c.disconnects.Add(1) }

func (c *droppableClient) Publish(_ string, _ byte, _ bool, payload interface{}) mqtt.Token {
	var req struct {
		ID int64 `json:"id"`
	}
	if data, ok := payload.([]byte); ok && json.Unmarshal(data, &req) == nil && c.transport != nil {
		resp, _ := json.Marshal(map[string]any{"id": req.ID, "result": map[string]any{"ok": true}})
		go c.transport.handleResponse(c, &mockMessage{payload: resp})
	}
	return &mockToken{}
}

// A client that lost its connection keeps redialing by itself, so Connect has
// to shut it down before replacing it, and must not keep a client whose dial
// was abandoned.
func TestMQTT_Connect_DropsReplacedAndAbandonedClients(t *testing.T) {
	m := NewMQTT("tcp://192.0.2.1:1883", "device-abc", WithReconnect(true))
	old := &droppableClient{}
	m.client = old

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Connect(ctx); err == nil {
		t.Fatal("Connect() with a canceled context error = nil")
	}

	if got := old.disconnects.Load(); got != 1 {
		t.Errorf("replaced client disconnected %d times, want 1", got)
	}
	m.connMu.Lock()
	kept := m.client
	m.connMu.Unlock()
	if kept != nil {
		t.Error("Connect() kept the client whose dial was abandoned")
	}
	if got := m.State(); got != StateDisconnected {
		t.Errorf("state = %v, want disconnected", got)
	}
}

func TestMQTT_EventsFromDroppedClientIgnored(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "device-abc", WithReconnect(false))
	current := &droppableClient{mockClient: mockClient{connected: true}}
	m.client = current
	m.onConnect(current)

	respChan := make(chan *rpcResponse, 1)
	m.pendingMu.Lock()
	m.pending[7] = respChan
	m.pendingMu.Unlock()

	stale := &droppableClient{}
	m.onConnectionLost(stale, nil)
	m.onConnect(stale)

	if got := m.State(); got != StateConnected {
		t.Errorf("state = %v after a dropped client's events, want connected", got)
	}
	m.pendingMu.Lock()
	_, stillPending := m.pending[7]
	m.pendingMu.Unlock()
	if !stillPending {
		t.Error("a dropped client's connection loss canceled a request on the current client")
	}
}

// The client delivers notifications and RPC responses on one goroutine, so a
// handler that makes a Call only finishes if it runs somewhere else.
func TestMQTT_NotificationHandlerMayCall(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "device-abc")
	client := &droppableClient{mockClient: mockClient{connected: true}, transport: m}
	m.client = client

	result := make(chan error, 1)
	if err := m.Subscribe(func(json.RawMessage) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := m.Call(ctx, newTestRPCRequest("Shelly.GetStatus", nil))
		result <- err
	}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	delivered := make(chan struct{})
	go func() {
		defer close(delivered)
		m.handleNotification(client, &mockMessage{payload: []byte(`{"method":"NotifyStatus"}`)})
	}()
	select {
	case <-delivered:
	case <-time.After(5 * time.Second):
		t.Fatal("handleNotification blocked the message goroutine")
	}

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("Call() from a notification handler error = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("notification handler never finished")
	}
}

// The handler appends to an unlocked slice, so the race detector fails this
// test if two notifications are ever delivered at once.
func TestMQTT_NotificationsSerializedAndOrdered(t *testing.T) {
	const total = 100
	m := NewMQTT("tcp://192.168.1.10:1883", "device-abc")

	var seen []string
	done := make(chan struct{})
	if err := m.Subscribe(func(raw json.RawMessage) {
		seen = append(seen, string(raw))
		if len(seen) == total {
			close(done)
		}
	}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	want := make([]string, total)
	for i := range total {
		payload, _ := json.Marshal(map[string]int{"seq": i})
		want[i] = string(payload)
		m.handleNotification(nil, &mockMessage{payload: payload})
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("not every notification was delivered")
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("position %d = %s, want %s", i, seen[i], want[i])
		}
	}
}

// A state callback that calls back into the transport must not deadlock on
// the lock Connect holds.
func TestMQTT_StateCallbackMayCallConnect(t *testing.T) {
	m := NewMQTT("tcp://192.0.2.1:1883", "device-abc")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Once only: each nested Connect reports its own state changes.
	var nested atomic.Bool
	m.OnStateChange(func(ConnectionState) {
		if !nested.CompareAndSwap(false, true) {
			return
		}
		if err := m.Connect(ctx); err == nil {
			t.Error("nested Connect() with a canceled context error = nil")
		}
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := m.Connect(ctx); err == nil {
			t.Error("Connect() with a canceled context error = nil")
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Connect() deadlocked on its own state callback")
	}
}

// orderClient records what the transport's state was when it subscribed.
type orderClient struct {
	transport        *MQTT
	stateAtSubscribe []ConnectionState
	mockClient
}

func (c *orderClient) Subscribe(string, byte, mqtt.MessageHandler) mqtt.Token {
	c.stateAtSubscribe = append(c.stateAtSubscribe, c.transport.State())
	return &mockToken{}
}

// A listener that makes a Call on StateConnected only gets its reply if the
// response topic was subscribed before the state was reported.
func TestMQTT_ConnectedReportedAfterResponseSubscription(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "device-abc")
	client := &orderClient{mockClient: mockClient{connected: true}, transport: m}
	ready := make(chan error, 1)
	m.client = client
	m.ready = ready

	m.onConnect(client)

	if len(client.stateAtSubscribe) == 0 {
		t.Fatal("onConnect did not subscribe")
	}
	for _, state := range client.stateAtSubscribe {
		if state == StateConnected {
			t.Error("Connected was reported before the subscription was made")
		}
	}
	if got := m.State(); got != StateConnected {
		t.Errorf("state = %v, want connected", got)
	}
	select {
	case err := <-ready:
		if err != nil {
			t.Errorf("ready error = %v, want nil", err)
		}
	default:
		t.Error("onConnect did not signal that the transport is ready")
	}
}

// Close must not wait for a dial that is still in progress.
func TestMQTT_CloseDoesNotWaitForDial(t *testing.T) {
	m := NewMQTT("tcp://192.0.2.1:1883", "device-abc", WithTimeout(30*time.Second))

	connectDone := make(chan error, 1)
	go func() { connectDone <- m.Connect(context.Background()) }()

	// Wait until the dial holds a client, so Close has something to take.
	deadline := time.After(5 * time.Second)
	for m.currentClient() == nil {
		select {
		case <-deadline:
			t.Fatal("Connect never created a client")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	closed := make(chan struct{})
	go func() {
		defer close(closed)
		m.Close()
	}()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close waited for the dial")
	}
	select {
	case err := <-connectDone:
		if err == nil {
			t.Error("Connect() error = nil after Close")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Connect did not return after Close")
	}
	if got := m.State(); got != StateClosed {
		t.Errorf("state = %v, want closed", got)
	}
}
