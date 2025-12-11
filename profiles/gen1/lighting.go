package gen1

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // Device profile auto-registration on package import
func init() {
	// Shelly Dimmer 2
	profiles.Register(&profiles.Profile{
		Model:       "SHDM-2",
		Name:        "Shelly Dimmer 2",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:      1,
			Inputs:      2,
			PowerMeters: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			DimmingSupport: true,
			PowerMetering:  true,
			NoNeutral:      true,
			InputEvents:    true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 1.1,
			MaxPower:         220,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Dimmer (original)
	profiles.Register(&profiles.Profile{
		Model:       "SHDM-1",
		Name:        "Shelly Dimmer",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:      1,
			Inputs:      2,
			PowerMeters: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			DimmingSupport: true,
			PowerMetering:  true,
			NoNeutral:      true,
			InputEvents:    true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 1.1,
			MaxPower:         220,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly RGBW2
	profiles.Register(&profiles.Profile{
		Model:       "SHRGBW2",
		Name:        "Shelly RGBW2",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceDC,
		Components: profiles.Components{
			Lights:        1,
			RGBChannels:   1,
			WhiteChannels: 1,
			Inputs:        1,
			PowerMeters:   1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			DimmingSupport: true,
			ColorSupport:   true,
			PowerMetering:  true,
			Effects:        true,
			InputEvents:    true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 4,
			MaxPower:         48, // 12V * 4A
			MinVoltage:       12,
			MaxVoltage:       24,
		}),
	})

	// Shelly Duo
	profiles.Register(&profiles.Profile{
		Model:       "SHBDUO-1",
		Name:        "Shelly Duo",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorBulb,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:        1,
			WhiteChannels: 1,
			PowerMeters:   1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			DimmingSupport:   true,
			ColorTemperature: true,
			PowerMetering:    true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxPower:   9,
			MinVoltage: 110,
			MaxVoltage: 240,
		}),
	})

	// Shelly Duo GU10
	profiles.Register(&profiles.Profile{
		Model:       "SHBDUO-G10",
		Name:        "Shelly Duo GU10",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorBulb,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:        1,
			WhiteChannels: 1,
			PowerMeters:   1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			DimmingSupport:   true,
			ColorTemperature: true,
			PowerMetering:    true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxPower:   5,
			MinVoltage: 110,
			MaxVoltage: 240,
		}),
	})

	// Shelly Vintage
	profiles.Register(&profiles.Profile{
		Model:       "SHVIN-1",
		Name:        "Shelly Vintage",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorBulb,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:      1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			DimmingSupport: true,
			PowerMetering:  true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxPower:   7,
			MinVoltage: 110,
			MaxVoltage: 240,
		}),
	})

	// Shelly Bulb
	//nolint:dupl // Similar profile structures are intentional - each device has distinct capabilities
	profiles.Register(&profiles.Profile{
		Model:       "SHBLB-1",
		Name:        "Shelly Bulb",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorBulb,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:        1,
			RGBChannels:   1,
			WhiteChannels: 1,
			PowerMeters:   1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			DimmingSupport:   true,
			ColorSupport:     true,
			ColorTemperature: true,
			PowerMetering:    true,
			Effects:          true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxPower:   9,
			MinVoltage: 110,
			MaxVoltage: 240,
		}),
	})

	// Shelly Duo RGBW (Color Bulb)
	//nolint:dupl // Similar profile structures are intentional - each device has distinct capabilities
	profiles.Register(&profiles.Profile{
		Model:       "SHCB-1",
		Name:        "Shelly Duo RGBW",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorBulb,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:        1,
			RGBChannels:   1,
			WhiteChannels: 1,
			PowerMeters:   1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			DimmingSupport:   true,
			ColorSupport:     true,
			ColorTemperature: true,
			PowerMetering:    true,
			Effects:          true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxPower:   9,
			MinVoltage: 110,
			MaxVoltage: 240,
		}),
	})
}
