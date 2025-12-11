package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tj-smith47/shelly-go/factory"
)

// ActionType represents the type of scene action.
type ActionType string

const (
	// ActionTypeSet turns a switch on or off.
	ActionTypeSet ActionType = "set"

	// ActionTypeToggle toggles a switch.
	ActionTypeToggle ActionType = "toggle"

	// ActionTypeBrightness sets brightness for a light.
	ActionTypeBrightness ActionType = "brightness"
)

// Action represents an action to perform on a device in a scene.
type Action struct {
	// Type is the action type.
	Type ActionType `json:"type"`

	// On is used for ActionTypeSet.
	On bool `json:"on,omitempty"`

	// Brightness is used for ActionTypeBrightness.
	Brightness int `json:"brightness,omitempty"`
}

// ActionSet creates a set on/off action.
func ActionSet(on bool) Action {
	return Action{Type: ActionTypeSet, On: on}
}

// ActionToggle creates a toggle action.
func ActionToggle() Action {
	return Action{Type: ActionTypeToggle}
}

// ActionSetBrightness creates a brightness action.
func ActionSetBrightness(brightness int) Action {
	return Action{Type: ActionTypeBrightness, Brightness: brightness}
}

// SceneAction associates an action with a device.
type SceneAction struct {
	Device        factory.Device `json:"-"`
	DeviceAddress string         `json:"device_address"`
	Action        Action         `json:"action"`
}

// Scene represents a collection of device actions that can be activated together.
//
// Scenes are useful for automating common scenarios like "Movie Night",
// "Good Morning", etc. They can be serialized to JSON for storage.
type Scene struct {
	name    string
	actions []SceneAction
	mu      sync.RWMutex
}

// SceneData is the JSON-serializable representation of a Scene.
type SceneData struct {
	Name    string        `json:"name"`
	Actions []SceneAction `json:"actions"`
}

// NewScene creates a new scene with the given name.
func NewScene(name string) *Scene {
	return &Scene{
		name:    name,
		actions: make([]SceneAction, 0),
	}
}

// Name returns the scene name.
func (s *Scene) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.name
}

// SetName updates the scene name.
func (s *Scene) SetName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.name = name
}

// AddAction adds an action to the scene.
func (s *Scene) AddAction(device factory.Device, action Action) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.actions = append(s.actions, SceneAction{
		DeviceAddress: device.Address(),
		Action:        action,
		Device:        device,
	})
}

// Actions returns a copy of the scene actions.
func (s *Scene) Actions() []SceneAction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SceneAction, len(s.actions))
	copy(result, s.actions)
	return result
}

// Len returns the number of actions in the scene.
func (s *Scene) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.actions)
}

// Clear removes all actions from the scene.
func (s *Scene) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actions = s.actions[:0]
}

// RemoveAction removes an action for the given device address.
// Returns true if an action was removed.
func (s *Scene) RemoveAction(address string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, action := range s.actions {
		if action.DeviceAddress == address {
			s.actions = append(s.actions[:i], s.actions[i+1:]...)
			return true
		}
	}
	return false
}

// SceneResult contains the result of a scene activation.
type SceneResult struct {
	Error         error
	DeviceAddress string
	Action        Action
	Success       bool
}

// SceneResults is a collection of scene activation results.
type SceneResults []SceneResult

// AllSuccessful returns true if all actions succeeded.
func (r SceneResults) AllSuccessful() bool {
	for _, res := range r {
		if !res.Success {
			return false
		}
	}
	return true
}

// Failures returns only the failed results.
func (r SceneResults) Failures() SceneResults {
	var failures SceneResults
	for _, res := range r {
		if !res.Success {
			failures = append(failures, res)
		}
	}
	return failures
}

// Activate executes all actions in the scene concurrently.
func (s *Scene) Activate(ctx context.Context) SceneResults {
	s.mu.RLock()
	actions := make([]SceneAction, len(s.actions))
	copy(actions, s.actions)
	s.mu.RUnlock()

	results := make(SceneResults, len(actions))
	var wg sync.WaitGroup

	for i, action := range actions {
		wg.Add(1)
		go func(index int, sa SceneAction) {
			defer wg.Done()

			result := SceneResult{
				DeviceAddress: sa.DeviceAddress,
				Action:        sa.Action,
			}

			if sa.Device == nil {
				result.Error = fmt.Errorf("device not set for %s", sa.DeviceAddress)
				results[index] = result
				return
			}

			err := executeAction(ctx, sa.Device, sa.Action)
			if err != nil {
				result.Error = err
			} else {
				result.Success = true
			}
			results[index] = result
		}(i, action)
	}

	wg.Wait()
	return results
}

// executeAction executes a single action on a device.
func executeAction(ctx context.Context, dev factory.Device, action Action) error {
	switch action.Type {
	case ActionTypeSet:
		results := BatchSet(ctx, []factory.Device{dev}, action.On)
		if !results.AllSuccessful() {
			return results[0].Error
		}
	case ActionTypeToggle:
		results := BatchToggle(ctx, []factory.Device{dev})
		if !results.AllSuccessful() {
			return results[0].Error
		}
	case ActionTypeBrightness:
		results := BatchSetBrightness(ctx, []factory.Device{dev}, action.Brightness)
		if !results.AllSuccessful() {
			return results[0].Error
		}
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
	return nil
}

// ToJSON serializes the scene to JSON.
func (s *Scene) ToJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := SceneData{
		Name:    s.name,
		Actions: s.actions,
	}

	return json.Marshal(data)
}

// SceneFromJSON deserializes a scene from JSON.
//
// Note: The returned scene will not have Device references populated.
// Use LinkDevices to associate devices with the scene after loading.
func SceneFromJSON(data []byte) (*Scene, error) {
	var sceneData SceneData
	if err := json.Unmarshal(data, &sceneData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scene: %w", err)
	}

	scene := &Scene{
		name:    sceneData.Name,
		actions: sceneData.Actions,
	}

	return scene, nil
}

// LinkDevices associates devices with scene actions by address.
//
// This is used after loading a scene from JSON to connect the
// Device references for each action.
func (s *Scene) LinkDevices(devices []factory.Device) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create address lookup map
	deviceMap := make(map[string]factory.Device)
	for _, d := range devices {
		deviceMap[d.Address()] = d
	}

	// Link devices to actions
	for i := range s.actions {
		if dev, ok := deviceMap[s.actions[i].DeviceAddress]; ok {
			s.actions[i].Device = dev
		}
	}
}

// UnlinkedAddresses returns the addresses of actions that don't have
// linked devices.
func (s *Scene) UnlinkedAddresses() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var unlinked []string
	for _, action := range s.actions {
		if action.Device == nil {
			unlinked = append(unlinked, action.DeviceAddress)
		}
	}
	return unlinked
}
