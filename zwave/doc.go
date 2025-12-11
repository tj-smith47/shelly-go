// Package zwave provides utilities for working with Shelly Wave Z-Wave devices.
//
// Shelly Wave devices are Z-Wave end devices (switches, dimmers, sensors)
// that require a third-party Z-Wave gateway/hub for full operation. They
// support both standard Z-Wave mesh networks and Z-Wave Long Range (ZWLR)
// star topology.
//
// # Device Access
//
// Many Wave devices also include WiFi or Ethernet connectivity, allowing
// direct control via the standard Gen2 RPC API. For devices with IP
// connectivity, use the gen2 package:
//
//	import "github.com/tj-smith47/shelly-go/gen2"
//
//	// Connect to a Wave device with WiFi/Ethernet
//	device := gen2.NewDevice(gen2.Options{
//	    Host: "192.168.1.100",
//	})
//
//	// Use standard Gen2 components
//	sw := components.NewSwitch(device.Client(), 0)
//	status, err := sw.GetStatus(ctx)
//
// # Z-Wave Only Access
//
// For Z-Wave-only operation (no IP connectivity), devices must be accessed
// through their Z-Wave gateway. This package provides utilities for working
// with Wave device profiles and capabilities, but actual device control
// requires integration with your Z-Wave gateway's API.
//
// # Supported Gateways
//
// Shelly Wave devices work with Z-Wave certified gateways including:
//   - Home Assistant with Z-Wave JS
//   - Hubitat Elevation (C-8 or later for ZWLR)
//   - HomeSeer HomeTroller
//   - SmartThings
//   - Vera/ezlo
//   - OpenHAB
//
// # Device Profiles
//
// Wave device profiles are registered in the profiles/wave package:
//
//	import "github.com/tj-smith47/shelly-go/profiles/wave"
//
//	// Profiles are auto-registered on import
//	_ = "github.com/tj-smith47/shelly-go/profiles/wave"
//
// # Security
//
// Wave devices support Z-Wave Security 2 (S2) with three levels:
//   - S2 Authenticated: Full security with DSK verification
//   - S2 Unauthenticated: Basic encryption without DSK
//   - Unsecure: No encryption (not recommended)
//
// The DSK (Device Specific Key) PIN is printed on the device label
// and required for S2 Authenticated inclusion.
//
// # Network Topology
//
// Wave devices support two network types:
//   - Mesh (Z-Wave): Devices can route through other mains-powered devices
//   - Star (Z-Wave Long Range): Direct communication with gateway only
//
// The network topology must be selected during device inclusion and cannot
// be changed without re-including the device.
package zwave
