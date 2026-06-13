package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/tj-smith47/shelly-go/gen1"
)

// Gen1NetworkOverride replaces the WiFi station settings applied during a Gen1
// restore. It lets one device's configuration be cloned onto another without
// copying the source's IP address, so both devices stay online with distinct
// addresses. SSID and Password are optional: when empty, the backup's own
// station credentials are kept.
type Gen1NetworkOverride struct {
	// SSID overrides the station SSID; empty keeps the backup's SSID.
	SSID string
	// Password overrides the station key; empty keeps the backup's key.
	Password string
	// StaticIP, when set, switches the station to a static IPv4 address.
	// Gateway and Netmask are required alongside it.
	StaticIP string
	// Gateway is the static IPv4 default gateway.
	Gateway string
	// Netmask is the static IPv4 subnet mask.
	Netmask string
	// DNS is the static IPv4 nameserver (optional).
	DNS string
}

// IsStatic reports whether a static IPv4 address was requested.
func (o *Gen1NetworkOverride) IsStatic() bool {
	return o != nil && o.StaticIP != ""
}

// Gen1RestoreOptions configures a Gen1 restore.
type Gen1RestoreOptions struct {
	// NetworkOverride, when non-nil, replaces the backup's WiFi station settings
	// before they are applied. Ignored when SkipNetwork is true.
	NetworkOverride *Gen1NetworkOverride
	// StepTrace, when non-nil, receives one human-readable line per restore step
	// recording the step's warnings/errors and the device's post-write uptime and
	// stability. It is the diagnostic seam for pinpointing which setting drives a
	// fragile device into a reboot loop: leave it nil for a normal restore (zero
	// overhead) and point it at a file from a debug flag to capture the trace.
	StepTrace io.Writer
	// Name overrides the device's stored display name. Empty leaves the name as
	// the backup's. Used so a cloned device is named distinctly from its source.
	Name string
	// FirmwareURL overrides the firmware image UpdateFirmware flashes. Empty derives
	// the official current-stable URL from the backup's device model.
	FirmwareURL string
	// SkipNetwork skips WiFi configuration.
	SkipNetwork bool
	// SkipAuth skips authentication configuration.
	SkipAuth bool
	// SkipState skips restoring captured live component state — color temperature
	// and brightness — so a restore leaves the target's current light look intact
	// and applies configuration only.
	SkipState bool
	// SkipMeters skips restoring meter / energy-meter configuration (e.g. Gen1
	// overpower limits), leaving the target's protection settings untouched.
	SkipMeters bool
	// SkipWebhooks skips webhook (action URL) restoration.
	SkipWebhooks bool
	// ClockDependentOnly restores only the configuration a device rejects while
	// it has no clock — light component config (schedules) and captured light
	// state (color temperature, brightness). It is used for the second pass of a
	// factory-AP restore: the first pass applies everything at the clockless AP
	// (where these writes fail with "Timezone and time should be set"), and once
	// the device has joined the LAN and obtained NTP time this pass re-applies
	// just those, without re-thrashing settings that already took at the AP.
	ClockDependentOnly bool
	// AllowFirmwareDowngrade overrides the firmware-downgrade refusal. By default a
	// restore refuses when the target device's firmware predates the firmware the
	// backup was captured from, because applying a newer firmware's configuration
	// onto older firmware can drive the device into a reboot loop. Set this only
	// after updating the device firmware is not an option and the risk is accepted.
	AllowFirmwareDowngrade bool
	// UpdateFirmware resolves a firmware downgrade by updating the device instead of
	// refusing: when the backup was captured from newer firmware than the device
	// runs, the device is OTA-updated to current stable firmware (FirmwareURL, or
	// derived from the backup's model) before any configuration is applied, so the
	// full restore lands on matched firmware and cannot reboot-loop. The device must
	// be able to reach the firmware host, which an isolated factory AP cannot — so
	// this is honored only on a LAN restore, never paired with a factory-AP pass.
	UpdateFirmware bool
	// NetworkOnly writes only the WiFi station configuration and returns, skipping
	// every other step and the firmware-downgrade gate (a station write is firmware
	// agnostic and cannot trigger the config-storm loop). It is the bootstrap pass
	// of a factory-AP restore: it moves a device from its AP onto the LAN with the
	// restored credentials, after which the device — now reachable and updatable —
	// receives its firmware update and full configuration on the LAN.
	NetworkOnly bool
}

// gen1IPv4ModeStatic is the value used to request static IPv4 addressing on Gen1
// WiFi station config (ipv4_method).
const gen1IPv4ModeStatic = "static"

// ExportGen1 builds a Backup from a Gen1 device by reading its settings and
// actions. The backup is stored in the same Backup struct used for Gen2, with
// DeviceInfo.Generation=1. DeviceInfo is populated from the device's /shelly
// response; callers may enrich it further afterward.
func ExportGen1(ctx context.Context, dev *gen1.Device) (*Backup, error) {
	bkp := &Backup{
		Version:   BackupVersion,
		CreatedAt: time.Now().UTC(),
	}

	// Get device info
	info, err := dev.GetDeviceInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}
	bkp.DeviceInfo = &DeviceInfo{
		Model:      info.Model,
		MAC:        info.MAC,
		Version:    info.Version,
		Generation: 1,
	}

	// Get full settings (this is Gen1's equivalent of Shelly.GetConfig)
	settings, err := dev.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Store full settings as Config
	configData, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}
	bkp.Config = configData

	// Extract WiFi settings
	bkp.WiFi = marshalGen1WiFi(settings)

	// Extract MQTT settings
	if settings.MQTT != nil {
		bkp.MQTT = mustMarshal(settings.MQTT)
	}

	// Extract Cloud settings
	if settings.Cloud != nil {
		bkp.Cloud = mustMarshal(settings.Cloud)
	}

	// Extract Auth info
	if settings.Login != nil {
		bkp.Auth = &AuthInfo{
			User:   settings.Login.Username,
			Enable: settings.Login.Enabled,
		}
	}

	// Extract component settings
	bkp.Components = marshalGen1Components(settings)

	// Capture live light/color/white state — the brightness, color temperature,
	// RGB and per-channel values the device keeps in /light, /color and /white
	// rather than /settings, which would otherwise be lost on restore.
	captureGen1LiveState(ctx, dev, settings, bkp)

	// Extract schedule rules from relay/light settings
	bkp.Schedules = marshalGen1Schedules(settings)

	// Get action URLs (Gen1's equivalent of webhooks)
	actions, err := dev.GetActions(ctx)
	if err == nil && actions != nil {
		bkp.Webhooks = mustMarshal(actions)
	}

	return bkp, nil
}

// Gen1 firmware older than this build date (the YYYYMMDD prefix of the
// /settings "fw" string) predates the firmware's tolerance for rapid
// consecutive settings writes. A full restore issues ~15 writes in sequence —
// several of which restart the device — and on legacy firmware that write storm
// crashes the device into a reboot loop (uptime resets every few seconds, the
// device keeps its half-applied config and never settles). Such firmware is
// paced more aggressively. The cutoff is conservative: anything older than, or
// with an unreadable, build date is treated as legacy.
const gen1LegacyFirmwareDate = 20210101

// Settle floors and recovery bounds for pacing consecutive Gen1 settings
// writes. These are deliberately not user-tunable: a restore that bricks a
// device into a boot loop is never an acceptable default, so the pacing that
// prevents it is always on.
const (
	gen1SettleModern = 750 * time.Millisecond
	gen1SettleLegacy = 2 * time.Second
	// gen1StableUptime is the uptime (seconds) a device must report before it is
	// judged stable — evidence it finished booting and is holding, rather than
	// being caught mid-restart from the previous write. It is deliberately set
	// above a reboot loop's period (a configuration-storm loop resets every
	// ~5-9s): a looping device momentarily reads a low uptime each cycle but can
	// never sustain one this high, so requiring it cleanly separates "booted and
	// holding" from "looping."
	gen1StableUptime = 12
	// gen1RecoveryBudget bounds the wait for a device to climb back to a stable
	// uptime after a write that restarted it, so a device that never recovers
	// lets the restore proceed (and fail loudly downstream) instead of hanging.
	gen1RecoveryBudget = 45 * time.Second
	gen1RecoveryPoll   = 1 * time.Second
)

// gen1Pacer throttles consecutive settings writes so a Gen1 device is never
// handed a new write while it is still applying — or rebooting from — the
// previous one. That back-to-back write storm is what drives an unpaced full
// restore into the configuration-storm boot loop.
type gen1Pacer struct {
	settle time.Duration
}

// gen1LiveFirmware reads the device's current firmware string, returning "" when
// the device cannot be read. An empty string sorts as oldest/legacy everywhere it
// is used (pacing and the downgrade gate), so an unreadable device is handled
// conservatively rather than optimistically.
func gen1LiveFirmware(ctx context.Context, dev *gen1.Device) string {
	settings, err := dev.GetSettings(ctx)
	if err != nil {
		return ""
	}
	return settings.FW
}

// gen1SettleFor selects a settle interval from a firmware build date. Firmware
// older than gen1LegacyFirmwareDate — or whose date is unparseable — is treated
// as legacy (slower, safer): being unable to confirm modern firmware is itself a
// reason to pace conservatively.
func gen1SettleFor(fw string) time.Duration {
	if parseGen1FirmwareDate(fw) < gen1LegacyFirmwareDate {
		return gen1SettleLegacy
	}
	return gen1SettleModern
}

// gen1FirmwareDowngrade reports whether applying a backup captured from backupFW
// onto a device running liveFW is a firmware downgrade — the target's firmware
// predates the backup's. Restoring a newer firmware's richer configuration onto
// older firmware is the proven trigger for the Gen1 configuration-storm boot loop
// (the older firmware mishandles fields the newer one wrote), so the restore
// refuses it by default. Returns false when either firmware date is unknown: an
// unparseable date is not affirmative evidence of a downgrade.
func gen1FirmwareDowngrade(liveFW, backupFW string) bool {
	live := parseGen1FirmwareDate(liveFW)
	backup := parseGen1FirmwareDate(backupFW)
	return live > 0 && backup > 0 && live < backup
}

// officialGen1FirmwareHost serves Shelly's public Gen1 firmware archive. Each
// model's current stable build is at /gen1/<MODEL>.zip over plain HTTP — the
// device fetches it itself, and HTTP avoids a TLS handshake a device with no clock
// set (a just-reset Gen1) cannot complete.
const officialGen1FirmwareHost = "http://firmware.shelly.cloud"

// officialGen1FirmwareURL returns the public current-stable firmware URL for a
// Gen1 model (e.g. "SHBDUO-1" → ".../gen1/SHBDUO-1.zip"). It returns "" for an
// empty model so the caller fails loudly rather than POSTing a malformed URL.
func officialGen1FirmwareURL(model string) string {
	if model == "" {
		return ""
	}
	return officialGen1FirmwareHost + "/gen1/" + model + ".zip"
}

// backupModel returns the device model recorded in a backup, or "" when the
// backup carries no device info (so a derived firmware URL is skipped rather than
// malformed).
func backupModel(bkp *Backup) string {
	if bkp == nil || bkp.DeviceInfo == nil {
		return ""
	}
	return bkp.DeviceInfo.Model
}

// Bounds for a pre-restore Gen1 OTA. A LAN OTA download+flash+reboot is ~1-2 min;
// the budget is generous so a slow download is not cut short, and bounded so a
// device that never returns fails loudly instead of hanging the restore.
const (
	gen1FirmwareUpdateBudget = 5 * time.Minute
	gen1FirmwareUpdatePoll   = 3 * time.Second
)

// updateGen1FirmwareAndWait triggers an OTA to fwURL and blocks until the device
// reboots onto a build different from priorFW (proof the flash took) and holds a
// stable uptime, or the budget elapses. The device must reach the firmware host —
// true on the LAN, never at an isolated factory AP — so callers run this only on a
// LAN restore. The trigger call commonly errors as the device drops the connection
// to begin flashing; that is expected, and only fatal if the device never leaves
// priorFW, which the wait detects. Transient read errors during the download/reboot
// window are tolerated; only a budget timeout (with no observed build change) fails.
func updateGen1FirmwareAndWait(ctx context.Context, dev *gen1.Device, fwURL, priorFW string) error {
	triggerErr := dev.Update(ctx, fwURL)

	deadline := time.Now().Add(gen1FirmwareUpdateBudget)
	updated := false
	for {
		// The build string is the definitive signal the flash took; read it the same
		// way the gate does. Once it changes, confirm a held uptime so the restore is
		// not handed a device still mid-reboot from the flash.
		if fw := gen1LiveFirmware(ctx, dev); fw != "" && fw != priorFW {
			updated = true
			if status, err := dev.GetFullStatus(ctx); err == nil && status.Uptime >= gen1StableUptime {
				return nil
			}
		}
		if time.Now().After(deadline) || !sleepCtx(ctx, gen1FirmwareUpdatePoll) {
			if updated {
				// New firmware booted; uptime just had not crossed the stability bar
				// within the budget. The build changed, which is the success signal.
				return nil
			}
			if triggerErr != nil {
				return fmt.Errorf("firmware update to %q never started (device still on %q): %w",
					fwURL, priorFW, triggerErr)
			}
			return fmt.Errorf("firmware update to %q did not complete within %s (device still on %q)",
				fwURL, gen1FirmwareUpdateBudget, priorFW)
		}
	}
}

// maybeUpdateGen1Firmware brings the device up to the backup's firmware before a
// restore when UpdateFirmware is set and the device runs older firmware than the
// backup captured — the clean alternative to refusing the restore or forcing a
// reboot-looping downgrade. It returns the device's firmware after the attempt
// (the unchanged liveFW when no update was needed). The device must reach the
// firmware host, so this only succeeds on a LAN restore, never at a factory AP.
func maybeUpdateGen1Firmware(
	ctx context.Context,
	dev *gen1.Device,
	bkp *Backup,
	opts *Gen1RestoreOptions,
	liveFW, backupFW string,
) (string, error) {
	if !opts.UpdateFirmware || !gen1FirmwareDowngrade(liveFW, backupFW) {
		return liveFW, nil
	}
	fwURL := opts.FirmwareURL
	if fwURL == "" {
		fwURL = officialGen1FirmwareURL(backupModel(bkp))
	}
	if fwURL == "" {
		return liveFW, fmt.Errorf(
			"cannot update firmware: the backup carries no device model to derive a firmware " +
				"URL from and none was supplied — set FirmwareURL")
	}
	if err := updateGen1FirmwareAndWait(ctx, dev, fwURL, liveFW); err != nil {
		return liveFW, fmt.Errorf("pre-restore firmware update failed: %w", err)
	}
	return gen1LiveFirmware(ctx, dev), nil
}

// parseGen1FirmwareDate extracts the YYYYMMDD build date that prefixes a Gen1
// "fw" string (e.g. "20230913-111821/v1.14.0-gcb84623" → 20230913). Returns 0
// when no leading 8-digit date is present, which sorts as oldest/legacy.
func parseGen1FirmwareDate(fw string) int {
	if len(fw) < 8 {
		return 0
	}
	date := 0
	for i := range 8 {
		c := fw[i]
		if c < '0' || c > '9' {
			return 0
		}
		date = date*10 + int(c-'0')
	}
	return date
}

// afterWrite paces after a settings write that leaves the device at the same
// address: it waits out the settle floor, then polls until the device reports a
// stable, held uptime (proof it is not mid-reboot) before the next write. A
// write that restarted the device drives its uptime toward zero; afterWrite
// waits for recovery, which is precisely what stops a full restore from stacking
// writes into a boot loop.
//
// It reports the last uptime observed and whether the device restabilized within
// the recovery budget. stable=false means the write left the device unable to
// hold a healthy uptime — a reboot loop — and the caller halts rather than
// writing further into a crashing device. lastUptime is the highest uptime seen
// (0 when the device was never reachable during the wait).
func (p gen1Pacer) afterWrite(ctx context.Context, dev *gen1.Device) (lastUptime int, stable bool) {
	if !sleepCtx(ctx, p.settle) {
		return 0, false
	}
	deadline := time.Now().Add(gen1RecoveryBudget)
	for {
		if status, err := dev.GetFullStatus(ctx); err == nil {
			if status.Uptime > lastUptime {
				lastUptime = status.Uptime
			}
			if status.Uptime >= gen1StableUptime {
				return status.Uptime, true
			}
		}
		if time.Now().After(deadline) || !sleepCtx(ctx, gen1RecoveryPoll) {
			return lastUptime, false
		}
	}
}

// afterNetworkWrite paces after a WiFi write, which on a LAN restore can move
// the device to a new address (static-IP change) and make it unreachable at the
// current one. It waits only the settle floor — never the uptime poll — so the
// pacer does not hang waiting for a device that has legitimately relocated; any
// restart is caught by the next afterWrite when the device is reachable again
// (at the factory AP it never moves, so the following write paces normally).
func (p gen1Pacer) afterNetworkWrite(ctx context.Context) {
	sleepCtx(ctx, p.settle)
}

// sleepCtx sleeps for d unless ctx is canceled first. It reports false when
// canceled, so callers can stop pacing promptly on shutdown.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// runGen1Step performs one same-address restore step — the write closure, then
// pacing — and judges whether the device survived it. When step tracing is
// enabled it records the step's outcome (the warnings/errors the write raised and
// the device's post-write uptime/stability), giving an exact map of which setting
// the device tolerated and which one broke it. It returns false when the write
// drove the device into a reboot loop, recording the breaking step on the result
// so the caller halts instead of stacking further writes onto a crashing device.
func runGen1Step(
	ctx context.Context,
	dev *gen1.Device,
	pacer gen1Pacer,
	opts *Gen1RestoreOptions,
	step string,
	result *RestoreResult,
	write func(),
) bool {
	warnBefore, errBefore := len(result.Warnings), len(result.Errors)
	write()
	uptime, stable := pacer.afterWrite(ctx, dev)
	traceGen1Step(opts.StepTrace, step, result, warnBefore, errBefore, uptime, stable)
	if !stable {
		result.DestabilizedStep = step
		result.Success = false
		result.Errors = append(result.Errors,
			fmt.Errorf("device became unstable after writing %s; restore halted to avoid a reboot loop", step))
	}
	return stable
}

// gen1RestoreStep is one entry in the Gen1 restore sequence: a labeled write,
// optionally skipped, optionally unprobed. probe is false for a network write
// that may relocate the device — it is paced but not polled for stability, since
// unreachability there is relocation rather than a crash.
type gen1RestoreStep struct {
	write func()
	name  string
	skip  bool
	probe bool
}

// runGen1RestoreStep runs one restore step: a probed step writes, paces, traces,
// and halts on destabilization (via runGen1Step); an unprobed step (a network
// write) writes, paces without polling, and traces as applied-not-probed. It
// reports whether the restore may continue.
func runGen1RestoreStep(
	ctx context.Context,
	dev *gen1.Device,
	pacer gen1Pacer,
	opts *Gen1RestoreOptions,
	step gen1RestoreStep,
	result *RestoreResult,
) bool {
	if step.probe {
		return runGen1Step(ctx, dev, pacer, opts, step.name, result, step.write)
	}
	warnBefore, errBefore := len(result.Warnings), len(result.Errors)
	step.write()
	pacer.afterNetworkWrite(ctx)
	traceGen1Step(opts.StepTrace, step.name, result, warnBefore, errBefore, -1, true)
	return true
}

// traceGen1Step appends one line per restore step to the trace sink, capturing
// the warnings and errors the step raised and the device's post-write uptime and
// stability. A negative uptime marks a step that could not be probed (a network
// write may relocate the device, so it is paced but not polled). It is a no-op
// when tracing is disabled (w == nil) — a normal restore pays nothing for it.
func traceGen1Step(w io.Writer, step string, result *RestoreResult, warnBefore, errBefore, uptime int, stable bool) {
	if w == nil {
		return
	}
	state := "ok"
	switch {
	case !stable:
		state = "DESTABILIZED"
	case uptime < 0:
		state = "applied (not probed)"
	}
	uptimeField := "n/a"
	if uptime >= 0 {
		uptimeField = strconv.Itoa(uptime)
	}
	fmt.Fprintf(w, "step=%-18s warnings=%d errors=%d uptime=%-5s %s\n",
		step, len(result.Warnings)-warnBefore, len(result.Errors)-errBefore, uptimeField, state)
}

// RestoreGen1 restores a Backup to a Gen1 device via individual HTTP settings
// calls. The order of operations mirrors the device's hardware-verified restore
// sequence and must not be reordered. Each write group is paced (see gen1Pacer)
// so the device is never handed a new setting while still rebooting from the
// previous one — without that pacing a full restore crashes Gen1 firmware,
// notably pre-2021 builds, into a reboot loop.
func RestoreGen1(ctx context.Context, dev *gen1.Device, bkp *Backup, opts *Gen1RestoreOptions) (*RestoreResult, error) {
	result := &RestoreResult{
		Success: true,
	}

	// Parse the full settings from backup Config
	var settings gen1.Settings
	if bkp.Config != nil {
		if err := json.Unmarshal(bkp.Config, &settings); err != nil {
			return nil, fmt.Errorf("failed to parse backup config: %w", err)
		}
	}

	// A name override replaces the backup's stored name (e.g. for a clone).
	if opts.Name != "" {
		settings.Name = opts.Name
	}

	// Read the device's live firmware once: it selects the pacing aggressiveness
	// for the whole restore (older firmware is more fragile to rapid writes) and
	// gates the restore against a firmware downgrade.
	liveFW := gen1LiveFirmware(ctx, dev)
	pacer := gen1Pacer{settle: gen1SettleFor(liveFW)}

	// NetworkOnly writes just the WiFi station config and returns — the firmware-
	// agnostic bootstrap that moves a device from its factory AP onto the LAN
	// without touching any firmware-sensitive configuration. It bypasses the
	// downgrade gate (a station write cannot trigger the config-storm loop), so a
	// device on older firmware can still be brought onto the LAN, where its firmware
	// is then updated before the full restore.
	if opts.NetworkOnly {
		runGen1RestoreStep(ctx, dev, pacer, opts, gen1RestoreStep{
			name: componentWiFi, probe: false, write: func() {
				restoreGen1WiFi(ctx, dev, bkp, opts.NetworkOverride, result)
			},
		}, result)
		return result, nil
	}

	// When a firmware update is permitted and the backup was captured from newer
	// firmware than the device runs, bring the device up to date before restoring —
	// the clean alternative to refusing, or to forcing a downgrade that reboot-loops.
	// A changed firmware re-selects pacing so the gate below sees the new version.
	updatedFW, err := maybeUpdateGen1Firmware(ctx, dev, bkp, opts, liveFW, settings.FW)
	if err != nil {
		return nil, err
	}
	if updatedFW != liveFW {
		liveFW = updatedFW
		pacer = gen1Pacer{settle: gen1SettleFor(liveFW)}
	}

	// Refuse a firmware downgrade before writing anything: applying a backup
	// captured from newer firmware onto a device running older firmware is the
	// proven trigger for the Gen1 reboot loop. The remedy is to update the
	// device's firmware first, so this fails loudly rather than bricking it.
	if !opts.AllowFirmwareDowngrade && gen1FirmwareDowngrade(liveFW, settings.FW) {
		return nil, fmt.Errorf(
			"refusing restore: device firmware %q predates the backup's firmware %q; "+
				"applying a newer firmware's configuration onto older firmware can drive the "+
				"device into a reboot loop — set UpdateFirmware to update the device first "+
				"(recommended), or AllowFirmwareDowngrade to override and accept the risk",
			liveFW, settings.FW)
	}

	// The factory-AP restore's LAN second pass re-applies only the clock-gated
	// config (light component config + captured light state), now that the device
	// has a clock, without re-writing settings that already took at the AP.
	if opts.ClockDependentOnly {
		restoreGen1ClockDependent(ctx, dev, pacer, opts, &settings, bkp, result)
		return result, nil
	}

	// The restore sequence mirrors the device's hardware-verified order and must
	// not be reordered. Each step is paced; the WiFi write is unprobed because a
	// static-IP change can relocate the device (unreachability there is relocation,
	// not a crash). The first step that drives the device into a reboot loop halts
	// the sequence with the breaking step recorded.
	steps := []gen1RestoreStep{
		// Device-level settings (name, timezone, mode — mode can restart the device
		// in place). Discoverable is applied only when the backup captured it.
		{name: "device-settings", probe: true, write: func() {
			restoreGen1DeviceSettings(ctx, dev, &settings, gen1ConfigHasKey(bkp.Config, "discoverable"), result)
		}},
		{name: componentWiFi, skip: opts.SkipNetwork, probe: false, write: func() {
			restoreGen1WiFi(ctx, dev, bkp, opts.NetworkOverride, result)
		}},
		{name: componentMQTT, probe: true, write: func() { restoreGen1MQTT(ctx, dev, &settings, result) }},
		{name: componentCloud, probe: true, write: func() { restoreGen1Cloud(ctx, dev, &settings, result) }},
		{name: "coiot", probe: true, write: func() { restoreGen1CoIoT(ctx, dev, &settings, result) }},
		{name: "sntp", probe: true, write: func() { restoreGen1SNTP(ctx, dev, &settings, result) }},
		{name: "auth", skip: opts.SkipAuth, probe: true, write: func() {
			restoreGen1Auth(ctx, dev, bkp, result)
		}},
		{name: "components", probe: true, write: func() { restoreGen1Components(ctx, dev, &settings, result) }},
		// Captured live light state (color temperature, brightness, color, white
		// channels) lives in /light, /color and /white, not /settings, so the
		// component config above does not carry it. Each apply is a no-op when its
		// state is absent from the backup.
		{name: "light-state", skip: opts.SkipState, probe: true, write: func() {
			applyGen1LightState(ctx, dev, bkp, result)
			applyGen1ColorState(ctx, dev, bkp, result)
			applyGen1WhiteState(ctx, dev, bkp, result)
		}},
		// Per-meter overpower limits, which restoreGen1Components does not cover (the
		// device-level max_power is restored with the device settings).
		{name: "meters", skip: opts.SkipMeters, probe: true, write: func() {
			restoreGen1Meters(ctx, dev, &settings, result)
			restoreGen1EMeters(ctx, dev, &settings, result)
		}},
	}
	for _, step := range steps {
		if step.skip {
			continue
		}
		if !runGen1RestoreStep(ctx, dev, pacer, opts, step, result) {
			return result, nil
		}
	}

	// Restore action URLs / webhooks (if not skipped). This is the final write and
	// carries no pacing — nothing follows it to be protected from a restart.
	if !opts.SkipWebhooks {
		restoreGen1Actions(ctx, dev, bkp, result)
	}

	return result, nil
}

// restoreGen1ClockDependent re-applies the clock-gated config — light component
// config, then captured light/color/white state — during the LAN second pass of a
// factory-AP restore, once the device has a clock. The light-state pass is skipped
// when the component write destabilizes the device (it returns without applying).
func restoreGen1ClockDependent(
	ctx context.Context,
	dev *gen1.Device,
	pacer gen1Pacer,
	opts *Gen1RestoreOptions,
	settings *gen1.Settings,
	bkp *Backup,
	result *RestoreResult,
) {
	if !runGen1Step(ctx, dev, pacer, opts, "light-config", result, func() {
		restoreGen1Components(ctx, dev, settings, result)
	}) {
		return
	}
	if opts.SkipState {
		return
	}
	runGen1Step(ctx, dev, pacer, opts, "light-state", result, func() {
		applyGen1LightState(ctx, dev, bkp, result)
		applyGen1ColorState(ctx, dev, bkp, result)
		applyGen1WhiteState(ctx, dev, bkp, result)
	})
}

// Component map keys under which captured live state is stored, separate from the
// /settings-derived config which carries only static fields (name, default_state).
const (
	lightStateKey = "light_state"
	colorStateKey = "color_state"
	whiteStateKey = "white_state"
)

// Gen1 device operating modes (settings.mode) for multi-mode RGBW devices.
const (
	gen1ModeColor = "color"
	gen1ModeWhite = "white"
)

// gen1LightState is a light's live, restorable state — the color temperature and
// brightness the Duo keeps in /light rather than /settings.
type gen1LightState struct {
	ID         int `json:"id"`
	Temp       int `json:"temp,omitempty"`
	Brightness int `json:"brightness"`
}

// captureGen1LiveState reads the device's live light/color/white state and stores
// each non-empty result under its Components key. The light state is captured for
// any device with lights (covers white-temp bulbs like the Duo); color and white
// state are gated on the device mode and best-effort, so a device that does not
// expose the matching endpoint contributes nothing.
func captureGen1LiveState(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, bkp *Backup) {
	put := func(key string, data json.RawMessage) {
		if data == nil {
			return
		}
		if bkp.Components == nil {
			bkp.Components = map[string]json.RawMessage{}
		}
		bkp.Components[key] = data
	}

	put(lightStateKey, captureGen1LightState(ctx, dev, len(settings.Lights)))
	if settings.Mode == gen1ModeColor {
		put(colorStateKey, captureGen1ColorState(ctx, dev))
	}
	if settings.Mode == gen1ModeWhite {
		put(whiteStateKey, captureGen1WhiteState(ctx, dev, len(settings.Lights)))
	}
}

// captureGen1LightState reads each light's live status so color temperature and
// brightness survive a backup. Returns nil when there are no lights or none could
// be read; per-light read failures are skipped rather than failing the backup.
func captureGen1LightState(ctx context.Context, dev *gen1.Device, numLights int) json.RawMessage {
	if numLights == 0 {
		return nil
	}
	states := make([]gen1LightState, 0, numLights)
	for i := range numLights {
		status, err := dev.Light(i).GetStatus(ctx)
		if err != nil {
			continue
		}
		states = append(states, gen1LightState{ID: i, Temp: status.Temp, Brightness: status.Brightness})
	}
	if len(states) == 0 {
		return nil
	}
	return mustMarshal(states)
}

// restoreGen1Meters re-applies per-meter overpower limits captured in
// settings.Meters. Only PowerLimit is writable on Gen1 — the API exposes no setter
// for the under/over limits — and a zero limit (unset) is left alone.
func restoreGen1Meters(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	for i := range settings.Meters {
		limit := settings.Meters[i].PowerLimit
		if limit <= 0 {
			continue
		}
		if err := dev.Meter(i).SetPowerLimit(ctx, limit); err != nil {
			addWarningf(result, "set meter %d power limit: %v", i, err)
		}
	}
}

// restoreGen1EMeters re-applies the writable energy-meter settings captured in
// settings.EMeters (Shelly EM / 3EM): the overpower threshold and the over/under
// power action URLs and their thresholds. SetEMeterConfig is a no-op when an
// entry carries none of those, so untouched meters are left alone.
func restoreGen1EMeters(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	for i := range settings.EMeters {
		if err := dev.SetEMeterConfig(ctx, i, settings.EMeters[i]); err != nil {
			addWarningf(result, "set emeter %d config: %v", i, err)
		}
	}
}

// applyGen1LightState re-applies captured color temperature and brightness to the
// device's lights. Absent state (older backups, non-light devices) is a no-op.
// At the clockless factory AP the temp command may be rejected; the LAN re-apply
// pass in RestoreToAP runs this again where the device has a clock.
func applyGen1LightState(ctx context.Context, dev *gen1.Device, bkp *Backup, result *RestoreResult) {
	raw, ok := bkp.Components[lightStateKey]
	if !ok {
		return
	}
	var states []gen1LightState
	if err := json.Unmarshal(raw, &states); err != nil {
		addWarningf(result, "parse light state: %v", err)
		return
	}
	for _, st := range states {
		light := dev.Light(st.ID)
		if st.Temp > 0 {
			if tErr := light.SetColorTemp(ctx, st.Temp); tErr != nil {
				addWarningf(result, "set light %d color temp: %v", st.ID, tErr)
			}
		}
		if st.Brightness > 0 {
			if bErr := light.SetBrightness(ctx, st.Brightness); bErr != nil {
				addWarningf(result, "set light %d brightness: %v", st.ID, bErr)
			}
		}
	}
}

// gen1ColorState is a color-mode light's live, restorable state — the RGB
// channels, white channel, gain (brightness) and effect the Bulb/RGBW2 keep in
// /color rather than /settings.
type gen1ColorState struct {
	Red    int `json:"red"`
	Green  int `json:"green"`
	Blue   int `json:"blue"`
	White  int `json:"white"`
	Gain   int `json:"gain"`
	Effect int `json:"effect"`
}

// gen1WhiteState is a white channel's live, restorable brightness, kept in
// /white on RGBW2 devices running in white mode.
type gen1WhiteState struct {
	ID         int `json:"id"`
	Brightness int `json:"brightness"`
}

// captureGen1ColorState reads the single color output's live status so RGB,
// white, gain and effect survive a backup. Returns nil when the device exposes no
// /color endpoint (non-color device) or the read fails, so callers can store it
// unconditionally without polluting white-only backups.
func captureGen1ColorState(ctx context.Context, dev *gen1.Device) json.RawMessage {
	status, err := dev.Color(0).GetStatus(ctx)
	if err != nil {
		return nil
	}
	return mustMarshal(gen1ColorState{
		Red:    status.Red,
		Green:  status.Green,
		Blue:   status.Blue,
		White:  status.White,
		Gain:   status.Gain,
		Effect: status.Effect,
	})
}

// captureGen1WhiteState reads each white channel's live brightness. Returns nil
// when there are no channels or none could be read (e.g. a single-output white-temp
// bulb, whose state the /light capture already covers); per-channel read failures
// are skipped rather than failing the backup.
func captureGen1WhiteState(ctx context.Context, dev *gen1.Device, numChannels int) json.RawMessage {
	if numChannels == 0 {
		return nil
	}
	states := make([]gen1WhiteState, 0, numChannels)
	for i := range numChannels {
		status, err := dev.White(i).GetStatus(ctx)
		if err != nil {
			continue
		}
		states = append(states, gen1WhiteState{ID: i, Brightness: status.Brightness})
	}
	if len(states) == 0 {
		return nil
	}
	return mustMarshal(states)
}

// applyGen1ColorState re-applies captured RGB, white, gain and effect to the
// device's color output. Absent state (white-only or older backups) is a no-op.
// Like the light state, the color command may be rejected at the clockless factory
// AP; the LAN re-apply pass in RestoreToAP runs this again where there is a clock.
func applyGen1ColorState(ctx context.Context, dev *gen1.Device, bkp *Backup, result *RestoreResult) {
	raw, ok := bkp.Components[colorStateKey]
	if !ok {
		return
	}
	var state gen1ColorState
	if err := json.Unmarshal(raw, &state); err != nil {
		addWarningf(result, "parse color state: %v", err)
		return
	}
	color := dev.Color(0)
	if err := color.SetRGBW(ctx, state.Red, state.Green, state.Blue, state.White); err != nil {
		addWarningf(result, "set color RGBW: %v", err)
	}
	if state.Gain > 0 {
		if err := color.SetGain(ctx, state.Gain); err != nil {
			addWarningf(result, "set color gain: %v", err)
		}
	}
	if state.Effect > 0 {
		if err := color.SetEffect(ctx, state.Effect); err != nil {
			addWarningf(result, "set color effect: %v", err)
		}
	}
}

// applyGen1WhiteState re-applies captured per-channel brightness to the device's
// white channels. Absent state (non-white-mode or older backups) is a no-op.
func applyGen1WhiteState(ctx context.Context, dev *gen1.Device, bkp *Backup, result *RestoreResult) {
	raw, ok := bkp.Components[whiteStateKey]
	if !ok {
		return
	}
	var states []gen1WhiteState
	if err := json.Unmarshal(raw, &states); err != nil {
		addWarningf(result, "parse white state: %v", err)
		return
	}
	for _, st := range states {
		if st.Brightness > 0 {
			if err := dev.White(st.ID).SetBrightness(ctx, st.Brightness); err != nil {
				addWarningf(result, "set white %d brightness: %v", st.ID, err)
			}
		}
	}
}

// marshalGen1WiFi extracts WiFi settings from Gen1 Settings into a JSON blob.
func marshalGen1WiFi(settings *gen1.Settings) json.RawMessage {
	wifi := map[string]any{}
	if settings.WiFiSta != nil {
		wifi["sta"] = settings.WiFiSta
	}
	if settings.WiFiSta1 != nil {
		wifi["sta1"] = settings.WiFiSta1
	}
	if settings.WiFiAp != nil {
		wifi["ap"] = settings.WiFiAp
	}
	if settings.ApRoaming != nil {
		wifi["ap_roaming"] = settings.ApRoaming
	}
	if len(wifi) == 0 {
		return nil
	}
	return mustMarshal(wifi)
}

// marshalGen1Components extracts component settings into the Components map.
func marshalGen1Components(settings *gen1.Settings) map[string]json.RawMessage {
	components := map[string]json.RawMessage{}
	if len(settings.Lights) > 0 {
		components["lights"] = mustMarshal(settings.Lights)
	}
	if len(settings.Relays) > 0 {
		components["relays"] = mustMarshal(settings.Relays)
	}
	if len(settings.Rollers) > 0 {
		components["rollers"] = mustMarshal(settings.Rollers)
	}
	if len(settings.Meters) > 0 {
		components["meters"] = mustMarshal(settings.Meters)
	}
	if len(settings.EMeters) > 0 {
		components["emeters"] = mustMarshal(settings.EMeters)
	}
	if len(components) == 0 {
		return nil
	}
	return components
}

// marshalGen1Schedules extracts schedule rules from relay and light settings.
func marshalGen1Schedules(settings *gen1.Settings) json.RawMessage {
	type scheduleEntry struct {
		Component string   `json:"component"`
		Rules     []string `json:"rules"`
		ID        int      `json:"id"`
		Enabled   bool     `json:"enabled"`
	}

	var entries []scheduleEntry
	for i := range settings.Relays {
		relay := &settings.Relays[i]
		if len(relay.ScheduleRules) > 0 {
			entries = append(entries, scheduleEntry{
				Component: "relay",
				ID:        i,
				Enabled:   relay.Schedule,
				Rules:     relay.ScheduleRules,
			})
		}
	}
	for i := range settings.Lights {
		light := &settings.Lights[i]
		if len(light.ScheduleRules) > 0 {
			entries = append(entries, scheduleEntry{
				Component: "light",
				ID:        i,
				Enabled:   light.Schedule,
				Rules:     light.ScheduleRules,
			})
		}
	}
	if len(entries) == 0 {
		return nil
	}
	return mustMarshal(entries)
}

// restoreGen1DeviceSettings restores device-level settings (name, timezone, etc.).
func restoreGen1DeviceSettings(
	ctx context.Context,
	dev *gen1.Device,
	settings *gen1.Settings,
	applyDiscoverable bool,
	result *RestoreResult,
) {
	if settings.Name != "" {
		if err := dev.SetName(ctx, settings.Name); err != nil {
			addWarningf(result, "set name: %v", err)
		}
	}
	if settings.Tz != "" {
		if err := dev.SetTimezone(ctx, settings.Tz); err != nil {
			addWarningf(result, "set timezone: %v", err)
		}
	}
	if settings.Lat != 0 || settings.Lng != 0 {
		if err := dev.SetLocation(ctx, settings.Lat, settings.Lng); err != nil {
			addWarningf(result, "set location: %v", err)
		}
	}
	if settings.Mode != "" {
		if err := dev.SetMode(ctx, settings.Mode); err != nil {
			addWarningf(result, "set mode: %v", err)
		}
	}
	// Only write discoverable when the backup actually captured it. A partial
	// backup (one taken while the device was unstable) omits the field, which
	// unmarshals to false; writing that would force-disable mDNS the source
	// never disabled.
	if applyDiscoverable {
		if err := dev.SetDiscoverable(ctx, settings.Discoverable); err != nil {
			addWarningf(result, "set discoverable: %v", err)
		}
	}
	if settings.MaxPower > 0 {
		if err := dev.SetMaxPower(ctx, settings.MaxPower); err != nil {
			addWarningf(result, "set max power: %v", err)
		}
	}
}

// gen1ConfigHasKey reports whether the backup's raw config JSON contains a
// top-level key. It distinguishes a value the source actually set from one
// merely defaulted by unmarshalling a partial backup, so restore does not write
// settings the backup never captured.
func gen1ConfigHasKey(raw json.RawMessage, key string) bool {
	if len(raw) == 0 {
		return false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}

// restoreGen1WiFi restores WiFi settings from the backup. When override is
// non-nil, its network fields replace the backup's station settings before they
// are applied (used to clone a device's config without copying its IP address).
func restoreGen1WiFi(
	ctx context.Context,
	dev *gen1.Device,
	bkp *Backup,
	override *Gen1NetworkOverride,
	result *RestoreResult,
) {
	if bkp.WiFi == nil && override == nil {
		return
	}
	var wifi struct {
		Sta       *gen1.WiFiStaSettings   `json:"sta,omitempty"`
		Sta1      *gen1.WiFiStaSettings   `json:"sta1,omitempty"`
		Ap        *gen1.WiFiApSettings    `json:"ap,omitempty"`
		ApRoaming *gen1.ApRoamingSettings `json:"ap_roaming,omitempty"`
	}
	if bkp.WiFi != nil {
		if err := json.Unmarshal(bkp.WiFi, &wifi); err != nil {
			addWarningf(result, "parse WiFi config: %v", err)
			return
		}
	}

	if override != nil {
		if wifi.Sta == nil {
			wifi.Sta = &gen1.WiFiStaSettings{Enabled: true}
		}
		applyGen1WiFiOverride(wifi.Sta, override)
	}

	if wifi.Sta != nil {
		restoreGen1WiFiStation(ctx, dev, wifi.Sta, result)
		result.RestartRequired = true
	}
	// Restore the secondary (backup) station so the device keeps its failover
	// network. Only meaningful when the source actually configured an SSID.
	if wifi.Sta1 != nil && wifi.Sta1.SSID != "" {
		restoreGen1WiFiStation1(ctx, dev, wifi.Sta1, result)
		result.RestartRequired = true
	}
	if wifi.Ap != nil {
		if err := dev.SetWiFiAP(ctx, wifi.Ap.Enabled, wifi.Ap.SSID, wifi.Ap.Key); err != nil {
			addWarningf(result, "set WiFi AP: %v", err)
		}
	}
	if wifi.ApRoaming != nil {
		if err := dev.SetApRoaming(ctx, wifi.ApRoaming.Enabled, wifi.ApRoaming.Threshold); err != nil {
			addWarningf(result, "set AP roaming: %v", err)
		}
	}
}

// applyGen1WiFiOverride overlays a Gen1NetworkOverride onto a Gen1 station config.
// SSID and Key are replaced only when explicitly provided; a static IP switches
// the station to static IPv4 addressing.
func applyGen1WiFiOverride(sta *gen1.WiFiStaSettings, ov *Gen1NetworkOverride) {
	sta.Enabled = true
	if ov.SSID != "" {
		sta.SSID = ov.SSID
	}
	if ov.Password != "" {
		sta.Key = ov.Password
	}
	if ov.IsStatic() {
		sta.Ipv4Method = gen1IPv4ModeStatic
		sta.IP = ov.StaticIP
		sta.Gw = ov.Gateway
		sta.Mask = ov.Netmask
		sta.DNS = ov.DNS
	}
}

// restoreGen1WiFiStation restores a WiFi station configuration.
func restoreGen1WiFiStation(ctx context.Context, dev *gen1.Device, sta *gen1.WiFiStaSettings, result *RestoreResult) {
	if sta.Ipv4Method == gen1IPv4ModeStatic {
		err := dev.SetWiFiStationStatic(ctx, sta.SSID, sta.Key, sta.IP, sta.Gw, sta.Mask, sta.DNS)
		if err != nil {
			addWarningf(result, "set WiFi station static: %v", err)
		}
		return
	}
	if err := dev.SetWiFiStation(ctx, sta.Enabled, sta.SSID, sta.Key); err != nil {
		addWarningf(result, "set WiFi station: %v", err)
	}
}

// restoreGen1WiFiStation1 restores the secondary (backup) WiFi station, mirroring
// restoreGen1WiFiStation but targeting the sta1 slot.
func restoreGen1WiFiStation1(ctx context.Context, dev *gen1.Device, sta *gen1.WiFiStaSettings, result *RestoreResult) {
	if sta.Ipv4Method == gen1IPv4ModeStatic {
		err := dev.SetWiFiStation1Static(ctx, sta.SSID, sta.Key, sta.IP, sta.Gw, sta.Mask, sta.DNS)
		if err != nil {
			addWarningf(result, "set WiFi station1 static: %v", err)
		}
		return
	}
	if err := dev.SetWiFiStation1(ctx, sta.Enabled, sta.SSID, sta.Key); err != nil {
		addWarningf(result, "set WiFi station1: %v", err)
	}
}

// restoreGen1MQTT restores MQTT settings.
func restoreGen1MQTT(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	if settings.MQTT == nil {
		return
	}
	cfg := &gen1.MQTTConfig{
		Enable:              settings.MQTT.Enable,
		Server:              settings.MQTT.Server,
		User:                settings.MQTT.User,
		Password:            settings.MQTT.Pass,
		ID:                  settings.MQTT.ID,
		KeepAlive:           settings.MQTT.KeepAlive,
		MaxQos:              settings.MQTT.MaxQos,
		CleanSession:        settings.MQTT.CleanSession,
		Retain:              settings.MQTT.Retain,
		UpdatePeriod:        settings.MQTT.UpdatePeriod,
		ReconnectTimeoutMax: settings.MQTT.ReconnectTimeoutMax,
		ReconnectTimeoutMin: settings.MQTT.ReconnectTimeoutMin,
	}
	if err := dev.SetMQTTConfig(ctx, cfg); err != nil {
		addWarningf(result, "set MQTT: %v", err)
	}
}

// restoreGen1Cloud restores Cloud settings.
func restoreGen1Cloud(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	if settings.Cloud == nil {
		return
	}
	if err := dev.SetCloud(ctx, settings.Cloud.Enabled); err != nil {
		addWarningf(result, "set cloud: %v", err)
	}
}

// restoreGen1CoIoT restores CoIoT protocol settings.
func restoreGen1CoIoT(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	if settings.CoIoT == nil {
		return
	}
	if err := dev.SetCoIoT(ctx, settings.CoIoT.Enabled, settings.CoIoT.UpdatePeriod, settings.CoIoT.Peer); err != nil {
		addWarningf(result, "set CoIoT: %v", err)
	}
}

// restoreGen1SNTP restores time server settings.
func restoreGen1SNTP(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	if settings.SNTP == nil {
		return
	}
	if settings.SNTP.Server != "" {
		if err := dev.SetTimeServer(ctx, settings.SNTP.Server); err != nil {
			addWarningf(result, "set time server: %v", err)
		}
	}
}

// restoreGen1Auth restores authentication settings.
func restoreGen1Auth(ctx context.Context, dev *gen1.Device, bkp *Backup, result *RestoreResult) {
	if bkp.Auth == nil {
		return
	}
	// Note: Gen1 auth restore enables/disables auth but cannot restore the password
	// (passwords are never exported). User must set password separately if needed.
	if err := dev.SetAuth(ctx, bkp.Auth.Enable, bkp.Auth.User, ""); err != nil {
		addWarningf(result, "set auth: %v", err)
	}
}

// restoreGen1Components restores component-specific configurations.
func restoreGen1Components(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	for i := range settings.Lights {
		light := &settings.Lights[i]
		cfg := gen1.LightConfig{
			Name:         light.Name,
			DefaultState: light.DefaultState,
			BtnType:      light.BtnType,
			AutoOn:       light.AutoOn,
			AutoOff:      light.AutoOff,
			BtnReverse:   light.BtnReverse != 0,
			Schedule:     light.Schedule,
			// Carry the rules in the same write as the schedule flag: a separate
			// follow-up schedule_rules call races the config write on the device
			// and is silently dropped (200 OK, rules not persisted).
			ScheduleRules: light.ScheduleRules,
		}
		if err := dev.SetLightConfig(ctx, i, &cfg); err != nil {
			addWarningf(result, "set light %d config: %v", i, err)
		}
	}

	for i := range settings.Relays {
		relay := &settings.Relays[i]
		cfg := &gen1.RelayConfig{
			Name:         relay.Name,
			DefaultState: relay.DefaultState,
			BtnType:      relay.BtnType,
			AutoOn:       relay.AutoOn,
			AutoOff:      relay.AutoOff,
			MaxPower:     relay.MaxPower,
			BtnReverse:   relay.IsBtnReverse(),
			Schedule:     relay.Schedule,
			// As with lights, carry the rules atomically with the config write so
			// the device does not silently drop a separate follow-up call.
			ScheduleRules: relay.ScheduleRules,
		}
		if err := dev.SetRelayConfig(ctx, i, cfg); err != nil {
			addWarningf(result, "set relay %d config: %v", i, err)
		}
	}

	for i := range settings.Rollers {
		roller := &settings.Rollers[i]
		cfg := &gen1.RollerConfig{
			MaxTimeOpen:    roller.MaxTimeOpen,
			MaxTimeClose:   roller.MaxTimeClose,
			DefaultState:   roller.DefaultState,
			Swap:           roller.Swap,
			SwapInputs:     roller.SwapInputs,
			InputMode:      roller.InputMode,
			BtnType:        roller.BtnType,
			BtnReverse:     roller.BtnReverse != 0,
			SafetyMode:     roller.SafetyMode,
			SafetyAction:   roller.SafetyAction,
			ObstacleMode:   roller.ObstacleMode,
			ObstacleAction: roller.ObstacleAction,
			ObstaclePower:  roller.ObstaclePower,
			ObstacleDelay:  roller.ObstacleDelay,
			Positioning:    roller.Positioning,
		}
		if err := dev.SetRollerConfig(ctx, i, cfg); err != nil {
			addWarningf(result, "set roller %d config: %v", i, err)
		}
	}
}

// restoreGen1Actions restores action URLs (Gen1's equivalent of webhooks).
func restoreGen1Actions(ctx context.Context, dev *gen1.Device, bkp *Backup, result *RestoreResult) {
	if bkp.Webhooks == nil {
		return
	}
	var actions gen1.ActionSettings
	if err := json.Unmarshal(bkp.Webhooks, &actions); err != nil {
		addWarningf(result, "parse actions: %v", err)
		return
	}

	for _, action := range actions.Actions {
		if len(action.URLs) > 0 {
			if err := dev.SetAction(ctx, action.Index, action.Event, action.URLs, action.Enabled); err != nil {
				addWarningf(result, "set action %s: %v", action.Event, err)
			}
		}
	}
}

// addWarningf appends a formatted warning message to the restore result.
func addWarningf(result *RestoreResult, format string, args ...any) {
	result.Warnings = append(result.Warnings, fmt.Sprintf(format, args...))
}

// mustMarshal marshals a value to JSON, panicking on error.
// Only used for values that are known to be valid JSON-serializable types.
func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("backup: failed to marshal %T: %v", v, err))
	}
	return data
}
