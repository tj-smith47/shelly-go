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

// Device field keys shared across mDNS TXT records, Gen1/Gen2 HTTP responses,
// and CoIoT status payloads.
const (
	fieldGen   = "gen"
	fieldApp   = "app"
	fieldVer   = "ver"
	fieldAuth  = "auth"
	fieldType  = "type"
	fieldModel = "model"
)

// MDNSDiscoverer discovers Shelly devices via mDNS/Zeroconf.
//
// Gen2+ Shelly devices advertise themselves using mDNS with the
// _shelly._tcp.local service type. This discoverer listens for
// these advertisements and parses the TXT records to extract
// device information.
type MDNSDiscoverer struct {
	devices         map[string]*DiscoveredDevice
	devicesCh       chan DiscoveredDevice
	stopCh          chan struct{}
	iface           *net.Interface
	tickInterval    time.Duration // injectable for tests; 0 → 10s production default
	discoverTimeout time.Duration // injectable for tests; 0 → 5s production default
	mu              sync.RWMutex
	running         bool
}

// MDNSOption configures an MDNSDiscoverer.
type MDNSOption func(*MDNSDiscoverer)

// WithMDNSInterface binds the discoverer's multicast listener to a specific
// network interface, so a multi-homed host receives announcements arriving on
// that interface's segment rather than only on the one the kernel selects by
// default. A nil interface keeps the default (kernel-selected) behavior.
func WithMDNSInterface(ifi *net.Interface) MDNSOption {
	return func(m *MDNSDiscoverer) { m.iface = ifi }
}

// NewMDNSDiscoverer creates a new mDNS discoverer.
func NewMDNSDiscoverer(opts ...MDNSOption) *MDNSDiscoverer {
	m := &MDNSDiscoverer{
		devices: make(map[string]*DiscoveredDevice),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
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
	// Join the mDNS multicast group to receive responses
	addr := &net.UDPAddr{
		IP:   net.ParseIP("224.0.0.251"),
		Port: 5353,
	}

	// Bind to the configured interface when set (multi-homed hosts), else let the
	// kernel pick the default multicast interface.
	conn, err := net.ListenMulticastUDP("udp4", m.iface, addr)
	if err != nil {
		// Fallback to regular UDP if multicast fails (e.g., permissions)
		// Some devices respond unicast to the query source, so this may work
		conn, err = net.ListenUDP("udp4", &net.UDPAddr{Port: 0})
		if err != nil {
			return nil, err
		}
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
		msg = append(msg, byte(len(label)&0xFF))
		msg = append(msg, []byte(label)...)
	}
	// Null terminator + QTYPE (2 bytes) + QCLASS IN=1 (2 bytes)
	msg = append(msg, 0, byte(qtype>>8), byte(qtype&0xFF), 0, 1)

	return msg
}

// DNS record types.
const (
	dnsTypeA   uint16 = 1
	dnsTypePTR uint16 = 12
	dnsTypeTXT uint16 = 16
	dnsTypeSRV uint16 = 33
)

// parseResponse parses an mDNS response using proper DNS message format.
func (m *MDNSDiscoverer) parseResponse(data []byte) *DiscoveredDevice {
	if len(data) < 12 {
		return nil
	}

	// Check if it's a response (bit 15 of flags)
	if data[2]&0x80 == 0 {
		return nil // Not a response
	}

	// Parse header
	ancount := int(data[6])<<8 | int(data[7])
	arcount := int(data[10])<<8 | int(data[11])

	if ancount == 0 && arcount == 0 {
		return nil
	}

	device := &DiscoveredDevice{
		Protocol:   ProtocolMDNS,
		Port:       80,
		Generation: types.Gen2, // Default for mDNS (Gen2+)
		LastSeen:   time.Now(),
	}

	// Skip header and questions, parse all resource records
	offset := 12

	// Skip questions section
	qdcount := int(data[4])<<8 | int(data[5])
	for i := 0; i < qdcount && offset < len(data); i++ {
		offset = m.skipName(data, offset)
		offset += 4 // QTYPE + QCLASS
	}

	// Parse answers and additional sections
	totalRecords := ancount + int(data[8])<<8 | int(data[9]) + arcount
	for i := 0; i < totalRecords && offset < len(data); i++ {
		newOffset := m.parseResourceRecord(data, offset, device)
		if newOffset <= offset {
			break // No progress, avoid infinite loop
		}
		offset = newOffset
	}

	// Validate we have required fields
	if device.ID == "" || device.Address == nil {
		return nil
	}

	device.MACAddress = device.ID
	return device
}

// parseResourceRecord parses a single DNS resource record and updates the device.
// Returns the new offset after the record.
func (m *MDNSDiscoverer) parseResourceRecord(data []byte, offset int, device *DiscoveredDevice) int {
	name, newOffset := m.parseName(data, offset)
	offset = newOffset

	if offset+10 > len(data) {
		return offset
	}

	rtype := uint16(data[offset])<<8 | uint16(data[offset+1])
	offset += 8 // TYPE + CLASS + TTL
	rdlength := int(data[offset])<<8 | int(data[offset+1])
	offset += 2

	if offset+rdlength > len(data) {
		return offset
	}

	rdata := data[offset : offset+rdlength]
	m.processRecord(data, offset, rtype, rdata, name, device)

	return offset + rdlength
}

// processRecord processes a parsed DNS record and updates the device accordingly.
func (m *MDNSDiscoverer) processRecord(
	data []byte, rdataOffset int, rtype uint16, rdata []byte, name string, device *DiscoveredDevice,
) {
	switch rtype {
	case dnsTypePTR:
		// PTR record contains instance name like "ShellyPlus1-A8032ABCA8D8._shelly._tcp.local."
		instanceName, _ := m.parseName(data, rdataOffset)
		if device.ID == "" {
			device.ID = m.extractDeviceID(instanceName)
		}

	case dnsTypeTXT:
		// TXT records contain key=value pairs with length prefixes
		m.parseTXTRecord(rdata, device)

	case dnsTypeA:
		// A record contains IPv4 address (4 bytes); guard requires >= 4 so the
		// static analyser can prove indices 0-3 are in-bounds.
		if len(rdata) >= 4 {
			ip := net.IPv4(rdata[0], rdata[1], rdata[2], rdata[3])
			if ip.IsPrivate() || ip.IsLoopback() {
				device.Address = ip
				// Extract device ID from A record name if not yet found
				if device.ID == "" {
					device.ID = m.extractDeviceID(name)
				}
			}
		}

	case dnsTypeSRV:
		// SRV record contains port and target
		if len(rdata) >= 6 {
			device.Port = int(rdata[4])<<8 | int(rdata[5])
		}
	}
}

// extractDeviceID extracts the device ID from an mDNS instance name.
// Example: "ShellyPlus1-A8032ABCA8D8._shelly._tcp.local." -> "ShellyPlus1-A8032ABCA8D8"
func (m *MDNSDiscoverer) extractDeviceID(name string) string {
	// Remove trailing dot
	name = strings.TrimSuffix(name, ".")

	// Find the service part and extract everything before it
	if idx := strings.Index(name, "._shelly._tcp"); idx > 0 {
		return name[:idx]
	}
	if idx := strings.Index(name, "._http._tcp"); idx > 0 {
		return name[:idx]
	}

	// For A records, the name might be "ShellyPlus1-A8032ABCA8D8.local"
	if idx := strings.Index(name, ".local"); idx > 0 {
		return name[:idx]
	}

	return ""
}

// parseTXTRecord parses a DNS TXT record with length-prefixed strings.
func (m *MDNSDiscoverer) parseTXTRecord(rdata []byte, device *DiscoveredDevice) {
	offset := 0
	for offset < len(rdata) {
		length := int(rdata[offset])
		offset++
		if offset+length > len(rdata) {
			break
		}

		kv := string(rdata[offset : offset+length])
		offset += length

		// Parse key=value
		if eqIdx := strings.Index(kv, "="); eqIdx > 0 {
			key := kv[:eqIdx]
			value := kv[eqIdx+1:]

			switch key {
			case fieldGen:
				switch value {
				case "1":
					device.Generation = types.Gen1
				case "2":
					device.Generation = types.Gen2
				case "3":
					device.Generation = types.Gen3
				}
			case fieldApp, fieldModel:
				if device.Model == "" {
					device.Model = value
				}
			case fieldVer, "fw":
				if device.Firmware == "" {
					device.Firmware = value
				}
			case fieldAuth:
				device.AuthRequired = value == "1"
			}
		}
	}
}

// parseName parses a DNS name from the message, handling compression pointers.
func (m *MDNSDiscoverer) parseName(data []byte, offset int) (name string, nextOffset int) {
	var sb strings.Builder
	jumped := false
	originalOffset := offset

	for offset < len(data) {
		length := int(data[offset])

		if length == 0 {
			if !jumped {
				originalOffset = offset + 1
			}
			break
		}

		// Check for compression pointer (top 2 bits set)
		if length&0xC0 == 0xC0 {
			if offset+1 >= len(data) {
				break
			}
			pointer := (length&0x3F)<<8 | int(data[offset+1])
			if !jumped {
				originalOffset = offset + 2
				jumped = true
			}
			offset = pointer
			continue
		}

		offset++
		if offset+length > len(data) {
			break
		}

		if sb.Len() > 0 {
			sb.WriteByte('.')
		}
		sb.Write(data[offset : offset+length])
		offset += length
	}

	return sb.String(), originalOffset
}

// skipName skips over a DNS name in the message.
func (m *MDNSDiscoverer) skipName(data []byte, offset int) int {
	for offset < len(data) {
		length := int(data[offset])
		if length == 0 {
			return offset + 1
		}
		if length&0xC0 == 0xC0 {
			return offset + 2 // Compression pointer
		}
		offset += 1 + length
	}
	return offset
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
	interval := m.tickInterval
	if interval == 0 {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			discTimeout := m.discoverTimeout
			if discTimeout == 0 {
				discTimeout = 5 * time.Second
			}
			devices, err := m.Discover(discTimeout)
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
