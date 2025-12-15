package gen1

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// mockTransport is a mock transport for testing.
type mockTransport struct {
	responses map[string]json.RawMessage
	errors    map[string]error
	calls     []string
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		responses: make(map[string]json.RawMessage),
		errors:    make(map[string]error),
		calls:     make([]string, 0),
	}
}

func (m *mockTransport) SetResponse(path string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err) // Test helper - should never fail
	}
	m.responses[path] = b
}

func (m *mockTransport) SetError(path string, err error) {
	m.errors[path] = err
}

func (m *mockTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	m.calls = append(m.calls, method)

	if err, ok := m.errors[method]; ok {
		return nil, err
	}

	if resp, ok := m.responses[method]; ok {
		return resp, nil
	}

	return nil, errors.New("no mock response for: " + method)
}

func (m *mockTransport) Close() error {
	return nil
}

func (m *mockTransport) GetCalls() []string {
	return m.calls
}

// TestNewDevice tests device creation.
func TestNewDevice(t *testing.T) {
	mt := newMockTransport()
	device := NewDevice(mt)

	if device == nil {
		t.Fatal("expected device to be created")
	}

	if device.transport != mt {
		t.Error("expected transport to be set")
	}
}

// TestGetDeviceInfo tests device info retrieval.
func TestGetDeviceInfo(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/shelly", &DeviceInfo{
		Type: "SHSW-1",
		MAC:  "AABBCCDDEEFF",
		Auth: false,
		FW:   "20210115-123456/v1.9.5@abc123",
	})

	device := NewDevice(mt)
	ctx := context.Background()

	info, err := device.GetDeviceInfo(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Model != "SHSW-1" {
		t.Errorf("expected model SHSW-1, got %s", info.Model)
	}

	if info.MAC != "AABBCCDDEEFF" {
		t.Errorf("expected MAC AABBCCDDEEFF, got %s", info.MAC)
	}

	if info.Generation != types.Gen1 {
		t.Errorf("expected generation Gen1, got %v", info.Generation)
	}

	// Test caching - second call should not make another request
	calls := mt.GetCalls()
	info2, err := device.GetDeviceInfo(ctx)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}

	if info2.Model != info.Model {
		t.Error("expected cached response")
	}

	// Verify only one call was made
	if len(mt.GetCalls()) != len(calls) {
		t.Error("expected no additional transport calls for cached info")
	}
}

// TestGetDeviceInfoError tests device info error handling.
func TestGetDeviceInfoError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/shelly", errors.New("connection refused"))

	device := NewDevice(mt)
	ctx := context.Background()

	_, err := device.GetDeviceInfo(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGetDeviceInfoInvalidJSON tests invalid JSON handling.
func TestGetDeviceInfoInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/shelly"] = json.RawMessage(`{invalid}`)

	device := NewDevice(mt)
	ctx := context.Background()

	_, err := device.GetDeviceInfo(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestGetStatus tests device status retrieval.
func TestGetStatus(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/status", &Status{
		WiFiSta: &WiFiStatus{
			Connected: true,
			SSID:      "TestNetwork",
			IP:        "192.168.1.100",
			RSSI:      -65,
		},
		Time:      "12:34",
		UnixTime:  1699295403,
		HasUpdate: false,
		MAC:       "AABBCCDDEEFF",
		Relays: []RelayStatus{
			{IsOn: true, Source: "input"},
		},
	})

	device := NewDevice(mt)
	ctx := context.Background()

	statusI, err := device.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	status, ok := statusI.(*Status)
	if !ok {
		t.Fatalf("expected *Status, got %T", statusI)
	}
	if !status.WiFiSta.Connected {
		t.Error("expected WiFi connected")
	}

	if status.WiFiSta.SSID != "TestNetwork" {
		t.Errorf("expected SSID TestNetwork, got %s", status.WiFiSta.SSID)
	}

	if len(status.Relays) != 1 || !status.Relays[0].IsOn {
		t.Error("expected relay to be on")
	}
}

// TestGetFullStatus tests GetFullStatus method.
func TestGetFullStatus(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/status", &Status{
		MAC:  "AABBCCDDEEFF",
		Time: "10:00",
	})

	device := NewDevice(mt)
	ctx := context.Background()

	status, err := device.GetFullStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.MAC != "AABBCCDDEEFF" {
		t.Errorf("expected MAC AABBCCDDEEFF, got %s", status.MAC)
	}
}

// TestGetStatusError tests status error handling.
func TestGetStatusError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/status", errors.New("timeout"))

	device := NewDevice(mt)
	ctx := context.Background()

	_, err := device.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGetConfig tests device config retrieval.
func TestGetConfig(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings", &Settings{
		Device: &DeviceSettings{
			Type: "SHSW-1",
			MAC:  "AABBCCDDEEFF",
		},
		Name: "TestDevice",
		FW:   "v1.9.5",
	})

	device := NewDevice(mt)
	ctx := context.Background()

	configI, err := device.GetConfig(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config, ok := configI.(*Settings)
	if !ok {
		t.Fatalf("expected *Settings, got %T", configI)
	}
	if config.Name != "TestDevice" {
		t.Errorf("expected name TestDevice, got %s", config.Name)
	}
}

// TestGetSettings tests GetSettings method.
func TestGetSettings(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings", &Settings{
		Name: "MyShelly",
		Mode: "relay",
	})

	device := NewDevice(mt)
	ctx := context.Background()

	settings, err := device.GetSettings(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if settings.Name != "MyShelly" {
		t.Errorf("expected name MyShelly, got %s", settings.Name)
	}

	if settings.Mode != "relay" {
		t.Errorf("expected mode relay, got %s", settings.Mode)
	}
}

// TestReboot tests device reboot.
func TestReboot(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/reboot", map[string]bool{"ok": true})

	device := NewDevice(mt)
	ctx := context.Background()

	err := device.Reboot(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mt.GetCalls()
	if len(calls) != 1 || calls[0] != "/reboot" {
		t.Errorf("expected /reboot call, got %v", calls)
	}
}

// TestRebootError tests reboot error handling.
func TestRebootError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/reboot", errors.New("device busy"))

	device := NewDevice(mt)
	ctx := context.Background()

	err := device.Reboot(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGeneration tests Generation method.
func TestGeneration(t *testing.T) {
	mt := newMockTransport()
	device := NewDevice(mt)

	gen := device.Generation()
	if gen != types.Gen1 {
		t.Errorf("expected Gen1, got %v", gen)
	}
}

// TestClose tests device close.
func TestClose(t *testing.T) {
	mt := newMockTransport()
	device := NewDevice(mt)

	err := device.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestFactoryReset tests factory reset.
func TestFactoryReset(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/reset", map[string]bool{"ok": true})

	device := NewDevice(mt)
	ctx := context.Background()

	err := device.FactoryReset(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCheckForUpdate tests update check.
func TestCheckForUpdate(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/ota/check", &UpdateInfo{
		HasUpdate:  true,
		NewVersion: "v1.10.0",
	})

	device := NewDevice(mt)
	ctx := context.Background()

	info, err := device.CheckForUpdate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !info.HasUpdate {
		t.Error("expected has_update to be true")
	}

	if info.NewVersion != "v1.10.0" {
		t.Errorf("expected version v1.10.0, got %s", info.NewVersion)
	}
}

// TestUpdate tests firmware update.
func TestUpdate(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/ota?update=true", map[string]bool{"ok": true})
	mt.SetResponse("/ota?url=http://example.com/fw.bin", map[string]bool{"ok": true})

	device := NewDevice(mt)
	ctx := context.Background()

	// Test update without URL
	err := device.Update(ctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test update with URL
	err = device.Update(ctx, "http://example.com/fw.bin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSetName tests setting device name.
func TestSetName(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings?name=NewName", map[string]bool{"ok": true})

	device := NewDevice(mt)
	ctx := context.Background()

	err := device.SetName(ctx, "NewName")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSetTimezone tests setting timezone.
func TestSetTimezone(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings?timezone=America/New_York", map[string]bool{"ok": true})

	device := NewDevice(mt)
	ctx := context.Background()

	err := device.SetTimezone(ctx, "America/New_York")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSetLocation tests setting location.
func TestSetLocation(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings?lat=40.712800&lng=-74.006000", map[string]bool{"ok": true})

	device := NewDevice(mt)
	ctx := context.Background()

	err := device.SetLocation(ctx, 40.7128, -74.006)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGetDebugLog tests debug log retrieval.
func TestGetDebugLog(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/debug/log"] = json.RawMessage(`"[INFO] Boot complete\n[DEBUG] WiFi connected"`)

	device := NewDevice(mt)
	ctx := context.Background()

	log, err := device.GetDebugLog(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if log == "" {
		t.Error("expected non-empty log")
	}
}

// TestTransport tests Transport getter.
func TestTransport(t *testing.T) {
	mt := newMockTransport()
	device := NewDevice(mt)

	if device.Transport() != mt {
		t.Error("expected transport to be returned")
	}
}

// TestCall tests raw Call method.
func TestCall(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/custom/endpoint", map[string]string{"custom": "response"})

	device := NewDevice(mt)
	ctx := context.Background()

	resp, err := device.Call(ctx, "/custom/endpoint")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Error("expected response")
	}
}

// TestClearCache tests cache clearing.
func TestClearCache(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/shelly", &DeviceInfo{
		Type: "SHSW-1",
		MAC:  "AABBCCDDEEFF",
	})

	device := NewDevice(mt)
	ctx := context.Background()

	// First call - caches info
	_, err := device.GetDeviceInfo(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	initialCalls := len(mt.GetCalls())

	// Clear cache
	device.ClearCache()

	// Second call should fetch again
	_, err = device.GetDeviceInfo(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mt.GetCalls()) != initialCalls+1 {
		t.Error("expected additional call after cache clear")
	}
}

// TestComponentAccessors tests component accessor methods.
func TestComponentAccessors(t *testing.T) {
	mt := newMockTransport()
	device := NewDevice(mt)

	// Test Relay accessor
	relay := device.Relay(0)
	if relay == nil {
		t.Error("expected relay accessor")
	}
	if relay.ID() != 0 {
		t.Errorf("expected relay ID 0, got %d", relay.ID())
	}

	// Test Roller accessor
	roller := device.Roller(1)
	if roller == nil {
		t.Error("expected roller accessor")
	}
	if roller.ID() != 1 {
		t.Errorf("expected roller ID 1, got %d", roller.ID())
	}

	// Test Light accessor
	light := device.Light(0)
	if light == nil {
		t.Error("expected light accessor")
	}
	if light.ID() != 0 {
		t.Errorf("expected light ID 0, got %d", light.ID())
	}

	// Test Color accessor
	color := device.Color(0)
	if color == nil {
		t.Error("expected color accessor")
	}
	if color.ID() != 0 {
		t.Errorf("expected color ID 0, got %d", color.ID())
	}

	// Test White accessor
	white := device.White(0)
	if white == nil {
		t.Error("expected white accessor")
	}
	if white.ID() != 0 {
		t.Errorf("expected white ID 0, got %d", white.ID())
	}

	// Test Meter accessor
	meter := device.Meter(0)
	if meter == nil {
		t.Error("expected meter accessor")
	}
	if meter.ID() != 0 {
		t.Errorf("expected meter ID 0, got %d", meter.ID())
	}

	// Test EMeter accessor
	emeter := device.EMeter(0)
	if emeter == nil {
		t.Error("expected emeter accessor")
	}
	if emeter.ID() != 0 {
		t.Errorf("expected emeter ID 0, got %d", emeter.ID())
	}

	// Test Input accessor
	input := device.Input(0)
	if input == nil {
		t.Error("expected input accessor")
	}
	if input.ID() != 0 {
		t.Errorf("expected input ID 0, got %d", input.ID())
	}
}

// TestContextCancellation tests context cancellation handling.
func TestContextCancellation(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/status", &Status{MAC: "TEST"})

	device := NewDevice(mt)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Sleep to trigger timeout
	time.Sleep(5 * time.Millisecond)

	// The mock doesn't actually check context, but this tests the pattern
	_, err := device.GetStatus(ctx)
	// In a real implementation, this would return a context error
	// For the mock, it still works
	if err != nil {
		// Context cancellation would cause this
		t.Logf("context error as expected: %v", err)
	}
}

// TestGetFullStatusError tests GetFullStatus error handling.
func TestGetFullStatusError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/status", errors.New("connection timeout"))

	device := NewDevice(mt)
	ctx := context.Background()

	_, err := device.GetFullStatus(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGetSettingsError tests GetSettings error handling.
func TestGetSettingsError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings", errors.New("unauthorized"))

	device := NewDevice(mt)
	ctx := context.Background()

	_, err := device.GetSettings(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestFactoryResetError tests factory reset error handling.
func TestFactoryResetError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/reset", errors.New("device busy"))

	device := NewDevice(mt)
	ctx := context.Background()

	err := device.FactoryReset(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestCheckForUpdateError tests update check error handling.
func TestCheckForUpdateError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/ota/check", errors.New("no internet"))

	device := NewDevice(mt)
	ctx := context.Background()

	_, err := device.CheckForUpdate(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestUpdateError tests update error handling.
func TestUpdateError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/ota?update=true", errors.New("update failed"))

	device := NewDevice(mt)
	ctx := context.Background()

	err := device.Update(ctx, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSetNameError tests set name error handling.
func TestSetNameError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings?name=Test", errors.New("invalid name"))

	device := NewDevice(mt)
	ctx := context.Background()

	err := device.SetName(ctx, "Test")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSetTimezoneError tests set timezone error handling.
func TestSetTimezoneError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings?timezone=Invalid/Zone", errors.New("invalid timezone"))

	device := NewDevice(mt)
	ctx := context.Background()

	err := device.SetTimezone(ctx, "Invalid/Zone")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSetLocationError tests set location error handling.
func TestSetLocationError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings?lat=91.000000&lng=0.000000", errors.New("invalid coordinates"))

	device := NewDevice(mt)
	ctx := context.Background()

	err := device.SetLocation(ctx, 91.0, 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGetDebugLogError tests debug log error handling.
func TestGetDebugLogError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/debug/log", errors.New("debug not enabled"))

	device := NewDevice(mt)
	ctx := context.Background()

	_, err := device.GetDebugLog(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGetConfigError tests GetConfig error handling.
func TestGetConfigError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings", errors.New("unauthorized"))

	device := NewDevice(mt)
	ctx := context.Background()

	_, err := device.GetConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGetFullStatusInvalidJSON tests GetFullStatus with invalid JSON response.
func TestGetFullStatusInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/status"] = json.RawMessage(`{invalid`)

	device := NewDevice(mt)
	ctx := context.Background()

	_, err := device.GetFullStatus(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestGetSettingsInvalidJSON tests GetSettings with invalid JSON response.
func TestGetSettingsInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/settings"] = json.RawMessage(`{invalid`)

	device := NewDevice(mt)
	ctx := context.Background()

	_, err := device.GetSettings(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestCheckForUpdateInvalidJSON tests CheckForUpdate with invalid JSON response.
func TestCheckForUpdateInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/ota/check"] = json.RawMessage(`{invalid`)

	device := NewDevice(mt)
	ctx := context.Background()

	_, err := device.CheckForUpdate(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
