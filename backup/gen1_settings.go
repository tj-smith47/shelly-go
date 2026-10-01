package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tj-smith47/shelly-go/gen1"
)

// gen1KeyClass says what a restore does with one key of a Gen1 /settings
// response.
type gen1KeyClass int

const (
	// gen1KeyUnknown is a key nobody has classified. A restore treats it as
	// gen1KeyGeneric so a setting introduced by new firmware still transfers.
	gen1KeyUnknown gen1KeyClass = iota
	// gen1KeyGeneric is written by the generic settings step and verified.
	gen1KeyGeneric
	// gen1KeyTyped is written by a dedicated restore step, because its write
	// needs ordering, an override or a different endpoint, and is verified.
	gen1KeyTyped
	// gen1KeyIdentity identifies the device itself and is never transferred.
	gen1KeyIdentity
	// gen1KeyReadOnly is reported by the device and cannot be set.
	gen1KeyReadOnly
	// gen1KeyState is live output state, written by the light-state step and
	// verified unless state restore is skipped.
	gen1KeyState
	// gen1KeyDerived follows from other restored settings: not written, not compared.
	gen1KeyDerived
)

const (
	gen1SettingsPath = "/settings"
	gen1Lights       = "lights"
	gen1Meters       = "meters"
	gen1Relays       = "relays"
	// fieldName is the name setting or parameter on both generations.
	fieldName = "name"
)

// gen1KeyPrefixes classifies whole subtrees. The first matching prefix wins, so
// the single identity or read-only key inside a typed subtree is listed first.
var gen1KeyPrefixes = []struct {
	prefix string
	class  gen1KeyClass
}{
	{"device.", gen1KeyIdentity},
	{"hwinfo.", gen1KeyIdentity},
	{"build_info.", gen1KeyReadOnly},
	{"wifi_ap.ssid", gen1KeyIdentity},
	{"cloud.connected", gen1KeyReadOnly},
	{"actions.names", gen1KeyReadOnly},
	{"wifi_sta.", gen1KeyTyped},
	{"wifi_sta1.", gen1KeyTyped},
	{"wifi_ap.", gen1KeyTyped},
	{"ap_roaming.", gen1KeyTyped},
	{"mqtt.", gen1KeyTyped},
	{"coiot.", gen1KeyTyped},
	{"sntp.", gen1KeyTyped},
	{"login.", gen1KeyTyped},
	{"cloud.", gen1KeyTyped},
	// The per-light copy of the device-level night mode block.
	{"lights[].night_mode.", gen1KeyDerived},
}

// gen1KeyClasses classifies every other known key, by its path with array
// indexes written as [] and numeric object keys as #.
var gen1KeyClasses = map[string]gen1KeyClass{
	fieldName: gen1KeyTyped, "timezone": gen1KeyTyped, "lat": gen1KeyTyped, "lng": gen1KeyTyped,
	"tzautodetect": gen1KeyTyped, "mode": gen1KeyTyped, "discoverable": gen1KeyTyped, "max_power": gen1KeyTyped,

	"fw": gen1KeyReadOnly, "time": gen1KeyReadOnly, "unixtime": gen1KeyReadOnly, "pin_code": gen1KeyReadOnly,
	"device_submodel": gen1KeyReadOnly, "actions.active": gen1KeyDerived,

	"tz_utc_offset": gen1KeyGeneric, "tz_dst": gen1KeyGeneric, "tz_dst_auto": gen1KeyGeneric,
	"transition": gen1KeyGeneric, "allow_cross_origin": gen1KeyGeneric, "debug_enable": gen1KeyGeneric,
	"eco_mode_enabled": gen1KeyGeneric, "pon_wifi_reset": gen1KeyGeneric,
	"led_status_disable": gen1KeyGeneric, "led_power_disable": gen1KeyGeneric,
	"wifirecovery_reboot_enabled": gen1KeyGeneric, "longpush_time": gen1KeyGeneric,
	"multipush_time": gen1KeyGeneric, "factory_reset_from_switch": gen1KeyGeneric,
	"favorites_enabled": gen1KeyGeneric, "supply_voltage": gen1KeyGeneric, "power_correction": gen1KeyGeneric,
	"min_brightness": gen1KeyGeneric, "warm_up": gen1KeyGeneric, "zcross_debounce": gen1KeyGeneric,
	"fw_mode": gen1KeyGeneric, "coiot_execute_enable": gen1KeyGeneric, "ext_switch_enable": gen1KeyGeneric,
	"ext_switch_reverse": gen1KeyGeneric, "ext_sensors.temperature_unit": gen1KeyGeneric,
	"temperature_units": gen1KeyGeneric, "temperature_offset": gen1KeyGeneric, "humidity_offset": gen1KeyGeneric,
	"temperature_threshold": gen1KeyGeneric, "humidity_threshold": gen1KeyGeneric,
	"sleep_mode.period": gen1KeyGeneric, "sleep_mode.unit": gen1KeyReadOnly,
	"sensors.temperature_unit": gen1KeyGeneric, "sensors.temperature_threshold": gen1KeyGeneric,
	"sensors.humidity_threshold": gen1KeyGeneric, "longpush_duration_ms.min": gen1KeyGeneric,
	"longpush_duration_ms.max": gen1KeyGeneric, "pulse_mode": gen1KeyGeneric,
	"external_power": gen1KeyReadOnly, "calibrated": gen1KeyReadOnly,
	"night_mode.enabled": gen1KeyGeneric, "night_mode.start_time": gen1KeyGeneric,
	"night_mode.end_time": gen1KeyGeneric, "night_mode.brightness": gen1KeyGeneric,

	"lights[].name": gen1KeyGeneric, "lights[].transition": gen1KeyGeneric,
	"lights[].default_state": gen1KeyGeneric, "lights[].auto_on": gen1KeyGeneric,
	"lights[].auto_off": gen1KeyGeneric, "lights[].schedule": gen1KeyGeneric,
	"lights[].schedule_rules[]": gen1KeyTyped, "lights[].btn_type": gen1KeyGeneric,
	"lights[].btn1_type": gen1KeyGeneric, "lights[].btn2_type": gen1KeyGeneric,
	"lights[].btn_reverse": gen1KeyGeneric, "lights[].btn1_reverse": gen1KeyGeneric,
	"lights[].btn2_reverse": gen1KeyGeneric, "lights[].btn_debounce": gen1KeyGeneric,
	"lights[].swap_inputs": gen1KeyGeneric,
	"lights[].brightness":  gen1KeyState, "lights[].white": gen1KeyState, "lights[].temp": gen1KeyState,
	"lights[].red": gen1KeyState, "lights[].green": gen1KeyState, "lights[].blue": gen1KeyState,
	"lights[].gain": gen1KeyState, "lights[].effect": gen1KeyState,
	"lights[].ison": gen1KeyReadOnly, "lights[].power": gen1KeyReadOnly, "lights[].overpower": gen1KeyReadOnly,
	"lights[].has_timer": gen1KeyReadOnly, "lights[].source": gen1KeyReadOnly,

	"relays[].name": gen1KeyGeneric, "relays[].appliance_type": gen1KeyGeneric,
	"relays[].default_state": gen1KeyGeneric, "relays[].btn_type": gen1KeyGeneric,
	"relays[].btn_reverse": gen1KeyGeneric, "relays[].auto_on": gen1KeyGeneric,
	"relays[].auto_off": gen1KeyGeneric, "relays[].max_power": gen1KeyGeneric,
	"relays[].schedule": gen1KeyGeneric, "relays[].schedule_rules[]": gen1KeyTyped,
	"relays[].ison": gen1KeyReadOnly, "relays[].has_timer": gen1KeyReadOnly,
	"relays[].overpower": gen1KeyReadOnly, "relays[].is_valid": gen1KeyReadOnly,
	"relays[].power": gen1KeyReadOnly,

	"rollers[].maxtime": gen1KeyGeneric, "rollers[].maxtime_open": gen1KeyGeneric,
	"rollers[].maxtime_close": gen1KeyGeneric, "rollers[].default_state": gen1KeyGeneric,
	"rollers[].swap": gen1KeyGeneric, "rollers[].swap_inputs": gen1KeyGeneric,
	"rollers[].input_mode": gen1KeyGeneric, "rollers[].button_type": gen1KeyGeneric,
	"rollers[].btn_reverse": gen1KeyGeneric, "rollers[].schedule": gen1KeyGeneric,
	"rollers[].schedule_rules[]": gen1KeyTyped, "rollers[].obstacle_mode": gen1KeyGeneric,
	"rollers[].obstacle_action": gen1KeyGeneric, "rollers[].obstacle_power": gen1KeyGeneric,
	"rollers[].obstacle_delay": gen1KeyGeneric, "rollers[].safety_mode": gen1KeyGeneric,
	"rollers[].safety_action": gen1KeyGeneric, "rollers[].safety_allowed_on_trigger": gen1KeyGeneric,
	"rollers[].off_power": gen1KeyGeneric, "rollers[].positioning": gen1KeyGeneric,
	"rollers[].state": gen1KeyReadOnly, "rollers[].power": gen1KeyReadOnly,
	"rollers[].is_valid": gen1KeyReadOnly, "rollers[].safety_switch": gen1KeyReadOnly,

	"meters[].power_limit": gen1KeyGeneric, "meters[].under_limit": gen1KeyGeneric,
	"meters[].power": gen1KeyReadOnly, "meters[].is_valid": gen1KeyReadOnly,
	"emeters[].appliance_type": gen1KeyGeneric, "emeters[].max_power": gen1KeyGeneric,
	"emeters[].ctraf_type": gen1KeyGeneric,

	"inputs[].name": gen1KeyGeneric, "inputs[].btn_type": gen1KeyGeneric, "inputs[].btn_reverse": gen1KeyGeneric,

	"ext_temperature.#.overtemp_threshold_tC":  gen1KeyGeneric,
	"ext_temperature.#.overtemp_threshold_tF":  gen1KeyGeneric,
	"ext_temperature.#.undertemp_threshold_tC": gen1KeyGeneric,
	"ext_temperature.#.undertemp_threshold_tF": gen1KeyGeneric,
	"ext_temperature.#.overtemp_act":           gen1KeyGeneric, "ext_temperature.#.undertemp_act": gen1KeyGeneric,
	"ext_temperature.#.offset_tC": gen1KeyGeneric, "ext_temperature.#.offset_tF": gen1KeyGeneric,
	"ext_humidity.#.overhumidity_threshold": gen1KeyGeneric, "ext_humidity.#.underhumidity_threshold": gen1KeyGeneric,
	"ext_humidity.#.overhumidity_act": gen1KeyGeneric, "ext_humidity.#.underhumidity_act": gen1KeyGeneric,
	"ext_humidity.#.offset":  gen1KeyGeneric,
	"ext_switch.#.relay_num": gen1KeyGeneric,
}

// gen1ClockOwned are the keys the device re-derives from its location service
// while timezone autodetect is on, so a restored value does not persist there.
var gen1ClockOwned = map[string]bool{
	"timezone": true, "tz_utc_offset": true, "tz_dst": true, "tz_dst_auto": true, "lat": true, "lng": true,
}

// gen1WriteOverrides lists keys whose write parameter does not follow the
// path-derived endpoint and name.
var gen1WriteOverrides = map[string][2]string{
	"ext_sensors.temperature_unit":  {gen1SettingsPath, "ext_sensors_temperature_unit"},
	"sensors.temperature_unit":      {gen1SettingsPath, "temperature_units"},
	"sensors.temperature_threshold": {gen1SettingsPath, "temperature_threshold"},
	"sensors.humidity_threshold":    {gen1SettingsPath, "humidity_threshold"},
	"sleep_mode.period":             {gen1SettingsPath, "sleep_mode_period"},
	"longpush_duration_ms.min":      {gen1SettingsPath, "longpush_duration_ms_min"},
	"longpush_duration_ms.max":      {gen1SettingsPath, "longpush_duration_ms_max"},
}

// gen1ComponentEndpoints maps a /settings array to its per-index write endpoint.
var gen1ComponentEndpoints = map[string]string{
	gen1Lights: "light", gen1Relays: "relay", "rollers": "roller",
	gen1Meters: "meter", "emeters": "emeter", "inputs": "input",
}

var (
	gen1IndexRe       = regexp.MustCompile(`\[\d+\]`)
	gen1NumKeyRe      = regexp.MustCompile(`\.\d+(\.|$)`)
	gen1TopPathRe     = regexp.MustCompile(`^([a-zA-Z0-9_]+)$`)
	gen1ArrayPathRe   = regexp.MustCompile(`^([a-zA-Z0-9_]+)\[(\d+)\]\.([a-zA-Z0-9_]+)$`)
	gen1IndexedPathRe = regexp.MustCompile(`^([a-zA-Z0-9_]+)\.(\d+)\.([a-zA-Z0-9_]+)$`)
	gen1ObjectPathRe  = regexp.MustCompile(`^([a-zA-Z0-9_]+)\.([a-zA-Z0-9_]+)$`)
)

// configLeaf is one scalar value of a /settings document.
type configLeaf struct {
	value any // string, bool, json.Number or nil
	path  string
}

// pattern is the leaf's path with indexes removed, the form the class tables use.
func (l configLeaf) pattern() string {
	p := gen1IndexRe.ReplaceAllString(l.path, "[]")
	// Twice, because adjacent numeric segments share a separator.
	p = gen1NumKeyRe.ReplaceAllString(p, ".#$1")
	return gen1NumKeyRe.ReplaceAllString(p, ".#$1")
}

// configLeaves flattens a /settings document into its scalar leaves, sorted by path.
func configLeaves(raw json.RawMessage) ([]configLeaf, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	// Keeps each number's exact text, so a value is written back and compared
	// without a float round trip.
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	return leavesOf(doc), nil
}

// leavesOf flattens a decoded JSON document into its scalar values, each with
// its dotted path, ordered by path.
func leavesOf(doc any) []configLeaf {
	var leaves []configLeaf
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch t := v.(type) {
		case map[string]any:
			for key, child := range t {
				if path == "" {
					walk(key, child)
				} else {
					walk(path+"."+key, child)
				}
			}
		case []any:
			for i, child := range t {
				walk(fmt.Sprintf("%s[%d]", path, i), child)
			}
		default:
			leaves = append(leaves, configLeaf{path: path, value: v})
		}
	}
	walk("", doc)
	sort.Slice(leaves, func(i, j int) bool { return leaves[i].path < leaves[j].path })
	return leaves
}

// classifyGen1Key returns the class of a key pattern, or gen1KeyUnknown when
// no table names it.
func classifyGen1Key(pattern string) gen1KeyClass {
	if class, ok := gen1KeyClasses[pattern]; ok {
		return class
	}
	for _, entry := range gen1KeyPrefixes {
		if strings.HasPrefix(pattern, entry.prefix) {
			return entry.class
		}
	}
	return gen1KeyUnknown
}

// gen1WriteTarget returns the settings endpoint and query parameter that set a
// leaf, or ok=false when the path has no generic write form.
func gen1WriteTarget(l configLeaf) (endpoint, param string, ok bool) {
	if o, found := gen1WriteOverrides[l.pattern()]; found {
		return o[0], o[1], true
	}
	if m := gen1TopPathRe.FindStringSubmatch(l.path); m != nil {
		return gen1SettingsPath, m[1], true
	}
	if m := gen1ArrayPathRe.FindStringSubmatch(l.path); m != nil {
		component, known := gen1ComponentEndpoints[m[1]]
		if !known {
			component = strings.TrimSuffix(m[1], "s")
		}
		return "/settings/" + component + "/" + m[2], m[3], true
	}
	if m := gen1IndexedPathRe.FindStringSubmatch(l.path); m != nil {
		return "/settings/" + m[1] + "/" + m[2], m[3], true
	}
	if m := gen1ObjectPathRe.FindStringSubmatch(l.path); m != nil {
		return "/settings/" + m[1], m[2], true
	}
	return "", "", false
}

// gen1QueryValue renders a leaf value for the device's query API.
func gen1QueryValue(v any) string {
	switch t := v.(type) {
	case bool:
		return strconv.FormatBool(t)
	case json.Number:
		return t.String()
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

// gen1MetersKey reports whether a key belongs to meter configuration.
func gen1MetersKey(pattern string) bool {
	return strings.HasPrefix(pattern, "meters[]") || strings.HasPrefix(pattern, "emeters[]")
}

// gen1LiveSettings returns the device's current settings by path. Only a
// value that differs is written: some writes reboot the device even when they
// change nothing (eco_mode_enabled on a Bulb Duo). A failed read returns an
// empty map, so every setting is written.
func gen1LiveSettings(ctx context.Context, dev *gen1.Device) map[string]any {
	live := map[string]any{}
	raw, err := dev.Call(ctx, gen1SettingsPath)
	if err != nil {
		return live
	}
	leaves, err := configLeaves(raw)
	if err != nil {
		return live
	}
	for _, leaf := range leaves {
		live[leaf.path] = leaf.value
	}
	return live
}

// gen1GenericWrite reports whether the generic step writes a backup setting:
// one no dedicated step covers, that has a value, and that this restore was
// not asked to leave alone.
func gen1GenericWrite(leaf configLeaf, settings *gen1.Settings, opts *Gen1RestoreOptions) bool {
	pattern := leaf.pattern()
	class := classifyGen1Key(pattern)
	if class != gen1KeyGeneric && class != gen1KeyUnknown {
		return false
	}
	return leaf.value != nil &&
		(!settings.Tzautodetect || !gen1ClockOwned[pattern]) &&
		(!opts.SkipMeters || !gen1MetersKey(pattern))
}

// restoreGen1RemainingSettings writes every setting in the backup that no
// dedicated step covers, one request per settings endpoint. It is what makes a
// restore complete for settings the typed steps were never taught, including
// ones added by firmware newer than this code.
func restoreGen1RemainingSettings(
	ctx context.Context,
	dev *gen1.Device,
	bkp *Backup,
	settings *gen1.Settings,
	opts *Gen1RestoreOptions,
	result *RestoreResult,
) {
	leaves, err := configLeaves(bkp.Config)
	if err != nil {
		addWarningf(result, "parse backup settings: %v", err)
		return
	}
	live := gen1LiveSettings(ctx, dev)
	writes := map[string]url.Values{}
	for _, leaf := range leaves {
		if have, reported := live[leaf.path]; reported && sameConfigValue(leaf.value, have) {
			continue
		}
		if !gen1GenericWrite(leaf, settings, opts) {
			continue
		}
		endpoint, param, ok := gen1WriteTarget(leaf)
		if !ok {
			continue
		}
		if writes[endpoint] == nil {
			writes[endpoint] = url.Values{}
		}
		writes[endpoint].Set(param, gen1QueryValue(leaf.value))
	}
	endpoints := make([]string, 0, len(writes))
	for endpoint := range writes {
		endpoints = append(endpoints, endpoint)
	}
	sort.Strings(endpoints)
	for _, endpoint := range endpoints {
		if _, err := dev.Call(ctx, endpoint+"?"+writes[endpoint].Encode()); err != nil {
			addWarningf(result, "set %s: %v", endpoint, err)
		}
	}
}

// gen1VerifySkipped reports whether a backup key is left out of verification:
// it was never meant to transfer, or this restore was asked not to write it.
func gen1VerifySkipped(leaf configLeaf, settings *gen1.Settings, opts *Gen1RestoreOptions) bool {
	pattern := leaf.pattern()
	switch classifyGen1Key(pattern) {
	case gen1KeyIdentity, gen1KeyReadOnly, gen1KeyDerived:
		// A derived value follows from settings that are compared themselves.
		return true
	case gen1KeyState:
		if opts.SkipState {
			return true
		}
	default:
	}
	switch {
	case settings.Tzautodetect && gen1ClockOwned[pattern]:
		return true
	case strings.HasPrefix(pattern, "wifi_sta"):
		// An override deliberately gives the target a different station config.
		return opts.SkipNetwork || opts.NetworkOverride != nil
	case strings.HasPrefix(pattern, "login."):
		return opts.SkipAuth
	case gen1MetersKey(pattern):
		return opts.SkipMeters
	case pattern == "mqtt.id":
		id, isString := leaf.value.(string)
		return isString && gen1SourceIdentity(id, settings)
	}
	return false
}

// sameConfigValue compares a backup value with the live one. Numbers compare by
// value, so 150 and 150.0 are equal.
func sameConfigValue(a, b any) bool {
	na, aIsNum := a.(json.Number)
	nb, bIsNum := b.(json.Number)
	if aIsNum && bIsNum {
		fa, errA := na.Float64()
		fb, errB := nb.Float64()
		return errA == nil && errB == nil && fa == fb
	}
	return a == b
}

// gen1CloudOwnedNote explains a mismatch on a setting the Shelly cloud keeps
// its own value for. While the device is connected, the cloud puts that value
// back within seconds of a local write, so no restore can make it hold.
func gen1CloudOwnedNote(path string, live map[string]any) string {
	if path != "discoverable" || live["cloud.connected"] != true {
		return ""
	}
	return " (the Shelly cloud keeps its own value for this setting" +
		" and puts it back while the device is connected to it)"
}

// verifyGen1Restore reads the device's settings back and records a warning for
// every backup setting the device does not now hold. A restore that reports no
// such warning left the device with the backup's settings; one that does names
// exactly what did not transfer.
func verifyGen1Restore(
	ctx context.Context,
	dev *gen1.Device,
	bkp *Backup,
	settings *gen1.Settings,
	opts *Gen1RestoreOptions,
	result *RestoreResult,
) {
	want, err := configLeaves(bkp.Config)
	if err != nil || len(want) == 0 {
		return
	}
	raw, err := dev.Call(ctx, gen1SettingsPath)
	if err != nil {
		addWarningf(result, "restore not verified: could not read settings back: %v", err)
		return
	}
	liveLeaves, err := configLeaves(raw)
	if err != nil {
		addWarningf(result, "restore not verified: could not parse settings: %v", err)
		return
	}
	live := make(map[string]any, len(liveLeaves))
	for _, leaf := range liveLeaves {
		live[leaf.path] = leaf.value
	}
	for _, leaf := range want {
		if gen1VerifySkipped(leaf, settings, opts) {
			continue
		}
		expected := leaf.value
		if leaf.path == fieldName && opts.Name != "" {
			expected = opts.Name
		}
		got, present := live[leaf.path]
		switch {
		case !present:
			addWarningf(result, "not applied: %s: backup has %v, device does not report it", leaf.path, expected)
		case !sameConfigValue(expected, got):
			addWarningf(result, "not applied: %s: backup has %v, device has %v%s",
				leaf.path, expected, got, gen1CloudOwnedNote(leaf.path, live))
		}
	}
	if !opts.SkipWebhooks {
		verifyGen1Actions(ctx, dev, bkp, result)
	}
}

// verifyGen1Actions compares the device's action URLs with the backup's.
func verifyGen1Actions(ctx context.Context, dev *gen1.Device, bkp *Backup, result *RestoreResult) {
	if bkp.Webhooks == nil {
		return
	}
	var want gen1.ActionSettings
	if err := json.Unmarshal(bkp.Webhooks, &want); err != nil {
		return
	}
	got, err := dev.GetActions(ctx)
	if err != nil {
		addWarningf(result, "actions not verified: %v", err)
		return
	}
	type key struct {
		event gen1.ActionEvent
		index int
	}
	live := make(map[key]gen1.Action, len(got.Actions))
	for _, action := range got.Actions {
		live[key{action.Event, action.Index}] = action
	}
	for _, action := range want.Actions {
		have := live[key{action.Event, action.Index}]
		if action.Enabled != have.Enabled || strings.Join(action.URLs, " ") != strings.Join(have.URLs, " ") {
			addWarningf(result,
				"not applied: action %s[%d]: backup has enabled=%t urls=%v, device has enabled=%t urls=%v",
				action.Event, action.Index, action.Enabled, action.URLs, have.Enabled, have.URLs)
		}
	}
}
