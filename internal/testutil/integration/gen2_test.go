package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/gen2/components"
)

func TestGen2Device_GetDeviceInfo(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	info, err := device.Shelly().GetDeviceInfo(ctx)
	if err != nil {
		t.Fatalf("GetDeviceInfo() error = %v", err)
	}

	if info.ID == "" {
		t.Error("ID should not be empty")
	}
	if info.Model == "" {
		t.Error("Model should not be empty")
	}
	if info.MAC == "" {
		t.Error("MAC should not be empty")
	}
	if info.FirmwareID == "" {
		t.Error("FirmwareID should not be empty")
	}

	t.Logf("Device Info: ID=%s Model=%s MAC=%s Gen=%d FW=%s",
		info.ID, info.Model, info.MAC, info.Gen, info.FirmwareID)
}

func TestGen2Device_GetStatus(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	status, err := device.Shelly().GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status == nil {
		t.Fatal("GetStatus() returned nil")
	}

	// Status is a map, just verify it has some content
	if len(status) == 0 {
		t.Error("GetStatus() returned empty status")
	}

	t.Logf("Device has %d status components", len(status))
}

func TestGen2Device_GetConfig(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	config, err := device.Shelly().GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if config == nil {
		t.Fatal("GetConfig() returned nil")
	}

	// Config is a map, just verify it has some content
	if len(config) == 0 {
		t.Error("GetConfig() returned empty config")
	}

	t.Logf("Device has %d config components", len(config))
}

func TestGen2Device_ListMethods(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	methods, err := device.Shelly().ListMethods(ctx)
	if err != nil {
		t.Fatalf("ListMethods() error = %v", err)
	}

	if len(methods) == 0 {
		t.Error("ListMethods() returned no methods")
	}

	t.Logf("Device supports %d methods", len(methods))

	// Log a few methods
	for i, method := range methods {
		if i >= 5 {
			t.Logf("  ... and %d more", len(methods)-5)
			break
		}
		t.Logf("  - %s", method)
	}
}

func TestGen2Device_Switch(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	// Get device info to check capabilities
	info, err := device.Shelly().GetDeviceInfo(ctx)
	if err != nil {
		t.Fatalf("GetDeviceInfo() error = %v", err)
	}

	// Check if device has switch profile
	if info.Profile != "" && info.Profile != "switch" {
		t.Skipf("Device profile is %s, not switch", info.Profile)
	}

	// Use typed Switch component for full API access
	sw := components.NewSwitch(device.Client(), 0)

	// Get status (always safe - read-only)
	status, err := sw.GetStatus(ctx)
	if err != nil {
		t.Skipf("Device may not have switch: %v", err)
	}

	apower := float64(0)
	if status.APower != nil {
		apower = *status.APower
	}
	voltage := float64(0)
	if status.Voltage != nil {
		voltage = *status.Voltage
	}
	t.Logf("Switch 0 Status: Output=%v, APower=%.2f, Voltage=%.2f",
		status.Output, apower, voltage)

	// Skip actuating tests unless explicitly enabled
	if !ActuateEnabled() {
		t.Log("Skipping switch actuation tests (set SHELLY_TEST_ACTUATE=1 to enable)")
		return
	}

	// Save original state
	originalState := status.Output

	// Test Set on
	result, err := sw.Set(ctx, &components.SwitchSetParams{On: true})
	if err != nil {
		t.Errorf("Switch.Set(true) error = %v", err)
	} else {
		t.Logf("Switch.Set(true) result: WasOn=%v", result.WasOn)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify on
	status, err = sw.GetStatus(ctx)
	if err != nil {
		t.Errorf("Switch.GetStatus() after Set(true) error = %v", err)
	} else if !status.Output {
		t.Error("Switch should be on after Set(true)")
	}

	// Test Set off
	result, err = sw.Set(ctx, &components.SwitchSetParams{On: false})
	if err != nil {
		t.Errorf("Switch.Set(false) error = %v", err)
	} else {
		t.Logf("Switch.Set(false) result: WasOn=%v", result.WasOn)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify off
	status, err = sw.GetStatus(ctx)
	if err != nil {
		t.Errorf("Switch.GetStatus() after Set(false) error = %v", err)
	} else if status.Output {
		t.Error("Switch should be off after Set(false)")
	}

	// Restore original state
	sw.Set(ctx, &components.SwitchSetParams{On: originalState})
}

func TestGen2Device_Switch_Toggle(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	// Skip actuating tests unless explicitly enabled
	SkipIfNoActuate(t)

	sw := components.NewSwitch(device.Client(), 0)

	// Get initial status
	status, err := sw.GetStatus(ctx)
	if err != nil {
		t.Skipf("Device may not have switch: %v", err)
	}

	originalState := status.Output

	// Toggle
	result, err := sw.Toggle(ctx)
	if err != nil {
		t.Fatalf("Switch.Toggle() error = %v", err)
	}

	if result.WasOn != originalState {
		t.Errorf("Toggle WasOn=%v, expected %v", result.WasOn, originalState)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify state changed
	status, err = sw.GetStatus(ctx)
	if err != nil {
		t.Errorf("Switch.GetStatus() after Toggle error = %v", err)
	} else if status.Output == originalState {
		t.Error("Switch state should have changed after Toggle")
	}

	// Toggle back to restore
	sw.Toggle(ctx)
}

func TestGen2Device_Switch_GetConfig(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	sw := components.NewSwitch(device.Client(), 0)

	config, err := sw.GetConfig(ctx)
	if err != nil {
		t.Skipf("Device may not have switch: %v", err)
	}

	name := ""
	if config.Name != nil {
		name = *config.Name
	}
	initialState := ""
	if config.InitialState != nil {
		initialState = *config.InitialState
	}
	t.Logf("Switch 0 Config: Name=%s, InitialState=%s, AutoOn=%v, AutoOff=%v",
		name, initialState, config.AutoOn, config.AutoOff)
}

func TestGen2Device_Input(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	input := components.NewInput(device.Client(), 0)

	status, err := input.GetStatus(ctx)
	if err != nil {
		t.Skipf("Device may not have input: %v", err)
	}

	t.Logf("Input 0 Status: State=%v", status.State)

	config, err := input.GetConfig(ctx)
	if err != nil {
		t.Errorf("Input.GetConfig() error = %v", err)
	} else {
		name := ""
		if config.Name != nil {
			name = *config.Name
		}
		t.Logf("Input 0 Config: Name=%s, Type=%s", name, config.Type)
	}
}

func TestGen2Device_WiFi(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	wifi := components.NewWiFi(device.Client())

	status, err := wifi.GetStatus(ctx)
	if err != nil {
		t.Fatalf("WiFi.GetStatus() error = %v", err)
	}

	staIP := ""
	if status.StaIP != nil {
		staIP = *status.StaIP
	}
	ssid := ""
	if status.SSID != nil {
		ssid = *status.SSID
	}
	rssi := float64(0)
	if status.RSSI != nil {
		rssi = *status.RSSI
	}
	t.Logf("WiFi Status: StaIP=%s, SSID=%s, RSSI=%.0f", staIP, ssid, rssi)

	config, err := wifi.GetConfig(ctx)
	if err != nil {
		t.Errorf("WiFi.GetConfig() error = %v", err)
	} else if config.AP != nil && config.STA != nil {
		t.Logf("WiFi Config: AP.Enable=%v, STA.Enable=%v", config.AP.Enable, config.STA.Enable)
	}
}

func TestGen2Device_Sys(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	sys := components.NewSys(device.Client())

	status, err := sys.GetStatus(ctx)
	if err != nil {
		t.Fatalf("Sys.GetStatus() error = %v", err)
	}

	t.Logf("Sys Status: MAC=%s, Uptime=%d, RAMSize=%d, RAMFree=%d, FSSize=%d, FSFree=%d",
		status.MAC, status.Uptime, status.RAMSize, status.RAMFree, status.FSSize, status.FSFree)

	config, err := sys.GetConfig(ctx)
	if err != nil {
		t.Errorf("Sys.GetConfig() error = %v", err)
	} else if config.Device != nil && config.Location != nil {
		name := ""
		if config.Device.Name != nil {
			name = *config.Device.Name
		}
		tz := ""
		if config.Location.TZ != nil {
			tz = *config.Location.TZ
		}
		t.Logf("Sys Config: Name=%s, Timezone=%s", name, tz)
	}
}

func TestGen2Device_CheckForUpdate(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	update, err := device.Shelly().CheckForUpdate(ctx)
	if err != nil {
		t.Fatalf("CheckForUpdate() error = %v", err)
	}

	hasStable := update.Stable != nil && update.Stable.Version != ""
	hasBeta := update.Beta != nil && update.Beta.Version != ""
	t.Logf("Update available: Stable=%v, Beta=%v", hasStable, hasBeta)

	if hasStable {
		t.Logf("Stable update: %s -> %s", update.Stable.Version, update.Stable.BuildID)
	}
}

func TestGen2Device_Cover(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	// Check if device has cover profile
	info, err := device.Shelly().GetDeviceInfo(ctx)
	if err != nil {
		t.Fatalf("GetDeviceInfo() error = %v", err)
	}

	if info.Profile != "" && info.Profile != "cover" {
		t.Skipf("Device profile is %s, not cover", info.Profile)
	}

	cover := components.NewCover(device.Client(), 0)

	status, err := cover.GetStatus(ctx)
	if err != nil {
		t.Skipf("Device may not have cover: %v", err)
	}

	t.Logf("Cover 0 Status: State=%s, CurrentPos=%d, TargetPos=%d",
		status.State, status.CurrentPos, status.TargetPos)
}

func TestGen2Device_Light(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	light := components.NewLight(device.Client(), 0)

	status, err := light.GetStatus(ctx)
	if err != nil {
		t.Skipf("Device may not have light: %v", err)
	}

	t.Logf("Light 0 Status: Output=%v, Brightness=%d", status.Output, status.Brightness)
}

func TestGen2Device_Script(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	script := components.NewScript(device.Client())

	listResp, err := script.List(ctx)
	if err != nil {
		t.Skipf("Device may not support scripts: %v", err)
	}

	t.Logf("Device has %d scripts", len(listResp.Scripts))
	for _, s := range listResp.Scripts {
		name := ""
		if s.Name != nil {
			name = *s.Name
		}
		t.Logf("  Script %d: %s (enable=%v, running=%v)", s.ID, name, s.Enable, s.Running)
	}
}

func TestGen2Device_Schedule(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	schedule := components.NewSchedule(device.Client())

	listResp, err := schedule.List(ctx)
	if err != nil {
		t.Skipf("Device may not support schedules: %v", err)
	}

	t.Logf("Device has %d scheduled jobs", len(listResp.Jobs))
	for _, job := range listResp.Jobs {
		t.Logf("  Job %d: enable=%v, timespec=%s", job.ID, job.Enable, job.Timespec)
	}
}

func TestGen2Device_Webhook(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	webhook := components.NewWebhook(device.Client())

	listResp, err := webhook.List(ctx)
	if err != nil {
		t.Skipf("Device may not support webhooks: %v", err)
	}

	t.Logf("Device has %d webhooks", len(listResp.Hooks))
	for _, hook := range listResp.Hooks {
		name := ""
		if hook.Name != nil {
			name = *hook.Name
		}
		t.Logf("  Webhook %d: %s (enable=%v, event=%s)", hook.ID, name, hook.Enable, hook.Event)
	}

	// List supported events
	supported, err := webhook.ListSupported(ctx)
	if err != nil {
		t.Errorf("Webhook.ListSupported() error = %v", err)
	} else {
		t.Logf("Supported webhook events: %d types", len(supported.HookTypes))
	}
}

func TestGen2Device_KVS(t *testing.T) {
	device := RequireGen2Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	kvs := components.NewKVS(device.Client())

	// List existing keys
	listResp, err := kvs.List(ctx)
	if err != nil {
		t.Skipf("Device may not support KVS: %v", err)
	}

	t.Logf("KVS has %d keys", len(listResp.Keys))

	// Set a test value
	testKey := "shelly_go_test_key"
	testValue := "test_value_123"

	result, err := kvs.Set(ctx, testKey, testValue)
	if err != nil {
		t.Errorf("KVS.Set() error = %v", err)
	} else {
		t.Logf("KVS.Set() etag=%s", result.Etag)
	}

	// Get the value back
	getResult, err := kvs.Get(ctx, testKey)
	if err != nil {
		t.Errorf("KVS.Get() error = %v", err)
	} else if getResult.Value != testValue {
		t.Errorf("KVS.Get() value = %v, want %v", getResult.Value, testValue)
	}

	// Delete the test key
	if _, err := kvs.Delete(ctx, testKey); err != nil {
		t.Errorf("KVS.Delete() error = %v", err)
	}
}

// Helper to suppress unused import warning for json package
var _ = json.RawMessage{}
