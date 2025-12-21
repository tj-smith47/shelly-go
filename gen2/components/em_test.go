package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewEM(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	em := NewEM(client, 0)

	if em == nil {
		t.Fatal("NewEM returned nil")
	}

	if em.Type() != "em" {
		t.Errorf("Type() = %q, want %q", em.Type(), "em")
	}

	if em.ID() != 0 {
		t.Errorf("ID() = %d, want %d", em.ID(), 0)
	}

	if em.Key() != "em:0" {
		t.Errorf("Key() = %q, want %q", em.Key(), "em:0")
	}
}

func TestEM_GetConfig(t *testing.T) {
	tests := []struct {
		wantName            *string
		wantBlinkMode       *string
		wantPhaseSelector   *string
		wantMonitorPhaseSeq *bool
		wantCTType          *string
		name                string
		result              string
		wantID              int
	}{
		{
			name: "full config",
			result: `{
				"id": 0,
				"name": "Main 3-Phase Meter",
				"blink_mode_selector": "active_energy",
				"phase_selector": "all",
				"monitor_phase_sequence": true,
				"ct_type": "120A"
			}`,
			wantID:              0,
			wantName:            ptr("Main 3-Phase Meter"),
			wantBlinkMode:       ptr("active_energy"),
			wantPhaseSelector:   ptr("all"),
			wantMonitorPhaseSeq: ptr(true),
			wantCTType:          ptr("120A"),
		},
		{
			name: "minimal config",
			result: `{
				"id": 0,
				"ct_type": "400A"
			}`,
			wantID:     0,
			wantCTType: ptr("400A"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "EM.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			em := NewEM(client, 0)

			config, err := em.GetConfig(context.Background())
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

			if tt.wantBlinkMode != nil && (config.BlinkModeSelector == nil || *config.BlinkModeSelector != *tt.wantBlinkMode) {
				t.Errorf("config.BlinkModeSelector = %v, want %v", config.BlinkModeSelector, tt.wantBlinkMode)
			}

			if tt.wantPhaseSelector != nil && (config.PhaseSelector == nil || *config.PhaseSelector != *tt.wantPhaseSelector) {
				t.Errorf("config.PhaseSelector = %v, want %v", config.PhaseSelector, tt.wantPhaseSelector)
			}

			if tt.wantMonitorPhaseSeq != nil && (config.MonitorPhaseSequence == nil || *config.MonitorPhaseSequence != *tt.wantMonitorPhaseSeq) {
				t.Errorf("config.MonitorPhaseSequence = %v, want %v", config.MonitorPhaseSequence, tt.wantMonitorPhaseSeq)
			}

			if tt.wantCTType != nil && (config.CTType == nil || *config.CTType != *tt.wantCTType) {
				t.Errorf("config.CTType = %v, want %v", config.CTType, tt.wantCTType)
			}
		})
	}
}

func TestEM_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em := NewEM(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := em.GetConfig(context.Background())
		return err
	})
}

func TestEM_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	em := NewEM(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := em.GetConfig(context.Background())
		return err
	})
}

func TestEM_SetConfig(t *testing.T) {
	config := &EMConfig{
		ID:                   0,
		Name:                 ptr("Industrial Meter"),
		BlinkModeSelector:    ptr("apparent_energy"),
		PhaseSelector:        ptr("all"),
		MonitorPhaseSequence: ptr(true),
		CTType:               ptr("120A"),
	}

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "EM.SetConfig" {
				t.Errorf("method = %q, want %q", method, "EM.SetConfig")
			}
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	em := NewEM(client, 0)

	err := em.SetConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}
}

func TestEM_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em := NewEM(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return em.SetConfig(context.Background(), &EMConfig{ID: 0})
	})
}

func TestEM_GetStatus(t *testing.T) {
	tests := []struct {
		wantNCurrent         *float64
		name                 string
		result               string
		wantErrors           []string
		wantBVoltage         float64
		wantAActivePower     float64
		wantACurrent         float64
		wantBCurrent         float64
		wantBActivePower     float64
		wantCVoltage         float64
		wantCCurrent         float64
		wantCActivePower     float64
		wantTotalActivePower float64
		wantAVoltage         float64
		wantID               int
	}{
		{
			name: "balanced 3-phase load",
			result: `{
				"id": 0,
				"a_voltage": 230.0,
				"a_current": 10.0,
				"a_act_power": 2300.0,
				"a_aprt_power": 2300.0,
				"a_pf": 1.0,
				"a_freq": 50.0,
				"b_voltage": 230.0,
				"b_current": 10.0,
				"b_act_power": 2300.0,
				"b_aprt_power": 2300.0,
				"b_pf": 1.0,
				"b_freq": 50.0,
				"c_voltage": 230.0,
				"c_current": 10.0,
				"c_act_power": 2300.0,
				"c_aprt_power": 2300.0,
				"c_pf": 1.0,
				"c_freq": 50.0,
				"n_current": 0.1,
				"total_current": 30.0,
				"total_act_power": 6900.0,
				"total_aprt_power": 6900.0
			}`,
			wantID:               0,
			wantAVoltage:         230.0,
			wantACurrent:         10.0,
			wantAActivePower:     2300.0,
			wantBVoltage:         230.0,
			wantBCurrent:         10.0,
			wantBActivePower:     2300.0,
			wantCVoltage:         230.0,
			wantCCurrent:         10.0,
			wantCActivePower:     2300.0,
			wantTotalActivePower: 6900.0,
			wantNCurrent:         ptr(0.1),
		},
		{
			name: "unbalanced load",
			result: `{
				"id": 0,
				"a_voltage": 232.0,
				"a_current": 15.0,
				"a_act_power": 3480.0,
				"a_aprt_power": 3480.0,
				"b_voltage": 229.0,
				"b_current": 8.0,
				"b_act_power": 1832.0,
				"b_aprt_power": 1832.0,
				"c_voltage": 231.0,
				"c_current": 12.0,
				"c_act_power": 2772.0,
				"c_aprt_power": 2772.0,
				"n_current": 5.5,
				"total_current": 35.0,
				"total_act_power": 8084.0,
				"total_aprt_power": 8084.0
			}`,
			wantID:               0,
			wantAVoltage:         232.0,
			wantACurrent:         15.0,
			wantAActivePower:     3480.0,
			wantBVoltage:         229.0,
			wantBCurrent:         8.0,
			wantBActivePower:     1832.0,
			wantCVoltage:         231.0,
			wantCCurrent:         12.0,
			wantCActivePower:     2772.0,
			wantTotalActivePower: 8084.0,
			wantNCurrent:         ptr(5.5),
		},
		{
			name: "with errors",
			result: `{
				"id": 0,
				"a_voltage": 250.0,
				"a_current": 5.0,
				"a_act_power": 1250.0,
				"a_aprt_power": 1250.0,
				"b_voltage": 230.0,
				"b_current": 5.0,
				"b_act_power": 1150.0,
				"b_aprt_power": 1150.0,
				"c_voltage": 230.0,
				"c_current": 5.0,
				"c_act_power": 1150.0,
				"c_aprt_power": 1150.0,
				"total_current": 15.0,
				"total_act_power": 3550.0,
				"total_aprt_power": 3550.0,
				"errors": ["ct_type_not_set", "out_of_range:voltage"]
			}`,
			wantID:               0,
			wantAVoltage:         250.0,
			wantACurrent:         5.0,
			wantAActivePower:     1250.0,
			wantBVoltage:         230.0,
			wantBCurrent:         5.0,
			wantBActivePower:     1150.0,
			wantCVoltage:         230.0,
			wantCCurrent:         5.0,
			wantCActivePower:     1150.0,
			wantTotalActivePower: 3550.0,
			wantErrors:           []string{"ct_type_not_set", "out_of_range:voltage"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "EM.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			em := NewEM(client, 0)

			status, err := em.GetStatus(context.Background())
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

			// Phase A
			if status.AVoltage != tt.wantAVoltage {
				t.Errorf("status.AVoltage = %.2f, want %.2f", status.AVoltage, tt.wantAVoltage)
			}
			if status.ACurrent != tt.wantACurrent {
				t.Errorf("status.ACurrent = %.2f, want %.2f", status.ACurrent, tt.wantACurrent)
			}
			if status.AActivePower != tt.wantAActivePower {
				t.Errorf("status.AActivePower = %.2f, want %.2f", status.AActivePower, tt.wantAActivePower)
			}

			// Phase B
			if status.BVoltage != tt.wantBVoltage {
				t.Errorf("status.BVoltage = %.2f, want %.2f", status.BVoltage, tt.wantBVoltage)
			}
			if status.BCurrent != tt.wantBCurrent {
				t.Errorf("status.BCurrent = %.2f, want %.2f", status.BCurrent, tt.wantBCurrent)
			}
			if status.BActivePower != tt.wantBActivePower {
				t.Errorf("status.BActivePower = %.2f, want %.2f", status.BActivePower, tt.wantBActivePower)
			}

			// Phase C
			if status.CVoltage != tt.wantCVoltage {
				t.Errorf("status.CVoltage = %.2f, want %.2f", status.CVoltage, tt.wantCVoltage)
			}
			if status.CCurrent != tt.wantCCurrent {
				t.Errorf("status.CCurrent = %.2f, want %.2f", status.CCurrent, tt.wantCCurrent)
			}
			if status.CActivePower != tt.wantCActivePower {
				t.Errorf("status.CActivePower = %.2f, want %.2f", status.CActivePower, tt.wantCActivePower)
			}

			// Total
			if status.TotalActivePower != tt.wantTotalActivePower {
				t.Errorf("status.TotalActivePower = %.2f, want %.2f", status.TotalActivePower, tt.wantTotalActivePower)
			}

			// Neutral current
			if tt.wantNCurrent != nil {
				if status.NCurrent == nil {
					t.Error("status.NCurrent is nil, want non-nil")
				} else if *status.NCurrent != *tt.wantNCurrent {
					t.Errorf("status.NCurrent = %.2f, want %.2f", *status.NCurrent, *tt.wantNCurrent)
				}
			}

			// Errors
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

func TestEM_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em := NewEM(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := em.GetStatus(context.Background())
		return err
	})
}

func TestEM_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	em := NewEM(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := em.GetStatus(context.Background())
		return err
	})
}

func TestEM_GetCTTypes(t *testing.T) {
	result := `{"types": ["120A", "400A", "630A"]}`

	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "EM.GetCTTypes" {
				t.Errorf("method = %q, want %q", method, "EM.GetCTTypes")
			}
			return jsonrpcResponse(result)
		},
	}
	client := rpc.NewClient(tr)
	em := NewEM(client, 0)

	ctTypes, err := em.GetCTTypes(context.Background())
	if err != nil {
		t.Fatalf("GetCTTypes() error = %v", err)
	}

	if ctTypes == nil {
		t.Fatal("GetCTTypes() returned nil")
	}

	wantTypes := []string{"120A", "400A", "630A"}
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

func TestEM_GetCTTypes_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em := NewEM(client, 0)
	testComponentError(t, "GetCTTypes", func() error {
		_, err := em.GetCTTypes(context.Background())
		return err
	})
}

func TestEM_GetCTTypes_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	em := NewEM(client, 0)
	testComponentInvalidJSON(t, "GetCTTypes", func() error {
		_, err := em.GetCTTypes(context.Background())
		return err
	})
}

func TestEM_ResetCounters(t *testing.T) {
	tests := []struct {
		name         string
		counterTypes []string
	}{
		{
			name:         "reset all counters",
			counterTypes: nil,
		},
		{
			name:         "reset specific counters",
			counterTypes: []string{"a_act_energy", "b_act_energy", "c_act_energy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "EM.ResetCounters" {
						t.Errorf("method = %q, want %q", method, "EM.ResetCounters")
					}
					return jsonrpcResponse(`{}`)
				},
			}
			client := rpc.NewClient(tr)
			em := NewEM(client, 0)

			err := em.ResetCounters(context.Background(), tt.counterTypes)
			if err != nil {
				t.Errorf("ResetCounters() error = %v", err)
			}
		})
	}
}

func TestEM_ResetCounters_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em := NewEM(client, 0)
	testComponentError(t, "ResetCounters", func() error {
		return em.ResetCounters(context.Background(), nil)
	})
}
