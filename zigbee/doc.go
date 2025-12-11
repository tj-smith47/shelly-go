// Package zigbee provides support for Shelly Gen2+ Zigbee component.
//
// The Zigbee component handles Zigbee connectivity services of a device,
// allowing Shelly devices to participate in Zigbee networks. This enables
// integration with Zigbee coordinators (e.g., Home Assistant, Zigbee2MQTT,
// deCONZ) and interoperability with other Zigbee devices.
//
// # Supported Devices
//
// Zigbee support is available on Gen4 devices and select Gen3 devices that
// have integrated Zigbee radios:
//   - Shelly 1 Gen4 (with Zigbee/Thread support)
//   - Shelly 1PM Gen4 (with Zigbee/Thread support)
//   - Other Gen4 devices with multi-protocol support
//
// # Features
//
// The Zigbee component provides:
//   - Enable/disable Zigbee connectivity
//   - Network steering for joining Zigbee networks
//   - Network state monitoring
//   - EUI64 address information
//   - PAN ID and channel information
//
// # Basic Usage
//
// Create a Zigbee component instance and query its status:
//
//	client := rpc.NewClient(transport)
//	zb := zigbee.NewZigbee(client)
//
//	// Check if Zigbee is enabled
//	enabled, err := zb.IsEnabled(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Zigbee enabled:", enabled)
//
//	// Get network status
//	status, err := zb.GetStatus(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Network state:", status.NetworkState)
//
// # Joining a Zigbee Network
//
// To join a Zigbee network, ensure Zigbee is enabled and trigger network
// steering while your coordinator is in pairing mode:
//
//	// Enable Zigbee if not already enabled
//	if err := zb.Enable(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Start network steering (device will attempt to join nearby networks)
//	if err := zb.StartNetworkSteering(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check if joined
//	status, err := zb.GetStatus(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if status.NetworkState == "joined" {
//	    fmt.Println("Successfully joined Zigbee network!")
//	}
//
// # Network States
//
// The NetworkState field in Status can have the following values:
//   - "not_configured" - Zigbee is disabled or not set up
//   - "ready" - Zigbee is enabled but not joined to a network
//   - "steering" - Currently attempting to join a network
//   - "joined" - Successfully joined a Zigbee network
//
// # Limitations
//
// When operating in Zigbee mode, some features may be unavailable:
//   - CloudRelay is not supported on Gen4 devices in Zigbee mode
//   - Bluetooth scanning is not supported in Zigbee mode
//   - The device operates as a Zigbee end device or router, not a coordinator
//
// # Integration with Coordinators
//
// Shelly devices with Zigbee support can be paired with various coordinators:
//   - Home Assistant (ZHA or Zigbee2MQTT)
//   - deCONZ (Phoscon)
//   - Zigbee2MQTT (standalone)
//   - Other Zigbee 3.0 compatible coordinators
//
// The Shelly RPC cluster (custom cluster ID 0xFC02) allows Zigbee networks
// to read, modify, and apply the station configuration of the Shelly WiFi
// component through Zigbee endpoint 239.
//
// # Thread Support
//
// Gen4 devices with Zigbee support often also support Thread/Matter.
// See the matter package for Matter protocol support. Thread and Zigbee
// typically share the same radio, so only one can be active at a time.
package zigbee
