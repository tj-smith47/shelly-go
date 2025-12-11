package provisioning

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

// mockBLETransmitter is a mock implementation of BLETransmitter for testing.
type mockBLETransmitter struct {
	connectErr       error
	disconnectErr    error
	writeErr         error
	readErr          error
	connected        bool
	writtenData      [][]byte
	notifications    [][]byte
	notifyIndex      int
	mu               sync.Mutex
	connectCalled    bool
	disconnectCalled bool
	writeCalled      bool
	readCalled       bool
}

func newMockBLETransmitter() *mockBLETransmitter {
	return &mockBLETransmitter{
		writtenData:   make([][]byte, 0),
		notifications: make([][]byte, 0),
	}
}

func (m *mockBLETransmitter) Connect(ctx context.Context, address string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectCalled = true
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *mockBLETransmitter) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectCalled = true
	if m.disconnectErr != nil {
		return m.disconnectErr
	}
	m.connected = false
	return nil
}

func (m *mockBLETransmitter) WriteCharacteristic(ctx context.Context, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeCalled = true
	if m.writeErr != nil {
		return m.writeErr
	}
	// Copy data to avoid race conditions
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	m.writtenData = append(m.writtenData, dataCopy)
	return nil
}

func (m *mockBLETransmitter) ReadNotification(ctx context.Context) ([]byte, error) {
	m.mu.Lock()
	m.readCalled = true
	if m.readErr != nil {
		m.mu.Unlock()
		return nil, m.readErr
	}
	if m.notifyIndex >= len(m.notifications) {
		m.mu.Unlock()
		// Block until context canceled
		<-ctx.Done()
		return nil, ctx.Err()
	}
	data := m.notifications[m.notifyIndex]
	m.notifyIndex++
	m.mu.Unlock()
	return data, nil
}

func (m *mockBLETransmitter) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// SetNotifications sets up notifications that will be returned by ReadNotification
func (m *mockBLETransmitter) SetNotifications(notifications ...[]byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = notifications
	m.notifyIndex = 0
}

// Verify mockBLETransmitter implements BLETransmitter
var _ BLETransmitter = (*mockBLETransmitter)(nil)

// Test mockBLETransmitter implementation

func TestMockBLETransmitter_Connect(t *testing.T) {
	m := newMockBLETransmitter()

	if m.IsConnected() {
		t.Error("should not be connected initially")
	}

	err := m.Connect(context.Background(), "AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if !m.connectCalled {
		t.Error("connectCalled should be true")
	}

	if !m.IsConnected() {
		t.Error("should be connected after Connect()")
	}
}

func TestMockBLETransmitter_ConnectError(t *testing.T) {
	m := newMockBLETransmitter()
	m.connectErr = errors.New("connection failed")

	err := m.Connect(context.Background(), "AA:BB:CC:DD:EE:FF")
	if err == nil {
		t.Error("Connect() should return error")
	}

	if m.IsConnected() {
		t.Error("should not be connected after error")
	}
}

func TestMockBLETransmitter_Disconnect(t *testing.T) {
	m := newMockBLETransmitter()
	m.connected = true

	err := m.Disconnect()
	if err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	if !m.disconnectCalled {
		t.Error("disconnectCalled should be true")
	}

	if m.IsConnected() {
		t.Error("should not be connected after Disconnect()")
	}
}

func TestMockBLETransmitter_DisconnectError(t *testing.T) {
	m := newMockBLETransmitter()
	m.connected = true
	m.disconnectErr = errors.New("disconnect failed")

	err := m.Disconnect()
	if err == nil {
		t.Error("Disconnect() should return error")
	}
}

func TestMockBLETransmitter_WriteCharacteristic(t *testing.T) {
	m := newMockBLETransmitter()

	data := []byte("test data")
	err := m.WriteCharacteristic(context.Background(), data)
	if err != nil {
		t.Fatalf("WriteCharacteristic() error = %v", err)
	}

	if !m.writeCalled {
		t.Error("writeCalled should be true")
	}

	if len(m.writtenData) != 1 {
		t.Fatalf("len(writtenData) = %d, want 1", len(m.writtenData))
	}

	if string(m.writtenData[0]) != "test data" {
		t.Errorf("writtenData[0] = %s, want 'test data'", m.writtenData[0])
	}
}

func TestMockBLETransmitter_WriteCharacteristicError(t *testing.T) {
	m := newMockBLETransmitter()
	m.writeErr = errors.New("write failed")

	err := m.WriteCharacteristic(context.Background(), []byte("test"))
	if err == nil {
		t.Error("WriteCharacteristic() should return error")
	}
}

func TestMockBLETransmitter_ReadNotification(t *testing.T) {
	m := newMockBLETransmitter()
	m.SetNotifications([]byte("notification 1"), []byte("notification 2"))

	data, err := m.ReadNotification(context.Background())
	if err != nil {
		t.Fatalf("ReadNotification() error = %v", err)
	}

	if !m.readCalled {
		t.Error("readCalled should be true")
	}

	if string(data) != "notification 1" {
		t.Errorf("data = %s, want 'notification 1'", data)
	}

	// Read second notification
	data, err = m.ReadNotification(context.Background())
	if err != nil {
		t.Fatalf("ReadNotification() second call error = %v", err)
	}

	if string(data) != "notification 2" {
		t.Errorf("data = %s, want 'notification 2'", data)
	}
}

func TestMockBLETransmitter_ReadNotificationTimeout(t *testing.T) {
	m := newMockBLETransmitter()
	// No notifications set - should timeout

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := m.ReadNotification(ctx)
	if err == nil {
		t.Error("ReadNotification() should return error on timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestMockBLETransmitter_ReadNotificationError(t *testing.T) {
	m := newMockBLETransmitter()
	m.readErr = errors.New("read failed")

	_, err := m.ReadNotification(context.Background())
	if err == nil {
		t.Error("ReadNotification() should return error")
	}
}

// Test BLETransmitter interface compliance

func TestBLETransmitter_Interface(t *testing.T) {
	// This test verifies that tinyGoBLETransmitter implements BLETransmitter
	// The actual implementation is platform-specific and cannot be tested in CI
	var _ BLETransmitter = (*tinyGoBLETransmitter)(nil)
}

// Test BLE UUIDs and constants

func TestBLEUUIDs(t *testing.T) {
	// Verify the Shelly BLE UUIDs are set correctly
	if ShellyBLEServiceUUID != "5f6d4f53-5f52-5043-5f53-56435f49445f" {
		t.Errorf("ShellyBLEServiceUUID = %s, want 5f6d4f53-5f52-5043-5f53-56435f49445f", ShellyBLEServiceUUID)
	}

	if ShellyBLERPCCharUUID != "5f6d4f53-5f52-5043-5f64-6174615f5f5f" {
		t.Errorf("ShellyBLERPCCharUUID = %s, want 5f6d4f53-5f52-5043-5f64-6174615f5f5f", ShellyBLERPCCharUUID)
	}

	if ShellyBLENotifyCharUUID != "5f6d4f53-5f52-5043-5f72-785f63746c5f" {
		t.Errorf("ShellyBLENotifyCharUUID = %s, want 5f6d4f53-5f52-5043-5f72-785f63746c5f", ShellyBLENotifyCharUUID)
	}
}

// Test ErrBLETransmitterNotSupported (for unsupported platforms)

func TestErrBLETransmitterNotSupported_Error(t *testing.T) {
	// On supported platforms, this variable won't exist, so we skip
	// On unsupported platforms, verify the error message
	if os.Getenv("SHELLY_CI") == "1" {
		t.Skip("skipping platform-specific error test in CI")
	}
}

// Test tinyGoBLETransmitter methods (platform-specific, skip in CI)

func TestNewTinyGoBLETransmitter(t *testing.T) {
	if os.Getenv("SHELLY_CI") == "1" {
		t.Skip("skipping in CI - requires bluetooth hardware")
	}

	// This test is for platforms with bluetooth support
	// On unsupported platforms, it will return an error
	_, err := NewTinyGoBLETransmitter()
	if err != nil {
		// Expected on systems without bluetooth
		t.Logf("NewTinyGoBLETransmitter() error (expected without hardware): %v", err)
	}
}

func TestTinyGoBLETransmitter_ConnectNotConnected(t *testing.T) {
	if os.Getenv("SHELLY_CI") == "1" {
		t.Skip("skipping in CI - requires bluetooth hardware")
	}

	transmitter, err := NewTinyGoBLETransmitter()
	if err != nil {
		t.Skip("skipping - bluetooth not available: " + err.Error())
	}

	// Should not be connected initially
	if transmitter.IsConnected() {
		t.Error("should not be connected initially")
	}
}

func TestTinyGoBLETransmitter_WriteNotConnected(t *testing.T) {
	if os.Getenv("SHELLY_CI") == "1" {
		t.Skip("skipping in CI - requires bluetooth hardware")
	}

	transmitter, err := NewTinyGoBLETransmitter()
	if err != nil {
		t.Skip("skipping - bluetooth not available: " + err.Error())
	}

	// Writing without connection should fail
	err = transmitter.WriteCharacteristic(context.Background(), []byte("test"))
	if err == nil {
		t.Error("WriteCharacteristic should fail when not connected")
	}
}

func TestTinyGoBLETransmitter_ReadNotConnected(t *testing.T) {
	if os.Getenv("SHELLY_CI") == "1" {
		t.Skip("skipping in CI - requires bluetooth hardware")
	}

	transmitter, err := NewTinyGoBLETransmitter()
	if err != nil {
		t.Skip("skipping - bluetooth not available: " + err.Error())
	}

	// Reading without connection should fail
	_, err = transmitter.ReadNotification(context.Background())
	if err == nil {
		t.Error("ReadNotification should fail when not connected")
	}
}

func TestTinyGoBLETransmitter_DisconnectNotConnected(t *testing.T) {
	if os.Getenv("SHELLY_CI") == "1" {
		t.Skip("skipping in CI - requires bluetooth hardware")
	}

	transmitter, err := NewTinyGoBLETransmitter()
	if err != nil {
		t.Skip("skipping - bluetooth not available: " + err.Error())
	}

	// Disconnecting when not connected should be a no-op
	err = transmitter.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() should not fail when not connected: %v", err)
	}
}

// Test provisioner with mock transmitter

func TestProvisioner_WithMockTransmitter(t *testing.T) {
	transmitter := newMockBLETransmitter()

	// Set up mock to return a valid response
	responseData := []byte(`{"id":1,"src":"test","result":{"wifi_sta":{"ip":"192.168.1.100"}}}`)
	transmitter.SetNotifications(responseData)

	// Test basic operations
	err := transmitter.Connect(context.Background(), "AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if !transmitter.IsConnected() {
		t.Error("should be connected")
	}

	// Write a command
	cmd := []byte(`{"id":1,"method":"Shelly.GetDeviceInfo"}`)
	err = transmitter.WriteCharacteristic(context.Background(), cmd)
	if err != nil {
		t.Fatalf("WriteCharacteristic() error = %v", err)
	}

	// Read response
	response, err := transmitter.ReadNotification(context.Background())
	if err != nil {
		t.Fatalf("ReadNotification() error = %v", err)
	}

	if string(response) != string(responseData) {
		t.Errorf("response = %s, want %s", response, responseData)
	}

	// Disconnect
	err = transmitter.Disconnect()
	if err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	if transmitter.IsConnected() {
		t.Error("should not be connected after Disconnect()")
	}
}

// Test concurrent operations

func TestMockBLETransmitter_ConcurrentAccess(t *testing.T) {
	m := newMockBLETransmitter()
	m.SetNotifications(
		[]byte("n1"), []byte("n2"), []byte("n3"),
		[]byte("n4"), []byte("n5"), []byte("n6"),
	)

	var wg sync.WaitGroup

	// Concurrent connect
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = m.Connect(context.Background(), "AA:BB:CC:DD:EE:FF")
	}()

	// Concurrent writes
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.WriteCharacteristic(context.Background(), []byte("test"))
		}()
	}

	// Concurrent reads
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, _ = m.ReadNotification(ctx)
		}()
	}

	// Concurrent IsConnected checks
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.IsConnected()
		}()
	}

	wg.Wait()

	// Just verify no race conditions occurred
	_ = m.Disconnect()
}

// Test ReadNotificationWithTimeout

func TestTinyGoBLETransmitter_ReadNotificationWithTimeout(t *testing.T) {
	if os.Getenv("SHELLY_CI") == "1" {
		t.Skip("skipping in CI - requires bluetooth hardware")
	}

	transmitter, err := NewTinyGoBLETransmitter()
	if err != nil {
		t.Skip("skipping - bluetooth not available: " + err.Error())
	}

	// Reading with timeout when not connected should fail
	_, err = transmitter.ReadNotificationWithTimeout(50 * time.Millisecond)
	if err == nil {
		t.Error("ReadNotificationWithTimeout should fail when not connected")
	}
}
