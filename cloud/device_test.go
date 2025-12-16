package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetAllDevices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/device/all" {
			t.Errorf("Unexpected path: %v", r.URL.Path)
		}

		resp := AllDevicesResponse{
			IsOK: true,
			Data: &AllDevicesData{
				DevicesStatus: map[string]*DeviceStatus{
					"device1": {ID: "device1", Online: true},
					"device2": {ID: "device2", Online: false},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	devices, err := client.GetAllDevices(ctx)
	if err != nil {
		t.Fatalf("GetAllDevices failed: %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("devices count = %v, want 2", len(devices))
	}

	if devices["device1"] == nil || !devices["device1"].Online {
		t.Error("device1 should be online")
	}
}

func TestGetAllDevicesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := AllDevicesResponse{
			IsOK:   false,
			Errors: []string{"Authentication failed"},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	_, err := client.GetAllDevices(ctx)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestGetDeviceStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/device/status" {
			t.Errorf("Unexpected path: %v", r.URL.Path)
		}

		id := r.URL.Query().Get("id")
		if id != "device123" {
			t.Errorf("id = %v, want device123", id)
		}

		resp := DeviceStatusResponse{
			IsOK: true,
			Data: &DeviceStatusData{
				DeviceStatus: &DeviceStatus{
					ID:     "device123",
					Online: true,
				},
				Online: true,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	status, err := client.GetDeviceStatus(ctx, "device123")
	if err != nil {
		t.Fatalf("GetDeviceStatus failed: %v", err)
	}

	if status.ID != "device123" {
		t.Errorf("ID = %v, want device123", status.ID)
	}
	if !status.Online {
		t.Error("Online = false, want true")
	}
}

func TestGetDeviceStatusNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := DeviceStatusResponse{
			IsOK:   false,
			Errors: []string{"Device not found"},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	_, err := client.GetDeviceStatus(ctx, "nonexistent")
	if err != ErrDeviceNotFound {
		t.Errorf("Expected ErrDeviceNotFound, got %v", err)
	}
}

func TestGetDeviceStatusOffline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := DeviceStatusResponse{
			IsOK:   false,
			Errors: []string{"Device is offline"},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	_, err := client.GetDeviceStatus(ctx, "offline-device")
	if err != ErrDeviceOffline {
		t.Errorf("Expected ErrDeviceOffline, got %v", err)
	}
}

func TestGetAllDevicesInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	_, err := client.GetAllDevices(ctx)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected 'failed to parse' in error, got: %v", err)
	}
}

func TestGetAllDevicesUnknownError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := AllDevicesResponse{
			IsOK:   false,
			Errors: []string{}, // Empty errors array
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	_, err := client.GetAllDevices(ctx)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("Expected 'unknown error' in error, got: %v", err)
	}
}

func TestGetAllDevicesNilData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := AllDevicesResponse{
			IsOK: true,
			Data: nil, // nil data
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	devices, err := client.GetAllDevices(ctx)
	if err != nil {
		t.Fatalf("GetAllDevices failed: %v", err)
	}
	if devices != nil {
		t.Errorf("Expected nil devices, got %v", devices)
	}
}

func TestGetDevicesV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/devices/api/get" {
			t.Errorf("Unexpected path: %v", r.URL.Path)
		}

		var req V2DevicesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if len(req.IDs) != 2 {
			t.Errorf("IDs count = %v, want 2", len(req.IDs))
		}

		resp := V2DevicesResponse{
			Devices: map[string]*V2DeviceData{
				"device1": {ID: "device1", Online: true},
				"device2": {ID: "device2", Online: true},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	resp, err := client.GetDevicesV2(ctx, []string{"device1", "device2"}, true, false)
	if err != nil {
		t.Fatalf("GetDevicesV2 failed: %v", err)
	}

	if len(resp.Devices) != 2 {
		t.Errorf("Devices count = %v, want 2", len(resp.Devices))
	}
}

func TestGetDevicesV2InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	_, err := client.GetDevicesV2(ctx, []string{"device1"}, true, true)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected 'failed to parse' in error, got: %v", err)
	}
}

func TestGetDevicesV2APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := V2DevicesResponse{
			Error: "Invalid device IDs",
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	_, err := client.GetDevicesV2(ctx, []string{"device1"}, true, false)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("Expected 'API error' in error, got: %v", err)
	}
}

func TestGetDeviceStatusInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	_, err := client.GetDeviceStatus(ctx, "device123")
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected 'failed to parse' in error, got: %v", err)
	}
}

func TestGetDeviceStatusNilData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := DeviceStatusResponse{
			IsOK: true,
			Data: nil, // nil data
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	_, err := client.GetDeviceStatus(ctx, "device123")
	if err != ErrDeviceNotFound {
		t.Errorf("Expected ErrDeviceNotFound, got %v", err)
	}
}

func TestSetSwitch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/device/relay/control" {
			t.Errorf("Unexpected path: %v", r.URL.Path)
		}

		id := r.URL.Query().Get("id")
		channel := r.URL.Query().Get("channel")
		turn := r.URL.Query().Get("turn")

		if id != "device123" {
			t.Errorf("id = %v, want device123", id)
		}
		if channel != "0" {
			t.Errorf("channel = %v, want 0", channel)
		}
		if turn != "on" {
			t.Errorf("turn = %v, want on", turn)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	err := client.SetSwitch(ctx, "device123", 0, true)
	if err != nil {
		t.Fatalf("SetSwitch failed: %v", err)
	}
}

func TestToggleSwitch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		turn := r.URL.Query().Get("turn")
		if turn != "toggle" {
			t.Errorf("turn = %v, want toggle", turn)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	err := client.ToggleSwitch(ctx, "device123", 0)
	if err != nil {
		t.Fatalf("ToggleSwitch failed: %v", err)
	}
}

func TestSetSwitchWithTimer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := r.URL.Query().Get("timer")
		if timer != "60" {
			t.Errorf("timer = %v, want 60", timer)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	err := client.SetSwitchWithTimer(ctx, "device123", 0, true, 60)
	if err != nil {
		t.Fatalf("SetSwitchWithTimer failed: %v", err)
	}
}

func TestCoverControl(t *testing.T) {
	tests := []struct {
		name      string
		action    func(c *Client, ctx context.Context) error
		wantParam string
		wantValue string
	}{
		{
			name:      "OpenCover",
			action:    func(c *Client, ctx context.Context) error { return c.OpenCover(ctx, "device123", 0) },
			wantParam: "direction",
			wantValue: "open",
		},
		{
			name:      "CloseCover",
			action:    func(c *Client, ctx context.Context) error { return c.CloseCover(ctx, "device123", 0) },
			wantParam: "direction",
			wantValue: "close",
		},
		{
			name:      "StopCover",
			action:    func(c *Client, ctx context.Context) error { return c.StopCover(ctx, "device123", 0) },
			wantParam: "direction",
			wantValue: "stop",
		},
		{
			name:      "SetCoverPosition",
			action:    func(c *Client, ctx context.Context) error { return c.SetCoverPosition(ctx, "device123", 0, 50) },
			wantParam: "pos",
			wantValue: "50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/device/roller/control" {
					t.Errorf("Unexpected path: %v", r.URL.Path)
				}
				value := r.URL.Query().Get(tt.wantParam)
				if value != tt.wantValue {
					t.Errorf("%v = %v, want %v", tt.wantParam, value, tt.wantValue)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewClient(
				WithAccessToken("test-token"),
				WithBaseURL(server.URL),
			)
			client.httpClient = server.Client()

			ctx := context.Background()
			err := tt.action(client, ctx)
			if err != nil {
				t.Fatalf("%v failed: %v", tt.name, err)
			}
		})
	}
}

func TestLightControl(t *testing.T) {
	tests := []struct {
		action func(c *Client, ctx context.Context) error
		verify func(r *http.Request) error
		name   string
	}{
		{
			name:   "SetLight on",
			action: func(c *Client, ctx context.Context) error { return c.SetLight(ctx, "device123", 0, true) },
			verify: func(r *http.Request) error {
				if r.URL.Query().Get("turn") != "on" {
					return errMismatch("turn", r.URL.Query().Get("turn"), "on")
				}
				return nil
			},
		},
		{
			name:   "ToggleLight",
			action: func(c *Client, ctx context.Context) error { return c.ToggleLight(ctx, "device123", 0) },
			verify: func(r *http.Request) error {
				if r.URL.Query().Get("turn") != "toggle" {
					return errMismatch("turn", r.URL.Query().Get("turn"), "toggle")
				}
				return nil
			},
		},
		{
			name:   "SetLightBrightness",
			action: func(c *Client, ctx context.Context) error { return c.SetLightBrightness(ctx, "device123", 0, 75) },
			verify: func(r *http.Request) error {
				if r.URL.Query().Get("brightness") != "75" {
					return errMismatch("brightness", r.URL.Query().Get("brightness"), "75")
				}
				return nil
			},
		},
		{
			name:   "SetLightRGB",
			action: func(c *Client, ctx context.Context) error { return c.SetLightRGB(ctx, "device123", 0, 255, 128, 64) },
			verify: func(r *http.Request) error {
				if r.URL.Query().Get("red") != "255" {
					return errMismatch("red", r.URL.Query().Get("red"), "255")
				}
				if r.URL.Query().Get("green") != "128" {
					return errMismatch("green", r.URL.Query().Get("green"), "128")
				}
				if r.URL.Query().Get("blue") != "64" {
					return errMismatch("blue", r.URL.Query().Get("blue"), "64")
				}
				return nil
			},
		},
		{
			name: "SetLightRGBW",
			action: func(c *Client, ctx context.Context) error {
				return c.SetLightRGBW(ctx, "device123", 0, 255, 128, 64, 32)
			},
			verify: func(r *http.Request) error {
				if r.URL.Query().Get("white") != "32" {
					return errMismatch("white", r.URL.Query().Get("white"), "32")
				}
				return nil
			},
		},
		{
			name:   "SetLightColorTemp",
			action: func(c *Client, ctx context.Context) error { return c.SetLightColorTemp(ctx, "device123", 0, 4000) },
			verify: func(r *http.Request) error {
				if r.URL.Query().Get("color_temp") != "4000" {
					return errMismatch("color_temp", r.URL.Query().Get("color_temp"), "4000")
				}
				return nil
			},
		},
		{
			name:   "SetLightEffect",
			action: func(c *Client, ctx context.Context) error { return c.SetLightEffect(ctx, "device123", 0, 2) },
			verify: func(r *http.Request) error {
				if r.URL.Query().Get("effect") != "2" {
					return errMismatch("effect", r.URL.Query().Get("effect"), "2")
				}
				return nil
			},
		},
		{
			name:   "SetLightGain",
			action: func(c *Client, ctx context.Context) error { return c.SetLightGain(ctx, "device123", 0, 80) },
			verify: func(r *http.Request) error {
				if r.URL.Query().Get("gain") != "80" {
					return errMismatch("gain", r.URL.Query().Get("gain"), "80")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/device/light/control" {
					t.Errorf("Unexpected path: %v", r.URL.Path)
				}
				if err := tt.verify(r); err != nil {
					t.Error(err)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewClient(
				WithAccessToken("test-token"),
				WithBaseURL(server.URL),
			)
			client.httpClient = server.Client()

			ctx := context.Background()
			err := tt.action(client, ctx)
			if err != nil {
				t.Fatalf("%v failed: %v", tt.name, err)
			}
		})
	}
}

type mismatchError struct {
	param string
	got   string
	want  string
}

func (e *mismatchError) Error() string {
	return e.param + " = " + e.got + ", want " + e.want
}

func errMismatch(param, got, want string) error {
	return &mismatchError{param: param, got: got, want: want}
}

func TestGroupControl(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/devices/api/set/groups" {
			t.Errorf("Unexpected path: %v", r.URL.Path)
		}

		var req GroupControlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		resp := ControlResponse{IsOK: true}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	req := &GroupControlRequest{
		Switches: []GroupSwitch{
			{IDs: []string{"device1_0", "device2_0"}, Turn: "on"},
		},
	}
	err := client.GroupControl(ctx, req)
	if err != nil {
		t.Fatalf("GroupControl failed: %v", err)
	}
}

func TestGroupControlError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ControlResponse{
			IsOK:   false,
			Errors: []string{"Some devices offline"},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	err := client.SetSwitchGroup(ctx, []string{"device1_0"}, true)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestSetSwitchGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GroupControlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		if len(req.Switches) != 1 {
			t.Errorf("Switches count = %v, want 1", len(req.Switches))
		}
		if req.Switches[0].Turn != "on" {
			t.Errorf("Turn = %v, want on", req.Switches[0].Turn)
		}

		resp := ControlResponse{IsOK: true}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	err := client.SetSwitchGroup(ctx, []string{"device1_0", "device2_0"}, true)
	if err != nil {
		t.Fatalf("SetSwitchGroup failed: %v", err)
	}
}

func TestToggleSwitchGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GroupControlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		if req.Switches[0].Turn != "toggle" {
			t.Errorf("Turn = %v, want toggle", req.Switches[0].Turn)
		}

		resp := ControlResponse{IsOK: true}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithAccessToken("test-token"),
		WithBaseURL(server.URL),
	)
	client.httpClient = server.Client()

	ctx := context.Background()
	err := client.ToggleSwitchGroup(ctx, []string{"device1_0"})
	if err != nil {
		t.Fatalf("ToggleSwitchGroup failed: %v", err)
	}
}

func TestCoverGroupOperations(t *testing.T) {
	tests := []struct {
		name      string
		action    func(c *Client, ctx context.Context) error
		wantField string
	}{
		{
			name: "SetCoverGroupPosition",
			action: func(c *Client, ctx context.Context) error {
				return c.SetCoverGroupPosition(ctx, []string{"device1_0"}, 50)
			},
			wantField: "position",
		},
		{
			name:      "OpenCoverGroup",
			action:    func(c *Client, ctx context.Context) error { return c.OpenCoverGroup(ctx, []string{"device1_0"}) },
			wantField: "direction",
		},
		{
			name:      "CloseCoverGroup",
			action:    func(c *Client, ctx context.Context) error { return c.CloseCoverGroup(ctx, []string{"device1_0"}) },
			wantField: "direction",
		},
		{
			name:      "StopCoverGroup",
			action:    func(c *Client, ctx context.Context) error { return c.StopCoverGroup(ctx, []string{"device1_0"}) },
			wantField: "direction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req GroupControlRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("decode failed: %v", err)
				}

				if len(req.Covers) != 1 {
					t.Errorf("Covers count = %v, want 1", len(req.Covers))
				}

				resp := ControlResponse{IsOK: true}
				if err := json.NewEncoder(w).Encode(resp); err != nil {
					t.Fatalf("encode failed: %v", err)
				}
			}))
			defer server.Close()

			client := NewClient(
				WithAccessToken("test-token"),
				WithBaseURL(server.URL),
			)
			client.httpClient = server.Client()

			ctx := context.Background()
			err := tt.action(client, ctx)
			if err != nil {
				t.Fatalf("%v failed: %v", tt.name, err)
			}
		})
	}
}

func TestLightGroupOperations(t *testing.T) {
	tests := []struct {
		action func(c *Client, ctx context.Context) error
		name   string
	}{
		{
			name: "SetLightGroupBrightness",
			action: func(c *Client, ctx context.Context) error {
				return c.SetLightGroupBrightness(ctx, []string{"device1_0"}, 75)
			},
		},
		{
			name:   "SetLightGroup on",
			action: func(c *Client, ctx context.Context) error { return c.SetLightGroup(ctx, []string{"device1_0"}, true) },
		},
		{
			name:   "ToggleLightGroup",
			action: func(c *Client, ctx context.Context) error { return c.ToggleLightGroup(ctx, []string{"device1_0"}) },
		},
		{
			name: "SetLightGroupRGB",
			action: func(c *Client, ctx context.Context) error {
				return c.SetLightGroupRGB(ctx, []string{"device1_0"}, 255, 128, 64)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req GroupControlRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("decode failed: %v", err)
				}

				if len(req.Lights) != 1 {
					t.Errorf("Lights count = %v, want 1", len(req.Lights))
				}

				resp := ControlResponse{IsOK: true}
				if err := json.NewEncoder(w).Encode(resp); err != nil {
					t.Fatalf("encode failed: %v", err)
				}
			}))
			defer server.Close()

			client := NewClient(
				WithAccessToken("test-token"),
				WithBaseURL(server.URL),
			)
			client.httpClient = server.Client()

			ctx := context.Background()
			err := tt.action(client, ctx)
			if err != nil {
				t.Fatalf("%v failed: %v", tt.name, err)
			}
		})
	}
}

func TestParseDeviceError(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name   string
		errors []string
		want   error
	}{
		{
			name:   "empty errors",
			errors: []string{},
			want:   nil, // Will check for "unknown error" message
		},
		{
			name:   "device not found",
			errors: []string{"device not found"},
			want:   ErrDeviceNotFound,
		},
		{
			name:   "device offline",
			errors: []string{"device is offline"},
			want:   ErrDeviceOffline,
		},
		{
			name:   "generic error",
			errors: []string{"some other error"},
			want:   nil, // Will check for "API error" message
		},
		{
			name:   "multiple errors with not found",
			errors: []string{"error1", "not found"},
			want:   ErrDeviceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.parseDeviceError(tt.errors)
			if err == nil {
				t.Fatal("parseDeviceError() should always return an error")
			}

			if tt.want != nil {
				if !errors.Is(err, tt.want) {
					t.Errorf("parseDeviceError() = %v, want %v", err, tt.want)
				}
			} else if tt.name == "empty errors" {
				if !strings.Contains(err.Error(), "unknown error") {
					t.Errorf("parseDeviceError() = %v, want to contain 'unknown error'", err)
				}
			} else {
				if !strings.Contains(err.Error(), "API error") {
					t.Errorf("parseDeviceError() = %v, want to contain 'API error'", err)
				}
			}
		})
	}
}

func TestGroupControlInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewClient(WithAccessToken("test-token"), WithBaseURL(server.URL))
	client.httpClient = server.Client()

	err := client.GroupControl(context.Background(), &GroupControlRequest{})
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestGroupControlUnknownError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ControlResponse{IsOK: false, Errors: []string{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithAccessToken("test-token"), WithBaseURL(server.URL))
	client.httpClient = server.Client()

	err := client.GroupControl(context.Background(), &GroupControlRequest{})
	if err == nil || !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("Expected 'unknown error', got %v", err)
	}
}

func TestGroupControlAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ControlResponse{IsOK: false, Errors: []string{"device offline"}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithAccessToken("test-token"), WithBaseURL(server.URL))
	client.httpClient = server.Client()

	err := client.GroupControl(context.Background(), &GroupControlRequest{})
	if err == nil || !strings.Contains(err.Error(), "device offline") {
		t.Errorf("Expected error containing 'device offline', got %v", err)
	}
}
