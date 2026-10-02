package reprovision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/backup"
	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/types"
)

// testBackup builds a backup of the given generation whose station network is
// the home network on DHCP.
func testBackup(generation int) *backup.Backup {
	return &backup.Backup{
		Version: 1,
		DeviceInfo: &backup.DeviceInfo{
			Model:      fakeGen1Model,
			Generation: generation,
			Version:    "20210101-000000/v1.0",
			MAC:        fakeMAC,
		},
		WiFi: json.RawMessage(`{"sta":{"ssid":"` + homeSSID + `","enable":true}}`),
	}
}

// stepRecorder collects OnStep calls.
type stepRecorder struct {
	mu    sync.Mutex
	steps []string
}

func (s *stepRecorder) record(step string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.steps = append(s.steps, step)
}

func (s *stepRecorder) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.steps...)
}

func TestRestore_RequiredOptions(t *testing.T) {
	t.Parallel()
	if _, err := Restore(context.Background(), &RestoreOptions{Backup: testBackup(1)}); !errors.Is(err, types.ErrInvalidParam) {
		t.Errorf("missing APSSID: err = %v, want ErrInvalidParam", err)
	}
	if _, err := Restore(context.Background(), &RestoreOptions{APSSID: fakeAPSSID}); !errors.Is(err, types.ErrInvalidParam) {
		t.Errorf("missing Backup: err = %v, want ErrInvalidParam", err)
	}
}

func TestRestore_NoPassphraseStopsBeforeHop(t *testing.T) {
	t.Parallel()
	s := &fakeScanner{current: &discovery.WiFiNetwork{SSID: homeSSID}}
	_, err := Restore(context.Background(), &RestoreOptions{APSSID: fakeAPSSID, Backup: testBackup(2), Scanner: s})
	if !errors.Is(err, ErrNoPassphrase) {
		t.Fatalf("err = %v, want ErrNoPassphrase", err)
	}
	if calls := s.connectCalls(); len(calls) != 0 {
		t.Errorf("connects = %v, want none before the passphrase is known", calls)
	}
}

func TestRestore_EndToEnd(t *testing.T) {
	t.Parallel()
	for _, gen := range []int{1, 2} {
		t.Run(fmt.Sprintf("gen%d", gen), func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, gen)
			s := homeScanner()
			r := testRunner(t, s, d.addr())
			steps := &stepRecorder{}
			r.onStep = steps.record

			res, err := r.restore(context.Background(), &RestoreOptions{
				APSSID:                 fakeAPSSID,
				Backup:                 testBackup(gen),
				Network:                Network{StaticIP: d.addr(), Gateway: "192.0.2.1", Netmask: "255.255.255.0"},
				Name:                   "Hall",
				AllowFirmwareDowngrade: true,
			})
			if err != nil {
				t.Fatalf("restore: %v", err)
			}
			if res.Address != d.addr() || res.MAC != fakeMAC || res.Restore == nil || res.SeenVia != viaProbe {
				t.Errorf("result = %+v, want address %q, MAC %q and a restore result", res, d.addr(), fakeMAC)
			}
			assertReturnedHome(t, s, fakeAPSSID)
			if gen == 1 {
				if d.hits("/settings/sta") == 0 || d.hits("/reboot") == 0 {
					t.Errorf("paths = %v, want a station write and a reboot", d.paths)
				}
			} else if !d.called("WiFi.SetConfig") || !d.called("Shelly.Reboot") || !d.called("Sys.SetConfig") {
				t.Errorf("methods = %v, want WiFi.SetConfig, Shelly.Reboot and Sys.SetConfig", d.rpcMethods)
			}
			got := steps.all()
			for _, want := range []string{
				"joining access point " + fakeAPSSID, "confirming the device identity", "writing WiFi settings",
				"returning to the home network", "waiting for the device on the LAN", "restoring the full configuration",
			} {
				if !slices.Contains(got, want) {
					t.Errorf("steps = %v, missing %q", got, want)
				}
			}
		})
	}
}

func TestRestore_ForeignBackupRefusedUnlessAllowed(t *testing.T) {
	t.Parallel()
	foreign := func(gen int) *backup.Backup {
		b := testBackup(gen)
		b.DeviceInfo.MAC = "112233445566"
		return b
	}
	for _, gen := range []int{1, 2} {
		t.Run(fmt.Sprintf("gen%d refused", gen), func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, gen)
			s := homeScanner()
			r := testRunner(t, s, d.addr())
			_, err := r.restore(context.Background(), &RestoreOptions{
				APSSID: fakeAPSSID, Backup: foreign(gen), AllowFirmwareDowngrade: true,
			})
			if !errors.Is(err, ErrIdentityMismatch) || !strings.Contains(err.Error(), "AllowForeignBackup") {
				t.Fatalf("err = %v, want ErrIdentityMismatch naming AllowForeignBackup", err)
			}
			if w := d.writes(); len(w) != 0 {
				t.Errorf("device received writes %v for a foreign backup", w)
			}
			assertReturnedHome(t, s, fakeAPSSID)
		})
		t.Run(fmt.Sprintf("gen%d allowed", gen), func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, gen)
			r := testRunner(t, homeScanner(), d.addr())
			res, err := r.restore(context.Background(), &RestoreOptions{
				APSSID:                 fakeAPSSID,
				Backup:                 foreign(gen),
				Network:                Network{StaticIP: d.addr(), Gateway: "192.0.2.1", Netmask: "255.255.255.0"},
				AllowFirmwareDowngrade: true,
				AllowForeignBackup:     true,
			})
			if err != nil {
				t.Fatalf("restore: %v", err)
			}
			if res.MAC != fakeMAC {
				t.Errorf("res.MAC = %q, want the device's %q", res.MAC, fakeMAC)
			}
			if len(d.writes()) == 0 {
				t.Error("device received no writes although the foreign backup was allowed")
			}
		})
	}
}

func TestCheckForeignBackup_NoBackupMACPasses(t *testing.T) {
	t.Parallel()
	if err := checkForeignBackup(fakeAPSSID, "", fakeMAC, false); err != nil {
		t.Errorf("err = %v, want nil for a backup without a MAC", err)
	}
}

func TestRestore_IdentityMismatchStopsBeforeAnyWrite(t *testing.T) {
	t.Parallel()
	for _, gen := range []int{1, 2} {
		t.Run(fmt.Sprintf("gen%d", gen), func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, gen)
			s := homeScanner()
			r := testRunner(t, s, d.addr())

			_, err := r.restore(context.Background(), &RestoreOptions{
				APSSID: otherAPSSID, Backup: testBackup(gen), AllowFirmwareDowngrade: true,
			})
			if !errors.Is(err, ErrIdentityMismatch) {
				t.Fatalf("err = %v, want ErrIdentityMismatch", err)
			}
			if w := d.writes(); len(w) != 0 {
				t.Errorf("device received writes %v after an identity mismatch", w)
			}
			assertReturnedHome(t, s, otherAPSSID)
		})
	}
}

func TestRestore_APNeverReadyReturnsHome(t *testing.T) {
	t.Parallel()
	s := homeScanner()
	r := testRunner(t, s, refusingAddr(t))
	_, err := r.restore(context.Background(), &RestoreOptions{APSSID: fakeAPSSID, Backup: testBackup(2)})
	if err == nil || !strings.Contains(err.Error(), "restore at AP") {
		t.Fatalf("err = %v, want a failed restore at the AP", err)
	}
	assertReturnedHome(t, s, fakeAPSSID)
}

func TestRestore_ConnectFailure(t *testing.T) {
	t.Parallel()
	s := homeScanner()
	s.connectErr = func(ssid string) error {
		if ssid == fakeAPSSID {
			return errors.New("association failed")
		}
		return nil
	}
	r := testRunner(t, s, refusingAddr(t))
	_, err := r.restore(context.Background(), &RestoreOptions{APSSID: fakeAPSSID, Backup: testBackup(2)})
	if !errors.Is(err, ErrAPUnreachable) || !strings.Contains(err.Error(), "AP hop for") {
		t.Fatalf("err = %v, want an ErrAPUnreachable hop failure", err)
	}
	assertReturnedHome(t, s, fakeAPSSID)
}

func TestRestore_UnstableGen1CancelledReturnsHome(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 1)
	d.uptime = 1
	s := homeScanner()
	r := testRunner(t, s, d.addr())
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := r.restore(ctx, &RestoreOptions{APSSID: fakeAPSSID, Backup: testBackup(1), AllowFirmwareDowngrade: true})
	if !errors.Is(err, ErrUnstable) {
		t.Fatalf("err = %v, want ErrUnstable", err)
	}
	if d.hits("/settings/sta") != 0 {
		t.Error("the station settings were written to an unstable device")
	}
	assertReturnedHome(t, s, fakeAPSSID)
}

func TestRestore_NotRejoined(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 2)
	r := testRunner(t, homeScanner(), d.addr())
	r.rejoinTimeout = 100 * time.Millisecond
	r.probeTimeout = 20 * time.Millisecond

	res, err := r.restore(context.Background(), &RestoreOptions{
		APSSID: fakeAPSSID, Backup: testBackup(2),
		Network: Network{StaticIP: refusingAddr(t), Gateway: "192.0.2.1", Netmask: "255.255.255.0"},
	})
	if !errors.Is(err, ErrNotRejoined) {
		t.Fatalf("err = %v, want ErrNotRejoined", err)
	}
	if res.Address != "" || res.MAC != fakeMAC {
		t.Errorf("result = %+v, want no address and the MAC read at the AP", res)
	}
}

func TestRestore_NoRoute(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 1)
	r := testRunner(t, homeScanner(), d.addr())
	r.rejoinTimeout = 200 * time.Millisecond
	r.probeTimeout = 20 * time.Millisecond
	seen := refusingAddr(t)
	r.scanPresence = func(context.Context, string, string, bool, time.Duration) (addr, via string, err error) {
		return seen, viaCoIoT, nil
	}

	res, err := r.restore(context.Background(), &RestoreOptions{
		APSSID: fakeAPSSID, Backup: testBackup(1), AllowFirmwareDowngrade: true,
	})
	if !errors.Is(err, ErrNoRoute) {
		t.Fatalf("err = %v, want ErrNoRoute", err)
	}
	if res.Address != seen || res.Reachable || res.SeenVia != viaCoIoT {
		t.Errorf("result = %+v, want address %s seen via %s with Reachable false", res, seen, viaCoIoT)
	}
}

func TestRestore_LANRestoreFailureKeepsAddress(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 2)
	r := testRunner(t, homeScanner(), d.addr())
	r.lanSettleDelay = time.Minute
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r.onStep = func(step string) {
		if step == "restoring the full configuration" {
			cancel()
		}
	}

	res, err := r.restore(ctx, &RestoreOptions{
		APSSID: fakeAPSSID, Backup: testBackup(2),
		Network: Network{StaticIP: d.addr(), Gateway: "192.0.2.1", Netmask: "255.255.255.0"},
	})
	if err == nil || !strings.Contains(err.Error(), "full configuration restore failed") {
		t.Fatalf("err = %v, want the LAN restore failure", err)
	}
	if res.Address != d.addr() || !res.Reachable {
		t.Errorf("result = %+v, want reachable address %q recorded although the LAN restore failed", res, d.addr())
	}
}

func TestRestoreGen2_NetworkOnlyWritesOnlyWiFi(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 2)
	r := testRunner(t, nil, d.addr())
	bkp := testBackup(2)
	bkp.Schedules = json.RawMessage(`{"jobs":[{"id":1,"enable":true,"timespec":"0 0 * * * *","calls":[]}]}`)

	_, err := r.restoreDevice(context.Background(), d.addr(), "", 2, &RestoreOptions{Backup: bkp, Name: "Hall"},
		passOptions{override: &Network{SSID: homeSSID, Password: homePass}, networkOnly: true})
	if err != nil {
		t.Fatalf("restoreDevice: %v", err)
	}
	if !d.called("WiFi.SetConfig") {
		t.Errorf("methods = %v, want WiFi.SetConfig", d.rpcMethods)
	}
	for _, m := range []string{"Schedule.Create", "Schedule.DeleteAll", "Sys.SetConfig"} {
		if d.called(m) {
			t.Errorf("network-only pass called %s", m)
		}
	}

	// The same backup without networkOnly does restore the schedules, so the
	// assertion above is not vacuous.
	full := newFakeDevice(t, 2)
	if _, err := r.restoreDevice(context.Background(), full.addr(), "", 2, &RestoreOptions{Backup: bkp},
		passOptions{skipNetwork: true}); err != nil {
		t.Fatalf("full restoreDevice: %v", err)
	}
	if !full.called("Schedule.Create") {
		t.Errorf("full pass methods = %v, want Schedule.Create", full.rpcMethods)
	}
}

func TestRestoreGen2_NameFailureIsWarning(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 2)
	d.failMethod = "Sys.SetConfig"
	r := testRunner(t, nil, d.addr())
	res, err := r.restoreDevice(context.Background(), d.addr(), "", 2,
		&RestoreOptions{Backup: testBackup(2), Name: "Hall"}, passOptions{skipNetwork: true})
	if err != nil {
		t.Fatalf("restoreDevice: %v", err)
	}
	if !slices.ContainsFunc(res.Warnings, func(w string) bool { return strings.Contains(w, "set device name") }) {
		t.Errorf("warnings = %v, want a name warning", res.Warnings)
	}
}

func TestRestoreGen2_Errors(t *testing.T) {
	t.Parallel()
	bad := testBackup(2)
	bad.WiFi = json.RawMessage(`not json`)
	r := testRunner(t, nil, refusingAddr(t))
	if _, err := r.restoreDevice(context.Background(), r.apAddr, "", 2, &RestoreOptions{Backup: bad},
		passOptions{override: &Network{SSID: homeSSID}}); err == nil {
		t.Error("expected an error for an unparseable WiFi blob")
	}
	if _, err := r.restoreDevice(context.Background(), r.apAddr, "", 2, &RestoreOptions{Backup: testBackup(2)},
		passOptions{skipNetwork: true}); err == nil {
		t.Error("expected an error for an unreachable device")
	}

	d := newFakeDevice(t, 2)
	d.failMethod = "Shelly.GetDeviceInfo"
	if _, err := r.restoreDevice(context.Background(), d.addr(), "", 2, &RestoreOptions{Backup: testBackup(2)},
		passOptions{skipNetwork: true}); err == nil {
		t.Error("expected an error when the device identity read fails")
	}
}

func TestRestoreGen1_Error(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 1)
	bad := testBackup(1)
	bad.Config = json.RawMessage(`not json`)
	r := testRunner(t, nil, d.addr())
	if _, err := r.restoreDevice(context.Background(), d.addr(), "", 1, &RestoreOptions{Backup: bad},
		passOptions{override: &Network{SSID: homeSSID}, networkOnly: true}); err == nil {
		t.Error("expected an error for an unparseable Gen1 config")
	}
}

func TestFullRestoreOnLAN_CancelledContext(t *testing.T) {
	t.Parallel()
	r := testRunner(t, nil, "")
	r.lanSettleDelay = time.Minute
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.fullRestoreOnLAN(ctx, "192.0.2.10", "", 1, &RestoreOptions{Backup: testBackup(1)}); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestApplyGen2WiFiOverride(t *testing.T) {
	t.Parallel()
	blob := json.RawMessage(`{"ap":{"enable":false},"sta":{"ssid":"Old","ipv4mode":"dhcp"}}`)

	out, err := applyGen2WiFiOverride(blob, &Network{
		SSID: homeSSID, Password: homePass, StaticIP: "192.0.2.10", Gateway: "192.0.2.1",
		Netmask: "255.255.255.0", DNS: "192.0.2.53",
	})
	if err != nil {
		t.Fatalf("applyGen2WiFiOverride: %v", err)
	}
	var got map[string]map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := map[string]any{
		"ssid": homeSSID, "pass": homePass, "enable": true, "ipv4mode": "static",
		"ip": "192.0.2.10", "gw": "192.0.2.1", "netmask": "255.255.255.0", "nameserver": "192.0.2.53",
	}
	for k, v := range want {
		if got["sta"][k] != v {
			t.Errorf("sta[%s] = %v, want %v", k, got["sta"][k], v)
		}
	}
	if _, ok := got["ap"]; !ok {
		t.Error("the ap block was dropped")
	}

	out, err = applyGen2WiFiOverride(nil, &Network{})
	if err != nil || string(out) != `{"sta":{"enable":true}}` {
		t.Errorf("empty override = %s, %v; want only enable", out, err)
	}
	if _, err := applyGen2WiFiOverride(json.RawMessage(`[`), &Network{}); err == nil {
		t.Error("expected an error for an unparseable blob")
	}
}

func TestApplyGen2WiFiOverride_Open(t *testing.T) {
	t.Parallel()
	decodeSta := func(t *testing.T, out json.RawMessage) map[string]any {
		t.Helper()
		var got map[string]map[string]any
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return got["sta"]
	}
	out, err := applyGen2WiFiOverride(json.RawMessage(`{"sta":{"ssid":"Old","pass":"x"}}`),
		&Network{SSID: "Guest", Open: true})
	if err != nil {
		t.Fatalf("applyGen2WiFiOverride: %v", err)
	}
	if sta := decodeSta(t, out); sta["ssid"] != "Guest" || sta["pass"] != "" || hasIsOpen(sta) {
		t.Errorf("sta = %v, want Guest with an empty pass and no is_open", sta)
	}

	out, err = applyGen2WiFiOverride(json.RawMessage(`{"sta":{"ssid":"Guest","is_open":true}}`),
		&Network{Password: "secret"})
	if err != nil {
		t.Fatalf("applyGen2WiFiOverride: %v", err)
	}
	if sta := decodeSta(t, out); sta["pass"] != "secret" || hasIsOpen(sta) {
		t.Errorf("sta = %v, want the passphrase and no is_open", sta)
	}
}

func TestRestore_OpenNetwork(t *testing.T) {
	t.Parallel()
	for _, gen := range []int{1, 2} {
		t.Run(fmt.Sprintf("gen%d", gen), func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, gen)
			// No stored passphrase: an open network must not ask for one.
			s := &fakeScanner{current: &discovery.WiFiNetwork{SSID: homeSSID}}
			r := testRunner(t, s, d.addr())
			res, err := r.restore(context.Background(), &RestoreOptions{
				APSSID: fakeAPSSID, Backup: testBackup(gen), AllowFirmwareDowngrade: true,
				Network: Network{
					Open: true, StaticIP: d.addr(), Gateway: "192.0.2.1", Netmask: "255.255.255.0",
				},
			})
			if err != nil {
				t.Fatalf("restore: %v", err)
			}
			if res.SeenVia != viaProbe {
				t.Errorf("SeenVia = %q, want %q", res.SeenVia, viaProbe)
			}
			if gen == 1 {
				if q := d.args("/settings/sta"); !strings.Contains(q, "key=&") {
					t.Errorf("station write %q, want an empty key", q)
				}
			} else if sta := writtenSta(t, d); sta["pass"] != "" || hasIsOpen(sta) {
				t.Errorf("written sta = %v, want an empty pass and no is_open", sta)
			}
		})
	}
}

// hasIsOpen reports whether a written station carries is_open.
func hasIsOpen(sta map[string]any) bool {
	_, ok := sta["is_open"]
	return ok
}

// writtenSta decodes the station of the last WiFi.SetConfig the fake received.
func writtenSta(t *testing.T, d *fakeDevice) map[string]any {
	t.Helper()
	var params struct {
		Config struct {
			Sta map[string]any `json:"sta"`
		} `json:"config"`
	}
	if err := json.Unmarshal([]byte(d.args("WiFi.SetConfig")), &params); err != nil {
		t.Fatalf("decode WiFi.SetConfig params: %v", err)
	}
	return params.Config.Sta
}

func TestRestore_OpenBackupStation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		override Network
		wantPass string
	}{
		{name: "no override rejoins open", wantPass: ""},
		{name: "a passphrase override joins secured", override: Network{Password: "secret"}, wantPass: "secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, 2)
			s := &fakeScanner{current: &discovery.WiFiNetwork{SSID: homeSSID}}
			r := testRunner(t, s, d.addr())
			bkp := testBackup(2)
			bkp.WiFi = json.RawMessage(`{"sta":{"ssid":"Guest","enable":true,"is_open":true,"ipv4mode":"static",` +
				`"ip":"` + d.addr() + `","gw":"192.0.2.1","netmask":"255.255.255.0"}}`)
			_, err := r.restore(context.Background(), &RestoreOptions{
				APSSID: fakeAPSSID, Backup: bkp, Network: tt.override,
			})
			if err != nil {
				t.Fatalf("restore: %v", err)
			}
			sta := writtenSta(t, d)
			if sta["ssid"] != "Guest" || sta["pass"] != tt.wantPass || hasIsOpen(sta) {
				t.Errorf("written sta = %v, want Guest with pass %q and no is_open", sta, tt.wantPass)
			}
		})
	}
}

func TestRestore_OpenWithPasswordRefused(t *testing.T) {
	t.Parallel()
	s := homeScanner()
	_, err := Restore(context.Background(), &RestoreOptions{
		APSSID: fakeAPSSID, Backup: testBackup(2), Scanner: s,
		Network: Network{SSID: homeSSID, Open: true, Password: "x"},
	})
	if !errors.Is(err, types.ErrInvalidParam) {
		t.Fatalf("err = %v, want ErrInvalidParam", err)
	}
	if calls := s.connectCalls(); len(calls) != 0 {
		t.Errorf("connects = %v, want none", calls)
	}
}

func TestNetworkFromBackup(t *testing.T) {
	t.Parallel()
	gen1Static := `{"ssid":"Home","key":"k1","ipv4_method":"static","ip":"192.0.2.10","gw":"192.0.2.1",` +
		`"mask":"255.255.255.0","dns":"192.0.2.53"}`
	tests := []struct {
		name string
		bkp  *backup.Backup
		want Network
	}{
		{
			name: "gen1 static blob",
			bkp:  &backup.Backup{WiFi: json.RawMessage(`{"sta":` + gen1Static + `}`)},
			want: Network{
				SSID: "Home", Password: "k1", StaticIP: "192.0.2.10", Gateway: "192.0.2.1",
				Netmask: "255.255.255.0", DNS: "192.0.2.53",
			},
		},
		{
			name: "gen2 static blob",
			bkp: &backup.Backup{WiFi: json.RawMessage(`{"sta":{"ssid":"Home","ipv4mode":"static","ip":"192.0.2.11",` +
				`"gw":"192.0.2.1","netmask":"255.255.0.0","nameserver":"1.1.1.1"}}`)},
			want: Network{SSID: "Home", StaticIP: "192.0.2.11", Gateway: "192.0.2.1", Netmask: "255.255.0.0", DNS: "1.1.1.1"},
		},
		{
			name: "dhcp drops stale addressing",
			bkp:  &backup.Backup{WiFi: json.RawMessage(`{"sta":{"ssid":"Home","ipv4mode":"dhcp","ip":"192.0.2.11"}}`)},
			want: Network{SSID: "Home"},
		},
		{
			name: "gen1 config fallback",
			bkp: &backup.Backup{
				DeviceInfo: &backup.DeviceInfo{Generation: 1},
				Config:     json.RawMessage(`{"wifi_sta":` + gen1Static + `}`),
			},
			want: Network{
				SSID: "Home", Password: "k1", StaticIP: "192.0.2.10", Gateway: "192.0.2.1",
				Netmask: "255.255.255.0", DNS: "192.0.2.53",
			},
		},
		{
			name: "gen2 config is not read",
			bkp: &backup.Backup{
				DeviceInfo: &backup.DeviceInfo{Generation: 2},
				Config:     json.RawMessage(`{"wifi_sta":` + gen1Static + `}`),
			},
		},
		{
			name: "gen2 open station",
			bkp:  &backup.Backup{WiFi: json.RawMessage(`{"sta":{"ssid":"Guest","is_open":true,"ipv4mode":"dhcp"}}`)},
			want: Network{SSID: "Guest", Open: true},
		},
		{name: "empty ssid blob", bkp: &backup.Backup{WiFi: json.RawMessage(`{"sta":{"ssid":""}}`)}},
		{
			name: "gen1 blob with empty ssid falls back to config",
			bkp: &backup.Backup{
				DeviceInfo: &backup.DeviceInfo{Generation: 1},
				WiFi:       json.RawMessage(`{"sta":{"ssid":"","ipv4_method":"static","ip":"192.0.2.99"}}`),
				Config:     json.RawMessage(`{"wifi_sta":` + gen1Static + `}`),
			},
			want: Network{
				SSID: "Home", Password: "k1", StaticIP: "192.0.2.10", Gateway: "192.0.2.1",
				Netmask: "255.255.255.0", DNS: "192.0.2.53",
			},
		},
		{name: "invalid blob", bkp: &backup.Backup{WiFi: json.RawMessage(`nope`)}},
		{name: "nothing", bkp: &backup.Backup{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NetworkFromBackup(tt.bkp); got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBackupDevice(t *testing.T) {
	t.Parallel()
	if gen, model, fw, mac := backupDevice(&backup.Backup{}); gen != 2 || model != "" || fw != "" || mac != "" {
		t.Errorf("no device info = (%d, %q, %q, %q), want Gen2 and empty fields", gen, model, fw, mac)
	}
	if gen, model, fw, mac := backupDevice(testBackup(1)); gen != 1 || model != fakeGen1Model ||
		fw != "20210101-000000/v1.0" || mac != fakeMAC {
		t.Errorf("got (%d, %q, %q, %q)", gen, model, fw, mac)
	}
}

func TestRebootAtAP(t *testing.T) {
	t.Parallel()
	for _, gen := range []int{1, 2} {
		t.Run(fmt.Sprintf("gen%d", gen), func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, gen)
			testRunner(t, nil, d.addr()).rebootAtAP(context.Background(), gen)
			if gen == 1 && d.hits("/reboot") != 1 {
				t.Errorf("reboot hits = %d, want 1", d.hits("/reboot"))
			}
			if gen == 2 && !d.called("Shelly.Reboot") {
				t.Error("Shelly.Reboot was not called")
			}
		})
	}
	t.Run("errors are logged", func(t *testing.T) {
		t.Parallel()
		d := newFakeDevice(t, 2)
		d.rebootErr = true
		testRunner(t, nil, d.addr()).rebootAtAP(context.Background(), 2)
		testRunner(t, nil, refusingAddr(t)).rebootAtAP(context.Background(), 2)
	})
}

func TestMACSuffixFromAPSSID(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, ssid, want string }{
		{"gen1 six-hex", "ShellyBulbDuo-6645B6", "6645B6"},
		{"gen2 full mac", "shellyplus2pm-aabbccddeeff", "AABBCCDDEEFF"},
		{"lowercase normalized", "ShellyBulbDuo-6645b6", "6645B6"},
		{"multiple dashes keeps last token", "Shelly-Bulb-DDEEFF", "DDEEFF"},
		{"no dash", "MyCustomAP", ""},
		{"trailing dash", "ShellyBulbDuo-", ""},
		{"non-hex suffix", "ShellyBulbDuo-LIVING", ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := macSuffixFromAPSSID(tc.ssid); got != tc.want {
				t.Errorf("macSuffixFromAPSSID(%q) = %q, want %q", tc.ssid, got, tc.want)
			}
		})
	}
}

func TestEvaluateAPIdentity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, mac, ssid     string
		wantMatch, wantSkip bool
	}{
		{"suffix match", "AABBCCDDEEFF", "ShellyBulbDuo-DDEEFF", true, false},
		{"full-mac ssid match", "AA:BB:CC:DD:EE:FF", "shellyplus2pm-AABBCCDDEEFF", true, false},
		{"genuine mismatch", "AABBCCDDEEFF", "ShellyBulbDuo-6645B6", false, false},
		{"no ssid suffix is skipped", "AABBCCDDEEFF", "CustomAP", false, true},
		{"unparseable mac is skipped", "not-a-mac", "ShellyBulbDuo-DDEEFF", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			matched, skip, _, _ := evaluateAPIdentity(tc.mac, tc.ssid)
			if matched != tc.wantMatch || skip != tc.wantSkip {
				t.Errorf("evaluateAPIdentity(%q, %q) = (%v, %v), want (%v, %v)",
					tc.mac, tc.ssid, matched, skip, tc.wantMatch, tc.wantSkip)
			}
		})
	}
}

func TestConfirmAPDeviceIdentity(t *testing.T) {
	t.Parallel()
	for _, gen := range []int{1, 2} {
		t.Run(fmt.Sprintf("gen%d match returns the MAC", gen), func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, gen)
			mac, err := testRunner(t, nil, d.addr()).confirmAPDeviceIdentity(context.Background(), gen, fakeAPSSID)
			if err != nil || mac != fakeMAC {
				t.Errorf("got (%q, %v), want (%q, nil)", mac, err, fakeMAC)
			}
		})
		t.Run(fmt.Sprintf("gen%d mismatch refused", gen), func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, gen)
			_, err := testRunner(t, nil, d.addr()).confirmAPDeviceIdentity(context.Background(), gen, otherAPSSID)
			if !errors.Is(err, ErrIdentityMismatch) || !strings.Contains(err.Error(), "nothing was written") {
				t.Errorf("err = %v, want ErrIdentityMismatch", err)
			}
		})
	}
	t.Run("no-suffix SSID still reports the MAC", func(t *testing.T) {
		t.Parallel()
		d := newFakeDevice(t, 1)
		mac, err := testRunner(t, nil, d.addr()).confirmAPDeviceIdentity(context.Background(), 1, "MyHouseAP")
		if err != nil || mac != fakeMAC {
			t.Errorf("got (%q, %v), want (%q, nil)", mac, err, fakeMAC)
		}
	})
	t.Run("no-suffix SSID with an unreadable device is allowed", func(t *testing.T) {
		t.Parallel()
		mac, err := testRunner(t, nil, refusingAddr(t)).confirmAPDeviceIdentity(context.Background(), 1, "MyHouseAP")
		if err != nil || mac != "" {
			t.Errorf("got (%q, %v), want (\"\", nil)", mac, err)
		}
	})
	t.Run("unreadable device with a suffix is refused", func(t *testing.T) {
		t.Parallel()
		_, err := testRunner(t, nil, refusingAddr(t)).confirmAPDeviceIdentity(context.Background(), 2, fakeAPSSID)
		if err == nil || !strings.Contains(err.Error(), "could not read the device identity") {
			t.Errorf("err = %v, want an identity read failure", err)
		}
	})
	t.Run("unparseable MAC skips the check", func(t *testing.T) {
		t.Parallel()
		r := testRunner(t, nil, "")
		if mac, err := r.checkAPIdentity(fakeAPSSID, "garbage", nil); err != nil || mac != "" {
			t.Errorf("got (%q, %v), want (\"\", nil)", mac, err)
		}
	})
}

func TestExportedEntryPoints_NilOptions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	if _, err := Restore(ctx, nil); !errors.Is(err, types.ErrInvalidParam) {
		t.Errorf("Restore(nil): err = %v, want ErrInvalidParam", err)
	}
	if _, err := Onboard(ctx, nil); !errors.Is(err, types.ErrInvalidParam) {
		t.Errorf("Onboard(nil): err = %v, want ErrInvalidParam", err)
	}
	if _, err := Inspect(ctx, nil); !errors.Is(err, types.ErrInvalidParam) {
		t.Errorf("Inspect(nil): err = %v, want ErrInvalidParam", err)
	}
}
