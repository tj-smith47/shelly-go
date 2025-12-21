package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewCover(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	cover := NewCover(client, 0)

	if cover == nil {
		t.Fatal("NewCover returned nil")
	}

	if cover.Type() != "cover" {
		t.Errorf("Type() = %q, want %q", cover.Type(), "cover")
	}

	if cover.ID() != 0 {
		t.Errorf("ID() = %d, want %d", cover.ID(), 0)
	}

	if cover.Key() != "cover:0" {
		t.Errorf("Key() = %q, want %q", cover.Key(), "cover:0")
	}
}

func TestCover_Open(t *testing.T) {
	tests := []struct {
		duration *float64
		name     string
	}{
		{
			name:     "open fully",
			duration: nil,
		},
		{
			name:     "open for 5 seconds",
			duration: ptr(5.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Cover.Open" {
						t.Errorf("method = %q, want %q", method, "Cover.Open")
					}
					return jsonrpcResponse(`{}`)
				},
			}
			client := rpc.NewClient(tr)
			cover := NewCover(client, 0)

			err := cover.Open(context.Background(), tt.duration)
			if err != nil {
				t.Fatalf("Open() error = %v", err)
			}
		})
	}
}

func TestCover_Open_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	cover := NewCover(client, 0)

	testComponentError(t, "Open", func() error {
		return cover.Open(context.Background(), nil)
	})
}

func TestCover_Close(t *testing.T) {
	tests := []struct {
		duration *float64
		name     string
	}{
		{
			name:     "close fully",
			duration: nil,
		},
		{
			name:     "close for 3 seconds",
			duration: ptr(3.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Cover.Close" {
						t.Errorf("method = %q, want %q", method, "Cover.Close")
					}
					return jsonrpcResponse(`{}`)
				},
			}
			client := rpc.NewClient(tr)
			cover := NewCover(client, 0)

			err := cover.Close(context.Background(), tt.duration)
			if err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})
	}
}

func TestCover_Close_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	cover := NewCover(client, 0)

	testComponentError(t, "Close", func() error {
		return cover.Close(context.Background(), nil)
	})
}

func TestCover_Stop(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Cover.Stop" {
				t.Errorf("method = %q, want %q", method, "Cover.Stop")
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	cover := NewCover(client, 0)

	err := cover.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestCover_Stop_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	cover := NewCover(client, 0)

	testComponentError(t, "Stop", func() error {
		return cover.Stop(context.Background())
	})
}

func TestCover_GoToPosition(t *testing.T) {
	tests := []struct {
		name string
		pos  int
	}{
		{
			name: "go to 50%",
			pos:  50,
		},
		{
			name: "go to fully closed",
			pos:  0,
		},
		{
			name: "go to fully open",
			pos:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Cover.GoToPosition" {
						t.Errorf("method = %q, want %q", method, "Cover.GoToPosition")
					}
					return jsonrpcResponse(`{}`)
				},
			}
			client := rpc.NewClient(tr)
			cover := NewCover(client, 0)

			err := cover.GoToPosition(context.Background(), tt.pos)
			if err != nil {
				t.Fatalf("GoToPosition() error = %v", err)
			}
		})
	}
}

func TestCover_GoToPosition_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	cover := NewCover(client, 0)

	testComponentError(t, "GoToPosition", func() error {
		return cover.GoToPosition(context.Background(), 50)
	})
}

func TestCover_Calibrate(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Cover.Calibrate" {
				t.Errorf("method = %q, want %q", method, "Cover.Calibrate")
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	cover := NewCover(client, 0)

	err := cover.Calibrate(context.Background())
	if err != nil {
		t.Fatalf("Calibrate() error = %v", err)
	}
}

func TestCover_Calibrate_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	cover := NewCover(client, 0)

	testComponentError(t, "Calibrate", func() error {
		return cover.Calibrate(context.Background())
	})
}

func TestCover_ResetCounters(t *testing.T) {
	tests := []struct {
		name         string
		counterTypes []string
	}{
		{
			name:         "reset specific counter",
			counterTypes: []string{"aenergy"},
		},
		{
			name:         "reset all counters",
			counterTypes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Cover.ResetCounters" {
						t.Errorf("method = %q, want %q", method, "Cover.ResetCounters")
					}
					return jsonrpcResponse(`{}`)
				},
			}
			client := rpc.NewClient(tr)
			cover := NewCover(client, 0)

			err := cover.ResetCounters(context.Background(), tt.counterTypes)
			if err != nil {
				t.Errorf("ResetCounters() error = %v", err)
			}
		})
	}
}

func TestCover_ResetCounters_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	cover := NewCover(client, 0)

	testComponentError(t, "ResetCounters", func() error {
		return cover.ResetCounters(context.Background(), []string{"aenergy"})
	})
}

func TestCover_GetConfig(t *testing.T) {
	result := `{
		"id": 0,
		"name": "Living Room Blinds",
		"initial_state": "stopped",
		"motor_idle_confirm_timeout": 0.5,
		"motor_move_timeout": 60.0,
		"obstruction_detection_level": 75,
		"swap_inputs": false,
		"invert_directions": false
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Cover.GetConfig" {
				t.Errorf("method = %q, want %q", method, "Cover.GetConfig")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	cover := NewCover(client, 0)

	config, err := cover.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if config.ID != 0 {
		t.Errorf("ID = %d, want 0", config.ID)
	}

	if config.Name == nil || *config.Name != "Living Room Blinds" {
		t.Errorf("Name = %v, want %q", config.Name, "Living Room Blinds")
	}

	if config.InitialState == nil || *config.InitialState != "stopped" {
		t.Errorf("InitialState = %v, want %q", config.InitialState, "stopped")
	}

	if config.ObstructionDetectionLevel == nil || *config.ObstructionDetectionLevel != 75 {
		t.Errorf("ObstructionDetectionLevel = %v, want 75", config.ObstructionDetectionLevel)
	}

	if config.SwapInputs == nil || *config.SwapInputs {
		t.Errorf("SwapInputs = %v, want false", config.SwapInputs)
	}
}

func TestCover_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	cover := NewCover(client, 0)

	testComponentError(t, "GetConfig", func() error {
		_, err := cover.GetConfig(context.Background())
		return err
	})
}

func TestCover_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	cover := NewCover(client, 0)

	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := cover.GetConfig(context.Background())
		return err
	})
}

func TestCover_SetConfig(t *testing.T) {
	expectedConfig := &CoverConfig{
		ID:                        0,
		Name:                      ptr("Bedroom Blinds"),
		ObstructionDetectionLevel: ptr(80),
		InvertDirections:          ptr(true),
	}

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Cover.SetConfig" {
				t.Errorf("method = %q, want %q", method, "Cover.SetConfig")
			}
			return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	cover := NewCover(client, 0)

	err := cover.SetConfig(context.Background(), expectedConfig)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestCover_SetConfig_AutoSetID(t *testing.T) {
	// Config without ID should be auto-set
	config := &CoverConfig{
		Name: ptr("Test"),
	}

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					_ = req.GetMethod()
					return jsonrpcResponse(`{}`)
		},
	}
	client := rpc.NewClient(tr)
	cover := NewCover(client, 0)

	err := cover.SetConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestCover_GetStatus(t *testing.T) {
	result := `{
		"id": 0,
		"source": "http",
		"state": "stopped",
		"apower": 0.0,
		"voltage": 230.1,
		"current": 0.0,
		"current_pos": 50,
		"target_pos": 50,
		"move_timeout": false,
		"last_direction": "close",
		"temperature": {
			"tC": 42.5,
			"tF": 108.5
		},
		"errors": []
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "Cover.GetStatus" {
				t.Errorf("method = %q, want %q", method, "Cover.GetStatus")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	cover := NewCover(client, 0)

	status, err := cover.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status.ID != 0 {
		t.Errorf("ID = %d, want 0", status.ID)
	}

	if status.Source != "http" {
		t.Errorf("Source = %q, want %q", status.Source, "http")
	}

	if status.State != "stopped" {
		t.Errorf("State = %q, want %q", status.State, "stopped")
	}

	if status.CurrentPos == nil || *status.CurrentPos != 50 {
		t.Errorf("CurrentPos = %v, want 50", status.CurrentPos)
	}

	if status.TargetPos == nil || *status.TargetPos != 50 {
		t.Errorf("TargetPos = %v, want 50", status.TargetPos)
	}

	if status.LastDirection == nil || *status.LastDirection != "close" {
		t.Errorf("LastDirection = %v, want %q", status.LastDirection, "close")
	}

	if status.Temperature == nil {
		t.Fatal("Temperature is nil")
	}

	if status.Temperature.TC == nil || *status.Temperature.TC != 42.5 {
		t.Errorf("Temperature.TC = %v, want 42.5", status.Temperature.TC)
	}

	if len(status.Errors) != 0 {
		t.Errorf("Errors length = %d, want 0", len(status.Errors))
	}
}

func TestCover_GetStatus_WithErrors(t *testing.T) {
	result := `{
		"id": 0,
		"source": "init",
		"state": "stopped",
		"errors": ["obstruction", "overcurrent"]
	}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					_ = req.GetMethod()
					return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	cover := NewCover(client, 0)

	status, err := cover.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if len(status.Errors) != 2 {
		t.Errorf("Errors length = %d, want 2", len(status.Errors))
	}

	if status.Errors[0] != "obstruction" {
		t.Errorf("Errors[0] = %q, want %q", status.Errors[0], "obstruction")
	}

	if status.Errors[1] != "overcurrent" {
		t.Errorf("Errors[1] = %q, want %q", status.Errors[1], "overcurrent")
	}
}

func TestCover_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	cover := NewCover(client, 0)

	testComponentError(t, "GetStatus", func() error {
		_, err := cover.GetStatus(context.Background())
		return err
	})
}

func TestCover_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	cover := NewCover(client, 0)

	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := cover.GetStatus(context.Background())
		return err
	})
}
