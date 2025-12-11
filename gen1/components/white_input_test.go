package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// ==================== White Tests ====================

// TestNewWhite tests white creation.
func TestNewWhite(t *testing.T) {
	mt := newMockTransport()
	white := NewWhite(mt, 0)

	if white == nil {
		t.Fatal("expected white to be created")
	}

	if white.ID() != 0 {
		t.Errorf("expected ID 0, got %d", white.ID())
	}
}

// TestWhiteGetStatus tests white status retrieval.
func TestWhiteGetStatus(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/white/0", &WhiteStatus{
		IsOn:       true,
		Brightness: 80,
		Temp:       3500,
		Source:     "http",
	})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	status, err := white.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.IsOn {
		t.Error("expected white on")
	}

	if status.Brightness != 80 {
		t.Errorf("expected brightness 80, got %d", status.Brightness)
	}

	if status.Temp != 3500 {
		t.Errorf("expected temp 3500, got %d", status.Temp)
	}
}

// TestWhiteTurnOn tests turning white on.
func TestWhiteTurnOn(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/white/0?turn=on", map[string]bool{"ison": true})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOn(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteTurnOff tests turning white off.
func TestWhiteTurnOff(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/white/0?turn=off", map[string]bool{"ison": false})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOff(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteToggle tests toggling white.
func TestWhiteToggle(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/white/0?turn=toggle", map[string]bool{"ison": true})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.Toggle(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteSet tests Set method.
func TestWhiteSet(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/white/0?turn=on", map[string]bool{"ison": true})
	mt.SetResponse("/white/0?turn=off", map[string]bool{"ison": false})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.Set(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = white.Set(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteSetBrightness tests brightness setting.
func TestWhiteSetBrightness(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/white/0?brightness=60", map[string]int{"brightness": 60})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.SetBrightness(ctx, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteSetBrightnessInvalid tests invalid brightness.
func TestWhiteSetBrightnessInvalid(t *testing.T) {
	mt := newMockTransport()
	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.SetBrightness(ctx, -1)
	if err == nil {
		t.Error("expected error for brightness -1")
	}

	err = white.SetBrightness(ctx, 101)
	if err == nil {
		t.Error("expected error for brightness 101")
	}
}

// TestWhiteSetBrightnessWithTransition tests brightness with transition.
func TestWhiteSetBrightnessWithTransition(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/white/0?brightness=70&transition=750", map[string]int{"brightness": 70})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.SetBrightnessWithTransition(ctx, 70, 750)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteSetColorTemp tests color temperature setting.
func TestWhiteSetColorTemp(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/white/0?temp=4000", map[string]int{"temp": 4000})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.SetColorTemp(ctx, 4000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteTurnOnWithBrightness tests turn on with brightness.
func TestWhiteTurnOnWithBrightness(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/white/0?turn=on&brightness=50", map[string]bool{"ison": true})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOnWithBrightness(ctx, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteTurnOnWithBrightnessInvalid tests invalid brightness.
func TestWhiteTurnOnWithBrightnessInvalid(t *testing.T) {
	mt := newMockTransport()
	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOnWithBrightness(ctx, 150)
	if err == nil {
		t.Error("expected error for brightness 150")
	}
}

// TestWhiteTurnOnWithColorTemp tests turn on with color temp.
func TestWhiteTurnOnWithColorTemp(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/white/0?turn=on&temp=5000&brightness=90", map[string]bool{"ison": true})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOnWithColorTemp(ctx, 5000, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteTurnOnWithColorTempInvalid tests invalid brightness.
func TestWhiteTurnOnWithColorTempInvalid(t *testing.T) {
	mt := newMockTransport()
	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOnWithColorTemp(ctx, 3000, 200)
	if err == nil {
		t.Error("expected error for brightness 200")
	}
}

// TestWhiteTurnOnForDuration tests timed on.
func TestWhiteTurnOnForDuration(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/white/0?turn=on&timer=180", map[string]bool{"ison": true})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOnForDuration(ctx, 180)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteGetConfig tests config retrieval.
func TestWhiteGetConfig(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/white/0", &WhiteConfig{
		Name:         "Warm White",
		DefaultState: "last",
		AutoOff:      3600,
	})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	config, err := white.GetConfig(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Name != "Warm White" {
		t.Errorf("expected name Warm White, got %s", config.Name)
	}
}

// TestWhiteSetConfig tests config update.
func TestWhiteSetConfig(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/white/0?name=Cool&default_state=off&auto_off=900", map[string]bool{"ok": true})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	config := &WhiteConfig{
		Name:         "Cool",
		DefaultState: "off",
		AutoOff:      900,
	}

	err := white.SetConfig(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteSetName tests name setting.
func TestWhiteSetName(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/white/0?name=Daylight", map[string]bool{"ok": true})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.SetName(ctx, "Daylight")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteSetDefaultState tests default state setting.
func TestWhiteSetDefaultState(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/white/0?default_state=on", map[string]bool{"ok": true})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.SetDefaultState(ctx, "on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteSetAutoOn tests auto-on setting.
func TestWhiteSetAutoOn(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/white/0?auto_on=30", map[string]bool{"ok": true})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.SetAutoOn(ctx, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteSetAutoOff tests auto-off setting.
func TestWhiteSetAutoOff(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/white/0?auto_off=600", map[string]bool{"ok": true})

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.SetAutoOff(ctx, 600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWhiteErrors tests error handling.
func TestWhiteErrors(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/white/0?turn=on", errors.New("LED error"))

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOn(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// ==================== Input Tests ====================

// TestNewInput tests input creation.
func TestNewInput(t *testing.T) {
	mt := newMockTransport()
	input := NewInput(mt, 0)

	if input == nil {
		t.Fatal("expected input to be created")
	}

	if input.ID() != 0 {
		t.Errorf("expected ID 0, got %d", input.ID())
	}
}

// TestInputGetStatus tests input status retrieval.
func TestInputGetStatus(t *testing.T) {
	mt := newMockTransport()
	statusResp := struct {
		Inputs []InputStatus `json:"inputs"`
	}{
		Inputs: []InputStatus{
			{Input: 1, Event: "S", EventCnt: 42},
			{Input: 0, Event: "L", EventCnt: 10},
		},
	}
	mt.SetResponse("/status", statusResp)

	input := NewInput(mt, 0)
	ctx := context.Background()

	status, err := input.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Input != 1 {
		t.Errorf("expected input state 1, got %d", status.Input)
	}

	if status.Event != "S" {
		t.Errorf("expected event S, got %s", status.Event)
	}

	if status.EventCnt != 42 {
		t.Errorf("expected event count 42, got %d", status.EventCnt)
	}
}

// TestInputGetState tests GetState method.
func TestInputGetState(t *testing.T) {
	mt := newMockTransport()
	statusResp := struct {
		Inputs []InputStatus `json:"inputs"`
	}{
		Inputs: []InputStatus{{Input: 1}},
	}
	mt.SetResponse("/status", statusResp)

	input := NewInput(mt, 0)
	ctx := context.Background()

	state, err := input.GetState(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state != 1 {
		t.Errorf("expected state 1, got %d", state)
	}
}

// TestInputIsOn tests IsOn method.
func TestInputIsOn(t *testing.T) {
	mt := newMockTransport()

	// Test when input is on
	statusOn := struct {
		Inputs []InputStatus `json:"inputs"`
	}{
		Inputs: []InputStatus{{Input: 1}},
	}
	mt.SetResponse("/status", statusOn)

	input := NewInput(mt, 0)
	ctx := context.Background()

	on, err := input.IsOn(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !on {
		t.Error("expected input on")
	}

	// Test when input is off
	statusOff := struct {
		Inputs []InputStatus `json:"inputs"`
	}{
		Inputs: []InputStatus{{Input: 0}},
	}
	mt.SetResponse("/status", statusOff)

	on, err = input.IsOn(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if on {
		t.Error("expected input off")
	}
}

// TestInputGetLastEvent tests GetLastEvent method.
func TestInputGetLastEvent(t *testing.T) {
	mt := newMockTransport()
	statusResp := struct {
		Inputs []InputStatus `json:"inputs"`
	}{
		Inputs: []InputStatus{{Event: "SS", EventCnt: 15}},
	}
	mt.SetResponse("/status", statusResp)

	input := NewInput(mt, 0)
	ctx := context.Background()

	event, cnt, err := input.GetLastEvent(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event != "SS" {
		t.Errorf("expected event SS, got %s", event)
	}

	if cnt != 15 {
		t.Errorf("expected count 15, got %d", cnt)
	}
}

// TestInputGetEventCount tests GetEventCount method.
func TestInputGetEventCount(t *testing.T) {
	mt := newMockTransport()
	statusResp := struct {
		Inputs []InputStatus `json:"inputs"`
	}{
		Inputs: []InputStatus{{EventCnt: 100}},
	}
	mt.SetResponse("/status", statusResp)

	input := NewInput(mt, 0)
	ctx := context.Background()

	cnt, err := input.GetEventCount(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cnt != 100 {
		t.Errorf("expected count 100, got %d", cnt)
	}
}

// TestInputSetButtonType tests button type setting.
func TestInputSetButtonType(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?btn_type=edge", map[string]bool{"ok": true})

	input := NewInput(mt, 0)
	ctx := context.Background()

	err := input.SetButtonType(ctx, "edge")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestInputSetButtonReverse tests button reverse setting.
func TestInputSetButtonReverse(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?btn_reverse=true", map[string]bool{"ok": true})
	mt.SetResponse("/settings/relay/0?btn_reverse=false", map[string]bool{"ok": true})

	input := NewInput(mt, 0)
	ctx := context.Background()

	err := input.SetButtonReverse(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = input.SetButtonReverse(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestInputNotFound tests input not found error.
func TestInputNotFound(t *testing.T) {
	mt := newMockTransport()
	statusResp := struct {
		Inputs []InputStatus `json:"inputs"`
	}{
		Inputs: []InputStatus{{Input: 0}}, // Only one input
	}
	mt.SetResponse("/status", statusResp)

	input := NewInput(mt, 5) // Request non-existent input
	ctx := context.Background()

	_, err := input.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error for non-existent input")
	}
}

// TestInputErrors tests error handling.
func TestInputErrors(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/status", errors.New("device offline"))

	input := NewInput(mt, 0)
	ctx := context.Background()

	_, err := input.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestInputInvalidJSON tests invalid JSON handling.
func TestInputInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/status"] = json.RawMessage(`{invalid}`)

	input := NewInput(mt, 0)
	ctx := context.Background()

	_, err := input.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestEventType tests EventType constants and methods.
func TestEventType(t *testing.T) {
	tests := []struct {
		event EventType
		str   string
	}{
		{EventNone, "None"},
		{EventShortPress, "Short Press"},
		{EventLongPress, "Long Press"},
		{EventDoubleShortPress, "Double Short Press"},
		{EventTripleShortPress, "Triple Short Press"},
		{EventShortLongPress, "Short-Long Press"},
		{EventLongShortPress, "Long-Short Press"},
	}

	for _, tt := range tests {
		if tt.event.String() != tt.str {
			t.Errorf("expected %s, got %s", tt.str, tt.event.String())
		}
	}

	// Test unknown event
	unknown := EventType("X")
	if unknown.String() != "X" {
		t.Errorf("expected X, got %s", unknown.String())
	}
}

// TestParseEventType tests ParseEventType function.
func TestParseEventType(t *testing.T) {
	tests := []struct {
		input    string
		expected EventType
	}{
		{"", EventNone},
		{"S", EventShortPress},
		{"L", EventLongPress},
		{"SS", EventDoubleShortPress},
		{"SSS", EventTripleShortPress},
		{"SL", EventShortLongPress},
		{"LS", EventLongShortPress},
	}

	for _, tt := range tests {
		result := ParseEventType(tt.input)
		if result != tt.expected {
			t.Errorf("ParseEventType(%s): expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}

// ==================== Additional Error Tests ====================

// TestWhiteTurnOffError tests TurnOff error handling.
func TestWhiteTurnOffError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/white/0?turn=off", errors.New("device busy"))

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOff(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestWhiteToggleError tests Toggle error handling.
func TestWhiteToggleError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/white/0?turn=toggle", errors.New("device busy"))

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.Toggle(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestWhiteGetStatusError tests GetStatus error handling.
func TestWhiteGetStatusError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/white/0", errors.New("offline"))

	white := NewWhite(mt, 0)
	ctx := context.Background()

	_, err := white.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestWhiteGetConfigError tests GetConfig error handling.
func TestWhiteGetConfigError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/white/0", errors.New("unauthorized"))

	white := NewWhite(mt, 0)
	ctx := context.Background()

	_, err := white.GetConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestWhiteSetBrightnessError tests SetBrightness error handling.
func TestWhiteSetBrightnessError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/white/0?brightness=50", errors.New("invalid"))

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.SetBrightness(ctx, 50)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestWhiteSetBrightnessWithTransitionError tests SetBrightnessWithTransition error handling.
func TestWhiteSetBrightnessWithTransitionError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/white/0?brightness=50&transition=500", errors.New("invalid"))

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.SetBrightnessWithTransition(ctx, 50, 500)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestWhiteSetColorTempError tests SetColorTemp error handling.
func TestWhiteSetColorTempError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/white/0?temp=4000", errors.New("not supported"))

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.SetColorTemp(ctx, 4000)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestWhiteTurnOnWithBrightnessError tests TurnOnWithBrightness error handling.
func TestWhiteTurnOnWithBrightnessError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/white/0?turn=on&brightness=50", errors.New("invalid"))

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOnWithBrightness(ctx, 50)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestWhiteTurnOnWithColorTempError tests TurnOnWithColorTemp error handling.
func TestWhiteTurnOnWithColorTempError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/white/0?turn=on&temp=4000&brightness=50", errors.New("invalid"))

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOnWithColorTemp(ctx, 4000, 50)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestWhiteTurnOnForDurationError tests TurnOnForDuration error handling.
func TestWhiteTurnOnForDurationError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/white/0?turn=on&timer=60", errors.New("timer failed"))

	white := NewWhite(mt, 0)
	ctx := context.Background()

	err := white.TurnOnForDuration(ctx, 60)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestInputGetStateError tests GetState error handling.
func TestInputGetStateError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/status", errors.New("offline"))

	input := NewInput(mt, 0)
	ctx := context.Background()

	_, err := input.GetState(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestInputIsOnError tests IsOn error handling.
func TestInputIsOnError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/status", errors.New("offline"))

	input := NewInput(mt, 0)
	ctx := context.Background()

	_, err := input.IsOn(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestInputGetLastEventError tests GetLastEvent error handling.
func TestInputGetLastEventError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/status", errors.New("offline"))

	input := NewInput(mt, 0)
	ctx := context.Background()

	_, _, err := input.GetLastEvent(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestInputGetEventCountError tests GetEventCount error handling.
func TestInputGetEventCountError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/status", errors.New("offline"))

	input := NewInput(mt, 0)
	ctx := context.Background()

	_, err := input.GetEventCount(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestInputSetButtonTypeError tests SetButtonType error handling.
func TestInputSetButtonTypeError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?btn_type=edge", errors.New("invalid"))

	input := NewInput(mt, 0)
	ctx := context.Background()

	err := input.SetButtonType(ctx, "edge")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestInputSetButtonReverseError tests SetButtonReverse error handling.
func TestInputSetButtonReverseError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?btn_reverse=true", errors.New("invalid"))

	input := NewInput(mt, 0)
	ctx := context.Background()

	err := input.SetButtonReverse(ctx, true)
	if err == nil {
		t.Fatal("expected error")
	}
}
