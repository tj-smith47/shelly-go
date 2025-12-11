package helpers

import (
	"context"
	"sync"

	"github.com/tj-smith47/shelly-go/factory"
)

// Group represents a named collection of devices that can be operated on together.
//
// Groups provide a convenient way to organize devices by location (e.g., "Living Room")
// or function (e.g., "All Lights") and perform batch operations on them.
type Group struct {
	name    string
	devices []factory.Device
	mu      sync.RWMutex
}

// GroupOption is a functional option for configuring a Group.
type GroupOption func(*Group)

// NewGroup creates a new device group with the given name and options.
//
// Example:
//
//	group := helpers.NewGroup("Kitchen Lights",
//	    helpers.WithDevice(light1),
//	    helpers.WithDevice(light2),
//	)
func NewGroup(name string, opts ...GroupOption) *Group {
	g := &Group{
		name:    name,
		devices: make([]factory.Device, 0),
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// WithDevice adds a device to the group during construction.
func WithDevice(d factory.Device) GroupOption {
	return func(g *Group) {
		g.devices = append(g.devices, d)
	}
}

// WithDevices adds multiple devices to the group during construction.
func WithDevices(devices ...factory.Device) GroupOption {
	return func(g *Group) {
		g.devices = append(g.devices, devices...)
	}
}

// Name returns the group name.
func (g *Group) Name() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.name
}

// SetName updates the group name.
func (g *Group) SetName(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.name = name
}

// Devices returns a copy of the devices in the group.
func (g *Group) Devices() []factory.Device {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]factory.Device, len(g.devices))
	copy(result, g.devices)
	return result
}

// Len returns the number of devices in the group.
func (g *Group) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.devices)
}

// Add adds a device to the group.
func (g *Group) Add(d factory.Device) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.devices = append(g.devices, d)
}

// AddAll adds multiple devices to the group.
func (g *Group) AddAll(devices ...factory.Device) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.devices = append(g.devices, devices...)
}

// Remove removes a device from the group by address.
// Returns true if a device was removed.
func (g *Group) Remove(address string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, d := range g.devices {
		if d.Address() == address {
			g.devices = append(g.devices[:i], g.devices[i+1:]...)
			return true
		}
	}
	return false
}

// Clear removes all devices from the group.
func (g *Group) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.devices = g.devices[:0]
}

// Contains returns true if the group contains a device with the given address.
func (g *Group) Contains(address string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, d := range g.devices {
		if d.Address() == address {
			return true
		}
	}
	return false
}

// AllOn turns on all switch devices in the group.
func (g *Group) AllOn(ctx context.Context) BatchResults {
	return AllOn(ctx, g.Devices())
}

// AllOff turns off all switch devices in the group.
func (g *Group) AllOff(ctx context.Context) BatchResults {
	return AllOff(ctx, g.Devices())
}

// Toggle toggles all switch devices in the group.
func (g *Group) Toggle(ctx context.Context) BatchResults {
	return BatchToggle(ctx, g.Devices())
}

// Set sets the on/off state for all switch devices in the group.
func (g *Group) Set(ctx context.Context, on bool) BatchResults {
	return BatchSet(ctx, g.Devices(), on)
}

// SetBrightness sets the brightness for all light devices in the group.
func (g *Group) SetBrightness(ctx context.Context, brightness int) BatchResults {
	return BatchSetBrightness(ctx, g.Devices(), brightness)
}

// ForEach executes a function for each device in the group.
// The function receives the device and can return an error to stop iteration.
func (g *Group) ForEach(fn func(factory.Device) error) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, d := range g.devices {
		if err := fn(d); err != nil {
			return err
		}
	}
	return nil
}

// Filter returns a new group containing only devices that match the predicate.
func (g *Group) Filter(predicate func(factory.Device) bool) *Group {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := NewGroup(g.name + " (filtered)")
	for _, d := range g.devices {
		if predicate(d) {
			result.Add(d)
		}
	}
	return result
}
