package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// wifiComponentType is the type identifier for the WiFi component.
const wifiComponentType = "wifi"

// WiFi represents a Shelly Gen2+ WiFi component.
//
// WiFi manages the device's wireless networking capabilities including:
//   - Station (STA) mode: Connect to existing WiFi networks
//   - Access Point (AP) mode: Create a local network for device access
//   - Roaming: Automatic switching between access points based on signal strength
//   - Range Extender: Extend WiFi coverage to other Shelly devices
//
// Note: WiFi component does not use component IDs like Switch or Input.
// It is a singleton component accessed via "wifi" key.
//
// Example:
//
//	wifi := components.NewWiFi(device.Client())
//	status, err := wifi.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Status: %s, SSID: %s\n", status.Status, *status.SSID)
//	}
type WiFi struct {
	client *rpc.Client
}

// NewWiFi creates a new WiFi component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	wifi := components.NewWiFi(device.Client())
func NewWiFi(client *rpc.Client) *WiFi {
	return &WiFi{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (w *WiFi) Client() *rpc.Client {
	return w.client
}

// WiFiConfig represents the configuration of the WiFi component.
//
// WiFi configuration includes station (STA) settings for connecting to networks,
// access point (AP) settings for hosting a network, and roaming settings.
type WiFiConfig struct {
	// AP is the access point configuration.
	// When enabled, the device creates its own WiFi network.
	AP *WiFiAPConfig `json:"ap,omitempty"`

	// STA is the primary station configuration.
	// Used to connect to an existing WiFi network.
	STA *WiFiStationConfig `json:"sta,omitempty"`

	// STA1 is the fallback station configuration.
	// Used when the device cannot connect to the primary STA network.
	STA1 *WiFiStationConfig `json:"sta1,omitempty"`

	// Roam configures access point roaming behavior.
	Roam *WiFiRoamConfig `json:"roam,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// WiFiAPConfig represents access point configuration.
type WiFiAPConfig struct {
	// SSID is the network name for the access point.
	// Read-only when retrieved from GetConfig, writable when setting.
	SSID *string `json:"ssid,omitempty"`

	// Pass is the password for the access point.
	// Write-only: not returned in GetConfig responses.
	// Must be provided when setting a password-protected AP.
	Pass *string `json:"pass,omitempty"`

	// IsOpen indicates if the access point is open (no password).
	// Read-only.
	IsOpen *bool `json:"is_open,omitempty"`

	// Enable enables or disables the access point.
	Enable *bool `json:"enable,omitempty"`

	// RangeExtender configures range extender functionality.
	// Available only on non-battery devices with range extender support.
	RangeExtender *WiFiAPRangeExtenderConfig `json:"range_extender,omitempty"`
}

// WiFiAPRangeExtenderConfig represents range extender configuration.
//
// Range extender enables a Gen2+ device to provide internet connectivity
// to other Shelly devices connected to its access point.
type WiFiAPRangeExtenderConfig struct {
	// Enable enables or disables the range extender functionality.
	Enable *bool `json:"enable,omitempty"`
}

// WiFiStationConfig represents station (client) configuration.
type WiFiStationConfig struct {
	// SSID is the network name to connect to.
	SSID *string `json:"ssid,omitempty"`

	// Pass is the network password.
	// Write-only: not returned in GetConfig responses.
	// Must be provided when connecting to a password-protected network.
	Pass *string `json:"pass,omitempty"`

	// IsOpen indicates if the network is open (no password required).
	// Read-only.
	IsOpen *bool `json:"is_open,omitempty"`

	// Enable enables or disables this station configuration.
	Enable *bool `json:"enable,omitempty"`

	// IPv4Mode specifies how to obtain an IP address.
	// Values: "dhcp" (automatic) or "static" (manual configuration).
	IPv4Mode *string `json:"ipv4mode,omitempty"`

	// IP is the static IP address.
	// Only used when IPv4Mode is "static".
	IP *string `json:"ip,omitempty"`

	// Netmask is the network mask for static IP.
	// Only used when IPv4Mode is "static".
	Netmask *string `json:"netmask,omitempty"`

	// GW is the gateway address for static IP.
	// Only used when IPv4Mode is "static".
	GW *string `json:"gw,omitempty"`

	// Nameserver is the DNS server address for static IP.
	// Only used when IPv4Mode is "static".
	Nameserver *string `json:"nameserver,omitempty"`
}

// WiFiRoamConfig represents roaming configuration.
//
// Roaming allows the device to automatically switch to a better access point
// when signal strength drops below a threshold.
type WiFiRoamConfig struct {
	// RSSIThr is the RSSI threshold in dBm.
	// When signal strength falls below this value, roaming is triggered.
	// Default: -80
	RSSIThr *float64 `json:"rssi_thr,omitempty"`

	// Interval is the scan interval in seconds.
	// How often to scan for better access points.
	// Set to 0 to disable roaming.
	// Default: 60
	Interval *float64 `json:"interval,omitempty"`
}

// WiFiStatus represents the current status of the WiFi component.
type WiFiStatus struct {
	StaIP         *string  `json:"sta_ip,omitempty"`
	SSID          *string  `json:"ssid,omitempty"`
	RSSI          *float64 `json:"rssi,omitempty"`
	APClientCount *int     `json:"ap_client_count,omitempty"`
	types.RawFields
	Status string `json:"status,omitempty"`
}

// WiFiScanResult represents a single network found during WiFi scanning.
type WiFiScanResult struct {
	// SSID is the network name.
	SSID *string `json:"ssid,omitempty"`

	// BSSID is the MAC address of the access point.
	BSSID *string `json:"bssid,omitempty"`

	// Auth is the authentication type.
	// Values: "open", "wep", "wpa_psk", "wpa2_psk", "wpa_wpa2_psk", "wpa2_enterprise", "wpa3_psk"
	Auth *string `json:"auth,omitempty"`

	// Channel is the WiFi channel number.
	Channel *int `json:"channel,omitempty"`

	// RSSI is the signal strength in dBm.
	RSSI *float64 `json:"rssi,omitempty"`
}

// WiFiScanResponse represents the response from Wifi.Scan.
type WiFiScanResponse struct {
	types.RawFields
	Results []WiFiScanResult `json:"results,omitempty"`
}

// WiFiAPClient represents a client connected to the device's access point.
type WiFiAPClient struct {
	IP    *string `json:"ip,omitempty"`
	Since *int64  `json:"since,omitempty"`
	MAC   string  `json:"mac"`
}

// WiFiListAPClientsResponse represents the response from Wifi.ListAPClients.
type WiFiListAPClientsResponse struct {
	TS *int64 `json:"ts,omitempty"`
	types.RawFields
	APClients []WiFiAPClient `json:"ap_clients,omitempty"`
}

// GetConfig retrieves the WiFi configuration.
//
// Note: Passwords (pass fields) are not returned for security reasons.
//
// Example:
//
//	config, err := wifi.GetConfig(ctx)
//	if err == nil && config.STA != nil {
//	    fmt.Printf("Connected to: %s\n", *config.STA.SSID)
//	}
func (w *WiFi) GetConfig(ctx context.Context) (*WiFiConfig, error) {
	resultJSON, err := w.client.Call(ctx, "Wifi.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config WiFiConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the WiFi configuration.
//
// Only non-nil fields will be updated. When setting a password-protected network,
// you must include the "pass" field along with the SSID.
//
// Example - Connect to a network:
//
//	err := wifi.SetConfig(ctx, &WiFiConfig{
//	    STA: &WiFiStationConfig{
//	        SSID:   ptr("MyNetwork"),
//	        Pass:   ptr("password123"),
//	        Enable: ptr(true),
//	    },
//	})
//
// Example - Enable access point:
//
//	err := wifi.SetConfig(ctx, &WiFiConfig{
//	    AP: &WiFiAPConfig{
//	        SSID:   ptr("ShellyAP"),
//	        Pass:   ptr("appassword"),
//	        Enable: ptr(true),
//	    },
//	})
func (w *WiFi) SetConfig(ctx context.Context, config *WiFiConfig) error {
	params := map[string]any{
		"config": config,
	}

	_, err := w.client.Call(ctx, "Wifi.SetConfig", params)
	return err
}

// GetStatus retrieves the current WiFi status.
//
// Returns connection status, IP address, SSID, and signal strength.
//
// Example:
//
//	status, err := wifi.GetStatus(ctx)
//	if err == nil {
//	    switch status.Status {
//	    case "got ip":
//	        fmt.Printf("Connected with IP: %s\n", *status.StaIP)
//	    case "disconnected":
//	        fmt.Println("Not connected to any network")
//	    }
//	}
func (w *WiFi) GetStatus(ctx context.Context) (*WiFiStatus, error) {
	resultJSON, err := w.client.Call(ctx, "Wifi.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status WiFiStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Scan scans for available WiFi networks.
//
// Returns a list of all visible networks with their SSID, signal strength,
// authentication type, and channel information.
//
// Note: Scanning may take several seconds to complete.
//
// Example:
//
//	result, err := wifi.Scan(ctx)
//	if err == nil {
//	    for _, network := range result.Results {
//	        fmt.Printf("Network: %s (RSSI: %.0f dBm)\n", *network.SSID, *network.RSSI)
//	    }
//	}
func (w *WiFi) Scan(ctx context.Context) (*WiFiScanResponse, error) {
	resultJSON, err := w.client.Call(ctx, "Wifi.Scan", nil)
	if err != nil {
		return nil, err
	}

	var result WiFiScanResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListAPClients lists clients connected to the device's access point.
//
// This method is only available when the access point is enabled.
// Returns MAC addresses and IP addresses of connected clients.
//
// Example:
//
//	result, err := wifi.ListAPClients(ctx)
//	if err == nil {
//	    fmt.Printf("%d clients connected\n", len(result.APClients))
//	    for _, client := range result.APClients {
//	        fmt.Printf("  MAC: %s, IP: %s\n", client.MAC, *client.IP)
//	    }
//	}
func (w *WiFi) ListAPClients(ctx context.Context) (*WiFiListAPClientsResponse, error) {
	resultJSON, err := w.client.Call(ctx, "Wifi.ListAPClients", nil)
	if err != nil {
		return nil, err
	}

	var result WiFiListAPClientsResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Type returns the component type identifier.
func (w *WiFi) Type() string {
	return wifiComponentType
}

// Key returns the component key for aggregated status/config responses.
func (w *WiFi) Key() string {
	return wifiComponentType
}

// Ensure WiFi implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*WiFi)(nil)
