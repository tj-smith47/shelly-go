package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/transport"
)

// Roller provides control for Gen1 roller/cover/shutter outputs.
//
// Rollers control motorized covers, shutters, blinds, and similar devices.
// They support open, close, stop, and position control.
type Roller struct {
	transport transport.Transport
	id        int
}

// NewRoller creates a new Roller component accessor.
//
// Parameters:
//   - t: The transport to use for API calls
//   - id: The roller index (0-based)
func NewRoller(t transport.Transport, id int) *Roller {
	return &Roller{
		transport: t,
		id:        id,
	}
}

// ID returns the roller index.
func (r *Roller) ID() int {
	return r.id
}

// RollerStatus contains the current roller state.
type RollerStatus struct {
	State           string  `json:"state,omitempty"`
	Source          string  `json:"source,omitempty"`
	StopReason      string  `json:"stop_reason,omitempty"`
	LastDirection   string  `json:"last_direction,omitempty"`
	Power           float64 `json:"power,omitempty"`
	CurrentPos      int     `json:"current_pos,omitempty"`
	IsValid         bool    `json:"is_valid,omitempty"`
	SafetySwitch    bool    `json:"safety_switch,omitempty"`
	Overtemperature bool    `json:"overtemperature,omitempty"`
	Calibrating     bool    `json:"calibrating,omitempty"`
	Positioning     bool    `json:"positioning,omitempty"`
}

// RollerConfig contains roller configuration options.
type RollerConfig struct {
	DefaultState           string  `json:"default_state,omitempty"`
	SafetyAction           string  `json:"safety_action,omitempty"`
	InputMode              string  `json:"input_mode,omitempty"`
	BtnType                string  `json:"btn_type,omitempty"`
	SafetyMode             string  `json:"safety_mode,omitempty"`
	ObstacleMode           string  `json:"obstacle_mode,omitempty"`
	ObstacleAction         string  `json:"obstacle_action,omitempty"`
	ObstacleDelay          int     `json:"obstacle_delay,omitempty"`
	MaxTimeOpen            float64 `json:"maxtime_open,omitempty"`
	MaxTimeClose           float64 `json:"maxtime_close,omitempty"`
	MaxTime                float64 `json:"maxtime,omitempty"`
	ObstaclePower          int     `json:"obstacle_power,omitempty"`
	SwapInputs             bool    `json:"swap_inputs,omitempty"`
	BtnReverse             bool    `json:"btn_reverse,omitempty"`
	Swap                   bool    `json:"swap,omitempty"`
	SafetyAllowedOnTrigger bool    `json:"safety_allowed_on_trigger,omitempty"`
	Positioning            bool    `json:"positioning,omitempty"`
}

// GetStatus retrieves the current roller status.
func (r *Roller) GetStatus(ctx context.Context) (*RollerStatus, error) {
	path := fmt.Sprintf("/roller/%d", r.id)
	resp, err := restCall(ctx, r.transport, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get roller status: %w", err)
	}

	var status RollerStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to parse roller status: %w", err)
	}

	return &status, nil
}

// Open starts opening the roller (moves to fully open position).
func (r *Roller) Open(ctx context.Context) error {
	path := fmt.Sprintf("/roller/%d?go=open", r.id)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to open roller: %w", err)
	}
	return nil
}

// Close starts closing the roller (moves to fully closed position).
func (r *Roller) Close(ctx context.Context) error {
	path := fmt.Sprintf("/roller/%d?go=close", r.id)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to close roller: %w", err)
	}
	return nil
}

// Stop stops the roller movement.
func (r *Roller) Stop(ctx context.Context) error {
	path := fmt.Sprintf("/roller/%d?go=stop", r.id)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to stop roller: %w", err)
	}
	return nil
}

// GoToPosition moves the roller to a specific position.
//
// Parameters:
//   - pos: Target position (0-100, 0=closed, 100=open)
//
// Note: Positioning must be calibrated for this to work.
func (r *Roller) GoToPosition(ctx context.Context, pos int) error {
	if pos < 0 || pos > 100 {
		return fmt.Errorf("position must be 0-100, got %d", pos)
	}

	path := fmt.Sprintf("/roller/%d?go=to_pos&roller_pos=%d", r.id, pos)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to go to position: %w", err)
	}
	return nil
}

// OpenForDuration opens the roller for a specified duration.
//
// Parameters:
//   - seconds: Duration in seconds
func (r *Roller) OpenForDuration(ctx context.Context, seconds float64) error {
	path := fmt.Sprintf("/roller/%d?go=open&duration=%v", r.id, seconds)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to open for duration: %w", err)
	}
	return nil
}

// CloseForDuration closes the roller for a specified duration.
//
// Parameters:
//   - seconds: Duration in seconds
func (r *Roller) CloseForDuration(ctx context.Context, seconds float64) error {
	path := fmt.Sprintf("/roller/%d?go=close&duration=%v", r.id, seconds)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to close for duration: %w", err)
	}
	return nil
}

// Calibrate starts the calibration procedure.
//
// Calibration measures the time required to fully open and close the roller.
// This is required for position control to work accurately.
func (r *Roller) Calibrate(ctx context.Context) error {
	path := fmt.Sprintf("/roller/%d/calibrate", r.id)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to start calibration: %w", err)
	}
	return nil
}

// GetConfig retrieves the roller configuration.
func (r *Roller) GetConfig(ctx context.Context) (*RollerConfig, error) {
	path := fmt.Sprintf("/settings/roller/%d", r.id)
	resp, err := restCall(ctx, r.transport, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get roller config: %w", err)
	}

	var config RollerConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("failed to parse roller config: %w", err)
	}

	return &config, nil
}

// SetConfig updates roller configuration.
//
// Only non-zero values in the config will be applied.
//
//nolint:gocyclo,cyclop // Roller SetConfig has many optional parameters per Shelly API
func (r *Roller) SetConfig(ctx context.Context, config *RollerConfig) error {
	path := fmt.Sprintf("/settings/roller/%d?", r.id)

	params := ""
	if config.MaxTime > 0 {
		params += fmt.Sprintf("maxtime=%v&", config.MaxTime)
	}
	if config.MaxTimeOpen > 0 {
		params += fmt.Sprintf("maxtime_open=%v&", config.MaxTimeOpen)
	}
	if config.MaxTimeClose > 0 {
		params += fmt.Sprintf("maxtime_close=%v&", config.MaxTimeClose)
	}
	if config.DefaultState != "" {
		params += fmt.Sprintf("default_state=%s&", config.DefaultState)
	}
	if config.SwapInputs {
		params += "swap_inputs=true&"
	}
	if config.Swap {
		params += "swap=true&"
	}
	if config.InputMode != "" {
		params += fmt.Sprintf("input_mode=%s&", config.InputMode)
	}
	if config.BtnType != "" {
		params += fmt.Sprintf("btn_type=%s&", config.BtnType)
	}
	if config.BtnReverse {
		params += btnReverseParam
	}
	if config.ObstacleMode != "" {
		params += fmt.Sprintf("obstacle_mode=%s&", config.ObstacleMode)
	}
	if config.ObstacleAction != "" {
		params += fmt.Sprintf("obstacle_action=%s&", config.ObstacleAction)
	}
	if config.ObstaclePower > 0 {
		params += fmt.Sprintf("obstacle_power=%d&", config.ObstaclePower)
	}
	if config.ObstacleDelay > 0 {
		params += fmt.Sprintf("obstacle_delay=%d&", config.ObstacleDelay)
	}
	if config.SafetyMode != "" {
		params += fmt.Sprintf("safety_mode=%s&", config.SafetyMode)
	}
	if config.SafetyAction != "" {
		params += fmt.Sprintf("safety_action=%s&", config.SafetyAction)
	}
	if config.SafetyAllowedOnTrigger {
		params += "safety_allowed_on_trigger=true&"
	}
	if config.Positioning {
		params += "positioning=true&"
	}

	if params == "" {
		return nil // Nothing to set
	}

	// Remove trailing &
	params = params[:len(params)-1]

	_, err := restCall(ctx, r.transport, path+params)
	if err != nil {
		return fmt.Errorf("failed to set roller config: %w", err)
	}

	return nil
}

// SetMaxTime sets the maximum operation time.
func (r *Roller) SetMaxTime(ctx context.Context, seconds float64) error {
	path := fmt.Sprintf("/settings/roller/%d?maxtime=%v", r.id, seconds)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set max time: %w", err)
	}
	return nil
}

// SetDefaultState sets the default power-on state.
//
// Parameters:
//   - state: "stop", "open", "close", or "last"
func (r *Roller) SetDefaultState(ctx context.Context, state string) error {
	path := fmt.Sprintf("/settings/roller/%d?default_state=%s", r.id, state)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set default state: %w", err)
	}
	return nil
}

// SetInputMode sets the input mode.
//
// Parameters:
//   - mode: "openclose" (two buttons) or "onebutton" (single toggle)
func (r *Roller) SetInputMode(ctx context.Context, mode string) error {
	path := fmt.Sprintf("/settings/roller/%d?input_mode=%s", r.id, mode)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set input mode: %w", err)
	}
	return nil
}

// SetButtonType sets the button input type.
//
// Parameters:
//   - btnType: "momentary", "toggle", or "detached"
func (r *Roller) SetButtonType(ctx context.Context, btnType string) error {
	path := fmt.Sprintf("/settings/roller/%d?btn_type=%s", r.id, btnType)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set button type: %w", err)
	}
	return nil
}

// EnablePositioning enables or disables position control.
func (r *Roller) EnablePositioning(ctx context.Context, enabled bool) error {
	val := boolFalse
	if enabled {
		val = boolTrue
	}
	path := fmt.Sprintf("/settings/roller/%d?positioning=%s", r.id, val)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set positioning: %w", err)
	}
	return nil
}

// SetObstacleDetection configures obstacle detection.
//
// Parameters:
//   - mode: "disabled", "while_opening", "while_closing", or "both"
//   - action: "stop" or "reverse"
//   - powerThreshold: Power threshold in watts for detection
//   - delay: Delay before checking in seconds
func (r *Roller) SetObstacleDetection(ctx context.Context, mode, action string, powerThreshold, delay int) error {
	path := fmt.Sprintf("/settings/roller/%d?obstacle_mode=%s&obstacle_action=%s&obstacle_power=%d&obstacle_delay=%d",
		r.id, mode, action, powerThreshold, delay)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set obstacle detection: %w", err)
	}
	return nil
}

// SetSafetySwitch configures safety switch behavior.
//
// Parameters:
//   - mode: "disabled", "while_opening", "while_closing", or "both"
//   - action: "stop", "pause", or "reverse"
func (r *Roller) SetSafetySwitch(ctx context.Context, mode, action string) error {
	path := fmt.Sprintf("/settings/roller/%d?safety_mode=%s&safety_action=%s", r.id, mode, action)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set safety switch: %w", err)
	}
	return nil
}

// SwapDirection swaps the open/close direction.
func (r *Roller) SwapDirection(ctx context.Context, swap bool) error {
	val := boolFalse
	if swap {
		val = boolTrue
	}
	path := fmt.Sprintf("/settings/roller/%d?swap=%s", r.id, val)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to swap direction: %w", err)
	}
	return nil
}

// SwapInputs swaps the input button assignments.
func (r *Roller) SwapInputs(ctx context.Context, swap bool) error {
	val := boolFalse
	if swap {
		val = boolTrue
	}
	path := fmt.Sprintf("/settings/roller/%d?swap_inputs=%s", r.id, val)
	_, err := restCall(ctx, r.transport, path)
	if err != nil {
		return fmt.Errorf("failed to swap inputs: %w", err)
	}
	return nil
}
