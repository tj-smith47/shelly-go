package matter

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
)

// Matter provides access to the Matter component on Gen4+ devices.
//
// Matter is a unified smart home connectivity standard. This component
// allows configuration and management of Matter functionality on
// supported Shelly devices.
type Matter struct {
	client *rpc.Client
}

// NewMatter creates a new Matter component instance.
func NewMatter(client *rpc.Client) *Matter {
	return &Matter{client: client}
}

// GetConfig retrieves the current Matter configuration.
//
// Returns the configuration including whether Matter is enabled.
func (m *Matter) GetConfig(ctx context.Context) (*Config, error) {
	result, err := m.client.Call(ctx, "Matter.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SetConfig updates the Matter configuration.
//
// Use this to enable or disable Matter on the device.
func (m *Matter) SetConfig(ctx context.Context, params *SetConfigParams) error {
	_, err := m.client.Call(ctx, "Matter.SetConfig", map[string]any{
		"config": params,
	})
	return err
}

// GetStatus retrieves the current Matter status.
//
// Returns status including whether the device is commissionable
// and the number of paired fabrics.
func (m *Matter) GetStatus(ctx context.Context) (*Status, error) {
	result, err := m.client.Call(ctx, "Matter.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status Status
	if err := json.Unmarshal(result, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// FactoryReset resets all Matter settings on the device.
//
// This unpairs the device from all fabrics and erases all Matter data.
// Unlike Shelly.FactoryReset, this only affects Matter settings,
// leaving WiFi, device settings, and other configurations intact.
func (m *Matter) FactoryReset(ctx context.Context) error {
	_, err := m.client.Call(ctx, "Matter.FactoryReset", nil)
	return err
}

// Enable enables Matter on the device.
//
// This is a convenience method that calls SetConfig with Enable=true.
func (m *Matter) Enable(ctx context.Context) error {
	enable := true
	return m.SetConfig(ctx, &SetConfigParams{Enable: &enable})
}

// Disable disables Matter on the device.
//
// This is a convenience method that calls SetConfig with Enable=false.
func (m *Matter) Disable(ctx context.Context) error {
	enable := false
	return m.SetConfig(ctx, &SetConfigParams{Enable: &enable})
}

// IsEnabled returns whether Matter is currently enabled.
//
// This is a convenience method that gets config and checks the Enable field.
func (m *Matter) IsEnabled(ctx context.Context) (bool, error) {
	config, err := m.GetConfig(ctx)
	if err != nil {
		return false, err
	}
	return config.Enable, nil
}

// IsCommissionable returns whether the device is ready to be commissioned.
//
// A device is commissionable when Matter is enabled and it's not
// at its maximum fabric count.
func (m *Matter) IsCommissionable(ctx context.Context) (bool, error) {
	status, err := m.GetStatus(ctx)
	if err != nil {
		return false, err
	}
	return status.Commissionable, nil
}

// GetFabricsCount returns the number of fabrics the device is paired with.
func (m *Matter) GetFabricsCount(ctx context.Context) (int, error) {
	status, err := m.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.FabricsCount, nil
}
