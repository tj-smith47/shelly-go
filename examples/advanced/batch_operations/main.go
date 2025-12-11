// Example: batch_operations demonstrates batch operations across multiple Shelly devices.
//
// This example shows how to:
//   - Discover devices on the network
//   - Create device instances from addresses
//   - Perform batch operations (all on, all off, toggle)
//   - Use device groups for organized control
//   - Handle batch operation results
//
// Usage:
//
//	go run main.go -addresses 192.168.1.100,192.168.1.101,192.168.1.102
//	go run main.go -discover  # Use mDNS discovery
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/factory"
	"github.com/tj-smith47/shelly-go/helpers"
)

func main() {
	// Parse command line flags
	addresses := flag.String("addresses", "", "Comma-separated list of device IP addresses")
	useDiscovery := flag.Bool("discover", false, "Use mDNS discovery to find devices")
	user := flag.String("user", "", "Username for authentication (optional)")
	pass := flag.String("pass", "", "Password for authentication (optional)")
	demoOperations := flag.Bool("demo", false, "Run demonstration of batch operations")
	flag.Parse()

	// Check for credentials from environment
	if *user == "" {
		*user = os.Getenv("SHELLY_USER")
	}
	if *pass == "" {
		*pass = os.Getenv("SHELLY_PASS")
	}

	// Need either addresses or discovery
	if *addresses == "" && !*useDiscovery {
		fmt.Println("Error: Either -addresses or -discover is required")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  go run main.go -addresses 192.168.1.100,192.168.1.101")
		fmt.Println("  go run main.go -discover")
		os.Exit(1)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("Shelly Batch Operations Example")
	fmt.Println("================================")
	fmt.Println()

	// Get devices
	var devices []factory.Device

	if *useDiscovery {
		var err error
		devices, err = discoverDevices(ctx, *user, *pass)
		if err != nil {
			log.Printf("Failed to get devices: %v", err)
			return
		}
	} else {
		devices = devicesFromAddresses(*addresses, *user, *pass)
	}

	if len(devices) == 0 {
		fmt.Println("No devices found.")
		return
	}

	fmt.Printf("Found %d device(s)\n\n", len(devices))

	// Show devices
	for i, dev := range devices {
		fmt.Printf("%d. %s (Gen%d)\n", i+1, dev.Address(), genNumber(dev.Generation()))
	}
	fmt.Println()

	// Demonstrate batch operations
	if *demoOperations && len(devices) > 0 {
		demonstrateBatchOperations(ctx, devices)
	} else {
		showBatchOperationExamples(ctx, devices)
	}

	fmt.Println("\nBatch operations example completed!")
}

// discoverDevices discovers devices on the network using mDNS.
func discoverDevices(_ context.Context, user, pass string) ([]factory.Device, error) {
	fmt.Println("Discovering devices via mDNS (5 second scan)...")
	fmt.Println()

	mdns := discovery.NewMDNSDiscoverer()
	discovered, err := mdns.Discover(5 * time.Second)
	if err != nil {
		return nil, err
	}

	if len(discovered) == 0 {
		return nil, nil
	}

	// Create factory options
	var opts []factory.Option
	if user != "" && pass != "" {
		opts = append(opts, factory.WithAuth(user, pass))
	}
	opts = append(opts, factory.WithTimeout(10*time.Second))

	// Create devices from discovery results
	devices, errs := factory.BatchFromDiscovery(discovered, opts...)

	// Log any errors
	for i, err := range errs {
		if err != nil {
			fmt.Printf("Warning: Failed to create device for %s: %v\n", discovered[i].ID, err)
		}
	}

	// Filter out nil devices
	var validDevices []factory.Device
	for _, dev := range devices {
		if dev != nil {
			validDevices = append(validDevices, dev)
		}
	}

	return validDevices, nil
}

// devicesFromAddresses creates devices from a comma-separated list of addresses.
func devicesFromAddresses(addresses, user, pass string) []factory.Device {
	addrs := strings.Split(addresses, ",")
	for i := range addrs {
		addrs[i] = strings.TrimSpace(addrs[i])
	}

	fmt.Printf("Creating devices from %d address(es)...\n", len(addrs))
	fmt.Println()

	// Create factory options
	var opts []factory.Option
	if user != "" && pass != "" {
		opts = append(opts, factory.WithAuth(user, pass))
	}
	opts = append(opts, factory.WithTimeout(10*time.Second))

	// Create devices concurrently
	devices, errs := factory.BatchFromAddresses(addrs, opts...)

	// Log any errors
	for i, err := range errs {
		if err != nil {
			fmt.Printf("Warning: Failed to create device for %s: %v\n", addrs[i], err)
		}
	}

	// Filter out nil devices
	var validDevices []factory.Device
	for _, dev := range devices {
		if dev != nil {
			validDevices = append(validDevices, dev)
		}
	}

	return validDevices
}

// demonstrateBatchOperations demonstrates batch operations on devices.
func demonstrateBatchOperations(ctx context.Context, devices []factory.Device) {
	fmt.Println("--- Demonstrating Batch Operations ---")
	fmt.Println()

	// 1. All Off
	fmt.Println("1. Turning all devices OFF...")
	results := helpers.AllOff(ctx, devices)
	printBatchResults("AllOff", results)

	time.Sleep(time.Second)

	// 2. All On
	fmt.Println("\n2. Turning all devices ON...")
	results = helpers.AllOn(ctx, devices)
	printBatchResults("AllOn", results)

	time.Sleep(time.Second)

	// 3. Toggle
	fmt.Println("\n3. Toggling all devices...")
	results = helpers.BatchToggle(ctx, devices)
	printBatchResults("BatchToggle", results)

	time.Sleep(time.Second)

	// 4. All Off again to restore
	fmt.Println("\n4. Turning all devices OFF (restore)...")
	results = helpers.AllOff(ctx, devices)
	printBatchResults("AllOff", results)
}

// showBatchOperationExamples shows examples of batch operations without executing.
func showBatchOperationExamples(_ context.Context, devices []factory.Device) {
	fmt.Println("--- Batch Operation Examples ---")
	fmt.Println()

	fmt.Println("Available batch operations:")
	fmt.Println()

	fmt.Println("1. Turn all devices on:")
	fmt.Println("   results := helpers.AllOn(ctx, devices)")
	fmt.Println()

	fmt.Println("2. Turn all devices off:")
	fmt.Println("   results := helpers.AllOff(ctx, devices)")
	fmt.Println()

	fmt.Println("3. Toggle all devices:")
	fmt.Println("   results := helpers.BatchToggle(ctx, devices)")
	fmt.Println()

	fmt.Println("4. Set specific state:")
	fmt.Println("   results := helpers.BatchSet(ctx, devices, true)  // on")
	fmt.Println("   results := helpers.BatchSet(ctx, devices, false) // off")
	fmt.Println()

	fmt.Println("5. Set brightness (for lights):")
	fmt.Println("   results := helpers.BatchSetBrightness(ctx, devices, 75)")
	fmt.Println()

	// Demonstrate device groups
	fmt.Println("--- Device Groups ---")
	fmt.Println()

	// Create groups
	if len(devices) >= 2 {
		group1 := helpers.NewGroup("Group A",
			helpers.WithDevice(devices[0]),
		)

		group2 := helpers.NewGroup("Group B",
			helpers.WithDevice(devices[len(devices)-1]),
		)

		fmt.Printf("Created '%s' with %d device(s)\n", group1.Name(), group1.Len())
		fmt.Printf("Created '%s' with %d device(s)\n", group2.Name(), group2.Len())
		fmt.Println()

		fmt.Println("Group operations:")
		fmt.Println("   group.AllOn(ctx)")
		fmt.Println("   group.AllOff(ctx)")
		fmt.Println("   group.Toggle(ctx)")
		fmt.Println("   group.Set(ctx, true)")
		fmt.Println("   group.SetBrightness(ctx, 50)")
		fmt.Println()

		fmt.Println("Group management:")
		fmt.Println("   group.Add(device)")
		fmt.Println("   group.Remove(address)")
		fmt.Println("   group.Contains(address)")
		fmt.Println("   group.Devices()")
		fmt.Println("   group.Len()")
	} else {
		fmt.Println("(Need 2+ devices to demonstrate groups)")
	}

	fmt.Println()
	fmt.Println("--- Result Handling ---")
	fmt.Println()

	fmt.Println("Checking results:")
	fmt.Println("   if results.AllSuccessful() {")
	fmt.Println("       fmt.Println(\"All operations succeeded\")")
	fmt.Println("   }")
	fmt.Println()
	fmt.Println("   for _, failure := range results.Failures() {")
	printfExample := `       fmt.Printf("Failed: %s - %v\n", failure.Device.Address(), failure.Error)`
	fmt.Println(printfExample)
	fmt.Println("   }")
	fmt.Println()

	fmt.Println("Run with -demo flag to execute actual batch operations.")
}

// printBatchResults prints the results of a batch operation.
func printBatchResults(operation string, results helpers.BatchResults) {
	if results.AllSuccessful() {
		fmt.Printf("   %s: All %d operations succeeded\n", operation, len(results))
	} else {
		successes := results.Successes()
		failures := results.Failures()
		fmt.Printf("   %s: %d succeeded, %d failed\n", operation, len(successes), len(failures))
		for _, failure := range failures {
			fmt.Printf("     - %s: %v\n", failure.Device.Address(), failure.Error)
		}
	}
}

// genNumber converts Generation to a number for display.
func genNumber(gen any) int {
	// Handle both types.Generation and factory.Device.Generation()
	switch g := gen.(type) {
	case int:
		return g
	default:
		// Assume it's types.Generation which is an int underneath
		return 0
	}
}
