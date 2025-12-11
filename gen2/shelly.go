package gen2

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Shelly provides access to device-level operations via the Shelly namespace.
//
// The Shelly namespace contains methods for device information, system
// operations, and global device configuration.
type Shelly struct {
	client *rpc.Client
}

// NewShelly creates a new Shelly namespace handler.
func NewShelly(client *rpc.Client) *Shelly {
	return &Shelly{
		client: client,
	}
}

// DeviceInfo contains information about the device.
type DeviceInfo struct {
	types.RawFields
	App             string `json:"app"`
	Profile         string `json:"profile,omitempty"`
	ID              string `json:"id"`
	Model           string `json:"model"`
	Batch           string `json:"batch,omitempty"`
	FirmwareID      string `json:"fw_id"`
	MAC             string `json:"mac"`
	AuthDomain      string `json:"auth_domain,omitempty"`
	Key             string `json:"key,omitempty"`
	Name            string `json:"name,omitempty"`
	FirmwareVersion string `json:"ver"`
	Gen             int    `json:"gen"`
	FWSize          int    `json:"fw_sbits,omitempty"`
	Slot            int    `json:"slot,omitempty"`
	Discoverable    bool   `json:"discoverable,omitempty"`
	AuthEnabled     bool   `json:"auth_en"`
}

// GetDeviceInfo retrieves device information.
//
// This method returns details about the device including model, firmware
// version, MAC address, and other identifying information.
//
// Example:
//
//	info, err := device.Shelly().GetDeviceInfo(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Device: %s (%s) - FW: %s\n", info.Name, info.Model, info.FirmwareVersion)
func (s *Shelly) GetDeviceInfo(ctx context.Context) (*DeviceInfo, error) {
	var info DeviceInfo
	err := s.client.CallResult(ctx, "Shelly.GetDeviceInfo", nil, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// GetStatus retrieves the full device status including all components.
//
// This method returns the status of all components in the device.
// For individual component status, use the component's GetStatus method.
//
// Returns a map where keys are component keys (e.g., "switch:0", "sys")
// and values are the component status objects.
func (s *Shelly) GetStatus(ctx context.Context) (map[string]json.RawMessage, error) {
	result, err := s.client.Call(ctx, "Shelly.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status map[string]json.RawMessage
	if err := json.Unmarshal(result, &status); err != nil {
		return nil, err
	}

	return status, nil
}

// GetConfig retrieves the full device configuration.
//
// Returns a map where keys are component keys and values are the
// component configuration objects.
func (s *Shelly) GetConfig(ctx context.Context) (map[string]json.RawMessage, error) {
	result, err := s.client.Call(ctx, "Shelly.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, err
	}

	return config, nil
}

// SetConfig updates device configuration.
//
// The config parameter should be a map of component keys to configuration
// objects. Only specified components will be updated.
func (s *Shelly) SetConfig(ctx context.Context, config any) error {
	_, err := s.client.Call(ctx, "Shelly.SetConfig", config)
	return err
}

// ListMethods returns a list of all available RPC methods.
//
// This is useful for discovering what methods are available on the device.
func (s *Shelly) ListMethods(ctx context.Context) ([]string, error) {
	result, err := s.client.Call(ctx, "Shelly.ListMethods", nil)
	if err != nil {
		return nil, err
	}

	var methods struct {
		Methods []string `json:"methods"`
	}
	if err := json.Unmarshal(result, &methods); err != nil {
		return nil, err
	}

	return methods.Methods, nil
}

// Reboot reboots the device.
//
// The device will restart and disconnect. Wait a few seconds before
// attempting to reconnect.
//
// Parameters:
//   - delayMS: Optional delay in milliseconds before reboot (0-60000)
func (s *Shelly) Reboot(ctx context.Context, delayMS int) error {
	params := map[string]any{}
	if delayMS > 0 {
		params["delay_ms"] = delayMS
	}

	_, err := s.client.Call(ctx, "Shelly.Reboot", params)
	return err
}

// FactoryReset resets the device to factory defaults.
//
// WARNING: This will erase all configuration and data!
//
// The device will reboot after the reset.
func (s *Shelly) FactoryReset(ctx context.Context) error {
	_, err := s.client.Call(ctx, "Shelly.FactoryReset", nil)
	return err
}

// ResetWiFiConfig resets WiFi configuration to default (AP mode).
//
// The device will reboot and start in AP mode for reconfiguration.
func (s *Shelly) ResetWiFiConfig(ctx context.Context) error {
	_, err := s.client.Call(ctx, "Shelly.ResetWiFiConfig", nil)
	return err
}

// UpdateInfo contains information about available firmware updates.
type UpdateInfo struct {
	Stable *FirmwareVersion `json:"stable,omitempty"`
	Beta   *FirmwareVersion `json:"beta,omitempty"`
	types.RawFields
	OldVersion string `json:"old_version,omitempty"`
}

// FirmwareVersion contains firmware version information.
type FirmwareVersion struct {
	types.RawFields
	Version string `json:"version"`
	BuildID string `json:"build_id,omitempty"`
}

// CheckForUpdate checks if a firmware update is available.
//
// Returns information about available updates (stable and beta channels).
func (s *Shelly) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	result, err := s.client.Call(ctx, "Shelly.CheckForUpdate", nil)
	if err != nil {
		return nil, err
	}

	var updateInfo UpdateInfo
	if err := json.Unmarshal(result, &updateInfo); err != nil {
		return nil, err
	}

	return &updateInfo, nil
}

// UpdateParams contains parameters for firmware update.
type UpdateParams struct {
	// Stage is the update channel ("stable" or "beta")
	Stage string `json:"stage,omitempty"`

	// URL is a custom firmware URL (overrides stage)
	URL string `json:"url,omitempty"`
}

// Update initiates a firmware update.
//
// The device will download and install the specified firmware, then reboot.
// Progress can be monitored via Shelly.GetStatus or notifications.
//
// Parameters:
//   - params: Update parameters (stage or URL)
//
// Example:
//
//	err := device.Shelly().Update(ctx, &UpdateParams{Stage: "stable"})
func (s *Shelly) Update(ctx context.Context, params *UpdateParams) error {
	if params == nil {
		params = &UpdateParams{Stage: "stable"}
	}

	_, err := s.client.Call(ctx, "Shelly.Update", params)
	return err
}

// SetAuthParams contains parameters for setting authentication.
type SetAuthParams struct {
	// User is the username
	User string `json:"user,omitempty"`

	// Realm is the authentication realm
	Realm string `json:"realm,omitempty"`

	// HA1 is the pre-calculated HA1 hash (username:realm:password)
	HA1 string `json:"ha1,omitempty"`
}

// SetAuth configures device authentication.
//
// This sets the username and password for the device. After calling this,
// all subsequent requests must include authentication.
//
// Note: The HA1 parameter should be calculated as:
// MD5(username:realm:password)
func (s *Shelly) SetAuth(ctx context.Context, params *SetAuthParams) error {
	_, err := s.client.Call(ctx, "Shelly.SetAuth", params)
	return err
}

// GetComponents retrieves information about all device components.
//
// This returns a list of all components including their types, IDs,
// and optionally their current status and configuration.
//
// Parameters:
//   - includeStatus: Include component status in the response
//   - includeConfig: Include component configuration in the response
func (s *Shelly) GetComponents(ctx context.Context, includeStatus, includeConfig bool) (*ComponentList, error) {
	params := map[string]any{
		"status": includeStatus,
		"config": includeConfig,
	}

	result, err := s.client.Call(ctx, "Shelly.GetComponents", params)
	if err != nil {
		return nil, err
	}

	var components ComponentList
	if err := json.Unmarshal(result, &components); err != nil {
		return nil, err
	}

	return &components, nil
}

// DetectLocationResult contains the result of location detection.
type DetectLocationResult struct {
	types.RawFields
	TZ  string  `json:"tz,omitempty"`
	Lat float64 `json:"lat,omitempty"`
	Lon float64 `json:"lon,omitempty"`
}

// DetectLocation attempts to detect the device's geographic location.
//
// This uses IP geolocation to determine the device's location and timezone.
// Useful for automatic timezone configuration.
func (s *Shelly) DetectLocation(ctx context.Context) (*DetectLocationResult, error) {
	result, err := s.client.Call(ctx, "Shelly.DetectLocation", nil)
	if err != nil {
		return nil, err
	}

	var location DetectLocationResult
	if err := json.Unmarshal(result, &location); err != nil {
		return nil, err
	}

	return &location, nil
}

// ListProfiles returns a list of available device profiles.
//
// Profiles are device-specific and may not be available on all devices.
// For example, Shelly Plus 2PM supports "switch" and "cover" profiles.
func (s *Shelly) ListProfiles(ctx context.Context) ([]string, error) {
	result, err := s.client.Call(ctx, "Shelly.ListProfiles", nil)
	if err != nil {
		return nil, err
	}

	var profiles struct {
		Profiles []string `json:"profiles"`
	}
	if err := json.Unmarshal(result, &profiles); err != nil {
		return nil, err
	}

	return profiles.Profiles, nil
}

// SetProfile changes the device profile.
//
// This requires a device reboot to take effect. The device will reboot
// automatically after the profile is set.
//
// WARNING: Changing the profile may reset device configuration!
func (s *Shelly) SetProfile(ctx context.Context, name string) error {
	params := map[string]any{
		"name": name,
	}

	_, err := s.client.Call(ctx, "Shelly.SetProfile", params)
	return err
}

// PutUserCA uploads a custom CA certificate.
//
// Parameters:
//   - data: PEM-encoded CA certificate
//   - appendCA: If true, append to existing CAs; if false, replace
func (s *Shelly) PutUserCA(ctx context.Context, data string, appendCA bool) error {
	params := map[string]any{
		"data":   data,
		"append": appendCA,
	}

	_, err := s.client.Call(ctx, "Shelly.PutUserCA", params)
	return err
}

// PutTLSClientCert uploads a client TLS certificate.
//
// Parameters:
//   - data: PEM-encoded client certificate
func (s *Shelly) PutTLSClientCert(ctx context.Context, data string) error {
	params := map[string]any{
		"data": data,
	}

	_, err := s.client.Call(ctx, "Shelly.PutTLSClientCert", params)
	return err
}

// PutTLSClientKey uploads a client TLS private key.
//
// Parameters:
//   - data: PEM-encoded private key
func (s *Shelly) PutTLSClientKey(ctx context.Context, data string) error {
	params := map[string]any{
		"data": data,
	}

	_, err := s.client.Call(ctx, "Shelly.PutTLSClientKey", params)
	return err
}
