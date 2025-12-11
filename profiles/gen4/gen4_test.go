package gen4

import (
	"testing"

	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

func TestGen4ProfilesRegistered(t *testing.T) {
	gen4Models := []string{
		"S4SW-001X16EU", // Shelly 1 Gen4
		"S4SW-001P16EU", // Shelly 1PM Gen4
		"S4SW-002P16EU", // Shelly 2PM Gen4
		"S4SW-001X8EU",  // Shelly 1 Mini Gen4
		"S4SW-001P8EU",  // Shelly 1PM Mini Gen4
		"S4EM-001XCEU",  // Shelly EM Mini Gen4
		"S4PL-00116US",  // Shelly Plug US Gen4
		"S4SN-0W1X",     // Shelly Flood Sensor Gen4
		"S4WD-00695EU",  // Shelly Wall Display X2
	}

	for _, model := range gen4Models {
		p, ok := profiles.Get(model)
		if !ok {
			t.Errorf("Gen4 profile %s not found", model)
			continue
		}
		if p.Generation != types.Gen4 {
			t.Errorf("Profile %s has wrong generation: %v, want Gen4", model, p.Generation)
		}
		if !p.Protocols.Matter {
			t.Errorf("Gen4 profile %s should support Matter", model)
		}
		if !p.Protocols.Zigbee {
			t.Errorf("Gen4 profile %s should support Zigbee", model)
		}
	}
}

func TestMergeGen4Caps(t *testing.T) {
	caps := mergeGen4Caps(profiles.Capabilities{
		PowerMetering: true,
		CoverSupport:  true,
	})

	if !caps.PowerMetering {
		t.Error("Should have PowerMetering")
	}
	if !caps.CoverSupport {
		t.Error("Should have CoverSupport")
	}
	if !caps.Scripting {
		t.Error("Should have Scripting (from defaults)")
	}
}

func TestMergeGen4Limits(t *testing.T) {
	limits := mergeGen4Limits(&profiles.Limits{
		MaxPower:         3500,
		MaxOutputCurrent: 16,
	})

	if limits.MaxPower != 3500 {
		t.Errorf("MaxPower = %v, want 3500", limits.MaxPower)
	}
	if limits.MaxOutputCurrent != 16 {
		t.Errorf("MaxOutputCurrent = %v, want 16", limits.MaxOutputCurrent)
	}
	if limits.MaxScripts != 10 {
		t.Errorf("MaxScripts should be 10 (from defaults)")
	}
}

func TestMergeGen4CapsAllOptions(t *testing.T) {
	// Test all capability merging branches
	caps := mergeGen4Caps(profiles.Capabilities{
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
	if !caps.Scripting {
		t.Error("Should have Scripting (from defaults)")
	}
}

func TestMergeGen4LimitsAllOptions(t *testing.T) {
	// Test all limit merging branches
	limits := mergeGen4Limits(&profiles.Limits{
		MaxInputCurrent:  100,
		MaxOutputCurrent: 16,
		MaxPower:         3500,
		MaxVoltage:       240,
		MinVoltage:       100,
	})

	if limits.MaxInputCurrent != 100 {
		t.Errorf("MaxInputCurrent = %v, want 100", limits.MaxInputCurrent)
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

func TestGen4WallDisplayProfile(t *testing.T) {
	p, ok := profiles.Get("S4WD-00695EU")
	if !ok {
		t.Fatal("Wall Display X2 profile not found")
	}

	if !p.Components.Display {
		t.Error("Should have Display component")
	}
	if p.Components.Switches != 2 {
		t.Errorf("Switches = %d, want 2", p.Components.Switches)
	}
	if p.Components.TemperatureSensors != 1 {
		t.Errorf("TemperatureSensors = %d, want 1", p.Components.TemperatureSensors)
	}
	if p.Components.HumiditySensors != 1 {
		t.Errorf("HumiditySensors = %d, want 1", p.Components.HumiditySensors)
	}
}

func TestGen4FloodSensorProfile(t *testing.T) {
	p, ok := profiles.Get("S4SN-0W1X")
	if !ok {
		t.Fatal("Flood Sensor Gen4 profile not found")
	}

	if p.PowerSource != profiles.PowerSourceBattery {
		t.Errorf("PowerSource = %v, want Battery", p.PowerSource)
	}
	if len(p.Sensors) != 3 {
		t.Errorf("Sensors count = %d, want 3", len(p.Sensors))
	}
}
