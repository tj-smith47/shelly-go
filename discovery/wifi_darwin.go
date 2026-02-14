//go:build darwin

package discovery

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// platformWiFiScanner implements WiFiScanner for macOS using networksetup and airport.
type platformWiFiScanner struct {
	iface string
}

// newPlatformWiFiScanner creates a platform-specific WiFi scanner for macOS.
func newPlatformWiFiScanner() WiFiScanner {
	return &platformWiFiScanner{
		iface: detectWiFiInterface(),
	}
}

// detectWiFiInterface finds the primary WiFi interface on macOS.
func detectWiFiInterface() string {
	cmd := exec.Command("networksetup", "-listallhardwareports")
	output, err := cmd.Output()
	if err != nil {
		return "en0"
	}

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

	return "en0"
}

// airportPath returns the path to the airport command-line tool.
func airportPath() string {
	return "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"
}

// ─── Scan ────────────────────────────────────────────────────────────────────

// Scan scans for available WiFi networks using the macOS airport tool.
func (s *platformWiFiScanner) Scan(ctx context.Context) ([]WiFiNetwork, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	airport := airportPath()
	cmd := exec.CommandContext(ctx, airport, "-s")
	output, err := cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "airport scan failed", Err: err}
	}

	return parseAirportScanOutput(string(output)), nil
}

// parseAirportScanOutput parses the output of airport -s.
// Format (fixed-width columns):
//
//	SSID                 BSSID             RSSI CHANNEL HT CC SECURITY
//	HomeNetwork          aa:bb:cc:dd:ee:ff -50  6       Y  -- WPA2(PSK/AES/AES)
func parseAirportScanOutput(output string) []WiFiNetwork {
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return nil
	}

	// Find column positions from the header line.
	header := lines[0]
	bssidCol := strings.Index(header, "BSSID")
	rssiCol := strings.Index(header, "RSSI")
	chanCol := strings.Index(header, "CHANNEL")
	secCol := strings.Index(header, "SECURITY")

	if bssidCol < 0 || rssiCol < 0 {
		return nil
	}

	var networks []WiFiNetwork
	for _, line := range lines[1:] {
		if len(line) <= bssidCol {
			continue
		}

		ssid := strings.TrimSpace(line[:bssidCol])
		if ssid == "" {
			continue
		}

		network := WiFiNetwork{
			SSID:     ssid,
			LastSeen: time.Now(),
		}

		if len(line) > rssiCol {
			network.BSSID = strings.TrimSpace(line[bssidCol:rssiCol])
		}

		if rssiCol > 0 && len(line) > rssiCol {
			end := chanCol
			if end <= rssiCol || end > len(line) {
				end = len(line)
			}
			rssiStr := strings.TrimSpace(line[rssiCol:end])
			if rssi, err := strconv.Atoi(rssiStr); err == nil {
				network.Signal = rssi
			}
		}

		if chanCol > 0 && len(line) > chanCol {
			end := secCol
			if end <= chanCol || end > len(line) {
				end = len(line)
			}
			chanStr := strings.TrimSpace(line[chanCol:end])
			// Channel might be "6" or "6,+1" or "36,-1".
			if idx := strings.IndexAny(chanStr, ", "); idx > 0 {
				chanStr = chanStr[:idx]
			}
			if ch, err := strconv.Atoi(chanStr); err == nil {
				network.Channel = ch
			}
		}

		if secCol > 0 && len(line) > secCol {
			network.Security = strings.TrimSpace(line[secCol:])
		}

		networks = append(networks, network)
	}

	return networks
}

// ─── Connect ─────────────────────────────────────────────────────────────────

// Connect connects to a WiFi network on macOS.
func (s *platformWiFiScanner) Connect(ctx context.Context, ssid, password string) error {
	var cmd *exec.Cmd
	if password == "" {
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

	return s.waitForConnection(ctx, ssid, 15*time.Second)
}

// ─── Disconnect ──────────────────────────────────────────────────────────────

// Disconnect disconnects from the current WiFi network on macOS.
func (s *platformWiFiScanner) Disconnect(ctx context.Context) error {
	offCmd := exec.CommandContext(ctx, "networksetup", "-setairportpower", s.iface, "off")
	if err := offCmd.Run(); err != nil {
		return &WiFiError{Message: "networksetup power off failed", Err: err}
	}

	time.Sleep(500 * time.Millisecond)

	onCmd := exec.CommandContext(ctx, "networksetup", "-setairportpower", s.iface, "on")
	if err := onCmd.Run(); err != nil {
		return &WiFiError{Message: "networksetup power on failed", Err: err}
	}

	return nil
}

// ─── CurrentNetwork ──────────────────────────────────────────────────────────

// CurrentNetwork returns the currently connected WiFi network on macOS.
func (s *platformWiFiScanner) CurrentNetwork(ctx context.Context) (*WiFiNetwork, error) {
	airport := airportPath()
	cmd := exec.CommandContext(ctx, airport, "-I")
	output, err := cmd.Output()
	if err == nil {
		return s.parseAirportInfo(string(output))
	}

	// Fallback to networksetup.
	cmd = exec.CommandContext(ctx, "networksetup", "-getairportnetwork", s.iface)
	output, err = cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "networksetup getairportnetwork failed", Err: err}
	}

	outputStr := strings.TrimSpace(string(output))
	prefix := "Current Wi-Fi Network: "
	if strings.HasPrefix(outputStr, prefix) {
		ssid := strings.TrimPrefix(outputStr, prefix)
		if ssid != "" && ssid != "You are not associated with an AirPort network." {
			return &WiFiNetwork{SSID: ssid, LastSeen: time.Now()}, nil
		}
	}

	return nil, &WiFiError{Message: "not connected to any WiFi network"}
}

// parseAirportInfo parses the output of airport -I.
func (s *platformWiFiScanner) parseAirportInfo(output string) (*WiFiNetwork, error) {
	network := &WiFiNetwork{LastSeen: time.Now()}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "SSID:"):
			network.SSID = strings.TrimSpace(strings.TrimPrefix(line, "SSID:"))
		case strings.HasPrefix(line, "BSSID:"):
			network.BSSID = strings.TrimSpace(strings.TrimPrefix(line, "BSSID:"))
		case strings.HasPrefix(line, "channel:"):
			chanStr := strings.TrimSpace(strings.TrimPrefix(line, "channel:"))
			if idx := strings.Index(chanStr, ","); idx != -1 {
				chanStr = chanStr[:idx]
			}
			if ch, err := strconv.Atoi(chanStr); err == nil {
				network.Channel = ch
			}
		case strings.HasPrefix(line, "agrCtlRSSI:"):
			rssiStr := strings.TrimSpace(strings.TrimPrefix(line, "agrCtlRSSI:"))
			if rssi, err := strconv.Atoi(rssiStr); err == nil {
				network.Signal = rssi
			}
		case strings.HasPrefix(line, "link auth:"):
			network.Security = strings.TrimSpace(strings.TrimPrefix(line, "link auth:"))
		}
	}

	if network.SSID == "" {
		return nil, &WiFiError{Message: "not connected to any WiFi network"}
	}

	return network, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

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
