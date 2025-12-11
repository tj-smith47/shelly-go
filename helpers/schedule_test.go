package helpers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/factory"
	"github.com/tj-smith47/shelly-go/types"
)

// TestWeekday tests weekday operations.
func TestWeekday(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		tests := []struct {
			want string
			day  Weekday
		}{
			{day: Sunday, want: "Sunday"},
			{day: Monday, want: "Monday"},
			{day: Tuesday, want: "Tuesday"},
			{day: Wednesday, want: "Wednesday"},
			{day: Thursday, want: "Thursday"},
			{day: Friday, want: "Friday"},
			{day: Saturday, want: "Saturday"},
			{day: Weekday(99), want: "Unknown"},
		}

		for _, tt := range tests {
			if got := tt.day.String(); got != tt.want {
				t.Errorf("%d.String() = %v, want %v", tt.day, got, tt.want)
			}
		}
	})

	t.Run("WeekdayFromTime", func(t *testing.T) {
		if got := WeekdayFromTime(time.Sunday); got != Sunday {
			t.Errorf("WeekdayFromTime(Sunday) = %v, want Sunday", got)
		}
		if got := WeekdayFromTime(time.Monday); got != Monday {
			t.Errorf("WeekdayFromTime(Monday) = %v, want Monday", got)
		}
	})
}

// TestWeekdays tests weekday sets.
func TestWeekdays(t *testing.T) {
	t.Run("EveryDay", func(t *testing.T) {
		days := EveryDay()
		if len(days) != 7 {
			t.Errorf("EveryDay() returned %d days, want 7", len(days))
		}
	})

	t.Run("WeekdaysOnly", func(t *testing.T) {
		days := WeekdaysOnly()
		if len(days) != 5 {
			t.Errorf("WeekdaysOnly() returned %d days, want 5", len(days))
		}
		if days.Contains(Saturday) || days.Contains(Sunday) {
			t.Errorf("WeekdaysOnly() should not contain weekends")
		}
	})

	t.Run("Weekends", func(t *testing.T) {
		days := Weekends()
		if len(days) != 2 {
			t.Errorf("Weekends() returned %d days, want 2", len(days))
		}
		if !days.Contains(Saturday) || !days.Contains(Sunday) {
			t.Errorf("Weekends() should contain Saturday and Sunday")
		}
	})

	t.Run("Contains", func(t *testing.T) {
		days := Weekdays{Monday, Wednesday, Friday}
		if !days.Contains(Monday) {
			t.Errorf("Contains(Monday) should be true")
		}
		if days.Contains(Tuesday) {
			t.Errorf("Contains(Tuesday) should be false")
		}
	})

	t.Run("ToGen1Format", func(t *testing.T) {
		days := Weekdays{Monday, Wednesday, Friday}
		got := days.ToGen1Format()
		if got != "135" {
			t.Errorf("ToGen1Format() = %v, want 135", got)
		}

		allDays := EveryDay()
		got = allDays.ToGen1Format()
		if got != "0123456" {
			t.Errorf("ToGen1Format() for EveryDay = %v, want 0123456", got)
		}
	})
}

// TestScheduleTime tests schedule time operations.
func TestScheduleTime(t *testing.T) {
	t.Run("NewScheduleTime", func(t *testing.T) {
		st := NewScheduleTime(14, 30)
		if st.Hour != 14 || st.Minute != 30 {
			t.Errorf("NewScheduleTime(14, 30) = %v:%v, want 14:30", st.Hour, st.Minute)
		}
	})

	t.Run("String", func(t *testing.T) {
		st := ScheduleTime{Hour: 9, Minute: 5}
		if got := st.String(); got != "09:05" {
			t.Errorf("String() = %v, want 09:05", got)
		}
	})

	t.Run("ToGen1Format", func(t *testing.T) {
		st := ScheduleTime{Hour: 9, Minute: 5}
		if got := st.ToGen1Format(); got != "0905" {
			t.Errorf("ToGen1Format() = %v, want 0905", got)
		}
	})

	t.Run("ParseScheduleTime", func(t *testing.T) {
		tests := []struct {
			input   string
			wantErr bool
			wantH   int
			wantM   int
		}{
			{"14:30", false, 14, 30},
			{"09:05", false, 9, 5},
			{"00:00", false, 0, 0},
			{"23:59", false, 23, 59},
			{"invalid", true, 0, 0},
			{"24:00", true, 0, 0},
			{"12:60", true, 0, 0},
			{"xx:00", true, 0, 0},
			{"12:xx", true, 0, 0},
		}

		for _, tt := range tests {
			got, err := ParseScheduleTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseScheduleTime(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				continue
			}
			if !tt.wantErr && (got.Hour != tt.wantH || got.Minute != tt.wantM) {
				t.Errorf("ParseScheduleTime(%v) = %v:%v, want %v:%v", tt.input, got.Hour, got.Minute, tt.wantH, tt.wantM)
			}
		}
	})
}

// TestScheduleEntry tests schedule entry operations.
func TestScheduleEntry(t *testing.T) {
	t.Run("ToGen1Rule on", func(t *testing.T) {
		entry := &ScheduleEntry{
			Time:    ScheduleTime{Hour: 6, Minute: 30},
			Days:    Weekdays{Monday, Tuesday, Wednesday, Thursday, Friday},
			Action:  ActionSet(true),
			Enabled: true,
		}
		got := entry.ToGen1Rule()
		if got != "0630-12345-on" {
			t.Errorf("ToGen1Rule() = %v, want 0630-12345-on", got)
		}
	})

	t.Run("ToGen1Rule off", func(t *testing.T) {
		entry := &ScheduleEntry{
			Time:   ScheduleTime{Hour: 22, Minute: 0},
			Days:   EveryDay(),
			Action: ActionSet(false),
		}
		got := entry.ToGen1Rule()
		if got != "2200-0123456-off" {
			t.Errorf("ToGen1Rule() = %v, want 2200-0123456-off", got)
		}
	})

	t.Run("ToGen1Rule toggle", func(t *testing.T) {
		entry := &ScheduleEntry{
			Time:   ScheduleTime{Hour: 12, Minute: 0},
			Days:   Weekdays{Saturday, Sunday},
			Action: ActionToggle(),
		}
		got := entry.ToGen1Rule()
		if got != "1200-60-toggle" {
			t.Errorf("ToGen1Rule() = %v, want 1200-60-toggle", got)
		}
	})
}

// TestParseGen1Rule tests parsing Gen1 schedule rules.
func TestParseGen1Rule(t *testing.T) {
	tests := []struct {
		rule    string
		wantH   int
		wantM   int
		wantErr bool
		wantOn  bool
	}{
		{rule: "0630-12345-on", wantErr: false, wantH: 6, wantM: 30, wantOn: true},
		{rule: "2200-0123456-off", wantErr: false, wantH: 22, wantM: 0, wantOn: false},
		{rule: "1200-06-toggle", wantErr: false, wantH: 12, wantM: 0, wantOn: false},
		{rule: "invalid", wantErr: true, wantH: 0, wantM: 0, wantOn: false},
		{rule: "063-12345-on", wantErr: true, wantH: 0, wantM: 0, wantOn: false},
		{rule: "0630-12345-unknown", wantErr: true, wantH: 0, wantM: 0, wantOn: false},
		{rule: "0630-128-on", wantErr: true, wantH: 0, wantM: 0, wantOn: false},
		{rule: "xx30-12345-on", wantErr: true, wantH: 0, wantM: 0, wantOn: false},
		{rule: "06xx-12345-on", wantErr: true, wantH: 0, wantM: 0, wantOn: false},
	}

	for _, tt := range tests {
		entry, err := ParseGen1Rule(tt.rule)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseGen1Rule(%v) error = %v, wantErr %v", tt.rule, err, tt.wantErr)
			continue
		}
		if tt.wantErr {
			continue
		}
		if entry.Time.Hour != tt.wantH || entry.Time.Minute != tt.wantM {
			t.Errorf("ParseGen1Rule(%v) time = %v:%v, want %v:%v", tt.rule, entry.Time.Hour, entry.Time.Minute, tt.wantH, tt.wantM)
		}
	}
}

// TestGetSchedules tests retrieving schedules from Gen2 devices.
func TestGetSchedules(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Schedule.List" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{"jobs":[{"id":1,"enable":true,"timespec":"30 6 * * 1,2,3,4,5"}]}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		schedules, err := GetSchedules(ctx, dev)
		if err != nil {
			t.Fatalf("GetSchedules() error: %v", err)
		}
		if len(schedules) != 1 {
			t.Errorf("GetSchedules() returned %d schedules, want 1", len(schedules))
		}
	})

	t.Run("nil device", func(t *testing.T) {
		dev := &factory.Gen2Device{Device: nil}
		_, err := GetSchedules(ctx, dev)
		if err != types.ErrNilDevice {
			t.Errorf("GetSchedules() error = %v, want %v", err, types.ErrNilDevice)
		}
	})

	t.Run("rpc error", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			return nil, types.ErrRPCMethod
		})

		_, err := GetSchedules(ctx, dev)
		if err == nil {
			t.Errorf("GetSchedules() should fail on RPC error")
		}
	})

	t.Run("invalid response", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			resp := `{"jsonrpc":"2.0","id":1,"result":"invalid"}`
			return json.RawMessage(resp), nil
		})

		_, err := GetSchedules(ctx, dev)
		if err == nil {
			t.Errorf("GetSchedules() should fail on invalid response")
		}
	})
}

// TestCreateSchedule tests creating schedules on Gen2 devices.
func TestCreateSchedule(t *testing.T) {
	ctx := context.Background()

	t.Run("set action", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Schedule.Create" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{"id":123}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		entry := &ScheduleEntry{
			Time:    ScheduleTime{Hour: 6, Minute: 30},
			Days:    WeekdaysOnly(),
			Action:  ActionSet(true),
			Enabled: true,
		}

		id, err := CreateSchedule(ctx, dev, entry)
		if err != nil {
			t.Fatalf("CreateSchedule() error: %v", err)
		}
		if id != 123 {
			t.Errorf("CreateSchedule() id = %v, want 123", id)
		}
	})

	t.Run("toggle action", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Schedule.Create" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{"id":124}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		entry := &ScheduleEntry{
			Time:    ScheduleTime{Hour: 12, Minute: 0},
			Days:    Weekends(),
			Action:  ActionToggle(),
			Enabled: true,
		}

		id, err := CreateSchedule(ctx, dev, entry)
		if err != nil {
			t.Fatalf("CreateSchedule() error: %v", err)
		}
		if id != 124 {
			t.Errorf("CreateSchedule() id = %v, want 124", id)
		}
	})

	t.Run("brightness action", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Schedule.Create" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{"id":125}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		entry := &ScheduleEntry{
			Time:    ScheduleTime{Hour: 22, Minute: 0},
			Days:    EveryDay(),
			Action:  ActionSetBrightness(30),
			Enabled: true,
		}

		id, err := CreateSchedule(ctx, dev, entry)
		if err != nil {
			t.Fatalf("CreateSchedule() error: %v", err)
		}
		if id != 125 {
			t.Errorf("CreateSchedule() id = %v, want 125", id)
		}
	})

	t.Run("nil device", func(t *testing.T) {
		dev := &factory.Gen2Device{Device: nil}
		entry := &ScheduleEntry{
			Time:   ScheduleTime{Hour: 6, Minute: 30},
			Days:   WeekdaysOnly(),
			Action: ActionSet(true),
		}
		_, err := CreateSchedule(ctx, dev, entry)
		if err != types.ErrNilDevice {
			t.Errorf("CreateSchedule() error = %v, want %v", err, types.ErrNilDevice)
		}
	})

	t.Run("rpc error", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			return nil, types.ErrRPCMethod
		})

		entry := &ScheduleEntry{
			Time:   ScheduleTime{Hour: 6, Minute: 30},
			Days:   WeekdaysOnly(),
			Action: ActionSet(true),
		}

		_, err := CreateSchedule(ctx, dev, entry)
		if err == nil {
			t.Errorf("CreateSchedule() should fail on RPC error")
		}
	})

	t.Run("invalid response", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Schedule.Create" {
				resp := `{"jsonrpc":"2.0","id":1,"result":"invalid"}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		entry := &ScheduleEntry{
			Time:   ScheduleTime{Hour: 6, Minute: 30},
			Days:   WeekdaysOnly(),
			Action: ActionSet(true),
		}

		_, err := CreateSchedule(ctx, dev, entry)
		if err == nil {
			t.Errorf("CreateSchedule() should fail on invalid response")
		}
	})
}

// TestDeleteSchedule tests deleting schedules from Gen2 devices.
func TestDeleteSchedule(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Schedule.Delete" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		err := DeleteSchedule(ctx, dev, 123)
		if err != nil {
			t.Errorf("DeleteSchedule() error: %v", err)
		}
	})

	t.Run("nil device", func(t *testing.T) {
		dev := &factory.Gen2Device{Device: nil}
		err := DeleteSchedule(ctx, dev, 123)
		if err != types.ErrNilDevice {
			t.Errorf("DeleteSchedule() error = %v, want %v", err, types.ErrNilDevice)
		}
	})

	t.Run("rpc error", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			return nil, types.ErrRPCMethod
		})

		err := DeleteSchedule(ctx, dev, 123)
		if err == nil {
			t.Errorf("DeleteSchedule() should fail on RPC error")
		}
	})
}

// TestEnableSchedule tests enabling/disabling schedules on Gen2 devices.
func TestEnableSchedule(t *testing.T) {
	ctx := context.Background()

	t.Run("enable", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Schedule.Update" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		err := EnableSchedule(ctx, dev, 123, true)
		if err != nil {
			t.Errorf("EnableSchedule() error: %v", err)
		}
	})

	t.Run("disable", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Schedule.Update" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		err := EnableSchedule(ctx, dev, 123, false)
		if err != nil {
			t.Errorf("EnableSchedule() error: %v", err)
		}
	})

	t.Run("nil device", func(t *testing.T) {
		dev := &factory.Gen2Device{Device: nil}
		err := EnableSchedule(ctx, dev, 123, true)
		if err != types.ErrNilDevice {
			t.Errorf("EnableSchedule() error = %v, want %v", err, types.ErrNilDevice)
		}
	})

	t.Run("rpc error", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			return nil, types.ErrRPCMethod
		})

		err := EnableSchedule(ctx, dev, 123, true)
		if err == nil {
			t.Errorf("EnableSchedule() should fail on RPC error")
		}
	})
}

// TestDeleteAllSchedules tests deleting all schedules from Gen2 devices.
func TestDeleteAllSchedules(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Schedule.DeleteAll" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		err := DeleteAllSchedules(ctx, dev)
		if err != nil {
			t.Errorf("DeleteAllSchedules() error: %v", err)
		}
	})

	t.Run("nil device", func(t *testing.T) {
		dev := &factory.Gen2Device{Device: nil}
		err := DeleteAllSchedules(ctx, dev)
		if err != types.ErrNilDevice {
			t.Errorf("DeleteAllSchedules() error = %v, want %v", err, types.ErrNilDevice)
		}
	})

	t.Run("rpc error", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			return nil, types.ErrRPCMethod
		})

		err := DeleteAllSchedules(ctx, dev)
		if err == nil {
			t.Errorf("DeleteAllSchedules() should fail on RPC error")
		}
	})
}
