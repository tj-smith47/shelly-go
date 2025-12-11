package gen1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tj-smith47/shelly-go/internal/testutil"
)

func TestSetWiFiStation(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("wifi_sta_enabled", json.RawMessage(`{}`), nil)

	err := device.SetWiFiStation(context.Background(), true, "MyNetwork", "pass123")
	if err != nil {
		t.Fatalf("SetWiFiStation failed: %v", err)
	}
}

func TestSetWiFiStationStatic(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("wifi_sta_ipv4_method=static", json.RawMessage(`{}`), nil)

	err := device.SetWiFiStationStatic(context.Background(), "Network", "pass", "192.168.1.100", "192.168.1.1", "255.255.255.0", "8.8.8.8")
	if err != nil {
		t.Fatalf("SetWiFiStationStatic failed: %v", err)
	}
}

func TestSetWiFiAP(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("wifi_ap_enabled", json.RawMessage(`{}`), nil)

	err := device.SetWiFiAP(context.Background(), true, "MyAP", "appassword")
	if err != nil {
		t.Fatalf("SetWiFiAP failed: %v", err)
	}
}

func TestSetMQTT(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("mqtt_enable", json.RawMessage(`{}`), nil)

	err := device.SetMQTT(context.Background(), true, "192.168.1.100:1883", "user", "pass")
	if err != nil {
		t.Fatalf("SetMQTT failed: %v", err)
	}
}

func TestSetMQTTConfig(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("mqtt_enable", json.RawMessage(`{}`), nil)

	config := MQTTConfig{
		Enable:              true,
		Server:              "mqtt.example.com:1883",
		User:                "user",
		Password:            "pass",
		ID:                  "shelly-123",
		ReconnectTimeoutMax: 60,
		ReconnectTimeoutMin: 5,
		CleanSession:        true,
		KeepAlive:           60,
		MaxQos:              1,
		Retain:              true,
		UpdatePeriod:        30,
	}

	err := device.SetMQTTConfig(context.Background(), &config)
	if err != nil {
		t.Fatalf("SetMQTTConfig failed: %v", err)
	}
}

func TestSetCoIoT(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("coiot_enable", json.RawMessage(`{}`), nil)

	err := device.SetCoIoT(context.Background(), true, 30, "192.168.1.100:5683")
	if err != nil {
		t.Fatalf("SetCoIoT failed: %v", err)
	}
}

func TestSetCloud(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings/cloud?enabled=true", `{}`)

	err := device.SetCloud(context.Background(), true)
	if err != nil {
		t.Fatalf("SetCloud failed: %v", err)
	}
}

func TestSetAuth(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/login?", json.RawMessage(`{}`), nil)

	err := device.SetAuth(context.Background(), true, "admin", "password123")
	if err != nil {
		t.Fatalf("SetAuth failed: %v", err)
	}
}

func TestSetTimeServer(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("sntp_server", json.RawMessage(`{}`), nil)

	err := device.SetTimeServer(context.Background(), "time.google.com")
	if err != nil {
		t.Fatalf("SetTimeServer failed: %v", err)
	}
}

func TestGetActions(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings/actions", `{
		"actions": [
			{"index": 0, "name": "out_on_url", "urls": ["http://example.com"], "enabled": true}
		]
	}`)

	actions, err := device.GetActions(context.Background())
	if err != nil {
		t.Fatalf("GetActions failed: %v", err)
	}

	if len(actions.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(actions.Actions))
	}
	if !actions.Actions[0].Enabled {
		t.Error("Expected action to be enabled")
	}
}

func TestSetAction(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/actions?", json.RawMessage(`{}`), nil)

	err := device.SetAction(context.Background(), 0, ActionOutputOnUrl, []string{"http://example.com/trigger"}, true)
	if err != nil {
		t.Fatalf("SetAction failed: %v", err)
	}
}

func TestSetActionURL(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/actions?", json.RawMessage(`{}`), nil)

	err := device.SetActionURL(context.Background(), 0, ActionOutputOff, "http://example.com/off", true)
	if err != nil {
		t.Fatalf("SetActionURL failed: %v", err)
	}
}

func TestClearAction(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/actions?", json.RawMessage(`{}`), nil)

	err := device.ClearAction(context.Background(), 0, ActionOutputOnUrl)
	if err != nil {
		t.Fatalf("ClearAction failed: %v", err)
	}
}

func TestSetRelayConfig(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/relay/0?", json.RawMessage(`{}`), nil)

	config := RelayConfig{
		Name:         "Living Room",
		DefaultState: "off",
		BtnType:      "momentary",
		AutoOn:       0,
		AutoOff:      300,
		MaxPower:     2000,
		Schedule:     true,
	}

	err := device.SetRelayConfig(context.Background(), 0, &config)
	if err != nil {
		t.Fatalf("SetRelayConfig failed: %v", err)
	}
}

func TestGetRelaySettings(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings/relay/0", `{
		"name": "Relay 0",
		"default_state": "off",
		"btn_type": "momentary",
		"max_power": 2500
	}`)

	settings, err := device.GetRelaySettings(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetRelaySettings failed: %v", err)
	}

	if settings.Name != "Relay 0" {
		t.Errorf("Expected name 'Relay 0', got '%s'", settings.Name)
	}
	if settings.MaxPower != 2500 {
		t.Errorf("Expected max power 2500, got %d", settings.MaxPower)
	}
}

func TestSetRollerConfig(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/roller/0?", json.RawMessage(`{}`), nil)

	config := RollerConfig{
		MaxTimeOpen:    60,
		MaxTimeClose:   60,
		DefaultState:   "stop",
		InputMode:      "openclose",
		Positioning:    true,
		ObstacleMode:   "while_opening",
		ObstacleAction: "stop",
		ObstaclePower:  80,
		ObstacleDelay:  100,
	}

	err := device.SetRollerConfig(context.Background(), 0, &config)
	if err != nil {
		t.Fatalf("SetRollerConfig failed: %v", err)
	}
}

func TestGetRollerSettings(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings/roller/0", `{
		"maxtime": 60,
		"default_state": "stop",
		"positioning": true
	}`)

	settings, err := device.GetRollerSettings(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetRollerSettings failed: %v", err)
	}

	if settings.MaxTime != 60 {
		t.Errorf("Expected max time 60, got %f", settings.MaxTime)
	}
	if !settings.Positioning {
		t.Error("Expected positioning to be true")
	}
}

func TestSetLightConfig(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/light/0?", json.RawMessage(`{}`), nil)

	config := LightConfig{
		Name:         "Bedroom Light",
		DefaultState: "last",
		AutoOff:      3600,
		BtnType:      "toggle",
		Schedule:     false,
	}

	err := device.SetLightConfig(context.Background(), 0, config)
	if err != nil {
		t.Fatalf("SetLightConfig failed: %v", err)
	}
}

func TestGetLightSettings(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings/light/0", `{
		"name": "Light 0",
		"default_state": "on",
		"auto_off": 0
	}`)

	settings, err := device.GetLightSettings(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetLightSettings failed: %v", err)
	}

	if settings.Name != "Light 0" {
		t.Errorf("Expected name 'Light 0', got '%s'", settings.Name)
	}
}

func TestGetCoIoTDescription(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/cit/d", `{
		"blk": [
			{"I": 0, "D": "Relay0"}
		],
		"sen": [
			{"I": 111, "T": "P", "D": "Power", "R": "0/2500", "L": 0}
		]
	}`)

	desc, err := device.GetCoIoTDescription(context.Background())
	if err != nil {
		t.Fatalf("GetCoIoTDescription failed: %v", err)
	}

	if len(desc.Blk) != 1 {
		t.Errorf("Expected 1 block, got %d", len(desc.Blk))
	}
	if len(desc.Sen) != 1 {
		t.Errorf("Expected 1 sensor, got %d", len(desc.Sen))
	}
}

func TestGetCoIoTStatusValues(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/cit/s", `{
		"G": [[0, 111, 45.5], [0, 112, 1]]
	}`)

	status, err := device.GetCoIoTStatusValues(context.Background())
	if err != nil {
		t.Fatalf("GetCoIoTStatusValues failed: %v", err)
	}

	if len(status.G) != 2 {
		t.Errorf("Expected 2 values, got %d", len(status.G))
	}
}

func TestGetTemperature(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/temperature", `{
		"tC": 23.5,
		"tF": 74.3,
		"is_valid": true
	}`)

	temp, err := device.GetTemperature(context.Background())
	if err != nil {
		t.Fatalf("GetTemperature failed: %v", err)
	}

	if temp.TC != 23.5 {
		t.Errorf("Expected 23.5 C, got %f", temp.TC)
	}
	if !temp.IsValid {
		t.Error("Expected valid reading")
	}
}

func TestGetHumidity(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/humidity", `{
		"value": 65.2,
		"is_valid": true
	}`)

	hum, err := device.GetHumidity(context.Background())
	if err != nil {
		t.Fatalf("GetHumidity failed: %v", err)
	}

	if hum.Value != 65.2 {
		t.Errorf("Expected 65.2%%, got %f", hum.Value)
	}
}

func TestGetExternalSensor(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/sensor/temperature", `{
		"tC": 21.0,
		"tF": 69.8,
		"is_valid": true
	}`)

	sensor, err := device.GetExternalSensor(context.Background())
	if err != nil {
		t.Fatalf("GetExternalSensor failed: %v", err)
	}

	if sensor.TC != 21.0 {
		t.Errorf("Expected 21.0 C, got %f", sensor.TC)
	}
}

func TestAddScheduleRule(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/relay/0?schedule_rules", json.RawMessage(`{}`), nil)

	err := device.AddScheduleRule(context.Background(), 0, "0800-7F-0-on")
	if err != nil {
		t.Fatalf("AddScheduleRule failed: %v", err)
	}
}

func TestSetScheduleRules(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/relay/0?", json.RawMessage(`{}`), nil)

	err := device.SetScheduleRules(context.Background(), 0, []string{"0800-7F-0-on", "2200-7F-0-off"})
	if err != nil {
		t.Fatalf("SetScheduleRules failed: %v", err)
	}
}

func TestEnableSchedule(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings/relay/0?schedule=true", `{}`)

	err := device.EnableSchedule(context.Background(), 0, true)
	if err != nil {
		t.Fatalf("EnableSchedule failed: %v", err)
	}
}

func TestSetMode(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings?mode=roller", `{}`)

	err := device.SetMode(context.Background(), "roller")
	if err != nil {
		t.Fatalf("SetMode failed: %v", err)
	}
}

func TestSetDiscoverable(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings?discoverable=false", `{}`)

	err := device.SetDiscoverable(context.Background(), false)
	if err != nil {
		t.Fatalf("SetDiscoverable failed: %v", err)
	}
}

func TestSetMaxPower(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings?max_power=2500", `{}`)

	err := device.SetMaxPower(context.Background(), 2500)
	if err != nil {
		t.Fatalf("SetMaxPower failed: %v", err)
	}
}

func TestScanWiFi(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/wifiscan", `{
		"results": [
			{"ssid": "Network1", "bssid": "AA:BB:CC:DD:EE:FF", "rssi": -45, "channel": 6, "auth": 3},
			{"ssid": "Network2", "bssid": "11:22:33:44:55:66", "rssi": -70, "channel": 11, "auth": 2}
		]
	}`)

	networks, err := device.ScanWiFi(context.Background())
	if err != nil {
		t.Fatalf("ScanWiFi failed: %v", err)
	}

	if len(networks) != 2 {
		t.Errorf("Expected 2 networks, got %d", len(networks))
	}
	if networks[0].SSID != "Network1" {
		t.Errorf("Expected SSID 'Network1', got '%s'", networks[0].SSID)
	}
}

func TestScanWiFiDirectArray(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	// Some devices return direct array instead of {results: [...]}
	mock.OnCallJSON("/wifiscan", `[
		{"ssid": "Network1", "rssi": -45}
	]`)

	networks, err := device.ScanWiFi(context.Background())
	if err != nil {
		t.Fatalf("ScanWiFi failed: %v", err)
	}

	if len(networks) != 1 {
		t.Errorf("Expected 1 network, got %d", len(networks))
	}
}

func TestBuildScheduleRule(t *testing.T) {
	tests := []struct {
		action string
		want   string
		hour   int
		minute int
		days   int
	}{
		{hour: 8, minute: 0, days: DayEveryDay, action: "on", want: "0800-7F-0-on"},
		{hour: 22, minute: 30, days: DayWeekdays, action: "off", want: "2230-3E-0-off"},
		{hour: 0, minute: 0, days: DaySunday, action: "on", want: "0000-01-0-on"},
		{hour: 12, minute: 15, days: DayWeekends, action: "off", want: "1215-41-0-off"},
	}

	for _, tt := range tests {
		got := BuildScheduleRule(tt.hour, tt.minute, tt.days, tt.action)
		if got != tt.want {
			t.Errorf("BuildScheduleRule(%d, %d, %d, %s) = %s, want %s",
				tt.hour, tt.minute, tt.days, tt.action, got, tt.want)
		}
	}
}

func TestParseScheduleRule(t *testing.T) {
	tests := []struct {
		rule       string
		wantAction string
		wantHour   int
		wantMinute int
		wantDays   int
		wantRelay  int
		wantErr    bool
	}{
		{rule: "0800-7F-0-on", wantHour: 8, wantMinute: 0, wantDays: 0x7F, wantRelay: 0, wantAction: "on", wantErr: false},
		{rule: "2230-3E-1-off", wantHour: 22, wantMinute: 30, wantDays: 0x3E, wantRelay: 1, wantAction: "off", wantErr: false},
		{rule: "invalid", wantHour: 0, wantMinute: 0, wantDays: 0, wantRelay: 0, wantAction: "", wantErr: true},
		{rule: "08-7F-0-on", wantHour: 0, wantMinute: 0, wantDays: 0, wantRelay: 0, wantAction: "", wantErr: true}, // Wrong time format
	}

	for _, tt := range tests {
		hour, minute, days, relay, action, err := ParseScheduleRule(tt.rule)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseScheduleRule(%s) error = %v, wantErr %v", tt.rule, err, tt.wantErr)
			continue
		}
		if err == nil {
			if hour != tt.wantHour || minute != tt.wantMinute || days != tt.wantDays ||
				relay != tt.wantRelay || action != tt.wantAction {
				t.Errorf("ParseScheduleRule(%s) = (%d, %d, %d, %d, %s), want (%d, %d, %d, %d, %s)",
					tt.rule, hour, minute, days, relay, action,
					tt.wantHour, tt.wantMinute, tt.wantDays, tt.wantRelay, tt.wantAction)
			}
		}
	}
}

func TestBoolToString(t *testing.T) {
	if boolToString(true) != "true" {
		t.Error("boolToString(true) should be 'true'")
	}
	if boolToString(false) != "false" {
		t.Error("boolToString(false) should be 'false'")
	}
}

func TestDayConstants(t *testing.T) {
	if DayEveryDay != 0x7F {
		t.Errorf("DayEveryDay = %x, want 0x7F", DayEveryDay)
	}
	if DayWeekdays != 0x3E {
		t.Errorf("DayWeekdays = %x, want 0x3E", DayWeekdays)
	}
	if DayWeekends != 0x41 {
		t.Errorf("DayWeekends = %x, want 0x41", DayWeekends)
	}
}

// Error path tests

func TestSetWiFiStationError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("wifi_sta_enabled", nil, errTest)

	err := device.SetWiFiStation(context.Background(), true, "MyNetwork", "pass123")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetWiFiStationStaticError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("wifi_sta_ipv4_method=static", nil, errTest)

	err := device.SetWiFiStationStatic(context.Background(), "Network", "pass", "192.168.1.100", "192.168.1.1", "255.255.255.0", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetWiFiAPError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("wifi_ap_enabled", nil, errTest)

	err := device.SetWiFiAP(context.Background(), true, "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetMQTTError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("mqtt_enable", nil, errTest)

	err := device.SetMQTT(context.Background(), true, "", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetMQTTConfigError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("mqtt_enable", nil, errTest)

	config := MQTTConfig{Enable: false}
	err := device.SetMQTTConfig(context.Background(), &config)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetCoIoTError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("coiot_enable", nil, errTest)

	err := device.SetCoIoT(context.Background(), true, 0, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetCloudError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/settings/cloud?enabled=false", errTest)

	err := device.SetCloud(context.Background(), false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetAuthError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/login?", nil, errTest)

	err := device.SetAuth(context.Background(), true, "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetTimeServerError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("sntp_server", nil, errTest)

	err := device.SetTimeServer(context.Background(), "time.google.com")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetActionsError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/settings/actions", errTest)

	_, err := device.GetActions(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetActionsParseError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings/actions", `{invalid}`)

	_, err := device.GetActions(context.Background())
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestSetActionError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/actions?", nil, errTest)

	err := device.SetAction(context.Background(), 0, ActionOutputOn, nil, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetRelayConfigError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/relay/0?", nil, errTest)

	config := RelayConfig{BtnReverse: true, AutoOn: 10}
	err := device.SetRelayConfig(context.Background(), 0, &config)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetRelaySettingsError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/settings/relay/0", errTest)

	_, err := device.GetRelaySettings(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetRelaySettingsParseError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings/relay/0", `{invalid}`)

	_, err := device.GetRelaySettings(context.Background(), 0)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestSetRollerConfigError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/roller/0?", nil, errTest)

	config := RollerConfig{SwapInputs: true, BtnReverse: true, SafetyMode: "disabled", BtnType: "toggle"}
	err := device.SetRollerConfig(context.Background(), 0, &config)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetRollerSettingsError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/settings/roller/0", errTest)

	_, err := device.GetRollerSettings(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetRollerSettingsParseError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings/roller/0", `{invalid}`)

	_, err := device.GetRollerSettings(context.Background(), 0)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestSetLightConfigError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/light/0?", nil, errTest)

	config := LightConfig{BtnReverse: true, AutoOn: 10}
	err := device.SetLightConfig(context.Background(), 0, config)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetLightSettingsError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/settings/light/0", errTest)

	_, err := device.GetLightSettings(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetLightSettingsParseError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/settings/light/0", `{invalid}`)

	_, err := device.GetLightSettings(context.Background(), 0)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestGetCoIoTDescriptionError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/cit/d", errTest)

	_, err := device.GetCoIoTDescription(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetCoIoTDescriptionParseError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/cit/d", `{invalid}`)

	_, err := device.GetCoIoTDescription(context.Background())
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestGetCoIoTStatusValuesError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/cit/s", errTest)

	_, err := device.GetCoIoTStatusValues(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetCoIoTStatusValuesParseError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/cit/s", `{invalid}`)

	_, err := device.GetCoIoTStatusValues(context.Background())
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestGetTemperatureError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/temperature", errTest)

	_, err := device.GetTemperature(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetTemperatureParseError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/temperature", `{invalid}`)

	_, err := device.GetTemperature(context.Background())
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestGetHumidityError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/humidity", errTest)

	_, err := device.GetHumidity(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetHumidityParseError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/humidity", `{invalid}`)

	_, err := device.GetHumidity(context.Background())
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestGetExternalSensorError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/sensor/temperature", errTest)

	_, err := device.GetExternalSensor(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetExternalSensorParseError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/sensor/temperature", `{invalid}`)

	_, err := device.GetExternalSensor(context.Background())
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestAddScheduleRuleError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/relay/0?schedule_rules", nil, errTest)

	err := device.AddScheduleRule(context.Background(), 0, "0800-7F-0-on")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetScheduleRulesError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/relay/0?", nil, errTest)

	err := device.SetScheduleRules(context.Background(), 0, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnableScheduleError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/settings/relay/0?schedule=false", errTest)

	err := device.EnableSchedule(context.Background(), 0, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetModeError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/settings?mode=roller", errTest)

	err := device.SetMode(context.Background(), "roller")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetDiscoverableError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/settings?discoverable=true", errTest)

	err := device.SetDiscoverable(context.Background(), true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetMaxPowerError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/settings?max_power=3000", errTest)

	err := device.SetMaxPower(context.Background(), 3000)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScanWiFiError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/wifiscan", errTest)

	_, err := device.ScanWiFi(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScanWiFiParseError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallJSON("/wifiscan", `{invalid json not array or object}`)

	_, err := device.ScanWiFi(context.Background())
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseScheduleRuleErrors(t *testing.T) {
	// Test various error cases
	tests := []string{
		"",             // empty
		"a-b-c",        // too few parts
		"abc-7F-0-on",  // bad time (3 chars)
		"0x00-7F-0-on", // invalid hour
		"08xx-7F-0-on", // invalid minute
		"0800-ZZ-0-on", // invalid days hex
		"0800-7F-x-on", // invalid relay
	}

	for _, rule := range tests {
		_, _, _, _, _, err := ParseScheduleRule(rule)
		if err == nil {
			t.Errorf("ParseScheduleRule(%q) expected error", rule)
		}
	}
}

var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }
