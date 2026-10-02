// Package reprovision puts a Shelly device that sits at its factory WiFi access
// point back onto the LAN, from a host that has a WiFi interface.
//
// Every operation hops the host onto the device's open access point, talks to the
// device at discovery.DefaultAPIP, and returns the host to its home network on every
// exit path, including failure and cancellation:
//
//   - Restore applies a backup: the network settings at the access point, then,
//     once the device is back on the LAN, the full configuration.
//   - Onboard writes only the WiFi credentials and confirms the device rejoined.
//   - Inspect reads the device's identity and stored WiFi station settings.
//   - ScanAPs lists the Shelly factory access points in range.
//
// Before any write the device's MAC is checked against the MAC suffix in the
// access point name, so a write never reaches a device the caller did not name.
// A Gen1 device on firmware older than the backup's is flashed at the access point
// (the image is downloaded before the hop and served from the host), and the
// station write is gated on the device holding a stable uptime.
//
// # Usage
//
//	res, err := reprovision.Restore(ctx, &reprovision.RestoreOptions{
//	    APSSID: "ShellyBulbDuo-D12965",
//	    Backup: bkp,
//	    Network: reprovision.Network{StaticIP: "10.23.47.219",
//	        Gateway: "10.23.47.1", Netmask: "255.255.254.0"},
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Println("device is at", res.Address)
//
// The WiFi network defaults to the backup's station settings, then to the host's
// current network with the passphrase the host has stored for it. Set
// Network.Open to join a network that takes no passphrase. A static address
// given without a gateway, netmask or DNS takes each one from the backup's
// static settings (backup.ResolveStaticNetwork) and is refused with
// ErrIncompleteStaticNetwork when no gateway or netmask is found.
//
// The result's SeenVia says how the device was seen back on the LAN, and a
// missing Gen1 firmware image is reported as a *FirmwareUnavailableError naming
// the device's firmware and the backup's.
//
// # Host requirements
//
// Joining an access point changes the host's WiFi connection, so the host must not
// depend on that interface for anything else while an operation runs. On Linux the
// hop uses the platform scanner from the discovery package (NetworkManager or
// wpa_supplicant); confirming the device on the LAN over a specific interface needs
// CAP_NET_RAW. Supply RestoreOptions.Scanner to use a different WiFi backend.
package reprovision
