package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewDevicePower(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	dp := NewDevicePower(client, 0)

	if dp == nil {
		t.Fatal("NewDevicePower returned nil")
	}

	if dp.Type() != "devicepower" {
		t.Errorf("Type() = %q, want %q", dp.Type(), "devicepower")
	}

	if dp.ID() != 0 {
		t.Errorf("ID() = %d, want %d", dp.ID(), 0)
	}

	if dp.Key() != "devicepower:0" {
		t.Errorf("Key() = %q, want %q", dp.Key(), "devicepower:0")
	}
}

func TestDevicePower_GetConfig(t *testing.T) {
	tests := []struct {
		name       string
		result     string
		errMessage string
		wantID     int
		wantErr    bool
	}{
		{
			name:   "empty config (typical)",
			result: `{"id": 0}`,
			wantID: 0,
		},
		{
			name:   "config with extra fields",
			result: `{"id": 0, "unknown_field": "value"}`,
			wantID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "DevicePower.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}

					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			dp := NewDevicePower(client, 0)

			config, err := dp.GetConfig(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if config == nil {
					t.Fatal("GetConfig() returned nil config")
				}

				if config.ID != tt.wantID {
					t.Errorf("config.ID = %d, want %d", config.ID, tt.wantID)
				}
			}
		})
	}
}

func TestDevicePower_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	dp := NewDevicePower(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := dp.GetConfig(context.Background())
		return err
	})
}

func TestDevicePower_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	dp := NewDevicePower(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := dp.GetConfig(context.Background())
		return err
	})
}

func TestDevicePower_SetConfig(t *testing.T) {
	config := &DevicePowerConfig{
		ID: 0,
	}

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "DevicePower.SetConfig" {
				t.Errorf("method = %q, want %q", method, "DevicePower.SetConfig")
			}
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	dp := NewDevicePower(client, 0)

	err := dp.SetConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestDevicePower_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	dp := NewDevicePower(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return dp.SetConfig(context.Background(), &DevicePowerConfig{ID: 0})
	})
}

func TestDevicePower_GetStatus(t *testing.T) {
	tests := []struct {
		name             string
		result           string
		wantID           int
		wantBatteryV     float64
		wantBatteryPct   int
		wantExternalPres bool
		wantErr          bool
	}{
		{
			name: "battery at 100%",
			result: `{
				"id": 0,
				"battery": {
					"V": 4.8,
					"percent": 100
				},
				"external": {
					"present": false
				}
			}`,
			wantID:           0,
			wantBatteryV:     4.8,
			wantBatteryPct:   100,
			wantExternalPres: false,
		},
		{
			name: "battery at 50%",
			result: `{
				"id": 0,
				"battery": {
					"V": 4.2,
					"percent": 50
				},
				"external": {
					"present": false
				}
			}`,
			wantID:           0,
			wantBatteryV:     4.2,
			wantBatteryPct:   50,
			wantExternalPres: false,
		},
		{
			name: "low battery at 11%",
			result: `{
				"id": 0,
				"battery": {
					"V": 4.59,
					"percent": 11
				},
				"external": {
					"present": false
				}
			}`,
			wantID:           0,
			wantBatteryV:     4.59,
			wantBatteryPct:   11,
			wantExternalPres: false,
		},
		{
			name: "charging with external power",
			result: `{
				"id": 0,
				"battery": {
					"V": 4.95,
					"percent": 85
				},
				"external": {
					"present": true
				}
			}`,
			wantID:           0,
			wantBatteryV:     4.95,
			wantBatteryPct:   85,
			wantExternalPres: true,
		},
		{
			name: "fully charged on external power",
			result: `{
				"id": 0,
				"battery": {
					"V": 5.0,
					"percent": 100
				},
				"external": {
					"present": true
				}
			}`,
			wantID:           0,
			wantBatteryV:     5.0,
			wantBatteryPct:   100,
			wantExternalPres: true,
		},
		{
			name: "critically low battery",
			result: `{
				"id": 0,
				"battery": {
					"V": 3.2,
					"percent": 0
				},
				"external": {
					"present": false
				}
			}`,
			wantID:           0,
			wantBatteryV:     3.2,
			wantBatteryPct:   0,
			wantExternalPres: false,
		},
		{
			name: "with extra fields",
			result: `{
				"id": 0,
				"battery": {
					"V": 4.5,
					"percent": 75,
					"unknown_field": "value"
				},
				"external": {
					"present": false,
					"future_field": 123
				},
				"extra": "data"
			}`,
			wantID:           0,
			wantBatteryV:     4.5,
			wantBatteryPct:   75,
			wantExternalPres: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "DevicePower.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}

					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			dp := NewDevicePower(client, 0)

			status, err := dp.GetStatus(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if status == nil {
					t.Fatal("GetStatus() returned nil status")
				}

				if status.ID != tt.wantID {
					t.Errorf("status.ID = %d, want %d", status.ID, tt.wantID)
				}

				if status.Battery.V != tt.wantBatteryV {
					t.Errorf("status.Battery.V = %.2f, want %.2f", status.Battery.V, tt.wantBatteryV)
				}

				if status.Battery.Percent != tt.wantBatteryPct {
					t.Errorf("status.Battery.Percent = %d, want %d", status.Battery.Percent, tt.wantBatteryPct)
				}

				if status.External.Present != tt.wantExternalPres {
					t.Errorf("status.External.Present = %v, want %v", status.External.Present, tt.wantExternalPres)
				}
			}
		})
	}
}

func TestDevicePower_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	dp := NewDevicePower(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := dp.GetStatus(context.Background())
		return err
	})
}

func TestDevicePower_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	dp := NewDevicePower(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := dp.GetStatus(context.Background())
		return err
	})
}

// TestDevicePower_MultipleBatteryScenarios tests realistic battery monitoring scenarios
func TestDevicePower_MultipleBatteryScenarios(t *testing.T) {
	scenarios := []struct {
		status   *DevicePowerStatus
		name     string
		checkLow bool
	}{
		{
			name: "good battery",
			status: &DevicePowerStatus{
				ID: 0,
				Battery: BatteryStatus{
					V:       4.5,
					Percent: 80,
				},
				External: ExternalPowerStatus{
					Present: false,
				},
			},
			checkLow: false,
		},
		{
			name: "low battery warning",
			status: &DevicePowerStatus{
				ID: 0,
				Battery: BatteryStatus{
					V:       4.0,
					Percent: 15,
				},
				External: ExternalPowerStatus{
					Present: false,
				},
			},
			checkLow: true,
		},
		{
			name: "critical battery",
			status: &DevicePowerStatus{
				ID: 0,
				Battery: BatteryStatus{
					V:       3.5,
					Percent: 5,
				},
				External: ExternalPowerStatus{
					Present: false,
				},
			},
			checkLow: true,
		},
		{
			name: "charging",
			status: &DevicePowerStatus{
				ID: 0,
				Battery: BatteryStatus{
					V:       4.9,
					Percent: 90,
				},
				External: ExternalPowerStatus{
					Present: true,
				},
			},
			checkLow: false,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			// Simulate checking battery level
			if sc.checkLow && sc.status.Battery.Percent > 20 {
				t.Errorf("Expected low battery warning but percent is %d", sc.status.Battery.Percent)
			}

			// Simulate checking if charging
			if sc.status.External.Present && sc.status.Battery.Percent < 100 {
				// Device is charging
				if sc.status.Battery.V < 3.0 {
					t.Error("Battery voltage too low even while charging")
				}
			}
		})
	}
}
