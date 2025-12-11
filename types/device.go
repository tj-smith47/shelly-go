package types

import "context"

// Device represents a Shelly device and provides methods for device-level operations.
// Different generations have different implementations, but all implement this interface.
//
// Implementations must be safe for concurrent use.
type Device interface {
	// GetDeviceInfo returns device identification and capabilities.
	GetDeviceInfo(ctx context.Context) (*DeviceInfo, error)

	// GetStatus returns the current status of all device components.
	GetStatus(ctx context.Context) (any, error)

	// GetConfig returns the current device configuration.
	GetConfig(ctx context.Context) (any, error)

	// Reboot reboots the device.
	Reboot(ctx context.Context) error

	// Generation returns the device generation.
	Generation() Generation
}

// DeviceInfo contains device identification and capability information.
// This is a unified structure that works across all device generations.
type DeviceInfo struct {
	RawFields    `json:"-"`
	WiFi         *WiFiInfo  `json:"wifi,omitempty"`
	Cloud        *CloudInfo `json:"cloud,omitempty"`
	MAC          string     `json:"mac"`
	Name         string     `json:"name,omitempty"`
	Model        string     `json:"model"`
	ID           string     `json:"id"`
	App          string     `json:"app,omitempty"`
	Version      string     `json:"version"`
	Profile      string     `json:"profile,omitempty"`
	AuthDomain   string     `json:"auth_domain,omitempty"`
	FirmwareID   string     `json:"fw_id,omitempty"`
	Generation   Generation `json:"gen"`
	Discoverable bool       `json:"discoverable,omitempty"`
	AuthEnabled  bool       `json:"auth_en"`
}

// WiFiInfo contains WiFi configuration information.
type WiFiInfo struct {
	RawFields `json:"-"`
	SSID      string `json:"ssid,omitempty"`
	StaIP     string `json:"sta_ip,omitempty"`
	RSSI      int    `json:"rssi,omitempty"`
}

// CloudInfo contains cloud connection information.
type CloudInfo struct {
	RawFields `json:"-"`
	Enabled   bool `json:"enabled"`
	Connected bool `json:"connected"`
}

// DeviceType represents the type of device (switch, cover, light, etc.).
type DeviceType string

const (
	DeviceTypeSwitch     DeviceType = "switch"
	DeviceTypeCover      DeviceType = "cover"
	DeviceTypeLight      DeviceType = "light"
	DeviceTypeDimmer     DeviceType = "dimmer"
	DeviceTypePlug       DeviceType = "plug"
	DeviceTypeRelay      DeviceType = "relay"
	DeviceTypeSensor     DeviceType = "sensor"
	DeviceTypePowerMeter DeviceType = "power_meter"
	DeviceTypeGateway    DeviceType = "gateway"
)

// String returns the string representation of the device type.
func (d DeviceType) String() string {
	return string(d)
}
