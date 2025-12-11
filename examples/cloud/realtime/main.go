// Example: cloud/realtime demonstrates real-time events from Shelly Cloud WebSocket.
//
// This example shows how to:
//   - Connect to Shelly Cloud WebSocket for real-time events
//   - Receive device online/offline events
//   - Receive device status change events
//   - Handle Gen2+ notification events
//
// Note: This example requires a WebSocket library like gorilla/websocket.
// The shelly-go cloud package provides interfaces for WebSocket connections
// that can be implemented with any WebSocket library.
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
	"os/signal"
	"syscall"
	"time"

	"github.com/tj-smith47/shelly-go/cloud"
)

// State string constants for display.
const (
	stateOn  = "on"
	stateOff = "off"
)

// NOTE: For a real implementation, you would use a WebSocket library like:
//   go get github.com/gorilla/websocket
//
// This example demonstrates the API and event handling patterns.

func main() {
	// Parse command line flags
	email := flag.String("email", "", "Shelly Cloud email address")
	password := flag.String("password", "", "Shelly Cloud password")
	wsURLOnly := flag.Bool("url-only", false, "Only show WebSocket URL (for use with external tools)")
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

	// Create context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Hash the password
	passwordHash := cloud.HashPassword(*password)

	fmt.Println("Shelly Cloud Real-Time Events Example")
	fmt.Println("=====================================")
	fmt.Println()

	// Authenticate
	fmt.Println("Authenticating with Shelly Cloud...")
	client, err := cloud.NewClientWithCredentials(ctx, *email, passwordHash)
	if err != nil {
		log.Printf("Authentication failed: %v", err)
		return
	}
	fmt.Println("Authentication successful!")
	fmt.Println()

	// Get WebSocket URL
	wsURL, err := client.GetWebSocketURL()
	if err != nil {
		log.Printf("Failed to get WebSocket URL: %v", err)
		return
	}

	if *wsURLOnly {
		fmt.Println("WebSocket URL:")
		fmt.Println(wsURL)
		fmt.Println()
		fmt.Println("You can connect to this URL using wscat or another WebSocket tool:")
		fmt.Println("  wscat -c \"" + wsURL + "\"")
		return
	}

	fmt.Println("WebSocket URL:", wsURL)
	fmt.Println()

	// Demonstrate event handling patterns
	demonstrateEventHandling(ctx, client)
}

// demonstrateEventHandling shows how to set up event handlers.
func demonstrateEventHandling(ctx context.Context, client *cloud.Client) {
	fmt.Println("Event Handling Patterns")
	fmt.Println("-----------------------")
	fmt.Println()
	fmt.Println("To use real-time events, you need a WebSocket library.")
	fmt.Println("Here's how to set up event handlers:")
	fmt.Println()

	// Create WebSocket manager (note: requires a dialer implementation)
	ws := cloud.NewWebSocket(client,
		cloud.WithReconnectInterval(5*time.Second),
		cloud.WithMaxReconnectInterval(5*time.Minute),
		cloud.WithPingInterval(30*time.Second),
		cloud.WithReadTimeout(60*time.Second),
	)

	// Register event handlers
	fmt.Println("1. Device Online Handler:")
	fmt.Println("   ws.OnDeviceOnline(func(deviceID string) {")
	printfOnline := `       fmt.Printf("Device %s came online\n", deviceID)`
	fmt.Println(printfOnline)
	fmt.Println("   })")
	fmt.Println()

	ws.OnDeviceOnline(func(deviceID string) {
		fmt.Printf("[EVENT] Device online: %s\n", deviceID)
	})

	fmt.Println("2. Device Offline Handler:")
	fmt.Println("   ws.OnDeviceOffline(func(deviceID string) {")
	printfOffline := `       fmt.Printf("Device %s went offline\n", deviceID)`
	fmt.Println(printfOffline)
	fmt.Println("   })")
	fmt.Println()

	ws.OnDeviceOffline(func(deviceID string) {
		fmt.Printf("[EVENT] Device offline: %s\n", deviceID)
	})

	fmt.Println("3. Status Change Handler:")
	fmt.Println("   ws.OnStatusChange(func(deviceID string, status json.RawMessage) {")
	printfStatus := `       fmt.Printf("Device %s status changed\n", deviceID)`
	fmt.Println(printfStatus)
	fmt.Println("   })")
	fmt.Println()

	ws.OnStatusChange(func(deviceID string, status json.RawMessage) {
		fmt.Printf("[EVENT] Status change: %s\n", deviceID)
		printStatusChange(status)
	})

	fmt.Println("4. Gen2+ NotifyStatus Handler:")
	fmt.Println("   ws.OnNotifyStatus(func(deviceID string, status json.RawMessage) {")
	fmt.Println("       // Handle Gen2+ status notifications")
	fmt.Println("   })")
	fmt.Println()

	ws.OnNotifyStatus(func(deviceID string, status json.RawMessage) {
		fmt.Printf("[EVENT] NotifyStatus: %s\n", deviceID)
		printGen2Status(status)
	})

	fmt.Println("5. Gen2+ NotifyEvent Handler (button presses, etc.):")
	fmt.Println("   ws.OnNotifyEvent(func(deviceID string, event json.RawMessage) {")
	fmt.Println("       // Handle button presses, input events, etc.")
	fmt.Println("   })")
	fmt.Println()

	ws.OnNotifyEvent(func(deviceID string, event json.RawMessage) {
		fmt.Printf("[EVENT] NotifyEvent: %s\n", deviceID)
		printGen2Event(event)
	})

	fmt.Println("6. Raw Message Handler (for all messages):")
	fmt.Println("   ws.OnMessage(func(msg *cloud.WebSocketMessage) {")
	fmt.Println("       // Handle any message type")
	fmt.Println("   })")
	fmt.Println()

	ws.OnMessage(func(msg *cloud.WebSocketMessage) {
		fmt.Printf("[RAW] Event: %s, Device: %s\n", msg.Event, msg.DeviceID)
	})

	// Demonstrate using an external WebSocket library
	fmt.Println("---")
	fmt.Println("Using with gorilla/websocket:")
	fmt.Println()
	fmt.Println("```go")
	fmt.Println("import \"github.com/gorilla/websocket\"")
	fmt.Println("")
	fmt.Println("// Create a dialer adapter")
	fmt.Println("type gorillaDialer struct{}")
	fmt.Println("")
	fmt.Println("func (d *gorillaDialer) Dial(ctx context.Context, url string, headers http.Header) " +
		"(cloud.WebSocketConn, error) {")
	fmt.Println("    conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, headers)")
	fmt.Println("    return conn, err")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("// Use the dialer")
	fmt.Println("ws := cloud.NewWebSocket(client, cloud.WithDialer(&gorillaDialer{}))")
	fmt.Println("```")
	fmt.Println()

	// Demonstrate polling as an alternative
	fmt.Println("---")
	fmt.Println("Alternative: Polling for status changes")
	fmt.Println()

	demonstratePolling(ctx, client)
}

// printStatusChange prints a status change payload.
func printStatusChange(data json.RawMessage) {
	var status map[string]any
	if err := json.Unmarshal(data, &status); err != nil {
		return
	}

	if relays, ok := status["relays"].([]any); ok {
		for i, relay := range relays {
			if r, ok := relay.(map[string]any); ok {
				if isOn, ok := r["ison"].(bool); ok {
					state := stateOff
					if isOn {
						state = stateOn
					}
					fmt.Printf("   Relay %d: %s\n", i, state)
				}
			}
		}
	}
}

// printGen2Status prints a Gen2+ status notification.
func printGen2Status(data json.RawMessage) {
	var status map[string]any
	if err := json.Unmarshal(data, &status); err != nil {
		return
	}

	// Check for switch status
	for key, value := range status {
		if v, ok := value.(map[string]any); ok {
			if output, ok := v["output"].(bool); ok {
				state := stateOff
				if output {
					state = stateOn
				}
				fmt.Printf("   %s: %s\n", key, state)
			}
			if apower, ok := v["apower"].(float64); ok {
				fmt.Printf("   %s power: %.1fW\n", key, apower)
			}
		}
	}
}

// printGen2Event prints a Gen2+ event notification.
func printGen2Event(data json.RawMessage) {
	var event struct {
		Events []struct {
			Component string `json:"component"`
			Event     string `json:"event"`
		} `json:"events"`
	}

	if err := json.Unmarshal(data, &event); err != nil {
		return
	}

	for _, e := range event.Events {
		fmt.Printf("   %s: %s\n", e.Component, e.Event)
	}
}

// demonstratePolling shows how to poll for status changes as an alternative.
func demonstratePolling(ctx context.Context, client *cloud.Client) {
	fmt.Println("Polling every 5 seconds for status changes...")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Track previous states
	previousStates := make(map[string]bool)

	// Create a polling ticker
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Poll once immediately
	pollDevices(ctx, client, previousStates)

	for {
		select {
		case <-ticker.C:
			pollDevices(ctx, client, previousStates)

		case <-sigCh:
			fmt.Println("\nStopping...")
			return

		case <-ctx.Done():
			return
		}
	}
}

// pollDevices polls all devices and reports changes.
func pollDevices(ctx context.Context, client *cloud.Client, previousStates map[string]bool) {
	devices, err := client.GetAllDevices(ctx)
	if err != nil {
		fmt.Printf("[POLL ERROR] %v\n", err)
		return
	}

	for id, status := range devices {
		// Parse raw status for relay states
		if len(status.Status) > 0 {
			var rawStatus map[string]any
			if err := json.Unmarshal(status.Status, &rawStatus); err == nil {
				// Check for Gen1 relays
				if relays, ok := rawStatus["relays"].([]any); ok {
					for i, relay := range relays {
						r, ok := relay.(map[string]any)
						if !ok {
							continue
						}
						isOn, ok := r["ison"].(bool)
						if !ok {
							continue
						}
						key := fmt.Sprintf("%s_relay_%d", id, i)
						if prev, exists := previousStates[key]; exists {
							if prev != isOn {
								state := stateOff
								if isOn {
									state = stateOn
								}
								fmt.Printf("[POLL] %s relay %d changed to %s\n", id, i, state)
							}
						}
						previousStates[key] = isOn
					}
				}

				// Check for Gen2+ switches
				for key, value := range rawStatus {
					if v, ok := value.(map[string]any); ok {
						if output, exists := v["output"]; exists {
							isOn, ok := output.(bool)
							if !ok {
								continue
							}
							stateKey := fmt.Sprintf("%s_%s", id, key)
							if prev, exists := previousStates[stateKey]; exists {
								if prev != isOn {
									state := stateOff
									if isOn {
										state = stateOn
									}
									fmt.Printf("[POLL] %s %s changed to %s\n", id, key, state)
								}
							}
							previousStates[stateKey] = isOn
						}
					}
				}
			}
		}

		// Check online status
		onlineKey := id + "_online"
		if prev, exists := previousStates[onlineKey]; exists {
			if prev != status.Online {
				state := "offline"
				if status.Online {
					state = "online"
				}
				fmt.Printf("[POLL] %s is now %s\n", id, state)
			}
		}
		previousStates[onlineKey] = status.Online
	}
}

// gorillaDialer is a placeholder showing how to implement a dialer.
// This would use the gorilla/websocket library in a real implementation.
//
// Example implementation:
//
//	type gorillaDialer struct{}
//
//	func (d *gorillaDialer) Dial(ctx context.Context, url string, headers http.Header) (cloud.WebSocketConn, error) {
//	    // import "github.com/gorilla/websocket"
//	    dialer := websocket.Dialer{
//	        HandshakeTimeout: 10 * time.Second,
//	    }
//	    conn, _, err := dialer.DialContext(ctx, url, headers)
//	    return conn, err
//	}
