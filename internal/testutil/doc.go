// Package testutil provides testing utilities for the shelly-go library.
//
// This package is internal and not intended for use outside of the library's
// test suite. It provides mock implementations, test fixtures, and helper
// functions that simplify writing unit tests for Shelly device interactions.
//
// # Mock Transport
//
// MockTransport implements the transport.Transport interface and allows
// tests to define custom responses for RPC calls:
//
//	mt := testutil.NewMockTransport()
//	mt.OnCall("Switch.Set", func(params any) (json.RawMessage, error) {
//	    return json.RawMessage(`{"was_on":true}`), nil
//	})
//
//	client := rpc.NewClient(mt)
//	// Use client in tests...
//
// # Mock Devices
//
// MockDevice provides a test implementation of the factory.Device interface:
//
//	device := testutil.NewMockGen1Device("192.168.1.100")
//	device.OnRelay(0, true, nil) // Set relay 0 state to on
//
//	device := testutil.NewMockGen2Device("192.168.1.101")
//	device.OnSwitch(0, &components.SwitchStatus{Output: true})
//
// # Test Fixtures
//
// The fixtures/ directory contains JSON response samples from real devices:
//
//	response := testutil.LoadFixture("gen2/switch_status.json")
//
// # Helper Functions
//
// Various helper functions simplify common test patterns:
//
//	testutil.AssertEqual(t, expected, actual)
//	testutil.AssertNoError(t, err)
//	testutil.MustJSON(t, data)
package testutil
