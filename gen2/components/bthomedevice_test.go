package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
)

func TestNewBTHomeDevice(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	device := NewBTHomeDevice(client, 200)

	if device == nil {
		t.Fatal("NewBTHomeDevice returned nil")
	}

	if device.Type() != "bthomedevice" {
		t.Errorf("Type() = %q, want %q", device.Type(), "bthomedevice")
	}

	if device.Key() != "bthomedevice" {
		t.Errorf("Key() = %q, want %q", device.Key(), "bthomedevice")
	}

	if device.Client() != client {
		t.Error("Client() did not return the expected client")
	}

	if device.ID() != 200 {
		t.Errorf("ID() = %d, want 200", device.ID())
	}
}

func TestBTHomeDevice_GetConfig(t *testing.T) {
	tests := []struct {
		wantName *string
		wantKey  *string
		name     string
		result   string
		wantAddr string
		id       int
	}{
		{
			name:     "basic config",
			id:       200,
			result:   `{"id": 200, "addr": "3c:2e:f5:71:d5:2a", "name": null, "key": null, "meta": null}`,
			wantAddr: "3c:2e:f5:71:d5:2a",
		},
		{
			name:     "config with name",
			id:       200,
			result:   `{"id": 200, "addr": "3c:2e:f5:71:d5:2a", "name": "Bathroom temperature", "key": null, "meta": null}`,
			wantAddr: "3c:2e:f5:71:d5:2a",
			wantName: ptr("Bathroom temperature"),
		},
		{
			name:     "config with encryption key",
			id:       201,
			result:   `{"id": 201, "addr": "11:22:33:44:55:66", "name": "Encrypted Sensor", "key": "aabbccdd", "meta": null}`,
			wantAddr: "11:22:33:44:55:66",
			wantName: ptr("Encrypted Sensor"),
			wantKey:  ptr("aabbccdd"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "BTHomeDevice.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			device := NewBTHomeDevice(client, tt.id)

			config, err := device.GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if config == nil {
				t.Fatal("GetConfig() returned nil config")
			}

			if config.ID != tt.id {
				t.Errorf("config.ID = %d, want %d", config.ID, tt.id)
			}

			if config.Addr != tt.wantAddr {
				t.Errorf("config.Addr = %q, want %q", config.Addr, tt.wantAddr)
			}

			if tt.wantName != nil {
				if config.Name == nil || *config.Name != *tt.wantName {
					t.Errorf("config.Name = %v, want %v", config.Name, *tt.wantName)
				}
			}

			if tt.wantKey != nil {
				if config.Key == nil || *config.Key != *tt.wantKey {
					t.Errorf("config.Key = %v, want %v", config.Key, *tt.wantKey)
				}
			}
		})
	}
}

func TestBTHomeDevice_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	device := NewBTHomeDevice(client, 200)
	testComponentError(t, "GetConfig", func() error {
		_, err := device.GetConfig(context.Background())
		return err
	})
}

func TestBTHomeDevice_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	device := NewBTHomeDevice(client, 200)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := device.GetConfig(context.Background())
		return err
	})
}

func TestBTHomeDevice_SetConfig(t *testing.T) {
	tests := []struct {
		config *BTHomeDeviceSetConfigRequest
		name   string
		id     int
	}{
		{
			name: "set name",
			id:   200,
			config: &BTHomeDeviceSetConfigRequest{
				Name: ptr("Kitchen Sensor"),
			},
		},
		{
			name: "set encryption key",
			id:   200,
			config: &BTHomeDeviceSetConfigRequest{
				Key: ptr("aabbccdd11223344"),
			},
		},
		{
			name: "set multiple fields",
			id:   201,
			config: &BTHomeDeviceSetConfigRequest{
				Name: ptr("Bedroom Sensor"),
				Key:  ptr("11223344aabbccdd"),
			},
		},
		{
			name:   "empty config",
			id:     200,
			config: &BTHomeDeviceSetConfigRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "BTHomeDevice.SetConfig" {
						t.Errorf("method = %q, want %q", method, "BTHomeDevice.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			device := NewBTHomeDevice(client, tt.id)

			err := device.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestBTHomeDevice_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	device := NewBTHomeDevice(client, 200)
	testComponentError(t, "SetConfig", func() error {
		return device.SetConfig(context.Background(), &BTHomeDeviceSetConfigRequest{})
	})
}

func TestBTHomeDevice_GetStatus(t *testing.T) {
	tests := []struct {
		wantRSSI       *int
		wantBattery    *int
		wantPacketID   *int
		name           string
		result         string
		wantErrors     []string
		id             int
		wantLastUpdate float64
	}{
		{
			name:           "full status",
			id:             200,
			result:         `{"id": 200, "rssi": -55, "battery": 100, "packet_id": 1, "last_updated_ts": 1706593991.91}`,
			wantRSSI:       ptr(-55),
			wantBattery:    ptr(100),
			wantPacketID:   ptr(1),
			wantLastUpdate: 1706593991.91,
		},
		{
			name:           "status without battery",
			id:             200,
			result:         `{"id": 200, "rssi": -70, "packet_id": 5, "last_updated_ts": 1706593991.91}`,
			wantRSSI:       ptr(-70),
			wantPacketID:   ptr(5),
			wantLastUpdate: 1706593991.91,
		},
		{
			name:           "status with errors",
			id:             200,
			result:         `{"id": 200, "last_updated_ts": 0, "errors": ["key_missing_or_bad", "decrypt_failed"]}`,
			wantLastUpdate: 0,
			wantErrors:     []string{"key_missing_or_bad", "decrypt_failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "BTHomeDevice.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			device := NewBTHomeDevice(client, tt.id)

			status, err := device.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if status.ID != tt.id {
				t.Errorf("status.ID = %d, want %d", status.ID, tt.id)
			}

			if tt.wantRSSI != nil {
				if status.RSSI == nil || *status.RSSI != *tt.wantRSSI {
					t.Errorf("status.RSSI = %v, want %v", status.RSSI, *tt.wantRSSI)
				}
			}

			if tt.wantBattery != nil {
				if status.Battery == nil || *status.Battery != *tt.wantBattery {
					t.Errorf("status.Battery = %v, want %v", status.Battery, *tt.wantBattery)
				}
			}

			if tt.wantPacketID != nil {
				if status.PacketID == nil || *status.PacketID != *tt.wantPacketID {
					t.Errorf("status.PacketID = %v, want %v", status.PacketID, *tt.wantPacketID)
				}
			}

			if status.LastUpdateTS != tt.wantLastUpdate {
				t.Errorf("status.LastUpdateTS = %v, want %v", status.LastUpdateTS, tt.wantLastUpdate)
			}

			if len(tt.wantErrors) > 0 {
				if len(status.Errors) != len(tt.wantErrors) {
					t.Errorf("status.Errors = %v, want %v", status.Errors, tt.wantErrors)
				}
				for i, err := range tt.wantErrors {
					if i < len(status.Errors) && status.Errors[i] != err {
						t.Errorf("status.Errors[%d] = %q, want %q", i, status.Errors[i], err)
					}
				}
			}
		})
	}
}

func TestBTHomeDevice_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	device := NewBTHomeDevice(client, 200)
	testComponentError(t, "GetStatus", func() error {
		_, err := device.GetStatus(context.Background())
		return err
	})
}

func TestBTHomeDevice_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	device := NewBTHomeDevice(client, 200)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := device.GetStatus(context.Background())
		return err
	})
}

func TestBTHomeDevice_GetKnownObjects(t *testing.T) {
	tests := []struct {
		name       string
		result     string
		wantObjIDs []int
		id         int
		wantCount  int
	}{
		{
			name:       "single object",
			id:         200,
			result:     `{"id": 200, "objects": [{"obj_id": 5, "idx": 3, "component": "bthomesensor:200"}]}`,
			wantCount:  1,
			wantObjIDs: []int{5},
		},
		{
			name:       "multiple objects",
			id:         200,
			result:     `{"id": 200, "objects": [{"obj_id": 2, "idx": 0, "component": "bthomesensor:201"}, {"obj_id": 3, "idx": 0, "component": null}]}`,
			wantCount:  2,
			wantObjIDs: []int{2, 3},
		},
		{
			name:       "no objects",
			id:         200,
			result:     `{"id": 200, "objects": []}`,
			wantCount:  0,
			wantObjIDs: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method != "BTHomeDevice.GetKnownObjects" {
						t.Errorf("method = %q, want %q", method, "BTHomeDevice.GetKnownObjects")
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			device := NewBTHomeDevice(client, tt.id)

			resp, err := device.GetKnownObjects(context.Background())
			if err != nil {
				t.Fatalf("GetKnownObjects() error = %v", err)
			}

			if resp.ID != tt.id {
				t.Errorf("ID = %d, want %d", resp.ID, tt.id)
			}

			if len(resp.Objects) != tt.wantCount {
				t.Errorf("len(Objects) = %d, want %d", len(resp.Objects), tt.wantCount)
			}

			for i, wantID := range tt.wantObjIDs {
				if i < len(resp.Objects) && resp.Objects[i].ObjID != wantID {
					t.Errorf("Objects[%d].ObjID = %d, want %d", i, resp.Objects[i].ObjID, wantID)
				}
			}
		})
	}
}

func TestBTHomeDevice_GetKnownObjects_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	device := NewBTHomeDevice(client, 200)
	testComponentError(t, "GetKnownObjects", func() error {
		_, err := device.GetKnownObjects(context.Background())
		return err
	})
}

func TestBTHomeDevice_GetKnownObjects_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	device := NewBTHomeDevice(client, 200)
	testComponentInvalidJSON(t, "GetKnownObjects", func() error {
		_, err := device.GetKnownObjects(context.Background())
		return err
	})
}

func TestBTHomeDeviceConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config BTHomeDeviceConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: BTHomeDeviceConfig{
				ID:   200,
				Addr: "3c:2e:f5:71:d5:2a",
				Name: ptr("Test Device"),
				Key:  ptr("aabbccdd"),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["id"].(float64) != 200 {
					t.Errorf("id = %v, want 200", data["id"])
				}
				if data["addr"].(string) != "3c:2e:f5:71:d5:2a" {
					t.Errorf("addr = %v, want 3c:2e:f5:71:d5:2a", data["addr"])
				}
				if data["name"].(string) != "Test Device" {
					t.Errorf("name = %v, want Test Device", data["name"])
				}
				if data["key"].(string) != "aabbccdd" {
					t.Errorf("key = %v, want aabbccdd", data["key"])
				}
			},
		},
		{
			name: "minimal config",
			config: BTHomeDeviceConfig{
				ID:   201,
				Addr: "11:22:33:44:55:66",
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["name"]; ok {
					t.Error("name should not be present")
				}
				if _, ok := data["key"]; ok {
					t.Error("key should not be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			tt.check(t, parsed)
		})
	}
}

func TestBTHomeDeviceStatus_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		wantID     int
		wantRSSI   bool
		wantErrors int
	}{
		{
			name:     "full status",
			json:     `{"id":200,"rssi":-55,"battery":100,"packet_id":1,"last_updated_ts":1706593991.91}`,
			wantID:   200,
			wantRSSI: true,
		},
		{
			name:       "status with errors",
			json:       `{"id":200,"last_updated_ts":0,"errors":["decrypt_failed"]}`,
			wantID:     200,
			wantErrors: 1,
		},
		{
			name:   "with unknown fields",
			json:   `{"id":200,"last_updated_ts":0,"future_field":"value"}`,
			wantID: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status BTHomeDeviceStatus
			if err := json.Unmarshal([]byte(tt.json), &status); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if status.ID != tt.wantID {
				t.Errorf("ID = %d, want %d", status.ID, tt.wantID)
			}

			hasRSSI := status.RSSI != nil
			if hasRSSI != tt.wantRSSI {
				t.Errorf("has RSSI = %v, want %v", hasRSSI, tt.wantRSSI)
			}

			if len(status.Errors) != tt.wantErrors {
				t.Errorf("len(Errors) = %d, want %d", len(status.Errors), tt.wantErrors)
			}
		})
	}
}

func TestBTHomeDevice_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"id": 200, "last_updated_ts": 0}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	device := NewBTHomeDevice(client, 200)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := device.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
