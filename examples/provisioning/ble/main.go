//go:build linux || darwin

// Package main demonstrates BLE provisioning for Shelly devices.
//
// This example shows how to:
//   - Discover Shelly devices via BLE
//   - Connect to a device using the BLE transmitter
//   - Send WiFi credentials to provision the device
//   - Handle provisioning results and errors
//
// Requirements:
//   - Linux with BlueZ or macOS with CoreBluetooth
//   - Bluetooth adapter must be available and enabled
//   - A Shelly Gen2+ device in BLE provisioning mode
//
// Run with: go run main.go -ssid "YourWiFi" -password "YourPassword"
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/provisioning"
)

func main() {
	// Parse command line flags
	ssid := flag.String("ssid", "", "WiFi SSID to provision (required)")
	password := flag.String("password", "", "WiFi password")
	deviceAddr := flag.String("device", "", "BLE address of device to provision (optional, will discover if not provided)")
	deviceName := flag.String("name", "", "Device name to set (optional)")
	timezone := flag.String("timezone", "", "Timezone to set, e.g. 'America/New_York' (optional)")
	timeout := flag.Duration("timeout", 30*time.Second, "Provisioning timeout")
	flag.Parse()

	if *ssid == "" {
		fmt.Println("Error: -ssid is required")
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	fmt.Println("Shelly BLE Provisioning Example")
	fmt.Println("================================")
	fmt.Println()

	// Create a BLE transmitter for RPC communication
	transmitter, err := provisioning.NewTinyGoBLETransmitter()
	if err != nil {
		fmt.Printf("Failed to create BLE transmitter: %v\n", err)
		fmt.Println()
		fmt.Println("Troubleshooting:")
		fmt.Println("  - On Linux: Ensure bluez is installed and bluetooth is enabled")
		fmt.Println("  - On macOS: Ensure Bluetooth is enabled in System Preferences")
		fmt.Println("  - Make sure no other application is using the bluetooth adapter")
		os.Exit(1)
	}

	// Create a BLE provisioner
	prov := provisioning.NewBLEProvisioner()
	prov.Transmitter = transmitter

	// If no device address provided, discover devices first
	var targetAddr string
	if *deviceAddr == "" {
		fmt.Println("No device address provided. Discovering BLE devices...")
		fmt.Println()

		addr, err := discoverDevice(prov)
		if err != nil {
			fmt.Printf("Failed to discover device: %v\n", err)
			os.Exit(1)
		}
		targetAddr = addr
	} else {
		targetAddr = *deviceAddr
		// Register the device with the provisioner
		prov.AddDiscoveredDevice(&provisioning.BLEDevice{
			Address:  targetAddr,
			Name:     "Manual Entry",
			IsShelly: true,
		})
	}

	fmt.Printf("Provisioning device: %s\n", targetAddr)
	fmt.Printf("WiFi SSID: %s\n", *ssid)
	fmt.Printf("Timeout: %v\n", *timeout)
	fmt.Println()

	// Create provisioning context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Build provisioning config
	config := &provisioning.BLEProvisionConfig{
		WiFi: &provisioning.WiFiConfig{
			SSID:     *ssid,
			Password: *password,
		},
		DeviceName: *deviceName,
		Timezone:   *timezone,
	}

	fmt.Println("Starting provisioning...")
	result, err := prov.ProvisionViaBLE(ctx, targetAddr, config)
	if err != nil {
		fmt.Printf("\nProvisioning failed: %v\n", err)
		fmt.Println()
		fmt.Println("Troubleshooting:")
		fmt.Println("  - Ensure the device is in BLE provisioning mode")
		fmt.Println("  - Check that the WiFi credentials are correct")
		fmt.Println("  - Move closer to the device for better signal")
		fmt.Println("  - Try power cycling the device")
		os.Exit(1)
	}

	// Print result
	fmt.Println()
	fmt.Println("=== Provisioning Result ===")
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Duration: %v\n", result.Duration())
	fmt.Printf("Commands sent: %d\n", len(result.Commands))

	if result.Success {
		fmt.Println()
		fmt.Println("Device provisioned successfully!")
		fmt.Println("The device should now be accessible on your WiFi network.")
		if result.Device != nil {
			fmt.Printf("Device: %s (%s)\n", result.Device.Name, result.Device.Address)
		}
	} else {
		if result.Error != nil {
			fmt.Printf("Error: %v\n", result.Error)
		}
	}
}

// discoverDevice discovers BLE devices and prompts user to select one.
func discoverDevice(prov *provisioning.BLEProvisioner) (string, error) {
	// Create a BLE discoverer
	bleDiscoverer, err := discovery.NewBLEDiscoverer()
	if err != nil {
		return "", fmt.Errorf("failed to create BLE discoverer: %w", err)
	}

	bleDiscoverer.ScanDuration = 10 * time.Second
	bleDiscoverer.IncludeBTHome = false // Only look for provisionable devices

	fmt.Println("Scanning for Shelly devices (10 seconds)...")
	fmt.Println()

	// Discover devices
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = bleDiscoverer.DiscoverWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("discovery failed: %w", err)
	}

	// Get discovered devices
	devices := bleDiscoverer.GetDiscoveredDevices()
	if len(devices) == 0 {
		return "", fmt.Errorf("no Shelly devices found - ensure device is in BLE provisioning mode")
	}

	// Filter to connectable devices only
	var connectableDevices []discovery.BLEDiscoveredDevice
	for _, d := range devices {
		if d.Connectable {
			connectableDevices = append(connectableDevices, d)
		}
	}

	if len(connectableDevices) == 0 {
		return "", fmt.Errorf("no connectable Shelly devices found")
	}

	fmt.Printf("Found %d connectable device(s):\n", len(connectableDevices))
	fmt.Println()

	for i, d := range connectableDevices {
		fmt.Printf("  [%d] %s (%s) - RSSI: %d dBm\n",
			i+1, d.Name, d.MACAddress, d.RSSI)

		// Register with provisioner
		prov.AddDiscoveredDevice(&provisioning.BLEDevice{
			Address:  d.MACAddress,
			Name:     d.Name,
			Model:    d.Model,
			RSSI:     d.RSSI,
			IsShelly: true,
		})
	}
	fmt.Println()

	// If only one device, use it automatically
	if len(connectableDevices) == 1 {
		addr := connectableDevices[0].MACAddress
		fmt.Printf("Auto-selecting the only available device: %s\n", addr)
		return addr, nil
	}

	// Prompt user to select a device
	fmt.Printf("Enter device number (1-%d): ", len(connectableDevices))

	var selection int
	_, err = fmt.Scanf("%d", &selection)
	if err != nil || selection < 1 || selection > len(connectableDevices) {
		return "", fmt.Errorf("invalid selection")
	}

	addr := connectableDevices[selection-1].MACAddress
	return addr, nil
}
