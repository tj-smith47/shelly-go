package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// mockWiFiScanner is a mock implementation of WiFiScanner for testing.
type mockWiFiScanner struct {
	networks       []WiFiNetwork
	currentNetwork *WiFiNetwork
	scanErr        error
	connectErr     error
	disconnectErr  error
	connectCalled  bool
	connectSSID    string
	connectPwd     string
}

func (m *mockWiFiScanner) Scan(ctx context.Context) ([]WiFiNetwork, error) {
	if m.scanErr != nil {
		return nil, m.scanErr
	}
	return m.networks, nil
}

func (m *mockWiFiScanner) Connect(ctx context.Context, ssid, password string) error {
	m.connectCalled = true
	m.connectSSID = ssid
	m.connectPwd = password
	if m.connectErr != nil {
		return m.connectErr
	}
	return nil
}

func (m *mockWiFiScanner) Disconnect(ctx context.Context) error {
	if m.disconnectErr != nil {
		return m.disconnectErr
	}
	return nil
}

func (m *mockWiFiScanner) CurrentNetwork(ctx context.Context) (*WiFiNetwork, error) {
	if m.currentNetwork == nil {
		return nil, &WiFiError{Message: "not connected"}
	}
	return m.currentNetwork, nil
}

func TestNewWiFiDiscoverer(t *testing.T) {
	d := NewWiFiDiscoverer()
	if d == nil {
		t.Fatal("NewWiFiDiscoverer() returned nil")
	}

	if d.Scanner == nil {
		t.Error("Scanner should be initialized")
	}

	if d.devices == nil {
		t.Error("devices map should be initialized")
	}

	if d.ProbeTimeout != 10*time.Second {
		t.Errorf("ProbeTimeout should be 10s, got %v", d.ProbeTimeout)
	}

	if d.HTTPClient == nil {
		t.Error("HTTPClient should be initialized")
	}
}

func TestNewWiFiDiscovererWithScanner(t *testing.T) {
	mock := &mockWiFiScanner{}
	d := NewWiFiDiscovererWithScanner(mock)

	if d == nil {
		t.Fatal("NewWiFiDiscovererWithScanner() returned nil")
	}

	if d.Scanner != mock {
		t.Error("Scanner should be the provided mock")
	}
}

func TestWiFiDiscoverer_Discover(t *testing.T) {
	mock := &mockWiFiScanner{
		networks: []WiFiNetwork{
			{SSID: "shellyplus1pm-AABBCC", Signal: -50},
			{SSID: "HomeNetwork", Signal: -40},
			{SSID: "shelly1-123456", Signal: -60},
		},
	}

	d := NewWiFiDiscovererWithScanner(mock)

	devices, err := d.Discover(5 * time.Second)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should only find Shelly devices
	if len(devices) != 2 {
		t.Errorf("expected 2 Shelly devices, got %d", len(devices))
	}

	// Verify device info
	found := make(map[string]bool)
	for _, dev := range devices {
		found[dev.Name] = true
		if dev.Protocol != ProtocolWiFiAP {
			t.Errorf("expected ProtocolWiFiAP, got %v", dev.Protocol)
		}
	}

	if !found["shellyplus1pm-AABBCC"] {
		t.Error("should find shellyplus1pm-AABBCC")
	}
	if !found["shelly1-123456"] {
		t.Error("should find shelly1-123456")
	}
}

func TestWiFiDiscoverer_DiscoverWithContext_NilScanner(t *testing.T) {
	d := &WiFiDiscoverer{
		Scanner: nil,
		devices: make(map[string]*WiFiDiscoveredDevice),
	}

	_, err := d.DiscoverWithContext(context.Background())
	if err != ErrWiFiNotSupported {
		t.Errorf("expected ErrWiFiNotSupported, got %v", err)
	}
}

func TestWiFiDiscoverer_DiscoverWithContext_ScanError(t *testing.T) {
	mock := &mockWiFiScanner{
		scanErr: &WiFiError{Message: "scan failed"},
	}

	d := NewWiFiDiscovererWithScanner(mock)

	_, err := d.DiscoverWithContext(context.Background())
	if err == nil {
		t.Error("expected error for scan failure")
	}
}

func TestWiFiDiscoverer_ScanNetworks(t *testing.T) {
	mock := &mockWiFiScanner{
		networks: []WiFiNetwork{
			{SSID: "shellyplus1pm-AABBCC", Signal: -50, Channel: 6},
			{SSID: "OtherNetwork", Signal: -40},
			{SSID: "ShellyPro4PM-DEADBEEF", Signal: -55},
		},
	}

	d := NewWiFiDiscovererWithScanner(mock)

	networks, err := d.ScanNetworks(context.Background())
	if err != nil {
		t.Fatalf("ScanNetworks failed: %v", err)
	}

	if len(networks) != 2 {
		t.Errorf("expected 2 Shelly networks, got %d", len(networks))
	}

	for _, net := range networks {
		if !net.IsShelly {
			t.Errorf("IsShelly should be true for %s", net.SSID)
		}
		if net.DeviceType == "" {
			t.Errorf("DeviceType should be set for %s", net.SSID)
		}
	}
}

func TestWiFiDiscoverer_Callbacks(t *testing.T) {
	mock := &mockWiFiScanner{
		networks: []WiFiNetwork{
			{SSID: "shellyplus1pm-AABBCC", Signal: -50},
		},
	}

	d := NewWiFiDiscovererWithScanner(mock)

	var foundNetwork *WiFiNetwork
	var foundDevice *WiFiDiscoveredDevice

	d.OnNetworkFound = func(n *WiFiNetwork) {
		foundNetwork = n
	}
	d.OnDeviceFound = func(dev *WiFiDiscoveredDevice) {
		foundDevice = dev
	}

	_, err := d.Discover(5 * time.Second)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if foundNetwork == nil {
		t.Error("OnNetworkFound callback should have been called")
	}
	if foundDevice == nil {
		t.Error("OnDeviceFound callback should have been called")
	}
}

func TestIsShellyAP(t *testing.T) {
	tests := []struct {
		ssid     string
		expected bool
	}{
		{"shellyplus1pm-AABBCC", true},
		{"shelly1-123456", true},
		{"ShellyPro4PM-DEADBEEF", true},
		{"shellyplugs-ABCD12", true}, // plug-s without extra hyphen
		{"shelly-AABBCC", true},
		{"shellyrgbw2-123ABC", true},
		{"HomeNetwork", false},
		{"shelly", false},              // No device ID
		{"notshelly-AABBCC", false},    // Wrong prefix
		{"shellyplug-s-ABCD12", false}, // Multiple hyphens not matched by pattern
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ssid, func(t *testing.T) {
			result := IsShellyAP(tt.ssid)
			if result != tt.expected {
				t.Errorf("IsShellyAP(%q) = %v, expected %v", tt.ssid, result, tt.expected)
			}
		})
	}
}

func TestParseShellySSID(t *testing.T) {
	tests := []struct {
		ssid     string
		wantType string
		wantID   string
	}{
		{"shellyplus1pm-AABBCC", "plus1pm", "AABBCC"},
		{"shelly1-123456", "1", "123456"},
		{"ShellyPro4PM-DEADBEEF", "pro4pm", "DEADBEEF"},
		{"shellyplugs-ABCD12", "plugs", "ABCD12"},
		{"shellyrgbw2-123ABC", "rgbw2", "123ABC"},
		{"shelly-AABBCC", "", "AABBCC"},
	}

	for _, tt := range tests {
		t.Run(tt.ssid, func(t *testing.T) {
			gotType, gotID := ParseShellySSID(tt.ssid)
			if gotType != tt.wantType {
				t.Errorf("ParseShellySSID(%q) type = %q, want %q", tt.ssid, gotType, tt.wantType)
			}
			if gotID != tt.wantID {
				t.Errorf("ParseShellySSID(%q) id = %q, want %q", tt.ssid, gotID, tt.wantID)
			}
		})
	}
}

func TestInferGenerationFromModel(t *testing.T) {
	tests := []struct {
		model      string
		generation types.Generation
	}{
		{"plus1pm", types.Gen2},
		{"pro4pm", types.Gen2},
		{"1pm", types.Gen1},
		{"rgbw2", types.Gen1},
		{"plus1-g3", types.Gen3},
		{"pro-gen4", types.Gen4},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := InferGenerationFromModel(tt.model)
			if result != tt.generation {
				t.Errorf("InferGenerationFromModel(%q) = %v, expected %v", tt.model, result, tt.generation)
			}
		})
	}
}

func TestWiFiDiscoverer_GetDiscoveredDevices(t *testing.T) {
	mock := &mockWiFiScanner{
		networks: []WiFiNetwork{
			{SSID: "shellyplus1pm-AABBCC", Signal: -50},
			{SSID: "shelly1-123456", Signal: -60},
		},
	}

	d := NewWiFiDiscovererWithScanner(mock)

	// Discover devices first
	_, err := d.Discover(5 * time.Second)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	devices := d.GetDiscoveredDevices()
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
}

func TestWiFiDiscoverer_Clear(t *testing.T) {
	mock := &mockWiFiScanner{
		networks: []WiFiNetwork{
			{SSID: "shellyplus1pm-AABBCC", Signal: -50},
		},
	}

	d := NewWiFiDiscovererWithScanner(mock)

	// Discover devices first
	_, err := d.Discover(5 * time.Second)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(d.GetDiscoveredDevices()) == 0 {
		t.Error("should have discovered devices before clear")
	}

	d.Clear()

	if len(d.GetDiscoveredDevices()) != 0 {
		t.Error("devices should be empty after clear")
	}
}

func TestWiFiDiscoverer_StartStop(t *testing.T) {
	mock := &mockWiFiScanner{
		networks: []WiFiNetwork{},
	}

	d := NewWiFiDiscovererWithScanner(mock)

	ch, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("StartDiscovery failed: %v", err)
	}

	if ch == nil {
		t.Error("channel should not be nil")
	}

	// Start again - should return same channel
	ch2, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("second StartDiscovery failed: %v", err)
	}
	if ch2 != ch {
		t.Error("should return same channel when already running")
	}

	// Stop
	if err := d.StopDiscovery(); err != nil {
		t.Fatalf("StopDiscovery failed: %v", err)
	}

	// Stop again - should be idempotent
	if err := d.StopDiscovery(); err != nil {
		t.Fatalf("second StopDiscovery failed: %v", err)
	}
}

func TestWiFiDiscoverer_StartDiscovery_NilScanner(t *testing.T) {
	d := &WiFiDiscoverer{
		Scanner: nil,
		devices: make(map[string]*WiFiDiscoveredDevice),
	}

	_, err := d.StartDiscovery()
	if err != ErrWiFiNotSupported {
		t.Errorf("expected ErrWiFiNotSupported, got %v", err)
	}
}

func TestWiFiError(t *testing.T) {
	err := &WiFiError{Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test error")
	}

	// With underlying error
	err2 := &WiFiError{Message: "wrapper", Err: err}
	expected := "wrapper: test error"
	if err2.Error() != expected {
		t.Errorf("Error() = %q, want %q", err2.Error(), expected)
	}

	// Unwrap
	if err2.Unwrap() != err {
		t.Error("Unwrap() should return underlying error")
	}
}

func TestWiFiNetwork_ToDevice(t *testing.T) {
	d := NewWiFiDiscoverer()

	network := &WiFiNetwork{
		SSID:     "shellyplus1pm-AABBCC",
		BSSID:    "AA:BB:CC:DD:EE:FF",
		Signal:   -50,
		Channel:  6,
		Security: "WPA2",
		LastSeen: time.Now(),
	}

	device := d.networkToDevice(network)

	if device.SSID != network.SSID {
		t.Errorf("SSID mismatch: got %q, want %q", device.SSID, network.SSID)
	}
	if device.BSSID != network.BSSID {
		t.Errorf("BSSID mismatch: got %q, want %q", device.BSSID, network.BSSID)
	}
	if device.Signal != network.Signal {
		t.Errorf("Signal mismatch: got %d, want %d", device.Signal, network.Signal)
	}
	if device.Protocol != ProtocolWiFiAP {
		t.Errorf("Protocol should be ProtocolWiFiAP, got %v", device.Protocol)
	}
	if device.Port != DefaultAPPort {
		t.Errorf("Port should be %d, got %d", DefaultAPPort, device.Port)
	}
}

func TestWifiscanScanner_NotImplemented(t *testing.T) {
	s := &wifiscanScanner{}

	// Connect should return not implemented error
	err := s.Connect(context.Background(), "test", "password")
	if err == nil {
		t.Error("Connect should return error")
	}

	// Disconnect should return not implemented error
	err = s.Disconnect(context.Background())
	if err == nil {
		t.Error("Disconnect should return error")
	}

	// CurrentNetwork should return not implemented error
	_, err = s.CurrentNetwork(context.Background())
	if err == nil {
		t.Error("CurrentNetwork should return error")
	}
}

func TestSentinelErrors(t *testing.T) {
	// Verify sentinel errors are properly defined
	errors := []*WiFiError{
		ErrWiFiNotSupported,
		ErrSSIDNotFound,
		ErrAuthFailed,
		ErrToolNotFound,
		ErrConnectionTimeout,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("sentinel error should not be nil")
			continue
		}
		if err.Message == "" {
			t.Error("sentinel error should have a message")
		}
	}
}

func TestShellyAPPattern(t *testing.T) {
	// Test the compiled regex pattern
	tests := []struct {
		ssid    string
		matches bool
	}{
		{"shellyplus1pm-AABBCC", true},
		{"shelly1-123456", true},
		{"ShellyPro4PM-DEADBEEF", true},
		{"shellyplugs-ABCD12", true}, // plug-s without extra hyphen
		{"SHELLY1PM-FFFFFF", true},
		{"shellyrgbw2-abc123", true},
		{"shelly", false},
		{"notshelly-AABBCC", false},
		{"", false},
		{"randomssid", false},
		{"shellyplug-s-ABCD12", false}, // Multiple hyphens not currently matched
	}

	for _, tt := range tests {
		t.Run(tt.ssid, func(t *testing.T) {
			result := ShellyAPPattern.MatchString(tt.ssid)
			if result != tt.matches {
				t.Errorf("ShellyAPPattern.MatchString(%q) = %v, expected %v", tt.ssid, result, tt.matches)
			}
		})
	}
}

func TestWiFiDiscoverer_Stop(t *testing.T) {
	mock := &mockWiFiScanner{
		networks: []WiFiNetwork{},
	}

	d := NewWiFiDiscovererWithScanner(mock)

	// Start first
	_, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("StartDiscovery failed: %v", err)
	}

	// Stop using Stop() alias
	if err := d.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
