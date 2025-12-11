//nolint:dupl // Sensor components share similar SetConfig patterns but have distinct types
package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// humidityComponentType is the type identifier for the Humidity component.
const humidityComponentType = "humidity"

// Humidity represents a Shelly Gen2+ Humidity sensor component.
//
// Humidity provides relative humidity readings from connected sensors
// (typically DHT22, HTU21D, or built-in sensors on devices with addon support).
//
// Note: Humidity component uses numeric IDs (humidity:0, humidity:1, etc.).
//
// Example:
//
//	humidity := components.NewHumidity(device.Client(), 0)
//	status, err := humidity.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Relative Humidity: %.1f%%\n", *status.RH)
//	}
type Humidity struct {
	client *rpc.Client
	id     int
}

// NewHumidity creates a new Humidity component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (0-based)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	humidity := components.NewHumidity(device.Client(), 0)
func NewHumidity(client *rpc.Client, id int) *Humidity {
	return &Humidity{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (h *Humidity) Client() *rpc.Client {
	return h.client
}

// ID returns the component ID.
func (h *Humidity) ID() int {
	return h.id
}

// HumidityConfig represents the configuration of a Humidity component.
type HumidityConfig struct {
	Name      *string  `json:"name,omitempty"`
	ReportThr *float64 `json:"report_thr,omitempty"`
	Offset    *float64 `json:"offset,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// HumidityStatus represents the status of a Humidity component.
type HumidityStatus struct {
	RH *float64 `json:"rh"`
	types.RawFields
	Errors []string `json:"errors,omitempty"`
	ID     int      `json:"id"`
}

// GetConfig retrieves the Humidity configuration.
//
// Example:
//
//	config, err := humidity.GetConfig(ctx)
//	if err == nil && config.Name != nil {
//	    fmt.Printf("Sensor name: %s\n", *config.Name)
//	}
func (h *Humidity) GetConfig(ctx context.Context) (*HumidityConfig, error) {
	params := map[string]any{
		"id": h.id,
	}

	resultJSON, err := h.client.Call(ctx, "Humidity.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config HumidityConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the Humidity configuration.
//
// Only non-nil fields will be updated.
//
// Example - Set sensor name:
//
//	err := humidity.SetConfig(ctx, &HumidityConfig{
//	    Name: ptr("Living Room"),
//	})
//
// Example - Set reporting threshold:
//
//	err := humidity.SetConfig(ctx, &HumidityConfig{
//	    ReportThr: ptrFloat(5.0), // Report on 5% change
//	})
//
// Example - Set calibration offset:
//
//	err := humidity.SetConfig(ctx, &HumidityConfig{
//	    Offset: ptrFloat(-3.0), // Correct sensor reading by -3%
//	})
func (h *Humidity) SetConfig(ctx context.Context, config *HumidityConfig) error {
	// Build params, including the ID
	configMap := map[string]any{
		"id": h.id,
	}
	params := map[string]any{
		"id":     h.id,
		"config": configMap,
	}

	if config.Name != nil {
		configMap["name"] = *config.Name
	}
	if config.ReportThr != nil {
		configMap["report_thr"] = *config.ReportThr
	}
	if config.Offset != nil {
		configMap["offset"] = *config.Offset
	}

	_, err := h.client.Call(ctx, "Humidity.SetConfig", params)
	return err
}

// GetStatus retrieves the current Humidity status.
//
// Returns the current relative humidity reading.
//
// Example:
//
//	status, err := humidity.GetStatus(ctx)
//	if err == nil && status.RH != nil {
//	    fmt.Printf("Current humidity: %.1f%%\n", *status.RH)
//	}
func (h *Humidity) GetStatus(ctx context.Context) (*HumidityStatus, error) {
	params := map[string]any{
		"id": h.id,
	}

	resultJSON, err := h.client.Call(ctx, "Humidity.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status HumidityStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (h *Humidity) Type() string {
	return humidityComponentType
}

// Key returns the component key for aggregated status/config responses.
func (h *Humidity) Key() string {
	return humidityComponentType
}

// Ensure Humidity implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Humidity)(nil)
