package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// RGBW represents a Shelly Gen2+ RGBW component.
//
// RGBW components control RGBW LED outputs with color, white channel, and brightness control.
// They support night mode for automatic brightness reduction.
//
// Example:
//
//	rgbw := components.NewRGBW(device.Client(), 0)
//	err := rgbw.Set(ctx, &RGBWSetParams{On: true, RGB: []int{255, 0, 0}, White: 100})
type RGBW struct {
	*gen2.BaseComponent
}

// NewRGBW creates a new RGBW component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (usually 0 for single-RGBW devices)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	rgbw := components.NewRGBW(device.Client(), 0)
func NewRGBW(client *rpc.Client, id int) *RGBW {
	return &RGBW{
		BaseComponent: gen2.NewBaseComponent(client, "rgbw", id),
	}
}

// RGBWConfig represents the configuration of an RGBW component.
type RGBWConfig struct {
	AutoOffDelay          *float64           `json:"auto_off_delay,omitempty"`
	ButtonPresets         *RGBWButtonPresets `json:"button_presets,omitempty"`
	InitialState          *string            `json:"initial_state,omitempty"`
	AutoOn                *bool              `json:"auto_on,omitempty"`
	AutoOnDelay           *float64           `json:"auto_on_delay,omitempty"`
	AutoOff               *bool              `json:"auto_off,omitempty"`
	Name                  *string            `json:"name,omitempty"`
	MinBrightnessOnToggle *int               `json:"min_brightness_on_toggle,omitempty"`
	types.RawFields
	NightMode          *RGBWNightModeConfig `json:"night_mode,omitempty"`
	TransitionDuration *float64             `json:"transition_duration,omitempty"`
	DefaultBrightness  *int                 `json:"default_brightness,omitempty"`
	DefaultWhite       *int                 `json:"default_white,omitempty"`
	DefaultRGB         []int                `json:"default_rgb,omitempty"`
	ID                 int                  `json:"id"`
}

// RGBWNightModeConfig represents night mode configuration for RGBW.
type RGBWNightModeConfig struct {
	Enable     *bool `json:"enable,omitempty"`
	Brightness *int  `json:"brightness,omitempty"`
	White      *int  `json:"white,omitempty"`
	types.RawFields
	RGB           []int    `json:"rgb,omitempty"`
	ActiveBetween []string `json:"active_between,omitempty"`
}

// RGBWButtonPresets represents button preset configuration for RGBW.
type RGBWButtonPresets struct {
	Brightness *int `json:"brightness,omitempty"`
	White      *int `json:"white,omitempty"`
	types.RawFields
	RGB []int `json:"rgb,omitempty"`
}

// RGBWStatus represents the current status of an RGBW component.
type RGBWStatus struct {
	Current *float64 `json:"current,omitempty"`
	APower  *float64 `json:"apower,omitempty"`
	types.RawFields
	Brightness         *int               `json:"brightness,omitempty"`
	Voltage            *float64           `json:"voltage,omitempty"`
	White              *int               `json:"white,omitempty"`
	TimerStartedAt     *float64           `json:"timer_started_at,omitempty"`
	Temperature        *TemperatureSensor `json:"temperature,omitempty"`
	TransitionDuration *float64           `json:"transition_duration,omitempty"`
	TimerDuration      *float64           `json:"timer_duration,omitempty"`
	Source             string             `json:"source"`
	RGB                []int              `json:"rgb,omitempty"`
	Flags              []string           `json:"flags,omitempty"`
	Errors             []string           `json:"errors,omitempty"`
	ID                 int                `json:"id"`
	Output             bool               `json:"output"`
}

// RGBWSetParams contains parameters for the RGBW.Set method.
type RGBWSetParams struct {
	On                 *bool    `json:"on,omitempty"`
	Brightness         *int     `json:"brightness,omitempty"`
	White              *int     `json:"white,omitempty"`
	TransitionDuration *float64 `json:"transition_duration,omitempty"`
	ToggleAfter        *float64 `json:"toggle_after,omitempty"`
	types.RawFields
	RGB []int `json:"rgb,omitempty"`
	ID  int   `json:"id"`
}

// RGBWSetResult contains the result of an RGBW.Set call.
type RGBWSetResult struct {
	types.RawFields
	WasOn bool `json:"was_on"`
}

// RGBWToggleParams contains parameters for the RGBW.Toggle method.
type RGBWToggleParams struct {
	types.RawFields
	ID int `json:"id"`
}

// RGBWToggleResult contains the result of an RGBW.Toggle call.
type RGBWToggleResult struct {
	types.RawFields
	WasOn bool `json:"was_on"`
}

// Set sets the RGBW output, color, white channel, and brightness.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - params: RGBW set parameters
//
// Returns the previous state of the RGBW.
//
// Example:
//
//	// Turn on with warm white
//	result, err := rgbw.Set(ctx, &RGBWSetParams{
//	    On: ptr(true),
//	    RGB: []int{255, 200, 150},
//	    White: ptr(128),
//	    Brightness: ptr(100),
//	})
func (r *RGBW) Set(ctx context.Context, params *RGBWSetParams) (*RGBWSetResult, error) {
	params = gen2.EnsureID(r.BaseComponent, params)

	var result RGBWSetResult
	resultJSON, err := r.BaseComponent.Client().Call(ctx, "RGBW.Set", params)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Toggle toggles the RGBW state (on -> off, off -> on).
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns the previous state of the RGBW.
//
// Example:
//
//	result, err := rgbw.Toggle(ctx)
func (r *RGBW) Toggle(ctx context.Context) (*RGBWToggleResult, error) {
	params := gen2.EnsureID(r.BaseComponent, &RGBWToggleParams{})

	var result RGBWToggleResult
	resultJSON, err := r.BaseComponent.Client().Call(ctx, "RGBW.Toggle", params)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetConfig retrieves the RGBW configuration.
//
// Example:
//
//	config, err := rgbw.GetConfig(ctx)
func (r *RGBW) GetConfig(ctx context.Context) (*RGBWConfig, error) {
	return gen2.UnmarshalConfig[RGBWConfig](ctx, r.BaseComponent)
}

// SetConfig updates the RGBW configuration.
//
// Example:
//
//	err := rgbw.SetConfig(ctx, &RGBWConfig{
//	    Name: ptr("LED Strip"),
//	    DefaultRGB: []int{255, 255, 255},
//	    DefaultWhite: ptr(128),
//	})
func (r *RGBW) SetConfig(ctx context.Context, config *RGBWConfig) error {
	return gen2.SetConfigWithID(ctx, r.BaseComponent, config)
}

// GetStatus retrieves the current RGBW status.
//
// Example:
//
//	status, err := rgbw.GetStatus(ctx)
//	fmt.Printf("Output: %v, RGB: %v, White: %d\n", status.Output, status.RGB, *status.White)
func (r *RGBW) GetStatus(ctx context.Context) (*RGBWStatus, error) {
	return gen2.UnmarshalStatus[RGBWStatus](ctx, r.BaseComponent)
}

// ResetCounters resets the energy counters.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - types: Optional list of counter types to reset
func (r *RGBW) ResetCounters(ctx context.Context, counterTypes []string) error {
	params := map[string]any{
		"id": r.ID(),
	}

	if len(counterTypes) > 0 {
		params["type"] = counterTypes
	}

	_, err := r.BaseComponent.Client().Call(ctx, "RGBW.ResetCounters", params)
	return err
}
