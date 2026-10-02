package reprovision

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/types"
)

var errWork = errors.New("work failed")

// minimalScanner implements only discovery.WiFiScanner, none of the optional
// interfaces.
type minimalScanner struct{ connects []string }

func (m *minimalScanner) Scan(context.Context) ([]discovery.WiFiNetwork, error) { return nil, nil }
func (m *minimalScanner) Connect(_ context.Context, ssid, _ string) error {
	m.connects = append(m.connects, ssid)
	return nil
}
func (m *minimalScanner) Disconnect(context.Context) error { return nil }
func (m *minimalScanner) CurrentNetwork(context.Context) (*discovery.WiFiNetwork, error) {
	return nil, errors.New("not connected")
}

func homeNetwork() *Network { return &Network{SSID: homeSSID, Password: homePass} }

// assertReturnedHome checks the host's final connect was to the home network and
// the access point block was dropped.
func assertReturnedHome(t *testing.T, s *fakeScanner, apSSID string) {
	t.Helper()
	if got := s.lastConnect(); got != homeSSID {
		t.Errorf("last connect = %q, want the home network %q (calls %v)", got, homeSSID, s.connectCalls())
	}
	if !slices.Contains(s.forgetCalls(), apSSID) {
		t.Errorf("forgotten = %v, want %q dropped", s.forgetCalls(), apSSID)
	}
}

func TestWithAPHop_WorkFailsReturnsHome(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 1)
	s := homeScanner()
	r := testRunner(t, s, d.addr())
	r.apHostIP = "192.168.33.150"

	err := r.withAPHop(context.Background(), fakeAPSSID, homeNetwork(), func(context.Context) error {
		if got := s.lastConnect(); got != fakeAPSSID {
			t.Errorf("work ran while connected to %q, want the AP", got)
		}
		return errWork
	})
	if !errors.Is(err, errWork) {
		t.Fatalf("err = %v, want the work error", err)
	}
	assertReturnedHome(t, s, fakeAPSSID)
	if s.apHostIP != "192.168.33.150" {
		t.Errorf("AP host IP = %q, want it passed to the scanner", s.apHostIP)
	}
	want := []string{fakeAPSSID + "|", homeSSID + "|" + homePass}
	if got := s.connectCalls(); !slices.Equal(got, want) {
		t.Errorf("connects = %v, want %v", got, want)
	}
}

func TestWithAPHop_ContextCancelledReturnsHome(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 1)
	s := homeScanner()
	r := testRunner(t, s, d.addr())
	ctx, cancel := context.WithCancel(context.Background())

	err := r.withAPHop(ctx, fakeAPSSID, homeNetwork(), func(ctx context.Context) error {
		cancel()
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	assertReturnedHome(t, s, fakeAPSSID)
}

func TestWithAPHop_AlreadyCancelledStillReturnsHome(t *testing.T) {
	t.Parallel()
	s := homeScanner()
	r := testRunner(t, s, refusingAddr(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.withAPHop(ctx, fakeAPSSID, homeNetwork(), func(ctx context.Context) error { return ctx.Err() })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	assertReturnedHome(t, s, fakeAPSSID)
}

func TestWithAPHop_APNeverReadyReturnsHome(t *testing.T) {
	t.Parallel()
	s := homeScanner()
	r := testRunner(t, s, refusingAddr(t))

	ran := false
	err := r.withAPHop(context.Background(), fakeAPSSID, homeNetwork(), func(context.Context) error {
		ran = true
		return errWork
	})
	if !ran || !errors.Is(err, errWork) {
		t.Fatalf("ran=%v err=%v, want the work to run after the readiness wait and fail", ran, err)
	}
	assertReturnedHome(t, s, fakeAPSSID)
}

func TestWithAPHop_ConnectFailureStillReturnsHome(t *testing.T) {
	t.Parallel()
	s := homeScanner()
	s.connectErr = func(ssid string) error {
		if ssid == fakeAPSSID {
			return errors.New("association failed")
		}
		return nil
	}
	r := testRunner(t, s, refusingAddr(t))

	err := r.withAPHop(context.Background(), fakeAPSSID, homeNetwork(), func(context.Context) error {
		t.Error("work must not run when the AP connect fails")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "failed to connect to Shelly AP") {
		t.Fatalf("err = %v, want the AP connect failure", err)
	}
	assertReturnedHome(t, s, fakeAPSSID)
}

func TestWithAPHop_NilScanner(t *testing.T) {
	t.Parallel()
	r := testRunner(t, nil, refusingAddr(t))
	r.scanner = nil
	err := r.withAPHop(context.Background(), fakeAPSSID, homeNetwork(), func(context.Context) error { return nil })
	if !errors.Is(err, types.ErrNotSupported) {
		t.Errorf("err = %v, want ErrNotSupported", err)
	}
}

func TestReturnFromAPHop_FallsBackToHomeCredentials(t *testing.T) {
	t.Parallel()
	s := homeScanner()
	s.connectErr = func(ssid string) error {
		if ssid == "OtherNet" {
			return errors.New("no saved profile")
		}
		return nil
	}
	s.forgetErr = errors.New("forget failed")
	r := testRunner(t, s, refusingAddr(t))

	r.returnFromAPHop(context.Background(), fakeAPSSID, &discovery.WiFiNetwork{SSID: "OtherNet"}, homeNetwork())
	want := []string{"OtherNet|", homeSSID + "|" + homePass}
	if got := s.connectCalls(); !slices.Equal(got, want) {
		t.Errorf("connects = %v, want %v", got, want)
	}
}

func TestReturnFromAPHop_HomeFailureIsLogged(t *testing.T) {
	t.Parallel()
	s := homeScanner()
	s.connectErr = func(string) error { return errors.New("down") }
	r := testRunner(t, s, refusingAddr(t))
	r.returnFromAPHop(context.Background(), fakeAPSSID, nil, homeNetwork())
	if got := s.connectCalls(); len(got) != 1 {
		t.Errorf("connects = %v, want one attempt with the home credentials", got)
	}
}

func TestReturnFromAPHop_ScannerWithoutForget(t *testing.T) {
	t.Parallel()
	m := &minimalScanner{}
	r := testRunner(t, m, refusingAddr(t))
	r.returnFromAPHop(context.Background(), fakeAPSSID, nil, homeNetwork())
	if !slices.Equal(m.connects, []string{homeSSID}) {
		t.Errorf("connects = %v, want [%s]", m.connects, homeSSID)
	}
}

func TestReconnectCredentials(t *testing.T) {
	t.Parallel()
	home := homeNetwork()
	tests := []struct {
		name     string
		original *discovery.WiFiNetwork
		wantSSID string
		wantPass string
	}{
		{"no original network", nil, homeSSID, homePass},
		{"original has no SSID", &discovery.WiFiNetwork{}, homeSSID, homePass},
		{"original is home", &discovery.WiFiNetwork{SSID: homeSSID}, homeSSID, homePass},
		{"original differs", &discovery.WiFiNetwork{SSID: "Office"}, "Office", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ssid, pass := reconnectCredentials(tt.original, home)
			if ssid != tt.wantSSID || pass != tt.wantPass {
				t.Errorf("got (%q, %q), want (%q, %q)", ssid, pass, tt.wantSSID, tt.wantPass)
			}
		})
	}
}

func TestResolveJoinNetwork(t *testing.T) {
	t.Parallel()
	static := Network{StaticIP: "192.0.2.10", Gateway: "192.0.2.1", Netmask: "255.255.255.0", DNS: "192.0.2.53"}
	tests := []struct {
		name       string
		fromBackup Network
		override   Network
		want       Network
	}{
		{
			name:       "override ssid and password win",
			fromBackup: Network{SSID: "BackupNet", Password: "backupkey"},
			override:   Network{SSID: "OverrideNet", Password: "overridekey"},
			want:       Network{SSID: "OverrideNet", Password: "overridekey"},
		},
		{
			name:       "override ssid takes the host passphrase and drops the backup key",
			fromBackup: Network{SSID: "BackupNet", Password: "backupkey"},
			override:   Network{SSID: homeSSID},
			want:       Network{SSID: homeSSID, Password: homePass},
		},
		{
			name:       "backup ssid and key",
			fromBackup: Network{SSID: "BackupNet", Password: "backupkey"},
			want:       Network{SSID: "BackupNet", Password: "backupkey"},
		},
		{
			name:       "host passphrase for the backup ssid",
			fromBackup: Network{SSID: homeSSID},
			want:       Network{SSID: homeSSID, Password: homePass},
		},
		{
			name: "host network and passphrase when the backup has none",
			want: Network{SSID: homeSSID, Password: homePass},
		},
		{
			name:       "backup static settings are kept",
			fromBackup: Network{SSID: homeSSID, StaticIP: "192.0.2.99", Gateway: "192.0.2.254", Netmask: "255.255.0.0"},
			want: Network{
				SSID: homeSSID, Password: homePass,
				StaticIP: "192.0.2.99", Gateway: "192.0.2.254", Netmask: "255.255.0.0",
			},
		},
		{
			name:       "override static settings replace the backup's",
			fromBackup: Network{SSID: homeSSID, StaticIP: "192.0.2.99", Gateway: "192.0.2.254", DNS: "8.8.8.8"},
			override:   static,
			want: Network{
				SSID: homeSSID, Password: homePass,
				StaticIP: static.StaticIP, Gateway: static.Gateway, Netmask: static.Netmask, DNS: static.DNS,
			},
		},
		{
			name: "override static ip alone keeps the backup's gateway, netmask and dns",
			fromBackup: Network{
				SSID: homeSSID, StaticIP: "192.0.2.99", Gateway: "192.0.2.254", Netmask: "255.255.0.0", DNS: "192.0.2.53",
			},
			override: Network{StaticIP: "192.0.2.10"},
			want: Network{
				SSID: homeSSID, Password: homePass,
				StaticIP: "192.0.2.10", Gateway: "192.0.2.254", Netmask: "255.255.0.0", DNS: "192.0.2.53",
			},
		},
		{
			name:       "override gateway replaces the backup's, netmask falls back",
			fromBackup: Network{SSID: homeSSID, StaticIP: "192.0.2.99", Gateway: "192.0.2.254", Netmask: "255.255.0.0"},
			override:   Network{StaticIP: "192.0.2.10", Gateway: "192.0.2.1"},
			want: Network{
				SSID: homeSSID, Password: homePass,
				StaticIP: "192.0.2.10", Gateway: "192.0.2.1", Netmask: "255.255.0.0",
			},
		},
		{
			name:       "dhcp backup stays dhcp",
			fromBackup: Network{SSID: homeSSID},
			want:       Network{SSID: homeSSID, Password: homePass},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := testRunner(t, homeScanner(), "")
			got, err := r.resolveJoinNetwork(context.Background(), &tt.fromBackup, &tt.override)
			if err != nil {
				t.Fatalf("resolveJoinNetwork: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolveJoinNetwork_StaticNeedsGatewayAndNetmask(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		fromBackup Network
		override   Network
	}{
		{
			name:       "static override on a dhcp backup",
			fromBackup: Network{SSID: homeSSID},
			override:   Network{StaticIP: "192.0.2.10"},
		},
		{
			name:       "static override with a gateway but no netmask anywhere",
			fromBackup: Network{SSID: homeSSID},
			override:   Network{StaticIP: "192.0.2.10", Gateway: "192.0.2.1"},
		},
		{
			name:       "static backup with no netmask",
			fromBackup: Network{SSID: homeSSID, StaticIP: "192.0.2.99", Gateway: "192.0.2.254"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := testRunner(t, homeScanner(), "")
			got, err := r.resolveJoinNetwork(context.Background(), &tt.fromBackup, &tt.override)
			if !errors.Is(err, ErrIncompleteStaticNetwork) || !errors.Is(err, types.ErrInvalidParam) {
				t.Fatalf("got (%+v, %v), want ErrIncompleteStaticNetwork and types.ErrInvalidParam", got, err)
			}
		})
	}
}

func TestResolveJoinNetwork_BackupKeyNeverJoinsAnotherNetwork(t *testing.T) {
	t.Parallel()
	r := testRunner(t, homeScanner(), "")
	fromBackup := Network{SSID: "BackupNet", Password: "backupkey"}
	_, err := r.resolveJoinNetwork(context.Background(), &fromBackup, &Network{SSID: "OverrideNet"})
	var pwErr *NoPassphraseError
	if !errors.As(err, &pwErr) || pwErr.SSID != "OverrideNet" {
		t.Fatalf("err = %v, want a NoPassphraseError naming OverrideNet", err)
	}
}

func TestResolveJoinNetwork_NoPassphrase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		scanner discovery.WiFiScanner
	}{
		{"host not on wifi", &fakeScanner{currentErr: errors.New("not connected")}},
		{"no stored passphrase", &fakeScanner{current: &discovery.WiFiNetwork{SSID: homeSSID}}},
		{"scanner cannot recover passphrases", &minimalScanner{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := testRunner(t, tt.scanner, "")
			_, err := r.resolveJoinNetwork(context.Background(), &Network{SSID: "Some"}, &Network{})
			if !errors.Is(err, ErrNoPassphrase) {
				t.Fatalf("err = %v, want ErrNoPassphrase", err)
			}
			if !strings.Contains(err.Error(), "Network.Password") {
				t.Errorf("err = %q, want it to name Network.Password", err)
			}
			var pwErr *NoPassphraseError
			if !errors.As(err, &pwErr) || pwErr.SSID != "Some" {
				t.Errorf("err = %#v, want a NoPassphraseError for %q", err, "Some")
			}
		})
	}
}

func TestResolveJoinNetwork_NoPassphraseNamesHostNetwork(t *testing.T) {
	t.Parallel()
	r := testRunner(t, &fakeScanner{current: &discovery.WiFiNetwork{SSID: homeSSID}}, "")
	_, err := r.resolveJoinNetwork(context.Background(), &Network{}, &Network{})
	var pwErr *NoPassphraseError
	if !errors.As(err, &pwErr) {
		t.Fatalf("err = %v, want a NoPassphraseError", err)
	}
	if pwErr.SSID != homeSSID {
		t.Errorf("SSID = %q, want the host's network %q", pwErr.SSID, homeSSID)
	}
}

func TestMergeNetwork_BackupKeyBelongsToBackupSSID(t *testing.T) {
	t.Parallel()
	fromBackup := Network{SSID: "Home", Password: "homekey"}
	tests := []struct {
		name     string
		override Network
		want     Network
	}{
		{"no override keeps the backup's key", Network{}, Network{SSID: "Home", Password: "homekey"}},
		{"same SSID keeps the backup's key", Network{SSID: "Home"}, Network{SSID: "Home", Password: "homekey"}},
		{"another SSID drops the backup's key", Network{SSID: "Other"}, Network{SSID: "Other"}},
		{
			"another SSID with a password uses it",
			Network{SSID: "Other", Password: "otherkey"},
			Network{SSID: "Other", Password: "otherkey"},
		},
		{"open drops the backup's key", Network{Open: true}, Network{SSID: "Home", Open: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := MergeNetwork(&fromBackup, &tt.override)
			if err != nil {
				t.Fatalf("MergeNetwork: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestMergeNetwork_NilArguments(t *testing.T) {
	t.Parallel()
	got, err := MergeNetwork(nil, nil)
	if err != nil || got != (Network{}) {
		t.Errorf("got (%+v, %v), want an empty network and no error", got, err)
	}
}

func TestHostWiFiPassword_NilScanner(t *testing.T) {
	t.Parallel()
	r := testRunner(t, nil, "")
	r.scanner = nil
	if _, err := r.hostWiFiPassword(context.Background(), homeSSID); !errors.Is(err, types.ErrNotSupported) {
		t.Errorf("err = %v, want ErrNotSupported", err)
	}
	if _, err := r.resolveJoinNetwork(context.Background(), &Network{}, &Network{}); !errors.Is(err, ErrNoPassphrase) {
		t.Errorf("err = %v, want ErrNoPassphrase", err)
	}
}

func TestResolveJoinNetwork_Open(t *testing.T) {
	t.Parallel()
	const guest = "GuestNet"
	tests := []struct {
		name       string
		fromBackup Network
		override   Network
		want       Network
	}{
		{
			name:     "open override takes no passphrase",
			override: Network{SSID: homeSSID, Open: true},
			want:     Network{SSID: homeSSID, Open: true},
		},
		{
			name:       "open override takes the backup's SSID",
			fromBackup: Network{SSID: guest},
			override:   Network{Open: true},
			want:       Network{SSID: guest, Open: true},
		},
		{
			name:     "open override takes the host's SSID",
			override: Network{Open: true},
			want:     Network{SSID: homeSSID, Open: true},
		},
		{
			name:       "open backup rejoins open",
			fromBackup: Network{SSID: homeSSID, Open: true},
			want:       Network{SSID: homeSSID, Open: true},
		},
		{
			name:       "open backup with a password override is secured",
			fromBackup: Network{SSID: guest, Open: true},
			override:   Network{Password: "secret"},
			want:       Network{SSID: guest, Password: "secret"},
		},
		{
			name:       "open backup with another SSID is secured",
			fromBackup: Network{SSID: guest, Open: true},
			override:   Network{SSID: homeSSID},
			want:       Network{SSID: homeSSID, Password: homePass},
		},
		{
			name:       "open with a static override still follows the static rule",
			fromBackup: Network{SSID: guest, Open: true, StaticIP: "192.0.2.99", Gateway: "192.0.2.1", Netmask: "255.255.255.0"},
			override:   Network{StaticIP: "192.0.2.10"},
			want: Network{
				SSID: guest, Open: true, StaticIP: "192.0.2.10", Gateway: "192.0.2.1", Netmask: "255.255.255.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := testRunner(t, homeScanner(), "")
			got, err := r.resolveJoinNetwork(context.Background(), &tt.fromBackup, &tt.override)
			if err != nil {
				t.Fatalf("resolveJoinNetwork: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolveJoinNetwork_OpenRefusals(t *testing.T) {
	t.Parallel()
	t.Run("open with a password", func(t *testing.T) {
		t.Parallel()
		r := testRunner(t, homeScanner(), "")
		_, err := r.resolveJoinNetwork(context.Background(), &Network{}, &Network{SSID: homeSSID, Open: true, Password: "x"})
		if !errors.Is(err, types.ErrInvalidParam) {
			t.Fatalf("err = %v, want ErrInvalidParam", err)
		}
	})
	t.Run("open with no SSID anywhere", func(t *testing.T) {
		t.Parallel()
		r := testRunner(t, &fakeScanner{currentErr: errors.New("not connected")}, "")
		_, err := r.resolveJoinNetwork(context.Background(), &Network{}, &Network{Open: true})
		if !errors.Is(err, types.ErrInvalidParam) || !strings.Contains(err.Error(), "Network.SSID") {
			t.Fatalf("err = %v, want ErrInvalidParam naming Network.SSID", err)
		}
	})
}
