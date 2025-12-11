package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// thermostatComponentType is the type identifier for the Thermostat component.
const thermostatComponentType = "thermostat"

// Thermostat represents a Shelly Gen2+ Thermostat component.
//
// The Thermostat component is available on specific devices like the Shelly BLU TRV
// (Thermostatic Radiator Valve) when accessed through a Shelly BLU Gateway. It provides
// temperature regulation control for heating systems.
//
// Note: Standard Gen2+ devices (Plus, Pro series) do not have a Thermostat component.
// Thermostat functionality on those devices is typically provided through virtual
// components or device-specific services.
//
// Example (via BLU Gateway):
//
//	thermostat := components.NewThermostat(client, 0)
//	status, err := thermostat.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Current: %.1f°C, Target: %.1f°C\n", status.CurrentC, status.TargetC)
//	}
type Thermostat struct {
	client *rpc.Client
	id     int
}

// NewThermostat creates a new Thermostat component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Thermostat component ID (usually 0)
//
// Example:
//
//	thermostat := components.NewThermostat(rpcClient, 0)
func NewThermostat(client *rpc.Client, id int) *Thermostat {
	return &Thermostat{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (t *Thermostat) Client() *rpc.Client {
	return t.client
}

// ID returns the Thermostat component ID.
func (t *Thermostat) ID() int {
	return t.id
}

// ThermostatConfig represents the configuration of a Thermostat component.
type ThermostatConfig struct {
	DefaultOverrideDuration *int             `json:"default_override_duration,omitempty"`
	Flags                   *ThermostatFlags `json:"flags,omitempty"`
	TargetC                 *float64         `json:"target_C,omitempty"`
	OverrideEnable          *bool            `json:"override_enable,omitempty"`
	MinValvePosition        *int             `json:"min_valve_position,omitempty"`
	DefaultBoostDuration    *int             `json:"default_boost_duration,omitempty"`
	TempOffset              *float64         `json:"temp_offset,omitempty"`
	types.RawFields
	Enable                 *bool    `json:"enable,omitempty"`
	HumidityOffset         *float64 `json:"humidity_offset,omitempty"`
	TempUnit               *string  `json:"temp_unit,omitempty"`
	ThermostatMode         *string  `json:"thermostat_mode,omitempty"`
	DefaultOverrideTargetC *float64 `json:"default_override_target_C,omitempty"`
	ID                     int      `json:"id"`
}

// ThermostatFlags contains optional configuration flags for the thermostat.
type ThermostatFlags struct {
	// FloorHeating indicates floor heating mode.
	FloorHeating *bool `json:"floor_heating,omitempty"`

	// Accel indicates accelerated heating mode.
	Accel *bool `json:"accel,omitempty"`

	// AutoCalibrate indicates automatic calibration correction.
	AutoCalibrate *bool `json:"auto_calibrate,omitempty"`

	// AntiClog indicates anti-clog function.
	AntiClog *bool `json:"anticlog,omitempty"`
}

// ThermostatStatus represents the status of a Thermostat component.
type ThermostatStatus struct {
	TargetHumidity *float64 `json:"target_humidity,omitempty"`
	ScheduleRev    *int     `json:"schedule_rev,omitempty"`
	Steps          *int     `json:"steps,omitempty"`
	CurrentC       *float64 `json:"current_C,omitempty"`
	CurrentF       *float64 `json:"current_F,omitempty"`
	TargetC        *float64 `json:"target_C,omitempty"`
	Pos            *int     `json:"pos,omitempty"`
	TargetF        *float64 `json:"target_F,omitempty"`
	types.RawFields
	CurrentHumidity *float64            `json:"current_humidity,omitempty"`
	Output          *bool               `json:"output,omitempty"`
	Boost           *ThermostatModeInfo `json:"boost,omitempty"`
	Override        *ThermostatModeInfo `json:"override,omitempty"`
	Flags           []string            `json:"flags,omitempty"`
	Errors          []string            `json:"errors,omitempty"`
	ID              int                 `json:"id"`
}

// ThermostatModeInfo contains information about boost or override mode.
type ThermostatModeInfo struct {
	// StartedAt is the Unix timestamp when the mode started.
	StartedAt int64 `json:"started_at,omitempty"`

	// Duration is the duration of the mode in seconds.
	Duration int `json:"duration,omitempty"`
}

// GetConfig retrieves the Thermostat configuration.
//
// Example:
//
//	config, err := thermostat.GetConfig(ctx)
//	if err == nil {
//	    fmt.Printf("Target: %.1f°C, Mode: %s\n", *config.TargetC, *config.ThermostatMode)
//	}
func (t *Thermostat) GetConfig(ctx context.Context) (*ThermostatConfig, error) {
	params := map[string]any{
		"id": t.id,
	}

	resultJSON, err := t.client.Call(ctx, "Thermostat.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config ThermostatConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the Thermostat configuration.
//
// Only fields that are set (non-nil) will be updated.
//
// Example - Set target temperature:
//
//	targetC := 21.5
//	err := thermostat.SetConfig(ctx, &ThermostatConfig{
//	    TargetC: &targetC,
//	})
//
// Example - Enable thermostat with mode:
//
//	enable := true
//	mode := "heat"
//	err := thermostat.SetConfig(ctx, &ThermostatConfig{
//	    Enable: &enable,
//	    ThermostatMode: &mode,
//	})
//
//nolint:gocyclo,cyclop // Thermostat SetConfig has many optional parameters per Shelly API
func (t *Thermostat) SetConfig(ctx context.Context, config *ThermostatConfig) error {
	configMap := make(map[string]any)

	if config.Enable != nil {
		configMap["enable"] = *config.Enable
	}
	if config.TargetC != nil {
		configMap["target_C"] = *config.TargetC
	}
	if config.OverrideEnable != nil {
		configMap["override_enable"] = *config.OverrideEnable
	}
	if config.MinValvePosition != nil {
		configMap["min_valve_position"] = *config.MinValvePosition
	}
	if config.DefaultBoostDuration != nil {
		configMap["default_boost_duration"] = *config.DefaultBoostDuration
	}
	if config.DefaultOverrideDuration != nil {
		configMap["default_override_duration"] = *config.DefaultOverrideDuration
	}
	if config.DefaultOverrideTargetC != nil {
		configMap["default_override_target_C"] = *config.DefaultOverrideTargetC
	}
	if config.TempOffset != nil {
		configMap["temp_offset"] = *config.TempOffset
	}
	if config.HumidityOffset != nil {
		configMap["humidity_offset"] = *config.HumidityOffset
	}
	if config.TempUnit != nil {
		configMap["temp_unit"] = *config.TempUnit
	}
	if config.ThermostatMode != nil {
		configMap["thermostat_mode"] = *config.ThermostatMode
	}
	//nolint:nestif // Nested config structure requires nested nil checks
	if config.Flags != nil {
		flagsMap := make(map[string]any)
		if config.Flags.FloorHeating != nil {
			flagsMap["floor_heating"] = *config.Flags.FloorHeating
		}
		if config.Flags.Accel != nil {
			flagsMap["accel"] = *config.Flags.Accel
		}
		if config.Flags.AutoCalibrate != nil {
			flagsMap["auto_calibrate"] = *config.Flags.AutoCalibrate
		}
		if config.Flags.AntiClog != nil {
			flagsMap["anticlog"] = *config.Flags.AntiClog
		}
		if len(flagsMap) > 0 {
			configMap["flags"] = flagsMap
		}
	}

	params := map[string]any{
		"id":     t.id,
		"config": configMap,
	}

	_, err := t.client.Call(ctx, "Thermostat.SetConfig", params)
	return err
}

// GetStatus retrieves the current Thermostat status.
//
// Example:
//
//	status, err := thermostat.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Current: %.1f°C, Target: %.1f°C, Valve: %d%%\n",
//	        *status.CurrentC, *status.TargetC, *status.Pos)
//	}
func (t *Thermostat) GetStatus(ctx context.Context) (*ThermostatStatus, error) {
	params := map[string]any{
		"id": t.id,
	}

	resultJSON, err := t.client.Call(ctx, "Thermostat.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status ThermostatStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// SetTarget sets the target temperature.
//
// This is a convenience method equivalent to calling SetConfig with TargetC.
//
// Example:
//
//	err := thermostat.SetTarget(ctx, 22.0) // Set to 22°C
func (t *Thermostat) SetTarget(ctx context.Context, targetC float64) error {
	return t.SetConfig(ctx, &ThermostatConfig{
		TargetC: &targetC,
	})
}

// Enable enables or disables the thermostat.
//
// Example:
//
//	err := thermostat.Enable(ctx, true)  // Enable
//	err := thermostat.Enable(ctx, false) // Disable
func (t *Thermostat) Enable(ctx context.Context, enable bool) error {
	return t.SetConfig(ctx, &ThermostatConfig{
		Enable: &enable,
	})
}

// SetMode sets the thermostat operating mode.
//
// Valid modes are "cool", "heat", or "auto".
//
// Example:
//
//	err := thermostat.SetMode(ctx, "heat")
func (t *Thermostat) SetMode(ctx context.Context, mode string) error {
	return t.SetConfig(ctx, &ThermostatConfig{
		ThermostatMode: &mode,
	})
}

// Boost activates boost mode with optional duration.
//
// Boost mode sets the valve to 100% for rapid heating.
// If duration is 0, the default boost duration is used.
//
// Example:
//
//	err := thermostat.Boost(ctx, 300) // Boost for 5 minutes
func (t *Thermostat) Boost(ctx context.Context, durationSec int) error {
	params := map[string]any{
		"id": t.id,
	}
	if durationSec > 0 {
		params["duration"] = durationSec
	}

	_, err := t.client.Call(ctx, "Thermostat.Boost", params)
	return err
}

// CancelBoost cancels an active boost mode.
//
// Example:
//
//	err := thermostat.CancelBoost(ctx)
func (t *Thermostat) CancelBoost(ctx context.Context) error {
	params := map[string]any{
		"id": t.id,
	}

	_, err := t.client.Call(ctx, "Thermostat.CancelBoost", params)
	return err
}

// Override activates temperature override mode.
//
// Override mode temporarily sets a different target temperature.
// If duration is 0, the default override duration is used.
// If targetC is 0, the default override target is used.
//
// Example:
//
//	err := thermostat.Override(ctx, 25.0, 1800) // 25°C for 30 minutes
func (t *Thermostat) Override(ctx context.Context, targetC float64, durationSec int) error {
	params := map[string]any{
		"id": t.id,
	}
	if targetC > 0 {
		params["target_C"] = targetC
	}
	if durationSec > 0 {
		params["duration"] = durationSec
	}

	_, err := t.client.Call(ctx, "Thermostat.Override", params)
	return err
}

// CancelOverride cancels an active override mode.
//
// Example:
//
//	err := thermostat.CancelOverride(ctx)
func (t *Thermostat) CancelOverride(ctx context.Context) error {
	params := map[string]any{
		"id": t.id,
	}

	_, err := t.client.Call(ctx, "Thermostat.CancelOverride", params)
	return err
}

// Calibrate initiates valve calibration.
//
// This should be called after installation or if the valve behavior
// seems incorrect.
//
// Example:
//
//	err := thermostat.Calibrate(ctx)
func (t *Thermostat) Calibrate(ctx context.Context) error {
	params := map[string]any{
		"id": t.id,
	}

	_, err := t.client.Call(ctx, "Thermostat.Calibrate", params)
	return err
}

// Type returns the component type identifier.
func (t *Thermostat) Type() string {
	return thermostatComponentType
}

// Key returns the component key for aggregated status/config responses.
func (t *Thermostat) Key() string {
	return thermostatComponentType
}

// Ensure Thermostat implements a minimal component-like interface.
var _ interface {
	Type() string
	Key() string
	ID() int
} = (*Thermostat)(nil)
