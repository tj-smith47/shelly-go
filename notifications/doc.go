// Package notifications provides a unified interface for receiving real-time
// events from Shelly devices across all generations and connection methods.
//
// This package builds upon the events package to provide convenient parsers
// for Gen2+ WebSocket notifications and Gen1 CoIoT multicast messages.
//
// # Notification Types
//
// Gen2+ devices send three types of notifications via WebSocket/MQTT:
//
//   - NotifyStatus: Component status changes (partial updates)
//   - NotifyFullStatus: Complete device status (sent on connection)
//   - NotifyEvent: Input/button events (single_push, double_push, etc.)
//
// Gen1 devices send CoIoT multicast messages with device status.
//
// # Parsing Gen2+ Notifications
//
// Use ParseGen2Notification to convert raw WebSocket messages to typed events:
//
//	// WebSocket message received
//	notification := json.RawMessage(`{
//	    "src": "shellyplus1-aabbcc",
//	    "dst": "client",
//	    "method": "NotifyStatus",
//	    "params": {"switch:0": {"output": true}}
//	}`)
//
//	event, err := notifications.ParseGen2Notification("shellyplus1-aabbcc", notification)
//	if err == nil {
//	    switch e := event.(type) {
//	    case *events.StatusChangeEvent:
//	        fmt.Printf("Component %s changed\n", e.Component)
//	    case *events.NotifyEvent:
//	        fmt.Printf("Event %s on %s\n", e.Event, e.Component)
//	    }
//	}
//
// # Integration with EventBus
//
// Parsed events can be published to an EventBus for distribution:
//
//	bus := events.NewEventBus()
//
//	// WebSocket handler
//	device.OnNotification(func(msg json.RawMessage) {
//	    event, err := notifications.ParseGen2Notification(deviceID, msg)
//	    if err == nil {
//	        bus.Publish(event)
//	    }
//	})
//
//	// Subscribe to switch events
//	bus.SubscribeFiltered(
//	    events.WithComponentType("switch"),
//	    func(e events.Event) {
//	        // Handle switch events
//	    },
//	)
//
// # MQTT Integration
//
// For MQTT notifications, subscribe to the device's events topic:
//
//	// Subscribe to: <shelly-id>/events/rpc
//	client.Subscribe("shellyplus1-aabbcc/events/rpc", func(msg []byte) {
//	    event, err := notifications.ParseGen2Notification("shellyplus1-aabbcc", msg)
//	    if err == nil {
//	        bus.Publish(event)
//	    }
//	})
//
// # GenericNotification
//
// For advanced use cases, you can work with the raw notification structure:
//
//	var notif notifications.GenericNotification
//	json.Unmarshal(msg, &notif)
//
//	switch notif.Method {
//	case "NotifyStatus":
//	    // Handle status change
//	case "NotifyFullStatus":
//	    // Handle full status
//	case "NotifyEvent":
//	    // Handle event
//	}
//
// # See Also
//
//   - events package: Event types and EventBus for event distribution
//   - cloud package: Cloud WebSocket for remote device events
//   - gen2 package: Local WebSocket connection to Gen2+ devices
package notifications
