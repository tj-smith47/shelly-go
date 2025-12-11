package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Light represents a Shelly Gen2+ Light component.
//
// Light components control dimmable lights with optional RGB/RGBW color support.
// They support brightness control, color temperature, and various lighting effects.
//
// Example:
//
//	light := components.NewLight(device.Client(), 0)
//	err := light.Set(ctx, &LightSetParams{On: true, Brightness: 75})
type Light struct {
	*gen2.BaseComponent
}

// NewLight creates a new Light component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (usually 0 for single-light devices)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	light := components.NewLight(device.Client(), 0)
func NewLight(client *rpc.Client, id int) *Light {
	return &Light{
		BaseComponent: gen2.NewBaseComponent(client, "light", id),
	}
}

// LightConfig represents the configuration of a Light component.
type LightConfig struct {
	Name                  *string          `json:"name,omitempty"`
	InitialState          *string          `json:"initial_state,omitempty"`
	AutoOn                *bool            `json:"auto_on,omitempty"`
	AutoOnDelay           *float64         `json:"auto_on_delay,omitempty"`
	AutoOff               *bool            `json:"auto_off,omitempty"`
	AutoOffDelay          *float64         `json:"auto_off_delay,omitempty"`
	TransitionDuration    *int             `json:"transition_duration,omitempty"`
	MinBrightnessOnToggle *int             `json:"min_brightness_on_toggle,omitempty"`
	NightMode             *NightModeConfig `json:"night_mode,omitempty"`
	DefaultBrightness     *int             `json:"default_brightness,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// NightModeConfig represents night mode configuration.
type NightModeConfig struct {
	Enable     *bool `json:"enable,omitempty"`
	Brightness *int  `json:"brightness,omitempty"`
	types.RawFields
	ActiveBetween []string `json:"active_between,omitempty"`
}

// LightStatus represents the current status of a Light component.
type LightStatus struct {
	TransitionDuration *int               `json:"transition_duration,omitempty"`
	Brightness         *int               `json:"brightness,omitempty"`
	TimerStartedAt     *float64           `json:"timer_started_at,omitempty"`
	TimerDuration      *float64           `json:"timer_duration,omitempty"`
	Temperature        *TemperatureSensor `json:"temperature,omitempty"`
	APower             *float64           `json:"apower,omitempty"`
	Voltage            *float64           `json:"voltage,omitempty"`
	Current            *float64           `json:"current,omitempty"`
	types.RawFields
	Source string   `json:"source"`
	Errors []string `json:"errors,omitempty"`
	ID     int      `json:"id"`
	Output bool     `json:"output"`
}

// LightSetParams contains parameters for the Light.Set method.
type LightSetParams struct {
	On                 *bool    `json:"on,omitempty"`
	Brightness         *int     `json:"brightness,omitempty"`
	TransitionDuration *int     `json:"transition_duration,omitempty"`
	ToggleAfter        *float64 `json:"toggle_after,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// LightSetResult contains the result of a Light.Set call.
type LightSetResult struct {
	// WasOn indicates the previous state of the light
	WasOn *bool `json:"was_on,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// LightToggleParams contains parameters for the Light.Toggle method.
type LightToggleParams struct {
	types.RawFields
	ID int `json:"id"`
}

// LightToggleResult contains the result of a Light.Toggle call.
type LightToggleResult struct {
	// WasOn indicates the previous state of the light
	WasOn *bool `json:"was_on,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// Set controls the light state and parameters.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - params: Light set parameters
//
// Returns the previous state of the light.
//
// Example:
//
//	// Turn on at 50% brightness
//	result, err := light.Set(ctx, &LightSetParams{
//	    ID: 0,
//	    On: ptr(true),
//	    Brightness: ptr(50),
//	})
//
//	// Turn on with fade effect
//	result, err := light.Set(ctx, &LightSetParams{
//	    ID: 0,
//	    On: ptr(true),
//	    Brightness: ptr(75),
//	    TransitionDuration: ptr(1000), // 1 second fade
//	})
func (l *Light) Set(ctx context.Context, params *LightSetParams) (*LightSetResult, error) {
	params = gen2.EnsureID(l.BaseComponent, params)

	var result LightSetResult
	resultJSON, err := l.Client().Call(ctx, "Light.Set", params)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Toggle toggles the light state (on -> off, off -> on).
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns the previous state of the light.
//
// Example:
//
//	result, err := light.Toggle(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Was on: %v\n", *result.WasOn)
func (l *Light) Toggle(ctx context.Context) (*LightToggleResult, error) {
	params := gen2.EnsureID(l.BaseComponent, &LightToggleParams{})

	var result LightToggleResult
	resultJSON, err := l.Client().Call(ctx, "Light.Toggle", params)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetConfig retrieves the light configuration.
//
// Example:
//
//	config, err := light.GetConfig(ctx)
func (l *Light) GetConfig(ctx context.Context) (*LightConfig, error) {
	return gen2.UnmarshalConfig[LightConfig](ctx, l.BaseComponent)
}

// SetConfig updates the light configuration.
//
// Example:
//
//	err := light.SetConfig(ctx, &LightConfig{
//	    Name: ptr("Living Room Light"),
//	    DefaultBrightness: ptr(75),
//	    TransitionDuration: ptr(500),
//	})
func (l *Light) SetConfig(ctx context.Context, config *LightConfig) error {
	return gen2.SetConfigWithID(ctx, l.BaseComponent, config)
}

// GetStatus retrieves the current light status.
//
// Example:
//
//	status, err := light.GetStatus(ctx)
//	fmt.Printf("Output: %v, Brightness: %d%%\n", status.Output, *status.Brightness)
func (l *Light) GetStatus(ctx context.Context) (*LightStatus, error) {
	return gen2.UnmarshalStatus[LightStatus](ctx, l.BaseComponent)
}
