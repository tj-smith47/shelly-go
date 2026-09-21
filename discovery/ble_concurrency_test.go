package discovery

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"tinygo.org/x/bluetooth"
)

const bleTestTimeout = 5 * time.Second

// waitBLE fails the test if ch is not closed or sent to within bleTestTimeout.
func waitBLE[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(bleTestTimeout):
		t.Fatalf("timed out waiting for %s", what)
		panic("unreachable")
	}
}

// fakeScanAdapter stands in for *bluetooth.Adapter. Each Scan call blocks
// until the test releases it, so a test decides when an old scan ends.
type fakeScanAdapter struct {
	scans     chan *fakeScan
	stopCalls atomic.Int32
}

type fakeScan struct {
	callback func(*bluetooth.Adapter, bluetooth.ScanResult)
	release  chan error
}

func newFakeScanAdapter() *fakeScanAdapter {
	return &fakeScanAdapter{scans: make(chan *fakeScan, 4)}
}

func (a *fakeScanAdapter) Scan(callback func(*bluetooth.Adapter, bluetooth.ScanResult)) error {
	scan := &fakeScan{callback: callback, release: make(chan error)}
	a.scans <- scan
	return <-scan.release
}

func (a *fakeScanAdapter) StopScan() error {
	a.stopCalls.Add(1)
	return nil
}

// fakeAdvPayload supplies the two payload methods convertAdvertisement reads.
type fakeAdvPayload struct {
	bluetooth.AdvertisementPayload
}

func (fakeAdvPayload) LocalName() string { return "SHELLY-TEST-0001" }

func (fakeAdvPayload) ManufacturerData() []bluetooth.ManufacturerDataElement { return nil }

func (s *tinyGoBLEScanner) isRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func TestTinyGoBLEScanner_RestartIsNotAffectedByOldScan(t *testing.T) {
	adapter := newFakeScanAdapter()
	s := &tinyGoBLEScanner{adapter: adapter}

	// The callbacks count without locking: each scan's callback is only ever
	// called from the test goroutine, and the race detector checks that.
	var firstSeen, secondSeen int

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- s.Start(context.Background(), func(*BLEAdvertisement) { firstSeen++ })
	}()
	first := waitBLE(t, adapter.scans, "first scan to begin")

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := waitBLE(t, firstDone, "first Start to return"); err != nil {
		t.Fatalf("first Start() error = %v", err)
	}

	secondDone := make(chan error, 1)
	go func() {
		secondDone <- s.Start(context.Background(), func(*BLEAdvertisement) { secondSeen++ })
	}()
	second := waitBLE(t, adapter.scans, "second scan to begin")

	// The adapter reports one more result for the stopped scan, then that
	// scan ends with an error. Neither may disturb the second scan.
	result := bluetooth.ScanResult{AdvertisementPayload: fakeAdvPayload{}}
	first.callback(nil, result)
	first.release <- errors.New("scan one ended")

	second.callback(nil, result)

	if !s.isRunning() {
		t.Fatal("the old scan ending cleared running while the second scan is active")
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("second Stop() error = %v", err)
	}
	if err := waitBLE(t, secondDone, "second Start to return after Stop"); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	close(second.release)

	if got := adapter.stopCalls.Load(); got != 2 {
		t.Errorf("StopScan called %d times, want 2", got)
	}
	if firstSeen != 0 {
		t.Errorf("stopped scan delivered %d advertisements, want 0", firstSeen)
	}
	if secondSeen != 1 {
		t.Errorf("second scan delivered %d advertisements, want 1", secondSeen)
	}
}

func TestTinyGoBLEScanner_StaleWaitLeavesCurrentScanAlone(t *testing.T) {
	tests := []struct {
		name string
		wait func(s *tinyGoBLEScanner, ctx context.Context, stale chan struct{}, errCh chan error) error
	}{
		{"waitForCompletion", func(s *tinyGoBLEScanner, ctx context.Context, stale chan struct{}, errCh chan error) error {
			return s.waitForCompletion(ctx, stale, errCh)
		}},
		{"waitForScanStart", func(s *tinyGoBLEScanner, ctx context.Context, stale chan struct{}, errCh chan error) error {
			return s.waitForScanStart(ctx, stale, make(chan struct{}), errCh)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/scan error", func(t *testing.T) {
			current := make(chan struct{})
			s := &tinyGoBLEScanner{running: true, stopCh: current}
			errCh := make(chan error, 1)
			errCh <- errors.New("old scan ended")

			if err := tt.wait(s, context.Background(), make(chan struct{}), errCh); err == nil {
				t.Error("expected the old scan's error")
			}
			assertScanStillCurrent(t, s, current)
		})

		t.Run(tt.name+"/context done", func(t *testing.T) {
			current := make(chan struct{})
			s := &tinyGoBLEScanner{running: true, stopCh: current}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			if err := tt.wait(s, ctx, make(chan struct{}), make(chan error)); err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("unexpected error %v", err)
			}
			assertScanStillCurrent(t, s, current)
		})
	}
}

func assertScanStillCurrent(t *testing.T, s *tinyGoBLEScanner, current chan struct{}) {
	t.Helper()
	if !s.isRunning() {
		t.Fatal("a wait for an old scan cleared running for the current scan")
	}
	select {
	case <-current:
		t.Fatal("a wait for an old scan stopped the current scan")
	default:
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	select {
	case <-current:
	default:
		t.Error("Stop() did not stop the current scan")
	}
}

func TestTinyGoBLEScanner_ScanEndedThenStopClosesOnce(t *testing.T) {
	stop := make(chan struct{})
	s := &tinyGoBLEScanner{running: true, stopCh: stop}

	s.scanEnded(stop)
	s.scanEnded(stop)
	if err := s.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if s.isRunning() {
		t.Error("running should be false after the scan ended")
	}
}

// callbackOnStopScanner delivers one advertisement from inside Stop and waits
// for the callback to return, as a scanner does when its Stop waits for
// callbacks that are still running.
type callbackOnStopScanner struct {
	started  chan struct{}
	mu       sync.Mutex
	callback func(*BLEAdvertisement)
}

func (s *callbackOnStopScanner) Start(ctx context.Context, callback func(*BLEAdvertisement)) error {
	s.mu.Lock()
	s.callback = callback
	s.mu.Unlock()
	select {
	case s.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil
}

func (s *callbackOnStopScanner) Stop() error {
	s.mu.Lock()
	callback := s.callback
	s.mu.Unlock()

	delivered := make(chan struct{})
	go func() {
		defer close(delivered)
		callback(&BLEAdvertisement{Address: "AA:BB:CC:DD:EE:01", LocalName: "SHELLY-PLUS1-ABC"})
	}()
	<-delivered
	return nil
}

func TestBLEDiscoverer_StopDiscoveryDoesNotHoldLockDuringScannerStop(t *testing.T) {
	scanner := &callbackOnStopScanner{started: make(chan struct{}, 1)}
	d := NewBLEDiscovererWithScanner(scanner)

	if _, err := d.StartDiscovery(); err != nil {
		t.Fatalf("StartDiscovery() error = %v", err)
	}
	waitBLE(t, scanner.started, "scan to begin")

	stopped := make(chan error, 1)
	go func() { stopped <- d.StopDiscovery() }()
	if err := waitBLE(t, stopped, "StopDiscovery to return"); err != nil {
		t.Fatalf("StopDiscovery() error = %v", err)
	}

	if d.DeviceByAddress("AA:BB:CC:DD:EE:01") == nil {
		t.Error("advertisement delivered during Stop was not recorded")
	}
}

// inertStopScanner ends a scan only when its context does; Stop does nothing.
type inertStopScanner struct {
	started  chan struct{}
	returned chan struct{}
}

func (s *inertStopScanner) Start(ctx context.Context, _ func(*BLEAdvertisement)) error {
	s.started <- struct{}{}
	<-ctx.Done()
	s.returned <- struct{}{}
	return nil
}

func (s *inertStopScanner) Stop() error { return nil }

func TestBLEDiscoverer_StopDiscoveryEndsScanInProgress(t *testing.T) {
	scanner := &inertStopScanner{started: make(chan struct{}, 1), returned: make(chan struct{}, 1)}
	d := NewBLEDiscovererWithScanner(scanner)
	d.ScanDuration = time.Hour

	if _, err := d.StartDiscovery(); err != nil {
		t.Fatalf("StartDiscovery() error = %v", err)
	}
	waitBLE(t, scanner.started, "scan to begin")

	if err := d.StopDiscovery(); err != nil {
		t.Fatalf("StopDiscovery() error = %v", err)
	}
	waitBLE(t, scanner.returned, "scan to end after StopDiscovery")
}

// instantScanner returns from Start at once, with or without an error.
type instantScanner struct {
	err    error
	starts atomic.Int32
	first  chan struct{}
	once   sync.Once
}

func (s *instantScanner) Start(context.Context, func(*BLEAdvertisement)) error {
	s.starts.Add(1)
	s.once.Do(func() { close(s.first) })
	return s.err
}

func (s *instantScanner) Stop() error { return nil }

func TestBLEDiscoverer_ContinuousDiscoveryWaitsBetweenScansThatEndAtOnce(t *testing.T) {
	for name, scanErr := range map[string]error{
		"start fails":           errors.New("bluetooth unavailable"),
		"start returns at once": nil,
	} {
		t.Run(name, func(t *testing.T) {
			scanner := &instantScanner{err: scanErr, first: make(chan struct{})}
			d := NewBLEDiscovererWithScanner(scanner)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			exited := make(chan struct{})
			go func() {
				defer close(exited)
				d.continuousDiscovery(ctx, scanner, time.Hour, time.Hour)
			}()

			waitBLE(t, scanner.first, "first scan")
			// With an hour between scans a second one can only mean the loop
			// did not wait. Yield so a spinning loop gets the chance to run.
			for range 200 {
				time.Sleep(time.Microsecond)
			}
			if got := scanner.starts.Load(); got != 1 {
				t.Errorf("Start called %d times, want 1 while waiting for the next scan", got)
			}

			cancel()
			waitBLE(t, exited, "loop to end while waiting for the next scan")
		})
	}
}

func TestBLEDiscoverer_StartDiscoveryUsesRescanDelay(t *testing.T) {
	scanner := &instantScanner{err: errors.New("bluetooth unavailable"), first: make(chan struct{})}
	d := NewBLEDiscovererWithScanner(scanner)
	d.rescanDelay = time.Millisecond

	if _, err := d.StartDiscovery(); err != nil {
		t.Fatalf("StartDiscovery() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.StopDiscovery(); err != nil {
			t.Errorf("StopDiscovery() error = %v", err)
		}
	})

	deadline := time.After(bleTestTimeout)
	for scanner.starts.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("Start called %d times, want the loop to keep retrying", scanner.starts.Load())
		case <-time.After(time.Millisecond):
		}
	}
}

// fakeBLEDialer blocks in Connect until the test releases it.
type fakeBLEDialer struct {
	entered chan struct{}
	release chan error
}

func (f *fakeBLEDialer) Connect(bluetooth.Address, bluetooth.ConnectionParams) (bluetooth.Device, error) {
	f.entered <- struct{}{}
	return bluetooth.Device{}, <-f.release
}

func newFakeBLEDialer() *fakeBLEDialer {
	return &fakeBLEDialer{entered: make(chan struct{}, 1), release: make(chan error, 1)}
}

func TestTinyGoBLEConnector_LateConnectionIsDropped(t *testing.T) {
	dialer := newFakeBLEDialer()
	dropped := make(chan struct{}, 1)
	c := &tinyGoBLEConnector{
		adapter: dialer,
		dropDevice: func(bluetooth.Device) error {
			dropped <- struct{}{}
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- c.Connect(ctx, "AA:BB:CC:DD:EE:FF") }()
	waitBLE(t, dialer.entered, "dial to begin")

	cancel()
	if err := waitBLE(t, result, "Connect to return"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Connect() error = %v, want context.Canceled", err)
	}

	// The adapter connects after Connect gave up.
	dialer.release <- nil
	waitBLE(t, dropped, "late connection to be dropped")

	// Read without the lock on purpose: nothing may write this field once
	// Connect has returned, and the race detector checks that.
	if c.connected {
		t.Error("a connection that arrived after Connect returned marked the connector connected")
	}
}

func TestTinyGoBLEConnector_LateFailureIsNotDropped(t *testing.T) {
	dialer := newFakeBLEDialer()
	var drops atomic.Int32
	c := &tinyGoBLEConnector{
		adapter:    dialer,
		dropDevice: func(bluetooth.Device) error { drops.Add(1); return nil },
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Connect(ctx, "AA:BB:CC:DD:EE:FF"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Connect() error = %v, want context.Canceled", err)
	}
	waitBLE(t, dialer.entered, "dial to begin")
	dialer.release <- errors.New("device out of range")

	// A second Connect can only get the lock and dial once the first is over.
	dialer.release <- nil
	if err := c.Connect(context.Background(), "AA:BB:CC:DD:EE:FF"); err != nil {
		t.Fatalf("second Connect() error = %v", err)
	}
	if !c.IsConnected() {
		t.Error("IsConnected() = false after a successful Connect")
	}
	if err := c.Disconnect(); err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}
	if got := drops.Load(); got != 1 {
		t.Errorf("device dropped %d times, want 1 (from Disconnect only)", got)
	}
}

func TestTinyGoBLEConnector_ConnectError(t *testing.T) {
	dialer := newFakeBLEDialer()
	dialer.release <- errors.New("device out of range")
	c := &tinyGoBLEConnector{adapter: dialer}

	err := c.Connect(context.Background(), "AA:BB:CC:DD:EE:FF")
	var bleErr *BLEError
	if !errors.As(err, &bleErr) {
		t.Fatalf("Connect() error = %v, want *BLEError", err)
	}
	if c.IsConnected() {
		t.Error("IsConnected() = true after a failed Connect")
	}
}
