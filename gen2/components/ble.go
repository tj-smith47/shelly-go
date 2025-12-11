package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// BLE represents a Shelly Gen2+ Bluetooth Low Energy component.
//
// BLE provides Bluetooth functionality for Shelly devices including:
//   - BLE RPC communication (allows control via Bluetooth)
//   - BLE Observer mode (receives broadcasts from BLU sensors)
//   - Device provisioning via Bluetooth
//
// Note: BLE component does not use component IDs.
// It is a singleton component accessed via "ble" key.
//
// Example:
//
//	ble := components.NewBLE(device.Client())
//	config, err := ble.GetConfig(ctx)
//	if err == nil && config.Enable != nil && *config.Enable {
//	    fmt.Println("Bluetooth is enabled")
//	}
type BLE struct {
	client *rpc.Client
}

// NewBLE creates a new BLE component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	ble := components.NewBLE(device.Client())
func NewBLE(client *rpc.Client) *BLE {
	return &BLE{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (b *BLE) Client() *rpc.Client {
	return b.client
}

// BLEConfig represents the configuration of the BLE component.
type BLEConfig struct {
	// Enable enables or disables Bluetooth.
	Enable *bool `json:"enable,omitempty"`

	// RPC configures the Bluetooth RPC service.
	// When enabled, the device can be controlled via Bluetooth.
	RPC *BLERPCConfig `json:"rpc,omitempty"`

	// Observer configures the BLE observer mode.
	// When enabled, the device can receive broadcasts from BLU sensors.
	// Not applicable for battery-operated devices.
	Observer *BLEObserverConfig `json:"observer,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// BLERPCConfig represents BLE RPC service configuration.
type BLERPCConfig struct {
	// Enable enables or disables the BLE RPC service.
	// When enabled, the device accepts RPC commands via Bluetooth.
	Enable *bool `json:"enable,omitempty"`
}

// BLEObserverConfig represents BLE observer configuration.
//
// The observer mode allows the device to receive broadcasts from
// Shelly BLU devices (buttons, door sensors, motion sensors, etc.).
type BLEObserverConfig struct {
	// Enable enables or disables the BLE observer.
	// Not applicable for battery-operated devices.
	Enable *bool `json:"enable,omitempty"`
}

// BLEStatus represents the current status of the BLE component.
//
// Currently, BLE status contains minimal information as most state
// is reflected in the configuration.
type BLEStatus struct {
	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// GetConfig retrieves the BLE configuration.
//
// Example:
//
//	config, err := ble.GetConfig(ctx)
//	if err == nil {
//	    if config.Enable != nil && *config.Enable {
//	        fmt.Println("Bluetooth is enabled")
//	    }
//	    if config.Observer != nil && config.Observer.Enable != nil && *config.Observer.Enable {
//	        fmt.Println("BLE Observer mode is active")
//	    }
//	}
func (b *BLE) GetConfig(ctx context.Context) (*BLEConfig, error) {
	resultJSON, err := b.client.Call(ctx, "BLE.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config BLEConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the BLE configuration.
//
// Only non-nil fields will be updated.
//
// Example - Enable Bluetooth and RPC:
//
//	err := ble.SetConfig(ctx, &BLEConfig{
//	    Enable: ptr(true),
//	    RPC: &BLERPCConfig{
//	        Enable: ptr(true),
//	    },
//	})
//
// Example - Enable observer for BLU sensors:
//
//	err := ble.SetConfig(ctx, &BLEConfig{
//	    Enable: ptr(true),
//	    Observer: &BLEObserverConfig{
//	        Enable: ptr(true),
//	    },
//	})
func (b *BLE) SetConfig(ctx context.Context, config *BLEConfig) error {
	params := map[string]any{
		"config": config,
	}

	_, err := b.client.Call(ctx, "BLE.SetConfig", params)
	return err
}

// GetStatus retrieves the current BLE status.
//
// Note: BLE status is minimal; most BLE state is in configuration.
//
// Example:
//
//	status, err := ble.GetStatus(ctx)
//	if err == nil {
//	    fmt.Println("BLE status retrieved successfully")
//	}
func (b *BLE) GetStatus(ctx context.Context) (*BLEStatus, error) {
	resultJSON, err := b.client.Call(ctx, "BLE.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status BLEStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (b *BLE) Type() string {
	return "ble"
}

// Key returns the component key for aggregated status/config responses.
func (b *BLE) Key() string {
	return "ble"
}

// Ensure BLE implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*BLE)(nil)
