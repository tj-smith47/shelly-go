package backup

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestParseGen1FirmwareDate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fw   string
		want int
	}{
		{"modern duo", "20230913-111821/v1.14.0-gcb84623", 20230913},
		{"legacy duo", "20191216-140245/v1.5.7@c30657ba", 20191216},
		{"corrupt build suffix still has date", "20191216-140245/???", 20191216},
		{"no leading date", "v1.9.5", 0},
		{"too short", "2023", 0},
		{"empty", "", 0},
		{"non-digit in date", "2023x913-111821", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseGen1FirmwareDate(tt.fw); got != tt.want {
				t.Errorf("parseGen1FirmwareDate(%q) = %d, want %d", tt.fw, got, tt.want)
			}
		})
	}
}

// gen1PaceDevice starts a fake Gen1 device that answers /settings with the given
// firmware string and /status with an uptime taken from upticks: each /status
// call returns the next value (the last value repeats once exhausted), so a
// device can be made to look mid-reboot first and recovered later.
func gen1PaceDevice(t *testing.T, fw string, upticks []int) *gen1.Device {
	t.Helper()
	var calls int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/shelly":
			body = `{"type":"SHBDUO-1","mac":"AABBCCDDEEFF","fw":"` + fw + `","auth":false,"num_outputs":1}`
		case "/settings":
			body = `{"fw":"` + fw + `","name":"x"}`
		case "/status":
			i := int(atomic.AddInt64(&calls, 1)) - 1
			if i >= len(upticks) {
				i = len(upticks) - 1
			}
			up := 0
			if len(upticks) > 0 {
				up = upticks[i]
			}
			body = `{"uptime":` + strconv.Itoa(up) + `}`
		default:
			body = `{}`
		}
		if _, err := io.WriteString(w, body); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return gen1.NewDevice(transport.NewHTTP(srv.URL))
}

func TestGen1SettleFor_FirmwareAware(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fw   string
		want time.Duration
	}{
		{"modern firmware paces lighter", "20230913-111821/v1.14.0-gcb84623", gen1SettleModern},
		{"legacy firmware paces heavier", "20191216-140245/v1.5.7", gen1SettleLegacy},
		{"unparseable firmware treated as legacy", "v1.9.5", gen1SettleLegacy},
		{"empty (unreachable) treated as legacy", "", gen1SettleLegacy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gen1SettleFor(tt.fw); got != tt.want {
				t.Errorf("gen1SettleFor(%q) = %v, want %v", tt.fw, got, tt.want)
			}
		})
	}
}

func TestGen1LiveFirmware_UnreachableIsEmpty(t *testing.T) {
	t.Parallel()
	// A device whose /settings cannot be read yields an empty firmware string,
	// which sorts as oldest/legacy in both pacing and the downgrade gate.
	dev := gen1.NewDevice(transport.NewHTTP("http://127.0.0.1:0"))
	if fw := gen1LiveFirmware(context.Background(), dev); fw != "" {
		t.Errorf("gen1LiveFirmware(unreachable) = %q, want empty", fw)
	}
}

func TestGen1FirmwareDowngrade(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		live   string
		backup string
		want   bool
	}{
		{"older target than backup is a downgrade", "20191216-140245/???", "20230913-111821/v1.14.0", true},
		{"same firmware is not a downgrade", "20230913-111821/v1.14.0", "20230913-111821/v1.14.0", false},
		{"newer target than backup is not a downgrade", "20230913-111821/v1.14.0", "20191216-140245/v1.5.7", false},
		{"unknown target firmware is not affirmative downgrade", "v1.9.5", "20230913-111821/v1.14.0", false},
		{"unknown backup firmware is not affirmative downgrade", "20191216-140245/v1.5.7", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gen1FirmwareDowngrade(tt.live, tt.backup); got != tt.want {
				t.Errorf("gen1FirmwareDowngrade(%q, %q) = %v, want %v", tt.live, tt.backup, got, tt.want)
			}
		})
	}
}

func TestRestoreGen1_RefusesFirmwareDowngrade(t *testing.T) {
	t.Parallel()
	// The target runs ancient firmware (2019); the backup was captured from newer
	// firmware (2023). The restore must refuse before writing anything, naming both
	// firmwares, rather than bricking the device into a reboot loop.
	dev, writes := gen1ColorDevice(t, `{"fw":"20191216-140245/???","uptime":9000}`)
	bkp := &Backup{Config: json.RawMessage(`{"name":"FR","fw":"20230913-111821/v1.14.0"}`)}

	_, err := RestoreGen1(context.Background(), dev, bkp, &Gen1RestoreOptions{})
	if err == nil {
		t.Fatal("RestoreGen1 did not refuse a firmware downgrade")
	}
	if !strings.Contains(err.Error(), "predates") {
		t.Errorf("error did not explain the downgrade: %v", err)
	}
	// The bare GET /settings is the firmware probe; a real write carries a query
	// (/settings?name=) or a subpath (/settings/mqtt). None must have been issued.
	for _, w := range *writes {
		if strings.HasPrefix(w, "/settings?") || strings.HasPrefix(w, "/settings/") {
			t.Errorf("restore wrote a setting (%q) despite refusing the downgrade", w)
		}
	}
}

func TestRestoreGen1_AllowFirmwareDowngradeOverride(t *testing.T) {
	t.Parallel()
	// With the override set, the same downgrade proceeds (writes settings) instead
	// of refusing — the escape hatch for an accepted-risk restore.
	dev, writes := gen1ColorDevice(t, `{"fw":"20191216-140245/???","uptime":9000}`)
	bkp := &Backup{Config: json.RawMessage(`{"name":"FR","fw":"20230913-111821/v1.14.0"}`)}

	_, err := RestoreGen1(context.Background(), dev, bkp,
		&Gen1RestoreOptions{AllowFirmwareDowngrade: true, SkipNetwork: true, SkipAuth: true})
	if err != nil {
		t.Fatalf("RestoreGen1 with override: %v", err)
	}
	wrote := false
	for _, w := range *writes {
		if strings.HasPrefix(w, "/settings?") || strings.HasPrefix(w, "/settings/") {
			wrote = true
			break
		}
	}
	if !wrote {
		t.Error("override restore did not write any settings")
	}
}

func TestAfterWrite_ReturnsFastWhenStable(t *testing.T) {
	t.Parallel()
	dev := gen1PaceDevice(t, "20230913-111821/v1.14.0", []int{9000})
	p := gen1Pacer{settle: time.Millisecond}
	start := time.Now()
	uptime, stable := p.afterWrite(context.Background(), dev)
	// A device already up for a long time clears the stable-uptime check on the
	// first poll, so afterWrite returns at roughly the settle floor, never the
	// recovery budget.
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("afterWrite took %v on a stable device; expected ~settle floor", elapsed)
	}
	if !stable {
		t.Error("afterWrite reported unstable for a device up 9000s")
	}
	if uptime != 9000 {
		t.Errorf("afterWrite uptime = %d, want 9000", uptime)
	}
}

func TestAfterWrite_WaitsForRebootRecovery(t *testing.T) {
	t.Parallel()
	// First /status reads as mid-reboot (uptime below the stable threshold), then
	// the device climbs past it: afterWrite must not return until it is stable.
	dev := gen1PaceDevice(t, "20191216-140245/v1.5.7", []int{1, 2, gen1StableUptime + 1})
	p := gen1Pacer{settle: time.Millisecond}
	start := time.Now()
	uptime, stable := p.afterWrite(context.Background(), dev)
	// Two sub-threshold reads force at least two recovery polls before the third
	// read clears the check.
	if elapsed := time.Since(start); elapsed < gen1RecoveryPoll {
		t.Errorf("afterWrite returned in %v; expected it to wait for uptime to recover", elapsed)
	}
	if !stable {
		t.Error("afterWrite reported unstable after the device climbed past the threshold")
	}
	if uptime != gen1StableUptime+1 {
		t.Errorf("afterWrite uptime = %d, want %d", uptime, gen1StableUptime+1)
	}
}

func TestRestoreGen1_ClockDependentOnly(t *testing.T) {
	t.Parallel()
	// A modern fw keeps the settle short, and a high uptime clears the stability
	// poll on the first read, so the pass does not sleep on the recovery budget.
	dev, writes := gen1ColorDevice(t, `{"fw":"20230913-111821/v1.14.0","uptime":9000}`)

	bkp := &Backup{
		Config: json.RawMessage(`{"name":"FR","timezone":"America/Chicago","mqtt":{"enable":true,"server":"x:1883"},` +
			`"lights":[{"name":"Bath","default_state":"on","schedule_rules":["0000asr-0123456-0;101;off"]}]}`),
		Components: map[string]json.RawMessage{
			lightStateKey: json.RawMessage(`[{"id":0,"temp":6500,"brightness":50}]`),
		},
	}

	_, err := RestoreGen1(context.Background(), dev, bkp, &Gen1RestoreOptions{ClockDependentOnly: true})
	if err != nil {
		t.Fatalf("RestoreGen1: %v", err)
	}

	var sawLightConfig, sawLightState bool
	for _, w := range *writes {
		switch {
		case strings.Contains(w, "/settings/light/0"):
			sawLightConfig = true
		case strings.HasPrefix(w, "/light/0"):
			sawLightState = true
		}
		// The clock-dependent pass must not re-write device or network settings
		// that already applied at the AP. Device-level settings are written to the
		// /settings root (e.g. /settings?name=, /settings?timezone=) and MQTT to
		// /settings/mqtt — distinct from the light config at /settings/light/N,
		// whose own name= query param is legitimate.
		if strings.HasPrefix(w, "/settings?") || strings.Contains(w, "/settings/mqtt") {
			t.Errorf("clock-dependent pass wrote a non-clock setting: %q", w)
		}
	}
	if !sawLightConfig {
		t.Error("clock-dependent pass did not write light config (/settings/light/0)")
	}
	if !sawLightState {
		t.Error("clock-dependent pass did not write light state (/light/0)")
	}
}

func TestAfterWrite_CancelledContextReturns(t *testing.T) {
	t.Parallel()
	// A device stuck mid-reboot forever must not hang afterWrite past context
	// cancellation.
	dev := gen1PaceDevice(t, "20191216-140245/v1.5.7", []int{0})
	p := gen1Pacer{settle: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, stable := p.afterWrite(ctx, dev)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("afterWrite ignored context cancellation; ran %v", elapsed)
	}
	if stable {
		t.Error("afterWrite reported stable for a device stuck at uptime 0")
	}
}

func TestRunGen1Step_HaltsAndRecordsOnDestabilize(t *testing.T) {
	t.Parallel()
	// A device whose uptime never climbs to the stable threshold is a reboot loop:
	// runGen1Step must report not-stable, name the breaking step, fail the result,
	// and (when tracing) mark the step DESTABILIZED — exactly the evidence needed
	// to answer "which write broke it." The short context bounds the recovery wait.
	dev := gen1PaceDevice(t, "20191216-140245/v1.5.7", []int{4})
	pacer := gen1Pacer{settle: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	var trace strings.Builder
	result := &RestoreResult{Success: true}
	opts := &Gen1RestoreOptions{StepTrace: &trace}

	ran := false
	ok := runGen1Step(ctx, dev, pacer, opts, "coiot", result, func() { ran = true })

	if !ran {
		t.Fatal("runGen1Step did not execute the write closure")
	}
	if ok {
		t.Error("runGen1Step reported stable for a device stuck below the stable uptime")
	}
	if result.DestabilizedStep != "coiot" {
		t.Errorf("DestabilizedStep = %q, want %q", result.DestabilizedStep, "coiot")
	}
	if result.Success {
		t.Error("Success should be false after a destabilizing step")
	}
	if len(result.Errors) == 0 {
		t.Error("expected an error recorded for the destabilizing step")
	}
	if got := trace.String(); !strings.Contains(got, "step=coiot") || !strings.Contains(got, "DESTABILIZED") {
		t.Errorf("trace missing step label or DESTABILIZED marker: %q", got)
	}
}

func TestRunGen1Step_TracesAndContinuesWhenStable(t *testing.T) {
	t.Parallel()
	// A stable device clears the step: runGen1Step returns true, leaves the result
	// undestabilized, and the trace records the step's warning count and an ok state.
	dev := gen1PaceDevice(t, "20230913-111821/v1.14.0", []int{9000})
	pacer := gen1Pacer{settle: time.Millisecond}

	var trace strings.Builder
	result := &RestoreResult{Success: true}
	opts := &Gen1RestoreOptions{StepTrace: &trace}

	ok := runGen1Step(context.Background(), dev, pacer, opts, "mqtt", result, func() {
		result.Warnings = append(result.Warnings, "minor")
	})

	if !ok {
		t.Error("runGen1Step reported unstable for a device up 9000s")
	}
	if result.DestabilizedStep != "" {
		t.Errorf("DestabilizedStep = %q, want empty", result.DestabilizedStep)
	}
	if got := trace.String(); !strings.Contains(got, "step=mqtt") ||
		!strings.Contains(got, "warnings=1") || !strings.Contains(got, "ok") {
		t.Errorf("unexpected trace line: %q", got)
	}
}

func TestRunGen1Step_NilTraceIsNoOp(t *testing.T) {
	t.Parallel()
	// With no trace sink configured (the normal restore), runGen1Step must still
	// pace and judge the step without panicking on the nil writer.
	dev := gen1PaceDevice(t, "20230913-111821/v1.14.0", []int{9000})
	pacer := gen1Pacer{settle: time.Millisecond}
	result := &RestoreResult{Success: true}
	opts := &Gen1RestoreOptions{}

	if ok := runGen1Step(context.Background(), dev, pacer, opts, "sntp", result, func() {}); !ok {
		t.Error("runGen1Step reported unstable for a stable device with no trace sink")
	}
}
