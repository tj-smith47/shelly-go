package transport

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestNewCoAP(t *testing.T) {
	tests := []struct {
		name    string
		address string
		opts    []Option
	}{
		{
			name:    "basic CoAP",
			address: "192.168.1.100",
		},
		{
			name:    "multicast",
			address: "224.0.1.187",
			opts:    []Option{WithCoAPMulticast()},
		},
		{
			name:    "custom port",
			address: "192.168.1.100",
			opts:    []Option{WithCoAPPort(5684)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coap := NewCoAP(tt.address, tt.opts...)
			if coap == nil {
				t.Fatal("NewCoAP() returned nil")
			}
			if coap.address != tt.address {
				t.Errorf("address = %v, want %v", coap.address, tt.address)
			}
		})
	}
}

func TestCoAP_Call(t *testing.T) {
	coap := NewCoAP("192.168.1.100")
	_, err := coap.Call(context.Background(), NewSimpleRequest("Switch.Set"))
	if err == nil {
		t.Error("Call() error = nil, want not implemented error")
	}
}

func TestCoAP_Subscribe(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	handler := NotificationHandler(func(data json.RawMessage) {})
	err := coap.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() error = %v", err)
	}

	// Second subscribe should fail
	err = coap.Subscribe(handler)
	if err == nil {
		t.Error("second Subscribe() error = nil, want error")
	}
}

func TestCoAP_Unsubscribe(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	handler := NotificationHandler(func(data json.RawMessage) {})
	coap.Subscribe(handler)

	err := coap.Unsubscribe()
	if err != nil {
		t.Errorf("Unsubscribe() error = %v", err)
	}

	// Should be able to subscribe again
	err = coap.Subscribe(handler)
	if err != nil {
		t.Errorf("Subscribe() after Unsubscribe() error = %v", err)
	}
}

func TestCoAP_State(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	if coap.State() != StateDisconnected {
		t.Errorf("initial State() = %v, want %v", coap.State(), StateDisconnected)
	}
}

func TestCoAP_OnStateChange(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	called := false
	coap.OnStateChange(func(state ConnectionState) {
		called = true
	})

	coap.setState(StateConnected)

	if !called {
		t.Error("state change callback not called")
	}
}

func TestCoAP_Close(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	err := coap.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if coap.State() != StateClosed {
		t.Errorf("State() after Close() = %v, want %v", coap.State(), StateClosed)
	}

	// Calling Close again should be safe
	err = coap.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestCoAP_ClosedCall(t *testing.T) {
	coap := NewCoAP("192.168.1.100")
	coap.Close()

	_, err := coap.Call(context.Background(), NewSimpleRequest("Switch.Set"))
	if err == nil {
		t.Error("Call() on closed CoAP error = nil, want error")
	}
}

func TestCoAP_Address(t *testing.T) {
	coap := NewCoAP("192.168.1.100")
	if got := coap.Address(); got != "192.168.1.100" {
		t.Errorf("Address() = %v, want %v", got, "192.168.1.100")
	}
}

func TestCoAP_IsMulticast(t *testing.T) {
	// Non-multicast CoAP
	coap := NewCoAP("192.168.1.100")
	if coap.IsMulticast() {
		t.Error("IsMulticast() = true for unicast, want false")
	}

	// Multicast CoAP
	coap = NewCoAP("224.0.1.187", WithCoAPMulticast())
	if !coap.IsMulticast() {
		t.Error("IsMulticast() = false for multicast, want true")
	}
}

func TestCoAP_IsConnected(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	// Initially not connected (no conn set)
	if coap.IsConnected() {
		t.Error("IsConnected() = true initially, want false")
	}

	// After closing - still not connected
	coap.Close()
	if coap.IsConnected() {
		t.Error("IsConnected() = true after close, want false")
	}
}

func TestCoAP_ConnectClosed(t *testing.T) {
	coap := NewCoAP("192.168.1.100")
	coap.Close()

	err := coap.Connect(context.Background())
	if err == nil {
		t.Error("Connect() on closed CoAP should return error")
	}
}

func TestCoAP_ConnectEmptyAddressUnicast(t *testing.T) {
	coap := NewCoAP("") // Empty address without multicast
	err := coap.Connect(context.Background())
	if err == nil {
		t.Error("Connect() with empty address should return error")
	}
}

func TestCoAP_parseCoAPMessage(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "message too short",
			data:    []byte{0x40, 0x01},
			wantErr: true,
		},
		{
			name:    "invalid CoAP version",
			data:    []byte{0x80, 0x01, 0x00, 0x01}, // version 2
			wantErr: true,
		},
		{
			name:    "invalid token length",
			data:    []byte{0x4F, 0x01, 0x00, 0x01}, // TKL = 15 (invalid)
			wantErr: true,
		},
		{
			name:    "no options or payload after token",
			data:    []byte{0x44, 0x01, 0x00, 0x01, 0x01, 0x02, 0x03, 0x04}, // TKL=4, then end
			wantErr: true,
		},
		{
			name: "valid message with payload marker",
			data: append(
				[]byte{0x40, 0x01, 0x00, 0x01, 0xFF}, // Version 1, TKL=0, Code, MsgID, Payload marker
				[]byte(`{"id":"test"}`)...,
			),
			wantErr: false,
		},
		{
			name:    "valid message with option and no payload",
			data:    []byte{0x40, 0x01, 0x00, 0x01, 0x11, 0x00}, // Option followed by end
			wantErr: true,                                       // no payload marker
		},
		{
			name: "valid message with option delta 13",
			data: append(
				[]byte{0x40, 0x01, 0x00, 0x01, 0xD0, 0x00, 0xFF}, // Delta=13 extended
				[]byte(`{"id":"test"}`)...,
			),
			wantErr: false,
		},
		{
			name: "valid message with option delta 14",
			data: append(
				[]byte{0x40, 0x01, 0x00, 0x01, 0xE0, 0x00, 0x00, 0xFF}, // Delta=14 extended
				[]byte(`{"id":"test"}`)...,
			),
			wantErr: false,
		},
		{
			name: "valid message with option length 13",
			data: append(
				append([]byte{0x40, 0x01, 0x00, 0x01, 0x0D, 0x00}, make([]byte, 13)...),
				append([]byte{0xFF}, []byte(`{"id":"test"}`)...)...,
			),
			wantErr: false,
		},
		{
			name: "valid message with option length 14",
			data: append(
				append([]byte{0x40, 0x01, 0x00, 0x01, 0x0E, 0x00, 0x00}, make([]byte, 269)...),
				append([]byte{0xFF}, []byte(`{"id":"test"}`)...)...,
			),
			wantErr: false,
		},
		{
			name:    "truncated option delta 13",
			data:    []byte{0x40, 0x01, 0x00, 0x01, 0xD0}, // Delta=13 but no extended byte
			wantErr: true,
		},
		{
			name:    "truncated option delta 14",
			data:    []byte{0x40, 0x01, 0x00, 0x01, 0xE0, 0x00}, // Delta=14 but only 1 extended byte
			wantErr: true,
		},
		{
			name:    "truncated option length 13",
			data:    []byte{0x40, 0x01, 0x00, 0x01, 0x0D}, // Length=13 but no extended byte
			wantErr: true,
		},
		{
			name:    "truncated option length 14",
			data:    []byte{0x40, 0x01, 0x00, 0x01, 0x0E, 0x00}, // Length=14 but only 1 extended byte
			wantErr: true,
		},
		{
			name: "invalid JSON payload",
			data: append(
				[]byte{0x40, 0x01, 0x00, 0x01, 0xFF},
				[]byte(`not json`)...,
			),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := coap.parseCoAPMessage(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCoAPMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCoAP_handleMessage(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	var received json.RawMessage
	coap.Subscribe(func(data json.RawMessage) {
		received = data
	})

	// Build a valid CoAP message
	payload := []byte(`{"id":"shelly1pm-abc123","type":"SHSW-PM"}`)
	data := append([]byte{0x40, 0x01, 0x00, 0x01, 0xFF}, payload...)

	// Create mock UDP address
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	coap.handleMessage(data, addr)

	if received == nil {
		t.Error("handler not called")
	}
}

func TestCoAP_handleMessageNoHandler(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	// Build a valid CoAP message without setting handler
	payload := []byte(`{"id":"shelly1pm-abc123"}`)
	data := append([]byte{0x40, 0x01, 0x00, 0x01, 0xFF}, payload...)

	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	// Should not panic
	coap.handleMessage(data, addr)
}

func TestCoAP_handleMessageInvalidCoAP(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	called := false
	coap.Subscribe(func(data json.RawMessage) {
		called = true
	})

	// Invalid CoAP message
	coap.handleMessage([]byte{0x00, 0x01}, nil)

	if called {
		t.Error("handler should not be called for invalid message")
	}
}

func TestCoAP_isClosed(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	// Initially not closed
	if coap.isClosed() {
		t.Error("isClosed() = true, want false initially")
	}

	// After closing
	coap.Close()
	if !coap.isClosed() {
		t.Error("isClosed() = false after close, want true")
	}
}

// udpPair returns a local listener socket and the address to send to it.
func udpPair(t *testing.T) (*net.UDPConn, *net.UDPAddr) {
	t.Helper()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("ListenUDP() error = %v", err)
	}
	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		t.Fatalf("LocalAddr() = %T, want *net.UDPAddr", conn.LocalAddr())
	}
	return conn, addr
}

func TestCoAP_listenLoopStopsOnClose(t *testing.T) {
	coap := NewCoAP("127.0.0.1")
	conn, _ := udpPair(t)

	coap.connMu.Lock()
	coap.conn = conn
	coap.stopListen = make(chan struct{})
	stop := coap.stopListen
	coap.connMu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		coap.listenLoop(conn, stop)
	}()

	// Close runs while the loop is reading; the race detector fails this test
	// if the two touch the connection or stop channel without the lock.
	if err := coap.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("listenLoop kept running after Close")
	}
}

// The handler runs on the single listen goroutine, so notifications arrive
// one at a time and a handler is free to close the transport.
func TestCoAP_HandlerMayCloseTransport(t *testing.T) {
	coap := NewCoAP("127.0.0.1")
	conn, addr := udpPair(t)

	coap.connMu.Lock()
	coap.conn = conn
	coap.stopListen = make(chan struct{})
	stop := coap.stopListen
	coap.connMu.Unlock()

	closed := make(chan error, 1)
	if err := coap.Subscribe(func(json.RawMessage) { closed <- coap.Close() }); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	go coap.listenLoop(conn, stop)

	sender, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		t.Fatalf("DialUDP() error = %v", err)
	}
	defer sender.Close()
	frame := append([]byte{0x40, 0x02, 0x00, 0x01, 0xFF}, []byte(`{"id":"shelly1-abc"}`)...)
	if _, err := sender.Write(frame); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	select {
	case err := <-closed:
		if err != nil {
			t.Errorf("Close() from a handler error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("handler never ran or Close() deadlocked")
	}
}

func TestCoAP_MultipleStateCallbacks(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	calls := make([]ConnectionState, 0)

	coap.OnStateChange(func(state ConnectionState) {
		calls = append(calls, state)
	})
	coap.OnStateChange(func(state ConnectionState) {
		calls = append(calls, state)
	})

	coap.setState(StateConnecting)

	if len(calls) != 2 {
		t.Errorf("expected 2 callbacks, got %d", len(calls))
	}
}

func TestCoAP_CloseWithConnection(t *testing.T) {
	coap := NewCoAP("192.168.1.100")

	// Create a real UDP connection for testing close
	addr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Cannot resolve UDP address: %v", err)
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		t.Skipf("Cannot create UDP connection: %v", err)
	}

	// Set the connection
	coap.connMu.Lock()
	coap.conn = conn
	coap.connMu.Unlock()

	// Initialize stopListen
	coap.stopListen = make(chan struct{})

	// Close should properly close the connection
	err = coap.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if coap.State() != StateClosed {
		t.Errorf("State() after Close() = %v, want %v", coap.State(), StateClosed)
	}
}
