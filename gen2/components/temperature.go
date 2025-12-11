//nolint:dupl // Sensor components share similar SetConfig patterns but have distinct types
package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// temperatureComponentType is the type identifier for the Temperature component.
const temperatureComponentType = "temperature"

// Temperature represents a Shelly Gen2+ Temperature sensor component.
//
// Temperature provides temperature readings from connected sensors
// (typically DS18B20 or built-in sensors on devices with addon support).
//
// Note: Temperature component uses numeric IDs (temperature:0, temperature:1, etc.).
//
// Example:
//
//	temp := components.NewTemperature(device.Client(), 0)
//	status, err := temp.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Temperature: %.1f°C / %.1f°F\n", *status.TC, *status.TF)
//	}
type Temperature struct {
	client *rpc.Client
	id     int
}

// NewTemperature creates a new Temperature component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (0-based)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	temp := components.NewTemperature(device.Client(), 0)
func NewTemperature(client *rpc.Client, id int) *Temperature {
	return &Temperature{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (t *Temperature) Client() *rpc.Client {
	return t.client
}

// ID returns the component ID.
func (t *Temperature) ID() int {
	return t.id
}

// TemperatureConfig represents the configuration of a Temperature component.
type TemperatureConfig struct {
	Name       *string  `json:"name,omitempty"`
	ReportThrC *float64 `json:"report_thr_C,omitempty"`
	OffsetC    *float64 `json:"offset_C,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// TemperatureStatus represents the status of a Temperature component.
type TemperatureStatus struct {
	TC *float64 `json:"tC"`
	TF *float64 `json:"tF"`
	types.RawFields
	Errors []string `json:"errors,omitempty"`
	ID     int      `json:"id"`
}

// GetConfig retrieves the Temperature configuration.
//
// Example:
//
//	config, err := temp.GetConfig(ctx)
//	if err == nil && config.Name != nil {
//	    fmt.Printf("Sensor name: %s\n", *config.Name)
//	}
func (t *Temperature) GetConfig(ctx context.Context) (*TemperatureConfig, error) {
	params := map[string]any{
		"id": t.id,
	}

	resultJSON, err := t.client.Call(ctx, "Temperature.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config TemperatureConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the Temperature configuration.
//
// Only non-nil fields will be updated.
//
// Example - Set sensor name:
//
//	err := temp.SetConfig(ctx, &TemperatureConfig{
//	    Name: ptr("Living Room"),
//	})
//
// Example - Set reporting threshold:
//
//	err := temp.SetConfig(ctx, &TemperatureConfig{
//	    ReportThrC: ptrFloat(0.5), // Report on 0.5°C change
//	})
//
// Example - Set calibration offset:
//
//	err := temp.SetConfig(ctx, &TemperatureConfig{
//	    OffsetC: ptrFloat(-0.3), // Correct sensor reading by -0.3°C
//	})
func (t *Temperature) SetConfig(ctx context.Context, config *TemperatureConfig) error {
	// Build params, including the ID
	configMap := map[string]any{
		"id": t.id,
	}
	params := map[string]any{
		"id":     t.id,
		"config": configMap,
	}

	if config.Name != nil {
		configMap["name"] = *config.Name
	}
	if config.ReportThrC != nil {
		configMap["report_thr_C"] = *config.ReportThrC
	}
	if config.OffsetC != nil {
		configMap["offset_C"] = *config.OffsetC
	}

	_, err := t.client.Call(ctx, "Temperature.SetConfig", params)
	return err
}

// GetStatus retrieves the current Temperature status.
//
// Returns the current temperature reading in Celsius and Fahrenheit.
//
// Example:
//
//	status, err := temp.GetStatus(ctx)
//	if err == nil && status.TC != nil {
//	    fmt.Printf("Current temperature: %.1f°C (%.1f°F)\n", *status.TC, *status.TF)
//	}
func (t *Temperature) GetStatus(ctx context.Context) (*TemperatureStatus, error) {
	params := map[string]any{
		"id": t.id,
	}

	resultJSON, err := t.client.Call(ctx, "Temperature.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status TemperatureStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (t *Temperature) Type() string {
	return temperatureComponentType
}

// Key returns the component key for aggregated status/config responses.
func (t *Temperature) Key() string {
	return temperatureComponentType
}

// Ensure Temperature implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Temperature)(nil)
