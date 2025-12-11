// Package blu provides device profiles for Shelly BLU Bluetooth devices.
//
// BLU devices are battery-powered Bluetooth Low Energy (BLE) sensors and
// buttons that communicate via BLE to a Shelly gateway device. They use
// the BTHome protocol for standardized BLE sensor data.
//
// # Registered Devices
//
// This package registers profiles for the following BLU devices:
//
// Buttons:
//   - Shelly BLU Button1 (SBBT-002C)
//   - Shelly BLU RC Button 4 (SBBT-004C)
//
// Sensors:
//   - Shelly BLU Door/Window (SBDW-002C)
//   - Shelly BLU Motion (SBMO-003Z)
//   - Shelly BLU H&T (SBHT-003C)
//
// Climate:
//   - Shelly BLU TRV (SBTR-001Z)
//
// Infrastructure:
//   - Shelly BLU Gateway (SNGW-BT01)
//
// # Architecture
//
// BLU devices communicate via Bluetooth to a gateway device (typically a
// Gen2+ Shelly device with BLE support). The gateway bridges BLE data to
// the local network via HTTP/WebSocket/MQTT.
//
// # Usage
//
// Import this package to automatically register all BLU device profiles:
//
//	import (
//	    _ "github.com/tj-smith47/shelly-go/profiles/blu"
//	    "github.com/tj-smith47/shelly-go/profiles"
//	)
//
//	func main() {
//	    profile := profiles.Get("SBBT-002C") // BLU Button1
//	    if profile != nil {
//	        fmt.Printf("Device: %s (BLE)\n", profile.Name)
//	        fmt.Printf("Battery powered: %v\n",
//	            profile.PowerSource == profiles.PowerSourceBattery)
//	    }
//	}
package blu
