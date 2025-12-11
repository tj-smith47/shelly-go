package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Cloud represents a Shelly Gen2+ Cloud component.
//
// Cloud manages the device's connection to the Shelly Cloud service,
// which provides remote access, monitoring, and control capabilities
// through the Shelly mobile app and cloud.shelly.cloud web interface.
//
// Note: Cloud component does not use component IDs.
// It is a singleton component accessed via "cloud" key.
//
// Example:
//
//	cloud := components.NewCloud(device.Client())
//	status, err := cloud.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Cloud connected: %t\n", status.Connected)
//	}
type Cloud struct {
	client *rpc.Client
}

// NewCloud creates a new Cloud component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	cloud := components.NewCloud(device.Client())
func NewCloud(client *rpc.Client) *Cloud {
	return &Cloud{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (c *Cloud) Client() *rpc.Client {
	return c.client
}

// CloudConfig represents the configuration of the Cloud component.
type CloudConfig struct {
	// Enable enables or disables the Shelly Cloud connection.
	Enable *bool `json:"enable,omitempty"`

	// Server is the cloud server hostname (optional).
	// Typically not needed unless using a custom cloud server.
	Server *string `json:"server,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// CloudStatus represents the current status of the Cloud component.
type CloudStatus struct {
	types.RawFields
	Connected bool `json:"connected"`
}

// GetConfig retrieves the Cloud configuration.
//
// Example:
//
//	config, err := cloud.GetConfig(ctx)
//	if err == nil && config.Enable != nil && *config.Enable {
//	    fmt.Println("Cloud connection is enabled")
//	}
func (c *Cloud) GetConfig(ctx context.Context) (*CloudConfig, error) {
	resultJSON, err := c.client.Call(ctx, "Cloud.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config CloudConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the Cloud configuration.
//
// Only non-nil fields will be updated.
//
// Example - Enable cloud connection:
//
//	err := cloud.SetConfig(ctx, &CloudConfig{
//	    Enable: ptr(true),
//	})
//
// Example - Disable cloud connection:
//
//	err := cloud.SetConfig(ctx, &CloudConfig{
//	    Enable: ptr(false),
//	})
func (c *Cloud) SetConfig(ctx context.Context, config *CloudConfig) error {
	params := map[string]any{
		"config": config,
	}

	_, err := c.client.Call(ctx, "Cloud.SetConfig", params)
	return err
}

// GetStatus retrieves the current Cloud status.
//
// Returns whether the device is currently connected to the Shelly Cloud.
//
// Example:
//
//	status, err := cloud.GetStatus(ctx)
//	if err == nil {
//	    if status.Connected {
//	        fmt.Println("Device is connected to Shelly Cloud")
//	    } else {
//	        fmt.Println("Device is not connected to cloud")
//	    }
//	}
func (c *Cloud) GetStatus(ctx context.Context) (*CloudStatus, error) {
	resultJSON, err := c.client.Call(ctx, "Cloud.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status CloudStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (c *Cloud) Type() string {
	return "cloud"
}

// Key returns the component key for aggregated status/config responses.
func (c *Cloud) Key() string {
	return "cloud"
}

// Ensure Cloud implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Cloud)(nil)
