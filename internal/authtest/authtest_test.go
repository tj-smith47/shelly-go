package authtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
