//go:build linux

package discovery

import (
	"context"
	"strings"
	"testing"
)

func TestParseNmcliLine(t *testing.T) {
	s := &platformWiFiScanner{
		iface: "wlan0",
	}

	tests := []struct {
		name       string
		line       string
		wantSSID   string
		wantSignal int
		wantNil    bool
	}{
		{
			name:    "not current network",
			line:    " :HomeNetwork:75:WPA2",
			wantNil: true,
		},
		{
			name:    "empty line",
			line:    "",
			wantNil: true,
		},
		{
			name:       "current network with all fields",
			line:       "*:MyNetwork:80:WPA2",
			wantSSID:   "MyNetwork",
			wantSignal: 80,
			wantNil:    false,
		},
		{
			name:       "current network SSID only",
			line:       "*:OnlySSID",
			wantSSID:   "OnlySSID",
			wantSignal: 0,
			wantNil:    false,
		},
		{
			name:       "current network with signal",
			line:       "*:NetworkName:65",
			wantSSID:   "NetworkName",
			wantSignal: 65,
			wantNil:    false,
		},
		{
			name:    "empty SSID after asterisk",
			line:    "*::80:WPA2",
			wantNil: true,
		},
		{
			name:    "only asterisk and colon",
			line:    "*:",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.parseNmcliLine(tt.line)
			if tt.wantNil {
				if result != nil {
					t.Errorf("parseNmcliLine(%q) = %+v, want nil", tt.line, result)
				}
				return
			}

			if result == nil {
				t.Fatalf("parseNmcliLine(%q) = nil, want non-nil", tt.line)
			}

			if result.SSID != tt.wantSSID {
				t.Errorf("parseNmcliLine(%q).SSID = %q, want %q", tt.line, result.SSID, tt.wantSSID)
			}

			if result.Signal != tt.wantSignal {
				t.Errorf("parseNmcliLine(%q).Signal = %d, want %d", tt.line, result.Signal, tt.wantSignal)
			}
		})
	}
}

func TestParseNmcliScanOutput(t *testing.T) {
	input := `HomeNetwork:AA\:BB\:CC\:DD\:EE\:FF:85:6:WPA2
ShellyBulb-AABBCC:11\:22\:33\:44\:55\:66:45:1:
Office:77\:88\:99\:AA\:BB\:CC:72:11:WPA2 WPA3
`
	networks := parseNmcliScanOutput(input)
	if len(networks) != 3 {
		t.Fatalf("expected 3 networks, got %d", len(networks))
	}

	// First network.
	if networks[0].SSID != "HomeNetwork" {
		t.Errorf("network[0].SSID = %q, want %q", networks[0].SSID, "HomeNetwork")
	}
	if networks[0].BSSID != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("network[0].BSSID = %q, want %q", networks[0].BSSID, "AA:BB:CC:DD:EE:FF")
	}
	if networks[0].Signal != -15 { // 85 - 100
		t.Errorf("network[0].Signal = %d, want -15", networks[0].Signal)
	}
	if networks[0].Channel != 6 {
		t.Errorf("network[0].Channel = %d, want 6", networks[0].Channel)
	}
	if networks[0].Security != "WPA2" {
		t.Errorf("network[0].Security = %q, want %q", networks[0].Security, "WPA2")
	}

	// Second network (open).
	if networks[1].SSID != "ShellyBulb-AABBCC" {
		t.Errorf("network[1].SSID = %q, want %q", networks[1].SSID, "ShellyBulb-AABBCC")
	}
	if networks[1].Security != "" {
		t.Errorf("network[1].Security = %q, want empty", networks[1].Security)
	}
}

func TestParseIwScanOutput(t *testing.T) {
	input := `BSS aa:bb:cc:dd:ee:ff(on wlan0)
	freq: 2437
	signal: -50.00 dBm
	SSID: HomeNetwork
	RSN:	 * Version: 1
BSS 11:22:33:44:55:66(on wlan0)
	freq: 5180
	signal: -72.00 dBm
	SSID: ShellyBulb-AABBCC
`
	networks := parseIwScanOutput(input)
	if len(networks) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(networks))
	}

	if networks[0].SSID != "HomeNetwork" {
		t.Errorf("network[0].SSID = %q, want %q", networks[0].SSID, "HomeNetwork")
	}
	if networks[0].BSSID != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("network[0].BSSID = %q, want %q", networks[0].BSSID, "AA:BB:CC:DD:EE:FF")
	}
	if networks[0].Signal != -50 {
		t.Errorf("network[0].Signal = %d, want -50", networks[0].Signal)
	}
	if networks[0].Channel != 6 {
		t.Errorf("network[0].Channel = %d, want 6", networks[0].Channel)
	}
	if networks[0].Security != "WPA2" {
		t.Errorf("network[0].Security = %q, want %q", networks[0].Security, "WPA2")
	}

	if networks[1].SSID != "ShellyBulb-AABBCC" {
		t.Errorf("network[1].SSID = %q, want %q", networks[1].SSID, "ShellyBulb-AABBCC")
	}
	if networks[1].Channel != 36 {
		t.Errorf("network[1].Channel = %d, want 36", networks[1].Channel)
	}
}

func TestFrequencyToChannel(t *testing.T) {
	tests := []struct {
		freq    int
		channel int
	}{
		{2412, 1},
		{2437, 6},
		{2462, 11},
		{2484, 14},
		{5180, 36},
		{5240, 48},
		{5745, 149},
		{5825, 165},
		{5955, 1},  // WiFi 6E
		{6115, 33}, // WiFi 6E
		{1000, 0},  // Unknown
	}

	for _, tt := range tests {
		got := frequencyToChannel(tt.freq)
		if got != tt.channel {
			t.Errorf("frequencyToChannel(%d) = %d, want %d", tt.freq, got, tt.channel)
		}
	}
}

func TestNmFlagsToSecurity(t *testing.T) {
	tests := []struct {
		name     string
		flags    uint32
		wpa      uint32
		rsn      uint32
		expected string
	}{
		{"open", 0, 0, 0, ""},
		{"WEP", 0x1, 0, 0, "WEP"},
		{"WPA", 0, 0x100, 0, "WPA"},
		{"WPA2", 0, 0, 0x100, "WPA2"},
		{"WPA3", 0, 0, 0x200, "WPA3"},
		{"WPA2 with privacy", 0x1, 0, 0x100, "WPA2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nmFlagsToSecurity(tt.flags, tt.wpa, tt.rsn)
			if got != tt.expected {
				t.Errorf("nmFlagsToSecurity(%d, %d, %d) = %q, want %q",
					tt.flags, tt.wpa, tt.rsn, got, tt.expected)
			}
		})
	}
}

func TestSplitNmcliFields(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"HomeNetwork:AA\\:BB\\:CC:80:6:WPA2", []string{"HomeNetwork", "AA:BB:CC", "80", "6", "WPA2"}},
		{"Simple:Field", []string{"Simple", "Field"}},
		{"", []string{""}},
		{"NoDelim", []string{"NoDelim"}},
	}

	for _, tt := range tests {
		got := splitNmcliFields(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitNmcliFields(%q) = %v (len %d), want %v (len %d)",
				tt.input, got, len(got), tt.want, len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitNmcliFields(%q)[%d] = %q, want %q",
					tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestDetectWiFiInterface(t *testing.T) {
	// This test just verifies the function doesn't panic
	// and returns a non-empty string (even if the default fallback)
	iface := detectWiFiInterface()
	if iface == "" {
		t.Error("detectWiFiInterface() returned empty string")
	}
}

func TestHasCommand(t *testing.T) {
	// Test with a command that should exist on any Linux system
	if !hasCommand("ls") {
		t.Skip("ls command not found, skipping")
	}

	// Test with a command that definitely doesn't exist
	if hasCommand("nonexistent-command-12345") {
		t.Error("hasCommand returned true for nonexistent command")
	}
}

func TestWpaPSKForSSID(t *testing.T) {
	const conf = `ctrl_interface=/run/wpa_supplicant
update_config=1

network={
	ssid="OnyxCheetah4.7"
	psk="hunter2hunter"
	key_mgmt=WPA-PSK
}

network={
	ssid="HashedNet"
	psk=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
}

network={
	ssid="OpenNet"
	key_mgmt=NONE
}
`
	tests := []struct {
		name string
		ssid string
		want string
	}{
		{"quoted passphrase recovered", "OnyxCheetah4.7", "hunter2hunter"},
		{"hashed psk skipped", "HashedNet", ""},
		{"open network has no psk", "OpenNet", ""},
		{"unknown ssid", "Nope", ""},
		{"empty ssid", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wpaPSKForSSID(conf, tt.ssid); got != tt.want {
				t.Errorf("wpaPSKForSSID(%q) = %q, want %q", tt.ssid, got, tt.want)
			}
		})
	}
}

func TestParseWpaNetworkList(t *testing.T) {
	out := "network id / ssid / bssid / flags\n" +
		"0\tOnyxCheetah4.7\tany\t[CURRENT]\n" +
		"1\tShellyBulb-AABBCC\tany\t[DISABLED]\n" +
		"2\tNoFlagsNet\tany\t\n" +
		"\n"
	got := parseWpaNetworkList(out)
	if len(got) != 3 {
		t.Fatalf("expected 3 networks, got %d: %+v", len(got), got)
	}
	if got[0].id != "0" || got[0].ssid != "OnyxCheetah4.7" || !got[0].current {
		t.Errorf("network[0] = %+v, want id=0 ssid=OnyxCheetah4.7 current=true", got[0])
	}
	if got[1].current {
		t.Errorf("network[1] (%+v) should not be current", got[1])
	}
	if got[2].id != "2" || got[2].ssid != "NoFlagsNet" || got[2].current {
		t.Errorf("network[2] = %+v, want id=2 ssid=NoFlagsNet current=false", got[2])
	}
}

// fakeWpa records each wpa_cli invocation and replays scripted output. A verb
// listed in failOn returns failErr; otherwise outputs[verb] (default "OK").
func fakeWpa(seq *[]string, outputs map[string]string, failOn map[string]error) func(context.Context, ...string) (string, error) {
	return func(_ context.Context, args ...string) (string, error) {
		*seq = append(*seq, strings.Join(args, " "))
		verb := args[0]
		if failOn != nil {
			if e, ok := failOn[verb]; ok {
				return "", e
			}
		}
		if o, ok := outputs[verb]; ok {
			return o, nil
		}
		return "OK", nil
	}
}

func seqHas(seq []string, want string) bool {
	for _, c := range seq {
		if c == want {
			return true
		}
	}
	return false
}

func TestConfigureWpaNetwork_FreshAddThenRollback(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{
			"list_networks": "network id / ssid / bssid / flags\n0\tHome\tany\t[CURRENT]\n",
			"add_network":   "5",
		}, nil),
	}

	id, rollback, err := s.configureWpaNetwork(context.Background(), "ShellyAP", "")
	if err != nil {
		t.Fatalf("configureWpaNetwork returned error: %v", err)
	}
	if id != "5" {
		t.Errorf("id = %q, want 5", id)
	}
	for _, want := range []string{
		"add_network",
		`set_network 5 ssid "ShellyAP"`,
		"set_network 5 key_mgmt NONE", // open AP: no password
		"enable_network 5",
		"select_network 5",
	} {
		if !seqHas(seq, want) {
			t.Errorf("connect sequence missing %q; got %v", want, seq)
		}
	}

	// Rollback must drop the block this call added and re-enable the prior network,
	// so a failed hop never strands the host with everything disabled.
	rollback()
	for _, want := range []string{"remove_network 5", "enable_network all", "select_network 0"} {
		if !seqHas(seq, want) {
			t.Errorf("rollback sequence missing %q; got %v", want, seq)
		}
	}
}

func TestConfigureWpaNetwork_ReusesExistingBlock(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{
			"list_networks": "network id / ssid / bssid / flags\n" +
				"0\tHome\tany\t[CURRENT]\n1\tShellyAP\tany\t[DISABLED]\n",
		}, nil),
	}

	id, rollback, err := s.configureWpaNetwork(context.Background(), "ShellyAP", "")
	if err != nil {
		t.Fatalf("configureWpaNetwork returned error: %v", err)
	}
	if id != "1" {
		t.Errorf("id = %q, want 1 (reused)", id)
	}
	if seqHas(seq, "add_network") {
		t.Errorf("reuse path must NOT add_network (that is the per-hop leak); got %v", seq)
	}
	for _, unwanted := range []string{`set_network 1 ssid "ShellyAP"`, "set_network 1 key_mgmt NONE"} {
		if seqHas(seq, unwanted) {
			t.Errorf("reuse path must not reconfigure block; unexpected %q in %v", unwanted, seq)
		}
	}
	if !seqHas(seq, "enable_network 1") || !seqHas(seq, "select_network 1") {
		t.Errorf("reuse path must enable+select existing block; got %v", seq)
	}

	// Reused block was not created here, so rollback must not remove it.
	rollback()
	if seqHas(seq, "remove_network 1") {
		t.Errorf("rollback must not remove a reused block; got %v", seq)
	}
	if !seqHas(seq, "enable_network all") || !seqHas(seq, "select_network 0") {
		t.Errorf("rollback must still restore prior network; got %v", seq)
	}
}

func TestConfigureWpaNetwork_FailureRollsBackAutomatically(t *testing.T) {
	var seq []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&seq, map[string]string{
			"list_networks": "network id / ssid / bssid / flags\n0\tHome\tany\t[CURRENT]\n",
			"add_network":   "7",
		}, map[string]error{
			"select_network": &WiFiError{Message: "boom"},
		}),
	}

	_, _, err := s.configureWpaNetwork(context.Background(), "ShellyAP", "secret")
	if err == nil {
		t.Fatal("expected error when select_network fails")
	}
	// WPA-PSK path set the psk, then the failed select_network must auto-roll-back.
	if !seqHas(seq, `set_network 7 psk "secret"`) {
		t.Errorf("password path must set psk; got %v", seq)
	}
	for _, want := range []string{"remove_network 7", "enable_network all", "select_network 0"} {
		if !seqHas(seq, want) {
			t.Errorf("auto-rollback missing %q; got %v", want, seq)
		}
	}
}

func TestUnquoteWpaValue(t *testing.T) {
	tests := map[string]string{
		`"quoted"`:  "quoted",
		`  "pad"  `: "pad",
		`bare`:      "bare",
		`"`:         `"`,
		``:          "",
	}
	for in, want := range tests {
		if got := unquoteWpaValue(in); got != want {
			t.Errorf("unquoteWpaValue(%q) = %q, want %q", in, got, want)
		}
	}
}
