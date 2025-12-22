package components

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/tj-smith47/shelly-go/transport"
)

// Light provides control for Gen1 light outputs.
//
// Light controls dimmable lights like the Shelly Dimmer and Shelly Bulb
// white channel. For RGBW control, use the Color component.
type Light struct {
	transport transport.Transport
	id        int
}

// NewLight creates a new Light component accessor.
//
// Parameters:
//   - t: The transport to use for API calls
//   - id: The light index (0-based)
func NewLight(t transport.Transport, id int) *Light {
	return &Light{
		transport: t,
		id:        id,
	}
}

// ID returns the light index.
func (l *Light) ID() int {
	return l.id
}

// LightStatus contains the current light state.
type LightStatus struct {
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

// LightConfig contains light configuration options.
type LightConfig struct {
	Name          string   `json:"name,omitempty"`
	DefaultState  string   `json:"default_state,omitempty"`
	BtnType       string   `json:"btn_type,omitempty"`
	ScheduleRules []string `json:"schedule_rules,omitempty"`
	AutoOn        float64  `json:"auto_on,omitempty"`
	AutoOff       float64  `json:"auto_off,omitempty"`
	MinBrightness int      `json:"min_brightness,omitempty"`
	BtnReverse    bool     `json:"btn_reverse,omitempty"`
	Schedule      bool     `json:"schedule,omitempty"`
}

// GetStatus retrieves the current light status.
func (l *Light) GetStatus(ctx context.Context) (*LightStatus, error) {
	path := fmt.Sprintf("/light/%d", l.id)
	resp, err := restCall(ctx, l.transport, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get light status: %w", err)
	}

	var status LightStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to parse light status: %w", err)
	}

	return &status, nil
}

// TurnOn turns the light on.
func (l *Light) TurnOn(ctx context.Context) error {
	path := fmt.Sprintf("/light/%d?turn=on", l.id)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn light on: %w", err)
	}
	return nil
}

// TurnOff turns the light off.
func (l *Light) TurnOff(ctx context.Context) error {
	path := fmt.Sprintf("/light/%d?turn=off", l.id)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn light off: %w", err)
	}
	return nil
}

// Toggle toggles the light state.
func (l *Light) Toggle(ctx context.Context) error {
	path := fmt.Sprintf("/light/%d?turn=toggle", l.id)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to toggle light: %w", err)
	}
	return nil
}

// Set sets the light state.
func (l *Light) Set(ctx context.Context, on bool) error {
	if on {
		return l.TurnOn(ctx)
	}
	return l.TurnOff(ctx)
}

// SetBrightness sets the brightness level.
//
// Parameters:
//   - brightness: Brightness level (0-100)
func (l *Light) SetBrightness(ctx context.Context, brightness int) error {
	if brightness < 0 || brightness > 100 {
		return fmt.Errorf("brightness must be 0-100, got %d", brightness)
	}

	path := fmt.Sprintf("/light/%d?brightness=%d", l.id, brightness)
	_, err := restCall(ctx, l.transport, path)
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
func (l *Light) SetBrightnessWithTransition(ctx context.Context, brightness, transitionMs int) error {
	if brightness < 0 || brightness > 100 {
		return fmt.Errorf("brightness must be 0-100, got %d", brightness)
	}

	path := fmt.Sprintf("/light/%d?brightness=%d&transition=%d", l.id, brightness, transitionMs)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}
	return nil
}

// TurnOnWithBrightness turns on with a specific brightness.
//
// Parameters:
//   - brightness: Brightness level (0-100)
func (l *Light) TurnOnWithBrightness(ctx context.Context, brightness int) error {
	if brightness < 0 || brightness > 100 {
		return fmt.Errorf("brightness must be 0-100, got %d", brightness)
	}

	path := fmt.Sprintf("/light/%d?turn=on&brightness=%d", l.id, brightness)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn on with brightness: %w", err)
	}
	return nil
}

// TurnOnForDuration turns the light on for a specified duration.
//
// Parameters:
//   - duration: Timer duration in seconds
func (l *Light) TurnOnForDuration(ctx context.Context, duration int) error {
	path := fmt.Sprintf("/light/%d?turn=on&timer=%d", l.id, duration)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn on with timer: %w", err)
	}
	return nil
}

// SetColorTemp sets the color temperature (for tunable white lights).
//
// Parameters:
//   - temp: Color temperature in Kelvin (device-dependent range)
func (l *Light) SetColorTemp(ctx context.Context, temp int) error {
	path := fmt.Sprintf("/light/%d?temp=%d", l.id, temp)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set color temp: %w", err)
	}
	return nil
}

// GetConfig retrieves the light configuration.
func (l *Light) GetConfig(ctx context.Context) (*LightConfig, error) {
	path := fmt.Sprintf("/settings/light/%d", l.id)
	resp, err := restCall(ctx, l.transport, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get light config: %w", err)
	}

	var config LightConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("failed to parse light config: %w", err)
	}

	return &config, nil
}

// SetConfig updates light configuration.
//
// Only non-zero values in the config will be applied.
func (l *Light) SetConfig(ctx context.Context, config *LightConfig) error {
	path := fmt.Sprintf("/settings/light/%d?", l.id)

	params := ""
	if config.Name != "" {
		params += fmt.Sprintf("name=%s&", config.Name)
	}
	if config.DefaultState != "" {
		params += fmt.Sprintf("default_state=%s&", config.DefaultState)
	}
	if config.AutoOn > 0 {
		params += fmt.Sprintf("auto_on=%v&", config.AutoOn)
	}
	if config.AutoOff > 0 {
		params += fmt.Sprintf("auto_off=%v&", config.AutoOff)
	}
	if config.BtnType != "" {
		params += fmt.Sprintf("btn_type=%s&", config.BtnType)
	}
	if config.BtnReverse {
		params += btnReverseParam
	}
	if config.Schedule {
		params += scheduleParam
	}
	if config.MinBrightness > 0 {
		params += fmt.Sprintf("min_brightness=%d&", config.MinBrightness)
	}

	if params == "" {
		return nil // Nothing to set
	}

	// Remove trailing &
	params = params[:len(params)-1]

	_, err := restCall(ctx, l.transport, path+params)
	if err != nil {
		return fmt.Errorf("failed to set light config: %w", err)
	}

	return nil
}

// SetName sets the light name.
func (l *Light) SetName(ctx context.Context, name string) error {
	path := fmt.Sprintf("/settings/light/%d?name=%s", l.id, url.QueryEscape(name))
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set light name: %w", err)
	}
	return nil
}

// SetDefaultState sets the default power-on state.
//
// Parameters:
//   - state: "off", "on", or "last"
func (l *Light) SetDefaultState(ctx context.Context, state string) error {
	path := fmt.Sprintf("/settings/light/%d?default_state=%s", l.id, state)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set default state: %w", err)
	}
	return nil
}

// SetButtonType sets the button input type.
//
// Parameters:
//   - btnType: "momentary", "toggle", "edge", or "detached"
func (l *Light) SetButtonType(ctx context.Context, btnType string) error {
	path := fmt.Sprintf("/settings/light/%d?btn_type=%s", l.id, btnType)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set button type: %w", err)
	}
	return nil
}

// SetAutoOn sets the auto-on timer.
//
// Parameters:
//   - seconds: Seconds until auto-on (0 to disable)
func (l *Light) SetAutoOn(ctx context.Context, seconds float64) error {
	path := fmt.Sprintf("/settings/light/%d?auto_on=%v", l.id, seconds)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set auto-on: %w", err)
	}
	return nil
}

// SetAutoOff sets the auto-off timer.
//
// Parameters:
//   - seconds: Seconds until auto-off (0 to disable)
func (l *Light) SetAutoOff(ctx context.Context, seconds float64) error {
	path := fmt.Sprintf("/settings/light/%d?auto_off=%v", l.id, seconds)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set auto-off: %w", err)
	}
	return nil
}

// SetMinBrightness sets the minimum brightness level.
//
// Parameters:
//   - brightness: Minimum brightness (1-100)
func (l *Light) SetMinBrightness(ctx context.Context, brightness int) error {
	if brightness < 1 || brightness > 100 {
		return fmt.Errorf("min brightness must be 1-100, got %d", brightness)
	}

	path := fmt.Sprintf("/settings/light/%d?min_brightness=%d", l.id, brightness)
	_, err := restCall(ctx, l.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set min brightness: %w", err)
	}
	return nil
}
