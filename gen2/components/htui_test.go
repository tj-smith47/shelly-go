package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

func TestNewHTUI(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	htui := NewHTUI(client)

	if htui == nil {
		t.Fatal("NewHTUI returned nil")
	}

	if htui.Type() != "ht_ui" {
		t.Errorf("Type() = %q, want %q", htui.Type(), "ht_ui")
	}

	if htui.Key() != "ht_ui" {
		t.Errorf("Key() = %q, want %q", htui.Key(), "ht_ui")
	}

	if htui.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestHTUI_GetConfig(t *testing.T) {
	tests := []struct {
		name         string
		result       string
		wantTempUnit string
	}{
		{
			name:         "celsius",
			result:       `{"temperature_unit": "C"}`,
			wantTempUnit: "C",
		},
		{
			name:         "fahrenheit",
			result:       `{"temperature_unit": "F"}`,
			wantTempUnit: "F",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "HT_UI.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			htui := NewHTUI(client)

			config, err := htui.GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if config == nil {
				t.Fatal("GetConfig() returned nil config")
			}

			if config.TemperatureUnit != tt.wantTempUnit {
				t.Errorf("config.TemperatureUnit = %q, want %q", config.TemperatureUnit, tt.wantTempUnit)
			}
		})
	}
}

func TestHTUI_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	htui := NewHTUI(client)
	testComponentError(t, "GetConfig", func() error {
		_, err := htui.GetConfig(context.Background())
		return err
	})
}

func TestHTUI_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	htui := NewHTUI(client)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := htui.GetConfig(context.Background())
		return err
	})
}

func TestHTUI_SetConfig(t *testing.T) {
	tests := []struct {
		config *HTUIConfig
		name   string
	}{
		{
			name: "set celsius",
			config: &HTUIConfig{
				TemperatureUnit: "C",
			},
		},
		{
			name: "set fahrenheit",
			config: &HTUIConfig{
				TemperatureUnit: "F",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "HT_UI.SetConfig" {
						t.Errorf("method = %q, want %q", method, "HT_UI.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			htui := NewHTUI(client)

			err := htui.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestHTUI_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	htui := NewHTUI(client)
	testComponentError(t, "SetConfig", func() error {
		return htui.SetConfig(context.Background(), &HTUIConfig{TemperatureUnit: "C"})
	})
}

func TestHTUIConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config HTUIConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "celsius",
			config: HTUIConfig{
				TemperatureUnit: "C",
			},
			check: func(t *testing.T, data map[string]any) {
				if data["temperature_unit"].(string) != "C" {
					t.Errorf("temperature_unit = %v, want C", data["temperature_unit"])
				}
			},
		},
		{
			name: "fahrenheit",
			config: HTUIConfig{
				TemperatureUnit: "F",
			},
			check: func(t *testing.T, data map[string]any) {
				if data["temperature_unit"].(string) != "F" {
					t.Errorf("temperature_unit = %v, want F", data["temperature_unit"])
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

func TestHTUI_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"temperature_unit": "C"}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	htui := NewHTUI(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := htui.GetConfig(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
