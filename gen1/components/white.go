package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/transport"
)

// White provides control for Gen1 white channel outputs.
//
// White controls the white channel on RGBW devices like Shelly RGBW2
// when in white mode, or tunable white bulbs like Shelly Duo.
type White struct {
	transport transport.Transport
	id        int
}

// NewWhite creates a new White component accessor.
//
// Parameters:
//   - t: The transport to use for API calls
//   - id: The white channel index (0-based)
func NewWhite(t transport.Transport, id int) *White {
	return &White{
		transport: t,
		id:        id,
	}
}

// ID returns the white channel index.
func (w *White) ID() int {
	return w.id
}

// WhiteStatus contains the current white channel state.
type WhiteStatus struct {
	Source         string `json:"source,omitempty"`
	Mode           string `json:"mode,omitempty"`
	TimerStarted   int64  `json:"timer_started,omitempty"`
	TimerDuration  int    `json:"timer_duration,omitempty"`
	TimerRemaining int    `json:"timer_remaining,omitempty"`
	Brightness     int    `json:"brightness,omitempty"`
	Temp           int    `json:"temp,omitempty"`
	Transition     int    `json:"transition,omitempty"`
	IsOn           bool   `json:"ison"`
	HasTimer       bool   `json:"has_timer,omitempty"`
}

// WhiteConfig contains white channel configuration options.
type WhiteConfig struct {
	Name          string   `json:"name,omitempty"`
	DefaultState  string   `json:"default_state,omitempty"`
	ScheduleRules []string `json:"schedule_rules,omitempty"`
	AutoOn        float64  `json:"auto_on,omitempty"`
	AutoOff       float64  `json:"auto_off,omitempty"`
	Schedule      bool     `json:"schedule,omitempty"`
}

func (c *WhiteConfig) getName() string         { return c.Name }
func (c *WhiteConfig) getDefaultState() string { return c.DefaultState }
func (c *WhiteConfig) getAutoOn() float64      { return c.AutoOn }
func (c *WhiteConfig) getAutoOff() float64     { return c.AutoOff }
func (c *WhiteConfig) getSchedule() bool       { return c.Schedule }

// GetStatus retrieves the current white channel status.
func (w *White) GetStatus(ctx context.Context) (*WhiteStatus, error) {
	path := fmt.Sprintf("/white/%d", w.id)
	resp, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get white status: %w", err)
	}

	var status WhiteStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to parse white status: %w", err)
	}

	return &status, nil
}

// TurnOn turns the white channel on.
func (w *White) TurnOn(ctx context.Context) error {
	path := fmt.Sprintf("/white/%d?turn=on", w.id)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to turn white on: %w", err)
	}
	return nil
}

// TurnOff turns the white channel off.
func (w *White) TurnOff(ctx context.Context) error {
	path := fmt.Sprintf("/white/%d?turn=off", w.id)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to turn white off: %w", err)
	}
	return nil
}

// Toggle toggles the white channel state.
func (w *White) Toggle(ctx context.Context) error {
	path := fmt.Sprintf("/white/%d?turn=toggle", w.id)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to toggle white: %w", err)
	}
	return nil
}

// Set sets the white channel state.
func (w *White) Set(ctx context.Context, on bool) error {
	if on {
		return w.TurnOn(ctx)
	}
	return w.TurnOff(ctx)
}

// SetBrightness sets the brightness level.
//
// Parameters:
//   - brightness: Brightness level (0-100)
func (w *White) SetBrightness(ctx context.Context, brightness int) error {
	if brightness < 0 || brightness > 100 {
		return fmt.Errorf("brightness must be 0-100, got %d", brightness)
	}

	path := fmt.Sprintf("/white/%d?brightness=%d", w.id, brightness)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}
	return nil
}

// SetBrightnessWithTransition sets brightness with a transition time.
//
// Parameters:
//   - brightness: Brightness level (0-100)
//   - transitionMs: Transition time in milliseconds
func (w *White) SetBrightnessWithTransition(ctx context.Context, brightness, transitionMs int) error {
	if brightness < 0 || brightness > 100 {
		return fmt.Errorf("brightness must be 0-100, got %d", brightness)
	}

	path := fmt.Sprintf("/white/%d?brightness=%d&transition=%d", w.id, brightness, transitionMs)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}
	return nil
}

// SetColorTemp sets the color temperature.
//
// Parameters:
//   - temp: Color temperature in Kelvin (device-dependent range)
func (w *White) SetColorTemp(ctx context.Context, temp int) error {
	path := fmt.Sprintf("/white/%d?temp=%d", w.id, temp)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set color temp: %w", err)
	}
	return nil
}

// TurnOnWithBrightness turns on with a specific brightness.
//
// Parameters:
//   - brightness: Brightness level (0-100)
func (w *White) TurnOnWithBrightness(ctx context.Context, brightness int) error {
	if brightness < 0 || brightness > 100 {
		return fmt.Errorf("brightness must be 0-100, got %d", brightness)
	}

	path := fmt.Sprintf("/white/%d?turn=on&brightness=%d", w.id, brightness)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to turn on with brightness: %w", err)
	}
	return nil
}

// TurnOnWithColorTemp turns on with a specific color temperature.
//
// Parameters:
//   - temp: Color temperature in Kelvin
//   - brightness: Brightness level (0-100)
func (w *White) TurnOnWithColorTemp(ctx context.Context, temp, brightness int) error {
	if brightness < 0 || brightness > 100 {
		return fmt.Errorf("brightness must be 0-100, got %d", brightness)
	}

	path := fmt.Sprintf("/white/%d?turn=on&temp=%d&brightness=%d", w.id, temp, brightness)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to turn on with color temp: %w", err)
	}
	return nil
}

// TurnOnForDuration turns the white channel on for a specified duration.
//
// Parameters:
//   - duration: Timer duration in seconds
func (w *White) TurnOnForDuration(ctx context.Context, duration int) error {
	path := fmt.Sprintf("/white/%d?turn=on&timer=%d", w.id, duration)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to turn on with timer: %w", err)
	}
	return nil
}

// GetConfig retrieves the white channel configuration.
func (w *White) GetConfig(ctx context.Context) (*WhiteConfig, error) {
	path := fmt.Sprintf("/settings/white/%d", w.id)
	resp, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get white config: %w", err)
	}

	var config WhiteConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("failed to parse white config: %w", err)
	}

	return &config, nil
}

// SetConfig updates white channel configuration.
//
// Only non-zero values in the config will be applied.
func (w *White) SetConfig(ctx context.Context, config *WhiteConfig) error {
	params := buildLightConfigQuery(config)
	if params == "" {
		return nil // Nothing to set
	}

	path := fmt.Sprintf("/settings/white/%d?%s", w.id, params)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set white config: %w", err)
	}

	return nil
}

// SetName sets the channel name.
func (w *White) SetName(ctx context.Context, name string) error {
	path := fmt.Sprintf("/settings/white/%d?name=%s", w.id, name)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set white name: %w", err)
	}
	return nil
}

// SetDefaultState sets the default power-on state.
//
// Parameters:
//   - state: "off", "on", or "last"
func (w *White) SetDefaultState(ctx context.Context, state string) error {
	path := fmt.Sprintf("/settings/white/%d?default_state=%s", w.id, state)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set default state: %w", err)
	}
	return nil
}

// SetAutoOn sets the auto-on timer.
//
// Parameters:
//   - seconds: Seconds until auto-on (0 to disable)
func (w *White) SetAutoOn(ctx context.Context, seconds float64) error {
	path := fmt.Sprintf("/settings/white/%d?auto_on=%v", w.id, seconds)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set auto-on: %w", err)
	}
	return nil
}

// SetAutoOff sets the auto-off timer.
//
// Parameters:
//   - seconds: Seconds until auto-off (0 to disable)
func (w *White) SetAutoOff(ctx context.Context, seconds float64) error {
	path := fmt.Sprintf("/settings/white/%d?auto_off=%v", w.id, seconds)
	_, err := w.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set auto-off: %w", err)
	}
	return nil
}
