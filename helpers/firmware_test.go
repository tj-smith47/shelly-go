package helpers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/tj-smith47/shelly-go/factory"
	"github.com/tj-smith47/shelly-go/types"
)

// TestFirmwareResults tests the FirmwareResults helper methods.
func TestFirmwareResults(t *testing.T) {
	t.Run("AllSuccessful", func(t *testing.T) {
		results := FirmwareResults{
			{Success: true},
			{Success: true},
		}
		if !results.AllSuccessful() {
			t.Errorf("AllSuccessful() should return true")
		}

		results = append(results, FirmwareResult{Success: false})
		if results.AllSuccessful() {
			t.Errorf("AllSuccessful() should return false with failures")
		}
	})

	t.Run("Failures", func(t *testing.T) {
		results := FirmwareResults{
			{Success: true},
			{Success: false, Error: types.ErrTimeout},
			{Success: true},
			{Success: false, Error: types.ErrAuth},
		}

		failures := results.Failures()
		if len(failures) != 2 {
			t.Errorf("Failures() returned %d results, want 2", len(failures))
		}
	})

	t.Run("UpdatesAvailable", func(t *testing.T) {
		results := FirmwareResults{
			{Success: true, Info: &FirmwareInfo{HasUpdate: true}},
			{Success: true, Info: &FirmwareInfo{HasUpdate: false}},
			{Success: false},
			{Success: true, Info: nil},
		}

		updates := results.UpdatesAvailable()
		if len(updates) != 1 {
			t.Errorf("UpdatesAvailable() returned %d results, want 1", len(updates))
		}
	})

	t.Run("empty results", func(t *testing.T) {
		results := FirmwareResults{}
		if !results.AllSuccessful() {
			t.Errorf("AllSuccessful() should return true for empty results")
		}
	})
}

// TestGetFirmwareInfo tests retrieving firmware info.
func TestGetFirmwareInfo(t *testing.T) {
	ctx := context.Background()

	t.Run("gen1 device", func(t *testing.T) {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/shelly":
				_, _ = w.Write([]byte(`{"type":"SHSW-1","mac":"AA:BB:CC:DD:EE:FF","fw":"20210115-103035/v1.9.4@d7e01e1"}`))
			case "/ota/check":
				_, _ = w.Write([]byte(`{"status":"ok","has_update":true,"new_version":"20220101-120000/v1.10.0@abcdef1"}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		})
		defer server.Close()

		info, err := GetFirmwareInfo(ctx, dev)
		if err != nil {
			t.Fatalf("GetFirmwareInfo() error: %v", err)
		}
		if info == nil {
			t.Fatal("GetFirmwareInfo() returned nil")
		}
	})

	t.Run("gen2 device", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				resp := `{"jsonrpc":"2.0","id":1,"result":{"id":"test","mac":"AA:BB:CC:DD:EE:FF","model":"SNSW-001P16EU","gen":2,"fw_id":"20220101","ver":"0.12.0","app":"Plus1PM"}}`
				return json.RawMessage(resp), nil
			case "Shelly.CheckForUpdate":
				resp := `{"jsonrpc":"2.0","id":1,"result":{"stable":{"version":"0.13.0"}}}`
				return json.RawMessage(resp), nil
			default:
				return nil, types.ErrRPCMethod
			}
		})

		info, err := GetFirmwareInfo(ctx, dev)
		if err != nil {
			t.Fatalf("GetFirmwareInfo() error: %v", err)
		}
		if info == nil {
			t.Fatal("GetFirmwareInfo() returned nil")
		}
		if info.CurrentVersion != "0.12.0" {
			t.Errorf("CurrentVersion = %v, want 0.12.0", info.CurrentVersion)
		}
		if !info.HasUpdate {
			t.Errorf("HasUpdate should be true")
		}
		if info.AvailableVersion != "0.13.0" {
			t.Errorf("AvailableVersion = %v, want 0.13.0", info.AvailableVersion)
		}
	})

	t.Run("gen2 no update", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				resp := `{"jsonrpc":"2.0","id":1,"result":{"id":"test","mac":"AA:BB:CC:DD:EE:FF","model":"SNSW-001P16EU","gen":2,"fw_id":"20220101","ver":"0.13.0","app":"Plus1PM"}}`
				return json.RawMessage(resp), nil
			case "Shelly.CheckForUpdate":
				resp := `{"jsonrpc":"2.0","id":1,"result":{"stable":{"version":"0.13.0"}}}`
				return json.RawMessage(resp), nil
			default:
				return nil, types.ErrRPCMethod
			}
		})

		info, err := GetFirmwareInfo(ctx, dev)
		if err != nil {
			t.Fatalf("GetFirmwareInfo() error: %v", err)
		}
		if info.HasUpdate {
			t.Errorf("HasUpdate should be false when versions match")
		}
	})

	t.Run("nil gen1 device", func(t *testing.T) {
		dev := &factory.Gen1Device{Device: nil}
		_, err := GetFirmwareInfo(ctx, dev)
		if err != types.ErrNilDevice {
			t.Errorf("GetFirmwareInfo() error = %v, want %v", err, types.ErrNilDevice)
		}
	})

	t.Run("nil gen2 device", func(t *testing.T) {
		dev := &factory.Gen2Device{Device: nil}
		_, err := GetFirmwareInfo(ctx, dev)
		if err != types.ErrNilDevice {
			t.Errorf("GetFirmwareInfo() error = %v, want %v", err, types.ErrNilDevice)
		}
	})

	t.Run("unsupported device", func(t *testing.T) {
		dev := &unsupportedDevice{}
		_, err := GetFirmwareInfo(ctx, dev)
		if err != types.ErrUnsupportedDevice {
			t.Errorf("GetFirmwareInfo() error = %v, want %v", err, types.ErrUnsupportedDevice)
		}
	})

	t.Run("gen1 device info error", func(t *testing.T) {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		defer server.Close()

		_, err := GetFirmwareInfo(ctx, dev)
		if err == nil {
			t.Errorf("GetFirmwareInfo() should fail on device info error")
		}
	})

	t.Run("gen2 device info error", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			return nil, types.ErrRPCMethod
		})

		_, err := GetFirmwareInfo(ctx, dev)
		if err == nil {
			t.Errorf("GetFirmwareInfo() should fail on device info error")
		}
	})
}

// TestCheckFirmwareUpdates tests checking for firmware updates on multiple devices.
func TestCheckFirmwareUpdates(t *testing.T) {
	ctx := context.Background()

	t.Run("multiple devices", func(t *testing.T) {
		dev1, server1 := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/shelly":
				_, _ = w.Write([]byte(`{"type":"SHSW-1","mac":"AA:BB:CC:DD:EE:FF","fw":"v1.9.4"}`))
			case "/ota/check":
				_, _ = w.Write([]byte(`{"status":"ok","has_update":true,"new_version":"v1.10.0"}`))
			}
		})
		defer server1.Close()

		dev2 := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				resp := `{"jsonrpc":"2.0","id":1,"result":{"id":"test","mac":"11:22:33:44:55:66","model":"SNSW-001P16EU","gen":2,"fw_id":"20220101","ver":"0.12.0","app":"Plus1PM"}}`
				return json.RawMessage(resp), nil
			case "Shelly.CheckForUpdate":
				resp := `{"jsonrpc":"2.0","id":1,"result":{}}`
				return json.RawMessage(resp), nil
			default:
				return nil, types.ErrRPCMethod
			}
		})

		results := CheckFirmwareUpdates(ctx, []factory.Device{dev1, dev2})
		if len(results) != 2 {
			t.Errorf("CheckFirmwareUpdates() returned %d results, want 2", len(results))
		}
	})

	t.Run("empty device list", func(t *testing.T) {
		results := CheckFirmwareUpdates(ctx, []factory.Device{})
		if len(results) != 0 {
			t.Errorf("CheckFirmwareUpdates() returned %d results, want 0", len(results))
		}
	})
}

// TestUpdateFirmware tests triggering firmware updates.
func TestUpdateFirmware(t *testing.T) {
	ctx := context.Background()

	t.Run("gen1 device", func(t *testing.T) {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/ota" && r.URL.Query().Get("update") == "true" {
				_, _ = w.Write([]byte(`{"status":"ok"}`))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
		})
		defer server.Close()

		err := UpdateFirmware(ctx, dev)
		if err != nil {
			t.Errorf("UpdateFirmware() error: %v", err)
		}
	})

	t.Run("gen2 device", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Shelly.Update" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		err := UpdateFirmware(ctx, dev)
		if err != nil {
			t.Errorf("UpdateFirmware() error: %v", err)
		}
	})

	t.Run("nil gen1 device", func(t *testing.T) {
		dev := &factory.Gen1Device{Device: nil}
		err := UpdateFirmware(ctx, dev)
		if err != types.ErrNilDevice {
			t.Errorf("UpdateFirmware() error = %v, want %v", err, types.ErrNilDevice)
		}
	})

	t.Run("nil gen2 device", func(t *testing.T) {
		dev := &factory.Gen2Device{Device: nil}
		err := UpdateFirmware(ctx, dev)
		if err != types.ErrNilDevice {
			t.Errorf("UpdateFirmware() error = %v, want %v", err, types.ErrNilDevice)
		}
	})

	t.Run("unsupported device", func(t *testing.T) {
		dev := &unsupportedDevice{}
		err := UpdateFirmware(ctx, dev)
		if err != types.ErrUnsupportedDevice {
			t.Errorf("UpdateFirmware() error = %v, want %v", err, types.ErrUnsupportedDevice)
		}
	})
}

// TestBatchUpdateFirmware tests batch firmware updates.
func TestBatchUpdateFirmware(t *testing.T) {
	ctx := context.Background()

	t.Run("multiple devices", func(t *testing.T) {
		dev1, server1 := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		defer server1.Close()

		dev2 := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			resp := `{"jsonrpc":"2.0","id":1,"result":{}}`
			return json.RawMessage(resp), nil
		})

		results := BatchUpdateFirmware(ctx, []factory.Device{dev1, dev2})
		if len(results) != 2 {
			t.Errorf("BatchUpdateFirmware() returned %d results, want 2", len(results))
		}
		if !results.AllSuccessful() {
			t.Errorf("BatchUpdateFirmware() had failures")
		}
	})

	t.Run("with failure", func(t *testing.T) {
		dev := &factory.Gen1Device{Device: nil}
		results := BatchUpdateFirmware(ctx, []factory.Device{dev})
		if results.AllSuccessful() {
			t.Errorf("BatchUpdateFirmware() should fail for nil device")
		}
	})
}

// TestUpdateDevicesWithAvailableUpdates tests selective firmware updates.
func TestUpdateDevicesWithAvailableUpdates(t *testing.T) {
	ctx := context.Background()

	t.Run("device with update", func(t *testing.T) {
		// Create device that reports an update available
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/shelly":
				_, _ = w.Write([]byte(`{"type":"SHSW-1","mac":"AA:BB:CC:DD:EE:FF","fw":"v1.9.4"}`))
			case "/ota/check":
				_, _ = w.Write([]byte(`{"status":"ok","has_update":true,"new_version":"v1.10.0"}`))
			case "/ota":
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			}
		})
		defer server.Close()

		results := UpdateDevicesWithAvailableUpdates(ctx, []factory.Device{dev})
		if len(results) != 1 {
			t.Errorf("UpdateDevicesWithAvailableUpdates() returned %d results, want 1", len(results))
		}
	})

	t.Run("no updates available", func(t *testing.T) {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/shelly":
				_, _ = w.Write([]byte(`{"type":"SHSW-1","mac":"AA:BB:CC:DD:EE:FF","fw":"v1.10.0"}`))
			case "/ota/check":
				_, _ = w.Write([]byte(`{"status":"ok","has_update":false}`))
			}
		})
		defer server.Close()

		results := UpdateDevicesWithAvailableUpdates(ctx, []factory.Device{dev})
		if len(results) != 0 {
			t.Errorf("UpdateDevicesWithAvailableUpdates() returned %d results, want 0", len(results))
		}
	})

	t.Run("empty device list", func(t *testing.T) {
		results := UpdateDevicesWithAvailableUpdates(ctx, []factory.Device{})
		if len(results) != 0 {
			t.Errorf("UpdateDevicesWithAvailableUpdates() returned %d results, want 0", len(results))
		}
	})
}
