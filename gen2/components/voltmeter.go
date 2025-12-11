package components

import (
	"context"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Voltmeter represents a Shelly Gen2+ Voltmeter component.
//
// Voltmeter is used to measure voltage on devices with voltage sensing capabilities,
// such as the Shelly Plus UNI. It supports configurable reporting thresholds,
// range settings, and custom transformations via JavaScript expressions.
//
// The component provides:
//   - Real-time voltage measurements
//   - Configurable report thresholds
//   - Range selection for different measurement scenarios
//   - XVoltage transformations for custom unit conversions
//
// Webhook Events:
//   - voltmeter.change: Triggered when voltage delta exceeds report_thr
//   - voltmeter.measurement: Triggered on 60-second measurement intervals
//
// Example:
//
//	voltmeter := components.NewVoltmeter(device.Client(), 0)
//	status, err := voltmeter.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Voltage: %.3fV\n", status.Voltage)
//	}
type Voltmeter struct {
	*gen2.BaseComponent
}

// NewVoltmeter creates a new Voltmeter component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (typically 0 for single voltmeter devices)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	voltmeter := components.NewVoltmeter(device.Client(), 0)
func NewVoltmeter(client *rpc.Client, id int) *Voltmeter {
	return &Voltmeter{
		BaseComponent: gen2.NewBaseComponent(client, "voltmeter", id),
	}
}

// VoltmeterConfig represents the configuration of a Voltmeter component.
//
// Use SetConfig to update voltmeter parameters like reporting threshold,
// range selection, and custom transformations.
type VoltmeterConfig struct {
	Name      *string                  `json:"name,omitempty"`
	ReportThr *float64                 `json:"report_thr,omitempty"`
	Range     *int                     `json:"range,omitempty"`
	XVoltage  *VoltmeterXVoltageConfig `json:"xvoltage,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// VoltmeterXVoltageConfig represents transformation configuration for voltage values.
//
// This allows converting raw voltage readings to custom values using JavaScript
// expressions. Common use cases include:
//   - Converting voltage to other measurement units
//   - Applying sensor-specific calibration
//   - Scaling voltage for specific sensors (e.g., voltage dividers)
//
// Example expressions:
//   - "x * 10"      - Scale voltage by 10
//   - "x + 0.5"     - Add offset
//   - "(x - 0.5) * 100" - Convert voltage range to percentage
type VoltmeterXVoltageConfig struct {
	// Expr is a JavaScript expression to transform the voltage value.
	// Variable 'x' represents the status.voltage value.
	// Set to null/nil to disable transformation.
	// Example: "x*10", "(x-0.5)*100", "x+1"
	Expr *string `json:"expr,omitempty"`

	// Unit is the unit name for the transformed value (max 20 characters).
	// Displayed alongside the xvoltage value in the status.
	// Example: "m/s", "°C", "kW"
	Unit *string `json:"unit,omitempty"`
}

// VoltmeterStatus represents the current status of a Voltmeter component.
type VoltmeterStatus struct {
	XVoltage *float64 `json:"xvoltage,omitempty"`
	types.RawFields
	Errors  []string `json:"errors,omitempty"`
	ID      int      `json:"id"`
	Voltage float64  `json:"voltage"`
}

// GetConfig retrieves the Voltmeter configuration.
//
// Returns the current configuration including name, report threshold,
// range setting, and xvoltage transformation settings.
//
// Example:
//
//	config, err := voltmeter.GetConfig(ctx)
//	if err == nil {
//	    if config.ReportThr != nil {
//	        fmt.Printf("Report threshold: %.2fV\n", *config.ReportThr)
//	    }
//	}
func (v *Voltmeter) GetConfig(ctx context.Context) (*VoltmeterConfig, error) {
	return gen2.UnmarshalConfig[VoltmeterConfig](ctx, v.BaseComponent)
}

// SetConfig updates the Voltmeter configuration.
//
// Only non-nil fields in the config will be updated. To remove a setting,
// you may need to set it to an empty/zero value explicitly.
//
// Example:
//
//	// Set report threshold and add transformation
//	reportThr := 0.5
//	err := voltmeter.SetConfig(ctx, &VoltmeterConfig{
//	    ReportThr: &reportThr,
//	    XVoltage: &VoltmeterXVoltageConfig{
//	        Expr: ptr("x * 10"),
//	        Unit: ptr("mV"),
//	    },
//	})
//
// Example with name and range:
//
//	name := "Battery Voltage"
//	rangeVal := 0
//	err := voltmeter.SetConfig(ctx, &VoltmeterConfig{
//	    Name:  &name,
//	    Range: &rangeVal,
//	})
func (v *Voltmeter) SetConfig(ctx context.Context, config *VoltmeterConfig) error {
	return gen2.SetConfigWithID(ctx, v.BaseComponent, config)
}

// GetStatus retrieves the current Voltmeter status.
//
// Returns the measured voltage and, if configured, the transformed xvoltage value.
// Check the Errors field for any sensor reading issues.
//
// Example:
//
//	status, err := voltmeter.GetStatus(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Voltage: %.3fV\n", status.Voltage)
//	if status.XVoltage != nil {
//	    fmt.Printf("Transformed: %.3f\n", *status.XVoltage)
//	}
//	if len(status.Errors) > 0 {
//	    fmt.Printf("Errors: %v\n", status.Errors)
//	}
func (v *Voltmeter) GetStatus(ctx context.Context) (*VoltmeterStatus, error) {
	return gen2.UnmarshalStatus[VoltmeterStatus](ctx, v.BaseComponent)
}
