package gen1

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	// CoIoTMulticastAddr is the CoIoT multicast address.
	CoIoTMulticastAddr = "224.0.1.187"

	// CoIoTPort is the default CoIoT port.
	CoIoTPort = 5683

	// DefaultCoIoTPeriod is the default status update period in seconds.
	DefaultCoIoTPeriod = 15
)

// CoIoTListener listens for CoIoT (CoAP) status updates from Gen1 devices.
//
// Gen1 devices broadcast status updates via CoAP multicast. This listener
// receives those updates and calls registered handlers.
//
// CoIoT Protocol:
//   - Uses CoAP (Constrained Application Protocol) over UDP
//   - Multicast address: 224.0.1.187:5683
//   - Devices publish status periodically (default 15s)
//   - Updates include device ID, type, and current status
//
// Example:
//
//	listener := gen1.NewCoIoTListener()
//	listener.OnStatus(func(deviceID string, status *gen1.CoIoTStatus) {
//	    fmt.Printf("Device %s: %+v\n", deviceID, status)
//	})
//	if err := listener.Start(); err != nil {
//	    log.Fatal(err)
//	}
//	defer listener.Stop()
type CoIoTListener struct {
	conn          *net.UDPConn
	stopCh        chan struct{}
	multicastAddr string
	handlers      []StatusHandler
	port          int
	bufferSize    int
	mu            sync.RWMutex
	running       bool
}

// StatusHandler is called when a status update is received.
type StatusHandler func(deviceID string, status *CoIoTStatus)

// CoIoTStatus contains status data from a CoIoT message.
type CoIoTStatus struct {
	Timestamp  time.Time      `json:"ts,omitempty"`
	Sensors    map[string]any `json:"sensors,omitempty"`
	Actuators  map[string]any `json:"actuators,omitempty"`
	DeviceID   string         `json:"id,omitempty"`
	DeviceType string         `json:"type,omitempty"`
	Raw        []byte         `json:"-"`
	Generation int            `json:"gen,omitempty"`
}

// CoIoTOption configures the CoIoT listener.
type CoIoTOption func(*CoIoTListener)

// WithCoIoTMulticastAddr sets the multicast address.
func WithCoIoTMulticastAddr(addr string) CoIoTOption {
	return func(l *CoIoTListener) {
		l.multicastAddr = addr
	}
}

// WithCoIoTPort sets the listening port.
func WithCoIoTPort(port int) CoIoTOption {
	return func(l *CoIoTListener) {
		l.port = port
	}
}

// WithCoIoTBufferSize sets the receive buffer size.
func WithCoIoTBufferSize(size int) CoIoTOption {
	return func(l *CoIoTListener) {
		l.bufferSize = size
	}
}

// NewCoIoTListener creates a new CoIoT status listener.
//
// Options can be provided to customize the listener configuration.
func NewCoIoTListener(opts ...CoIoTOption) *CoIoTListener {
	l := &CoIoTListener{
		multicastAddr: CoIoTMulticastAddr,
		port:          CoIoTPort,
		bufferSize:    1500, // Standard MTU
		handlers:      make([]StatusHandler, 0),
		stopCh:        make(chan struct{}),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// OnStatus registers a handler for status updates.
//
// Multiple handlers can be registered and will all be called
// when a status update is received.
func (l *CoIoTListener) OnStatus(handler StatusHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers = append(l.handlers, handler)
}

// Start begins listening for CoIoT messages.
//
// This starts a background goroutine that listens for multicast
// messages and dispatches them to registered handlers.
func (l *CoIoTListener) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.running {
		return fmt.Errorf("listener already running")
	}

	// Resolve multicast address
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", l.multicastAddr, l.port))
	if err != nil {
		return fmt.Errorf("failed to resolve multicast address: %w", err)
	}

	// Join multicast group
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		return fmt.Errorf("failed to join multicast group: %w", err)
	}

	// Set receive buffer size
	if err := conn.SetReadBuffer(l.bufferSize); err != nil {
		conn.Close()
		return fmt.Errorf("failed to set read buffer: %w", err)
	}

	l.conn = conn
	l.running = true
	l.stopCh = make(chan struct{})

	// Start receive loop
	go l.receiveLoop()

	return nil
}

// Stop stops listening for CoIoT messages.
func (l *CoIoTListener) Stop() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.running {
		return nil
	}

	close(l.stopCh)
	l.running = false

	if l.conn != nil {
		return l.conn.Close()
	}

	return nil
}

// IsRunning returns whether the listener is running.
func (l *CoIoTListener) IsRunning() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.running
}

// receiveLoop listens for incoming CoAP messages.
func (l *CoIoTListener) receiveLoop() {
	buf := make([]byte, l.bufferSize)

	for {
		select {
		case <-l.stopCh:
			return
		default:
			// Set read deadline to allow periodic stop checks
			//nolint:errcheck // SetReadDeadline errors are non-fatal for listener
			l.conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			n, _, err := l.conn.ReadFromUDP(buf)
			if err != nil {
				// Timeout is expected, continue
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				// Check if stopped
				select {
				case <-l.stopCh:
					return
				default:
					// Log error and continue
					continue
				}
			}

			if n > 0 {
				// Parse and dispatch message
				go l.handleMessage(buf[:n])
			}
		}
	}
}

// handleMessage parses a CoAP message and dispatches to handlers.
func (l *CoIoTListener) handleMessage(data []byte) {
	status, err := l.parseCoAPMessage(data)
	if err != nil {
		// Invalid message, ignore
		return
	}

	// Get handlers
	l.mu.RLock()
	handlers := make([]StatusHandler, len(l.handlers))
	copy(handlers, l.handlers)
	l.mu.RUnlock()

	// Dispatch to handlers
	for _, handler := range handlers {
		handler(status.DeviceID, status)
	}
}

// parseCoAPMessage parses a CoAP message.
//
// CoIoT uses a simplified CoAP format with JSON or CBOR payload.
// This is a simplified implementation that handles common cases.
func (l *CoIoTListener) parseCoAPMessage(data []byte) (*CoIoTStatus, error) {
	// CoAP header format:
	// - Version (2 bits), Type (2 bits), Token Length (4 bits)
	// - Code (8 bits)
	// - Message ID (16 bits)
	// - Token (0-8 bytes)
	// - Options (variable)
	// - Payload marker (0xFF)
	// - Payload

	if len(data) < 4 {
		return nil, fmt.Errorf("message too short")
	}

	// Find payload marker (0xFF)
	payloadStart := -1
	for i := 4; i < len(data); i++ {
		if data[i] == 0xFF {
			payloadStart = i + 1
			break
		}
	}

	if payloadStart == -1 || payloadStart >= len(data) {
		return nil, fmt.Errorf("no payload found")
	}

	payload := data[payloadStart:]

	// Try to parse as JSON
	status := &CoIoTStatus{
		Timestamp: time.Now(),
		Raw:       data,
		Sensors:   make(map[string]any),
		Actuators: make(map[string]any),
	}

	// CoIoT status messages typically have a specific structure
	// Try parsing as JSON first
	var jsonPayload map[string]any
	//nolint:nestif // JSON parsing with optional fields requires nested type assertions
	if err := json.Unmarshal(payload, &jsonPayload); err == nil {
		// Extract device info from JSON
		if id, ok := jsonPayload["id"].(string); ok {
			status.DeviceID = id
		}
		if devType, ok := jsonPayload["type"].(string); ok {
			status.DeviceType = devType
		}
		if sensors, ok := jsonPayload["G"].([]any); ok {
			// Parse sensor groups
			for _, s := range sensors {
				if sArr, ok := s.([]any); ok && len(sArr) >= 3 {
					// Format: [channel, id, value]
					key := fmt.Sprintf("%v_%v", sArr[0], sArr[1])
					status.Sensors[key] = sArr[2]
				}
			}
		}
	} else {
		// May be CBOR encoded, store raw for now
		// Full CBOR parsing would require a CBOR library
		status.DeviceID = "unknown"
	}

	return status, nil
}

// ParseCoIoTDescription parses a CoIoT device description.
//
// Device description is obtained from /cit/d endpoint and describes
// available sensors and actuators.
type CoIoTDescription struct {
	// DeviceID is the device identifier.
	DeviceID string `json:"id"`

	// DeviceType is the device type.
	DeviceType string `json:"type"`

	// Blocks contains component blocks.
	Blocks []CoIoTBlock `json:"blk"`

	// Sensors contains sensor definitions.
	Sensors []CoIoTSensor `json:"sen"`
}

// CoIoTBlock represents a component block in CoIoT description.
type CoIoTBlock struct {
	Description string `json:"D"`
	ID          int    `json:"I"`
}

// CoIoTSensor represents a sensor in CoIoT description.
type CoIoTSensor struct {
	Type        string `json:"T"`
	Description string `json:"D"`
	Unit        string `json:"U,omitempty"`
	Links       []int  `json:"L,omitempty"`
	ID          int    `json:"I"`
	Block       int    `json:"B,omitempty"`
}

// GetDeviceDescription retrieves the CoIoT device description.
//
// This is typically called via HTTP on /cit/d endpoint, not via multicast.
func GetDeviceDescription(deviceAddr string) (*CoIoTDescription, error) {
	// This would typically be called via HTTP
	// For now, return an error indicating to use HTTP
	return nil, fmt.Errorf("use HTTP transport to get device description from /cit/d")
}
