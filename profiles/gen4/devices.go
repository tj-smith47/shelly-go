package gen4

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // Device profile auto-registration on package import
func init() {
	// Gen4 protocols include Matter and Zigbee support
	gen4Protocols := profiles.DefaultGen2Protocols()
	gen4Protocols.Matter = true
	gen4Protocols.Zigbee = true

	// Shelly 1 Gen4
	profiles.Register(&profiles.Profile{
		Model:       "S4SW-001X16EU",
		Name:        "Shelly 1 Gen4",
		App:         "Plus1G4",
		Generation:  types.Gen4,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   1,
		},
		Capabilities: mergeGen4Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: gen4Protocols,
		Limits: mergeGen4Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly 1PM Gen4
	profiles.Register(&profiles.Profile{
		Model:       "S4SW-001P16EU",
		Name:        "Shelly 1PM Gen4",
		App:         "Plus1PMG4",
		Generation:  types.Gen4,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			Inputs:      1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen4Caps(profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		}),
		Protocols: gen4Protocols,
		Limits: mergeGen4Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly 2PM Gen4
	profiles.Register(&profiles.Profile{
		Model:       "S4SW-002P16EU",
		Name:        "Shelly 2PM Gen4",
		App:         "Plus2PMG4",
		Generation:  types.Gen4,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    2,
			Covers:      1,
			Inputs:      2,
			PowerMeters: 2,
		},
		Capabilities: mergeGen4Caps(profiles.Capabilities{
			PowerMetering: true,
			CoverSupport:  true,
			Calibration:   true,
			InputEvents:   true,
		}),
		Protocols: gen4Protocols,
		Limits: mergeGen4Limits(&profiles.Limits{
			MaxOutputCurrent: 10,
			MaxPower:         2300,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly 1 Mini Gen4
	profiles.Register(&profiles.Profile{
		Model:       "S4SW-001X8EU",
		Name:        "Shelly 1 Mini Gen4",
		App:         "Plus1MiniG4",
		Generation:  types.Gen4,
		Series:      profiles.SeriesMini,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   1,
		},
		Capabilities: mergeGen4Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: gen4Protocols,
		Limits: mergeGen4Limits(&profiles.Limits{
			MaxOutputCurrent: 8,
			MaxPower:         1800,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly 1PM Mini Gen4
	profiles.Register(&profiles.Profile{
		Model:       "S4SW-001P8EU",
		Name:        "Shelly 1PM Mini Gen4",
		App:         "Plus1PMMiniG4",
		Generation:  types.Gen4,
		Series:      profiles.SeriesMini,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			Inputs:      1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen4Caps(profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		}),
		Protocols: gen4Protocols,
		Limits: mergeGen4Limits(&profiles.Limits{
			MaxOutputCurrent: 8,
			MaxPower:         1800,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly EM Mini Gen4
	profiles.Register(&profiles.Profile{
		Model:       "S4EM-001XCEU",
		Name:        "Shelly EM Mini Gen4",
		App:         "EMMiniG4",
		Generation:  types.Gen4,
		Series:      profiles.SeriesMini,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			EnergyMeters: 1,
		},
		Capabilities: mergeGen4Caps(profiles.Capabilities{
			PowerMetering:  true,
			EnergyMetering: true,
		}),
		Protocols: gen4Protocols,
		Sensors: []profiles.SensorType{
			profiles.SensorVoltage,
			profiles.SensorCurrent,
			profiles.SensorPower,
			profiles.SensorEnergy,
			profiles.SensorPowerFactor,
		},
		Limits: mergeGen4Limits(&profiles.Limits{
			MaxInputCurrent: 50, // 50A CT clamp
			MaxVoltage:      265,
			MinVoltage:      110,
		}),
	})

	// Shelly Plug US Gen4
	profiles.Register(&profiles.Profile{
		Model:       "S4PL-00116US",
		Name:        "Shelly Plug US Gen4",
		App:         "PlugUSG4",
		Generation:  types.Gen4,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen4Caps(profiles.Capabilities{
			PowerMetering: true,
		}),
		Protocols: gen4Protocols,
		Limits: mergeGen4Limits(&profiles.Limits{
			MaxOutputCurrent: 15,
			MaxPower:         1800,
			MinVoltage:       100,
			MaxVoltage:       125,
		}),
	})

	// Shelly Flood Sensor Gen4
	profiles.Register(&profiles.Profile{
		Model:       "S4SN-0W1X",
		Name:        "Shelly Flood Sensor Gen4",
		App:         "FloodG4",
		Generation:  types.Gen4,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
		},
		Capabilities: profiles.DefaultGen2Capabilities(),
		Protocols:    gen4Protocols,
		Sensors: []profiles.SensorType{
			profiles.SensorFlood,
			profiles.SensorTemperature,
			profiles.SensorBattery,
		},
	})

	// Shelly Wall Display X2
	profiles.Register(&profiles.Profile{
		Model:       "S4WD-00695EU",
		Name:        "Shelly Wall Display X2",
		App:         "WallDisplayX2",
		Generation:  types.Gen4,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorWallMount,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Display:            true,
			Switches:           2,
			TemperatureSensors: 1,
			HumiditySensors:    1,
		},
		Capabilities: mergeGen4Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: gen4Protocols,
		Sensors: []profiles.SensorType{
			profiles.SensorTemperature,
			profiles.SensorHumidity,
			profiles.SensorIlluminance,
		},
		Limits: mergeGen4Limits(&profiles.Limits{
			MaxOutputCurrent: 5,
			MaxPower:         1150,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})
}

// mergeGen4Caps merges device-specific capabilities with Gen2 defaults.
// Gen4 devices have the same base capabilities as Gen2.
func mergeGen4Caps(caps profiles.Capabilities) profiles.Capabilities {
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

// mergeGen4Limits merges device-specific limits with Gen2 defaults.
// Gen4 devices have the same limits structure as Gen2.
func mergeGen4Limits(limits *profiles.Limits) profiles.Limits {
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
