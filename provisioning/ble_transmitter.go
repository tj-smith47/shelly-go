//go:build linux || darwin || windows

package provisioning

import (
	"context"
	"errors"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

// bleConnector is the minimal interface for the BLE adapter operations used
// by tinyGoBLETransmitter.  The production implementation delegates to
// *bluetooth.Adapter; tests inject a fake.
type bleConnector interface {
	Connect(addr bluetooth.Address, params bluetooth.ConnectionParams) (bluetooth.Device, error)
}

// tinyGoBLETransmitter implements BLETransmitter using tinygo.org/x/bluetooth.
// This implementation works on Linux (with BlueZ) and macOS (with CoreBluetooth).
//
//nolint:govet // Field order optimized for readability over alignment
type tinyGoBLETransmitter struct {
	connector bleConnector
	// dropDevice replaces device.Disconnect in tests, where a bluetooth.Device
	// has no real connection behind it.
	dropDevice func(bluetooth.Device) error
	device     bluetooth.Device
	rpcChar    bluetooth.DeviceCharacteristic
	notifyChar bluetooth.DeviceCharacteristic
	mu         sync.Mutex
	notifyCh   chan []byte
	// gone is closed by Disconnect so a blocked ReadNotification returns.
	gone      chan struct{}
	connected bool
}

// bleDialResult is what a connection attempt hands back to Connect.
type bleDialResult struct {
	err    error
	device bluetooth.Device
}

// NewTinyGoBLETransmitter creates a new BLE transmitter using TinyGo bluetooth.
// Returns an error if the bluetooth adapter cannot be enabled.
func NewTinyGoBLETransmitter() (*tinyGoBLETransmitter, error) {
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		return nil, errors.New("failed to enable bluetooth adapter: " + err.Error())
	}

	return &tinyGoBLETransmitter{
		connector: adapter,
		notifyCh:  make(chan []byte, 10),
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

	// The goroutine only reports its result. It may outlive this call, so it
	// must not touch the transmitter's fields, which belong to whoever holds
	// t.mu.
	connector := t.connector
	done := make(chan bleDialResult, 1)
	go func() {
		device, err := connector.Connect(addr, params)
		done <- bleDialResult{device: device, err: err}
	}()

	select {
	case <-ctx.Done():
		go t.dropLate(done)
		return ctx.Err()
	case res := <-done:
		if res.err != nil {
			return errors.New("failed to connect: " + res.err.Error())
		}
		t.device = res.device
	}

	// Discover services
	services, err := t.device.DiscoverServices(nil)
	if err != nil {
		t.drop(t.device) //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("failed to discover services: " + err.Error())
	}

	// Parse the Shelly service UUID
	shellyServiceUUID, err := bluetooth.ParseUUID(ShellyBLEServiceUUID)
	if err != nil {
		t.drop(t.device) //nolint:errcheck // Best-effort cleanup on failure
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
		t.drop(t.device) //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("shelly BLE service not found")
	}

	// Discover characteristics
	chars, err := shellyService.DiscoverCharacteristics(nil)
	if err != nil {
		t.drop(t.device) //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("failed to discover characteristics: " + err.Error())
	}

	// Parse characteristic UUIDs
	rpcUUID, err := bluetooth.ParseUUID(ShellyBLERPCCharUUID)
	if err != nil {
		t.drop(t.device) //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("invalid RPC characteristic UUID: " + err.Error())
	}

	notifyUUID, err := bluetooth.ParseUUID(ShellyBLENotifyCharUUID)
	if err != nil {
		t.drop(t.device) //nolint:errcheck // Best-effort cleanup on failure
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
		t.drop(t.device) //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("RPC characteristic not found")
	}

	if !foundNotify {
		t.drop(t.device) //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("notify characteristic not found")
	}

	// Enable notifications on the notify characteristic
	err = t.notifyChar.EnableNotifications(t.queueNotification)
	if err != nil {
		t.drop(t.device) //nolint:errcheck // Best-effort cleanup on failure
		return errors.New("failed to enable notifications: " + err.Error())
	}

	t.gone = make(chan struct{})
	t.connected = true
	return nil
}

// queueNotification stores a copy of data for ReadNotification, discarding the
// oldest stored notification when the buffer is full. It never blocks: it runs
// on the Bluetooth stack's callback goroutine.
func (t *tinyGoBLETransmitter) queueNotification(data []byte) {
	// The stack may reuse data once this callback returns.
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	for {
		select {
		case t.notifyCh <- dataCopy:
			return
		default:
		}
		select {
		case <-t.notifyCh:
		default:
		}
	}
}

// dropLate closes a connection that is made after Connect stopped waiting for
// it. The adapter call cannot be interrupted, and nobody owns its result.
func (t *tinyGoBLETransmitter) dropLate(done <-chan bleDialResult) {
	res := <-done
	if res.err != nil {
		return
	}
	// Connect has already returned, so a failed disconnect has no one to go to.
	if err := t.drop(res.device); err != nil {
		return
	}
}

// drop closes the connection to device.
func (t *tinyGoBLETransmitter) drop(device bluetooth.Device) error {
	if t.dropDevice != nil {
		return t.dropDevice(device)
	}
	return device.Disconnect()
}

// Disconnect disconnects from the currently connected device.
func (t *tinyGoBLETransmitter) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	err := t.drop(t.device)
	t.connected = false
	if t.gone != nil {
		close(t.gone)
		t.gone = nil
	}

	// Discard notifications left over from this connection. A reader may be
	// taking from the channel at the same time, so never block on it.
	for {
		select {
		case <-t.notifyCh:
			continue
		default:
		}
		break
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

	// The goroutine may outlive this call, and a later Connect replaces
	// t.rpcChar, so it works on a copy.
	rpcChar := t.rpcChar
	done := make(chan error, 1)
	go func() {
		_, err := rpcChar.WriteWithoutResponse(data)
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
// This blocks until a notification is received, the context is canceled, or
// Disconnect is called, in which case it returns a "not connected" error.
func (t *tinyGoBLETransmitter) ReadNotification(ctx context.Context) ([]byte, error) {
	t.mu.Lock()
	connected, gone := t.connected, t.gone
	t.mu.Unlock()

	if !connected {
		return nil, errors.New("not connected")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-gone:
		return nil, errors.New("not connected")
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
