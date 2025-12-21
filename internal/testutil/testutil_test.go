package testutil

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/transport"
	"github.com/tj-smith47/shelly-go/types"
)

func TestNewMockTransport(t *testing.T) {
	mt := NewMockTransport()
	if mt == nil {
		t.Fatal("NewMockTransport() returned nil")
	}
	if mt.CallCount() != 0 {
		t.Errorf("CallCount() = %v, want 0", mt.CallCount())
	}
}

func TestMockTransport_OnCall(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCall("Test.Method", func(params any) (json.RawMessage, error) {
		return json.RawMessage(`{"result":"ok"}`), nil
	})

	result, err := mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	if err != nil {
		t.Errorf("Call() error = %v", err)
	}
	if string(result) != `{"result":"ok"}` {
		t.Errorf("Call() result = %v, want %v", string(result), `{"result":"ok"}`)
	}
}

func TestMockTransport_OnCallReturn(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCallReturn("Test.Method", map[string]string{"status": "ok"}, nil)

	result, err := mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	if err != nil {
		t.Errorf("Call() error = %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result, &data); err != nil {
		t.Errorf("Unmarshal error = %v", err)
	}
	if data["status"] != "ok" {
		t.Errorf("status = %v, want ok", data["status"])
	}
}

func TestMockTransport_OnCallReturn_NilResponse(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCallReturn("Test.Method", nil, nil)

	result, err := mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	if err != nil {
		t.Errorf("Call() error = %v", err)
	}
	if string(result) != `{}` {
		t.Errorf("Call() result = %v, want {}", string(result))
	}
}

func TestMockTransport_OnCallReturn_Error(t *testing.T) {
	mt := NewMockTransport()
	expectedErr := errors.New("test error")
	mt.OnCallReturn("Test.Method", nil, expectedErr)

	_, err := mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	if err != expectedErr {
		t.Errorf("Call() error = %v, want %v", err, expectedErr)
	}
}

func TestMockTransport_OnCallJSON(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCallJSON("Test.Method", `{"json":"value"}`)

	result, err := mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	if err != nil {
		t.Errorf("Call() error = %v", err)
	}
	if string(result) != `{"json":"value"}` {
		t.Errorf("Call() result = %v, want %v", string(result), `{"json":"value"}`)
	}
}

func TestMockTransport_OnCallError(t *testing.T) {
	mt := NewMockTransport()
	expectedErr := errors.New("method error")
	mt.OnCallError("Test.Method", expectedErr)

	_, err := mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	if err != expectedErr {
		t.Errorf("Call() error = %v, want %v", err, expectedErr)
	}
}

func TestMockTransport_Call_NoHandler(t *testing.T) {
	mt := NewMockTransport()

	_, err := mt.Call(context.Background(), transport.NewSimpleRequest("Unknown.Method"))
	if err == nil {
		t.Error("Call() should return error for unknown method")
	}
}

func TestMockTransport_Call_Closed(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCallJSON("Test.Method", `{}`)
	mt.Close()

	_, err := mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	if err == nil {
		t.Error("Call() should return error when closed")
	}
}

func TestMockTransport_Call_ContextCanceled(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCall("Test.Method", func(params any) (json.RawMessage, error) {
		return json.RawMessage(`{}`), nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mt.Call(ctx, transport.NewSimpleRequest("Test.Method"))
	if err != context.Canceled {
		t.Errorf("Call() error = %v, want %v", err, context.Canceled)
	}
}

func TestMockTransport_Calls(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCallJSON("Test.Method1", `{}`)
	mt.OnCallJSON("Test.Method2", `{}`)

	_, _ = mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method1"))
	_, _ = mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method2"))

	calls := mt.Calls()
	if len(calls) != 2 {
		t.Errorf("len(Calls()) = %v, want 2", len(calls))
	}
	if calls[0].Method != "Test.Method1" {
		t.Errorf("calls[0].Method = %v, want Test.Method1", calls[0].Method)
	}
	// Note: SimpleRequest has no params, so Params should be empty/nil json.RawMessage
	if len(calls[0].Params.(json.RawMessage)) != 0 {
		t.Errorf("calls[0].Params = %v, want empty", calls[0].Params)
	}
}

func TestMockTransport_LastCall(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCallJSON("Test.Method", `{}`)

	if mt.LastCall() != nil {
		t.Error("LastCall() should be nil with no calls")
	}

	_, _ = mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	last := mt.LastCall()
	if last == nil {
		t.Fatal("LastCall() returned nil")
	}
	if last.Method != "Test.Method" {
		t.Errorf("LastCall().Method = %v, want Test.Method", last.Method)
	}
}

func TestMockTransport_Reset(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCallJSON("Test.Method", `{}`)
	_, _ = mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	mt.Close()

	mt.Reset()

	if mt.CallCount() != 0 {
		t.Errorf("CallCount() after reset = %v, want 0", mt.CallCount())
	}

	// Should be able to call again after reset
	_, err := mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	if err != nil {
		t.Errorf("Call() after reset error = %v", err)
	}
}

func TestMockTransport_ClearHandlers(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCallJSON("Test.Method", `{}`)

	mt.ClearHandlers()

	_, err := mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	if err == nil {
		t.Error("Call() should fail after ClearHandlers()")
	}
}

func TestMockTransport_WasCalled(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCallJSON("Test.Method", `{}`)

	if mt.WasCalled("Test.Method") {
		t.Error("WasCalled() should be false before call")
	}

	_, _ = mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))

	if !mt.WasCalled("Test.Method") {
		t.Error("WasCalled() should be true after call")
	}
	if mt.WasCalled("Other.Method") {
		t.Error("WasCalled() should be false for uncalled method")
	}
}

func TestMockTransport_CallsFor(t *testing.T) {
	mt := NewMockTransport()
	mt.OnCallJSON("Test.Method", `{}`)
	mt.OnCallJSON("Other.Method", `{}`)

	_, _ = mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))
	_, _ = mt.Call(context.Background(), transport.NewSimpleRequest("Other.Method"))
	_, _ = mt.Call(context.Background(), transport.NewSimpleRequest("Test.Method"))

	calls := mt.CallsFor("Test.Method")
	if len(calls) != 2 {
		t.Errorf("len(CallsFor) = %v, want 2", len(calls))
	}
}

func TestNewMockGen1Device(t *testing.T) {
	device := NewMockGen1Device("192.168.1.100")

	if device.Address() != "192.168.1.100" {
		t.Errorf("Address() = %v, want 192.168.1.100", device.Address())
	}
	if device.Generation() != types.Gen1 {
		t.Errorf("Generation() = %v, want Gen1", device.Generation())
	}
}

func TestMockGen1Device_Info(t *testing.T) {
	device := NewMockGen1Device("192.168.1.100")

	info, err := device.Info(context.Background())
	if err != nil {
		t.Errorf("Info() error = %v", err)
	}
	if info.ID != "shelly1-aabbcc" {
		t.Errorf("info.ID = %v, want shelly1-aabbcc", info.ID)
	}
}

func TestMockGen1Device_Info_Error(t *testing.T) {
	device := NewMockGen1Device("192.168.1.100")
	expectedErr := errors.New("info error")
	device.SetError("Info", expectedErr)

	_, err := device.Info(context.Background())
	if err != expectedErr {
		t.Errorf("Info() error = %v, want %v", err, expectedErr)
	}
}

func TestMockGen1Device_SetDeviceInfo(t *testing.T) {
	device := NewMockGen1Device("192.168.1.100")
	newInfo := &types.DeviceInfo{ID: "custom-id"}
	device.SetDeviceInfo(newInfo)

	info, err := device.Info(context.Background())
	if err != nil {
		t.Errorf("Info() error = %v", err)
	}
	if info.ID != "custom-id" {
		t.Errorf("info.ID = %v, want custom-id", info.ID)
	}
}

func TestMockGen1Device_RelayState(t *testing.T) {
	device := NewMockGen1Device("192.168.1.100")

	if device.GetRelayState(0) {
		t.Error("GetRelayState(0) should be false initially")
	}

	device.SetRelayState(0, true)
	if !device.GetRelayState(0) {
		t.Error("GetRelayState(0) should be true after SetRelayState")
	}
}

func TestMockGen1Device_RollerPosition(t *testing.T) {
	device := NewMockGen1Device("192.168.1.100")

	device.SetRollerPosition(0, 50)
	if device.GetRollerPosition(0) != 50 {
		t.Errorf("GetRollerPosition(0) = %v, want 50", device.GetRollerPosition(0))
	}
}

func TestMockGen1Device_MeterPower(t *testing.T) {
	device := NewMockGen1Device("192.168.1.100")

	device.SetMeterPower(0, 123.45)
	if device.GetMeterPower(0) != 123.45 {
		t.Errorf("GetMeterPower(0) = %v, want 123.45", device.GetMeterPower(0))
	}
}

func TestNewMockGen2Device(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")

	if device.Address() != "192.168.1.101" {
		t.Errorf("Address() = %v, want 192.168.1.101", device.Address())
	}
	if device.Generation() != types.Gen2 {
		t.Errorf("Generation() = %v, want Gen2", device.Generation())
	}
	if device.Transport() == nil {
		t.Error("Transport() should not be nil")
	}
}

func TestMockGen2Device_Info(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")

	info, err := device.Info(context.Background())
	if err != nil {
		t.Errorf("Info() error = %v", err)
	}
	if info.ID != "shellyplus1-aabbcc" {
		t.Errorf("info.ID = %v, want shellyplus1-aabbcc", info.ID)
	}
}

func TestMockGen2Device_SwitchStatus(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")

	status := &MockSwitchStatus{Output: true, APower: 50.5}
	device.SetSwitchStatus(0, status)

	retrieved := device.GetSwitchStatus(0)
	if retrieved == nil {
		t.Fatal("GetSwitchStatus() returned nil")
	}
	if !retrieved.Output {
		t.Error("retrieved.Output should be true")
	}
	if retrieved.APower != 50.5 {
		t.Errorf("retrieved.APower = %v, want 50.5", retrieved.APower)
	}
}

func TestMockGen2Device_CoverStatus(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")

	status := &MockCoverStatus{State: "open", CurrentPos: 100}
	device.SetCoverStatus(0, status)

	retrieved := device.GetCoverStatus(0)
	if retrieved == nil {
		t.Fatal("GetCoverStatus() returned nil")
	}
	if retrieved.State != "open" {
		t.Errorf("retrieved.State = %v, want open", retrieved.State)
	}
}

func TestMockGen2Device_LightStatus(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")

	status := &MockLightStatus{Output: true, Brightness: 75}
	device.SetLightStatus(0, status)

	retrieved := device.GetLightStatus(0)
	if retrieved == nil {
		t.Fatal("GetLightStatus() returned nil")
	}
	if retrieved.Brightness != 75 {
		t.Errorf("retrieved.Brightness = %v, want 75", retrieved.Brightness)
	}
}

func TestMockGen2Device_InputStatus(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")

	status := &MockInputStatus{State: true, Type: "button"}
	device.SetInputStatus(0, status)

	retrieved := device.GetInputStatus(0)
	if retrieved == nil {
		t.Fatal("GetInputStatus() returned nil")
	}
	if retrieved.Type != "button" {
		t.Errorf("retrieved.Type = %v, want button", retrieved.Type)
	}
}

func TestMockGen2Device_Script(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")

	script := &MockScript{ID: 1, Name: "test", Enable: true}
	device.AddScript(script)

	retrieved := device.GetScript(1)
	if retrieved == nil {
		t.Fatal("GetScript() returned nil")
	}
	if retrieved.Name != "test" {
		t.Errorf("retrieved.Name = %v, want test", retrieved.Name)
	}
}

func TestMockGen2Device_Schedule(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")

	schedule := &MockSchedule{ID: 1, Enable: true, Timespec: "0 0 * * *"}
	device.AddSchedule(schedule)

	retrieved := device.GetSchedule(1)
	if retrieved == nil {
		t.Fatal("GetSchedule() returned nil")
	}
	if retrieved.Timespec != "0 0 * * *" {
		t.Errorf("retrieved.Timespec = %v, want 0 0 * * *", retrieved.Timespec)
	}
}

func TestMockGen2Device_Webhook(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")

	webhook := &MockWebhook{ID: 1, Enable: true, Event: "switch.on"}
	device.AddWebhook(webhook)

	retrieved := device.GetWebhook(1)
	if retrieved == nil {
		t.Fatal("GetWebhook() returned nil")
	}
	if retrieved.Event != "switch.on" {
		t.Errorf("retrieved.Event = %v, want switch.on", retrieved.Event)
	}
}

func TestMockGen2Device_KVS(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")

	device.SetKVS("key1", "value1")
	retrieved := device.GetKVS("key1")
	if retrieved != "value1" {
		t.Errorf("GetKVS() = %v, want value1", retrieved)
	}
}

func TestMockGen2Device_Config(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")

	device.SetConfig("sys.device.name", "Test Device")
	retrieved := device.GetConfig("sys.device.name")
	if retrieved != "Test Device" {
		t.Errorf("GetConfig() = %v, want Test Device", retrieved)
	}
}

func TestMockGen2Device_SetError(t *testing.T) {
	device := NewMockGen2Device("192.168.1.101")
	expectedErr := errors.New("info error")

	// Set error
	device.SetError("Info", expectedErr)
	_, err := device.Info(context.Background())
	if err != expectedErr {
		t.Errorf("Info() error = %v, want %v", err, expectedErr)
	}

	// Clear error
	device.SetError("Info", nil)
	_, err = device.Info(context.Background())
	if err != nil {
		t.Errorf("Info() after clearing error = %v, want nil", err)
	}
}

func TestToJSON(t *testing.T) {
	data := map[string]string{"key": "value"}
	result := ToJSON(data)

	if string(result) != `{"key":"value"}` {
		t.Errorf("ToJSON() = %v, want %v", string(result), `{"key":"value"}`)
	}
}

func TestToJSON_InvalidValue(t *testing.T) {
	// Channels can't be marshaled to JSON
	ch := make(chan int)
	result := ToJSON(ch)

	if string(result) != `{}` {
		t.Errorf("ToJSON() with invalid value = %v, want {}", string(result))
	}
}

func TestLoadFixture(t *testing.T) {
	data, err := LoadFixture("gen2/switch_status.json")
	if err != nil {
		t.Fatalf("LoadFixture() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("LoadFixture() returned empty data")
	}

	var status map[string]any
	if err := json.Unmarshal(data, &status); err != nil {
		t.Errorf("Unmarshal fixture error = %v", err)
	}
	if status["output"] != true {
		t.Errorf("status.output = %v, want true", status["output"])
	}
}

func TestLoadFixture_NotFound(t *testing.T) {
	_, err := LoadFixture("nonexistent.json")
	if err == nil {
		t.Error("LoadFixture() should return error for missing file")
	}
}

func TestMustLoadFixture(t *testing.T) {
	data := MustLoadFixture("gen2/switch_status.json")
	if len(data) == 0 {
		t.Error("MustLoadFixture() returned empty data")
	}
}

func TestMustLoadFixture_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoadFixture() should panic on missing file")
		}
	}()
	MustLoadFixture("nonexistent.json")
}

func TestLoadFixtureJSON(t *testing.T) {
	var status map[string]any
	err := LoadFixtureJSON("gen2/switch_status.json", &status)
	if err != nil {
		t.Fatalf("LoadFixtureJSON() error = %v", err)
	}
	if status["output"] != true {
		t.Errorf("status.output = %v, want true", status["output"])
	}
}

func TestAssertEqual(t *testing.T) {
	// This test verifies the helper works without panicking
	mockT := &testing.T{}
	AssertEqual(mockT, 1, 1)
	AssertEqual(mockT, "hello", "hello")
}

func TestAssertNotEqual(t *testing.T) {
	mockT := &testing.T{}
	AssertNotEqual(mockT, 1, 2)
}

func TestAssertNil(t *testing.T) {
	mockT := &testing.T{}
	var nilPtr *int
	AssertNil(mockT, nilPtr)
}

func TestAssertNotNil(t *testing.T) {
	mockT := &testing.T{}
	val := 42
	AssertNotNil(mockT, &val)
}

func TestAssertNoError(t *testing.T) {
	mockT := &testing.T{}
	AssertNoError(mockT, nil)
}

func TestAssertError(t *testing.T) {
	mockT := &testing.T{}
	AssertError(mockT, errors.New("test error"))
}

func TestAssertErrorContains(t *testing.T) {
	mockT := &testing.T{}
	AssertErrorContains(mockT, errors.New("test error message"), "error")
}

func TestAssertTrue(t *testing.T) {
	mockT := &testing.T{}
	AssertTrue(mockT, true)
}

func TestAssertFalse(t *testing.T) {
	mockT := &testing.T{}
	AssertFalse(mockT, false)
}

func TestAssertLen(t *testing.T) {
	mockT := &testing.T{}
	AssertLen(mockT, []int{1, 2, 3}, 3)
	AssertLen(mockT, "hello", 5)
}

func TestAssertContains(t *testing.T) {
	mockT := &testing.T{}
	AssertContains(mockT, []int{1, 2, 3}, 2)
}

func TestAssertStringContains(t *testing.T) {
	mockT := &testing.T{}
	AssertStringContains(mockT, "hello world", "world")
}

func TestMustJSON(t *testing.T) {
	data := MustJSON(map[string]int{"x": 1})
	if string(data) != `{"x":1}` {
		t.Errorf("MustJSON() = %v, want %v", string(data), `{"x":1}`)
	}
}

func TestMustJSONRaw(t *testing.T) {
	raw := MustJSONRaw(map[string]int{"x": 1})
	if string(raw) != `{"x":1}` {
		t.Errorf("MustJSONRaw() = %v, want %v", string(raw), `{"x":1}`)
	}
}

func TestJSONEqual(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"equal objects", `{"a":1,"b":2}`, `{"b":2,"a":1}`, true},
		{"different values", `{"a":1}`, `{"a":2}`, false},
		{"different keys", `{"a":1}`, `{"b":1}`, false},
		{"invalid json a", `{invalid`, `{}`, false},
		{"invalid json b", `{}`, `{invalid`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JSONEqual([]byte(tt.a), []byte(tt.b)); got != tt.want {
				t.Errorf("JSONEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAssertJSONEqual(t *testing.T) {
	mockT := &testing.T{}
	AssertJSONEqual(mockT, []byte(`{"a":1}`), []byte(`{"a":1}`))
}

func TestCreateTestContext(t *testing.T) {
	ctx := CreateTestContext()
	if ctx == nil {
		t.Error("CreateTestContext() returned nil")
	}
}

func TestMockTransport_OnPathMatch(t *testing.T) {
	mt := NewMockTransport()
	defer mt.ClearMatchers()

	mt.OnPathMatch(func(path string) bool {
		return path == "/rpc/Switch.GetStatus"
	}, func(params any) (json.RawMessage, error) {
		return json.RawMessage(`{"output":true}`), nil
	})

	ctx := context.Background()
	resp, err := mt.Call(ctx, transport.NewSimpleRequest("/rpc/Switch.GetStatus"))
	if err != nil {
		t.Errorf("Call() error = %v", err)
	}
	if string(resp) != `{"output":true}` {
		t.Errorf("Call() response = %v, want %v", string(resp), `{"output":true}`)
	}

	// Non-matching path should error
	_, err = mt.Call(ctx, transport.NewSimpleRequest("/rpc/Other.Method"))
	if err == nil {
		t.Error("Call() for non-matching path should error")
	}
}

func TestMockTransport_OnPathContains(t *testing.T) {
	mt := NewMockTransport()
	defer mt.ClearMatchers()

	mt.OnPathContains("Switch", map[string]bool{"output": true}, nil)

	ctx := context.Background()
	resp, err := mt.Call(ctx, transport.NewSimpleRequest("/rpc/Switch.GetStatus"))
	if err != nil {
		t.Errorf("Call() error = %v", err)
	}
	if string(resp) != `{"output":true}` {
		t.Errorf("Call() response = %v", string(resp))
	}

	// Also matches other Switch methods
	_, err = mt.Call(ctx, transport.NewSimpleRequest("Switch.Set"))
	if err != nil {
		t.Errorf("Call() error = %v", err)
	}
}

func TestMockTransport_OnPathContains_Error(t *testing.T) {
	mt := NewMockTransport()
	defer mt.ClearMatchers()

	expectedErr := errors.New("test error")
	mt.OnPathContains("Test", nil, expectedErr)

	ctx := context.Background()
	_, err := mt.Call(ctx, transport.NewSimpleRequest("Test.Method"))
	if err == nil {
		t.Error("Call() should return error")
	}
}

func TestMockTransport_OnPathPrefix(t *testing.T) {
	mt := NewMockTransport()
	defer mt.ClearMatchers()

	mt.OnPathPrefix("/rpc/", map[string]string{"status": "ok"}, nil)

	ctx := context.Background()
	resp, err := mt.Call(ctx, transport.NewSimpleRequest("/rpc/Any.Method"))
	if err != nil {
		t.Errorf("Call() error = %v", err)
	}
	if string(resp) != `{"status":"ok"}` {
		t.Errorf("Call() response = %v", string(resp))
	}

	// Non-prefixed path should error
	_, err = mt.Call(ctx, transport.NewSimpleRequest("Other.Method"))
	if err == nil {
		t.Error("Call() for non-prefixed path should error")
	}
}

func TestMockTransport_ClearMatchers(t *testing.T) {
	mt := NewMockTransport()
	mt.OnPathContains("Test", nil, nil)

	mt.ClearMatchers()

	ctx := context.Background()
	_, err := mt.Call(ctx, transport.NewSimpleRequest("Test.Method"))
	if err == nil {
		t.Error("Call() after ClearMatchers() should error")
	}
}

func TestMakeResponseHandler(t *testing.T) {
	tests := []struct {
		name     string
		response any
		err      error
		wantErr  bool
		wantResp string
	}{
		{
			name:    "with error",
			err:     errors.New("test error"),
			wantErr: true,
		},
		{
			name:     "with nil response",
			response: nil,
			wantResp: `{}`,
		},
		{
			name:     "with json.RawMessage",
			response: json.RawMessage(`{"raw":"message"}`),
			wantResp: `{"raw":"message"}`,
		},
		{
			name:     "with string",
			response: `{"string":"value"}`,
			wantResp: `{"string":"value"}`,
		},
		{
			name:     "with struct",
			response: struct{ Name string }{Name: "test"},
			wantResp: `{"Name":"test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := makeResponseHandler(tt.response, tt.err)
			resp, err := handler(nil)

			if tt.wantErr {
				if err == nil {
					t.Error("handler error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("handler error = %v", err)
			}
			if string(resp) != tt.wantResp {
				t.Errorf("handler response = %v, want %v", string(resp), tt.wantResp)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "lo wo", true},
		{"hello", "hello", true},
		{"hello", "", true},
		{"hello", "x", false},
		{"hello", "hello world", false},
		{"", "a", false},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		s      string
		prefix string
		want   bool
	}{
		{"hello world", "hello", true},
		{"hello", "hello", true},
		{"hello", "", true},
		{"hello", "hi", false},
		{"hello", "hello world", false},
		{"", "", true},
		{"", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.prefix, func(t *testing.T) {
			got := hasPrefix(tt.s, tt.prefix)
			if got != tt.want {
				t.Errorf("hasPrefix(%q, %q) = %v, want %v", tt.s, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestFindSubstr(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   int
	}{
		{"hello world", "world", 6},
		{"hello world", "hello", 0},
		{"hello world", "o", 4},
		{"hello", "x", -1},
		{"hello", "hello", 0},
		{"", "a", -1},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := findSubstr(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("findSubstr(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}
