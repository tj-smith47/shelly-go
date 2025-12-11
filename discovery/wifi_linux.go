//go:build linux

package discovery

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// platformWiFiScanner implements WiFiScanner for Linux using nmcli, wpa_cli, or iwconfig.
type platformWiFiScanner struct {
	*wifiscanScanner
	iface string
}

// newPlatformWiFiScanner creates a platform-specific WiFi scanner for Linux.
func newPlatformWiFiScanner() WiFiScanner {
	return &platformWiFiScanner{
		wifiscanScanner: &wifiscanScanner{},
		iface:           detectWiFiInterface(),
	}
}

// detectWiFiInterface finds the primary WiFi interface on Linux.
func detectWiFiInterface() string {
	// Check /sys/class/net/*/wireless for WiFi interfaces
	matches, err := filepath.Glob("/sys/class/net/*/wireless")
	if err == nil && len(matches) > 0 {
		// Extract interface name from path like /sys/class/net/wlan0/wireless
		parts := strings.Split(matches[0], "/")
		if len(parts) >= 5 {
			return parts[4]
		}
	}

	// Fallback to common interface names
	commonNames := []string{"wlan0", "wlan1", "wlp2s0", "wlp3s0", "wlo1"}
	for _, name := range commonNames {
		if _, err := os.Stat("/sys/class/net/" + name); err == nil {
			return name
		}
	}

	return "wlan0" // Default fallback
}

// Connect connects to a WiFi network on Linux.
// It tries nmcli first, then wpa_cli, then iwconfig as fallbacks.
// Note: nmcli works without sudo when NetworkManager is running.
// wpa_cli and iwconfig may require elevated privileges.
func (s *platformWiFiScanner) Connect(ctx context.Context, ssid, password string) error {
	// Try nmcli first (NetworkManager - most common on modern Linux)
	if hasCommand("nmcli") {
		err := s.connectNmcli(ctx, ssid, password)
		if err == nil {
			return nil
		}
		// If nmcli failed but we have other options, try them
		if !hasCommand("wpa_cli") && !hasCommand("iwconfig") {
			return err
		}
	}

	// Try wpa_cli (wpa_supplicant - common on minimal systems)
	if hasCommand("wpa_cli") {
		err := s.connectWpaCli(ctx, ssid, password)
		if err == nil {
			return nil
		}
		if !hasCommand("iwconfig") {
			return err
		}
	}

	// Try iwconfig (deprecated but still available on some systems)
	if hasCommand("iwconfig") {
		return s.connectIwconfig(ctx, ssid, password)
	}

	return ErrToolNotFound
}

// connectNmcli connects using NetworkManager's nmcli.
func (s *platformWiFiScanner) connectNmcli(ctx context.Context, ssid, password string) error {
	var cmd *exec.Cmd
	if password == "" {
		// Open network (like Shelly AP)
		cmd = exec.CommandContext(ctx, "nmcli", "device", "wifi", "connect", ssid)
	} else {
		cmd = exec.CommandContext(ctx, "nmcli", "device", "wifi", "connect", ssid, "password", password)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if strings.Contains(errMsg, "No network with SSID") {
			return ErrSSIDNotFound
		}
		if strings.Contains(errMsg, "Secrets were required") ||
			strings.Contains(errMsg, "password") {
			return ErrAuthFailed
		}
		return &WiFiError{Message: "nmcli connect failed", Err: err}
	}

	// Wait for connection to establish
	return s.waitForConnection(ctx, ssid, 15*time.Second)
}

// connectWpaCli connects using wpa_supplicant's wpa_cli.
// The interface name and network ID are validated/generated internally.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input; network ID from wpa_cli output
func (s *platformWiFiScanner) connectWpaCli(ctx context.Context, ssid, password string) error {
	// Add network
	addCmd := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "add_network")
	output, err := addCmd.Output()
	if err != nil {
		return &WiFiError{Message: "wpa_cli add_network failed", Err: err}
	}
	networkID := strings.TrimSpace(string(output))

	// Set SSID
	setSSID := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "set_network", networkID, "ssid", `"`+ssid+`"`)
	if err := setSSID.Run(); err != nil {
		return &WiFiError{Message: "wpa_cli set ssid failed", Err: err}
	}

	// Set password or key_mgmt for open networks
	if password == "" {
		setKeyMgmt := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "set_network", networkID, "key_mgmt", "NONE")
		if err := setKeyMgmt.Run(); err != nil {
			return &WiFiError{Message: "wpa_cli set key_mgmt failed", Err: err}
		}
	} else {
		setPSK := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "set_network", networkID, "psk", `"`+password+`"`)
		if err := setPSK.Run(); err != nil {
			return &WiFiError{Message: "wpa_cli set psk failed", Err: err}
		}
	}

	// Enable network
	enableCmd := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "enable_network", networkID)
	if err := enableCmd.Run(); err != nil {
		return &WiFiError{Message: "wpa_cli enable_network failed", Err: err}
	}

	// Select network
	selectCmd := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "select_network", networkID)
	if err := selectCmd.Run(); err != nil {
		return &WiFiError{Message: "wpa_cli select_network failed", Err: err}
	}

	return s.waitForConnection(ctx, ssid, 15*time.Second)
}

// connectIwconfig connects using iwconfig (deprecated, for legacy systems).
// The interface name is auto-detected from system, not user input.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input
func (s *platformWiFiScanner) connectIwconfig(ctx context.Context, ssid, password string) error {
	// Set ESSID
	essidCmd := exec.CommandContext(ctx, "iwconfig", s.iface, "essid", ssid)
	if err := essidCmd.Run(); err != nil {
		return &WiFiError{Message: "iwconfig essid failed", Err: err}
	}

	// Set key if password provided
	if password != "" {
		keyCmd := exec.CommandContext(ctx, "iwconfig", s.iface, "key", "s:"+password)
		if err := keyCmd.Run(); err != nil {
			return &WiFiError{Message: "iwconfig key failed", Err: err}
		}
	}

	// Bring interface up
	upCmd := exec.CommandContext(ctx, "ip", "link", "set", s.iface, "up")
	if err := upCmd.Run(); err != nil {
		// Try ifconfig as fallback
		upCmd = exec.CommandContext(ctx, "ifconfig", s.iface, "up")
		if err := upCmd.Run(); err != nil {
			return &WiFiError{Message: "failed to bring interface up", Err: err}
		}
	}

	return s.waitForConnection(ctx, ssid, 15*time.Second)
}

// Disconnect disconnects from the current WiFi network.
// The interface name is auto-detected from system, not user input.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input
func (s *platformWiFiScanner) Disconnect(ctx context.Context) error {
	if hasCommand("nmcli") {
		cmd := exec.CommandContext(ctx, "nmcli", "device", "disconnect", s.iface)
		if err := cmd.Run(); err != nil {
			return &WiFiError{Message: "nmcli disconnect failed", Err: err}
		}
		return nil
	}

	if hasCommand("wpa_cli") {
		cmd := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "disconnect")
		if err := cmd.Run(); err != nil {
			return &WiFiError{Message: "wpa_cli disconnect failed", Err: err}
		}
		return nil
	}

	if hasCommand("iwconfig") {
		cmd := exec.CommandContext(ctx, "iwconfig", s.iface, "essid", "off")
		if err := cmd.Run(); err != nil {
			return &WiFiError{Message: "iwconfig disconnect failed", Err: err}
		}
		return nil
	}

	return ErrToolNotFound
}

// CurrentNetwork returns the currently connected WiFi network.
func (s *platformWiFiScanner) CurrentNetwork(ctx context.Context) (*WiFiNetwork, error) {
	if hasCommand("nmcli") {
		return s.currentNetworkNmcli(ctx)
	}

	if hasCommand("wpa_cli") {
		return s.currentNetworkWpaCli(ctx)
	}

	if hasCommand("iwconfig") {
		return s.currentNetworkIwconfig(ctx)
	}

	return nil, ErrToolNotFound
}

// currentNetworkNmcli gets current network using nmcli.
func (s *platformWiFiScanner) currentNetworkNmcli(ctx context.Context) (*WiFiNetwork, error) {
	// nmcli -t -f IN-USE,SSID,SIGNAL,SECURITY device wifi list
	cmd := exec.CommandContext(ctx, "nmcli", "-t", "-f", "IN-USE,SSID,SIGNAL,SECURITY", "device", "wifi", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "nmcli wifi list failed", Err: err}
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		network := s.parseNmcliLine(line)
		if network != nil {
			return network, nil
		}
	}

	return nil, &WiFiError{Message: "not connected to any WiFi network"}
}

// parseNmcliLine parses a single line from nmcli wifi list output.
// Returns nil if line is not the current network.
func (s *platformWiFiScanner) parseNmcliLine(line string) *WiFiNetwork {
	// Format: *:SSID:SIGNAL:SECURITY (asterisk indicates current network)
	if !strings.HasPrefix(line, "*:") {
		return nil
	}

	parts := strings.SplitN(line[2:], ":", 3)
	if len(parts) < 1 || parts[0] == "" {
		return nil
	}

	network := &WiFiNetwork{
		SSID:     parts[0],
		LastSeen: time.Now(),
	}

	if len(parts) >= 2 {
		var signal int
		if _, err := parseIntFromString(parts[1], &signal); err == nil {
			network.Signal = signal
		}
	}

	if len(parts) >= 3 {
		network.Security = parts[2]
	}

	return network
}

// currentNetworkWpaCli gets current network using wpa_cli.
// The interface name is auto-detected from system, not user input.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input
func (s *platformWiFiScanner) currentNetworkWpaCli(ctx context.Context) (*WiFiNetwork, error) {
	cmd := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "status")
	output, err := cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "wpa_cli status failed", Err: err}
	}

	var ssid string
	var state string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ssid=") {
			ssid = strings.TrimPrefix(line, "ssid=")
		}
		if strings.HasPrefix(line, "wpa_state=") {
			state = strings.TrimPrefix(line, "wpa_state=")
		}
	}

	if state != "COMPLETED" || ssid == "" {
		return nil, &WiFiError{Message: "not connected to any WiFi network"}
	}

	return &WiFiNetwork{
		SSID:     ssid,
		LastSeen: time.Now(),
	}, nil
}

// currentNetworkIwconfig gets current network using iwconfig.
// The interface name is auto-detected from system, not user input.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input
func (s *platformWiFiScanner) currentNetworkIwconfig(ctx context.Context) (*WiFiNetwork, error) {
	cmd := exec.CommandContext(ctx, "iwconfig", s.iface)
	output, err := cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "iwconfig failed", Err: err}
	}

	// Parse ESSID from output like: wlan0  IEEE 802.11  ESSID:"NetworkName"
	outputStr := string(output)
	essidIdx := strings.Index(outputStr, `ESSID:"`)
	if essidIdx == -1 {
		return nil, &WiFiError{Message: "not connected to any WiFi network"}
	}

	// Find the closing quote
	start := essidIdx + 7
	end := strings.Index(outputStr[start:], `"`)
	if end == -1 {
		return nil, &WiFiError{Message: "failed to parse ESSID"}
	}

	ssid := outputStr[start : start+end]
	if ssid == "" || ssid == "off/any" {
		return nil, &WiFiError{Message: "not connected to any WiFi network"}
	}

	return &WiFiNetwork{
		SSID:     ssid,
		LastSeen: time.Now(),
	}, nil
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

// hasCommand checks if a command is available in PATH.
func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// parseIntFromString is a helper to parse int from string.
func parseIntFromString(s string, result *int) (bool, error) {
	var n int
	_, err := strings.NewReader(s).Read([]byte{})
	if err != nil {
		return false, err
	}
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	*result = n
	return true, nil
}
