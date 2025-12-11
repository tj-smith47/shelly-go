package helpers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/tj-smith47/shelly-go/factory"
	"github.com/tj-smith47/shelly-go/types"
)

// TestActionCreators tests the action creator functions.
func TestActionCreators(t *testing.T) {
	t.Run("ActionSet", func(t *testing.T) {
		action := ActionSet(true)
		if action.Type != ActionTypeSet {
			t.Errorf("Type = %v, want %v", action.Type, ActionTypeSet)
		}
		if !action.On {
			t.Errorf("On = %v, want true", action.On)
		}

		actionOff := ActionSet(false)
		if actionOff.On {
			t.Errorf("On = %v, want false", actionOff.On)
		}
	})

	t.Run("ActionToggle", func(t *testing.T) {
		action := ActionToggle()
		if action.Type != ActionTypeToggle {
			t.Errorf("Type = %v, want %v", action.Type, ActionTypeToggle)
		}
	})

	t.Run("ActionSetBrightness", func(t *testing.T) {
		action := ActionSetBrightness(75)
		if action.Type != ActionTypeBrightness {
			t.Errorf("Type = %v, want %v", action.Type, ActionTypeBrightness)
		}
		if action.Brightness != 75 {
			t.Errorf("Brightness = %v, want 75", action.Brightness)
		}
	})
}

// TestNewScene tests scene creation.
func TestNewScene(t *testing.T) {
	scene := NewScene("Movie Night")

	if scene.Name() != "Movie Night" {
		t.Errorf("Name() = %v, want Movie Night", scene.Name())
	}
	if scene.Len() != 0 {
		t.Errorf("Len() = %v, want 0", scene.Len())
	}
}

// TestSceneName tests getting and setting the scene name.
func TestSceneName(t *testing.T) {
	scene := NewScene("Original")

	if scene.Name() != "Original" {
		t.Errorf("Name() = %v, want Original", scene.Name())
	}

	scene.SetName("Updated")
	if scene.Name() != "Updated" {
		t.Errorf("Name() = %v, want Updated", scene.Name())
	}
}

// TestSceneAddAction tests adding actions to a scene.
func TestSceneAddAction(t *testing.T) {
	scene := NewScene("Test")
	dev := &mockDevice{addr: "192.168.1.100"}

	scene.AddAction(dev, ActionSet(true))
	if scene.Len() != 1 {
		t.Errorf("Len() = %v, want 1", scene.Len())
	}

	actions := scene.Actions()
	if len(actions) != 1 {
		t.Fatalf("Actions() returned %d actions, want 1", len(actions))
	}
	if actions[0].DeviceAddress != "192.168.1.100" {
		t.Errorf("DeviceAddress = %v, want 192.168.1.100", actions[0].DeviceAddress)
	}
	if actions[0].Action.Type != ActionTypeSet {
		t.Errorf("Action.Type = %v, want %v", actions[0].Action.Type, ActionTypeSet)
	}
}

// TestSceneRemoveAction tests removing actions from a scene.
func TestSceneRemoveAction(t *testing.T) {
	scene := NewScene("Test")
	dev1 := &mockDevice{addr: "192.168.1.100"}
	dev2 := &mockDevice{addr: "192.168.1.101"}

	scene.AddAction(dev1, ActionSet(true))
	scene.AddAction(dev2, ActionSet(false))

	// Remove existing action
	if !scene.RemoveAction("192.168.1.100") {
		t.Errorf("RemoveAction() should return true for existing action")
	}
	if scene.Len() != 1 {
		t.Errorf("Len() = %v, want 1", scene.Len())
	}

	// Remove non-existing action
	if scene.RemoveAction("192.168.1.200") {
		t.Errorf("RemoveAction() should return false for non-existing action")
	}
}

// TestSceneClear tests clearing all actions from a scene.
func TestSceneClear(t *testing.T) {
	scene := NewScene("Test")
	dev1 := &mockDevice{addr: "192.168.1.100"}
	dev2 := &mockDevice{addr: "192.168.1.101"}

	scene.AddAction(dev1, ActionSet(true))
	scene.AddAction(dev2, ActionSet(false))

	scene.Clear()
	if scene.Len() != 0 {
		t.Errorf("Len() = %v, want 0", scene.Len())
	}
}

// TestSceneResults tests the SceneResults helper methods.
func TestSceneResults(t *testing.T) {
	t.Run("AllSuccessful", func(t *testing.T) {
		results := SceneResults{
			{Success: true},
			{Success: true},
		}
		if !results.AllSuccessful() {
			t.Errorf("AllSuccessful() should return true")
		}

		results = append(results, SceneResult{Success: false})
		if results.AllSuccessful() {
			t.Errorf("AllSuccessful() should return false with failures")
		}
	})

	t.Run("Failures", func(t *testing.T) {
		results := SceneResults{
			{Success: true},
			{Success: false, Error: types.ErrTimeout},
			{Success: true},
			{Success: false, Error: types.ErrAuth},
		}

		failures := results.Failures()
		if len(failures) != 2 {
			t.Errorf("Failures() returned %d results, want 2", len(failures))
		}
	})

	t.Run("empty results", func(t *testing.T) {
		results := SceneResults{}
		if !results.AllSuccessful() {
			t.Errorf("AllSuccessful() should return true for empty results")
		}
	})
}

// TestSceneActivate tests activating a scene.
func TestSceneActivate(t *testing.T) {
	ctx := context.Background()

	t.Run("set action", func(t *testing.T) {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"ison":true}`))
		})
		defer server.Close()

		scene := NewScene("Test")
		scene.AddAction(dev, ActionSet(true))

		results := scene.Activate(ctx)
		if !results.AllSuccessful() {
			t.Errorf("Activate() failed: %v", results[0].Error)
		}
	})

	t.Run("toggle action", func(t *testing.T) {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"ison":true}`))
		})
		defer server.Close()

		scene := NewScene("Test")
		scene.AddAction(dev, ActionToggle())

		results := scene.Activate(ctx)
		if !results.AllSuccessful() {
			t.Errorf("Activate() failed: %v", results[0].Error)
		}
	})

	t.Run("brightness action", func(t *testing.T) {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"ison":true,"brightness":50}`))
		})
		defer server.Close()

		scene := NewScene("Test")
		scene.AddAction(dev, ActionSetBrightness(50))

		results := scene.Activate(ctx)
		if !results.AllSuccessful() {
			t.Errorf("Activate() failed: %v", results[0].Error)
		}
	})

	t.Run("nil device", func(t *testing.T) {
		scene := NewScene("Test")
		// Manually add action without device
		scene.mu.Lock()
		scene.actions = append(scene.actions, SceneAction{
			DeviceAddress: "192.168.1.100",
			Action:        ActionSet(true),
			Device:        nil,
		})
		scene.mu.Unlock()

		results := scene.Activate(ctx)
		if results.AllSuccessful() {
			t.Errorf("Activate() should fail for nil device")
		}
	})

	t.Run("multiple actions", func(t *testing.T) {
		dev1, server1 := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"ison":true}`))
		})
		defer server1.Close()

		dev2, server2 := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"ison":false}`))
		})
		defer server2.Close()

		scene := NewScene("Test")
		scene.AddAction(dev1, ActionSet(true))
		scene.AddAction(dev2, ActionSet(false))

		results := scene.Activate(ctx)
		if len(results) != 2 {
			t.Errorf("Activate() returned %d results, want 2", len(results))
		}
		if !results.AllSuccessful() {
			t.Errorf("Activate() failed")
		}
	})

	t.Run("empty scene", func(t *testing.T) {
		scene := NewScene("Empty")
		results := scene.Activate(ctx)
		if len(results) != 0 {
			t.Errorf("Activate() returned %d results, want 0", len(results))
		}
		if !results.AllSuccessful() {
			t.Errorf("Activate() should succeed for empty scene")
		}
	})

	t.Run("gen2 device", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			resp := `{"jsonrpc":"2.0","id":1,"result":{}}`
			return json.RawMessage(resp), nil
		})

		scene := NewScene("Test")
		scene.AddAction(dev, ActionSet(true))

		results := scene.Activate(ctx)
		if !results.AllSuccessful() {
			t.Errorf("Activate() failed: %v", results[0].Error)
		}
	})
}

// TestSceneJSON tests scene serialization and deserialization.
func TestSceneJSON(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		dev := &mockDevice{addr: "192.168.1.100"}
		scene := NewScene("Movie Night")
		scene.AddAction(dev, ActionSet(false))
		scene.AddAction(&mockDevice{addr: "192.168.1.101"}, ActionSetBrightness(30))

		data, err := scene.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON() error: %v", err)
		}

		loaded, err := SceneFromJSON(data)
		if err != nil {
			t.Fatalf("SceneFromJSON() error: %v", err)
		}

		if loaded.Name() != "Movie Night" {
			t.Errorf("Name() = %v, want Movie Night", loaded.Name())
		}
		if loaded.Len() != 2 {
			t.Errorf("Len() = %v, want 2", loaded.Len())
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := SceneFromJSON([]byte("invalid json"))
		if err == nil {
			t.Errorf("SceneFromJSON() should fail for invalid JSON")
		}
	})
}

// TestSceneLinkDevices tests linking devices to a scene.
func TestSceneLinkDevices(t *testing.T) {
	// Create scene from JSON (no device references)
	data := []byte(`{"name":"Test","actions":[{"device_address":"192.168.1.100","action":{"type":"set","on":true}},{"device_address":"192.168.1.101","action":{"type":"set","on":false}}]}`)
	scene, err := SceneFromJSON(data)
	if err != nil {
		t.Fatalf("SceneFromJSON() error: %v", err)
	}

	// Initially unlinked
	unlinked := scene.UnlinkedAddresses()
	if len(unlinked) != 2 {
		t.Errorf("UnlinkedAddresses() returned %d, want 2", len(unlinked))
	}

	// Link devices
	dev1 := &mockDevice{addr: "192.168.1.100"}
	dev2 := &mockDevice{addr: "192.168.1.101"}
	scene.LinkDevices([]factory.Device{dev1, dev2})

	// Check linked
	unlinked = scene.UnlinkedAddresses()
	if len(unlinked) != 0 {
		t.Errorf("UnlinkedAddresses() returned %d after linking, want 0", len(unlinked))
	}

	// Verify devices are linked
	actions := scene.Actions()
	if actions[0].Device != dev1 {
		t.Errorf("First action device not linked correctly")
	}
	if actions[1].Device != dev2 {
		t.Errorf("Second action device not linked correctly")
	}
}

// TestSceneLinkDevicesPartial tests partial device linking.
func TestSceneLinkDevicesPartial(t *testing.T) {
	data := []byte(`{"name":"Test","actions":[{"device_address":"192.168.1.100","action":{"type":"set","on":true}},{"device_address":"192.168.1.101","action":{"type":"set","on":false}}]}`)
	scene, err := SceneFromJSON(data)
	if err != nil {
		t.Fatalf("SceneFromJSON() error: %v", err)
	}

	// Only link one device
	dev1 := &mockDevice{addr: "192.168.1.100"}
	scene.LinkDevices([]factory.Device{dev1})

	unlinked := scene.UnlinkedAddresses()
	if len(unlinked) != 1 {
		t.Errorf("UnlinkedAddresses() returned %d, want 1", len(unlinked))
	}
	if unlinked[0] != "192.168.1.101" {
		t.Errorf("UnlinkedAddresses()[0] = %v, want 192.168.1.101", unlinked[0])
	}
}

// TestExecuteActionUnknownType tests executing an unknown action type.
func TestExecuteActionUnknownType(t *testing.T) {
	ctx := context.Background()
	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ison":true}`))
	})
	defer server.Close()

	err := executeAction(ctx, dev, Action{Type: "unknown"})
	if err == nil {
		t.Errorf("executeAction() should fail for unknown action type")
	}
}

// TestExecuteActionErrors tests error handling in executeAction.
func TestExecuteActionErrors(t *testing.T) {
	ctx := context.Background()

	// Create a device that will fail for all operations
	failingDev := &factory.Gen1Device{Device: nil}

	t.Run("set action error", func(t *testing.T) {
		err := executeAction(ctx, failingDev, ActionSet(true))
		if err == nil {
			t.Errorf("executeAction() should fail for set action with nil device")
		}
	})

	t.Run("toggle action error", func(t *testing.T) {
		err := executeAction(ctx, failingDev, ActionToggle())
		if err == nil {
			t.Errorf("executeAction() should fail for toggle action with nil device")
		}
	})

	t.Run("brightness action error", func(t *testing.T) {
		err := executeAction(ctx, failingDev, ActionSetBrightness(50))
		if err == nil {
			t.Errorf("executeAction() should fail for brightness action with nil device")
		}
	})
}
