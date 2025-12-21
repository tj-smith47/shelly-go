package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewBTHomeSensor(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	sensor := NewBTHomeSensor(client, 200)

	if sensor == nil {
		t.Fatal("NewBTHomeSensor returned nil")
	}

	if sensor.Type() != "bthomesensor" {
		t.Errorf("Type() = %q, want %q", sensor.Type(), "bthomesensor")
	}

	if sensor.Key() != "bthomesensor" {
		t.Errorf("Key() = %q, want %q", sensor.Key(), "bthomesensor")
	}

	if sensor.Client() != client {
		t.Error("Client() did not return the expected client")
	}

	if sensor.ID() != 200 {
		t.Errorf("ID() = %d, want 200", sensor.ID())
	}
}

func TestBTHomeSensor_GetConfig(t *testing.T) {
	tests := []struct {
		wantName  *string
		name      string
		result    string
		wantAddr  string
		id        int
		wantObjID int
		wantIdx   int
	}{
		{
			name:      "basic config",
			id:        200,
			result:    `{"id": 200, "addr": "3c:2e:f5:71:d5:2a", "name": null, "meta": null, "obj_id": 45, "idx": 3}`,
			wantAddr:  "3c:2e:f5:71:d5:2a",
			wantObjID: 45,
			wantIdx:   3,
		},
		{
			name:      "config with name",
			id:        200,
			result:    `{"id": 200, "addr": "3c:2e:f5:71:d5:2a", "name": "Door status", "meta": null, "obj_id": 45, "idx": 3}`,
			wantAddr:  "3c:2e:f5:71:d5:2a",
			wantObjID: 45,
			wantIdx:   3,
			wantName:  ptr("Door status"),
		},
		{
			name:      "temperature sensor",
			id:        201,
			result:    `{"id": 201, "addr": "11:22:33:44:55:66", "name": "Room Temperature", "obj_id": 2, "idx": 0}`,
			wantAddr:  "11:22:33:44:55:66",
			wantObjID: 2,
			wantIdx:   0,
			wantName:  ptr("Room Temperature"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "BTHomeSensor.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			sensor := NewBTHomeSensor(client, tt.id)

			config, err := sensor.GetConfig(context.Background())
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

			if config.ObjID != tt.wantObjID {
				t.Errorf("config.ObjID = %d, want %d", config.ObjID, tt.wantObjID)
			}

			if config.Idx != tt.wantIdx {
				t.Errorf("config.Idx = %d, want %d", config.Idx, tt.wantIdx)
			}

			if tt.wantName != nil {
				if config.Name == nil || *config.Name != *tt.wantName {
					t.Errorf("config.Name = %v, want %v", config.Name, *tt.wantName)
				}
			}
		})
	}
}

func TestBTHomeSensor_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	sensor := NewBTHomeSensor(client, 200)
	testComponentError(t, "GetConfig", func() error {
		_, err := sensor.GetConfig(context.Background())
		return err
	})
}

func TestBTHomeSensor_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	sensor := NewBTHomeSensor(client, 200)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := sensor.GetConfig(context.Background())
		return err
	})
}

func TestBTHomeSensor_SetConfig(t *testing.T) {
	tests := []struct {
		config *BTHomeSensorSetConfigRequest
		name   string
		id     int
	}{
		{
			name: "set name",
			id:   200,
			config: &BTHomeSensorSetConfigRequest{
				Name: ptr("Kitchen Door"),
			},
		},
		{
			name: "set object id",
			id:   200,
			config: &BTHomeSensorSetConfigRequest{
				ObjID: ptr(45),
			},
		},
		{
			name: "set multiple fields",
			id:   201,
			config: &BTHomeSensorSetConfigRequest{
				Name:  ptr("Window Sensor"),
				ObjID: ptr(46),
			},
		},
		{
			name:   "empty config",
			id:     200,
			config: &BTHomeSensorSetConfigRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "BTHomeSensor.SetConfig" {
						t.Errorf("method = %q, want %q", method, "BTHomeSensor.SetConfig")
					}
					return jsonrpcResponse(`{"restart_required": false}`)
				},
			}
			client := rpc.NewClient(tr)
			sensor := NewBTHomeSensor(client, tt.id)

			err := sensor.SetConfig(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestBTHomeSensor_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	sensor := NewBTHomeSensor(client, 200)
	testComponentError(t, "SetConfig", func() error {
		return sensor.SetConfig(context.Background(), &BTHomeSensorSetConfigRequest{})
	})
}

func TestBTHomeSensor_GetStatus(t *testing.T) {
	tests := []struct {
		name           string
		result         string
		wantValueType  string
		id             int
		wantLastUpdate float64
	}{
		{
			name:           "numeric value",
			id:             200,
			result:         `{"id": 200, "value": 23.5, "last_updated_ts": 1706593991.91}`,
			wantValueType:  "float64",
			wantLastUpdate: 1706593991.91,
		},
		{
			name:           "boolean value false",
			id:             200,
			result:         `{"id": 200, "value": false, "last_updated_ts": 1706593991.91}`,
			wantValueType:  "bool",
			wantLastUpdate: 1706593991.91,
		},
		{
			name:           "boolean value true",
			id:             200,
			result:         `{"id": 200, "value": true, "last_updated_ts": 1706593991.91}`,
			wantValueType:  "bool",
			wantLastUpdate: 1706593991.91,
		},
		{
			name:           "string value",
			id:             201,
			result:         `{"id": 201, "value": "open", "last_updated_ts": 1706594000.00}`,
			wantValueType:  "string",
			wantLastUpdate: 1706594000.00,
		},
		{
			name:           "integer value as float",
			id:             200,
			result:         `{"id": 200, "value": 100, "last_updated_ts": 1706593991.91}`,
			wantValueType:  "float64",
			wantLastUpdate: 1706593991.91,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "BTHomeSensor.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			sensor := NewBTHomeSensor(client, tt.id)

			status, err := sensor.GetStatus(context.Background())
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

			// Check value type
			switch tt.wantValueType {
			case "float64":
				if _, ok := status.Value.(float64); !ok {
					t.Errorf("status.Value type = %T, want float64", status.Value)
				}
			case "bool":
				if _, ok := status.Value.(bool); !ok {
					t.Errorf("status.Value type = %T, want bool", status.Value)
				}
			case "string":
				if _, ok := status.Value.(string); !ok {
					t.Errorf("status.Value type = %T, want string", status.Value)
				}
			}

			if status.LastUpdateTS != tt.wantLastUpdate {
				t.Errorf("status.LastUpdateTS = %v, want %v", status.LastUpdateTS, tt.wantLastUpdate)
			}
		})
	}
}

func TestBTHomeSensor_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	sensor := NewBTHomeSensor(client, 200)
	testComponentError(t, "GetStatus", func() error {
		_, err := sensor.GetStatus(context.Background())
		return err
	})
}

func TestBTHomeSensor_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	sensor := NewBTHomeSensor(client, 200)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := sensor.GetStatus(context.Background())
		return err
	})
}

func TestBTHomeSensorConfig_JSONSerialization(t *testing.T) {
	tests := []struct {
		config BTHomeSensorConfig
		check  func(t *testing.T, data map[string]any)
		name   string
	}{
		{
			name: "full config",
			config: BTHomeSensorConfig{
				ID:    200,
				Addr:  "3c:2e:f5:71:d5:2a",
				Name:  ptr("Test Sensor"),
				ObjID: 45,
				Idx:   3,
			},
			check: func(t *testing.T, data map[string]any) {
				if data["id"].(float64) != 200 {
					t.Errorf("id = %v, want 200", data["id"])
				}
				if data["addr"].(string) != "3c:2e:f5:71:d5:2a" {
					t.Errorf("addr = %v, want 3c:2e:f5:71:d5:2a", data["addr"])
				}
				if data["name"].(string) != "Test Sensor" {
					t.Errorf("name = %v, want Test Sensor", data["name"])
				}
				if data["obj_id"].(float64) != 45 {
					t.Errorf("obj_id = %v, want 45", data["obj_id"])
				}
				if data["idx"].(float64) != 3 {
					t.Errorf("idx = %v, want 3", data["idx"])
				}
			},
		},
		{
			name: "minimal config",
			config: BTHomeSensorConfig{
				ID:    201,
				Addr:  "11:22:33:44:55:66",
				ObjID: 2,
				Idx:   0,
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["name"]; ok {
					t.Error("name should not be present")
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

func TestBTHomeSensorStatus_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		wantValueType string
		wantID        int
	}{
		{
			name:          "numeric value",
			json:          `{"id":200,"value":25.5,"last_updated_ts":1706593991.91}`,
			wantID:        200,
			wantValueType: "float64",
		},
		{
			name:          "boolean value",
			json:          `{"id":200,"value":true,"last_updated_ts":1706593991.91}`,
			wantID:        200,
			wantValueType: "bool",
		},
		{
			name:          "string value",
			json:          `{"id":200,"value":"open","last_updated_ts":1706593991.91}`,
			wantID:        200,
			wantValueType: "string",
		},
		{
			name:          "with unknown fields",
			json:          `{"id":200,"value":0,"last_updated_ts":0,"future_field":"value"}`,
			wantID:        200,
			wantValueType: "float64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status BTHomeSensorStatus
			if err := json.Unmarshal([]byte(tt.json), &status); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if status.ID != tt.wantID {
				t.Errorf("ID = %d, want %d", status.ID, tt.wantID)
			}

			switch tt.wantValueType {
			case "float64":
				if _, ok := status.Value.(float64); !ok {
					t.Errorf("Value type = %T, want float64", status.Value)
				}
			case "bool":
				if _, ok := status.Value.(bool); !ok {
					t.Errorf("Value type = %T, want bool", status.Value)
				}
			case "string":
				if _, ok := status.Value.(string); !ok {
					t.Errorf("Value type = %T, want string", status.Value)
				}
			}
		})
	}
}

func TestBTHomeSensor_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"id": 200, "value": 0, "last_updated_ts": 0}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	sensor := NewBTHomeSensor(client, 200)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := sensor.GetStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
