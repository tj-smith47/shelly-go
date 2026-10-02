package reprovision

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/tj-smith47/shelly-go/backup"
	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

// Errors a caller can test for with errors.Is.
var (
	// ErrNoPassphrase means no WiFi passphrase was given and none could be found
	// in the backup or in the host's stored credentials. The error carrying it
	// is a *NoPassphraseError, which names the network.
	ErrNoPassphrase = errors.New("no WiFi passphrase")

	// ErrIdentityMismatch means the device answering at the access point is not
	// the one the access point name identifies. Nothing was written to it.
	ErrIdentityMismatch = errors.New("device identity mismatch")

	// ErrAPUnreachable means the host could not join the device's access point,
	// usually because it is out of range or not in AP mode. Nothing was written
	// to the device; retrying later may succeed.
	ErrAPUnreachable = errors.New("access point unreachable")

	// ErrFirmwareUnavailable means a Gen1 device runs older firmware than the
	// backup and needs an update at the access point, but no firmware image
	// could be downloaded before the hop. Nothing was written to the device.
	// The error carrying it is a *FirmwareUnavailableError, which names both
	// firmware versions.
	ErrFirmwareUnavailable = errors.New("firmware image unavailable")

	// ErrFirmwareUpdate means the firmware check or update at the access point
	// failed: the device's firmware could not be read, the image could not be
	// served from this host, or the flash did not take. The device's station
	// settings were not written.
	ErrFirmwareUpdate = errors.New("firmware update at the access point failed")

	// ErrUnstable means a Gen1 device could not hold a stable uptime, the
	// signature of a reboot loop.
	ErrUnstable = errors.New("device is not stable")

	// ErrNotRejoined means the device was not seen on the LAN after it left its
	// access point. It may still be on the access point.
	ErrNotRejoined = errors.New("device not seen on the network")

	// ErrNoRoute means the device announced itself on the LAN but no unicast
	// route from this host reaches it.
	ErrNoRoute = errors.New("no route from this host to the device")

	// ErrIncompleteStaticNetwork means the network to join has a static address
	// with no gateway or no netmask, from neither the options nor the backup.
	// Errors carrying it also match types.ErrInvalidParam. It is the same value
	// as backup.ErrIncompleteStaticNetwork, so either name matches.
	ErrIncompleteStaticNetwork = backup.ErrIncompleteStaticNetwork
)

// FirmwareUnavailableError reports a Gen1 device whose firmware is older than
// the backup's when no firmware image was available to update it. It matches
// ErrFirmwareUnavailable with errors.Is; read it with errors.As:
//
//	var fwErr *reprovision.FirmwareUnavailableError
//	if errors.As(err, &fwErr) {
//		fmt.Printf("device runs %s, backup needs %s\n", fwErr.Current, fwErr.Required)
//	}
type FirmwareUnavailableError struct {
	// Current is the firmware the device runs.
	Current string
	// Required is the firmware the backup was taken on.
	Required string
}

// Error describes the missing update and how to get past it.
func (e *FirmwareUnavailableError) Error() string {
	return fmt.Sprintf(
		"%v: device on firmware %q needs an update to the backup's %q before restore, but no "+
			"firmware image is available (the factory AP has no internet, so the image is "+
			"prefetched before the hop; its URL was underivable or the download failed); "+
			"retry with connectivity, set FirmwareURL, or set AllowFirmwareDowngrade to "+
			"force the downgrade and accept the reboot-loop risk",
		ErrFirmwareUnavailable, e.Current, e.Required)
}

// Unwrap returns ErrFirmwareUnavailable, so errors.Is matches it.
func (e *FirmwareUnavailableError) Unwrap() error { return ErrFirmwareUnavailable }

// NoPassphraseError reports a network to join for which no passphrase was
// given and none was found in the host's stored credentials. It matches
// ErrNoPassphrase with errors.Is; read it with errors.As to learn which network
// was resolved:
//
//	var pwErr *reprovision.NoPassphraseError
//	if errors.As(err, &pwErr) {
//		fmt.Printf("no passphrase for %s\n", pwErr.SSID)
//	}
type NoPassphraseError struct {
	// SSID is the network the device was to join. It is empty when no network
	// was named and the host is not on one.
	SSID string
}

// Error names the network and the two ways to get past the refusal.
func (e *NoPassphraseError) Error() string {
	return fmt.Sprintf(
		"%v for %q: Shelly devices return no station key and none was found in this "+
			"host's stored credentials; set Network.Password, or Network.Open for a "+
			"network with no passphrase",
		ErrNoPassphrase, e.SSID)
}

// Unwrap returns ErrNoPassphrase, so errors.Is matches it.
func (e *NoPassphraseError) Unwrap() error { return ErrNoPassphrase }

// Network is the WiFi network a device joins when it leaves its access point.
// Every field is optional. SSID and Password default to the backup's, then to
// the host's current network and stored passphrase. An empty StaticIP keeps the
// backup's addressing (static or DHCP). With a StaticIP, an empty Gateway,
// Netmask or DNS is taken from the backup's static settings (the rule is
// backup.ResolveStaticNetwork); a static address that ends up with no gateway
// or netmask is refused with ErrIncompleteStaticNetwork.
//
// Open joins a network that takes no passphrase: none is required and none is
// looked up on the host, and the SSID is resolved as above. Open together with
// a Password is refused with types.ErrInvalidParam. An open network is written
// as an empty passphrase; Gen2+ devices derive "is_open" from it. A backup
// whose station was open (Gen2+ "is_open") yields Open from NetworkFromBackup,
// and a restore with no SSID or Password override rejoins it as open. Gen1
// backups do not record whether the station was open, so a Gen1 device joins an
// open network only when Open is set.
type Network struct {
	SSID     string
	Password string
	StaticIP string
	Gateway  string
	Netmask  string
	DNS      string
	Open     bool
}

// RestoreOptions configures Restore. APSSID and Backup are required.
type RestoreOptions struct {
	// Scanner joins and leaves WiFi networks. Nil uses the platform scanner from
	// discovery.NewWiFiDiscoverer.
	Scanner discovery.WiFiScanner
	// StepTrace, when set, receives one line per Gen1 restore step with the
	// device's uptime after it, to find the setting that destabilizes a device.
	StepTrace io.Writer
	// Backup is the configuration to restore.
	Backup *backup.Backup
	// Logger receives debug traces. Nil discards them.
	Logger *slog.Logger
	// OnStep is called with a short description as each stage starts.
	OnStep func(step string)
	// APSSID is the device's factory access point, e.g. "ShellyBulbDuo-D12965".
	APSSID string
	// Name overrides the device name stored in the backup.
	Name string
	// APHostIP is the host's address on the access point subnet. Empty uses
	// discovery.DefaultAPHostIP.
	APHostIP string
	// FirmwareURL overrides the Gen1 firmware image flashed when the device runs
	// older firmware than the backup. Empty derives the official image URL from
	// the backup's model.
	FirmwareURL string
	// Network overrides the WiFi network the device joins. Each empty field is
	// derived: SSID and Password from the backup's station settings, then from
	// the host's current network and stored passphrase; the static address
	// settings from the backup's station settings.
	Network Network
	// AllowFirmwareDowngrade skips the firmware update and writes the backup
	// onto older firmware, accepting the reboot-loop risk.
	AllowFirmwareDowngrade bool
	// SkipAuth leaves the device's authentication settings alone.
	SkipAuth bool
	// SkipScripts leaves the device's scripts alone (Gen2+).
	SkipScripts bool
	// SkipSchedules leaves the device's schedules alone (Gen2+).
	SkipSchedules bool
	// SkipKVS leaves the device's key-value store alone (Gen2+).
	SkipKVS bool
	// SkipWebhooks leaves the device's webhooks or Gen1 action URLs alone.
	SkipWebhooks bool
	// SkipState leaves the device's light state (brightness, color) alone.
	SkipState bool
	// SkipMeters leaves the device's meter settings alone (Gen1).
	SkipMeters bool
}

// RestoreResult reports a Restore.
type RestoreResult struct {
	// Restore is the outcome of the full restore on the LAN, or of the network
	// restore at the access point when the device did not get that far. It can
	// be nil when an error stopped the restore before it ran.
	//
	// A nil error does not mean every setting was accepted: the device can
	// reject individual sections while the restore as a whole completes. Check
	// Restore.Success, Restore.Errors and Restore.Warnings.
	Restore *backup.RestoreResult
	// Address is the device's LAN address once it was seen there, empty when
	// it was not seen. Check Reachable before writing to it.
	Address string
	// MAC is the device's MAC address, upper-case without separators.
	MAC string
	// SeenVia says how the device was seen back on the LAN: "probe" (it
	// answered a unicast HTTP request), "mdns" or "coiot" (it announced itself
	// but this host has no route to it). Empty when it was not seen.
	SeenVia string
	// Reachable reports that this host reached the device at Address over
	// unicast. False with a non-empty Address means the device announced itself
	// on the LAN (mDNS or CoIoT) but this host has no route to it, so the full
	// restore did not run; Restore then returns an error wrapping ErrNoRoute.
	Reachable bool
}

// OnboardOptions configures Onboard. APSSID is required.
type OnboardOptions struct {
	// Scanner joins and leaves WiFi networks. Nil uses the platform scanner.
	Scanner discovery.WiFiScanner
	// Logger receives debug traces. Nil discards them.
	Logger *slog.Logger
	// OnStep is called with a short description as each stage starts.
	OnStep func(step string)
	// APSSID is the device's factory access point.
	APSSID string
	// APHostIP is the host's address on the access point subnet. Empty uses
	// discovery.DefaultAPHostIP.
	APHostIP string
	// Network is the WiFi network the device joins. An empty SSID or Password
	// is taken from the host's current network and stored passphrase; with
	// Network.Open no passphrase is used.
	Network Network
	// Generation is the device generation, used only when the device's
	// identity cannot be read at the access point. Zero reads it from the
	// device, and the read must then succeed. When both are known and differ,
	// the device's answer wins. An access point whose name carries a MAC
	// suffix always needs the identity read, to confirm the target before
	// writing.
	Generation int
}

// OnboardResult reports an Onboard.
type OnboardResult struct {
	// Address is the device's LAN address, empty when it was not found there.
	// Check Reachable before writing to it.
	Address string
	// MAC is the device's MAC address, upper-case without separators.
	MAC string
	// Note explains a partial success: the WiFi settings were written but the
	// device was not found on the LAN afterwards.
	Note string
	// SeenVia says how the device was seen on the LAN: "probe" (it answered a
	// unicast HTTP request), "mdns" or "coiot" (it announced itself but this
	// host has no route to it). Empty when it was not seen.
	SeenVia string
	// Generation is the device generation (1 for Gen1, 2 and up for RPC devices).
	Generation int
	// Reachable reports that this host reached the device at Address over
	// unicast. False with a non-empty Address means the device announced itself
	// on the LAN but this host has no route to it.
	Reachable bool
}

// InspectOptions configures Inspect. APSSID is required.
type InspectOptions struct {
	// Scanner joins and leaves WiFi networks. Nil uses the platform scanner.
	Scanner discovery.WiFiScanner
	// Logger receives debug traces. Nil discards them.
	Logger *slog.Logger
	// OnStep is called with a short description as each stage starts.
	OnStep func(step string)
	// APSSID is the device's factory access point.
	APSSID string
	// APHostIP is the host's address on the access point subnet. Empty uses
	// discovery.DefaultAPHostIP.
	APHostIP string
}

// runner carries the injected dependencies and tuning of one operation. Tests
// build one directly to point the device address at a fake and shorten waits.
type runner struct {
	scanner      discovery.WiFiScanner
	log          *slog.Logger
	onStep       func(string)
	scanPresence presenceScanFunc
	hostIfaces   func() ([]probeIface, error)
	newMDNS      func(*net.Interface) presenceSweeper
	newCoIoT     func(*net.Interface) presenceSweeper
	// firmwareClient downloads Gen1 firmware images from the public CDN.
	firmwareClient *http.Client
	// apAddr is where the device answers while the host is on its access point.
	apAddr   string
	apHostIP string

	apReadyTimeout  time.Duration
	lanSettleDelay  time.Duration
	rejoinTimeout   time.Duration
	rejoinInterval  time.Duration
	presenceTimeout time.Duration
	probeTimeout    time.Duration
}

// newRunner builds a runner with the production defaults.
func newRunner(scanner discovery.WiFiScanner, logger *slog.Logger, onStep func(string), apHostIP string) *runner {
	if scanner == nil {
		scanner = discovery.NewWiFiDiscoverer().Scanner
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	r := &runner{
		scanner:         scanner,
		log:             logger,
		onStep:          onStep,
		hostIfaces:      hostProbeIfaces,
		newMDNS:         newMDNSSweeper,
		newCoIoT:        newCoIoTSweeper,
		firmwareClient:  &http.Client{Timeout: firmwareFetchTimeout},
		apAddr:          discovery.DefaultAPIP,
		apHostIP:        apHostIP,
		apReadyTimeout:  apReadyTimeout,
		lanSettleDelay:  lanSettleDelay,
		rejoinTimeout:   lanRejoinTimeout,
		rejoinInterval:  lanRejoinPollInterval,
		presenceTimeout: lanRejoinPresenceTimeout,
		probeTimeout:    lanRejoinProbeTimeout,
	}
	r.scanPresence = r.scanPresenceOnce
	return r
}

// step reports the start of a stage to the caller's OnStep hook.
func (r *runner) step(s string) {
	if r.onStep != nil {
		r.onStep(s)
	}
}

// newHTTP builds a device transport, bound to iface when one is named.
func newHTTP(addr, iface string) *transport.HTTP {
	var opts []transport.Option
	if iface != "" {
		opts = append(opts, transport.WithBindInterface(iface))
	}
	return transport.NewHTTP(addr, opts...)
}

// withGen1 connects to the Gen1 device at addr, reads its identity from /shelly
// and runs fn with the device and its reported MAC. A device that does not answer
// the identity read fails the connection before fn runs.
func (r *runner) withGen1(ctx context.Context, addr, iface string, fn func(*gen1.Device, string) error) error {
	dev := gen1.NewDevice(newHTTP(addr, iface))
	defer func() {
		if err := dev.Close(); err != nil {
			r.log.Debug("close Gen1 connection", "addr", addr, "error", err)
		}
	}()
	info, err := dev.GetDeviceInfo(ctx)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", addr, err)
	}
	return fn(dev, info.MAC)
}

// withGen2 connects to the Gen2+ device at addr, reads its identity with
// Shelly.GetDeviceInfo and runs fn with the RPC client and the device's MAC.
func (r *runner) withGen2(ctx context.Context, addr, iface string, fn func(*rpc.Client, string) error) error {
	client := rpc.NewClient(newHTTP(addr, iface))
	defer func() {
		if err := client.Close(); err != nil {
			r.log.Debug("close RPC connection", "addr", addr, "error", err)
		}
	}()
	info, err := gen2.NewDevice(client).GetDeviceInfo(ctx)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", addr, err)
	}
	return fn(client, info.MAC)
}
