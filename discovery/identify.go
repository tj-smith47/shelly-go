package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// DeviceInfo contains information about an identified device.
type DeviceInfo struct {
	Raw          any              `json:"-"`
	ID           string           `json:"id"`
	Name         string           `json:"name,omitempty"`
	Model        string           `json:"model"`
	Firmware     string           `json:"firmware,omitempty"`
	MACAddress   string           `json:"mac_address,omitempty"`
	App          string           `json:"app,omitempty"`
	Profile      string           `json:"profile,omitempty"`
	Generation   types.Generation `json:"generation"`
	AuthRequired bool             `json:"auth_required"`
}

// Identify identifies a device at the given address.
//
// This function probes the device to determine its generation and
// extract device information. It first tries the Gen2+ /shelly endpoint,
// then falls back to the Gen1 /shelly endpoint.
func Identify(ctx context.Context, address string) (*DeviceInfo, error) {
	return IdentifyWithTimeout(ctx, address, 5*time.Second)
}

// IdentifyWithTimeout identifies a device with a custom timeout.
func IdentifyWithTimeout(ctx context.Context, address string, timeout time.Duration) (*DeviceInfo, error) {
	client := &http.Client{Timeout: timeout}

	// Normalize address
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}
	address = strings.TrimSuffix(address, "/")

	// Try Gen2+ endpoint first (returns JSON with gen field)
	info, err := identifyGen2(ctx, client, address)
	if err == nil {
		return info, nil
	}

	// Fallback to Gen1 endpoint
	return identifyGen1(ctx, client, address)
}

// identifyGen2 tries to identify a Gen2+ device.
func identifyGen2(ctx context.Context, client *http.Client, address string) (*DeviceInfo, error) {
	url := address + "/shelly"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data gen2ShellyResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	// Check if this is a Gen2+ device
	if data.Gen < 2 {
		return nil, fmt.Errorf("not a Gen2+ device")
	}

	info := &DeviceInfo{
		ID:           data.ID,
		Name:         data.Name,
		Model:        data.Model,
		Firmware:     data.FWVersion,
		MACAddress:   data.MAC,
		AuthRequired: data.AuthEn,
		App:          data.App,
		Profile:      data.Profile,
		Raw:          data,
	}

	// Set generation
	switch data.Gen {
	case 2:
		info.Generation = types.Gen2
	case 3:
		info.Generation = types.Gen3
	case 4:
		info.Generation = types.Gen4
	default:
		info.Generation = types.Gen2
	}

	return info, nil
}

// identifyGen1 tries to identify a Gen1 device.
func identifyGen1(ctx context.Context, client *http.Client, address string) (*DeviceInfo, error) {
	url := address + "/shelly"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data gen1ShellyResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	info := &DeviceInfo{
		ID:           data.MAC,
		Model:        data.Type,
		Firmware:     data.FW,
		MACAddress:   data.MAC,
		AuthRequired: data.Auth,
		Generation:   types.Gen1,
		Raw:          data,
	}

	// Try to get device name from settings
	if name := getGen1Name(ctx, client, address); name != "" {
		info.Name = name
	}

	return info, nil
}

// getGen1Name retrieves the device name from Gen1 settings.
// Gen1 devices store the user-configured name at the root level "name" field.
// The device.hostname is the auto-generated name (e.g., "shelly1pm-C45BBE6C4DDC").
func getGen1Name(ctx context.Context, client *http.Client, address string) string {
	url := address + "/settings"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return ""
	}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var settings map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return ""
	}

	// First check the root-level "name" field (user-configured name)
	if name, ok := settings["name"].(string); ok && name != "" {
		return name
	}

	// Fall back to device hostname if no name is set
	if device, ok := settings["device"].(map[string]any); ok {
		if name, ok := device["hostname"].(string); ok {
			return name
		}
	}

	return ""
}

// gen2ShellyResponse represents the Gen2+ /shelly response.
type gen2ShellyResponse struct {
	Name      string `json:"name"`
	ID        string `json:"id"`
	MAC       string `json:"mac"`
	Model     string `json:"model"`
	FWVersion string `json:"fw_id"`
	Ver       string `json:"ver"`
	App       string `json:"app"`
	Profile   string `json:"profile,omitempty"`
	AuthDom   string `json:"auth_domain,omitempty"`
	Slot      int    `json:"slot"`
	Gen       int    `json:"gen"`
	AuthEn    bool   `json:"auth_en"`
}

// gen1ShellyResponse represents the Gen1 /shelly response.
type gen1ShellyResponse struct {
	Type         string `json:"type"`
	MAC          string `json:"mac"`
	FW           string `json:"fw"`
	LongID       int    `json:"longid"`
	NumOutputs   int    `json:"num_outputs"`
	NumMeters    int    `json:"num_meters"`
	NumRollers   int    `json:"num_rollers"`
	Auth         bool   `json:"auth"`
	Discoverable bool   `json:"discoverable"`
}

// ProbeProgress represents the progress of a probe operation.
type ProbeProgress struct {
	Device  *DiscoveredDevice // Device found (nil if not found)
	Error   error             // Error if probe failed
	Address string            // IP address being probed
	Total   int               // Total number of addresses to probe
	Done    int               // Number of addresses probed so far
	Found   bool              // Whether a device was found at this address
}

// ProbeProgressCallback is called for each address probed.
// Return false to cancel the probe operation.
type ProbeProgressCallback func(progress ProbeProgress) bool

// ProbeAddresses probes a range of addresses to find Shelly devices.
//
// This is useful for finding devices on a network when mDNS/CoIoT
// discovery is not available or not working.
func ProbeAddresses(ctx context.Context, addresses []string) []DiscoveredDevice {
	return ProbeAddressesWithProgress(ctx, addresses, nil)
}

// ProbeAddressesWithProgress probes addresses with progress reporting.
//
// The callback is called for each address after probing, with information
// about whether a device was found. Return false from the callback to
// cancel the operation.
func ProbeAddressesWithProgress(
	ctx context.Context,
	addresses []string,
	callback ProbeProgressCallback,
) []DiscoveredDevice {
	var devices []DiscoveredDevice
	var mu sync.Mutex
	var wg sync.WaitGroup
	var done int
	var canceled bool

	total := len(addresses)

	// Limit concurrent probes
	sem := make(chan struct{}, 20)

	for _, addr := range addresses {
		// Check if canceled
		mu.Lock()
		if canceled {
			mu.Unlock()
			break
		}
		mu.Unlock()

		wg.Add(1)
		go func(address string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			// Check context cancellation
			if ctx.Err() != nil {
				return
			}

			info, err := IdentifyWithTimeout(ctx, address, 2*time.Second)

			var device *DiscoveredDevice
			if err == nil {
				d := DiscoveredDevice{
					ID:           info.ID,
					Name:         info.Name,
					Model:        info.Model,
					Generation:   info.Generation,
					Address:      parseIP(address),
					Port:         80,
					MACAddress:   info.MACAddress,
					Firmware:     info.Firmware,
					AuthRequired: info.AuthRequired,
					Protocol:     ProtocolManual,
					LastSeen:     time.Now(),
					Raw:          info.Raw,
				}
				device = &d

				mu.Lock()
				devices = append(devices, d)
				mu.Unlock()
			}

			// Report progress
			if callback != nil {
				mu.Lock()
				done++
				currentDone := done
				mu.Unlock()

				progress := ProbeProgress{
					Address: address,
					Total:   total,
					Done:    currentDone,
					Found:   device != nil,
					Device:  device,
					Error:   err,
				}

				if !callback(progress) {
					mu.Lock()
					canceled = true
					mu.Unlock()
				}
			}
		}(addr)
	}

	wg.Wait()
	return devices
}

// GenerateSubnetAddresses generates all addresses in a subnet.
//
// Example: GenerateSubnetAddresses("192.168.1.0/24") returns
// ["192.168.1.1", "192.168.1.2", ..., "192.168.1.254"]
func GenerateSubnetAddresses(cidr string) []string {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}

	var addresses []string
	networkIP := ipnet.IP.Mask(ipnet.Mask)

	// Calculate broadcast address
	broadcast := make(net.IP, len(networkIP))
	for i := range networkIP {
		broadcast[i] = networkIP[i] | ^ipnet.Mask[i]
	}

	ip := make(net.IP, len(networkIP))
	copy(ip, networkIP)

	for {
		ip = nextIP(ip)
		if !ipnet.Contains(ip) {
			break
		}

		// Skip broadcast address
		if ip.Equal(broadcast) {
			continue
		}

		addresses = append(addresses, ip.String())
	}

	return addresses
}

// nextIP returns the next IP address.
func nextIP(ip net.IP) net.IP {
	next := make(net.IP, len(ip))
	copy(next, ip)

	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] > 0 {
			break
		}
	}

	return next
}

// parseIP parses an IP address from a string.
func parseIP(s string) net.IP {
	// Remove protocol prefix
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "https://")

	// Remove port
	if idx := strings.LastIndex(s, ":"); idx != -1 {
		s = s[:idx]
	}

	return net.ParseIP(s)
}
