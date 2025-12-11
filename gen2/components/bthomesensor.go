package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

const componentTypeBTHomeSensor = "bthomesensor"

// BTHomeSensor represents a Shelly Gen2+ BTHomeSensor component.
//
// The BTHomeSensor component represents a single sensor/object from a BTHomeDevice.
// Each sensor is identified by its MAC address, BTHome object ID, and object index.
//
// BTHomeSensor component IDs range from 200-299 (dynamic components).
// A BTHomeDevice with the same MAC address must exist before creating a sensor.
//
// Example:
//
//	sensor := components.NewBTHomeSensor(client, 200)
//	status, err := sensor.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Sensor value: %v, last update: %v\n", status.Value, status.LastUpdateTS)
//	}
type BTHomeSensor struct {
	client *rpc.Client
	id     int
}

// NewBTHomeSensor creates a new BTHomeSensor component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component instance ID (200-299)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	sensor := components.NewBTHomeSensor(device.Client(), 200)
func NewBTHomeSensor(client *rpc.Client, id int) *BTHomeSensor {
	return &BTHomeSensor{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (s *BTHomeSensor) Client() *rpc.Client {
	return s.client
}

// ID returns the component instance ID.
func (s *BTHomeSensor) ID() int {
	return s.id
}

// BTHomeSensorConfig represents the configuration of a BTHomeSensor component.
type BTHomeSensorConfig struct {
	Name *string        `json:"name,omitempty"`
	Meta map[string]any `json:"meta,omitempty"`
	types.RawFields
	Addr  string `json:"addr"`
	ID    int    `json:"id"`
	ObjID int    `json:"obj_id"`
	Idx   int    `json:"idx"`
}

// BTHomeSensorStatus represents the status of a BTHomeSensor component.
type BTHomeSensorStatus struct {
	Value any `json:"value"`
	types.RawFields
	ID           int     `json:"id"`
	LastUpdateTS float64 `json:"last_updated_ts"`
}

// GetConfig retrieves the BTHomeSensor configuration.
//
// Example:
//
//	config, err := sensor.GetConfig(ctx)
//	if err == nil {
//	    fmt.Printf("Sensor name: %s, Object ID: %d\n", *config.Name, config.ObjID)
//	}
func (s *BTHomeSensor) GetConfig(ctx context.Context) (*BTHomeSensorConfig, error) {
	params := map[string]any{
		"id": s.id,
	}

	resultJSON, err := s.client.Call(ctx, "BTHomeSensor.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config BTHomeSensorConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// BTHomeSensorSetConfigRequest represents the configuration fields that can be updated.
type BTHomeSensorSetConfigRequest struct {
	// Name is the display name of the sensor.
	Name *string `json:"name,omitempty"`

	// ObjID is the BTHome object ID in decimal.
	// Can be changed to monitor a different object type.
	ObjID *int `json:"obj_id,omitempty"`

	// Meta stores component metadata.
	Meta map[string]any `json:"meta,omitempty"`
}

// SetConfig updates the BTHomeSensor configuration.
//
// Example:
//
//	err := sensor.SetConfig(ctx, &BTHomeSensorSetConfigRequest{
//	    Name: ptr("Door Status"),
//	})
func (s *BTHomeSensor) SetConfig(ctx context.Context, config *BTHomeSensorSetConfigRequest) error {
	params := map[string]any{
		"id":     s.id,
		"config": config,
	}

	_, err := s.client.Call(ctx, "BTHomeSensor.SetConfig", params)
	return err
}

// GetStatus retrieves the current BTHomeSensor status.
//
// The Value field can be a number, string, or boolean depending on the sensor type:
//   - Numeric sensors (temperature, humidity): float64
//   - Binary sensors (door, motion): bool
//   - Text sensors: string
//
// Example:
//
//	status, err := sensor.GetStatus(ctx)
//	if err == nil {
//	    switch v := status.Value.(type) {
//	    case float64:
//	        fmt.Printf("Numeric value: %.1f\n", v)
//	    case bool:
//	        fmt.Printf("Binary value: %v\n", v)
//	    case string:
//	        fmt.Printf("Text value: %s\n", v)
//	    }
//	}
func (s *BTHomeSensor) GetStatus(ctx context.Context) (*BTHomeSensorStatus, error) {
	params := map[string]any{
		"id": s.id,
	}

	resultJSON, err := s.client.Call(ctx, "BTHomeSensor.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status BTHomeSensorStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (s *BTHomeSensor) Type() string {
	return componentTypeBTHomeSensor
}

// Key returns the component key for aggregated status/config responses.
func (s *BTHomeSensor) Key() string {
	return componentTypeBTHomeSensor
}

// Ensure BTHomeSensor implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
	ID() int
} = (*BTHomeSensor)(nil)
