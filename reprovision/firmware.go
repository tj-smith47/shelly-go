package reprovision

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/tj-smith47/shelly-go/backup"
	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/gen1"
)

// firmwareServePort is the port the host serves a Gen1 firmware image on during an
// at-AP update; the device fetches http://<apHostIP>:<port>/firmware.zip from it.
const firmwareServePort = 8512

// firmwareFetchTimeout bounds downloading the image from the public CDN.
const firmwareFetchTimeout = 90 * time.Second

// fetchGen1Firmware downloads the Gen1 firmware image at url to a temp file and
// returns its path. It runs before the host hops onto the device's AP, while the
// host still has internet; the factory AP itself has none. The caller removes the
// returned file when done.
func (r *runner) fetchGen1Firmware(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, firmwareFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("build firmware request: %w", err)
	}
	resp, err := r.firmwareClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download firmware from %s: %w", url, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			r.log.Debug("firmware: close download body", "error", cerr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download firmware from %s: unexpected status %s", url, resp.Status)
	}

	f, err := os.CreateTemp("", "shelly-gen1-fw-*.zip")
	if err != nil {
		return "", fmt.Errorf("create firmware temp file: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		if cerr := f.Close(); cerr != nil {
			r.log.Debug("firmware: close temp after copy error", "error", cerr)
		}
		r.removeFirmwareTemp(f.Name())
		return "", fmt.Errorf("write firmware temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		r.removeFirmwareTemp(f.Name())
		return "", fmt.Errorf("close firmware temp file: %w", err)
	}
	return f.Name(), nil
}

// apFirmwareBindIP resolves the host's actual AP-subnet address for serving the
// firmware image. An empty apHostIP means the scanner keeps
// discovery.DefaultAPHostIP, so the same resolution must happen here; otherwise
// the served URL would carry no host (http://:8512/...) and the device could
// never fetch the image.
func apFirmwareBindIP(apHostIP string) string {
	if apHostIP == "" {
		return discovery.DefaultAPHostIP
	}
	return apHostIP
}

// prefetchAPFirmware downloads, before the host hops onto the device's factory AP,
// the Gen1 firmware image that may need flashing there. It resolves the image URL
// (firmwareURL, else one derived from the backup's model) and downloads it now,
// while the host still has internet. It is best-effort: it returns an empty path
// whenever no image can or should be staged (a Gen2 target, a forced downgrade, an
// underivable URL, or a failed download), leaving the at-AP check to fail loudly
// only if an update turns out to be required. The caller removes the returned file.
func (r *runner) prefetchAPFirmware(
	ctx context.Context,
	generation int,
	model, firmwareURL string,
	allowDowngrade bool,
) string {
	// Whether the device needs an update is only knowable at the AP (its live
	// build cannot be read until the host has hopped onto it), and the factory AP
	// has no internet, so the image is fetched now and the at-AP check decides
	// whether to flash. A forced downgrade means no update, so nothing is fetched.
	if generation != 1 || allowDowngrade {
		return ""
	}
	fwURL := firmwareURL
	if fwURL == "" {
		fwURL = backup.OfficialGen1FirmwareURL(model)
	}
	if fwURL == "" {
		r.log.Debug("firmware: no URL derivable from model; skipping prefetch", "model", model)
		return ""
	}
	// A download hiccup must not block a restore that may not even need an
	// update; if one is needed and the image is missing, the at-AP check fails.
	fwPath, err := r.fetchGen1Firmware(ctx, fwURL)
	if err != nil {
		r.log.Debug("firmware: prefetch failed; the at-AP check decides if it mattered", "url", fwURL, "error", err)
		return ""
	}
	return fwPath
}

// removeFirmwareTemp deletes a downloaded firmware temp file, logging a failure
// rather than returning it.
func (r *runner) removeFirmwareTemp(path string) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		r.log.Debug("firmware: remove temp image", "path", path, "error", err)
	}
}

// serveFirmwareFile starts an HTTP server bound to bindIP that serves the image at
// path as /firmware.zip, returning the URL the device should fetch and a stop
// function. A device at its factory AP has no internet, so the image is served
// from the host's own address on the AP subnet.
func (r *runner) serveFirmwareFile(ctx context.Context, bindIP, path string) (url string, stop func(), err error) {
	addr := net.JoinHostPort(bindIP, strconv.Itoa(firmwareServePort))
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return "", nil, fmt.Errorf("listen on %s to serve firmware: %w", addr, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/firmware.zip", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, path)
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			r.log.Debug("firmware: serve", "error", serveErr)
		}
	}()

	stop = func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutErr := srv.Shutdown(shutdownCtx); shutErr != nil {
			r.log.Debug("firmware: shut down serve", "error", shutErr)
		}
	}
	return "http://" + addr + "/firmware.zip", stop, nil
}

// ensureGen1FirmwareAtAP brings the device at its factory AP up to the backup's
// firmware before the config restore when its build is older than the backup's,
// notably a corrupt build that reboot-loops the instant WiFi station mode is
// active and so can never complete an OTA once on the LAN. It is a no-op when the
// device is already at or beyond the backup's firmware, and when allowDowngrade
// forces the older config write.
//
// The image (prefetched to fwPath before the hop) is served from the host's
// AP-subnet address and the device is pointed at it. If an update is needed but
// no image was prefetched, this fails rather than writing a station config that
// would reboot-loop the device onto a dead LAN.
func (r *runner) ensureGen1FirmwareAtAP(
	ctx context.Context,
	bindIP, fwPath, backupFW string,
	allowDowngrade bool,
) error {
	r.step("checking firmware")
	return r.withGen1(ctx, r.apAddr, "", func(dev *gen1.Device, _ string) error {
		liveFW := backup.Gen1LiveFirmware(ctx, dev)
		if liveFW == "" {
			return fmt.Errorf("%w: could not read the device's firmware at its AP to decide on an update",
				ErrFirmwareUpdate)
		}
		if allowDowngrade || !backup.Gen1FirmwareDowngrade(liveFW, backupFW) {
			r.log.Debug("firmware: no update", "live", liveFW, "backup", backupFW, "allowDowngrade", allowDowngrade)
			return nil
		}
		if fwPath == "" {
			return &FirmwareUnavailableError{Current: liveFW, Required: backupFW}
		}
		fwURL, stop, err := r.serveFirmwareFile(ctx, bindIP, fwPath)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFirmwareUpdate, err)
		}
		defer stop()
		r.step("updating firmware")
		r.log.Debug("firmware: serving image for OTA", "url", fwURL, "live", liveFW, "backup", backupFW)
		if updErr := backup.UpdateGen1FirmwareAndWait(ctx, dev, fwURL, liveFW); updErr != nil {
			return fmt.Errorf("%w: %w", ErrFirmwareUpdate, updErr)
		}
		return nil
	})
}

// confirmGen1StableAtAP verifies the Gen1 device is booted and holding at its
// factory AP, not caught in a reboot loop, immediately before the station config
// write. That write reboots the device, and on firmware that cannot survive
// station mode it would strand it off the LAN; the device is still on its
// recoverable AP here, so a failure aborts with the device intact.
func (r *runner) confirmGen1StableAtAP(ctx context.Context) error {
	r.step("confirming the device is stable")
	return r.withGen1(ctx, r.apAddr, "", func(dev *gen1.Device, _ string) error {
		uptime, required, stable := backup.Gen1ConfirmStable(ctx, dev)
		if !stable {
			return fmt.Errorf(
				"%w: refusing to write the station config: the device is not holding a stable uptime "+
					"at its factory AP (highest uptime %ds, need %ds held, the signature of a reboot loop); "+
					"writing it now would reboot the device onto firmware it cannot hold and strand it off "+
					"the LAN. The device remains on its recoverable factory AP",
				ErrUnstable, uptime, required)
		}
		r.log.Debug("firmware: device stable at AP", "uptime", uptime, "required", required)
		return nil
	})
}
