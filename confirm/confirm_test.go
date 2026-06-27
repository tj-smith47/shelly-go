package confirm

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// fastOpts keeps the loop snappy so tests finish in well under a second while still
// exercising the re-apply window (reapplyAfter = Timeout/MaxApplies = 40ms here).
func fastOpts() Options {
	return Options{Timeout: 120 * time.Millisecond, PollInterval: 3 * time.Millisecond, MaxApplies: 3}
}

func TestUntil_ConvergesOnFirstPoll(t *testing.T) {
	var applies int32
	apply := func(context.Context) error { atomic.AddInt32(&applies, 1); return nil }
	check := func(context.Context) (bool, string, error) { return true, "on=true brightness=20", nil }

	res, err := Until(context.Background(), apply, check, fastOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Converged {
		t.Fatal("expected converged")
	}
	if got := atomic.LoadInt32(&applies); got != 1 {
		t.Fatalf("expected exactly 1 apply, got %d", got)
	}
	if res.Observed != "on=true brightness=20" {
		t.Fatalf("observed not recorded: %q", res.Observed)
	}
}

func TestUntil_ConvergesAfterReapply(t *testing.T) {
	var applies int32
	apply := func(context.Context) error { atomic.AddInt32(&applies, 1); return nil }
	// Only converges once the command has been applied at least twice, forcing the
	// re-apply path to run.
	check := func(context.Context) (bool, string, error) {
		return atomic.LoadInt32(&applies) >= 2, "stale", nil
	}

	res, err := Until(context.Background(), apply, check, fastOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Converged {
		t.Fatal("expected converged after re-apply")
	}
	if got := atomic.LoadInt32(&applies); got < 2 {
		t.Fatalf("expected >= 2 applies, got %d", got)
	}
}

func TestUntil_NeverConverges(t *testing.T) {
	var applies int32
	apply := func(context.Context) error { atomic.AddInt32(&applies, 1); return nil }
	check := func(context.Context) (bool, string, error) { return false, "brightness=100", nil }

	res, err := Until(context.Background(), apply, check, fastOpts())
	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("expected ErrNotConverged, got %v", err)
	}
	if res.Converged {
		t.Fatal("must not report converged")
	}
	if got := atomic.LoadInt32(&applies); got != 3 {
		t.Fatalf("expected MaxApplies (3) applies, got %d", got)
	}
	if res.Observed != "brightness=100" {
		t.Fatalf("expected last observation surfaced, got %q", res.Observed)
	}
}

func TestUntil_TerminalApplyErrorFailsFast(t *testing.T) {
	sentinel := errors.New("device offline")
	var applies, checks int32
	apply := func(context.Context) error { atomic.AddInt32(&applies, 1); return sentinel }
	check := func(context.Context) (bool, string, error) { atomic.AddInt32(&checks, 1); return false, "", nil }

	start := time.Now()
	_, err := Until(context.Background(), apply, check, fastOpts())
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel apply error, got %v", err)
	}
	if errors.Is(err, ErrNotConverged) {
		t.Fatal("a terminal apply error must not be wrapped as ErrNotConverged")
	}
	if got := atomic.LoadInt32(&applies); got != 1 {
		t.Fatalf("expected fail-fast after 1 apply, got %d", got)
	}
	if got := atomic.LoadInt32(&checks); got != 0 {
		t.Fatalf("check must not run when apply fails, ran %d", got)
	}
	if elapsed := time.Since(start); elapsed > 60*time.Millisecond {
		t.Fatalf("fail-fast took too long: %v", elapsed)
	}
}

func TestUntil_TransientCheckErrorTolerated(t *testing.T) {
	var checks int32
	apply := func(context.Context) error { return nil }
	// First two status reads error (transient), then it converges.
	check := func(context.Context) (bool, string, error) {
		if atomic.AddInt32(&checks, 1) < 3 {
			return false, "", errors.New("read timeout")
		}
		return true, "on=true", nil
	}

	res, err := Until(context.Background(), apply, check, fastOpts())
	if err != nil {
		t.Fatalf("transient check errors should be tolerated, got %v", err)
	}
	if !res.Converged {
		t.Fatal("expected eventual convergence")
	}
}

func TestUntil_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	apply := func(context.Context) error { return nil }
	check := func(context.Context) (bool, string, error) { return false, "pending", nil }

	go func() {
		time.Sleep(15 * time.Millisecond)
		cancel()
	}()

	_, err := Until(ctx, apply, check, Options{Timeout: 5 * time.Second, PollInterval: 3 * time.Millisecond})
	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("expected ErrNotConverged on cancel, got %v", err)
	}
}

func TestOptions_Defaults(t *testing.T) {
	o := Options{}.withDefaults()
	if o.Timeout != DefaultTimeout || o.PollInterval != DefaultPollInterval || o.MaxApplies != DefaultMaxApplies {
		t.Fatalf("zero Options did not pick up defaults: %+v", o)
	}
	if o.Settle != 0 {
		t.Fatalf("default Settle should be 0, got %v", o.Settle)
	}
	if got := (Options{Settle: -1}).withDefaults().Settle; got != 0 {
		t.Fatalf("negative Settle should clamp to 0, got %v", got)
	}
}

func TestUntil_MaxAppliesOneDisablesReapply(t *testing.T) {
	var applies int32
	apply := func(context.Context) error { atomic.AddInt32(&applies, 1); return nil }
	check := func(context.Context) (bool, string, error) { return false, "x", nil }

	_, err := Until(context.Background(), apply, check,
		Options{Timeout: 40 * time.Millisecond, PollInterval: 3 * time.Millisecond, MaxApplies: 1})
	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("expected ErrNotConverged, got %v", err)
	}
	if got := atomic.LoadInt32(&applies); got != 1 {
		t.Fatalf("MaxApplies=1 must apply exactly once, got %d", got)
	}
}
