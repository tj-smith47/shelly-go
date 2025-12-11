package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// RGB represents a Shelly Gen2+ RGB component.
//
// RGB components control RGB LED outputs with color and brightness control.
// They support night mode for automatic brightness reduction.
//
// Example:
//
//	rgb := components.NewRGB(device.Client(), 0)
//	err := rgb.Set(ctx, &RGBSetParams{On: true, RGB: []int{255, 0, 0}})
type RGB struct {
	*gen2.BaseComponent
}

// NewRGB creates a new RGB component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (usually 0 for single-RGB devices)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	rgb := components.NewRGB(device.Client(), 0)
func NewRGB(client *rpc.Client, id int) *RGB {
	return &RGB{
		BaseComponent: gen2.NewBaseComponent(client, "rgb", id),
	}
}

// RGBConfig represents the configuration of an RGB component.
type RGBConfig struct {
	AutoOffDelay          *float64          `json:"auto_off_delay,omitempty"`
	ButtonPresets         *RGBButtonPresets `json:"button_presets,omitempty"`
	InitialState          *string           `json:"initial_state,omitempty"`
	AutoOn                *bool             `json:"auto_on,omitempty"`
	AutoOnDelay           *float64          `json:"auto_on_delay,omitempty"`
	AutoOff               *bool             `json:"auto_off,omitempty"`
	MinBrightnessOnToggle *int              `json:"min_brightness_on_toggle,omitempty"`
	types.RawFields
	Name               *string             `json:"name,omitempty"`
	NightMode          *RGBNightModeConfig `json:"night_mode,omitempty"`
	TransitionDuration *float64            `json:"transition_duration,omitempty"`
	DefaultBrightness  *int                `json:"default_brightness,omitempty"`
	DefaultRGB         []int               `json:"default_rgb,omitempty"`
	ID                 int                 `json:"id"`
}

// RGBNightModeConfig represents night mode configuration for RGB.
type RGBNightModeConfig struct {
	Enable     *bool `json:"enable,omitempty"`
	Brightness *int  `json:"brightness,omitempty"`
	types.RawFields
	RGB           []int    `json:"rgb,omitempty"`
	ActiveBetween []string `json:"active_between,omitempty"`
}

// RGBButtonPresets represents button preset configuration for RGB.
type RGBButtonPresets struct {
	Brightness *int `json:"brightness,omitempty"`
	types.RawFields
	RGB []int `json:"rgb,omitempty"`
}

// RGBStatus represents the current status of an RGB component.
type RGBStatus struct {
	TimerDuration      *float64 `json:"timer_duration,omitempty"`
	TransitionDuration *float64 `json:"transition_duration,omitempty"`
	types.RawFields
	Brightness     *int               `json:"brightness,omitempty"`
	Current        *float64           `json:"current,omitempty"`
	TimerStartedAt *float64           `json:"timer_started_at,omitempty"`
	Temperature    *TemperatureSensor `json:"temperature,omitempty"`
	Voltage        *float64           `json:"voltage,omitempty"`
	APower         *float64           `json:"apower,omitempty"`
	Source         string             `json:"source"`
	RGB            []int              `json:"rgb,omitempty"`
	Flags          []string           `json:"flags,omitempty"`
	Errors         []string           `json:"errors,omitempty"`
	ID             int                `json:"id"`
	Output         bool               `json:"output"`
}

// RGBSetParams contains parameters for the RGB.Set method.
type RGBSetParams struct {
	On                 *bool    `json:"on,omitempty"`
	Brightness         *int     `json:"brightness,omitempty"`
	TransitionDuration *float64 `json:"transition_duration,omitempty"`
	ToggleAfter        *float64 `json:"toggle_after,omitempty"`
	types.RawFields
	RGB []int `json:"rgb,omitempty"`
	ID  int   `json:"id"`
}

// RGBSetResult contains the result of an RGB.Set call.
type RGBSetResult struct {
	types.RawFields
	WasOn bool `json:"was_on"`
}

// RGBToggleParams contains parameters for the RGB.Toggle method.
type RGBToggleParams struct {
	types.RawFields
	ID int `json:"id"`
}

// RGBToggleResult contains the result of an RGB.Toggle call.
type RGBToggleResult struct {
	types.RawFields
	WasOn bool `json:"was_on"`
}

// Set sets the RGB output, color, and brightness.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - params: RGB set parameters
//
// Returns the previous state of the RGB.
//
// Example:
//
//	// Turn on with red color
//	result, err := rgb.Set(ctx, &RGBSetParams{
//	    On: ptr(true),
//	    RGB: []int{255, 0, 0},
//	    Brightness: ptr(100),
//	})
func (r *RGB) Set(ctx context.Context, params *RGBSetParams) (*RGBSetResult, error) {
	params = gen2.EnsureID(r.BaseComponent, params)

	var result RGBSetResult
	resultJSON, err := r.BaseComponent.Client().Call(ctx, "RGB.Set", params)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Toggle toggles the RGB state (on -> off, off -> on).
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns the previous state of the RGB.
//
// Example:
//
//	result, err := rgb.Toggle(ctx)
func (r *RGB) Toggle(ctx context.Context) (*RGBToggleResult, error) {
	params := gen2.EnsureID(r.BaseComponent, &RGBToggleParams{})

	var result RGBToggleResult
	resultJSON, err := r.BaseComponent.Client().Call(ctx, "RGB.Toggle", params)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetConfig retrieves the RGB configuration.
//
// Example:
//
//	config, err := rgb.GetConfig(ctx)
func (r *RGB) GetConfig(ctx context.Context) (*RGBConfig, error) {
	return gen2.UnmarshalConfig[RGBConfig](ctx, r.BaseComponent)
}

// SetConfig updates the RGB configuration.
//
// Example:
//
//	err := rgb.SetConfig(ctx, &RGBConfig{
//	    Name: ptr("Accent Light"),
//	    DefaultRGB: []int{255, 128, 0},
//	})
func (r *RGB) SetConfig(ctx context.Context, config *RGBConfig) error {
	return gen2.SetConfigWithID(ctx, r.BaseComponent, config)
}

// GetStatus retrieves the current RGB status.
//
// Example:
//
//	status, err := rgb.GetStatus(ctx)
//	fmt.Printf("Output: %v, RGB: %v\n", status.Output, status.RGB)
func (r *RGB) GetStatus(ctx context.Context) (*RGBStatus, error) {
	return gen2.UnmarshalStatus[RGBStatus](ctx, r.BaseComponent)
}

// ResetCounters resets the energy counters.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - types: Optional list of counter types to reset
func (r *RGB) ResetCounters(ctx context.Context, counterTypes []string) error {
	params := map[string]any{
		"id": r.ID(),
	}

	if len(counterTypes) > 0 {
		params["type"] = counterTypes
	}

	_, err := r.BaseComponent.Client().Call(ctx, "RGB.ResetCounters", params)
	return err
}
