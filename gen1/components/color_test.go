package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// TestNewColor tests color creation.
func TestNewColor(t *testing.T) {
	mt := newMockTransport()
	color := NewColor(mt, 0)

	if color == nil {
		t.Fatal("expected color to be created")
	}

	if color.ID() != 0 {
		t.Errorf("expected ID 0, got %d", color.ID())
	}
}

// TestColorGetStatus tests color status retrieval.
func TestColorGetStatus(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0", &ColorStatus{
		IsOn:   true,
		Mode:   "color",
		Red:    255,
		Green:  128,
		Blue:   0,
		White:  0,
		Gain:   80,
		Effect: 0,
	})

	color := NewColor(mt, 0)
	ctx := context.Background()

	status, err := color.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.IsOn {
		t.Error("expected color on")
	}

	if status.Red != 255 {
		t.Errorf("expected red 255, got %d", status.Red)
	}

	if status.Green != 128 {
		t.Errorf("expected green 128, got %d", status.Green)
	}
}

// TestColorTurnOn tests turning color on.
func TestColorTurnOn(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?turn=on", map[string]bool{"ison": true})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.TurnOn(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorTurnOff tests turning color off.
func TestColorTurnOff(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?turn=off", map[string]bool{"ison": false})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.TurnOff(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorToggle tests toggling color.
func TestColorToggle(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?turn=toggle", map[string]bool{"ison": true})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.Toggle(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorSet tests Set method.
func TestColorSet(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?turn=on", map[string]bool{"ison": true})
	mt.SetResponse("/color/0?turn=off", map[string]bool{"ison": false})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.Set(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = color.Set(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorSetRGB tests RGB setting.
func TestColorSetRGB(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?red=255&green=0&blue=0", map[string]bool{"ok": true})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetRGB(ctx, 255, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorSetRGBInvalid tests invalid RGB values.
func TestColorSetRGBInvalid(t *testing.T) {
	mt := newMockTransport()
	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetRGB(ctx, 256, 0, 0)
	if err == nil {
		t.Error("expected error for red 256")
	}

	err = color.SetRGB(ctx, 0, -1, 0)
	if err == nil {
		t.Error("expected error for green -1")
	}
}

// TestColorSetRGBW tests RGBW setting.
func TestColorSetRGBW(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?red=100&green=150&blue=200&white=50", map[string]bool{"ok": true})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetRGBW(ctx, 100, 150, 200, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorSetRGBWInvalid tests invalid RGBW values.
func TestColorSetRGBWInvalid(t *testing.T) {
	mt := newMockTransport()
	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetRGBW(ctx, 0, 0, 0, 300)
	if err == nil {
		t.Error("expected error for white 300")
	}
}

// TestColorSetGain tests gain setting.
func TestColorSetGain(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?gain=75", map[string]int{"gain": 75})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetGain(ctx, 75)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorSetGainInvalid tests invalid gain.
func TestColorSetGainInvalid(t *testing.T) {
	mt := newMockTransport()
	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetGain(ctx, -1)
	if err == nil {
		t.Error("expected error for gain -1")
	}

	err = color.SetGain(ctx, 101)
	if err == nil {
		t.Error("expected error for gain 101")
	}
}

// TestColorSetWhiteChannel tests white channel setting.
func TestColorSetWhiteChannel(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?white=128", map[string]int{"white": 128})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetWhiteChannel(ctx, 128)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorSetWhiteChannelInvalid tests invalid white channel.
func TestColorSetWhiteChannelInvalid(t *testing.T) {
	mt := newMockTransport()
	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetWhiteChannel(ctx, 256)
	if err == nil {
		t.Error("expected error for white 256")
	}
}

// TestColorSetEffect tests effect setting.
func TestColorSetEffect(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?effect=2", map[string]int{"effect": 2})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetEffect(ctx, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorSetTransition tests transition setting.
func TestColorSetTransition(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?transition=1000", map[string]int{"transition": 1000})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetTransition(ctx, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorTurnOnWithRGB tests turn on with RGB.
func TestColorTurnOnWithRGB(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?turn=on&red=255&green=100&blue=50&gain=90", map[string]bool{"ison": true})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.TurnOnWithRGB(ctx, 255, 100, 50, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorTurnOnWithRGBInvalid tests invalid turn on with RGB.
func TestColorTurnOnWithRGBInvalid(t *testing.T) {
	mt := newMockTransport()
	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.TurnOnWithRGB(ctx, 300, 0, 0, 50)
	if err == nil {
		t.Error("expected error for red 300")
	}

	err = color.TurnOnWithRGB(ctx, 100, 100, 100, 150)
	if err == nil {
		t.Error("expected error for gain 150")
	}
}

// TestColorTurnOnForDuration tests timed on.
func TestColorTurnOnForDuration(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/color/0?turn=on&timer=300", map[string]bool{"ison": true})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.TurnOnForDuration(ctx, 300)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorGetConfig tests config retrieval.
func TestColorGetConfig(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/color/0", &ColorConfig{
		Name:         "RGB Strip",
		DefaultState: "last",
		AutoOff:      3600,
	})

	color := NewColor(mt, 0)
	ctx := context.Background()

	config, err := color.GetConfig(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Name != "RGB Strip" {
		t.Errorf("expected name RGB Strip, got %s", config.Name)
	}
}

// TestColorSetConfig tests config update.
func TestColorSetConfig(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/color/0?name=LEDs&default_state=off", map[string]bool{"ok": true})

	color := NewColor(mt, 0)
	ctx := context.Background()

	config := &ColorConfig{
		Name:         "LEDs",
		DefaultState: "off",
	}

	err := color.SetConfig(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorSetName tests name setting.
func TestColorSetName(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/color/0?name=Ambient", map[string]bool{"ok": true})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetName(ctx, "Ambient")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorSetDefaultState tests default state setting.
func TestColorSetDefaultState(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/color/0?default_state=last", map[string]bool{"ok": true})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetDefaultState(ctx, "last")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorSetAutoOn tests auto-on setting.
func TestColorSetAutoOn(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/color/0?auto_on=120", map[string]bool{"ok": true})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetAutoOn(ctx, 120)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorSetAutoOff tests auto-off setting.
func TestColorSetAutoOff(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/color/0?auto_off=7200", map[string]bool{"ok": true})

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetAutoOff(ctx, 7200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestColorErrors tests error handling.
func TestColorErrors(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0?turn=on", errors.New("device busy"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.TurnOn(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorTurnOffError tests TurnOff error handling.
func TestColorTurnOffError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0?turn=off", errors.New("device busy"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.TurnOff(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorToggleError tests Toggle error handling.
func TestColorToggleError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0?turn=toggle", errors.New("device busy"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.Toggle(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorGetStatusError tests GetStatus error handling.
func TestColorGetStatusError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0", errors.New("offline"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	_, err := color.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorGetConfigError tests GetConfig error handling.
func TestColorGetConfigError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/color/0", errors.New("unauthorized"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	_, err := color.GetConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorSetRGBError tests SetRGB error handling.
func TestColorSetRGBError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0?red=255&green=0&blue=0", errors.New("invalid"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetRGB(ctx, 255, 0, 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorSetRGBWError tests SetRGBW error handling.
func TestColorSetRGBWError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0?red=100&green=100&blue=100&white=50", errors.New("invalid"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetRGBW(ctx, 100, 100, 100, 50)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorSetGainError tests SetGain error handling.
func TestColorSetGainError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0?gain=75", errors.New("invalid"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetGain(ctx, 75)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorSetWhiteChannelError tests SetWhiteChannel error handling.
func TestColorSetWhiteChannelError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0?white=128", errors.New("invalid"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetWhiteChannel(ctx, 128)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorSetEffectError tests SetEffect error handling.
func TestColorSetEffectError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0?effect=1", errors.New("not supported"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetEffect(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorSetTransitionError tests SetTransition error handling.
func TestColorSetTransitionError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0?transition=500", errors.New("invalid"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.SetTransition(ctx, 500)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorTurnOnWithRGBError tests TurnOnWithRGB error handling.
func TestColorTurnOnWithRGBError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0?turn=on&red=255&green=0&blue=0&gain=50", errors.New("invalid"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.TurnOnWithRGB(ctx, 255, 0, 0, 50)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorTurnOnForDurationError tests TurnOnForDuration error handling.
func TestColorTurnOnForDurationError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/color/0?turn=on&timer=60", errors.New("timer failed"))

	color := NewColor(mt, 0)
	ctx := context.Background()

	err := color.TurnOnForDuration(ctx, 60)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestColorGetStatusInvalidJSON tests GetStatus with invalid JSON response.
func TestColorGetStatusInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/color/0"] = json.RawMessage(`{invalid`)

	color := NewColor(mt, 0)
	ctx := context.Background()

	_, err := color.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestColorGetConfigInvalidJSON tests GetConfig with invalid JSON response.
func TestColorGetConfigInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/settings/color/0"] = json.RawMessage(`{invalid`)

	color := NewColor(mt, 0)
	ctx := context.Background()

	_, err := color.GetConfig(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
