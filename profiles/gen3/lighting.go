package gen3

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // init() is the standard pattern for auto-registering device profiles
func init() {
	// Shelly Dimmer Gen3
	//nolint:dupl // Similar profile structures are intentional - each device has distinct capabilities
	profiles.Register(&profiles.Profile{
		Model:       "S3DM-0010WW",
		Name:        "Shelly Dimmer Gen3",
		App:         "DimmerG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:      1,
			Inputs:      2,
			PowerMeters: 1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			DimmingSupport: true,
			PowerMetering:  true,
			NoNeutral:      true,
			InputEvents:    true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxOutputCurrent: 1.1,
			MaxPower:         200,
			MinVoltage:       110,
			MaxVoltage:       240,
		}),
	})

	// Shelly Dimmer 0/1-10V PM Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3DM-0D10WW",
		Name:        "Shelly Dimmer 0/1-10V PM Gen3",
		App:         "Dimmer010VG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights:      1,
			Inputs:      2,
			PowerMeters: 1,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			DimmingSupport: true,
			PowerMetering:  true,
			InputEvents:    true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MinVoltage: 110,
			MaxVoltage: 240,
		}),
	})

	// Shelly DALI Dimmer Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3DM-0D0WW",
		Name:        "Shelly DALI Dimmer Gen3",
		App:         "DALIDimmerG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Lights: 1,
			Inputs: 2,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			DimmingSupport: true,
			InputEvents:    true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MinVoltage: 110,
			MaxVoltage: 240,
		}),
	})
}
