package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

func TestNewPM1(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	pm1 := NewPM1(client, 0)

	if pm1 == nil {
		t.Fatal("NewPM1 returned nil")
	}

	if pm1.Type() != "pm1" {
		t.Errorf("Type() = %q, want %q", pm1.Type(), "pm1")
	}

	if pm1.ID() != 0 {
		t.Errorf("ID() = %d, want %d", pm1.ID(), 0)
	}

	if pm1.Key() != "pm1:0" {
		t.Errorf("Key() = %q, want %q", pm1.Key(), "pm1:0")
	}
}

func TestPM1_GetConfig(t *testing.T) {
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
			name:        "bidirectional metering",
			result:      `{"id": 0, "name": "Solar Meter", "reverse": true}`,
			wantID:      0,
			wantName:    ptr("Solar Meter"),
			wantReverse: ptr(true),
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
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "PM1.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			pm1 := NewPM1(client, 0)

			config, err := pm1.GetConfig(context.Background())
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

func TestPM1_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	pm1 := NewPM1(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := pm1.GetConfig(context.Background())
		return err
	})
}

func TestPM1_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	pm1 := NewPM1(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := pm1.GetConfig(context.Background())
		return err
	})
}

func TestPM1_SetConfig(t *testing.T) {
	config := &PM1Config{
		ID:      0,
		Name:    ptr("Main Meter"),
		Reverse: ptr(true),
	}

	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "PM1.SetConfig" {
				t.Errorf("method = %q, want %q", method, "PM1.SetConfig")
			}
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	pm1 := NewPM1(client, 0)

	err := pm1.SetConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestPM1_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	pm1 := NewPM1(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return pm1.SetConfig(context.Background(), &PM1Config{ID: 0})
	})
}

func TestPM1_GetStatus(t *testing.T) {
	tests := []struct {
		wantFreq            *float64
		wantAEnergyTotal    *float64
		wantRetAEnergyTotal *float64
		name                string
		result              string
		wantErrors          []string
		wantID              int
		wantVoltage         float64
		wantCurrent         float64
		wantAPower          float64
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
					"total": 1234.56,
					"by_minute": [10.5, 11.2, 9.8],
					"minute_ts": 1234567890
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
			name: "bidirectional with return energy",
			result: `{
				"id": 0,
				"voltage": 240.0,
				"current": -2.5,
				"apower": -600.0,
				"freq": 60.0,
				"aenergy": {
					"total": 5000.0
				},
				"ret_aenergy": {
					"total": 1500.0
				}
			}`,
			wantID:              0,
			wantVoltage:         240.0,
			wantCurrent:         -2.5,
			wantAPower:          -600.0,
			wantFreq:            ptr(60.0),
			wantAEnergyTotal:    ptr(5000.0),
			wantRetAEnergyTotal: ptr(1500.0),
		},
		{
			name: "with errors",
			result: `{
				"id": 0,
				"voltage": 250.0,
				"current": 0.5,
				"apower": 125.0,
				"errors": ["out_of_range:voltage"]
			}`,
			wantID:      0,
			wantVoltage: 250.0,
			wantCurrent: 0.5,
			wantAPower:  125.0,
			wantErrors:  []string{"out_of_range:voltage"},
		},
		{
			name: "low power consumption",
			result: `{
				"id": 0,
				"voltage": 230.0,
				"current": 0.05,
				"apower": 11.5,
				"freq": 50.0,
				"aenergy": {
					"total": 25.3
				}
			}`,
			wantID:           0,
			wantVoltage:      230.0,
			wantCurrent:      0.05,
			wantAPower:       11.5,
			wantFreq:         ptr(50.0),
			wantAEnergyTotal: ptr(25.3),
		},
		{
			name: "high power consumption",
			result: `{
				"id": 0,
				"voltage": 230.0,
				"current": 16.0,
				"apower": 3680.0,
				"freq": 50.0,
				"aenergy": {
					"total": 15000.0,
					"by_minute": [61.3, 61.5, 61.2],
					"minute_ts": 1234567890
				}
			}`,
			wantID:           0,
			wantVoltage:      230.0,
			wantCurrent:      16.0,
			wantAPower:       3680.0,
			wantFreq:         ptr(50.0),
			wantAEnergyTotal: ptr(15000.0),
		},
		{
			name: "no load",
			result: `{
				"id": 0,
				"voltage": 230.0,
				"current": 0.0,
				"apower": 0.0,
				"freq": 50.0,
				"aenergy": {
					"total": 100.0
				}
			}`,
			wantID:           0,
			wantVoltage:      230.0,
			wantCurrent:      0.0,
			wantAPower:       0.0,
			wantFreq:         ptr(50.0),
			wantAEnergyTotal: ptr(100.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "PM1.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			pm1 := NewPM1(client, 0)

			status, err := pm1.GetStatus(context.Background())
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

			if tt.wantRetAEnergyTotal != nil {
				if status.RetAEnergy == nil {
					t.Error("status.RetAEnergy is nil, want non-nil")
				} else if status.RetAEnergy.Total != *tt.wantRetAEnergyTotal {
					t.Errorf("status.RetAEnergy.Total = %.2f, want %.2f", status.RetAEnergy.Total, *tt.wantRetAEnergyTotal)
				}
			}

			if len(tt.wantErrors) > 0 {
				if len(status.Errors) != len(tt.wantErrors) {
					t.Errorf("len(status.Errors) = %d, want %d", len(status.Errors), len(tt.wantErrors))
				} else {
					for i, want := range tt.wantErrors {
						if status.Errors[i] != want {
							t.Errorf("status.Errors[%d] = %q, want %q", i, status.Errors[i], want)
						}
					}
				}
			}
		})
	}
}

func TestPM1_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	pm1 := NewPM1(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := pm1.GetStatus(context.Background())
		return err
	})
}

func TestPM1_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	pm1 := NewPM1(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := pm1.GetStatus(context.Background())
		return err
	})
}

func TestPM1_ResetCounters(t *testing.T) {
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
		{
			name:         "reset multiple counters",
			counterTypes: []string{"aenergy", "ret_aenergy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "PM1.ResetCounters" {
						t.Errorf("method = %q, want %q", method, "PM1.ResetCounters")
					}
					return jsonrpcResponse(`{}`)
				},
			}
			client := rpc.NewClient(tr)
			pm1 := NewPM1(client, 0)

			err := pm1.ResetCounters(context.Background(), tt.counterTypes)
			if err != nil {
				t.Errorf("ResetCounters() error = %v", err)
			}
		})
	}
}

func TestPM1_ResetCounters_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	pm1 := NewPM1(client, 0)
	testComponentError(t, "ResetCounters", func() error {
		return pm1.ResetCounters(context.Background(), nil)
	})
}
