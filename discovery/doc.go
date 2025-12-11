// Package discovery provides device discovery functionality for Shelly devices.
//
// This package implements multiple discovery protocols to find Shelly devices
// on the local network:
//
//   - mDNS/Zeroconf: Standard multicast DNS discovery for devices advertising
//     the _shelly._tcp.local service
//   - CoIoT: CoAP-based multicast discovery for Gen1 devices
//   - BLE: Bluetooth Low Energy discovery for device provisioning
//
// # Quick Start
//
// Use the Scanner interface for a unified discovery experience:
//
//	scanner := discovery.NewScanner()
//	defer scanner.Stop()
//
//	devices, err := scanner.Scan(ctx, 5*time.Second)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, device := range devices {
//	    fmt.Printf("Found: %s (%s) at %s\n",
//	        device.Name, device.Model, device.Address)
//	}
//
// # mDNS Discovery
//
// For Gen2+ devices that advertise via mDNS:
//
//	mdns := discovery.NewMDNS()
//	defer mdns.Stop()
//
//	devices, err := mdns.Discover(ctx, 5*time.Second)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # CoIoT Discovery
//
// For Gen1 devices that broadcast status via CoAP:
//
//	coiot := discovery.NewCoIoT()
//	defer coiot.Stop()
//
//	devices, err := coiot.Discover(ctx, 10*time.Second)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Device Identification
//
// The identify subpackage provides device fingerprinting:
//
//	info, err := discovery.Identify(ctx, "192.168.1.100")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Device: %s (Gen%d)\n", info.Model, info.Generation)
//
// # Factory Pattern
//
// Use the factory to create appropriate device instances from discovery results:
//
//	device, err := factory.FromDiscoveryResult(result)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// The factory automatically detects the device generation and returns
// the appropriate implementation (gen1.Device or gen2.Device).
package discovery
