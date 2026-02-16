//go:build linux

package discovery

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gnm "github.com/Wifx/gonetworkmanager/v2"
	"github.com/mdlayher/wifi"
)

// WiFi security protocol names.
const (
	securityWPA  = "WPA"
	securityWPA2 = "WPA2"
	securityWPA3 = "WPA3"
	securityWEP  = "WEP"
)

// platformWiFiScanner implements WiFiScanner for Linux.
// Primary: nl80211 netlink (pure Go, zero external dependencies).
// Fallback 1: NetworkManager D-Bus API.
// Fallback 2: nmcli, iw CLI tools.
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

// ─── nl80211 netlink scanner (primary — zero external dependencies) ──────────

// ensureInterfaceUp brings the WiFi interface up if it's currently down.
// Scanning requires the interface to be in an "up" state.
// The interface is left up after this call — callers that need WiFi (scan,
// connect, provision) inherently require an active interface.
//
//nolint:gosec // G204: Interface name is auto-detected from /sys/class/net, not user input
func (s *platformWiFiScanner) ensureInterfaceUp(ctx context.Context) {
	state, err := os.ReadFile("/sys/class/net/" + s.iface + "/operstate")
	if err != nil {
		return
	}
	if strings.TrimSpace(string(state)) != "down" {
		return
	}
	// Interface is down — try to bring it up.
	// Failure is non-fatal: scan will fail with a more descriptive error.
	if err := exec.CommandContext(ctx, "ip", "link", "set", s.iface, "up").Run(); err != nil {
		return
	}
	// Brief pause for the interface to initialize.
	time.Sleep(500 * time.Millisecond)
}

// scanViaNl80211 scans using the kernel's nl80211 netlink interface directly.
// This requires no external services or binaries — just a WiFi-capable kernel.
func (s *platformWiFiScanner) scanViaNl80211(ctx context.Context) ([]WiFiNetwork, error) {
	// Ensure the interface is up before scanning.
	s.ensureInterfaceUp(ctx)

	client, err := wifi.New()
	if err != nil {
		return nil, fmt.Errorf("nl80211 client: %w", err)
	}
	defer client.Close()

	// Find our WiFi interface.
	ifaces, err := client.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("nl80211 interfaces: %w", err)
	}

	var ifi *wifi.Interface
	for _, i := range ifaces {
		if i.Name == s.iface {
			ifi = i
			break
		}
	}
	if ifi == nil {
		// Try any station-mode interface.
		for _, i := range ifaces {
			if i.Type == wifi.InterfaceTypeStation {
				ifi = i
				break
			}
		}
	}
	if ifi == nil {
		return nil, fmt.Errorf("nl80211: no WiFi interface %q found", s.iface)
	}

	// Trigger a fresh scan. This requires CAP_NET_ADMIN.
	if scanErr := client.Scan(ctx, ifi); scanErr != nil {
		// If scan fails (permissions, busy, etc.), still try to get cached results.
		// GetAllAccessPoints returns whatever the kernel already knows.
		aps, apErr := client.AccessPoints(ifi)
		if apErr != nil || len(aps) == 0 {
			return nil, fmt.Errorf("nl80211 scan: %w", scanErr)
		}
		// We got cached results despite scan failure — use them.
		return bssListToNetworks(aps), nil
	}

	// Get scan results.
	aps, err := client.AccessPoints(ifi)
	if err != nil {
		return nil, fmt.Errorf("nl80211 access points: %w", err)
	}

	return bssListToNetworks(aps), nil
}

// bssListToNetworks converts nl80211 BSS results to WiFiNetwork structs.
func bssListToNetworks(bssList []*wifi.BSS) []WiFiNetwork {
	networks := make([]WiFiNetwork, 0, len(bssList))
	for _, bss := range bssList {
		if bss.SSID == "" {
			continue
		}
		network := WiFiNetwork{
			SSID:     bss.SSID,
			BSSID:    bss.BSSID.String(),
			Signal:   int(bss.Signal / 100), // mBm to dBm
			Channel:  frequencyToChannel(bss.Frequency),
			LastSeen: time.Now(),
		}

		// Determine security from RSN info.
		if len(bss.RSN.AKMs) > 0 {
			network.Security = rsnToSecurity(bss.RSN)
		}

		networks = append(networks, network)
	}
	return networks
}

// rsnToSecurity converts RSN info to a human-readable security string.
func rsnToSecurity(rsn wifi.RSNInfo) string {
	for _, akm := range rsn.AKMs {
		switch akm {
		case wifi.RSNAkmSAE, wifi.RSNAkmFTSAE:
			return securityWPA3
		}
	}
	if rsn.Version >= 1 {
		return securityWPA2
	}
	return securityWPA
}

// ─── NetworkManager D-Bus helpers ────────────────────────────────────────────

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
			return securityWPA3
		}
		return securityWPA2
	}
	if wpaFlags != 0 {
		return securityWPA
	}
	if flags&nmAPFlagPrivacy != 0 {
		return securityWEP
	}
	return ""
}

// ─── Scan ────────────────────────────────────────────────────────────────────

// Scan scans for available WiFi networks.
// Tries nl80211 netlink first (zero dependencies), then NetworkManager D-Bus,
// then falls back to CLI tools.
func (s *platformWiFiScanner) Scan(ctx context.Context) ([]WiFiNetwork, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var errors []string

	// Primary: nl80211 netlink (pure Go, no external dependencies).
	if networks, err := s.scanViaNl80211(ctx); err == nil {
		return networks, nil
	} else {
		errors = append(errors, "nl80211: "+err.Error())
	}

	// Fallback 1: NetworkManager D-Bus.
	if networks, err := s.scanViaNM(ctx); err == nil {
		return networks, nil
	} else {
		errors = append(errors, "NetworkManager D-Bus: "+err.Error())
	}

	// Fallback 2: nmcli CLI.
	if hasCommand("nmcli") {
		if networks, err := s.scanViaNmcli(ctx); err == nil {
			return networks, nil
		} else {
			errors = append(errors, "nmcli: "+err.Error())
		}
	}

	// Fallback 3: iw CLI.
	if hasCommand("iw") {
		networks, err := s.scanViaIw(ctx)
		if err == nil {
			return networks, nil
		}
		errors = append(errors, "iw: "+err.Error())
	}

	return nil, &WiFiError{
		Message: "WiFi scan failed, all methods tried: " +
			strings.Join(errors, "; "),
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
		n.Security = securityWPA2
	case strings.HasPrefix(line, "WPA:") && n.Security == "":
		n.Security = securityWPA
	}
}

// ─── Connect ─────────────────────────────────────────────────────────────────

// Connect connects to a WiFi network on Linux.
func (s *platformWiFiScanner) Connect(ctx context.Context, ssid, password string) error {
	var errors []string

	// Primary: nl80211 netlink (pure Go).
	if err := s.connectViaNl80211(ctx, ssid, password); err == nil {
		return nil
	} else {
		errors = append(errors, "nl80211: "+err.Error())
	}

	// Fallback 1: NetworkManager D-Bus.
	if err := s.connectViaNM(ctx, ssid, password); err == nil {
		return nil
	} else {
		errors = append(errors, "NetworkManager D-Bus: "+err.Error())
	}

	// Fallback 2: nmcli CLI.
	if hasCommand("nmcli") {
		if err := s.connectNmcli(ctx, ssid, password); err == nil {
			return nil
		} else {
			errors = append(errors, "nmcli: "+err.Error())
		}
	}

	// Fallback 3: wpa_cli.
	if hasCommand("wpa_cli") {
		if err := s.connectWpaCli(ctx, ssid, password); err == nil {
			return nil
		} else {
			errors = append(errors, "wpa_cli: "+err.Error())
		}
	}

	// Fallback 4: iwconfig (deprecated).
	if hasCommand("iwconfig") {
		if err := s.connectIwconfig(ctx, ssid, password); err == nil {
			return nil
		} else {
			errors = append(errors, "iwconfig: "+err.Error())
		}
	}

	return &WiFiError{
		Message: "WiFi connect failed, all methods tried: " +
			strings.Join(errors, "; "),
	}
}

// connectViaNl80211 connects using the nl80211 netlink interface.
func (s *platformWiFiScanner) connectViaNl80211(ctx context.Context, ssid, password string) error {
	// Ensure the interface is up.
	s.ensureInterfaceUp(ctx)

	client, err := wifi.New()
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	defer client.Close()

	ifi, err := s.findNl80211Interface(client)
	if err != nil {
		return fmt.Errorf("interface: %w", err)
	}

	// Disconnect from any current network first to avoid state issues.
	// Errors are expected if not currently connected — ignore them.
	if disconnectErr := client.Disconnect(ifi); disconnectErr != nil {
		// Not connected — that's fine, proceed with connect.
		_ = disconnectErr
	}
	time.Sleep(500 * time.Millisecond)

	// Connect to the target network.
	if password == "" {
		err = client.Connect(ifi, ssid)
	} else {
		err = client.ConnectWPAPSK(ifi, ssid, password)
	}
	if err != nil {
		return fmt.Errorf("connect %q: %w", ssid, err)
	}

	// Wait for association and DHCP. Shelly APs run a DHCP server
	// at 192.168.33.1 — the kernel DHCP client needs time to obtain a lease.
	time.Sleep(3 * time.Second)

	return nil
}

// findNl80211Interface finds the WiFi interface for nl80211 operations.
func (s *platformWiFiScanner) findNl80211Interface(client *wifi.Client) (*wifi.Interface, error) {
	ifaces, err := client.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, i := range ifaces {
		if i.Name == s.iface {
			return i, nil
		}
	}
	for _, i := range ifaces {
		if i.Type == wifi.InterfaceTypeStation {
			return i, nil
		}
	}
	return nil, fmt.Errorf("no WiFi interface found for nl80211")
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
	// Primary: nl80211 netlink.
	if client, err := wifi.New(); err == nil {
		defer client.Close()
		if ifi, err := s.findNl80211Interface(client); err == nil {
			if err := client.Disconnect(ifi); err == nil {
				// Brief pause to let the disassociation complete.
				time.Sleep(500 * time.Millisecond)
				return nil
			}
		}
	}

	// Fallback 1: NM D-Bus.
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

// currentNetworkViaNl80211 tries to get the current network via nl80211 netlink.
func (s *platformWiFiScanner) currentNetworkViaNl80211() *WiFiNetwork {
	client, err := wifi.New()
	if err != nil {
		return nil
	}
	defer client.Close()

	ifi, err := s.findNl80211Interface(client)
	if err != nil {
		return nil
	}

	bss, err := client.BSS(ifi)
	if err != nil || bss == nil || bss.SSID == "" {
		return nil
	}

	networks := bssListToNetworks([]*wifi.BSS{bss})
	if len(networks) == 0 {
		return nil
	}
	return &networks[0]
}

// CurrentNetwork returns the currently connected WiFi network.
func (s *platformWiFiScanner) CurrentNetwork(ctx context.Context) (*WiFiNetwork, error) {
	// Primary: nl80211 netlink.
	if n := s.currentNetworkViaNl80211(); n != nil {
		return n, nil
	}

	// Fallback 1: NM D-Bus.
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
