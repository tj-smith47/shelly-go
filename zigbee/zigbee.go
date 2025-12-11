package zigbee

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
	"github.com/tj-smith47/shelly-go/types"
)

// Common errors for Zigbee operations.
var (
	// ErrZigbeeNotSupported indicates the device does not support Zigbee.
	ErrZigbeeNotSupported = errors.New("device does not support Zigbee")

	// ErrPairingTimeout indicates the pairing operation timed out.
	ErrPairingTimeout = errors.New(
		"pairing timeout: device did not join network within timeout period",
	)

	// ErrPairingFailed indicates the pairing operation failed.
	ErrPairingFailed = errors.New("pairing failed")

	// ErrAlreadyJoined indicates the device is already joined to a network.
	ErrAlreadyJoined = errors.New("device is already joined to a Zigbee network")

	// ErrNotJoined indicates the device is not joined to a network.
	ErrNotJoined = errors.New("device is not joined to a Zigbee network")
)

// Zigbee provides access to the Zigbee component on Gen2+ Shelly devices.
// The Zigbee component handles Zigbee connectivity services, allowing devices
// to participate in Zigbee networks for home automation integration.
type Zigbee struct {
	client *rpc.Client
}

// NewZigbee creates a new Zigbee component instance.
func NewZigbee(client *rpc.Client) *Zigbee {
	return &Zigbee{client: client}
}

// GetConfig retrieves the current Zigbee configuration.
func (z *Zigbee) GetConfig(ctx context.Context) (*Config, error) {
	result, err := z.client.Call(ctx, "Zigbee.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SetConfig updates the Zigbee configuration.
// Use this to enable or disable Zigbee connectivity.
func (z *Zigbee) SetConfig(ctx context.Context, params *SetConfigParams) error {
	reqParams := map[string]any{
		"config": params,
	}
	_, err := z.client.Call(ctx, "Zigbee.SetConfig", reqParams)
	return err
}

// GetStatus retrieves the current Zigbee status.
// The status includes network state, EUI64 address, PAN ID, and channel
// information when connected to a network.
func (z *Zigbee) GetStatus(ctx context.Context) (*Status, error) {
	result, err := z.client.Call(ctx, "Zigbee.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status Status
	if err := json.Unmarshal(result, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// StartNetworkSteering triggers network steering, causing the device to
// attempt to join nearby Zigbee networks. The coordinator must be in
// pairing mode for the device to successfully join.
//
// After calling this method, poll GetStatus to check if the device has
// joined a network (NetworkState will be "joined").
func (z *Zigbee) StartNetworkSteering(ctx context.Context) error {
	_, err := z.client.Call(ctx, "Zigbee.StartNetworkSteering", nil)
	return err
}

// Enable enables Zigbee connectivity on the device.
// This is a convenience method equivalent to SetConfig with Enable=true.
func (z *Zigbee) Enable(ctx context.Context) error {
	enable := true
	return z.SetConfig(ctx, &SetConfigParams{Enable: &enable})
}

// Disable disables Zigbee connectivity on the device.
// This is a convenience method equivalent to SetConfig with Enable=false.
// When disabled, the device will leave any joined network.
func (z *Zigbee) Disable(ctx context.Context) error {
	disable := false
	return z.SetConfig(ctx, &SetConfigParams{Enable: &disable})
}

// IsEnabled returns whether Zigbee is currently enabled on the device.
func (z *Zigbee) IsEnabled(ctx context.Context) (bool, error) {
	config, err := z.GetConfig(ctx)
	if err != nil {
		return false, err
	}
	return config.Enable, nil
}

// IsJoined returns whether the device is currently joined to a Zigbee network.
func (z *Zigbee) IsJoined(ctx context.Context) (bool, error) {
	status, err := z.GetStatus(ctx)
	if err != nil {
		return false, err
	}
	return status.NetworkState == NetworkStateJoined, nil
}

// GetNetworkState returns the current Zigbee network state.
// Possible values: "not_configured", "ready", "steering", "joined"
func (z *Zigbee) GetNetworkState(ctx context.Context) (string, error) {
	status, err := z.GetStatus(ctx)
	if err != nil {
		return "", err
	}
	return status.NetworkState, nil
}

// GetEUI64 returns the device's EUI64 address.
// The EUI64 is the device's unique 64-bit identifier in the Zigbee network.
func (z *Zigbee) GetEUI64(ctx context.Context) (string, error) {
	status, err := z.GetStatus(ctx)
	if err != nil {
		return "", err
	}
	return status.EUI64, nil
}

// GetChannel returns the Zigbee radio channel currently in use.
// Returns 0 if not connected to a network.
func (z *Zigbee) GetChannel(ctx context.Context) (int, error) {
	status, err := z.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.Channel, nil
}

// GetPANID returns the Personal Area Network ID the device is connected to.
// Returns 0 if not connected to a network.
func (z *Zigbee) GetPANID(ctx context.Context) (uint16, error) {
	status, err := z.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.PANID, nil
}

// GetNetworkInfo returns detailed information about the joined Zigbee network.
// Returns ErrNotJoined if the device is not connected to a network.
func (z *Zigbee) GetNetworkInfo(ctx context.Context) (*NetworkInfo, error) {
	status, err := z.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	if status.NetworkState != NetworkStateJoined {
		return nil, ErrNotJoined
	}

	return &NetworkInfo{
		PANID:            status.PANID,
		Channel:          status.Channel,
		CoordinatorEUI64: status.CoordinatorEUI64,
	}, nil
}

// LeaveNetwork causes the device to leave its current Zigbee network.
// This disables Zigbee and clears the network configuration.
func (z *Zigbee) LeaveNetwork(ctx context.Context) error {
	return z.Disable(ctx)
}

// PairToNetwork initiates the pairing process to join a Zigbee network.
// The coordinator must be in pairing mode for the device to join.
//
// Parameters:
//   - timeout: Maximum time to wait for pairing (default 180 seconds if 0)
//   - pollInterval: How often to check pairing status (default 2 seconds if 0)
//
// Returns a PairingResult with the final state and network info if successful.
//
// Example:
//
//	result, err := zigbee.PairToNetwork(ctx, 180*time.Second, 2*time.Second)
//	if err == nil && result.State == zigbee.PairingStateJoined {
//	    fmt.Printf("Joined network on channel %d\n", result.NetworkInfo.Channel)
//	}
//
//nolint:gocyclo,cyclop // Zigbee pairing has multiple steps and retry logic
func (z *Zigbee) PairToNetwork(ctx context.Context, timeout, pollInterval time.Duration) (*PairingResult, error) {
	if timeout == 0 {
		timeout = 180 * time.Second
	}
	if pollInterval == 0 {
		pollInterval = 2 * time.Second
	}

	// Check current status
	status, err := z.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	// Already joined?
	if status.NetworkState == NetworkStateJoined {
		return &PairingResult{
			State: PairingStateJoined,
			NetworkInfo: &NetworkInfo{
				PANID:            status.PANID,
				Channel:          status.Channel,
				CoordinatorEUI64: status.CoordinatorEUI64,
			},
		}, ErrAlreadyJoined
	}

	// Enable Zigbee if not already enabled
	config, err := z.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	if !config.Enable {
		if err := z.Enable(ctx); err != nil {
			return &PairingResult{State: PairingStateFailed, Error: err}, err
		}
	}

	// Start network steering
	if err := z.StartNetworkSteering(ctx); err != nil {
		return &PairingResult{State: PairingStateFailed, Error: err}, err
	}

	// Poll for status until joined, failed, or timeout
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return &PairingResult{State: PairingStateFailed, Error: ctx.Err()}, ctx.Err()

		case <-ticker.C:
			if time.Now().After(deadline) {
				return &PairingResult{State: PairingStateTimeout, Error: ErrPairingTimeout}, ErrPairingTimeout
			}

			status, err := z.GetStatus(ctx)
			if err != nil {
				continue // Transient errors during pairing are expected
			}

			switch status.NetworkState {
			case NetworkStateJoined:
				return &PairingResult{
					State: PairingStateJoined,
					NetworkInfo: &NetworkInfo{
						PANID:            status.PANID,
						Channel:          status.Channel,
						CoordinatorEUI64: status.CoordinatorEUI64,
					},
				}, nil

			case NetworkStateFailed:
				return &PairingResult{State: PairingStateFailed, Error: ErrPairingFailed}, ErrPairingFailed

			case NetworkStateSteering:
				// Still trying, continue polling
				continue

			default:
				// Other states (initializing, disabled), continue polling
				continue
			}
		}
	}
}

// Scanner provides methods to discover Zigbee-capable Shelly devices on the network.
type Scanner struct {
	// HTTPClient is the HTTP client to use for device probing.
	// If nil, a default client with 5-second timeout is used.
	HTTPClient *http.Client

	// Concurrency controls how many devices are probed in parallel.
	// Default is 10.
	Concurrency int
}

// NewScanner creates a new Zigbee device scanner.
func NewScanner() *Scanner {
	return &Scanner{
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		Concurrency: 10,
	}
}

// DiscoverDevices discovers Zigbee-capable Shelly devices at the given addresses.
// It probes each address to determine if it's a Shelly device with Zigbee support.
//
// Parameters:
//   - addresses: IP addresses or hostnames to probe
//
// Returns discovered devices (both with and without Zigbee based on filter).
//
// Example:
//
//	scanner := zigbee.NewScanner()
//	devices, err := scanner.DiscoverDevices(ctx, []string{"10.23.47.220", "10.23.47.221"})
//	for _, dev := range devices {
//	    if dev.HasZigbee {
//	        fmt.Printf("%s supports Zigbee\n", dev.DeviceID)
//	    }
//	}
func (s *Scanner) DiscoverDevices(ctx context.Context, addresses []string) ([]DiscoveredDevice, error) {
	if s.HTTPClient == nil {
		s.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}

	concurrency := s.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	var (
		mu      sync.Mutex
		devices []DiscoveredDevice
		sem     = make(chan struct{}, concurrency)
		wg      sync.WaitGroup
	)

	for _, addr := range addresses {
		select {
		case <-ctx.Done():
			break
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(address string) {
			defer wg.Done()
			defer func() { <-sem }()

			device, err := s.probeDevice(ctx, address)
			if err != nil {
				return // Device unreachable or not a Shelly device
			}

			mu.Lock()
			devices = append(devices, *device)
			mu.Unlock()
		}(addr)
	}

	wg.Wait()
	return devices, nil
}

// DiscoverZigbeeDevices discovers only Zigbee-capable Shelly devices at the given addresses.
// This is a convenience method that filters out devices without Zigbee support.
func (s *Scanner) DiscoverZigbeeDevices(ctx context.Context, addresses []string) ([]DiscoveredDevice, error) {
	allDevices, err := s.DiscoverDevices(ctx, addresses)
	if err != nil {
		return nil, err
	}

	var zigbeeDevices []DiscoveredDevice
	for _, dev := range allDevices {
		if dev.HasZigbee {
			zigbeeDevices = append(zigbeeDevices, dev)
		}
	}
	return zigbeeDevices, nil
}

// probeDevice probes a single device to determine its Zigbee capabilities.
func (s *Scanner) probeDevice(ctx context.Context, address string) (*DiscoveredDevice, error) {
	// Create transport and client
	httpTransport := transport.NewHTTP(address)
	client := rpc.NewClient(httpTransport)
	defer func() { _ = httpTransport.Close() }()

	// Get device info
	result, err := client.Call(ctx, "Shelly.GetDeviceInfo", nil)
	if err != nil {
		return nil, err
	}

	var deviceInfo struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		App   string `json:"app"`
		Gen   int    `json:"gen"`
	}
	if unmarshalErr := json.Unmarshal(result, &deviceInfo); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	// Determine generation
	var gen types.Generation
	switch deviceInfo.Gen {
	case 2:
		gen = types.Gen2
	case 3:
		gen = types.Gen3
	case 4:
		gen = types.Gen4
	default:
		gen = types.Gen2 // Fallback
	}

	device := &DiscoveredDevice{
		Address:    address,
		DeviceID:   deviceInfo.ID,
		Model:      deviceInfo.Model,
		Generation: gen,
		HasZigbee:  false,
	}

	// Check for Zigbee component
	zigbee := NewZigbee(client)
	status, err := zigbee.GetStatus(ctx)
	if err == nil {
		device.HasZigbee = true
		device.ZigbeeStatus = status
	}

	return device, nil
}

// ZigbeeClusterID constants for common Zigbee clusters.
const (
	// Basic Cluster (0x0000)
	ClusterBasic uint16 = 0x0000

	// Power Configuration Cluster (0x0001)
	ClusterPowerConfiguration uint16 = 0x0001

	// Identify Cluster (0x0003)
	ClusterIdentify uint16 = 0x0003

	// Groups Cluster (0x0004)
	ClusterGroups uint16 = 0x0004

	// Scenes Cluster (0x0005)
	ClusterScenes uint16 = 0x0005

	// On/Off Cluster (0x0006)
	ClusterOnOff uint16 = 0x0006

	// Level Control Cluster (0x0008)
	ClusterLevelControl uint16 = 0x0008

	// Color Control Cluster (0x0300)
	ClusterColorControl uint16 = 0x0300

	// Temperature Measurement Cluster (0x0402)
	ClusterTemperatureMeasurement uint16 = 0x0402

	// Humidity Measurement Cluster (0x0405)
	ClusterHumidityMeasurement uint16 = 0x0405

	// Illuminance Measurement Cluster (0x0400)
	ClusterIlluminanceMeasurement uint16 = 0x0400

	// Occupancy Sensing Cluster (0x0406)
	ClusterOccupancySensing uint16 = 0x0406

	// Electrical Measurement Cluster (0x0B04)
	ClusterElectricalMeasurement uint16 = 0x0B04

	// Metering Cluster (0x0702)
	ClusterMetering uint16 = 0x0702

	// Window Covering Cluster (0x0102)
	ClusterWindowCovering uint16 = 0x0102

	// Thermostat Cluster (0x0201)
	ClusterThermostat uint16 = 0x0201

	// IAS Zone Cluster (0x0500)
	ClusterIASZone uint16 = 0x0500

	// Shelly RPC Cluster (0xFC01) - Custom Shelly cluster
	ClusterShellyRPC uint16 = 0xFC01

	// Shelly WiFi Setup Cluster (0xFC02) - Custom Shelly cluster
	ClusterShellyWiFiSetup uint16 = 0xFC02
)

// ClusterMapping maps Zigbee clusters to Shelly component types.
// This allows translating between Zigbee cluster capabilities and
// the corresponding Shelly device components.
var ClusterMapping = map[uint16]ClusterCapability{
	ClusterOnOff: {
		ClusterID:     ClusterOnOff,
		ClusterName:   "On/Off",
		ComponentType: "switch",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "OnOff", Type: "bool", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x4000, Name: "GlobalSceneControl", Type: "bool", Readable: true,
				Writable: false, Reportable: false},
			{ID: 0x4001, Name: "OnTime", Type: "uint16", Readable: true, Writable: true,
				Reportable: false},
			{ID: 0x4002, Name: "OffWaitTime", Type: "uint16", Readable: true, Writable: true,
				Reportable: false},
		},
		Commands: []ClusterCommand{
			{ID: 0x00, Name: "Off", Direction: "client_to_server"},
			{ID: 0x01, Name: "On", Direction: "client_to_server"},
			{ID: 0x02, Name: "Toggle", Direction: "client_to_server"},
		},
	},
	ClusterLevelControl: {
		ClusterID:     ClusterLevelControl,
		ClusterName:   "Level Control",
		ComponentType: "light",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "CurrentLevel", Type: "uint8", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x0001, Name: "RemainingTime", Type: "uint16", Readable: true, Writable: false,
				Reportable: false},
			{ID: 0x0010, Name: "OnOffTransitionTime", Type: "uint16", Readable: true,
				Writable: true, Reportable: false},
			{ID: 0x0011, Name: "OnLevel", Type: "uint8", Readable: true, Writable: true,
				Reportable: false},
		},
		Commands: []ClusterCommand{
			{ID: 0x00, Name: "MoveToLevel", Direction: "client_to_server"},
			{ID: 0x01, Name: "Move", Direction: "client_to_server"},
			{ID: 0x02, Name: "Step", Direction: "client_to_server"},
			{ID: 0x03, Name: "Stop", Direction: "client_to_server"},
			{ID: 0x04, Name: "MoveToLevelWithOnOff", Direction: "client_to_server"},
		},
	},
	ClusterColorControl: {
		ClusterID:     ClusterColorControl,
		ClusterName:   "Color Control",
		ComponentType: "rgb",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "CurrentHue", Type: "uint8", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x0001, Name: "CurrentSaturation", Type: "uint8", Readable: true,
				Writable: false, Reportable: true},
			{ID: 0x0003, Name: "CurrentX", Type: "uint16", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x0004, Name: "CurrentY", Type: "uint16", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x0007, Name: "ColorTemperatureMireds", Type: "uint16", Readable: true,
				Writable: false, Reportable: true},
			{ID: 0x0008, Name: "ColorMode", Type: "enum8", Readable: true, Writable: false,
				Reportable: false},
		},
		Commands: []ClusterCommand{
			{ID: 0x00, Name: "MoveToHue", Direction: "client_to_server"},
			{ID: 0x01, Name: "MoveHue", Direction: "client_to_server"},
			{ID: 0x03, Name: "MoveToSaturation", Direction: "client_to_server"},
			{ID: 0x06, Name: "MoveToHueAndSaturation", Direction: "client_to_server"},
			{ID: 0x07, Name: "MoveToColor", Direction: "client_to_server"},
			{ID: 0x0A, Name: "MoveToColorTemperature", Direction: "client_to_server"},
		},
	},
	//nolint:dupl // Zigbee cluster definitions have similar structure by design
	ClusterWindowCovering: {
		ClusterID:     ClusterWindowCovering,
		ClusterName:   "Window Covering",
		ComponentType: "cover",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "Type", Type: "enum8", Readable: true, Writable: false,
				Reportable: false},
			{ID: 0x0003, Name: "CurrentPositionLiftPercent100ths", Type: "uint16",
				Readable: true, Writable: false, Reportable: true},
			{ID: 0x0007, Name: "ConfigStatus", Type: "bitmap8", Readable: true, Writable: false,
				Reportable: false},
			{ID: 0x0008, Name: "CurrentPositionLiftPercentage", Type: "uint8", Readable: true,
				Writable: false, Reportable: true},
			{ID: 0x000A, Name: "OperationalStatus", Type: "bitmap8", Readable: true,
				Writable: false, Reportable: true},
		},
		Commands: []ClusterCommand{
			{ID: 0x00, Name: "UpOrOpen", Direction: "client_to_server"},
			{ID: 0x01, Name: "DownOrClose", Direction: "client_to_server"},
			{ID: 0x02, Name: "Stop", Direction: "client_to_server"},
			{ID: 0x05, Name: "GoToLiftPercentage", Direction: "client_to_server"},
		},
	},
	ClusterTemperatureMeasurement: {
		ClusterID:     ClusterTemperatureMeasurement,
		ClusterName:   "Temperature Measurement",
		ComponentType: "temperature",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "MeasuredValue", Type: "int16", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x0001, Name: "MinMeasuredValue", Type: "int16", Readable: true, Writable: false,
				Reportable: false},
			{ID: 0x0002, Name: "MaxMeasuredValue", Type: "int16", Readable: true, Writable: false,
				Reportable: false},
			{ID: 0x0003, Name: "Tolerance", Type: "uint16", Readable: true, Writable: false,
				Reportable: false},
		},
	},
	ClusterHumidityMeasurement: {
		ClusterID:     ClusterHumidityMeasurement,
		ClusterName:   "Humidity Measurement",
		ComponentType: "humidity",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "MeasuredValue", Type: "uint16", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x0001, Name: "MinMeasuredValue", Type: "uint16", Readable: true, Writable: false,
				Reportable: false},
			{ID: 0x0002, Name: "MaxMeasuredValue", Type: "uint16", Readable: true, Writable: false,
				Reportable: false},
			{ID: 0x0003, Name: "Tolerance", Type: "uint16", Readable: true, Writable: false,
				Reportable: false},
		},
	},
	ClusterIlluminanceMeasurement: {
		ClusterID:     ClusterIlluminanceMeasurement,
		ClusterName:   "Illuminance Measurement",
		ComponentType: "illuminance",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "MeasuredValue", Type: "uint16", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x0001, Name: "MinMeasuredValue", Type: "uint16", Readable: true, Writable: false,
				Reportable: false},
			{ID: 0x0002, Name: "MaxMeasuredValue", Type: "uint16", Readable: true, Writable: false,
				Reportable: false},
		},
	},
	ClusterElectricalMeasurement: {
		ClusterID:     ClusterElectricalMeasurement,
		ClusterName:   "Electrical Measurement",
		ComponentType: "em",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "MeasurementType", Type: "bitmap32", Readable: true, Writable: false,
				Reportable: false},
			{ID: 0x0505, Name: "RMSVoltage", Type: "uint16", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x0508, Name: "RMSCurrent", Type: "uint16", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x050B, Name: "ActivePower", Type: "int16", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x050E, Name: "ReactivePower", Type: "int16", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x050F, Name: "ApparentPower", Type: "uint16", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x0510, Name: "PowerFactor", Type: "int8", Readable: true, Writable: false,
				Reportable: true},
		},
	},
	ClusterMetering: {
		ClusterID:     ClusterMetering,
		ClusterName:   "Metering",
		ComponentType: "pm",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "CurrentSummationDelivered", Type: "uint48", Readable: true,
				Writable: false, Reportable: true},
			{ID: 0x0001, Name: "CurrentSummationReceived", Type: "uint48", Readable: true,
				Writable: false, Reportable: true},
			{ID: 0x0400, Name: "InstantaneousDemand", Type: "int24", Readable: true,
				Writable: false, Reportable: true},
		},
	},
	//nolint:dupl // Zigbee cluster definitions have similar structure by design
	ClusterThermostat: {
		ClusterID:     ClusterThermostat,
		ClusterName:   "Thermostat",
		ComponentType: "thermostat",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "LocalTemperature", Type: "int16", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x0011, Name: "OccupiedCoolingSetpoint", Type: "int16", Readable: true,
				Writable: true, Reportable: true},
			{ID: 0x0012, Name: "OccupiedHeatingSetpoint", Type: "int16", Readable: true,
				Writable: true, Reportable: true},
			{ID: 0x001B, Name: "ControlSequenceOfOperation", Type: "enum8", Readable: true,
				Writable: true, Reportable: false},
			{ID: 0x001C, Name: "SystemMode", Type: "enum8", Readable: true, Writable: true,
				Reportable: true},
		},
		Commands: []ClusterCommand{
			{ID: 0x00, Name: "SetpointRaiseLower", Direction: "client_to_server"},
		},
	},
	ClusterIASZone: {
		ClusterID:     ClusterIASZone,
		ClusterName:   "IAS Zone",
		ComponentType: "input",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "ZoneState", Type: "enum8", Readable: true, Writable: false,
				Reportable: false},
			{ID: 0x0001, Name: "ZoneType", Type: "enum16", Readable: true, Writable: false,
				Reportable: false},
			{ID: 0x0002, Name: "ZoneStatus", Type: "bitmap16", Readable: true, Writable: false,
				Reportable: true},
		},
	},
	ClusterOccupancySensing: {
		ClusterID:     ClusterOccupancySensing,
		ClusterName:   "Occupancy Sensing",
		ComponentType: "input",
		Attributes: []ClusterAttribute{
			{ID: 0x0000, Name: "Occupancy", Type: "bitmap8", Readable: true, Writable: false,
				Reportable: true},
			{ID: 0x0001, Name: "OccupancySensorType", Type: "enum8", Readable: true,
				Writable: false, Reportable: false},
		},
	},
}

// GetClusterCapability returns the capability mapping for a Zigbee cluster.
// Returns nil if the cluster is not mapped.
func GetClusterCapability(clusterID uint16) *ClusterCapability {
	if capability, ok := ClusterMapping[clusterID]; ok {
		return &capability
	}
	return nil
}

// GetComponentTypeForCluster returns the Shelly component type for a Zigbee cluster.
// Returns an empty string if the cluster is not mapped.
func GetComponentTypeForCluster(clusterID uint16) string {
	if capability, ok := ClusterMapping[clusterID]; ok {
		return capability.ComponentType
	}
	return ""
}

// GetClustersForComponentType returns all Zigbee clusters that map to a component type.
func GetClustersForComponentType(componentType string) []ClusterCapability {
	var clusters []ClusterCapability
	for _, capability := range ClusterMapping {
		if capability.ComponentType == componentType {
			clusters = append(clusters, capability)
		}
	}
	return clusters
}

// deviceTypeClusters maps device types to their expected cluster IDs.
var deviceTypeClusters = map[DeviceType][]uint16{
	DeviceTypeOnOffSwitch:             {ClusterOnOff},
	DeviceTypeOnOffLight:              {ClusterOnOff},
	DeviceTypeLevelControllableOutput: {ClusterOnOff, ClusterLevelControl},
	DeviceTypeDimmableLight:           {ClusterOnOff, ClusterLevelControl},
	DeviceTypeDimmerSwitch:            {ClusterOnOff, ClusterLevelControl},
	DeviceTypeColorDimmableLight:      {ClusterOnOff, ClusterLevelControl, ClusterColorControl},
	DeviceTypeColorDimmerSwitch:       {ClusterOnOff, ClusterLevelControl, ClusterColorControl},
	DeviceTypeWindowCovering:          {ClusterWindowCovering},
	DeviceTypeThermostat:              {ClusterThermostat, ClusterTemperatureMeasurement},
	DeviceTypeTemperatureSensor:       {ClusterTemperatureMeasurement},
	DeviceTypeOccupancySensor:         {ClusterOccupancySensing},
	DeviceTypeContactSensor:           {ClusterIASZone},
	DeviceTypeFloodSensor:             {ClusterIASZone},
	DeviceTypeSmokeSensor:             {ClusterIASZone},
	DeviceTypePowerMeter:              {ClusterElectricalMeasurement, ClusterMetering},
}

// InferCapabilitiesFromDeviceType returns the expected cluster capabilities
// for a given Zigbee device type.
func InferCapabilitiesFromDeviceType(deviceType DeviceType) []ClusterCapability {
	// All devices have Basic cluster
	capabilities := []ClusterCapability{{
		ClusterID:     ClusterBasic,
		ClusterName:   "Basic",
		ComponentType: "sys",
	}}

	// Add device-type specific clusters
	if clusterIDs, ok := deviceTypeClusters[deviceType]; ok {
		for _, clusterID := range clusterIDs {
			if c := GetClusterCapability(clusterID); c != nil {
				capabilities = append(capabilities, *c)
			}
		}
	}

	return capabilities
}

// MapShellyModelToDeviceType maps a Shelly model identifier to a Zigbee device type.
// This is used when a Shelly device is operating in Zigbee mode.
func MapShellyModelToDeviceType(model string) DeviceType {
	// Shelly model mapping based on device capabilities
	switch model {
	// Gen4 switches/relays
	case "S3SW-001X16EU", "S3SW-001P16EU", "S3SW-002P16EU":
		return DeviceTypeOnOffSwitch

	// Gen4 mini switches
	case "S3SW-001X8EU", "S3SW-001P8EU":
		return DeviceTypeOnOffSwitch

	// Gen4 plug
	case "S3PL-00112US":
		return DeviceTypeOnOffSwitch

	// Dimmers
	case "S3DM-001P10EU":
		return DeviceTypeDimmerSwitch

	// Covers/Shutters
	case "S3SH-002P16EU":
		return DeviceTypeWindowCovering

	// Energy meters
	case "S3EM-002CXCEU":
		return DeviceTypePowerMeter

	// Sensors
	case "S3FL-001P01EU":
		return DeviceTypeFloodSensor

	default:
		// Default to on/off switch for unknown models
		return DeviceTypeOnOffSwitch
	}
}

// DeviceProfile represents the Zigbee profile of a Shelly device.
type DeviceProfile struct {
	SupportedClusters []uint16            `json:"supported_clusters"`
	Capabilities      []ClusterCapability `json:"capabilities"`
	DeviceType        DeviceType          `json:"device_type"`
}

// GetDeviceProfile returns the Zigbee profile for a Shelly device based on its model.
func GetDeviceProfile(model string) *DeviceProfile {
	deviceType := MapShellyModelToDeviceType(model)
	capabilities := InferCapabilitiesFromDeviceType(deviceType)

	supportedClusters := make([]uint16, 0, len(capabilities))
	for _, cap := range capabilities {
		supportedClusters = append(supportedClusters, cap.ClusterID)
	}

	return &DeviceProfile{
		DeviceType:        deviceType,
		SupportedClusters: supportedClusters,
		Capabilities:      capabilities,
	}
}

// deviceTypeNames maps device types to human-readable names.
var deviceTypeNames = map[DeviceType]string{
	DeviceTypeOnOffSwitch:             "On/Off Switch",
	DeviceTypeLevelControllableOutput: "Level Controllable Output",
	DeviceTypeOnOffLight:              "On/Off Light",
	DeviceTypeDimmableLight:           "Dimmable Light",
	DeviceTypeColorDimmableLight:      "Color Dimmable Light",
	DeviceTypeOnOffLightSwitch:        "On/Off Light Switch",
	DeviceTypeDimmerSwitch:            "Dimmer Switch",
	DeviceTypeColorDimmerSwitch:       "Color Dimmer Switch",
	DeviceTypeWindowCovering:          "Window Covering",
	DeviceTypeThermostat:              "Thermostat",
	DeviceTypeTemperatureSensor:       "Temperature Sensor",
	DeviceTypePumpController:          "Pump Controller",
	DeviceTypeOccupancySensor:         "Occupancy Sensor",
	DeviceTypeContactSensor:           "Contact Sensor",
	DeviceTypeFloodSensor:             "Flood Sensor",
	DeviceTypeSmokeSensor:             "Smoke Sensor",
	DeviceTypePowerMeter:              "Power Meter",
}

// String returns a human-readable description of the device type.
func (dt DeviceType) String() string {
	if name, ok := deviceTypeNames[dt]; ok {
		return name
	}
	return fmt.Sprintf("Unknown Device Type (0x%04X)", uint16(dt))
}
