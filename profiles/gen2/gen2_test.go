package gen2

import (
	"testing"

	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

func TestGen2PlusProfilesRegistered(t *testing.T) {
	gen2Models := []string{
		"SNSW-001X16EU", // Shelly Plus 1
		"SNSW-001P16EU", // Shelly Plus 1PM
		"SNSW-001X8EU",  // Shelly Plus 1 Mini
		"SNSW-001P8EU",  // Shelly Plus 1PM Mini
		"SNSW-002P16EU", // Shelly Plus 2PM
		"SNPL-00112EU",  // Shelly Plus Plug S
		"SNPL-00116US",  // Shelly Plus Plug US
		"SNPL-00110IT",  // Shelly Plus Plug IT
		"SNPL-00112UK",  // Shelly Plus Plug UK
		"SNDM-0013US",   // Shelly Plus Wall Dimmer
		"SNDM-00100WW",  // Shelly Plus 0-10V Dimmer
		"SNDC-0D4P10WW", // Shelly Plus RGBW PM
		"SNSN-0024X",    // Shelly Plus i4
		"SNSN-0D24X",    // Shelly Plus i4 DC
		"SNUN-001",      // Shelly Plus UNI
		"SNSN-0013A",    // Shelly Plus H&T
		"SNSN-0031Z",    // Shelly Plus Smoke
		"SNPM-001PCEU",  // Shelly Plus PM Mini
	}

	for _, model := range gen2Models {
		p, ok := profiles.Get(model)
		if !ok {
			t.Errorf("Gen2 Plus profile %s not found", model)
			continue
		}
		if p.Generation != types.Gen2 {
			t.Errorf("Profile %s has wrong generation: %v", model, p.Generation)
		}
		if !p.Protocols.HTTP {
			t.Errorf("Gen2 profile %s should support HTTP", model)
		}
		if !p.Protocols.WebSocket {
			t.Errorf("Gen2 profile %s should support WebSocket", model)
		}
	}
}

func TestProProfilesRegistered(t *testing.T) {
	proModels := []string{
		"SPSW-001XE16EU",  // Shelly Pro 1
		"SPSW-001PE16EU",  // Shelly Pro 1PM
		"SPSW-002XE16EU",  // Shelly Pro 2
		"SPSW-002PE16EU",  // Shelly Pro 2PM
		"SPSW-003XE16EU",  // Shelly Pro 3
		"SPEM-003CEBEU",   // Shelly Pro 3EM
		"SPSW-004PE16EU",  // Shelly Pro 4PM
		"SPDM-001PE01EU",  // Shelly Pro Dimmer 1PM
		"SPDM-002PE01EU",  // Shelly Pro Dimmer 2PM
		"SPSH-002PE16EU",  // Shelly Pro Dual Cover PM
		"SPEM-002CEBEU50", // Shelly Pro EM-50
	}

	for _, model := range proModels {
		p, ok := profiles.Get(model)
		if !ok {
			t.Errorf("Pro profile %s not found", model)
			continue
		}
		if p.Series != profiles.SeriesPro {
			t.Errorf("Profile %s should be Pro series: %v", model, p.Series)
		}
		if !p.Protocols.Ethernet {
			t.Errorf("Pro profile %s should support Ethernet", model)
		}
	}
}

func TestMergeGen2Caps(t *testing.T) {
	caps := mergeGen2Caps(profiles.Capabilities{
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
	if !caps.Webhooks {
		t.Error("Should have Webhooks (from defaults)")
	}
}

func TestMergeGen2Limits(t *testing.T) {
	limits := mergeGen2Limits(&profiles.Limits{
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
	if limits.MaxSchedules != 20 {
		t.Errorf("MaxSchedules should be 20 (from defaults)")
	}
}
