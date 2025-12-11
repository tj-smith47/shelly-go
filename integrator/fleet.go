package integrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// FleetManager provides fleet-wide operations across multiple devices and hosts.
type FleetManager struct {
	client        *Client
	accounts      *AccountManager
	connections   map[string]*Connection
	statusCache   map[string]*DeviceStatus
	groups        map[string]*DeviceGroup
	healthMonitor *HealthMonitor
	mu            sync.RWMutex
}

// NewFleetManager creates a new fleet manager.
func NewFleetManager(client *Client) *FleetManager {
	return &FleetManager{
		client:        client,
		accounts:      NewAccountManager(),
		connections:   make(map[string]*Connection),
		statusCache:   make(map[string]*DeviceStatus),
		groups:        make(map[string]*DeviceGroup),
		healthMonitor: NewHealthMonitor(),
	}
}

// SetAccountManager sets a custom account manager.
func (fm *FleetManager) SetAccountManager(am *AccountManager) {
	fm.mu.Lock()
	fm.accounts = am
	fm.mu.Unlock()
}

// AccountManager returns the account manager.
func (fm *FleetManager) AccountManager() *AccountManager {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.accounts
}

// DeviceStatus represents the cached status of a device.
type DeviceStatus struct {
	LastSeen time.Time `json:"last_seen"`
	types.RawFields
	DeviceID     string          `json:"device_id"`
	Host         string          `json:"host"`
	LastStatus   json.RawMessage `json:"last_status,omitempty"`
	LastSettings json.RawMessage `json:"last_settings,omitempty"`
	Online       bool            `json:"online"`
}

// Connect connects to a specific Shelly cloud host.
func (fm *FleetManager) Connect(ctx context.Context, host string, opts *ConnectOptions) (*Connection, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Check if already connected
	if conn, ok := fm.connections[host]; ok && !conn.IsClosed() {
		return conn, nil
	}

	conn, err := fm.client.Connect(ctx, host, opts)
	if err != nil {
		return nil, err
	}

	fm.connections[host] = conn

	// Set up event handlers
	conn.OnStatusChange(fm.handleStatusChange)
	conn.OnOnlineStatus(fm.handleOnlineStatus)
	conn.OnSettingsChange(fm.handleSettingsChange)

	return conn, nil
}

// ConnectAll connects to all hosts that have devices.
func (fm *FleetManager) ConnectAll(ctx context.Context, opts *ConnectOptions) map[string]error {
	hosts := fm.getUniqueHosts()
	errors := make(map[string]error)

	for _, host := range hosts {
		if _, err := fm.Connect(ctx, host, opts); err != nil {
			errors[host] = err
		}
	}

	return errors
}

// Disconnect disconnects from a specific host.
func (fm *FleetManager) Disconnect(host string) error {
	fm.mu.Lock()
	conn, ok := fm.connections[host]
	if ok {
		delete(fm.connections, host)
	}
	fm.mu.Unlock()

	if !ok {
		return nil
	}
	return conn.Close()
}

// DisconnectAll disconnects from all hosts.
func (fm *FleetManager) DisconnectAll() error {
	fm.mu.Lock()
	conns := make([]*Connection, 0, len(fm.connections))
	for _, conn := range fm.connections {
		conns = append(conns, conn)
	}
	fm.connections = make(map[string]*Connection)
	fm.mu.Unlock()

	var lastErr error
	for _, conn := range conns {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (fm *FleetManager) getUniqueHosts() []string {
	devices := fm.accounts.ListDevices()
	hostSet := make(map[string]bool)
	for i := range devices {
		if devices[i].Host != "" {
			hostSet[devices[i].Host] = true
		}
	}

	hosts := make([]string, 0, len(hostSet))
	for host := range hostSet {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	return hosts
}

func (fm *FleetManager) handleStatusChange(event *StatusChangeEvent) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	status, ok := fm.statusCache[event.DeviceID]
	if !ok {
		status = &DeviceStatus{DeviceID: event.DeviceID}
		fm.statusCache[event.DeviceID] = status
	}

	status.LastStatus = event.Status
	status.LastSeen = time.Now()
	status.Online = true

	// Update health monitor
	fm.healthMonitor.RecordActivity(event.DeviceID)
}

func (fm *FleetManager) handleOnlineStatus(event *OnlineStatusEvent) {
	fm.mu.Lock()
	status, ok := fm.statusCache[event.DeviceID]
	if !ok {
		status = &DeviceStatus{DeviceID: event.DeviceID}
		fm.statusCache[event.DeviceID] = status
	}

	status.Online = event.Online
	status.LastSeen = time.Now()
	fm.mu.Unlock()

	// Update account manager
	fm.accounts.UpdateDeviceOnlineStatus(event.DeviceID, event.Online)

	// Update health monitor
	fm.healthMonitor.RecordOnlineStatus(event.DeviceID, event.Online)
}

func (fm *FleetManager) handleSettingsChange(event *SettingsChangeEvent) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	status, ok := fm.statusCache[event.DeviceID]
	if !ok {
		status = &DeviceStatus{DeviceID: event.DeviceID}
		fm.statusCache[event.DeviceID] = status
	}

	status.LastSettings = event.Settings
	status.LastSeen = time.Now()
}

// GetDeviceStatus returns the cached status for a device.
func (fm *FleetManager) GetDeviceStatus(deviceID string) (*DeviceStatus, bool) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	status, ok := fm.statusCache[deviceID]
	return status, ok
}

// ListDeviceStatuses returns all cached device statuses.
func (fm *FleetManager) ListDeviceStatuses() []*DeviceStatus {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	statuses := make([]*DeviceStatus, 0, len(fm.statusCache))
	for _, status := range fm.statusCache {
		statuses = append(statuses, status)
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].DeviceID < statuses[j].DeviceID
	})

	return statuses
}

// SendCommand sends a command to a specific device.
func (fm *FleetManager) SendCommand(ctx context.Context, deviceID, action string, params any) error {
	device, _, ok := fm.accounts.GetDevice(deviceID)
	if !ok {
		return fmt.Errorf("device %s not found", deviceID)
	}

	if !device.CanControl() {
		return fmt.Errorf("no control access to device %s", deviceID)
	}

	fm.mu.RLock()
	conn, ok := fm.connections[device.Host]
	fm.mu.RUnlock()

	if !ok || conn.IsClosed() {
		return fmt.Errorf("not connected to host %s", device.Host)
	}

	return conn.SendCommand(ctx, deviceID, action, params)
}

// SendRelayCommand sends a relay on/off command to a device.
func (fm *FleetManager) SendRelayCommand(ctx context.Context, deviceID string, channel int, on bool) error {
	turn := turnOff
	if on {
		turn = turnOn
	}
	return fm.SendCommand(ctx, deviceID, "relay", map[string]any{
		"id":   channel,
		"turn": turn,
	})
}

// SendRollerCommand sends a roller command to a device.
func (fm *FleetManager) SendRollerCommand(ctx context.Context, deviceID string, channel int, action string) error {
	return fm.SendCommand(ctx, deviceID, "roller", map[string]any{
		"id": channel,
		"go": action,
	})
}

// SendRollerPosition sends a roller to a specific position.
func (fm *FleetManager) SendRollerPosition(ctx context.Context, deviceID string, channel, position int) error {
	return fm.SendCommand(ctx, deviceID, "roller", map[string]any{
		"id":         channel,
		"go":         "to_pos",
		"roller_pos": position,
	})
}

// SendLightCommand sends a light on/off command to a device.
func (fm *FleetManager) SendLightCommand(ctx context.Context, deviceID string, channel int, on bool) error {
	turn := "off"
	if on {
		turn = "on"
	}
	return fm.SendCommand(ctx, deviceID, "light", map[string]any{
		"id":   channel,
		"turn": turn,
	})
}

// BatchCommand sends a command to multiple devices.
type BatchCommand struct {
	Params   any    `json:"params,omitempty"`
	DeviceID string `json:"device_id"`
	Action   string `json:"action"`
}

// BatchResult contains the result of a batch command.
type BatchResult struct {
	DeviceID string `json:"device_id"`
	Error    string `json:"error,omitempty"`
	Success  bool   `json:"success"`
}

// SendBatchCommands sends commands to multiple devices.
func (fm *FleetManager) SendBatchCommands(ctx context.Context, commands []BatchCommand) []BatchResult {
	results := make([]BatchResult, len(commands))

	for i, cmd := range commands {
		results[i].DeviceID = cmd.DeviceID
		if err := fm.SendCommand(ctx, cmd.DeviceID, cmd.Action, cmd.Params); err != nil {
			results[i].Success = false
			results[i].Error = err.Error()
		} else {
			results[i].Success = true
		}
	}

	return results
}

// AllRelaysOn turns on all relay devices.
func (fm *FleetManager) AllRelaysOn(ctx context.Context) []BatchResult {
	devices := fm.accounts.GetControllableDevices()
	commands := make([]BatchCommand, 0, len(devices))

	for i := range devices {
		// Filter to relay-capable devices (switches, plugs)
		if isRelayDevice(devices[i].DeviceType) {
			commands = append(commands, BatchCommand{
				DeviceID: devices[i].DeviceID,
				Action:   "relay",
				Params:   map[string]any{"id": 0, "turn": "on"},
			})
		}
	}

	return fm.SendBatchCommands(ctx, commands)
}

// AllRelaysOff turns off all relay devices.
func (fm *FleetManager) AllRelaysOff(ctx context.Context) []BatchResult {
	devices := fm.accounts.GetControllableDevices()
	commands := make([]BatchCommand, 0, len(devices))

	for i := range devices {
		if isRelayDevice(devices[i].DeviceType) {
			commands = append(commands, BatchCommand{
				DeviceID: devices[i].DeviceID,
				Action:   "relay",
				Params:   map[string]any{"id": 0, "turn": "off"},
			})
		}
	}

	return fm.SendBatchCommands(ctx, commands)
}

func isRelayDevice(deviceType string) bool {
	relayTypes := []string{
		"SHSW-1", "SHSW-25", "SHSW-PM", "SHSW-44",
		"SHPLG-S", "SHPLG-1", "SHPLG-U1", "SHPLG2-1",
		"SNSW-001", "SNSW-001P", "SNSW-002",
		"SPSW-001", "SPSW-001PE", "SPSW-002PE",
	}
	for _, rt := range relayTypes {
		if deviceType == rt {
			return true
		}
	}
	return false
}

// DeviceGroup represents a group of devices for batch operations.
type DeviceGroup struct {
	CreatedAt time.Time `json:"created_at"`
	types.RawFields
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	DeviceIDs   []string `json:"device_ids"`
}

// CreateGroup creates a new device group.
func (fm *FleetManager) CreateGroup(id, name string, deviceIDs []string) *DeviceGroup {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	group := &DeviceGroup{
		ID:        id,
		Name:      name,
		DeviceIDs: deviceIDs,
		CreatedAt: time.Now(),
	}
	fm.groups[id] = group
	return group
}

// GetGroup returns a group by ID.
func (fm *FleetManager) GetGroup(id string) (*DeviceGroup, bool) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	group, ok := fm.groups[id]
	return group, ok
}

// ListGroups returns all groups.
func (fm *FleetManager) ListGroups() []*DeviceGroup {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	groups := make([]*DeviceGroup, 0, len(fm.groups))
	for _, g := range fm.groups {
		groups = append(groups, g)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})

	return groups
}

// DeleteGroup deletes a group.
func (fm *FleetManager) DeleteGroup(id string) bool {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	_, ok := fm.groups[id]
	if ok {
		delete(fm.groups, id)
	}
	return ok
}

// AddToGroup adds devices to a group.
func (fm *FleetManager) AddToGroup(groupID string, deviceIDs ...string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	group, ok := fm.groups[groupID]
	if !ok {
		return fmt.Errorf("group %s not found", groupID)
	}

	// Add devices that aren't already in the group
	existing := make(map[string]bool)
	for _, id := range group.DeviceIDs {
		existing[id] = true
	}

	for _, id := range deviceIDs {
		if !existing[id] {
			group.DeviceIDs = append(group.DeviceIDs, id)
			existing[id] = true
		}
	}

	return nil
}

// RemoveFromGroup removes devices from a group.
func (fm *FleetManager) RemoveFromGroup(groupID string, deviceIDs ...string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	group, ok := fm.groups[groupID]
	if !ok {
		return fmt.Errorf("group %s not found", groupID)
	}

	toRemove := make(map[string]bool)
	for _, id := range deviceIDs {
		toRemove[id] = true
	}

	newDeviceIDs := make([]string, 0, len(group.DeviceIDs))
	for _, id := range group.DeviceIDs {
		if !toRemove[id] {
			newDeviceIDs = append(newDeviceIDs, id)
		}
	}
	group.DeviceIDs = newDeviceIDs

	return nil
}

// SendGroupCommand sends a command to all devices in a group.
func (fm *FleetManager) SendGroupCommand(ctx context.Context, groupID, action string, params any) []BatchResult {
	fm.mu.RLock()
	group, ok := fm.groups[groupID]
	fm.mu.RUnlock()

	if !ok {
		return []BatchResult{{Error: fmt.Sprintf("group %s not found", groupID)}}
	}

	commands := make([]BatchCommand, len(group.DeviceIDs))
	for i, deviceID := range group.DeviceIDs {
		commands[i] = BatchCommand{
			DeviceID: deviceID,
			Action:   action,
			Params:   params,
		}
	}

	return fm.SendBatchCommands(ctx, commands)
}

// GroupRelaysOn turns on all relays in a group.
func (fm *FleetManager) GroupRelaysOn(ctx context.Context, groupID string) []BatchResult {
	return fm.SendGroupCommand(ctx, groupID, "relay", map[string]any{"id": 0, "turn": "on"})
}

// GroupRelaysOff turns off all relays in a group.
func (fm *FleetManager) GroupRelaysOff(ctx context.Context, groupID string) []BatchResult {
	return fm.SendGroupCommand(ctx, groupID, "relay", map[string]any{"id": 0, "turn": "off"})
}

// HealthMonitor tracks device health metrics.
type HealthMonitor struct {
	deviceHealth  map[string]*DeviceHealth
	checkInterval time.Duration
	mu            sync.RWMutex
}

// DeviceHealth contains health metrics for a device.
type DeviceHealth struct {
	LastSeen  time.Time `json:"last_seen"`
	FirstSeen time.Time `json:"first_seen"`
	types.RawFields
	DeviceID      string `json:"device_id"`
	OnlineCount   int    `json:"online_count"`
	OfflineCount  int    `json:"offline_count"`
	ActivityCount int    `json:"activity_count"`
	Online        bool   `json:"online"`
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor() *HealthMonitor {
	return &HealthMonitor{
		deviceHealth:  make(map[string]*DeviceHealth),
		checkInterval: 5 * time.Minute,
	}
}

// RecordActivity records device activity.
func (hm *HealthMonitor) RecordActivity(deviceID string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health, ok := hm.deviceHealth[deviceID]
	if !ok {
		now := time.Now()
		health = &DeviceHealth{
			DeviceID:  deviceID,
			FirstSeen: now,
		}
		hm.deviceHealth[deviceID] = health
	}

	health.LastSeen = time.Now()
	health.ActivityCount++
	health.Online = true
}

// RecordOnlineStatus records an online/offline event.
func (hm *HealthMonitor) RecordOnlineStatus(deviceID string, online bool) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health, ok := hm.deviceHealth[deviceID]
	if !ok {
		now := time.Now()
		health = &DeviceHealth{
			DeviceID:  deviceID,
			FirstSeen: now,
		}
		hm.deviceHealth[deviceID] = health
	}

	health.LastSeen = time.Now()
	health.Online = online

	if online {
		health.OnlineCount++
	} else {
		health.OfflineCount++
	}
}

// GetDeviceHealth returns health data for a device.
func (hm *HealthMonitor) GetDeviceHealth(deviceID string) (*DeviceHealth, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	health, ok := hm.deviceHealth[deviceID]
	return health, ok
}

// ListDeviceHealth returns health data for all devices.
func (hm *HealthMonitor) ListDeviceHealth() []*DeviceHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	healthList := make([]*DeviceHealth, 0, len(hm.deviceHealth))
	for _, h := range hm.deviceHealth {
		healthList = append(healthList, h)
	}

	sort.Slice(healthList, func(i, j int) bool {
		return healthList[i].DeviceID < healthList[j].DeviceID
	})

	return healthList
}

// GetUnhealthyDevices returns devices that haven't been seen recently.
func (hm *HealthMonitor) GetUnhealthyDevices(threshold time.Duration) []*DeviceHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var unhealthy []*DeviceHealth
	cutoff := time.Now().Add(-threshold)

	for _, h := range hm.deviceHealth {
		if h.LastSeen.Before(cutoff) || !h.Online {
			unhealthy = append(unhealthy, h)
		}
	}

	return unhealthy
}

// GetOnlineDevices returns currently online devices.
func (hm *HealthMonitor) GetOnlineDevices() []*DeviceHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var online []*DeviceHealth
	for _, h := range hm.deviceHealth {
		if h.Online {
			online = append(online, h)
		}
	}

	return online
}

// FleetStats contains aggregate statistics for the fleet.
type FleetStats struct {
	AccountStats *AccountStats `json:"account_stats,omitempty"`
	types.RawFields
	TotalDevices     int `json:"total_devices"`
	OnlineDevices    int `json:"online_devices"`
	OfflineDevices   int `json:"offline_devices"`
	TotalConnections int `json:"total_connections"`
	TotalGroups      int `json:"total_groups"`
}

// GetStats returns fleet-wide statistics.
func (fm *FleetManager) GetStats() *FleetStats {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	stats := &FleetStats{
		TotalDevices:     len(fm.statusCache),
		TotalConnections: len(fm.connections),
		TotalGroups:      len(fm.groups),
		AccountStats:     fm.accounts.GetStats(),
	}

	for _, status := range fm.statusCache {
		if status.Online {
			stats.OnlineDevices++
		} else {
			stats.OfflineDevices++
		}
	}

	return stats
}

// HealthMonitor returns the health monitor.
func (fm *FleetManager) HealthMonitor() *HealthMonitor {
	return fm.healthMonitor
}

// ToJSON serializes the fleet manager state to JSON.
func (fm *FleetManager) ToJSON() ([]byte, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	state := struct {
		Groups   []*DeviceGroup  `json:"groups"`
		Statuses []*DeviceStatus `json:"statuses"`
		Health   []*DeviceHealth `json:"health"`
	}{
		Groups:   fm.ListGroups(),
		Statuses: fm.ListDeviceStatuses(),
		Health:   fm.healthMonitor.ListDeviceHealth(),
	}

	return json.Marshal(state)
}
