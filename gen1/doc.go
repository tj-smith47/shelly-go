// Package gen1 provides support for Shelly Gen1 devices.
//
// Gen1 devices (released before 2021) use a REST API over HTTP for configuration
// and control, with optional CoIoT (CoAP) protocol for real-time status updates.
//
// # Supported Devices
//
// Gen1 devices include:
//   - Shelly 1, 1PM, 1L
//   - Shelly 2, 2.5
//   - Shelly 4Pro
//   - Shelly Plug, Plug S, Plug US
//   - Shelly Bulb, Vintage, Duo
//   - Shelly RGBW, RGBW2
//   - Shelly Dimmer, Dimmer 2
//   - Shelly EM, 3EM
//   - Shelly i3
//   - Shelly Button1
//   - Shelly Gas, Smoke, Flood
//   - Shelly Door/Window, Door/Window 2
//   - Shelly H&T
//   - Shelly Motion
//   - Shelly UNI
//
// # Protocol Overview
//
// Gen1 devices use HTTP REST API with these key endpoints:
//
//   - /shelly - Device identification and capabilities
//   - /settings - Device configuration
//   - /settings/relay/{id} - Per-relay settings
//   - /settings/roller/{id} - Per-roller settings
//   - /status - Current device status
//   - /relay/{id} - Relay control
//   - /roller/{id} - Roller/cover control
//   - /light/{id} - Light control
//   - /color/{id} - RGBW color control
//   - /white/{id} - White channel control
//   - /meter/{id} - Power meter readings
//   - /emeter/{id} - Energy meter readings
//
// # Usage
//
// Create a device with an HTTP transport:
//
//	import (
//	    "github.com/tj-smith47/shelly-go/gen1"
//	    "github.com/tj-smith47/shelly-go/transport"
//	)
//
//	// Create transport
//	t := transport.NewHTTP("http://192.168.1.100")
//
//	// Create device
//	device := gen1.NewDevice(t)
//
//	// Get device info
//	info, err := device.GetDeviceInfo(ctx)
//
//	// Control a relay
//	relay := device.Relay(0)
//	err = relay.TurnOn(ctx)
//
// # Authentication
//
// Gen1 devices support HTTP Basic authentication. Configure credentials
// using transport options:
//
//	t := transport.NewHTTP("http://192.168.1.100",
//	    transport.WithAuth("admin", "password"))
//
// # Real-Time Updates (CoIoT)
//
// Gen1 devices can publish status updates via CoIoT (CoAP over UDP).
// Use the CoIoT listener for real-time status updates:
//
//	listener := gen1.NewCoIoTListener()
//	err := listener.Start()
//
//	listener.OnStatus(func(deviceID string, status *gen1.Status) {
//	    fmt.Printf("Device %s status: %+v\n", deviceID, status)
//	})
//
// # Differences from Gen2
//
// Gen1 and Gen2+ devices have different APIs:
//
//   - Gen1 uses HTTP REST endpoints; Gen2+ uses RPC over HTTP/WebSocket
//   - Gen1 uses action URLs for automation; Gen2+ uses webhooks/scripts
//   - Gen1 component IDs are in URL path; Gen2+ uses method parameters
//   - Gen1 status is single JSON blob; Gen2+ has per-component status
//
// For Gen2+ devices, use the gen2 package instead.
package gen1
