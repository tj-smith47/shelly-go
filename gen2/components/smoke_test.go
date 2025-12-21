package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewSmoke(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	smoke := NewSmoke(client, 0)

	if smoke == nil {
		t.Fatal("NewSmoke returned nil")
	}

	if smoke.Type() != "smoke" {
		t.Errorf("Type() = %q, want %q", smoke.Type(), "smoke")
	}

	if smoke.Key() != "smoke" {
		t.Errorf("Key() = %q, want %q", smoke.Key(), "smoke")
	}

	if smoke.Client() != client {
		t.Error("Client() did not return the expected client")
	}

	if smoke.ID() != 0 {
		t.Errorf("ID() = %d, want 0", smoke.ID())
	}
}

func TestSmoke_GetConfig(t *testing.T) {
	tests := []struct {
		wantName *string
		name     string
		result   string
		id       int
	}{
		{
			name:     "full config",
			id:       0,
			result:   `{"id": 0, "name": "Kitchen Smoke Detector"}`,
			wantName: ptr("Kitchen Smoke Detector"),
		},
		{
			name:   "minimal config",
			id:     0,
			result: `{"id": 0}`,
		},
		{
			name:     "different ID",
			id:       1,
			result:   `{"id": 1, "name": "Bedroom Smoke Detector"}`,
			wantName: ptr("Bedroom Smoke Detector"),
		},
		{
			name:   "null name",
			id:     0,
			result: `{"id": 0, "name": null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Smoke.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			smoke := NewSmoke(client, tt.id)

			config, err := smoke.GetConfig(context.Background())
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
			} else if config.Name != nil {
				t.Errorf("config.Name = %v, want nil", *config.Name)
			}
		})
	}
}

func TestSmoke_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	smoke := NewSmoke(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := smoke.GetConfig(context.Background())
		return err
	})
}

func TestSmoke_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	smoke := NewSmoke(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := smoke.GetConfig(context.Background())
		return err
	})
}

func TestSmoke_SetConfig(t *testing.T) {
	tests := []struct {
		config *SmokeConfig
		name   string
		id     int
	}{
		{
			name: "set name",
			id:   0,
			config: &SmokeConfig{
				Name: ptr("Updated Smoke Detector"),
			},
		},
		{
			name:   "empty config",
			id:     0,
			config: &SmokeConfig{},
		},
		{
			name: "different ID",
			id:   1,
			config: &SmokeConfig{
				Name: ptr("Garage Smoke Detector"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Smoke.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Smoke.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			smoke := NewSmoke(client, tt.id)

			err := smoke.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestSmoke_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	smoke := NewSmoke(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return smoke.SetConfig(context.Background(), &SmokeConfig{})
	})
}

func TestSmoke_GetStatus(t *testing.T) {
	tests := []struct {
		name      string
		result    string
		id        int
		wantAlarm bool
		wantMute  bool
	}{
		{
			name:      "no alarm",
			id:        0,
			result:    `{"id": 0, "alarm": false, "mute": false}`,
			wantAlarm: false,
			wantMute:  false,
		},
		{
			name:      "alarm active",
			id:        0,
			result:    `{"id": 0, "alarm": true, "mute": false}`,
			wantAlarm: true,
			wantMute:  false,
		},
		{
			name:      "alarm active and muted",
			id:        0,
			result:    `{"id": 0, "alarm": true, "mute": true}`,
			wantAlarm: true,
			wantMute:  true,
		},
		{
			name:      "different ID",
			id:        1,
			result:    `{"id": 1, "alarm": false, "mute": false}`,
			wantAlarm: false,
			wantMute:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Smoke.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			smoke := NewSmoke(client, tt.id)

			status, err := smoke.GetStatus(context.Background())
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

			if status.Alarm != tt.wantAlarm {
				t.Errorf("status.Alarm = %v, want %v", status.Alarm, tt.wantAlarm)
			}

			if status.Mute != tt.wantMute {
				t.Errorf("status.Mute = %v, want %v", status.Mute, tt.wantMute)
			}
		})
	}
}

func TestSmoke_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	smoke := NewSmoke(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := smoke.GetStatus(context.Background())
		return err
	})
}

func TestSmoke_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	smoke := NewSmoke(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := smoke.GetStatus(context.Background())
		return err
	})
}

func TestSmoke_Mute(t *testing.T) {
	tests := []struct {
		name string
		id   int
	}{
		{
			name: "mute ID 0",
			id:   0,
		},
		{
			name: "mute ID 1",
			id:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Smoke.Mute" {
						t.Errorf("method = %q, want %q", method, "Smoke.Mute")
					}
					return jsonrpcResponse(`null`)
				},
			}
			client := rpc.NewClient(tr)
			smoke := NewSmoke(client, tt.id)

			err := smoke.Mute(context.Background())
			if err != nil {
				t.Fatalf("Mute() error = %v", err)
			}
		})
	}
}

func TestSmoke_Mute_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	smoke := NewSmoke(client, 0)
	testComponentError(t, "Mute", func() error {
		return smoke.Mute(context.Background())
	})
}

func TestSmokeConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config SmokeConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: SmokeConfig{
				ID:   0,
				Name: ptr("Test Detector"),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["id"].(float64) != 0 {
					t.Errorf("id = %v, want 0", data["id"])
				}
				if data["name"].(string) != "Test Detector" {
					t.Errorf("name = %v, want Test Detector", data["name"])
				}
			},
		},
		{
			name: "minimal config",
			config: SmokeConfig{
				ID: 1,
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["name"]; ok {
					t.Error("name should not be present")
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

func TestSmokeStatus_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantID    int
		wantAlarm bool
		wantMute  bool
	}{
		{
			name:      "alarm active",
			json:      `{"id":0,"alarm":true,"mute":false}`,
			wantID:    0,
			wantAlarm: true,
			wantMute:  false,
		},
		{
			name:      "alarm muted",
			json:      `{"id":0,"alarm":true,"mute":true}`,
			wantID:    0,
			wantAlarm: true,
			wantMute:  true,
		},
		{
			name:      "no alarm",
			json:      `{"id":1,"alarm":false,"mute":false}`,
			wantID:    1,
			wantAlarm: false,
			wantMute:  false,
		},
		{
			name:      "with unknown fields",
			json:      `{"id":0,"alarm":true,"mute":false,"future_field":"value"}`,
			wantID:    0,
			wantAlarm: true,
			wantMute:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status SmokeStatus
			if err := json.Unmarshal([]byte(tt.json), &status); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if status.ID != tt.wantID {
				t.Errorf("ID = %d, want %d", status.ID, tt.wantID)
			}
			if status.Alarm != tt.wantAlarm {
				t.Errorf("Alarm = %v, want %v", status.Alarm, tt.wantAlarm)
			}
			if status.Mute != tt.wantMute {
				t.Errorf("Mute = %v, want %v", status.Mute, tt.wantMute)
			}
		})
	}
}

func TestSmoke_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					_ = req.GetMethod()
					select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"id": 0, "alarm": false, "mute": false}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	smoke := NewSmoke(client, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := smoke.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
