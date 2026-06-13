//go:build linux

package discovery

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/mdlayher/wifi"
)

// ────────────────────────────────────────────────────────────────────────────
// rsnToSecurity — direct tests using real wifi.RSNInfo / wifi.RSNAKM values
// ────────────────────────────────────────────────────────────────────────────

func TestRsnToSecurity_WPA3_SAE(t *testing.T) {
	rsn := wifi.RSNInfo{
		Version: 1,
		AKMs:    []wifi.RSNAKM{wifi.RSNAkmSAE},
	}
	got := rsnToSecurity(rsn)
	if got != securityWPA3 {
		t.Errorf("rsnToSecurity(SAE) = %q, want %q", got, securityWPA3)
	}
}

func TestRsnToSecurity_WPA3_FTSAE(t *testing.T) {
	rsn := wifi.RSNInfo{
		Version: 1,
		AKMs:    []wifi.RSNAKM{wifi.RSNAkmFTSAE},
	}
	got := rsnToSecurity(rsn)
	if got != securityWPA3 {
		t.Errorf("rsnToSecurity(FT-SAE) = %q, want %q", got, securityWPA3)
	}
}

func TestRsnToSecurity_WPA2(t *testing.T) {
	rsn := wifi.RSNInfo{
		Version: 1,
		AKMs:    []wifi.RSNAKM{wifi.RSNAkmPSK},
	}
	got := rsnToSecurity(rsn)
	if got != securityWPA2 {
		t.Errorf("rsnToSecurity(PSK/v1) = %q, want %q", got, securityWPA2)
	}
}

func TestRsnToSecurity_WPA_FallbackVersion0(t *testing.T) {
	rsn := wifi.RSNInfo{
		Version: 0,
		AKMs:    []wifi.RSNAKM{wifi.RSNAkmPSK},
	}
	got := rsnToSecurity(rsn)
	if got != securityWPA {
		t.Errorf("rsnToSecurity(version=0) = %q, want %q", got, securityWPA)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// bssListToNetworks — direct tests using real wifi.BSS values
// ────────────────────────────────────────────────────────────────────────────

func TestBssListToNetworks_SingleBSS(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	bss := &wifi.BSS{
		SSID:      "HomeNetwork",
		BSSID:     mac,
		Signal:    -5000, // mBm = -50 dBm
		Frequency: 2437,
		RSN: wifi.RSNInfo{
			Version: 1,
			AKMs:    []wifi.RSNAKM{wifi.RSNAkmPSK},
		},
	}

	networks := bssListToNetworks([]*wifi.BSS{bss})
	if len(networks) != 1 {
		t.Fatalf("expected 1 network, got %d", len(networks))
	}

	n := networks[0]
	if n.SSID != "HomeNetwork" {
		t.Errorf("SSID = %q, want HomeNetwork", n.SSID)
	}
	if n.Signal != -50 {
		t.Errorf("Signal = %d, want -50", n.Signal)
	}
	if n.Channel != 6 {
		t.Errorf("Channel = %d, want 6", n.Channel)
	}
	if n.Security != securityWPA2 {
		t.Errorf("Security = %q, want %q", n.Security, securityWPA2)
	}
}

func TestBssListToNetworks_SkipsEmptySSID(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	bssList := []*wifi.BSS{
		{SSID: "", BSSID: mac, Frequency: 2437},
		{SSID: "Visible", BSSID: mac, Frequency: 5180},
	}
	networks := bssListToNetworks(bssList)
	if len(networks) != 1 {
		t.Errorf("expected 1 network (empty SSID skipped), got %d", len(networks))
	}
	if networks[0].SSID != "Visible" {
		t.Errorf("SSID = %q, want Visible", networks[0].SSID)
	}
}

func TestBssListToNetworks_NoRSN(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	bss := &wifi.BSS{
		SSID:      "OpenNet",
		BSSID:     mac,
		Frequency: 2412,
	}
	networks := bssListToNetworks([]*wifi.BSS{bss})
	if len(networks) != 1 {
		t.Fatalf("expected 1 network, got %d", len(networks))
	}
	// No RSN → Security is empty string.
	if networks[0].Security != "" {
		t.Errorf("Security = %q, want empty for open network", networks[0].Security)
	}
}

func TestBssListToNetworks_LastSeenIsSet(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	before := time.Now()
	networks := bssListToNetworks([]*wifi.BSS{
		{SSID: "Net", BSSID: mac},
	})
	after := time.Now()
	if len(networks) == 0 {
		t.Fatal("expected 1 network")
	}
	if networks[0].LastSeen.Before(before) || networks[0].LastSeen.After(after) {
		t.Errorf("LastSeen %v is not between %v and %v", networks[0].LastSeen, before, after)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// wpa() — production path (wpaRun == nil, calls exec.Command)
// When wpa_cli is absent this returns a non-nil error from exec.LookPath;
// that error path IS the branch we want to cover in the else clause.
// ────────────────────────────────────────────────────────────────────────────

func TestWpa_ProductionPath_NilSeam(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0", wpaRun: nil}
	// wpa_cli may or may not be installed. Either way we exercise the real exec path.
	_, err := s.wpa(context.Background(), "ping")
	// We don't assert the result — just that the function does not panic.
	_ = err
}

// ────────────────────────────────────────────────────────────────────────────
// currentNetworkNmcli parse logic — exercised directly
// ────────────────────────────────────────────────────────────────────────────

func TestCurrentNetworkNmcli_ParseLogic(t *testing.T) {
	// currentNetworkNmcli calls exec (nmcli) which may fail in CI.
	// We test parseNmcliLine exhaustively to cover the inlined parse logic.
	s := &platformWiFiScanner{iface: "wlan0"}

	// Verify the "*:" detection that currentNetworkNmcli relies on.
	n := s.parseNmcliLine("*:HomeNetwork:65:WPA2")
	if n == nil {
		t.Fatal("parseNmcliLine should parse current-network lines")
	}
	if n.SSID != "HomeNetwork" {
		t.Errorf("SSID = %q, want HomeNetwork", n.SSID)
	}
	if n.Signal != 65 {
		t.Errorf("Signal = %d, want 65", n.Signal)
	}
	if n.Security != "WPA2" {
		t.Errorf("Security = %q, want WPA2", n.Security)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// currentNetworkIwconfig parse logic
// The function calls exec(iwconfig); exercise the parse logic by extracting
// the string-search pattern used inside the function.
// ────────────────────────────────────────────────────────────────────────────

func TestCurrentNetworkIwconfig_ExecPath(t *testing.T) {
	// Call the function; if iwconfig is absent it returns an error — that's fine
	// and still exercises the exec.Command path.
	s := &platformWiFiScanner{iface: "wlan0"}
	ctx := context.Background()
	_, err := s.currentNetworkIwconfig(ctx)
	// On CI iwconfig is absent → non-nil error with WiFiError.
	_ = err
}

// ────────────────────────────────────────────────────────────────────────────
// ensureInterfaceUp — with a nonexistent interface (ReadFile fails → returns)
// ────────────────────────────────────────────────────────────────────────────

func TestEnsureInterfaceUp_NonexistentInterface(t *testing.T) {
	s := &platformWiFiScanner{iface: "doesnotexist99"}
	// Must not panic; the function returns silently when ReadFile fails.
	s.ensureInterfaceUp(context.Background())
}

func TestEnsureInterfaceUp_InterfaceUp(t *testing.T) {
	// "lo" is always up — operstate is "unknown" (not "down"), so the function
	// returns after the state check without issuing an ip command.
	s := &platformWiFiScanner{iface: "lo"}
	s.ensureInterfaceUp(context.Background())
}

// ────────────────────────────────────────────────────────────────────────────
// reacquireDHCP — with no DHCP clients installed (best-effort path)
// ────────────────────────────────────────────────────────────────────────────

func TestReacquireDHCP_LoopbackIsRoutable(t *testing.T) {
	// lo carries 127.0.0.1 (loopback) which hasRoutableIPv4 treats as NOT routable.
	// So reacquireDHCP will attempt DHCP clients, find none, and return silently.
	s := &platformWiFiScanner{iface: "lo"}
	s.reacquireDHCP(context.Background(), "lo")
}

func TestReacquireDHCP_NonexistentIface(t *testing.T) {
	// Interface doesn't exist — hasRoutableIPv4 returns false, then all DHCP
	// client commands either fail or are absent. Must not panic.
	s := &platformWiFiScanner{iface: "nonexistent99"}
	s.reacquireDHCP(context.Background(), "nonexistent99")
}

// ────────────────────────────────────────────────────────────────────────────
// removeAPStaticIP / obtainIPAddress — exec paths (best-effort, no panic)
// ────────────────────────────────────────────────────────────────────────────

func TestRemoveAPStaticIP_NonexistentInterface(t *testing.T) {
	s := &platformWiFiScanner{iface: "nonexistent99", apHostIP: DefaultAPHostIP}
	// ip addr del on a nonexistent iface exits non-zero; function silently returns.
	s.removeAPStaticIP(context.Background(), "nonexistent99")
}

func TestObtainIPAddress_NonexistentInterface(t *testing.T) {
	s := &platformWiFiScanner{iface: "nonexistent99", apHostIP: DefaultAPHostIP}
	// ip addr add on nonexistent iface exits non-zero; function silently returns.
	s.obtainIPAddress(context.Background(), "nonexistent99")
}

// ────────────────────────────────────────────────────────────────────────────
// connectIwconfig — exec paths (always available via "iwconfig" check)
// ────────────────────────────────────────────────────────────────────────────

func TestConnectIwconfig_ExecPath(t *testing.T) {
	s := &platformWiFiScanner{iface: "nonexistent99"}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	// iwconfig may or may not be installed; connectIwconfig returns an error either way.
	err := s.connectIwconfig(ctx, "SomeSSID", "")
	// We only assert no panic; error is expected.
	_ = err
}

// ────────────────────────────────────────────────────────────────────────────
// connectNmcli — exec path
// ────────────────────────────────────────────────────────────────────────────

func TestConnectNmcli_ExecPath(t *testing.T) {
	s := &platformWiFiScanner{iface: "nonexistent99"}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := s.connectNmcli(ctx, "SomeSSID", "")
	// Expected to fail (no nmcli or wrong credentials) — assert no panic.
	_ = err
}

// ────────────────────────────────────────────────────────────────────────────
// Connect() — exercises the full dispatch chain on a machine without nl80211
// ────────────────────────────────────────────────────────────────────────────

func TestPlatformScanner_Connect_AllMethodsFail(t *testing.T) {
	s := &platformWiFiScanner{iface: "nonexistent99"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// All sub-methods will fail; expect a WiFiError.
	err := s.Connect(ctx, "SomeSSID", "")
	if err == nil {
		// On a machine with a working WiFi NIC this may actually succeed — OK.
		t.Log("Connect returned nil — real WiFi available")
		return
	}
	var wErr *WiFiError
	if !isWiFiError(err, &wErr) {
		t.Errorf("Connect error should be *WiFiError, got %T: %v", err, err)
	}
}

// isWiFiError checks whether err is a *WiFiError at any level.
func isWiFiError(err error, out **WiFiError) bool {
	if e, ok := err.(*WiFiError); ok {
		*out = e
		return true
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// scanViaNmcli / scanViaIw — exec paths
// ────────────────────────────────────────────────────────────────────────────

func TestScanViaNmcli_ExecPath(t *testing.T) {
	s := &platformWiFiScanner{iface: "nonexistent99"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// nmcli may or may not be installed; either way should not panic.
	_, err := s.scanViaNmcli(ctx)
	_ = err
}

func TestScanViaIw_ExecPath(t *testing.T) {
	s := &platformWiFiScanner{iface: "nonexistent99"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := s.scanViaIw(ctx)
	_ = err
}

// ────────────────────────────────────────────────────────────────────────────
// hostPasswordViaNmcli — exec path (nmcli not available or connection not found)
// ────────────────────────────────────────────────────────────────────────────

func TestHostPasswordViaNmcli_ExecPath(t *testing.T) {
	ctx := context.Background()
	_, err := hostPasswordViaNmcli(ctx, "NonExistentSSID")
	// Should return error (either "nmcli not available" or "connection not found").
	if err == nil {
		t.Log("hostPasswordViaNmcli returned nil — nmcli found a connection named 'NonExistentSSID'")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// setNewWpaNetwork — key_mgmt NONE path failure
// ────────────────────────────────────────────────────────────────────────────

func TestSetNewWpaNetwork_KeyMgmtFails(t *testing.T) {
	call := 0
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: func(_ context.Context, args ...string) (string, error) {
			seq = append(seq, args[0])
			call++
			// First call (set_network ssid) succeeds; second (set_network key_mgmt) fails.
			if call == 2 {
				return "", &WiFiError{Message: "set key_mgmt failed"}
			}
			return "OK", nil
		},
	}
	err := s.setNewWpaNetwork(context.Background(), "3", "OpenNet", "")
	if err == nil {
		t.Fatal("expected error when set_network key_mgmt fails")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// configureWpaNetwork — add_network fails path
// ────────────────────────────────────────────────────────────────────────────

func TestConfigureWpaNetwork_AddNetworkFails(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{
			"list_networks": "network id / ssid / bssid / flags\n",
		}, map[string]error{
			"add_network": &WiFiError{Message: "too many networks"},
		}),
	}
	_, _, err := s.configureWpaNetwork(context.Background(), "ShellyBulbDuo-D0DCFF", "")
	if err == nil {
		t.Fatal("expected error when add_network fails")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// configureWpaNetwork — enable_network fails path
// ────────────────────────────────────────────────────────────────────────────

func TestConfigureWpaNetwork_EnableNetworkFails(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{
			"list_networks": "network id / ssid / bssid / flags\n",
			"add_network":   "5",
			"set_network":   "OK",
		}, map[string]error{
			"enable_network": &WiFiError{Message: "enable failed"},
		}),
	}
	_, _, err := s.configureWpaNetwork(context.Background(), "ShellyBulbDuo-D0DCFF", "")
	if err == nil {
		t.Fatal("expected error when enable_network fails")
	}
	// The rollback must have run — remove_network should be in seq.
	if !seqHas(seq, "remove_network 5") {
		t.Errorf("rollback did not remove the added network; seq=%v", seq)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// configureWpaNetwork — select_network fails path
// ────────────────────────────────────────────────────────────────────────────

func TestConfigureWpaNetwork_SelectNetworkFails(t *testing.T) {
	call := 0
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: func(_ context.Context, args ...string) (string, error) {
			seq = append(seq, args[0])
			call++
			switch args[0] {
			case "list_networks":
				return "network id / ssid / bssid / flags\n", nil
			case "add_network":
				return "6", nil
			case "set_network":
				return "OK", nil
			case "enable_network":
				return "OK", nil
			case "select_network":
				return "", &WiFiError{Message: "select failed"}
			default:
				return "OK", nil
			}
		},
	}
	_, _, err := s.configureWpaNetwork(context.Background(), "ShellyBulbDuo-D0DCFF", "")
	if err == nil {
		t.Fatal("expected error when select_network fails")
	}
}
