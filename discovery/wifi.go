package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/schollz/wifiscan"

	"github.com/tj-smith47/shelly-go/types"
)

// WiFi AP discovery constants.
const (
	// ShellyAPPrefix is the SSID prefix for Shelly devices in AP mode.
	ShellyAPPrefix = "shelly"

	// DefaultAPIP is the default IP address of Shelly devices in AP mode.
	DefaultAPIP = "192.168.33.1"

	// DefaultAPPort is the default HTTP port for Shelly devices.
	DefaultAPPort = 80
)

// ShellyAPPattern matches Shelly AP SSIDs.
// Examples: shelly1-AABBCC, shellyplus1pm-123456, ShellyPro4PM-AABBCCDD
var ShellyAPPattern = regexp.MustCompile(`(?i)^shelly[a-z0-9]*[-_]?[a-f0-9]+$`)

// WiFiNetwork represents a discovered WiFi network.
type WiFiNetwork struct {
	LastSeen   time.Time `json:"last_seen"`
	SSID       string    `json:"ssid"`
	BSSID      string    `json:"bssid"`
	Security   string    `json:"security"`
	DeviceType string    `json:"device_type,omitempty"`
	DeviceID   string    `json:"device_id,omitempty"`
	Signal     int       `json:"signal"`
	Channel    int       `json:"channel"`
	IsShelly   bool      `json:"is_shelly"`
}

// WiFiDiscoveredDevice extends DiscoveredDevice with WiFi AP information.
type WiFiDiscoveredDevice struct {
	SSID          string `json:"ssid"`
	BSSID         string `json:"bssid"`
	Security      string `json:"security"`
	ConnectedSSID string `json:"connected_ssid,omitempty"`
	DiscoveredDevice
	Signal      int  `json:"signal"`
	Channel     int  `json:"channel"`
	IsConnected bool `json:"is_connected"`
}

// WiFiScanner is the interface for platform-specific WiFi scanning.
// The default implementation uses github.com/schollz/wifiscan.
type WiFiScanner interface {
	// Scan scans for available WiFi networks.
	Scan(ctx context.Context) ([]WiFiNetwork, error)

	// Connect connects to a WiFi network.
	Connect(ctx context.Context, ssid, password string) error

	// Disconnect disconnects from the current WiFi network.
	Disconnect(ctx context.Context) error

	// CurrentNetwork returns the currently connected network, if any.
	CurrentNetwork(ctx context.Context) (*WiFiNetwork, error)
}

// wifiscanScanner implements WiFiScanner using github.com/schollz/wifiscan.
// This is the default scanner used by WiFiDiscoverer.
// Note: Connect/Disconnect are not implemented - they require platform-specific code.
// Platform-specific scanners (platformWiFiScanner) embed this type and add connect support.
type wifiscanScanner struct{}

// newDefaultWiFiScanner creates the default WiFi scanner with platform-specific
// connect/disconnect support. Falls back to scan-only on unsupported platforms.
func newDefaultWiFiScanner() WiFiScanner {
	return newPlatformWiFiScanner()
}

// Scan scans for available WiFi networks.
func (s *wifiscanScanner) Scan(ctx context.Context) ([]WiFiNetwork, error) {
	// Check for cancellation first
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Perform the scan
	networks, err := wifiscan.Scan()
	if err != nil {
		return nil, &WiFiError{Message: "wifi scan failed", Err: err}
	}

	// Convert to our format
	result := make([]WiFiNetwork, 0, len(networks))
	for _, net := range networks {
		result = append(result, WiFiNetwork{
			SSID:   net.SSID,
			Signal: net.RSSI,
			// wifiscan only provides SSID and RSSI
		})
	}

	return result, nil
}

// Connect connects to a WiFi network.
// Note: Not implemented - requires platform-specific tools (nmcli, netsh, etc.).
func (s *wifiscanScanner) Connect(ctx context.Context, ssid, password string) error {
	return &WiFiError{Message: "Connect not implemented - use platform-specific tools (nmcli, netsh, etc.)"}
}

// Disconnect disconnects from the current WiFi network.
// Note: Not implemented - requires platform-specific tools.
func (s *wifiscanScanner) Disconnect(ctx context.Context) error {
	return &WiFiError{Message: "Disconnect not implemented - use platform-specific tools"}
}

// CurrentNetwork returns the currently connected network.
// Note: Not implemented - requires platform-specific tools.
func (s *wifiscanScanner) CurrentNetwork(ctx context.Context) (*WiFiNetwork, error) {
	return nil, &WiFiError{Message: "CurrentNetwork not implemented - use platform-specific tools"}
}

// WiFi error sentinels.
var (
	// ErrWiFiNotSupported indicates WiFi scanning is not available.
	ErrWiFiNotSupported = &WiFiError{Message: "WiFi scanning not supported on this platform - set Scanner field"}

	// ErrSSIDNotFound indicates the requested SSID was not found in scan results.
	ErrSSIDNotFound = &WiFiError{Message: "SSID not found in scan results"}

	// ErrAuthFailed indicates WiFi authentication failed (wrong password).
	ErrAuthFailed = &WiFiError{Message: "WiFi authentication failed - check password"}

	// ErrToolNotFound indicates no WiFi connection tool is available.
	ErrToolNotFound = &WiFiError{Message: "no WiFi connection tool available; " +
		"linux requires nmcli, wpa_cli, or iwconfig; " +
		"macOS requires networksetup; " +
		"windows requires netsh"}

	// ErrConnectionTimeout indicates the WiFi connection attempt timed out.
	ErrConnectionTimeout = &WiFiError{Message: "WiFi connection timeout - device may be out of range"}
)

// WiFiError represents a WiFi-related error.
type WiFiError struct {
	Err     error
	Message string
}

func (e *WiFiError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *WiFiError) Unwrap() error {
	return e.Err
}

// WiFiDiscoverer discovers Shelly devices via their WiFi Access Points.
//
// When a Shelly device cannot connect to WiFi, it creates its own AP
// with an SSID like "shelly1pm-AABBCC". This discoverer scans for these
// networks to find disconnected/unconfigured devices.
//
// Discovery flow:
//  1. Scan for WiFi networks
//  2. Filter for Shelly AP SSIDs (matching ShellyAPPattern)
//  3. Optionally connect and probe for device info
type WiFiDiscoverer struct {
	Scanner        WiFiScanner
	devices        map[string]*WiFiDiscoveredDevice
	devicesCh      chan DiscoveredDevice
	stopCh         chan struct{}
	OnNetworkFound func(*WiFiNetwork)
	OnDeviceFound  func(*WiFiDiscoveredDevice)
	HTTPClient     *http.Client
	ProbeTimeout   time.Duration
	mu             sync.RWMutex
	running        bool
	ProbeDevices   bool
}

// NewWiFiDiscoverer creates a new WiFi AP discoverer with platform-specific
// connect/disconnect support. On Linux, uses nmcli/wpa_cli/iwconfig.
// On macOS, uses networksetup. On Windows, uses netsh.
func NewWiFiDiscoverer() *WiFiDiscoverer {
	return &WiFiDiscoverer{
		Scanner:      newDefaultWiFiScanner(),
		devices:      make(map[string]*WiFiDiscoveredDevice),
		ProbeTimeout: 10 * time.Second,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// NewWiFiDiscovererWithScanner creates a WiFi discoverer with a custom scanner.
// Use this if you want to provide your own WiFiScanner implementation.
func NewWiFiDiscovererWithScanner(scanner WiFiScanner) *WiFiDiscoverer {
	return &WiFiDiscoverer{
		Scanner:      scanner,
		devices:      make(map[string]*WiFiDiscoveredDevice),
		ProbeTimeout: 10 * time.Second,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Discover scans for Shelly WiFi APs for the specified duration.
func (w *WiFiDiscoverer) Discover(timeout time.Duration) ([]DiscoveredDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return w.DiscoverWithContext(ctx)
}

// DiscoverWithContext scans for Shelly WiFi APs until the context is canceled.
func (w *WiFiDiscoverer) DiscoverWithContext(ctx context.Context) ([]DiscoveredDevice, error) {
	if w.Scanner == nil {
		return nil, ErrWiFiNotSupported
	}

	networks, err := w.Scanner.Scan(ctx)
	if err != nil {
		return nil, &WiFiError{Message: "WiFi scan failed", Err: err}
	}

	devices := make([]DiscoveredDevice, 0, len(networks))

	for i := range networks {
		network := &networks[i]

		if !IsShellyAP(network.SSID) {
			continue
		}

		// Parse device info from SSID
		network.IsShelly = true
		network.DeviceType, network.DeviceID = ParseShellySSID(network.SSID)
		network.LastSeen = time.Now()

		// Notify callback
		if w.OnNetworkFound != nil {
			w.OnNetworkFound(network)
		}

		// Create basic device from network info
		device := w.networkToDevice(network)

		// Optionally probe for more details
		if w.ProbeDevices {
			probed, err := w.probeDevice(ctx, network)
			if err == nil && probed != nil {
				device = probed
			}
		}

		devices = append(devices, device.DiscoveredDevice)

		// Store device
		w.mu.Lock()
		w.devices[network.SSID] = device
		w.mu.Unlock()

		// Notify callback
		if w.OnDeviceFound != nil {
			w.OnDeviceFound(device)
		}
	}

	return devices, nil
}

// networkToDevice creates a WiFiDiscoveredDevice from a WiFiNetwork.
func (w *WiFiDiscoverer) networkToDevice(network *WiFiNetwork) *WiFiDiscoveredDevice {
	deviceType, deviceID := ParseShellySSID(network.SSID)

	return &WiFiDiscoveredDevice{
		DiscoveredDevice: DiscoveredDevice{
			ID:         deviceID,
			Name:       network.SSID,
			Model:      deviceType,
			MACAddress: network.BSSID,
			Protocol:   ProtocolWiFiAP,
			Address:    net.ParseIP(DefaultAPIP),
			Port:       DefaultAPPort,
			Generation: InferGenerationFromModel(deviceType),
			LastSeen:   network.LastSeen,
		},
		SSID:     network.SSID,
		BSSID:    network.BSSID,
		Signal:   network.Signal,
		Channel:  network.Channel,
		Security: network.Security,
	}
}

// probeDevice connects to a Shelly AP and probes for device information.
// WARNING: This temporarily disconnects from your current network!
func (w *WiFiDiscoverer) probeDevice(ctx context.Context, network *WiFiNetwork) (*WiFiDiscoveredDevice, error) {
	// Remember current network (ignore error - may not be connected)
	originalNetwork, _ := w.Scanner.CurrentNetwork(ctx) //nolint:errcheck // Optional - may not be connected

	// Connect to Shelly AP (usually open/no password)
	if err := w.Scanner.Connect(ctx, network.SSID, ""); err != nil {
		return nil, &WiFiError{Message: "failed to connect to Shelly AP", Err: err}
	}

	// Ensure we reconnect to original network
	defer func() {
		if originalNetwork != nil {
			//nolint:errcheck // Best-effort reconnect
			w.Scanner.Connect(ctx, originalNetwork.SSID, "")
		}
	}()

	// Wait a moment for DHCP
	time.Sleep(2 * time.Second)

	// Probe the device
	device := w.networkToDevice(network)
	device.IsConnected = true

	// Try Gen2+ endpoint first
	info, err := w.probeGen2(ctx)
	if err == nil {
		w.applyGen2Info(device, info)
		return device, nil
	}

	// Try Gen1 endpoint
	info1, err := w.probeGen1(ctx)
	if err == nil {
		w.applyGen1Info(device, info1)
		return device, nil
	}

	// Return device with basic info even if probe failed
	return device, nil
}

// probeEndpoint fetches JSON from a device endpoint.
func (w *WiFiDiscoverer) probeEndpoint(ctx context.Context, endpoint string) (map[string]any, error) {
	url := fmt.Sprintf("http://%s%s", DefaultAPIP, endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := w.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var info map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return info, nil
}

// probeGen2 probes a Gen2+ device for information.
func (w *WiFiDiscoverer) probeGen2(ctx context.Context) (map[string]any, error) {
	return w.probeEndpoint(ctx, "/rpc/Shelly.GetDeviceInfo")
}

// probeGen1 probes a Gen1 device for information.
func (w *WiFiDiscoverer) probeGen1(ctx context.Context) (map[string]any, error) {
	return w.probeEndpoint(ctx, "/shelly")
}

// applyGen2Info applies Gen2 device info to a WiFiDiscoveredDevice.
func (w *WiFiDiscoverer) applyGen2Info(device *WiFiDiscoveredDevice, info map[string]any) {
	if name, ok := info["name"].(string); ok {
		device.Name = name
	}
	if model, ok := info["model"].(string); ok {
		device.Model = model
	}
	if mac, ok := info["mac"].(string); ok {
		device.MACAddress = mac
	}
	if fw, ok := info["fw_id"].(string); ok {
		device.Firmware = fw
	}
	if gen, ok := info["gen"].(float64); ok {
		device.Generation = types.Generation(int(gen))
	}
	if auth, ok := info["auth_en"].(bool); ok {
		device.AuthRequired = auth
	}
	if id, ok := info["id"].(string); ok {
		device.ID = id
	}
}

// applyGen1Info applies Gen1 device info to a WiFiDiscoveredDevice.
func (w *WiFiDiscoverer) applyGen1Info(device *WiFiDiscoveredDevice, info map[string]any) {
	if model, ok := info["type"].(string); ok {
		device.Model = model
	}
	if mac, ok := info["mac"].(string); ok {
		device.MACAddress = mac
		if device.ID == "" {
			device.ID = strings.ReplaceAll(mac, ":", "")
		}
	}
	if fw, ok := info["fw"].(string); ok {
		device.Firmware = fw
	}
	if auth, ok := info["auth"].(bool); ok {
		device.AuthRequired = auth
	}
	device.Generation = types.Gen1
}

// StartDiscovery begins continuous WiFi AP discovery.
func (w *WiFiDiscoverer) StartDiscovery() (<-chan DiscoveredDevice, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return w.devicesCh, nil
	}

	if w.Scanner == nil {
		return nil, ErrWiFiNotSupported
	}

	w.devicesCh = make(chan DiscoveredDevice, 100)
	w.stopCh = make(chan struct{})
	w.running = true

	go w.continuousDiscovery()

	return w.devicesCh, nil
}

// continuousDiscovery runs continuous WiFi AP discovery.
func (w *WiFiDiscoverer) continuousDiscovery() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			//nolint:errcheck // Continuous discovery ignores errors
			devices, _ := w.DiscoverWithContext(ctx)
			cancel()

			for i := range devices {
				select {
				case w.devicesCh <- devices[i]:
				default:
				}
			}
		}
	}
}

// StopDiscovery stops continuous WiFi AP discovery.
func (w *WiFiDiscoverer) StopDiscovery() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return nil
	}

	close(w.stopCh)
	w.running = false

	return nil
}

// Stop stops the discoverer and releases resources.
func (w *WiFiDiscoverer) Stop() error {
	return w.StopDiscovery()
}

// ScanNetworks scans for all WiFi networks and returns Shelly APs.
func (w *WiFiDiscoverer) ScanNetworks(ctx context.Context) ([]WiFiNetwork, error) {
	if w.Scanner == nil {
		return nil, ErrWiFiNotSupported
	}

	networks, err := w.Scanner.Scan(ctx)
	if err != nil {
		return nil, err
	}

	shellyNetworks := make([]WiFiNetwork, 0, len(networks))
	for i := range networks {
		if IsShellyAP(networks[i].SSID) {
			networks[i].IsShelly = true
			networks[i].DeviceType, networks[i].DeviceID = ParseShellySSID(networks[i].SSID)
			networks[i].LastSeen = time.Now()
			shellyNetworks = append(shellyNetworks, networks[i])
		}
	}

	return shellyNetworks, nil
}

// IsShellyAP checks if an SSID belongs to a Shelly device in AP mode.
func IsShellyAP(ssid string) bool {
	return ShellyAPPattern.MatchString(ssid)
}

// ParseShellySSID parses a Shelly AP SSID into device type and ID.
// Example: "shellyplus1pm-AABBCC" -> ("plus1pm", "AABBCC")
func ParseShellySSID(ssid string) (deviceType, deviceID string) {
	ssid = strings.ToLower(ssid)

	// Remove "shelly" prefix
	ssid = strings.TrimPrefix(ssid, "shelly")

	// Split on hyphen or underscore
	parts := strings.FieldsFunc(ssid, func(r rune) bool {
		return r == '-' || r == '_'
	})

	if len(parts) >= 2 {
		deviceType = parts[0]
		deviceID = parts[len(parts)-1]
	} else if len(parts) == 1 {
		// Try to split alphanumeric from hex
		for i := len(parts[0]) - 1; i >= 0; i-- {
			if !isHexDigit(parts[0][i]) {
				deviceType = parts[0][:i+1]
				deviceID = parts[0][i+1:]
				return
			}
		}
		deviceID = parts[0]
	}

	return deviceType, strings.ToUpper(deviceID)
}

// InferGenerationFromModel infers the device generation from the model name.
func InferGenerationFromModel(model string) types.Generation {
	model = strings.ToLower(model)

	// Gen4 indicators
	if strings.Contains(model, "g4") || strings.Contains(model, "gen4") {
		return types.Gen4
	}

	// Gen3 indicators
	if strings.Contains(model, "g3") || strings.Contains(model, "gen3") {
		return types.Gen3
	}

	// Gen2/Plus/Pro indicators
	if strings.Contains(model, "plus") ||
		strings.Contains(model, "pro") ||
		strings.Contains(model, "g2") ||
		strings.Contains(model, "gen2") {
		return types.Gen2
	}

	// Default to Gen1 for unknown
	return types.Gen1
}

// GetDiscoveredDevices returns all currently discovered WiFi AP devices.
func (w *WiFiDiscoverer) GetDiscoveredDevices() []WiFiDiscoveredDevice {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make([]WiFiDiscoveredDevice, 0, len(w.devices))
	for _, d := range w.devices {
		result = append(result, *d)
	}
	return result
}

// Clear clears all discovered devices.
func (w *WiFiDiscoverer) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.devices = make(map[string]*WiFiDiscoveredDevice)
}
