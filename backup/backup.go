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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/gen2/components"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Component name identifiers used for restore tracking and migration reporting.
const (
	componentCloud = "cloud"
	componentBLE   = "ble"
	componentMQTT  = "mqtt"
	componentWiFi  = "wifi"
	componentCoIoT = "coiot"
	componentSys   = "sys"
)

// RPC parameter names used by more than one call.
const (
	paramConfig = "config"
	paramKey    = "key"
)

// Common errors.
var (
	// ErrInvalidBackup indicates the backup data is invalid.
	ErrInvalidBackup = errors.New("invalid backup data")

	// ErrVersionMismatch indicates the backup version is not supported.
	ErrVersionMismatch = errors.New("backup version not supported")

	// ErrDeviceMismatch indicates the backup is for a different device model.
	ErrDeviceMismatch = errors.New("backup device model mismatch")

	// ErrIncompleteStaticNetwork means a network override sets a static address
	// that has no gateway or no netmask, from neither the override nor the
	// backup. Errors carrying it also match types.ErrInvalidParam.
	ErrIncompleteStaticNetwork = errors.New("static address without a gateway or netmask")
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

	if err := m.exportOptionalConfigs(ctx, opts, backup); err != nil {
		return nil, err
	}

	return json.MarshalIndent(backup, "", "  ")
}

// exportOptionalConfigs reads every section the options include. A device
// without the component (no BLE radio, no scripting) answers that the method
// does not exist, and its backup is complete without that section. Any other
// failure is returned, because the backup would lack settings the device has.
func (m *Manager) exportOptionalConfigs(ctx context.Context, opts *ExportOptions, backup *Backup) error {
	config := func(method string, dst *json.RawMessage) func() error {
		return func() (err error) {
			*dst, err = m.getConfig(ctx, method)
			return err
		}
	}
	sections := []struct {
		read    func() error
		name    string
		include bool
	}{
		{name: "WiFi config", include: opts.IncludeWiFi, read: config("WiFi.GetConfig", &backup.WiFi)},
		{name: "Cloud config", include: opts.IncludeCloud, read: config("Cloud.GetConfig", &backup.Cloud)},
		{name: "BLE config", include: opts.IncludeBLE, read: config("BLE.GetConfig", &backup.BLE)},
		{name: "MQTT config", include: opts.IncludeMQTT, read: config("MQTT.GetConfig", &backup.MQTT)},
		{name: "webhook list", include: opts.IncludeWebhooks, read: config("Webhook.List", &backup.Webhooks)},
		{name: "schedule list", include: opts.IncludeSchedules, read: config("Schedule.List", &backup.Schedules)},
		{name: "script list", include: opts.IncludeScripts, read: func() (err error) {
			backup.Scripts, err = m.listScripts(ctx)
			return err
		}},
		{name: "KVS", include: opts.IncludeKVS, read: func() (err error) {
			backup.KVS, err = m.listKVS(ctx)
			return err
		}},
		{name: "auth info", include: opts.IncludeAuth, read: func() (err error) {
			backup.Auth, err = m.getAuthInfo(ctx)
			return err
		}},
	}
	for _, section := range sections {
		if !section.include {
			continue
		}
		err := section.read()
		if err != nil && !errors.Is(err, types.ErrNotFound) && !errors.Is(err, types.ErrNotSupported) {
			return fmt.Errorf("failed to read %s: %w", section.name, err)
		}
	}
	return nil
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

	rejected := gen2Rejected{}
	m.restoreOptionalConfigs(ctx, opts, &backup, rejected, result)
	m.restoreDeviceConfig(ctx, opts, &backup, rejected, result)
	m.restoreComplexItems(ctx, opts, &backup, rejected, result)
	m.verifyRestore(ctx, opts, &backup, rejected, result)

	result.Success = len(result.Errors) == 0
	return result, nil
}

// restoreDeviceConfig applies the per-component configuration captured in
// backup.Config (the Shelly.GetConfig result) that the optional-config and
// complex-item restores do not cover: Sys, Ethernet, WebSocket, and every
// component (Switch, Cover, Light, Input, PM, EM, ...). It first switches the
// device to the backup's profile and creates the user-created components the
// device lacks, so the sections that follow have a component to configure.
//
// What the device refuses is recorded in rejected and reported by the check
// that runs after the restore, so one refused field does not hide the rest.
func (m *Manager) restoreDeviceConfig(
	ctx context.Context,
	opts *RestoreOptions,
	backup *Backup,
	rejected gen2Rejected,
	result *RestoreResult,
) {
	if len(backup.Config) == 0 {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(backup.Config, &cfg); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("parse device config: %w", err))
		return
	}

	// Without the device's current config every section is written as if its
	// component exists; the device refuses the ones that do not.
	live, liveErr := m.liveConfig(ctx)
	if liveErr == nil && opts.RestoreComponents {
		var err error
		if live, err = m.restoreProfile(ctx, gen2Profile(cfg), live); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("profile: %w", err))
		}
	}

	keys := make([]string, 0, len(cfg))
	for key := range cfg {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if !shouldRestoreConfigKey(key, opts) {
			continue
		}
		_, present := live[key]
		if m.restoreSection(ctx, key, cfg[key], backup.DeviceInfo, liveErr == nil && !present, rejected) ||
			strings.HasPrefix(key, "eth") {
			result.RestartRequired = true
		}
	}
}

// restoreSection writes one Shelly.GetConfig section, creating its component
// first when the device lacks it and the type is one a user creates. It
// reports whether the device asked for a restart.
func (m *Manager) restoreSection(
	ctx context.Context, key string, raw json.RawMessage, src *DeviceInfo, missing bool, rejected gen2Rejected,
) (restart bool) {
	section, err := decodeConfig(raw)
	if err != nil {
		rejected[key] = err.Error()
		return false
	}
	stripGen2Identity(key, section, src)
	if missing {
		created, addErr := m.addComponent(ctx, key, section)
		if addErr != nil {
			rejected[key] = addErr.Error()
			return false
		}
		if created {
			return false
		}
	}
	return m.applySection(ctx, key, nil, section, rejected)
}

// shouldRestoreConfigKey reports whether a Shelly.GetConfig section should be
// applied during restore, honoring the restore options. WiFi/Cloud/BLE/MQTT are
// handled by restoreOptionalConfigs and scripts by restoreScripts; Ethernet follows the network option; Sys
// and WebSocket are always restored; everything else is treated as a component.
func shouldRestoreConfigKey(key string, opts *RestoreOptions) bool {
	base := key
	if i := strings.IndexByte(key, ':'); i >= 0 {
		base = key[:i]
	}
	switch base {
	case componentWiFi, componentCloud, componentBLE, componentMQTT, "script":
		// Scripts are recreated with their code by the script restore.
		return false
	case "eth":
		return opts.RestoreWiFi
	case componentSys, "ws":
		return true
	default:
		return opts.RestoreComponents
	}
}

// restoreOptionalConfigs restores the WiFi, Cloud, BLE and MQTT sections from
// their own backup fields. A refused WiFi config is an error, because the
// caller asked for the device to change networks. The other three go through
// applySection: Cloud.SetConfig, for one, accepts only the enable flag.
func (m *Manager) restoreOptionalConfigs(
	ctx context.Context,
	opts *RestoreOptions,
	backup *Backup,
	rejected gen2Rejected,
	result *RestoreResult,
) {
	if opts.RestoreWiFi && backup.WiFi != nil {
		if _, err := m.client.Call(ctx, "WiFi.SetConfig", map[string]any{paramConfig: backup.WiFi}); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("WiFi: %w", err))
		} else {
			result.RestartRequired = true
		}
	}
	sections := []struct {
		key     string
		data    json.RawMessage
		restore bool
	}{
		{key: componentCloud, data: backup.Cloud, restore: opts.RestoreCloud},
		{key: componentBLE, data: backup.BLE, restore: opts.RestoreBLE},
		{key: componentMQTT, data: backup.MQTT, restore: opts.RestoreMQTT},
	}
	for _, section := range sections {
		if !section.restore || section.data == nil {
			continue
		}
		cfg, err := decodeConfig(section.data)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", section.key, err))
			continue
		}
		stripGen2Identity(section.key, cfg, backup.DeviceInfo)
		if m.applySection(ctx, section.key, nil, cfg, rejected) {
			result.RestartRequired = true
		}
	}
}

// restoreComplexItems restores schedules, webhooks, scripts, and KVS.
func (m *Manager) restoreComplexItems(
	ctx context.Context,
	opts *RestoreOptions,
	backup *Backup,
	rejected gen2Rejected,
	result *RestoreResult,
) {
	// Scripts go first: a webhook or schedule may start one.
	if opts.RestoreScripts && len(backup.Scripts) > 0 {
		m.restoreScripts(ctx, backup.Scripts, rejected)
	}
	if opts.RestoreSchedules && backup.Schedules != nil {
		err := m.restoreItems(ctx, backup.Schedules, itemsJobs, "Schedule.DeleteAll", "Schedule.Create", rejected)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("schedules: %w", err))
		}
	}
	if opts.RestoreWebhooks && backup.Webhooks != nil {
		err := m.restoreItems(ctx, backup.Webhooks, itemsHooks, "Webhook.DeleteAll", "Webhook.Create", rejected)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("webhooks: %w", err))
		}
	}
	if opts.RestoreKVS && len(backup.KVS) > 0 {
		m.restoreKVS(ctx, backup.KVS, rejected)
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

// listScripts retrieves all scripts with their code.
func (m *Manager) listScripts(ctx context.Context) ([]*Script, error) {
	sc := components.NewScript(m.client)
	list, err := sc.List(ctx)
	if err != nil {
		return nil, err
	}
	scripts := make([]*Script, 0, len(list.Scripts))
	for _, s := range list.Scripts {
		code, err := sc.GetCode(ctx, s.ID)
		if err != nil {
			return nil, fmt.Errorf("script %d code: %w", s.ID, err)
		}
		script := &Script{ID: s.ID, Enable: s.Enable, Running: s.Running, Code: code.Data}
		if s.Name != nil {
			script.Name = *s.Name
		}
		scripts = append(scripts, script)
	}
	return scripts, nil
}

// listKVS retrieves all KVS entries, each as the device's KVS.Get answer.
func (m *Manager) listKVS(ctx context.Context) (map[string]json.RawMessage, error) {
	list, err := components.NewKVS(m.client).List(ctx)
	if err != nil {
		return nil, err
	}
	if len(list.Keys) == 0 {
		return nil, nil
	}
	kvs := make(map[string]json.RawMessage, len(list.Keys))
	for key := range list.Keys {
		value, err := m.client.Call(ctx, "KVS.Get", map[string]any{paramKey: key})
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", key, err)
		}
		kvs[key] = value
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

// restoreItems replaces the device's schedules or webhooks with the backup's.
// The backup is parsed before anything is deleted, and a failed delete stops
// the restore of that list, so the device is never left with neither its own
// items nor the backup's because of a bad backup, and never with both.
func (m *Manager) restoreItems(
	ctx context.Context,
	data json.RawMessage,
	itemsKey, deleteMethod, createMethod string,
	rejected gen2Rejected,
) error {
	items, err := parseItems(data, itemsKey)
	if err != nil {
		return err
	}
	if _, err := m.client.Call(ctx, deleteMethod, nil); err != nil {
		return fmt.Errorf("%s: %w", deleteMethod, err)
	}
	for _, item := range items {
		if _, err := m.client.Call(ctx, createMethod, item); err != nil {
			rejected[itemLabel(itemsKey, item)] = err.Error()
		}
	}
	return nil
}

// restoreScripts recreates each script with its code, enable flag and running
// state. A script already on the device under the same name is replaced, so
// restoring twice does not leave two copies. Scripts are created in backup id
// order, which keeps their ids when the device hands out the lowest free one.
func (m *Manager) restoreScripts(ctx context.Context, scripts []*Script, rejected gen2Rejected) {
	sc := components.NewScript(m.client)
	existing, err := sc.List(ctx)
	if err != nil {
		for _, script := range scripts {
			rejected[scriptLabel(script)] = err.Error()
		}
		return
	}
	ordered := append([]*Script(nil), scripts...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	for _, script := range ordered {
		if err := m.restoreScript(ctx, sc, existing.Scripts, script); err != nil {
			rejected[scriptLabel(script)] = err.Error()
		}
	}
}

func (m *Manager) restoreScript(
	ctx context.Context, sc *components.Script, existing []components.ScriptListItem, script *Script,
) error {
	var name *string
	if script.Name != "" {
		name = &script.Name
		for _, old := range existing {
			if old.Name == nil || *old.Name != script.Name {
				continue
			}
			if err := sc.Delete(ctx, old.ID); err != nil {
				return fmt.Errorf("delete existing: %w", err)
			}
		}
	}
	created, err := sc.Create(ctx, name)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	if script.Code != "" {
		if err := sc.PutCode(ctx, created.ID, script.Code, false); err != nil {
			return fmt.Errorf("upload code: %w", err)
		}
	}
	if script.Enable {
		enable := true
		if err := sc.SetConfig(ctx, created.ID, &components.ScriptConfig{Enable: &enable}); err != nil {
			return fmt.Errorf("enable: %w", err)
		}
	}
	if script.Running {
		if err := sc.Start(ctx, created.ID); err != nil {
			return fmt.Errorf("start: %w", err)
		}
	}
	return nil
}

// restoreKVS restores KVS data from backup.
func (m *Manager) restoreKVS(ctx context.Context, kvs map[string]json.RawMessage, rejected gen2Rejected) {
	for key, value := range kvs {
		var item struct {
			Value any `json:"value"`
		}
		if err := json.Unmarshal(value, &item); err != nil {
			rejected[kvsLabel(key)] = err.Error()
			continue
		}
		if _, err := m.client.Call(ctx, "KVS.Set", map[string]any{paramKey: key, "value": item.Value}); err != nil {
			rejected[kvsLabel(key)] = err.Error()
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
		{name: componentWiFi, include: opts.IncludeWiFi},
		{name: componentCloud, include: opts.IncludeCloud},
		{name: componentMQTT, include: opts.IncludeMQTT},
		{name: componentBLE, include: opts.IncludeBLE},
		{name: "schedule list", include: opts.IncludeSchedules},
		{name: "webhook list", include: opts.IncludeWebhooks},
		{name: "script list", include: opts.IncludeScripts},
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
