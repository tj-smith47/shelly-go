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

	// ErrWebSocketAlreadyListening indicates Listen was called while another
	// Listen call is still running on the same WebSocket.
	ErrWebSocketAlreadyListening = errors.New("websocket already listening")
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
	dialer   WebSocketDialer
	conn     WebSocketConn
	client   *Client
	handlers *EventHandlers

	// stopCh is closed by Close. A Connect that follows replaces it, so a
	// Listen loop holding the closed channel still stops.
	stopCh chan struct{}

	// listening is the stop channel of the running Listen loop, nil when
	// none runs.
	listening chan struct{}

	reconnectInterval    time.Duration
	maxReconnectInterval time.Duration
	pingInterval         time.Duration
	readTimeout          time.Duration

	// connID counts successful connects, so a Listen loop can tell whether
	// the current conn is still the one it was reading.
	connID uint64

	mu sync.RWMutex

	// writeMu serializes writes. WebSocket connections allow one writer at a
	// time.
	writeMu sync.Mutex

	// dialMu serializes dialing. The dialer is caller-supplied and must not
	// run under mu, where it could not call back into the WebSocket.
	dialMu sync.Mutex

	connected bool
	stopped   bool
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
//
// Calling Connect after Close makes the WebSocket usable again: Listen can be
// called anew.
func (ws *WebSocket) Connect(ctx context.Context) error {
	return ws.connect(ctx, nil)
}

// connect dials unless already connected. A nil stop is a caller's Connect,
// which revives a closed WebSocket. A non-nil stop is a Listen loop's
// reconnect, which fails with ErrWebSocketClosed once that loop was stopped.
func (ws *WebSocket) connect(ctx context.Context, stop chan struct{}) error {
	ws.dialMu.Lock()
	defer ws.dialMu.Unlock()

	ws.mu.RLock()
	stale := ws.staleLocked(stop)
	connected := ws.connected
	dialer := ws.dialer
	ws.mu.RUnlock()

	if stale {
		return ErrWebSocketClosed
	}
	if connected {
		return nil
	}

	// Build WebSocket URL
	wsURL, err := ws.buildWebSocketURL()
	if err != nil {
		return err
	}

	// Connect
	if dialer == nil {
		return errors.New("no WebSocket dialer configured - external WebSocket library required")
	}

	conn, err := dialer.Dial(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	ws.mu.Lock()
	// Close may have run during the dial.
	if ws.staleLocked(stop) {
		ws.mu.Unlock()
		conn.Close()
		return ErrWebSocketClosed
	}
	if ws.stopped {
		ws.stopCh = make(chan struct{})
		ws.stopped = false
	}
	ws.conn = conn
	ws.connID++
	ws.connected = true
	ws.mu.Unlock()

	return nil
}

// staleLocked reports whether the Listen loop owning stop has been stopped.
// A nil stop is never stale. The caller holds mu.
func (ws *WebSocket) staleLocked(stop chan struct{}) bool {
	return stop != nil && (ws.stopped || ws.stopCh != stop)
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

// Close closes the WebSocket connection and stops a running Listen, whether
// it is reading or waiting to reconnect. Calling Close again is a no-op.
func (ws *WebSocket) Close() error {
	ws.mu.Lock()
	if !ws.stopped {
		close(ws.stopCh)
		ws.stopped = true
	}
	conn := ws.conn
	ws.conn = nil
	ws.connected = false
	ws.mu.Unlock()

	if conn != nil {
		conn.Close()
	}
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
// It automatically reconnects on connection loss with exponential backoff:
// every lost connection is followed by a wait, which starts at the reconnect
// interval and doubles while connections keep dropping without delivering a
// message.
//
// Only one Listen runs at a time: a call made while another is running returns
// ErrWebSocketAlreadyListening. Close makes Listen return nil.
func (ws *WebSocket) Listen(ctx context.Context) error {
	ws.mu.Lock()
	stop := ws.stopCh
	if ws.listening == stop {
		ws.mu.Unlock()
		return ErrWebSocketAlreadyListening
	}
	ws.listening = stop
	ws.mu.Unlock()

	defer func() {
		ws.mu.Lock()
		// A loop stopped by Close may outlive a Connect and the Listen that
		// follows; it must not clear that newer loop's mark.
		if ws.listening == stop {
			ws.listening = nil
		}
		ws.mu.Unlock()
	}()

	reconnectInterval := ws.reconnectInterval
	// A server that accepts the connection and then drops it would otherwise
	// be redialed in a tight loop.
	dropInterval := ws.reconnectInterval

	for {
		if done, err := checkStopConditions(ctx, stop); done {
			return err
		}

		newInterval, shouldContinue, err := ws.attemptConnect(ctx, stop, reconnectInterval)
		reconnectInterval = newInterval
		if err != nil {
			return err
		}
		if shouldContinue {
			continue
		}

		// A failed read has already dropped the connection; the next pass
		// reconnects or stops.
		received, err := ws.readLoop(ctx, stop)
		if err == nil {
			continue
		}
		// A connection that delivered something starts the backoff over, but
		// is still waited for: a server that sends one message, such as an
		// authentication error, and hangs up would otherwise be redialed
		// without pause.
		if received {
			dropInterval = ws.reconnectInterval
		}
		next, stopped, err := ws.waitBackoff(ctx, stop, dropInterval)
		if stopped {
			return err
		}
		dropInterval = next
	}
}

// waitBackoff waits out interval and returns the doubled interval to use next
// time, capped at the maximum. stopped is true when ctx or stop ended the wait.
func (ws *WebSocket) waitBackoff(
	ctx context.Context,
	stop <-chan struct{},
	interval time.Duration,
) (next time.Duration, stopped bool, err error) {
	select {
	case <-time.After(interval):
		return min(interval*2, ws.maxReconnectInterval), false, nil
	case <-ctx.Done():
		return interval, true, ctx.Err()
	case <-stop:
		return interval, true, nil
	}
}

// checkStopConditions checks if the listener should stop.
// Returns (true, error) if should stop, (false, nil) otherwise.
func checkStopConditions(ctx context.Context, stop <-chan struct{}) (bool, error) {
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-stop:
		return true, nil
	default:
		return false, nil
	}
}

// attemptConnect tries to establish a connection with exponential backoff.
// Returns (newInterval, shouldContinue, error).
func (ws *WebSocket) attemptConnect(
	ctx context.Context,
	stop chan struct{},
	currentInterval time.Duration,
) (time.Duration, bool, error) {
	if err := ws.connect(ctx, stop); err != nil {
		next, stopped, err := ws.waitBackoff(ctx, stop, currentInterval)
		return next, !stopped, err
	}
	// Reset reconnect interval on successful connect
	return ws.reconnectInterval, false, nil
}

// handleDisconnection cleans up after the connection identified by connID was
// lost. A connection made since then belongs to someone else and is left alone.
func (ws *WebSocket) handleDisconnection(connID uint64) {
	ws.mu.Lock()
	var conn WebSocketConn
	if ws.connID == connID {
		conn = ws.conn
		ws.conn = nil
		ws.connected = false
	}
	ws.mu.Unlock()

	if conn != nil {
		conn.Close()
	}
}

// readLoop reads messages from the WebSocket connection until it fails or the
// Listen loop owning stop is stopped, then releases the connection it read.
// received reports whether the connection delivered at least one message.
func (ws *WebSocket) readLoop(ctx context.Context, stop chan struct{}) (received bool, err error) {
	ws.mu.RLock()
	stale := ws.staleLocked(stop)
	conn := ws.conn
	connID := ws.connID
	ws.mu.RUnlock()

	if stale {
		return false, nil
	}
	if conn == nil {
		return false, ErrWebSocketNotConnected
	}

	received, err = ws.readMessages(ctx, stop, conn)
	if err != nil {
		ws.handleDisconnection(connID)
	}
	return received, err
}

// readMessages dispatches messages read from conn.
func (ws *WebSocket) readMessages(
	ctx context.Context,
	stop <-chan struct{},
	conn WebSocketConn,
) (received bool, err error) {
	for {
		if done, err := checkStopConditions(ctx, stop); done {
			return received, err
		}

		// Set read deadline
		if err := conn.SetReadDeadline(time.Now().Add(ws.readTimeout)); err != nil {
			return received, fmt.Errorf("failed to set read deadline: %w", err)
		}

		// Read message
		_, data, err := conn.ReadMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return received, ErrWebSocketClosed
			}
			return received, fmt.Errorf("failed to read message: %w", err)
		}
		received = true

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

// SendMessage sends a message over the WebSocket connection. It is safe to
// call from several goroutines; the writes happen one at a time.
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

	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()

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
