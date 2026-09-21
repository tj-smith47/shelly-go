package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/tj-smith47/shelly-go/internal/serial"
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
	client        mqtt.Client
	done          chan struct{}
	opts          *options
	notifyHandler NotificationHandler
	pending       map[int64]chan *rpcResponse
	ready         chan error
	broker        string
	deviceID      string
	src           string
	notifications serial.Queue[[]byte]
	connState
	requestID atomic.Int64
	mu        sync.RWMutex
	notifyMu  sync.RWMutex
	connMu    sync.Mutex
	clientMu  sync.Mutex
	pendingMu sync.Mutex
	closed    bool
}

var errMQTTClosed = errors.New("MQTT transport is closed")

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
		broker:   broker,
		deviceID: deviceID,
		src:      options.mqttClientID,
		opts:     options,
		pending:  make(map[int64]chan *rpcResponse),
		done:     make(chan struct{}),
	}
}

// Connect establishes the MQTT connection.
// This must be called before making any RPC calls.
func (m *MQTT) Connect(ctx context.Context) error {
	_, err := m.connect(ctx)
	return err
}

// connect returns the connected client, dialing the broker if there is none.
//
// connMu only keeps two dials from running at once. The client itself is
// guarded by clientMu, so Close and the client's own event handlers never wait
// behind a dial.
func (m *MQTT) connect(ctx context.Context) (mqtt.Client, error) {
	// State callbacks run after connMu is released, so one of them may call
	// back into the transport.
	defer m.flushState()

	m.connMu.Lock()
	defer m.connMu.Unlock()

	if m.isClosed() {
		return nil, errMQTTClosed
	}

	if client := m.currentClient(); client != nil && client.IsConnected() {
		return client, nil // already connected
	}

	// A client that lost its connection keeps redialing on its own; left
	// running next to its replacement it would answer on the same client ID.
	m.dropClient(nil)

	m.queueState(StateConnecting)

	// Create and connect client
	client := mqtt.NewClient(m.clientOptions())
	ready := make(chan error, 1)
	m.clientMu.Lock()
	m.client = client
	m.ready = ready
	m.clientMu.Unlock()
	token := client.Connect()

	err := m.awaitReady(ctx, token, ready)
	if err != nil {
		if m.dropClient(client) {
			m.queueState(StateDisconnected)
		}
		return nil, err
	}

	// Close may have run during the dial and taken the client with it.
	if m.currentClient() != client {
		return nil, errMQTTClosed
	}
	return client, nil
}

// clientOptions builds the broker client's options from the transport's.
func (m *MQTT) clientOptions() *mqtt.ClientOptions {
	mqttOpts := mqtt.NewClientOptions().
		AddBroker(m.broker).
		SetClientID(m.opts.mqttClientID).
		SetAutoReconnect(m.opts.reconnect).
		SetConnectTimeout(m.opts.timeout).
		SetOnConnectHandler(m.onConnect).
		SetConnectionLostHandler(m.onConnectionLost)

	if m.opts.username != "" {
		mqttOpts.SetUsername(m.opts.username)
		mqttOpts.SetPassword(m.opts.password)
	}
	if m.opts.tlsConfig != nil {
		mqttOpts.SetTLSConfig(m.opts.tlsConfig)
	}
	return mqttOpts
}

// awaitReady waits for the broker connection and then for onConnect to report
// the response subscription. ctx or Close ends either wait.
func (m *MQTT) awaitReady(ctx context.Context, token mqtt.Token, ready <-chan error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m.done:
		return errMQTTClosed
	case <-token.Done():
		if token.Error() != nil {
			return fmt.Errorf("mqtt connect: %w", token.Error())
		}
	}
	// The broker connection alone is not enough: a request published before
	// the response topic is subscribed never gets its reply.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m.done:
		return errMQTTClosed
	case err := <-ready:
		return err
	}
}

// currentClient returns the client in use, or nil.
func (m *MQTT) currentClient() mqtt.Client {
	m.clientMu.Lock()
	defer m.clientMu.Unlock()
	return m.client
}

// dropClient disconnects and forgets the current client. With a non-nil only
// it does so just when that is still the current client, and reports whether
// it was.
func (m *MQTT) dropClient(only mqtt.Client) bool {
	m.clientMu.Lock()
	client := m.client
	if client == nil || (only != nil && client != only) {
		m.clientMu.Unlock()
		return false
	}
	m.client = nil
	m.ready = nil
	m.clientMu.Unlock()

	client.Disconnect(0)
	return true
}

// isCurrent reports whether client is the one the transport is using. Events
// from a client that was dropped must not touch the state of its replacement.
func (m *MQTT) isCurrent(client mqtt.Client) bool {
	return m.currentClient() == client
}

// onConnect is called when MQTT connection is established, including each
// time the client reconnects by itself.
func (m *MQTT) onConnect(client mqtt.Client) {
	if !m.isCurrent(client) {
		return
	}

	err := m.subscribeTopics(client)

	m.clientMu.Lock()
	ready := m.ready
	m.clientMu.Unlock()
	if ready != nil {
		select {
		case ready <- err:
		default:
		}
	}

	// Connected is reported only once replies can arrive, so a listener that
	// makes a Call straight away gets its response.
	if err != nil {
		m.setState(StateDisconnected)
		return
	}
	m.setState(StateConnected)
}

// subscribeTopics subscribes to the response topic and, when a notification
// handler is registered, the device's events topic.
func (m *MQTT) subscribeTopics(client mqtt.Client) error {
	token := client.Subscribe(m.src+"/rpc", m.opts.mqttQoS, m.handleResponse)
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("subscribe to responses: %w", token.Error())
	}

	m.notifyMu.RLock()
	hasHandler := m.notifyHandler != nil
	m.notifyMu.RUnlock()

	if hasHandler {
		token = client.Subscribe(m.deviceID+"/events/rpc", m.opts.mqttQoS, m.handleNotification)
		token.Wait()
		if token.Error() != nil {
			return fmt.Errorf("subscribe to events: %w", token.Error())
		}
	}
	return nil
}

// onConnectionLost is called when MQTT connection is lost.
func (m *MQTT) onConnectionLost(client mqtt.Client, err error) {
	if m.isClosed() || !m.isCurrent(client) {
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
//
// The handler runs off the client's message goroutine: that goroutine also
// delivers RPC responses, so a handler that makes a Call would wait forever
// for a response queued behind itself.
func (m *MQTT) handleNotification(client mqtt.Client, msg mqtt.Message) {
	m.notifications.Add(msg.Payload())
	go m.notifications.Drain(m.notify)
}

// notify passes one notification to the registered handler.
func (m *MQTT) notify(payload []byte) {
	m.notifyMu.RLock()
	handler := m.notifyHandler
	m.notifyMu.RUnlock()

	if handler != nil {
		handler(payload)
	}
}

// waitForPublish waits for an MQTT publish token to complete with context cancellation.
func waitForPublish(ctx context.Context, token mqtt.Token) error {
	done := make(chan struct{})
	go func() {
		token.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		if token.Error() != nil {
			return fmt.Errorf("publish: %w", token.Error())
		}
		return nil
	}
}

// Call executes an RPC method call via MQTT.
//
// If not connected, this will attempt to connect first.
// The request is published to the device's RPC topic.
// The response is received on the client's response topic.
func (m *MQTT) Call(ctx context.Context, rpcReq RPCRequest) (json.RawMessage, error) {
	if m.isClosed() {
		return nil, errMQTTClosed
	}

	// Auto-connect if not connected
	client, err := m.connect(ctx)
	if err != nil {
		return nil, err
	}

	// Build request body from RPCRequest interface
	reqBody := map[string]any{
		"id":           rpcReq.GetID(),
		rpcFieldSrc:    m.src,
		rpcFieldMethod: rpcReq.GetMethod(),
	}

	// Unmarshal params from json.RawMessage and add to request
	if params := rpcReq.GetParams(); len(params) > 0 {
		var p any
		if unmarshalErr := json.Unmarshal(params, &p); unmarshalErr != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", unmarshalErr)
		}
		reqBody["params"] = p
	}

	// Add auth if present
	if auth := rpcReq.GetAuth(); auth != nil {
		reqBody["auth"] = auth
	}

	// Get request ID for response correlation
	requestID := toInt64ID(rpcReq.GetID())
	if requestID < 0 {
		requestID = m.requestID.Add(1)
		reqBody["id"] = requestID
	}

	// Create response channel
	respChan := make(chan *rpcResponse, 1)
	m.pendingMu.Lock()
	m.pending[requestID] = respChan
	m.pendingMu.Unlock()

	defer func() {
		m.pendingMu.Lock()
		delete(m.pending, requestID)
		m.pendingMu.Unlock()
	}()

	// Marshal request
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Publish to device RPC topic
	rpcTopic := m.deviceID + "/rpc"
	token := client.Publish(rpcTopic, m.opts.mqttQoS, false, data)
	if err := waitForPublish(ctx, token); err != nil {
		return nil, err
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
	if m.notifyHandler != nil {
		m.notifyMu.Unlock()
		return errHandlerAlreadyRegistered
	}
	m.notifyHandler = handler
	m.notifyMu.Unlock()

	// Subscribe to events topic if connected
	client := m.currentClient()

	if client != nil && client.IsConnected() {
		eventsTopic := m.deviceID + "/events/rpc"
		token := client.Subscribe(eventsTopic, m.opts.mqttQoS, m.handleNotification)
		token.Wait()
		if token.Error() != nil {
			m.notifyMu.Lock()
			m.notifyHandler = nil
			m.notifyMu.Unlock()
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
	client := m.currentClient()

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
	close(m.done)
	m.mu.Unlock()

	// Cancel all pending requests
	m.pendingMu.Lock()
	for id, ch := range m.pending {
		close(ch)
		delete(m.pending, id)
	}
	m.pendingMu.Unlock()

	// Disconnect from broker. Not under connMu: a dial in progress would make
	// Close wait for it; connect notices its client is gone when it returns.
	m.clientMu.Lock()
	client := m.client
	m.client = nil
	m.ready = nil
	m.clientMu.Unlock()
	if client != nil {
		client.Disconnect(250) // 250ms quiesce period
	}

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
