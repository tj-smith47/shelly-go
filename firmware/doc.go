// Package firmware provides OTA (Over-The-Air) firmware management for Shelly devices.
//
// This package enables checking for firmware updates, applying updates, and
// managing firmware across multiple devices. It supports both stable and beta
// firmware channels.
//
// # Basic Usage
//
// Check for available updates:
//
//	client := rpc.NewClient(transport)
//	fw := firmware.New(client)
//
//	// Check for updates
//	info, err := fw.CheckForUpdate(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if info.HasUpdate() {
//	    fmt.Printf("Update available: %s -> %s\n",
//	        info.Current, info.Available)
//	}
//
// Apply an update:
//
//	if info.HasUpdate() {
//	    err := fw.Update(ctx, nil)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    fmt.Println("Update started, device will reboot")
//	}
//
// # Update Options
//
// Control update behavior:
//
//	opts := &firmware.UpdateOptions{
//	    // Use beta channel
//	    Stage: "beta",
//
//	    // Or specify a specific URL
//	    URL: "http://firmware.server/custom.zip",
//	}
//	err := fw.Update(ctx, opts)
//
// # Batch Updates
//
// Update multiple devices:
//
//	devices := []types.Device{device1, device2, device3}
//	results := firmware.BatchCheckUpdates(ctx, devices)
//
//	for _, r := range results {
//	    if r.Info.HasUpdate() {
//	        fmt.Printf("%s: update available\n", r.Device.Address())
//	    }
//	}
//
// # Gen1 vs Gen2+ Differences
//
// Firmware update methods differ between device generations:
//
// Gen2+ devices:
//   - Use Shelly.CheckForUpdate and Shelly.Update RPC methods
//   - Support staging (beta, stable)
//   - Support automatic updates via Cloud
//
// Gen1 devices:
//   - Use /ota/check and /ota HTTP endpoints
//   - Support manual URL-based updates
//
// This package provides a unified interface that works with both generations.
//
// # Update Process
//
// The firmware update process:
//  1. Check for available updates (CheckForUpdate)
//  2. Download firmware (handled by device)
//  3. Apply update and reboot (Update)
//  4. Device restarts with new firmware
//
// The update process typically takes 1-5 minutes depending on device and
// firmware size. The device will be unavailable during the update.
//
// # Rollback
//
// Shelly devices support firmware rollback in case of issues:
//
//	// Get rollback status
//	status, err := fw.GetRollbackStatus(ctx)
//	if status.CanRollback {
//	    err := fw.Rollback(ctx)
//	}
//
// Note: Rollback is only available for a limited time after an update.
//
// # Safety Considerations
//
//   - Always check current firmware version before updating
//   - Ensure stable network connection during updates
//   - Don't power cycle devices during updates
//   - Consider staged rollouts for production environments
//   - Test updates on non-critical devices first
package firmware
