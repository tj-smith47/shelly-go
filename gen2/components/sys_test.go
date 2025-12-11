package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

func TestNewSys(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	sys := NewSys(client)

	if sys == nil {
		t.Fatal("NewSys returned nil")
	}

	if sys.Type() != "sys" {
		t.Errorf("Type() = %q, want %q", sys.Type(), "sys")
	}

	if sys.Key() != "sys" {
		t.Errorf("Key() = %q, want %q", sys.Key(), "sys")
	}

	if sys.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestSys_GetConfig(t *testing.T) {
	tests := []struct {
		wantName *string
		wantTZ   *string
		wantEco  *bool
		name     string
		result   string
	}{
		{
			name: "full config",
			result: `{
				"device": {
					"name": "My Shelly",
					"mac": "AABBCCDDEEFF",
					"fw_id": "20231107-164738/1.0.8-g8192e12",
					"profile": "switch",
					"eco_mode": false,
					"discoverable": true
				},
				"location": {
					"tz": "America/New_York",
					"lat": 40.7128,
					"lng": -74.0060
				},
				"debug": {
					"mqtt": {"enable": false},
					"websocket": {"enable": false},
					"udp": {"addr": null}
				},
				"ui_data": {},
				"rpc_udp": {"dst_addr": null, "listen_port": null},
				"sntp": {"server": "time.google.com"},
				"cfg_rev": 15
			}`,
			wantName: ptr("My Shelly"),
			wantTZ:   ptr("America/New_York"),
			wantEco:  ptr(false),
		},
		{
			name: "minimal config",
			result: `{
				"device": {"name": null, "mac": "AABBCCDDEEFF"},
				"location": {"tz": "UTC"},
				"cfg_rev": 1
			}`,
			wantTZ: ptr("UTC"),
		},
		{
			name: "eco mode enabled",
			result: `{
				"device": {
					"name": "Eco Device",
					"eco_mode": true
				},
				"cfg_rev": 5
			}`,
			wantName: ptr("Eco Device"),
			wantEco:  ptr(true),
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
					if method != "Sys.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			sys := NewSys(client)

			config, err := sys.GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if config == nil {
				t.Fatal("GetConfig() returned nil config")
			}

			if tt.wantName != nil {
				if config.Device == nil || config.Device.Name == nil || *config.Device.Name != *tt.wantName {
					var got *string
					if config.Device != nil {
						got = config.Device.Name
					}
					t.Errorf("config.Device.Name = %v, want %v", got, *tt.wantName)
				}
			}

			if tt.wantTZ != nil {
				if config.Location == nil || config.Location.TZ == nil || *config.Location.TZ != *tt.wantTZ {
					var got *string
					if config.Location != nil {
						got = config.Location.TZ
					}
					t.Errorf("config.Location.TZ = %v, want %v", got, *tt.wantTZ)
				}
			}

			if tt.wantEco != nil {
				if config.Device == nil || config.Device.EcoMode == nil || *config.Device.EcoMode != *tt.wantEco {
					var got *bool
					if config.Device != nil {
						got = config.Device.EcoMode
					}
					t.Errorf("config.Device.EcoMode = %v, want %v", got, *tt.wantEco)
				}
			}
		})
	}
}

func TestSys_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	sys := NewSys(client)
	testComponentError(t, "GetConfig", func() error {
		_, err := sys.GetConfig(context.Background())
		return err
	})
}

func TestSys_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	sys := NewSys(client)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := sys.GetConfig(context.Background())
		return err
	})
}

func TestSys_SetConfig(t *testing.T) {
	tests := []struct {
		config *SysConfig
		name   string
	}{
		{
			name: "set device name",
			config: &SysConfig{
				Device: &SysDeviceConfig{
					Name: ptr("New Device Name"),
				},
			},
		},
		{
			name: "set timezone",
			config: &SysConfig{
				Location: &SysLocationConfig{
					TZ: ptr("Europe/London"),
				},
			},
		},
		{
			name: "set location coordinates",
			config: &SysConfig{
				Location: &SysLocationConfig{
					Lat: ptrFloat(51.5074),
					Lng: ptrFloat(-0.1278),
				},
			},
		},
		{
			name: "enable eco mode",
			config: &SysConfig{
				Device: &SysDeviceConfig{
					EcoMode: ptr(true),
				},
			},
		},
		{
			name: "set debug config",
			config: &SysConfig{
				Debug: &SysDebugConfig{
					MQTT: &SysDebugTargetConfig{
						Enable: ptr(true),
					},
					Websocket: &SysDebugTargetConfig{
						Enable: ptr(false),
					},
					UDP: &SysDebugUDPConfig{
						Addr: ptr("192.168.1.100:1234"),
					},
				},
			},
		},
		{
			name: "set sntp server",
			config: &SysConfig{
				SNTP: &SysSNTPConfig{
					Server: ptr("pool.ntp.org"),
				},
			},
		},
		{
			name: "set rpc udp config",
			config: &SysConfig{
				RPCUDP: &SysRPCUDPConfig{
					ListenPort: ptr(5683),
				},
			},
		},
		{
			name: "set discoverable",
			config: &SysConfig{
				Device: &SysDeviceConfig{
					Discoverable: ptr(false),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Sys.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Sys.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			sys := NewSys(client)

			err := sys.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestSys_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	sys := NewSys(client)
	testComponentError(t, "SetConfig", func() error {
		return sys.SetConfig(context.Background(), &SysConfig{})
	})
}

func TestSys_GetStatus(t *testing.T) {
	tests := []struct {
		wantUpdateVersion *string
		name              string
		result            string
		wantMAC           string
		wantUptime        int
		wantRAMFree       int
		wantRestart       bool
	}{
		{
			name: "full status",
			result: `{
				"mac": "AABBCCDDEEFF",
				"restart_required": false,
				"time": "14:30",
				"unixtime": 1699369800,
				"uptime": 86400,
				"ram_size": 262144,
				"ram_free": 131072,
				"fs_size": 458752,
				"fs_free": 229376,
				"cfg_rev": 15,
				"kvs_rev": 5,
				"schedule_rev": 2,
				"webhook_rev": 3,
				"available_updates": {
					"stable": {"version": "1.0.9", "build_id": "20231115-123456/1.0.9-g1234567"},
					"beta": {"version": "1.1.0-beta1"}
				}
			}`,
			wantMAC:           "AABBCCDDEEFF",
			wantUptime:        86400,
			wantRAMFree:       131072,
			wantRestart:       false,
			wantUpdateVersion: ptr("1.0.9"),
		},
		{
			name: "restart required",
			result: `{
				"mac": "112233445566",
				"restart_required": true,
				"uptime": 3600,
				"ram_size": 262144,
				"ram_free": 100000,
				"fs_size": 458752,
				"fs_free": 200000,
				"cfg_rev": 20,
				"kvs_rev": 10
			}`,
			wantMAC:     "112233445566",
			wantUptime:  3600,
			wantRAMFree: 100000,
			wantRestart: true,
		},
		{
			name: "battery device with wakeup",
			result: `{
				"mac": "AABBCCDDEEFF",
				"restart_required": false,
				"uptime": 120,
				"ram_size": 262144,
				"ram_free": 150000,
				"fs_size": 458752,
				"fs_free": 250000,
				"cfg_rev": 5,
				"kvs_rev": 2,
				"wakeup_reason": {
					"boot": "deepsleep_wake",
					"cause": "button"
				},
				"wakeup_period": 3600
			}`,
			wantMAC:     "AABBCCDDEEFF",
			wantUptime:  120,
			wantRAMFree: 150000,
			wantRestart: false,
		},
		{
			name: "with reset reason",
			result: `{
				"mac": "FFEEDDCCBBAA",
				"restart_required": false,
				"uptime": 60,
				"ram_size": 262144,
				"ram_free": 200000,
				"fs_size": 458752,
				"fs_free": 300000,
				"cfg_rev": 1,
				"kvs_rev": 0,
				"reset_reason": 1
			}`,
			wantMAC:     "FFEEDDCCBBAA",
			wantUptime:  60,
			wantRAMFree: 200000,
			wantRestart: false,
		},
		{
			name: "time not synced",
			result: `{
				"mac": "AABBCCDDEEFF",
				"restart_required": false,
				"time": null,
				"unixtime": null,
				"uptime": 30,
				"ram_size": 262144,
				"ram_free": 180000,
				"fs_size": 458752,
				"fs_free": 350000,
				"cfg_rev": 1,
				"kvs_rev": 0
			}`,
			wantMAC:     "AABBCCDDEEFF",
			wantUptime:  30,
			wantRAMFree: 180000,
			wantRestart: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Sys.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			sys := NewSys(client)

			status, err := sys.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if status.MAC != tt.wantMAC {
				t.Errorf("status.MAC = %q, want %q", status.MAC, tt.wantMAC)
			}

			if status.Uptime != tt.wantUptime {
				t.Errorf("status.Uptime = %d, want %d", status.Uptime, tt.wantUptime)
			}

			if status.RAMFree != tt.wantRAMFree {
				t.Errorf("status.RAMFree = %d, want %d", status.RAMFree, tt.wantRAMFree)
			}

			if status.RestartRequired != tt.wantRestart {
				t.Errorf("status.RestartRequired = %v, want %v", status.RestartRequired, tt.wantRestart)
			}

			if tt.wantUpdateVersion != nil {
				if status.AvailableUpdates == nil || status.AvailableUpdates.Stable == nil {
					t.Error("expected available updates to be present")
				} else if status.AvailableUpdates.Stable.Version != *tt.wantUpdateVersion {
					t.Errorf("status.AvailableUpdates.Stable.Version = %q, want %q",
						status.AvailableUpdates.Stable.Version, *tt.wantUpdateVersion)
				}
			}
		})
	}
}

func TestSys_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	sys := NewSys(client)
	testComponentError(t, "GetStatus", func() error {
		_, err := sys.GetStatus(context.Background())
		return err
	})
}

func TestSys_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	sys := NewSys(client)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := sys.GetStatus(context.Background())
		return err
	})
}

func TestSysConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config SysConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: SysConfig{
				Device: &SysDeviceConfig{
					Name:         ptr("Test Device"),
					EcoMode:      ptr(true),
					Discoverable: ptr(true),
				},
				Location: &SysLocationConfig{
					TZ:  ptr("America/Chicago"),
					Lat: ptrFloat(41.8781),
					Lng: ptrFloat(-87.6298),
				},
				Debug: &SysDebugConfig{
					MQTT: &SysDebugTargetConfig{Enable: ptr(true)},
				},
				SNTP: &SysSNTPConfig{
					Server: ptr("ntp.example.com"),
				},
			},
			check: func(t *testing.T, data map[string]any) {
				device := data["device"].(map[string]any)
				if device["name"].(string) != "Test Device" {
					t.Errorf("device.name = %v, want Test Device", device["name"])
				}
				if device["eco_mode"].(bool) != true {
					t.Errorf("device.eco_mode = %v, want true", device["eco_mode"])
				}
				location := data["location"].(map[string]any)
				if location["tz"].(string) != "America/Chicago" {
					t.Errorf("location.tz = %v, want America/Chicago", location["tz"])
				}
			},
		},
		{
			name: "minimal config",
			config: SysConfig{
				Device: &SysDeviceConfig{
					Name: ptr("Minimal"),
				},
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["location"]; ok {
					t.Error("location should not be present")
				}
				if _, ok := data["debug"]; ok {
					t.Error("debug should not be present")
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

func TestSys_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"mac": "AABBCCDDEEFF", "restart_required": false, "uptime": 100, "ram_size": 262144, "ram_free": 131072, "fs_size": 458752, "fs_free": 229376, "cfg_rev": 1, "kvs_rev": 0}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	sys := NewSys(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := sys.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

// Helper function for float pointers
func ptrFloat(v float64) *float64 {
	return &v
}
