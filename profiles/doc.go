// Package profiles provides device profile definitions for all Shelly devices.
//
// Device profiles define the capabilities, components, and limitations of each
// Shelly device model. This information is used for capability detection,
// validation, and auto-configuration.
//
// # Profile Structure
//
// Each device profile contains:
//   - Model identifier and device name
//   - Generation (Gen1, Gen2, Gen3, Gen4)
//   - Available components and their counts
//   - Capability flags (power metering, cover support, etc.)
//   - Protocol support (HTTP, WebSocket, MQTT, CoIoT, Matter, Zigbee)
//   - Resource limits (max scripts, schedules, etc.)
//
// # Usage
//
//	// Get profile by model ID
//	profile, ok := profiles.Get("SNSW-001P16EU")
//	if ok {
//	    fmt.Printf("Device: %s\n", profile.Name)
//	    fmt.Printf("Switches: %d\n", profile.Components.Switches)
//	    fmt.Printf("Has Power Metering: %v\n", profile.Capabilities.PowerMetering)
//	}
//
//	// Get profile by device app name
//	profile, ok = profiles.GetByApp("Plus1PM")
//	if ok {
//	    fmt.Printf("Model: %s\n", profile.Model)
//	}
//
//	// List all profiles for a generation
//	gen3Profiles := profiles.ListByGeneration(types.Gen3)
//	for _, p := range gen3Profiles {
//	    fmt.Printf("- %s (%s)\n", p.Name, p.Model)
//	}
//
// # Capability Detection
//
// Profiles can be used to detect what operations a device supports:
//
//	profile, _ := profiles.Get("SHSW-25")
//	if profile.Capabilities.CoverSupport {
//	    // Device supports roller/cover mode
//	}
//	if profile.Capabilities.PowerMetering {
//	    // Device has power metering
//	}
//
// # Protocol Support
//
// Each profile indicates which protocols the device supports:
//
//	if profile.Protocols.WebSocket {
//	    // Connect via WebSocket for real-time updates
//	}
//	if profile.Protocols.CoIoT {
//	    // Gen1 device with CoIoT support
//	}
//	if profile.Protocols.Matter {
//	    // Gen4 device with Matter support
//	}
//
// # Device Categories
//
// Profiles are organized into categories by generation:
//   - Gen1: Classic devices (Shelly 1, 2.5, Plug, RGBW2, etc.)
//   - Gen2 Plus: Plus series (Plus 1, Plus 1PM, Plus 2PM, etc.)
//   - Gen2 Pro: Professional series (Pro 1PM, Pro 3EM, etc.)
//   - Gen3: Third generation (1 Gen3, 1PM Gen3, 2PM Gen3, etc.)
//   - Gen4: Fourth generation with Matter/Zigbee (1 Gen4, 1PM Gen4, etc.)
//   - BLU: Bluetooth devices (BLU Button, BLU Door/Window, etc.)
//   - Wave: Z-Wave devices (Wave 1, Wave 1PM, Wave 2PM, etc.)
//
// For the complete list of supported devices, see the DEVICES.md file.
package profiles
