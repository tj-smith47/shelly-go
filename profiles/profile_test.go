package profiles

import (
	"testing"

	"github.com/tj-smith47/shelly-go/types"
)

func TestProfile_HasComponent(t *testing.T) {
	profile := &Profile{
		Components: Components{
			Switches:           2,
			Covers:             1,
			Lights:             1,
			Inputs:             2,
			PowerMeters:        2,
			TemperatureSensors: 1,
		},
		Capabilities: Capabilities{
			CoverSupport: true,
		},
		Sensors: []SensorType{SensorSmoke},
	}

	tests := []struct {
		ct   types.ComponentType
		want bool
	}{
		{types.ComponentTypeSwitch, true},
		{types.ComponentTypeCover, true},
		{types.ComponentTypeLight, true},
		{types.ComponentTypeInput, true},
		{types.ComponentTypePM, true},
		{types.ComponentTypePM1, true},
		{types.ComponentTypeTemperature, true},
		{types.ComponentTypeSmoke, true},
		{types.ComponentTypeEM, false},
		{types.ComponentTypeVoltmeter, false},
		{types.ComponentTypeHumidity, false},
		{types.ComponentTypeRGB, false},
		{types.ComponentTypeThermostat, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.ct), func(t *testing.T) {
			if got := profile.HasComponent(tt.ct); got != tt.want {
				t.Errorf("HasComponent(%v) = %v, want %v", tt.ct, got, tt.want)
			}
		})
	}
}

func TestProfile_ComponentCount(t *testing.T) {
	profile := &Profile{
		Components: Components{
			Switches:           2,
			Covers:             1,
			Lights:             3,
			Inputs:             4,
			PowerMeters:        2,
			EnergyMeters:       3,
			Voltmeters:         1,
			TemperatureSensors: 2,
			HumiditySensors:    1,
			RGBChannels:        1,
		},
	}

	tests := []struct {
		ct   types.ComponentType
		want int
	}{
		{types.ComponentTypeSwitch, 2},
		{types.ComponentTypeCover, 1},
		{types.ComponentTypeLight, 3},
		{types.ComponentTypeInput, 4},
		{types.ComponentTypePM, 2},
		{types.ComponentTypeEM, 3},
		{types.ComponentTypeVoltmeter, 1},
		{types.ComponentTypeTemperature, 2},
		{types.ComponentTypeHumidity, 1},
		{types.ComponentTypeRGB, 1},
		{types.ComponentTypeSys, 0}, // Unknown type
	}

	for _, tt := range tests {
		t.Run(string(tt.ct), func(t *testing.T) {
			if got := profile.ComponentCount(tt.ct); got != tt.want {
				t.Errorf("ComponentCount(%v) = %v, want %v", tt.ct, got, tt.want)
			}
		})
	}
}

func TestProfile_GenerationHelpers(t *testing.T) {
	gen1 := &Profile{Generation: types.Gen1}
	gen2Plus := &Profile{Generation: types.Gen2, Series: SeriesPlus}
	gen2Pro := &Profile{Generation: types.Gen2, Series: SeriesPro}
	gen3 := &Profile{Generation: types.Gen3}
	gen4 := &Profile{Generation: types.Gen4}
	blu := &Profile{Generation: types.Gen2, Series: SeriesBLU}
	wave := &Profile{Generation: types.Gen2, Series: SeriesWave}
	wavePro := &Profile{Generation: types.Gen2, Series: SeriesWavePro}
	mini := &Profile{Generation: types.Gen3, Series: SeriesMini}

	tests := []struct {
		profile *Profile
		name    string
		isGen1  bool
		isGen2P bool
		isGen2R bool
		isGen3  bool
		isGen4  bool
		isBLU   bool
		isWave  bool
		isMini  bool
	}{
		{name: "Gen1", profile: gen1, isGen1: true, isGen2P: false, isGen2R: false, isGen3: false, isGen4: false, isBLU: false, isWave: false, isMini: false},
		{name: "Gen2Plus", profile: gen2Plus, isGen1: false, isGen2P: true, isGen2R: false, isGen3: false, isGen4: false, isBLU: false, isWave: false, isMini: false},
		{name: "Gen2Pro", profile: gen2Pro, isGen1: false, isGen2P: false, isGen2R: true, isGen3: false, isGen4: false, isBLU: false, isWave: false, isMini: false},
		{name: "Gen3", profile: gen3, isGen1: false, isGen2P: false, isGen2R: false, isGen3: true, isGen4: false, isBLU: false, isWave: false, isMini: false},
		{name: "Gen4", profile: gen4, isGen1: false, isGen2P: false, isGen2R: false, isGen3: false, isGen4: true, isBLU: false, isWave: false, isMini: false},
		{name: "BLU", profile: blu, isGen1: false, isGen2P: false, isGen2R: false, isGen3: false, isGen4: false, isBLU: true, isWave: false, isMini: false},
		{name: "Wave", profile: wave, isGen1: false, isGen2P: false, isGen2R: false, isGen3: false, isGen4: false, isBLU: false, isWave: true, isMini: false},
		{name: "WavePro", profile: wavePro, isGen1: false, isGen2P: false, isGen2R: false, isGen3: false, isGen4: false, isBLU: false, isWave: true, isMini: false},
		{name: "Mini", profile: mini, isGen1: false, isGen2P: false, isGen2R: false, isGen3: true, isGen4: false, isBLU: false, isWave: false, isMini: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.profile.IsGen1(); got != tt.isGen1 {
				t.Errorf("IsGen1() = %v, want %v", got, tt.isGen1)
			}
			if got := tt.profile.IsGen2Plus(); got != tt.isGen2P {
				t.Errorf("IsGen2Plus() = %v, want %v", got, tt.isGen2P)
			}
			if got := tt.profile.IsGen2Pro(); got != tt.isGen2R {
				t.Errorf("IsGen2Pro() = %v, want %v", got, tt.isGen2R)
			}
			if got := tt.profile.IsGen3(); got != tt.isGen3 {
				t.Errorf("IsGen3() = %v, want %v", got, tt.isGen3)
			}
			if got := tt.profile.IsGen4(); got != tt.isGen4 {
				t.Errorf("IsGen4() = %v, want %v", got, tt.isGen4)
			}
			if got := tt.profile.IsBLU(); got != tt.isBLU {
				t.Errorf("IsBLU() = %v, want %v", got, tt.isBLU)
			}
			if got := tt.profile.IsWave(); got != tt.isWave {
				t.Errorf("IsWave() = %v, want %v", got, tt.isWave)
			}
			if got := tt.profile.IsMini(); got != tt.isMini {
				t.Errorf("IsMini() = %v, want %v", got, tt.isMini)
			}
		})
	}
}

func TestProfile_SupportsRPCAndREST(t *testing.T) {
	gen1 := &Profile{Generation: types.Gen1}
	gen2 := &Profile{Generation: types.Gen2}
	gen3 := &Profile{Generation: types.Gen3}
	gen4 := &Profile{Generation: types.Gen4}

	if gen1.SupportsRPC() {
		t.Error("Gen1 should not support RPC")
	}
	if !gen1.SupportsREST() {
		t.Error("Gen1 should support REST")
	}

	if !gen2.SupportsRPC() {
		t.Error("Gen2 should support RPC")
	}
	if gen2.SupportsREST() {
		t.Error("Gen2 should not support REST")
	}

	if !gen3.SupportsRPC() {
		t.Error("Gen3 should support RPC")
	}
	if !gen4.SupportsRPC() {
		t.Error("Gen4 should support RPC")
	}
}

func TestProfile_IsBatteryPowered(t *testing.T) {
	battery := &Profile{PowerSource: PowerSourceBattery}
	mains := &Profile{PowerSource: PowerSourceMains}
	dc := &Profile{PowerSource: PowerSourceDC}

	if !battery.IsBatteryPowered() {
		t.Error("Battery should be battery powered")
	}
	if mains.IsBatteryPowered() {
		t.Error("Mains should not be battery powered")
	}
	if dc.IsBatteryPowered() {
		t.Error("DC should not be battery powered")
	}
}

func TestDefaultGen2Limits(t *testing.T) {
	limits := DefaultGen2Limits()

	if limits.MaxScripts != 10 {
		t.Errorf("MaxScripts = %d, want 10", limits.MaxScripts)
	}
	if limits.MaxSchedules != 20 {
		t.Errorf("MaxSchedules = %d, want 20", limits.MaxSchedules)
	}
	if limits.MaxWebhooks != 20 {
		t.Errorf("MaxWebhooks = %d, want 20", limits.MaxWebhooks)
	}
	if limits.MaxKVSEntries != 100 {
		t.Errorf("MaxKVSEntries = %d, want 100", limits.MaxKVSEntries)
	}
	if limits.MaxScriptSize != 16384 {
		t.Errorf("MaxScriptSize = %d, want 16384", limits.MaxScriptSize)
	}
}

func TestDefaultGen1Limits(t *testing.T) {
	limits := DefaultGen1Limits()

	if limits.MaxSchedules != 10 {
		t.Errorf("MaxSchedules = %d, want 10", limits.MaxSchedules)
	}
	if limits.MaxScripts != 0 {
		t.Errorf("MaxScripts should be 0 for Gen1")
	}
}

func TestDefaultGen2Capabilities(t *testing.T) {
	caps := DefaultGen2Capabilities()

	if !caps.Scripting {
		t.Error("Gen2 should have Scripting")
	}
	if !caps.Schedules {
		t.Error("Gen2 should have Schedules")
	}
	if !caps.AdvancedSchedules {
		t.Error("Gen2 should have AdvancedSchedules")
	}
	if !caps.Webhooks {
		t.Error("Gen2 should have Webhooks")
	}
	if !caps.KVS {
		t.Error("Gen2 should have KVS")
	}
	if !caps.VirtualComponents {
		t.Error("Gen2 should have VirtualComponents")
	}
	if !caps.InputEvents {
		t.Error("Gen2 should have InputEvents")
	}
}

func TestDefaultGen1Capabilities(t *testing.T) {
	caps := DefaultGen1Capabilities()

	if !caps.Schedules {
		t.Error("Gen1 should have Schedules")
	}
	if !caps.Actions {
		t.Error("Gen1 should have Actions")
	}
	if caps.Scripting {
		t.Error("Gen1 should not have Scripting")
	}
	if caps.Webhooks {
		t.Error("Gen1 should not have Webhooks")
	}
}

func TestDefaultGen2Protocols(t *testing.T) {
	proto := DefaultGen2Protocols()

	if !proto.HTTP {
		t.Error("Gen2 should have HTTP")
	}
	if !proto.WebSocket {
		t.Error("Gen2 should have WebSocket")
	}
	if !proto.MQTT {
		t.Error("Gen2 should have MQTT")
	}
	if !proto.BLE {
		t.Error("Gen2 should have BLE")
	}
	if proto.CoIoT {
		t.Error("Gen2 should not have CoIoT")
	}
}

func TestDefaultGen1Protocols(t *testing.T) {
	proto := DefaultGen1Protocols()

	if !proto.HTTP {
		t.Error("Gen1 should have HTTP")
	}
	if !proto.MQTT {
		t.Error("Gen1 should have MQTT")
	}
	if !proto.CoIoT {
		t.Error("Gen1 should have CoIoT")
	}
	if proto.WebSocket {
		t.Error("Gen1 should not have WebSocket")
	}
}

func TestHasSensor(t *testing.T) {
	profile := &Profile{
		Sensors: []SensorType{SensorTemperature, SensorHumidity, SensorBattery},
	}

	if !profile.hasSensor(SensorTemperature) {
		t.Error("Should have Temperature sensor")
	}
	if !profile.hasSensor(SensorHumidity) {
		t.Error("Should have Humidity sensor")
	}
	if !profile.hasSensor(SensorBattery) {
		t.Error("Should have Battery sensor")
	}
	if profile.hasSensor(SensorMotion) {
		t.Error("Should not have Motion sensor")
	}
}

func TestSeriesString(t *testing.T) {
	tests := []struct {
		series Series
		want   string
	}{
		{SeriesClassic, "classic"},
		{SeriesPlus, "plus"},
		{SeriesPro, "pro"},
		{SeriesMini, "mini"},
		{SeriesBLU, "blu"},
		{SeriesWave, "wave"},
		{SeriesWavePro, "wave_pro"},
		{SeriesStandard, "standard"},
	}

	for _, tt := range tests {
		t.Run(string(tt.series), func(t *testing.T) {
			if got := string(tt.series); got != tt.want {
				t.Errorf("Series = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSensorTypes(t *testing.T) {
	sensors := []SensorType{
		SensorTemperature,
		SensorHumidity,
		SensorMotion,
		SensorContact,
		SensorVibration,
		SensorTilt,
		SensorIlluminance,
		SensorFlood,
		SensorSmoke,
		SensorGas,
		SensorBattery,
		SensorVoltage,
		SensorCurrent,
		SensorPower,
		SensorEnergy,
		SensorPowerFactor,
	}

	// Just verify all sensor types are defined
	for _, s := range sensors {
		if s == "" {
			t.Error("Empty sensor type")
		}
	}
}

func TestPowerSources(t *testing.T) {
	sources := []PowerSource{
		PowerSourceMains,
		PowerSourceBattery,
		PowerSourceUSB,
		PowerSourceDC,
		PowerSourcePoE,
	}

	for _, s := range sources {
		if s == "" {
			t.Error("Empty power source")
		}
	}
}

func TestFormFactors(t *testing.T) {
	forms := []FormFactor{
		FormFactorDIN,
		FormFactorFlush,
		FormFactorPlug,
		FormFactorBulb,
		FormFactorSensor,
		FormFactorWallMount,
		FormFactorDesktop,
		FormFactorOutdoor,
		FormFactorRadiator,
	}

	for _, f := range forms {
		if f == "" {
			t.Error("Empty form factor")
		}
	}
}
