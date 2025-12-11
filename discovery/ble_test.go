package discovery

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// mockBLEScanner is a mock implementation of BLEScanner for testing.
type mockBLEScanner struct {
	advertisements []*BLEAdvertisement
	startErr       error
	stopErr        error
	callback       func(*BLEAdvertisement)
	mu             sync.Mutex
	running        bool
	startCalled    bool
	stopCalled     bool
}

func newMockBLEScanner(advs ...*BLEAdvertisement) *mockBLEScanner {
	return &mockBLEScanner{
		advertisements: advs,
	}
}

func (m *mockBLEScanner) Start(ctx context.Context, callback func(*BLEAdvertisement)) error {
	m.mu.Lock()
	m.startCalled = true
	if m.startErr != nil {
		m.mu.Unlock()
		return m.startErr
	}
	m.callback = callback
	m.running = true
	m.mu.Unlock()

	// Send all advertisements
	for _, adv := range m.advertisements {
		if callback != nil {
			callback(adv)
		}
	}

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

func (m *mockBLEScanner) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCalled = true
	m.running = false
	return m.stopErr
}

// mockBLEConnector is a mock implementation of BLEConnector for testing.
type mockBLEConnector struct {
	connectErr    error
	disconnectErr error
	connected     bool
	mu            sync.Mutex
}

func (m *mockBLEConnector) Connect(ctx context.Context, address string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *mockBLEConnector) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return m.disconnectErr
}

func (m *mockBLEConnector) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// Test BTHome data parsing

func TestParseBTHomeData(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantNil bool
		checkFn func(t *testing.T, result *BTHomeData)
	}{
		{
			name:    "empty data",
			data:    []byte{},
			wantNil: true,
		},
		{
			name:    "encrypted data",
			data:    []byte{0x41}, // encrypted bit set
			wantNil: true,
		},
		{
			name: "battery only",
			data: []byte{
				0x40,       // Device info: version 2, not encrypted
				0x01, 0x64, // Battery 100%
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Battery == nil {
					t.Fatal("Battery should not be nil")
				}
				if *result.Battery != 100 {
					t.Errorf("Battery = %d, want 100", *result.Battery)
				}
			},
		},
		{
			name: "temperature",
			data: []byte{
				0x40,             // Device info: version 2, not encrypted
				0x02, 0xE8, 0x03, // Temperature 10.00C (1000 = 0x03E8)
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Temperature == nil {
					t.Fatal("Temperature should not be nil")
				}
				if *result.Temperature != 10.0 {
					t.Errorf("Temperature = %f, want 10.0", *result.Temperature)
				}
			},
		},
		{
			name: "negative temperature",
			data: []byte{
				0x40,             // Device info: version 2, not encrypted
				0x02, 0x18, 0xFC, // Temperature -10.00C (-1000 as int16)
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Temperature == nil {
					t.Fatal("Temperature should not be nil")
				}
				if *result.Temperature != -10.0 {
					t.Errorf("Temperature = %f, want -10.0", *result.Temperature)
				}
			},
		},
		{
			name: "humidity",
			data: []byte{
				0x40,             // Device info: version 2, not encrypted
				0x03, 0x88, 0x13, // Humidity 50.00% (5000 = 0x1388)
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Humidity == nil {
					t.Fatal("Humidity should not be nil")
				}
				if *result.Humidity != 50.0 {
					t.Errorf("Humidity = %f, want 50.0", *result.Humidity)
				}
			},
		},
		{
			name: "illuminance",
			data: []byte{
				0x40,                   // Device info: version 2, not encrypted
				0x05, 0x10, 0x27, 0x00, // Illuminance 100.00 lux (10000 = 0x002710)
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Illuminance == nil {
					t.Fatal("Illuminance should not be nil")
				}
				if *result.Illuminance != 10000 {
					t.Errorf("Illuminance = %d, want 10000", *result.Illuminance)
				}
			},
		},
		{
			name: "motion detected",
			data: []byte{
				0x40,       // Device info: version 2, not encrypted
				0x21, 0x01, // Motion detected
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Motion == nil {
					t.Fatal("Motion should not be nil")
				}
				if *result.Motion != true {
					t.Error("Motion = false, want true")
				}
			},
		},
		{
			name: "motion clear",
			data: []byte{
				0x40,       // Device info: version 2, not encrypted
				0x21, 0x00, // Motion clear
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Motion == nil {
					t.Fatal("Motion should not be nil")
				}
				if *result.Motion != false {
					t.Error("Motion = true, want false")
				}
			},
		},
		{
			name: "window open",
			data: []byte{
				0x40,       // Device info: version 2, not encrypted
				0x2D, 0x01, // Window open
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.WindowOpen == nil {
					t.Fatal("WindowOpen should not be nil")
				}
				if *result.WindowOpen != true {
					t.Error("WindowOpen = false, want true")
				}
			},
		},
		{
			name: "button press",
			data: []byte{
				0x40,       // Device info: version 2, not encrypted
				0x3A, 0x01, // Button single press
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Button == nil {
					t.Fatal("Button should not be nil")
				}
				if *result.Button != 1 {
					t.Errorf("Button = %d, want 1", *result.Button)
				}
			},
		},
		{
			name: "rotation",
			data: []byte{
				0x40,             // Device info: version 2, not encrypted
				0x3F, 0x84, 0x03, // Rotation 90.0 (900 = 0x0384)
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Rotation == nil {
					t.Fatal("Rotation should not be nil")
				}
				if *result.Rotation != 90.0 {
					t.Errorf("Rotation = %f, want 90.0", *result.Rotation)
				}
			},
		},
		{
			name: "negative rotation",
			data: []byte{
				0x40,             // Device info: version 2, not encrypted
				0x3F, 0x7C, 0xFC, // Rotation -90.0 (-900 as int16)
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Rotation == nil {
					t.Fatal("Rotation should not be nil")
				}
				if *result.Rotation != -90.0 {
					t.Errorf("Rotation = %f, want -90.0", *result.Rotation)
				}
			},
		},
		{
			name: "packet ID",
			data: []byte{
				0x40,       // Device info: version 2, not encrypted
				0x00, 0x42, // Packet ID 66
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.PacketID != 66 {
					t.Errorf("PacketID = %d, want 66", result.PacketID)
				}
			},
		},
		{
			name: "multiple objects",
			data: []byte{
				0x40,       // Device info: version 2, not encrypted
				0x00, 0x01, // Packet ID 1
				0x01, 0x64, // Battery 100%
				0x02, 0xE8, 0x03, // Temperature 10.00C
				0x03, 0x88, 0x13, // Humidity 50.00%
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.PacketID != 1 {
					t.Errorf("PacketID = %d, want 1", result.PacketID)
				}
				if result.Battery == nil || *result.Battery != 100 {
					t.Errorf("Battery = %v, want 100", result.Battery)
				}
				if result.Temperature == nil || *result.Temperature != 10.0 {
					t.Errorf("Temperature = %v, want 10.0", result.Temperature)
				}
				if result.Humidity == nil || *result.Humidity != 50.0 {
					t.Errorf("Humidity = %v, want 50.0", result.Humidity)
				}
			},
		},
		{
			name: "unknown object skipped",
			data: []byte{
				0x40,       // Device info: version 2, not encrypted
				0xFF, 0x00, // Unknown object (skipped with size 1)
				0x01, 0x64, // Battery 100%
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Battery == nil {
					t.Fatal("Battery should not be nil")
				}
				if *result.Battery != 100 {
					t.Errorf("Battery = %d, want 100", *result.Battery)
				}
			},
		},
		{
			name: "truncated data",
			data: []byte{
				0x40,       // Device info: version 2, not encrypted
				0x02, 0xE8, // Temperature missing second byte
			},
			checkFn: func(t *testing.T, result *BTHomeData) {
				// Should parse but temperature should be nil due to truncation
				if result.Temperature != nil {
					t.Error("Temperature should be nil for truncated data")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBTHomeData(tt.data)
			if tt.wantNil {
				if result != nil {
					t.Errorf("parseBTHomeData() = %v, want nil", result)
				}
				return
			}
			if result == nil {
				t.Fatal("parseBTHomeData() = nil, want non-nil")
			}
			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

func TestParseBTHomeObject(t *testing.T) {
	// Test edge cases for each object type
	tests := []struct {
		name     string
		objectID uint8
		data     []byte
		checkFn  func(t *testing.T, result *BTHomeData)
	}{
		{
			name:     "battery 0%",
			objectID: 0x01,
			data:     []byte{0x00},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Battery == nil || *result.Battery != 0 {
					t.Errorf("Battery = %v, want 0", result.Battery)
				}
			},
		},
		{
			name:     "battery 255%", // edge case - max uint8
			objectID: 0x01,
			data:     []byte{0xFF},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Battery == nil || *result.Battery != 255 {
					t.Errorf("Battery = %v, want 255", result.Battery)
				}
			},
		},
		{
			name:     "temperature zero",
			objectID: 0x02,
			data:     []byte{0x00, 0x00},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Temperature == nil || *result.Temperature != 0.0 {
					t.Errorf("Temperature = %v, want 0.0", result.Temperature)
				}
			},
		},
		{
			name:     "humidity zero",
			objectID: 0x03,
			data:     []byte{0x00, 0x00},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Humidity == nil || *result.Humidity != 0.0 {
					t.Errorf("Humidity = %v, want 0.0", result.Humidity)
				}
			},
		},
		{
			name:     "illuminance zero",
			objectID: 0x05,
			data:     []byte{0x00, 0x00, 0x00},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Illuminance == nil || *result.Illuminance != 0 {
					t.Errorf("Illuminance = %v, want 0", result.Illuminance)
				}
			},
		},
		{
			name:     "illuminance max uint24",
			objectID: 0x05,
			data:     []byte{0xFF, 0xFF, 0xFF},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Illuminance == nil || *result.Illuminance != 0xFFFFFF {
					t.Errorf("Illuminance = %v, want %d", result.Illuminance, 0xFFFFFF)
				}
			},
		},
		{
			name:     "window closed",
			objectID: 0x2D,
			data:     []byte{0x00},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.WindowOpen == nil || *result.WindowOpen != false {
					t.Errorf("WindowOpen = %v, want false", result.WindowOpen)
				}
			},
		},
		{
			name:     "button no press",
			objectID: 0x3A,
			data:     []byte{0x00},
			checkFn: func(t *testing.T, result *BTHomeData) {
				if result.Button == nil || *result.Button != 0 {
					t.Errorf("Button = %v, want 0", result.Button)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &BTHomeData{}
			parseBTHomeObject(result, tt.objectID, tt.data)
			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

func TestBTHomeData_Fields(t *testing.T) {
	// Test that BTHomeData struct can hold all field types
	data := &BTHomeData{}

	// Set all fields
	battery := uint8(50)
	temp := 25.5
	hum := 60.0
	lux := uint32(1000)
	motion := true
	window := false
	button := uint8(2)
	rotation := 45.0

	data.PacketID = 1
	data.Battery = &battery
	data.Temperature = &temp
	data.Humidity = &hum
	data.Illuminance = &lux
	data.Motion = &motion
	data.WindowOpen = &window
	data.Button = &button
	data.Rotation = &rotation

	// Verify all fields
	if data.PacketID != 1 {
		t.Errorf("PacketID = %d, want 1", data.PacketID)
	}
	if *data.Battery != 50 {
		t.Errorf("Battery = %d, want 50", *data.Battery)
	}
	if *data.Temperature != 25.5 {
		t.Errorf("Temperature = %f, want 25.5", *data.Temperature)
	}
	if *data.Humidity != 60.0 {
		t.Errorf("Humidity = %f, want 60.0", *data.Humidity)
	}
	if *data.Illuminance != 1000 {
		t.Errorf("Illuminance = %d, want 1000", *data.Illuminance)
	}
	if *data.Motion != true {
		t.Error("Motion = false, want true")
	}
	if *data.WindowOpen != false {
		t.Error("WindowOpen = true, want false")
	}
	if *data.Button != 2 {
		t.Errorf("Button = %d, want 2", *data.Button)
	}
	if *data.Rotation != 45.0 {
		t.Errorf("Rotation = %f, want 45.0", *data.Rotation)
	}
}

func TestBLEDiscoveredDevice_Fields(t *testing.T) {
	device := &BLEDiscoveredDevice{
		DiscoveredDevice: DiscoveredDevice{
			Name:       "SBBT-002C",
			MACAddress: "AA:BB:CC:DD:EE:FF",
			Model:      "button",
		},
		RSSI:        -65,
		Connectable: true,
	}

	if device.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MACAddress = %s, want AA:BB:CC:DD:EE:FF", device.MACAddress)
	}
	if device.Name != "SBBT-002C" {
		t.Errorf("Name = %s, want SBBT-002C", device.Name)
	}
	if device.Model != "button" {
		t.Errorf("Model = %s, want button", device.Model)
	}
	if device.RSSI != -65 {
		t.Errorf("RSSI = %d, want -65", device.RSSI)
	}
	if !device.Connectable {
		t.Error("Connectable = false, want true")
	}
}

func TestErrBLENotSupported(t *testing.T) {
	if ErrBLENotSupported == nil {
		t.Error("ErrBLENotSupported should not be nil")
	}
	// Just verify the error contains expected keywords
	errMsg := ErrBLENotSupported.Error()
	if errMsg == "" {
		t.Error("ErrBLENotSupported.Error() should not be empty")
	}
}

// Test BLEDiscoverer constructor and configuration

func TestNewBLEDiscovererWithScanner(t *testing.T) {
	scanner := newMockBLEScanner()
	d := NewBLEDiscovererWithScanner(scanner)

	if d == nil {
		t.Fatal("NewBLEDiscovererWithScanner returned nil")
	}
	if d.Scanner != scanner {
		t.Error("Scanner not set correctly")
	}
	if d.devices == nil {
		t.Error("devices map should be initialized")
	}
	if d.ScanDuration != 10*time.Second {
		t.Errorf("ScanDuration = %v, want 10s", d.ScanDuration)
	}
	if d.FilterPrefix != ShellyBLEAdvertisementPrefix {
		t.Errorf("FilterPrefix = %s, want %s", d.FilterPrefix, ShellyBLEAdvertisementPrefix)
	}
	if !d.IncludeBTHome {
		t.Error("IncludeBTHome should be true by default")
	}
}

func TestBLEDiscoverer_NilScanner(t *testing.T) {
	d := NewBLEDiscovererWithScanner(nil)

	_, err := d.Discover(100 * time.Millisecond)
	if err == nil {
		t.Error("Discover should return error with nil scanner")
	}
	if err != ErrBLENotSupported {
		t.Errorf("err = %v, want ErrBLENotSupported", err)
	}

	_, err = d.StartDiscovery()
	if err == nil {
		t.Error("StartDiscovery should return error with nil scanner")
	}
	if err != ErrBLENotSupported {
		t.Errorf("err = %v, want ErrBLENotSupported", err)
	}
}

// Test isShellyDevice

func TestBLEDiscoverer_IsShellyDevice(t *testing.T) {
	d := NewBLEDiscovererWithScanner(newMockBLEScanner())

	tests := []struct {
		name     string
		adv      *BLEAdvertisement
		expected bool
	}{
		{
			name: "Shelly prefix uppercase",
			adv: &BLEAdvertisement{
				LocalName: "SHELLY-PLUS1-ABC123",
			},
			expected: true,
		},
		{
			name: "Shelly prefix lowercase",
			adv: &BLEAdvertisement{
				LocalName: "shelly-plus2-def456",
			},
			expected: true,
		},
		{
			name: "Shelly service UUID",
			adv: &BLEAdvertisement{
				LocalName:    "Unknown",
				ServiceUUIDs: []string{ShellyBLEServiceUUID},
			},
			expected: true,
		},
		{
			name: "BTHome service data",
			adv: &BLEAdvertisement{
				LocalName:   "SBDW-002C",
				ServiceData: map[string][]byte{BTHomeServiceUUID: {0x40, 0x01, 0x64}},
			},
			expected: true,
		},
		{
			name: "BTHome disabled",
			adv: &BLEAdvertisement{
				LocalName:   "SBDW-002C",
				ServiceData: map[string][]byte{BTHomeServiceUUID: {0x40, 0x01, 0x64}},
			},
			expected: false,
		},
		{
			name: "Non-Shelly device",
			adv: &BLEAdvertisement{
				LocalName: "RandomDevice",
			},
			expected: false,
		},
		{
			name: "Empty advertisement",
			adv: &BLEAdvertisement{
				ServiceData: make(map[string][]byte),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle BTHome disabled test case
			if tt.name == "BTHome disabled" {
				d.IncludeBTHome = false
				defer func() { d.IncludeBTHome = true }()
			}

			result := d.isShellyDevice(tt.adv)
			if result != tt.expected {
				t.Errorf("isShellyDevice() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test parseAdvertisement

func TestBLEDiscoverer_ParseAdvertisement(t *testing.T) {
	d := NewBLEDiscovererWithScanner(newMockBLEScanner())

	tests := []struct {
		name    string
		adv     *BLEAdvertisement
		checkFn func(t *testing.T, device *BLEDiscoveredDevice)
	}{
		{
			name: "basic device",
			adv: &BLEAdvertisement{
				Address:     "AA:BB:CC:DD:EE:FF",
				LocalName:   "SHELLY-PLUS1-ABC123",
				RSSI:        -70,
				Connectable: true,
				ServiceData: make(map[string][]byte),
			},
			checkFn: func(t *testing.T, device *BLEDiscoveredDevice) {
				if device.ID != "AA:BB:CC:DD:EE:FF" {
					t.Errorf("ID = %s, want AA:BB:CC:DD:EE:FF", device.ID)
				}
				if device.MACAddress != "AA:BB:CC:DD:EE:FF" {
					t.Errorf("MACAddress = %s, want AA:BB:CC:DD:EE:FF", device.MACAddress)
				}
				if device.Name != "SHELLY-PLUS1-ABC123" {
					t.Errorf("Name = %s, want SHELLY-PLUS1-ABC123", device.Name)
				}
				if device.Model != "PLUS1" {
					t.Errorf("Model = %s, want PLUS1", device.Model)
				}
				if device.RSSI != -70 {
					t.Errorf("RSSI = %d, want -70", device.RSSI)
				}
				if !device.Connectable {
					t.Error("Connectable should be true")
				}
				if device.Protocol != ProtocolBLE {
					t.Errorf("Protocol = %s, want %s", device.Protocol, ProtocolBLE)
				}
				if device.Generation != types.Gen2 {
					t.Errorf("Generation = %v, want Gen2", device.Generation)
				}
			},
		},
		{
			name: "with service UUID",
			adv: &BLEAdvertisement{
				Address:      "11:22:33:44:55:66",
				LocalName:    "SHELLY-DIMMINUS-XYZ",
				RSSI:         -55,
				ServiceUUIDs: []string{ShellyBLEServiceUUID},
				ServiceData:  make(map[string][]byte),
			},
			checkFn: func(t *testing.T, device *BLEDiscoveredDevice) {
				if device.ServiceUUID != ShellyBLEServiceUUID {
					t.Errorf("ServiceUUID = %s, want %s", device.ServiceUUID, ShellyBLEServiceUUID)
				}
			},
		},
		{
			name: "with BTHome data",
			adv: &BLEAdvertisement{
				Address:   "AA:BB:CC:DD:EE:00",
				LocalName: "SBDW-002C",
				RSSI:      -60,
				ServiceData: map[string][]byte{
					BTHomeServiceUUID: {0x40, 0x01, 0x64}, // Battery 100%
				},
			},
			checkFn: func(t *testing.T, device *BLEDiscoveredDevice) {
				if device.BTHomeData == nil {
					t.Fatal("BTHomeData should not be nil")
				}
				if device.BTHomeData.Battery == nil || *device.BTHomeData.Battery != 100 {
					t.Errorf("Battery = %v, want 100", device.BTHomeData.Battery)
				}
			},
		},
		{
			name: "model extraction - three parts",
			adv: &BLEAdvertisement{
				Address:     "AA:BB:CC:DD:EE:11",
				LocalName:   "SHELLY-PRO4PM-ABCD",
				ServiceData: make(map[string][]byte),
			},
			checkFn: func(t *testing.T, device *BLEDiscoveredDevice) {
				if device.Model != "PRO4PM" {
					t.Errorf("Model = %s, want PRO4PM", device.Model)
				}
			},
		},
		{
			name: "non-shelly name format",
			adv: &BLEAdvertisement{
				Address:      "AA:BB:CC:DD:EE:22",
				LocalName:    "SomethingElse",
				ServiceUUIDs: []string{ShellyBLEServiceUUID},
				ServiceData:  make(map[string][]byte),
			},
			checkFn: func(t *testing.T, device *BLEDiscoveredDevice) {
				// Model should be empty for non-SHELLY- prefix names
				if device.Model != "" {
					t.Errorf("Model = %s, want empty", device.Model)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := d.parseAdvertisement(tt.adv)
			if device == nil {
				t.Fatal("parseAdvertisement returned nil")
			}
			if tt.checkFn != nil {
				tt.checkFn(t, device)
			}
		})
	}
}

// Test Discover and DiscoverWithContext

func TestBLEDiscoverer_Discover(t *testing.T) {
	adv := &BLEAdvertisement{
		Address:     "AA:BB:CC:DD:EE:FF",
		LocalName:   "SHELLY-PLUS1-123",
		RSSI:        -70,
		ServiceData: make(map[string][]byte),
	}
	scanner := newMockBLEScanner(adv)
	d := NewBLEDiscovererWithScanner(scanner)

	devices, err := d.Discover(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(devices) != 1 {
		t.Errorf("len(devices) = %d, want 1", len(devices))
	}

	if !scanner.startCalled {
		t.Error("Scanner.Start was not called")
	}
}

func TestBLEDiscoverer_DiscoverWithContext(t *testing.T) {
	adv1 := &BLEAdvertisement{
		Address:     "AA:BB:CC:DD:EE:01",
		LocalName:   "SHELLY-PLUS1-001",
		RSSI:        -65,
		ServiceData: make(map[string][]byte),
	}
	adv2 := &BLEAdvertisement{
		Address:     "AA:BB:CC:DD:EE:02",
		LocalName:   "SHELLY-PLUS2-002",
		RSSI:        -70,
		ServiceData: make(map[string][]byte),
	}
	scanner := newMockBLEScanner(adv1, adv2)
	d := NewBLEDiscovererWithScanner(scanner)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	devices, err := d.DiscoverWithContext(ctx)
	if err != nil {
		t.Fatalf("DiscoverWithContext() error = %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("len(devices) = %d, want 2", len(devices))
	}
}

func TestBLEDiscoverer_DiscoverFiltersNonShelly(t *testing.T) {
	shellyAdv := &BLEAdvertisement{
		Address:     "AA:BB:CC:DD:EE:01",
		LocalName:   "SHELLY-PLUS1-001",
		RSSI:        -65,
		ServiceData: make(map[string][]byte),
	}
	nonShellyAdv := &BLEAdvertisement{
		Address:     "11:22:33:44:55:66",
		LocalName:   "SomeOtherDevice",
		RSSI:        -50,
		ServiceData: make(map[string][]byte),
	}
	scanner := newMockBLEScanner(shellyAdv, nonShellyAdv)
	d := NewBLEDiscovererWithScanner(scanner)

	devices, err := d.Discover(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should only contain the Shelly device
	if len(devices) != 1 {
		t.Errorf("len(devices) = %d, want 1", len(devices))
	}
}

// Test continuous discovery

func TestBLEDiscoverer_StartStopDiscovery(t *testing.T) {
	scanner := newMockBLEScanner()
	d := NewBLEDiscovererWithScanner(scanner)
	d.ScanDuration = 50 * time.Millisecond

	ch, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("StartDiscovery() error = %v", err)
	}
	if ch == nil {
		t.Error("channel should not be nil")
	}

	// Wait for scanner to start
	time.Sleep(10 * time.Millisecond)

	d.mu.RLock()
	running := d.running
	d.mu.RUnlock()

	if !running {
		t.Error("discoverer should be running")
	}

	// Second call should return same channel
	ch2, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("StartDiscovery() second call error = %v", err)
	}
	if ch2 != ch {
		t.Error("should return same channel")
	}

	// Stop discovery
	err = d.StopDiscovery()
	if err != nil {
		t.Fatalf("StopDiscovery() error = %v", err)
	}

	d.mu.RLock()
	running = d.running
	d.mu.RUnlock()

	if running {
		t.Error("discoverer should not be running")
	}

	// Stopping again should be no-op
	err = d.StopDiscovery()
	if err != nil {
		t.Fatalf("StopDiscovery() second call error = %v", err)
	}
}

func TestBLEDiscoverer_Stop(t *testing.T) {
	scanner := newMockBLEScanner()
	d := NewBLEDiscovererWithScanner(scanner)
	d.ScanDuration = 50 * time.Millisecond

	_, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("StartDiscovery() error = %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = d.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	d.mu.RLock()
	running := d.running
	d.mu.RUnlock()

	if running {
		t.Error("discoverer should not be running after Stop()")
	}
}

// Test device management

func TestBLEDiscoverer_GetDiscoveredDevices(t *testing.T) {
	adv1 := &BLEAdvertisement{
		Address:     "AA:BB:CC:DD:EE:01",
		LocalName:   "SHELLY-PLUS1-001",
		ServiceData: make(map[string][]byte),
	}
	adv2 := &BLEAdvertisement{
		Address:     "AA:BB:CC:DD:EE:02",
		LocalName:   "SHELLY-PLUS2-002",
		ServiceData: make(map[string][]byte),
	}
	scanner := newMockBLEScanner(adv1, adv2)
	d := NewBLEDiscovererWithScanner(scanner)

	// Discover devices first
	_, err := d.Discover(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	devices := d.GetDiscoveredDevices()
	if len(devices) != 2 {
		t.Errorf("len(devices) = %d, want 2", len(devices))
	}
}

func TestBLEDiscoverer_DeviceByAddress(t *testing.T) {
	adv := &BLEAdvertisement{
		Address:     "AA:BB:CC:DD:EE:FF",
		LocalName:   "SHELLY-PLUS1-123",
		ServiceData: make(map[string][]byte),
	}
	scanner := newMockBLEScanner(adv)
	d := NewBLEDiscovererWithScanner(scanner)

	_, err := d.Discover(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	device := d.DeviceByAddress("AA:BB:CC:DD:EE:FF")
	if device == nil {
		t.Fatal("DeviceByAddress returned nil")
	}
	if device.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MACAddress = %s, want AA:BB:CC:DD:EE:FF", device.MACAddress)
	}

	// Non-existent address
	device = d.DeviceByAddress("00:00:00:00:00:00")
	if device != nil {
		t.Error("DeviceByAddress should return nil for unknown address")
	}
}

func TestBLEDiscoverer_Clear(t *testing.T) {
	adv := &BLEAdvertisement{
		Address:     "AA:BB:CC:DD:EE:FF",
		LocalName:   "SHELLY-PLUS1-123",
		ServiceData: make(map[string][]byte),
	}
	scanner := newMockBLEScanner(adv)
	d := NewBLEDiscovererWithScanner(scanner)

	_, err := d.Discover(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Verify device was found
	if len(d.GetDiscoveredDevices()) == 0 {
		t.Fatal("should have discovered devices")
	}

	// Clear and verify
	d.Clear()

	devices := d.GetDiscoveredDevices()
	if len(devices) != 0 {
		t.Errorf("len(devices) = %d, want 0 after Clear()", len(devices))
	}
}

// Test callback

func TestBLEDiscoverer_OnDeviceFound(t *testing.T) {
	adv := &BLEAdvertisement{
		Address:     "AA:BB:CC:DD:EE:FF",
		LocalName:   "SHELLY-PLUS1-123",
		ServiceData: make(map[string][]byte),
	}
	scanner := newMockBLEScanner(adv)
	d := NewBLEDiscovererWithScanner(scanner)

	var callbackDevice *BLEDiscoveredDevice
	var callbackCalled bool

	d.OnDeviceFound = func(device *BLEDiscoveredDevice) {
		callbackCalled = true
		callbackDevice = device
	}

	_, err := d.Discover(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if !callbackCalled {
		t.Error("OnDeviceFound callback was not called")
	}

	if callbackDevice == nil {
		t.Fatal("callbackDevice is nil")
	}

	if callbackDevice.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("callbackDevice.MACAddress = %s, want AA:BB:CC:DD:EE:FF", callbackDevice.MACAddress)
	}
}

// Test IsDeviceProvisioned

func TestIsDeviceProvisioned(t *testing.T) {
	tests := []struct {
		name     string
		device   *BLEDiscoveredDevice
		expected bool
	}{
		{
			name: "nil address",
			device: &BLEDiscoveredDevice{
				DiscoveredDevice: DiscoveredDevice{
					Address: nil,
				},
			},
			expected: false,
		},
		{
			name: "zero IP",
			device: &BLEDiscoveredDevice{
				DiscoveredDevice: DiscoveredDevice{
					Address: []byte{0, 0, 0, 0},
				},
			},
			expected: false,
		},
		{
			name: "valid IP",
			device: &BLEDiscoveredDevice{
				DiscoveredDevice: DiscoveredDevice{
					Address: []byte{192, 168, 1, 100},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDeviceProvisioned(tt.device)
			if result != tt.expected {
				t.Errorf("IsDeviceProvisioned() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test BLEConnector and connectability cache

func TestIsConnectable_NilDevice(t *testing.T) {
	_, err := IsConnectable(context.Background(), nil, nil)
	if err == nil {
		t.Error("IsConnectable should return error for nil device")
	}
}

func TestIsConnectable_NoConnector(t *testing.T) {
	device := &BLEDiscoveredDevice{
		DiscoveredDevice: DiscoveredDevice{
			MACAddress: "AA:BB:CC:DD:EE:FF",
		},
		Connectable: true,
	}

	result, err := IsConnectable(context.Background(), device, nil)
	if err != nil {
		t.Fatalf("IsConnectable() error = %v", err)
	}
	if result != true {
		t.Error("IsConnectable should return advertisement value when no connector")
	}
}

func TestIsConnectable_WithConnector(t *testing.T) {
	// Clear cache first
	ClearConnectabilityCache()

	device := &BLEDiscoveredDevice{
		DiscoveredDevice: DiscoveredDevice{
			MACAddress: "11:22:33:44:55:66",
		},
		Connectable: false, // Advertisement says false
	}

	connector := &mockBLEConnector{}

	// Connector succeeds - should return true
	result, err := IsConnectable(context.Background(), device, connector)
	if err != nil {
		t.Fatalf("IsConnectable() error = %v", err)
	}
	if result != true {
		t.Error("IsConnectable should return true when connector succeeds")
	}

	// Result should be cached
	result, err = IsConnectable(context.Background(), device, connector)
	if err != nil {
		t.Fatalf("IsConnectable() second call error = %v", err)
	}
	if result != true {
		t.Error("cached result should be true")
	}
}

func TestIsConnectable_ConnectorFails(t *testing.T) {
	ClearConnectabilityCache()

	device := &BLEDiscoveredDevice{
		DiscoveredDevice: DiscoveredDevice{
			MACAddress: "AA:AA:AA:AA:AA:AA",
		},
		Connectable: true, // Advertisement says true
	}

	connector := &mockBLEConnector{
		connectErr: &BLEError{Message: "connection failed"},
	}

	result, err := IsConnectable(context.Background(), device, connector)
	if err != nil {
		t.Fatalf("IsConnectable() error = %v", err)
	}
	if result != false {
		t.Error("IsConnectable should return false when connector fails")
	}
}

func TestClearConnectabilityCache(t *testing.T) {
	// Add something to cache
	bleConnectabilityCache.mu.Lock()
	bleConnectabilityCache.results["test-address"] = true
	bleConnectabilityCache.mu.Unlock()

	// Verify it's there
	bleConnectabilityCache.mu.RLock()
	if _, ok := bleConnectabilityCache.results["test-address"]; !ok {
		bleConnectabilityCache.mu.RUnlock()
		t.Fatal("test setup failed - item not in cache")
	}
	bleConnectabilityCache.mu.RUnlock()

	// Clear cache
	ClearConnectabilityCache()

	// Verify it's gone
	bleConnectabilityCache.mu.RLock()
	if len(bleConnectabilityCache.results) != 0 {
		bleConnectabilityCache.mu.RUnlock()
		t.Error("cache should be empty after ClearConnectabilityCache()")
	}
	bleConnectabilityCache.mu.RUnlock()
}

// Test BLEError

func TestBLEError(t *testing.T) {
	// Error with wrapped error
	wrapped := &BLEError{Message: "outer", Err: &BLEError{Message: "inner"}}
	if wrapped.Error() != "outer: inner" {
		t.Errorf("Error() = %s, want 'outer: inner'", wrapped.Error())
	}
	if wrapped.Unwrap() == nil {
		t.Error("Unwrap() should not return nil")
	}

	// Error without wrapped error
	simple := &BLEError{Message: "simple error"}
	if simple.Error() != "simple error" {
		t.Errorf("Error() = %s, want 'simple error'", simple.Error())
	}
	if simple.Unwrap() != nil {
		t.Error("Unwrap() should return nil")
	}
}

// Test constants

func TestBLEConstants(t *testing.T) {
	if ShellyBLEServiceUUID != "5f6d4f53-5f52-5043-5f53-56435f49445f" {
		t.Errorf("ShellyBLEServiceUUID = %s, want 5f6d4f53-5f52-5043-5f53-56435f49445f", ShellyBLEServiceUUID)
	}
	if ShellyBLEAdvertisementPrefix != "SHELLY-" {
		t.Errorf("ShellyBLEAdvertisementPrefix = %s, want SHELLY-", ShellyBLEAdvertisementPrefix)
	}
	if BTHomeServiceUUID != "fcd2" {
		t.Errorf("BTHomeServiceUUID = %s, want fcd2", BTHomeServiceUUID)
	}
}

// Test bthomeObjectSizes map

func TestBthomeObjectSizes(t *testing.T) {
	expectedSizes := map[uint8]int{
		0x00: 1, // Packet ID
		0x01: 1, // Battery
		0x02: 2, // Temperature
		0x03: 2, // Humidity
		0x05: 3, // Illuminance
		0x21: 1, // Motion
		0x2D: 1, // Window
		0x3A: 1, // Button
		0x3F: 2, // Rotation
	}

	for id, expectedSize := range expectedSizes {
		actualSize, ok := bthomeObjectSizes[id]
		if !ok {
			t.Errorf("object ID 0x%02X not in bthomeObjectSizes", id)
			continue
		}
		if actualSize != expectedSize {
			t.Errorf("bthomeObjectSizes[0x%02X] = %d, want %d", id, actualSize, expectedSize)
		}
	}
}

// Test mockBLEConnector implementation

func TestMockBLEConnector(t *testing.T) {
	connector := &mockBLEConnector{}

	if connector.IsConnected() {
		t.Error("should not be connected initially")
	}

	err := connector.Connect(context.Background(), "AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if !connector.IsConnected() {
		t.Error("should be connected after Connect()")
	}

	err = connector.Disconnect()
	if err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	if connector.IsConnected() {
		t.Error("should not be connected after Disconnect()")
	}
}
