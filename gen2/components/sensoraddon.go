package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// sensoraddonComponentType is the type identifier for the SensorAddon component.
const sensoraddonComponentType = "sensoraddon"

// PeripheralType represents the type of sensor add-on peripheral.
type PeripheralType string

const (
	// PeripheralTypeDS18B20 is a Dallas DS18B20 1-Wire temperature sensor.
	PeripheralTypeDS18B20 PeripheralType = "ds18b20"

	// PeripheralTypeDHT22 is a DHT22 temperature and humidity sensor.
	PeripheralTypeDHT22 PeripheralType = "dht22"

	// PeripheralTypeDigitalIn is a digital input peripheral.
	PeripheralTypeDigitalIn PeripheralType = "digital_in"

	// PeripheralTypeAnalogIn is an analog input peripheral.
	PeripheralTypeAnalogIn PeripheralType = "analog_in"
)

// SensorAddon represents a Shelly Gen2+ Sensor Add-on management component.
//
// The SensorAddon component provides management for external sensors connected
// via the Shelly Sensor Add-on board. It supports DS18B20 temperature sensors,
// DHT22 temperature/humidity sensors, and digital/analog inputs.
//
// The add-on must be enabled via Sys.SetConfig with device.addon_type = "sensor".
// Changes to peripheral configuration require a device reboot to take effect.
//
// Supported devices: Plus1, Plus1PM, Plus2PM, PlusI4, Plus10V, PlusRGBWPM,
// Dimmer0110VPM G3, Shelly1G3, Shelly1PMG3, Shelly2PMG3, ShellyI4G3
//
// Example:
//
//	addon := components.NewSensorAddon(device.Client())
//	peripherals, err := addon.GetPeripherals(ctx)
//	if err == nil {
//	    for pType, components := range peripherals {
//	        fmt.Printf("%s: %d components\n", pType, len(components))
//	    }
//	}
type SensorAddon struct {
	client *rpc.Client
}

// NewSensorAddon creates a new SensorAddon component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	addon := components.NewSensorAddon(device.Client())
func NewSensorAddon(client *rpc.Client) *SensorAddon {
	return &SensorAddon{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (s *SensorAddon) Client() *rpc.Client {
	return s.client
}

// AddPeripheralAttrs represents the attributes for adding a peripheral.
type AddPeripheralAttrs struct {
	// CID is the component ID. Optional - if omitted, first available ID is used.
	CID *int `json:"cid,omitempty"`

	// Addr is the address of the DS18B20 sensor. Required for DS18B20 sensors.
	Addr *string `json:"addr,omitempty"`
}

// AddPeripheralResponse represents the response from AddPeripheral.
// The keys are component keys (e.g., "temperature:100", "input:100").
type AddPeripheralResponse map[string]map[string]any

// UpdatePeripheralAttrs represents the attributes for updating a peripheral.
type UpdatePeripheralAttrs struct {
	// Addr is the address of the DS18B20 sensor. Required for DS18B20 sensors.
	Addr string `json:"addr"`
}

// PeripheralInfo represents information about a linked peripheral.
type PeripheralInfo struct {
	// Addr is the address (for DS18B20 sensors).
	Addr *string `json:"addr,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// GetPeripheralsResponse represents the response from GetPeripherals.
// Keys are peripheral types (ds18b20, dht22, digital_in, analog_in).
// Values are maps of component keys to peripheral info.
type GetPeripheralsResponse map[PeripheralType]map[string]PeripheralInfo

// OneWireDevice represents a discovered OneWire device.
type OneWireDevice struct {
	Component *string `json:"component"`
	Type      string  `json:"type"`
	Addr      string  `json:"addr"`
}

// OneWireScanResponse represents the response from OneWireScan.
type OneWireScanResponse struct {
	// Devices is the list of discovered OneWire devices.
	Devices []OneWireDevice `json:"devices"`
}

// AddPeripheral links an add-on peripheral to a component instance.
//
// Changes require a device reboot to take effect.
//
// Parameters:
//   - peripheralType: Type of peripheral (ds18b20, dht22, digital_in, analog_in)
//   - attrs: Optional attributes (CID for component ID, Addr for DS18B20 address)
//
// Returns a map of created component keys to their info.
//
// Example - Add DS18B20 sensor:
//
//	resp, err := addon.AddPeripheral(ctx, PeripheralTypeDS18B20, &AddPeripheralAttrs{
//	    CID:  ptr(101),
//	    Addr: ptr("40:255:100:6:199:204:149:177"),
//	})
//	// Returns: {"temperature:101": {}}
//
// Example - Add DHT22 sensor (creates both temperature and humidity):
//
//	resp, err := addon.AddPeripheral(ctx, PeripheralTypeDHT22, nil)
//	// Returns: {"temperature:100": {}, "humidity:100": {}}
//
// Example - Add digital input:
//
//	resp, err := addon.AddPeripheral(ctx, PeripheralTypeDigitalIn, &AddPeripheralAttrs{
//	    CID: ptr(100),
//	})
//	// Returns: {"input:100": {}}
func (s *SensorAddon) AddPeripheral(
	ctx context.Context, peripheralType PeripheralType, attrs *AddPeripheralAttrs,
) (AddPeripheralResponse, error) {
	params := map[string]any{
		"type": peripheralType,
	}
	if attrs != nil {
		params["attrs"] = attrs
	}

	resultJSON, err := s.client.Call(ctx, "SensorAddon.AddPeripheral", params)
	if err != nil {
		return nil, err
	}

	var resp AddPeripheralResponse
	if err := json.Unmarshal(resultJSON, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// GetPeripherals returns the configured links between add-on peripherals and components.
//
// Example:
//
//	peripherals, err := addon.GetPeripherals(ctx)
//	if err == nil {
//	    for pType, components := range peripherals {
//	        for compKey, info := range components {
//	            fmt.Printf("%s -> %s", pType, compKey)
//	            if info.Addr != nil {
//	                fmt.Printf(" (addr: %s)", *info.Addr)
//	            }
//	            fmt.Println()
//	        }
//	    }
//	}
func (s *SensorAddon) GetPeripherals(ctx context.Context) (GetPeripheralsResponse, error) {
	resultJSON, err := s.client.Call(ctx, "SensorAddon.GetPeripherals", nil)
	if err != nil {
		return nil, err
	}

	var resp GetPeripheralsResponse
	if err := json.Unmarshal(resultJSON, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// RemovePeripheral removes a peripheral link from a component.
//
// Changes require a device reboot to take effect.
//
// Parameters:
//   - component: Component key (e.g., "temperature:100", "input:100")
//
// Example:
//
//	err := addon.RemovePeripheral(ctx, "temperature:100")
func (s *SensorAddon) RemovePeripheral(ctx context.Context, component string) error {
	params := map[string]any{
		"component": component,
	}

	_, err := s.client.Call(ctx, "SensorAddon.RemovePeripheral", params)
	return err
}

// UpdatePeripheral updates the configuration of an existing peripheral.
//
// Currently only DS18B20 peripherals have updateable attributes (address).
// Changes require a device reboot to take effect.
//
// Parameters:
//   - component: Component key (e.g., "temperature:100")
//   - attrs: Update attributes (currently only Addr for DS18B20)
//
// Example:
//
//	err := addon.UpdatePeripheral(ctx, "temperature:100", &UpdatePeripheralAttrs{
//	    Addr: "40:255:100:6:199:204:149:178",
//	})
func (s *SensorAddon) UpdatePeripheral(ctx context.Context, component string, attrs *UpdatePeripheralAttrs) error {
	params := map[string]any{
		"component": component,
		"attrs":     attrs,
	}

	_, err := s.client.Call(ctx, "SensorAddon.UpdatePeripheral", params)
	return err
}

// OneWireScan scans for OneWire devices on the bus.
//
// This method returns an error if a DHT22 peripheral is currently in use,
// as DHT22 occupies the same GPIOs used for OneWire.
//
// Currently only DS18B20 OneWire devices are supported.
//
// Example:
//
//	resp, err := addon.OneWireScan(ctx)
//	if err == nil {
//	    for _, device := range resp.Devices {
//	        fmt.Printf("Found %s at %s", device.Type, device.Addr)
//	        if device.Component != nil {
//	            fmt.Printf(" (linked to %s)", *device.Component)
//	        }
//	        fmt.Println()
//	    }
//	}
func (s *SensorAddon) OneWireScan(ctx context.Context) (*OneWireScanResponse, error) {
	resultJSON, err := s.client.Call(ctx, "SensorAddon.OneWireScan", nil)
	if err != nil {
		return nil, err
	}

	var resp OneWireScanResponse
	if err := json.Unmarshal(resultJSON, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Type returns the component type identifier.
func (s *SensorAddon) Type() string {
	return sensoraddonComponentType
}

// Key returns the component key for aggregated status/config responses.
func (s *SensorAddon) Key() string {
	return sensoraddonComponentType
}

// Ensure SensorAddon implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*SensorAddon)(nil)
