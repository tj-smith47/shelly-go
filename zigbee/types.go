package zigbee

import (
	"github.com/tj-smith47/shelly-go/types"
)

// Config represents the Zigbee component configuration.
type Config struct {
	types.RawFields
	Enable bool `json:"enable"`
}

// Status represents the Zigbee component status.
type Status struct {
	types.RawFields
	NetworkState     string `json:"network_state,omitempty"`
	EUI64            string `json:"eui64,omitempty"`
	CoordinatorEUI64 string `json:"coordinator_eui64,omitempty"`
	Channel          int    `json:"channel,omitempty"`
	PANID            uint16 `json:"pan_id,omitempty"`
}

// SetConfigParams represents parameters for setting Zigbee configuration.
type SetConfigParams struct {
	// Enable enables or disables Zigbee on the device.
	Enable *bool `json:"enable,omitempty"`
}

// NetworkState constants for Zigbee network states.
const (
	// NetworkStateDisabled indicates Zigbee is disabled.
	NetworkStateDisabled = "disabled"

	// NetworkStateInitializing indicates Zigbee stack is initializing.
	NetworkStateInitializing = "initializing"

	// NetworkStateNotConfigured indicates Zigbee is disabled or not set up.
	//
	// Deprecated: Use NetworkStateDisabled instead.
	NetworkStateNotConfigured = "not_configured"

	// NetworkStateReady indicates Zigbee is enabled but not joined to a network.
	//
	// Deprecated: Use NetworkStateInitializing instead.
	NetworkStateReady = "ready"

	// NetworkStateSteering indicates the device is attempting to join a network.
	NetworkStateSteering = "steering"

	// NetworkStateJoined indicates the device has successfully joined a network.
	NetworkStateJoined = "joined"

	// NetworkStateFailed indicates joining the network failed.
	NetworkStateFailed = "failed"
)

// DiscoveredDevice represents a Zigbee-capable Shelly device discovered on the network.
type DiscoveredDevice struct {
	ZigbeeStatus *Status          `json:"zigbee_status,omitempty"`
	Address      string           `json:"address"`
	DeviceID     string           `json:"device_id"`
	Model        string           `json:"model"`
	Generation   types.Generation `json:"generation"`
	HasZigbee    bool             `json:"has_zigbee"`
}

// PairingState represents the current state of the pairing workflow.
type PairingState string

const (
	// PairingStateIdle indicates no pairing is in progress.
	PairingStateIdle PairingState = "idle"

	// PairingStateEnabling indicates Zigbee is being enabled.
	PairingStateEnabling PairingState = "enabling"

	// PairingStateSteering indicates network steering is in progress.
	PairingStateSteering PairingState = "steering"

	// PairingStateJoined indicates the device has joined a network.
	PairingStateJoined PairingState = "joined"

	// PairingStateFailed indicates pairing failed.
	PairingStateFailed PairingState = "failed"

	// PairingStateTimeout indicates pairing timed out.
	PairingStateTimeout PairingState = "timeout"
)

// PairingResult represents the result of a pairing operation.
type PairingResult struct {
	Error       error        `json:"error,omitempty"`
	NetworkInfo *NetworkInfo `json:"network_info,omitempty"`
	State       PairingState `json:"state"`
}

// NetworkInfo contains information about a joined Zigbee network.
type NetworkInfo struct {
	CoordinatorEUI64 string `json:"coordinator_eui64"`
	Channel          int    `json:"channel"`
	PANID            uint16 `json:"pan_id"`
}

// ClusterCapability represents a Zigbee cluster capability mapping.
type ClusterCapability struct {
	ClusterName   string             `json:"cluster_name"`
	ComponentType string             `json:"component_type"`
	Attributes    []ClusterAttribute `json:"attributes,omitempty"`
	Commands      []ClusterCommand   `json:"commands,omitempty"`
	ClusterID     uint16             `json:"cluster_id"`
}

// ClusterAttribute represents a Zigbee cluster attribute.
type ClusterAttribute struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	ID         uint16 `json:"id"`
	Readable   bool   `json:"readable"`
	Writable   bool   `json:"writable"`
	Reportable bool   `json:"reportable"`
}

// ClusterCommand represents a Zigbee cluster command.
type ClusterCommand struct {
	Name      string `json:"name"`
	Direction string `json:"direction"`
	ID        uint8  `json:"id"`
}

// DeviceType represents the Zigbee device type.
type DeviceType uint16

// Zigbee device type constants.
const (
	// DeviceTypeOnOffSwitch is a basic on/off switch (0x0000).
	DeviceTypeOnOffSwitch DeviceType = 0x0000

	// DeviceTypeLevelControllableOutput is a dimmable output (0x0003).
	DeviceTypeLevelControllableOutput DeviceType = 0x0003

	// DeviceTypeOnOffLight is an on/off light (0x0100).
	DeviceTypeOnOffLight DeviceType = 0x0100

	// DeviceTypeDimmableLight is a dimmable light (0x0101).
	DeviceTypeDimmableLight DeviceType = 0x0101

	// DeviceTypeColorDimmableLight is a color dimmable light (0x0102).
	DeviceTypeColorDimmableLight DeviceType = 0x0102

	// DeviceTypeOnOffLightSwitch is an on/off light switch (0x0103).
	DeviceTypeOnOffLightSwitch DeviceType = 0x0103

	// DeviceTypeDimmerSwitch is a dimmer switch (0x0104).
	DeviceTypeDimmerSwitch DeviceType = 0x0104

	// DeviceTypeColorDimmerSwitch is a color dimmer switch (0x0105).
	DeviceTypeColorDimmerSwitch DeviceType = 0x0105

	// DeviceTypeWindowCovering is a window covering/shade (0x0202).
	DeviceTypeWindowCovering DeviceType = 0x0202

	// DeviceTypeThermostat is a thermostat (0x0301).
	DeviceTypeThermostat DeviceType = 0x0301

	// DeviceTypeTemperatureSensor is a temperature sensor (0x0302).
	DeviceTypeTemperatureSensor DeviceType = 0x0302

	// DeviceTypePumpController is a pump controller (0x0303).
	DeviceTypePumpController DeviceType = 0x0303

	// DeviceTypeOccupancySensor is an occupancy/motion sensor (0x0107).
	DeviceTypeOccupancySensor DeviceType = 0x0107

	// DeviceTypeContactSensor is a door/window contact sensor (0x0402).
	DeviceTypeContactSensor DeviceType = 0x0402

	// DeviceTypeFloodSensor is a water/flood sensor (0x0403).
	DeviceTypeFloodSensor DeviceType = 0x0403

	// DeviceTypeSmokeSensor is a smoke sensor (0x0404).
	DeviceTypeSmokeSensor DeviceType = 0x0404

	// DeviceTypePowerMeter is a power/energy meter (0x0501).
	DeviceTypePowerMeter DeviceType = 0x0501
)
