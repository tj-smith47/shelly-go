package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewHumidity(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	humidity := NewHumidity(client, 0)

	if humidity == nil {
		t.Fatal("NewHumidity returned nil")
	}

	if humidity.Type() != "humidity" {
		t.Errorf("Type() = %q, want %q", humidity.Type(), "humidity")
	}

	if humidity.Key() != "humidity" {
		t.Errorf("Key() = %q, want %q", humidity.Key(), "humidity")
	}

	if humidity.Client() != client {
		t.Error("Client() did not return the expected client")
	}

	if humidity.ID() != 0 {
		t.Errorf("ID() = %d, want 0", humidity.ID())
	}
}

func TestHumidity_GetConfig(t *testing.T) {
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
			result:        `{"id": 0, "name": "Living Room", "report_thr": 5.0, "offset": -2.5}`,
			wantName:      ptr("Living Room"),
			wantReportThr: ptrFloat(5.0),
			wantOffset:    ptrFloat(-2.5),
		},
		{
			name:   "minimal config",
			id:     1,
			result: `{"id": 1}`,
		},
		{
			name:     "named sensor",
			id:       0,
			result:   `{"id": 0, "name": "Bathroom Sensor"}`,
			wantName: ptr("Bathroom Sensor"),
		},
		{
			name:       "with offset only",
			id:         2,
			result:     `{"id": 2, "offset": 3.0}`,
			wantOffset: ptrFloat(3.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Humidity.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			humidity := NewHumidity(client, tt.id)

			config, err := humidity.GetConfig(context.Background())
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
				if config.ReportThr == nil || *config.ReportThr != *tt.wantReportThr {
					t.Errorf("config.ReportThr = %v, want %v", config.ReportThr, *tt.wantReportThr)
				}
			}

			if tt.wantOffset != nil {
				if config.Offset == nil || *config.Offset != *tt.wantOffset {
					t.Errorf("config.Offset = %v, want %v", config.Offset, *tt.wantOffset)
				}
			}
		})
	}
}

func TestHumidity_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	humidity := NewHumidity(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := humidity.GetConfig(context.Background())
		return err
	})
}

func TestHumidity_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	humidity := NewHumidity(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := humidity.GetConfig(context.Background())
		return err
	})
}

func TestHumidity_SetConfig(t *testing.T) {
	tests := []struct {
		config *HumidityConfig
		name   string
		id     int
	}{
		{
			name: "set name",
			id:   0,
			config: &HumidityConfig{
				Name: ptr("Bedroom Sensor"),
			},
		},
		{
			name: "set report threshold",
			id:   1,
			config: &HumidityConfig{
				ReportThr: ptrFloat(10.0),
			},
		},
		{
			name: "set offset",
			id:   0,
			config: &HumidityConfig{
				Offset: ptrFloat(-5.0),
			},
		},
		{
			name: "set all fields",
			id:   2,
			config: &HumidityConfig{
				Name:      ptr("Basement Sensor"),
				ReportThr: ptrFloat(2.5),
				Offset:    ptrFloat(1.0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Humidity.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Humidity.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			humidity := NewHumidity(client, tt.id)

			err := humidity.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestHumidity_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	humidity := NewHumidity(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return humidity.SetConfig(context.Background(), &HumidityConfig{})
	})
}

func TestHumidity_GetStatus(t *testing.T) {
	tests := []struct {
		wantRH *float64
		name   string
		result string
		id     int
	}{
		{
			name:   "normal reading",
			id:     0,
			result: `{"id": 0, "rh": 55.2}`,
			wantRH: ptrFloat(55.2),
		},
		{
			name:   "low humidity",
			id:     0,
			result: `{"id": 0, "rh": 15.5}`,
			wantRH: ptrFloat(15.5),
		},
		{
			name:   "high humidity",
			id:     1,
			result: `{"id": 1, "rh": 95.8}`,
			wantRH: ptrFloat(95.8),
		},
		{
			name:   "sensor error",
			id:     0,
			result: `{"id": 0, "rh": null, "errors": ["read"]}`,
			wantRH: nil,
		},
		{
			name:   "zero humidity",
			id:     0,
			result: `{"id": 0, "rh": 0}`,
			wantRH: ptrFloat(0),
		},
		{
			name:   "100% humidity",
			id:     0,
			result: `{"id": 0, "rh": 100}`,
			wantRH: ptrFloat(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Humidity.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			humidity := NewHumidity(client, tt.id)

			status, err := humidity.GetStatus(context.Background())
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

			if tt.wantRH != nil {
				if status.RH == nil {
					t.Errorf("status.RH = nil, want %v", *tt.wantRH)
				} else if *status.RH != *tt.wantRH {
					t.Errorf("status.RH = %v, want %v", *status.RH, *tt.wantRH)
				}
			} else {
				if status.RH != nil {
					t.Errorf("status.RH = %v, want nil", *status.RH)
				}
			}
		})
	}
}

func TestHumidity_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	humidity := NewHumidity(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := humidity.GetStatus(context.Background())
		return err
	})
}

func TestHumidity_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	humidity := NewHumidity(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := humidity.GetStatus(context.Background())
		return err
	})
}

func TestHumidityConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config HumidityConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: HumidityConfig{
				ID:        0,
				Name:      ptr("Test Sensor"),
				ReportThr: ptrFloat(5.0),
				Offset:    ptrFloat(-2.0),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["id"].(float64) != 0 {
					t.Errorf("id = %v, want 0", data["id"])
				}
				if data["name"].(string) != "Test Sensor" {
					t.Errorf("name = %v, want Test Sensor", data["name"])
				}
				if data["report_thr"].(float64) != 5.0 {
					t.Errorf("report_thr = %v, want 5.0", data["report_thr"])
				}
				if data["offset"].(float64) != -2.0 {
					t.Errorf("offset = %v, want -2.0", data["offset"])
				}
			},
		},
		{
			name: "minimal config",
			config: HumidityConfig{
				ID: 1,
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["name"]; ok {
					t.Error("name should not be present")
				}
				if _, ok := data["report_thr"]; ok {
					t.Error("report_thr should not be present")
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

func TestHumidity_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"id": 0, "rh": 55.5}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	humidity := NewHumidity(client, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := humidity.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
