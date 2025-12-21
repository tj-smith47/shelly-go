package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket is a WebSocket transport for Shelly devices.
// Supports bidirectional real-time communication for Gen2+ devices.
//
// The WebSocket transport provides:
//   - Full duplex communication
//   - Real-time notifications
//   - Automatic reconnection
//   - Request/response correlation
//   - Ping/pong keepalive
type WebSocket struct {
	opts           *options
	conn           *websocket.Conn
	notifyHandler  NotificationHandler
	pending        map[int64]chan *rpcResponse
	stopPing       chan struct{}
	url            string
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

// rpcResponse represents a JSON-RPC response.
type rpcResponse struct {
	Error  *rpcError       `json:"error,omitempty"`
	Src    string          `json:"src,omitempty"`
	Dst    string          `json:"dst,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	ID     int64           `json:"id"`
}

// rpcError represents a JSON-RPC error.
type rpcError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// rpcNotification represents a JSON-RPC notification.
type rpcNotification struct {
	Src    string          `json:"src,omitempty"`
	Dst    string          `json:"dst,omitempty"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// toInt64ID converts an interface ID to int64, returning -1 if not a numeric type.
func toInt64ID(id any) int64 {
	switch v := id.(type) {
	case int64:
		return v
	case uint64:
		return int64(v)
	case int:
		return int64(v)
	default:
		return -1
	}
}

// NewWebSocket creates a new WebSocket transport.
//
// The url should be the WebSocket endpoint (e.g., "ws://192.168.1.100/rpc").
// A unique source identifier is used for request/response correlation.
//
// Example:
//
//	ws := NewWebSocket("ws://192.168.1.100/rpc",
//	    WithReconnect(true),
//	    WithPingInterval(30*time.Second))
func NewWebSocket(url string, opts ...Option) *WebSocket {
	options := defaultOptions()
	applyOptions(options, opts)

	return &WebSocket{
		url:            url,
		src:            fmt.Sprintf("shelly-go-%d", time.Now().UnixNano()),
		opts:           options,
		state:          StateDisconnected,
		pending:        make(map[int64]chan *rpcResponse),
		stateCallbacks: make([]func(ConnectionState), 0),
	}
}

// Connect establishes the WebSocket connection.
// This must be called before making any RPC calls.
func (w *WebSocket) Connect(ctx context.Context) error {
	w.connMu.Lock()
	defer w.connMu.Unlock()

	if w.isClosed() {
		return fmt.Errorf("websocket is closed")
	}

	if w.conn != nil {
		return nil // already connected
	}

	w.setState(StateConnecting)

	dialer := websocket.Dialer{
		TLSClientConfig:  w.opts.tlsConfig,
		HandshakeTimeout: w.opts.timeout,
	}

	var header http.Header
	if w.opts.authType == authTypeBasic && w.opts.username != "" {
		header = http.Header{}
		header.Set("Authorization", "Basic "+basicAuth(w.opts.username, w.opts.password))
	}

	conn, resp, err := dialer.DialContext(ctx, w.url, header)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		w.setState(StateDisconnected)
		return fmt.Errorf("websocket dial: %w", err)
	}

	w.conn = conn
	w.stopPing = make(chan struct{})
	w.setState(StateConnected)

	// Start message reader
	go w.readLoop()

	// Start ping loop if enabled
	if w.opts.pingInterval > 0 {
		go w.pingLoop()
	}

	return nil
}

// Call executes an RPC method call over WebSocket.
//
// If not connected, this will attempt to connect first.
// The request is correlated with the response using a unique ID.
func (w *WebSocket) Call(ctx context.Context, rpcReq RPCRequest) (json.RawMessage, error) {
	if w.isClosed() {
		return nil, fmt.Errorf("websocket is closed")
	}

	// Auto-connect if not connected
	if w.conn == nil {
		if err := w.Connect(ctx); err != nil {
			return nil, err
		}
	}

	// Build request body from RPCRequest interface
	reqBody := map[string]any{
		"id":     rpcReq.GetID(),
		"src":    w.src,
		"method": rpcReq.GetMethod(),
	}

	// Unmarshal params from json.RawMessage and add to request
	if params := rpcReq.GetParams(); len(params) > 0 {
		var p any
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
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
		requestID = w.requestID.Add(1)
		reqBody["id"] = requestID
	}

	// Create response channel
	respChan := make(chan *rpcResponse, 1)
	w.pendingMu.Lock()
	w.pending[requestID] = respChan
	w.pendingMu.Unlock()

	defer func() {
		w.pendingMu.Lock()
		delete(w.pending, requestID)
		w.pendingMu.Unlock()
	}()

	// Marshal and send request
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	w.connMu.Lock()
	err = w.conn.WriteMessage(websocket.TextMessage, data)
	w.connMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("write message: %w", err)
	}

	// Wait for response
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

// readLoop reads messages from the WebSocket connection.
func (w *WebSocket) readLoop() {
	for {
		if w.isClosed() {
			return
		}

		w.connMu.Lock()
		conn := w.conn
		w.connMu.Unlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if w.isClosed() {
				return
			}

			// Handle reconnection
			w.handleDisconnect(err)
			return
		}

		w.handleMessage(message)
	}
}

// handleMessage processes an incoming WebSocket message.
func (w *WebSocket) handleMessage(message []byte) {
	// Try to parse as response first (has "id" and "result"/"error")
	var resp rpcResponse
	if err := json.Unmarshal(message, &resp); err == nil && resp.ID != 0 {
		w.pendingMu.Lock()
		if ch, ok := w.pending[resp.ID]; ok {
			select {
			case ch <- &resp:
			default:
			}
		}
		w.pendingMu.Unlock()
		return
	}

	// Try to parse as notification (has "method" but no "id")
	var notif rpcNotification
	if err := json.Unmarshal(message, &notif); err == nil && notif.Method != "" {
		w.notifyMu.RLock()
		handler := w.notifyHandler
		w.notifyMu.RUnlock()

		if handler != nil {
			handler(message)
		}
	}
}

// handleDisconnect handles a WebSocket disconnection.
func (w *WebSocket) handleDisconnect(_ error) {
	w.connMu.Lock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.connMu.Unlock()

	// Close stop ping channel
	select {
	case <-w.stopPing:
	default:
		close(w.stopPing)
	}

	// Cancel all pending requests
	w.pendingMu.Lock()
	for id, ch := range w.pending {
		close(ch)
		delete(w.pending, id)
	}
	w.pendingMu.Unlock()

	w.setState(StateDisconnected)

	// Attempt reconnection if enabled
	if w.opts.reconnect && !w.isClosed() {
		go w.reconnect()
	}
}

// reconnect attempts to reconnect with exponential backoff.
func (w *WebSocket) reconnect() {
	delay := w.opts.retryDelay
	maxRetries := w.opts.maxRetries

	for attempt := 0; attempt < maxRetries; attempt++ {
		if w.isClosed() {
			return
		}

		w.setState(StateReconnecting)

		ctx, cancel := context.WithTimeout(context.Background(), w.opts.timeout)
		err := w.Connect(ctx)
		cancel()

		if err == nil {
			return
		}

		// Exponential backoff
		time.Sleep(delay)
		delay = time.Duration(float64(delay) * w.opts.retryBackoff)
	}

	w.setState(StateDisconnected)
}

// pingLoop sends periodic pings to keep the connection alive.
func (w *WebSocket) pingLoop() {
	ticker := time.NewTicker(w.opts.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopPing:
			return
		case <-ticker.C:
			w.connMu.Lock()
			conn := w.conn
			w.connMu.Unlock()

			if conn == nil {
				return
			}

			err := conn.WriteControl(
				websocket.PingMessage,
				[]byte{},
				time.Now().Add(w.opts.pongTimeout),
			)
			if err != nil {
				return
			}
		}
	}
}

// Subscribe registers a handler for incoming notifications.
func (w *WebSocket) Subscribe(handler NotificationHandler) error {
	w.notifyMu.Lock()
	defer w.notifyMu.Unlock()

	if w.notifyHandler != nil {
		return errHandlerAlreadyRegistered
	}

	w.notifyHandler = handler
	return nil
}

// Unsubscribe removes the notification handler.
func (w *WebSocket) Unsubscribe() error {
	w.notifyMu.Lock()
	defer w.notifyMu.Unlock()
	w.notifyHandler = nil
	return nil
}

// State returns the current connection state.
func (w *WebSocket) State() ConnectionState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state
}

// OnStateChange registers a callback for connection state changes.
func (w *WebSocket) OnStateChange(callback func(ConnectionState)) {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	w.stateCallbacks = append(w.stateCallbacks, callback)
}

// setState updates the connection state and notifies callbacks.
func (w *WebSocket) setState(state ConnectionState) {
	w.mu.Lock()
	w.state = state
	w.mu.Unlock()

	w.stateMu.RLock()
	callbacks := make([]func(ConnectionState), len(w.stateCallbacks))
	copy(callbacks, w.stateCallbacks)
	w.stateMu.RUnlock()

	for _, cb := range callbacks {
		cb(state)
	}
}

// isClosed returns true if the transport is closed.
func (w *WebSocket) isClosed() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.closed
}

// Close closes the WebSocket connection.
func (w *WebSocket) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()

	// Stop ping loop (only if it was started)
	if w.stopPing != nil {
		select {
		case <-w.stopPing:
		default:
			close(w.stopPing)
		}
	}

	// Close connection
	w.connMu.Lock()
	if w.conn != nil {
		err := w.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
		if err == nil {
			w.conn.Close()
		}
		w.conn = nil
	}
	w.connMu.Unlock()

	w.setState(StateClosed)
	return nil
}

// basicAuth encodes credentials for basic authentication.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64Encode([]byte(auth))
}

// base64Encode encodes data as base64.
func base64Encode(data []byte) string {
	const encoding = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	result := make([]byte, 0, (len(data)+2)/3*4)

	for i := 0; i < len(data); i += 3 {
		var n uint32
		switch len(data) - i {
		case 1:
			n = uint32(data[i]) << 16
			result = append(result, encoding[n>>18], encoding[(n>>12)&0x3f], '=', '=')
		case 2:
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8
			result = append(result, encoding[n>>18], encoding[(n>>12)&0x3f], encoding[(n>>6)&0x3f], '=')
		default:
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
			result = append(result, encoding[n>>18], encoding[(n>>12)&0x3f], encoding[(n>>6)&0x3f], encoding[n&0x3f])
		}
	}

	return string(result)
}
