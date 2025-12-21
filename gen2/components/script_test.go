package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewScript(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	script := NewScript(client)

	if script == nil {
		t.Fatal("NewScript returned nil")
	}

	if script.Type() != "script" {
		t.Errorf("Type() = %q, want %q", script.Type(), "script")
	}

	if script.Key() != "script" {
		t.Errorf("Key() = %q, want %q", script.Key(), "script")
	}

	if script.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestScript_List(t *testing.T) {
	tests := []struct {
		name      string
		result    string
		wantCount int
	}{
		{
			name: "multiple scripts",
			result: `{
				"scripts": [
					{"id": 1, "name": "Script 1", "enable": true, "running": true},
					{"id": 2, "name": "Script 2", "enable": false, "running": false}
				]
			}`,
			wantCount: 2,
		},
		{
			name:      "no scripts",
			result:    `{"scripts": []}`,
			wantCount: 0,
		},
		{
			name: "single running script",
			result: `{
				"scripts": [
					{"id": 1, "name": "Automation", "enable": true, "running": true}
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
					if method != "Script.List" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			script := NewScript(client)

			result, err := script.List(context.Background())
			if err != nil {
				t.Errorf("List() error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("List() returned nil result")
			}

			if len(result.Scripts) != tt.wantCount {
				t.Errorf("len(Scripts) = %d, want %d", len(result.Scripts), tt.wantCount)
			}
		})
	}
}

func TestScript_List_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	script := NewScript(client)
	testComponentError(t, "List", func() error {
		_, err := script.List(context.Background())
		return err
	})
}

func TestScript_List_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	script := NewScript(client)
	testComponentInvalidJSON(t, "List", func() error {
		_, err := script.List(context.Background())
		return err
	})
}

func TestScript_Create(t *testing.T) {
	tests := []struct {
		sname  *string
		name   string
		wantID int
	}{
		{
			name:   "create with name",
			sname:  ptr("My Script"),
			wantID: 1,
		},
		{
			name:   "create without name",
			sname:  nil,
			wantID: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Script.Create" {
						t.Errorf("method = %q, want %q", method, "Script.Create")
					}
					return jsonrpcResponse(`{"id": ` + string(rune('0'+tt.wantID)) + `}`)
				},
			}
			client := rpc.NewClient(tr)
			script := NewScript(client)

			result, err := script.Create(context.Background(), tt.sname)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if result.ID != tt.wantID {
				t.Errorf("result.ID = %d, want %d", result.ID, tt.wantID)
			}
		})
	}
}

func TestScript_Create_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	script := NewScript(client)
	testComponentError(t, "Create", func() error {
		_, err := script.Create(context.Background(), ptr("Test"))
		return err
	})
}

func TestScript_GetConfig(t *testing.T) {
	tests := []struct {
		wantName   *string
		wantEnable *bool
		name       string
		result     string
		id         int
	}{
		{
			name:       "enabled script",
			id:         1,
			result:     `{"id": 1, "name": "My Script", "enable": true}`,
			wantName:   ptr("My Script"),
			wantEnable: ptr(true),
		},
		{
			name:       "disabled script",
			id:         2,
			result:     `{"id": 2, "name": "Disabled Script", "enable": false}`,
			wantName:   ptr("Disabled Script"),
			wantEnable: ptr(false),
		},
		{
			name:   "minimal config",
			id:     3,
			result: `{"id": 3}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Script.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			script := NewScript(client)

			config, err := script.GetConfig(context.Background(), tt.id)
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if config == nil {
				t.Fatal("GetConfig() returned nil config")
			}

			if config.ID != tt.id {
				t.Errorf("config.ID = %d, want %d", config.ID, tt.id)
			}

			if tt.wantName != nil {
				if config.Name == nil || *config.Name != *tt.wantName {
					t.Errorf("config.Name = %v, want %v", config.Name, *tt.wantName)
				}
			}

			if tt.wantEnable != nil {
				if config.Enable == nil || *config.Enable != *tt.wantEnable {
					t.Errorf("config.Enable = %v, want %v", config.Enable, *tt.wantEnable)
				}
			}
		})
	}
}

func TestScript_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	script := NewScript(client)
	testComponentError(t, "GetConfig", func() error {
		_, err := script.GetConfig(context.Background(), 1)
		return err
	})
}

func TestScript_SetConfig(t *testing.T) {
	tests := []struct {
		config *ScriptConfig
		name   string
		id     int
	}{
		{
			name: "set name",
			id:   1,
			config: &ScriptConfig{
				Name: ptr("New Name"),
			},
		},
		{
			name: "enable script",
			id:   2,
			config: &ScriptConfig{
				Enable: ptr(true),
			},
		},
		{
			name: "disable script",
			id:   3,
			config: &ScriptConfig{
				Enable: ptr(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Script.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Script.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			script := NewScript(client)

			err := script.SetConfig(context.Background(), tt.id, tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestScript_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	script := NewScript(client)
	testComponentError(t, "SetConfig", func() error {
		return script.SetConfig(context.Background(), 1, &ScriptConfig{})
	})
}

func TestScript_GetStatus(t *testing.T) {
	tests := []struct {
		wantMemUsage *int
		name         string
		result       string
		id           int
		wantRunning  bool
	}{
		{
			name:         "running script",
			id:           1,
			result:       `{"id": 1, "running": true, "mem_usage": 1024, "mem_peak": 2048, "mem_free": 30720}`,
			wantRunning:  true,
			wantMemUsage: ptr(1024),
		},
		{
			name:        "stopped script",
			id:          2,
			result:      `{"id": 2, "running": false}`,
			wantRunning: false,
		},
		{
			name:        "script with errors",
			id:          3,
			result:      `{"id": 3, "running": false, "errors": ["SyntaxError: Unexpected token"]}`,
			wantRunning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Script.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			script := NewScript(client)

			status, err := script.GetStatus(context.Background(), tt.id)
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if status.ID != tt.id {
				t.Errorf("status.ID = %d, want %d", status.ID, tt.id)
			}

			if status.Running != tt.wantRunning {
				t.Errorf("status.Running = %v, want %v", status.Running, tt.wantRunning)
			}

			if tt.wantMemUsage != nil {
				if status.MemUsage == nil || *status.MemUsage != *tt.wantMemUsage {
					t.Errorf("status.MemUsage = %v, want %v", status.MemUsage, *tt.wantMemUsage)
				}
			}
		})
	}
}

func TestScript_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	script := NewScript(client)
	testComponentError(t, "GetStatus", func() error {
		_, err := script.GetStatus(context.Background(), 1)
		return err
	})
}

func TestScript_GetCode(t *testing.T) {
	tests := []struct {
		name     string
		result   string
		wantCode string
		id       int
	}{
		{
			name:     "simple script",
			id:       1,
			result:   `{"data": "print('Hello, World!');"}`,
			wantCode: "print('Hello, World!');",
		},
		{
			name:     "multiline script",
			id:       2,
			result:   `{"data": "let x = 1;\nlet y = 2;\nprint(x + y);"}`,
			wantCode: "let x = 1;\nlet y = 2;\nprint(x + y);",
		},
		{
			name:     "empty script",
			id:       3,
			result:   `{"data": ""}`,
			wantCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Script.GetCode" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			script := NewScript(client)

			result, err := script.GetCode(context.Background(), tt.id)
			if err != nil {
				t.Errorf("GetCode() error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("GetCode() returned nil result")
			}

			if result.Data != tt.wantCode {
				t.Errorf("result.Data = %q, want %q", result.Data, tt.wantCode)
			}
		})
	}
}

func TestScript_GetCode_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	script := NewScript(client)
	testComponentError(t, "GetCode", func() error {
		_, err := script.GetCode(context.Background(), 1)
		return err
	})
}

func TestScript_PutCode(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		id     int
		append bool
	}{
		{
			name:   "replace code",
			id:     1,
			code:   "print('Hello');",
			append: false,
		},
		{
			name:   "append code",
			id:     2,
			code:   "\nprint('Appended');",
			append: true,
		},
		{
			name:   "empty code",
			id:     3,
			code:   "",
			append: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Script.PutCode" {
						t.Errorf("method = %q, want %q", method, "Script.PutCode")
					}
					return jsonrpcResponse(`{"len": 100}`)
				},
			}
			client := rpc.NewClient(tr)
			script := NewScript(client)

			err := script.PutCode(context.Background(), tt.id, tt.code, tt.append)
			if err != nil {
				t.Fatalf("PutCode() error = %v", err)
			}
		})
	}
}

func TestScript_PutCode_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	script := NewScript(client)
	testComponentError(t, "PutCode", func() error {
		return script.PutCode(context.Background(), 1, "code", false)
	})
}

func TestScript_Start(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Script.Start" {
				t.Errorf("method = %q, want %q", method, "Script.Start")
			}
			return jsonrpcResponse(`{"was_running": false}`)
		},
	}
	client := rpc.NewClient(tr)
	script := NewScript(client)

	err := script.Start(context.Background(), 1)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

func TestScript_Start_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	script := NewScript(client)
	testComponentError(t, "Start", func() error {
		return script.Start(context.Background(), 1)
	})
}

func TestScript_Stop(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Script.Stop" {
				t.Errorf("method = %q, want %q", method, "Script.Stop")
			}
			return jsonrpcResponse(`{"was_running": true}`)
		},
	}
	client := rpc.NewClient(tr)
	script := NewScript(client)

	err := script.Stop(context.Background(), 1)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestScript_Stop_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	script := NewScript(client)
	testComponentError(t, "Stop", func() error {
		return script.Stop(context.Background(), 1)
	})
}

func TestScript_Delete(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Script.Delete" {
				t.Errorf("method = %q, want %q", method, "Script.Delete")
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	script := NewScript(client)

	err := script.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestScript_Delete_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	script := NewScript(client)
	testComponentError(t, "Delete", func() error {
		return script.Delete(context.Background(), 1)
	})
}

func TestScript_Eval(t *testing.T) {
	tests := []struct {
		wantResult any
		name       string
		code       string
		result     string
		id         int
	}{
		{
			name:       "numeric result",
			id:         1,
			code:       "1 + 2",
			result:     `{"result": 3}`,
			wantResult: float64(3),
		},
		{
			name:       "string result",
			id:         1,
			code:       "'hello'",
			result:     `{"result": "hello"}`,
			wantResult: "hello",
		},
		{
			name:       "boolean result",
			id:         1,
			code:       "true",
			result:     `{"result": true}`,
			wantResult: true,
		},
		{
			name:       "null result",
			id:         1,
			code:       "null",
			result:     `{"result": null}`,
			wantResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Script.Eval" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			script := NewScript(client)

			result, err := script.Eval(context.Background(), tt.id, tt.code)
			if err != nil {
				t.Errorf("Eval() error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("Eval() returned nil result")
			}

			if result.Result != tt.wantResult {
				t.Errorf("result.Result = %v (%T), want %v (%T)", result.Result, result.Result, tt.wantResult, tt.wantResult)
			}
		})
	}
}

func TestScript_Eval_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	script := NewScript(client)
	testComponentError(t, "Eval", func() error {
		_, err := script.Eval(context.Background(), 1, "1+1")
		return err
	})
}

func TestScriptConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config ScriptConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: ScriptConfig{
				ID:     1,
				Name:   ptr("Test Script"),
				Enable: ptr(true),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["id"].(float64) != 1 {
					t.Errorf("id = %v, want 1", data["id"])
				}
				if data["name"].(string) != "Test Script" {
					t.Errorf("name = %v, want Test Script", data["name"])
				}
				if data["enable"].(bool) != true {
					t.Errorf("enable = %v, want true", data["enable"])
				}
			},
		},
		{
			name: "minimal config",
			config: ScriptConfig{
				ID: 2,
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["name"]; ok {
					t.Error("name should not be present")
				}
				if _, ok := data["enable"]; ok {
					t.Error("enable should not be present")
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

func TestScript_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"scripts": []}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	script := NewScript(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := script.List(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
