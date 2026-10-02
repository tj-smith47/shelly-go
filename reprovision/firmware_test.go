package reprovision

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
)

const fakeFirmwareBody = "PK\x03\x04 fake gen1 firmware image"

// fakeFirmwareServer serves fakeFirmwareBody at any path, as the Shelly CDN does
// for a Gen1 image.
func fakeFirmwareServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.WriteString(w, fakeFirmwareBody); err != nil {
			t.Errorf("serve fake firmware: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// stageFirmwareImage downloads a fake image to a temp file and removes it on
// cleanup. The serve path reads the real filesystem, so the image must exist.
func stageFirmwareImage(t *testing.T, r *runner) string {
	t.Helper()
	src := fakeFirmwareServer(t)
	path, err := r.fetchGen1Firmware(context.Background(), src.URL+"/fw.zip")
	if err != nil {
		t.Fatalf("stage firmware image: %v", err)
	}
	t.Cleanup(func() { r.removeFirmwareTemp(path) })
	return path
}

func TestFetchGen1Firmware_DownloadsToTemp(t *testing.T) {
	t.Parallel()
	r := testRunner(t, nil, "")
	path := stageFirmwareImage(t, r)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read temp image: %v", err)
	}
	if string(got) != fakeFirmwareBody {
		t.Errorf("temp image = %q, want %q", got, fakeFirmwareBody)
	}
}

func TestFetchGen1Firmware_Errors(t *testing.T) {
	t.Parallel()
	r := testRunner(t, nil, "")
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	t.Cleanup(notFound.Close)
	truncated := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "4096")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, "PK\x03\x04 partial"); err != nil {
			return
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			return
		}
		if cerr := conn.Close(); cerr != nil {
			t.Logf("close hijacked conn: %v", cerr)
		}
	}))
	t.Cleanup(truncated.Close)

	for name, url := range map[string]string{
		"non-200":       notFound.URL,
		"truncated":     truncated.URL + "/fw.zip",
		"malformed URL": "://not a url",
		"unreachable":   "http://" + refusingAddr(t) + "/fw.zip",
	} {
		if _, err := r.fetchGen1Firmware(context.Background(), url); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// Not parallel: it changes TMPDIR for the process.
func TestFetchGen1Firmware_TempCreateError(t *testing.T) {
	srv := fakeFirmwareServer(t)
	notDir := t.TempDir() + "/afile"
	if err := os.WriteFile(notDir, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed non-dir: %v", err)
	}
	t.Setenv("TMPDIR", notDir+"/nope")
	if _, err := testRunner(t, nil, "").fetchGen1Firmware(context.Background(), srv.URL+"/fw.zip"); err == nil {
		t.Fatal("expected a temp-file creation error when TMPDIR is unusable")
	}
}

func TestRemoveFirmwareTemp(t *testing.T) {
	t.Parallel()
	r := testRunner(t, nil, "")
	r.removeFirmwareTemp("")
	r.removeFirmwareTemp(t.TempDir() + "/missing.zip")

	dir := t.TempDir()
	if err := os.WriteFile(dir+"/keep", []byte("x"), 0o600); err != nil {
		t.Fatalf("seed dir: %v", err)
	}
	r.removeFirmwareTemp(dir)

	f, err := os.CreateTemp(t.TempDir(), "shelly-gen1-fw-*.zip")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	if cerr := f.Close(); cerr != nil {
		t.Fatalf("close temp: %v", cerr)
	}
	r.removeFirmwareTemp(f.Name())
	if _, statErr := os.Stat(f.Name()); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("temp image still present: %v", statErr)
	}
}

// Not parallel: the firmware tests that serve share the fixed port 8512.
func TestServeFirmwareFile_ServesImage(t *testing.T) {
	r := testRunner(t, nil, "")
	imgPath := stageFirmwareImage(t, r)
	url, stop, err := r.serveFirmwareFile(context.Background(), "127.0.0.1", imgPath)
	if err != nil {
		t.Fatalf("serveFirmwareFile: %v", err)
	}
	defer stop()
	if !strings.HasSuffix(url, "/firmware.zip") {
		t.Errorf("served URL %q does not end in /firmware.zip", url)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			t.Errorf("close body: %v", cerr)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read served body: %v", err)
	}
	if string(body) != fakeFirmwareBody {
		t.Errorf("served body = %q, want %q", body, fakeFirmwareBody)
	}
}

func TestServeFirmwareFile_ListenError(t *testing.T) {
	t.Parallel()
	// 192.0.2.1 (TEST-NET-1) is not a local address, so it cannot be bound.
	_, stop, err := testRunner(t, nil, "").serveFirmwareFile(context.Background(), "192.0.2.1", "unused.zip")
	if err == nil {
		stop()
		t.Fatal("expected a listen error binding an unassignable address")
	}
}

func TestAPFirmwareBindIP(t *testing.T) {
	t.Parallel()
	if got := apFirmwareBindIP(""); got != discovery.DefaultAPHostIP {
		t.Errorf("apFirmwareBindIP(\"\") = %q, want %q", got, discovery.DefaultAPHostIP)
	}
	if got := apFirmwareBindIP("192.168.33.150"); got != "192.168.33.150" {
		t.Errorf("apFirmwareBindIP(explicit) = %q, want it unchanged", got)
	}
}

func TestPrefetchAPFirmware(t *testing.T) {
	t.Parallel()
	r := testRunner(t, nil, "")
	ctx := context.Background()
	if p := r.prefetchAPFirmware(ctx, 2, fakeGen1Model, "", false); p != "" {
		t.Errorf("gen2: path = %q, want empty", p)
	}
	if p := r.prefetchAPFirmware(ctx, 1, fakeGen1Model, "", true); p != "" {
		t.Errorf("allow downgrade: path = %q, want empty", p)
	}
	if p := r.prefetchAPFirmware(ctx, 1, "", "", false); p != "" {
		t.Errorf("no model: path = %q, want empty", p)
	}
	if p := r.prefetchAPFirmware(ctx, 1, "", "http://"+refusingAddr(t)+"/fw.zip", false); p != "" {
		t.Errorf("download failure: path = %q, want empty", p)
	}

	srv := fakeFirmwareServer(t)
	p := r.prefetchAPFirmware(ctx, 1, "", srv.URL+"/fw.zip", false)
	defer r.removeFirmwareTemp(p)
	got, err := os.ReadFile(p)
	if err != nil || string(got) != fakeFirmwareBody {
		t.Errorf("explicit URL: image = %q, %v; want the downloaded image", got, err)
	}
}

func TestEnsureGen1FirmwareAtAP_NoUpdate(t *testing.T) {
	t.Parallel()
	t.Run("device current", func(t *testing.T) {
		t.Parallel()
		d := newFakeDevice(t, 1)
		d.setFW("20230101-000000/v2.0")
		if err := testRunner(t, nil, d.addr()).ensureGen1FirmwareAtAP(context.Background(), "127.0.0.1", "",
			"20210101-000000/v1.0", false); err != nil {
			t.Fatalf("ensureGen1FirmwareAtAP: %v", err)
		}
		if d.hits("/ota") != 0 {
			t.Error("an OTA was triggered on a current device")
		}
	})
	t.Run("downgrade allowed", func(t *testing.T) {
		t.Parallel()
		d := newFakeDevice(t, 1)
		d.setFW("20200101-000000/v0.9")
		if err := testRunner(t, nil, d.addr()).ensureGen1FirmwareAtAP(context.Background(), "127.0.0.1", "",
			"20210601-000000/v1.5", true); err != nil {
			t.Fatalf("ensureGen1FirmwareAtAP: %v", err)
		}
		if d.hits("/ota") != 0 {
			t.Error("an OTA was triggered although the downgrade was allowed")
		}
	})
}

func TestEnsureGen1FirmwareAtAP_Refusals(t *testing.T) {
	t.Parallel()
	t.Run("update needed without an image", func(t *testing.T) {
		t.Parallel()
		d := newFakeDevice(t, 1)
		d.setFW("20200101-000000/v0.9")
		err := testRunner(t, nil, d.addr()).ensureGen1FirmwareAtAP(context.Background(), "127.0.0.1", "",
			"20210601-000000/v1.5", false)
		if !errors.Is(err, ErrFirmwareUnavailable) || !strings.Contains(err.Error(), "no firmware image") {
			t.Fatalf("err = %v, want it to name the missing firmware image", err)
		}
	})
	t.Run("unreadable firmware", func(t *testing.T) {
		t.Parallel()
		d := newFakeDevice(t, 1)
		d.settingsErr = true
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		err := testRunner(t, nil, d.addr()).ensureGen1FirmwareAtAP(ctx, "127.0.0.1", "", "20210601-000000/v1.5", false)
		if !errors.Is(err, ErrFirmwareUpdate) {
			t.Fatalf("err = %v, want an ErrFirmwareUpdate for unreadable firmware", err)
		}
	})
	t.Run("unreachable device", func(t *testing.T) {
		t.Parallel()
		err := testRunner(t, nil, refusingAddr(t)).ensureGen1FirmwareAtAP(context.Background(), "127.0.0.1", "",
			"20210601-000000/v1.5", false)
		if err == nil {
			t.Fatal("expected a connection error")
		}
	})
	t.Run("serve bind failure", func(t *testing.T) {
		t.Parallel()
		d := newFakeDevice(t, 1)
		d.setFW("20200101-000000/v0.9")
		r := testRunner(t, nil, d.addr())
		err := r.ensureGen1FirmwareAtAP(context.Background(), "192.0.2.1", stageFirmwareImage(t, r),
			"20210601-000000/v1.5", false)
		if !errors.Is(err, ErrFirmwareUpdate) {
			t.Fatalf("err = %v, want the serve-bind failure as ErrFirmwareUpdate", err)
		}
		if d.hits("/ota") != 0 {
			t.Error("an OTA was triggered although the image was never served")
		}
	})
}

// Not parallel: serves on the fixed port 8512.
func TestEnsureGen1FirmwareAtAP_FlashesAndWaits(t *testing.T) {
	const backupFW = "20210601-000000/v1.5"
	d := newFakeDevice(t, 1)
	d.setFW("20200101-000000/v0.9")
	d.otaFlipFW = backupFW
	r := testRunner(t, nil, d.addr())
	if err := r.ensureGen1FirmwareAtAP(context.Background(), "127.0.0.1", stageFirmwareImage(t, r),
		backupFW, false); err != nil {
		t.Fatalf("ensureGen1FirmwareAtAP: %v", err)
	}
	if d.hits("/ota") == 0 {
		t.Error("the device's /ota endpoint was never triggered")
	}
	if got := d.currentFW(); got != backupFW {
		t.Errorf("firmware after flash = %q, want %q", got, backupFW)
	}
}

// Not parallel: serves on the fixed port 8512.
func TestEnsureGen1FirmwareAtAP_OTAFailurePropagates(t *testing.T) {
	d := newFakeDevice(t, 1)
	d.setFW("20200101-000000/v0.9")
	d.otaErr = true
	r := testRunner(t, nil, d.addr())
	img := stageFirmwareImage(t, r)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := r.ensureGen1FirmwareAtAP(ctx, "127.0.0.1", img, "20210601-000000/v1.5", false)
	if !errors.Is(err, ErrFirmwareUpdate) {
		t.Fatalf("err = %v, want ErrFirmwareUpdate when the OTA never lands the new build", err)
	}
}

func TestConfirmGen1StableAtAP(t *testing.T) {
	t.Parallel()
	t.Run("stable", func(t *testing.T) {
		t.Parallel()
		d := newFakeDevice(t, 1)
		if err := testRunner(t, nil, d.addr()).confirmGen1StableAtAP(context.Background()); err != nil {
			t.Fatalf("confirmGen1StableAtAP: %v", err)
		}
	})
	t.Run("unstable", func(t *testing.T) {
		t.Parallel()
		d := newFakeDevice(t, 1)
		d.uptime = 1
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		err := testRunner(t, nil, d.addr()).confirmGen1StableAtAP(ctx)
		if !errors.Is(err, ErrUnstable) || !strings.Contains(err.Error(), "refusing to write") {
			t.Fatalf("err = %v, want an ErrUnstable refusal", err)
		}
	})
}

func TestLoopbackOnlyBlocksTheCDN(t *testing.T) {
	t.Parallel()
	var blocked []string
	r := testRunner(t, nil, "")
	r.firmwareClient = &http.Client{Transport: loopbackOnly{fail: func(format string, args ...any) {
		blocked = append(blocked, fmt.Sprintf(format, args...))
	}}}
	if p := r.prefetchAPFirmware(context.Background(), 1, fakeGen1Model, "", false); p != "" {
		t.Errorf("path = %q, want empty when the CDN download is blocked", p)
	}
	if len(blocked) != 1 || !strings.Contains(blocked[0], "firmware.shelly.cloud") {
		t.Errorf("blocked = %q, want the derived CDN URL refused", blocked)
	}
	path := stageFirmwareImage(t, r)
	if path == "" {
		t.Error("a loopback download was blocked")
	}
}

func TestEnsureGen1FirmwareAtAP_UnavailableNamesBothVersions(t *testing.T) {
	t.Parallel()
	d := newFakeDevice(t, 1)
	d.setFW("20200101-000000/v0.9")
	err := testRunner(t, nil, d.addr()).ensureGen1FirmwareAtAP(context.Background(), "127.0.0.1", "",
		"20210601-000000/v1.5", false)
	var fwErr *FirmwareUnavailableError
	if !errors.As(err, &fwErr) {
		t.Fatalf("err = %v, want a *FirmwareUnavailableError", err)
	}
	if fwErr.Current != "20200101-000000/v0.9" || fwErr.Required != "20210601-000000/v1.5" {
		t.Errorf("versions = %q -> %q, want the device's and the backup's", fwErr.Current, fwErr.Required)
	}
	wrapped := fmt.Errorf("restore at AP: %w", err)
	if !errors.Is(wrapped, ErrFirmwareUnavailable) || !errors.As(wrapped, &fwErr) {
		t.Errorf("wrapped error %v lost ErrFirmwareUnavailable or the typed error", wrapped)
	}
}
