package reprovision

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
)

// The fakes below keep every test in-process: the device is an httptest server
// the runner's apAddr points at, and the WiFi scanner only records calls. No
// test reaches a real access point, joins a network or runs a host command.

const (
	fakeMAC       = "AABBCCDDEEFF"
	fakeAPSSID    = "ShellyBulbDuo-DDEEFF"
	otherAPSSID   = "ShellyBulbDuo-6645B6"
	homeSSID      = "HomeNet"
	homePass      = "homepass"
	fakeGen1Model = "SHBDUO-1"
)

// fakeDevice is an in-process stand-in for a Shelly device that records the
// requests it was sent.
type fakeDevice struct {
	srv *httptest.Server

	mu         sync.Mutex
	fw         string
	otaFlipFW  string
	paths      []string
	rpcMethods []string
	uptime     int

	// failMethod makes that RPC method answer with an error.
	failMethod string

	settingsErr bool
	// shellyErr makes the /shelly identity endpoint fail.
	shellyErr bool
	otaErr    bool
	rebootErr bool
}

// newFakeDevice starts a fake device of the given generation.
func newFakeDevice(t *testing.T, generation int) *fakeDevice {
	t.Helper()
	d := &fakeDevice{fw: "20210101-000000/v1.0", uptime: 99}
	mux := http.NewServeMux()
	if generation == 1 {
		d.registerGen1(mux)
	} else {
		d.registerGen2(t, mux)
	}
	d.srv = httptest.NewServer(mux)
	t.Cleanup(d.srv.Close)
	return d
}

func (d *fakeDevice) addr() string { return strings.TrimPrefix(d.srv.URL, "http://") }

func (d *fakeDevice) record(path string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.paths = append(d.paths, path)
}

func (d *fakeDevice) recordRPC(method string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.rpcMethods = append(d.rpcMethods, method)
}

func (d *fakeDevice) currentFW() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.fw
}

func (d *fakeDevice) setFW(fw string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.fw = fw
}

// hits counts requests whose path starts with prefix.
func (d *fakeDevice) hits(prefix string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	for _, p := range d.paths {
		if strings.HasPrefix(p, prefix) {
			n++
		}
	}
	return n
}

// called reports whether the RPC method was called.
func (d *fakeDevice) called(method string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, m := range d.rpcMethods {
		if m == method {
			return true
		}
	}
	return false
}

// writes reports whether any request other than an identity or status read
// reached the device.
func (d *fakeDevice) writes() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []string
	for _, p := range d.paths {
		switch p {
		case "/shelly", "/settings", "/status":
			continue
		}
		out = append(out, p)
	}
	for _, m := range d.rpcMethods {
		if m != "Shelly.GetDeviceInfo" {
			out = append(out, m)
		}
	}
	return out
}

func (d *fakeDevice) registerGen1(mux *http.ServeMux) {
	mux.HandleFunc("/shelly", func(w http.ResponseWriter, r *http.Request) {
		d.record(r.URL.Path)
		if d.shellyErr {
			http.Error(w, "identity unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"type": fakeGen1Model, "mac": fakeMAC, "fw": d.currentFW()})
	})
	mux.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		d.record(r.URL.Path)
		if d.settingsErr {
			http.Error(w, "settings refused", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{
			"fw":       d.currentFW(),
			"device":   map[string]any{"type": fakeGen1Model, "mac": fakeMAC},
			"wifi_sta": map[string]any{"enabled": true, "ssid": homeSSID, "key": "***", "ipv4_method": "dhcp"},
		})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		d.record(r.URL.Path)
		writeJSON(w, map[string]any{
			"uptime": d.uptime, "unixtime": 1700000000,
			"wifi_sta": map[string]any{"connected": true, "ip": "192.0.2.7"},
		})
	})
	mux.HandleFunc("/reboot", func(w http.ResponseWriter, r *http.Request) {
		d.record(r.URL.Path)
		if d.rebootErr {
			http.Error(w, "reboot refused", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/ota", func(w http.ResponseWriter, r *http.Request) {
		d.record(r.URL.Path)
		if d.otaFlipFW != "" {
			d.setFW(d.otaFlipFW)
		}
		if d.otaErr {
			http.Error(w, "ota refused", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"status": "updating"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		d.record(r.URL.Path)
		writeJSON(w, map[string]any{})
	})
}

func (d *fakeDevice) registerGen2(t *testing.T, mux *http.ServeMux) {
	t.Helper()
	mux.HandleFunc("/shelly", func(w http.ResponseWriter, r *http.Request) {
		d.record(r.URL.Path)
		if d.shellyErr {
			http.Error(w, "identity unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{
			"id": "shellyplus1-aabbccddeeff", "mac": fakeMAC, "gen": 2,
			"model": "SNSW-001P16EU", "ver": "1.0.0",
		})
	})
	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read rpc body: %v", err)
			return
		}
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if uerr := json.Unmarshal(body, &req); uerr != nil {
			t.Errorf("decode rpc body: %v", uerr)
			return
		}
		d.recordRPC(req.Method)
		if req.Method == d.failMethod {
			writeJSON(w, map[string]any{
				"id": req.ID, "jsonrpc": "2.0",
				"error": map[string]any{"code": 500, "message": "refused"},
			})
			return
		}
		var result any = map[string]any{}
		switch req.Method {
		case "Shelly.GetDeviceInfo":
			result = map[string]any{
				"id": "shellyplus1-aabbccddeeff", "mac": fakeMAC, "gen": 2,
				"model": "SNSW-001P16EU", "fw_id": "20230101-000000",
			}
		case "Shelly.Reboot":
			if d.rebootErr {
				writeJSON(w, map[string]any{
					"id": req.ID, "jsonrpc": "2.0",
					"error": map[string]any{"code": 500, "message": "reboot refused"},
				})
				return
			}
		case "Schedule.List":
			result = map[string]any{"jobs": []any{}}
		case "Webhook.List":
			result = map[string]any{"hooks": []any{}}
		case "Script.List":
			result = map[string]any{"scripts": []any{}}
		}
		writeJSON(w, map[string]any{"id": req.ID, "jsonrpc": "2.0", "result": result})
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// fakeScanner records the WiFi calls a hop makes. It implements every optional
// scanner interface the package uses.
type fakeScanner struct {
	current    *discovery.WiFiNetwork
	currentErr error
	connectErr func(ssid string) error
	passwords  map[string]string
	networks   []discovery.WiFiNetwork
	scanErr    error

	mu        sync.Mutex
	connects  []string
	forgotten []string
	apHostIP  string
	forgetErr error
}

func (f *fakeScanner) Scan(context.Context) ([]discovery.WiFiNetwork, error) {
	return f.networks, f.scanErr
}

func (f *fakeScanner) Connect(_ context.Context, ssid, password string) error {
	f.mu.Lock()
	f.connects = append(f.connects, ssid+"|"+password)
	f.mu.Unlock()
	if f.connectErr != nil {
		return f.connectErr(ssid)
	}
	return nil
}

func (f *fakeScanner) Disconnect(context.Context) error { return nil }

func (f *fakeScanner) CurrentNetwork(context.Context) (*discovery.WiFiNetwork, error) {
	return f.current, f.currentErr
}

func (f *fakeScanner) SetAPHostIP(ip string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.apHostIP = ip
}

func (f *fakeScanner) ForgetNetwork(_ context.Context, ssid string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forgotten = append(f.forgotten, ssid)
	return f.forgetErr
}

func (f *fakeScanner) HostNetworkPassword(_ context.Context, ssid string) (string, error) {
	if pw, ok := f.passwords[ssid]; ok {
		return pw, nil
	}
	return "", errors.New("no stored credentials")
}

func (f *fakeScanner) connectCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.connects...)
}

func (f *fakeScanner) forgetCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.forgotten...)
}

// lastConnect is the final SSID the host was connected to.
func (f *fakeScanner) lastConnect() string {
	calls := f.connectCalls()
	if len(calls) == 0 {
		return ""
	}
	ssid, _, _ := strings.Cut(calls[len(calls)-1], "|")
	return ssid
}

// homeScanner is a scanner on the home network with its passphrase stored.
func homeScanner() *fakeScanner {
	return &fakeScanner{
		current:   &discovery.WiFiNetwork{SSID: homeSSID},
		passwords: map[string]string{homeSSID: homePass},
	}
}

// testRunner builds a runner whose device address is apAddr, with short waits,
// a recording scanner in place of a nil one, no host interface enumeration, a
// presence scan that never sees anything, multicast sweepers that announce
// nothing, and a firmware client that fails the test on any non-loopback URL.
func testRunner(t *testing.T, scanner discovery.WiFiScanner, apAddr string) *runner {
	t.Helper()
	if scanner == nil {
		scanner = &minimalScanner{}
	}
	r := newRunner(scanner, slog.New(slog.DiscardHandler), nil, "")
	r.apAddr = apAddr
	r.apReadyTimeout = 200 * time.Millisecond
	r.lanSettleDelay = 0
	r.rejoinTimeout = 2 * time.Second
	r.rejoinInterval = 10 * time.Millisecond
	r.presenceTimeout = 10 * time.Millisecond
	r.probeTimeout = time.Second
	r.hostIfaces = func() ([]probeIface, error) { return nil, nil }
	r.scanPresence = func(context.Context, string, string, bool, time.Duration) (string, error) {
		return "", nil
	}
	r.newMDNS = func(*net.Interface) presenceSweeper { return &fakeSweeper{} }
	r.newCoIoT = func(*net.Interface) presenceSweeper { return &fakeSweeper{} }
	r.firmwareClient = &http.Client{Transport: loopbackOnly{fail: t.Errorf}}
	return r
}

// fakeSweeper is a presence sweeper that reports devices and a stop error.
type fakeSweeper struct {
	devices []discovery.DiscoveredDevice
	err     error
	stopErr error
	stopped atomic.Bool
}

func (f *fakeSweeper) DiscoverWithContext(context.Context) ([]discovery.DiscoveredDevice, error) {
	return f.devices, f.err
}

func (f *fakeSweeper) Stop() error {
	f.stopped.Store(true)
	return f.stopErr
}

// loopbackOnly lets requests reach test servers on loopback and fails the
// test for any other host, so no test can download from the public CDN.
type loopbackOnly struct {
	fail func(format string, args ...any)
}

func (l loopbackOnly) RoundTrip(req *http.Request) (*http.Response, error) {
	if ip := net.ParseIP(req.URL.Hostname()); ip == nil || !ip.IsLoopback() {
		l.fail("test tried to fetch %s outside loopback", req.URL)
		return nil, errors.New("non-loopback request blocked in tests")
	}
	return http.DefaultTransport.RoundTrip(req)
}

// refusingAddr returns a localhost address whose port is closed.
func refusingAddr(t *testing.T) string {
	t.Helper()
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	if cerr := ln.Close(); cerr != nil {
		t.Fatalf("close listener: %v", cerr)
	}
	return addr
}
