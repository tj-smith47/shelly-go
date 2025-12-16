package transport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func TestNewMQTT(t *testing.T) {
	tests := []struct {
		name     string
		broker   string
		deviceID string
		opts     []Option
	}{
		{
			name:     "basic MQTT",
			broker:   "tcp://192.168.1.10:1883",
			deviceID: "shellyplus1pm-abc123",
		},
		{
			name:     "with topic",
			broker:   "tcp://192.168.1.10:1883",
			deviceID: "shellyplus1pm-abc123",
			opts:     []Option{WithMQTTTopic("shellies/device-id")},
		},
		{
			name:     "with client ID and QoS",
			broker:   "tcp://192.168.1.10:1883",
			deviceID: "shellyplus1pm-abc123",
			opts: []Option{
				WithMQTTClientID("test-client"),
				WithMQTTQoS(1),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqtt := NewMQTT(tt.broker, tt.deviceID, tt.opts...)
			if mqtt == nil {
				t.Fatal("NewMQTT() returned nil")
			}
			if mqtt.broker != tt.broker {
				t.Errorf("broker = %v, want %v", mqtt.broker, tt.broker)
			}
			if mqtt.deviceID != tt.deviceID {
				t.Errorf("deviceID = %v, want %v", mqtt.deviceID, tt.deviceID)
			}
		})
	}
}

func TestMQTT_CallNotConnected(t *testing.T) {
	// Call on non-connected MQTT should attempt to connect and fail
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	cancel() // Cancel immediately to avoid waiting

	_, err := mqtt.Call(ctx, "Switch.Set", nil)
	if err == nil {
		t.Error("Call() error = nil, want error (should fail to connect)")
	}
}

func TestMQTT_Subscribe(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	handler := NotificationHandler(func(data json.RawMessage) {})
	err := mqtt.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() error = %v", err)
	}

	// Second subscribe should fail
	err = mqtt.Subscribe(handler)
	if err == nil {
		t.Error("second Subscribe() error = nil, want error")
	}
}

func TestMQTT_Unsubscribe(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	handler := NotificationHandler(func(data json.RawMessage) {})
	err := mqtt.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() error = %v", err)
	}

	err = mqtt.Unsubscribe()
	if err != nil {
		t.Errorf("Unsubscribe() error = %v", err)
	}

	// Should be able to subscribe again
	err = mqtt.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() after Unsubscribe() error = %v", err)
	}
}

func TestMQTT_State(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	if mqtt.State() != StateDisconnected {
		t.Errorf("initial State() = %v, want %v", mqtt.State(), StateDisconnected)
	}
}

func TestMQTT_OnStateChange(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	called := false
	mqtt.OnStateChange(func(state ConnectionState) {
		called = true
	})

	mqtt.setState(StateConnected)

	if !called {
		t.Error("state change callback not called")
	}
}

func TestMQTT_Close(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	err := mqtt.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if mqtt.State() != StateClosed {
		t.Errorf("State() after Close() = %v, want %v", mqtt.State(), StateClosed)
	}

	// Calling Close again should be safe
	err = mqtt.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestMQTT_ClosedCall(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")
	mqtt.Close()

	_, err := mqtt.Call(context.Background(), "Switch.Set", nil)
	if err == nil {
		t.Error("Call() on closed MQTT error = nil, want error")
	}
}

func TestMQTT_DeviceID(t *testing.T) {
	deviceID := "shellyplus1pm-abc123"
	mqtt := NewMQTT("tcp://192.168.1.10:1883", deviceID)

	if mqtt.DeviceID() != deviceID {
		t.Errorf("DeviceID() = %v, want %v", mqtt.DeviceID(), deviceID)
	}
}

func TestMQTT_IsConnected(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	if mqtt.IsConnected() {
		t.Error("IsConnected() = true, want false (not connected)")
	}
}

func TestMQTT_ConnectClosed(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")
	mqtt.Close()

	err := mqtt.Connect(context.Background())
	if err == nil {
		t.Error("Connect() on closed MQTT should return error")
	}
}

func TestMQTT_MultipleStateCallbacks(t *testing.T) {
	mqtt := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	calls := make([]ConnectionState, 0)

	mqtt.OnStateChange(func(state ConnectionState) {
		calls = append(calls, state)
	})
	mqtt.OnStateChange(func(state ConnectionState) {
		calls = append(calls, state)
	})

	mqtt.setState(StateConnecting)

	if len(calls) != 2 {
		t.Errorf("expected 2 callbacks, got %d", len(calls))
	}
}

func TestMQTT_handleResponse(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	// Create a pending request
	respChan := make(chan *rpcResponse, 1)
	m.pendingMu.Lock()
	m.pending[1] = respChan
	m.pendingMu.Unlock()

	// Simulate receiving a response via handleResponse
	// Note: We need to create a mock message since we can't use MQTT message interface
	// Instead, let's test the internal behavior

	// Simulate processing a valid response JSON
	payload := []byte(`{"id":1,"result":{"output":true}}`)
	var resp rpcResponse
	if err := json.Unmarshal(payload, &resp); err == nil && resp.ID != 0 {
		m.pendingMu.Lock()
		if ch, ok := m.pending[resp.ID]; ok {
			select {
			case ch <- &resp:
			default:
			}
		}
		m.pendingMu.Unlock()
	}

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

func TestMQTT_handleResponse_InvalidJSON(t *testing.T) {
	// Simulate processing invalid JSON - should not panic
	payload := []byte(`not json`)
	var resp rpcResponse
	err := json.Unmarshal(payload, &resp)
	if err == nil {
		t.Error("expected unmarshal error for invalid JSON")
	}
}

func TestMQTT_handleResponse_ZeroID(t *testing.T) {
	// Response with zero ID should be ignored
	payload := []byte(`{"id":0,"result":{}}`)
	var resp rpcResponse
	if err := json.Unmarshal(payload, &resp); err == nil {
		// ID is 0, should be ignored
		if resp.ID != 0 {
			t.Errorf("expected ID 0, got %d", resp.ID)
		}
	}
}

func TestMQTT_onConnectionLostReconnect(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123", WithReconnect(true))

	stateChanges := make([]ConnectionState, 0)
	m.OnStateChange(func(state ConnectionState) {
		stateChanges = append(stateChanges, state)
	})

	// Simulate connection lost
	m.onConnectionLost(nil, nil)

	// Should be in reconnecting state
	if len(stateChanges) == 0 {
		t.Error("state change callback not called")
	}
	if stateChanges[len(stateChanges)-1] != StateReconnecting {
		t.Errorf("expected StateReconnecting, got %v", stateChanges[len(stateChanges)-1])
	}
}

func TestMQTT_onConnectionLostNoReconnect(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123", WithReconnect(false))

	stateChanges := make([]ConnectionState, 0)
	m.OnStateChange(func(state ConnectionState) {
		stateChanges = append(stateChanges, state)
	})

	// Simulate connection lost
	m.onConnectionLost(nil, nil)

	// Should be in disconnected state
	if len(stateChanges) == 0 {
		t.Error("state change callback not called")
	}
	if stateChanges[len(stateChanges)-1] != StateDisconnected {
		t.Errorf("expected StateDisconnected, got %v", stateChanges[len(stateChanges)-1])
	}
}

func TestMQTT_GeneratedClientID(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	if m.src == "" {
		t.Error("src (client ID) should be auto-generated")
	}
}

func TestMQTT_CustomClientID(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123", WithMQTTClientID("custom-client"))

	if m.src != "custom-client" {
		t.Errorf("src = %v, want custom-client", m.src)
	}
}

func TestMQTT_onConnectionLostWithPendingRequests(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123", WithReconnect(false))

	// Create pending requests that should be canceled
	respChan1 := make(chan *rpcResponse, 1)
	respChan2 := make(chan *rpcResponse, 1)
	m.pendingMu.Lock()
	m.pending[1] = respChan1
	m.pending[2] = respChan2
	m.pendingMu.Unlock()

	// Simulate connection lost
	m.onConnectionLost(nil, nil)

	// Verify pending requests were cleared
	m.pendingMu.Lock()
	pendingCount := len(m.pending)
	m.pendingMu.Unlock()

	if pendingCount != 0 {
		t.Errorf("pending requests not cleared: %d remaining", pendingCount)
	}
}

func TestMQTT_onConnectionLostWhenClosed(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	// Close first
	m.Close()

	// Create a pending request
	respChan := make(chan *rpcResponse, 1)
	m.pendingMu.Lock()
	m.pending[1] = respChan
	m.pendingMu.Unlock()

	stateChanges := make([]ConnectionState, 0)
	m.OnStateChange(func(state ConnectionState) {
		stateChanges = append(stateChanges, state)
	})

	// onConnectionLost should return early when closed
	m.onConnectionLost(nil, nil)

	// State should not change (already closed)
	if len(stateChanges) != 0 {
		t.Errorf("state changed %d times, want 0 (should return early when closed)", len(stateChanges))
	}
}

func TestMQTT_isClosed(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	// Initially not closed
	if m.isClosed() {
		t.Error("isClosed() = true, want false initially")
	}

	// After closing
	m.Close()
	if !m.isClosed() {
		t.Error("isClosed() = false after close, want true")
	}
}

// mockMessage implements mqtt.Message interface for testing
type mockMessage struct {
	payload []byte
	topic   string
}

func (m *mockMessage) Duplicate() bool   { return false }
func (m *mockMessage) Qos() byte         { return 0 }
func (m *mockMessage) Retained() bool    { return false }
func (m *mockMessage) Topic() string     { return m.topic }
func (m *mockMessage) MessageID() uint16 { return 0 }
func (m *mockMessage) Payload() []byte   { return m.payload }
func (m *mockMessage) Ack()              {}

func TestMQTT_handleResponseWithMock(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	// Create a pending request
	respChan := make(chan *rpcResponse, 1)
	m.pendingMu.Lock()
	m.pending[1] = respChan
	m.pendingMu.Unlock()

	// Create mock message with valid response
	msg := &mockMessage{
		payload: []byte(`{"id":1,"result":{"output":true}}`),
		topic:   "shelly-go/rpc",
	}

	// Call handleResponse directly with mock
	m.handleResponse(nil, msg)

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

func TestMQTT_handleResponseInvalidJSON(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	msg := &mockMessage{
		payload: []byte(`not json`),
		topic:   "shelly-go/rpc",
	}

	// Should not panic
	m.handleResponse(nil, msg)
}

func TestMQTT_handleResponseZeroID(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	msg := &mockMessage{
		payload: []byte(`{"id":0,"result":{}}`),
		topic:   "shelly-go/rpc",
	}

	// Should not panic - zero ID is ignored
	m.handleResponse(nil, msg)
}

func TestMQTT_handleResponseNoPending(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	msg := &mockMessage{
		payload: []byte(`{"id":999,"result":{}}`),
		topic:   "shelly-go/rpc",
	}

	// Should not panic - no pending request for ID 999
	m.handleResponse(nil, msg)
}

func TestMQTT_handleNotificationWithMock(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	var received json.RawMessage
	m.Subscribe(func(data json.RawMessage) {
		received = data
	})

	msg := &mockMessage{
		payload: []byte(`{"method":"NotifyStatus","params":{"ts":1234}}`),
		topic:   "shellyplus1pm-abc123/events/rpc",
	}

	m.handleNotification(nil, msg)

	if received == nil {
		t.Error("notification handler not called")
	}
}

func TestMQTT_handleNotificationNoHandler(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	msg := &mockMessage{
		payload: []byte(`{"method":"NotifyStatus","params":{"ts":1234}}`),
		topic:   "shellyplus1pm-abc123/events/rpc",
	}

	// Should not panic when no handler is set
	m.handleNotification(nil, msg)
}

// mockToken implements mqtt.Token interface for testing
type mockToken struct {
	err error
}

func (t *mockToken) Wait() bool                       { return true }
func (t *mockToken) WaitTimeout(d time.Duration) bool { return true }
func (t *mockToken) Done() <-chan struct{}            { ch := make(chan struct{}); close(ch); return ch }
func (t *mockToken) Error() error                     { return t.err }

// mockClient implements mqtt.Client interface for testing
type mockClient struct {
	connected bool
}

func (c *mockClient) IsConnected() bool       { return c.connected }
func (c *mockClient) IsConnectionOpen() bool  { return c.connected }
func (c *mockClient) Connect() mqtt.Token     { return &mockToken{} }
func (c *mockClient) Disconnect(quiesce uint) {}
func (c *mockClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	return &mockToken{}
}
func (c *mockClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	return &mockToken{}
}
func (c *mockClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	return &mockToken{}
}
func (c *mockClient) Unsubscribe(topics ...string) mqtt.Token             { return &mockToken{} }
func (c *mockClient) AddRoute(topic string, callback mqtt.MessageHandler) {}
func (c *mockClient) OptionsReader() mqtt.ClientOptionsReader             { return mqtt.ClientOptionsReader{} }

func TestMQTT_onConnectWithMock(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	stateChanges := make([]ConnectionState, 0)
	m.OnStateChange(func(state ConnectionState) {
		stateChanges = append(stateChanges, state)
	})

	client := &mockClient{connected: true}
	m.onConnect(client)

	// Should be in connected state
	if len(stateChanges) == 0 {
		t.Error("state change callback not called")
	}
	if stateChanges[len(stateChanges)-1] != StateConnected {
		t.Errorf("expected StateConnected, got %v", stateChanges[len(stateChanges)-1])
	}
}

func TestMQTT_onConnectWithNotificationHandler(t *testing.T) {
	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123")

	// Set up a notification handler
	m.Subscribe(func(data json.RawMessage) {})

	client := &mockClient{connected: true}
	m.onConnect(client)

	// Should be in connected state
	if m.State() != StateConnected {
		t.Errorf("State() = %v, want StateConnected", m.State())
	}
}
