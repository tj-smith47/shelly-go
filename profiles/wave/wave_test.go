package wave

import (
	"testing"

	"github.com/tj-smith47/shelly-go/profiles"
)

func TestWaveProfilesRegistered(t *testing.T) {
	waveModels := []string{
		"SNSW-001X16ZW",  // Wave 1
		"SNSW-001P16ZW",  // Wave 1PM
		"SNSW-002P16ZW",  // Wave 2PM
		"SNPL-00116USZW", // Wave Plug US
		"SNSW-102P16ZW",  // Wave Shutter
	}

	for _, model := range waveModels {
		p, ok := profiles.Get(model)
		if !ok {
			t.Errorf("Wave profile %s not found", model)
			continue
		}
		if p.Series != profiles.SeriesWave {
			t.Errorf("Profile %s should be Wave series: %v", model, p.Series)
		}
		if !p.Protocols.ZWave {
			t.Errorf("Wave profile %s should support Z-Wave", model)
		}
	}
}

func TestWaveProProfilesRegistered(t *testing.T) {
	waveProModels := []string{
		"SPSW-001XE16ZW", // Wave Pro 1
		"SPSW-001PE16ZW", // Wave Pro 1PM
		"SPSW-002XE16ZW", // Wave Pro 2
		"SPSW-002PE16ZW", // Wave Pro 2PM
		"SPSW-003XE16ZW", // Wave Pro 3
	}

	for _, model := range waveProModels {
		p, ok := profiles.Get(model)
		if !ok {
			t.Errorf("Wave Pro profile %s not found", model)
			continue
		}
		if p.Series != profiles.SeriesWavePro {
			t.Errorf("Profile %s should be WavePro series: %v", model, p.Series)
		}
		if !p.Protocols.ZWave {
			t.Errorf("Wave Pro profile %s should support Z-Wave", model)
		}
		if !p.Protocols.Ethernet {
			t.Errorf("Wave Pro profile %s should support Ethernet", model)
		}
	}
}

func TestWave2PMProfile(t *testing.T) {
	p, ok := profiles.Get("SNSW-002P16ZW")
	if !ok {
		t.Fatal("Wave 2PM profile not found")
	}

	if p.Components.Switches != 2 {
		t.Errorf("Switches = %d, want 2", p.Components.Switches)
	}
	if p.Components.Covers != 1 {
		t.Errorf("Covers = %d, want 1", p.Components.Covers)
	}
	if !p.Capabilities.PowerMetering {
		t.Error("Should have PowerMetering")
	}
	if !p.Capabilities.CoverSupport {
		t.Error("Should have CoverSupport")
	}
}

func TestWaveShutterProfile(t *testing.T) {
	p, ok := profiles.Get("SNSW-102P16ZW")
	if !ok {
		t.Fatal("Wave Shutter profile not found")
	}

	if p.Components.Covers != 1 {
		t.Errorf("Covers = %d, want 1", p.Components.Covers)
	}
	if !p.Capabilities.CoverSupport {
		t.Error("Should have CoverSupport")
	}
	if !p.Capabilities.Calibration {
		t.Error("Should have Calibration")
	}
}
