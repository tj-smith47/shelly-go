// Package matter provides Matter protocol support for Shelly Gen4+ devices.
//
// Matter is a unified smart home connectivity standard that allows devices
// from different manufacturers to work together seamlessly. Shelly Gen4
// devices support Matter natively, alongside Wi-Fi, Bluetooth, and Zigbee.
//
// # Overview
//
// The matter package provides:
//   - Matter component configuration via RPC
//   - Commissioning status and code retrieval
//   - Fabric (network) management
//   - Factory reset for Matter data
//
// # Supported Devices
//
// Matter is supported on Gen4 devices including:
//   - Shelly 1 Gen4, Shelly 1PM Gen4
//   - Shelly 1 Mini Gen4, Shelly 1PM Mini Gen4
//   - Shelly 2PM Gen4
//   - Shelly EM Mini Gen4
//   - Shelly Plug US Gen4
//
// # Commissioning Process
//
// To commission a Shelly device into a Matter network:
//
//  1. Enable Matter on the device:
//
//     matter := NewMatter(rpcClient)
//     config, _ := matter.GetConfig(ctx)
//     config.Enable = boolPtr(true)
//     matter.SetConfig(ctx, config)
//
//  2. Get the pairing code from device status or web UI
//
//  3. Use the pairing code in your Matter controller (Apple Home, Google Home, etc.)
//
// # Switching Protocols
//
// Gen4 devices can switch between Matter and Zigbee protocols:
//   - Press the button 5 times to switch from Matter (default) to Zigbee
//   - The device enters inclusion mode for 3 minutes
//   - Press 3 times to restart inclusion if the window is missed
//
// # Example Usage
//
//	// Create Matter component
//	matter := matter.NewMatter(rpcClient)
//
//	// Get current configuration
//	config, err := matter.GetConfig(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check if Matter is enabled
//	if config.Enable {
//	    fmt.Println("Matter is enabled")
//	}
//
//	// Get commissioning status
//	status, err := matter.GetStatus(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if status.Commissionable {
//	    fmt.Println("Device is ready for commissioning")
//	    fmt.Printf("Paired fabrics: %d\n", status.FabricsCount)
//	}
//
//	// Factory reset Matter settings (unpairs all fabrics)
//	err = matter.FactoryReset(ctx)
//
// # Fabric Management
//
// A "fabric" in Matter terminology represents a network/ecosystem that the
// device is paired with. A device can be commissioned to multiple fabrics
// (multi-admin). The status includes the count of paired fabrics.
//
// # Platform Compatibility
//
// Matter-enabled Shelly devices work with:
//   - Apple HomeKit / Home app
//   - Google Home
//   - Amazon Alexa
//   - Samsung SmartThings
//   - Other Matter-certified controllers
//
// # Notes
//
//   - Matter commissioning codes can be obtained from the device web UI,
//     Shelly Cloud app, or from the Matter sticker on some devices
//   - The device must have Matter enabled and be in commissionable state
//   - Only Gen4+ devices support Matter; earlier generations do not
//   - Matter.FactoryReset only resets Matter settings, not the entire device
package matter
