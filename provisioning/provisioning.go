package provisioning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/rpc"
)

// DefaultAPAddress is the default IP address of Shelly devices in AP mode.
const DefaultAPAddress = "192.168.33.1"

// IPv4 mode constants.
const ipv4ModeStatic = "static"

// Common errors.
var (
	// ErrNotConnected indicates the device is not connected to WiFi.
	ErrNotConnected = errors.New("device not connected to WiFi")

	// ErrTimeout indicates a timeout waiting for connection.
	ErrTimeout = errors.New("timeout waiting for connection")

	// ErrNoSSID indicates no SSID was provided.
	ErrNoSSID = errors.New("SSID is required")
)

// Provisioner handles device provisioning operations.
type Provisioner struct {
	client *rpc.Client
}

// New creates a new Provisioner with the given RPC client.
// The client should be connected to a device (typically at 192.168.33.1
// when the device is in AP mode).
func New(client *rpc.Client) *Provisioner {
	return &Provisioner{client: client}
}

// ConfigureWiFi configures the device's station mode WiFi.
// After calling this, the device will attempt to connect to the specified
// network. You may lose connection if accessing via the device's AP.
func (p *Provisioner) ConfigureWiFi(ctx context.Context, config *WiFiConfig) error {
	if config.SSID == "" {
		return ErrNoSSID
	}

	// Build station config
	staConfig := map[string]any{
		"ssid": config.SSID,
	}

	// Enable station mode by default
	enable := true
	if config.Enable != nil {
		enable = *config.Enable
	}
	staConfig["enable"] = enable

	// Add password if provided
	if config.Password != "" {
		staConfig["pass"] = config.Password
	}

	// Add static IP configuration if provided
	//nolint:nestif // Static IP config requires multiple optional fields
	if config.StaticIP == ipv4ModeStatic {
		staConfig["ipv4mode"] = ipv4ModeStatic
		if config.IP != "" {
			staConfig["ip"] = config.IP
		}
		if config.Netmask != "" {
			staConfig["netmask"] = config.Netmask
		}
		if config.Gateway != "" {
			staConfig["gw"] = config.Gateway
		}
		if config.Nameserver != "" {
			staConfig["nameserver"] = config.Nameserver
		}
	}

	params := map[string]any{
		"config": map[string]any{
			"sta": staConfig,
		},
	}

	_, err := p.client.Call(ctx, "WiFi.SetConfig", params)
	return err
}

// ConfigureAP configures the device's access point.
func (p *Provisioner) ConfigureAP(ctx context.Context, config *APConfig) error {
	apConfig := make(map[string]any)

	if config.Enable != nil {
		apConfig["enable"] = *config.Enable
	}
	if config.SSID != "" {
		apConfig["ssid"] = config.SSID
	}
	if config.Password != "" {
		apConfig["pass"] = config.Password
	}
	if config.RangeExtender != nil {
		apConfig["range_extender"] = map[string]any{
			"enable": *config.RangeExtender,
		}
	}

	params := map[string]any{
		"config": map[string]any{
			"ap": apConfig,
		},
	}

	_, err := p.client.Call(ctx, "WiFi.SetConfig", params)
	return err
}

// SetAuth configures HTTP authentication on the device.
func (p *Provisioner) SetAuth(ctx context.Context, config *AuthConfig) error {
	params := map[string]any{}

	if config.Enable != nil && !*config.Enable {
		// Disable auth
		params["user"] = nil
		params["realm"] = nil
		params["ha1"] = nil
	} else {
		// Enable auth with credentials
		params["user"] = config.User
		if config.Password != "" {
			params["pass"] = config.Password
		}
	}

	_, err := p.client.Call(ctx, "Shelly.SetAuth", params)
	return err
}

// SetDeviceName sets the device's human-readable name.
func (p *Provisioner) SetDeviceName(ctx context.Context, name string) error {
	params := map[string]any{
		"config": map[string]any{
			"device": map[string]any{
				"name": name,
			},
		},
	}

	_, err := p.client.Call(ctx, "Sys.SetConfig", params)
	return err
}

// SetTimezone sets the device's timezone.
func (p *Provisioner) SetTimezone(ctx context.Context, timezone string) error {
	params := map[string]any{
		"config": map[string]any{
			"location": map[string]any{
				"tz": timezone,
			},
		},
	}

	_, err := p.client.Call(ctx, "Sys.SetConfig", params)
	return err
}

// SetLocation sets the device's geographic location.
func (p *Provisioner) SetLocation(ctx context.Context, lat, lon float64) error {
	params := map[string]any{
		"config": map[string]any{
			"location": map[string]any{
				"lat": lat,
				"lon": lon,
			},
		},
	}

	_, err := p.client.Call(ctx, "Sys.SetConfig", params)
	return err
}

// ConfigureCloud configures Shelly Cloud connection.
func (p *Provisioner) ConfigureCloud(ctx context.Context, config *CloudConfig) error {
	cloudConfig := make(map[string]any)

	if config.Enable != nil {
		cloudConfig["enable"] = *config.Enable
	}
	if config.Server != "" {
		cloudConfig["server"] = config.Server
	}

	params := map[string]any{
		"config": cloudConfig,
	}

	_, err := p.client.Call(ctx, "Cloud.SetConfig", params)
	return err
}

// GetDeviceInfo retrieves device information.
func (p *Provisioner) GetDeviceInfo(ctx context.Context) (*DeviceInfo, error) {
	result, err := p.client.Call(ctx, "Shelly.GetDeviceInfo", nil)
	if err != nil {
		return nil, err
	}

	var info DeviceInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// GetWiFiStatus retrieves the current WiFi status.
func (p *Provisioner) GetWiFiStatus(ctx context.Context) (*WiFiStatus, error) {
	result, err := p.client.Call(ctx, "WiFi.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status WiFiStatus
	if err := json.Unmarshal(result, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// WaitForConnection waits for the device to connect to WiFi.
// Returns the device's station IP address when connected.
func (p *Provisioner) WaitForConnection(ctx context.Context, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		status, err := p.GetWiFiStatus(ctx)
		if err == nil && status.StaIP != "" {
			return status.StaIP, nil
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
			// Continue polling
		}
	}

	return "", ErrTimeout
}

// IsConnected checks if the device is connected to WiFi.
func (p *Provisioner) IsConnected(ctx context.Context) (bool, error) {
	status, err := p.GetWiFiStatus(ctx)
	if err != nil {
		return false, err
	}
	return status.StaIP != "", nil
}

// Reboot reboots the device.
func (p *Provisioner) Reboot(ctx context.Context) error {
	_, err := p.client.Call(ctx, "Shelly.Reboot", nil)
	return err
}

// DisableBLE disables Bluetooth Low Energy on the device.
func (p *Provisioner) DisableBLE(ctx context.Context) error {
	params := map[string]any{
		"config": map[string]any{
			"enable": false,
		},
	}
	_, err := p.client.Call(ctx, "BLE.SetConfig", params)
	return err
}

// EnableBLE enables Bluetooth Low Energy on the device.
func (p *Provisioner) EnableBLE(ctx context.Context) error {
	params := map[string]any{
		"config": map[string]any{
			"enable": true,
		},
	}
	_, err := p.client.Call(ctx, "BLE.SetConfig", params)
	return err
}

// Provision performs complete device provisioning with the given configuration.
func (p *Provisioner) Provision(
	ctx context.Context, config *DeviceConfig, opts *ProvisionOptions,
) (*ProvisionResult, error) {
	if opts == nil {
		opts = DefaultProvisionOptions()
	}

	result := &ProvisionResult{
		Address: DefaultAPAddress,
	}

	// Get device info first
	info, err := p.GetDeviceInfo(ctx)
	if err != nil {
		result.Error = fmt.Errorf("failed to get device info: %w", err)
		return result, result.Error
	}
	result.DeviceInfo = info

	// Apply all configuration steps
	if err := p.applyProvisioningConfig(ctx, config); err != nil {
		result.Error = err
		return result, err
	}

	// Handle post-configuration options
	if err := p.applyPostProvisioningOptions(ctx, config, opts, result); err != nil {
		// Post-provisioning errors are non-fatal, stored in result.Error
		result.Error = err
	}

	result.Success = true
	return result, nil
}

// applyProvisioningConfig applies the device configuration steps.
func (p *Provisioner) applyProvisioningConfig(ctx context.Context, config *DeviceConfig) error {
	// Apply WiFi configuration
	if config.WiFi != nil {
		if err := p.ConfigureWiFi(ctx, config.WiFi); err != nil {
			return fmt.Errorf("failed to configure WiFi: %w", err)
		}
	}

	// Apply AP configuration
	if config.AP != nil {
		if err := p.ConfigureAP(ctx, config.AP); err != nil {
			return fmt.Errorf("failed to configure AP: %w", err)
		}
	}

	// Set device name
	if config.DeviceName != "" {
		if err := p.SetDeviceName(ctx, config.DeviceName); err != nil {
			return fmt.Errorf("failed to set device name: %w", err)
		}
	}

	// Set timezone
	if config.Timezone != "" {
		if err := p.SetTimezone(ctx, config.Timezone); err != nil {
			return fmt.Errorf("failed to set timezone: %w", err)
		}
	}

	// Set location
	if config.Location != nil {
		if err := p.SetLocation(ctx, config.Location.Lat, config.Location.Lon); err != nil {
			return fmt.Errorf("failed to set location: %w", err)
		}
	}

	// Configure cloud
	if config.Cloud != nil {
		if err := p.ConfigureCloud(ctx, config.Cloud); err != nil {
			return fmt.Errorf("failed to configure cloud: %w", err)
		}
	}

	// Set authentication
	if config.Auth != nil {
		if err := p.SetAuth(ctx, config.Auth); err != nil {
			return fmt.Errorf("failed to set auth: %w", err)
		}
	}

	return nil
}

// applyPostProvisioningOptions applies post-provisioning options like waiting for connection.
func (p *Provisioner) applyPostProvisioningOptions(
	ctx context.Context,
	config *DeviceConfig,
	opts *ProvisionOptions,
	result *ProvisionResult,
) error {
	// Wait for WiFi connection if requested
	if opts.WaitForConnection && config.WiFi != nil {
		timeout := time.Duration(opts.ConnectionTimeout) * time.Second
		newIP, err := p.WaitForConnection(ctx, timeout)
		if err != nil {
			return fmt.Errorf("failed to connect to WiFi: %w", err)
		}
		result.NewAddress = newIP
	}

	// Disable AP if requested and WiFi is configured
	if opts.DisableAP && config.WiFi != nil {
		disable := false
		if err := p.ConfigureAP(ctx, &APConfig{Enable: &disable}); err != nil {
			return fmt.Errorf("failed to disable AP: %w", err)
		}
	}

	// Disable BLE if requested
	if opts.DisableBLE {
		if err := p.DisableBLE(ctx); err != nil {
			return fmt.Errorf("failed to disable BLE: %w", err)
		}
	}

	return nil
}

// FactoryReset performs a factory reset on the device.
// This will reset all settings to defaults.
func (p *Provisioner) FactoryReset(ctx context.Context) error {
	_, err := p.client.Call(ctx, "Shelly.FactoryReset", nil)
	return err
}

// BLETransmitter is the interface for BLE GATT communication.
// This interface allows platform-specific implementations for actual BLE data transmission.
// Users should implement this interface using a BLE library appropriate for their platform
// (e.g., tinygo.org/x/bluetooth, github.com/go-ble/ble, or CoreBluetooth on macOS).
type BLETransmitter interface {
	// Connect connects to a BLE device by address and discovers services.
	Connect(ctx context.Context, address string) error

	// Disconnect disconnects from the current device.
	Disconnect() error

	// WriteCharacteristic writes data to the RPC characteristic.
	// Uses ShellyBLERPCCharUUID for command transmission.
	WriteCharacteristic(ctx context.Context, data []byte) error

	// ReadNotification reads a notification from the device.
	// Uses ShellyBLENotifyCharUUID for receiving responses.
	ReadNotification(ctx context.Context) ([]byte, error)

	// IsConnected returns true if currently connected to a device.
	IsConnected() bool
}

// BLEProvisioner handles BLE-based provisioning for Gen2+ devices.
// This provides an alternative to WiFi-based provisioning when
// the device's AP is not accessible.
//
// To perform actual BLE provisioning, set the Transmitter field to
// a platform-specific implementation of BLETransmitter.
type BLEProvisioner struct {
	// Transmitter is the optional BLE transmitter for actual GATT communication.
	// If nil, ProvisionViaBLE will only build the commands without transmitting them.
	Transmitter    BLETransmitter
	devices        map[string]*BLEDevice
	ScanTimeout    time.Duration
	ConnectTimeout time.Duration
	mu             sync.RWMutex
}

// BLEDevice represents a Shelly device discovered via BLE.
type BLEDevice struct {
	DiscoveredAt time.Time
	Name         string
	Address      string
	ServiceUUID  string
	Model        string
	RSSI         int
	Generation   int
	IsShelly     bool
}

// NewBLEProvisioner creates a new BLE-based provisioner.
func NewBLEProvisioner() *BLEProvisioner {
	return &BLEProvisioner{
		ScanTimeout:    10 * time.Second,
		ConnectTimeout: 30 * time.Second,
		devices:        make(map[string]*BLEDevice),
	}
}

// BLE Provisioning errors.
var (
	// ErrBLENotSupported indicates BLE provisioning is not supported.
	ErrBLENotSupported = errors.New("BLE provisioning not supported on this platform")

	// ErrBLEDeviceNotFound indicates the specified BLE device was not found.
	ErrBLEDeviceNotFound = errors.New("BLE device not found")

	// ErrBLEConnectionFailed indicates BLE connection failed.
	ErrBLEConnectionFailed = errors.New("BLE connection failed")

	// ErrBLEScanFailed indicates BLE scanning failed.
	ErrBLEScanFailed = errors.New("BLE scan failed")

	// ErrBLEWriteFailed indicates BLE characteristic write failed.
	ErrBLEWriteFailed = errors.New("BLE write failed")

	// ErrInvalidProfile indicates an invalid provisioning profile.
	ErrInvalidProfile = errors.New("invalid provisioning profile")

	// ErrProfileNotFound indicates the requested profile was not found.
	ErrProfileNotFound = errors.New("provisioning profile not found")

	// ErrBulkProvisioningFailed indicates bulk provisioning failed.
	ErrBulkProvisioningFailed = errors.New("bulk provisioning failed")
)

// Shelly BLE Service UUIDs (Gen2+).
const (
	// ShellyBLEServiceUUID is the primary Shelly BLE service.
	ShellyBLEServiceUUID = "5f6d4f53-5f52-5043-5f53-56435f49445f"

	// ShellyBLERPCCharUUID is the RPC characteristic for commands.
	ShellyBLERPCCharUUID = "5f6d4f53-5f52-5043-5f64-6174615f5f5f"

	// ShellyBLENotifyCharUUID is the notification characteristic.
	ShellyBLENotifyCharUUID = "5f6d4f53-5f52-5043-5f72-785f63746c5f"
)

// DiscoverBLEDevices discovers Shelly devices advertising via BLE.
// This is a simulated scan that returns previously registered devices,
// as actual BLE scanning requires platform-specific implementation.
func (b *BLEProvisioner) DiscoverBLEDevices(ctx context.Context) ([]*BLEDevice, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	devices := make([]*BLEDevice, 0, len(b.devices))
	for _, d := range b.devices {
		if d.IsShelly {
			devices = append(devices, d)
		}
	}
	return devices, nil
}

// AddDiscoveredDevice registers a BLE device that was discovered externally.
// This allows integration with platform-specific BLE libraries.
func (b *BLEProvisioner) AddDiscoveredDevice(device *BLEDevice) {
	b.mu.Lock()
	defer b.mu.Unlock()

	device.DiscoveredAt = time.Now()
	b.devices[device.Address] = device
}

// GetDevice returns a discovered device by its BLE address.
func (b *BLEProvisioner) GetDevice(address string) (*BLEDevice, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	device, ok := b.devices[address]
	if !ok {
		return nil, ErrBLEDeviceNotFound
	}
	return device, nil
}

// ClearDevices clears all discovered devices.
func (b *BLEProvisioner) ClearDevices() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.devices = make(map[string]*BLEDevice)
}

// DeviceCount returns the number of discovered devices.
func (b *BLEProvisioner) DeviceCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.devices)
}

// IsShellyDevice checks if a BLE device name matches Shelly naming patterns.
func IsShellyDevice(name string) bool {
	// Shelly devices typically have names like:
	// ShellyPlus1-XXXX, ShellyPro4PM-XXXX, etc.
	if len(name) < 6 {
		return false
	}
	return name[:6] == "Shelly"
}

// ParseBLEDeviceName parses a Shelly BLE device name to extract model information.
func ParseBLEDeviceName(name string) (model, macSuffix string) {
	if !IsShellyDevice(name) {
		return "", ""
	}

	// Find the separator (usually '-' before MAC suffix)
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '-' {
			model = name[:i]
			macSuffix = name[i+1:]
			return
		}
	}
	return name, ""
}

// BLEProvisionConfig contains configuration for BLE-based provisioning.
type BLEProvisionConfig struct {
	WiFi        *WiFiConfig
	EnableCloud *bool
	DeviceName  string
	Timezone    string
}

// ProvisionViaBLE provisions a device using BLE RPC commands.
//
// If b.Transmitter is set, the commands will be transmitted to the device via BLE.
// If b.Transmitter is nil, the method builds and returns the commands without
// transmitting them, which is useful for testing or when using an external BLE library.
//
// The provisioning protocol works as follows:
//  1. Connect to the device using the Shelly BLE service UUID
//  2. For each configuration command, serialize to JSON and write to RPC characteristic
//  3. Read notification characteristic for response
//  4. Disconnect when complete
func (b *BLEProvisioner) ProvisionViaBLE(
	ctx context.Context, address string, config *BLEProvisionConfig,
) (*BLEProvisionResult, error) {
	device, err := b.GetDevice(address)
	if err != nil {
		return nil, err
	}

	result := &BLEProvisionResult{
		Device:    device,
		StartedAt: time.Now(),
	}

	// Build the RPC commands
	commands := b.buildProvisionCommands(config)
	result.Commands = commands

	// If no transmitter, just return the commands without transmitting
	if b.Transmitter == nil {
		result.CompletedAt = time.Now()
		result.Success = true
		return result, nil
	}

	// Transmit commands via BLE
	if err := b.transmitCommands(ctx, address, commands); err != nil {
		result.CompletedAt = time.Now()
		result.Error = err
		result.Success = false
		return result, err
	}

	result.CompletedAt = time.Now()
	result.Success = true
	return result, nil
}

// buildProvisionCommands builds the RPC commands for BLE provisioning.
//
//nolint:nestif // WiFi config with optional static IP requires nested structure
func (b *BLEProvisioner) buildProvisionCommands(config *BLEProvisionConfig) []BLERPCCommand {
	var commands []BLERPCCommand

	// Configure WiFi
	if config.WiFi != nil && config.WiFi.SSID != "" {
		staConfig := map[string]any{
			"ssid":   config.WiFi.SSID,
			"enable": true,
		}
		if config.WiFi.Password != "" {
			staConfig["pass"] = config.WiFi.Password
		}
		if config.WiFi.StaticIP == ipv4ModeStatic {
			staConfig["ipv4mode"] = ipv4ModeStatic
			if config.WiFi.IP != "" {
				staConfig["ip"] = config.WiFi.IP
			}
			if config.WiFi.Netmask != "" {
				staConfig["netmask"] = config.WiFi.Netmask
			}
			if config.WiFi.Gateway != "" {
				staConfig["gw"] = config.WiFi.Gateway
			}
		}

		commands = append(commands, BLERPCCommand{
			Method: "WiFi.SetConfig",
			Params: map[string]any{
				"config": map[string]any{
					"sta": staConfig,
				},
			},
		})
	}

	// Set device name
	if config.DeviceName != "" {
		commands = append(commands, BLERPCCommand{
			Method: "Sys.SetConfig",
			Params: map[string]any{
				"config": map[string]any{
					"device": map[string]any{
						"name": config.DeviceName,
					},
				},
			},
		})
	}

	// Set timezone
	if config.Timezone != "" {
		commands = append(commands, BLERPCCommand{
			Method: "Sys.SetConfig",
			Params: map[string]any{
				"config": map[string]any{
					"location": map[string]any{
						"tz": config.Timezone,
					},
				},
			},
		})
	}

	// Configure cloud
	if config.EnableCloud != nil {
		commands = append(commands, BLERPCCommand{
			Method: "Cloud.SetConfig",
			Params: map[string]any{
				"config": map[string]any{
					"enable": *config.EnableCloud,
				},
			},
		})
	}

	return commands
}

// transmitCommands sends RPC commands to a device via BLE.
func (b *BLEProvisioner) transmitCommands(
	ctx context.Context, address string, commands []BLERPCCommand,
) error {
	// Connect to device
	if err := b.Transmitter.Connect(ctx, address); err != nil {
		return fmt.Errorf("%w: %w", ErrBLEConnectionFailed, err)
	}
	defer b.Transmitter.Disconnect() //nolint:errcheck // Best-effort disconnect

	// Send each command and wait for response
	for i, cmd := range commands {
		data, err := cmd.ToJSON(i + 1)
		if err != nil {
			return fmt.Errorf("failed to serialize command %d: %w", i+1, err)
		}

		// Write command to RPC characteristic
		writeErr := b.Transmitter.WriteCharacteristic(ctx, data)
		if writeErr != nil {
			return fmt.Errorf("%w: command %d (%s): %w", ErrBLEWriteFailed, i+1, cmd.Method, writeErr)
		}

		// Read response (optional - some commands may not send response)
		_, readErr := b.Transmitter.ReadNotification(ctx)
		if readErr != nil {
			// Log but don't fail - some commands don't return responses
			// In production, you may want to verify responses
			continue
		}
	}

	return nil
}

// BLERPCCommand represents an RPC command to be sent via BLE.
type BLERPCCommand struct {
	Params map[string]any
	Method string
}

// ToJSON converts the command to JSON bytes for BLE transmission.
func (c *BLERPCCommand) ToJSON(id int) ([]byte, error) {
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  c.Method,
	}
	if c.Params != nil {
		request["params"] = c.Params
	}
	return json.Marshal(request)
}

// BLEProvisionResult represents the result of a BLE provisioning operation.
type BLEProvisionResult struct {
	StartedAt   time.Time
	CompletedAt time.Time
	Error       error
	Device      *BLEDevice
	Commands    []BLERPCCommand
	Success     bool
}

// Duration returns how long provisioning took.
func (r *BLEProvisionResult) Duration() time.Duration {
	return r.CompletedAt.Sub(r.StartedAt)
}

// BulkProvisioner handles provisioning multiple devices.
type BulkProvisioner struct {
	ClientFactory func(address string) (*rpc.Client, error)
	Concurrency   int
	RetryCount    int
	RetryDelay    time.Duration
}

// NewBulkProvisioner creates a new bulk provisioner.
func NewBulkProvisioner(clientFactory func(address string) (*rpc.Client, error)) *BulkProvisioner {
	return &BulkProvisioner{
		Concurrency:   3,
		RetryCount:    2,
		RetryDelay:    5 * time.Second,
		ClientFactory: clientFactory,
	}
}

// BulkProvisionTarget represents a device to be provisioned.
type BulkProvisionTarget struct {
	// Address is the device address (IP or hostname).
	Address string

	// Config is the device-specific configuration.
	// If nil, the default profile configuration will be used.
	Config *DeviceConfig

	// ProfileName is the name of the profile to use (if Config is nil).
	ProfileName string
}

// BulkProvisionResult represents the result of bulk provisioning.
type BulkProvisionResult struct {
	StartedAt    time.Time
	CompletedAt  time.Time
	Results      map[string]*ProvisionResult
	TotalDevices int
	SuccessCount int
	FailureCount int
	Duration     time.Duration
}

// ProvisionBulk provisions multiple devices concurrently.
//
//nolint:gocyclo,cyclop // Bulk provisioning orchestrates multiple devices
func (b *BulkProvisioner) ProvisionBulk(
	ctx context.Context,
	targets []*BulkProvisionTarget,
	profiles *ProfileRegistry,
	defaultOpts *ProvisionOptions,
) (*BulkProvisionResult, error) {
	if b.ClientFactory == nil {
		return nil, errors.New("client factory not configured")
	}

	result := &BulkProvisionResult{
		TotalDevices: len(targets),
		Results:      make(map[string]*ProvisionResult),
		StartedAt:    time.Now(),
	}

	if len(targets) == 0 {
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		return result, nil
	}

	// Set default options if not provided
	if defaultOpts == nil {
		defaultOpts = DefaultProvisionOptions()
	}

	// Create work channel and results channel
	type workItem struct {
		target *BulkProvisionTarget
		config *DeviceConfig
		opts   *ProvisionOptions
	}

	work := make(chan workItem, len(targets))
	results := make(chan struct {
		result  *ProvisionResult
		address string
	}, len(targets))

	// Determine concurrency
	concurrency := b.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	if concurrency > len(targets) {
		concurrency = len(targets)
	}

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range work {
				r := b.provisionWithRetry(ctx, item.target.Address, item.config, item.opts)
				results <- struct {
					result  *ProvisionResult
					address string
				}{result: r, address: item.target.Address}
			}
		}()
	}

	// Queue work items
	for _, target := range targets {
		config, opts := b.resolveTargetConfig(target, profiles, defaultOpts)

		// Skip if no config available
		if config == nil {
			results <- struct {
				result  *ProvisionResult
				address string
			}{
				address: target.Address,
				result: &ProvisionResult{
					Address: target.Address,
					Success: false,
					Error:   errors.New("no configuration provided"),
				},
			}
			continue
		}

		work <- workItem{target: target, config: config, opts: opts}
	}
	close(work)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for r := range results {
		result.Results[r.address] = r.result
		if r.result.Success {
			result.SuccessCount++
		} else {
			result.FailureCount++
		}
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	return result, nil
}

// resolveTargetConfig resolves the configuration and options for a bulk provision target.
// Returns nil config if no configuration is available.
func (b *BulkProvisioner) resolveTargetConfig(
	target *BulkProvisionTarget, profiles *ProfileRegistry, defaultOpts *ProvisionOptions,
) (*DeviceConfig, *ProvisionOptions) {
	config := target.Config

	// If no direct config, try to get from profile
	if config == nil && target.ProfileName != "" && profiles != nil {
		if profile, err := profiles.Get(target.ProfileName); err == nil {
			config = profile.Config
		}
	}

	// Resolve options
	opts := defaultOpts
	if target.ProfileName != "" && profiles != nil {
		if profile, err := profiles.Get(target.ProfileName); err == nil && profile.Options != nil {
			opts = profile.Options
		}
	}

	return config, opts
}

// provisionWithRetry provisions a single device with retry support.
func (b *BulkProvisioner) provisionWithRetry(
	ctx context.Context, address string, config *DeviceConfig, opts *ProvisionOptions,
) *ProvisionResult {
	var lastResult *ProvisionResult

	retries := b.RetryCount
	if retries < 0 {
		retries = 0
	}

	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return &ProvisionResult{
					Address: address,
					Success: false,
					Error:   ctx.Err(),
				}
			case <-time.After(b.RetryDelay):
			}
		}

		client, err := b.ClientFactory(address)
		if err != nil {
			lastResult = &ProvisionResult{
				Address: address,
				Success: false,
				Error:   fmt.Errorf("failed to create client: %w", err),
			}
			continue
		}

		prov := New(client)
		result, err := prov.Provision(ctx, config, opts)
		if err == nil && result.Success {
			return result
		}

		lastResult = result
		if lastResult == nil {
			lastResult = &ProvisionResult{
				Address: address,
				Success: false,
				Error:   err,
			}
		}
	}

	return lastResult
}

// Profile represents a reusable provisioning configuration.
type Profile struct {
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Config      *DeviceConfig
	Options     *ProvisionOptions
	Tags        map[string]string
	Name        string
	Description string
}

// ProfileRegistry manages provisioning profiles.
type ProfileRegistry struct {
	profiles map[string]*Profile
	mu       sync.RWMutex
}

// NewProfileRegistry creates a new profile registry.
func NewProfileRegistry() *ProfileRegistry {
	return &ProfileRegistry{
		profiles: make(map[string]*Profile),
	}
}

// Register adds or updates a profile in the registry.
func (r *ProfileRegistry) Register(profile *Profile) error {
	if profile.Name == "" {
		return ErrInvalidProfile
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if existing, ok := r.profiles[profile.Name]; ok {
		profile.CreatedAt = existing.CreatedAt
	} else {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	r.profiles[profile.Name] = profile
	return nil
}

// Get retrieves a profile by name.
func (r *ProfileRegistry) Get(name string) (*Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.profiles[name]
	if !ok {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

// Delete removes a profile by name.
func (r *ProfileRegistry) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.profiles[name]; !ok {
		return ErrProfileNotFound
	}

	delete(r.profiles, name)
	return nil
}

// List returns all registered profiles.
func (r *ProfileRegistry) List() []*Profile {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profiles := make([]*Profile, 0, len(r.profiles))
	for _, p := range r.profiles {
		profiles = append(profiles, p)
	}
	return profiles
}

// ListByTag returns profiles with a specific tag.
func (r *ProfileRegistry) ListByTag(key, value string) []*Profile {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var profiles []*Profile
	for _, p := range r.profiles {
		if p.Tags != nil && p.Tags[key] == value {
			profiles = append(profiles, p)
		}
	}
	return profiles
}

// Count returns the number of registered profiles.
func (r *ProfileRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.profiles)
}

// Clear removes all profiles.
func (r *ProfileRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles = make(map[string]*Profile)
}

// Export exports all profiles to JSON.
func (r *ProfileRegistry) Export() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profiles := make([]*Profile, 0, len(r.profiles))
	for _, p := range r.profiles {
		profiles = append(profiles, p)
	}
	return json.Marshal(profiles)
}

// Import imports profiles from JSON.
func (r *ProfileRegistry) Import(data []byte) error {
	var profiles []*Profile
	if err := json.Unmarshal(data, &profiles); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, p := range profiles {
		r.profiles[p.Name] = p
	}
	return nil
}

// StandardProfiles returns commonly used provisioning profiles.
func StandardProfiles() []*Profile {
	enableCloud := true
	disableCloud := false

	return []*Profile{
		{
			Name:        "home-basic",
			Description: "Basic home network configuration",
			Config: &DeviceConfig{
				Timezone: "America/New_York",
				Cloud: &CloudConfig{
					Enable: &enableCloud,
				},
			},
			Options: DefaultProvisionOptions(),
			Tags:    map[string]string{"type": "home", "level": "basic"},
		},
		{
			Name:        "home-advanced",
			Description: "Advanced home configuration with auth",
			Config: &DeviceConfig{
				Timezone: "America/New_York",
				Cloud: &CloudConfig{
					Enable: &enableCloud,
				},
				Auth: &AuthConfig{
					User: "admin",
				},
			},
			Options: &ProvisionOptions{
				WaitForConnection: true,
				ConnectionTimeout: 60,
				VerifyConnection:  true,
				DisableAP:         true,
			},
			Tags: map[string]string{"type": "home", "level": "advanced"},
		},
		{
			Name:        "enterprise-secure",
			Description: "Enterprise configuration with no cloud, auth enabled",
			Config: &DeviceConfig{
				Cloud: &CloudConfig{
					Enable: &disableCloud,
				},
				Auth: &AuthConfig{
					User: "admin",
				},
			},
			Options: &ProvisionOptions{
				WaitForConnection: true,
				ConnectionTimeout: 60,
				VerifyConnection:  true,
				DisableAP:         true,
				DisableBLE:        true,
			},
			Tags: map[string]string{"type": "enterprise", "level": "secure"},
		},
		{
			Name:        "iot-minimal",
			Description: "Minimal IoT device configuration",
			Config: &DeviceConfig{
				Cloud: &CloudConfig{
					Enable: &disableCloud,
				},
			},
			Options: &ProvisionOptions{
				WaitForConnection: true,
				ConnectionTimeout: 30,
				VerifyConnection:  false,
				DisableAP:         false,
			},
			Tags: map[string]string{"type": "iot", "level": "minimal"},
		},
	}
}
