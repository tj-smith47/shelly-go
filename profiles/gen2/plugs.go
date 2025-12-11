package gen2

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // init() is the standard pattern for auto-registering device profiles
func init() {
	// Shelly Plus Plug S
	profiles.Register(&profiles.Profile{
		Model:       "SNPL-00112EU",
		Name:        "Shelly Plus Plug S",
		App:         "PlusPlugS",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 12,
			MaxPower:         2500,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Plus Plug US
	profiles.Register(&profiles.Profile{
		Model:       "SNPL-00116US",
		Name:        "Shelly Plus Plug US",
		App:         "PlusPlugUS",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 15,
			MaxPower:         1800,
			MinVoltage:       100,
			MaxVoltage:       125,
		}),
	})

	// Shelly Plus Plug IT
	profiles.Register(&profiles.Profile{
		Model:       "SNPL-00110IT",
		Name:        "Shelly Plus Plug IT",
		App:         "PlusPlugIT",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 10,
			MaxPower:         2300,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Plus Plug UK
	profiles.Register(&profiles.Profile{
		Model:       "SNPL-00112UK",
		Name:        "Shelly Plus Plug UK",
		App:         "PlusPlugUK",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxOutputCurrent: 13,
			MaxPower:         3000,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})
}
