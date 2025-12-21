package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewSensorAddon(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	addon := NewSensorAddon(client)

	if addon == nil {
		t.Fatal("NewSensorAddon returned nil")
	}

	if addon.Type() != "sensoraddon" {
		t.Errorf("Type() = %q, want %q", addon.Type(), "sensoraddon")
	}

	if addon.Key() != "sensoraddon" {
		t.Errorf("Key() = %q, want %q", addon.Key(), "sensoraddon")
	}

	if addon.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestSensorAddon_AddPeripheral(t *testing.T) {
	tests := []struct {
		name           string
		peripheralType PeripheralType
		attrs          *AddPeripheralAttrs
		result         string
		wantKeys       []string
	}{
		{
			name:           "add digital input",
			peripheralType: PeripheralTypeDigitalIn,
			attrs:          &AddPeripheralAttrs{CID: ptr(100)},
			result:         `{"input:100": {}}`,
			wantKeys:       []string{"input:100"},
		},
		{
			name:           "add analog input",
			peripheralType: PeripheralTypeAnalogIn,
			attrs:          nil,
			result:         `{"input:100": {}}`,
			wantKeys:       []string{"input:100"},
		},
		{
			name:           "add ds18b20",
			peripheralType: PeripheralTypeDS18B20,
			attrs: &AddPeripheralAttrs{
				CID:  ptr(101),
				Addr: ptr("40:255:100:6:199:204:149:177"),
			},
			result:   `{"temperature:101": {}}`,
			wantKeys: []string{"temperature:101"},
		},
		{
			name:           "add dht22",
			peripheralType: PeripheralTypeDHT22,
			attrs:          nil,
			result:         `{"humidity:100": {}, "temperature:100": {}}`,
			wantKeys:       []string{"humidity:100", "temperature:100"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "SensorAddon.AddPeripheral" {
						t.Errorf("method = %q, want %q", method, "SensorAddon.AddPeripheral")
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			addon := NewSensorAddon(client)

			resp, err := addon.AddPeripheral(context.Background(), tt.peripheralType, tt.attrs)
			if err != nil {
				t.Fatalf("AddPeripheral() error = %v", err)
			}

			for _, key := range tt.wantKeys {
				if _, ok := resp[key]; !ok {
					t.Errorf("response missing key %q", key)
				}
			}
		})
	}
}

func TestSensorAddon_AddPeripheral_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	addon := NewSensorAddon(client)
	testComponentError(t, "AddPeripheral", func() error {
		_, err := addon.AddPeripheral(context.Background(), PeripheralTypeDigitalIn, nil)
		return err
	})
}

func TestSensorAddon_AddPeripheral_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	addon := NewSensorAddon(client)
	testComponentInvalidJSON(t, "AddPeripheral", func() error {
		_, err := addon.AddPeripheral(context.Background(), PeripheralTypeDigitalIn, nil)
		return err
	})
}

func TestSensorAddon_GetPeripherals(t *testing.T) {
	tests := []struct {
		check  func(t *testing.T, resp GetPeripheralsResponse)
		name   string
		result string
	}{
		{
			name:   "empty peripherals",
			result: `{"digital_in": {}, "ds18b20": {}, "dht22": {}, "analog_in": {}}`,
			check: func(t *testing.T, resp GetPeripheralsResponse) {
				if len(resp[PeripheralTypeDigitalIn]) != 0 {
					t.Error("expected empty digital_in")
				}
			},
		},
		{
			name:   "with peripherals",
			result: `{"digital_in": {"input:100": {}}, "ds18b20": {"temperature:100": {"addr": "40:255:100:6:199:204:149:177"}}, "dht22": {}, "analog_in": {"input:101": {}}}`,
			check: func(t *testing.T, resp GetPeripheralsResponse) {
				if _, ok := resp[PeripheralTypeDigitalIn]["input:100"]; !ok {
					t.Error("expected input:100 in digital_in")
				}
				temp, ok := resp[PeripheralTypeDS18B20]["temperature:100"]
				if !ok {
					t.Error("expected temperature:100 in ds18b20")
				}
				if temp.Addr == nil || *temp.Addr != "40:255:100:6:199:204:149:177" {
					t.Errorf("ds18b20 addr = %v, want 40:255:100:6:199:204:149:177", temp.Addr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "SensorAddon.GetPeripherals" {
						t.Errorf("method = %q, want %q", method, "SensorAddon.GetPeripherals")
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			addon := NewSensorAddon(client)

			resp, err := addon.GetPeripherals(context.Background())
			if err != nil {
				t.Fatalf("GetPeripherals() error = %v", err)
			}

			tt.check(t, resp)
		})
	}
}

func TestSensorAddon_GetPeripherals_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	addon := NewSensorAddon(client)
	testComponentError(t, "GetPeripherals", func() error {
		_, err := addon.GetPeripherals(context.Background())
		return err
	})
}

func TestSensorAddon_GetPeripherals_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	addon := NewSensorAddon(client)
	testComponentInvalidJSON(t, "GetPeripherals", func() error {
		_, err := addon.GetPeripherals(context.Background())
		return err
	})
}

func TestSensorAddon_RemovePeripheral(t *testing.T) {
	tests := []struct {
		name      string
		component string
	}{
		{
			name:      "remove temperature",
			component: "temperature:100",
		},
		{
			name:      "remove input",
			component: "input:100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "SensorAddon.RemovePeripheral" {
						t.Errorf("method = %q, want %q", method, "SensorAddon.RemovePeripheral")
					}
					return jsonrpcResponse(`null`)
				},
			}
			client := rpc.NewClient(tr)
			addon := NewSensorAddon(client)

			err := addon.RemovePeripheral(context.Background(), tt.component)
			if err != nil {
				t.Fatalf("RemovePeripheral() error = %v", err)
			}
		})
	}
}

func TestSensorAddon_RemovePeripheral_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	addon := NewSensorAddon(client)
	testComponentError(t, "RemovePeripheral", func() error {
		return addon.RemovePeripheral(context.Background(), "temperature:100")
	})
}

func TestSensorAddon_UpdatePeripheral(t *testing.T) {
	tests := []struct {
		attrs     *UpdatePeripheralAttrs
		name      string
		component string
	}{
		{
			name:      "update ds18b20 address",
			component: "temperature:100",
			attrs: &UpdatePeripheralAttrs{
				Addr: "40:255:100:6:199:204:149:178",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "SensorAddon.UpdatePeripheral" {
						t.Errorf("method = %q, want %q", method, "SensorAddon.UpdatePeripheral")
					}
					return jsonrpcResponse(`null`)
				},
			}
			client := rpc.NewClient(tr)
			addon := NewSensorAddon(client)

			err := addon.UpdatePeripheral(context.Background(), tt.component, tt.attrs)
			if err != nil {
				t.Fatalf("UpdatePeripheral() error = %v", err)
			}
		})
	}
}

func TestSensorAddon_UpdatePeripheral_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	addon := NewSensorAddon(client)
	testComponentError(t, "UpdatePeripheral", func() error {
		return addon.UpdatePeripheral(context.Background(), "temperature:100", &UpdatePeripheralAttrs{Addr: "test"})
	})
}

func TestSensorAddon_OneWireScan(t *testing.T) {
	tests := []struct {
		name        string
		result      string
		wantDevices int
	}{
		{
			name:        "no devices",
			result:      `{"devices": []}`,
			wantDevices: 0,
		},
		{
			name:        "single device",
			result:      `{"devices": [{"type": "ds18b20", "addr": "40:255:100:6:199:204:149:177", "component": null}]}`,
			wantDevices: 1,
		},
		{
			name:        "multiple devices",
			result:      `{"devices": [{"type": "ds18b20", "addr": "40:255:100:6:199:204:149:177", "component": null}, {"type": "ds18b20", "addr": "40:255:100:6:199:204:149:178", "component": "temperature:100"}]}`,
			wantDevices: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "SensorAddon.OneWireScan" {
						t.Errorf("method = %q, want %q", method, "SensorAddon.OneWireScan")
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			addon := NewSensorAddon(client)

			resp, err := addon.OneWireScan(context.Background())
			if err != nil {
				t.Fatalf("OneWireScan() error = %v", err)
			}

			if len(resp.Devices) != tt.wantDevices {
				t.Errorf("len(Devices) = %d, want %d", len(resp.Devices), tt.wantDevices)
			}
		})
	}
}

func TestSensorAddon_OneWireScan_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	addon := NewSensorAddon(client)
	testComponentError(t, "OneWireScan", func() error {
		_, err := addon.OneWireScan(context.Background())
		return err
	})
}

func TestSensorAddon_OneWireScan_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	addon := NewSensorAddon(client)
	testComponentInvalidJSON(t, "OneWireScan", func() error {
		_, err := addon.OneWireScan(context.Background())
		return err
	})
}

func TestOneWireDevice_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		wantComponent *string
		name          string
		json          string
		wantType      string
		wantAddr      string
	}{
		{
			name:          "unlinked device",
			json:          `{"type":"ds18b20","addr":"40:255:100:6:199:204:149:177","component":null}`,
			wantType:      "ds18b20",
			wantAddr:      "40:255:100:6:199:204:149:177",
			wantComponent: nil,
		},
		{
			name:          "linked device",
			json:          `{"type":"ds18b20","addr":"40:255:100:6:199:204:149:178","component":"temperature:100"}`,
			wantType:      "ds18b20",
			wantAddr:      "40:255:100:6:199:204:149:178",
			wantComponent: ptr("temperature:100"),
		},
		{
			name:          "unknown type",
			json:          `{"type":"unknown","addr":"00:00:00:00:00:00:00:00","component":null}`,
			wantType:      "unknown",
			wantAddr:      "00:00:00:00:00:00:00:00",
			wantComponent: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var device OneWireDevice
			if err := json.Unmarshal([]byte(tt.json), &device); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if device.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", device.Type, tt.wantType)
			}

			if device.Addr != tt.wantAddr {
				t.Errorf("Addr = %q, want %q", device.Addr, tt.wantAddr)
			}

			if tt.wantComponent != nil {
				if device.Component == nil || *device.Component != *tt.wantComponent {
					t.Errorf("Component = %v, want %v", device.Component, *tt.wantComponent)
				}
			} else {
				if device.Component != nil {
					t.Errorf("Component = %v, want nil", device.Component)
				}
			}
		})
	}
}

func TestPeripheralType_Constants(t *testing.T) {
	if PeripheralTypeDS18B20 != "ds18b20" {
		t.Errorf("PeripheralTypeDS18B20 = %q, want %q", PeripheralTypeDS18B20, "ds18b20")
	}
	if PeripheralTypeDHT22 != "dht22" {
		t.Errorf("PeripheralTypeDHT22 = %q, want %q", PeripheralTypeDHT22, "dht22")
	}
	if PeripheralTypeDigitalIn != "digital_in" {
		t.Errorf("PeripheralTypeDigitalIn = %q, want %q", PeripheralTypeDigitalIn, "digital_in")
	}
	if PeripheralTypeAnalogIn != "analog_in" {
		t.Errorf("PeripheralTypeAnalogIn = %q, want %q", PeripheralTypeAnalogIn, "analog_in")
	}
}

func TestSensorAddon_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			_ = req.GetMethod()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"devices": []}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	addon := NewSensorAddon(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := addon.OneWireScan(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
