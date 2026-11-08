package reprovision

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/gen2/components"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// AP is a Shelly factory access point in range of the host.
type AP struct {
	// SSID is the access point name, e.g. "ShellyBulbDuo-D12965".
	SSID string
	// MACSuffix is the device MAC suffix the name carries, upper-case, or ""
	// when the name has none.
	MACSuffix string
	// Model is the device model the SSID encodes, lower-case without the
	// "shelly" prefix (for example "bulbduo" or "plus1pm"), or "" when the
	// SSID does not name one.
	Model string
	// Signal is the signal strength reported by the scanner.
	Signal int
	// Generation is the device generation inferred from Model, or 0 when it
	// cannot be inferred.
	Generation int
	// Secured reports whether the scanner saw any WiFi security on the access
	// point; factory access points are normally open.
	Secured bool
}

// ScanAPs scans for Shelly factory access points in range of the host. A nil
// scanner uses the platform scanner. Each access point appears once, with its
// strongest signal, sorted by SSID.
func ScanAPs(ctx context.Context, scanner discovery.WiFiScanner) ([]AP, error) {
	if scanner == nil {
		scanner = discovery.NewWiFiDiscoverer().Scanner
	}
	networks, err := discovery.NewWiFiDiscovererWithScanner(scanner).ScanNetworks(ctx)
	if err != nil {
		return nil, err
	}
	best := make(map[string]AP, len(networks))
	for i := range networks {
		n := &networks[i]
		if cur, ok := best[n.SSID]; ok && cur.Signal >= n.Signal {
			continue
		}
		best[n.SSID] = AP{
			SSID:       n.SSID,
			MACSuffix:  macSuffixFromAPSSID(n.SSID),
			Signal:     n.Signal,
			Model:      n.DeviceType,
			Generation: int(discovery.InferGenerationFromModel(n.DeviceType)),
			Secured:    n.Security != "",
		}
	}
	aps := make([]AP, 0, len(best))
	for _, ap := range best {
		aps = append(aps, ap)
	}
	sort.Slice(aps, func(i, j int) bool { return aps[i].SSID < aps[j].SSID })
	return aps, nil
}

// Onboard joins a device at its factory access point to a WiFi network. It hops
// the host onto the access point, checks the device's MAC against the access
// point name, writes the station settings, returns the host to its home network
// and looks for the device on the LAN.
//
// A device that took the settings but was not found on the LAN is not an error:
// the result has an empty Address and a Note saying why.
func Onboard(ctx context.Context, opts *OnboardOptions) (*OnboardResult, error) {
	if opts == nil {
		return nil, fmt.Errorf("%w: options are required", types.ErrInvalidParam)
	}
	if opts.APSSID == "" {
		return nil, fmt.Errorf("%w: APSSID is required", types.ErrInvalidParam)
	}
	r := newRunner(opts.Scanner, opts.Logger, opts.OnStep, opts.APHostIP)
	r.disableAP = opts.DisableAP
	r.identity = identityOptions{name: opts.Name, tz: opts.Timezone, disableCloud: opts.DisableCloud}
	return r.onboard(ctx, opts.APSSID, &opts.Network, opts.Generation)
}

// onboard runs an Onboard with the runner's dependencies.
func (r *runner) onboard(
	ctx context.Context,
	apSSID string,
	override *Network,
	generation int,
) (*OnboardResult, error) {
	join, err := r.resolveJoinNetwork(ctx, &Network{}, override)
	if err != nil {
		return nil, err
	}

	res := &OnboardResult{}
	var configErr error
	hopErr := r.withAPHop(ctx, apSSID, &join, func(ctx context.Context) error {
		r.step("confirming the device identity")
		res.Generation = generation
		var mac string
		info, idErr := discovery.Identify(ctx, r.apAddr)
		switch {
		case idErr == nil:
			if generation != 0 && generation != int(info.Generation) {
				r.log.Debug("device generation differs from the one supplied; using the device's",
					"supplied", generation, "device", info.Generation)
			}
			res.Generation = int(info.Generation)
			mac = info.MACAddress
		case generation == 0:
			configErr = fmt.Errorf("identify device: %w", idErr)
			return configErr
		}
		// With no MAC suffix in the SSID, a failed identity read is only logged
		// and the supplied generation is used.
		res.MAC, configErr = r.checkAPIdentity(apSSID, mac, idErr)
		if configErr != nil {
			return configErr
		}
		r.step("writing WiFi settings")
		configErr = r.configureWiFiAtAP(ctx, res.Generation, &join)
		return configErr
	})
	if configErr != nil {
		return res, fmt.Errorf("failed to configure WiFi on device: %w", configErr)
	}
	if hopErr != nil {
		return res, fmt.Errorf("AP hop for %q failed: %w", apSSID, hopErr)
	}

	// Onboarding needs only the device's address, so a presence sighting with no
	// unicast route from this host still counts; a total miss becomes a note.
	conf, err := r.confirmRejoin(ctx, res.Generation, join.StaticIP, res.MAC)
	if err != nil {
		res.Note = fmt.Sprintf("provisioned but %v", err)
		return res, nil
	}
	if !conf.writeable {
		r.log.Debug("onboard: device seen with no unicast route from this host", "via", conf.via, "addr", conf.addr)
	}
	res.Address = conf.addr
	res.Reachable = conf.writeable
	res.SeenVia = conf.via
	r.afterRejoin(ctx, res, &conf)
	return res, nil
}

// afterRejoin runs the writes that need the device on the LAN: turning its
// access point off, then applying the name, timezone and cloud setting.
func (r *runner) afterRejoin(ctx context.Context, res *OnboardResult, conf *rejoinConfirmation) {
	if r.disableAP && res.Generation != 1 && conf.writeable {
		r.step("turning the device's access point off")
		if apErr := r.disableGen2AP(ctx, conf.addr, conf.bindIface); apErr != nil {
			res.Note = fmt.Sprintf("joined but its access point is still on: %v", apErr)
		} else {
			res.APDisabled = true
		}
	}
	switch {
	case conf.writeable:
		r.applyIdentity(ctx, res, conf.addr, conf.bindIface)
	case r.identity.requested() != nil:
		res.Warnings = append(res.Warnings,
			strings.Join(r.identity.requested(), ", ")+" not applied: device not reachable on the LAN")
	}
}

// requested lists the settings asked for, named as OnboardResult.Applied names
// them.
func (o identityOptions) requested() []string {
	var out []string
	if o.name != "" {
		out = append(out, "name")
	}
	if o.tz != "" {
		out = append(out, "timezone")
	}
	if o.disableCloud {
		out = append(out, "cloud")
	}
	return out
}

// applyIdentity writes the requested name, timezone and cloud setting in that
// order. Each is independent: one that fails is reported in res.Warnings and
// the others still run.
func (r *runner) applyIdentity(ctx context.Context, res *OnboardResult, addr, iface string) {
	if len(r.identity.requested()) == 0 {
		return
	}
	r.step("applying the device name, timezone and cloud setting")
	gen1Steps := map[string]func(context.Context, *gen1.Device) error{
		"name":     func(ctx context.Context, d *gen1.Device) error { return d.SetName(ctx, r.identity.name) },
		"timezone": func(ctx context.Context, d *gen1.Device) error { return d.SetTimezone(ctx, r.identity.tz) },
		"cloud":    func(ctx context.Context, d *gen1.Device) error { return d.SetCloud(ctx, false) },
	}
	gen2Steps := map[string]func(context.Context, *rpc.Client) error{
		"name": func(ctx context.Context, c *rpc.Client) error {
			return components.NewSys(c).SetConfig(ctx, &components.SysConfig{
				Device: &components.SysDeviceConfig{Name: &r.identity.name},
			})
		},
		"timezone": func(ctx context.Context, c *rpc.Client) error {
			return components.NewSys(c).SetConfig(ctx, &components.SysConfig{
				Location: &components.SysLocationConfig{TZ: &r.identity.tz},
			})
		},
		"cloud": func(ctx context.Context, c *rpc.Client) error {
			off := false
			return components.NewCloud(c).SetConfig(ctx, &components.CloudConfig{Enable: &off})
		},
	}
	for _, what := range r.identity.requested() {
		var err error
		if res.Generation == 1 {
			err = r.withGen1(ctx, addr, iface, func(d *gen1.Device, _ string) error {
				return gen1Steps[what](ctx, d)
			})
		} else {
			err = r.withGen2(ctx, addr, iface, func(c *rpc.Client, _ string) error {
				return gen2Steps[what](ctx, c)
			})
		}
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s not applied: %v", what, err))
			continue
		}
		res.Applied = append(res.Applied, what)
	}
}

// disableGen2AP turns off an RPC device's access point over its LAN address.
// A Gen1 device has no such step: it leaves access point mode by itself when
// it joins a network.
func (r *runner) disableGen2AP(ctx context.Context, addr, iface string) error {
	return r.withGen2(ctx, addr, iface, func(client *rpc.Client, _ string) error {
		_, err := client.Call(ctx, "WiFi.SetConfig", json.RawMessage(`{"config":{"ap":{"enable":false}}}`))
		return err
	})
}

// configureWiFiAtAP writes the station settings to the device at its access
// point: Gen1 through /settings/sta, Gen2+ through WiFi.SetConfig.
func (r *runner) configureWiFiAtAP(ctx context.Context, generation int, n *Network) error {
	if generation == 1 {
		return r.withGen1(ctx, r.apAddr, "", func(dev *gen1.Device, _ string) error {
			if n.StaticIP != "" && n.Open {
				return dev.SetWiFiStationStaticOpen(ctx, n.SSID, n.StaticIP, n.Gateway, n.Netmask, n.DNS)
			}
			if n.StaticIP != "" {
				return dev.SetWiFiStationStatic(ctx, n.SSID, n.Password, n.StaticIP, n.Gateway, n.Netmask, n.DNS)
			}
			if n.Open {
				return dev.SetWiFiStationOpen(ctx, n.SSID)
			}
			return dev.SetWiFiStation(ctx, true, n.SSID, n.Password)
		})
	}
	cfg, err := applyGen2WiFiOverride(nil, n)
	if err != nil {
		return err
	}
	return r.withGen2(ctx, r.apAddr, "", func(client *rpc.Client, _ string) error {
		_, callErr := client.Call(ctx, "WiFi.SetConfig", map[string]any{"config": cfg})
		return callErr
	})
}
