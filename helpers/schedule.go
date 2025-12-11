package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tj-smith47/shelly-go/factory"
	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/types"
)

// Gen1 schedule state string constants.
const (
	stateOn     = "on"
	stateOff    = "off"
	stateToggle = "toggle"
)

// Weekday represents a day of the week for scheduling.
type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

// WeekdayFromTime converts a time.Weekday to our Weekday type.
func WeekdayFromTime(w time.Weekday) Weekday {
	return Weekday(w)
}

// String returns the string representation of the weekday.
func (w Weekday) String() string {
	switch w {
	case Sunday:
		return "Sunday"
	case Monday:
		return "Monday"
	case Tuesday:
		return "Tuesday"
	case Wednesday:
		return "Wednesday"
	case Thursday:
		return "Thursday"
	case Friday:
		return "Friday"
	case Saturday:
		return "Saturday"
	default:
		return "Unknown"
	}
}

// Weekdays represents a set of days for a schedule.
type Weekdays []Weekday

// EveryDay returns all days of the week.
func EveryDay() Weekdays {
	return Weekdays{Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday}
}

// Weekdays returns Monday through Friday.
func WeekdaysOnly() Weekdays {
	return Weekdays{Monday, Tuesday, Wednesday, Thursday, Friday}
}

// Weekends returns Saturday and Sunday.
func Weekends() Weekdays {
	return Weekdays{Saturday, Sunday}
}

// Contains returns true if the weekday is in the set.
func (w Weekdays) Contains(day Weekday) bool {
	for _, d := range w {
		if d == day {
			return true
		}
	}
	return false
}

// ToGen1Format converts weekdays to Gen1 schedule format.
// Gen1 uses a string like "0123456" where each digit is a day (0=Sunday).
func (w Weekdays) ToGen1Format() string {
	days := make([]byte, 0, 7)
	for _, day := range w {
		days = append(days, '0'+byte(day))
	}
	return string(days)
}

// ScheduleTime represents a time of day for scheduling.
type ScheduleTime struct {
	Hour   int
	Minute int
}

// NewScheduleTime creates a schedule time.
func NewScheduleTime(hour, minute int) ScheduleTime {
	return ScheduleTime{Hour: hour, Minute: minute}
}

// String returns the time in HH:MM format.
func (t ScheduleTime) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

// ToGen1Format converts to Gen1 HHMM format.
func (t ScheduleTime) ToGen1Format() string {
	return fmt.Sprintf("%02d%02d", t.Hour, t.Minute)
}

// ParseScheduleTime parses a time string in HH:MM format.
func ParseScheduleTime(s string) (ScheduleTime, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return ScheduleTime{}, fmt.Errorf("invalid time format: %s", s)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return ScheduleTime{}, fmt.Errorf("invalid hour: %s", parts[0])
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return ScheduleTime{}, fmt.Errorf("invalid minute: %s", parts[1])
	}

	return ScheduleTime{Hour: hour, Minute: minute}, nil
}

// ScheduleEntry represents a scheduled action.
type ScheduleEntry struct {
	Name    string       `json:"name,omitempty"`
	Days    Weekdays     `json:"days"`
	Action  Action       `json:"action"`
	Time    ScheduleTime `json:"time"`
	ID      int          `json:"id,omitempty"`
	Enabled bool         `json:"enabled"`
}

// ToGen1Rule converts the schedule to a Gen1 rule string.
// Format: HHMM-0123456-on/off
func (e *ScheduleEntry) ToGen1Rule() string {
	timeStr := e.Time.ToGen1Format()
	days := e.Days.ToGen1Format()
	state := stateOff
	if e.Action.Type == ActionTypeSet && e.Action.On {
		state = stateOn
	} else if e.Action.Type == ActionTypeToggle {
		state = stateToggle
	}
	return fmt.Sprintf("%s-%s-%s", timeStr, days, state)
}

// ParseGen1Rule parses a Gen1 schedule rule string.
func ParseGen1Rule(rule string) (*ScheduleEntry, error) {
	parts := strings.Split(rule, "-")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid rule format: %s", rule)
	}

	// Parse time (HHMM)
	if len(parts[0]) != 4 {
		return nil, fmt.Errorf("invalid time format: %s", parts[0])
	}
	hour, err := strconv.Atoi(parts[0][:2])
	if err != nil {
		return nil, fmt.Errorf("invalid hour: %s", parts[0][:2])
	}
	minute, err := strconv.Atoi(parts[0][2:])
	if err != nil {
		return nil, fmt.Errorf("invalid minute: %s", parts[0][2:])
	}

	// Parse days
	days := make(Weekdays, 0, len(parts[1]))
	for _, c := range parts[1] {
		day := int(c - '0')
		if day < 0 || day > 6 {
			return nil, fmt.Errorf("invalid day: %c", c)
		}
		days = append(days, Weekday(day))
	}

	// Parse action
	var action Action
	switch parts[2] {
	case stateOn:
		action = ActionSet(true)
	case stateOff:
		action = ActionSet(false)
	case stateToggle:
		action = ActionToggle()
	default:
		return nil, fmt.Errorf("invalid action: %s", parts[2])
	}

	return &ScheduleEntry{
		Time:    ScheduleTime{Hour: hour, Minute: minute},
		Days:    days,
		Action:  action,
		Enabled: true,
	}, nil
}

// Gen2Schedule represents a Gen2 device schedule entry from the API.
type Gen2Schedule struct {
	RawFields types.RawFields
	Timespec  string          `json:"timespec"`
	Calls     json.RawMessage `json:"calls"`
	ID        int             `json:"id"`
	Enable    bool            `json:"enable"`
}

// GetSchedules retrieves schedules from a Gen2 device.
func GetSchedules(ctx context.Context, dev *factory.Gen2Device) ([]Gen2Schedule, error) {
	if dev.Device == nil {
		return nil, types.ErrNilDevice
	}

	// Use the schedule component to list schedules
	schedComp, ok := dev.Schedule(0).(*gen2.BaseComponent)
	if !ok {
		return nil, types.ErrUnsupportedDevice
	}
	params := map[string]any{}
	result, err := schedComp.Client().Call(ctx, "Schedule.List", params)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	var response struct {
		Jobs []Gen2Schedule `json:"jobs"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse schedules: %w", err)
	}

	return response.Jobs, nil
}

// CreateSchedule creates a new schedule on a Gen2 device.
func CreateSchedule(ctx context.Context, dev *factory.Gen2Device, entry *ScheduleEntry) (int, error) {
	if dev.Device == nil {
		return 0, types.ErrNilDevice
	}

	// Build timespec in cron format: MM HH * * D1,D2,...
	dayList := make([]string, len(entry.Days))
	for i, d := range entry.Days {
		dayList[i] = strconv.Itoa(int(d))
	}
	timespec := fmt.Sprintf("%d %d * * %s", entry.Time.Minute, entry.Time.Hour, strings.Join(dayList, ","))

	// Build calls based on action
	var calls []map[string]any
	switch entry.Action.Type {
	case ActionTypeSet:
		calls = []map[string]any{
			{"method": "Switch.Set", "params": map[string]any{"id": 0, "on": entry.Action.On}},
		}
	case ActionTypeToggle:
		calls = []map[string]any{
			{"method": "Switch.Toggle", "params": map[string]any{"id": 0}},
		}
	case ActionTypeBrightness:
		calls = []map[string]any{
			{"method": "Light.Set", "params": map[string]any{"id": 0, "brightness": entry.Action.Brightness}},
		}
	}

	params := map[string]any{
		"enable":   entry.Enabled,
		"timespec": timespec,
		"calls":    calls,
	}

	schedComp, ok := dev.Schedule(0).(*gen2.BaseComponent)
	if !ok {
		return 0, types.ErrUnsupportedDevice
	}
	result, err := schedComp.Client().Call(ctx, "Schedule.Create", params)
	if err != nil {
		return 0, fmt.Errorf("failed to create schedule: %w", err)
	}

	var response struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return 0, fmt.Errorf("failed to parse create response: %w", err)
	}

	return response.ID, nil
}

// DeleteSchedule deletes a schedule from a Gen2 device.
func DeleteSchedule(ctx context.Context, dev *factory.Gen2Device, id int) error {
	if dev.Device == nil {
		return types.ErrNilDevice
	}

	params := map[string]any{"id": id}
	schedComp, ok := dev.Schedule(0).(*gen2.BaseComponent)
	if !ok {
		return types.ErrUnsupportedDevice
	}
	_, err := schedComp.Client().Call(ctx, "Schedule.Delete", params)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	return nil
}

// EnableSchedule enables or disables a schedule on a Gen2 device.
func EnableSchedule(ctx context.Context, dev *factory.Gen2Device, id int, enable bool) error {
	if dev.Device == nil {
		return types.ErrNilDevice
	}

	params := map[string]any{"id": id, "enable": enable}
	schedComp, ok := dev.Schedule(0).(*gen2.BaseComponent)
	if !ok {
		return types.ErrUnsupportedDevice
	}
	_, err := schedComp.Client().Call(ctx, "Schedule.Update", params)
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	return nil
}

// DeleteAllSchedules deletes all schedules from a Gen2 device.
func DeleteAllSchedules(ctx context.Context, dev *factory.Gen2Device) error {
	if dev.Device == nil {
		return types.ErrNilDevice
	}

	params := map[string]any{}
	schedComp, ok := dev.Schedule(0).(*gen2.BaseComponent)
	if !ok {
		return types.ErrUnsupportedDevice
	}
	_, err := schedComp.Client().Call(ctx, "Schedule.DeleteAll", params)
	if err != nil {
		return fmt.Errorf("failed to delete all schedules: %w", err)
	}

	return nil
}
