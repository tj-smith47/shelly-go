package gen2

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // init() is the standard pattern for auto-registering device profiles
func init() {
	// Shelly Plus i4
	profiles.Register(&profiles.Profile{
		Model:       "SNSN-0024X",
		Name:        "Shelly Plus i4",
		App:         "Plusi4",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Inputs: 4,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MinVoltage: 110,
			MaxVoltage: 240,
		}),
	})

	// Shelly Plus i4 DC
	profiles.Register(&profiles.Profile{
		Model:       "SNSN-0D24X",
		Name:        "Shelly Plus i4 DC",
		App:         "Plusi4DC",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceDC,
		Components: profiles.Components{
			Inputs: 4,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MinVoltage: 5,
			MaxVoltage: 24,
		}),
	})

	// Shelly Plus UNI
	profiles.Register(&profiles.Profile{
		Model:       "SNUN-001",
		Name:        "Shelly Plus UNI",
		App:         "PlusUNI",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceDC,
		Components: profiles.Components{
			Switches:    2,
			Inputs:      2,
			ADCChannels: 1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			InputEvents:     true,
			ExternalSensors: true,
			SensorAddon:     true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MinVoltage:       12,
			MaxVoltage:       36,
			MaxOutputCurrent: 0.5,
		}),
	})
}
