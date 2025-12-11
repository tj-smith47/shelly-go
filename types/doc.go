// Package types provides core interfaces and type definitions used throughout
// the shelly-go library.
//
// This package defines the fundamental abstractions that all device implementations,
// components, and transports must adhere to. It provides a consistent API across
// different device generations and communication protocols.
//
// # Core Interfaces
//
// The package defines several key interfaces:
//
//   - Device: Represents a Shelly device with device-level operations
//   - Component: Represents a device component (switch, cover, light, etc.)
//   - Transport: Handles communication with devices
//
// # Type Safety
//
// All types are designed to be type-safe while remaining extensible for future
// firmware updates. The RawFields pattern allows capturing unknown fields without
// breaking compatibility.
//
// # Error Handling
//
// Standard error types are defined as sentinel errors that can be checked with
// errors.Is():
//
//	if errors.Is(err, types.ErrNotFound) {
//	    // Handle not found error
//	}
//
// # Example Usage
//
//	// Working with device info
//	info, err := device.GetDeviceInfo(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Device: %s (Gen%d)\n", info.Model, info.Generation)
//
//	// Checking errors
//	err := component.Set(ctx, value)
//	if errors.Is(err, types.ErrAuth) {
//	    // Handle authentication error
//	}
package types
