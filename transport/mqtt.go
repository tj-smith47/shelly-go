package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTT is an MQTT transport for Shelly devices.
// Supports pub/sub messaging for Gen2+ devices.
//
// The MQTT transport provides:
//   - RPC over MQTT topics
//   - Real-time notifications
//   - Automatic reconnection
//   - Request/response correlation
//   - Last will and testament support
type MQTT struct {
	client         mqtt.Client
	opts           *options
	notifyHandler  NotificationHandler
	pending        map[int64]chan *rpcResponse
	broker         string
	deviceID       string
	src            string
	stateCallbacks []func(ConnectionState)
	requestID      atomic.Int64
	state          ConnectionState
	mu             sync.RWMutex
	notifyMu       sync.RWMutex
	stateMu        sync.RWMutex
	connMu         sync.Mutex
	pendingMu      sync.Mutex
	closed         bool
}

// NewMQTT creates a new MQTT transport.
//
// The broker should be the MQTT broker URL (e.g., "tcp://192.168.1.10:1883").
// The deviceID is the Shelly device ID (e.g., "shellyplus1pm-abc123").
//
// Example:
//
//	m := NewMQTT("tcp://192.168.1.10:1883", "shellyplus1pm-abc123",
//	    WithMQTTClientID("shelly-go-client"),
//	    WithMQTTQoS(1))
func NewMQTT(broker, deviceID string, opts ...Option) *MQTT {
	options := defaultOptions()
	applyOptions(options, opts)

	// Generate client ID if not provided
	if options.mqttClientID == "" {
		options.mqttClientID = fmt.Sprintf("shelly-go-%d", time.Now().UnixNano())
	}

	return &MQTT{
		broker:         broker,
		deviceID:       deviceID,
		src:            options.mqttClientID,
		opts:           options,
		state:          StateDisconnected,
		pending:        make(map[int64]chan *rpcResponse),
		stateCallbacks: make([]func(ConnectionState), 0),
	}
}

// Connect establishes the MQTT connection.
// This must be called before making any RPC calls.
func (m *MQTT) Connect(ctx context.Context) error {
	m.connMu.Lock()
	defer m.connMu.Unlock()

	if m.isClosed() {
		return fmt.Errorf("MQTT transport is closed")
	}

	if m.client != nil && m.client.IsConnected() {
		return nil // already connected
	}

	m.setState(StateConnecting)

	// Create MQTT client options
	mqttOpts := mqtt.NewClientOptions().
		AddBroker(m.broker).
		SetClientID(m.opts.mqttClientID).
		SetAutoReconnect(m.opts.reconnect).
		SetConnectTimeout(m.opts.timeout).
		SetOnConnectHandler(m.onConnect).
		SetConnectionLostHandler(m.onConnectionLost)

	// Add authentication if provided
	if m.opts.username != "" {
		mqttOpts.SetUsername(m.opts.username)
		mqttOpts.SetPassword(m.opts.password)
	}

	// Add TLS config if provided
	if m.opts.tlsConfig != nil {
		mqttOpts.SetTLSConfig(m.opts.tlsConfig)
	}

	// Create and connect client
	m.client = mqtt.NewClient(mqttOpts)
	token := m.client.Connect()

	// Wait for connection with context timeout
	done := make(chan struct{})
	go func() {
		token.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		m.setState(StateDisconnected)
		return ctx.Err()
	case <-done:
		if token.Error() != nil {
			m.setState(StateDisconnected)
			return fmt.Errorf("mqtt connect: %w", token.Error())
		}
	}

	return nil
}

// onConnect is called when MQTT connection is established.
func (m *MQTT) onConnect(client mqtt.Client) {
	m.setState(StateConnected)

	// Subscribe to response topic
	responseTopic := m.src + "/rpc"
	token := client.Subscribe(responseTopic, m.opts.mqttQoS, m.handleResponse)
	token.Wait()
	if token.Error() != nil {
		return
	}

	// Subscribe to events topic if notification handler is set
	m.notifyMu.RLock()
	hasHandler := m.notifyHandler != nil
	m.notifyMu.RUnlock()

	if hasHandler {
		eventsTopic := m.deviceID + "/events/rpc"
		token = client.Subscribe(eventsTopic, m.opts.mqttQoS, m.handleNotification)
		token.Wait()
	}
}

// onConnectionLost is called when MQTT connection is lost.
func (m *MQTT) onConnectionLost(client mqtt.Client, err error) {
	if m.isClosed() {
		return
	}

	// Cancel all pending requests
	m.pendingMu.Lock()
	for id, ch := range m.pending {
		close(ch)
		delete(m.pending, id)
	}
	m.pendingMu.Unlock()

	if m.opts.reconnect {
		m.setState(StateReconnecting)
	} else {
		m.setState(StateDisconnected)
	}
}

// handleResponse handles incoming MQTT response messages.
func (m *MQTT) handleResponse(client mqtt.Client, msg mqtt.Message) {
	var resp rpcResponse
	if err := json.Unmarshal(msg.Payload(), &resp); err != nil {
		return
	}

	if resp.ID == 0 {
		return
	}

	m.pendingMu.Lock()
	if ch, ok := m.pending[resp.ID]; ok {
		select {
		case ch <- &resp:
		default:
		}
	}
	m.pendingMu.Unlock()
}

// handleNotification handles incoming MQTT notification messages.
func (m *MQTT) handleNotification(client mqtt.Client, msg mqtt.Message) {
	m.notifyMu.RLock()
	handler := m.notifyHandler
	m.notifyMu.RUnlock()

	if handler != nil {
		handler(msg.Payload())
	}
}

// Call executes an RPC method call via MQTT.
//
// If not connected, this will attempt to connect first.
// The request is published to the device's RPC topic.
// The response is received on the client's response topic.
func (m *MQTT) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if m.isClosed() {
		return nil, fmt.Errorf("MQTT transport is closed")
	}

	// Auto-connect if not connected
	if m.client == nil || !m.client.IsConnected() {
		if err := m.Connect(ctx); err != nil {
			return nil, err
		}
	}

	// Create request with unique ID
	id := m.requestID.Add(1)
	req := rpcRequest{
		ID:     id,
		Src:    m.src,
		Method: method,
		Params: params,
	}

	// Create response channel
	respChan := make(chan *rpcResponse, 1)
	m.pendingMu.Lock()
	m.pending[id] = respChan
	m.pendingMu.Unlock()

	defer func() {
		m.pendingMu.Lock()
		delete(m.pending, id)
		m.pendingMu.Unlock()
	}()

	// Marshal request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Publish to device RPC topic
	rpcTopic := m.deviceID + "/rpc"
	token := m.client.Publish(rpcTopic, m.opts.mqttQoS, false, data)

	// Wait for publish with context timeout
	done := make(chan struct{})
	go func() {
		token.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		if token.Error() != nil {
			return nil, fmt.Errorf("publish: %w", token.Error())
		}
	}

	// Wait for response
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respChan:
		if resp == nil {
			return nil, fmt.Errorf("connection lost while waiting for response")
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

// Subscribe registers a handler for incoming MQTT notifications.
// This subscribes to the device's events topic.
func (m *MQTT) Subscribe(handler NotificationHandler) error {
	m.notifyMu.Lock()
	defer m.notifyMu.Unlock()

	if m.notifyHandler != nil {
		return errHandlerAlreadyRegistered
	}

	m.notifyHandler = handler

	// Subscribe to events topic if connected
	m.connMu.Lock()
	client := m.client
	m.connMu.Unlock()

	if client != nil && client.IsConnected() {
		eventsTopic := m.deviceID + "/events/rpc"
		token := client.Subscribe(eventsTopic, m.opts.mqttQoS, m.handleNotification)
		token.Wait()
		if token.Error() != nil {
			m.notifyHandler = nil
			return fmt.Errorf("subscribe to events: %w", token.Error())
		}
	}

	return nil
}

// Unsubscribe removes the notification handler.
// This unsubscribes from the device's events topic.
func (m *MQTT) Unsubscribe() error {
	m.notifyMu.Lock()
	m.notifyHandler = nil
	m.notifyMu.Unlock()

	// Unsubscribe from events topic if connected
	m.connMu.Lock()
	client := m.client
	m.connMu.Unlock()

	if client != nil && client.IsConnected() {
		eventsTopic := m.deviceID + "/events/rpc"
		token := client.Unsubscribe(eventsTopic)
		token.Wait()
		if token.Error() != nil {
			return fmt.Errorf("unsubscribe from events: %w", token.Error())
		}
	}

	return nil
}

// State returns the current connection state.
func (m *MQTT) State() ConnectionState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// OnStateChange registers a callback for connection state changes.
func (m *MQTT) OnStateChange(callback func(ConnectionState)) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	m.stateCallbacks = append(m.stateCallbacks, callback)
}

// setState updates the connection state and notifies callbacks.
func (m *MQTT) setState(state ConnectionState) {
	m.mu.Lock()
	m.state = state
	m.mu.Unlock()

	m.stateMu.RLock()
	callbacks := make([]func(ConnectionState), len(m.stateCallbacks))
	copy(callbacks, m.stateCallbacks)
	m.stateMu.RUnlock()

	for _, cb := range callbacks {
		cb(state)
	}
}

// isClosed returns true if the transport is closed.
func (m *MQTT) isClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

// Close closes the MQTT connection.
func (m *MQTT) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	m.mu.Unlock()

	// Cancel all pending requests
	m.pendingMu.Lock()
	for id, ch := range m.pending {
		close(ch)
		delete(m.pending, id)
	}
	m.pendingMu.Unlock()

	// Disconnect from broker
	m.connMu.Lock()
	if m.client != nil {
		m.client.Disconnect(250) // 250ms quiesce period
		m.client = nil
	}
	m.connMu.Unlock()

	m.setState(StateClosed)
	return nil
}

// DeviceID returns the device ID this transport is connected to.
func (m *MQTT) DeviceID() string {
	return m.deviceID
}

// IsConnected returns true if the transport is connected.
func (m *MQTT) IsConnected() bool {
	m.connMu.Lock()
	defer m.connMu.Unlock()
	return m.client != nil && m.client.IsConnected()
}
