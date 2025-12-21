package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewRGBW(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	rgbw := NewRGBW(client, 0)

	if rgbw == nil {
		t.Fatal("NewRGBW returned nil")
	}

	if rgbw.Type() != "rgbw" {
		t.Errorf("Type() = %q, want %q", rgbw.Type(), "rgbw")
	}

	if rgbw.ID() != 0 {
		t.Errorf("ID() = %d, want %d", rgbw.ID(), 0)
	}

	if rgbw.Key() != "rgbw:0" {
		t.Errorf("Key() = %q, want %q", rgbw.Key(), "rgbw:0")
	}
}

func TestRGBW_Set(t *testing.T) {
	tests := []struct {
		name      string
		params    *RGBWSetParams
		result    string
		wantWasOn bool
	}{
		{
			name: "turn on with color and white",
			params: &RGBWSetParams{
				ID:    0,
				On:    ptr(true),
				RGB:   []int{255, 128, 0},
				White: ptr(100),
			},
			result:    `{"was_on": false}`,
			wantWasOn: false,
		},
		{
			name: "turn off",
			params: &RGBWSetParams{
				ID: 0,
				On: ptr(false),
			},
			result:    `{"was_on": true}`,
			wantWasOn: true,
		},
		{
			name: "with brightness and transition",
			params: &RGBWSetParams{
				ID:                 0,
				On:                 ptr(true),
				RGB:                []int{0, 0, 255},
				White:              ptr(50),
				Brightness:         ptr(90),
				TransitionDuration: ptr(1000.0),
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
					if method != "RGBW.Set" {
						t.Errorf("method = %q, want %q", method, "RGBW.Set")
					}
					return jsonrpcResponse(tt.result)
				},
			}

			client := rpc.NewClient(tr)
			rgbw := NewRGBW(client, 0)

			result, err := rgbw.Set(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			if result.WasOn != tt.wantWasOn {
				t.Errorf("WasOn = %v, want %v", result.WasOn, tt.wantWasOn)
			}
		})
	}
}

func TestRGBW_SetError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	_, err := rgbw.Set(context.Background(), &RGBWSetParams{On: ptr(true)})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRGBW_SetInvalidJSON(t *testing.T) {
	tr := invalidJSONTransport()
	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	_, err := rgbw.Set(context.Background(), &RGBWSetParams{On: ptr(true)})
	if err == nil {
		t.Fatal("expected json error")
	}
}

func TestRGBW_Toggle(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "RGBW.Toggle" {
				t.Errorf("method = %q, want %q", method, "RGBW.Toggle")
			}
			return jsonrpcResponse(`{"was_on": false}`)
		},
	}

	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	result, err := rgbw.Toggle(context.Background())
	if err != nil {
		t.Fatalf("Toggle() error = %v", err)
	}

	if result.WasOn {
		t.Error("WasOn = true, want false")
	}
}

func TestRGBW_ToggleError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	_, err := rgbw.Toggle(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRGBW_ToggleInvalidJSON(t *testing.T) {
	tr := invalidJSONTransport()
	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	_, err := rgbw.Toggle(context.Background())
	if err == nil {
		t.Fatal("expected json error")
	}
}

func TestRGBW_GetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "RGBW.GetConfig" {
				t.Errorf("method = %q, want %q", method, "RGBW.GetConfig")
			}
			return jsonrpcResponse(`{
				"id": 0,
				"name": "LED Strip",
				"initial_state": "restore_last",
				"auto_off": false,
				"default_brightness": 100,
				"default_rgb": [255, 255, 255],
				"default_white": 128
			}`)
		},
	}

	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	config, err := rgbw.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if config.ID != 0 {
		t.Errorf("ID = %d, want 0", config.ID)
	}

	if config.Name == nil || *config.Name != "LED Strip" {
		t.Errorf("Name = %v, want 'LED Strip'", config.Name)
	}

	if config.DefaultWhite == nil || *config.DefaultWhite != 128 {
		t.Errorf("DefaultWhite = %v, want 128", config.DefaultWhite)
	}
}

func TestRGBW_GetConfigError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	_, err := rgbw.GetConfig(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRGBW_SetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "RGBW.SetConfig" {
				t.Errorf("method = %q, want %q", method, "RGBW.SetConfig")
			}
			return jsonrpcResponse(`{}`)
		},
	}

	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	err := rgbw.SetConfig(context.Background(), &RGBWConfig{
		Name:         ptr("Updated Strip"),
		DefaultRGB:   []int{128, 128, 128},
		DefaultWhite: ptr(255),
	})
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestRGBW_SetConfigError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	err := rgbw.SetConfig(context.Background(), &RGBWConfig{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRGBW_GetStatus(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "RGBW.GetStatus" {
				t.Errorf("method = %q, want %q", method, "RGBW.GetStatus")
			}
			return jsonrpcResponse(`{
				"id": 0,
				"source": "http",
				"output": true,
				"brightness": 100,
				"rgb": [255, 200, 100],
				"white": 64,
				"apower": 12.5,
				"voltage": 24.0
			}`)
		},
	}

	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	status, err := rgbw.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if !status.Output {
		t.Error("Output = false, want true")
	}

	if status.Brightness == nil || *status.Brightness != 100 {
		t.Errorf("Brightness = %v, want 100", status.Brightness)
	}

	if len(status.RGB) != 3 || status.RGB[0] != 255 {
		t.Errorf("RGB = %v, want [255, 200, 100]", status.RGB)
	}

	if status.White == nil || *status.White != 64 {
		t.Errorf("White = %v, want 64", status.White)
	}
}

func TestRGBW_GetStatusError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	_, err := rgbw.GetStatus(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRGBW_ResetCounters(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "RGBW.ResetCounters" {
				t.Errorf("method = %q, want %q", method, "RGBW.ResetCounters")
			}
			return jsonrpcResponse(`{}`)
		},
	}

	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	err := rgbw.ResetCounters(context.Background(), nil)
	if err != nil {
		t.Fatalf("ResetCounters() error = %v", err)
	}
}

func TestRGBW_ResetCountersError(t *testing.T) {
	tr := errorTransport(errors.New("rpc error"))
	client := rpc.NewClient(tr)
	rgbw := NewRGBW(client, 0)

	err := rgbw.ResetCounters(context.Background(), []string{"aenergy"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRGBWNightModeConfig(t *testing.T) {
	// Test night mode configuration struct
	config := RGBWNightModeConfig{
		Enable:        ptr(true),
		Brightness:    ptr(30),
		RGB:           []int{255, 128, 0},
		White:         ptr(50),
		ActiveBetween: []string{"22:00", "06:00"},
	}

	if config.Enable == nil || !*config.Enable {
		t.Error("Enable should be true")
	}

	if config.Brightness == nil || *config.Brightness != 30 {
		t.Errorf("Brightness = %v, want 30", config.Brightness)
	}

	if config.White == nil || *config.White != 50 {
		t.Errorf("White = %v, want 50", config.White)
	}
}

func TestRGBWButtonPresets(t *testing.T) {
	// Test button preset configuration struct
	presets := RGBWButtonPresets{
		Brightness: ptr(100),
		RGB:        []int{255, 255, 255},
		White:      ptr(255),
	}

	if presets.Brightness == nil || *presets.Brightness != 100 {
		t.Errorf("Brightness = %v, want 100", presets.Brightness)
	}

	if len(presets.RGB) != 3 {
		t.Errorf("RGB length = %d, want 3", len(presets.RGB))
	}

	if presets.White == nil || *presets.White != 255 {
		t.Errorf("White = %v, want 255", presets.White)
	}
}
