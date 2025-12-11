package zwave

import (
	"github.com/tj-smith47/shelly-go/profiles"
	"github.com/tj-smith47/shelly-go/types"
)

// NetworkTopology represents the Z-Wave network type.
type NetworkTopology string

const (
	// TopologyMesh is a standard Z-Wave mesh network where devices can
	// route through other mains-powered devices.
	TopologyMesh NetworkTopology = "mesh"

	// TopologyLongRange is a Z-Wave Long Range star topology where devices
	// communicate directly with the gateway only.
	TopologyLongRange NetworkTopology = "long_range"
)

// SecurityLevel represents the Z-Wave S2 security level.
type SecurityLevel string

const (
	// SecurityS2Authenticated provides full S2 security with DSK verification.
	// Requires the 5-digit PIN from the device label during inclusion.
	SecurityS2Authenticated SecurityLevel = "s2_authenticated"

	// SecurityS2Unauthenticated provides basic S2 encryption without DSK.
	SecurityS2Unauthenticated SecurityLevel = "s2_unauthenticated"

	// SecurityUnsecure provides no encryption (legacy mode, not recommended).
	SecurityUnsecure SecurityLevel = "unsecure"
)

// Device represents a Shelly Wave Z-Wave device.
//
// Wave devices are Z-Wave end devices that can optionally be controlled
// via their IP interface (for models with WiFi/Ethernet) using the standard
// Gen2 RPC API.
type Device struct {
	Profile  *profiles.Profile
	Topology NetworkTopology
	Security SecurityLevel
	DSK      string
	NodeID   int
	HomeID   uint32
}

// NewDevice creates a Device from a profile.
//
// This creates a device reference that can be populated with Z-Wave
// network information after inclusion.
//
// Example:
//
//	profile := profiles.Get("SNSW-001P16ZW")
//	device := zwave.NewDevice(profile)
//	device.NodeID = 5
//	device.HomeID = 0xA1B2C3D4
func NewDevice(profile *profiles.Profile) *Device {
	return &Device{
		Profile: profile,
	}
}

// Model returns the device model identifier.
func (d *Device) Model() string {
	if d.Profile == nil {
		return ""
	}
	return d.Profile.Model
}

// Name returns the device display name.
func (d *Device) Name() string {
	if d.Profile == nil {
		return ""
	}
	return d.Profile.Name
}

// Generation returns the device generation.
func (d *Device) Generation() types.Generation {
	if d.Profile == nil {
		return types.GenUnknown
	}
	return d.Profile.Generation
}

// IsZWave returns true if the device supports Z-Wave protocol.
func (d *Device) IsZWave() bool {
	if d.Profile == nil {
		return false
	}
	return d.Profile.Protocols.ZWave
}

// HasEthernet returns true if the device has Ethernet connectivity.
func (d *Device) HasEthernet() bool {
	if d.Profile == nil {
		return false
	}
	return d.Profile.Protocols.Ethernet
}

// HasWiFi returns true if the device has WiFi connectivity.
//
// Note: Most Gen2+ devices have WiFi by default. Wave devices typically
// have Z-Wave only, while Wave Pro devices have Ethernet but not WiFi.
// WiFi connectivity is assumed if the device has HTTP protocol support
// and is not Z-Wave only.
func (d *Device) HasWiFi() bool {
	if d.Profile == nil {
		return false
	}
	// Wave devices are Z-Wave only unless they have Ethernet (Wave Pro)
	// Standard WiFi devices would have HTTP but not be Z-Wave only
	return d.Profile.Protocols.HTTP && !d.Profile.Protocols.ZWave
}

// HasIPAccess returns true if the device can be controlled via IP (WiFi or Ethernet).
func (d *Device) HasIPAccess() bool {
	return d.HasWiFi() || d.HasEthernet()
}

// SupportsLongRange returns true if the device supports Z-Wave Long Range.
//
// Note: All modern Shelly Wave devices support ZWLR, but the actual capability
// depends on the gateway and may require specific firmware versions.
func (d *Device) SupportsLongRange() bool {
	return d.IsZWave()
}

// IsPro returns true if this is a Wave Pro series device.
func (d *Device) IsPro() bool {
	if d.Profile == nil {
		return false
	}
	return d.Profile.Series == profiles.SeriesWavePro
}
