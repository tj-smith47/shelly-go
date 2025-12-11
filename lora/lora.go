package lora

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/rpc"
)

// LoRa provides access to the LoRa add-on component on Gen2+ Shelly devices.
// The LoRa component enables long-range, low-power wireless communication
// between Shelly devices and other LoRa-compatible devices.
type LoRa struct {
	client *rpc.Client
	id     int
}

// NewLoRa creates a new LoRa component instance with the specified component ID.
// The default LoRa component ID is typically 100.
func NewLoRa(client *rpc.Client, id int) *LoRa {
	return &LoRa{client: client, id: id}
}

// GetConfig retrieves the current LoRa configuration.
func (l *LoRa) GetConfig(ctx context.Context) (*Config, error) {
	params := map[string]any{
		"id": l.id,
	}
	result, err := l.client.Call(ctx, "LoRa.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SetConfig updates the LoRa configuration.
// Use this to modify radio parameters like frequency, bandwidth, and transmit power.
func (l *LoRa) SetConfig(ctx context.Context, params *SetConfigParams) error {
	reqParams := map[string]any{
		"id":     l.id,
		"config": params,
	}
	_, err := l.client.Call(ctx, "LoRa.SetConfig", reqParams)
	return err
}

// GetStatus retrieves the current LoRa status.
// The status includes signal quality information from the last received packet.
func (l *LoRa) GetStatus(ctx context.Context) (*Status, error) {
	params := map[string]any{
		"id": l.id,
	}
	result, err := l.client.Call(ctx, "LoRa.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status Status
	if err := json.Unmarshal(result, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// SendBytes sends data over LoRa RF.
// The data parameter must be base64-encoded.
//
// Example:
//
//	import "encoding/base64"
//	data := base64.StdEncoding.EncodeToString([]byte("Hello!"))
//	err := lora.SendBytes(ctx, data)
func (l *LoRa) SendBytes(ctx context.Context, data string) error {
	params := map[string]any{
		"id":   l.id,
		"data": data,
	}
	_, err := l.client.Call(ctx, "LoRa.SendBytes", params)
	return err
}

// SendString is a convenience method that encodes the string to base64
// and sends it over LoRa RF.
func (l *LoRa) SendString(ctx context.Context, s string) error {
	return l.SendBytes(ctx, encodeBase64([]byte(s)))
}

// SendRaw is a convenience method that encodes raw bytes to base64
// and sends them over LoRa RF.
func (l *LoRa) SendRaw(ctx context.Context, data []byte) error {
	return l.SendBytes(ctx, encodeBase64(data))
}

// GetAddOnInfo retrieves information about the LoRa add-on.
func (l *LoRa) GetAddOnInfo(ctx context.Context) (*AddOnInfo, error) {
	result, err := l.client.Call(ctx, "AddOn.GetInfo", nil)
	if err != nil {
		return nil, err
	}

	var info AddOnInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// CheckForUpdate checks if a firmware update is available for the LoRa add-on.
func (l *LoRa) CheckForUpdate(ctx context.Context) (bool, error) {
	result, err := l.client.Call(ctx, "AddOn.CheckForUpdate", nil)
	if err != nil {
		return false, err
	}

	var response struct {
		UpdateAvailable bool `json:"update_available"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return false, err
	}
	return response.UpdateAvailable, nil
}

// Update updates the LoRa add-on firmware if an update is available.
func (l *LoRa) Update(ctx context.Context) error {
	_, err := l.client.Call(ctx, "AddOn.Update", nil)
	return err
}

// ID returns the component ID.
func (l *LoRa) ID() int {
	return l.id
}

// SetFrequency sets the RF frequency in Hz.
// This is a convenience method that calls SetConfig.
func (l *LoRa) SetFrequency(ctx context.Context, freq int64) error {
	return l.SetConfig(ctx, &SetConfigParams{Freq: &freq})
}

// SetTransmitPower sets the transmit power in dBm.
// This is a convenience method that calls SetConfig.
func (l *LoRa) SetTransmitPower(ctx context.Context, txp int) error {
	return l.SetConfig(ctx, &SetConfigParams{TxP: &txp})
}

// SetDataRate sets the data rate (spreading factor).
// This is a convenience method that calls SetConfig.
func (l *LoRa) SetDataRate(ctx context.Context, dr int) error {
	return l.SetConfig(ctx, &SetConfigParams{DR: &dr})
}

// SetBandwidth sets the bandwidth setting.
// This is a convenience method that calls SetConfig.
func (l *LoRa) SetBandwidth(ctx context.Context, bw int) error {
	return l.SetConfig(ctx, &SetConfigParams{BW: &bw})
}

// GetFrequency returns the current RF frequency in Hz.
func (l *LoRa) GetFrequency(ctx context.Context) (int64, error) {
	config, err := l.GetConfig(ctx)
	if err != nil {
		return 0, err
	}
	return config.Freq, nil
}

// GetTransmitPower returns the current transmit power in dBm.
func (l *LoRa) GetTransmitPower(ctx context.Context) (int, error) {
	config, err := l.GetConfig(ctx)
	if err != nil {
		return 0, err
	}
	return config.TxP, nil
}

// GetLastRSSI returns the RSSI of the last received packet.
func (l *LoRa) GetLastRSSI(ctx context.Context) (int, error) {
	status, err := l.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.RSSI, nil
}

// GetLastSNR returns the SNR of the last received packet.
func (l *LoRa) GetLastSNR(ctx context.Context) (float64, error) {
	status, err := l.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.SNR, nil
}

// encodeBase64 encodes data to base64.
func encodeBase64(data []byte) string {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	result := make([]byte, 0, ((len(data)+2)/3)*4)

	for i := 0; i < len(data); i += 3 {
		var n uint32
		remaining := len(data) - i

		switch remaining {
		case 1:
			n = uint32(data[i]) << 16
			result = append(result,
				base64Chars[n>>18&0x3F],
				base64Chars[n>>12&0x3F],
				'=',
				'=')
		case 2:
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8
			result = append(result,
				base64Chars[n>>18&0x3F],
				base64Chars[n>>12&0x3F],
				base64Chars[n>>6&0x3F],
				'=')
		default:
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
			result = append(result,
				base64Chars[n>>18&0x3F],
				base64Chars[n>>12&0x3F],
				base64Chars[n>>6&0x3F],
				base64Chars[n&0x3F])
		}
	}

	return string(result)
}

// Common errors for LoRa operations.
var (
	// ErrDeviceNotFound indicates the device is not registered.
	ErrDeviceNotFound = errors.New("device not found in registry")

	// ErrDeviceAlreadyRegistered indicates the device is already registered.
	ErrDeviceAlreadyRegistered = errors.New("device already registered")

	// ErrRouterStopped indicates the router has been stopped.
	ErrRouterStopped = errors.New("router has been stopped")

	// ErrNoHandler indicates no handler is registered for the message type.
	ErrNoHandler = errors.New("no handler registered")
)

// DeviceRegistry manages registered LoRa devices.
type DeviceRegistry struct {
	devices       map[string]*RegisteredDevice
	OnlineTimeout time.Duration
	mu            sync.RWMutex
}

// NewDeviceRegistry creates a new device registry.
func NewDeviceRegistry() *DeviceRegistry {
	return &DeviceRegistry{
		devices:       make(map[string]*RegisteredDevice),
		OnlineTimeout: 5 * time.Minute,
	}
}

// Register adds a new device to the registry.
// Returns ErrDeviceAlreadyRegistered if the device ID is already registered.
func (r *DeviceRegistry) Register(device *RegisteredDevice) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.devices[device.DeviceID]; exists {
		return ErrDeviceAlreadyRegistered
	}

	r.devices[device.DeviceID] = device
	return nil
}

// RegisterOrUpdate registers a new device or updates an existing one.
func (r *DeviceRegistry) RegisterOrUpdate(device *RegisteredDevice) {
	r.mu.Lock()
	defer r.mu.Unlock()

	//nolint:nestif // Device update logic needs to check each field individually
	if existing, exists := r.devices[device.DeviceID]; exists {
		// Preserve existing metadata and update other fields
		if device.Name != "" {
			existing.Name = device.Name
		}
		if device.Group != "" {
			existing.Group = device.Group
		}
		if device.Address != "" {
			existing.Address = device.Address
		}
		if device.Metadata != nil {
			if existing.Metadata == nil {
				existing.Metadata = make(map[string]any)
			}
			for k, v := range device.Metadata {
				existing.Metadata[k] = v
			}
		}
		return
	}

	r.devices[device.DeviceID] = device
}

// Unregister removes a device from the registry.
func (r *DeviceRegistry) Unregister(deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.devices[deviceID]; !exists {
		return ErrDeviceNotFound
	}

	delete(r.devices, deviceID)
	return nil
}

// Get retrieves a device by its ID.
func (r *DeviceRegistry) Get(deviceID string) (*RegisteredDevice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	device, exists := r.devices[deviceID]
	if !exists {
		return nil, ErrDeviceNotFound
	}
	return device, nil
}

// GetAll returns all registered devices.
func (r *DeviceRegistry) GetAll() []*RegisteredDevice {
	r.mu.RLock()
	defer r.mu.RUnlock()

	devices := make([]*RegisteredDevice, 0, len(r.devices))
	for _, device := range r.devices {
		devices = append(devices, device)
	}
	return devices
}

// GetByGroup returns all devices in a specific group.
func (r *DeviceRegistry) GetByGroup(group string) []*RegisteredDevice {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var devices []*RegisteredDevice
	for _, device := range r.devices {
		if device.Group == group {
			devices = append(devices, device)
		}
	}
	return devices
}

// GetOnline returns all devices that are currently online.
func (r *DeviceRegistry) GetOnline() []*RegisteredDevice {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := float64(time.Now().Unix())
	var devices []*RegisteredDevice
	for _, device := range r.devices {
		if device.Online || (device.LastSeen > 0 && now-device.LastSeen < r.OnlineTimeout.Seconds()) {
			devices = append(devices, device)
		}
	}
	return devices
}

// UpdateLastSeen updates the last seen timestamp and signal quality for a device.
func (r *DeviceRegistry) UpdateLastSeen(deviceID string, rssi int, snr float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	device, exists := r.devices[deviceID]
	if !exists {
		return ErrDeviceNotFound
	}

	device.LastSeen = float64(time.Now().Unix())
	device.LastRSSI = rssi
	device.LastSNR = snr
	device.Online = true
	return nil
}

// SetOnline sets the online status of a device.
func (r *DeviceRegistry) SetOnline(deviceID string, online bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	device, exists := r.devices[deviceID]
	if !exists {
		return ErrDeviceNotFound
	}

	device.Online = online
	return nil
}

// Count returns the number of registered devices.
func (r *DeviceRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.devices)
}

// Clear removes all devices from the registry.
func (r *DeviceRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices = make(map[string]*RegisteredDevice)
}

// MessageRouter routes incoming LoRa messages to registered handlers.
type MessageRouter struct {
	Registry     *DeviceRegistry
	handlers     []handlerEntry
	mu           sync.RWMutex
	stopped      bool
	AutoRegister bool
}

type handlerEntry struct {
	handler MessageHandler
	filter  *MessageFilter
}

// NewMessageRouter creates a new message router.
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		handlers: make([]handlerEntry, 0),
	}
}

// Handle registers a handler for messages matching the given filter.
// If filter is nil, the handler receives all messages.
//
// Example:
//
//	router.Handle(func(msg *RoutedMessage) {
//	    fmt.Printf("Received from %s: %s\n", msg.FromDevice, msg.Data)
//	}, nil) // All messages
//
//	router.Handle(handler, &MessageFilter{FromDevice: "sensor1"}) // Only from sensor1
func (r *MessageRouter) Handle(handler MessageHandler, filter *MessageFilter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers = append(r.handlers, handlerEntry{
		handler: handler,
		filter:  filter,
	})
}

// HandleDevice registers a handler for messages from a specific device.
func (r *MessageRouter) HandleDevice(deviceID string, handler MessageHandler) {
	r.Handle(handler, &MessageFilter{FromDevice: deviceID})
}

// HandleGroup registers a handler for messages to a specific group.
func (r *MessageRouter) HandleGroup(group string, handler MessageHandler) {
	r.Handle(handler, &MessageFilter{Group: group})
}

// Route routes a message to all matching handlers.
// Returns the number of handlers that processed the message.
func (r *MessageRouter) Route(msg *RoutedMessage) (int, error) {
	r.mu.RLock()
	if r.stopped {
		r.mu.RUnlock()
		return 0, ErrRouterStopped
	}

	// Copy handlers slice to avoid holding lock during handler execution
	handlers := make([]handlerEntry, len(r.handlers))
	copy(handlers, r.handlers)
	registry := r.Registry
	autoRegister := r.AutoRegister
	r.mu.RUnlock()

	// Auto-register device if enabled
	if autoRegister && registry != nil && msg.FromDevice != "" {
		if _, err := registry.Get(msg.FromDevice); errors.Is(err, ErrDeviceNotFound) {
			registry.RegisterOrUpdate(&RegisteredDevice{
				DeviceID: msg.FromDevice,
				LastSeen: msg.Timestamp,
				LastRSSI: msg.RSSI,
				LastSNR:  msg.SNR,
				Online:   true,
			})
		} else if err == nil {
			//nolint:errcheck // Update is best-effort tracking; doesn't affect message handling
			registry.UpdateLastSeen(msg.FromDevice, msg.RSSI, msg.SNR)
		}
	}

	count := 0
	for _, entry := range handlers {
		if entry.filter == nil || entry.filter.Match(msg) {
			entry.handler(msg)
			count++
		}
	}

	return count, nil
}

// RouteEvent routes an Event to all matching handlers.
// This is a convenience method that converts an Event to a RoutedMessage.
func (r *MessageRouter) RouteEvent(event *Event, fromDevice string) (int, error) {
	data, err := decodeBase64(event.Info.Data)
	if err != nil {
		data = []byte(event.Info.Data) // Use raw data if decode fails
	}

	msg := &RoutedMessage{
		FromDevice: fromDevice,
		Data:       data,
		RSSI:       event.Info.RSSI,
		SNR:        event.Info.SNR,
		Timestamp:  event.Info.TS,
	}

	return r.Route(msg)
}

// Stop stops the router from processing messages.
func (r *MessageRouter) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopped = true
}

// Start resumes the router to process messages.
func (r *MessageRouter) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopped = false
}

// IsStopped returns true if the router is stopped.
func (r *MessageRouter) IsStopped() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stopped
}

// ClearHandlers removes all registered handlers.
func (r *MessageRouter) ClearHandlers() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = make([]handlerEntry, 0)
}

// HandlerCount returns the number of registered handlers.
func (r *MessageRouter) HandlerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.handlers)
}

// decodeBase64 decodes a base64-encoded string.
//
//nolint:gocyclo,cyclop // Base64 decoding with custom charset requires multiple conversions
func decodeBase64(s string) ([]byte, error) {
	if s == "" {
		return []byte{}, nil
	}

	// Calculate output length
	padding := 0
	if s != "" && s[len(s)-1] == '=' {
		padding++
		if len(s) > 1 && s[len(s)-2] == '=' {
			padding++
		}
	}

	outputLen := (len(s) * 3 / 4) - padding
	result := make([]byte, outputLen)

	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	charIndex := make(map[byte]int)
	for i, c := range []byte(base64Chars) {
		charIndex[c] = i
	}

	j := 0
	for i := 0; i < len(s); i += 4 {
		// Count valid characters in this block
		validChars := 0
		var n uint32
		for k := 0; k < 4 && i+k < len(s); k++ {
			c := s[i+k]
			if c == '=' {
				continue
			}
			if idx, ok := charIndex[c]; ok {
				n = n<<6 | uint32(idx)
				validChars++
			} else {
				return nil, errors.New("invalid base64 character")
			}
		}

		// Shift remaining bits to the correct position based on how many characters we read
		// 4 chars = 24 bits (shift 0), 3 chars = 18 bits (shift 6), 2 chars = 12 bits (shift 12)
		n <<= (4 - validChars) * 6

		// Extract bytes based on valid characters
		if validChars >= 2 && j < outputLen {
			result[j] = byte(n >> 16)
			j++
		}
		if validChars >= 3 && j < outputLen {
			result[j] = byte(n >> 8)
			j++
		}
		if validChars >= 4 && j < outputLen {
			result[j] = byte(n)
			j++
		}
	}

	return result[:j], nil
}
