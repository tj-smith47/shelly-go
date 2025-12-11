// Package events provides a typed event system for Shelly device notifications.
//
// The events package implements a publish-subscribe pattern for handling
// real-time events from Shelly devices across all generations (Gen1, Gen2+)
// and connection methods (local, cloud).
//
// # Event Bus
//
// The EventBus is the central hub for event distribution:
//
//	bus := events.NewEventBus()
//	defer bus.Close()
//
//	// Subscribe to all events
//	bus.Subscribe(func(e events.Event) {
//	    fmt.Printf("Event: %s from %s\n", e.Type(), e.DeviceID())
//	})
//
//	// Publish an event
//	bus.Publish(events.NewStatusChangeEvent(deviceID, "switch:0", status))
//
// # Event Types
//
// The package defines typed events for all Shelly notifications:
//
//   - StatusChangeEvent: Component status changed
//   - NotifyFullStatusEvent: Full device status notification
//   - NotifyEvent: Input/button events (single_push, double_push, etc.)
//   - DeviceOnlineEvent: Device came online
//   - DeviceOfflineEvent: Device went offline
//   - UpdateAvailableEvent: Firmware update available
//   - ScriptEvent: Script output event
//
// Each event type provides typed access to event data:
//
//	bus.Subscribe(func(e events.Event) {
//	    if statusEvent, ok := e.(*events.StatusChangeEvent); ok {
//	        fmt.Printf("Component %s changed: %v\n",
//	            statusEvent.Component, statusEvent.Status)
//	    }
//	})
//
// # Filtered Subscriptions
//
// Use filters to receive only relevant events:
//
//	// Only switch events
//	bus.SubscribeFiltered(
//	    events.WithComponentType("switch"),
//	    func(e events.Event) {
//	        // Handle switch events
//	    },
//	)
//
//	// Only events from specific device
//	bus.SubscribeFiltered(
//	    events.WithDeviceID("shellyplus1-aabbcc"),
//	    func(e events.Event) {
//	        // Handle events from this device
//	    },
//	)
//
//	// Combine filters
//	bus.SubscribeFiltered(
//	    events.And(
//	        events.WithDeviceID("shellyplus1-aabbcc"),
//	        events.WithEventType(events.EventTypeStatusChange),
//	    ),
//	    func(e events.Event) {
//	        // Handle specific events
//	    },
//	)
//
// # Handler Registration
//
// For more structured event handling, use the HandlerRegistry:
//
//	registry := events.NewHandlerRegistry(bus)
//
//	// Register typed handlers
//	registry.OnStatusChange(func(e *events.StatusChangeEvent) {
//	    fmt.Printf("Status changed: %s\n", e.Component)
//	})
//
//	registry.OnDeviceOffline(func(e *events.DeviceOfflineEvent) {
//	    fmt.Printf("Device offline: %s\n", e.DeviceID())
//	})
//
// # Integration with Device Connections
//
// Events can be sourced from multiple connection types:
//
//	// Gen2+ WebSocket notifications
//	device.Subscribe(ctx, func(notification json.RawMessage) {
//	    event := events.ParseGen2Notification(deviceID, notification)
//	    bus.Publish(event)
//	})
//
//	// Gen1 CoIoT multicast
//	listener.OnMessage(func(msg coiot.Message) {
//	    event := events.ParseCoIoTMessage(msg)
//	    bus.Publish(event)
//	})
//
//	// Cloud WebSocket events
//	client.OnStatusChange(func(deviceID string, status json.RawMessage) {
//	    event := events.NewStatusChangeEvent(deviceID, "", status)
//	    bus.Publish(event)
//	})
//
// # Thread Safety
//
// The EventBus is fully thread-safe. Subscribers are invoked synchronously
// in the order they were registered. For async processing, subscribers
// should dispatch to their own goroutines.
//
// # Best Practices
//
//   - Close the EventBus when done to release resources
//   - Use filters to reduce unnecessary handler invocations
//   - Keep handlers fast; offload heavy processing to goroutines
//   - Use typed handlers via HandlerRegistry when possible
package events
