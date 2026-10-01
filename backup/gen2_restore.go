package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/gen2/components"
	"github.com/tj-smith47/shelly-go/rpc"
)

// gen2Identity lists the Shelly.GetConfig values that describe the device
// itself or that the device maintains. A restore does not write them and the
// check after a restore does not compare them. The profile is switched with
// Shelly.SetProfile and checked on its own.
var gen2Identity = map[string]bool{
	"sys.device.mac":     true,
	"sys.device.fw_id":   true,
	"sys.device.profile": true,
	"sys.cfg_rev":        true,
}

// gen2VirtualTypes are the component types a user creates with Virtual.Add.
var gen2VirtualTypes = map[string]bool{
	"boolean": true, "number": true, "text": true, "enum": true, "button": true, "group": true,
}

// gen2AddMethods are the other user-created component types and the method
// that creates each.
var gen2AddMethods = map[string]string{
	"bthomedevice": "BTHome.AddDevice",
	"bthomesensor": "BTHome.AddSensor",
}

// The keys Webhook.List and Schedule.List put their items under.
const (
	itemsHooks = "hooks"
	itemsJobs  = "jobs"
)

// gen2ProfileWait bounds the wait for the reboot that follows a profile change.
var (
	gen2ProfileWait = 90 * time.Second
	gen2ProfilePoll = 2 * time.Second
)

// gen2Rejected records, by setting path or item label, what the device
// answered when it refused a write. The check after the restore reports an
// entry only when the device really differs from the backup there.
type gen2Rejected map[string]string

// reason returns the device's answer for path, or for the closest enclosing
// object that was refused as a whole.
func (r gen2Rejected) reason(path string) string {
	for key, msg := range r {
		if path == key || strings.HasPrefix(path, key+".") || strings.HasPrefix(path, key+"[") {
			return msg
		}
	}
	return ""
}

func (r gen2Rejected) paths() []string {
	paths := make([]string, 0, len(r))
	for path := range r {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func decodeConfig(raw json.RawMessage) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	// Keeps each number's exact text, so a value is written back and compared
	// without a float round trip.
	dec.UseNumber()
	var cfg map[string]any
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// stripGen2Identity removes from one config section the values a restore must
// not copy to another device.
func stripGen2Identity(key string, cfg map[string]any, src *DeviceInfo) {
	var walk func(path string, node map[string]any)
	walk = func(path string, node map[string]any) {
		for name, child := range node {
			childPath := path + "." + name
			if gen2Identity[childPath] {
				delete(node, name)
				continue
			}
			if sub, ok := child.(map[string]any); ok {
				walk(childPath, sub)
			}
		}
	}
	walk(key, cfg)
	if key != componentMQTT || src == nil || src.ID == "" {
		return
	}
	// Both default to the device id. Copying them would make the target
	// connect and publish as the source.
	for _, name := range []string{"client_id", "topic_prefix"} {
		if id, ok := cfg[name].(string); ok && strings.EqualFold(id, src.ID) {
			delete(cfg, name)
		}
	}
}

// setSection sends one SetConfig call for the section named by key.
func (m *Manager) setSection(ctx context.Context, key string, config any) (restart bool, err error) {
	method := gen2.ComponentClassName(key) + ".SetConfig"
	params := map[string]any{paramConfig: config}
	if strings.ContainsRune(key, ':') {
		typ, id, parseErr := gen2.ParseComponentKey(key)
		if parseErr != nil {
			return false, parseErr
		}
		method = gen2.ComponentClassName(typ) + ".SetConfig"
		params["id"] = id
	}
	raw, err := m.client.Call(ctx, method, params)
	if err != nil {
		return false, err
	}
	var res struct {
		RestartRequired bool `json:"restart_required"`
	}
	if json.Unmarshal(raw, &res) != nil {
		return false, nil
	}
	return res.RestartRequired, nil
}

// applySection writes value at path inside the section named by key. A device
// refuses a whole SetConfig call when one field in it is read-only or invalid
// for its firmware, so a refused object is retried one member at a time and
// only the members the device still refuses stay unwritten.
func (m *Manager) applySection(
	ctx context.Context, key string, path []string, value any, rejected gen2Rejected,
) (restart bool) {
	config := value
	for i := len(path) - 1; i >= 0; i-- {
		config = map[string]any{path[i]: config}
	}
	restart, err := m.setSection(ctx, key, config)
	if err == nil {
		return restart
	}
	members, isObject := value.(map[string]any)
	// Only an answer from the device is worth retrying in smaller pieces. A
	// transport failure would repeat for every member.
	var deviceErr *rpc.ErrorObject
	if !isObject || len(members) == 0 || !errors.As(err, &deviceErr) {
		rejected[strings.Join(append([]string{key}, path...), ".")] = err.Error()
		return false
	}
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if m.applySection(ctx, key, append(path[:len(path):len(path)], name), members[name], rejected) {
			restart = true
		}
	}
	return restart
}

// liveConfig reads Shelly.GetConfig as a map of sections.
func (m *Manager) liveConfig(ctx context.Context) (map[string]json.RawMessage, error) {
	raw, err := m.client.Call(ctx, "Shelly.GetConfig", nil)
	if err != nil {
		return nil, err
	}
	var live map[string]json.RawMessage
	if err := json.Unmarshal(raw, &live); err != nil {
		return nil, err
	}
	return live, nil
}

func gen2Profile(cfg map[string]json.RawMessage) string {
	var sys struct {
		Device struct {
			Profile string `json:"profile"`
		} `json:"device"`
	}
	if json.Unmarshal(cfg[componentSys], &sys) != nil {
		return ""
	}
	return sys.Device.Profile
}

// restoreProfile switches the device to the backup's profile and waits for
// the reboot that follows, because the profile decides which components exist
// to receive the rest of the restore. It returns the device config afterwards.
func (m *Manager) restoreProfile(
	ctx context.Context, want string, live map[string]json.RawMessage,
) (map[string]json.RawMessage, error) {
	if want == "" || gen2Profile(live) == "" || gen2Profile(live) == want {
		return live, nil
	}
	if _, err := m.client.Call(ctx, "Shelly.SetProfile", map[string]any{fieldName: want}); err != nil {
		return live, fmt.Errorf("set profile %q: %w", want, err)
	}
	deadline := time.NewTimer(gen2ProfileWait)
	defer deadline.Stop()
	tick := time.NewTicker(gen2ProfilePoll)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return live, ctx.Err()
		case <-deadline.C:
			return live, fmt.Errorf("device did not come back with profile %q within %s", want, gen2ProfileWait)
		case <-tick.C:
			if now, err := m.liveConfig(ctx); err == nil && gen2Profile(now) == want {
				return now, nil
			}
		}
	}
}

// addComponent creates a user-created component the target does not have yet.
// It reports false when key is not a type that can be created.
func (m *Manager) addComponent(ctx context.Context, key string, cfg map[string]any) (created bool, err error) {
	if !strings.ContainsRune(key, ':') {
		return false, nil
	}
	typ, id, err := gen2.ParseComponentKey(key)
	if err != nil {
		return false, err
	}
	delete(cfg, "id")
	params := map[string]any{"id": id, paramConfig: cfg}
	method, known := gen2AddMethods[typ]
	if gen2VirtualTypes[typ] {
		method, known = "Virtual.Add", true
		params["type"] = typ
	}
	if !known {
		return false, nil
	}
	_, err = m.client.Call(ctx, method, params)
	return true, err
}

func itemLabel(kind string, item map[string]any) string {
	switch kind {
	case itemsHooks:
		return fmt.Sprintf("webhook %q (%v)", item[fieldName], item["event"])
	default:
		return fmt.Sprintf("schedule %q", item["timespec"])
	}
}

func scriptLabel(s *Script) string { return fmt.Sprintf("script %q", s.Name) }

func kvsLabel(key string) string { return fmt.Sprintf("kvs %q", key) }

// parseItems returns the list stored under itemsKey with each item's id
// removed, because the target assigns its own.
func parseItems(data json.RawMessage, itemsKey string) ([]map[string]any, error) {
	container, err := decodeConfig(data)
	if err != nil {
		return nil, err
	}
	list, ok := container[itemsKey].([]any)
	if !ok {
		return nil, nil
	}
	items := make([]map[string]any, 0, len(list))
	for _, entry := range list {
		item, isObject := entry.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("%s entry is not an object", itemsKey)
		}
		delete(item, "id")
		items = append(items, item)
	}
	return items, nil
}

func addNotApplied(result *RestoreResult, rejected gen2Rejected, path, detail string) {
	msg := "not applied: " + path
	if detail != "" {
		msg += ": " + detail
	}
	if reason := rejected.reason(path); reason != "" {
		msg += " (device answered: " + reason + ")"
	}
	result.Warnings = append(result.Warnings, msg)
}

// restoredSection returns the backup's config for the section named by key
// when the restore options apply it, with identity values removed.
func restoredSection(key string, cfg map[string]json.RawMessage, opts *RestoreOptions, bkp *Backup) map[string]any {
	raw := cfg[key]
	switch key {
	case componentWiFi:
		// Network settings change the address the device answers on, and a
		// station password is never readable, so they are not compared.
		return nil
	case componentCloud:
		raw = optionalSection(bkp.Cloud, opts.RestoreCloud)
	case componentBLE:
		raw = optionalSection(bkp.BLE, opts.RestoreBLE)
	case componentMQTT:
		raw = optionalSection(bkp.MQTT, opts.RestoreMQTT)
	default:
		if strings.HasPrefix(key, "eth") || !shouldRestoreConfigKey(key, opts) {
			return nil
		}
	}
	if len(raw) == 0 {
		return nil
	}
	section, err := decodeConfig(raw)
	if err != nil {
		return nil
	}
	stripGen2Identity(key, section, bkp.DeviceInfo)
	return section
}

func optionalSection(raw json.RawMessage, restore bool) json.RawMessage {
	if !restore {
		return nil
	}
	return raw
}

// verifyRestore reads the device back and adds one "not applied" warning for
// every setting, webhook, schedule, script and stored value that differs from
// the backup.
func (m *Manager) verifyRestore(
	ctx context.Context, opts *RestoreOptions, bkp *Backup, rejected gen2Rejected, result *RestoreResult,
) {
	var cfg map[string]json.RawMessage
	if len(bkp.Config) > 0 {
		if err := json.Unmarshal(bkp.Config, &cfg); err != nil {
			return
		}
	}
	live, err := m.liveConfig(ctx)
	if err != nil {
		addWarningf(result, "restore not verified: could not read config back: %v", err)
		for _, path := range rejected.paths() {
			addNotApplied(result, rejected, path, "")
		}
		return
	}
	m.verifyConfig(cfg, live, opts, bkp, rejected, result)
	if want := gen2Profile(cfg); opts.RestoreComponents && want != "" && gen2Profile(live) != want {
		addNotApplied(result, rejected, "sys.device.profile",
			fmt.Sprintf("backup has %v, device has %v", want, gen2Profile(live)))
	}
	if opts.RestoreWebhooks && bkp.Webhooks != nil {
		m.verifyItems(ctx, bkp.Webhooks, "Webhook.List", itemsHooks, rejected, result)
	}
	if opts.RestoreSchedules && bkp.Schedules != nil {
		m.verifyItems(ctx, bkp.Schedules, "Schedule.List", itemsJobs, rejected, result)
	}
	if opts.RestoreScripts {
		m.verifyScripts(ctx, bkp.Scripts, rejected, result)
	}
	if opts.RestoreKVS {
		m.verifyKVS(ctx, bkp.KVS, rejected, result)
	}
}

// verifyConfig compares every restored config section with the device's.
func (m *Manager) verifyConfig(
	cfg, live map[string]json.RawMessage, opts *RestoreOptions, bkp *Backup,
	rejected gen2Rejected, result *RestoreResult,
) {
	keys := make([]string, 0, len(cfg))
	for key := range cfg {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		want := restoredSection(key, cfg, opts, bkp)
		if want == nil {
			continue
		}
		liveRaw, present := live[key]
		if !present {
			addNotApplied(result, rejected, key, "the device has no such component")
			continue
		}
		have := map[string]any{}
		if section, decodeErr := decodeConfig(liveRaw); decodeErr == nil {
			for _, leaf := range leavesOf(section) {
				have[leaf.path] = leaf.value
			}
		}
		for _, leaf := range leavesOf(want) {
			got, reported := have[leaf.path]
			switch {
			case !reported && leaf.value == nil, reported && sameConfigValue(leaf.value, got):
			case !reported:
				addNotApplied(result, rejected, key+"."+leaf.path,
					fmt.Sprintf("backup has %v, device does not report it", leaf.value))
			default:
				addNotApplied(result, rejected, key+"."+leaf.path,
					fmt.Sprintf("backup has %v, device has %v", leaf.value, got))
			}
		}
	}
}

// verifyItems checks that every webhook or schedule in the backup exists on
// the device with the same content.
func (m *Manager) verifyItems(
	ctx context.Context, data json.RawMessage, listMethod, itemsKey string,
	rejected gen2Rejected, result *RestoreResult,
) {
	want, err := parseItems(data, itemsKey)
	if err != nil || len(want) == 0 {
		return
	}
	raw, err := m.client.Call(ctx, listMethod, nil)
	if err != nil {
		addWarningf(result, "restore not verified: %s: %v", listMethod, err)
		return
	}
	have, err := parseItems(raw, itemsKey)
	if err != nil {
		addWarningf(result, "restore not verified: %s: %v", listMethod, err)
		return
	}
	remaining := map[string]int{}
	for _, item := range have {
		remaining[canonicalJSON(item)]++
	}
	for _, item := range want {
		if id := canonicalJSON(item); remaining[id] > 0 {
			remaining[id]--
			continue
		}
		addNotApplied(result, rejected, itemLabel(itemsKey, item), "")
	}
}

func canonicalJSON(v any) string {
	out, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(out)
}

func (m *Manager) verifyScripts(ctx context.Context, scripts []*Script, rejected gen2Rejected, result *RestoreResult) {
	if len(scripts) == 0 {
		return
	}
	sc := components.NewScript(m.client)
	list, err := sc.List(ctx)
	if err != nil {
		addWarningf(result, "restore not verified: Script.List: %v", err)
		return
	}
	for _, want := range scripts {
		label := scriptLabel(want)
		var found *components.ScriptListItem
		for i := range list.Scripts {
			if name := list.Scripts[i].Name; name != nil && *name == want.Name {
				found = &list.Scripts[i]
			}
		}
		if found == nil {
			addNotApplied(result, rejected, label, "")
			continue
		}
		code, codeErr := sc.GetCode(ctx, found.ID)
		switch {
		case codeErr != nil:
			addWarningf(result, "restore not verified: %s: %v", label, codeErr)
		case code.Data != want.Code:
			addNotApplied(result, rejected, label, "the code on the device differs from the backup")
		case found.Enable != want.Enable:
			addNotApplied(result, rejected, label,
				fmt.Sprintf("backup has enable %v, device has %v", want.Enable, found.Enable))
		case found.ID != want.ID:
			// A webhook or schedule that starts the script names it by id.
			addWarningf(result, "%s has id %d on the device, the backup has id %d", label, found.ID, want.ID)
		}
	}
}

func (m *Manager) verifyKVS(
	ctx context.Context, kvs map[string]json.RawMessage, rejected gen2Rejected, result *RestoreResult,
) {
	keys := make([]string, 0, len(kvs))
	for key := range kvs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		raw, err := m.client.Call(ctx, "KVS.Get", map[string]any{paramKey: key})
		if err != nil {
			addNotApplied(result, rejected, kvsLabel(key), "")
			continue
		}
		var want, have struct {
			Value any `json:"value"`
		}
		if json.Unmarshal(kvs[key], &want) != nil || json.Unmarshal(raw, &have) != nil ||
			canonicalJSON(want.Value) != canonicalJSON(have.Value) {
			addNotApplied(result, rejected, kvsLabel(key), "the stored value differs from the backup")
		}
	}
}
