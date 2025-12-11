// Example: mdns demonstrates discovering Shelly devices on the local network via mDNS.
//
// This example shows how to:
//   - Discover Shelly devices using mDNS/Zeroconf
//   - Parse discovered device information
//   - Identify device generation and model
//   - Create device instances from discovery results
//
// Usage:
//
//	go run main.go [-timeout 5s] [-continuous]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/factory"
	"github.com/tj-smith47/shelly-go/types"
)

func main() {
	// Parse command line flags
	timeout := flag.Duration("timeout", 5*time.Second, "Discovery timeout")
	continuous := flag.Bool("continuous", false, "Run continuous discovery")
	createDevices := flag.Bool("create", false, "Create device instances from results")
	flag.Parse()

	fmt.Println("Shelly Device Discovery via mDNS")
	fmt.Println("=================================")
	fmt.Println()

	if *continuous {
		runContinuousDiscovery()
	} else {
		runSingleDiscovery(*timeout, *createDevices)
	}
}

// runSingleDiscovery performs a one-time discovery scan.
func runSingleDiscovery(timeout time.Duration, createDevices bool) {
	fmt.Printf("Scanning for Shelly devices (timeout: %v)...\n\n", timeout)

	// Create mDNS discoverer
	mdns := discovery.NewMDNSDiscoverer()

	// Perform discovery
	devices, err := mdns.Discover(timeout)
	if err != nil {
		log.Fatalf("Discovery failed: %v", err)
	}

	if len(devices) == 0 {
		fmt.Println("No Shelly devices found.")
		fmt.Println()
		fmt.Println("Troubleshooting tips:")
		fmt.Println("  - Ensure devices are powered on and connected to your network")
		fmt.Println("  - Check that mDNS is not blocked by your firewall (UDP port 5353)")
		fmt.Println("  - Try increasing the timeout with -timeout flag")
		fmt.Println("  - Gen1 devices may not advertise via mDNS; use CoIoT discovery")
		return
	}

	fmt.Printf("Found %d device(s):\n\n", len(devices))

	for i := range devices {
		printDevice(i+1, &devices[i])
	}

	// Create device instances if requested
	if createDevices {
		fmt.Println("\nCreating device instances...")
		createDeviceInstances(devices)
	}

	fmt.Println("\nDiscovery complete!")
}

// runContinuousDiscovery performs continuous discovery until interrupted.
func runContinuousDiscovery() {
	fmt.Println("Starting continuous discovery (Ctrl+C to stop)...")
	fmt.Println()

	// Create mDNS discoverer
	mdns := discovery.NewMDNSDiscoverer()

	// Start continuous discovery
	deviceCh, err := mdns.StartDiscovery()
	if err != nil {
		log.Fatalf("Failed to start continuous discovery: %v", err)
	}

	// Set up signal handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	seenDevices := make(map[string]bool)

	fmt.Println("Listening for Shelly device advertisements...")
	fmt.Println()

	for {
		select {
		case device := <-deviceCh:
			// Only print new devices
			if !seenDevices[device.ID] {
				seenDevices[device.ID] = true
				fmt.Printf("[%s] New device found:\n", time.Now().Format("15:04:05"))
				printDevice(len(seenDevices), &device)
			}
			// Update existing device (comment out to reduce output)
			// fmt.Printf("[%s] Device %s still present\n", time.Now().Format("15:04:05"), device.ID)

		case <-sigCh:
			fmt.Println("\nStopping discovery...")
			if err := mdns.StopDiscovery(); err != nil {
				fmt.Printf("Warning: error stopping discovery: %v\n", err)
			}
			fmt.Printf("Total unique devices found: %d\n", len(seenDevices))
			return
		}
	}
}

// printDevice prints device information in a readable format.
func printDevice(index int, d *discovery.DiscoveredDevice) {
	fmt.Printf("%d. %s\n", index, deviceName(d))
	fmt.Printf("   ID: %s\n", d.ID)
	fmt.Printf("   Address: %s:%d\n", d.Address, d.Port)
	fmt.Printf("   Generation: %s\n", generationString(d.Generation))
	fmt.Printf("   Model: %s\n", d.Model)
	if d.Firmware != "" {
		fmt.Printf("   Firmware: %s\n", d.Firmware)
	}
	if d.MACAddress != "" {
		fmt.Printf("   MAC: %s\n", d.MACAddress)
	}
	if d.AuthRequired {
		fmt.Printf("   Auth Required: yes\n")
	}
	fmt.Printf("   Protocol: %s\n", d.Protocol)
	fmt.Printf("   URL: %s\n", d.URL())
	fmt.Println()
}

// deviceName returns a human-readable name for the device.
func deviceName(d *discovery.DiscoveredDevice) string {
	if d.Model != "" {
		return d.Model
	}
	if d.ID != "" {
		return "Shelly " + d.ID
	}
	return "Unknown Shelly Device"
}

// generationString returns a human-readable generation string.
func generationString(gen types.Generation) string {
	switch gen {
	case types.Gen1:
		return "Gen1"
	case types.Gen2:
		return "Gen2 (Plus)"
	case types.Gen3:
		return "Gen3"
	case types.Gen4:
		return "Gen4"
	default:
		return "Unknown"
	}
}

// createDeviceInstances creates device instances from discovery results.
func createDeviceInstances(devices []discovery.DiscoveredDevice) {
	for i := range devices {
		d := &devices[i]
		fmt.Printf("\nCreating device for %s (%s)...\n", d.ID, d.Address)

		// Create device from discovery result
		dev, err := factory.FromDiscovery(d)
		if err != nil {
			fmt.Printf("   Error: %v\n", err)
			continue
		}

		fmt.Printf("   Success! Device at %s (Gen%d)\n", dev.Address(), generationNumber(dev.Generation()))

		// For Gen2+ devices, we could get more info
		if dev.Generation() >= types.Gen2 {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			gen2Dev, ok := dev.(*factory.Gen2Device)
			if ok {
				info, err := gen2Dev.Shelly().GetDeviceInfo(ctx)
				if err == nil {
					fmt.Printf("   Device Name: %s\n", info.Name)
					fmt.Printf("   App: %s\n", info.App)
				}
			}
			cancel() // Cancel context at end of loop iteration, not function exit
		}
	}
}

// generationNumber returns the numeric generation value.
func generationNumber(gen types.Generation) int {
	switch gen {
	case types.Gen1:
		return 1
	case types.Gen2:
		return 2
	case types.Gen3:
		return 3
	case types.Gen4:
		return 4
	default:
		return 0
	}
}
