package transport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/tj-smith47/shelly-go/internal/digest"
	"github.com/tj-smith47/shelly-go/internal/serial"
	"github.com/tj-smith47/shelly-go/types"
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
	done          chan struct{}
	opts          *options
	digest        *digest.Session
	session       *wsSession
	notifyHandler NotificationHandler
	pending       map[int64]chan *rpcResponse
	url           string
	src           string
	notifications serial.Queue[[]byte]
	connState
	requestID atomic.Int64
	mu        sync.RWMutex
	notifyMu  sync.RWMutex
	connMu    sync.Mutex
	pendingMu sync.Mutex
	closed    bool
}

// wsSession is one established connection. The read and ping loops are handed
// the session they were started for and never look at the transport's current
// one, so a loop left over from an earlier connection cannot tear down, read
// from or keep alive its successor.
type wsSession struct {
	conn     *websocket.Conn
	stopPing chan struct{}
	writeMu  sync.Mutex
	stopOnce sync.Once
}

// stop ends the session's ping loop and closes its connection. It is safe to
// call from several goroutines.
func (s *wsSession) stop() {
	s.stopOnce.Do(func() {
		close(s.stopPing)
		s.conn.Close()
	})
}

// rpcResponse represents a JSON-RPC response.
type rpcResponse struct {
	Error  *RPCError       `json:"error,omitempty"`
	Src    string          `json:"src,omitempty"`
	Dst    string          `json:"dst,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	ID     int64           `json:"id"`
}

// RPCError is an error reply from a device to an RPC frame sent over the
// WebSocket or MQTT transport. errors.Is matches the types error its code
// maps to (types.MapErrorCode): a 401 is types.ErrAuth.
type RPCError struct {
	// Message is the device's error message. For a 401 it is the JSON digest
	// challenge.
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Error implements the error interface.
func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// Unwrap returns the types error the code maps to.
func (e *RPCError) Unwrap() error {
	return types.MapErrorCode(e.Code)
}

// rpcNotification represents a JSON-RPC notification.
type rpcNotification struct {
	Src    string          `json:"src,omitempty"`
	Dst    string          `json:"dst,omitempty"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// toInt64ID converts an interface ID to int64, returning -1 if not a numeric type.
// For uint64 values that exceed int64 max, returns -1 to indicate overflow.
func toInt64ID(id any) int64 {
	switch v := id.(type) {
	case int64:
		return v
	case uint64:
		// Check for overflow: max int64 is 9223372036854775807
		if v > 1<<63-1 {
			return -1
		}
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
		url:     url,
		src:     fmt.Sprintf("shelly-go-%d", time.Now().UnixNano()),
		opts:    options,
		digest:  options.digestSession(),
		pending: make(map[int64]chan *rpcResponse),
		done:    make(chan struct{}),
	}
}

// Connect establishes the WebSocket connection.
// This must be called before making any RPC calls.
func (w *WebSocket) Connect(ctx context.Context) error {
	_, err := w.connect(ctx)
	return err
}

// connect returns the current session, dialing first when there is none.
func (w *WebSocket) connect(ctx context.Context) (*wsSession, error) {
	// Registered first so it runs last: state listeners are told only after
	// connMu is released, and may therefore call Connect or Close themselves.
	defer w.flushState()

	w.connMu.Lock()
	defer w.connMu.Unlock()

	if w.isClosed() {
		return nil, fmt.Errorf("websocket is closed")
	}

	if w.session != nil {
		// A reconnect attempt that finds the connection already restored
		// must not leave the state at reconnecting.
		w.queueState(StateConnected)
		return w.session, nil
	}

	w.queueState(StateConnecting)

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
		w.queueState(StateDisconnected)
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	session := &wsSession{conn: conn, stopPing: make(chan struct{})}
	w.session = session
	w.queueState(StateConnected)

	go w.readLoop(session)

	if w.opts.pingInterval > 0 {
		go w.pingLoop(session)
	}

	return session, nil
}

// Call executes an RPC method call over WebSocket.
//
// If not connected, this will attempt to connect first.
// The request is correlated with the response using a unique ID.
//
// A device error reply is returned as a *RPCError. With WithDigestAuth, a 401
// carrying a digest challenge is answered and the frame sent once more; later
// frames, including those after a reconnect, reuse the device's nonce until
// it is refused. A request that carries its own auth object is sent as is.
func (w *WebSocket) Call(ctx context.Context, rpcReq RPCRequest) (json.RawMessage, error) {
	if w.isClosed() {
		return nil, fmt.Errorf("websocket is closed")
	}

	reqBody, requestID, err := w.buildFrame(rpcReq)
	if err != nil {
		return nil, err
	}

	answerChallenge := w.digest != nil
	if auth := rpcReq.GetAuth(); auth != nil {
		reqBody["auth"] = auth
		answerChallenge = false
	} else if answerChallenge && w.digest.Ready() {
		frame, frameErr := w.digest.Frame()
		if frameErr != nil {
			return nil, frameErr
		}
		reqBody["auth"] = frame
	}

	resp, err := w.roundTrip(ctx, reqBody, requestID)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil && resp.Error.Code == http.StatusUnauthorized && answerChallenge {
		if resp, err = w.answerChallenge(ctx, resp.Error, reqBody, requestID); err != nil {
			return nil, err
		}
	}

	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp.Result, nil
}

// buildFrame builds the request frame for rpcReq without its auth object and
// returns the id its response will carry.
func (w *WebSocket) buildFrame(rpcReq RPCRequest) (reqBody map[string]any, requestID int64, err error) {
	reqBody = map[string]any{
		"id":           rpcReq.GetID(),
		rpcFieldSrc:    w.src,
		rpcFieldMethod: rpcReq.GetMethod(),
	}

	if params := rpcReq.GetParams(); len(params) > 0 {
		var p any
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal params: %w", err)
		}
		reqBody["params"] = p
	}

	requestID = toInt64ID(rpcReq.GetID())
	if requestID < 0 {
		requestID = w.requestID.Add(1)
		reqBody["id"] = requestID
	}
	return reqBody, requestID, nil
}

// answerChallenge answers the digest challenge in a 401 reply and sends the
// frame once more.
func (w *WebSocket) answerChallenge(
	ctx context.Context, unauthorized *RPCError, reqBody map[string]any, requestID int64,
) (*rpcResponse, error) {
	ch, err := digest.ParseFrameChallenge(unauthorized.Message)
	if err != nil {
		return nil, fmt.Errorf("%w (%w)", unauthorized, err)
	}
	w.digest.Accept(ch)
	frame, err := w.digest.Frame()
	if err != nil {
		return nil, err
	}
	reqBody["auth"] = frame
	return w.roundTrip(ctx, reqBody, requestID)
}

// roundTrip sends one frame, connecting first when there is no session, and
// waits for the response with requestID.
func (w *WebSocket) roundTrip(ctx context.Context, reqBody map[string]any, requestID int64) (*rpcResponse, error) {
	session, err := w.connect(ctx)
	if err != nil {
		return nil, err
	}

	respChan := make(chan *rpcResponse, 1)
	w.pendingMu.Lock()
	w.pending[requestID] = respChan
	w.pendingMu.Unlock()

	defer func() {
		w.pendingMu.Lock()
		delete(w.pending, requestID)
		w.pendingMu.Unlock()
	}()

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Written to the session this call started with: if the connection drops
	// meanwhile the write fails, where the transport's current session could
	// by now be nil.
	session.writeMu.Lock()
	err = session.conn.WriteMessage(websocket.TextMessage, data)
	session.writeMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("write message: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-w.done:
		return nil, errors.New("WebSocket transport closed while waiting for response")
	case resp := <-respChan:
		if resp == nil {
			return nil, errors.New("connection lost while waiting for response")
		}
		return resp, nil
	}
}

// readLoop reads messages from one session until it fails or is stopped.
func (w *WebSocket) readLoop(session *wsSession) {
	for {
		_, message, err := session.conn.ReadMessage()
		if err != nil {
			w.handleDisconnect(session)
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
		// Delivered off this goroutine: it is the only one that can hand a
		// response to Call, so a handler that makes a call from here would
		// wait for a reply nobody can deliver. The queue keeps notifications
		// in arrival order and the handler from overlapping itself.
		w.notifications.Add(message)
		go w.notifications.Drain(w.notify)
	}
}

// notify passes one notification to the handler subscribed at that moment.
func (w *WebSocket) notify(message []byte) {
	w.notifyMu.RLock()
	handler := w.notifyHandler
	w.notifyMu.RUnlock()

	if handler != nil {
		handler(message)
	}
}

// handleDisconnect tears down a failed session. A session that has already
// been replaced or closed is only stopped: its successor's pending calls and
// state are not this loop's to touch.
func (w *WebSocket) handleDisconnect(session *wsSession) {
	session.stop()

	w.connMu.Lock()
	current := w.session == session
	if current {
		w.session = nil
	}
	w.connMu.Unlock()

	if !current || w.isClosed() {
		return
	}

	// Cancel all pending requests
	w.pendingMu.Lock()
	for id, ch := range w.pending {
		close(ch)
		delete(w.pending, id)
	}
	w.pendingMu.Unlock()

	w.setState(StateDisconnected)

	if w.opts.reconnect {
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

		// Close must not have to wait out the backoff.
		select {
		case <-w.done:
			return
		case <-time.After(delay):
		}
		delay = time.Duration(float64(delay) * w.opts.retryBackoff)
	}

	if !w.isClosed() {
		w.setState(StateDisconnected)
	}
}

// pingLoop sends periodic pings to keep one session alive.
func (w *WebSocket) pingLoop(session *wsSession) {
	ticker := time.NewTicker(w.opts.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-session.stopPing:
			return
		case <-ticker.C:
			err := session.conn.WriteControl(
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
	close(w.done)
	w.mu.Unlock()

	w.connMu.Lock()
	session := w.session
	w.session = nil
	w.connMu.Unlock()

	var closeErr error
	if session != nil {
		closeErr = session.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
		session.stop()
	}

	w.setState(StateClosed)

	// A peer that has already gone away cannot be sent a close frame; the
	// transport is closed all the same, so that is not a failure to report.
	if closeErr != nil && !errors.Is(closeErr, net.ErrClosed) && !errors.Is(closeErr, websocket.ErrCloseSent) {
		return fmt.Errorf("send close frame: %w", closeErr)
	}
	return nil
}

// basicAuth encodes credentials for basic authentication.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64Encode([]byte(auth))
}

// base64Encode encodes data as base64.
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
