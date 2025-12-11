package events

import (
	"sync/atomic"
	"testing"
)

func TestNewHandlerRegistry(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	registry := NewHandlerRegistry(bus)
	if registry == nil {
		t.Fatal("NewHandlerRegistry() returned nil")
	}
	if registry.SubscriptionCount() != 0 {
		t.Errorf("SubscriptionCount() = %v, want 0", registry.SubscriptionCount())
	}
}

func TestHandlerRegistry_OnStatusChange(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var received *StatusChangeEvent
	registry.OnStatusChange(func(e *StatusChangeEvent) {
		received = e
	})

	bus.Publish(NewStatusChangeEvent("device1", "switch:0", nil))
	bus.Publish(NewDeviceOnlineEvent("device1")) // Should not trigger

	if received == nil {
		t.Error("handler was not called")
	}
	if received.Component != "switch:0" {
		t.Errorf("Component = %v, want switch:0", received.Component)
	}
}

func TestHandlerRegistry_OnStatusChangeFor(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var count int
	registry.OnStatusChangeFor("device1", func(e *StatusChangeEvent) {
		count++
	})

	bus.Publish(NewStatusChangeEvent("device1", "switch:0", nil))
	bus.Publish(NewStatusChangeEvent("device2", "switch:0", nil)) // Should not trigger
	bus.Publish(NewStatusChangeEvent("device1", "switch:1", nil))

	if count != 2 {
		t.Errorf("count = %v, want 2", count)
	}
}

func TestHandlerRegistry_OnFullStatus(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var received *FullStatusEvent
	registry.OnFullStatus(func(e *FullStatusEvent) {
		received = e
	})

	bus.Publish(NewFullStatusEvent("device1", nil))
	bus.Publish(NewStatusChangeEvent("device1", "switch:0", nil)) // Should not trigger

	if received == nil {
		t.Error("handler was not called")
	}
	if received.DeviceID() != "device1" {
		t.Errorf("DeviceID() = %v, want device1", received.DeviceID())
	}
}

func TestHandlerRegistry_OnNotify(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var received *NotifyEvent
	registry.OnNotify(func(e *NotifyEvent) {
		received = e
	})

	bus.Publish(NewNotifyEvent("device1", "input:0", InputEventSinglePush))
	bus.Publish(NewStatusChangeEvent("device1", "input:0", nil)) // Should not trigger

	if received == nil {
		t.Error("handler was not called")
	}
	if received.Event != InputEventSinglePush {
		t.Errorf("Event = %v, want %v", received.Event, InputEventSinglePush)
	}
}

func TestHandlerRegistry_OnNotifyFor(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var count int
	registry.OnNotifyFor("device1", func(e *NotifyEvent) {
		count++
	})

	bus.Publish(NewNotifyEvent("device1", "input:0", InputEventSinglePush))
	bus.Publish(NewNotifyEvent("device2", "input:0", InputEventSinglePush)) // Should not trigger
	bus.Publish(NewNotifyEvent("device1", "input:1", InputEventDoublePush))

	if count != 2 {
		t.Errorf("count = %v, want 2", count)
	}
}

func TestHandlerRegistry_OnInputEvent(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var count int
	registry.OnInputEvent(InputEventDoublePush, func(e *NotifyEvent) {
		count++
	})

	bus.Publish(NewNotifyEvent("device1", "input:0", InputEventSinglePush)) // Should not trigger
	bus.Publish(NewNotifyEvent("device1", "input:0", InputEventDoublePush))
	bus.Publish(NewNotifyEvent("device1", "input:1", InputEventDoublePush))

	if count != 2 {
		t.Errorf("count = %v, want 2", count)
	}
}

func TestHandlerRegistry_OnSinglePush(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var called bool
	registry.OnSinglePush(func(e *NotifyEvent) {
		called = true
	})

	bus.Publish(NewNotifyEvent("device1", "input:0", InputEventSinglePush))

	if !called {
		t.Error("handler was not called")
	}
}

func TestHandlerRegistry_OnDoublePush(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var called bool
	registry.OnDoublePush(func(e *NotifyEvent) {
		called = true
	})

	bus.Publish(NewNotifyEvent("device1", "input:0", InputEventDoublePush))

	if !called {
		t.Error("handler was not called")
	}
}

func TestHandlerRegistry_OnLongPush(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var called bool
	registry.OnLongPush(func(e *NotifyEvent) {
		called = true
	})

	bus.Publish(NewNotifyEvent("device1", "input:0", InputEventLongPush))

	if !called {
		t.Error("handler was not called")
	}
}

func TestHandlerRegistry_OnDeviceOnline(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var received *DeviceOnlineEvent
	registry.OnDeviceOnline(func(e *DeviceOnlineEvent) {
		received = e
	})

	bus.Publish(NewDeviceOnlineEvent("device1").WithAddress("192.168.1.100"))
	bus.Publish(NewDeviceOfflineEvent("device1")) // Should not trigger

	if received == nil {
		t.Error("handler was not called")
	}
	if received.Address != "192.168.1.100" {
		t.Errorf("Address = %v, want 192.168.1.100", received.Address)
	}
}

func TestHandlerRegistry_OnDeviceOffline(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var received *DeviceOfflineEvent
	registry.OnDeviceOffline(func(e *DeviceOfflineEvent) {
		received = e
	})

	bus.Publish(NewDeviceOfflineEvent("device1").WithReason("timeout"))
	bus.Publish(NewDeviceOnlineEvent("device1")) // Should not trigger

	if received == nil {
		t.Error("handler was not called")
	}
	if received.Reason != "timeout" {
		t.Errorf("Reason = %v, want timeout", received.Reason)
	}
}

func TestHandlerRegistry_OnUpdateAvailable(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var received *UpdateAvailableEvent
	registry.OnUpdateAvailable(func(e *UpdateAvailableEvent) {
		received = e
	})

	bus.Publish(NewUpdateAvailableEvent("device1", "1.0.0", "1.1.0"))
	bus.Publish(NewDeviceOnlineEvent("device1")) // Should not trigger

	if received == nil {
		t.Error("handler was not called")
	}
	if received.AvailableVersion != "1.1.0" {
		t.Errorf("AvailableVersion = %v, want 1.1.0", received.AvailableVersion)
	}
}

func TestHandlerRegistry_OnScript(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var received *ScriptEvent
	registry.OnScript(func(e *ScriptEvent) {
		received = e
	})

	bus.Publish(NewScriptEvent("device1", 1, "hello"))
	bus.Publish(NewDeviceOnlineEvent("device1")) // Should not trigger

	if received == nil {
		t.Error("handler was not called")
	}
	if received.Output != "hello" {
		t.Errorf("Output = %v, want hello", received.Output)
	}
}

func TestHandlerRegistry_OnConfigChange(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var received *ConfigChangeEvent
	registry.OnConfigChange(func(e *ConfigChangeEvent) {
		received = e
	})

	bus.Publish(NewConfigChangeEvent("device1", "switch:0", nil))
	bus.Publish(NewStatusChangeEvent("device1", "switch:0", nil)) // Should not trigger

	if received == nil {
		t.Error("handler was not called")
	}
	if received.Component != "switch:0" {
		t.Errorf("Component = %v, want switch:0", received.Component)
	}
}

func TestHandlerRegistry_OnError(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var received *ErrorEvent
	registry.OnError(func(e *ErrorEvent) {
		received = e
	})

	bus.Publish(NewErrorEvent("device1", 500, "internal error"))
	bus.Publish(NewDeviceOnlineEvent("device1")) // Should not trigger

	if received == nil {
		t.Error("handler was not called")
	}
	if received.Code != 500 {
		t.Errorf("Code = %v, want 500", received.Code)
	}
}

func TestHandlerRegistry_OnComponentType(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var count int
	registry.OnComponentType("switch", func(e *StatusChangeEvent) {
		count++
	})

	bus.Publish(NewStatusChangeEvent("device1", "switch:0", nil))
	bus.Publish(NewStatusChangeEvent("device1", "switch:1", nil))
	bus.Publish(NewStatusChangeEvent("device1", "cover:0", nil)) // Should not trigger

	if count != 2 {
		t.Errorf("count = %v, want 2", count)
	}
}

func TestHandlerRegistry_OnSwitch(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var called bool
	registry.OnSwitch(func(e *StatusChangeEvent) {
		called = true
	})

	bus.Publish(NewStatusChangeEvent("device1", "switch:0", nil))

	if !called {
		t.Error("handler was not called")
	}
}

func TestHandlerRegistry_OnCover(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var called bool
	registry.OnCover(func(e *StatusChangeEvent) {
		called = true
	})

	bus.Publish(NewStatusChangeEvent("device1", "cover:0", nil))

	if !called {
		t.Error("handler was not called")
	}
}

func TestHandlerRegistry_OnLight(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var called bool
	registry.OnLight(func(e *StatusChangeEvent) {
		called = true
	})

	bus.Publish(NewStatusChangeEvent("device1", "light:0", nil))

	if !called {
		t.Error("handler was not called")
	}
}

func TestHandlerRegistry_OnInput(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var called bool
	registry.OnInput(func(e *StatusChangeEvent) {
		called = true
	})

	bus.Publish(NewStatusChangeEvent("device1", "input:0", nil))

	if !called {
		t.Error("handler was not called")
	}
}

func TestHandlerRegistry_Unsubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	var count int
	id := registry.OnStatusChange(func(e *StatusChangeEvent) {
		count++
	})

	bus.Publish(NewStatusChangeEvent("device1", "switch:0", nil))
	if count != 1 {
		t.Errorf("count = %v, want 1", count)
	}

	if !registry.Unsubscribe(id) {
		t.Error("Unsubscribe() returned false")
	}
	if registry.SubscriptionCount() != 0 {
		t.Errorf("SubscriptionCount() = %v, want 0", registry.SubscriptionCount())
	}

	bus.Publish(NewStatusChangeEvent("device1", "switch:0", nil))
	if count != 1 {
		t.Errorf("count after unsubscribe = %v, want 1", count)
	}
}

func TestHandlerRegistry_Unsubscribe_NotFound(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	if registry.Unsubscribe(999) {
		t.Error("Unsubscribe() should return false for unknown ID")
	}
}

func TestHandlerRegistry_UnsubscribeAll(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	registry.OnStatusChange(func(e *StatusChangeEvent) {})
	registry.OnDeviceOnline(func(e *DeviceOnlineEvent) {})
	registry.OnDeviceOffline(func(e *DeviceOfflineEvent) {})

	if registry.SubscriptionCount() != 3 {
		t.Errorf("SubscriptionCount() = %v, want 3", registry.SubscriptionCount())
	}

	registry.UnsubscribeAll()

	if registry.SubscriptionCount() != 0 {
		t.Errorf("SubscriptionCount() after UnsubscribeAll = %v, want 0", registry.SubscriptionCount())
	}
	if bus.SubscriberCount() != 0 {
		t.Errorf("bus.SubscriberCount() = %v, want 0", bus.SubscriberCount())
	}
}

func TestHandlerRegistry_SubscriptionCount(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()
	registry := NewHandlerRegistry(bus)

	if registry.SubscriptionCount() != 0 {
		t.Errorf("SubscriptionCount() = %v, want 0", registry.SubscriptionCount())
	}

	id1 := registry.OnStatusChange(func(e *StatusChangeEvent) {})
	if registry.SubscriptionCount() != 1 {
		t.Errorf("SubscriptionCount() = %v, want 1", registry.SubscriptionCount())
	}

	registry.OnDeviceOnline(func(e *DeviceOnlineEvent) {})
	if registry.SubscriptionCount() != 2 {
		t.Errorf("SubscriptionCount() = %v, want 2", registry.SubscriptionCount())
	}

	registry.Unsubscribe(id1)
	if registry.SubscriptionCount() != 1 {
		t.Errorf("SubscriptionCount() after unsubscribe = %v, want 1", registry.SubscriptionCount())
	}
}

func TestHandlerRegistry_MultipleRegistries(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	registry1 := NewHandlerRegistry(bus)
	registry2 := NewHandlerRegistry(bus)

	var count1, count2 atomic.Int32

	registry1.OnStatusChange(func(e *StatusChangeEvent) {
		count1.Add(1)
	})
	registry2.OnStatusChange(func(e *StatusChangeEvent) {
		count2.Add(1)
	})

	bus.Publish(NewStatusChangeEvent("device1", "switch:0", nil))

	if count1.Load() != 1 {
		t.Errorf("count1 = %v, want 1", count1.Load())
	}
	if count2.Load() != 1 {
		t.Errorf("count2 = %v, want 1", count2.Load())
	}

	// Unsubscribe registry1, registry2 should still receive
	registry1.UnsubscribeAll()
	bus.Publish(NewStatusChangeEvent("device1", "switch:0", nil))

	if count1.Load() != 1 {
		t.Errorf("count1 after unsubscribe = %v, want 1", count1.Load())
	}
	if count2.Load() != 2 {
		t.Errorf("count2 after unsubscribe = %v, want 2", count2.Load())
	}
}
