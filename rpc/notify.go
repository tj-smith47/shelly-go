package rpc

import (
	"encoding/json"
	"sync"
)

// NotificationHandler is a callback function for handling notifications.
type NotificationHandler func(method string, params json.RawMessage)

// MethodNotificationHandler is a callback function for handling notifications
// for a specific method.
type MethodNotificationHandler func(params json.RawMessage)

// NotificationRouter manages notification subscriptions and routing.
//
// The router is thread-safe and can be used concurrently from multiple
// goroutines.
type NotificationRouter struct {
	methodHandlers map[string][]MethodNotificationHandler
	handlers       []NotificationHandler
	mu             sync.RWMutex
}

// NewNotificationRouter creates a new notification router.
func NewNotificationRouter() *NotificationRouter {
	return &NotificationRouter{
		methodHandlers: make(map[string][]MethodNotificationHandler),
	}
}

// OnNotification registers a global notification handler.
// The handler will be called for all notifications.
//
// Multiple handlers can be registered and they will be called in the
// order they were registered.
func (nr *NotificationRouter) OnNotification(handler NotificationHandler) {
	if handler == nil {
		return
	}

	nr.mu.Lock()
	defer nr.mu.Unlock()

	nr.handlers = append(nr.handlers, handler)
}

// OnNotificationMethod registers a method-specific notification handler.
// The handler will only be called for notifications with the given method.
//
// Multiple handlers can be registered for the same method and they will
// be called in the order they were registered.
func (nr *NotificationRouter) OnNotificationMethod(method string, handler MethodNotificationHandler) {
	if handler == nil || method == "" {
		return
	}

	nr.mu.Lock()
	defer nr.mu.Unlock()

	nr.methodHandlers[method] = append(nr.methodHandlers[method], handler)
}

// RemoveNotificationHandlers removes all global notification handlers.
func (nr *NotificationRouter) RemoveNotificationHandlers() {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	nr.handlers = nil
}

// RemoveMethodHandlers removes all handlers for a specific method.
func (nr *NotificationRouter) RemoveMethodHandlers(method string) {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	delete(nr.methodHandlers, method)
}

// RemoveAllHandlers removes all notification handlers (global and method-specific).
func (nr *NotificationRouter) RemoveAllHandlers() {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	nr.handlers = nil
	nr.methodHandlers = make(map[string][]MethodNotificationHandler)
}

// Route routes a notification to registered handlers.
//
// This method calls all global handlers and all method-specific handlers
// for the notification's method.
//
// Handlers are called synchronously in the order they were registered.
// If a handler panics, the panic is not recovered and will propagate to
// the caller.
func (nr *NotificationRouter) Route(notification *Notification) {
	if notification == nil {
		return
	}

	nr.mu.RLock()
	defer nr.mu.RUnlock()

	// Call global handlers
	for _, handler := range nr.handlers {
		handler(notification.Method, notification.Params)
	}

	// Call method-specific handlers
	if handlers, ok := nr.methodHandlers[notification.Method]; ok {
		for _, handler := range handlers {
			handler(notification.Params)
		}
	}
}

// HasHandlers returns true if there are any registered handlers
// (global or method-specific).
func (nr *NotificationRouter) HasHandlers() bool {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	return len(nr.handlers) > 0 || len(nr.methodHandlers) > 0
}

// HasMethodHandlers returns true if there are any handlers registered
// for the given method.
func (nr *NotificationRouter) HasMethodHandlers(method string) bool {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	handlers, ok := nr.methodHandlers[method]
	return ok && len(handlers) > 0
}

// HandlerCount returns the total number of registered handlers
// (global + method-specific).
func (nr *NotificationRouter) HandlerCount() int {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	count := len(nr.handlers)
	for _, handlers := range nr.methodHandlers {
		count += len(handlers)
	}
	return count
}

// MethodHandlerCount returns the number of handlers registered for
// a specific method.
func (nr *NotificationRouter) MethodHandlerCount(method string) int {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	return len(nr.methodHandlers[method])
}

// Methods returns a list of all methods that have registered handlers.
func (nr *NotificationRouter) Methods() []string {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	methods := make([]string, 0, len(nr.methodHandlers))
	for method := range nr.methodHandlers {
		methods = append(methods, method)
	}
	return methods
}

// RouteRaw parses and routes a notification from raw JSON data.
//
// This is a convenience method that combines ParseNotification and Route.
// If parsing fails, an error is returned and no handlers are called.
func (nr *NotificationRouter) RouteRaw(data []byte) error {
	notification, err := ParseNotification(data)
	if err != nil {
		return err
	}

	nr.Route(notification)
	return nil
}
