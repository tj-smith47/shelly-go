package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewFlood(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	flood := NewFlood(client, 0)

	if flood == nil {
		t.Fatal("NewFlood returned nil")
	}

	if flood.Type() != "flood" {
		t.Errorf("Type() = %q, want %q", flood.Type(), "flood")
	}

	if flood.Key() != "flood" {
		t.Errorf("Key() = %q, want %q", flood.Key(), "flood")
	}

	if flood.Client() != client {
		t.Error("Client() did not return the expected client")
	}

	if flood.ID() != 0 {
		t.Errorf("ID() = %d, want 0", flood.ID())
	}
}

func TestFlood_GetConfig(t *testing.T) {
	tests := []struct {
		wantName      *string
		wantAlarmMode *FloodAlarmMode
		wantHoldoff   *int
		name          string
		result        string
		id            int
	}{
		{
			name:          "full config",
			id:            0,
			result:        `{"id": 0, "name": "Bathroom Flood Sensor", "alarm_mode": "normal", "report_holdoff": 10}`,
			wantName:      ptr("Bathroom Flood Sensor"),
			wantAlarmMode: func() *FloodAlarmMode { m := FloodAlarmModeNormal; return &m }(),
			wantHoldoff:   ptr(10),
		},
		{
			name:   "minimal config",
			id:     0,
			result: `{"id": 0}`,
		},
		{
			name:          "intense alarm mode",
			id:            1,
			result:        `{"id": 1, "alarm_mode": "intense"}`,
			wantAlarmMode: func() *FloodAlarmMode { m := FloodAlarmModeIntense; return &m }(),
		},
		{
			name:          "disabled alarm mode",
			id:            0,
			result:        `{"id": 0, "alarm_mode": "disabled", "report_holdoff": 30}`,
			wantAlarmMode: func() *FloodAlarmMode { m := FloodAlarmModeDisabled; return &m }(),
			wantHoldoff:   ptr(30),
		},
		{
			name:          "rain alarm mode",
			id:            0,
			result:        `{"id": 0, "alarm_mode": "rain"}`,
			wantAlarmMode: func() *FloodAlarmMode { m := FloodAlarmModeRain; return &m }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Flood.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			flood := NewFlood(client, tt.id)

			config, err := flood.GetConfig(context.Background())
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

			if tt.wantAlarmMode != nil {
				if config.AlarmMode == nil || *config.AlarmMode != *tt.wantAlarmMode {
					t.Errorf("config.AlarmMode = %v, want %v", config.AlarmMode, *tt.wantAlarmMode)
				}
			}

			if tt.wantHoldoff != nil {
				if config.ReportHoldoff == nil || *config.ReportHoldoff != *tt.wantHoldoff {
					t.Errorf("config.ReportHoldoff = %v, want %v", config.ReportHoldoff, *tt.wantHoldoff)
				}
			}
		})
	}
}

func TestFlood_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	flood := NewFlood(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := flood.GetConfig(context.Background())
		return err
	})
}

func TestFlood_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	flood := NewFlood(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := flood.GetConfig(context.Background())
		return err
	})
}

func TestFlood_SetConfig(t *testing.T) {
	tests := []struct {
		config *FloodConfig
		name   string
		id     int
	}{
		{
			name: "set name",
			id:   0,
			config: &FloodConfig{
				Name: ptr("Kitchen Flood Sensor"),
			},
		},
		{
			name: "set alarm mode normal",
			id:   0,
			config: &FloodConfig{
				AlarmMode: func() *FloodAlarmMode { m := FloodAlarmModeNormal; return &m }(),
			},
		},
		{
			name: "set alarm mode intense",
			id:   1,
			config: &FloodConfig{
				AlarmMode: func() *FloodAlarmMode { m := FloodAlarmModeIntense; return &m }(),
			},
		},
		{
			name: "set report holdoff",
			id:   0,
			config: &FloodConfig{
				ReportHoldoff: ptr(15),
			},
		},
		{
			name: "set all fields",
			id:   0,
			config: &FloodConfig{
				Name:          ptr("Basement Sensor"),
				AlarmMode:     func() *FloodAlarmMode { m := FloodAlarmModeIntense; return &m }(),
				ReportHoldoff: ptr(20),
			},
		},
		{
			name:   "empty config",
			id:     0,
			config: &FloodConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Flood.SetConfig" {
						t.Errorf("method = %q, want %q", method, "Flood.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			flood := NewFlood(client, tt.id)

			err := flood.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestFlood_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	flood := NewFlood(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return flood.SetConfig(context.Background(), &FloodConfig{})
	})
}

func TestFlood_GetStatus(t *testing.T) {
	tests := []struct {
		name       string
		result     string
		wantErrors []string
		id         int
		wantAlarm  bool
		wantMute   bool
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
			name:       "with errors",
			id:         0,
			result:     `{"id": 0, "alarm": false, "mute": false, "errors": ["cable_unplugged"]}`,
			wantAlarm:  false,
			wantMute:   false,
			wantErrors: []string{"cable_unplugged"},
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
					if method != "Flood.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			flood := NewFlood(client, tt.id)

			status, err := flood.GetStatus(context.Background())
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

			if len(tt.wantErrors) > 0 {
				if len(status.Errors) != len(tt.wantErrors) {
					t.Errorf("status.Errors = %v, want %v", status.Errors, tt.wantErrors)
				}
				for i, err := range tt.wantErrors {
					if i < len(status.Errors) && status.Errors[i] != err {
						t.Errorf("status.Errors[%d] = %q, want %q", i, status.Errors[i], err)
					}
				}
			}
		})
	}
}

func TestFlood_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	flood := NewFlood(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := flood.GetStatus(context.Background())
		return err
	})
}

func TestFlood_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	flood := NewFlood(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := flood.GetStatus(context.Background())
		return err
	})
}

func TestFloodConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config FloodConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: FloodConfig{
				ID:            0,
				Name:          ptr("Test Sensor"),
				AlarmMode:     func() *FloodAlarmMode { m := FloodAlarmModeIntense; return &m }(),
				ReportHoldoff: ptr(15),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["id"].(float64) != 0 {
					t.Errorf("id = %v, want 0", data["id"])
				}
				if data["name"].(string) != "Test Sensor" {
					t.Errorf("name = %v, want Test Sensor", data["name"])
				}
				if data["alarm_mode"].(string) != "intense" {
					t.Errorf("alarm_mode = %v, want intense", data["alarm_mode"])
				}
				if data["report_holdoff"].(float64) != 15 {
					t.Errorf("report_holdoff = %v, want 15", data["report_holdoff"])
				}
			},
		},
		{
			name: "minimal config",
			config: FloodConfig{
				ID: 1,
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["name"]; ok {
					t.Error("name should not be present")
				}
				if _, ok := data["alarm_mode"]; ok {
					t.Error("alarm_mode should not be present")
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

func TestFloodStatus_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		wantErrors []string
		wantID     int
		wantAlarm  bool
		wantMute   bool
	}{
		{
			name:      "alarm active",
			json:      `{"id":0,"alarm":true,"mute":false}`,
			wantID:    0,
			wantAlarm: true,
			wantMute:  false,
		},
		{
			name:       "with errors",
			json:       `{"id":0,"alarm":false,"mute":false,"errors":["cable_unplugged"]}`,
			wantID:     0,
			wantAlarm:  false,
			wantMute:   false,
			wantErrors: []string{"cable_unplugged"},
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
			var status FloodStatus
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
			if len(tt.wantErrors) > 0 && len(status.Errors) != len(tt.wantErrors) {
				t.Errorf("Errors = %v, want %v", status.Errors, tt.wantErrors)
			}
		})
	}
}

func TestFlood_ContextCancellation(t *testing.T) {
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
	flood := NewFlood(client, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := flood.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestFloodAlarmMode_Constants(t *testing.T) {
	// Verify constant values match expected strings
	if FloodAlarmModeDisabled != "disabled" {
		t.Errorf("FloodAlarmModeDisabled = %q, want %q", FloodAlarmModeDisabled, "disabled")
	}
	if FloodAlarmModeNormal != "normal" {
		t.Errorf("FloodAlarmModeNormal = %q, want %q", FloodAlarmModeNormal, "normal")
	}
	if FloodAlarmModeIntense != "intense" {
		t.Errorf("FloodAlarmModeIntense = %q, want %q", FloodAlarmModeIntense, "intense")
	}
	if FloodAlarmModeRain != "rain" {
		t.Errorf("FloodAlarmModeRain = %q, want %q", FloodAlarmModeRain, "rain")
	}
}
