package reprovision

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/types"
)

func TestOnboard_RequiredOptions(t *testing.T) {
	t.Parallel()
	if _, err := Onboard(context.Background(), &OnboardOptions{}); !errors.Is(err, types.ErrInvalidParam) {
		t.Errorf("err = %v, want ErrInvalidParam", err)
	}
}

func TestOnboard_ConnectFailureReturnsHome(t *testing.T) {
	t.Parallel()
	s := homeScanner()
	s.connectErr = func(ssid string) error {
		if ssid == fakeAPSSID {
			return errors.New("association failed")
		}
		return nil
	}
	_, err := Onboard(context.Background(), &OnboardOptions{APSSID: fakeAPSSID, Scanner: s})
	if !errors.Is(err, ErrAPUnreachable) || !strings.Contains(err.Error(), "AP hop for") {
		t.Fatalf("err = %v, want an AP hop failure", err)
	}
	assertReturnedHome(t, s, fakeAPSSID)
}

func TestOnboard_NoPassphrase(t *testing.T) {
	t.Parallel()
	r := testRunner(t, &fakeScanner{currentErr: errors.New("not connected")}, "")
	if _, err := r.onboard(context.Background(), fakeAPSSID, &Network{}, 0); !errors.Is(err, ErrNoPassphrase) {
		t.Errorf("err = %v, want ErrNoPassphrase", err)
	}
}

func TestOnboard_EndToEnd(t *testing.T) {
	t.Parallel()
	for _, gen := range []int{1, 2} {
		for _, static := range []bool{false, true} {
			t.Run(fmt.Sprintf("gen%d static=%v", gen, static), func(t *testing.T) {
				t.Parallel()
				d := newFakeDevice(t, gen)
				s := homeScanner()
				r := testRunner(t, s, d.addr())
				n := Network{}
				if static {
					n = Network{StaticIP: d.addr(), Gateway: "192.0.2.1", Netmask: "255.255.255.0", DNS: "192.0.2.53"}
				} else {
					r.scanPresence = func(context.Context, string, string, bool, time.Duration) (addr, via string, err error) {
						return d.addr(), viaMDNS, nil
					}
				}

				res, err := r.onboard(context.Background(), fakeAPSSID, &n, 0)
				if err != nil {
					t.Fatalf("onboard: %v", err)
				}
				if res.Address != d.addr() || res.MAC != fakeMAC || res.Generation != gen || res.Note != "" {
					t.Errorf("result = %+v, want address %s, MAC %s, gen %d", res, d.addr(), fakeMAC, gen)
				}
				if gen == 1 && d.hits("/settings/sta") == 0 {
					t.Errorf("paths = %v, want a station write", d.paths)
				}
				if gen != 1 && !d.called("WiFi.SetConfig") {
					t.Errorf("methods = %v, want WiFi.SetConfig", d.rpcMethods)
				}
				assertReturnedHome(t, s, fakeAPSSID)
			})
		}
	}
}

func TestOnboard_NotFoundIsANote(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 2)
	r := testRunner(t, homeScanner(), d.addr())
	r.rejoinTimeout = 50 * time.Millisecond
	res, err := r.onboard(context.Background(), fakeAPSSID, &Network{}, 0)
	if err != nil {
		t.Fatalf("onboard: %v", err)
	}
	if res.Address != "" || !strings.HasPrefix(res.Note, "provisioned but") {
		t.Errorf("result = %+v, want no address and a 'provisioned but' note", res)
	}
}

func TestOnboard_PresenceOnlyStillGivesAddress(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 1)
	r := testRunner(t, homeScanner(), d.addr())
	r.rejoinTimeout = 100 * time.Millisecond
	r.probeTimeout = 20 * time.Millisecond
	seen := refusingAddr(t)
	r.scanPresence = func(context.Context, string, string, bool, time.Duration) (addr, via string, err error) {
		return seen, viaCoIoT, nil
	}
	res, err := r.onboard(context.Background(), fakeAPSSID, &Network{}, 0)
	if err != nil || res.Address != seen || res.Reachable || res.SeenVia != viaCoIoT {
		t.Errorf("got (%+v, %v), want unreachable address %s seen via %s", res, err, seen, viaCoIoT)
	}
}

func TestOnboard_OpenNetwork(t *testing.T) {
	t.Parallel()
	for _, gen := range []int{1, 2} {
		t.Run(fmt.Sprintf("gen%d", gen), func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, gen)
			s := &fakeScanner{current: &discovery.WiFiNetwork{SSID: homeSSID}}
			r := testRunner(t, s, d.addr())
			r.scanPresence = func(context.Context, string, string, bool, time.Duration) (addr, via string, err error) {
				return d.addr(), viaMDNS, nil
			}
			res, err := r.onboard(context.Background(), fakeAPSSID, &Network{SSID: "Guest", Open: true}, 0)
			if err != nil {
				t.Fatalf("onboard: %v", err)
			}
			if res.SeenVia != viaProbe || !res.Reachable {
				t.Errorf("result = %+v, want a reachable device seen via %s", res, viaProbe)
			}
			if gen == 1 {
				if q := d.args("/settings/sta"); !strings.Contains(q, "key=&") || !strings.Contains(q, "ssid=Guest") {
					t.Errorf("station write %q, want Guest with an empty key", q)
				}
			} else if sta := writtenSta(t, d); sta["ssid"] != "Guest" || sta["pass"] != "" || hasIsOpen(sta) {
				t.Errorf("written sta = %v, want Guest with an empty pass and no is_open", sta)
			}
		})
	}
}

func TestOnboard_Gen1StaticOpenNetwork(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 1)
	r := testRunner(t, &fakeScanner{current: &discovery.WiFiNetwork{SSID: homeSSID}}, d.addr())
	r.scanPresence = func(context.Context, string, string, bool, time.Duration) (addr, via string, err error) {
		return d.addr(), viaMDNS, nil
	}
	n := &Network{SSID: "Guest", Open: true, StaticIP: d.addr(), Gateway: "192.0.2.1", Netmask: "255.255.255.0"}
	if _, err := r.onboard(context.Background(), fakeAPSSID, n, 0); err != nil {
		t.Fatalf("onboard: %v", err)
	}
	q := d.args("/settings/sta")
	if !strings.Contains(q, "key=&") || !strings.Contains(q, "ipv4_method=static") || !strings.Contains(q, "ssid=Guest") {
		t.Errorf("station write %q, want a static write for Guest with an empty key", q)
	}
}

// The SSID carries no MAC suffix to check, so the identity read only supplies
// the generation and MAC; onboarding proceeds.
func TestOnboard_SSIDWithoutMACSuffix(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 2)
	const ssid = "my-custom-ap"
	s := homeScanner()
	r := testRunner(t, s, d.addr())
	r.rejoinTimeout = 50 * time.Millisecond
	res, err := r.onboard(context.Background(), ssid, &Network{}, 0)
	if err != nil {
		t.Fatalf("onboard: %v", err)
	}
	if res.MAC != fakeMAC || res.Generation != 2 {
		t.Errorf("result = %+v, want MAC %s and generation 2", res, fakeMAC)
	}
	if !d.called("WiFi.SetConfig") {
		t.Error("WiFi settings were not written")
	}
	assertReturnedHome(t, s, ssid)
}

func TestOnboard_IdentityMismatchStopsBeforeAnyWrite(t *testing.T) {
	t.Parallel()
	for _, gen := range []int{1, 2} {
		t.Run(fmt.Sprintf("gen%d", gen), func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, gen)
			s := homeScanner()
			_, err := testRunner(t, s, d.addr()).onboard(context.Background(), otherAPSSID, &Network{}, 0)
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

func TestOnboard_UnreachableDevice(t *testing.T) {
	t.Parallel()
	s := homeScanner()
	_, err := testRunner(t, s, refusingAddr(t)).onboard(context.Background(), fakeAPSSID, &Network{}, 0)
	if err == nil || !strings.Contains(err.Error(), "identify device") {
		t.Fatalf("err = %v, want an identify failure", err)
	}
	assertReturnedHome(t, s, fakeAPSSID)
}

func TestConfigureWiFiAtAP_Gen2Error(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 2)
	d.failMethod = "WiFi.SetConfig"
	err := testRunner(t, nil, d.addr()).configureWiFiAtAP(context.Background(), 2, &Network{SSID: homeSSID})
	if err == nil {
		t.Error("expected the WiFi.SetConfig error")
	}
}

func TestScanAPs(t *testing.T) {
	t.Parallel()
	s := &fakeScanner{networks: []discovery.WiFiNetwork{
		{SSID: "shellyplus1-AABBCC", Signal: -70},
		{SSID: "HomeNet", Signal: -40},
		{SSID: "ShellyBulbDuo-D12965", Signal: -60},
		{SSID: "shellyplus1-AABBCC", Signal: -50},
		{SSID: "shellyplus1-AABBCC", Signal: -80},
	}}
	aps, err := ScanAPs(context.Background(), s)
	if err != nil {
		t.Fatalf("ScanAPs: %v", err)
	}
	want := []AP{
		{SSID: "ShellyBulbDuo-D12965", MACSuffix: "D12965", Signal: -60},
		{SSID: "shellyplus1-AABBCC", MACSuffix: "AABBCC", Signal: -50},
	}
	if !slices.Equal(aps, want) {
		t.Errorf("ScanAPs = %+v, want %+v", aps, want)
	}

	if _, err := ScanAPs(context.Background(), &fakeScanner{scanErr: errors.New("busy")}); err == nil {
		t.Error("expected the scan error")
	}
}

func TestOnboard_IdentityReadFailure(t *testing.T) {
	t.Parallel()
	const noSuffix = "my-custom-ap"
	tests := []struct {
		name       string
		ssid       string
		generation int
		wantWrite  bool
	}{
		{name: "no suffix, generation supplied", ssid: noSuffix, generation: 2, wantWrite: true},
		{name: "no suffix, generation zero", ssid: noSuffix},
		{name: "suffix present", ssid: fakeAPSSID, generation: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, 2)
			d.shellyErr = true
			s := homeScanner()
			r := testRunner(t, s, d.addr())
			r.rejoinTimeout = 50 * time.Millisecond
			res, err := r.onboard(context.Background(), tt.ssid, &Network{}, tt.generation)
			if tt.wantWrite {
				if err != nil {
					t.Fatalf("onboard: %v", err)
				}
				if !d.called("WiFi.SetConfig") || res.Generation != tt.generation {
					t.Errorf("result = %+v, want WiFi written with generation %d", res, tt.generation)
				}
			} else {
				if err == nil {
					t.Fatal("expected an error when the identity read is required and fails")
				}
				if w := d.writes(); len(w) != 0 {
					t.Errorf("device received writes %v although the identity read failed", w)
				}
			}
			assertReturnedHome(t, s, tt.ssid)
		})
	}
}

func TestOnboard_DeviceGenerationWins(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 2)
	r := testRunner(t, homeScanner(), d.addr())
	r.rejoinTimeout = 50 * time.Millisecond
	res, err := r.onboard(context.Background(), fakeAPSSID, &Network{}, 1)
	if err != nil {
		t.Fatalf("onboard: %v", err)
	}
	if res.Generation != 2 || !d.called("WiFi.SetConfig") {
		t.Errorf("result = %+v, want the device's generation 2 used for the WiFi write", res)
	}
}
