package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// EM1 represents a Shelly Gen2+ EM1 (Single-phase Energy Monitor) component.
//
// EM1 components handle single-phase electrical energy monitoring. They measure:
//   - Voltage (V)
//   - Current (A)
//   - Active Power (W)
//   - Apparent Power (VA)
//   - Power Factor
//   - Network Frequency (Hz)
//   - Energy consumption and return (for bidirectional metering)
//
// This component is typically found on devices like:
//   - Shelly Pro EM (single-phase or dual-phase monitoring)
//   - Shelly Pro EM-50 (professional energy monitor)
//
// Example:
//
//	em1 := components.NewEM1(device.Client(), 0)
//	status, err := em1.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Voltage: %.2fV, Current: %.2fA, Power: %.2fW\n",
//	        status.Voltage, status.Current, status.ActPower)
//	}
type EM1 struct {
	*gen2.BaseComponent
}

// NewEM1 creates a new EM1 component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (0 for first phase, 1 for second phase, etc.)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	em1 := components.NewEM1(device.Client(), 0)
func NewEM1(client *rpc.Client, id int) *EM1 {
	return &EM1{
		BaseComponent: gen2.NewBaseComponent(client, "em1", id),
	}
}

// EM1Config represents the configuration of an EM1 component.
type EM1Config struct {
	Name    *string `json:"name,omitempty"`
	CTType  *string `json:"ct_type,omitempty"`
	Reverse *bool   `json:"reverse,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// EM1Status represents the current status of an EM1 component.
type EM1Status struct {
	PF   *float64 `json:"pf,omitempty"`
	Freq *float64 `json:"freq,omitempty"`
	types.RawFields
	Errors    []string `json:"errors,omitempty"`
	ID        int      `json:"id"`
	Voltage   float64  `json:"voltage"`
	Current   float64  `json:"current"`
	ActPower  float64  `json:"act_power"`
	AprtPower float64  `json:"aprt_power"`
}

// EM1GetCTTypesResult contains the list of supported CT types.
type EM1GetCTTypesResult struct {
	types.RawFields
	Types []string `json:"types"`
}

// GetConfig retrieves the EM1 configuration.
//
// Example:
//
//	config, err := em1.GetConfig(ctx)
//	if err != nil {
//	    return err
//	}
//	if config.CTType != nil {
//	    fmt.Printf("CT Type: %s\n", *config.CTType)
//	}
func (e *EM1) GetConfig(ctx context.Context) (*EM1Config, error) {
	return gen2.UnmarshalConfig[EM1Config](ctx, e.BaseComponent)
}

// SetConfig updates the EM1 configuration.
//
// Note: Changes to ct_type require a device restart.
//
// Example:
//
//	// Configure for 120A CT with bidirectional metering
//	err := em1.SetConfig(ctx, &EM1Config{
//	    Name:    ptr("Solar Phase A"),
//	    CTType:  ptr("120A"),
//	    Reverse: ptr(true),
//	})
func (e *EM1) SetConfig(ctx context.Context, config *EM1Config) error {
	return gen2.SetConfigWithID(ctx, e.BaseComponent, config)
}

// GetStatus retrieves the current EM1 status.
//
// Returns voltage, current, power, power factor, and frequency measurements.
//
// Example:
//
//	status, err := em1.GetStatus(ctx)
//	if err != nil {
//	    return err
//	}
//
//	fmt.Printf("Voltage: %.2fV\n", status.Voltage)
//	fmt.Printf("Current: %.2fA\n", status.Current)
//	fmt.Printf("Active Power: %.2fW\n", status.ActPower)
//	fmt.Printf("Apparent Power: %.2fVA\n", status.AprtPower)
//
//	if status.PF != nil {
//	    fmt.Printf("Power Factor: %.3f\n", *status.PF)
//	}
//
//	if status.Freq != nil {
//	    fmt.Printf("Frequency: %.2fHz\n", *status.Freq)
//	}
//
//	if len(status.Errors) > 0 {
//	    fmt.Printf("Errors: %v\n", status.Errors)
//	}
func (e *EM1) GetStatus(ctx context.Context) (*EM1Status, error) {
	return gen2.UnmarshalStatus[EM1Status](ctx, e.BaseComponent)
}

// GetCTTypes retrieves the list of supported CT (Current Transformer) types.
//
// This method returns the available CT types that can be configured for the device.
//
// Example:
//
//	result, err := em1.GetCTTypes(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Supported CT Types: %v\n", result.Types)
func (e *EM1) GetCTTypes(ctx context.Context) (*EM1GetCTTypesResult, error) {
	params := map[string]any{
		"id": e.ID(),
	}

	var result EM1GetCTTypesResult
	resultJSON, err := e.BaseComponent.Client().Call(ctx, "EM1.GetCTTypes", params)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
