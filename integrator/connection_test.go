package integrator

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// mockWSConnector implements WSConnector for testing.
type mockWSConnector struct {
	writeFunc        func(messageType int, data []byte) error
	readFunc         func() (int, []byte, error)
	closeFunc        func() error
	setReadDeadline  func(t time.Time) error
	setWriteDeadline func(t time.Time) error
	closed           bool
}

func (m *mockWSConnector) WriteMessage(messageType int, data []byte) error {
	if m.writeFunc != nil {
		return m.writeFunc(messageType, data)
	}
	return nil
}

func (m *mockWSConnector) ReadMessage() (int, []byte, error) {
	if m.readFunc != nil {
		return m.readFunc()
	}
	return 0, nil, nil
}

func (m *mockWSConnector) Close() error {
	m.closed = true
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockWSConnector) SetReadDeadline(t time.Time) error {
	if m.setReadDeadline != nil {
		return m.setReadDeadline(t)
	}
	return nil
}

func (m *mockWSConnector) SetWriteDeadline(t time.Time) error {
	if m.setWriteDeadline != nil {
		return m.setWriteDeadline(t)
	}
	return nil
}

func TestConnection_Host(t *testing.T) {
	conn := &Connection{host: "test-host.shelly.cloud"}
	if conn.Host() != "test-host.shelly.cloud" {
		t.Errorf("Host() = %v, want test-host.shelly.cloud", conn.Host())
	}
}

func TestConnection_IsClosed(t *testing.T) {
	conn := &Connection{closeCh: make(chan struct{})}

	if conn.IsClosed() {
		t.Error("IsClosed() = true, want false initially")
	}

	conn.closed = true
	if !conn.IsClosed() {
		t.Error("IsClosed() = false, want true")
	}
}

func TestConnection_Close(t *testing.T) {
	mockWS := &mockWSConnector{}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
	}

	err := conn.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if !conn.IsClosed() {
		t.Error("IsClosed() = false after Close()")
	}

	if !mockWS.closed {
		t.Error("WebSocket not closed")
	}

	// Second close should be no-op
	err = conn.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestConnection_Close_NoWS(t *testing.T) {
	conn := &Connection{
		closeCh: make(chan struct{}),
	}

	err := conn.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestConnection_SendCommand(t *testing.T) {
	var sentData []byte
	mockWS := &mockWSConnector{
		writeFunc: func(messageType int, data []byte) error {
			sentData = data
			return nil
		},
	}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
	}

	err := conn.SendCommand(context.Background(), "device123", "relay", map[string]any{"turn": "on"})
	if err != nil {
		t.Fatalf("SendCommand() error = %v", err)
	}

	var cmd DeviceCommand
	if err := json.Unmarshal(sentData, &cmd); err != nil {
		t.Fatalf("failed to unmarshal sent data: %v", err)
	}

	if cmd.Event != "Integrator:ActionRequest" {
		t.Errorf("Event = %v, want Integrator:ActionRequest", cmd.Event)
	}
	if cmd.DeviceID != "device123" {
		t.Errorf("DeviceID = %v, want device123", cmd.DeviceID)
	}
	if cmd.Action != "relay" {
		t.Errorf("Action = %v, want relay", cmd.Action)
	}
}

func TestConnection_SendCommand_Closed(t *testing.T) {
	conn := &Connection{
		closed:  true,
		closeCh: make(chan struct{}),
	}

	err := conn.SendCommand(context.Background(), "device", "relay", nil)
	if err != ErrConnectionClosed {
		t.Errorf("SendCommand() error = %v, want ErrConnectionClosed", err)
	}
}

func TestConnection_SendCommand_NoWS(t *testing.T) {
	conn := &Connection{
		closeCh: make(chan struct{}),
	}

	err := conn.SendCommand(context.Background(), "device", "relay", nil)
	if err != ErrConnectionClosed {
		t.Errorf("SendCommand() error = %v, want ErrConnectionClosed", err)
	}
}

func TestConnection_SendCommand_WriteError(t *testing.T) {
	mockWS := &mockWSConnector{
		writeFunc: func(messageType int, data []byte) error {
			return errors.New("write failed")
		},
	}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
	}

	err := conn.SendCommand(context.Background(), "device", "relay", nil)
	if err == nil {
		t.Error("SendCommand() expected error")
	}
}

func TestConnection_SendRelayCommand(t *testing.T) {
	var sentData []byte
	mockWS := &mockWSConnector{
		writeFunc: func(messageType int, data []byte) error {
			sentData = data
			return nil
		},
	}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
	}

	// Test on
	err := conn.SendRelayCommand(context.Background(), "device123", 0, true)
	if err != nil {
		t.Fatalf("SendRelayCommand() error = %v", err)
	}

	var cmd DeviceCommand
	_ = json.Unmarshal(sentData, &cmd)
	params := cmd.Params.(map[string]any)
	if params["turn"] != "on" {
		t.Errorf("turn = %v, want on", params["turn"])
	}

	// Test off
	err = conn.SendRelayCommand(context.Background(), "device123", 0, false)
	if err != nil {
		t.Fatalf("SendRelayCommand() error = %v", err)
	}

	_ = json.Unmarshal(sentData, &cmd)
	params = cmd.Params.(map[string]any)
	if params["turn"] != "off" {
		t.Errorf("turn = %v, want off", params["turn"])
	}
}

func TestConnection_SendRollerCommand(t *testing.T) {
	var sentData []byte
	mockWS := &mockWSConnector{
		writeFunc: func(messageType int, data []byte) error {
			sentData = data
			return nil
		},
	}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
	}

	err := conn.SendRollerCommand(context.Background(), "device123", 0, "open")
	if err != nil {
		t.Fatalf("SendRollerCommand() error = %v", err)
	}

	var cmd DeviceCommand
	_ = json.Unmarshal(sentData, &cmd)
	if cmd.Action != "roller" {
		t.Errorf("Action = %v, want roller", cmd.Action)
	}
	params := cmd.Params.(map[string]any)
	if params["go"] != "open" {
		t.Errorf("go = %v, want open", params["go"])
	}
}

func TestConnection_SendRollerPosition(t *testing.T) {
	var sentData []byte
	mockWS := &mockWSConnector{
		writeFunc: func(messageType int, data []byte) error {
			sentData = data
			return nil
		},
	}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
	}

	err := conn.SendRollerPosition(context.Background(), "device123", 0, 50)
	if err != nil {
		t.Fatalf("SendRollerPosition() error = %v", err)
	}

	var cmd DeviceCommand
	_ = json.Unmarshal(sentData, &cmd)
	params := cmd.Params.(map[string]any)
	if params["go"] != "to_pos" {
		t.Errorf("go = %v, want to_pos", params["go"])
	}
	if params["roller_pos"] != float64(50) {
		t.Errorf("roller_pos = %v, want 50", params["roller_pos"])
	}
}

func TestConnection_SendLightCommand(t *testing.T) {
	var sentData []byte
	mockWS := &mockWSConnector{
		writeFunc: func(messageType int, data []byte) error {
			sentData = data
			return nil
		},
	}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
	}

	err := conn.SendLightCommand(context.Background(), "device123", 0, true)
	if err != nil {
		t.Fatalf("SendLightCommand() error = %v", err)
	}

	var cmd DeviceCommand
	_ = json.Unmarshal(sentData, &cmd)
	if cmd.Action != "light" {
		t.Errorf("Action = %v, want light", cmd.Action)
	}
	params := cmd.Params.(map[string]any)
	if params["turn"] != "on" {
		t.Errorf("turn = %v, want on", params["turn"])
	}
}

func TestConnection_VerifyDevice(t *testing.T) {
	var sentData []byte
	mockWS := &mockWSConnector{
		writeFunc: func(messageType int, data []byte) error {
			sentData = data
			return nil
		},
	}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
	}

	err := conn.VerifyDevice(context.Background(), "device123")
	if err != nil {
		t.Fatalf("VerifyDevice() error = %v", err)
	}

	var cmd DeviceCommand
	_ = json.Unmarshal(sentData, &cmd)
	if cmd.Action != "DeviceVerify" {
		t.Errorf("Action = %v, want DeviceVerify", cmd.Action)
	}
}

func TestConnection_GetDeviceSettings(t *testing.T) {
	var sentData []byte
	mockWS := &mockWSConnector{
		writeFunc: func(messageType int, data []byte) error {
			sentData = data
			return nil
		},
	}
	conn := &Connection{
		ws:      mockWS,
		closeCh: make(chan struct{}),
	}

	err := conn.GetDeviceSettings(context.Background(), "device123")
	if err != nil {
		t.Fatalf("GetDeviceSettings() error = %v", err)
	}

	var cmd DeviceCommand
	_ = json.Unmarshal(sentData, &cmd)
	if cmd.Action != "DeviceGetSettings" {
		t.Errorf("Action = %v, want DeviceGetSettings", cmd.Action)
	}
}

func TestConnection_EventHandlers(t *testing.T) {
	conn := &Connection{closeCh: make(chan struct{})}

	// Test OnStatusChange
	conn.OnStatusChange(func(event *StatusChangeEvent) {
		_ = event
	})
	if conn.onStatusChange == nil {
		t.Error("onStatusChange not set")
	}

	// Test OnSettingsChange
	conn.OnSettingsChange(func(event *SettingsChangeEvent) {
		_ = event
	})
	if conn.onSettingsChange == nil {
		t.Error("onSettingsChange not set")
	}

	// Test OnOnlineStatus
	conn.OnOnlineStatus(func(event *OnlineStatusEvent) {
		_ = event
	})
	if conn.onOnlineStatus == nil {
		t.Error("onOnlineStatus not set")
	}

	// Test OnError
	conn.OnError(func(err error) {})
	if conn.onError == nil {
		t.Error("onError not set")
	}

	// Test OnRawMessage
	conn.OnRawMessage(func(msg *WSMessage) {})
	if conn.onRawMessage == nil {
		t.Error("onRawMessage not set")
	}
}

func TestConnection_handleMessage_StatusChange(t *testing.T) {
	conn := &Connection{closeCh: make(chan struct{})}

	var received *StatusChangeEvent
	conn.OnStatusChange(func(event *StatusChangeEvent) {
		received = event
	})

	msg := `{"event":"Shelly:StatusOnChange","device_id":"dev123","status":{"output":true},"ts":1234567890}`
	conn.handleMessage([]byte(msg))

	if received == nil {
		t.Fatal("StatusChangeEvent not received")
	}
	if received.DeviceID != "dev123" {
		t.Errorf("DeviceID = %v, want dev123", received.DeviceID)
	}
}

func TestConnection_handleMessage_Settings(t *testing.T) {
	conn := &Connection{closeCh: make(chan struct{})}

	var received *SettingsChangeEvent
	conn.OnSettingsChange(func(event *SettingsChangeEvent) {
		received = event
	})

	msg := `{"event":"Shelly:Settings","device_id":"dev123","settings":{"name":"test"},"ts":1234567890}`
	conn.handleMessage([]byte(msg))

	if received == nil {
		t.Fatal("SettingsChangeEvent not received")
	}
	if received.DeviceID != "dev123" {
		t.Errorf("DeviceID = %v, want dev123", received.DeviceID)
	}
}

func TestConnection_handleMessage_Online(t *testing.T) {
	conn := &Connection{closeCh: make(chan struct{})}

	var received *OnlineStatusEvent
	conn.OnOnlineStatus(func(event *OnlineStatusEvent) {
		received = event
	})

	msg := `{"event":"Shelly:Online","device_id":"dev123","online":1,"ts":1234567890}`
	conn.handleMessage([]byte(msg))

	if received == nil {
		t.Fatal("OnlineStatusEvent not received")
	}
	if received.DeviceID != "dev123" {
		t.Errorf("DeviceID = %v, want dev123", received.DeviceID)
	}
	if !received.Online {
		t.Error("Online = false, want true")
	}
}

func TestConnection_handleMessage_RawMessage(t *testing.T) {
	conn := &Connection{closeCh: make(chan struct{})}

	var received *WSMessage
	conn.OnRawMessage(func(msg *WSMessage) {
		received = msg
	})

	msg := `{"event":"Shelly:StatusOnChange","device_id":"dev123"}`
	conn.handleMessage([]byte(msg))

	if received == nil {
		t.Fatal("raw message not received")
	}
	if received.Event != "Shelly:StatusOnChange" {
		t.Errorf("Event = %v, want Shelly:StatusOnChange", received.Event)
	}
}

func TestConnection_handleMessage_InvalidJSON(t *testing.T) {
	conn := &Connection{closeCh: make(chan struct{})}

	var errorReceived error
	conn.OnError(func(err error) {
		errorReceived = err
	})

	conn.handleMessage([]byte("invalid json"))

	if errorReceived == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestConnection_handleMessage_NoHandlers(t *testing.T) {
	conn := &Connection{closeCh: make(chan struct{})}

	// Should not panic without handlers
	msg := `{"event":"Shelly:StatusOnChange","device_id":"dev123"}`
	conn.handleMessage([]byte(msg))
}
