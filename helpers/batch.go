package helpers

import (
	"context"
	"sync"

	"github.com/tj-smith47/shelly-go/factory"
	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/types"
)

// BatchResult contains the result of a batch operation on a single device.
type BatchResult struct {
	Device  factory.Device
	Error   error
	Success bool
}

// BatchResults is a collection of batch operation results.
type BatchResults []BatchResult

// AllSuccessful returns true if all operations succeeded.
func (r BatchResults) AllSuccessful() bool {
	for _, res := range r {
		if !res.Success {
			return false
		}
	}
	return true
}

// Failures returns only the failed results.
func (r BatchResults) Failures() BatchResults {
	var failures BatchResults
	for _, res := range r {
		if !res.Success {
			failures = append(failures, res)
		}
	}
	return failures
}

// Successes returns only the successful results.
func (r BatchResults) Successes() BatchResults {
	var successes BatchResults
	for _, res := range r {
		if res.Success {
			successes = append(successes, res)
		}
	}
	return successes
}

// deviceOperation is a function type for operations on a single device.
type deviceOperation func(ctx context.Context, dev factory.Device) error

// batchExecute runs an operation on multiple devices concurrently.
func batchExecute(ctx context.Context, devices []factory.Device, op deviceOperation) BatchResults {
	results := make(BatchResults, len(devices))
	var wg sync.WaitGroup

	for i, device := range devices {
		wg.Add(1)
		go func(index int, dev factory.Device) {
			defer wg.Done()

			result := BatchResult{Device: dev}
			if err := op(ctx, dev); err != nil {
				result.Error = err
			} else {
				result.Success = true
			}
			results[index] = result
		}(i, device)
	}

	wg.Wait()
	return results
}

// BatchSet sets the on/off state for multiple switch devices concurrently.
func BatchSet(ctx context.Context, devices []factory.Device, on bool) BatchResults {
	return batchExecute(ctx, devices, func(ctx context.Context, dev factory.Device) error {
		return setSwitchState(ctx, dev, on)
	})
}

// BatchToggle toggles the state of multiple switch devices concurrently.
func BatchToggle(ctx context.Context, devices []factory.Device) BatchResults {
	return batchExecute(ctx, devices, toggleSwitch)
}

// AllOff turns off all devices in the slice.
func AllOff(ctx context.Context, devices []factory.Device) BatchResults {
	return BatchSet(ctx, devices, false)
}

// AllOn turns on all devices in the slice.
func AllOn(ctx context.Context, devices []factory.Device) BatchResults {
	return BatchSet(ctx, devices, true)
}

// BatchSetBrightness sets the brightness for multiple light devices.
func BatchSetBrightness(ctx context.Context, devices []factory.Device, brightness int) BatchResults {
	return batchExecute(ctx, devices, func(ctx context.Context, dev factory.Device) error {
		return setBrightness(ctx, dev, brightness)
	})
}

// setSwitchState sets the switch state for a device.
func setSwitchState(ctx context.Context, dev factory.Device, on bool) error {
	switch d := dev.(type) {
	case *factory.Gen1Device:
		return setGen1SwitchState(ctx, d, on)
	case *factory.Gen2Device:
		return setGen2SwitchState(ctx, d, on)
	default:
		return types.ErrUnsupportedDevice
	}
}

// setGen1SwitchState sets the switch state for a Gen1 device.
func setGen1SwitchState(ctx context.Context, dev *factory.Gen1Device, on bool) error {
	if dev.Device == nil {
		return types.ErrNilDevice
	}
	relay := dev.Relay(0)
	if on {
		return relay.TurnOn(ctx)
	}
	return relay.TurnOff(ctx)
}

// setGen2SwitchState sets the switch state for a Gen2 device.
func setGen2SwitchState(ctx context.Context, dev *factory.Gen2Device, on bool) error {
	if dev.Device == nil {
		return types.ErrNilDevice
	}
	sw, ok := dev.Switch(0).(*gen2.BaseComponent)
	if !ok {
		return types.ErrUnsupportedDevice
	}
	params := map[string]any{"id": sw.ID(), "on": on}
	_, err := sw.Client().Call(ctx, "Switch.Set", params)
	return err
}

// toggleSwitch toggles the switch state for a device.
func toggleSwitch(ctx context.Context, dev factory.Device) error {
	switch d := dev.(type) {
	case *factory.Gen1Device:
		return toggleGen1Switch(ctx, d)
	case *factory.Gen2Device:
		return toggleGen2Switch(ctx, d)
	default:
		return types.ErrUnsupportedDevice
	}
}

// toggleGen1Switch toggles a Gen1 device switch.
func toggleGen1Switch(ctx context.Context, dev *factory.Gen1Device) error {
	if dev.Device == nil {
		return types.ErrNilDevice
	}
	relay := dev.Relay(0)
	return relay.Toggle(ctx)
}

// toggleGen2Switch toggles a Gen2 device switch.
func toggleGen2Switch(ctx context.Context, dev *factory.Gen2Device) error {
	if dev.Device == nil {
		return types.ErrNilDevice
	}
	sw, ok := dev.Switch(0).(*gen2.BaseComponent)
	if !ok {
		return types.ErrUnsupportedDevice
	}
	params := map[string]any{"id": sw.ID()}
	_, err := sw.Client().Call(ctx, "Switch.Toggle", params)
	return err
}

// setBrightness sets the brightness for a light device.
func setBrightness(ctx context.Context, dev factory.Device, brightness int) error {
	switch d := dev.(type) {
	case *factory.Gen1Device:
		return setGen1Brightness(ctx, d, brightness)
	case *factory.Gen2Device:
		return setGen2Brightness(ctx, d, brightness)
	default:
		return types.ErrUnsupportedDevice
	}
}

// setGen1Brightness sets brightness on a Gen1 device.
func setGen1Brightness(ctx context.Context, dev *factory.Gen1Device, brightness int) error {
	if dev.Device == nil {
		return types.ErrNilDevice
	}
	light := dev.Light(0)
	return light.SetBrightness(ctx, brightness)
}

// setGen2Brightness sets brightness on a Gen2 device.
func setGen2Brightness(ctx context.Context, dev *factory.Gen2Device, brightness int) error {
	if dev.Device == nil {
		return types.ErrNilDevice
	}
	light, ok := dev.Light(0).(*gen2.BaseComponent)
	if !ok {
		return types.ErrUnsupportedDevice
	}
	params := map[string]any{"id": light.ID(), "brightness": brightness}
	_, err := light.Client().Call(ctx, "Light.Set", params)
	return err
}
