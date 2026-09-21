//go:build linux || darwin || windows

package provisioning

import (
	"context"
	"errors"
	"testing"
	"time"

	"tinygo.org/x/bluetooth"
)

const bleTxTestTimeout = 5 * time.Second

// blockingBLEConnector blocks in Connect until the test releases it.
type blockingBLEConnector struct {
	entered chan struct{}
	release chan error
}

func (f *blockingBLEConnector) Connect(bluetooth.Address, bluetooth.ConnectionParams) (bluetooth.Device, error) {
	f.entered <- struct{}{}
	return bluetooth.Device{}, <-f.release
}

func TestTinyGoBLETransmitter_LateConnectionIsDropped(t *testing.T) {
	connector := &blockingBLEConnector{entered: make(chan struct{}, 1), release: make(chan error, 1)}
	dropped := make(chan struct{}, 1)
	tx := &tinyGoBLETransmitter{
		connector: connector,
		notifyCh:  make(chan []byte, 10),
		dropDevice: func(bluetooth.Device) error {
			dropped <- struct{}{}
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- tx.Connect(ctx, "AA:BB:CC:DD:EE:FF") }()

	select {
	case <-connector.entered:
	case <-time.After(bleTxTestTimeout):
		t.Fatal("timed out waiting for the dial to begin")
	}

	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Connect() error = %v, want context.Canceled", err)
		}
	case <-time.After(bleTxTestTimeout):
		t.Fatal("timed out waiting for Connect to return")
	}

	// The adapter connects after Connect gave up.
	connector.release <- nil
	select {
	case <-dropped:
	case <-time.After(bleTxTestTimeout):
		t.Fatal("a connection that arrived after Connect returned was never closed")
	}

	// Read without the lock on purpose: nothing may write the transmitter's
	// fields once Connect has returned, and the race detector checks that.
	if tx.connected {
		t.Error("connected = true after a canceled Connect")
	}
	_ = tx.device
}

func TestTinyGoBLETransmitter_DisconnectEndsReadNotification(t *testing.T) {
	tx := buildConnectedTransmitter()
	tx.gone = make(chan struct{})
	tx.dropDevice = func(bluetooth.Device) error { return nil }

	reading := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		close(reading)
		_, err := tx.ReadNotification(context.Background())
		result <- err
	}()
	<-reading

	if err := tx.Disconnect(); err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	select {
	case err := <-result:
		// The reader either saw the disconnect while waiting or arrived
		// after it; both report the same error.
		if err == nil || err.Error() != "not connected" {
			t.Errorf("ReadNotification() error = %v, want %q", err, "not connected")
		}
	case <-time.After(bleTxTestTimeout):
		t.Fatal("ReadNotification still blocked after Disconnect")
	}
}

func TestTinyGoBLETransmitter_QueueNotificationKeepsNewest(t *testing.T) {
	tx := buildDisconnectedTransmitter()
	total := cap(tx.notifyCh) + 3

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range total {
			tx.queueNotification([]byte{byte(i)})
		}
	}()
	select {
	case <-done:
	case <-time.After(bleTxTestTimeout):
		t.Fatal("queueNotification blocked on a full buffer")
	}

	if len(tx.notifyCh) != cap(tx.notifyCh) {
		t.Fatalf("buffer holds %d notifications, want %d", len(tx.notifyCh), cap(tx.notifyCh))
	}
	if first := <-tx.notifyCh; first[0] != 3 {
		t.Errorf("oldest kept notification = %d, want 3", first[0])
	}
}

func TestTinyGoBLETransmitter_QueueNotificationCopiesData(t *testing.T) {
	tx := buildDisconnectedTransmitter()
	data := []byte("abc")
	tx.queueNotification(data)
	data[0] = 'X'

	if got := <-tx.notifyCh; string(got) != "abc" {
		t.Errorf("queued notification = %q, want %q", got, "abc")
	}
}
