//go:build !linux && !darwin && !windows

package discovery

import (
	"context"
	"runtime"
)

// platformWiFiScanner implements WiFiScanner for unsupported platforms.
type platformWiFiScanner struct{}

// newPlatformWiFiScanner creates a platform-specific WiFi scanner.
// On unsupported platforms, all operations return ErrWiFiNotSupported.
func newPlatformWiFiScanner() WiFiScanner {
	return &platformWiFiScanner{}
}

// Scan returns an error on unsupported platforms.
func (s *platformWiFiScanner) Scan(ctx context.Context) ([]WiFiNetwork, error) {
	return nil, &WiFiError{
		Message: "WiFi scanning not supported on " + runtime.GOOS + "; " +
			"supported platforms: linux (NetworkManager/nmcli/iw), darwin (airport), windows (netsh)",
	}
}

// Connect returns an error on unsupported platforms.
func (s *platformWiFiScanner) Connect(ctx context.Context, ssid, password string) error {
	return &WiFiError{
		Message: "WiFi connection not supported on " + runtime.GOOS + "; " +
			"supported platforms: linux (nmcli/wpa_cli), darwin (networksetup), windows (netsh)",
	}
}

// Disconnect returns an error on unsupported platforms.
func (s *platformWiFiScanner) Disconnect(ctx context.Context) error {
	return &WiFiError{
		Message: "WiFi disconnection not supported on " + runtime.GOOS + "; " +
			"supported platforms: linux (nmcli/wpa_cli), darwin (networksetup), windows (netsh)",
	}
}

// CurrentNetwork returns an error on unsupported platforms.
func (s *platformWiFiScanner) CurrentNetwork(ctx context.Context) (*WiFiNetwork, error) {
	return nil, &WiFiError{
		Message: "current network detection not supported on " + runtime.GOOS + "; " +
			"supported platforms: linux (nmcli/wpa_cli), darwin (networksetup), windows (netsh)",
	}
}
