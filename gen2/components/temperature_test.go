package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

func TestNewTemperature(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	temp := NewTemperature(client, 0)

	if temp == nil {
		t.Fatal("NewTemperature returned nil")
	}

	if temp.Type() != "temperature" {
		t.Errorf("Type() = %q, want %q", temp.Type(), "temperature")
	}

	if temp.Key() != "temperature" {
		t.Errorf("Key() = %q, want %q", temp.Key(), "temperature")
	}

	if temp.Client() != client {
		t.Error("Client() did not return the expected client")
	}

	if temp.ID() != 0 {
		t.Errorf("ID() = %d, want 0", temp.ID())
	}
}

func TestTemperature_GetConfig(t *testing.T) {
	tests := []struct {
		wantName      *string
		wantReportThr *float64
		wantOffset    *float64
		name          string
		result        string
		id            int
	}{
		{
			name:          "full config",
			id:            0,
			result:        `{"id": 0, "name": "Living Room", "report_thr_C": 0.5, "offset_C": -0.2}`,
			wantName:      ptr("Living Room"),
			wantReportThr: ptrFloat(0.5),
			wantOffset:    ptrFloat(-0.2),
		},
		{
			name:   "minimal config",
			id:     1,
			result: `{"id": 1}`,
		},
		{
			name:     "named sensor",
			id:       0,
			result:   `{"id": 0, "name": "Outdoor Sensor"}`,
			wantName: ptr("Outdoor Sensor"),
		},
		{
			name:       "with offset only",
			id:         2,
			result:     `{"id": 2, "offset_C": 1.5}`,
			wantOffset: ptrFloat(1.5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Temperature.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			temp := NewTemperature(client, tt.id)

			config, err := temp.GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if config == nil {
				t.Fatal("GetConfig() returned nil config")
			}

			if config.ID != tt.id {
				t.Errorf("config.ID = %d, want %d", config.ID, tt.id)
			}

			if tt.wantName != nil {
				if config.Name == nil || *config.Name != *tt.wantName {
					t.Errorf("config.Name = %v, want %v", config.Name, *tt.wantName)
				}
			}

			if tt.wantReportThr != nil {
				if config.ReportThrC == nil || *config.ReportThrC != *tt.wantReportThr {
					t.Errorf("config.ReportThrC = %v, want %v", config.ReportThrC, *tt.wantReportThr)
				}
			}

			if tt.wantOffset != nil {
				if config.OffsetC == nil || *config.OffsetC != *tt.wantOffset {
					t.Errorf("config.OffsetC = %v, want %v", config.OffsetC, *tt.wantOffset)
				}
			}
		})
	}
}

func TestTemperature_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	temp := NewTemperature(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := temp.GetConfig(context.Background())
		return err
	})
}

func TestTemperature_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	temp := NewTemperature(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := temp.GetConfig(context.Background())
		return err
	})
}

func TestTemperature_SetConfig(t *testing.T) {
	tests := []struct {
		config *TemperatureConfig
		name   string
		id     int
	}{
		{
			name: "set name",
			id:   0,
			config: &TemperatureConfig{
				Name: ptr("Bedroom Sensor"),
			},
		},
		{
			name: "set report threshold",
			id:   1,
			config: &TemperatureConfig{
				ReportThrC: ptrFloat(1.0),
			},
		},
		{
			name: "set offset",
			id:   0,
			config: &TemperatureConfig{
				OffsetC: ptrFloat(-0.5),
			},
		},
		{
			name: "set all fields",
			id:   2,
			config: &TemperatureConfig{
				Name:       ptr("Garage Sensor"),
				ReportThrC: ptrFloat(0.25),
				OffsetC:    ptrFloat(0.1),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Temperature.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Temperature.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			temp := NewTemperature(client, tt.id)

			err := temp.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestTemperature_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	temp := NewTemperature(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return temp.SetConfig(context.Background(), &TemperatureConfig{})
	})
}

func TestTemperature_GetStatus(t *testing.T) {
	tests := []struct {
		wantTC *float64
		wantTF *float64
		name   string
		result string
		id     int
	}{
		{
			name:   "normal reading",
			id:     0,
			result: `{"id": 0, "tC": 23.5, "tF": 74.3}`,
			wantTC: ptrFloat(23.5),
			wantTF: ptrFloat(74.3),
		},
		{
			name:   "cold temperature",
			id:     0,
			result: `{"id": 0, "tC": -10.2, "tF": 13.64}`,
			wantTC: ptrFloat(-10.2),
			wantTF: ptrFloat(13.64),
		},
		{
			name:   "hot temperature",
			id:     1,
			result: `{"id": 1, "tC": 45.8, "tF": 114.44}`,
			wantTC: ptrFloat(45.8),
			wantTF: ptrFloat(114.44),
		},
		{
			name:   "sensor error",
			id:     0,
			result: `{"id": 0, "tC": null, "tF": null, "errors": ["read"]}`,
			wantTC: nil,
			wantTF: nil,
		},
		{
			name:   "zero temperature",
			id:     0,
			result: `{"id": 0, "tC": 0, "tF": 32}`,
			wantTC: ptrFloat(0),
			wantTF: ptrFloat(32),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Temperature.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			temp := NewTemperature(client, tt.id)

			status, err := temp.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if status.ID != tt.id {
				t.Errorf("status.ID = %d, want %d", status.ID, tt.id)
			}

			if tt.wantTC != nil {
				if status.TC == nil {
					t.Errorf("status.TC = nil, want %v", *tt.wantTC)
				} else if *status.TC != *tt.wantTC {
					t.Errorf("status.TC = %v, want %v", *status.TC, *tt.wantTC)
				}
			} else {
				if status.TC != nil {
					t.Errorf("status.TC = %v, want nil", *status.TC)
				}
			}

			if tt.wantTF != nil {
				if status.TF == nil {
					t.Errorf("status.TF = nil, want %v", *tt.wantTF)
				} else if *status.TF != *tt.wantTF {
					t.Errorf("status.TF = %v, want %v", *status.TF, *tt.wantTF)
				}
			} else {
				if status.TF != nil {
					t.Errorf("status.TF = %v, want nil", *status.TF)
				}
			}
		})
	}
}

func TestTemperature_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	temp := NewTemperature(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := temp.GetStatus(context.Background())
		return err
	})
}

func TestTemperature_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	temp := NewTemperature(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := temp.GetStatus(context.Background())
		return err
	})
}

func TestTemperatureConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config TemperatureConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: TemperatureConfig{
				ID:         0,
				Name:       ptr("Test Sensor"),
				ReportThrC: ptrFloat(0.5),
				OffsetC:    ptrFloat(-0.2),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["id"].(float64) != 0 {
					t.Errorf("id = %v, want 0", data["id"])
				}
				if data["name"].(string) != "Test Sensor" {
					t.Errorf("name = %v, want Test Sensor", data["name"])
				}
				if data["report_thr_C"].(float64) != 0.5 {
					t.Errorf("report_thr_C = %v, want 0.5", data["report_thr_C"])
				}
				if data["offset_C"].(float64) != -0.2 {
					t.Errorf("offset_C = %v, want -0.2", data["offset_C"])
				}
			},
		},
		{
			name: "minimal config",
			config: TemperatureConfig{
				ID: 1,
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["name"]; ok {
					t.Error("name should not be present")
				}
				if _, ok := data["report_thr_C"]; ok {
					t.Error("report_thr_C should not be present")
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

func TestTemperature_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"id": 0, "tC": 23.5, "tF": 74.3}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	temp := NewTemperature(client, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := temp.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
