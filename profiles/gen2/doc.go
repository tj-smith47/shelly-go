// Package gen2 provides device profiles for Shelly Gen2 Plus and Pro devices.
//
// Gen2 devices use the modern RPC-based API with support for HTTP, WebSocket,
// MQTT, and BLE transports. They include both the Plus consumer line and
// Pro professional devices.
//
// # Registered Devices
//
// This package registers profiles for the following Gen2 devices:
//
// Plus Relays:
//   - Shelly Plus 1 (SNSW-001X16EU)
//   - Shelly Plus 1PM (SNSW-001P16EU)
//   - Shelly Plus 2PM (SNSW-002P16EU)
//   - Shelly Plus 1 Mini (SNSW-001X8EU)
//   - Shelly Plus 1PM Mini (SNSW-001P8EU)
//
// Plus Plugs:
//   - Shelly Plus Plug S (SNPL-00112EU)
//   - Shelly Plus Plug US (SNPL-00116US)
//   - Shelly Plus Plug UK (SNPL-00110GB)
//   - Shelly Plus Plug IT (SNPL-00110IT)
//   - Shelly Plus PM Mini (SNPM-001PCEU16)
//
// Plus Lighting:
//   - Shelly Plus Wall Dimmer (SNDM-0013US)
//   - Shelly Plus RGBW PM (SNDC-0D4P10WW)
//   - Shelly Plus 0-10V Dimmer (SNDM-00100WW)
//
// Plus Inputs:
//   - Shelly Plus i4 (SNSN-0024X)
//   - Shelly Plus i4 DC (SNSN-0D24X)
//   - Shelly Plus UNI (SHPUNI)
//
// Plus Sensors:
//   - Shelly Plus H&T (SNSN-0013A)
//   - Shelly Plus Smoke (SNSN-0031Z)
//
// Pro Relays:
//   - Shelly Pro 1 (SPSW-001XE16EU)
//   - Shelly Pro 1PM (SPSW-001PE16EU)
//   - Shelly Pro 2 (SPSW-002XE16EU)
//   - Shelly Pro 2PM (SPSW-002PE16EU)
//   - Shelly Pro 3 (SPSW-003XE16EU)
//   - Shelly Pro 4PM (SPSW-004PE16EU)
//
// Pro Energy:
//   - Shelly Pro 3EM (SPEM-003CEBEU400)
//   - Shelly Pro EM-50 (SPEM-002CEBEU50)
//
// Pro Lighting:
//   - Shelly Pro Dimmer 1PM (SPDM-001PE01EU)
//   - Shelly Pro Dimmer 2PM (SPDM-002PE01EU)
//
// Pro Covers:
//   - Shelly Pro Dual Cover PM (SPSH-002PE16EU)
//
// # Usage
//
// Import this package to automatically register all Gen2 device profiles:
//
//	import (
//	    _ "github.com/tj-smith47/shelly-go/profiles/gen2"
//	    "github.com/tj-smith47/shelly-go/profiles"
//	)
//
//	func main() {
//	    profile := profiles.GetByApp("Plus1PM")
//	    if profile != nil {
//	        fmt.Printf("Device: %s\n", profile.Name)
//	    }
//	}
package gen2
