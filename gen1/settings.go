package gen1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// WiFi settings methods

// SetWiFiStation configures WiFi station settings.
//
// Parameters:
//   - enabled: Enable/disable WiFi station mode
//   - ssid: Network name to connect to
//   - password: Network password
//
// Example:
//
//	err := device.SetWiFiStation(ctx, true, "MyNetwork", "password123")
func (d *Device) SetWiFiStation(ctx context.Context, enabled bool, ssid, password string) error {
	params := url.Values{}
	params.Set("wifi_sta_enabled", boolToString(enabled))
	if ssid != "" {
		params.Set("wifi_sta_ssid", ssid)
	}
	if password != "" {
		params.Set("wifi_sta_key", password)
	}

	endpoint := "/settings?" + params.Encode()
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set WiFi station: %w", err)
	}
	return nil
}

// SetWiFiStationStatic configures WiFi station with static IP.
//
// Parameters:
//   - ssid: Network name
//   - password: Network password
//   - ip: Static IP address
//   - gateway: Gateway address
//   - mask: Subnet mask
//   - dns: DNS server (optional, empty string to skip)
func (d *Device) SetWiFiStationStatic(ctx context.Context, ssid, password, ip, gateway, mask, dns string) error {
	params := url.Values{}
	params.Set("wifi_sta_enabled", "true")
	params.Set("wifi_sta_ssid", ssid)
	params.Set("wifi_sta_key", password)
	params.Set("wifi_sta_ipv4_method", "static")
	params.Set("wifi_sta_ip", ip)
	params.Set("wifi_sta_gw", gateway)
	params.Set("wifi_sta_mask", mask)
	if dns != "" {
		params.Set("wifi_sta_dns", dns)
	}

	endpoint := "/settings?" + params.Encode()
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set WiFi station static: %w", err)
	}
	return nil
}

// SetWiFiAP configures WiFi access point settings.
//
// Parameters:
//   - enabled: Enable/disable AP mode
//   - ssid: AP network name (optional, empty to keep current)
//   - password: AP password (optional, empty to keep current)
func (d *Device) SetWiFiAP(ctx context.Context, enabled bool, ssid, password string) error {
	params := url.Values{}
	params.Set("wifi_ap_enabled", boolToString(enabled))
	if ssid != "" {
		params.Set("wifi_ap_ssid", ssid)
	}
	if password != "" {
		params.Set("wifi_ap_key", password)
	}

	endpoint := "/settings?" + params.Encode()
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set WiFi AP: %w", err)
	}
	return nil
}

// MQTT settings methods

// SetMQTT configures MQTT broker connection.
//
// Parameters:
//   - enabled: Enable/disable MQTT
//   - server: MQTT broker address (e.g., "192.168.1.100:1883")
//   - user: MQTT username (optional)
//   - password: MQTT password (optional)
func (d *Device) SetMQTT(ctx context.Context, enabled bool, server, user, password string) error {
	params := url.Values{}
	params.Set("mqtt_enable", boolToString(enabled))
	if server != "" {
		params.Set("mqtt_server", server)
	}
	if user != "" {
		params.Set("mqtt_user", user)
	}
	if password != "" {
		params.Set("mqtt_pass", password)
	}

	endpoint := "/settings?" + params.Encode()
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set MQTT: %w", err)
	}
	return nil
}

// MQTTConfig contains MQTT configuration options.
type MQTTConfig struct {
	Server              string
	User                string
	Password            string
	ID                  string
	ReconnectTimeoutMax float64
	ReconnectTimeoutMin float64
	KeepAlive           int
	MaxQos              int
	UpdatePeriod        int
	Enable              bool
	CleanSession        bool
	Retain              bool
}

// SetMQTTConfig configures MQTT with all options.
func (d *Device) SetMQTTConfig(ctx context.Context, config *MQTTConfig) error {
	params := url.Values{}
	params.Set("mqtt_enable", boolToString(config.Enable))

	if config.Server != "" {
		params.Set("mqtt_server", config.Server)
	}
	if config.User != "" {
		params.Set("mqtt_user", config.User)
	}
	if config.Password != "" {
		params.Set("mqtt_pass", config.Password)
	}
	if config.ID != "" {
		params.Set("mqtt_id", config.ID)
	}
	if config.ReconnectTimeoutMax > 0 {
		params.Set("mqtt_reconnect_timeout_max", strconv.FormatFloat(config.ReconnectTimeoutMax, 'f', -1, 64))
	}
	if config.ReconnectTimeoutMin > 0 {
		params.Set("mqtt_reconnect_timeout_min", strconv.FormatFloat(config.ReconnectTimeoutMin, 'f', -1, 64))
	}
	params.Set("mqtt_clean_session", boolToString(config.CleanSession))
	if config.KeepAlive > 0 {
		params.Set("mqtt_keep_alive", strconv.Itoa(config.KeepAlive))
	}
	if config.MaxQos >= 0 && config.MaxQos <= 2 {
		params.Set("mqtt_max_qos", strconv.Itoa(config.MaxQos))
	}
	params.Set("mqtt_retain", boolToString(config.Retain))
	if config.UpdatePeriod > 0 {
		params.Set("mqtt_update_period", strconv.Itoa(config.UpdatePeriod))
	}

	endpoint := "/settings?" + params.Encode()
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set MQTT config: %w", err)
	}
	return nil
}

// CoIoT settings methods

// SetCoIoT configures CoIoT protocol settings.
//
// Parameters:
//   - enabled: Enable/disable CoIoT
//   - updatePeriod: Status update period in seconds (0 to disable periodic updates)
//   - peer: CoIoT peer address (optional, empty for multicast)
func (d *Device) SetCoIoT(ctx context.Context, enabled bool, updatePeriod int, peer string) error {
	params := url.Values{}
	params.Set("coiot_enable", boolToString(enabled))
	if updatePeriod > 0 {
		params.Set("coiot_update_period", strconv.Itoa(updatePeriod))
	}
	if peer != "" {
		params.Set("coiot_peer", peer)
	}

	endpoint := "/settings?" + params.Encode()
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set CoIoT: %w", err)
	}
	return nil
}

// Cloud settings methods

// SetCloud enables or disables Shelly cloud connection.
func (d *Device) SetCloud(ctx context.Context, enabled bool) error {
	endpoint := fmt.Sprintf("/settings/cloud?enabled=%s", boolToString(enabled))
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set cloud: %w", err)
	}
	return nil
}

// Authentication settings methods

// SetAuth enables or disables authentication.
//
// Parameters:
//   - enabled: Enable/disable authentication
//   - username: Username (typically "admin")
//   - password: Password
func (d *Device) SetAuth(ctx context.Context, enabled bool, username, password string) error {
	params := url.Values{}
	params.Set("enabled", boolToString(enabled))
	if username != "" {
		params.Set("username", username)
	}
	if password != "" {
		params.Set("password", password)
	}

	endpoint := "/settings/login?" + params.Encode()
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set auth: %w", err)
	}
	return nil
}

// Time settings methods

// SetTimeServer sets the NTP server for time synchronization.
func (d *Device) SetTimeServer(ctx context.Context, server string) error {
	endpoint := fmt.Sprintf("/settings?sntp_server=%s", url.QueryEscape(server))
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set time server: %w", err)
	}
	return nil
}

// Action URLs configuration

// ActionType represents the type of action that can trigger a URL.
type ActionType string

const (
	// ActionOutput represents output state change actions.
	ActionOutput ActionType = "out"
	// ActionInput represents input state change actions.
	ActionInput ActionType = "inp"
	// ActionButton represents button press actions.
	ActionButton ActionType = "btn"
	// ActionSensor represents sensor trigger actions.
	ActionSensor ActionType = "sensor"
	// ActionReport represents periodic report actions.
	ActionReport ActionType = "report"
)

// ActionEvent represents specific events that trigger actions.
type ActionEvent string

const (
	// Output events
	ActionOutputOn        ActionEvent = "out_on"
	ActionOutputOff       ActionEvent = "out_off"
	ActionOutputOnUrl     ActionEvent = "out_on_url"
	ActionOutputOffUrl    ActionEvent = "out_off_url"
	ActionRollerOpen      ActionEvent = "roller_open"
	ActionRollerClose     ActionEvent = "roller_close"
	ActionRollerStop      ActionEvent = "roller_stop"
	ActionRollerOpenUrl   ActionEvent = "roller_open_url"
	ActionRollerCloseUrl  ActionEvent = "roller_close_url"
	ActionRollerStopUrl   ActionEvent = "roller_stop_url"
	ActionRollerAtPos     ActionEvent = "roller_at_pos"
	ActionLongpush        ActionEvent = "longpush_url"
	ActionShortpush       ActionEvent = "shortpush_url"
	ActionDoublepush      ActionEvent = "double_shortpush_url"
	ActionTriplepush      ActionEvent = "triple_shortpush_url"
	ActionBtn1On          ActionEvent = "btn1_on_url"
	ActionBtn1Off         ActionEvent = "btn1_off_url"
	ActionBtn2On          ActionEvent = "btn2_on_url"
	ActionBtn2Off         ActionEvent = "btn2_off_url"
	ActionInputOn         ActionEvent = "input_on_url"
	ActionInputOff        ActionEvent = "input_off_url"
	ActionSensorOpen      ActionEvent = "dark_url"
	ActionSensorClose     ActionEvent = "twilight_url"
	ActionSensorMotion    ActionEvent = "motion_url"
	ActionSensorNoMotion  ActionEvent = "no_motion_url"
	ActionSensorFlood     ActionEvent = "flood_detected_url"
	ActionSensorNoFlood   ActionEvent = "flood_gone_url"
	ActionSensorSmoke     ActionEvent = "smoke_url"
	ActionSensorNoSmoke   ActionEvent = "no_smoke_url"
	ActionSensorGas       ActionEvent = "gas_detected_url"
	ActionSensorNoGas     ActionEvent = "gas_gone_url"
	ActionSensorVibration ActionEvent = "vibration_url"
	ActionSensorTemp      ActionEvent = "temp_over_url"
	ActionSensorTempUnder ActionEvent = "temp_under_url"
	ActionReportUrl       ActionEvent = "report_url"
	ActionOverpower       ActionEvent = "overpower_url"
	ActionOvervoltage     ActionEvent = "overvoltage_url"
	ActionUndervoltage    ActionEvent = "undervoltage_url"
	ActionOvertemperature ActionEvent = "overtemperature_url"
)

// Action represents a configured action URL.
type Action struct {
	Name    string      `json:"name,omitempty"`
	Event   ActionEvent `json:"-"`
	URLs    []string    `json:"urls,omitempty"`
	Index   int         `json:"index,omitempty"`
	Enabled bool        `json:"enabled"`
}

// ActionSettings contains all action URL settings.
type ActionSettings struct {
	Actions []Action `json:"actions,omitempty"`
}

// GetActions retrieves all configured action URLs.
func (d *Device) GetActions(ctx context.Context) (*ActionSettings, error) {
	resp, err := d.transport.Call(ctx, "/settings/actions", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}

	var settings ActionSettings
	if err := json.Unmarshal(resp, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse actions: %w", err)
	}

	return &settings, nil
}

// SetAction configures an action URL for a specific event.
//
// Parameters:
//   - index: Component index (0 for relay 0, 1 for relay 1, etc.)
//   - event: The event that triggers the action
//   - urls: URLs to call when event occurs (can be multiple)
//   - enabled: Enable/disable the action
//
// Example:
//
//	err := device.SetAction(ctx, 0, gen1.ActionOutputOn, []string{"http://192.168.1.100/trigger"}, true)
func (d *Device) SetAction(ctx context.Context, index int, event ActionEvent, urls []string, enabled bool) error {
	params := url.Values{}
	params.Set("index", strconv.Itoa(index))
	params.Set("name", string(event))
	params.Set("enabled", boolToString(enabled))

	for i, u := range urls {
		params.Set(fmt.Sprintf("urls[%d]", i), u)
	}

	endpoint := "/settings/actions?" + params.Encode()
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set action: %w", err)
	}
	return nil
}

// SetActionURL is a convenience method to set a single action URL.
func (d *Device) SetActionURL(ctx context.Context, index int, event ActionEvent, actionURL string, enabled bool) error {
	return d.SetAction(ctx, index, event, []string{actionURL}, enabled)
}

// ClearAction disables and clears an action.
func (d *Device) ClearAction(ctx context.Context, index int, event ActionEvent) error {
	return d.SetAction(ctx, index, event, nil, false)
}

// Relay settings methods

// RelayConfig contains relay configuration options.
type RelayConfig struct {
	Name         string
	DefaultState string
	BtnType      string
	AutoOn       float64
	AutoOff      float64
	MaxPower     int
	BtnReverse   bool
	Schedule     bool
}

// SetRelayConfig configures a relay's settings.
func (d *Device) SetRelayConfig(ctx context.Context, id int, config *RelayConfig) error {
	params := url.Values{}
	if config.Name != "" {
		params.Set("name", config.Name)
	}
	if config.DefaultState != "" {
		params.Set("default_state", config.DefaultState)
	}
	if config.BtnType != "" {
		params.Set("btn_type", config.BtnType)
	}
	params.Set("btn_reverse", boolToString(config.BtnReverse))
	if config.AutoOn > 0 {
		params.Set("auto_on", strconv.FormatFloat(config.AutoOn, 'f', -1, 64))
	}
	if config.AutoOff > 0 {
		params.Set("auto_off", strconv.FormatFloat(config.AutoOff, 'f', -1, 64))
	}
	if config.MaxPower > 0 {
		params.Set("max_power", strconv.Itoa(config.MaxPower))
	}
	params.Set("schedule", boolToString(config.Schedule))

	endpoint := fmt.Sprintf("/settings/relay/%d?%s", id, params.Encode())
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set relay config: %w", err)
	}
	return nil
}

// GetRelaySettings gets settings for a specific relay.
func (d *Device) GetRelaySettings(ctx context.Context, id int) (*RelaySettings, error) {
	resp, err := d.transport.Call(ctx, fmt.Sprintf("/settings/relay/%d", id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get relay settings: %w", err)
	}

	var settings RelaySettings
	if err := json.Unmarshal(resp, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse relay settings: %w", err)
	}

	return &settings, nil
}

// Roller settings methods

// RollerConfig contains roller configuration options.
type RollerConfig struct {
	SafetyAction   string
	ObstacleAction string
	DefaultState   string
	SafetyMode     string
	InputMode      string
	ObstacleMode   string
	BtnType        string
	ObstacleDelay  int
	MaxTimeClose   float64
	MaxTimeOpen    float64
	ObstaclePower  int
	Swap           bool
	BtnReverse     bool
	SwapInputs     bool
	Positioning    bool
}

// SetRollerConfig configures a roller's settings.
func (d *Device) SetRollerConfig(ctx context.Context, id int, config *RollerConfig) error {
	params := url.Values{}
	if config.MaxTimeOpen > 0 {
		params.Set("maxtime_open", strconv.FormatFloat(config.MaxTimeOpen, 'f', -1, 64))
	}
	if config.MaxTimeClose > 0 {
		params.Set("maxtime_close", strconv.FormatFloat(config.MaxTimeClose, 'f', -1, 64))
	}
	if config.DefaultState != "" {
		params.Set("default_state", config.DefaultState)
	}
	params.Set("swap_inputs", boolToString(config.SwapInputs))
	params.Set("swap", boolToString(config.Swap))
	if config.InputMode != "" {
		params.Set("input_mode", config.InputMode)
	}
	if config.BtnType != "" {
		params.Set("btn_type", config.BtnType)
	}
	params.Set("btn_reverse", boolToString(config.BtnReverse))
	if config.SafetyMode != "" {
		params.Set("safety_mode", config.SafetyMode)
	}
	if config.SafetyAction != "" {
		params.Set("safety_action", config.SafetyAction)
	}
	if config.ObstacleMode != "" {
		params.Set("obstacle_mode", config.ObstacleMode)
	}
	if config.ObstacleAction != "" {
		params.Set("obstacle_action", config.ObstacleAction)
	}
	if config.ObstaclePower > 0 {
		params.Set("obstacle_power", strconv.Itoa(config.ObstaclePower))
	}
	if config.ObstacleDelay > 0 {
		params.Set("obstacle_delay", strconv.Itoa(config.ObstacleDelay))
	}
	params.Set("positioning", boolToString(config.Positioning))

	endpoint := fmt.Sprintf("/settings/roller/%d?%s", id, params.Encode())
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set roller config: %w", err)
	}
	return nil
}

// GetRollerSettings gets settings for a specific roller.
func (d *Device) GetRollerSettings(ctx context.Context, id int) (*RollerSettings, error) {
	resp, err := d.transport.Call(ctx, fmt.Sprintf("/settings/roller/%d", id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get roller settings: %w", err)
	}

	var settings RollerSettings
	if err := json.Unmarshal(resp, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse roller settings: %w", err)
	}

	return &settings, nil
}

// Light settings methods

// LightConfig contains light configuration options.
type LightConfig struct {
	Name         string
	DefaultState string
	BtnType      string
	AutoOn       float64
	AutoOff      float64
	BtnReverse   bool
	Schedule     bool
}

// SetLightConfig configures a light's settings.
func (d *Device) SetLightConfig(ctx context.Context, id int, config LightConfig) error {
	params := url.Values{}
	if config.Name != "" {
		params.Set("name", config.Name)
	}
	if config.DefaultState != "" {
		params.Set("default_state", config.DefaultState)
	}
	if config.AutoOn > 0 {
		params.Set("auto_on", strconv.FormatFloat(config.AutoOn, 'f', -1, 64))
	}
	if config.AutoOff > 0 {
		params.Set("auto_off", strconv.FormatFloat(config.AutoOff, 'f', -1, 64))
	}
	if config.BtnType != "" {
		params.Set("btn_type", config.BtnType)
	}
	params.Set("btn_reverse", boolToString(config.BtnReverse))
	params.Set("schedule", boolToString(config.Schedule))

	endpoint := fmt.Sprintf("/settings/light/%d?%s", id, params.Encode())
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set light config: %w", err)
	}
	return nil
}

// GetLightSettings gets settings for a specific light.
func (d *Device) GetLightSettings(ctx context.Context, id int) (*LightSettings, error) {
	resp, err := d.transport.Call(ctx, fmt.Sprintf("/settings/light/%d", id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get light settings: %w", err)
	}

	var settings LightSettings
	if err := json.Unmarshal(resp, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse light settings: %w", err)
	}

	return &settings, nil
}

// CoIoT description/status methods

// HTTPCoIoTDescription contains the CoIoT device description from HTTP endpoint.
// Note: CoIoT types are also defined in coiot.go for the UDP multicast interface.
type HTTPCoIoTDescription struct {
	Blk []HTTPCoIoTBlock  `json:"blk,omitempty"`
	Sen []HTTPCoIoTSensor `json:"sen,omitempty"`
}

// HTTPCoIoTBlock represents a CoIoT block (component group).
type HTTPCoIoTBlock struct {
	D string `json:"D"`
	I int    `json:"I"`
}

// HTTPCoIoTSensor represents a CoIoT sensor description.
type HTTPCoIoTSensor struct {
	T string `json:"T"`
	D string `json:"D"`
	R string `json:"R"`
	I int    `json:"I"`
	L int    `json:"L"`
}

// GetCoIoTDescription retrieves the CoIoT device description.
//
// This returns the device's CoIoT protocol capabilities including
// blocks (component groups) and sensors (data points).
func (d *Device) GetCoIoTDescription(ctx context.Context) (*HTTPCoIoTDescription, error) {
	resp, err := d.transport.Call(ctx, "/cit/d", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get CoIoT description: %w", err)
	}

	var desc HTTPCoIoTDescription
	if err := json.Unmarshal(resp, &desc); err != nil {
		return nil, fmt.Errorf("failed to parse CoIoT description: %w", err)
	}

	return &desc, nil
}

// HTTPCoIoTStatusValues contains the CoIoT status values from HTTP endpoint.
type HTTPCoIoTStatusValues struct {
	G [][]any `json:"G,omitempty"` // [[channel, sensor_id, value], ...]
}

// GetCoIoTStatusValues retrieves the current CoIoT status.
//
// This returns the current values of all CoIoT sensors.
// Each value is a triplet: [channel, sensor_id, value].
func (d *Device) GetCoIoTStatusValues(ctx context.Context) (*HTTPCoIoTStatusValues, error) {
	resp, err := d.transport.Call(ctx, "/cit/s", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get CoIoT status: %w", err)
	}

	var status HTTPCoIoTStatusValues
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to parse CoIoT status: %w", err)
	}

	return &status, nil
}

// Temperature/Humidity sensor methods

// TemperatureStatus contains temperature sensor reading.
type TemperatureStatus struct {
	TC      float64 `json:"tC,omitempty"`       // Temperature in Celsius
	TF      float64 `json:"tF,omitempty"`       // Temperature in Fahrenheit
	IsValid bool    `json:"is_valid,omitempty"` // Reading validity
}

// GetTemperature retrieves the temperature sensor reading.
//
// This calls the /temperature endpoint available on sensor devices.
func (d *Device) GetTemperature(ctx context.Context) (*TemperatureStatus, error) {
	resp, err := d.transport.Call(ctx, "/temperature", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get temperature: %w", err)
	}

	var temp TemperatureStatus
	if err := json.Unmarshal(resp, &temp); err != nil {
		return nil, fmt.Errorf("failed to parse temperature: %w", err)
	}

	return &temp, nil
}

// HumidityStatus contains humidity sensor reading.
type HumidityStatus struct {
	Value   float64 `json:"value,omitempty"`    // Humidity percentage
	IsValid bool    `json:"is_valid,omitempty"` // Reading validity
}

// GetHumidity retrieves the humidity sensor reading.
//
// This calls the /humidity endpoint available on H&T devices.
func (d *Device) GetHumidity(ctx context.Context) (*HumidityStatus, error) {
	resp, err := d.transport.Call(ctx, "/humidity", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get humidity: %w", err)
	}

	var hum HumidityStatus
	if err := json.Unmarshal(resp, &hum); err != nil {
		return nil, fmt.Errorf("failed to parse humidity: %w", err)
	}

	return &hum, nil
}

// ExternalSensorStatus contains external sensor reading.
type ExternalSensorStatus struct {
	TC       float64 `json:"tC,omitempty"`       // Temperature in Celsius
	TF       float64 `json:"tF,omitempty"`       // Temperature in Fahrenheit
	Humidity float64 `json:"hum,omitempty"`      // Humidity percentage (if available)
	IsValid  bool    `json:"is_valid,omitempty"` // Reading validity
}

// GetExternalSensor retrieves external temperature sensor reading.
//
// This calls the /sensor/temperature endpoint for external sensors.
func (d *Device) GetExternalSensor(ctx context.Context) (*ExternalSensorStatus, error) {
	resp, err := d.transport.Call(ctx, "/sensor/temperature", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get external sensor: %w", err)
	}

	var sensor ExternalSensorStatus
	if err := json.Unmarshal(resp, &sensor); err != nil {
		return nil, fmt.Errorf("failed to parse external sensor: %w", err)
	}

	return &sensor, nil
}

// Schedule rule methods

// AddScheduleRule adds a schedule rule to a relay.
//
// The rule format is: "HHMM-D-R" where:
//   - HHMM is the time in 24h format
//   - D is day bits (0x7F for every day, 0x3E for weekdays, etc.)
//   - R is the relay index
//
// Example rules:
//   - "0800-7F-0-on" - Turn relay 0 on at 8:00 every day
//   - "2200-1F-0-off" - Turn relay 0 off at 22:00 Mon-Fri
func (d *Device) AddScheduleRule(ctx context.Context, relayID int, rule string) error {
	endpoint := fmt.Sprintf("/settings/relay/%d?schedule_rules[]=%s", relayID, url.QueryEscape(rule))
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to add schedule rule: %w", err)
	}
	return nil
}

// SetScheduleRules replaces all schedule rules for a relay.
func (d *Device) SetScheduleRules(ctx context.Context, relayID int, rules []string) error {
	params := url.Values{}
	for i, rule := range rules {
		params.Set(fmt.Sprintf("schedule_rules[%d]", i), rule)
	}

	endpoint := fmt.Sprintf("/settings/relay/%d?%s", relayID, params.Encode())
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set schedule rules: %w", err)
	}
	return nil
}

// EnableSchedule enables or disables the schedule for a relay.
func (d *Device) EnableSchedule(ctx context.Context, relayID int, enabled bool) error {
	endpoint := fmt.Sprintf("/settings/relay/%d?schedule=%s", relayID, boolToString(enabled))
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set schedule enabled: %w", err)
	}
	return nil
}

// Device mode methods

// SetMode sets the device operating mode.
//
// Valid modes depend on device type:
//   - "relay" - Standard relay mode
//   - "roller" - Roller/shutter mode (for 2.5)
//   - "color" - Color mode (for RGBW devices)
//   - "white" - White mode (for RGBW devices)
func (d *Device) SetMode(ctx context.Context, mode string) error {
	endpoint := fmt.Sprintf("/settings?mode=%s", url.QueryEscape(mode))
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}
	return nil
}

// SetDiscoverable sets whether the device is discoverable.
func (d *Device) SetDiscoverable(ctx context.Context, discoverable bool) error {
	endpoint := fmt.Sprintf("/settings?discoverable=%s", boolToString(discoverable))
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set discoverable: %w", err)
	}
	return nil
}

// SetMaxPower sets the maximum power limit for the device.
func (d *Device) SetMaxPower(ctx context.Context, maxPower int) error {
	endpoint := fmt.Sprintf("/settings?max_power=%d", maxPower)
	_, err := d.transport.Call(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to set max power: %w", err)
	}
	return nil
}

// WiFi scan methods

// WiFiNetwork represents a WiFi network found during scan.
type WiFiNetwork struct {
	SSID    string `json:"ssid"`
	BSSID   string `json:"bssid"`
	RSSI    int    `json:"rssi"`
	Channel int    `json:"channel"`
	Auth    int    `json:"auth"`
}

// ScanWiFi scans for available WiFi networks.
func (d *Device) ScanWiFi(ctx context.Context) ([]WiFiNetwork, error) {
	resp, err := d.transport.Call(ctx, "/wifiscan", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to scan WiFi: %w", err)
	}

	// Response may be wrapped in {"results": [...]}
	var wrapper struct {
		Results []WiFiNetwork `json:"results"`
	}
	if err := json.Unmarshal(resp, &wrapper); err != nil {
		// Try direct array format
		var networks []WiFiNetwork
		if err2 := json.Unmarshal(resp, &networks); err2 != nil {
			return nil, fmt.Errorf("failed to parse WiFi scan: %w", err)
		}
		return networks, nil
	}

	return wrapper.Results, nil
}

// Helper functions

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// BuildScheduleRule creates a schedule rule string.
//
// Parameters:
//   - hour: Hour (0-23)
//   - minute: Minute (0-59)
//   - days: Day bitmask (bit 0=Sun, bit 1=Mon, ... bit 6=Sat, 0x7F=every day)
//   - action: "on" or "off"
//
// Example:
//
//	rule := gen1.BuildScheduleRule(8, 0, 0x7F, "on")  // 8:00 every day, turn on
func BuildScheduleRule(hour, minute, days int, action string) string {
	return fmt.Sprintf("%02d%02d-%02X-0-%s", hour, minute, days, action)
}

// ParseScheduleRule parses a schedule rule string.
//
// Returns hour, minute, days bitmask, relay index, action.
//
//nolint:gocritic // tooManyResultsChecker: all return values are meaningful parts of the parsed rule
func ParseScheduleRule(rule string) (hour, minute, days, relay int, action string, err error) {
	parts := strings.Split(rule, "-")
	if len(parts) != 4 {
		return 0, 0, 0, 0, "", fmt.Errorf("invalid rule format")
	}

	if len(parts[0]) != 4 {
		return 0, 0, 0, 0, "", fmt.Errorf("invalid time format")
	}

	hour, err = strconv.Atoi(parts[0][:2])
	if err != nil {
		return 0, 0, 0, 0, "", fmt.Errorf("invalid hour: %w", err)
	}

	minute, err = strconv.Atoi(parts[0][2:])
	if err != nil {
		return 0, 0, 0, 0, "", fmt.Errorf("invalid minute: %w", err)
	}

	days64, err := strconv.ParseInt(parts[1], 16, 32)
	if err != nil {
		return 0, 0, 0, 0, "", fmt.Errorf("invalid days: %w", err)
	}
	days = int(days64)

	relay, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, 0, "", fmt.Errorf("invalid relay: %w", err)
	}

	action = parts[3]
	return hour, minute, days, relay, action, nil
}

// Day constants for schedule rules
const (
	DaySunday    = 1 << 0 // 0x01
	DayMonday    = 1 << 1 // 0x02
	DayTuesday   = 1 << 2 // 0x04
	DayWednesday = 1 << 3 // 0x08
	DayThursday  = 1 << 4 // 0x10
	DayFriday    = 1 << 5 // 0x20
	DaySaturday  = 1 << 6 // 0x40
	DayEveryDay  = 0x7F   // All days
	DayWeekdays  = DayMonday | DayTuesday | DayWednesday | DayThursday | DayFriday
	DayWeekends  = DaySaturday | DaySunday
)
