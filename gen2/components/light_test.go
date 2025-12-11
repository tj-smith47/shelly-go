package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

func TestNewLight(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	light := NewLight(client, 0)

	if light == nil {
		t.Fatal("NewLight returned nil")
	}

	if light.Type() != "light" {
		t.Errorf("Type() = %q, want %q", light.Type(), "light")
	}

	if light.ID() != 0 {
		t.Errorf("ID() = %d, want %d", light.ID(), 0)
	}

	if light.Key() != "light:0" {
		t.Errorf("Key() = %q, want %q", light.Key(), "light:0")
	}
}

func TestLight_Set(t *testing.T) {
	tests := []struct {
		params    *LightSetParams
		wantWasOn *bool
		name      string
		result    string
	}{
		{
			name: "turn on with brightness",
			params: &LightSetParams{
				ID:         0,
				On:         ptr(true),
				Brightness: ptr(75),
			},
			result:    `{"was_on": false}`,
			wantWasOn: ptr(false),
		},
		{
			name: "turn off",
			params: &LightSetParams{
				ID: 0,
				On: ptr(false),
			},
			result:    `{"was_on": true}`,
			wantWasOn: ptr(true),
		},
		{
			name: "with transition",
			params: &LightSetParams{
				ID:                 0,
				On:                 ptr(true),
				Brightness:         ptr(50),
				TransitionDuration: ptr(1000),
			},
			result:    `{"was_on": false}`,
			wantWasOn: ptr(false),
		},
		{
			name:      "nil params uses component ID",
			params:    nil,
			result:    `{"was_on": true}`,
			wantWasOn: ptr(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Light.Set" {
						t.Errorf("method = %q, want %q", method, "Light.Set")
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			light := NewLight(client, 0)

			result, err := light.Set(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			if tt.wantWasOn != nil {
				if result.WasOn == nil {
					t.Error("result.WasOn is nil")
				} else if *result.WasOn != *tt.wantWasOn {
					t.Errorf("WasOn = %v, want %v", *result.WasOn, *tt.wantWasOn)
				}
			}
		})
	}
}

func TestLight_Set_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	light := NewLight(client, 0)

	testComponentError(t, "Set", func() error {
		_, err := light.Set(context.Background(), &LightSetParams{On: ptr(true)})
		return err
	})
}

func TestLight_Set_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	light := NewLight(client, 0)

	testComponentInvalidJSON(t, "Set", func() error {
		_, err := light.Set(context.Background(), &LightSetParams{On: ptr(true)})
		return err
	})
}

func TestLight_Toggle(t *testing.T) {
	tests := []struct {
		wantWasOn *bool
		name      string
		result    string
	}{
		{
			name:      "toggle from on to off",
			result:    `{"was_on": true}`,
			wantWasOn: ptr(true),
		},
		{
			name:      "toggle from off to on",
			result:    `{"was_on": false}`,
			wantWasOn: ptr(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Light.Toggle" {
						t.Errorf("method = %q, want %q", method, "Light.Toggle")
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			light := NewLight(client, 0)

			result, err := light.Toggle(context.Background())
			if err != nil {
				t.Fatalf("Toggle() error = %v", err)
			}

			if tt.wantWasOn != nil {
				if result.WasOn == nil {
					t.Error("result.WasOn is nil")
				} else if *result.WasOn != *tt.wantWasOn {
					t.Errorf("WasOn = %v, want %v", *result.WasOn, *tt.wantWasOn)
				}
			}
		})
	}
}

func TestLight_Toggle_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	light := NewLight(client, 0)

	testComponentError(t, "Toggle", func() error {
		_, err := light.Toggle(context.Background())
		return err
	})
}

func TestLight_Toggle_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	light := NewLight(client, 0)

	testComponentInvalidJSON(t, "Toggle", func() error {
		_, err := light.Toggle(context.Background())
		return err
	})
}

func TestLight_GetConfig(t *testing.T) {
	result := `{
		"id": 0,
		"name": "Living Room Light",
		"initial_state": "off",
		"auto_on": false,
		"auto_off": true,
		"auto_off_delay": 300.0,
		"transition_duration": 500,
		"default_brightness": 75,
		"night_mode": {
			"enable": true,
			"brightness": 20,
			"active_between": ["22:00", "06:00"]
		}
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "Light.GetConfig" {
				t.Errorf("method = %q, want %q", method, "Light.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	light := NewLight(client, 0)

	config, err := light.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if config.ID != 0 {
		t.Errorf("ID = %d, want 0", config.ID)
	}

	if config.Name == nil || *config.Name != "Living Room Light" {
		t.Errorf("Name = %v, want %q", config.Name, "Living Room Light")
	}

	if config.DefaultBrightness == nil || *config.DefaultBrightness != 75 {
		t.Errorf("DefaultBrightness = %v, want 75", config.DefaultBrightness)
	}

	if config.TransitionDuration == nil || *config.TransitionDuration != 500 {
		t.Errorf("TransitionDuration = %v, want 500", config.TransitionDuration)
	}

	if config.NightMode == nil {
		t.Fatal("NightMode is nil")
	}

	if config.NightMode.Enable == nil || !*config.NightMode.Enable {
		t.Errorf("NightMode.Enable = %v, want true", config.NightMode.Enable)
	}

	if config.NightMode.Brightness == nil || *config.NightMode.Brightness != 20 {
		t.Errorf("NightMode.Brightness = %v, want 20", config.NightMode.Brightness)
	}
}

func TestLight_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	light := NewLight(client, 0)

	testComponentError(t, "GetConfig", func() error {
		_, err := light.GetConfig(context.Background())
		return err
	})
}

func TestLight_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	light := NewLight(client, 0)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := light.GetConfig(context.Background())
		return err
	})
}

func TestLight_SetConfig(t *testing.T) {
	expectedConfig := &LightConfig{
		ID:                 0,
		Name:               ptr("Bedroom Light"),
		DefaultBrightness:  ptr(80),
		TransitionDuration: ptr(1000),
	}

	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "Light.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Light.SetConfig")
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	light := NewLight(client, 0)

	err := light.SetConfig(context.Background(), expectedConfig)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestLight_SetConfig_AutoSetID(t *testing.T) {
	config := &LightConfig{
		Name: ptr("Test"),
	}

	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	light := NewLight(client, 0)

	err := light.SetConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestLight_GetStatus(t *testing.T) {
	result := `{
		"id": 0,
		"source": "http",
		"output": true,
		"brightness": 75,
		"transition_duration": 500,
		"apower": 8.5,
		"voltage": 230.1,
		"current": 0.04,
		"temperature": {
			"tC": 35.2,
			"tF": 95.4
		},
		"errors": []
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "Light.GetStatus" {
				t.Errorf("method = %q, want %q", method, "Light.GetStatus")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	light := NewLight(client, 0)

	status, err := light.GetStatus(context.Background())
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

	if status.Brightness == nil || *status.Brightness != 75 {
		t.Errorf("Brightness = %v, want 75", status.Brightness)
	}

	if status.TransitionDuration == nil || *status.TransitionDuration != 500 {
		t.Errorf("TransitionDuration = %v, want 500", status.TransitionDuration)
	}

	if status.APower == nil || *status.APower != 8.5 {
		t.Errorf("APower = %v, want 8.5", status.APower)
	}

	if status.Temperature == nil {
		t.Fatal("Temperature is nil")
	}

	if status.Temperature.TC == nil || *status.Temperature.TC != 35.2 {
		t.Errorf("Temperature.TC = %v, want 35.2", status.Temperature.TC)
	}

	if len(status.Errors) != 0 {
		t.Errorf("Errors length = %d, want 0", len(status.Errors))
	}
}

func TestLight_GetStatus_WithErrors(t *testing.T) {
	result := `{
		"id": 0,
		"source": "init",
		"output": false,
		"errors": ["overtemp", "overpower"]
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	light := NewLight(client, 0)

	status, err := light.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if len(status.Errors) != 2 {
		t.Errorf("Errors length = %d, want 2", len(status.Errors))
	}

	if status.Errors[0] != "overtemp" {
		t.Errorf("Errors[0] = %q, want %q", status.Errors[0], "overtemp")
	}

	if status.Errors[1] != "overpower" {
		t.Errorf("Errors[1] = %q, want %q", status.Errors[1], "overpower")
	}
}

func TestLight_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	light := NewLight(client, 0)

	testComponentError(t, "GetStatus", func() error {
		_, err := light.GetStatus(context.Background())
		return err
	})
}

func TestLight_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	light := NewLight(client, 0)

	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := light.GetStatus(context.Background())
		return err
	})
}
