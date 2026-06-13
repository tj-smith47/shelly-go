package gen1

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"
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

	// CoAP option IDs.
	optionURIPath     = 11   // URI-Path option
	optionGlobalDevID = 3332 // CoIoT Global Device ID option

	// CoAP codes.
	codeStatus = 30 // CoIoT STATUS code (0.30)
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
//   - Device ID is in CoAP option 3332 (GlobalDevID) with format: DeviceType#DeviceID#Version
//   - Payload contains sensor data in JSON format: {"G": [[channel, id, value], ...]}
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
	listenFn      func(addr *net.UDPAddr) (*net.UDPConn, error)
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
	Timestamp   time.Time      `json:"ts,omitempty"`
	Sensors     map[string]any `json:"sensors,omitempty"`
	Actuators   map[string]any `json:"actuators,omitempty"`
	SourceAddr  string         `json:"source,omitempty"`
	DeviceID    string         `json:"id,omitempty"`
	DeviceType  string         `json:"type,omitempty"`
	DeviceMAC   string         `json:"mac,omitempty"`
	URIPath     string         `json:"uri_path,omitempty"`
	Raw         []byte         `json:"-"`
	Generation  int            `json:"gen,omitempty"`
	Version     int            `json:"version,omitempty"`
	Serial      int            `json:"serial,omitempty"`
	CoAPCode    int            `json:"coap_code,omitempty"`
	CoAPType    int            `json:"coap_type,omitempty"`
	ValidityRaw int            `json:"validity,omitempty"`
	MessageID   uint16         `json:"message_id,omitempty"`
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

	// Join multicast group. Tests inject listenFn to bypass the real multicast
	// bind (which requires a multicast NIC and a fixed port) so that the receive
	// loop can be exercised over a local UDP socket.
	listenFn := l.listenFn
	if listenFn == nil {
		listenFn = func(a *net.UDPAddr) (*net.UDPConn, error) {
			return net.ListenMulticastUDP("udp4", nil, a)
		}
	}
	conn, err := listenFn(addr)
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
			if err := l.conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
				// Non-fatal, continue
				continue
			}

			n, srcAddr, err := l.conn.ReadFromUDP(buf)
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
				// Make a copy of the data for async processing
				data := make([]byte, n)
				copy(data, buf[:n])

				// Parse and dispatch message
				var sourceIP string
				if srcAddr != nil {
					sourceIP = srcAddr.IP.String()
				}
				go l.handleMessage(data, sourceIP)
			}
		}
	}
}

// handleMessage parses a CoAP message and dispatches to handlers.
func (l *CoIoTListener) handleMessage(data []byte, sourceAddr string) {
	status, err := ParseCoAPMessage(data, sourceAddr)
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

// parseExtendedValue parses CoAP extended delta/length encoding.
// Returns the value and new offset, or error if truncated.
func parseExtendedValue(nibble int, data []byte, offset int) (value, newOffset int, err error) {
	switch nibble {
	case 13:
		if offset >= len(data) {
			return 0, offset, fmt.Errorf("truncated extended value")
		}
		return int(data[offset]) + 13, offset + 1, nil
	case 14:
		if offset+1 >= len(data) {
			return 0, offset, fmt.Errorf("truncated extended value")
		}
		return int(binary.BigEndian.Uint16(data[offset:offset+2])) + 269, offset + 2, nil
	case 15:
		return -1, offset, nil // End marker or reserved
	default:
		return nibble, offset, nil
	}
}

// parseCoAPOptions parses CoAP options from the message.
// Returns URI path parts and updates status with device ID info.
func parseCoAPOptions(data []byte, offset int, status *CoIoTStatus) (uriParts []string, finalOffset int, err error) {
	var uriPathParts []string
	currentOptionNumber := 0

	for offset < len(data) {
		// Check for payload marker
		if data[offset] == 0xFF {
			offset++
			break
		}

		optionByte := data[offset]
		offset++

		deltaNibble := int(optionByte >> 4)
		lengthNibble := int(optionByte & 0x0F)

		// Parse extended delta
		delta, nextOffset, parseErr := parseExtendedValue(deltaNibble, data, offset)
		if parseErr != nil {
			return uriPathParts, offset, fmt.Errorf("truncated extended delta")
		}
		if delta < 0 {
			break // End of options marker
		}
		offset = nextOffset

		// Parse extended length
		length, nextOffset, parseErr := parseExtendedValue(lengthNibble, data, offset)
		if parseErr != nil {
			return uriPathParts, offset, fmt.Errorf("truncated extended length")
		}
		if length < 0 {
			return uriPathParts, offset, fmt.Errorf("invalid option length")
		}
		offset = nextOffset

		// Calculate option number
		currentOptionNumber += delta

		// Extract option value
		if offset+length > len(data) {
			return uriPathParts, offset, fmt.Errorf("option value exceeds message length")
		}
		optionValue := data[offset : offset+length]
		offset += length

		// Process known options
		switch currentOptionNumber {
		case optionURIPath:
			uriPathParts = append(uriPathParts, string(optionValue))
		case optionGlobalDevID:
			parseGlobalDevID(string(optionValue), status)
		}
	}

	return uriPathParts, offset, nil
}

// ParseCoAPMessage parses a CoAP message and extracts CoIoT status.
//
// CoAP message format:
//   - Header: Version (2 bits), Type (2 bits), Token Length (4 bits), Code (8 bits), Message ID (16 bits)
//   - Token: 0-8 bytes (length from header)
//   - Options: Variable length, delta-encoded
//   - Payload marker: 0xFF
//   - Payload: JSON encoded status data
//
// The device ID is extracted from option 3332 (GlobalDevID) in format: DeviceType#DeviceID#Version
func ParseCoAPMessage(data []byte, sourceAddr string) (*CoIoTStatus, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("message too short: %d bytes", len(data))
	}

	status := &CoIoTStatus{
		Timestamp:  time.Now(),
		Raw:        data,
		Sensors:    make(map[string]any),
		Actuators:  make(map[string]any),
		Generation: 1,
		SourceAddr: sourceAddr,
	}

	// Parse CoAP header
	tokenLen := int(data[0] & 0x0F)
	status.CoAPType = int((data[0] >> 4) & 0x03)
	status.CoAPCode = int(data[1])
	status.MessageID = binary.BigEndian.Uint16(data[2:4])

	if tokenLen > 8 {
		return nil, fmt.Errorf("invalid token length: %d", tokenLen)
	}

	offset := 4 + tokenLen
	if offset > len(data) {
		return nil, fmt.Errorf("message too short for token")
	}

	// Parse options
	uriPathParts, offset, err := parseCoAPOptions(data, offset, status)
	if err != nil {
		return nil, err
	}

	// Build URI path
	if len(uriPathParts) > 0 {
		status.URIPath = "/" + strings.Join(uriPathParts, "/")
	}

	// Parse payload if present
	if offset < len(data) {
		parseCoIoTPayload(data[offset:], status)
	}

	return status, nil
}

// parseGlobalDevID parses the GlobalDevID option value.
// Format: DeviceType#DeviceID#Version (e.g., "SHSW-PM#C45BBE6C2D3A#2")
func parseGlobalDevID(value string, status *CoIoTStatus) {
	parts := strings.Split(value, "#")
	if len(parts) >= 1 {
		status.DeviceType = parts[0]
	}
	if len(parts) >= 2 {
		status.DeviceMAC = parts[1]
		// Device ID is "devicetype-mac" format for consistency with Shelly naming
		status.DeviceID = strings.ToLower(status.DeviceType) + "-" + strings.ToLower(parts[1])
	}
	if len(parts) >= 3 {
		var version int
		if _, err := fmt.Sscanf(parts[2], "%d", &version); err == nil {
			status.Version = version
		}
	}
}

// parseCoIoTPayload parses the JSON payload from a CoIoT message.
func parseCoIoTPayload(payload []byte, status *CoIoTStatus) {
	var jsonPayload map[string]any
	if err := json.Unmarshal(payload, &jsonPayload); err != nil {
		// Invalid JSON, keep raw data
		return
	}

	// Parse sensor groups: "G": [[channel, id, value], ...]
	if sensors, ok := jsonPayload["G"].([]any); ok {
		for _, s := range sensors {
			if sArr, ok := s.([]any); ok && len(sArr) >= 3 {
				// Format: [channel, id, value]
				key := fmt.Sprintf("%v_%v", sArr[0], sArr[1])
				status.Sensors[key] = sArr[2]
			}
		}
	}

	// Parse validity period if present
	if validity, ok := jsonPayload["V"].(float64); ok {
		status.ValidityRaw = int(validity)
	}

	// Parse serial if present
	if serial, ok := jsonPayload["S"].(float64); ok {
		status.Serial = int(serial)
	}
}

// CoIoTDescription contains device description from /cit/d endpoint.
type CoIoTDescription struct {
	// DeviceID is the device identifier (e.g., "shellyem-AABBCC").
	DeviceID string `json:"id"`

	// DeviceType is the device type code (e.g., "SHEM").
	DeviceType string `json:"type"`

	// Blocks contains component blocks.
	Blocks []CoIoTBlock `json:"blk"`

	// Sensors contains sensor definitions.
	Sensors []CoIoTSensor `json:"sen"`

	// Actuators contains actuator definitions (optional).
	Actuators []CoIoTActuator `json:"act,omitempty"`
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
	Range       string `json:"R,omitempty"`
	Links       []int  `json:"L,omitempty"`
	ID          int    `json:"I"`
	Block       int    `json:"B,omitempty"`
}

// CoIoTActuator represents an actuator in CoIoT description.
type CoIoTActuator struct {
	Type        string `json:"T"`
	Description string `json:"D"`
	Range       string `json:"R,omitempty"`
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

// ParseCoIoTDescription parses a CoIoT device description JSON.
func ParseCoIoTDescription(data []byte) (*CoIoTDescription, error) {
	var desc CoIoTDescription
	if err := json.Unmarshal(data, &desc); err != nil {
		return nil, fmt.Errorf("failed to parse CoIoT description: %w", err)
	}
	return &desc, nil
}

// SensorIDToDescription maps common sensor IDs to human-readable descriptions.
var SensorIDToDescription = map[int]string{
	1101: "Relay State",
	2101: "Input State",
	2102: "Input Event",
	2103: "Input Event Count",
	3101: "Active Power",
	3104: "Voltage",
	3105: "Current",
	3106: "Power Factor",
	3107: "Frequency",
	3108: "Apparent Power",
	3109: "Reactive Power",
	3110: "Energy Counter 0",
	3111: "Energy Counter 1",
	3112: "Energy Counter 2",
	3117: "Total Returned Energy",
	4101: "Power (W)",
	4102: "Energy (Wmin)",
	4103: "Energy Counter (Wmin)",
	4104: "Lamp Life",
	4105: "Lamp Life (Pct)",
	5101: "Flood Detected",
	5102: "Motion Detected",
	5103: "Vibration Detected",
	6101: "Overpower",
	6102: "Over Temperature",
	6103: "Overload",
	6104: "Voltage Error",
	6105: "Under Voltage",
	6106: "Over Voltage",
	6107: "Firmware Update Available",
	6108: "Cloud Status",
	6109: "Status (Error)",
	6110: "Errors",
	9101: "Temperature",
	9102: "Humidity",
	9103: "Battery",
	9104: "External Temperature",
	9105: "External Humidity",
}
