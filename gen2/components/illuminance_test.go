package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewIlluminance(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	illum := NewIlluminance(client, 0)

	if illum == nil {
		t.Fatal("NewIlluminance returned nil")
	}

	if illum.Type() != "illuminance" {
		t.Errorf("Type() = %q, want %q", illum.Type(), "illuminance")
	}

	if illum.Key() != "illuminance" {
		t.Errorf("Key() = %q, want %q", illum.Key(), "illuminance")
	}

	if illum.Client() != client {
		t.Error("Client() did not return the expected client")
	}

	if illum.ID() != 0 {
		t.Errorf("ID() = %d, want 0", illum.ID())
	}
}

func TestIlluminance_GetConfig(t *testing.T) {
	tests := []struct {
		wantName      *string
		wantDarkThr   *int
		wantBrightThr *int
		name          string
		result        string
		id            int
	}{
		{
			name:          "full config",
			id:            0,
			result:        `{"id": 0, "name": "Motion Sensor Light", "dark_thr": 50, "bright_thr": 500}`,
			wantName:      ptr("Motion Sensor Light"),
			wantDarkThr:   ptr(50),
			wantBrightThr: ptr(500),
		},
		{
			name:   "minimal config",
			id:     0,
			result: `{"id": 0}`,
		},
		{
			name:          "with thresholds only",
			id:            1,
			result:        `{"id": 1, "dark_thr": 100, "bright_thr": 1000}`,
			wantDarkThr:   ptr(100),
			wantBrightThr: ptr(1000),
		},
		{
			name:     "with name only",
			id:       0,
			result:   `{"id": 0, "name": "Light Sensor"}`,
			wantName: ptr("Light Sensor"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Illuminance.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			illum := NewIlluminance(client, tt.id)

			config, err := illum.GetConfig(context.Background())
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

			if tt.wantDarkThr != nil {
				if config.DarkThr == nil || *config.DarkThr != *tt.wantDarkThr {
					t.Errorf("config.DarkThr = %v, want %v", config.DarkThr, *tt.wantDarkThr)
				}
			}

			if tt.wantBrightThr != nil {
				if config.BrightThr == nil || *config.BrightThr != *tt.wantBrightThr {
					t.Errorf("config.BrightThr = %v, want %v", config.BrightThr, *tt.wantBrightThr)
				}
			}
		})
	}
}

func TestIlluminance_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	illum := NewIlluminance(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := illum.GetConfig(context.Background())
		return err
	})
}

func TestIlluminance_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	illum := NewIlluminance(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := illum.GetConfig(context.Background())
		return err
	})
}

func TestIlluminance_SetConfig(t *testing.T) {
	tests := []struct {
		config *IlluminanceConfig
		name   string
		id     int
	}{
		{
			name: "set name",
			id:   0,
			config: &IlluminanceConfig{
				Name: ptr("Light Sensor"),
			},
		},
		{
			name: "set dark threshold",
			id:   0,
			config: &IlluminanceConfig{
				DarkThr: ptr(75),
			},
		},
		{
			name: "set bright threshold",
			id:   1,
			config: &IlluminanceConfig{
				BrightThr: ptr(750),
			},
		},
		{
			name: "set both thresholds",
			id:   0,
			config: &IlluminanceConfig{
				DarkThr:   ptr(50),
				BrightThr: ptr(500),
			},
		},
		{
			name: "set all fields",
			id:   0,
			config: &IlluminanceConfig{
				Name:      ptr("Motion Sensor"),
				DarkThr:   ptr(100),
				BrightThr: ptr(1000),
			},
		},
		{
			name:   "empty config",
			id:     0,
			config: &IlluminanceConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Illuminance.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Illuminance.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			illum := NewIlluminance(client, tt.id)

			err := illum.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestIlluminance_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	illum := NewIlluminance(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return illum.SetConfig(context.Background(), &IlluminanceConfig{})
	})
}

func TestIlluminance_GetStatus(t *testing.T) {
	tests := []struct {
		wantLux          *int
		wantIllumination *IlluminationLevel
		name             string
		result           string
		wantErrors       []string
		id               int
	}{
		{
			name:             "normal reading - dark",
			id:               0,
			result:           `{"id": 0, "lux": 25, "illumination": "dark"}`,
			wantLux:          ptr(25),
			wantIllumination: func() *IlluminationLevel { l := IlluminationDark; return &l }(),
		},
		{
			name:             "normal reading - twilight",
			id:               0,
			result:           `{"id": 0, "lux": 200, "illumination": "twilight"}`,
			wantLux:          ptr(200),
			wantIllumination: func() *IlluminationLevel { l := IlluminationTwilight; return &l }(),
		},
		{
			name:             "normal reading - bright",
			id:               0,
			result:           `{"id": 0, "lux": 1000, "illumination": "bright"}`,
			wantLux:          ptr(1000),
			wantIllumination: func() *IlluminationLevel { l := IlluminationBright; return &l }(),
		},
		{
			name:             "null values - sensor error",
			id:               0,
			result:           `{"id": 0, "lux": null, "illumination": null}`,
			wantLux:          nil,
			wantIllumination: nil,
		},
		{
			name:             "with errors",
			id:               0,
			result:           `{"id": 0, "lux": null, "illumination": null, "errors": ["read"]}`,
			wantLux:          nil,
			wantIllumination: nil,
			wantErrors:       []string{"read"},
		},
		{
			name:             "out of range error",
			id:               0,
			result:           `{"id": 0, "lux": null, "illumination": null, "errors": ["out_of_range"]}`,
			wantLux:          nil,
			wantIllumination: nil,
			wantErrors:       []string{"out_of_range"},
		},
		{
			name:             "different ID",
			id:               1,
			result:           `{"id": 1, "lux": 500, "illumination": "twilight"}`,
			wantLux:          ptr(500),
			wantIllumination: func() *IlluminationLevel { l := IlluminationTwilight; return &l }(),
		},
		{
			name:             "zero lux",
			id:               0,
			result:           `{"id": 0, "lux": 0, "illumination": "dark"}`,
			wantLux:          ptr(0),
			wantIllumination: func() *IlluminationLevel { l := IlluminationDark; return &l }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Illuminance.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			illum := NewIlluminance(client, tt.id)

			status, err := illum.GetStatus(context.Background())
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

			if tt.wantLux != nil {
				if status.Lux == nil {
					t.Errorf("status.Lux = nil, want %v", *tt.wantLux)
				} else if *status.Lux != *tt.wantLux {
					t.Errorf("status.Lux = %v, want %v", *status.Lux, *tt.wantLux)
				}
			} else {
				if status.Lux != nil {
					t.Errorf("status.Lux = %v, want nil", *status.Lux)
				}
			}

			if tt.wantIllumination != nil {
				if status.Illumination == nil {
					t.Errorf("status.Illumination = nil, want %v", *tt.wantIllumination)
				} else if *status.Illumination != *tt.wantIllumination {
					t.Errorf("status.Illumination = %v, want %v", *status.Illumination, *tt.wantIllumination)
				}
			} else {
				if status.Illumination != nil {
					t.Errorf("status.Illumination = %v, want nil", *status.Illumination)
				}
			}

			if len(tt.wantErrors) > 0 {
				if len(status.Errors) != len(tt.wantErrors) {
					t.Errorf("status.Errors = %v, want %v", status.Errors, tt.wantErrors)
				}
				for i, err := range tt.wantErrors {
					if i < len(status.Errors) && status.Errors[i] != err {
						t.Errorf("status.Errors[%d] = %q, want %q", i, status.Errors[i], err)
					}
				}
			}
		})
	}
}

func TestIlluminance_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	illum := NewIlluminance(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := illum.GetStatus(context.Background())
		return err
	})
}

func TestIlluminance_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	illum := NewIlluminance(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := illum.GetStatus(context.Background())
		return err
	})
}

func TestIlluminanceConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config IlluminanceConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: IlluminanceConfig{
				ID:        0,
				Name:      ptr("Test Sensor"),
				DarkThr:   ptr(50),
				BrightThr: ptr(500),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["id"].(float64) != 0 {
					t.Errorf("id = %v, want 0", data["id"])
				}
				if data["name"].(string) != "Test Sensor" {
					t.Errorf("name = %v, want Test Sensor", data["name"])
				}
				if data["dark_thr"].(float64) != 50 {
					t.Errorf("dark_thr = %v, want 50", data["dark_thr"])
				}
				if data["bright_thr"].(float64) != 500 {
					t.Errorf("bright_thr = %v, want 500", data["bright_thr"])
				}
			},
		},
		{
			name: "minimal config",
			config: IlluminanceConfig{
				ID: 1,
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["name"]; ok {
					t.Error("name should not be present")
				}
				if _, ok := data["dark_thr"]; ok {
					t.Error("dark_thr should not be present")
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

func TestIlluminanceStatus_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		wantLux          *int
		wantIllumination *IlluminationLevel
		name             string
		json             string
		wantErrors       []string
		wantID           int
	}{
		{
			name:             "normal reading",
			json:             `{"id":0,"lux":250,"illumination":"twilight"}`,
			wantID:           0,
			wantLux:          ptr(250),
			wantIllumination: func() *IlluminationLevel { l := IlluminationTwilight; return &l }(),
		},
		{
			name:             "with errors",
			json:             `{"id":0,"lux":null,"illumination":null,"errors":["read","out_of_range"]}`,
			wantID:           0,
			wantLux:          nil,
			wantIllumination: nil,
			wantErrors:       []string{"read", "out_of_range"},
		},
		{
			name:             "with unknown fields",
			json:             `{"id":0,"lux":100,"illumination":"dark","future_field":"value"}`,
			wantID:           0,
			wantLux:          ptr(100),
			wantIllumination: func() *IlluminationLevel { l := IlluminationDark; return &l }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status IlluminanceStatus
			if err := json.Unmarshal([]byte(tt.json), &status); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if status.ID != tt.wantID {
				t.Errorf("ID = %d, want %d", status.ID, tt.wantID)
			}
			if tt.wantLux != nil {
				if status.Lux == nil || *status.Lux != *tt.wantLux {
					t.Errorf("Lux = %v, want %v", status.Lux, *tt.wantLux)
				}
			}
			if tt.wantIllumination != nil {
				if status.Illumination == nil || *status.Illumination != *tt.wantIllumination {
					t.Errorf("Illumination = %v, want %v", status.Illumination, *tt.wantIllumination)
				}
			}
			if len(tt.wantErrors) > 0 && len(status.Errors) != len(tt.wantErrors) {
				t.Errorf("Errors = %v, want %v", status.Errors, tt.wantErrors)
			}
		})
	}
}

func TestIlluminance_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"id": 0, "lux": 100, "illumination": "dark"}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	illum := NewIlluminance(client, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := illum.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestIlluminationLevel_Constants(t *testing.T) {
	// Verify constant values match expected strings
	if IlluminationDark != "dark" {
		t.Errorf("IlluminationDark = %q, want %q", IlluminationDark, "dark")
	}
	if IlluminationTwilight != "twilight" {
		t.Errorf("IlluminationTwilight = %q, want %q", IlluminationTwilight, "twilight")
	}
	if IlluminationBright != "bright" {
		t.Errorf("IlluminationBright = %q, want %q", IlluminationBright, "bright")
	}
}
