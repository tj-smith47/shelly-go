package integration

import (
	"testing"
	"time"
)

func TestGen1Device_Info(t *testing.T) {
	device := RequireGen1Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	info, err := device.GetDeviceInfo(ctx)
	if err != nil {
		t.Fatalf("GetDeviceInfo() error = %v", err)
	}

	if info.ID == "" {
		t.Error("Info.ID should not be empty")
	}
	if info.Model == "" {
		t.Error("Info.Model should not be empty")
	}
	if info.MAC == "" {
		t.Error("Info.MAC should not be empty")
	}
	if info.Version == "" {
		t.Error("Info.Version should not be empty")
	}

	t.Logf("Device Info: ID=%s Model=%s MAC=%s Version=%s",
		info.ID, info.Model, info.MAC, info.Version)
}

func TestGen1Device_Status(t *testing.T) {
	device := RequireGen1Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	status, err := device.GetFullStatus(ctx)
	if err != nil {
		t.Fatalf("GetFullStatus() error = %v", err)
	}

	if status == nil {
		t.Fatal("GetFullStatus() returned nil")
	}

	// Log WiFi status if available
	if status.WiFiSta != nil {
		t.Logf("Device Status: WiFi SSID=%s", status.WiFiSta.SSID)
	}
	t.Logf("Device Status: Uptime=%d, RAM Free=%d", status.Uptime, status.RAMFree)
}

func TestGen1Device_Settings(t *testing.T) {
	device := RequireGen1Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	settings, err := device.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}

	if settings == nil {
		t.Fatal("GetSettings() returned nil")
	}

	t.Logf("Device Settings: Name=%s, Timezone=%s",
		settings.Name, settings.Tz)
}

func TestGen1Device_Relay(t *testing.T) {
	device := RequireGen1Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	// Get full status to check if device has relays
	status, err := device.GetFullStatus(ctx)
	if err != nil {
		t.Fatalf("GetFullStatus() error = %v", err)
	}

	if len(status.Relays) == 0 {
		t.Skip("Device has no relays")
	}

	relay := device.Relay(0)

	// Get current status (always safe - read-only)
	relayStatus, err := relay.GetStatus(ctx)
	if err != nil {
		t.Fatalf("Relay.GetStatus() error = %v", err)
	}

	t.Logf("Relay 0 Status: IsOn=%v, HasTimer=%v", relayStatus.IsOn, relayStatus.HasTimer)

	// Skip actuating tests unless explicitly enabled
	if !ActuateEnabled() {
		t.Log("Skipping relay actuation tests (set SHELLY_TEST_ACTUATE=1 to enable)")
		return
	}

	// Test toggle (turn on then restore)
	originalState := relayStatus.IsOn

	// Turn on
	if err := relay.TurnOn(ctx); err != nil {
		t.Errorf("Relay.TurnOn() error = %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Verify on
	relayStatus, err = relay.GetStatus(ctx)
	if err != nil {
		t.Errorf("Relay.GetStatus() after TurnOn error = %v", err)
	} else if !relayStatus.IsOn {
		t.Error("Relay should be on after TurnOn()")
	}

	// Turn off
	if err := relay.TurnOff(ctx); err != nil {
		t.Errorf("Relay.TurnOff() error = %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Verify off
	relayStatus, err = relay.GetStatus(ctx)
	if err != nil {
		t.Errorf("Relay.GetStatus() after TurnOff error = %v", err)
	} else if relayStatus.IsOn {
		t.Error("Relay should be off after TurnOff()")
	}

	// Restore original state
	if originalState {
		relay.TurnOn(ctx)
	}
}

func TestGen1Device_Meter(t *testing.T) {
	device := RequireGen1Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	// Get full status to check if device has meters
	status, err := device.GetFullStatus(ctx)
	if err != nil {
		t.Fatalf("GetFullStatus() error = %v", err)
	}

	if len(status.Meters) == 0 {
		t.Skip("Device has no meters in status")
	}

	// Test meter data from full status (some devices don't expose /meter endpoint)
	t.Logf("Meter 0 Status from FullStatus: Power=%.2f, IsValid=%v", status.Meters[0].Power, status.Meters[0].IsValid)

	// Try dedicated meter endpoint (may return 404 on some devices like Shelly 1PM)
	meter := device.Meter(0)
	meterStatus, err := meter.GetStatus(ctx)
	if err != nil {
		t.Logf("Meter.GetStatus() not available (expected for some devices): %v", err)
		return // Not a failure - meter data is available via status
	}

	t.Logf("Meter 0 Status from endpoint: Power=%.2f, Total=%d", meterStatus.Power, meterStatus.Total)
}

func TestGen1Device_Input(t *testing.T) {
	device := RequireGen1Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	// Get full status to check if device has inputs
	status, err := device.GetFullStatus(ctx)
	if err != nil {
		t.Fatalf("GetFullStatus() error = %v", err)
	}

	if len(status.Inputs) == 0 {
		t.Skip("Device has no inputs")
	}

	input := device.Input(0)

	inputStatus, err := input.GetStatus(ctx)
	if err != nil {
		t.Fatalf("Input.GetStatus() error = %v", err)
	}

	t.Logf("Input 0 Status: Input=%d, Event=%s", inputStatus.Input, inputStatus.Event)
}

func TestGen1Device_Roller(t *testing.T) {
	device := RequireGen1Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	// Get full status to check if device has rollers
	status, err := device.GetFullStatus(ctx)
	if err != nil {
		t.Fatalf("GetFullStatus() error = %v", err)
	}

	if len(status.Rollers) == 0 {
		t.Skip("Device has no rollers")
	}

	roller := device.Roller(0)

	rollerStatus, err := roller.GetStatus(ctx)
	if err != nil {
		t.Fatalf("Roller.GetStatus() error = %v", err)
	}

	t.Logf("Roller 0 Status: State=%s, Position=%d", rollerStatus.State, rollerStatus.CurrentPos)
}

func TestGen1Device_Light(t *testing.T) {
	device := RequireGen1Device(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	// Check if device supports lights
	info, err := device.GetDeviceInfo(ctx)
	if err != nil {
		t.Fatalf("GetDeviceInfo() error = %v", err)
	}

	// Light devices have specific model patterns
	isLightDevice := false
	lightModels := []string{"SHBLB", "SHVIN", "SHBDUO", "SHDM", "SHRGBW"}
	for _, model := range lightModels {
		if len(info.Model) >= len(model) && info.Model[:len(model)] == model {
			isLightDevice = true
			break
		}
	}

	if !isLightDevice {
		t.Skip("Device does not support lights")
	}

	light := device.Light(0)

	lightStatus, err := light.GetStatus(ctx)
	if err != nil {
		t.Fatalf("Light.GetStatus() error = %v", err)
	}

	t.Logf("Light 0 Status: IsOn=%v, Brightness=%d", lightStatus.IsOn, lightStatus.Brightness)
}
