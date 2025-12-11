package integration

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/factory"
)

const (
	// EnvDiscoverySubnet is the subnet to scan for devices.
	// Format: "10.23.47.220-10.23.47.247" or "192.168.1.0/24"
	EnvDiscoverySubnet = "SHELLY_TEST_DISCOVERY_SUBNET"

	// DefaultDiscoveryRange is a default range for testing if user has devices.
	// Set to the user's known device range.
	DefaultDiscoveryRange = "10.23.47.220-10.23.47.247"
)

func TestDiscovery_ScanRange(t *testing.T) {
	SkipIfNoIntegration(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Generate addresses in the range
	addresses := generateAddressRange("10.23.47.220", "10.23.47.247")
	t.Logf("Scanning %d addresses in range 10.23.47.220-10.23.47.247", len(addresses))

	var (
		discovered []discovery.DeviceInfo
		mu         sync.Mutex
		wg         sync.WaitGroup
	)

	// Scan concurrently with limited parallelism
	sem := make(chan struct{}, 10) // 10 concurrent probes

	for _, addr := range addresses {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			info, err := discovery.IdentifyWithTimeout(ctx, address, 2*time.Second)
			if err != nil {
				return // Device not found or not reachable
			}

			mu.Lock()
			discovered = append(discovered, *info)
			mu.Unlock()
		}(addr)
	}

	wg.Wait()

	t.Logf("Discovered %d Shelly devices", len(discovered))

	for _, d := range discovered {
		t.Logf("  %s: %s (%s) Gen%d",
			d.ID, d.Model, d.Name, d.Generation)
	}

	if len(discovered) == 0 {
		t.Log("No devices found in range - this may be expected if devices are not available")
	}
}

func TestDiscovery_IdentifyDevice(t *testing.T) {
	SkipIfNoIntegration(t)

	// Use a specific device address if configured
	config := LoadConfig()
	var address string

	if config.Gen2Addr != "" {
		address = config.Gen2Addr
	} else if config.Gen1Addr != "" {
		address = config.Gen1Addr
	} else {
		t.Skip("No device address configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := discovery.Identify(ctx, address)
	if err != nil {
		t.Fatalf("Identify(%s) error = %v", address, err)
	}

	t.Logf("Identified device at %s:", address)
	t.Logf("  ID: %s", info.ID)
	t.Logf("  Model: %s", info.Model)
	t.Logf("  Name: %s", info.Name)
	t.Logf("  Generation: %d", info.Generation)
	t.Logf("  MAC: %s", info.MACAddress)
	t.Logf("  Firmware: %s", info.Firmware)
}

func TestDiscovery_Scanner(t *testing.T) {
	SkipIfNoIntegration(t)

	t.Log("Starting device scanner (10 second scan)...")

	scanner := discovery.NewScanner()
	devices, err := scanner.Scan(10 * time.Second)

	if err != nil {
		t.Fatalf("Scanner.Scan() error = %v", err)
	}

	t.Logf("Scanner discovered %d devices", len(devices))

	for _, d := range devices {
		t.Logf("  %s: %s (%s) at %s",
			d.ID, d.Model, d.Name, d.Address)
	}
}

func TestFactory_FromAddress(t *testing.T) {
	SkipIfNoIntegration(t)

	config := LoadConfig()
	var address string

	if config.Gen2Addr != "" {
		address = config.Gen2Addr
	} else if config.Gen1Addr != "" {
		address = config.Gen1Addr
	} else {
		t.Skip("No device address configured")
	}

	device, err := factory.FromAddress(address, factory.WithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("FromAddress(%s) error = %v", address, err)
	}

	t.Logf("Created device from address %s:", address)
	t.Logf("  Address: %s", device.Address())
	t.Logf("  Generation: %d", device.Generation())
}

func TestFactory_BatchFromAddresses(t *testing.T) {
	SkipIfNoIntegration(t)

	// Use a few addresses from the known range
	addresses := []string{
		"10.23.47.220",
		"10.23.47.221",
		"10.23.47.222",
		"10.23.47.223",
		"10.23.47.224",
	}

	t.Logf("Creating devices from %d addresses", len(addresses))

	devices, errs := factory.BatchFromAddresses(addresses, factory.WithTimeout(5*time.Second))

	var successCount, errorCount int
	for i, device := range devices {
		if device != nil {
			successCount++
			t.Logf("  %s: Gen%d", device.Address(), device.Generation())
		} else if i < len(errs) && errs[i] != nil {
			errorCount++
		}
	}

	t.Logf("Created %d devices, %d errors", successCount, errorCount)
}

func TestDiscovery_ScanSubnet(t *testing.T) {
	SkipIfNoIntegration(t)

	// Generate addresses for a /28 subnet
	baseIP := net.ParseIP("10.23.47.224").To4()
	if baseIP == nil {
		t.Fatal("Invalid base IP")
	}

	var addresses []string
	for i := 0; i < 16; i++ {
		ip := make(net.IP, 4)
		copy(ip, baseIP)
		ip[3] = baseIP[3] + byte(i)
		addresses = append(addresses, ip.String())
	}

	t.Logf("Scanning subnet with %d addresses", len(addresses))

	devices, _ := factory.BatchFromAddresses(addresses, factory.WithTimeout(5*time.Second))

	var validDevices []factory.Device
	for _, d := range devices {
		if d != nil {
			validDevices = append(validDevices, d)
		}
	}

	t.Logf("Found %d Shelly devices in subnet", len(validDevices))

	for _, d := range validDevices {
		t.Logf("  %s: Gen%d", d.Address(), d.Generation())
	}
}

// generateAddressRange generates IP addresses between start and end (inclusive).
func generateAddressRange(start, end string) []string {
	startIP := net.ParseIP(start).To4()
	endIP := net.ParseIP(end).To4()

	if startIP == nil || endIP == nil {
		return nil
	}

	var addresses []string
	for ip := startIP; !ipGreater(ip, endIP); incrementIP(ip) {
		addresses = append(addresses, ip.String())
	}

	return addresses
}

// incrementIP increments an IP address by 1.
func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

// ipGreater returns true if a > b.
func ipGreater(a, b net.IP) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
	}
	return false
}

func TestDiscovery_ListAllDevices(t *testing.T) {
	SkipIfNoIntegration(t)

	// Full scan of the known device range
	addresses := generateAddressRange("10.23.47.220", "10.23.47.247")

	t.Logf("Scanning for all Shelly devices in range (this may take a while)...")

	devices, _ := factory.BatchFromAddresses(addresses, factory.WithTimeout(5*time.Second))

	// Group by generation
	var gen1Count, gen2Count int
	for _, device := range devices {
		if device == nil {
			continue
		}

		switch device.Generation() {
		case 1:
			gen1Count++
			t.Logf("  Gen1: %s at %s", device.Address(), device.Address())
		default:
			gen2Count++
			t.Logf("  Gen2+: %s at %s", device.Address(), device.Address())
		}
	}

	t.Logf("\nSummary: Found %d Gen1 devices, %d Gen2+ devices", gen1Count, gen2Count)

	// Build environment variable suggestions
	if gen1Count > 0 || gen2Count > 0 {
		t.Log("\nTo run device-specific integration tests, set these environment variables:")
		t.Log("  export SHELLY_INTEGRATION_TESTS=1")

		for _, device := range devices {
			if device != nil && device.Generation() == 1 {
				t.Logf("  export SHELLY_TEST_GEN1_ADDR=%s", device.Address())
				break
			}
		}

		for _, device := range devices {
			if device != nil && device.Generation() >= 2 {
				t.Logf("  export SHELLY_TEST_GEN2_ADDR=%s", device.Address())
				break
			}
		}
	}
}

// TestDiscovery_DeviceInventory creates a detailed inventory of all discovered devices.
func TestDiscovery_DeviceInventory(t *testing.T) {
	SkipIfNoIntegration(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	addresses := generateAddressRange("10.23.47.220", "10.23.47.247")

	t.Log("\n=== Device Inventory ===\n")

	modelCounts := make(map[string]int)
	var totalDevices int

	for _, addr := range addresses {
		info, err := discovery.IdentifyWithTimeout(ctx, addr, 2*time.Second)
		if err != nil {
			continue
		}

		totalDevices++
		modelCounts[info.Model]++

		t.Logf("%-15s | %-20s | %-20s | Gen%d | %s",
			addr, info.ID, info.Model, info.Generation, info.Firmware)
	}

	t.Log("\n=== Summary by Model ===\n")
	for model, count := range modelCounts {
		t.Logf("  %-20s: %d", model, count)
	}
	t.Logf("\n  Total devices: %d", totalDevices)
}

// Example_identifyDevice demonstrates how to identify a Shelly device.
func Example_identifyDevice() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Scan a single address
	info, err := discovery.Identify(ctx, "10.23.47.220")
	if err != nil {
		fmt.Printf("Device not found: %v\n", err)
		return
	}

	fmt.Printf("Found device: %s (%s) Gen%d\n", info.ID, info.Model, info.Generation)
}
