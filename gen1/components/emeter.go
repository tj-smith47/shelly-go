package components

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/tj-smith47/shelly-go/transport"
)

// EMeter provides access to Gen1 energy meter readings.
//
// Energy meters provide more detailed power monitoring including
// voltage, current, power factor, and bidirectional energy tracking.
// Available on devices like Shelly EM and Shelly 3EM.
type EMeter struct {
	transport transport.Transport
	id        int
}

// NewEMeter creates a new EMeter component accessor.
//
// Parameters:
//   - t: The transport to use for API calls
//   - id: The emeter index (0-based, 0-2 for 3EM)
func NewEMeter(t transport.Transport, id int) *EMeter {
	return &EMeter{
		transport: t,
		id:        id,
	}
}

// ID returns the emeter index.
func (e *EMeter) ID() int {
	return e.id
}

// EMeterStatus contains energy meter readings.
type EMeterStatus struct {
	// Power is current power in watts (negative = returned).
	Power float64 `json:"power"`

	// Reactive is reactive power in VAR.
	Reactive float64 `json:"reactive,omitempty"`

	// Apparent is apparent power in VA.
	Apparent float64 `json:"apparent,omitempty"`

	// PF is the power factor (-1 to 1).
	PF float64 `json:"pf,omitempty"`

	// Current is the current in amps.
	Current float64 `json:"current,omitempty"`

	// Voltage is the voltage in volts.
	Voltage float64 `json:"voltage,omitempty"`

	// IsValid indicates if the reading is valid.
	IsValid bool `json:"is_valid,omitempty"`

	// Total is total consumed energy in watt-hours.
	Total float64 `json:"total,omitempty"`

	// TotalReturned is total returned energy in watt-hours.
	TotalReturned float64 `json:"total_returned,omitempty"`
}

// EMeterConfig contains the writable energy meter configuration
// (/settings/emeter/{index}). These are the only writable parameters the Gen1
// API exposes for an energy meter; Gen1 has no CT-type setting (ct_type is a
// Gen2 EM/EM1 concept).
type EMeterConfig struct {
	// OverPowerURL is the action URL called when power exceeds OverPowerURLThreshold.
	OverPowerURL string `json:"over_power_url,omitempty"`
	// UnderPowerURL is the action URL called when power drops below UnderPowerURLThreshold.
	UnderPowerURL string `json:"under_power_url,omitempty"`
	// OverPowerURLThreshold is the overpower action threshold in watts.
	OverPowerURLThreshold float64 `json:"over_power_url_threshold,omitempty"`
	// UnderPowerURLThreshold is the underpower action threshold in watts.
	UnderPowerURLThreshold float64 `json:"under_power_url_threshold,omitempty"`
	// MaxPower is the overpower-protection threshold in watts.
	MaxPower float64 `json:"max_power,omitempty"`
}

// EMeterData contains historical energy data.
type EMeterData struct {
	// Timestamp is when the data was recorded.
	Timestamp int64 `json:"timestamp,omitempty"`

	// Total is total energy in watt-hours.
	Total float64 `json:"total,omitempty"`

	// TotalReturned is total returned energy.
	TotalReturned float64 `json:"total_returned,omitempty"`
}

// GetStatus retrieves the current energy meter readings.
func (e *EMeter) GetStatus(ctx context.Context) (*EMeterStatus, error) {
	path := fmt.Sprintf("/emeter/%d", e.id)
	resp, err := restCall(ctx, e.transport, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get emeter status: %w", err)
	}

	var status EMeterStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to parse emeter status: %w", err)
	}

	return &status, nil
}

// GetPower returns the current power consumption in watts.
// Negative values indicate power being returned to the grid.
func (e *EMeter) GetPower(ctx context.Context) (float64, error) {
	status, err := e.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.Power, nil
}

// GetVoltage returns the current voltage in volts.
func (e *EMeter) GetVoltage(ctx context.Context) (float64, error) {
	status, err := e.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.Voltage, nil
}

// GetCurrent returns the current in amps.
func (e *EMeter) GetCurrent(ctx context.Context) (float64, error) {
	status, err := e.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.Current, nil
}

// GetPowerFactor returns the power factor (-1 to 1).
func (e *EMeter) GetPowerFactor(ctx context.Context) (float64, error) {
	status, err := e.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.PF, nil
}

// GetTotal returns the total consumed energy in watt-hours.
func (e *EMeter) GetTotal(ctx context.Context) (float64, error) {
	status, err := e.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.Total, nil
}

// GetTotalKWh returns the total consumed energy in kilowatt-hours.
func (e *EMeter) GetTotalKWh(ctx context.Context) (float64, error) {
	status, err := e.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.Total / 1000.0, nil
}

// GetTotalReturned returns the total returned energy in watt-hours.
func (e *EMeter) GetTotalReturned(ctx context.Context) (float64, error) {
	status, err := e.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.TotalReturned, nil
}

// GetTotalReturnedKWh returns the total returned energy in kilowatt-hours.
func (e *EMeter) GetTotalReturnedKWh(ctx context.Context) (float64, error) {
	status, err := e.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.TotalReturned / 1000.0, nil
}

// GetNetEnergy returns the net energy (consumed - returned) in watt-hours.
func (e *EMeter) GetNetEnergy(ctx context.Context) (float64, error) {
	status, err := e.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.Total - status.TotalReturned, nil
}

// ResetCounters resets the energy counters.
//
// Note: This may not be supported on all devices or require
// certain firmware versions.
func (e *EMeter) ResetCounters(ctx context.Context) error {
	path := fmt.Sprintf("/emeter/%d?reset_totals=true", e.id)
	_, err := restCall(ctx, e.transport, path)
	if err != nil {
		return fmt.Errorf("failed to reset counters: %w", err)
	}
	return nil
}

// GetConfig retrieves the energy meter configuration.
func (e *EMeter) GetConfig(ctx context.Context) (*EMeterConfig, error) {
	path := fmt.Sprintf("/settings/emeter/%d", e.id)
	resp, err := restCall(ctx, e.transport, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get emeter config: %w", err)
	}

	var config EMeterConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("failed to parse emeter config: %w", err)
	}

	return &config, nil
}

// SetConfig writes the writable energy meter settings to /settings/emeter/{index}:
// the overpower-protection threshold and the optional over/under-power action URLs
// and their thresholds. Zero-valued numeric fields and empty URLs are omitted. Per
// the Gen1 API these are the only writable emeter parameters — there is no ct_type
// on Gen1 (that belongs to the Gen2 EM/EM1 components).
func (e *EMeter) SetConfig(ctx context.Context, config EMeterConfig) error {
	params := url.Values{}
	if config.MaxPower > 0 {
		params.Set("max_power", strconv.FormatFloat(config.MaxPower, 'f', -1, 64))
	}
	if config.OverPowerURL != "" {
		params.Set("over_power_url", config.OverPowerURL)
	}
	if config.OverPowerURLThreshold > 0 {
		params.Set("over_power_url_threshold", strconv.FormatFloat(config.OverPowerURLThreshold, 'f', -1, 64))
	}
	if config.UnderPowerURL != "" {
		params.Set("under_power_url", config.UnderPowerURL)
	}
	if config.UnderPowerURLThreshold > 0 {
		params.Set("under_power_url_threshold", strconv.FormatFloat(config.UnderPowerURLThreshold, 'f', -1, 64))
	}
	if len(params) == 0 {
		return nil
	}
	path := fmt.Sprintf("/settings/emeter/%d?%s", e.id, params.Encode())
	if _, err := restCall(ctx, e.transport, path); err != nil {
		return fmt.Errorf("failed to set emeter config: %w", err)
	}
	return nil
}

// GetData retrieves historical energy data.
//
// Note: Historical data retrieval may have different endpoints
// depending on the device firmware version.
func (e *EMeter) GetData(ctx context.Context) ([]EMeterData, error) {
	path := fmt.Sprintf("/emeter/%d/em_data", e.id)
	resp, err := restCall(ctx, e.transport, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get emeter data: %w", err)
	}

	var data []EMeterData
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("failed to parse emeter data: %w", err)
	}

	return data, nil
}
