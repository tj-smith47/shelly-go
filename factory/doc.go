// Package factory provides device creation utilities.
//
// The factory package simplifies device creation by automatically detecting
// the device generation and returning the appropriate implementation.
// It can create devices from discovery results, addresses, or manual
// configuration.
//
// Basic usage:
//
//	// Create device from address with auto-detection
//	device, err := factory.FromAddress("192.168.1.100")
//
//	// Create device from discovery result
//	scanner := discovery.NewScanner()
//	devices, _ := scanner.Scan(5 * time.Second)
//	for i := range devices {
//	    device, _ := factory.FromDiscovery(&devices[i])
//	    // Use device...
//	}
//
//	// Create device with authentication
//	device, err := factory.FromAddress("192.168.1.100",
//	    factory.WithAuth("admin", "password"))
package factory
