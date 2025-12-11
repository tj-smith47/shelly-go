package integration

import (
	"testing"
)

func TestCloud_Authentication(t *testing.T) {
	client := RequireCloudClient(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	// Verify client is authenticated by getting devices
	devices, err := client.GetAllDevices(ctx)
	if err != nil {
		t.Fatalf("GetAllDevices() error = %v", err)
	}

	t.Logf("Cloud account has %d devices", len(devices))
}

func TestCloud_GetAllDevices(t *testing.T) {
	client := RequireCloudClient(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	devices, err := client.GetAllDevices(ctx)
	if err != nil {
		t.Fatalf("GetAllDevices() error = %v", err)
	}

	i := 0
	for deviceID, device := range devices {
		t.Logf("Device %d: ID=%s Online=%v",
			i, deviceID, device.Online)

		if deviceID == "" {
			t.Errorf("Device %d has empty ID", i)
		}
		i++
	}
}

func TestCloud_GetDeviceStatus(t *testing.T) {
	client := RequireCloudClient(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	// First get devices
	devices, err := client.GetAllDevices(ctx)
	if err != nil {
		t.Fatalf("GetAllDevices() error = %v", err)
	}

	if len(devices) == 0 {
		t.Skip("No devices in cloud account")
	}

	// Get status of first online device
	var deviceID string
	for id, d := range devices {
		if d.Online {
			deviceID = id
			break
		}
	}

	if deviceID == "" {
		t.Skip("No online devices found")
	}

	status, err := client.GetDeviceStatus(ctx, deviceID)
	if err != nil {
		t.Fatalf("GetDeviceStatus(%s) error = %v", deviceID, err)
	}

	t.Logf("Device %s status: Online=%v", deviceID, status.Online)
}

func TestCloud_GetDevicesV2(t *testing.T) {
	client := RequireCloudClient(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	response, err := client.GetDevicesV2(ctx, nil, true, false)
	if err != nil {
		// V2 API might not be available for all accounts
		t.Skipf("GetDevicesV2() error = %v (may not be supported)", err)
	}

	if response == nil {
		t.Skip("GetDevicesV2() returned nil response")
	}

	t.Logf("V2 API request successful")
}

func TestCloud_SetSwitch(t *testing.T) {
	client := RequireCloudClient(t)
	ctx, cancel := TestContext(t)
	defer cancel()

	// Skip actuating tests unless explicitly enabled
	SkipIfNoActuate(t)

	// First get devices
	devices, err := client.GetAllDevices(ctx)
	if err != nil {
		t.Fatalf("GetAllDevices() error = %v", err)
	}

	// Find an online switch device
	var deviceID string
	var channel int
	for id, d := range devices {
		if d.Online && d.DevInfo != nil {
			// Gen2+ switch devices
			deviceID = id
			channel = 0
			break
		}
	}

	if deviceID == "" {
		t.Skip("No online switch devices found")
	}

	t.Logf("Testing with device %s channel %d", deviceID, channel)

	// Turn on
	err = client.SetSwitch(ctx, deviceID, channel, true)
	if err != nil {
		t.Errorf("SetSwitch(true) error = %v", err)
	}

	// Turn off
	err = client.SetSwitch(ctx, deviceID, channel, false)
	if err != nil {
		t.Errorf("SetSwitch(false) error = %v", err)
	}
}
