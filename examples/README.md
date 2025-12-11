# Shelly-Go Examples

This directory contains comprehensive, runnable examples demonstrating all major features of the shelly-go library.

## Directory Structure

```
examples/
├── basic/              # Basic device control examples
│   ├── switch_control/     # Switch on/off, toggle, status
│   ├── cover_control/      # Cover open/close/position control
│   └── light_control/      # Light dimming, color, effects
├── discovery/          # Device discovery examples
│   ├── mdns/              # mDNS/Zeroconf discovery
│   ├── ble/               # Bluetooth Low Energy discovery
│   └── coiot/             # CoIoT multicast discovery (Gen1)
├── cloud/              # Shelly Cloud API examples
│   ├── auth/              # OAuth authentication
│   ├── realtime/          # Real-time WebSocket events
│   └── integrator/        # Integrator API (B2B)
├── provisioning/       # Device provisioning examples
│   ├── wifi/              # WiFi provisioning (AP mode)
│   └── ble/               # BLE-based provisioning
├── firmware/           # Firmware management examples
│   └── ota_update/        # OTA update process
├── backup/             # Backup and restore examples
│   ├── export/            # Export device configuration
│   └── restore/           # Restore configuration
└── advanced/           # Advanced use cases
    ├── batch_operations/  # Control multiple devices
    ├── scenes/            # Scene management
    ├── schedules/         # Schedule creation
    └── custom_component/  # Extending with custom components
```

## Running Examples

Each example is a standalone Go program that can be run directly:

```bash
# Basic switch control
cd basic/switch_control
go run main.go

# Device discovery
cd discovery/mdns
go run main.go

# Cloud authentication
cd cloud/auth
go run main.go
```

## Prerequisites

Most examples require:

1. **Go 1.25.5 or later** installed
2. **Shelly device** accessible on your network (for local examples)
3. **Shelly Cloud account** (for cloud examples)

Some examples may require additional setup documented in their respective directories.

## Configuration

Examples use environment variables or command-line flags for configuration:

```bash
# Using environment variables
export SHELLY_HOST=192.168.1.100
export SHELLY_USER=admin
export SHELLY_PASS=password
go run main.go

# Using flags
go run main.go -host 192.168.1.100 -user admin -pass password
```

## Basic Examples

### Switch Control

Demonstrates controlling a Shelly switch (Gen2+):

```bash
cd basic/switch_control
go run main.go -host 192.168.1.100
```

Features:
- Turning switch on/off
- Toggling switch state
- Getting switch status
- Getting power consumption
- Resetting energy counters

### Cover Control

Demonstrates controlling roller shutters/blinds:

```bash
cd basic/cover_control
go run main.go -host 192.168.1.100
```

Features:
- Opening/closing cover
- Setting position (percentage)
- Stopping cover
- Calibration
- Getting cover status

### Light Control

Demonstrates controlling Shelly lights with dimming and color:

```bash
cd basic/light_control
go run main.go -host 192.168.1.100
```

Features:
- Turning light on/off with brightness
- Setting RGB color
- Setting color temperature
- Light effects
- Getting light status

## Discovery Examples

### mDNS Discovery

Discovers Shelly devices on your local network using mDNS/Zeroconf:

```bash
cd discovery/mdns
go run main.go
```

Output:
```
Scanning for Shelly devices...
Found: Shelly Plus 1PM at 192.168.1.100 (Gen2)
Found: Shelly 1 at 192.168.1.101 (Gen1)
Found: Shelly Pro 2PM at 192.168.1.102 (Gen2)
Total: 3 devices found
```

### BLE Discovery

Discovers Shelly BLU devices via Bluetooth:

```bash
cd discovery/ble
go run main.go
```

### CoIoT Discovery

Discovers Gen1 devices using CoIoT multicast:

```bash
cd discovery/coiot
go run main.go
```

## Cloud Examples

### Authentication

Demonstrates OAuth authentication with Shelly Cloud:

```bash
cd cloud/auth
go run main.go -username your@email.com -password yourpassword
```

### Real-Time Events

Subscribe to real-time device events via Cloud WebSocket:

```bash
cd cloud/realtime
go run main.go
```

Features:
- WebSocket connection management
- Event subscription
- Event filtering
- Automatic reconnection

### Integrator API

Enterprise/B2B multi-account management:

```bash
cd cloud/integrator
go run main.go -api-key YOUR_API_KEY
```

## Provisioning Examples

### WiFi Provisioning

Provision a new Shelly device via its WiFi AP:

```bash
cd provisioning/wifi
go run main.go -ssid YourNetwork -password YourPassword
```

### BLE Provisioning

Provision Gen2+ devices via Bluetooth:

```bash
cd provisioning/ble
go run main.go
```

## Firmware Examples

### OTA Update

Check for and apply firmware updates:

```bash
cd firmware/ota_update
go run main.go -host 192.168.1.100
```

Features:
- Check for available updates
- Download firmware
- Apply update
- Monitor update progress
- Staged rollout

## Backup Examples

### Export Configuration

Export complete device configuration to JSON:

```bash
cd backup/export
go run main.go -host 192.168.1.100 -output device-backup.json
```

### Restore Configuration

Restore device from backup:

```bash
cd backup/restore
go run main.go -host 192.168.1.100 -input device-backup.json
```

## Advanced Examples

### Batch Operations

Control multiple devices simultaneously:

```bash
cd advanced/batch_operations
go run main.go
```

Features:
- Turn off all switches
- Set multiple devices to same state
- Get status from all devices
- Parallel execution

### Scenes

Create and manage scenes:

```bash
cd advanced/scenes
go run main.go
```

Features:
- Define named scenes
- Activate/deactivate scenes
- Save scenes to JSON
- Scene transitions

### Schedules

Create device schedules:

```bash
cd advanced/schedules
go run main.go
```

Features:
- Sunrise/sunset schedules
- Recurring daily schedules
- Weekly schedules
- Custom schedule expressions

### Custom Components

Extend the library with custom components:

```bash
cd advanced/custom_component
go run main.go
```

Shows how to:
- Implement custom component types
- Add new RPC methods
- Integrate with existing transports
- Test custom components

## Testing Examples

To verify examples work without real hardware:

```bash
# Run with mock devices
cd basic/switch_control
go test -v
```

Each example includes tests that use mock transports from the `internal/testutil` package.

## Troubleshooting

### Device Not Found

If discovery doesn't find your device:

1. Ensure device is on the same network
2. Check firewall settings (UDP port 5353 for mDNS)
3. Try discovery with longer timeout
4. Manually specify device IP address

### Authentication Errors

If you get auth errors:

1. Check username/password
2. Ensure authentication is enabled on device
3. Update device firmware to latest version
4. Try resetting device credentials

### Connection Timeouts

If operations timeout:

1. Verify device IP address
2. Check network connectivity
3. Increase timeout in client options
4. Check device isn't overloaded

## Contributing Examples

To add a new example:

1. Create directory under appropriate category
2. Write `main.go` with clear comments
3. Add `README.md` explaining the example
4. Include error handling
5. Add tests using mock devices
6. Update this README with example description

See [CONTRIBUTING.md](../CONTRIBUTING.md) for detailed guidelines.

## License

All examples are licensed under MIT License - see [LICENSE](../LICENSE).
