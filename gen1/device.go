package gen1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/tj-smith47/shelly-go/gen1/components"
	"github.com/tj-smith47/shelly-go/transport"
	"github.com/tj-smith47/shelly-go/types"
)

// Device represents a Gen1 Shelly device.
//
// Device provides access to Gen1 device functionality including relays,
// rollers, lights, meters, and device settings via the HTTP REST API.
// It implements the types.Device interface.
type Device struct {
	transport transport.Transport
	info      *DeviceInfo
	mu        sync.RWMutex
}

// NewDevice creates a new Gen1 device with the given transport.
//
// The transport should be configured for HTTP REST API access.
// For authentication, configure the transport with credentials.
//
// Example:
//
//	t := transport.NewHTTP("http://192.168.1.100",
//	    transport.WithAuth("admin", "password"))
//	device := gen1.NewDevice(t)
func NewDevice(t transport.Transport) *Device {
	return &Device{
		transport: t,
	}
}

// restCall is a helper to make Gen1 REST API calls.
func (d *Device) restCall(ctx context.Context, path string) (json.RawMessage, error) {
	return d.transport.Call(ctx, transport.NewSimpleRequest(path))
}

// GetDeviceInfo retrieves and caches device information.
//
// This calls the /shelly endpoint to get device identification
// and capabilities. The result is cached for subsequent calls.
func (d *Device) GetDeviceInfo(ctx context.Context) (*types.DeviceInfo, error) {
	d.mu.RLock()
	if d.info != nil {
		d.mu.RUnlock()
		info := d.info.ToTypesDeviceInfo()
		return info, nil
	}
	d.mu.RUnlock()

	info, err := d.fetchDeviceInfo(ctx)
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	d.info = info
	d.mu.Unlock()

	return info.ToTypesDeviceInfo(), nil
}

// fetchDeviceInfo fetches device info from the /shelly endpoint.
func (d *Device) fetchDeviceInfo(ctx context.Context) (*DeviceInfo, error) {
	resp, err := d.transport.Call(ctx, transport.NewSimpleRequest("/shelly"))
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	var info DeviceInfo
	if err := json.Unmarshal(resp, &info); err != nil {
		return nil, fmt.Errorf("failed to parse device info: %w", err)
	}

	return &info, nil
}

// GetStatus returns the current status of all device components.
//
// This calls the /status endpoint and returns the complete device status.
func (d *Device) GetStatus(ctx context.Context) (any, error) {
	return d.GetFullStatus(ctx)
}

// GetFullStatus returns the complete device status from /status endpoint.
//
// Returns a Status struct with all device component statuses.
func (d *Device) GetFullStatus(ctx context.Context) (*Status, error) {
	resp, err := d.restCall(ctx, "/status")
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var status Status
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	return &status, nil
}

// GetConfig returns the current device configuration.
//
// This calls the /settings endpoint and returns all device settings.
func (d *Device) GetConfig(ctx context.Context) (any, error) {
	return d.GetSettings(ctx)
}

// GetSettings returns all device settings from /settings endpoint.
//
// Returns a Settings struct with all device configuration.
func (d *Device) GetSettings(ctx context.Context) (*Settings, error) {
	resp, err := d.restCall(ctx, "/settings")
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal(resp, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings: %w", err)
	}

	return &settings, nil
}

// Reboot reboots the device.
//
// This calls the /reboot endpoint. The device will restart
// and temporarily be unavailable.
func (d *Device) Reboot(ctx context.Context) error {
	_, err := d.restCall(ctx, "/reboot")
	if err != nil {
		return fmt.Errorf("failed to reboot: %w", err)
	}
	return nil
}

// Generation returns the device generation.
//
// For Gen1 devices, this always returns types.Gen1.
func (d *Device) Generation() types.Generation {
	return types.Gen1
}

// Close closes the transport connection.
func (d *Device) Close() error {
	return d.transport.Close()
}

// Relay returns a Relay component accessor.
//
// Parameters:
//   - id: Relay index (0-based, e.g., 0 for first relay)
//
// Example:
//
//	relay := device.Relay(0)
//	err := relay.TurnOn(ctx)
func (d *Device) Relay(id int) *components.Relay {
	return components.NewRelay(d.transport, id)
}

// Roller returns a Roller (cover/shutter) component accessor.
//
// Parameters:
//   - id: Roller index (0-based)
//
// Example:
//
//	roller := device.Roller(0)
//	err := roller.Open(ctx)
func (d *Device) Roller(id int) *components.Roller {
	return components.NewRoller(d.transport, id)
}

// Light returns a Light component accessor.
//
// Parameters:
//   - id: Light index (0-based)
//
// Example:
//
//	light := device.Light(0)
//	err := light.SetBrightness(ctx, 50)
func (d *Device) Light(id int) *components.Light {
	return components.NewLight(d.transport, id)
}

// Color returns a Color (RGBW) component accessor.
//
// Parameters:
//   - id: Color index (0-based)
//
// Example:
//
//	color := device.Color(0)
//	err := color.SetRGB(ctx, 255, 0, 0)  // Red
func (d *Device) Color(id int) *components.Color {
	return components.NewColor(d.transport, id)
}

// White returns a White channel component accessor.
//
// Parameters:
//   - id: White channel index (0-based)
//
// Example:
//
//	white := device.White(0)
//	err := white.SetBrightness(ctx, 75)
func (d *Device) White(id int) *components.White {
	return components.NewWhite(d.transport, id)
}

// Meter returns a power Meter component accessor.
//
// Parameters:
//   - id: Meter index (0-based)
//
// Example:
//
//	meter := device.Meter(0)
//	status, err := meter.GetStatus(ctx)
func (d *Device) Meter(id int) *components.Meter {
	return components.NewMeter(d.transport, id)
}

// EMeter returns an Energy Meter component accessor.
//
// Parameters:
//   - id: EMeter index (0-based)
//
// Example:
//
//	emeter := device.EMeter(0)
//	status, err := emeter.GetStatus(ctx)
func (d *Device) EMeter(id int) *components.EMeter {
	return components.NewEMeter(d.transport, id)
}

// Input returns an Input component accessor.
//
// Parameters:
//   - id: Input index (0-based)
//
// Example:
//
//	input := device.Input(0)
//	status, err := input.GetStatus(ctx)
func (d *Device) Input(id int) *components.Input {
	return components.NewInput(d.transport, id)
}

// FactoryReset performs a factory reset on the device.
//
// This calls the /reset endpoint. All settings will be reset
// to factory defaults and the device will restart.
func (d *Device) FactoryReset(ctx context.Context) error {
	_, err := d.restCall(ctx, "/reset")
	if err != nil {
		return fmt.Errorf("failed to factory reset: %w", err)
	}
	return nil
}

// CheckForUpdate checks if a firmware update is available.
//
// This calls the /ota/check endpoint to check for updates.
func (d *Device) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	resp, err := d.restCall(ctx, "/ota/check")
	if err != nil {
		return nil, fmt.Errorf("failed to check for update: %w", err)
	}

	var info UpdateInfo
	if err := json.Unmarshal(resp, &info); err != nil {
		return nil, fmt.Errorf("failed to parse update info: %w", err)
	}

	return &info, nil
}

// Update starts a firmware update.
//
// If firmwareURL is empty, updates to the latest stable firmware.
// If firmwareURL is provided, updates to firmware at that URL.
func (d *Device) Update(ctx context.Context, firmwareURL string) error {
	endpoint := "/ota?update=true"
	if firmwareURL != "" {
		endpoint = "/ota?url=" + url.QueryEscape(firmwareURL)
	}

	_, err := d.restCall(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to start update: %w", err)
	}
	return nil
}

// SetName sets the device name.
func (d *Device) SetName(ctx context.Context, name string) error {
	endpoint := "/settings?name=" + url.QueryEscape(name)
	_, err := d.restCall(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to set name: %w", err)
	}
	return nil
}

// SetTimezone sets the device timezone.
func (d *Device) SetTimezone(ctx context.Context, timezone string) error {
	endpoint := "/settings?timezone=" + url.QueryEscape(timezone)
	_, err := d.restCall(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to set timezone: %w", err)
	}
	return nil
}

// SetLocation sets the device geographic location.
func (d *Device) SetLocation(ctx context.Context, lat, lng float64) error {
	endpoint := fmt.Sprintf("/settings?lat=%f&lng=%f", lat, lng)
	_, err := d.restCall(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to set location: %w", err)
	}
	return nil
}

// GetDebugLog retrieves device debug logs.
func (d *Device) GetDebugLog(ctx context.Context) (string, error) {
	resp, err := d.restCall(ctx, "/debug/log")
	if err != nil {
		return "", fmt.Errorf("failed to get debug log: %w", err)
	}
	return string(resp), nil
}

// Transport returns the underlying transport.
//
// This can be used for advanced operations or custom API calls.
func (d *Device) Transport() transport.Transport {
	return d.transport
}

// Call executes a raw API call to the device.
//
// Parameters:
//   - path: The REST endpoint path (e.g., "/relay/0?turn=on")
//
// This method is useful for accessing endpoints not covered
// by the typed methods.
func (d *Device) Call(ctx context.Context, path string) (json.RawMessage, error) {
	return d.restCall(ctx, path)
}

// ClearCache clears the cached device info.
//
// The next call to GetDeviceInfo will fetch fresh data.
func (d *Device) ClearCache() {
	d.mu.Lock()
	d.info = nil
	d.mu.Unlock()
}
