package discovery

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

func TestIdentify_Gen2Device(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/shelly" {
			http.NotFound(w, r)
			return
		}

		response := gen2ShellyResponse{
			Name:      "Test Device",
			ID:        "shellyplus1-abc123",
			MAC:       "AA:BB:CC:DD:EE:FF",
			Model:     "SNSW-001P16EU",
			Gen:       2,
			FWVersion: "1.0.0-beta1",
			App:       "Plus1",
			AuthEn:    false,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	info, err := Identify(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if info.ID != "shellyplus1-abc123" {
		t.Errorf("ID = %v, want 'shellyplus1-abc123'", info.ID)
	}

	if info.Name != "Test Device" {
		t.Errorf("Name = %v, want 'Test Device'", info.Name)
	}

	if info.Model != "SNSW-001P16EU" {
		t.Errorf("Model = %v, want 'SNSW-001P16EU'", info.Model)
	}

	if info.Generation != types.Gen2 {
		t.Errorf("Generation = %v, want Gen2", info.Generation)
	}

	if info.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MACAddress = %v, want 'AA:BB:CC:DD:EE:FF'", info.MACAddress)
	}
}

func TestIdentify_Gen3Device(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := gen2ShellyResponse{
			Name:      "Gen3 Device",
			ID:        "shelly1gen3-xyz789",
			MAC:       "11:22:33:44:55:66",
			Model:     "S3SW-001P16EU",
			Gen:       3,
			FWVersion: "1.0.0",
			App:       "1Gen3",
			AuthEn:    true,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	info, err := Identify(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if info.Generation != types.Gen3 {
		t.Errorf("Generation = %v, want Gen3", info.Generation)
	}

	if !info.AuthRequired {
		t.Error("AuthRequired should be true")
	}
}

func TestIdentify_Gen1Device(t *testing.T) {
	shellyHandler := func(w http.ResponseWriter, _ *http.Request) {
		response := gen1ShellyResponse{
			Type: "SHSW-1",
			MAC:  "AABBCCDDEEFF",
			Auth: false,
			FW:   "20211109-130952/v1.11.7-g682a0db",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}

	settingsHandler := func(w http.ResponseWriter, _ *http.Request) {
		response := map[string]any{
			"device": map[string]any{
				"hostname": "My Shelly Device",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/shelly":
			shellyHandler(w, r)
		case "/settings":
			settingsHandler(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// First try Gen2, then fall back to Gen1
	info, err := identifyGen1(context.Background(), http.DefaultClient, server.URL)
	if err != nil {
		t.Fatalf("identifyGen1() error = %v", err)
	}

	if info.Model != "SHSW-1" {
		t.Errorf("Model = %v, want 'SHSW-1'", info.Model)
	}

	if info.Generation != types.Gen1 {
		t.Errorf("Generation = %v, want Gen1", info.Generation)
	}

	if info.Name != "My Shelly Device" {
		t.Errorf("Name = %v, want 'My Shelly Device'", info.Name)
	}

	if info.MACAddress != "AABBCCDDEEFF" {
		t.Errorf("MACAddress = %v, want 'AABBCCDDEEFF'", info.MACAddress)
	}
}

func TestIdentify_NotFoundError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	_, err := Identify(context.Background(), server.URL)
	if err == nil {
		t.Error("Identify() should return error for 404")
	}
}

func TestIdentify_NormalizeAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := gen2ShellyResponse{
			ID:  "test",
			Gen: 2,
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Get server port
	addr := server.URL[7:] // Remove "http://"

	// Test without protocol prefix
	_, err := Identify(context.Background(), addr)
	if err != nil {
		t.Errorf("Identify() with bare address error = %v", err)
	}
}

func TestGenerateSubnetAddresses(t *testing.T) {
	addresses := GenerateSubnetAddresses("192.168.1.0/24")

	if len(addresses) != 254 {
		t.Errorf("GenerateSubnetAddresses() returned %d addresses, want 254", len(addresses))
	}

	// Check first and last addresses
	if addresses[0] != "192.168.1.1" {
		t.Errorf("First address = %v, want '192.168.1.1'", addresses[0])
	}

	if addresses[len(addresses)-1] != "192.168.1.254" {
		t.Errorf("Last address = %v, want '192.168.1.254'", addresses[len(addresses)-1])
	}
}

func TestGenerateSubnetAddresses_SmallSubnet(t *testing.T) {
	addresses := GenerateSubnetAddresses("10.0.0.0/30")

	// /30 has 4 addresses: network, 2 hosts, broadcast
	// We skip network (0) and broadcast (3)
	if len(addresses) != 2 {
		t.Errorf("GenerateSubnetAddresses() returned %d addresses, want 2", len(addresses))
	}
}

func TestGenerateSubnetAddresses_InvalidCIDR(t *testing.T) {
	addresses := GenerateSubnetAddresses("invalid")

	if addresses != nil {
		t.Error("GenerateSubnetAddresses() should return nil for invalid CIDR")
	}
}

func TestNextIP(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	next := nextIP(ip)

	expected := net.ParseIP("192.168.1.2")
	if !next.Equal(expected) {
		t.Errorf("nextIP() = %v, want %v", next, expected)
	}
}

func TestNextIP_Overflow(t *testing.T) {
	ip := net.ParseIP("192.168.1.255")
	next := nextIP(ip)

	expected := net.ParseIP("192.168.2.0")
	if !next.Equal(expected) {
		t.Errorf("nextIP() = %v, want %v", next, expected)
	}
}

func TestParseIP(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"192.168.1.100", "192.168.1.100"},
		{"http://192.168.1.100", "192.168.1.100"},
		{"https://192.168.1.100", "192.168.1.100"},
		{"192.168.1.100:80", "192.168.1.100"},
		{"http://192.168.1.100:8080", "192.168.1.100"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseIP(tt.input)
			want := net.ParseIP(tt.want)

			if !got.Equal(want) {
				t.Errorf("parseIP(%q) = %v, want %v", tt.input, got, want)
			}
		})
	}
}

func TestDeviceInfoFields(t *testing.T) {
	info := DeviceInfo{
		ID:           "test-id",
		Name:         "Test Device",
		Model:        "SHSW-1",
		Generation:   types.Gen1,
		Firmware:     "1.0.0",
		MACAddress:   "AA:BB:CC:DD:EE:FF",
		AuthRequired: true,
		App:          "Switch",
		Profile:      "switch",
	}

	if info.ID != "test-id" {
		t.Errorf("ID = %v, want 'test-id'", info.ID)
	}

	if info.App != "Switch" {
		t.Errorf("App = %v, want 'Switch'", info.App)
	}

	if info.Profile != "switch" {
		t.Errorf("Profile = %v, want 'switch'", info.Profile)
	}
}

func TestProbeAddresses(t *testing.T) {
	// Create multiple test servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := gen2ShellyResponse{
			Name:  "Device 1",
			ID:    "device1",
			Gen:   2,
			Model: "SNSW-001P16EU",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := gen2ShellyResponse{
			Name:  "Device 2",
			ID:    "device2",
			Gen:   3,
			Model: "S3SW-001P16EU",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server2.Close()

	// Server that returns 404
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server3.Close()

	addresses := []string{
		server1.URL,
		server2.URL,
		server3.URL,
	}

	ctx := context.Background()
	devices := ProbeAddresses(ctx, addresses)

	// Should find 2 devices (server3 returns 404)
	if len(devices) != 2 {
		t.Errorf("ProbeAddresses() returned %d devices, want 2", len(devices))
	}

	// Verify devices are found
	foundDevice1 := false
	foundDevice2 := false
	for _, d := range devices {
		if d.ID == "device1" {
			foundDevice1 = true
			if d.Generation != types.Gen2 {
				t.Errorf("device1 Generation = %v, want Gen2", d.Generation)
			}
		}
		if d.ID == "device2" {
			foundDevice2 = true
			if d.Generation != types.Gen3 {
				t.Errorf("device2 Generation = %v, want Gen3", d.Generation)
			}
		}
	}

	if !foundDevice1 {
		t.Error("device1 not found")
	}
	if !foundDevice2 {
		t.Error("device2 not found")
	}
}

func TestProbeAddresses_ContextCancellation(t *testing.T) {
	// Server with slow response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		response := gen2ShellyResponse{ID: "slow", Gen: 2}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	addresses := []string{server.URL}
	devices := ProbeAddresses(ctx, addresses)

	// Should timeout and return empty
	if len(devices) != 0 {
		t.Errorf("ProbeAddresses() with canceled context returned %d devices, want 0", len(devices))
	}
}

func TestProbeAddresses_EmptyAddresses(t *testing.T) {
	ctx := context.Background()
	devices := ProbeAddresses(ctx, []string{})

	if len(devices) != 0 {
		t.Errorf("ProbeAddresses() with empty addresses returned %d devices, want 0", len(devices))
	}
}

func TestIdentify_Gen4Device(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := gen2ShellyResponse{
			Name:      "Gen4 Device",
			ID:        "shelly1gen4-xyz",
			MAC:       "AA:BB:CC:DD:EE:FF",
			Model:     "S4SW-001P16EU",
			Gen:       4,
			FWVersion: "1.0.0",
			App:       "1Gen4",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	info, err := Identify(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if info.Generation != types.Gen4 {
		t.Errorf("Generation = %v, want Gen4", info.Generation)
	}
}

func TestIdentify_UnknownGeneration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := gen2ShellyResponse{
			ID:  "test",
			Gen: 99, // Unknown future generation
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	info, err := Identify(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	// Unknown generations should default to Gen2
	if info.Generation != types.Gen2 {
		t.Errorf("Generation = %v, want Gen2 (default)", info.Generation)
	}
}

func TestIdentify_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	_, err := Identify(context.Background(), server.URL)
	if err == nil {
		t.Error("Identify() should return error for invalid JSON")
	}
}

func TestIdentify_HTTPError(t *testing.T) {
	// Non-existent server
	_, err := Identify(context.Background(), "http://192.168.255.255:12345")
	if err == nil {
		t.Error("Identify() should return error for unreachable server")
	}
}

func TestIdentify_Gen1Fallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/shelly":
			// Return Gen1-style response
			response := gen1ShellyResponse{
				Type: "SHSW-1",
				MAC:  "AABBCCDDEEFF",
				Auth: false,
				FW:   "1.11.0",
			}
			_ = json.NewEncoder(w).Encode(response)
		case "/settings":
			response := map[string]any{
				"device": map[string]any{
					"hostname": "Gen1 Device",
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	info, err := Identify(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Identify() error = %v", err)
	}

	if info.Generation != types.Gen1 {
		t.Errorf("Generation = %v, want Gen1", info.Generation)
	}

	if info.Name != "Gen1 Device" {
		t.Errorf("Name = %v, want 'Gen1 Device'", info.Name)
	}
}

func TestGetGen1Name_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/settings" {
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(gen1ShellyResponse{Type: "SHSW-1", MAC: "AABB"})
	}))
	defer server.Close()

	info, err := identifyGen1(context.Background(), http.DefaultClient, server.URL)
	if err != nil {
		t.Fatalf("identifyGen1() error = %v", err)
	}

	// Name should be empty when settings endpoint fails
	if info.Name != "" {
		t.Errorf("Name = %v, want ''", info.Name)
	}
}

func TestGetGen1Name_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/settings" {
			_, _ = w.Write([]byte("invalid json"))
			return
		}
		_ = json.NewEncoder(w).Encode(gen1ShellyResponse{Type: "SHSW-1", MAC: "AABB"})
	}))
	defer server.Close()

	info, err := identifyGen1(context.Background(), http.DefaultClient, server.URL)
	if err != nil {
		t.Fatalf("identifyGen1() error = %v", err)
	}

	if info.Name != "" {
		t.Errorf("Name = %v, want ''", info.Name)
	}
}

func TestGetGen1Name_MissingDeviceField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/settings" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"other_field": "value",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(gen1ShellyResponse{Type: "SHSW-1", MAC: "AABB"})
	}))
	defer server.Close()

	info, err := identifyGen1(context.Background(), http.DefaultClient, server.URL)
	if err != nil {
		t.Fatalf("identifyGen1() error = %v", err)
	}

	if info.Name != "" {
		t.Errorf("Name = %v, want ''", info.Name)
	}
}
