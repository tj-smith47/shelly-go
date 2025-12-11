package zwave

// InclusionMode represents the Z-Wave inclusion method.
type InclusionMode string

const (
	// InclusionSmartStart uses the device QR code for automatic inclusion.
	// The gateway scans the QR code and the device is automatically added
	// when powered on within range.
	InclusionSmartStart InclusionMode = "smart_start"

	// InclusionButton uses the device's S button for manual inclusion.
	// Press and hold the S button until the LED turns solid blue, then
	// press and hold again until the LED blinks faster.
	InclusionButton InclusionMode = "button"

	// InclusionSwitch uses the connected switch for manual inclusion.
	// Toggle the connected switch 3 times quickly to enter Learn mode.
	InclusionSwitch InclusionMode = "switch"
)

// InclusionInfo provides information for including a device in a Z-Wave network.
type InclusionInfo struct {
	Mode         InclusionMode
	Instructions []string
	DSKRequired  bool
}

// GetInclusionInfo returns inclusion instructions for a device.
//
// Example:
//
//	device := zwave.NewDevice(profiles.Get("SNSW-001P16ZW"))
//	info := zwave.GetInclusionInfo(device, zwave.InclusionButton)
//	for _, step := range info.Instructions {
//	    fmt.Println(step)
//	}
func GetInclusionInfo(device *Device, mode InclusionMode) *InclusionInfo {
	info := &InclusionInfo{
		Mode:        mode,
		DSKRequired: true, // S2 Authenticated recommended
	}

	switch mode {
	case InclusionSmartStart:
		info.Instructions = []string{
			"1. Enable SmartStart on your Z-Wave gateway",
			"2. Scan the QR code on the device or its packaging",
			"3. Power on the device within range of the gateway",
			"4. The device will automatically join within 10 minutes",
		}
	case InclusionButton:
		info.Instructions = []string{
			"1. Put your Z-Wave gateway into inclusion mode",
			"2. Press and hold the S button until the LED turns solid blue",
			"3. Release the S button",
			"4. Press and hold the S button again (> 2 seconds) until the LED blinks faster",
			"5. Release the S button to start Learn mode",
			"6. Enter the 5-digit DSK PIN from the device label if prompted",
		}
	case InclusionSwitch:
		info.Instructions = []string{
			"1. Put your Z-Wave gateway into inclusion mode",
			"2. Toggle the connected switch 3 times quickly",
			"3. The device will enter Learn mode (LED indicates status)",
			"4. Enter the 5-digit DSK PIN from the device label if prompted",
		}
	}

	return info
}

// ExclusionInfo provides information for removing a device from a Z-Wave network.
type ExclusionInfo struct {
	// Mode is the recommended exclusion method.
	Mode InclusionMode

	// Instructions provides step-by-step exclusion instructions.
	Instructions []string
}

// GetExclusionInfo returns exclusion instructions for a device.
//
// Example:
//
//	device := zwave.NewDevice(profiles.Get("SNSW-001P16ZW"))
//	info := zwave.GetExclusionInfo(device, zwave.InclusionButton)
//	for _, step := range info.Instructions {
//	    fmt.Println(step)
//	}
func GetExclusionInfo(device *Device, mode InclusionMode) *ExclusionInfo {
	info := &ExclusionInfo{
		Mode: mode,
	}

	switch mode {
	case InclusionButton:
		info.Instructions = []string{
			"1. Put your Z-Wave gateway into exclusion mode",
			"2. Press and hold the S button until the LED turns solid blue",
			"3. Release the S button",
			"4. Press and hold the S button again (> 2 seconds) until the LED blinks faster",
			"5. Release the S button to start Learn mode",
			"6. The device will be removed from the network",
		}
	case InclusionSwitch:
		info.Instructions = []string{
			"1. Put your Z-Wave gateway into exclusion mode",
			"2. Toggle the connected switch 3 times quickly",
			"3. The device will enter Learn mode and be removed from the network",
		}
	default:
		// SmartStart devices still need manual exclusion
		info.Instructions = []string{
			"1. Put your Z-Wave gateway into exclusion mode",
			"2. Press and hold the S button until the LED turns solid blue",
			"3. Release the S button",
			"4. Press and hold the S button again (> 2 seconds) until the LED blinks faster",
			"5. Release the S button to start Learn mode",
			"6. The device will be removed from the network",
		}
	}

	return info
}

// FactoryResetInfo provides factory reset instructions.
type FactoryResetInfo struct {
	// Warning provides important information about the reset.
	Warning string

	// Instructions provides step-by-step reset instructions.
	Instructions []string
}

// GetFactoryResetInfo returns factory reset instructions for a device.
//
// Factory reset should only be used when the gateway is missing or
// inoperable. All custom parameters, associations, and routing information
// will be lost.
//
// Example:
//
//	device := zwave.NewDevice(profiles.Get("SNSW-001P16ZW"))
//	info := zwave.GetFactoryResetInfo(device)
//	fmt.Println(info.Warning)
func GetFactoryResetInfo(device *Device) *FactoryResetInfo {
	return &FactoryResetInfo{
		Warning: "Factory reset will delete all custom parameters, stored values " +
			"(kWh, associations, routings), HOME ID, and NODE ID. Use only when " +
			"the gateway is missing or inoperable.",
		Instructions: []string{
			"1. Press and hold the S button for more than 5 seconds",
			"2. The LED will start blinking rapidly",
			"3. Continue holding for another 5 seconds (10 seconds total)",
			"4. Release the S button when the LED turns solid",
			"5. The device is now factory reset",
		},
	}
}
