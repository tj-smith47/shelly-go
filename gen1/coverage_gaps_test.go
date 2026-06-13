package gen1

// Tests targeting specific coverage gaps identified by go tool cover -func:
//
//   - device.go  SetTimezoneAutodetect (80%)   — missing false + error branches
//   - settings.go SetWiFiStation1    (90.9%)   — missing error branch
//   - settings.go SetApRoaming       (88.9%)   — missing error branch
//   - settings.go AddScheduleRule    (87.5%)   — missing GetRelaySettings error
//   - settings.go SetLightScheduleRules (85.7%) — missing error branch
//   - settings.go SetEMeterConfig    (83.3%)   — missing error branch
//   - settings.go SetRollerConfig    (96.9%)   — missing error branch
//   - coiot.go    Start              (0%)       — full happy path via UDP seam
//   - coiot.go    Stop               (44.9%)   — running→stopped transition
//   - coiot.go    receiveLoop        (0%)       — packet delivery via UDP seam

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/internal/testutil"
)

// ---------------------------------------------------------------------------
// device.go — SetTimezoneAutodetect
// ---------------------------------------------------------------------------

// TestSetTimezoneAutodetect_False asserts the false branch issues
// /settings?tzautodetect=false and propagates success.
func TestSetTimezoneAutodetect_False(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings?tzautodetect=false", nil, nil)

	if err := device.SetTimezoneAutodetect(context.Background(), false); err != nil {
		t.Fatalf("SetTimezoneAutodetect(false): %v", err)
	}
	calls := mock.Calls()
	if len(calls) != 1 || !strings.Contains(calls[0].Method, "tzautodetect=false") {
		t.Errorf("expected tzautodetect=false call, got %v", calls)
	}
}

// TestSetTimezoneAutodetect_Error asserts the error branch wraps and returns
// the transport error.
func TestSetTimezoneAutodetect_Error(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings?tzautodetect=true", nil, errors.New("device busy"))

	err := device.SetTimezoneAutodetect(context.Background(), true)
	if err == nil {
		t.Fatal("expected error from SetTimezoneAutodetect")
	}
	if !strings.Contains(err.Error(), "timezone autodetect") {
		t.Errorf("error message should mention 'timezone autodetect', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// settings.go — SetWiFiStation1 error branch
// ---------------------------------------------------------------------------

func TestSetWiFiStation1_Error(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/sta1?", nil, errors.New("no response"))

	err := device.SetWiFiStation1(context.Background(), true, "Net", "pass")
	if err == nil {
		t.Fatal("expected error from SetWiFiStation1")
	}
	if !strings.Contains(err.Error(), "station1") {
		t.Errorf("error should mention 'station1', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// settings.go — SetApRoaming error branch
// ---------------------------------------------------------------------------

func TestSetApRoaming_Error(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("ap_roaming_enabled", nil, errors.New("timeout"))

	err := device.SetApRoaming(context.Background(), true, -70)
	if err == nil {
		t.Fatal("expected error from SetApRoaming")
	}
	if !strings.Contains(err.Error(), "AP roaming") {
		t.Errorf("error should mention 'AP roaming', got: %v", err)
	}
}

// TestSetApRoaming_ZeroThreshold exercises the zero-threshold path (param
// omitted) so that branch is covered.
func TestSetApRoaming_ZeroThreshold(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("ap_roaming_enabled", nil, nil)

	if err := device.SetApRoaming(context.Background(), false, 0); err != nil {
		t.Fatalf("SetApRoaming(threshold=0): %v", err)
	}
	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if strings.Contains(calls[0].Method, "ap_roaming_threshold") {
		t.Error("zero threshold should not be sent as a parameter")
	}
}

// ---------------------------------------------------------------------------
// settings.go — AddScheduleRule: GetRelaySettings error branch
// ---------------------------------------------------------------------------

func TestAddScheduleRule_GetSettingsError(t *testing.T) {
	mock := testutil.NewMockTransport()
	device := NewDevice(mock)

	mock.OnCallError("/settings/relay/0", errors.New("read failed"))

	err := device.AddScheduleRule(context.Background(), 0, "0800-7F-0-on")
	if err == nil {
		t.Fatal("expected error when GetRelaySettings fails")
	}
	if !strings.Contains(err.Error(), "schedule rules") {
		t.Errorf("error should mention 'schedule rules', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// settings.go — SetLightScheduleRules error branch
// ---------------------------------------------------------------------------

func TestSetLightScheduleRules_Error(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/light/0?schedule_rules", nil, errors.New("write failed"))

	err := device.SetLightScheduleRules(context.Background(), 0, []string{"0800-7F-0-on"})
	if err == nil {
		t.Fatal("expected error from SetLightScheduleRules")
	}
	if !strings.Contains(err.Error(), "light schedule rules") {
		t.Errorf("error should mention 'light schedule rules', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// settings.go — SetEMeterConfig error branch
// ---------------------------------------------------------------------------

func TestSetEMeterConfig_Error(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/emeter/0?", nil, errors.New("emeter failed"))

	cfg := EMeterSettings{MaxPower: 2300}
	err := device.SetEMeterConfig(context.Background(), 0, cfg)
	if err == nil {
		t.Fatal("expected error from SetEMeterConfig")
	}
	if !strings.Contains(err.Error(), "emeter") {
		t.Errorf("error should mention 'emeter', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// settings.go — SetRollerConfig error branch
// ---------------------------------------------------------------------------

func TestSetRollerConfig_Error(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	mock.OnPathContains("/settings/roller/0?", nil, errors.New("roller failed"))

	cfg := &RollerConfig{DefaultState: "stop"}
	err := device.SetRollerConfig(context.Background(), 0, cfg)
	if err == nil {
		t.Fatal("expected error from SetRollerConfig")
	}
	if !strings.Contains(err.Error(), "roller") {
		t.Errorf("error should mention 'roller', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// coiot.go — Start / Stop / receiveLoop via injected UDP socket seam
// ---------------------------------------------------------------------------

// openLocalUDPPair opens a server socket on a random loopback UDP port and
// returns it along with its resolved address.  The caller owns both.
func openLocalUDPPair(t *testing.T) (srv *net.UDPConn, addr *net.UDPAddr) {
	t.Helper()
	a, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	conn, err := net.ListenUDP("udp4", a)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	bound, err := net.ResolveUDPAddr("udp4", conn.LocalAddr().String())
	if err != nil {
		conn.Close()
		t.Fatalf("resolve bound: %v", err)
	}
	return conn, bound
}

// seamedListener builds a CoIoTListener whose Start() will bind to srvConn
// instead of a real multicast socket.  The listenFn ignores the multicast
// address it is given and returns the pre-opened connection directly.
func seamedListener(srvConn *net.UDPConn) *CoIoTListener {
	l := NewCoIoTListener(WithCoIoTBufferSize(1500))
	l.listenFn = func(_ *net.UDPAddr) (*net.UDPConn, error) {
		return srvConn, nil
	}
	return l
}

// TestStart_Stop_Running exercises Start → IsRunning → Stop and asserts the
// running flag transitions correctly.
func TestStart_Stop_Running(t *testing.T) {
	srvConn, _ := openLocalUDPPair(t)
	// srvConn ownership transfers to the listener; it will be closed by Stop.

	listener := seamedListener(srvConn)

	if listener.IsRunning() {
		t.Fatal("should not be running before Start")
	}

	if err := listener.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !listener.IsRunning() {
		t.Error("should be running after Start")
	}

	// Double-Start must return an error.
	if err := listener.Start(); err == nil {
		t.Error("second Start should return 'already running' error")
	}

	if err := listener.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if listener.IsRunning() {
		t.Error("should not be running after Stop")
	}

	// Double-Stop must be a no-op (nil error).
	if err := listener.Stop(); err != nil {
		t.Fatalf("second Stop returned error: %v", err)
	}
}

// TestReceiveLoop_PacketDelivery starts a listener via the UDP seam, sends a
// valid CoAP/CoIoT message from a separate UDP socket, and asserts the handler
// fires with the correct device ID and sensor data.
func TestReceiveLoop_PacketDelivery(t *testing.T) {
	srvConn, srvAddr := openLocalUDPPair(t)
	// The listener will own and close srvConn.

	listener := seamedListener(srvConn)

	var (
		mu           sync.Mutex
		receivedID   string
		receivedStat *CoIoTStatus
		done         = make(chan struct{})
	)
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {
		mu.Lock()
		defer mu.Unlock()
		if receivedID == "" { // capture first delivery only
			receivedID = deviceID
			receivedStat = status
			close(done)
		}
	})

	if err := listener.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = listener.Stop() })

	// Build a valid CoAP message with GlobalDevID option + JSON sensor payload.
	options := map[int][]byte{
		optionURIPath:     []byte("cit"),
		optionGlobalDevID: []byte("SHSW-PM#AABBCC112233#2"),
	}
	payload := []byte(`{"G":[[0,1101,1],[0,4101,42.5]]}`)
	msg := buildCoAPMessage(0, codeStatus, options, payload)

	// Send the packet from a separate socket to the listener's bound address.
	sender, err := net.DialUDP("udp4", nil, srvAddr)
	if err != nil {
		t.Fatalf("dial sender: %v", err)
	}
	defer sender.Close()

	if _, err := sender.Write(msg); err != nil {
		t.Fatalf("send: %v", err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for handler to fire")
	}

	mu.Lock()
	defer mu.Unlock()

	wantID := "shsw-pm-aabbcc112233"
	if receivedID != wantID {
		t.Errorf("DeviceID = %q, want %q", receivedID, wantID)
	}
	if receivedStat == nil {
		t.Fatal("receivedStat is nil")
	}
	if receivedStat.DeviceType != "SHSW-PM" {
		t.Errorf("DeviceType = %q, want SHSW-PM", receivedStat.DeviceType)
	}
	if v, ok := receivedStat.Sensors["0_4101"]; !ok || v != 42.5 {
		t.Errorf("sensor 0_4101 = %v, want 42.5", v)
	}
}

// TestReceiveLoop_StopUnblocksLoop verifies that Stop causes the receive loop
// goroutine to exit cleanly (no goroutine leak).  We detect a leak by waiting
// a short time and confirming IsRunning returns false.
func TestReceiveLoop_StopUnblocksLoop(t *testing.T) {
	srvConn, _ := openLocalUDPPair(t)
	listener := seamedListener(srvConn)

	if err := listener.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Stop must complete within 3 s (the loop uses a 1 s read deadline).
	stopDone := make(chan error, 1)
	go func() { stopDone <- listener.Stop() }()

	select {
	case err := <-stopDone:
		if err != nil {
			t.Errorf("Stop error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Stop did not return within 3 s")
	}

	if listener.IsRunning() {
		t.Error("IsRunning should be false after Stop")
	}
}

// ---------------------------------------------------------------------------
// Start error branches: listenFn failure and SetReadBuffer failure
// ---------------------------------------------------------------------------

// TestStart_ListenFnError exercises the branch where the injected listenFn
// returns an error (simulating a failed multicast join).
func TestStart_ListenFnError(t *testing.T) {
	listener := NewCoIoTListener()
	listener.listenFn = func(_ *net.UDPAddr) (*net.UDPConn, error) {
		return nil, errors.New("permission denied")
	}

	err := listener.Start()
	if err == nil {
		t.Fatal("expected Start to return an error when listenFn fails")
	}
	if !strings.Contains(err.Error(), "multicast group") {
		t.Errorf("error should mention 'multicast group', got: %v", err)
	}
}

// TestStart_ResolveError exercises the branch where the multicast address is
// syntactically invalid so net.ResolveUDPAddr returns an error.
func TestStart_ResolveError(t *testing.T) {
	listener := NewCoIoTListener(WithCoIoTMulticastAddr("not-a-valid:::addr"))

	err := listener.Start()
	if err == nil {
		t.Fatal("expected Start to return an error for invalid multicast address")
	}
	if !strings.Contains(err.Error(), "resolve multicast address") {
		t.Errorf("error should mention 'resolve multicast address', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// receiveLoop: non-timeout network error branch
// ---------------------------------------------------------------------------

// TestReceiveLoop_NonTimeoutError covers the non-timeout, non-stop network
// error path inside receiveLoop.  We pre-close the underlying connection
// immediately after Start so that ReadFromUDP returns a non-timeout error,
// then Stop the listener to drain the goroutine cleanly.
func TestReceiveLoop_NonTimeoutError(t *testing.T) {
	srvConn, _ := openLocalUDPPair(t)
	listener := seamedListener(srvConn)

	if err := listener.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Close the underlying connection directly — this causes ReadFromUDP to
	// return a non-timeout error on the next iteration without closing stopCh,
	// exercising the "Log error and continue" branch.
	srvConn.Close()

	// Give the loop a moment to hit the error path.
	time.Sleep(50 * time.Millisecond)

	// Stop the listener; it will see the connection is already closed and
	// return an error from conn.Close() — that is acceptable.
	_ = listener.Stop()
}

// ---------------------------------------------------------------------------
// settings.go — AddScheduleRule: SetScheduleRules write-back error branch
// ---------------------------------------------------------------------------

// TestAddScheduleRule_WriteError covers the branch where GetRelaySettings
// succeeds but the follow-up SetScheduleRules write fails.
func TestAddScheduleRule_WriteError(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)

	// Read succeeds with existing rules.
	mock.OnCallJSON("/settings/relay/0", `{"schedule_rules":["0700-0123456-0-on"]}`)
	// Write fails.
	mock.OnPathContains("/settings/relay/0?schedule_rules", nil, errors.New("write failed"))

	err := device.AddScheduleRule(context.Background(), 0, "2200-0123456-0-off")
	if err == nil {
		t.Fatal("expected error when SetScheduleRules fails")
	}
	if !strings.Contains(err.Error(), "add schedule rule") {
		t.Errorf("error should mention 'add schedule rule', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// settings.go — SetEMeterConfig: UnderPower branches
// ---------------------------------------------------------------------------

// TestSetEMeterConfig_UnderPowerFields exercises the UnderPowerURL and
// UnderPowerURLThreshold conditional branches, which the existing happy-path
// test leaves untouched.
func TestSetEMeterConfig_UnderPowerFields(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)
	mock.OnPathContains("/settings/emeter/0?", nil, nil)

	cfg := EMeterSettings{
		UnderPowerURL:          "http://hook/under",
		UnderPowerURLThreshold: 100,
	}
	if err := device.SetEMeterConfig(context.Background(), 0, cfg); err != nil {
		t.Fatalf("SetEMeterConfig with under-power fields: %v", err)
	}
	calls := mock.Calls()
	if len(calls) == 0 {
		t.Fatal("expected a call to /settings/emeter/0")
	}
	q := calls[len(calls)-1].Method
	if !strings.Contains(q, "under_power_url") {
		t.Errorf("under_power_url not in request: %s", q)
	}
	if !strings.Contains(q, "under_power_url_threshold") {
		t.Errorf("under_power_url_threshold not in request: %s", q)
	}
}

// ---------------------------------------------------------------------------
// settings.go — SetRollerConfig: BtnType / SafetyMode / SafetyAction branches
// ---------------------------------------------------------------------------

// TestSetRollerConfig_OptionalFields exercises the BtnType, SafetyMode, and
// SafetyAction conditional branches that the existing happy-path test skips.
func TestSetRollerConfig_OptionalFields(t *testing.T) {
	mock := testutil.NewMockTransport()
	defer mock.ClearMatchers()
	device := NewDevice(mock)
	mock.OnPathContains("/settings/roller/0?", nil, nil)

	cfg := &RollerConfig{
		BtnType:      "momentary",
		SafetyMode:   "while_opening",
		SafetyAction: "stop",
	}
	if err := device.SetRollerConfig(context.Background(), 0, cfg); err != nil {
		t.Fatalf("SetRollerConfig with optional fields: %v", err)
	}
	calls := mock.Calls()
	if len(calls) == 0 {
		t.Fatal("expected a call to /settings/roller/0")
	}
	q := calls[len(calls)-1].Method
	if !strings.Contains(q, "btn_type") {
		t.Errorf("btn_type not in request: %s", q)
	}
	if !strings.Contains(q, "safety_mode") {
		t.Errorf("safety_mode not in request: %s", q)
	}
	if !strings.Contains(q, "safety_action") {
		t.Errorf("safety_action not in request: %s", q)
	}
}
