package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

// extractUIParams is a helper to extract params from the RPC request for testing
func extractUIParams(params any) map[string]any {
	req, ok := params.(*rpc.Request)
	if !ok {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(req.Params, &result); err != nil {
		return nil
	}
	return result
}

func TestNewUI(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	ui := NewUI(client)

	if ui == nil {
		t.Fatal("expected non-nil UI")
	}
	if ui.Client() != client {
		t.Error("client mismatch")
	}
	if ui.Type() != "ui" {
		t.Errorf("expected type 'ui', got %s", ui.Type())
	}
	if ui.Key() != "ui" {
		t.Errorf("expected key 'ui', got %s", ui.Key())
	}
}

func TestUI_GetConfig(t *testing.T) {
	result := `{
		"idle_brightness": 50,
		"lock": false,
		"temp_units": "C",
		"flip": false,
		"brightness": 7
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "Ui.GetConfig" {
				t.Errorf("method = %q, want %q", method, "Ui.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	ui := NewUI(client)

	config, err := ui.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.IdleBrightness == nil || *config.IdleBrightness != 50 {
		t.Errorf("expected IdleBrightness 50, got %v", config.IdleBrightness)
	}
	if config.Lock == nil || *config.Lock != false {
		t.Errorf("expected Lock false, got %v", config.Lock)
	}
	if config.TempUnits == nil || *config.TempUnits != "C" {
		t.Errorf("expected TempUnits 'C', got %v", config.TempUnits)
	}
}

func TestUI_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	ui := NewUI(client)

	testComponentError(t, "GetConfig", func() error {
		_, err := ui.GetConfig(context.Background())
		return err
	})
}

func TestUI_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	ui := NewUI(client)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := ui.GetConfig(context.Background())
		return err
	})
}

func TestUI_SetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "Ui.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Ui.SetConfig")
			}
			paramsMap := extractUIParams(params)
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatal("expected config map")
			}
			if brightness, ok := config["idle_brightness"].(float64); !ok || int(brightness) != 75 {
				t.Errorf("expected idle_brightness 75, got %v", config["idle_brightness"])
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	ui := NewUI(client)

	brightness := 75
	err := ui.SetConfig(context.Background(), &UIConfig{
		IdleBrightness: &brightness,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUI_SetConfig_AllFields(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			paramsMap := extractUIParams(params)
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatalf("config type assertion failed")
			}
			expectedFields := []string{"idle_brightness", "lock", "temp_units", "flip", "brightness"}
			for _, field := range expectedFields {
				if _, exists := config[field]; !exists {
					t.Errorf("expected field %s in config", field)
				}
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	ui := NewUI(client)

	idleBrightness := 50
	lock := true
	tempUnits := "F"
	flip := true
	brightness := 5

	err := ui.SetConfig(context.Background(), &UIConfig{
		IdleBrightness: &idleBrightness,
		Lock:           &lock,
		TempUnits:      &tempUnits,
		Flip:           &flip,
		Brightness:     &brightness,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUI_SetIdleBrightness(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			paramsMap := extractUIParams(params)
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatalf("config type assertion failed")
			}
			if brightness, ok := config["idle_brightness"].(float64); !ok || int(brightness) != 30 {
				t.Errorf("expected idle_brightness 30, got %v", config["idle_brightness"])
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	ui := NewUI(client)

	err := ui.SetIdleBrightness(context.Background(), 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUI_SetLock(t *testing.T) {
	tests := []struct {
		name string
		lock bool
	}{
		{"enable lock", true},
		{"disable lock", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					paramsMap := extractUIParams(params)
					config, ok := paramsMap["config"].(map[string]any)
					if !ok {
						t.Fatalf("config type assertion failed")
					}
					if config["lock"] != tt.lock {
						t.Errorf("expected lock %v, got %v", tt.lock, config["lock"])
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			ui := NewUI(client)

			err := ui.SetLock(context.Background(), tt.lock)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUI_SetTempUnits(t *testing.T) {
	units := []string{"C", "F"}

	for _, unit := range units {
		t.Run(unit, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					paramsMap := extractUIParams(params)
					config, ok := paramsMap["config"].(map[string]any)
					if !ok {
						t.Fatalf("config type assertion failed")
					}
					if config["temp_units"] != unit {
						t.Errorf("expected temp_units %s, got %v", unit, config["temp_units"])
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			ui := NewUI(client)

			err := ui.SetTempUnits(context.Background(), unit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUIConfig_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"idle_brightness": 75,
		"lock": true,
		"temp_units": "F",
		"flip": true,
		"brightness": 5
	}`

	var config UIConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if config.IdleBrightness == nil || *config.IdleBrightness != 75 {
		t.Error("expected IdleBrightness 75")
	}
	if config.Lock == nil || *config.Lock != true {
		t.Error("expected Lock true")
	}
	if config.TempUnits == nil || *config.TempUnits != "F" {
		t.Error("expected TempUnits 'F'")
	}
	if config.Flip == nil || *config.Flip != true {
		t.Error("expected Flip true")
	}
	if config.Brightness == nil || *config.Brightness != 5 {
		t.Error("expected Brightness 5")
	}
}

// PlugsUI Tests

func TestNewPlugsUI(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	plugsUI := NewPlugsUI(client)

	if plugsUI == nil {
		t.Fatal("expected non-nil PlugsUI")
	}
	if plugsUI.Client() != client {
		t.Error("client mismatch")
	}
	if plugsUI.Type() != "plugs_ui" {
		t.Errorf("expected type 'plugs_ui', got %s", plugsUI.Type())
	}
	if plugsUI.Key() != "plugs_ui" {
		t.Errorf("expected key 'plugs_ui', got %s", plugsUI.Key())
	}
}

func TestPlugsUI_GetConfig(t *testing.T) {
	result := `{
		"leds": {
			"mode": "power",
			"brightness": 80,
			"colors": [
				{"power": 0, "rgb": 65280},
				{"power": 1000, "rgb": 16711680}
			]
		}
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "PLUGS_UI.GetConfig" {
				t.Errorf("method = %q, want %q", method, "PLUGS_UI.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	plugsUI := NewPlugsUI(client)

	config, err := plugsUI.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.LEDs == nil {
		t.Fatal("expected LEDs to be set")
	}
	if config.LEDs.Mode == nil || *config.LEDs.Mode != "power" {
		t.Errorf("expected Mode 'power', got %v", config.LEDs.Mode)
	}
	if config.LEDs.Brightness == nil || *config.LEDs.Brightness != 80 {
		t.Errorf("expected Brightness 80, got %v", config.LEDs.Brightness)
	}
	if len(config.LEDs.Colors) != 2 {
		t.Errorf("expected 2 colors, got %d", len(config.LEDs.Colors))
	}
}

func TestPlugsUI_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	plugsUI := NewPlugsUI(client)

	testComponentError(t, "GetConfig", func() error {
		_, err := plugsUI.GetConfig(context.Background())
		return err
	})
}

func TestPlugsUI_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	plugsUI := NewPlugsUI(client)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := plugsUI.GetConfig(context.Background())
		return err
	})
}

func TestPlugsUI_SetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "PLUGS_UI.SetConfig" {
				t.Errorf("method = %q, want %q", method, "PLUGS_UI.SetConfig")
			}
			paramsMap := extractUIParams(params)
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatal("expected config map")
			}
			leds, ok := config["leds"].(map[string]any)
			if !ok {
				t.Fatal("expected leds map")
			}
			if leds["mode"] != "switch" {
				t.Errorf("expected mode 'switch', got %v", leds["mode"])
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	plugsUI := NewPlugsUI(client)

	mode := "switch"
	err := plugsUI.SetConfig(context.Background(), &PlugsUIConfig{
		LEDs: &PlugsUILEDConfig{
			Mode: &mode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlugsUI_SetConfig_WithColors(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			paramsMap := extractUIParams(params)
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatalf("config type assertion failed")
			}
			leds, ok := config["leds"].(map[string]any)
			if !ok {
				t.Fatalf("leds type assertion failed")
			}
			colors, ok := leds["colors"].([]any)
			if !ok {
				t.Fatal("expected colors array")
			}
			if len(colors) != 2 {
				t.Errorf("expected 2 colors, got %d", len(colors))
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	plugsUI := NewPlugsUI(client)

	err := plugsUI.SetConfig(context.Background(), &PlugsUIConfig{
		LEDs: &PlugsUILEDConfig{
			Colors: []PlugsUIColor{
				{Power: 0, RGB: 0x00FF00},
				{Power: 1000, RGB: 0xFF0000},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlugsUI_SetLEDMode(t *testing.T) {
	modes := []string{"power", "switch", "off"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					paramsMap := extractUIParams(params)
					config, ok := paramsMap["config"].(map[string]any)
					if !ok {
						t.Fatalf("config type assertion failed")
					}
					leds, ok := config["leds"].(map[string]any)
					if !ok {
						t.Fatalf("leds type assertion failed")
					}
					if leds["mode"] != mode {
						t.Errorf("expected mode %s, got %v", mode, leds["mode"])
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			plugsUI := NewPlugsUI(client)

			err := plugsUI.SetLEDMode(context.Background(), mode)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPlugsUI_SetLEDBrightness(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			paramsMap := extractUIParams(params)
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatalf("config type assertion failed")
			}
			leds, ok := config["leds"].(map[string]any)
			if !ok {
				t.Fatalf("leds type assertion failed")
			}
			if brightness, ok := leds["brightness"].(float64); !ok || int(brightness) != 60 {
				t.Errorf("expected brightness 60, got %v", leds["brightness"])
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	plugsUI := NewPlugsUI(client)

	err := plugsUI.SetLEDBrightness(context.Background(), 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlugsUIConfig_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"leds": {
			"mode": "power",
			"brightness": 100,
			"colors": [
				{"power": 0, "rgb": 65280},
				{"power": 500, "rgb": 16776960},
				{"power": 1000, "rgb": 16711680}
			]
		}
	}`

	var config PlugsUIConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if config.LEDs == nil {
		t.Fatal("expected LEDs to be set")
	}
	if config.LEDs.Mode == nil || *config.LEDs.Mode != "power" {
		t.Error("expected Mode 'power'")
	}
	if config.LEDs.Brightness == nil || *config.LEDs.Brightness != 100 {
		t.Error("expected Brightness 100")
	}
	if len(config.LEDs.Colors) != 3 {
		t.Error("expected 3 colors")
	}
	if config.LEDs.Colors[0].Power != 0 {
		t.Error("expected first color power 0")
	}
	if config.LEDs.Colors[0].RGB != 65280 {
		t.Error("expected first color RGB 65280 (green)")
	}
}
