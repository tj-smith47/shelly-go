package gen1

import (
	"testing"

	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

func TestGen1ProfilesRegistered(t *testing.T) {
	gen1Models := []string{
		"SHSW-1",   // Shelly 1
		"SHSW-PM",  // Shelly 1PM
		"SHSW-L",   // Shelly 1L
		"SHSW-21",  // Shelly 2
		"SHSW-25",  // Shelly 2.5
		"SHSW-44",  // Shelly 4Pro
		"SHPLG-1",  // Shelly Plug
		"SHPLG-S",  // Shelly Plug S
		"SHPLG-U1", // Shelly Plug US
		"SHDM-1",   // Shelly Dimmer
		"SHDM-2",   // Shelly Dimmer 2
		"SHRGBW2",  // Shelly RGBW2
		"SHBDUO-1", // Shelly Duo
		"SHVIN-1",  // Shelly Vintage
		"SHBLB-1",  // Shelly Bulb
		"SHEM",     // Shelly EM
		"SHEM-3",   // Shelly 3EM
		"SHIX3-1",  // Shelly i3
		"SHBTN-1",  // Shelly Button1
		"SHUNI-1",  // Shelly UNI
		"SHHT-1",   // Shelly H&T
		"SHWT-1",   // Shelly Flood
		"SHDW-2",   // Shelly Door/Window 2
		"SHGS-1",   // Shelly Gas
		"SHSM-01",  // Shelly Smoke
		"SHMOS-01", // Shelly Motion
		"SHTRV-01", // Shelly TRV
	}

	for _, model := range gen1Models {
		p, ok := profiles.Get(model)
		if !ok {
			t.Errorf("Gen1 profile %s not found", model)
			continue
		}
		if p.Generation != types.Gen1 {
			t.Errorf("Profile %s has wrong generation: %v", model, p.Generation)
		}
		if !p.Protocols.HTTP {
			t.Errorf("Gen1 profile %s should support HTTP", model)
		}
		if !p.Protocols.CoIoT {
			t.Errorf("Gen1 profile %s should support CoIoT", model)
		}
	}
}

func TestMergeGen1Caps(t *testing.T) {
	caps := mergeGen1Caps(profiles.Capabilities{
		PowerMetering: true,
		CoverSupport:  true,
	})

	if !caps.PowerMetering {
		t.Error("Should have PowerMetering")
	}
	if !caps.CoverSupport {
		t.Error("Should have CoverSupport")
	}
	if !caps.Schedules {
		t.Error("Should have Schedules (from defaults)")
	}
	if !caps.Actions {
		t.Error("Should have Actions (from defaults)")
	}
}

func TestMergeGen1Limits(t *testing.T) {
	limits := mergeGen1Limits(&profiles.Limits{
		MaxPower:         1000,
		MaxOutputCurrent: 10,
	})

	if limits.MaxPower != 1000 {
		t.Errorf("MaxPower = %v, want 1000", limits.MaxPower)
	}
	if limits.MaxOutputCurrent != 10 {
		t.Errorf("MaxOutputCurrent = %v, want 10", limits.MaxOutputCurrent)
	}
	if limits.MaxSchedules != 10 {
		t.Errorf("MaxSchedules should be 10 (from defaults)")
	}
}
