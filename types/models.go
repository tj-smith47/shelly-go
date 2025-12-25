package types

// modelNames maps Shelly model codes to human-readable product names.
// Model codes come from DeviceInfo.Model field.
var modelNames = map[string]string{
	// Gen1 devices
	"SHSW-1":     "Shelly 1",
	"SHSW-PM":    "Shelly 1PM",
	"SHSW-21":    "Shelly 2",
	"SHSW-25":    "Shelly 2.5",
	"SHSW-44":    "Shelly 4Pro",
	"SHPLG-1":    "Shelly Plug",
	"SHPLG-S":    "Shelly Plug S",
	"SHPLG-U1":   "Shelly Plug US",
	"SHEM":       "Shelly EM",
	"SHEM-3":     "Shelly 3EM",
	"SHRGBW2":    "Shelly RGBW2",
	"SHDW-1":     "Shelly Door/Window",
	"SHDW-2":     "Shelly Door/Window 2",
	"SHMOS-01":   "Shelly Motion",
	"SHMOS-02":   "Shelly Motion 2",
	"SHHT-1":     "Shelly H&T",
	"SHFLOOD-1":  "Shelly Flood",
	"SHDM-1":     "Shelly Dimmer",
	"SHDM-2":     "Shelly Dimmer 2",
	"SHIX3-1":    "Shelly i3",
	"SH2LED-1":   "Shelly 2LED",
	"SHBTN-1":    "Shelly Button1",
	"SHBTN-2":    "Shelly Button1 v2",
	"SHGS-1":     "Shelly Gas",
	"SHVIN-1":    "Shelly Vintage",
	"SHUNI-1":    "Shelly UNI",
	"SHBDUO-1":   "Shelly Bulb Duo",
	"SHBLB-1":    "Shelly Bulb",
	"SHCL-255":   "Shelly Color",
	"SHCB-1":     "Shelly Color Bulb",
	"SHTRV-01":   "Shelly TRV",

	// Gen2 Plus devices (EU variants - most common)
	"SNSW-001P16EU":  "Shelly Plus 1",
	"SNSW-001X16EU":  "Shelly Plus 1 Mini",
	"SNSW-102P16EU":  "Shelly Plus 1PM",
	"SNSW-102X16EU":  "Shelly Plus 1PM Mini",
	"SNSW-002P16EU":  "Shelly Plus 2PM",
	"SNPL-00112EU":   "Shelly Plus Plug S",
	"SNPL-00116US":   "Shelly Plus Plug US",
	"SNPL-00112US":   "Shelly Plus Plug US",
	"SNSN-0024X":     "Shelly Plus Add-On",
	"SNDC-0D4P10WW":  "Shelly Plus 0-10V Dimmer",
	"SNDM-00100WW":   "Shelly Plus Wall Dimmer",
	"SNDM-0013US":    "Shelly Plus Wall Dimmer US",
	"SNSN-0031Z":     "Shelly Plus Smoke",
	"SNSW-001P8EU":   "Shelly Plus 1 (8A)",
	"SNGW-BT01":      "Shelly Plus BT Gateway",
	"SNSN-0043X":     "Shelly Plus HT Gen3",

	// Gen2 Plus i4 variants
	"SNSN-0024XNEU":  "Shelly Plus i4",
	"SNSN-0024XUS":   "Shelly Plus i4 US",
	"SNSN-0D24X":     "Shelly Plus i4 DC",

	// Gen2 Pro devices
	"SPSW-001XE16EU": "Shelly Pro 1",
	"SPSW-001PE16EU": "Shelly Pro 1PM",
	"SPSW-002XE16EU": "Shelly Pro 2",
	"SPSW-002PE16EU": "Shelly Pro 2PM",
	"SPSW-003XE16EU": "Shelly Pro 3",
	"SPSW-004PE16EU": "Shelly Pro 4PM",
	"SPEM-003CEBEU":  "Shelly Pro 3EM",
	"SPEM-002CEBEU50": "Shelly Pro EM-50",
	"SPDM-001PE01EU": "Shelly Pro Dimmer 1PM",
	"SPDM-002PE01EU": "Shelly Pro Dimmer 2PM",
	"SPSH-002PE16EU": "Shelly Pro Dual Cover PM",

	// Gen3 devices
	"S3SW-001P16EU":  "Shelly 1 Gen3",
	"S3SW-001X16EU":  "Shelly 1 Mini Gen3",
	"S3SW-002P16EU":  "Shelly 1PM Gen3",
	"S3SW-002X16EU":  "Shelly 1PM Mini Gen3",
	"S3PL-00112EU":   "Shelly Plug S Gen3",

	// BLU devices
	"SBBT-002C":      "Shelly BLU Button1",
	"SBDW-002C":      "Shelly BLU Door/Window",
	"SBMO-003Z":      "Shelly BLU Motion",
	"SBHT-003C":      "Shelly BLU H&T",
	"SBTR-001Z":      "Shelly BLU TRV",
	"SBRC-001ZB":     "Shelly BLU Remote Control",

	// Wall displays
	"SAWD-0A1XX10EU": "Shelly Wall Display",
}

// ModelDisplayName returns a human-readable product name for a Shelly model code.
// If the model is not recognized, it returns the original model code.
//
// Example:
//
//	ModelDisplayName("SHSW-PM") returns "Shelly 1PM"
//	ModelDisplayName("SNSW-102P16EU") returns "Shelly Plus 1PM"
//	ModelDisplayName("unknown") returns "unknown"
func ModelDisplayName(model string) string {
	if name, ok := modelNames[model]; ok {
		return name
	}
	return model
}

// IsKnownModel returns true if the model code is recognized.
func IsKnownModel(model string) bool {
	_, ok := modelNames[model]
	return ok
}

// GetModelCategory returns the general category of the device based on model code.
// Categories include: switch, dimmer, plug, meter, sensor, bulb, cover, gateway.
func GetModelCategory(model string) string {
	// Check prefixes for Gen2/Gen3
	switch {
	case hasPrefix(model, "SHSW-"), hasPrefix(model, "SNSW-"), hasPrefix(model, "SPSW-"), hasPrefix(model, "S3SW-"):
		return "switch"
	case hasPrefix(model, "SHDM-"), hasPrefix(model, "SNDM-"), hasPrefix(model, "SNDC-"), hasPrefix(model, "SPDM-"):
		return "dimmer"
	case hasPrefix(model, "SHPLG-"), hasPrefix(model, "SNPL-"), hasPrefix(model, "S3PL-"):
		return "plug"
	case hasPrefix(model, "SHEM"), hasPrefix(model, "SPEM-"):
		return "meter"
	case hasPrefix(model, "SHDW-"), hasPrefix(model, "SHMOS-"), hasPrefix(model, "SHHT-"), hasPrefix(model, "SHFLOOD-"):
		return "sensor"
	case hasPrefix(model, "SBDW-"), hasPrefix(model, "SBMO-"), hasPrefix(model, "SBHT-"):
		return "sensor"
	case hasPrefix(model, "SHBLB-"), hasPrefix(model, "SHBDUO-"), hasPrefix(model, "SHVIN-"), hasPrefix(model, "SHCL-"):
		return "bulb"
	case hasPrefix(model, "SHTRV-"), hasPrefix(model, "SBTR-"):
		return "trv"
	case hasPrefix(model, "SHRGBW"):
		return "rgbw"
	case hasPrefix(model, "SNGW-"), hasPrefix(model, "SBRC-"):
		return "gateway"
	case hasPrefix(model, "SAWD-"):
		return "display"
	default:
		return "unknown"
	}
}

// hasPrefix checks if s starts with prefix.
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
