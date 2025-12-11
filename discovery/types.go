package discovery

import (
	"net"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// DiscoveredDevice represents a device found during discovery.
type DiscoveredDevice struct {
	LastSeen     time.Time        `json:"last_seen"`
	Raw          any              `json:"-"`
	ID           string           `json:"id"`
	Name         string           `json:"name,omitempty"`
	Model        string           `json:"model,omitempty"`
	MACAddress   string           `json:"mac_address,omitempty"`
	Firmware     string           `json:"firmware,omitempty"`
	Protocol     Protocol         `json:"protocol"`
	Address      net.IP           `json:"address"`
	Generation   types.Generation `json:"generation"`
	Port         int              `json:"port"`
	AuthRequired bool             `json:"auth_required,omitempty"`
}

// URL returns the base URL for the device.
func (d *DiscoveredDevice) URL() string {
	port := d.Port
	if port == 0 {
		port = 80
	}
	return "http://" + d.Address.String() + ":" + itoa(port)
}

// Protocol represents the protocol used to discover a device.
type Protocol string

// Discovery protocol constants.
const (
	// ProtocolMDNS indicates discovery via mDNS/Zeroconf.
	ProtocolMDNS Protocol = "mdns"

	// ProtocolCoIoT indicates discovery via CoAP/CoIoT multicast.
	ProtocolCoIoT Protocol = "coiot"

	// ProtocolBLE indicates discovery via Bluetooth Low Energy.
	ProtocolBLE Protocol = "ble"

	// ProtocolWiFiAP indicates discovery via WiFi Access Point scanning.
	// This finds disconnected Shelly devices broadcasting their own network.
	ProtocolWiFiAP Protocol = "wifi_ap"

	// ProtocolManual indicates manual/probe-based discovery.
	ProtocolManual Protocol = "manual"
)

// Discoverer is the interface for discovering Shelly devices.
type Discoverer interface {
	// Discover searches for devices for the specified duration.
	// Returns all devices found during the scan period.
	Discover(timeout time.Duration) ([]DiscoveredDevice, error)

	// StartDiscovery begins continuous discovery.
	// Found devices are sent to the channel.
	StartDiscovery() (<-chan DiscoveredDevice, error)

	// StopDiscovery stops continuous discovery.
	StopDiscovery() error

	// Stop stops the discoverer and releases resources.
	Stop() error
}

// ScannerOption is a functional option for configuring the Scanner.
type ScannerOption func(*Scanner)

// Scanner provides unified device discovery across all protocols.
type Scanner struct {
	mdns        *MDNSDiscoverer
	coiot       *CoIoTDiscoverer
	ble         *BLEDiscoverer
	wifi        *WiFiDiscoverer
	devices     map[string]*DiscoveredDevice
	enableMDNS  bool
	enableCoIoT bool
	enableBLE   bool
	enableWiFi  bool
}

// NewScanner creates a new unified device scanner.
func NewScanner(opts ...ScannerOption) *Scanner {
	s := &Scanner{
		enableMDNS:  true,
		enableCoIoT: true,
		enableBLE:   false, // Disabled by default (requires platform-specific implementation)
		enableWiFi:  false, // Disabled by default (requires platform-specific implementation)
		devices:     make(map[string]*DiscoveredDevice),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithMDNS enables or disables mDNS discovery.
func WithMDNS(enable bool) ScannerOption {
	return func(s *Scanner) {
		s.enableMDNS = enable
	}
}

// WithCoIoT enables or disables CoIoT discovery.
func WithCoIoT(enable bool) ScannerOption {
	return func(s *Scanner) {
		s.enableCoIoT = enable
	}
}

// WithBLE enables BLE discovery with the provided scanner implementation.
func WithBLE(scanner BLEScanner) ScannerOption {
	return func(s *Scanner) {
		s.enableBLE = scanner != nil
		if scanner != nil {
			s.ble = NewBLEDiscovererWithScanner(scanner)
		}
	}
}

// WithWiFi enables WiFi AP discovery with the provided scanner implementation.
func WithWiFi(scanner WiFiScanner) ScannerOption {
	return func(s *Scanner) {
		s.enableWiFi = scanner != nil
		if scanner != nil {
			s.wifi = NewWiFiDiscovererWithScanner(scanner)
		}
	}
}

// Scan scans for devices using all enabled protocols.
// Returns a deduplicated list of discovered devices.
// Partial results are returned even if some discovery methods fail.
func (s *Scanner) Scan(timeout time.Duration) ([]DiscoveredDevice, error) {
	var allDevices []DiscoveredDevice

	// mDNS discovery (for Gen2+ devices)
	if s.enableMDNS {
		mdns := NewMDNSDiscoverer()
		devices, err := mdns.Discover(timeout)
		if err == nil {
			allDevices = append(allDevices, devices...)
		}
		// Ignore discovery errors - partial results are acceptable
		//nolint:errcheck // Stop cleanup errors don't affect discovery results
		mdns.Stop()
	}

	// CoIoT discovery (for Gen1 devices)
	if s.enableCoIoT {
		coiot := NewCoIoTDiscoverer()
		devices, err := coiot.Discover(timeout)
		if err == nil {
			allDevices = append(allDevices, devices...)
		}
		// Ignore discovery errors - partial results are acceptable
		//nolint:errcheck // Stop cleanup errors don't affect discovery results
		coiot.Stop()
	}

	// BLE discovery (for Gen2+ devices in provisioning mode)
	if s.enableBLE && s.ble != nil {
		devices, err := s.ble.Discover(timeout)
		if err == nil {
			allDevices = append(allDevices, devices...)
		}
	}

	// WiFi AP discovery (for disconnected devices)
	if s.enableWiFi && s.wifi != nil {
		devices, err := s.wifi.Discover(timeout)
		if err == nil {
			allDevices = append(allDevices, devices...)
		}
	}

	result := s.deduplicateDevices(allDevices)
	return result, nil
}

// deduplicateDevices removes duplicate devices, preferring more recent discoveries.
func (s *Scanner) deduplicateDevices(devices []DiscoveredDevice) []DiscoveredDevice {
	seen := make(map[string]*DiscoveredDevice)

	for i := range devices {
		d := &devices[i]
		key := d.ID
		if key == "" {
			key = d.MACAddress
		}
		if key == "" {
			key = d.Address.String()
		}

		existing, ok := seen[key]
		if !ok || d.LastSeen.After(existing.LastSeen) {
			seen[key] = d
		}
	}

	result := make([]DiscoveredDevice, 0, len(seen))
	for _, d := range seen {
		result = append(result, *d)
	}

	return result
}

// Stop stops the scanner and releases resources.
func (s *Scanner) Stop() error {
	var errs []error
	if s.mdns != nil {
		if err := s.mdns.Stop(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.coiot != nil {
		if err := s.coiot.Stop(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.ble != nil {
		if err := s.ble.Stop(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.wifi != nil {
		if err := s.wifi.Stop(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// itoa converts an int to a string (simple implementation to avoid strconv import).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var result []byte
	negative := n < 0
	if negative {
		n = -n
	}

	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}

	if negative {
		result = append([]byte{'-'}, result...)
	}

	return string(result)
}
