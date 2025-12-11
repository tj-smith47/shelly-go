package components

import (
	"context"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// DevicePower represents a Shelly Gen2+ DevicePower component.
//
// DevicePower components provide battery status monitoring for battery-powered devices.
// They report battery voltage, charge percentage, and external power source status.
//
// This component is typically found on devices like:
//   - Shelly Plus H&T (temperature/humidity sensor)
//   - Shelly Plus Smoke (smoke detector)
//   - Shelly BLU devices (Bluetooth sensors)
//
// Example:
//
//	devicePower := components.NewDevicePower(device.Client(), 0)
//	status, err := devicePower.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Battery: %.2fV (%d%%)\n", status.Battery.V, status.Battery.Percent)
//	    fmt.Printf("External power: %v\n", status.External.Present)
//	}
type DevicePower struct {
	*gen2.BaseComponent
}

// NewDevicePower creates a new DevicePower component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (usually 0 for single-battery devices)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	devicePower := components.NewDevicePower(device.Client(), 0)
func NewDevicePower(client *rpc.Client, id int) *DevicePower {
	return &DevicePower{
		BaseComponent: gen2.NewBaseComponent(client, "devicepower", id),
	}
}

// DevicePowerConfig represents the configuration of a DevicePower component.
//
// Note: The DevicePower component does not own any configuration properties
// according to the official Shelly API documentation. This struct is provided
// for API consistency and future compatibility.
type DevicePowerConfig struct {
	types.RawFields
	ID int `json:"id"`
}

// DevicePowerStatus represents the current status of a DevicePower component.
type DevicePowerStatus struct {
	types.RawFields
	External ExternalPowerStatus `json:"external"`
	Battery  BatteryStatus       `json:"battery"`
	ID       int                 `json:"id"`
}

// BatteryStatus represents battery status information.
type BatteryStatus struct {
	types.RawFields
	V       float64 `json:"V"`
	Percent int     `json:"percent"`
}

// ExternalPowerStatus represents external power source status.
type ExternalPowerStatus struct {
	types.RawFields
	Present bool `json:"present"`
}

// GetConfig retrieves the devicepower configuration.
//
// Note: The DevicePower component does not own any configuration properties.
// This method is provided for API consistency.
//
// Example:
//
//	config, err := devicePower.GetConfig(ctx)
func (d *DevicePower) GetConfig(ctx context.Context) (*DevicePowerConfig, error) {
	return gen2.UnmarshalConfig[DevicePowerConfig](ctx, d.BaseComponent)
}

// SetConfig updates the devicepower configuration.
//
// Note: The DevicePower component does not own any configuration properties.
// This method is provided for API consistency and will typically have no effect.
//
// Example:
//
//	err := devicePower.SetConfig(ctx, &DevicePowerConfig{})
func (d *DevicePower) SetConfig(ctx context.Context, config *DevicePowerConfig) error {
	return gen2.SetConfigWithID(ctx, d.BaseComponent, config)
}

// GetStatus retrieves the current devicepower status.
//
// Returns battery voltage, charge percentage, and external power status.
//
// Example:
//
//	status, err := devicePower.GetStatus(ctx)
//	if err != nil {
//	    return err
//	}
//
//	if status.Battery.Percent < 20 {
//	    fmt.Println("Low battery warning!")
//	}
//
//	if status.External.Present {
//	    fmt.Println("Charging from external power")
//	}
func (d *DevicePower) GetStatus(ctx context.Context) (*DevicePowerStatus, error) {
	return gen2.UnmarshalStatus[DevicePowerStatus](ctx, d.BaseComponent)
}
