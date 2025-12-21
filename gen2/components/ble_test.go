package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewBLE(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	ble := NewBLE(client)

	if ble == nil {
		t.Fatal("NewBLE returned nil")
	}

	if ble.Type() != "ble" {
		t.Errorf("Type() = %q, want %q", ble.Type(), "ble")
	}

	if ble.Key() != "ble" {
		t.Errorf("Key() = %q, want %q", ble.Key(), "ble")
	}

	if ble.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestBLE_GetConfig(t *testing.T) {
	tests := []struct {
		wantEnable   *bool
		wantRPC      *bool
		wantObserver *bool
		name         string
		result       string
	}{
		{
			name:         "fully enabled",
			result:       `{"enable": true, "rpc": {"enable": true}, "observer": {"enable": true}}`,
			wantEnable:   ptr(true),
			wantRPC:      ptr(true),
			wantObserver: ptr(true),
		},
		{
			name:       "disabled",
			result:     `{"enable": false}`,
			wantEnable: ptr(false),
		},
		{
			name:       "bluetooth enabled, rpc disabled",
			result:     `{"enable": true, "rpc": {"enable": false}}`,
			wantEnable: ptr(true),
			wantRPC:    ptr(false),
		},
		{
			name:         "with observer only",
			result:       `{"enable": true, "observer": {"enable": true}}`,
			wantEnable:   ptr(true),
			wantObserver: ptr(true),
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
					if method != "BLE.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			ble := NewBLE(client)

			config, err := ble.GetConfig(context.Background())
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

			if tt.wantRPC != nil {
				if config.RPC == nil {
					t.Error("config.RPC is nil, want non-nil")
				} else if config.RPC.Enable == nil || *config.RPC.Enable != *tt.wantRPC {
					t.Errorf("config.RPC.Enable = %v, want %v", config.RPC.Enable, *tt.wantRPC)
				}
			}

			if tt.wantObserver != nil {
				if config.Observer == nil {
					t.Error("config.Observer is nil, want non-nil")
				} else if config.Observer.Enable == nil || *config.Observer.Enable != *tt.wantObserver {
					t.Errorf("config.Observer.Enable = %v, want %v", config.Observer.Enable, *tt.wantObserver)
				}
			}
		})
	}
}

func TestBLE_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	ble := NewBLE(client)
	testComponentError(t, "GetConfig", func() error {
		_, err := ble.GetConfig(context.Background())
		return err
	})
}

func TestBLE_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	ble := NewBLE(client)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := ble.GetConfig(context.Background())
		return err
	})
}

func TestBLE_SetConfig(t *testing.T) {
	tests := []struct {
		config *BLEConfig
		name   string
	}{
		{
			name: "enable bluetooth",
			config: &BLEConfig{
				Enable: ptr(true),
			},
		},
		{
			name: "disable bluetooth",
			config: &BLEConfig{
				Enable: ptr(false),
			},
		},
		{
			name: "enable rpc",
			config: &BLEConfig{
				Enable: ptr(true),
				RPC: &BLERPCConfig{
					Enable: ptr(true),
				},
			},
		},
		{
			name: "enable observer",
			config: &BLEConfig{
				Enable: ptr(true),
				Observer: &BLEObserverConfig{
					Enable: ptr(true),
				},
			},
		},
		{
			name: "full configuration",
			config: &BLEConfig{
				Enable: ptr(true),
				RPC: &BLERPCConfig{
					Enable: ptr(true),
				},
				Observer: &BLEObserverConfig{
					Enable: ptr(true),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "BLE.SetConfig" {
						t.Errorf("method = %q, want %q", method, "BLE.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			ble := NewBLE(client)

			err := ble.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestBLE_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	ble := NewBLE(client)
	testComponentError(t, "SetConfig", func() error {
		return ble.SetConfig(context.Background(), &BLEConfig{})
	})
}

func TestBLE_GetStatus(t *testing.T) {
	tests := []struct {
		name   string
		result string
	}{
		{
			name:   "empty status",
			result: `{}`,
		},
		{
			name:   "status with unknown fields",
			result: `{"some_future_field": true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "BLE.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			ble := NewBLE(client)

			status, err := ble.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}
		})
	}
}

func TestBLE_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	ble := NewBLE(client)
	testComponentError(t, "GetStatus", func() error {
		_, err := ble.GetStatus(context.Background())
		return err
	})
}

func TestBLE_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	ble := NewBLE(client)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := ble.GetStatus(context.Background())
		return err
	})
}

func TestBLEConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config BLEConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: BLEConfig{
				Enable: ptr(true),
				RPC: &BLERPCConfig{
					Enable: ptr(true),
				},
				Observer: &BLEObserverConfig{
					Enable: ptr(false),
				},
			},
			check: func(t *testing.T, data map[string]any) {
				enable, ok := data["enable"].(bool)
				if !ok || enable != true {
					t.Errorf("enable = %v, want true", data["enable"])
				}
				rpc, ok := data["rpc"].(map[string]any)
				if !ok {
					t.Fatalf("rpc type assertion failed")
				}
				rpcEnable, ok := rpc["enable"].(bool)
				if !ok || rpcEnable != true {
					t.Errorf("rpc.enable = %v, want true", rpc["enable"])
				}
				observer, ok := data["observer"].(map[string]any)
				if !ok {
					t.Fatalf("observer type assertion failed")
				}
				observerEnable, ok := observer["enable"].(bool)
				if !ok || observerEnable != false {
					t.Errorf("observer.enable = %v, want false", observer["enable"])
				}
			},
		},
		{
			name: "minimal config",
			config: BLEConfig{
				Enable: ptr(false),
			},
			check: func(t *testing.T, data map[string]any) {
				enable, ok := data["enable"].(bool)
				if !ok || enable != false {
					t.Errorf("enable = %v, want false", data["enable"])
				}
				if _, ok := data["rpc"]; ok {
					t.Error("rpc should not be present")
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

func TestBLE_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					_ = req.GetMethod()
					select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	ble := NewBLE(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ble.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
