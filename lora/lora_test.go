package lora

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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

func TestNewLoRa(t *testing.T) {
	transport := &mockTransport{}
	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	if lora == nil {
		t.Fatal("NewLoRa returned nil")
	}
	if lora.client != client {
		t.Error("client not set correctly")
	}
	if lora.id != 100 {
		t.Errorf("id = %d, want 100", lora.id)
	}
}

func TestLoRa_ID(t *testing.T) {
	transport := &mockTransport{}
	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	if lora.ID() != 100 {
		t.Errorf("ID() = %d, want 100", lora.ID())
	}
}

func TestLoRa_GetConfig(t *testing.T) {
	tests := []struct {
		want   *Config
		name   string
		result string
	}{
		{
			name:   "full config",
			result: `{"id":100,"freq":865000000,"bw":12,"dr":5,"plen":8,"txp":14}`,
			want: &Config{
				ID:   100,
				Freq: 865000000,
				BW:   12,
				DR:   5,
				Plen: 8,
				TxP:  14,
			},
		},
		{
			name:   "minimal config",
			result: `{"id":100}`,
			want:   &Config{ID: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "LoRa.GetConfig" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}

			client := rpc.NewClient(transport)
			lora := NewLoRa(client, 100)

			got, err := lora.GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if got.ID != tt.want.ID {
				t.Errorf("ID = %d, want %d", got.ID, tt.want.ID)
			}
			if got.Freq != tt.want.Freq {
				t.Errorf("Freq = %d, want %d", got.Freq, tt.want.Freq)
			}
			if got.BW != tt.want.BW {
				t.Errorf("BW = %d, want %d", got.BW, tt.want.BW)
			}
			if got.DR != tt.want.DR {
				t.Errorf("DR = %d, want %d", got.DR, tt.want.DR)
			}
			if got.Plen != tt.want.Plen {
				t.Errorf("Plen = %d, want %d", got.Plen, tt.want.Plen)
			}
			if got.TxP != tt.want.TxP {
				t.Errorf("TxP = %d, want %d", got.TxP, tt.want.TxP)
			}
		})
	}
}

func TestLoRa_GetConfig_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.GetConfig(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLoRa_GetConfig_InvalidJSON(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{invalid json}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.GetConfig(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestLoRa_SetConfig(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "LoRa.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	freq := int64(868000000)
	txp := 10
	err := lora.SetConfig(context.Background(), &SetConfigParams{
		Freq: &freq,
		TxP:  &txp,
	})
	if err != nil {
		t.Errorf("SetConfig() error = %v", err)
	}
}

func TestLoRa_SetConfig_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	freq := int64(868000000)
	err := lora.SetConfig(context.Background(), &SetConfigParams{Freq: &freq})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLoRa_GetStatus(t *testing.T) {
	tests := []struct {
		want   *Status
		name   string
		result string
	}{
		{
			name:   "with signal info",
			result: `{"id":100,"rssi":-75,"snr":8.5}`,
			want:   &Status{ID: 100, RSSI: -75, SNR: 8.5},
		},
		{
			name:   "no signal info",
			result: `{"id":100}`,
			want:   &Status{ID: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "LoRa.GetStatus" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}

			client := rpc.NewClient(transport)
			lora := NewLoRa(client, 100)

			got, err := lora.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if got.ID != tt.want.ID {
				t.Errorf("ID = %d, want %d", got.ID, tt.want.ID)
			}
			if got.RSSI != tt.want.RSSI {
				t.Errorf("RSSI = %d, want %d", got.RSSI, tt.want.RSSI)
			}
			if got.SNR != tt.want.SNR {
				t.Errorf("SNR = %f, want %f", got.SNR, tt.want.SNR)
			}
		})
	}
}

func TestLoRa_GetStatus_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.GetStatus(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLoRa_GetStatus_InvalidJSON(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{invalid json}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.GetStatus(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestLoRa_SendBytes(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "LoRa.SendBytes" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	err := lora.SendBytes(context.Background(), "SGVsbG8gV29ybGQh") // "Hello World!" in base64
	if err != nil {
		t.Errorf("SendBytes() error = %v", err)
	}
}

func TestLoRa_SendBytes_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	err := lora.SendBytes(context.Background(), "SGVsbG8h")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLoRa_SendString(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "LoRa.SendBytes" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	err := lora.SendString(context.Background(), "Hello!")
	if err != nil {
		t.Errorf("SendString() error = %v", err)
	}
}

func TestLoRa_SendRaw(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "LoRa.SendBytes" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	err := lora.SendRaw(context.Background(), []byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Errorf("SendRaw() error = %v", err)
	}
}

func TestLoRa_GetAddOnInfo(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "AddOn.GetInfo" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"type":"LoRa","version":"1.2.3"}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	info, err := lora.GetAddOnInfo(context.Background())
	if err != nil {
		t.Errorf("GetAddOnInfo() error = %v", err)
		return
	}

	if info.Type != "LoRa" {
		t.Errorf("Type = %s, want LoRa", info.Type)
	}
	if info.Version != "1.2.3" {
		t.Errorf("Version = %s, want 1.2.3", info.Version)
	}
}

func TestLoRa_GetAddOnInfo_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.GetAddOnInfo(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLoRa_GetAddOnInfo_InvalidJSON(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{invalid json}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.GetAddOnInfo(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestLoRa_CheckForUpdate(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{name: "update available", result: `{"update_available":true}`, want: true},
		{name: "no update", result: `{"update_available":false}`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
					if method != "AddOn.CheckForUpdate" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}

			client := rpc.NewClient(transport)
			lora := NewLoRa(client, 100)

			got, err := lora.CheckForUpdate(context.Background())
			if err != nil {
				t.Errorf("CheckForUpdate() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("CheckForUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoRa_CheckForUpdate_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.CheckForUpdate(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLoRa_CheckForUpdate_InvalidJSON(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{invalid json}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.CheckForUpdate(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestLoRa_Update(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "AddOn.Update" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	err := lora.Update(context.Background())
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}
}

func TestLoRa_Update_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	err := lora.Update(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLoRa_SetFrequency(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "LoRa.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	err := lora.SetFrequency(context.Background(), 868000000)
	if err != nil {
		t.Errorf("SetFrequency() error = %v", err)
	}
}

func TestLoRa_SetTransmitPower(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "LoRa.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	err := lora.SetTransmitPower(context.Background(), 10)
	if err != nil {
		t.Errorf("SetTransmitPower() error = %v", err)
	}
}

func TestLoRa_SetDataRate(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "LoRa.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	err := lora.SetDataRate(context.Background(), 7)
	if err != nil {
		t.Errorf("SetDataRate() error = %v", err)
	}
}

func TestLoRa_SetBandwidth(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				method := req.GetMethod()
			if method != "LoRa.SetConfig" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	err := lora.SetBandwidth(context.Background(), 125)
	if err != nil {
		t.Errorf("SetBandwidth() error = %v", err)
	}
}

func TestLoRa_GetFrequency(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{"id":100,"freq":865000000}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	freq, err := lora.GetFrequency(context.Background())
	if err != nil {
		t.Errorf("GetFrequency() error = %v", err)
		return
	}
	if freq != 865000000 {
		t.Errorf("GetFrequency() = %d, want 865000000", freq)
	}
}

func TestLoRa_GetFrequency_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.GetFrequency(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLoRa_GetTransmitPower(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{"id":100,"txp":14}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	txp, err := lora.GetTransmitPower(context.Background())
	if err != nil {
		t.Errorf("GetTransmitPower() error = %v", err)
		return
	}
	if txp != 14 {
		t.Errorf("GetTransmitPower() = %d, want 14", txp)
	}
}

func TestLoRa_GetTransmitPower_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.GetTransmitPower(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLoRa_GetLastRSSI(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{"id":100,"rssi":-75}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	rssi, err := lora.GetLastRSSI(context.Background())
	if err != nil {
		t.Errorf("GetLastRSSI() error = %v", err)
		return
	}
	if rssi != -75 {
		t.Errorf("GetLastRSSI() = %d, want -75", rssi)
	}
}

func TestLoRa_GetLastRSSI_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.GetLastRSSI(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLoRa_GetLastSNR(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return jsonrpcResponse(`{"id":100,"snr":8.5}`)
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	snr, err := lora.GetLastSNR(context.Background())
	if err != nil {
		t.Errorf("GetLastSNR() error = %v", err)
		return
	}
	if snr != 8.5 {
		t.Errorf("GetLastSNR() = %f, want 8.5", snr)
	}
}

func TestLoRa_GetLastSNR_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				_ = req.GetMethod()
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	lora := NewLoRa(client, 100)

	_, err := lora.GetLastSNR(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestEncodeBase64(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		input []byte
	}{
		{name: "hello", input: []byte("Hello"), want: "SGVsbG8="},
		{name: "hello world", input: []byte("Hello World!"), want: "SGVsbG8gV29ybGQh"},
		{name: "empty", input: []byte{}, want: ""},
		{name: "single byte", input: []byte{0x41}, want: "QQ=="},
		{name: "two bytes", input: []byte{0x41, 0x42}, want: "QUI="},
		{name: "three bytes", input: []byte{0x41, 0x42, 0x43}, want: "QUJD"},
		{name: "binary data", input: []byte{0x00, 0xFF, 0x7F}, want: "AP9/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeBase64(tt.input)
			if got != tt.want {
				t.Errorf("encodeBase64(%v) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

// Test Device Registry

func TestNewDeviceRegistry(t *testing.T) {
	registry := NewDeviceRegistry()
	if registry == nil {
		t.Fatal("NewDeviceRegistry returned nil")
	}
	if registry.devices == nil {
		t.Error("devices map is nil")
	}
	if registry.OnlineTimeout != 5*60*1000000000 { // 5 minutes in nanoseconds
		t.Errorf("OnlineTimeout = %v, want 5 minutes", registry.OnlineTimeout)
	}
}

func TestDeviceRegistry_Register(t *testing.T) {
	registry := NewDeviceRegistry()

	device := &RegisteredDevice{
		DeviceID: "sensor1",
		Name:     "Temperature Sensor",
		Group:    "sensors",
	}

	err := registry.Register(device)
	if err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	// Duplicate registration should fail
	err = registry.Register(device)
	if !errors.Is(err, ErrDeviceAlreadyRegistered) {
		t.Errorf("Register() error = %v, want ErrDeviceAlreadyRegistered", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}
}

func TestDeviceRegistry_RegisterOrUpdate(t *testing.T) {
	registry := NewDeviceRegistry()

	device := &RegisteredDevice{
		DeviceID: "sensor1",
		Name:     "Temperature Sensor",
	}

	registry.RegisterOrUpdate(device)
	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}

	// Update existing device
	updated := &RegisteredDevice{
		DeviceID: "sensor1",
		Name:     "Updated Sensor",
		Group:    "sensors",
	}
	registry.RegisterOrUpdate(updated)

	got, err := registry.Get("sensor1")
	if err != nil {
		t.Errorf("Get() error = %v", err)
		return
	}
	if got.Name != "Updated Sensor" {
		t.Errorf("Name = %s, want Updated Sensor", got.Name)
	}
	if got.Group != "sensors" {
		t.Errorf("Group = %s, want sensors", got.Group)
	}
}

func TestDeviceRegistry_RegisterOrUpdate_Metadata(t *testing.T) {
	registry := NewDeviceRegistry()

	device := &RegisteredDevice{
		DeviceID: "sensor1",
		Metadata: map[string]any{"key1": "value1"},
	}
	registry.RegisterOrUpdate(device)

	updated := &RegisteredDevice{
		DeviceID: "sensor1",
		Metadata: map[string]any{"key2": "value2"},
	}
	registry.RegisterOrUpdate(updated)

	got, err := registry.Get("sensor1")
	if err != nil {
		t.Errorf("Get() error = %v", err)
		return
	}
	if got.Metadata["key1"] != "value1" {
		t.Error("key1 should be preserved")
	}
	if got.Metadata["key2"] != "value2" {
		t.Error("key2 should be added")
	}
}

func TestDeviceRegistry_Unregister(t *testing.T) {
	registry := NewDeviceRegistry()

	device := &RegisteredDevice{DeviceID: "sensor1"}
	if err := registry.Register(device); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	err := registry.Unregister("sensor1")
	if err != nil {
		t.Errorf("Unregister() error = %v", err)
	}

	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}

	// Unregister non-existent device
	err = registry.Unregister("nonexistent")
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Errorf("Unregister() error = %v, want ErrDeviceNotFound", err)
	}
}

func TestDeviceRegistry_Get(t *testing.T) {
	registry := NewDeviceRegistry()

	device := &RegisteredDevice{DeviceID: "sensor1", Name: "Test"}
	if err := registry.Register(device); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	got, err := registry.Get("sensor1")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if got.Name != "Test" {
		t.Errorf("Name = %s, want Test", got.Name)
	}

	_, err = registry.Get("nonexistent")
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Errorf("Get() error = %v, want ErrDeviceNotFound", err)
	}
}

func TestDeviceRegistry_GetAll(t *testing.T) {
	registry := NewDeviceRegistry()

	if err := registry.Register(&RegisteredDevice{DeviceID: "sensor1"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}
	if err := registry.Register(&RegisteredDevice{DeviceID: "sensor2"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	all := registry.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll() returned %d devices, want 2", len(all))
	}
}

func TestDeviceRegistry_GetByGroup(t *testing.T) {
	registry := NewDeviceRegistry()

	if err := registry.Register(&RegisteredDevice{DeviceID: "sensor1", Group: "sensors"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}
	if err := registry.Register(&RegisteredDevice{DeviceID: "sensor2", Group: "sensors"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}
	if err := registry.Register(&RegisteredDevice{DeviceID: "switch1", Group: "switches"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	sensors := registry.GetByGroup("sensors")
	if len(sensors) != 2 {
		t.Errorf("GetByGroup(sensors) returned %d, want 2", len(sensors))
	}

	switches := registry.GetByGroup("switches")
	if len(switches) != 1 {
		t.Errorf("GetByGroup(switches) returned %d, want 1", len(switches))
	}

	empty := registry.GetByGroup("nonexistent")
	if len(empty) != 0 {
		t.Errorf("GetByGroup(nonexistent) returned %d, want 0", len(empty))
	}
}

func TestDeviceRegistry_GetOnline(t *testing.T) {
	registry := NewDeviceRegistry()
	registry.OnlineTimeout = 1 * 60 * 1000000000 // 1 minute in nanoseconds

	if err := registry.Register(&RegisteredDevice{DeviceID: "sensor1", Online: true}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}
	if err := registry.Register(&RegisteredDevice{DeviceID: "sensor2", Online: false}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	online := registry.GetOnline()
	if len(online) != 1 {
		t.Errorf("GetOnline() returned %d, want 1", len(online))
	}
}

func TestDeviceRegistry_UpdateLastSeen(t *testing.T) {
	registry := NewDeviceRegistry()

	if err := registry.Register(&RegisteredDevice{DeviceID: "sensor1"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	err := registry.UpdateLastSeen("sensor1", -75, 8.5)
	if err != nil {
		t.Errorf("UpdateLastSeen() error = %v", err)
		return
	}

	device, err := registry.Get("sensor1")
	if err != nil {
		t.Errorf("Get() error = %v", err)
		return
	}
	if device.LastRSSI != -75 {
		t.Errorf("LastRSSI = %d, want -75", device.LastRSSI)
	}
	if device.LastSNR != 8.5 {
		t.Errorf("LastSNR = %f, want 8.5", device.LastSNR)
	}
	if !device.Online {
		t.Error("Online should be true")
	}

	err = registry.UpdateLastSeen("nonexistent", 0, 0)
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Errorf("UpdateLastSeen() error = %v, want ErrDeviceNotFound", err)
	}
}

func TestDeviceRegistry_SetOnline(t *testing.T) {
	registry := NewDeviceRegistry()

	if err := registry.Register(&RegisteredDevice{DeviceID: "sensor1", Online: true}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	err := registry.SetOnline("sensor1", false)
	if err != nil {
		t.Errorf("SetOnline() error = %v", err)
		return
	}

	device, err := registry.Get("sensor1")
	if err != nil {
		t.Errorf("Get() error = %v", err)
		return
	}
	if device.Online {
		t.Error("Online should be false")
	}

	err = registry.SetOnline("nonexistent", true)
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Errorf("SetOnline() error = %v, want ErrDeviceNotFound", err)
	}
}

func TestDeviceRegistry_Clear(t *testing.T) {
	registry := NewDeviceRegistry()

	if err := registry.Register(&RegisteredDevice{DeviceID: "sensor1"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}
	if err := registry.Register(&RegisteredDevice{DeviceID: "sensor2"}); err != nil {
		t.Errorf("Register() error = %v", err)
		return
	}

	registry.Clear()
	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}
}

// Test Message Router

func TestNewMessageRouter(t *testing.T) {
	router := NewMessageRouter()
	if router == nil {
		t.Fatal("NewMessageRouter returned nil")
	}
	if router.handlers == nil {
		t.Error("handlers is nil")
	}
}

func TestMessageRouter_Handle(t *testing.T) {
	router := NewMessageRouter()

	var called bool
	router.Handle(func(msg *RoutedMessage) {
		called = true
	}, nil)

	if router.HandlerCount() != 1 {
		t.Errorf("HandlerCount() = %d, want 1", router.HandlerCount())
	}

	msg := &RoutedMessage{FromDevice: "sensor1", Data: []byte("test")}
	count, err := router.Route(msg)
	if err != nil {
		t.Errorf("Route() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Route() count = %d, want 1", count)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestMessageRouter_HandleDevice(t *testing.T) {
	router := NewMessageRouter()

	var receivedMsg *RoutedMessage
	router.HandleDevice("sensor1", func(msg *RoutedMessage) {
		receivedMsg = msg
	})

	// Message from sensor1 should be handled
	msg1 := &RoutedMessage{FromDevice: "sensor1", Data: []byte("test1")}
	count, err := router.Route(msg1)
	if err != nil {
		t.Errorf("Route() error = %v", err)
		return
	}
	if count != 1 {
		t.Errorf("Route() count = %d, want 1", count)
	}
	if receivedMsg == nil || string(receivedMsg.Data) != "test1" {
		t.Error("handler was not called with correct message")
	}

	// Message from sensor2 should not be handled
	receivedMsg = nil
	msg2 := &RoutedMessage{FromDevice: "sensor2", Data: []byte("test2")}
	count, err = router.Route(msg2)
	if err != nil {
		t.Errorf("Route() error = %v", err)
		return
	}
	if count != 0 {
		t.Errorf("Route() count = %d, want 0", count)
	}
	if receivedMsg != nil {
		t.Error("handler should not be called for different device")
	}
}

func TestMessageRouter_HandleGroup(t *testing.T) {
	router := NewMessageRouter()

	var callCount int
	router.HandleGroup("sensors", func(msg *RoutedMessage) {
		callCount++
	})

	// Message to sensors group should be handled
	msg1 := &RoutedMessage{Group: "sensors", Data: []byte("test")}
	if _, err := router.Route(msg1); err != nil {
		t.Errorf("Route() error = %v", err)
		return
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	// Message to different group should not be handled
	msg2 := &RoutedMessage{Group: "switches", Data: []byte("test")}
	if _, err := router.Route(msg2); err != nil {
		t.Errorf("Route() error = %v", err)
		return
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestMessageRouter_Stop(t *testing.T) {
	router := NewMessageRouter()
	router.Handle(func(msg *RoutedMessage) {}, nil)

	router.Stop()
	if !router.IsStopped() {
		t.Error("IsStopped() should return true")
	}

	msg := &RoutedMessage{Data: []byte("test")}
	_, err := router.Route(msg)
	if !errors.Is(err, ErrRouterStopped) {
		t.Errorf("Route() error = %v, want ErrRouterStopped", err)
	}

	router.Start()
	if router.IsStopped() {
		t.Error("IsStopped() should return false after Start()")
	}
}

func TestMessageRouter_ClearHandlers(t *testing.T) {
	router := NewMessageRouter()
	router.Handle(func(msg *RoutedMessage) {}, nil)
	router.Handle(func(msg *RoutedMessage) {}, nil)

	if router.HandlerCount() != 2 {
		t.Errorf("HandlerCount() = %d, want 2", router.HandlerCount())
	}

	router.ClearHandlers()
	if router.HandlerCount() != 0 {
		t.Errorf("HandlerCount() = %d, want 0", router.HandlerCount())
	}
}

func TestMessageRouter_AutoRegister(t *testing.T) {
	registry := NewDeviceRegistry()
	router := NewMessageRouter()
	router.Registry = registry
	router.AutoRegister = true

	router.Handle(func(msg *RoutedMessage) {}, nil)

	msg := &RoutedMessage{
		FromDevice: "new-sensor",
		Data:       []byte("test"),
		RSSI:       -70,
		SNR:        9.0,
		Timestamp:  1234567890,
	}
	if _, err := router.Route(msg); err != nil {
		t.Errorf("Route() error = %v", err)
		return
	}

	// Device should be auto-registered
	device, err := registry.Get("new-sensor")
	if err != nil {
		t.Errorf("Device was not auto-registered: %v", err)
	}
	if device.LastRSSI != -70 {
		t.Errorf("LastRSSI = %d, want -70", device.LastRSSI)
	}
}

func TestMessageRouter_RouteEvent(t *testing.T) {
	router := NewMessageRouter()

	var receivedMsg *RoutedMessage
	router.Handle(func(msg *RoutedMessage) {
		receivedMsg = msg
	}, nil)

	event := &Event{
		Component: "lora:100",
		Event:     "lora",
		Info: ReceivedData{
			Data: "SGVsbG8h", // "Hello!" in base64
			RSSI: -75,
			SNR:  8.5,
			TS:   1234567890,
		},
	}

	count, err := router.RouteEvent(event, "sensor1")
	if err != nil {
		t.Errorf("RouteEvent() error = %v", err)
	}
	if count != 1 {
		t.Errorf("RouteEvent() count = %d, want 1", count)
	}
	if receivedMsg == nil {
		t.Fatal("handler was not called")
	}
	if receivedMsg.FromDevice != "sensor1" {
		t.Errorf("FromDevice = %s, want sensor1", receivedMsg.FromDevice)
	}
	if string(receivedMsg.Data) != "Hello!" {
		t.Errorf("Data = %s, want Hello!", string(receivedMsg.Data))
	}
}

func TestMessageFilter_Match(t *testing.T) {
	tests := []struct {
		name   string
		filter MessageFilter
		msg    RoutedMessage
		want   bool
	}{
		{
			name:   "empty filter matches all",
			filter: MessageFilter{},
			msg:    RoutedMessage{FromDevice: "any", Group: "any"},
			want:   true,
		},
		{
			name:   "device filter matches",
			filter: MessageFilter{FromDevice: "sensor1"},
			msg:    RoutedMessage{FromDevice: "sensor1"},
			want:   true,
		},
		{
			name:   "device filter no match",
			filter: MessageFilter{FromDevice: "sensor1"},
			msg:    RoutedMessage{FromDevice: "sensor2"},
			want:   false,
		},
		{
			name:   "group filter matches",
			filter: MessageFilter{Group: "sensors"},
			msg:    RoutedMessage{Group: "sensors"},
			want:   true,
		},
		{
			name:   "group filter no match",
			filter: MessageFilter{Group: "sensors"},
			msg:    RoutedMessage{Group: "switches"},
			want:   false,
		},
		{
			name:   "custom filter matches",
			filter: MessageFilter{Custom: func(msg *RoutedMessage) bool { return len(msg.Data) > 0 }},
			msg:    RoutedMessage{Data: []byte("test")},
			want:   true,
		},
		{
			name:   "custom filter no match",
			filter: MessageFilter{Custom: func(msg *RoutedMessage) bool { return len(msg.Data) > 0 }},
			msg:    RoutedMessage{Data: []byte{}},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Match(&tt.msg)
			if got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeBase64(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "hello", input: "SGVsbG8=", want: "Hello"},
		{name: "hello world", input: "SGVsbG8gV29ybGQh", want: "Hello World!"},
		{name: "empty", input: "", want: ""},
		{name: "single byte", input: "QQ==", want: "A"},
		{name: "two bytes", input: "QUI=", want: "AB"},
		{name: "three bytes", input: "QUJD", want: "ABC"},
		{name: "invalid char", input: "!!!", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeBase64(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeBase64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("decodeBase64(%s) = %s, want %s", tt.input, string(got), tt.want)
			}
		})
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrDeviceNotFound == nil {
		t.Error("ErrDeviceNotFound is nil")
	}
	if ErrDeviceAlreadyRegistered == nil {
		t.Error("ErrDeviceAlreadyRegistered is nil")
	}
	if ErrRouterStopped == nil {
		t.Error("ErrRouterStopped is nil")
	}
	if ErrNoHandler == nil {
		t.Error("ErrNoHandler is nil")
	}
}

func TestRegisteredDevice_Fields(t *testing.T) {
	device := RegisteredDevice{
		DeviceID: "sensor1",
		Address:  "10.23.47.220",
		Name:     "Temperature Sensor",
		Group:    "sensors",
		LastSeen: 1234567890,
		LastRSSI: -75,
		LastSNR:  8.5,
		Online:   true,
		Metadata: map[string]any{"key": "value"},
	}

	if device.DeviceID != "sensor1" {
		t.Errorf("DeviceID = %s, want sensor1", device.DeviceID)
	}
	if device.Group != "sensors" {
		t.Errorf("Group = %s, want sensors", device.Group)
	}
	if device.Metadata["key"] != "value" {
		t.Error("Metadata not set correctly")
	}
}

func TestRoutedMessage_Fields(t *testing.T) {
	msg := RoutedMessage{
		FromDevice: "sensor1",
		ToDevice:   "gateway",
		Group:      "sensors",
		Data:       []byte("test"),
		RSSI:       -75,
		SNR:        8.5,
		Timestamp:  1234567890,
	}

	if msg.FromDevice != "sensor1" {
		t.Errorf("FromDevice = %s, want sensor1", msg.FromDevice)
	}
	if msg.ToDevice != "gateway" {
		t.Errorf("ToDevice = %s, want gateway", msg.ToDevice)
	}
	if string(msg.Data) != "test" {
		t.Errorf("Data = %s, want test", string(msg.Data))
	}
}
