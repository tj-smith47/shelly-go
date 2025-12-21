package matter

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

// mockTransport implements transport.Transport for testing
type mockTransport struct {
	callFunc  func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error)
	closeFunc func() error
}

func (m *mockTransport) Call(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
	if m.callFunc != nil {
		return m.callFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTransport) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// Helper function to create a proper JSON-RPC response
func jsonrpcResponse(result string) (json.RawMessage, error) {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  json.RawMessage(result),
	}
	return json.Marshal(response)
}

func TestMatter_GetConfig(t *testing.T) {
	tests := []struct {
		want     *Config
		name     string
		response string
		wantErr  bool
	}{
		{
			name:     "enabled",
			response: `{"enable": true}`,
			want: &Config{
				Enable: true,
			},
		},
		{
			name:     "disabled",
			response: `{"enable": false}`,
			want: &Config{
				Enable: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "Matter.GetConfig" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(tt.response)
				},
			}

			client := rpc.NewClient(mt)
			matter := NewMatter(client)

			got, err := matter.GetConfig(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.Enable != tt.want.Enable {
				t.Errorf("GetConfig().Enable = %v, want %v", got.Enable, tt.want.Enable)
			}
		})
	}
}

func TestMatter_SetConfig(t *testing.T) {
	tests := []struct {
		params  *SetConfigParams
		name    string
		wantErr bool
	}{
		{
			name: "enable",
			params: &SetConfigParams{
				Enable: boolPtr(true),
			},
		},
		{
			name: "disable",
			params: &SetConfigParams{
				Enable: boolPtr(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "Matter.SetConfig" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}

			client := rpc.NewClient(mt)
			matter := NewMatter(client)

			err := matter.SetConfig(context.Background(), tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMatter_GetStatus(t *testing.T) {
	tests := []struct {
		want     *Status
		name     string
		response string
		wantErr  bool
	}{
		{
			name:     "commissionable_no_fabrics",
			response: `{"commissionable": true, "fabrics_count": 0}`,
			want: &Status{
				Commissionable: true,
				FabricsCount:   0,
			},
		},
		{
			name:     "not_commissionable_with_fabrics",
			response: `{"commissionable": false, "fabrics_count": 3}`,
			want: &Status{
				Commissionable: false,
				FabricsCount:   3,
			},
		},
		{
			name:     "with_single_fabric",
			response: `{"commissionable": true, "fabrics_count": 1}`,
			want: &Status{
				Commissionable: true,
				FabricsCount:   1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "Matter.GetStatus" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(tt.response)
				},
			}

			client := rpc.NewClient(mt)
			matter := NewMatter(client)

			got, err := matter.GetStatus(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Commissionable != tt.want.Commissionable {
					t.Errorf("GetStatus().Commissionable = %v, want %v", got.Commissionable, tt.want.Commissionable)
				}
				if got.FabricsCount != tt.want.FabricsCount {
					t.Errorf("GetStatus().FabricsCount = %v, want %v", got.FabricsCount, tt.want.FabricsCount)
				}
			}
		})
	}
}

func TestMatter_FactoryReset(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "Matter.FactoryReset" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(mt)
	matter := NewMatter(client)

	err := matter.FactoryReset(context.Background())
	if err != nil {
		t.Errorf("FactoryReset() error = %v", err)
	}
}

func TestMatter_Enable(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "Matter.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}

	client := rpc.NewClient(mt)
	matter := NewMatter(client)

	err := matter.Enable(context.Background())
	if err != nil {
		t.Errorf("Enable() error = %v", err)
	}
}

func TestMatter_Disable(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "Matter.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}

	client := rpc.NewClient(mt)
	matter := NewMatter(client)

	err := matter.Disable(context.Background())
	if err != nil {
		t.Errorf("Disable() error = %v", err)
	}
}

func TestMatter_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
		wantErr  bool
	}{
		{
			name:     "enabled",
			response: `{"enable": true}`,
			want:     true,
		},
		{
			name:     "disabled",
			response: `{"enable": false}`,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "Matter.GetConfig" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(tt.response)
				},
			}

			client := rpc.NewClient(mt)
			matter := NewMatter(client)

			got, err := matter.IsEnabled(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("IsEnabled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatter_IsCommissionable(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
		wantErr  bool
	}{
		{
			name:     "commissionable",
			response: `{"commissionable": true, "fabrics_count": 0}`,
			want:     true,
		},
		{
			name:     "not_commissionable",
			response: `{"commissionable": false, "fabrics_count": 5}`,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "Matter.GetStatus" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(tt.response)
				},
			}

			client := rpc.NewClient(mt)
			matter := NewMatter(client)

			got, err := matter.IsCommissionable(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("IsCommissionable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("IsCommissionable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatter_GetFabricsCount(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     int
		wantErr  bool
	}{
		{
			name:     "no_fabrics",
			response: `{"commissionable": true, "fabrics_count": 0}`,
			want:     0,
		},
		{
			name:     "one_fabric",
			response: `{"commissionable": true, "fabrics_count": 1}`,
			want:     1,
		},
		{
			name:     "multiple_fabrics",
			response: `{"commissionable": false, "fabrics_count": 5}`,
			want:     5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "Matter.GetStatus" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(tt.response)
				},
			}

			client := rpc.NewClient(mt)
			matter := NewMatter(client)

			got, err := matter.GetFabricsCount(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFabricsCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("GetFabricsCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_UnmarshalJSON_WithRawFields(t *testing.T) {
	data := []byte(`{
		"enable": true,
		"unknown_field": "some_value"
	}`)

	var config Config
	err := json.Unmarshal(data, &config)
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
		return
	}

	if !config.Enable {
		t.Error("UnmarshalJSON() Enable should be true")
	}
}

func TestStatus_UnmarshalJSON_WithRawFields(t *testing.T) {
	data := []byte(`{
		"commissionable": true,
		"fabrics_count": 2,
		"future_field": 123
	}`)

	var status Status
	err := json.Unmarshal(data, &status)
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
		return
	}

	if !status.Commissionable {
		t.Error("UnmarshalJSON() Commissionable should be true")
	}
	if status.FabricsCount != 2 {
		t.Errorf("UnmarshalJSON() FabricsCount = %v, want 2", status.FabricsCount)
	}
}

func TestFabric_UnmarshalJSON(t *testing.T) {
	data := []byte(`{
		"fabric_id": "123456",
		"fabric_index": 1,
		"vendor_id": 4417,
		"label": "Apple Home",
		"extra_field": "value"
	}`)

	var fabric Fabric
	err := json.Unmarshal(data, &fabric)
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
		return
	}

	if fabric.FabricID != "123456" {
		t.Errorf("UnmarshalJSON() FabricID = %v, want 123456", fabric.FabricID)
	}
	if fabric.FabricIndex != 1 {
		t.Errorf("UnmarshalJSON() FabricIndex = %v, want 1", fabric.FabricIndex)
	}
	if fabric.VendorID != 4417 {
		t.Errorf("UnmarshalJSON() VendorID = %v, want 4417", fabric.VendorID)
	}
	if fabric.Label != "Apple Home" {
		t.Errorf("UnmarshalJSON() Label = %v, want Apple Home", fabric.Label)
	}
}

func TestCommissioningInfo_UnmarshalJSON(t *testing.T) {
	data := []byte(`{
		"qr_code": "MT:Y.K9042C00KA0648G00",
		"manual_code": "34970112332",
		"discriminator": 3840,
		"setup_pin_code": 20202021
	}`)

	var info CommissioningInfo
	err := json.Unmarshal(data, &info)
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
		return
	}

	if info.QRCode != "MT:Y.K9042C00KA0648G00" {
		t.Errorf("UnmarshalJSON() QRCode = %v", info.QRCode)
	}
	if info.ManualCode != "34970112332" {
		t.Errorf("UnmarshalJSON() ManualCode = %v", info.ManualCode)
	}
	if info.Discriminator != 3840 {
		t.Errorf("UnmarshalJSON() Discriminator = %v, want 3840", info.Discriminator)
	}
	if info.SetupPinCode != 20202021 {
		t.Errorf("UnmarshalJSON() SetupPinCode = %v, want 20202021", info.SetupPinCode)
	}
}

func TestMatter_ErrorHandling(t *testing.T) {
	errTest := errors.New("test error")

	tests := []struct {
		action  func(context.Context, *Matter) error
		name    string
		method  string
		wantErr bool
	}{
		{
			name:   "GetConfig_error",
			method: "Matter.GetConfig",
			action: func(ctx context.Context, m *Matter) error {
				_, err := m.GetConfig(ctx)
				return err
			},
			wantErr: true,
		},
		{
			name:   "GetStatus_error",
			method: "Matter.GetStatus",
			action: func(ctx context.Context, m *Matter) error {
				_, err := m.GetStatus(ctx)
				return err
			},
			wantErr: true,
		},
		{
			name:   "SetConfig_error",
			method: "Matter.SetConfig",
			action: func(ctx context.Context, m *Matter) error {
				return m.SetConfig(ctx, &SetConfigParams{Enable: boolPtr(true)})
			},
			wantErr: true,
		},
		{
			name:   "FactoryReset_error",
			method: "Matter.FactoryReset",
			action: func(ctx context.Context, m *Matter) error {
				return m.FactoryReset(ctx)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
					return nil, errTest
				},
			}

			client := rpc.NewClient(mt)
			matter := NewMatter(client)

			err := tt.action(context.Background(), matter)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}

func TestNewMatter(t *testing.T) {
	mt := &mockTransport{}
	client := rpc.NewClient(mt)
	matter := NewMatter(client)

	if matter == nil {
		t.Fatal("NewMatter returned nil")
	}
	if matter.client != client {
		t.Error("client not set correctly")
	}
}

func TestMatter_GetConfig_InvalidJSON(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{invalid json}`)
		},
	}

	client := rpc.NewClient(mt)
	matter := NewMatter(client)

	_, err := matter.GetConfig(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestMatter_GetStatus_InvalidJSON(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{invalid json}`)
		},
	}

	client := rpc.NewClient(mt)
	matter := NewMatter(client)

	_, err := matter.GetStatus(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestMatter_IsEnabled_Error(t *testing.T) {
	errTest := errors.New("test error")
	mt := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(mt)
	matter := NewMatter(client)

	_, err := matter.IsEnabled(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestMatter_IsCommissionable_Error(t *testing.T) {
	errTest := errors.New("test error")
	mt := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(mt)
	matter := NewMatter(client)

	_, err := matter.IsCommissionable(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestMatter_GetFabricsCount_Error(t *testing.T) {
	errTest := errors.New("test error")
	mt := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(mt)
	matter := NewMatter(client)

	_, err := matter.GetFabricsCount(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestMatter_Enable_Error(t *testing.T) {
	errTest := errors.New("test error")
	mt := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(mt)
	matter := NewMatter(client)

	err := matter.Enable(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestMatter_Disable_Error(t *testing.T) {
	errTest := errors.New("test error")
	mt := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(mt)
	matter := NewMatter(client)

	err := matter.Disable(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestSetConfigParams(t *testing.T) {
	enable := true
	params := SetConfigParams{
		Enable: &enable,
	}

	if params.Enable == nil {
		t.Error("Enable is nil")
	}
	if !*params.Enable {
		t.Error("Enable should be true")
	}
}

func TestConfig_Fields(t *testing.T) {
	config := Config{
		Enable: true,
	}

	if !config.Enable {
		t.Error("Enable should be true")
	}
}

func TestStatus_Fields(t *testing.T) {
	status := Status{
		Commissionable: true,
		FabricsCount:   3,
	}

	if !status.Commissionable {
		t.Error("Commissionable should be true")
	}
	if status.FabricsCount != 3 {
		t.Errorf("FabricsCount = %d, want 3", status.FabricsCount)
	}
}

func TestFabric_Fields(t *testing.T) {
	fabric := Fabric{
		FabricID:    "abc123",
		FabricIndex: 1,
		VendorID:    4417,
		Label:       "Apple Home",
	}

	if fabric.FabricID != "abc123" {
		t.Errorf("FabricID = %s, want abc123", fabric.FabricID)
	}
	if fabric.FabricIndex != 1 {
		t.Errorf("FabricIndex = %d, want 1", fabric.FabricIndex)
	}
	if fabric.VendorID != 4417 {
		t.Errorf("VendorID = %d, want 4417", fabric.VendorID)
	}
	if fabric.Label != "Apple Home" {
		t.Errorf("Label = %s, want Apple Home", fabric.Label)
	}
}

func TestCommissioningInfo_Fields(t *testing.T) {
	info := CommissioningInfo{
		QRCode:        "MT:Y.K9042C00KA0648G00",
		ManualCode:    "34970112332",
		Discriminator: 3840,
		SetupPinCode:  20202021,
	}

	if info.QRCode != "MT:Y.K9042C00KA0648G00" {
		t.Errorf("QRCode = %s", info.QRCode)
	}
	if info.ManualCode != "34970112332" {
		t.Errorf("ManualCode = %s", info.ManualCode)
	}
	if info.Discriminator != 3840 {
		t.Errorf("Discriminator = %d, want 3840", info.Discriminator)
	}
	if info.SetupPinCode != 20202021 {
		t.Errorf("SetupPinCode = %d, want 20202021", info.SetupPinCode)
	}
}
