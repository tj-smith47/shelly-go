package authtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The accepted frame is built from the documented formula
// response = SHA256(ha1:nonce:nc:cnonce:auth:ha2); each mutation breaks one
// documented rule.
func TestFrameAuth_Rejects(t *testing.T) {
	good := func() map[string]any {
		return map[string]any{
			"realm": Realm, "username": "admin", "nonce": "n", "cnonce": 7, "nc": "00000001",
			"algorithm": "SHA-256",
			"response":  sha256Hex(HA1(Realm, "pw") + ":n:00000001:7:auth:" + sha256Hex("dummy_method:dummy_uri")),
		}
	}
	marshal := func(m map[string]any) json.RawMessage {
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	if _, err := FrameAuth(marshal(good()), Realm, "n", "pw"); err != nil {
		t.Fatalf("good frame rejected: %v", err)
	}
	mutations := map[string]func(map[string]any){
		"string cnonce": func(m map[string]any) { m["cnonce"] = "7" },
		"numeric nonce": func(m map[string]any) { m["nonce"] = 1 },
		"numeric nc":    func(m map[string]any) { m["nc"] = 1 },
		"md5":           func(m map[string]any) { m["algorithm"] = "MD5" },
		"other user":    func(m map[string]any) { m["username"] = "root" },
		"bad response":  func(m map[string]any) { m["response"] = "00" },
	}
	for name, mutate := range mutations {
		m := good()
		mutate(m)
		if _, err := FrameAuth(marshal(m), Realm, "n", "pw"); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := FrameAuth(json.RawMessage(`[`), Realm, "n", "pw"); err == nil {
		t.Error("malformed auth object accepted")
	}
	if _, err := FrameAuth(marshal(good()), Realm, func() {}, "pw"); err == nil {
		t.Error("unmarshalable nonce accepted")
	}
	if got := FrameChallenge(Realm, func() {}); got == "" {
		t.Error("FrameChallenge returned nothing for a bad nonce")
	}
}

func TestHeaderAuth_Rejects(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/rpc", http.NoBody)
	if _, err := HeaderAuth(r, Realm, "n", "pw"); err == nil {
		t.Error("request without a header accepted")
	}
	r.Header.Set("Authorization", `Digest username="admin", realm="`+Realm+`", nonce="n", uri="/rpc", qop=auth, algorithm=SHA-256, nc=1, cnonce="c", response="x"`)
	if _, err := HeaderAuth(r, Realm, "n", "pw"); err == nil {
		t.Error("short nc accepted")
	}
	r.Header.Set("Authorization", `Digest username="admin", realm="`+Realm+`", nonce="n", uri="/rpc", qop=auth, algorithm=SHA-256, nc=00000001, cnonce="c", response="x"`)
	if _, err := HeaderAuth(r, Realm, "n", "pw"); err == nil {
		t.Error("wrong response accepted")
	}
	r.Header.Set("Authorization", `Digest username="admin", realm="other", nonce="n"`)
	if _, err := HeaderAuth(r, Realm, "n", "pw"); err == nil {
		t.Error("wrong realm accepted")
	}
}

func TestFrameError(t *testing.T) {
	refused := []string{``, `[`, `null`, `[]`, `{"id":1,"auth":null}`, `{"id":1,"auth":"x"}`, `{"id":1,"auth":[]}`}
	for _, frame := range refused {
		e := FrameError([]byte(frame))
		if e == nil || e["code"] != http.StatusBadRequest || e["message"] != BadRequestMessage {
			t.Errorf("FrameError(%q) = %v, want code 400 %q", frame, e, BadRequestMessage)
		}
	}
	for _, frame := range []string{`{"id":1,"method":"Shelly.GetStatus"}`, `{"id":1,"auth":{"realm":"r"}}`} {
		if e := FrameError([]byte(frame)); e != nil {
			t.Errorf("FrameError(%q) = %v, want nil", frame, e)
		}
	}
}

func TestReadHTTP(t *testing.T) {
	read := func(body string) (*httptest.ResponseRecorder, bool) {
		w := httptest.NewRecorder()
		_, ok := ReadHTTP(w, httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body)))
		return w, ok
	}
	if w, ok := read(""); ok || w.Code != http.StatusBadRequest {
		t.Errorf("empty body: ok=%v status=%d, want HTTP 400", ok, w.Code)
	}
	w, ok := read(`{"id":7,"method":"Shelly.GetStatus","auth":null}`)
	if ok || !strings.Contains(w.Body.String(), `"id":7`) || !strings.Contains(w.Body.String(), `"code":400`) {
		t.Errorf(`"auth": null: ok=%v body=%s, want error 400 for id 7`, ok, w.Body)
	}
	if _, ok := read(`{"id":1,"method":"Shelly.GetStatus"}`); !ok {
		t.Error("a frame without auth was refused")
	}
}

// The challenge a Plus 2PM on firmware 1.7.5 sent, numeric nonce and nc.
func TestFrameChallenge_DeviceShape(t *testing.T) {
	var got map[string]any
	if err := json.Unmarshal([]byte(FrameChallenge("shellyplus2pm-c049ef86ac10", 1791087502)), &got); err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal([]byte(`{"auth_type": "digest", "nonce": 1791087502, "nc": 1,`+
		` "realm": "shellyplus2pm-c049ef86ac10", "algorithm": "SHA-256"}`), &want); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("FrameChallenge = %v, want %v", got, want)
	}
}
