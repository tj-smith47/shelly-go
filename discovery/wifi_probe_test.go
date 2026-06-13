package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// ────────────────────────────────────────────────────────────────────────────
// probeEndpoint
// ────────────────────────────────────────────────────────────────────────────

// newProbeDiscoverer builds a WiFiDiscoverer whose HTTPClient points at a test
// server at addr. probeEndpoint uses DefaultAPIP for its URL; we patch the
// discoverer's HTTPClient transport so all http://192.168.33.1/... requests are
// redirected to the test server.
func newProbeDiscoverer(ts *httptest.Server) *WiFiDiscoverer {
	mock := &mockWiFiScanner{}
	d := NewWiFiDiscovererWithScanner(mock)
	// Replace the URL scheme+host used by probeEndpoint via a custom RoundTripper.
	d.HTTPClient = &http.Client{
		Timeout:   2 * time.Second,
		Transport: &redirectTransport{base: ts.URL},
	}
	return d
}

// redirectTransport rewrites the host:port of every request to the test server.
type redirectTransport struct {
	base string
}

func (r *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request and replace the scheme+host.
	clone := req.Clone(req.Context())
	// Parse the base URL.
	import_url := r.base
	clone.URL.Scheme = "http"
	// Strip the scheme from base.
	host := import_url
	if len(host) > 7 && host[:7] == "http://" {
		host = host[7:]
	}
	clone.URL.Host = host
	return http.DefaultTransport.RoundTrip(clone)
}

func TestProbeEndpoint_Success(t *testing.T) {
	payload := map[string]any{"name": "ShellyPlus1", "gen": float64(2)}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(payload) //nolint:errcheck
	}))
	defer ts.Close()

	d := newProbeDiscoverer(ts)
	ctx := context.Background()
	result, err := d.probeEndpoint(ctx, "/rpc/Shelly.GetDeviceInfo")
	if err != nil {
		t.Fatalf("probeEndpoint: %v", err)
	}
	if result["name"] != "ShellyPlus1" {
		t.Errorf("name = %v, want ShellyPlus1", result["name"])
	}
}

func TestProbeEndpoint_Non200Status(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	d := newProbeDiscoverer(ts)
	_, err := d.probeEndpoint(context.Background(), "/rpc/Shelly.GetDeviceInfo")
	if err == nil {
		t.Fatal("probeEndpoint must return error for non-200 status")
	}
}

func TestProbeEndpoint_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json")) //nolint:errcheck
	}))
	defer ts.Close()

	d := newProbeDiscoverer(ts)
	_, err := d.probeEndpoint(context.Background(), "/rpc/Shelly.GetDeviceInfo")
	if err == nil {
		t.Fatal("probeEndpoint must return error for invalid JSON")
	}
}

func TestProbeEndpoint_ContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the test times out — the context should cancel us first.
		<-r.Context().Done()
	}))
	defer ts.Close()

	d := newProbeDiscoverer(ts)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := d.probeEndpoint(ctx, "/rpc/Shelly.GetDeviceInfo")
	if err == nil {
		t.Fatal("probeEndpoint must return error when context is canceled")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// probeGen2 / probeGen1
// ────────────────────────────────────────────────────────────────────────────

func TestProbeGen2_CallsCorrectEndpoint(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(map[string]any{"gen": float64(2)}) //nolint:errcheck
	}))
	defer ts.Close()

	d := newProbeDiscoverer(ts)
	_, err := d.probeGen2(context.Background())
	if err != nil {
		t.Fatalf("probeGen2: %v", err)
	}
	if gotPath != "/rpc/Shelly.GetDeviceInfo" {
		t.Errorf("probeGen2 called %q, want /rpc/Shelly.GetDeviceInfo", gotPath)
	}
}

func TestProbeGen1_CallsCorrectEndpoint(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(map[string]any{"type": "SHSW-1"}) //nolint:errcheck
	}))
	defer ts.Close()

	d := newProbeDiscoverer(ts)
	_, err := d.probeGen1(context.Background())
	if err != nil {
		t.Fatalf("probeGen1: %v", err)
	}
	if gotPath != "/shelly" {
		t.Errorf("probeGen1 called %q, want /shelly", gotPath)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// probeDevice — uses applyGen2Info / applyGen1Info; network scanner mock
// ────────────────────────────────────────────────────────────────────────────

func TestProbeDevice_AppliesGen2Info(t *testing.T) {
	// Serve a Gen2 response on the probe endpoint.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rpc/Shelly.GetDeviceInfo" {
			json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
				"name":    "LivingRoomPlug",
				"model":   "SNSW-001P16EU",
				"mac":     "AABBCCDDEEFF",
				"fw_id":   "20240101-123456/v1.0.0",
				"gen":     float64(2),
				"auth_en": false,
				"id":      "shellyplus1pm-aabbccddeeff",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	mock := &mockWiFiScanner{}
	d := NewWiFiDiscovererWithScanner(mock)
	d.HTTPClient = &http.Client{
		Timeout:   2 * time.Second,
		Transport: &redirectTransport{base: ts.URL},
	}

	network := &WiFiNetwork{SSID: "shellyplus1pm-AABBCC"}
	ctx := context.Background()
	device, err := d.probeDevice(ctx, network)
	if err != nil {
		t.Fatalf("probeDevice: %v", err)
	}
	if device == nil {
		t.Fatal("probeDevice returned nil device")
	}
	if device.Name != "LivingRoomPlug" {
		t.Errorf("Name = %q, want LivingRoomPlug", device.Name)
	}
	if device.Model != "SNSW-001P16EU" {
		t.Errorf("Model = %q, want SNSW-001P16EU", device.Model)
	}
	if device.Generation != types.Generation(2) {
		t.Errorf("Generation = %v, want 2", device.Generation)
	}
}

func TestProbeDevice_AppliesGen1Info(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/shelly" {
			json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
				"type": "SHSW-1",
				"mac":  "AABBCCDDEEFF",
				"fw":   "1.11.8",
				"auth": false,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	mock := &mockWiFiScanner{}
	d := NewWiFiDiscovererWithScanner(mock)
	d.HTTPClient = &http.Client{
		Timeout:   2 * time.Second,
		Transport: &redirectTransport{base: ts.URL},
	}

	network := &WiFiNetwork{SSID: "shelly1-AABBCC"}
	device, err := d.probeDevice(context.Background(), network)
	if err != nil {
		t.Fatalf("probeDevice (Gen1): %v", err)
	}
	if device == nil {
		t.Fatal("probeDevice returned nil device")
	}
	if device.Model != "SHSW-1" {
		t.Errorf("Model = %q, want SHSW-1", device.Model)
	}
	if device.Generation != types.Gen1 {
		t.Errorf("Generation = %v, want Gen1", device.Generation)
	}
}

func TestProbeDevice_BothProbeFail_ReturnsBasicDevice(t *testing.T) {
	// Both /rpc/Shelly.GetDeviceInfo and /shelly return 404 — probeDevice must
	// still return a device (with basic info from the WiFiNetwork).
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()

	mock := &mockWiFiScanner{}
	d := NewWiFiDiscovererWithScanner(mock)
	d.HTTPClient = &http.Client{
		Timeout:   2 * time.Second,
		Transport: &redirectTransport{base: ts.URL},
	}

	network := &WiFiNetwork{SSID: "shellyplus1pm-UNKNOWN"}
	device, err := d.probeDevice(context.Background(), network)
	if err != nil {
		t.Fatalf("probeDevice must not error when probes fail: %v", err)
	}
	if device == nil {
		t.Fatal("probeDevice must return a basic device even when probes fail")
	}
	// SSID should have been preserved.
	if device.SSID != "shellyplus1pm-UNKNOWN" {
		t.Errorf("SSID = %q, want shellyplus1pm-UNKNOWN", device.SSID)
	}
}

func TestProbeDevice_ConnectFails_ReturnsError(t *testing.T) {
	mock := &mockWiFiScanner{
		connectErr: &WiFiError{Message: "connect refused"},
	}
	d := NewWiFiDiscovererWithScanner(mock)

	network := &WiFiNetwork{SSID: "shellyplus1pm-AABBCC"}
	_, err := d.probeDevice(context.Background(), network)
	if err == nil {
		t.Fatal("probeDevice must return error when Connect fails")
	}
}
