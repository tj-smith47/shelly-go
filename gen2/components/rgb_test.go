package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewRGB(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	rgb := NewRGB(client, 0)

	if rgb == nil {
		t.Fatal("NewRGB returned nil")
	}

	if rgb.Type() != "rgb" {
		t.Errorf("Type() = %q, want %q", rgb.Type(), "rgb")
	}

	if rgb.ID() != 0 {
		t.Errorf("ID() = %d, want %d", rgb.ID(), 0)
	}

	if rgb.Key() != "rgb:0" {
		t.Errorf("Key() = %q, want %q", rgb.Key(), "rgb:0")
	}
}

func TestRGB_Set(t *testing.T) {
	tests := []struct {
		name      string
		params    *RGBSetParams
		result    string
		wantWasOn bool
	}{
		{
			name: "turn on with color",
			params: &RGBSetParams{
				ID:  0,
				On:  ptr(true),
				RGB: []int{255, 0, 0},
			},
			result:    `{"was_on": false}`,
			wantWasOn: false,
		},
		{
			name: "turn off",
			params: &RGBSetParams{
				ID: 0,
				On: ptr(false),
			},
			result:    `{"was_on": true}`,
			wantWasOn: true,
		},
		{
			name: "with brightness and transition",
			params: &RGBSetParams{
				ID:                 0,
				On:                 ptr(true),
				RGB:                []int{0, 255, 0},
				Brightness:         ptr(75),
				TransitionDuration: ptr(500.0),
			},
			result:    `{"was_on": false}`,
			wantWasOn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "RGB.Set" {
						t.Errorf("method = %q, want %q", method, "RGB.Set")
					}
					return jsonrpcResponse(tt.result)
				},
			}

			client := rpc.NewClient(tr)
			rgb := NewRGB(client, 0)

			result, err := rgb.Set(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			if result.WasOn != tt.wantWasOn {
				t.Errorf("WasOn = %v, want %v", result.WasOn, tt.wantWasOn)
			}
		})
	}
}

func TestRGB_SetError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	_, err := rgb.Set(context.Background(), &RGBSetParams{On: ptr(true)})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRGB_SetInvalidJSON(t *testing.T) {
	tr := invalidJSONTransport()
	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	_, err := rgb.Set(context.Background(), &RGBSetParams{On: ptr(true)})
	if err == nil {
		t.Fatal("expected json error")
	}
}

func TestRGB_Toggle(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "RGB.Toggle" {
				t.Errorf("method = %q, want %q", method, "RGB.Toggle")
			}
			return jsonrpcResponse(`{"was_on": true}`)
		},
	}

	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	result, err := rgb.Toggle(context.Background())
	if err != nil {
		t.Fatalf("Toggle() error = %v", err)
	}

	if !result.WasOn {
		t.Error("WasOn = false, want true")
	}
}

func TestRGB_ToggleError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	_, err := rgb.Toggle(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRGB_ToggleInvalidJSON(t *testing.T) {
	tr := invalidJSONTransport()
	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	_, err := rgb.Toggle(context.Background())
	if err == nil {
		t.Fatal("expected json error")
	}
}

func TestRGB_GetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "RGB.GetConfig" {
				t.Errorf("method = %q, want %q", method, "RGB.GetConfig")
			}
			return jsonrpcResponse(`{
				"id": 0,
				"name": "Accent Light",
				"initial_state": "restore_last",
				"auto_off": true,
				"auto_off_delay": 3600,
				"default_brightness": 100,
				"default_rgb": [255, 128, 0]
			}`)
		},
	}

	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	config, err := rgb.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if config.ID != 0 {
		t.Errorf("ID = %d, want 0", config.ID)
	}

	if config.Name == nil || *config.Name != "Accent Light" {
		t.Errorf("Name = %v, want 'Accent Light'", config.Name)
	}

	if len(config.DefaultRGB) != 3 || config.DefaultRGB[0] != 255 {
		t.Errorf("DefaultRGB = %v, want [255, 128, 0]", config.DefaultRGB)
	}
}

func TestRGB_GetConfigError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	_, err := rgb.GetConfig(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRGB_SetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "RGB.SetConfig" {
				t.Errorf("method = %q, want %q", method, "RGB.SetConfig")
			}
			return jsonrpcResponse(`{}`)
		},
	}

	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	err := rgb.SetConfig(context.Background(), &RGBConfig{
		Name:       ptr("Updated Name"),
		DefaultRGB: []int{255, 255, 255},
	})
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestRGB_SetConfigError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	err := rgb.SetConfig(context.Background(), &RGBConfig{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRGB_GetStatus(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "RGB.GetStatus" {
				t.Errorf("method = %q, want %q", method, "RGB.GetStatus")
			}
			return jsonrpcResponse(`{
				"id": 0,
				"source": "init",
				"output": true,
				"brightness": 80,
				"rgb": [255, 0, 0],
				"apower": 5.5,
				"voltage": 230.0
			}`)
		},
	}

	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	status, err := rgb.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if !status.Output {
		t.Error("Output = false, want true")
	}

	if status.Brightness == nil || *status.Brightness != 80 {
		t.Errorf("Brightness = %v, want 80", status.Brightness)
	}

	if len(status.RGB) != 3 || status.RGB[0] != 255 {
		t.Errorf("RGB = %v, want [255, 0, 0]", status.RGB)
	}
}

func TestRGB_GetStatusError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	_, err := rgb.GetStatus(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRGB_ResetCounters(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "RGB.ResetCounters" {
				t.Errorf("method = %q, want %q", method, "RGB.ResetCounters")
			}
			return jsonrpcResponse(`{}`)
		},
	}

	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	err := rgb.ResetCounters(context.Background(), []string{"aenergy"})
	if err != nil {
		t.Fatalf("ResetCounters() error = %v", err)
	}
}

func TestRGB_ResetCountersError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgb := NewRGB(client, 0)

	err := rgb.ResetCounters(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
