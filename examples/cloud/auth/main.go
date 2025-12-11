// Example: cloud/auth demonstrates authenticating with the Shelly Cloud API.
//
// This example shows how to:
//   - Authenticate using email and password
//   - Parse and validate JWT tokens
//   - List all devices in your account
//   - Get device status
//   - Control devices remotely via Cloud API
//
// Usage:
//
//	go run main.go -email your@email.com -password yourpassword
//
// Or using environment variables:
//
//	export SHELLY_CLOUD_EMAIL=your@email.com
//	export SHELLY_CLOUD_PASSWORD=yourpassword
//	go run main.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tj-smith47/shelly-go/cloud"
)

func main() {
	// Parse command line flags
	email := flag.String("email", "", "Shelly Cloud email address")
	password := flag.String("password", "", "Shelly Cloud password")
	listDevices := flag.Bool("list", true, "List all devices")
	controlDemo := flag.Bool("demo", false, "Run control demonstration (toggles first switch)")
	flag.Parse()

	// Check for credentials from environment if not provided via flags
	if *email == "" {
		*email = os.Getenv("SHELLY_CLOUD_EMAIL")
	}
	if *password == "" {
		*password = os.Getenv("SHELLY_CLOUD_PASSWORD")
	}

	if *email == "" || *password == "" {
		fmt.Println("Error: Email and password are required")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  go run main.go -email your@email.com -password yourpassword")
		fmt.Println()
		fmt.Println("Or set environment variables:")
		fmt.Println("  export SHELLY_CLOUD_EMAIL=your@email.com")
		fmt.Println("  export SHELLY_CLOUD_PASSWORD=yourpassword")
		os.Exit(1)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Hash the password (Shelly Cloud API requires SHA1 hash)
	passwordHash := cloud.HashPassword(*password)

	fmt.Println("Shelly Cloud API Authentication Example")
	fmt.Println("=======================================")
	fmt.Println()

	// Method 1: Create client and authenticate
	fmt.Println("Authenticating with Shelly Cloud...")
	client, err := cloud.NewClientWithCredentials(ctx, *email, passwordHash)
	if err != nil {
		log.Printf("Authentication failed: %v", err)
		return
	}
	fmt.Println("Authentication successful!")
	fmt.Println()

	// Get token information
	token := client.GetToken()
	if token != "" {
		fmt.Println("Token Information:")
		fmt.Printf("  API URL: %s\n", client.GetBaseURL())

		// Parse token for expiry info
		expiry, tokenErr := cloud.TokenExpiry(token)
		if tokenErr == nil && !expiry.IsZero() {
			fmt.Printf("  Token Expires: %s\n", expiry.Format(time.RFC1123))
			fmt.Printf("  Time Until Expiry: %s\n", time.Until(expiry).Round(time.Second))
		}

		// Check if token needs refresh
		if cloud.ShouldRefresh(token, 5*time.Minute) {
			fmt.Println("  Status: Should refresh soon")
		} else {
			fmt.Println("  Status: Valid")
		}
	}
	fmt.Println()

	// Method 2: Using TokenSource for automatic refresh
	fmt.Println("Setting up automatic token refresh...")
	tokenSource := cloud.CredentialTokenSource(*email, passwordHash,
		cloud.WithRefreshThreshold(5*time.Minute),
	)

	// Get a token from the source
	newToken, err := tokenSource.Token()
	if err != nil {
		log.Printf("Failed to get token: %v", err)
		return
	}
	fmt.Printf("  Token source configured (expires: %s)\n", newToken.Expiry.Format(time.RFC1123))
	fmt.Println()

	// List devices if requested
	if *listDevices {
		listAllDevices(ctx, client)
	}

	// Run control demo if requested
	if *controlDemo {
		controlDemonstration(ctx, client)
	}

	fmt.Println("\nCloud authentication example completed!")
}

// listAllDevices lists all devices in the account.
func listAllDevices(ctx context.Context, client *cloud.Client) {
	fmt.Println("Fetching devices...")
	devices, err := client.GetAllDevices(ctx)
	if err != nil {
		log.Printf("Failed to get devices: %v", err)
		return
	}

	if len(devices) == 0 {
		fmt.Println("No devices found in your account.")
		fmt.Println()
		fmt.Println("Tips:")
		fmt.Println("  - Ensure devices are connected to the Shelly Cloud")
		fmt.Println("  - Check that Cloud is enabled in device settings")
		fmt.Println("  - Devices may take a few minutes to appear after initial setup")
		return
	}

	fmt.Printf("\nFound %d device(s):\n\n", len(devices))

	for id, status := range devices {
		printDeviceStatus(id, status)
	}
}

// printDeviceStatus prints device status in a readable format.
func printDeviceStatus(id string, status *cloud.DeviceStatus) {
	fmt.Printf("Device: %s\n", id)

	if status.DevInfo != nil {
		if status.DevInfo.Code != "" {
			fmt.Printf("  Type: %s\n", status.DevInfo.Code)
		}
		if status.DevInfo.Generation != 0 {
			fmt.Printf("  Generation: %d\n", status.DevInfo.Generation)
		}
	}

	// Online status
	if status.Online {
		fmt.Println("  Status: Online")
	} else {
		fmt.Println("  Status: Offline")
	}

	// Parse raw status for more details
	if len(status.Status) > 0 {
		var rawStatus map[string]any
		if err := json.Unmarshal(status.Status, &rawStatus); err == nil {
			printRawStatus(rawStatus)
		}
	}

	fmt.Println()
}

// printRawStatus prints parsed status data.
func printRawStatus(status map[string]any) {
	// Check for relays (Gen1)
	if relays, ok := status["relays"].([]any); ok && len(relays) > 0 {
		fmt.Println("  Relays:")
		for i, relay := range relays {
			r, ok := relay.(map[string]any)
			if !ok {
				continue
			}
			state := "off"
			if isOn, ok := r["ison"].(bool); ok && isOn {
				state = "on"
			}
			fmt.Printf("    [%d] %s", i, state)
			if power, ok := r["power"].(float64); ok {
				fmt.Printf(" (%.1fW)", power)
			}
			fmt.Println()
		}
	}

	// Check for switches (Gen2+)
	for key, value := range status {
		if v, ok := value.(map[string]any); ok {
			if output, exists := v["output"]; exists {
				state := "off"
				if on, ok := output.(bool); ok && on {
					state = "on"
				}
				fmt.Printf("  %s: %s", key, state)
				if apower, ok := v["apower"].(float64); ok {
					fmt.Printf(" (%.1fW)", apower)
				}
				fmt.Println()
			}
		}
	}

	// Check for rollers
	if rollers, ok := status["rollers"].([]any); ok && len(rollers) > 0 {
		fmt.Println("  Rollers:")
		for i, roller := range rollers {
			r, ok := roller.(map[string]any)
			if !ok {
				continue
			}
			state := "unknown"
			if s, ok := r["state"].(string); ok {
				state = s
			}
			fmt.Printf("    [%d] state: %s", i, state)
			if pos, ok := r["current_pos"].(float64); ok {
				fmt.Printf(", position: %.0f%%", pos)
			}
			fmt.Println()
		}
	}

	// Check for meters
	if meters, ok := status["meters"].([]any); ok && len(meters) > 0 {
		fmt.Println("  Power Meters:")
		for i, meter := range meters {
			if m, ok := meter.(map[string]any); ok {
				if power, ok := m["power"].(float64); ok {
					fmt.Printf("    [%d] %.1fW", i, power)
					if total, ok := m["total"].(float64); ok {
						fmt.Printf(", total: %.2f Wh", total)
					}
					fmt.Println()
				}
			}
		}
	}
}

// controlDemonstration demonstrates device control via Cloud API.
func controlDemonstration(ctx context.Context, client *cloud.Client) {
	fmt.Println("\n--- Control Demonstration ---")
	fmt.Println()

	// Get devices
	devices, err := client.GetAllDevices(ctx)
	if err != nil {
		log.Printf("Failed to get devices: %v", err)
		return
	}

	// Find an online switch device
	var switchID string
	for id, status := range devices {
		if !status.Online {
			continue
		}

		// Check if device has switches/relays
		if len(status.Status) > 0 {
			var rawStatus map[string]any
			if unmarshalErr := json.Unmarshal(status.Status, &rawStatus); unmarshalErr == nil {
				if _, hasRelays := rawStatus["relays"]; hasRelays {
					switchID = id
					break
				}
				// Check for Gen2+ switches
				for key := range rawStatus {
					if key == "switch:0" {
						switchID = id
						break
					}
				}
			}
		}
	}

	if switchID == "" {
		fmt.Println("No online switch devices found for demonstration.")
		return
	}

	fmt.Printf("Using device: %s\n\n", switchID)

	// Toggle the switch
	fmt.Println("1. Toggling switch...")
	err = client.ToggleSwitch(ctx, switchID, 0)
	if err != nil {
		log.Printf("   Failed to toggle: %v", err)
	} else {
		fmt.Println("   Toggle command sent successfully")
	}

	// Wait a moment
	time.Sleep(2 * time.Second)

	// Turn off
	fmt.Println("\n2. Turning switch off...")
	err = client.SetSwitch(ctx, switchID, 0, false)
	if err != nil {
		log.Printf("   Failed to turn off: %v", err)
	} else {
		fmt.Println("   Switch turned off")
	}

	// Wait a moment
	time.Sleep(time.Second)

	// Turn on with timer
	fmt.Println("\n3. Turning switch on with 5-second timer...")
	err = client.SetSwitchWithTimer(ctx, switchID, 0, true, 5)
	if err != nil {
		log.Printf("   Failed to set timer: %v", err)
	} else {
		fmt.Println("   Switch turned on, will turn off in 5 seconds")
	}

	// Get final status
	fmt.Println("\n4. Getting current status...")
	status, err := client.GetDeviceStatus(ctx, switchID)
	if err != nil {
		log.Printf("   Failed to get status: %v", err)
	} else if len(status.Status) > 0 {
		var rawStatus map[string]any
		if err := json.Unmarshal(status.Status, &rawStatus); err == nil {
			printRawStatus(rawStatus)
		}
	}
}
