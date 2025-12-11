//nolint:dupl // PM and PM1 are distinct RPC namespaces with identical structure - intentional
package components

import (
	"context"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// PM represents a Shelly Gen2+ PM (Power Meter) component.
//
// PM is a power metering component with the same functionality as PM1.
// It measures voltage, current, power, frequency, and energy consumption.
//
// Note: This component uses "PM" as the RPC namespace instead of "PM1".
// For most devices, PM1 is the preferred component. PM may be present on
// certain legacy or specific device models.
//
// Example:
//
//	pm := components.NewPM(device.Client(), 0)
//	status, err := pm.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Power: %.2fW\n", status.APower)
//	}
type PM struct {
	*gen2.BaseComponent
}

// NewPM creates a new PM component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	pm := components.NewPM(device.Client(), 0)
func NewPM(client *rpc.Client, id int) *PM {
	return &PM{
		BaseComponent: gen2.NewBaseComponent(client, "pm", id),
	}
}

// PMConfig represents the configuration of a PM component.
//
// PM configuration is identical to PM1 configuration.
type PMConfig struct {
	Name    *string `json:"name,omitempty"`
	Reverse *bool   `json:"reverse,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// PMStatus represents the current status of a PM component.
//
// PM status is identical to PM1 status.
type PMStatus struct {
	Freq       *float64          `json:"freq,omitempty"`
	AEnergy    *PMEnergyCounters `json:"aenergy,omitempty"`
	RetAEnergy *PMEnergyCounters `json:"ret_aenergy,omitempty"`
	types.RawFields
	Errors  []string `json:"errors,omitempty"`
	ID      int      `json:"id"`
	Voltage float64  `json:"voltage"`
	Current float64  `json:"current"`
	APower  float64  `json:"apower"`
}

// PMEnergyCounters represents accumulated energy measurements for PM.
type PMEnergyCounters struct {
	MinuteTs *int64 `json:"minute_ts,omitempty"`
	types.RawFields
	ByMinute []float64 `json:"by_minute,omitempty"`
	Total    float64   `json:"total"`
}

// PMResetCountersParams contains parameters for the PM.ResetCounters method.
type PMResetCountersParams struct {
	types.RawFields
	Type []string `json:"type,omitempty"`
	ID   int      `json:"id"`
}

// GetConfig retrieves the PM configuration.
//
// Example:
//
//	config, err := pm.GetConfig(ctx)
func (p *PM) GetConfig(ctx context.Context) (*PMConfig, error) {
	return gen2.UnmarshalConfig[PMConfig](ctx, p.BaseComponent)
}

// SetConfig updates the PM configuration.
//
// Example:
//
//	err := pm.SetConfig(ctx, &PMConfig{
//	    Name:    ptr("Main Meter"),
//	    Reverse: ptr(false),
//	})
func (p *PM) SetConfig(ctx context.Context, config *PMConfig) error {
	return gen2.SetConfigWithID(ctx, p.BaseComponent, config)
}

// GetStatus retrieves the current PM status.
//
// Example:
//
//	status, err := pm.GetStatus(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Power: %.2fW, Energy: %.2fWh\n", status.APower, status.AEnergy.Total)
func (p *PM) GetStatus(ctx context.Context) (*PMStatus, error) {
	return gen2.UnmarshalStatus[PMStatus](ctx, p.BaseComponent)
}

// ResetCounters resets the energy counters.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - counterTypes: Optional list of counter types to reset
//     If empty or nil, all counters are reset
//
// Example:
//
//	// Reset all counters
//	err := pm.ResetCounters(ctx, nil)
//
//	// Reset only active energy counter
//	err := pm.ResetCounters(ctx, []string{"aenergy"})
func (p *PM) ResetCounters(ctx context.Context, counterTypes []string) error {
	params := &PMResetCountersParams{
		ID: p.ID(),
	}

	if len(counterTypes) > 0 {
		params.Type = counterTypes
	}

	_, err := p.BaseComponent.Client().Call(ctx, "PM.ResetCounters", params)
	return err
}
