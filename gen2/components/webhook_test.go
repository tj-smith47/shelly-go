package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewWebhook(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	webhook := NewWebhook(client)

	if webhook == nil {
		t.Fatal("NewWebhook returned nil")
	}

	if webhook.Type() != "webhook" {
		t.Errorf("Type() = %q, want %q", webhook.Type(), "webhook")
	}

	if webhook.Key() != "webhook" {
		t.Errorf("Key() = %q, want %q", webhook.Key(), "webhook")
	}

	if webhook.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestWebhook_List(t *testing.T) {
	tests := []struct {
		name      string
		result    string
		wantRev   int
		wantCount int
	}{
		{
			name: "multiple webhooks",
			result: `{
				"rev": 5,
				"hooks": [
					{"id": 1, "cid": 0, "enable": true, "event": "switch.on", "urls": ["http://example.com/on"]},
					{"id": 2, "cid": 0, "enable": true, "event": "switch.off", "urls": ["http://example.com/off"]}
				]
			}`,
			wantRev:   5,
			wantCount: 2,
		},
		{
			name:      "no webhooks",
			result:    `{"rev": 0, "hooks": []}`,
			wantRev:   0,
			wantCount: 0,
		},
		{
			name: "webhook with all fields",
			result: `{
				"rev": 3,
				"hooks": [{
					"id": 1,
					"cid": 0,
					"enable": true,
					"event": "input.toggle_on",
					"name": "Notify input",
					"ssl_ca": "ca.pem",
					"urls": ["https://secure.example.com/webhook"],
					"active_between": ["08:00", "22:00"],
					"condition": "event.component == 'input:0'",
					"repeat_period": 60
				}]
			}`,
			wantRev:   3,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Webhook.List" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			webhook := NewWebhook(client)

			result, err := webhook.List(context.Background())
			if err != nil {
				t.Errorf("List() error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("List() returned nil result")
			}

			if result.Rev != tt.wantRev {
				t.Errorf("result.Rev = %d, want %d", result.Rev, tt.wantRev)
			}

			if len(result.Hooks) != tt.wantCount {
				t.Errorf("len(Hooks) = %d, want %d", len(result.Hooks), tt.wantCount)
			}
		})
	}
}

func TestWebhook_List_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	webhook := NewWebhook(client)
	testComponentError(t, "List", func() error {
		_, err := webhook.List(context.Background())
		return err
	})
}

func TestWebhook_List_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	webhook := NewWebhook(client)
	testComponentInvalidJSON(t, "List", func() error {
		_, err := webhook.List(context.Background())
		return err
	})
}

func TestWebhook_Create(t *testing.T) {
	tests := []struct {
		config *WebhookConfig
		name   string
		wantID int
	}{
		{
			name: "basic webhook",
			config: &WebhookConfig{
				Cid:    0,
				Enable: true,
				Event:  "switch.on",
				URLs:   []string{"http://example.com/webhook"},
			},
			wantID: 1,
		},
		{
			name: "webhook with name",
			config: &WebhookConfig{
				Cid:    0,
				Enable: true,
				Event:  "switch.off",
				Name:   ptr("My Webhook"),
				URLs:   []string{"http://example.com/webhook"},
			},
			wantID: 2,
		},
		{
			name: "webhook with TLS",
			config: &WebhookConfig{
				Cid:    0,
				Enable: true,
				Event:  "input.toggle_on",
				SSLCA:  ptr("ca.pem"),
				URLs:   []string{"https://secure.example.com/webhook"},
			},
			wantID: 3,
		},
		{
			name: "webhook with condition",
			config: &WebhookConfig{
				Cid:       0,
				Enable:    true,
				Event:     "switch.on",
				Condition: ptr("event.component == 'switch:0'"),
				URLs:      []string{"http://example.com/webhook"},
			},
			wantID: 4,
		},
		{
			name: "webhook with time range",
			config: &WebhookConfig{
				Cid:           0,
				Enable:        true,
				Event:         "switch.on",
				ActiveBetween: []string{"08:00", "22:00"},
				URLs:          []string{"http://example.com/webhook"},
			},
			wantID: 5,
		},
		{
			name: "webhook with repeat period",
			config: &WebhookConfig{
				Cid:          0,
				Enable:       true,
				Event:        "switch.on",
				RepeatPeriod: ptr(60),
				URLs:         []string{"http://example.com/webhook"},
			},
			wantID: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Webhook.Create" {
						t.Errorf("method = %q, want %q", method, "Webhook.Create")
					}
					idStr := string(rune('0' + tt.wantID))
					return jsonrpcResponse(`{"id": ` + idStr + `, "rev": 1}`)
				},
			}
			client := rpc.NewClient(tr)
			webhook := NewWebhook(client)

			result, err := webhook.Create(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if result.ID != tt.wantID {
				t.Errorf("result.ID = %d, want %d", result.ID, tt.wantID)
			}
		})
	}
}

func TestWebhook_Create_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	webhook := NewWebhook(client)
	testComponentError(t, "Create", func() error {
		_, err := webhook.Create(context.Background(), &WebhookConfig{
			Cid:   0,
			Event: "switch.on",
			URLs:  []string{"http://example.com"},
		})
		return err
	})
}

func TestWebhook_Update(t *testing.T) {
	tests := []struct {
		config *WebhookConfig
		name   string
		id     int
	}{
		{
			name: "disable webhook",
			id:   1,
			config: &WebhookConfig{
				Enable: false,
			},
		},
		{
			name: "enable webhook",
			id:   1,
			config: &WebhookConfig{
				Enable: true,
			},
		},
		{
			name: "update URLs",
			id:   2,
			config: &WebhookConfig{
				URLs: []string{"http://new-url.example.com/webhook"},
			},
		},
		{
			name: "update event",
			id:   3,
			config: &WebhookConfig{
				Event: "switch.off",
			},
		},
		{
			name: "update name",
			id:   4,
			config: &WebhookConfig{
				Name: ptr("MyWebhook"),
			},
		},
		{
			name: "update ssl_ca",
			id:   5,
			config: &WebhookConfig{
				SSLCA: ptr("ca-cert"),
			},
		},
		{
			name: "update active_between",
			id:   6,
			config: &WebhookConfig{
				ActiveBetween: []string{"09:00", "17:00"},
			},
		},
		{
			name: "update condition",
			id:   7,
			config: &WebhookConfig{
				Condition: ptr("temperature > 30"),
			},
		},
		{
			name: "update repeat_period",
			id:   8,
			config: &WebhookConfig{
				RepeatPeriod: ptr(60),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Webhook.Update" {
						t.Errorf("method = %q, want %q", method, "Webhook.Update")
					}
					return jsonrpcResponse(`{"rev": 10}`)
				},
			}
			client := rpc.NewClient(tr)
			webhook := NewWebhook(client)

			result, err := webhook.Update(context.Background(), tt.id, tt.config)
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			if result.Rev != 10 {
				t.Errorf("result.Rev = %d, want 10", result.Rev)
			}
		})
	}
}

func TestWebhook_Update_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	webhook := NewWebhook(client)
	testComponentError(t, "Update", func() error {
		_, err := webhook.Update(context.Background(), 1, &WebhookConfig{Enable: false})
		return err
	})
}

func TestWebhook_Delete(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Webhook.Delete" {
				t.Errorf("method = %q, want %q", method, "Webhook.Delete")
			}
			return jsonrpcResponse(`{"rev": 6}`)
		},
	}
	client := rpc.NewClient(tr)
	webhook := NewWebhook(client)

	result, err := webhook.Delete(context.Background(), 5)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if result.Rev != 6 {
		t.Errorf("result.Rev = %d, want 6", result.Rev)
	}
}

func TestWebhook_Delete_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	webhook := NewWebhook(client)
	testComponentError(t, "Delete", func() error {
		_, err := webhook.Delete(context.Background(), 1)
		return err
	})
}

func TestWebhook_DeleteAll(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Webhook.DeleteAll" {
				t.Errorf("method = %q, want %q", method, "Webhook.DeleteAll")
			}
			return jsonrpcResponse(`{"rev": 0}`)
		},
	}
	client := rpc.NewClient(tr)
	webhook := NewWebhook(client)

	result, err := webhook.DeleteAll(context.Background())
	if err != nil {
		t.Fatalf("DeleteAll() error = %v", err)
	}

	if result.Rev != 0 {
		t.Errorf("result.Rev = %d, want 0", result.Rev)
	}
}

func TestWebhook_DeleteAll_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	webhook := NewWebhook(client)
	testComponentError(t, "DeleteAll", func() error {
		_, err := webhook.DeleteAll(context.Background())
		return err
	})
}

func TestWebhook_ListSupported(t *testing.T) {
	tests := []struct {
		name      string
		result    string
		wantCount int
	}{
		{
			name:      "switch events",
			result:    `{"hook_types": ["switch.on", "switch.off"]}`,
			wantCount: 2,
		},
		{
			name:      "many events",
			result:    `{"hook_types": ["switch.on", "switch.off", "input.toggle_on", "input.toggle_off", "input.button_push"]}`,
			wantCount: 5,
		},
		{
			name:      "empty",
			result:    `{"hook_types": []}`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Webhook.ListSupported" {
						t.Errorf("method = %q, want %q", method, "Webhook.ListSupported")
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			webhook := NewWebhook(client)

			result, err := webhook.ListSupported(context.Background())
			if err != nil {
				t.Errorf("ListSupported() error = %v", err)
				return
			}

			if len(result.HookTypes) != tt.wantCount {
				t.Errorf("len(HookTypes) = %d, want %d", len(result.HookTypes), tt.wantCount)
			}
		})
	}
}

func TestWebhook_ListSupported_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	webhook := NewWebhook(client)
	testComponentError(t, "ListSupported", func() error {
		_, err := webhook.ListSupported(context.Background())
		return err
	})
}

func TestWebhookConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config WebhookConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: WebhookConfig{
				ID:            ptr(1),
				Cid:           0,
				Enable:        true,
				Event:         "switch.on",
				Name:          ptr("Test Webhook"),
				SSLCA:         ptr("ca.pem"),
				URLs:          []string{"https://example.com/webhook"},
				ActiveBetween: []string{"09:00", "21:00"},
				Condition:     ptr("true"),
				RepeatPeriod:  ptr(30),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["cid"].(float64) != 0 {
					t.Errorf("cid = %v, want 0", data["cid"])
				}
				if data["enable"].(bool) != true {
					t.Errorf("enable = %v, want true", data["enable"])
				}
				if data["event"].(string) != "switch.on" {
					t.Errorf("event = %v, want switch.on", data["event"])
				}
				urls := data["urls"].([]any)
				if len(urls) != 1 {
					t.Errorf("len(urls) = %d, want 1", len(urls))
				}
			},
		},
		{
			name: "minimal config",
			config: WebhookConfig{
				Cid:    0,
				Enable: true,
				Event:  "switch.off",
				URLs:   []string{"http://example.com"},
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["name"]; ok {
					t.Error("name should not be present")
				}
				if _, ok := data["ssl_ca"]; ok {
					t.Error("ssl_ca should not be present")
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

func TestWebhook_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"rev": 0, "hooks": []}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	webhook := NewWebhook(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := webhook.List(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
