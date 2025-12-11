// Package shelly provides a comprehensive Go library for controlling Shelly smart home devices
// across all device generations (Gen1, Gen2, Gen3, Gen4) and communication protocols
// (HTTP, WebSocket, MQTT, CoAP/CoIoT).
//
// # Overview
//
// This library provides a unified interface for interacting with Shelly devices, whether
// you're controlling them locally via HTTP/WebSocket, discovering them on your network,
// or managing them through the Shelly Cloud API.
//
// # Quick Start
//
// For Gen2/Gen3/Gen4 devices (RPC-based):
//
//	client := gen2.NewClient("http://192.168.1.100")
//	sw := components.NewSwitch(client, 0)
//	err := sw.Set(context.Background(), true)
//
// For Gen1 devices (REST-based):
//
//	client := gen1.NewClient("http://192.168.1.101")
//	relay := components.NewRelay(client, 0)
//	err := relay.Set(context.Background(), true)
//
// # Package Organization
//
// The library is organized into several focused packages:
//
//   - types: Core interfaces and type definitions
//   - transport: Communication layer implementations (HTTP, WebSocket, MQTT, CoAP)
//   - rpc: RPC framework for Gen2+ devices
//   - gen1: Support for Gen1 devices
//   - gen2: Support for Gen2+ devices (Plus, Pro, Gen3, Gen4)
//   - cloud: Shelly Cloud API integration
//   - discovery: Device discovery via mDNS, BLE, and CoIoT
//   - events: Event bus and real-time notifications
//   - helpers: Convenience utilities for batch operations, groups, and scenes
//   - profiles: Device profiles and capability detection
//
// # Device Generations
//
// Shelly devices come in multiple generations with different protocols:
//
// Gen1: REST API over HTTP, CoIoT for real-time updates
//   - Examples: Shelly 1, 1PM, 2.5, Plug, Bulb, RGBW2, Dimmer, EM, H&T
//
// Gen2 (Plus): RPC over HTTP/WebSocket, MQTT support
//   - Examples: Shelly Plus 1, 1PM, 2PM, i4, Plug S, H&T
//
// Pro: Enhanced Gen2 with Ethernet, ModBus, additional I/O
//   - Examples: Shelly Pro 1, 1PM, 2PM, 3, 3EM, 4PM, Dimmer
//
// Gen3: Latest generation with improved hardware and features
//   - Examples: Shelly 1 Gen3, 1PM Gen3, 2PM Gen3, Wall Display
//
// Gen4: Future-ready devices (as released by Allterco)
//   - Examples: Shelly 1 Gen4, 1PM Gen4
//
// BLU: Bluetooth Low Energy devices
//   - Examples: BLU Button, Door/Window, Motion, H&T, TRV
//
// Wave: Z-Wave devices
//   - Examples: Wave 1, 1PM, 2PM, Plug, Shutter
//
// # Components
//
// Devices are composed of components that provide specific functionality:
//
//   - Switch: On/off control with power monitoring
//   - Cover: Roller shutter/blind control with position
//   - Light: Dimming and color control
//   - Input: Button and sensor inputs
//   - PM/EM: Power and energy monitoring
//   - WiFi/Ethernet/BLE: Network configuration
//   - Cloud/MQTT/Webhook: Integration services
//   - Script: JavaScript automation
//   - Temperature/Humidity/Smoke: Environmental sensors
//   - Thermostat: Climate control
//
// Each component implements standard methods:
//
//   - GetConfig: Retrieve component configuration
//   - SetConfig: Update component configuration
//   - GetStatus: Get current component status
//
// # Transports
//
// Communication with devices uses pluggable transports:
//
//	// HTTP transport (most common)
//	http := transport.NewHTTP("http://192.168.1.100",
//	    transport.WithTimeout(30*time.Second),
//	    transport.WithAuth("admin", "password"))
//
//	// WebSocket transport for real-time communication
//	ws := transport.NewWebSocket("ws://192.168.1.100/rpc")
//
//	// MQTT transport
//	mqtt := transport.NewMQTT("mqtt://broker:1883",
//	    transport.WithMQTTTopic("shellies/device-id"))
//
// # Discovery
//
// Devices can be discovered automatically using multiple methods:
//
//	scanner := discovery.NewScanner()
//	devices, err := scanner.Scan(ctx)
//	for _, device := range devices {
//	    fmt.Printf("Found %s at %s\n", device.Name, device.Address)
//	}
//
// # Cloud API
//
// Control devices remotely via Shelly Cloud:
//
//	client := cloud.NewClient()
//	err := client.Authenticate(ctx, "username", "password")
//	devices, err := client.ListDevices(ctx)
//	err = client.SetSwitch(ctx, devices[0].ID, 0, true)
//
// # Events
//
// Subscribe to real-time device events:
//
//	bus := events.NewBus()
//	bus.Subscribe(events.FilterByDevice(deviceID), func(e events.Event) {
//	    fmt.Printf("Event: %+v\n", e)
//	})
//	device.AttachEventBus(bus)
//
// # Error Handling
//
// The library defines standard error types:
//
//   - ErrNotFound: Resource not found
//   - ErrAuth: Authentication failed
//   - ErrTimeout: Operation timed out
//   - ErrNotSupported: Feature not supported by device
//
// All errors are wrapped with context using fmt.Errorf with %w.
//
// # Context Support
//
// All operations accept context.Context for cancellation and timeout:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	status, err := sw.GetStatus(ctx)
//
// # Thread Safety
//
// All client and device implementations are safe for concurrent use unless
// otherwise documented. Components share the underlying client and can be
// used from multiple goroutines.
//
// # Extensibility
//
// The library is designed to be extensible for new devices and firmware features:
//
//   - All structs include RawFields map[string]json.RawMessage for unknown fields
//   - Component interface allows custom implementations
//   - Transport interface supports new protocols
//
// # Testing
//
// The library includes comprehensive testing utilities:
//
//	import "github.com/tj-smith47/shelly-go/internal/testutil"
//
//	mock := testutil.NewMockTransport()
//	mock.AddResponse("Shelly.GetDeviceInfo", testutil.LoadFixture("device.json"))
//	client := gen2.NewClient(mock)
//
// # Examples
//
// See the examples/ directory for complete, runnable examples covering:
//
//   - Basic device control
//   - Device discovery
//   - Cloud API usage
//   - Real-time events
//   - Batch operations
//   - Scene management
//   - Device provisioning
//   - Firmware updates
//
// # Official Documentation
//
// For more information about Shelly devices and protocols, see:
//
//   - Gen2+ API: https://shelly-api-docs.shelly.cloud/gen2/
//   - Gen1 API: https://shelly-api-docs.shelly.cloud/gen1/
//   - Cloud API: https://shelly-api-docs.shelly.cloud/cloud-control-api/
//
// # License
//
// # MIT License - Copyright (c) 2025 TJ Smith
//
// Shelly® is a registered trademark of Allterco Robotics.
// This project is not affiliated with or endorsed by Allterco Robotics.
package shelly
