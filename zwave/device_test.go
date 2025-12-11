package zwave

import (
	"testing"

	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

func TestNewDevice(t *testing.T) {
	profile := &profiles.Profile{
		Model:      "SNSW-001P16ZW",
		Name:       "Shelly Wave 1PM",
		Generation: types.Gen2,
		Series:     profiles.SeriesWave,
		Protocols: profiles.Protocols{
			ZWave: true,
		},
	}

	device := NewDevice(profile)

	if device == nil {
		t.Fatal("expected non-nil device")
	}
	if device.Profile != profile {
		t.Error("profile mismatch")
	}
}

func TestDevice_NilProfile(t *testing.T) {
	device := NewDevice(nil)

	if device.Model() != "" {
		t.Errorf("Model() = %q, want empty", device.Model())
	}
	if device.Name() != "" {
		t.Errorf("Name() = %q, want empty", device.Name())
	}
	if device.Generation() != types.GenUnknown {
		t.Errorf("Generation() = %v, want GenUnknown", device.Generation())
	}
	if device.IsZWave() {
		t.Error("IsZWave() = true, want false for nil profile")
	}
	if device.HasEthernet() {
		t.Error("HasEthernet() = true, want false for nil profile")
	}
	if device.HasWiFi() {
		t.Error("HasWiFi() = true, want false for nil profile")
	}
	if device.HasIPAccess() {
		t.Error("HasIPAccess() = true, want false for nil profile")
	}
	if device.IsPro() {
		t.Error("IsPro() = true, want false for nil profile")
	}
}

func TestDevice_WaveDevice(t *testing.T) {
	profile := &profiles.Profile{
		Model:      "SNSW-001P16ZW",
		Name:       "Shelly Wave 1PM",
		Generation: types.Gen2,
		Series:     profiles.SeriesWave,
		Protocols: profiles.Protocols{
			ZWave: true,
		},
	}

	device := NewDevice(profile)

	if device.Model() != "SNSW-001P16ZW" {
		t.Errorf("Model() = %q, want %q", device.Model(), "SNSW-001P16ZW")
	}
	if device.Name() != "Shelly Wave 1PM" {
		t.Errorf("Name() = %q, want %q", device.Name(), "Shelly Wave 1PM")
	}
	if device.Generation() != types.Gen2 {
		t.Errorf("Generation() = %v, want Gen2", device.Generation())
	}
	if !device.IsZWave() {
		t.Error("IsZWave() = false, want true")
	}
	if device.HasEthernet() {
		t.Error("HasEthernet() = true, want false for Wave device")
	}
	if device.HasWiFi() {
		t.Error("HasWiFi() = true, want false for Wave device")
	}
	if device.HasIPAccess() {
		t.Error("HasIPAccess() = true, want false for Z-Wave only")
	}
	if !device.SupportsLongRange() {
		t.Error("SupportsLongRange() = false, want true for Z-Wave device")
	}
	if device.IsPro() {
		t.Error("IsPro() = true, want false for Wave series")
	}
}

func TestDevice_WaveProDevice(t *testing.T) {
	profile := &profiles.Profile{
		Model:      "SPSW-001PE16ZW",
		Name:       "Shelly Wave Pro 1PM",
		Generation: types.Gen2,
		Series:     profiles.SeriesWavePro,
		Protocols: profiles.Protocols{
			ZWave:    true,
			Ethernet: true,
		},
	}

	device := NewDevice(profile)

	if !device.IsZWave() {
		t.Error("IsZWave() = false, want true")
	}
	if !device.HasEthernet() {
		t.Error("HasEthernet() = false, want true for Wave Pro")
	}
	if device.HasWiFi() {
		t.Error("HasWiFi() = true, want false")
	}
	if !device.HasIPAccess() {
		t.Error("HasIPAccess() = false, want true (has Ethernet)")
	}
	if !device.IsPro() {
		t.Error("IsPro() = false, want true for Wave Pro series")
	}
}

func TestDevice_NetworkInfo(t *testing.T) {
	profile := &profiles.Profile{
		Model:      "SNSW-001P16ZW",
		Name:       "Shelly Wave 1PM",
		Generation: types.Gen2,
		Protocols: profiles.Protocols{
			ZWave: true,
		},
	}

	device := NewDevice(profile)
	device.NodeID = 5
	device.HomeID = 0xA1B2C3D4
	device.Topology = TopologyMesh
	device.Security = SecurityS2Authenticated
	device.DSK = "12345"

	if device.NodeID != 5 {
		t.Errorf("NodeID = %d, want 5", device.NodeID)
	}
	if device.HomeID != 0xA1B2C3D4 {
		t.Errorf("HomeID = %x, want A1B2C3D4", device.HomeID)
	}
	if device.Topology != TopologyMesh {
		t.Errorf("Topology = %q, want %q", device.Topology, TopologyMesh)
	}
	if device.Security != SecurityS2Authenticated {
		t.Errorf("Security = %q, want %q", device.Security, SecurityS2Authenticated)
	}
	if device.DSK != "12345" {
		t.Errorf("DSK = %q, want %q", device.DSK, "12345")
	}
}

func TestNetworkTopology_Constants(t *testing.T) {
	tests := []struct {
		topology NetworkTopology
		want     string
	}{
		{TopologyMesh, "mesh"},
		{TopologyLongRange, "long_range"},
	}

	for _, tt := range tests {
		if string(tt.topology) != tt.want {
			t.Errorf("topology %v = %q, want %q", tt.topology, tt.topology, tt.want)
		}
	}
}

func TestSecurityLevel_Constants(t *testing.T) {
	tests := []struct {
		security SecurityLevel
		want     string
	}{
		{SecurityS2Authenticated, "s2_authenticated"},
		{SecurityS2Unauthenticated, "s2_unauthenticated"},
		{SecurityUnsecure, "unsecure"},
	}

	for _, tt := range tests {
		if string(tt.security) != tt.want {
			t.Errorf("security %v = %q, want %q", tt.security, tt.security, tt.want)
		}
	}
}
