package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/tj-smith47/shelly-go/gen1"
	"github.com/tj-smith47/shelly-go/transport"
)

const (
	gen1TestFW        = "20230913-111821/v1.14.0"
	gen1NoActionsJSON = `{"actions":{}}`
)

// gen1SettingsFixtures returns every captured /settings document in testdata,
// keyed by file name.
func gen1SettingsFixtures(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	paths, err := filepath.Glob("testdata/gen1/*.json")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no Gen1 settings fixtures: %v", err)
	}
	out := make(map[string]json.RawMessage, len(paths))
	for _, p := range paths {
		raw, readErr := os.ReadFile(p)
		if readErr != nil {
			t.Fatalf("read %s: %v", p, readErr)
		}
		out[filepath.Base(p)] = raw
	}
	return out
}

// gen1SettingsServer serves live as the device's /settings and actions as its
// /settings/actions, and records the URI of every request that carries a query.
func gen1SettingsServer(t *testing.T, live, actions string) (*gen1.Device, func() []string) {
	t.Helper()
	var (
		mu     sync.Mutex
		writes []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := `{}`
		switch {
		case r.URL.RawQuery != "":
			mu.Lock()
			writes = append(writes, r.URL.RequestURI())
			mu.Unlock()
		case r.URL.Path == "/shelly":
			body = `{"type":"SHBDUO-1","mac":"AABBCCDDEEFF","fw":"` + gen1TestFW + `","auth":false}`
		case r.URL.Path == "/settings":
			body = live
		case r.URL.Path == "/settings/actions":
			body = actions
		case r.URL.Path == "/status":
			body = `{"uptime":3600,"unixtime":1700000000}`
		}
		if _, err := io.WriteString(w, body); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return gen1.NewDevice(transport.NewHTTP(srv.URL)), func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), writes...)
	}
}

// sentParams merges the query parameters of the recorded writes per endpoint.
func sentParams(t *testing.T, uris []string) map[string]url.Values {
	t.Helper()
	sent := map[string]url.Values{}
	for _, uri := range uris {
		u, err := url.Parse(uri)
		if err != nil {
			t.Fatalf("parse %q: %v", uri, err)
		}
		if sent[u.Path] == nil {
			sent[u.Path] = url.Values{}
		}
		for k, v := range u.Query() {
			sent[u.Path][k] = v
		}
	}
	return sent
}

// A key in a fixture that no table classifies fails here, so whoever adds a
// device capture decides for each new key whether a restore transfers it.
func TestGen1SettingsKeys_AllClassified(t *testing.T) {
	t.Parallel()
	for name, raw := range gen1SettingsFixtures(t) {
		leaves, err := configLeaves(raw)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, leaf := range leaves {
			if classifyGen1Key(leaf.pattern()) == gen1KeyUnknown {
				t.Errorf("%s: key %q (pattern %q) is not classified in gen1KeyClasses or gen1KeyPrefixes",
					name, leaf.path, leaf.pattern())
			}
		}
	}
}

// changeGenericSettings returns doc with every generic setting given another
// value, so a device serving it differs from the backup in each of them.
func changeGenericSettings(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	var walk func(path string, v any) any
	walk = func(path string, v any) any {
		switch node := v.(type) {
		case map[string]any:
			for key, child := range node {
				node[key] = walk(strings.TrimPrefix(path+"."+key, "."), child)
			}
		case []any:
			for i, child := range node {
				node[i] = walk(fmt.Sprintf("%s[%d]", path, i), child)
			}
		default:
			if classifyGen1Key(configLeaf{path: path}.pattern()) != gen1KeyGeneric {
				return v
			}
			switch value := v.(type) {
			case bool:
				return !value
			case json.Number:
				n, err := value.Float64()
				if err != nil {
					t.Fatalf("%s: %v", path, err)
				}
				return n + 1
			case string:
				return value + "-other"
			}
		}
		return v
	}
	out, err := json.Marshal(walk("", doc))
	if err != nil {
		t.Fatalf("encode changed fixture: %v", err)
	}
	return string(out)
}

// Every setting classed generic must reach the device in a write to its own
// endpoint, with the backup's value, for every captured device type. The fake
// device never applies a write, so the check after the restore must also
// report each of those settings as not applied.
func TestGen1Restore_WritesEveryGenericSetting(t *testing.T) {
	t.Parallel()
	for name, raw := range gen1SettingsFixtures(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dev, writes := gen1SettingsServer(t, changeGenericSettings(t, raw), gen1NoActionsJSON)
			result, err := RestoreGen1(t.Context(), dev, &Backup{Config: raw},
				&Gen1RestoreOptions{SkipNetwork: true, SkipClockWait: true})
			if err != nil {
				t.Fatalf("RestoreGen1: %v", err)
			}
			warnings := strings.Join(result.Warnings, "\n")
			sent := sentParams(t, writes())
			var settings gen1.Settings
			if err := json.Unmarshal(raw, &settings); err != nil {
				t.Fatalf("parse fixture: %v", err)
			}
			leaves, err := configLeaves(raw)
			if err != nil {
				t.Fatal(err)
			}
			for _, leaf := range leaves {
				pattern := leaf.pattern()
				if classifyGen1Key(pattern) != gen1KeyGeneric || leaf.value == nil ||
					(settings.Tzautodetect && gen1ClockOwned[pattern]) {
					continue
				}
				endpoint, param, ok := gen1WriteTarget(leaf)
				if !ok {
					t.Errorf("generic key %q has no write target", leaf.path)
					continue
				}
				if got := sent[endpoint].Get(param); got != gen1QueryValue(leaf.value) {
					t.Errorf("%s: %s?%s = %q, want %q", leaf.path, endpoint, param, got, gen1QueryValue(leaf.value))
				}
				if !strings.Contains(warnings, "not applied: "+leaf.path+": ") {
					t.Errorf("%s was not applied and no warning says so", leaf.path)
				}
			}
		})
	}
}

// A device that already holds the backup's values gets no settings write:
// some writes reboot it even when nothing changes.
func TestGen1Restore_SkipsSettingsAlreadyInPlace(t *testing.T) {
	t.Parallel()
	for name, raw := range gen1SettingsFixtures(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dev, writes := gen1SettingsServer(t, string(raw), gen1NoActionsJSON)
			var settings gen1.Settings
			if err := json.Unmarshal(raw, &settings); err != nil {
				t.Fatalf("parse fixture: %v", err)
			}
			result := &RestoreResult{}
			restoreGen1RemainingSettings(t.Context(), dev, &Backup{Config: raw}, &settings,
				&Gen1RestoreOptions{}, result)
			if sent := writes(); len(sent) != 0 || len(result.Warnings) != 0 {
				t.Errorf("writes %q, warnings %q; want none", sent, result.Warnings)
			}
		})
	}
}

// A setting the device did not take, or does not report at all, is named in a
// warning; identity, state and overridden values are left out.
func TestGen1Restore_ReportsSettingsThatDidNotApply(t *testing.T) {
	t.Parallel()
	source := `{"transition":150,"eco_mode_enabled":true,"name":"src","fw":"` + gen1TestFW + `",
		"device":{"hostname":"shelly-A","mac":"A"},"lights":[{"transition":150,"ison":true}]}`
	live := `{"transition":1000,"name":"clone","fw":"` + gen1TestFW + `",
		"device":{"hostname":"shelly-B","mac":"B"},"lights":[{"transition":150,"ison":false}]}`
	actions := `{"actions":{"out_on_url":[{"index":0,"urls":[],"enabled":false}]}}`
	bkp := &Backup{
		Config:   json.RawMessage(source),
		Webhooks: json.RawMessage(`{"actions":{"out_on_url":[{"index":0,"urls":["http://localhost/x"],"enabled":true}]}}`),
	}
	dev, writes := gen1SettingsServer(t, live, actions)
	result, err := RestoreGen1(t.Context(), dev, bkp,
		&Gen1RestoreOptions{SkipNetwork: true, SkipClockWait: true, Name: "clone"})
	if err != nil {
		t.Fatalf("RestoreGen1: %v", err)
	}
	joined := strings.Join(result.Warnings, "\n")
	for _, want := range []string{
		"not applied: transition: backup has 150, device has 1000",
		"not applied: eco_mode_enabled: backup has true, device does not report it",
		"not applied: action out_on_url[0]",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing warning %q in:\n%s", want, joined)
		}
	}
	// Identity, read-only state and the overridden name are never reported.
	for _, banned := range []string{"device.", "ison", "not applied: name"} {
		if strings.Contains(joined, banned) {
			t.Errorf("warning mentions %q:\n%s", banned, joined)
		}
	}
	sent := strings.Join(writes(), "\n")
	for _, want := range []string{
		"/settings?", "transition=150", "eco_mode_enabled=true",
		"/settings/actions?", "name=out_on_url", "urls[]=http%3A%2F%2Flocalhost%2Fx",
	} {
		if !strings.Contains(sent, want) {
			t.Errorf("write missing %q in:\n%s", want, sent)
		}
	}
}

// The Shelly cloud reverts discoverable on a connected device, so the warning
// says so instead of leaving an unexplained mismatch.
func TestGen1Restore_ExplainsCloudOwnedSetting(t *testing.T) {
	t.Parallel()
	source := `{"fw":"` + gen1TestFW + `","discoverable":true,"cloud":{"enabled":true,"connected":true}}`
	for connected, wantNote := range map[bool]bool{true: true, false: false} {
		live := fmt.Sprintf(`{"fw":%q,"discoverable":false,"cloud":{"enabled":true,"connected":%t}}`, gen1TestFW, connected)
		dev, _ := gen1SettingsServer(t, live, gen1NoActionsJSON)
		result, err := RestoreGen1(t.Context(), dev, &Backup{Config: json.RawMessage(source)},
			&Gen1RestoreOptions{SkipNetwork: true, SkipClockWait: true})
		if err != nil {
			t.Fatalf("RestoreGen1: %v", err)
		}
		joined := strings.Join(result.Warnings, "\n")
		if !strings.Contains(joined, "not applied: discoverable: backup has true, device has false") {
			t.Fatalf("connected=%t: missing mismatch warning in:\n%s", connected, joined)
		}
		if got := strings.Contains(joined, "Shelly cloud keeps its own value"); got != wantNote {
			t.Errorf("connected=%t: cloud note present = %t, want %t:\n%s", connected, got, wantNote, joined)
		}
	}
}

// A key this code has never seen still transfers.
func TestGen1Restore_WritesUnclassifiedSetting(t *testing.T) {
	t.Parallel()
	raw := `{"fw":"` + gen1TestFW + `","future_setting":7,"future_block":{"level":"high"}}`
	live := `{"fw":"` + gen1TestFW + `","future_setting":1,"future_block":{"level":"low"}}`
	dev, writes := gen1SettingsServer(t, live, gen1NoActionsJSON)
	if _, err := RestoreGen1(t.Context(), dev, &Backup{Config: json.RawMessage(raw)},
		&Gen1RestoreOptions{SkipNetwork: true, SkipClockWait: true}); err != nil {
		t.Fatalf("RestoreGen1: %v", err)
	}
	sent := strings.Join(writes(), "\n")
	for _, want := range []string{"/settings?future_setting=7", "/settings/future_block?level=high"} {
		if !strings.Contains(sent, want) {
			t.Errorf("missing write %q in:\n%s", want, sent)
		}
	}
}

func TestExportGen1_KeepsEverySettingAndTheActions(t *testing.T) {
	t.Parallel()
	settings := `{"fw":"1.0","transition":150,"future_setting":7,"lights":[{"transition":150}]}`
	actions := `{"actions":{"out_on_url":[{"index":0,"urls":["http://localhost/x"],"enabled":true}]}}`
	dev, _ := gen1SettingsServer(t, settings, actions)
	bkp, err := ExportGen1(t.Context(), dev)
	if err != nil {
		t.Fatalf("ExportGen1: %v", err)
	}
	if string(bkp.Config) != settings {
		t.Errorf("Config = %s, want the device response verbatim", bkp.Config)
	}
	var got gen1.ActionSettings
	if err := json.Unmarshal(bkp.Webhooks, &got); err != nil {
		t.Fatalf("parse stored actions: %v", err)
	}
	if len(got.Actions) != 1 || got.Actions[0].Event != gen1.ActionOutputOnUrl ||
		!got.Actions[0].Enabled || got.Actions[0].URLs[0] != "http://localhost/x" {
		t.Errorf("stored actions = %+v", got.Actions)
	}
}

func TestExportGen1_FailsWhenActionsCannotBeRead(t *testing.T) {
	t.Parallel()
	dev, _ := gen1SettingsServer(t, `{"fw":"1.0"}`, `{"actions":[]}`)
	if _, err := ExportGen1(t.Context(), dev); err == nil {
		t.Fatal("ExportGen1 succeeded with an unreadable actions response")
	}
}
