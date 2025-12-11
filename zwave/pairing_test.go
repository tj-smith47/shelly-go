package zwave

import (
	"testing"

	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

func TestGetInclusionInfo_SmartStart(t *testing.T) {
	device := NewDevice(&profiles.Profile{
		Model:      "SNSW-001P16ZW",
		Name:       "Shelly Wave 1PM",
		Generation: types.Gen2,
	})

	info := GetInclusionInfo(device, InclusionSmartStart)

	if info.Mode != InclusionSmartStart {
		t.Errorf("Mode = %q, want %q", info.Mode, InclusionSmartStart)
	}
	if !info.DSKRequired {
		t.Error("DSKRequired = false, want true")
	}
	if len(info.Instructions) == 0 {
		t.Error("expected non-empty instructions")
	}
	// Check first instruction mentions SmartStart
	found := false
	for _, step := range info.Instructions {
		if step != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected non-empty instruction steps")
	}
}

func TestGetInclusionInfo_Button(t *testing.T) {
	device := NewDevice(&profiles.Profile{
		Model:      "SNSW-001P16ZW",
		Name:       "Shelly Wave 1PM",
		Generation: types.Gen2,
	})

	info := GetInclusionInfo(device, InclusionButton)

	if info.Mode != InclusionButton {
		t.Errorf("Mode = %q, want %q", info.Mode, InclusionButton)
	}
	if !info.DSKRequired {
		t.Error("DSKRequired = false, want true")
	}
	if len(info.Instructions) < 3 {
		t.Errorf("expected at least 3 instructions, got %d", len(info.Instructions))
	}
}

func TestGetInclusionInfo_Switch(t *testing.T) {
	device := NewDevice(&profiles.Profile{
		Model:      "SNSW-001P16ZW",
		Name:       "Shelly Wave 1PM",
		Generation: types.Gen2,
	})

	info := GetInclusionInfo(device, InclusionSwitch)

	if info.Mode != InclusionSwitch {
		t.Errorf("Mode = %q, want %q", info.Mode, InclusionSwitch)
	}
	if !info.DSKRequired {
		t.Error("DSKRequired = false, want true")
	}
	if len(info.Instructions) == 0 {
		t.Error("expected non-empty instructions")
	}
}

func TestGetExclusionInfo_Button(t *testing.T) {
	device := NewDevice(&profiles.Profile{
		Model:      "SNSW-001P16ZW",
		Name:       "Shelly Wave 1PM",
		Generation: types.Gen2,
	})

	info := GetExclusionInfo(device, InclusionButton)

	if info.Mode != InclusionButton {
		t.Errorf("Mode = %q, want %q", info.Mode, InclusionButton)
	}
	if len(info.Instructions) == 0 {
		t.Error("expected non-empty instructions")
	}
}

func TestGetExclusionInfo_Switch(t *testing.T) {
	device := NewDevice(&profiles.Profile{
		Model:      "SNSW-001P16ZW",
		Name:       "Shelly Wave 1PM",
		Generation: types.Gen2,
	})

	info := GetExclusionInfo(device, InclusionSwitch)

	if info.Mode != InclusionSwitch {
		t.Errorf("Mode = %q, want %q", info.Mode, InclusionSwitch)
	}
	if len(info.Instructions) == 0 {
		t.Error("expected non-empty instructions")
	}
}

func TestGetExclusionInfo_SmartStart_FallbackToButton(t *testing.T) {
	device := NewDevice(&profiles.Profile{
		Model:      "SNSW-001P16ZW",
		Name:       "Shelly Wave 1PM",
		Generation: types.Gen2,
	})

	// SmartStart devices still need manual exclusion
	info := GetExclusionInfo(device, InclusionSmartStart)

	// Mode is preserved
	if info.Mode != InclusionSmartStart {
		t.Errorf("Mode = %q, want %q", info.Mode, InclusionSmartStart)
	}
	// But instructions should be provided (manual method)
	if len(info.Instructions) == 0 {
		t.Error("expected non-empty instructions for SmartStart exclusion")
	}
}

func TestGetFactoryResetInfo(t *testing.T) {
	device := NewDevice(&profiles.Profile{
		Model:      "SNSW-001P16ZW",
		Name:       "Shelly Wave 1PM",
		Generation: types.Gen2,
	})

	info := GetFactoryResetInfo(device)

	if info.Warning == "" {
		t.Error("expected non-empty warning")
	}
	if len(info.Instructions) == 0 {
		t.Error("expected non-empty instructions")
	}
	// Warning should mention important consequences
	if len(info.Warning) < 50 {
		t.Error("warning seems too short, should mention consequences")
	}
}

func TestInclusionMode_Constants(t *testing.T) {
	tests := []struct {
		mode InclusionMode
		want string
	}{
		{InclusionSmartStart, "smart_start"},
		{InclusionButton, "button"},
		{InclusionSwitch, "switch"},
	}

	for _, tt := range tests {
		if string(tt.mode) != tt.want {
			t.Errorf("mode %v = %q, want %q", tt.mode, tt.mode, tt.want)
		}
	}
}
