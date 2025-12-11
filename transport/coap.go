package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	// CoIoT multicast address for Shelly devices.
	coiotMulticastAddr = "224.0.1.187:5683"
	// Default CoAP/CoIoT port.
	defaultCoAPPort = 5683
	// CoAP message buffer size.
	coapBufferSize = 8192
)

// CoAP is a CoAP/CoIoT transport for Gen1 Shelly devices.
// Supports real-time status updates via CoAP multicast.
//
// The CoAP transport provides:
//   - UDP multicast listening for device discovery
//   - Unicast communication with individual devices
//   - CoIoT protocol message parsing
//   - Real-time status notifications
//
// Note: CoAP is primarily used for receiving status updates from Gen1 devices.
// For control commands, use the HTTP transport with REST API.
type CoAP struct {
	opts           *options
	conn           *net.UDPConn
	notifyHandler  NotificationHandler
	stopListen     chan struct{}
	address        string
	stateCallbacks []func(ConnectionState)
	state          ConnectionState
	mu             sync.RWMutex
	notifyMu       sync.RWMutex
	stateMu        sync.RWMutex
	connMu         sync.Mutex
	closed         bool
}

// CoIoTMessage represents a CoIoT status message from a Gen1 device.
type CoIoTMessage struct {
	DeviceID   string          `json:"id,omitempty"`
	DeviceType string          `json:"type,omitempty"`
	Status     json.RawMessage `json:"G,omitempty"`
	Serial     int             `json:"serial,omitempty"`
}

// NewCoAP creates a new CoAP transport.
//
// For unicast mode, address should be the device address (e.g., "192.168.1.100").
// For multicast mode, use WithCoAPMulticast() option to join the CoIoT multicast group.
//
// Example:
//
//	// Unicast to specific device
//	coap := NewCoAP("192.168.1.100")
//
//	// Multicast listening for all devices
//	coap := NewCoAP("", WithCoAPMulticast())
func NewCoAP(address string, opts ...Option) *CoAP {
	options := defaultOptions()
	applyOptions(options, opts)

	return &CoAP{
		address:        address,
		opts:           options,
		state:          StateDisconnected,
		stateCallbacks: make([]func(ConnectionState), 0),
	}
}

// Connect starts the CoAP listener.
// For multicast mode, this joins the CoIoT multicast group.
// For unicast mode, this prepares for communication with the specified device.
func (c *CoAP) Connect(ctx context.Context) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.isClosed() {
		return fmt.Errorf("CoAP transport is closed")
	}

	if c.conn != nil {
		return nil // already connected
	}

	c.setState(StateConnecting)

	var conn *net.UDPConn
	var err error

	if c.opts.coapMulticast {
		// Multicast mode - listen for CoIoT broadcasts
		conn, err = c.startMulticastListener()
	} else {
		// Unicast mode - communicate with specific device
		conn, err = c.startUnicastConnection()
	}

	if err != nil {
		c.setState(StateDisconnected)
		return err
	}

	c.conn = conn
	c.stopListen = make(chan struct{})
	c.setState(StateConnected)

	// Start listener
	go c.listenLoop()

	return nil
}

// startMulticastListener joins the CoIoT multicast group.
func (c *CoAP) startMulticastListener() (*net.UDPConn, error) {
	// Resolve multicast address
	addr, err := net.ResolveUDPAddr("udp4", coiotMulticastAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve multicast addr: %w", err)
	}

	// Listen on all interfaces
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("listen multicast: %w", err)
	}

	// Set read buffer size
	if err := conn.SetReadBuffer(coapBufferSize); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set read buffer: %w", err)
	}

	return conn, nil
}

// startUnicastConnection creates a UDP connection for unicast communication.
func (c *CoAP) startUnicastConnection() (*net.UDPConn, error) {
	if c.address == "" {
		return nil, fmt.Errorf("address required for unicast mode")
	}

	// Resolve device address
	port := c.opts.coapPort
	if port == 0 {
		port = defaultCoAPPort
	}
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", c.address, port))
	if err != nil {
		return nil, fmt.Errorf("resolve addr: %w", err)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("dial udp: %w", err)
	}

	return conn, nil
}

// listenLoop reads messages from the UDP connection.
func (c *CoAP) listenLoop() {
	buf := make([]byte, coapBufferSize)

	for {
		select {
		case <-c.stopListen:
			return
		default:
		}

		c.connMu.Lock()
		conn := c.conn
		c.connMu.Unlock()

		if conn == nil {
			return
		}

		// Set read deadline for non-blocking check of stop channel
		if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			continue
		}

		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // timeout, check stop channel
			}
			if c.isClosed() {
				return
			}
			continue
		}

		// Process message
		c.handleMessage(buf[:n], remoteAddr)
	}
}

// handleMessage processes an incoming CoAP/CoIoT message.
func (c *CoAP) handleMessage(data []byte, addr *net.UDPAddr) {
	// Parse CoAP message
	msg, err := c.parseCoAPMessage(data)
	if err != nil {
		return
	}

	c.notifyMu.RLock()
	handler := c.notifyHandler
	c.notifyMu.RUnlock()

	if handler != nil {
		// Include source address in message
		notification := struct {
			*CoIoTMessage
			SourceAddr string `json:"source_addr,omitempty"`
		}{
			CoIoTMessage: msg,
			SourceAddr:   addr.IP.String(),
		}

		jsonData, err := json.Marshal(notification)
		if err != nil {
			return
		}
		handler(jsonData)
	}
}

// parseCoAPMessage parses a raw CoAP message into a CoIoTMessage.
// CoIoT messages have a CoAP header followed by JSON payload.
//
//nolint:gocyclo,cyclop // CoAP message parsing requires handling multiple message structure variations
func (c *CoAP) parseCoAPMessage(data []byte) (*CoIoTMessage, error) {
	// CoAP message structure:
	// - 4 byte header: Ver(2) + Type(2) + TKL(4) + Code(8) + MessageID(16)
	// - Token (TKL bytes)
	// - Options (variable)
	// - Payload marker (0xFF) + Payload

	if len(data) < 4 {
		return nil, fmt.Errorf("message too short")
	}

	// Parse header
	header := data[0]
	version := (header >> 6) & 0x03
	if version != 1 {
		return nil, fmt.Errorf("unsupported CoAP version: %d", version)
	}

	tkl := int(header & 0x0f)
	if tkl > 8 {
		return nil, fmt.Errorf("invalid token length: %d", tkl)
	}

	// Skip header + token
	offset := 4 + tkl
	if offset >= len(data) {
		return nil, fmt.Errorf("no options or payload")
	}

	// Skip options until payload marker (0xFF)
	for offset < len(data) {
		if data[offset] == 0xFF {
			offset++ // skip payload marker
			break
		}

		// Parse option delta and length
		delta := int((data[offset] >> 4) & 0x0f)
		length := int(data[offset] & 0x0f)
		offset++

		// Handle extended delta (we only need to skip the bytes, not use delta value)
		switch delta {
		case 13:
			if offset >= len(data) {
				return nil, fmt.Errorf("truncated option delta")
			}
			offset++
		case 14:
			if offset+1 >= len(data) {
				return nil, fmt.Errorf("truncated option delta")
			}
			offset += 2
		}

		// Handle extended length
		switch length {
		case 13:
			if offset >= len(data) {
				return nil, fmt.Errorf("truncated option length")
			}
			length = int(data[offset]) + 13
			offset++
		case 14:
			if offset+1 >= len(data) {
				return nil, fmt.Errorf("truncated option length")
			}
			length = int(data[offset])<<8 + int(data[offset+1]) + 269
			offset += 2
		}

		// Skip option value
		offset += length
	}

	if offset >= len(data) {
		return nil, fmt.Errorf("no payload")
	}

	// Parse JSON payload
	payload := data[offset:]
	var msg CoIoTMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	return &msg, nil
}

// Call executes a method call via CoAP.
//
// Note: CoAP for Gen1 devices is primarily used for receiving status updates.
// For control commands, use the HTTP transport with REST API.
// This method sends a CoAP request and waits for a response.
func (c *CoAP) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if c.isClosed() {
		return nil, fmt.Errorf("CoAP transport is closed")
	}

	// Auto-connect if not connected
	if c.conn == nil {
		if err := c.Connect(ctx); err != nil {
			return nil, err
		}
	}

	// For Gen1 devices, CoAP is primarily for listening
	// Control should use HTTP REST API
	return nil, fmt.Errorf("CoAP Call not supported for Gen1 devices - use HTTP transport for control commands")
}

// Subscribe registers a handler for incoming CoIoT notifications.
// This is the primary use case for CoAP with Gen1 devices.
func (c *CoAP) Subscribe(handler NotificationHandler) error {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()

	if c.notifyHandler != nil {
		return errHandlerAlreadyRegistered
	}

	c.notifyHandler = handler
	return nil
}

// Unsubscribe removes the notification handler.
func (c *CoAP) Unsubscribe() error {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	c.notifyHandler = nil
	return nil
}

// State returns the current connection state.
func (c *CoAP) State() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// OnStateChange registers a callback for connection state changes.
func (c *CoAP) OnStateChange(callback func(ConnectionState)) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.stateCallbacks = append(c.stateCallbacks, callback)
}

// setState updates the connection state and notifies callbacks.
func (c *CoAP) setState(state ConnectionState) {
	c.mu.Lock()
	c.state = state
	c.mu.Unlock()

	c.stateMu.RLock()
	callbacks := make([]func(ConnectionState), len(c.stateCallbacks))
	copy(callbacks, c.stateCallbacks)
	c.stateMu.RUnlock()

	for _, cb := range callbacks {
		cb(state)
	}
}

// isClosed returns true if the transport is closed.
func (c *CoAP) isClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// Close closes the CoAP transport.
func (c *CoAP) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	// Stop listener (only if it was started)
	if c.stopListen != nil {
		select {
		case <-c.stopListen:
		default:
			close(c.stopListen)
		}
	}

	// Close connection
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	c.setState(StateClosed)
	return nil
}

// Address returns the address this transport is configured for.
func (c *CoAP) Address() string {
	return c.address
}

// IsMulticast returns true if the transport is in multicast mode.
func (c *CoAP) IsMulticast() bool {
	return c.opts.coapMulticast
}

// IsConnected returns true if the transport is connected.
func (c *CoAP) IsConnected() bool {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.conn != nil
}
