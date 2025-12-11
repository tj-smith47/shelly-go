package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

const plugsUIComponentType = "plugs_ui"

// UI represents a Shelly Gen2+ UI component for devices with displays or LEDs.
//
// The UI component handles the settings of a device's screen or LED indicators.
// Configuration options vary by device type:
//   - Shelly Pro 4PM, Pro 3: Screen idle brightness
//   - Shelly Plug S (PLUGS_UI): LED color mode (power/switch/off)
//   - Shelly Plus H&T (HT_UI): Temperature unit display
//
// For HT_UI specifically, use the HTUI component instead.
//
// Example:
//
//	ui := components.NewUI(client)
//	config, err := ui.GetConfig(ctx)
//	if err == nil {
//	    fmt.Printf("Idle brightness: %d\n", *config.IdleBrightness)
//	}
type UI struct {
	client *rpc.Client
}

// NewUI creates a new UI component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	ui := components.NewUI(rpcClient)
func NewUI(client *rpc.Client) *UI {
	return &UI{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (u *UI) Client() *rpc.Client {
	return u.client
}

// UIConfig represents the configuration of a UI component.
type UIConfig struct {
	// IdleBrightness is the display brightness when idle (Pro devices with screens).
	// Range: 0-100
	IdleBrightness *int `json:"idle_brightness,omitempty"`

	// LockChildLock enables child lock mode to prevent button presses.
	Lock *bool `json:"lock,omitempty"`

	// TempUnits is the temperature unit display ("C" or "F").
	// Used by devices with temperature displays.
	TempUnits *string `json:"temp_units,omitempty"`

	// Flip rotates the display 180 degrees.
	Flip *bool `json:"flip,omitempty"`

	// Brightness is the display brightness level.
	// Range varies by device (1-7 for some devices, 0-100 for others).
	Brightness *int `json:"brightness,omitempty"`

	// RawFields captures any additional fields for future compatibility.
	types.RawFields
}

// UIStatus represents the status of a UI component.
// Note: The UI component typically does not have status properties.
type UIStatus struct {
	// RawFields captures any additional fields for future compatibility.
	types.RawFields
}

// GetConfig retrieves the UI configuration.
//
// Example:
//
//	config, err := ui.GetConfig(ctx)
//	if err == nil && config.IdleBrightness != nil {
//	    fmt.Printf("Idle brightness: %d%%\n", *config.IdleBrightness)
//	}
func (u *UI) GetConfig(ctx context.Context) (*UIConfig, error) {
	resultJSON, err := u.client.Call(ctx, "Ui.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config UIConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the UI configuration.
//
// Only fields that are set (non-nil) will be updated.
//
// Example - Set idle brightness:
//
//	brightness := 50
//	err := ui.SetConfig(ctx, &UIConfig{
//	    IdleBrightness: &brightness,
//	})
//
// Example - Enable child lock:
//
//	lock := true
//	err := ui.SetConfig(ctx, &UIConfig{
//	    Lock: &lock,
//	})
func (u *UI) SetConfig(ctx context.Context, config *UIConfig) error {
	configMap := make(map[string]any)

	if config.IdleBrightness != nil {
		configMap["idle_brightness"] = *config.IdleBrightness
	}
	if config.Lock != nil {
		configMap["lock"] = *config.Lock
	}
	if config.TempUnits != nil {
		configMap["temp_units"] = *config.TempUnits
	}
	if config.Flip != nil {
		configMap["flip"] = *config.Flip
	}
	if config.Brightness != nil {
		configMap["brightness"] = *config.Brightness
	}

	params := map[string]any{
		"config": configMap,
	}

	_, err := u.client.Call(ctx, "Ui.SetConfig", params)
	return err
}

// SetIdleBrightness sets the screen idle brightness.
//
// This is a convenience method for devices with screens (Pro series).
//
// Example:
//
//	err := ui.SetIdleBrightness(ctx, 50) // 50% brightness
func (u *UI) SetIdleBrightness(ctx context.Context, brightness int) error {
	return u.SetConfig(ctx, &UIConfig{
		IdleBrightness: &brightness,
	})
}

// SetLock enables or disables the child lock.
//
// When enabled, button presses on the device are ignored.
//
// Example:
//
//	err := ui.SetLock(ctx, true)  // Enable child lock
//	err := ui.SetLock(ctx, false) // Disable child lock
func (u *UI) SetLock(ctx context.Context, lock bool) error {
	return u.SetConfig(ctx, &UIConfig{
		Lock: &lock,
	})
}

// SetTempUnits sets the temperature display unit.
//
// Valid values are "C" (Celsius) or "F" (Fahrenheit).
//
// Example:
//
//	err := ui.SetTempUnits(ctx, "F") // Display in Fahrenheit
func (u *UI) SetTempUnits(ctx context.Context, units string) error {
	return u.SetConfig(ctx, &UIConfig{
		TempUnits: &units,
	})
}

// Type returns the component type identifier.
func (u *UI) Type() string {
	return "ui"
}

// Key returns the component key for aggregated status/config responses.
func (u *UI) Key() string {
	return "ui"
}

// Ensure UI implements a minimal component-like interface.
var _ interface {
	Type() string
	Key() string
} = (*UI)(nil)

// PlugsUI represents a Shelly Plug S UI component for LED control.
//
// The PLUGS_UI component handles the settings of a Plus Plug S device's LEDs.
// It allows setting the LED color mode to indicate power, switch state, or off.
//
// Example:
//
//	plugsUI := components.NewPlugsUI(client)
//	config, err := plugsUI.GetConfig(ctx)
//	if err == nil {
//	    fmt.Printf("LED mode: %s\n", *config.LEDs.Mode)
//	}
type PlugsUI struct {
	client *rpc.Client
}

// NewPlugsUI creates a new PLUGS_UI component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	plugsUI := components.NewPlugsUI(rpcClient)
func NewPlugsUI(client *rpc.Client) *PlugsUI {
	return &PlugsUI{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (p *PlugsUI) Client() *rpc.Client {
	return p.client
}

// PlugsUIConfig represents the configuration of a PLUGS_UI component.
type PlugsUIConfig struct {
	// LEDs contains the LED configuration.
	LEDs *PlugsUILEDConfig `json:"leds,omitempty"`

	// RawFields captures any additional fields for future compatibility.
	types.RawFields
}

// PlugsUILEDConfig represents the LED settings.
type PlugsUILEDConfig struct {
	Mode       *string        `json:"mode,omitempty"`
	Brightness *int           `json:"brightness,omitempty"`
	Colors     []PlugsUIColor `json:"colors,omitempty"`
}

// PlugsUIColor represents a color configuration for power-based LED indication.
type PlugsUIColor struct {
	// Power is the power threshold in watts.
	Power float64 `json:"power"`

	// RGB is the color value as 24-bit RGB integer.
	RGB int `json:"rgb"`
}

// PlugsUIStatus represents the status of a PLUGS_UI component.
type PlugsUIStatus struct {
	// RawFields captures any additional fields for future compatibility.
	types.RawFields
}

// GetConfig retrieves the PLUGS_UI configuration.
//
// Example:
//
//	config, err := plugsUI.GetConfig(ctx)
//	if err == nil && config.LEDs != nil {
//	    fmt.Printf("LED mode: %s\n", *config.LEDs.Mode)
//	}
func (p *PlugsUI) GetConfig(ctx context.Context) (*PlugsUIConfig, error) {
	resultJSON, err := p.client.Call(ctx, "PLUGS_UI.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config PlugsUIConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the PLUGS_UI configuration.
//
// Example - Set LED mode to switch:
//
//	mode := "switch"
//	err := plugsUI.SetConfig(ctx, &PlugsUIConfig{
//	    LEDs: &PlugsUILEDConfig{
//	        Mode: &mode,
//	    },
//	})
func (p *PlugsUI) SetConfig(ctx context.Context, config *PlugsUIConfig) error {
	configMap := make(map[string]any)

	//nolint:nestif // Nested LED config structure requires nested nil checks
	if config.LEDs != nil {
		ledsMap := make(map[string]any)
		if config.LEDs.Mode != nil {
			ledsMap["mode"] = *config.LEDs.Mode
		}
		if config.LEDs.Brightness != nil {
			ledsMap["brightness"] = *config.LEDs.Brightness
		}
		if len(config.LEDs.Colors) > 0 {
			colors := make([]map[string]any, len(config.LEDs.Colors))
			for i, c := range config.LEDs.Colors {
				colors[i] = map[string]any{
					"power": c.Power,
					"rgb":   c.RGB,
				}
			}
			ledsMap["colors"] = colors
		}
		if len(ledsMap) > 0 {
			configMap["leds"] = ledsMap
		}
	}

	params := map[string]any{
		"config": configMap,
	}

	_, err := p.client.Call(ctx, "PLUGS_UI.SetConfig", params)
	return err
}

// SetLEDMode sets the LED indicator mode.
//
// Valid modes are:
//   - "power": LEDs indicate power consumption
//   - "switch": LEDs indicate switch state (on/off)
//   - "off": LEDs are disabled
//
// Example:
//
//	err := plugsUI.SetLEDMode(ctx, "switch")
func (p *PlugsUI) SetLEDMode(ctx context.Context, mode string) error {
	return p.SetConfig(ctx, &PlugsUIConfig{
		LEDs: &PlugsUILEDConfig{
			Mode: &mode,
		},
	})
}

// SetLEDBrightness sets the LED brightness level.
//
// Example:
//
//	err := plugsUI.SetLEDBrightness(ctx, 75) // 75% brightness
func (p *PlugsUI) SetLEDBrightness(ctx context.Context, brightness int) error {
	return p.SetConfig(ctx, &PlugsUIConfig{
		LEDs: &PlugsUILEDConfig{
			Brightness: &brightness,
		},
	})
}

// Type returns the component type identifier.
func (p *PlugsUI) Type() string {
	return plugsUIComponentType
}

// Key returns the component key for aggregated status/config responses.
func (p *PlugsUI) Key() string {
	return plugsUIComponentType
}

// Ensure PlugsUI implements a minimal component-like interface.
var _ interface {
	Type() string
	Key() string
} = (*PlugsUI)(nil)
