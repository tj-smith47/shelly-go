package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// TestNewLight tests light creation.
func TestNewLight(t *testing.T) {
	mt := newMockTransport()
	light := NewLight(mt, 0)

	if light == nil {
		t.Fatal("expected light to be created")
	}

	if light.ID() != 0 {
		t.Errorf("expected ID 0, got %d", light.ID())
	}
}

// TestLightGetStatus tests light status retrieval.
func TestLightGetStatus(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/light/0", &LightStatus{
		IsOn:       true,
		Brightness: 75,
		Temp:       4000,
		Source:     "http",
	})

	light := NewLight(mt, 0)
	ctx := context.Background()

	status, err := light.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.IsOn {
		t.Error("expected light on")
	}

	if status.Brightness != 75 {
		t.Errorf("expected brightness 75, got %d", status.Brightness)
	}
}

// TestLightTurnOn tests turning light on.
func TestLightTurnOn(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/light/0?turn=on", map[string]bool{"ison": true})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.TurnOn(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightTurnOff tests turning light off.
func TestLightTurnOff(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/light/0?turn=off", map[string]bool{"ison": false})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.TurnOff(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightToggle tests toggling light.
func TestLightToggle(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/light/0?turn=toggle", map[string]bool{"ison": true})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.Toggle(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightSet tests Set method.
func TestLightSet(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/light/0?turn=on", map[string]bool{"ison": true})
	mt.SetResponse("/light/0?turn=off", map[string]bool{"ison": false})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.Set(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = light.Set(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightSetBrightness tests brightness setting.
func TestLightSetBrightness(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/light/0?brightness=50", map[string]int{"brightness": 50})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetBrightness(ctx, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightSetBrightnessInvalid tests invalid brightness.
func TestLightSetBrightnessInvalid(t *testing.T) {
	mt := newMockTransport()
	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetBrightness(ctx, -1)
	if err == nil {
		t.Error("expected error for brightness -1")
	}

	err = light.SetBrightness(ctx, 101)
	if err == nil {
		t.Error("expected error for brightness 101")
	}
}

// TestLightSetBrightnessWithTransition tests brightness with transition.
func TestLightSetBrightnessWithTransition(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/light/0?brightness=80&transition=500", map[string]int{"brightness": 80})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetBrightnessWithTransition(ctx, 80, 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightTurnOnWithBrightness tests turn on with brightness.
func TestLightTurnOnWithBrightness(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/light/0?turn=on&brightness=60", map[string]bool{"ison": true})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.TurnOnWithBrightness(ctx, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightTurnOnForDuration tests timed on.
func TestLightTurnOnForDuration(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/light/0?turn=on&timer=120", map[string]bool{"ison": true})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.TurnOnForDuration(ctx, 120)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightSetColorTemp tests color temperature setting.
func TestLightSetColorTemp(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/light/0?temp=3500", map[string]int{"temp": 3500})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetColorTemp(ctx, 3500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightGetConfig tests config retrieval.
func TestLightGetConfig(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/light/0", &LightConfig{
		Name:         "Desk Lamp",
		DefaultState: "last",
		AutoOff:      1800,
	})

	light := NewLight(mt, 0)
	ctx := context.Background()

	config, err := light.GetConfig(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Name != "Desk Lamp" {
		t.Errorf("expected name Desk Lamp, got %s", config.Name)
	}
}

// TestLightSetConfig tests config update.
func TestLightSetConfig(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/light/0?name=NewLight&default_state=off&auto_off=600", map[string]bool{"ok": true})

	light := NewLight(mt, 0)
	ctx := context.Background()

	config := &LightConfig{
		Name:         "NewLight",
		DefaultState: "off",
		AutoOff:      600,
	}

	err := light.SetConfig(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightSetName tests name setting.
func TestLightSetName(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/light/0?name=Kitchen", map[string]bool{"ok": true})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetName(ctx, "Kitchen")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightSetDefaultState tests default state setting.
func TestLightSetDefaultState(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/light/0?default_state=on", map[string]bool{"ok": true})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetDefaultState(ctx, "on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightSetButtonType tests button type setting.
func TestLightSetButtonType(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/light/0?btn_type=edge", map[string]bool{"ok": true})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetButtonType(ctx, "edge")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightSetAutoOn tests auto-on setting.
func TestLightSetAutoOn(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/light/0?auto_on=60", map[string]bool{"ok": true})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetAutoOn(ctx, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightSetAutoOff tests auto-off setting.
func TestLightSetAutoOff(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/light/0?auto_off=1800", map[string]bool{"ok": true})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetAutoOff(ctx, 1800)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightSetMinBrightness tests min brightness setting.
func TestLightSetMinBrightness(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/light/0?min_brightness=10", map[string]bool{"ok": true})

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetMinBrightness(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLightSetMinBrightnessInvalid tests invalid min brightness.
func TestLightSetMinBrightnessInvalid(t *testing.T) {
	mt := newMockTransport()
	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetMinBrightness(ctx, 0)
	if err == nil {
		t.Error("expected error for min brightness 0")
	}

	err = light.SetMinBrightness(ctx, 101)
	if err == nil {
		t.Error("expected error for min brightness 101")
	}
}

// TestLightErrors tests error handling.
func TestLightErrors(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/light/0?turn=on", errors.New("device offline"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.TurnOn(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightTurnOffError tests TurnOff error handling.
func TestLightTurnOffError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/light/0?turn=off", errors.New("device busy"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.TurnOff(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightToggleError tests Toggle error handling.
func TestLightToggleError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/light/0?turn=toggle", errors.New("device busy"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.Toggle(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightGetStatusError tests GetStatus error handling.
func TestLightGetStatusError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/light/0", errors.New("offline"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	_, err := light.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightGetConfigError tests GetConfig error handling.
func TestLightGetConfigError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/light/0", errors.New("unauthorized"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	_, err := light.GetConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightSetBrightnessError tests SetBrightness error handling.
func TestLightSetBrightnessError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/light/0?brightness=50", errors.New("invalid"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetBrightness(ctx, 50)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightSetBrightnessWithTransitionError tests SetBrightnessWithTransition error handling.
func TestLightSetBrightnessWithTransitionError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/light/0?brightness=50&transition=500", errors.New("invalid"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetBrightnessWithTransition(ctx, 50, 500)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightTurnOnWithBrightnessError tests TurnOnWithBrightness error handling.
func TestLightTurnOnWithBrightnessError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/light/0?turn=on&brightness=50", errors.New("invalid"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.TurnOnWithBrightness(ctx, 50)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightTurnOnForDurationError tests TurnOnForDuration error handling.
func TestLightTurnOnForDurationError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/light/0?turn=on&timer=60", errors.New("timer failed"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.TurnOnForDuration(ctx, 60)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightSetColorTempError tests SetColorTemp error handling.
func TestLightSetColorTempError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/light/0?temp=4000", errors.New("not supported"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetColorTemp(ctx, 4000)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightGetStatusInvalidJSON tests GetStatus with invalid JSON response.
func TestLightGetStatusInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/light/0"] = json.RawMessage(`{invalid`)

	light := NewLight(mt, 0)
	ctx := context.Background()

	_, err := light.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestLightGetConfigInvalidJSON tests GetConfig with invalid JSON response.
func TestLightGetConfigInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/settings/light/0"] = json.RawMessage(`{invalid`)

	light := NewLight(mt, 0)
	ctx := context.Background()

	_, err := light.GetConfig(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestLightSetNameError tests SetName error handling.
func TestLightSetNameError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/light/0?name=Test", errors.New("set failed"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetName(ctx, "Test")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightSetDefaultStateError tests SetDefaultState error handling.
func TestLightSetDefaultStateError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/light/0?default_state=on", errors.New("set failed"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetDefaultState(ctx, "on")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightSetButtonTypeError tests SetButtonType error handling.
func TestLightSetButtonTypeError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/light/0?btn_type=toggle", errors.New("set failed"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetButtonType(ctx, "toggle")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightSetAutoOnError tests SetAutoOn error handling.
func TestLightSetAutoOnError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/light/0?auto_on=60", errors.New("set failed"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetAutoOn(ctx, 60)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightSetAutoOffError tests SetAutoOff error handling.
func TestLightSetAutoOffError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/light/0?auto_off=120", errors.New("set failed"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetAutoOff(ctx, 120)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightSetMinBrightnessError tests SetMinBrightness error handling.
func TestLightSetMinBrightnessError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/light/0?min_brightness=10", errors.New("set failed"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	err := light.SetMinBrightness(ctx, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLightSetConfigError tests SetConfig error handling.
func TestLightSetConfigError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/light/0?name=Test", errors.New("set failed"))

	light := NewLight(mt, 0)
	ctx := context.Background()

	config := &LightConfig{
		Name: "Test",
	}
	err := light.SetConfig(ctx, config)
	if err == nil {
		t.Fatal("expected error")
	}
}
