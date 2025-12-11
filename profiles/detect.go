package profiles

import (
	"encoding/json"
	"strings"

	"github.com/tj-smith47/shelly-go/types"
)

// DeviceInfo represents the device information returned by Shelly.GetDeviceInfo.
// This is used for profile detection.
type DeviceInfo struct {
	RawFields   map[string]json.RawMessage `json:"-"`
	ID          string                     `json:"id"`
	Model       string                     `json:"model"`
	App         string                     `json:"app"`
	FWVersion   string                     `json:"fw_id"`
	Profile     string                     `json:"profile,omitempty"`
	AuthDomain  string                     `json:"auth_domain,omitempty"`
	MAC         string                     `json:"mac,omitempty"`
	Gen         int                        `json:"gen"`
	AuthEnabled bool                       `json:"auth_en"`
}

// Gen1Status represents Gen1 device status/shelly response used for detection.
type Gen1Status struct {
	Type       string `json:"type"`
	MAC        string `json:"mac"`
	FW         string `json:"fw"`
	Hostname   string `json:"hostname,omitempty"`
	NumOutputs int    `json:"num_outputs,omitempty"`
	NumMeters  int    `json:"num_meters,omitempty"`
	NumEMeters int    `json:"num_emeters,omitempty"`
	NumRollers int    `json:"num_rollers,omitempty"`
	Auth       bool   `json:"auth"`
}

// DetectResult contains the results of device detection.
type DetectResult struct {
	Error      error
	Profile    *Profile
	Model      string
	App        string
	Generation types.Generation
}

// DetectFromDeviceInfo detects the device profile from GetDeviceInfo response.
// This is the primary detection method for Gen2+ devices.
func DetectFromDeviceInfo(info *DeviceInfo) *DetectResult {
	result := &DetectResult{
		Model: info.Model,
		App:   info.App,
	}

	// Determine generation
	switch info.Gen {
	case 2:
		result.Generation = types.Gen2
	case 3:
		result.Generation = types.Gen3
	case 4:
		result.Generation = types.Gen4
	default:
		result.Generation = types.GenerationUnknown
	}

	// Try to find profile by model first
	if profile, ok := Get(info.Model); ok {
		result.Profile = profile
		return result
	}

	// Try by app name
	if profile, ok := GetByApp(info.App); ok {
		result.Profile = profile
		return result
	}

	return result
}

// DetectFromGen1Status detects the device profile from Gen1 /shelly response.
// This is the primary detection method for Gen1 devices.
func DetectFromGen1Status(status *Gen1Status) *DetectResult {
	result := &DetectResult{
		Model:      status.Type,
		Generation: types.Gen1,
	}

	// Try to find profile by type
	if profile, ok := Get(status.Type); ok {
		result.Profile = profile
		return result
	}

	return result
}

// DetectFromModel detects the device profile from just a model string.
// This can be used when only the model identifier is available.
func DetectFromModel(model string) *DetectResult {
	result := &DetectResult{
		Model:      model,
		Generation: DetectGeneration(model),
	}

	if profile, ok := Get(model); ok {
		result.Profile = profile
		return result
	}

	return result
}

// DetectFromJSON detects the device profile from JSON response.
// It auto-detects whether this is a Gen1 or Gen2+ response.
func DetectFromJSON(data []byte) *DetectResult {
	// Try Gen2+ DeviceInfo first
	var info DeviceInfo
	if err := json.Unmarshal(data, &info); err == nil && info.Gen > 0 {
		return DetectFromDeviceInfo(&info)
	}

	// Try Gen1 status
	var status Gen1Status
	if err := json.Unmarshal(data, &status); err == nil && status.Type != "" {
		return DetectFromGen1Status(&status)
	}

	return &DetectResult{
		Generation: types.GenerationUnknown,
	}
}

// MatchCapabilities returns profiles matching the given capability requirements.
func MatchCapabilities(requirements Capabilities) []*Profile {
	var result []*Profile

	for _, p := range List() {
		if matchesCapabilities(p, requirements) {
			result = append(result, p)
		}
	}

	return result
}

// matchesCapabilities checks if a profile satisfies capability requirements.
//
//nolint:gocyclo,cyclop // Capability matching checks multiple fields by design
func matchesCapabilities(p *Profile, req Capabilities) bool {
	// Check each required capability
	if req.PowerMetering && !p.Capabilities.PowerMetering {
		return false
	}
	if req.EnergyMetering && !p.Capabilities.EnergyMetering {
		return false
	}
	if req.CoverSupport && !p.Capabilities.CoverSupport {
		return false
	}
	if req.DimmingSupport && !p.Capabilities.DimmingSupport {
		return false
	}
	if req.ColorSupport && !p.Capabilities.ColorSupport {
		return false
	}
	if req.ColorTemperature && !p.Capabilities.ColorTemperature {
		return false
	}
	if req.Scripting && !p.Capabilities.Scripting {
		return false
	}
	if req.Schedules && !p.Capabilities.Schedules {
		return false
	}
	if req.Webhooks && !p.Capabilities.Webhooks {
		return false
	}
	if req.ThreePhase && !p.Capabilities.ThreePhase {
		return false
	}
	return true
}

// MatchComponents returns profiles with at least the specified component counts.
func MatchComponents(requirements *Components) []*Profile {
	var result []*Profile

	for _, p := range List() {
		if matchesComponents(p, requirements) {
			result = append(result, p)
		}
	}

	return result
}

// matchesComponents checks if a profile has at least the required components.
//
//nolint:gocyclo,cyclop // Component matching checks multiple component types
func matchesComponents(p *Profile, req *Components) bool {
	if req.Switches > 0 && p.Components.Switches < req.Switches {
		return false
	}
	if req.Covers > 0 && p.Components.Covers < req.Covers {
		return false
	}
	if req.Lights > 0 && p.Components.Lights < req.Lights {
		return false
	}
	if req.Inputs > 0 && p.Components.Inputs < req.Inputs {
		return false
	}
	if req.PowerMeters > 0 && p.Components.PowerMeters < req.PowerMeters {
		return false
	}
	if req.EnergyMeters > 0 && p.Components.EnergyMeters < req.EnergyMeters {
		return false
	}
	if req.TemperatureSensors > 0 && p.Components.TemperatureSensors < req.TemperatureSensors {
		return false
	}
	if req.HumiditySensors > 0 && p.Components.HumiditySensors < req.HumiditySensors {
		return false
	}
	return true
}

// FindSimilar finds profiles similar to the given one based on generation and components.
func FindSimilar(model string) []*Profile {
	profile, ok := Get(model)
	if !ok {
		return nil
	}

	var result []*Profile
	for _, p := range ListByGeneration(profile.Generation) {
		if p.Model == model {
			continue
		}
		// Similar if same form factor or similar component count
		if p.FormFactor == profile.FormFactor ||
			(p.Components.Switches == profile.Components.Switches &&
				p.Components.Covers == profile.Components.Covers) {
			result = append(result, p)
		}
	}

	return result
}

// InferCapabilitiesFromApp attempts to infer capabilities from the app name.
// This is useful when a profile isn't registered but basic detection is needed.
func InferCapabilitiesFromApp(app string) Capabilities {
	caps := Capabilities{}
	app = strings.ToLower(app)

	// Power metering indicators
	if strings.Contains(app, "pm") || strings.Contains(app, "em") {
		caps.PowerMetering = true
		caps.EnergyMetering = true
	}

	// Cover/shutter support
	if strings.Contains(app, "2pm") || strings.Contains(app, "cover") ||
		strings.Contains(app, "shutter") || strings.Contains(app, "roller") {
		caps.CoverSupport = true
	}

	// Dimming support
	if strings.Contains(app, "dimmer") || strings.Contains(app, "rgbw") ||
		strings.Contains(app, "bulb") || strings.Contains(app, "duo") {
		caps.DimmingSupport = true
	}

	// Color support
	if strings.Contains(app, "rgb") || strings.Contains(app, "bulb") ||
		strings.Contains(app, "color") {
		caps.ColorSupport = true
	}

	// Three phase
	if strings.Contains(app, "3em") {
		caps.ThreePhase = true
	}

	return caps
}
