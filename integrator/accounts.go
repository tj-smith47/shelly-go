package integrator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// Account represents a user account that has granted access to the integrator.
type Account struct {
	GrantedAt      time.Time  `json:"granted_at"`
	LastActivityAt *time.Time `json:"last_activity_at,omitempty"`
	types.RawFields
	UserID  string          `json:"user_id"`
	Email   string          `json:"email,omitempty"`
	Name    string          `json:"name,omitempty"`
	Devices []AccountDevice `json:"devices"`
}

// AccountDevice represents a device within a user account.
type AccountDevice struct {
	GrantedAt time.Time `json:"granted_at"`
	types.RawFields
	DeviceID     string `json:"device_id"`
	DeviceType   string `json:"device_type"`
	DeviceCode   string `json:"device_code,omitempty"`
	Name         string `json:"name"`
	AccessGroups string `json:"access_groups"`
	Host         string `json:"host"`
	Online       bool   `json:"online"`
}

// CanControl returns true if the integrator has control access to this device.
func (d *AccountDevice) CanControl() bool {
	if len(d.AccessGroups) >= 2 {
		return d.AccessGroups[1] == '1'
	}
	return false
}

// AccountManager manages user accounts that have granted access to the integrator.
type AccountManager struct {
	accounts         map[string]*Account
	deviceIndex      map[string]string
	onAccountAdded   func(*Account)
	onAccountRemoved func(userID string)
	onDeviceAdded    func(userID string, device *AccountDevice)
	onDeviceRemoved  func(userID string, deviceID string)
	mu               sync.RWMutex
}

// NewAccountManager creates a new account manager.
func NewAccountManager() *AccountManager {
	return &AccountManager{
		accounts:    make(map[string]*Account),
		deviceIndex: make(map[string]string),
	}
}

// AddAccount adds or updates a user account.
func (am *AccountManager) AddAccount(account *Account) {
	am.mu.Lock()
	defer am.mu.Unlock()

	existing, exists := am.accounts[account.UserID]
	am.accounts[account.UserID] = account

	// Update device index
	for i := range account.Devices {
		am.deviceIndex[account.Devices[i].DeviceID] = account.UserID
	}

	if !exists && am.onAccountAdded != nil {
		am.onAccountAdded(account)
	} else if exists {
		// Check for removed devices
		existingDevices := make(map[string]bool)
		for i := range existing.Devices {
			existingDevices[existing.Devices[i].DeviceID] = true
		}
		for i := range account.Devices {
			delete(existingDevices, account.Devices[i].DeviceID)
		}
		// Remove devices no longer present
		for deviceID := range existingDevices {
			delete(am.deviceIndex, deviceID)
		}
	}
}

// RemoveAccount removes a user account.
func (am *AccountManager) RemoveAccount(userID string) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	account, exists := am.accounts[userID]
	if !exists {
		return false
	}

	// Remove device index entries
	for i := range account.Devices {
		device := &account.Devices[i]
		delete(am.deviceIndex, device.DeviceID)
	}

	delete(am.accounts, userID)

	if am.onAccountRemoved != nil {
		am.onAccountRemoved(userID)
	}

	return true
}

// GetAccount returns an account by user ID.
func (am *AccountManager) GetAccount(userID string) (*Account, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	account, exists := am.accounts[userID]
	return account, exists
}

// ListAccounts returns all accounts.
func (am *AccountManager) ListAccounts() []*Account {
	am.mu.RLock()
	defer am.mu.RUnlock()

	accounts := make([]*Account, 0, len(am.accounts))
	for _, account := range am.accounts {
		accounts = append(accounts, account)
	}

	// Sort by user ID for consistent ordering
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].UserID < accounts[j].UserID
	})

	return accounts
}

// AccountCount returns the number of accounts.
func (am *AccountManager) AccountCount() int {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return len(am.accounts)
}

// AddDevice adds a device to an account.
func (am *AccountManager) AddDevice(userID string, device *AccountDevice) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	account, exists := am.accounts[userID]
	if !exists {
		// Create new account
		account = &Account{
			UserID:    userID,
			Devices:   []AccountDevice{},
			GrantedAt: time.Now(),
		}
		am.accounts[userID] = account
	}

	// Check if device already exists
	for i := range account.Devices {
		if account.Devices[i].DeviceID == device.DeviceID {
			// Update existing device
			account.Devices[i] = *device
			am.deviceIndex[device.DeviceID] = userID
			return nil
		}
	}

	// Add new device
	account.Devices = append(account.Devices, *device)
	am.deviceIndex[device.DeviceID] = userID

	if am.onDeviceAdded != nil {
		am.onDeviceAdded(userID, device)
	}

	return nil
}

// RemoveDevice removes a device from an account.
func (am *AccountManager) RemoveDevice(userID, deviceID string) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	account, exists := am.accounts[userID]
	if !exists {
		return false
	}

	for i := range account.Devices {
		if account.Devices[i].DeviceID == deviceID {
			account.Devices = append(account.Devices[:i], account.Devices[i+1:]...)
			delete(am.deviceIndex, deviceID)

			if am.onDeviceRemoved != nil {
				am.onDeviceRemoved(userID, deviceID)
			}
			return true
		}
	}

	return false
}

// GetDevice returns a device by ID.
func (am *AccountManager) GetDevice(deviceID string) (*AccountDevice, *Account, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	userID, exists := am.deviceIndex[deviceID]
	if !exists {
		return nil, nil, false
	}

	account := am.accounts[userID]
	for i := range account.Devices {
		if account.Devices[i].DeviceID == deviceID {
			return &account.Devices[i], account, true
		}
	}

	return nil, nil, false
}

// ListDevices returns all devices across all accounts.
func (am *AccountManager) ListDevices() []AccountDevice {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var devices []AccountDevice
	for _, account := range am.accounts {
		devices = append(devices, account.Devices...)
	}

	// Sort by device ID for consistent ordering
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].DeviceID < devices[j].DeviceID
	})

	return devices
}

// DeviceCount returns the total number of devices across all accounts.
func (am *AccountManager) DeviceCount() int {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return len(am.deviceIndex)
}

// GetDevicesByHost returns all devices on a specific host.
func (am *AccountManager) GetDevicesByHost(host string) []AccountDevice {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var devices []AccountDevice
	for _, account := range am.accounts {
		for i := range account.Devices {
			if account.Devices[i].Host == host {
				devices = append(devices, account.Devices[i])
			}
		}
	}
	return devices
}

// GetDevicesByType returns all devices of a specific type.
func (am *AccountManager) GetDevicesByType(deviceType string) []AccountDevice {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var devices []AccountDevice
	for _, account := range am.accounts {
		for i := range account.Devices {
			if account.Devices[i].DeviceType == deviceType {
				devices = append(devices, account.Devices[i])
			}
		}
	}
	return devices
}

// GetOnlineDevices returns all currently online devices.
func (am *AccountManager) GetOnlineDevices() []AccountDevice {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var devices []AccountDevice
	for _, account := range am.accounts {
		for i := range account.Devices {
			if account.Devices[i].Online {
				devices = append(devices, account.Devices[i])
			}
		}
	}
	return devices
}

// GetControllableDevices returns all devices with control access.
func (am *AccountManager) GetControllableDevices() []AccountDevice {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var devices []AccountDevice
	for _, account := range am.accounts {
		for i := range account.Devices {
			if account.Devices[i].CanControl() {
				devices = append(devices, account.Devices[i])
			}
		}
	}
	return devices
}

// UpdateDeviceOnlineStatus updates the online status of a device.
func (am *AccountManager) UpdateDeviceOnlineStatus(deviceID string, online bool) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	userID, exists := am.deviceIndex[deviceID]
	if !exists {
		return false
	}

	account := am.accounts[userID]
	for i := range account.Devices {
		if account.Devices[i].DeviceID == deviceID {
			account.Devices[i].Online = online
			now := time.Now()
			account.LastActivityAt = &now
			return true
		}
	}

	return false
}

// OnAccountAdded sets a callback for when an account is added.
func (am *AccountManager) OnAccountAdded(callback func(*Account)) {
	am.mu.Lock()
	am.onAccountAdded = callback
	am.mu.Unlock()
}

// OnAccountRemoved sets a callback for when an account is removed.
func (am *AccountManager) OnAccountRemoved(callback func(userID string)) {
	am.mu.Lock()
	am.onAccountRemoved = callback
	am.mu.Unlock()
}

// OnDeviceAdded sets a callback for when a device is added.
func (am *AccountManager) OnDeviceAdded(callback func(userID string, device *AccountDevice)) {
	am.mu.Lock()
	am.onDeviceAdded = callback
	am.mu.Unlock()
}

// OnDeviceRemoved sets a callback for when a device is removed.
func (am *AccountManager) OnDeviceRemoved(callback func(userID string, deviceID string)) {
	am.mu.Lock()
	am.onDeviceRemoved = callback
	am.mu.Unlock()
}

// ToJSON serializes the account manager state to JSON.
func (am *AccountManager) ToJSON() ([]byte, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	accounts := make([]*Account, 0, len(am.accounts))
	for _, a := range am.accounts {
		accounts = append(accounts, a)
	}

	return json.Marshal(accounts)
}

// FromJSON loads account manager state from JSON.
func (am *AccountManager) FromJSON(data []byte) error {
	var accounts []*Account
	if err := json.Unmarshal(data, &accounts); err != nil {
		return fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	am.accounts = make(map[string]*Account)
	am.deviceIndex = make(map[string]string)

	for _, account := range accounts {
		am.accounts[account.UserID] = account
		for i := range account.Devices {
			device := &account.Devices[i]
			am.deviceIndex[device.DeviceID] = account.UserID
		}
	}

	return nil
}

// DeviceCallback represents a callback notification from Shelly
// when a user grants or revokes device access.
type DeviceCallback struct {
	types.RawFields
	UserID       string `json:"userId"`
	DeviceID     string `json:"deviceId"`
	DeviceType   string `json:"deviceType"`
	DeviceCode   string `json:"deviceCode"`
	AccessGroups string `json:"accessGroups"`
	Action       string `json:"action"`
	Host         string `json:"host"`
	Name         string `json:"name"`
	Token        string `json:"token,omitempty"`
}

// IsAddAction returns true if this is an add/grant action.
func (dc *DeviceCallback) IsAddAction() bool {
	return dc.Action == "add"
}

// IsRemoveAction returns true if this is a remove/revoke action.
func (dc *DeviceCallback) IsRemoveAction() bool {
	return dc.Action == "remove"
}

// ToAccountDevice converts the callback to an AccountDevice.
func (dc *DeviceCallback) ToAccountDevice() *AccountDevice {
	return &AccountDevice{
		DeviceID:     dc.DeviceID,
		DeviceType:   dc.DeviceType,
		DeviceCode:   dc.DeviceCode,
		Name:         dc.Name,
		AccessGroups: dc.AccessGroups,
		Host:         dc.Host,
		GrantedAt:    time.Now(),
		Online:       false, // Unknown at callback time
	}
}

// ProcessCallback processes a device callback and updates the account manager.
func (am *AccountManager) ProcessCallback(callback *DeviceCallback) error {
	if callback.IsAddAction() {
		device := callback.ToAccountDevice()
		return am.AddDevice(callback.UserID, device)
	} else if callback.IsRemoveAction() {
		am.RemoveDevice(callback.UserID, callback.DeviceID)
		return nil
	}
	return fmt.Errorf("unknown callback action: %s", callback.Action)
}

// UnsubscribeDevice requests Shelly to remove a device from the integrator's access.
// This is done via the /integrator/unsubscribe_device endpoint.
func (c *Client) UnsubscribeDevice(ctx context.Context, host, deviceID string) error {
	token, err := c.GetToken()
	if err != nil {
		return err
	}

	// Build the unsubscribe URL for the specific host
	reqURL := fmt.Sprintf("https://%s/integrator/unsubscribe_device", host)

	// Create form data
	formData := url.Values{}
	formData.Set("id", deviceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unsubscribe request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unsubscribe failed: status %d", resp.StatusCode)
	}

	return nil
}

// GetConsentURL returns the URL for users to grant device access.
func GetConsentURL(integratorTag, callbackURL string) string {
	return fmt.Sprintf(
		"https://my.shelly.cloud/integrator.html?itg=%s&cb=%s",
		integratorTag,
		callbackURL,
	)
}

// AccountStats contains statistics about managed accounts.
type AccountStats struct {
	DevicesByType map[string]int `json:"devices_by_type"`
	DevicesByHost map[string]int `json:"devices_by_host"`
	types.RawFields
	TotalAccounts       int `json:"total_accounts"`
	TotalDevices        int `json:"total_devices"`
	OnlineDevices       int `json:"online_devices"`
	ControllableDevices int `json:"controllable_devices"`
}

// GetStats returns statistics about managed accounts.
func (am *AccountManager) GetStats() *AccountStats {
	am.mu.RLock()
	defer am.mu.RUnlock()

	stats := &AccountStats{
		TotalAccounts: len(am.accounts),
		TotalDevices:  len(am.deviceIndex),
		DevicesByType: make(map[string]int),
		DevicesByHost: make(map[string]int),
	}

	for _, account := range am.accounts {
		for i := range account.Devices {
			device := &account.Devices[i]
			if device.Online {
				stats.OnlineDevices++
			}
			if device.CanControl() {
				stats.ControllableDevices++
			}
			stats.DevicesByType[device.DeviceType]++
			stats.DevicesByHost[device.Host]++
		}
	}

	return stats
}
