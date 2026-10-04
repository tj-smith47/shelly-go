package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/tj-smith47/shelly-go/internal/authtest"
	"github.com/tj-smith47/shelly-go/types"
)

const digestTestPassword = "s3cret"

// digestHTTPDevice is a fake Gen2+ device that refuses a frame a device
// cannot parse and checks the Authorization header, both with authtest.
type digestHTTPDevice struct {
	nonce      string
	challenge  string
	ncs        []string
	mu         sync.Mutex
	requests   int
	challenges int
}

func (d *digestHTTPDevice) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.requests++
	if _, ok := authtest.ReadHTTP(w, r); !ok {
		return
	}
	nc, err := authtest.HeaderAuth(r, authtest.Realm, d.nonce, digestTestPassword)
	if err != nil {
		d.challenges++
		challenge := d.challenge
		if challenge == "" {
			challenge = authtest.HeaderChallenge(authtest.Realm, d.nonce)
		}
		w.Header().Set("WWW-Authenticate", challenge)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	d.ncs = append(d.ncs, nc)
	_, _ = w.Write([]byte(`{"id":1,"result":{}}`))
}

func (d *digestHTTPDevice) rotate(nonce string) {
	d.mu.Lock()
	d.nonce = nonce
	d.mu.Unlock()
}

func newDigestHTTP(t *testing.T, d *digestHTTPDevice, password string) *HTTP {
	t.Helper()
	srv := httptest.NewServer(d)
	t.Cleanup(srv.Close)
	return NewHTTP(srv.URL, WithDigestAuth("admin", password), WithRetry(0, 0))
}

func TestHTTP_DigestAuth_ReusesNonce(t *testing.T) {
	d := &digestHTTPDevice{nonce: "AAAAAABnZWVrc2Zvcmdl"}
	h := newDigestHTTP(t, d, digestTestPassword)

	for range 3 {
		if _, err := h.Call(context.Background(), newTestRPCRequest("Shelly.GetStatus", nil)); err != nil {
			t.Fatalf("Call() error = %v", err)
		}
	}
	if d.challenges != 1 || d.requests != 4 {
		t.Errorf("requests/challenges = %d/%d, want 4/1 (one challenge, then the nonce is reused)", d.requests, d.challenges)
	}
	want := []string{"00000001", "00000002", "00000003"}
	for i, nc := range d.ncs {
		if nc != want[i] {
			t.Errorf("nc[%d] = %s, want %s", i, nc, want[i])
		}
	}
}

func TestHTTP_DigestAuth_StaleNonce(t *testing.T) {
	d := &digestHTTPDevice{nonce: "n1"}
	h := newDigestHTTP(t, d, digestTestPassword)

	if _, err := h.Call(context.Background(), newTestRPCRequest("Shelly.GetStatus", nil)); err != nil {
		t.Fatalf("first Call() error = %v", err)
	}
	d.rotate("n2")
	if _, err := h.Call(context.Background(), newTestRPCRequest("Shelly.GetStatus", nil)); err != nil {
		t.Fatalf("Call() after the nonce changed error = %v", err)
	}
	if d.challenges != 2 || d.ncs[1] != "00000001" {
		t.Errorf("challenges = %d, nc after new nonce = %s; want 2 and 00000001", d.challenges, d.ncs[1])
	}
}

func TestHTTP_DigestAuth_Rejected(t *testing.T) {
	tests := map[string]struct {
		challenge string
		password  string
	}{
		"wrong password": {password: "wrong"},
		"md5 challenge":  {password: digestTestPassword, challenge: `Digest qop="auth", realm="r", nonce="n", algorithm=MD5`},
		"not digest":     {password: digestTestPassword, challenge: `Basic realm="r"`},
		"no realm":       {password: digestTestPassword, challenge: `Digest nonce="n", algorithm=SHA-256`},
		"no nonce":       {password: digestTestPassword, challenge: `Digest realm="r", algorithm=SHA-256`},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			d := &digestHTTPDevice{nonce: "n", challenge: tt.challenge}
			h := newDigestHTTP(t, d, tt.password)
			_, err := h.Call(context.Background(), newTestRPCRequest("Shelly.GetStatus", nil))
			if !errors.Is(err, types.ErrAuth) {
				t.Errorf("Call() error = %v, want ErrAuth", err)
			}
			if d.requests > 2 {
				t.Errorf("requests = %d, want at most 2", d.requests)
			}
		})
	}
}

func TestHTTP_NoCredentials_401(t *testing.T) {
	d := &digestHTTPDevice{nonce: "n"}
	srv := httptest.NewServer(d)
	defer srv.Close()

	_, err := NewHTTP(srv.URL, WithRetry(0, 0)).Call(context.Background(), newTestRPCRequest("Shelly.GetStatus", nil))
	if !errors.Is(err, types.ErrAuth) || d.requests != 1 {
		t.Errorf("Call() error = %v after %d requests, want ErrAuth after 1", err, d.requests)
	}
}

func TestHTTP_DigestAuth_NoAuthNeeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("credentials sent to a device that never asked for them")
		}
		_, _ = w.Write([]byte(`{"id":1,"result":{}}`))
	}))
	defer srv.Close()

	h := NewHTTP(srv.URL, WithDigestAuth("admin", digestTestPassword))
	if _, err := h.Call(context.Background(), newTestRPCRequest("Shelly.GetStatus", nil)); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
}
