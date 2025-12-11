package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

func TestNewCloud(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	cloud := NewCloud(client)

	if cloud == nil {
		t.Fatal("NewCloud returned nil")
	}

	if cloud.Type() != "cloud" {
		t.Errorf("Type() = %q, want %q", cloud.Type(), "cloud")
	}

	if cloud.Key() != "cloud" {
		t.Errorf("Key() = %q, want %q", cloud.Key(), "cloud")
	}

	if cloud.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestCloud_GetConfig(t *testing.T) {
	tests := []struct {
		wantEnable *bool
		wantServer *string
		name       string
		result     string
	}{
		{
			name:       "enabled with default server",
			result:     `{"enable": true}`,
			wantEnable: ptr(true),
		},
		{
			name:       "disabled",
			result:     `{"enable": false}`,
			wantEnable: ptr(false),
		},
		{
			name:       "enabled with custom server",
			result:     `{"enable": true, "server": "custom.shelly.cloud"}`,
			wantEnable: ptr(true),
			wantServer: ptr("custom.shelly.cloud"),
		},
		{
			name:   "empty config",
			result: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Cloud.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			cloud := NewCloud(client)

			config, err := cloud.GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if config == nil {
				t.Fatal("GetConfig() returned nil config")
			}

			if tt.wantEnable != nil {
				if config.Enable == nil || *config.Enable != *tt.wantEnable {
					t.Errorf("config.Enable = %v, want %v", config.Enable, *tt.wantEnable)
				}
			}

			if tt.wantServer != nil {
				if config.Server == nil || *config.Server != *tt.wantServer {
					t.Errorf("config.Server = %v, want %v", config.Server, *tt.wantServer)
				}
			}
		})
	}
}

func TestCloud_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	cloud := NewCloud(client)
	testComponentError(t, "GetConfig", func() error {
		_, err := cloud.GetConfig(context.Background())
		return err
	})
}

func TestCloud_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	cloud := NewCloud(client)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := cloud.GetConfig(context.Background())
		return err
	})
}

func TestCloud_SetConfig(t *testing.T) {
	tests := []struct {
		config *CloudConfig
		name   string
	}{
		{
			name: "enable cloud",
			config: &CloudConfig{
				Enable: ptr(true),
			},
		},
		{
			name: "disable cloud",
			config: &CloudConfig{
				Enable: ptr(false),
			},
		},
		{
			name: "set custom server",
			config: &CloudConfig{
				Enable: ptr(true),
				Server: ptr("custom.shelly.cloud"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Cloud.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Cloud.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			cloud := NewCloud(client)

			err := cloud.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestCloud_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	cloud := NewCloud(client)
	testComponentError(t, "SetConfig", func() error {
		return cloud.SetConfig(context.Background(), &CloudConfig{})
	})
}

func TestCloud_GetStatus(t *testing.T) {
	tests := []struct {
		name          string
		result        string
		wantConnected bool
	}{
		{
			name:          "connected",
			result:        `{"connected": true}`,
			wantConnected: true,
		},
		{
			name:          "disconnected",
			result:        `{"connected": false}`,
			wantConnected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Cloud.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			cloud := NewCloud(client)

			status, err := cloud.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if status.Connected != tt.wantConnected {
				t.Errorf("status.Connected = %v, want %v", status.Connected, tt.wantConnected)
			}
		})
	}
}

func TestCloud_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	cloud := NewCloud(client)
	testComponentError(t, "GetStatus", func() error {
		_, err := cloud.GetStatus(context.Background())
		return err
	})
}

func TestCloud_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	cloud := NewCloud(client)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := cloud.GetStatus(context.Background())
		return err
	})
}

func TestCloudConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config CloudConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: CloudConfig{
				Enable: ptr(true),
				Server: ptr("shelly.cloud"),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["enable"].(bool) != true {
					t.Errorf("enable = %v, want true", data["enable"])
				}
				if data["server"].(string) != "shelly.cloud" {
					t.Errorf("server = %v, want shelly.cloud", data["server"])
				}
			},
		},
		{
			name: "enable only",
			config: CloudConfig{
				Enable: ptr(false),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["enable"].(bool) != false {
					t.Errorf("enable = %v, want false", data["enable"])
				}
				if _, ok := data["server"]; ok {
					t.Error("server should not be present")
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

func TestCloud_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"connected": true}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	cloud := NewCloud(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := cloud.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
