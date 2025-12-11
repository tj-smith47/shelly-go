package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// scheduleComponentType is the type identifier for the Schedule component.
const scheduleComponentType = "schedule"

// Schedule represents a Shelly Gen2+ Schedule component.
//
// Schedule provides time-based automation by executing RPC calls at
// specified times. Schedules use cron-like timespec expressions for
// flexible timing control.
//
// Limits:
//   - Maximum 20 schedules per device
//
// Timespec format:
//   - Similar to cron: "ss mm hh DD WW" (seconds, minutes, hours, day of month, weekday)
//   - Supports wildcards (*), ranges (1-5), lists (1,3,5), and steps (0-59/10)
//   - Special values: @sunrise, @sunset with optional offset (+/-minutes)
//
// Example:
//
//	schedule := components.NewSchedule(device.Client())
//	schedules, err := schedule.List(ctx)
//	if err == nil {
//	    for _, s := range schedules.Jobs {
//	        fmt.Printf("Schedule %d: %s\n", s.ID, s.Timespec)
//	    }
//	}
type Schedule struct {
	client *rpc.Client
}

// NewSchedule creates a new Schedule component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	schedule := components.NewSchedule(device.Client())
func NewSchedule(client *rpc.Client) *Schedule {
	return &Schedule{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (s *Schedule) Client() *rpc.Client {
	return s.client
}

// ScheduleCall represents an RPC call to execute.
type ScheduleCall struct {
	Params any `json:"params,omitempty"`
	types.RawFields
	Method string `json:"method"`
}

// ScheduleJob represents a scheduled job.
type ScheduleJob struct {
	types.RawFields
	Timespec string         `json:"timespec"`
	Calls    []ScheduleCall `json:"calls"`
	ID       int            `json:"id"`
	Enable   bool           `json:"enable"`
}

// ScheduleListResponse represents the response from Schedule.List.
type ScheduleListResponse struct {
	types.RawFields
	Jobs []ScheduleJob `json:"jobs"`
	Rev  int           `json:"rev,omitempty"`
}

// ScheduleCreateRequest represents the parameters for creating a schedule.
type ScheduleCreateRequest struct {
	Timespec string         `json:"timespec"`
	Calls    []ScheduleCall `json:"calls"`
	Enable   bool           `json:"enable"`
}

// ScheduleCreateResponse represents the response from Schedule.Create.
type ScheduleCreateResponse struct {
	types.RawFields
	ID  int `json:"id"`
	Rev int `json:"rev"`
}

// ScheduleUpdateRequest represents the parameters for updating a schedule.
type ScheduleUpdateRequest struct {
	Enable   *bool          `json:"enable,omitempty"`
	Timespec *string        `json:"timespec,omitempty"`
	Calls    []ScheduleCall `json:"calls,omitempty"`
	ID       int            `json:"id"`
}

// ScheduleUpdateResponse represents the response from Schedule.Update.
type ScheduleUpdateResponse struct {
	types.RawFields
	Rev int `json:"rev"`
}

// ScheduleDeleteResponse represents the response from Schedule.Delete.
type ScheduleDeleteResponse struct {
	types.RawFields
	Rev int `json:"rev"`
}

// List retrieves all scheduled jobs.
//
// Example:
//
//	result, err := schedule.List(ctx)
//	if err == nil {
//	    for _, job := range result.Jobs {
//	        fmt.Printf("Job %d: %s (enabled: %t)\n", job.ID, job.Timespec, job.Enable)
//	    }
//	}
func (s *Schedule) List(ctx context.Context) (*ScheduleListResponse, error) {
	resultJSON, err := s.client.Call(ctx, "Schedule.List", nil)
	if err != nil {
		return nil, err
	}

	var result ScheduleListResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Create creates a new scheduled job.
//
// Parameters:
//   - enable: Whether the schedule is enabled
//   - timespec: Cron-like time specification
//   - calls: List of RPC calls to execute
//
// Example - Turn on switch at 8:00 AM every day:
//
//	result, err := schedule.Create(ctx, &ScheduleCreateRequest{
//	    Enable:   true,
//	    Timespec: "0 0 8 * *",
//	    Calls: []ScheduleCall{
//	        {Method: "Switch.Set", Params: map[string]any{"id": 0, "on": true}},
//	    },
//	})
//
// Example - Turn off at sunset:
//
//	result, err := schedule.Create(ctx, &ScheduleCreateRequest{
//	    Enable:   true,
//	    Timespec: "@sunset",
//	    Calls: []ScheduleCall{
//	        {Method: "Switch.Set", Params: map[string]any{"id": 0, "on": false}},
//	    },
//	})
func (s *Schedule) Create(ctx context.Context, req *ScheduleCreateRequest) (*ScheduleCreateResponse, error) {
	params := map[string]any{
		"enable":   req.Enable,
		"timespec": req.Timespec,
		"calls":    req.Calls,
	}

	resultJSON, err := s.client.Call(ctx, "Schedule.Create", params)
	if err != nil {
		return nil, err
	}

	var result ScheduleCreateResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Update updates an existing scheduled job.
//
// Parameters:
//   - req: Update request with schedule ID and fields to update
//
// Example - Disable a schedule:
//
//	_, err := schedule.Update(ctx, &ScheduleUpdateRequest{
//	    ID:     1,
//	    Enable: ptr(false),
//	})
//
// Example - Change schedule time:
//
//	_, err := schedule.Update(ctx, &ScheduleUpdateRequest{
//	    ID:       1,
//	    Timespec: ptr("0 0 9 * *"),
//	})
func (s *Schedule) Update(ctx context.Context, req *ScheduleUpdateRequest) (*ScheduleUpdateResponse, error) {
	params := map[string]any{
		"id": req.ID,
	}

	if req.Enable != nil {
		params["enable"] = *req.Enable
	}
	if req.Timespec != nil {
		params["timespec"] = *req.Timespec
	}
	if req.Calls != nil {
		params["calls"] = req.Calls
	}

	resultJSON, err := s.client.Call(ctx, "Schedule.Update", params)
	if err != nil {
		return nil, err
	}

	var result ScheduleUpdateResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete deletes a scheduled job.
//
// Parameters:
//   - id: Schedule ID to delete
//
// Example:
//
//	_, err := schedule.Delete(ctx, 1)
func (s *Schedule) Delete(ctx context.Context, id int) (*ScheduleDeleteResponse, error) {
	params := map[string]any{
		"id": id,
	}

	resultJSON, err := s.client.Call(ctx, "Schedule.Delete", params)
	if err != nil {
		return nil, err
	}

	var result ScheduleDeleteResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteAll deletes all scheduled jobs.
//
// Example:
//
//	_, err := schedule.DeleteAll(ctx)
func (s *Schedule) DeleteAll(ctx context.Context) (*ScheduleDeleteResponse, error) {
	resultJSON, err := s.client.Call(ctx, "Schedule.DeleteAll", nil)
	if err != nil {
		return nil, err
	}

	var result ScheduleDeleteResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Type returns the component type identifier.
func (s *Schedule) Type() string {
	return scheduleComponentType
}

// Key returns the component key for aggregated status/config responses.
func (s *Schedule) Key() string {
	return scheduleComponentType
}

// Ensure Schedule implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Schedule)(nil)
