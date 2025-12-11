package gen1

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // init() is the standard pattern for auto-registering device profiles
func init() {
	// Shelly i3
	profiles.Register(&profiles.Profile{
		Model:       "SHIX3-1",
		Name:        "Shelly i3",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Inputs: 3,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MinVoltage: 110,
			MaxVoltage: 240,
		}),
	})

	// Shelly Button 1
	profiles.Register(&profiles.Profile{
		Model:       "SHBTN-1",
		Name:        "Shelly Button1",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			Inputs: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorBattery,
		},
	})

	// Shelly Button 2
	profiles.Register(&profiles.Profile{
		Model:       "SHBTN-2",
		Name:        "Shelly Button2",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			Inputs: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorBattery,
		},
	})

	// Shelly UNI
	profiles.Register(&profiles.Profile{
		Model:       "SHUNI-1",
		Name:        "Shelly UNI",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceDC,
		Components: profiles.Components{
			Switches:    2,
			Inputs:      2,
			ADCChannels: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			InputEvents:     true,
			ExternalSensors: true,
			SensorAddon:     true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Limits: mergeGen1Limits(&profiles.Limits{
			MinVoltage:       12,
			MaxVoltage:       24,
			MaxOutputCurrent: 0.5,
		}),
	})
}
