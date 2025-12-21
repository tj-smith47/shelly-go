package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewSchedule(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	schedule := NewSchedule(client)

	if schedule == nil {
		t.Fatal("NewSchedule returned nil")
	}

	if schedule.Type() != "schedule" {
		t.Errorf("Type() = %q, want %q", schedule.Type(), "schedule")
	}

	if schedule.Key() != "schedule" {
		t.Errorf("Key() = %q, want %q", schedule.Key(), "schedule")
	}

	if schedule.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestSchedule_List(t *testing.T) {
	tests := []struct {
		name      string
		result    string
		wantCount int
		wantRev   int
	}{
		{
			name: "multiple schedules",
			result: `{
				"jobs": [
					{"id": 1, "enable": true, "timespec": "0 0 8 * *", "calls": [{"method": "Switch.Set", "params": {"id": 0, "on": true}}]},
					{"id": 2, "enable": false, "timespec": "0 30 22 * *", "calls": [{"method": "Switch.Set", "params": {"id": 0, "on": false}}]}
				],
				"rev": 5
			}`,
			wantCount: 2,
			wantRev:   5,
		},
		{
			name:      "no schedules",
			result:    `{"jobs": [], "rev": 0}`,
			wantCount: 0,
			wantRev:   0,
		},
		{
			name: "single schedule with multiple calls",
			result: `{
				"jobs": [
					{
						"id": 1,
						"enable": true,
						"timespec": "@sunrise+30",
						"calls": [
							{"method": "Switch.Set", "params": {"id": 0, "on": true}},
							{"method": "Light.Set", "params": {"id": 0, "brightness": 100}}
						]
					}
				],
				"rev": 3
			}`,
			wantCount: 1,
			wantRev:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Schedule.List" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			schedule := NewSchedule(client)

			result, err := schedule.List(context.Background())
			if err != nil {
				t.Errorf("List() error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("List() returned nil result")
			}

			if len(result.Jobs) != tt.wantCount {
				t.Errorf("len(Jobs) = %d, want %d", len(result.Jobs), tt.wantCount)
			}

			if result.Rev != tt.wantRev {
				t.Errorf("Rev = %d, want %d", result.Rev, tt.wantRev)
			}
		})
	}
}

func TestSchedule_List_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	schedule := NewSchedule(client)
	testComponentError(t, "List", func() error {
		_, err := schedule.List(context.Background())
		return err
	})
}

func TestSchedule_List_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	schedule := NewSchedule(client)
	testComponentInvalidJSON(t, "List", func() error {
		_, err := schedule.List(context.Background())
		return err
	})
}

func TestSchedule_Create(t *testing.T) {
	tests := []struct {
		req     *ScheduleCreateRequest
		name    string
		wantID  int
		wantRev int
	}{
		{
			name: "basic schedule",
			req: &ScheduleCreateRequest{
				Enable:   true,
				Timespec: "0 0 8 * *",
				Calls: []ScheduleCall{
					{Method: "Switch.Set", Params: map[string]any{"id": 0, "on": true}},
				},
			},
			wantID:  1,
			wantRev: 1,
		},
		{
			name: "sunset schedule",
			req: &ScheduleCreateRequest{
				Enable:   true,
				Timespec: "@sunset",
				Calls: []ScheduleCall{
					{Method: "Switch.Set", Params: map[string]any{"id": 0, "on": false}},
				},
			},
			wantID:  2,
			wantRev: 2,
		},
		{
			name: "disabled schedule",
			req: &ScheduleCreateRequest{
				Enable:   false,
				Timespec: "0 30 7 * 1-5",
				Calls: []ScheduleCall{
					{Method: "Switch.Set", Params: map[string]any{"id": 0, "on": true}},
				},
			},
			wantID:  3,
			wantRev: 3,
		},
		{
			name: "multiple calls",
			req: &ScheduleCreateRequest{
				Enable:   true,
				Timespec: "0 0 0 * *",
				Calls: []ScheduleCall{
					{Method: "Switch.Set", Params: map[string]any{"id": 0, "on": true}},
					{Method: "Switch.Set", Params: map[string]any{"id": 1, "on": true}},
				},
			},
			wantID:  4,
			wantRev: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Schedule.Create" {
						t.Errorf("method = %q, want %q", method, "Schedule.Create")
					}
					return jsonrpcResponse(`{"id": ` + string(rune('0'+tt.wantID)) + `, "rev": ` + string(rune('0'+tt.wantRev)) + `}`)
				},
			}
			client := rpc.NewClient(tr)
			schedule := NewSchedule(client)

			result, err := schedule.Create(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if result.ID != tt.wantID {
				t.Errorf("result.ID = %d, want %d", result.ID, tt.wantID)
			}

			if result.Rev != tt.wantRev {
				t.Errorf("result.Rev = %d, want %d", result.Rev, tt.wantRev)
			}
		})
	}
}

func TestSchedule_Create_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	schedule := NewSchedule(client)
	testComponentError(t, "Create", func() error {
		_, err := schedule.Create(context.Background(), &ScheduleCreateRequest{
			Enable:   true,
			Timespec: "0 0 8 * *",
			Calls:    []ScheduleCall{{Method: "Switch.Set"}},
		})
		return err
	})
}

func TestSchedule_Update(t *testing.T) {
	tests := []struct {
		req     *ScheduleUpdateRequest
		name    string
		wantRev int
	}{
		{
			name: "disable schedule",
			req: &ScheduleUpdateRequest{
				ID:     1,
				Enable: ptr(false),
			},
			wantRev: 6,
		},
		{
			name: "update timespec",
			req: &ScheduleUpdateRequest{
				ID:       1,
				Timespec: ptr("0 0 9 * *"),
			},
			wantRev: 7,
		},
		{
			name: "update calls",
			req: &ScheduleUpdateRequest{
				ID: 1,
				Calls: []ScheduleCall{
					{Method: "Switch.Toggle", Params: map[string]any{"id": 0}},
				},
			},
			wantRev: 8,
		},
		{
			name: "update multiple fields",
			req: &ScheduleUpdateRequest{
				ID:       1,
				Enable:   ptr(true),
				Timespec: ptr("@sunrise-15"),
				Calls: []ScheduleCall{
					{Method: "Switch.Set", Params: map[string]any{"id": 0, "on": true}},
				},
			},
			wantRev: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "Schedule.Update" {
						t.Errorf("method = %q, want %q", method, "Schedule.Update")
					}
					return jsonrpcResponse(`{"rev": ` + string(rune('0'+tt.wantRev)) + `}`)
				},
			}
			client := rpc.NewClient(tr)
			schedule := NewSchedule(client)

			result, err := schedule.Update(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			if result.Rev != tt.wantRev {
				t.Errorf("result.Rev = %d, want %d", result.Rev, tt.wantRev)
			}
		})
	}
}

func TestSchedule_Update_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	schedule := NewSchedule(client)
	testComponentError(t, "Update", func() error {
		_, err := schedule.Update(context.Background(), &ScheduleUpdateRequest{ID: 1})
		return err
	})
}

func TestSchedule_Delete(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Schedule.Delete" {
				t.Errorf("method = %q, want %q", method, "Schedule.Delete")
			}
			return jsonrpcResponse(`{"rev": 10}`)
		},
	}
	client := rpc.NewClient(tr)
	schedule := NewSchedule(client)

	result, err := schedule.Delete(context.Background(), 5)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if result.Rev != 10 {
		t.Errorf("result.Rev = %d, want 10", result.Rev)
	}
}

func TestSchedule_Delete_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	schedule := NewSchedule(client)
	testComponentError(t, "Delete", func() error {
		_, err := schedule.Delete(context.Background(), 1)
		return err
	})
}

func TestSchedule_DeleteAll(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "Schedule.DeleteAll" {
				t.Errorf("method = %q, want %q", method, "Schedule.DeleteAll")
			}
			return jsonrpcResponse(`{"rev": 0}`)
		},
	}
	client := rpc.NewClient(tr)
	schedule := NewSchedule(client)

	result, err := schedule.DeleteAll(context.Background())
	if err != nil {
		t.Fatalf("DeleteAll() error = %v", err)
	}

	if result.Rev != 0 {
		t.Errorf("result.Rev = %d, want 0", result.Rev)
	}
}

func TestSchedule_DeleteAll_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	schedule := NewSchedule(client)
	testComponentError(t, "DeleteAll", func() error {
		_, err := schedule.DeleteAll(context.Background())
		return err
	})
}

func TestScheduleJob_JSONSerialization(t *testing.T) {
	tests := []struct {
		job   ScheduleJob
		check func(t *testing.T, data map[string]any)
		name  string
	}{
		{
			name: "full job",
			job: ScheduleJob{
				ID:       1,
				Enable:   true,
				Timespec: "0 0 8 * *",
				Calls: []ScheduleCall{
					{Method: "Switch.Set", Params: map[string]any{"id": 0, "on": true}},
				},
			},
			check: func(t *testing.T, data map[string]any) {
				id, ok := data["id"].(float64)
				if !ok || id != 1 {
					t.Errorf("id = %v, want 1", data["id"])
				}
				enable, ok := data["enable"].(bool)
				if !ok || enable != true {
					t.Errorf("enable = %v, want true", data["enable"])
				}
				timespec, ok := data["timespec"].(string)
				if !ok || timespec != "0 0 8 * *" {
					t.Errorf("timespec = %v, want 0 0 8 * *", data["timespec"])
				}
				calls, ok := data["calls"].([]any)
				if !ok || len(calls) != 1 {
					t.Errorf("len(calls) = %d, want 1", len(calls))
				}
			},
		},
		{
			name: "disabled job",
			job: ScheduleJob{
				ID:       2,
				Enable:   false,
				Timespec: "@sunset",
				Calls:    []ScheduleCall{},
			},
			check: func(t *testing.T, data map[string]any) {
				enable, ok := data["enable"].(bool)
				if !ok || enable != false {
					t.Errorf("enable = %v, want false", data["enable"])
				}
				timespec, ok := data["timespec"].(string)
				if !ok || timespec != "@sunset" {
					t.Errorf("timespec = %v, want @sunset", data["timespec"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.job)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			tt.check(t, parsed)
		})
	}
}

func TestScheduleCall_JSONSerialization(t *testing.T) {
	tests := []struct {
		call  ScheduleCall
		check func(t *testing.T, data map[string]any)
		name  string
	}{
		{
			name: "call with params",
			call: ScheduleCall{
				Method: "Switch.Set",
				Params: map[string]any{"id": 0, "on": true},
			},
			check: func(t *testing.T, data map[string]any) {
				method, ok := data["method"].(string)
				if !ok || method != "Switch.Set" {
					t.Errorf("method = %v, want Switch.Set", data["method"])
				}
				params, ok := data["params"].(map[string]any)
				if !ok {
					t.Fatalf("params type assertion failed")
				}
				on, ok := params["on"].(bool)
				if !ok || on != true {
					t.Errorf("params.on = %v, want true", params["on"])
				}
			},
		},
		{
			name: "call without params",
			call: ScheduleCall{
				Method: "Shelly.Reboot",
			},
			check: func(t *testing.T, data map[string]any) {
				method, ok := data["method"].(string)
				if !ok || method != "Shelly.Reboot" {
					t.Errorf("method = %v, want Shelly.Reboot", data["method"])
				}
				if _, ok := data["params"]; ok {
					t.Error("params should not be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.call)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			tt.check(t, parsed)
		})
	}
}

func TestSchedule_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"jobs": [], "rev": 0}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	schedule := NewSchedule(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := schedule.List(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
