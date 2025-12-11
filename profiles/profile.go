package profiles

import (
	"github.com/tj-smith47/shelly-go/types"
)

// Profile defines the capabilities and characteristics of a Shelly device model.
type Profile struct {
	Model        string           `json:"model"`
	Name         string           `json:"name"`
	App          string           `json:"app,omitempty"`
	Series       Series           `json:"series"`
	PowerSource  PowerSource      `json:"power_source"`
	FormFactor   FormFactor       `json:"form_factor"`
	Sensors      []SensorType     `json:"sensors,omitempty"`
	Components   Components       `json:"components"`
	Limits       Limits           `json:"limits"`
	Generation   types.Generation `json:"generation"`
	Capabilities Capabilities     `json:"capabilities"`
	Protocols    Protocols        `json:"protocols"`
}

// Series represents the device series within a generation.
type Series string

const (
	// SeriesClassic represents Gen1 classic devices.
	SeriesClassic Series = "classic"

	// SeriesPlus represents Gen2 Plus series (consumer-grade).
	SeriesPlus Series = "plus"

	// SeriesPro represents Gen2 Pro series (professional-grade).
	SeriesPro Series = "pro"

	// SeriesMini represents compact Mini variants.
	SeriesMini Series = "mini"

	// SeriesBLU represents Bluetooth devices.
	SeriesBLU Series = "blu"

	// SeriesWave represents Z-Wave devices.
	SeriesWave Series = "wave"

	// SeriesWavePro represents Z-Wave Pro devices.
	SeriesWavePro Series = "wave_pro"

	// SeriesStandard represents standard Gen3/Gen4 devices.
	SeriesStandard Series = "standard"
)

// Components defines the available components and their counts.
type Components struct {
	// Switches is the number of switch/relay outputs.
	Switches int `json:"switches,omitempty"`

	// Covers is the number of cover/roller outputs.
	Covers int `json:"covers,omitempty"`

	// Lights is the number of dimmable light outputs.
	Lights int `json:"lights,omitempty"`

	// Inputs is the number of digital inputs.
	Inputs int `json:"inputs,omitempty"`

	// Outputs is the number of generic outputs.
	Outputs int `json:"outputs,omitempty"`

	// PowerMeters is the number of power meter channels.
	PowerMeters int `json:"power_meters,omitempty"`

	// EnergyMeters is the number of energy meter channels (CT clamps).
	EnergyMeters int `json:"energy_meters,omitempty"`

	// Voltmeters is the number of voltmeter channels.
	Voltmeters int `json:"voltmeters,omitempty"`

	// TemperatureSensors is the number of temperature sensors.
	TemperatureSensors int `json:"temperature_sensors,omitempty"`

	// HumiditySensors is the number of humidity sensors.
	HumiditySensors int `json:"humidity_sensors,omitempty"`

	// ADCChannels is the number of analog-to-digital converter channels.
	ADCChannels int `json:"adc_channels,omitempty"`

	// RGBChannels is the number of RGB(W) channels.
	RGBChannels int `json:"rgb_channels,omitempty"`

	// WhiteChannels is the number of white/CCT channels.
	WhiteChannels int `json:"white_channels,omitempty"`

	// Thermostat indicates if the device has thermostat functionality.
	Thermostat bool `json:"thermostat,omitempty"`

	// Display indicates if the device has a display.
	Display bool `json:"display,omitempty"`
}

// Capabilities defines what features a device supports.
type Capabilities struct {
	// PowerMetering indicates the device can measure power consumption.
	PowerMetering bool `json:"power_metering,omitempty"`

	// EnergyMetering indicates the device can measure energy usage over time.
	EnergyMetering bool `json:"energy_metering,omitempty"`

	// CoverSupport indicates the device supports roller/cover mode.
	CoverSupport bool `json:"cover_support,omitempty"`

	// DimmingSupport indicates the device supports dimming.
	DimmingSupport bool `json:"dimming_support,omitempty"`

	// ColorSupport indicates the device supports RGB/color control.
	ColorSupport bool `json:"color_support,omitempty"`

	// ColorTemperature indicates the device supports color temperature control.
	ColorTemperature bool `json:"color_temperature,omitempty"`

	// Scripting indicates the device supports scripts (Gen2+).
	Scripting bool `json:"scripting,omitempty"`

	// Schedules indicates the device supports schedules.
	Schedules bool `json:"schedules,omitempty"`

	// AdvancedSchedules indicates the device supports cron-like schedules (Gen2+).
	AdvancedSchedules bool `json:"advanced_schedules,omitempty"`

	// Webhooks indicates the device supports webhooks (Gen2+).
	Webhooks bool `json:"webhooks,omitempty"`

	// KVS indicates the device supports key-value storage (Gen2+).
	KVS bool `json:"kvs,omitempty"`

	// VirtualComponents indicates the device supports virtual components (Gen2+).
	VirtualComponents bool `json:"virtual_components,omitempty"`

	// Actions indicates the device supports action URLs (Gen1).
	Actions bool `json:"actions,omitempty"`

	// SensorAddon indicates the device supports sensor add-on.
	SensorAddon bool `json:"sensor_addon,omitempty"`

	// ExternalSensors indicates the device supports external temperature sensors.
	ExternalSensors bool `json:"external_sensors,omitempty"`

	// Calibration indicates the device supports calibration (covers).
	Calibration bool `json:"calibration,omitempty"`

	// InputEvents indicates the device supports input events (button presses).
	InputEvents bool `json:"input_events,omitempty"`

	// Effects indicates the device supports lighting effects.
	Effects bool `json:"effects,omitempty"`

	// NoNeutral indicates the device works without a neutral wire.
	NoNeutral bool `json:"no_neutral,omitempty"`

	// BidirectionalMetering indicates the device supports bidirectional energy metering.
	BidirectionalMetering bool `json:"bidirectional_metering,omitempty"`

	// ThreePhase indicates the device supports 3-phase monitoring.
	ThreePhase bool `json:"three_phase,omitempty"`
}

// Protocols defines which communication protocols are supported.
type Protocols struct {
	// HTTP indicates HTTP REST API support (Gen1) or HTTP RPC (Gen2+).
	HTTP bool `json:"http"`

	// WebSocket indicates WebSocket RPC support.
	WebSocket bool `json:"websocket,omitempty"`

	// MQTT indicates MQTT support.
	MQTT bool `json:"mqtt,omitempty"`

	// CoIoT indicates CoIoT/CoAP support (Gen1 only).
	CoIoT bool `json:"coiot,omitempty"`

	// BLE indicates Bluetooth Low Energy support.
	BLE bool `json:"ble,omitempty"`

	// Matter indicates Matter protocol support (Gen4).
	Matter bool `json:"matter,omitempty"`

	// Zigbee indicates Zigbee protocol support (Gen4).
	Zigbee bool `json:"zigbee,omitempty"`

	// ZWave indicates Z-Wave protocol support (Wave series).
	ZWave bool `json:"zwave,omitempty"`

	// Ethernet indicates Ethernet support (Pro series).
	Ethernet bool `json:"ethernet,omitempty"`
}

// Limits defines resource limits for the device.
type Limits struct {
	// MaxScripts is the maximum number of scripts (0 = not supported).
	MaxScripts int `json:"max_scripts,omitempty"`

	// MaxSchedules is the maximum number of schedules.
	MaxSchedules int `json:"max_schedules,omitempty"`

	// MaxWebhooks is the maximum number of webhooks.
	MaxWebhooks int `json:"max_webhooks,omitempty"`

	// MaxKVSEntries is the maximum number of KVS entries.
	MaxKVSEntries int `json:"max_kvs_entries,omitempty"`

	// MaxScriptSize is the maximum script size in bytes.
	MaxScriptSize int `json:"max_script_size,omitempty"`

	// MaxInputCurrent is the maximum input current in Amps.
	MaxInputCurrent float64 `json:"max_input_current,omitempty"`

	// MaxOutputCurrent is the maximum output current in Amps.
	MaxOutputCurrent float64 `json:"max_output_current,omitempty"`

	// MaxPower is the maximum power handling in Watts.
	MaxPower float64 `json:"max_power,omitempty"`

	// MaxVoltage is the maximum voltage in Volts.
	MaxVoltage float64 `json:"max_voltage,omitempty"`

	// MinVoltage is the minimum voltage in Volts.
	MinVoltage float64 `json:"min_voltage,omitempty"`
}

// SensorType represents a type of sensor.
type SensorType string

const (
	SensorTemperature SensorType = "temperature"
	SensorHumidity    SensorType = "humidity"
	SensorMotion      SensorType = "motion"
	SensorContact     SensorType = "contact"
	SensorVibration   SensorType = "vibration"
	SensorTilt        SensorType = "tilt"
	SensorIlluminance SensorType = "illuminance"
	SensorFlood       SensorType = "flood"
	SensorSmoke       SensorType = "smoke"
	SensorGas         SensorType = "gas"
	SensorBattery     SensorType = "battery"
	SensorVoltage     SensorType = "voltage"
	SensorCurrent     SensorType = "current"
	SensorPower       SensorType = "power"
	SensorEnergy      SensorType = "energy"
	SensorPowerFactor SensorType = "power_factor"
)

// PowerSource indicates the device's power source.
type PowerSource string

const (
	// PowerSourceMains indicates mains (AC) powered.
	PowerSourceMains PowerSource = "mains"

	// PowerSourceBattery indicates battery powered.
	PowerSourceBattery PowerSource = "battery"

	// PowerSourceUSB indicates USB powered.
	PowerSourceUSB PowerSource = "usb"

	// PowerSourceDC indicates DC powered.
	PowerSourceDC PowerSource = "dc"

	// PowerSourcePoE indicates Power over Ethernet.
	PowerSourcePoE PowerSource = "poe"
)

// FormFactor indicates the physical form of the device.
type FormFactor string

const (
	// FormFactorDIN indicates DIN rail mount.
	FormFactorDIN FormFactor = "din"

	// FormFactorFlush indicates flush mount (behind switch).
	FormFactorFlush FormFactor = "flush"

	// FormFactorPlug indicates plug-in device.
	FormFactorPlug FormFactor = "plug"

	// FormFactorBulb indicates light bulb form factor.
	FormFactorBulb FormFactor = "bulb"

	// FormFactorSensor indicates standalone sensor.
	FormFactorSensor FormFactor = "sensor"

	// FormFactorWallMount indicates wall-mounted device.
	FormFactorWallMount FormFactor = "wall_mount"

	// FormFactorDesktop indicates desktop/tabletop device.
	FormFactorDesktop FormFactor = "desktop"

	// FormFactorOutdoor indicates outdoor-rated device.
	FormFactorOutdoor FormFactor = "outdoor"

	// FormFactorRadiator indicates radiator valve.
	FormFactorRadiator FormFactor = "radiator"
)

// HasComponent returns true if the device has the specified component type.
//
//nolint:gocyclo,cyclop // HasComponent checks all component types
func (p *Profile) HasComponent(ct types.ComponentType) bool {
	switch ct {
	case types.ComponentTypeSwitch:
		return p.Components.Switches > 0
	case types.ComponentTypeCover:
		return p.Components.Covers > 0 || p.Capabilities.CoverSupport
	case types.ComponentTypeLight:
		return p.Components.Lights > 0
	case types.ComponentTypeInput:
		return p.Components.Inputs > 0
	case types.ComponentTypePM, types.ComponentTypePM1:
		return p.Components.PowerMeters > 0
	case types.ComponentTypeEM, types.ComponentTypeEM1:
		return p.Components.EnergyMeters > 0
	case types.ComponentTypeVoltmeter:
		return p.Components.Voltmeters > 0
	case types.ComponentTypeTemperature:
		return p.Components.TemperatureSensors > 0 || p.hasSensor(SensorTemperature)
	case types.ComponentTypeHumidity:
		return p.Components.HumiditySensors > 0 || p.hasSensor(SensorHumidity)
	case types.ComponentTypeSmoke:
		return p.hasSensor(SensorSmoke)
	case types.ComponentTypeRGB, types.ComponentTypeRGBW:
		return p.Components.RGBChannels > 0
	case types.ComponentTypeThermostat:
		return p.Components.Thermostat
	default:
		return false
	}
}

// hasSensor checks if the profile includes a specific sensor type.
func (p *Profile) hasSensor(st SensorType) bool {
	for _, s := range p.Sensors {
		if s == st {
			return true
		}
	}
	return false
}

// ComponentCount returns the count of a specific component type.
func (p *Profile) ComponentCount(ct types.ComponentType) int {
	switch ct {
	case types.ComponentTypeSwitch:
		return p.Components.Switches
	case types.ComponentTypeCover:
		return p.Components.Covers
	case types.ComponentTypeLight:
		return p.Components.Lights
	case types.ComponentTypeInput:
		return p.Components.Inputs
	case types.ComponentTypePM, types.ComponentTypePM1:
		return p.Components.PowerMeters
	case types.ComponentTypeEM, types.ComponentTypeEM1:
		return p.Components.EnergyMeters
	case types.ComponentTypeVoltmeter:
		return p.Components.Voltmeters
	case types.ComponentTypeTemperature:
		return p.Components.TemperatureSensors
	case types.ComponentTypeHumidity:
		return p.Components.HumiditySensors
	case types.ComponentTypeRGB, types.ComponentTypeRGBW:
		return p.Components.RGBChannels
	default:
		return 0
	}
}

// IsGen1 returns true if this is a Gen1 device.
func (p *Profile) IsGen1() bool {
	return p.Generation == types.Gen1
}

// IsGen2Plus returns true if this is a Gen2 Plus series device.
func (p *Profile) IsGen2Plus() bool {
	return p.Generation == types.Gen2 && p.Series == SeriesPlus
}

// IsGen2Pro returns true if this is a Gen2 Pro series device.
func (p *Profile) IsGen2Pro() bool {
	return p.Generation == types.Gen2 && p.Series == SeriesPro
}

// IsGen3 returns true if this is a Gen3 device.
func (p *Profile) IsGen3() bool {
	return p.Generation == types.Gen3
}

// IsGen4 returns true if this is a Gen4 device.
func (p *Profile) IsGen4() bool {
	return p.Generation == types.Gen4
}

// IsBLU returns true if this is a BLU (Bluetooth) device.
func (p *Profile) IsBLU() bool {
	return p.Series == SeriesBLU
}

// IsWave returns true if this is a Wave (Z-Wave) device.
func (p *Profile) IsWave() bool {
	return p.Series == SeriesWave || p.Series == SeriesWavePro
}

// IsMini returns true if this is a Mini variant.
func (p *Profile) IsMini() bool {
	return p.Series == SeriesMini
}

// SupportsRPC returns true if the device supports RPC protocol (Gen2+).
func (p *Profile) SupportsRPC() bool {
	return p.Generation.IsRPC()
}

// SupportsREST returns true if the device uses REST API (Gen1).
func (p *Profile) SupportsREST() bool {
	return p.Generation.IsREST()
}

// IsBatteryPowered returns true if the device is battery powered.
func (p *Profile) IsBatteryPowered() bool {
	return p.PowerSource == PowerSourceBattery
}

// DefaultGen2Limits returns standard limits for Gen2+ devices.
func DefaultGen2Limits() Limits {
	return Limits{
		MaxScripts:    10,
		MaxSchedules:  20,
		MaxWebhooks:   20,
		MaxKVSEntries: 100,
		MaxScriptSize: 16384, // 16KB
	}
}

// DefaultGen1Limits returns standard limits for Gen1 devices.
func DefaultGen1Limits() Limits {
	return Limits{
		MaxSchedules: 10,
	}
}

// DefaultGen2Capabilities returns standard capabilities for Gen2+ devices.
func DefaultGen2Capabilities() Capabilities {
	return Capabilities{
		Scripting:         true,
		Schedules:         true,
		AdvancedSchedules: true,
		Webhooks:          true,
		KVS:               true,
		VirtualComponents: true,
		InputEvents:       true,
	}
}

// DefaultGen2Protocols returns standard protocols for Gen2+ devices.
func DefaultGen2Protocols() Protocols {
	return Protocols{
		HTTP:      true,
		WebSocket: true,
		MQTT:      true,
		BLE:       true,
	}
}

// DefaultGen1Capabilities returns standard capabilities for Gen1 devices.
func DefaultGen1Capabilities() Capabilities {
	return Capabilities{
		Schedules: true,
		Actions:   true,
	}
}

// DefaultGen1Protocols returns standard protocols for Gen1 devices.
func DefaultGen1Protocols() Protocols {
	return Protocols{
		HTTP:  true,
		MQTT:  true,
		CoIoT: true,
	}
}
