package backup

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

type recordedCall struct {
	method string
	params json.RawMessage
}

func newRecordingClient(calls *[]recordedCall) *rpc.Client {
	tr := &mockTransport{
		callFunc: func(_ context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			*calls = append(*calls, recordedCall{method: req.GetMethod(), params: req.GetParams()})
			return jsonrpcResponse(`{}`)
		},
	}
	return rpc.NewClient(tr)
}

func methodSet(calls []recordedCall) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(calls))
	for _, c := range calls {
		out[c.method] = c.params
	}
	return out
}

func paramHasID(t *testing.T, params json.RawMessage) bool {
	t.Helper()
	var p struct {
		ID *int `json:"id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		t.Fatalf("unmarshal params %s: %v", params, err)
	}
	return p.ID != nil
}

func TestRestore_AppliesFullDeviceConfig(t *testing.T) {
	config := `{
		"sys": {"device": {"name": "Bath"}},
		"wifi": {"sta": {"ssid": "x"}},
		"eth": {"ipv4mode": "dhcp"},
		"ws": {"enable": false},
		"switch:0": {"id": 0, "name": "Relay"},
		"input:1": {"id": 1, "type": "switch"}
	}`
	data, err := json.Marshal(&Backup{Version: BackupVersion, Config: json.RawMessage(config)})
	if err != nil {
		t.Fatalf("marshal backup: %v", err)
	}

	var calls []recordedCall
	mgr := New(newRecordingClient(&calls))

	opts := DefaultRestoreOptions()
	opts.RestoreWiFi = true // allow eth (network)

	if _, err := mgr.Restore(context.Background(), data, opts); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	methods := methodSet(calls)
	for _, want := range []string{"Sys.SetConfig", "Eth.SetConfig", "Ws.SetConfig", "Switch.SetConfig", "Input.SetConfig"} {
		if _, ok := methods[want]; !ok {
			t.Errorf("expected %s to be called", want)
		}
	}

	// WiFi must NOT be applied via the generic device-config path; it is owned by
	// the dedicated WiFi field / restoreOptionalConfigs.
	if _, ok := methods["WiFi.SetConfig"]; ok {
		t.Error("WiFi.SetConfig should be skipped in device-config restore")
	}

	// Instanced components carry a top-level id; singletons do not.
	if !paramHasID(t, methods["Switch.SetConfig"]) {
		t.Error("Switch.SetConfig should include id")
	}
	if paramHasID(t, methods["Sys.SetConfig"]) {
		t.Error("Sys.SetConfig should not include id")
	}
}

func TestRestore_RespectsComponentAndNetworkGates(t *testing.T) {
	config := `{"sys": {}, "eth": {}, "switch:0": {"id": 0}}`
	data, err := json.Marshal(&Backup{Version: BackupVersion, Config: json.RawMessage(config)})
	if err != nil {
		t.Fatalf("marshal backup: %v", err)
	}

	var calls []recordedCall
	mgr := New(newRecordingClient(&calls))

	opts := DefaultRestoreOptions()
	opts.RestoreComponents = false
	opts.RestoreWiFi = false // eth gated off

	if _, err := mgr.Restore(context.Background(), data, opts); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	methods := methodSet(calls)
	if _, ok := methods["Sys.SetConfig"]; !ok {
		t.Error("Sys config should always be restored")
	}
	if _, ok := methods["Switch.SetConfig"]; ok {
		t.Error("components should be skipped when RestoreComponents is false")
	}
	if _, ok := methods["Eth.SetConfig"]; ok {
		t.Error("eth should be skipped when RestoreWiFi is false")
	}
}

func TestShouldRestoreConfigKey(t *testing.T) {
	opts := DefaultRestoreOptions()
	opts.RestoreWiFi = true
	opts.RestoreComponents = true

	cases := map[string]bool{
		"sys":      true,
		"ws":       true,
		"eth":      true,
		"switch:0": true,
		"input:1":  true,
		"wifi":     false,
		"cloud":    false,
		"ble":      false,
		"mqtt":     false,
	}
	for key, want := range cases {
		if got := shouldRestoreConfigKey(key, opts); got != want {
			t.Errorf("shouldRestoreConfigKey(%q) = %v, want %v", key, got, want)
		}
	}
}
