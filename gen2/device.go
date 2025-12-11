package gen2

import (
	"context"
	"fmt"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Device represents a Gen2+ Shelly device.
//
// Device provides access to all device components and the Shelly namespace.
// It implements the types.Device interface and provides type-safe access to
// components like Switch, Cover, Light, etc.
type Device struct {
	client *rpc.Client
	shelly *Shelly
	info   *DeviceInfo // Cached device info
}

// NewDevice creates a new Gen2+ device with the given RPC client.
//
// The client should be configured with the appropriate transport (HTTP,
// WebSocket, or MQTT) and authentication if required.
//
// Example:
//
//	// HTTP transport
//	httpTransport := transport.NewHTTP("http://192.168.1.100")
//	client := rpc.NewClient(httpTransport)
//	device := gen2.NewDevice(client)
func NewDevice(client *rpc.Client) *Device {
	return &Device{
		client: client,
		shelly: NewShelly(client),
	}
}

// Shelly returns the Shelly namespace handler for device-level operations.
//
// The Shelly namespace provides methods like GetDeviceInfo, Reboot,
// Update, etc.
//
// Example:
//
//	info, err := device.Shelly().GetDeviceInfo(ctx)
func (d *Device) Shelly() *Shelly {
	return d.shelly
}

// Client returns the underlying RPC client.
//
// This can be used for advanced operations or custom RPC calls.
func (d *Device) Client() *rpc.Client {
	return d.client
}

// GetDeviceInfo retrieves and caches device information.
//
// This is a convenience method that calls Shelly.GetDeviceInfo and caches
// the result. Subsequent calls return the cached value.
//
// To force a refresh, call device.Shelly().GetDeviceInfo() directly.
func (d *Device) GetDeviceInfo(ctx context.Context) (*DeviceInfo, error) {
	if d.info != nil {
		return d.info, nil
	}

	info, err := d.shelly.GetDeviceInfo(ctx)
	if err != nil {
		return nil, err
	}

	d.info = info
	return info, nil
}

// Generation returns the device generation.
//
// Returns types.Gen2, types.Gen3, or types.Gen4 based on device info.
func (d *Device) Generation(ctx context.Context) (types.Generation, error) {
	info, err := d.GetDeviceInfo(ctx)
	if err != nil {
		return types.GenUnknown, err
	}

	switch info.Gen {
	case 2:
		// Determine if Plus or Pro based on model
		// Pro devices have "Pro" in the model name
		if len(info.Model) > 3 && info.Model[:3] == "SPR" {
			return types.Gen2Pro, nil
		}
		return types.Gen2Plus, nil
	case 3:
		return types.Gen3, nil
	case 4:
		return types.Gen4, nil
	default:
		return types.GenUnknown, fmt.Errorf("unknown generation: %d", info.Gen)
	}
}

// Component creates a generic component accessor.
//
// This is useful when you need to access a component by type and ID
// without using the type-specific methods.
//
// Example:
//
//	comp := device.Component("switch", 0)
//	status, err := comp.GetStatus(ctx)
func (d *Device) Component(typ string, id int) *BaseComponent {
	return NewBaseComponent(d.client, typ, id)
}

// Close closes the underlying transport connection.
//
// For stateless transports (HTTP), this may be a no-op.
// For stateful transports (WebSocket, MQTT), this will disconnect.
func (d *Device) Close() error {
	return d.client.Close()
}

// Switch returns a Switch component accessor.
//
// Parameters:
//   - id: Component ID (usually 0 for single-switch devices)
//
// Example:
//
//	sw := device.Switch(0)
//	err := sw.Set(ctx, true)  // Turn on
func (d *Device) Switch(id int) Component {
	return NewBaseComponent(d.client, "switch", id)
}

// Cover returns a Cover component accessor.
//
// Parameters:
//   - id: Component ID (usually 0 for single-cover devices)
//
// Example:
//
//	cover := device.Cover(0)
//	err := cover.Open(ctx)
func (d *Device) Cover(id int) Component {
	return NewBaseComponent(d.client, "cover", id)
}

// Light returns a Light component accessor.
//
// Parameters:
//   - id: Component ID (usually 0 for single-light devices)
//
// Example:
//
//	light := device.Light(0)
//	err := light.Set(ctx, true, 50)  // Turn on at 50% brightness
func (d *Device) Light(id int) Component {
	return NewBaseComponent(d.client, "light", id)
}

// Input returns an Input component accessor.
//
// Parameters:
//   - id: Component ID
//
// Example:
//
//	input := device.Input(0)
//	status, err := input.GetStatus(ctx)
func (d *Device) Input(id int) Component {
	return NewBaseComponent(d.client, "input", id)
}

// DevicePower returns a DevicePower component accessor.
//
// DevicePower is used for battery-powered devices to monitor battery status.
//
// Parameters:
//   - id: Component ID (usually 0)
func (d *Device) DevicePower(id int) Component {
	return NewBaseComponent(d.client, "devicepower", id)
}

// PM returns a Power Meter component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) PM(id int) Component {
	return NewBaseComponent(d.client, "pm", id)
}

// PM1 returns a PM1 component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) PM1(id int) Component {
	return NewBaseComponent(d.client, "pm1", id)
}

// EM returns an Energy Monitor component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) EM(id int) Component {
	return NewBaseComponent(d.client, "em", id)
}

// EM1 returns an EM1 component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) EM1(id int) Component {
	return NewBaseComponent(d.client, "em1", id)
}

// Voltmeter returns a Voltmeter component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) Voltmeter(id int) Component {
	return NewBaseComponent(d.client, "voltmeter", id)
}

// Temperature returns a Temperature sensor component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) Temperature(id int) Component {
	return NewBaseComponent(d.client, "temperature", id)
}

// Humidity returns a Humidity sensor component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) Humidity(id int) Component {
	return NewBaseComponent(d.client, "humidity", id)
}

// Smoke returns a Smoke detector component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) Smoke(id int) Component {
	return NewBaseComponent(d.client, "smoke", id)
}

// Thermostat returns a Thermostat component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) Thermostat(id int) Component {
	return NewBaseComponent(d.client, "thermostat", id)
}

// Script returns a Script component accessor.
//
// Parameters:
//   - id: Script ID
func (d *Device) Script(id int) Component {
	return NewBaseComponent(d.client, "script", id)
}

// Schedule returns a Schedule component accessor.
//
// Parameters:
//   - id: Schedule ID
func (d *Device) Schedule(id int) Component {
	return NewBaseComponent(d.client, "schedule", id)
}

// Webhook returns a Webhook component accessor.
//
// Parameters:
//   - id: Webhook ID
func (d *Device) Webhook(id int) Component {
	return NewBaseComponent(d.client, "webhook", id)
}

// WiFi returns a WiFi component accessor.
//
// Parameters:
//   - id: Component ID (usually 0)
func (d *Device) WiFi(id int) Component {
	return NewBaseComponent(d.client, "wifi", id)
}

// Ethernet returns an Ethernet component accessor.
//
// Parameters:
//   - id: Component ID (usually 0)
func (d *Device) Ethernet(id int) Component {
	return NewBaseComponent(d.client, "eth", id)
}

// BLE returns a BLE component accessor.
//
// Parameters:
//   - id: Component ID (usually 0)
func (d *Device) BLE(id int) Component {
	return NewBaseComponent(d.client, "ble", id)
}

// Cloud returns a Cloud component accessor.
//
// Parameters:
//   - id: Component ID (usually 0)
func (d *Device) Cloud(id int) Component {
	return NewBaseComponent(d.client, "cloud", id)
}

// MQTT returns an MQTT component accessor.
//
// Parameters:
//   - id: Component ID (usually 0)
func (d *Device) MQTT(id int) Component {
	return NewBaseComponent(d.client, "mqtt", id)
}

// WS returns an Outbound WebSocket component accessor.
//
// Parameters:
//   - id: Component ID (usually 0)
func (d *Device) WS(id int) Component {
	return NewBaseComponent(d.client, "ws", id)
}

// Sys returns a System component accessor.
//
// Parameters:
//   - id: Component ID (usually 0)
func (d *Device) Sys(id int) Component {
	return NewBaseComponent(d.client, "sys", id)
}

// UI returns a UI component accessor.
//
// Parameters:
//   - id: Component ID (usually 0)
func (d *Device) UI(id int) Component {
	return NewBaseComponent(d.client, "ui", id)
}

// KVS returns a Key-Value Storage accessor.
//
// KVS allows storing arbitrary key-value pairs on the device.
func (d *Device) KVS() Component {
	// KVS doesn't have an ID
	return NewBaseComponent(d.client, "kvs", 0)
}

// BTHome returns a BTHome component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) BTHome(id int) Component {
	return NewBaseComponent(d.client, "bthome", id)
}

// BTHomeDevice returns a BTHomeDevice component accessor.
//
// Parameters:
//   - id: Device ID
func (d *Device) BTHomeDevice(id int) Component {
	return NewBaseComponent(d.client, "bthomedevice", id)
}

// BTHomeSensor returns a BTHomeSensor component accessor.
//
// Parameters:
//   - id: Sensor ID
func (d *Device) BTHomeSensor(id int) Component {
	return NewBaseComponent(d.client, "bthomesensor", id)
}

// RGB returns an RGB component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) RGB(id int) Component {
	return NewBaseComponent(d.client, "rgb", id)
}

// RGBW returns an RGBW component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) RGBW(id int) Component {
	return NewBaseComponent(d.client, "rgbw", id)
}

// ModBus returns a ModBus component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) ModBus(id int) Component {
	return NewBaseComponent(d.client, "modbus", id)
}

// SensorAddon returns a Sensor Add-on component accessor.
//
// Parameters:
//   - id: Component ID
func (d *Device) SensorAddon(id int) Component {
	return NewBaseComponent(d.client, "sensoraddon", id)
}
