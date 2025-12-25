package types

import "testing"

func TestModelDisplayName(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		// Gen1
		{"SHSW-1", "Shelly 1"},
		{"SHSW-PM", "Shelly 1PM"},
		{"SHSW-25", "Shelly 2.5"},
		{"SHPLG-S", "Shelly Plug S"},
		{"SHEM-3", "Shelly 3EM"},

		// Gen2 Plus
		{"SNSW-001P16EU", "Shelly Plus 1"},
		{"SNSW-102P16EU", "Shelly Plus 1PM"},
		{"SNSW-002P16EU", "Shelly Plus 2PM"},
		{"SNPL-00112EU", "Shelly Plus Plug S"},

		// Gen2 Pro
		{"SPSW-001PE16EU", "Shelly Pro 1PM"},
		{"SPSW-004PE16EU", "Shelly Pro 4PM"},
		{"SPEM-003CEBEU", "Shelly Pro 3EM"},

		// Gen3
		{"S3SW-001P16EU", "Shelly 1 Gen3"},
		{"S3SW-002P16EU", "Shelly 1PM Gen3"},

		// BLU
		{"SBBT-002C", "Shelly BLU Button1"},
		{"SBDW-002C", "Shelly BLU Door/Window"},

		// Unknown returns original
		{"UNKNOWN-MODEL", "UNKNOWN-MODEL"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := ModelDisplayName(tt.model)
			if got != tt.expected {
				t.Errorf("ModelDisplayName(%q) = %q, want %q", tt.model, got, tt.expected)
			}
		})
	}
}

func TestIsKnownModel(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"SHSW-1", true},
		{"SNSW-102P16EU", true},
		{"UNKNOWN", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := IsKnownModel(tt.model)
			if got != tt.expected {
				t.Errorf("IsKnownModel(%q) = %v, want %v", tt.model, got, tt.expected)
			}
		})
	}
}

func TestGetModelCategory(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		// Switches
		{"SHSW-1", "switch"},
		{"SNSW-102P16EU", "switch"},
		{"SPSW-004PE16EU", "switch"},
		{"S3SW-001P16EU", "switch"},

		// Dimmers
		{"SHDM-1", "dimmer"},
		{"SNDM-00100WW", "dimmer"},

		// Plugs
		{"SHPLG-S", "plug"},
		{"SNPL-00112EU", "plug"},
		{"S3PL-00112EU", "plug"},

		// Meters
		{"SHEM-3", "meter"},
		{"SPEM-003CEBEU", "meter"},

		// Sensors
		{"SHDW-1", "sensor"},
		{"SHMOS-01", "sensor"},
		{"SBMO-003Z", "sensor"},

		// Bulbs
		{"SHBLB-1", "bulb"},
		{"SHBDUO-1", "bulb"},

		// Unknown
		{"UNKNOWN", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := GetModelCategory(tt.model)
			if got != tt.expected {
				t.Errorf("GetModelCategory(%q) = %q, want %q", tt.model, got, tt.expected)
			}
		})
	}
}
