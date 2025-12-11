package discovery

import (
	"testing"
	"time"
)

// BenchmarkIsShellyAP benchmarks SSID pattern matching.
func BenchmarkIsShellyAP(b *testing.B) {
	ssids := []string{
		"shellyplus1pm-AABBCC",
		"shelly1-123456",
		"ShellyPro4PM-DEADBEEF",
		"HomeNetwork",
		"shellyrgbw2-abc123",
		"OtherSSID",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ssid := ssids[i%len(ssids)]
		_ = IsShellyAP(ssid)
	}
}

// BenchmarkParseShellySSID benchmarks SSID parsing.
func BenchmarkParseShellySSID(b *testing.B) {
	ssids := []string{
		"shellyplus1pm-AABBCC",
		"shelly1-123456",
		"ShellyPro4PM-DEADBEEF",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ssid := ssids[i%len(ssids)]
		_, _ = ParseShellySSID(ssid)
	}
}

// BenchmarkInferGenerationFromModel benchmarks generation inference.
func BenchmarkInferGenerationFromModel(b *testing.B) {
	models := []string{
		"plus1pm",
		"pro4pm",
		"1pm",
		"rgbw2",
		"plus1-g3",
		"pro-gen4",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model := models[i%len(models)]
		_ = InferGenerationFromModel(model)
	}
}

// BenchmarkShellyAPPattern_Match benchmarks regex pattern matching.
func BenchmarkShellyAPPattern_Match(b *testing.B) {
	ssids := []string{
		"shellyplus1pm-AABBCC",
		"shelly1-123456",
		"ShellyPro4PM-DEADBEEF",
		"HomeNetwork",
		"NotAShelly",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ssid := ssids[i%len(ssids)]
		_ = ShellyAPPattern.MatchString(ssid)
	}
}

// BenchmarkMDNS_BuildDNSQuery benchmarks DNS query construction.
func BenchmarkMDNS_BuildDNSQuery(b *testing.B) {
	d := NewMDNSDiscoverer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.buildDNSQuery(MDNSService, 12)
	}
}

// BenchmarkDiscoveredDevice_Create benchmarks device struct creation.
func BenchmarkDiscoveredDevice_Create(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DiscoveredDevice{
			ID:           "shellyplus1pm-aabbcc",
			Name:         "Test Device",
			Model:        "SNSW-001P16EU",
			MACAddress:   "AA:BB:CC:DD:EE:FF",
			Firmware:     "1.0.0",
			AuthRequired: false,
			LastSeen:     time.Now(),
		}
	}
}

// BenchmarkWiFiNetwork_Parse benchmarks WiFi network creation from scan.
func BenchmarkWiFiNetwork_Parse(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		network := WiFiNetwork{
			SSID:     "shellyplus1pm-AABBCC",
			BSSID:    "AA:BB:CC:DD:EE:FF",
			Signal:   -50,
			Channel:  6,
			Security: "WPA2",
			LastSeen: time.Now(),
		}

		if IsShellyAP(network.SSID) {
			network.IsShelly = true
			network.DeviceType, network.DeviceID = ParseShellySSID(network.SSID)
		}
	}
}

// BenchmarkBTHome_ParseData benchmarks BTHome data parsing.
func BenchmarkBTHome_ParseData(b *testing.B) {
	// Sample BTHome v2 data with temperature, humidity, and battery
	data := []byte{
		0x40,       // Device info: version 2, not encrypted
		0x00, 0x01, // Packet ID
		0x01, 0x64, // Battery 100%
		0x02, 0xE8, 0x03, // Temperature 10.00°C
		0x03, 0x88, 0x13, // Humidity 50.00%
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseBTHomeData(data)
	}
}

// BenchmarkWiFiDiscoverer_Create benchmarks discoverer creation.
func BenchmarkWiFiDiscoverer_Create(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewWiFiDiscoverer()
	}
}
