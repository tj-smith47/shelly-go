package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

// mockTransport implements transport.Transport for testing
type mockTransport struct {
	callFunc  func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error)
	closeFunc func() error
}

func (m *mockTransport) Call(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
	if m.callFunc != nil {
		return m.callFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTransport) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// Helper function to create a proper JSON-RPC response
func jsonrpcResponse(result string) (json.RawMessage, error) {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  json.RawMessage(result),
	}
	return json.Marshal(response)
}

// Helper function to create pointer to any type
func ptr[T any](v T) *T {
	return &v
}

func TestNewSwitch(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	sw := NewSwitch(client, 0)

	if sw == nil {
		t.Fatal("NewSwitch returned nil")
	}

	if sw.Type() != "switch" {
		t.Errorf("Type() = %q, want %q", sw.Type(), "switch")
	}

	if sw.ID() != 0 {
		t.Errorf("ID() = %d, want %d", sw.ID(), 0)
	}

	if sw.Key() != "switch:0" {
		t.Errorf("Key() = %q, want %q", sw.Key(), "switch:0")
	}
}

func TestSwitch_Set(t *testing.T) {
	tests := []struct {
		name      string
		params    *SwitchSetParams
		result    string
		wantWasOn bool
	}{
		{
			name: "turn on",
			params: &SwitchSetParams{
				ID: 0,
				On: ptr(true),
			},
			result:    `{"was_on": false}`,
			wantWasOn: false,
		},
		{
			name: "turn off",
			params: &SwitchSetParams{
				ID: 0,
				On: ptr(false),
			},
			result:    `{"was_on": true}`,
			wantWasOn: true,
		},
		{
			name: "with toggle_after",
			params: &SwitchSetParams{
				ID:          0,
				On:          ptr(true),
				ToggleAfter: ptr(10.0),
			},
			result:    `{"was_on": false}`,
			wantWasOn: false,
		},
		{
			name:      "nil params uses component ID",
			params:    nil,
			result:    `{"was_on": true}`,
			wantWasOn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Switch.Set" {
						t.Errorf("method = %q, want %q", method, "Switch.Set")
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			sw := NewSwitch(client, 0)

			result, err := sw.Set(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			if result.WasOn != tt.wantWasOn {
				t.Errorf("WasOn = %v, want %v", result.WasOn, tt.wantWasOn)
			}
		})
	}
}

func TestSwitch_Set_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	sw := NewSwitch(client, 0)

	testComponentError(t, "Set", func() error {
		_, err := sw.Set(context.Background(), &SwitchSetParams{On: ptr(true)})
		return err
	})
}

func TestSwitch_Set_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	sw := NewSwitch(client, 0)

	testComponentInvalidJSON(t, "Set", func() error {
		_, err := sw.Set(context.Background(), &SwitchSetParams{On: ptr(true)})
		return err
	})
}

func TestSwitch_Toggle(t *testing.T) {
	tests := []struct {
		name      string
		result    string
		wantWasOn bool
	}{
		{
			name:      "toggle from on to off",
			result:    `{"was_on": true}`,
			wantWasOn: true,
		},
		{
			name:      "toggle from off to on",
			result:    `{"was_on": false}`,
			wantWasOn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Switch.Toggle" {
						t.Errorf("method = %q, want %q", method, "Switch.Toggle")
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			sw := NewSwitch(client, 0)

			result, err := sw.Toggle(context.Background())
			if err != nil {
				t.Fatalf("Toggle() error = %v", err)
			}

			if result.WasOn != tt.wantWasOn {
				t.Errorf("WasOn = %v, want %v", result.WasOn, tt.wantWasOn)
			}
		})
	}
}

func TestSwitch_Toggle_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	sw := NewSwitch(client, 0)

	testComponentError(t, "Toggle", func() error {
		_, err := sw.Toggle(context.Background())
		return err
	})
}

func TestSwitch_Toggle_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	sw := NewSwitch(client, 0)

	testComponentInvalidJSON(t, "Toggle", func() error {
		_, err := sw.Toggle(context.Background())
		return err
	})
}

func TestSwitch_ResetCounters(t *testing.T) {
	tests := []struct {
		name         string
		counterTypes []string
	}{
		{
			name:         "reset specific counter",
			counterTypes: []string{"aenergy"},
		},
		{
			name:         "reset all counters",
			counterTypes: []string{},
		},
		{
			name:         "reset multiple counters",
			counterTypes: []string{"aenergy", "ret_aenergy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Switch.ResetCounters" {
						t.Errorf("method = %q, want %q", method, "Switch.ResetCounters")
					}
					return jsonrpcResponse(`{}`)
				},
			}
			client := rpc.NewClient(tr)
			sw := NewSwitch(client, 0)

			err := sw.ResetCounters(context.Background(), tt.counterTypes)
			if err != nil {
				t.Errorf("ResetCounters() error = %v", err)
			}
		})
	}
}

func TestSwitch_ResetCounters_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	sw := NewSwitch(client, 0)

	testComponentError(t, "ResetCounters", func() error {
		return sw.ResetCounters(context.Background(), []string{"aenergy"})
	})
}

func TestSwitch_GetConfig(t *testing.T) {
	result := `{
		"id": 0,
		"name": "Living Room",
		"initial_state": "off",
		"auto_on": true,
		"auto_on_delay": 60.0,
		"auto_off": false,
		"power_limit": 2300.0
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Switch.GetConfig" {
				t.Errorf("method = %q, want %q", method, "Switch.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	sw := NewSwitch(client, 0)

	config, err := sw.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if config.ID != 0 {
		t.Errorf("ID = %d, want 0", config.ID)
	}

	if config.Name == nil || *config.Name != "Living Room" {
		t.Errorf("Name = %v, want %q", config.Name, "Living Room")
	}

	if config.InitialState == nil || *config.InitialState != "off" {
		t.Errorf("InitialState = %v, want %q", config.InitialState, "off")
	}

	if config.AutoOn == nil || !*config.AutoOn {
		t.Errorf("AutoOn = %v, want true", config.AutoOn)
	}

	if config.AutoOnDelay == nil || *config.AutoOnDelay != 60.0 {
		t.Errorf("AutoOnDelay = %v, want 60.0", config.AutoOnDelay)
	}

	if config.PowerLimit == nil || *config.PowerLimit != 2300.0 {
		t.Errorf("PowerLimit = %v, want 2300.0", config.PowerLimit)
	}
}

func TestSwitch_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	sw := NewSwitch(client, 0)

	testComponentError(t, "GetConfig", func() error {
		_, err := sw.GetConfig(context.Background())
		return err
	})
}

func TestSwitch_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	sw := NewSwitch(client, 0)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := sw.GetConfig(context.Background())
		return err
	})
}

func TestSwitch_SetConfig(t *testing.T) {
	expectedConfig := &SwitchConfig{
		ID:           0,
		Name:         ptr("Kitchen"),
		InitialState: ptr("on"),
		AutoOff:      ptr(true),
		AutoOffDelay: ptr(300.0),
	}

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Switch.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Switch.SetConfig")
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	sw := NewSwitch(client, 0)

	err := sw.SetConfig(context.Background(), expectedConfig)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestSwitch_SetConfig_AutoSetID(t *testing.T) {
	// Config without ID should be auto-set
	config := &SwitchConfig{
		Name: ptr("Test"),
	}

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	sw := NewSwitch(client, 0)

	err := sw.SetConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestSwitch_GetStatus(t *testing.T) {
	result := `{
		"id": 0,
		"source": "http",
		"output": true,
		"apower": 123.5,
		"voltage": 230.2,
		"current": 0.54,
		"pf": 0.98,
		"freq": 50.0,
		"aenergy": {
			"total": 12345.67,
			"by_minute": [10.5, 11.2, 9.8],
			"minute_ts": 1234567890
		},
		"temperature": {
			"tC": 45.2,
			"tF": 113.4
		},
		"errors": []
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Switch.GetStatus" {
				t.Errorf("method = %q, want %q", method, "Switch.GetStatus")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	sw := NewSwitch(client, 0)

	status, err := sw.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status.ID != 0 {
		t.Errorf("ID = %d, want 0", status.ID)
	}

	if status.Source != "http" {
		t.Errorf("Source = %q, want %q", status.Source, "http")
	}

	if !status.Output {
		t.Error("Output = false, want true")
	}

	if status.APower == nil || *status.APower != 123.5 {
		t.Errorf("APower = %v, want 123.5", status.APower)
	}

	if status.Voltage == nil || *status.Voltage != 230.2 {
		t.Errorf("Voltage = %v, want 230.2", status.Voltage)
	}

	if status.Current == nil || *status.Current != 0.54 {
		t.Errorf("Current = %v, want 0.54", status.Current)
	}

	if status.PF == nil || *status.PF != 0.98 {
		t.Errorf("PF = %v, want 0.98", status.PF)
	}

	if status.Freq == nil || *status.Freq != 50.0 {
		t.Errorf("Freq = %v, want 50.0", status.Freq)
	}

	if status.AEnergy == nil {
		t.Fatal("AEnergy is nil")
	}

	if status.AEnergy.Total != 12345.67 {
		t.Errorf("AEnergy.Total = %f, want 12345.67", status.AEnergy.Total)
	}

	if len(status.AEnergy.ByMinute) != 3 {
		t.Errorf("AEnergy.ByMinute length = %d, want 3", len(status.AEnergy.ByMinute))
	}

	if status.Temperature == nil {
		t.Fatal("Temperature is nil")
	}

	if status.Temperature.TC == nil || *status.Temperature.TC != 45.2 {
		t.Errorf("Temperature.TC = %v, want 45.2", status.Temperature.TC)
	}

	if status.Temperature.TF == nil || *status.Temperature.TF != 113.4 {
		t.Errorf("Temperature.TF = %v, want 113.4", status.Temperature.TF)
	}

	if len(status.Errors) != 0 {
		t.Errorf("Errors length = %d, want 0", len(status.Errors))
	}
}

func TestSwitch_GetStatus_WithErrors(t *testing.T) {
	result := `{
		"id": 0,
		"source": "init",
		"output": false,
		"errors": ["overvoltage", "overtemp"]
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	sw := NewSwitch(client, 0)

	status, err := sw.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if len(status.Errors) != 2 {
		t.Errorf("Errors length = %d, want 2", len(status.Errors))
	}

	if status.Errors[0] != "overvoltage" {
		t.Errorf("Errors[0] = %q, want %q", status.Errors[0], "overvoltage")
	}

	if status.Errors[1] != "overtemp" {
		t.Errorf("Errors[1] = %q, want %q", status.Errors[1], "overtemp")
	}
}

func TestSwitch_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	sw := NewSwitch(client, 0)

	testComponentError(t, "GetStatus", func() error {
		_, err := sw.GetStatus(context.Background())
		return err
	})
}

func TestSwitch_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	sw := NewSwitch(client, 0)

	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := sw.GetStatus(context.Background())
		return err
	})
}
