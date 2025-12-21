package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/transport"
)

// Color provides control for Gen1 RGBW outputs.
//
// Color controls RGBW devices like Shelly RGBW2 and Shelly Bulb
// when in color mode. For white-only control, use the White component.
type Color struct {
	transport transport.Transport
	id        int
}

// NewColor creates a new Color component accessor.
//
// Parameters:
//   - t: The transport to use for API calls
//   - id: The color index (0-based)
func NewColor(t transport.Transport, id int) *Color {
	return &Color{
		transport: t,
		id:        id,
	}
}

// ID returns the color index.
func (c *Color) ID() int {
	return c.id
}

// ColorStatus contains the current color state.
type ColorStatus struct {
	Mode           string `json:"mode,omitempty"`
	Source         string `json:"source,omitempty"`
	Green          int    `json:"green,omitempty"`
	Temp           int    `json:"temp,omitempty"`
	TimerDuration  int    `json:"timer_duration,omitempty"`
	TimerRemaining int    `json:"timer_remaining,omitempty"`
	Transition     int    `json:"transition,omitempty"`
	Red            int    `json:"red,omitempty"`
	Effect         int    `json:"effect,omitempty"`
	Blue           int    `json:"blue,omitempty"`
	White          int    `json:"white,omitempty"`
	Gain           int    `json:"gain,omitempty"`
	TimerStarted   int64  `json:"timer_started,omitempty"`
	Brightness     int    `json:"brightness,omitempty"`
	IsOn           bool   `json:"ison"`
	HasTimer       bool   `json:"has_timer,omitempty"`
}

// ColorConfig contains color configuration options.
type ColorConfig struct {
	Name          string   `json:"name,omitempty"`
	DefaultState  string   `json:"default_state,omitempty"`
	ScheduleRules []string `json:"schedule_rules,omitempty"`
	AutoOn        float64  `json:"auto_on,omitempty"`
	AutoOff       float64  `json:"auto_off,omitempty"`
	Schedule      bool     `json:"schedule,omitempty"`
}

func (c *ColorConfig) getName() string         { return c.Name }
func (c *ColorConfig) getDefaultState() string { return c.DefaultState }
func (c *ColorConfig) getAutoOn() float64      { return c.AutoOn }
func (c *ColorConfig) getAutoOff() float64     { return c.AutoOff }
func (c *ColorConfig) getSchedule() bool       { return c.Schedule }

// GetStatus retrieves the current color status.
func (c *Color) GetStatus(ctx context.Context) (*ColorStatus, error) {
	path := fmt.Sprintf("/color/%d", c.id)
	resp, err := restCall(ctx, c.transport, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get color status: %w", err)
	}

	var status ColorStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to parse color status: %w", err)
	}

	return &status, nil
}

// TurnOn turns the light on.
func (c *Color) TurnOn(ctx context.Context) error {
	path := fmt.Sprintf("/color/%d?turn=on", c.id)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn color on: %w", err)
	}
	return nil
}

// TurnOff turns the light off.
func (c *Color) TurnOff(ctx context.Context) error {
	path := fmt.Sprintf("/color/%d?turn=off", c.id)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn color off: %w", err)
	}
	return nil
}

// Toggle toggles the light state.
func (c *Color) Toggle(ctx context.Context) error {
	path := fmt.Sprintf("/color/%d?turn=toggle", c.id)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to toggle color: %w", err)
	}
	return nil
}

// Set sets the light state.
func (c *Color) Set(ctx context.Context, on bool) error {
	if on {
		return c.TurnOn(ctx)
	}
	return c.TurnOff(ctx)
}

// SetRGB sets the RGB color values.
//
// Parameters:
//   - red: Red value (0-255)
//   - green: Green value (0-255)
//   - blue: Blue value (0-255)
func (c *Color) SetRGB(ctx context.Context, red, green, blue int) error {
	if red < 0 || red > 255 || green < 0 || green > 255 || blue < 0 || blue > 255 {
		return fmt.Errorf("RGB values must be 0-255")
	}

	path := fmt.Sprintf("/color/%d?red=%d&green=%d&blue=%d", c.id, red, green, blue)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set RGB: %w", err)
	}
	return nil
}

// SetRGBW sets the RGBW color values.
//
// Parameters:
//   - red: Red value (0-255)
//   - green: Green value (0-255)
//   - blue: Blue value (0-255)
//   - white: White value (0-255)
func (c *Color) SetRGBW(ctx context.Context, red, green, blue, white int) error {
	if red < 0 || red > 255 || green < 0 || green > 255 || blue < 0 || blue > 255 || white < 0 || white > 255 {
		return fmt.Errorf("RGBW values must be 0-255")
	}

	path := fmt.Sprintf("/color/%d?red=%d&green=%d&blue=%d&white=%d", c.id, red, green, blue, white)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set RGBW: %w", err)
	}
	return nil
}

// SetGain sets the color gain/brightness.
//
// Parameters:
//   - gain: Gain value (0-100)
func (c *Color) SetGain(ctx context.Context, gain int) error {
	if gain < 0 || gain > 100 {
		return fmt.Errorf("gain must be 0-100, got %d", gain)
	}

	path := fmt.Sprintf("/color/%d?gain=%d", c.id, gain)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set gain: %w", err)
	}
	return nil
}

// SetWhiteChannel sets the white channel value.
//
// Parameters:
//   - white: White value (0-255)
func (c *Color) SetWhiteChannel(ctx context.Context, white int) error {
	if white < 0 || white > 255 {
		return fmt.Errorf("white must be 0-255, got %d", white)
	}

	path := fmt.Sprintf("/color/%d?white=%d", c.id, white)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set white channel: %w", err)
	}
	return nil
}

// SetEffect sets the light effect.
//
// Parameters:
//   - effect: Effect index (0 = off, 1+ = various effects)
func (c *Color) SetEffect(ctx context.Context, effect int) error {
	path := fmt.Sprintf("/color/%d?effect=%d", c.id, effect)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set effect: %w", err)
	}
	return nil
}

// SetTransition sets the transition time for changes.
//
// Parameters:
//   - transitionMs: Transition time in milliseconds
func (c *Color) SetTransition(ctx context.Context, transitionMs int) error {
	path := fmt.Sprintf("/color/%d?transition=%d", c.id, transitionMs)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set transition: %w", err)
	}
	return nil
}

// TurnOnWithRGB turns on with specific RGB values.
//
// Parameters:
//   - red, green, blue: RGB values (0-255)
//   - gain: Brightness (0-100)
func (c *Color) TurnOnWithRGB(ctx context.Context, red, green, blue, gain int) error {
	if red < 0 || red > 255 || green < 0 || green > 255 || blue < 0 || blue > 255 {
		return fmt.Errorf("RGB values must be 0-255")
	}
	if gain < 0 || gain > 100 {
		return fmt.Errorf("gain must be 0-100, got %d", gain)
	}

	path := fmt.Sprintf("/color/%d?turn=on&red=%d&green=%d&blue=%d&gain=%d", c.id, red, green, blue, gain)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn on with RGB: %w", err)
	}
	return nil
}

// TurnOnForDuration turns the light on for a specified duration.
//
// Parameters:
//   - duration: Timer duration in seconds
func (c *Color) TurnOnForDuration(ctx context.Context, duration int) error {
	path := fmt.Sprintf("/color/%d?turn=on&timer=%d", c.id, duration)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to turn on with timer: %w", err)
	}
	return nil
}

// GetConfig retrieves the color configuration.
func (c *Color) GetConfig(ctx context.Context) (*ColorConfig, error) {
	path := fmt.Sprintf("/settings/color/%d", c.id)
	resp, err := restCall(ctx, c.transport, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get color config: %w", err)
	}

	var config ColorConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("failed to parse color config: %w", err)
	}

	return &config, nil
}

// SetConfig updates color configuration.
//
// Only non-zero values in the config will be applied.
func (c *Color) SetConfig(ctx context.Context, config *ColorConfig) error {
	params := buildLightConfigQuery(config)
	if params == "" {
		return nil // Nothing to set
	}

	path := fmt.Sprintf("/settings/color/%d?%s", c.id, params)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set color config: %w", err)
	}

	return nil
}

// SetName sets the light name.
func (c *Color) SetName(ctx context.Context, name string) error {
	path := fmt.Sprintf("/settings/color/%d?name=%s", c.id, name)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set color name: %w", err)
	}
	return nil
}

// SetDefaultState sets the default power-on state.
//
// Parameters:
//   - state: "off", "on", or "last"
func (c *Color) SetDefaultState(ctx context.Context, state string) error {
	path := fmt.Sprintf("/settings/color/%d?default_state=%s", c.id, state)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set default state: %w", err)
	}
	return nil
}

// SetAutoOn sets the auto-on timer.
//
// Parameters:
//   - seconds: Seconds until auto-on (0 to disable)
func (c *Color) SetAutoOn(ctx context.Context, seconds float64) error {
	path := fmt.Sprintf("/settings/color/%d?auto_on=%v", c.id, seconds)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set auto-on: %w", err)
	}
	return nil
}

// SetAutoOff sets the auto-off timer.
//
// Parameters:
//   - seconds: Seconds until auto-off (0 to disable)
func (c *Color) SetAutoOff(ctx context.Context, seconds float64) error {
	path := fmt.Sprintf("/settings/color/%d?auto_off=%v", c.id, seconds)
	_, err := restCall(ctx, c.transport, path)
	if err != nil {
		return fmt.Errorf("failed to set auto-off: %w", err)
	}
	return nil
}
