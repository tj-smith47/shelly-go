package blu

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

//nolint:gochecknoinits // Device profile auto-registration on package import
func init() {
	// BLU devices use Bluetooth Low Energy
	bluProtocols := profiles.Protocols{
		BLE: true,
	}

	// Shelly BLU Button1
	profiles.Register(&profiles.Profile{
		Model:       "SBBT-002C",
		Name:        "Shelly BLU Button1",
		App:         "BLUButton1",
		Generation:  types.Gen2, // BLU devices work with Gen2+ gateways
		Series:      profiles.SeriesBLU,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			Inputs: 1,
		},
		Capabilities: profiles.Capabilities{
			InputEvents: true,
		},
		Protocols: bluProtocols,
		Sensors: []profiles.SensorType{
			profiles.SensorBattery,
		},
	})

	// Shelly BLU Door/Window
	profiles.Register(&profiles.Profile{
		Model:       "SBDW-002C",
		Name:        "Shelly BLU Door/Window",
		App:         "BLUDoorWindow",
		Generation:  types.Gen2,
		Series:      profiles.SeriesBLU,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
		},
		Capabilities: profiles.Capabilities{
			InputEvents: true,
		},
		Protocols: bluProtocols,
		Sensors: []profiles.SensorType{
			profiles.SensorContact,
			profiles.SensorIlluminance,
			profiles.SensorTemperature,
			profiles.SensorBattery,
		},
	})

	// Shelly BLU Motion
	profiles.Register(&profiles.Profile{
		Model:       "SBMO-003Z",
		Name:        "Shelly BLU Motion",
		App:         "BLUMotion",
		Generation:  types.Gen2,
		Series:      profiles.SeriesBLU,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components:  profiles.Components{},
		Capabilities: profiles.Capabilities{
			InputEvents: true,
		},
		Protocols: bluProtocols,
		Sensors: []profiles.SensorType{
			profiles.SensorMotion,
			profiles.SensorIlluminance,
			profiles.SensorBattery,
		},
	})

	// Shelly BLU H&T
	profiles.Register(&profiles.Profile{
		Model:       "SBHT-003C",
		Name:        "Shelly BLU H&T",
		App:         "BLUHT",
		Generation:  types.Gen2,
		Series:      profiles.SeriesBLU,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
			HumiditySensors:    1,
		},
		Capabilities: profiles.Capabilities{},
		Protocols:    bluProtocols,
		Sensors: []profiles.SensorType{
			profiles.SensorTemperature,
			profiles.SensorHumidity,
			profiles.SensorBattery,
		},
	})

	// Shelly BLU RC Button 4
	profiles.Register(&profiles.Profile{
		Model:       "SBBT-004C",
		Name:        "Shelly BLU RC Button 4",
		App:         "BLURCButton4",
		Generation:  types.Gen2,
		Series:      profiles.SeriesBLU,
		FormFactor:  profiles.FormFactorSensor,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			Inputs: 4,
		},
		Capabilities: profiles.Capabilities{
			InputEvents: true,
		},
		Protocols: bluProtocols,
		Sensors: []profiles.SensorType{
			profiles.SensorBattery,
		},
	})

	// Shelly BLU TRV
	profiles.Register(&profiles.Profile{
		Model:       "SBTRV-001",
		Name:        "Shelly BLU TRV",
		App:         "BLUTRV",
		Generation:  types.Gen2,
		Series:      profiles.SeriesBLU,
		FormFactor:  profiles.FormFactorRadiator,
		PowerSource: profiles.PowerSourceBattery,
		Components: profiles.Components{
			TemperatureSensors: 1,
			Thermostat:         true,
		},
		Capabilities: profiles.Capabilities{
			Calibration: true,
		},
		Protocols: bluProtocols,
		Sensors: []profiles.SensorType{
			profiles.SensorTemperature,
			profiles.SensorBattery,
		},
	})

	// Shelly BLU Gateway
	profiles.Register(&profiles.Profile{
		Model:       "SNGW-BT01",
		Name:        "Shelly BLU Gateway",
		App:         "BLUGateway",
		Generation:  types.Gen2,
		Series:      profiles.SeriesBLU,
		FormFactor:  profiles.FormFactorPlug,
		PowerSource: profiles.PowerSourceMains,
		Components:  profiles.Components{},
		Capabilities: profiles.Capabilities{
			Scripting:         true,
			Schedules:         true,
			AdvancedSchedules: true,
			Webhooks:          true,
			KVS:               true,
		},
		Protocols: profiles.Protocols{
			HTTP:      true,
			WebSocket: true,
			MQTT:      true,
			BLE:       true,
		},
	})
}
