package discovery

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// CoIoT multicast address and port.
const (
	CoIoTMulticastAddr = "224.0.1.187"
	CoIoTPort          = 5683
)

// CoIoTDiscoverer discovers Gen1 Shelly devices via CoAP/CoIoT multicast.
//
// Gen1 Shelly devices broadcast their status periodically via CoAP
// multicast to 224.0.1.187:5683. This discoverer listens for these
// broadcasts and extracts device information.
type CoIoTDiscoverer struct {
	devices   map[string]*DiscoveredDevice
	devicesCh chan DiscoveredDevice
	conn      *net.UDPConn
	stopCh    chan struct{}
	running   bool
	mu        sync.RWMutex
}

// NewCoIoTDiscoverer creates a new CoIoT discoverer.
func NewCoIoTDiscoverer() *CoIoTDiscoverer {
	return &CoIoTDiscoverer{
		devices: make(map[string]*DiscoveredDevice),
	}
}

// Discover scans for devices for the specified duration.
func (c *CoIoTDiscoverer) Discover(timeout time.Duration) ([]DiscoveredDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.DiscoverWithContext(ctx)
}

// DiscoverWithContext scans for devices until the context is canceled.
func (c *CoIoTDiscoverer) DiscoverWithContext(ctx context.Context) ([]DiscoveredDevice, error) {
	// Create multicast listener
	conn, err := c.createMulticastConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Collect responses
	devices := make(map[string]*DiscoveredDevice)
	readCh := make(chan coiotMessage, 100)

	// Start reader goroutine
	go func() {
		buf := make([]byte, 65536)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			select {
			case readCh <- coiotMessage{data: buf[:n], addr: addr}:
			default:
			}
		}
	}()

	// Process responses until timeout
	for {
		select {
		case <-ctx.Done():
			// Convert map to slice
			result := make([]DiscoveredDevice, 0, len(devices))
			for _, d := range devices {
				result = append(result, *d)
			}
			return result, nil

		case msg := <-readCh:
			device := c.parseCoAPMessage(msg.data, msg.addr)
			if device != nil && device.ID != "" {
				devices[device.ID] = device
			}
		}
	}
}

type coiotMessage struct {
	addr *net.UDPAddr
	data []byte
}

// createMulticastConn creates a UDP connection for CoIoT multicast.
func (c *CoIoTDiscoverer) createMulticastConn() (*net.UDPConn, error) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(CoIoTMulticastAddr),
		Port: CoIoTPort,
	}

	// Listen on all interfaces
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		// Fallback to regular UDP if multicast fails
		conn, err = net.ListenUDP("udp4", &net.UDPAddr{Port: CoIoTPort})
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}

// parseCoAPMessage parses a CoAP message from a Gen1 device.
func (c *CoIoTDiscoverer) parseCoAPMessage(data []byte, addr *net.UDPAddr) *DiscoveredDevice {
	if len(data) < 4 {
		return nil
	}

	// CoAP header
	// Version (2 bits) | Type (2 bits) | Token Length (4 bits) | Code (8 bits) | Message ID (16 bits)
	version := (data[0] >> 6) & 0x03
	if version != 1 {
		return nil // Not CoAP v1
	}

	tokenLen := data[0] & 0x0F
	if len(data) < 4+int(tokenLen) {
		return nil
	}

	// Skip header and token, look for payload
	offset := 4 + int(tokenLen)

	// Skip options (until 0xFF marker or end)
	offset = c.skipCoAPOptions(data, offset)

	// Skip payload marker
	if offset < len(data) && data[offset] == 0xFF {
		offset++
	}

	if offset >= len(data) {
		return nil
	}

	// Parse JSON payload
	payload := data[offset:]
	return c.parsePayload(payload, addr)
}

// skipCoAPOptions skips CoAP options and returns the new offset.
func (c *CoIoTDiscoverer) skipCoAPOptions(data []byte, offset int) int {
	for offset < len(data) && data[offset] != 0xFF {
		if data[offset] == 0 {
			offset++
			continue
		}

		// Option delta and length
		delta := (data[offset] >> 4) & 0x0F
		length := data[offset] & 0x0F

		switch delta {
		case 13:
			offset += 2
		case 14:
			offset += 3
		default:
			offset++
		}

		switch length {
		case 13:
			offset++
			if offset < len(data) {
				length = data[offset-1] + 13
			}
		case 14:
			offset += 2
		}

		offset += int(length)
	}
	return offset
}

// parsePayload parses the JSON payload from a CoIoT message.
func (c *CoIoTDiscoverer) parsePayload(payload []byte, addr *net.UDPAddr) *DiscoveredDevice {
	// Try to parse as JSON
	var status map[string]any
	if err := json.Unmarshal(payload, &status); err != nil {
		// Try to extract from raw data
		return c.extractDeviceFromRaw(string(payload), addr)
	}

	device := &DiscoveredDevice{
		Protocol:   ProtocolCoIoT,
		Port:       80,
		Generation: types.Gen1,
		Address:    addr.IP,
		LastSeen:   time.Now(),
		Raw:        status,
	}

	// Extract device info from status
	if id, ok := status["id"].(string); ok {
		device.ID = id
	}

	if mac, ok := status["mac"].(string); ok {
		device.MACAddress = mac
		if device.ID == "" {
			device.ID = mac
		}
	}

	if model, ok := status["type"].(string); ok {
		device.Model = model
	}

	if fw, ok := status["fw_ver"].(string); ok {
		device.Firmware = fw
	}

	// Check for auth in settings
	if settings, ok := status["settings"].(map[string]any); ok {
		if auth, ok := settings["device"].(map[string]any); ok {
			if name, ok := auth["name"].(string); ok {
				device.Name = name
			}
		}
	}

	return device
}

// extractDeviceFromRaw extracts device info from raw payload.
func (c *CoIoTDiscoverer) extractDeviceFromRaw(payload string, addr *net.UDPAddr) *DiscoveredDevice {
	device := &DiscoveredDevice{
		Protocol:   ProtocolCoIoT,
		Port:       80,
		Generation: types.Gen1,
		Address:    addr.IP,
		LastSeen:   time.Now(),
	}

	// Try to extract MAC address (common format: XX:XX:XX:XX:XX:XX)
	for i := 0; i < len(payload)-17; i++ {
		if payload[i] == ':' && i > 1 &&
			i+17 < len(payload) &&
			payload[i+3] == ':' &&
			payload[i+6] == ':' &&
			payload[i+9] == ':' &&
			payload[i+12] == ':' {
			// Potential MAC address
			mac := payload[i-2 : i+15]
			if isValidMAC(mac) {
				device.MACAddress = mac
				device.ID = strings.ReplaceAll(mac, ":", "")
				break
			}
		}
	}

	return device
}

// isValidMAC checks if a string is a valid MAC address.
func isValidMAC(s string) bool {
	if len(s) != 17 {
		return false
	}
	for i, c := range s {
		if i%3 == 2 {
			if c != ':' {
				return false
			}
		} else {
			if !isHexDigit(byte(c)) {
				return false
			}
		}
	}
	return true
}

// isHexDigit checks if a byte is a hex digit.
func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// StartDiscovery begins continuous discovery.
func (c *CoIoTDiscoverer) StartDiscovery() (<-chan DiscoveredDevice, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return c.devicesCh, nil
	}

	conn, err := c.createMulticastConn()
	if err != nil {
		return nil, err
	}

	c.conn = conn
	c.devicesCh = make(chan DiscoveredDevice, 100)
	c.stopCh = make(chan struct{})
	c.running = true

	go c.continuousDiscovery()

	return c.devicesCh, nil
}

// continuousDiscovery runs continuous discovery.
func (c *CoIoTDiscoverer) continuousDiscovery() {
	buf := make([]byte, 65536)

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		//nolint:errcheck // SetReadDeadline errors are non-fatal for discovery
		c.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		device := c.parseCoAPMessage(buf[:n], addr)
		if device != nil && device.ID != "" {
			select {
			case c.devicesCh <- *device:
			default:
			}
		}
	}
}

// StopDiscovery stops continuous discovery.
func (c *CoIoTDiscoverer) StopDiscovery() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	close(c.stopCh)
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.running = false

	return nil
}

// Stop stops the discoverer and releases resources.
func (c *CoIoTDiscoverer) Stop() error {
	return c.StopDiscovery()
}
