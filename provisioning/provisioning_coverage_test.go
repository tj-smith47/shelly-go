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

// jsonrpcSuccessRaw returns a valid JSON-RPC envelope whose result field is
// the provided raw bytes verbatim.  This lets callers inject a structurally
// valid JSON-RPC response whose result cannot be unmarshalled into a typed
// struct (e.g. a JSON array instead of an object), exercising the
// json.Unmarshal error branches inside GetDeviceInfo and GetWiFiStatus.
func jsonrpcSuccessRaw(resultRaw string) (json.RawMessage, error) {
	// Build manually so the outer envelope is always valid JSON-RPC while the
	// inner result may be anything.
	envelope := `{"jsonrpc":"2.0","id":1,"result":` + resultRaw + `}`
	return json.RawMessage(envelope), nil
}

// ── ConfigureWiFi: Enable field set (line 86) ─────────────────────────────────

func TestProvisioner_ConfigureWiFi_ExplicitEnable(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(_ context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			if req.GetMethod() != methodWiFiSetConfig {
				t.Errorf("unexpected method: %s", req.GetMethod())
			}
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(tr)
	prov := New(client)

	enable := false
	err := prov.ConfigureWiFi(context.Background(), &WiFiConfig{
		SSID:   "TestNet",
		Enable: &enable,
	})
	if err != nil {
		t.Errorf("ConfigureWiFi() error = %v", err)
	}
}

// ── GetDeviceInfo: unmarshal error path (line 243-244) ────────────────────────

func TestProvisioner_GetDeviceInfo_UnmarshalError(t *testing.T) {
	// Return a JSON array — valid JSON but cannot unmarshal into DeviceInfo.
	tr := &mockTransport{
		callFunc: func(_ context.Context, _ transport.RPCRequest) (json.RawMessage, error) {
			return jsonrpcSuccessRaw(`["not","an","object"]`)
		},
	}

	client := rpc.NewClient(tr)
	prov := New(client)

	_, err := prov.GetDeviceInfo(context.Background())
	if err == nil {
		t.Error("GetDeviceInfo() should error when result cannot be unmarshalled")
	}
}

// ── GetWiFiStatus: unmarshal error path (line 257-258) ────────────────────────

func TestProvisioner_GetWiFiStatus_UnmarshalError(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(_ context.Context, _ transport.RPCRequest) (json.RawMessage, error) {
			return jsonrpcSuccessRaw(`["not","an","object"]`)
		},
	}

	client := rpc.NewClient(tr)
	prov := New(client)

	_, err := prov.GetWiFiStatus(context.Background())
	if err == nil {
		t.Error("GetWiFiStatus() should error when result cannot be unmarshalled")
	}
}

// ── transmitCommands: connect error (line 775) ────────────────────────────────

func TestBLEProvisioner_transmitCommands_ConnectError(t *testing.T) {
	b := NewBLEProvisioner()
	mock := newMockBLETransmitter()
	mock.connectErr = errors.New("no adapter")
	b.Transmitter = mock

	b.AddDiscoveredDevice(&BLEDevice{
		Name:    "ShellyPlus1-AABBCC",
		Address: "AA:BB:CC:DD:EE:FF",
	})

	cmds := []BLERPCCommand{{Method: "Shelly.GetDeviceInfo"}}
	err := b.transmitCommands(context.Background(), "AA:BB:CC:DD:EE:FF", cmds)
	if err == nil {
		t.Fatal("transmitCommands should fail when Connect returns error")
	}
	if !errors.Is(err, ErrBLEConnectionFailed) {
		t.Errorf("err = %v, want wrapping ErrBLEConnectionFailed", err)
	}
}

// ── transmitCommands: write error (line 793-794) ──────────────────────────────

func TestBLEProvisioner_transmitCommands_WriteError(t *testing.T) {
	b := NewBLEProvisioner()
	mock := newMockBLETransmitter()
	mock.connected = true
	mock.writeErr = errors.New("characteristic unavailable")
	b.Transmitter = mock

	// Make mock skip Connect (already marked connected)
	mock.connectErr = nil

	cmds := []BLERPCCommand{{Method: "WiFi.SetConfig", Params: map[string]any{"ssid": "x"}}}
	err := b.transmitCommands(context.Background(), "AA:BB:CC:DD:EE:FF", cmds)
	if err == nil {
		t.Fatal("transmitCommands should fail when WriteCharacteristic returns error")
	}
	if !errors.Is(err, ErrBLEWriteFailed) {
		t.Errorf("err = %v, want wrapping ErrBLEWriteFailed", err)
	}
}

// ── transmitCommands: read error continues (line 799-803) ────────────────────

func TestBLEProvisioner_transmitCommands_ReadError_Continues(t *testing.T) {
	b := NewBLEProvisioner()
	mock := newMockBLETransmitter()
	mock.connected = true
	mock.readErr = errors.New("no notification")
	b.Transmitter = mock

	// Two commands: both writes succeed, read fails → loop continues, function
	// returns nil because read errors are non-fatal by design.
	cmds := []BLERPCCommand{
		{Method: "WiFi.SetConfig"},
		{Method: "Cloud.SetConfig"},
	}
	err := b.transmitCommands(context.Background(), "AA:BB:CC:DD:EE:FF", cmds)
	if err != nil {
		t.Errorf("transmitCommands should not fail on read error; got %v", err)
	}
	if len(mock.writtenData) != 2 {
		t.Errorf("expected 2 writes, got %d", len(mock.writtenData))
	}
}

// ── transmitCommands: disconnect error (deferred, non-fatal) ─────────────────

func TestBLEProvisioner_transmitCommands_DisconnectError(t *testing.T) {
	b := NewBLEProvisioner()
	mock := newMockBLETransmitter()
	mock.connected = true
	mock.disconnectErr = errors.New("disconnect failed")
	// Provide a notification so ReadNotification succeeds for the one command.
	mock.SetNotifications([]byte(`{"id":1}`))
	b.Transmitter = mock

	cmds := []BLERPCCommand{{Method: "Shelly.GetDeviceInfo"}}
	// Disconnect error is non-fatal; transmitCommands should still succeed.
	err := b.transmitCommands(context.Background(), "AA:BB:CC:DD:EE:FF", cmds)
	if err != nil {
		t.Errorf("transmitCommands should succeed despite disconnect error; got %v", err)
	}
}

// ── ProvisionBulk: concurrency <= 0 → clamps to 1 (line 931) ─────────────────

func TestBulkProvisioner_ProvisionBulk_ZeroConcurrency(t *testing.T) {
	factory := func(address string) (*rpc.Client, error) {
		tr := &mockTransport{
			callFunc: func(_ context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				switch req.GetMethod() {
				case "Shelly.GetDeviceInfo":
					return jsonrpcResponse(`{"id":"test"}`)
				case "WiFi.SetConfig":
					return jsonrpcResponse(`{"restart_required":false}`)
				default:
					return jsonrpcResponse(`null`)
				}
			},
		}
		return rpc.NewClient(tr), nil
	}

	b := NewBulkProvisioner(factory)
	b.Concurrency = 0 // triggers the clamp to 1
	b.RetryCount = 0

	targets := []*BulkProvisionTarget{
		{
			Address: "192.168.1.100",
			Config:  &DeviceConfig{WiFi: &WiFiConfig{SSID: "Net"}},
		},
	}
	opts := &ProvisionOptions{WaitForConnection: false}

	result, err := b.ProvisionBulk(context.Background(), targets, nil, opts)
	if err != nil {
		t.Fatalf("ProvisionBulk() error = %v", err)
	}
	if result.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", result.SuccessCount)
	}
}

// ── provisionWithRetry: negative RetryCount clamps to 0 (line 1032) ──────────

func TestBulkProvisioner_provisionWithRetry_NegativeRetryCount(t *testing.T) {
	factory := func(address string) (*rpc.Client, error) {
		tr := &mockTransport{
			callFunc: func(_ context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				switch req.GetMethod() {
				case "Shelly.GetDeviceInfo":
					return jsonrpcResponse(`{"id":"test"}`)
				case "WiFi.SetConfig":
					return jsonrpcResponse(`{"restart_required":false}`)
				default:
					return jsonrpcResponse(`null`)
				}
			},
		}
		return rpc.NewClient(tr), nil
	}

	b := NewBulkProvisioner(factory)
	b.RetryCount = -5 // triggers clamp to 0

	result := b.provisionWithRetry(
		context.Background(),
		"192.168.1.100",
		&DeviceConfig{WiFi: &WiFiConfig{SSID: "Net"}},
		&ProvisionOptions{WaitForConnection: false},
	)
	if result == nil {
		t.Fatal("provisionWithRetry returned nil")
	}
	if !result.Success {
		t.Errorf("provisionWithRetry Success = false; err = %v", result.Error)
	}
}

// ── provisionWithRetry: ctx canceled during retry delay (line 1038-1043) ─────

func TestBulkProvisioner_provisionWithRetry_ContextCanceledDuringDelay(t *testing.T) {
	callCount := 0
	factory := func(address string) (*rpc.Client, error) {
		callCount++
		return nil, errors.New("always fails")
	}

	b := NewBulkProvisioner(factory)
	b.RetryCount = 3
	b.RetryDelay = 10 * time.Second // long enough that ctx cancels first

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after first attempt completes so the retry delay is interrupted.
	go func() {
		// Give the first attempt time to run then cancel.
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	result := b.provisionWithRetry(
		ctx,
		"192.168.1.100",
		&DeviceConfig{WiFi: &WiFiConfig{SSID: "Net"}},
		&ProvisionOptions{WaitForConnection: false},
	)
	if result == nil {
		t.Fatal("provisionWithRetry returned nil")
	}
	if result.Success {
		t.Error("provisionWithRetry should not succeed when context canceled")
	}
	if !errors.Is(result.Error, context.Canceled) {
		t.Errorf("result.Error = %v, want context.Canceled", result.Error)
	}
}

// ── provisionWithRetry: lastResult nil after Provision returns err, nil ───────

func TestBulkProvisioner_provisionWithRetry_NilResultFromProvision(t *testing.T) {
	// Provision returns (nil, err) when GetDeviceInfo fails with a nil result
	// from the provisioner.  This path covers the `lastResult == nil` guard at
	// line 1065.
	factory := func(address string) (*rpc.Client, error) {
		tr := &mockTransport{
			callFunc: func(_ context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				// GetDeviceInfo fails; Provision returns (result{Error:…}, err).
				return nil, errors.New("device unavailable")
			},
		}
		return rpc.NewClient(tr), nil
	}

	b := NewBulkProvisioner(factory)
	b.RetryCount = 0

	result := b.provisionWithRetry(
		context.Background(),
		"192.168.1.100",
		&DeviceConfig{WiFi: &WiFiConfig{SSID: "Net"}},
		&ProvisionOptions{WaitForConnection: false},
	)
	if result == nil {
		t.Fatal("provisionWithRetry returned nil")
	}
	if result.Success {
		t.Error("provisionWithRetry should not succeed")
	}
}
