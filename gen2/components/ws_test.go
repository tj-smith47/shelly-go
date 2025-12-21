package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewWs(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	ws := NewWs(client)

	if ws == nil {
		t.Fatal("NewWs returned nil")
	}

	if ws.Type() != "ws" {
		t.Errorf("Type() = %q, want %q", ws.Type(), "ws")
	}

	if ws.Key() != "ws" {
		t.Errorf("Key() = %q, want %q", ws.Key(), "ws")
	}

	if ws.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestWs_GetConfig(t *testing.T) {
	tests := []struct {
		wantEnable *bool
		wantServer *string
		wantSSLCA  *string
		name       string
		result     string
	}{
		{
			name:       "basic config",
			result:     `{"enable": true, "server": "ws://example.com:8080/shelly"}`,
			wantEnable: ptr(true),
			wantServer: ptr("ws://example.com:8080/shelly"),
		},
		{
			name:       "disabled",
			result:     `{"enable": false}`,
			wantEnable: ptr(false),
		},
		{
			name:       "tls with ca",
			result:     `{"enable": true, "server": "wss://secure.example.com/shelly", "ssl_ca": "ca.pem"}`,
			wantEnable: ptr(true),
			wantServer: ptr("wss://secure.example.com/shelly"),
			wantSSLCA:  ptr("ca.pem"),
		},
		{
			name:      "tls no verify",
			result:    `{"enable": true, "server": "wss://example.com/shelly", "ssl_ca": "*"}`,
			wantSSLCA: ptr("*"),
		},
		{
			name:   "empty config",
			result: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Ws.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			ws := NewWs(client)

			config, err := ws.GetConfig(context.Background())
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

			if tt.wantSSLCA != nil {
				if config.SSLCA == nil || *config.SSLCA != *tt.wantSSLCA {
					t.Errorf("config.SSLCA = %v, want %v", config.SSLCA, *tt.wantSSLCA)
				}
			}
		})
	}
}

func TestWs_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	ws := NewWs(client)
	testComponentError(t, "GetConfig", func() error {
		_, err := ws.GetConfig(context.Background())
		return err
	})
}

func TestWs_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	ws := NewWs(client)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := ws.GetConfig(context.Background())
		return err
	})
}

func TestWs_SetConfig(t *testing.T) {
	tests := []struct {
		config *WsConfig
		name   string
	}{
		{
			name: "enable ws",
			config: &WsConfig{
				Enable: ptr(true),
				Server: ptr("ws://example.com:8080/shelly"),
			},
		},
		{
			name: "disable ws",
			config: &WsConfig{
				Enable: ptr(false),
			},
		},
		{
			name: "tls with default ca",
			config: &WsConfig{
				Enable: ptr(true),
				Server: ptr("wss://secure.example.com/shelly"),
				SSLCA:  ptr("ca.pem"),
			},
		},
		{
			name: "tls with user ca",
			config: &WsConfig{
				Enable: ptr(true),
				Server: ptr("wss://secure.example.com/shelly"),
				SSLCA:  ptr("user_ca.pem"),
			},
		},
		{
			name: "tls without verification",
			config: &WsConfig{
				Enable: ptr(true),
				Server: ptr("wss://example.com/shelly"),
				SSLCA:  ptr("*"),
			},
		},
		{
			name: "change server only",
			config: &WsConfig{
				Server: ptr("ws://new-server.example.com/shelly"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Ws.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Ws.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			ws := NewWs(client)

			err := ws.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestWs_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	ws := NewWs(client)
	testComponentError(t, "SetConfig", func() error {
		return ws.SetConfig(context.Background(), &WsConfig{})
	})
}

func TestWs_GetStatus(t *testing.T) {
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
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Ws.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			ws := NewWs(client)

			status, err := ws.GetStatus(context.Background())
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

func TestWs_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	ws := NewWs(client)
	testComponentError(t, "GetStatus", func() error {
		_, err := ws.GetStatus(context.Background())
		return err
	})
}

func TestWs_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	ws := NewWs(client)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := ws.GetStatus(context.Background())
		return err
	})
}

func TestWsConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config WsConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: WsConfig{
				Enable: ptr(true),
				Server: ptr("wss://example.com/shelly"),
				SSLCA:  ptr("ca.pem"),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["enable"].(bool) != true {
					t.Errorf("enable = %v, want true", data["enable"])
				}
				if data["server"].(string) != "wss://example.com/shelly" {
					t.Errorf("server = %v, want wss://example.com/shelly", data["server"])
				}
				if data["ssl_ca"].(string) != "ca.pem" {
					t.Errorf("ssl_ca = %v, want ca.pem", data["ssl_ca"])
				}
			},
		},
		{
			name: "minimal config",
			config: WsConfig{
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

func TestWs_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"connected": true}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	ws := NewWs(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ws.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
