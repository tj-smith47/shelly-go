package discovery

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

func TestNewCoIoTDiscoverer(t *testing.T) {
	d := NewCoIoTDiscoverer()
	if d == nil {
		t.Fatal("NewCoIoTDiscoverer() returned nil")
	}

	if d.devices == nil {
		t.Error("devices map should be initialized")
	}
}

func TestCoIoTConstants(t *testing.T) {
	if CoIoTMulticastAddr != "224.0.1.187" {
		t.Errorf("CoIoTMulticastAddr = %v, want '224.0.1.187'", CoIoTMulticastAddr)
	}

	if CoIoTPort != 5683 {
		t.Errorf("CoIoTPort = %v, want 5683", CoIoTPort)
	}
}

func TestCoIoTDiscoverer_ParseCoAPMessage_TooShort(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	// Too short
	data := make([]byte, 3)
	device := d.parseCoAPMessage(data, addr)
	if device != nil {
		t.Error("should return nil for short packets")
	}
}

func TestCoIoTDiscoverer_ParseCoAPMessage_WrongVersion(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	// CoAP version 2 (should be 1)
	data := make([]byte, 10)
	data[0] = 0x80 // Version 2

	device := d.parseCoAPMessage(data, addr)
	if device != nil {
		t.Error("should return nil for wrong CoAP version")
	}
}

func TestCoIoTDiscoverer_ParseCoAPMessage_ValidMessage(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	// Build a valid CoAP message
	// Header: Version=1, Type=NON (0x50), Token Length=0
	payload := map[string]any{
		"id":     "AABBCCDDEEFF",
		"mac":    "AA:BB:CC:DD:EE:FF",
		"type":   "SHSW-1",
		"fw_ver": "1.11.0",
	}
	payloadBytes, _ := json.Marshal(payload)

	data := make([]byte, 0, 100)
	data = append(data, 0x50)       // Ver=1, Type=NON, TKL=0
	data = append(data, 0x00)       // Code
	data = append(data, 0x00, 0x01) // Message ID
	data = append(data, 0xFF)       // Payload marker
	data = append(data, payloadBytes...)

	device := d.parseCoAPMessage(data, addr)
	if device == nil {
		t.Fatal("should parse valid CoAP message")
	}

	if device.ID != "AABBCCDDEEFF" {
		t.Errorf("ID = %v, want 'AABBCCDDEEFF'", device.ID)
	}

	if device.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MACAddress = %v, want 'AA:BB:CC:DD:EE:FF'", device.MACAddress)
	}

	if device.Model != "SHSW-1" {
		t.Errorf("Model = %v, want 'SHSW-1'", device.Model)
	}

	if device.Firmware != "1.11.0" {
		t.Errorf("Firmware = %v, want '1.11.0'", device.Firmware)
	}

	if device.Generation != types.Gen1 {
		t.Errorf("Generation = %v, want Gen1", device.Generation)
	}

	if device.Protocol != ProtocolCoIoT {
		t.Errorf("Protocol = %v, want 'coiot'", device.Protocol)
	}

	if !device.Address.Equal(net.ParseIP("192.168.1.100")) {
		t.Errorf("Address = %v, want 192.168.1.100", device.Address)
	}
}

func TestCoIoTDiscoverer_ParsePayload_ValidJSON(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.50"), Port: 5683}

	payload := map[string]any{
		"id":   "test-device",
		"mac":  "11:22:33:44:55:66",
		"type": "SHSW-PM",
		"settings": map[string]any{
			"device": map[string]any{
				"name": "Living Room Switch",
			},
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	device := d.parsePayload(payloadBytes, addr)
	if device == nil {
		t.Fatal("should parse valid JSON payload")
	}

	if device.ID != "test-device" {
		t.Errorf("ID = %v, want 'test-device'", device.ID)
	}

	if device.Name != "Living Room Switch" {
		t.Errorf("Name = %v, want 'Living Room Switch'", device.Name)
	}
}

func TestCoIoTDiscoverer_ParsePayload_MACAsID(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.50"), Port: 5683}

	// No ID field, should use MAC
	payload := map[string]any{
		"mac":  "AA:BB:CC:DD:EE:FF",
		"type": "SHSW-1",
	}
	payloadBytes, _ := json.Marshal(payload)

	device := d.parsePayload(payloadBytes, addr)
	if device == nil {
		t.Fatal("should parse payload")
	}

	if device.ID != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("ID = %v, want 'AA:BB:CC:DD:EE:FF'", device.ID)
	}
}

func TestCoIoTDiscoverer_ParsePayload_InvalidJSON(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.50"), Port: 5683}

	// Invalid JSON with MAC address pattern
	payload := []byte("invalid json but AA:BB:CC:DD:EE:FF is here")

	device := d.parsePayload(payload, addr)
	if device == nil {
		t.Fatal("should extract device from raw payload")
	}

	if device.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MACAddress = %v, want 'AA:BB:CC:DD:EE:FF'", device.MACAddress)
	}
}

func TestCoIoTDiscoverer_ExtractDeviceFromRaw(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	// Raw payload with MAC address
	payload := "some binary data AA:BB:CC:DD:EE:FF more data"

	device := d.extractDeviceFromRaw(payload, addr)
	if device == nil {
		t.Fatal("should extract device")
	}

	if device.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MACAddress = %v, want 'AA:BB:CC:DD:EE:FF'", device.MACAddress)
	}

	// ID should be MAC without colons
	if device.ID != "AABBCCDDEEFF" {
		t.Errorf("ID = %v, want 'AABBCCDDEEFF'", device.ID)
	}

	if device.Generation != types.Gen1 {
		t.Errorf("Generation = %v, want Gen1", device.Generation)
	}

	if device.Port != 80 {
		t.Errorf("Port = %v, want 80", device.Port)
	}
}

func TestCoIoTDiscoverer_ExtractDeviceFromRaw_NoMAC(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	// No MAC address
	payload := "some binary data without mac"

	device := d.extractDeviceFromRaw(payload, addr)
	if device == nil {
		t.Fatal("should return device even without MAC")
	}

	if device.MACAddress != "" {
		t.Errorf("MACAddress = %v, want ''", device.MACAddress)
	}
}

func TestIsValidMAC(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"AA:BB:CC:DD:EE:FF", true},
		{"aa:bb:cc:dd:ee:ff", true},
		{"11:22:33:44:55:66", true},
		{"A1:B2:C3:D4:E5:F6", true},
		{"AABBCCDDEEFF", false},         // No colons
		{"AA:BB:CC:DD:EE", false},       // Too short
		{"AA:BB:CC:DD:EE:FF:GG", false}, // Too long
		{"GG:HH:II:JJ:KK:LL", false},    // Invalid hex
		{"AA-BB-CC-DD-EE-FF", false},    // Wrong separator
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isValidMAC(tt.input)
			if got != tt.want {
				t.Errorf("isValidMAC(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsHexDigit(t *testing.T) {
	tests := []struct {
		input byte
		want  bool
	}{
		{'0', true},
		{'5', true},
		{'9', true},
		{'a', true},
		{'f', true},
		{'A', true},
		{'F', true},
		{'g', false},
		{'G', false},
		{'z', false},
		{' ', false},
		{':', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := isHexDigit(tt.input)
			if got != tt.want {
				t.Errorf("isHexDigit(%c) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCoIoTDiscoverer_StartStopDiscovery(t *testing.T) {
	d := NewCoIoTDiscoverer()

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

	if d.conn != nil {
		t.Error("connection should be nil after stop")
	}

	// Stopping again should be no-op
	err = d.StopDiscovery()
	if err != nil {
		t.Fatalf("StopDiscovery() second call error = %v", err)
	}
}

func TestCoIoTDiscoverer_Stop(t *testing.T) {
	d := NewCoIoTDiscoverer()

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

func TestCoIoTDiscoverer_DiscoverWithContext_Timeout(t *testing.T) {
	d := NewCoIoTDiscoverer()

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

func TestCoIoTDiscoverer_Discover_Timeout(t *testing.T) {
	d := NewCoIoTDiscoverer()

	// Very short timeout
	devices, err := d.Discover(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if devices == nil {
		t.Error("should return empty slice, not nil")
	}
}

func TestCoIoTDiscoverer_ParseCoAPMessage_WithOptions(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	// Build CoAP message with options before payload
	payload := map[string]any{
		"id":   "test",
		"type": "SHSW-1",
	}
	payloadBytes, _ := json.Marshal(payload)

	data := make([]byte, 0, 100)
	data = append(data, 0x50)       // Ver=1, Type=NON, TKL=0
	data = append(data, 0x00)       // Code
	data = append(data, 0x00, 0x01) // Message ID
	// Add some options (delta=1, length=2)
	data = append(data, 0x12)       // delta=1, length=2
	data = append(data, 0xAA, 0xBB) // option value
	data = append(data, 0xFF)       // Payload marker
	data = append(data, payloadBytes...)

	device := d.parseCoAPMessage(data, addr)
	if device == nil {
		t.Fatal("should parse CoAP message with options")
	}

	if device.ID != "test" {
		t.Errorf("ID = %v, want 'test'", device.ID)
	}
}

func TestCoIoTDiscoverer_ParseCoAPMessage_WithToken(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	// Build CoAP message with 4-byte token
	payload := map[string]any{
		"id":   "test",
		"type": "SHSW-1",
	}
	payloadBytes, _ := json.Marshal(payload)

	data := make([]byte, 0, 100)
	data = append(data, 0x54)                   // Ver=1, Type=NON, TKL=4
	data = append(data, 0x00)                   // Code
	data = append(data, 0x00, 0x01)             // Message ID
	data = append(data, 0x01, 0x02, 0x03, 0x04) // Token
	data = append(data, 0xFF)                   // Payload marker
	data = append(data, payloadBytes...)

	device := d.parseCoAPMessage(data, addr)
	if device == nil {
		t.Fatal("should parse CoAP message with token")
	}

	if device.ID != "test" {
		t.Errorf("ID = %v, want 'test'", device.ID)
	}
}

func TestCoIoTDiscoverer_ParseCoAPMessage_TokenTooLong(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	// Packet claims token length of 15 but packet is too short
	data := make([]byte, 10)
	data[0] = 0x4F // Ver=1, Type=CON, TKL=15
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x01

	device := d.parseCoAPMessage(data, addr)
	if device != nil {
		t.Error("should return nil when token length exceeds packet")
	}
}

func TestCoIoTDiscoverer_ParseCoAPMessage_NoPayloadMarker(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	// CoAP message without payload marker
	data := make([]byte, 10)
	data[0] = 0x50 // Ver=1, Type=NON, TKL=0
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x01

	device := d.parseCoAPMessage(data, addr)
	if device != nil {
		t.Error("should return nil when no payload")
	}
}

func TestCoIoTDiscoverer_ParseCoAPMessage_ExtendedOptionDelta13(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	payload := map[string]any{"id": "test", "type": "SHSW-1"}
	payloadBytes, _ := json.Marshal(payload)

	data := make([]byte, 0, 100)
	data = append(data, 0x50) // Ver=1, Type=NON, TKL=0
	data = append(data, 0x00)
	data = append(data, 0x00, 0x01)
	// Option with delta=13 (extended), length=0
	data = append(data, 0xD0) // delta=13, length=0
	data = append(data, 0x00) // extended delta byte
	data = append(data, 0xFF)
	data = append(data, payloadBytes...)

	device := d.parseCoAPMessage(data, addr)
	if device == nil {
		t.Fatal("should parse message with extended option delta")
	}
}

func TestCoIoTDiscoverer_ParseCoAPMessage_ExtendedOptionDelta14(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 5683}

	payload := map[string]any{"id": "test", "type": "SHSW-1"}
	payloadBytes, _ := json.Marshal(payload)

	data := make([]byte, 0, 100)
	data = append(data, 0x50)
	data = append(data, 0x00)
	data = append(data, 0x00, 0x01)
	// Option with delta=14 (2-byte extended), length=0
	data = append(data, 0xE0)       // delta=14, length=0
	data = append(data, 0x00, 0x00) // extended delta bytes
	data = append(data, 0xFF)
	data = append(data, payloadBytes...)

	device := d.parseCoAPMessage(data, addr)
	if device == nil {
		t.Fatal("should parse message with 2-byte extended option delta")
	}
}
