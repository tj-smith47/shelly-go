// Package integration provides integration test utilities for testing against
// real Shelly devices and the Shelly Cloud API.
//
// Integration tests are skipped by default when the required environment variables
// or test devices are not available. This allows the test suite to run in CI
// environments without real hardware.
//
// # Environment Variables
//
// The following environment variables control integration test behavior:
//
//   - SHELLY_TEST_GEN1_ADDR: IP address of a Gen1 device for testing
//   - SHELLY_TEST_GEN2_ADDR: IP address of a Gen2+ device for testing
//   - SHELLY_TEST_CLOUD_EMAIL: Email for Cloud API testing
//   - SHELLY_TEST_CLOUD_PASSWORD: Password for Cloud API testing
//   - SHELLY_INTEGRATION_TESTS: Set to "1" to enable integration tests
//
// # Usage
//
//	func TestGen2Switch(t *testing.T) {
//	    device := integration.RequireGen2Device(t)
//	    // Test with real device...
//	}
//
//	func TestCloudAPI(t *testing.T) {
//	    client := integration.RequireCloudClient(t)
//	    // Test with real Cloud API...
//	}
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/cloud"
	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

const (
	// EnvIntegrationTests enables integration tests when set to "1".
	EnvIntegrationTests = "SHELLY_INTEGRATION_TESTS"

	// EnvGen1Addr is the IP address of a Gen1 device for testing.
	EnvGen1Addr = "SHELLY_TEST_GEN1_ADDR"

	// EnvGen2Addr is the IP address of a Gen2+ device for testing.
	EnvGen2Addr = "SHELLY_TEST_GEN2_ADDR"

	// EnvCloudEmail is the email for Cloud API authentication.
	EnvCloudEmail = "SHELLY_TEST_CLOUD_EMAIL"

	// EnvCloudPassword is the password for Cloud API authentication.
	EnvCloudPassword = "SHELLY_TEST_CLOUD_PASSWORD"

	// EnvActuate enables tests that actuate devices (turn on/off switches, relays).
	// CAUTION: Only set this when it's safe to control physical devices.
	EnvActuate = "SHELLY_TEST_ACTUATE"

	// DefaultTimeout is the default timeout for integration test operations.
	DefaultTimeout = 30 * time.Second
)

// Config holds integration test configuration.
type Config struct {
	Gen1Addr      string
	Gen2Addr      string
	CloudEmail    string
	CloudPassword string
	Timeout       time.Duration
	Actuate       bool // Whether to run tests that actuate devices
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() *Config {
	timeout := DefaultTimeout
	if t := os.Getenv("SHELLY_TEST_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	return &Config{
		Gen1Addr:      os.Getenv(EnvGen1Addr),
		Gen2Addr:      os.Getenv(EnvGen2Addr),
		CloudEmail:    os.Getenv(EnvCloudEmail),
		CloudPassword: os.Getenv(EnvCloudPassword),
		Timeout:       timeout,
		Actuate:       os.Getenv(EnvActuate) == "1",
	}
}

// Enabled returns true if integration tests are enabled.
func Enabled() bool {
	return os.Getenv(EnvIntegrationTests) == "1"
}

// SkipIfNoIntegration skips the test if integration tests are disabled.
func SkipIfNoIntegration(t *testing.T) {
	t.Helper()
	if !Enabled() {
		t.Skip("integration tests disabled (set SHELLY_INTEGRATION_TESTS=1 to enable)")
	}
}

// SkipIfNoGen1Device skips the test if no Gen1 device is configured.
func SkipIfNoGen1Device(t *testing.T) {
	t.Helper()
	SkipIfNoIntegration(t)
	if os.Getenv(EnvGen1Addr) == "" {
		t.Skipf("no Gen1 device configured (set %s)", EnvGen1Addr)
	}
}

// SkipIfNoGen2Device skips the test if no Gen2 device is configured.
func SkipIfNoGen2Device(t *testing.T) {
	t.Helper()
	SkipIfNoIntegration(t)
	if os.Getenv(EnvGen2Addr) == "" {
		t.Skipf("no Gen2 device configured (set %s)", EnvGen2Addr)
	}
}

// SkipIfNoCloudCredentials skips the test if no Cloud credentials are configured.
func SkipIfNoCloudCredentials(t *testing.T) {
	t.Helper()
	SkipIfNoIntegration(t)
	if os.Getenv(EnvCloudEmail) == "" || os.Getenv(EnvCloudPassword) == "" {
		t.Skipf("no Cloud credentials configured (set %s and %s)", EnvCloudEmail, EnvCloudPassword)
	}
}

// SkipIfNoActuate skips tests that would actuate devices (turn on/off switches, relays).
// Use this for any test that changes physical device state.
func SkipIfNoActuate(t *testing.T) {
	t.Helper()
	if os.Getenv(EnvActuate) != "1" {
		t.Skipf("actuate tests disabled (set %s=1 to enable device control)", EnvActuate)
	}
}

// ActuateEnabled returns true if actuate tests are enabled.
func ActuateEnabled() bool {
	return os.Getenv(EnvActuate) == "1"
}

// RequireGen1Device returns a Gen1 device for testing, or skips the test.
func RequireGen1Device(t *testing.T) *gen1.Device {
	t.Helper()
	SkipIfNoGen1Device(t)

	config := LoadConfig()
	tr := transport.NewHTTP(config.Gen1Addr)
	device := gen1.NewDevice(tr)

	// Verify device is reachable
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	info, err := device.GetDeviceInfo(ctx)
	if err != nil {
		t.Skipf("Gen1 device at %s not reachable: %v", config.Gen1Addr, err)
	}

	t.Logf("Testing with Gen1 device: %s (%s) at %s", info.ID, info.Model, config.Gen1Addr)
	return device
}

// RequireGen2Device returns a Gen2+ device for testing, or skips the test.
func RequireGen2Device(t *testing.T) *gen2.Device {
	t.Helper()
	SkipIfNoGen2Device(t)

	config := LoadConfig()
	tr := transport.NewHTTP(config.Gen2Addr)
	client := rpc.NewClient(tr)
	device := gen2.NewDevice(client)

	// Verify device is reachable
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	info, err := device.Shelly().GetDeviceInfo(ctx)
	if err != nil {
		t.Skipf("Gen2 device at %s not reachable: %v", config.Gen2Addr, err)
	}

	t.Logf("Testing with Gen2 device: %s (%s) at %s", info.ID, info.Model, config.Gen2Addr)
	return device
}

// RequireCloudClient returns a Cloud API client for testing, or skips the test.
func RequireCloudClient(t *testing.T) *cloud.Client {
	t.Helper()
	SkipIfNoCloudCredentials(t)

	config := LoadConfig()

	// Create client with credentials
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	client, err := cloud.NewClientWithCredentials(ctx, config.CloudEmail, config.CloudPassword)
	if err != nil {
		t.Skipf("Cloud API authentication failed: %v", err)
	}

	t.Log("Testing with Cloud API client")
	return client
}

// TestContext returns a context with the configured timeout.
func TestContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	config := LoadConfig()
	return context.WithTimeout(context.Background(), config.Timeout)
}

// RunWithTimeout runs a test function with a timeout.
func RunWithTimeout(t *testing.T, timeout time.Duration, fn func(ctx context.Context)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(ctx)
	}()

	select {
	case <-done:
		// Test completed
	case <-ctx.Done():
		t.Fatalf("test timed out after %v", timeout)
	}
}
