package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tj-smith47/shelly-go/internal/authtest"
	"github.com/tj-smith47/shelly-go/types"
)

const testPassword = "s3cret"

func TestAuthMethod_String(t *testing.T) {
	tests := []struct {
		want   string
		method AuthMethod
	}{
		{method: AuthMethodNone, want: "none"},
		{method: AuthMethodBasic, want: "basic"},
		{method: AuthMethodDigest, want: "digest"},
		{method: AuthMethodRPC, want: "rpc"},
		{method: AuthMethod(999), want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.method.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasicAuth_PasswordNeverSent(t *testing.T) {
	auth := BasicAuth("admin", "password")
	if auth.Username != "admin" || auth.Password != "password" {
		t.Fatalf("BasicAuth() = %+v", auth)
	}
	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "password") {
		t.Errorf("auth object %s carries the password", data)
	}
}

// TestDigestAuth_DeviceAccepts checks DigestAuth's frame against a verifier
// written from Shelly's documentation, for both nonce types.
func TestDigestAuth_DeviceAccepts(t *testing.T) {
	nonces := map[string]any{
		"string nonce (fw >= 2.0.0)": "AAAAAABnZWVrc2Zvcmdl",
		"numeric nonce (fw < 2.0.0)": 1625214011,
	}
	for name, nonce := range nonces {
		t.Run(name, func(t *testing.T) {
			auth, err := DigestAuth("", testPassword, authtest.Realm, nonce)
			if err != nil {
				t.Fatalf("DigestAuth() error = %v", err)
			}
			frame, err := json.Marshal(auth)
			if err != nil {
				t.Fatal(err)
			}
			nc, err := authtest.FrameAuth(frame, authtest.Realm, nonce, testPassword)
			if err != nil {
				t.Fatalf("device rejects %s: %v", frame, err)
			}
			if nc != "00000001" {
				t.Errorf("nc = %q, want 00000001", nc)
			}
			if err := ValidateAuthData(auth); err != nil {
				t.Errorf("ValidateAuthData() = %v", err)
			}
		})
	}
}

func TestDigestAuth_RawMessageNonce(t *testing.T) {
	auth, err := DigestAuth("admin", testPassword, authtest.Realm, json.RawMessage(`42`))
	if err != nil {
		t.Fatal(err)
	}
	frame, err := json.Marshal(auth)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authtest.FrameAuth(frame, authtest.Realm, 42, testPassword); err != nil {
		t.Errorf("device rejects %s: %v", frame, err)
	}
}

func TestDigestAuth_WrongPassword(t *testing.T) {
	auth, err := DigestAuth("admin", "wrong", authtest.Realm, "n1")
	if err != nil {
		t.Fatal(err)
	}
	frame, err := json.Marshal(auth)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authtest.FrameAuth(frame, authtest.Realm, "n1", testPassword); err == nil {
		t.Error("device accepted a frame built with the wrong password")
	}
}

func TestDigestAuthFromHA1(t *testing.T) {
	ha1 := CalculateHA1("admin", testPassword, authtest.Realm)
	if ha1 != authtest.HA1(authtest.Realm, testPassword) {
		t.Fatalf("CalculateHA1() = %s, want SHA256(admin:realm:password)", ha1)
	}
	auth, err := DigestAuthFromHA1("admin", ha1, authtest.Realm, "n2")
	if err != nil {
		t.Fatal(err)
	}
	frame, err := json.Marshal(auth)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authtest.FrameAuth(frame, authtest.Realm, "n2", testPassword); err != nil {
		t.Errorf("device rejects %s: %v", frame, err)
	}
}

func TestDigestAuth_InvalidInput(t *testing.T) {
	tests := map[string]struct {
		nonce any
		realm string
	}{
		"no realm":         {realm: "", nonce: "n"},
		"empty nonce":      {realm: authtest.Realm, nonce: ""},
		"object nonce":     {realm: authtest.Realm, nonce: map[string]int{"a": 1}},
		"unmarshalable":    {realm: authtest.Realm, nonce: func() {}},
		"bool nonce":       {realm: authtest.Realm, nonce: true},
		"null nonce (nil)": {realm: authtest.Realm, nonce: nil},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := DigestAuth("admin", testPassword, tt.realm, tt.nonce)
			if !errors.Is(err, types.ErrInvalidParam) {
				t.Errorf("DigestAuth() error = %v, want ErrInvalidParam", err)
			}
		})
	}
}

func TestDigestAuth_FreshCNonce(t *testing.T) {
	a, err := DigestAuth("admin", testPassword, authtest.Realm, "n")
	if err != nil {
		t.Fatal(err)
	}
	b, err := DigestAuth("admin", testPassword, authtest.Realm, "n")
	if err != nil {
		t.Fatal(err)
	}
	if a.CNonce == 0 || a.CNonce == b.CNonce {
		t.Errorf("cnonces %d and %d, want two different non-zero values", a.CNonce, b.CNonce)
	}
}

func TestValidateAuthData(t *testing.T) {
	valid := func() *AuthData {
		return &AuthData{
			Username: "admin", Realm: authtest.Realm, Nonce: json.RawMessage(`"n"`),
			CNonce: 7, NC: "00000001", Algorithm: AlgorithmSHA256, Response: "r",
		}
	}
	tests := []struct {
		auth    *AuthData
		name    string
		wantErr bool
	}{
		{name: "nil", auth: nil, wantErr: true},
		{name: "no username", auth: &AuthData{Password: "p"}, wantErr: true},
		{name: "username and password", auth: &AuthData{Username: "admin", Password: "p"}},
		{name: "username without password", auth: &AuthData{Username: "admin"}, wantErr: true},
		{name: "documented frame", auth: valid()},
		{name: "no realm", auth: func() *AuthData { a := valid(); a.Realm = ""; return a }(), wantErr: true},
		{name: "no nonce", auth: func() *AuthData { a := valid(); a.Nonce = nil; return a }(), wantErr: true},
		{name: "no cnonce", auth: func() *AuthData { a := valid(); a.CNonce = 0; return a }(), wantErr: true},
		{name: "no nc", auth: func() *AuthData { a := valid(); a.NC = ""; return a }(), wantErr: true},
		{name: "md5", auth: func() *AuthData { a := valid(); a.Algorithm = "MD5"; return a }(), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateAuthData(tt.auth); (err != nil) != tt.wantErr {
				t.Errorf("ValidateAuthData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewHTTPClient_DigestAuth runs a call through a fake device that
// requires the documented HTTP digest header.
func TestNewHTTPClient_DigestAuth(t *testing.T) {
	const nonce = "AAAAAABnZWVrc2Zvcmdl"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := authtest.HeaderAuth(r, authtest.Realm, nonce, testPassword); err != nil {
			w.Header().Set("WWW-Authenticate", authtest.HeaderChallenge(authtest.Realm, nonce))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"id":1,"result":{"ok":true}}`))
	}))
	defer srv.Close()

	client, err := NewHTTPClient(srv.URL, WithDigestAuth("admin", testPassword))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Call(context.Background(), "Shelly.GetStatus", nil); err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	wrong, err := NewHTTPClient(srv.URL, WithDigestAuth("admin", "wrong"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrong.Call(context.Background(), "Shelly.GetStatus", nil); !errors.Is(err, types.ErrAuth) {
		t.Errorf("Call() with a wrong password error = %v, want ErrAuth", err)
	}
}
