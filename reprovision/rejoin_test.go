package reprovision

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
)

func fastRejoinConfig() rejoinConfig {
	return rejoinConfig{
		log:             slog.New(slog.DiscardHandler),
		mac:             "AA:BB:CC:DD:EE:FF",
		generation:      1,
		timeout:         500 * time.Millisecond,
		interval:        5 * time.Millisecond,
		presenceTimeout: 10 * time.Millisecond,
		probeTimeout:    10 * time.Millisecond,
	}
}

func neverSeen(context.Context, string, string, bool, time.Duration) (addr, via string, err error) {
	return "", "", nil
}

func noRoute(context.Context, string, string, int) error { return errors.New("no route to host") }

func TestRaceRejoin(t *testing.T) {
	t.Parallel()

	t.Run("unicast probe wins and names the reaching interface", func(t *testing.T) {
		t.Parallel()
		cfg := fastRejoinConfig()
		cfg.staticIP = "192.0.2.10"
		cfg.candidates = []string{"", "eth0"}
		cfg.scanPresence = neverSeen
		cfg.probe = func(_ context.Context, addr, iface string, _ int) error {
			if iface == "eth0" && addr == "192.0.2.10" {
				return nil
			}
			return errors.New("no route")
		}
		conf, err := raceRejoin(context.Background(), &cfg)
		if err != nil {
			t.Fatalf("raceRejoin: %v", err)
		}
		if !conf.writeable || conf.addr != "192.0.2.10" || conf.bindIface != "eth0" || conf.via != viaProbe {
			t.Errorf("conf = %+v, want a writeable probe confirmation over eth0", conf)
		}
	})

	t.Run("presence only is not writeable", func(t *testing.T) {
		t.Parallel()
		cfg := fastRejoinConfig()
		cfg.staticIP = "192.0.2.10"
		cfg.candidates = []string{""}
		cfg.scanPresence = func(context.Context, string, string, bool, time.Duration) (addr, via string, err error) {
			return "192.0.2.10", viaCoIoT, errors.New("partial sweep")
		}
		cfg.probe = noRoute
		conf, err := raceRejoin(context.Background(), &cfg)
		if err != nil {
			t.Fatalf("raceRejoin: %v", err)
		}
		if conf.writeable || conf.addr != "192.0.2.10" || conf.via != viaCoIoT {
			t.Errorf("conf = %+v, want a non-writeable CoIoT sighting", conf)
		}
	})

	t.Run("never seen is an error", func(t *testing.T) {
		t.Parallel()
		cfg := fastRejoinConfig()
		cfg.staticIP = "192.0.2.10"
		cfg.candidates = []string{""}
		cfg.scanPresence = neverSeen
		cfg.probe = noRoute
		conf, err := raceRejoin(context.Background(), &cfg)
		if !errors.Is(err, ErrNotRejoined) {
			t.Fatalf("err = %v, want ErrNotRejoined", err)
		}
		if conf.addr != "" {
			t.Errorf("addr = %q, want empty", conf.addr)
		}
	})

	t.Run("dhcp address learned by presence is probed by every interface", func(t *testing.T) {
		t.Parallel()
		cfg := fastRejoinConfig()
		cfg.candidates = []string{"", "wlan0"}
		cfg.scanPresence = func(_ context.Context, _, iface string, _ bool, _ time.Duration) (addr, via string, err error) {
			if iface == "wlan0" {
				return "192.0.2.55", viaMDNS, nil
			}
			return "", "", nil
		}
		cfg.probe = func(_ context.Context, addr, _ string, _ int) error {
			if addr == "192.0.2.55" {
				return nil
			}
			return fmt.Errorf("unknown target %q", addr)
		}
		conf, err := raceRejoin(context.Background(), &cfg)
		if err != nil {
			t.Fatalf("raceRejoin: %v", err)
		}
		if !conf.writeable || conf.addr != "192.0.2.55" || conf.via != viaProbe {
			t.Errorf("conf = %+v, want a writeable probe at 192.0.2.55", conf)
		}
	})
}

func TestRejoinCandidateInterfaces(t *testing.T) {
	t.Parallel()
	subnet := mustCIDR(t, "192.0.2.0/24")
	ifaces := []probeIface{
		{Name: "eth0", Nets: []*net.IPNet{subnet}},
		{Name: "wlan0", IsWireless: true, Nets: []*net.IPNet{subnet}},
	}
	dup := []probeIface{ifaces[0], ifaces[0], ifaces[1]}
	tests := []struct {
		name     string
		staticIP string
		ifaces   []probeIface
		want     []string
	}{
		{"static IP narrows to same-subnet interfaces", "192.0.2.10", ifaces, []string{"", "eth0", "wlan0"}},
		{"dhcp fans out across every interface", "", ifaces, []string{"", "eth0", "wlan0"}},
		{"dhcp default route only with no interfaces", "", nil, []string{""}},
		{"dhcp de-duplicates repeated names", "", dup, []string{"", "eth0", "wlan0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := rejoinCandidateInterfaces(tt.staticIP, tt.ifaces); !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestScanPresenceOnce_InputValidation covers the guards that fail before any
// listener is created.
func TestScanPresenceOnce_InputValidation(t *testing.T) {
	t.Parallel()
	r := testRunner(t, nil, "")
	if _, _, err := r.scanPresenceOnce(context.Background(), "not-a-mac", "", true, time.Second); err == nil {
		t.Error("expected an error for an unparseable MAC")
	}
	_, _, err := r.scanPresenceOnce(context.Background(), fakeMAC, "definitely-not-a-real-iface", true, time.Second)
	if err == nil || !strings.Contains(err.Error(), "resolve interface") {
		t.Errorf("err = %v, want a 'resolve interface' failure", err)
	}
}

func TestScanForMAC(t *testing.T) {
	t.Parallel()
	r := testRunner(t, nil, "")
	sweep := func(context.Context) ([]discovery.DiscoveredDevice, error) {
		return []discovery.DiscoveredDevice{
			{MACAddress: "11:22:33:44:55:66", Address: netip.MustParseAddr("192.0.2.1").AsSlice()},
			{MACAddress: "aa:bb:cc:dd:ee:ff", Address: netip.MustParseAddr("192.0.2.2").AsSlice()},
		}, errors.New("one listener failed")
	}
	if got := r.scanForMAC(context.Background(), sweep, fakeMAC); got != "192.0.2.2" {
		t.Errorf("scanForMAC = %q, want 192.0.2.2", got)
	}
	if got := r.scanForMAC(context.Background(), sweep, "000000000000"); got != "" {
		t.Errorf("scanForMAC = %q, want empty for an absent MAC", got)
	}
}

func TestRejoinRaceSharedAddr(t *testing.T) {
	t.Parallel()
	r := &rejoinRace{}
	if got := r.sharedAddr(); got != "" {
		t.Errorf("sharedAddr() before any sighting = %q, want empty", got)
	}
	r.recordWeak("192.0.2.55", viaMDNS)
	r.recordWeak("192.0.2.99", viaCoIoT)
	if got := r.sharedAddr(); got != "192.0.2.55" {
		t.Errorf("sharedAddr() = %q, want the first sighting", got)
	}
}

func TestIfaceLabel(t *testing.T) {
	t.Parallel()
	if ifaceLabel("") != "default route" || ifaceLabel("wlan0") != "wlan0" {
		t.Error("ifaceLabel names the default route and keeps other names")
	}
}

func TestNormalizeMAC(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"aa:bb:cc:dd:ee:ff": fakeMAC,
		"AABBCCDDEEFF":      fakeMAC,
		"aa-bb-cc-dd-ee-ff": fakeMAC,
		"aabbcc":            "",
		"":                  "",
		"zz:zz:zz:zz:zz:zz": "",
	} {
		if got := normalizeMAC(in); got != want {
			t.Errorf("normalizeMAC(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProbeReachable(t *testing.T) {
	t.Parallel()
	for _, gen := range []int{1, 2} {
		t.Run(fmt.Sprintf("gen%d", gen), func(t *testing.T) {
			t.Parallel()
			d := newFakeDevice(t, gen)
			r := testRunner(t, nil, "")
			if err := r.probeReachable(context.Background(), d.addr(), "", gen, fakeMAC); err != nil {
				t.Errorf("matching MAC: %v", err)
			}
			if err := r.probeReachable(context.Background(), d.addr(), "", gen, ""); err != nil {
				t.Errorf("unknown MAC: %v", err)
			}
			err := r.probeReachable(context.Background(), d.addr(), "", gen, "112233445566")
			if err == nil || !strings.Contains(err.Error(), "not 112233445566") {
				t.Errorf("err = %v, want a MAC mismatch", err)
			}
		})
	}
	t.Run("gen1 settings failure", func(t *testing.T) {
		t.Parallel()
		d := newFakeDevice(t, 1)
		d.settingsErr = true
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		if err := testRunner(t, nil, "").probeReachable(ctx, d.addr(), "", 1, fakeMAC); err == nil {
			t.Error("expected an error when /settings fails")
		}
	})
}

func TestConfirmRejoin_StaticSuccess(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 1)
	r := testRunner(t, nil, "")
	r.hostIfaces = func() ([]probeIface, error) { return nil, errors.New("no interfaces") }
	conf, err := r.confirmRejoin(context.Background(), 1, d.addr(), fakeMAC)
	if err != nil {
		t.Fatalf("confirmRejoin: %v", err)
	}
	if !conf.writeable || conf.addr != d.addr() || conf.bindIface != "" {
		t.Errorf("conf = %+v, want a writeable confirmation at %s over the default route", conf, d.addr())
	}
}

func TestScanPresenceOnce_Sweeps(t *testing.T) {
	t.Parallel()
	device := func(mac, ip string) []discovery.DiscoveredDevice {
		return []discovery.DiscoveredDevice{{MACAddress: mac, Address: netip.MustParseAddr(ip).AsSlice()}}
	}
	tests := []struct {
		name         string
		isGen1       bool
		mdns, coiot  *fakeSweeper
		want         string
		wantVia      string
		coiotStarted bool
	}{
		{
			name: "found via mDNS", mdns: &fakeSweeper{devices: device(fakeMAC, "192.0.2.10")},
			coiot: &fakeSweeper{}, want: "192.0.2.10", wantVia: viaMDNS,
		},
		{
			name: "found via CoIoT for Gen1", isGen1: true, mdns: &fakeSweeper{err: errors.New("no mDNS")},
			coiot: &fakeSweeper{devices: device(fakeMAC, "192.0.2.11")}, want: "192.0.2.11", wantVia: viaCoIoT, coiotStarted: true,
		},
		{
			name: "CoIoT skipped for Gen2", mdns: &fakeSweeper{},
			coiot: &fakeSweeper{devices: device(fakeMAC, "192.0.2.12")},
		},
		{
			name: "other device only", isGen1: true, mdns: &fakeSweeper{devices: device("112233445566", "192.0.2.13")},
			coiot: &fakeSweeper{stopErr: errors.New("stop failed")}, coiotStarted: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := testRunner(t, nil, "")
			var coiotStarted bool
			r.newMDNS = func(ifi *net.Interface) presenceSweeper {
				if ifi != nil {
					t.Errorf("mDNS sweeper bound to %v, want every interface", ifi)
				}
				return tt.mdns
			}
			r.newCoIoT = func(*net.Interface) presenceSweeper { coiotStarted = true; return tt.coiot }
			got, via, err := r.scanPresenceOnce(context.Background(), fakeMAC, "", tt.isGen1, time.Second)
			if err != nil {
				t.Fatalf("scanPresenceOnce: %v", err)
			}
			if got != tt.want || via != tt.wantVia {
				t.Errorf("sighting = %q via %q, want %q via %q", got, via, tt.want, tt.wantVia)
			}
			if coiotStarted != tt.coiotStarted {
				t.Errorf("CoIoT sweep started = %v, want %v", coiotStarted, tt.coiotStarted)
			}
			if tt.want == "" && !tt.mdns.stopped.Load() {
				t.Error("the mDNS sweeper was not stopped")
			}
			if tt.coiotStarted && tt.want == "" && !tt.coiot.stopped.Load() {
				t.Error("the CoIoT sweeper was not stopped")
			}
		})
	}
}

func TestScanPresenceOnce_BindsNamedInterface(t *testing.T) {
	t.Parallel()
	ifaces, err := net.Interfaces()
	if err != nil || len(ifaces) == 0 {
		t.Skipf("no interfaces to resolve: %v", err)
	}
	name := ifaces[0].Name
	r := testRunner(t, nil, "")
	var bound string
	r.newMDNS = func(ifi *net.Interface) presenceSweeper {
		if ifi != nil {
			bound = ifi.Name
		}
		return &fakeSweeper{}
	}
	if _, _, err := r.scanPresenceOnce(context.Background(), fakeMAC, name, false, time.Second); err != nil {
		t.Fatalf("scanPresenceOnce: %v", err)
	}
	if bound != name {
		t.Errorf("sweeper bound to %q, want %q", bound, name)
	}
}

// The production constructors only build the discoverers; nothing listens
// until DiscoverWithContext runs, so this opens no sockets.
func TestPresenceSweeperConstructors(t *testing.T) {
	t.Parallel()
	ifi := &net.Interface{Index: 1, Name: "lo"}
	if _, ok := newMDNSSweeper(ifi).(*discovery.MDNSDiscoverer); !ok {
		t.Error("newMDNSSweeper does not build an mDNS discoverer")
	}
	if _, ok := newCoIoTSweeper(ifi).(*discovery.CoIoTDiscoverer); !ok {
		t.Error("newCoIoTSweeper does not build a CoIoT discoverer")
	}
}
