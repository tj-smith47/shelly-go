package gen2

import (
	"context"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/types"
)

func TestNewDevice(t *testing.T) {
	mt := &mockTransport{}
	client := newTestClient(mt)

	device := NewDevice(client)

	if device == nil {
		t.Fatal("NewDevice() returned nil")
	}

	if device.client == nil {
		t.Error("NewDevice() client is nil")
	}

	if device.shelly == nil {
		t.Error("NewDevice() shelly namespace is nil")
	}
}

func TestDevice_Shelly(t *testing.T) {
	mt := &mockTransport{}
	client := newTestClient(mt)
	device := NewDevice(client)

	shelly := device.Shelly()

	if shelly == nil {
		t.Fatal("Shelly() returned nil")
	}

	if shelly != device.shelly {
		t.Error("Shelly() returned different instance")
	}
}

func TestDevice_Client(t *testing.T) {
	mt := &mockTransport{}
	client := newTestClient(mt)
	device := NewDevice(client)

	gotClient := device.Client()

	if gotClient == nil {
		t.Fatal("Client() returned nil")
	}

	if gotClient != client {
		t.Error("Client() returned different instance")
	}
}

func TestDevice_GetDeviceInfo(t *testing.T) {
	t.Run("first call fetches from device", func(t *testing.T) {
		mt := &mockTransport{
			response: []byte(`{
				"name": "My Shelly",
				"id": "shellypro1pm-abc123",
				"mac": "ABC123",
				"model": "SPR-1PCBA1EU",
				"gen": 2,
				"fw_id": "test",
				"ver": "1.0.0",
				"app": "Pro1PM"
			}`),
		}
		client := newTestClient(mt)
		device := NewDevice(client)

		info, err := device.GetDeviceInfo(context.Background())

		if err != nil {
			t.Fatalf("GetDeviceInfo() error = %v", err)
		}

		if info.Name != "My Shelly" {
			t.Errorf("Name = %v, want My Shelly", info.Name)
		}

		if info.Model != "SPR-1PCBA1EU" {
			t.Errorf("Model = %v, want SPR-1PCBA1EU", info.Model)
		}
	})

	t.Run("subsequent calls return cached value", func(t *testing.T) {
		mt := &mockTransport{
			response: []byte(`{
				"name": "First Call",
				"id": "test-id",
				"mac": "ABC123",
				"model": "TEST",
				"gen": 2,
				"fw_id": "test",
				"ver": "1.0.0",
				"app": "Test"
			}`),
		}
		client := newTestClient(mt)
		device := NewDevice(client)

		// First call
		info1, err := device.GetDeviceInfo(context.Background())
		if err != nil {
			t.Fatalf("GetDeviceInfo() first call error = %v", err)
		}

		// Change mock response
		mt.response = []byte(`{
			"name": "Second Call",
			"id": "test-id",
			"mac": "ABC123",
			"model": "TEST",
			"gen": 2,
			"fw_id": "test",
			"ver": "1.0.0",
			"app": "Test"
		}`)

		// Second call should return cached value
		info2, err := device.GetDeviceInfo(context.Background())
		if err != nil {
			t.Fatalf("GetDeviceInfo() second call error = %v", err)
		}

		if info2.Name != "First Call" {
			t.Errorf("Second call returned Name = %v, want First Call (cached)", info2.Name)
		}

		// Verify same instance
		if info1 != info2 {
			t.Error("GetDeviceInfo() returned different instance on second call")
		}
	})

	t.Run("error on first call", func(t *testing.T) {
		mt := &mockTransport{
			err: errors.New("connection failed"),
		}
		client := newTestClient(mt)
		device := NewDevice(client)

		_, err := device.GetDeviceInfo(context.Background())

		if err == nil {
			t.Error("GetDeviceInfo() expected error, got nil")
		}
	})
}

func TestDevice_Generation(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     types.Generation
		wantErr  bool
	}{
		{
			name: "Gen2 Plus device",
			response: `{
				"id": "shellyplus1pm-abc123",
				"mac": "ABC123",
				"model": "SNSW-001P16EU",
				"gen": 2,
				"fw_id": "test",
				"ver": "1.0.0",
				"app": "Plus1PM"
			}`,
			want:    types.Gen2Plus,
			wantErr: false,
		},
		{
			name: "Gen2 Pro device",
			response: `{
				"id": "shellypro1pm-abc123",
				"mac": "ABC123",
				"model": "SPR-1PCBA1EU",
				"gen": 2,
				"fw_id": "test",
				"ver": "1.0.0",
				"app": "Pro1PM"
			}`,
			want:    types.Gen2Pro,
			wantErr: false,
		},
		{
			name: "Gen3 device",
			response: `{
				"id": "shelly1gen3-abc123",
				"mac": "ABC123",
				"model": "S3SW-001X16EU",
				"gen": 3,
				"fw_id": "test",
				"ver": "1.0.0",
				"app": "Switch1Gen3"
			}`,
			want:    types.Gen3,
			wantErr: false,
		},
		{
			name: "Gen4 device",
			response: `{
				"id": "shelly1gen4-abc123",
				"mac": "ABC123",
				"model": "S4SW-001X16EU",
				"gen": 4,
				"fw_id": "test",
				"ver": "1.0.0",
				"app": "Switch1Gen4"
			}`,
			want:    types.Gen4,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				response: []byte(tt.response),
			}
			client := newTestClient(mt)
			device := NewDevice(client)

			got, err := device.Generation(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Generation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("Generation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevice_Generation_Error(t *testing.T) {
	t.Run("unknown generation number", func(t *testing.T) {
		mt := &mockTransport{
			response: []byte(`{
				"id": "test",
				"mac": "ABC123",
				"model": "TEST",
				"gen": 99,
				"fw_id": "test",
				"ver": "1.0.0",
				"app": "Test"
			}`),
		}
		client := newTestClient(mt)
		device := NewDevice(client)

		gen, err := device.Generation(context.Background())

		if err == nil {
			t.Error("Generation() expected error for unknown gen, got nil")
		}

		if gen != types.GenUnknown {
			t.Errorf("Generation() = %v, want GenUnknown", gen)
		}
	})

	t.Run("device info fetch error", func(t *testing.T) {
		mt := &mockTransport{
			err: errors.New("connection failed"),
		}
		client := newTestClient(mt)
		device := NewDevice(client)

		_, err := device.Generation(context.Background())

		if err == nil {
			t.Error("Generation() expected error, got nil")
		}
	})
}

func TestDevice_Component(t *testing.T) {
	mt := &mockTransport{}
	client := newTestClient(mt)
	device := NewDevice(client)

	comp := device.Component("switch", 0)

	if comp == nil {
		t.Fatal("Component() returned nil")
	}

	if comp.Type() != "switch" {
		t.Errorf("Component().Type() = %v, want switch", comp.Type())
	}

	if comp.ID() != 0 {
		t.Errorf("Component().ID() = %v, want 0", comp.ID())
	}

	if comp.Key() != "switch:0" {
		t.Errorf("Component().Key() = %v, want switch:0", comp.Key())
	}
}

func TestDevice_Close(t *testing.T) {
	mt := &mockTransport{}
	client := newTestClient(mt)
	device := NewDevice(client)

	err := device.Close()

	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestDevice_ComponentAccessors(t *testing.T) {
	tests := []struct {
		accessor func(*Device) Component
		name     string
		wantType string
		wantKey  string
		wantID   int
	}{
		{
			name:     "Switch",
			accessor: func(d *Device) Component { return d.Switch(0) },
			wantType: "switch",
			wantID:   0,
			wantKey:  "switch:0",
		},
		{
			name:     "Cover",
			accessor: func(d *Device) Component { return d.Cover(0) },
			wantType: "cover",
			wantID:   0,
			wantKey:  "cover:0",
		},
		{
			name:     "Light",
			accessor: func(d *Device) Component { return d.Light(0) },
			wantType: "light",
			wantID:   0,
			wantKey:  "light:0",
		},
		{
			name:     "Input",
			accessor: func(d *Device) Component { return d.Input(0) },
			wantType: "input",
			wantID:   0,
			wantKey:  "input:0",
		},
		{
			name:     "DevicePower",
			accessor: func(d *Device) Component { return d.DevicePower(0) },
			wantType: "devicepower",
			wantID:   0,
			wantKey:  "devicepower:0",
		},
		{
			name:     "PM",
			accessor: func(d *Device) Component { return d.PM(0) },
			wantType: "pm",
			wantID:   0,
			wantKey:  "pm:0",
		},
		{
			name:     "PM1",
			accessor: func(d *Device) Component { return d.PM1(0) },
			wantType: "pm1",
			wantID:   0,
			wantKey:  "pm1:0",
		},
		{
			name:     "EM",
			accessor: func(d *Device) Component { return d.EM(0) },
			wantType: "em",
			wantID:   0,
			wantKey:  "em:0",
		},
		{
			name:     "EM1",
			accessor: func(d *Device) Component { return d.EM1(0) },
			wantType: "em1",
			wantID:   0,
			wantKey:  "em1:0",
		},
		{
			name:     "Voltmeter",
			accessor: func(d *Device) Component { return d.Voltmeter(0) },
			wantType: "voltmeter",
			wantID:   0,
			wantKey:  "voltmeter:0",
		},
		{
			name:     "Temperature",
			accessor: func(d *Device) Component { return d.Temperature(0) },
			wantType: "temperature",
			wantID:   0,
			wantKey:  "temperature:0",
		},
		{
			name:     "Humidity",
			accessor: func(d *Device) Component { return d.Humidity(0) },
			wantType: "humidity",
			wantID:   0,
			wantKey:  "humidity:0",
		},
		{
			name:     "Smoke",
			accessor: func(d *Device) Component { return d.Smoke(0) },
			wantType: "smoke",
			wantID:   0,
			wantKey:  "smoke:0",
		},
		{
			name:     "Thermostat",
			accessor: func(d *Device) Component { return d.Thermostat(0) },
			wantType: "thermostat",
			wantID:   0,
			wantKey:  "thermostat:0",
		},
		{
			name:     "Script",
			accessor: func(d *Device) Component { return d.Script(1) },
			wantType: "script",
			wantID:   1,
			wantKey:  "script:1",
		},
		{
			name:     "Schedule",
			accessor: func(d *Device) Component { return d.Schedule(2) },
			wantType: "schedule",
			wantID:   2,
			wantKey:  "schedule:2",
		},
		{
			name:     "Webhook",
			accessor: func(d *Device) Component { return d.Webhook(3) },
			wantType: "webhook",
			wantID:   3,
			wantKey:  "webhook:3",
		},
		{
			name:     "WiFi",
			accessor: func(d *Device) Component { return d.WiFi(0) },
			wantType: "wifi",
			wantID:   0,
			wantKey:  "wifi:0",
		},
		{
			name:     "Ethernet",
			accessor: func(d *Device) Component { return d.Ethernet(0) },
			wantType: "eth",
			wantID:   0,
			wantKey:  "eth:0",
		},
		{
			name:     "BLE",
			accessor: func(d *Device) Component { return d.BLE(0) },
			wantType: "ble",
			wantID:   0,
			wantKey:  "ble:0",
		},
		{
			name:     "Cloud",
			accessor: func(d *Device) Component { return d.Cloud(0) },
			wantType: "cloud",
			wantID:   0,
			wantKey:  "cloud:0",
		},
		{
			name:     "MQTT",
			accessor: func(d *Device) Component { return d.MQTT(0) },
			wantType: "mqtt",
			wantID:   0,
			wantKey:  "mqtt:0",
		},
		{
			name:     "WS",
			accessor: func(d *Device) Component { return d.WS(0) },
			wantType: "ws",
			wantID:   0,
			wantKey:  "ws:0",
		},
		{
			name:     "Sys",
			accessor: func(d *Device) Component { return d.Sys(0) },
			wantType: "sys",
			wantID:   0,
			wantKey:  "sys:0",
		},
		{
			name:     "UI",
			accessor: func(d *Device) Component { return d.UI(0) },
			wantType: "ui",
			wantID:   0,
			wantKey:  "ui:0",
		},
		{
			name:     "KVS",
			accessor: func(d *Device) Component { return d.KVS() },
			wantType: "kvs",
			wantID:   0,
			wantKey:  "kvs:0",
		},
		{
			name:     "BTHome",
			accessor: func(d *Device) Component { return d.BTHome(0) },
			wantType: "bthome",
			wantID:   0,
			wantKey:  "bthome:0",
		},
		{
			name:     "BTHomeDevice",
			accessor: func(d *Device) Component { return d.BTHomeDevice(1) },
			wantType: "bthomedevice",
			wantID:   1,
			wantKey:  "bthomedevice:1",
		},
		{
			name:     "BTHomeSensor",
			accessor: func(d *Device) Component { return d.BTHomeSensor(2) },
			wantType: "bthomesensor",
			wantID:   2,
			wantKey:  "bthomesensor:2",
		},
		{
			name:     "RGB",
			accessor: func(d *Device) Component { return d.RGB(0) },
			wantType: "rgb",
			wantID:   0,
			wantKey:  "rgb:0",
		},
		{
			name:     "RGBW",
			accessor: func(d *Device) Component { return d.RGBW(0) },
			wantType: "rgbw",
			wantID:   0,
			wantKey:  "rgbw:0",
		},
		{
			name:     "ModBus",
			accessor: func(d *Device) Component { return d.ModBus(0) },
			wantType: "modbus",
			wantID:   0,
			wantKey:  "modbus:0",
		},
		{
			name:     "SensorAddon",
			accessor: func(d *Device) Component { return d.SensorAddon(0) },
			wantType: "sensoraddon",
			wantID:   0,
			wantKey:  "sensoraddon:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{}
			client := newTestClient(mt)
			device := NewDevice(client)

			comp := tt.accessor(device)

			if comp == nil {
				t.Fatal("accessor returned nil")
			}

			if comp.Type() != tt.wantType {
				t.Errorf("Type() = %v, want %v", comp.Type(), tt.wantType)
			}

			if comp.ID() != tt.wantID {
				t.Errorf("ID() = %v, want %v", comp.ID(), tt.wantID)
			}

			if comp.Key() != tt.wantKey {
				t.Errorf("Key() = %v, want %v", comp.Key(), tt.wantKey)
			}
		})
	}
}

func TestDevice_MultipleComponentIDs(t *testing.T) {
	mt := &mockTransport{}
	client := newTestClient(mt)
	device := NewDevice(client)

	// Test that different IDs work correctly
	switch0 := device.Switch(0)
	switch1 := device.Switch(1)
	switch2 := device.Switch(2)

	if switch0.Key() != "switch:0" {
		t.Errorf("switch0.Key() = %v, want switch:0", switch0.Key())
	}

	if switch1.Key() != "switch:1" {
		t.Errorf("switch1.Key() = %v, want switch:1", switch1.Key())
	}

	if switch2.Key() != "switch:2" {
		t.Errorf("switch2.Key() = %v, want switch:2", switch2.Key())
	}
}
