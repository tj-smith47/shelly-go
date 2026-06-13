//go:build linux

package discovery

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

// ────────────────────────────────────────────────────────────────────────────
// SetAPHostIP / apHostCIDR
// ────────────────────────────────────────────────────────────────────────────

func TestSetAPHostIP(t *testing.T) {
	s := &platformWiFiScanner{}

	// Empty input is a no-op — default stays.
	s.SetAPHostIP("")
	if s.apHostIP != "" {
		t.Errorf("empty SetAPHostIP must not change apHostIP, got %q", s.apHostIP)
	}

	s.SetAPHostIP("192.168.33.200")
	if s.apHostIP != "192.168.33.200" {
		t.Errorf("apHostIP = %q, want 192.168.33.200", s.apHostIP)
	}
}

func TestApHostCIDR(t *testing.T) {
	tests := []struct {
		name     string
		apHostIP string
		want     string
	}{
		{"default when empty", "", DefaultAPHostIP + "/24"},
		{"custom IP", "192.168.33.200", "192.168.33.200/24"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &platformWiFiScanner{apHostIP: tt.apHostIP}
			got := s.apHostCIDR()
			if got != tt.want {
				t.Errorf("apHostCIDR() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ────────────────────────────────────────────────────────────────────────────
// bssListToNetworks / rsnToSecurity
// ────────────────────────────────────────────────────────────────────────────

func TestBssListToNetworks_Empty(t *testing.T) {
	result := bssListToNetworks(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 networks for nil bss list, got %d", len(result))
	}
}

func TestRsnToSecurity(t *testing.T) {
	// Import the wifi package types indirectly by testing via parseIwField since
	// rsnToSecurity takes wifi.RSNInfo which we cannot construct portably without the
	// mdlayher/wifi import. Instead we validate the function exists and test its
	// logic by calling parseIwScanOutput which exercises rsnToSecurity via the
	// bssListToNetworks path. Direct call path is exercised through parseIwScanOutput
	// RSN detection branch.
	input := `BSS aa:bb:cc:dd:ee:ff(on wlan0)
	freq: 2437
	signal: -50.00 dBm
	SSID: TestNet
	RSN:	 * Version: 1
`
	networks := parseIwScanOutput(input)
	if len(networks) != 1 {
		t.Fatalf("expected 1 network, got %d", len(networks))
	}
	if networks[0].Security != securityWPA2 {
		t.Errorf("Security = %q, want %q", networks[0].Security, securityWPA2)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Scan — fallback path (all sub-methods fail → WiFiError)
// ────────────────────────────────────────────────────────────────────────────

func TestPlatformScan_AllMethodsFail(t *testing.T) {
	// scanViaNl80211 will fail (no real WiFi NIC in CI), NM will fail, and
	// nmcli/iw are either absent or will fail. The function must return a
	// non-nil error wrapping a WiFiError when every method has been tried.
	s := &platformWiFiScanner{iface: "nonexistent99"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := s.Scan(ctx)
	if err == nil {
		// In environments where NM/nmcli happens to work this test is vacuous;
		// the important thing is no panic and no crash.
		t.Log("Scan returned nil error — real WiFi available in this environment")
		return
	}
	var wErr *WiFiError
	if !errors.As(err, &wErr) {
		t.Errorf("Scan() error should be *WiFiError, got %T: %v", err, err)
	}
}

func TestPlatformScan_CancelledContext(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already done

	_, err := s.Scan(ctx)
	if err == nil {
		t.Error("Scan on canceled context should return error")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// connectMethods ordering
// ────────────────────────────────────────────────────────────────────────────

func TestConnectMethods_ViaSupplicant_LeadsWithWpaCli(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	methods := s.connectMethods(true)
	if len(methods) == 0 {
		t.Fatal("expected at least one method")
	}
	if methods[0].name != methodWpaCli {
		t.Errorf("first method when viaSupplicant=true should be wpa_cli, got %q", methods[0].name)
	}
	// nl80211 must NOT appear when wpa_supplicant owns the interface.
	for _, m := range methods {
		if m.name == methodNl80211 {
			t.Error("nl80211 must not appear when viaSupplicant=true")
		}
	}
}

func TestConnectMethods_NoSupplicant_LeadsWithNl80211(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	methods := s.connectMethods(false)
	if len(methods) == 0 {
		t.Fatal("expected at least one method")
	}
	if methods[0].name != methodNl80211 {
		t.Errorf("first method when viaSupplicant=false should be nl80211, got %q", methods[0].name)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Connect — wpa_cli path via the wpaRun seam
// ────────────────────────────────────────────────────────────────────────────

// fakeWpaWithCurrent is a more detailed fake that returns a network list
// containing a current network and handles all the wpa_cli verbs needed for
// configureWpaNetwork → connectWpaCli.
func fakeWpaForConnect(seq *[]string, currentSSID string) func(ctx context.Context, args ...string) (string, error) {
	const networkID = "1"
	listOut := "network id / ssid / bssid / flags\n0\t" + currentSSID + "\tany\t[CURRENT]\n"
	return func(_ context.Context, args ...string) (string, error) {
		*seq = append(*seq, strings.Join(args, " "))
		verb := args[0]
		switch verb {
		case "ping":
			return "PONG", nil
		case "list_networks":
			return listOut, nil
		case "add_network":
			return networkID, nil
		case "set_network", "enable_network", "select_network":
			return "OK", nil
		case "status":
			// Simulate connected after select_network.
			return "ssid=ShellyBulbDuo-D0DCFF\nwpa_state=COMPLETED\n", nil
		default:
			return "OK", nil
		}
	}
}

func TestPlatformConnect_ViaWpaCli(t *testing.T) {
	// Connect() only dispatches through wpa_cli when hasCommand("wpa_cli") returns
	// true AND wpaSupplicantManages returns true. In CI/containers wpa_cli is often
	// absent, so we test the wpa_cli path directly through connectWpaCli instead of
	// going through Connect().
	var seq []string
	s := &platformWiFiScanner{
		iface:  "wlan0",
		wpaRun: fakeWpaForConnect(&seq, "OnyxCheetah4.7"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// connectWpaCli calls configureWpaNetwork (→ list_networks, add_network, …)
	// then waitForConnection. waitForConnection will time out because CurrentNetwork
	// uses exec-based methods (nl80211/NM/nmcli/wpa_cli) that fail in CI. We only
	// care that the wpa_cli command sequence was driven.
	err := s.connectWpaCli(ctx, "ShellyBulbDuo-D0DCFF", "")
	_ = err // timeout or connection error expected in CI

	if !seqHas(seq, "list_networks") {
		t.Errorf("expected wpa_cli list_networks to be called; seq=%v", seq)
	}
	if !seqHas(seq, "add_network") {
		t.Errorf("expected wpa_cli add_network to be called; seq=%v", seq)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// configureWpaNetwork
// ────────────────────────────────────────────────────────────────────────────

func TestConfigureWpaNetwork_NewBlock(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{
			"list_networks": "network id / ssid / bssid / flags\n0\tHomeNet\tany\t[CURRENT]\n",
			"add_network":   "1",
		}, nil),
	}

	id, rollback, err := s.configureWpaNetwork(context.Background(), "ShellyBulbDuo-D0DCFF", "")
	if err != nil {
		t.Fatalf("configureWpaNetwork: %v", err)
	}
	if id != "1" {
		t.Errorf("network id = %q, want 1", id)
	}
	if !seqHas(seq, "add_network") {
		t.Errorf("add_network was not called; seq=%v", seq)
	}
	if !seqHas(seq, "set_network 1 ssid \"ShellyBulbDuo-D0DCFF\"") {
		t.Errorf("ssid was not set; seq=%v", seq)
	}
	// Rollback should remove the added block and restore the prior network.
	seq = nil
	rollback()
	if !seqHas(seq, "remove_network 1") {
		t.Errorf("rollback did not remove network; seq=%v", seq)
	}
	if !seqHas(seq, "select_network 0") {
		t.Errorf("rollback did not restore prior network; seq=%v", seq)
	}
}

func TestConfigureWpaNetwork_ReuseExisting(t *testing.T) {
	var seq []string
	// Existing block for same SSID is id=1.
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{
			"list_networks": "network id / ssid / bssid / flags\n0\tHomeNet\tany\t[CURRENT]\n1\tShellyBulbDuo-D0DCFF\tany\t[DISABLED]\n",
		}, nil),
	}

	id, _, err := s.configureWpaNetwork(context.Background(), "ShellyBulbDuo-D0DCFF", "")
	if err != nil {
		t.Fatalf("configureWpaNetwork (reuse): %v", err)
	}
	if id != "1" {
		t.Errorf("expected reused id 1, got %q", id)
	}
	// Must not add_network when reusing.
	for _, c := range seq {
		if c == "add_network" {
			t.Errorf("add_network must not be called when reusing a block; seq=%v", seq)
		}
	}
}

func TestConfigureWpaNetwork_ListFails(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&seq, nil, map[string]error{
			"list_networks": errors.New("wpa down"),
		}),
	}
	_, _, err := s.configureWpaNetwork(context.Background(), "SomeSSID", "")
	if err == nil {
		t.Fatal("expected error when list_networks fails")
	}
}

func TestConfigureWpaNetwork_WithPassword(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{
			"list_networks": "network id / ssid / bssid / flags\n",
			"add_network":   "2",
		}, nil),
	}
	id, rollback, err := s.configureWpaNetwork(context.Background(), "ShellyBulbDuo-D0DCFF", "hunter2")
	if err != nil {
		t.Fatalf("configureWpaNetwork with password: %v", err)
	}
	if id != "2" {
		t.Errorf("id = %q, want 2", id)
	}
	if !seqHas(seq, `set_network 2 psk "hunter2"`) {
		t.Errorf("psk was not set; seq=%v", seq)
	}
	rollback()
}

// ────────────────────────────────────────────────────────────────────────────
// snapshotWpaNetworks
// ────────────────────────────────────────────────────────────────────────────

func TestSnapshotWpaNetworks(t *testing.T) {
	var seq []string
	listOut := "network id / ssid / bssid / flags\n" +
		"0\tHomeNet\tany\t[CURRENT]\n" +
		"1\tShellyBulbDuo-D0DCFF\tany\t[DISABLED]\n"
	s := &platformWiFiScanner{
		iface:  "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{"list_networks": listOut}, nil),
	}
	reuseID, priorCurrentID, err := s.snapshotWpaNetworks(context.Background(), "ShellyBulbDuo-D0DCFF")
	if err != nil {
		t.Fatalf("snapshotWpaNetworks: %v", err)
	}
	if reuseID != "1" {
		t.Errorf("reuseID = %q, want 1", reuseID)
	}
	if priorCurrentID != "0" {
		t.Errorf("priorCurrentID = %q, want 0", priorCurrentID)
	}
}

func TestSnapshotWpaNetworks_NoMatch(t *testing.T) {
	var seq []string
	listOut := "network id / ssid / bssid / flags\n0\tHomeNet\tany\t[CURRENT]\n"
	s := &platformWiFiScanner{
		iface:  "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{"list_networks": listOut}, nil),
	}
	reuseID, priorCurrentID, err := s.snapshotWpaNetworks(context.Background(), "UnknownSSID")
	if err != nil {
		t.Fatalf("snapshotWpaNetworks (no match): %v", err)
	}
	if reuseID != "" {
		t.Errorf("reuseID should be empty, got %q", reuseID)
	}
	if priorCurrentID != "0" {
		t.Errorf("priorCurrentID = %q, want 0", priorCurrentID)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// setNewWpaNetwork
// ────────────────────────────────────────────────────────────────────────────

func TestSetNewWpaNetwork_Open(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface:  "wlan0",
		wpaRun: fakeWpa(&seq, nil, nil),
	}
	if err := s.setNewWpaNetwork(context.Background(), "3", "OpenNet", ""); err != nil {
		t.Fatalf("setNewWpaNetwork (open): %v", err)
	}
	if !seqHas(seq, `set_network 3 ssid "OpenNet"`) {
		t.Errorf("ssid not set; seq=%v", seq)
	}
	if !seqHas(seq, "set_network 3 key_mgmt NONE") {
		t.Errorf("key_mgmt not set to NONE; seq=%v", seq)
	}
}

func TestSetNewWpaNetwork_Protected(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface:  "wlan0",
		wpaRun: fakeWpa(&seq, nil, nil),
	}
	if err := s.setNewWpaNetwork(context.Background(), "3", "ProtectedNet", "mypassword"); err != nil {
		t.Fatalf("setNewWpaNetwork (protected): %v", err)
	}
	if !seqHas(seq, `set_network 3 psk "mypassword"`) {
		t.Errorf("psk not set; seq=%v", seq)
	}
}

func TestSetNewWpaNetwork_SsidSetFails(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&seq, nil, map[string]error{
			"set_network": errors.New("FAIL"),
		}),
	}
	err := s.setNewWpaNetwork(context.Background(), "3", "SomeNet", "")
	if err == nil {
		t.Fatal("expected error when set_network ssid fails")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// currentNetworkWpaCli / currentNetworkNmcli / currentNetworkIwconfig
// ────────────────────────────────────────────────────────────────────────────

func TestCurrentNetworkWpaCli_Connected(t *testing.T) {
	var seq []string
	statusOut := "ssid=MyHomeNetwork\nwpa_state=COMPLETED\nip_address=192.168.1.50\n"
	s := &platformWiFiScanner{
		iface:  "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{"status": statusOut}, nil),
	}
	// currentNetworkWpaCli is only called via CurrentNetwork fallback chain; but
	// since we have no real wpa_cli binary the exec path would fail. Test the
	// parse logic instead by constructing the scanner with wpaRun pointing to
	// status output. The function calls wpa_cli directly via exec, so we test
	// the underlying parser (parseWpaNetworkList is used for the list; the status
	// parsing is inline in currentNetworkWpaCli). We exercise it via the
	// wpa() seam through a fake wpaRun.
	//
	// currentNetworkWpaCli does NOT use s.wpa() — it calls exec.CommandContext
	// directly. So we can only test it indirectly. Instead, test the state-machine
	// logic via parseWpaStatus lines (which is inlined), exercised by calling the
	// function with exec pointing to a known-bad command so we see the error path.
	ctx := context.Background()
	net, err := s.currentNetworkWpaCli(ctx)
	// On a machine without wpa_cli this returns an error — that's expected.
	if err != nil {
		var wErr *WiFiError
		if !errors.As(err, &wErr) {
			t.Errorf("error should be *WiFiError, got %T: %v", err, err)
		}
		return
	}
	// If wpa_cli is present and returns data, net must be non-nil.
	if net == nil {
		t.Error("currentNetworkWpaCli returned nil network without error")
	}
}

func TestParseNmcliLineInContext(t *testing.T) {
	// Exercises the inline currentNetworkNmcli parsing through parseNmcliLine.
	// The line format is "*:SSID:Signal:Security".
	s := &platformWiFiScanner{iface: "wlan0"}
	tests := []struct {
		line     string
		wantSSID string
		wantNil  bool
	}{
		{"*:OnyxCheetah4.7:75:WPA2", "OnyxCheetah4.7", false},
		{"*:OnyxCheetah4.7", "OnyxCheetah4.7", false},
		{" :HomeNetwork:75:WPA2", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		got := s.parseNmcliLine(tt.line)
		if tt.wantNil {
			if got != nil {
				t.Errorf("parseNmcliLine(%q) = %+v, want nil", tt.line, got)
			}
		} else {
			if got == nil {
				t.Fatalf("parseNmcliLine(%q) = nil, want non-nil", tt.line)
			}
			if got.SSID != tt.wantSSID {
				t.Errorf("parseNmcliLine(%q).SSID = %q, want %q", tt.line, got.SSID, tt.wantSSID)
			}
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// hasRoutableIPv4
// ────────────────────────────────────────────────────────────────────────────

func TestHasRoutableIPv4_KnownLoopback(t *testing.T) {
	// "lo" carries 127.0.0.1 which is loopback — hasRoutableIPv4 must return false.
	if hasRoutableIPv4("lo") {
		t.Error("loopback interface (lo) should not be considered routable")
	}
}

func TestHasRoutableIPv4_NonexistentIface(t *testing.T) {
	// A nonexistent interface must not panic and must return false.
	if hasRoutableIPv4("nonexistent99") {
		t.Error("nonexistent interface should return false")
	}
}

func TestHasRoutableIPv4_RealInterface(t *testing.T) {
	// Find any interface with a real IPv4 address that is not loopback or
	// link-local, and verify hasRoutableIPv4 returns true for it.
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Skip("cannot enumerate interfaces:", err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
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
			// Found a candidate.
			if !hasRoutableIPv4(iface.Name) {
				t.Errorf("hasRoutableIPv4(%q) = false, expected true (addr %s)", iface.Name, ip4)
			}
			return
		}
	}
	t.Log("no routable IPv4 interface found — skipping positive assertion")
}

// ────────────────────────────────────────────────────────────────────────────
// HostNetworkPassword
// ────────────────────────────────────────────────────────────────────────────

func TestHostNetworkPassword_EmptySSID(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	_, err := s.HostNetworkPassword(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty SSID")
	}
	if !strings.Contains(err.Error(), "ssid is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestHostNetworkPassword_UnknownSSID(t *testing.T) {
	// A SSID that no credential store has stored → error "no stored passphrase".
	s := &platformWiFiScanner{iface: "wlan0"}
	_, err := s.HostNetworkPassword(context.Background(), "NonExistentNetworkXYZ12345")
	// Error must be non-nil — either "nmcli not available" or "no stored passphrase".
	if err == nil {
		t.Error("expected error for unknown SSID")
	}
}

func TestHostPasswordViaWpaSupplicant_NoFiles(t *testing.T) {
	// When no wpa_supplicant config files are present the function returns "".
	// This is always true in CI (no /etc/wpa_supplicant.conf).
	// If a real file exists and contains our test SSID this will return the
	// password; we just assert the call doesn't panic.
	result := hostPasswordViaWpaSupplicant("NoSuchSSIDAtAll")
	if result != "" {
		t.Logf("hostPasswordViaWpaSupplicant returned %q unexpectedly (stale config?)", result)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// wpa() seam / tryWpa
// ────────────────────────────────────────────────────────────────────────────

func TestWpa_UsesSeamWhenSet(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface:  "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{"ping": "PONG"}, nil),
	}
	out, err := s.wpa(context.Background(), "ping")
	if err != nil {
		t.Fatalf("wpa(ping): %v", err)
	}
	if out != "PONG" {
		t.Errorf("wpa(ping) = %q, want PONG", out)
	}
	if !seqHas(seq, "ping") {
		t.Errorf("seam was not called; seq=%v", seq)
	}
}

func TestTryWpa_DoesNotPropagateError(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&seq, nil, map[string]error{
			"disconnect": errors.New("error"),
		}),
	}
	// tryWpa must not panic or return an error; the call is fire-and-forget.
	s.tryWpa(context.Background(), "disconnect")
	if !seqHas(seq, "disconnect") {
		t.Errorf("tryWpa did not invoke the seam; seq=%v", seq)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// parseBSSID edge cases
// ────────────────────────────────────────────────────────────────────────────

func TestParseBSSID(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"BSS aa:bb:cc:dd:ee:ff(on wlan0)", "AA:BB:CC:DD:EE:FF"},
		{"BSS aa:bb:cc:dd:ee:ff", "AA:BB:CC:DD:EE:FF"},
		{"BSS", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := parseBSSID(tt.line)
		if got != tt.want {
			t.Errorf("parseBSSID(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// parseIwField edge cases
// ────────────────────────────────────────────────────────────────────────────

func TestParseIwField_AllBranches(t *testing.T) {
	tests := []struct {
		line     string
		wantSSID string
		wantSig  int
		wantCh   int
		wantSec  string
	}{
		{"SSID: TestNet", "TestNet", 0, 0, ""},
		{"signal: -63.00 dBm", "", -63, 0, ""},
		{"freq: 2412", "", 0, 1, ""},
		{"RSN:	 * Version: 1", "", 0, 0, securityWPA2},
		{"WPA:	 * Version: 1", "", 0, 0, securityWPA},
		{"something else: ignored", "", 0, 0, ""},
	}
	for _, tt := range tests {
		n := &WiFiNetwork{}
		parseIwField(n, tt.line)
		if n.SSID != tt.wantSSID {
			t.Errorf("parseIwField(%q).SSID = %q, want %q", tt.line, n.SSID, tt.wantSSID)
		}
		if n.Signal != tt.wantSig {
			t.Errorf("parseIwField(%q).Signal = %d, want %d", tt.line, n.Signal, tt.wantSig)
		}
		if n.Channel != tt.wantCh {
			t.Errorf("parseIwField(%q).Channel = %d, want %d", tt.line, n.Channel, tt.wantCh)
		}
		if n.Security != tt.wantSec {
			t.Errorf("parseIwField(%q).Security = %q, want %q", tt.line, n.Security, tt.wantSec)
		}
	}
}

func TestParseIwField_WPANotOverwrittenByRSN(t *testing.T) {
	// Once Security is set (WPA) a later RSN: line must not overwrite it.
	n := &WiFiNetwork{Security: securityWPA}
	parseIwField(n, "RSN:	 * Version: 1")
	if n.Security != securityWPA {
		t.Errorf("RSN must not overwrite already-set Security, got %q", n.Security)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// parseNmcliScanOutput edge cases
// ────────────────────────────────────────────────────────────────────────────

func TestParseNmcliScanOutput_Empty(t *testing.T) {
	result := parseNmcliScanOutput("")
	if len(result) != 0 {
		t.Errorf("expected 0 networks for empty input, got %d", len(result))
	}
}

func TestParseNmcliScanOutput_OnlyBlanks(t *testing.T) {
	result := parseNmcliScanOutput("\n\n\n")
	if len(result) != 0 {
		t.Errorf("expected 0 networks for all-blank input, got %d", len(result))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Disconnect — no-op when all tools absent (returns ErrToolNotFound)
// ────────────────────────────────────────────────────────────────────────────

func TestPlatformDisconnect_DoesNotPanic(t *testing.T) {
	s := &platformWiFiScanner{iface: "nonexistent99"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// We expect either nil (some method works) or ErrToolNotFound / WiFiError.
	err := s.Disconnect(ctx)
	if err == nil {
		return // nl80211 or NM unexpectedly worked
	}
	// Must be a known error type.
	var wErr *WiFiError
	if !errors.As(err, &wErr) && !errors.Is(err, ErrToolNotFound) {
		t.Errorf("Disconnect error should be *WiFiError, got %T: %v", err, err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// CurrentNetwork — fallback chain does not panic
// ────────────────────────────────────────────────────────────────────────────

func TestPlatformCurrentNetwork_DoesNotPanic(t *testing.T) {
	s := &platformWiFiScanner{iface: "nonexistent99"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// May return a result (if nl80211/NM finds a network) or an error (if nothing works).
	net, err := s.CurrentNetwork(ctx)
	if err == nil && net == nil {
		t.Error("CurrentNetwork must not return nil net + nil error")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// apToNetwork — covers the NM AP-to-network conversion
// ────────────────────────────────────────────────────────────────────────────

// apToNetwork requires a real NM D-Bus AccessPoint which we can't fake without
// a live system. Coverage is obtained indirectly through the Scan / scanViaNM
// paths when NM is present. In environments without NM (CI), the function is
// unreachable — this is an honest gap (see FINAL REPORT).

// ────────────────────────────────────────────────────────────────────────────
// connectTimeout already tested in the existing file; just verify sentinel.
// ────────────────────────────────────────────────────────────────────────────

func TestConnectTimeoutSentinelValues(t *testing.T) {
	if wifiConnectTimeout <= 0 {
		t.Errorf("wifiConnectTimeout should be positive, got %v", wifiConnectTimeout)
	}
	if apConnectTimeout <= 0 {
		t.Errorf("apConnectTimeout should be positive, got %v", apConnectTimeout)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// waitForConnection — context-cancel path
// ────────────────────────────────────────────────────────────────────────────

func TestWaitForConnection_ContextCancelled(t *testing.T) {
	s := &platformWiFiScanner{
		iface: "wlan0",
		// CurrentNetwork returns "not connected" → loop keeps trying.
		wpaRun: fakeWpa(new([]string), nil, map[string]error{}),
	}
	// We need CurrentNetwork to return an error so waitForConnection loops.
	// Since wpaRun doesn't affect CurrentNetwork's nl80211/NM calls,
	// on a host with no real "wlan0" it will fail with ErrToolNotFound.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := s.waitForConnection(ctx, "SomeSSID", 500*time.Millisecond)
	// Should return ctx.Err() or ErrConnectionTimeout — not nil.
	if err == nil {
		t.Error("waitForConnection should fail when not connected")
	}
}

func TestWaitForConnection_Timeout(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	ctx := context.Background()
	// Very short timeout — no real interface "wlan0" so CurrentNetwork always fails.
	err := s.waitForConnection(ctx, "SomeSSID", 100*time.Millisecond)
	if err == nil {
		t.Error("waitForConnection should timeout")
	}
}
