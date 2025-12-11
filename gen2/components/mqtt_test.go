package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

func TestNewMQTT(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	mqtt := NewMQTT(client)

	if mqtt == nil {
		t.Fatal("NewMQTT returned nil")
	}

	if mqtt.Type() != "mqtt" {
		t.Errorf("Type() = %q, want %q", mqtt.Type(), "mqtt")
	}

	if mqtt.Key() != "mqtt" {
		t.Errorf("Key() = %q, want %q", mqtt.Key(), "mqtt")
	}

	if mqtt.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestMQTT_GetConfig(t *testing.T) {
	tests := []struct {
		wantEnable   *bool
		wantServer   *string
		wantClientID *string
		wantUser     *string
		wantSSLCA    *string
		name         string
		result       string
	}{
		{
			name:       "basic config",
			result:     `{"enable": true, "server": "mqtt.example.com:1883"}`,
			wantEnable: ptr(true),
			wantServer: ptr("mqtt.example.com:1883"),
		},
		{
			name:         "full config",
			result:       `{"enable": true, "server": "mqtt.example.com:8883", "client_id": "shelly-device", "user": "mqttuser", "ssl_ca": "ca.pem"}`,
			wantEnable:   ptr(true),
			wantServer:   ptr("mqtt.example.com:8883"),
			wantClientID: ptr("shelly-device"),
			wantUser:     ptr("mqttuser"),
			wantSSLCA:    ptr("ca.pem"),
		},
		{
			name:       "disabled",
			result:     `{"enable": false}`,
			wantEnable: ptr(false),
		},
		{
			name:      "tls no verify",
			result:    `{"enable": true, "server": "mqtt.example.com:8883", "ssl_ca": "*"}`,
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
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "MQTT.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			mqtt := NewMQTT(client)

			config, err := mqtt.GetConfig(context.Background())
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

			if tt.wantClientID != nil {
				if config.ClientID == nil || *config.ClientID != *tt.wantClientID {
					t.Errorf("config.ClientID = %v, want %v", config.ClientID, *tt.wantClientID)
				}
			}

			if tt.wantUser != nil {
				if config.User == nil || *config.User != *tt.wantUser {
					t.Errorf("config.User = %v, want %v", config.User, *tt.wantUser)
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

func TestMQTT_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	mqtt := NewMQTT(client)
	testComponentError(t, "GetConfig", func() error {
		_, err := mqtt.GetConfig(context.Background())
		return err
	})
}

func TestMQTT_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	mqtt := NewMQTT(client)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := mqtt.GetConfig(context.Background())
		return err
	})
}

func TestMQTT_SetConfig(t *testing.T) {
	tests := []struct {
		config *MQTTConfig
		name   string
	}{
		{
			name: "enable mqtt",
			config: &MQTTConfig{
				Enable: ptr(true),
				Server: ptr("mqtt.example.com:1883"),
			},
		},
		{
			name: "disable mqtt",
			config: &MQTTConfig{
				Enable: ptr(false),
			},
		},
		{
			name: "full config with auth",
			config: &MQTTConfig{
				Enable:   ptr(true),
				Server:   ptr("mqtt.example.com:1883"),
				ClientID: ptr("my-shelly"),
				User:     ptr("mqttuser"),
				Pass:     ptr("mqttpass"),
			},
		},
		{
			name: "tls with default ca",
			config: &MQTTConfig{
				Enable: ptr(true),
				Server: ptr("mqtt.example.com:8883"),
				SSLCA:  ptr("ca.pem"),
			},
		},
		{
			name: "tls with user ca",
			config: &MQTTConfig{
				Enable: ptr(true),
				Server: ptr("mqtt.example.com:8883"),
				SSLCA:  ptr("user_ca.pem"),
			},
		},
		{
			name: "tls without verification",
			config: &MQTTConfig{
				Enable: ptr(true),
				Server: ptr("mqtt.example.com:8883"),
				SSLCA:  ptr("*"),
			},
		},
		{
			name: "custom topic prefix",
			config: &MQTTConfig{
				Enable:      ptr(true),
				Server:      ptr("mqtt.example.com"),
				TopicPrefix: ptr("home/basement/shelly"),
			},
		},
		{
			name: "notification settings",
			config: &MQTTConfig{
				Enable:    ptr(true),
				Server:    ptr("mqtt.example.com"),
				RPCNTF:    ptr(true),
				StatusNTF: ptr(true),
			},
		},
		{
			name: "client cert",
			config: &MQTTConfig{
				Enable:        ptr(true),
				Server:        ptr("mqtt.example.com:8883"),
				SSLCA:         ptr("ca.pem"),
				UseClientCert: ptr(true),
			},
		},
		{
			name: "disable control",
			config: &MQTTConfig{
				Enable:        ptr(true),
				EnableControl: ptr(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "MQTT.SetConfig" {
						t.Errorf("method = %q, want %q", method, "MQTT.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			mqtt := NewMQTT(client)

			err := mqtt.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestMQTT_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	mqtt := NewMQTT(client)
	testComponentError(t, "SetConfig", func() error {
		return mqtt.SetConfig(context.Background(), &MQTTConfig{})
	})
}

func TestMQTT_GetStatus(t *testing.T) {
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
					if method != "MQTT.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			mqtt := NewMQTT(client)

			status, err := mqtt.GetStatus(context.Background())
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

func TestMQTT_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	mqtt := NewMQTT(client)
	testComponentError(t, "GetStatus", func() error {
		_, err := mqtt.GetStatus(context.Background())
		return err
	})
}

func TestMQTT_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	mqtt := NewMQTT(client)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := mqtt.GetStatus(context.Background())
		return err
	})
}

func TestMQTTConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config MQTTConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: MQTTConfig{
				Enable:        ptr(true),
				Server:        ptr("mqtt.example.com:1883"),
				ClientID:      ptr("shelly-1"),
				User:          ptr("admin"),
				Pass:          ptr("secret"),
				TopicPrefix:   ptr("home/devices"),
				RPCNTF:        ptr(true),
				StatusNTF:     ptr(false),
				UseClientCert: ptr(false),
				EnableControl: ptr(true),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["enable"].(bool) != true {
					t.Errorf("enable = %v, want true", data["enable"])
				}
				if data["server"].(string) != "mqtt.example.com:1883" {
					t.Errorf("server = %v, want mqtt.example.com:1883", data["server"])
				}
				if data["client_id"].(string) != "shelly-1" {
					t.Errorf("client_id = %v, want shelly-1", data["client_id"])
				}
				if data["topic_prefix"].(string) != "home/devices" {
					t.Errorf("topic_prefix = %v, want home/devices", data["topic_prefix"])
				}
			},
		},
		{
			name: "minimal config",
			config: MQTTConfig{
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

func TestMQTT_ContextCancellation(t *testing.T) {
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
	mqtt := NewMQTT(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mqtt.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
