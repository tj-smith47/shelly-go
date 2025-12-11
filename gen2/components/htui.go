package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

const htUIComponentType = "ht_ui"

// HTUI represents a Shelly Gen2+ HT_UI component for H&T device display settings.
//
// The HT_UI component handles the settings of a Plus H&T device's screen,
// primarily the temperature unit display setting (Celsius or Fahrenheit).
//
// Note: This component is specific to Shelly Plus H&T devices and may not
// be present on other device types.
//
// Example:
//
//	htui := components.NewHTUI(device.Client())
//	config, err := htui.GetConfig(ctx)
//	if err == nil {
//	    fmt.Printf("Temperature unit: %s\n", config.TemperatureUnit)
//	}
type HTUI struct {
	client *rpc.Client
}

// NewHTUI creates a new HT_UI component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	htui := components.NewHTUI(device.Client())
func NewHTUI(client *rpc.Client) *HTUI {
	return &HTUI{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (h *HTUI) Client() *rpc.Client {
	return h.client
}

// HTUIConfig represents the configuration of a HT_UI component.
type HTUIConfig struct {
	types.RawFields
	TemperatureUnit string `json:"temperature_unit"`
}

// HTUIStatus represents the status of a HT_UI component.
// Note: The HT_UI component does not own any status properties.
type HTUIStatus struct {
	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// GetConfig retrieves the HT_UI configuration.
//
// Example:
//
//	config, err := htui.GetConfig(ctx)
//	if err == nil {
//	    fmt.Printf("Using temperature unit: %s\n", config.TemperatureUnit)
//	}
func (h *HTUI) GetConfig(ctx context.Context) (*HTUIConfig, error) {
	resultJSON, err := h.client.Call(ctx, "HT_UI.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config HTUIConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the HT_UI configuration.
//
// Example - Set temperature unit to Fahrenheit:
//
//	err := htui.SetConfig(ctx, &HTUIConfig{
//	    TemperatureUnit: "F",
//	})
//
// Example - Set temperature unit to Celsius:
//
//	err := htui.SetConfig(ctx, &HTUIConfig{
//	    TemperatureUnit: "C",
//	})
func (h *HTUI) SetConfig(ctx context.Context, config *HTUIConfig) error {
	params := map[string]any{
		"config": map[string]any{
			"temperature_unit": config.TemperatureUnit,
		},
	}

	_, err := h.client.Call(ctx, "HT_UI.SetConfig", params)
	return err
}

// Type returns the component type identifier.
func (h *HTUI) Type() string {
	return htUIComponentType
}

// Key returns the component key for aggregated status/config responses.
func (h *HTUI) Key() string {
	return htUIComponentType
}

// Ensure HTUI implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*HTUI)(nil)
