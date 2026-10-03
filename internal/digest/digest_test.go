package digest

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tj-smith47/shelly-go/internal/authtest"
)

const password = "s3cret"

func TestParseFrameChallenge(t *testing.T) {
	for _, nonce := range []any{"AAAAAABnZWVrc2Zvcmdl", 1625214011} {
		ch, err := ParseFrameChallenge(authtest.FrameChallenge(authtest.Realm, nonce))
		if err != nil {
			t.Fatalf("ParseFrameChallenge(%v) error = %v", nonce, err)
		}
		want, err := json.Marshal(nonce)
		if err != nil {
			t.Fatal(err)
		}
		if ch.Realm != authtest.Realm || string(ch.Nonce) != string(want) {
			t.Errorf("challenge = %+v", ch)
		}
	}
}

func TestParseFrameChallenge_Invalid(t *testing.T) {
	messages := []string{
		`not json`,
		`{"auth_type":"basic","nonce":"n","realm":"r","algorithm":"SHA-256"}`,
		`{"auth_type":"digest","nonce":"n","realm":"r","algorithm":"MD5"}`,
		`{"auth_type":"digest","nonce":"n","algorithm":"SHA-256"}`,
		`{"auth_type":"digest","realm":"r","algorithm":"SHA-256"}`,
		`{"auth_type":"digest","nonce":"","realm":"r","algorithm":"SHA-256"}`,
		`{"auth_type":"digest","nonce":true,"realm":"r","algorithm":"SHA-256"}`,
	}
	for _, m := range messages {
		if _, err := ParseFrameChallenge(m); !errors.Is(err, ErrChallenge) {
			t.Errorf("ParseFrameChallenge(%s) error = %v, want ErrChallenge", m, err)
		}
	}
}

func TestParseHeader(t *testing.T) {
	ch, err := ParseHeader(authtest.HeaderChallenge(authtest.Realm, "abc"))
	if err != nil {
		t.Fatal(err)
	}
	if ch.Realm != authtest.Realm || string(ch.Nonce) != `"abc"` {
		t.Errorf("challenge = %+v", ch)
	}
	for _, h := range []string{"", `Basic realm="r"`, `Digest realm="r", nonce="n", algorithm=MD5`, `Digest realm="r", algorithm=SHA-256`} {
		if _, err := ParseHeader(h); !errors.Is(err, ErrChallenge) {
			t.Errorf("ParseHeader(%q) error = %v, want ErrChallenge", h, err)
		}
	}
}

// TestSession_Frame checks frames against the documented verifier, and that
// the nonce count rises and restarts with a new challenge.
func TestSession_Frame(t *testing.T) {
	s := NewSession("", password)
	if s.Ready() {
		t.Fatal("Ready() before any challenge")
	}
	if _, err := s.Frame(); err == nil {
		t.Fatal("Frame() before any challenge succeeded")
	}
	if _, err := s.Header("POST", "/rpc"); err == nil {
		t.Fatal("Header() before any challenge succeeded")
	}

	for _, nonce := range []any{1625214011, "AAAAAABnZWVrc2Zvcmdl"} {
		ch, err := ParseFrameChallenge(authtest.FrameChallenge(authtest.Realm, nonce))
		if err != nil {
			t.Fatal(err)
		}
		s.Accept(ch)
		for i, wantNC := range []string{"00000001", "00000002"} {
			f, err := s.Frame()
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(f)
			if err != nil {
				t.Fatal(err)
			}
			nc, err := authtest.FrameAuth(data, authtest.Realm, nonce, password)
			if err != nil || nc != wantNC {
				t.Errorf("frame %d for nonce %v: nc %s, %v; want nc %s accepted (%s)", i, nonce, nc, err, wantNC, data)
			}
		}
	}

	wrong := NewSession(User, "wrong")
	ch, err := ParseFrameChallenge(authtest.FrameChallenge(authtest.Realm, "n"))
	if err != nil {
		t.Fatal(err)
	}
	wrong.Accept(ch)
	f, err := wrong.Frame()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authtest.FrameAuth(data, authtest.Realm, "n", password); err == nil {
		t.Error("frame built with the wrong password accepted")
	}
}

func TestSession_Header(t *testing.T) {
	s := NewSession(User, password)
	ch, err := ParseHeader(authtest.HeaderChallenge(authtest.Realm, "abc"))
	if err != nil {
		t.Fatal(err)
	}
	s.Accept(ch)
	r := httptest.NewRequest(http.MethodPost, "/rpc?x=1", http.NoBody)
	h, err := s.Header(r.Method, r.URL.RequestURI())
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", h)
	if nc, err := authtest.HeaderAuth(r, authtest.Realm, "abc", password); err != nil || nc != "00000001" {
		t.Errorf("header %s: nc %s, %v", h, nc, err)
	}
}

func TestNewFrame_BadNonce(t *testing.T) {
	if _, err := NewFrame(User, "ha1", "r", json.RawMessage(`{}`), 1); !errors.Is(err, ErrChallenge) {
		t.Errorf("NewFrame() error = %v, want ErrChallenge", err)
	}
}
