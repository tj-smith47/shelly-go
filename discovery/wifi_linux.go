//go:build linux

package discovery

import (
	"bytes"
	"context"
	"fmt"
	"net"
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

// Connect-method label constants used as human-readable names in connectMethod.
const (
	methodWpaCli  = "wpa_cli"
	methodNl80211 = "nl80211"
)

// defaultWiFiIface is the conventional name for the primary WiFi
// interface, used as the detection fallback when no interface is found.
const defaultWiFiIface = "wlan0"

// WiFi association deadlines. An infrastructure network associates within a few
// seconds, so a tight deadline surfaces a real failure (wrong PSK, AP gone)
// quickly. A Shelly factory AP is a low-power radio that is frequently weak or
// only intermittently broadcasting; wpa_supplicant keeps scanning and
// re-associating the selected network in the background, so a longer deadline
// lets a connect ride through a flapping AP instead of failing the whole hop on
// a single missed scan window.
const (
	wifiConnectTimeout = 15 * time.Second
	apConnectTimeout   = 60 * time.Second
)

// connectTimeout returns the association deadline for ssid: the longer
// apConnectTimeout for a Shelly factory AP, the tight wifiConnectTimeout for an
// ordinary infrastructure network.
func connectTimeout(ssid string) time.Duration {
	if IsShellyAP(ssid) {
		return apConnectTimeout
	}
	return wifiConnectTimeout
}

// platformWiFiScanner implements WiFiScanner for Linux.
// Primary: nl80211 netlink (pure Go, zero external dependencies).
// Fallback 1: NetworkManager D-Bus API.
// Fallback 2: nmcli, iw CLI tools.
type platformWiFiScanner struct {
	// wpaRun, when non-nil, replaces the real wpa_cli invocation in wpa(). It
	// exists so the connect/rollback command sequence can be exercised without a
	// live wpa_supplicant; production code leaves it nil.
	wpaRun func(ctx context.Context, args ...string) (string, error)

	iface    string
	apHostIP string // static host IPv4 used on a Shelly AP subnet (no mask)
}

// newPlatformWiFiScanner creates a platform-specific WiFi scanner for Linux.
func newPlatformWiFiScanner() WiFiScanner {
	return &platformWiFiScanner{
		iface:    detectWiFiInterface(),
		apHostIP: DefaultAPHostIP,
	}
}

// SetAPHostIP overrides the static host IP assigned on a Shelly AP subnet.
// Empty input is ignored so the default remains in effect. Implements
// APHostIPSetter.
func (s *platformWiFiScanner) SetAPHostIP(ip string) {
	if ip != "" {
		s.apHostIP = ip
	}
}

// apHostCIDR returns the host's AP-subnet address in CIDR form (e.g.
// "192.168.33.133/24"), falling back to the default if unset.
func (s *platformWiFiScanner) apHostCIDR() string {
	ip := s.apHostIP
	if ip == "" {
		ip = DefaultAPHostIP
	}
	return ip + "/24"
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

	commonNames := []string{defaultWiFiIface, "wlan1", "wlp2s0", "wlp3s0", "wlp6s0", "wlo1"}
	for _, name := range commonNames {
		if _, err := os.Stat("/sys/class/net/" + name); err == nil {
			return name
		}
	}

	return defaultWiFiIface
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

// connectMethod is one association strategy paired with the label used for it in
// the aggregated "all methods tried" error.
type connectMethod struct {
	fn   func(context.Context, string, string) error
	name string
}

// connectMethods returns the association strategies to attempt, in order.
//
// A raw nl80211 association fights a running wpa_supplicant for control of the
// interface: the kernel rejects the second driver with EALREADY ("operation
// already in progress"), and the half-issued netlink op then wedges the link so
// even the cooperating wpa_cli path times out behind it. So when wpa_supplicant
// owns the interface, wpa_cli leads and raw nl80211 is omitted entirely;
// otherwise nl80211 (pure Go, zero dependencies) leads.
func (s *platformWiFiScanner) connectMethods(viaSupplicant bool) []connectMethod {
	var methods []connectMethod
	if viaSupplicant {
		methods = append(methods, connectMethod{s.connectWpaCli, methodWpaCli})
	} else {
		methods = append(methods, connectMethod{s.connectViaNl80211, methodNl80211})
	}
	methods = append(methods, connectMethod{s.connectViaNM, "NetworkManager D-Bus"})
	if hasCommand("nmcli") {
		methods = append(methods, connectMethod{s.connectNmcli, "nmcli"})
	}
	if !viaSupplicant && hasCommand(methodWpaCli) {
		methods = append(methods, connectMethod{s.connectWpaCli, methodWpaCli})
	}
	if hasCommand("iwconfig") {
		methods = append(methods, connectMethod{s.connectIwconfig, "iwconfig"})
	}
	return methods
}

// Connect connects to a WiFi network on Linux.
func (s *platformWiFiScanner) Connect(ctx context.Context, ssid, password string) error {
	viaSupplicant := hasCommand(methodWpaCli) && s.wpaSupplicantManages(ctx)

	var errs []string
	connected := false
	for _, m := range s.connectMethods(viaSupplicant) {
		if err := m.fn(ctx, ssid, password); err == nil {
			connected = true
			break
		} else {
			errs = append(errs, m.name+": "+err.Error())
		}
	}

	if !connected {
		return &WiFiError{
			Message: "WiFi connect failed, all methods tried: " +
				strings.Join(errs, "; "),
		}
	}

	// Ensure the interface carries a usable address for the network just joined.
	// A Shelly device in AP mode runs an unreliable DHCP server that also rewrites
	// /etc/resolv.conf, so a static IP on its 192.168.33.0/24 subnet is assigned
	// instead; every other network must shed any leftover AP address so its own
	// DHCP lease is the only one. This applies to ALL connect methods — nl80211
	// and the wpa_cli/nmcli/NetworkManager fallbacks alike. Previously only the
	// nl80211 path assigned the AP IP, so a host that fell back to wpa_cli (e.g.
	// where wpa_supplicant owns the interface) associated with the AP but had no
	// route to the device at 192.168.33.1.
	if IsShellyAP(ssid) {
		s.obtainIPAddress(ctx, s.iface)
	} else {
		s.removeAPStaticIP(ctx, s.iface)
		s.reacquireDHCP(ctx, s.iface)
	}
	return nil
}

// reacquireDHCP obtains a DHCP lease on iface after returning to a real network.
// Driving association manually (wpa_cli/nl80211) bypasses the host's normal
// DHCP trigger, so a host whose WiFi this scanner connected can be left without
// a lease — losing all connectivity on that interface until something re-leases
// it. It is skipped when the interface already carries a routable IPv4 (a managed
// host — NetworkManager or systemd-networkd — already handled it), so it never
// fights an existing manager. Best-effort: the first available DHCP client wins
// and any failure is left for the caller's next operation to surface.
func (s *platformWiFiScanner) reacquireDHCP(ctx context.Context, iface string) {
	if hasRoutableIPv4(iface) {
		return
	}
	// Ordered by ubiquity; -1/-n/-q variants make a single bounded attempt and
	// exit rather than daemonizing.
	clients := [][]string{
		{"dhclient", "-1", iface},
		{"dhcpcd", "-n", iface},
		{"udhcpc", "-i", iface, "-q", "-n"},
	}
	for _, c := range clients {
		if !hasCommand(c[0]) {
			continue
		}
		//nolint:gosec // G204: client name is a fixed literal; iface comes from the kernel
		if err := exec.CommandContext(ctx, c[0], c[1:]...).Run(); err == nil {
			return
		}
	}
}

// hasRoutableIPv4 reports whether iface carries an IPv4 address usable on a real
// network — excluding loopback, link-local (169.254), and the Shelly AP subnet
// (192.168.33.0/24), none of which indicate a working home-network lease.
func hasRoutableIPv4(iface string) bool {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return false
	}
	addrs, err := ifi.Addrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip4 := ipNet.IP.To4()
		if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
			continue
		}
		if strings.HasPrefix(ip4.String(), "192.168.33.") {
			continue
		}
		return true
	}
	return false
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

	// Remove any lingering Shelly AP static IP from a previous provisioning
	// session. This prevents stale 192.168.33.x addresses from polluting
	// subnet detection after reconnecting to the home network.
	s.removeAPStaticIP(ctx, ifi.Name)

	// Disconnect from any current network first to avoid state issues.
	// Errors are expected if not currently connected — ignore them.
	if disconnectErr := client.Disconnect(ifi); disconnectErr != nil {
		// Not connected — that's fine, proceed with connect.
		_ = disconnectErr
	}
	time.Sleep(500 * time.Millisecond)

	// Trigger a scan so the kernel has the target BSS in its cache.
	// Without this, Connect fails with EINVAL if the BSS was evicted
	// (e.g. after reconnecting to a different network between provisions).
	if scanErr := client.Scan(ctx, ifi); scanErr != nil {
		// Non-fatal: cached results may still contain the target.
		_ = scanErr
	}

	// Connect to the target network.
	if password == "" {
		err = client.Connect(ifi, ssid)
	} else {
		err = client.ConnectWPAPSK(ifi, ssid, password)
	}
	if err != nil {
		return fmt.Errorf("connect %q: %w", ssid, err)
	}

	// Wait for association to complete. IP addressing is applied centrally by
	// Connect once any method succeeds, so it covers every connect path.
	time.Sleep(1 * time.Second)

	return nil
}

// removeAPStaticIP removes any lingering Shelly AP static IP (192.168.33.10/24)
// from the WiFi interface. This is safe to call even if the address was never
// assigned — the error is silently ignored.
//
//nolint:gosec // G204: Interface name is from kernel, not user input
func (s *platformWiFiScanner) removeAPStaticIP(ctx context.Context, ifaceName string) {
	staticIP := s.apHostCIDR()
	// The AP static IP is usually absent (exit 2, "RTNETLINK answers: Cannot assign
	// requested address") when none was ever assigned — the normal case. The delete
	// is best-effort: a failure here must not block returning to the home network,
	// so the checked error is intentionally not propagated.
	if err := exec.CommandContext(ctx, "ip", "addr", "del", staticIP, "dev", ifaceName).Run(); err != nil {
		return
	}
}

// obtainIPAddress assigns the host a static IP on the Shelly AP subnet. Shelly
// devices in AP mode serve 192.168.33.1, so the host takes apHostIP
// (DefaultAPHostIP, .133, unless overridden via SetAPHostIP) to stay clear of the
// device (.1) and the typical DHCP range (.100+).
//
// We intentionally avoid running dhclient/dhcpcd because DHCP from the Shelly
// AP overwrites /etc/resolv.conf with nameserver 192.168.33.1, which breaks
// DNS on the host and is never restored after disconnect.
//
//nolint:gosec // G204: Interface name is from kernel, not user input
func (s *platformWiFiScanner) obtainIPAddress(ctx context.Context, ifaceName string) {
	staticIP := s.apHostCIDR()
	if err := exec.CommandContext(ctx, "ip", "addr", "add", staticIP, "dev", ifaceName).Run(); err != nil {
		// May fail if address already assigned — that's fine.
		return
	}
	time.Sleep(500 * time.Millisecond)
}

// HostNetworkPassword recovers the passphrase the host has stored for ssid,
// trying NetworkManager (nmcli) first, then wpa_supplicant config files. Only a
// plaintext passphrase is returned; a network whose secret is stored solely as a
// 64-hex pre-computed PSK cannot be reversed to a passphrase and yields an error.
// Reading wpa_supplicant config generally requires root.
func (s *platformWiFiScanner) HostNetworkPassword(ctx context.Context, ssid string) (string, error) {
	if ssid == "" {
		return "", fmt.Errorf("ssid is required")
	}
	if pw, err := hostPasswordViaNmcli(ctx, ssid); err == nil && pw != "" {
		return pw, nil
	}
	if pw := hostPasswordViaWpaSupplicant(ssid); pw != "" {
		return pw, nil
	}
	return "", fmt.Errorf("no stored passphrase found for %q on this host", ssid)
}

// hostPasswordViaNmcli asks NetworkManager for the stored passphrase of the
// connection whose id matches ssid. -s reveals secrets; -g extracts one field.
//
//nolint:gosec // G204: ssid names a network the host already trusts, not untrusted input
func hostPasswordViaNmcli(ctx context.Context, ssid string) (string, error) {
	if !hasCommand("nmcli") {
		return "", fmt.Errorf("nmcli not available")
	}
	out, err := exec.CommandContext(ctx, "nmcli", "-s", "-g",
		"802-11-wireless-security.psk", "connection", "show", "id", ssid).Output()
	if err != nil {
		return "", fmt.Errorf("nmcli connection show %q: %w", ssid, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// wpaSupplicantConfigGlobs lists the standard wpa_supplicant config locations,
// including the systemd per-interface variant (wpa_supplicant-<iface>.conf).
var wpaSupplicantConfigGlobs = []string{
	"/etc/wpa_supplicant/wpa_supplicant-*.conf",
	"/etc/wpa_supplicant/wpa_supplicant.conf",
	"/etc/wpa_supplicant.conf",
}

// hostPasswordViaWpaSupplicant scans wpa_supplicant config files for a network
// block whose ssid matches and returns its plaintext psk. Returns "" when no
// usable passphrase is found (unknown network, or only a hashed psk stored).
func hostPasswordViaWpaSupplicant(ssid string) string {
	for _, glob := range wpaSupplicantConfigGlobs {
		paths, err := filepath.Glob(glob)
		if err != nil {
			continue
		}
		for _, path := range paths {
			data, readErr := os.ReadFile(path) //nolint:gosec // G304: fixed system config paths, not user input
			if readErr != nil {
				continue
			}
			if pw := wpaPSKForSSID(string(data), ssid); pw != "" {
				return pw
			}
		}
	}
	return ""
}

// wpaPSKForSSID extracts the plaintext psk for ssid from wpa_supplicant config
// text. It walks network={...} blocks, matching ssid="..." and returning the
// block's psk when it is a quoted passphrase. An unquoted 64-hex psk is a
// pre-computed hash that cannot be turned back into a passphrase, so it is
// skipped. Returns "" when no matching block carries a usable passphrase.
func wpaPSKForSSID(conf, ssid string) string {
	var (
		inBlock   bool
		blockSSID string
		blockPSK  string
	)
	for _, raw := range strings.Split(conf, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "network="):
			inBlock, blockSSID, blockPSK = true, "", ""
		case inBlock && strings.HasPrefix(line, "}"):
			if blockSSID == ssid && blockPSK != "" {
				return blockPSK
			}
			inBlock = false
		case inBlock && strings.HasPrefix(line, "ssid="):
			blockSSID = unquoteWpaValue(strings.TrimPrefix(line, "ssid="))
		case inBlock && strings.HasPrefix(line, "psk="):
			// Only a quoted value is a usable passphrase; a bare 64-hex string is
			// a pre-computed PSK that cannot be reversed to a passphrase.
			if val := strings.TrimSpace(strings.TrimPrefix(line, "psk=")); strings.HasPrefix(val, `"`) {
				blockPSK = unquoteWpaValue(val)
			}
		}
	}
	return ""
}

// unquoteWpaValue strips surrounding double quotes from a wpa_supplicant value.
func unquoteWpaValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) {
		return v[1 : len(v)-1]
	}
	return v
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
			fieldType: "802-11-wireless",
			"id":      ssid,
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
	deadline := time.Now().Add(connectTimeout(ssid))
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

	return s.waitForConnection(ctx, ssid, connectTimeout(ssid))
}

// wpa runs a single wpa_cli subcommand against this scanner's interface and
// returns its trimmed stdout. The wpaRun seam (nil in production) lets the
// connect/rollback sequence be tested without a live wpa_supplicant.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input; args are fixed verbs + ids from wpa_cli output
func (s *platformWiFiScanner) wpa(ctx context.Context, args ...string) (string, error) {
	if s.wpaRun != nil {
		return s.wpaRun(ctx, args...)
	}
	full := append([]string{"-i", s.iface}, args...)
	out, err := exec.CommandContext(ctx, methodWpaCli, full...).Output()
	return strings.TrimSpace(string(out)), err
}

// tryWpa runs a wpa_cli subcommand for its side effect only, used on best-effort
// rollback paths where a failure leaves nothing further to attempt. The error is
// examined and intentionally not propagated.
func (s *platformWiFiScanner) tryWpa(ctx context.Context, args ...string) {
	if _, err := s.wpa(ctx, args...); err != nil {
		return
	}
}

// ForgetNetwork removes any wpa_supplicant network block configured for ssid from
// the running config — the transient block created to hop onto a device's open
// factory AP. It does not persist the change (no save_config): these blocks are
// runtime-only, so dropping them keeps wpa_supplicant from accumulating stale,
// disabled entries across a fleet of AP hops. Best-effort — a list failure is
// returned, but a block that is missing or currently associated is left alone.
func (s *platformWiFiScanner) ForgetNetwork(ctx context.Context, ssid string) error {
	out, err := s.wpa(ctx, "list_networks")
	if err != nil {
		return &WiFiError{Message: "wpa_cli list_networks failed", Err: err}
	}
	for _, n := range parseWpaNetworkList(out) {
		// Never remove the block the supplicant is currently on. ForgetNetwork runs
		// after the host has rejoined home, so the AP block is no longer [CURRENT];
		// skipping a current match avoids cutting a live association if mis-ordered.
		if n.ssid == ssid && !n.current {
			s.tryWpa(ctx, "remove_network", n.id)
		}
	}
	return nil
}

// wpaNetwork is one row of `wpa_cli list_networks` output.
type wpaNetwork struct {
	id      string
	ssid    string
	current bool // carries the [CURRENT] flag — the network wpa_supplicant is on
}

// parseWpaNetworkList parses `wpa_cli list_networks` output. The first line is a
// header ("network id / ssid / bssid / flags"); each remaining line is
// tab-separated: id, ssid, bssid, flags. A network whose flags contain [CURRENT]
// is the one wpa_supplicant is currently associated with.
func parseWpaNetworkList(out string) []wpaNetwork {
	lines := strings.Split(out, "\n")
	nets := make([]wpaNetwork, 0, len(lines))
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // header / blank
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		n := wpaNetwork{id: strings.TrimSpace(fields[0]), ssid: fields[1]}
		if len(fields) >= 4 {
			n.current = strings.Contains(fields[3], "[CURRENT]")
		}
		nets = append(nets, n)
	}
	return nets
}

// connectWpaCli connects using wpa_supplicant's wpa_cli. It reuses an existing
// network block for the SSID when one is present (so repeated AP hops do not leak
// a duplicate block per hop), and on any failure it rolls the supplicant back to
// the network it was on before — removing only a block it actually added and
// re-enabling the prior networks — so a failed hop never strands the host with
// every network disabled.
// wpaSupplicantManages reports whether a live wpa_supplicant instance is driving
// this scanner's interface. When it is, the interface must be associated through
// wpa_cli rather than a raw nl80211 connect, which the kernel would reject with
// EALREADY and leave wedged. Detected by a PONG on the per-interface wpa_cli
// control channel (also respects the wpaRun test seam).
func (s *platformWiFiScanner) wpaSupplicantManages(ctx context.Context) bool {
	out, err := s.wpa(ctx, "ping")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToUpper(out), "PONG")
}

func (s *platformWiFiScanner) connectWpaCli(ctx context.Context, ssid, password string) error {
	_, rollback, err := s.configureWpaNetwork(ctx, ssid, password)
	if err != nil {
		return err
	}

	if err := s.waitForConnection(ctx, ssid, connectTimeout(ssid)); err != nil {
		rollback()
		return err
	}
	return nil
}

// configureWpaNetwork selects (creating if absent) a wpa_cli network block for
// ssid/password and returns its id together with a rollback that undoes the
// in-memory change. select_network disables every other network in the running
// config, so without rollback a later failure would leave the host unable to fall
// back to its home network. On its own error path configureWpaNetwork has already
// rolled back. The returned rollback is idempotent-safe to call once on a later
// failure (e.g. association timeout).
func (s *platformWiFiScanner) configureWpaNetwork(
	ctx context.Context,
	ssid, password string,
) (networkID string, rollback func(), err error) {
	noop := func() {}

	// Snapshot existing networks: reuse a block already configured for this SSID
	// and remember the one the supplicant is on, so a failure can restore it.
	reuseID, priorCurrentID, err := s.snapshotWpaNetworks(ctx, ssid)
	if err != nil {
		return "", noop, err
	}

	// enable_network all + reselect the prior network restores the running config
	// after select_network disabled everything but our target.
	restorePrior := func() {
		s.tryWpa(ctx, "enable_network", "all")
		if priorCurrentID != "" {
			s.tryWpa(ctx, "select_network", priorCurrentID)
		}
	}

	added := false
	networkID = reuseID
	if networkID == "" {
		out, addErr := s.wpa(ctx, "add_network")
		if addErr != nil {
			return "", noop, &WiFiError{Message: "wpa_cli add_network failed", Err: addErr}
		}
		networkID = strings.TrimSpace(out)
		added = true
	}

	// rollback removes only a block this call added (never a reused one) and
	// restores the supplicant to the network it was on before this attempt.
	rollback = func() {
		if added {
			s.tryWpa(ctx, "remove_network", networkID)
		}
		restorePrior()
	}

	fail := func(msg string, cause error) (string, func(), error) {
		rollback()
		return "", noop, &WiFiError{Message: msg, Err: cause}
	}

	// A reused block already carries SSID + credentials; only a freshly added block
	// needs configuring.
	if added {
		if e := s.setNewWpaNetwork(ctx, networkID, ssid, password); e != nil {
			return fail(e.Message, e.Err)
		}
	}

	if _, e := s.wpa(ctx, "enable_network", networkID); e != nil {
		return fail("wpa_cli enable_network failed", e)
	}
	if _, e := s.wpa(ctx, "select_network", networkID); e != nil {
		return fail("wpa_cli select_network failed", e)
	}

	return networkID, rollback, nil
}

// snapshotWpaNetworks lists the configured networks and returns the id of an
// existing block for ssid (empty if none, so a fresh block is added) and the id of
// the network the supplicant is currently on (empty if none), used to restore the
// running config if the connect attempt fails.
func (s *platformWiFiScanner) snapshotWpaNetworks(
	ctx context.Context,
	ssid string,
) (reuseID, priorCurrentID string, err error) {
	list, err := s.wpa(ctx, "list_networks")
	if err != nil {
		return "", "", &WiFiError{Message: "wpa_cli list_networks failed", Err: err}
	}
	for _, n := range parseWpaNetworkList(list) {
		if n.current {
			priorCurrentID = n.id
		}
		if n.ssid == ssid {
			reuseID = n.id
		}
	}
	return reuseID, priorCurrentID, nil
}

// setNewWpaNetwork sets the SSID and credentials on a freshly added block: an open
// SSID gets key_mgmt NONE, a protected one gets the quoted PSK. On failure it
// returns a *WiFiError so the caller can roll the block back.
func (s *platformWiFiScanner) setNewWpaNetwork(ctx context.Context, id, ssid, password string) *WiFiError {
	if _, e := s.wpa(ctx, "set_network", id, "ssid", `"`+ssid+`"`); e != nil {
		return &WiFiError{Message: "wpa_cli set ssid failed", Err: e}
	}
	if password == "" {
		if _, e := s.wpa(ctx, "set_network", id, "key_mgmt", "NONE"); e != nil {
			return &WiFiError{Message: "wpa_cli set key_mgmt failed", Err: e}
		}
		return nil
	}
	if _, e := s.wpa(ctx, "set_network", id, "psk", `"`+password+`"`); e != nil {
		return &WiFiError{Message: "wpa_cli set psk failed", Err: e}
	}
	return nil
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

	return s.waitForConnection(ctx, ssid, connectTimeout(ssid))
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

	if hasCommand(methodWpaCli) {
		cmd := exec.CommandContext(ctx, methodWpaCli, "-i", s.iface, "disconnect")
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

	if hasCommand(methodWpaCli) {
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

	return nil, &WiFiError{Message: msgNotConnected}
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
	output, err := s.wpa(ctx, "status")
	if err != nil {
		return nil, &WiFiError{Message: "wpa_cli status failed", Err: err}
	}

	var ssid, state string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "ssid=") {
			ssid = strings.TrimPrefix(line, "ssid=")
		}
		if strings.HasPrefix(line, "wpa_state=") {
			state = strings.TrimPrefix(line, "wpa_state=")
		}
	}

	if state != "COMPLETED" || ssid == "" {
		return nil, &WiFiError{Message: msgNotConnected}
	}

	return &WiFiNetwork{SSID: ssid, LastSeen: time.Now()}, nil
}

// currentNetworkIwconfig gets current network using iwconfig.
//
//nolint:gosec // G204: Interface name is auto-detected, not user input
func (s *platformWiFiScanner) currentNetworkIwconfig(ctx context.Context) (*WiFiNetwork, error) {
	cmd := exec.CommandContext(ctx, "iwconfig", s.iface) //nolint:gosec // G204: iface auto-detected
	output, err := cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "iwconfig failed", Err: err}
	}
	return parseIwconfigOutput(string(output))
}

// parseIwconfigOutput extracts the connected SSID from iwconfig output.
func parseIwconfigOutput(outputStr string) (*WiFiNetwork, error) {
	essidIdx := strings.Index(outputStr, `ESSID:"`)
	if essidIdx == -1 {
		return nil, &WiFiError{Message: msgNotConnected}
	}

	start := essidIdx + 7
	end := strings.Index(outputStr[start:], `"`)
	if end == -1 {
		return nil, &WiFiError{Message: "failed to parse ESSID"}
	}

	ssid := outputStr[start : start+end]
	if ssid == "" || ssid == "off/any" {
		return nil, &WiFiError{Message: msgNotConnected}
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
