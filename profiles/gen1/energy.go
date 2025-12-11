package gen1

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // init() is the standard pattern for auto-registering device profiles
func init() {
	// Shelly EM
	profiles.Register(&profiles.Profile{
		Model:       "SHEM",
		Name:        "Shelly EM",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:     1,
			EnergyMeters: 2,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			PowerMetering:  true,
			EnergyMetering: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorVoltage,
			profiles.SensorCurrent,
			profiles.SensorPower,
			profiles.SensorEnergy,
			profiles.SensorPowerFactor,
		},
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxInputCurrent: 120, // CT clamps
			MaxVoltage:      265,
			MinVoltage:      110,
		}),
	})

	// Shelly 3EM
	profiles.Register(&profiles.Profile{
		Model:       "SHEM-3",
		Name:        "Shelly 3EM",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorDIN,
		PowerSource: profiles.PowerSourceMains,
		Components: profiles.Components{
			Switches:     1,
			EnergyMeters: 3,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			PowerMetering:  true,
			EnergyMetering: true,
			ThreePhase:     true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorVoltage,
			profiles.SensorCurrent,
			profiles.SensorPower,
			profiles.SensorEnergy,
			profiles.SensorPowerFactor,
		},
		Limits: mergeGen1Limits(&profiles.Limits{
			MaxInputCurrent: 120, // CT clamps
			MaxVoltage:      265,
			MinVoltage:      110,
		}),
	})
}
