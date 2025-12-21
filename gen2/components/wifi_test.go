package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewWiFi(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	wifi := NewWiFi(client)

	if wifi == nil {
		t.Fatal("NewWiFi returned nil")
	}

	if wifi.Type() != "wifi" {
		t.Errorf("Type() = %q, want %q", wifi.Type(), "wifi")
	}

	if wifi.Key() != "wifi" {
		t.Errorf("Key() = %q, want %q", wifi.Key(), "wifi")
	}

	if wifi.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestWiFi_GetConfig(t *testing.T) {
	tests := []struct {
		wantSTASSID *string
		wantAPSSID  *string
		name        string
		result      string
		wantRoam    bool
	}{
		{
			name: "full config",
			result: `{
				"ap": {
					"ssid": "ShellyAP",
					"is_open": false,
					"enable": true,
					"range_extender": {"enable": false}
				},
				"sta": {
					"ssid": "HomeNetwork",
					"is_open": false,
					"enable": true,
					"ipv4mode": "dhcp"
				},
				"sta1": {
					"ssid": "BackupNetwork",
					"is_open": false,
					"enable": false
				},
				"roam": {
					"rssi_thr": -80,
					"interval": 60
				}
			}`,
			wantSTASSID: ptr("HomeNetwork"),
			wantAPSSID:  ptr("ShellyAP"),
			wantRoam:    true,
		},
		{
			name: "minimal config",
			result: `{
				"sta": {
					"ssid": "MyNetwork",
					"enable": true
				}
			}`,
			wantSTASSID: ptr("MyNetwork"),
		},
		{
			name: "ap only config",
			result: `{
				"ap": {
					"ssid": "DeviceAP",
					"enable": true
				}
			}`,
			wantAPSSID: ptr("DeviceAP"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Wifi.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			wifi := NewWiFi(client)

			config, err := wifi.GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if config == nil {
				t.Fatal("GetConfig() returned nil config")
			}

			if tt.wantSTASSID != nil {
				if config.STA == nil {
					t.Error("config.STA is nil, want non-nil")
				} else if config.STA.SSID == nil || *config.STA.SSID != *tt.wantSTASSID {
					t.Errorf("config.STA.SSID = %v, want %v", config.STA.SSID, *tt.wantSTASSID)
				}
			}

			if tt.wantAPSSID != nil {
				if config.AP == nil {
					t.Error("config.AP is nil, want non-nil")
				} else if config.AP.SSID == nil || *config.AP.SSID != *tt.wantAPSSID {
					t.Errorf("config.AP.SSID = %v, want %v", config.AP.SSID, *tt.wantAPSSID)
				}
			}

			if tt.wantRoam {
				if config.Roam == nil {
					t.Error("config.Roam is nil, want non-nil")
				}
			}
		})
	}
}

func TestWiFi_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	wifi := NewWiFi(client)
	testComponentError(t, "GetConfig", func() error {
		_, err := wifi.GetConfig(context.Background())
		return err
	})
}

func TestWiFi_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	wifi := NewWiFi(client)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := wifi.GetConfig(context.Background())
		return err
	})
}

func TestWiFi_SetConfig(t *testing.T) {
	tests := []struct {
		config *WiFiConfig
		name   string
	}{
		{
			name: "set station config",
			config: &WiFiConfig{
				STA: &WiFiStationConfig{
					SSID:   ptr("MyNetwork"),
					Pass:   ptr("password123"),
					Enable: ptr(true),
				},
			},
		},
		{
			name: "set AP config",
			config: &WiFiConfig{
				AP: &WiFiAPConfig{
					SSID:   ptr("ShellyAP"),
					Pass:   ptr("appassword"),
					Enable: ptr(true),
				},
			},
		},
		{
			name: "set static IP",
			config: &WiFiConfig{
				STA: &WiFiStationConfig{
					SSID:       ptr("MyNetwork"),
					Pass:       ptr("password"),
					Enable:     ptr(true),
					IPv4Mode:   ptr("static"),
					IP:         ptr("192.168.1.100"),
					Netmask:    ptr("255.255.255.0"),
					GW:         ptr("192.168.1.1"),
					Nameserver: ptr("8.8.8.8"),
				},
			},
		},
		{
			name: "set roaming config",
			config: &WiFiConfig{
				Roam: &WiFiRoamConfig{
					RSSIThr:  ptr(-75.0),
					Interval: ptr(30.0),
				},
			},
		},
		{
			name: "enable range extender",
			config: &WiFiConfig{
				AP: &WiFiAPConfig{
					Enable: ptr(true),
					RangeExtender: &WiFiAPRangeExtenderConfig{
						Enable: ptr(true),
					},
				},
			},
		},
		{
			name: "full configuration",
			config: &WiFiConfig{
				STA: &WiFiStationConfig{
					SSID:   ptr("PrimaryNetwork"),
					Pass:   ptr("password1"),
					Enable: ptr(true),
				},
				STA1: &WiFiStationConfig{
					SSID:   ptr("BackupNetwork"),
					Pass:   ptr("password2"),
					Enable: ptr(true),
				},
				AP: &WiFiAPConfig{
					SSID:   ptr("DeviceAP"),
					Pass:   ptr("appass"),
					Enable: ptr(false),
				},
				Roam: &WiFiRoamConfig{
					RSSIThr:  ptr(-80.0),
					Interval: ptr(60.0),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Wifi.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Wifi.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			wifi := NewWiFi(client)

			err := wifi.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestWiFi_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	wifi := NewWiFi(client)
	testComponentError(t, "SetConfig", func() error {
		return wifi.SetConfig(context.Background(), &WiFiConfig{})
	})
}

func TestWiFi_GetStatus(t *testing.T) {
	tests := []struct {
		wantSSID      *string
		wantStaIP     *string
		wantRSSI      *float64
		wantAPClients *int
		name          string
		result        string
		wantStatus    string
	}{
		{
			name: "connected with IP",
			result: `{
				"sta_ip": "192.168.1.100",
				"status": "got ip",
				"ssid": "HomeNetwork",
				"rssi": -65
			}`,
			wantStatus: "got ip",
			wantSSID:   ptr("HomeNetwork"),
			wantStaIP:  ptr("192.168.1.100"),
			wantRSSI:   ptr(-65.0),
		},
		{
			name:       "disconnected",
			result:     `{"status": "disconnected"}`,
			wantStatus: "disconnected",
		},
		{
			name:       "connecting",
			result:     `{"status": "connecting"}`,
			wantStatus: "connecting",
		},
		{
			name: "connected with AP clients",
			result: `{
				"sta_ip": "192.168.1.50",
				"status": "got ip",
				"ssid": "MainNetwork",
				"rssi": -55,
				"ap_client_count": 3
			}`,
			wantStatus:    "got ip",
			wantSSID:      ptr("MainNetwork"),
			wantStaIP:     ptr("192.168.1.50"),
			wantRSSI:      ptr(-55.0),
			wantAPClients: ptr(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Wifi.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			wifi := NewWiFi(client)

			status, err := wifi.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if status.Status != tt.wantStatus {
				t.Errorf("status.Status = %q, want %q", status.Status, tt.wantStatus)
			}

			if tt.wantSSID != nil {
				if status.SSID == nil || *status.SSID != *tt.wantSSID {
					t.Errorf("status.SSID = %v, want %v", status.SSID, *tt.wantSSID)
				}
			}

			if tt.wantStaIP != nil {
				if status.StaIP == nil || *status.StaIP != *tt.wantStaIP {
					t.Errorf("status.StaIP = %v, want %v", status.StaIP, *tt.wantStaIP)
				}
			}

			if tt.wantRSSI != nil {
				if status.RSSI == nil || *status.RSSI != *tt.wantRSSI {
					t.Errorf("status.RSSI = %v, want %v", status.RSSI, *tt.wantRSSI)
				}
			}

			if tt.wantAPClients != nil {
				if status.APClientCount == nil || *status.APClientCount != *tt.wantAPClients {
					t.Errorf("status.APClientCount = %v, want %v", status.APClientCount, *tt.wantAPClients)
				}
			}
		})
	}
}

func TestWiFi_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	wifi := NewWiFi(client)
	testComponentError(t, "GetStatus", func() error {
		_, err := wifi.GetStatus(context.Background())
		return err
	})
}

func TestWiFi_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	wifi := NewWiFi(client)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := wifi.GetStatus(context.Background())
		return err
	})
}

func TestWiFi_Scan(t *testing.T) {
	tests := []struct {
		checkFirst func(t *testing.T, r *WiFiScanResult)
		name       string
		result     string
		wantCount  int
	}{
		{
			name: "multiple networks",
			result: `{
				"results": [
					{"ssid": "Network1", "bssid": "AA:BB:CC:DD:EE:01", "auth": "wpa2_psk",
						"channel": 6, "rssi": -45},
					{"ssid": "Network2", "bssid": "AA:BB:CC:DD:EE:02", "auth": "wpa_wpa2_psk",
						"channel": 11, "rssi": -70},
					{"ssid": "OpenNetwork", "bssid": "AA:BB:CC:DD:EE:03", "auth": "open",
						"channel": 1, "rssi": -80}
				]
			}`,
			wantCount: 3,
			checkFirst: func(t *testing.T, r *WiFiScanResult) {
				if r.SSID == nil || *r.SSID != "Network1" {
					t.Errorf("first result SSID = %v, want Network1", r.SSID)
				}
				if r.Auth == nil || *r.Auth != "wpa2_psk" {
					t.Errorf("first result Auth = %v, want wpa2_psk", r.Auth)
				}
				if r.RSSI == nil || *r.RSSI != -45 {
					t.Errorf("first result RSSI = %v, want -45", r.RSSI)
				}
			},
		},
		{
			name:      "empty scan",
			result:    `{"results": []}`,
			wantCount: 0,
		},
		{
			name: "single network",
			result: `{
				"results": [
					{"ssid": "SingleNetwork", "auth": "wpa3_psk", "channel": 36, "rssi": -50}
				]
			}`,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Wifi.Scan" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			wifi := NewWiFi(client)

			result, err := wifi.Scan(context.Background())
			if err != nil {
				t.Errorf("Scan() error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("Scan() returned nil result")
			}

			if len(result.Results) != tt.wantCount {
				t.Errorf("len(Results) = %d, want %d", len(result.Results), tt.wantCount)
			}

			if tt.checkFirst != nil && len(result.Results) > 0 {
				tt.checkFirst(t, &result.Results[0])
			}
		})
	}
}

func TestWiFi_Scan_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	wifi := NewWiFi(client)
	testComponentError(t, "Scan", func() error {
		_, err := wifi.Scan(context.Background())
		return err
	})
}

func TestWiFi_Scan_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	wifi := NewWiFi(client)
	testComponentInvalidJSON(t, "Scan", func() error {
		_, err := wifi.Scan(context.Background())
		return err
	})
}

func TestWiFi_ListAPClients(t *testing.T) {
	tests := []struct {
		checkFirst func(t *testing.T, c *WiFiAPClient)
		name       string
		result     string
		wantCount  int
	}{
		{
			name: "multiple clients",
			result: `{
				"ts": 1699999999,
				"ap_clients": [
					{"mac": "AA:BB:CC:DD:EE:01", "ip": "192.168.33.2", "since": 1699990000},
					{"mac": "AA:BB:CC:DD:EE:02", "ip": "192.168.33.3", "since": 1699995000}
				]
			}`,
			wantCount: 2,
			checkFirst: func(t *testing.T, c *WiFiAPClient) {
				if c.MAC != "AA:BB:CC:DD:EE:01" {
					t.Errorf("first client MAC = %v, want AA:BB:CC:DD:EE:01", c.MAC)
				}
				if c.IP == nil || *c.IP != "192.168.33.2" {
					t.Errorf("first client IP = %v, want 192.168.33.2", c.IP)
				}
			},
		},
		{
			name:      "no clients",
			result:    `{"ts": 1699999999, "ap_clients": []}`,
			wantCount: 0,
		},
		{
			name: "single client",
			result: `{
				"ts": 1699999999,
				"ap_clients": [
					{"mac": "11:22:33:44:55:66", "ip": "192.168.33.5"}
				]
			}`,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Wifi.ListAPClients" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			wifi := NewWiFi(client)

			result, err := wifi.ListAPClients(context.Background())
			if err != nil {
				t.Errorf("ListAPClients() error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("ListAPClients() returned nil result")
			}

			if len(result.APClients) != tt.wantCount {
				t.Errorf("len(APClients) = %d, want %d", len(result.APClients), tt.wantCount)
			}

			if tt.checkFirst != nil && len(result.APClients) > 0 {
				tt.checkFirst(t, &result.APClients[0])
			}
		})
	}
}

func TestWiFi_ListAPClients_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	wifi := NewWiFi(client)
	testComponentError(t, "ListAPClients", func() error {
		_, err := wifi.ListAPClients(context.Background())
		return err
	})
}

func TestWiFi_ListAPClients_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	wifi := NewWiFi(client)
	testComponentInvalidJSON(t, "ListAPClients", func() error {
		_, err := wifi.ListAPClients(context.Background())
		return err
	})
}

func TestWiFiConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config WiFiConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "station config with password",
			config: WiFiConfig{
				STA: &WiFiStationConfig{
					SSID:   ptr("TestNetwork"),
					Pass:   ptr("secret123"),
					Enable: ptr(true),
				},
			},
			check: func(t *testing.T, data map[string]any) {
				sta, ok := data["sta"].(map[string]any)
				if !ok {
					t.Fatalf("sta type assertion failed")
				}
				ssid, ok := sta["ssid"].(string)
				if !ok || ssid != "TestNetwork" {
					t.Errorf("sta.ssid = %v, want TestNetwork", sta["ssid"])
				}
				pass, ok := sta["pass"].(string)
				if !ok || pass != "secret123" {
					t.Errorf("sta.pass = %v, want secret123", sta["pass"])
				}
			},
		},
		{
			name: "ap with range extender",
			config: WiFiConfig{
				AP: &WiFiAPConfig{
					SSID:   ptr("DeviceAP"),
					Enable: ptr(true),
					RangeExtender: &WiFiAPRangeExtenderConfig{
						Enable: ptr(true),
					},
				},
			},
			check: func(t *testing.T, data map[string]any) {
				ap, ok := data["ap"].(map[string]any)
				if !ok {
					t.Fatalf("ap type assertion failed")
				}
				re, ok := ap["range_extender"].(map[string]any)
				if !ok {
					t.Fatalf("range_extender type assertion failed")
				}
				enable, ok := re["enable"].(bool)
				if !ok || enable != true {
					t.Errorf("range_extender.enable = %v, want true", re["enable"])
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

func TestWiFiStationConfig_StaticIP(t *testing.T) {
	config := WiFiStationConfig{
		SSID:       ptr("StaticNetwork"),
		Pass:       ptr("password"),
		Enable:     ptr(true),
		IPv4Mode:   ptr("static"),
		IP:         ptr("192.168.1.100"),
		Netmask:    ptr("255.255.255.0"),
		GW:         ptr("192.168.1.1"),
		Nameserver: ptr("8.8.8.8"),
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	ipv4mode, ok := parsed["ipv4mode"].(string)
	if !ok || ipv4mode != "static" {
		t.Errorf("ipv4mode = %v, want static", parsed["ipv4mode"])
	}
	ip, ok := parsed["ip"].(string)
	if !ok || ip != "192.168.1.100" {
		t.Errorf("ip = %v, want 192.168.1.100", parsed["ip"])
	}
	netmask, ok := parsed["netmask"].(string)
	if !ok || netmask != "255.255.255.0" {
		t.Errorf("netmask = %v, want 255.255.255.0", parsed["netmask"])
	}
	gw, ok := parsed["gw"].(string)
	if !ok || gw != "192.168.1.1" {
		t.Errorf("gw = %v, want 192.168.1.1", parsed["gw"])
	}
	nameserver, ok := parsed["nameserver"].(string)
	if !ok || nameserver != "8.8.8.8" {
		t.Errorf("nameserver = %v, want 8.8.8.8", parsed["nameserver"])
	}
}

func TestWiFiScanResult_AllAuthTypes(t *testing.T) {
	authTypes := []string{
		"open", "wep", "wpa_psk", "wpa2_psk",
		"wpa_wpa2_psk", "wpa2_enterprise", "wpa3_psk",
	}

	for _, auth := range authTypes {
		t.Run(auth, func(t *testing.T) {
			jsonStr := `{"ssid": "TestNetwork", "auth": "` + auth + `"}`
			var result WiFiScanResult
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if result.Auth == nil || *result.Auth != auth {
				t.Errorf("Auth = %v, want %v", result.Auth, auth)
			}
		})
	}
}

func TestWiFi_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"status": "got ip"}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	wifi := NewWiFi(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := wifi.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
