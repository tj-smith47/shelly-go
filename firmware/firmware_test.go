package firmware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/rpc"
)

// mockTransport implements transport.Transport for testing.
type mockTransport struct {
	callFunc func(ctx context.Context, method string, params any) (json.RawMessage, error)
}

func (m *mockTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if m.callFunc != nil {
		return m.callFunc(ctx, method, params)
	}
	return nil, errors.New("no mock response configured")
}

func (m *mockTransport) Close() error {
	return nil
}

// jsonrpcResponse wraps a result in a JSON-RPC response envelope.
func jsonrpcResponse(result any) (json.RawMessage, error) {
	resultData, _ := json.Marshal(result)
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  json.RawMessage(resultData),
	}
	return json.Marshal(response)
}

// mockDevice implements Device interface for testing.
type mockDevice struct {
	client  *rpc.Client
	address string
}

func (m *mockDevice) Address() string {
	return m.address
}

func (m *mockDevice) Client() *rpc.Client {
	return m.client
}

func TestNew(t *testing.T) {
	mt := &mockTransport{}
	client := rpc.NewClient(mt)
	mgr := New(client)

	if mgr == nil {
		t.Fatal("New returned nil")
	}
	if mgr.client != client {
		t.Error("client not set correctly")
	}
}

func TestManager_CheckForUpdate(t *testing.T) {
	tests := []struct {
		response    any
		deviceInfo  any
		name        string
		wantVersion string
		wantErr     bool
		wantHasUpd  bool
		wantBeta    bool
	}{
		{
			name: "stable update available",
			response: map[string]any{
				"stable": map[string]any{
					"version": "1.2.0",
				},
			},
			deviceInfo: map[string]any{
				"ver": "1.1.0",
			},
			wantHasUpd:  true,
			wantVersion: "1.2.0",
		},
		{
			name: "beta available",
			response: map[string]any{
				"stable": map[string]any{
					"version": "1.1.0",
				},
				"beta": map[string]any{
					"version": "1.3.0-beta",
				},
			},
			deviceInfo: map[string]any{
				"ver": "1.1.0",
			},
			wantHasUpd:  false, // same version as stable
			wantBeta:    true,
			wantVersion: "1.1.0",
		},
		{
			name: "no update available",
			response: map[string]any{
				"stable": map[string]any{
					"version": "1.1.0",
				},
			},
			deviceInfo: map[string]any{
				"ver": "1.1.0",
			},
			wantHasUpd:  false,
			wantVersion: "1.1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					switch method {
					case "Shelly.CheckForUpdate":
						return jsonrpcResponse(tt.response)
					case "Shelly.GetDeviceInfo":
						return jsonrpcResponse(tt.deviceInfo)
					}
					return nil, errors.New("unknown method")
				},
			}
			client := rpc.NewClient(mt)
			mgr := New(client)

			info, err := mgr.CheckForUpdate(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckForUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if info.HasUpdate() != tt.wantHasUpd {
				t.Errorf("HasUpdate() = %v, want %v", info.HasUpdate(), tt.wantHasUpd)
			}
			if info.HasBeta() != tt.wantBeta {
				t.Errorf("HasBeta() = %v, want %v", info.HasBeta(), tt.wantBeta)
			}
			if info.Available != tt.wantVersion {
				t.Errorf("Available = %v, want %v", info.Available, tt.wantVersion)
			}
		})
	}
}

func TestManager_CheckForUpdate_Error(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errors.New("connection failed")
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	_, err := mgr.CheckForUpdate(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestManager_Update(t *testing.T) {
	tests := []struct {
		opts    *UpdateOptions
		name    string
		wantErr bool
	}{
		{
			name:    "update with nil options",
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "update with stable stage",
			opts:    &UpdateOptions{Stage: "stable"},
			wantErr: false,
		},
		{
			name:    "update with beta stage",
			opts:    &UpdateOptions{Stage: "beta"},
			wantErr: false,
		},
		{
			name:    "update with URL",
			opts:    &UpdateOptions{URL: "http://example.com/firmware.bin"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "Shelly.Update" {
						t.Errorf("unexpected method: %s", method)
					}
					return jsonrpcResponse(nil)
				},
			}
			client := rpc.NewClient(mt)
			mgr := New(client)

			err := mgr.Update(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_Update_Error(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errors.New("update failed")
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	err := mgr.Update(context.Background(), nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestManager_GetVersion(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "Shelly.GetDeviceInfo" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(map[string]any{
				"ver":   "1.2.0",
				"app":   "Plus1PM",
				"model": "SNSW-001P16EU",
				"gen":   2,
			})
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	version, err := mgr.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}

	if version.FirmwareVersion != "1.2.0" {
		t.Errorf("FirmwareVersion = %v, want 1.2.0", version.FirmwareVersion)
	}
	if version.App != "Plus1PM" {
		t.Errorf("App = %v, want Plus1PM", version.App)
	}
	if version.Model != "SNSW-001P16EU" {
		t.Errorf("Model = %v, want SNSW-001P16EU", version.Model)
	}
	if version.Generation != 2 {
		t.Errorf("Generation = %v, want 2", version.Generation)
	}
}

func TestManager_GetVersion_Error(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errors.New("device not found")
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	_, err := mgr.GetVersion(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestManager_GetStatus(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "Shelly.GetStatus" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(map[string]any{
				"sys": map[string]any{
					"available_updates": map[string]any{
						"stable": map[string]any{
							"version": "1.3.0",
						},
					},
				},
			})
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	status, err := mgr.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if !status.HasUpdate {
		t.Error("HasUpdate should be true")
	}
	if status.NewVersion != "1.3.0" {
		t.Errorf("NewVersion = %v, want 1.3.0", status.NewVersion)
	}
}

func TestManager_GetStatus_NoUpdate(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(map[string]any{
				"sys": map[string]any{},
			})
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	status, err := mgr.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status.HasUpdate {
		t.Error("HasUpdate should be false")
	}
	if status.Status != "idle" {
		t.Errorf("Status = %v, want idle", status.Status)
	}
}

func TestManager_Rollback(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method != "Shelly.Rollback" {
				t.Errorf("unexpected method: %s", method)
			}
			return jsonrpcResponse(nil)
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	err := mgr.Rollback(context.Background())
	if err != nil {
		t.Errorf("Rollback() error = %v", err)
	}
}

func TestManager_Rollback_Error(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errors.New("rollback not available")
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	err := mgr.Rollback(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestManager_GetRollbackStatus(t *testing.T) {
	tests := []struct {
		name            string
		safeMode        bool
		wantCanRollback bool
	}{
		{
			name:            "safe mode enabled",
			safeMode:        true,
			wantCanRollback: true,
		},
		{
			name:            "normal mode",
			safeMode:        false,
			wantCanRollback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					return jsonrpcResponse(map[string]any{
						"sys": map[string]any{
							"safe_mode": tt.safeMode,
						},
					})
				},
			}
			client := rpc.NewClient(mt)
			mgr := New(client)

			status, err := mgr.GetRollbackStatus(context.Background())
			if err != nil {
				t.Fatalf("GetRollbackStatus() error = %v", err)
			}

			if status.CanRollback != tt.wantCanRollback {
				t.Errorf("CanRollback = %v, want %v", status.CanRollback, tt.wantCanRollback)
			}
		})
	}
}

func TestManager_UpdateStable(t *testing.T) {
	var calledMethod string
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			calledMethod = method
			return jsonrpcResponse(nil)
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	err := mgr.UpdateStable(context.Background())
	if err != nil {
		t.Errorf("UpdateStable() error = %v", err)
	}

	if calledMethod != "Shelly.Update" {
		t.Errorf("method = %v, want Shelly.Update", calledMethod)
	}
}

func TestManager_UpdateBeta(t *testing.T) {
	var calledMethod string
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			calledMethod = method
			return jsonrpcResponse(nil)
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	err := mgr.UpdateBeta(context.Background())
	if err != nil {
		t.Errorf("UpdateBeta() error = %v", err)
	}

	if calledMethod != "Shelly.Update" {
		t.Errorf("method = %v, want Shelly.Update", calledMethod)
	}
}

func TestManager_UpdateFromURL(t *testing.T) {
	var calledMethod string
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			calledMethod = method
			return jsonrpcResponse(nil)
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	url := "http://example.com/custom-firmware.bin"
	err := mgr.UpdateFromURL(context.Background(), url)
	if err != nil {
		t.Errorf("UpdateFromURL() error = %v", err)
	}

	if calledMethod != "Shelly.Update" {
		t.Errorf("method = %v, want Shelly.Update", calledMethod)
	}
}

func TestManager_IsUpdateAvailable(t *testing.T) {
	tests := []struct {
		response      any
		deviceInfo    any
		name          string
		wantAvailable bool
	}{
		{
			name: "update available",
			response: map[string]any{
				"stable": map[string]any{
					"version": "1.2.0",
				},
			},
			deviceInfo: map[string]any{
				"ver": "1.1.0",
			},
			wantAvailable: true,
		},
		{
			name: "no update",
			response: map[string]any{
				"stable": map[string]any{
					"version": "1.1.0",
				},
			},
			deviceInfo: map[string]any{
				"ver": "1.1.0",
			},
			wantAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					switch method {
					case "Shelly.CheckForUpdate":
						return jsonrpcResponse(tt.response)
					case "Shelly.GetDeviceInfo":
						return jsonrpcResponse(tt.deviceInfo)
					}
					return nil, errors.New("unknown method")
				},
			}
			client := rpc.NewClient(mt)
			mgr := New(client)

			available, err := mgr.IsUpdateAvailable(context.Background())
			if err != nil {
				t.Fatalf("IsUpdateAvailable() error = %v", err)
			}

			if available != tt.wantAvailable {
				t.Errorf("IsUpdateAvailable() = %v, want %v", available, tt.wantAvailable)
			}
		})
	}
}

func TestManager_IsBetaAvailable(t *testing.T) {
	tests := []struct {
		response      any
		deviceInfo    any
		name          string
		wantAvailable bool
	}{
		{
			name: "beta available",
			response: map[string]any{
				"beta": map[string]any{
					"version": "1.3.0-beta",
				},
			},
			deviceInfo: map[string]any{
				"ver": "1.1.0",
			},
			wantAvailable: true,
		},
		{
			name: "no beta",
			response: map[string]any{
				"stable": map[string]any{
					"version": "1.2.0",
				},
			},
			deviceInfo: map[string]any{
				"ver": "1.1.0",
			},
			wantAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					switch method {
					case "Shelly.CheckForUpdate":
						return jsonrpcResponse(tt.response)
					case "Shelly.GetDeviceInfo":
						return jsonrpcResponse(tt.deviceInfo)
					}
					return nil, errors.New("unknown method")
				},
			}
			client := rpc.NewClient(mt)
			mgr := New(client)

			available, err := mgr.IsBetaAvailable(context.Background())
			if err != nil {
				t.Fatalf("IsBetaAvailable() error = %v", err)
			}

			if available != tt.wantAvailable {
				t.Errorf("IsBetaAvailable() = %v, want %v", available, tt.wantAvailable)
			}
		})
	}
}

func TestUpdateInfo_HasUpdate(t *testing.T) {
	tests := []struct {
		name string
		info UpdateInfo
		want bool
	}{
		{
			name: "flag set",
			info: UpdateInfo{HasUpdateFlag: true},
			want: true,
		},
		{
			name: "different versions",
			info: UpdateInfo{Current: "1.0.0", Available: "1.1.0"},
			want: true,
		},
		{
			name: "same versions",
			info: UpdateInfo{Current: "1.0.0", Available: "1.0.0"},
			want: false,
		},
		{
			name: "empty available",
			info: UpdateInfo{Current: "1.0.0", Available: ""},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.HasUpdate(); got != tt.want {
				t.Errorf("HasUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateInfo_HasBeta(t *testing.T) {
	tests := []struct {
		name string
		info UpdateInfo
		want bool
	}{
		{
			name: "beta available",
			info: UpdateInfo{Current: "1.0.0", Beta: "1.1.0-beta"},
			want: true,
		},
		{
			name: "same as current",
			info: UpdateInfo{Current: "1.0.0", Beta: "1.0.0"},
			want: false,
		},
		{
			name: "empty beta",
			info: UpdateInfo{Current: "1.0.0", Beta: ""},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.HasBeta(); got != tt.want {
				t.Errorf("HasBeta() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBatchCheckUpdates(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.CheckForUpdate":
				return jsonrpcResponse(map[string]any{
					"stable": map[string]any{
						"version": "1.2.0",
					},
				})
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(map[string]any{
					"ver": "1.1.0",
				})
			}
			return nil, errors.New("unknown method")
		},
	}
	client := rpc.NewClient(mt)

	devices := []Device{
		&mockDevice{address: "192.168.1.100", client: client},
		&mockDevice{address: "192.168.1.101", client: client},
	}

	results := BatchCheckUpdates(context.Background(), devices)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for i, r := range results {
		if r.Error != nil {
			t.Errorf("result[%d] error = %v", i, r.Error)
		}
		if r.Info == nil {
			t.Errorf("result[%d] Info is nil", i)
		}
		if !r.Info.HasUpdate() {
			t.Errorf("result[%d] should have update", i)
		}
	}
}

func TestBatchCheckUpdates_NilClient(t *testing.T) {
	devices := []Device{
		&mockDevice{address: "192.168.1.100", client: nil},
	}

	results := BatchCheckUpdates(context.Background(), devices)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Error("expected error for nil client")
	}
}

func TestBatchUpdate(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(nil)
		},
	}
	client := rpc.NewClient(mt)

	devices := []Device{
		&mockDevice{address: "192.168.1.100", client: client},
		&mockDevice{address: "192.168.1.101", client: client},
	}

	results := BatchUpdate(context.Background(), devices, &UpdateOptions{Stage: "stable"})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for i, r := range results {
		if r.Error != nil {
			t.Errorf("result[%d] error = %v", i, r.Error)
		}
		if !r.Success {
			t.Errorf("result[%d] should be successful", i)
		}
	}
}

func TestBatchUpdate_NilClient(t *testing.T) {
	devices := []Device{
		&mockDevice{address: "192.168.1.100", client: nil},
	}

	results := BatchUpdate(context.Background(), devices, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Error("expected error for nil client")
	}
	if results[0].Success {
		t.Error("should not be successful with nil client")
	}
}

func TestUpdateDevicesWithUpdates(t *testing.T) {
	updateCalls := 0
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.CheckForUpdate":
				return jsonrpcResponse(map[string]any{
					"stable": map[string]any{
						"version": "1.2.0",
					},
				})
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(map[string]any{
					"ver": "1.1.0",
				})
			case "Shelly.Update":
				updateCalls++
				return jsonrpcResponse(nil)
			}
			return nil, errors.New("unknown method")
		},
	}
	client := rpc.NewClient(mt)

	devices := []Device{
		&mockDevice{address: "192.168.1.100", client: client},
		&mockDevice{address: "192.168.1.101", client: client},
	}

	results := UpdateDevicesWithUpdates(context.Background(), devices, nil)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if updateCalls != 2 {
		t.Errorf("expected 2 update calls, got %d", updateCalls)
	}
}

func TestUpdateDevicesWithUpdates_NoUpdates(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.CheckForUpdate":
				return jsonrpcResponse(map[string]any{
					"stable": map[string]any{
						"version": "1.1.0",
					},
				})
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(map[string]any{
					"ver": "1.1.0",
				})
			}
			return nil, errors.New("unknown method")
		},
	}
	client := rpc.NewClient(mt)

	devices := []Device{
		&mockDevice{address: "192.168.1.100", client: client},
	}

	results := UpdateDevicesWithUpdates(context.Background(), devices, nil)

	if results != nil {
		t.Errorf("expected nil results when no updates, got %v", results)
	}
}

func TestUpdateDevicesWithUpdates_Empty(t *testing.T) {
	results := UpdateDevicesWithUpdates(context.Background(), []Device{}, nil)
	if results != nil {
		t.Errorf("expected nil results for empty devices, got %v", results)
	}
}

func TestNewDownloader(t *testing.T) {
	d := NewDownloader()
	if d == nil {
		t.Fatal("NewDownloader returned nil")
	}
	if d.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

func TestDownloader_Download(t *testing.T) {
	firmwareData := []byte("fake firmware data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(firmwareData)
	}))
	defer server.Close()

	d := NewDownloader()
	result, err := d.Download(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	if !bytes.Equal(result.Data, firmwareData) {
		t.Errorf("Download() data = %v, want %v", result.Data, firmwareData)
	}
	if result.Size != int64(len(firmwareData)) {
		t.Errorf("Download() size = %v, want %v", result.Size, len(firmwareData))
	}
	if result.ContentType != "application/octet-stream" {
		t.Errorf("Download() contentType = %v, want application/octet-stream", result.ContentType)
	}
}

func TestDownloader_Download_EmptyURL(t *testing.T) {
	d := NewDownloader()
	_, err := d.Download(context.Background(), "")
	if !errors.Is(err, ErrInvalidURL) {
		t.Errorf("Download() error = %v, want ErrInvalidURL", err)
	}
}

func TestDownloader_Download_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	d := NewDownloader()
	_, err := d.Download(context.Background(), server.URL)
	if !errors.Is(err, ErrDownloadFailed) {
		t.Errorf("Download() error = %v, want ErrDownloadFailed", err)
	}
}

func TestDownloader_Download_NilClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	d := &Downloader{HTTPClient: nil}
	result, err := d.Download(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if len(result.Data) == 0 {
		t.Error("Download() data should not be empty")
	}
}

func TestDownloader_DownloadToWriter(t *testing.T) {
	firmwareData := []byte("fake firmware data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(firmwareData)
	}))
	defer server.Close()

	d := NewDownloader()
	var buf bytes.Buffer
	n, err := d.DownloadToWriter(context.Background(), server.URL, &buf)
	if err != nil {
		t.Fatalf("DownloadToWriter() error = %v", err)
	}

	if n != int64(len(firmwareData)) {
		t.Errorf("DownloadToWriter() n = %v, want %v", n, len(firmwareData))
	}
	if !bytes.Equal(buf.Bytes(), firmwareData) {
		t.Errorf("DownloadToWriter() data = %v, want %v", buf.Bytes(), firmwareData)
	}
}

func TestDownloader_DownloadToWriter_EmptyURL(t *testing.T) {
	d := NewDownloader()
	var buf bytes.Buffer
	_, err := d.DownloadToWriter(context.Background(), "", &buf)
	if !errors.Is(err, ErrInvalidURL) {
		t.Errorf("DownloadToWriter() error = %v, want ErrInvalidURL", err)
	}
}

func TestDownloader_DownloadToWriter_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	d := NewDownloader()
	var buf bytes.Buffer
	_, err := d.DownloadToWriter(context.Background(), server.URL, &buf)
	if !errors.Is(err, ErrDownloadFailed) {
		t.Errorf("DownloadToWriter() error = %v, want ErrDownloadFailed", err)
	}
}

func TestDownloader_DownloadToWriter_NilClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	d := &Downloader{HTTPClient: nil}
	var buf bytes.Buffer
	_, err := d.DownloadToWriter(context.Background(), server.URL, &buf)
	if err != nil {
		t.Fatalf("DownloadToWriter() error = %v", err)
	}
}

type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}

func TestDownloader_DownloadToWriter_WriteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	d := NewDownloader()
	_, err := d.DownloadToWriter(context.Background(), server.URL, &errorWriter{})
	if !errors.Is(err, ErrDownloadFailed) {
		t.Errorf("DownloadToWriter() error = %v, want ErrDownloadFailed", err)
	}
}

func TestManager_GetFirmwareURL(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.CheckForUpdate":
				return jsonrpcResponse(map[string]any{
					"stable": map[string]any{
						"version":  "1.2.0",
						"build_id": "20231201-1234",
					},
				})
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(map[string]any{
					"ver":   "1.1.0",
					"model": "SNSW-001P16EU",
				})
			}
			return nil, errors.New("unknown method")
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	url, err := mgr.GetFirmwareURL(context.Background(), "stable")
	if err != nil {
		t.Fatalf("GetFirmwareURL() error = %v", err)
	}

	expected := "http://archive.shelly-tools.de/20231201-1234/SNSW-001P16EU.zip"
	if url != expected {
		t.Errorf("GetFirmwareURL() = %v, want %v", url, expected)
	}
}

func TestManager_GetFirmwareURL_Beta(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.CheckForUpdate":
				return jsonrpcResponse(map[string]any{
					"stable": map[string]any{
						"version":  "1.2.0",
						"build_id": "20231201-1234",
					},
					"beta": map[string]any{
						"version":  "1.3.0-beta",
						"build_id": "20231215-5678",
					},
				})
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(map[string]any{
					"ver":   "1.1.0",
					"model": "SNSW-001P16EU",
				})
			}
			return nil, errors.New("unknown method")
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	url, err := mgr.GetFirmwareURL(context.Background(), "beta")
	if err != nil {
		t.Fatalf("GetFirmwareURL() error = %v", err)
	}

	expected := "http://archive.shelly-tools.de/20231215-5678/SNSW-001P16EU.zip"
	if url != expected {
		t.Errorf("GetFirmwareURL() = %v, want %v", url, expected)
	}
}

func TestManager_GetFirmwareURL_NoUpdate(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.CheckForUpdate":
				return jsonrpcResponse(map[string]any{})
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(map[string]any{
					"ver":   "1.1.0",
					"model": "SNSW-001P16EU",
				})
			}
			return nil, errors.New("unknown method")
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	_, err := mgr.GetFirmwareURL(context.Background(), "stable")
	if !errors.Is(err, ErrNoUpdate) {
		t.Errorf("GetFirmwareURL() error = %v, want ErrNoUpdate", err)
	}
}

func TestManager_GetFirmwareURL_Error(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errors.New("connection failed")
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	_, err := mgr.GetFirmwareURL(context.Background(), "stable")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestNewStagedRollout(t *testing.T) {
	devices := []Device{
		&mockDevice{address: "192.168.1.100"},
		&mockDevice{address: "192.168.1.101"},
	}

	rollout := NewStagedRollout(devices, 50, nil)
	if rollout == nil {
		t.Fatal("NewStagedRollout returned nil")
	}
	if rollout.Percentage != 50 {
		t.Errorf("Percentage = %d, want 50", rollout.Percentage)
	}
	if len(rollout.Devices) != 2 {
		t.Errorf("Devices = %d, want 2", len(rollout.Devices))
	}
}

func TestNewStagedRollout_ClampPercentage(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{-10, 0},
		{0, 0},
		{50, 50},
		{100, 100},
		{150, 100},
	}

	for _, tt := range tests {
		rollout := NewStagedRollout(nil, tt.input, nil)
		if rollout.Percentage != tt.want {
			t.Errorf("NewStagedRollout(%d) Percentage = %d, want %d", tt.input, rollout.Percentage, tt.want)
		}
	}
}

func TestStagedRollout_TargetDeviceCount(t *testing.T) {
	tests := []struct {
		name       string
		devices    int
		percentage int
		want       int
	}{
		{"10 devices 50%", 10, 50, 5},
		{"10 devices 100%", 10, 100, 10},
		{"10 devices 0%", 10, 0, 0},
		{"10 devices 10%", 10, 10, 1},
		{"10 devices 5%", 10, 5, 1}, // At least 1 if percentage > 0
		{"0 devices", 0, 50, 0},
		{"1 device 1%", 1, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devices := make([]Device, tt.devices)
			for i := range devices {
				devices[i] = &mockDevice{address: "test"}
			}
			rollout := NewStagedRollout(devices, tt.percentage, nil)
			got := rollout.TargetDeviceCount()
			if got != tt.want {
				t.Errorf("TargetDeviceCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStagedRollout_SelectDevices(t *testing.T) {
	devices := make([]Device, 10)
	for i := range devices {
		devices[i] = &mockDevice{address: "test"}
	}

	rollout := NewStagedRollout(devices, 50, nil)
	selected := rollout.SelectDevices()

	if len(selected) != 5 {
		t.Errorf("SelectDevices() = %d, want 5", len(selected))
	}
}

func TestStagedRollout_SelectDevices_All(t *testing.T) {
	devices := make([]Device, 10)
	for i := range devices {
		devices[i] = &mockDevice{address: "test"}
	}

	rollout := NewStagedRollout(devices, 100, nil)
	selected := rollout.SelectDevices()

	if len(selected) != 10 {
		t.Errorf("SelectDevices() = %d, want 10", len(selected))
	}
}

func TestStagedRollout_Start(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(nil)
		},
	}
	client := rpc.NewClient(mt)

	devices := []Device{
		&mockDevice{address: "192.168.1.100", client: client},
		&mockDevice{address: "192.168.1.101", client: client},
	}

	rollout := NewStagedRollout(devices, 100, nil)
	rollout.DelayBetweenBatches = 0

	results, err := rollout.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Start() results = %d, want 2", len(results))
	}
}

func TestStagedRollout_Start_WithProgress(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(nil)
		},
	}
	client := rpc.NewClient(mt)

	devices := []Device{
		&mockDevice{address: "192.168.1.100", client: client},
		&mockDevice{address: "192.168.1.101", client: client},
	}

	rollout := NewStagedRollout(devices, 100, nil)
	rollout.DelayBetweenBatches = 0

	progressCalls := 0
	rollout.OnProgress = func(device Device, result *UpdateResult, completed, total int) {
		progressCalls++
	}

	completeCalled := false
	rollout.OnComplete = func(results []UpdateResult) {
		completeCalled = true
	}

	_, err := rollout.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if progressCalls != 2 {
		t.Errorf("OnProgress called %d times, want 2", progressCalls)
	}
	if !completeCalled {
		t.Error("OnComplete not called")
	}
}

func TestStagedRollout_Start_AlreadyInProgress(t *testing.T) {
	rollout := NewStagedRollout(nil, 100, nil)
	rollout.inProgress = true

	_, err := rollout.Start(context.Background())
	if !errors.Is(err, ErrRolloutInProgress) {
		t.Errorf("Start() error = %v, want ErrRolloutInProgress", err)
	}
}

func TestStagedRollout_Start_NoDevices(t *testing.T) {
	rollout := NewStagedRollout([]Device{}, 100, nil)

	results, err := rollout.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if results != nil {
		t.Errorf("Start() results = %v, want nil", results)
	}
}

func TestStagedRollout_Start_ContextCancelled(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(nil)
		},
	}
	client := rpc.NewClient(mt)

	devices := make([]Device, 100)
	for i := range devices {
		devices[i] = &mockDevice{address: "test", client: client}
	}

	rollout := NewStagedRollout(devices, 100, nil)
	rollout.BatchSize = 1
	rollout.DelayBetweenBatches = time.Second

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after first batch
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := rollout.Start(ctx)
	if err != context.Canceled {
		t.Errorf("Start() error = %v, want context.Canceled", err)
	}
}

func TestStagedRollout_Cancel(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(nil)
		},
	}
	client := rpc.NewClient(mt)

	devices := make([]Device, 100)
	for i := range devices {
		devices[i] = &mockDevice{address: "test", client: client}
	}

	rollout := NewStagedRollout(devices, 100, nil)
	rollout.BatchSize = 1
	rollout.DelayBetweenBatches = 100 * time.Millisecond

	// Cancel after first few batches
	go func() {
		time.Sleep(50 * time.Millisecond)
		rollout.Cancel()
	}()

	results, err := rollout.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Should have stopped before all devices
	if len(results) >= len(devices) {
		t.Errorf("Start() should have stopped early, got %d results", len(results))
	}
}

func TestStagedRollout_IsInProgress(t *testing.T) {
	rollout := NewStagedRollout(nil, 100, nil)

	if rollout.IsInProgress() {
		t.Error("IsInProgress() should be false initially")
	}

	rollout.inProgress = true
	if !rollout.IsInProgress() {
		t.Error("IsInProgress() should be true")
	}
}

func TestStagedRollout_SetPercentage(t *testing.T) {
	rollout := NewStagedRollout(nil, 50, nil)

	rollout.SetPercentage(75)
	if rollout.Percentage != 75 {
		t.Errorf("Percentage = %d, want 75", rollout.Percentage)
	}

	rollout.SetPercentage(-10)
	if rollout.Percentage != 0 {
		t.Errorf("Percentage = %d, want 0", rollout.Percentage)
	}

	rollout.SetPercentage(150)
	if rollout.Percentage != 100 {
		t.Errorf("Percentage = %d, want 100", rollout.Percentage)
	}
}

func TestStagedRollout_GetStatus(t *testing.T) {
	devices := make([]Device, 10)
	for i := range devices {
		devices[i] = &mockDevice{address: "test"}
	}

	rollout := NewStagedRollout(devices, 50, nil)
	rollout.Results = []UpdateResult{
		{Success: true},
		{Success: true},
		{Success: false},
	}

	status := rollout.GetStatus()

	if status.TotalDevices != 10 {
		t.Errorf("TotalDevices = %d, want 10", status.TotalDevices)
	}
	if status.TargetDevices != 5 {
		t.Errorf("TargetDevices = %d, want 5", status.TargetDevices)
	}
	if status.CompletedDevices != 3 {
		t.Errorf("CompletedDevices = %d, want 3", status.CompletedDevices)
	}
	if status.SuccessfulUpdates != 2 {
		t.Errorf("SuccessfulUpdates = %d, want 2", status.SuccessfulUpdates)
	}
	if status.FailedUpdates != 1 {
		t.Errorf("FailedUpdates = %d, want 1", status.FailedUpdates)
	}
	if status.Percentage != 50 {
		t.Errorf("Percentage = %d, want 50", status.Percentage)
	}
}

func TestManager_GetRollbackStatus_Error(t *testing.T) {
	t.Run("call error", func(t *testing.T) {
		mt := &mockTransport{
			callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection failed")
			},
		}
		client := rpc.NewClient(mt)
		mgr := New(client)

		_, err := mgr.GetRollbackStatus(context.Background())
		if err == nil {
			t.Error("GetRollbackStatus() expected error")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mt := &mockTransport{
			callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`{invalid json`), nil
			},
		}
		client := rpc.NewClient(mt)
		mgr := New(client)

		_, err := mgr.GetRollbackStatus(context.Background())
		if err == nil {
			t.Error("GetRollbackStatus() expected error for invalid JSON")
		}
	})
}

func TestManager_IsUpdateAvailable_Error(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errors.New("connection failed")
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	_, err := mgr.IsUpdateAvailable(context.Background())
	if err == nil {
		t.Error("IsUpdateAvailable() expected error")
	}
}

func TestManager_IsBetaAvailable_Error(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errors.New("connection failed")
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	_, err := mgr.IsBetaAvailable(context.Background())
	if err == nil {
		t.Error("IsBetaAvailable() expected error")
	}
}

func TestManager_GetVersion_InvalidJSON(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return json.RawMessage(`{invalid`), nil
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	_, err := mgr.GetVersion(context.Background())
	if err == nil {
		t.Error("GetVersion() expected error for invalid JSON")
	}
}

func TestManager_GetStatus_InvalidJSON(t *testing.T) {
	mt := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return json.RawMessage(`{invalid`), nil
		},
	}
	client := rpc.NewClient(mt)
	mgr := New(client)

	_, err := mgr.GetStatus(context.Background())
	if err == nil {
		t.Error("GetStatus() expected error for invalid JSON")
	}
}
