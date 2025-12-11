package blu

import (
	"testing"

	"github.com/tj-smith47/shelly-go/profiles"
)

func TestBLUProfilesRegistered(t *testing.T) {
	bluModels := []string{
		"SBBT-002C", // BLU Button1
		"SBDW-002C", // BLU Door/Window
		"SBMO-003Z", // BLU Motion
		"SBHT-003C", // BLU H&T
		"SBBT-004C", // BLU RC Button 4
		"SBTRV-001", // BLU TRV
		"SNGW-BT01", // BLU Gateway
	}

	for _, model := range bluModels {
		p, ok := profiles.Get(model)
		if !ok {
			t.Errorf("BLU profile %s not found", model)
			continue
		}
		if p.Series != profiles.SeriesBLU {
			t.Errorf("Profile %s should be BLU series: %v", model, p.Series)
		}
		if !p.Protocols.BLE {
			t.Errorf("BLU profile %s should support BLE", model)
		}
	}
}

func TestBLUButtonProfile(t *testing.T) {
	p, ok := profiles.Get("SBBT-002C")
	if !ok {
		t.Fatal("BLU Button profile not found")
	}

	if p.Components.Inputs != 1 {
		t.Errorf("Inputs = %d, want 1", p.Components.Inputs)
	}
	if !p.Capabilities.InputEvents {
		t.Error("Should have InputEvents")
	}
	if p.PowerSource != profiles.PowerSourceBattery {
		t.Errorf("PowerSource = %v, want Battery", p.PowerSource)
	}
}

func TestBLUGatewayProfile(t *testing.T) {
	p, ok := profiles.Get("SNGW-BT01")
	if !ok {
		t.Fatal("BLU Gateway profile not found")
	}

	if p.PowerSource != profiles.PowerSourceMains {
		t.Errorf("PowerSource = %v, want Mains", p.PowerSource)
	}
	if !p.Capabilities.Scripting {
		t.Error("Gateway should support Scripting")
	}
	if !p.Protocols.HTTP {
		t.Error("Gateway should support HTTP")
	}
}
