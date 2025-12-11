//go:build darwin

package discovery

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// platformWiFiScanner implements WiFiScanner for macOS using networksetup and airport.
type platformWiFiScanner struct {
	*wifiscanScanner
	iface string
}

// newPlatformWiFiScanner creates a platform-specific WiFi scanner for macOS.
func newPlatformWiFiScanner() WiFiScanner {
	return &platformWiFiScanner{
		wifiscanScanner: &wifiscanScanner{},
		iface:           detectWiFiInterface(),
	}
}

// detectWiFiInterface finds the primary WiFi interface on macOS.
func detectWiFiInterface() string {
	// Use networksetup to find WiFi interface
	cmd := exec.Command("networksetup", "-listallhardwareports")
	output, err := cmd.Output()
	if err != nil {
		return "en0" // Default fallback
	}

	// Parse output looking for Wi-Fi device
	lines := strings.Split(string(output), "\n")
	foundWiFi := false
	for _, line := range lines {
		if strings.Contains(line, "Wi-Fi") || strings.Contains(line, "AirPort") {
			foundWiFi = true
			continue
		}
		if foundWiFi && strings.HasPrefix(line, "Device:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Device:"))
		}
	}

	return "en0" // Default for most Macs
}

// airportPath returns the path to the airport command-line tool.
func airportPath() string {
	return "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"
}

// Connect connects to a WiFi network on macOS.
// Note: This may require administrator privileges.
func (s *platformWiFiScanner) Connect(ctx context.Context, ssid, password string) error {
	// Use networksetup to connect
	var cmd *exec.Cmd
	if password == "" {
		// Open network (like Shelly AP)
		cmd = exec.CommandContext(ctx, "networksetup", "-setairportnetwork", s.iface, ssid)
	} else {
		cmd = exec.CommandContext(ctx, "networksetup", "-setairportnetwork", s.iface, ssid, password)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if strings.Contains(errMsg, "not found") ||
			strings.Contains(errMsg, "Could not find network") {
			return ErrSSIDNotFound
		}
		if strings.Contains(errMsg, "password") ||
			strings.Contains(errMsg, "authentication") {
			return ErrAuthFailed
		}
		return &WiFiError{Message: "networksetup connect failed: " + errMsg, Err: err}
	}

	// Wait for connection to establish
	return s.waitForConnection(ctx, ssid, 15*time.Second)
}

// Disconnect disconnects from the current WiFi network on macOS.
// This toggles the WiFi off and on to disconnect.
func (s *platformWiFiScanner) Disconnect(ctx context.Context) error {
	// Turn WiFi off
	offCmd := exec.CommandContext(ctx, "networksetup", "-setairportpower", s.iface, "off")
	if err := offCmd.Run(); err != nil {
		return &WiFiError{Message: "networksetup power off failed", Err: err}
	}

	// Brief pause
	time.Sleep(500 * time.Millisecond)

	// Turn WiFi back on
	onCmd := exec.CommandContext(ctx, "networksetup", "-setairportpower", s.iface, "on")
	if err := onCmd.Run(); err != nil {
		return &WiFiError{Message: "networksetup power on failed", Err: err}
	}

	return nil
}

// CurrentNetwork returns the currently connected WiFi network on macOS.
func (s *platformWiFiScanner) CurrentNetwork(ctx context.Context) (*WiFiNetwork, error) {
	// Try airport -I first for detailed info
	airport := airportPath()
	cmd := exec.CommandContext(ctx, airport, "-I")
	output, err := cmd.Output()
	if err == nil {
		return s.parseAirportInfo(string(output))
	}

	// Fallback to networksetup
	cmd = exec.CommandContext(ctx, "networksetup", "-getairportnetwork", s.iface)
	output, err = cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "networksetup getairportnetwork failed", Err: err}
	}

	// Parse output like: "Current Wi-Fi Network: NetworkName"
	outputStr := strings.TrimSpace(string(output))
	prefix := "Current Wi-Fi Network: "
	if strings.HasPrefix(outputStr, prefix) {
		ssid := strings.TrimPrefix(outputStr, prefix)
		if ssid != "" && ssid != "You are not associated with an AirPort network." {
			return &WiFiNetwork{
				SSID:     ssid,
				LastSeen: time.Now(),
			}, nil
		}
	}

	return nil, &WiFiError{Message: "not connected to any WiFi network"}
}

// parseAirportInfo parses the output of airport -I.
func (s *platformWiFiScanner) parseAirportInfo(output string) (*WiFiNetwork, error) {
	network := &WiFiNetwork{
		LastSeen: time.Now(),
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SSID:") {
			network.SSID = strings.TrimSpace(strings.TrimPrefix(line, "SSID:"))
		} else if strings.HasPrefix(line, "BSSID:") {
			network.BSSID = strings.TrimSpace(strings.TrimPrefix(line, "BSSID:"))
		} else if strings.HasPrefix(line, "channel:") {
			channelStr := strings.TrimSpace(strings.TrimPrefix(line, "channel:"))
			// Channel might be like "6" or "6,+1"
			if idx := strings.Index(channelStr, ","); idx != -1 {
				channelStr = channelStr[:idx]
			}
			var channel int
			for _, c := range channelStr {
				if c >= '0' && c <= '9' {
					channel = channel*10 + int(c-'0')
				} else {
					break
				}
			}
			network.Channel = channel
		} else if strings.HasPrefix(line, "agrCtlRSSI:") {
			rssiStr := strings.TrimSpace(strings.TrimPrefix(line, "agrCtlRSSI:"))
			var rssi int
			negative := false
			for _, c := range rssiStr {
				if c == '-' {
					negative = true
				} else if c >= '0' && c <= '9' {
					rssi = rssi*10 + int(c-'0')
				}
			}
			if negative {
				rssi = -rssi
			}
			network.Signal = rssi
		} else if strings.HasPrefix(line, "link auth:") {
			network.Security = strings.TrimSpace(strings.TrimPrefix(line, "link auth:"))
		}
	}

	if network.SSID == "" {
		return nil, &WiFiError{Message: "not connected to any WiFi network"}
	}

	return network, nil
}

// waitForConnection waits for the WiFi connection to be established.
func (s *platformWiFiScanner) waitForConnection(ctx context.Context, ssid string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		network, err := s.CurrentNetwork(ctx)
		if err == nil && network != nil && network.SSID == ssid {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return ErrConnectionTimeout
}
