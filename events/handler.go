package events

import "sync"

// StatusChangeHandler handles status change events.
type StatusChangeHandler func(*StatusChangeEvent)

// FullStatusHandler handles full status events.
type FullStatusHandler func(*FullStatusEvent)

// NotifyHandler handles notify (input/button) events.
type NotifyHandler func(*NotifyEvent)

// DeviceOnlineHandler handles device online events.
type DeviceOnlineHandler func(*DeviceOnlineEvent)

// DeviceOfflineHandler handles device offline events.
type DeviceOfflineHandler func(*DeviceOfflineEvent)

// UpdateAvailableHandler handles update available events.
type UpdateAvailableHandler func(*UpdateAvailableEvent)

// ScriptHandler handles script events.
type ScriptHandler func(*ScriptEvent)

// ConfigChangeHandler handles config change events.
type ConfigChangeHandler func(*ConfigChangeEvent)

// ErrorHandler handles error events.
type ErrorHandler func(*ErrorEvent)

// HandlerRegistry provides typed event handler registration.
type HandlerRegistry struct {
	bus           *EventBus
	subscriptions []uint64
	mu            sync.Mutex
}

// NewHandlerRegistry creates a new handler registry.
func NewHandlerRegistry(bus *EventBus) *HandlerRegistry {
	return &HandlerRegistry{
		bus:           bus,
		subscriptions: make([]uint64, 0),
	}
}

// track records a subscription ID for later cleanup.
func (r *HandlerRegistry) track(id uint64) {
	r.mu.Lock()
	r.subscriptions = append(r.subscriptions, id)
	r.mu.Unlock()
}

// OnStatusChange registers a handler for status change events.
func (r *HandlerRegistry) OnStatusChange(handler StatusChangeHandler) uint64 {
	id := r.bus.SubscribeFiltered(StatusChange(), func(e Event) {
		if evt, ok := e.(*StatusChangeEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnStatusChangeFor registers a handler for status change events from a specific device.
func (r *HandlerRegistry) OnStatusChangeFor(deviceID string, handler StatusChangeHandler) uint64 {
	filter := And(StatusChange(), WithDeviceID(deviceID))
	id := r.bus.SubscribeFiltered(filter, func(e Event) {
		if evt, ok := e.(*StatusChangeEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnFullStatus registers a handler for full status events.
func (r *HandlerRegistry) OnFullStatus(handler FullStatusHandler) uint64 {
	id := r.bus.SubscribeFiltered(WithEventType(EventTypeFullStatus), func(e Event) {
		if evt, ok := e.(*FullStatusEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnNotify registers a handler for notify (input/button) events.
func (r *HandlerRegistry) OnNotify(handler NotifyHandler) uint64 {
	id := r.bus.SubscribeFiltered(InputEvents(), func(e Event) {
		if evt, ok := e.(*NotifyEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnNotifyFor registers a handler for notify events from a specific device.
func (r *HandlerRegistry) OnNotifyFor(deviceID string, handler NotifyHandler) uint64 {
	filter := And(InputEvents(), WithDeviceID(deviceID))
	id := r.bus.SubscribeFiltered(filter, func(e Event) {
		if evt, ok := e.(*NotifyEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnInputEvent registers a handler for a specific input event type.
func (r *HandlerRegistry) OnInputEvent(event string, handler NotifyHandler) uint64 {
	filter := And(InputEvents(), WithInputEvent(event))
	id := r.bus.SubscribeFiltered(filter, func(e Event) {
		if evt, ok := e.(*NotifyEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnSinglePush registers a handler for single push events.
func (r *HandlerRegistry) OnSinglePush(handler NotifyHandler) uint64 {
	return r.OnInputEvent(InputEventSinglePush, handler)
}

// OnDoublePush registers a handler for double push events.
func (r *HandlerRegistry) OnDoublePush(handler NotifyHandler) uint64 {
	return r.OnInputEvent(InputEventDoublePush, handler)
}

// OnLongPush registers a handler for long push events.
func (r *HandlerRegistry) OnLongPush(handler NotifyHandler) uint64 {
	return r.OnInputEvent(InputEventLongPush, handler)
}

// OnDeviceOnline registers a handler for device online events.
func (r *HandlerRegistry) OnDeviceOnline(handler DeviceOnlineHandler) uint64 {
	id := r.bus.SubscribeFiltered(DeviceOnline(), func(e Event) {
		if evt, ok := e.(*DeviceOnlineEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnDeviceOffline registers a handler for device offline events.
func (r *HandlerRegistry) OnDeviceOffline(handler DeviceOfflineHandler) uint64 {
	id := r.bus.SubscribeFiltered(DeviceOffline(), func(e Event) {
		if evt, ok := e.(*DeviceOfflineEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnUpdateAvailable registers a handler for update available events.
func (r *HandlerRegistry) OnUpdateAvailable(handler UpdateAvailableHandler) uint64 {
	id := r.bus.SubscribeFiltered(WithEventType(EventTypeUpdateAvailable), func(e Event) {
		if evt, ok := e.(*UpdateAvailableEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnScript registers a handler for script events.
func (r *HandlerRegistry) OnScript(handler ScriptHandler) uint64 {
	id := r.bus.SubscribeFiltered(WithEventType(EventTypeScript), func(e Event) {
		if evt, ok := e.(*ScriptEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnConfigChange registers a handler for config change events.
func (r *HandlerRegistry) OnConfigChange(handler ConfigChangeHandler) uint64 {
	id := r.bus.SubscribeFiltered(WithEventType(EventTypeConfig), func(e Event) {
		if evt, ok := e.(*ConfigChangeEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnError registers a handler for error events.
func (r *HandlerRegistry) OnError(handler ErrorHandler) uint64 {
	id := r.bus.SubscribeFiltered(Errors(), func(e Event) {
		if evt, ok := e.(*ErrorEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnComponentType registers a handler for status change events on a component type.
func (r *HandlerRegistry) OnComponentType(componentType string, handler StatusChangeHandler) uint64 {
	filter := And(StatusChange(), WithComponentType(componentType))
	id := r.bus.SubscribeFiltered(filter, func(e Event) {
		if evt, ok := e.(*StatusChangeEvent); ok {
			handler(evt)
		}
	})
	r.track(id)
	return id
}

// OnSwitch registers a handler for switch status change events.
func (r *HandlerRegistry) OnSwitch(handler StatusChangeHandler) uint64 {
	return r.OnComponentType("switch", handler)
}

// OnCover registers a handler for cover status change events.
func (r *HandlerRegistry) OnCover(handler StatusChangeHandler) uint64 {
	return r.OnComponentType("cover", handler)
}

// OnLight registers a handler for light status change events.
func (r *HandlerRegistry) OnLight(handler StatusChangeHandler) uint64 {
	return r.OnComponentType("light", handler)
}

// OnInput registers a handler for input status change events.
func (r *HandlerRegistry) OnInput(handler StatusChangeHandler) uint64 {
	return r.OnComponentType("input", handler)
}

// Unsubscribe removes a specific subscription.
func (r *HandlerRegistry) Unsubscribe(id uint64) bool {
	r.mu.Lock()
	for i, subID := range r.subscriptions {
		if subID == id {
			r.subscriptions[i] = r.subscriptions[len(r.subscriptions)-1]
			r.subscriptions = r.subscriptions[:len(r.subscriptions)-1]
			break
		}
	}
	r.mu.Unlock()
	return r.bus.Unsubscribe(id)
}

// UnsubscribeAll removes all subscriptions registered through this registry.
func (r *HandlerRegistry) UnsubscribeAll() {
	r.mu.Lock()
	subs := make([]uint64, len(r.subscriptions))
	copy(subs, r.subscriptions)
	r.subscriptions = r.subscriptions[:0]
	r.mu.Unlock()

	for _, id := range subs {
		r.bus.Unsubscribe(id)
	}
}

// SubscriptionCount returns the number of subscriptions in this registry.
func (r *HandlerRegistry) SubscriptionCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.subscriptions)
}
