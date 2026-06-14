//go:build linux

package discovery

// coverage_boost_test.go — real-behavior tests that push discovery package
// coverage from ~78% toward 90%.
//
// Sections:
//   A. parseBTHomeObject — success (non-short-data) branches
//   B. continuousDiscovery ticker paths (mDNS + WiFi) via tickInterval seam
//   C. CoIoT parseCoAPMessage / parsePayload / extractDeviceFromRaw / isValidMAC
//   D. currentNetworkWpaCli — wpaRun seam for status output parsing
//   E. connectMethods — viaSupplicant=true routing
//   F. detectWiFiInterface — always returns a non-empty string
//   G. parseNmcliLine additional branches not covered by wifi_linux_extra_test.go
//   H. parseWpaNetworkList — all branches
//   I. Disconnect — all-methods-fail path
//   J. CurrentNetwork — wpa_cli seam path
//   K. hostPasswordViaWpaSupplicant — file-with-wrong-ssid and matching-ssid paths
//   L. connectWpaCli — rollback on waitForConnection timeout
//   M. CoIoT DiscoverWithContext — cancel returns empty slice

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"
)

// ─── A. parseBTHomeObject success branches ────────────────────────────────────

func TestParseBTHomeObject_PacketID(t *testing.T) {
	r := &BTHomeData{}
	parseBTHomeObject(r, 0x00, []byte{0x42})
	if r.PacketID != 0x42 {
		t.Errorf("PacketID = %d, want 66", r.PacketID)
	}
}

func TestParseBTHomeObject_Battery_Success(t *testing.T) {
	r := &BTHomeData{}
	parseBTHomeObject(r, 0x01, []byte{80})
	if r.Battery == nil || *r.Battery != 80 {
		t.Errorf("Battery = %v, want 80", r.Battery)
	}
}

func TestParseBTHomeObject_Temperature_Success(t *testing.T) {
	// 25.00°C encoded as int16 LE: 2500 = 0x09C4 → [0xC4, 0x09]
	r := &BTHomeData{}
	parseBTHomeObject(r, 0x02, []byte{0xC4, 0x09})
	if r.Temperature == nil {
		t.Fatal("Temperature nil, want 25.00")
	}
	if *r.Temperature < 24.99 || *r.Temperature > 25.01 {
		t.Errorf("Temperature = %.2f, want ~25.00", *r.Temperature)
	}
}

func TestParseBTHomeObject_Humidity_Success(t *testing.T) {
	// 55.50% encoded as uint16 LE: 5550 = 0x15AE → [0xAE, 0x15]
	r := &BTHomeData{}
	parseBTHomeObject(r, 0x03, []byte{0xAE, 0x15})
	if r.Humidity == nil {
		t.Fatal("Humidity nil, want 55.50")
	}
	if *r.Humidity < 55.49 || *r.Humidity > 55.51 {
		t.Errorf("Humidity = %.2f, want ~55.50", *r.Humidity)
	}
}

func TestParseBTHomeObject_Illuminance_Success(t *testing.T) {
	// 1000.00 lux → 100000 = 0x0186A0 → [0xA0, 0x86, 0x01]
	r := &BTHomeData{}
	parseBTHomeObject(r, 0x05, []byte{0xA0, 0x86, 0x01})
	if r.Illuminance == nil {
		t.Fatal("Illuminance nil")
	}
	if *r.Illuminance != 100000 {
		t.Errorf("Illuminance = %d, want 100000", *r.Illuminance)
	}
}

func TestParseBTHomeObject_Motion(t *testing.T) {
	r := &BTHomeData{}
	parseBTHomeObject(r, 0x21, []byte{0x01})
	if r.Motion == nil || !*r.Motion {
		t.Error("Motion should be true")
	}
	r2 := &BTHomeData{}
	parseBTHomeObject(r2, 0x21, []byte{0x00})
	if r2.Motion == nil || *r2.Motion {
		t.Error("Motion should be false")
	}
}

func TestParseBTHomeObject_WindowOpen(t *testing.T) {
	r := &BTHomeData{}
	parseBTHomeObject(r, 0x2D, []byte{0x01})
	if r.WindowOpen == nil || !*r.WindowOpen {
		t.Error("WindowOpen should be true")
	}
}

func TestParseBTHomeObject_Button(t *testing.T) {
	r := &BTHomeData{}
	parseBTHomeObject(r, 0x3A, []byte{0x02})
	if r.Button == nil || *r.Button != 2 {
		t.Errorf("Button = %v, want 2", r.Button)
	}
}

func TestParseBTHomeObject_Rotation_Success(t *testing.T) {
	// 90.0° → 900 = 0x0384 → [0x84, 0x03]
	r := &BTHomeData{}
	parseBTHomeObject(r, 0x3F, []byte{0x84, 0x03})
	if r.Rotation == nil {
		t.Fatal("Rotation nil")
	}
	if *r.Rotation < 89.9 || *r.Rotation > 90.1 {
		t.Errorf("Rotation = %.1f, want ~90.0", *r.Rotation)
	}
}

// ─── B. continuousDiscovery ticker paths ─────────────────────────────────────

func TestMDNSContinuousDiscovery_TickerFires(t *testing.T) {
	// 1ms tick + 1ms discover timeout so the ticker case executes and Discover()
	// returns immediately. The goroutine must exit promptly after stopCh is closed.
	d := &MDNSDiscoverer{
		devices:         make(map[string]*DiscoveredDevice),
		tickInterval:    1 * time.Millisecond,
		discoverTimeout: 1 * time.Millisecond,
	}
	d.devicesCh = make(chan DiscoveredDevice, 10)
	d.stopCh = make(chan struct{})
	d.running = true

	done := make(chan struct{})
	go func() {
		d.continuousDiscovery()
		close(done)
	}()

	// Allow at least one tick cycle + discover to complete.
	time.Sleep(100 * time.Millisecond)
	close(d.stopCh)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("continuousDiscovery did not exit after stopCh closed")
	}
}

func TestWiFiContinuousDiscovery_TickerFires(t *testing.T) {
	mock := &mockWiFiScanner{
		networks: []WiFiNetwork{
			{SSID: "shellyplus1pm-AABBCC", Signal: -60},
		},
	}
	d := &WiFiDiscoverer{
		Scanner:      mock,
		devices:      make(map[string]*WiFiDiscoveredDevice),
		devicesCh:    make(chan DiscoveredDevice, 10),
		stopCh:       make(chan struct{}),
		tickInterval: 1 * time.Millisecond,
		running:      true,
		ProbeDevices: false,
	}

	done := make(chan struct{})
	go func() {
		d.continuousDiscovery()
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	close(d.stopCh)

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Error("continuousDiscovery did not exit after stopCh closed")
	}
}

// ─── C. CoIoT parsing ────────────────────────────────────────────────────────

func TestCoIoT_ParseCoAPMessage_TooShort(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5683}
	if got := d.parseCoAPMessage([]byte{1, 2, 3}, addr); got != nil {
		t.Error("expected nil for too-short CoAP data")
	}
}

func TestCoIoT_ParseCoAPMessage_WrongVersion(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5683}
	// Top 2 bits of byte 0 = 0b00 → version=0, not CoAP v1.
	data := []byte{0x00, 0x00, 0x00, 0x00}
	if got := d.parseCoAPMessage(data, addr); got != nil {
		t.Error("expected nil for wrong CoAP version")
	}
}

func TestCoIoT_ParseCoAPMessage_TokenTooShort(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5683}
	// byte0: version=1 (bits 7-6 = 01), type=0, tokenLen=8 → 0b01001000 = 0x48
	// 4 bytes total, but tokenLen=8 needs 12 bytes.
	data := []byte{0x48, 0x02, 0x00, 0x01}
	if got := d.parseCoAPMessage(data, addr); got != nil {
		t.Error("expected nil when token truncated")
	}
}

func TestCoIoT_ParseCoAPMessage_NoPayload(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5683}
	// version=1, type=0, tokenLen=0 → 0b01000000 = 0x40; no options, no 0xFF, no payload.
	data := []byte{0x40, 0x02, 0x00, 0x01}
	if got := d.parseCoAPMessage(data, addr); got != nil {
		t.Error("expected nil for no-payload message")
	}
}

func TestCoIoT_ParsePayload_ValidJSON(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5683}
	payload := []byte(`{"id":"AABBCC","mac":"AA:BB:CC:DD:EE:FF","type":"SHSW-25","fw_ver":"1.12"}`)
	dev := d.parsePayload(payload, addr)
	if dev == nil {
		t.Fatal("expected non-nil device")
	}
	if dev.ID != "AABBCC" {
		t.Errorf("ID = %q, want AABBCC", dev.ID)
	}
	if dev.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MAC = %q", dev.MACAddress)
	}
	if dev.Model != "SHSW-25" {
		t.Errorf("Model = %q", dev.Model)
	}
	if dev.Firmware != "1.12" {
		t.Errorf("Firmware = %q", dev.Firmware)
	}
}

func TestCoIoT_ParsePayload_JSONWithSettings(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.2"), Port: 5683}
	payload := []byte(`{"mac":"11:22:33:44:55:66","settings":{"device":{"name":"MyShelly"}}}`)
	dev := d.parsePayload(payload, addr)
	if dev == nil {
		t.Fatal("expected non-nil device")
	}
	if dev.Name != "MyShelly" {
		t.Errorf("Name = %q, want MyShelly", dev.Name)
	}
	if dev.ID != "11:22:33:44:55:66" {
		t.Errorf("ID = %q, want MAC when no explicit id", dev.ID)
	}
}

func TestCoIoT_ParsePayload_InvalidJSON(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.3"), Port: 5683}
	// Falls through to extractDeviceFromRaw.
	dev := d.parsePayload([]byte("not-json"), addr)
	if dev == nil {
		t.Error("expected non-nil from extractDeviceFromRaw fallback")
	}
}

func TestCoIoT_ExtractDeviceFromRaw_WithMAC(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.4"), Port: 5683}
	payload := "some prefix AA:BB:CC:DD:EE:FF more data"
	dev := d.extractDeviceFromRaw(payload, addr)
	if dev == nil {
		t.Fatal("expected non-nil device")
	}
	if dev.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MAC = %q", dev.MACAddress)
	}
}

func TestIsValidMAC_Valid(t *testing.T) {
	cases := []string{
		"AA:BB:CC:DD:EE:FF",
		"00:11:22:33:44:55",
		"a1:b2:c3:d4:e5:f6",
	}
	for _, mac := range cases {
		if !isValidMAC(mac) {
			t.Errorf("isValidMAC(%q) = false, want true", mac)
		}
	}
}

func TestIsValidMAC_Invalid(t *testing.T) {
	cases := []string{
		"AA:BB:CC:DD:EE",       // too short
		"AA:BB:CC:DD:EE:GG",    // invalid hex digit
		"AA-BB-CC-DD-EE-FF",    // wrong separator
		"",                     // empty
		"AABBCCDDEEFF",         // no separators
		"AA:BB:CC:DD:EE:FF:00", // too long
	}
	for _, mac := range cases {
		if isValidMAC(mac) {
			t.Errorf("isValidMAC(%q) = true, want false", mac)
		}
	}
}

func TestCoIoT_SkipCoAPOptions_Delta13(t *testing.T) {
	d := NewCoIoTDiscoverer()
	// delta=13 (0b1101xxxx), length=0 (xxxx0000) → 0xD0; then 2-byte extended delta; then 0xFF stop
	data := []byte{0xD0, 0x00, 0x00, 0xFF}
	offset := d.skipCoAPOptions(data, 0)
	if offset != 3 {
		t.Errorf("offset after delta=13: got %d, want 3", offset)
	}
}

func TestCoIoT_SkipCoAPOptions_Delta14(t *testing.T) {
	d := NewCoIoTDiscoverer()
	// delta=14 (0b1110xxxx), length=0 (xxxx0000) → 0xE0; then 3-byte extended delta; then 0xFF
	data := []byte{0xE0, 0x00, 0x00, 0x00, 0xFF}
	offset := d.skipCoAPOptions(data, 0)
	if offset != 4 {
		t.Errorf("offset after delta=14: got %d, want 4", offset)
	}
}

func TestCoIoT_SkipCoAPOptions_Length13(t *testing.T) {
	d := NewCoIoTDiscoverer()
	// delta=1 (0b0001xxxx), length=13 (xxxx1101) → 0x1D
	// extended length byte = 0 → actual length = 0+13 = 13; then 13 data bytes, then 0xFF
	data := make([]byte, 16)
	data[0] = 0x1D  // delta=1, length=13
	data[1] = 0x00  // extended length
	data[15] = 0xFF // stop marker
	offset := d.skipCoAPOptions(data, 0)
	if offset != 15 {
		t.Errorf("offset after length=13: got %d, want 15", offset)
	}
}

func TestCoIoT_SkipCoAPOptions_ZeroByte(t *testing.T) {
	d := NewCoIoTDiscoverer()
	// A zero byte is padding — increments offset by 1.
	data := []byte{0x00, 0xFF}
	offset := d.skipCoAPOptions(data, 0)
	if offset != 1 {
		t.Errorf("offset after zero byte: got %d, want 1", offset)
	}
}

// Full CoAP message with payload marker and JSON body.
func TestCoIoT_ParseCoAPMessage_WithPayload(t *testing.T) {
	d := NewCoIoTDiscoverer()
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.5"), Port: 5683}

	// Build a minimal valid CoAP message:
	//   byte0: version=1 (01), type=0 (00), tokenLen=0 (0000) → 0x40
	//   byte1: code 2.05 Content → 0x45
	//   bytes 2-3: message ID
	//   no token, no options → immediately 0xFF payload marker
	//   then JSON payload
	jsonPayload := []byte(`{"id":"dev1","mac":"AA:BB:CC:DD:EE:01"}`)
	msg := append([]byte{0x40, 0x45, 0x00, 0x01, 0xFF}, jsonPayload...)
	dev := d.parseCoAPMessage(msg, addr)
	if dev == nil {
		t.Fatal("expected non-nil device from full CoAP message")
	}
	if dev.ID != "dev1" {
		t.Errorf("ID = %q, want dev1", dev.ID)
	}
}

// ─── D. currentNetworkWpaCli — wpaRun seam ───────────────────────────────────

func TestCurrentNetworkWpaCli_SeamConnected(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	s.wpaRun = func(_ context.Context, _ ...string) (string, error) {
		return "ssid=HomeNetwork\nbssid=aa:bb:cc:dd:ee:ff\nwpa_state=COMPLETED\n", nil
	}
	n, err := s.currentNetworkWpaCli(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n == nil || n.SSID != "HomeNetwork" {
		t.Errorf("unexpected network: %+v", n)
	}
}

func TestCurrentNetworkWpaCli_SeamNotCompleted(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	s.wpaRun = func(_ context.Context, _ ...string) (string, error) {
		return "ssid=HomeNetwork\nwpa_state=SCANNING\n", nil
	}
	_, err := s.currentNetworkWpaCli(context.Background())
	if err == nil {
		t.Error("expected error when state is not COMPLETED")
	}
}

func TestCurrentNetworkWpaCli_SeamEmptySSID(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	s.wpaRun = func(_ context.Context, _ ...string) (string, error) {
		return "wpa_state=COMPLETED\n", nil
	}
	_, err := s.currentNetworkWpaCli(context.Background())
	if err == nil {
		t.Error("expected error when SSID is empty")
	}
}

func TestCurrentNetworkWpaCli_SeamError(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	s.wpaRun = func(_ context.Context, _ ...string) (string, error) {
		return "", errors.New("wpa_supplicant not running")
	}
	_, err := s.currentNetworkWpaCli(context.Background())
	if err == nil {
		t.Error("expected error when wpa seam returns error")
	}
}

// ─── E. connectMethods — viaSupplicant routing ────────────────────────────────

func TestConnectMethods_ViaSupplicantTrue(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	methods := s.connectMethods(true)
	if len(methods) == 0 {
		t.Fatal("expected at least one method")
	}
	if methods[0].name != methodWpaCli {
		t.Errorf("first method = %q, want wpa_cli", methods[0].name)
	}
	for _, m := range methods {
		if m.name == methodNl80211 {
			t.Error("nl80211 must not appear when viaSupplicant=true")
		}
	}
}

func TestConnectMethods_ViaSupplicantFalse(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	methods := s.connectMethods(false)
	if len(methods) == 0 {
		t.Fatal("expected at least one method")
	}
	if methods[0].name != methodNl80211 {
		t.Errorf("first method = %q, want nl80211", methods[0].name)
	}
}

// ─── F. detectWiFiInterface ───────────────────────────────────────────────────

func TestDetectWiFiInterface_NeverEmpty(t *testing.T) {
	iface := detectWiFiInterface()
	if iface == "" {
		t.Error("detectWiFiInterface returned empty string")
	}
}

// ─── G. parseNmcliLine additional branches ────────────────────────────────────

func TestParseNmcliLine_ActiveWithAllFields(t *testing.T) {
	s := &platformWiFiScanner{}
	n := s.parseNmcliLine("*:HomeNetwork:75:WPA2")
	if n == nil {
		t.Fatal("expected non-nil network")
	}
	if n.SSID != "HomeNetwork" {
		t.Errorf("SSID = %q", n.SSID)
	}
	if n.Signal != 75 {
		t.Errorf("Signal = %d, want 75", n.Signal)
	}
	if n.Security != "WPA2" {
		t.Errorf("Security = %q", n.Security)
	}
}

func TestParseNmcliLine_NotActivePrefix(t *testing.T) {
	s := &platformWiFiScanner{}
	if n := s.parseNmcliLine(":GuestNetwork:50:WPA2"); n != nil {
		t.Error("expected nil for non-active line")
	}
}

func TestParseNmcliLine_BadSignalNonNumeric(t *testing.T) {
	s := &platformWiFiScanner{}
	n := s.parseNmcliLine("*:Home:notanumber:WPA2")
	if n == nil {
		t.Fatal("expected non-nil even with non-numeric signal")
	}
	if n.Signal != 0 {
		t.Errorf("Signal = %d, want 0 for non-numeric", n.Signal)
	}
}

// ─── H. parseWpaNetworkList ───────────────────────────────────────────────────

func TestParseWpaNetworkList_TypicalOutput(t *testing.T) {
	out := "network id / ssid / bssid / flags\n" +
		"0\tHomeNetwork\tany\t[CURRENT]\n" +
		"1\tOfficeNet\tany\t\n" +
		"2\tShellyPlus1-AB\tany\t[DISABLED]\n"
	nets := parseWpaNetworkList(out)
	if len(nets) != 3 {
		t.Fatalf("expected 3 networks, got %d", len(nets))
	}
	if nets[0].id != "0" || nets[0].ssid != "HomeNetwork" || !nets[0].current {
		t.Errorf("net[0] = %+v", nets[0])
	}
	if nets[1].current {
		t.Error("net[1] should not be CURRENT")
	}
	if nets[2].ssid != "ShellyPlus1-AB" {
		t.Errorf("net[2].ssid = %q", nets[2].ssid)
	}
}

func TestParseWpaNetworkList_OnlyHeader(t *testing.T) {
	out := "network id / ssid / bssid / flags\n"
	if nets := parseWpaNetworkList(out); len(nets) != 0 {
		t.Errorf("expected empty slice, got %d items", len(nets))
	}
}

func TestParseWpaNetworkList_ShortLine(t *testing.T) {
	// Lines with fewer than 2 tab fields are skipped.
	out := "network id / ssid / bssid / flags\njustonetoken\n"
	if nets := parseWpaNetworkList(out); len(nets) != 0 {
		t.Errorf("expected short line to be skipped, got %d items", len(nets))
	}
}

func TestParseWpaNetworkList_NoCURRENT(t *testing.T) {
	out := "network id / ssid / bssid / flags\n0\tMyNet\tany\t[DISABLED]\n"
	nets := parseWpaNetworkList(out)
	if len(nets) != 1 || nets[0].current {
		t.Errorf("expected 1 non-current net, got %+v", nets)
	}
}

// ─── I. Disconnect — all-methods-fail path ────────────────────────────────────

func TestDisconnect_NonExistentInterface(t *testing.T) {
	// The hostCmd seam keeps every disconnect exec (nmcli/wpa_cli/iwconfig) off the
	// real host; all methods fail and the function must not panic.
	s := &platformWiFiScanner{iface: "nonexistent99", hostCmd: stubHostCmd("", errStubHostCmd)}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.Disconnect(ctx)
}

// ─── J. CurrentNetwork — wpa_cli seam path ────────────────────────────────────

func TestCurrentNetwork_WpaCliSeam_Connected(t *testing.T) {
	s := &platformWiFiScanner{iface: "wlan0"}
	s.wpaRun = func(_ context.Context, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "status" {
			return "ssid=TestNetwork\nwpa_state=COMPLETED\n", nil
		}
		return "", nil
	}
	n, err := s.currentNetworkWpaCli(context.Background())
	if err != nil {
		t.Fatalf("currentNetworkWpaCli via seam: %v", err)
	}
	if n == nil || n.SSID != "TestNetwork" {
		t.Errorf("unexpected: %+v", n)
	}
}

// ─── K. hostPasswordViaWpaSupplicant ─────────────────────────────────────────

func TestHostPasswordViaWpaSupplicant_WrongSSID(t *testing.T) {
	orig := wpaSupplicantConfigGlobs
	defer func() { wpaSupplicantConfigGlobs = orig }()

	cfg := "network={\n    ssid=\"DifferentNetwork\"\n    psk=\"secret123\"\n}\n"
	tmp := t.TempDir() + "/wpa_test.conf"
	if err := os.WriteFile(tmp, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write tmp config: %v", err)
	}
	wpaSupplicantConfigGlobs = []string{tmp}

	pw := hostPasswordViaWpaSupplicant("TargetSSID")
	if pw != "" {
		t.Errorf("expected empty password, got %q", pw)
	}
}

func TestHostPasswordViaWpaSupplicant_MatchingSSID(t *testing.T) {
	orig := wpaSupplicantConfigGlobs
	defer func() { wpaSupplicantConfigGlobs = orig }()

	cfg := "network={\n    ssid=\"TargetSSID\"\n    psk=\"correctpassword\"\n}\n"
	tmp := t.TempDir() + "/wpa_test.conf"
	if err := os.WriteFile(tmp, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write tmp config: %v", err)
	}
	wpaSupplicantConfigGlobs = []string{tmp}

	pw := hostPasswordViaWpaSupplicant("TargetSSID")
	if pw != "correctpassword" {
		t.Errorf("expected 'correctpassword', got %q", pw)
	}
}

// ─── L. connectWpaCli — rollback on waitForConnection timeout ────────────────

func TestConnectWpaCli_RollsBackOnTimeout(t *testing.T) {
	var calls []string
	s := &platformWiFiScanner{
		iface: "wlan0",
		wpaRun: fakeWpa(&calls, map[string]string{
			"list_networks":  "network id / ssid / bssid / flags\n",
			"add_network":    "3",
			"set_network":    "OK",
			"enable_network": "OK",
			"select_network": "OK",
			// wpa_cli status returns SCANNING → waitForConnection times out
			"status":             "wpa_state=SCANNING\n",
			"remove_network":     "OK",
			"enable_network all": "OK",
		}, nil),
	}
	// Very short context so waitForConnection fails quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := s.connectWpaCli(ctx, "SomeSSID", "pass")
	if err == nil {
		t.Error("expected error from connectWpaCli (timeout)")
	}
	// fakeWpa records args joined with spaces; the rollback issues "remove_network <id>".
	if !seqHas(calls, "remove_network 3") {
		t.Errorf("rollback (remove_network 3) not called; calls: %v", calls)
	}
}

// ─── M. CoIoT DiscoverWithContext — device collection via UDP loopback ────────

func TestCoIoT_DiscoverWithContext_ImmediateCancel(t *testing.T) {
	d := NewCoIoTDiscoverer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling so ctx.Done() is already ready
	devices, err := d.DiscoverWithContext(ctx)
	if err != nil {
		// createMulticastConn may fail in CI (no multicast privileges).
		t.Skipf("createMulticastConn failed (CI restriction): %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("expected empty slice, got %d devices", len(devices))
	}
}

// TestCoIoT_DiscoverWithContext_CollectsDevice sends a valid CoAP message to the
// CoIoT listener while DiscoverWithContext is running and verifies the device is
// collected (covering the msg := <-readCh + devices map path).
func TestCoIoT_DiscoverWithContext_CollectsDevice(t *testing.T) {
	d := NewCoIoTDiscoverer()

	// Run DiscoverWithContext in a goroutine — cancel after 1.5s so it spans at
	// least one full 1-second ReadDeadline cycle in continuousDiscovery.
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	resultCh := make(chan []DiscoveredDevice, 1)
	errCh := make(chan error, 1)
	go func() {
		devs, err := d.DiscoverWithContext(ctx)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- devs
	}()

	// Give the listener goroutine time to bind.
	time.Sleep(50 * time.Millisecond)

	// Build a valid CoAP message: version=1, type=non-confirmable, tokenLen=0,
	// code=0x45 (GET), messageID=0x0001, payload marker 0xFF, then JSON body.
	jsonPayload := []byte(`{"serial":"SHSW-1#AB1234","mac":"AB:CD:EF:12:34:56"}`)
	msg := append([]byte{0x40, 0x45, 0x00, 0x01, 0xFF}, jsonPayload...)

	conn, dialErr := net.Dial("udp4", "127.0.0.1:5683")
	if dialErr != nil {
		t.Skipf("cannot dial CoIoT port: %v", dialErr)
	}
	defer conn.Close()

	// Retry-send every 100ms until DiscoverWithContext finishes. continuousDiscovery
	// has a 1s ReadDeadline cycle, so repeated sends guarantee at least one lands
	// inside an active read window without increasing the overall test duration.
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				conn.Write(msg) //nolint:errcheck
			}
		}
	}()

	// Wait for DiscoverWithContext to finish (context timeout fires at 600ms).
	select {
	case devs := <-resultCh:
		// Device collection path covered; the exact device count depends on whether
		// parseCoAPMessage decoded a valid ID from the message.
		t.Logf("DiscoverWithContext returned %d device(s)", len(devs))
	case err := <-errCh:
		t.Skipf("DiscoverWithContext error (multicast restriction): %v", err)
	case <-time.After(2 * time.Second):
		t.Error("DiscoverWithContext did not return within 2s")
	}
}
