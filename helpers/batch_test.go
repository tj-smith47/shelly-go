package helpers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tj-smith47/shelly-go/factory"
	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
	"github.com/tj-smith47/shelly-go/types"
)

// mockTransport is a simple mock transport for testing.
type mockTransport struct {
	handler func(method string, params any) (json.RawMessage, error)
}

func (m *mockTransport) Call(_ context.Context, req transport.RPCRequest) (json.RawMessage, error) {
	return m.handler(req.GetMethod(), req.GetParams())
}

func (m *mockTransport) Close() error {
	return nil
}

// createMockGen2DeviceWithTransport creates a mock Gen2 device with a mock transport.
func createMockGen2DeviceWithTransport(handler func(method string, params any) (json.RawMessage, error)) *factory.Gen2Device {
	mt := &mockTransport{handler: handler}
	client := rpc.NewClient(mt)
	dev := gen2.NewDevice(client)
	return &factory.Gen2Device{Device: dev}
}

// TestBatchResults_AllSuccessful tests the AllSuccessful method.
func TestBatchResults_AllSuccessful(t *testing.T) {
	tests := []struct {
		name    string
		results BatchResults
		want    bool
	}{
		{
			name:    "empty results",
			results: BatchResults{},
			want:    true,
		},
		{
			name: "all successful",
			results: BatchResults{
				{Success: true},
				{Success: true},
				{Success: true},
			},
			want: true,
		},
		{
			name: "one failure",
			results: BatchResults{
				{Success: true},
				{Success: false},
				{Success: true},
			},
			want: false,
		},
		{
			name: "all failures",
			results: BatchResults{
				{Success: false},
				{Success: false},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.results.AllSuccessful(); got != tt.want {
				t.Errorf("AllSuccessful() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestBatchResults_Failures tests the Failures method.
func TestBatchResults_Failures(t *testing.T) {
	results := BatchResults{
		{Success: true},
		{Success: false, Error: types.ErrTimeout},
		{Success: true},
		{Success: false, Error: types.ErrAuth},
	}

	failures := results.Failures()

	if len(failures) != 2 {
		t.Errorf("Failures() returned %d results, want 2", len(failures))
	}

	for _, f := range failures {
		if f.Success {
			t.Errorf("Failures() returned a successful result")
		}
	}
}

// TestBatchResults_Successes tests the Successes method.
func TestBatchResults_Successes(t *testing.T) {
	results := BatchResults{
		{Success: true},
		{Success: false},
		{Success: true},
		{Success: false},
	}

	successes := results.Successes()

	if len(successes) != 2 {
		t.Errorf("Successes() returned %d results, want 2", len(successes))
	}

	for _, s := range successes {
		if !s.Success {
			t.Errorf("Successes() returned a failed result")
		}
	}
}

// createMockGen1Device creates a mock Gen1 device with a test server.
func createMockGen1Device(_ *testing.T, handler http.HandlerFunc) (*factory.Gen1Device, *httptest.Server) {
	server := httptest.NewServer(handler)
	tr := transport.NewHTTP(server.URL)
	dev := gen1.NewDevice(tr)
	return &factory.Gen1Device{Device: dev}, server
}

// mustWrite is a test helper that writes to an http.ResponseWriter and ignores errors.
func mustWrite(w http.ResponseWriter, data []byte) {
	_, _ = w.Write(data) //nolint:errcheck // Test helper, errors not important
}

// TestBatchSet tests the BatchSet function.
func TestBatchSet(t *testing.T) {
	ctx := context.Background()

	t.Run("gen1 devices on", func(t *testing.T) {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/relay/0" && r.URL.Query().Get("turn") == "on" {
				w.WriteHeader(http.StatusOK)
				mustWrite(w, []byte(`{"ison":true}`))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
		})
		defer server.Close()

		results := BatchSet(ctx, []factory.Device{dev}, true)
		if !results.AllSuccessful() {
			t.Errorf("BatchSet() failed: %v", results[0].Error)
		}
	})

	t.Run("gen1 devices off", func(t *testing.T) {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/relay/0" && r.URL.Query().Get("turn") == "off" {
				w.WriteHeader(http.StatusOK)
				mustWrite(w, []byte(`{"ison":false}`))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
		})
		defer server.Close()

		results := BatchSet(ctx, []factory.Device{dev}, false)
		if !results.AllSuccessful() {
			t.Errorf("BatchSet() failed: %v", results[0].Error)
		}
	})

	t.Run("gen2 devices on", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Switch.Set" {
				// Return valid JSON-RPC response
				resp := `{"jsonrpc":"2.0","id":1,"result":{"was_on":false}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		results := BatchSet(ctx, []factory.Device{dev}, true)
		if !results.AllSuccessful() {
			t.Errorf("BatchSet() failed: %v", results[0].Error)
		}
	})

	t.Run("gen2 devices off", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Switch.Set" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{"was_on":true}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		results := BatchSet(ctx, []factory.Device{dev}, false)
		if !results.AllSuccessful() {
			t.Errorf("BatchSet() failed: %v", results[0].Error)
		}
	})

	t.Run("multiple devices", func(t *testing.T) {
		dev1, server1 := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			mustWrite(w, []byte(`{"ison":true}`))
		})
		defer server1.Close()

		dev2, server2 := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			mustWrite(w, []byte(`{"ison":true}`))
		})
		defer server2.Close()

		results := BatchSet(ctx, []factory.Device{dev1, dev2}, true)
		if len(results) != 2 {
			t.Errorf("BatchSet() returned %d results, want 2", len(results))
		}
		if !results.AllSuccessful() {
			t.Errorf("BatchSet() had failures")
		}
	})

	t.Run("nil gen1 device", func(t *testing.T) {
		dev := &factory.Gen1Device{Device: nil}
		results := BatchSet(ctx, []factory.Device{dev}, true)
		if results.AllSuccessful() {
			t.Errorf("BatchSet() should fail for nil device")
		}
		if results[0].Error != types.ErrNilDevice {
			t.Errorf("BatchSet() error = %v, want %v", results[0].Error, types.ErrNilDevice)
		}
	})

	t.Run("nil gen2 device", func(t *testing.T) {
		dev := &factory.Gen2Device{Device: nil}
		results := BatchSet(ctx, []factory.Device{dev}, true)
		if results.AllSuccessful() {
			t.Errorf("BatchSet() should fail for nil device")
		}
		if results[0].Error != types.ErrNilDevice {
			t.Errorf("BatchSet() error = %v, want %v", results[0].Error, types.ErrNilDevice)
		}
	})

	t.Run("empty device list", func(t *testing.T) {
		results := BatchSet(ctx, []factory.Device{}, true)
		if len(results) != 0 {
			t.Errorf("BatchSet() returned %d results, want 0", len(results))
		}
		if !results.AllSuccessful() {
			t.Errorf("Empty results should be AllSuccessful")
		}
	})
}

// TestBatchToggle tests the BatchToggle function.
func TestBatchToggle(t *testing.T) {
	ctx := context.Background()

	t.Run("gen1 devices", func(t *testing.T) {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/relay/0" && r.URL.Query().Get("turn") == "toggle" {
				mustWrite(w, []byte(`{"ison":true}`))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
		})
		defer server.Close()

		results := BatchToggle(ctx, []factory.Device{dev})
		if !results.AllSuccessful() {
			t.Errorf("BatchToggle() failed: %v", results[0].Error)
		}
	})

	t.Run("gen2 devices", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Switch.Toggle" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{"was_on":false}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		results := BatchToggle(ctx, []factory.Device{dev})
		if !results.AllSuccessful() {
			t.Errorf("BatchToggle() failed: %v", results[0].Error)
		}
	})

	t.Run("nil gen1 device", func(t *testing.T) {
		dev := &factory.Gen1Device{Device: nil}
		results := BatchToggle(ctx, []factory.Device{dev})
		if results.AllSuccessful() {
			t.Errorf("BatchToggle() should fail for nil device")
		}
	})

	t.Run("nil gen2 device", func(t *testing.T) {
		dev := &factory.Gen2Device{Device: nil}
		results := BatchToggle(ctx, []factory.Device{dev})
		if results.AllSuccessful() {
			t.Errorf("BatchToggle() should fail for nil device")
		}
	})
}

// TestAllOff tests the AllOff function.
func TestAllOff(t *testing.T) {
	ctx := context.Background()

	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("turn") == "off" {
			mustWrite(w, []byte(`{"ison":false}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	})
	defer server.Close()

	results := AllOff(ctx, []factory.Device{dev})
	if !results.AllSuccessful() {
		t.Errorf("AllOff() failed: %v", results[0].Error)
	}
}

// TestAllOn tests the AllOn function.
func TestAllOn(t *testing.T) {
	ctx := context.Background()

	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("turn") == "on" {
			mustWrite(w, []byte(`{"ison":true}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	})
	defer server.Close()

	results := AllOn(ctx, []factory.Device{dev})
	if !results.AllSuccessful() {
		t.Errorf("AllOn() failed: %v", results[0].Error)
	}
}

// TestBatchSetBrightness tests the BatchSetBrightness function.
func TestBatchSetBrightness(t *testing.T) {
	ctx := context.Background()

	t.Run("gen1 devices", func(t *testing.T) {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/light/0" && r.URL.Query().Get("brightness") == "75" {
				mustWrite(w, []byte(`{"ison":true,"brightness":75}`))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
		})
		defer server.Close()

		results := BatchSetBrightness(ctx, []factory.Device{dev}, 75)
		if !results.AllSuccessful() {
			t.Errorf("BatchSetBrightness() failed: %v", results[0].Error)
		}
	})

	t.Run("gen2 devices", func(t *testing.T) {
		dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
			if method == "Light.Set" {
				resp := `{"jsonrpc":"2.0","id":1,"result":{}}`
				return json.RawMessage(resp), nil
			}
			return nil, types.ErrRPCMethod
		})

		results := BatchSetBrightness(ctx, []factory.Device{dev}, 75)
		if !results.AllSuccessful() {
			t.Errorf("BatchSetBrightness() failed: %v", results[0].Error)
		}
	})

	t.Run("nil gen1 device", func(t *testing.T) {
		dev := &factory.Gen1Device{Device: nil}
		results := BatchSetBrightness(ctx, []factory.Device{dev}, 75)
		if results.AllSuccessful() {
			t.Errorf("BatchSetBrightness() should fail for nil device")
		}
	})

	t.Run("nil gen2 device", func(t *testing.T) {
		dev := &factory.Gen2Device{Device: nil}
		results := BatchSetBrightness(ctx, []factory.Device{dev}, 75)
		if results.AllSuccessful() {
			t.Errorf("BatchSetBrightness() should fail for nil device")
		}
	})
}

// TestUnsupportedDeviceType tests that unsupported device types return errors.
func TestUnsupportedDeviceType(t *testing.T) {
	ctx := context.Background()

	// Create a custom device type that's not supported
	unsupported := &unsupportedDevice{}

	t.Run("BatchSet unsupported", func(t *testing.T) {
		results := BatchSet(ctx, []factory.Device{unsupported}, true)
		if results.AllSuccessful() {
			t.Errorf("BatchSet() should fail for unsupported device")
		}
		if results[0].Error != types.ErrUnsupportedDevice {
			t.Errorf("BatchSet() error = %v, want %v", results[0].Error, types.ErrUnsupportedDevice)
		}
	})

	t.Run("BatchToggle unsupported", func(t *testing.T) {
		results := BatchToggle(ctx, []factory.Device{unsupported})
		if results.AllSuccessful() {
			t.Errorf("BatchToggle() should fail for unsupported device")
		}
		if results[0].Error != types.ErrUnsupportedDevice {
			t.Errorf("BatchToggle() error = %v, want %v", results[0].Error, types.ErrUnsupportedDevice)
		}
	})

	t.Run("BatchSetBrightness unsupported", func(t *testing.T) {
		results := BatchSetBrightness(ctx, []factory.Device{unsupported}, 75)
		if results.AllSuccessful() {
			t.Errorf("BatchSetBrightness() should fail for unsupported device")
		}
		if results[0].Error != types.ErrUnsupportedDevice {
			t.Errorf("BatchSetBrightness() error = %v, want %v", results[0].Error, types.ErrUnsupportedDevice)
		}
	})
}

// unsupportedDevice is a test device type that's not supported by the batch operations.
type unsupportedDevice struct{}

func (d *unsupportedDevice) Address() string              { return "test" }
func (d *unsupportedDevice) Generation() types.Generation { return types.GenUnknown }

// TestConcurrency tests that batch operations execute concurrently.
func TestConcurrency(t *testing.T) {
	ctx := context.Background()

	// Create multiple mock devices
	var devices []factory.Device
	var servers []*httptest.Server

	for i := 0; i < 5; i++ {
		dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
			mustWrite(w, []byte(`{"ison":true}`))
		})
		devices = append(devices, dev)
		servers = append(servers, server)
	}
	defer func() {
		for _, s := range servers {
			s.Close()
		}
	}()

	// Execute batch operation
	results := BatchSet(ctx, devices, true)

	// Verify all results
	if len(results) != 5 {
		t.Errorf("BatchSet() returned %d results, want 5", len(results))
	}
	if !results.AllSuccessful() {
		t.Errorf("BatchSet() had failures")
	}
}

// TestMixedDevices tests batch operations with mixed Gen1 and Gen2 devices.
func TestMixedDevices(t *testing.T) {
	ctx := context.Background()

	gen1Dev, server1 := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		mustWrite(w, []byte(`{"ison":true}`))
	})
	defer server1.Close()

	gen2Dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
		resp := `{"jsonrpc":"2.0","id":1,"result":{}}`
		return json.RawMessage(resp), nil
	})

	results := BatchSet(ctx, []factory.Device{gen1Dev, gen2Dev}, true)
	if len(results) != 2 {
		t.Errorf("BatchSet() returned %d results, want 2", len(results))
	}
	if !results.AllSuccessful() {
		t.Errorf(
			"BatchSet() failed for mixed devices: gen1 err=%v, gen2 err=%v",
			results[0].Error, results[1].Error,
		)
	}
}

// TestBatchResultDevice tests that the Device field is properly set in results.
func TestBatchResultDevice(t *testing.T) {
	ctx := context.Background()

	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		mustWrite(w, []byte(`{"ison":true}`))
	})
	defer server.Close()

	results := BatchSet(ctx, []factory.Device{dev}, true)
	if len(results) != 1 {
		t.Fatalf("BatchSet() returned %d results, want 1", len(results))
	}
	if results[0].Device != dev {
		t.Errorf("BatchSet() result device doesn't match input device")
	}
}
