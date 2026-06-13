package backup

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestOfficialGen1FirmwareURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		model string
		want  string
	}{
		{"SHBDUO-1", "http://firmware.shelly.cloud/gen1/SHBDUO-1.zip"},
		{"SHRGBW2", "http://firmware.shelly.cloud/gen1/SHRGBW2.zip"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := officialGen1FirmwareURL(tt.model); got != tt.want {
			t.Errorf("officialGen1FirmwareURL(%q) = %q, want %q", tt.model, got, tt.want)
		}
	}
}

func TestBackupModel(t *testing.T) {
	t.Parallel()
	if got := backupModel(nil); got != "" {
		t.Errorf("backupModel(nil) = %q, want empty", got)
	}
	if got := backupModel(&Backup{}); got != "" {
		t.Errorf("backupModel(no DeviceInfo) = %q, want empty", got)
	}
	if got := backupModel(&Backup{DeviceInfo: &DeviceInfo{Model: "SHBDUO-1"}}); got != "SHBDUO-1" {
		t.Errorf("backupModel = %q, want SHBDUO-1", got)
	}
}

// gen1FirmwareUpdateDevice starts a fake Gen1 device that reports oldFW until its
// /ota endpoint is hit, after which it reports newFW with a stable uptime — a
// device that "updates" when told to. It records every request URI so the OTA
// trigger and post-update writes can be asserted, and exposes the OTA call count.
func gen1FirmwareUpdateDevice(t *testing.T, oldFW, newFW string) (*gen1.Device, *[]string, *int64) {
	t.Helper()
	var (
		mu       sync.Mutex
		reqs     []string
		otaCalls int64
	)
	curFW := func() string {
		if atomic.LoadInt64(&otaCalls) > 0 {
			return newFW
		}
		return oldFW
	}
	write := func(w http.ResponseWriter, body string) {
		if _, err := io.WriteString(w, body); err != nil {
			t.Errorf("write response: %v", err)
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqs = append(reqs, r.URL.RequestURI())
		mu.Unlock()
		switch {
		case r.URL.Path == "/shelly":
			write(w, `{"type":"SHBDUO-1","mac":"98F4ABD0DCFF","fw":"`+curFW()+`","auth":false,"num_outputs":1}`)
		case r.URL.Path == "/ota":
			atomic.AddInt64(&otaCalls, 1)
			write(w, `{"status":"updating"}`)
		case r.URL.Path == "/settings" && r.URL.RawQuery == "":
			write(w, `{"fw":"`+curFW()+`","name":"x"}`)
		case r.URL.Path == "/status":
			write(w, `{"uptime":9000,"update":{"old_version":"`+curFW()+`"}}`)
		default:
			write(w, `{}`)
		}
	}))
	t.Cleanup(srv.Close)
	return gen1.NewDevice(transport.NewHTTP(srv.URL)), &reqs, &otaCalls
}

func TestRestoreGen1_NetworkOnlyBypassesGateAndWritesOnlyWiFi(t *testing.T) {
	t.Parallel()
	// Old firmware + newer backup is normally refused; NetworkOnly must bypass the
	// gate (a station write cannot trigger the loop) and write only the station
	// config — the firmware-agnostic bootstrap onto the LAN.
	dev, writes := gen1ColorDevice(t, `{"fw":"20191216-140245/???","uptime":9000}`)
	bkp := &Backup{Config: json.RawMessage(`{"name":"FR","fw":"20230913-111821/v1.14.0"}`)}

	_, err := RestoreGen1(context.Background(), dev, bkp, &Gen1RestoreOptions{
		NetworkOnly: true,
		NetworkOverride: &Gen1NetworkOverride{
			SSID: "Home", Password: "secret",
			StaticIP: "10.23.47.227", Gateway: "10.23.47.1", Netmask: "255.255.254.0",
		},
	})
	if err != nil {
		t.Fatalf("NetworkOnly restore errored (gate not bypassed?): %v", err)
	}
	var sawSta, sawForbidden bool
	for _, w := range *writes {
		if strings.HasPrefix(w, "/settings/sta?") {
			sawSta = true
		}
		if strings.HasPrefix(w, "/settings?") || strings.Contains(w, "/settings/mqtt") || strings.Contains(w, "/settings/cloud") {
			sawForbidden = true
		}
	}
	if !sawSta {
		t.Errorf("NetworkOnly did not write the station config; writes=%v", *writes)
	}
	if sawForbidden {
		t.Errorf("NetworkOnly wrote non-network settings; writes=%v", *writes)
	}
}

func TestRestoreGen1_UpdateFirmwareResolvesDowngrade(t *testing.T) {
	t.Parallel()
	oldFW := "20191216-140245/???"
	newFW := "20230913-111821/v1.14.0-gcb84623"
	dev, reqs, otaCalls := gen1FirmwareUpdateDevice(t, oldFW, newFW)
	bkp := &Backup{
		DeviceInfo: &DeviceInfo{Model: "SHBDUO-1"},
		Config:     json.RawMessage(`{"name":"FR","fw":"` + newFW + `"}`),
	}

	// UpdateFirmware turns a refused downgrade into: OTA the device to matched
	// firmware, then restore — no error, and the config writes proceed.
	_, err := RestoreGen1(context.Background(), dev, bkp,
		&Gen1RestoreOptions{UpdateFirmware: true, SkipNetwork: true, SkipAuth: true})
	if err != nil {
		t.Fatalf("RestoreGen1 with UpdateFirmware: %v", err)
	}
	if atomic.LoadInt64(otaCalls) == 0 {
		t.Fatal("expected an OTA update to be triggered before the restore")
	}

	wantURL := url.QueryEscape("http://firmware.shelly.cloud/gen1/SHBDUO-1.zip")
	var sawOTAURL, sawConfigWrite bool
	for _, u := range *reqs {
		if strings.HasPrefix(u, "/ota?") && strings.Contains(u, wantURL) {
			sawOTAURL = true
		}
		if strings.HasPrefix(u, "/settings?") {
			sawConfigWrite = true
		}
	}
	if !sawOTAURL {
		t.Errorf("OTA not triggered to the derived model URL; requests=%v", *reqs)
	}
	if !sawConfigWrite {
		t.Errorf("restore did not proceed to write config after the update; requests=%v", *reqs)
	}
}

func TestRestoreGen1_UpdateFirmwareWithoutModelErrors(t *testing.T) {
	t.Parallel()
	// UpdateFirmware on a downgrade with no model to derive a URL from (and no
	// explicit FirmwareURL) must fail loudly, not POST a malformed URL.
	dev, _ := gen1ColorDevice(t, `{"fw":"20191216-140245/???","uptime":9000}`)
	bkp := &Backup{Config: json.RawMessage(`{"name":"FR","fw":"20230913-111821/v1.14.0"}`)}

	_, err := RestoreGen1(context.Background(), dev, bkp, &Gen1RestoreOptions{UpdateFirmware: true})
	if err == nil || !strings.Contains(err.Error(), "cannot update firmware") {
		t.Fatalf("expected a no-model firmware-URL error, got: %v", err)
	}
}

func TestUpdateGen1FirmwareAndWait_ReturnsWhenBuildChanges(t *testing.T) {
	t.Parallel()
	oldFW, newFW := "20191216-140245/???", "20230913-111821/v1.14.0"
	dev, _, otaCalls := gen1FirmwareUpdateDevice(t, oldFW, newFW)

	if err := updateGen1FirmwareAndWait(context.Background(), dev, "http://x/fw.zip", oldFW); err != nil {
		t.Fatalf("updateGen1FirmwareAndWait: %v", err)
	}
	if atomic.LoadInt64(otaCalls) == 0 {
		t.Error("OTA trigger was not called")
	}
}

func TestUpdateGen1FirmwareAndWait_TimesOutWhenFirmwareNeverChanges(t *testing.T) {
	t.Parallel()
	// The device never reports a new build; the wait must give up (bounded here by a
	// short context) instead of blocking on the full update budget.
	dev, _ := gen1ColorDevice(t, `{"fw":"20191216-140245/???","uptime":9000}`)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := updateGen1FirmwareAndWait(ctx, dev, "http://x/fw.zip", "20191216-140245/???")
	if err == nil {
		t.Fatal("expected an error when the firmware never changes")
	}
}
