package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// ethernetComponentType is the type identifier for the Ethernet component.
const ethernetComponentType = "eth"

// Ethernet represents a Shelly Gen2+ Ethernet component.
//
// Ethernet is available on Pro devices that have an Ethernet port.
// It provides wired network connectivity as an alternative to WiFi,
// offering more reliable and faster connections for fixed installations.
//
// Note: Ethernet component does not use component IDs.
// It is a singleton component accessed via "eth" key.
//
// Example:
//
//	eth := components.NewEthernet(device.Client())
//	status, err := eth.GetStatus(ctx)
//	if err == nil && status.IP != nil {
//	    fmt.Printf("Ethernet IP: %s\n", *status.IP)
//	}
type Ethernet struct {
	client *rpc.Client
}

// NewEthernet creates a new Ethernet component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	eth := components.NewEthernet(device.Client())
func NewEthernet(client *rpc.Client) *Ethernet {
	return &Ethernet{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (e *Ethernet) Client() *rpc.Client {
	return e.client
}

// EthernetConfig represents the configuration of the Ethernet component.
type EthernetConfig struct {
	// Enable enables or disables the Ethernet interface.
	Enable *bool `json:"enable,omitempty"`

	// IPv4Mode specifies how to obtain an IP address.
	// Values: "dhcp" (automatic) or "static" (manual configuration).
	IPv4Mode *string `json:"ipv4mode,omitempty"`

	// IP is the static IP address.
	// Only used when IPv4Mode is "static".
	IP *string `json:"ip,omitempty"`

	// Netmask is the network mask for static IP.
	// Only used when IPv4Mode is "static".
	Netmask *string `json:"netmask,omitempty"`

	// GW is the gateway address for static IP.
	// Only used when IPv4Mode is "static".
	GW *string `json:"gw,omitempty"`

	// Nameserver is the DNS server address for static IP.
	// Only used when IPv4Mode is "static".
	Nameserver *string `json:"nameserver,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// EthernetStatus represents the current status of the Ethernet component.
type EthernetStatus struct {
	// IP is the IP address of the device on the Ethernet network.
	// Null when not connected or Ethernet is disabled.
	IP *string `json:"ip,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// GetConfig retrieves the Ethernet configuration.
//
// Example:
//
//	config, err := eth.GetConfig(ctx)
//	if err == nil && config.Enable != nil && *config.Enable {
//	    fmt.Println("Ethernet is enabled")
//	}
func (e *Ethernet) GetConfig(ctx context.Context) (*EthernetConfig, error) {
	resultJSON, err := e.client.Call(ctx, "Eth.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config EthernetConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the Ethernet configuration.
//
// Only non-nil fields will be updated.
//
// Example - Enable DHCP:
//
//	err := eth.SetConfig(ctx, &EthernetConfig{
//	    Enable:   ptr(true),
//	    IPv4Mode: ptr("dhcp"),
//	})
//
// Example - Configure static IP:
//
//	err := eth.SetConfig(ctx, &EthernetConfig{
//	    Enable:     ptr(true),
//	    IPv4Mode:   ptr("static"),
//	    IP:         ptr("192.168.1.50"),
//	    Netmask:    ptr("255.255.255.0"),
//	    GW:         ptr("192.168.1.1"),
//	    Nameserver: ptr("8.8.8.8"),
//	})
func (e *Ethernet) SetConfig(ctx context.Context, config *EthernetConfig) error {
	params := map[string]any{
		"config": config,
	}

	_, err := e.client.Call(ctx, "Eth.SetConfig", params)
	return err
}

// GetStatus retrieves the current Ethernet status.
//
// Returns the IP address if connected via Ethernet.
//
// Example:
//
//	status, err := eth.GetStatus(ctx)
//	if err == nil {
//	    if status.IP != nil {
//	        fmt.Printf("Connected with IP: %s\n", *status.IP)
//	    } else {
//	        fmt.Println("Not connected via Ethernet")
//	    }
//	}
func (e *Ethernet) GetStatus(ctx context.Context) (*EthernetStatus, error) {
	resultJSON, err := e.client.Call(ctx, "Eth.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status EthernetStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (e *Ethernet) Type() string {
	return ethernetComponentType
}

// Key returns the component key for aggregated status/config responses.
func (e *Ethernet) Key() string {
	return ethernetComponentType
}

// Ensure Ethernet implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Ethernet)(nil)
