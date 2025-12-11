// Package gen1 provides device profiles for Shelly Gen1 devices.
//
// Gen1 devices use HTTP REST API and CoIoT protocol for communication.
// They include the original Shelly product line before the Gen2 architecture.
//
// # Registered Devices
//
// This package registers profiles for the following Gen1 devices:
//
// Relays:
//   - Shelly 1 (SHSW-1)
//   - Shelly 1PM (SHSW-PM)
//   - Shelly 1L (SHSW-L)
//   - Shelly 2 (SHSW-21)
//   - Shelly 2.5 (SHSW-25)
//   - Shelly 4Pro (SHSW-44)
//
// Plugs:
//   - Shelly Plug (SHPLG-1)
//   - Shelly Plug S (SHPLG-S)
//   - Shelly Plug US (SHPLG-U1)
//
// Lighting:
//   - Shelly Dimmer 2 (SHDM-2)
//   - Shelly Bulb RGBW (SHBLB-1)
//   - Shelly Vintage (SHVIN-1)
//   - Shelly Duo (SHBDUO-1)
//   - Shelly RGBW2 (SHRGBW2)
//
// Energy Monitoring:
//   - Shelly EM (SHEM)
//   - Shelly 3EM (SHEM-3)
//
// Inputs:
//   - Shelly i3 (SHIX3-1)
//   - Shelly Button1 (SHBTN-1)
//   - Shelly UNI (SHUNI-1)
//
// Sensors:
//   - Shelly H&T (SHHT-1)
//   - Shelly Flood (SHWT-1)
//   - Shelly Door/Window (SHDW-1)
//   - Shelly Door/Window 2 (SHDW-2)
//   - Shelly Motion (SHMOS-01)
//   - Shelly Gas (SHGS-1)
//   - Shelly Smoke (SHSM-01)
//
// # Usage
//
// Import this package to automatically register all Gen1 device profiles:
//
//	import (
//	    _ "github.com/tj-smith47/shelly-go/profiles/gen1"
//	    "github.com/tj-smith47/shelly-go/profiles"
//	)
//
//	func main() {
//	    profile := profiles.Get("SHSW-1") // Shelly 1
//	    if profile != nil {
//	        fmt.Printf("Device: %s (%s)\n", profile.Name, profile.Model)
//	    }
//	}
package gen1
