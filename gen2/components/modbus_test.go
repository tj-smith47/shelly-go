package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

func TestNewModbus(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	modbus := NewModbus(client)

	if modbus == nil {
		t.Fatal("NewModbus returned nil")
	}

	if modbus.Type() != "modbus" {
		t.Errorf("Type() = %q, want %q", modbus.Type(), "modbus")
	}

	if modbus.Key() != "modbus" {
		t.Errorf("Key() = %q, want %q", modbus.Key(), "modbus")
	}

	if modbus.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestModbus_GetConfig(t *testing.T) {
	tests := []struct {
		name       string
		result     string
		wantEnable bool
	}{
		{
			name:       "enabled",
			result:     `{"enable": true}`,
			wantEnable: true,
		},
		{
			name:       "disabled",
			result:     `{"enable": false}`,
			wantEnable: false,
		},
		{
			name:       "with unknown fields",
			result:     `{"enable": true, "future_field": "value"}`,
			wantEnable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Modbus.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			modbus := NewModbus(client)

			config, err := modbus.GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if config == nil {
				t.Fatal("GetConfig() returned nil config")
			}

			if config.Enable != tt.wantEnable {
				t.Errorf("config.Enable = %v, want %v", config.Enable, tt.wantEnable)
			}
		})
	}
}

func TestModbus_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	modbus := NewModbus(client)
	testComponentError(t, "GetConfig", func() error {
		_, err := modbus.GetConfig(context.Background())
		return err
	})
}

func TestModbus_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	modbus := NewModbus(client)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := modbus.GetConfig(context.Background())
		return err
	})
}

func TestModbus_SetConfig(t *testing.T) {
	tests := []struct {
		config *ModbusConfig
		name   string
	}{
		{
			name: "enable modbus",
			config: &ModbusConfig{
				Enable: true,
			},
		},
		{
			name: "disable modbus",
			config: &ModbusConfig{
				Enable: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Modbus.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Modbus.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			modbus := NewModbus(client)

			err := modbus.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestModbus_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	modbus := NewModbus(client)
	testComponentError(t, "SetConfig", func() error {
		return modbus.SetConfig(context.Background(), &ModbusConfig{Enable: true})
	})
}

func TestModbus_GetStatus(t *testing.T) {
	tests := []struct {
		name        string
		result      string
		wantEnabled bool
	}{
		{
			name:        "enabled",
			result:      `{"enabled": true}`,
			wantEnabled: true,
		},
		{
			name:        "disabled",
			result:      `{"enabled": false}`,
			wantEnabled: false,
		},
		{
			name:        "with unknown fields",
			result:      `{"enabled": true, "connections": 5}`,
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Modbus.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			modbus := NewModbus(client)

			status, err := modbus.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if status.Enabled != tt.wantEnabled {
				t.Errorf("status.Enabled = %v, want %v", status.Enabled, tt.wantEnabled)
			}
		})
	}
}

func TestModbus_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	modbus := NewModbus(client)
	testComponentError(t, "GetStatus", func() error {
		_, err := modbus.GetStatus(context.Background())
		return err
	})
}

func TestModbus_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	modbus := NewModbus(client)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := modbus.GetStatus(context.Background())
		return err
	})
}

func TestModbusConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config ModbusConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "enabled",
			config: ModbusConfig{
				Enable: true,
			},
			check: func(t *testing.T, data map[string]any) {
				if data["enable"].(bool) != true {
					t.Errorf("enable = %v, want true", data["enable"])
				}
			},
		},
		{
			name: "disabled",
			config: ModbusConfig{
				Enable: false,
			},
			check: func(t *testing.T, data map[string]any) {
				if data["enable"].(bool) != false {
					t.Errorf("enable = %v, want false", data["enable"])
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

func TestModbusStatus_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantEnabled bool
	}{
		{
			name:        "enabled",
			json:        `{"enabled":true}`,
			wantEnabled: true,
		},
		{
			name:        "disabled",
			json:        `{"enabled":false}`,
			wantEnabled: false,
		},
		{
			name:        "with unknown fields",
			json:        `{"enabled":true,"future_field":"value"}`,
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status ModbusStatus
			if err := json.Unmarshal([]byte(tt.json), &status); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if status.Enabled != tt.wantEnabled {
				t.Errorf("Enabled = %v, want %v", status.Enabled, tt.wantEnabled)
			}
		})
	}
}

func TestModbus_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"enabled": true}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	modbus := NewModbus(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := modbus.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
