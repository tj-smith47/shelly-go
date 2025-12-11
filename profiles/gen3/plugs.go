package gen3

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // init() is the standard pattern for auto-registering device profiles
func init() {
	// Shelly Plug S Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3PL-00112EU",
		Name:        "Shelly Plug S Gen3",
		App:         "PlugSG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			PowerMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 12,
			MaxPower:         2500,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly Plug S MTR Gen3 (Enhanced metering)
	profiles.Register(&profiles.Profile{
		Model:       "S3PL-00112EUMTR",
		Name:        "Shelly Plug S MTR Gen3",
		App:         "PlugSMTRG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			PowerMetering:  true,
			EnergyMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 12,
			MaxPower:         2500,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly Outdoor Plug S Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3PL-00212EU",
		Name:        "Shelly Outdoor Plug S Gen3",
		App:         "OutdoorPlugSG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorOutdoor,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			PowerMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 12,
			MaxPower:         2500,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})

	// Shelly Plug PM Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3PL-10112EU",
		Name:        "Shelly Plug PM Gen3",
		App:         "PlugPMG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:    1,
			PowerMeters: 1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			PowerMetering:  true,
			EnergyMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 16,
			MaxPower:         3680,
			MinVoltage:       100,
			MaxVoltage:       240,
		}),
	})
}
