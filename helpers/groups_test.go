package helpers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/tj-smith47/shelly-go/factory"
	"github.com/tj-smith47/shelly-go/types"
)

// TestNewGroup tests group creation with options.
func TestNewGroup(t *testing.T) {
	t.Run("empty group", func(t *testing.T) {
		g := NewGroup("Test Group")
		if g.Name() != "Test Group" {
			t.Errorf("Name() = %v, want %v", g.Name(), "Test Group")
		}
		if g.Len() != 0 {
			t.Errorf("Len() = %v, want 0", g.Len())
		}
	})

	t.Run("with device", func(t *testing.T) {
		dev := &mockDevice{addr: "192.168.1.100"}
		g := NewGroup("Test", WithDevice(dev))
		if g.Len() != 1 {
			t.Errorf("Len() = %v, want 1", g.Len())
		}
	})

	t.Run("with devices", func(t *testing.T) {
		dev1 := &mockDevice{addr: "192.168.1.100"}
		dev2 := &mockDevice{addr: "192.168.1.101"}
		g := NewGroup("Test", WithDevices(dev1, dev2))
		if g.Len() != 2 {
			t.Errorf("Len() = %v, want 2", g.Len())
		}
	})

	t.Run("combined options", func(t *testing.T) {
		dev1 := &mockDevice{addr: "192.168.1.100"}
		dev2 := &mockDevice{addr: "192.168.1.101"}
		dev3 := &mockDevice{addr: "192.168.1.102"}
		g := NewGroup("Test", WithDevice(dev1), WithDevices(dev2, dev3))
		if g.Len() != 3 {
			t.Errorf("Len() = %v, want 3", g.Len())
		}
	})
}

// TestGroupName tests getting and setting the group name.
func TestGroupName(t *testing.T) {
	g := NewGroup("Original")

	if g.Name() != "Original" {
		t.Errorf("Name() = %v, want Original", g.Name())
	}

	g.SetName("Updated")
	if g.Name() != "Updated" {
		t.Errorf("Name() = %v, want Updated", g.Name())
	}
}

// TestGroupDevices tests the Devices method returns a copy.
func TestGroupDevices(t *testing.T) {
	dev := &mockDevice{addr: "192.168.1.100"}
	g := NewGroup("Test", WithDevice(dev))

	devices := g.Devices()
	if len(devices) != 1 {
		t.Errorf("Devices() returned %d devices, want 1", len(devices))
	}

	// Modify returned slice should not affect group
	devices[0] = nil
	if g.Devices()[0] != dev {
		t.Errorf("Devices() should return a copy, not the original slice")
	}
}

// TestGroupAdd tests adding devices to a group.
func TestGroupAdd(t *testing.T) {
	g := NewGroup("Test")

	dev := &mockDevice{addr: "192.168.1.100"}
	g.Add(dev)
	if g.Len() != 1 {
		t.Errorf("Len() = %v, want 1", g.Len())
	}

	// AddAll
	dev2 := &mockDevice{addr: "192.168.1.101"}
	dev3 := &mockDevice{addr: "192.168.1.102"}
	g.AddAll(dev2, dev3)
	if g.Len() != 3 {
		t.Errorf("Len() = %v, want 3", g.Len())
	}
}

// TestGroupRemove tests removing devices from a group.
func TestGroupRemove(t *testing.T) {
	dev1 := &mockDevice{addr: "192.168.1.100"}
	dev2 := &mockDevice{addr: "192.168.1.101"}
	g := NewGroup("Test", WithDevices(dev1, dev2))

	// Remove existing device
	if !g.Remove("192.168.1.100") {
		t.Errorf("Remove() should return true for existing device")
	}
	if g.Len() != 1 {
		t.Errorf("Len() = %v, want 1", g.Len())
	}

	// Remove non-existing device
	if g.Remove("192.168.1.200") {
		t.Errorf("Remove() should return false for non-existing device")
	}
}

// TestGroupClear tests clearing all devices from a group.
func TestGroupClear(t *testing.T) {
	dev1 := &mockDevice{addr: "192.168.1.100"}
	dev2 := &mockDevice{addr: "192.168.1.101"}
	g := NewGroup("Test", WithDevices(dev1, dev2))

	g.Clear()
	if g.Len() != 0 {
		t.Errorf("Len() = %v, want 0", g.Len())
	}
}

// TestGroupContains tests checking if a device is in the group.
func TestGroupContains(t *testing.T) {
	dev := &mockDevice{addr: "192.168.1.100"}
	g := NewGroup("Test", WithDevice(dev))

	if !g.Contains("192.168.1.100") {
		t.Errorf("Contains() should return true for existing device")
	}
	if g.Contains("192.168.1.200") {
		t.Errorf("Contains() should return false for non-existing device")
	}
}

// TestGroupAllOn tests the AllOn method.
func TestGroupAllOn(t *testing.T) {
	ctx := context.Background()
	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ison":true}`))
	})
	defer server.Close()

	g := NewGroup("Test", WithDevice(dev))
	results := g.AllOn(ctx)
	if !results.AllSuccessful() {
		t.Errorf("AllOn() failed: %v", results[0].Error)
	}
}

// TestGroupAllOff tests the AllOff method.
func TestGroupAllOff(t *testing.T) {
	ctx := context.Background()
	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ison":false}`))
	})
	defer server.Close()

	g := NewGroup("Test", WithDevice(dev))
	results := g.AllOff(ctx)
	if !results.AllSuccessful() {
		t.Errorf("AllOff() failed: %v", results[0].Error)
	}
}

// TestGroupToggle tests the Toggle method.
func TestGroupToggle(t *testing.T) {
	ctx := context.Background()
	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ison":true}`))
	})
	defer server.Close()

	g := NewGroup("Test", WithDevice(dev))
	results := g.Toggle(ctx)
	if !results.AllSuccessful() {
		t.Errorf("Toggle() failed: %v", results[0].Error)
	}
}

// TestGroupSet tests the Set method.
func TestGroupSet(t *testing.T) {
	ctx := context.Background()
	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ison":true}`))
	})
	defer server.Close()

	g := NewGroup("Test", WithDevice(dev))
	results := g.Set(ctx, true)
	if !results.AllSuccessful() {
		t.Errorf("Set() failed: %v", results[0].Error)
	}
}

// TestGroupSetBrightness tests the SetBrightness method.
func TestGroupSetBrightness(t *testing.T) {
	ctx := context.Background()
	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ison":true,"brightness":75}`))
	})
	defer server.Close()

	g := NewGroup("Test", WithDevice(dev))
	results := g.SetBrightness(ctx, 75)
	if !results.AllSuccessful() {
		t.Errorf("SetBrightness() failed: %v", results[0].Error)
	}
}

// TestGroupForEach tests iterating over devices.
func TestGroupForEach(t *testing.T) {
	dev1 := &mockDevice{addr: "192.168.1.100"}
	dev2 := &mockDevice{addr: "192.168.1.101"}
	g := NewGroup("Test", WithDevices(dev1, dev2))

	var addresses []string
	err := g.ForEach(func(d factory.Device) error {
		addresses = append(addresses, d.Address())
		return nil
	})

	if err != nil {
		t.Errorf("ForEach() returned error: %v", err)
	}
	if len(addresses) != 2 {
		t.Errorf("ForEach() visited %d devices, want 2", len(addresses))
	}
}

// TestGroupForEachError tests that ForEach stops on error.
func TestGroupForEachError(t *testing.T) {
	dev1 := &mockDevice{addr: "192.168.1.100"}
	dev2 := &mockDevice{addr: "192.168.1.101"}
	g := NewGroup("Test", WithDevices(dev1, dev2))

	expectedErr := errors.New("test error")
	count := 0
	err := g.ForEach(func(d factory.Device) error {
		count++
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("ForEach() error = %v, want %v", err, expectedErr)
	}
	if count != 1 {
		t.Errorf("ForEach() visited %d devices, want 1", count)
	}
}

// TestGroupFilter tests filtering devices in a group.
func TestGroupFilter(t *testing.T) {
	dev1 := &mockDevice{addr: "192.168.1.100", gen: types.Gen1}
	dev2 := &mockDevice{addr: "192.168.1.101", gen: types.Gen2Plus}
	dev3 := &mockDevice{addr: "192.168.1.102", gen: types.Gen1}
	g := NewGroup("Test", WithDevices(dev1, dev2, dev3))

	// Filter for Gen1 devices
	filtered := g.Filter(func(d factory.Device) bool {
		return d.Generation() == types.Gen1
	})

	if filtered.Len() != 2 {
		t.Errorf("Filter() returned %d devices, want 2", filtered.Len())
	}
	if !filtered.Contains("192.168.1.100") || !filtered.Contains("192.168.1.102") {
		t.Errorf("Filter() returned wrong devices")
	}
}

// TestGroupEmptyOperations tests operations on empty groups.
func TestGroupEmptyOperations(t *testing.T) {
	ctx := context.Background()
	g := NewGroup("Empty")

	results := g.AllOn(ctx)
	if !results.AllSuccessful() {
		t.Errorf("AllOn() on empty group should succeed")
	}
	if len(results) != 0 {
		t.Errorf("AllOn() on empty group should return 0 results")
	}
}

// TestGroupWithGen2Device tests group operations with Gen2 devices.
func TestGroupWithGen2Device(t *testing.T) {
	ctx := context.Background()

	dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
		resp := `{"jsonrpc":"2.0","id":1,"result":{}}`
		return json.RawMessage(resp), nil
	})

	g := NewGroup("Test", WithDevice(dev))
	results := g.AllOn(ctx)
	if !results.AllSuccessful() {
		t.Errorf("AllOn() failed: %v", results[0].Error)
	}
}

// mockDevice is a simple mock device for testing.
type mockDevice struct {
	addr string
	gen  types.Generation
}

func (d *mockDevice) Address() string {
	return d.addr
}

func (d *mockDevice) Generation() types.Generation {
	if d.gen == 0 {
		return types.GenUnknown
	}
	return d.gen
}
