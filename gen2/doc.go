// Package gen2 provides support for Gen2+ Shelly devices (Plus, Pro, Gen3, Gen4).
//
// Gen2+ devices use a unified JSON-RPC 2.0 API over HTTP, WebSocket, or MQTT.
// This package provides device implementations, component abstractions, and
// high-level methods for controlling and monitoring Gen2+ devices.
//
// # Supported Generations
//
// - Gen2 (Plus): Shelly Plus 1, Plus 1PM, Plus 2PM, Plus Plug S, etc.
// - Gen2 (Pro): Shelly Pro 1, Pro 1PM, Pro 2PM, Pro 3, Pro 3EM, Pro 4PM, etc.
// - Gen3: Shelly 1 Gen3, 1PM Gen3, 2PM Gen3, Wall Display, etc.
// - Gen4: Latest generation devices
//
// # Architecture
//
// Gen2+ devices use a component-based architecture where each device has one
// or more components (Switch, Cover, Light, Input, etc.). Each component has:
//   - Configuration (GetConfig/SetConfig methods)
//   - Status (GetStatus method)
//   - Actions (component-specific methods like Switch.Set, Cover.Open, etc.)
//
// The Shelly namespace provides device-level operations like GetDeviceInfo,
// Reboot, Update, etc.
//
// # Basic Usage
//
//	import (
//	    "context"
//	    "time"
//
//	    "github.com/tj-smith47/shelly-go/gen2"
//	    "github.com/tj-smith47/shelly-go/rpc"
//	    "github.com/tj-smith47/shelly-go/transport"
//	)
//
//	// Create HTTP transport
//	httpTransport := transport.NewHTTP("http://192.168.1.100",
//	    transport.WithTimeout(30*time.Second),
//	    transport.WithAuth("admin", "password"))
//
//	// Create RPC client
//	client := rpc.NewClient(httpTransport)
//	defer client.Close()
//
//	// Create Gen2 device
//	device := gen2.NewDevice(client)
//
//	// Get device information
//	info, err := device.Shelly().GetDeviceInfo(context.Background())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Device: %s (FW: %s)\n", info.Name, info.FirmwareVersion)
//
//	// Control a switch component
//	switchComp := device.Switch(0)
//	if err := switchComp.Set(context.Background(), true); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get switch status
//	status, err := switchComp.GetStatus(context.Background())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Switch output: %v\n", status.Output)
//
// # Components
//
// Gen2+ devices support the following component types:
//
//   - Switch: On/off relay control with power monitoring
//   - Cover: Roller shutter/blind control with positioning
//   - Light: Dimming and RGB/RGBW lighting control
//   - Input: Digital input monitoring with event detection
//   - DevicePower: Battery monitoring for battery-powered devices
//   - PM (Power Meter): Energy measurement and monitoring
//   - EM (Energy Monitor): Multi-phase energy monitoring
//   - EM1: Single-phase energy monitoring with data logging
//   - Voltmeter: Voltage measurement
//   - Temperature: Temperature sensor
//   - Humidity: Humidity sensor
//   - Smoke: Smoke detector
//   - Thermostat: Temperature control
//   - Script: Script management
//   - Schedule: Scheduled actions
//   - Webhook: Webhook configuration
//   - KVS: Key-value storage
//   - WiFi, Ethernet, BLE: Network configuration
//   - Cloud, MQTT, Ws: Cloud and messaging services
//   - Sys: System information
//   - UI: User interface settings
//
// # RPC Methods
//
// All component methods follow the JSON-RPC 2.0 pattern:
//   - <Component>.GetConfig - Get component configuration
//   - <Component>.SetConfig - Set component configuration
//   - <Component>.GetStatus - Get component status
//   - <Component>.<Action> - Component-specific actions
//
// For example:
//   - Switch.Set, Switch.Toggle
//   - Cover.Open, Cover.Close, Cover.Stop, Cover.GoToPosition
//   - Light.Set, Light.Toggle
//
// # Error Handling
//
// All methods return errors that can be checked with errors.Is():
//
//	err := device.Switch(0).Set(ctx, true)
//	if errors.Is(err, types.ErrNotFound) {
//	    // Component doesn't exist
//	}
//	if errors.Is(err, types.ErrAuth) {
//	    // Authentication failed
//	}
//
// # Notifications
//
// Gen2+ devices send notifications for status changes and events.
// Use a stateful transport (WebSocket or MQTT) to receive notifications:
//
//	wsTransport := transport.NewWebSocket("ws://192.168.1.100/rpc")
//	client := rpc.NewClient(wsTransport)
//
//	// Register notification handler
//	client.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {
//	    var notification struct {
//	        Src    string          `json:"src"`
//	        Status json.RawMessage `json:"status"`
//	    }
//	    json.Unmarshal(params, &notification)
//	    fmt.Printf("Status update from %s\n", notification.Src)
//	})
//
// # Batch Operations
//
// Multiple RPC calls can be batched into a single request for efficiency:
//
//	results, err := client.Batch().
//	    Add("Switch.GetStatus", map[string]any{"id": 0}).
//	    Add("Switch.GetStatus", map[string]any{"id": 1}).
//	    Add("Sys.GetStatus", nil).
//	    Execute(ctx)
//
// # Thread Safety
//
// All Gen2 device and component methods are safe for concurrent use.
// The underlying RPC client uses atomic operations and is thread-safe.
//
// # API Reference
//
// Official Shelly Gen2+ API documentation:
// https://shelly-api-docs.shelly.cloud/gen2/
package gen2
