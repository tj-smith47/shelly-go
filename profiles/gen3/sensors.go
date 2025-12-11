package gen3

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // Device profile auto-registration on package import
func init() {
	// Shelly i4 Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3SN-0024X",
		Name:        "Shelly i4 Gen3",
		App:         "i4G3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorFlush,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Inputs: 4,
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Limits: mergeGen3Limits(&profiles.Limits{
			MinVoltage: 110,
			MaxVoltage: 240,
		}),
	})

	// Shelly H&T Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3SN-0U12A",
		Name:        "Shelly H&T Gen3",
		App:         "HTG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
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

	// Shelly EM Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3EM-002CXCEU",
		Name:        "Shelly EM Gen3",
		App:         "EMG3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			EnergyMeters: 2,
			Switches:     1, // Contactor control
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			PowerMetering:  true,
			EnergyMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorVoltage,
			profiles.SensorCurrent,
			profiles.SensorPower,
			profiles.SensorEnergy,
			profiles.SensorPowerFactor,
		},
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxInputCurrent: 50, // 50A CT clamps
			MaxVoltage:      265,
			MinVoltage:      110,
		}),
	})

	// Shelly 3EM-63 Gen3
	profiles.Register(&profiles.Profile{
		Model:       "S3EM-003CXCEU",
		Name:        "Shelly 3EM-63 Gen3",
		App:         "3EM63G3",
		Generation:  types.Gen3,
		Series:      profiles.SeriesStandard,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			EnergyMeters: 3,
			Switches:     1, // Contactor control
		},
		Capabilities: mergeGen3Caps(profiles.Capabilities{
			PowerMetering:         true,
			EnergyMetering:        true,
			ThreePhase:            true,
			BidirectionalMetering: true,
		}),
		Protocols: profiles.DefaultGen2Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorVoltage,
			profiles.SensorCurrent,
			profiles.SensorPower,
			profiles.SensorEnergy,
			profiles.SensorPowerFactor,
		},
		Limits: mergeGen3Limits(&profiles.Limits{
			MaxInputCurrent: 63, // 63A CT clamps
			MaxVoltage:      265,
			MinVoltage:      110,
		}),
	})
}
