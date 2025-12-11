//go:build linux || darwin

// Package main demonstrates BLE discovery for Shelly devices.
//
// This example shows how to:
//   - Create a BLE discoverer with the TinyGo bluetooth scanner
//   - Scan for Shelly devices in provisioning mode
//   - Parse BTHome sensor data from Shelly BLU devices
//   - Handle discovered devices with callbacks
//
// Requirements:
//   - Linux with BlueZ or macOS with CoreBluetooth
//   - Bluetooth adapter must be available and enabled
//
// Run with: go run main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
)

func main() {
	fmt.Println("Shelly BLE Discovery Example")
	fmt.Println("=============================")
	fmt.Println()

	// Create a new BLE discoverer with the default TinyGo scanner
	discoverer, err := discovery.NewBLEDiscoverer()
	if err != nil {
		fmt.Printf("Failed to create BLE discoverer: %v\n", err)
		fmt.Println()
		fmt.Println("Troubleshooting:")
		fmt.Println("  - On Linux: Ensure bluez is installed and bluetooth is enabled")
		fmt.Println("  - On macOS: Ensure Bluetooth is enabled in System Preferences")
		fmt.Println("  - Make sure no other application is using the bluetooth adapter")
		os.Exit(1)
	}

	// Configure the discoverer
	discoverer.ScanDuration = 10 * time.Second
	discoverer.IncludeBTHome = true // Include Shelly BLU devices

	// Set up a callback for discovered devices
	discoverer.OnDeviceFound = func(device *discovery.BLEDiscoveredDevice) {
		printDevice(device)
	}

	// Handle interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Starting BLE discovery...")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Method 1: One-shot discovery with timeout
	fmt.Println("=== One-shot Discovery (10 seconds) ===")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	devices, err := discoverer.DiscoverWithContext(ctx)
	if err != nil {
		fmt.Printf("Discovery error: %v\n", err)
	}

	fmt.Printf("\nFound %d device(s)\n\n", len(devices))

	// Method 2: Continuous discovery (uncomment to use)
	/*
		fmt.Println("=== Continuous Discovery ===")
		deviceCh, err := discoverer.StartDiscovery()
		if err != nil {
			fmt.Printf("Failed to start continuous discovery: %v\n", err)
			os.Exit(1)
		}

		// Process devices as they're discovered
		go func() {
			for device := range deviceCh {
				fmt.Printf("Discovered: %s (%s)\n", device.Name, device.ID)
			}
		}()

		// Wait for interrupt
		<-sigCh
		fmt.Println("\nStopping discovery...")
		discoverer.StopDiscovery()
	*/

	// Print summary of all discovered devices
	fmt.Println("=== Summary ===")
	allDevices := discoverer.GetDiscoveredDevices()
	if len(allDevices) == 0 {
		fmt.Println("No Shelly devices found.")
		fmt.Println()
		fmt.Println("Tips:")
		fmt.Println("  - Ensure your Shelly device is in BLE provisioning mode")
		fmt.Println("  - For Gen2+ devices: Reset to factory or enable BLE in settings")
		fmt.Println("  - For BLU devices: They broadcast continuously when powered on")
		fmt.Println("  - Move closer to the device for better signal")
	} else {
		for _, device := range allDevices {
			printDeviceSummary(&device)
		}
	}

	fmt.Println("\nDone.")
}

// printDevice prints detailed information about a discovered device.
func printDevice(device *discovery.BLEDiscoveredDevice) {
	fmt.Println("----------------------------------------")
	fmt.Printf("Device Found!\n")
	fmt.Printf("  Name:        %s\n", device.Name)
	fmt.Printf("  MAC Address: %s\n", device.MACAddress)
	fmt.Printf("  RSSI:        %d dBm\n", device.RSSI)
	fmt.Printf("  Connectable: %v\n", device.Connectable)
	fmt.Printf("  Model:       %s\n", device.Model)
	fmt.Printf("  Generation:  %s\n", device.Generation)
	fmt.Printf("  Protocol:    %s\n", device.Protocol)

	if device.ServiceUUID != "" {
		fmt.Printf("  Service UUID: %s\n", device.ServiceUUID)
	}

	// Print BTHome sensor data if available
	if device.BTHomeData != nil {
		fmt.Println("  BTHome Sensor Data:")
		printBTHomeData(device.BTHomeData)
	}

	fmt.Println("----------------------------------------")
}

// printDeviceSummary prints a one-line summary of a device.
func printDeviceSummary(device *discovery.BLEDiscoveredDevice) {
	provisionStatus := "unprovisioned"
	if discovery.IsDeviceProvisioned(device) {
		provisionStatus = fmt.Sprintf("provisioned (%s)", device.Address.String())
	}

	fmt.Printf("  %s (%s) - RSSI: %d dBm - %s\n",
		device.Name,
		device.MACAddress,
		device.RSSI,
		provisionStatus,
	)
}

// printBTHomeData prints BTHome sensor data.
func printBTHomeData(data *discovery.BTHomeData) {
	fmt.Printf("    Packet ID: %d\n", data.PacketID)

	if data.Battery != nil {
		fmt.Printf("    Battery: %d%%\n", *data.Battery)
	}
	if data.Temperature != nil {
		fmt.Printf("    Temperature: %.1f°C\n", *data.Temperature)
	}
	if data.Humidity != nil {
		fmt.Printf("    Humidity: %.1f%%\n", *data.Humidity)
	}
	if data.Illuminance != nil {
		fmt.Printf("    Illuminance: %d lux\n", *data.Illuminance)
	}
	if data.Motion != nil {
		if *data.Motion {
			fmt.Println("    Motion: detected")
		} else {
			fmt.Println("    Motion: clear")
		}
	}
	if data.WindowOpen != nil {
		if *data.WindowOpen {
			fmt.Println("    Window: open")
		} else {
			fmt.Println("    Window: closed")
		}
	}
	if data.Button != nil {
		fmt.Printf("    Button: event %d\n", *data.Button)
	}
	if data.Rotation != nil {
		fmt.Printf("    Rotation: %.1f°\n", *data.Rotation)
	}
}
