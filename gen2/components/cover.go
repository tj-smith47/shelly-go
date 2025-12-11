package components

import (
	"context"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Cover represents a Shelly Gen2+ Cover component.
//
// Cover components control roller shutters, blinds, and similar motorized covers.
// They support position control, calibration, and obstruction detection.
//
// Example:
//
//	cover := components.NewCover(device.Client(), 0)
//	err := cover.Open(ctx)
type Cover struct {
	*gen2.BaseComponent
}

// NewCover creates a new Cover component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (usually 0 for single-cover devices)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	cover := components.NewCover(device.Client(), 0)
func NewCover(client *rpc.Client, id int) *Cover {
	return &Cover{
		BaseComponent: gen2.NewBaseComponent(client, "cover", id),
	}
}

// CoverConfig represents the configuration of a Cover component.
type CoverConfig struct {
	InvertDirections          *bool    `json:"invert_directions,omitempty"`
	Name                      *string  `json:"name,omitempty"`
	MotorIdleConfirmTimeout   *float64 `json:"motor_idle_confirm_timeout,omitempty"`
	MotorMoveTimeout          *float64 `json:"motor_move_timeout,omitempty"`
	ObstructionDetectionLevel *int     `json:"obstruction_detection_level,omitempty"`
	SwapInputs                *bool    `json:"swap_inputs,omitempty"`
	InitialState              *string  `json:"initial_state,omitempty"`
	PowerLimit                *float64 `json:"power_limit,omitempty"`
	VoltageLimit              *float64 `json:"voltage_limit,omitempty"`
	UndervoltageLimit         *float64 `json:"undervoltage_limit,omitempty"`
	CurrentLimit              *float64 `json:"current_limit,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// CoverStatus represents the current status of a Cover component.
type CoverStatus struct {
	AEnergy   *EnergyCounters `json:"aenergy,omitempty"`
	TargetPos *int            `json:"target_pos,omitempty"`
	types.RawFields
	APower        *float64           `json:"apower,omitempty"`
	Voltage       *float64           `json:"voltage,omitempty"`
	Current       *float64           `json:"current,omitempty"`
	PF            *float64           `json:"pf,omitempty"`
	Freq          *float64           `json:"freq,omitempty"`
	LastDirection *string            `json:"last_direction,omitempty"`
	Temperature   *TemperatureSensor `json:"temperature,omitempty"`
	CurrentPos    *int               `json:"current_pos,omitempty"`
	MoveTimeout   *bool              `json:"move_timeout,omitempty"`
	MoveStartedAt *float64           `json:"move_started_at,omitempty"`
	PosDelta      *int               `json:"pos_delta,omitempty"`
	Source        string             `json:"source"`
	State         string             `json:"state"`
	Errors        []string           `json:"errors,omitempty"`
	ID            int                `json:"id"`
}

// CoverOpenParams contains parameters for the Cover.Open method.
type CoverOpenParams struct {
	Duration *float64 `json:"duration,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// CoverCloseParams contains parameters for the Cover.Close method.
type CoverCloseParams struct {
	Duration *float64 `json:"duration,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// CoverStopParams contains parameters for the Cover.Stop method.
type CoverStopParams struct {
	types.RawFields
	ID int `json:"id"`
}

// CoverGoToPositionParams contains parameters for the Cover.GoToPosition method.
type CoverGoToPositionParams struct {
	types.RawFields
	ID  int `json:"id"`
	Pos int `json:"pos"`
}

// CoverCalibrateParams contains parameters for the Cover.Calibrate method.
type CoverCalibrateParams struct {
	types.RawFields
	ID int `json:"id"`
}

// Open opens the cover (moves to fully open position).
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - duration: Optional time limit in seconds
//
// Example:
//
//	// Open fully
//	err := cover.Open(ctx, nil)
//
//	// Open for 5 seconds
//	err := cover.Open(ctx, ptr(5.0))
func (c *Cover) Open(ctx context.Context, duration *float64) error {
	params := &CoverOpenParams{
		ID:       c.ID(),
		Duration: duration,
	}

	_, err := c.Client().Call(ctx, "Cover.Open", params)
	return err
}

// Close closes the cover (moves to fully closed position).
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - duration: Optional time limit in seconds
//
// Example:
//
//	// Close fully
//	err := cover.Close(ctx, nil)
//
//	// Close for 5 seconds
//	err := cover.Close(ctx, ptr(5.0))
func (c *Cover) Close(ctx context.Context, duration *float64) error {
	params := &CoverCloseParams{
		ID:       c.ID(),
		Duration: duration,
	}

	_, err := c.Client().Call(ctx, "Cover.Close", params)
	return err
}

// Stop stops the cover movement.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Example:
//
//	err := cover.Stop(ctx)
func (c *Cover) Stop(ctx context.Context) error {
	params := &CoverStopParams{
		ID: c.ID(),
	}

	_, err := c.Client().Call(ctx, "Cover.Stop", params)
	return err
}

// GoToPosition moves the cover to a specific position.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - pos: Target position in percent (0 = fully closed, 100 = fully open)
//
// Example:
//
//	// Move to 50% open
//	err := cover.GoToPosition(ctx, 50)
//
//	// Move to fully closed
//	err := cover.GoToPosition(ctx, 0)
func (c *Cover) GoToPosition(ctx context.Context, pos int) error {
	params := &CoverGoToPositionParams{
		ID:  c.ID(),
		Pos: pos,
	}

	_, err := c.Client().Call(ctx, "Cover.GoToPosition", params)
	return err
}

// Calibrate starts the calibration procedure.
//
// During calibration, the cover moves to fully closed and fully open positions
// to learn the motor timings and position limits.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Example:
//
//	err := cover.Calibrate(ctx)
func (c *Cover) Calibrate(ctx context.Context) error {
	params := &CoverCalibrateParams{
		ID: c.ID(),
	}

	_, err := c.Client().Call(ctx, "Cover.Calibrate", params)
	return err
}

// ResetCounters resets the energy counters.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - types: Optional list of counter types to reset (e.g., ["aenergy"])
//
// Example:
//
//	err := cover.ResetCounters(ctx, []string{"aenergy"})
func (c *Cover) ResetCounters(ctx context.Context, counterTypes []string) error {
	params := map[string]any{
		"id": c.ID(),
	}

	if len(counterTypes) > 0 {
		params["type"] = counterTypes
	}

	_, err := c.Client().Call(ctx, "Cover.ResetCounters", params)
	return err
}

// GetConfig retrieves the cover configuration.
//
// Example:
//
//	config, err := cover.GetConfig(ctx)
func (c *Cover) GetConfig(ctx context.Context) (*CoverConfig, error) {
	return gen2.UnmarshalConfig[CoverConfig](ctx, c.BaseComponent)
}

// SetConfig updates the cover configuration.
//
// Example:
//
//	err := cover.SetConfig(ctx, &CoverConfig{
//	    Name: ptr("Living Room Blinds"),
//	    ObstructionDetectionLevel: ptr(75),
//	})
func (c *Cover) SetConfig(ctx context.Context, config *CoverConfig) error {
	return gen2.SetConfigWithID(ctx, c.BaseComponent, config)
}

// GetStatus retrieves the current cover status.
//
// Example:
//
//	status, err := cover.GetStatus(ctx)
//	fmt.Printf("State: %s, Position: %d%%\n", status.State, *status.CurrentPos)
func (c *Cover) GetStatus(ctx context.Context) (*CoverStatus, error) {
	return gen2.UnmarshalStatus[CoverStatus](ctx, c.BaseComponent)
}
