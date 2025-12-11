package gen1

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // Device profile auto-registration on package import
func init() {
	// Shelly H&T (Humidity & Temperature)
	profiles.Register(&profiles.Profile{
		Model:       "SHHT-1",
		Name:        "Shelly H&T",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
			HumiditySensors:    1,
		},
		Capabilities: profiles.DefaultGen1Capabilities(),
		Protocols:    profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorTemperature,
			profiles.SensorHumidity,
			profiles.SensorBattery,
		},
	})

	// Shelly Flood
	profiles.Register(&profiles.Profile{
		Model:       "SHWT-1",
		Name:        "Shelly Flood",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
		},
		Capabilities: profiles.DefaultGen1Capabilities(),
		Protocols:    profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorFlood,
			profiles.SensorTemperature,
			profiles.SensorBattery,
		},
	})

	// Shelly Door/Window 2
	profiles.Register(&profiles.Profile{
		Model:       "SHDW-2",
		Name:        "Shelly Door/Window 2",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
		},
		Capabilities: profiles.DefaultGen1Capabilities(),
		Protocols:    profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorContact,
			profiles.SensorVibration,
			profiles.SensorTilt,
			profiles.SensorIlluminance,
			profiles.SensorTemperature,
			profiles.SensorBattery,
		},
	})

	// Shelly Door/Window (original)
	profiles.Register(&profiles.Profile{
		Model:       "SHDW-1",
		Name:        "Shelly Door/Window",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
		},
		Capabilities: profiles.DefaultGen1Capabilities(),
		Protocols:    profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorContact,
			profiles.SensorIlluminance,
			profiles.SensorTemperature,
			profiles.SensorBattery,
		},
	})

	// Shelly Gas
	profiles.Register(&profiles.Profile{
		Model:       "SHGS-1",
		Name:        "Shelly Gas",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorWallMount,
		PowerSource: profiles.PowerSourceMains,
		Components:  profiles.Components{},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorGas,
		},
	})

	// Shelly Smoke
	profiles.Register(&profiles.Profile{
		Model:       "SHSM-01",
		Name:        "Shelly Smoke",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
		},
		Capabilities: profiles.DefaultGen1Capabilities(),
		Protocols:    profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorSmoke,
			profiles.SensorTemperature,
			profiles.SensorBattery,
		},
	})

	// Shelly Motion
	profiles.Register(&profiles.Profile{
		Model:       "SHMOS-01",
		Name:        "Shelly Motion",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components:  profiles.Components{},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorMotion,
			profiles.SensorIlluminance,
			profiles.SensorBattery,
		},
	})

	// Shelly Motion 2
	profiles.Register(&profiles.Profile{
		Model:       "SHMOS-02",
		Name:        "Shelly Motion 2",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			InputEvents: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorMotion,
			profiles.SensorIlluminance,
			profiles.SensorTemperature,
			profiles.SensorBattery,
		},
	})

	// Shelly TRV (Thermostatic Radiator Valve)
	profiles.Register(&profiles.Profile{
		Model:       "SHTRV-01",
		Name:        "Shelly TRV",
		Generation:  types.Gen1,
		Series:      profiles.SeriesClassic,
		FormFactor:  profiles.FormFactorRadiator,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
			Thermostat:         true,
		},
		Capabilities: mergeGen1Caps(profiles.Capabilities{
			Calibration: true,
		}),
		Protocols: profiles.DefaultGen1Protocols(),
		Sensors: []profiles.SensorType{
			profiles.SensorTemperature,
			profiles.SensorBattery,
		},
	})
}
