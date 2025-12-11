package integrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSConnector interface for WebSocket connections (allows mocking).
type WSConnector interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// Connection represents an active WebSocket connection to a Shelly cloud server.
type Connection struct {
	ws               WSConnector
	opts             *ConnectOptions
	closeCh          chan struct{}
	onStatusChange   func(*StatusChangeEvent)
	onSettingsChange func(*SettingsChangeEvent)
	onOnlineStatus   func(*OnlineStatusEvent)
	onError          func(error)
	onRawMessage     func(*WSMessage)
	host             string
	token            string
	mu               sync.RWMutex
	closed           bool
}

// newConnection creates and initializes a new WebSocket connection.
func newConnection(ctx context.Context, host, token string, opts *ConnectOptions) (*Connection, error) {
	conn := &Connection{
		host:    host,
		token:   token,
		opts:    opts,
		closeCh: make(chan struct{}),
	}

	// Build WebSocket URL: wss://{host}:6113/shelly/wss/hk_sock?t={token}
	wsURL := fmt.Sprintf("wss://%s:6113/shelly/wss/hk_sock?t=%s", host, token)

	// Configure dialer with timeout
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	// Dial the WebSocket connection
	ws, resp, err := dialer.DialContext(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	conn.ws = ws

	// Start the read loop
	go conn.readLoop()

	// Start ping loop for keepalive
	go conn.pingLoop()

	return conn, nil
}

// readLoop reads messages from the WebSocket connection.
func (c *Connection) readLoop() {
	for {
		select {
		case <-c.closeCh:
			return
		default:
		}

		c.mu.RLock()
		ws := c.ws
		c.mu.RUnlock()

		if ws == nil {
			return
		}

		_, message, err := ws.ReadMessage()
		if err != nil {
			c.mu.RLock()
			closed := c.closed
			onError := c.onError
			c.mu.RUnlock()

			if !closed && onError != nil {
				onError(fmt.Errorf("read error: %w", err))
			}
			return
		}

		c.handleMessage(message)
	}
}

// pingLoop sends periodic pings to keep the connection alive.
func (c *Connection) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.closeCh:
			return
		case <-ticker.C:
			c.mu.RLock()
			ws := c.ws
			c.mu.RUnlock()

			if ws == nil {
				return
			}

			// Type assert to *websocket.Conn for WriteControl
			if wsConn, ok := ws.(*websocket.Conn); ok {
				if err := wsConn.WriteControl(
					websocket.PingMessage,
					[]byte{},
					time.Now().Add(10*time.Second),
				); err != nil {
					return
				}
			}
		}
	}
}

// Host returns the connected host.
func (c *Connection) Host() string {
	return c.host
}

// IsClosed returns true if the connection is closed.
func (c *Connection) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// OnStatusChange registers a handler for device status change events.
func (c *Connection) OnStatusChange(handler func(*StatusChangeEvent)) {
	c.mu.Lock()
	c.onStatusChange = handler
	c.mu.Unlock()
}

// OnSettingsChange registers a handler for device settings change events.
func (c *Connection) OnSettingsChange(handler func(*SettingsChangeEvent)) {
	c.mu.Lock()
	c.onSettingsChange = handler
	c.mu.Unlock()
}

// OnOnlineStatus registers a handler for device online/offline events.
func (c *Connection) OnOnlineStatus(handler func(*OnlineStatusEvent)) {
	c.mu.Lock()
	c.onOnlineStatus = handler
	c.mu.Unlock()
}

// OnError registers a handler for connection errors.
func (c *Connection) OnError(handler func(error)) {
	c.mu.Lock()
	c.onError = handler
	c.mu.Unlock()
}

// OnRawMessage registers a handler for all raw messages.
func (c *Connection) OnRawMessage(handler func(*WSMessage)) {
	c.mu.Lock()
	c.onRawMessage = handler
	c.mu.Unlock()
}

// SendCommand sends a control command to a device.
func (c *Connection) SendCommand(ctx context.Context, deviceID, action string, params any) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrConnectionClosed
	}
	ws := c.ws
	c.mu.RUnlock()

	if ws == nil {
		return ErrConnectionClosed
	}

	cmd := DeviceCommand{
		Event:    "Integrator:ActionRequest",
		DeviceID: deviceID,
		Action:   action,
		Params:   params,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	// WebSocket text message type (1 = TextMessage in gorilla/websocket)
	const textMessage = 1
	if err := ws.WriteMessage(textMessage, data); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	return nil
}

// SendRelayCommand sends a relay on/off command.
func (c *Connection) SendRelayCommand(ctx context.Context, deviceID string, channel int, on bool) error {
	turn := "off"
	if on {
		turn = "on"
	}
	return c.SendCommand(ctx, deviceID, "relay", map[string]any{
		"id":   channel,
		"turn": turn,
	})
}

// SendRollerCommand sends a roller open/close/stop command.
func (c *Connection) SendRollerCommand(ctx context.Context, deviceID string, channel int, action string) error {
	return c.SendCommand(ctx, deviceID, "roller", map[string]any{
		"id": channel,
		"go": action,
	})
}

// SendRollerPosition sends a roller to a specific position (0-100).
func (c *Connection) SendRollerPosition(ctx context.Context, deviceID string, channel, position int) error {
	return c.SendCommand(ctx, deviceID, "roller", map[string]any{
		"id":         channel,
		"go":         "to_pos",
		"roller_pos": position,
	})
}

// SendLightCommand sends a light on/off command.
func (c *Connection) SendLightCommand(ctx context.Context, deviceID string, channel int, on bool) error {
	turn := "off"
	if on {
		turn = "on"
	}
	return c.SendCommand(ctx, deviceID, "light", map[string]any{
		"id":   channel,
		"turn": turn,
	})
}

// VerifyDevice sends a device verification request.
func (c *Connection) VerifyDevice(ctx context.Context, deviceID string) error {
	return c.SendCommand(ctx, deviceID, "DeviceVerify", nil)
}

// GetDeviceSettings requests device settings.
func (c *Connection) GetDeviceSettings(ctx context.Context, deviceID string) error {
	return c.SendCommand(ctx, deviceID, "DeviceGetSettings", nil)
}

// Close closes the WebSocket connection.
func (c *Connection) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	close(c.closeCh)
	ws := c.ws
	c.mu.Unlock()

	if ws != nil {
		return ws.Close()
	}
	return nil
}

// handleMessage processes an incoming WebSocket message.
func (c *Connection) handleMessage(data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.mu.RLock()
		if c.onError != nil {
			c.onError(fmt.Errorf("failed to parse message: %w", err))
		}
		c.mu.RUnlock()
		return
	}

	// Call raw message handler if registered
	c.mu.RLock()
	if c.onRawMessage != nil {
		c.onRawMessage(&msg)
	}
	c.mu.RUnlock()

	// Route to specific handlers based on event type
	switch EventType(msg.Event) {
	case EventStatusOnChange:
		c.handleStatusChange(&msg)
	case EventSettings:
		c.handleSettingsChange(&msg)
	case EventOnline:
		c.handleOnlineStatus(&msg)
	}
}

func (c *Connection) handleStatusChange(msg *WSMessage) {
	c.mu.RLock()
	handler := c.onStatusChange
	c.mu.RUnlock()

	if handler != nil {
		event := &StatusChangeEvent{
			DeviceID:  msg.GetDeviceID(),
			Status:    msg.Status,
			Timestamp: time.Unix(msg.Timestamp, 0),
		}
		handler(event)
	}
}

func (c *Connection) handleSettingsChange(msg *WSMessage) {
	c.mu.RLock()
	handler := c.onSettingsChange
	c.mu.RUnlock()

	if handler != nil {
		event := &SettingsChangeEvent{
			DeviceID:  msg.GetDeviceID(),
			Settings:  msg.Settings,
			Timestamp: time.Unix(msg.Timestamp, 0),
		}
		handler(event)
	}
}

func (c *Connection) handleOnlineStatus(msg *WSMessage) {
	c.mu.RLock()
	handler := c.onOnlineStatus
	c.mu.RUnlock()

	if handler != nil {
		event := &OnlineStatusEvent{
			DeviceID:  msg.GetDeviceID(),
			Online:    msg.IsOnline(),
			Timestamp: time.Unix(msg.Timestamp, 0),
		}
		handler(event)
	}
}
