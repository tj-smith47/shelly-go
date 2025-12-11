package gen1

import (
	"testing"
	"time"
)

// TestNewCoIoTListener tests CoIoT listener creation.
func TestNewCoIoTListener(t *testing.T) {
	listener := NewCoIoTListener()

	if listener == nil {
		t.Fatal("expected listener to be created")
	}

	if listener.multicastAddr != CoIoTMulticastAddr {
		t.Errorf("expected multicast addr %s, got %s", CoIoTMulticastAddr, listener.multicastAddr)
	}

	if listener.port != CoIoTPort {
		t.Errorf("expected port %d, got %d", CoIoTPort, listener.port)
	}
}

// TestCoIoTListenerOptions tests listener options.
func TestCoIoTListenerOptions(t *testing.T) {
	listener := NewCoIoTListener(
		WithCoIoTMulticastAddr("224.0.0.1"),
		WithCoIoTPort(5684),
		WithCoIoTBufferSize(2000),
	)

	if listener.multicastAddr != "224.0.0.1" {
		t.Errorf("expected multicast addr 224.0.0.1, got %s", listener.multicastAddr)
	}

	if listener.port != 5684 {
		t.Errorf("expected port 5684, got %d", listener.port)
	}

	if listener.bufferSize != 2000 {
		t.Errorf("expected buffer size 2000, got %d", listener.bufferSize)
	}
}

// TestCoIoTListenerOnStatus tests handler registration.
func TestCoIoTListenerOnStatus(t *testing.T) {
	listener := NewCoIoTListener()

	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {
		// Handler registered
	})

	if len(listener.handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(listener.handlers))
	}

	// Multiple handlers
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {})
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {})

	if len(listener.handlers) != 3 {
		t.Errorf("expected 3 handlers, got %d", len(listener.handlers))
	}
}

// TestCoIoTListenerIsRunning tests running state.
func TestCoIoTListenerIsRunning(t *testing.T) {
	listener := NewCoIoTListener()

	if listener.IsRunning() {
		t.Error("expected not running initially")
	}

	// Note: Can't test Start() without a real network interface
	// but we can test the state management
}

// TestCoIoTListenerStopWithoutStart tests stopping before starting.
func TestCoIoTListenerStopWithoutStart(t *testing.T) {
	listener := NewCoIoTListener()

	err := listener.Stop()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCoIoTStatus tests CoIoTStatus struct.
func TestCoIoTStatus(t *testing.T) {
	status := &CoIoTStatus{
		DeviceID:   "AABBCCDDEEFF",
		DeviceType: "SHSW-1",
		Generation: 1,
		Timestamp:  time.Now(),
		Sensors: map[string]any{
			"0_0": true,
			"0_1": 25.5,
		},
		Actuators: map[string]any{
			"0_0": true,
		},
		Raw: []byte(`test`),
	}

	if status.DeviceID != "AABBCCDDEEFF" {
		t.Errorf("expected device ID AABBCCDDEEFF, got %s", status.DeviceID)
	}

	if status.DeviceType != "SHSW-1" {
		t.Errorf("expected device type SHSW-1, got %s", status.DeviceType)
	}

	if len(status.Sensors) != 2 {
		t.Errorf("expected 2 sensors, got %d", len(status.Sensors))
	}

	if len(status.Actuators) != 1 {
		t.Errorf("expected 1 actuator, got %d", len(status.Actuators))
	}
}

// TestParseCoAPMessage tests CoAP message parsing.
func TestParseCoAPMessage(t *testing.T) {
	listener := NewCoIoTListener()

	// Test message too short
	_, err := listener.parseCoAPMessage([]byte{0x01, 0x02})
	if err == nil {
		t.Error("expected error for short message")
	}

	// Test no payload marker
	_, err = listener.parseCoAPMessage([]byte{0x40, 0x01, 0x00, 0x01, 0x00, 0x00})
	if err == nil {
		t.Error("expected error for no payload marker")
	}

	// Test valid message with JSON payload
	validMsg := append(
		[]byte{0x40, 0x01, 0x00, 0x01, 0xFF}, // CoAP header + payload marker
		[]byte(`{"id":"TEST123","type":"SHSW-1"}`)...,
	)

	status, err := listener.parseCoAPMessage(validMsg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.DeviceID != "TEST123" {
		t.Errorf("expected device ID TEST123, got %s", status.DeviceID)
	}

	if status.DeviceType != "SHSW-1" {
		t.Errorf("expected device type SHSW-1, got %s", status.DeviceType)
	}
}

// TestParseCoAPMessageWithSensors tests parsing sensor data.
func TestParseCoAPMessageWithSensors(t *testing.T) {
	listener := NewCoIoTListener()

	// CoAP header + payload marker + JSON with sensor groups
	msg := append(
		[]byte{0x40, 0x01, 0x00, 0x01, 0xFF},
		[]byte(`{"id":"TEST","G":[[0,0,true],[0,1,25.5],[1,0,false]]}`)...,
	)

	status, err := listener.parseCoAPMessage(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(status.Sensors) != 3 {
		t.Errorf("expected 3 sensors, got %d", len(status.Sensors))
	}

	// Check sensor values
	if v, ok := status.Sensors["0_0"]; !ok || v != true {
		t.Errorf("expected sensor 0_0 = true, got %v", v)
	}

	if v, ok := status.Sensors["0_1"]; !ok || v != 25.5 {
		t.Errorf("expected sensor 0_1 = 25.5, got %v", v)
	}
}

// TestParseCoAPMessageInvalidJSON tests invalid JSON handling.
func TestParseCoAPMessageInvalidJSON(t *testing.T) {
	listener := NewCoIoTListener()

	// CoAP header + payload marker + invalid JSON
	msg := append(
		[]byte{0x40, 0x01, 0x00, 0x01, 0xFF},
		[]byte(`{invalid}`)...,
	)

	status, err := listener.parseCoAPMessage(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fall back to unknown device
	if status.DeviceID != "unknown" {
		t.Errorf("expected device ID unknown, got %s", status.DeviceID)
	}
}

// TestCoIoTDescription tests CoIoTDescription struct.
func TestCoIoTDescription(t *testing.T) {
	desc := &CoIoTDescription{
		DeviceID:   "TEST123",
		DeviceType: "SHSW-25",
		Blocks: []CoIoTBlock{
			{ID: 0, Description: "Relay 0"},
			{ID: 1, Description: "Relay 1"},
		},
		Sensors: []CoIoTSensor{
			{ID: 0, Type: "S", Description: "Power", Unit: "W", Block: 0},
			{ID: 1, Type: "S", Description: "Energy", Unit: "Wh", Block: 0},
		},
	}

	if desc.DeviceID != "TEST123" {
		t.Errorf("expected device ID TEST123, got %s", desc.DeviceID)
	}

	if len(desc.Blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(desc.Blocks))
	}

	if len(desc.Sensors) != 2 {
		t.Errorf("expected 2 sensors, got %d", len(desc.Sensors))
	}

	if desc.Sensors[0].Unit != "W" {
		t.Errorf("expected unit W, got %s", desc.Sensors[0].Unit)
	}
}

// TestCoIoTBlock tests CoIoTBlock struct.
func TestCoIoTBlock(t *testing.T) {
	block := CoIoTBlock{
		ID:          0,
		Description: "Relay",
	}

	if block.ID != 0 {
		t.Errorf("expected ID 0, got %d", block.ID)
	}

	if block.Description != "Relay" {
		t.Errorf("expected description Relay, got %s", block.Description)
	}
}

// TestCoIoTSensor tests CoIoTSensor struct.
func TestCoIoTSensor(t *testing.T) {
	sensor := CoIoTSensor{
		ID:          0,
		Type:        "T",
		Description: "Temperature",
		Unit:        "C",
		Block:       0,
		Links:       []int{1, 2},
	}

	if sensor.Type != "T" {
		t.Errorf("expected type T, got %s", sensor.Type)
	}

	if sensor.Unit != "C" {
		t.Errorf("expected unit C, got %s", sensor.Unit)
	}

	if len(sensor.Links) != 2 {
		t.Errorf("expected 2 links, got %d", len(sensor.Links))
	}
}

// TestGetDeviceDescription tests GetDeviceDescription function.
func TestGetDeviceDescription(t *testing.T) {
	// This should return an error since it requires HTTP
	_, err := GetDeviceDescription("192.168.1.100")
	if err == nil {
		t.Error("expected error")
	}
}

// TestCoIoTConstants tests CoIoT constants.
func TestCoIoTConstants(t *testing.T) {
	if CoIoTMulticastAddr != "224.0.1.187" {
		t.Errorf("expected multicast addr 224.0.1.187, got %s", CoIoTMulticastAddr)
	}

	if CoIoTPort != 5683 {
		t.Errorf("expected port 5683, got %d", CoIoTPort)
	}

	if DefaultCoIoTPeriod != 15 {
		t.Errorf("expected period 15, got %d", DefaultCoIoTPeriod)
	}
}

// TestHandleMessage tests message handling.
func TestHandleMessage(t *testing.T) {
	listener := NewCoIoTListener()

	receivedDeviceID := ""
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {
		receivedDeviceID = deviceID
	})

	// Valid message
	msg := append(
		[]byte{0x40, 0x01, 0x00, 0x01, 0xFF},
		[]byte(`{"id":"DEVICE123","type":"SHSW-1"}`)...,
	)

	listener.handleMessage(msg)

	// Give the goroutine time to run
	time.Sleep(10 * time.Millisecond)

	if receivedDeviceID != "DEVICE123" {
		t.Errorf("expected device ID DEVICE123, got %s", receivedDeviceID)
	}
}

// TestHandleMessageInvalid tests handling invalid messages.
func TestHandleMessageInvalid(t *testing.T) {
	listener := NewCoIoTListener()

	called := false
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {
		called = true
	})

	// Invalid message (too short)
	listener.handleMessage([]byte{0x00, 0x01})

	// Give time
	time.Sleep(10 * time.Millisecond)

	// Should not be called since message is invalid
	if called {
		t.Error("handler should not be called for invalid message")
	}
}

// TestMultipleHandlers tests multiple status handlers.
func TestMultipleHandlers(t *testing.T) {
	listener := NewCoIoTListener()

	count := 0
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {
		count++
	})
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {
		count++
	})
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {
		count++
	})

	// Valid message
	msg := append(
		[]byte{0x40, 0x01, 0x00, 0x01, 0xFF},
		[]byte(`{"id":"TEST"}`)...,
	)

	listener.handleMessage(msg)

	// Give time
	time.Sleep(10 * time.Millisecond)

	if count != 3 {
		t.Errorf("expected 3 handler calls, got %d", count)
	}
}
