package provisioning

import (
	"github.com/tj-smith47/shelly-go/types"
)

// WiFiConfig represents WiFi configuration for provisioning.
type WiFiConfig struct {
	// SSID is the network name to connect to.
	SSID string `json:"ssid"`

	// Password is the network password.
	Password string `json:"pass,omitempty"`

	// Enable enables station mode (default true when SSID is set).
	Enable *bool `json:"enable,omitempty"`

	// StaticIP is an optional static IP address.
	StaticIP string `json:"ipv4mode,omitempty"`

	// IP is the static IP address (when StaticIP is "static").
	IP string `json:"ip,omitempty"`

	// Netmask is the network mask (when StaticIP is "static").
	Netmask string `json:"netmask,omitempty"`

	// Gateway is the default gateway (when StaticIP is "static").
	Gateway string `json:"gw,omitempty"`

	// Nameserver is the DNS server (when StaticIP is "static").
	Nameserver string `json:"nameserver,omitempty"`
}

// APConfig represents Access Point configuration.
type APConfig struct {
	Enable        *bool  `json:"enable,omitempty"`
	RangeExtender *bool  `json:"range_extender,omitempty"`
	SSID          string `json:"ssid,omitempty"`
	Password      string `json:"pass,omitempty"`
}

// AuthConfig represents authentication configuration.
type AuthConfig struct {
	// Enable enables HTTP authentication.
	Enable *bool `json:"enable,omitempty"`

	// User is the username (typically "admin").
	User string `json:"user,omitempty"`

	// Password is the authentication password.
	Password string `json:"pass,omitempty"`
}

// CloudConfig represents Shelly Cloud configuration.
type CloudConfig struct {
	// Enable enables Shelly Cloud connection.
	Enable *bool `json:"enable,omitempty"`

	// Server is the cloud server URL (optional, uses default).
	Server string `json:"server,omitempty"`
}

// DeviceConfig represents complete device configuration for provisioning.
type DeviceConfig struct {
	WiFi       *WiFiConfig  `json:"wifi,omitempty"`
	AP         *APConfig    `json:"ap,omitempty"`
	Auth       *AuthConfig  `json:"auth,omitempty"`
	Cloud      *CloudConfig `json:"cloud,omitempty"`
	Location   *Location    `json:"location,omitempty"`
	DeviceName string       `json:"name,omitempty"`
	Timezone   string       `json:"timezone,omitempty"`
}

// Location represents geographic coordinates.
type Location struct {
	// Lat is the latitude (-90 to 90).
	Lat float64 `json:"lat"`

	// Lon is the longitude (-180 to 180).
	Lon float64 `json:"lon"`
}

// ProvisionResult represents the result of a provisioning operation.
type ProvisionResult struct {
	Error      error
	DeviceInfo *DeviceInfo
	Address    string
	NewAddress string
	Success    bool
}

// DeviceInfo contains basic device information.
type DeviceInfo struct {
	types.RawFields
	ID         string `json:"id,omitempty"`
	Model      string `json:"model,omitempty"`
	App        string `json:"app,omitempty"`
	Version    string `json:"ver,omitempty"`
	MAC        string `json:"mac,omitempty"`
	Generation int    `json:"gen,omitempty"`
}

// WiFiStatus represents current WiFi status.
type WiFiStatus struct {
	types.RawFields
	StaIP  string `json:"sta_ip,omitempty"`
	Status string `json:"status,omitempty"`
	SSID   string `json:"ssid,omitempty"`
	RSSI   int    `json:"rssi,omitempty"`
}

// ProvisionOptions represents options for the provisioning process.
type ProvisionOptions struct {
	ConnectionTimeout int
	WaitForConnection bool
	VerifyConnection  bool
	DisableAP         bool
	DisableBLE        bool
}

// DefaultProvisionOptions returns the default provisioning options.
func DefaultProvisionOptions() *ProvisionOptions {
	return &ProvisionOptions{
		WaitForConnection: true,
		ConnectionTimeout: 30,
		VerifyConnection:  true,
		DisableAP:         false,
		DisableBLE:        false,
	}
}
