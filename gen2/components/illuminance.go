//nolint:dupl // Sensor components share similar SetConfig patterns but have distinct types
package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// illuminanceComponentType is the type identifier for the Illuminance component.
const illuminanceComponentType = "illuminance"

// Illuminance represents a Shelly Gen2+ Illuminance sensor component.
//
// The Illuminance component handles monitoring of light level sensors.
// It provides lux measurements and categorizes light levels as dark,
// twilight, or bright based on configurable thresholds.
//
// Note: Illuminance component uses numeric IDs (illuminance:0, illuminance:1, etc.).
//
// Webhook Events:
//   - illuminance.change - produced when illumination (dark/twilight/bright)
//     has changed between two measurements
//
// Example:
//
//	illum := components.NewIlluminance(device.Client(), 0)
//	status, err := illum.GetStatus(ctx)
//	if err == nil && status.Lux != nil {
//	    fmt.Printf("Light level: %d lux (%s)\n", *status.Lux, *status.Illumination)
//	}
type Illuminance struct {
	client *rpc.Client
	id     int
}

// NewIlluminance creates a new Illuminance component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (0-based)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	illum := components.NewIlluminance(device.Client(), 0)
func NewIlluminance(client *rpc.Client, id int) *Illuminance {
	return &Illuminance{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (i *Illuminance) Client() *rpc.Client {
	return i.client
}

// ID returns the component ID.
func (i *Illuminance) ID() int {
	return i.id
}

// IlluminationLevel represents the interpreted light level.
type IlluminationLevel string

const (
	// IlluminationDark indicates lux is below dark_thr.
	IlluminationDark IlluminationLevel = "dark"
	// IlluminationTwilight indicates lux is between dark_thr and bright_thr.
	IlluminationTwilight IlluminationLevel = "twilight"
	// IlluminationBright indicates lux is above bright_thr.
	IlluminationBright IlluminationLevel = "bright"
)

// IlluminanceConfig represents the configuration of an Illuminance component.
type IlluminanceConfig struct {
	Name      *string `json:"name,omitempty"`
	DarkThr   *int    `json:"dark_thr,omitempty"`
	BrightThr *int    `json:"bright_thr,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// IlluminanceStatus represents the status of an Illuminance component.
type IlluminanceStatus struct {
	Lux          *int               `json:"lux"`
	Illumination *IlluminationLevel `json:"illumination"`
	types.RawFields
	Errors []string `json:"errors,omitempty"`
	ID     int      `json:"id"`
}

// GetConfig retrieves the Illuminance configuration.
//
// Example:
//
//	config, err := illum.GetConfig(ctx)
//	if err == nil {
//	    fmt.Printf("Dark threshold: %d lux\n", *config.DarkThr)
//	    fmt.Printf("Bright threshold: %d lux\n", *config.BrightThr)
//	}
func (i *Illuminance) GetConfig(ctx context.Context) (*IlluminanceConfig, error) {
	params := map[string]any{
		"id": i.id,
	}

	resultJSON, err := i.client.Call(ctx, "Illuminance.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config IlluminanceConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the Illuminance configuration.
//
// Only non-nil fields will be updated.
//
// Example - Set sensor name:
//
//	name := "Motion Sensor Light"
//	err := illum.SetConfig(ctx, &IlluminanceConfig{
//	    Name: &name,
//	})
//
// Example - Set thresholds:
//
//	darkThr := 50    // Below 50 lux is "dark"
//	brightThr := 500 // Above 500 lux is "bright"
//	err := illum.SetConfig(ctx, &IlluminanceConfig{
//	    DarkThr:   &darkThr,
//	    BrightThr: &brightThr,
//	})
func (i *Illuminance) SetConfig(ctx context.Context, config *IlluminanceConfig) error {
	configMap := map[string]any{
		"id": i.id,
	}
	params := map[string]any{
		"id":     i.id,
		"config": configMap,
	}

	if config.Name != nil {
		configMap["name"] = *config.Name
	}
	if config.DarkThr != nil {
		configMap["dark_thr"] = *config.DarkThr
	}
	if config.BrightThr != nil {
		configMap["bright_thr"] = *config.BrightThr
	}

	_, err := i.client.Call(ctx, "Illuminance.SetConfig", params)
	return err
}

// GetStatus retrieves the current Illuminance status.
//
// Returns the current light level in lux and interpreted illumination level.
//
// Example:
//
//	status, err := illum.GetStatus(ctx)
//	if err == nil {
//	    if status.Lux != nil {
//	        fmt.Printf("Light level: %d lux\n", *status.Lux)
//	    }
//	    if status.Illumination != nil {
//	        switch *status.Illumination {
//	        case IlluminationDark:
//	            fmt.Println("It's dark")
//	        case IlluminationTwilight:
//	            fmt.Println("It's twilight")
//	        case IlluminationBright:
//	            fmt.Println("It's bright")
//	        }
//	    }
//	    if len(status.Errors) > 0 {
//	        fmt.Printf("Errors: %v\n", status.Errors)
//	    }
//	}
func (i *Illuminance) GetStatus(ctx context.Context) (*IlluminanceStatus, error) {
	params := map[string]any{
		"id": i.id,
	}

	resultJSON, err := i.client.Call(ctx, "Illuminance.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status IlluminanceStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (i *Illuminance) Type() string {
	return illuminanceComponentType
}

// Key returns the component key for aggregated status/config responses.
func (i *Illuminance) Key() string {
	return illuminanceComponentType
}

// Ensure Illuminance implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Illuminance)(nil)
