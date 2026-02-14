//go:build linux

package discovery

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gnm "github.com/Wifx/gonetworkmanager/v2"
)

// platformWiFiScanner implements WiFiScanner for Linux.
// Primary: NetworkManager D-Bus API via gonetworkmanager (no binary dependencies).
// Fallback: nmcli, wpa_cli, iwconfig CLI tools.
type platformWiFiScanner struct {
	iface string
}

// newPlatformWiFiScanner creates a platform-specific WiFi scanner for Linux.
func newPlatformWiFiScanner() WiFiScanner {
	return &platformWiFiScanner{
		iface: detectWiFiInterface(),
	}
}

// detectWiFiInterface finds the primary WiFi interface on Linux.
func detectWiFiInterface() string {
	// Check /sys/class/net/*/wireless for WiFi interfaces.
	matches, err := filepath.Glob("/sys/class/net/*/wireless")
	if err == nil && len(matches) > 0 {
		parts := strings.Split(matches[0], "/")
		if len(parts) >= 5 {
			return parts[4]
		}
	}

	commonNames := []string{"wlan0", "wlan1", "wlp2s0", "wlp3s0", "wlp6s0", "wlo1"}
	for _, name := range commonNames {
		if _, err := os.Stat("/sys/class/net/" + name); err == nil {
			return name
		}
	}

	return "wlan0"
}

// ─── NetworkManager helpers ──────────────────────────────────────────────────

// nmWirelessDevice returns the first WiFi device from NetworkManager.
func nmWirelessDevice() (gnm.DeviceWireless, error) {
	nm, err := gnm.NewNetworkManager()
	if err != nil {
		return nil, err
	}

	devices, err := nm.GetDevices()
	if err != nil {
		return nil, err
	}

	for _, dev := range devices {
		devType, err := dev.GetPropertyDeviceType()
		if err != nil {
			continue
		}
		if devType == gnm.NmDeviceTypeWifi {
			wireless, err := gnm.NewDeviceWireless(dev.GetPath())
			if err != nil {
				continue
			}
			return wireless, nil
		}
	}

	return nil, &WiFiError{Message: "no WiFi device found via NetworkManager"}
}

// apToNetwork converts a NetworkManager AccessPoint to a WiFiNetwork.
func apToNetwork(ap gnm.AccessPoint) *WiFiNetwork {
	ssid, err := ap.GetPropertySSID()
	if err != nil || ssid == "" {
		return nil
	}

	network := &WiFiNetwork{
		SSID:     ssid,
		LastSeen: time.Now(),
	}

	if hw, err := ap.GetPropertyHWAddress(); err == nil {
		network.BSSID = hw
	}

	if strength, err := ap.GetPropertyStrength(); err == nil {
		// Convert percentage (0-100) to approximate dBm.
		network.Signal = int(strength) - 100
	}

	if freq, err := ap.GetPropertyFrequency(); err == nil {
		network.Channel = frequencyToChannel(int(freq))
	}

	var flags, wpaFlags, rsnFlags uint32
	if f, err := ap.GetPropertyFlags(); err == nil {
		flags = f
	}
	if f, err := ap.GetPropertyWPAFlags(); err == nil {
		wpaFlags = f
	}
	if f, err := ap.GetPropertyRSNFlags(); err == nil {
		rsnFlags = f
	}
	network.Security = nmFlagsToSecurity(flags, wpaFlags, rsnFlags)

	return network
}

// frequencyToChannel converts WiFi frequency (MHz) to channel number.
func frequencyToChannel(freq int) int {
	switch {
	case freq >= 2412 && freq <= 2484:
		if freq == 2484 {
			return 14
		}
		return (freq-2412)/5 + 1
	case freq >= 5170 && freq <= 5825:
		return (freq - 5000) / 5
	case freq >= 5955 && freq <= 7115:
		return (freq - 5950) / 5 // WiFi 6E
	}
	return 0
}

// nmFlagsToSecurity converts NM AP security flags to a human-readable string.
func nmFlagsToSecurity(flags, wpaFlags, rsnFlags uint32) string {
	const nmAPFlagPrivacy = 0x1

	if rsnFlags != 0 {
		if rsnFlags&0x200 != 0 { // NM_802_11_AP_SEC_KEY_MGMT_SAE (WPA3)
			return "WPA3"
		}
		return "WPA2"
	}
	if wpaFlags != 0 {
		return "WPA"
	}
	if flags&nmAPFlagPrivacy != 0 {
		return "WEP"
	}
	return ""
}

// ─── Scan ────────────────────────────────────────────────────────────────────

// Scan scans for available WiFi networks.
// Tries NetworkManager D-Bus first, then falls back to CLI tools.
func (s *platformWiFiScanner) Scan(ctx context.Context) ([]WiFiNetwork, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Primary: NetworkManager D-Bus (no binary dependency).
	if networks, err := s.scanViaNM(ctx); err == nil {
		return networks, nil
	}

	// Fallback: nmcli CLI.
	if hasCommand("nmcli") {
		if networks, err := s.scanViaNmcli(ctx); err == nil {
			return networks, nil
		}
	}

	// Fallback: iw CLI.
	if hasCommand("iw") {
		return s.scanViaIw(ctx)
	}

	return nil, &WiFiError{
		Message: "WiFi scan failed: no scanning method available; " +
			"ensure NetworkManager is running, or install nmcli or iw",
	}
}

// scanViaNM scans using the NetworkManager D-Bus API.
func (s *platformWiFiScanner) scanViaNM(ctx context.Context) ([]WiFiNetwork, error) {
	wireless, err := nmWirelessDevice()
	if err != nil {
		return nil, err
	}

	// Request a fresh scan (ignore "already scanning" errors).
	if scanErr := wireless.RequestScan(); scanErr != nil {
		if !strings.Contains(scanErr.Error(), "Scanning not allowed") {
			return nil, scanErr
		}
	}

	// Brief pause for results.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
	}

	aps, err := wireless.GetAllAccessPoints()
	if err != nil {
		return nil, err
	}

	networks := make([]WiFiNetwork, 0, len(aps))
	for _, ap := range aps {
		if n := apToNetwork(ap); n != nil {
			networks = append(networks, *n)
		}
	}

	return networks, nil
}

// scanViaNmcli scans for WiFi networks using the nmcli command.
func (s *platformWiFiScanner) scanViaNmcli(ctx context.Context) ([]WiFiNetwork, error) {
	cmd := exec.CommandContext(ctx, "nmcli", "-t", "-f", "SSID,BSSID,SIGNAL,CHAN,SECURITY",
		"device", "wifi", "list", "--rescan", "yes")
	output, err := cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "nmcli scan failed", Err: err}
	}

	return parseNmcliScanOutput(string(output)), nil
}

// parseNmcliScanOutput parses the terse output of nmcli device wifi list.
func parseNmcliScanOutput(output string) []WiFiNetwork {
	var networks []WiFiNetwork
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if n := parseNmcliScanLine(line); n != nil {
			networks = append(networks, *n)
		}
	}
	return networks
}

// parseNmcliScanLine parses a single line from nmcli terse wifi list.
func parseNmcliScanLine(line string) *WiFiNetwork {
	fields := splitNmcliFields(line)
	if len(fields) < 1 || fields[0] == "" {
		return nil
	}

	network := &WiFiNetwork{
		SSID:     fields[0],
		LastSeen: time.Now(),
	}

	if len(fields) >= 2 {
		network.BSSID = fields[1]
	}
	if len(fields) >= 3 {
		if signal, err := strconv.Atoi(fields[2]); err == nil {
			network.Signal = signal - 100
		}
	}
	if len(fields) >= 4 {
		if ch, err := strconv.Atoi(fields[3]); err == nil {
			network.Channel = ch
		}
	}
	if len(fields) >= 5 {
		network.Security = fields[4]
	}

	return network
}

// splitNmcliFields splits a nmcli terse line on unescaped colons.
func splitNmcliFields(line string) []string {
	var fields []string
	var current strings.Builder
	for i := 0; i < len(line); i++ {
		switch {
		case line[i] == '\\' && i+1 < len(line) && line[i+1] == ':':
			current.WriteByte(':')
			i++
		case line[i] == ':':
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteByte(line[i])
		}
	}
	fields = append(fields, current.String())
	return fields
}

// scanViaIw scans using the iw command.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input
func (s *platformWiFiScanner) scanViaIw(ctx context.Context) ([]WiFiNetwork, error) {
	cmd := exec.CommandContext(ctx, "iw", "dev", s.iface, "scan")
	output, err := cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "iw scan failed", Err: err}
	}

	return parseIwScanOutput(string(output)), nil
}

// parseIwScanOutput parses the output of iw dev <iface> scan.
func parseIwScanOutput(output string) []WiFiNetwork {
	var networks []WiFiNetwork
	var current *WiFiNetwork

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(line, "BSS ") {
			if current != nil && current.SSID != "" {
				networks = append(networks, *current)
			}
			current = &WiFiNetwork{LastSeen: time.Now()}
			current.BSSID = parseBSSID(line)
			continue
		}

		if current == nil {
			continue
		}

		parseIwField(current, trimmed)
	}

	if current != nil && current.SSID != "" {
		networks = append(networks, *current)
	}

	return networks
}

// parseBSSID extracts BSSID from a BSS line like "BSS aa:bb:cc:dd:ee:ff(on wlan0)".
func parseBSSID(line string) string {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return ""
	}
	bssid := parts[1]
	if idx := strings.Index(bssid, "("); idx != -1 {
		bssid = bssid[:idx]
	}
	return strings.ToUpper(bssid)
}

// parseIwField parses a single field line from iw scan output into a WiFiNetwork.
func parseIwField(n *WiFiNetwork, line string) {
	switch {
	case strings.HasPrefix(line, "SSID: "):
		n.SSID = strings.TrimPrefix(line, "SSID: ")
	case strings.HasPrefix(line, "signal: "):
		sigStr := strings.TrimPrefix(line, "signal: ")
		sigStr = strings.TrimSuffix(sigStr, " dBm")
		if f, err := strconv.ParseFloat(sigStr, 64); err == nil {
			n.Signal = int(f)
		}
	case strings.HasPrefix(line, "freq: "):
		if freq, err := strconv.Atoi(strings.TrimPrefix(line, "freq: ")); err == nil {
			n.Channel = frequencyToChannel(freq)
		}
	case strings.HasPrefix(line, "RSN:") && n.Security == "":
		n.Security = "WPA2"
	case strings.HasPrefix(line, "WPA:") && n.Security == "":
		n.Security = "WPA"
	}
}

// ─── Connect ─────────────────────────────────────────────────────────────────

// Connect connects to a WiFi network on Linux.
func (s *platformWiFiScanner) Connect(ctx context.Context, ssid, password string) error {
	// Primary: NetworkManager D-Bus.
	if err := s.connectViaNM(ctx, ssid, password); err == nil {
		return nil
	}

	// Fallback: nmcli CLI.
	if hasCommand("nmcli") {
		if err := s.connectNmcli(ctx, ssid, password); err == nil {
			return nil
		} else if !hasCommand("wpa_cli") && !hasCommand("iwconfig") {
			return err
		}
	}

	// Fallback: wpa_cli.
	if hasCommand("wpa_cli") {
		if err := s.connectWpaCli(ctx, ssid, password); err == nil {
			return nil
		} else if !hasCommand("iwconfig") {
			return err
		}
	}

	// Fallback: iwconfig (deprecated).
	if hasCommand("iwconfig") {
		return s.connectIwconfig(ctx, ssid, password)
	}

	return ErrToolNotFound
}

// connectViaNM connects using the NetworkManager D-Bus API.
func (s *platformWiFiScanner) connectViaNM(ctx context.Context, ssid, password string) error {
	nm, err := gnm.NewNetworkManager()
	if err != nil {
		return err
	}

	wireless, err := nmWirelessDevice()
	if err != nil {
		return err
	}

	settings := map[string]map[string]interface{}{
		"connection": {
			"type": "802-11-wireless",
			"id":   ssid,
		},
		"802-11-wireless": {
			"ssid": []byte(ssid),
			"mode": "infrastructure",
		},
	}

	if password != "" {
		settings["802-11-wireless-security"] = map[string]interface{}{
			"key-mgmt": "wpa-psk",
			"psk":      password,
		}
		settings["802-11-wireless"]["security"] = "802-11-wireless-security"
	}

	activeConn, err := nm.AddAndActivateConnection(settings, wireless)
	if err != nil {
		return &WiFiError{Message: "NM connect failed", Err: err}
	}

	// Wait for activation.
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		state, err := activeConn.GetPropertyState()
		if err == nil {
			switch state {
			case gnm.NmActiveConnectionStateActivated:
				return nil
			case gnm.NmActiveConnectionStateDeactivated:
				return &WiFiError{Message: "connection was deactivated"}
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	return ErrConnectionTimeout
}

// connectNmcli connects using nmcli.
func (s *platformWiFiScanner) connectNmcli(ctx context.Context, ssid, password string) error {
	var cmd *exec.Cmd
	if password == "" {
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

	return s.waitForConnection(ctx, ssid, 15*time.Second)
}

// connectWpaCli connects using wpa_supplicant's wpa_cli.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input; network ID from wpa_cli output
func (s *platformWiFiScanner) connectWpaCli(ctx context.Context, ssid, password string) error {
	addCmd := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "add_network")
	output, err := addCmd.Output()
	if err != nil {
		return &WiFiError{Message: "wpa_cli add_network failed", Err: err}
	}
	networkID := strings.TrimSpace(string(output))

	setSSID := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "set_network", networkID, "ssid", `"`+ssid+`"`)
	if err := setSSID.Run(); err != nil {
		return &WiFiError{Message: "wpa_cli set ssid failed", Err: err}
	}

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

	enableCmd := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "enable_network", networkID)
	if err := enableCmd.Run(); err != nil {
		return &WiFiError{Message: "wpa_cli enable_network failed", Err: err}
	}

	selectCmd := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "select_network", networkID)
	if err := selectCmd.Run(); err != nil {
		return &WiFiError{Message: "wpa_cli select_network failed", Err: err}
	}

	return s.waitForConnection(ctx, ssid, 15*time.Second)
}

// connectIwconfig connects using iwconfig (deprecated).
//
//nolint:gosec // G204: Interface name is auto-detected, not user input
func (s *platformWiFiScanner) connectIwconfig(ctx context.Context, ssid, password string) error {
	essidCmd := exec.CommandContext(ctx, "iwconfig", s.iface, "essid", ssid)
	if err := essidCmd.Run(); err != nil {
		return &WiFiError{Message: "iwconfig essid failed", Err: err}
	}

	if password != "" {
		keyCmd := exec.CommandContext(ctx, "iwconfig", s.iface, "key", "s:"+password)
		if err := keyCmd.Run(); err != nil {
			return &WiFiError{Message: "iwconfig key failed", Err: err}
		}
	}

	upCmd := exec.CommandContext(ctx, "ip", "link", "set", s.iface, "up")
	if err := upCmd.Run(); err != nil {
		upCmd = exec.CommandContext(ctx, "ifconfig", s.iface, "up")
		if err := upCmd.Run(); err != nil {
			return &WiFiError{Message: "failed to bring interface up", Err: err}
		}
	}

	return s.waitForConnection(ctx, ssid, 15*time.Second)
}

// ─── Disconnect ──────────────────────────────────────────────────────────────

// Disconnect disconnects from the current WiFi network.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input
func (s *platformWiFiScanner) Disconnect(ctx context.Context) error {
	// Primary: NM D-Bus.
	if wireless, err := nmWirelessDevice(); err == nil {
		if err := wireless.Disconnect(); err == nil {
			return nil
		}
	}

	if hasCommand("nmcli") {
		cmd := exec.CommandContext(ctx, "nmcli", "device", "disconnect", s.iface)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	if hasCommand("wpa_cli") {
		cmd := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "disconnect")
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	if hasCommand("iwconfig") {
		cmd := exec.CommandContext(ctx, "iwconfig", s.iface, "essid", "off")
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return ErrToolNotFound
}

// ─── CurrentNetwork ──────────────────────────────────────────────────────────

// CurrentNetwork returns the currently connected WiFi network.
func (s *platformWiFiScanner) CurrentNetwork(ctx context.Context) (*WiFiNetwork, error) {
	// Primary: NM D-Bus.
	if wireless, err := nmWirelessDevice(); err == nil {
		if ap, err := wireless.GetPropertyActiveAccessPoint(); err == nil && ap != nil {
			if n := apToNetwork(ap); n != nil {
				return n, nil
			}
		}
	}

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
	cmd := exec.CommandContext(ctx, "nmcli", "-t", "-f", "IN-USE,SSID,SIGNAL,SECURITY", "device", "wifi", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "nmcli wifi list failed", Err: err}
	}

	for _, line := range strings.Split(string(output), "\n") {
		if n := s.parseNmcliLine(line); n != nil {
			return n, nil
		}
	}

	return nil, &WiFiError{Message: "not connected to any WiFi network"}
}

// parseNmcliLine parses a single line from nmcli wifi list output.
func (s *platformWiFiScanner) parseNmcliLine(line string) *WiFiNetwork {
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
		if signal, err := strconv.Atoi(parts[1]); err == nil {
			network.Signal = signal
		}
	}

	if len(parts) >= 3 {
		network.Security = parts[2]
	}

	return network
}

// currentNetworkWpaCli gets current network using wpa_cli.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input
func (s *platformWiFiScanner) currentNetworkWpaCli(ctx context.Context) (*WiFiNetwork, error) {
	cmd := exec.CommandContext(ctx, "wpa_cli", "-i", s.iface, "status")
	output, err := cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "wpa_cli status failed", Err: err}
	}

	var ssid, state string
	for _, line := range strings.Split(string(output), "\n") {
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

	return &WiFiNetwork{SSID: ssid, LastSeen: time.Now()}, nil
}

// currentNetworkIwconfig gets current network using iwconfig.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input
func (s *platformWiFiScanner) currentNetworkIwconfig(ctx context.Context) (*WiFiNetwork, error) {
	cmd := exec.CommandContext(ctx, "iwconfig", s.iface)
	output, err := cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "iwconfig failed", Err: err}
	}

	outputStr := string(output)
	essidIdx := strings.Index(outputStr, `ESSID:"`)
	if essidIdx == -1 {
		return nil, &WiFiError{Message: "not connected to any WiFi network"}
	}

	start := essidIdx + 7
	end := strings.Index(outputStr[start:], `"`)
	if end == -1 {
		return nil, &WiFiError{Message: "failed to parse ESSID"}
	}

	ssid := outputStr[start : start+end]
	if ssid == "" || ssid == "off/any" {
		return nil, &WiFiError{Message: "not connected to any WiFi network"}
	}

	return &WiFiNetwork{SSID: ssid, LastSeen: time.Now()}, nil
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

// hasCommand checks if a command is available in PATH.
func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
