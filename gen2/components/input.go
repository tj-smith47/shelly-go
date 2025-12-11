package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Input represents a Shelly Gen2+ Input component.
//
// Input components handle external digital or analog input terminals. They can:
//   - Sense discrete HIGH/LOW input states (digital mode)
//   - Read analog values as a percentage range 0-100% (analog mode)
//   - Count accumulated pulses and measure frequency (counter mode)
//   - Trigger webhooks and control switches
//   - Perform factory reset when toggled 5 times in first 60 seconds
//
// Input types:
//   - "switch": Discrete HIGH/LOW states with toggle switches
//   - "button": Discrete states with momentary buttons
//   - "analog": Percentage-based analog readings (0-100%)
//   - "count": Accumulated counts and frequency measurements
//
// Example:
//
//	input := components.NewInput(device.Client(), 0)
//	status, err := input.GetStatus(ctx)
//	if err == nil && status.State != nil {
//	    fmt.Printf("Input state: %t\n", *status.State)
//	}
type Input struct {
	*gen2.BaseComponent
}

// NewInput creates a new Input component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (0-based, e.g., 0 for first input)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	input := components.NewInput(device.Client(), 0)
func NewInput(client *rpc.Client, id int) *Input {
	return &Input{
		BaseComponent: gen2.NewBaseComponent(client, "input", id),
	}
}

// InputConfig represents the configuration of an Input component.
type InputConfig struct {
	ReportThr  *float64 `json:"report_thr,omitempty"`
	FreqWindow *float64 `json:"freq_window,omitempty"`
	types.RawFields
	Enable       *bool                `json:"enable,omitempty"`
	Invert       *bool                `json:"invert,omitempty"`
	FactoryReset *bool                `json:"factory_reset,omitempty"`
	Name         *string              `json:"name,omitempty"`
	XPercent     *InputXPercentConfig `json:"xpercent,omitempty"`
	XFreq        *InputXFreqConfig    `json:"xfreq,omitempty"`
	CountRepThr  *int                 `json:"count_rep_thr,omitempty"`
	XCounts      *InputXCountsConfig  `json:"xcounts,omitempty"`
	FreqRepThr   *float64             `json:"freq_rep_thr,omitempty"`
	Type         string               `json:"type"`
	RangeMap     []float64            `json:"range_map,omitempty"`
	ID           int                  `json:"id"`
}

// InputXPercentConfig represents transformation configuration for analog percent values.
type InputXPercentConfig struct {
	// Expr is a JavaScript expression to transform the value
	// Variable 'x' represents status.percent value
	// Example: "x+1", "x*2", null to disable
	Expr *string `json:"expr,omitempty"`

	// Unit is the unit name for transformed value (max 20 chars)
	Unit *string `json:"unit,omitempty"`
}

// InputXCountsConfig represents transformation configuration for count values.
type InputXCountsConfig struct {
	// Expr is a JavaScript expression to transform count values
	// Variable 'x' represents status.counts.total or status.counts.by_minute
	Expr *string `json:"expr,omitempty"`

	// Unit is the unit name for transformed values (max 20 chars)
	Unit *string `json:"unit,omitempty"`
}

// InputXFreqConfig represents transformation configuration for frequency values.
type InputXFreqConfig struct {
	// Expr is a JavaScript expression to transform frequency
	// Variable 'x' represents status.freq value
	Expr *string `json:"expr,omitempty"`

	// Unit is the unit name for transformed value (max 20 chars)
	Unit *string `json:"unit,omitempty"`
}

// InputStatus represents the current status of an Input component.
type InputStatus struct {
	State    *bool        `json:"state"`
	Percent  *float64     `json:"percent"`
	XPercent *float64     `json:"xpercent"`
	Counts   *InputCounts `json:"counts,omitempty"`
	Freq     *float64     `json:"freq"`
	XFreq    *float64     `json:"xfreq"`
	types.RawFields
	Errors []string `json:"errors,omitempty"`
	ID     int      `json:"id"`
}

// InputCounts represents counter values for count-type inputs.
type InputCounts struct {
	MinuteTS  *int64    `json:"minute_ts,omitempty"`
	XTotal    *float64  `json:"xtotal,omitempty"`
	ByMinute  []int     `json:"by_minute,omitempty"`
	XByMinute []float64 `json:"xby_minute,omitempty"`
	Total     int       `json:"total"`
}

// CheckExpression validates a JavaScript expression against test input values.
//
// This method evaluates a JavaScript expression with up to 5 test input values
// and returns the results. Useful for testing xpercent, xcounts, or xfreq expressions
// before applying them to configuration.
//
// Parameters:
//   - expr: JavaScript expression to evaluate (variable 'x' represents input)
//   - inputs: Up to 5 test input values (can include nil values)
//
// Returns array of [input, output] pairs. On error, returns [input, output, error_msg].
//
// Example:
//
//	results, err := input.CheckExpression(ctx, "x*2", []any{10, 20, 30})
//	// results = [["10", "20"], ["20", "40"], ["30", "60"]]
func (i *Input) CheckExpression(ctx context.Context, expr string, inputs []any) (*InputCheckExpressionResult, error) {
	params := map[string]any{
		"expr":   expr,
		"inputs": inputs,
	}

	resultJSON, err := i.BaseComponent.Client().Call(ctx, "Input.CheckExpression", params)
	if err != nil {
		return nil, err
	}

	var result InputCheckExpressionResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// InputCheckExpressionResult represents the result of CheckExpression.
type InputCheckExpressionResult struct {
	types.RawFields
	Results [][]any `json:"results"`
}

// ResetCounters resets the input's counters (for count type inputs).
//
// Parameters:
//   - resetTypes: Optional array of counter types to reset (e.g., ["total"])
//     If nil or empty, resets all available counters
//
// Returns the counter values prior to reset.
//
// Example:
//
//	result, err := input.ResetCounters(ctx, []string{"total"})
func (i *Input) ResetCounters(ctx context.Context, resetTypes []string) (*InputResetCountersResult, error) {
	params := map[string]any{
		"id": i.ID(),
	}

	if len(resetTypes) > 0 {
		params["type"] = resetTypes
	}

	resultJSON, err := i.BaseComponent.Client().Call(ctx, "Input.ResetCounters", params)
	if err != nil {
		return nil, err
	}

	var result InputResetCountersResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// InputResetCountersResult represents the result of ResetCounters.
type InputResetCountersResult struct {
	// Counts contains the counter values before reset
	Counts *InputResetCounts `json:"counts,omitempty"`

	// RawFields captures any additional fields
	types.RawFields
}

// InputResetCounts represents counter values before reset.
type InputResetCounts struct {
	// Total count value before reset
	Total int `json:"total"`
}

// Trigger emits input events on demand without physical input changes.
//
// Only available for type="button" on Shelly Plus I4, Plus I4 DC, I4 Gen3, and I4 DC Gen3.
//
// Parameters:
//   - eventType: Event to emit - one of:
//     "btn_down", "btn_up", "single_push", "double_push", "triple_push", "long_push"
//
// Example:
//
//	err := input.Trigger(ctx, "single_push")
func (i *Input) Trigger(ctx context.Context, eventType string) error {
	params := map[string]any{
		"id":         i.ID(),
		"event_type": eventType,
	}

	_, err := i.BaseComponent.Client().Call(ctx, "Input.Trigger", params)
	return err
}

// GetConfig retrieves the component's configuration.
func (i *Input) GetConfig(ctx context.Context) (*InputConfig, error) {
	return gen2.UnmarshalConfig[InputConfig](ctx, i.BaseComponent)
}

// SetConfig updates the component's configuration.
//
// Only non-nil fields in the config will be updated.
//
// Example:
//
//	enable := true
//	invertVal := true
//	err := input.SetConfig(ctx, &InputConfig{
//	    Enable: &enable,
//	    Invert: &invertVal,
//	})
func (i *Input) SetConfig(ctx context.Context, config *InputConfig) error {
	return gen2.SetConfigWithID(ctx, i.BaseComponent, config)
}

// GetStatus retrieves the component's current status.
func (i *Input) GetStatus(ctx context.Context) (*InputStatus, error) {
	return gen2.UnmarshalStatus[InputStatus](ctx, i.BaseComponent)
}
