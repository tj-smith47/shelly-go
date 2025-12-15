package discovery

import (
	"net"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

func TestDiscoveredDevice_URL(t *testing.T) {
	tests := []struct {
		name   string
		want   string
		device DiscoveredDevice
	}{
		{
			name: "default port",
			device: DiscoveredDevice{
				Address: net.ParseIP("192.168.1.100"),
				Port:    0,
			},
			want: "http://192.168.1.100:80",
		},
		{
			name: "custom port",
			device: DiscoveredDevice{
				Address: net.ParseIP("192.168.1.100"),
				Port:    8080,
			},
			want: "http://192.168.1.100:8080",
		},
		{
			name: "standard port",
			device: DiscoveredDevice{
				Address: net.ParseIP("10.0.0.50"),
				Port:    80,
			},
			want: "http://10.0.0.50:80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.URL()
			if got != tt.want {
				t.Errorf("URL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		want  string
		input int
	}{
		{input: 0, want: "0"},
		{input: 1, want: "1"},
		{input: 42, want: "42"},
		{input: 80, want: "80"},
		{input: 8080, want: "8080"},
		{input: -1, want: "-1"},
		{input: -42, want: "-42"},
	}

	for _, tt := range tests {
		got := itoa(tt.input)
		if got != tt.want {
			t.Errorf("itoa(%d) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNewScanner(t *testing.T) {
	s := NewScanner()
	if s == nil {
		t.Fatal("NewScanner() returned nil")
	}

	if !s.enableMDNS {
		t.Error("mDNS should be enabled by default")
	}

	if !s.enableCoIoT {
		t.Error("CoIoT should be enabled by default")
	}
}

func TestScannerOptions(t *testing.T) {
	s := NewScanner(
		WithMDNS(false),
		WithCoIoT(false),
	)

	if s.enableMDNS {
		t.Error("mDNS should be disabled")
	}

	if s.enableCoIoT {
		t.Error("CoIoT should be disabled")
	}
}

func TestScanner_DeduplicateDevices(t *testing.T) {
	s := NewScanner()

	now := time.Now()
	earlier := now.Add(-1 * time.Minute)

	devices := []DiscoveredDevice{
		{
			ID:       "device1",
			Name:     "Device 1 Old",
			LastSeen: earlier,
		},
		{
			ID:       "device1",
			Name:     "Device 1 New",
			LastSeen: now,
		},
		{
			ID:       "device2",
			Name:     "Device 2",
			LastSeen: now,
		},
	}

	result := s.deduplicateDevices(devices)

	if len(result) != 2 {
		t.Errorf("deduplicateDevices() returned %d devices, want 2", len(result))
	}

	// Find device1 in result
	var device1 *DiscoveredDevice
	for i := range result {
		if result[i].ID == "device1" {
			device1 = &result[i]
			break
		}
	}

	if device1 == nil {
		t.Fatal("device1 not found in result")
	}

	if device1.Name != "Device 1 New" {
		t.Errorf("device1.Name = %v, want 'Device 1 New'", device1.Name)
	}
}

func TestScanner_DeduplicateByMAC(t *testing.T) {
	s := NewScanner()

	devices := []DiscoveredDevice{
		{
			MACAddress: "AA:BB:CC:DD:EE:FF",
			Name:       "Device 1",
			LastSeen:   time.Now(),
		},
		{
			MACAddress: "AA:BB:CC:DD:EE:FF",
			Name:       "Device 1 Updated",
			LastSeen:   time.Now().Add(1 * time.Minute),
		},
	}

	result := s.deduplicateDevices(devices)

	if len(result) != 1 {
		t.Errorf("deduplicateDevices() returned %d devices, want 1", len(result))
	}

	if result[0].Name != "Device 1 Updated" {
		t.Errorf("device.Name = %v, want 'Device 1 Updated'", result[0].Name)
	}
}

func TestScanner_DeduplicateByAddress(t *testing.T) {
	s := NewScanner()

	devices := []DiscoveredDevice{
		{
			Address:  net.ParseIP("192.168.1.100"),
			Name:     "Device 1",
			LastSeen: time.Now(),
		},
		{
			Address:  net.ParseIP("192.168.1.100"),
			Name:     "Device 1 Updated",
			LastSeen: time.Now().Add(1 * time.Minute),
		},
	}

	result := s.deduplicateDevices(devices)

	if len(result) != 1 {
		t.Errorf("deduplicateDevices() returned %d devices, want 1", len(result))
	}
}

func TestProtocolConstants(t *testing.T) {
	if ProtocolMDNS != "mdns" {
		t.Errorf("ProtocolMDNS = %v, want 'mdns'", ProtocolMDNS)
	}

	if ProtocolCoIoT != "coiot" {
		t.Errorf("ProtocolCoIoT = %v, want 'coiot'", ProtocolCoIoT)
	}

	if ProtocolBLE != "ble" {
		t.Errorf("ProtocolBLE = %v, want 'ble'", ProtocolBLE)
	}

	if ProtocolManual != "manual" {
		t.Errorf("ProtocolManual = %v, want 'manual'", ProtocolManual)
	}
}

func TestDiscoveredDeviceFields(t *testing.T) {
	d := DiscoveredDevice{
		ID:           "test-id",
		Name:         "Test Device",
		Model:        "SHSW-1",
		Generation:   types.Gen1,
		Address:      net.ParseIP("192.168.1.100"),
		Port:         80,
		MACAddress:   "AA:BB:CC:DD:EE:FF",
		Firmware:     "1.2.3",
		AuthRequired: true,
		Protocol:     ProtocolMDNS,
		LastSeen:     time.Now(),
	}

	if d.ID != "test-id" {
		t.Errorf("ID = %v, want 'test-id'", d.ID)
	}

	if d.Name != "Test Device" {
		t.Errorf("Name = %v, want 'Test Device'", d.Name)
	}

	if d.Model != "SHSW-1" {
		t.Errorf("Model = %v, want 'SHSW-1'", d.Model)
	}

	if d.Generation != types.Gen1 {
		t.Errorf("Generation = %v, want Gen1", d.Generation)
	}

	if !d.AuthRequired {
		t.Error("AuthRequired should be true")
	}
}

func TestScanner_Scan(t *testing.T) {
	// Create scanner with both protocols disabled to test fast path
	s := NewScanner(
		WithMDNS(false),
		WithCoIoT(false),
	)

	devices, err := s.Scan(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// Should return empty slice (no discoverers running)
	if devices == nil {
		t.Error("should return empty slice, not nil")
	}
}

func TestScanner_Scan_WithMDNS(t *testing.T) {
	s := NewScanner(
		WithMDNS(true),
		WithCoIoT(false),
	)

	// Short timeout scan
	devices, err := s.Scan(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// Should return empty slice (no devices found in short timeout)
	if devices == nil {
		t.Error("should return empty slice, not nil")
	}
}

func TestScanner_Scan_WithCoIoT(t *testing.T) {
	s := NewScanner(
		WithMDNS(false),
		WithCoIoT(true),
	)

	devices, err := s.Scan(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if devices == nil {
		t.Error("should return empty slice, not nil")
	}
}

func TestScanner_Scan_WithBoth(t *testing.T) {
	s := NewScanner(
		WithMDNS(true),
		WithCoIoT(true),
	)

	devices, err := s.Scan(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if devices == nil {
		t.Error("should return empty slice, not nil")
	}
}

func TestScanner_Stop(t *testing.T) {
	s := NewScanner()

	// Stop should work even if nothing is running
	err := s.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestScanner_Stop_WithDiscoverers(t *testing.T) {
	s := NewScanner()

	// Initialize the discoverers
	s.mdns = NewMDNSDiscoverer()
	s.coiot = NewCoIoTDiscoverer()

	err := s.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestScanner_DeduplicateDevices_EmptyList(t *testing.T) {
	s := NewScanner()

	devices := s.deduplicateDevices([]DiscoveredDevice{})

	if len(devices) != 0 {
		t.Errorf("deduplicateDevices() returned %d devices, want 0", len(devices))
	}
}

func TestScanner_DeduplicateDevices_SingleDevice(t *testing.T) {
	s := NewScanner()

	devices := []DiscoveredDevice{
		{ID: "device1", Name: "Device 1", LastSeen: time.Now()},
	}

	result := s.deduplicateDevices(devices)

	if len(result) != 1 {
		t.Errorf("deduplicateDevices() returned %d devices, want 1", len(result))
	}
}

func TestScanner_DeduplicateDevices_MultipleUniqueDevices(t *testing.T) {
	s := NewScanner()

	devices := []DiscoveredDevice{
		{ID: "device1", Name: "Device 1", LastSeen: time.Now()},
		{ID: "device2", Name: "Device 2", LastSeen: time.Now()},
		{ID: "device3", Name: "Device 3", LastSeen: time.Now()},
	}

	result := s.deduplicateDevices(devices)

	if len(result) != 3 {
		t.Errorf("deduplicateDevices() returned %d devices, want 3", len(result))
	}
}

func TestScannerWithBLE(t *testing.T) {
	// Test with nil scanner - should not enable BLE
	s := NewScanner(WithBLE(nil))
	if s.enableBLE {
		t.Error("BLE should be disabled when scanner is nil")
	}
	if s.ble != nil {
		t.Error("ble discoverer should be nil when scanner is nil")
	}
}

func TestScannerWithWiFi(t *testing.T) {
	// Test with nil scanner - should not enable WiFi
	s := NewScanner(WithWiFi(nil))
	if s.enableWiFi {
		t.Error("WiFi should be disabled when scanner is nil")
	}
	if s.wifi != nil {
		t.Error("wifi discoverer should be nil when scanner is nil")
	}
}

func TestScanner_Scan_WithWiFiEnabled(t *testing.T) {
	mock := &mockWiFiScanner{
		networks: []WiFiNetwork{
			{SSID: "shellyplus1pm-AABBCC", Signal: -50},
		},
	}

	s := NewScanner(
		WithMDNS(false),
		WithCoIoT(false),
		WithWiFi(mock),
	)

	devices, err := s.Scan(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if devices == nil {
		t.Error("should return slice, not nil")
	}
}

func TestScanner_Stop_WithWiFi(t *testing.T) {
	wifiMock := &mockWiFiScanner{}

	s := NewScanner(
		WithWiFi(wifiMock),
	)

	err := s.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestDiscoveredDevice_URLWithPort(t *testing.T) {
	d := DiscoveredDevice{
		Address: net.ParseIP("192.168.1.100"),
		Port:    8080,
	}

	url := d.URL()
	if url != "http://192.168.1.100:8080" {
		t.Errorf("URL() = %v, want http://192.168.1.100:8080", url)
	}
}

func TestItoa_LargeNumbers(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{input: 65535, want: "65535"},
		{input: 100000, want: "100000"},
		{input: -999, want: "-999"},
	}

	for _, tt := range tests {
		got := itoa(tt.input)
		if got != tt.want {
			t.Errorf("itoa(%d) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestProtocolWiFiAP(t *testing.T) {
	if ProtocolWiFiAP != "wifi_ap" {
		t.Errorf("ProtocolWiFiAP = %v, want 'wifi_ap'", ProtocolWiFiAP)
	}
}
