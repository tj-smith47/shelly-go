// Package components provides Gen1 Shelly device components.
package components

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/tj-smith47/shelly-go/transport"
)

// Relay provides control for Gen1 relay outputs.
//
// Relays are the basic switching elements in Shelly devices.
// They can be turned on/off, toggled, and configured with timers.
type Relay struct {
	transport transport.Transport
	id        int
}

// NewRelay creates a new Relay component accessor.
//
// Parameters:
//   - t: The transport to use for API calls
//   - id: The relay index (0-based)
func NewRelay(t transport.Transport, id int) *Relay {
	return &Relay{
		transport: t,
		id:        id,
	}
}

// ID returns the relay index.
func (r *Relay) ID() int {
	return r.id
}

// RelayStatus contains the current relay state.
type RelayStatus struct {
	Source         string `json:"source,omitempty"`
	TimerStarted   int64  `json:"timer_started,omitempty"`
	TimerDuration  int    `json:"timer_duration,omitempty"`
	TimerRemaining int    `json:"timer_remaining,omitempty"`
	IsOn           bool   `json:"ison"`
	HasTimer       bool   `json:"has_timer,omitempty"`
	Overpower      bool   `json:"overpower,omitempty"`
}

// RelayConfig contains relay configuration options.
type RelayConfig struct {
	Name          string   `json:"name,omitempty"`
	ApplianceType string   `json:"appliance_type,omitempty"`
	DefaultState  string   `json:"default_state,omitempty"`
	BtnType       string   `json:"btn_type,omitempty"`
	ScheduleRules []string `json:"schedule_rules,omitempty"`
	AutoOn        float64  `json:"auto_on,omitempty"`
	AutoOff       float64  `json:"auto_off,omitempty"`
	MaxPower      int      `json:"max_power,omitempty"`
	BtnReverse    bool     `json:"btn_reverse,omitempty"`
	Schedule      bool     `json:"schedule,omitempty"`
}

// GetStatus retrieves the current relay status.
func (r *Relay) GetStatus(ctx context.Context) (*RelayStatus, error) {
	path := fmt.Sprintf("/relay/%d", r.id)
	resp, err := restCall(ctx, r.transport, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get relay status: %w", err)
	}

	var status RelayStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to parse relay status: %w", err)
	}

	return &status, nil
}

// TurnOn turns the relay on.
func (r *Relay) TurnOn(ctx context.Context) error {
	path := fmt.Sprintf("/relay/%d?turn=on", r.id)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn relay on: %w", err)
	}
	return nil
}

// TurnOff turns the relay off.
func (r *Relay) TurnOff(ctx context.Context) error {
	path := fmt.Sprintf("/relay/%d?turn=off", r.id)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn relay off: %w", err)
	}
	return nil
}

// Toggle toggles the relay state.
func (r *Relay) Toggle(ctx context.Context) error {
	path := fmt.Sprintf("/relay/%d?turn=toggle", r.id)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to toggle relay: %w", err)
	}
	return nil
}

// Set sets the relay to a specific state.
func (r *Relay) Set(ctx context.Context, on bool) error {
	if on {
		return r.TurnOn(ctx)
	}
	return r.TurnOff(ctx)
}

// TurnOnForDuration turns the relay on for a specified duration.
//
// Parameters:
//   - duration: Timer duration in seconds
func (r *Relay) TurnOnForDuration(ctx context.Context, duration int) error {
	path := fmt.Sprintf("/relay/%d?turn=on&timer=%d", r.id, duration)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn relay on with timer: %w", err)
	}
	return nil
}

// TurnOffForDuration turns the relay off for a specified duration.
//
// Parameters:
//   - duration: Timer duration in seconds
func (r *Relay) TurnOffForDuration(ctx context.Context, duration int) error {
	path := fmt.Sprintf("/relay/%d?turn=off&timer=%d", r.id, duration)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn relay off with timer: %w", err)
	}
	return nil
}

// GetConfig retrieves the relay configuration.
func (r *Relay) GetConfig(ctx context.Context) (*RelayConfig, error) {
	path := fmt.Sprintf("/settings/relay/%d", r.id)
	resp, err := restCall(ctx, r.transport, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get relay config: %w", err)
	}

	var config RelayConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("failed to parse relay config: %w", err)
	}

	return &config, nil
}

// SetConfig updates relay configuration.
//
// Only non-zero values in the config will be applied.
func (r *Relay) SetConfig(ctx context.Context, config *RelayConfig) error {
	params := url.Values{}

	if config.Name != "" {
		params.Set("name", config.Name)
	}
	if config.ApplianceType != "" {
		params.Set("appliance_type", config.ApplianceType)
	}
	if config.DefaultState != "" {
		params.Set("default_state", config.DefaultState)
	}
	if config.BtnType != "" {
		params.Set("btn_type", config.BtnType)
	}
	if config.BtnReverse {
		params.Set("btn_reverse", boolTrue)
	}
	if config.AutoOn > 0 {
		params.Set("auto_on", strconv.FormatFloat(config.AutoOn, 'f', -1, 64))
	}
	if config.AutoOff > 0 {
		params.Set("auto_off", strconv.FormatFloat(config.AutoOff, 'f', -1, 64))
	}
	if config.MaxPower > 0 {
		params.Set("max_power", strconv.Itoa(config.MaxPower))
	}
	if config.Schedule {
		params.Set("schedule", boolTrue)
	}

	if len(params) == 0 {
		return nil // Nothing to set
	}

	path := fmt.Sprintf("/settings/relay/%d?%s", r.id, params.Encode())
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set relay config: %w", err)
	}

	return nil
}

// SetName sets the relay name.
func (r *Relay) SetName(ctx context.Context, name string) error {
	path := fmt.Sprintf("/settings/relay/%d?name=%s", r.id, url.QueryEscape(name))
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set relay name: %w", err)
	}
	return nil
}

// SetDefaultState sets the default power-on state.
//
// Parameters:
//   - state: "off", "on", "last", or "switch"
func (r *Relay) SetDefaultState(ctx context.Context, state string) error {
	path := fmt.Sprintf("/settings/relay/%d?default_state=%s", r.id, state)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set default state: %w", err)
	}
	return nil
}

// SetButtonType sets the button input type.
//
// Parameters:
//   - btnType: "momentary", "toggle", "edge", or "detached"
func (r *Relay) SetButtonType(ctx context.Context, btnType string) error {
	path := fmt.Sprintf("/settings/relay/%d?btn_type=%s", r.id, btnType)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set button type: %w", err)
	}
	return nil
}

// SetAutoOn sets the auto-on timer.
//
// Parameters:
//   - seconds: Seconds until auto-on (0 to disable)
func (r *Relay) SetAutoOn(ctx context.Context, seconds float64) error {
	path := fmt.Sprintf("/settings/relay/%d?auto_on=%v", r.id, seconds)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set auto-on: %w", err)
	}
	return nil
}

// SetAutoOff sets the auto-off timer.
//
// Parameters:
//   - seconds: Seconds until auto-off (0 to disable)
func (r *Relay) SetAutoOff(ctx context.Context, seconds float64) error {
	path := fmt.Sprintf("/settings/relay/%d?auto_off=%v", r.id, seconds)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set auto-off: %w", err)
	}
	return nil
}

// SetMaxPower sets the maximum power limit.
//
// Parameters:
//   - watts: Maximum power in watts (0 to disable)
func (r *Relay) SetMaxPower(ctx context.Context, watts int) error {
	path := fmt.Sprintf("/settings/relay/%d?max_power=%d", r.id, watts)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set max power: %w", err)
	}
	return nil
}

// SetSchedule enables or disables the schedule.
func (r *Relay) SetSchedule(ctx context.Context, enabled bool) error {
	val := boolFalse
	if enabled {
		val = boolTrue
	}
	path := fmt.Sprintf("/settings/relay/%d?schedule=%s", r.id, val)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set schedule: %w", err)
	}
	return nil
}

// AddScheduleRule adds a schedule rule.
//
// Parameters:
//   - rule: Schedule rule in format "HHMM-0123456-on" or "HHMM-0123456-off"
//     where HHMM is time and 0123456 are days (0=Sunday)
func (r *Relay) AddScheduleRule(ctx context.Context, rule string) error {
	// Get current rules
	config, err := r.GetConfig(ctx)
	if err != nil {
		return err
	}

	// Add new rule (intentionally creating new slice to avoid modifying original config)
	rules := append(config.ScheduleRules, rule) //nolint:gocritic // intentional: create new slice

	// Set rules (need to format as JSON array in query string)
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	path := fmt.Sprintf("/settings/relay/%d?schedule_rules=%s", r.id, string(rulesJSON))
	_, err = restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to add schedule rule: %w", err)
	}
	return nil
}

// ClearScheduleRules removes all schedule rules.
func (r *Relay) ClearScheduleRules(ctx context.Context) error {
	path := fmt.Sprintf("/settings/relay/%d?schedule_rules=[]", r.id)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to clear schedule rules: %w", err)
	}
	return nil
}
