package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// WebSocket connection errors.
var (
	// ErrWebSocketClosed indicates the WebSocket connection is closed.
	ErrWebSocketClosed = errors.New("websocket closed")

	// ErrWebSocketNotConnected indicates the WebSocket is not connected.
	ErrWebSocketNotConnected = errors.New("websocket not connected")
)

// WebSocketDialer is an interface for establishing WebSocket connections.
// This allows for testing with mock dialers.
type WebSocketDialer interface {
	// Dial establishes a WebSocket connection.
	Dial(ctx context.Context, url string, headers http.Header) (WebSocketConn, error)
}

// WebSocketConn is an interface for WebSocket connections.
// This allows for testing with mock connections.
type WebSocketConn interface {
	// ReadMessage reads a message from the connection.
	ReadMessage() (messageType int, data []byte, err error)

	// WriteMessage writes a message to the connection.
	WriteMessage(messageType int, data []byte) error

	// Close closes the connection.
	Close() error

	// SetReadDeadline sets the read deadline.
	SetReadDeadline(t time.Time) error

	// SetWriteDeadline sets the write deadline.
	SetWriteDeadline(t time.Time) error
}

// WebSocket manages a WebSocket connection to the Shelly Cloud.
type WebSocket struct {
	dialer               WebSocketDialer
	conn                 WebSocketConn
	client               *Client
	handlers             *EventHandlers
	stopCh               chan struct{}
	reconnectInterval    time.Duration
	maxReconnectInterval time.Duration
	pingInterval         time.Duration
	readTimeout          time.Duration
	mu                   sync.RWMutex
	connected            bool
}

// WebSocketOption is a functional option for configuring the WebSocket.
type WebSocketOption func(*WebSocket)

// WithDialer sets a custom WebSocket dialer.
func WithDialer(dialer WebSocketDialer) WebSocketOption {
	return func(ws *WebSocket) {
		ws.dialer = dialer
	}
}

// WithReconnectInterval sets the initial reconnection interval.
func WithReconnectInterval(interval time.Duration) WebSocketOption {
	return func(ws *WebSocket) {
		ws.reconnectInterval = interval
	}
}

// WithMaxReconnectInterval sets the maximum reconnection interval.
func WithMaxReconnectInterval(interval time.Duration) WebSocketOption {
	return func(ws *WebSocket) {
		ws.maxReconnectInterval = interval
	}
}

// WithPingInterval sets the ping interval.
func WithPingInterval(interval time.Duration) WebSocketOption {
	return func(ws *WebSocket) {
		ws.pingInterval = interval
	}
}

// WithReadTimeout sets the read timeout.
func WithReadTimeout(timeout time.Duration) WebSocketOption {
	return func(ws *WebSocket) {
		ws.readTimeout = timeout
	}
}

// NewWebSocket creates a new WebSocket connection manager.
func NewWebSocket(client *Client, opts ...WebSocketOption) *WebSocket {
	ws := &WebSocket{
		client:               client,
		handlers:             NewEventHandlers(),
		reconnectInterval:    5 * time.Second,
		maxReconnectInterval: 5 * time.Minute,
		pingInterval:         30 * time.Second,
		readTimeout:          60 * time.Second,
		stopCh:               make(chan struct{}),
	}

	for _, opt := range opts {
		opt(ws)
	}

	return ws
}

// ConnectWebSocket creates a new WebSocket connection to the Shelly Cloud.
func (c *Client) ConnectWebSocket(ctx context.Context, opts ...WebSocketOption) (*WebSocket, error) {
	ws := NewWebSocket(c, opts...)
	if err := ws.Connect(ctx); err != nil {
		return nil, err
	}
	return ws, nil
}

// Connect establishes the WebSocket connection.
func (ws *WebSocket) Connect(ctx context.Context) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.connected {
		return nil
	}

	// Build WebSocket URL
	wsURL, err := ws.buildWebSocketURL()
	if err != nil {
		return err
	}

	// Connect
	if ws.dialer == nil {
		return errors.New("no WebSocket dialer configured - external WebSocket library required")
	}

	conn, err := ws.dialer.Dial(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	ws.conn = conn
	ws.connected = true
	ws.stopCh = make(chan struct{})

	return nil
}

// buildWebSocketURL builds the WebSocket URL for the Cloud API.
func (ws *WebSocket) buildWebSocketURL() (string, error) {
	baseURL := ws.client.GetBaseURL()
	token := ws.client.GetToken()

	if baseURL == "" {
		return "", ErrNoUserAPIURL
	}

	if token == "" {
		return "", ErrUnauthorized
	}

	// Parse the base URL
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Build WebSocket URL
	// wss://<server>:6113/shelly/wss/hk_sock?t=<token>
	wsURL := fmt.Sprintf("wss://%s:%d/shelly/wss/hk_sock?t=%s",
		u.Hostname(),
		DefaultWSPort,
		url.QueryEscape(token),
	)

	return wsURL, nil
}

// Close closes the WebSocket connection.
func (ws *WebSocket) Close() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if !ws.connected {
		return nil
	}

	// Signal stop
	close(ws.stopCh)

	// Close connection
	if ws.conn != nil {
		ws.conn.Close()
	}

	ws.connected = false
	return nil
}

// IsConnected returns true if the WebSocket is connected.
func (ws *WebSocket) IsConnected() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.connected
}

// Listen starts listening for events on the WebSocket connection.
// This method blocks until the connection is closed or an error occurs.
// It automatically reconnects on connection loss with exponential backoff.
func (ws *WebSocket) Listen(ctx context.Context) error {
	reconnectInterval := ws.reconnectInterval

	for {
		if done, err := ws.checkStopConditions(ctx); done {
			return err
		}

		// Ensure connected
		if !ws.IsConnected() {
			newInterval, shouldContinue, err := ws.attemptConnect(ctx, reconnectInterval)
			reconnectInterval = newInterval
			if err != nil {
				return err
			}
			if shouldContinue {
				continue
			}
		}

		// Read and process messages
		if err := ws.readLoop(ctx); err != nil {
			ws.handleDisconnection()

			if done, stopErr := ws.checkStopConditions(ctx); done {
				return stopErr
			}
		}
	}
}

// checkStopConditions checks if the listener should stop.
// Returns (true, error) if should stop, (false, nil) otherwise.
func (ws *WebSocket) checkStopConditions(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-ws.stopCh:
		return true, nil
	default:
		return false, nil
	}
}

// attemptConnect tries to establish a connection with exponential backoff.
// Returns (newInterval, shouldContinue, error).
func (ws *WebSocket) attemptConnect(
	ctx context.Context,
	currentInterval time.Duration,
) (time.Duration, bool, error) {
	if err := ws.Connect(ctx); err != nil {
		// Wait before retry with exponential backoff
		select {
		case <-time.After(currentInterval):
			newInterval := currentInterval * 2
			if newInterval > ws.maxReconnectInterval {
				newInterval = ws.maxReconnectInterval
			}
			return newInterval, true, nil
		case <-ctx.Done():
			return currentInterval, false, ctx.Err()
		case <-ws.stopCh:
			return currentInterval, false, nil
		}
	}
	// Reset reconnect interval on successful connect
	return ws.reconnectInterval, false, nil
}

// handleDisconnection cleans up after a connection loss.
func (ws *WebSocket) handleDisconnection() {
	ws.mu.Lock()
	ws.connected = false
	if ws.conn != nil {
		ws.conn.Close()
		ws.conn = nil
	}
	ws.mu.Unlock()
}

// readLoop reads messages from the WebSocket connection.
func (ws *WebSocket) readLoop(ctx context.Context) error {
	ws.mu.RLock()
	conn := ws.conn
	ws.mu.RUnlock()

	if conn == nil {
		return ErrWebSocketNotConnected
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ws.stopCh:
			return nil
		default:
		}

		// Set read deadline
		if err := conn.SetReadDeadline(time.Now().Add(ws.readTimeout)); err != nil {
			return fmt.Errorf("failed to set read deadline: %w", err)
		}

		// Read message
		_, data, err := conn.ReadMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return ErrWebSocketClosed
			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		// Parse and dispatch message
		ws.handleMessage(data)
	}
}

// handleMessage parses and dispatches a WebSocket message.
func (ws *WebSocket) handleMessage(data []byte) {
	var msg WebSocketMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		// Invalid JSON, ignore
		return
	}

	// Dispatch based on event type
	ws.handlers.Dispatch(&msg)
}

// OnDeviceOnline registers a handler for device online events.
func (ws *WebSocket) OnDeviceOnline(handler func(deviceID string)) {
	ws.handlers.OnDeviceOnline(handler)
}

// OnDeviceOffline registers a handler for device offline events.
func (ws *WebSocket) OnDeviceOffline(handler func(deviceID string)) {
	ws.handlers.OnDeviceOffline(handler)
}

// OnStatusChange registers a handler for device status change events.
func (ws *WebSocket) OnStatusChange(handler func(deviceID string, status json.RawMessage)) {
	ws.handlers.OnStatusChange(handler)
}

// OnNotifyStatus registers a handler for Gen2+ NotifyStatus events.
func (ws *WebSocket) OnNotifyStatus(handler func(deviceID string, status json.RawMessage)) {
	ws.handlers.OnNotifyStatus(handler)
}

// OnNotifyFullStatus registers a handler for Gen2+ NotifyFullStatus events.
func (ws *WebSocket) OnNotifyFullStatus(handler func(deviceID string, status json.RawMessage)) {
	ws.handlers.OnNotifyFullStatus(handler)
}

// OnNotifyEvent registers a handler for Gen2+ NotifyEvent events.
func (ws *WebSocket) OnNotifyEvent(handler func(deviceID string, event json.RawMessage)) {
	ws.handlers.OnNotifyEvent(handler)
}

// OnMessage registers a handler for all messages.
func (ws *WebSocket) OnMessage(handler func(msg *WebSocketMessage)) {
	ws.handlers.OnMessage(handler)
}

// SendMessage sends a message over the WebSocket connection.
func (ws *WebSocket) SendMessage(ctx context.Context, msg any) error {
	ws.mu.RLock()
	conn := ws.conn
	connected := ws.connected
	ws.mu.RUnlock()

	if !connected || conn == nil {
		return ErrWebSocketNotConnected
	}

	// Marshal message
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Set write deadline
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send message (text message type = 1)
	if err := conn.WriteMessage(1, data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// GetWebSocketURL returns the WebSocket URL for the Cloud API.
// This can be used to connect with an external WebSocket library.
func (c *Client) GetWebSocketURL() (string, error) {
	baseURL := c.GetBaseURL()
	token := c.GetToken()

	if baseURL == "" {
		return "", ErrNoUserAPIURL
	}

	if token == "" {
		return "", ErrUnauthorized
	}

	// Parse the base URL
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Build WebSocket URL
	wsURL := fmt.Sprintf("wss://%s:%d/shelly/wss/hk_sock?t=%s",
		u.Hostname(),
		DefaultWSPort,
		url.QueryEscape(token),
	)

	return wsURL, nil
}

// extractHostname extracts the hostname from a URL.
func extractHostname(rawURL string) string {
	// Remove protocol
	if idx := strings.Index(rawURL, "://"); idx >= 0 {
		rawURL = rawURL[idx+3:]
	}

	// Remove path
	if idx := strings.Index(rawURL, "/"); idx >= 0 {
		rawURL = rawURL[:idx]
	}

	// Remove port
	if idx := strings.Index(rawURL, ":"); idx >= 0 {
		rawURL = rawURL[:idx]
	}

	return rawURL
}
