package helpers

import (
	"context"
	"fmt"

	"github.com/tj-smith47/shelly-go/confirm"
	"github.com/tj-smith47/shelly-go/factory"
	gen1components "github.com/tj-smith47/shelly-go/gen1/components"
	gen2components "github.com/tj-smith47/shelly-go/gen2/components"
)

// LightTarget is the desired light state to apply and confirm. A nil pointer field
// means "leave this attribute unchanged"; convergence only checks the attributes that
// were actually requested.
type LightTarget struct {
	// On sets the on/off state. nil leaves it unchanged.
	On *bool

	// Brightness sets the level (0-100). nil leaves it unchanged.
	Brightness *int

	// TransitionMs is the fade duration applied with the change, in milliseconds. 0
	// uses the device default. It affects only how the change is issued, never the
	// convergence check.
	TransitionMs int

	// Tolerance is the maximum acceptable absolute difference between the requested
	// and observed brightness, absorbing rounding and mid-transition sampling. 0
	// requires an exact match.
	Tolerance int
}

// SetLightConfirmed applies t to light id on dev, then polls the device's LIVE status
// until it reflects the target (or o's deadline elapses), re-applying to the device if
// it lags. It works across Gen1 (REST /light/N) and Gen2+ (RPC Light.Set / Light.GetStatus).
//
// Unlike a fire-and-forget set, it reads back live output state — not persisted
// configuration — so a silently dropped transition is detected and surfaced as a
// wrapped [confirm.ErrNotConverged]. An unreachable device fails fast with the
// underlying transport error so the caller can classify it.
func SetLightConfirmed(
	ctx context.Context,
	dev factory.Device,
	id int,
	t LightTarget,
	o confirm.Options,
) (confirm.Result, error) {
	apply, check, err := lightClosures(dev, id, t)
	if err != nil {
		return confirm.Result{}, err
	}
	return confirm.Until(ctx, apply, check, o)
}

// SetLightsConfirmed applies t to light id on every device concurrently, confirming
// each independently. Because each device runs its own confirm loop, only the bulbs
// that lag are re-applied — a healthy bulb is never re-commanded because a sibling
// stalled. Results align by index with devices.
func SetLightsConfirmed(
	ctx context.Context,
	devices []factory.Device,
	id int,
	t LightTarget,
	o confirm.Options,
) BatchResults {
	return batchExecute(ctx, devices, func(ctx context.Context, dev factory.Device) error {
		_, err := SetLightConfirmed(ctx, dev, id, t, o)
		return err
	})
}

// lightClosures builds the generation-appropriate apply and check closures for a
// light target, isolating the Gen1/Gen2 branching from the confirm loop.
func lightClosures(dev factory.Device, id int, t LightTarget) (confirm.Apply, confirm.Check, error) {
	switch d := dev.(type) {
	case *factory.Gen1Device:
		light := d.Light(id)
		apply := func(ctx context.Context) error {
			if err := applyGen1Brightness(ctx, light, t); err != nil {
				return err
			}
			if t.On != nil {
				return light.Set(ctx, *t.On)
			}
			return nil
		}
		check := func(ctx context.Context) (bool, string, error) {
			st, err := light.GetStatus(ctx)
			if err != nil {
				return false, "", err
			}
			return lightConverged(t, st.IsOn, st.Brightness),
				fmt.Sprintf("on=%t brightness=%d", st.IsOn, st.Brightness), nil
		}
		return apply, check, nil

	case *factory.Gen2Device:
		light := gen2components.NewLight(d.Client(), id)
		apply := func(ctx context.Context) error {
			params := &gen2components.LightSetParams{ID: id, On: t.On, Brightness: t.Brightness}
			if t.TransitionMs > 0 {
				transition := t.TransitionMs
				params.TransitionDuration = &transition
			}
			_, err := light.Set(ctx, params)
			return err
		}
		check := func(ctx context.Context) (bool, string, error) {
			st, err := light.GetStatus(ctx)
			if err != nil {
				return false, "", err
			}
			brightness := 0
			if st.Brightness != nil {
				brightness = *st.Brightness
			}
			return lightConverged(t, st.Output, brightness),
				fmt.Sprintf("output=%t brightness=%d", st.Output, brightness), nil
		}
		return apply, check, nil

	default:
		return nil, nil, fmt.Errorf("helpers: unsupported device generation %v", dev.Generation())
	}
}

// applyGen1Brightness issues the brightness portion of a target to a Gen1 light,
// choosing the transition-aware setter when a fade was requested. A nil Brightness
// leaves the level unchanged.
func applyGen1Brightness(ctx context.Context, light *gen1components.Light, t LightTarget) error {
	if t.Brightness == nil {
		return nil
	}
	if t.TransitionMs > 0 {
		return light.SetBrightnessWithTransition(ctx, *t.Brightness, t.TransitionMs)
	}
	return light.SetBrightness(ctx, *t.Brightness)
}

// lightConverged reports whether an observed (on, brightness) reading satisfies the
// requested target within tolerance. Target attributes left unset are ignored.
func lightConverged(t LightTarget, on bool, brightness int) bool {
	if t.On != nil && *t.On != on {
		return false
	}
	if t.Brightness != nil {
		tol := t.Tolerance
		if tol < 0 {
			tol = 0
		}
		if diff := brightness - *t.Brightness; diff < -tol || diff > tol {
			return false
		}
	}
	return true
}
