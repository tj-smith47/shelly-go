package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewEM1(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	em1 := NewEM1(client, 0)

	if em1 == nil {
		t.Fatal("NewEM1 returned nil")
	}

	if em1.Type() != "em1" {
		t.Errorf("Type() = %q, want %q", em1.Type(), "em1")
	}

	if em1.ID() != 0 {
		t.Errorf("ID() = %d, want %d", em1.ID(), 0)
	}

	if em1.Key() != "em1:0" {
		t.Errorf("Key() = %q, want %q", em1.Key(), "em1:0")
	}
}

func TestEM1_GetConfig(t *testing.T) {
	tests := []struct {
		wantName    *string
		wantCTType  *string
		wantReverse *bool
		name        string
		result      string
		wantID      int
	}{
		{
			name: "full config",
			result: `{
				"id": 0,
				"name": "Phase A Meter",
				"ct_type": "120A",
				"reverse": false
			}`,
			wantID:      0,
			wantName:    ptr("Phase A Meter"),
			wantCTType:  ptr("120A"),
			wantReverse: ptr(false),
		},
		{
			name: "bidirectional config",
			result: `{
				"id": 1,
				"name": "Solar Meter",
				"ct_type": "400A",
				"reverse": true
			}`,
			wantID:      1,
			wantName:    ptr("Solar Meter"),
			wantCTType:  ptr("400A"),
			wantReverse: ptr(true),
		},
		{
			name:       "minimal config",
			result:     `{"id": 0, "ct_type": "120A"}`,
			wantID:     0,
			wantCTType: ptr("120A"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "EM1.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			em1 := NewEM1(client, tt.wantID)

			config, err := em1.GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if config == nil {
				t.Fatal("GetConfig() returned nil config")
			}

			if config.ID != tt.wantID {
				t.Errorf("config.ID = %d, want %d", config.ID, tt.wantID)
			}

			if tt.wantName != nil && (config.Name == nil || *config.Name != *tt.wantName) {
				t.Errorf("config.Name = %v, want %v", config.Name, tt.wantName)
			}

			if tt.wantCTType != nil && (config.CTType == nil || *config.CTType != *tt.wantCTType) {
				t.Errorf("config.CTType = %v, want %v", config.CTType, tt.wantCTType)
			}

			if tt.wantReverse != nil && (config.Reverse == nil || *config.Reverse != *tt.wantReverse) {
				t.Errorf("config.Reverse = %v, want %v", config.Reverse, tt.wantReverse)
			}
		})
	}
}

func TestEM1_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em1 := NewEM1(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := em1.GetConfig(context.Background())
		return err
	})
}

func TestEM1_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	em1 := NewEM1(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := em1.GetConfig(context.Background())
		return err
	})
}

func TestEM1_SetConfig(t *testing.T) {
	config := &EM1Config{
		ID:      0,
		Name:    ptr("Main Phase Meter"),
		CTType:  ptr("120A"),
		Reverse: ptr(false),
	}

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "EM1.SetConfig" {
				t.Errorf("method = %q, want %q", method, "EM1.SetConfig")
			}
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	em1 := NewEM1(client, 0)

	err := em1.SetConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestEM1_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em1 := NewEM1(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return em1.SetConfig(context.Background(), &EM1Config{ID: 0})
	})
}

func TestEM1_GetStatus(t *testing.T) {
	tests := []struct {
		wantPF        *float64
		wantFreq      *float64
		name          string
		result        string
		wantErrors    []string
		wantID        int
		wantVoltage   float64
		wantCurrent   float64
		wantActPower  float64
		wantAprtPower float64
	}{
		{
			name: "normal operation",
			result: `{
				"id": 0,
				"voltage": 230.5,
				"current": 10.5,
				"act_power": 2420.25,
				"aprt_power": 2420.25,
				"pf": 1.0,
				"freq": 50.0
			}`,
			wantID:        0,
			wantVoltage:   230.5,
			wantCurrent:   10.5,
			wantActPower:  2420.25,
			wantAprtPower: 2420.25,
			wantPF:        ptr(1.0),
			wantFreq:      ptr(50.0),
		},
		{
			name: "low power factor",
			result: `{
				"id": 0,
				"voltage": 230.0,
				"current": 5.0,
				"act_power": 920.0,
				"aprt_power": 1150.0,
				"pf": 0.8,
				"freq": 50.0
			}`,
			wantID:        0,
			wantVoltage:   230.0,
			wantCurrent:   5.0,
			wantActPower:  920.0,
			wantAprtPower: 1150.0,
			wantPF:        ptr(0.8),
			wantFreq:      ptr(50.0),
		},
		{
			name: "bidirectional - generating",
			result: `{
				"id": 0,
				"voltage": 240.0,
				"current": -8.0,
				"act_power": -1920.0,
				"aprt_power": 1920.0,
				"pf": -1.0,
				"freq": 60.0
			}`,
			wantID:        0,
			wantVoltage:   240.0,
			wantCurrent:   -8.0,
			wantActPower:  -1920.0,
			wantAprtPower: 1920.0,
			wantPF:        ptr(-1.0),
			wantFreq:      ptr(60.0),
		},
		{
			name: "no load",
			result: `{
				"id": 0,
				"voltage": 230.0,
				"current": 0.0,
				"act_power": 0.0,
				"aprt_power": 0.0,
				"freq": 50.0
			}`,
			wantID:        0,
			wantVoltage:   230.0,
			wantCurrent:   0.0,
			wantActPower:  0.0,
			wantAprtPower: 0.0,
			wantFreq:      ptr(50.0),
		},
		{
			name: "with errors",
			result: `{
				"id": 0,
				"voltage": 250.0,
				"current": 5.0,
				"act_power": 1250.0,
				"aprt_power": 1250.0,
				"errors": ["ct_type_not_set", "out_of_range:voltage"]
			}`,
			wantID:        0,
			wantVoltage:   250.0,
			wantCurrent:   5.0,
			wantActPower:  1250.0,
			wantAprtPower: 1250.0,
			wantErrors:    []string{"ct_type_not_set", "out_of_range:voltage"},
		},
		{
			name: "high current",
			result: `{
				"id": 0,
				"voltage": 230.0,
				"current": 50.0,
				"act_power": 11500.0,
				"aprt_power": 11500.0,
				"pf": 1.0,
				"freq": 50.0
			}`,
			wantID:        0,
			wantVoltage:   230.0,
			wantCurrent:   50.0,
			wantActPower:  11500.0,
			wantAprtPower: 11500.0,
			wantPF:        ptr(1.0),
			wantFreq:      ptr(50.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "EM1.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			em1 := NewEM1(client, tt.wantID)

			status, err := em1.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if status.ID != tt.wantID {
				t.Errorf("status.ID = %d, want %d", status.ID, tt.wantID)
			}

			if status.Voltage != tt.wantVoltage {
				t.Errorf("status.Voltage = %.2f, want %.2f", status.Voltage, tt.wantVoltage)
			}

			if status.Current != tt.wantCurrent {
				t.Errorf("status.Current = %.2f, want %.2f", status.Current, tt.wantCurrent)
			}

			if status.ActPower != tt.wantActPower {
				t.Errorf("status.ActPower = %.2f, want %.2f", status.ActPower, tt.wantActPower)
			}

			if status.AprtPower != tt.wantAprtPower {
				t.Errorf("status.AprtPower = %.2f, want %.2f", status.AprtPower, tt.wantAprtPower)
			}

			if tt.wantPF != nil {
				if status.PF == nil {
					t.Error("status.PF is nil, want non-nil")
				} else if *status.PF != *tt.wantPF {
					t.Errorf("status.PF = %.3f, want %.3f", *status.PF, *tt.wantPF)
				}
			}

			if tt.wantFreq != nil {
				if status.Freq == nil {
					t.Error("status.Freq is nil, want non-nil")
				} else if *status.Freq != *tt.wantFreq {
					t.Errorf("status.Freq = %.2f, want %.2f", *status.Freq, *tt.wantFreq)
				}
			}

			if len(tt.wantErrors) > 0 {
				if len(status.Errors) != len(tt.wantErrors) {
					t.Errorf("len(status.Errors) = %d, want %d", len(status.Errors), len(tt.wantErrors))
				} else {
					for i, want := range tt.wantErrors {
						if status.Errors[i] != want {
							t.Errorf("status.Errors[%d] = %q, want %q", i, status.Errors[i], want)
						}
					}
				}
			}
		})
	}
}

func TestEM1_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em1 := NewEM1(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := em1.GetStatus(context.Background())
		return err
	})
}

func TestEM1_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	em1 := NewEM1(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := em1.GetStatus(context.Background())
		return err
	})
}

func TestEM1_GetCTTypes(t *testing.T) {
	result := `{"types": ["120A", "400A"]}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "EM1.GetCTTypes" {
				t.Errorf("method = %q, want %q", method, "EM1.GetCTTypes")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	em1 := NewEM1(client, 0)

	ctTypes, err := em1.GetCTTypes(context.Background())
	if err != nil {
		t.Fatalf("GetCTTypes() error = %v", err)
	}

	if ctTypes == nil {
		t.Fatal("GetCTTypes() returned nil")
	}

	wantTypes := []string{"120A", "400A"}
	if len(ctTypes.Types) != len(wantTypes) {
		t.Errorf("len(ctTypes.Types) = %d, want %d", len(ctTypes.Types), len(wantTypes))
	}

	for i, want := range wantTypes {
		if i >= len(ctTypes.Types) {
			break
		}
		if ctTypes.Types[i] != want {
			t.Errorf("ctTypes.Types[%d] = %q, want %q", i, ctTypes.Types[i], want)
		}
	}
}

func TestEM1_GetCTTypes_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em1 := NewEM1(client, 0)
	testComponentError(t, "GetCTTypes", func() error {
		_, err := em1.GetCTTypes(context.Background())
		return err
	})
}

func TestEM1_GetCTTypes_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	em1 := NewEM1(client, 0)
	testComponentInvalidJSON(t, "GetCTTypes", func() error {
		_, err := em1.GetCTTypes(context.Background())
		return err
	})
}
