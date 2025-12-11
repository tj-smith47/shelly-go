// Example: switch_control demonstrates controlling a Shelly Gen2+ switch.
//
// This example shows how to:
//   - Connect to a Shelly device via HTTP
//   - Turn a switch on/off
//   - Toggle the switch state
//   - Get switch status including power consumption
//   - Reset energy counters
//
// Usage:
//
//	go run main.go -host 192.168.1.100 [-user admin] [-pass password]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/gen2/components"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func main() {
	// Parse command line flags
	host := flag.String("host", "", "Device IP address or hostname (required)")
	user := flag.String("user", "", "Username for authentication (optional)")
	pass := flag.String("pass", "", "Password for authentication (optional)")
	switchID := flag.Int("id", 0, "Switch component ID (default: 0)")
	flag.Parse()

	// Check for required host flag or environment variable
	if *host == "" {
		*host = os.Getenv("SHELLY_HOST")
	}
	if *host == "" {
		fmt.Println("Error: -host flag or SHELLY_HOST environment variable is required")
		flag.Usage()
		os.Exit(1)
	}

	// Check for auth from environment if not provided via flags
	if *user == "" {
		*user = os.Getenv("SHELLY_USER")
	}
	if *pass == "" {
		*pass = os.Getenv("SHELLY_PASS")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create HTTP transport
	transportURL := "http://" + *host
	var transportOpts []transport.Option
	transportOpts = append(transportOpts, transport.WithTimeout(10*time.Second))

	if *user != "" && *pass != "" {
		transportOpts = append(transportOpts, transport.WithAuth(*user, *pass))
	}

	httpTransport := transport.NewHTTP(transportURL, transportOpts...)

	// Create RPC client
	var client *rpc.Client
	if *user != "" && *pass != "" {
		auth := &rpc.AuthData{
			Realm:    "shelly",
			Username: *user,
			Password: *pass,
		}
		client = rpc.NewClientWithAuth(httpTransport, auth)
	} else {
		client = rpc.NewClient(httpTransport)
	}
	defer client.Close()

	// Create Gen2 device
	device := gen2.NewDevice(client)

	// Get device info first
	fmt.Println("Connecting to device...")
	info, err := device.Shelly().GetDeviceInfo(ctx)
	if err != nil {
		log.Printf("Failed to get device info: %v", err)
		return
	}
	fmt.Printf("Connected to: %s (%s)\n", info.Name, info.Model)
	fmt.Printf("Firmware: %s\n", info.FirmwareID)
	fmt.Println()

	// Create switch component
	sw := components.NewSwitch(client, *switchID)

	// Get initial status
	fmt.Printf("--- Switch %d Status ---\n", *switchID)
	status, err := sw.GetStatus(ctx)
	if err != nil {
		log.Printf("Failed to get switch status: %v", err)
		return
	}
	printSwitchStatus(status)

	// Get configuration
	fmt.Println("\n--- Switch Configuration ---")
	config, err := sw.GetConfig(ctx)
	if err != nil {
		log.Printf("Failed to get switch config: %v", err)
		return
	}
	printSwitchConfig(config)

	// Demonstrate switch operations
	fmt.Println("\n--- Demonstrating Switch Operations ---")

	// Toggle the switch
	fmt.Println("Toggling switch...")
	toggleResult, err := sw.Toggle(ctx)
	if err != nil {
		log.Printf("Failed to toggle switch: %v", err)
		return
	}
	fmt.Printf("Toggle complete. Previous state was: %s\n", boolToOnOff(toggleResult.WasOn))

	// Wait a moment
	time.Sleep(500 * time.Millisecond)

	// Get status after toggle
	status, err = sw.GetStatus(ctx)
	if err != nil {
		log.Printf("Failed to get switch status: %v", err)
		return
	}
	fmt.Printf("Current output: %s\n", boolToOnOff(status.Output))

	// Set switch to specific state (turn off)
	fmt.Println("\nTurning switch off...")
	setResult, err := sw.Set(ctx, &components.SwitchSetParams{On: false})
	if err != nil {
		log.Printf("Failed to set switch: %v", err)
		return
	}
	fmt.Printf("Set complete. Previous state was: %s\n", boolToOnOff(setResult.WasOn))

	// Wait a moment
	time.Sleep(500 * time.Millisecond)

	// Turn on with timer (toggle after 5 seconds)
	fmt.Println("\nTurning switch on with 5-second timer...")
	toggleAfter := 5.0
	_, err = sw.Set(ctx, &components.SwitchSetParams{On: true, ToggleAfter: &toggleAfter})
	if err != nil {
		log.Printf("Failed to set switch with timer: %v", err)
		return
	}
	fmt.Println("Switch is now on and will toggle off in 5 seconds")

	// Get status showing timer
	status, err = sw.GetStatus(ctx)
	if err != nil {
		log.Printf("Failed to get switch status: %v", err)
		return
	}
	fmt.Printf("Current output: %s\n", boolToOnOff(status.Output))
	if status.TimerDuration != nil && *status.TimerDuration > 0 {
		fmt.Printf("Timer duration: %.1f seconds\n", *status.TimerDuration)
	}

	// Show power consumption if available
	if status.APower != nil {
		fmt.Printf("\n--- Power Measurements ---\n")
		fmt.Printf("Active Power: %.2f W\n", *status.APower)
		if status.Voltage != nil {
			fmt.Printf("Voltage: %.2f V\n", *status.Voltage)
		}
		if status.Current != nil {
			fmt.Printf("Current: %.3f A\n", *status.Current)
		}
		if status.AEnergy != nil {
			fmt.Printf("Total Energy: %.2f Wh\n", status.AEnergy.Total)
		}
	}

	// Reset energy counters example (commented out to avoid data loss)
	// fmt.Println("\nResetting energy counters...")
	// err = sw.ResetCounters(ctx, []string{"aenergy"})
	// if err != nil {
	//     log.Fatalf("Failed to reset counters: %v", err)
	// }
	// fmt.Println("Energy counters reset")

	fmt.Println("\nSwitch control example completed!")
}

// printSwitchStatus prints the switch status in a readable format.
func printSwitchStatus(status *components.SwitchStatus) {
	fmt.Printf("Output: %s\n", boolToOnOff(status.Output))
	fmt.Printf("Source: %s\n", status.Source)

	if status.APower != nil {
		fmt.Printf("Power: %.2f W\n", *status.APower)
	}
	if status.Voltage != nil {
		fmt.Printf("Voltage: %.2f V\n", *status.Voltage)
	}
	if status.Current != nil {
		fmt.Printf("Current: %.3f A\n", *status.Current)
	}
	if status.AEnergy != nil {
		fmt.Printf("Energy: %.2f Wh\n", status.AEnergy.Total)
	}
	if status.Temperature != nil && status.Temperature.TC != nil {
		fmt.Printf("Temperature: %.1f C\n", *status.Temperature.TC)
	}
	if len(status.Errors) > 0 {
		fmt.Printf("Errors: %v\n", status.Errors)
	}
}

// printSwitchConfig prints the switch configuration in a readable format.
func printSwitchConfig(config *components.SwitchConfig) {
	if config.Name != nil {
		fmt.Printf("Name: %s\n", *config.Name)
	}
	if config.InitialState != nil {
		fmt.Printf("Initial State: %s\n", *config.InitialState)
	}
	if config.AutoOff != nil {
		fmt.Printf("Auto Off: %v\n", *config.AutoOff)
		if config.AutoOffDelay != nil {
			fmt.Printf("Auto Off Delay: %.1f s\n", *config.AutoOffDelay)
		}
	}
	if config.AutoOn != nil {
		fmt.Printf("Auto On: %v\n", *config.AutoOn)
		if config.AutoOnDelay != nil {
			fmt.Printf("Auto On Delay: %.1f s\n", *config.AutoOnDelay)
		}
	}
	if config.PowerLimit != nil {
		fmt.Printf("Power Limit: %.0f W\n", *config.PowerLimit)
	}
}

// boolToOnOff converts a boolean to "on" or "off" string.
func boolToOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
