package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// smokeComponentType is the type identifier for the Smoke component.
const smokeComponentType = "smoke"

// Smoke represents a Shelly Gen2+ Smoke detector component.
//
// The Smoke component handles monitoring of smoke sensors. It provides
// alarm state detection and the ability to mute active alarms.
//
// Note: Smoke component uses numeric IDs (smoke:0, smoke:1, etc.).
//
// Webhook Events:
//   - smoke.alarm - produced when smoke alarm is triggered
//   - smoke.alarm_off - produced when smoke alarm goes off
//   - smoke.alarm_test - produced when alarm test is invoked
//
// Example:
//
//	smoke := components.NewSmoke(device.Client(), 0)
//	status, err := smoke.GetStatus(ctx)
//	if err == nil && status.Alarm {
//	    fmt.Println("SMOKE ALARM ACTIVE!")
//	    // Optionally mute the alarm
//	    smoke.Mute(ctx)
//	}
type Smoke struct {
	client *rpc.Client
	id     int
}

// NewSmoke creates a new Smoke component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (0-based)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	smoke := components.NewSmoke(device.Client(), 0)
func NewSmoke(client *rpc.Client, id int) *Smoke {
	return &Smoke{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (s *Smoke) Client() *rpc.Client {
	return s.client
}

// ID returns the component ID.
func (s *Smoke) ID() int {
	return s.id
}

// SmokeConfig represents the configuration of a Smoke component.
type SmokeConfig struct {
	Name *string `json:"name,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// SmokeStatus represents the status of a Smoke component.
type SmokeStatus struct {
	types.RawFields
	ID    int  `json:"id"`
	Alarm bool `json:"alarm"`
	Mute  bool `json:"mute"`
}

// GetConfig retrieves the Smoke configuration.
//
// Example:
//
//	config, err := smoke.GetConfig(ctx)
//	if err == nil && config.Name != nil {
//	    fmt.Printf("Smoke detector name: %s\n", *config.Name)
//	}
func (s *Smoke) GetConfig(ctx context.Context) (*SmokeConfig, error) {
	params := map[string]any{
		"id": s.id,
	}

	resultJSON, err := s.client.Call(ctx, "Smoke.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config SmokeConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the Smoke configuration.
//
// Only non-nil fields will be updated.
//
// Example - Set detector name:
//
//	name := "Kitchen Smoke Detector"
//	err := smoke.SetConfig(ctx, &SmokeConfig{
//	    Name: &name,
//	})
func (s *Smoke) SetConfig(ctx context.Context, config *SmokeConfig) error {
	// Build params, including the ID
	configMap := map[string]any{
		"id": s.id,
	}
	params := map[string]any{
		"id":     s.id,
		"config": configMap,
	}

	if config.Name != nil {
		configMap["name"] = *config.Name
	}

	_, err := s.client.Call(ctx, "Smoke.SetConfig", params)
	return err
}

// GetStatus retrieves the current Smoke status.
//
// Returns the current alarm and mute state.
//
// Example:
//
//	status, err := smoke.GetStatus(ctx)
//	if err == nil {
//	    if status.Alarm {
//	        fmt.Println("SMOKE DETECTED!")
//	        if status.Mute {
//	            fmt.Println("(Alarm is muted)")
//	        }
//	    } else {
//	        fmt.Println("All clear - no smoke detected")
//	    }
//	}
func (s *Smoke) GetStatus(ctx context.Context) (*SmokeStatus, error) {
	params := map[string]any{
		"id": s.id,
	}

	resultJSON, err := s.client.Call(ctx, "Smoke.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status SmokeStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Mute silences the alarm of the associated smoke sensor.
//
// This method mutes an active alarm. The alarm will remain muted until
// the smoke condition clears and potentially re-triggers.
//
// Example:
//
//	// Check if alarm is active and mute it
//	status, err := smoke.GetStatus(ctx)
//	if err == nil && status.Alarm && !status.Mute {
//	    if err := smoke.Mute(ctx); err != nil {
//	        fmt.Printf("Failed to mute alarm: %v\n", err)
//	    } else {
//	        fmt.Println("Alarm muted successfully")
//	    }
//	}
func (s *Smoke) Mute(ctx context.Context) error {
	params := map[string]any{
		"id": s.id,
	}

	_, err := s.client.Call(ctx, "Smoke.Mute", params)
	return err
}

// Type returns the component type identifier.
func (s *Smoke) Type() string {
	return smokeComponentType
}

// Key returns the component key for aggregated status/config responses.
func (s *Smoke) Key() string {
	return smokeComponentType
}

// Ensure Smoke implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Smoke)(nil)
