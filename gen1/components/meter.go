package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/transport"
)

// Meter provides access to Gen1 power meter readings.
//
// Meters are available on devices with power monitoring like
// Shelly 1PM, 2.5, Plug S, etc.
type Meter struct {
	transport transport.Transport
	id        int
}

// NewMeter creates a new Meter component accessor.
//
// Parameters:
//   - t: The transport to use for API calls
//   - id: The meter index (0-based)
func NewMeter(t transport.Transport, id int) *Meter {
	return &Meter{
		transport: t,
		id:        id,
	}
}

// ID returns the meter index.
func (m *Meter) ID() int {
	return m.id
}

// MeterStatus contains power meter readings.
type MeterStatus struct {
	Counters  []float64 `json:"counters,omitempty"`
	Power     float64   `json:"power"`
	Overpower float64   `json:"overpower,omitempty"`
	Timestamp int64     `json:"timestamp,omitempty"`
	Total     int       `json:"total,omitempty"`
	IsValid   bool      `json:"is_valid,omitempty"`
}

// MeterConfig contains meter configuration options.
type MeterConfig struct {
	// PowerLimit is the overpower limit in watts.
	PowerLimit float64 `json:"power_limit,omitempty"`

	// UnderLimit is the under-power limit in watts.
	UnderLimit float64 `json:"under_limit,omitempty"`

	// OverLimit is the over-power limit in watts.
	OverLimit float64 `json:"over_limit,omitempty"`
}

// GetStatus retrieves the current meter readings.
func (m *Meter) GetStatus(ctx context.Context) (*MeterStatus, error) {
	path := fmt.Sprintf("/meter/%d", m.id)
	resp, err := m.transport.Call(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get meter status: %w", err)
	}

	var status MeterStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to parse meter status: %w", err)
	}

	return &status, nil
}

// GetPower returns the current power consumption in watts.
func (m *Meter) GetPower(ctx context.Context) (float64, error) {
	status, err := m.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.Power, nil
}

// GetTotal returns the total energy in watt-minutes.
func (m *Meter) GetTotal(ctx context.Context) (int, error) {
	status, err := m.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.Total, nil
}

// GetTotalKWh returns the total energy in kilowatt-hours.
func (m *Meter) GetTotalKWh(ctx context.Context) (float64, error) {
	status, err := m.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	// Total is in watt-minutes, convert to kWh
	return float64(status.Total) / (60.0 * 1000.0), nil
}

// GetCounters returns the rolling energy counters.
//
// Returns 3 values representing energy consumption for the last
// three rolling periods (typically minutes).
func (m *Meter) GetCounters(ctx context.Context) ([]float64, error) {
	status, err := m.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	return status.Counters, nil
}

// ResetCounters resets the energy counters.
//
// Note: This may not be supported on all devices.
func (m *Meter) ResetCounters(ctx context.Context) error {
	path := fmt.Sprintf("/meter/%d?reset_totals=true", m.id)
	_, err := m.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to reset counters: %w", err)
	}
	return nil
}

// SetPowerLimit sets the overpower protection limit.
//
// Parameters:
//   - watts: Power limit in watts (0 to disable)
func (m *Meter) SetPowerLimit(ctx context.Context, watts float64) error {
	// Power limit is typically set on the relay settings
	path := fmt.Sprintf("/settings/relay/%d?max_power=%v", m.id, watts)
	_, err := m.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set power limit: %w", err)
	}
	return nil
}
