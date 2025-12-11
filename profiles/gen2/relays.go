package gen2

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // Device profile auto-registration on package import
func init() {
	// Shelly Plus 1
	profiles.Register(&profiles.Profile{
		Model:       "SNSW-001X16EU",
		Name:        "Shelly Plus 1",
		App:         "Plus1",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Plus 1PM
	profiles.Register(&profiles.Profile{
		Model:       "SNSW-001P16EU",
		Name:        "Shelly Plus 1PM",
		App:         "Plus1PM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			Inputs:      1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Plus 1 Mini
	profiles.Register(&profiles.Profile{
		Model:       "SNSW-001X8EU",
		Name:        "Shelly Plus 1 Mini",
		App:         "Plus1Mini",
		Generation:  types.Gen2,
		Series:      profiles.SeriesMini,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 8,
			MaxPower:         1800,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Plus 1PM Mini
	profiles.Register(&profiles.Profile{
		Model:       "SNSW-001P8EU",
		Name:        "Shelly Plus 1PM Mini",
		App:         "Plus1PMMini",
		Generation:  types.Gen2,
		Series:      profiles.SeriesMini,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			Inputs:      1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 8,
			MaxPower:         1800,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Plus 2PM
	profiles.Register(&profiles.Profile{
		Model:       "SNSW-002P16EU",
		Name:        "Shelly Plus 2PM",
		App:         "Plus2PM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    2,
			Covers:      1,
			Inputs:      2,
			PowerMeters: 2,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering: true,
			CoverSupport:  true,
			Calibration:   true,
			InputEvents:   true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 10,
			MaxPower:         2300,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})
}

// mergeGen2Caps merges device-specific capabilities with Gen2 defaults.
func mergeGen2Caps(caps profiles.Capabilities) profiles.Capabilities {
	defaults := profiles.DefaultGen2Capabilities()
	if caps.PowerMetering {
		defaults.PowerMetering = true
	}
	if caps.EnergyMetering {
		defaults.EnergyMetering = true
	}
	if caps.CoverSupport {
		defaults.CoverSupport = true
	}
	if caps.DimmingSupport {
		defaults.DimmingSupport = true
	}
	if caps.ColorSupport {
		defaults.ColorSupport = true
	}
	if caps.ColorTemperature {
		defaults.ColorTemperature = true
	}
	if caps.SensorAddon {
		defaults.SensorAddon = true
	}
	if caps.ExternalSensors {
		defaults.ExternalSensors = true
	}
	if caps.Calibration {
		defaults.Calibration = true
	}
	if caps.Effects {
		defaults.Effects = true
	}
	if caps.NoNeutral {
		defaults.NoNeutral = true
	}
	if caps.BidirectionalMetering {
		defaults.BidirectionalMetering = true
	}
	if caps.ThreePhase {
		defaults.ThreePhase = true
	}
	return defaults
}

// mergeGen2Limits merges device-specific limits with Gen2 defaults.
func mergeGen2Limits(limits *profiles.Limits) profiles.Limits {
	defaults := profiles.DefaultGen2Limits()
	if limits.MaxInputCurrent > 0 {
		defaults.MaxInputCurrent = limits.MaxInputCurrent
	}
	if limits.MaxOutputCurrent > 0 {
		defaults.MaxOutputCurrent = limits.MaxOutputCurrent
	}
	if limits.MaxPower > 0 {
		defaults.MaxPower = limits.MaxPower
	}
	if limits.MaxVoltage > 0 {
		defaults.MaxVoltage = limits.MaxVoltage
	}
	if limits.MinVoltage > 0 {
		defaults.MinVoltage = limits.MinVoltage
	}
	return defaults
}
