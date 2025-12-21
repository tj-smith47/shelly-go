package provisioning

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

// mockTransport implements transport.Transport for testing.
type mockTransport struct {
	callFunc  func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error)
	closeFunc func() error
}

func (m *mockTransport) Call(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
	if m.callFunc != nil {
		return m.callFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockTransport) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// jsonrpcResponse wraps a result in a JSON-RPC response envelope.
func jsonrpcResponse(result string) (json.RawMessage, error) {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  json.RawMessage(result),
	}
	return json.Marshal(response)
}

var errTest = errors.New("test error")

func TestNew(t *testing.T) {
	transport := &mockTransport{}
	client := rpc.NewClient(transport)
	prov := New(client)

	if prov == nil {
		t.Fatal("New returned nil")
	}
	if prov.client != client {
		t.Error("client not set correctly")
	}
}

func TestProvisioner_ConfigureWiFi(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "WiFi.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.ConfigureWiFi(context.Background(), &WiFiConfig{
		SSID:     "TestNetwork",
		Password: "TestPassword",
	})
	if err != nil {
		t.Errorf("ConfigureWiFi() error = %v", err)
	}
}

func TestProvisioner_ConfigureWiFi_NoSSID(t *testing.T) {
	transport := &mockTransport{}
	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.ConfigureWiFi(context.Background(), &WiFiConfig{})
	if !errors.Is(err, ErrNoSSID) {
		t.Errorf("ConfigureWiFi() error = %v, want ErrNoSSID", err)
	}
}

func TestProvisioner_ConfigureWiFi_StaticIP(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.ConfigureWiFi(context.Background(), &WiFiConfig{
		SSID:       "TestNetwork",
		Password:   "TestPassword",
		StaticIP:   "static",
		IP:         "192.168.1.100",
		Netmask:    "255.255.255.0",
		Gateway:    "192.168.1.1",
		Nameserver: "8.8.8.8",
	})
	if err != nil {
		t.Errorf("ConfigureWiFi() error = %v", err)
	}
}

func TestProvisioner_ConfigureWiFi_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.ConfigureWiFi(context.Background(), &WiFiConfig{SSID: "Test"})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_ConfigureAP(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "WiFi.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	enable := true
	rangeExt := true
	err := prov.ConfigureAP(context.Background(), &APConfig{
		Enable:        &enable,
		SSID:          "MyAP",
		Password:      "APPassword",
		RangeExtender: &rangeExt,
	})
	if err != nil {
		t.Errorf("ConfigureAP() error = %v", err)
	}
}

func TestProvisioner_ConfigureAP_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	enable := false
	err := prov.ConfigureAP(context.Background(), &APConfig{Enable: &enable})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_SetAuth(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Shelly.SetAuth" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	enable := true
	err := prov.SetAuth(context.Background(), &AuthConfig{
		Enable:   &enable,
		User:     "admin",
		Password: "secret",
	})
	if err != nil {
		t.Errorf("SetAuth() error = %v", err)
	}
}

func TestProvisioner_SetAuth_Disable(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	enable := false
	err := prov.SetAuth(context.Background(), &AuthConfig{Enable: &enable})
	if err != nil {
		t.Errorf("SetAuth() error = %v", err)
	}
}

func TestProvisioner_SetAuth_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.SetAuth(context.Background(), &AuthConfig{})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_SetDeviceName(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Sys.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.SetDeviceName(context.Background(), "Kitchen Light")
	if err != nil {
		t.Errorf("SetDeviceName() error = %v", err)
	}
}

func TestProvisioner_SetDeviceName_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.SetDeviceName(context.Background(), "Test")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_SetTimezone(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Sys.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.SetTimezone(context.Background(), "America/New_York")
	if err != nil {
		t.Errorf("SetTimezone() error = %v", err)
	}
}

func TestProvisioner_SetTimezone_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.SetTimezone(context.Background(), "UTC")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_SetLocation(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Sys.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.SetLocation(context.Background(), 40.7128, -74.0060)
	if err != nil {
		t.Errorf("SetLocation() error = %v", err)
	}
}

func TestProvisioner_SetLocation_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.SetLocation(context.Background(), 0, 0)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_ConfigureCloud(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Cloud.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	enable := true
	err := prov.ConfigureCloud(context.Background(), &CloudConfig{
		Enable: &enable,
		Server: "custom.shelly.cloud",
	})
	if err != nil {
		t.Errorf("ConfigureCloud() error = %v", err)
	}
}

func TestProvisioner_ConfigureCloud_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.ConfigureCloud(context.Background(), &CloudConfig{})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_GetDeviceInfo(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Shelly.GetDeviceInfo" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"id":"shellyplus1-123456","model":"SNSW-001X16EU","gen":2,"ver":"1.0.0","mac":"123456789ABC"}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	info, err := prov.GetDeviceInfo(context.Background())
	if err != nil {
		t.Errorf("GetDeviceInfo() error = %v", err)
		return
	}

	if info.ID != "shellyplus1-123456" {
		t.Errorf("ID = %s, want shellyplus1-123456", info.ID)
	}
	if info.Model != "SNSW-001X16EU" {
		t.Errorf("Model = %s, want SNSW-001X16EU", info.Model)
	}
	if info.Generation != 2 {
		t.Errorf("Generation = %d, want 2", info.Generation)
	}
}

func TestProvisioner_GetDeviceInfo_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	_, err := prov.GetDeviceInfo(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_GetDeviceInfo_InvalidJSON(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return jsonrpcResponse(`{invalid json}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	_, err := prov.GetDeviceInfo(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestProvisioner_GetWiFiStatus(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "WiFi.GetStatus" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"sta_ip":"192.168.1.100","status":"got ip","ssid":"TestNetwork","rssi":-65}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	status, err := prov.GetWiFiStatus(context.Background())
	if err != nil {
		t.Errorf("GetWiFiStatus() error = %v", err)
		return
	}

	if status.StaIP != "192.168.1.100" {
		t.Errorf("StaIP = %s, want 192.168.1.100", status.StaIP)
	}
	if status.SSID != "TestNetwork" {
		t.Errorf("SSID = %s, want TestNetwork", status.SSID)
	}
	if status.RSSI != -65 {
		t.Errorf("RSSI = %d, want -65", status.RSSI)
	}
}

func TestProvisioner_GetWiFiStatus_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	_, err := prov.GetWiFiStatus(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_GetWiFiStatus_InvalidJSON(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return jsonrpcResponse(`{invalid json}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	_, err := prov.GetWiFiStatus(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestProvisioner_IsConnected(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{name: "connected", result: `{"sta_ip":"192.168.1.100"}`, want: true},
		{name: "not connected", result: `{"sta_ip":""}`, want: false},
		{name: "no sta_ip", result: `{}`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
					return jsonrpcResponse(tt.result)
				},
			}

			client := rpc.NewClient(transport)
			prov := New(client)

			got, err := prov.IsConnected(context.Background())
			if err != nil {
				t.Errorf("IsConnected() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("IsConnected() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvisioner_IsConnected_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	_, err := prov.IsConnected(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_WaitForConnection_Success(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			callCount++
			if callCount < 2 {
				return jsonrpcResponse(`{"sta_ip":""}`)
			}
			return jsonrpcResponse(`{"sta_ip":"192.168.1.100"}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	ip, err := prov.WaitForConnection(context.Background(), 10*time.Second)
	if err != nil {
		t.Errorf("WaitForConnection() error = %v", err)
		return
	}
	if ip != "192.168.1.100" {
		t.Errorf("WaitForConnection() = %s, want 192.168.1.100", ip)
	}
}

func TestProvisioner_WaitForConnection_Timeout(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return jsonrpcResponse(`{"sta_ip":""}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	_, err := prov.WaitForConnection(context.Background(), 100*time.Millisecond)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("WaitForConnection() error = %v, want ErrTimeout", err)
	}
}

func TestProvisioner_WaitForConnection_ContextCanceled(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return jsonrpcResponse(`{"sta_ip":""}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := prov.WaitForConnection(ctx, 10*time.Second)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestProvisioner_Reboot(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Shelly.Reboot" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.Reboot(context.Background())
	if err != nil {
		t.Errorf("Reboot() error = %v", err)
	}
}

func TestProvisioner_Reboot_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.Reboot(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_DisableBLE(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "BLE.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.DisableBLE(context.Background())
	if err != nil {
		t.Errorf("DisableBLE() error = %v", err)
	}
}

func TestProvisioner_EnableBLE(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "BLE.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.EnableBLE(context.Background())
	if err != nil {
		t.Errorf("EnableBLE() error = %v", err)
	}
}

func TestProvisioner_FactoryReset(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Shelly.FactoryReset" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.FactoryReset(context.Background())
	if err != nil {
		t.Errorf("FactoryReset() error = %v", err)
	}
}

func TestProvisioner_FactoryReset_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	err := prov.FactoryReset(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvisioner_Provision(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456","model":"SNSW-001X16EU","gen":2}`)
			case "WiFi.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "Sys.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "Cloud.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "Shelly.SetAuth":
				return jsonrpcResponse(`null`)
			case "WiFi.GetStatus":
				return jsonrpcResponse(`{"sta_ip":"192.168.1.100"}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	enable := true
	config := &DeviceConfig{
		WiFi: &WiFiConfig{
			SSID:     "TestNetwork",
			Password: "TestPassword",
		},
		DeviceName: "Kitchen Light",
		Timezone:   "America/New_York",
		Location: &Location{
			Lat: 40.7128,
			Lon: -74.0060,
		},
		Cloud: &CloudConfig{
			Enable: &enable,
		},
		Auth: &AuthConfig{
			Enable:   &enable,
			User:     "admin",
			Password: "secret",
		},
	}

	opts := &ProvisionOptions{
		WaitForConnection: true,
		ConnectionTimeout: 1,
		VerifyConnection:  true,
	}

	result, err := prov.Provision(context.Background(), config, opts)
	if err != nil {
		t.Errorf("Provision() error = %v", err)
		return
	}

	if !result.Success {
		t.Errorf("Provision() Success = false, want true")
	}
	if result.NewAddress != "192.168.1.100" {
		t.Errorf("Provision() NewAddress = %s, want 192.168.1.100", result.NewAddress)
	}
	if result.DeviceInfo == nil {
		t.Error("Provision() DeviceInfo = nil")
	}
}

func TestProvisioner_Provision_DefaultOptions(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456"}`)
			case "WiFi.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "WiFi.GetStatus":
				return jsonrpcResponse(`{"sta_ip":"192.168.1.100"}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	config := &DeviceConfig{
		WiFi: &WiFiConfig{SSID: "Test"},
	}

	// Pass nil options to use defaults
	result, err := prov.Provision(context.Background(), config, nil)
	if err != nil {
		t.Errorf("Provision() error = %v", err)
		return
	}

	if !result.Success {
		t.Errorf("Provision() Success = false, want true")
	}
}

func TestProvisioner_Provision_GetDeviceInfoError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method == "Shelly.GetDeviceInfo" {
				return nil, errTest
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	config := &DeviceConfig{
		WiFi: &WiFiConfig{SSID: "Test"},
	}

	result, err := prov.Provision(context.Background(), config, nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result.Success {
		t.Error("Provision() Success = true, want false")
	}
}

func TestProvisioner_Provision_WiFiError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456"}`)
			case "WiFi.SetConfig":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	config := &DeviceConfig{
		WiFi: &WiFiConfig{SSID: "Test"},
	}

	result, err := prov.Provision(context.Background(), config, &ProvisionOptions{WaitForConnection: false})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result.Success {
		t.Error("Provision() Success = true, want false")
	}
}

func TestDefaultProvisionOptions(t *testing.T) {
	opts := DefaultProvisionOptions()

	if !opts.WaitForConnection {
		t.Error("WaitForConnection should be true by default")
	}
	if opts.ConnectionTimeout != 30 {
		t.Errorf("ConnectionTimeout = %d, want 30", opts.ConnectionTimeout)
	}
	if !opts.VerifyConnection {
		t.Error("VerifyConnection should be true by default")
	}
	if opts.DisableAP {
		t.Error("DisableAP should be false by default")
	}
	if opts.DisableBLE {
		t.Error("DisableBLE should be false by default")
	}
}

func TestDefaultAPAddress(t *testing.T) {
	if DefaultAPAddress != "192.168.33.1" {
		t.Errorf("DefaultAPAddress = %s, want 192.168.33.1", DefaultAPAddress)
	}
}

// BLE Provisioner Tests

func TestNewBLEProvisioner(t *testing.T) {
	b := NewBLEProvisioner()
	if b == nil {
		t.Fatal("NewBLEProvisioner returned nil")
	}
	if b.ScanTimeout != 10*time.Second {
		t.Errorf("ScanTimeout = %v, want 10s", b.ScanTimeout)
	}
	if b.ConnectTimeout != 30*time.Second {
		t.Errorf("ConnectTimeout = %v, want 30s", b.ConnectTimeout)
	}
	if b.devices == nil {
		t.Error("devices map is nil")
	}
}

func TestBLEProvisioner_AddDiscoveredDevice(t *testing.T) {
	b := NewBLEProvisioner()

	device := &BLEDevice{
		Name:     "ShellyPlus1-123456",
		Address:  "AA:BB:CC:DD:EE:FF",
		RSSI:     -65,
		IsShelly: true,
	}

	b.AddDiscoveredDevice(device)

	if b.DeviceCount() != 1 {
		t.Errorf("DeviceCount() = %d, want 1", b.DeviceCount())
	}

	// Verify device was stored correctly
	got, err := b.GetDevice("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Errorf("GetDevice() error = %v", err)
		return
	}
	if got.Name != "ShellyPlus1-123456" {
		t.Errorf("Name = %s, want ShellyPlus1-123456", got.Name)
	}
	if got.DiscoveredAt.IsZero() {
		t.Error("DiscoveredAt should be set")
	}
}

func TestBLEProvisioner_GetDevice_NotFound(t *testing.T) {
	b := NewBLEProvisioner()

	_, err := b.GetDevice("AA:BB:CC:DD:EE:FF")
	if !errors.Is(err, ErrBLEDeviceNotFound) {
		t.Errorf("GetDevice() error = %v, want ErrBLEDeviceNotFound", err)
	}
}

func TestBLEProvisioner_DiscoverBLEDevices(t *testing.T) {
	b := NewBLEProvisioner()

	// Add some devices
	b.AddDiscoveredDevice(&BLEDevice{
		Name:     "ShellyPlus1-123456",
		Address:  "AA:BB:CC:DD:EE:FF",
		IsShelly: true,
	})
	b.AddDiscoveredDevice(&BLEDevice{
		Name:     "ShellyPro4PM-789012",
		Address:  "11:22:33:44:55:66",
		IsShelly: true,
	})
	b.AddDiscoveredDevice(&BLEDevice{
		Name:     "RandomDevice",
		Address:  "99:88:77:66:55:44",
		IsShelly: false,
	})

	devices, err := b.DiscoverBLEDevices(context.Background())
	if err != nil {
		t.Errorf("DiscoverBLEDevices() error = %v", err)
		return
	}

	// Should only return Shelly devices
	if len(devices) != 2 {
		t.Errorf("len(devices) = %d, want 2", len(devices))
	}
}

func TestBLEProvisioner_ClearDevices(t *testing.T) {
	b := NewBLEProvisioner()

	b.AddDiscoveredDevice(&BLEDevice{
		Name:     "ShellyPlus1-123456",
		Address:  "AA:BB:CC:DD:EE:FF",
		IsShelly: true,
	})

	if b.DeviceCount() != 1 {
		t.Errorf("DeviceCount() = %d, want 1", b.DeviceCount())
	}

	b.ClearDevices()

	if b.DeviceCount() != 0 {
		t.Errorf("DeviceCount() after Clear() = %d, want 0", b.DeviceCount())
	}
}

func TestIsShellyDevice(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"ShellyPlus1-123456", true},
		{"ShellyPro4PM-ABCD", true},
		{"Shelly1-XYZ", true},
		{"shellyplus1-123456", false}, // Case sensitive
		{"RandomDevice", false},
		{"Shell", false}, // Too short
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsShellyDevice(tt.name); got != tt.want {
				t.Errorf("IsShellyDevice(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestParseBLEDeviceName(t *testing.T) {
	tests := []struct {
		name      string
		wantModel string
		wantMAC   string
	}{
		{"ShellyPlus1-123456", "ShellyPlus1", "123456"},
		{"ShellyPro4PM-ABCD", "ShellyPro4PM", "ABCD"},
		{"ShellyPlugS-XYZ123", "ShellyPlugS", "XYZ123"},
		{"Shelly1", "Shelly1", ""},
		{"RandomDevice", "", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, mac := ParseBLEDeviceName(tt.name)
			if model != tt.wantModel {
				t.Errorf("model = %q, want %q", model, tt.wantModel)
			}
			if mac != tt.wantMAC {
				t.Errorf("mac = %q, want %q", mac, tt.wantMAC)
			}
		})
	}
}

func TestBLEProvisioner_ProvisionViaBLE(t *testing.T) {
	b := NewBLEProvisioner()

	// Add a device first
	b.AddDiscoveredDevice(&BLEDevice{
		Name:     "ShellyPlus1-123456",
		Address:  "AA:BB:CC:DD:EE:FF",
		IsShelly: true,
	})

	enableCloud := true
	config := &BLEProvisionConfig{
		WiFi: &WiFiConfig{
			SSID:     "TestNetwork",
			Password: "TestPassword",
		},
		DeviceName:  "Kitchen Light",
		Timezone:    "America/New_York",
		EnableCloud: &enableCloud,
	}

	result, err := b.ProvisionViaBLE(context.Background(), "AA:BB:CC:DD:EE:FF", config)
	if err != nil {
		t.Errorf("ProvisionViaBLE() error = %v", err)
		return
	}

	if !result.Success {
		t.Error("ProvisionViaBLE() Success = false, want true")
	}
	if result.Device == nil {
		t.Error("ProvisionViaBLE() Device = nil")
	}

	// Should have 4 commands: WiFi, DeviceName, Timezone, Cloud
	if len(result.Commands) != 4 {
		t.Errorf("len(Commands) = %d, want 4", len(result.Commands))
	}
}

func TestBLEProvisioner_ProvisionViaBLE_DeviceNotFound(t *testing.T) {
	b := NewBLEProvisioner()

	config := &BLEProvisionConfig{
		WiFi: &WiFiConfig{SSID: "Test"},
	}

	_, err := b.ProvisionViaBLE(context.Background(), "AA:BB:CC:DD:EE:FF", config)
	if !errors.Is(err, ErrBLEDeviceNotFound) {
		t.Errorf("ProvisionViaBLE() error = %v, want ErrBLEDeviceNotFound", err)
	}
}

func TestBLEProvisioner_ProvisionViaBLE_StaticIP(t *testing.T) {
	b := NewBLEProvisioner()

	b.AddDiscoveredDevice(&BLEDevice{
		Name:     "ShellyPlus1-123456",
		Address:  "AA:BB:CC:DD:EE:FF",
		IsShelly: true,
	})

	config := &BLEProvisionConfig{
		WiFi: &WiFiConfig{
			SSID:     "TestNetwork",
			Password: "TestPassword",
			StaticIP: "static",
			IP:       "192.168.1.100",
			Netmask:  "255.255.255.0",
			Gateway:  "192.168.1.1",
		},
	}

	result, err := b.ProvisionViaBLE(context.Background(), "AA:BB:CC:DD:EE:FF", config)
	if err != nil {
		t.Errorf("ProvisionViaBLE() error = %v", err)
		return
	}

	if !result.Success {
		t.Error("ProvisionViaBLE() Success = false, want true")
	}
	if len(result.Commands) != 1 {
		t.Errorf("len(Commands) = %d, want 1", len(result.Commands))
	}
}

func TestBLERPCCommand_ToJSON(t *testing.T) {
	cmd := BLERPCCommand{
		Method: "WiFi.SetConfig",
		Params: map[string]any{
			"config": map[string]any{
				"sta": map[string]any{
					"ssid": "Test",
				},
			},
		},
	}

	data, err := cmd.ToJSON(1)
	if err != nil {
		t.Errorf("ToJSON() error = %v", err)
		return
	}

	var parsed map[string]any
	if err = json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	if parsed["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", parsed["jsonrpc"])
	}
	if parsed["method"] != "WiFi.SetConfig" {
		t.Errorf("method = %v, want WiFi.SetConfig", parsed["method"])
	}
	if id, ok := parsed["id"].(float64); !ok || id != 1 {
		t.Errorf("id = %v, want 1", parsed["id"])
	}
}

func TestBLERPCCommand_ToJSON_NoParams(t *testing.T) {
	cmd := BLERPCCommand{
		Method: "Shelly.GetDeviceInfo",
	}

	data, err := cmd.ToJSON(1)
	if err != nil {
		t.Errorf("ToJSON() error = %v", err)
		return
	}

	var parsed map[string]any
	if err = json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	if _, ok := parsed["params"]; ok {
		t.Error("params should not be present when nil")
	}
}

func TestBLEProvisionResult_Duration(t *testing.T) {
	start := time.Now()
	result := &BLEProvisionResult{
		StartedAt:   start,
		CompletedAt: start.Add(5 * time.Second),
	}

	if result.Duration() != 5*time.Second {
		t.Errorf("Duration() = %v, want 5s", result.Duration())
	}
}

// Profile Registry Tests

func TestNewProfileRegistry(t *testing.T) {
	r := NewProfileRegistry()
	if r == nil {
		t.Fatal("NewProfileRegistry returned nil")
	}
	if r.profiles == nil {
		t.Error("profiles map is nil")
	}
}

func TestProfileRegistry_Register(t *testing.T) {
	r := NewProfileRegistry()

	profile := &Profile{
		Name:        "test-profile",
		Description: "Test profile",
		Config: &DeviceConfig{
			Timezone: "UTC",
		},
	}

	err := r.Register(profile)
	if err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	if r.Count() != 1 {
		t.Errorf("Count() = %d, want 1", r.Count())
	}

	// Verify timestamps were set
	got, err := r.Get("test-profile")
	if err != nil {
		t.Errorf("Get() error = %v", err)
		return
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestProfileRegistry_Register_EmptyName(t *testing.T) {
	r := NewProfileRegistry()

	profile := &Profile{
		Description: "Test profile",
	}

	err := r.Register(profile)
	if !errors.Is(err, ErrInvalidProfile) {
		t.Errorf("Register() error = %v, want ErrInvalidProfile", err)
	}
}

func TestProfileRegistry_Register_Update(t *testing.T) {
	r := NewProfileRegistry()

	// Register initial profile
	profile1 := &Profile{
		Name:        "test-profile",
		Description: "Original",
	}
	if err := r.Register(profile1); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	// Wait a moment to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Update with new profile
	profile2 := &Profile{
		Name:        "test-profile",
		Description: "Updated",
	}
	if err := r.Register(profile2); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	got, err := r.Get("test-profile")
	if err != nil {
		t.Errorf("Get() error = %v", err)
		return
	}
	if got.Description != "Updated" {
		t.Errorf("Description = %s, want Updated", got.Description)
	}

	// CreatedAt should be preserved from original
	if got.CreatedAt != profile1.CreatedAt {
		t.Error("CreatedAt should be preserved on update")
	}
}

func TestProfileRegistry_Get(t *testing.T) {
	r := NewProfileRegistry()

	profile := &Profile{
		Name:        "test-profile",
		Description: "Test",
	}
	if err := r.Register(profile); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	got, err := r.Get("test-profile")
	if err != nil {
		t.Errorf("Get() error = %v", err)
		return
	}
	if got.Name != "test-profile" {
		t.Errorf("Name = %s, want test-profile", got.Name)
	}
}

func TestProfileRegistry_Get_NotFound(t *testing.T) {
	r := NewProfileRegistry()

	_, err := r.Get("nonexistent")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("Get() error = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileRegistry_Delete(t *testing.T) {
	r := NewProfileRegistry()

	profile := &Profile{Name: "test-profile"}
	if err := r.Register(profile); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	err := r.Delete("test-profile")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	if r.Count() != 0 {
		t.Errorf("Count() = %d, want 0", r.Count())
	}
}

func TestProfileRegistry_Delete_NotFound(t *testing.T) {
	r := NewProfileRegistry()

	err := r.Delete("nonexistent")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("Delete() error = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileRegistry_List(t *testing.T) {
	r := NewProfileRegistry()

	if err := r.Register(&Profile{Name: "profile1"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}
	if err := r.Register(&Profile{Name: "profile2"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}
	if err := r.Register(&Profile{Name: "profile3"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	profiles := r.List()
	if len(profiles) != 3 {
		t.Errorf("len(profiles) = %d, want 3", len(profiles))
	}
}

func TestProfileRegistry_ListByTag(t *testing.T) {
	r := NewProfileRegistry()

	if err := r.Register(&Profile{
		Name: "home1",
		Tags: map[string]string{"type": "home"},
	}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}
	if err := r.Register(&Profile{
		Name: "home2",
		Tags: map[string]string{"type": "home"},
	}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}
	if err := r.Register(&Profile{
		Name: "enterprise1",
		Tags: map[string]string{"type": "enterprise"},
	}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	homeProfiles := r.ListByTag("type", "home")
	if len(homeProfiles) != 2 {
		t.Errorf("len(homeProfiles) = %d, want 2", len(homeProfiles))
	}

	enterpriseProfiles := r.ListByTag("type", "enterprise")
	if len(enterpriseProfiles) != 1 {
		t.Errorf("len(enterpriseProfiles) = %d, want 1", len(enterpriseProfiles))
	}

	unknownProfiles := r.ListByTag("type", "unknown")
	if len(unknownProfiles) != 0 {
		t.Errorf("len(unknownProfiles) = %d, want 0", len(unknownProfiles))
	}
}

func TestProfileRegistry_Clear(t *testing.T) {
	r := NewProfileRegistry()

	if err := r.Register(&Profile{Name: "profile1"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}
	if err := r.Register(&Profile{Name: "profile2"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	r.Clear()

	if r.Count() != 0 {
		t.Errorf("Count() = %d, want 0", r.Count())
	}
}

func TestProfileRegistry_ExportImport(t *testing.T) {
	r := NewProfileRegistry()

	enableCloud := true
	if err := r.Register(&Profile{
		Name:        "profile1",
		Description: "First profile",
		Config: &DeviceConfig{
			Timezone: "UTC",
			Cloud: &CloudConfig{
				Enable: &enableCloud,
			},
		},
		Tags: map[string]string{"env": "test"},
	}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}
	if err := r.Register(&Profile{
		Name:        "profile2",
		Description: "Second profile",
	}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	// Export
	data, err := r.Export()
	if err != nil {
		t.Errorf("Export() error = %v", err)
		return
	}

	// Create new registry and import
	r2 := NewProfileRegistry()
	err = r2.Import(data)
	if err != nil {
		t.Errorf("Import() error = %v", err)
		return
	}

	if r2.Count() != 2 {
		t.Errorf("Count() after import = %d, want 2", r2.Count())
	}

	// Verify profile data was preserved
	got, err := r2.Get("profile1")
	if err != nil {
		t.Errorf("Get() error = %v", err)
		return
	}
	if got.Description != "First profile" {
		t.Errorf("Description = %s, want First profile", got.Description)
	}
	if got.Tags["env"] != "test" {
		t.Errorf("Tags[env] = %s, want test", got.Tags["env"])
	}
}

func TestProfileRegistry_Import_InvalidJSON(t *testing.T) {
	r := NewProfileRegistry()

	err := r.Import([]byte("invalid json"))
	if err == nil {
		t.Error("Import() should error on invalid JSON")
	}
}

func TestStandardProfiles(t *testing.T) {
	profiles := StandardProfiles()

	if len(profiles) != 4 {
		t.Errorf("len(profiles) = %d, want 4", len(profiles))
	}

	// Verify profile names
	names := make(map[string]bool)
	for _, p := range profiles {
		names[p.Name] = true
	}

	expectedNames := []string{"home-basic", "home-advanced", "enterprise-secure", "iot-minimal"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("missing expected profile: %s", name)
		}
	}
}

// Bulk Provisioner Tests

func TestNewBulkProvisioner(t *testing.T) {
	factory := func(address string) (*rpc.Client, error) {
		return nil, nil
	}

	b := NewBulkProvisioner(factory)
	if b == nil {
		t.Fatal("NewBulkProvisioner returned nil")
	}
	if b.Concurrency != 3 {
		t.Errorf("Concurrency = %d, want 3", b.Concurrency)
	}
	if b.RetryCount != 2 {
		t.Errorf("RetryCount = %d, want 2", b.RetryCount)
	}
	if b.RetryDelay != 5*time.Second {
		t.Errorf("RetryDelay = %v, want 5s", b.RetryDelay)
	}
}

func TestBulkProvisioner_ProvisionBulk_NoFactory(t *testing.T) {
	b := &BulkProvisioner{}

	_, err := b.ProvisionBulk(context.Background(), nil, nil, nil)
	if err == nil {
		t.Error("ProvisionBulk() should error with no client factory")
	}
}

func TestBulkProvisioner_ProvisionBulk_EmptyTargets(t *testing.T) {
	factory := func(address string) (*rpc.Client, error) {
		return nil, nil
	}

	b := NewBulkProvisioner(factory)

	result, err := b.ProvisionBulk(context.Background(), []*BulkProvisionTarget{}, nil, nil)
	if err != nil {
		t.Errorf("ProvisionBulk() error = %v", err)
		return
	}

	if result.TotalDevices != 0 {
		t.Errorf("TotalDevices = %d, want 0", result.TotalDevices)
	}
	if result.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0", result.SuccessCount)
	}
}

func TestBulkProvisioner_ProvisionBulk_NoConfig(t *testing.T) {
	factory := func(address string) (*rpc.Client, error) {
		transport := &mockTransport{
			callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
				return jsonrpcResponse(`{"id":"test"}`)
			},
		}
		return rpc.NewClient(transport), nil
	}

	b := NewBulkProvisioner(factory)

	targets := []*BulkProvisionTarget{
		{Address: "192.168.1.100"},
	}

	result, err := b.ProvisionBulk(context.Background(), targets, nil, nil)
	if err != nil {
		t.Errorf("ProvisionBulk() error = %v", err)
		return
	}

	// Should fail because no config provided
	if result.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", result.FailureCount)
	}
}

func TestBulkProvisioner_ProvisionBulk_WithProfile(t *testing.T) {
	callCount := 0
	factory := func(address string) (*rpc.Client, error) {
		transport := &mockTransport{
			callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
				callCount++
				switch method {
				case "Shelly.GetDeviceInfo":
					return jsonrpcResponse(`{"id":"test"}`)
				case "WiFi.SetConfig":
					return jsonrpcResponse(`{"restart_required":false}`)
				case "WiFi.GetStatus":
					return jsonrpcResponse(`{"sta_ip":"192.168.1.100"}`)
				default:
					return jsonrpcResponse(`null`)
				}
			},
		}
		return rpc.NewClient(transport), nil
	}

	b := NewBulkProvisioner(factory)
	b.Concurrency = 1
	b.RetryCount = 0

	// Set up profile registry
	registry := NewProfileRegistry()
	if err := registry.Register(&Profile{
		Name: "test-profile",
		Config: &DeviceConfig{
			WiFi: &WiFiConfig{SSID: "TestNet"},
		},
		Options: &ProvisionOptions{
			WaitForConnection: true,
			ConnectionTimeout: 1,
		},
	}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	targets := []*BulkProvisionTarget{
		{Address: "192.168.1.100", ProfileName: "test-profile"},
	}

	result, err := b.ProvisionBulk(context.Background(), targets, registry, nil)
	if err != nil {
		t.Errorf("ProvisionBulk() error = %v", err)
		return
	}

	if result.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", result.SuccessCount)
	}
}

func TestBulkProvisioner_ProvisionBulk_WithDirectConfig(t *testing.T) {
	factory := func(address string) (*rpc.Client, error) {
		transport := &mockTransport{
			callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
				switch method {
				case "Shelly.GetDeviceInfo":
					return jsonrpcResponse(`{"id":"test"}`)
				case "WiFi.SetConfig":
					return jsonrpcResponse(`{"restart_required":false}`)
				case "WiFi.GetStatus":
					return jsonrpcResponse(`{"sta_ip":"192.168.1.100"}`)
				default:
					return jsonrpcResponse(`null`)
				}
			},
		}
		return rpc.NewClient(transport), nil
	}

	b := NewBulkProvisioner(factory)
	b.Concurrency = 1
	b.RetryCount = 0

	targets := []*BulkProvisionTarget{
		{
			Address: "192.168.1.100",
			Config: &DeviceConfig{
				WiFi: &WiFiConfig{SSID: "DirectConfig"},
			},
		},
	}

	opts := &ProvisionOptions{
		WaitForConnection: true,
		ConnectionTimeout: 1,
	}

	result, err := b.ProvisionBulk(context.Background(), targets, nil, opts)
	if err != nil {
		t.Errorf("ProvisionBulk() error = %v", err)
		return
	}

	if result.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", result.SuccessCount)
	}
}

func TestBulkProvisioner_ProvisionBulk_MultipleDevices(t *testing.T) {
	factory := func(address string) (*rpc.Client, error) {
		transport := &mockTransport{
			callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
				switch method {
				case "Shelly.GetDeviceInfo":
					return jsonrpcResponse(`{"id":"test"}`)
				case "WiFi.SetConfig":
					return jsonrpcResponse(`{"restart_required":false}`)
				case "WiFi.GetStatus":
					return jsonrpcResponse(`{"sta_ip":"` + address + `"}`)
				default:
					return jsonrpcResponse(`null`)
				}
			},
		}
		return rpc.NewClient(transport), nil
	}

	b := NewBulkProvisioner(factory)
	b.Concurrency = 3
	b.RetryCount = 0

	config := &DeviceConfig{
		WiFi: &WiFiConfig{SSID: "TestNet"},
	}

	targets := []*BulkProvisionTarget{
		{Address: "192.168.1.100", Config: config},
		{Address: "192.168.1.101", Config: config},
		{Address: "192.168.1.102", Config: config},
	}

	opts := &ProvisionOptions{
		WaitForConnection: true,
		ConnectionTimeout: 1,
	}

	result, err := b.ProvisionBulk(context.Background(), targets, nil, opts)
	if err != nil {
		t.Errorf("ProvisionBulk() error = %v", err)
		return
	}

	if result.TotalDevices != 3 {
		t.Errorf("TotalDevices = %d, want 3", result.TotalDevices)
	}
	if result.SuccessCount != 3 {
		t.Errorf("SuccessCount = %d, want 3", result.SuccessCount)
	}
	if len(result.Results) != 3 {
		t.Errorf("len(Results) = %d, want 3", len(result.Results))
	}
}

func TestBulkProvisioner_ProvisionBulk_ClientFactoryError(t *testing.T) {
	factory := func(address string) (*rpc.Client, error) {
		return nil, errors.New("connection failed")
	}

	b := NewBulkProvisioner(factory)
	b.Concurrency = 1
	b.RetryCount = 0

	targets := []*BulkProvisionTarget{
		{
			Address: "192.168.1.100",
			Config:  &DeviceConfig{WiFi: &WiFiConfig{SSID: "Test"}},
		},
	}

	opts := &ProvisionOptions{WaitForConnection: false}

	result, err := b.ProvisionBulk(context.Background(), targets, nil, opts)
	if err != nil {
		t.Errorf("ProvisionBulk() error = %v", err)
		return
	}

	if result.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", result.FailureCount)
	}
}

func TestBulkProvisioner_ProvisionBulk_ContextCanceled(t *testing.T) {
	factory := func(address string) (*rpc.Client, error) {
		// Simulate slow connection
		time.Sleep(100 * time.Millisecond)
		return nil, errors.New("timeout")
	}

	b := NewBulkProvisioner(factory)
	b.Concurrency = 1
	b.RetryCount = 1
	b.RetryDelay = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	targets := []*BulkProvisionTarget{
		{
			Address: "192.168.1.100",
			Config:  &DeviceConfig{WiFi: &WiFiConfig{SSID: "Test"}},
		},
	}

	result, err := b.ProvisionBulk(ctx, targets, nil, nil)
	if err != nil {
		t.Errorf("ProvisionBulk() error = %v", err)
		return
	}

	// Should fail due to context cancellation
	if result.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", result.FailureCount)
	}
}

func TestBLEServiceUUIDs(t *testing.T) {
	// Just verify the constants are set
	if ShellyBLEServiceUUID == "" {
		t.Error("ShellyBLEServiceUUID is empty")
	}
	if ShellyBLERPCCharUUID == "" {
		t.Error("ShellyBLERPCCharUUID is empty")
	}
	if ShellyBLENotifyCharUUID == "" {
		t.Error("ShellyBLENotifyCharUUID is empty")
	}
}

func TestBulkProvisionResult_Duration(t *testing.T) {
	start := time.Now()
	result := &BulkProvisionResult{
		StartedAt:   start,
		CompletedAt: start.Add(10 * time.Second),
		Duration:    10 * time.Second,
	}

	if result.Duration != 10*time.Second {
		t.Errorf("Duration = %v, want 10s", result.Duration)
	}
}

func TestBLEProvisioner_Concurrency(t *testing.T) {
	b := NewBLEProvisioner()

	// Test concurrent access to device registry
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(i int) {
			b.AddDiscoveredDevice(&BLEDevice{
				Name:     "Device" + string(rune('0'+i)),
				Address:  "AA:BB:CC:DD:EE:" + string(rune('0'+i)) + string(rune('0'+i)),
				IsShelly: true,
			})
			// Ignore errors in concurrency test
			_, _ = b.DiscoverBLEDevices(context.Background())
			_ = b.DeviceCount()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 10 devices
	if b.DeviceCount() != 10 {
		t.Errorf("DeviceCount() = %d, want 10", b.DeviceCount())
	}
}

func TestProfileRegistry_Concurrency(t *testing.T) {
	r := NewProfileRegistry()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(i int) {
			// Ignore errors in concurrency test
			_ = r.Register(&Profile{
				Name: "profile" + string(rune('0'+i)),
			})
			_ = r.List()
			_ = r.Count()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 10 profiles
	if r.Count() != 10 {
		t.Errorf("Count() = %d, want 10", r.Count())
	}
}

// Test applyProvisioningConfig failure paths

func TestProvisioner_Provision_APError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456"}`)
			case "WiFi.SetConfig":
				// Check if this is AP config or WiFi config based on params
				paramsBytes := req.GetParams()
				if len(paramsBytes) > 50 { // AP config has more data
					return nil, errTest
				}
				return jsonrpcResponse(`{"restart_required":false}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	enable := true
	config := &DeviceConfig{
		AP: &APConfig{Enable: &enable, SSID: "TestAP", Password: "TestPass123"},
	}

	result, err := prov.Provision(context.Background(), config, &ProvisionOptions{WaitForConnection: false})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result.Success {
		t.Error("Provision() Success = true, want false")
	}
}

func TestProvisioner_Provision_DeviceNameError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456"}`)
			case "Sys.SetConfig":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	config := &DeviceConfig{
		DeviceName: "Test Device",
	}

	result, err := prov.Provision(context.Background(), config, &ProvisionOptions{WaitForConnection: false})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result.Success {
		t.Error("Provision() Success = true, want false")
	}
}

func TestProvisioner_Provision_TimezoneError(t *testing.T) {
	sysCallCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456"}`)
			case "Sys.SetConfig":
				sysCallCount++
				if sysCallCount == 2 { // Timezone is second Sys.SetConfig call
					return nil, errTest
				}
				return jsonrpcResponse(`{"restart_required":false}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	config := &DeviceConfig{
		DeviceName: "Test",
		Timezone:   "America/New_York",
	}

	result, err := prov.Provision(context.Background(), config, &ProvisionOptions{WaitForConnection: false})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result.Success {
		t.Error("Provision() Success = true, want false")
	}
}

func TestProvisioner_Provision_LocationError(t *testing.T) {
	sysCallCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456"}`)
			case "Sys.SetConfig":
				sysCallCount++
				if sysCallCount == 3 { // Location is third Sys.SetConfig call
					return nil, errTest
				}
				return jsonrpcResponse(`{"restart_required":false}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	config := &DeviceConfig{
		DeviceName: "Test",
		Timezone:   "UTC",
		Location:   &Location{Lat: 40.0, Lon: -74.0},
	}

	result, err := prov.Provision(context.Background(), config, &ProvisionOptions{WaitForConnection: false})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result.Success {
		t.Error("Provision() Success = true, want false")
	}
}

func TestProvisioner_Provision_CloudError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456"}`)
			case "Cloud.SetConfig":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	enable := true
	config := &DeviceConfig{
		Cloud: &CloudConfig{Enable: &enable},
	}

	result, err := prov.Provision(context.Background(), config, &ProvisionOptions{WaitForConnection: false})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result.Success {
		t.Error("Provision() Success = true, want false")
	}
}

func TestProvisioner_Provision_AuthError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456"}`)
			case "Shelly.SetAuth":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	enable := true
	config := &DeviceConfig{
		Auth: &AuthConfig{Enable: &enable, User: "admin", Password: "secret"},
	}

	result, err := prov.Provision(context.Background(), config, &ProvisionOptions{WaitForConnection: false})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result.Success {
		t.Error("Provision() Success = true, want false")
	}
}

// Test applyPostProvisioningOptions paths

func TestProvisioner_Provision_DisableAPError(t *testing.T) {
	wifiCallCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456"}`)
			case "WiFi.SetConfig":
				wifiCallCount++
				if wifiCallCount == 2 { // Second call is DisableAP
					return nil, errTest
				}
				return jsonrpcResponse(`{"restart_required":false}`)
			case "WiFi.GetStatus":
				return jsonrpcResponse(`{"sta_ip":"192.168.1.100"}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	config := &DeviceConfig{
		WiFi: &WiFiConfig{SSID: "Test"},
	}

	opts := &ProvisionOptions{
		WaitForConnection: true,
		ConnectionTimeout: 1,
		DisableAP:         true,
	}

	result, err := prov.Provision(context.Background(), config, opts)
	// Post-provisioning errors are stored in result.Error but don't return error
	if err != nil {
		t.Errorf("Provision() error = %v, should be nil", err)
	}
	if result.Error == nil {
		t.Error("result.Error should contain AP disable error")
	}
}

func TestProvisioner_Provision_DisableBLEError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456"}`)
			case "WiFi.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "WiFi.GetStatus":
				return jsonrpcResponse(`{"sta_ip":"192.168.1.100"}`)
			case "BLE.SetConfig":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	config := &DeviceConfig{
		WiFi: &WiFiConfig{SSID: "Test"},
	}

	opts := &ProvisionOptions{
		WaitForConnection: true,
		ConnectionTimeout: 1,
		DisableBLE:        true,
	}

	result, err := prov.Provision(context.Background(), config, opts)
	// Post-provisioning errors are stored in result.Error but don't return error
	if err != nil {
		t.Errorf("Provision() error = %v, should be nil", err)
	}
	if result.Error == nil {
		t.Error("result.Error should contain BLE disable error")
	}
}

func TestProvisioner_Provision_WaitForConnectionError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456"}`)
			case "WiFi.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "WiFi.GetStatus":
				return jsonrpcResponse(`{"sta_ip":""}`) // Never connects
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	prov := New(client)

	config := &DeviceConfig{
		WiFi: &WiFiConfig{SSID: "Test"},
	}

	opts := &ProvisionOptions{
		WaitForConnection: true,
		ConnectionTimeout: 1, // Very short timeout
	}

	result, err := prov.Provision(context.Background(), config, opts)
	// Post-provisioning errors are stored in result.Error
	if err != nil {
		t.Errorf("Provision() error = %v, should be nil", err)
	}
	if result.Error == nil {
		t.Error("result.Error should contain connection timeout error")
	}
}

// Test BLE provisioner with mock transmitter

func TestBLEProvisioner_ProvisionViaBLE_WithTransmitter(t *testing.T) {
	b := NewBLEProvisioner()
	mock := newMockBLETransmitter()
	mock.SetNotifications([]byte(`{"id":1,"result":{}}`), []byte(`{"id":2,"result":{}}`))
	b.Transmitter = mock

	b.AddDiscoveredDevice(&BLEDevice{
		Name:     "ShellyPlus1-123456",
		Address:  "AA:BB:CC:DD:EE:FF",
		IsShelly: true,
	})

	config := &BLEProvisionConfig{
		WiFi: &WiFiConfig{
			SSID:     "TestNetwork",
			Password: "TestPassword",
		},
		DeviceName: "Kitchen Light",
	}

	result, err := b.ProvisionViaBLE(context.Background(), "AA:BB:CC:DD:EE:FF", config)
	if err != nil {
		t.Errorf("ProvisionViaBLE() error = %v", err)
		return
	}

	if !result.Success {
		t.Error("ProvisionViaBLE() Success = false, want true")
	}

	if !mock.disconnectCalled {
		t.Error("Transmitter should be disconnected after provisioning")
	}
	if len(mock.writtenData) != 2 { // WiFi and DeviceName
		t.Errorf("len(writtenData) = %d, want 2", len(mock.writtenData))
	}
}

func TestBLEProvisioner_ProvisionViaBLE_ConnectError(t *testing.T) {
	b := NewBLEProvisioner()
	mock := newMockBLETransmitter()
	mock.connectErr = errTest
	b.Transmitter = mock

	b.AddDiscoveredDevice(&BLEDevice{
		Name:     "ShellyPlus1-123456",
		Address:  "AA:BB:CC:DD:EE:FF",
		IsShelly: true,
	})

	config := &BLEProvisionConfig{
		WiFi: &WiFiConfig{SSID: "Test"},
	}

	result, err := b.ProvisionViaBLE(context.Background(), "AA:BB:CC:DD:EE:FF", config)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, ErrBLEConnectionFailed) {
		t.Errorf("error = %v, want ErrBLEConnectionFailed", err)
	}
	if result.Success {
		t.Error("Success = true, want false")
	}
}

func TestBLEProvisioner_ProvisionViaBLE_WriteError(t *testing.T) {
	b := NewBLEProvisioner()
	mock := newMockBLETransmitter()
	mock.writeErr = errTest
	b.Transmitter = mock

	b.AddDiscoveredDevice(&BLEDevice{
		Name:     "ShellyPlus1-123456",
		Address:  "AA:BB:CC:DD:EE:FF",
		IsShelly: true,
	})

	config := &BLEProvisionConfig{
		WiFi: &WiFiConfig{SSID: "Test"},
	}

	result, err := b.ProvisionViaBLE(context.Background(), "AA:BB:CC:DD:EE:FF", config)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, ErrBLEWriteFailed) {
		t.Errorf("error = %v, want ErrBLEWriteFailed", err)
	}
	if result.Success {
		t.Error("Success = true, want false")
	}
}

func TestBLEProvisioner_ProvisionViaBLE_ReadError(t *testing.T) {
	b := NewBLEProvisioner()
	// Read errors are non-fatal, should still succeed
	mock := newMockBLETransmitter()
	mock.readErr = errTest
	b.Transmitter = mock

	b.AddDiscoveredDevice(&BLEDevice{
		Name:     "ShellyPlus1-123456",
		Address:  "AA:BB:CC:DD:EE:FF",
		IsShelly: true,
	})

	config := &BLEProvisionConfig{
		WiFi: &WiFiConfig{SSID: "Test"},
	}

	result, err := b.ProvisionViaBLE(context.Background(), "AA:BB:CC:DD:EE:FF", config)
	if err != nil {
		t.Errorf("ProvisionViaBLE() error = %v, want nil (read errors are non-fatal)", err)
	}
	if !result.Success {
		t.Error("Success = false, want true (read errors are non-fatal)")
	}
}

func TestBLEProvisioner_BuildProvisionCommands_Empty(t *testing.T) {
	b := NewBLEProvisioner()

	// Empty config should produce no commands
	config := &BLEProvisionConfig{}
	commands := b.buildProvisionCommands(config)

	if len(commands) != 0 {
		t.Errorf("len(commands) = %d, want 0", len(commands))
	}
}

func TestBLEProvisioner_BuildProvisionCommands_WiFiNoPassword(t *testing.T) {
	b := NewBLEProvisioner()

	// WiFi with no password (open network)
	config := &BLEProvisionConfig{
		WiFi: &WiFiConfig{SSID: "OpenNetwork"},
	}
	commands := b.buildProvisionCommands(config)

	if len(commands) != 1 {
		t.Errorf("len(commands) = %d, want 1", len(commands))
	}
}

func TestBLEProvisioner_BuildProvisionCommands_CloudDisabled(t *testing.T) {
	b := NewBLEProvisioner()

	disableCloud := false
	config := &BLEProvisionConfig{
		EnableCloud: &disableCloud,
	}
	commands := b.buildProvisionCommands(config)

	if len(commands) != 1 {
		t.Errorf("len(commands) = %d, want 1", len(commands))
	}
	if commands[0].Method != "Cloud.SetConfig" {
		t.Errorf("Method = %s, want Cloud.SetConfig", commands[0].Method)
	}
}
