package rpc

import (
	"encoding/json"
	"testing"
)

func TestNewNotificationRouter(t *testing.T) {
	nr := NewNotificationRouter()

	if nr == nil {
		t.Fatal("NewNotificationRouter() returned nil")
	}

	if nr.methodHandlers == nil {
		t.Error("methodHandlers should be initialized")
	}

	if nr.HasHandlers() {
		t.Error("new router should not have any handlers")
	}
}

func TestNotificationRouter_OnNotification(t *testing.T) {
	nr := NewNotificationRouter()

	called := false
	handler := func(method string, params json.RawMessage) {
		called = true
	}

	nr.OnNotification(handler)

	if !nr.HasHandlers() {
		t.Error("router should have handlers after registration")
	}

	// Route a notification to trigger the handler
	notif := &Notification{
		JSONRPC: "2.0",
		Method:  "Test",
		Params:  json.RawMessage(`{}`),
	}

	nr.Route(notif)

	if !called {
		t.Error("handler was not called")
	}
}

func TestNotificationRouter_OnNotification_Nil(t *testing.T) {
	nr := NewNotificationRouter()

	// Should not panic with nil handler
	nr.OnNotification(nil)

	if nr.HasHandlers() {
		t.Error("nil handler should not be registered")
	}
}

func TestNotificationRouter_OnNotificationMethod(t *testing.T) {
	nr := NewNotificationRouter()

	var receivedMethod string
	var receivedParams json.RawMessage

	handler := func(params json.RawMessage) {
		receivedMethod = "NotifyStatus"
		receivedParams = params
	}

	nr.OnNotificationMethod("NotifyStatus", handler)

	if !nr.HasMethodHandlers("NotifyStatus") {
		t.Error("router should have method handler after registration")
	}

	// Route a notification with the correct method
	notif := &Notification{
		JSONRPC: "2.0",
		Method:  "NotifyStatus",
		Params:  json.RawMessage(`{"status":"ok"}`),
	}

	nr.Route(notif)

	if receivedMethod != "NotifyStatus" {
		t.Errorf("method = %v, want NotifyStatus", receivedMethod)
	}

	if string(receivedParams) != `{"status":"ok"}` {
		t.Errorf("params = %v, want %v", string(receivedParams), `{"status":"ok"}`)
	}
}

func TestNotificationRouter_OnNotificationMethod_Nil(t *testing.T) {
	nr := NewNotificationRouter()

	// Should not panic with nil handler
	nr.OnNotificationMethod("Test", nil)

	if nr.HasMethodHandlers("Test") {
		t.Error("nil handler should not be registered")
	}
}

func TestNotificationRouter_OnNotificationMethod_EmptyMethod(t *testing.T) {
	nr := NewNotificationRouter()

	handler := func(params json.RawMessage) {}

	// Should not panic with empty method
	nr.OnNotificationMethod("", handler)

	if nr.HasMethodHandlers("") {
		t.Error("handler with empty method should not be registered")
	}
}

func TestNotificationRouter_Route(t *testing.T) {
	nr := NewNotificationRouter()

	var globalCalled bool
	var methodCalled bool

	nr.OnNotification(func(method string, params json.RawMessage) {
		globalCalled = true
	})

	nr.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {
		methodCalled = true
	})

	notif := &Notification{
		JSONRPC: "2.0",
		Method:  "NotifyStatus",
		Params:  json.RawMessage(`{}`),
	}

	nr.Route(notif)

	if !globalCalled {
		t.Error("global handler was not called")
	}

	if !methodCalled {
		t.Error("method handler was not called")
	}
}

func TestNotificationRouter_Route_Nil(t *testing.T) {
	nr := NewNotificationRouter()

	called := false
	nr.OnNotification(func(method string, params json.RawMessage) {
		called = true
	})

	// Should not panic with nil notification
	nr.Route(nil)

	if called {
		t.Error("handler should not be called for nil notification")
	}
}

func TestNotificationRouter_Route_NoMatchingMethod(t *testing.T) {
	nr := NewNotificationRouter()

	var methodCalled bool

	nr.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {
		methodCalled = true
	})

	// Route a notification with a different method
	notif := &Notification{
		JSONRPC: "2.0",
		Method:  "OtherMethod",
		Params:  json.RawMessage(`{}`),
	}

	nr.Route(notif)

	if methodCalled {
		t.Error("method handler should not be called for non-matching method")
	}
}

func TestNotificationRouter_RemoveNotificationHandlers(t *testing.T) {
	nr := NewNotificationRouter()

	nr.OnNotification(func(method string, params json.RawMessage) {})
	nr.OnNotification(func(method string, params json.RawMessage) {})

	if !nr.HasHandlers() {
		t.Error("router should have handlers")
	}

	nr.RemoveNotificationHandlers()

	// Method handlers should still be present if any were added
	if len(nr.handlers) > 0 {
		t.Error("global handlers should be removed")
	}
}

func TestNotificationRouter_RemoveMethodHandlers(t *testing.T) {
	nr := NewNotificationRouter()

	nr.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {})
	nr.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {})

	if !nr.HasMethodHandlers("NotifyStatus") {
		t.Error("router should have method handlers")
	}

	nr.RemoveMethodHandlers("NotifyStatus")

	if nr.HasMethodHandlers("NotifyStatus") {
		t.Error("method handlers should be removed")
	}
}

func TestNotificationRouter_RemoveAllHandlers(t *testing.T) {
	nr := NewNotificationRouter()

	nr.OnNotification(func(method string, params json.RawMessage) {})
	nr.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {})

	if !nr.HasHandlers() {
		t.Error("router should have handlers")
	}

	nr.RemoveAllHandlers()

	if nr.HasHandlers() {
		t.Error("all handlers should be removed")
	}
}

func TestNotificationRouter_HasHandlers(t *testing.T) {
	nr := NewNotificationRouter()

	if nr.HasHandlers() {
		t.Error("new router should not have handlers")
	}

	nr.OnNotification(func(method string, params json.RawMessage) {})

	if !nr.HasHandlers() {
		t.Error("router should have handlers after registration")
	}

	nr.RemoveAllHandlers()

	nr.OnNotificationMethod("Test", func(params json.RawMessage) {})

	if !nr.HasHandlers() {
		t.Error("router should have handlers after method registration")
	}
}

func TestNotificationRouter_HasMethodHandlers(t *testing.T) {
	nr := NewNotificationRouter()

	if nr.HasMethodHandlers("NotifyStatus") {
		t.Error("router should not have method handlers initially")
	}

	nr.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {})

	if !nr.HasMethodHandlers("NotifyStatus") {
		t.Error("router should have method handlers after registration")
	}

	if nr.HasMethodHandlers("OtherMethod") {
		t.Error("router should not have handlers for non-registered method")
	}
}

func TestNotificationRouter_HandlerCount(t *testing.T) {
	nr := NewNotificationRouter()

	if nr.HandlerCount() != 0 {
		t.Errorf("handler count = %v, want 0", nr.HandlerCount())
	}

	nr.OnNotification(func(method string, params json.RawMessage) {})
	nr.OnNotification(func(method string, params json.RawMessage) {})
	nr.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {})

	if nr.HandlerCount() != 3 {
		t.Errorf("handler count = %v, want 3", nr.HandlerCount())
	}
}

func TestNotificationRouter_MethodHandlerCount(t *testing.T) {
	nr := NewNotificationRouter()

	if nr.MethodHandlerCount("NotifyStatus") != 0 {
		t.Errorf("method handler count = %v, want 0", nr.MethodHandlerCount("NotifyStatus"))
	}

	nr.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {})
	nr.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {})

	if nr.MethodHandlerCount("NotifyStatus") != 2 {
		t.Errorf("method handler count = %v, want 2", nr.MethodHandlerCount("NotifyStatus"))
	}

	if nr.MethodHandlerCount("OtherMethod") != 0 {
		t.Errorf("method handler count for non-registered method = %v, want 0",
			nr.MethodHandlerCount("OtherMethod"))
	}
}

func TestNotificationRouter_Methods(t *testing.T) {
	nr := NewNotificationRouter()

	methods := nr.Methods()
	if len(methods) != 0 {
		t.Errorf("methods count = %v, want 0", len(methods))
	}

	nr.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {})
	nr.OnNotificationMethod("NotifyEvent", func(params json.RawMessage) {})

	methods = nr.Methods()
	if len(methods) != 2 {
		t.Errorf("methods count = %v, want 2", len(methods))
	}

	// Check that both methods are present
	hasNotifyStatus := false
	hasNotifyEvent := false
	for _, m := range methods {
		if m == "NotifyStatus" {
			hasNotifyStatus = true
		}
		if m == "NotifyEvent" {
			hasNotifyEvent = true
		}
	}

	if !hasNotifyStatus || !hasNotifyEvent {
		t.Error("methods list is missing registered methods")
	}
}

func TestNotificationRouter_RouteRaw(t *testing.T) {
	nr := NewNotificationRouter()

	var receivedMethod string

	nr.OnNotification(func(method string, params json.RawMessage) {
		receivedMethod = method
	})

	data := []byte(`{"jsonrpc":"2.0","method":"NotifyStatus","params":{"status":"ok"}}`)

	err := nr.RouteRaw(data)
	if err != nil {
		t.Fatalf("RouteRaw() error = %v", err)
	}

	if receivedMethod != "NotifyStatus" {
		t.Errorf("received method = %v, want NotifyStatus", receivedMethod)
	}
}

func TestNotificationRouter_RouteRaw_Invalid(t *testing.T) {
	nr := NewNotificationRouter()

	called := false
	nr.OnNotification(func(method string, params json.RawMessage) {
		called = true
	})

	data := []byte(`{invalid}`)

	err := nr.RouteRaw(data)
	if err == nil {
		t.Error("RouteRaw() should return error for invalid JSON")
	}

	if called {
		t.Error("handler should not be called for invalid notification")
	}
}

func TestNotificationRouter_MultipleHandlers(t *testing.T) {
	nr := NewNotificationRouter()

	callCount := 0

	nr.OnNotification(func(method string, params json.RawMessage) {
		callCount++
	})

	nr.OnNotification(func(method string, params json.RawMessage) {
		callCount++
	})

	notif := &Notification{
		JSONRPC: "2.0",
		Method:  "Test",
		Params:  json.RawMessage(`{}`),
	}

	nr.Route(notif)

	if callCount != 2 {
		t.Errorf("handler call count = %v, want 2", callCount)
	}
}

func TestNotificationRouter_ConcurrentAccess(t *testing.T) {
	nr := NewNotificationRouter()
	done := make(chan bool)

	// Register handlers concurrently
	for i := 0; i < 10; i++ {
		go func() {
			nr.OnNotification(func(method string, params json.RawMessage) {})
			done <- true
		}()
	}

	// Wait for all registrations
	for i := 0; i < 10; i++ {
		<-done
	}

	// Route notifications concurrently
	for i := 0; i < 10; i++ {
		go func() {
			notif := &Notification{
				JSONRPC: "2.0",
				Method:  "Test",
				Params:  json.RawMessage(`{}`),
			}
			nr.Route(notif)
			done <- true
		}()
	}

	// Wait for all routes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 10 handlers registered
	if nr.HandlerCount() != 10 {
		t.Errorf("handler count = %v, want 10", nr.HandlerCount())
	}
}
