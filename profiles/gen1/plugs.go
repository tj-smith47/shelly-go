package gen1

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // init() is the standard pattern for auto-registering device profiles
func init() {
	// Shelly Plug
	profiles.Register(&profiles.Profile{
		Model:       "SHPLG-1",
		Name:        "Shelly Plug",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			PowerMetering: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3500,
			MinVoltage:       110,
			MaxVoltage:       230,
		}),
	})

	// Shelly Plug S
	profiles.Register(&profiles.Profile{
		Model:       "SHPLG-S",
		Name:        "Shelly Plug S",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			PowerMetering: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 12,
			MaxPower:         2500,
			MinVoltage:       110,
			MaxVoltage:       230,
		}),
	})

	// Shelly Plug US
	profiles.Register(&profiles.Profile{
		Model:       "SHPLG-U1",
		Name:        "Shelly Plug US",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			PowerMetering: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxOutputCurrent: 15,
			MaxPower:         1800,
			MinVoltage:       100,
			MaxVoltage:       120,
		}),
	})
}
