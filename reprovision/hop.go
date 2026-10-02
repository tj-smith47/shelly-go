package reprovision

import (
	"cmp"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/tj-smith47/shelly-go/backup"
	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/types"
)

// apReturnHomeTimeout bounds the reconnect-to-home (and AP-block cleanup) that
// runs on return from an AP hop. It runs on a cancel-immune context so it still
// executes when the hop's own context was canceled, but must not block shutdown
// indefinitely.
const apReturnHomeTimeout = 30 * time.Second

// apReadyTimeout bounds how long withAPHop waits for the host to associate with
// a Shelly AP and obtain a route to it before sending the first device request.
const apReadyTimeout = 25 * time.Second

// withAPHop connects the host's WiFi to an open Shelly AP, runs fn (which talks
// to the device at r.apAddr), then returns the host to its original network. home
// supplies the credentials to rejoin the home network on platforms without a
// saved-credential store (nl80211); reconnectCredentials prefers the
// previously-joined network when NetworkManager can restore it from saved
// profiles. An error reaching the AP is returned before fn runs; otherwise fn's
// error is returned.
func (r *runner) withAPHop(ctx context.Context, apSSID string, home *Network, fn func(context.Context) error) error {
	if r.scanner == nil {
		return fmt.Errorf("%w: WiFi scanning not supported on this platform", types.ErrNotSupported)
	}

	// Override the host's static AP-subnet IP when requested (empty is ignored,
	// keeping discovery.DefaultAPHostIP). Only the Linux scanner needs this.
	if setter, ok := r.scanner.(discovery.APHostIPSetter); ok {
		setter.SetAPHostIP(r.apHostIP)
	}

	// Remember current network for reconnection (may fail if not connected).
	originalNet, netErr := r.scanner.CurrentNetwork(ctx)
	if netErr != nil {
		r.log.Debug("AP hop: could not detect current network", "error", netErr)
	}

	// The return is armed before the connect: a connect that fails part-way can
	// already have dropped the home network, and the host must never be left on
	// the device AP. A cancel-immune context lets the reconnect run even when ctx
	// was already canceled, bounded by its own timeout.
	defer func() {
		r.step("returning to the home network")
		returnCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), apReturnHomeTimeout)
		defer cancel()
		r.returnFromAPHop(returnCtx, apSSID, originalNet, home)
	}()

	r.step("joining access point " + apSSID)
	if err := r.scanner.Connect(ctx, apSSID, ""); err != nil {
		return fmt.Errorf("%w: failed to connect to Shelly AP %q: %w", ErrAPUnreachable, apSSID, err)
	}

	// Wait until the AP link is actually up and the device answers at its AP
	// address before sending any request. A fixed DHCP delay races association
	// on slower WiFi stacks, so the first request would leave before a route to
	// the AP subnet exists and fail with a deadline-exceeded error.
	r.waitForAPReady(ctx, r.apReadyTimeout)

	return fn(ctx)
}

// returnFromAPHop reconnects the host to its home network after an AP hop and
// drops the transient device-AP block. It prefers the previously-joined network
// (NetworkManager may have saved creds), falling back to the home credentials
// when that SSID differs and the first attempt fails (nl80211 has no credential
// store). It runs as cleanup, so failures are logged, not returned.
func (r *runner) returnFromAPHop(
	ctx context.Context,
	apSSID string,
	originalNet *discovery.WiFiNetwork,
	home *Network,
) {
	reconnectSSID, reconnectPass := reconnectCredentials(originalNet, home)
	if reconnErr := r.scanner.Connect(ctx, reconnectSSID, reconnectPass); reconnErr != nil {
		r.log.Debug("AP hop: reconnect failed", "ssid", reconnectSSID, "error", reconnErr)
		if reconnectSSID != home.SSID {
			if err := r.scanner.Connect(ctx, home.SSID, home.Password); err != nil {
				r.log.Debug("AP hop: fallback connect failed", "ssid", home.SSID, "error", err)
			}
		}
	}

	// Drop the transient AP block so hops across a fleet of devices do not leave
	// stale (disabled) wpa_supplicant blocks behind. Only the Linux/wpa_cli
	// scanner needs this; other platforms manage AP profiles through the OS.
	if forgetter, ok := r.scanner.(discovery.APNetworkForgetter); ok {
		if err := forgetter.ForgetNetwork(ctx, apSSID); err != nil {
			r.log.Debug("AP hop: forget AP network failed", "ssid", apSSID, "error", err)
		}
	}
}

// reconnectCredentials determines the SSID and password for reconnecting to the
// home network after an AP hop. If the original network differs from home, it
// returns the original SSID with an empty password (NetworkManager may have saved
// credentials). Otherwise it returns the home credentials directly, since nl80211
// has no credential store.
func reconnectCredentials(originalNet *discovery.WiFiNetwork, home *Network) (ssid, password string) {
	if originalNet != nil && originalNet.SSID != "" && originalNet.SSID != home.SSID {
		return originalNet.SSID, ""
	}
	return home.SSID, home.Password
}

// waitForAPReady polls the device's AP address until a TCP connection to its
// HTTP port succeeds or the deadline elapses, confirming the host has associated
// with the AP and obtained a route. The device is unreachable for the first
// second or two after the connect returns, so an immediate request fails before
// any route to the AP subnet exists.
func (r *runner) waitForAPReady(ctx context.Context, timeout time.Duration) {
	addr := r.apAddr
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, "80")
	}
	deadline := time.Now().Add(timeout)
	dialer := net.Dialer{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			if cerr := conn.Close(); cerr != nil {
				r.log.Debug("AP hop: closing readiness probe", "error", cerr)
			}
			return
		}
		r.log.Debug("AP hop: waiting for device", "addr", addr, "error", err)

		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
		}
	}
}

// mergeJoinNetwork lays the override over the backup's network: the open flag,
// SSID, passphrase and static addressing. It refuses an open network with a
// passphrase and a static address with no gateway or netmask.
func mergeJoinNetwork(fromBackup, override *Network) (Network, error) {
	if override.Open && override.Password != "" {
		return Network{}, fmt.Errorf("%w: Network.Open is set, and an open network takes no Network.Password",
			types.ErrInvalidParam)
	}
	join := *fromBackup
	// The backup's open flag belongs to the backup's network, so a password or
	// a different SSID in the override means a secured network.
	join.Open = override.Open || (fromBackup.Open && override.Password == "" &&
		(override.SSID == "" || override.SSID == fromBackup.SSID))
	join.SSID = cmp.Or(override.SSID, join.SSID)
	join.Password = cmp.Or(override.Password, join.Password)
	static, err := backup.ResolveStaticNetwork(
		backup.StaticNetwork{
			IP: override.StaticIP, Gateway: override.Gateway, Netmask: override.Netmask, DNS: override.DNS,
		},
		backup.StaticNetwork{IP: join.StaticIP, Gateway: join.Gateway, Netmask: join.Netmask, DNS: join.DNS})
	if err != nil {
		return Network{}, fmt.Errorf("%w; set Network.Gateway and Network.Netmask", err)
	}
	join.StaticIP, join.Gateway, join.Netmask, join.DNS = static.IP, static.Gateway, static.Netmask, static.DNS
	return join, nil
}

// hostWiFiPassword recovers the passphrase the host has stored for ssid from the
// OS credential store, when the scanner supports it. No Shelly device returns its
// station key, so this lets a device join the network the host is already on.
func (r *runner) hostWiFiPassword(ctx context.Context, ssid string) (string, error) {
	if r.scanner == nil {
		return "", fmt.Errorf("%w: WiFi not supported on this platform", types.ErrNotSupported)
	}
	provider, ok := r.scanner.(discovery.HostNetworkPasswordProvider)
	if !ok {
		return "", fmt.Errorf("%w: host passphrase recovery not supported by this scanner", types.ErrNotSupported)
	}
	return provider.HostNetworkPassword(ctx, ssid)
}

// resolveJoinNetwork determines the network the device joins. No Shelly device
// returns its station key (Gen1 masks it, Gen2+ makes it write-only), so the
// passphrase comes by precedence: the override, then the backup's own key
// (unmasked Gen1 only), then the host's stored credentials for that network. The
// SSID comes from the override, then the backup, then the host's current network.
// An open network (the override's Open, or an open backup station the override
// leaves alone) takes no passphrase. The static addressing follows
// backup.ResolveStaticNetwork.
func (r *runner) resolveJoinNetwork(ctx context.Context, fromBackup, override *Network) (Network, error) {
	join, err := mergeJoinNetwork(fromBackup, override)
	if err != nil {
		return Network{}, err
	}
	if join.SSID == "" && r.scanner != nil {
		if current, err := r.scanner.CurrentNetwork(ctx); err == nil && current != nil {
			join.SSID = current.SSID
		} else {
			r.log.Debug("host is not on a WiFi network", "error", err)
		}
	}
	if join.Open {
		if join.SSID == "" {
			return Network{}, fmt.Errorf("%w: Network.Open is set but there is no SSID to join; set Network.SSID",
				types.ErrInvalidParam)
		}
		join.Password = ""
		return join, nil
	}
	if join.Password == "" && join.SSID != "" {
		if pw, lookupErr := r.hostWiFiPassword(ctx, join.SSID); lookupErr != nil {
			r.log.Debug("host passphrase not recovered", "ssid", join.SSID, "error", lookupErr)
		} else {
			r.log.Debug("recovered passphrase from host credentials", "ssid", join.SSID)
			join.Password = pw
		}
	}
	if join.Password == "" {
		return Network{}, fmt.Errorf(
			"%w for %q: Shelly devices return no station key and none was found in this "+
				"host's stored credentials; set Network.Password", ErrNoPassphrase, join.SSID)
	}
	return join, nil
}
