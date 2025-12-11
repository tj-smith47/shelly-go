package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// BTHome represents a Shelly Gen2+ BTHome management component.
//
// The BTHome component provides management for Bluetooth devices that emit data
// in BTHome format. It allows adding/removing BTHomeDevice and BTHomeSensor
// components, discovering nearby BTHome devices, and querying object information.
//
// Available only for Gen 2 Pro*, Gen 3, and Gen 4 devices in Matter mode.
// Bluetooth must be enabled on the device.
//
// Example:
//
//	bthome := components.NewBTHome(device.Client())
//	status, err := bthome.GetStatus(ctx)
//	if err == nil {
//	    if status.Discovery != nil {
//	        fmt.Println("Discovery in progress...")
//	    }
//	}
type BTHome struct {
	client *rpc.Client
}

// NewBTHome creates a new BTHome component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	bthome := components.NewBTHome(device.Client())
func NewBTHome(client *rpc.Client) *BTHome {
	return &BTHome{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (b *BTHome) Client() *rpc.Client {
	return b.client
}

// BTHomeStatus represents the status of the BTHome component.
type BTHomeStatus struct {
	Discovery *BTHomeDiscoveryStatus `json:"discovery,omitempty"`
	types.RawFields
	Errors []string `json:"errors,omitempty"`
}

// BTHomeDiscoveryStatus represents the status of an ongoing BTHome device discovery.
type BTHomeDiscoveryStatus struct {
	// StartedAt is the Unix timestamp when the scan started.
	StartedAt float64 `json:"started_at"`

	// Duration is the duration of the scan process in seconds.
	Duration int `json:"duration"`
}

// BTHomeAddDeviceConfig represents the configuration for adding a new BTHome device.
type BTHomeAddDeviceConfig struct {
	Name *string        `json:"name,omitempty"`
	Key  *string        `json:"key,omitempty"`
	Meta map[string]any `json:"meta,omitempty"`
	Addr string         `json:"addr"`
}

// BTHomeAddDeviceResponse represents the response from adding a BTHome device.
type BTHomeAddDeviceResponse struct {
	// Key is the component key (format: "bthomedevice:<id>", e.g., "bthomedevice:200").
	Key string `json:"key"`
}

// BTHomeAddSensorConfig represents the configuration for adding a new BTHome sensor.
type BTHomeAddSensorConfig struct {
	Name  *string        `json:"name,omitempty"`
	Meta  map[string]any `json:"meta,omitempty"`
	Addr  string         `json:"addr"`
	ObjID int            `json:"obj_id"`
	Idx   int            `json:"idx"`
}

// BTHomeAddSensorResponse represents the response from adding a BTHome sensor.
type BTHomeAddSensorResponse struct {
	// Key is the component key (format: "bthomesensor:<id>", e.g., "bthomesensor:200").
	Key string `json:"key"`
}

// BTHomeObjectInfo represents information about a BTHome object type.
type BTHomeObjectInfo struct {
	types.RawFields
	Name  string `json:"name"`
	Type  string `json:"type"`
	Unit  string `json:"unit,omitempty"`
	ObjID int    `json:"obj_id"`
}

// BTHomeGetObjectInfosResponse represents the response from GetObjectInfos.
type BTHomeGetObjectInfosResponse struct {
	// Infos contains information about the requested object types.
	Infos []BTHomeObjectInfo `json:"infos"`

	// Offset is the next offset for pagination, or -1 if no more results.
	Offset int `json:"offset"`
}

// GetStatus retrieves the current BTHome component status.
//
// Returns discovery status when a scan is in progress, and any error conditions.
//
// Example:
//
//	status, err := bthome.GetStatus(ctx)
//	if err == nil {
//	    if status.Discovery != nil {
//	        fmt.Printf("Discovery started at %v, duration: %ds\n",
//	            status.Discovery.StartedAt, status.Discovery.Duration)
//	    }
//	    if len(status.Errors) > 0 {
//	        fmt.Printf("Errors: %v\n", status.Errors)
//	    }
//	}
func (b *BTHome) GetStatus(ctx context.Context) (*BTHomeStatus, error) {
	resultJSON, err := b.client.Call(ctx, "BTHome.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status BTHomeStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// AddDevice creates a new BTHomeDevice component.
//
// Parameters:
//   - config: Device configuration including MAC address
//   - id: Optional component ID (200-299). If nil, first available ID is used.
//
// Returns the component key of the newly created device.
//
// Example:
//
//	resp, err := bthome.AddDevice(ctx, &BTHomeAddDeviceConfig{
//	    Addr: "3c:2e:f5:71:d5:2a",
//	    Name: ptr("Living Room Sensor"),
//	}, nil)
//	if err == nil {
//	    fmt.Printf("Created device: %s\n", resp.Key) // "bthomedevice:200"
//	}
func (b *BTHome) AddDevice(
	ctx context.Context, config *BTHomeAddDeviceConfig, id *int,
) (*BTHomeAddDeviceResponse, error) {
	params := map[string]any{
		"config": config,
	}
	if id != nil {
		params["id"] = *id
	}

	resultJSON, err := b.client.Call(ctx, "BTHome.AddDevice", params)
	if err != nil {
		return nil, err
	}

	var resp BTHomeAddDeviceResponse
	if err := json.Unmarshal(resultJSON, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteDevice removes an existing BTHomeDevice component.
//
// Parameters:
//   - id: Component ID of the device to delete
//
// Example:
//
//	err := bthome.DeleteDevice(ctx, 200)
func (b *BTHome) DeleteDevice(ctx context.Context, id int) error {
	params := map[string]any{
		"id": id,
	}

	_, err := b.client.Call(ctx, "BTHome.DeleteDevice", params)
	return err
}

// AddSensor creates a new BTHomeSensor component.
//
// A BTHomeDevice with the same MAC address must exist before adding a sensor.
//
// Parameters:
//   - config: Sensor configuration including MAC address and object type
//   - id: Optional component ID (200-299). If nil, first available ID is used.
//
// Returns the component key of the newly created sensor.
//
// Example:
//
//	resp, err := bthome.AddSensor(ctx, &BTHomeAddSensorConfig{
//	    Addr:  "3c:2e:f5:71:d5:2a",
//	    ObjID: 45, // Door status
//	    Idx:   0,
//	    Name:  ptr("Front Door"),
//	}, nil)
//	if err == nil {
//	    fmt.Printf("Created sensor: %s\n", resp.Key) // "bthomesensor:200"
//	}
func (b *BTHome) AddSensor(
	ctx context.Context, config *BTHomeAddSensorConfig, id *int,
) (*BTHomeAddSensorResponse, error) {
	params := map[string]any{
		"config": config,
	}
	if id != nil {
		params["id"] = *id
	}

	resultJSON, err := b.client.Call(ctx, "BTHome.AddSensor", params)
	if err != nil {
		return nil, err
	}

	var resp BTHomeAddSensorResponse
	if err := json.Unmarshal(resultJSON, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteSensor removes an existing BTHomeSensor component.
//
// Parameters:
//   - id: Component ID of the sensor to delete
//
// Example:
//
//	err := bthome.DeleteSensor(ctx, 200)
func (b *BTHome) DeleteSensor(ctx context.Context, id int) error {
	params := map[string]any{
		"id": id,
	}

	_, err := b.client.Call(ctx, "BTHome.DeleteSensor", params)
	return err
}

// StartDeviceDiscovery starts an active scan for BTHome devices.
//
// During scanning, device_discovered events are emitted for each found device.
// When complete, a discovery_done event is dispatched.
//
// Parameters:
//   - duration: Scan duration in seconds. If nil, defaults to 30 seconds.
//
// Example:
//
//	// Start 60 second discovery
//	duration := 60
//	err := bthome.StartDeviceDiscovery(ctx, &duration)
//
//	// Or use default 30 second duration
//	err := bthome.StartDeviceDiscovery(ctx, nil)
func (b *BTHome) StartDeviceDiscovery(ctx context.Context, duration *int) error {
	var params map[string]any
	if duration != nil {
		params = map[string]any{
			"duration": *duration,
		}
	}

	_, err := b.client.Call(ctx, "BTHome.StartDeviceDiscovery", params)
	return err
}

// GetObjectInfos retrieves information about BTHome object types.
//
// Supports pagination for large result sets.
//
// Parameters:
//   - objIDs: List of BTHome object IDs to query
//   - offset: Optional pagination offset
//
// Example:
//
//	// Get info for temperature (0x02) and humidity (0x03) objects
//	resp, err := bthome.GetObjectInfos(ctx, []int{2, 3}, nil)
//	if err == nil {
//	    for _, info := range resp.Infos {
//	        fmt.Printf("%s (%s): %s\n", info.Name, info.Type, info.Unit)
//	    }
//	}
func (b *BTHome) GetObjectInfos(ctx context.Context, objIDs []int, offset *int) (*BTHomeGetObjectInfosResponse, error) {
	params := map[string]any{
		"obj_ids": objIDs,
	}
	if offset != nil {
		params["offset"] = *offset
	}

	resultJSON, err := b.client.Call(ctx, "BTHome.GetObjectInfos", params)
	if err != nil {
		return nil, err
	}

	var resp BTHomeGetObjectInfosResponse
	if err := json.Unmarshal(resultJSON, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Type returns the component type identifier.
func (b *BTHome) Type() string {
	return "bthome"
}

// Key returns the component key for aggregated status/config responses.
func (b *BTHome) Key() string {
	return "bthome"
}

// Ensure BTHome implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*BTHome)(nil)
