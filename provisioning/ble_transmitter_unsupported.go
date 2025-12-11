//go:build !linux && !darwin

package provisioning

import (
	"context"
	"errors"
	"time"
)

// ErrBLENotSupported is returned when BLE transmitter is not supported on the platform.
var ErrBLETransmitterNotSupported = errors.New("BLE transmitter not supported on this platform (requires Linux or macOS)")

// tinyGoBLETransmitter is a stub for unsupported platforms.
type tinyGoBLETransmitter struct{}

// NewTinyGoBLETransmitter returns an error on unsupported platforms.
func NewTinyGoBLETransmitter() (*tinyGoBLETransmitter, error) {
	return nil, ErrBLETransmitterNotSupported
}

// Connect is not supported on this platform.
func (t *tinyGoBLETransmitter) Connect(ctx context.Context, address string) error {
	return ErrBLETransmitterNotSupported
}

// Disconnect is not supported on this platform.
func (t *tinyGoBLETransmitter) Disconnect() error {
	return ErrBLETransmitterNotSupported
}

// WriteCharacteristic is not supported on this platform.
func (t *tinyGoBLETransmitter) WriteCharacteristic(ctx context.Context, data []byte) error {
	return ErrBLETransmitterNotSupported
}

// ReadNotification is not supported on this platform.
func (t *tinyGoBLETransmitter) ReadNotification(ctx context.Context) ([]byte, error) {
	return nil, ErrBLETransmitterNotSupported
}

// ReadNotificationWithTimeout is not supported on this platform.
func (t *tinyGoBLETransmitter) ReadNotificationWithTimeout(timeout time.Duration) ([]byte, error) {
	return nil, ErrBLETransmitterNotSupported
}

// IsConnected always returns false on unsupported platforms.
func (t *tinyGoBLETransmitter) IsConnected() bool {
	return false
}

// Ensure tinyGoBLETransmitter implements BLETransmitter.
var _ BLETransmitter = (*tinyGoBLETransmitter)(nil)
