package discovery

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// buildMDNSResponse constructs a proper mDNS response message for testing.
// This creates a realistic DNS response with PTR, TXT, and A records.
func buildMDNSResponse(instanceName string, txtRecords map[string]string, ip [4]byte) []byte {
	msg := make([]byte, 0, 512)

	// Header (12 bytes)
	// ID=0, Flags=0x8400 (response, authoritative), QDCOUNT=0, ANCOUNT=1, NSCOUNT=0, ARCOUNT=2
	msg = append(msg, 0, 0, 0x84, 0x00, 0, 0, 0, 1, 0, 0, 0, 2)

	// Answer section: PTR record pointing to instance name
	// Name: _shelly._tcp.local (encoded)
	msg = append(msg, 7, '_', 's', 'h', 'e', 'l', 'l', 'y')
	msg = append(msg, 4, '_', 't', 'c', 'p')
	msg = append(msg, 5, 'l', 'o', 'c', 'a', 'l')
	msg = append(msg, 0) // null terminator

	// Type: PTR (12), Class: IN (1), TTL: 4500
	msg = append(msg, 0, 12, 0, 1, 0, 0, 0x11, 0x94)

	// RDLENGTH and RDATA (instance name)
	instanceParts := strings.Split(instanceName, ".")
	rdataStart := len(msg)
	msg = append(msg, 0, 0) // placeholder for RDLENGTH

	for _, part := range instanceParts {
		if part == "" {
			continue
		}
		msg = append(msg, byte(len(part)))
		msg = append(msg, []byte(part)...)
	}
	msg = append(msg, 0) // null terminator

	rdataLen := len(msg) - rdataStart - 2
	msg[rdataStart] = byte(rdataLen >> 8)
	msg[rdataStart+1] = byte(rdataLen & 0xFF)

	// Additional section: TXT record
	// Name: instance name (use compression pointer to save space, or encode again)
	for _, part := range instanceParts {
		if part == "" {
			continue
		}
		msg = append(msg, byte(len(part)))
		msg = append(msg, []byte(part)...)
	}
	msg = append(msg, 0)

	// Type: TXT (16), Class: IN (1), TTL: 4500
	msg = append(msg, 0, 16, 0, 1, 0, 0, 0x11, 0x94)

	// Build TXT RDATA with length-prefixed strings
	txtRdataStart := len(msg)
	msg = append(msg, 0, 0) // placeholder for RDLENGTH

	for key, value := range txtRecords {
		kv := key + "=" + value
		msg = append(msg, byte(len(kv)))
		msg = append(msg, []byte(kv)...)
	}

	txtRdataLen := len(msg) - txtRdataStart - 2
	msg[txtRdataStart] = byte(txtRdataLen >> 8)
	msg[txtRdataStart+1] = byte(txtRdataLen & 0xFF)

	// Additional section: A record
	// Name: hostname.local
	hostname := strings.Split(instanceName, ".")[0]
	msg = append(msg, byte(len(hostname)))
	msg = append(msg, []byte(hostname)...)
	msg = append(msg, 5, 'l', 'o', 'c', 'a', 'l')
	msg = append(msg, 0)

	// Type: A (1), Class: IN (1), TTL: 120
	msg = append(msg, 0, 1, 0, 1, 0, 0, 0, 120)

	// RDLENGTH: 4, RDATA: IP address
	msg = append(msg, 0, 4)
	msg = append(msg, ip[0], ip[1], ip[2], ip[3])

	return msg
}

func TestNewMDNSDiscoverer(t *testing.T) {
	d := NewMDNSDiscoverer()
	if d == nil {
		t.Fatal("NewMDNSDiscoverer() returned nil")
	}

	if d.devices == nil {
		t.Error("devices map should be initialized")
	}
}

func TestWithMDNSInterface(t *testing.T) {
	ifi := &net.Interface{Index: 7, Name: "ifbind-mdns"}
	d := NewMDNSDiscoverer(WithMDNSInterface(ifi))
	if d.iface != ifi {
		t.Fatalf("WithMDNSInterface did not set iface: got %v, want %v", d.iface, ifi)
	}

	// No option leaves the kernel-default (nil) interface.
	def := NewMDNSDiscoverer()
	if def.iface != nil {
		t.Errorf("default iface should be nil, got %v", def.iface)
	}
}

func TestMDNSDiscoverer_BuildDNSQuery(t *testing.T) {
	d := NewMDNSDiscoverer()

	query := d.buildDNSQuery(MDNSService, 12) // PTR = 12

	// Verify header
	if len(query) < 12 {
		t.Fatalf("query too short: %d bytes", len(query))
	}

	// ID should be 0 for mDNS
	if query[0] != 0 || query[1] != 0 {
		t.Error("ID should be 0")
	}

	// Flags should be 0 for standard query
	if query[2] != 0 || query[3] != 0 {
		t.Error("Flags should be 0")
	}

	// QDCOUNT should be 1
	if query[4] != 0 || query[5] != 1 {
		t.Error("QDCOUNT should be 1")
	}

	// ANCOUNT, NSCOUNT, ARCOUNT should be 0
	for i := 6; i < 12; i++ {
		if query[i] != 0 {
			t.Errorf("byte %d should be 0", i)
		}
	}
}

func TestMDNSDiscoverer_ParseResponse_NotResponse(t *testing.T) {
	d := NewMDNSDiscoverer()

	// Query packet (bit 15 of flags is 0)
	data := make([]byte, 20)
	data[2] = 0x00 // Not a response

	device := d.parseResponse(data)
	if device != nil {
		t.Error("should return nil for query packets")
	}
}

func TestMDNSDiscoverer_ParseResponse_TooShort(t *testing.T) {
	d := NewMDNSDiscoverer()

	data := make([]byte, 10) // Too short
	device := d.parseResponse(data)
	if device != nil {
		t.Error("should return nil for short packets")
	}
}

func TestMDNSDiscoverer_ParseResponse_ValidResponse(t *testing.T) {
	d := NewMDNSDiscoverer()

	txtRecords := map[string]string{
		"gen":  "2",
		"app":  "SNSW-001P16EU",
		"ver":  "1.0.0",
		"auth": "0",
	}
	data := buildMDNSResponse(
		"ShellyPlus1-ABC123._shelly._tcp.local",
		txtRecords,
		[4]byte{192, 168, 1, 100},
	)

	device := d.parseResponse(data)
	if device == nil {
		t.Fatal("should parse valid response")
	}

	if device.ID != "ShellyPlus1-ABC123" {
		t.Errorf("ID = %v, want 'ShellyPlus1-ABC123'", device.ID)
	}

	if device.Model != "SNSW-001P16EU" {
		t.Errorf("Model = %v, want 'SNSW-001P16EU'", device.Model)
	}

	if device.Generation != types.Gen2 {
		t.Errorf("Generation = %v, want Gen2", device.Generation)
	}

	if device.Protocol != ProtocolMDNS {
		t.Errorf("Protocol = %v, want 'mdns'", device.Protocol)
	}

	if device.Address.String() != "192.168.1.100" {
		t.Errorf("Address = %v, want '192.168.1.100'", device.Address)
	}
}

func TestMDNSDiscoverer_ParseResponse_Gen3Device(t *testing.T) {
	d := NewMDNSDiscoverer()

	txtRecords := map[string]string{
		"gen":  "3",
		"app":  "S3SW-001P16EU",
		"auth": "1",
	}
	data := buildMDNSResponse(
		"Shelly1Gen3-XYZ789._shelly._tcp.local",
		txtRecords,
		[4]byte{10, 0, 0, 50},
	)

	device := d.parseResponse(data)
	if device == nil {
		t.Fatal("should parse valid response")
	}

	if device.Generation != types.Gen3 {
		t.Errorf("Generation = %v, want Gen3", device.Generation)
	}

	if !device.AuthRequired {
		t.Error("AuthRequired should be true")
	}
}

func TestMDNSDiscoverer_ParseResponse_Gen1Device(t *testing.T) {
	d := NewMDNSDiscoverer()

	txtRecords := map[string]string{
		"gen": "1",
	}
	data := buildMDNSResponse(
		"Shelly1-ABC._shelly._tcp.local",
		txtRecords,
		[4]byte{172, 16, 0, 10},
	)

	device := d.parseResponse(data)
	if device == nil {
		t.Fatal("should parse valid response")
	}

	if device.Generation != types.Gen1 {
		t.Errorf("Generation = %v, want Gen1", device.Generation)
	}
}

func TestMDNSDiscoverer_ParseResponse_NoRecords(t *testing.T) {
	d := NewMDNSDiscoverer()

	// Response with no answer or additional records
	data := make([]byte, 12)
	data[2] = 0x80 // Response flag

	device := d.parseResponse(data)
	if device != nil {
		t.Error("should return nil when no records present")
	}
}

func TestMDNSDiscoverer_ExtractDeviceID(t *testing.T) {
	d := NewMDNSDiscoverer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "shelly service",
			input:    "ShellyPlus1-ABC123._shelly._tcp.local.",
			expected: "ShellyPlus1-ABC123",
		},
		{
			name:     "http service",
			input:    "ShellyPlus1-ABC123._http._tcp.local.",
			expected: "ShellyPlus1-ABC123",
		},
		{
			name:     "local hostname",
			input:    "ShellyPlus1-ABC123.local.",
			expected: "ShellyPlus1-ABC123",
		},
		{
			name:     "no trailing dot",
			input:    "ShellyPlus1-ABC123._shelly._tcp.local",
			expected: "ShellyPlus1-ABC123",
		},
		{
			name:     "unknown format",
			input:    "randomname",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.extractDeviceID(tt.input)
			if result != tt.expected {
				t.Errorf("extractDeviceID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMDNSDiscoverer_ParseTXTRecord(t *testing.T) {
	d := NewMDNSDiscoverer()

	// Build proper TXT record RDATA with length-prefixed strings
	// Each string is prefixed by its length in bytes
	rdata := []byte{
		5, 'g', 'e', 'n', '=', '2', // "gen=2" = 5 chars
		11, 'a', 'p', 'p', '=', 'P', 'l', 'u', 's', '1', 'P', 'M', // "app=Plus1PM" = 11 chars
		9, 'v', 'e', 'r', '=', '1', '.', '0', '.', '0', // "ver=1.0.0" = 9 chars
		6, 'a', 'u', 't', 'h', '=', '1', // "auth=1" = 6 chars
	}

	device := &DiscoveredDevice{}
	d.parseTXTRecord(rdata, device)

	if device.Generation != types.Gen2 {
		t.Errorf("Generation = %v, want Gen2", device.Generation)
	}

	if device.Model != "Plus1PM" {
		t.Errorf("Model = %v, want 'Plus1PM'", device.Model)
	}

	if device.Firmware != "1.0.0" {
		t.Errorf("Firmware = %v, want '1.0.0'", device.Firmware)
	}

	if !device.AuthRequired {
		t.Error("AuthRequired should be true")
	}
}

func TestMDNSDiscoverer_ParseName(t *testing.T) {
	d := NewMDNSDiscoverer()

	// Build DNS name: "shelly.local"
	data := []byte{
		6, 's', 'h', 'e', 'l', 'l', 'y',
		5, 'l', 'o', 'c', 'a', 'l',
		0,
	}

	name, offset := d.parseName(data, 0)
	if name != "shelly.local" {
		t.Errorf("parseName() = %q, want 'shelly.local'", name)
	}
	if offset != len(data) {
		t.Errorf("offset = %d, want %d", offset, len(data))
	}
}

func TestMDNSDiscoverer_ParseName_Compression(t *testing.T) {
	d := NewMDNSDiscoverer()

	// Build data with compression pointer
	// "local" at offset 0, then compression pointer at offset 7
	data := []byte{
		5, 'l', 'o', 'c', 'a', 'l', 0, // offset 0-6
		6, 's', 'h', 'e', 'l', 'l', 'y', // offset 7-13
		0xC0, 0x00, // compression pointer to offset 0
	}

	// Parse starting at "shelly" (offset 7)
	name, offset := d.parseName(data, 7)
	if name != "shelly.local" {
		t.Errorf("parseName() with compression = %q, want 'shelly.local'", name)
	}
	if offset != 16 { // Should be after the compression pointer
		t.Errorf("offset = %d, want 16", offset)
	}
}

func TestMDNSDiscoverer_SkipName(t *testing.T) {
	d := NewMDNSDiscoverer()

	// Regular name
	data := []byte{
		6, 's', 'h', 'e', 'l', 'l', 'y',
		5, 'l', 'o', 'c', 'a', 'l',
		0,
	}

	offset := d.skipName(data, 0)
	if offset != len(data) {
		t.Errorf("skipName() = %d, want %d", offset, len(data))
	}

	// Compression pointer
	data2 := []byte{6, 's', 'h', 'e', 'l', 'l', 'y', 0xC0, 0x00}
	offset = d.skipName(data2, 0)
	if offset != 9 { // Stop after compression pointer
		t.Errorf("skipName() with compression = %d, want 9", offset)
	}
}

func TestMDNSDiscoverer_StartStopDiscovery(t *testing.T) {
	if os.Getenv("SHELLY_CI") == "1" {
		t.Skip("skipping in CI - requires multicast networking")
	}

	d := NewMDNSDiscoverer()

	// Start discovery
	ch, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("StartDiscovery() error = %v", err)
	}

	if ch == nil {
		t.Error("should return channel")
	}

	if !d.running {
		t.Error("should be running")
	}

	// Starting again should return same channel
	ch2, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("StartDiscovery() second call error = %v", err)
	}

	if ch2 != ch {
		t.Error("should return same channel on second call")
	}

	// Stop discovery
	err = d.StopDiscovery()
	if err != nil {
		t.Fatalf("StopDiscovery() error = %v", err)
	}

	if d.running {
		t.Error("should not be running")
	}

	// Stopping again should be no-op
	err = d.StopDiscovery()
	if err != nil {
		t.Fatalf("StopDiscovery() second call error = %v", err)
	}
}

func TestMDNSDiscoverer_Stop(t *testing.T) {
	if os.Getenv("SHELLY_CI") == "1" {
		t.Skip("skipping in CI - requires multicast networking")
	}

	d := NewMDNSDiscoverer()

	_, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("StartDiscovery() error = %v", err)
	}

	err = d.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if d.running {
		t.Error("should not be running after Stop()")
	}
}

func TestMDNSDiscoverer_DiscoverWithContext_Timeout(t *testing.T) {
	if os.Getenv("SHELLY_CI") == "1" {
		t.Skip("skipping in CI - requires multicast networking")
	}

	d := NewMDNSDiscoverer()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This will timeout quickly since no real devices
	devices, err := d.DiscoverWithContext(ctx)
	if err != nil {
		t.Fatalf("DiscoverWithContext() error = %v", err)
	}

	// Should return empty slice (no devices found in short timeout)
	if devices == nil {
		t.Error("should return empty slice, not nil")
	}
}

func TestMDNSDiscoverer_Discover_Timeout(t *testing.T) {
	if os.Getenv("SHELLY_CI") == "1" {
		t.Skip("skipping in CI - requires multicast networking")
	}

	d := NewMDNSDiscoverer()

	// Very short timeout
	devices, err := d.Discover(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if devices == nil {
		t.Error("should return empty slice, not nil")
	}
}

func TestMDNSService_Constant(t *testing.T) {
	if MDNSService != "_shelly._tcp.local." {
		t.Errorf("MDNSService = %v, want '_shelly._tcp.local.'", MDNSService)
	}
}

func TestMDNSDiscoverer_ParseResponse_FirmwareExtraction(t *testing.T) {
	d := NewMDNSDiscoverer()

	txtRecords := map[string]string{
		"ver": "1.2.3-beta",
	}
	data := buildMDNSResponse(
		"ShellyTest._shelly._tcp.local",
		txtRecords,
		[4]byte{192, 168, 1, 100},
	)

	device := d.parseResponse(data)
	if device == nil {
		t.Fatal("should parse valid response")
	}

	if device.Firmware != "1.2.3-beta" {
		t.Errorf("Firmware = %v, want '1.2.3-beta'", device.Firmware)
	}
}

func TestMDNSDiscoverer_ParseResponse_DefaultGeneration(t *testing.T) {
	d := NewMDNSDiscoverer()

	// No gen field - should default to Gen2
	txtRecords := map[string]string{
		"app": "Plus1PM",
	}
	data := buildMDNSResponse(
		"ShellyTest._shelly._tcp.local",
		txtRecords,
		[4]byte{192, 168, 1, 100},
	)

	device := d.parseResponse(data)
	if device == nil {
		t.Fatal("should parse valid response")
	}

	// Should default to Gen2 for mDNS devices
	if device.Generation != types.Gen2 {
		t.Errorf("Generation = %v, want Gen2 (default)", device.Generation)
	}
}
