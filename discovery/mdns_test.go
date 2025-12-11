package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

func TestNewMDNSDiscoverer(t *testing.T) {
	d := NewMDNSDiscoverer()
	if d == nil {
		t.Fatal("NewMDNSDiscoverer() returned nil")
	}

	if d.devices == nil {
		t.Error("devices map should be initialized")
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

	// Verify service name is encoded
	// Should contain "_shelly", "_tcp", "local"
	queryStr := string(query[12:])
	if len(queryStr) == 0 {
		t.Error("query should contain service name")
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

	// Construct a mock mDNS response with device info
	data := make([]byte, 200)
	// Header - response flag set
	data[2] = 0x80

	// Add TXT record content
	txtContent := "id=shellyplus1-abc123\x00gen=2\x00model=SNSW-001P16EU\x00fw=1.0.0\x00auth=0"
	copy(data[50:], txtContent)

	// Add A record data (IP address 192.168.1.100)
	data[150] = 192
	data[151] = 168
	data[152] = 1
	data[153] = 100

	device := d.parseResponse(data)
	if device == nil {
		t.Fatal("should parse valid response")
	}

	if device.ID != "shellyplus1-abc123" {
		t.Errorf("ID = %v, want 'shellyplus1-abc123'", device.ID)
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
}

func TestMDNSDiscoverer_ParseResponse_Gen3Device(t *testing.T) {
	d := NewMDNSDiscoverer()

	data := make([]byte, 200)
	data[2] = 0x80

	txtContent := "id=shelly1gen3-xyz789\x00gen=3\x00model=S3SW-001P16EU\x00auth=1"
	copy(data[50:], txtContent)

	data[150] = 10
	data[151] = 0
	data[152] = 0
	data[153] = 50

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

	data := make([]byte, 200)
	data[2] = 0x80

	txtContent := "id=shelly1-abc\x00gen=1\x00"
	copy(data[50:], txtContent)

	data[150] = 172
	data[151] = 16
	data[152] = 0
	data[153] = 10

	device := d.parseResponse(data)
	if device == nil {
		t.Fatal("should parse valid response")
	}

	if device.Generation != types.Gen1 {
		t.Errorf("Generation = %v, want Gen1", device.Generation)
	}
}

func TestMDNSDiscoverer_ParseResponse_NoID(t *testing.T) {
	d := NewMDNSDiscoverer()

	data := make([]byte, 200)
	data[2] = 0x80

	// No ID field
	txtContent := "gen=2\x00model=SNSW-001P16EU"
	copy(data[50:], txtContent)

	data[150] = 192
	data[151] = 168
	data[152] = 1
	data[153] = 100

	device := d.parseResponse(data)
	if device != nil {
		t.Error("should return nil when ID is missing")
	}
}

func TestMDNSDiscoverer_ExtractIP(t *testing.T) {
	d := NewMDNSDiscoverer()

	tests := []struct {
		name     string
		expected string
		data     []byte
	}{
		{
			name: "private IP 192.168.x.x",
			data: func() []byte {
				b := make([]byte, 50)
				b[20] = 192
				b[21] = 168
				b[22] = 1
				b[23] = 100
				return b
			}(),
			expected: "192.168.1.100",
		},
		{
			name: "private IP 10.x.x.x",
			data: func() []byte {
				b := make([]byte, 50)
				b[20] = 10
				b[21] = 0
				b[22] = 0
				b[23] = 50
				return b
			}(),
			expected: "10.0.0.50",
		},
		{
			name: "private IP 172.16.x.x",
			data: func() []byte {
				b := make([]byte, 50)
				b[20] = 172
				b[21] = 16
				b[22] = 0
				b[23] = 10
				return b
			}(),
			expected: "172.16.0.10",
		},
		{
			name: "loopback",
			data: func() []byte {
				b := make([]byte, 50)
				b[20] = 127
				b[21] = 0
				b[22] = 0
				b[23] = 1
				return b
			}(),
			expected: "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := d.extractIP(tt.data)
			if ip == nil {
				t.Fatalf("extractIP() returned nil")
			}
			if ip.String() != tt.expected {
				t.Errorf("extractIP() = %v, want %v", ip.String(), tt.expected)
			}
		})
	}
}

func TestMDNSDiscoverer_ExtractIP_NoValidIP(t *testing.T) {
	d := NewMDNSDiscoverer()

	// Data with only public IPs (not private/loopback)
	data := make([]byte, 50)
	data[20] = 8
	data[21] = 8
	data[22] = 8
	data[23] = 8

	ip := d.extractIP(data)
	if ip != nil {
		t.Error("should return nil for public IPs")
	}
}

func TestMDNSDiscoverer_ExtractIP_ShortData(t *testing.T) {
	d := NewMDNSDiscoverer()

	data := make([]byte, 15)
	ip := d.extractIP(data)
	if ip != nil {
		t.Error("should return nil for short data")
	}
}

func TestMDNSDiscoverer_StartStopDiscovery(t *testing.T) {
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

	data := make([]byte, 200)
	data[2] = 0x80

	txtContent := "id=test\x00fw=1.2.3-beta\x00"
	copy(data[50:], txtContent)

	data[150] = 192
	data[151] = 168
	data[152] = 1
	data[153] = 100

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

	data := make([]byte, 200)
	data[2] = 0x80

	// Unknown generation
	txtContent := "id=test\x00gen=9\x00"
	copy(data[50:], txtContent)

	data[150] = 192
	data[151] = 168
	data[152] = 1
	data[153] = 100

	device := d.parseResponse(data)
	if device == nil {
		t.Fatal("should parse valid response")
	}

	// Should default to Gen2 for unknown generation
	if device.Generation != types.Gen2 {
		t.Errorf("Generation = %v, want Gen2 (default)", device.Generation)
	}
}
