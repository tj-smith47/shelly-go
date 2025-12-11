package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// modbusComponentType is the type identifier for the Modbus component.
const modbusComponentType = "modbus"

// Modbus represents a Shelly Gen2+ Modbus-TCP component.
//
// The Modbus component provides Modbus-TCP communication protocol on TCP port 502
// for supported Shelly devices. This allows integration with industrial automation
// systems and SCADA software that support the Modbus protocol.
//
// Modbus registers are exposed per-component (EM, EMData, Switch, Input, etc.)
// and documented in each component's section.
//
// Device info registers (available when enabled):
//   - 30000: Device MAC (6 registers / 12 bytes)
//   - 30006: Device model (10 registers / 20 bytes)
//   - 30016: Device name (32 registers / 64 bytes)
//
// Example:
//
//	modbus := components.NewModbus(device.Client())
//	status, err := modbus.GetStatus(ctx)
//	if err == nil {
//	    if status.Enabled {
//	        fmt.Println("Modbus-TCP is enabled on port 502")
//	    }
//	}
type Modbus struct {
	client *rpc.Client
}

// NewModbus creates a new Modbus component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	modbus := components.NewModbus(device.Client())
func NewModbus(client *rpc.Client) *Modbus {
	return &Modbus{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (m *Modbus) Client() *rpc.Client {
	return m.client
}

// ModbusConfig represents the configuration of a Modbus component.
type ModbusConfig struct {
	types.RawFields
	Enable bool `json:"enable"`
}

// ModbusStatus represents the status of a Modbus component.
type ModbusStatus struct {
	types.RawFields
	Enabled bool `json:"enabled"`
}

// GetConfig retrieves the Modbus configuration.
//
// Example:
//
//	config, err := modbus.GetConfig(ctx)
//	if err == nil {
//	    fmt.Printf("Modbus enabled: %v\n", config.Enable)
//	}
func (m *Modbus) GetConfig(ctx context.Context) (*ModbusConfig, error) {
	resultJSON, err := m.client.Call(ctx, "Modbus.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config ModbusConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the Modbus configuration.
//
// Example - Enable Modbus:
//
//	err := modbus.SetConfig(ctx, &ModbusConfig{
//	    Enable: true,
//	})
//
// Example - Disable Modbus:
//
//	err := modbus.SetConfig(ctx, &ModbusConfig{
//	    Enable: false,
//	})
func (m *Modbus) SetConfig(ctx context.Context, config *ModbusConfig) error {
	params := map[string]any{
		"config": map[string]any{
			"enable": config.Enable,
		},
	}

	_, err := m.client.Call(ctx, "Modbus.SetConfig", params)
	return err
}

// GetStatus retrieves the current Modbus status.
//
// Example:
//
//	status, err := modbus.GetStatus(ctx)
//	if err == nil {
//	    if status.Enabled {
//	        fmt.Println("Modbus-TCP server is running on port 502")
//	    } else {
//	        fmt.Println("Modbus-TCP server is disabled")
//	    }
//	}
func (m *Modbus) GetStatus(ctx context.Context) (*ModbusStatus, error) {
	resultJSON, err := m.client.Call(ctx, "Modbus.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status ModbusStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (m *Modbus) Type() string {
	return modbusComponentType
}

// Key returns the component key for aggregated status/config responses.
func (m *Modbus) Key() string {
	return modbusComponentType
}

// Ensure Modbus implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Modbus)(nil)
