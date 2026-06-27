// Package confirm provides a generic apply-then-poll-until-converged primitive for
// driving a device to a desired state and proving it actually got there.
//
// Shelly devices occasionally drop a state-change request: a fan-out set returns
// HTTP 200 but the bulb never transitions, or a busy device silently ignores the
// command. A fire-and-forget Set therefore cannot prove the device reached the
// target. [Until] closes that gap — it applies the desired state, polls the device's
// live status until a caller-supplied [Check] reports convergence, and re-applies to
// the same device if it lags, all within a single bounded deadline.
//
// The primitive is deliberately transport- and generation-agnostic: callers supply
// an [Apply] that issues the change and a [Check] that reads live status and reports
// whether the target was reached. Higher-level helpers (see package helpers) wire
// these closures for concrete components such as lights and switches.
package confirm

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Default tuning applied to any [Options] field left at its zero value. The defaults
// target a LAN-attached bulb: confirm within a few hundred milliseconds on the happy
// path, give up by roughly five seconds.
const (
	DefaultTimeout      = 5 * time.Second
	DefaultPollInterval = 250 * time.Millisecond
	DefaultMaxApplies   = 3
)

// ErrNotConverged is returned (wrapped) when a device does not reach the desired
// state before the deadline. Match it with [errors.Is].
var ErrNotConverged = errors.New("shelly: state did not converge before deadline")

// Apply issues the desired state to the device. It is called once up front and again
// on each re-apply attempt.
//
// A terminal error (for example, the device is offline) is returned to the caller
// immediately rather than retried here: the transport already retries transient wire
// failures, and an unreachable device should fail fast so the caller can classify it
// instead of burning the whole deadline.
type Apply func(ctx context.Context) error

// Check reports whether the device has reached the desired state. converged is the
// verdict and observed is a short human-readable description of what was seen, used in
// diagnostics and surfaced on [Result] and in the wrapped [ErrNotConverged].
//
// A non-nil err is treated as transient — the poll loop tolerates it and tries again
// until the deadline — so Check should return (false, "", err) for a status read it
// could not complete rather than giving up.
type Check func(ctx context.Context) (converged bool, observed string, err error)

// Options tunes the apply→poll→re-apply loop. The zero value is valid; each zero field
// falls back to its package default.
type Options struct {
	// Timeout bounds the whole operation across every attempt. Values <= 0 use
	// [DefaultTimeout].
	Timeout time.Duration

	// PollInterval is the delay between live-status polls. Values <= 0 use
	// [DefaultPollInterval].
	PollInterval time.Duration

	// Settle delays the first poll after an Apply so a fast transition can land before
	// it is read, avoiding a guaranteed-miss poll. The zero value polls immediately;
	// negative values are clamped to zero.
	Settle time.Duration

	// MaxApplies caps how many times Apply runs in total — the initial apply plus any
	// re-applies to a lagging device. Values <= 0 use [DefaultMaxApplies]; 1 disables
	// re-apply (apply once, then only poll).
	MaxApplies int
}

func (o Options) withDefaults() Options {
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.PollInterval <= 0 {
		o.PollInterval = DefaultPollInterval
	}
	if o.MaxApplies <= 0 {
		o.MaxApplies = DefaultMaxApplies
	}
	if o.Settle < 0 {
		o.Settle = 0
	}
	return o
}

// Result describes how an [Until] call resolved, for logging and diagnostics. It is
// returned on both success and failure so callers can record the effort spent.
type Result struct {
	Converged bool          // whether the device reached the desired state
	Applies   int           // number of Apply calls made
	Polls     int           // number of Check calls made
	Elapsed   time.Duration // wall-clock spent
	Observed  string        // last non-empty observation from Check
}

// Until applies the desired state and polls until the device converges to it,
// re-applying to the same device if it lags, all within o.Timeout.
//
// It returns a Result with Converged true and a nil error on success. If apply returns
// a terminal error it is returned immediately with Converged false (the device is
// treated as unreachable; the caller classifies it). If the deadline passes without
// convergence it returns an error wrapping [ErrNotConverged] annotated with the last
// observation. Context cancellation is honored throughout and is reported as a
// canceled [ErrNotConverged].
func Until(ctx context.Context, apply Apply, check Check, o Options) (Result, error) {
	o = o.withDefaults()
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()

	// Each apply attempt gets an equal slice of the deadline to converge before we
	// re-issue the command to a lagging device.
	reapplyAfter := o.Timeout / time.Duration(o.MaxApplies)

	var res Result
	doApply := func() error {
		res.Applies++
		return apply(ctx)
	}

	if err := doApply(); err != nil {
		res.Elapsed = time.Since(start)
		return res, err // terminal: fail fast so the caller can classify (e.g. offline)
	}
	attemptStart := time.Now()
	if err := wait(ctx, o.Settle); err != nil {
		res.Elapsed = time.Since(start)
		return res, notConverged(ctx, res)
	}

	for {
		res.Polls++
		converged, observed, checkErr := check(ctx)
		if observed != "" {
			res.Observed = observed
		}
		if checkErr == nil && converged {
			res.Converged = true
			res.Elapsed = time.Since(start)
			return res, nil
		}
		// A non-nil checkErr is transient by contract; keep polling until the deadline.

		if res.Applies < o.MaxApplies && time.Since(attemptStart) >= reapplyAfter {
			if err := doApply(); err != nil {
				res.Elapsed = time.Since(start)
				return res, err
			}
			attemptStart = time.Now()
			if err := wait(ctx, o.Settle); err != nil {
				res.Elapsed = time.Since(start)
				return res, notConverged(ctx, res)
			}
			continue
		}

		if err := wait(ctx, o.PollInterval); err != nil {
			res.Elapsed = time.Since(start)
			return res, notConverged(ctx, res)
		}
	}
}

// wait sleeps for d or until ctx is done, returning ctx.Err() if the context ended
// first. A non-positive d does not sleep and simply reports the current context state.
func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// notConverged builds the terminal failure error, distinguishing an explicit caller
// cancellation from an elapsed deadline while reporting both as [ErrNotConverged].
func notConverged(ctx context.Context, res Result) error {
	observed := res.Observed
	if observed == "" {
		observed = "no successful status read"
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("%w: canceled after %d applies, %d polls (last seen: %s)",
			ErrNotConverged, res.Applies, res.Polls, observed)
	}
	return fmt.Errorf("%w after %d applies, %d polls (last seen: %s)",
		ErrNotConverged, res.Applies, res.Polls, observed)
}
