package backup

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/transport"
)

// gen1ColorDevice starts a fake Gen1 color device. It answers /shelly with
// device info and every other path with the given status, recording the full
// request URI of each non-/shelly call (so set calls can be asserted).
func gen1ColorDevice(t *testing.T, statusJSON string) (*gen1.Device, *[]string) {
	t.Helper()
	var (
		mu       sync.Mutex
		setQuery []string
	)
	write := func(w http.ResponseWriter, body string) {
		if _, err := io.WriteString(w, body); err != nil {
			t.Errorf("write response: %v", err)
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/shelly" {
			write(w, `{"type":"SHRGBW2","mac":"AABBCCDDEEFF","fw":"1.0","auth":false,"num_outputs":1}`)
			return
		}
		// Record the full request URI (path + query) of every other call so both
		// component reads (/color, /white) and config writes (/settings*) can be
		// asserted; reply with the configured status body (ignored by writes).
		mu.Lock()
		setQuery = append(setQuery, r.URL.RequestURI())
		mu.Unlock()
		write(w, statusJSON)
	}))
	t.Cleanup(srv.Close)

	dev := gen1.NewDevice(transport.NewHTTP(srv.URL))
	return dev, &setQuery
}

func TestGen1ColorState_CaptureAndApply(t *testing.T) {
	t.Parallel()
	dev, sets := gen1ColorDevice(t, `{"ison":true,"red":10,"green":20,"blue":30,"white":40,"gain":55,"effect":2}`)

	raw := captureGen1ColorState(t.Context(), dev)
	if raw == nil {
		t.Fatal("captureGen1ColorState returned nil")
	}
	var cs gen1ColorState
	if err := json.Unmarshal(raw, &cs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := gen1ColorState{Red: 10, Green: 20, Blue: 30, White: 40, Gain: 55, Effect: 2}
	if cs != want {
		t.Errorf("captured = %+v, want %+v", cs, want)
	}

	bkp := &Backup{Components: map[string]json.RawMessage{colorStateKey: raw}}
	result := &RestoreResult{Success: true}
	applyGen1ColorState(t.Context(), dev, bkp, result)
	if len(result.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
	joined := strings.Join(*sets, " ")
	for _, want := range []string{"red=10", "green=20", "blue=30", "white=40", "gain=55", "effect=2"} {
		if !strings.Contains(joined, want) {
			t.Errorf("set call %q not issued; calls=%v", want, *sets)
		}
	}
}

func TestGen1WhiteState_CaptureAndApply(t *testing.T) {
	t.Parallel()
	dev, sets := gen1ColorDevice(t, `{"ison":true,"brightness":77}`)

	raw := captureGen1WhiteState(t.Context(), dev, 1)
	if raw == nil {
		t.Fatal("captureGen1WhiteState returned nil")
	}
	var ws []gen1WhiteState
	if err := json.Unmarshal(raw, &ws); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(ws) != 1 || ws[0].ID != 0 || ws[0].Brightness != 77 {
		t.Errorf("captured = %+v", ws)
	}

	bkp := &Backup{Components: map[string]json.RawMessage{whiteStateKey: raw}}
	result := &RestoreResult{Success: true}
	applyGen1WhiteState(t.Context(), dev, bkp, result)
	if len(result.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
	if !strings.Contains(strings.Join(*sets, " "), "brightness=77") {
		t.Errorf("brightness set call not issued; calls=%v", *sets)
	}
}

func TestRestoreGen1WiFi_Sta1AndApRoaming(t *testing.T) {
	t.Parallel()
	dev, sets := gen1ColorDevice(t, `{}`)

	// Capture path: marshalGen1WiFi produces the same WiFi blob the backup stores.
	settings := &gen1.Settings{
		WiFiSta1:  &gen1.WiFiStaSettings{Enabled: true, SSID: "Backup", Key: "secret"},
		ApRoaming: &gen1.ApRoamingSettings{Enabled: true, Threshold: -70},
	}
	bkp := &Backup{WiFi: marshalGen1WiFi(settings)}
	result := &RestoreResult{Success: true}

	restoreGen1WiFi(t.Context(), dev, bkp, nil, result)
	if len(result.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
	joined := strings.Join(*sets, " ")
	for _, want := range []string{"/settings/sta1?", "ssid=Backup", "ap_roaming_enabled=true", "ap_roaming_threshold=-70"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in requests: %v", want, *sets)
		}
	}
}

// TestRestoreGen1EMeters asserts captured energy-meter settings are written back to
// /settings/emeter/{index} using the real Gen1 params (max_power + over/under power
// URLs and thresholds), never ct_type.
func TestRestoreGen1EMeters(t *testing.T) {
	t.Parallel()
	dev, sets := gen1ColorDevice(t, `{}`)

	settings := &gen1.Settings{
		EMeters: []gen1.EMeterSettings{
			{MaxPower: 2300, OverPowerURL: "http://hook/over", OverPowerURLThreshold: 2000},
			{}, // empty entry must produce no write
		},
	}
	result := &RestoreResult{Success: true}

	restoreGen1EMeters(t.Context(), dev, settings, result)
	if len(result.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
	joined := strings.Join(*sets, " ")
	for _, want := range []string{"/settings/emeter/0?", "max_power=2300", "over_power_url=", "over_power_url_threshold=2000"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in requests: %v", want, *sets)
		}
	}
	for _, unwanted := range []string{"/settings/emeter/1?", "cttype", "ct_type"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("unexpected %q in requests: %v", unwanted, *sets)
		}
	}
}

func TestRestoreGen1Components_ScheduleRules(t *testing.T) {
	t.Parallel()
	dev, sets := gen1ColorDevice(t, `{}`)

	settings := &gen1.Settings{
		Lights: []gen1.LightSettings{{Name: "l0", ScheduleRules: []string{"0800-7F-0-on"}}},
		Relays: []gen1.RelaySettings{{Name: "r0", ScheduleRules: []string{"2200-7F-0-off"}}},
	}
	result := &RestoreResult{Success: true}

	restoreGen1Components(t.Context(), dev, settings, result)
	if len(result.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
	joined := strings.Join(*sets, " ")
	for _, want := range []string{"/settings/light/0?", "/settings/relay/0?", "schedule_rules", "0800-7F-0-on", "2200-7F-0-off"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in requests: %v", want, *sets)
		}
	}
}

func TestCaptureGen1WhiteState_NoChannels(t *testing.T) {
	t.Parallel()
	if got := captureGen1WhiteState(t.Context(), nil, 0); got != nil {
		t.Errorf("expected nil for 0 channels, got %s", got)
	}
}

// TestGen1Settings_TimezoneRoundTrips guards the tag bug that dropped the timezone
// from every Gen1 backup: the device's /settings key is "timezone", and Settings.Tz
// must read AND re-marshal under that key. When it was tagged "tz", GetSettings read
// nothing and the re-marshaled backup omitted the field, so restore never called
// SetTimezone — leaving the device with no clock and silently dropping its
// astronomical (sunrise/sunset) light schedule rules.
func TestGen1Settings_TimezoneRoundTrips(t *testing.T) {
	t.Parallel()
	var s gen1.Settings
	if err := json.Unmarshal([]byte(`{"timezone":"America/Los_Angeles"}`), &s); err != nil {
		t.Fatalf("unmarshal device settings: %v", err)
	}
	if s.Tz != "America/Los_Angeles" {
		t.Fatalf("device 'timezone' not read into Tz; got %q", s.Tz)
	}
	out, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"timezone":"America/Los_Angeles"`) {
		t.Errorf("re-marshaled backup dropped timezone: %s", out)
	}
}

func TestGen1SettingsHaveScheduleRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		settings *gen1.Settings
		want     bool
	}{
		{name: "none", settings: &gen1.Settings{}, want: false},
		{
			name:     "light rule",
			settings: &gen1.Settings{Lights: []gen1.LightSettings{{ScheduleRules: []string{"0000asr-0123456-0;101;off"}}}},
			want:     true,
		},
		{
			name:     "relay rule",
			settings: &gen1.Settings{Relays: []gen1.RelaySettings{{ScheduleRules: []string{"0800-7F-0-on"}}}},
			want:     true,
		},
		{
			name:     "empty rule slice",
			settings: &gen1.Settings{Lights: []gen1.LightSettings{{ScheduleRules: []string{}}}},
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gen1SettingsHaveScheduleRules(tt.settings); got != tt.want {
				t.Errorf("gen1SettingsHaveScheduleRules() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWaitGen1ClockSettle_ReturnsWhenClockSet asserts the wait returns without a
// warning as soon as the device reports a non-zero clock.
func TestWaitGen1ClockSettle_ReturnsWhenClockSet(t *testing.T) {
	t.Parallel()
	dev, _ := gen1ColorDevice(t, `{"unixtime":1781366055}`)
	result := &RestoreResult{Success: true}
	waitGen1ClockSettle(t.Context(), dev, result)
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warning when clock is set, got %v", result.Warnings)
	}
}

// TestWaitGen1ClockSettle_WarnsWhenClockNeverSets asserts that a device whose clock
// never appears records a warning (rather than failing the restore) and does not
// block past the deadline. A canceled context makes the bounded wait return at
// once without sleeping the full timeout.
func TestWaitGen1ClockSettle_WarnsWhenClockNeverSets(t *testing.T) {
	t.Parallel()
	dev, _ := gen1ColorDevice(t, `{"unixtime":0}`)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	result := &RestoreResult{Success: true}
	waitGen1ClockSettle(ctx, dev, result)
	if len(result.Warnings) != 1 {
		t.Fatalf("expected exactly one clock warning, got %v", result.Warnings)
	}
	if !strings.Contains(result.Warnings[0], "clock not set") {
		t.Errorf("warning = %q, want it to mention the clock", result.Warnings[0])
	}
}

// TestApplyGen1State_AbsentIsNoOp confirms the color/white apply steps are a
// no-op (no warnings, no device call) when the backup carries no such state, so
// restoring a plain switch or white-temp bulb does not error.
func TestApplyGen1State_AbsentIsNoOp(t *testing.T) {
	t.Parallel()
	bkp := &Backup{}
	result := &RestoreResult{Success: true}
	applyGen1ColorState(t.Context(), nil, bkp, result)
	applyGen1WhiteState(t.Context(), nil, bkp, result)
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings for absent state, got %v", result.Warnings)
	}
}

func TestApplyGen1ColorState_BadJSON(t *testing.T) {
	t.Parallel()
	bkp := &Backup{
		Components: map[string]json.RawMessage{colorStateKey: json.RawMessage(`{bad`)},
	}
	result := &RestoreResult{Success: true}
	applyGen1ColorState(t.Context(), nil, bkp, result)
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "parse color state") {
		t.Errorf("expected a parse warning, got %v", result.Warnings)
	}
}

func TestMarshalGen1WiFi(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings *gen1.Settings
		wantNil  bool
		wantKeys []string
	}{
		{
			name:     "empty settings",
			settings: &gen1.Settings{},
			wantNil:  true,
		},
		{
			name: "with station",
			settings: &gen1.Settings{
				WiFiSta: &gen1.WiFiStaSettings{SSID: "TestNetwork", Key: "pass123"},
			},
			wantKeys: []string{"sta"},
		},
		{
			name: "with station and AP",
			settings: &gen1.Settings{
				WiFiSta: &gen1.WiFiStaSettings{SSID: "TestNetwork"},
				WiFiAp:  &gen1.WiFiApSettings{SSID: "ShellyAP", Enabled: true},
			},
			wantKeys: []string{"sta", "ap"},
		},
		{
			name: "all WiFi fields",
			settings: &gen1.Settings{
				WiFiSta:   &gen1.WiFiStaSettings{SSID: "Net1"},
				WiFiSta1:  &gen1.WiFiStaSettings{SSID: "Net2"},
				WiFiAp:    &gen1.WiFiApSettings{SSID: "AP"},
				ApRoaming: &gen1.ApRoamingSettings{Enabled: true},
			},
			wantKeys: []string{"sta", "sta1", "ap", "ap_roaming"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := marshalGen1WiFi(tt.settings)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %s", string(result))
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			var parsed map[string]json.RawMessage
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("failed to parse result: %v", err)
			}

			for _, key := range tt.wantKeys {
				if _, ok := parsed[key]; !ok {
					t.Errorf("missing key %q in WiFi result", key)
				}
			}
		})
	}
}

func TestMarshalGen1Components(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings *gen1.Settings
		wantNil  bool
		wantKeys []string
	}{
		{
			name:     "empty settings",
			settings: &gen1.Settings{},
			wantNil:  true,
		},
		{
			name: "with relays",
			settings: &gen1.Settings{
				Relays: []gen1.RelaySettings{
					{Name: "relay0"},
				},
			},
			wantKeys: []string{"relays"},
		},
		{
			name: "with lights and relays",
			settings: &gen1.Settings{
				Lights: []gen1.LightSettings{
					{Name: "light0"},
				},
				Relays: []gen1.RelaySettings{
					{Name: "relay0"},
				},
			},
			wantKeys: []string{"lights", "relays"},
		},
		{
			name: "all component types",
			settings: &gen1.Settings{
				Lights:  []gen1.LightSettings{{Name: "light0"}},
				Relays:  []gen1.RelaySettings{{Name: "relay0"}},
				Rollers: []gen1.RollerSettings{{DefaultState: "stop"}},
				Meters:  []gen1.MeterSettings{{PowerLimit: 100}},
				EMeters: []gen1.EMeterSettings{{MaxPower: 2300}},
			},
			wantKeys: []string{"lights", "relays", "rollers", "meters", "emeters"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := marshalGen1Components(tt.settings)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %d keys", len(result))
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			for _, key := range tt.wantKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("missing key %q in components", key)
				}
			}
		})
	}
}

func TestMarshalGen1Schedules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings *gen1.Settings
		wantNil  bool
		wantLen  int
	}{
		{
			name:     "no schedules",
			settings: &gen1.Settings{},
			wantNil:  true,
		},
		{
			name: "relay schedules",
			settings: &gen1.Settings{
				Relays: []gen1.RelaySettings{
					{Schedule: true, ScheduleRules: []string{"0 0 8 * * *"}},
					{Schedule: false, ScheduleRules: []string{}},
				},
			},
			wantLen: 1,
		},
		{
			name: "light schedules",
			settings: &gen1.Settings{
				Lights: []gen1.LightSettings{
					{Schedule: true, ScheduleRules: []string{"0 0 20 * * *", "0 0 22 * * *"}},
				},
			},
			wantLen: 1,
		},
		{
			name: "relay and light schedules",
			settings: &gen1.Settings{
				Relays: []gen1.RelaySettings{
					{Schedule: true, ScheduleRules: []string{"0 0 8 * * *"}},
				},
				Lights: []gen1.LightSettings{
					{Schedule: true, ScheduleRules: []string{"0 0 20 * * *"}},
				},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := marshalGen1Schedules(tt.settings)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %s", string(result))
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			var entries []json.RawMessage
			if err := json.Unmarshal(result, &entries); err != nil {
				t.Fatalf("failed to parse schedules: %v", err)
			}

			if len(entries) != tt.wantLen {
				t.Errorf("got %d schedule entries, want %d", len(entries), tt.wantLen)
			}
		})
	}
}

func TestMustMarshal(t *testing.T) {
	t.Parallel()

	data := map[string]string{"field": "data"}
	result := mustMarshal(data)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if parsed["field"] != "data" {
		t.Errorf("got %q, want %q", parsed["field"], "data")
	}
}

func TestMustMarshal_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unmarshalable value")
		}
	}()

	// Channels cannot be marshaled to JSON
	mustMarshal(make(chan int))
}

func TestAddWarning(t *testing.T) {
	t.Parallel()

	result := &RestoreResult{}

	addWarningf(result, "error %d: %s", 1, "test")

	if len(result.Warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(result.Warnings))
	}

	if result.Warnings[0] != "error 1: test" {
		t.Errorf("got %q, want %q", result.Warnings[0], "error 1: test")
	}

	addWarningf(result, "second warning")

	if len(result.Warnings) != 2 {
		t.Fatalf("got %d warnings, want 2", len(result.Warnings))
	}
}

func TestCaptureGen1LightState_NoLights(t *testing.T) {
	t.Parallel()
	if got := captureGen1LightState(t.Context(), nil, 0); got != nil {
		t.Errorf("expected nil for 0 lights, got %s", got)
	}
}

func TestGen1LightStateRoundTrip(t *testing.T) {
	t.Parallel()
	raw := mustMarshal([]gen1LightState{{ID: 0, Temp: 3000, Brightness: 50}})
	var out []gen1LightState
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 || out[0].Temp != 3000 || out[0].Brightness != 50 {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

func TestRestoreGen1Meters_SkipsZeroLimit(t *testing.T) {
	t.Parallel()
	// A zero PowerLimit must be left alone (no setter call, no warning). With a nil
	// device this would panic if it tried to write, proving the zero-limit skip.
	result := &RestoreResult{Success: true}
	settings := &gen1.Settings{Meters: []gen1.MeterSettings{{PowerLimit: 0}}}
	restoreGen1Meters(t.Context(), nil, settings, result)
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings for zero limit, got %v", result.Warnings)
	}
}

func TestGen1NetworkOverride_IsStatic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ov   *Gen1NetworkOverride
		want bool
	}{
		{name: "nil", ov: nil, want: false},
		{name: "empty static ip", ov: &Gen1NetworkOverride{Gateway: "10.0.0.1"}, want: false},
		{name: "static ip set", ov: &Gen1NetworkOverride{StaticIP: "10.23.47.221"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.ov.IsStatic(); got != tt.want {
				t.Errorf("IsStatic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyGen1WiFiOverride(t *testing.T) {
	t.Parallel()
	t.Run("static ip with backup credentials preserved", func(t *testing.T) {
		t.Parallel()
		sta := &gen1.WiFiStaSettings{SSID: "OnyxCheetah4.7", Key: "secret", Ipv4Method: "dhcp"}
		applyGen1WiFiOverride(sta, &Gen1NetworkOverride{
			StaticIP: "10.23.47.221",
			Gateway:  "10.23.47.1",
			Netmask:  "255.255.254.0",
			DNS:      "10.23.47.1",
		})

		if !sta.Enabled {
			t.Error("station should be enabled")
		}
		if sta.SSID != "OnyxCheetah4.7" || sta.Key != "secret" {
			t.Errorf("credentials should be preserved, got SSID=%q Key=%q", sta.SSID, sta.Key)
		}
		if sta.Ipv4Method != "static" {
			t.Errorf("Ipv4Method = %q, want static", sta.Ipv4Method)
		}
		if sta.IP != "10.23.47.221" || sta.Gw != "10.23.47.1" || sta.Mask != "255.255.254.0" || sta.DNS != "10.23.47.1" {
			t.Errorf("static fields not applied: %+v", sta)
		}
	})

	t.Run("explicit ssid and password override", func(t *testing.T) {
		t.Parallel()
		sta := &gen1.WiFiStaSettings{SSID: "old", Key: "oldkey"}
		applyGen1WiFiOverride(sta, &Gen1NetworkOverride{SSID: "new", Password: "newkey", StaticIP: "10.0.0.5", Gateway: "10.0.0.1", Netmask: "255.255.255.0"})
		if sta.SSID != "new" || sta.Key != "newkey" {
			t.Errorf("credentials not overridden, got SSID=%q Key=%q", sta.SSID, sta.Key)
		}
	})

	t.Run("non-static override leaves ipv4 method untouched", func(t *testing.T) {
		t.Parallel()
		sta := &gen1.WiFiStaSettings{SSID: "net", Key: "k", Ipv4Method: "dhcp", IP: "10.0.0.9"}
		applyGen1WiFiOverride(sta, &Gen1NetworkOverride{SSID: "net2"})
		if sta.Ipv4Method != "dhcp" || sta.IP != "10.0.0.9" {
			t.Errorf("non-static override mutated addressing: %+v", sta)
		}
		if sta.SSID != "net2" {
			t.Errorf("ssid override not applied: %q", sta.SSID)
		}
	})
}
