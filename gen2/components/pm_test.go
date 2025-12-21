package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewPM(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	pm := NewPM(client, 0)

	if pm == nil {
		t.Fatal("NewPM returned nil")
	}

	if pm.Type() != "pm" {
		t.Errorf("Type() = %q, want %q", pm.Type(), "pm")
	}

	if pm.ID() != 0 {
		t.Errorf("ID() = %d, want %d", pm.ID(), 0)
	}

	if pm.Key() != "pm:0" {
		t.Errorf("Key() = %q, want %q", pm.Key(), "pm:0")
	}
}

func TestPM_GetConfig(t *testing.T) {
	tests := []struct {
		wantName    *string
		wantReverse *bool
		name        string
		result      string
		wantID      int
	}{
		{
			name:        "basic config",
			result:      `{"id": 0, "name": "Power Meter", "reverse": false}`,
			wantID:      0,
			wantName:    ptr("Power Meter"),
			wantReverse: ptr(false),
		},
		{
			name:   "minimal config",
			result: `{"id": 0}`,
			wantID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "PM.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			pm := NewPM(client, 0)

			config, err := pm.GetConfig(context.Background())
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

			if tt.wantReverse != nil {
				if config.Reverse == nil {
					t.Error("config.Reverse is nil, want non-nil")
				} else if *config.Reverse != *tt.wantReverse {
					t.Errorf("config.Reverse = %v, want %v", *config.Reverse, *tt.wantReverse)
				}
			}
		})
	}
}

func TestPM_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	pm := NewPM(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := pm.GetConfig(context.Background())
		return err
	})
}

func TestPM_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	pm := NewPM(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := pm.GetConfig(context.Background())
		return err
	})
}

func TestPM_SetConfig(t *testing.T) {
	config := &PMConfig{
		ID:      0,
		Name:    ptr("Main Meter"),
		Reverse: ptr(false),
	}

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "PM.SetConfig" {
				t.Errorf("method = %q, want %q", method, "PM.SetConfig")
			}
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	pm := NewPM(client, 0)

	err := pm.SetConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestPM_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	pm := NewPM(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return pm.SetConfig(context.Background(), &PMConfig{ID: 0})
	})
}

func TestPM_GetStatus(t *testing.T) {
	tests := []struct {
		wantFreq         *float64
		wantAEnergyTotal *float64
		name             string
		result           string
		wantID           int
		wantVoltage      float64
		wantCurrent      float64
		wantAPower       float64
	}{
		{
			name: "normal operation",
			result: `{
				"id": 0,
				"voltage": 230.5,
				"current": 1.5,
				"apower": 345.75,
				"freq": 50.0,
				"aenergy": {
					"total": 1234.56
				}
			}`,
			wantID:           0,
			wantVoltage:      230.5,
			wantCurrent:      1.5,
			wantAPower:       345.75,
			wantFreq:         ptr(50.0),
			wantAEnergyTotal: ptr(1234.56),
		},
		{
			name: "low power",
			result: `{
				"id": 0,
				"voltage": 230.0,
				"current": 0.1,
				"apower": 23.0,
				"aenergy": {
					"total": 50.0
				}
			}`,
			wantID:           0,
			wantVoltage:      230.0,
			wantCurrent:      0.1,
			wantAPower:       23.0,
			wantAEnergyTotal: ptr(50.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "PM.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			pm := NewPM(client, 0)

			status, err := pm.GetStatus(context.Background())
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
				t.Errorf("status.Voltage = %.2f, want %.2f", status.Voltage, tt.wantVoltage)
			}

			if status.Current != tt.wantCurrent {
				t.Errorf("status.Current = %.2f, want %.2f", status.Current, tt.wantCurrent)
			}

			if status.APower != tt.wantAPower {
				t.Errorf("status.APower = %.2f, want %.2f", status.APower, tt.wantAPower)
			}

			if tt.wantFreq != nil {
				if status.Freq == nil {
					t.Error("status.Freq is nil, want non-nil")
				} else if *status.Freq != *tt.wantFreq {
					t.Errorf("status.Freq = %.2f, want %.2f", *status.Freq, *tt.wantFreq)
				}
			}

			if tt.wantAEnergyTotal != nil {
				if status.AEnergy == nil {
					t.Error("status.AEnergy is nil, want non-nil")
				} else if status.AEnergy.Total != *tt.wantAEnergyTotal {
					t.Errorf("status.AEnergy.Total = %.2f, want %.2f", status.AEnergy.Total, *tt.wantAEnergyTotal)
				}
			}
		})
	}
}

func TestPM_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	pm := NewPM(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := pm.GetStatus(context.Background())
		return err
	})
}

func TestPM_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	pm := NewPM(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := pm.GetStatus(context.Background())
		return err
	})
}

func TestPM_ResetCounters(t *testing.T) {
	tests := []struct {
		name         string
		counterTypes []string
	}{
		{
			name:         "reset all counters",
			counterTypes: nil,
		},
		{
			name:         "reset specific counter",
			counterTypes: []string{"aenergy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "PM.ResetCounters" {
						t.Errorf("method = %q, want %q", method, "PM.ResetCounters")
					}
					return jsonrpcResponse(`{}`)
				},
			}
			client := rpc.NewClient(tr)
			pm := NewPM(client, 0)

			err := pm.ResetCounters(context.Background(), tt.counterTypes)
			if err != nil {
				t.Errorf("ResetCounters() error = %v", err)
			}
		})
	}
}

func TestPM_ResetCounters_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	pm := NewPM(client, 0)
	testComponentError(t, "ResetCounters", func() error {
		return pm.ResetCounters(context.Background(), nil)
	})
}
