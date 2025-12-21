package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewBTHome(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	bthome := NewBTHome(client)

	if bthome == nil {
		t.Fatal("NewBTHome returned nil")
	}

	if bthome.Type() != "bthome" {
		t.Errorf("Type() = %q, want %q", bthome.Type(), "bthome")
	}

	if bthome.Key() != "bthome" {
		t.Errorf("Key() = %q, want %q", bthome.Key(), "bthome")
	}

	if bthome.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestBTHome_GetStatus(t *testing.T) {
	tests := []struct {
		name              string
		result            string
		wantErrors        []string
		wantDiscoveryTime float64
		wantDuration      int
		wantDiscovery     bool
	}{
		{
			name:   "idle status",
			result: `{}`,
		},
		{
			name:              "discovery in progress",
			result:            `{"discovery": {"started_at": 1706593991.91, "duration": 30}}`,
			wantDiscovery:     true,
			wantDiscoveryTime: 1706593991.91,
			wantDuration:      30,
		},
		{
			name:       "bluetooth disabled error",
			result:     `{"errors": ["bluetooth_disabled"]}`,
			wantErrors: []string{"bluetooth_disabled"},
		},
		{
			name:              "discovery with errors",
			result:            `{"discovery": {"started_at": 1706593991.91, "duration": 60}, "errors": ["bluetooth_disabled"]}`,
			wantDiscovery:     true,
			wantDiscoveryTime: 1706593991.91,
			wantDuration:      60,
			wantErrors:        []string{"bluetooth_disabled"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "BTHome.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			bthome := NewBTHome(client)

			status, err := bthome.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if tt.wantDiscovery {
				if status.Discovery == nil {
					t.Error("expected Discovery to be present")
				} else {
					if status.Discovery.StartedAt != tt.wantDiscoveryTime {
						t.Errorf("Discovery.StartedAt = %v, want %v", status.Discovery.StartedAt, tt.wantDiscoveryTime)
					}
					if status.Discovery.Duration != tt.wantDuration {
						t.Errorf("Discovery.Duration = %v, want %v", status.Discovery.Duration, tt.wantDuration)
					}
				}
			} else {
				if status.Discovery != nil {
					t.Error("expected Discovery to be nil")
				}
			}

			if len(tt.wantErrors) > 0 {
				if len(status.Errors) != len(tt.wantErrors) {
					t.Errorf("Errors = %v, want %v", status.Errors, tt.wantErrors)
				}
				for i, err := range tt.wantErrors {
					if i < len(status.Errors) && status.Errors[i] != err {
						t.Errorf("Errors[%d] = %q, want %q", i, status.Errors[i], err)
					}
				}
			}
		})
	}
}

func TestBTHome_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	bthome := NewBTHome(client)
	testComponentError(t, "GetStatus", func() error {
		_, err := bthome.GetStatus(context.Background())
		return err
	})
}

func TestBTHome_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	bthome := NewBTHome(client)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := bthome.GetStatus(context.Background())
		return err
	})
}

func TestBTHome_AddDevice(t *testing.T) {
	tests := []struct {
		name    string
		config  *BTHomeAddDeviceConfig
		id      *int
		wantKey string
	}{
		{
			name: "add device with auto id",
			config: &BTHomeAddDeviceConfig{
				Addr: "3c:2e:f5:71:d5:2a",
			},
			wantKey: "bthomedevice:200",
		},
		{
			name: "add device with specific id",
			config: &BTHomeAddDeviceConfig{
				Addr: "3c:2e:f5:71:d5:2a",
				Name: ptr("Living Room"),
			},
			id:      ptr(201),
			wantKey: "bthomedevice:201",
		},
		{
			name: "add encrypted device",
			config: &BTHomeAddDeviceConfig{
				Addr: "3c:2e:f5:71:d5:2a",
				Name: ptr("Encrypted Sensor"),
				Key:  ptr("aabbccdd11223344"),
			},
			wantKey: "bthomedevice:200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "BTHome.AddDevice" {
						t.Errorf("method = %q, want %q", method, "BTHome.AddDevice")
					}
					return jsonrpcResponse(`{"key": "` + tt.wantKey + `"}`)
				},
			}
			client := rpc.NewClient(tr)
			bthome := NewBTHome(client)

			resp, err := bthome.AddDevice(context.Background(), tt.config, tt.id)
			if err != nil {
				t.Fatalf("AddDevice() error = %v", err)
			}

			if resp.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", resp.Key, tt.wantKey)
			}
		})
	}
}

func TestBTHome_AddDevice_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	bthome := NewBTHome(client)
	testComponentError(t, "AddDevice", func() error {
		_, err := bthome.AddDevice(context.Background(), &BTHomeAddDeviceConfig{Addr: "test"}, nil)
		return err
	})
}

func TestBTHome_AddDevice_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	bthome := NewBTHome(client)
	testComponentInvalidJSON(t, "AddDevice", func() error {
		_, err := bthome.AddDevice(context.Background(), &BTHomeAddDeviceConfig{Addr: "test"}, nil)
		return err
	})
}

func TestBTHome_DeleteDevice(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "BTHome.DeleteDevice" {
				t.Errorf("method = %q, want %q", method, "BTHome.DeleteDevice")
			}
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	bthome := NewBTHome(client)

	err := bthome.DeleteDevice(context.Background(), 200)
	if err != nil {
		t.Fatalf("DeleteDevice() error = %v", err)
	}
}

func TestBTHome_DeleteDevice_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	bthome := NewBTHome(client)
	testComponentError(t, "DeleteDevice", func() error {
		return bthome.DeleteDevice(context.Background(), 200)
	})
}

func TestBTHome_AddSensor(t *testing.T) {
	tests := []struct {
		name    string
		config  *BTHomeAddSensorConfig
		id      *int
		wantKey string
	}{
		{
			name: "add sensor with auto id",
			config: &BTHomeAddSensorConfig{
				Addr:  "3c:2e:f5:71:d5:2a",
				ObjID: 45,
				Idx:   0,
			},
			wantKey: "bthomesensor:200",
		},
		{
			name: "add sensor with specific id",
			config: &BTHomeAddSensorConfig{
				Addr:  "3c:2e:f5:71:d5:2a",
				ObjID: 2,
				Idx:   0,
				Name:  ptr("Temperature"),
			},
			id:      ptr(201),
			wantKey: "bthomesensor:201",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "BTHome.AddSensor" {
						t.Errorf("method = %q, want %q", method, "BTHome.AddSensor")
					}
					return jsonrpcResponse(`{"key": "` + tt.wantKey + `"}`)
				},
			}
			client := rpc.NewClient(tr)
			bthome := NewBTHome(client)

			resp, err := bthome.AddSensor(context.Background(), tt.config, tt.id)
			if err != nil {
				t.Fatalf("AddSensor() error = %v", err)
			}

			if resp.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", resp.Key, tt.wantKey)
			}
		})
	}
}

func TestBTHome_AddSensor_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	bthome := NewBTHome(client)
	testComponentError(t, "AddSensor", func() error {
		_, err := bthome.AddSensor(context.Background(), &BTHomeAddSensorConfig{Addr: "test", ObjID: 1, Idx: 0}, nil)
		return err
	})
}

func TestBTHome_AddSensor_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	bthome := NewBTHome(client)
	testComponentInvalidJSON(t, "AddSensor", func() error {
		_, err := bthome.AddSensor(context.Background(), &BTHomeAddSensorConfig{Addr: "test", ObjID: 1, Idx: 0}, nil)
		return err
	})
}

func TestBTHome_DeleteSensor(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "BTHome.DeleteSensor" {
				t.Errorf("method = %q, want %q", method, "BTHome.DeleteSensor")
			}
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	bthome := NewBTHome(client)

	err := bthome.DeleteSensor(context.Background(), 200)
	if err != nil {
		t.Fatalf("DeleteSensor() error = %v", err)
	}
}

func TestBTHome_DeleteSensor_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	bthome := NewBTHome(client)
	testComponentError(t, "DeleteSensor", func() error {
		return bthome.DeleteSensor(context.Background(), 200)
	})
}

func TestBTHome_StartDeviceDiscovery(t *testing.T) {
	tests := []struct {
		duration *int
		name     string
	}{
		{
			name:     "default duration",
			duration: nil,
		},
		{
			name:     "custom duration",
			duration: ptr(60),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "BTHome.StartDeviceDiscovery" {
						t.Errorf("method = %q, want %q", method, "BTHome.StartDeviceDiscovery")
					}
					return jsonrpcResponse(`null`)
				},
			}
			client := rpc.NewClient(tr)
			bthome := NewBTHome(client)

			err := bthome.StartDeviceDiscovery(context.Background(), tt.duration)
			if err != nil {
				t.Fatalf("StartDeviceDiscovery() error = %v", err)
			}
		})
	}
}

func TestBTHome_StartDeviceDiscovery_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	bthome := NewBTHome(client)
	testComponentError(t, "StartDeviceDiscovery", func() error {
		return bthome.StartDeviceDiscovery(context.Background(), nil)
	})
}

func TestBTHome_GetObjectInfos(t *testing.T) {
	tests := []struct {
		offset     *int
		name       string
		result     string
		objIDs     []int
		wantCount  int
		wantOffset int
	}{
		{
			name:       "single object",
			objIDs:     []int{2},
			result:     `{"infos": [{"obj_id": 2, "name": "Temperature", "type": "sensor", "unit": "°C"}], "offset": -1}`,
			wantCount:  1,
			wantOffset: -1,
		},
		{
			name:       "multiple objects",
			objIDs:     []int{2, 3},
			result:     `{"infos": [{"obj_id": 2, "name": "Temperature", "type": "sensor", "unit": "°C"}, {"obj_id": 3, "name": "Humidity", "type": "sensor", "unit": "%"}], "offset": -1}`,
			wantCount:  2,
			wantOffset: -1,
		},
		{
			name:       "with pagination",
			objIDs:     []int{1, 2, 3, 4, 5},
			offset:     ptr(2),
			result:     `{"infos": [{"obj_id": 3, "name": "Humidity", "type": "sensor"}], "offset": 3}`,
			wantCount:  1,
			wantOffset: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "BTHome.GetObjectInfos" {
						t.Errorf("method = %q, want %q", method, "BTHome.GetObjectInfos")
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			bthome := NewBTHome(client)

			resp, err := bthome.GetObjectInfos(context.Background(), tt.objIDs, tt.offset)
			if err != nil {
				t.Fatalf("GetObjectInfos() error = %v", err)
			}

			if len(resp.Infos) != tt.wantCount {
				t.Errorf("len(Infos) = %d, want %d", len(resp.Infos), tt.wantCount)
			}

			if resp.Offset != tt.wantOffset {
				t.Errorf("Offset = %d, want %d", resp.Offset, tt.wantOffset)
			}
		})
	}
}

func TestBTHome_GetObjectInfos_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	bthome := NewBTHome(client)
	testComponentError(t, "GetObjectInfos", func() error {
		_, err := bthome.GetObjectInfos(context.Background(), []int{1}, nil)
		return err
	})
}

func TestBTHome_GetObjectInfos_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	bthome := NewBTHome(client)
	testComponentInvalidJSON(t, "GetObjectInfos", func() error {
		_, err := bthome.GetObjectInfos(context.Background(), []int{1}, nil)
		return err
	})
}

func TestBTHomeStatus_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		wantDiscovery bool
		wantErrors    int
	}{
		{
			name:          "empty status",
			json:          `{}`,
			wantDiscovery: false,
			wantErrors:    0,
		},
		{
			name:          "with discovery",
			json:          `{"discovery":{"started_at":1706593991.91,"duration":30}}`,
			wantDiscovery: true,
			wantErrors:    0,
		},
		{
			name:          "with errors",
			json:          `{"errors":["bluetooth_disabled"]}`,
			wantDiscovery: false,
			wantErrors:    1,
		},
		{
			name:          "with unknown fields",
			json:          `{"future_field":"value"}`,
			wantDiscovery: false,
			wantErrors:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status BTHomeStatus
			if err := json.Unmarshal([]byte(tt.json), &status); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			hasDiscovery := status.Discovery != nil
			if hasDiscovery != tt.wantDiscovery {
				t.Errorf("has Discovery = %v, want %v", hasDiscovery, tt.wantDiscovery)
			}

			if len(status.Errors) != tt.wantErrors {
				t.Errorf("len(Errors) = %d, want %d", len(status.Errors), tt.wantErrors)
			}
		})
	}
}

func TestBTHome_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					_ = req.GetMethod()
					select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	bthome := NewBTHome(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := bthome.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
