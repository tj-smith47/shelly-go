package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

// extractParams is a helper to extract params from the RPC request for testing
func extractThermostatParams(params json.RawMessage) map[string]any {
	var result map[string]any
	if err := json.Unmarshal(params, &result); err != nil {
		return nil
	}
	return result
}

func TestNewThermostat(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	thermostat := NewThermostat(client, 0)

	if thermostat == nil {
		t.Fatal("expected non-nil Thermostat")
	}
	if thermostat.Client() != client {
		t.Error("client mismatch")
	}
	if thermostat.ID() != 0 {
		t.Errorf("expected ID 0, got %d", thermostat.ID())
	}
	if thermostat.Type() != "thermostat" {
		t.Errorf("expected type 'thermostat', got %s", thermostat.Type())
	}
	if thermostat.Key() != "thermostat" {
		t.Errorf("expected key 'thermostat', got %s", thermostat.Key())
	}
}

func TestThermostat_GetConfig(t *testing.T) {
	result := `{
		"id": 0,
		"enable": true,
		"target_C": 21.5,
		"thermostat_mode": "heat",
		"override_enable": false,
		"min_valve_position": 5,
		"default_boost_duration": 300,
		"default_override_duration": 1800,
		"default_override_target_C": 25.0,
		"temp_offset": 0.5,
		"temp_unit": "C"
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Thermostat.GetConfig" {
				t.Errorf("method = %q, want %q", method, "Thermostat.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	config, err := thermostat.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ID != 0 {
		t.Errorf("expected ID 0, got %d", config.ID)
	}
	if config.Enable == nil || *config.Enable != true {
		t.Error("expected Enable to be true")
	}
	if config.TargetC == nil || *config.TargetC != 21.5 {
		t.Errorf("expected TargetC 21.5, got %v", config.TargetC)
	}
	if config.ThermostatMode == nil || *config.ThermostatMode != "heat" {
		t.Errorf("expected ThermostatMode 'heat', got %v", config.ThermostatMode)
	}
	if config.MinValvePosition == nil || *config.MinValvePosition != 5 {
		t.Errorf("expected MinValvePosition 5, got %v", config.MinValvePosition)
	}
}

func TestThermostat_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	thermostat := NewThermostat(client, 0)

	testComponentError(t, "GetConfig", func() error {
		_, err := thermostat.GetConfig(context.Background())
		return err
	})
}

func TestThermostat_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	thermostat := NewThermostat(client, 0)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := thermostat.GetConfig(context.Background())
		return err
	})
}

func TestThermostat_SetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Thermostat.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Thermostat.SetConfig")
			}
			// Verify params structure
			paramsMap := extractThermostatParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			// ID is an int, JSON unmarshal gives float64
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 0 {
				t.Errorf("expected id 0, got %v", paramsMap["id"])
			}
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatal("expected config to be map[string]any")
			}
			if config["target_C"] != 22.0 {
				t.Errorf("expected target_C 22.0, got %v", config["target_C"])
			}
			if config["enable"] != true {
				t.Errorf("expected enable true, got %v", config["enable"])
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	enable := true
	targetC := 22.0
	err := thermostat.SetConfig(context.Background(), &ThermostatConfig{
		Enable:  &enable,
		TargetC: &targetC,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestThermostat_SetConfig_WithFlags(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			paramsMap := extractThermostatParams(req.GetParams())
			config := paramsMap["config"].(map[string]any)
			flags, ok := config["flags"].(map[string]any)
			if !ok {
				t.Fatal("expected flags map")
			}
			if flags["floor_heating"] != true {
				t.Errorf("expected floor_heating true, got %v", flags["floor_heating"])
			}
			if flags["auto_calibrate"] != true {
				t.Errorf("expected auto_calibrate true, got %v", flags["auto_calibrate"])
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	floorHeating := true
	autoCalibrate := true
	err := thermostat.SetConfig(context.Background(), &ThermostatConfig{
		Flags: &ThermostatFlags{
			FloorHeating:  &floorHeating,
			AutoCalibrate: &autoCalibrate,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestThermostat_SetConfig_AllFields(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			paramsMap := extractThermostatParams(req.GetParams())
			config := paramsMap["config"].(map[string]any)
			// Verify all fields are present
			expectedFields := []string{
				"enable", "target_C", "override_enable", "min_valve_position",
				"default_boost_duration", "default_override_duration",
				"default_override_target_C", "temp_offset", "humidity_offset",
				"temp_unit", "thermostat_mode",
			}
			for _, field := range expectedFields {
				if _, exists := config[field]; !exists {
					t.Errorf("expected field %s in config", field)
				}
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	enable := true
	targetC := 22.0
	overrideEnable := true
	minValve := 10
	boostDuration := 300
	overrideDuration := 1800
	overrideTarget := 25.0
	tempOffset := 0.5
	humidityOffset := -2.0
	tempUnit := "C"
	mode := "auto"

	err := thermostat.SetConfig(context.Background(), &ThermostatConfig{
		Enable:                  &enable,
		TargetC:                 &targetC,
		OverrideEnable:          &overrideEnable,
		MinValvePosition:        &minValve,
		DefaultBoostDuration:    &boostDuration,
		DefaultOverrideDuration: &overrideDuration,
		DefaultOverrideTargetC:  &overrideTarget,
		TempOffset:              &tempOffset,
		HumidityOffset:          &humidityOffset,
		TempUnit:                &tempUnit,
		ThermostatMode:          &mode,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestThermostat_GetStatus(t *testing.T) {
	result := `{
		"id": 0,
		"pos": 45,
		"steps": 1234,
		"current_C": 20.5,
		"current_F": 68.9,
		"target_C": 22.0,
		"target_F": 71.6,
		"schedule_rev": 3,
		"flags": [],
		"output": true
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Thermostat.GetStatus" {
				t.Errorf("method = %q, want %q", method, "Thermostat.GetStatus")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	status, err := thermostat.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.ID != 0 {
		t.Errorf("expected ID 0, got %d", status.ID)
	}
	if status.Pos == nil || *status.Pos != 45 {
		t.Errorf("expected Pos 45, got %v", status.Pos)
	}
	if status.Steps == nil || *status.Steps != 1234 {
		t.Errorf("expected Steps 1234, got %v", status.Steps)
	}
	if status.CurrentC == nil || *status.CurrentC != 20.5 {
		t.Errorf("expected CurrentC 20.5, got %v", status.CurrentC)
	}
	if status.TargetC == nil || *status.TargetC != 22.0 {
		t.Errorf("expected TargetC 22.0, got %v", status.TargetC)
	}
	if status.Output == nil || *status.Output != true {
		t.Errorf("expected Output true, got %v", status.Output)
	}
}

func TestThermostat_GetStatus_WithBoost(t *testing.T) {
	result := `{
		"id": 0,
		"pos": 100,
		"boost": {
			"started_at": 1702123456,
			"duration": 300
		}
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	status, err := thermostat.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Boost == nil {
		t.Fatal("expected Boost to be set")
	}
	if status.Boost.StartedAt != 1702123456 {
		t.Errorf("expected StartedAt 1702123456, got %d", status.Boost.StartedAt)
	}
	if status.Boost.Duration != 300 {
		t.Errorf("expected Duration 300, got %d", status.Boost.Duration)
	}
}

func TestThermostat_GetStatus_WithOverride(t *testing.T) {
	result := `{
		"id": 0,
		"pos": 75,
		"override": {
			"started_at": 1702123456,
			"duration": 1800
		}
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	status, err := thermostat.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Override == nil {
		t.Fatal("expected Override to be set")
	}
	if status.Override.StartedAt != 1702123456 {
		t.Errorf("expected StartedAt 1702123456, got %d", status.Override.StartedAt)
	}
	if status.Override.Duration != 1800 {
		t.Errorf("expected Duration 1800, got %d", status.Override.Duration)
	}
}

func TestThermostat_GetStatus_WithErrors(t *testing.T) {
	result := `{
		"id": 0,
		"pos": 0,
		"flags": ["not_calibrated", "not_mounted"],
		"errors": ["ext_temp_missing"]
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	status, err := thermostat.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(status.Flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(status.Flags))
	}
	if len(status.Errors) != 1 || status.Errors[0] != "ext_temp_missing" {
		t.Errorf("expected Errors ['ext_temp_missing'], got %v", status.Errors)
	}
}

func TestThermostat_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	thermostat := NewThermostat(client, 0)

	testComponentError(t, "GetStatus", func() error {
		_, err := thermostat.GetStatus(context.Background())
		return err
	})
}

func TestThermostat_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	thermostat := NewThermostat(client, 0)

	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := thermostat.GetStatus(context.Background())
		return err
	})
}

func TestThermostat_SetTarget(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Thermostat.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Thermostat.SetConfig")
			}
			paramsMap := extractThermostatParams(req.GetParams())
			config := paramsMap["config"].(map[string]any)
			if config["target_C"] != 22.0 {
				t.Errorf("expected target_C 22.0, got %v", config["target_C"])
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	err := thermostat.SetTarget(context.Background(), 22.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestThermostat_Enable(t *testing.T) {
	tests := []struct {
		name   string
		enable bool
	}{
		{"enable", true},
		{"disable", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					_ = req.GetMethod()
					paramsMap := extractThermostatParams(req.GetParams())
					config := paramsMap["config"].(map[string]any)
					if config["enable"] != tt.enable {
						t.Errorf("expected enable %v, got %v", tt.enable, config["enable"])
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			thermostat := NewThermostat(client, 0)

			err := thermostat.Enable(context.Background(), tt.enable)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestThermostat_SetMode(t *testing.T) {
	modes := []string{"heat", "cool", "auto"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					_ = req.GetMethod()
					paramsMap := extractThermostatParams(req.GetParams())
					config := paramsMap["config"].(map[string]any)
					if config["thermostat_mode"] != mode {
						t.Errorf("expected mode %s, got %v", mode, config["thermostat_mode"])
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			thermostat := NewThermostat(client, 0)

			err := thermostat.SetMode(context.Background(), mode)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestThermostat_Boost(t *testing.T) {
	t.Run("with duration", func(t *testing.T) {
		tr := &mockTransport{
			callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
				if method != "Thermostat.Boost" {
					t.Errorf("method = %q, want %q", method, "Thermostat.Boost")
				}
				paramsMap := extractThermostatParams(req.GetParams())
				if id, ok := paramsMap["id"].(float64); !ok || int(id) != 0 {
					t.Errorf("expected id 0, got %v", paramsMap["id"])
				}
				if dur, ok := paramsMap["duration"].(float64); !ok || int(dur) != 300 {
					t.Errorf("expected duration 300, got %v", paramsMap["duration"])
				}
				return jsonrpcResponse(`null`)
			},
		}
		client := rpc.NewClient(tr)
		thermostat := NewThermostat(client, 0)

		err := thermostat.Boost(context.Background(), 300)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("without duration", func(t *testing.T) {
		tr := &mockTransport{
			callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
				paramsMap := extractThermostatParams(req.GetParams())
				if id, ok := paramsMap["id"].(float64); !ok || int(id) != 0 {
					t.Errorf("expected id 0, got %v", paramsMap["id"])
				}
				if _, exists := paramsMap["duration"]; exists {
					t.Error("duration should not be present")
				}
				return jsonrpcResponse(`null`)
			},
		}
		client := rpc.NewClient(tr)
		thermostat := NewThermostat(client, 0)

		err := thermostat.Boost(context.Background(), 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestThermostat_CancelBoost(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Thermostat.CancelBoost" {
				t.Errorf("method = %q, want %q", method, "Thermostat.CancelBoost")
			}
			paramsMap := extractThermostatParams(req.GetParams())
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 0 {
				t.Errorf("expected id 0, got %v", paramsMap["id"])
			}
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	err := thermostat.CancelBoost(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestThermostat_Override(t *testing.T) {
	t.Run("with target and duration", func(t *testing.T) {
		tr := &mockTransport{
			callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
				if method != "Thermostat.Override" {
					t.Errorf("method = %q, want %q", method, "Thermostat.Override")
				}
				paramsMap := extractThermostatParams(req.GetParams())
				if id, ok := paramsMap["id"].(float64); !ok || int(id) != 0 {
					t.Errorf("expected id 0, got %v", paramsMap["id"])
				}
				if paramsMap["target_C"] != 25.0 {
					t.Errorf("expected target_C 25.0, got %v", paramsMap["target_C"])
				}
				if dur, ok := paramsMap["duration"].(float64); !ok || int(dur) != 1800 {
					t.Errorf("expected duration 1800, got %v", paramsMap["duration"])
				}
				return jsonrpcResponse(`null`)
			},
		}
		client := rpc.NewClient(tr)
		thermostat := NewThermostat(client, 0)

		err := thermostat.Override(context.Background(), 25.0, 1800)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("without optional params", func(t *testing.T) {
		tr := &mockTransport{
			callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
				paramsMap := extractThermostatParams(req.GetParams())
				if id, ok := paramsMap["id"].(float64); !ok || int(id) != 0 {
					t.Errorf("expected id 0, got %v", paramsMap["id"])
				}
				if _, exists := paramsMap["target_C"]; exists {
					t.Error("target_C should not be present")
				}
				if _, exists := paramsMap["duration"]; exists {
					t.Error("duration should not be present")
				}
				return jsonrpcResponse(`null`)
			},
		}
		client := rpc.NewClient(tr)
		thermostat := NewThermostat(client, 0)

		err := thermostat.Override(context.Background(), 0, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestThermostat_CancelOverride(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Thermostat.CancelOverride" {
				t.Errorf("method = %q, want %q", method, "Thermostat.CancelOverride")
			}
			paramsMap := extractThermostatParams(req.GetParams())
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 0 {
				t.Errorf("expected id 0, got %v", paramsMap["id"])
			}
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	err := thermostat.CancelOverride(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestThermostat_Calibrate(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Thermostat.Calibrate" {
				t.Errorf("method = %q, want %q", method, "Thermostat.Calibrate")
			}
			paramsMap := extractThermostatParams(req.GetParams())
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 0 {
				t.Errorf("expected id 0, got %v", paramsMap["id"])
			}
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	thermostat := NewThermostat(client, 0)

	err := thermostat.Calibrate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestThermostatConfig_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"id": 0,
		"enable": true,
		"target_C": 21.5,
		"override_enable": false,
		"min_valve_position": 5,
		"default_boost_duration": 300,
		"default_override_duration": 1800,
		"default_override_target_C": 25.0,
		"temp_offset": 0.5,
		"humidity_offset": -1.0,
		"temp_unit": "C",
		"thermostat_mode": "heat",
		"flags": {
			"floor_heating": false,
			"accel": true,
			"auto_calibrate": true,
			"anticlog": true
		}
	}`

	var config ThermostatConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if config.ID != 0 {
		t.Errorf("expected ID 0, got %d", config.ID)
	}
	if config.Enable == nil || *config.Enable != true {
		t.Error("expected Enable true")
	}
	if config.TargetC == nil || *config.TargetC != 21.5 {
		t.Error("expected TargetC 21.5")
	}
	if config.TempUnit == nil || *config.TempUnit != "C" {
		t.Error("expected TempUnit 'C'")
	}
	if config.Flags == nil {
		t.Fatal("expected Flags to be set")
	}
	if config.Flags.Accel == nil || *config.Flags.Accel != true {
		t.Error("expected Accel true")
	}
}

func TestThermostatStatus_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"id": 0,
		"pos": 45,
		"steps": 1234,
		"current_C": 20.5,
		"current_F": 68.9,
		"target_C": 22.0,
		"target_F": 71.6,
		"current_humidity": 45.5,
		"target_humidity": 50.0,
		"schedule_rev": 3,
		"flags": ["not_calibrated"],
		"boost": {
			"started_at": 1702123456,
			"duration": 300
		},
		"output": true,
		"errors": ["ext_temp_missing"]
	}`

	var status ThermostatStatus
	err := json.Unmarshal([]byte(jsonData), &status)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if status.ID != 0 {
		t.Errorf("expected ID 0, got %d", status.ID)
	}
	if status.Pos == nil || *status.Pos != 45 {
		t.Error("expected Pos 45")
	}
	if status.Steps == nil || *status.Steps != 1234 {
		t.Error("expected Steps 1234")
	}
	if status.CurrentC == nil || *status.CurrentC != 20.5 {
		t.Error("expected CurrentC 20.5")
	}
	if status.CurrentHumidity == nil || *status.CurrentHumidity != 45.5 {
		t.Error("expected CurrentHumidity 45.5")
	}
	if len(status.Flags) != 1 || status.Flags[0] != "not_calibrated" {
		t.Error("expected Flags ['not_calibrated']")
	}
	if status.Boost == nil {
		t.Fatal("expected Boost to be set")
	}
	if status.Boost.StartedAt != 1702123456 {
		t.Error("expected Boost.StartedAt 1702123456")
	}
	if status.Output == nil || *status.Output != true {
		t.Error("expected Output true")
	}
	if len(status.Errors) != 1 || status.Errors[0] != "ext_temp_missing" {
		t.Error("expected Errors ['ext_temp_missing']")
	}
}

func TestThermostatFlags_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"floor_heating": true,
		"accel": false,
		"auto_calibrate": true,
		"anticlog": false
	}`

	var flags ThermostatFlags
	err := json.Unmarshal([]byte(jsonData), &flags)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if flags.FloorHeating == nil || *flags.FloorHeating != true {
		t.Error("expected FloorHeating true")
	}
	if flags.Accel == nil || *flags.Accel != false {
		t.Error("expected Accel false")
	}
	if flags.AutoCalibrate == nil || *flags.AutoCalibrate != true {
		t.Error("expected AutoCalibrate true")
	}
	if flags.AntiClog == nil || *flags.AntiClog != false {
		t.Error("expected AntiClog false")
	}
}
