package reprovision

import (
	"context"
	"fmt"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Inspection is the state read from a device at its factory access point: its
// identity and the WiFi station settings it would use to join a
// network. It shows whether a provisioning write took without the device having
// to reach the LAN first.
type Inspection struct {
	// Model is the device's model code, e.g. "SHBDUO-1".
	Model string
	// MAC is the device's MAC address as the device reports it.
	MAC string
	// Firmware is the device's firmware build.
	Firmware string
	// StaSSID is the WiFi network the device is set to join.
	StaSSID string
	// Ipv4Method is the station addressing mode, "dhcp" or "static".
	Ipv4Method string
	// StaIP is the station address: the live one when the device is connected,
	// otherwise the configured static address.
	StaIP string
	// StaGateway is the configured static gateway.
	StaGateway string
	// Generation is the device generation (1 for Gen1, 2 and up for RPC devices).
	Generation int
	// StaKeySet reports that a station passphrase is stored; the device masks
	// the key itself. An RPC device never returns its
	// passphrase, so there this is true for any configured network that is not open.
	StaKeySet bool
	// StaConnected reports that the device is connected to its station network
	// right now.
	StaConnected bool
}

// Inspect hops the host onto a device's factory access point, reads the device's
// identity and stored WiFi station settings, and returns the host to its home
// network. Nothing is written to the device.
//
// When the WiFi read fails after the device was identified, the partial
// Inspection is returned with the error.
func Inspect(ctx context.Context, opts *InspectOptions) (*Inspection, error) {
	if opts == nil {
		return nil, fmt.Errorf("%w: options are required", types.ErrInvalidParam)
	}
	if opts.APSSID == "" {
		return nil, fmt.Errorf("%w: APSSID is required", types.ErrInvalidParam)
	}
	r := newRunner(opts.Scanner, opts.Logger, opts.OnStep, opts.APHostIP)
	return r.inspect(ctx, opts.APSSID)
}

// inspect runs an Inspect with the runner's dependencies.
func (r *runner) inspect(ctx context.Context, apSSID string) (*Inspection, error) {
	var (
		insp    *Inspection
		readErr error
	)
	hopErr := r.withAPHop(ctx, apSSID, &Network{}, func(ctx context.Context) error {
		r.step("reading the device")
		insp, readErr = r.readDeviceAtAP(ctx)
		return readErr
	})
	if readErr != nil {
		return insp, fmt.Errorf("read device at AP %q: %w", apSSID, readErr)
	}
	if hopErr != nil {
		return nil, fmt.Errorf("AP hop for %q failed: %w", apSSID, hopErr)
	}
	return insp, nil
}

// readDeviceAtAP identifies the device through the /shelly endpoint every
// generation serves and reads its station settings. The identity is returned
// even when the WiFi read fails.
func (r *runner) readDeviceAtAP(ctx context.Context) (*Inspection, error) {
	info, err := discovery.Identify(ctx, r.apAddr)
	if err != nil {
		return nil, fmt.Errorf("identify device: %w", err)
	}
	insp := &Inspection{
		Generation: int(info.Generation),
		Model:      info.Model,
		MAC:        info.MACAddress,
		Firmware:   info.Firmware,
	}
	if info.Generation == types.Gen1 {
		if wifiErr := r.readGen1WiFiAtAP(ctx, insp); wifiErr != nil {
			return insp, fmt.Errorf("read Gen1 WiFi config: %w", wifiErr)
		}
		return insp, nil
	}
	if wifiErr := r.readGen2WiFiAtAP(ctx, insp); wifiErr != nil {
		return insp, fmt.Errorf("read WiFi config: %w", wifiErr)
	}
	return insp, nil
}

// readGen2WiFiAtAP fills insp with an RPC device's stored station settings and
// its live connection state. As for Gen1, the live state is best-effort.
func (r *runner) readGen2WiFiAtAP(ctx context.Context, insp *Inspection) error {
	return r.withGen2(ctx, r.apAddr, "", func(client *rpc.Client, _ string) error {
		var cfg struct {
			Sta struct {
				SSID     string `json:"ssid"`
				Ipv4Mode string `json:"ipv4mode"`
				IP       string `json:"ip"`
				Gw       string `json:"gw"`
				IsOpen   bool   `json:"is_open"`
			} `json:"sta"`
		}
		if err := client.CallResult(ctx, "WiFi.GetConfig", nil, &cfg); err != nil {
			return err
		}
		insp.StaSSID = cfg.Sta.SSID
		insp.StaKeySet = cfg.Sta.SSID != "" && !cfg.Sta.IsOpen
		insp.Ipv4Method = cfg.Sta.Ipv4Mode
		insp.StaIP = cfg.Sta.IP
		insp.StaGateway = cfg.Sta.Gw
		var status struct {
			StaIP  string `json:"sta_ip"`
			Status string `json:"status"`
		}
		if err := client.CallResult(ctx, "WiFi.GetStatus", nil, &status); err == nil {
			insp.StaConnected = status.Status == "got ip"
			if status.StaIP != "" {
				insp.StaIP = status.StaIP
			}
		}
		return nil
	})
}

// readGen1WiFiAtAP fills insp with the device's stored station settings and its
// live connection state. The live state is best-effort: a device whose station
// settings were just changed may not have associated yet, which is what the
// caller wants to observe.
func (r *runner) readGen1WiFiAtAP(ctx context.Context, insp *Inspection) error {
	return r.withGen1(ctx, r.apAddr, "", func(dev *gen1.Device, _ string) error {
		settings, err := dev.GetSettings(ctx)
		if err != nil {
			return err
		}
		if sta := settings.WiFiSta; sta != nil {
			insp.StaSSID = sta.SSID
			insp.StaKeySet = sta.Key != ""
			insp.Ipv4Method = sta.Ipv4Method
			insp.StaIP = sta.IP
			insp.StaGateway = sta.Gw
		}
		if status, statusErr := dev.GetFullStatus(ctx); statusErr == nil && status.WiFiSta != nil {
			insp.StaConnected = status.WiFiSta.Connected
			if status.WiFiSta.IP != "" {
				insp.StaIP = status.WiFiSta.IP
			}
		}
		return nil
	})
}
