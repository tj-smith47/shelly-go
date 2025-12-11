package gen1

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // Device profile auto-registration on package import
func init() {
	// Shelly 1
	profiles.Register(&profiles.Profile{
		Model:       "SHSW-1",
		Name:        "Shelly 1",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly 1PM
	profiles.Register(&profiles.Profile{
		Model:       "SHSW-PM",
		Name:        "Shelly 1PM",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			Inputs:      1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly 1L (No Neutral)
	profiles.Register(&profiles.Profile{
		Model:       "SHSW-L",
		Name:        "Shelly 1L",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   2,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			NoNeutral:   true,
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 4,
			MaxPower:         1000,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly 2
	profiles.Register(&profiles.Profile{
		Model:       "SHSW-21",
		Name:        "Shelly 2",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    2,
			Inputs:      2,
			PowerMeters: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			PowerMetering: true,
			CoverSupport:  true,
			Calibration:   true,
			InputEvents:   true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 10,
			MaxPower:         2300,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly 2.5
	profiles.Register(&profiles.Profile{
		Model:       "SHSW-25",
		Name:        "Shelly 2.5",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    2,
			Covers:      1,
			Inputs:      2,
			PowerMeters: 2,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			PowerMetering: true,
			CoverSupport:  true,
			Calibration:   true,
			InputEvents:   true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 10,
			MaxPower:         2300,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly 4Pro
	profiles.Register(&profiles.Profile{
		Model:       "SHSW-44",
		Name:        "Shelly 4Pro",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    4,
			Inputs:      4,
			PowerMeters: 4,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 10,
			MaxPower:         2300,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})
}

// mergeGen1Caps merges device-specific capabilities with Gen1 defaults.
func mergeGen1Caps(caps profiles.Capabilities) profiles.Capabilities {
	defaults := profiles.DefaultGen1Capabilities()
	// Copy defaults and override with specific values
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
	if caps.InputEvents {
		defaults.InputEvents = true
	}
	if caps.Effects {
		defaults.Effects = true
	}
	if caps.NoNeutral {
		defaults.NoNeutral = true
	}
	if caps.ThreePhase {
		defaults.ThreePhase = true
	}
	return defaults
}

// mergeGen1Limits merges device-specific limits with Gen1 defaults.
func mergeGen1Limits(limits *profiles.Limits) profiles.Limits {
	defaults := profiles.DefaultGen1Limits()
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
