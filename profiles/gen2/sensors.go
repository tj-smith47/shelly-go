package gen2

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // init() is the standard pattern for auto-registering device profiles
func init() {
	// Shelly Plus H&T
	profiles.Register(&profiles.Profile{
		Model:       "SNSN-0013A",
		Name:        "Shelly Plus H&T",
		App:         "PlusHT",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
			HumiditySensors:    1,
		},
		Capabilities: profiles.DefaultGen2Capabilities(),
		Protocols:    profiles.DefaultGen2Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorTemperature,
			profiles.SensorHumidity,
			profiles.SensorBattery,
		},
	})

	// Shelly Plus Smoke
	profiles.Register(&profiles.Profile{
		Model:       "SNSN-0031Z",
		Name:        "Shelly Plus Smoke",
		App:         "PlusSmoke",
		Generation:  types.Gen2,
		Series:      profiles.SeriesPlus,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
		},
		Capabilities: profiles.DefaultGen2Capabilities(),
		Protocols:    profiles.DefaultGen2Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorSmoke,
			profiles.SensorTemperature,
			profiles.SensorBattery,
		},
	})

	// Shelly Plus PM Mini
	profiles.Register(&profiles.Profile{
		Model:       "SNPM-001PCEU",
		Name:        "Shelly Plus PM Mini",
		App:         "PlusPMMini",
		Generation:  types.Gen2,
		Series:      profiles.SeriesMini,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			PowerMeters: 1,
		},
		Capabilities: mergeGen2Caps(profiles.Capabilities{
			PowerMetering:  true,
			EnergyMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen2Limits(&profiles.Limits{
			MaxInputCurrent: 16,
			MaxPower:        3680,
			MinVoltage:      110,
			MaxVoltage:      240,
		}),
	})
}
