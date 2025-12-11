package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/transport"
)

// Input provides access to Gen1 input channels.
//
// Inputs are physical buttons or switches connected to Shelly devices.
// They can detect state changes and button events.
type Input struct {
	transport transport.Transport
	id        int
}

// NewInput creates a new Input component accessor.
//
// Parameters:
//   - t: The transport to use for API calls
//   - id: The input index (0-based)
func NewInput(t transport.Transport, id int) *Input {
	return &Input{
		transport: t,
		id:        id,
	}
}

// ID returns the input index.
func (i *Input) ID() int {
	return i.id
}

// InputStatus contains the current input state.
type InputStatus struct {
	Event    string `json:"event,omitempty"`
	Input    int    `json:"input"`
	EventCnt int    `json:"event_cnt,omitempty"`
}

// InputConfig contains input configuration options.
type InputConfig struct {
	// Name is the input name.
	Name string `json:"name,omitempty"`

	// BtnType is the button type ("momentary", "toggle", "edge", "detached").
	BtnType string `json:"btn_type,omitempty"`

	// BtnReverse reverses button logic.
	BtnReverse bool `json:"btn_reverse,omitempty"`
}

// GetStatus retrieves the current input status.
func (i *Input) GetStatus(ctx context.Context) (*InputStatus, error) {
	// Input status is part of device status, need to parse from /status
	resp, err := i.transport.Call(ctx, "/status", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	// Parse the status to get inputs
	var deviceStatus struct {
		Inputs []InputStatus `json:"inputs"`
	}
	if err := json.Unmarshal(resp, &deviceStatus); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	if i.id >= len(deviceStatus.Inputs) {
		return nil, fmt.Errorf("input %d not found (device has %d inputs)", i.id, len(deviceStatus.Inputs))
	}

	return &deviceStatus.Inputs[i.id], nil
}

// GetState returns the current input state (0 or 1).
func (i *Input) GetState(ctx context.Context) (int, error) {
	status, err := i.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.Input, nil
}

// IsOn returns true if the input is in the "on" state.
func (i *Input) IsOn(ctx context.Context) (bool, error) {
	state, err := i.GetState(ctx)
	if err != nil {
		return false, err
	}
	return state == 1, nil
}

// GetLastEvent returns the last event type and counter.
func (i *Input) GetLastEvent(ctx context.Context) (eventType string, eventCount int, err error) {
	status, err := i.GetStatus(ctx)
	if err != nil {
		return "", 0, err
	}
	return status.Event, status.EventCnt, nil
}

// GetEventCount returns the event counter.
func (i *Input) GetEventCount(ctx context.Context) (int, error) {
	status, err := i.GetStatus(ctx)
	if err != nil {
		return 0, err
	}
	return status.EventCnt, nil
}

// SetButtonType sets the button input type.
//
// Parameters:
//   - btnType: "momentary", "toggle", "edge", or "detached"
//
// For relay devices, this affects how the input controls the relay.
// For i3 and similar devices, this affects event detection.
func (i *Input) SetButtonType(ctx context.Context, btnType string) error {
	// Button type is typically set per-relay
	path := fmt.Sprintf("/settings/relay/%d?btn_type=%s", i.id, btnType)
	_, err := i.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set button type: %w", err)
	}
	return nil
}

// SetButtonReverse enables or disables button logic reversal.
func (i *Input) SetButtonReverse(ctx context.Context, reverse bool) error {
	val := boolFalse
	if reverse {
		val = boolTrue
	}
	path := fmt.Sprintf("/settings/relay/%d?btn_reverse=%s", i.id, val)
	_, err := i.transport.Call(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set button reverse: %w", err)
	}
	return nil
}

// EventType represents input event types.
type EventType string

const (
	// EventNone represents no event.
	EventNone EventType = ""

	// EventShortPress represents a short button press (S).
	EventShortPress EventType = "S"

	// EventLongPress represents a long button press (L).
	EventLongPress EventType = "L"

	// EventDoubleShortPress represents a double short press (SS).
	EventDoubleShortPress EventType = "SS"

	// EventTripleShortPress represents a triple short press (SSS).
	EventTripleShortPress EventType = "SSS"

	// EventShortLongPress represents a short then long press (SL).
	EventShortLongPress EventType = "SL"

	// EventLongShortPress represents a long then short press (LS).
	EventLongShortPress EventType = "LS"
)

// String returns the string representation of the event type.
func (e EventType) String() string {
	switch e {
	case EventNone:
		return "None"
	case EventShortPress:
		return "Short Press"
	case EventLongPress:
		return "Long Press"
	case EventDoubleShortPress:
		return "Double Short Press"
	case EventTripleShortPress:
		return "Triple Short Press"
	case EventShortLongPress:
		return "Short-Long Press"
	case EventLongShortPress:
		return "Long-Short Press"
	default:
		return string(e)
	}
}

// ParseEventType converts a string to an EventType.
func ParseEventType(s string) EventType {
	return EventType(s)
}
