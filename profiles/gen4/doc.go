// Package gen4 provides device profiles for Shelly Gen4 devices.
//
// Gen4 devices represent the latest generation with native Matter and
// Zigbee protocol support in addition to the standard WiFi-based RPC API.
// They are designed for seamless integration with smart home ecosystems.
//
// # Registered Devices
//
// This package registers profiles for the following Gen4 devices:
//
// Relays:
//   - Shelly 1 Gen4 (S4SW-001X16EU)
//   - Shelly 1PM Gen4 (S4SW-001P16EU)
//   - Shelly 2PM Gen4 (S4SW-002P16EU)
//   - Shelly 1 Mini Gen4 (S4SW-001X8EU)
//   - Shelly 1PM Mini Gen4 (S4SW-001P8EU)
//
// Energy:
//   - Shelly EM Mini Gen4 (S4EM-001XCEU)
//
// Plugs:
//   - Shelly Plug US Gen4 (S4PL-00116US)
//
// Sensors:
//   - Shelly Flood Sensor Gen4 (S4SN-001X)
//
// Displays:
//   - Shelly Wall Display X2 (S4WD-002X)
//
// # Protocol Support
//
// Gen4 devices support:
//   - HTTP/WebSocket RPC API (same as Gen2/Gen3)
//   - MQTT
//   - BLE
//   - Matter (native, no bridge required)
//   - Zigbee (via gateway integration)
//
// # Usage
//
// Import this package to automatically register all Gen4 device profiles:
//
//	import (
//	    _ "github.com/tj-smith47/shelly-go/profiles/gen4"
//	    "github.com/tj-smith47/shelly-go/profiles"
//	)
//
//	func main() {
//	    profile := profiles.GetByApp("Plus1G4")
//	    if profile != nil {
//	        fmt.Printf("Device: %s (Gen4 with Matter)\n", profile.Name)
//	        fmt.Printf("Matter support: %v\n", profile.Protocols.Matter)
//	    }
//	}
package gen4
