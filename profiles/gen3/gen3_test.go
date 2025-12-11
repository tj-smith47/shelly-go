package gen3

import (
	"testing"

	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

func TestGen3ProfilesRegistered(t *testing.T) {
	gen3Models := []string{
		"S3SW-001X16EU",   // Shelly 1 Gen3
		"S3SW-001P16EU",   // Shelly 1PM Gen3
		"S3SW-001L16EU",   // Shelly 1L Gen3
		"S3SW-002P16EU",   // Shelly 2PM Gen3
		"S3SW-002L16EU",   // Shelly 2L Gen3
		"S3SW-001X8EU",    // Shelly 1 Mini Gen3
		"S3SW-001P8EU",    // Shelly 1PM Mini Gen3
		"S3PM-001PCEU",    // Shelly PM Mini Gen3
		"S3SW-002PCEU",    // Shelly Shutter Gen3
		"S3PL-00112EU",    // Shelly Plug S Gen3
		"S3PL-00112EUMTR", // Shelly Plug S MTR Gen3
		"S3PL-00212EU",    // Shelly Outdoor Plug S Gen3
		"S3PL-10112EU",    // Shelly Plug PM Gen3
		"S3DM-0010WW",     // Shelly Dimmer Gen3
		"S3DM-0D10WW",     // Shelly Dimmer 0/1-10V PM Gen3
		"S3DM-0D0WW",      // Shelly DALI Dimmer Gen3
		"S3SN-0024X",      // Shelly i4 Gen3
		"S3SN-0U12A",      // Shelly H&T Gen3
		"S3EM-002CXCEU",   // Shelly EM Gen3
		"S3EM-003CXCEU",   // Shelly 3EM-63 Gen3
	}

	for _, model := range gen3Models {
		p, ok := profiles.Get(model)
		if !ok {
			t.Errorf("Gen3 profile %s not found", model)
			continue
		}
		if p.Generation != types.Gen3 {
			t.Errorf("Profile %s has wrong generation: %v, want Gen3", model, p.Generation)
		}
		if !p.Protocols.HTTP {
			t.Errorf("Gen3 profile %s should support HTTP", model)
		}
		if !p.Protocols.WebSocket {
			t.Errorf("Gen3 profile %s should support WebSocket", model)
		}
	}
}

func TestMergeGen3Caps(t *testing.T) {
	caps := mergeGen3Caps(profiles.Capabilities{
		PowerMetering: true,
		NoNeutral:     true,
	})

	if !caps.PowerMetering {
		t.Error("Should have PowerMetering")
	}
	if !caps.NoNeutral {
		t.Error("Should have NoNeutral")
	}
	if !caps.Scripting {
		t.Error("Should have Scripting (from defaults)")
	}
}

func TestMergeGen3Limits(t *testing.T) {
	limits := mergeGen3Limits(&profiles.Limits{
		MaxPower:   2300,
		MinVoltage: 100,
		MaxVoltage: 240,
	})

	if limits.MaxPower != 2300 {
		t.Errorf("MaxPower = %v, want 2300", limits.MaxPower)
	}
	if limits.MinVoltage != 100 {
		t.Errorf("MinVoltage = %v, want 100", limits.MinVoltage)
	}
}

func TestMergeGen3CapsAllOptions(t *testing.T) {
	// Test all capability merging branches
	caps := mergeGen3Caps(profiles.Capabilities{
		PowerMetering:         true,
		EnergyMetering:        true,
		CoverSupport:          true,
		DimmingSupport:        true,
		ColorSupport:          true,
		ColorTemperature:      true,
		SensorAddon:           true,
		ExternalSensors:       true,
		Calibration:           true,
		Effects:               true,
		NoNeutral:             true,
		BidirectionalMetering: true,
		ThreePhase:            true,
	})

	if !caps.PowerMetering {
		t.Error("Should have PowerMetering")
	}
	if !caps.EnergyMetering {
		t.Error("Should have EnergyMetering")
	}
	if !caps.CoverSupport {
		t.Error("Should have CoverSupport")
	}
	if !caps.DimmingSupport {
		t.Error("Should have DimmingSupport")
	}
	if !caps.ColorSupport {
		t.Error("Should have ColorSupport")
	}
	if !caps.ColorTemperature {
		t.Error("Should have ColorTemperature")
	}
	if !caps.SensorAddon {
		t.Error("Should have SensorAddon")
	}
	if !caps.ExternalSensors {
		t.Error("Should have ExternalSensors")
	}
	if !caps.Calibration {
		t.Error("Should have Calibration")
	}
	if !caps.Effects {
		t.Error("Should have Effects")
	}
	if !caps.NoNeutral {
		t.Error("Should have NoNeutral")
	}
	if !caps.BidirectionalMetering {
		t.Error("Should have BidirectionalMetering")
	}
	if !caps.ThreePhase {
		t.Error("Should have ThreePhase")
	}
}

func TestMergeGen3LimitsAllOptions(t *testing.T) {
	// Test all limit merging branches
	limits := mergeGen3Limits(&profiles.Limits{
		MaxInputCurrent:  50,
		MaxOutputCurrent: 16,
		MaxPower:         3500,
		MaxVoltage:       240,
		MinVoltage:       100,
	})

	if limits.MaxInputCurrent != 50 {
		t.Errorf("MaxInputCurrent = %v, want 50", limits.MaxInputCurrent)
	}
	if limits.MaxOutputCurrent != 16 {
		t.Errorf("MaxOutputCurrent = %v, want 16", limits.MaxOutputCurrent)
	}
	if limits.MaxPower != 3500 {
		t.Errorf("MaxPower = %v, want 3500", limits.MaxPower)
	}
	if limits.MaxVoltage != 240 {
		t.Errorf("MaxVoltage = %v, want 240", limits.MaxVoltage)
	}
	if limits.MinVoltage != 100 {
		t.Errorf("MinVoltage = %v, want 100", limits.MinVoltage)
	}
}

func TestGen31LProfile(t *testing.T) {
	p, ok := profiles.Get("S3SW-001L16EU")
	if !ok {
		t.Fatal("1L Gen3 profile not found")
	}

	if !p.Capabilities.NoNeutral {
		t.Error("Should have NoNeutral capability")
	}
}
