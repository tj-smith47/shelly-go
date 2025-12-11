package discovery

import (
	"testing"
)

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
				0x02, 0xE8, 0x03, // Temperature 10.00°C (1000 = 0x03E8)
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
				0x02, 0x18, 0xFC, // Temperature -10.00°C (-1000 as int16)
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
				0x3F, 0x84, 0x03, // Rotation 90.0° (900 = 0x0384)
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
				0x3F, 0x7C, 0xFC, // Rotation -90.0° (-900 as int16)
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
				0x02, 0xE8, 0x03, // Temperature 10.00°C
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
