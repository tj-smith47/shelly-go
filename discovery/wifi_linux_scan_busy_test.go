//go:build linux

package discovery

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/mdlayher/wifi"
)

// A busy radio (another scan in progress) is waited out, not reported as an
// empty neighborhood.
func TestTriggerScanRetriesWhileBusy(t *testing.T) {
	calls := 0
	s := &platformWiFiScanner{nlScanRetry: time.Millisecond, nlScan: func(context.Context, *wifi.Interface) error {
		calls++
		if calls < 3 {
			return syscall.EBUSY
		}
		return nil
	}}
	if err := s.triggerScan(context.Background(), &wifi.Interface{}); err != nil {
		t.Fatalf("triggerScan: %v", err)
	}
	if calls != 3 {
		t.Fatalf("trigger called %d times, want 3", calls)
	}
}

func TestTriggerScanStopsAtContextEnd(t *testing.T) {
	s := &platformWiFiScanner{
		nlScanRetry: time.Millisecond,
		nlScan:      func(context.Context, *wifi.Interface) error { return syscall.EBUSY },
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := s.triggerScan(ctx, &wifi.Interface{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context deadline", err)
	}
}

func TestTriggerScanReturnsOtherErrorsAtOnce(t *testing.T) {
	calls := 0
	s := &platformWiFiScanner{
		nlScanRetry: time.Millisecond,
		nlScan:      func(context.Context, *wifi.Interface) error { calls++; return os.ErrPermission },
	}
	if err := s.triggerScan(context.Background(), &wifi.Interface{}); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("err = %v, want permission error", err)
	}
	if calls != 1 {
		t.Fatalf("trigger called %d times, want 1", calls)
	}
}
