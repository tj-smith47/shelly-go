package profiles

import (
	"strings"
	"sync"

	"github.com/tj-smith47/shelly-go/types"
)

// registry holds all registered device profiles.
var (
	registry     = make(map[string]*Profile)
	appIndex     = make(map[string]*Profile)
	registryLock sync.RWMutex
)

// Register adds a device profile to the registry.
// It indexes by both model ID and app name.
func Register(profile *Profile) {
	registryLock.Lock()
	defer registryLock.Unlock()

	registry[profile.Model] = profile
	if profile.App != "" {
		appIndex[profile.App] = profile
	}
}

// Get retrieves a profile by model ID.
// Returns the profile and true if found, nil and false otherwise.
func Get(model string) (*Profile, bool) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	profile, ok := registry[model]
	return profile, ok
}

// GetByApp retrieves a profile by application name.
// Returns the profile and true if found, nil and false otherwise.
func GetByApp(app string) (*Profile, bool) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	profile, ok := appIndex[app]
	return profile, ok
}

// MustGet retrieves a profile by model ID.
// Panics if the profile is not found.
func MustGet(model string) *Profile {
	profile, ok := Get(model)
	if !ok {
		panic("profile not found: " + model)
	}
	return profile
}

// Exists returns true if a profile exists for the given model ID.
func Exists(model string) bool {
	registryLock.RLock()
	defer registryLock.RUnlock()

	_, ok := registry[model]
	return ok
}

// List returns all registered profiles.
func List() []*Profile {
	registryLock.RLock()
	defer registryLock.RUnlock()

	result := make([]*Profile, 0, len(registry))
	for _, p := range registry {
		result = append(result, p)
	}
	return result
}

// ListByGeneration returns all profiles for a specific generation.
func ListByGeneration(gen types.Generation) []*Profile {
	registryLock.RLock()
	defer registryLock.RUnlock()

	var result []*Profile
	for _, p := range registry {
		if p.Generation == gen {
			result = append(result, p)
		}
	}
	return result
}

// ListBySeries returns all profiles for a specific series.
func ListBySeries(series Series) []*Profile {
	registryLock.RLock()
	defer registryLock.RUnlock()

	var result []*Profile
	for _, p := range registry {
		if p.Series == series {
			result = append(result, p)
		}
	}
	return result
}

// ListByCapability returns all profiles with a specific capability.
func ListByCapability(capability string) []*Profile {
	registryLock.RLock()
	defer registryLock.RUnlock()

	var result []*Profile
	for _, p := range registry {
		if hasCapability(p, capability) {
			result = append(result, p)
		}
	}
	return result
}

// Canonical capability names used as map values, switch labels, and alias keys.
const (
	capPowerMetering         = "power_metering"
	capEnergyMetering        = "energy_metering"
	capCoverSupport          = "cover_support"
	capCover                 = "cover"
	capDimmingSupport        = "dimming_support"
	capColorSupport          = "color_support"
	capColor                 = "color"
	capColorTemperature      = "color_temperature"
	capScripting             = "scripting"
	capSchedules             = "schedules"
	capWebhooks              = "webhooks"
	capKVS                   = "kvs"
	capVirtualComponents     = "virtual_components"
	capActions               = "actions"
	capSensorAddon           = "sensor_addon"
	capCalibration           = "calibration"
	capInputEvents           = "input_events"
	capEffects               = "effects"
	capNoNeutral             = "no_neutral"
	capBidirectionalMetering = "bidirectional_metering"
	capThreePhase            = "three_phase"
)

// capabilityAliases maps various capability name formats to canonical names.
var capabilityAliases = map[string]string{
	capPowerMetering:         capPowerMetering,
	"powermetering":          capPowerMetering,
	capEnergyMetering:        capEnergyMetering,
	"energymetering":         capEnergyMetering,
	capCoverSupport:          capCoverSupport,
	"coversupport":           capCoverSupport,
	capCover:                 capCoverSupport,
	capDimmingSupport:        capDimmingSupport,
	"dimmingsupport":         capDimmingSupport,
	"dimming":                capDimmingSupport,
	capColorSupport:          capColorSupport,
	"colorsupport":           capColorSupport,
	capColor:                 capColorSupport,
	"rgb":                    capColorSupport,
	capColorTemperature:      capColorTemperature,
	"colortemperature":       capColorTemperature,
	"cct":                    capColorTemperature,
	capScripting:             capScripting,
	"scripts":                capScripting,
	capSchedules:             capSchedules,
	capWebhooks:              capWebhooks,
	capKVS:                   capKVS,
	capVirtualComponents:     capVirtualComponents,
	"virtualcomponents":      capVirtualComponents,
	capActions:               capActions,
	capSensorAddon:           capSensorAddon,
	"sensoraddon":            capSensorAddon,
	capCalibration:           capCalibration,
	capInputEvents:           capInputEvents,
	"inputevents":            capInputEvents,
	capEffects:               capEffects,
	capNoNeutral:             capNoNeutral,
	"noneutral":              capNoNeutral,
	capBidirectionalMetering: capBidirectionalMetering,
	"bidirectionalmetering":  capBidirectionalMetering,
	capThreePhase:            capThreePhase,
	"threephase":             capThreePhase,
	"3phase":                 capThreePhase,
}

// hasCapability checks if a profile has a specific capability by name.
func hasCapability(p *Profile, capability string) bool {
	canonical, ok := capabilityAliases[strings.ToLower(capability)]
	if !ok {
		return false
	}
	return getCapabilityValue(p, canonical)
}

// getCapabilityValue returns the value of a canonical capability name.
//
//nolint:gocyclo,cyclop // Capability value extraction checks multiple capability fields
func getCapabilityValue(p *Profile, canonical string) bool {
	switch canonical {
	case capPowerMetering:
		return p.Capabilities.PowerMetering
	case capEnergyMetering:
		return p.Capabilities.EnergyMetering
	case capCoverSupport:
		return p.Capabilities.CoverSupport
	case capDimmingSupport:
		return p.Capabilities.DimmingSupport
	case capColorSupport:
		return p.Capabilities.ColorSupport
	case capColorTemperature:
		return p.Capabilities.ColorTemperature
	case capScripting:
		return p.Capabilities.Scripting
	case capSchedules:
		return p.Capabilities.Schedules
	case capWebhooks:
		return p.Capabilities.Webhooks
	case capKVS:
		return p.Capabilities.KVS
	case capVirtualComponents:
		return p.Capabilities.VirtualComponents
	case capActions:
		return p.Capabilities.Actions
	case capSensorAddon:
		return p.Capabilities.SensorAddon
	case capCalibration:
		return p.Capabilities.Calibration
	case capInputEvents:
		return p.Capabilities.InputEvents
	case capEffects:
		return p.Capabilities.Effects
	case capNoNeutral:
		return p.Capabilities.NoNeutral
	case capBidirectionalMetering:
		return p.Capabilities.BidirectionalMetering
	case capThreePhase:
		return p.Capabilities.ThreePhase
	default:
		return false
	}
}

// ListByProtocol returns all profiles supporting a specific protocol.
func ListByProtocol(protocol string) []*Profile {
	registryLock.RLock()
	defer registryLock.RUnlock()

	var result []*Profile
	for _, p := range registry {
		if hasProtocol(p, protocol) {
			result = append(result, p)
		}
	}
	return result
}

// hasProtocol checks if a profile supports a specific protocol by name.
func hasProtocol(p *Profile, protocol string) bool {
	switch strings.ToLower(protocol) {
	case "http":
		return p.Protocols.HTTP
	case "websocket", "ws":
		return p.Protocols.WebSocket
	case "mqtt":
		return p.Protocols.MQTT
	case "coiot", "coap":
		return p.Protocols.CoIoT
	case "ble", "bluetooth":
		return p.Protocols.BLE
	case "matter":
		return p.Protocols.Matter
	case "zigbee":
		return p.Protocols.Zigbee
	case "zwave", "z-wave":
		return p.Protocols.ZWave
	case "ethernet", "eth":
		return p.Protocols.Ethernet
	default:
		return false
	}
}

// ListByComponent returns all profiles that have a specific component type.
func ListByComponent(ct types.ComponentType) []*Profile {
	registryLock.RLock()
	defer registryLock.RUnlock()

	var result []*Profile
	for _, p := range registry {
		if p.HasComponent(ct) {
			result = append(result, p)
		}
	}
	return result
}

// ListByFormFactor returns all profiles with a specific form factor.
func ListByFormFactor(ff FormFactor) []*Profile {
	registryLock.RLock()
	defer registryLock.RUnlock()

	var result []*Profile
	for _, p := range registry {
		if p.FormFactor == ff {
			result = append(result, p)
		}
	}
	return result
}

// ListByPowerSource returns all profiles with a specific power source.
func ListByPowerSource(ps PowerSource) []*Profile {
	registryLock.RLock()
	defer registryLock.RUnlock()

	var result []*Profile
	for _, p := range registry {
		if p.PowerSource == ps {
			result = append(result, p)
		}
	}
	return result
}

// Count returns the total number of registered profiles.
func Count() int {
	registryLock.RLock()
	defer registryLock.RUnlock()

	return len(registry)
}

// Search returns profiles matching the search query.
// Searches model, name, and app fields.
func Search(query string) []*Profile {
	registryLock.RLock()
	defer registryLock.RUnlock()

	query = strings.ToLower(query)
	var result []*Profile
	for _, p := range registry {
		if strings.Contains(strings.ToLower(p.Model), query) ||
			strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.App), query) {
			result = append(result, p)
		}
	}
	return result
}

// DetectGeneration attempts to detect the device generation from the model ID.
func DetectGeneration(model string) types.Generation {
	model = strings.ToUpper(model)

	switch {
	case strings.HasPrefix(model, "S4"):
		return types.Gen4
	case strings.HasPrefix(model, "S3"):
		return types.Gen3
	case strings.HasPrefix(model, "SN"):
		return types.Gen2
	case strings.HasPrefix(model, "SH"):
		return types.Gen1
	default:
		return types.GenerationUnknown
	}
}

// Clear removes all profiles from the registry.
// This is primarily useful for testing.
func Clear() {
	registryLock.Lock()
	defer registryLock.Unlock()

	registry = make(map[string]*Profile)
	appIndex = make(map[string]*Profile)
}
