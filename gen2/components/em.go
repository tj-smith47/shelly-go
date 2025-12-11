package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// EM represents a Shelly Gen2+ EM (Energy Monitor) component.
//
// EM components handle 3-phase electrical energy monitoring. They measure:
//   - Per-phase voltage, current, power (active/apparent), power factor, frequency
//   - Neutral current
//   - Total energy consumption and return (for bidirectional metering)
//   - Phase sequence monitoring
//
// This component is typically found on devices like:
//   - Shelly Pro 3EM (3-phase energy monitor)
//   - Shelly Pro EM-50 (professional energy monitor)
//
// Example:
//
//	em := components.NewEM(device.Client(), 0)
//	status, err := em.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Phase A: %.2fV, %.2fA, %.2fW\n",
//	        status.AVoltage, status.ACurrent, status.AActivePower)
//	    fmt.Printf("Total power: %.2fW\n", status.TotalActivePower)
//	}
type EM struct {
	*gen2.BaseComponent
}

// NewEM creates a new EM component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (usually 0)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	em := components.NewEM(device.Client(), 0)
func NewEM(client *rpc.Client, id int) *EM {
	return &EM{
		BaseComponent: gen2.NewBaseComponent(client, "em", id),
	}
}

// EMConfig represents the configuration of an EM component.
type EMConfig struct {
	Name                 *string `json:"name,omitempty"`
	BlinkModeSelector    *string `json:"blink_mode_selector,omitempty"`
	PhaseSelector        *string `json:"phase_selector,omitempty"`
	MonitorPhaseSequence *bool   `json:"monitor_phase_sequence,omitempty"`
	CTType               *string `json:"ct_type,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// EMStatus represents the current status of an EM component.
type EMStatus struct {
	BFreq        *float64 `json:"b_freq,omitempty"`
	BPowerFactor *float64 `json:"b_pf,omitempty"`
	types.RawFields
	UserCalibratedPhase *string  `json:"user_calibrated_phase,omitempty"`
	APowerFactor        *float64 `json:"a_pf,omitempty"`
	AFreq               *float64 `json:"a_freq,omitempty"`
	NCurrent            *float64 `json:"n_current,omitempty"`
	CPowerFactor        *float64 `json:"c_pf,omitempty"`
	CFreq               *float64 `json:"c_freq,omitempty"`
	Errors              []string `json:"errors,omitempty"`
	BVoltage            float64  `json:"b_voltage"`
	BCurrent            float64  `json:"b_current"`
	AVoltage            float64  `json:"a_voltage"`
	TotalApparentPower  float64  `json:"total_aprt_power"`
	BActivePower        float64  `json:"b_act_power"`
	CActivePower        float64  `json:"c_act_power"`
	CApparentPower      float64  `json:"c_aprt_power"`
	BApparentPower      float64  `json:"b_aprt_power"`
	CCurrent            float64  `json:"c_current"`
	ID                  int      `json:"id"`
	TotalCurrent        float64  `json:"total_current"`
	TotalActivePower    float64  `json:"total_act_power"`
	CVoltage            float64  `json:"c_voltage"`
	AApparentPower      float64  `json:"a_aprt_power"`
	AActivePower        float64  `json:"a_act_power"`
	ACurrent            float64  `json:"a_current"`
}

// EMGetCTTypesResult contains the list of supported CT types.
type EMGetCTTypesResult struct {
	types.RawFields
	Types []string `json:"types"`
}

// EMResetCountersParams contains parameters for the EM.ResetCounters method.
type EMResetCountersParams struct {
	types.RawFields
	Type []string `json:"type,omitempty"`
	ID   int      `json:"id"`
}

// GetConfig retrieves the EM configuration.
//
// Example:
//
//	config, err := em.GetConfig(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("CT Type: %s\n", *config.CTType)
//	fmt.Printf("Phase Selector: %s\n", *config.PhaseSelector)
func (e *EM) GetConfig(ctx context.Context) (*EMConfig, error) {
	return gen2.UnmarshalConfig[EMConfig](ctx, e.BaseComponent)
}

// SetConfig updates the EM configuration.
//
// Note: Changes to blink_mode_selector, phase_selector, and ct_type require a device restart.
//
// Example:
//
//	// Configure for 120A CT and monitor all phases
//	err := em.SetConfig(ctx, &EMConfig{
//	    Name:                  ptr("Main 3-Phase Meter"),
//	    CTType:                ptr("120A"),
//	    PhaseSelector:         ptr("all"),
//	    MonitorPhaseSequence:  ptr(true),
//	})
func (e *EM) SetConfig(ctx context.Context, config *EMConfig) error {
	return gen2.SetConfigWithID(ctx, e.BaseComponent, config)
}

// GetStatus retrieves the current EM status.
//
// Returns measurements for all phases (A, B, C), neutral current, and totals.
//
// Example:
//
//	status, err := em.GetStatus(ctx)
//	if err != nil {
//	    return err
//	}
//
//	fmt.Printf("Phase A: %.2fV, %.2fA, %.2fW\n",
//	    status.AVoltage, status.ACurrent, status.AActivePower)
//	fmt.Printf("Phase B: %.2fV, %.2fA, %.2fW\n",
//	    status.BVoltage, status.BCurrent, status.BActivePower)
//	fmt.Printf("Phase C: %.2fV, %.2fA, %.2fW\n",
//	    status.CVoltage, status.CCurrent, status.CActivePower)
//	fmt.Printf("Total Power: %.2fW\n", status.TotalActivePower)
//
//	if status.NCurrent != nil {
//	    fmt.Printf("Neutral Current: %.2fA\n", *status.NCurrent)
//	}
//
//	if len(status.Errors) > 0 {
//	    fmt.Printf("Errors: %v\n", status.Errors)
//	}
func (e *EM) GetStatus(ctx context.Context) (*EMStatus, error) {
	return gen2.UnmarshalStatus[EMStatus](ctx, e.BaseComponent)
}

// GetCTTypes retrieves the list of supported CT (Current Transformer) types.
//
// This method returns the available CT types that can be configured for the device.
//
// Example:
//
//	result, err := em.GetCTTypes(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Supported CT Types: %v\n", result.Types)
func (e *EM) GetCTTypes(ctx context.Context) (*EMGetCTTypesResult, error) {
	params := map[string]any{
		"id": e.ID(),
	}

	var result EMGetCTTypesResult
	resultJSON, err := e.BaseComponent.Client().Call(ctx, "EM.GetCTTypes", params)
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
//   - counterTypes: Optional list of counter types to reset
//     If empty or nil, all counters are reset
//
// Example:
//
//	// Reset all counters
//	err := em.ResetCounters(ctx, nil)
//
//	// Reset specific counters
//	err := em.ResetCounters(ctx, []string{"a_act_energy", "b_act_energy", "c_act_energy"})
func (e *EM) ResetCounters(ctx context.Context, counterTypes []string) error {
	params := &EMResetCountersParams{
		ID: e.ID(),
	}

	if len(counterTypes) > 0 {
		params.Type = counterTypes
	}

	_, err := e.BaseComponent.Client().Call(ctx, "EM.ResetCounters", params)
	return err
}
