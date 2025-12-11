package components

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

func TestNewInput(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)
	input := NewInput(client, 0)

	if input == nil {
		t.Fatal("NewInput returned nil")
	}

	if input.BaseComponent == nil {
		t.Fatal("BaseComponent not initialized")
	}
}

func TestInput_GetConfig(t *testing.T) {
	tests := []struct {
		validateFunc func(*testing.T, *InputConfig)
		name         string
		response     string
		wantErr      bool
	}{
		{
			name: "digital switch input",
			response: `{
				"id": 0,
				"name": "Front Door",
				"type": "switch",
				"enable": true,
				"invert": false,
				"factory_reset": true
			}`,
			wantErr: false,
			validateFunc: func(t *testing.T, config *InputConfig) {
				if config.ID != 0 {
					t.Errorf("ID = %d, want 0", config.ID)
				}
				if config.Name == nil || *config.Name != "Front Door" {
					t.Errorf("Name = %v, want 'Front Door'", config.Name)
				}
				if config.Type != "switch" {
					t.Errorf("Type = %s, want 'switch'", config.Type)
				}
				if config.Enable == nil || !*config.Enable {
					t.Errorf("Enable = %v, want true", config.Enable)
				}
				if config.Invert == nil || *config.Invert {
					t.Errorf("Invert = %v, want false", config.Invert)
				}
				if config.FactoryReset == nil || !*config.FactoryReset {
					t.Errorf("FactoryReset = %v, want true", config.FactoryReset)
				}
			},
		},
		{
			name: "button input",
			response: `{
				"id": 1,
				"name": "Button 1",
				"type": "button",
				"enable": true,
				"invert": true,
				"factory_reset": false
			}`,
			wantErr: false,
			validateFunc: func(t *testing.T, config *InputConfig) {
				if config.Type != "button" {
					t.Errorf("Type = %s, want 'button'", config.Type)
				}
				if config.Invert == nil || !*config.Invert {
					t.Errorf("Invert = %v, want true", config.Invert)
				}
			},
		},
		{
			name: "analog input with xpercent",
			response: `{
				"id": 0,
				"name": "Analog 0",
				"type": "analog",
				"enable": true,
				"invert": false,
				"report_thr": 5.0,
				"range_map": [0, 100],
				"xpercent": {
					"expr": "x*2",
					"unit": "cm"
				}
			}`,
			wantErr: false,
			validateFunc: func(t *testing.T, config *InputConfig) {
				if config.Type != "analog" {
					t.Errorf("Type = %s, want 'analog'", config.Type)
				}
				if config.ReportThr == nil || *config.ReportThr != 5.0 {
					t.Errorf("ReportThr = %v, want 5.0", config.ReportThr)
				}
				if len(config.RangeMap) != 2 || config.RangeMap[0] != 0 || config.RangeMap[1] != 100 {
					t.Errorf("RangeMap = %v, want [0, 100]", config.RangeMap)
				}
				if config.XPercent == nil {
					t.Fatal("XPercent is nil")
				}
				if config.XPercent.Expr == nil || *config.XPercent.Expr != "x*2" {
					t.Errorf("XPercent.Expr = %v, want 'x*2'", config.XPercent.Expr)
				}
				if config.XPercent.Unit == nil || *config.XPercent.Unit != "cm" {
					t.Errorf("XPercent.Unit = %v, want 'cm'", config.XPercent.Unit)
				}
			},
		},
		{
			name: "count input with transformations",
			response: `{
				"id": 2,
				"name": "Counter",
				"type": "count",
				"enable": true,
				"count_rep_thr": 10,
				"freq_window": 1.0,
				"freq_rep_thr": 0.5,
				"xcounts": {
					"expr": "x/1000",
					"unit": "kWh"
				},
				"xfreq": {
					"expr": "x*60",
					"unit": "rpm"
				}
			}`,
			wantErr: false,
			validateFunc: func(t *testing.T, config *InputConfig) {
				if config.Type != "count" {
					t.Errorf("Type = %s, want 'count'", config.Type)
				}
				if config.CountRepThr == nil || *config.CountRepThr != 10 {
					t.Errorf("CountRepThr = %v, want 10", config.CountRepThr)
				}
				if config.FreqWindow == nil || *config.FreqWindow != 1.0 {
					t.Errorf("FreqWindow = %v, want 1.0", config.FreqWindow)
				}
				if config.FreqRepThr == nil || *config.FreqRepThr != 0.5 {
					t.Errorf("FreqRepThr = %v, want 0.5", config.FreqRepThr)
				}
				if config.XCounts == nil {
					t.Fatal("XCounts is nil")
				}
				if config.XCounts.Expr == nil || *config.XCounts.Expr != "x/1000" {
					t.Errorf("XCounts.Expr = %v, want 'x/1000'", config.XCounts.Expr)
				}
				if config.XFreq == nil {
					t.Fatal("XFreq is nil")
				}
				if config.XFreq.Expr == nil || *config.XFreq.Expr != "x*60" {
					t.Errorf("XFreq.Expr = %v, want 'x*60'", config.XFreq.Expr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					return jsonrpcResponse(tt.response)
				},
			}

			client := rpc.NewClient(tr)
			input := NewInput(client, 0)

			config, err := input.GetConfig(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validateFunc != nil {
				tt.validateFunc(t, config)
			}
		})
	}
}

func TestInput_SetConfig(t *testing.T) {
	tests := []struct {
		config       *InputConfig
		validateFunc func(*testing.T, string, any)
		name         string
	}{
		{
			name: "enable and invert",
			config: &InputConfig{
				Enable: ptr(true),
				Invert: ptr(true),
			},
			validateFunc: func(t *testing.T, method string, params any) {
				if method != "Input.SetConfig" {
					t.Errorf("method = %s, want Input.SetConfig", method)
				}
			},
		},
		{
			name: "set name and type",
			config: &InputConfig{
				Name: ptr("New Input"),
				Type: "button",
			},
			validateFunc: func(t *testing.T, method string, params any) {
				if method != "Input.SetConfig" {
					t.Errorf("method = %s, want Input.SetConfig", method)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedMethod string
			var capturedParams any

			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					capturedMethod = method
					capturedParams = params
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}

			client := rpc.NewClient(tr)
			input := NewInput(client, 0)

			err := input.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Errorf("SetConfig() error = %v", err)
				return
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, capturedMethod, capturedParams)
			}
		})
	}
}

func TestInput_GetStatus(t *testing.T) {
	tests := []struct {
		validateFunc func(*testing.T, *InputStatus)
		name         string
		response     string
		wantErr      bool
	}{
		{
			name: "switch input on",
			response: `{
				"id": 0,
				"state": true
			}`,
			wantErr: false,
			validateFunc: func(t *testing.T, status *InputStatus) {
				if status.ID != 0 {
					t.Errorf("ID = %d, want 0", status.ID)
				}
				if status.State == nil || !*status.State {
					t.Errorf("State = %v, want true", status.State)
				}
			},
		},
		{
			name: "analog input with percent",
			response: `{
				"id": 0,
				"state": null,
				"percent": 75.5,
				"xpercent": 151.0
			}`,
			wantErr: false,
			validateFunc: func(t *testing.T, status *InputStatus) {
				if status.Percent == nil || *status.Percent != 75.5 {
					t.Errorf("Percent = %v, want 75.5", status.Percent)
				}
				if status.XPercent == nil || *status.XPercent != 151.0 {
					t.Errorf("XPercent = %v, want 151.0", status.XPercent)
				}
			},
		},
		{
			name: "count input with frequency",
			response: `{
				"id": 2,
				"state": null,
				"percent": null,
				"counts": {
					"total": 12345,
					"by_minute": [100, 95, 102],
					"minute_ts": 1640000000,
					"xtotal": 12.345
				},
				"freq": 1.5,
				"xfreq": 90.0
			}`,
			wantErr: false,
			validateFunc: func(t *testing.T, status *InputStatus) {
				if status.Counts == nil {
					t.Fatal("Counts is nil")
				}
				if status.Counts.Total != 12345 {
					t.Errorf("Counts.Total = %d, want 12345", status.Counts.Total)
				}
				if len(status.Counts.ByMinute) != 3 {
					t.Errorf("Counts.ByMinute length = %d, want 3", len(status.Counts.ByMinute))
				}
				if status.Freq == nil || *status.Freq != 1.5 {
					t.Errorf("Freq = %v, want 1.5", status.Freq)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					return jsonrpcResponse(tt.response)
				},
			}

			client := rpc.NewClient(tr)
			input := NewInput(client, 0)

			status, err := input.GetStatus(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validateFunc != nil {
				tt.validateFunc(t, status)
			}
		})
	}
}

func TestInput_CheckExpression(t *testing.T) {
	tests := []struct {
		validateFunc func(*testing.T, *InputCheckExpressionResult)
		name         string
		expr         string
		response     string
		inputs       []any
		wantErr      bool
	}{
		{
			name:   "simple multiplication",
			expr:   "x*2",
			inputs: []any{10, 20, 30},
			response: `{
				"results": [
					["10", 20],
					["20", 40],
					["30", 60]
				]
			}`,
			wantErr: false,
			validateFunc: func(t *testing.T, result *InputCheckExpressionResult) {
				if len(result.Results) != 3 {
					t.Errorf("Results length = %d, want 3", len(result.Results))
					return
				}
				if len(result.Results[0]) != 2 {
					t.Errorf("Results[0] length = %d, want 2", len(result.Results[0]))
					return
				}
				if result.Results[0][0] != "10" {
					t.Errorf("Results[0][0] = %v, want '10'", result.Results[0][0])
				}
			},
		},
		{
			name:   "expression with null input",
			expr:   "x+1",
			inputs: []any{5, nil, 10},
			response: `{
				"results": [
					["5", 6],
					[null, null],
					["10", 11]
				]
			}`,
			wantErr: false,
			validateFunc: func(t *testing.T, result *InputCheckExpressionResult) {
				if len(result.Results) != 3 {
					t.Fatalf("Results length = %d, want 3", len(result.Results))
				}
				if result.Results[1][0] != nil {
					t.Errorf("Results[1][0] = %v, want nil", result.Results[1][0])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					return jsonrpcResponse(tt.response)
				},
			}

			client := rpc.NewClient(tr)
			input := NewInput(client, 0)

			result, err := input.CheckExpression(context.Background(), tt.expr, tt.inputs)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestInput_ResetCounters(t *testing.T) {
	tests := []struct {
		validateFunc func(*testing.T, *InputResetCountersResult)
		name         string
		response     string
		resetTypes   []string
		wantErr      bool
	}{
		{
			name:       "reset all counters",
			resetTypes: nil,
			response: `{
				"counts": {
					"total": 54321
				}
			}`,
			wantErr: false,
			validateFunc: func(t *testing.T, result *InputResetCountersResult) {
				if result.Counts == nil {
					t.Fatal("Counts is nil")
				}
				if result.Counts.Total != 54321 {
					t.Errorf("Counts.Total = %d, want 54321", result.Counts.Total)
				}
			},
		},
		{
			name:       "reset specific counter type",
			resetTypes: []string{"total"},
			response: `{
				"counts": {
					"total": 99999
				}
			}`,
			wantErr: false,
			validateFunc: func(t *testing.T, result *InputResetCountersResult) {
				if result.Counts == nil {
					t.Fatal("Counts is nil")
				}
				if result.Counts.Total != 99999 {
					t.Errorf("Counts.Total = %d, want 99999", result.Counts.Total)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					return jsonrpcResponse(tt.response)
				},
			}

			client := rpc.NewClient(tr)
			input := NewInput(client, 0)

			result, err := input.ResetCounters(context.Background(), tt.resetTypes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResetCounters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestInput_Trigger(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		wantErr   bool
	}{
		{
			name:      "btn_down event",
			eventType: "btn_down",
			wantErr:   false,
		},
		{
			name:      "single_push event",
			eventType: "single_push",
			wantErr:   false,
		},
		{
			name:      "long_push event",
			eventType: "long_push",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					return jsonrpcResponse(`null`)
				},
			}

			client := rpc.NewClient(tr)
			input := NewInput(client, 0)

			err := input.Trigger(context.Background(), tt.eventType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Trigger() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
