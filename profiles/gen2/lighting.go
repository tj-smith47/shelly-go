package gen2

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // init() is the standard pattern for auto-registering device profiles
func init() {
	// Shelly Plus Wall Dimmer US
	profiles.Register(&profiles.Profile{
		Model:       "SNDM-0013US",
		Name:        "Shelly Plus Wall Dimmer",
		App:         "PlusWallDimmer",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorWallMount,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:      1,
			Inputs:      2,
			PowerMeters: 1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			DimmingSupport: true,
			PowerMetering:  true,
			NoNeutral:      true,
			InputEvents:    true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 1.1,
			MaxPower:         200,
			MinVoltage:       110,
			MaxVoltage:       120,
		}),
	})

	// Shelly Plus 0-10V Dimmer
	profiles.Register(&profiles.Profile{
		Model:       "SNDM-00100WW",
		Name:        "Shelly Plus 0-10V Dimmer",
		App:         "Plus010VDimmer",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights: 1,
			Inputs: 2,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			DimmingSupport: true,
			InputEvents:    true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MinVoltage: 110,
			MaxVoltage: 240,
		}),
	})

	// Shelly Plus RGBW PM
	profiles.Register(&profiles.Profile{
		Model:       "SNDC-0D4P10WW",
		Name:        "Shelly Plus RGBW PM",
		App:         "PlusRGBWPM",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceDC,
		Components: profiles.Components{
			Lights:        1,
			RGBChannels:   1,
			WhiteChannels: 1,
			Inputs:        4,
			PowerMeters:   1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			DimmingSupport:   true,
			ColorSupport:     true,
			ColorTemperature: true,
			PowerMetering:    true,
			Effects:          true,
			InputEvents:      true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 4.5,
			MaxPower:         54, // 12V * 4.5A
			MinVoltage:       12,
			MaxVoltage:       24,
		}),
	})
}
