package testutil

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/tj-smith47/shelly-go/types"
)

// MockGen1Device is a mock Gen1 device for testing.
type MockGen1Device struct {
	deviceInfo   *types.DeviceInfo
	relayStates  map[int]bool
	rollerStates map[int]int
	meterPowers  map[int]float64
	settings     map[string]any
	errors       map[string]error
	address      string
	mu           sync.RWMutex
}

// NewMockGen1Device creates a new mock Gen1 device.
func NewMockGen1Device(address string) *MockGen1Device {
	return &MockGen1Device{
		address: address,
		deviceInfo: &types.DeviceInfo{
			ID:         "shelly1-aabbcc",
			Model:      "SHSW-1",
			Generation: types.Gen1,
			Version:    "1.11.0",
			MAC:        "AA:BB:CC:DD:EE:FF",
		},
		relayStates:  make(map[int]bool),
		rollerStates: make(map[int]int),
		meterPowers:  make(map[int]float64),
		settings:     make(map[string]any),
		errors:       make(map[string]error),
	}
}

// Address returns the device address.
func (d *MockGen1Device) Address() string {
	return d.address
}

// Generation returns Gen1.
func (d *MockGen1Device) Generation() types.Generation {
	return types.Gen1
}

// Info returns the device info.
func (d *MockGen1Device) Info(ctx context.Context) (*types.DeviceInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if err, ok := d.errors["Info"]; ok {
		return nil, err
	}
	return d.deviceInfo, nil
}

// SetDeviceInfo sets the device info for testing.
func (d *MockGen1Device) SetDeviceInfo(info *types.DeviceInfo) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.deviceInfo = info
}

// SetRelayState sets a relay state for testing.
func (d *MockGen1Device) SetRelayState(id int, on bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.relayStates[id] = on
}

// GetRelayState gets a relay state.
func (d *MockGen1Device) GetRelayState(id int) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.relayStates[id]
}

// SetRollerPosition sets a roller position for testing.
func (d *MockGen1Device) SetRollerPosition(id, pos int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.rollerStates[id] = pos
}

// GetRollerPosition gets a roller position.
func (d *MockGen1Device) GetRollerPosition(id int) int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.rollerStates[id]
}

// SetMeterPower sets meter power for testing.
func (d *MockGen1Device) SetMeterPower(id int, power float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.meterPowers[id] = power
}

// GetMeterPower gets meter power.
func (d *MockGen1Device) GetMeterPower(id int) float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.meterPowers[id]
}

// SetError sets an error to be returned for a method.
func (d *MockGen1Device) SetError(method string, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err == nil {
		delete(d.errors, method)
	} else {
		d.errors[method] = err
	}
}

// MockGen2Device is a mock Gen2+ device for testing.
type MockGen2Device struct {
	lightStates  map[int]*MockLightStatus
	config       map[string]any
	transport    *MockTransport
	deviceInfo   *types.DeviceInfo
	switchStates map[int]*MockSwitchStatus
	coverStates  map[int]*MockCoverStatus
	scripts      map[int]*MockScript
	errors       map[string]error
	inputStates  map[int]*MockInputStatus
	schedules    map[int]*MockSchedule
	webhooks     map[int]*MockWebhook
	kvsData      map[string]any
	address      string
	mu           sync.RWMutex
}

// MockSwitchStatus represents a switch status for testing.
type MockSwitchStatus struct {
	Output      bool
	APower      float64
	Voltage     float64
	Current     float64
	Temperature float64
}

// MockCoverStatus represents a cover status for testing.
type MockCoverStatus struct {
	State         string // "open", "closed", "opening", "closing", "stopped"
	CurrentPos    int
	TargetPos     int
	APower        float64
	IsCalibrating bool
}

// MockLightStatus represents a light status for testing.
type MockLightStatus struct {
	Output     bool
	Brightness int
	ColorTemp  int
}

// MockInputStatus represents an input status for testing.
type MockInputStatus struct {
	Type  string
	State bool
}

// MockScript represents a script for testing.
type MockScript struct {
	Name    string
	Code    string
	ID      int
	Enable  bool
	Running bool
}

// MockSchedule represents a schedule for testing.
type MockSchedule struct {
	Timespec string
	Calls    []any
	ID       int
	Enable   bool
}

// MockWebhook represents a webhook for testing.
type MockWebhook struct {
	Event  string
	Name   string
	URLs   []string
	ID     int
	Enable bool
}

// NewMockGen2Device creates a new mock Gen2+ device.
func NewMockGen2Device(address string) *MockGen2Device {
	return &MockGen2Device{
		address:   address,
		transport: NewMockTransport(),
		deviceInfo: &types.DeviceInfo{
			ID:         "shellyplus1-aabbcc",
			Model:      "SNSW-001X16EU",
			Generation: types.Gen2,
			Version:    "1.0.0",
			MAC:        "AA:BB:CC:DD:EE:FF",
		},
		switchStates: make(map[int]*MockSwitchStatus),
		coverStates:  make(map[int]*MockCoverStatus),
		lightStates:  make(map[int]*MockLightStatus),
		inputStates:  make(map[int]*MockInputStatus),
		scripts:      make(map[int]*MockScript),
		schedules:    make(map[int]*MockSchedule),
		webhooks:     make(map[int]*MockWebhook),
		kvsData:      make(map[string]any),
		config:       make(map[string]any),
		errors:       make(map[string]error),
	}
}

// Address returns the device address.
func (d *MockGen2Device) Address() string {
	return d.address
}

// Generation returns Gen2.
func (d *MockGen2Device) Generation() types.Generation {
	return types.Gen2
}

// Transport returns the mock transport.
func (d *MockGen2Device) Transport() *MockTransport {
	return d.transport
}

// Info returns the device info.
func (d *MockGen2Device) Info(ctx context.Context) (*types.DeviceInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if err, ok := d.errors["Info"]; ok {
		return nil, err
	}
	return d.deviceInfo, nil
}

// SetDeviceInfo sets the device info for testing.
func (d *MockGen2Device) SetDeviceInfo(info *types.DeviceInfo) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.deviceInfo = info
}

// SetSwitchStatus sets a switch status for testing.
func (d *MockGen2Device) SetSwitchStatus(id int, status *MockSwitchStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.switchStates[id] = status
}

// GetSwitchStatus gets a switch status.
func (d *MockGen2Device) GetSwitchStatus(id int) *MockSwitchStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.switchStates[id]
}

// SetCoverStatus sets a cover status for testing.
func (d *MockGen2Device) SetCoverStatus(id int, status *MockCoverStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.coverStates[id] = status
}

// GetCoverStatus gets a cover status.
func (d *MockGen2Device) GetCoverStatus(id int) *MockCoverStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.coverStates[id]
}

// SetLightStatus sets a light status for testing.
func (d *MockGen2Device) SetLightStatus(id int, status *MockLightStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lightStates[id] = status
}

// GetLightStatus gets a light status.
func (d *MockGen2Device) GetLightStatus(id int) *MockLightStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.lightStates[id]
}

// SetInputStatus sets an input status for testing.
func (d *MockGen2Device) SetInputStatus(id int, status *MockInputStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.inputStates[id] = status
}

// GetInputStatus gets an input status.
func (d *MockGen2Device) GetInputStatus(id int) *MockInputStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.inputStates[id]
}

// AddScript adds a script for testing.
func (d *MockGen2Device) AddScript(script *MockScript) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.scripts[script.ID] = script
}

// GetScript gets a script.
func (d *MockGen2Device) GetScript(id int) *MockScript {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.scripts[id]
}

// AddSchedule adds a schedule for testing.
func (d *MockGen2Device) AddSchedule(schedule *MockSchedule) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.schedules[schedule.ID] = schedule
}

// GetSchedule gets a schedule.
func (d *MockGen2Device) GetSchedule(id int) *MockSchedule {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.schedules[id]
}

// AddWebhook adds a webhook for testing.
func (d *MockGen2Device) AddWebhook(webhook *MockWebhook) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.webhooks[webhook.ID] = webhook
}

// GetWebhook gets a webhook.
func (d *MockGen2Device) GetWebhook(id int) *MockWebhook {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.webhooks[id]
}

// SetKVS sets a KVS value for testing.
func (d *MockGen2Device) SetKVS(key string, value any) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.kvsData[key] = value
}

// GetKVS gets a KVS value.
func (d *MockGen2Device) GetKVS(key string) any {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.kvsData[key]
}

// SetConfig sets a config value for testing.
func (d *MockGen2Device) SetConfig(key string, value any) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.config[key] = value
}

// GetConfig gets a config value.
func (d *MockGen2Device) GetConfig(key string) any {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config[key]
}

// SetError sets an error to be returned for a method.
func (d *MockGen2Device) SetError(method string, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err == nil {
		delete(d.errors, method)
	} else {
		d.errors[method] = err
	}
}

// ToJSON converts a value to JSON.RawMessage.
func ToJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return data
}
