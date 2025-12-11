package gen2

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // Device profile auto-registration on package import
func init() {
	// Pro devices have Ethernet support
	proProtocols := profiles.DefaultGen2Protocols()
	proProtocols.Ethernet = true

	// Shelly Pro 1
	profiles.Register(&profiles.Profile{
		Model:       "SPSW-001XE16EU",
		Name:        "Shelly Pro 1",
		App:         "Pro1",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   2,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: proProtocols,
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Pro 1PM
	profiles.Register(&profiles.Profile{
		Model:       "SPSW-001PE16EU",
		Name:        "Shelly Pro 1PM",
		App:         "Pro1PM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			Inputs:      2,
			PowerMeters: 1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		}),
		Protocols: proProtocols,
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Pro 2
	profiles.Register(&profiles.Profile{
		Model:       "SPSW-002XE16EU",
		Name:        "Shelly Pro 2",
		App:         "Pro2",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 2,
			Inputs:   2,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: proProtocols,
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Pro 2PM
	profiles.Register(&profiles.Profile{
		Model:       "SPSW-002PE16EU",
		Name:        "Shelly Pro 2PM",
		App:         "Pro2PM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPro,
		FormFactor:  profiles.FormFactorDIN,
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
		Protocols: proProtocols,
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Pro 3
	profiles.Register(&profiles.Profile{
		Model:       "SPSW-003XE16EU",
		Name:        "Shelly Pro 3",
		App:         "Pro3",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 3,
			Inputs:   3,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: proProtocols,
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Pro 3EM
	profiles.Register(&profiles.Profile{
		Model:       "SPEM-003CEBEU",
		Name:        "Shelly Pro 3EM",
		App:         "Pro3EM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			EnergyMeters: 3,
			Switches:     1, // Contactor control
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering:         true,
			EnergyMetering:        true,
			ThreePhase:            true,
			BidirectionalMetering: true,
		}),
		Protocols: proProtocols,
		Sensors: []profiles.SensorType{
			profiles.SensorVoltage,
			profiles.SensorCurrent,
			profiles.SensorPower,
			profiles.SensorEnergy,
			profiles.SensorPowerFactor,
		},
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxInputCurrent: 120, // CT clamps up to 120A
			MaxVoltage:      265,
			MinVoltage:      110,
		}),
	})

	// Shelly Pro 4PM
	profiles.Register(&profiles.Profile{
		Model:       "SPSW-004PE16EU",
		Name:        "Shelly Pro 4PM",
		App:         "Pro4PM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    4,
			Inputs:      4,
			PowerMeters: 4,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		}),
		Protocols: proProtocols,
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Pro Dimmer 1PM
	profiles.Register(&profiles.Profile{
		Model:       "SPDM-001PE01EU",
		Name:        "Shelly Pro Dimmer 1PM",
		App:         "ProDimmer1PM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:      1,
			Inputs:      2,
			PowerMeters: 1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			DimmingSupport: true,
			PowerMetering:  true,
			InputEvents:    true,
		}),
		Protocols: proProtocols,
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 1.1,
			MaxPower:         200,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Pro Dimmer 2PM
	profiles.Register(&profiles.Profile{
		Model:       "SPDM-002PE01EU",
		Name:        "Shelly Pro Dimmer 2PM",
		App:         "ProDimmer2PM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:      2,
			Inputs:      4,
			PowerMeters: 2,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			DimmingSupport: true,
			PowerMetering:  true,
			InputEvents:    true,
		}),
		Protocols: proProtocols,
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 1.1,
			MaxPower:         200,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Pro Dual Cover PM
	profiles.Register(&profiles.Profile{
		Model:       "SPSH-002PE16EU",
		Name:        "Shelly Pro Dual Cover PM",
		App:         "ProDualCoverPM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Covers:      2,
			Inputs:      4,
			PowerMeters: 2,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			CoverSupport:  true,
			PowerMetering: true,
			Calibration:   true,
			InputEvents:   true,
		}),
		Protocols: proProtocols,
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 10,
			MaxPower:         2300,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Pro EM-50
	profiles.Register(&profiles.Profile{
		Model:       "SPEM-002CEBEU50",
		Name:        "Shelly Pro EM-50",
		App:         "ProEM50",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			EnergyMeters: 2,
			Switches:     1, // Contactor control
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering:  true,
			EnergyMetering: true,
		}),
		Protocols: proProtocols,
		Sensors: []profiles.SensorType{
			profiles.SensorVoltage,
			profiles.SensorCurrent,
			profiles.SensorPower,
			profiles.SensorEnergy,
			profiles.SensorPowerFactor,
		},
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxInputCurrent: 50, // 50A CT clamps
			MaxVoltage:      265,
			MinVoltage:      110,
		}),
	})
}
