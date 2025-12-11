# Architecture Overview

This document describes the internal architecture of shelly-go.

## Package Structure

```
shelly-go/
├── types/          # Common types and errors
├── transport/      # Network transport layer
├── rpc/            # RPC client (JSON-RPC 2.0)
├── gen1/           # Gen1 device support
├── gen2/           # Gen2+ device support
│   └── components/ # Type-safe component implementations
├── discovery/      # Device discovery (mDNS, CoIoT)
├── factory/        # Device creation utilities
├── helpers/        # High-level utilities
├── events/         # Event bus and handlers
├── cloud/          # Shelly Cloud API client
└── examples/       # Usage examples
```

## Layer Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Application Layer                    │
│  (Your Code, examples/, helpers/, events/)              │
└────────────────────────────┬────────────────────────────┘
                             │
┌────────────────────────────┼────────────────────────────┐
│                      Device Layer                       │
│  ┌─────────────┐    ┌──────┴───────┐    ┌───────────┐   │
│  │   gen1/     │    │    gen2/     │    │  cloud/   │   │
│  │   Device    │    │    Device    │    │  Client   │   │
│  │   Relay     │    │   Components │    │           │   │
│  │   Light     │    │   Switch     │    │           │   │
│  │   Roller    │    │   Cover      │    │           │   │
│  └──────┬──────┘    │   Light      │    └─────┬─────┘   │
│         │           │   Input      │          │         │
│         │           └──────┬───────┘          │         │
└─────────┼──────────────────┼──────────────────┼─────────┘
          │                  │                  │
┌─────────┼──────────────────┼──────────────────┼────────┐
│         │           Protocol Layer            │        │
│  ┌──────┴──────┐    ┌──────┴──────┐     ┌─────┴─────┐  │
│  │   HTTP      │    │    RPC      │     │   HTTP    │  │
│  │   REST      │    │  JSON-RPC   │     │   REST    │  │
│  └──────┬──────┘    └──────┬──────┘     └─────┬─────┘  │
└─────────┼──────────────────┼──────────────────┼────────┘
          │                  │                  │
┌─────────┴──────────────────┴──────────────────┴────────┐
│                    Transport Layer                     │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐ │
│  │    HTTP     │  │   WebSocket  │  │     MQTT       │ │
│  │  (http.go)  │  │   (ws.go)    │  │   (mqtt.go)    │ │
│  └─────────────┘  └──────────────┘  └────────────────┘ │
└────────────────────────────────────────────────────────┘
```

## Core Packages

### types/

Common types, interfaces, and errors used across all packages.

```go
// Core interfaces
type Transport interface {
    Call(ctx context.Context, method string, params any) (json.RawMessage, error)
    Close() error
}

// Generation identifiers
type Generation int
const (
    Gen1      Generation = 1
    Gen2      Generation = 2
    Gen2Plus  Generation = 2
    Gen2Pro   Generation = 2
    Gen3      Generation = 3
    Gen4      Generation = 4
)

// Common errors
var (
    ErrAuth            = errors.New("authentication failed")
    ErrTimeout         = errors.New("request timeout")
    ErrNotFound        = errors.New("resource not found")
    ErrInvalidResponse = errors.New("invalid response")
)
```

### transport/

Network transports implementing the `Transport` interface.

**HTTP Transport:**
- Supports both Gen1 REST and Gen2+ RPC
- Configurable timeouts, retries, and authentication
- Connection pooling via standard `http.Client`

**WebSocket Transport:**
- Persistent connection for real-time notifications
- Automatic reconnection with exponential backoff
- Message multiplexing for concurrent requests

**MQTT Transport:**
- Pub/sub model for IoT scenarios
- Topic-based routing
- QoS levels support

### rpc/

JSON-RPC 2.0 client for Gen2+ devices.

```go
// RPC client wraps transport with JSON-RPC protocol
type Client struct {
    transport Transport
    auth      *AuthData  // Optional digest auth
}

// Call executes an RPC method
func (c *Client) Call(ctx context.Context, method string, params any) (json.RawMessage, error)

// Batch executes multiple RPC calls
func (c *Client) Batch() *BatchRequest
```

### gen1/

Gen1 device support using REST API.

```go
// Device provides access to Gen1 components
type Device struct {
    transport Transport
}

// Components
func (d *Device) Relay(id int) *Relay
func (d *Device) Roller(id int) *Roller
func (d *Device) Light(id int) *Light

// Relay operations
type Relay struct { ... }
func (r *Relay) TurnOn(ctx context.Context) error
func (r *Relay) TurnOff(ctx context.Context) error
func (r *Relay) Toggle(ctx context.Context) error
func (r *Relay) GetStatus(ctx context.Context) (*RelayStatus, error)
```

### gen2/

Gen2+ device support using JSON-RPC 2.0.

```go
// Device provides access to Gen2+ components
type Device struct {
    client *rpc.Client
    shelly *Shelly  // Shelly namespace
}

// Component access
func (d *Device) Switch(id int) Component
func (d *Device) Cover(id int) Component
func (d *Device) Light(id int) Component
func (d *Device) Input(id int) Component

// Shelly namespace
type Shelly struct { ... }
func (s *Shelly) GetDeviceInfo(ctx context.Context) (*DeviceInfo, error)
func (s *Shelly) Reboot(ctx context.Context) error
func (s *Shelly) Update(ctx context.Context) error
```

### gen2/components/

Type-safe component implementations.

```go
// Each component has:
// - Status struct (current state)
// - Config struct (configuration)
// - Action methods (control)

type Switch struct {
    *gen2.BaseComponent
}

func (s *Switch) Set(ctx context.Context, params *SwitchSetParams) (*SwitchSetResult, error)
func (s *Switch) Toggle(ctx context.Context) (*SwitchToggleResult, error)
func (s *Switch) GetStatus(ctx context.Context) (*SwitchStatus, error)
func (s *Switch) GetConfig(ctx context.Context) (*SwitchConfig, error)
func (s *Switch) SetConfig(ctx context.Context, config *SwitchConfig) error
```

### discovery/

Device discovery mechanisms.

```go
// mDNS discovery for Gen2+ devices
type MDNSDiscoverer struct { ... }
func (m *MDNSDiscoverer) Discover(timeout time.Duration) ([]DiscoveredDevice, error)

// CoIoT discovery for Gen1 devices
type CoIoTDiscoverer struct { ... }
func (c *CoIoTDiscoverer) Discover(timeout time.Duration) ([]DiscoveredDevice, error)

// Device identification
func Identify(ctx context.Context, address string) (*DeviceInfo, error)
```

### factory/

Device creation utilities with auto-detection.

```go
// Create device from address with auto-detection
func FromAddress(address string, opts ...Option) (Device, error)

// Create device from discovery result
func FromDiscovery(d DiscoveredDevice, opts ...Option) (Device, error)

// Batch creation
func BatchFromAddresses(addresses []string, opts ...Option) ([]Device, []error)
```

### helpers/

High-level utilities for common operations.

```go
// Batch operations
func AllOn(ctx context.Context, devices []Device) BatchResults
func AllOff(ctx context.Context, devices []Device) BatchResults
func BatchToggle(ctx context.Context, devices []Device) BatchResults

// Device groups
type Group struct { ... }
func NewGroup(name string, opts ...GroupOption) *Group
func (g *Group) AllOn(ctx context.Context) BatchResults

// Scene management
type Scene struct { ... }
func NewScene(name string) *Scene
func (s *Scene) AddAction(device Device, action Action)
func (s *Scene) Activate(ctx context.Context) SceneResults
```

### events/

Event bus for handling device notifications.

```go
// Event bus
type EventBus struct { ... }
func NewEventBus() *EventBus
func (b *EventBus) Subscribe(handler func(Event))
func (b *EventBus) SubscribeFiltered(filter Filter, handler func(Event))
func (b *EventBus) Publish(event Event)

// Event types
type StatusChangeEvent struct { ... }
type NotifyEvent struct { ... }
type DeviceOnlineEvent struct { ... }
type DeviceOfflineEvent struct { ... }
```

### cloud/

Shelly Cloud API client.

```go
// Cloud client
type Client struct { ... }
func NewClient(opts ...ClientOption) *Client
func (c *Client) GetAllDevices(ctx context.Context) (map[string]*DeviceStatus, error)
func (c *Client) SetSwitch(ctx context.Context, deviceID string, channel int, on bool) error

// WebSocket for real-time events
type WebSocket struct { ... }
func (ws *WebSocket) OnDeviceOnline(handler func(deviceID string))
func (ws *WebSocket) OnStatusChange(handler func(deviceID string, status json.RawMessage))
```

## Design Principles

### 1. Generation Abstraction

The library abstracts away generation-specific details while still providing full access to generation-specific features when needed.

```go
// Abstract: Factory creates appropriate type
device, _ := factory.FromAddress("192.168.1.100")

// Specific: Direct access when needed
gen2Dev := device.(*factory.Gen2Device)
info, _ := gen2Dev.Shelly().GetDeviceInfo(ctx)
```

### 2. Component-Based Architecture

Gen2+ devices use a component-based model that mirrors the actual device architecture.

```go
device.Switch(0)      // Access switch component
device.Cover(0)       // Access cover component
device.Input(0)       // Access input component
```

### 3. Forward Compatibility

RawFields embedded in structs capture unknown fields from new firmware versions.

```go
type SwitchStatus struct {
    ID     int  `json:"id"`
    Output bool `json:"output"`
    // ... known fields ...

    types.RawFields  // Captures unknown fields
}

// Access unknown fields
rawFields := status.GetRawFields()
newFeature, ok := rawFields["new_feature"]
```

### 4. Context-Based Cancellation

All operations accept a context for cancellation and timeout.

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

status, err := sw.GetStatus(ctx)
```

### 5. Functional Options

Configuration uses functional options for flexibility.

```go
transport := transport.NewHTTP(url,
    transport.WithTimeout(30*time.Second),
    transport.WithAuth(user, pass),
    transport.WithRetry(3, time.Second),
)
```

## Thread Safety

- All transport implementations are thread-safe
- RPC client uses atomic operations for request IDs
- Event bus uses mutex for subscriber management
- Device/component methods are safe for concurrent use

## Error Handling

Errors are wrapped with context using standard `errors` package:

```go
if errors.Is(err, types.ErrAuth) {
    // Handle authentication error
}

if errors.Is(err, types.ErrTimeout) {
    // Handle timeout
}

// Unwrap for full error chain
fmt.Printf("Error: %v\n", err)
```

## Testing

The library uses interfaces to enable testing:

```go
// Mock transport for unit tests
type MockTransport struct {
    responses map[string]json.RawMessage
}

func (m *MockTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
    return m.responses[method], nil
}

// Use in tests
transport := &MockTransport{
    responses: map[string]json.RawMessage{
        "Switch.GetStatus": json.RawMessage(`{"id":0,"output":true}`),
    },
}
client := rpc.NewClient(transport)
```

## Extension Points

1. **Custom Transports**: Implement `Transport` interface
2. **Custom Components**: Embed `BaseComponent` struct
3. **Event Handlers**: Register with `EventBus`
4. **WebSocket Dialers**: Implement `WebSocketDialer` interface
