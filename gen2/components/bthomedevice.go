package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// BTHomeDevice represents a Shelly Gen2+ BTHomeDevice component.
//
// The BTHomeDevice component represents an individual physical Bluetooth device
// identified by its MAC address. It uses BTHomeDevice as RPC namespace and provides
// methods to configure the device and retrieve its status.
//
// BTHomeDevice component IDs range from 200-299 (dynamic components).
//
// Example:
//
//	device := components.NewBTHomeDevice(client, 200)
//	status, err := device.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("RSSI: %d dBm, Battery: %d%%\n", *status.RSSI, *status.Battery)
//	}
type BTHomeDevice struct {
	client *rpc.Client
	id     int
}

// NewBTHomeDevice creates a new BTHomeDevice component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component instance ID (200-299)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	btDevice := components.NewBTHomeDevice(device.Client(), 200)
func NewBTHomeDevice(client *rpc.Client, id int) *BTHomeDevice {
	return &BTHomeDevice{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (d *BTHomeDevice) Client() *rpc.Client {
	return d.client
}

// ID returns the component instance ID.
func (d *BTHomeDevice) ID() int {
	return d.id
}

// BTHomeDeviceConfig represents the configuration of a BTHomeDevice component.
type BTHomeDeviceConfig struct {
	Name *string        `json:"name,omitempty"`
	Key  *string        `json:"key,omitempty"`
	Meta map[string]any `json:"meta,omitempty"`
	types.RawFields
	Addr string `json:"addr"`
	ID   int    `json:"id"`
}

// BTHomeDeviceStatus represents the status of a BTHomeDevice component.
type BTHomeDeviceStatus struct {
	RSSI     *int `json:"rssi,omitempty"`
	Battery  *int `json:"battery,omitempty"`
	PacketID *int `json:"packet_id,omitempty"`
	types.RawFields
	Errors       []string `json:"errors,omitempty"`
	ID           int      `json:"id"`
	LastUpdateTS float64  `json:"last_updated_ts"`
}

// BTHomeDeviceKnownObject represents a known object from a BTHomeDevice.
type BTHomeDeviceKnownObject struct {
	Component *string `json:"component"`
	ObjID     int     `json:"obj_id"`
	Idx       int     `json:"idx"`
}

// BTHomeDeviceKnownObjectsResponse represents the response from GetKnownObjects.
type BTHomeDeviceKnownObjectsResponse struct {
	Objects []BTHomeDeviceKnownObject `json:"objects"`
	ID      int                       `json:"id"`
}

// GetConfig retrieves the BTHomeDevice configuration.
//
// Example:
//
//	config, err := btDevice.GetConfig(ctx)
//	if err == nil {
//	    fmt.Printf("Device name: %s, MAC: %s\n", *config.Name, config.Addr)
//	}
func (d *BTHomeDevice) GetConfig(ctx context.Context) (*BTHomeDeviceConfig, error) {
	params := map[string]any{
		"id": d.id,
	}

	resultJSON, err := d.client.Call(ctx, "BTHomeDevice.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config BTHomeDeviceConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// BTHomeDeviceSetConfigRequest represents the configuration fields that can be updated.
type BTHomeDeviceSetConfigRequest struct {
	// Name is the display name of the device.
	Name *string `json:"name,omitempty"`

	// Key is the AES encryption key as hexadecimal string for encrypted devices.
	Key *string `json:"key,omitempty"`

	// Meta stores component metadata.
	Meta map[string]any `json:"meta,omitempty"`
}

// SetConfig updates the BTHomeDevice configuration.
//
// Example:
//
//	err := btDevice.SetConfig(ctx, &BTHomeDeviceSetConfigRequest{
//	    Name: ptr("Living Room Temperature"),
//	})
func (d *BTHomeDevice) SetConfig(ctx context.Context, config *BTHomeDeviceSetConfigRequest) error {
	params := map[string]any{
		"id":     d.id,
		"config": config,
	}

	_, err := d.client.Call(ctx, "BTHomeDevice.SetConfig", params)
	return err
}

// GetStatus retrieves the current BTHomeDevice status.
//
// Example:
//
//	status, err := btDevice.GetStatus(ctx)
//	if err == nil {
//	    if status.RSSI != nil {
//	        fmt.Printf("Signal strength: %d dBm\n", *status.RSSI)
//	    }
//	    if status.Battery != nil {
//	        fmt.Printf("Battery: %d%%\n", *status.Battery)
//	    }
//	    if len(status.Errors) > 0 {
//	        fmt.Printf("Errors: %v\n", status.Errors)
//	    }
//	}
func (d *BTHomeDevice) GetStatus(ctx context.Context) (*BTHomeDeviceStatus, error) {
	params := map[string]any{
		"id": d.id,
	}

	resultJSON, err := d.client.Call(ctx, "BTHomeDevice.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status BTHomeDeviceStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// GetKnownObjects retrieves the list of known object IDs from the device's packets.
//
// This can be used to discover what sensors the device provides before adding
// BTHomeSensor components.
//
// Example:
//
//	resp, err := btDevice.GetKnownObjects(ctx)
//	if err == nil {
//	    for _, obj := range resp.Objects {
//	        fmt.Printf("Object ID: %d, Index: %d", obj.ObjID, obj.Idx)
//	        if obj.Component != nil {
//	            fmt.Printf(", Managed by: %s", *obj.Component)
//	        }
//	        fmt.Println()
//	    }
//	}
func (d *BTHomeDevice) GetKnownObjects(ctx context.Context) (*BTHomeDeviceKnownObjectsResponse, error) {
	params := map[string]any{
		"id": d.id,
	}

	resultJSON, err := d.client.Call(ctx, "BTHomeDevice.GetKnownObjects", params)
	if err != nil {
		return nil, err
	}

	var resp BTHomeDeviceKnownObjectsResponse
	if err := json.Unmarshal(resultJSON, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Type returns the component type identifier.
func (d *BTHomeDevice) Type() string {
	return "bthomedevice"
}

// Key returns the component key for aggregated status/config responses.
func (d *BTHomeDevice) Key() string {
	return "bthomedevice"
}

// Ensure BTHomeDevice implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
	ID() int
} = (*BTHomeDevice)(nil)
