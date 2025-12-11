package transport

import (
	"net/url"
	"testing"
	"time"
)

// BenchmarkHTTPTransport_BuildURL benchmarks URL construction.
func BenchmarkHTTPTransport_BuildURL(b *testing.B) {
	baseURL, _ := url.Parse("http://192.168.1.100")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		u := *baseURL
		u.Path = "/rpc/Shelly.GetDeviceInfo"
		_ = u.String()
	}
}

// BenchmarkOptions_DefaultOptions benchmarks default options creation.
func BenchmarkOptions_DefaultOptions(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = defaultOptions()
	}
}

// BenchmarkWithOptions benchmarks functional option application.
func BenchmarkWithOptions(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opts := defaultOptions()
		WithTimeout(30 * time.Second)(opts)
		WithRetry(3, 1*time.Second)(opts)
	}
}

// BenchmarkAddressParser benchmarks address parsing with port.
func BenchmarkAddressParser(b *testing.B) {
	addresses := []string{
		"192.168.1.100",
		"192.168.1.100:80",
		"shelly.local",
		"shelly.local:8080",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		addr := addresses[i%len(addresses)]
		_, _ = url.Parse("http://" + addr)
	}
}

// BenchmarkURLConstruction benchmarks full URL construction.
func BenchmarkURLConstruction(b *testing.B) {
	methods := []string{
		"Shelly.GetDeviceInfo",
		"Switch.GetStatus",
		"WiFi.GetConfig",
		"Sys.GetStatus",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		method := methods[i%len(methods)]
		u := &url.URL{
			Scheme: "http",
			Host:   "192.168.1.100",
			Path:   "/rpc/" + method,
		}
		_ = u.String()
	}
}
