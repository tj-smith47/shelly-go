//go:build linux || darwin || windows

package provisioning

import (
	"context"
	"errors"
	"testing"
	"time"

	"tinygo.org/x/bluetooth"
)

// These tests exercise the guard-path branches of tinyGoBLETransmitter
// without requiring a real Bluetooth adapter.  Because both the test file and
// the implementation share package provisioning, we can construct the struct
// directly, bypassing NewTinyGoBLETransmitter (which calls adapter.Enable).

// ── Fake bleConnector ──────────────────────────────────────────────────────────

// fakeBLEConnector implements bleConnector for testing.
type fakeBLEConnector struct {
	connectErr error
	connectFn  func(addr bluetooth.Address, params bluetooth.ConnectionParams) (bluetooth.Device, error)
	// connectDelay lets tests insert a delay so context cancellation fires first.
	connectDelay time.Duration
}

func (f *fakeBLEConnector) Connect(addr bluetooth.Address, params bluetooth.ConnectionParams) (bluetooth.Device, error) {
	if f.connectDelay > 0 {
		time.Sleep(f.connectDelay)
	}
	if f.connectFn != nil {
		return f.connectFn(addr, params)
	}
	if f.connectErr != nil {
		return bluetooth.Device{}, f.connectErr
	}
	return bluetooth.Device{}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// buildDisconnectedTransmitter returns a tinyGoBLETransmitter in the initial
// disconnected state with a live notification channel but no real BLE connector.
func buildDisconnectedTransmitter() *tinyGoBLETransmitter {
	return &tinyGoBLETransmitter{
		notifyCh: make(chan []byte, 10),
	}
}

// buildConnectedTransmitter returns a tinyGoBLETransmitter whose connected
// field is pre-set to true so that guard checks inside Write/Read pass.
func buildConnectedTransmitter() *tinyGoBLETransmitter {
	t := buildDisconnectedTransmitter()
	t.connected = true
	return t
}

// buildTransmitterWithFakeConnector returns a tinyGoBLETransmitter wired to
// the given fake connector so Connect() tests can run without BlueZ.
func buildTransmitterWithFakeConnector(fc *fakeBLEConnector) *tinyGoBLETransmitter {
	return &tinyGoBLETransmitter{
		connector: fc,
		notifyCh:  make(chan []byte, 10),
	}
}

// ── IsConnected ───────────────────────────────────────────────────────────────

func TestTinyGoBLETransmitter_IsConnected_False(t *testing.T) {
	tx := buildDisconnectedTransmitter()
	if tx.IsConnected() {
		t.Error("IsConnected() = true, want false for newly created transmitter")
	}
}

func TestTinyGoBLETransmitter_IsConnected_True(t *testing.T) {
	tx := buildConnectedTransmitter()
	if !tx.IsConnected() {
		t.Error("IsConnected() = false, want true for connected transmitter")
	}
}

// ── Connect: already-connected guard ─────────────────────────────────────────

func TestTinyGoBLETransmitter_Connect_AlreadyConnected(t *testing.T) {
	tx := buildConnectedTransmitter()
	err := tx.Connect(context.Background(), "AA:BB:CC:DD:EE:FF")
	if err == nil {
		t.Fatal("Connect() should fail when already connected")
	}
	if err.Error() != "already connected to a device" {
		t.Errorf("Connect() error = %q, want %q", err.Error(), "already connected to a device")
	}
}

// ── Connect: connector returns error ──────────────────────────────────────────

func TestTinyGoBLETransmitter_Connect_ConnectorError(t *testing.T) {
	fc := &fakeBLEConnector{connectErr: errors.New("no device found")}
	tx := buildTransmitterWithFakeConnector(fc)

	err := tx.Connect(context.Background(), "AA:BB:CC:DD:EE:FF")
	if err == nil {
		t.Fatal("Connect() should fail when connector returns error")
	}
	if err.Error() != "failed to connect: no device found" {
		t.Errorf("Connect() error = %q", err)
	}
	if tx.connected {
		t.Error("connected should be false after connector error")
	}
}

// ── Connect: context canceled before connector returns ────────────────────────

func TestTinyGoBLETransmitter_Connect_ContextCanceled(t *testing.T) {
	// Connector hangs long enough that the pre-canceled context wins the select.
	fc := &fakeBLEConnector{connectDelay: 200 * time.Millisecond}
	tx := buildTransmitterWithFakeConnector(fc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before Connect is called

	err := tx.Connect(ctx, "AA:BB:CC:DD:EE:FF")
	if err == nil {
		t.Fatal("Connect() should fail when context is already canceled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Connect() error = %v, want context.Canceled", err)
	}
	if tx.connected {
		t.Error("connected should be false after context cancellation")
	}
}

// ── Disconnect: not-connected early-return ────────────────────────────────────

func TestTinyGoBLETransmitter_Disconnect_NotConnected(t *testing.T) {
	tx := buildDisconnectedTransmitter()
	err := tx.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() when not connected = %v, want nil", err)
	}
	if tx.connected {
		t.Error("connected should remain false after Disconnect() no-op")
	}
}

// ── WriteCharacteristic: not-connected guard ──────────────────────────────────

func TestTinyGoBLETransmitter_WriteCharacteristic_NotConnected(t *testing.T) {
	tx := buildDisconnectedTransmitter()
	err := tx.WriteCharacteristic(context.Background(), []byte("data"))
	if err == nil {
		t.Fatal("WriteCharacteristic() should fail when not connected")
	}
	if err.Error() != "not connected" {
		t.Errorf("WriteCharacteristic() error = %q, want %q", err.Error(), "not connected")
	}
}

// ── ReadNotification: not-connected guard ─────────────────────────────────────

func TestTinyGoBLETransmitter_ReadNotification_NotConnected(t *testing.T) {
	tx := buildDisconnectedTransmitter()
	_, err := tx.ReadNotification(context.Background())
	if err == nil {
		t.Fatal("ReadNotification() should fail when not connected")
	}
	if err.Error() != "not connected" {
		t.Errorf("ReadNotification() error = %q, want %q", err.Error(), "not connected")
	}
}

// ── ReadNotification: context cancellation while connected ───────────────────

func TestTinyGoBLETransmitter_ReadNotification_ContextCanceled(t *testing.T) {
	tx := buildConnectedTransmitter()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	_, err := tx.ReadNotification(ctx)
	if err == nil {
		t.Fatal("ReadNotification() should fail on canceled context")
	}
}

// ── ReadNotification: notification delivered ──────────────────────────────────

func TestTinyGoBLETransmitter_ReadNotification_DeliversData(t *testing.T) {
	tx := buildConnectedTransmitter()
	want := []byte("hello from shelly")
	tx.notifyCh <- want

	data, err := tx.ReadNotification(context.Background())
	if err != nil {
		t.Fatalf("ReadNotification() error = %v", err)
	}
	if string(data) != string(want) {
		t.Errorf("ReadNotification() = %q, want %q", data, want)
	}
}

// ── ReadNotificationWithTimeout: not-connected ────────────────────────────────

func TestTinyGoBLETransmitter_ReadNotificationWithTimeout_NotConnected(t *testing.T) {
	tx := buildDisconnectedTransmitter()
	_, err := tx.ReadNotificationWithTimeout(50 * time.Millisecond)
	if err == nil {
		t.Fatal("ReadNotificationWithTimeout() should fail when not connected")
	}
}

// ── ReadNotificationWithTimeout: timeout fires ────────────────────────────────

func TestTinyGoBLETransmitter_ReadNotificationWithTimeout_Timeout(t *testing.T) {
	tx := buildConnectedTransmitter()
	_, err := tx.ReadNotificationWithTimeout(30 * time.Millisecond)
	if err == nil {
		t.Fatal("ReadNotificationWithTimeout() should error on timeout")
	}
}

// ── ReadNotificationWithTimeout: notification before timeout ──────────────────

func TestTinyGoBLETransmitter_ReadNotificationWithTimeout_Success(t *testing.T) {
	tx := buildConnectedTransmitter()
	want := []byte("pong")
	tx.notifyCh <- want

	data, err := tx.ReadNotificationWithTimeout(500 * time.Millisecond)
	if err != nil {
		t.Fatalf("ReadNotificationWithTimeout() error = %v", err)
	}
	if string(data) != string(want) {
		t.Errorf("ReadNotificationWithTimeout() = %q, want %q", data, want)
	}
}
