package types

import "fmt"

// Generation represents the hardware/firmware generation of a Shelly device.
// Different generations use different protocols and APIs.
type Generation int

const (
	// GenerationUnknown indicates the generation could not be determined.
	GenerationUnknown Generation = 0
	// GenUnknown is an alias for GenerationUnknown
	GenUnknown = GenerationUnknown

	// Generation1 represents Gen1 devices (pre-2021).
	// Protocol: HTTP REST API, CoIoT for real-time updates
	// Examples: Shelly 1, 1PM, 2.5, Plug, Bulb, RGBW2, Dimmer, EM, H&T
	Generation1 Generation = 1
	// Gen1 is an alias for Generation1
	Gen1 = Generation1

	// Generation2 represents Gen2 devices (Plus series, Pro series).
	// Protocol: RPC over HTTP/WebSocket, MQTT
	// Examples: Shelly Plus 1, 1PM, 2PM, Pro 1PM, Pro 2PM, Pro 3EM
	Generation2 Generation = 2
	// Gen2 is an alias for Generation2
	Gen2 = Generation2

	// Gen2Plus represents Gen2 Plus devices (consumer-grade Gen2).
	// Protocol: RPC over HTTP/WebSocket, MQTT
	// Examples: Shelly Plus 1, 1PM, 2PM, Plug S, H&T
	Gen2Plus = Generation2

	// Gen2Pro represents Gen2 Pro devices (professional-grade Gen2).
	// Protocol: RPC over HTTP/WebSocket, MQTT
	// Features: Ethernet, DIN rail mounting, extended temperature range
	// Examples: Shelly Pro 1, 1PM, 2PM, 3EM, 4PM
	Gen2Pro = Generation2

	// Generation3 represents Gen3 devices (2023+).
	// Protocol: Same as Gen2 (RPC) with hardware improvements
	// Examples: Shelly 1 Gen3, 1PM Gen3, 2PM Gen3, Wall Display
	Generation3 Generation = 3
	// Gen3 is an alias for Generation3
	Gen3 = Generation3

	// Generation4 represents Gen4 devices (2024+).
	// Protocol: Same as Gen2 (RPC) with latest features
	// Examples: Shelly 1 Gen4, 1PM Gen4
	Generation4 Generation = 4
	// Gen4 is an alias for Generation4
	Gen4 = Generation4
)

// String returns the string representation of the generation.
func (g Generation) String() string {
	switch g {
	case Generation1:
		return "Gen1"
	case Generation2:
		return "Gen2"
	case Generation3:
		return "Gen3"
	case Generation4:
		return "Gen4"
	default:
		return "Unknown"
	}
}

// IsRPC returns true if the generation uses RPC protocol.
// Gen2, Gen3, and Gen4 all use RPC over HTTP/WebSocket.
func (g Generation) IsRPC() bool {
	return g >= Generation2 && g <= Generation4
}

// IsREST returns true if the generation uses REST protocol.
// Only Gen1 devices use REST API.
func (g Generation) IsREST() bool {
	return g == Generation1
}

// MarshalText implements encoding.TextMarshaler.
func (g Generation) MarshalText() ([]byte, error) {
	return []byte(g.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (g *Generation) UnmarshalText(text []byte) error {
	switch string(text) {
	case "Gen1", "gen1", "1":
		*g = Generation1
	case "Gen2", "gen2", "2", "Plus", "Pro":
		*g = Generation2
	case "Gen3", "gen3", "3":
		*g = Generation3
	case "Gen4", "gen4", "4":
		*g = Generation4
	case "Unknown", "unknown", "":
		*g = GenerationUnknown
	default:
		return fmt.Errorf("unknown generation: %s", text)
	}
	return nil
}
