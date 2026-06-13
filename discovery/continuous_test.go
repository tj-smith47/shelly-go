package discovery

import (
	"testing"
	"time"
)

// ────────────────────────────────────────────────────────────────────────────
// CoIoT continuousDiscovery — stop path
// ────────────────────────────────────────────────────────────────────────────

func TestCoIoT_StartStop_ContinuousDiscovery(t *testing.T) {
	d := NewCoIoTDiscoverer()

	ch, err := d.StartDiscovery()
	if err != nil {
		// createMulticastConn may fail in restricted CI environments.
		t.Skipf("StartDiscovery failed (no multicast?): %v", err)
	}
	if ch == nil {
		t.Fatal("StartDiscovery returned nil channel")
	}

	// Verify running state.
	d.mu.RLock()
	running := d.running
	d.mu.RUnlock()
	if !running {
		t.Error("discoverer should be running after StartDiscovery")
	}

	// Second call must return same channel.
	ch2, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("second StartDiscovery: %v", err)
	}
	if ch2 != ch {
		t.Error("second StartDiscovery should return same channel")
	}

	// Stop — must close the stopCh so continuousDiscovery exits.
	if err := d.StopDiscovery(); err != nil {
		t.Fatalf("StopDiscovery: %v", err)
	}

	d.mu.RLock()
	running = d.running
	d.mu.RUnlock()
	if running {
		t.Error("discoverer should not be running after StopDiscovery")
	}

	// Idempotent stop.
	if err := d.StopDiscovery(); err != nil {
		t.Fatalf("second StopDiscovery: %v", err)
	}
}

func TestCoIoT_Stop_NeverStarted(t *testing.T) {
	d := NewCoIoTDiscoverer()
	if err := d.StopDiscovery(); err != nil {
		t.Errorf("StopDiscovery on never-started discoverer: %v", err)
	}
	if err := d.Stop(); err != nil {
		t.Errorf("Stop on never-started discoverer: %v", err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// mDNS continuousDiscovery — stop path
// ────────────────────────────────────────────────────────────────────────────

func TestMDNS_StartStop_ContinuousDiscovery(t *testing.T) {
	d := NewMDNSDiscoverer()

	ch, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("StartDiscovery: %v", err)
	}
	if ch == nil {
		t.Fatal("StartDiscovery returned nil channel")
	}

	// Brief pause to let the goroutine start.
	time.Sleep(10 * time.Millisecond)

	d.mu.RLock()
	running := d.running
	d.mu.RUnlock()
	if !running {
		t.Error("discoverer should be running")
	}

	// Second call returns same channel.
	ch2, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("second StartDiscovery: %v", err)
	}
	if ch2 != ch {
		t.Error("second StartDiscovery should return same channel")
	}

	if err := d.StopDiscovery(); err != nil {
		t.Fatalf("StopDiscovery: %v", err)
	}

	// Allow goroutine to drain.
	time.Sleep(10 * time.Millisecond)

	d.mu.RLock()
	running = d.running
	d.mu.RUnlock()
	if running {
		t.Error("discoverer should not be running after StopDiscovery")
	}

	if err := d.StopDiscovery(); err != nil {
		t.Fatalf("second StopDiscovery: %v", err)
	}

	if err := d.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// WiFiDiscoverer continuousDiscovery — stop path via low-ticker interval
// ────────────────────────────────────────────────────────────────────────────

func TestWiFi_StartStop_ContinuousDiscovery(t *testing.T) {
	mock := &mockWiFiScanner{}
	d := NewWiFiDiscovererWithScanner(mock)

	ch, err := d.StartDiscovery()
	if err != nil {
		t.Fatalf("StartDiscovery: %v", err)
	}
	if ch == nil {
		t.Fatal("StartDiscovery returned nil channel")
	}

	time.Sleep(10 * time.Millisecond)

	d.mu.RLock()
	running := d.running
	d.mu.RUnlock()
	if !running {
		t.Error("discoverer should be running")
	}

	if err := d.StopDiscovery(); err != nil {
		t.Fatalf("StopDiscovery: %v", err)
	}

	d.mu.RLock()
	running = d.running
	d.mu.RUnlock()
	if running {
		t.Error("discoverer should not be running")
	}

	if err := d.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Scanner.Stop — with wifi discoverer having an active goroutine
// ────────────────────────────────────────────────────────────────────────────

func TestScanner_Stop_WithActiveDiscoverers(t *testing.T) {
	mock := &mockWiFiScanner{}
	bleMock := newMockBLEScanner()
	s := NewScanner(WithWiFi(mock), WithBLE(bleMock))

	// Start the wifi discoverer's continuous loop.
	_, err := s.wifi.StartDiscovery()
	if err != nil {
		t.Fatalf("wifi.StartDiscovery: %v", err)
	}

	_, err = s.ble.StartDiscovery()
	if err != nil {
		t.Fatalf("ble.StartDiscovery: %v", err)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("Scanner.Stop: %v", err)
	}
}
