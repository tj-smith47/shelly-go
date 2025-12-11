package backup

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/rpc"
)

// Common errors.
var (
	// ErrInvalidBackup indicates the backup data is invalid.
	ErrInvalidBackup = errors.New("invalid backup data")

	// ErrVersionMismatch indicates the backup version is not supported.
	ErrVersionMismatch = errors.New("backup version not supported")

	// ErrDeviceMismatch indicates the backup is for a different device model.
	ErrDeviceMismatch = errors.New("backup device model mismatch")
)

// Manager handles backup and restore operations.
type Manager struct {
	client *rpc.Client
}

// New creates a new backup Manager with the given RPC client.
func New(client *rpc.Client) *Manager {
	return &Manager{client: client}
}

// Export creates a backup of the device configuration.
func (m *Manager) Export(ctx context.Context, opts *ExportOptions) ([]byte, error) {
	if opts == nil {
		opts = DefaultExportOptions()
	}

	backup := &Backup{
		Version:   BackupVersion,
		CreatedAt: time.Now().UTC(),
	}

	// Get device info
	info, err := m.getDeviceInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}
	backup.DeviceInfo = info

	// Get system config
	config, err := m.getConfig(ctx, "Shelly.GetConfig")
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	backup.Config = config

	// Export optional configs using table-driven approach
	m.exportOptionalConfigs(ctx, opts, backup)

	return json.MarshalIndent(backup, "", "  ")
}

// exportOptionalConfigs exports optional configuration sections based on options.
func (m *Manager) exportOptionalConfigs(ctx context.Context, opts *ExportOptions, backup *Backup) {
	// Table of optional configs to export
	type configExport struct {
		setter  func(json.RawMessage)
		method  string
		include bool
	}
	configExports := []configExport{
		{setter: func(v json.RawMessage) { backup.WiFi = v }, method: "WiFi.GetConfig", include: opts.IncludeWiFi},
		{setter: func(v json.RawMessage) { backup.Cloud = v }, method: "Cloud.GetConfig", include: opts.IncludeCloud},
		{setter: func(v json.RawMessage) { backup.BLE = v }, method: "BLE.GetConfig", include: opts.IncludeBLE},
		{setter: func(v json.RawMessage) { backup.MQTT = v }, method: "MQTT.GetConfig", include: opts.IncludeMQTT},
	}

	for _, ce := range configExports {
		if ce.include {
			if data, err := m.getConfig(ctx, ce.method); err == nil {
				ce.setter(data)
			}
		}
	}

	// Export webhooks if requested
	if opts.IncludeWebhooks {
		if webhooks, err := m.listWebhooks(ctx); err == nil {
			backup.Webhooks = webhooks
		}
	}

	// Export schedules if requested
	if opts.IncludeSchedules {
		if schedules, err := m.listSchedules(ctx); err == nil {
			backup.Schedules = schedules
		}
	}

	// Export scripts if requested
	if opts.IncludeScripts {
		if scripts, err := m.listScripts(ctx); err == nil {
			backup.Scripts = scripts
		}
	}

	// Export KVS if requested
	if opts.IncludeKVS {
		if kvs, err := m.listKVS(ctx); err == nil {
			backup.KVS = kvs
		}
	}

	// Export auth info if requested
	if opts.IncludeAuth {
		if auth, err := m.getAuthInfo(ctx); err == nil {
			backup.Auth = auth
		}
	}
}

// Restore restores configuration from a backup.
func (m *Manager) Restore(ctx context.Context, data []byte, opts *RestoreOptions) (*RestoreResult, error) {
	if opts == nil {
		opts = DefaultRestoreOptions()
	}

	result := &RestoreResult{
		Warnings: []string{},
		Errors:   []error{},
	}

	// Parse backup
	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBackup, err)
	}

	// Validate version
	if backup.Version > BackupVersion {
		return nil, fmt.Errorf("%w: backup version %d, supported up to %d",
			ErrVersionMismatch, backup.Version, BackupVersion)
	}

	// If dry run, just validate and return
	if opts.DryRun {
		result.Success = true
		return result, nil
	}

	// Stop scripts before restore if requested
	if opts.StopScripts && len(backup.Scripts) > 0 {
		m.stopAllScripts(ctx)
	}

	// Restore optional configs using table-driven approach
	m.restoreOptionalConfigs(ctx, opts, &backup, result)

	// Restore complex items
	m.restoreComplexItems(ctx, opts, &backup, result)

	result.Success = len(result.Errors) == 0
	return result, nil
}

// restoreOptionalConfigs restores optional configuration sections based on options.
func (m *Manager) restoreOptionalConfigs(
	ctx context.Context,
	opts *RestoreOptions,
	backup *Backup,
	result *RestoreResult,
) {
	// Table of optional configs to restore
	type configRestore struct {
		method     string
		name       string
		data       json.RawMessage
		restore    bool
		setRestart bool
	}
	configRestores := []configRestore{
		{data: backup.WiFi, method: "WiFi.SetConfig", name: "WiFi", restore: opts.RestoreWiFi, setRestart: true},
		{data: backup.Cloud, method: "Cloud.SetConfig", name: "cloud", restore: opts.RestoreCloud, setRestart: false},
		{data: backup.BLE, method: "BLE.SetConfig", name: "ble", restore: opts.RestoreBLE, setRestart: false},
		{data: backup.MQTT, method: "MQTT.SetConfig", name: "mqtt", restore: opts.RestoreMQTT, setRestart: false},
	}

	for _, cr := range configRestores {
		if cr.restore && cr.data != nil {
			if err := m.setConfig(ctx, cr.method, cr.data); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", cr.name, err))
			} else if cr.setRestart {
				result.RestartRequired = true
			}
		}
	}
}

// restoreComplexItems restores schedules, webhooks, scripts, and KVS.
func (m *Manager) restoreComplexItems(
	ctx context.Context,
	opts *RestoreOptions,
	backup *Backup,
	result *RestoreResult,
) {
	// Restore schedules
	if opts.RestoreSchedules && backup.Schedules != nil {
		if err := m.restoreSchedules(ctx, backup.Schedules); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("schedules: %w", err))
		}
	}

	// Restore webhooks
	if opts.RestoreWebhooks && backup.Webhooks != nil {
		if err := m.restoreWebhooks(ctx, backup.Webhooks); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("webhooks: %w", err))
		}
	}

	// Restore scripts
	if opts.RestoreScripts && len(backup.Scripts) > 0 {
		m.restoreScripts(ctx, backup.Scripts)
	}

	// Restore KVS
	if opts.RestoreKVS && len(backup.KVS) > 0 {
		m.restoreKVS(ctx, backup.KVS)
	}
}

// ParseBackup parses backup data without restoring.
func (m *Manager) ParseBackup(data []byte) (*Backup, error) {
	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBackup, err)
	}
	return &backup, nil
}

// getDeviceInfo retrieves device information.
func (m *Manager) getDeviceInfo(ctx context.Context) (*DeviceInfo, error) {
	result, err := m.client.Call(ctx, "Shelly.GetDeviceInfo", nil)
	if err != nil {
		return nil, err
	}

	var info DeviceInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// getConfig retrieves configuration using the specified method.
func (m *Manager) getConfig(ctx context.Context, method string) (json.RawMessage, error) {
	result, err := m.client.Call(ctx, method, nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// setConfig sets configuration using the specified method.
func (m *Manager) setConfig(ctx context.Context, method string, config json.RawMessage) error {
	params := map[string]any{
		"config": config,
	}
	_, err := m.client.Call(ctx, method, params)
	return err
}

// listWebhooks retrieves all webhooks.
func (m *Manager) listWebhooks(ctx context.Context) (json.RawMessage, error) {
	result, err := m.client.Call(ctx, "Webhook.List", nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// listSchedules retrieves all schedules.
func (m *Manager) listSchedules(ctx context.Context) (json.RawMessage, error) {
	result, err := m.client.Call(ctx, "Schedule.List", nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// listScripts retrieves all scripts with their code.
func (m *Manager) listScripts(ctx context.Context) ([]*Script, error) {
	// Get script list
	result, err := m.client.Call(ctx, "Script.List", nil)
	if err != nil {
		return nil, err
	}

	var list struct {
		Scripts []struct {
			Name   string `json:"name"`
			ID     int    `json:"id"`
			Enable bool   `json:"enable"`
		} `json:"scripts"`
	}
	if err := json.Unmarshal(result, &list); err != nil {
		return nil, err
	}

	// Get code for each script
	scripts := make([]*Script, 0, len(list.Scripts))
	for _, s := range list.Scripts {
		script := &Script{
			ID:     s.ID,
			Name:   s.Name,
			Enable: s.Enable,
		}

		// Get script code
		codeResult, err := m.client.Call(ctx, "Script.GetCode", map[string]any{
			"id": s.ID,
		})
		if err == nil {
			var codeResp struct {
				Data string `json:"data"`
			}
			if json.Unmarshal(codeResult, &codeResp) == nil {
				script.Code = codeResp.Data
			}
		}

		scripts = append(scripts, script)
	}

	return scripts, nil
}

// listKVS retrieves all KVS entries.
func (m *Manager) listKVS(ctx context.Context) (map[string]json.RawMessage, error) {
	// Get KVS list
	result, err := m.client.Call(ctx, "KVS.List", nil)
	if err != nil {
		return nil, err
	}

	var list struct {
		Keys []string `json:"keys"`
	}
	if err := json.Unmarshal(result, &list); err != nil {
		return nil, err
	}

	if len(list.Keys) == 0 {
		return nil, nil
	}

	// Get all values
	kvs := make(map[string]json.RawMessage)
	for _, key := range list.Keys {
		valueResult, err := m.client.Call(ctx, "KVS.Get", map[string]any{
			"key": key,
		})
		if err == nil {
			kvs[key] = valueResult
		}
	}

	return kvs, nil
}

// getAuthInfo retrieves authentication info.
func (m *Manager) getAuthInfo(ctx context.Context) (*AuthInfo, error) {
	result, err := m.client.Call(ctx, "Shelly.GetDeviceInfo", nil)
	if err != nil {
		return nil, err
	}

	var info struct {
		AuthUser string `json:"auth_user"`
		Auth     bool   `json:"auth_en"`
	}
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, err
	}

	return &AuthInfo{
		Enable: info.Auth,
		User:   info.AuthUser,
	}, nil
}

// stopAllScripts stops all running scripts.
func (m *Manager) stopAllScripts(ctx context.Context) {
	result, err := m.client.Call(ctx, "Script.List", nil)
	if err != nil {
		return
	}

	var list struct {
		Scripts []struct {
			ID      int  `json:"id"`
			Running bool `json:"running"`
		} `json:"scripts"`
	}
	if json.Unmarshal(result, &list) != nil {
		return
	}

	for _, s := range list.Scripts {
		if s.Running {
			//nolint:errcheck // Best-effort stop, script may not exist or already be stopped
			m.client.Call(ctx, "Script.Stop", map[string]any{"id": s.ID})
		}
	}
}

// restoreItems is a generic helper for restoring schedules, webhooks, etc.
// It handles the common pattern of deleting all items, parsing JSON array, and recreating them.
func (m *Manager) restoreItems(
	ctx context.Context,
	data json.RawMessage,
	itemsKey, deleteMethod, createMethod string,
) error {
	// Delete existing items (best-effort, may fail if none exist)
	//nolint:errcheck // Intentionally ignoring - delete may fail if no items exist
	m.client.Call(ctx, deleteMethod, nil)

	// Parse container with dynamic key
	var rawContainer map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawContainer); err != nil {
		return err
	}

	itemsJSON, ok := rawContainer[itemsKey]
	if !ok {
		return nil
	}

	var items []json.RawMessage
	if err := json.Unmarshal(itemsJSON, &items); err != nil {
		return err
	}

	for _, item := range items {
		var itemMap map[string]any
		if json.Unmarshal(item, &itemMap) == nil {
			// Remove ID to create new
			delete(itemMap, "id")
			//nolint:errcheck // Best-effort create, continue with other items on failure
			m.client.Call(ctx, createMethod, itemMap)
		}
	}

	return nil
}

// restoreSchedules restores schedules from backup.
func (m *Manager) restoreSchedules(ctx context.Context, data json.RawMessage) error {
	return m.restoreItems(ctx, data, "jobs", "Schedule.DeleteAll", "Schedule.Create")
}

// restoreWebhooks restores webhooks from backup.
func (m *Manager) restoreWebhooks(ctx context.Context, data json.RawMessage) error {
	return m.restoreItems(ctx, data, "hooks", "Webhook.DeleteAll", "Webhook.Create")
}

// restoreScripts restores scripts from backup.
func (m *Manager) restoreScripts(ctx context.Context, scripts []*Script) {
	for _, script := range scripts {
		// Create script
		createResult, err := m.client.Call(ctx, "Script.Create", map[string]any{
			"name": script.Name,
		})
		if err != nil {
			continue
		}

		var created struct {
			ID int `json:"id"`
		}
		if json.Unmarshal(createResult, &created) != nil {
			continue
		}

		// Put code if available
		if script.Code != "" {
			//nolint:errcheck // Best-effort restore, continue with other scripts on failure
			m.client.Call(ctx, "Script.PutCode", map[string]any{
				"id":   created.ID,
				"code": script.Code,
			})
		}

		// Enable if was enabled
		if script.Enable {
			//nolint:errcheck // Best-effort restore, continue with other scripts on failure
			m.client.Call(ctx, "Script.SetConfig", map[string]any{
				"id": created.ID,
				"config": map[string]any{
					"enable": true,
				},
			})
		}
	}
}

// restoreKVS restores KVS data from backup.
func (m *Manager) restoreKVS(ctx context.Context, kvs map[string]json.RawMessage) {
	for key, value := range kvs {
		var item struct {
			Value any `json:"value"`
		}
		if json.Unmarshal(value, &item) == nil {
			//nolint:errcheck // Best-effort restore, continue with other keys on failure
			m.client.Call(ctx, "KVS.Set", map[string]any{
				"key":   key,
				"value": item.Value,
			})
		}
	}
}

// Migration errors.
var (
	// ErrMigrationInProgress indicates a migration is already running.
	ErrMigrationInProgress = errors.New("migration already in progress")

	// ErrMigrationFailed indicates the migration failed.
	ErrMigrationFailed = errors.New("migration failed")

	// ErrSourceDeviceOffline indicates the source device is offline.
	ErrSourceDeviceOffline = errors.New("source device is offline")

	// ErrTargetDeviceOffline indicates the target device is offline.
	ErrTargetDeviceOffline = errors.New("target device is offline")

	// ErrIncompatibleDevices indicates the devices are not compatible for migration.
	ErrIncompatibleDevices = errors.New("incompatible device types for migration")

	// ErrEncryptionFailed indicates encryption failed.
	ErrEncryptionFailed = errors.New("encryption failed")

	// ErrDecryptionFailed indicates decryption failed.
	ErrDecryptionFailed = errors.New("decryption failed")
)

// Migrator handles device-to-device migration operations.
type Migrator struct {
	SourceClient              *rpc.Client
	TargetClient              *rpc.Client
	OnProgress                func(step string, progress float64)
	mu                        sync.Mutex
	AllowDifferentModels      bool
	AllowDifferentGenerations bool
	inProgress                bool
}

// NewMigrator creates a new device migrator.
func NewMigrator(source, target *rpc.Client) *Migrator {
	return &Migrator{
		SourceClient: source,
		TargetClient: target,
	}
}

// MigrationOptions controls the migration process.
type MigrationOptions struct {
	// IncludeWiFi migrates WiFi configuration.
	IncludeWiFi bool

	// IncludeCloud migrates Cloud configuration.
	IncludeCloud bool

	// IncludeMQTT migrates MQTT configuration.
	IncludeMQTT bool

	// IncludeBLE migrates BLE configuration.
	IncludeBLE bool

	// IncludeSchedules migrates schedules.
	IncludeSchedules bool

	// IncludeWebhooks migrates webhooks.
	IncludeWebhooks bool

	// IncludeScripts migrates scripts.
	IncludeScripts bool

	// IncludeKVS migrates KVS data.
	IncludeKVS bool

	// RebootAfter reboots the target device after migration.
	RebootAfter bool

	// DryRun simulates the migration without making changes.
	DryRun bool
}

// DefaultMigrationOptions returns the default migration options.
func DefaultMigrationOptions() *MigrationOptions {
	return &MigrationOptions{
		IncludeWiFi:      false, // Requires explicit opt-in
		IncludeCloud:     true,
		IncludeMQTT:      true,
		IncludeBLE:       true,
		IncludeSchedules: true,
		IncludeWebhooks:  true,
		IncludeScripts:   true,
		IncludeKVS:       true,
		RebootAfter:      true,
		DryRun:           false,
	}
}

// MigrationResult contains the result of a migration operation.
type MigrationResult struct {
	StartedAt          time.Time
	CompletedAt        time.Time
	SourceDevice       *DeviceInfo
	TargetDevice       *DeviceInfo
	ComponentsMigrated []string
	Warnings           []string
	Errors             []error
	Success            bool
	RestartRequired    bool
}

// Duration returns the migration duration.
func (r *MigrationResult) Duration() time.Duration {
	return r.CompletedAt.Sub(r.StartedAt)
}

// Migrate performs a device-to-device migration.
//
//nolint:gocyclo,cyclop,funlen // Migration orchestration inherently requires multiple sequential steps
func (m *Migrator) Migrate(ctx context.Context, opts *MigrationOptions) (*MigrationResult, error) {
	m.mu.Lock()
	if m.inProgress {
		m.mu.Unlock()
		return nil, ErrMigrationInProgress
	}
	m.inProgress = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.inProgress = false
		m.mu.Unlock()
	}()

	if opts == nil {
		opts = DefaultMigrationOptions()
	}

	result := &MigrationResult{
		StartedAt:          time.Now(),
		ComponentsMigrated: []string{},
		Warnings:           []string{},
		Errors:             []error{},
	}

	// Get source device info
	m.reportProgress("Getting source device info", 0.05)
	srcMgr := New(m.SourceClient)
	srcInfo, err := srcMgr.getDeviceInfo(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("source device: %w", err))
		result.CompletedAt = time.Now()
		return result, ErrSourceDeviceOffline
	}
	result.SourceDevice = srcInfo

	// Get target device info
	m.reportProgress("Getting target device info", 0.10)
	tgtMgr := New(m.TargetClient)
	tgtInfo, err := tgtMgr.getDeviceInfo(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("target device: %w", err))
		result.CompletedAt = time.Now()
		return result, ErrTargetDeviceOffline
	}
	result.TargetDevice = tgtInfo

	// Check compatibility
	m.reportProgress("Checking compatibility", 0.15)
	if !m.AllowDifferentModels && srcInfo.Model != tgtInfo.Model {
		result.Errors = append(result.Errors, fmt.Errorf("model mismatch: %s vs %s", srcInfo.Model, tgtInfo.Model))
		result.CompletedAt = time.Now()
		return result, ErrIncompatibleDevices
	}

	if !m.AllowDifferentGenerations && srcInfo.Generation != tgtInfo.Generation {
		errMsg := fmt.Errorf("generation mismatch: %d vs %d", srcInfo.Generation, tgtInfo.Generation)
		result.Errors = append(result.Errors, errMsg)
		result.CompletedAt = time.Now()
		return result, ErrIncompatibleDevices
	}

	// If dry run, stop here
	if opts.DryRun {
		result.Success = true
		result.CompletedAt = time.Now()
		return result, nil
	}

	// Export from source
	m.reportProgress("Exporting source configuration", 0.25)
	exportOpts := &ExportOptions{
		IncludeWiFi:       opts.IncludeWiFi,
		IncludeCloud:      opts.IncludeCloud,
		IncludeMQTT:       opts.IncludeMQTT,
		IncludeBLE:        opts.IncludeBLE,
		IncludeSchedules:  opts.IncludeSchedules,
		IncludeWebhooks:   opts.IncludeWebhooks,
		IncludeScripts:    opts.IncludeScripts,
		IncludeKVS:        opts.IncludeKVS,
		IncludeAuth:       false, // Never migrate auth
		IncludeComponents: true,
	}

	backupData, err := srcMgr.Export(ctx, exportOpts)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("export failed: %w", err))
		result.CompletedAt = time.Now()
		return result, ErrMigrationFailed
	}

	// Restore to target
	m.reportProgress("Importing to target device", 0.50)
	restoreOpts := &RestoreOptions{
		RestoreWiFi:       opts.IncludeWiFi,
		RestoreCloud:      opts.IncludeCloud,
		RestoreMQTT:       opts.IncludeMQTT,
		RestoreBLE:        opts.IncludeBLE,
		RestoreSchedules:  opts.IncludeSchedules,
		RestoreWebhooks:   opts.IncludeWebhooks,
		RestoreScripts:    opts.IncludeScripts,
		RestoreKVS:        opts.IncludeKVS,
		RestoreAuth:       false, // Never restore auth automatically
		RestoreComponents: true,
		DryRun:            false,
		StopScripts:       true,
	}

	restoreResult, err := tgtMgr.Restore(ctx, backupData, restoreOpts)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("restore failed: %w", err))
		result.CompletedAt = time.Now()
		return result, ErrMigrationFailed
	}

	// Collect results
	result.Warnings = append(result.Warnings, restoreResult.Warnings...)
	result.Errors = append(result.Errors, restoreResult.Errors...)
	result.RestartRequired = restoreResult.RestartRequired

	// Track what was migrated using table-driven approach
	componentFlags := []struct {
		name    string
		include bool
	}{
		{name: "wifi", include: opts.IncludeWiFi},
		{name: "cloud", include: opts.IncludeCloud},
		{name: "mqtt", include: opts.IncludeMQTT},
		{name: "ble", include: opts.IncludeBLE},
		{name: "schedules", include: opts.IncludeSchedules},
		{name: "webhooks", include: opts.IncludeWebhooks},
		{name: "scripts", include: opts.IncludeScripts},
		{name: "kvs", include: opts.IncludeKVS},
	}
	for _, cf := range componentFlags {
		if cf.include {
			result.ComponentsMigrated = append(result.ComponentsMigrated, cf.name)
		}
	}

	// Reboot target if requested
	if opts.RebootAfter && result.RestartRequired {
		m.reportProgress("Rebooting target device", 0.90)
		if _, err := m.TargetClient.Call(ctx, "Shelly.Reboot", nil); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("reboot failed: %v", err))
		}
	}

	m.reportProgress("Migration complete", 1.0)
	result.Success = len(result.Errors) == 0
	result.CompletedAt = time.Now()
	return result, nil
}

// IsInProgress returns true if a migration is in progress.
func (m *Migrator) IsInProgress() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.inProgress
}

// reportProgress calls the progress callback if set.
func (m *Migrator) reportProgress(step string, progress float64) {
	if m.OnProgress != nil {
		m.OnProgress(step, progress)
	}
}

// ValidateMigration checks if migration between two devices is possible.
func (m *Migrator) ValidateMigration(ctx context.Context) (*MigrationValidation, error) {
	validation := &MigrationValidation{
		Warnings: []string{},
		Errors:   []string{},
	}

	// Get source device info
	srcMgr := New(m.SourceClient)
	srcInfo, err := srcMgr.getDeviceInfo(ctx)
	if err != nil {
		validation.Errors = append(validation.Errors, "Source device unreachable")
		return validation, nil
	}
	validation.SourceDevice = srcInfo

	// Get target device info
	tgtMgr := New(m.TargetClient)
	tgtInfo, err := tgtMgr.getDeviceInfo(ctx)
	if err != nil {
		validation.Errors = append(validation.Errors, "Target device unreachable")
		return validation, nil
	}
	validation.TargetDevice = tgtInfo

	// Check model compatibility
	if srcInfo.Model != tgtInfo.Model {
		if m.AllowDifferentModels {
			validation.Warnings = append(validation.Warnings,
				fmt.Sprintf("Different models: %s -> %s (allowed)", srcInfo.Model, tgtInfo.Model))
		} else {
			validation.Errors = append(validation.Errors,
				fmt.Sprintf("Model mismatch: %s -> %s", srcInfo.Model, tgtInfo.Model))
		}
	}

	// Check generation compatibility
	if srcInfo.Generation != tgtInfo.Generation {
		if m.AllowDifferentGenerations {
			validation.Warnings = append(validation.Warnings,
				fmt.Sprintf("Different generations: %d -> %d (allowed)", srcInfo.Generation, tgtInfo.Generation))
		} else {
			validation.Errors = append(validation.Errors,
				fmt.Sprintf("Generation mismatch: %d -> %d", srcInfo.Generation, tgtInfo.Generation))
		}
	}

	validation.Valid = len(validation.Errors) == 0
	return validation, nil
}

// MigrationValidation contains the result of migration validation.
type MigrationValidation struct {
	SourceDevice *DeviceInfo
	TargetDevice *DeviceInfo
	Warnings     []string
	Errors       []string
	Valid        bool
}

// Encryptor handles backup encryption and decryption.
type Encryptor struct {
	// key is the derived encryption key.
	key []byte
}

// NewEncryptor creates a new encryptor with the given password.
// The password is used to derive an AES-256 encryption key.
func NewEncryptor(password string) *Encryptor {
	// Derive a 32-byte key from password using SHA-256
	hash := sha256.Sum256([]byte(password))
	return &Encryptor{key: hash[:]}
}

// Encrypt encrypts backup data.
func (e *Encryptor) Encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt decrypts backup data.
func (e *Encryptor) Decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrDecryptionFailed)
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// EncryptToBase64 encrypts data and returns base64-encoded string.
func (e *Encryptor) EncryptToBase64(data []byte) (string, error) {
	encrypted, err := e.Encrypt(data)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptFromBase64 decrypts base64-encoded encrypted data.
func (e *Encryptor) DecryptFromBase64(encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64: %v", ErrDecryptionFailed, err)
	}
	return e.Decrypt(data)
}

// EncryptedBackup represents an encrypted backup.
type EncryptedBackup struct {
	CreatedAt     time.Time `json:"created_at"`
	DeviceModel   string    `json:"device_model,omitempty"`
	DeviceID      string    `json:"device_id,omitempty"`
	EncryptedData string    `json:"encrypted_data"`
	Version       int       `json:"version"`
}

// EncryptedBackupVersion is the current encrypted backup format version.
const EncryptedBackupVersion = 1

// ExportEncrypted creates an encrypted backup of the device configuration.
func (m *Manager) ExportEncrypted(ctx context.Context, password string, opts *ExportOptions) (*EncryptedBackup, error) {
	// Get regular backup
	data, err := m.Export(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Parse to get device info for the wrapper
	var backup Backup
	if unmarshalErr := json.Unmarshal(data, &backup); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	// Encrypt
	enc := NewEncryptor(password)
	var encryptedData string
	encryptedData, err = enc.EncryptToBase64(data)
	if err != nil {
		return nil, err
	}

	result := &EncryptedBackup{
		Version:       EncryptedBackupVersion,
		CreatedAt:     time.Now().UTC(),
		EncryptedData: encryptedData,
	}

	if backup.DeviceInfo != nil {
		result.DeviceModel = backup.DeviceInfo.Model
		result.DeviceID = backup.DeviceInfo.ID
	}

	return result, nil
}

// RestoreEncrypted restores configuration from an encrypted backup.
func (m *Manager) RestoreEncrypted(
	ctx context.Context, encBackup *EncryptedBackup, password string, opts *RestoreOptions,
) (*RestoreResult, error) {
	// Decrypt
	enc := NewEncryptor(password)
	data, err := enc.DecryptFromBase64(encBackup.EncryptedData)
	if err != nil {
		return nil, err
	}

	// Restore
	return m.Restore(ctx, data, opts)
}

// SecureCredentials represents credentials that can be encrypted.
type SecureCredentials struct {
	Custom       map[string]string `json:"custom,omitempty"`
	WiFiSSID     string            `json:"wifi_ssid,omitempty"`
	WiFiPassword string            `json:"wifi_pass,omitempty"`
	MQTTUser     string            `json:"mqtt_user,omitempty"`
	MQTTPassword string            `json:"mqtt_pass,omitempty"`
	AuthUser     string            `json:"auth_user,omitempty"`
	AuthPassword string            `json:"auth_pass,omitempty"`
	CloudToken   string            `json:"cloud_token,omitempty"`
}

// CredentialStore provides secure credential storage.
type CredentialStore struct {
	encryptor *Encryptor
	creds     map[string]*SecureCredentials
	mu        sync.RWMutex
}

// NewCredentialStore creates a new credential store with the given encryption password.
func NewCredentialStore(password string) *CredentialStore {
	return &CredentialStore{
		encryptor: NewEncryptor(password),
		creds:     make(map[string]*SecureCredentials),
	}
}

// Store stores credentials for a device.
func (s *CredentialStore) Store(deviceID string, creds *SecureCredentials) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creds[deviceID] = creds
}

// Get retrieves credentials for a device.
func (s *CredentialStore) Get(deviceID string) (*SecureCredentials, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	creds, ok := s.creds[deviceID]
	return creds, ok
}

// Delete removes credentials for a device.
func (s *CredentialStore) Delete(deviceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.creds, deviceID)
}

// Count returns the number of stored credentials.
func (s *CredentialStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.creds)
}

// Export exports all credentials as encrypted JSON.
func (s *CredentialStore) Export() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.Marshal(s.creds)
	if err != nil {
		return nil, err
	}

	return s.encryptor.Encrypt(data)
}

// Import imports credentials from encrypted data.
func (s *CredentialStore) Import(encryptedData []byte) error {
	data, err := s.encryptor.Decrypt(encryptedData)
	if err != nil {
		return err
	}

	var creds map[string]*SecureCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.creds = creds
	return nil
}

// Clear removes all stored credentials.
func (s *CredentialStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creds = make(map[string]*SecureCredentials)
}
