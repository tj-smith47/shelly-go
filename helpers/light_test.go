package helpers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/confirm"
	"github.com/tj-smith47/shelly-go/factory"
)

// fastOpts returns a confirm.Options tuned for fast, non-flaky unit tests.
// Timeout is short enough to make non-converge cases exit quickly; PollInterval
// and MaxApplies are small enough to stay deterministic in a CI environment.
func fastOpts() confirm.Options {
	return confirm.Options{
		Timeout:      150 * time.Millisecond,
		PollInterval: 3 * time.Millisecond,
		MaxApplies:   3,
	}
}

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }

// TestSetLightConfirmed_Gen1Converges verifies that Gen1 light convergence
// succeeds when the device immediately reflects the target in its status.
func TestSetLightConfirmed_Gen1Converges(t *testing.T) {
	// Handler returns the target state for every request (status and set commands).
	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		mustWrite(w, []byte(`{"ison":true,"brightness":20}`))
	})
	defer server.Close()

	target := LightTarget{On: boolPtr(true), Brightness: intPtr(20)}
	res, err := SetLightConfirmed(context.Background(), dev, 0, target, fastOpts())

	if err != nil {
		t.Fatalf("SetLightConfirmed() error = %v, want nil", err)
	}
	if !res.Converged {
		t.Errorf("Result.Converged = false, want true")
	}
	if res.Applies != 1 {
		t.Errorf("Result.Applies = %d, want 1", res.Applies)
	}
}

// TestSetLightConfirmed_Gen2Converges verifies that Gen2 light convergence
// succeeds when the mock transport returns the target brightness and output state.
func TestSetLightConfirmed_Gen2Converges(t *testing.T) {
	dev := createMockGen2DeviceWithTransport(func(method string, params any) (json.RawMessage, error) {
		switch method {
		case "Light.Set":
			return json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":{}}`), nil
		case "Light.GetStatus":
			return json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":{"id":0,"output":true,"brightness":20}}`), nil
		default:
			return nil, errors.New("unexpected RPC method: " + method)
		}
	})

	target := LightTarget{On: boolPtr(true), Brightness: intPtr(20)}
	res, err := SetLightConfirmed(context.Background(), dev, 0, target, fastOpts())

	if err != nil {
		t.Fatalf("SetLightConfirmed() error = %v, want nil", err)
	}
	if !res.Converged {
		t.Errorf("Result.Converged = false, want true")
	}
	if res.Applies != 1 {
		t.Errorf("Applies = %d, want 1 (a bulb already at target must not be re-applied)", res.Applies)
	}
}

// TestSetLightConfirmed_Gen1StaleBrightness is the core regression test: a Gen1
// bulb that silently ignores the set command (returns 200 but keeps the old
// brightness) must be reported as non-converged, not as success.
func TestSetLightConfirmed_Gen1StaleBrightness(t *testing.T) {
	opts := fastOpts()

	// Set commands succeed, but GetStatus always returns the stale brightness 10.
	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("brightness") || r.URL.Query().Has("turn") {
			// Set command: acknowledge with 200, body is ignored by callers.
			mustWrite(w, []byte(`{"ison":true,"brightness":10}`))
			return
		}
		// GetStatus: stale — device never transitioned.
		mustWrite(w, []byte(`{"ison":true,"brightness":10}`))
	})
	defer server.Close()

	target := LightTarget{Brightness: intPtr(20)}
	res, err := SetLightConfirmed(context.Background(), dev, 0, target, opts)

	if !errors.Is(err, confirm.ErrNotConverged) {
		t.Errorf("SetLightConfirmed() error = %v, want errors.Is(confirm.ErrNotConverged)", err)
	}
	if res.Converged {
		t.Errorf("Result.Converged = true, want false")
	}
	if res.Applies != opts.MaxApplies {
		t.Errorf("Result.Applies = %d, want %d (MaxApplies)", res.Applies, opts.MaxApplies)
	}
}

// TestSetLightConfirmed_Tolerance verifies that the Tolerance field absorbs
// small rounding differences. A device reporting brightness=21 for a target of
// 20 should converge with Tolerance=1 but not with Tolerance=0.
func TestSetLightConfirmed_Tolerance(t *testing.T) {
	tests := []struct {
		name      string
		tolerance int
		wantConv  bool
	}{
		{name: "within tolerance", tolerance: 1, wantConv: true},
		{name: "outside tolerance", tolerance: 0, wantConv: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Device always reports brightness=21 (one off from the target 20).
			dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
				mustWrite(w, []byte(`{"ison":true,"brightness":21}`))
			})
			defer server.Close()

			target := LightTarget{Brightness: intPtr(20), Tolerance: tt.tolerance}
			res, err := SetLightConfirmed(context.Background(), dev, 0, target, fastOpts())

			if tt.wantConv {
				if err != nil {
					t.Fatalf("SetLightConfirmed() error = %v, want nil", err)
				}
				if !res.Converged {
					t.Errorf("Result.Converged = false, want true (tolerance=%d, reported=21, target=20)", tt.tolerance)
				}
			} else {
				if !errors.Is(err, confirm.ErrNotConverged) {
					t.Errorf("SetLightConfirmed() error = %v, want ErrNotConverged (tolerance=%d)", err, tt.tolerance)
				}
				if res.Converged {
					t.Errorf("Result.Converged = true, want false (tolerance=%d)", tt.tolerance)
				}
			}
		})
	}
}

// TestSetLightConfirmed_OnOnlyTarget verifies that setting only On=true (with no
// Brightness) converges as long as the observed ison matches, regardless of brightness.
func TestSetLightConfirmed_OnOnlyTarget(t *testing.T) {
	// Device reports on=true with an arbitrary brightness — convergence should
	// not require the brightness to match since Brightness is nil in the target.
	dev, server := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		mustWrite(w, []byte(`{"ison":true,"brightness":50}`))
	})
	defer server.Close()

	target := LightTarget{On: boolPtr(true)} // Brightness intentionally nil
	res, err := SetLightConfirmed(context.Background(), dev, 0, target, fastOpts())

	if err != nil {
		t.Fatalf("SetLightConfirmed() error = %v, want nil", err)
	}
	if !res.Converged {
		t.Errorf("Result.Converged = false, want true (on-only target, observed ison=true)")
	}
}

// TestSetLightsConfirmed_BatchPartialFailure verifies that SetLightsConfirmed
// reports exactly one success and one ErrNotConverged failure when one device
// converges and one silently drops the set command.
func TestSetLightsConfirmed_BatchPartialFailure(t *testing.T) {
	// Good device: GetStatus immediately returns the target brightness.
	goodDev, goodServer := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		mustWrite(w, []byte(`{"ison":true,"brightness":30}`))
	})
	defer goodServer.Close()

	// Stale device: GetStatus always returns the old brightness 10, never transitions.
	staleDev, staleServer := createMockGen1Device(t, func(w http.ResponseWriter, r *http.Request) {
		mustWrite(w, []byte(`{"ison":true,"brightness":10}`))
	})
	defer staleServer.Close()

	target := LightTarget{Brightness: intPtr(30)}
	results := SetLightsConfirmed(
		context.Background(),
		[]factory.Device{goodDev, staleDev},
		0,
		target,
		fastOpts(),
	)

	successes := results.Successes()
	failures := results.Failures()

	if len(successes) != 1 {
		t.Errorf("successes = %d, want 1", len(successes))
	}
	if len(failures) != 1 {
		t.Errorf("failures = %d, want 1", len(failures))
	}
	if len(failures) == 1 && !errors.Is(failures[0].Error, confirm.ErrNotConverged) {
		t.Errorf("failure error = %v, want errors.Is(confirm.ErrNotConverged)", failures[0].Error)
	}
}

// TestSetLightConfirmed_UnsupportedDevice verifies that an unrecognized device
// type returns a non-nil error and does not panic.
func TestSetLightConfirmed_UnsupportedDevice(t *testing.T) {
	dev := &unsupportedDevice{}
	target := LightTarget{On: boolPtr(true)}
	_, err := SetLightConfirmed(context.Background(), dev, 0, target, fastOpts())

	if err == nil {
		t.Errorf("SetLightConfirmed() error = nil, want non-nil for unsupported device")
	}
}
