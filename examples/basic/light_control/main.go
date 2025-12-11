// Example: light_control demonstrates controlling a Shelly Gen2+ light/dimmer.
//
// This example shows how to:
//   - Connect to a Shelly light/dimmer device via HTTP
//   - Turn the light on/off
//   - Set brightness level
//   - Use transition/fade effects
//   - Toggle the light
//   - Get light status
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
	lightID := flag.Int("id", 0, "Light component ID (default: 0)")
	demoEffects := flag.Bool("demo", false, "Run demo with brightness effects")
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

	// Create light component
	light := components.NewLight(client, *lightID)

	// Get initial status
	fmt.Printf("--- Light %d Status ---\n", *lightID)
	status, err := light.GetStatus(ctx)
	if err != nil {
		log.Printf("Failed to get light status: %v", err)
		return
	}
	printLightStatus(status)

	// Get configuration
	fmt.Println("\n--- Light Configuration ---")
	config, err := light.GetConfig(ctx)
	if err != nil {
		log.Printf("Failed to get light config: %v", err)
		return
	}
	printLightConfig(config)

	// Run demo effects if requested
	if *demoEffects {
		fmt.Println("\n--- Demonstrating Light Operations ---")
		demoOperations(ctx, light)
	} else {
		fmt.Println("\n--- Basic Light Operations ---")
		basicOperations(ctx, light)
	}

	fmt.Println("\nLight control example completed!")
}

// basicOperations demonstrates simple on/off/toggle operations.
func basicOperations(ctx context.Context, light *components.Light) {
	// Toggle the light
	fmt.Println("Toggling light...")
	toggleResult, err := light.Toggle(ctx)
	if err != nil {
		log.Fatalf("Failed to toggle light: %v", err)
	}
	if toggleResult.WasOn != nil {
		fmt.Printf("Toggle complete. Previous state was: %s\n", boolToOnOff(*toggleResult.WasOn))
	}

	// Wait a moment
	time.Sleep(500 * time.Millisecond)

	// Get status after toggle
	status, err := light.GetStatus(ctx)
	if err != nil {
		log.Fatalf("Failed to get light status: %v", err)
	}
	fmt.Printf("Current state: %s", boolToOnOff(status.Output))
	if status.Brightness != nil {
		fmt.Printf(" at %d%% brightness", *status.Brightness)
	}
	fmt.Println()

	fmt.Println("\nTip: Run with -demo flag to see brightness effects")
}

// demoOperations demonstrates brightness and transition effects.
func demoOperations(ctx context.Context, light *components.Light) {
	// Get initial status
	initialStatus, err := light.GetStatus(ctx)
	if err != nil {
		log.Printf("Failed to get initial status: %v", err)
		return
	}

	initialOn := initialStatus.Output
	initialBrightness := 50
	if initialStatus.Brightness != nil {
		initialBrightness = *initialStatus.Brightness
	}

	fmt.Printf("Initial state: %s", boolToOnOff(initialOn))
	if initialStatus.Brightness != nil {
		fmt.Printf(" at %d%% brightness", *initialStatus.Brightness)
	}
	fmt.Println()

	// Turn on at 100% brightness
	fmt.Println("\n1. Turning on at 100% brightness...")
	on := true
	brightness := 100
	_, err = light.Set(ctx, &components.LightSetParams{
		On:         &on,
		Brightness: &brightness,
	})
	if err != nil {
		log.Printf("Failed to set light: %v", err)
		return
	}
	time.Sleep(time.Second)

	// Dim to 25% with fade effect
	fmt.Println("\n2. Dimming to 25% with 2-second fade...")
	brightness = 25
	transition := 2000 // 2 seconds in milliseconds
	_, err = light.Set(ctx, &components.LightSetParams{
		On:                 &on,
		Brightness:         &brightness,
		TransitionDuration: &transition,
	})
	if err != nil {
		log.Printf("Failed to set light: %v", err)
		return
	}

	// Wait for transition
	time.Sleep(3 * time.Second)

	// Check status
	status, err := light.GetStatus(ctx)
	if err == nil {
		fmt.Printf("   Current brightness: %d%%\n", *status.Brightness)
	}

	// Increase to 75% with fade
	fmt.Println("\n3. Increasing to 75% with 1-second fade...")
	brightness = 75
	transition = 1000
	_, err = light.Set(ctx, &components.LightSetParams{
		On:                 &on,
		Brightness:         &brightness,
		TransitionDuration: &transition,
	})
	if err != nil {
		log.Printf("Failed to set light: %v", err)
		return
	}

	// Wait for transition
	time.Sleep(2 * time.Second)

	// Check status
	status, err = light.GetStatus(ctx)
	if err == nil {
		fmt.Printf("   Current brightness: %d%%\n", *status.Brightness)
	}

	// Turn on with timer (toggle after 5 seconds)
	fmt.Println("\n4. Setting 5-second timer...")
	toggleAfter := 5.0
	brightness = 50
	_, err = light.Set(ctx, &components.LightSetParams{
		On:          &on,
		Brightness:  &brightness,
		ToggleAfter: &toggleAfter,
	})
	if err != nil {
		log.Printf("Failed to set light with timer: %v", err)
		return
	}
	fmt.Println("   Light will toggle in 5 seconds")

	// Show timer status
	status, err = light.GetStatus(ctx)
	if err == nil && status.TimerDuration != nil {
		fmt.Printf("   Timer duration: %.1f seconds\n", *status.TimerDuration)
	}

	// Wait a bit
	time.Sleep(2 * time.Second)

	// Restore initial state
	fmt.Println("\n5. Restoring initial state...")
	_, err = light.Set(ctx, &components.LightSetParams{
		On:         &initialOn,
		Brightness: &initialBrightness,
	})
	if err != nil {
		log.Printf("Failed to restore initial state: %v", err)
	}

	// Final status
	fmt.Println("\n6. Final status:")
	status, err = light.GetStatus(ctx)
	if err == nil {
		printLightStatus(status)
	}
}

// printLightStatus prints the light status in a readable format.
func printLightStatus(status *components.LightStatus) {
	fmt.Printf("Output: %s\n", boolToOnOff(status.Output))
	if status.Brightness != nil {
		fmt.Printf("Brightness: %d%%\n", *status.Brightness)
	}
	fmt.Printf("Source: %s\n", status.Source)

	if status.TransitionDuration != nil {
		fmt.Printf("Transition Duration: %d ms\n", *status.TransitionDuration)
	}
	if status.TimerDuration != nil && *status.TimerDuration > 0 {
		fmt.Printf("Timer Duration: %.1f s\n", *status.TimerDuration)
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
	if status.Temperature != nil && status.Temperature.TC != nil {
		fmt.Printf("Temperature: %.1f C\n", *status.Temperature.TC)
	}
	if len(status.Errors) > 0 {
		fmt.Printf("Errors: %v\n", status.Errors)
	}
}

// printLightConfig prints the light configuration in a readable format.
func printLightConfig(config *components.LightConfig) {
	if config.Name != nil {
		fmt.Printf("Name: %s\n", *config.Name)
	}
	if config.InitialState != nil {
		fmt.Printf("Initial State: %s\n", *config.InitialState)
	}
	if config.DefaultBrightness != nil {
		fmt.Printf("Default Brightness: %d%%\n", *config.DefaultBrightness)
	}
	if config.TransitionDuration != nil {
		fmt.Printf("Transition Duration: %d ms\n", *config.TransitionDuration)
	}
	if config.MinBrightnessOnToggle != nil {
		fmt.Printf("Min Brightness on Toggle: %d%%\n", *config.MinBrightnessOnToggle)
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
	if config.NightMode != nil {
		fmt.Println("Night Mode:")
		if config.NightMode.Enable != nil {
			fmt.Printf("  Enabled: %v\n", *config.NightMode.Enable)
		}
		if config.NightMode.Brightness != nil {
			fmt.Printf("  Brightness: %d%%\n", *config.NightMode.Brightness)
		}
		if len(config.NightMode.ActiveBetween) >= 2 {
			fmt.Printf("  Active Between: %s - %s\n",
				config.NightMode.ActiveBetween[0], config.NightMode.ActiveBetween[1])
		}
	}
}

// boolToOnOff converts a boolean to "on" or "off" string.
func boolToOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
