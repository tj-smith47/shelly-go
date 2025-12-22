package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// TestNewRoller tests roller creation.
func TestNewRoller(t *testing.T) {
	mt := newMockTransport()
	roller := NewRoller(mt, 0)

	if roller == nil {
		t.Fatal("expected roller to be created")
	}

	if roller.ID() != 0 {
		t.Errorf("expected ID 0, got %d", roller.ID())
	}
}

// TestRollerGetStatus tests roller status retrieval.
func TestRollerGetStatus(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/roller/0", &RollerStatus{
		State:         "stop",
		Source:        "http",
		Power:         0,
		CurrentPos:    75,
		LastDirection: "open",
		Positioning:   true,
	})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	status, err := roller.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
}

// TestRollerOpen tests opening roller.
func TestRollerOpen(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/roller/0?go=open", map[string]string{"state": "open"})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.Open(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mt.GetCalls()
	if len(calls) != 1 || calls[0] != "/roller/0?go=open" {
		t.Errorf("expected /roller/0?go=open call, got %v", calls)
	}
}

// TestRollerClose tests closing roller.
func TestRollerClose(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/roller/0?go=close", map[string]string{"state": "close"})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.Close(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerStop tests stopping roller.
func TestRollerStop(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/roller/0?go=stop", map[string]string{"state": "stop"})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.Stop(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerGoToPosition tests position control.
func TestRollerGoToPosition(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/roller/0?go=to_pos&roller_pos=50", map[string]int{"current_pos": 50})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.GoToPosition(ctx, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerGoToPositionInvalid tests invalid position.
func TestRollerGoToPositionInvalid(t *testing.T) {
	mt := newMockTransport()
	roller := NewRoller(mt, 0)
	ctx := context.Background()

	// Test below range
	err := roller.GoToPosition(ctx, -1)
	if err == nil {
		t.Error("expected error for position -1")
	}

	// Test above range
	err = roller.GoToPosition(ctx, 101)
	if err == nil {
		t.Error("expected error for position 101")
	}
}

// TestRollerOpenForDuration tests timed open.
func TestRollerOpenForDuration(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/roller/0?go=open&duration=5", map[string]string{"state": "open"})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.OpenForDuration(ctx, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerCloseForDuration tests timed close.
func TestRollerCloseForDuration(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/roller/0?go=close&duration=3", map[string]string{"state": "close"})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.CloseForDuration(ctx, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerCalibrate tests calibration.
func TestRollerCalibrate(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/roller/0/calibrate", map[string]bool{"calibrating": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.Calibrate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerGetConfig tests config retrieval.
func TestRollerGetConfig(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/roller/0", &RollerConfig{
		MaxTime:      30,
		DefaultState: "stop",
		InputMode:    "openclose",
		Positioning:  true,
	})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	config, err := roller.GetConfig(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.MaxTime != 30 {
		t.Errorf("expected max time 30, got %f", config.MaxTime)
	}

	if !config.Positioning {
		t.Error("expected positioning enabled")
	}
}

// TestRollerSetConfig tests config update.
func TestRollerSetConfig(t *testing.T) {
	mt := newMockTransport()
	// url.Values encodes parameters in alphabetical order
	mt.SetResponse("/settings/roller/0?default_state=stop&maxtime=25&positioning=true", map[string]bool{"ok": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	config := &RollerConfig{
		MaxTime:      25,
		DefaultState: "stop",
		Positioning:  true,
	}

	err := roller.SetConfig(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerSetMaxTime tests max time setting.
func TestRollerSetMaxTime(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/roller/0?maxtime=20", map[string]bool{"ok": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetMaxTime(ctx, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerSetDefaultState tests default state setting.
func TestRollerSetDefaultState(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/roller/0?default_state=last", map[string]bool{"ok": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetDefaultState(ctx, "last")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerSetInputMode tests input mode setting.
func TestRollerSetInputMode(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/roller/0?input_mode=onebutton", map[string]bool{"ok": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetInputMode(ctx, "onebutton")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerSetButtonType tests button type setting.
func TestRollerSetButtonType(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/roller/0?btn_type=detached", map[string]bool{"ok": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetButtonType(ctx, "detached")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerEnablePositioning tests positioning enable/disable.
func TestRollerEnablePositioning(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/roller/0?positioning=true", map[string]bool{"ok": true})
	mt.SetResponse("/settings/roller/0?positioning=false", map[string]bool{"ok": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	// Enable
	err := roller.EnablePositioning(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Disable
	err = roller.EnablePositioning(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerSetObstacleDetection tests obstacle detection config.
func TestRollerSetObstacleDetection(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/roller/0?obstacle_mode=both&obstacle_action=reverse&obstacle_power=100&obstacle_delay=1", map[string]bool{"ok": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetObstacleDetection(ctx, "both", "reverse", 100, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerSetSafetySwitch tests safety switch config.
func TestRollerSetSafetySwitch(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/roller/0?safety_mode=while_opening&safety_action=stop", map[string]bool{"ok": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetSafetySwitch(ctx, "while_opening", "stop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerSwapDirection tests direction swap.
func TestRollerSwapDirection(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/roller/0?swap=true", map[string]bool{"ok": true})
	mt.SetResponse("/settings/roller/0?swap=false", map[string]bool{"ok": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	// Swap
	err := roller.SwapDirection(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unswap
	err = roller.SwapDirection(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerSwapInputs tests input swap.
func TestRollerSwapInputs(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/roller/0?swap_inputs=true", map[string]bool{"ok": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SwapInputs(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerErrors tests error handling.
func TestRollerErrors(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/roller/0?go=open", errors.New("motor fault"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.Open(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerCloseError tests Close error handling.
func TestRollerCloseError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/roller/0?go=close", errors.New("motor blocked"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.Close(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerStopError tests Stop error handling.
func TestRollerStopError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/roller/0?go=stop", errors.New("device busy"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.Stop(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerGoToPositionError tests GoToPosition error handling.
func TestRollerGoToPositionError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/roller/0?go=to_pos&roller_pos=50", errors.New("calibration needed"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.GoToPosition(ctx, 50)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerGetStatusError tests GetStatus error handling.
func TestRollerGetStatusError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/roller/0", errors.New("device offline"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	_, err := roller.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerGetConfigError tests GetConfig error handling.
func TestRollerGetConfigError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/roller/0", errors.New("unauthorized"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	_, err := roller.GetConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerSetConfigFull tests SetConfig with all fields.
func TestRollerSetConfigFull(t *testing.T) {
	mt := newMockTransport()
	// url.Values encodes parameters in alphabetical order
	mt.SetResponse("/settings/roller/0?btn_type=momentary&default_state=stop&input_mode=dual&maxtime=30&maxtime_close=28&maxtime_open=25&obstacle_action=stop&obstacle_delay=2&obstacle_mode=both&obstacle_power=200&positioning=true&safety_action=pause&safety_mode=while_opening&swap=true&swap_inputs=true", map[string]bool{"ok": true})

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	config := &RollerConfig{
		DefaultState:   "stop",
		Swap:           true,
		SwapInputs:     true,
		InputMode:      "dual",
		BtnType:        "momentary",
		Positioning:    true,
		MaxTime:        30,
		MaxTimeOpen:    25,
		MaxTimeClose:   28,
		ObstacleMode:   "both",
		ObstacleAction: "stop",
		ObstaclePower:  200,
		ObstacleDelay:  2,
		SafetyMode:     "while_opening",
		SafetyAction:   "pause",
	}

	err := roller.SetConfig(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRollerSetConfigError tests SetConfig error handling.
func TestRollerSetConfigError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/roller/0?default_state=stop", errors.New("invalid config"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetConfig(ctx, &RollerConfig{DefaultState: "stop"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerOpenForDurationError tests OpenForDuration error handling.
func TestRollerOpenForDurationError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/roller/0?go=open&duration=5", errors.New("motor fault"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.OpenForDuration(ctx, 5)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerCloseForDurationError tests CloseForDuration error handling.
func TestRollerCloseForDurationError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/roller/0?go=close&duration=5", errors.New("motor fault"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.CloseForDuration(ctx, 5)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerCalibrateError tests Calibrate error handling.
func TestRollerCalibrateError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/roller/0?calibrate=true", errors.New("calibration failed"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.Calibrate(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerGetStatusInvalidJSON tests GetStatus with invalid JSON response.
func TestRollerGetStatusInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/roller/0"] = json.RawMessage(`{invalid`)

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	_, err := roller.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestRollerGetConfigInvalidJSON tests GetConfig with invalid JSON response.
func TestRollerGetConfigInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/settings/roller/0"] = json.RawMessage(`{invalid`)

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	_, err := roller.GetConfig(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestRollerSetMaxTimeError tests SetMaxTime error handling.
func TestRollerSetMaxTimeError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/roller/0?maxtime=60", errors.New("set failed"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetMaxTime(ctx, 60)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerSetDefaultStateError tests SetDefaultState error handling.
func TestRollerSetDefaultStateError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/roller/0?default_state=stop", errors.New("set failed"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetDefaultState(ctx, "stop")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerSetInputModeError tests SetInputMode error handling.
func TestRollerSetInputModeError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/roller/0?input_mode=openclose", errors.New("set failed"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetInputMode(ctx, "openclose")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerSetButtonTypeError tests SetButtonType error handling.
func TestRollerSetButtonTypeError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/roller/0?btn_type=toggle", errors.New("set failed"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetButtonType(ctx, "toggle")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerEnablePositioningError tests EnablePositioning error handling.
func TestRollerEnablePositioningError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/roller/0?positioning=true", errors.New("set failed"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.EnablePositioning(ctx, true)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerSetObstacleDetectionError tests SetObstacleDetection error handling.
func TestRollerSetObstacleDetectionError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/roller/0?obstacle_mode=while_opening&obstacle_action=stop&obstacle_power=100&obstacle_delay=0", errors.New("set failed"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetObstacleDetection(ctx, "while_opening", "stop", 100, 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerSetSafetySwitchError tests SetSafetySwitch error handling.
func TestRollerSetSafetySwitchError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/roller/0?safety_mode=while_opening&safety_action=stop", errors.New("set failed"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SetSafetySwitch(ctx, "while_opening", "stop")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerSwapDirectionError tests SwapDirection error handling.
func TestRollerSwapDirectionError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/roller/0?swap=true", errors.New("set failed"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SwapDirection(ctx, true)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRollerSwapInputsError tests SwapInputs error handling.
func TestRollerSwapInputsError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/roller/0?swap_inputs=true", errors.New("set failed"))

	roller := NewRoller(mt, 0)
	ctx := context.Background()

	err := roller.SwapInputs(ctx, true)
	if err == nil {
		t.Fatal("expected error")
	}
}
