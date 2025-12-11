package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// sysComponentType is the type identifier for the Sys component.
const sysComponentType = "sys"

// Sys represents a Shelly Gen2+ System component.
//
// Sys provides access to system-level configuration and status including:
//   - Device identification (name, MAC address, firmware)
//   - Location and timezone settings
//   - Debug logging configuration
//   - System resources (RAM, filesystem)
//   - Available firmware updates
//   - Restart/wake-up information
//
// Note: Sys component does not use component IDs.
// It is a singleton component accessed via "sys" key.
//
// Example:
//
//	sys := components.NewSys(device.Client())
//	status, err := sys.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Uptime: %d seconds\n", status.Uptime)
//	}
type Sys struct {
	client *rpc.Client
}

// NewSys creates a new Sys (System) component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	sys := components.NewSys(device.Client())
func NewSys(client *rpc.Client) *Sys {
	return &Sys{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (s *Sys) Client() *rpc.Client {
	return s.client
}

// SysDeviceConfig represents device identification configuration.
type SysDeviceConfig struct {
	// Name is a user-assigned name for the device.
	Name *string `json:"name,omitempty"`

	// MAC is the device's MAC address (read-only).
	MAC *string `json:"mac,omitempty"`

	// FwID is the current firmware identifier (read-only).
	FwID *string `json:"fw_id,omitempty"`

	// Profile is the current device profile (for multi-profile devices).
	Profile *string `json:"profile,omitempty"`

	// EcoMode enables low-power mode when true.
	EcoMode *bool `json:"eco_mode,omitempty"`

	// Discoverable makes the device visible on local network when true.
	Discoverable *bool `json:"discoverable,omitempty"`

	// Addon is the type of connected addon.
	Addon *string `json:"addon,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// SysLocationConfig represents location/timezone configuration.
type SysLocationConfig struct {
	// TZ is the timezone in POSIX TZ format (e.g., "America/New_York").
	TZ *string `json:"tz,omitempty"`

	// Lat is the latitude in degrees (-90 to 90).
	Lat *float64 `json:"lat,omitempty"`

	// Lng is the longitude in degrees (-180 to 180).
	Lng *float64 `json:"lng,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// SysDebugConfig represents debug logging configuration.
type SysDebugConfig struct {
	// MQTT enables debug logs via MQTT.
	MQTT *SysDebugTargetConfig `json:"mqtt,omitempty"`

	// Websocket enables debug logs via WebSocket.
	Websocket *SysDebugTargetConfig `json:"websocket,omitempty"`

	// UDP enables debug logs via UDP.
	UDP *SysDebugUDPConfig `json:"udp,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// SysDebugTargetConfig represents debug target (MQTT/WebSocket) configuration.
type SysDebugTargetConfig struct {
	// Enable enables debug logging to this target.
	Enable *bool `json:"enable,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// SysDebugUDPConfig represents UDP debug logging configuration.
type SysDebugUDPConfig struct {
	// Addr is the destination address for UDP debug logs (host:port).
	Addr *string `json:"addr,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// SysRPCUDPConfig represents UDP RPC configuration.
type SysRPCUDPConfig struct {
	// DstAddr is the destination address for UDP RPC responses.
	DstAddr *string `json:"dst_addr,omitempty"`

	// ListenPort is the port for incoming UDP RPC requests.
	ListenPort *int `json:"listen_port,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// SysSNTPConfig represents SNTP (time synchronization) configuration.
type SysSNTPConfig struct {
	// Server is the SNTP server address.
	Server *string `json:"server,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// SysConfig represents the configuration of the Sys component.
type SysConfig struct {
	// Device contains device identification settings.
	Device *SysDeviceConfig `json:"device,omitempty"`

	// Location contains timezone and geo-location settings.
	Location *SysLocationConfig `json:"location,omitempty"`

	// Debug contains debug logging settings.
	Debug *SysDebugConfig `json:"debug,omitempty"`

	// UIData is custom data for UI applications (max 256 bytes).
	UIData any `json:"ui_data,omitempty"`

	// RPCUDP contains UDP RPC settings.
	RPCUDP *SysRPCUDPConfig `json:"rpc_udp,omitempty"`

	// SNTP contains SNTP server settings.
	SNTP *SysSNTPConfig `json:"sntp,omitempty"`

	// CfgRev is the configuration revision number (read-only).
	CfgRev *int `json:"cfg_rev,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// SysAvailableUpdates represents available firmware updates.
type SysAvailableUpdates struct {
	// Stable is the latest stable firmware version available.
	Stable *SysFirmwareVersion `json:"stable,omitempty"`

	// Beta is the latest beta firmware version available.
	Beta *SysFirmwareVersion `json:"beta,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// SysFirmwareVersion represents a firmware version.
type SysFirmwareVersion struct {
	types.RawFields
	Version string `json:"version"`
	BuildID string `json:"build_id,omitempty"`
}

// SysWakeupReason represents the reason a battery device woke up.
type SysWakeupReason struct {
	types.RawFields
	Boot  string `json:"boot"`
	Cause string `json:"cause"`
}

// SysStatus represents the current status of the Sys component.
type SysStatus struct {
	ScheduleRev *int `json:"schedule_rev,omitempty"`
	types.RawFields
	Time             *string              `json:"time,omitempty"`
	Unixtime         *int64               `json:"unixtime,omitempty"`
	ResetReason      *int                 `json:"reset_reason,omitempty"`
	WakeupPeriod     *int                 `json:"wakeup_period,omitempty"`
	WakeupReason     *SysWakeupReason     `json:"wakeup_reason,omitempty"`
	AvailableUpdates *SysAvailableUpdates `json:"available_updates,omitempty"`
	WebhookRev       *int                 `json:"webhook_rev,omitempty"`
	MAC              string               `json:"mac"`
	Uptime           int                  `json:"uptime"`
	KVSRev           int                  `json:"kvs_rev"`
	CfgRev           int                  `json:"cfg_rev"`
	FSFree           int                  `json:"fs_free"`
	FSSize           int                  `json:"fs_size"`
	RAMFree          int                  `json:"ram_free"`
	RAMSize          int                  `json:"ram_size"`
	RestartRequired  bool                 `json:"restart_required"`
}

// GetConfig retrieves the Sys configuration.
//
// Example:
//
//	config, err := sys.GetConfig(ctx)
//	if err == nil && config.Device != nil && config.Device.Name != nil {
//	    fmt.Printf("Device name: %s\n", *config.Device.Name)
//	}
func (s *Sys) GetConfig(ctx context.Context) (*SysConfig, error) {
	resultJSON, err := s.client.Call(ctx, "Sys.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config SysConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the Sys configuration.
//
// Only non-nil fields will be updated.
//
// Example - Set device name:
//
//	err := sys.SetConfig(ctx, &SysConfig{
//	    Device: &SysDeviceConfig{
//	        Name: ptr("My Shelly Device"),
//	    },
//	})
//
// Example - Set timezone:
//
//	err := sys.SetConfig(ctx, &SysConfig{
//	    Location: &SysLocationConfig{
//	        TZ: ptr("America/New_York"),
//	    },
//	})
func (s *Sys) SetConfig(ctx context.Context, config *SysConfig) error {
	params := map[string]any{
		"config": config,
	}

	_, err := s.client.Call(ctx, "Sys.SetConfig", params)
	return err
}

// GetStatus retrieves the current Sys status.
//
// Returns system information including uptime, memory usage, and available updates.
//
// Example:
//
//	status, err := sys.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("MAC: %s\n", status.MAC)
//	    fmt.Printf("Uptime: %d seconds\n", status.Uptime)
//	    fmt.Printf("Free RAM: %d bytes\n", status.RAMFree)
//	    if status.AvailableUpdates != nil && status.AvailableUpdates.Stable != nil {
//	        fmt.Printf("Update available: %s\n", status.AvailableUpdates.Stable.Version)
//	    }
//	}
func (s *Sys) GetStatus(ctx context.Context) (*SysStatus, error) {
	resultJSON, err := s.client.Call(ctx, "Sys.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status SysStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (s *Sys) Type() string {
	return sysComponentType
}

// Key returns the component key for aggregated status/config responses.
func (s *Sys) Key() string {
	return sysComponentType
}

// Ensure Sys implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Sys)(nil)
