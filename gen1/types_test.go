package gen1

import (
	"testing"

	"github.com/tj-smith47/shelly-go/types"
)

// TestDeviceInfoToTypesDeviceInfo tests DeviceInfo conversion.
func TestDeviceInfoToTypesDeviceInfo(t *testing.T) {
	info := &DeviceInfo{
		Type: "SHSW-25",
		MAC:  "AABBCCDDEEFF",
		Auth: true,
		FW:   "20210115-123456/v1.9.5@abc123",
	}

	typesInfo := info.ToTypesDeviceInfo()

	if typesInfo.ID != "AABBCCDDEEFF" {
		t.Errorf("expected ID AABBCCDDEEFF, got %s", typesInfo.ID)
	}

	if typesInfo.Model != "SHSW-25" {
		t.Errorf("expected Model SHSW-25, got %s", typesInfo.Model)
	}

	if typesInfo.Version != "20210115-123456/v1.9.5@abc123" {
		t.Errorf("expected Version to match, got %s", typesInfo.Version)
	}

	if typesInfo.Generation != types.Gen1 {
		t.Errorf("expected Generation Gen1, got %v", typesInfo.Generation)
	}

	if !typesInfo.AuthEnabled {
		t.Error("expected AuthEnabled to be true")
	}

	if typesInfo.MAC != "AABBCCDDEEFF" {
		t.Errorf("expected MAC AABBCCDDEEFF, got %s", typesInfo.MAC)
	}
}

// TestStatusFields tests Status struct fields.
func TestStatusFields(t *testing.T) {
	status := &Status{
		WiFiSta: &WiFiStatus{
			Connected: true,
			SSID:      "TestNet",
			IP:        "192.168.1.100",
			RSSI:      -50,
		},
		Cloud: &CloudStatus{
			Enabled:   true,
			Connected: true,
		},
		MQTT: &MQTTStatus{
			Connected: false,
		},
		Time:          "14:30",
		UnixTime:      1699300000,
		Serial:        42,
		HasUpdate:     true,
		MAC:           "AABBCCDDEEFF",
		CfgChangedCnt: 5,
		Temperature:   45.5,
		Uptime:        86400,
	}

	if !status.WiFiSta.Connected {
		t.Error("expected WiFi connected")
	}

	if status.WiFiSta.SSID != "TestNet" {
		t.Errorf("expected SSID TestNet, got %s", status.WiFiSta.SSID)
	}

	if status.WiFiSta.RSSI != -50 {
		t.Errorf("expected RSSI -50, got %d", status.WiFiSta.RSSI)
	}

	if !status.Cloud.Enabled {
		t.Error("expected cloud enabled")
	}

	if !status.Cloud.Connected {
		t.Error("expected cloud connected")
	}

	if status.MQTT.Connected {
		t.Error("expected MQTT not connected")
	}

	if status.Serial != 42 {
		t.Errorf("expected serial 42, got %d", status.Serial)
	}

	if !status.HasUpdate {
		t.Error("expected has_update true")
	}

	if status.Temperature != 45.5 {
		t.Errorf("expected temperature 45.5, got %f", status.Temperature)
	}

	if status.Uptime != 86400 {
		t.Errorf("expected uptime 86400, got %d", status.Uptime)
	}
}

// TestRelayStatus tests RelayStatus struct.
func TestRelayStatus(t *testing.T) {
	status := RelayStatus{
		IsOn:           true,
		HasTimer:       true,
		TimerStarted:   1699300000,
		TimerDuration:  60,
		TimerRemaining: 30,
		Overpower:      false,
		Source:         "input",
	}

	if !status.IsOn {
		t.Error("expected relay on")
	}

	if !status.HasTimer {
		t.Error("expected timer active")
	}

	if status.TimerRemaining != 30 {
		t.Errorf("expected 30 seconds remaining, got %d", status.TimerRemaining)
	}

	if status.Source != "input" {
		t.Errorf("expected source input, got %s", status.Source)
	}
}

// TestRollerStatus tests RollerStatus struct.
func TestRollerStatus(t *testing.T) {
	status := RollerStatus{
		State:         "stop",
		Source:        "http",
		Power:         15.5,
		IsValid:       true,
		SafetySwitch:  false,
		StopReason:    "normal",
		LastDirection: "open",
		CurrentPos:    75,
		Calibrating:   false,
		Positioning:   true,
	}

	if status.State != "stop" {
		t.Errorf("expected state stop, got %s", status.State)
	}

	if status.CurrentPos != 75 {
		t.Errorf("expected position 75, got %d", status.CurrentPos)
	}

	if !status.Positioning {
		t.Error("expected positioning enabled")
	}

	if status.LastDirection != "open" {
		t.Errorf("expected last direction open, got %s", status.LastDirection)
	}
}

// TestLightStatus tests LightStatus struct.
func TestLightStatus(t *testing.T) {
	status := LightStatus{
		IsOn:       true,
		Source:     "button",
		Brightness: 80,
		Temp:       3000,
		Transition: 500,
	}

	if !status.IsOn {
		t.Error("expected light on")
	}

	if status.Brightness != 80 {
		t.Errorf("expected brightness 80, got %d", status.Brightness)
	}

	if status.Temp != 3000 {
		t.Errorf("expected temp 3000K, got %d", status.Temp)
	}
}

// TestMeterStatus tests MeterStatus struct.
func TestMeterStatus(t *testing.T) {
	status := MeterStatus{
		Power:     150.5,
		Overpower: 2000,
		IsValid:   true,
		Timestamp: 1699300000,
		Counters:  []float64{10.5, 20.3, 15.7},
		Total:     100000,
	}

	if status.Power != 150.5 {
		t.Errorf("expected power 150.5, got %f", status.Power)
	}

	if len(status.Counters) != 3 {
		t.Errorf("expected 3 counters, got %d", len(status.Counters))
	}

	if status.Total != 100000 {
		t.Errorf("expected total 100000, got %d", status.Total)
	}
}

// TestEMeterStatus tests EMeterStatus struct.
func TestEMeterStatus(t *testing.T) {
	status := EMeterStatus{
		Power:         -500.0, // Negative = returning power
		PF:            0.95,
		Current:       2.5,
		Voltage:       230.0,
		IsValid:       true,
		Total:         50000.0,
		TotalReturned: 10000.0,
	}

	if status.Power != -500.0 {
		t.Errorf("expected power -500, got %f", status.Power)
	}

	if status.Voltage != 230.0 {
		t.Errorf("expected voltage 230, got %f", status.Voltage)
	}

	if status.Current != 2.5 {
		t.Errorf("expected current 2.5, got %f", status.Current)
	}

	if status.PF != 0.95 {
		t.Errorf("expected PF 0.95, got %f", status.PF)
	}

	if status.TotalReturned != 10000.0 {
		t.Errorf("expected total returned 10000, got %f", status.TotalReturned)
	}
}

// TestInputStatus tests InputStatus struct.
func TestInputStatus(t *testing.T) {
	status := InputStatus{
		Input:    1,
		Event:    "S",
		EventCnt: 42,
	}

	if status.Input != 1 {
		t.Errorf("expected input 1, got %d", status.Input)
	}

	if status.Event != "S" {
		t.Errorf("expected event S, got %s", status.Event)
	}

	if status.EventCnt != 42 {
		t.Errorf("expected event count 42, got %d", status.EventCnt)
	}
}

// TestTemperatureData tests TemperatureData struct.
func TestTemperatureData(t *testing.T) {
	data := TemperatureData{
		TC:      25.5,
		TF:      77.9,
		Value:   25.5,
		Units:   "C",
		IsValid: true,
	}

	if data.TC != 25.5 {
		t.Errorf("expected TC 25.5, got %f", data.TC)
	}

	if data.TF != 77.9 {
		t.Errorf("expected TF 77.9, got %f", data.TF)
	}

	if data.Units != "C" {
		t.Errorf("expected units C, got %s", data.Units)
	}
}

// TestHumidityData tests HumidityData struct.
func TestHumidityData(t *testing.T) {
	data := HumidityData{
		Value:   65.0,
		IsValid: true,
	}

	if data.Value != 65.0 {
		t.Errorf("expected value 65, got %f", data.Value)
	}

	if !data.IsValid {
		t.Error("expected valid reading")
	}
}

// TestBatteryStatus tests BatteryStatus struct.
func TestBatteryStatus(t *testing.T) {
	status := BatteryStatus{
		Value:   85,
		Voltage: 3.7,
	}

	if status.Value != 85 {
		t.Errorf("expected value 85, got %d", status.Value)
	}

	if status.Voltage != 3.7 {
		t.Errorf("expected voltage 3.7, got %f", status.Voltage)
	}
}

// TestSettings tests Settings struct.
func TestSettings(t *testing.T) {
	settings := &Settings{
		Device: &DeviceSettings{
			Type:       "SHSW-1",
			MAC:        "AABBCCDDEEFF",
			Hostname:   "shelly1-EEFF",
			NumOutputs: 1,
		},
		WiFiAp: &WiFiApSettings{
			Enabled: true,
			SSID:    "shelly1-AP",
		},
		WiFiSta: &WiFiStaSettings{
			Enabled:    true,
			SSID:       "HomeNetwork",
			Ipv4Method: "dhcp",
		},
		MQTT: &MQTTSettings{
			Enable:       true,
			Server:       "mqtt.example.com:1883",
			CleanSession: true,
		},
		CoIoT: &CoIoTSettings{
			Enabled:      true,
			UpdatePeriod: 15,
		},
		Name:         "Living Room Switch",
		FW:           "v1.9.5",
		Discoverable: true,
		Lat:          40.7128,
		Lng:          -74.006,
		Mode:         "relay",
	}

	if settings.Device.Type != "SHSW-1" {
		t.Errorf("expected device type SHSW-1, got %s", settings.Device.Type)
	}

	if !settings.WiFiAp.Enabled {
		t.Error("expected WiFi AP enabled")
	}

	if !settings.WiFiSta.Enabled {
		t.Error("expected WiFi station enabled")
	}

	if !settings.MQTT.Enable {
		t.Error("expected MQTT enabled")
	}

	if settings.CoIoT.UpdatePeriod != 15 {
		t.Errorf("expected CoIoT period 15, got %d", settings.CoIoT.UpdatePeriod)
	}

	if settings.Name != "Living Room Switch" {
		t.Errorf("expected name 'Living Room Switch', got %s", settings.Name)
	}

	if settings.Mode != "relay" {
		t.Errorf("expected mode relay, got %s", settings.Mode)
	}
}

// TestRelaySettings tests RelaySettings struct.
func TestRelaySettings(t *testing.T) {
	settings := RelaySettings{
		Name:          "Main Light",
		ApplianceType: "light",
		DefaultState:  "last",
		BtnType:       "momentary",
		AutoOn:        0,
		AutoOff:       300,
		MaxPower:      1000,
		Schedule:      true,
	}

	if settings.Name != "Main Light" {
		t.Errorf("expected name Main Light, got %s", settings.Name)
	}

	if settings.DefaultState != "last" {
		t.Errorf("expected default state last, got %s", settings.DefaultState)
	}

	if settings.BtnType != "momentary" {
		t.Errorf("expected btn type momentary, got %s", settings.BtnType)
	}

	if settings.AutoOff != 300 {
		t.Errorf("expected auto off 300, got %f", settings.AutoOff)
	}

	if settings.MaxPower != 1000 {
		t.Errorf("expected max power 1000, got %d", settings.MaxPower)
	}
}

// TestRollerSettings tests RollerSettings struct.
func TestRollerSettings(t *testing.T) {
	settings := RollerSettings{
		MaxTime:        30,
		MaxTimeOpen:    28,
		MaxTimeClose:   32,
		DefaultState:   "stop",
		SwapInputs:     false,
		Swap:           false,
		InputMode:      "openclose",
		BtnType:        "momentary",
		ObstacleMode:   "both",
		ObstacleAction: "reverse",
		ObstaclePower:  100,
		ObstacleDelay:  1,
		SafetyMode:     "while_opening",
		SafetyAction:   "stop",
		Positioning:    true,
	}

	if settings.MaxTime != 30 {
		t.Errorf("expected max time 30, got %f", settings.MaxTime)
	}

	if settings.InputMode != "openclose" {
		t.Errorf("expected input mode openclose, got %s", settings.InputMode)
	}

	if settings.ObstacleMode != "both" {
		t.Errorf("expected obstacle mode both, got %s", settings.ObstacleMode)
	}

	if !settings.Positioning {
		t.Error("expected positioning enabled")
	}
}

// TestUpdateInfo tests UpdateInfo struct.
func TestUpdateInfo(t *testing.T) {
	info := UpdateInfo{
		HasUpdate:  true,
		NewVersion: "v1.10.0",
		Status:     "pending",
	}

	if !info.HasUpdate {
		t.Error("expected has update")
	}

	if info.NewVersion != "v1.10.0" {
		t.Errorf("expected version v1.10.0, got %s", info.NewVersion)
	}

	if info.Status != "pending" {
		t.Errorf("expected status pending, got %s", info.Status)
	}
}

// TestUpdateStatus tests UpdateStatus struct.
func TestUpdateStatus(t *testing.T) {
	status := UpdateStatus{
		Status:      "idle",
		HasUpdate:   true,
		NewVersion:  "v1.10.0",
		OldVersion:  "v1.9.5",
		BetaVersion: "v1.10.1-beta",
	}

	if status.Status != "idle" {
		t.Errorf("expected status idle, got %s", status.Status)
	}

	if status.OldVersion != "v1.9.5" {
		t.Errorf("expected old version v1.9.5, got %s", status.OldVersion)
	}

	if status.BetaVersion != "v1.10.1-beta" {
		t.Errorf("expected beta version v1.10.1-beta, got %s", status.BetaVersion)
	}
}

// TestGasStatus tests GasStatus struct.
func TestGasStatus(t *testing.T) {
	status := GasStatus{
		SensorState:   "normal",
		AlarmState:    "none",
		SelfTestState: "completed",
	}

	if status.SensorState != "normal" {
		t.Errorf("expected sensor state normal, got %s", status.SensorState)
	}

	if status.AlarmState != "none" {
		t.Errorf("expected alarm state none, got %s", status.AlarmState)
	}
}

// TestLuxData tests LuxData struct.
func TestLuxData(t *testing.T) {
	data := LuxData{
		Value:        1500.0,
		Illumination: "bright",
		IsValid:      true,
	}

	if data.Value != 1500.0 {
		t.Errorf("expected lux 1500, got %f", data.Value)
	}

	if data.Illumination != "bright" {
		t.Errorf("expected illumination bright, got %s", data.Illumination)
	}
}

// TestAccelData tests AccelData struct.
func TestAccelData(t *testing.T) {
	data := AccelData{
		Tilt:      45,
		Vibration: true,
	}

	if data.Tilt != 45 {
		t.Errorf("expected tilt 45, got %d", data.Tilt)
	}

	if !data.Vibration {
		t.Error("expected vibration detected")
	}
}

// TestConcentrationData tests ConcentrationData struct.
func TestConcentrationData(t *testing.T) {
	data := ConcentrationData{
		PPM:     500,
		IsValid: true,
	}

	if data.PPM != 500 {
		t.Errorf("expected PPM 500, got %d", data.PPM)
	}

	if !data.IsValid {
		t.Error("expected valid reading")
	}
}

// TestBuildInfo tests BuildInfo struct.
func TestBuildInfo(t *testing.T) {
	info := BuildInfo{
		BuildID:        "20210115-123456",
		BuildTimestamp: "2021-01-15T12:34:56Z",
		BuildVersion:   "v1.9.5",
	}

	if info.BuildID != "20210115-123456" {
		t.Errorf("expected build ID 20210115-123456, got %s", info.BuildID)
	}

	if info.BuildVersion != "v1.9.5" {
		t.Errorf("expected build version v1.9.5, got %s", info.BuildVersion)
	}
}

// TestLoginSettings tests LoginSettings struct.
func TestLoginSettings(t *testing.T) {
	settings := LoginSettings{
		Enabled:     true,
		Unprotected: false,
		Username:    "admin",
	}

	if !settings.Enabled {
		t.Error("expected login enabled")
	}

	if settings.Unprotected {
		t.Error("expected not unprotected")
	}

	if settings.Username != "admin" {
		t.Errorf("expected username admin, got %s", settings.Username)
	}
}

// TestSNTPSettings tests SNTPSettings struct.
func TestSNTPSettings(t *testing.T) {
	settings := SNTPSettings{
		Server:  "pool.ntp.org",
		Enabled: true,
	}

	if settings.Server != "pool.ntp.org" {
		t.Errorf("expected server pool.ntp.org, got %s", settings.Server)
	}

	if !settings.Enabled {
		t.Error("expected SNTP enabled")
	}
}

// TestApRoamingSettings tests ApRoamingSettings struct.
func TestApRoamingSettings(t *testing.T) {
	settings := ApRoamingSettings{
		Enabled:   true,
		Threshold: -70,
	}

	if !settings.Enabled {
		t.Error("expected AP roaming enabled")
	}

	if settings.Threshold != -70 {
		t.Errorf("expected threshold -70, got %d", settings.Threshold)
	}
}

// TestMeterSettings tests MeterSettings struct.
func TestMeterSettings(t *testing.T) {
	settings := MeterSettings{
		PowerLimit: 2000,
		UnderLimit: 10,
		OverLimit:  2200,
	}

	if settings.PowerLimit != 2000 {
		t.Errorf("expected power limit 2000, got %f", settings.PowerLimit)
	}

	if settings.UnderLimit != 10 {
		t.Errorf("expected under limit 10, got %f", settings.UnderLimit)
	}

	if settings.OverLimit != 2200 {
		t.Errorf("expected over limit 2200, got %f", settings.OverLimit)
	}
}

// TestEMeterSettings tests EMeterSettings struct.
func TestEMeterSettings(t *testing.T) {
	settings := EMeterSettings{
		CTType: 1, // 120A CT
	}

	if settings.CTType != 1 {
		t.Errorf("expected CT type 1, got %d", settings.CTType)
	}
}

// TestLightSettings tests LightSettings struct.
func TestLightSettings(t *testing.T) {
	settings := LightSettings{
		Name:         "Bedroom Light",
		DefaultState: "off",
		AutoOn:       0,
		AutoOff:      1800, // 30 minutes
		BtnType:      "momentary",
		Schedule:     true,
	}

	if settings.Name != "Bedroom Light" {
		t.Errorf("expected name Bedroom Light, got %s", settings.Name)
	}

	if settings.DefaultState != "off" {
		t.Errorf("expected default state off, got %s", settings.DefaultState)
	}

	if settings.AutoOff != 1800 {
		t.Errorf("expected auto off 1800, got %f", settings.AutoOff)
	}
}
