package discovery

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// MDNSService is the mDNS service type for Shelly devices.
const MDNSService = "_shelly._tcp.local."

// MDNSDiscoverer discovers Shelly devices via mDNS/Zeroconf.
//
// Gen2+ Shelly devices advertise themselves using mDNS with the
// _shelly._tcp.local service type. This discoverer listens for
// these advertisements and parses the TXT records to extract
// device information.
type MDNSDiscoverer struct {
	devices   map[string]*DiscoveredDevice
	devicesCh chan DiscoveredDevice
	stopCh    chan struct{}
	running   bool
	mu        sync.RWMutex
}

// NewMDNSDiscoverer creates a new mDNS discoverer.
func NewMDNSDiscoverer() *MDNSDiscoverer {
	return &MDNSDiscoverer{
		devices: make(map[string]*DiscoveredDevice),
	}
}

// Discover scans for devices for the specified duration.
func (m *MDNSDiscoverer) Discover(timeout time.Duration) ([]DiscoveredDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return m.DiscoverWithContext(ctx)
}

// DiscoverWithContext scans for devices until the context is canceled.
func (m *MDNSDiscoverer) DiscoverWithContext(ctx context.Context) ([]DiscoveredDevice, error) {
	// Start mDNS listener
	conn, err := m.createMulticastConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Send mDNS query for Shelly service
	if err := m.sendQuery(conn); err != nil {
		return nil, err
	}

	// Collect responses
	devices := make(map[string]*DiscoveredDevice)
	readCh := make(chan []byte, 100)

	// Start reader goroutine
	go func() {
		buf := make([]byte, 65536)
		for {
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			data := make([]byte, n)
			copy(data, buf[:n])
			select {
			case readCh <- data:
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

		case data := <-readCh:
			device := m.parseResponse(data)
			if device != nil && device.ID != "" {
				devices[device.ID] = device
			}
		}
	}
}

// createMulticastConn creates a UDP connection for mDNS multicast.
func (m *MDNSDiscoverer) createMulticastConn() (*net.UDPConn, error) {
	// Try to listen on a random port
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: 0})
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// sendQuery sends an mDNS query for the Shelly service.
func (m *MDNSDiscoverer) sendQuery(conn *net.UDPConn) error {
	// Build DNS query message
	// Header: ID=0, Flags=0 (standard query), QDCOUNT=1
	// Question: _shelly._tcp.local PTR IN
	query := m.buildDNSQuery(MDNSService, 12) // PTR = 12

	// Send to multicast address
	addr := &net.UDPAddr{
		IP:   net.ParseIP("224.0.0.251"),
		Port: 5353,
	}

	_, err := conn.WriteToUDP(query, addr)
	return err
}

// buildDNSQuery builds a DNS query message.
func (m *MDNSDiscoverer) buildDNSQuery(name string, qtype uint16) []byte {
	// Pre-allocate: 12 (header) + len(name) + 1 (labels) + 1 (null) + 4 (qtype+qclass)
	// Approximate with some extra space for label length bytes
	msg := make([]byte, 0, 12+len(name)+6)

	// Header (12 bytes): ID, Flags, QDCOUNT=1, ANCOUNT=0, NSCOUNT=0, ARCOUNT=0
	msg = append(msg, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0)

	// Question section
	// Name encoding
	for _, label := range strings.Split(strings.TrimSuffix(name, "."), ".") {
		msg = append(msg, byte(len(label)))
		msg = append(msg, []byte(label)...)
	}
	// Null terminator + QTYPE (2 bytes) + QCLASS IN=1 (2 bytes)
	msg = append(msg, 0, byte(qtype>>8), byte(qtype&0xFF), 0, 1)

	return msg
}

// parseResponse parses an mDNS response.
//
//nolint:gocyclo,cyclop // DNS response parsing requires checking multiple TXT record fields
func (m *MDNSDiscoverer) parseResponse(data []byte) *DiscoveredDevice {
	if len(data) < 12 {
		return nil
	}

	// Skip header
	// Check if it's a response (bit 15 of flags)
	if data[2]&0x80 == 0 {
		return nil // Not a response
	}

	// Parse answers
	// This is a simplified parser - a full implementation would
	// need complete DNS message parsing
	device := &DiscoveredDevice{
		Protocol: ProtocolMDNS,
		Port:     80,
		LastSeen: time.Now(),
	}

	// Look for TXT records containing device info
	// Shelly devices include: id, gen, model, fw, auth, app
	content := string(data)

	// Extract device ID
	if idx := strings.Index(content, "id="); idx != -1 {
		end := strings.IndexAny(content[idx+3:], " \x00\n")
		if end == -1 {
			end = len(content) - idx - 3
		}
		device.ID = content[idx+3 : idx+3+end]
	}

	// Extract model
	if idx := strings.Index(content, "model="); idx != -1 {
		end := strings.IndexAny(content[idx+6:], " \x00\n")
		if end == -1 {
			end = len(content) - idx - 6
		}
		device.Model = content[idx+6 : idx+6+end]
	}

	// Extract generation
	if idx := strings.Index(content, "gen="); idx != -1 {
		genStr := content[idx+4 : idx+5]
		switch genStr {
		case "1":
			device.Generation = types.Gen1
		case "2":
			device.Generation = types.Gen2
		case "3":
			device.Generation = types.Gen3
		default:
			device.Generation = types.Gen2
		}
	}

	// Extract firmware
	if idx := strings.Index(content, "fw="); idx != -1 {
		end := strings.IndexAny(content[idx+3:], " \x00\n")
		if end == -1 {
			end = len(content) - idx - 3
		}
		device.Firmware = content[idx+3 : idx+3+end]
	}

	// Extract auth
	if idx := strings.Index(content, "auth="); idx != -1 {
		device.AuthRequired = content[idx+5] == '1'
	}

	// Extract IP from source address in A record
	// This simplified parser looks for IPv4 addresses in the data
	device.Address = m.extractIP(data)

	if device.ID == "" || device.Address == nil {
		return nil
	}

	device.MACAddress = device.ID

	return device
}

// extractIP extracts an IPv4 address from DNS data.
func (m *MDNSDiscoverer) extractIP(data []byte) net.IP {
	// Look for A record data (4 consecutive bytes that look like an IP)
	for i := 12; i < len(data)-3; i++ {
		// Check if this looks like a valid IP (not 0.0.0.0, not 255.255.255.255)
		if data[i] > 0 && data[i] < 255 && data[i+3] > 0 && data[i+3] < 255 {
			ip := net.IPv4(data[i], data[i+1], data[i+2], data[i+3])
			// Basic sanity check - should be a private IP
			if ip.IsPrivate() || ip.IsLoopback() {
				return ip
			}
		}
	}
	return nil
}

// StartDiscovery begins continuous discovery.
func (m *MDNSDiscoverer) StartDiscovery() (<-chan DiscoveredDevice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return m.devicesCh, nil
	}

	m.devicesCh = make(chan DiscoveredDevice, 100)
	m.stopCh = make(chan struct{})
	m.running = true

	go m.continuousDiscovery()

	return m.devicesCh, nil
}

// continuousDiscovery runs continuous discovery.
func (m *MDNSDiscoverer) continuousDiscovery() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			devices, err := m.Discover(5 * time.Second)
			if err != nil {
				continue
			}
			for i := range devices {
				select {
				case m.devicesCh <- devices[i]:
				default:
				}
			}
		}
	}
}

// StopDiscovery stops continuous discovery.
func (m *MDNSDiscoverer) StopDiscovery() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	close(m.stopCh)
	m.running = false

	return nil
}

// Stop stops the discoverer and releases resources.
func (m *MDNSDiscoverer) Stop() error {
	return m.StopDiscovery()
}
