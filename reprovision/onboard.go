package reprovision

import (
	"context"
	"fmt"
	"sort"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/gen1"
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
	// Signal is the signal strength reported by the scanner.
	Signal int
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
		best[n.SSID] = AP{SSID: n.SSID, MACSuffix: macSuffixFromAPSSID(n.SSID), Signal: n.Signal}
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
	return res, nil
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
