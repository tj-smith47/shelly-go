package types

import "context"

// Component represents a device component (switch, cover, light, sensor, etc.).
// Components provide specific functionality within a device and follow a
// consistent GetConfig/SetConfig/GetStatus pattern.
//
// Implementations should be safe for concurrent use.
type Component interface {
	// ID returns the component identifier (e.g., 0, 1, 2 for multiple instances).
	ID() int

	// Type returns the component type (switch, cover, light, etc.).
	Type() ComponentType

	// GetStatus returns the current component status.
	GetStatus(ctx context.Context) (any, error)
}

// ConfigurableComponent is a component that supports configuration.
type ConfigurableComponent interface {
	Component

	// GetConfig returns the component configuration.
	GetConfig(ctx context.Context) (any, error)

	// SetConfig updates the component configuration.
	SetConfig(ctx context.Context, config any) error
}

// ComponentType represents the type of component.
type ComponentType string

// Component types for Gen2+ devices.
const (
	ComponentTypeSwitch       ComponentType = "switch"
	ComponentTypeCover        ComponentType = "cover"
	ComponentTypeLight        ComponentType = "light"
	ComponentTypeRGB          ComponentType = "rgb"
	ComponentTypeRGBW         ComponentType = "rgbw"
	ComponentTypeInput        ComponentType = "input"
	ComponentTypePM           ComponentType = "pm"
	ComponentTypePM1          ComponentType = "pm1"
	ComponentTypeEM           ComponentType = "em"
	ComponentTypeEM1          ComponentType = "em1"
	ComponentTypeEMData       ComponentType = "emdata"
	ComponentTypeEM1Data      ComponentType = "em1data"
	ComponentTypeVoltmeter    ComponentType = "voltmeter"
	ComponentTypeTemperature  ComponentType = "temperature"
	ComponentTypeHumidity     ComponentType = "humidity"
	ComponentTypeDevicePower  ComponentType = "devicepower"
	ComponentTypeSmoke        ComponentType = "smoke"
	ComponentTypeThermostat   ComponentType = "thermostat"
	ComponentTypeWiFi         ComponentType = "wifi"
	ComponentTypeEthernet     ComponentType = "eth"
	ComponentTypeBLE          ComponentType = "ble"
	ComponentTypeCloud        ComponentType = "cloud"
	ComponentTypeMQTT         ComponentType = "mqtt"
	ComponentTypeWebhook      ComponentType = "webhook"
	ComponentTypeWS           ComponentType = "ws"
	ComponentTypeSys          ComponentType = "sys"
	ComponentTypeScript       ComponentType = "script"
	ComponentTypeSchedule     ComponentType = "schedule"
	ComponentTypeKVS          ComponentType = "kvs"
	ComponentTypeUI           ComponentType = "ui"
	ComponentTypeBTHome       ComponentType = "bthome"
	ComponentTypeBTHomeDevice ComponentType = "bthomedevice"
	ComponentTypeBTHomeSensor ComponentType = "bthomesensor"
	ComponentTypeModBus       ComponentType = "modbus"
)

// String returns the string representation of the component type.
func (c ComponentType) String() string {
	return string(c)
}

// ComponentID represents a component identifier in the format "type:id".
// For example: "switch:0", "cover:1", "light:2".
type ComponentID string

// ParseComponentID parses a component ID string.
func ParseComponentID(s string) ComponentID {
	return ComponentID(s)
}

// String returns the string representation.
func (c ComponentID) String() string {
	return string(c)
}
