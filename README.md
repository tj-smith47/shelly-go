# shelly-go

[![Go Reference](https://pkg.go.dev/badge/github.com/tj-smith47/shelly-go.svg)](https://pkg.go.dev/github.com/tj-smith47/shelly-go)
[![Coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/tj-smith47/shelly-go/badges/coverage.json)](https://github.com/tj-smith47/shelly-go/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/tj-smith47/shelly-go)](https://goreportcard.com/report/github.com/tj-smith47/shelly-go)
[![CI](https://github.com/tj-smith47/shelly-go/workflows/CI/badge.svg)](https://github.com/tj-smith47/shelly-go/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A comprehensive, production-ready Go library for controlling Shelly smart home devices across **all generations** (Gen1, Gen2, Gen3, Gen4) and **all communication protocols** (HTTP, WebSocket, MQTT, CoAP/CoIoT).

## Features

- **🎯 Complete Device Coverage** - Support for Gen1, Gen2 (Plus), Gen3, Gen4, Pro, BLU, and Wave devices
- **🔌 Multiple Protocols** - HTTP, WebSocket, MQTT, CoAP/CoIoT, Matter (upcoming)
- **☁️ Cloud API Integration** - First Go library with full Cloud Control API support
- **🔍 Auto-Discovery** - mDNS, BLE, and CoIoT device discovery
- **📊 Real-Time Events** - WebSocket and notification support for live device updates
- **🎨 Extensible Architecture** - Easy to add new devices and components
- **🧪 Thoroughly Tested** - 90% test coverage with comprehensive unit and integration tests
- **📚 Well Documented** - Complete godoc documentation with runnable examples
- **⚡ Modern Go** - Built for Go 1.25.5+ with latest features

## Installation

```bash
go get github.com/tj-smith47/shelly-go
```

Requires Go 1.25.5 or later.

## Quick Start

### Gen2/Gen3/Gen4 Devices (RPC-based)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tj-smith47/shelly-go/gen2"
    "github.com/tj-smith47/shelly-go/gen2/components"
)

func main() {
    // Create a client for your Shelly device
    client := gen2.NewClient("http://192.168.1.100")

    // Control a switch
    sw := components.NewSwitch(client, 0)
    err := sw.Set(context.Background(), true)
    if err != nil {
        log.Fatal(err)
    }

    // Get switch status
    status, err := sw.GetStatus(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Switch is %s\n", status.Output)
}
```

### Gen1 Devices (REST-based)

```go
package main

import (
    "context"
    "log"

    "github.com/tj-smith47/shelly-go/gen1"
    "github.com/tj-smith47/shelly-go/gen1/components"
)

func main() {
    client := gen1.NewClient("http://192.168.1.101")

    relay := components.NewRelay(client, 0)
    err := relay.Set(context.Background(), true)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Device Discovery

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/tj-smith47/shelly-go/discovery"
)

func main() {
    scanner := discovery.NewScanner()

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    devices, err := scanner.Scan(ctx)
    if err != nil {
        log.Fatal(err)
    }

    for _, device := range devices {
        fmt.Printf("Found: %s at %s (Gen%d)\n", device.Name, device.Address, device.Generation)
    }
}
```

### Cloud API

```go
package main

import (
    "context"
    "log"

    "github.com/tj-smith47/shelly-go/cloud"
)

func main() {
    // Authenticate with Shelly Cloud
    client := cloud.NewClient()
    err := client.Authenticate(context.Background(), "username", "password")
    if err != nil {
        log.Fatal(err)
    }

    // List all devices
    devices, err := client.ListDevices(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Control a device via cloud
    err = client.SetSwitch(context.Background(), devices[0].ID, 0, true)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Supported Devices

This library supports **all Shelly devices** across all generations:

- ✅ **Gen1** - Shelly 1/1PM/2.5/4Pro/Plug/Bulb/RGBW2/Dimmer/EM/3EM/H&T/Smoke/Flood/Door/Window/Motion/etc.
- ✅ **Gen2 (Plus)** - Shelly Plus 1/1PM/2PM/i4/Plug S/H&T/Smoke/Wall Dimmer/RGBW PM/UNI/etc.
- ✅ **Pro** - Shelly Pro 1/1PM/2/2PM/3/3EM/4PM/Dimmer/Dual Cover/EM-50/etc.
- ✅ **Gen3** - Shelly 1/1PM/2PM/1L/2L/PM Mini/Plug S/i4/H&T/EM/Dimmer/Wall Display/etc.
- ✅ **Gen4** - Shelly 1/1PM (and future Gen4 devices)
- ✅ **BLU (Bluetooth)** - Shelly BLU Button/Door/Motion/H&T/TRV/Gateway/etc.
- ✅ **Wave (Z-Wave)** - Shelly Wave 1/1PM/2PM/Plug/Shutter/etc.

See [DEVICES.md](DEVICES.md) for complete device matrix.

## Architecture

The library is organized into focused packages:

```
shelly-go/
├── types/          Core interfaces and types
├── transport/      Communication layer (HTTP, WebSocket, MQTT, CoAP)
├── rpc/            RPC framework for Gen2+ devices
├── gen1/           Gen1 device support
├── gen2/           Gen2+ device support (Plus, Pro, Gen3, Gen4)
├── cloud/          Shelly Cloud API
├── discovery/      Device discovery (mDNS, BLE, CoIoT)
├── events/         Event bus and notification system
├── helpers/        Convenience utilities (batch, groups, scenes)
├── profiles/       Device profiles and capabilities
└── examples/       Runnable examples
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for design decisions and patterns.

## Key Concepts

### Transports

All communication goes through transport implementations:

```go
import "github.com/tj-smith47/shelly-go/transport"

// HTTP transport (most common)
http := transport.NewHTTP("http://192.168.1.100",
    transport.WithTimeout(30*time.Second),
    transport.WithAuth("admin", "password"))

// WebSocket transport (real-time)
ws := transport.NewWebSocket("ws://192.168.1.100/rpc",
    transport.WithReconnect(true))

// MQTT transport
mqtt := transport.NewMQTT("mqtt://192.168.1.10:1883",
    transport.WithMQTTTopic("shellies/shellyplus1-abc123"))
```

### Components

Devices are composed of components (switches, covers, lights, sensors, etc.):

```go
// Each component has GetConfig, SetConfig, and GetStatus
config, err := sw.GetConfig(ctx)
config.Name = "Living Room Light"
err = sw.SetConfig(ctx, config)

status, err := sw.GetStatus(ctx)
fmt.Printf("Power: %.2fW\n", status.APower)
```

### Events

Subscribe to real-time device events:

```go
import "github.com/tj-smith47/shelly-go/events"

bus := events.NewBus()

// Subscribe to switch events
bus.Subscribe(events.FilterByComponent("switch:0"), func(e events.Event) {
    fmt.Printf("Switch changed: %+v\n", e)
})

// Connect device to event bus
device.AttachEventBus(bus)
```

### Batch Operations

Control multiple devices at once:

```go
import "github.com/tj-smith47/shelly-go/helpers"

// Turn off all switches
err := helpers.BatchSet(ctx, devices, false)

// Create a scene
scene := helpers.NewScene("Movie Time")
scene.Add(livingRoomLight, helpers.WithBrightness(20))
scene.Add(tvBacklight, helpers.WithRGB(255, 0, 0))
scene.Activate(ctx)
```

## Documentation

- [API Reference](https://pkg.go.dev/github.com/tj-smith47/shelly-go) - Complete godoc documentation
- [Examples](examples/) - Runnable code examples
- [DEVICES.md](DEVICES.md) - Supported devices matrix
- [ARCHITECTURE.md](ARCHITECTURE.md) - Internal architecture
- [MIGRATION.md](MIGRATION.md) - Migrating from other libraries
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [CHANGELOG.md](CHANGELOG.md) - Version history

## Examples

See the [examples/](examples/) directory for complete, runnable examples:

- **Basic Control** - Switch, cover, and light control
- **Discovery** - mDNS, BLE, and CoIoT discovery
- **Cloud API** - Authentication and remote control
- **Real-Time Events** - WebSocket event handling
- **Energy Monitoring** - Historical energy data retrieval and analysis
- **Batch Operations** - Controlling multiple devices
- **Scenes** - Creating and managing scenes
- **Provisioning** - WiFi and BLE device setup
- **Firmware Updates** - OTA update management

## Testing

Run tests:

```bash
# Unit tests
go test ./...

# With coverage
go test -cover ./...

# Integration tests (requires devices)
go test -tags=integration ./...
```

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Comparison with Other Libraries

| Feature | shelly-go | jcodybaker/go-shelly | jojomi/go-shelly |
|---------|-----------|---------------------|------------------|
| Gen1 Support | ✅ Full | ❌ | ✅ Limited |
| Gen2+ Support | ✅ Full | ✅ | ❌ |
| Gen3/Gen4 | ✅ | ✅ | ❌ |
| Cloud API | ✅ | ❌ | ❌ |
| Discovery | ✅ All protocols | ❌ | ❌ |
| WebSocket | ✅ Bidirectional | ✅ | ❌ |
| MQTT | ✅ | ❌ | ❌ |
| CoAP/CoIoT | ✅ | ❌ | ❌ |
| Test Coverage | ≥90% | ~60% | <20% |
| Documentation | Complete | Good | Minimal |

## Roadmap

- [x] Gen1 support (HTTP, CoIoT)
- [x] Gen2/Gen3/Gen4 support (RPC)
- [x] Cloud API integration
- [x] Device discovery (mDNS, CoIoT, WiFi AP, BLE)
- [x] Event system
- [x] WiFi provisioning (platform-specific: Linux/macOS/Windows)
- [x] BLE discovery (TinyGo implementation for Linux/macOS; see `examples/discovery/ble`)
- [x] BLE provisioning (TinyGo implementation for Linux/macOS; see `examples/provisioning/ble`)
- [x] Backup/restore functionality
- [x] Firmware update management
- [x] Batch operations and device groups
- [x] Scene management
- [x] Matter protocol support (RPC control of Matter-enabled Shelly devices)
- [x] Zigbee support (RPC control of Zigbee-enabled Gen4 devices)
- [x] Z-Wave support (Wave device profiles; IP-enabled Wave devices use Gen2 RPC)
- [x] LoRa add-on support (full RPC implementation)
- [x] Integrator API (B2B fleet management, analytics, provisioning)

## Known Limitations

### Bluetooth (BLE)

BLE discovery and provisioning are fully implemented using the TinyGo bluetooth library:

- **Linux**: Works with BlueZ (requires `bluez` package)
- **macOS**: Works with CoreBluetooth
- **Windows**: Not supported (returns `ErrBLETransmitterNotSupported`)

**Requirements:**
- Bluetooth adapter must be available and enabled
- Appropriate OS permissions (may need root/sudo on Linux)
- No other application should be blocking the bluetooth adapter

**Features:**
- Discover Shelly devices in BLE provisioning mode (Gen2+)
- Parse BTHome sensor data from Shelly BLU devices (buttons, sensors, etc.)
- Connect to devices and send RPC commands over BLE GATT
- Provision WiFi credentials to unprovisioned devices

See the examples at `examples/discovery/ble` and `examples/provisioning/ble`.

### Integration Tests

Integration tests exist but require real Shelly devices at specific IP addresses. They are skipped in CI environments (`SHELLY_CI=1`).

## License

MIT License - see [LICENSE](LICENSE) for details.

## Credits

Generated by Claude Opus 4.5 over many iterations 🤖

Shelly® is a registered trademark of Allterco Robotics. This project is not affiliated with or endorsed by Allterco Robotics.

## Support

- [GitHub Issues](https://github.com/tj-smith47/shelly-go/issues) - Bug reports and feature requests
- [Discussions](https://github.com/tj-smith47/shelly-go/discussions) - Questions and community support
- [Official Shelly API Docs](https://shelly-api-docs.shelly.cloud/) - Reference documentation

## Acknowledgments

Thanks to the Shelly community and other library authors for inspiration and testing.
