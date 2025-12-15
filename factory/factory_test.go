package factory

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
	"github.com/tj-smith47/shelly-go/types"
)

func TestFromAddress_Gen2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"name":  "Test Device",
			"id":    "shellyplus1-abc123",
			"mac":   "AA:BB:CC:DD:EE:FF",
			"model": "SNSW-001P16EU",
			"gen":   2,
			"fw_id": "1.0.0",
			"app":   "Plus1",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	device, err := FromAddress(server.URL)
	if err != nil {
		t.Fatalf("FromAddress() error = %v", err)
	}

	if device == nil {
		t.Fatal("device should not be nil")
	}

	if device.Generation() != types.Gen2 {
		t.Errorf("Generation() = %v, want Gen2", device.Generation())
	}

	gen2Device, ok := device.(*Gen2Device)
	if !ok {
		t.Fatal("device should be *Gen2Device")
	}

	if gen2Device.addr == "" {
		t.Error("address should not be empty")
	}
}

func TestFromAddress_Gen3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":  "shelly1gen3-xyz",
			"gen": 3,
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	device, err := FromAddress(server.URL)
	if err != nil {
		t.Fatalf("FromAddress() error = %v", err)
	}

	if device.Generation() != types.Gen3 {
		t.Errorf("Generation() = %v, want Gen3", device.Generation())
	}
}

func TestFromAddress_Gen4(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":  "shelly1gen4-xyz",
			"gen": 4,
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	device, err := FromAddress(server.URL)
	if err != nil {
		t.Fatalf("FromAddress() error = %v", err)
	}

	if device.Generation() != types.Gen4 {
		t.Errorf("Generation() = %v, want Gen4", device.Generation())
	}
}

func TestFromAddress_Gen1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Gen1 /shelly response (no "gen" field)
		response := map[string]any{
			"type": "SHSW-1",
			"mac":  "AABBCCDDEEFF",
			"auth": false,
			"fw":   "1.11.0",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	device, err := FromAddress(server.URL)
	if err != nil {
		t.Fatalf("FromAddress() error = %v", err)
	}

	if device.Generation() != types.Gen1 {
		t.Errorf("Generation() = %v, want Gen1", device.Generation())
	}

	gen1Device, ok := device.(*Gen1Device)
	if !ok {
		t.Fatal("device should be *Gen1Device")
	}

	if gen1Device.addr == "" {
		t.Error("address should not be empty")
	}
}

func TestFromAddress_WithGeneration(t *testing.T) {
	// No server needed - generation is forced
	device, err := FromAddress("192.168.1.100",
		WithGeneration(types.Gen2))
	if err != nil {
		t.Fatalf("FromAddress() error = %v", err)
	}

	if device.Generation() != types.Gen2 {
		t.Errorf("Generation() = %v, want Gen2", device.Generation())
	}
}

func TestFromAddress_WithAuth(t *testing.T) {
	device, err := FromAddress("192.168.1.100",
		WithGeneration(types.Gen2),
		WithAuth("admin", "password"))
	if err != nil {
		t.Fatalf("FromAddress() error = %v", err)
	}

	if device == nil {
		t.Error("device should not be nil")
	}
}

func TestFromAddress_WithTimeout(t *testing.T) {
	device, err := FromAddress("192.168.1.100",
		WithGeneration(types.Gen1),
		WithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("FromAddress() error = %v", err)
	}

	if device == nil {
		t.Error("device should not be nil")
	}
}

func TestFromAddress_WithHTTPClient(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}

	device, err := FromAddress("192.168.1.100",
		WithGeneration(types.Gen1),
		WithHTTPClient(client))
	if err != nil {
		t.Fatalf("FromAddress() error = %v", err)
	}

	if device == nil {
		t.Error("device should not be nil")
	}
}

func TestFromAddress_WithContext(t *testing.T) {
	ctx := context.Background()

	device, err := FromAddress("192.168.1.100",
		WithGeneration(types.Gen2),
		WithContext(ctx))
	if err != nil {
		t.Fatalf("FromAddress() error = %v", err)
	}

	if device == nil {
		t.Error("device should not be nil")
	}
}

func TestFromAddress_NormalizeAddress(t *testing.T) {
	tests := []string{
		"192.168.1.100",
		"http://192.168.1.100",
		"http://192.168.1.100/",
	}

	for _, addr := range tests {
		device, err := FromAddress(addr, WithGeneration(types.Gen2))
		if err != nil {
			t.Errorf("FromAddress(%q) error = %v", addr, err)
			continue
		}

		if device == nil {
			t.Errorf("FromAddress(%q) returned nil device", addr)
		}
	}
}

func TestFromAddress_DetectionFailed(t *testing.T) {
	// Unreachable address
	_, err := FromAddress("192.168.255.255:12345",
		WithTimeout(100*time.Millisecond))
	if err == nil {
		t.Error("FromAddress() should return error for unreachable address")
	}
}

func TestFromDiscovery(t *testing.T) {
	d := &discovery.DiscoveredDevice{
		ID:         "test-device",
		Address:    net.ParseIP("192.168.1.100"),
		Port:       80,
		Generation: types.Gen2,
	}

	device, err := FromDiscovery(d)
	if err != nil {
		t.Fatalf("FromDiscovery() error = %v", err)
	}

	if device == nil {
		t.Fatal("device should not be nil")
	}

	if device.Generation() != types.Gen2 {
		t.Errorf("Generation() = %v, want Gen2", device.Generation())
	}
}

func TestFromDiscovery_Gen1(t *testing.T) {
	d := &discovery.DiscoveredDevice{
		ID:         "test-device",
		Address:    net.ParseIP("192.168.1.100"),
		Port:       80,
		Generation: types.Gen1,
	}

	device, err := FromDiscovery(d)
	if err != nil {
		t.Fatalf("FromDiscovery() error = %v", err)
	}

	if device.Generation() != types.Gen1 {
		t.Errorf("Generation() = %v, want Gen1", device.Generation())
	}
}

func TestFromDiscovery_WithAuth(t *testing.T) {
	d := &discovery.DiscoveredDevice{
		ID:           "test-device",
		Address:      net.ParseIP("192.168.1.100"),
		Port:         80,
		Generation:   types.Gen2,
		AuthRequired: true,
	}

	device, err := FromDiscovery(d, WithAuth("admin", "password"))
	if err != nil {
		t.Fatalf("FromDiscovery() error = %v", err)
	}

	if device == nil {
		t.Error("device should not be nil")
	}
}

func TestFromInfo(t *testing.T) {
	info := &discovery.DeviceInfo{
		ID:         "test-device",
		Generation: types.Gen2,
	}

	device, err := FromInfo(info, "192.168.1.100")
	if err != nil {
		t.Fatalf("FromInfo() error = %v", err)
	}

	if device == nil {
		t.Fatal("device should not be nil")
	}

	if device.Generation() != types.Gen2 {
		t.Errorf("Generation() = %v, want Gen2", device.Generation())
	}
}

func TestFromInfo_Gen1(t *testing.T) {
	info := &discovery.DeviceInfo{
		ID:         "test-device",
		Generation: types.Gen1,
	}

	device, err := FromInfo(info, "192.168.1.100")
	if err != nil {
		t.Fatalf("FromInfo() error = %v", err)
	}

	if device.Generation() != types.Gen1 {
		t.Errorf("Generation() = %v, want Gen1", device.Generation())
	}
}

func TestFromInfo_NormalizeAddress(t *testing.T) {
	info := &discovery.DeviceInfo{
		ID:         "test-device",
		Generation: types.Gen2,
	}

	device, err := FromInfo(info, "192.168.1.100")
	if err != nil {
		t.Fatalf("FromInfo() error = %v", err)
	}

	if !hasPrefix(device.Address(), "http://") {
		t.Errorf("Address() = %v, should start with 'http://'", device.Address())
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func TestMustFromAddress(t *testing.T) {
	device := MustFromAddress("192.168.1.100", WithGeneration(types.Gen2))

	if device == nil {
		t.Error("device should not be nil")
	}
}

func TestMustFromAddress_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustFromAddress should panic on error")
		}
	}()

	// This will try to detect generation and fail
	MustFromAddress("192.168.255.255:12345", WithTimeout(100*time.Millisecond))
}

func TestMustFromDiscovery(t *testing.T) {
	d := &discovery.DiscoveredDevice{
		ID:         "test-device",
		Address:    net.ParseIP("192.168.1.100"),
		Port:       80,
		Generation: types.Gen2,
	}

	device := MustFromDiscovery(d)

	if device == nil {
		t.Error("device should not be nil")
	}
}

func TestMustFromDiscovery_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustFromDiscovery should panic on error")
		}
	}()

	d := &discovery.DiscoveredDevice{
		Generation: 0, // Unknown generation
	}

	MustFromDiscovery(d)
}

func TestBatchFromAddresses(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "dev1", "gen": 2})
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "dev2", "gen": 3})
	}))
	defer server2.Close()

	addresses := []string{server1.URL, server2.URL}

	devices, errs := BatchFromAddresses(addresses)

	if len(devices) != 2 {
		t.Errorf("BatchFromAddresses() returned %d devices, want 2", len(devices))
	}

	if len(errs) != 2 {
		t.Errorf("BatchFromAddresses() returned %d errors, want 2", len(errs))
	}

	for i, err := range errs {
		if err != nil {
			t.Errorf("errs[%d] = %v, want nil", i, err)
		}
	}

	for i, dev := range devices {
		if dev == nil {
			t.Errorf("devices[%d] should not be nil", i)
		}
	}
}

func TestBatchFromAddresses_PartialFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "dev1", "gen": 2})
	}))
	defer server.Close()

	addresses := []string{
		server.URL,
		"http://192.168.255.255:12345", // Will fail
	}

	devices, errs := BatchFromAddresses(addresses, WithTimeout(100*time.Millisecond))

	if len(devices) != 2 {
		t.Errorf("BatchFromAddresses() returned %d devices, want 2", len(devices))
	}

	// First should succeed
	if errs[0] != nil {
		t.Errorf("errs[0] = %v, want nil", errs[0])
	}
	if devices[0] == nil {
		t.Error("devices[0] should not be nil")
	}

	// Second should fail
	if errs[1] == nil {
		t.Error("errs[1] should not be nil")
	}
	if devices[1] != nil {
		t.Error("devices[1] should be nil")
	}
}

func TestBatchFromDiscovery(t *testing.T) {
	discovered := []discovery.DiscoveredDevice{
		{ID: "dev1", Address: net.ParseIP("192.168.1.100"), Port: 80, Generation: types.Gen2},
		{ID: "dev2", Address: net.ParseIP("192.168.1.101"), Port: 80, Generation: types.Gen1},
	}

	devices, errs := BatchFromDiscovery(discovered)

	if len(devices) != 2 {
		t.Errorf("BatchFromDiscovery() returned %d devices, want 2", len(devices))
	}

	for i, err := range errs {
		if err != nil {
			t.Errorf("errs[%d] = %v, want nil", i, err)
		}
	}

	if devices[0].Generation() != types.Gen2 {
		t.Errorf("devices[0].Generation() = %v, want Gen2", devices[0].Generation())
	}

	if devices[1].Generation() != types.Gen1 {
		t.Errorf("devices[1].Generation() = %v, want Gen1", devices[1].Generation())
	}
}

func TestCreateDevice_UnknownGeneration(t *testing.T) {
	_, err := createDevice("http://192.168.1.100", 0, &Options{})
	if err != ErrUnknownGeneration {
		t.Errorf("createDevice() error = %v, want ErrUnknownGeneration", err)
	}
}

func TestGen1Device_Methods(t *testing.T) {
	device := &Gen1Device{
		addr:       "http://192.168.1.100",
		generation: types.Gen1,
	}

	if device.Address() != "http://192.168.1.100" {
		t.Errorf("Address() = %v, want 'http://192.168.1.100'", device.Address())
	}

	if device.Generation() != types.Gen1 {
		t.Errorf("Generation() = %v, want Gen1", device.Generation())
	}
}

func TestGen2Device_Methods(t *testing.T) {
	device := &Gen2Device{
		addr:       "http://192.168.1.100",
		generation: types.Gen3,
	}

	if device.Address() != "http://192.168.1.100" {
		t.Errorf("Address() = %v, want 'http://192.168.1.100'", device.Address())
	}

	if device.Generation() != types.Gen3 {
		t.Errorf("Generation() = %v, want Gen3", device.Generation())
	}
}

