package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// floodComponentType is the type identifier for the Flood component.
const floodComponentType = "flood"

// Flood represents a Shelly Gen2+ Flood sensor component.
//
// The Flood component handles monitoring of flood sensors. It provides
// water leak detection with configurable alarm modes.
//
// Note: Flood component uses numeric IDs (flood:0, flood:1, etc.).
//
// Webhook Events:
//   - flood.alarm - produced when flood alarm is triggered
//   - flood.alarm_off - produced when flood alarm goes off
//   - flood.cable_unplugged - produced when cable for flood detection is unplugged
//
// Example:
//
//	flood := components.NewFlood(device.Client(), 0)
//	status, err := flood.GetStatus(ctx)
//	if err == nil && status.Alarm {
//	    fmt.Println("WATER LEAK DETECTED!")
//	}
type Flood struct {
	client *rpc.Client
	id     int
}

// NewFlood creates a new Flood component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (0-based)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	flood := components.NewFlood(device.Client(), 0)
func NewFlood(client *rpc.Client, id int) *Flood {
	return &Flood{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (f *Flood) Client() *rpc.Client {
	return f.client
}

// ID returns the component ID.
func (f *Flood) ID() int {
	return f.id
}

// FloodAlarmMode represents the alarm sound configuration mode.
type FloodAlarmMode string

const (
	// FloodAlarmModeDisabled disables alarm sound.
	FloodAlarmModeDisabled FloodAlarmMode = "disabled"
	// FloodAlarmModeNormal sets normal alarm sound.
	FloodAlarmModeNormal FloodAlarmMode = "normal"
	// FloodAlarmModeIntense sets intense (louder) alarm sound.
	FloodAlarmModeIntense FloodAlarmMode = "intense"
	// FloodAlarmModeRain sets rain detection mode alarm sound.
	FloodAlarmModeRain FloodAlarmMode = "rain"
)

// FloodConfig represents the configuration of a Flood component.
type FloodConfig struct {
	Name          *string         `json:"name,omitempty"`
	AlarmMode     *FloodAlarmMode `json:"alarm_mode,omitempty"`
	ReportHoldoff *int            `json:"report_holdoff,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// FloodStatus represents the status of a Flood component.
type FloodStatus struct {
	types.RawFields
	Errors []string `json:"errors,omitempty"`
	ID     int      `json:"id"`
	Alarm  bool     `json:"alarm"`
	Mute   bool     `json:"mute"`
}

// GetConfig retrieves the Flood configuration.
//
// Example:
//
//	config, err := flood.GetConfig(ctx)
//	if err == nil && config.Name != nil {
//	    fmt.Printf("Flood sensor name: %s\n", *config.Name)
//	}
func (f *Flood) GetConfig(ctx context.Context) (*FloodConfig, error) {
	params := map[string]any{
		"id": f.id,
	}

	resultJSON, err := f.client.Call(ctx, "Flood.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config FloodConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the Flood configuration.
//
// Only non-nil fields will be updated.
//
// Example - Set sensor name:
//
//	name := "Bathroom Flood Sensor"
//	err := flood.SetConfig(ctx, &FloodConfig{
//	    Name: &name,
//	})
//
// Example - Set alarm mode:
//
//	mode := FloodAlarmModeIntense
//	err := flood.SetConfig(ctx, &FloodConfig{
//	    AlarmMode: &mode,
//	})
//
// Example - Set report holdoff:
//
//	holdoff := 10 // 10 seconds
//	err := flood.SetConfig(ctx, &FloodConfig{
//	    ReportHoldoff: &holdoff,
//	})
func (f *Flood) SetConfig(ctx context.Context, config *FloodConfig) error {
	configMap := map[string]any{
		"id": f.id,
	}
	params := map[string]any{
		"id":     f.id,
		"config": configMap,
	}

	if config.Name != nil {
		configMap["name"] = *config.Name
	}
	if config.AlarmMode != nil {
		configMap["alarm_mode"] = string(*config.AlarmMode)
	}
	if config.ReportHoldoff != nil {
		configMap["report_holdoff"] = *config.ReportHoldoff
	}

	_, err := f.client.Call(ctx, "Flood.SetConfig", params)
	return err
}

// GetStatus retrieves the current Flood status.
//
// Returns the current alarm and mute state.
//
// Example:
//
//	status, err := flood.GetStatus(ctx)
//	if err == nil {
//	    if status.Alarm {
//	        fmt.Println("WATER LEAK DETECTED!")
//	        if status.Mute {
//	            fmt.Println("(Alarm is muted)")
//	        }
//	    } else {
//	        fmt.Println("All clear - no water detected")
//	    }
//	    if len(status.Errors) > 0 {
//	        fmt.Printf("Errors: %v\n", status.Errors)
//	    }
//	}
func (f *Flood) GetStatus(ctx context.Context) (*FloodStatus, error) {
	params := map[string]any{
		"id": f.id,
	}

	resultJSON, err := f.client.Call(ctx, "Flood.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status FloodStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (f *Flood) Type() string {
	return floodComponentType
}

// Key returns the component key for aggregated status/config responses.
func (f *Flood) Key() string {
	return floodComponentType
}

// Ensure Flood implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Flood)(nil)
