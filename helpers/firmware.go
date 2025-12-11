package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tj-smith47/shelly-go/factory"
	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/types"
)

// FirmwareInfo contains information about a device's firmware.
type FirmwareInfo struct {
	Device           factory.Device `json:"-"`
	CurrentVersion   string         `json:"current_version"`
	AvailableVersion string         `json:"available_version,omitempty"`
	Address          string         `json:"address"`
	HasUpdate        bool           `json:"has_update"`
}

// FirmwareResult contains the result of a firmware operation.
type FirmwareResult struct {
	Device  factory.Device
	Error   error
	Info    *FirmwareInfo
	Success bool
}

// FirmwareResults is a collection of firmware operation results.
type FirmwareResults []FirmwareResult

// AllSuccessful returns true if all operations succeeded.
func (r FirmwareResults) AllSuccessful() bool {
	for _, res := range r {
		if !res.Success {
			return false
		}
	}
	return true
}

// Failures returns only the failed results.
func (r FirmwareResults) Failures() FirmwareResults {
	var failures FirmwareResults
	for _, res := range r {
		if !res.Success {
			failures = append(failures, res)
		}
	}
	return failures
}

// UpdatesAvailable returns devices that have firmware updates available.
func (r FirmwareResults) UpdatesAvailable() FirmwareResults {
	var updates FirmwareResults
	for _, res := range r {
		if res.Success && res.Info != nil && res.Info.HasUpdate {
			updates = append(updates, res)
		}
	}
	return updates
}

// GetFirmwareInfo retrieves firmware information for a single device.
func GetFirmwareInfo(ctx context.Context, dev factory.Device) (*FirmwareInfo, error) {
	switch d := dev.(type) {
	case *factory.Gen1Device:
		return getGen1FirmwareInfo(ctx, d)
	case *factory.Gen2Device:
		return getGen2FirmwareInfo(ctx, d)
	default:
		return nil, types.ErrUnsupportedDevice
	}
}

// getGen1FirmwareInfo retrieves firmware info for a Gen1 device.
func getGen1FirmwareInfo(ctx context.Context, dev *factory.Gen1Device) (*FirmwareInfo, error) {
	if dev.Device == nil {
		return nil, types.ErrNilDevice
	}

	// Get device info first for current version
	info, err := dev.GetDeviceInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get device info: %w", err,
		)
	}

	fwInfo := &FirmwareInfo{
		CurrentVersion: info.Version,
		Address:        dev.Address(),
		Device:         dev,
	}

	// Check for updates
	updateInfo, err := dev.CheckForUpdate(ctx)
	if err == nil && updateInfo != nil && updateInfo.HasUpdate {
		fwInfo.HasUpdate = true
		fwInfo.AvailableVersion = updateInfo.NewVersion
	}

	return fwInfo, nil
}

// getGen2FirmwareInfo retrieves firmware info for a Gen2 device.
func getGen2FirmwareInfo(ctx context.Context, dev *factory.Gen2Device) (*FirmwareInfo, error) {
	if dev.Device == nil {
		return nil, types.ErrNilDevice
	}

	// Get device info for current version
	devInfo, err := dev.GetDeviceInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get device info: %w", err,
		)
	}

	fwInfo := &FirmwareInfo{
		CurrentVersion: devInfo.FirmwareVersion,
		Address:        dev.Address(),
		Device:         dev,
	}

	// Check for updates using Shelly.CheckForUpdate
	sysComp, ok := dev.Sys(0).(*gen2.BaseComponent)
	if !ok {
		return nil, types.ErrUnsupportedDevice
	}
	result, err := sysComp.Client().Call(ctx, "Shelly.CheckForUpdate", nil)
	if err == nil && result != nil {
		var updateResp struct {
			Stable *struct {
				Version string `json:"version"`
			} `json:"stable"`
		}
		if err := json.Unmarshal(result, &updateResp); err == nil {
			if updateResp.Stable != nil && updateResp.Stable.Version != "" &&
				updateResp.Stable.Version != devInfo.FirmwareVersion {
				fwInfo.HasUpdate = true
				fwInfo.AvailableVersion = updateResp.Stable.Version
			}
		}
	}

	return fwInfo, nil
}

// CheckFirmwareUpdates checks for firmware updates on multiple devices concurrently.
func CheckFirmwareUpdates(ctx context.Context, devices []factory.Device) FirmwareResults {
	results := make(FirmwareResults, len(devices))
	var wg sync.WaitGroup

	for i, device := range devices {
		wg.Add(1)
		go func(index int, dev factory.Device) {
			defer wg.Done()

			result := FirmwareResult{Device: dev}
			info, err := GetFirmwareInfo(ctx, dev)
			if err != nil {
				result.Error = err
			} else {
				result.Info = info
				result.Success = true
			}
			results[index] = result
		}(i, device)
	}

	wg.Wait()
	return results
}

// UpdateFirmware triggers a firmware update on a single device.
func UpdateFirmware(ctx context.Context, dev factory.Device) error {
	switch d := dev.(type) {
	case *factory.Gen1Device:
		return updateGen1Firmware(ctx, d)
	case *factory.Gen2Device:
		return updateGen2Firmware(ctx, d)
	default:
		return types.ErrUnsupportedDevice
	}
}

// updateGen1Firmware triggers firmware update on a Gen1 device.
func updateGen1Firmware(ctx context.Context, dev *factory.Gen1Device) error {
	if dev.Device == nil {
		return types.ErrNilDevice
	}

	// Pass empty string to update to latest stable firmware
	return dev.Update(ctx, "")
}

// updateGen2Firmware triggers firmware update on a Gen2 device.
func updateGen2Firmware(ctx context.Context, dev *factory.Gen2Device) error {
	if dev.Device == nil {
		return types.ErrNilDevice
	}

	shellyNS := dev.Shelly()
	return shellyNS.Update(ctx, nil)
}

// BatchUpdateFirmware triggers firmware updates on multiple devices concurrently.
func BatchUpdateFirmware(ctx context.Context, devices []factory.Device) FirmwareResults {
	results := make(FirmwareResults, len(devices))
	var wg sync.WaitGroup

	for i, device := range devices {
		wg.Add(1)
		go func(index int, dev factory.Device) {
			defer wg.Done()

			result := FirmwareResult{Device: dev}
			err := UpdateFirmware(ctx, dev)
			if err != nil {
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

// UpdateDevicesWithAvailableUpdates updates only devices that have firmware updates.
//
// First checks all devices for updates, then triggers updates on devices
// that have new firmware available.
func UpdateDevicesWithAvailableUpdates(ctx context.Context, devices []factory.Device) FirmwareResults {
	// First check for updates
	checkResults := CheckFirmwareUpdates(ctx, devices)

	// Get devices with updates
	updatesAvailable := checkResults.UpdatesAvailable()
	if len(updatesAvailable) == 0 {
		return FirmwareResults{} // No updates available
	}

	// Collect devices that need updates
	devicesToUpdate := make([]factory.Device, len(updatesAvailable))
	for i, r := range updatesAvailable {
		devicesToUpdate[i] = r.Device
	}

	// Trigger updates
	return BatchUpdateFirmware(ctx, devicesToUpdate)
}

// Gen1UpdateInfo contains Gen1 device update information.
type Gen1UpdateInfo = gen1.UpdateInfo
