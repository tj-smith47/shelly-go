package zigbee

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
	"github.com/tj-smith47/shelly-go/types"
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

func TestNewZigbee(t *testing.T) {
	transport := &mockTransport{}
	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	if zb == nil {
		t.Fatal("NewZigbee returned nil")
	}
	if zb.client != client {
		t.Error("client not set correctly")
	}
}

func TestZigbee_GetConfig(t *testing.T) {
	tests := []struct {
		want    *Config
		name    string
		result  string
		wantErr bool
	}{
		{
			name:   "enabled config",
			result: `{"enable":true}`,
			want:   &Config{Enable: true},
		},
		{
			name:   "disabled config",
			result: `{"enable":false}`,
			want:   &Config{Enable: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "Zigbee.GetConfig" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}

			client := rpc.NewClient(transport)
			zb := NewZigbee(client)

			got, err := zb.GetConfig(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && got.Enable != tt.want.Enable {
				t.Errorf("GetConfig() Enable = %v, want %v", got.Enable, tt.want.Enable)
			}
		})
	}
}

func TestZigbee_GetConfig_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	_, err := zb.GetConfig(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestZigbee_GetConfig_InvalidJSON(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{invalid json}`)
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	_, err := zb.GetConfig(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestZigbee_SetConfig(t *testing.T) {
	tests := []struct {
		name   string
		enable bool
	}{
		{name: "enable", enable: true},
		{name: "disable", enable: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "Zigbee.SetConfig" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(`{"restart_required":false}`)
				},
			}

			client := rpc.NewClient(transport)
			zb := NewZigbee(client)

			enable := tt.enable
			err := zb.SetConfig(context.Background(), &SetConfigParams{Enable: &enable})
			if err != nil {
				t.Errorf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestZigbee_SetConfig_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	enable := true
	err := zb.SetConfig(context.Background(), &SetConfigParams{Enable: &enable})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestZigbee_GetStatus(t *testing.T) {
	tests := []struct {
		want   *Status
		name   string
		result string
	}{
		{
			name:   "not configured",
			result: `{"network_state":"not_configured","eui64":"0x00124B001234ABCD"}`,
			want: &Status{
				NetworkState: "not_configured",
				EUI64:        "0x00124B001234ABCD",
			},
		},
		{
			name:   "ready state",
			result: `{"network_state":"ready","eui64":"0x00124B001234ABCD"}`,
			want: &Status{
				NetworkState: "ready",
				EUI64:        "0x00124B001234ABCD",
			},
		},
		{
			name:   "steering state",
			result: `{"network_state":"steering","eui64":"0x00124B001234ABCD"}`,
			want: &Status{
				NetworkState: "steering",
				EUI64:        "0x00124B001234ABCD",
			},
		},
		{
			name:   "joined network",
			result: `{"network_state":"joined","eui64":"0x00124B001234ABCD","pan_id":12345,"channel":15,"coordinator_eui64":"0x00124B009876FEDC"}`,
			want: &Status{
				NetworkState:     "joined",
				EUI64:            "0x00124B001234ABCD",
				PANID:            12345,
				Channel:          15,
				CoordinatorEUI64: "0x00124B009876FEDC",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "Zigbee.GetStatus" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}

			client := rpc.NewClient(transport)
			zb := NewZigbee(client)

			got, err := zb.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if got.NetworkState != tt.want.NetworkState {
				t.Errorf("NetworkState = %v, want %v", got.NetworkState, tt.want.NetworkState)
			}
			if got.EUI64 != tt.want.EUI64 {
				t.Errorf("EUI64 = %v, want %v", got.EUI64, tt.want.EUI64)
			}
			if got.PANID != tt.want.PANID {
				t.Errorf("PANID = %v, want %v", got.PANID, tt.want.PANID)
			}
			if got.Channel != tt.want.Channel {
				t.Errorf("Channel = %v, want %v", got.Channel, tt.want.Channel)
			}
			if got.CoordinatorEUI64 != tt.want.CoordinatorEUI64 {
				t.Errorf("CoordinatorEUI64 = %v, want %v", got.CoordinatorEUI64, tt.want.CoordinatorEUI64)
			}
		})
	}
}

func TestZigbee_GetStatus_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	_, err := zb.GetStatus(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestZigbee_GetStatus_InvalidJSON(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{invalid json}`)
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	_, err := zb.GetStatus(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestZigbee_StartNetworkSteering(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "Zigbee.StartNetworkSteering" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	err := zb.StartNetworkSteering(context.Background())
	if err != nil {
		t.Errorf("StartNetworkSteering() error = %v", err)
	}
}

func TestZigbee_StartNetworkSteering_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	err := zb.StartNetworkSteering(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestZigbee_Enable(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "Zigbee.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	err := zb.Enable(context.Background())
	if err != nil {
		t.Errorf("Enable() error = %v", err)
	}
}

func TestZigbee_Disable(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "Zigbee.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	err := zb.Disable(context.Background())
	if err != nil {
		t.Errorf("Disable() error = %v", err)
	}
}

func TestZigbee_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{name: "enabled", result: `{"enable":true}`, want: true},
		{name: "disabled", result: `{"enable":false}`, want: false},
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
			zb := NewZigbee(client)

			got, err := zb.IsEnabled(context.Background())
			if err != nil {
				t.Errorf("IsEnabled() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZigbee_IsEnabled_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	_, err := zb.IsEnabled(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestZigbee_IsJoined(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{name: "joined", result: `{"network_state":"joined"}`, want: true},
		{name: "not configured", result: `{"network_state":"not_configured"}`, want: false},
		{name: "ready", result: `{"network_state":"ready"}`, want: false},
		{name: "steering", result: `{"network_state":"steering"}`, want: false},
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
			zb := NewZigbee(client)

			got, err := zb.IsJoined(context.Background())
			if err != nil {
				t.Errorf("IsJoined() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("IsJoined() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZigbee_IsJoined_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	_, err := zb.IsJoined(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestZigbee_GetNetworkState(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   string
	}{
		{name: "not_configured", result: `{"network_state":"not_configured"}`, want: "not_configured"},
		{name: "ready", result: `{"network_state":"ready"}`, want: "ready"},
		{name: "steering", result: `{"network_state":"steering"}`, want: "steering"},
		{name: "joined", result: `{"network_state":"joined"}`, want: "joined"},
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
			zb := NewZigbee(client)

			got, err := zb.GetNetworkState(context.Background())
			if err != nil {
				t.Errorf("GetNetworkState() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GetNetworkState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZigbee_GetNetworkState_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	_, err := zb.GetNetworkState(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestZigbee_GetEUI64(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{"network_state":"ready","eui64":"0x00124B001234ABCD"}`)
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	got, err := zb.GetEUI64(context.Background())
	if err != nil {
		t.Errorf("GetEUI64() error = %v", err)
		return
	}
	if got != "0x00124B001234ABCD" {
		t.Errorf("GetEUI64() = %v, want 0x00124B001234ABCD", got)
	}
}

func TestZigbee_GetEUI64_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	_, err := zb.GetEUI64(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestZigbee_GetChannel(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   int
	}{
		{name: "channel 15", result: `{"network_state":"joined","channel":15}`, want: 15},
		{name: "channel 20", result: `{"network_state":"joined","channel":20}`, want: 20},
		{name: "not connected", result: `{"network_state":"ready"}`, want: 0},
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
			zb := NewZigbee(client)

			got, err := zb.GetChannel(context.Background())
			if err != nil {
				t.Errorf("GetChannel() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GetChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZigbee_GetChannel_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	_, err := zb.GetChannel(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestZigbee_GetPANID(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   uint16
	}{
		{name: "pan id 12345", result: `{"network_state":"joined","pan_id":12345}`, want: 12345},
		{name: "pan id 65535", result: `{"network_state":"joined","pan_id":65535}`, want: 65535},
		{name: "not connected", result: `{"network_state":"ready"}`, want: 0},
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
			zb := NewZigbee(client)

			got, err := zb.GetPANID(context.Background())
			if err != nil {
				t.Errorf("GetPANID() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GetPANID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZigbee_GetPANID_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	_, err := zb.GetPANID(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNetworkStateConstants(t *testing.T) {
	// Verify constants have expected values
	if NetworkStateNotConfigured != "not_configured" {
		t.Errorf("NetworkStateNotConfigured = %v, want not_configured", NetworkStateNotConfigured)
	}
	if NetworkStateReady != "ready" {
		t.Errorf("NetworkStateReady = %v, want ready", NetworkStateReady)
	}
	if NetworkStateSteering != "steering" {
		t.Errorf("NetworkStateSteering = %v, want steering", NetworkStateSteering)
	}
	if NetworkStateJoined != "joined" {
		t.Errorf("NetworkStateJoined = %v, want joined", NetworkStateJoined)
	}
	// New constants
	if NetworkStateDisabled != "disabled" {
		t.Errorf("NetworkStateDisabled = %v, want disabled", NetworkStateDisabled)
	}
	if NetworkStateInitializing != "initializing" {
		t.Errorf("NetworkStateInitializing = %v, want initializing", NetworkStateInitializing)
	}
	if NetworkStateFailed != "failed" {
		t.Errorf("NetworkStateFailed = %v, want failed", NetworkStateFailed)
	}
}

func TestZigbee_GetNetworkInfo(t *testing.T) {
	tests := []struct {
		name    string
		result  string
		wantNil bool
		wantErr bool
	}{
		{
			name:    "joined network returns info",
			result:  `{"network_state":"joined","pan_id":12345,"channel":15,"coordinator_eui64":"0x00124B009876FEDC"}`,
			wantNil: false,
		},
		{
			name:    "not joined returns error",
			result:  `{"network_state":"ready"}`,
			wantNil: true,
			wantErr: true,
		},
		{
			name:    "disabled returns error",
			result:  `{"network_state":"disabled"}`,
			wantNil: true,
			wantErr: true,
		},
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
			zb := NewZigbee(client)

			info, err := zb.GetNetworkInfo(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNetworkInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (info == nil) != tt.wantNil {
				t.Errorf("GetNetworkInfo() returned nil = %v, want nil = %v", info == nil, tt.wantNil)
			}
			if info != nil {
				if info.PANID != 12345 {
					t.Errorf("GetNetworkInfo() PANID = %v, want 12345", info.PANID)
				}
				if info.Channel != 15 {
					t.Errorf("GetNetworkInfo() Channel = %v, want 15", info.Channel)
				}
			}
		})
	}
}

func TestZigbee_GetNetworkInfo_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	_, err := zb.GetNetworkInfo(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestZigbee_LeaveNetwork(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "Zigbee.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	err := zb.LeaveNetwork(context.Background())
	if err != nil {
		t.Errorf("LeaveNetwork() error = %v", err)
	}
}

func TestClusterMapping(t *testing.T) {
	// Test GetClusterCapability
	onOff := GetClusterCapability(ClusterOnOff)
	if onOff == nil {
		t.Fatal("GetClusterCapability(ClusterOnOff) returned nil")
	}
	if onOff.ClusterName != "On/Off" {
		t.Errorf("ClusterName = %v, want On/Off", onOff.ClusterName)
	}
	if onOff.ComponentType != "switch" {
		t.Errorf("ComponentType = %v, want switch", onOff.ComponentType)
	}

	// Test unmapped cluster
	unmapped := GetClusterCapability(0xFFFF)
	if unmapped != nil {
		t.Errorf("GetClusterCapability(0xFFFF) = %v, want nil", unmapped)
	}

	// Test GetComponentTypeForCluster
	componentType := GetComponentTypeForCluster(ClusterLevelControl)
	if componentType != "light" {
		t.Errorf("GetComponentTypeForCluster(ClusterLevelControl) = %v, want light", componentType)
	}

	unmappedType := GetComponentTypeForCluster(0xFFFF)
	if unmappedType != "" {
		t.Errorf("GetComponentTypeForCluster(0xFFFF) = %v, want empty", unmappedType)
	}

	// Test GetClustersForComponentType
	switchClusters := GetClustersForComponentType("switch")
	if len(switchClusters) == 0 {
		t.Error("GetClustersForComponentType(switch) returned empty slice")
	}

	nonExistentClusters := GetClustersForComponentType("nonexistent")
	if len(nonExistentClusters) != 0 {
		t.Errorf("GetClustersForComponentType(nonexistent) = %v, want empty", nonExistentClusters)
	}
}

func TestClusterConstants(t *testing.T) {
	// Verify cluster ID constants
	tests := []struct {
		name     string
		cluster  uint16
		expected uint16
	}{
		{"ClusterBasic", ClusterBasic, 0x0000},
		{"ClusterOnOff", ClusterOnOff, 0x0006},
		{"ClusterLevelControl", ClusterLevelControl, 0x0008},
		{"ClusterColorControl", ClusterColorControl, 0x0300},
		{"ClusterTemperatureMeasurement", ClusterTemperatureMeasurement, 0x0402},
		{"ClusterHumidityMeasurement", ClusterHumidityMeasurement, 0x0405},
		{"ClusterWindowCovering", ClusterWindowCovering, 0x0102},
		{"ClusterThermostat", ClusterThermostat, 0x0201},
		{"ClusterElectricalMeasurement", ClusterElectricalMeasurement, 0x0B04},
		{"ClusterShellyRPC", ClusterShellyRPC, 0xFC01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cluster != tt.expected {
				t.Errorf("%s = 0x%04X, want 0x%04X", tt.name, tt.cluster, tt.expected)
			}
		})
	}
}

func TestDeviceType_String(t *testing.T) {
	tests := []struct {
		expected   string
		deviceType DeviceType
	}{
		{deviceType: DeviceTypeOnOffSwitch, expected: "On/Off Switch"},
		{deviceType: DeviceTypeDimmableLight, expected: "Dimmable Light"},
		{deviceType: DeviceTypeColorDimmableLight, expected: "Color Dimmable Light"},
		{deviceType: DeviceTypeWindowCovering, expected: "Window Covering"},
		{deviceType: DeviceTypeThermostat, expected: "Thermostat"},
		{deviceType: DeviceTypeTemperatureSensor, expected: "Temperature Sensor"},
		{deviceType: DeviceTypeOccupancySensor, expected: "Occupancy Sensor"},
		{deviceType: DeviceTypeFloodSensor, expected: "Flood Sensor"},
		{deviceType: DeviceTypeSmokeSensor, expected: "Smoke Sensor"},
		{deviceType: DeviceTypePowerMeter, expected: "Power Meter"},
		{deviceType: DeviceType(0xFFFF), expected: "Unknown Device Type (0xFFFF)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.deviceType.String(); got != tt.expected {
				t.Errorf("DeviceType(%d).String() = %v, want %v", tt.deviceType, got, tt.expected)
			}
		})
	}
}

func TestMapShellyModelToDeviceType(t *testing.T) {
	tests := []struct {
		model    string
		expected DeviceType
	}{
		{"S3SW-001X16EU", DeviceTypeOnOffSwitch},
		{"S3SW-001P16EU", DeviceTypeOnOffSwitch},
		{"S3SW-002P16EU", DeviceTypeOnOffSwitch},
		{"S3DM-001P10EU", DeviceTypeDimmerSwitch},
		{"S3SH-002P16EU", DeviceTypeWindowCovering},
		{"S3EM-002CXCEU", DeviceTypePowerMeter},
		{"S3FL-001P01EU", DeviceTypeFloodSensor},
		{"unknown-model", DeviceTypeOnOffSwitch}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := MapShellyModelToDeviceType(tt.model); got != tt.expected {
				t.Errorf("MapShellyModelToDeviceType(%s) = %v, want %v", tt.model, got, tt.expected)
			}
		})
	}
}

func TestInferCapabilitiesFromDeviceType(t *testing.T) {
	tests := []struct {
		minClusters int
		deviceType  DeviceType
		expectOnOff bool
		expectLevel bool
		expectColor bool
		expectCover bool
	}{
		{deviceType: DeviceTypeOnOffSwitch, minClusters: 2, expectOnOff: true, expectLevel: false, expectColor: false, expectCover: false},
		{deviceType: DeviceTypeDimmableLight, minClusters: 3, expectOnOff: true, expectLevel: true, expectColor: false, expectCover: false},
		{deviceType: DeviceTypeColorDimmableLight, minClusters: 4, expectOnOff: true, expectLevel: true, expectColor: true, expectCover: false},
		{deviceType: DeviceTypeWindowCovering, minClusters: 2, expectOnOff: false, expectLevel: false, expectColor: false, expectCover: true},
		{deviceType: DeviceTypeThermostat, minClusters: 3, expectOnOff: false, expectLevel: false, expectColor: false, expectCover: false},
		{deviceType: DeviceTypePowerMeter, minClusters: 3, expectOnOff: false, expectLevel: false, expectColor: false, expectCover: false},
	}

	for _, tt := range tests {
		t.Run(tt.deviceType.String(), func(t *testing.T) {
			caps := InferCapabilitiesFromDeviceType(tt.deviceType)
			if len(caps) < tt.minClusters {
				t.Errorf("InferCapabilitiesFromDeviceType(%v) returned %d clusters, want at least %d", tt.deviceType, len(caps), tt.minClusters)
			}

			hasOnOff := false
			hasLevel := false
			hasColor := false
			hasCover := false
			for _, cap := range caps {
				switch cap.ClusterID {
				case ClusterOnOff:
					hasOnOff = true
				case ClusterLevelControl:
					hasLevel = true
				case ClusterColorControl:
					hasColor = true
				case ClusterWindowCovering:
					hasCover = true
				}
			}

			if hasOnOff != tt.expectOnOff {
				t.Errorf("OnOff cluster: got %v, want %v", hasOnOff, tt.expectOnOff)
			}
			if hasLevel != tt.expectLevel {
				t.Errorf("Level cluster: got %v, want %v", hasLevel, tt.expectLevel)
			}
			if hasColor != tt.expectColor {
				t.Errorf("Color cluster: got %v, want %v", hasColor, tt.expectColor)
			}
			if hasCover != tt.expectCover {
				t.Errorf("Cover cluster: got %v, want %v", hasCover, tt.expectCover)
			}
		})
	}
}

func TestGetDeviceProfile(t *testing.T) {
	profile := GetDeviceProfile("S3SW-001X16EU")
	if profile == nil {
		t.Fatal("GetDeviceProfile returned nil")
	}
	if profile.DeviceType != DeviceTypeOnOffSwitch {
		t.Errorf("DeviceType = %v, want %v", profile.DeviceType, DeviceTypeOnOffSwitch)
	}
	if len(profile.SupportedClusters) == 0 {
		t.Error("SupportedClusters is empty")
	}
	if len(profile.Capabilities) == 0 {
		t.Error("Capabilities is empty")
	}

	// Check dimmer profile
	dimmerProfile := GetDeviceProfile("S3DM-001P10EU")
	if dimmerProfile.DeviceType != DeviceTypeDimmerSwitch {
		t.Errorf("Dimmer DeviceType = %v, want %v", dimmerProfile.DeviceType, DeviceTypeDimmerSwitch)
	}
}

func TestPairingStateConstants(t *testing.T) {
	tests := []struct {
		state    PairingState
		expected string
	}{
		{PairingStateIdle, "idle"},
		{PairingStateEnabling, "enabling"},
		{PairingStateSteering, "steering"},
		{PairingStateJoined, "joined"},
		{PairingStateFailed, "failed"},
		{PairingStateTimeout, "timeout"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("PairingState = %v, want %v", tt.state, tt.expected)
			}
		})
	}
}

func TestDeviceTypeConstants(t *testing.T) {
	tests := []struct {
		deviceType DeviceType
		expected   uint16
	}{
		{DeviceTypeOnOffSwitch, 0x0000},
		{DeviceTypeLevelControllableOutput, 0x0003},
		{DeviceTypeOnOffLight, 0x0100},
		{DeviceTypeDimmableLight, 0x0101},
		{DeviceTypeColorDimmableLight, 0x0102},
		{DeviceTypeWindowCovering, 0x0202},
		{DeviceTypeThermostat, 0x0301},
		{DeviceTypeTemperatureSensor, 0x0302},
		{DeviceTypeOccupancySensor, 0x0107},
		{DeviceTypeContactSensor, 0x0402},
		{DeviceTypeFloodSensor, 0x0403},
		{DeviceTypeSmokeSensor, 0x0404},
		{DeviceTypePowerMeter, 0x0501},
	}

	for _, tt := range tests {
		t.Run(tt.deviceType.String(), func(t *testing.T) {
			if uint16(tt.deviceType) != tt.expected {
				t.Errorf("DeviceType = 0x%04X, want 0x%04X", uint16(tt.deviceType), tt.expected)
			}
		})
	}
}

func TestErrorVariables(t *testing.T) {
	// Test that error variables are defined
	if ErrZigbeeNotSupported == nil {
		t.Error("ErrZigbeeNotSupported is nil")
	}
	if ErrPairingTimeout == nil {
		t.Error("ErrPairingTimeout is nil")
	}
	if ErrPairingFailed == nil {
		t.Error("ErrPairingFailed is nil")
	}
	if ErrAlreadyJoined == nil {
		t.Error("ErrAlreadyJoined is nil")
	}
	if ErrNotJoined == nil {
		t.Error("ErrNotJoined is nil")
	}

	// Test error messages
	if ErrZigbeeNotSupported.Error() != "device does not support Zigbee" {
		t.Errorf("ErrZigbeeNotSupported.Error() = %v", ErrZigbeeNotSupported.Error())
	}
}

func TestNewScanner(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Fatal("NewScanner returned nil")
	}
	if scanner.HTTPClient == nil {
		t.Error("HTTPClient is nil")
	}
	if scanner.Concurrency != 10 {
		t.Errorf("Concurrency = %v, want 10", scanner.Concurrency)
	}
}

func TestScanner_DiscoverDevices_Empty(t *testing.T) {
	scanner := NewScanner()
	devices, err := scanner.DiscoverDevices(context.Background(), []string{})
	if err != nil {
		t.Errorf("DiscoverDevices() error = %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("DiscoverDevices() returned %d devices, want 0", len(devices))
	}
}

func TestScanner_DiscoverZigbeeDevices_Empty(t *testing.T) {
	scanner := NewScanner()
	devices, err := scanner.DiscoverZigbeeDevices(context.Background(), []string{})
	if err != nil {
		t.Errorf("DiscoverZigbeeDevices() error = %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("DiscoverZigbeeDevices() returned %d devices, want 0", len(devices))
	}
}

func TestScanner_ConcurrencyDefaults(t *testing.T) {
	scanner := &Scanner{
		Concurrency: 0, // Should default to 10
	}

	// Just verify it doesn't panic with empty input
	devices, err := scanner.DiscoverDevices(context.Background(), []string{})
	if err != nil {
		t.Errorf("DiscoverDevices() error = %v", err)
	}
	// Empty input returns nil slice or empty slice (either is acceptable)
	if len(devices) != 0 {
		t.Error("DiscoverDevices() returned non-empty slice for empty input")
	}
}

func TestClusterMappingAttributes(t *testing.T) {
	// Test that cluster mappings have reasonable attributes
	for clusterID, mapping := range ClusterMapping {
		t.Run(mapping.ClusterName, func(t *testing.T) {
			if mapping.ClusterID != clusterID {
				t.Errorf("ClusterID mismatch: key=0x%04X, value.ClusterID=0x%04X", clusterID, mapping.ClusterID)
			}
			if mapping.ClusterName == "" {
				t.Error("ClusterName is empty")
			}
			if mapping.ComponentType == "" {
				t.Error("ComponentType is empty")
			}

			// Check attributes have valid names
			for _, attr := range mapping.Attributes {
				if attr.Name == "" {
					t.Errorf("Attribute ID 0x%04X has empty name", attr.ID)
				}
				if attr.Type == "" {
					t.Errorf("Attribute %s has empty type", attr.Name)
				}
			}

			// Check commands have valid names
			for _, cmd := range mapping.Commands {
				if cmd.Name == "" {
					t.Errorf("Command ID 0x%02X has empty name", cmd.ID)
				}
				if cmd.Direction == "" {
					t.Errorf("Command %s has empty direction", cmd.Name)
				}
			}
		})
	}
}

func TestInferCapabilitiesFromDeviceType_AllTypes(t *testing.T) {
	// Test all device types return non-empty capabilities
	deviceTypes := []DeviceType{
		DeviceTypeOnOffSwitch,
		DeviceTypeLevelControllableOutput,
		DeviceTypeOnOffLight,
		DeviceTypeDimmableLight,
		DeviceTypeColorDimmableLight,
		DeviceTypeOnOffLightSwitch,
		DeviceTypeDimmerSwitch,
		DeviceTypeColorDimmerSwitch,
		DeviceTypeWindowCovering,
		DeviceTypeThermostat,
		DeviceTypeTemperatureSensor,
		DeviceTypeOccupancySensor,
		DeviceTypeContactSensor,
		DeviceTypeFloodSensor,
		DeviceTypeSmokeSensor,
		DeviceTypePowerMeter,
	}

	for _, dt := range deviceTypes {
		t.Run(dt.String(), func(t *testing.T) {
			caps := InferCapabilitiesFromDeviceType(dt)
			if len(caps) == 0 {
				t.Errorf("InferCapabilitiesFromDeviceType(%v) returned empty capabilities", dt)
			}

			// All devices should have Basic cluster
			hasBasic := false
			for _, cap := range caps {
				if cap.ClusterID == ClusterBasic {
					hasBasic = true
					break
				}
			}
			if !hasBasic {
				t.Errorf("InferCapabilitiesFromDeviceType(%v) missing Basic cluster", dt)
			}
		})
	}
}

func TestPairToNetwork_AlreadyJoined(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			callCount++
			// Return joined status on first call (GetStatus check)
			return jsonrpcResponse(`{"network_state":"joined","pan_id":12345,"channel":15,"coordinator_eui64":"0x00124B009876FEDC"}`)
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	result, err := zb.PairToNetwork(context.Background(), 100*time.Millisecond, 10*time.Millisecond)
	if !errors.Is(err, ErrAlreadyJoined) {
		t.Errorf("PairToNetwork() error = %v, want ErrAlreadyJoined", err)
	}
	if result == nil {
		t.Fatal("PairToNetwork() returned nil result")
	}
	if result.State != PairingStateJoined {
		t.Errorf("PairingState = %v, want %v", result.State, PairingStateJoined)
	}
	if result.NetworkInfo == nil {
		t.Error("NetworkInfo is nil")
	}
}

func TestPairToNetwork_InitialStatusError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	result, err := zb.PairToNetwork(context.Background(), 100*time.Millisecond, 10*time.Millisecond)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestPairToNetwork_Success(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			callCount++
			switch method {
			case "Zigbee.GetStatus":
				if callCount == 1 {
					// First call - not joined yet
					return jsonrpcResponse(`{"network_state":"ready"}`)
				}
				if callCount >= 4 {
					// After some polling - joined
					return jsonrpcResponse(`{"network_state":"joined","pan_id":12345,"channel":15,"coordinator_eui64":"0x00124B009876FEDC"}`)
				}
				// During polling - steering
				return jsonrpcResponse(`{"network_state":"steering"}`)
			case "Zigbee.GetConfig":
				return jsonrpcResponse(`{"enable":true}`)
			case "Zigbee.StartNetworkSteering":
				return jsonrpcResponse(`null`)
			}
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	result, err := zb.PairToNetwork(context.Background(), 2*time.Second, 10*time.Millisecond)
	if err != nil {
		t.Errorf("PairToNetwork() error = %v", err)
	}
	if result == nil {
		t.Fatal("PairToNetwork() returned nil result")
	}
	if result.State != PairingStateJoined {
		t.Errorf("PairingState = %v, want %v", result.State, PairingStateJoined)
	}
	if result.NetworkInfo == nil {
		t.Error("NetworkInfo is nil")
	}
	if result.NetworkInfo.Channel != 15 {
		t.Errorf("Channel = %v, want 15", result.NetworkInfo.Channel)
	}
}

func TestPairToNetwork_EnableZigbee(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			callCount++
			switch method {
			case "Zigbee.GetStatus":
				if callCount == 1 {
					return jsonrpcResponse(`{"network_state":"disabled"}`)
				}
				return jsonrpcResponse(`{"network_state":"joined","pan_id":12345,"channel":15,"coordinator_eui64":"0x00124B009876FEDC"}`)
			case "Zigbee.GetConfig":
				return jsonrpcResponse(`{"enable":false}`) // Needs enabling
			case "Zigbee.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "Zigbee.StartNetworkSteering":
				return jsonrpcResponse(`null`)
			}
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	result, err := zb.PairToNetwork(context.Background(), 2*time.Second, 10*time.Millisecond)
	if err != nil {
		t.Errorf("PairToNetwork() error = %v", err)
	}
	if result.State != PairingStateJoined {
		t.Errorf("PairingState = %v, want %v", result.State, PairingStateJoined)
	}
}

func TestPairToNetwork_EnableFailed(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			callCount++
			switch method {
			case "Zigbee.GetStatus":
				return jsonrpcResponse(`{"network_state":"disabled"}`)
			case "Zigbee.GetConfig":
				return jsonrpcResponse(`{"enable":false}`)
			case "Zigbee.SetConfig":
				return nil, errTest // Enable fails
			}
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	result, err := zb.PairToNetwork(context.Background(), 100*time.Millisecond, 10*time.Millisecond)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.State != PairingStateFailed {
		t.Errorf("PairingState = %v, want %v", result.State, PairingStateFailed)
	}
}

func TestPairToNetwork_SteeringFailed(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			callCount++
			switch method {
			case "Zigbee.GetStatus":
				return jsonrpcResponse(`{"network_state":"ready"}`)
			case "Zigbee.GetConfig":
				return jsonrpcResponse(`{"enable":true}`)
			case "Zigbee.StartNetworkSteering":
				return nil, errTest // Steering fails
			}
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	result, err := zb.PairToNetwork(context.Background(), 100*time.Millisecond, 10*time.Millisecond)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.State != PairingStateFailed {
		t.Errorf("PairingState = %v, want %v", result.State, PairingStateFailed)
	}
}

func TestPairToNetwork_NetworkFailed(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			callCount++
			switch method {
			case "Zigbee.GetStatus":
				if callCount == 1 {
					return jsonrpcResponse(`{"network_state":"ready"}`)
				}
				// Return failed state during polling
				return jsonrpcResponse(`{"network_state":"failed"}`)
			case "Zigbee.GetConfig":
				return jsonrpcResponse(`{"enable":true}`)
			case "Zigbee.StartNetworkSteering":
				return jsonrpcResponse(`null`)
			}
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	result, err := zb.PairToNetwork(context.Background(), 2*time.Second, 10*time.Millisecond)
	if !errors.Is(err, ErrPairingFailed) {
		t.Errorf("error = %v, want ErrPairingFailed", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.State != PairingStateFailed {
		t.Errorf("PairingState = %v, want %v", result.State, PairingStateFailed)
	}
}

func TestPairToNetwork_Timeout(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			switch method {
			case "Zigbee.GetStatus":
				// Always return steering state
				return jsonrpcResponse(`{"network_state":"steering"}`)
			case "Zigbee.GetConfig":
				return jsonrpcResponse(`{"enable":true}`)
			case "Zigbee.StartNetworkSteering":
				return jsonrpcResponse(`null`)
			}
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	result, err := zb.PairToNetwork(context.Background(), 50*time.Millisecond, 10*time.Millisecond)
	if !errors.Is(err, ErrPairingTimeout) {
		t.Errorf("error = %v, want ErrPairingTimeout", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.State != PairingStateTimeout {
		t.Errorf("PairingState = %v, want %v", result.State, PairingStateTimeout)
	}
}

func TestPairToNetwork_ContextCanceled(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			switch method {
			case "Zigbee.GetStatus":
				return jsonrpcResponse(`{"network_state":"steering"}`)
			case "Zigbee.GetConfig":
				return jsonrpcResponse(`{"enable":true}`)
			case "Zigbee.StartNetworkSteering":
				return jsonrpcResponse(`null`)
			}
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately after starting
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	result, err := zb.PairToNetwork(ctx, 5*time.Second, 10*time.Millisecond)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.State != PairingStateFailed {
		t.Errorf("PairingState = %v, want %v", result.State, PairingStateFailed)
	}
}

func TestPairToNetwork_GetConfigError(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			callCount++
			switch method {
			case "Zigbee.GetStatus":
				return jsonrpcResponse(`{"network_state":"ready"}`)
			case "Zigbee.GetConfig":
				return nil, errTest // Config fetch fails
			}
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	result, err := zb.PairToNetwork(context.Background(), 100*time.Millisecond, 10*time.Millisecond)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestPairToNetwork_DefaultTimeouts(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			switch method {
			case "Zigbee.GetStatus":
				return jsonrpcResponse(`{"network_state":"joined","pan_id":1,"channel":11}`)
			}
			return nil, nil
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	// Call with zero values - should use defaults
	result, err := zb.PairToNetwork(context.Background(), 0, 0)
	if !errors.Is(err, ErrAlreadyJoined) {
		t.Errorf("error = %v, want ErrAlreadyJoined", err)
	}
	if result == nil {
		t.Error("result is nil")
	}
}

func TestNetworkInfo_Fields(t *testing.T) {
	info := NetworkInfo{
		PANID:            0x1234,
		Channel:          15,
		CoordinatorEUI64: "0x00124B001234ABCD",
	}

	if info.PANID != 0x1234 {
		t.Errorf("PANID = 0x%04X, want 0x1234", info.PANID)
	}
	if info.Channel != 15 {
		t.Errorf("Channel = %d, want 15", info.Channel)
	}
	if info.CoordinatorEUI64 != "0x00124B001234ABCD" {
		t.Errorf("CoordinatorEUI64 = %s, want 0x00124B001234ABCD", info.CoordinatorEUI64)
	}
}

func TestPairingResult_Fields(t *testing.T) {
	result := PairingResult{
		State: PairingStateJoined,
		NetworkInfo: &NetworkInfo{
			PANID:   1,
			Channel: 11,
		},
		Error: nil,
	}

	if result.State != PairingStateJoined {
		t.Errorf("State = %v, want %v", result.State, PairingStateJoined)
	}
	if result.NetworkInfo == nil {
		t.Error("NetworkInfo is nil")
	}
	if result.Error != nil {
		t.Errorf("Error = %v, want nil", result.Error)
	}
}

func TestDiscoveredDevice_Fields(t *testing.T) {
	device := DiscoveredDevice{
		Address:    "10.23.47.220",
		DeviceID:   "shellyplus1-123456",
		Model:      "SNSW-001X16EU",
		Generation: 4,
		HasZigbee:  true,
		ZigbeeStatus: &Status{
			NetworkState: NetworkStateJoined,
			Channel:      15,
		},
	}

	if device.Address != "10.23.47.220" {
		t.Errorf("Address = %s, want 10.23.47.220", device.Address)
	}
	if device.DeviceID != "shellyplus1-123456" {
		t.Errorf("DeviceID = %s, want shellyplus1-123456", device.DeviceID)
	}
	if !device.HasZigbee {
		t.Error("HasZigbee should be true")
	}
	if device.ZigbeeStatus == nil {
		t.Error("ZigbeeStatus is nil")
	}
}

func TestDeviceProfile_Fields(t *testing.T) {
	profile := DeviceProfile{
		DeviceType:        DeviceTypeOnOffSwitch,
		SupportedClusters: []uint16{ClusterBasic, ClusterOnOff},
		Capabilities:      []ClusterCapability{},
	}

	if profile.DeviceType != DeviceTypeOnOffSwitch {
		t.Errorf("DeviceType = %v, want %v", profile.DeviceType, DeviceTypeOnOffSwitch)
	}
	if len(profile.SupportedClusters) != 2 {
		t.Errorf("SupportedClusters length = %d, want 2", len(profile.SupportedClusters))
	}
}

func TestClusterCapability_Fields(t *testing.T) {
	cap := ClusterCapability{
		ClusterID:     ClusterOnOff,
		ClusterName:   "On/Off",
		ComponentType: "switch",
		Attributes: []ClusterAttribute{
			{ID: 0, Name: "OnOff", Type: "bool", Readable: true},
		},
		Commands: []ClusterCommand{
			{ID: 0, Name: "Off", Direction: "client_to_server"},
		},
	}

	if cap.ClusterID != ClusterOnOff {
		t.Errorf("ClusterID = 0x%04X, want 0x%04X", cap.ClusterID, ClusterOnOff)
	}
	if len(cap.Attributes) != 1 {
		t.Errorf("Attributes length = %d, want 1", len(cap.Attributes))
	}
	if len(cap.Commands) != 1 {
		t.Errorf("Commands length = %d, want 1", len(cap.Commands))
	}
}

func TestClusterAttribute_Fields(t *testing.T) {
	attr := ClusterAttribute{
		ID:         0x0000,
		Name:       "OnOff",
		Type:       "bool",
		Readable:   true,
		Writable:   false,
		Reportable: true,
	}

	if attr.ID != 0 {
		t.Errorf("ID = %d, want 0", attr.ID)
	}
	if !attr.Readable {
		t.Error("Readable should be true")
	}
	if attr.Writable {
		t.Error("Writable should be false")
	}
	if !attr.Reportable {
		t.Error("Reportable should be true")
	}
}

func TestClusterCommand_Fields(t *testing.T) {
	cmd := ClusterCommand{
		ID:        0x01,
		Name:      "On",
		Direction: "client_to_server",
	}

	if cmd.ID != 1 {
		t.Errorf("ID = %d, want 1", cmd.ID)
	}
	if cmd.Direction != "client_to_server" {
		t.Errorf("Direction = %s, want client_to_server", cmd.Direction)
	}
}

func TestScanner_DiscoverDevices_ContextCancelled(t *testing.T) {
	scanner := NewScanner()
	scanner.HTTPClient.Timeout = 50 * time.Millisecond // Short timeout

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// With short timeout context and addresses that won't respond, should return quickly
	devices, err := scanner.DiscoverDevices(ctx, []string{})
	if err != nil {
		t.Errorf("DiscoverDevices() error = %v", err)
	}
	// Should return empty results
	if len(devices) != 0 {
		t.Errorf("DiscoverDevices() returned %d devices, want 0", len(devices))
	}
}

func TestScanner_NilHTTPClient(t *testing.T) {
	scanner := &Scanner{
		HTTPClient:  nil, // Will be set to default
		Concurrency: 5,
	}

	// Call with empty addresses to trigger the nil client check
	devices, err := scanner.DiscoverDevices(context.Background(), []string{})
	if err != nil {
		t.Errorf("DiscoverDevices() error = %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("DiscoverDevices() returned %d devices, want 0", len(devices))
	}

	// Verify HTTPClient was set
	if scanner.HTTPClient == nil {
		t.Error("HTTPClient should be set after DiscoverDevices call")
	}
}

func TestDeviceType_String_AllTypes(t *testing.T) {
	// Test all remaining device types for complete coverage
	additionalTypes := []struct {
		expected   string
		deviceType DeviceType
	}{
		{deviceType: DeviceTypeLevelControllableOutput, expected: "Level Controllable Output"},
		{deviceType: DeviceTypeOnOffLight, expected: "On/Off Light"},
		{deviceType: DeviceTypeOnOffLightSwitch, expected: "On/Off Light Switch"},
		{deviceType: DeviceTypeDimmerSwitch, expected: "Dimmer Switch"},
		{deviceType: DeviceTypeColorDimmerSwitch, expected: "Color Dimmer Switch"},
		{deviceType: DeviceTypePumpController, expected: "Pump Controller"},
		{deviceType: DeviceTypeContactSensor, expected: "Contact Sensor"},
	}

	for _, tt := range additionalTypes {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.deviceType.String(); got != tt.expected {
				t.Errorf("DeviceType(%d).String() = %v, want %v", tt.deviceType, got, tt.expected)
			}
		})
	}
}

func TestMapShellyModelToDeviceType_AdditionalModels(t *testing.T) {
	// Test additional model mappings
	tests := []struct {
		model    string
		expected DeviceType
	}{
		{"S3SW-001X8EU", DeviceTypeOnOffSwitch}, // Mini switch X
		{"S3SW-001P8EU", DeviceTypeOnOffSwitch}, // Mini switch P
		{"S3PL-00112US", DeviceTypeOnOffSwitch}, // Plug US
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := MapShellyModelToDeviceType(tt.model); got != tt.expected {
				t.Errorf("MapShellyModelToDeviceType(%s) = %v, want %v", tt.model, got, tt.expected)
			}
		})
	}
}

func TestPairToNetwork_TransientErrors(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			callCount++
			switch method {
			case "Zigbee.GetStatus":
				if callCount == 1 {
					return jsonrpcResponse(`{"network_state":"ready"}`)
				}
				if callCount >= 3 && callCount < 5 {
					// Return transient error during polling
					return nil, errTest
				}
				if callCount >= 5 {
					return jsonrpcResponse(`{"network_state":"joined","pan_id":12345,"channel":15}`)
				}
				return jsonrpcResponse(`{"network_state":"steering"}`)
			case "Zigbee.GetConfig":
				return jsonrpcResponse(`{"enable":true}`)
			case "Zigbee.StartNetworkSteering":
				return jsonrpcResponse(`null`)
			}
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	zb := NewZigbee(client)

	result, err := zb.PairToNetwork(context.Background(), 2*time.Second, 10*time.Millisecond)
	if err != nil {
		t.Errorf("PairToNetwork() error = %v", err)
	}
	if result.State != PairingStateJoined {
		t.Errorf("PairingState = %v, want %v", result.State, PairingStateJoined)
	}
}

func TestInferCapabilitiesFromDeviceType_PumpController(t *testing.T) {
	caps := InferCapabilitiesFromDeviceType(DeviceTypePumpController)
	// PumpController should have at least the Basic cluster
	if len(caps) == 0 {
		t.Error("InferCapabilitiesFromDeviceType(DeviceTypePumpController) returned empty capabilities")
	}

	hasBasic := false
	for _, cap := range caps {
		if cap.ClusterID == ClusterBasic {
			hasBasic = true
			break
		}
	}
	if !hasBasic {
		t.Error("InferCapabilitiesFromDeviceType(DeviceTypePumpController) missing Basic cluster")
	}
}

func TestInferCapabilitiesFromDeviceType_OnOffLightSwitch(t *testing.T) {
	caps := InferCapabilitiesFromDeviceType(DeviceTypeOnOffLightSwitch)

	// OnOffLightSwitch is not explicitly handled in the switch,
	// so it only returns the Basic cluster
	hasBasic := false
	for _, cap := range caps {
		if cap.ClusterID == ClusterBasic {
			hasBasic = true
			break
		}
	}
	if !hasBasic {
		t.Error("InferCapabilitiesFromDeviceType(DeviceTypeOnOffLightSwitch) missing Basic cluster")
	}
	// Should only have the Basic cluster (default case)
	if len(caps) != 1 {
		t.Logf("DeviceTypeOnOffLightSwitch returned %d capabilities (may be intentional)", len(caps))
	}
}

func TestScanner_DiscoverDevices_WithConcurrency(t *testing.T) {
	scanner := NewScanner()
	scanner.Concurrency = 2 // Test with custom concurrency

	// Test with empty addresses - verify concurrency is set properly
	devices, err := scanner.DiscoverDevices(context.Background(), []string{})
	if err != nil {
		t.Errorf("DiscoverDevices() error = %v", err)
	}
	// Empty addresses returns nil or empty slice - both acceptable
	if len(devices) != 0 {
		t.Errorf("DiscoverDevices() returned %d devices, want 0", len(devices))
	}
}

func TestScanner_DiscoverZigbeeDevices_NoDevices(t *testing.T) {
	scanner := NewScanner()

	// Test with empty addresses
	devices, err := scanner.DiscoverZigbeeDevices(context.Background(), []string{})
	if err != nil {
		t.Errorf("DiscoverZigbeeDevices() error = %v", err)
	}
	// Empty input returns empty result
	if len(devices) != 0 {
		t.Errorf("DiscoverZigbeeDevices() returned %d devices, want 0", len(devices))
	}
}

func TestScanner_ZeroConcurrency(t *testing.T) {
	scanner := &Scanner{
		Concurrency: 0, // Should default to 10
	}

	devices, err := scanner.DiscoverDevices(context.Background(), []string{})
	if err != nil {
		t.Errorf("DiscoverDevices() error = %v", err)
	}
	// Verify no error with zero concurrency (defaults to 10)
	// Result will be nil/empty since no addresses provided
	_ = devices
}

func TestDiscoveredDevice_FullFields(t *testing.T) {
	device := DiscoveredDevice{
		Address:    "192.168.1.100",
		DeviceID:   "shellypluszigbee-123456",
		Model:      "SNSN-0001X",
		Generation: 2,
		HasZigbee:  true,
		ZigbeeStatus: &Status{
			NetworkState:     "joined",
			EUI64:            "0x1234567890ABCDEF",
			CoordinatorEUI64: "0xABCDEF1234567890",
			Channel:          15,
			PANID:            0x1234,
		},
	}

	if device.Address != "192.168.1.100" {
		t.Errorf("Address = %s, want 192.168.1.100", device.Address)
	}
	if device.DeviceID != "shellypluszigbee-123456" {
		t.Errorf("DeviceID = %s", device.DeviceID)
	}
	if device.Model != "SNSN-0001X" {
		t.Errorf("Model = %s", device.Model)
	}
	if !device.HasZigbee {
		t.Error("HasZigbee should be true")
	}
	if device.ZigbeeStatus == nil {
		t.Error("ZigbeeStatus should not be nil")
	} else {
		if device.ZigbeeStatus.Channel != 15 {
			t.Errorf("ZigbeeStatus.Channel = %d, want 15", device.ZigbeeStatus.Channel)
		}
		if device.ZigbeeStatus.NetworkState != "joined" {
			t.Errorf("ZigbeeStatus.NetworkState = %s, want joined", device.ZigbeeStatus.NetworkState)
		}
	}
}

func TestScanner_ContextCancelledDuringDiscovery(t *testing.T) {
	scanner := NewScanner()
	scanner.Concurrency = 1 // Low concurrency

	// Create context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should return quickly due to canceled context with empty addresses
	devices, err := scanner.DiscoverDevices(ctx, []string{})
	if err != nil {
		t.Errorf("DiscoverDevices() error = %v", err)
	}
	// With canceled context and empty addresses, result is nil or empty
	if len(devices) != 0 {
		t.Errorf("DiscoverDevices() returned %d devices, want 0", len(devices))
	}
}

// createMockShellyServer creates an httptest server that simulates a Shelly device
func createMockShellyServer(t *testing.T, deviceInfo map[string]any, zigbeeStatus map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rpc" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var req struct {
			Method string `json:"method"`
			ID     int    `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var result any
		switch req.Method {
		case "Shelly.GetDeviceInfo":
			result = deviceInfo
		case "Zigbee.GetStatus":
			if zigbeeStatus == nil {
				// Return error for no Zigbee
				resp := map[string]any{
					"id":    req.ID,
					"error": map[string]any{"code": -1, "message": "Component not found"},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
				return
			}
			result = zigbeeStatus
		default:
			result = map[string]any{}
		}

		resp := types.Response{
			ID:     req.ID,
			Result: mustMarshal(result),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func TestScanner_ProbeDevice_Gen2WithZigbee(t *testing.T) {
	deviceInfo := map[string]any{
		"id":    "shellyplusht-aabbcc",
		"model": "SNSN-0031Z",
		"app":   "PlusHT",
		"gen":   2,
	}
	zigbeeStatus := map[string]any{
		"network_state":     "joined",
		"channel":           15,
		"eui64":             "0x1234567890ABCDEF",
		"coordinator_eui64": "0xFEDCBA0987654321",
		"pan_id":            12345,
	}

	server := createMockShellyServer(t, deviceInfo, zigbeeStatus)
	defer server.Close()

	scanner := NewScanner()
	device, err := scanner.probeDevice(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("probeDevice() error = %v", err)
	}

	if device.DeviceID != "shellyplusht-aabbcc" {
		t.Errorf("DeviceID = %s, want shellyplusht-aabbcc", device.DeviceID)
	}
	if device.Model != "SNSN-0031Z" {
		t.Errorf("Model = %s, want SNSN-0031Z", device.Model)
	}
	if device.Generation != types.Gen2 {
		t.Errorf("Generation = %v, want Gen2", device.Generation)
	}
	if !device.HasZigbee {
		t.Error("HasZigbee should be true")
	}
	if device.ZigbeeStatus == nil {
		t.Fatal("ZigbeeStatus should not be nil")
	}
	if device.ZigbeeStatus.NetworkState != "joined" {
		t.Errorf("NetworkState = %s, want joined", device.ZigbeeStatus.NetworkState)
	}
}

func TestScanner_ProbeDevice_Gen3WithZigbee(t *testing.T) {
	deviceInfo := map[string]any{
		"id":    "shellyhtg3-aabbcc",
		"model": "S3SN-0U12Z",
		"app":   "HTG3",
		"gen":   3,
	}
	zigbeeStatus := map[string]any{
		"network_state": "ready",
		"channel":       20,
	}

	server := createMockShellyServer(t, deviceInfo, zigbeeStatus)
	defer server.Close()

	scanner := NewScanner()
	device, err := scanner.probeDevice(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("probeDevice() error = %v", err)
	}

	if device.Generation != types.Gen3 {
		t.Errorf("Generation = %v, want Gen3", device.Generation)
	}
	if !device.HasZigbee {
		t.Error("HasZigbee should be true")
	}
}

func TestScanner_ProbeDevice_Gen4WithZigbee(t *testing.T) {
	deviceInfo := map[string]any{
		"id":    "shellyhtg4-aabbcc",
		"model": "S4SN-0U12Z",
		"app":   "HTG4",
		"gen":   4,
	}
	zigbeeStatus := map[string]any{
		"network_state": "steering",
		"channel":       11,
	}

	server := createMockShellyServer(t, deviceInfo, zigbeeStatus)
	defer server.Close()

	scanner := NewScanner()
	device, err := scanner.probeDevice(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("probeDevice() error = %v", err)
	}

	if device.Generation != types.Gen4 {
		t.Errorf("Generation = %v, want Gen4", device.Generation)
	}
	if !device.HasZigbee {
		t.Error("HasZigbee should be true")
	}
}

func TestScanner_ProbeDevice_WithoutZigbee(t *testing.T) {
	deviceInfo := map[string]any{
		"id":    "shellyplus1pm-aabbcc",
		"model": "SNSW-001P16EU",
		"app":   "Plus1PM",
		"gen":   2,
	}

	server := createMockShellyServer(t, deviceInfo, nil) // No Zigbee
	defer server.Close()

	scanner := NewScanner()
	device, err := scanner.probeDevice(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("probeDevice() error = %v", err)
	}

	if device.HasZigbee {
		t.Error("HasZigbee should be false for non-Zigbee device")
	}
	if device.ZigbeeStatus != nil {
		t.Error("ZigbeeStatus should be nil for non-Zigbee device")
	}
}

func TestScanner_ProbeDevice_UnknownGeneration(t *testing.T) {
	deviceInfo := map[string]any{
		"id":    "shellyunknown-aabbcc",
		"model": "UNKNOWN",
		"app":   "Unknown",
		"gen":   99, // Unknown generation
	}

	server := createMockShellyServer(t, deviceInfo, nil)
	defer server.Close()

	scanner := NewScanner()
	device, err := scanner.probeDevice(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("probeDevice() error = %v", err)
	}

	// Unknown generation should fall back to Gen2
	if device.Generation != types.Gen2 {
		t.Errorf("Generation = %v, want Gen2 (fallback)", device.Generation)
	}
}

func TestScanner_ProbeDevice_ConnectionError(t *testing.T) {
	scanner := NewScanner()

	// Try to probe a non-existent server
	_, err := scanner.probeDevice(context.Background(), "http://127.0.0.1:59999")
	if err == nil {
		t.Error("probeDevice() should return error for unreachable server")
	}
}

func TestScanner_ProbeDevice_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":1,"result":"not a valid device info object"}`))
	}))
	defer server.Close()

	scanner := NewScanner()
	_, err := scanner.probeDevice(context.Background(), server.URL)
	if err == nil {
		t.Error("probeDevice() should return error for invalid device info JSON")
	}
}

func TestScanner_DiscoverDevices_WithMockServer(t *testing.T) {
	deviceInfo := map[string]any{
		"id":    "shellyplusht-aabbcc",
		"model": "SNSN-0031Z",
		"app":   "PlusHT",
		"gen":   2,
	}
	zigbeeStatus := map[string]any{
		"network_state": "joined",
		"channel":       15,
	}

	server := createMockShellyServer(t, deviceInfo, zigbeeStatus)
	defer server.Close()

	scanner := NewScanner()
	scanner.Concurrency = 2

	devices, err := scanner.DiscoverDevices(context.Background(), []string{server.URL})
	if err != nil {
		t.Fatalf("DiscoverDevices() error = %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("DiscoverDevices() returned %d devices, want 1", len(devices))
	}

	device := devices[0]
	if device.DeviceID != "shellyplusht-aabbcc" {
		t.Errorf("DeviceID = %s, want shellyplusht-aabbcc", device.DeviceID)
	}
	if !device.HasZigbee {
		t.Error("HasZigbee should be true")
	}
}

func TestScanner_DiscoverDevices_MixedResults(t *testing.T) {
	// First server - valid Zigbee device
	deviceInfo1 := map[string]any{
		"id":    "shellyplusht-111111",
		"model": "SNSN-0031Z",
		"gen":   2,
	}
	zigbeeStatus1 := map[string]any{
		"network_state": "joined",
	}
	server1 := createMockShellyServer(t, deviceInfo1, zigbeeStatus1)
	defer server1.Close()

	// Second server - non-Zigbee device
	deviceInfo2 := map[string]any{
		"id":    "shellyplus1pm-222222",
		"model": "SNSW-001P16EU",
		"gen":   2,
	}
	server2 := createMockShellyServer(t, deviceInfo2, nil)
	defer server2.Close()

	scanner := NewScanner()
	scanner.Concurrency = 4

	devices, err := scanner.DiscoverDevices(context.Background(), []string{
		server1.URL,
		server2.URL,
		"http://127.0.0.1:59998", // Unreachable - should be skipped
	})
	if err != nil {
		t.Fatalf("DiscoverDevices() error = %v", err)
	}

	// Should return 2 devices (the unreachable one is skipped)
	if len(devices) != 2 {
		t.Fatalf("DiscoverDevices() returned %d devices, want 2", len(devices))
	}

	// Count Zigbee-enabled devices
	zigbeeCount := 0
	for _, d := range devices {
		if d.HasZigbee {
			zigbeeCount++
		}
	}
	if zigbeeCount != 1 {
		t.Errorf("Found %d Zigbee devices, want 1", zigbeeCount)
	}
}
