package integrator

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAccountDevice_CanControl(t *testing.T) {
	tests := []struct {
		accessGroups string
		want         bool
	}{
		{"00", false},
		{"01", true},
		{"0", false},
		{"", false},
		{"10", false},
		{"11", true},
	}

	for _, tt := range tests {
		d := AccountDevice{AccessGroups: tt.accessGroups}
		if got := d.CanControl(); got != tt.want {
			t.Errorf("AccessGroups=%q: CanControl() = %v, want %v", tt.accessGroups, got, tt.want)
		}
	}
}

func TestAccountManager_AddAccount(t *testing.T) {
	am := NewAccountManager()

	var addedAccount *Account
	am.OnAccountAdded(func(a *Account) {
		addedAccount = a
	})

	account := &Account{
		UserID: "user1",
		Email:  "user1@example.com",
		Devices: []AccountDevice{
			{DeviceID: "dev1", DeviceType: "SHSW-1"},
		},
		GrantedAt: time.Now(),
	}

	am.AddAccount(account)

	if addedAccount == nil {
		t.Error("OnAccountAdded callback not called")
	}

	got, ok := am.GetAccount("user1")
	if !ok {
		t.Error("GetAccount() returned false")
	}
	if got.Email != "user1@example.com" {
		t.Errorf("Email = %v, want user1@example.com", got.Email)
	}
}

func TestAccountManager_RemoveAccount(t *testing.T) {
	am := NewAccountManager()

	var removedUserID string
	am.OnAccountRemoved(func(userID string) {
		removedUserID = userID
	})

	am.AddAccount(&Account{
		UserID: "user1",
		Devices: []AccountDevice{
			{DeviceID: "dev1"},
		},
	})

	removed := am.RemoveAccount("user1")
	if !removed {
		t.Error("RemoveAccount() = false, want true")
	}
	if removedUserID != "user1" {
		t.Errorf("removedUserID = %v, want user1", removedUserID)
	}

	_, ok := am.GetAccount("user1")
	if ok {
		t.Error("GetAccount() should return false after removal")
	}

	// Remove nonexistent
	removed = am.RemoveAccount("nonexistent")
	if removed {
		t.Error("RemoveAccount(nonexistent) = true, want false")
	}
}

func TestAccountManager_ListAccounts(t *testing.T) {
	am := NewAccountManager()

	am.AddAccount(&Account{UserID: "user2"})
	am.AddAccount(&Account{UserID: "user1"})
	am.AddAccount(&Account{UserID: "user3"})

	accounts := am.ListAccounts()
	if len(accounts) != 3 {
		t.Fatalf("len(ListAccounts()) = %d, want 3", len(accounts))
	}

	// Should be sorted by user ID
	if accounts[0].UserID != "user1" || accounts[1].UserID != "user2" || accounts[2].UserID != "user3" {
		t.Error("accounts not sorted by UserID")
	}
}

func TestAccountManager_AccountCount(t *testing.T) {
	am := NewAccountManager()

	if am.AccountCount() != 0 {
		t.Errorf("AccountCount() = %d, want 0", am.AccountCount())
	}

	am.AddAccount(&Account{UserID: "user1"})
	am.AddAccount(&Account{UserID: "user2"})

	if am.AccountCount() != 2 {
		t.Errorf("AccountCount() = %d, want 2", am.AccountCount())
	}
}

func TestAccountManager_AddDevice(t *testing.T) {
	am := NewAccountManager()

	var addedDeviceID string
	am.OnDeviceAdded(func(userID string, device *AccountDevice) {
		addedDeviceID = device.DeviceID
	})

	device := &AccountDevice{
		DeviceID:   "dev1",
		DeviceType: "SHSW-1",
		Name:       "Test Device",
	}

	err := am.AddDevice("user1", device)
	if err != nil {
		t.Fatalf("AddDevice() error = %v", err)
	}

	if addedDeviceID != "dev1" {
		t.Errorf("addedDeviceID = %v, want dev1", addedDeviceID)
	}

	// Should have created user account
	account, ok := am.GetAccount("user1")
	if !ok {
		t.Error("GetAccount() returned false")
	}
	if len(account.Devices) != 1 {
		t.Errorf("len(Devices) = %d, want 1", len(account.Devices))
	}

	// Update existing device
	device2 := &AccountDevice{
		DeviceID:   "dev1",
		DeviceType: "SHSW-1",
		Name:       "Updated Name",
	}
	err = am.AddDevice("user1", device2)
	if err != nil {
		t.Fatalf("AddDevice() error = %v", err)
	}

	account, _ = am.GetAccount("user1")
	if account.Devices[0].Name != "Updated Name" {
		t.Error("device not updated")
	}
}

func TestAccountManager_RemoveDevice(t *testing.T) {
	am := NewAccountManager()

	var removedDeviceID string
	am.OnDeviceRemoved(func(userID string, deviceID string) {
		removedDeviceID = deviceID
	})

	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev1"})
	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev2"})

	removed := am.RemoveDevice("user1", "dev1")
	if !removed {
		t.Error("RemoveDevice() = false, want true")
	}
	if removedDeviceID != "dev1" {
		t.Errorf("removedDeviceID = %v, want dev1", removedDeviceID)
	}

	account, _ := am.GetAccount("user1")
	if len(account.Devices) != 1 {
		t.Errorf("len(Devices) = %d, want 1", len(account.Devices))
	}

	// Remove from nonexistent user
	removed = am.RemoveDevice("nonexistent", "dev1")
	if removed {
		t.Error("RemoveDevice(nonexistent) = true, want false")
	}

	// Remove nonexistent device
	removed = am.RemoveDevice("user1", "nonexistent")
	if removed {
		t.Error("RemoveDevice(nonexistent device) = true, want false")
	}
}

func TestAccountManager_GetDevice(t *testing.T) {
	am := NewAccountManager()

	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev1", Name: "Device 1"})

	device, account, ok := am.GetDevice("dev1")
	if !ok {
		t.Error("GetDevice() returned false")
	}
	if device.Name != "Device 1" {
		t.Errorf("Name = %v, want Device 1", device.Name)
	}
	if account.UserID != "user1" {
		t.Errorf("UserID = %v, want user1", account.UserID)
	}

	// Nonexistent device
	_, _, ok = am.GetDevice("nonexistent")
	if ok {
		t.Error("GetDevice(nonexistent) should return false")
	}
}

func TestAccountManager_ListDevices(t *testing.T) {
	am := NewAccountManager()

	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev2"})
	_ = am.AddDevice("user2", &AccountDevice{DeviceID: "dev1"})
	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev3"})

	devices := am.ListDevices()
	if len(devices) != 3 {
		t.Fatalf("len(ListDevices()) = %d, want 3", len(devices))
	}

	// Should be sorted by device ID
	if devices[0].DeviceID != "dev1" || devices[1].DeviceID != "dev2" || devices[2].DeviceID != "dev3" {
		t.Error("devices not sorted by DeviceID")
	}
}

func TestAccountManager_DeviceCount(t *testing.T) {
	am := NewAccountManager()

	if am.DeviceCount() != 0 {
		t.Errorf("DeviceCount() = %d, want 0", am.DeviceCount())
	}

	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev1"})
	_ = am.AddDevice("user2", &AccountDevice{DeviceID: "dev2"})

	if am.DeviceCount() != 2 {
		t.Errorf("DeviceCount() = %d, want 2", am.DeviceCount())
	}
}

func TestAccountManager_GetDevicesByHost(t *testing.T) {
	am := NewAccountManager()

	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev1", Host: "host1"})
	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev2", Host: "host2"})
	_ = am.AddDevice("user2", &AccountDevice{DeviceID: "dev3", Host: "host1"})

	devices := am.GetDevicesByHost("host1")
	if len(devices) != 2 {
		t.Errorf("len(GetDevicesByHost) = %d, want 2", len(devices))
	}

	devices = am.GetDevicesByHost("unknown")
	if len(devices) != 0 {
		t.Errorf("len(GetDevicesByHost(unknown)) = %d, want 0", len(devices))
	}
}

func TestAccountManager_GetDevicesByType(t *testing.T) {
	am := NewAccountManager()

	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev1", DeviceType: "SHSW-1"})
	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev2", DeviceType: "SHPLG-S"})
	_ = am.AddDevice("user2", &AccountDevice{DeviceID: "dev3", DeviceType: "SHSW-1"})

	devices := am.GetDevicesByType("SHSW-1")
	if len(devices) != 2 {
		t.Errorf("len(GetDevicesByType) = %d, want 2", len(devices))
	}
}

func TestAccountManager_GetOnlineDevices(t *testing.T) {
	am := NewAccountManager()

	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev1", Online: true})
	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev2", Online: false})
	_ = am.AddDevice("user2", &AccountDevice{DeviceID: "dev3", Online: true})

	devices := am.GetOnlineDevices()
	if len(devices) != 2 {
		t.Errorf("len(GetOnlineDevices) = %d, want 2", len(devices))
	}
}

func TestAccountManager_GetControllableDevices(t *testing.T) {
	am := NewAccountManager()

	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev1", AccessGroups: "01"})
	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev2", AccessGroups: "00"})
	_ = am.AddDevice("user2", &AccountDevice{DeviceID: "dev3", AccessGroups: "01"})

	devices := am.GetControllableDevices()
	if len(devices) != 2 {
		t.Errorf("len(GetControllableDevices) = %d, want 2", len(devices))
	}
}

func TestAccountManager_UpdateDeviceOnlineStatus(t *testing.T) {
	am := NewAccountManager()

	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev1", Online: false})

	updated := am.UpdateDeviceOnlineStatus("dev1", true)
	if !updated {
		t.Error("UpdateDeviceOnlineStatus() = false, want true")
	}

	device, _, _ := am.GetDevice("dev1")
	if !device.Online {
		t.Error("device.Online = false, want true")
	}

	account, _ := am.GetAccount("user1")
	if account.LastActivityAt == nil {
		t.Error("LastActivityAt not set")
	}

	// Update nonexistent
	updated = am.UpdateDeviceOnlineStatus("nonexistent", true)
	if updated {
		t.Error("UpdateDeviceOnlineStatus(nonexistent) = true, want false")
	}
}

func TestAccountManager_ToFromJSON(t *testing.T) {
	am := NewAccountManager()

	am.AddAccount(&Account{
		UserID: "user1",
		Email:  "user1@example.com",
		Devices: []AccountDevice{
			{DeviceID: "dev1", Name: "Device 1"},
		},
		GrantedAt: time.Now(),
	})

	data, err := am.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	am2 := NewAccountManager()
	err = am2.FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if am2.AccountCount() != 1 {
		t.Errorf("AccountCount() = %d, want 1", am2.AccountCount())
	}
	if am2.DeviceCount() != 1 {
		t.Errorf("DeviceCount() = %d, want 1", am2.DeviceCount())
	}

	account, _ := am2.GetAccount("user1")
	if account.Email != "user1@example.com" {
		t.Errorf("Email = %v, want user1@example.com", account.Email)
	}
}

func TestAccountManager_FromJSON_Invalid(t *testing.T) {
	am := NewAccountManager()

	err := am.FromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("FromJSON() should error for invalid JSON")
	}
}

func TestDeviceCallback_IsAddAction(t *testing.T) {
	dc := &DeviceCallback{Action: "add"}
	if !dc.IsAddAction() {
		t.Error("IsAddAction() = false, want true")
	}
	if dc.IsRemoveAction() {
		t.Error("IsRemoveAction() = true, want false")
	}

	dc.Action = "remove"
	if dc.IsAddAction() {
		t.Error("IsAddAction() = true, want false")
	}
	if !dc.IsRemoveAction() {
		t.Error("IsRemoveAction() = false, want true")
	}
}

func TestDeviceCallback_ToAccountDevice(t *testing.T) {
	dc := &DeviceCallback{
		DeviceID:     "dev1",
		DeviceType:   "SHSW-1",
		DeviceCode:   "code",
		Name:         "Test Device",
		Host:         "host1",
		AccessGroups: "01",
	}

	device := dc.ToAccountDevice()
	if device.DeviceID != "dev1" {
		t.Errorf("DeviceID = %v, want dev1", device.DeviceID)
	}
	if device.Name != "Test Device" {
		t.Errorf("Name = %v, want Test Device", device.Name)
	}
	if device.Online {
		t.Error("Online = true, want false")
	}
}

func TestAccountManager_ProcessCallback(t *testing.T) {
	am := NewAccountManager()

	// Add callback
	callback := &DeviceCallback{
		UserID:       "user1",
		DeviceID:     "dev1",
		DeviceType:   "SHSW-1",
		Name:         "Test Device",
		Host:         "host1",
		AccessGroups: "01",
		Action:       "add",
	}

	err := am.ProcessCallback(callback)
	if err != nil {
		t.Fatalf("ProcessCallback(add) error = %v", err)
	}

	if am.DeviceCount() != 1 {
		t.Errorf("DeviceCount() = %d, want 1", am.DeviceCount())
	}

	// Remove callback
	callback.Action = "remove"
	err = am.ProcessCallback(callback)
	if err != nil {
		t.Fatalf("ProcessCallback(remove) error = %v", err)
	}

	if am.DeviceCount() != 0 {
		t.Errorf("DeviceCount() = %d, want 0", am.DeviceCount())
	}

	// Unknown action
	callback.Action = "unknown"
	err = am.ProcessCallback(callback)
	if err == nil {
		t.Error("ProcessCallback(unknown) should error")
	}
}

func TestGetConsentURL(t *testing.T) {
	url := GetConsentURL("my-tag", "https://callback.example.com")
	expected := "https://my.shelly.cloud/integrator.html?itg=my-tag&cb=https://callback.example.com"
	if url != expected {
		t.Errorf("GetConsentURL() = %v, want %v", url, expected)
	}
}

func TestAccountManager_GetStats(t *testing.T) {
	am := NewAccountManager()

	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev1", DeviceType: "SHSW-1", Host: "host1", Online: true, AccessGroups: "01"})
	_ = am.AddDevice("user1", &AccountDevice{DeviceID: "dev2", DeviceType: "SHSW-1", Host: "host1", Online: false, AccessGroups: "00"})
	_ = am.AddDevice("user2", &AccountDevice{DeviceID: "dev3", DeviceType: "SHPLG-S", Host: "host2", Online: true, AccessGroups: "01"})

	stats := am.GetStats()

	if stats.TotalAccounts != 2 {
		t.Errorf("TotalAccounts = %d, want 2", stats.TotalAccounts)
	}
	if stats.TotalDevices != 3 {
		t.Errorf("TotalDevices = %d, want 3", stats.TotalDevices)
	}
	if stats.OnlineDevices != 2 {
		t.Errorf("OnlineDevices = %d, want 2", stats.OnlineDevices)
	}
	if stats.ControllableDevices != 2 {
		t.Errorf("ControllableDevices = %d, want 2", stats.ControllableDevices)
	}
	if stats.DevicesByType["SHSW-1"] != 2 {
		t.Errorf("DevicesByType[SHSW-1] = %d, want 2", stats.DevicesByType["SHSW-1"])
	}
	if stats.DevicesByHost["host1"] != 2 {
		t.Errorf("DevicesByHost[host1] = %d, want 2", stats.DevicesByHost["host1"])
	}
}

func TestAccount_Serialization(t *testing.T) {
	account := &Account{
		UserID: "user1",
		Email:  "test@example.com",
		Devices: []AccountDevice{
			{DeviceID: "dev1", Name: "Device 1"},
		},
		GrantedAt: time.Now(),
	}

	data, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded Account
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.UserID != account.UserID {
		t.Errorf("UserID = %v, want %v", decoded.UserID, account.UserID)
	}
	if decoded.Email != account.Email {
		t.Errorf("Email = %v, want %v", decoded.Email, account.Email)
	}
}
