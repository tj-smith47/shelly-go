package integrator

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestNewFleetManager(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	if fm == nil {
		t.Fatal("NewFleetManager() returned nil")
	}
	if fm.accounts == nil {
		t.Error("accounts is nil")
	}
	if fm.healthMonitor == nil {
		t.Error("healthMonitor is nil")
	}
}

func TestFleetManager_SetAccountManager(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	am := NewAccountManager()
	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev1"})

	fm.SetAccountManager(am)

	if fm.AccountManager().DeviceCount() != 1 {
		t.Error("AccountManager not set correctly")
	}
}

func TestFleetManager_GetDeviceStatus(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	fm.statusCache["dev1"] = &DeviceStatus{
		DeviceID: "dev1",
		Online:   true,
		LastSeen: time.Now(),
	}

	status, ok := fm.GetDeviceStatus("dev1")
	if !ok {
		t.Error("GetDeviceStatus() returned false")
	}
	if !status.Online {
		t.Error("Online = false, want true")
	}

	_, ok = fm.GetDeviceStatus("nonexistent")
	if ok {
		t.Error("GetDeviceStatus(nonexistent) should return false")
	}
}

func TestFleetManager_ListDeviceStatuses(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	fm.statusCache["dev2"] = &DeviceStatus{DeviceID: "dev2"}
	fm.statusCache["dev1"] = &DeviceStatus{DeviceID: "dev1"}

	statuses := fm.ListDeviceStatuses()
	if len(statuses) != 2 {
		t.Fatalf("len(ListDeviceStatuses()) = %d, want 2", len(statuses))
	}

	// Should be sorted
	if statuses[0].DeviceID != "dev1" || statuses[1].DeviceID != "dev2" {
		t.Error("statuses not sorted")
	}
}

func TestFleetManager_handleStatusChange(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	event := &StatusChangeEvent{
		DeviceID:  "dev1",
		Status:    json.RawMessage(`{"output":true}`),
		Timestamp: time.Now(),
	}

	fm.handleStatusChange(event)

	status, ok := fm.GetDeviceStatus("dev1")
	if !ok {
		t.Fatal("device status not created")
	}
	if !status.Online {
		t.Error("Online = false, want true")
	}
	if status.LastStatus == nil {
		t.Error("LastStatus is nil")
	}
}

func TestFleetManager_handleOnlineStatus(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1"})

	event := &OnlineStatusEvent{
		DeviceID:  "dev1",
		Online:    true,
		Timestamp: time.Now(),
	}

	fm.handleOnlineStatus(event)

	status, ok := fm.GetDeviceStatus("dev1")
	if !ok {
		t.Fatal("device status not created")
	}
	if !status.Online {
		t.Error("Online = false, want true")
	}

	// Also updates account manager
	device, _, _ := fm.accounts.GetDevice("dev1")
	if !device.Online {
		t.Error("account device Online = false, want true")
	}
}

func TestFleetManager_handleSettingsChange(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	event := &SettingsChangeEvent{
		DeviceID:  "dev1",
		Settings:  json.RawMessage(`{"name":"Test"}`),
		Timestamp: time.Now(),
	}

	fm.handleSettingsChange(event)

	status, ok := fm.GetDeviceStatus("dev1")
	if !ok {
		t.Fatal("device status not created")
	}
	if status.LastSettings == nil {
		t.Error("LastSettings is nil")
	}
}

func TestFleetManager_SendCommand_NotFound(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	err := fm.SendCommand(context.Background(), "nonexistent", "relay", nil)
	if err == nil {
		t.Error("SendCommand() should error for nonexistent device")
	}
}

func TestFleetManager_SendCommand_NoControl(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", AccessGroups: "00"})

	err := fm.SendCommand(context.Background(), "dev1", "relay", nil)
	if err == nil {
		t.Error("SendCommand() should error for read-only device")
	}
}

func TestFleetManager_SendCommand_NotConnected(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", AccessGroups: "01", Host: "host1"})

	err := fm.SendCommand(context.Background(), "dev1", "relay", nil)
	if err == nil {
		t.Error("SendCommand() should error when not connected")
	}
}

func TestFleetManager_SendRelayCommand(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	// Will fail because not connected, but tests the wrapper
	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", AccessGroups: "01", Host: "host1"})

	err := fm.SendRelayCommand(context.Background(), "dev1", 0, true)
	if err == nil {
		t.Error("should error when not connected")
	}
}

func TestFleetManager_SendRollerCommand(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", AccessGroups: "01", Host: "host1"})

	err := fm.SendRollerCommand(context.Background(), "dev1", 0, "open")
	if err == nil {
		t.Error("should error when not connected")
	}
}

func TestFleetManager_SendRollerPosition(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", AccessGroups: "01", Host: "host1"})

	err := fm.SendRollerPosition(context.Background(), "dev1", 0, 50)
	if err == nil {
		t.Error("should error when not connected")
	}
}

func TestFleetManager_SendLightCommand(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", AccessGroups: "01", Host: "host1"})

	err := fm.SendLightCommand(context.Background(), "dev1", 0, true)
	if err == nil {
		t.Error("should error when not connected")
	}
}

func TestFleetManager_SendBatchCommands(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", AccessGroups: "01", Host: "host1"})
	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev2", AccessGroups: "01", Host: "host1"})

	commands := []BatchCommand{
		{DeviceID: "dev1", Action: "relay", Params: map[string]any{"turn": "on"}},
		{DeviceID: "dev2", Action: "relay", Params: map[string]any{"turn": "on"}},
		{DeviceID: "nonexistent", Action: "relay"},
	}

	results := fm.SendBatchCommands(context.Background(), commands)
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	// All should fail (not connected)
	for _, r := range results {
		if r.Success {
			t.Errorf("device %s Success = true, want false", r.DeviceID)
		}
	}
}

func TestFleetManager_AllRelaysOff(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", DeviceType: "SHSW-1", AccessGroups: "01", Host: "host1"})
	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev2", DeviceType: "SHPLG-S", AccessGroups: "01", Host: "host1"})

	results := fm.AllRelaysOff(context.Background())
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(results))
	}
}

func TestFleetManager_AllRelaysOn(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", DeviceType: "SHSW-1", AccessGroups: "01", Host: "host1"})

	results := fm.AllRelaysOn(context.Background())
	if len(results) != 1 {
		t.Errorf("len(results) = %d, want 1", len(results))
	}
}

func TestIsRelayDevice(t *testing.T) {
	tests := []struct {
		deviceType string
		want       bool
	}{
		{"SHSW-1", true},
		{"SHPLG-S", true},
		{"SHSW-25", true},
		{"UNKNOWN", false},
		{"SHHT-1", false},
	}

	for _, tt := range tests {
		if got := isRelayDevice(tt.deviceType); got != tt.want {
			t.Errorf("isRelayDevice(%q) = %v, want %v", tt.deviceType, got, tt.want)
		}
	}
}

func TestFleetManager_Groups(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	// Create group
	group := fm.CreateGroup("g1", "Group 1", []string{"dev1", "dev2"})
	if group.ID != "g1" {
		t.Errorf("ID = %v, want g1", group.ID)
	}

	// Get group
	g, ok := fm.GetGroup("g1")
	if !ok {
		t.Error("GetGroup() returned false")
	}
	if g.Name != "Group 1" {
		t.Errorf("Name = %v, want Group 1", g.Name)
	}

	// List groups
	groups := fm.ListGroups()
	if len(groups) != 1 {
		t.Errorf("len(ListGroups()) = %d, want 1", len(groups))
	}

	// Get nonexistent
	_, ok = fm.GetGroup("nonexistent")
	if ok {
		t.Error("GetGroup(nonexistent) should return false")
	}

	// Delete group
	deleted := fm.DeleteGroup("g1")
	if !deleted {
		t.Error("DeleteGroup() = false, want true")
	}

	deleted = fm.DeleteGroup("nonexistent")
	if deleted {
		t.Error("DeleteGroup(nonexistent) = true, want false")
	}
}

func TestFleetManager_AddRemoveFromGroup(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	fm.CreateGroup("g1", "Group 1", []string{"dev1"})

	// Add devices
	err := fm.AddToGroup("g1", "dev2", "dev3")
	if err != nil {
		t.Fatalf("AddToGroup() error = %v", err)
	}

	g, _ := fm.GetGroup("g1")
	if len(g.DeviceIDs) != 3 {
		t.Errorf("len(DeviceIDs) = %d, want 3", len(g.DeviceIDs))
	}

	// Add duplicate (should not add)
	_ = fm.AddToGroup("g1", "dev1")
	g, _ = fm.GetGroup("g1")
	if len(g.DeviceIDs) != 3 {
		t.Error("duplicate device added")
	}

	// Remove devices
	err = fm.RemoveFromGroup("g1", "dev2")
	if err != nil {
		t.Fatalf("RemoveFromGroup() error = %v", err)
	}

	g, _ = fm.GetGroup("g1")
	if len(g.DeviceIDs) != 2 {
		t.Errorf("len(DeviceIDs) = %d, want 2", len(g.DeviceIDs))
	}

	// Add to nonexistent group
	err = fm.AddToGroup("nonexistent", "dev1")
	if err == nil {
		t.Error("AddToGroup(nonexistent) should error")
	}

	// Remove from nonexistent group
	err = fm.RemoveFromGroup("nonexistent", "dev1")
	if err == nil {
		t.Error("RemoveFromGroup(nonexistent) should error")
	}
}

func TestFleetManager_SendGroupCommand(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", AccessGroups: "01", Host: "host1"})
	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev2", AccessGroups: "01", Host: "host1"})
	fm.CreateGroup("g1", "Group 1", []string{"dev1", "dev2"})

	results := fm.SendGroupCommand(context.Background(), "g1", "relay", map[string]any{"turn": "on"})
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(results))
	}

	// Nonexistent group
	results = fm.SendGroupCommand(context.Background(), "nonexistent", "relay", nil)
	if len(results) != 1 || results[0].Error == "" {
		t.Error("SendGroupCommand(nonexistent) should return error result")
	}
}

func TestFleetManager_GroupRelaysOnOff(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", AccessGroups: "01", Host: "host1"})
	fm.CreateGroup("g1", "Group 1", []string{"dev1"})

	results := fm.GroupRelaysOn(context.Background(), "g1")
	if len(results) != 1 {
		t.Errorf("len(GroupRelaysOn) = %d, want 1", len(results))
	}

	results = fm.GroupRelaysOff(context.Background(), "g1")
	if len(results) != 1 {
		t.Errorf("len(GroupRelaysOff) = %d, want 1", len(results))
	}
}

func TestHealthMonitor_RecordActivity(t *testing.T) {
	hm := NewHealthMonitor()

	hm.RecordActivity("dev1")
	hm.RecordActivity("dev1")
	hm.RecordActivity("dev2")

	health, ok := hm.GetDeviceHealth("dev1")
	if !ok {
		t.Fatal("GetDeviceHealth() returned false")
	}
	if health.ActivityCount != 2 {
		t.Errorf("ActivityCount = %d, want 2", health.ActivityCount)
	}
	if !health.Online {
		t.Error("Online = false, want true")
	}
}

func TestHealthMonitor_RecordOnlineStatus(t *testing.T) {
	hm := NewHealthMonitor()

	hm.RecordOnlineStatus("dev1", true)
	hm.RecordOnlineStatus("dev1", false)
	hm.RecordOnlineStatus("dev1", true)

	health, _ := hm.GetDeviceHealth("dev1")
	if health.OnlineCount != 2 {
		t.Errorf("OnlineCount = %d, want 2", health.OnlineCount)
	}
	if health.OfflineCount != 1 {
		t.Errorf("OfflineCount = %d, want 1", health.OfflineCount)
	}
}

func TestHealthMonitor_ListDeviceHealth(t *testing.T) {
	hm := NewHealthMonitor()

	hm.RecordActivity("dev2")
	hm.RecordActivity("dev1")

	healthList := hm.ListDeviceHealth()
	if len(healthList) != 2 {
		t.Fatalf("len(ListDeviceHealth()) = %d, want 2", len(healthList))
	}

	// Should be sorted
	if healthList[0].DeviceID != "dev1" {
		t.Error("not sorted")
	}
}

func TestHealthMonitor_GetUnhealthyDevices(t *testing.T) {
	hm := NewHealthMonitor()

	hm.RecordActivity("dev1")
	hm.deviceHealth["dev1"].LastSeen = time.Now().Add(-10 * time.Minute)

	hm.RecordActivity("dev2")
	hm.deviceHealth["dev2"].Online = false

	hm.RecordActivity("dev3")

	unhealthy := hm.GetUnhealthyDevices(5 * time.Minute)
	if len(unhealthy) != 2 {
		t.Errorf("len(unhealthy) = %d, want 2", len(unhealthy))
	}
}

func TestHealthMonitor_GetOnlineDevices(t *testing.T) {
	hm := NewHealthMonitor()

	hm.RecordActivity("dev1")
	hm.RecordOnlineStatus("dev2", false)
	hm.RecordActivity("dev3")

	online := hm.GetOnlineDevices()
	if len(online) != 2 {
		t.Errorf("len(online) = %d, want 2", len(online))
	}
}

func TestFleetManager_GetStats(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	fm.statusCache["dev1"] = &DeviceStatus{DeviceID: "dev1", Online: true}
	fm.statusCache["dev2"] = &DeviceStatus{DeviceID: "dev2", Online: false}
	fm.CreateGroup("g1", "Group 1", nil)

	stats := fm.GetStats()

	if stats.TotalDevices != 2 {
		t.Errorf("TotalDevices = %d, want 2", stats.TotalDevices)
	}
	if stats.OnlineDevices != 1 {
		t.Errorf("OnlineDevices = %d, want 1", stats.OnlineDevices)
	}
	if stats.OfflineDevices != 1 {
		t.Errorf("OfflineDevices = %d, want 1", stats.OfflineDevices)
	}
	if stats.TotalGroups != 1 {
		t.Errorf("TotalGroups = %d, want 1", stats.TotalGroups)
	}
}

func TestFleetManager_HealthMonitor(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	hm := fm.HealthMonitor()
	if hm == nil {
		t.Error("HealthMonitor() returned nil")
	}
}

func TestFleetManager_ToJSON(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	fm.statusCache["dev1"] = &DeviceStatus{DeviceID: "dev1", Online: true}
	fm.CreateGroup("g1", "Group 1", []string{"dev1"})
	fm.healthMonitor.RecordActivity("dev1")

	data, err := fm.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var state struct {
		Groups   []*DeviceGroup  `json:"groups"`
		Statuses []*DeviceStatus `json:"statuses"`
		Health   []*DeviceHealth `json:"health"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(state.Groups) != 1 {
		t.Errorf("len(Groups) = %d, want 1", len(state.Groups))
	}
	if len(state.Statuses) != 1 {
		t.Errorf("len(Statuses) = %d, want 1", len(state.Statuses))
	}
	if len(state.Health) != 1 {
		t.Errorf("len(Health) = %d, want 1", len(state.Health))
	}
}

func TestFleetManager_Disconnect(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	// Disconnect nonexistent should not error
	err := fm.Disconnect("nonexistent")
	if err != nil {
		t.Errorf("Disconnect(nonexistent) error = %v", err)
	}
}

func TestFleetManager_DisconnectAll(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	// Add a mock connection
	fm.connections["host1"] = &Connection{host: "host1", closeCh: make(chan struct{})}

	err := fm.DisconnectAll()
	if err != nil {
		t.Errorf("DisconnectAll() error = %v", err)
	}

	if len(fm.connections) != 0 {
		t.Error("connections not cleared")
	}
}

func TestFleetManager_getUniqueHosts(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", Host: "host1"})
	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev2", Host: "host2"})
	_ = fm.accounts.AddDevice("user2", &AccountDevice{DeviceID: "dev3", Host: "host1"})

	hosts := fm.getUniqueHosts()
	if len(hosts) != 2 {
		t.Errorf("len(hosts) = %d, want 2", len(hosts))
	}

	// Should be sorted
	if hosts[0] != "host1" || hosts[1] != "host2" {
		t.Error("hosts not sorted")
	}
}

func TestFleetManager_Disconnect_CloseError(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)

	// Create connection with mock WS that returns error on Close
	closeErr := fmt.Errorf("close error")
	conn := &Connection{
		host:    "host1",
		closeCh: make(chan struct{}),
		ws: &mockWSConnector{
			closeFunc: func() error { return closeErr },
		},
	}
	fm.connections["host1"] = conn

	err := fm.Disconnect("host1")
	if err == nil {
		t.Error("Disconnect() should return error when Close() fails")
	}
	if err.Error() != "close error" {
		t.Errorf("error = %v, want 'close error'", err)
	}

	// Connection should be removed from map even on error
	if _, ok := fm.connections["host1"]; ok {
		t.Error("connection should be removed from map after disconnect")
	}
}
