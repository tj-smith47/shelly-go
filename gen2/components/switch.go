package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Switch represents a Shelly Gen2+ Switch component.
//
// Switch components control relay outputs and can measure power consumption
// on devices with power metering capabilities.
//
// Example:
//
//	sw := components.NewSwitch(device.Client(), 0)
//	err := sw.Set(ctx, &SwitchSetParams{On: true})
type Switch struct {
	*gen2.BaseComponent
}

// NewSwitch creates a new Switch component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (usually 0 for single-switch devices)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	sw := components.NewSwitch(device.Client(), 0)
func NewSwitch(client *rpc.Client, id int) *Switch {
	return &Switch{
		BaseComponent: gen2.NewBaseComponent(client, "switch", id),
	}
}

// SwitchConfig represents the configuration of a Switch component.
type SwitchConfig struct {
	AutoOffDelay *float64 `json:"auto_off_delay,omitempty"`
	PowerLimit   *float64 `json:"power_limit,omitempty"`
	InitialState *string  `json:"initial_state,omitempty"`
	AutoOn       *bool    `json:"auto_on,omitempty"`
	AutoOnDelay  *float64 `json:"auto_on_delay,omitempty"`
	AutoOff      *bool    `json:"auto_off,omitempty"`
	Name         *string  `json:"name,omitempty"`
	InputID      *int     `json:"input_id,omitempty"`
	types.RawFields
	InputMode                *string  `json:"input_mode,omitempty"`
	AutorecoverVoltageErrors *bool    `json:"autorecover_voltage_errors,omitempty"`
	VoltageLimit             *float64 `json:"voltage_limit,omitempty"`
	UndervoltageLimit        *float64 `json:"undervoltage_limit,omitempty"`
	CurrentLimit             *float64 `json:"current_limit,omitempty"`
	ID                       int      `json:"id"`
}

// SwitchStatus represents the current status of a Switch component.
type SwitchStatus struct {
	Voltage *float64        `json:"voltage,omitempty"`
	AEnergy *EnergyCounters `json:"aenergy,omitempty"`
	types.RawFields
	TimerStartedAt *float64           `json:"timer_started_at,omitempty"`
	TimerDuration  *float64           `json:"timer_duration,omitempty"`
	APower         *float64           `json:"apower,omitempty"`
	PF             *float64           `json:"pf,omitempty"`
	Current        *float64           `json:"current,omitempty"`
	Temperature    *TemperatureSensor `json:"temperature,omitempty"`
	Freq           *float64           `json:"freq,omitempty"`
	Source         string             `json:"source"`
	Errors         []string           `json:"errors,omitempty"`
	ID             int                `json:"id"`
	Output         bool               `json:"output"`
}

// EnergyCounters represents accumulated energy measurements.
type EnergyCounters struct {
	MinuteTs *int64 `json:"minute_ts,omitempty"`
	types.RawFields
	ByMinute []float64 `json:"by_minute,omitempty"`
	Total    float64   `json:"total"`
}

// TemperatureSensor represents temperature sensor data.
type TemperatureSensor struct {
	// TC is the temperature in Celsius
	TC *float64 `json:"tC,omitempty"`

	// TF is the temperature in Fahrenheit
	TF *float64 `json:"tF,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// SwitchSetParams contains parameters for the Switch.Set method.
type SwitchSetParams struct {
	ToggleAfter *float64 `json:"toggle_after,omitempty"`
	types.RawFields
	ID int  `json:"id"`
	On bool `json:"on"`
}

// SwitchSetResult contains the result of a Switch.Set call.
type SwitchSetResult struct {
	types.RawFields
	WasOn bool `json:"was_on"`
}

// SwitchToggleParams contains parameters for the Switch.Toggle method.
type SwitchToggleParams struct {
	types.RawFields
	ID int `json:"id"`
}

// SwitchToggleResult contains the result of a Switch.Toggle call.
type SwitchToggleResult struct {
	types.RawFields
	WasOn bool `json:"was_on"`
}

// Set turns the switch on or off.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - params: Switch set parameters
//
// Returns the previous state of the switch.
//
// Example:
//
//	// Turn on
//	result, err := sw.Set(ctx, &SwitchSetParams{ID: 0, On: true})
//
//	// Turn on for 10 seconds, then toggle off
//	result, err := sw.Set(ctx, &SwitchSetParams{
//	    ID: 0,
//	    On: true,
//	    ToggleAfter: ptr(10.0),
//	})
func (s *Switch) Set(ctx context.Context, params *SwitchSetParams) (*SwitchSetResult, error) {
	params = gen2.EnsureID(s.BaseComponent, params)

	var result SwitchSetResult
	resultJSON, err := s.BaseComponent.Client().Call(ctx, "Switch.Set", params)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Toggle toggles the switch state (on -> off, off -> on).
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns the previous state of the switch.
//
// Example:
//
//	result, err := sw.Toggle(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Was on: %v\n", result.WasOn)
func (s *Switch) Toggle(ctx context.Context) (*SwitchToggleResult, error) {
	params := gen2.EnsureID(s.BaseComponent, &SwitchToggleParams{})

	var result SwitchToggleResult
	resultJSON, err := s.BaseComponent.Client().Call(ctx, "Switch.Toggle", params)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ResetCounters resets the energy counters.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - types: Optional list of counter types to reset (e.g., ["aenergy"])
//
// Example:
//
//	err := sw.ResetCounters(ctx, []string{"aenergy"})
func (s *Switch) ResetCounters(ctx context.Context, counterTypes []string) error {
	params := map[string]any{
		"id": s.ID(),
	}

	if len(counterTypes) > 0 {
		params["type"] = counterTypes
	}

	_, err := s.BaseComponent.Client().Call(ctx, "Switch.ResetCounters", params)
	return err
}

// GetConfig retrieves the switch configuration.
//
// Example:
//
//	config, err := sw.GetConfig(ctx)
func (s *Switch) GetConfig(ctx context.Context) (*SwitchConfig, error) {
	return gen2.UnmarshalConfig[SwitchConfig](ctx, s.BaseComponent)
}

// SetConfig updates the switch configuration.
//
// Example:
//
//	err := sw.SetConfig(ctx, &SwitchConfig{
//	    Name: ptr("Living Room Light"),
//	    AutoOff: ptr(true),
//	    AutoOffDelay: ptr(300.0), // 5 minutes
//	})
func (s *Switch) SetConfig(ctx context.Context, config *SwitchConfig) error {
	return gen2.SetConfigWithID(ctx, s.BaseComponent, config)
}

// GetStatus retrieves the current switch status.
//
// Example:
//
//	status, err := sw.GetStatus(ctx)
//	fmt.Printf("Output: %v, Power: %.2f W\n", status.Output, *status.APower)
func (s *Switch) GetStatus(ctx context.Context) (*SwitchStatus, error) {
	return gen2.UnmarshalStatus[SwitchStatus](ctx, s.BaseComponent)
}
