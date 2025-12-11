// Package provisioning provides utilities for initial setup and configuration
// of Shelly devices.
//
// Device provisioning is the process of configuring a new Shelly device with
// network credentials, authentication settings, and other initial configuration.
// This package provides helpers for common provisioning workflows.
//
// # Provisioning Methods
//
// Shelly devices support multiple provisioning methods:
//
//   - WiFi AP Mode: Connect to the device's access point (default SSID:
//     shelly<MODEL>-XXXXXXXXXXXX) and configure via HTTP at 192.168.33.1
//
//   - Bluetooth Low Energy (BLE): Gen2+ devices support configuration via
//     BLE, making provisioning easier without requiring WiFi connection changes
//
//   - USB/Serial: Some devices support configuration via USB connection
//
// # WiFi AP Mode Provisioning
//
// New devices start in AP mode by default. Connect to the device's AP and
// use the WiFi component to configure station mode:
//
//	// Connect to device at 192.168.33.1 (device's AP mode address)
//	transport, _ := transport.NewHTTP("192.168.33.1", nil)
//	client := rpc.NewClient(transport)
//
//	// Configure station mode to connect to your network
//	prov := provisioning.New(client)
//	err := prov.ConfigureWiFi(ctx, &provisioning.WiFiConfig{
//	    SSID:     "YourNetworkSSID",
//	    Password: "YourNetworkPassword",
//	})
//
// # Bulk Provisioning
//
// For provisioning multiple devices with the same configuration:
//
//	config := &provisioning.DeviceConfig{
//	    WiFi: &provisioning.WiFiConfig{
//	        SSID:     "YourNetwork",
//	        Password: "YourPassword",
//	    },
//	    DeviceName: "Kitchen Light",
//	    Timezone:   "America/New_York",
//	}
//
//	// Provision discovered devices
//	results := prov.BulkProvision(ctx, devices, config)
//	for _, r := range results {
//	    if r.Error != nil {
//	        log.Printf("Failed to provision %s: %v", r.Address, r.Error)
//	    }
//	}
//
// # Configuration Steps
//
// A typical provisioning workflow includes:
//
//  1. Discovery: Find new/unconfigured devices (see discovery package)
//  2. Connect: Establish connection to device (AP mode or BLE)
//  3. Configure WiFi: Set station mode credentials
//  4. Set Device Name: Optional but recommended for identification
//  5. Configure Timezone: For accurate scheduling
//  6. Set Authentication: Optional password protection
//  7. Configure Cloud: Optional Shelly Cloud connection
//  8. Verify: Confirm device connects to target network
//
// # Security Considerations
//
// During provisioning:
//   - Devices start with no authentication (open AP)
//   - Set authentication credentials after WiFi is configured
//   - Consider disabling the device's AP after station mode is working
//   - Passwords are sent in clear text over HTTP; use TLS in production
//
// # Post-Provisioning
//
// After successful provisioning:
//   - Device will connect to your WiFi network
//   - Device can be discovered via mDNS (see discovery package)
//   - Use the device's new IP address for further configuration
//   - Consider disabling BLE if not needed for security
package provisioning
