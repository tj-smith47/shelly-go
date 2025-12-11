package backup

import (
	"encoding/json"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// BackupVersion is the current backup format version.
const BackupVersion = 1

// Backup represents a complete device backup.
type Backup struct {
	CreatedAt  time.Time                  `json:"created_at"`
	KVS        map[string]json.RawMessage `json:"kvs,omitempty"`
	DeviceInfo *DeviceInfo                `json:"device_info,omitempty"`
	Components map[string]json.RawMessage `json:"components,omitempty"`
	Auth       *AuthInfo                  `json:"auth,omitempty"`
	BLE        json.RawMessage            `json:"ble,omitempty"`
	MQTT       json.RawMessage            `json:"mqtt,omitempty"`
	Webhooks   json.RawMessage            `json:"webhooks,omitempty"`
	Schedules  json.RawMessage            `json:"schedules,omitempty"`
	Scripts    []*Script                  `json:"scripts,omitempty"`
	Cloud      json.RawMessage            `json:"cloud,omitempty"`
	WiFi       json.RawMessage            `json:"wifi,omitempty"`
	Config     json.RawMessage            `json:"config,omitempty"`
	Version    int                        `json:"version"`
}

// DeviceInfo contains information about the device.
type DeviceInfo struct {
	types.RawFields
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Model      string `json:"model,omitempty"`
	App        string `json:"app,omitempty"`
	Version    string `json:"ver,omitempty"`
	MAC        string `json:"mac,omitempty"`
	Generation int    `json:"gen,omitempty"`
}

// Script represents a script configuration.
type Script struct {
	Name   string `json:"name,omitempty"`
	Code   string `json:"code,omitempty"`
	ID     int    `json:"id"`
	Enable bool   `json:"enable"`
}

// AuthInfo contains authentication information.
// Note: Passwords are never exported for security.
type AuthInfo struct {
	User   string `json:"user,omitempty"`
	Enable bool   `json:"enable"`
}

// ExportOptions controls what is included in the backup.
type ExportOptions struct {
	// IncludeWiFi includes WiFi configuration including credentials.
	// Default: false (security consideration)
	IncludeWiFi bool

	// IncludeCloud includes Shelly Cloud configuration.
	// Default: true
	IncludeCloud bool

	// IncludeAuth includes authentication configuration.
	// Note: Passwords are never exported.
	// Default: true
	IncludeAuth bool

	// IncludeBLE includes BLE configuration.
	// Default: true
	IncludeBLE bool

	// IncludeMQTT includes MQTT configuration.
	// Default: true
	IncludeMQTT bool

	// IncludeWebhooks includes webhook configurations.
	// Default: true
	IncludeWebhooks bool

	// IncludeSchedules includes schedule configurations.
	// Default: true
	IncludeSchedules bool

	// IncludeScripts includes script code and configuration.
	// Default: true
	IncludeScripts bool

	// IncludeKVS includes key-value storage data.
	// Default: true
	IncludeKVS bool

	// IncludeComponents includes component-specific configurations.
	// Default: true
	IncludeComponents bool
}

// DefaultExportOptions returns the default export options.
func DefaultExportOptions() *ExportOptions {
	return &ExportOptions{
		IncludeWiFi:       false, // Security: don't export WiFi by default
		IncludeCloud:      true,
		IncludeAuth:       true,
		IncludeBLE:        true,
		IncludeMQTT:       true,
		IncludeWebhooks:   true,
		IncludeSchedules:  true,
		IncludeScripts:    true,
		IncludeKVS:        true,
		IncludeComponents: true,
	}
}

// RestoreOptions controls what is restored from the backup.
type RestoreOptions struct {
	// RestoreWiFi restores WiFi configuration.
	// Default: false (requires explicit opt-in)
	RestoreWiFi bool

	// RestoreCloud restores Shelly Cloud configuration.
	// Default: true
	RestoreCloud bool

	// RestoreAuth restores authentication configuration.
	// Note: Passwords must be set separately after restore.
	// Default: false
	RestoreAuth bool

	// RestoreBLE restores BLE configuration.
	// Default: true
	RestoreBLE bool

	// RestoreMQTT restores MQTT configuration.
	// Default: true
	RestoreMQTT bool

	// RestoreWebhooks restores webhook configurations.
	// Default: true
	RestoreWebhooks bool

	// RestoreSchedules restores schedule configurations.
	// Default: true
	RestoreSchedules bool

	// RestoreScripts restores script code and configuration.
	// Default: true
	RestoreScripts bool

	// RestoreKVS restores key-value storage data.
	// Default: true
	RestoreKVS bool

	// RestoreComponents restores component-specific configurations.
	// Default: true
	RestoreComponents bool

	// DryRun validates the backup without applying changes.
	// Default: false
	DryRun bool

	// StopScripts stops running scripts before restore.
	// Default: true
	StopScripts bool
}

// DefaultRestoreOptions returns the default restore options.
func DefaultRestoreOptions() *RestoreOptions {
	return &RestoreOptions{
		RestoreWiFi:       false, // Requires explicit opt-in
		RestoreCloud:      true,
		RestoreAuth:       false, // Security: don't restore auth by default
		RestoreBLE:        true,
		RestoreMQTT:       true,
		RestoreWebhooks:   true,
		RestoreSchedules:  true,
		RestoreScripts:    true,
		RestoreKVS:        true,
		RestoreComponents: true,
		DryRun:            false,
		StopScripts:       true,
	}
}

// RestoreResult contains the result of a restore operation.
type RestoreResult struct {
	Warnings        []string
	Errors          []error
	Success         bool
	RestartRequired bool
}
