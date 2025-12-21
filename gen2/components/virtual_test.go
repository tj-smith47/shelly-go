package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

// extractVirtualParams is a helper to extract params from the RPC request for testing
func extractVirtualParams(params json.RawMessage) map[string]any {
	var result map[string]any
	if err := json.Unmarshal(params, &result); err != nil {
		return nil
	}
	return result
}

// =====================================================
// Virtual Manager Tests
// =====================================================

func TestNewVirtual(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	virtual := NewVirtual(client)

	if virtual == nil {
		t.Fatal("expected non-nil Virtual")
	}
	if virtual.Client() != client {
		t.Error("client mismatch")
	}
	if virtual.Type() != "virtual" {
		t.Errorf("expected type 'virtual', got %s", virtual.Type())
	}
}

func TestVirtual_Add(t *testing.T) {
	result := `{"id": 200}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Virtual.Add" {
				t.Errorf("method = %q, want %q", method, "Virtual.Add")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if paramsMap["type"] != "boolean" {
				t.Errorf("expected type 'boolean', got %v", paramsMap["type"])
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	virtual := NewVirtual(client)

	res, err := virtual.Add(context.Background(), "boolean", nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ID != 200 {
		t.Errorf("expected ID 200, got %d", res.ID)
	}
}

func TestVirtual_Add_WithConfig(t *testing.T) {
	result := `{"id": 201}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Virtual.Add" {
				t.Errorf("method = %q, want %q", method, "Virtual.Add")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if paramsMap["type"] != "number" {
				t.Errorf("expected type 'number', got %v", paramsMap["type"])
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 201 {
				t.Errorf("expected id 201, got %v", paramsMap["id"])
			}
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatal("expected config to be map[string]any")
			}
			if config["name"] != "Temperature" {
				t.Errorf("expected name 'Temperature', got %v", config["name"])
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	virtual := NewVirtual(client)

	res, err := virtual.Add(context.Background(), "number", map[string]any{
		"name": "Temperature",
		"min":  15.0,
		"max":  30.0,
	}, 201)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ID != 201 {
		t.Errorf("expected ID 201, got %d", res.ID)
	}
}

func TestVirtual_Add_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	virtual := NewVirtual(client)

	testComponentError(t, "Add", func() error {
		_, err := virtual.Add(context.Background(), "boolean", nil, 0)
		return err
	})
}

func TestVirtual_Add_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	virtual := NewVirtual(client)

	testComponentInvalidJSON(t, "Add", func() error {
		_, err := virtual.Add(context.Background(), "boolean", nil, 0)
		return err
	})
}

func TestVirtual_Delete(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Virtual.Delete" {
				t.Errorf("method = %q, want %q", method, "Virtual.Delete")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if paramsMap["key"] != "boolean:200" {
				t.Errorf("expected key 'boolean:200', got %v", paramsMap["key"])
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	virtual := NewVirtual(client)

	err := virtual.Delete(context.Background(), "boolean:200")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtual_Delete_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	virtual := NewVirtual(client)

	testComponentError(t, "Delete", func() error {
		return virtual.Delete(context.Background(), "boolean:200")
	})
}

// =====================================================
// VirtualBoolean Tests
// =====================================================

func TestNewVirtualBoolean(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	vBool := NewVirtualBoolean(client, 200)

	if vBool == nil {
		t.Fatal("expected non-nil VirtualBoolean")
	}
	if vBool.Client() != client {
		t.Error("client mismatch")
	}
	if vBool.ID() != 200 {
		t.Errorf("expected ID 200, got %d", vBool.ID())
	}
	if vBool.Type() != "boolean" {
		t.Errorf("expected type 'boolean', got %s", vBool.Type())
	}
	if vBool.Key() != "boolean:200" {
		t.Errorf("expected key 'boolean:200', got %s", vBool.Key())
	}
}

func TestVirtualBoolean_GetConfig(t *testing.T) {
	result := `{
		"id": 200,
		"name": "Test Boolean",
		"default_value": false,
		"persisted": true
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Boolean.GetConfig" {
				t.Errorf("method = %q, want %q", method, "Boolean.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vBool := NewVirtualBoolean(client, 200)

	config, err := vBool.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ID != 200 {
		t.Errorf("expected ID 200, got %d", config.ID)
	}
	if config.Name == nil || *config.Name != "Test Boolean" {
		t.Errorf("expected Name 'Test Boolean', got %v", config.Name)
	}
	if config.DefaultValue == nil || *config.DefaultValue != false {
		t.Error("expected DefaultValue to be false")
	}
	if config.Persisted == nil || *config.Persisted != true {
		t.Error("expected Persisted to be true")
	}
}

func TestVirtualBoolean_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vBool := NewVirtualBoolean(client, 200)

	testComponentError(t, "GetConfig", func() error {
		_, err := vBool.GetConfig(context.Background())
		return err
	})
}

func TestVirtualBoolean_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vBool := NewVirtualBoolean(client, 200)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := vBool.GetConfig(context.Background())
		return err
	})
}

func TestVirtualBoolean_SetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Boolean.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Boolean.SetConfig")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 200 {
				t.Errorf("expected id 200, got %v", paramsMap["id"])
			}
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatal("expected config to be map[string]any")
			}
			if config["name"] != "My Boolean" {
				t.Errorf("expected name 'My Boolean', got %v", config["name"])
			}
			if config["default_value"] != true {
				t.Errorf("expected default_value true, got %v", config["default_value"])
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	vBool := NewVirtualBoolean(client, 200)

	name := "My Boolean"
	defaultVal := true
	persisted := true
	err := vBool.SetConfig(context.Background(), &VirtualBooleanConfig{
		Name:         &name,
		DefaultValue: &defaultVal,
		Persisted:    &persisted,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualBoolean_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vBool := NewVirtualBoolean(client, 200)

	testComponentError(t, "SetConfig", func() error {
		return vBool.SetConfig(context.Background(), &VirtualBooleanConfig{})
	})
}

func TestVirtualBoolean_GetStatus(t *testing.T) {
	result := `{
		"id": 200,
		"value": true
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Boolean.GetStatus" {
				t.Errorf("method = %q, want %q", method, "Boolean.GetStatus")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vBool := NewVirtualBoolean(client, 200)

	status, err := vBool.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.ID != 200 {
		t.Errorf("expected ID 200, got %d", status.ID)
	}
	if status.Value == nil || *status.Value != true {
		t.Error("expected Value to be true")
	}
}

func TestVirtualBoolean_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vBool := NewVirtualBoolean(client, 200)

	testComponentError(t, "GetStatus", func() error {
		_, err := vBool.GetStatus(context.Background())
		return err
	})
}

func TestVirtualBoolean_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vBool := NewVirtualBoolean(client, 200)

	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := vBool.GetStatus(context.Background())
		return err
	})
}

func TestVirtualBoolean_Set(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Boolean.Set" {
				t.Errorf("method = %q, want %q", method, "Boolean.Set")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 200 {
				t.Errorf("expected id 200, got %v", paramsMap["id"])
			}
			if paramsMap["value"] != true {
				t.Errorf("expected value true, got %v", paramsMap["value"])
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	vBool := NewVirtualBoolean(client, 200)

	err := vBool.Set(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualBoolean_Set_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vBool := NewVirtualBoolean(client, 200)

	testComponentError(t, "Set", func() error {
		return vBool.Set(context.Background(), true)
	})
}

func TestVirtualBoolean_Toggle(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Boolean.Toggle" {
				t.Errorf("method = %q, want %q", method, "Boolean.Toggle")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 200 {
				t.Errorf("expected id 200, got %v", paramsMap["id"])
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	vBool := NewVirtualBoolean(client, 200)

	err := vBool.Toggle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualBoolean_Toggle_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vBool := NewVirtualBoolean(client, 200)

	testComponentError(t, "Toggle", func() error {
		return vBool.Toggle(context.Background())
	})
}

// =====================================================
// VirtualNumber Tests
// =====================================================

func TestNewVirtualNumber(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	vNum := NewVirtualNumber(client, 201)

	if vNum == nil {
		t.Fatal("expected non-nil VirtualNumber")
	}
	if vNum.Client() != client {
		t.Error("client mismatch")
	}
	if vNum.ID() != 201 {
		t.Errorf("expected ID 201, got %d", vNum.ID())
	}
	if vNum.Type() != "number" {
		t.Errorf("expected type 'number', got %s", vNum.Type())
	}
	if vNum.Key() != "number:201" {
		t.Errorf("expected key 'number:201', got %s", vNum.Key())
	}
}

func TestVirtualNumber_GetConfig(t *testing.T) {
	result := `{
		"id": 201,
		"name": "Temperature Setpoint",
		"min": 15.0,
		"max": 30.0,
		"step": 0.5,
		"default_value": 21.0,
		"persisted": true,
		"unit": "°C"
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Number.GetConfig" {
				t.Errorf("method = %q, want %q", method, "Number.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vNum := NewVirtualNumber(client, 201)

	config, err := vNum.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ID != 201 {
		t.Errorf("expected ID 201, got %d", config.ID)
	}
	if config.Name == nil || *config.Name != "Temperature Setpoint" {
		t.Errorf("expected Name 'Temperature Setpoint', got %v", config.Name)
	}
	if config.Min == nil || *config.Min != 15.0 {
		t.Errorf("expected Min 15.0, got %v", config.Min)
	}
	if config.Max == nil || *config.Max != 30.0 {
		t.Errorf("expected Max 30.0, got %v", config.Max)
	}
	if config.Step == nil || *config.Step != 0.5 {
		t.Errorf("expected Step 0.5, got %v", config.Step)
	}
	if config.Unit == nil || *config.Unit != "°C" {
		t.Errorf("expected Unit '°C', got %v", config.Unit)
	}
}

func TestVirtualNumber_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vNum := NewVirtualNumber(client, 201)

	testComponentError(t, "GetConfig", func() error {
		_, err := vNum.GetConfig(context.Background())
		return err
	})
}

func TestVirtualNumber_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vNum := NewVirtualNumber(client, 201)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := vNum.GetConfig(context.Background())
		return err
	})
}

func TestVirtualNumber_SetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Number.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Number.SetConfig")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 201 {
				t.Errorf("expected id 201, got %v", paramsMap["id"])
			}
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatal("expected config to be map[string]any")
			}
			if config["name"] != "My Number" {
				t.Errorf("expected name 'My Number', got %v", config["name"])
			}
			if config["min"] != 0.0 {
				t.Errorf("expected min 0.0, got %v", config["min"])
			}
			if config["max"] != 100.0 {
				t.Errorf("expected max 100.0, got %v", config["max"])
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	vNum := NewVirtualNumber(client, 201)

	name := "My Number"
	min := 0.0
	max := 100.0
	step := 1.0
	unit := "W"
	err := vNum.SetConfig(context.Background(), &VirtualNumberConfig{
		Name: &name,
		Min:  &min,
		Max:  &max,
		Step: &step,
		Unit: &unit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualNumber_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vNum := NewVirtualNumber(client, 201)

	testComponentError(t, "SetConfig", func() error {
		return vNum.SetConfig(context.Background(), &VirtualNumberConfig{})
	})
}

func TestVirtualNumber_GetStatus(t *testing.T) {
	result := `{
		"id": 201,
		"value": 22.5
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Number.GetStatus" {
				t.Errorf("method = %q, want %q", method, "Number.GetStatus")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vNum := NewVirtualNumber(client, 201)

	status, err := vNum.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.ID != 201 {
		t.Errorf("expected ID 201, got %d", status.ID)
	}
	if status.Value == nil || *status.Value != 22.5 {
		t.Errorf("expected Value 22.5, got %v", status.Value)
	}
}

func TestVirtualNumber_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vNum := NewVirtualNumber(client, 201)

	testComponentError(t, "GetStatus", func() error {
		_, err := vNum.GetStatus(context.Background())
		return err
	})
}

func TestVirtualNumber_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vNum := NewVirtualNumber(client, 201)

	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := vNum.GetStatus(context.Background())
		return err
	})
}

func TestVirtualNumber_Set(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Number.Set" {
				t.Errorf("method = %q, want %q", method, "Number.Set")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 201 {
				t.Errorf("expected id 201, got %v", paramsMap["id"])
			}
			if paramsMap["value"] != 25.5 {
				t.Errorf("expected value 25.5, got %v", paramsMap["value"])
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	vNum := NewVirtualNumber(client, 201)

	err := vNum.Set(context.Background(), 25.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualNumber_Set_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vNum := NewVirtualNumber(client, 201)

	testComponentError(t, "Set", func() error {
		return vNum.Set(context.Background(), 25.5)
	})
}

// =====================================================
// VirtualText Tests
// =====================================================

func TestNewVirtualText(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	vText := NewVirtualText(client, 202)

	if vText == nil {
		t.Fatal("expected non-nil VirtualText")
	}
	if vText.Client() != client {
		t.Error("client mismatch")
	}
	if vText.ID() != 202 {
		t.Errorf("expected ID 202, got %d", vText.ID())
	}
	if vText.Type() != "text" {
		t.Errorf("expected type 'text', got %s", vText.Type())
	}
	if vText.Key() != "text:202" {
		t.Errorf("expected key 'text:202', got %s", vText.Key())
	}
}

func TestVirtualText_GetConfig(t *testing.T) {
	result := `{
		"id": 202,
		"name": "Message",
		"max_len": 100,
		"default_value": "Hello",
		"persisted": true
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Text.GetConfig" {
				t.Errorf("method = %q, want %q", method, "Text.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vText := NewVirtualText(client, 202)

	config, err := vText.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ID != 202 {
		t.Errorf("expected ID 202, got %d", config.ID)
	}
	if config.Name == nil || *config.Name != "Message" {
		t.Errorf("expected Name 'Message', got %v", config.Name)
	}
	if config.MaxLen == nil || *config.MaxLen != 100 {
		t.Errorf("expected MaxLen 100, got %v", config.MaxLen)
	}
	if config.DefaultValue == nil || *config.DefaultValue != "Hello" {
		t.Errorf("expected DefaultValue 'Hello', got %v", config.DefaultValue)
	}
}

func TestVirtualText_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vText := NewVirtualText(client, 202)

	testComponentError(t, "GetConfig", func() error {
		_, err := vText.GetConfig(context.Background())
		return err
	})
}

func TestVirtualText_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vText := NewVirtualText(client, 202)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := vText.GetConfig(context.Background())
		return err
	})
}

func TestVirtualText_SetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Text.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Text.SetConfig")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 202 {
				t.Errorf("expected id 202, got %v", paramsMap["id"])
			}
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatal("expected config to be map[string]any")
			}
			if config["name"] != "My Text" {
				t.Errorf("expected name 'My Text', got %v", config["name"])
			}
			// max_len is an int, but JSON unmarshal gives float64
			if maxLen, ok := config["max_len"].(float64); !ok || int(maxLen) != 50 {
				t.Errorf("expected max_len 50, got %v", config["max_len"])
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	vText := NewVirtualText(client, 202)

	name := "My Text"
	maxLen := 50
	defaultVal := "Default"
	err := vText.SetConfig(context.Background(), &VirtualTextConfig{
		Name:         &name,
		MaxLen:       &maxLen,
		DefaultValue: &defaultVal,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualText_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vText := NewVirtualText(client, 202)

	testComponentError(t, "SetConfig", func() error {
		return vText.SetConfig(context.Background(), &VirtualTextConfig{})
	})
}

func TestVirtualText_GetStatus(t *testing.T) {
	result := `{
		"id": 202,
		"value": "Hello, World!"
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Text.GetStatus" {
				t.Errorf("method = %q, want %q", method, "Text.GetStatus")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vText := NewVirtualText(client, 202)

	status, err := vText.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.ID != 202 {
		t.Errorf("expected ID 202, got %d", status.ID)
	}
	if status.Value == nil || *status.Value != "Hello, World!" {
		t.Errorf("expected Value 'Hello, World!', got %v", status.Value)
	}
}

func TestVirtualText_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vText := NewVirtualText(client, 202)

	testComponentError(t, "GetStatus", func() error {
		_, err := vText.GetStatus(context.Background())
		return err
	})
}

func TestVirtualText_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vText := NewVirtualText(client, 202)

	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := vText.GetStatus(context.Background())
		return err
	})
}

func TestVirtualText_Set(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Text.Set" {
				t.Errorf("method = %q, want %q", method, "Text.Set")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 202 {
				t.Errorf("expected id 202, got %v", paramsMap["id"])
			}
			if paramsMap["value"] != "New Value" {
				t.Errorf("expected value 'New Value', got %v", paramsMap["value"])
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	vText := NewVirtualText(client, 202)

	err := vText.Set(context.Background(), "New Value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualText_Set_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vText := NewVirtualText(client, 202)

	testComponentError(t, "Set", func() error {
		return vText.Set(context.Background(), "test")
	})
}

// =====================================================
// VirtualEnum Tests
// =====================================================

func TestNewVirtualEnum(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	vEnum := NewVirtualEnum(client, 203)

	if vEnum == nil {
		t.Fatal("expected non-nil VirtualEnum")
	}
	if vEnum.Client() != client {
		t.Error("client mismatch")
	}
	if vEnum.ID() != 203 {
		t.Errorf("expected ID 203, got %d", vEnum.ID())
	}
	if vEnum.Type() != "enum" {
		t.Errorf("expected type 'enum', got %s", vEnum.Type())
	}
	if vEnum.Key() != "enum:203" {
		t.Errorf("expected key 'enum:203', got %s", vEnum.Key())
	}
}

func TestVirtualEnum_GetConfig(t *testing.T) {
	result := `{
		"id": 203,
		"name": "Mode",
		"options": ["off", "heat", "cool", "auto"],
		"default_value": "off",
		"persisted": true
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Enum.GetConfig" {
				t.Errorf("method = %q, want %q", method, "Enum.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vEnum := NewVirtualEnum(client, 203)

	config, err := vEnum.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ID != 203 {
		t.Errorf("expected ID 203, got %d", config.ID)
	}
	if config.Name == nil || *config.Name != "Mode" {
		t.Errorf("expected Name 'Mode', got %v", config.Name)
	}
	if len(config.Options) != 4 {
		t.Errorf("expected 4 options, got %d", len(config.Options))
	}
	if config.Options[0] != "off" {
		t.Errorf("expected first option 'off', got %s", config.Options[0])
	}
	if config.DefaultValue == nil || *config.DefaultValue != "off" {
		t.Errorf("expected DefaultValue 'off', got %v", config.DefaultValue)
	}
}

func TestVirtualEnum_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vEnum := NewVirtualEnum(client, 203)

	testComponentError(t, "GetConfig", func() error {
		_, err := vEnum.GetConfig(context.Background())
		return err
	})
}

func TestVirtualEnum_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vEnum := NewVirtualEnum(client, 203)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := vEnum.GetConfig(context.Background())
		return err
	})
}

func TestVirtualEnum_SetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Enum.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Enum.SetConfig")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 203 {
				t.Errorf("expected id 203, got %v", paramsMap["id"])
			}
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatal("expected config to be map[string]any")
			}
			if config["name"] != "My Enum" {
				t.Errorf("expected name 'My Enum', got %v", config["name"])
			}
			options, ok := config["options"].([]any)
			if !ok {
				t.Fatal("expected options to be []any")
			}
			if len(options) != 3 {
				t.Errorf("expected 3 options, got %d", len(options))
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	vEnum := NewVirtualEnum(client, 203)

	name := "My Enum"
	defaultVal := "option1"
	err := vEnum.SetConfig(context.Background(), &VirtualEnumConfig{
		Name:         &name,
		Options:      []string{"option1", "option2", "option3"},
		DefaultValue: &defaultVal,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualEnum_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vEnum := NewVirtualEnum(client, 203)

	testComponentError(t, "SetConfig", func() error {
		return vEnum.SetConfig(context.Background(), &VirtualEnumConfig{})
	})
}

func TestVirtualEnum_GetStatus(t *testing.T) {
	result := `{
		"id": 203,
		"value": "heat"
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Enum.GetStatus" {
				t.Errorf("method = %q, want %q", method, "Enum.GetStatus")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vEnum := NewVirtualEnum(client, 203)

	status, err := vEnum.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.ID != 203 {
		t.Errorf("expected ID 203, got %d", status.ID)
	}
	if status.Value == nil || *status.Value != "heat" {
		t.Errorf("expected Value 'heat', got %v", status.Value)
	}
}

func TestVirtualEnum_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vEnum := NewVirtualEnum(client, 203)

	testComponentError(t, "GetStatus", func() error {
		_, err := vEnum.GetStatus(context.Background())
		return err
	})
}

func TestVirtualEnum_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vEnum := NewVirtualEnum(client, 203)

	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := vEnum.GetStatus(context.Background())
		return err
	})
}

func TestVirtualEnum_Set(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Enum.Set" {
				t.Errorf("method = %q, want %q", method, "Enum.Set")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 203 {
				t.Errorf("expected id 203, got %v", paramsMap["id"])
			}
			if paramsMap["value"] != "cool" {
				t.Errorf("expected value 'cool', got %v", paramsMap["value"])
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	vEnum := NewVirtualEnum(client, 203)

	err := vEnum.Set(context.Background(), "cool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualEnum_Set_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vEnum := NewVirtualEnum(client, 203)

	testComponentError(t, "Set", func() error {
		return vEnum.Set(context.Background(), "heat")
	})
}

// =====================================================
// VirtualButton Tests
// =====================================================

func TestNewVirtualButton(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	vButton := NewVirtualButton(client, 204)

	if vButton == nil {
		t.Fatal("expected non-nil VirtualButton")
	}
	if vButton.Client() != client {
		t.Error("client mismatch")
	}
	if vButton.ID() != 204 {
		t.Errorf("expected ID 204, got %d", vButton.ID())
	}
	if vButton.Type() != "button" {
		t.Errorf("expected type 'button', got %s", vButton.Type())
	}
	if vButton.Key() != "button:204" {
		t.Errorf("expected key 'button:204', got %s", vButton.Key())
	}
}

func TestVirtualButton_GetConfig(t *testing.T) {
	result := `{
		"id": 204,
		"name": "Reset Button"
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Button.GetConfig" {
				t.Errorf("method = %q, want %q", method, "Button.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vButton := NewVirtualButton(client, 204)

	config, err := vButton.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ID != 204 {
		t.Errorf("expected ID 204, got %d", config.ID)
	}
	if config.Name == nil || *config.Name != "Reset Button" {
		t.Errorf("expected Name 'Reset Button', got %v", config.Name)
	}
}

func TestVirtualButton_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vButton := NewVirtualButton(client, 204)

	testComponentError(t, "GetConfig", func() error {
		_, err := vButton.GetConfig(context.Background())
		return err
	})
}

func TestVirtualButton_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vButton := NewVirtualButton(client, 204)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := vButton.GetConfig(context.Background())
		return err
	})
}

func TestVirtualButton_SetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Button.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Button.SetConfig")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 204 {
				t.Errorf("expected id 204, got %v", paramsMap["id"])
			}
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatal("expected config to be map[string]any")
			}
			if config["name"] != "My Button" {
				t.Errorf("expected name 'My Button', got %v", config["name"])
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	vButton := NewVirtualButton(client, 204)

	name := "My Button"
	err := vButton.SetConfig(context.Background(), &VirtualButtonConfig{
		Name: &name,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualButton_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vButton := NewVirtualButton(client, 204)

	testComponentError(t, "SetConfig", func() error {
		return vButton.SetConfig(context.Background(), &VirtualButtonConfig{})
	})
}

func TestVirtualButton_GetStatus(t *testing.T) {
	result := `{
		"id": 204,
		"last_pressed": 1704067200
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Button.GetStatus" {
				t.Errorf("method = %q, want %q", method, "Button.GetStatus")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vButton := NewVirtualButton(client, 204)

	status, err := vButton.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.ID != 204 {
		t.Errorf("expected ID 204, got %d", status.ID)
	}
	if status.LastPressed == nil || *status.LastPressed != 1704067200 {
		t.Errorf("expected LastPressed 1704067200, got %v", status.LastPressed)
	}
}

func TestVirtualButton_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vButton := NewVirtualButton(client, 204)

	testComponentError(t, "GetStatus", func() error {
		_, err := vButton.GetStatus(context.Background())
		return err
	})
}

func TestVirtualButton_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vButton := NewVirtualButton(client, 204)

	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := vButton.GetStatus(context.Background())
		return err
	})
}

func TestVirtualButton_Trigger(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Button.Trigger" {
				t.Errorf("method = %q, want %q", method, "Button.Trigger")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 204 {
				t.Errorf("expected id 204, got %v", paramsMap["id"])
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	vButton := NewVirtualButton(client, 204)

	err := vButton.Trigger(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualButton_Trigger_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vButton := NewVirtualButton(client, 204)

	testComponentError(t, "Trigger", func() error {
		return vButton.Trigger(context.Background())
	})
}

// =====================================================
// VirtualGroup Tests
// =====================================================

func TestNewVirtualGroup(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	vGroup := NewVirtualGroup(client, 205)

	if vGroup == nil {
		t.Fatal("expected non-nil VirtualGroup")
	}
	if vGroup.Client() != client {
		t.Error("client mismatch")
	}
	if vGroup.ID() != 205 {
		t.Errorf("expected ID 205, got %d", vGroup.ID())
	}
	if vGroup.Type() != "group" {
		t.Errorf("expected type 'group', got %s", vGroup.Type())
	}
	if vGroup.Key() != "group:205" {
		t.Errorf("expected key 'group:205', got %s", vGroup.Key())
	}
}

func TestVirtualGroup_GetConfig(t *testing.T) {
	result := `{
		"id": 205,
		"name": "Living Room",
		"members": ["boolean:200", "number:201", "text:202"]
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Group.GetConfig" {
				t.Errorf("method = %q, want %q", method, "Group.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vGroup := NewVirtualGroup(client, 205)

	config, err := vGroup.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ID != 205 {
		t.Errorf("expected ID 205, got %d", config.ID)
	}
	if config.Name == nil || *config.Name != "Living Room" {
		t.Errorf("expected Name 'Living Room', got %v", config.Name)
	}
	if len(config.Members) != 3 {
		t.Errorf("expected 3 members, got %d", len(config.Members))
	}
	if config.Members[0] != "boolean:200" {
		t.Errorf("expected first member 'boolean:200', got %s", config.Members[0])
	}
}

func TestVirtualGroup_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vGroup := NewVirtualGroup(client, 205)

	testComponentError(t, "GetConfig", func() error {
		_, err := vGroup.GetConfig(context.Background())
		return err
	})
}

func TestVirtualGroup_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vGroup := NewVirtualGroup(client, 205)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := vGroup.GetConfig(context.Background())
		return err
	})
}

func TestVirtualGroup_SetConfig(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Group.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Group.SetConfig")
			}
			paramsMap := extractVirtualParams(req.GetParams())
			if paramsMap == nil {
				t.Fatal("expected params to be extractable")
			}
			if id, ok := paramsMap["id"].(float64); !ok || int(id) != 205 {
				t.Errorf("expected id 205, got %v", paramsMap["id"])
			}
			config, ok := paramsMap["config"].(map[string]any)
			if !ok {
				t.Fatal("expected config to be map[string]any")
			}
			if config["name"] != "My Group" {
				t.Errorf("expected name 'My Group', got %v", config["name"])
			}
			members, ok := config["members"].([]any)
			if !ok {
				t.Fatal("expected members to be []any")
			}
			if len(members) != 2 {
				t.Errorf("expected 2 members, got %d", len(members))
			}
			return jsonrpcResponse(`{"restart_required": false}`)
		},
	}
	client := rpc.NewClient(tr)
	vGroup := NewVirtualGroup(client, 205)

	name := "My Group"
	err := vGroup.SetConfig(context.Background(), &VirtualGroupConfig{
		Name:    &name,
		Members: []string{"boolean:200", "number:201"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVirtualGroup_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vGroup := NewVirtualGroup(client, 205)

	testComponentError(t, "SetConfig", func() error {
		return vGroup.SetConfig(context.Background(), &VirtualGroupConfig{})
	})
}

func TestVirtualGroup_GetStatus(t *testing.T) {
	result := `{
		"id": 205
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Group.GetStatus" {
				t.Errorf("method = %q, want %q", method, "Group.GetStatus")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	vGroup := NewVirtualGroup(client, 205)

	status, err := vGroup.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.ID != 205 {
		t.Errorf("expected ID 205, got %d", status.ID)
	}
}

func TestVirtualGroup_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device error")))
	vGroup := NewVirtualGroup(client, 205)

	testComponentError(t, "GetStatus", func() error {
		_, err := vGroup.GetStatus(context.Background())
		return err
	})
}

func TestVirtualGroup_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	vGroup := NewVirtualGroup(client, 205)

	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := vGroup.GetStatus(context.Background())
		return err
	})
}
