package cloud

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

// testTimeout bounds every wait, so a regression fails instead of hanging.
const testTimeout = 5 * time.Second

// heldConn is a connection whose ReadMessage blocks until the test releases
// it, which lets a test decide exactly when a read loop wakes up.
type heldConn struct {
	reading     chan struct{}
	release     chan struct{}
	releaseOnce sync.Once
	mu          sync.Mutex
	readers     int
	closed      bool
	holdOnClose bool
}

func newHeldConn(holdOnClose bool) *heldConn {
	return &heldConn{
		reading:     make(chan struct{}, 16),
		release:     make(chan struct{}),
		holdOnClose: holdOnClose,
	}
}

func (c *heldConn) ReadMessage() (int, []byte, error) {
	c.mu.Lock()
	c.readers++
	c.mu.Unlock()

	c.reading <- struct{}{}
	<-c.release
	return 0, nil, ErrWebSocketClosed
}

func (c *heldConn) WriteMessage(int, []byte) error { return nil }

func (c *heldConn) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()

	if !c.holdOnClose {
		c.unblock()
	}
	return nil
}

func (c *heldConn) SetReadDeadline(time.Time) error { return nil }

func (c *heldConn) SetWriteDeadline(time.Time) error { return nil }

func (c *heldConn) unblock() { c.releaseOnce.Do(func() { close(c.release) }) }

func (c *heldConn) state() (readers int, closed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readers, c.closed
}

// scriptedDialer hands out its conns in order and fails once none are left.
// Every call is announced on dialed.
type scriptedDialer struct {
	dialed chan struct{}
	conns  []WebSocketConn
	mu     sync.Mutex
}

func newScriptedDialer(conns ...WebSocketConn) *scriptedDialer {
	return &scriptedDialer{dialed: make(chan struct{}, 64), conns: conns}
}

func (d *scriptedDialer) Dial(context.Context, string, http.Header) (WebSocketConn, error) {
	d.mu.Lock()
	var conn WebSocketConn
	if len(d.conns) > 0 {
		conn, d.conns = d.conns[0], d.conns[1:]
	}
	d.mu.Unlock()

	select {
	case d.dialed <- struct{}{}:
	default:
	}
	if conn == nil {
		return nil, errors.New("connection refused")
	}
	return conn, nil
}

func (d *scriptedDialer) add(conn WebSocketConn) {
	d.mu.Lock()
	d.conns = append(d.conns, conn)
	d.mu.Unlock()
}

func newTestWebSocket(dialer WebSocketDialer, opts ...WebSocketOption) *WebSocket {
	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL("https://shelly-49-eu.shelly.cloud"),
	)
	return NewWebSocket(client, append([]WebSocketOption{WithDialer(dialer)}, opts...)...)
}

func listenAsync(ws *WebSocket) <-chan error {
	errCh := make(chan error, 1)
	go func() { errCh <- ws.Listen(context.Background()) }()
	return errCh
}

func waitSignal(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func waitListen(t *testing.T, errCh <-chan error, what string) error {
	t.Helper()
	select {
	case err := <-errCh:
		return err
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", what)
		return nil
	}
}

func TestEventHandlersDispatchNeverRunsHandlerConcurrently(t *testing.T) {
	handlers := NewEventHandlers()

	// Deliberately unsynchronized: the race detector fails the test if two
	// deliveries ever overlap.
	var seen []string
	handlers.OnMessage(func(msg *WebSocketMessage) {
		seen = append(seen, msg.DeviceID)
	})
	var online int
	handlers.OnDeviceOnline(func(string) { online++ })

	const dispatchers, perDispatcher = 8, 50
	var wg sync.WaitGroup
	for range dispatchers {
		wg.Go(func() {
			for range perDispatcher {
				handlers.Dispatch(&WebSocketMessage{Event: EventDeviceOnline, DeviceID: "d"})
			}
		})
	}
	wg.Wait()

	if want := dispatchers * perDispatcher; len(seen) != want || online != want {
		t.Errorf("delivered %d messages and %d online events, want %d of each", len(seen), online, want)
	}
}

func TestEventHandlersDispatchPreservesOrder(t *testing.T) {
	handlers := NewEventHandlers()

	var seen []string
	handlers.OnMessage(func(msg *WebSocketMessage) {
		seen = append(seen, msg.DeviceID)
		if msg.DeviceID == "first" {
			// Dispatched mid-delivery, so it must come after "first" is done
			// and must not recurse into the handler.
			handlers.Dispatch(&WebSocketMessage{DeviceID: "nested"})
			seen = append(seen, "first-done")
		}
	})

	handlers.Dispatch(&WebSocketMessage{DeviceID: "first"})
	handlers.Dispatch(&WebSocketMessage{DeviceID: "second"})

	want := []string{"first", "first-done", "nested", "second"}
	if len(seen) != len(want) {
		t.Fatalf("seen = %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("seen = %v, want %v", seen, want)
		}
	}
}

func TestEventHandlersHandlerMayRegisterAndClear(t *testing.T) {
	handlers := NewEventHandlers()

	var late, offline int
	handlers.OnDeviceOnline(func(string) {
		handlers.OnDeviceOnline(func(string) { late++ })
		handlers.OnDeviceOffline(func(string) { offline++ })
	})
	handlers.OnDeviceOffline(func(string) { handlers.Clear() })

	done := make(chan struct{})
	go func() {
		defer close(done)
		handlers.Dispatch(&WebSocketMessage{Event: EventDeviceOnline})
		handlers.Dispatch(&WebSocketMessage{Event: EventDeviceOnline})
		handlers.Dispatch(&WebSocketMessage{Event: EventDeviceOffline})
		handlers.Dispatch(&WebSocketMessage{Event: EventDeviceOffline})
	}()
	waitSignal(t, done, "Dispatch with handlers that register and clear")

	// The handler added during the first event sees the second one only.
	if late != 1 {
		t.Errorf("late handler ran %d times, want 1", late)
	}
	// Two offline handlers were registered by then; Clear removed them before
	// the second offline event.
	if offline != 2 {
		t.Errorf("offline handlers ran %d times, want 2", offline)
	}
}

func TestWebSocketCloseThenConnectLeavesOneLoop(t *testing.T) {
	first := newHeldConn(true)
	second := newHeldConn(false)
	ws := newTestWebSocket(newScriptedDialer(first, second))
	ctx := context.Background()

	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	oldLoop := listenAsync(ws)
	waitSignal(t, first.reading, "the first loop to read")

	if err := ws.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("second Connect() error = %v", err)
	}
	newLoop := listenAsync(ws)
	waitSignal(t, second.reading, "the second loop to read")

	// Only now does the first loop wake up, with the second connection live.
	first.unblock()
	if err := waitListen(t, oldLoop, "the first Listen to return"); err != nil {
		t.Errorf("first Listen() error = %v, want nil", err)
	}

	if readers, closed := second.state(); readers != 1 || closed {
		t.Errorf("second conn: readers = %d, closed = %v; want 1 reader and open", readers, closed)
	}
	if !ws.IsConnected() {
		t.Error("IsConnected() = false after the first loop exited, want true")
	}

	if err := ws.Close(); err != nil {
		t.Fatalf("final Close() error = %v", err)
	}
	if err := waitListen(t, newLoop, "the second Listen to return"); err != nil {
		t.Errorf("second Listen() error = %v, want nil", err)
	}
}

func TestWebSocketCloseDuringBackoffStopsListen(t *testing.T) {
	dialer := newScriptedDialer()
	ws := newTestWebSocket(dialer,
		WithReconnectInterval(time.Hour),
		WithMaxReconnectInterval(time.Hour),
	)

	loop := listenAsync(ws)
	waitSignal(t, dialer.dialed, "the failed dial")

	for range 2 {
		if err := ws.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}
	if err := waitListen(t, loop, "Listen to return after Close"); err != nil {
		t.Errorf("Listen() error = %v, want nil", err)
	}
}

func TestWebSocketConnectDuringBackoffKeepsStopSignal(t *testing.T) {
	dialer := newScriptedDialer()
	ws := newTestWebSocket(dialer,
		WithReconnectInterval(time.Hour),
		WithMaxReconnectInterval(time.Hour),
	)

	loop := listenAsync(ws)
	waitSignal(t, dialer.dialed, "the failed dial")

	conn := newHeldConn(false)
	dialer.add(conn)
	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if err := ws.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := waitListen(t, loop, "Listen to return after Close"); err != nil {
		t.Errorf("Listen() error = %v, want nil", err)
	}
	if _, closed := conn.state(); !closed {
		t.Error("Close() left the connection open")
	}
}

func TestWebSocketSecondListenIsRejected(t *testing.T) {
	conn := newHeldConn(false)
	ws := newTestWebSocket(newScriptedDialer(conn))

	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	loop := listenAsync(ws)
	waitSignal(t, conn.reading, "the first loop to read")

	err := waitListen(t, listenAsync(ws), "the second Listen to return")
	if !errors.Is(err, ErrWebSocketAlreadyListening) {
		t.Errorf("second Listen() error = %v, want ErrWebSocketAlreadyListening", err)
	}
	if readers, _ := conn.state(); readers != 1 {
		t.Errorf("readers = %d, want 1", readers)
	}

	if err := ws.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := waitListen(t, loop, "the first Listen to return"); err != nil {
		t.Errorf("first Listen() error = %v, want nil", err)
	}

	// The rejected call must not have left the WebSocket marked as busy.
	if err := waitListen(t, listenAsync(ws), "Listen after Close"); err != nil {
		t.Errorf("Listen() after Close error = %v, want nil", err)
	}
}

// reentrantDialer calls back into the WebSocket while dialing.
type reentrantDialer struct {
	ws   *WebSocket
	conn WebSocketConn
}

func (d *reentrantDialer) Dial(context.Context, string, http.Header) (WebSocketConn, error) {
	d.ws.IsConnected()
	d.ws.OnDeviceOnline(func(string) {})
	return d.conn, nil
}

func TestWebSocketDialerMayCallBack(t *testing.T) {
	dialer := &reentrantDialer{conn: newHeldConn(false)}
	ws := newTestWebSocket(dialer)
	dialer.ws = ws

	done := make(chan struct{})
	var err error
	go func() {
		defer close(done)
		err = ws.Connect(context.Background())
	}()
	waitSignal(t, done, "Connect with a dialer that calls back")

	if err != nil {
		t.Errorf("Connect() error = %v", err)
	}
	if closeErr := ws.Close(); closeErr != nil {
		t.Errorf("Close() error = %v", closeErr)
	}
}

// droppingDialer hands out connections that fail their first read, like a
// server that accepts a connection and drops it straight away.
type droppingDialer struct {
	dials sync.Mutex
	count int
}

func (d *droppingDialer) Dial(context.Context, string, http.Header) (WebSocketConn, error) {
	d.dials.Lock()
	d.count++
	d.dials.Unlock()

	conn := newHeldConn(false)
	conn.unblock()
	return conn, nil
}

func TestWebSocketListenBacksOffWhenConnectionsDrop(t *testing.T) {
	dialer := &droppingDialer{}
	ws := newTestWebSocket(dialer,
		WithReconnectInterval(50*time.Millisecond),
		WithMaxReconnectInterval(time.Hour))

	errCh := listenAsync(ws)
	time.Sleep(200 * time.Millisecond)
	if err := ws.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := waitListen(t, errCh, "Listen to stop"); err != nil {
		t.Errorf("Listen() error = %v, want nil", err)
	}

	// Waits of 50ms then 100ms allow three dials in 200ms; without a wait
	// there are thousands.
	dialer.dials.Lock()
	defer dialer.dials.Unlock()
	if dialer.count > 5 {
		t.Errorf("dialed %d times in 200ms, want the reconnects spaced out", dialer.count)
	}
}

// oneMessageConn delivers a single message and then fails, like a server that
// answers with an authentication error and hangs up. Its write methods touch
// unlocked state so the race detector notices overlapping writers.
type oneMessageConn struct {
	writes int
	read   bool
}

func (c *oneMessageConn) Close() error { return nil }

func (c *oneMessageConn) SetReadDeadline(time.Time) error { return nil }

func (c *oneMessageConn) ReadMessage() (int, []byte, error) {
	if !c.read {
		c.read = true
		return 1, []byte(`{"event":"Error"}`), nil
	}
	return 0, nil, ErrWebSocketClosed
}

func (c *oneMessageConn) SetWriteDeadline(time.Time) error {
	c.writes++
	return nil
}

func (c *oneMessageConn) WriteMessage(int, []byte) error {
	c.writes++
	return nil
}

// oneMessageDialer counts the connections it hands out.
type oneMessageDialer struct {
	dials sync.Mutex
	count int
}

func (d *oneMessageDialer) Dial(context.Context, string, http.Header) (WebSocketConn, error) {
	d.dials.Lock()
	d.count++
	d.dials.Unlock()
	return &oneMessageConn{}, nil
}

func TestWebSocketListenBacksOffWhenConnectionsDropAfterOneMessage(t *testing.T) {
	dialer := &oneMessageDialer{}
	ws := newTestWebSocket(dialer,
		WithReconnectInterval(50*time.Millisecond),
		WithMaxReconnectInterval(time.Hour))

	errCh := listenAsync(ws)
	time.Sleep(200 * time.Millisecond)
	if err := ws.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := waitListen(t, errCh, "Listen to stop"); err != nil {
		t.Errorf("Listen() error = %v, want nil", err)
	}

	// Each connection delivered a message, so every wait is the base 50ms:
	// about four dials in 200ms. Without a wait there are thousands.
	dialer.dials.Lock()
	defer dialer.dials.Unlock()
	if dialer.count > 8 {
		t.Errorf("dialed %d times in 200ms, want the reconnects spaced out", dialer.count)
	}
}

// TestWebSocketSendMessageWritesOneAtATime relies on the race detector: the
// connection's write methods mutate unlocked state.
func TestWebSocketSendMessageWritesOneAtATime(t *testing.T) {
	conn := &oneMessageConn{}
	ws := newTestWebSocket(newScriptedDialer(conn))
	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	const senders = 20
	var wg sync.WaitGroup
	for range senders {
		wg.Go(func() {
			if err := ws.SendMessage(context.Background(), map[string]string{"k": "v"}); err != nil {
				t.Errorf("SendMessage() error = %v", err)
			}
		})
	}
	wg.Wait()

	if conn.writes != 2*senders {
		t.Errorf("write calls = %d, want %d", conn.writes, 2*senders)
	}
}

// TestBrowserCallbackHandlerNeverBlocks repeats the OAuth redirect. A handler
// stuck on a full channel would keep the login server's Shutdown waiting.
func TestBrowserCallbackHandlerNeverBlocks(t *testing.T) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	handler := browserCallbackHandler(codeCh, errCh)

	served := make(chan struct{})
	go func() {
		defer close(served)
		for range 3 {
			handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/callback?code=abc", http.NoBody))
			handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/callback?error=denied", http.NoBody))
		}
	}()

	select {
	case <-served:
	case <-time.After(testTimeout):
		t.Fatal("callback handler blocked on a repeated request")
	}

	if code := <-codeCh; code != "abc" {
		t.Errorf("code = %q, want %q", code, "abc")
	}
	if err := <-errCh; err == nil {
		t.Error("expected the first callback error to be kept")
	}
}

// A caller queued behind other waiters must still return as soon as its own
// context ends.
func TestRateLimiterQueuedWaiterHonorsContext(t *testing.T) {
	limiter := newRateLimiter(1.0 / 3600)
	if err := limiter.wait(context.Background()); err != nil {
		t.Fatalf("first wait() error = %v", err)
	}

	// Takes the next slot, an hour away, and stays waiting for it.
	blocker, stopBlocker := context.WithCancel(context.Background())
	defer stopBlocker()
	blockerDone := make(chan error, 1)
	go func() { blockerDone <- limiter.wait(blocker) }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() { done <- limiter.wait(ctx) }()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("wait() error = %v, want context.Canceled", err)
		}
	case <-time.After(testTimeout):
		t.Fatal("a queued wait ignored its canceled context")
	}

	stopBlocker()
	if err := waitListen(t, blockerDone, "the first waiter to stop"); !errors.Is(err, context.Canceled) {
		t.Errorf("first waiter error = %v, want context.Canceled", err)
	}
}

// With an auto-selected port the caller can only open the login page if it is
// told the URL while BrowserLogin is still waiting, and the callback server
// has to be reachable by then.
func TestBrowserLoginReportsAuthorizeURLBeforeWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	urlCh := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		_, err := BrowserLogin(ctx, &BrowserLoginOptions{
			OnAuthorizeURL: func(authorizeURL string) { urlCh <- authorizeURL },
		})
		done <- err
	}()

	var authorizeURL string
	select {
	case authorizeURL = <-urlCh:
	case <-time.After(testTimeout):
		t.Fatal("OnAuthorizeURL was not called while the login was waiting")
	}

	parsed, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("authorize URL %q did not parse: %v", authorizeURL, err)
	}
	redirect := parsed.Query().Get("redirect_uri")
	if redirect == "" {
		t.Fatalf("authorize URL %q has no redirect_uri", authorizeURL)
	}

	// The error callback ends the login without needing the real token exchange.
	resp, err := http.Get(redirect + "?error=denied")
	if err != nil {
		t.Fatalf("callback server not reachable at %s: %v", redirect, err)
	}
	resp.Body.Close()

	if err := waitListen(t, done, "BrowserLogin to return"); err == nil {
		t.Error("BrowserLogin() error = nil after an error callback")
	}
}
