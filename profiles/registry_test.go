package profiles

import (
	"testing"

	"github.com/tj-smith47/shelly-go/types"
)

func TestRegisterAndGet(t *testing.T) {
	// Save state and clear for test
	Clear()
	defer func() {
		// Re-import profiles after test
		Clear()
	}()

	profile := &Profile{
		Model:      "TEST-001",
		Name:       "Test Device",
		App:        "TestApp",
		Generation: types.Gen2,
		Series:     SeriesPlus,
	}

	Register(profile)

	got, ok := Get("TEST-001")
	if !ok {
		t.Fatal("Get() returned false")
	}
	if got.Name != "Test Device" {
		t.Errorf("Name = %v, want Test Device", got.Name)
	}
}

func TestGetByApp(t *testing.T) {
	Clear()
	defer Clear()

	profile := &Profile{
		Model:      "TEST-002",
		Name:       "Test Device 2",
		App:        "TestApp2",
		Generation: types.Gen2,
	}

	Register(profile)

	got, ok := GetByApp("TestApp2")
	if !ok {
		t.Fatal("GetByApp() returned false")
	}
	if got.Model != "TEST-002" {
		t.Errorf("Model = %v, want TEST-002", got.Model)
	}

	// Non-existent app
	_, ok = GetByApp("NonExistent")
	if ok {
		t.Error("GetByApp(NonExistent) should return false")
	}
}

func TestMustGet(t *testing.T) {
	Clear()
	defer Clear()

	profile := &Profile{Model: "TEST-003"}
	Register(profile)

	// Should not panic
	got := MustGet("TEST-003")
	if got.Model != "TEST-003" {
		t.Error("MustGet returned wrong profile")
	}

	// Should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet should panic for non-existent model")
		}
	}()
	MustGet("NONEXISTENT")
}

func TestExists(t *testing.T) {
	Clear()
	defer Clear()

	profile := &Profile{Model: "TEST-004"}
	Register(profile)

	if !Exists("TEST-004") {
		t.Error("Exists() = false, want true")
	}
	if Exists("NONEXISTENT") {
		t.Error("Exists(NONEXISTENT) = true, want false")
	}
}

func TestList(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{Model: "TEST-A"})
	Register(&Profile{Model: "TEST-B"})
	Register(&Profile{Model: "TEST-C"})

	profiles := List()
	if len(profiles) != 3 {
		t.Errorf("len(List()) = %d, want 3", len(profiles))
	}
}

func TestListByGeneration(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{Model: "GEN1-A", Generation: types.Gen1})
	Register(&Profile{Model: "GEN1-B", Generation: types.Gen1})
	Register(&Profile{Model: "GEN2-A", Generation: types.Gen2})
	Register(&Profile{Model: "GEN3-A", Generation: types.Gen3})

	gen1 := ListByGeneration(types.Gen1)
	if len(gen1) != 2 {
		t.Errorf("len(ListByGeneration(Gen1)) = %d, want 2", len(gen1))
	}

	gen2 := ListByGeneration(types.Gen2)
	if len(gen2) != 1 {
		t.Errorf("len(ListByGeneration(Gen2)) = %d, want 1", len(gen2))
	}

	gen4 := ListByGeneration(types.Gen4)
	if len(gen4) != 0 {
		t.Errorf("len(ListByGeneration(Gen4)) = %d, want 0", len(gen4))
	}
}

func TestListBySeries(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{Model: "PLUS-A", Series: SeriesPlus})
	Register(&Profile{Model: "PLUS-B", Series: SeriesPlus})
	Register(&Profile{Model: "PRO-A", Series: SeriesPro})

	plus := ListBySeries(SeriesPlus)
	if len(plus) != 2 {
		t.Errorf("len(ListBySeries(Plus)) = %d, want 2", len(plus))
	}

	pro := ListBySeries(SeriesPro)
	if len(pro) != 1 {
		t.Errorf("len(ListBySeries(Pro)) = %d, want 1", len(pro))
	}
}

func TestListByCapability(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{
		Model:        "PM-A",
		Capabilities: Capabilities{PowerMetering: true},
	})
	Register(&Profile{
		Model:        "PM-B",
		Capabilities: Capabilities{PowerMetering: true, CoverSupport: true},
	})
	Register(&Profile{
		Model:        "NOPW-A",
		Capabilities: Capabilities{CoverSupport: true},
	})

	pm := ListByCapability("power_metering")
	if len(pm) != 2 {
		t.Errorf("len(ListByCapability(power_metering)) = %d, want 2", len(pm))
	}

	cover := ListByCapability("cover_support")
	if len(cover) != 2 {
		t.Errorf("len(ListByCapability(cover_support)) = %d, want 2", len(cover))
	}

	// Test various capability name formats
	variants := []string{
		"powermetering", "PowerMetering",
		"cover", "coversupport",
		"dimming", "dimmingsupport",
		"color", "colorsupport", "rgb",
		"colortemperature", "cct",
		"scripting", "scripts",
		"schedules",
		"webhooks",
		"kvs",
		"virtualcomponents",
		"actions",
		"sensoraddon",
		"calibration",
		"inputevents",
		"effects",
		"noneutral",
		"bidirectionalmetering",
		"threephase", "3phase",
	}

	for _, v := range variants {
		// Just ensure no panic
		ListByCapability(v)
	}
}

func TestListByProtocol(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{
		Model:     "WS-A",
		Protocols: Protocols{HTTP: true, WebSocket: true},
	})
	Register(&Profile{
		Model:     "COIOT-A",
		Protocols: Protocols{HTTP: true, CoIoT: true},
	})

	ws := ListByProtocol("websocket")
	if len(ws) != 1 {
		t.Errorf("len(ListByProtocol(websocket)) = %d, want 1", len(ws))
	}

	coiot := ListByProtocol("coiot")
	if len(coiot) != 1 {
		t.Errorf("len(ListByProtocol(coiot)) = %d, want 1", len(coiot))
	}

	// Test various protocol name formats
	variants := []string{
		"http", "ws", "mqtt", "coap",
		"ble", "bluetooth",
		"matter", "zigbee",
		"zwave", "z-wave",
		"ethernet", "eth",
	}

	for _, v := range variants {
		ListByProtocol(v)
	}
}

func TestListByComponent(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{
		Model:      "SW-A",
		Components: Components{Switches: 2},
	})
	Register(&Profile{
		Model:      "LIGHT-A",
		Components: Components{Lights: 1},
	})

	switches := ListByComponent(types.ComponentTypeSwitch)
	if len(switches) != 1 {
		t.Errorf("len(ListByComponent(Switch)) = %d, want 1", len(switches))
	}

	lights := ListByComponent(types.ComponentTypeLight)
	if len(lights) != 1 {
		t.Errorf("len(ListByComponent(Light)) = %d, want 1", len(lights))
	}
}

func TestListByFormFactor(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{Model: "DIN-A", FormFactor: FormFactorDIN})
	Register(&Profile{Model: "DIN-B", FormFactor: FormFactorDIN})
	Register(&Profile{Model: "PLUG-A", FormFactor: FormFactorPlug})

	din := ListByFormFactor(FormFactorDIN)
	if len(din) != 2 {
		t.Errorf("len(ListByFormFactor(DIN)) = %d, want 2", len(din))
	}

	plug := ListByFormFactor(FormFactorPlug)
	if len(plug) != 1 {
		t.Errorf("len(ListByFormFactor(Plug)) = %d, want 1", len(plug))
	}
}

func TestListByPowerSource(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{Model: "BAT-A", PowerSource: PowerSourceBattery})
	Register(&Profile{Model: "MAINS-A", PowerSource: PowerSourceMains})
	Register(&Profile{Model: "MAINS-B", PowerSource: PowerSourceMains})

	battery := ListByPowerSource(PowerSourceBattery)
	if len(battery) != 1 {
		t.Errorf("len(ListByPowerSource(Battery)) = %d, want 1", len(battery))
	}

	mains := ListByPowerSource(PowerSourceMains)
	if len(mains) != 2 {
		t.Errorf("len(ListByPowerSource(Mains)) = %d, want 2", len(mains))
	}
}

func TestCount(t *testing.T) {
	Clear()
	defer Clear()

	if Count() != 0 {
		t.Errorf("Count() = %d, want 0", Count())
	}

	Register(&Profile{Model: "A"})
	Register(&Profile{Model: "B"})

	if Count() != 2 {
		t.Errorf("Count() = %d, want 2", Count())
	}
}

func TestSearch(t *testing.T) {
	Clear()
	defer Clear()

	Register(&Profile{Model: "SHSW-1", Name: "Shelly 1"})
	Register(&Profile{Model: "SNSW-001P16EU", Name: "Shelly Plus 1PM", App: "Plus1PM"})
	Register(&Profile{Model: "SNSW-002P16EU", Name: "Shelly Plus 2PM", App: "Plus2PM"})

	// Search by model
	results := Search("SHSW")
	if len(results) != 1 {
		t.Errorf("Search(SHSW) len = %d, want 1", len(results))
	}

	// Search by name
	results = Search("plus")
	if len(results) != 2 {
		t.Errorf("Search(plus) len = %d, want 2", len(results))
	}

	// Search by app
	results = Search("PM")
	if len(results) != 2 { // Plus1PM and Plus2PM both contain PM
		t.Errorf("Search(PM) len = %d, want 2", len(results))
	}

	// No results
	results = Search("xyz")
	if len(results) != 0 {
		t.Errorf("Search(xyz) len = %d, want 0", len(results))
	}
}

func TestDetectGeneration(t *testing.T) {
	tests := []struct {
		model string
		want  types.Generation
	}{
		{"SHSW-1", types.Gen1},
		{"SHPLG-S", types.Gen1},
		{"SNSW-001P16EU", types.Gen2},
		{"SNPL-00112EU", types.Gen2},
		{"S3SW-001P16EU", types.Gen3},
		{"S3PL-00112EU", types.Gen3},
		{"S4SW-001P16EU", types.Gen4},
		{"S4PL-00116US", types.Gen4},
		{"UNKNOWN", types.GenerationUnknown},
		{"", types.GenerationUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := DetectGeneration(tt.model); got != tt.want {
				t.Errorf("DetectGeneration(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestClear(t *testing.T) {
	// Start fresh
	Clear()

	Register(&Profile{Model: "A"})
	Register(&Profile{Model: "B", App: "AppB"})

	if Count() != 2 {
		t.Errorf("Count() = %d, want 2", Count())
	}

	Clear()

	if Count() != 0 {
		t.Errorf("Count() after Clear = %d, want 0", Count())
	}

	// App index should also be cleared
	_, ok := GetByApp("AppB")
	if ok {
		t.Error("GetByApp should return false after Clear")
	}
}
