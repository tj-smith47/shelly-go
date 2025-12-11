# Supported Devices

This document lists all Shelly device generations and models supported by shelly-go.

## Table of Contents

- [Generation Overview](#generation-overview)
- [Gen1 Devices](#gen1-devices)
- [Gen2 (Plus) Devices](#gen2-plus-devices)
- [Gen3 Devices](#gen3-devices)
- [Gen4 Devices](#gen4-devices)
- [Components by Device](#components-by-device)
- [Feature Support Matrix](#feature-support-matrix)

## Generation Overview

| Generation | API Type | Protocol | Discovery | Auth Method | Library Package |
|------------|----------|----------|-----------|-------------|-----------------|
| Gen1 | REST | HTTP | CoIoT (mDNS) | Basic Auth | `gen1/` |
| Gen2 (Plus) | JSON-RPC 2.0 | HTTP/WebSocket | mDNS | Digest Auth | `gen2/` |
| Gen3 | JSON-RPC 2.0 | HTTP/WebSocket | mDNS | Digest Auth | `gen2/` |
| Gen4 | JSON-RPC 2.0 | HTTP/WebSocket | mDNS | Digest Auth | `gen2/` |

**Note:** Gen2, Gen3, and Gen4 devices all use the same JSON-RPC 2.0 API and are handled by the `gen2/` package. The library automatically detects the device generation.

## Gen1 Devices

Gen1 devices use a REST API over HTTP. They are identified by device types without "Plus", "Pro", or "Gen" in the name.

### Relays & Switches

| Device | Model | Components | Power Metering |
|--------|-------|------------|----------------|
| Shelly 1 | SHSW-1 | 1 Relay | No |
| Shelly 1PM | SHSW-PM | 1 Relay | Yes |
| Shelly 1L | SHSW-L | 1 Relay (no neutral) | No |
| Shelly 2 | SHSW-21 | 2 Relays | Yes |
| Shelly 2.5 | SHSW-25 | 2 Relays/Roller | Yes |
| Shelly 4Pro | SHSW-44 | 4 Relays | Yes |

### Plugs

| Device | Model | Components | Power Metering |
|--------|-------|------------|----------------|
| Shelly Plug | SHPLG-1 | 1 Relay | Yes |
| Shelly Plug S | SHPLG-S | 1 Relay | Yes |
| Shelly Plug US | SHPLG-U1 | 1 Relay | Yes |

### Lighting

| Device | Model | Components | Features |
|--------|-------|------------|----------|
| Shelly Dimmer 2 | SHDM-2 | 1 Light | Brightness 0-100% |
| Shelly RGBW2 | SHRGBW2 | 4 Channels | RGB + White, Color/White modes |
| Shelly Duo | SHBDUO-1 | 1 Light | Brightness, Color Temperature |
| Shelly Duo GU10 | SHBDUO-G10 | 1 Light | Brightness, Color Temperature |
| Shelly Vintage | SHVIN-1 | 1 Light | Warm White, Brightness |
| Shelly Bulb | SHBLB-1 | 1 Light | RGBW, Effects |
| Shelly Duo RGBW | SHCB-1 | 1 Light | RGB + White |

### Energy Monitoring

| Device | Model | Channels | Features |
|--------|-------|----------|----------|
| Shelly EM | SHEM | 2 CT | Power, Energy, Power Factor |
| Shelly 3EM | SHEM-3 | 3 CT | Power, Energy, PF, Voltage |

### Inputs & Sensors

| Device | Model | Components | Features |
|--------|-------|------------|----------|
| Shelly i3 | SHIX3-1 | 3 Inputs | Digital inputs, actions |
| Shelly Button 1 | SHBTN-1 | 1 Button | Single/double/triple/long press |
| Shelly Uni | SHUNI-1 | 2 Relays, 2 Inputs | ADC, Sensors |

### Environmental Sensors

| Device | Model | Sensors | Battery |
|--------|-------|---------|---------|
| Shelly H&T | SHHT-1 | Temperature, Humidity | Yes |
| Shelly Flood | SHWT-1 | Water, Temperature | Yes |
| Shelly Door/Window 2 | SHDW-2 | Contact, Vibration, Lux, Tilt | Yes |
| Shelly Gas | SHGS-1 | Gas (CH4/LPG) | No |
| Shelly Smoke | SHSM-01 | Smoke | Yes |
| Shelly Motion | SHMOS-01 | Motion, Lux | Yes |
| Shelly Motion 2 | SHMOS-02 | Motion, Lux | Yes |

### HVAC

| Device | Model | Features |
|--------|-------|----------|
| Shelly TRV | SHTRV-01 | Thermostatic Radiator Valve |

## Gen2 (Plus) Devices

Gen2 "Plus" devices use JSON-RPC 2.0 over HTTP or WebSocket. They have ESP32 processors and support mDNS discovery.

### Relays & Switches

| Device | Model | Components | Power Metering |
|--------|-------|------------|----------------|
| Shelly Plus 1 | SNSW-001X16EU | 1 Switch | No |
| Shelly Plus 1PM | SNSW-001P16EU | 1 Switch | Yes |
| Shelly Plus 1 Mini | SNSW-001X8EU | 1 Switch | No |
| Shelly Plus 1PM Mini | SNSW-001P8EU | 1 Switch | Yes |
| Shelly Plus 2PM | SNSW-002P16EU | 2 Switches/Cover | Yes |

### Plugs

| Device | Model | Components | Power Metering |
|--------|-------|------------|----------------|
| Shelly Plus Plug S | SNPL-00112EU | 1 Switch | Yes |
| Shelly Plus Plug US | SNPL-00116US | 1 Switch | Yes |
| Shelly Plus Plug IT | SNPL-00110IT | 1 Switch | Yes |
| Shelly Plus Plug UK | SNPL-00112UK | 1 Switch | Yes |

### Lighting

| Device | Model | Components | Features |
|--------|-------|------------|----------|
| Shelly Plus Wall Dimmer | SNDM-0013US | 1 Light | Brightness, Leading/Trailing Edge |
| Shelly Plus 0-10V Dimmer | SNDM-00100WW | 1 Light | 0-10V Output |
| Shelly Plus RGBW PM | SNDC-0D4P10WW | 4 Channels | RGBW, CCT, Power Metering |

### Inputs

| Device | Model | Inputs | Features |
|--------|-------|--------|----------|
| Shelly Plus i4 | SNSN-0024X | 4 Inputs | Digital inputs, actions |
| Shelly Plus i4 DC | SNSN-0D24X | 4 Inputs | DC-powered variant |

### Energy Monitoring

| Device | Model | Channels | Features |
|--------|-------|----------|----------|
| Shelly Plus PM Mini | SNPM-001PCEU | 1 | Power, Energy |

### Environmental Sensors

| Device | Model | Sensors | Battery |
|--------|-------|---------|---------|
| Shelly Plus H&T | SNSN-0013A | Temperature, Humidity | Yes |
| Shelly Plus Smoke | SNSN-0031Z | Smoke | Yes |

### Special

| Device | Model | Features |
|--------|-------|----------|
| Shelly Plus Uni | SNUN-001 | Universal I/O, ADC |

## Gen3 Devices

Gen3 devices have improved processors and increased memory. They use the same JSON-RPC 2.0 API as Gen2.

### Relays & Switches

| Device | Model | Components | Power Metering |
|--------|-------|------------|----------------|
| Shelly 1 Gen3 | S3SW-001X16EU | 1 Switch | No |
| Shelly 1PM Gen3 | S3SW-001P16EU | 1 Switch | Yes |
| Shelly 1L Gen3 | S3SW-001L16EU | 1 Switch (no neutral) | No |
| Shelly 2PM Gen3 | S3SW-002P16EU | 2 Switches/Cover | Yes |
| Shelly 2L Gen3 | S3SW-002L16EU | 2 Switches (no neutral) | No |

### Mini Series

| Device | Model | Components | Power Metering |
|--------|-------|------------|----------------|
| Shelly 1 Mini Gen3 | S3SW-001X8EU | 1 Switch | No |
| Shelly 1PM Mini Gen3 | S3SW-001P8EU | 1 Switch | Yes |
| Shelly PM Mini Gen3 | S3PM-001PCEU | Power Monitor | Yes |

### Plugs

| Device | Model | Components | Power Metering |
|--------|-------|------------|----------------|
| Shelly Plug S Gen3 | S3PL-00112EU | 1 Switch | Yes |
| Shelly Plug S MTR Gen3 | S3PL-00112EUMTR | 1 Switch | Yes (Enhanced) |
| Shelly Outdoor Plug S Gen3 | S3PL-00212EU | 1 Switch | Yes |
| Shelly Plug PM Gen3 | S3PL-10112EU | 1 Switch | Yes |

### Lighting

| Device | Model | Components | Features |
|--------|-------|------------|----------|
| Shelly Dimmer Gen3 | S3DM-0010WW | 1 Light | Brightness |
| Shelly Dimmer 0/1-10V PM Gen3 | S3DM-0D10WW | 1 Light | 0-10V, Power Metering |
| Shelly DALI Dimmer Gen3 | S3DM-0D0WW | DALI | DALI Protocol |

### Inputs

| Device | Model | Inputs | Features |
|--------|-------|--------|----------|
| Shelly i4 Gen3 | S3SN-0024X | 4 Inputs | Digital inputs, actions |

### Energy Monitoring

| Device | Model | Channels | Features |
|--------|-------|----------|----------|
| Shelly EM Gen3 | S3EM-002CXCEU | 2 CT | Power, Energy, CT Clamps |
| Shelly 3EM-63 Gen3 | S3EM-003CXCEU | 3 CT | Power, Energy, PF, 63A |

### Environmental Sensors

| Device | Model | Sensors | Battery |
|--------|-------|---------|---------|
| Shelly H&T Gen3 | S3SN-0U12A | Temperature, Humidity | Yes |

### Covers

| Device | Model | Features |
|--------|-------|----------|
| Shelly Shutter Gen3 | S3SW-002PCEU | Roller/Cover, Position Control |

## Gen4 Devices

Gen4 devices feature multi-protocol support (Wi-Fi 6, Bluetooth, Zigbee, Matter). They use the same JSON-RPC 2.0 API.

### Relays & Switches

| Device | Model | Components | Power Metering | Protocols |
|--------|-------|------------|----------------|-----------|
| Shelly 1 Gen4 | S4SW-001X16EU | 1 Switch | No | Wi-Fi/Zigbee/Matter |
| Shelly 1PM Gen4 | S4SW-001P16EU | 1 Switch | Yes | Wi-Fi/Zigbee/Matter |
| Shelly 2PM Gen4 | S4SW-002P16EU | 2 Switches/Cover | Yes | Wi-Fi/Zigbee/Matter |

### Mini Series

| Device | Model | Components | Power Metering | Protocols |
|--------|-------|------------|----------------|-----------|
| Shelly 1 Mini Gen4 | S4SW-001X8EU | 1 Switch | No | Wi-Fi/Zigbee/Matter |
| Shelly 1PM Mini Gen4 | S4SW-001P8EU | 1 Switch | Yes | Wi-Fi/Zigbee/Matter |

### Energy Monitoring

| Device | Model | Channels | Features |
|--------|-------|----------|----------|
| Shelly EM Mini Gen4 | S4EM-001XCEU | 1 CT | Power, Energy, Zigbee Repeater |

### Plugs

| Device | Model | Components | Power Metering |
|--------|-------|------------|----------------|
| Shelly Plug US Gen4 | S4PL-00116US | 1 Switch | Yes |

### Sensors

| Device | Model | Sensors | Features |
|--------|-------|---------|----------|
| Shelly Flood Sensor Gen4 | S4SN-0W1X | Water | Wireless, Real-time Alerts |

### Displays

| Device | Model | Features |
|--------|-------|----------|
| Shelly Wall Display X2 | S4WD-00695EU | 6.95" Touch, Temp/Humidity/Light Sensors |

## Components by Device

### Component Type Mapping

| Gen1 Component | Gen2+ Component | Description |
|----------------|-----------------|-------------|
| `relay` | `switch` | On/off output control |
| `roller` | `cover` | Blinds/shutters/covers |
| `light` | `light` | Dimmable light |
| `emeter` | `em` | Energy meter |
| `meter` | `pm` | Power meter |
| - | `input` | Digital input |
| `temperature` | `temperature` | Temperature sensor |
| `humidity` | `humidity` | Humidity sensor |

### Gen2+ Components Reference

| Component | Methods | Description |
|-----------|---------|-------------|
| `Switch` | Set, Toggle, GetStatus, GetConfig, SetConfig | Relay/switch control |
| `Cover` | Open, Close, Stop, GoToPosition, GetStatus | Blinds/shutters |
| `Light` | Set, Toggle, GetStatus, GetConfig | Dimmable lights |
| `Input` | GetStatus, GetConfig, SetConfig | Digital inputs |
| `PM` | GetStatus | Power meter (standalone) |
| `EM` | GetStatus, GetConfig | Energy meter |
| `Temperature` | GetStatus, GetConfig | Temperature sensor |
| `Humidity` | GetStatus, GetConfig | Humidity sensor |
| `DevicePower` | GetStatus | Battery/power status |
| `Sys` | GetStatus, GetConfig, SetConfig | System status |
| `WiFi` | GetStatus, GetConfig, SetConfig, Scan | WiFi management |
| `Ethernet` | GetStatus, GetConfig | Ethernet (Pro only) |
| `BLE` | GetStatus, GetConfig | Bluetooth LE |
| `Cloud` | GetStatus, GetConfig, SetConfig | Cloud connectivity |
| `MQTT` | GetStatus, GetConfig, SetConfig | MQTT client |
| `Webhook` | List, Create, Update, Delete | Webhook management |
| `Script` | List, Create, Start, Stop, Eval | Scripting engine |
| `Schedule` | List, Create, Update, Delete | Scheduled actions |
| `KVS` | Get, Set, GetMany, Delete | Key-value storage |

## Feature Support Matrix

### Protocol Support

| Feature | Gen1 | Gen2 | Gen3 | Gen4 |
|---------|------|------|------|------|
| HTTP REST | Yes | - | - | - |
| HTTP RPC | - | Yes | Yes | Yes |
| WebSocket | - | Yes | Yes | Yes |
| MQTT | Yes | Yes | Yes | Yes |
| CoIoT | Yes | - | - | - |
| mDNS Discovery | - | Yes | Yes | Yes |
| CoIoT Discovery | Yes | - | - | - |
| Matter | - | - | - | Yes |
| Zigbee | - | - | - | Yes |
| Wi-Fi 6 | - | - | - | Yes |

### Feature Support

| Feature | Gen1 | Gen2 | Gen3 | Gen4 |
|---------|------|------|------|------|
| Scripting | No | Yes | Yes | Yes |
| Schedules | Basic | Advanced | Advanced | Advanced |
| Webhooks | No | Yes | Yes | Yes |
| KVS Storage | No | Yes | Yes | Yes |
| OTA Updates | Yes | Yes | Yes | Yes |
| Cloud Integration | Yes | Yes | Yes | Yes |
| Actions (Scenes) | Basic | Advanced | Advanced | Advanced |
| Virtual Components | No | Yes | Yes | Yes |
| Input Events | Basic | Advanced | Advanced | Advanced |

### Library Package Usage

```go
import (
    "github.com/tj-smith47/shelly-go/gen1"      // Gen1 devices
    "github.com/tj-smith47/shelly-go/gen2"      // Gen2/Gen3/Gen4 devices
    "github.com/tj-smith47/shelly-go/factory"   // Auto-detection
    "github.com/tj-smith47/shelly-go/discovery" // Device discovery
)

// Auto-detect generation
device, _ := factory.FromAddress("192.168.1.100")

// Explicit Gen1
gen1Dev := gen1.NewDevice(transport)
gen1Dev.Relay(0).TurnOn(ctx)

// Explicit Gen2/Gen3/Gen4
gen2Dev := gen2.NewDevice(client)
sw := components.NewSwitch(client, 0)
sw.Set(ctx, &components.SwitchSetParams{On: true})
```

## Device Identification

### By mDNS Service Type

- Gen1: `_http._tcp` (standard HTTP)
- Gen2+: `_shelly._tcp` (Shelly-specific)

### By Device Info Response

```go
// Gen2+ devices
info, _ := device.Shelly().GetDeviceInfo(ctx)
fmt.Printf("Gen: %d, Model: %s, App: %s\n", info.Gen, info.Model, info.App)

// Example outputs:
// Gen: 2, Model: SNSW-001P16EU, App: Plus1PM
// Gen: 3, Model: S3SW-001P16EU, App: Plus1PMG3
// Gen: 4, Model: S4SW-001P16EU, App: Plus1PMG4
```

### Model Code Prefixes

| Prefix | Generation |
|--------|------------|
| SH* | Gen1 |
| SN* | Gen2 (Plus) |
| S3* | Gen3 |
| S4* | Gen4 |

## Resources

- [Gen1 API Documentation](https://shelly-api-docs.shelly.cloud/gen1/)
- [Gen2+ API Documentation](https://shelly-api-docs.shelly.cloud/gen2/)
- [Shelly Knowledge Base](https://kb.shelly.cloud/)
- [Shelly Support](https://support.shelly.cloud/)
