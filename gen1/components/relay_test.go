package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/transport"
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

func (m *mockTransport) Call(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
	method := req.GetMethod()
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

// TestNewRelay tests relay creation.
func TestNewRelay(t *testing.T) {
	mt := newMockTransport()
	relay := NewRelay(mt, 0)

	if relay == nil {
		t.Fatal("expected relay to be created")
	}

	if relay.ID() != 0 {
		t.Errorf("expected ID 0, got %d", relay.ID())
	}
}

// TestRelayGetStatus tests relay status retrieval.
func TestRelayGetStatus(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/relay/0", &RelayStatus{
		IsOn:           true,
		HasTimer:       true,
		TimerDuration:  60,
		TimerRemaining: 30,
		Source:         "input",
	})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	status, err := relay.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.IsOn {
		t.Error("expected relay on")
	}

	if !status.HasTimer {
		t.Error("expected timer active")
	}

	if status.TimerRemaining != 30 {
		t.Errorf("expected 30s remaining, got %d", status.TimerRemaining)
	}

	if status.Source != "input" {
		t.Errorf("expected source input, got %s", status.Source)
	}
}

// TestRelayGetStatusError tests status error handling.
func TestRelayGetStatusError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/relay/0", errors.New("timeout"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	_, err := relay.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelayGetStatusInvalidJSON tests invalid JSON handling.
func TestRelayGetStatusInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/relay/0"] = json.RawMessage(`{invalid}`)

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	_, err := relay.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestRelayTurnOn tests turning relay on.
func TestRelayTurnOn(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/relay/0?turn=on", map[string]bool{"ison": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.TurnOn(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mt.GetCalls()
	if len(calls) != 1 || calls[0] != "/relay/0?turn=on" {
		t.Errorf("expected /relay/0?turn=on call, got %v", calls)
	}
}

// TestRelayTurnOff tests turning relay off.
func TestRelayTurnOff(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/relay/0?turn=off", map[string]bool{"ison": false})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.TurnOff(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mt.GetCalls()
	if len(calls) != 1 || calls[0] != "/relay/0?turn=off" {
		t.Errorf("expected /relay/0?turn=off call, got %v", calls)
	}
}

// TestRelayToggle tests toggling relay.
func TestRelayToggle(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/relay/0?turn=toggle", map[string]bool{"ison": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.Toggle(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelaySet tests Set method.
func TestRelaySet(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/relay/0?turn=on", map[string]bool{"ison": true})
	mt.SetResponse("/relay/0?turn=off", map[string]bool{"ison": false})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	// Test setting on
	err := relay.Set(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test setting off
	err = relay.Set(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelayTurnOnForDuration tests timer-based on.
func TestRelayTurnOnForDuration(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/relay/0?turn=on&timer=60", map[string]bool{"ison": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.TurnOnForDuration(ctx, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mt.GetCalls()
	if len(calls) != 1 || calls[0] != "/relay/0?turn=on&timer=60" {
		t.Errorf("expected timer call, got %v", calls)
	}
}

// TestRelayTurnOffForDuration tests timer-based off.
func TestRelayTurnOffForDuration(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/relay/0?turn=off&timer=30", map[string]bool{"ison": false})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.TurnOffForDuration(ctx, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelayGetConfig tests config retrieval.
func TestRelayGetConfig(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0", &RelayConfig{
		Name:         "Main Light",
		DefaultState: "last",
		BtnType:      "momentary",
		AutoOff:      300,
		MaxPower:     1000,
	})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	config, err := relay.GetConfig(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Name != "Main Light" {
		t.Errorf("expected name Main Light, got %s", config.Name)
	}

	if config.DefaultState != "last" {
		t.Errorf("expected default state last, got %s", config.DefaultState)
	}

	if config.MaxPower != 1000 {
		t.Errorf("expected max power 1000, got %d", config.MaxPower)
	}
}

// TestRelaySetConfig tests config update.
func TestRelaySetConfig(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?name=Test&default_state=off&btn_type=toggle&max_power=500", map[string]bool{"ok": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	config := &RelayConfig{
		Name:         "Test",
		DefaultState: "off",
		BtnType:      "toggle",
		MaxPower:     500,
	}

	err := relay.SetConfig(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelaySetConfigEmpty tests config with no changes.
func TestRelaySetConfigEmpty(t *testing.T) {
	mt := newMockTransport()
	relay := NewRelay(mt, 0)
	ctx := context.Background()

	// Empty config should do nothing
	err := relay.SetConfig(ctx, &RelayConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No calls should be made
	if len(mt.GetCalls()) != 0 {
		t.Error("expected no calls for empty config")
	}
}

// TestRelaySetName tests name setting.
func TestRelaySetName(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?name=NewName", map[string]bool{"ok": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetName(ctx, "NewName")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelaySetDefaultState tests default state setting.
func TestRelaySetDefaultState(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?default_state=on", map[string]bool{"ok": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetDefaultState(ctx, "on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelaySetButtonType tests button type setting.
func TestRelaySetButtonType(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?btn_type=toggle", map[string]bool{"ok": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetButtonType(ctx, "toggle")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelaySetAutoOn tests auto-on timer setting.
func TestRelaySetAutoOn(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?auto_on=60", map[string]bool{"ok": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetAutoOn(ctx, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelaySetAutoOff tests auto-off timer setting.
func TestRelaySetAutoOff(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?auto_off=300", map[string]bool{"ok": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetAutoOff(ctx, 300)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelaySetMaxPower tests max power setting.
func TestRelaySetMaxPower(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?max_power=2000", map[string]bool{"ok": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetMaxPower(ctx, 2000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelaySetSchedule tests schedule enable/disable.
func TestRelaySetSchedule(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?schedule=true", map[string]bool{"ok": true})
	mt.SetResponse("/settings/relay/0?schedule=false", map[string]bool{"ok": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	// Enable schedule
	err := relay.SetSchedule(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Disable schedule
	err = relay.SetSchedule(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelayAddScheduleRule tests adding schedule rules.
func TestRelayAddScheduleRule(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0", &RelayConfig{
		ScheduleRules: []string{"0800-0123456-on"},
	})
	mt.SetResponse("/settings/relay/0?schedule_rules=[\"0800-0123456-on\",\"2000-0123456-off\"]", map[string]bool{"ok": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.AddScheduleRule(ctx, "2000-0123456-off")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelayClearScheduleRules tests clearing schedule rules.
func TestRelayClearScheduleRules(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?schedule_rules=[]", map[string]bool{"ok": true})

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.ClearScheduleRules(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRelayMultipleIDs tests relays with different IDs.
func TestRelayMultipleIDs(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/relay/0?turn=on", map[string]bool{"ison": true})
	mt.SetResponse("/relay/1?turn=on", map[string]bool{"ison": true})
	mt.SetResponse("/relay/2?turn=on", map[string]bool{"ison": true})

	ctx := context.Background()

	for id := 0; id < 3; id++ {
		relay := NewRelay(mt, id)
		if relay.ID() != id {
			t.Errorf("expected ID %d, got %d", id, relay.ID())
		}

		err := relay.TurnOn(ctx)
		if err != nil {
			t.Fatalf("unexpected error for relay %d: %v", id, err)
		}
	}

	calls := mt.GetCalls()
	if len(calls) != 3 {
		t.Errorf("expected 3 calls, got %d", len(calls))
	}
}

// TestRelayErrorPropagation tests error propagation.
func TestRelayErrorPropagation(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/relay/0?turn=on", errors.New("device busy"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.TurnOn(ctx)
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "failed to turn relay on: device busy" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestRelayTurnOffError tests TurnOff error handling.
func TestRelayTurnOffError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/relay/0?turn=off", errors.New("device busy"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.TurnOff(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelayToggleError tests Toggle error handling.
func TestRelayToggleError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/relay/0?turn=toggle", errors.New("device busy"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.Toggle(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelayGetStatusOffline tests GetStatus offline error handling.
func TestRelayGetStatusOffline(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/relay/0", errors.New("offline"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	_, err := relay.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelayGetConfigUnauthorized tests GetConfig unauthorized error handling.
func TestRelayGetConfigUnauthorized(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0", errors.New("unauthorized"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	_, err := relay.GetConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelayTurnOnForDurationError tests TurnOnForDuration error handling.
func TestRelayTurnOnForDurationError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/relay/0?turn=on&timer=60", errors.New("timer failed"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.TurnOnForDuration(ctx, 60)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelayTurnOffForDurationError tests TurnOffForDuration error handling.
func TestRelayTurnOffForDurationError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/relay/0?turn=off&timer=60", errors.New("timer failed"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.TurnOffForDuration(ctx, 60)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelayAddScheduleRuleError tests AddScheduleRule error handling.
func TestRelayAddScheduleRuleError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0", errors.New("config error"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.AddScheduleRule(ctx, "0800-0123456-on")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelayClearScheduleRulesError tests ClearScheduleRules error handling.
func TestRelayClearScheduleRulesError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?schedule_rules=[]", errors.New("clear failed"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.ClearScheduleRules(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelayGetConfigInvalidJSON tests GetConfig with invalid JSON response.
func TestRelayGetConfigInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.responses["/settings/relay/0"] = json.RawMessage(`{invalid`)

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	_, err := relay.GetConfig(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestRelaySetNameError tests SetName error handling.
func TestRelaySetNameError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?name=Test", errors.New("set failed"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetName(ctx, "Test")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelaySetDefaultStateError tests SetDefaultState error handling.
func TestRelaySetDefaultStateError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?default_state=on", errors.New("set failed"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetDefaultState(ctx, "on")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelaySetButtonTypeError tests SetButtonType error handling.
func TestRelaySetButtonTypeError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?btn_type=toggle", errors.New("set failed"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetButtonType(ctx, "toggle")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelaySetAutoOnError tests SetAutoOn error handling.
func TestRelaySetAutoOnError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?auto_on=60", errors.New("set failed"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetAutoOn(ctx, 60)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelaySetAutoOffError tests SetAutoOff error handling.
func TestRelaySetAutoOffError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?auto_off=120", errors.New("set failed"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetAutoOff(ctx, 120)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelaySetMaxPowerError tests SetMaxPower error handling.
func TestRelaySetMaxPowerError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?max_power=1000", errors.New("set failed"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetMaxPower(ctx, 1000)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelaySetScheduleError tests SetSchedule error handling.
func TestRelaySetScheduleError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?schedule=true", errors.New("set failed"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	err := relay.SetSchedule(ctx, true)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRelaySetConfigError tests SetConfig error handling.
func TestRelaySetConfigError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?name=Test", errors.New("set failed"))

	relay := NewRelay(mt, 0)
	ctx := context.Background()

	config := &RelayConfig{
		Name: "Test",
	}
	err := relay.SetConfig(ctx, config)
	if err == nil {
		t.Fatal("expected error")
	}
}
