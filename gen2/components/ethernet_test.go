package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewEthernet(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	eth := NewEthernet(client)

	if eth == nil {
		t.Fatal("NewEthernet returned nil")
	}

	if eth.Type() != "eth" {
		t.Errorf("Type() = %q, want %q", eth.Type(), "eth")
	}

	if eth.Key() != "eth" {
		t.Errorf("Key() = %q, want %q", eth.Key(), "eth")
	}

	if eth.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestEthernet_GetConfig(t *testing.T) {
	tests := []struct {
		wantEnable  *bool
		wantIPv4    *string
		wantIP      *string
		wantNetmask *string
		wantGW      *string
		wantNS      *string
		name        string
		result      string
	}{
		{
			name:       "dhcp config",
			result:     `{"enable": true, "ipv4mode": "dhcp"}`,
			wantEnable: ptr(true),
			wantIPv4:   ptr("dhcp"),
		},
		{
			name: "static config",
			result: `{
				"enable": true,
				"ipv4mode": "static",
				"ip": "192.168.1.50",
				"netmask": "255.255.255.0",
				"gw": "192.168.1.1",
				"nameserver": "8.8.8.8"
			}`,
			wantEnable:  ptr(true),
			wantIPv4:    ptr("static"),
			wantIP:      ptr("192.168.1.50"),
			wantNetmask: ptr("255.255.255.0"),
			wantGW:      ptr("192.168.1.1"),
			wantNS:      ptr("8.8.8.8"),
		},
		{
			name:       "disabled",
			result:     `{"enable": false}`,
			wantEnable: ptr(false),
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
					if method != "Eth.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			eth := NewEthernet(client)

			config, err := eth.GetConfig(context.Background())
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

			if tt.wantIPv4 != nil {
				if config.IPv4Mode == nil || *config.IPv4Mode != *tt.wantIPv4 {
					t.Errorf("config.IPv4Mode = %v, want %v", config.IPv4Mode, *tt.wantIPv4)
				}
			}

			if tt.wantIP != nil {
				if config.IP == nil || *config.IP != *tt.wantIP {
					t.Errorf("config.IP = %v, want %v", config.IP, *tt.wantIP)
				}
			}

			if tt.wantNetmask != nil {
				if config.Netmask == nil || *config.Netmask != *tt.wantNetmask {
					t.Errorf("config.Netmask = %v, want %v", config.Netmask, *tt.wantNetmask)
				}
			}

			if tt.wantGW != nil {
				if config.GW == nil || *config.GW != *tt.wantGW {
					t.Errorf("config.GW = %v, want %v", config.GW, *tt.wantGW)
				}
			}

			if tt.wantNS != nil {
				if config.Nameserver == nil || *config.Nameserver != *tt.wantNS {
					t.Errorf("config.Nameserver = %v, want %v", config.Nameserver, *tt.wantNS)
				}
			}
		})
	}
}

func TestEthernet_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	eth := NewEthernet(client)
	testComponentError(t, "GetConfig", func() error {
		_, err := eth.GetConfig(context.Background())
		return err
	})
}

func TestEthernet_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	eth := NewEthernet(client)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := eth.GetConfig(context.Background())
		return err
	})
}

func TestEthernet_SetConfig(t *testing.T) {
	tests := []struct {
		config *EthernetConfig
		name   string
	}{
		{
			name: "enable with dhcp",
			config: &EthernetConfig{
				Enable:   ptr(true),
				IPv4Mode: ptr("dhcp"),
			},
		},
		{
			name: "disable ethernet",
			config: &EthernetConfig{
				Enable: ptr(false),
			},
		},
		{
			name: "static ip configuration",
			config: &EthernetConfig{
				Enable:     ptr(true),
				IPv4Mode:   ptr("static"),
				IP:         ptr("192.168.1.100"),
				Netmask:    ptr("255.255.255.0"),
				GW:         ptr("192.168.1.1"),
				Nameserver: ptr("8.8.8.8"),
			},
		},
		{
			name: "change nameserver only",
			config: &EthernetConfig{
				Nameserver: ptr("1.1.1.1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Eth.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Eth.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			eth := NewEthernet(client)

			err := eth.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestEthernet_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	eth := NewEthernet(client)
	testComponentError(t, "SetConfig", func() error {
		return eth.SetConfig(context.Background(), &EthernetConfig{})
	})
}

func TestEthernet_GetStatus(t *testing.T) {
	tests := []struct {
		wantIP *string
		name   string
		result string
	}{
		{
			name:   "connected",
			result: `{"ip": "192.168.1.50"}`,
			wantIP: ptr("192.168.1.50"),
		},
		{
			name:   "disconnected",
			result: `{}`,
			wantIP: nil,
		},
		{
			name:   "null ip",
			result: `{"ip": null}`,
			wantIP: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Eth.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			eth := NewEthernet(client)

			status, err := eth.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if tt.wantIP != nil {
				if status.IP == nil || *status.IP != *tt.wantIP {
					t.Errorf("status.IP = %v, want %v", status.IP, *tt.wantIP)
				}
			} else if status.IP != nil {
				t.Errorf("status.IP = %v, want nil", *status.IP)
			}
		})
	}
}

func TestEthernet_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	eth := NewEthernet(client)
	testComponentError(t, "GetStatus", func() error {
		_, err := eth.GetStatus(context.Background())
		return err
	})
}

func TestEthernet_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	eth := NewEthernet(client)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := eth.GetStatus(context.Background())
		return err
	})
}

func TestEthernetConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config EthernetConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full static config",
			config: EthernetConfig{
				Enable:     ptr(true),
				IPv4Mode:   ptr("static"),
				IP:         ptr("10.0.0.100"),
				Netmask:    ptr("255.255.255.0"),
				GW:         ptr("10.0.0.1"),
				Nameserver: ptr("10.0.0.1"),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["enable"].(bool) != true {
					t.Errorf("enable = %v, want true", data["enable"])
				}
				if data["ipv4mode"].(string) != "static" {
					t.Errorf("ipv4mode = %v, want static", data["ipv4mode"])
				}
				if data["ip"].(string) != "10.0.0.100" {
					t.Errorf("ip = %v, want 10.0.0.100", data["ip"])
				}
			},
		},
		{
			name: "dhcp only",
			config: EthernetConfig{
				Enable:   ptr(true),
				IPv4Mode: ptr("dhcp"),
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["ip"]; ok {
					t.Error("ip should not be present for dhcp config")
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

func TestEthernet_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					_ = req.GetMethod()
					select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"ip": "192.168.1.1"}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	eth := NewEthernet(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := eth.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
