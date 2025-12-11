package wave

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // Device profile auto-registration on package import
func init() {
	// Wave devices use Z-Wave protocol
	waveProtocols := profiles.Protocols{
		ZWave: true,
	}

	// Shelly Wave 1
	profiles.Register(&profiles.Profile{
		Model:       "SNSW-001X16ZW",
		Name:        "Shelly Wave 1",
		Generation:  types.Gen2, // Wave devices share Gen2 RPC when accessed via gateway
		Series:      profiles.SeriesWave,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   1,
		},
		Capabilities: profiles.Capabilities{
			InputEvents: true,
		},
		Protocols: waveProtocols,
		Limits: profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		},
	})

	// Shelly Wave 1PM
	profiles.Register(&profiles.Profile{
		Model:       "SNSW-001P16ZW",
		Name:        "Shelly Wave 1PM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesWave,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			Inputs:      1,
			PowerMeters: 1,
		},
		Capabilities: profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		},
		Protocols: waveProtocols,
		Limits: profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		},
	})

	// Shelly Wave 2PM
	profiles.Register(&profiles.Profile{
		Model:       "SNSW-002P16ZW",
		Name:        "Shelly Wave 2PM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesWave,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    2,
			Covers:      1,
			Inputs:      2,
			PowerMeters: 2,
		},
		Capabilities: profiles.Capabilities{
			PowerMetering: true,
			CoverSupport:  true,
			Calibration:   true,
			InputEvents:   true,
		},
		Protocols: waveProtocols,
		Limits: profiles.Limits{
			MaxOutputCurrent: 10,
			MaxPower:         2300,
			MinVoltage:       110,
			MaxVoltage:       240,
		},
	})

	// Shelly Wave Plug US
	profiles.Register(&profiles.Profile{
		Model:       "SNPL-00116USZW",
		Name:        "Shelly Wave Plug US",
		Generation:  types.Gen2,
		Series:      profiles.SeriesWave,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: profiles.Capabilities{
			PowerMetering: true,
		},
		Protocols: waveProtocols,
		Limits: profiles.Limits{
			MaxOutputCurrent: 15,
			MaxPower:         1800,
			MinVoltage:       100,
			MaxVoltage:       125,
		},
	})

	// Wave Pro devices (professional grade)
	waveProProtocols := profiles.Protocols{
		ZWave:    true,
		Ethernet: true,
	}

	// Shelly Wave Pro 1
	profiles.Register(&profiles.Profile{
		Model:       "SPSW-001XE16ZW",
		Name:        "Shelly Wave Pro 1",
		Generation:  types.Gen2,
		Series:      profiles.SeriesWavePro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 1,
			Inputs:   2,
		},
		Capabilities: profiles.Capabilities{
			InputEvents: true,
		},
		Protocols: waveProProtocols,
		Limits: profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		},
	})

	// Shelly Wave Pro 1PM
	profiles.Register(&profiles.Profile{
		Model:       "SPSW-001PE16ZW",
		Name:        "Shelly Wave Pro 1PM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesWavePro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			Inputs:      2,
			PowerMeters: 1,
		},
		Capabilities: profiles.Capabilities{
			PowerMetering: true,
			InputEvents:   true,
		},
		Protocols: waveProProtocols,
		Limits: profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		},
	})

	// Shelly Wave Pro 2
	profiles.Register(&profiles.Profile{
		Model:       "SPSW-002XE16ZW",
		Name:        "Shelly Wave Pro 2",
		Generation:  types.Gen2,
		Series:      profiles.SeriesWavePro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 2,
			Inputs:   2,
		},
		Capabilities: profiles.Capabilities{
			InputEvents: true,
		},
		Protocols: waveProProtocols,
		Limits: profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		},
	})

	// Shelly Wave Pro 2PM
	profiles.Register(&profiles.Profile{
		Model:       "SPSW-002PE16ZW",
		Name:        "Shelly Wave Pro 2PM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesWavePro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    2,
			Covers:      1,
			Inputs:      2,
			PowerMeters: 2,
		},
		Capabilities: profiles.Capabilities{
			PowerMetering: true,
			CoverSupport:  true,
			Calibration:   true,
			InputEvents:   true,
		},
		Protocols: waveProProtocols,
		Limits: profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		},
	})

	// Shelly Wave Pro 3
	profiles.Register(&profiles.Profile{
		Model:       "SPSW-003XE16ZW",
		Name:        "Shelly Wave Pro 3",
		Generation:  types.Gen2,
		Series:      profiles.SeriesWavePro,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches: 3,
			Inputs:   3,
		},
		Capabilities: profiles.Capabilities{
			InputEvents: true,
		},
		Protocols: waveProProtocols,
		Limits: profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       240,
		},
	})

	// Shelly Wave Shutter
	profiles.Register(&profiles.Profile{
		Model:       "SNSW-102P16ZW",
		Name:        "Shelly Wave Shutter",
		Generation:  types.Gen2,
		Series:      profiles.SeriesWave,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Covers:      1,
			Inputs:      2,
			PowerMeters: 1,
		},
		Capabilities: profiles.Capabilities{
			CoverSupport:  true,
			PowerMetering: true,
			Calibration:   true,
			InputEvents:   true,
		},
		Protocols: waveProtocols,
		Limits: profiles.Limits{
			MaxOutputCurrent: 10,
			MaxPower:         2300,
			MinVoltage:       110,
			MaxVoltage:       240,
		},
	})
}
