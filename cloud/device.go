package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Device control endpoints.
const (
	endpointDeviceAll         = "/device/all"
	endpointDeviceStatus      = "/device/status"
	endpointSwitchControl     = "/device/relay/control"
	endpointCoverControl      = "/device/roller/control"
	endpointLightControl      = "/device/light/control"
	endpointV2DevicesGet      = "/v2/devices/api/get"
	endpointV2DevicesSetGroup = "/v2/devices/api/set/groups"
)

// GetAllDevices returns all devices associated with the account.
func (c *Client) GetAllDevices(ctx context.Context) (map[string]*DeviceStatus, error) {
	// Make request
	respBody, err := c.doGet(ctx, endpointDeviceAll, nil)
	if err != nil {
		return nil, err
	}

	// Parse response
	var resp AllDevicesResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if !resp.IsOK {
		if len(resp.Errors) > 0 {
			return nil, fmt.Errorf("API error: %s", strings.Join(resp.Errors, ", "))
		}
		return nil, fmt.Errorf("API error: unknown error")
	}

	if resp.Data == nil {
		return nil, nil
	}

	return resp.Data.DevicesStatus, nil
}

// GetDeviceStatus returns the status of a specific device.
func (c *Client) GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error) {
	// Build query parameters
	params := url.Values{}
	params.Set("id", deviceID)

	// Make request
	respBody, err := c.doGet(ctx, endpointDeviceStatus, params)
	if err != nil {
		return nil, err
	}

	// Parse response
	var resp DeviceStatusResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if !resp.IsOK {
		return nil, c.parseDeviceError(resp.Errors)
	}

	if resp.Data == nil || resp.Data.DeviceStatus == nil {
		return nil, ErrDeviceNotFound
	}

	return resp.Data.DeviceStatus, nil
}

// GetDevicesV2 fetches device information using the v2 API.
// This can fetch up to 10 devices at once with optional status and settings.
func (c *Client) GetDevicesV2(
	ctx context.Context, deviceIDs []string, fetchStatus, fetchSettings bool,
) (*V2DevicesResponse, error) {
	// Build request
	req := V2DevicesRequest{
		IDs: deviceIDs,
	}

	// Add select options
	if fetchStatus {
		req.Select = append(req.Select, "status")
	}
	if fetchSettings {
		req.Select = append(req.Select, "settings")
	}

	// Make request
	respBody, err := c.doPost(ctx, endpointV2DevicesGet, req)
	if err != nil {
		return nil, err
	}

	// Parse response
	var resp V2DevicesResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if resp.Error != "" {
		return nil, fmt.Errorf("API error: %s", resp.Error)
	}

	return &resp, nil
}

// SetSwitch controls a switch (relay) device.
func (c *Client) SetSwitch(ctx context.Context, deviceID string, channel int, on bool) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	if on {
		params.Set("turn", "on")
	} else {
		params.Set("turn", "off")
	}

	_, err := c.doGet(ctx, endpointSwitchControl, params)
	return err
}

// ToggleSwitch toggles a switch (relay) device.
func (c *Client) ToggleSwitch(ctx context.Context, deviceID string, channel int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("turn", "toggle")

	_, err := c.doGet(ctx, endpointSwitchControl, params)
	return err
}

// SetSwitchWithTimer controls a switch with an auto-off timer.
func (c *Client) SetSwitchWithTimer(
	ctx context.Context, deviceID string, channel int, on bool, timerSeconds int,
) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	if on {
		params.Set("turn", "on")
	} else {
		params.Set("turn", "off")
	}
	params.Set("timer", fmt.Sprintf("%d", timerSeconds))

	_, err := c.doGet(ctx, endpointSwitchControl, params)
	return err
}

// OpenCover opens a cover/roller.
func (c *Client) OpenCover(ctx context.Context, deviceID string, channel int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("direction", "open")

	_, err := c.doGet(ctx, endpointCoverControl, params)
	return err
}

// CloseCover closes a cover/roller.
func (c *Client) CloseCover(ctx context.Context, deviceID string, channel int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("direction", "close")

	_, err := c.doGet(ctx, endpointCoverControl, params)
	return err
}

// StopCover stops a cover/roller.
func (c *Client) StopCover(ctx context.Context, deviceID string, channel int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("direction", "stop")

	_, err := c.doGet(ctx, endpointCoverControl, params)
	return err
}

// SetCoverPosition sets the cover position (0-100).
func (c *Client) SetCoverPosition(ctx context.Context, deviceID string, channel, position int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("pos", fmt.Sprintf("%d", position))

	_, err := c.doGet(ctx, endpointCoverControl, params)
	return err
}

// SetLight controls a light device.
func (c *Client) SetLight(ctx context.Context, deviceID string, channel int, on bool) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	if on {
		params.Set("turn", "on")
	} else {
		params.Set("turn", "off")
	}

	_, err := c.doGet(ctx, endpointLightControl, params)
	return err
}

// ToggleLight toggles a light device.
func (c *Client) ToggleLight(ctx context.Context, deviceID string, channel int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("turn", "toggle")

	_, err := c.doGet(ctx, endpointLightControl, params)
	return err
}

// SetLightBrightness sets the light brightness (0-100).
func (c *Client) SetLightBrightness(ctx context.Context, deviceID string, channel, brightness int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("turn", "on")
	params.Set("brightness", fmt.Sprintf("%d", brightness))

	_, err := c.doGet(ctx, endpointLightControl, params)
	return err
}

// SetLightRGB sets the light to a specific RGB color.
func (c *Client) SetLightRGB(ctx context.Context, deviceID string, channel, red, green, blue int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("turn", "on")
	params.Set("red", fmt.Sprintf("%d", red))
	params.Set("green", fmt.Sprintf("%d", green))
	params.Set("blue", fmt.Sprintf("%d", blue))

	_, err := c.doGet(ctx, endpointLightControl, params)
	return err
}

// SetLightRGBW sets the light to a specific RGBW color.
func (c *Client) SetLightRGBW(ctx context.Context, deviceID string, channel, red, green, blue, white int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("turn", "on")
	params.Set("red", fmt.Sprintf("%d", red))
	params.Set("green", fmt.Sprintf("%d", green))
	params.Set("blue", fmt.Sprintf("%d", blue))
	params.Set("white", fmt.Sprintf("%d", white))

	_, err := c.doGet(ctx, endpointLightControl, params)
	return err
}

// SetLightColorTemp sets the light color temperature (in Kelvin).
func (c *Client) SetLightColorTemp(ctx context.Context, deviceID string, channel, colorTemp int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("turn", "on")
	params.Set("color_temp", fmt.Sprintf("%d", colorTemp))

	_, err := c.doGet(ctx, endpointLightControl, params)
	return err
}

// SetLightEffect sets the light effect.
func (c *Client) SetLightEffect(ctx context.Context, deviceID string, channel, effect int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("turn", "on")
	params.Set("effect", fmt.Sprintf("%d", effect))

	_, err := c.doGet(ctx, endpointLightControl, params)
	return err
}

// SetLightGain sets the light gain (0-100).
func (c *Client) SetLightGain(ctx context.Context, deviceID string, channel, gain int) error {
	params := url.Values{}
	params.Set("id", deviceID)
	params.Set("channel", fmt.Sprintf("%d", channel))
	params.Set("turn", "on")
	params.Set("gain", fmt.Sprintf("%d", gain))

	_, err := c.doGet(ctx, endpointLightControl, params)
	return err
}

// GroupControl performs group control operations on multiple devices.
func (c *Client) GroupControl(ctx context.Context, req *GroupControlRequest) error {
	// Make request
	respBody, err := c.doPost(ctx, endpointV2DevicesSetGroup, req)
	if err != nil {
		return err
	}

	// Parse response
	var resp ControlResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if !resp.IsOK {
		if len(resp.Errors) > 0 {
			return fmt.Errorf("API error: %s", strings.Join(resp.Errors, ", "))
		}
		return fmt.Errorf("API error: unknown error")
	}

	return nil
}

// SetSwitchGroup turns on/off multiple switches at once.
func (c *Client) SetSwitchGroup(ctx context.Context, deviceChannels []string, on bool) error {
	turn := "off"
	if on {
		turn = "on"
	}

	req := &GroupControlRequest{
		Switches: []GroupSwitch{
			{IDs: deviceChannels, Turn: turn},
		},
	}

	return c.GroupControl(ctx, req)
}

// ToggleSwitchGroup toggles multiple switches at once.
func (c *Client) ToggleSwitchGroup(ctx context.Context, deviceChannels []string) error {
	req := &GroupControlRequest{
		Switches: []GroupSwitch{
			{IDs: deviceChannels, Turn: "toggle"},
		},
	}

	return c.GroupControl(ctx, req)
}

// SetCoverGroupPosition sets the position for multiple covers.
func (c *Client) SetCoverGroupPosition(ctx context.Context, deviceChannels []string, position int) error {
	pos := position
	req := &GroupControlRequest{
		Covers: []GroupCover{
			{IDs: deviceChannels, Position: &pos},
		},
	}

	return c.GroupControl(ctx, req)
}

// OpenCoverGroup opens multiple covers at once.
func (c *Client) OpenCoverGroup(ctx context.Context, deviceChannels []string) error {
	req := &GroupControlRequest{
		Covers: []GroupCover{
			{IDs: deviceChannels, Direction: "open"},
		},
	}

	return c.GroupControl(ctx, req)
}

// CloseCoverGroup closes multiple covers at once.
func (c *Client) CloseCoverGroup(ctx context.Context, deviceChannels []string) error {
	req := &GroupControlRequest{
		Covers: []GroupCover{
			{IDs: deviceChannels, Direction: "close"},
		},
	}

	return c.GroupControl(ctx, req)
}

// StopCoverGroup stops multiple covers at once.
func (c *Client) StopCoverGroup(ctx context.Context, deviceChannels []string) error {
	req := &GroupControlRequest{
		Covers: []GroupCover{
			{IDs: deviceChannels, Direction: "stop"},
		},
	}

	return c.GroupControl(ctx, req)
}

// SetLightGroupBrightness sets the brightness for multiple lights.
func (c *Client) SetLightGroupBrightness(ctx context.Context, deviceChannels []string, brightness int) error {
	b := brightness
	req := &GroupControlRequest{
		Lights: []GroupLight{
			{IDs: deviceChannels, Turn: "on", Brightness: &b},
		},
	}

	return c.GroupControl(ctx, req)
}

// SetLightGroup turns on/off multiple lights at once.
func (c *Client) SetLightGroup(ctx context.Context, deviceChannels []string, on bool) error {
	turn := "off"
	if on {
		turn = "on"
	}

	req := &GroupControlRequest{
		Lights: []GroupLight{
			{IDs: deviceChannels, Turn: turn},
		},
	}

	return c.GroupControl(ctx, req)
}

// ToggleLightGroup toggles multiple lights at once.
func (c *Client) ToggleLightGroup(ctx context.Context, deviceChannels []string) error {
	req := &GroupControlRequest{
		Lights: []GroupLight{
			{IDs: deviceChannels, Turn: "toggle"},
		},
	}

	return c.GroupControl(ctx, req)
}

// SetLightGroupRGB sets the RGB color for multiple lights.
func (c *Client) SetLightGroupRGB(ctx context.Context, deviceChannels []string, red, green, blue int) error {
	r, g, b := red, green, blue
	req := &GroupControlRequest{
		Lights: []GroupLight{
			{IDs: deviceChannels, Turn: "on", Red: &r, Green: &g, Blue: &b},
		},
	}

	return c.GroupControl(ctx, req)
}

// parseDeviceError converts API error strings into appropriate error types.
func (c *Client) parseDeviceError(errors []string) error {
	if len(errors) == 0 {
		return fmt.Errorf("API error: unknown error")
	}

	errMsg := strings.Join(errors, ", ")
	switch {
	case strings.Contains(errMsg, "not found"):
		return ErrDeviceNotFound
	case strings.Contains(errMsg, "offline"):
		return ErrDeviceOffline
	default:
		return fmt.Errorf("API error: %s", errMsg)
	}
}
