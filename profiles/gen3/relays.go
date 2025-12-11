package gen3

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // Device profile auto-registration on package import
func init() {
	// Shelly 1 Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3SW-001X16EU",
		Name:        "Shelly 1 Gen3",
		App:         "Plus1G3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly 1PM Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3SW-001P16EU",
		Name:        "Shelly 1PM Gen3",
		App:         "Plus1PMG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			Inputs:      1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly 1L Gen3 (No Neutral)
	profiles.Register(&profiles.Profile{
		Model:       "S3SW-001L16EU",
		Name:        "Shelly 1L Gen3",
		App:         "Plus1LG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   2,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			NoNeutral:   true,
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 4,
			MaxPower:         1000,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly 2PM Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3SW-002P16EU",
		Name:        "Shelly 2PM Gen3",
		App:         "Plus2PMG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    2,
			Covers:      1,
			Inputs:      2,
			PowerMeters: 2,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			PowerMetering: true,
			CoverSupport:  true,
			Calibration:   true,
			InputEvents:   true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 10,
			MaxPower:         2300,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly 2L Gen3 (No Neutral)
	profiles.Register(&profiles.Profile{
		Model:       "S3SW-002L16EU",
		Name:        "Shelly 2L Gen3",
		App:         "Plus2LG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 2,
			Inputs:   2,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			NoNeutral:   true,
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 4,
			MaxPower:         1000,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly 1 Mini Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3SW-001X8EU",
		Name:        "Shelly 1 Mini Gen3",
		App:         "Plus1MiniG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesMini,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 8,
			MaxPower:         1800,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly 1PM Mini Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3SW-001P8EU",
		Name:        "Shelly 1PM Mini Gen3",
		App:         "Plus1PMMiniG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesMini,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			Inputs:      1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 8,
			MaxPower:         1800,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly PM Mini Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3PM-001PCEU",
		Name:        "Shelly PM Mini Gen3",
		App:         "PMMiniG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesMini,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			PowerMeters: 1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			PowerMetering:  true,
			EnergyMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxInputCurrent: 16,
			MaxPower:        3680,
			MinVoltage:      100,
			MaxVoltage:      240,
		}),
	})

	// Shelly Shutter Gen3
	//nolint:dupl // Similar profile structures are intentional - each device has distinct capabilities
	profiles.Register(&profiles.Profile{
		Model:       "S3SW-002PCEU",
		Name:        "Shelly Shutter Gen3",
		App:         "ShutterG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Covers:      1,
			Inputs:      2,
			PowerMeters: 1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			CoverSupport:  true,
			PowerMetering: true,
			Calibration:   true,
			InputEvents:   true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 10,
			MaxPower:         2300,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})
}

// mergeGen3Caps merges device-specific capabilities with Gen2 defaults.
// Gen3 devices have the same capabilities as Gen2.
func mergeGen3Caps(caps profiles.Capabilities) profiles.Capabilities {
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

// mergeGen3Limits merges device-specific limits with Gen2 defaults.
// Gen3 devices have the same limits structure as Gen2.
func mergeGen3Limits(limits *profiles.Limits) profiles.Limits {
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
