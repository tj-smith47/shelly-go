package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

func TestNewVoltmeter(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	voltmeter := NewVoltmeter(client, 0)

	if voltmeter == nil {
		t.Fatal("NewVoltmeter returned nil")
	}

	if voltmeter.Type() != "voltmeter" {
		t.Errorf("Type() = %q, want %q", voltmeter.Type(), "voltmeter")
	}

	if voltmeter.ID() != 0 {
		t.Errorf("ID() = %d, want %d", voltmeter.ID(), 0)
	}

	if voltmeter.Key() != "voltmeter:0" {
		t.Errorf("Key() = %q, want %q", voltmeter.Key(), "voltmeter:0")
	}
}

func TestNewVoltmeter_DifferentIDs(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	tests := []struct {
		wantKey string
		id      int
	}{
		{id: 0, wantKey: "voltmeter:0"},
		{id: 1, wantKey: "voltmeter:1"},
		{id: 100, wantKey: "voltmeter:100"},
	}

	for _, tt := range tests {
		voltmeter := NewVoltmeter(client, tt.id)
		if voltmeter.Key() != tt.wantKey {
			t.Errorf("NewVoltmeter(%d).Key() = %q, want %q", tt.id, voltmeter.Key(), tt.wantKey)
		}
	}
}

func TestVoltmeter_GetConfig(t *testing.T) {
	tests := []struct {
		wantName     *string
		wantRepThr   *float64
		wantRange    *int
		wantXVoltage *VoltmeterXVoltageConfig
		name         string
		result       string
		wantID       int
	}{
		{
			name:       "basic config",
			result:     `{"id": 0, "name": "Voltmeter0", "report_thr": 2.5, "range": 0}`,
			wantID:     0,
			wantName:   ptr("Voltmeter0"),
			wantRepThr: ptr(2.5),
			wantRange:  ptr(0),
		},
		{
			name:   "minimal config",
			result: `{"id": 0}`,
			wantID: 0,
		},
		{
			name: "full config with xvoltage",
			result: `{
				"id": 0,
				"name": "Battery Sensor",
				"report_thr": 0.5,
				"range": 1,
				"xvoltage": {
					"expr": "x * 10",
					"unit": "mV"
				}
			}`,
			wantID:     0,
			wantName:   ptr("Battery Sensor"),
			wantRepThr: ptr(0.5),
			wantRange:  ptr(1),
			wantXVoltage: &VoltmeterXVoltageConfig{
				Expr: ptr("x * 10"),
				Unit: ptr("mV"),
			},
		},
		{
			name: "xvoltage with only expression",
			result: `{
				"id": 0,
				"xvoltage": {
					"expr": "x + 0.5"
				}
			}`,
			wantID: 0,
			wantXVoltage: &VoltmeterXVoltageConfig{
				Expr: ptr("x + 0.5"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Voltmeter.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			voltmeter := NewVoltmeter(client, 0)

			config, err := voltmeter.GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if config == nil {
				t.Fatal("GetConfig() returned nil config")
			}

			if config.ID != tt.wantID {
				t.Errorf("config.ID = %d, want %d", config.ID, tt.wantID)
			}

			if tt.wantName != nil {
				if config.Name == nil {
					t.Error("config.Name is nil, want non-nil")
				} else if *config.Name != *tt.wantName {
					t.Errorf("config.Name = %q, want %q", *config.Name, *tt.wantName)
				}
			}

			if tt.wantRepThr != nil {
				if config.ReportThr == nil {
					t.Error("config.ReportThr is nil, want non-nil")
				} else if *config.ReportThr != *tt.wantRepThr {
					t.Errorf("config.ReportThr = %.2f, want %.2f", *config.ReportThr, *tt.wantRepThr)
				}
			}

			if tt.wantRange != nil {
				if config.Range == nil {
					t.Error("config.Range is nil, want non-nil")
				} else if *config.Range != *tt.wantRange {
					t.Errorf("config.Range = %d, want %d", *config.Range, *tt.wantRange)
				}
			}

			if tt.wantXVoltage != nil {
				if config.XVoltage == nil {
					t.Error("config.XVoltage is nil, want non-nil")
				} else {
					if tt.wantXVoltage.Expr != nil {
						if config.XVoltage.Expr == nil {
							t.Error("config.XVoltage.Expr is nil, want non-nil")
						} else if *config.XVoltage.Expr != *tt.wantXVoltage.Expr {
							t.Errorf("config.XVoltage.Expr = %q, want %q", *config.XVoltage.Expr, *tt.wantXVoltage.Expr)
						}
					}
					if tt.wantXVoltage.Unit != nil {
						if config.XVoltage.Unit == nil {
							t.Error("config.XVoltage.Unit is nil, want non-nil")
						} else if *config.XVoltage.Unit != *tt.wantXVoltage.Unit {
							t.Errorf("config.XVoltage.Unit = %q, want %q", *config.XVoltage.Unit, *tt.wantXVoltage.Unit)
						}
					}
				}
			}
		})
	}
}

func TestVoltmeter_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	voltmeter := NewVoltmeter(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := voltmeter.GetConfig(context.Background())
		return err
	})
}

func TestVoltmeter_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	voltmeter := NewVoltmeter(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := voltmeter.GetConfig(context.Background())
		return err
	})
}

func TestVoltmeter_SetConfig(t *testing.T) {
	tests := []struct {
		config *VoltmeterConfig
		name   string
	}{
		{
			name: "set name only",
			config: &VoltmeterConfig{
				ID:   0,
				Name: ptr("My Voltmeter"),
			},
		},
		{
			name: "set report threshold",
			config: &VoltmeterConfig{
				ID:        0,
				ReportThr: ptr(1.5),
			},
		},
		{
			name: "set range",
			config: &VoltmeterConfig{
				ID:    0,
				Range: ptr(1),
			},
		},
		{
			name: "set xvoltage transformation",
			config: &VoltmeterConfig{
				ID: 0,
				XVoltage: &VoltmeterXVoltageConfig{
					Expr: ptr("x * 10"),
					Unit: ptr("mV"),
				},
			},
		},
		{
			name: "full configuration",
			config: &VoltmeterConfig{
				ID:        0,
				Name:      ptr("Battery Voltage"),
				ReportThr: ptr(0.5),
				Range:     ptr(0),
				XVoltage: &VoltmeterXVoltageConfig{
					Expr: ptr("(x - 2.5) * 100"),
					Unit: ptr("%"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Voltmeter.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Voltmeter.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			voltmeter := NewVoltmeter(client, 0)

			err := voltmeter.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestVoltmeter_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	voltmeter := NewVoltmeter(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return voltmeter.SetConfig(context.Background(), &VoltmeterConfig{ID: 0})
	})
}

func TestVoltmeter_GetStatus(t *testing.T) {
	tests := []struct {
		wantXVoltage *float64
		name         string
		result       string
		wantErrors   []string
		wantID       int
		wantVoltage  float64
	}{
		{
			name:        "basic voltage reading",
			result:      `{"id": 0, "voltage": 4.321}`,
			wantID:      0,
			wantVoltage: 4.321,
		},
		{
			name:         "voltage with xvoltage",
			result:       `{"id": 0, "voltage": 4.321, "xvoltage": 43.21}`,
			wantID:       0,
			wantVoltage:  4.321,
			wantXVoltage: ptr(43.21),
		},
		{
			name:        "high voltage",
			result:      `{"id": 0, "voltage": 12.5}`,
			wantID:      0,
			wantVoltage: 12.5,
		},
		{
			name:        "zero voltage",
			result:      `{"id": 0, "voltage": 0.0}`,
			wantID:      0,
			wantVoltage: 0.0,
		},
		{
			name:         "voltage with transformation result",
			result:       `{"id": 0, "voltage": 2.5, "xvoltage": 25.0}`,
			wantID:       0,
			wantVoltage:  2.5,
			wantXVoltage: ptr(25.0),
		},
		{
			name:        "voltage with errors",
			result:      `{"id": 0, "voltage": 0, "errors": ["read", "out_of_range"]}`,
			wantID:      0,
			wantVoltage: 0,
			wantErrors:  []string{"read", "out_of_range"},
		},
		{
			name:        "voltage with single error",
			result:      `{"id": 0, "voltage": 0, "errors": ["out_of_range"]}`,
			wantID:      0,
			wantVoltage: 0,
			wantErrors:  []string{"out_of_range"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Voltmeter.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			voltmeter := NewVoltmeter(client, 0)

			status, err := voltmeter.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if status.ID != tt.wantID {
				t.Errorf("status.ID = %d, want %d", status.ID, tt.wantID)
			}

			if status.Voltage != tt.wantVoltage {
				t.Errorf("status.Voltage = %.3f, want %.3f", status.Voltage, tt.wantVoltage)
			}

			if tt.wantXVoltage != nil {
				if status.XVoltage == nil {
					t.Error("status.XVoltage is nil, want non-nil")
				} else if *status.XVoltage != *tt.wantXVoltage {
					t.Errorf("status.XVoltage = %.3f, want %.3f", *status.XVoltage, *tt.wantXVoltage)
				}
			} else if status.XVoltage != nil {
				t.Errorf("status.XVoltage = %.3f, want nil", *status.XVoltage)
			}

			if len(tt.wantErrors) > 0 {
				if len(status.Errors) != len(tt.wantErrors) {
					t.Errorf("len(status.Errors) = %d, want %d", len(status.Errors), len(tt.wantErrors))
				} else {
					for i, e := range tt.wantErrors {
						if status.Errors[i] != e {
							t.Errorf("status.Errors[%d] = %q, want %q", i, status.Errors[i], e)
						}
					}
				}
			}
		})
	}
}

func TestVoltmeter_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	voltmeter := NewVoltmeter(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := voltmeter.GetStatus(context.Background())
		return err
	})
}

func TestVoltmeter_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	voltmeter := NewVoltmeter(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := voltmeter.GetStatus(context.Background())
		return err
	})
}

func TestVoltmeterConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config VoltmeterConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config serialization",
			config: VoltmeterConfig{
				ID:        0,
				Name:      ptr("Test Voltmeter"),
				ReportThr: ptr(1.0),
				Range:     ptr(0),
				XVoltage: &VoltmeterXVoltageConfig{
					Expr: ptr("x*2"),
					Unit: ptr("mV"),
				},
			},
			check: func(t *testing.T, data map[string]any) {
				id, ok := data["id"].(float64)
				if !ok || id != 0 {
					t.Errorf("id = %v, want 0", data["id"])
				}
				name, ok := data["name"].(string)
				if !ok || name != "Test Voltmeter" {
					t.Errorf("name = %v, want Test Voltmeter", data["name"])
				}
				reportThr, ok := data["report_thr"].(float64)
				if !ok || reportThr != 1.0 {
					t.Errorf("report_thr = %v, want 1.0", data["report_thr"])
				}
				rangeVal, ok := data["range"].(float64)
				if !ok || rangeVal != 0 {
					t.Errorf("range = %v, want 0", data["range"])
				}
				xv, ok := data["xvoltage"].(map[string]any)
				if !ok {
					t.Fatalf("xvoltage type assertion failed")
				}
				expr, ok := xv["expr"].(string)
				if !ok || expr != "x*2" {
					t.Errorf("xvoltage.expr = %v, want x*2", xv["expr"])
				}
				unit, ok := xv["unit"].(string)
				if !ok || unit != "mV" {
					t.Errorf("xvoltage.unit = %v, want mV", xv["unit"])
				}
			},
		},
		{
			name: "minimal config with only ID",
			config: VoltmeterConfig{
				ID: 0,
			},
			check: func(t *testing.T, data map[string]any) {
				id, ok := data["id"].(float64)
				if !ok || id != 0 {
					t.Errorf("id = %v, want 0", data["id"])
				}
				// Optional fields should not be present
				if _, ok := data["name"]; ok {
					t.Error("name should not be present for minimal config")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			tt.check(t, parsed)
		})
	}
}

func TestVoltmeterStatus_JSONDeserialization(t *testing.T) {
	tests := []struct {
		check func(t *testing.T, status *VoltmeterStatus)
		name  string
		json  string
	}{
		{
			name: "full status with all fields",
			json: `{"id": 0, "voltage": 5.5, "xvoltage": 55.0, "errors": ["read"]}`,
			check: func(t *testing.T, status *VoltmeterStatus) {
				if status.ID != 0 {
					t.Errorf("ID = %d, want 0", status.ID)
				}
				if status.Voltage != 5.5 {
					t.Errorf("Voltage = %f, want 5.5", status.Voltage)
				}
				if status.XVoltage == nil || *status.XVoltage != 55.0 {
					t.Errorf("XVoltage = %v, want 55.0", status.XVoltage)
				}
				if len(status.Errors) != 1 || status.Errors[0] != "read" {
					t.Errorf("Errors = %v, want [read]", status.Errors)
				}
			},
		},
		{
			name: "status with negative voltage",
			json: `{"id": 0, "voltage": -0.5}`,
			check: func(t *testing.T, status *VoltmeterStatus) {
				if status.Voltage != -0.5 {
					t.Errorf("Voltage = %f, want -0.5", status.Voltage)
				}
			},
		},
		{
			name: "status with high precision voltage",
			json: `{"id": 0, "voltage": 3.14159265}`,
			check: func(t *testing.T, status *VoltmeterStatus) {
				if status.Voltage != 3.14159265 {
					t.Errorf("Voltage = %f, want 3.14159265", status.Voltage)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status VoltmeterStatus
			if err := json.Unmarshal([]byte(tt.json), &status); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			tt.check(t, &status)
		})
	}
}

func TestVoltmeterXVoltageConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config VoltmeterXVoltageConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full xvoltage config",
			config: VoltmeterXVoltageConfig{
				Expr: ptr("x * 10"),
				Unit: ptr("mV"),
			},
			check: func(t *testing.T, data map[string]any) {
				expr, ok := data["expr"].(string)
				if !ok || expr != "x * 10" {
					t.Errorf("expr = %v, want 'x * 10'", data["expr"])
				}
				unit, ok := data["unit"].(string)
				if !ok || unit != "mV" {
					t.Errorf("unit = %v, want 'mV'", data["unit"])
				}
			},
		},
		{
			name: "expression only",
			config: VoltmeterXVoltageConfig{
				Expr: ptr("x + 1"),
			},
			check: func(t *testing.T, data map[string]any) {
				expr, ok := data["expr"].(string)
				if !ok || expr != "x + 1" {
					t.Errorf("expr = %v, want 'x + 1'", data["expr"])
				}
				if _, ok := data["unit"]; ok {
					t.Error("unit should not be present")
				}
			},
		},
		{
			name: "complex expression",
			config: VoltmeterXVoltageConfig{
				Expr: ptr("(x - 0.5) * 100 / 2.5"),
				Unit: ptr("%"),
			},
			check: func(t *testing.T, data map[string]any) {
				expr, ok := data["expr"].(string)
				if !ok || expr != "(x - 0.5) * 100 / 2.5" {
					t.Errorf("expr = %v, want complex expression", data["expr"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			tt.check(t, parsed)
		})
	}
}

func TestVoltmeter_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"id": 0, "voltage": 5.0}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	voltmeter := NewVoltmeter(client, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := voltmeter.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
