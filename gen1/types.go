package gen1

import (
	"encoding/json"

	"github.com/tj-smith47/shelly-go/types"
)

// DeviceInfo contains Gen1 device identification and capabilities.
// Returned by the /shelly endpoint.
type DeviceInfo struct {
	types.RawFields `json:"-"`
	Type            string `json:"type"`
	MAC             string `json:"mac"`
	FW              string `json:"fw"`
	LongID          int    `json:"longid,omitempty"`
	NumOutputs      int    `json:"num_outputs,omitempty"`
	NumMeters       int    `json:"num_meters,omitempty"`
	NumRollers      int    `json:"num_rollers,omitempty"`
	NumInputs       int    `json:"num_inputs,omitempty"`
	Auth            bool   `json:"auth"`
	Discoverable    bool   `json:"discoverable,omitempty"`
}

// ToTypesDeviceInfo converts Gen1 DeviceInfo to types.DeviceInfo.
func (d *DeviceInfo) ToTypesDeviceInfo() *types.DeviceInfo {
	return &types.DeviceInfo{
		ID:          d.MAC,
		Model:       d.Type,
		Version:     d.FW,
		Generation:  types.Gen1,
		AuthEnabled: d.Auth,
		MAC:         d.MAC,
	}
}

// Status contains the complete device status from /status endpoint.
type Status struct {
	Concentration   *ConcentrationData          `json:"concentration,omitempty"`
	ExtTemperature  map[string]*TemperatureData `json:"ext_temperature,omitempty"`
	MQTT            *MQTTStatus                 `json:"mqtt,omitempty"`
	types.RawFields `json:"-"`
	Update          *UpdateStatus              `json:"update,omitempty"`
	ExtSensors      map[string]json.RawMessage `json:"ext_sensors,omitempty"`
	ExtHumidity     map[string]*HumidityData   `json:"ext_humidity,omitempty"`
	WiFiSta         *WiFiStatus                `json:"wifi_sta,omitempty"`
	Gas             *GasStatus                 `json:"gas,omitempty"`
	ActionsStats    *ActionsStats              `json:"actions_stats,omitempty"`
	Accel           *AccelData                 `json:"accel,omitempty"`
	Lux             *LuxData                   `json:"lux,omitempty"`
	Humidity        *HumidityData              `json:"hum,omitempty"`
	Bat             *BatteryStatus             `json:"bat,omitempty"`
	Cloud           *CloudStatus               `json:"cloud,omitempty"`
	Tmp             *TemperatureData           `json:"tmp,omitempty"`
	MAC             string                     `json:"mac,omitempty"`
	Time            string                     `json:"time,omitempty"`
	Inputs          []InputStatus              `json:"inputs,omitempty"`
	EMeters         []EMeterStatus             `json:"emeters,omitempty"`
	Meters          []MeterStatus              `json:"meters,omitempty"`
	Lights          []LightStatus              `json:"lights,omitempty"`
	Rollers         []RollerStatus             `json:"rollers,omitempty"`
	Relays          []RelayStatus              `json:"relays,omitempty"`
	FSSize          int                        `json:"fs_size,omitempty"`
	RAMTotal        int                        `json:"ram_total,omitempty"`
	Temperature     float64                    `json:"temperature,omitempty"`
	Uptime          int                        `json:"uptime,omitempty"`
	Serial          int                        `json:"serial,omitempty"`
	FSFree          int                        `json:"fs_free,omitempty"`
	RAMFree         int                        `json:"ram_free,omitempty"`
	UnixTime        int64                      `json:"unixtime,omitempty"`
	CfgChangedCnt   int                        `json:"cfg_changed_cnt,omitempty"`
	HasUpdate       bool                       `json:"has_update,omitempty"`
	Smoke           bool                       `json:"smoke,omitempty"`
	Charger         bool                       `json:"charger,omitempty"`
	Vibration       bool                       `json:"vibration,omitempty"`
	Flood           bool                       `json:"flood,omitempty"`
	Motion          bool                       `json:"motion,omitempty"`
	Overtemperature bool                       `json:"overtemperature,omitempty"`
}

// WiFiStatus contains WiFi connection status.
type WiFiStatus struct {
	SSID      string `json:"ssid,omitempty"`
	IP        string `json:"ip,omitempty"`
	RSSI      int    `json:"rssi,omitempty"`
	Connected bool   `json:"connected"`
}

// CloudStatus contains Shelly cloud connection status.
type CloudStatus struct {
	// Enabled indicates if cloud is enabled.
	Enabled bool `json:"enabled"`

	// Connected indicates if connected to cloud.
	Connected bool `json:"connected"`
}

// MQTTStatus contains MQTT connection status.
type MQTTStatus struct {
	// Connected indicates if connected to MQTT broker.
	Connected bool `json:"connected"`
}

// ActionsStats contains action execution statistics.
type ActionsStats struct {
	// Skipped is the number of skipped actions.
	Skipped int `json:"skipped,omitempty"`
}

// RelayStatus contains status for a single relay.
type RelayStatus struct {
	Source         string `json:"source,omitempty"`
	TimerStarted   int64  `json:"timer_started,omitempty"`
	TimerDuration  int    `json:"timer_duration,omitempty"`
	TimerRemaining int    `json:"timer_remaining,omitempty"`
	IsOn           bool   `json:"ison"`
	HasTimer       bool   `json:"has_timer,omitempty"`
	Overpower      bool   `json:"overpower,omitempty"`
}

// RollerStatus contains status for a single roller/cover.
type RollerStatus struct {
	State           string  `json:"state,omitempty"`
	Source          string  `json:"source,omitempty"`
	StopReason      string  `json:"stop_reason,omitempty"`
	LastDirection   string  `json:"last_direction,omitempty"`
	Power           float64 `json:"power,omitempty"`
	CurrentPos      int     `json:"current_pos,omitempty"`
	IsValid         bool    `json:"is_valid,omitempty"`
	SafetySwitch    bool    `json:"safety_switch,omitempty"`
	Overtemperature bool    `json:"overtemperature,omitempty"`
	Calibrating     bool    `json:"calibrating,omitempty"`
	Positioning     bool    `json:"positioning,omitempty"`
}

// LightStatus contains status for a single light.
type LightStatus struct {
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

// MeterStatus contains status for a power meter.
type MeterStatus struct {
	Counters  []float64 `json:"counters,omitempty"`
	Power     float64   `json:"power"`
	Overpower float64   `json:"overpower,omitempty"`
	Timestamp int64     `json:"timestamp,omitempty"`
	Total     int       `json:"total,omitempty"`
	IsValid   bool      `json:"is_valid,omitempty"`
}

// EMeterStatus contains status for an energy meter.
type EMeterStatus struct {
	// Power is current power in watts.
	Power float64 `json:"power"`

	// PF is the power factor.
	PF float64 `json:"pf,omitempty"`

	// Current is the current in amps.
	Current float64 `json:"current,omitempty"`

	// Voltage is the voltage in volts.
	Voltage float64 `json:"voltage,omitempty"`

	// IsValid indicates if the reading is valid.
	IsValid bool `json:"is_valid,omitempty"`

	// Total is total energy in watt-hours.
	Total float64 `json:"total,omitempty"`

	// TotalReturned is total returned energy in watt-hours.
	TotalReturned float64 `json:"total_returned,omitempty"`
}

// InputStatus contains status for an input.
type InputStatus struct {
	Event    string `json:"event,omitempty"`
	Input    int    `json:"input"`
	EventCnt int    `json:"event_cnt,omitempty"`
}

// TemperatureData contains temperature sensor data.
type TemperatureData struct {
	Units   string  `json:"units,omitempty"`
	TC      float64 `json:"tC,omitempty"`
	TF      float64 `json:"tF,omitempty"`
	Value   float64 `json:"value,omitempty"`
	IsValid bool    `json:"is_valid,omitempty"`
}

// HumidityData contains humidity sensor data.
type HumidityData struct {
	// Value is the humidity percentage.
	Value float64 `json:"value,omitempty"`

	// IsValid indicates if the reading is valid.
	IsValid bool `json:"is_valid,omitempty"`
}

// BatteryStatus contains battery information.
type BatteryStatus struct {
	// Value is the battery percentage (0-100).
	Value int `json:"value,omitempty"`

	// Voltage is the battery voltage.
	Voltage float64 `json:"voltage,omitempty"`
}

// LuxData contains light level sensor data.
type LuxData struct {
	Illumination string  `json:"illumination,omitempty"`
	Value        float64 `json:"value,omitempty"`
	IsValid      bool    `json:"is_valid,omitempty"`
}

// AccelData contains accelerometer data.
type AccelData struct {
	// Tilt is the tilt angle.
	Tilt int `json:"tilt,omitempty"`

	// Vibration indicates vibration detected.
	Vibration bool `json:"vibration,omitempty"`
}

// GasStatus contains gas sensor status.
type GasStatus struct {
	// SensorState is the sensor state.
	SensorState string `json:"sensor_state,omitempty"`

	// AlarmState is the alarm state.
	AlarmState string `json:"alarm_state,omitempty"`

	// SelfTestState is the self-test state.
	SelfTestState string `json:"self_test_state,omitempty"`
}

// ConcentrationData contains gas concentration data.
type ConcentrationData struct {
	// PPM is the concentration in parts per million.
	PPM int `json:"ppm,omitempty"`

	// IsValid indicates if the reading is valid.
	IsValid bool `json:"is_valid,omitempty"`
}

// UpdateStatus contains firmware update information.
type UpdateStatus struct {
	Status      string `json:"status,omitempty"`
	NewVersion  string `json:"new_version,omitempty"`
	OldVersion  string `json:"old_version,omitempty"`
	BetaVersion string `json:"beta_version,omitempty"`
	HasUpdate   bool   `json:"has_update,omitempty"`
}

// Settings contains all device settings from /settings endpoint.
type Settings struct {
	types.RawFields `json:"-"`
	WiFiAp          *WiFiApSettings    `json:"wifi_ap,omitempty"`
	WiFiSta         *WiFiStaSettings   `json:"wifi_sta,omitempty"`
	WiFiSta1        *WiFiStaSettings   `json:"wifi_sta1,omitempty"`
	ApRoaming       *ApRoamingSettings `json:"ap_roaming,omitempty"`
	MQTT            *MQTTSettings      `json:"mqtt,omitempty"`
	CoIoT           *CoIoTSettings     `json:"coiot,omitempty"`
	SNTP            *SNTPSettings      `json:"sntp,omitempty"`
	Login           *LoginSettings     `json:"login,omitempty"`
	Cloud           *CloudSettings     `json:"cloud,omitempty"`
	BuildInfo       *BuildInfo         `json:"build_info,omitempty"`
	Device          *DeviceSettings    `json:"device,omitempty"`
	Tz              string             `json:"tz,omitempty"`
	Name            string             `json:"name,omitempty"`
	FW              string             `json:"fw,omitempty"`
	Time            string             `json:"time,omitempty"`
	Mode            string             `json:"mode,omitempty"`
	Meters          []MeterSettings    `json:"meters,omitempty"`
	Lights          []LightSettings    `json:"lights,omitempty"`
	EMeters         []EMeterSettings   `json:"emeters,omitempty"`
	Relays          []RelaySettings    `json:"relays,omitempty"`
	Rollers         []RollerSettings   `json:"rollers,omitempty"`
	Lng             float64            `json:"lng,omitempty"`
	TzUtcOffset     int                `json:"tz_utc_offset,omitempty"`
	MaxPower        int                `json:"max_power,omitempty"`
	Lat             float64            `json:"lat,omitempty"`
	Unixtime        int64              `json:"unixtime,omitempty"`
	Discoverable    bool               `json:"discoverable,omitempty"`
	TzDstAuto       bool               `json:"tz_dst_auto,omitempty"`
	TzDst           bool               `json:"tz_dst,omitempty"`
	Tzautodetect    bool               `json:"tzautodetect,omitempty"`
}

// DeviceSettings contains device-level settings.
type DeviceSettings struct {
	// Type is the device type.
	Type string `json:"type,omitempty"`

	// MAC is the device MAC address.
	MAC string `json:"mac,omitempty"`

	// Hostname is the device hostname.
	Hostname string `json:"hostname,omitempty"`

	// NumOutputs is the number of outputs.
	NumOutputs int `json:"num_outputs,omitempty"`

	// NumMeters is the number of meters.
	NumMeters int `json:"num_meters,omitempty"`

	// NumRollers is the number of rollers.
	NumRollers int `json:"num_rollers,omitempty"`
}

// WiFiApSettings contains WiFi access point settings.
type WiFiApSettings struct {
	SSID    string `json:"ssid,omitempty"`
	Key     string `json:"key,omitempty"`
	Enabled bool   `json:"enabled"`
}

// WiFiStaSettings contains WiFi station settings.
type WiFiStaSettings struct {
	SSID       string `json:"ssid,omitempty"`
	Key        string `json:"key,omitempty"`
	Ipv4Method string `json:"ipv4_method,omitempty"`
	IP         string `json:"ip,omitempty"`
	Gw         string `json:"gw,omitempty"`
	Mask       string `json:"mask,omitempty"`
	DNS        string `json:"dns,omitempty"`
	Enabled    bool   `json:"enabled"`
}

// ApRoamingSettings contains AP roaming settings.
type ApRoamingSettings struct {
	// Enabled indicates if AP roaming is enabled.
	Enabled bool `json:"enabled"`

	// Threshold is the RSSI threshold for roaming.
	Threshold int `json:"threshold,omitempty"`
}

// MQTTSettings contains MQTT broker settings.
type MQTTSettings struct {
	Server              string  `json:"server,omitempty"`
	User                string  `json:"user,omitempty"`
	Pass                string  `json:"pass,omitempty"`
	ID                  string  `json:"id,omitempty"`
	ReconnectTimeoutMax float64 `json:"reconnect_timeout_max,omitempty"`
	ReconnectTimeoutMin float64 `json:"reconnect_timeout_min,omitempty"`
	KeepAlive           int     `json:"keep_alive,omitempty"`
	MaxQos              int     `json:"max_qos,omitempty"`
	UpdatePeriod        int     `json:"update_period,omitempty"`
	Enable              bool    `json:"enable"`
	CleanSession        bool    `json:"clean_session,omitempty"`
	Retain              bool    `json:"retain,omitempty"`
}

// CoIoTSettings contains CoIoT protocol settings.
type CoIoTSettings struct {
	Peer         string `json:"peer,omitempty"`
	UpdatePeriod int    `json:"update_period,omitempty"`
	Enabled      bool   `json:"enabled"`
}

// SNTPSettings contains time synchronization settings.
type SNTPSettings struct {
	// Server is the NTP server address.
	Server string `json:"server,omitempty"`

	// Enabled indicates if SNTP is enabled.
	Enabled bool `json:"enabled"`
}

// LoginSettings contains authentication settings.
type LoginSettings struct {
	Username    string `json:"username,omitempty"`
	Enabled     bool   `json:"enabled"`
	Unprotected bool   `json:"unprotected,omitempty"`
}

// CloudSettings contains Shelly cloud settings.
type CloudSettings struct {
	// Enabled indicates if cloud is enabled.
	Enabled bool `json:"enabled"`

	// Connected indicates if connected to cloud.
	Connected bool `json:"connected,omitempty"`
}

// BuildInfo contains firmware build information.
type BuildInfo struct {
	// BuildID is the build identifier.
	BuildID string `json:"build_id,omitempty"`

	// BuildTimestamp is when the build was created.
	BuildTimestamp string `json:"build_timestamp,omitempty"`

	// BuildVersion is the build version.
	BuildVersion string `json:"build_version,omitempty"`
}

// RelaySettings contains settings for a single relay.
type RelaySettings struct {
	Name          string   `json:"name,omitempty"`
	ApplianceType string   `json:"appliance_type,omitempty"`
	DefaultState  string   `json:"default_state,omitempty"`
	BtnType       string   `json:"btn_type,omitempty"`
	ScheduleRules []string `json:"schedule_rules,omitempty"`
	AutoOn        float64  `json:"auto_on,omitempty"`
	AutoOff       float64  `json:"auto_off,omitempty"`
	MaxPower      int      `json:"max_power,omitempty"`
	BtnReverse    int      `json:"btn_reverse,omitempty"`
	IsOn          bool     `json:"ison,omitempty"`
	HasTimer      bool     `json:"has_timer,omitempty"`
	Schedule      bool     `json:"schedule,omitempty"`
}

// IsBtnReverse returns true if button reverse is enabled.
func (r *RelaySettings) IsBtnReverse() bool {
	return r.BtnReverse != 0
}

// RollerSettings contains settings for a single roller.
type RollerSettings struct {
	State                  string   `json:"state,omitempty"`
	ObstacleAction         string   `json:"obstacle_action,omitempty"`
	DefaultState           string   `json:"default_state,omitempty"`
	ObstacleMode           string   `json:"obstacle_mode,omitempty"`
	SafetyAction           string   `json:"safety_action,omitempty"`
	InputMode              string   `json:"input_mode,omitempty"`
	BtnType                string   `json:"btn_type,omitempty"`
	SafetyMode             string   `json:"safety_mode,omitempty"`
	ScheduleRules          []string `json:"schedule_rules,omitempty"`
	MaxTime                float64  `json:"maxtime,omitempty"`
	Power                  float64  `json:"power,omitempty"`
	MaxTimeClose           float64  `json:"maxtime_close,omitempty"`
	ObstaclePower          int      `json:"obstacle_power,omitempty"`
	ObstacleDelay          int      `json:"obstacle_delay,omitempty"`
	MaxTimeOpen            float64  `json:"maxtime_open,omitempty"`
	BtnReverse             int      `json:"btn_reverse,omitempty"`
	SafetySwitch           bool     `json:"safety_switch,omitempty"`
	Swap                   bool     `json:"swap,omitempty"`
	SafetyAllowedOnTrigger bool     `json:"safety_allowed_on_trigger,omitempty"`
	SwapInputs             bool     `json:"swap_inputs,omitempty"`
	Positioning            bool     `json:"positioning,omitempty"`
}

// LightSettings contains settings for a single light.
type LightSettings struct {
	Name          string   `json:"name,omitempty"`
	DefaultState  string   `json:"default_state,omitempty"`
	BtnType       string   `json:"btn_type,omitempty"`
	ScheduleRules []string `json:"schedule_rules,omitempty"`
	AutoOn        float64  `json:"auto_on,omitempty"`
	AutoOff       float64  `json:"auto_off,omitempty"`
	BtnReverse    int      `json:"btn_reverse,omitempty"` // 0 or 1
	Schedule      bool     `json:"schedule,omitempty"`
}

// MeterSettings contains settings for a power meter.
type MeterSettings struct {
	// PowerLimit is the power limit in watts.
	PowerLimit float64 `json:"power_limit,omitempty"`

	// UnderLimit is the under-power limit.
	UnderLimit float64 `json:"under_limit,omitempty"`

	// OverLimit is the over-power limit.
	OverLimit float64 `json:"over_limit,omitempty"`
}

// EMeterSettings contains settings for an energy meter.
type EMeterSettings struct {
	// CTType is the current transformer type.
	CTType int `json:"cttype,omitempty"`
}

// UpdateInfo contains firmware update check results.
type UpdateInfo struct {
	NewVersion string `json:"new_version,omitempty"`
	Status     string `json:"status,omitempty"`
	HasUpdate  bool   `json:"has_update,omitempty"`
}
