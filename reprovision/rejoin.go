package reprovision

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/rpc"
)

// lanRejoinTimeout bounds how long a flow waits for a device to reappear on the
// LAN after the host returns from the AP. It must absorb the device's reboot and
// WiFi association AND the host reacquiring its own DHCP lease after hopping
// back, so it is generous.
const lanRejoinTimeout = 120 * time.Second

// lanRejoinPollInterval is the delay between reachability probes, and
// lanRejoinProbeTimeout bounds a single probe so one slow attempt (the transport
// retries network errors with backoff) cannot blow past the poll cadence and
// starve the remaining probes of the time budget.
const (
	lanRejoinPollInterval = 3 * time.Second
	lanRejoinProbeTimeout = 8 * time.Second
)

// lanRejoinPresenceTimeout bounds a single route-independent presence scan. It is
// shorter than the probe timeout because a multicast listen either hears the
// device's periodic announcement within a couple of cycles or not at all; a long
// window would only delay the next unicast probe attempt.
const lanRejoinPresenceTimeout = 4 * time.Second

// rejoinConfirmation is the outcome of confirming a device returned to the LAN
// after an AP hop. addr is the device's LAN address. bindIface is the host
// interface with a proven unicast route to it ("" = the default route); it is
// only meaningful when writeable is true. writeable distinguishes the two
// strengths of confirmation:
//
//   - writeable=true (via "probe"): a unicast probe reached the device, so this
//     host can both confirm it AND push the post-rejoin configuration to it.
//   - writeable=false (via "mdns"/"coiot"): the device announced itself over
//     link-local multicast, proof it rejoined the LAN that needs no unicast
//     route, but no unicast route from this host reached it. That is enough to
//     learn the address but not to write configuration.
type rejoinConfirmation struct {
	addr      string
	bindIface string
	via       string
	writeable bool
}

// presenceSweeper is a multicast discoverer a presence scan listens with: mDNS
// for every generation, CoIoT for Gen1.
type presenceSweeper interface {
	DiscoverWithContext(ctx context.Context) ([]discovery.DiscoveredDevice, error)
	Stop() error
}

// newMDNSSweeper listens for mDNS announcements on ifi (nil = every interface).
func newMDNSSweeper(ifi *net.Interface) presenceSweeper {
	return discovery.NewMDNSDiscoverer(discovery.WithMDNSInterface(ifi))
}

// newCoIoTSweeper listens for CoIoT announcements on ifi (nil = every interface).
func newCoIoTSweeper(ifi *net.Interface) presenceSweeper {
	return discovery.NewCoIoTDiscoverer(discovery.WithCoIoTInterface(ifi))
}

// presenceScanFunc performs one bounded, route-independent presence scan for the
// device with MAC mac over host interface iface ("" = kernel default), returning
// the device's announced address or "" if it was not seen within timeout. gen1
// enables the CoIoT listener (Gen1-only) alongside mDNS.
type presenceScanFunc func(ctx context.Context, mac, iface string, isGen1 bool, timeout time.Duration) (string, error)

// rejoinProbeFunc runs one unicast reachability probe against addr over host
// interface iface, returning nil when the device answered. A nil return is what
// upgrades a route-independent sighting to a writeable confirmation.
type rejoinProbeFunc func(ctx context.Context, addr, iface string, generation int) error

// rejoinConfig parameterizes raceRejoin. The scanPresence and probe seams are
// injected so the race logic is unit-testable without real multicast or sockets.
type rejoinConfig struct {
	log             *slog.Logger
	scanPresence    presenceScanFunc
	probe           rejoinProbeFunc
	mac             string
	staticIP        string
	candidates      []string
	generation      int
	timeout         time.Duration
	interval        time.Duration
	presenceTimeout time.Duration
	probeTimeout    time.Duration
}

// How a rejoin was confirmed, for diagnostics. "probe" is a writeable unicast
// confirmation; "mdns"/"coiot" are route-independent presence sightings.
const (
	viaProbe = "probe"
	viaMDNS  = "mdns"
	viaCoIoT = "coiot"
)

// rejoinRace holds the shared state of an in-flight confirmation race: the config
// and the best weak (presence-only) sighting any interface has recorded.
type rejoinRace struct {
	weak *rejoinConfirmation
	cfg  rejoinConfig
	mu   sync.Mutex
}

// recordWeak retains the first route-independent sighting, the fallback returned
// when no interface ever gets a unicast route to the device.
func (r *rejoinRace) recordWeak(addr, via string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.weak == nil {
		r.weak = &rejoinConfirmation{addr: addr, via: via}
	}
}

// sharedAddr lets a DHCP address discovered by one interface's presence scan be
// probed by every interface, so the worker that can actually reach it need not be
// the one that saw it announced.
func (r *rejoinRace) sharedAddr() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.weak == nil {
		return ""
	}
	return r.weak.addr
}

// weakConfirmation returns the retained weak (presence-only) sighting, if any.
func (r *rejoinRace) weakConfirmation() *rejoinConfirmation {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.weak
}

// probeOnce runs one unicast probe bounded by the configured probe timeout.
func (r *rejoinRace) probeOnce(ctx context.Context, addr, iface string) error {
	pctx, cancel := context.WithTimeout(ctx, r.cfg.probeTimeout)
	defer cancel()
	return r.cfg.probe(pctx, addr, iface, r.cfg.generation)
}

// tick runs one confirmation attempt over a single interface: probe the best-known
// target unicast, then, on failure or a still-unknown address, fall back to a
// route-free presence scan that records a weak sighting and, on a first DHCP
// sighting, probes it at once. It returns a writeable confirmation and true only
// when a unicast probe reached the device.
func (r *rejoinRace) tick(ctx context.Context, iface string) (rejoinConfirmation, bool) {
	target := r.cfg.staticIP
	if target == "" {
		target = r.sharedAddr()
	}
	if target != "" && r.probeOnce(ctx, target, iface) == nil {
		return rejoinConfirmation{addr: target, bindIface: iface, via: viaProbe, writeable: true}, true
	}

	addr, err := r.cfg.scanPresence(ctx, r.cfg.mac, iface, r.cfg.generation == 1, r.cfg.presenceTimeout)
	if err != nil {
		r.cfg.log.Debug("rejoin: presence scan failed", "iface", ifaceLabel(iface), "error", err)
	}
	if addr == "" {
		return rejoinConfirmation{}, false
	}
	r.recordWeak(addr, presenceProto(r.cfg.generation == 1))
	if target == "" && r.probeOnce(ctx, addr, iface) == nil {
		return rejoinConfirmation{addr: addr, bindIface: iface, via: viaProbe, writeable: true}, true
	}
	return rejoinConfirmation{}, false
}

// raceRejoin confirms a device rejoined the LAN by racing every route-independent
// signal across every candidate host interface, first success wins. Each interface
// runs its own worker ticking until a unicast probe succeeds or the deadline
// elapses.
//
// A unicast probe success is the strong outcome (writeable=true) and ends the race
// immediately. A presence sighting with no reachable unicast route is the weak
// outcome (writeable=false), returned only if the deadline elapses before any
// worker gets a unicast route: the dual-homed or AP-isolated case where the device
// is provably back yet unreachable from this host. If nothing is seen at all, the
// device never came back and an error wrapping ErrNotRejoined is returned.
func raceRejoin(ctx context.Context, cfg *rejoinConfig) (rejoinConfirmation, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	race := &rejoinRace{cfg: *cfg}
	strong := make(chan rejoinConfirmation, 1)

	var wg sync.WaitGroup
	for _, iface := range cfg.candidates {
		wg.Go(func() {
			for ctx.Err() == nil {
				if conf, ok := race.tick(ctx, iface); ok {
					select {
					case strong <- conf:
					default:
					}
					return
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(cfg.interval):
				}
			}
		})
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case res := <-strong:
		cancel()
		<-done
		return res, nil
	case <-done:
		select {
		case res := <-strong:
			return res, nil
		default:
		}
		if w := race.weakConfirmation(); w != nil {
			return *w, nil
		}
		return rejoinConfirmation{}, fmt.Errorf("%w within %s", ErrNotRejoined, cfg.timeout)
	}
}

// presenceProto names the route-independent protocol a weak sighting came from,
// for diagnostics. Gen1 devices announce over both mDNS and CoIoT; the CoIoT
// listener is the Gen1-specific addition, so Gen1 sightings are attributed to it.
func presenceProto(isGen1 bool) string {
	if isGen1 {
		return viaCoIoT
	}
	return viaMDNS
}

// ifaceLabel renders an interface name for logs, naming the default route explicitly.
func ifaceLabel(iface string) string {
	if iface == "" {
		return "default route"
	}
	return iface
}

// confirmRejoin confirms a device returned to the LAN after an AP hop using
// route-independent presence signals raced against unicast probes across every
// candidate host interface (see raceRejoin). A probe only counts when the device
// that answers reports mac, so a different device holding the address is never
// mistaken for the one that left its AP.
func (r *runner) confirmRejoin(ctx context.Context, generation int, staticIP, mac string) (rejoinConfirmation, error) {
	r.step("waiting for the device on the LAN")
	ifaces, ifErr := r.hostIfaces()
	if ifErr != nil {
		r.log.Debug("rejoin: host interface enumeration failed, using default route", "error", ifErr)
	}

	return raceRejoin(ctx, &rejoinConfig{
		log:          r.log,
		scanPresence: r.scanPresence,
		probe: func(ctx context.Context, addr, iface string, generation int) error {
			return r.probeReachable(ctx, addr, iface, generation, mac)
		},
		mac:             mac,
		staticIP:        staticIP,
		candidates:      rejoinCandidateInterfaces(staticIP, ifaces),
		generation:      generation,
		timeout:         r.rejoinTimeout,
		interval:        r.rejoinInterval,
		presenceTimeout: r.presenceTimeout,
		probeTimeout:    r.probeTimeout,
	})
}

// probeReachable runs a single generation-aware reachability probe against addr
// over the host interface iface ("" = default route). The probe is
// generation-aware (Gen1 /settings vs a Gen2+ RPC) because the target keeps the
// backup's generation and a Gen2 device does not serve the Gen1 /settings
// endpoint. When wantMAC is known, a device reporting a different MAC fails the
// probe.
func (r *runner) probeReachable(ctx context.Context, addr, iface string, generation int, wantMAC string) error {
	check := func(gotMAC string) error {
		got := normalizeMAC(gotMAC)
		if wantMAC != "" && got != "" && got != wantMAC {
			return fmt.Errorf("device at %s is %s, not %s", addr, got, wantMAC)
		}
		return nil
	}
	if generation == 1 {
		return r.withGen1(ctx, addr, iface, func(dev *gen1.Device, mac string) error {
			if _, err := dev.GetSettings(ctx); err != nil {
				return err
			}
			return check(mac)
		})
	}
	return r.withGen2(ctx, addr, iface, func(client *rpc.Client, mac string) error {
		if _, err := client.Call(ctx, "Shelly.GetDeviceInfo", nil); err != nil {
			return err
		}
		return check(mac)
	})
}

// rejoinCandidateInterfaces lists the host interfaces to confirm a rejoin over.
// With a known static IP the subnet is known, so only same-subnet interfaces
// (plus the default route) are worth trying. On DHCP the device's address, and
// thus its subnet, is not yet known, so every host interface is a candidate: the
// device announces on exactly one segment and binding a presence listener to each
// is how a multi-homed host hears it.
func rejoinCandidateInterfaces(staticIP string, ifaces []probeIface) []string {
	if staticIP != "" {
		return selectProbeBindInterfaces(net.ParseIP(staticIP), ifaces)
	}

	candidates := []string{""}
	seen := map[string]bool{"": true}
	for _, ifc := range ifaces {
		if seen[ifc.Name] {
			continue
		}
		seen[ifc.Name] = true
		candidates = append(candidates, ifc.Name)
	}
	return candidates
}

// scanPresenceOnce performs one route-independent presence scan for mac,
// returning the device's announced address or "" if it did not announce within
// timeout. It listens over mDNS (all generations) and, for Gen1, CoIoT, each
// bound to the given host interface so a multi-homed host hears announcements
// arriving on a non-default segment. Both are link-local multicast, so a sighting
// proves the device is on the segment reachable via iface without needing a
// unicast route.
func (r *runner) scanPresenceOnce(
	ctx context.Context,
	mac, iface string,
	isGen1 bool,
	timeout time.Duration,
) (string, error) {
	want := normalizeMAC(mac)
	if want == "" {
		return "", fmt.Errorf("invalid MAC address: %q", mac)
	}

	var ifi *net.Interface
	if iface != "" {
		var err error
		ifi, err = net.InterfaceByName(iface)
		if err != nil {
			return "", fmt.Errorf("resolve interface %q: %w", iface, err)
		}
	}

	scanCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	found := make(chan string, 2)
	var wg sync.WaitGroup
	report := func(addr string) {
		if addr == "" {
			return
		}
		select {
		case found <- addr:
		default:
		}
	}

	sweep := func(proto string, d presenceSweeper) {
		wg.Go(func() {
			defer func() {
				if err := d.Stop(); err != nil {
					r.log.Debug("stopping presence scan", "proto", proto, "error", err)
				}
			}()
			report(r.scanForMAC(scanCtx, d.DiscoverWithContext, want))
		})
	}
	sweep(viaMDNS, r.newMDNS(ifi))
	if isGen1 {
		sweep(viaCoIoT, r.newCoIoT(ifi))
	}

	go func() { wg.Wait(); close(found) }()

	addr, ok := <-found
	if !ok {
		return "", nil
	}
	return addr, nil
}

// scanForMAC runs a single context-bounded discovery sweep and returns the address
// of the first device whose MAC matches want, or "" if none did.
func (r *runner) scanForMAC(
	ctx context.Context,
	sweep func(context.Context) ([]discovery.DiscoveredDevice, error),
	want string,
) string {
	devices, err := sweep(ctx)
	if err != nil {
		r.log.Debug("rejoin: presence sweep error", "error", err)
	}
	for i := range devices {
		if normalizeMAC(devices[i].MACAddress) == want {
			return devices[i].Address.String()
		}
	}
	return ""
}

// normalizeMAC returns mac as 12 upper-case hex digits with separators removed,
// or "" when it does not hold exactly 12 hex digits.
func normalizeMAC(mac string) string {
	result := make([]byte, 0, 12)
	for i := range len(mac) {
		switch c := mac[i]; {
		case c >= '0' && c <= '9', c >= 'A' && c <= 'F':
			result = append(result, c)
		case c >= 'a' && c <= 'f':
			result = append(result, c-'a'+'A')
		}
	}
	if len(result) != 12 {
		return ""
	}
	return string(result)
}
