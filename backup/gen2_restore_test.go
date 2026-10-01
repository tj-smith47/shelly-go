package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

type fakeScript struct {
	name    string
	code    string
	id      int
	enable  bool
	running bool
}

// fakeGen2 is a Gen2 device that keeps what it is told, answers its lists in
// the shapes a real device uses, and can be told to refuse or ignore writes.
type fakeGen2 struct {
	config  map[string]map[string]any
	kvs     map[string]any
	refuse  func(method string, params map[string]any) string
	ignore  map[string]bool
	hooks   []map[string]any
	jobs    []map[string]any
	scripts []*fakeScript
	calls   []string
	mu      sync.Mutex
	nextID  int
}

func rpcError(code int, msg string) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"id": 1, "error": map[string]any{"code": code, "message": msg}})
}

func rpcResult(v any) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"id": 1, "result": v})
}

func mergeConfig(dst, src map[string]any) {
	for k, v := range src {
		sub, isObject := v.(map[string]any)
		cur, curIsObject := dst[k].(map[string]any)
		if isObject && curIsObject {
			mergeConfig(cur, sub)
			continue
		}
		dst[k] = v
	}
}

func (f *fakeGen2) client() *rpc.Client {
	return rpc.NewClient(&mockTransport{callFunc: f.call})
}

func (f *fakeGen2) called(prefix string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeGen2) script(id int) *fakeScript {
	for _, s := range f.scripts {
		if s.id == id {
			return s
		}
	}
	return nil
}

func (f *fakeGen2) call(_ context.Context, req transport.RPCRequest) (json.RawMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	method := req.GetMethod()
	params := map[string]any{}
	if raw := req.GetParams(); len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
	}
	f.calls = append(f.calls, method+" "+string(req.GetParams()))
	if f.refuse != nil {
		if msg := f.refuse(method, params); msg != "" {
			return rpcError(-103, msg)
		}
	}
	id := 0
	if n, ok := params["id"].(float64); ok {
		id = int(n)
	}
	switch method {
	case "Shelly.GetDeviceInfo":
		return rpcResult(map[string]any{"id": "shellyplus2pm-bbbbbb", "auth_en": false})
	case "Shelly.GetConfig":
		return rpcResult(f.config)
	case "Shelly.SetProfile":
		sys := f.config["sys"]["device"].(map[string]any)
		sys["profile"] = params["name"]
		return rpcResult(map[string]any{})
	case "Virtual.Add":
		key := fmt.Sprintf("%v:%d", params["type"], id)
		cfg, _ := params["config"].(map[string]any)
		cfg["id"] = float64(id)
		f.config[key] = cfg
		return rpcResult(map[string]any{"id": id})
	case "Webhook.List":
		return rpcResult(map[string]any{"hooks": f.hooks, "rev": 1})
	case "Webhook.DeleteAll":
		f.hooks = nil
		return rpcResult(map[string]any{})
	case "Webhook.Create":
		f.nextID++
		params["id"] = f.nextID
		f.hooks = append(f.hooks, params)
		return rpcResult(map[string]any{"id": f.nextID})
	case "Schedule.List":
		return rpcResult(map[string]any{"jobs": f.jobs, "rev": 1})
	case "Schedule.DeleteAll":
		f.jobs = nil
		return rpcResult(map[string]any{})
	case "Schedule.Create":
		f.nextID++
		params["id"] = f.nextID
		f.jobs = append(f.jobs, params)
		return rpcResult(map[string]any{"id": f.nextID})
	case "Script.List":
		list := []map[string]any{}
		for _, s := range f.scripts {
			list = append(list, map[string]any{"id": s.id, "name": s.name, "enable": s.enable, "running": s.running})
		}
		return rpcResult(map[string]any{"scripts": list})
	case "Script.Create":
		next := 1
		for f.script(next) != nil {
			next++
		}
		name, _ := params["name"].(string)
		f.scripts = append(f.scripts, &fakeScript{id: next, name: name})
		return rpcResult(map[string]any{"id": next})
	case "Script.Delete":
		for i, s := range f.scripts {
			if s.id == id {
				f.scripts = append(f.scripts[:i], f.scripts[i+1:]...)
				break
			}
		}
		return rpcResult(map[string]any{})
	case "Script.PutCode":
		s := f.script(id)
		if appendCode, _ := params["append"].(bool); !appendCode {
			s.code = ""
		}
		s.code += params["code"].(string)
		return rpcResult(map[string]any{"len": len(s.code)})
	case "Script.GetCode":
		// Answers 1000 bytes at a time, as a device does for long code.
		code := f.script(id).code
		offset := 0
		if n, ok := params["offset"].(float64); ok {
			offset = int(n)
		}
		end := min(offset+1000, len(code))
		return rpcResult(map[string]any{"data": code[offset:end], "left": len(code) - end})
	case "Script.SetConfig":
		cfg, _ := params["config"].(map[string]any)
		f.script(id).enable, _ = cfg["enable"].(bool)
		return rpcResult(map[string]any{})
	case "Script.Start", "Script.Stop":
		f.script(id).running = method == "Script.Start"
		return rpcResult(map[string]any{})
	case "KVS.List":
		keys := map[string]any{}
		for k := range f.kvs {
			keys[k] = map[string]any{"etag": "e"}
		}
		return rpcResult(map[string]any{"keys": keys, "rev": 1})
	case "KVS.Get":
		value, ok := f.kvs[params["key"].(string)]
		if !ok {
			return rpcError(-105, "Argument 'key', value not found!")
		}
		return rpcResult(map[string]any{"etag": "e", "value": value})
	case "KVS.Set":
		f.kvs[params["key"].(string)] = params["value"]
		return rpcResult(map[string]any{"rev": 2})
	}
	if class, isSet := strings.CutSuffix(method, ".SetConfig"); isSet {
		key := strings.ToLower(class)
		if _, hasID := params["id"]; hasID {
			key = fmt.Sprintf("%s:%d", key, id)
		}
		section, exists := f.config[key]
		if !exists {
			return rpcError(404, "No handler for "+method)
		}
		cfg, _ := params["config"].(map[string]any)
		for name := range cfg {
			if f.ignore[key+"."+name] {
				delete(cfg, name)
			}
		}
		mergeConfig(section, cfg)
		return rpcResult(map[string]any{"restart_required": false})
	}
	return rpcError(404, "No handler for "+method)
}

func restoreInto(t *testing.T, dev *fakeGen2, bkp *Backup) *RestoreResult {
	t.Helper()
	data, err := json.Marshal(bkp)
	if err != nil {
		t.Fatalf("marshal backup: %v", err)
	}
	result, err := New(dev.client()).Restore(t.Context(), data, DefaultRestoreOptions())
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	sort.Strings(result.Warnings)
	return result
}

func wantWarnings(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("warnings:\n%s\nwant %d:\n%s", strings.Join(got, "\n"), len(want), strings.Join(want, "\n"))
	}
	for i := range want {
		if !strings.HasPrefix(got[i], want[i]) {
			t.Errorf("warning %d = %q, want prefix %q", i, got[i], want[i])
		}
	}
}

// Cloud.SetConfig on a real device refuses any call that names the server.
// The enable flag must still be applied, with nothing reported.
func TestRestore_RefusedSectionIsRetriedPerField(t *testing.T) {
	t.Parallel()
	dev := &fakeGen2{
		config: map[string]map[string]any{
			"cloud": {"enable": false, "server": "a.shelly.cloud:6022/jrpc"},
			"ws":    {"enable": true, "server": nil},
		},
		refuse: func(method string, params map[string]any) string {
			cfg, _ := params["config"].(map[string]any)
			_, hasServer := cfg["server"]
			switch {
			case method == "Cloud.SetConfig" && hasServer:
				return "Permission denied: Only cloud can update!"
			case method == "Ws.SetConfig" && hasServer && cfg["server"] == nil:
				return "Invalid argument 'server': string is expected!"
			}
			return ""
		},
	}
	bkp := &Backup{
		Version: BackupVersion,
		Config:  json.RawMessage(`{"cloud":{"enable":true,"server":"a.shelly.cloud:6022/jrpc"},"ws":{"enable":false,"server":null}}`),
		Cloud:   json.RawMessage(`{"enable":true,"server":"a.shelly.cloud:6022/jrpc"}`),
	}
	result := restoreInto(t, dev, bkp)
	if !result.Success || len(result.Warnings) != 0 {
		t.Fatalf("Success=%v warnings=%v errors=%v", result.Success, result.Warnings, result.Errors)
	}
	if dev.config["cloud"]["enable"] != true || dev.config["ws"]["enable"] != false {
		t.Errorf("fields next to a refused one were not applied: %v", dev.config)
	}
}

func TestRestore_ReportsWhatDidNotApply(t *testing.T) {
	t.Parallel()
	dev := &fakeGen2{
		config: map[string]map[string]any{
			"sys":      {"device": map[string]any{"name": "Old", "mac": "BBBBBB", "fw_id": "fw-b", "profile": "switch"}, "cfg_rev": 3},
			"switch:0": {"id": 0, "name": "Old", "in_mode": "flip", "auto_off": false},
		},
		kvs:    map[string]any{},
		ignore: map[string]bool{"switch:0.name": true},
		refuse: func(method string, params map[string]any) string {
			cfg, _ := params["config"].(map[string]any)
			if _, has := cfg["in_mode"]; method == "Switch.SetConfig" && has {
				return "Invalid argument 'in_mode'"
			}
			if method == "Webhook.Create" && params["name"] == "Night" {
				return "Invalid argument 'event'"
			}
			if method == "KVS.Set" && params["key"] == "full" {
				return "KVS is full"
			}
			return ""
		},
	}
	bkp := &Backup{
		Version: BackupVersion,
		Config: json.RawMessage(`{
			"sys":{"device":{"name":"Bath","mac":"AAAAAA","fw_id":"fw-a","profile":"switch"},"cfg_rev":58},
			"switch:0":{"id":0,"name":"Fan","in_mode":"follow","auto_off":true},
			"switch:1":{"id":1,"name":"Light"}}`),
		Webhooks:  json.RawMessage(`{"hooks":[{"id":3,"cid":0,"enable":true,"event":"switch.on","name":"Night","urls":["http://h/n"]},{"id":4,"cid":0,"enable":true,"event":"switch.off","name":"Day","urls":["http://h/d"]}],"rev":6}`),
		Schedules: json.RawMessage(`{"jobs":[{"id":1,"enable":true,"timespec":"0 0 0 * * *","calls":[{"method":"Shelly.Update"}]}]}`),
		KVS:       map[string]json.RawMessage{"units": json.RawMessage(`{"etag":"x","value":"C"}`), "full": json.RawMessage(`{"etag":"y","value":1}`)},
	}
	result := restoreInto(t, dev, bkp)
	wantWarnings(t, result.Warnings,
		`not applied: kvs "full" (device answered: RPC error -103: KVS is full)`,
		`not applied: switch:0.in_mode: backup has follow, device has flip (device answered: RPC error -103: Invalid argument 'in_mode')`,
		`not applied: switch:0.name: backup has Fan, device has Old`,
		`not applied: switch:1: the device has no such component`,
		`not applied: webhook "Night" (switch.on) (device answered: RPC error -103: Invalid argument 'event')`,
	)
	// Applied next to the refused and ignored fields.
	if dev.config["switch:0"]["auto_off"] != true || dev.config["sys"]["device"].(map[string]any)["name"] != "Bath" {
		t.Errorf("config = %v", dev.config)
	}
	if mac := dev.config["sys"]["device"].(map[string]any)["mac"]; mac != "BBBBBB" {
		t.Errorf("device MAC was overwritten with %v", mac)
	}
	if len(dev.hooks) != 1 || dev.hooks[0]["name"] != "Day" || len(dev.jobs) != 1 || dev.kvs["units"] != "C" {
		t.Errorf("hooks=%v jobs=%v kvs=%v", dev.hooks, dev.jobs, dev.kvs)
	}
}

func TestRestore_SwitchesProfileAndCreatesVirtualComponents(t *testing.T) {
	gen2ProfilePoll = time.Millisecond
	t.Cleanup(func() { gen2ProfilePoll = 2 * time.Second })
	dev := &fakeGen2{config: map[string]map[string]any{
		"sys": {"device": map[string]any{"profile": "cover"}},
	}}
	bkp := &Backup{
		Version: BackupVersion,
		Config: json.RawMessage(`{
			"sys":{"device":{"profile":"switch"}},
			"boolean:200":{"id":200,"name":"Away","persisted":true,"default_value":false}}`),
	}
	result := restoreInto(t, dev, bkp)
	if !result.Success || len(result.Warnings) != 0 {
		t.Fatalf("Success=%v warnings=%v errors=%v", result.Success, result.Warnings, result.Errors)
	}
	if got := dev.called("Shelly.SetProfile"); len(got) != 1 || !strings.Contains(got[0], `"name":"switch"`) {
		t.Errorf("SetProfile calls = %v", got)
	}
	if dev.config["boolean:200"]["name"] != "Away" {
		t.Errorf("virtual component not created: %v", dev.config)
	}
}

func TestRestore_ReplacesScriptsWithFullCode(t *testing.T) {
	t.Parallel()
	// Longer than one upload and one download piece, with a multi-byte
	// character on the upload boundary.
	code := strings.Repeat("a", 1023) + "é" + strings.Repeat("b", 1600)
	dev := &fakeGen2{
		config:  map[string]map[string]any{},
		scripts: []*fakeScript{{id: 1, name: "recover", code: "old"}, {id: 2, name: "other", code: "keep"}},
	}
	bkp := &Backup{
		Version: BackupVersion,
		Scripts: []*Script{{ID: 1, Name: "recover", Code: code, Enable: true, Running: true}},
	}
	result := restoreInto(t, dev, bkp)
	if !result.Success || len(result.Warnings) != 0 {
		t.Fatalf("Success=%v warnings=%v errors=%v", result.Success, result.Warnings, result.Errors)
	}
	if len(dev.scripts) != 2 {
		t.Fatalf("scripts = %d, want the replaced one and the untouched one", len(dev.scripts))
	}
	got := dev.script(1)
	if got == nil || got.name != "recover" || got.code != code || !got.enable || !got.running {
		t.Errorf("restored script = %+v", got)
	}
	if puts := dev.called("Script.PutCode"); len(puts) != 3 || !strings.Contains(puts[0], `"append":false`) {
		t.Errorf("PutCode calls = %d, want 3 with the first replacing", len(puts))
	}
}

func TestExport_CapturesKVSAndWholeScripts(t *testing.T) {
	t.Parallel()
	code := strings.Repeat("x", 2500)
	dev := &fakeGen2{
		config:  map[string]map[string]any{"sys": {"cfg_rev": 1}},
		kvs:     map[string]any{"units": "C", "types": `{"a":[2,0]}`},
		scripts: []*fakeScript{{id: 2, name: "ble", code: code, running: true}},
		hooks:   []map[string]any{{"id": 1, "name": "Night", "event": "switch.on"}},
	}
	data, err := New(dev.client()).Export(t.Context(), DefaultExportOptions())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	var bkp Backup
	if err := json.Unmarshal(data, &bkp); err != nil {
		t.Fatal(err)
	}
	var units struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(bkp.KVS["units"], &units); err != nil || len(bkp.KVS) != 2 || units.Value != "C" {
		t.Errorf("KVS = %s (%v)", bkp.KVS, err)
	}
	if len(bkp.Scripts) != 1 || bkp.Scripts[0].Code != code || !bkp.Scripts[0].Running || bkp.Scripts[0].Name != "ble" {
		t.Errorf("scripts = %+v", bkp.Scripts)
	}
	// The fake has no BLE, Cloud, MQTT or WiFi handler, as a device without
	// those components would not: the export skips them and still succeeds.
	if bkp.BLE != nil || !strings.Contains(string(bkp.Webhooks), "Night") {
		t.Errorf("BLE=%s Webhooks=%s", bkp.BLE, bkp.Webhooks)
	}
}

// A section the device has but could not be read must fail the export. A
// backup that silently lacks it restores as an incomplete device.
func TestExport_FailsWhenASectionCannotBeRead(t *testing.T) {
	t.Parallel()
	for _, method := range []string{
		"WiFi.GetConfig", "Cloud.GetConfig", "BLE.GetConfig", "MQTT.GetConfig",
		"Webhook.List", "Schedule.List", "Script.List", "Script.GetCode", "KVS.List", "KVS.Get",
	} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			dev := &fakeGen2{
				config:  map[string]map[string]any{},
				kvs:     map[string]any{"k": 1},
				scripts: []*fakeScript{{id: 1, name: "s", code: "c"}},
			}
			inner := dev.call
			client := rpc.NewClient(&mockTransport{callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
				if req.GetMethod() == method {
					return nil, errTest
				}
				if strings.HasSuffix(req.GetMethod(), ".GetConfig") && req.GetMethod() != "Shelly.GetConfig" {
					return rpcResult(map[string]any{})
				}
				return inner(ctx, req)
			}})
			if _, err := New(client).Export(t.Context(), DefaultExportOptions()); err == nil {
				t.Fatalf("Export succeeded although %s failed", method)
			}
		})
	}
}
