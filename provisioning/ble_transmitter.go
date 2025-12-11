//go:build linux || darwin

package provisioning

import (
	"context"
	"errors"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

// tinyGoBLETransmitter implements BLETransmitter using tinygo.org/x/bluetooth.
// This implementation works on Linux (with BlueZ) and macOS (with CoreBluetooth).
//
//nolint:govet // Field order optimized for readability over alignment
type tinyGoBLETransmitter struct {
	adapter    *bluetooth.Adapter
	device     bluetooth.Device
	rpcChar    bluetooth.DeviceCharacteristic
	notifyChar bluetooth.DeviceCharacteristic
	mu         sync.Mutex
	notifyCh   chan []byte
	connected  bool
}

// NewTinyGoBLETransmitter creates a new BLE transmitter using TinyGo bluetooth.
// Returns an error if the bluetooth adapter cannot be enabled.
func NewTinyGoBLETransmitter() (*tinyGoBLETransmitter, error) {
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		return nil, errors.New("failed to enable bluetooth adapter: " + err.Error())
	}

	return &tinyGoBLETransmitter{
		adapter:  adapter,
		notifyCh: make(chan []byte, 10),
	}, nil
}

// Connect connects to a BLE device by address and discovers the Shelly service
// and RPC/notify characteristics.
//
//nolint:gocyclo,cyclop,funlen // BLE connection requires many sequential steps
func (t *tinyGoBLETransmitter) Connect(ctx context.Context, address string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return errors.New("already connected to a device")
	}

	// Parse address using platform-agnostic Set method
	var addr bluetooth.Address
	addr.Set(address)

	// Connect to device
	params := bluetooth.ConnectionParams{}

	// Connection with timeout
	done := make(chan error, 1)
	go func() {
		device, err := t.adapter.Connect(addr, params)
		if err != nil {
			done <- errors.New("failed to connect: " + err.Error())
			return
		}
		t.device = device
		done <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return err
		}
	}

	// Discover services
	services, err := t.device.DiscoverServices(nil)
	if err != nil {
		t.device.Disconnect() //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("failed to discover services: " + err.Error())
	}

	// Parse the Shelly service UUID
	shellyServiceUUID, err := bluetooth.ParseUUID(ShellyBLEServiceUUID)
	if err != nil {
		t.device.Disconnect() //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("invalid service UUID: " + err.Error())
	}

	// Find Shelly service
	var shellyService bluetooth.DeviceService
	found := false
	for _, svc := range services {
		if svc.UUID() == shellyServiceUUID {
			shellyService = svc
			found = true
			break
		}
	}

	if !found {
		t.device.Disconnect() //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("shelly BLE service not found")
	}

	// Discover characteristics
	chars, err := shellyService.DiscoverCharacteristics(nil)
	if err != nil {
		t.device.Disconnect() //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("failed to discover characteristics: " + err.Error())
	}

	// Parse characteristic UUIDs
	rpcUUID, err := bluetooth.ParseUUID(ShellyBLERPCCharUUID)
	if err != nil {
		t.device.Disconnect() //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("invalid RPC characteristic UUID: " + err.Error())
	}

	notifyUUID, err := bluetooth.ParseUUID(ShellyBLENotifyCharUUID)
	if err != nil {
		t.device.Disconnect() //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("invalid notify characteristic UUID: " + err.Error())
	}

	// Find RPC and notify characteristics
	foundRPC := false
	foundNotify := false
	for _, char := range chars {
		if char.UUID() == rpcUUID {
			t.rpcChar = char
			foundRPC = true
		}
		if char.UUID() == notifyUUID {
			t.notifyChar = char
			foundNotify = true
		}
	}

	if !foundRPC {
		t.device.Disconnect() //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("RPC characteristic not found")
	}

	if !foundNotify {
		t.device.Disconnect() //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("notify characteristic not found")
	}

	// Enable notifications on the notify characteristic
	err = t.notifyChar.EnableNotifications(func(data []byte) {
		// Copy data to prevent race conditions
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		// Non-blocking send to channel
		select {
		case t.notifyCh <- dataCopy:
		default:
			// Channel full, drop oldest
			select {
			case <-t.notifyCh:
			default:
			}
			t.notifyCh <- dataCopy
		}
	})
	if err != nil {
		t.device.Disconnect() //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("failed to enable notifications: " + err.Error())
	}

	t.connected = true
	return nil
}

// Disconnect disconnects from the currently connected device.
func (t *tinyGoBLETransmitter) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	err := t.device.Disconnect()
	t.connected = false

	// Drain notification channel
	for len(t.notifyCh) > 0 {
		<-t.notifyCh
	}

	return err
}

// WriteCharacteristic writes data to the RPC characteristic.
func (t *tinyGoBLETransmitter) WriteCharacteristic(ctx context.Context, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return errors.New("not connected")
	}

	// Write with timeout
	done := make(chan error, 1)
	go func() {
		_, err := t.rpcChar.WriteWithoutResponse(data)
		done <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return errors.New("failed to write characteristic: " + err.Error())
		}
		return nil
	}
}

// ReadNotification reads a notification from the device.
// This blocks until a notification is received or the context is canceled.
func (t *tinyGoBLETransmitter) ReadNotification(ctx context.Context) ([]byte, error) {
	if !t.IsConnected() {
		return nil, errors.New("not connected")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data := <-t.notifyCh:
		return data, nil
	}
}

// ReadNotificationWithTimeout reads a notification with a specific timeout.
func (t *tinyGoBLETransmitter) ReadNotificationWithTimeout(timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return t.ReadNotification(ctx)
}

// IsConnected returns true if currently connected to a device.
func (t *tinyGoBLETransmitter) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connected
}

// Ensure tinyGoBLETransmitter implements BLETransmitter.
var _ BLETransmitter = (*tinyGoBLETransmitter)(nil)
