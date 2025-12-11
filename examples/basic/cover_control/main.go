// Example: cover_control demonstrates controlling a Shelly Gen2+ cover (roller shutter).
//
// This example shows how to:
//   - Connect to a Shelly device via HTTP
//   - Open and close the cover
//   - Move to a specific position
//   - Stop movement
//   - Calibrate the cover
//   - Get cover status including position
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
	coverID := flag.Int("id", 0, "Cover component ID (default: 0)")
	demoMove := flag.Bool("demo", false, "Run demo movements (open/close/position)")
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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

	// Create cover component
	cover := components.NewCover(client, *coverID)

	// Get initial status
	fmt.Printf("--- Cover %d Status ---\n", *coverID)
	status, err := cover.GetStatus(ctx)
	if err != nil {
		log.Printf("Failed to get cover status: %v", err)
		return
	}
	printCoverStatus(status)

	// Get configuration
	fmt.Println("\n--- Cover Configuration ---")
	config, err := cover.GetConfig(ctx)
	if err != nil {
		log.Printf("Failed to get cover config: %v", err)
		return
	}
	printCoverConfig(config)

	// Run demo movements if requested
	if *demoMove {
		fmt.Println("\n--- Demonstrating Cover Operations ---")
		demoOperations(ctx, cover)
	} else {
		fmt.Println("\nTip: Run with -demo flag to demonstrate cover movements")
	}

	fmt.Println("\nCover control example completed!")
}

// demoOperations demonstrates cover operations with actual movements.
func demoOperations(ctx context.Context, cover *components.Cover) {
	// Get initial status to know starting position
	status, err := cover.GetStatus(ctx)
	if err != nil {
		log.Printf("Failed to get initial status: %v", err)
		return
	}

	initialPos := 0
	if status.CurrentPos != nil {
		initialPos = *status.CurrentPos
	}
	fmt.Printf("Starting position: %d%%\n", initialPos)

	// Open the cover (move to fully open)
	fmt.Println("\n1. Opening cover...")
	err = cover.Open(ctx, nil)
	if err != nil {
		log.Printf("Failed to open cover: %v", err)
		return
	}
	fmt.Println("   Open command sent")

	// Wait for movement to start
	time.Sleep(2 * time.Second)

	// Stop the cover
	fmt.Println("\n2. Stopping cover...")
	err = cover.Stop(ctx)
	if err != nil {
		log.Printf("Failed to stop cover: %v", err)
		return
	}
	fmt.Println("   Stop command sent")

	// Check position after stop
	time.Sleep(500 * time.Millisecond)
	status, err = cover.GetStatus(ctx)
	if err == nil && status.CurrentPos != nil {
		fmt.Printf("   Current position: %d%%\n", *status.CurrentPos)
	}

	// Move to 50% position
	fmt.Println("\n3. Moving to 50% position...")
	err = cover.GoToPosition(ctx, 50)
	if err != nil {
		log.Printf("Failed to move to position: %v", err)
		return
	}
	fmt.Println("   GoToPosition(50) command sent")

	// Wait for movement
	fmt.Println("   Waiting for movement...")
	time.Sleep(3 * time.Second)

	// Check position
	status, err = cover.GetStatus(ctx)
	if err == nil {
		printCoverStatus(status)
	}

	// Close for 2 seconds
	fmt.Println("\n4. Closing for 2 seconds...")
	duration := 2.0
	err = cover.Close(ctx, &duration)
	if err != nil {
		log.Printf("Failed to close cover: %v", err)
		return
	}
	fmt.Println("   Close with duration command sent")

	// Wait for movement to complete
	time.Sleep(3 * time.Second)

	// Final status
	fmt.Println("\n5. Final status:")
	status, err = cover.GetStatus(ctx)
	if err == nil {
		printCoverStatus(status)
	}

	// Return to initial position
	fmt.Printf("\n6. Returning to initial position (%d%%)...\n", initialPos)
	err = cover.GoToPosition(ctx, initialPos)
	if err != nil {
		log.Printf("Failed to return to initial position: %v", err)
		return
	}
	fmt.Println("   GoToPosition command sent")
}

// printCoverStatus prints the cover status in a readable format.
func printCoverStatus(status *components.CoverStatus) {
	fmt.Printf("State: %s\n", status.State)
	fmt.Printf("Source: %s\n", status.Source)

	if status.CurrentPos != nil {
		fmt.Printf("Current Position: %d%%\n", *status.CurrentPos)
	} else {
		fmt.Println("Current Position: unknown (not calibrated?)")
	}
	if status.TargetPos != nil {
		fmt.Printf("Target Position: %d%%\n", *status.TargetPos)
	}
	if status.LastDirection != nil {
		fmt.Printf("Last Direction: %s\n", *status.LastDirection)
	}
	if status.MoveTimeout != nil && *status.MoveTimeout {
		fmt.Println("Move Timeout: yes (movement may be obstructed)")
	}

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

// printCoverConfig prints the cover configuration in a readable format.
func printCoverConfig(config *components.CoverConfig) {
	if config.Name != nil {
		fmt.Printf("Name: %s\n", *config.Name)
	}
	if config.InitialState != nil {
		fmt.Printf("Initial State: %s\n", *config.InitialState)
	}
	if config.SwapInputs != nil {
		fmt.Printf("Swap Inputs: %v\n", *config.SwapInputs)
	}
	if config.InvertDirections != nil {
		fmt.Printf("Invert Directions: %v\n", *config.InvertDirections)
	}
	if config.MotorMoveTimeout != nil {
		fmt.Printf("Motor Move Timeout: %.1f s\n", *config.MotorMoveTimeout)
	}
	if config.ObstructionDetectionLevel != nil {
		fmt.Printf("Obstruction Detection: %d%%\n", *config.ObstructionDetectionLevel)
	}
	if config.PowerLimit != nil {
		fmt.Printf("Power Limit: %.0f W\n", *config.PowerLimit)
	}
}
