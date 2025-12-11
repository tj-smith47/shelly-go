package firmware

import (
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Device represents a device that can be managed by the firmware package.
// This interface is used for batch operations.
type Device interface {
	// Address returns the device address (IP or hostname).
	Address() string

	// Client returns the RPC client for the device.
	Client() *rpc.Client
}

// UpdateInfo contains information about available firmware updates.
type UpdateInfo struct {
	types.RawFields
	Current       string `json:"current,omitempty"`
	Available     string `json:"available,omitempty"`
	Beta          string `json:"beta,omitempty"`
	URL           string `json:"url,omitempty"`
	HasUpdateFlag bool   `json:"has_update,omitempty"`
}

// HasUpdate returns true if a firmware update is available.
func (u *UpdateInfo) HasUpdate() bool {
	if u.HasUpdateFlag {
		return true
	}
	return u.Available != "" && u.Available != u.Current
}

// HasBeta returns true if a beta firmware update is available.
func (u *UpdateInfo) HasBeta() bool {
	return u.Beta != "" && u.Beta != u.Current
}

// UpdateOptions contains options for firmware updates.
type UpdateOptions struct {
	// Stage specifies the update channel: "stable" or "beta".
	// Default: "stable"
	Stage string `json:"stage,omitempty"`

	// URL specifies a custom firmware URL to use instead of the official update.
	// When set, Stage is ignored.
	URL string `json:"url,omitempty"`
}

// RollbackStatus contains information about firmware rollback availability.
type RollbackStatus struct {
	types.RawFields
	PreviousVersion string `json:"previous_version,omitempty"`
	CanRollback     bool   `json:"can_rollback"`
}

// UpdateStatus contains the current status of a firmware update.
type UpdateStatus struct {
	types.RawFields
	Status     string `json:"status,omitempty"`
	NewVersion string `json:"new_version,omitempty"`
	Progress   int    `json:"progress,omitempty"`
	HasUpdate  bool   `json:"has_update,omitempty"`
}

// CheckResult contains the result of checking for updates on a device.
type CheckResult struct {
	Device  Device
	Error   error
	Info    *UpdateInfo
	Address string
}

// UpdateResult contains the result of updating a device.
type UpdateResult struct {
	Device  Device
	Error   error
	Address string
	Success bool
}

// DeviceVersion contains version information for a device.
type DeviceVersion struct {
	types.RawFields
	FirmwareVersion string `json:"ver,omitempty"`
	App             string `json:"app,omitempty"`
	Model           string `json:"model,omitempty"`
	Generation      int    `json:"gen,omitempty"`
}
