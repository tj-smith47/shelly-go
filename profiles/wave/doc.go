// Package wave provides device profiles for Shelly Wave Z-Wave devices.
//
// Wave devices use the Z-Wave protocol for mesh networking and communicate
// with Z-Wave controllers. They provide reliable, low-latency home automation
// with excellent range through the Z-Wave mesh.
//
// # Registered Devices
//
// This package registers profiles for the following Wave devices:
//
// Relays:
//   - Shelly Wave 1 (SNSW-001X16ZW)
//   - Shelly Wave 1PM (SNSW-001P16ZW)
//   - Shelly Wave 2PM (SNSW-002P16ZW)
//
// Plugs:
//   - Shelly Wave Plug US (SNPL-00116ZW)
//
// Covers:
//   - Shelly Wave Shutter (SNSH-002P16ZW)
//
// Pro Relays:
//   - Shelly Wave Pro 1 (SPSW-001XE16ZW)
//   - Shelly Wave Pro 1PM (SPSW-001PE16ZW)
//   - Shelly Wave Pro 2 (SPSW-002XE16ZW)
//   - Shelly Wave Pro 2PM (SPSW-002PE16ZW)
//   - Shelly Wave Pro 3 (SPSW-003XE16ZW)
//
// # Z-Wave Features
//
// Wave devices support standard Z-Wave features:
//   - S2 Security framework
//   - SmartStart for easy inclusion
//   - Z-Wave mesh networking
//   - Association groups for direct device control
//   - Central Scene for multi-tap button events
//
// # Usage
//
// Import this package to automatically register all Wave device profiles:
//
//	import (
//	    _ "github.com/tj-smith47/shelly-go/profiles/wave"
//	    "github.com/tj-smith47/shelly-go/profiles"
//	)
//
//	func main() {
//	    profile := profiles.Get("SNSW-001X16ZW") // Wave 1
//	    if profile != nil {
//	        fmt.Printf("Device: %s (Z-Wave)\n", profile.Name)
//	        fmt.Printf("Z-Wave support: %v\n", profile.Protocols.ZWave)
//	    }
//	}
package wave
