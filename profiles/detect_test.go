package profiles

import (
	"encoding/json"
	"testing"

	"github.com/tj-smith47/shelly-go/types"
)

func TestDetectFromDeviceInfo(t *testing.T) {
	Clear()
	defer Clear()

	// Register a test profile
	profile := &Profile{
		Model:      "SNSW-001P16EU",
		Name:       "Shelly Plus 1PM",
		App:        "Plus1PM",
		Generation: types.Gen2,
	}
	Register(profile)

	tests := []struct {
		info      *DeviceInfo
		name      string
		wantModel string
		wantGen   types.Generation
		wantFound bool
	}{
		{
			name: "Gen2 device found by model",
			info: &DeviceInfo{
				Model: "SNSW-001P16EU",
				Gen:   2,
				App:   "Plus1PM",
			},
			wantGen:   types.Gen2,
			wantModel: "SNSW-001P16EU",
			wantFound: true,
		},
		{
			name: "Gen3 device found by app",
			info: &DeviceInfo{
				Model: "UNKNOWN-MODEL",
				Gen:   3,
				App:   "Plus1PM",
			},
			wantGen:   types.Gen3,
			wantModel: "UNKNOWN-MODEL",
			wantFound: true,
		},
		{
			name: "Gen4 device not found",
			info: &DeviceInfo{
				Model: "S4SW-999",
				Gen:   4,
				App:   "UnknownApp",
			},
			wantGen:   types.Gen4,
			wantModel: "S4SW-999",
			wantFound: false,
		},
		{
			name: "Unknown generation",
			info: &DeviceInfo{
				Model: "TEST",
				Gen:   99,
			},
			wantGen:   types.GenerationUnknown,
			wantModel: "TEST",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFromDeviceInfo(tt.info)

			if result.Generation != tt.wantGen {
				t.Errorf("Generation = %v, want %v", result.Generation, tt.wantGen)
			}
			if result.Model != tt.wantModel {
				t.Errorf("Model = %v, want %v", result.Model, tt.wantModel)
			}
			if (result.Profile != nil) != tt.wantFound {
				t.Errorf("Profile found = %v, want %v", result.Profile != nil, tt.wantFound)
			}
		})
	}
}

func TestDetectFromGen1Status(t *testing.T) {
	Clear()
	defer Clear()

	// Register a test Gen1 profile
	profile := &Profile{
		Model:      "SHSW-1",
		Name:       "Shelly 1",
		Generation: types.Gen1,
	}
	Register(profile)

	tests := []struct {
		status    *Gen1Status
		name      string
		wantFound bool
	}{
		{
			name: "Gen1 device found",
			status: &Gen1Status{
				Type: "SHSW-1",
				MAC:  "001122334455",
			},
			wantFound: true,
		},
		{
			name: "Gen1 device not found",
			status: &Gen1Status{
				Type: "SHSW-UNKNOWN",
				MAC:  "001122334455",
			},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFromGen1Status(tt.status)

			if result.Generation != types.Gen1 {
				t.Errorf("Generation = %v, want Gen1", result.Generation)
			}
			if (result.Profile != nil) != tt.wantFound {
				t.Errorf("Profile found = %v, want %v", result.Profile != nil, tt.wantFound)
			}
		})
	}
}

func TestDetectFromModel(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{Model: "SHSW-1", Generation: types.Gen1})
	Register(&Profile{Model: "SNSW-001P16EU", Generation: types.Gen2})

	tests := []struct {
		model     string
		wantGen   types.Generation
		wantFound bool
	}{
		{"SHSW-1", types.Gen1, true},
		{"SNSW-001P16EU", types.Gen2, true},
		{"UNKNOWN", types.GenerationUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := DetectFromModel(tt.model)

			if result.Generation != tt.wantGen {
				t.Errorf("Generation = %v, want %v", result.Generation, tt.wantGen)
			}
			if (result.Profile != nil) != tt.wantFound {
				t.Errorf("Profile found = %v, want %v", result.Profile != nil, tt.wantFound)
			}
		})
	}
}

func TestDetectFromJSON(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{Model: "SHSW-1", Generation: types.Gen1})
	Register(&Profile{Model: "SNSW-001P16EU", App: "Plus1PM", Generation: types.Gen2})

	tests := []struct {
		name      string
		json      string
		wantGen   types.Generation
		wantFound bool
	}{
		{
			name:      "Gen2 device info",
			json:      `{"id":"abc123","model":"SNSW-001P16EU","gen":2,"app":"Plus1PM"}`,
			wantGen:   types.Gen2,
			wantFound: true,
		},
		{
			name:      "Gen1 status",
			json:      `{"type":"SHSW-1","mac":"001122334455","auth":false,"fw":"1.0"}`,
			wantGen:   types.Gen1,
			wantFound: true,
		},
		{
			name:      "Invalid JSON",
			json:      `{invalid}`,
			wantGen:   types.GenerationUnknown,
			wantFound: false,
		},
		{
			name:      "Empty object",
			json:      `{}`,
			wantGen:   types.GenerationUnknown,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFromJSON([]byte(tt.json))

			if result.Generation != tt.wantGen {
				t.Errorf("Generation = %v, want %v", result.Generation, tt.wantGen)
			}
			if (result.Profile != nil) != tt.wantFound {
				t.Errorf("Profile found = %v, want %v", result.Profile != nil, tt.wantFound)
			}
		})
	}
}

func TestMatchCapabilities(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{
		Model:        "PM-DEVICE",
		Capabilities: Capabilities{PowerMetering: true, EnergyMetering: true},
	})
	Register(&Profile{
		Model:        "COVER-DEVICE",
		Capabilities: Capabilities{CoverSupport: true, Calibration: true},
	})
	Register(&Profile{
		Model:        "FULL-DEVICE",
		Capabilities: Capabilities{PowerMetering: true, CoverSupport: true, Scripting: true},
	})

	// Match power metering
	results := MatchCapabilities(Capabilities{PowerMetering: true})
	if len(results) != 2 {
		t.Errorf("len(MatchCapabilities(PM)) = %d, want 2", len(results))
	}

	// Match cover support
	results = MatchCapabilities(Capabilities{CoverSupport: true})
	if len(results) != 2 {
		t.Errorf("len(MatchCapabilities(Cover)) = %d, want 2", len(results))
	}

	// Match both PM and Cover
	results = MatchCapabilities(Capabilities{PowerMetering: true, CoverSupport: true})
	if len(results) != 1 {
		t.Errorf("len(MatchCapabilities(PM+Cover)) = %d, want 1", len(results))
	}

	// No match
	results = MatchCapabilities(Capabilities{ThreePhase: true})
	if len(results) != 0 {
		t.Errorf("len(MatchCapabilities(ThreePhase)) = %d, want 0", len(results))
	}
}

func TestMatchCapabilities_AllFields(t *testing.T) {
	Clear()
	defer Clear()

	// Register a profile with many capabilities
	Register(&Profile{
		Model: "FULL",
		Capabilities: Capabilities{
			PowerMetering:    true,
			EnergyMetering:   true,
			CoverSupport:     true,
			DimmingSupport:   true,
			ColorSupport:     true,
			ColorTemperature: true,
			Scripting:        true,
			Schedules:        true,
			Webhooks:         true,
			ThreePhase:       true,
		},
	})

	// All requirements met
	results := MatchCapabilities(Capabilities{
		PowerMetering:  true,
		EnergyMetering: true,
		CoverSupport:   true,
	})
	if len(results) != 1 {
		t.Errorf("Should match full profile, got %d", len(results))
	}
}

func TestMatchComponents(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{
		Model:      "2SW",
		Components: Components{Switches: 2, Inputs: 2},
	})
	Register(&Profile{
		Model:      "4SW",
		Components: Components{Switches: 4, Inputs: 4, PowerMeters: 4},
	})
	Register(&Profile{
		Model:      "COVER",
		Components: Components{Covers: 1, Inputs: 2},
	})

	// At least 2 switches
	results := MatchComponents(&Components{Switches: 2})
	if len(results) != 2 {
		t.Errorf("len(MatchComponents(2 switches)) = %d, want 2", len(results))
	}

	// At least 4 switches
	results = MatchComponents(&Components{Switches: 4})
	if len(results) != 1 {
		t.Errorf("len(MatchComponents(4 switches)) = %d, want 1", len(results))
	}

	// Cover
	results = MatchComponents(&Components{Covers: 1})
	if len(results) != 1 {
		t.Errorf("len(MatchComponents(1 cover)) = %d, want 1", len(results))
	}

	// No match
	results = MatchComponents(&Components{Switches: 10})
	if len(results) != 0 {
		t.Errorf("len(MatchComponents(10 switches)) = %d, want 0", len(results))
	}
}

func TestMatchComponents_AllFields(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{
		Model: "FULL",
		Components: Components{
			Switches:           2,
			Covers:             1,
			Lights:             1,
			Inputs:             4,
			PowerMeters:        2,
			EnergyMeters:       1,
			TemperatureSensors: 2,
			HumiditySensors:    1,
		},
	})

	// Match some components
	results := MatchComponents(&Components{
		Switches:           2,
		Inputs:             4,
		TemperatureSensors: 1,
	})
	if len(results) != 1 {
		t.Errorf("Should match, got %d", len(results))
	}

	// Exceed limits
	results = MatchComponents(&Components{
		Switches: 3,
	})
	if len(results) != 0 {
		t.Errorf("Should not match, got %d", len(results))
	}
}

func TestFindSimilar(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{
		Model:      "SNSW-001P16EU",
		Name:       "Plus 1PM",
		Generation: types.Gen2,
		FormFactor: FormFactorFlush,
		Components: Components{Switches: 1},
	})
	Register(&Profile{
		Model:      "SNSW-001X16EU",
		Name:       "Plus 1",
		Generation: types.Gen2,
		FormFactor: FormFactorFlush,
		Components: Components{Switches: 1},
	})
	Register(&Profile{
		Model:      "SNPL-00112EU",
		Name:       "Plus Plug S",
		Generation: types.Gen2,
		FormFactor: FormFactorPlug,
		Components: Components{Switches: 1},
	})

	similar := FindSimilar("SNSW-001P16EU")
	if len(similar) != 2 {
		t.Errorf("len(FindSimilar) = %d, want 2", len(similar))
	}

	// Non-existent model
	similar = FindSimilar("NONEXISTENT")
	if similar != nil {
		t.Error("FindSimilar(nonexistent) should return nil")
	}
}

func TestInferCapabilitiesFromApp(t *testing.T) {
	tests := []struct {
		app  string
		want Capabilities
	}{
		{
			app:  "Plus1PM",
			want: Capabilities{PowerMetering: true, EnergyMetering: true},
		},
		{
			app:  "Plus2PM",
			want: Capabilities{PowerMetering: true, EnergyMetering: true, CoverSupport: true},
		},
		{
			app:  "ProDimmer1PM",
			want: Capabilities{PowerMetering: true, EnergyMetering: true, DimmingSupport: true},
		},
		{
			app:  "RGBWPM",
			want: Capabilities{PowerMetering: true, EnergyMetering: true, DimmingSupport: true, ColorSupport: true},
		},
		{
			app:  "Pro3EM",
			want: Capabilities{PowerMetering: true, EnergyMetering: true, ThreePhase: true},
		},
		{
			app:  "Plus1",
			want: Capabilities{},
		},
		{
			app:  "bulb",
			want: Capabilities{DimmingSupport: true, ColorSupport: true},
		},
		{
			app:  "duo",
			want: Capabilities{DimmingSupport: true},
		},
		{
			app:  "cover",
			want: Capabilities{CoverSupport: true},
		},
		{
			app:  "shutter",
			want: Capabilities{CoverSupport: true},
		},
		{
			app:  "roller",
			want: Capabilities{CoverSupport: true},
		},
		{
			app:  "color",
			want: Capabilities{ColorSupport: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.app, func(t *testing.T) {
			got := InferCapabilitiesFromApp(tt.app)

			if got.PowerMetering != tt.want.PowerMetering {
				t.Errorf("PowerMetering = %v, want %v", got.PowerMetering, tt.want.PowerMetering)
			}
			if got.EnergyMetering != tt.want.EnergyMetering {
				t.Errorf("EnergyMetering = %v, want %v", got.EnergyMetering, tt.want.EnergyMetering)
			}
			if got.CoverSupport != tt.want.CoverSupport {
				t.Errorf("CoverSupport = %v, want %v", got.CoverSupport, tt.want.CoverSupport)
			}
			if got.DimmingSupport != tt.want.DimmingSupport {
				t.Errorf("DimmingSupport = %v, want %v", got.DimmingSupport, tt.want.DimmingSupport)
			}
			if got.ColorSupport != tt.want.ColorSupport {
				t.Errorf("ColorSupport = %v, want %v", got.ColorSupport, tt.want.ColorSupport)
			}
			if got.ThreePhase != tt.want.ThreePhase {
				t.Errorf("ThreePhase = %v, want %v", got.ThreePhase, tt.want.ThreePhase)
			}
		})
	}
}

func TestDeviceInfoJSONFields(t *testing.T) {
	jsonData := `{
		"id": "abc123",
		"model": "SNSW-001P16EU",
		"gen": 2,
		"app": "Plus1PM",
		"fw_id": "1.0.0",
		"profile": "switch",
		"auth_en": true,
		"auth_domain": "example.com",
		"mac": "001122334455"
	}`

	var info DeviceInfo
	err := json.Unmarshal([]byte(jsonData), &info)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if info.ID != "abc123" {
		t.Errorf("ID = %v, want abc123", info.ID)
	}
	if info.Model != "SNSW-001P16EU" {
		t.Errorf("Model = %v, want SNSW-001P16EU", info.Model)
	}
	if info.Gen != 2 {
		t.Errorf("Gen = %v, want 2", info.Gen)
	}
	if info.App != "Plus1PM" {
		t.Errorf("App = %v, want Plus1PM", info.App)
	}
	if info.FWVersion != "1.0.0" {
		t.Errorf("FWVersion = %v, want 1.0.0", info.FWVersion)
	}
	if info.Profile != "switch" {
		t.Errorf("Profile = %v, want switch", info.Profile)
	}
	if !info.AuthEnabled {
		t.Error("AuthEnabled should be true")
	}
	if info.AuthDomain != "example.com" {
		t.Errorf("AuthDomain = %v, want example.com", info.AuthDomain)
	}
	if info.MAC != "001122334455" {
		t.Errorf("MAC = %v, want 001122334455", info.MAC)
	}
}

func TestGen1StatusJSONFields(t *testing.T) {
	jsonData := `{
		"type": "SHSW-1",
		"mac": "001122334455",
		"auth": true,
		"fw": "1.0.0",
		"hostname": "shelly1-001122",
		"num_outputs": 1,
		"num_meters": 1,
		"num_emeters": 0,
		"num_rollers": 0
	}`

	var status Gen1Status
	err := json.Unmarshal([]byte(jsonData), &status)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if status.Type != "SHSW-1" {
		t.Errorf("Type = %v, want SHSW-1", status.Type)
	}
	if status.MAC != "001122334455" {
		t.Errorf("MAC = %v, want 001122334455", status.MAC)
	}
	if !status.Auth {
		t.Error("Auth should be true")
	}
	if status.FW != "1.0.0" {
		t.Errorf("FW = %v, want 1.0.0", status.FW)
	}
	if status.Hostname != "shelly1-001122" {
		t.Errorf("Hostname = %v, want shelly1-001122", status.Hostname)
	}
	if status.NumOutputs != 1 {
		t.Errorf("NumOutputs = %v, want 1", status.NumOutputs)
	}
	if status.NumMeters != 1 {
		t.Errorf("NumMeters = %v, want 1", status.NumMeters)
	}
}
