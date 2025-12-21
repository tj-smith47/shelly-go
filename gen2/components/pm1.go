//nolint:dupl // PM and PM1 are distinct RPC namespaces with identical structure - intentional
package components

import (
	"context"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// PM1 represents a Shelly Gen2+ PM1 (Power Meter) component.
//
// PM1 components handle electrical power metering capabilities. They measure:
//   - Voltage (V)
//   - Current (A)
//   - Active Power (W)
//   - Network Frequency (Hz)
//   - Active Energy consumption and return (Wh)
//
// This component is typically found on devices like:
//   - Shelly Plus 1PM (relay with power metering)
//   - Shelly Plus PM Mini (standalone power meter)
//   - Shelly Pro 1PM, Pro 2PM, Pro 4PM (professional devices)
//   - Shelly Plus RGBW PM (RGBW controller with metering)
//
// Example:
//
//	pm1 := components.NewPM1(device.Client(), 0)
//	status, err := pm1.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Power: %.2fW, Voltage: %.2fV, Current: %.2fA\n",
//	        status.APower, status.Voltage, status.Current)
//	    fmt.Printf("Total energy: %.2fWh\n", status.AEnergy.Total)
//	}
type PM1 struct {
	*gen2.BaseComponent
}

// NewPM1 creates a new PM1 component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (usually 0 for single PM devices)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	pm1 := components.NewPM1(device.Client(), 0)
func NewPM1(client *rpc.Client, id int) *PM1 {
	return &PM1{
		BaseComponent: gen2.NewBaseComponent(client, "pm1", id),
	}
}

// PM1Config represents the configuration of a PM1 component.
type PM1Config struct {
	Name    *string `json:"name,omitempty"`
	Reverse *bool   `json:"reverse,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// PM1Status represents the current status of a PM1 component.
type PM1Status struct {
	Freq       *float64           `json:"freq,omitempty"`
	AEnergy    *PM1EnergyCounters `json:"aenergy,omitempty"`
	RetAEnergy *PM1EnergyCounters `json:"ret_aenergy,omitempty"`
	types.RawFields
	Errors  []string `json:"errors,omitempty"`
	ID      int      `json:"id"`
	Voltage float64  `json:"voltage"`
	Current float64  `json:"current"`
	APower  float64  `json:"apower"`
}

// PM1EnergyCounters represents accumulated energy measurements for PM1.
type PM1EnergyCounters struct {
	MinuteTs *int64 `json:"minute_ts,omitempty"`
	types.RawFields
	ByMinute []float64 `json:"by_minute,omitempty"`
	Total    float64   `json:"total"`
}

// PM1ResetCountersParams contains parameters for the PM1.ResetCounters method.
type PM1ResetCountersParams struct {
	types.RawFields `json:"-"`
	Type            []string `json:"type,omitempty"`
	ID              int      `json:"id"`
}

// GetConfig retrieves the PM1 configuration.
//
// Example:
//
//	config, err := pm1.GetConfig(ctx)
//	if err != nil {
//	    return err
//	}
//	if config.Reverse != nil && *config.Reverse {
//	    fmt.Println("Bidirectional metering enabled")
//	}
func (p *PM1) GetConfig(ctx context.Context) (*PM1Config, error) {
	return gen2.UnmarshalConfig[PM1Config](ctx, p.BaseComponent)
}

// SetConfig updates the PM1 configuration.
//
// Example:
//
//	// Enable bidirectional metering for solar installation
//	err := pm1.SetConfig(ctx, &PM1Config{
//	    Name:    ptr("Solar Meter"),
//	    Reverse: ptr(true),
//	})
func (p *PM1) SetConfig(ctx context.Context, config *PM1Config) error {
	return gen2.SetConfigWithID(ctx, p.BaseComponent, config)
}

// GetStatus retrieves the current PM1 status.
//
// Returns voltage, current, power, frequency, and energy measurements.
//
// Example:
//
//	status, err := pm1.GetStatus(ctx)
//	if err != nil {
//	    return err
//	}
//
//	fmt.Printf("Voltage: %.2fV\n", status.Voltage)
//	fmt.Printf("Current: %.2fA\n", status.Current)
//	fmt.Printf("Power: %.2fW\n", status.APower)
//
//	if status.Freq != nil {
//	    fmt.Printf("Frequency: %.2fHz\n", *status.Freq)
//	}
//
//	if status.AEnergy != nil {
//	    fmt.Printf("Total energy consumed: %.2fWh\n", status.AEnergy.Total)
//	}
//
//	if status.RetAEnergy != nil {
//	    fmt.Printf("Total energy returned: %.2fWh\n", status.RetAEnergy.Total)
//	}
//
//	if len(status.Errors) > 0 {
//	    fmt.Printf("Errors: %v\n", status.Errors)
//	}
func (p *PM1) GetStatus(ctx context.Context) (*PM1Status, error) {
	return gen2.UnmarshalStatus[PM1Status](ctx, p.BaseComponent)
}

// ResetCounters resets the energy counters.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - counterTypes: Optional list of counter types to reset (e.g., ["aenergy", "ret_aenergy"])
//     If empty or nil, all counters are reset
//
// Example:
//
//	// Reset all counters
//	err := pm1.ResetCounters(ctx, nil)
//
//	// Reset only active energy counter
//	err := pm1.ResetCounters(ctx, []string{"aenergy"})
//
//	// Reset both active and returned energy counters
//	err := pm1.ResetCounters(ctx, []string{"aenergy", "ret_aenergy"})
func (p *PM1) ResetCounters(ctx context.Context, counterTypes []string) error {
	params := &PM1ResetCountersParams{
		ID: p.ID(),
	}

	if len(counterTypes) > 0 {
		params.Type = counterTypes
	}

	_, err := p.BaseComponent.Client().Call(ctx, "PM1.ResetCounters", params)
	return err
}
