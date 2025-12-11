# Migration Guide

This guide helps you migrate from other Shelly libraries or direct API usage to shelly-go.

## Table of Contents

- [From Direct HTTP API](#from-direct-http-api)
- [Gen1 to Gen2+ Differences](#gen1-to-gen2-differences)
- [Breaking Changes](#breaking-changes)
- [Version Migration](#version-migration)

## From Direct HTTP API

### Gen1 Devices (REST API)

If you were using direct HTTP calls for Gen1 devices:

**Before (direct HTTP):**
```go
// Turn on relay
resp, err := http.Get("http://192.168.1.100/relay/0?turn=on")
```

**After (shelly-go):**
```go
import (
    "github.com/tj-smith47/shelly-go/factory"
    "github.com/tj-smith47/shelly-go/types"
)

device, _ := factory.FromAddress("192.168.1.100",
    factory.WithGeneration(types.Gen1))

gen1Dev := device.(*factory.Gen1Device)
err := gen1Dev.Relay(0).TurnOn(ctx)
```

### Gen2+ Devices (RPC API)

If you were using direct JSON-RPC calls:

**Before (direct HTTP):**
```go
body := `{"id":1,"method":"Switch.Set","params":{"id":0,"on":true}}`
resp, err := http.Post("http://192.168.1.100/rpc", "application/json",
    strings.NewReader(body))
```

**After (shelly-go):**
```go
import (
    "github.com/tj-smith47/shelly-go/gen2"
    "github.com/tj-smith47/shelly-go/gen2/components"
    "github.com/tj-smith47/shelly-go/rpc"
    "github.com/tj-smith47/shelly-go/transport"
)

httpTransport := transport.NewHTTP("http://192.168.1.100")
client := rpc.NewClient(httpTransport)
defer client.Close()

device := gen2.NewDevice(client)
sw := components.NewSwitch(client, 0)
result, err := sw.Set(ctx, &components.SwitchSetParams{On: true})
```

## Gen1 to Gen2+ Differences

### Component Naming

| Gen1 | Gen2+ | Description |
|------|-------|-------------|
| `relay` | `switch` | On/off control |
| `roller` | `cover` | Blinds/shutters |
| `light` | `light` | Same name |
| `emeter` | `em` | Energy meter |
| `temperature` | `temperature` | Same name |

### Method Patterns

**Gen1 (REST path-based):**
```
GET /relay/0?turn=on
GET /roller/0?go=open
GET /settings/relay/0?name=Kitchen
```

**Gen2+ (JSON-RPC method-based):**
```
POST /rpc: {"method":"Switch.Set","params":{"id":0,"on":true}}
POST /rpc: {"method":"Cover.Open","params":{"id":0}}
POST /rpc: {"method":"Switch.SetConfig","params":{"id":0,"name":"Kitchen"}}
```

### Status Structure

**Gen1:**
```json
{
  "relays": [{"ison": true, "has_timer": false, "timer_remaining": 0}],
  "meters": [{"power": 45.2, "total": 12345}]
}
```

**Gen2+:**
```json
{
  "switch:0": {
    "output": true,
    "apower": 45.2,
    "aenergy": {"total": 12345}
  }
}
```

### Authentication

**Gen1:** HTTP Basic Authentication
```go
transport.WithAuth("admin", "password") // Basic auth
```

**Gen2+:** Digest Authentication with realm
```go
auth := &rpc.AuthData{
    Realm:    "shelly",
    Username: "admin",
    Password: "password",
}
client := rpc.NewClientWithAuth(transport, auth)
```

## Breaking Changes

### v1.0.0

Initial release - no breaking changes.

## Version Migration

### Migrating from v0.x to v1.x

If you were using pre-release versions:

1. **Import paths changed:**
   ```go
   // Old (hypothetical)
   import "github.com/tj-smith47/shelly/device"

   // New
   import "github.com/tj-smith47/shelly-go/gen2"
   ```

2. **Factory pattern introduced:**
   ```go
   // Old: Direct device creation
   device := gen2.NewDevice(client)

   // New: Can use factory for auto-detection
   device, _ := factory.FromAddress("192.168.1.100")
   ```

3. **Component types are now separate:**
   ```go
   // Old: Components embedded in device
   device.Switch(0).Set(ctx, true)

   // New: Type-safe component objects
   sw := components.NewSwitch(client, 0)
   sw.Set(ctx, &components.SwitchSetParams{On: true})
   ```

## Code Patterns

### Handling Both Generations

```go
import (
    "github.com/tj-smith47/shelly-go/factory"
    "github.com/tj-smith47/shelly-go/types"
)

device, err := factory.FromAddress(address)
if err != nil {
    return err
}

switch d := device.(type) {
case *factory.Gen1Device:
    // Use Gen1 API
    d.Relay(0).TurnOn(ctx)

case *factory.Gen2Device:
    // Use Gen2+ API
    sw := d.Device.Switch(0)
    // ...
}
```

### Using Device Groups

```go
import "github.com/tj-smith47/shelly-go/helpers"

// Create a group
group := helpers.NewGroup("Living Room",
    helpers.WithDevice(light1),
    helpers.WithDevice(light2),
)

// Operate on all devices
group.AllOff(ctx)
```

### Using Scenes

```go
import "github.com/tj-smith47/shelly-go/helpers"

scene := helpers.NewScene("Movie Night")
scene.AddAction(mainLight, helpers.ActionSet(false))
scene.AddAction(tvBacklight, helpers.ActionSetBrightness(30))

// Activate
results := scene.Activate(ctx)

// Persist to JSON
data, _ := scene.ToJSON()
```

## Common Migration Issues

### Issue: "No such method"
- Gen2+ uses different method names
- Check the component type (Switch vs Relay)

### Issue: "Authentication failed"
- Gen1 uses Basic auth, Gen2+ uses Digest auth
- Ensure proper auth configuration for device generation

### Issue: "Component not found"
- Gen2+ components are indexed by ID
- Verify the component exists on your device

### Issue: "Invalid response"
- Gen1 returns flat JSON, Gen2+ returns nested
- Use the appropriate type for your device generation

## Getting Help

- [API Documentation](https://shelly-api-docs.shelly.cloud/)
- [GitHub Issues](https://github.com/tj-smith47/shelly-go/issues)
- [Examples](./examples/)
