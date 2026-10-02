package reprovision

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tj-smith47/shelly-go/backup"
	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/gen2/components"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// lanSettleDelay gives a freshly-joined device a moment to obtain NTP time before
// the full restore on the LAN, so time-dependent settings do not hit the clock
// error they would at the clockless access point.
const lanSettleDelay = 8 * time.Second

// lanFullRestoreBudget caps the LAN pass that applies the full configuration. It
// writes every setting group and the device can restart mid-pass, so it is
// generous; any firmware update already happened at the access point, so this
// pass is config-only.
const lanFullRestoreBudget = 4 * time.Minute

// ipv4ModeStatic is the Gen2+ station addressing mode for a fixed address.
const ipv4ModeStatic = "static"

// Restore restores a backup onto a device sitting at its factory access point.
//
// It hops the host onto the device's open access point, checks the device's MAC
// against the access point name, brings a Gen1 device up to the backup's firmware
// and confirms it is stable, writes only the WiFi station settings, reboots the
// device, and returns the host to its home network. Once the device is seen on
// the LAN it applies the full configuration there, where the device has a clock
// and is stable.
//
// The returned result is non-nil whenever the device was reached on the LAN, and
// its Address is set even when the full restore there failed, so a caller can
// record where the device now lives.
func Restore(ctx context.Context, opts *RestoreOptions) (*RestoreResult, error) {
	if opts == nil {
		return nil, fmt.Errorf("%w: options are required", types.ErrInvalidParam)
	}
	if opts.APSSID == "" {
		return nil, fmt.Errorf("%w: APSSID is required", types.ErrInvalidParam)
	}
	if opts.Backup == nil {
		return nil, fmt.Errorf("%w: Backup is required", types.ErrInvalidParam)
	}
	r := newRunner(opts.Scanner, opts.Logger, opts.OnStep, opts.APHostIP)
	return r.restore(ctx, opts)
}

// passOptions selects what one restore pass writes.
type passOptions struct {
	override       *Network
	networkOnly    bool
	skipNetwork    bool
	allowDowngrade bool
	skipClockWait  bool
}

// restore runs a Restore with the runner's dependencies.
func (r *runner) restore(ctx context.Context, opts *RestoreOptions) (*RestoreResult, error) {
	bkp := opts.Backup
	fromBackup := NetworkFromBackup(bkp)
	join, err := r.resolveJoinNetwork(ctx, &fromBackup, &opts.Network)
	if err != nil {
		return nil, err
	}

	// At the access point the device carries no generation hint and a Gen1 device
	// does not answer the Gen2 RPC probe, so the backup's generation routes the
	// writes; the target is the same model as the source.
	generation, model, backupFW, backupMAC := backupDevice(bkp)

	fwPath := r.prefetchAPFirmware(ctx, generation, model, opts.FirmwareURL, opts.AllowFirmwareDowngrade)
	defer r.removeFirmwareTemp(fwPath)

	res := &RestoreResult{MAC: normalizeMAC(backupMAC)}
	var restoreErr error
	hopErr := r.withAPHop(ctx, opts.APSSID, &join, func(ctx context.Context) error {
		mac, idErr := r.confirmAPDeviceIdentity(ctx, generation, opts.APSSID)
		if idErr != nil {
			restoreErr = idErr
			return idErr
		}
		if mac != "" {
			fErr := checkForeignBackup(opts.APSSID, normalizeMAC(backupMAC), mac, opts.AllowForeignBackup)
			if fErr != nil {
				restoreErr = fErr
				return fErr
			}
			res.MAC = mac
		}
		if generation == 1 {
			if fwErr := r.ensureGen1FirmwareAtAP(ctx, apFirmwareBindIP(r.apHostIP), fwPath, backupFW,
				opts.AllowFirmwareDowngrade); fwErr != nil {
				return fwErr
			}
			if stErr := r.confirmGen1StableAtAP(ctx); stErr != nil {
				return stErr
			}
		}
		// The access point pass writes only the station settings: the access
		// point has no clock, and the instant the station settings are written
		// the device starts leaving it, so any further write there races a device
		// that is dropping the connection. The full restore follows on the LAN.
		r.step("writing WiFi settings")
		res.Restore, restoreErr = r.restoreDevice(ctx, r.apAddr, "", generation, opts, passOptions{
			override:      &join,
			networkOnly:   true,
			skipClockWait: true,
		})
		if restoreErr != nil {
			return restoreErr
		}
		// A device keeps serving its access point until it reboots, so it is
		// rebooted while the host is still there.
		r.rebootAtAP(ctx, generation)
		return nil
	})
	if restoreErr != nil {
		return res, fmt.Errorf("restore at AP %q failed: %w", opts.APSSID, restoreErr)
	}
	if hopErr != nil {
		return res, fmt.Errorf("AP hop for %q failed: %w", opts.APSSID, hopErr)
	}

	conf, confErr := r.confirmRejoin(ctx, generation, join.StaticIP, res.MAC)
	if confErr != nil {
		return res, fmt.Errorf(
			"restore applied at AP %q but the device was not seen back on the LAN (%w); the device may "+
				"still be on its factory AP; if this host is not on the device's subnet, restore from one that is",
			opts.APSSID, confErr)
	}
	res.Address = conf.addr
	res.SeenVia = conf.via
	if !conf.writeable {
		return res, fmt.Errorf(
			"%w: restore applied at AP %q and the device rejoined the LAN at %s (seen via %s), but this host "+
				"cannot reach it to write the full configuration; run the restore from a host on the device's subnet",
			ErrNoRoute, opts.APSSID, conf.addr, conf.via)
	}

	res.Reachable = true
	lanResult, lanErr := r.fullRestoreOnLAN(ctx, conf.addr, conf.bindIface, generation, opts)
	if lanResult != nil {
		res.Restore = lanResult
	}
	if lanErr != nil {
		return res, fmt.Errorf("the device joined the LAN at %s but the full configuration restore failed: %w",
			conf.addr, lanErr)
	}
	return res, nil
}

// fullRestoreOnLAN applies the complete configuration at the device's LAN address
// after the access point pass wrote only the station settings. The network is
// skipped because it already took, and the firmware gate is bypassed because any
// update already happened at the access point; an update on the LAN would
// reboot-loop a device that cannot hold station mode.
func (r *runner) fullRestoreOnLAN(
	ctx context.Context,
	addr, iface string,
	generation int,
	opts *RestoreOptions,
) (*backup.RestoreResult, error) {
	r.step("restoring the full configuration")
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(r.lanSettleDelay):
	}
	lanCtx, cancel := context.WithTimeout(ctx, lanFullRestoreBudget)
	defer cancel()
	return r.restoreDevice(lanCtx, addr, iface, generation, opts, passOptions{
		skipNetwork:    true,
		allowDowngrade: true,
	})
}

// restoreDevice runs one restore pass against the device at addr, bound to iface
// when one is named.
func (r *runner) restoreDevice(
	ctx context.Context,
	addr, iface string,
	generation int,
	opts *RestoreOptions,
	pass passOptions,
) (*backup.RestoreResult, error) {
	if generation == 1 {
		return r.restoreGen1(ctx, addr, iface, opts, pass)
	}
	return r.restoreGen2(ctx, addr, iface, opts, pass)
}

// restoreGen1 runs one restore pass against a Gen1 device. A step that drives the
// device into a reboot loop halts the restore and is returned as an error.
func (r *runner) restoreGen1(
	ctx context.Context,
	addr, iface string,
	opts *RestoreOptions,
	pass passOptions,
) (*backup.RestoreResult, error) {
	gopts := &backup.Gen1RestoreOptions{
		StepTrace:              opts.StepTrace,
		Name:                   opts.Name,
		FirmwareURL:            opts.FirmwareURL,
		SkipNetwork:            pass.skipNetwork,
		SkipAuth:               opts.SkipAuth,
		SkipState:              opts.SkipState,
		SkipMeters:             opts.SkipMeters,
		SkipWebhooks:           opts.SkipWebhooks,
		AllowFirmwareDowngrade: opts.AllowFirmwareDowngrade || pass.allowDowngrade,
		NetworkOnly:            pass.networkOnly,
		SkipClockWait:          pass.skipClockWait,
	}
	if ov := pass.override; ov != nil {
		gopts.NetworkOverride = &backup.Gen1NetworkOverride{
			SSID:     ov.SSID,
			Password: ov.Password,
			StaticIP: ov.StaticIP,
			Gateway:  ov.Gateway,
			Netmask:  ov.Netmask,
			DNS:      ov.DNS,
			Open:     ov.Open,
		}
	}
	var result *backup.RestoreResult
	err := r.withGen1(ctx, addr, iface, func(dev *gen1.Device, _ string) error {
		res, err := backup.RestoreGen1(ctx, dev, opts.Backup, gopts)
		if err != nil {
			return err
		}
		result = res
		if res.DestabilizedStep != "" {
			return fmt.Errorf(
				"%w: restore halted: the device became unstable after the %q step, a write drove it "+
					"into a reboot loop; set StepTrace to capture the per-step trace",
				ErrUnstable, res.DestabilizedStep)
		}
		return nil
	})
	return result, err
}

// restoreGen2 runs one restore pass against a Gen2+ device. A network-only pass
// restores just the WiFi settings; a name override is set through Sys.SetConfig,
// which the backup restore does not apply, and its failure is a warning.
func (r *runner) restoreGen2(
	ctx context.Context,
	addr, iface string,
	opts *RestoreOptions,
	pass passOptions,
) (*backup.RestoreResult, error) {
	toRestore := opts.Backup
	if pass.override != nil && !pass.skipNetwork {
		wifi, err := applyGen2WiFiOverride(toRestore.WiFi, pass.override)
		if err != nil {
			return nil, err
		}
		clone := *toRestore
		clone.WiFi = wifi
		toRestore = &clone
	}
	ropts := backup.DefaultRestoreOptions()
	ropts.RestoreWiFi = !pass.skipNetwork
	ropts.RestoreAuth = !opts.SkipAuth
	ropts.RestoreWebhooks = !opts.SkipWebhooks
	ropts.RestoreSchedules = !opts.SkipSchedules
	ropts.RestoreScripts = !opts.SkipScripts
	ropts.RestoreKVS = !opts.SkipKVS
	if pass.networkOnly {
		toRestore = &backup.Backup{
			Version:    toRestore.Version,
			DeviceInfo: toRestore.DeviceInfo,
			WiFi:       toRestore.WiFi,
		}
		ropts = &backup.RestoreOptions{RestoreWiFi: true}
	}
	data, err := json.Marshal(toRestore)
	if err != nil {
		return nil, fmt.Errorf("serialize backup: %w", err)
	}

	var result *backup.RestoreResult
	err = r.withGen2(ctx, addr, iface, func(client *rpc.Client, _ string) error {
		res, rerr := backup.New(client).Restore(ctx, data, ropts)
		if rerr != nil {
			return fmt.Errorf("restore backup: %w", rerr)
		}
		result = res
		if opts.Name != "" && !pass.networkOnly {
			name := opts.Name
			sys := components.NewSys(client)
			if nameErr := sys.SetConfig(ctx, &components.SysConfig{
				Device: &components.SysDeviceConfig{Name: &name},
			}); nameErr != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("set device name: %v", nameErr))
			}
		}
		return nil
	})
	return result, err
}

// applyGen2WiFiOverride overlays a network onto a Gen2+ WiFi config blob (the raw
// WiFi.GetConfig result) and returns the rewritten blob. SSID and password are
// replaced only when set; an open network is written as an empty passphrase; a
// static IP switches the station to static addressing.
// The {ap, sta, sta1} shape is kept so it round-trips through WiFi.SetConfig.
func applyGen2WiFiOverride(wifiBlob json.RawMessage, ov *Network) (json.RawMessage, error) {
	cfg := map[string]any{}
	if len(wifiBlob) > 0 {
		if err := json.Unmarshal(wifiBlob, &cfg); err != nil {
			return nil, fmt.Errorf("parse WiFi config for override: %w", err)
		}
	}
	sta, ok := cfg["sta"].(map[string]any)
	if !ok || sta == nil {
		sta = map[string]any{}
	}
	sta["enable"] = true
	if ov.SSID != "" {
		sta["ssid"] = ov.SSID
	}
	if ov.Password != "" {
		sta["pass"] = ov.Password
	}
	if ov.Open {
		sta["pass"] = ""
	}
	// The device derives is_open from the passphrase and ignores a write to it,
	// so the backup's read-back value is dropped rather than sent.
	delete(sta, "is_open")
	if ov.StaticIP != "" {
		sta["ipv4mode"] = ipv4ModeStatic
		sta["ip"] = ov.StaticIP
		sta["netmask"] = ov.Netmask
		sta["gw"] = ov.Gateway
		if ov.DNS != "" {
			sta["nameserver"] = ov.DNS
		}
	}
	cfg["sta"] = sta
	return json.Marshal(cfg)
}

// rebootAtAP reboots the device at its access point address so it applies the
// new station settings and joins the LAN. The call usually fails as the device
// drops the connection while rebooting, so the error is logged.
func (r *runner) rebootAtAP(ctx context.Context, generation int) {
	var err error
	if generation == 1 {
		err = r.withGen1(ctx, r.apAddr, "", func(dev *gen1.Device, _ string) error {
			return dev.Reboot(ctx)
		})
	} else {
		err = r.withGen2(ctx, r.apAddr, "", func(client *rpc.Client, _ string) error {
			_, callErr := client.Call(ctx, "Shelly.Reboot", nil)
			return callErr
		})
	}
	if err != nil {
		r.log.Debug("reboot at AP (expected as the device drops the connection)", "error", err)
	}
}

// confirmAPDeviceIdentity verifies the device answering at the access point is
// the one whose access point the host joined, before any write, and returns its
// MAC. A factory access point name ends with the device's MAC suffix (e.g.
// "ShellyBulbDuo-6645B6"); a device whose MAC does not end with it is a different
// device, and writing to it would strand a bystander. When the name carries no
// MAC suffix (a renamed access point) the check cannot be made and is skipped.
func (r *runner) confirmAPDeviceIdentity(ctx context.Context, generation int, apSSID string) (string, error) {
	r.step("confirming the device identity")
	actual, err := r.readDeviceMACAtAP(ctx, generation)
	return r.checkAPIdentity(apSSID, actual, err)
}

// checkForeignBackup refuses a backup recorded from one device when the device
// at the access point reports another MAC, unless the caller allowed it. A
// backup with no MAC cannot be checked and passes.
func checkForeignBackup(apSSID, backupMAC, deviceMAC string, allow bool) error {
	if allow || backupMAC == "" || backupMAC == deviceMAC {
		return nil
	}
	return fmt.Errorf("%w: the backup was taken from device %s but AP %q is serving device %s; "+
		"set AllowForeignBackup to write one device's backup onto another; nothing was written",
		ErrIdentityMismatch, backupMAC, apSSID, deviceMAC)
}

// checkAPIdentity applies the identity rule to a MAC read at the access point
// (or the error reading it) and returns the device's normalized MAC.
func (r *runner) checkAPIdentity(apSSID, actual string, readErr error) (string, error) {
	if readErr != nil {
		if macSuffixFromAPSSID(apSSID) == "" {
			r.log.Debug("AP carries no MAC suffix and the device MAC was not read", "ssid", apSSID, "error", readErr)
			return "", nil
		}
		return "", fmt.Errorf("could not read the device identity at AP %q to confirm the target before writing: %w",
			apSSID, readErr)
	}
	matched, skip, want, got := evaluateAPIdentity(actual, apSSID)
	if skip {
		r.log.Debug("identity check skipped", "ssid", apSSID, "mac", actual)
		return normalizeMAC(actual), nil
	}
	if !matched {
		return "", fmt.Errorf(
			"%w: AP %q is serving device %s, whose MAC does not match the AP name's suffix %q; nothing was "+
				"written; confirm the AP name and that the intended device is the one in AP mode, then retry",
			ErrIdentityMismatch, apSSID, got, want)
	}
	return got, nil
}

// evaluateAPIdentity decides whether a device reporting actualMAC is the one the
// access point apSSID belongs to. skip is true when identity cannot be derived
// (no MAC suffix in the name, or an unparseable MAC); want and got are the
// normalized suffix and MAC.
func evaluateAPIdentity(actualMAC, apSSID string) (matched, skip bool, want, got string) {
	want = macSuffixFromAPSSID(apSSID)
	if want == "" {
		return false, true, "", ""
	}
	got = normalizeMAC(actualMAC)
	if got == "" {
		return false, true, want, ""
	}
	return strings.HasSuffix(got, want), false, want, got
}

// macSuffixFromAPSSID extracts the trailing MAC hex from a factory access point
// name (the token after the final '-'), upper-cased, or "" when the token is
// absent or not pure hex.
func macSuffixFromAPSSID(ssid string) string {
	idx := strings.LastIndex(ssid, "-")
	if idx < 0 || idx == len(ssid)-1 {
		return ""
	}
	suffix := strings.ToUpper(ssid[idx+1:])
	for _, c := range suffix {
		if (c < '0' || c > '9') && (c < 'A' || c > 'F') {
			return ""
		}
	}
	return suffix
}

// readDeviceMACAtAP reads the MAC of the device answering at the access point.
func (r *runner) readDeviceMACAtAP(ctx context.Context, generation int) (string, error) {
	var mac string
	if generation == 1 {
		err := r.withGen1(ctx, r.apAddr, "", func(_ *gen1.Device, m string) error {
			mac = m
			return nil
		})
		return mac, err
	}
	err := r.withGen2(ctx, r.apAddr, "", func(_ *rpc.Client, m string) error {
		mac = m
		return nil
	})
	return mac, err
}

// backupDevice returns the generation, model, firmware and MAC recorded in a
// backup. A backup without device info is treated as Gen2+.
func backupDevice(bkp *backup.Backup) (generation int, model, firmware, mac string) {
	if bkp.DeviceInfo == nil {
		return 2, "", "", ""
	}
	info := bkp.DeviceInfo
	return info.Generation, info.Model, info.Version, info.MAC
}

// backupStation is the station block of a backup's WiFi settings. Gen1 and Gen2+
// name the addressing fields differently, so both spellings are read.
type backupStation struct {
	SSID       string `json:"ssid"`
	Key        string `json:"key"`
	Ipv4Method string `json:"ipv4_method"`
	IPv4Mode   string `json:"ipv4mode"`
	IP         string `json:"ip"`
	Gw         string `json:"gw"`
	Mask       string `json:"mask"`
	Netmask    string `json:"netmask"`
	DNS        string `json:"dns"`
	Nameserver string `json:"nameserver"`
	IsOpen     bool   `json:"is_open"`
}

// NetworkFromBackup reads the station network recorded in a backup: SSID,
// passphrase (unmasked Gen1 only), whether it is open (Gen2+ only; a Gen1
// backup does not record it) and static addressing. The station is the one
// backup.Backup.Station picks. StaticIP is empty when the backup's station uses
// DHCP. This is the network Restore joins the device to when
// RestoreOptions.Network overrides nothing.
func NetworkFromBackup(bkp *backup.Backup) Network {
	raw := bkp.Station()
	var sta backupStation
	if raw == nil || json.Unmarshal(raw, &sta) != nil {
		return Network{}
	}
	n := Network{SSID: sta.SSID, Password: sta.Key, Open: sta.IsOpen}
	if sta.Ipv4Method == ipv4ModeStatic || sta.IPv4Mode == ipv4ModeStatic {
		n.StaticIP = sta.IP
		n.Gateway = sta.Gw
		n.Netmask = cmp.Or(sta.Mask, sta.Netmask)
		n.DNS = cmp.Or(sta.DNS, sta.Nameserver)
	}
	return n
}
