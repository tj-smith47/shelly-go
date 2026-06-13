package discovery

import (
	"testing"
	"time"
)

// ────────────────────────────────────────────────────────────────────────────
// WithBLE option
// ────────────────────────────────────────────────────────────────────────────

func TestWithBLE_NilScanner(t *testing.T) {
	s := NewScanner(WithBLE(nil))
	if s.enableBLE {
		t.Error("enableBLE must be false when nil scanner is provided")
	}
	if s.ble != nil {
		t.Error("ble must be nil when nil scanner is provided")
	}
}

func TestWithBLE_RealScanner(t *testing.T) {
	mock := newMockBLEScanner()
	s := NewScanner(WithBLE(mock))
	if !s.enableBLE {
		t.Error("enableBLE must be true when a scanner is provided")
	}
	if s.ble == nil {
		t.Error("ble discoverer must be set when a scanner is provided")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// WithWiFi option
// ────────────────────────────────────────────────────────────────────────────

func TestWithWiFi_NilScanner(t *testing.T) {
	s := NewScanner(WithWiFi(nil))
	if s.enableWiFi {
		t.Error("enableWiFi must be false when nil scanner is provided")
	}
}

func TestWithWiFi_RealScanner(t *testing.T) {
	mock := &mockWiFiScanner{}
	s := NewScanner(WithWiFi(mock))
	if !s.enableWiFi {
		t.Error("enableWiFi must be true when a scanner is provided")
	}
	if s.wifi == nil {
		t.Error("wifi discoverer must be set when a scanner is provided")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Scanner.Scan — BLE path
// ────────────────────────────────────────────────────────────────────────────

func TestScanner_Scan_WithBLE(t *testing.T) {
	// Provide a mock BLE scanner that delivers one Shelly advertisement.
	adv := &BLEAdvertisement{
		Address:     "AA:BB:CC:DD:EE:99",
		LocalName:   "SHELLY-PLUS1-099",
		RSSI:        -55,
		ServiceData: make(map[string][]byte),
	}
	mock := newMockBLEScanner(adv)
	s := NewScanner(
		WithMDNS(false),
		WithCoIoT(false),
		WithBLE(mock),
	)

	devices, err := s.Scan(200 * time.Millisecond)
	if err != nil {
		t.Fatalf("Scan() with BLE error = %v", err)
	}
	if len(devices) == 0 {
		t.Error("expected at least one BLE device in scan results")
	}
}

func TestScanner_Scan_WithBLE_StartError(t *testing.T) {
	// A scanner whose Start always fails must not crash Scan — errors are ignored.
	errScanner := &mockBLEScanner{startErr: &BLEError{Message: "BT unavailable"}}
	s := NewScanner(
		WithMDNS(false),
		WithCoIoT(false),
		WithBLE(errScanner),
	)

	devices, err := s.Scan(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Scan() must not return error when BLE discovery fails; got: %v", err)
	}
	// No devices found (all failed), but must not be nil.
	if devices == nil {
		t.Error("Scan should return empty slice, not nil")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Scanner.Scan — WiFi path
// ────────────────────────────────────────────────────────────────────────────

func TestScanner_Scan_WithWiFi(t *testing.T) {
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

	devices, err := s.Scan(200 * time.Millisecond)
	if err != nil {
		t.Fatalf("Scan() with WiFi error = %v", err)
	}
	if len(devices) == 0 {
		t.Error("expected at least one WiFi device in scan results")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Scanner.Stop — with BLE and WiFi discoverers attached
// ────────────────────────────────────────────────────────────────────────────

func TestScanner_Stop_WithBLEAndWiFi(t *testing.T) {
	bleMock := newMockBLEScanner()
	wifiMock := &mockWiFiScanner{}
	s := NewScanner(WithBLE(bleMock), WithWiFi(wifiMock))

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop() with BLE+WiFi discoverers returned error: %v", err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Scanner.Scan — BLE enabled but ble pointer is nil (edge case guard)
// ────────────────────────────────────────────────────────────────────────────

func TestScanner_Scan_BLEEnabledNilPointer(t *testing.T) {
	s := NewScanner(WithMDNS(false), WithCoIoT(false))
	s.enableBLE = true
	s.ble = nil // simulate a corrupt state

	// Must not panic.
	devices, err := s.Scan(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("Scan must not error for nil ble pointer: %v", err)
	}
	if devices == nil {
		t.Error("expected non-nil slice")
	}
}

func TestScanner_Scan_WiFiEnabledNilPointer(t *testing.T) {
	s := NewScanner(WithMDNS(false), WithCoIoT(false))
	s.enableWiFi = true
	s.wifi = nil

	devices, err := s.Scan(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("Scan must not error for nil wifi pointer: %v", err)
	}
	if devices == nil {
		t.Error("expected non-nil slice")
	}
}
