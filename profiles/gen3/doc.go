// Package gen3 provides device profiles for Shelly Gen3 devices.
//
// Gen3 devices build on the Gen2 architecture with improved hardware,
// better performance, and enhanced energy efficiency. They maintain
// API compatibility with Gen2 devices.
//
// # Registered Devices
//
// This package registers profiles for the following Gen3 devices:
//
// Relays:
//   - Shelly 1 Gen3 (S3SW-001X16EU)
//   - Shelly 1PM Gen3 (S3SW-001P16EU)
//   - Shelly 1L Gen3 (S3SW-001X10EU)
//   - Shelly 2PM Gen3 (S3SW-002P16EU)
//   - Shelly 2L Gen3 (S3SW-002X10EU)
//   - Shelly 1 Mini Gen3 (S3SW-001X8EU)
//   - Shelly 1PM Mini Gen3 (S3SW-001P8EU)
//   - Shelly PM Mini Gen3 (S3PM-001PCEU16)
//   - Shelly Shutter Gen3 (S3SH-002P16EU)
//
// Plugs:
//   - Shelly Plug S Gen3 (S3PL-00112EU)
//   - Shelly Outdoor Plug S Gen3 (S3PL-00212EU)
//   - Shelly Plug PM Gen3 (S3PM-001P16EU)
//   - Shelly Plug S MTR Gen3 (S3PL-10112EU)
//
// Lighting:
//   - Shelly Dimmer Gen3 (S3DM-0010WW)
//   - Shelly Dimmer 0/1-10V PM Gen3 (S3DM-0D10WW)
//   - Shelly DALI Dimmer Gen3 (S3DL-0010WW)
//
// Sensors:
//   - Shelly i4 Gen3 (S3SN-0024X)
//   - Shelly H&T Gen3 (S3HT-0A01)
//   - Shelly EM Gen3 (S3EM-002CXCEU)
//   - Shelly 3EM-63 Gen3 (S3EM-003CXCEU63)
//
// # Usage
//
// Import this package to automatically register all Gen3 device profiles:
//
//	import (
//	    _ "github.com/tj-smith47/shelly-go/profiles/gen3"
//	    "github.com/tj-smith47/shelly-go/profiles"
//	)
//
//	func main() {
//	    profile := profiles.GetByApp("Plus1G3")
//	    if profile != nil {
//	        fmt.Printf("Device: %s (Gen3)\n", profile.Name)
//	    }
//	}
package gen3
