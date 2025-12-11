package matter

import (
	"github.com/tj-smith47/shelly-go/types"
)

// Config represents the Matter component configuration.
type Config struct {
	types.RawFields
	Enable bool `json:"enable"`
}

// Status represents the Matter component status.
type Status struct {
	types.RawFields
	FabricsCount   int  `json:"fabrics_count"`
	Commissionable bool `json:"commissionable"`
}

// SetConfigParams represents parameters for setting Matter configuration.
type SetConfigParams struct {
	// Enable enables or disables Matter on the device.
	Enable *bool `json:"enable,omitempty"`
}

// Fabric represents a Matter fabric (network) that the device is paired with.
type Fabric struct {
	types.RawFields
	FabricID    string `json:"fabric_id,omitempty"`
	Label       string `json:"label,omitempty"`
	FabricIndex int    `json:"fabric_index,omitempty"`
	VendorID    int    `json:"vendor_id,omitempty"`
}

// CommissioningInfo contains information for commissioning a device.
type CommissioningInfo struct {
	types.RawFields
	QRCode        string `json:"qr_code,omitempty"`
	ManualCode    string `json:"manual_code,omitempty"`
	Discriminator int    `json:"discriminator,omitempty"`
	SetupPinCode  int    `json:"setup_pin_code,omitempty"`
}
