package gen2

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

// mockTransport for testing
type mockTransport struct {
	lastCall struct {
		params any
		method string
	}
	err      error
	response []byte
}

func (m *mockTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	m.lastCall.method = method
	m.lastCall.params = params

	if m.err != nil {
		return nil, m.err
	}

	// Wrap response in JSON-RPC format
	rpcResponse := struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result"`
		ID      int             `json:"id"`
	}{
		JSONRPC: "2.0",
		ID:      1,
		Result:  m.response,
	}

	wrapped, err := json.Marshal(rpcResponse)
	if err != nil {
		return nil, err
	}

	return wrapped, nil
}

func (m *mockTransport) Close() error {
	return nil
}

func TestNewBaseComponent(t *testing.T) {
	mt := &mockTransport{}
	client := newTestClient(mt)

	comp := NewBaseComponent(client, "switch", 0)

	if comp == nil {
		t.Fatal("NewBaseComponent() returned nil")
	}

	if comp.Type() != "switch" {
		t.Errorf("Type() = %v, want switch", comp.Type())
	}

	if comp.ID() != 0 {
		t.Errorf("ID() = %v, want 0", comp.ID())
	}

	if comp.Key() != "switch:0" {
		t.Errorf("Key() = %v, want switch:0", comp.Key())
	}
}

func TestBaseComponent_GetConfig(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"id":0,"name":"My Switch","initial_state":"off"}`),
	}
	client := newTestClient(mt)

	comp := NewBaseComponent(client, "switch", 0)
	result, err := comp.GetConfig(context.Background())

	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(result, &config); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if config["name"] != "My Switch" {
		t.Errorf("config name = %v, want My Switch", config["name"])
	}
}

func TestBaseComponent_SetConfig(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"restart_required":false}`),
	}
	client := newTestClient(mt)

	comp := NewBaseComponent(client, "switch", 0)

	config := map[string]any{
		"name":          "New Name",
		"initial_state": "on",
	}

	err := comp.SetConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestBaseComponent_SetConfig_WithStruct(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"restart_required":false}`),
	}
	client := newTestClient(mt)

	comp := NewBaseComponent(client, "switch", 0)

	type SwitchConfig struct {
		Name         string `json:"name"`
		InitialState string `json:"initial_state"`
	}

	config := SwitchConfig{
		Name:         "New Name",
		InitialState: "on",
	}

	err := comp.SetConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestBaseComponent_GetStatus(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"id":0,"output":true,"apower":123.4}`),
	}
	client := newTestClient(mt)

	comp := NewBaseComponent(client, "switch", 0)
	result, err := comp.GetStatus(context.Background())

	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	var status map[string]any
	if err := json.Unmarshal(result, &status); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if status["output"] != true {
		t.Errorf("status output = %v, want true", status["output"])
	}
}

func TestBaseComponent_CapitalizedType(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want string
	}{
		{
			name: "switch",
			typ:  "switch",
			want: "Switch",
		},
		{
			name: "cover",
			typ:  "cover",
			want: "Cover",
		},
		{
			name: "em",
			typ:  "em",
			want: "EM",
		},
		{
			name: "em1",
			typ:  "em1",
			want: "EM1",
		},
		{
			name: "pm",
			typ:  "pm",
			want: "PM",
		},
		{
			name: "pm1",
			typ:  "pm1",
			want: "PM1",
		},
		{
			name: "kvs",
			typ:  "kvs",
			want: "KVS",
		},
		{
			name: "wifi",
			typ:  "wifi",
			want: "WiFi",
		},
		{
			name: "ble",
			typ:  "ble",
			want: "BLE",
		},
		{
			name: "mqtt",
			typ:  "mqtt",
			want: "Mqtt",
		},
		{
			name: "ui",
			typ:  "ui",
			want: "UI",
		},
		{
			name: "sys",
			typ:  "sys",
			want: "Sys",
		},
		{
			name: "ws",
			typ:  "ws",
			want: "Ws",
		},
		{
			name: "bthome",
			typ:  "bthome",
			want: "BTHome",
		},
		{
			name: "bthomedevice",
			typ:  "bthomedevice",
			want: "BTHomeDevice",
		},
		{
			name: "rgb",
			typ:  "rgb",
			want: "RGB",
		},
		{
			name: "rgbw",
			typ:  "rgbw",
			want: "RGBW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := &BaseComponent{typ: tt.typ}
			if got := comp.capitalizedType(); got != tt.want {
				t.Errorf("capitalizedType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseComponent_ErrorHandling(t *testing.T) {
	testErr := errors.New("test error")
	mt := &mockTransport{
		err: testErr,
	}
	client := newTestClient(mt)

	comp := NewBaseComponent(client, "switch", 0)

	t.Run("GetConfig error", func(t *testing.T) {
		_, err := comp.GetConfig(context.Background())
		if err == nil {
			t.Error("GetConfig() should return error")
		}
	})

	t.Run("SetConfig error", func(t *testing.T) {
		err := comp.SetConfig(context.Background(), map[string]any{"name": "test"})
		if err == nil {
			t.Error("SetConfig() should return error")
		}
	})

	t.Run("GetStatus error", func(t *testing.T) {
		_, err := comp.GetStatus(context.Background())
		if err == nil {
			t.Error("GetStatus() should return error")
		}
	})
}

func TestParseComponentKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantTyp string
		wantID  int
		wantErr bool
	}{
		{
			name:    "valid switch:0",
			key:     "switch:0",
			wantTyp: "switch",
			wantID:  0,
			wantErr: false,
		},
		{
			name:    "valid cover:1",
			key:     "cover:1",
			wantTyp: "cover",
			wantID:  1,
			wantErr: false,
		},
		{
			name:    "invalid format",
			key:     "switch",
			wantTyp: "",
			wantID:  0,
			wantErr: true,
		},
		{
			name:    "invalid ID",
			key:     "switch:abc",
			wantTyp: "",
			wantID:  0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, id, err := ParseComponentKey(tt.key)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseComponentKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if typ != tt.wantTyp {
					t.Errorf("ParseComponentKey() typ = %v, want %v", typ, tt.wantTyp)
				}
				if id != tt.wantID {
					t.Errorf("ParseComponentKey() id = %v, want %v", id, tt.wantID)
				}
			}
		})
	}
}

// Helper to create test client with mock transport
func newTestClient(mt transport.Transport) *rpc.Client {
	return rpc.NewClient(mt)
}

func TestBaseComponent_Client(t *testing.T) {
	mt := &mockTransport{}
	client := newTestClient(mt)

	comp := NewBaseComponent(client, "switch", 0)

	if comp.Client() != client {
		t.Error("Client() should return the underlying RPC client")
	}
}

func TestBaseComponent_Call(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"output":true}`),
	}
	client := newTestClient(mt)
	comp := NewBaseComponent(client, "switch", 0)

	t.Run("call with result", func(t *testing.T) {
		var result struct {
			Output bool `json:"output"`
		}
		err := comp.call(context.Background(), "GetStatus", map[string]int{"id": 0}, &result)
		if err != nil {
			t.Fatalf("call() error = %v", err)
		}
		if !result.Output {
			t.Error("call() result.Output = false, want true")
		}
	})

	t.Run("call without result", func(t *testing.T) {
		err := comp.call(context.Background(), "Toggle", map[string]int{"id": 0}, nil)
		if err != nil {
			t.Fatalf("call() error = %v", err)
		}
	})
}

type testConfig struct {
	Name  string `json:"name"`
	ID    int    `json:"id"`
	Value int    `json:"value"`
}

type testStatus struct {
	ID     int  `json:"id"`
	Output bool `json:"output"`
}

func TestUnmarshalConfig(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"id":0,"name":"Test Config","value":42}`),
	}
	client := newTestClient(mt)
	comp := NewBaseComponent(client, "switch", 0)

	config, err := UnmarshalConfig[testConfig](context.Background(), comp)
	if err != nil {
		t.Fatalf("UnmarshalConfig() error = %v", err)
	}

	if config.Name != "Test Config" {
		t.Errorf("config.Name = %v, want 'Test Config'", config.Name)
	}
	if config.Value != 42 {
		t.Errorf("config.Value = %v, want 42", config.Value)
	}
}

func TestUnmarshalConfig_Error(t *testing.T) {
	mt := &mockTransport{
		err: errors.New("test error"),
	}
	client := newTestClient(mt)
	comp := NewBaseComponent(client, "switch", 0)

	_, err := UnmarshalConfig[testConfig](context.Background(), comp)
	if err == nil {
		t.Error("UnmarshalConfig() should return error")
	}
}

func TestUnmarshalStatus(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"id":0,"output":true}`),
	}
	client := newTestClient(mt)
	comp := NewBaseComponent(client, "switch", 0)

	status, err := UnmarshalStatus[testStatus](context.Background(), comp)
	if err != nil {
		t.Fatalf("UnmarshalStatus() error = %v", err)
	}

	if !status.Output {
		t.Error("status.Output = false, want true")
	}
}

func TestUnmarshalStatus_Error(t *testing.T) {
	mt := &mockTransport{
		err: errors.New("test error"),
	}
	client := newTestClient(mt)
	comp := NewBaseComponent(client, "switch", 0)

	_, err := UnmarshalStatus[testStatus](context.Background(), comp)
	if err == nil {
		t.Error("UnmarshalStatus() should return error")
	}
}

func TestSetConfigWithID(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"restart_required":false}`),
	}
	client := newTestClient(mt)
	comp := NewBaseComponent(client, "switch", 5)

	t.Run("config with zero ID gets component ID", func(t *testing.T) {
		config := &testConfig{ID: 0, Name: "Test", Value: 42}
		err := SetConfigWithID(context.Background(), comp, config)
		if err != nil {
			t.Fatalf("SetConfigWithID() error = %v", err)
		}
		if config.ID != 5 {
			t.Errorf("config.ID = %v, want 5", config.ID)
		}
	})

	t.Run("config with non-zero ID keeps original", func(t *testing.T) {
		config := &testConfig{ID: 3, Name: "Test", Value: 42}
		err := SetConfigWithID(context.Background(), comp, config)
		if err != nil {
			t.Fatalf("SetConfigWithID() error = %v", err)
		}
		if config.ID != 3 {
			t.Errorf("config.ID = %v, want 3", config.ID)
		}
	})

	t.Run("config as map", func(t *testing.T) {
		config := map[string]any{"name": "Test"}
		err := SetConfigWithID(context.Background(), comp, config)
		if err != nil {
			t.Fatalf("SetConfigWithID() error = %v", err)
		}
	})
}

type testParams struct {
	ID     int  `json:"id"`
	Toggle bool `json:"toggle"`
}

func TestEnsureIDParam(t *testing.T) {
	mt := &mockTransport{}
	client := newTestClient(mt)
	comp := NewBaseComponent(client, "switch", 5)

	t.Run("nil params returns map with ID", func(t *testing.T) {
		result := EnsureIDParam(comp, nil)
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("EnsureIDParam() returned %T, want map[string]any", result)
		}
		if m["id"] != 5 {
			t.Errorf("result[id] = %v, want 5", m["id"])
		}
	})

	t.Run("struct with zero ID gets component ID", func(t *testing.T) {
		params := &testParams{ID: 0, Toggle: true}
		result := EnsureIDParam(comp, params)
		p, ok := result.(*testParams)
		if !ok {
			t.Fatalf("type assertion failed")
		}
		if p.ID != 5 {
			t.Errorf("params.ID = %v, want 5", p.ID)
		}
	})

	t.Run("struct with non-zero ID keeps original", func(t *testing.T) {
		params := &testParams{ID: 3, Toggle: true}
		result := EnsureIDParam(comp, params)
		p, ok := result.(*testParams)
		if !ok {
			t.Fatalf("type assertion failed")
		}
		if p.ID != 3 {
			t.Errorf("params.ID = %v, want 3", p.ID)
		}
	})

	t.Run("map params passed through", func(t *testing.T) {
		params := map[string]any{"id": 7, "toggle": true}
		result := EnsureIDParam(comp, params)
		// Check the result is a map with expected values
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("EnsureIDParam() returned %T, want map[string]any", result)
		}
		if m["id"] != 7 {
			t.Errorf("result[id] = %v, want 7", m["id"])
		}
	})
}

func TestCapitalizedType_EmptyString(t *testing.T) {
	comp := &BaseComponent{typ: ""}
	if got := comp.capitalizedType(); got != "" {
		t.Errorf("capitalizedType() = %v, want empty string", got)
	}
}

func TestCapitalizedType_BTHomeSensor(t *testing.T) {
	comp := &BaseComponent{typ: "bthomesensor"}
	if got := comp.capitalizedType(); got != "BTHomeSensor" {
		t.Errorf("capitalizedType() = %v, want BTHomeSensor", got)
	}
}
