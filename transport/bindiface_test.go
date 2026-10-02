package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tj-smith47/shelly-go/types"
)

func TestWithBindInterface_EmptyIsNoOp(t *testing.T) {
	h := NewHTTP("http://127.0.0.1", WithBindInterface(""))
	tr, ok := h.client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("transport is not *http.Transport")
	}
	if tr.DialContext != nil {
		t.Error("empty interface name must not install a dialer")
	}
}

func TestWithBindInterface_SetsDialer(t *testing.T) {
	h := NewHTTP("http://127.0.0.1", WithBindInterface("lo"))
	if h.opts.bindIface != "lo" {
		t.Errorf("bindIface = %q, want lo", h.opts.bindIface)
	}
	tr, ok := h.client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("transport is not *http.Transport")
	}
	if tr.DialContext == nil {
		t.Error("expected a bound dialer")
	}
}

func TestWithBindInterface_RejectsCustomClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("request must not be sent over a client the bind cannot reach")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	h := NewHTTP(srv.URL, WithBindInterface("lo"), WithClient(srv.Client()))
	_, err := h.Get(context.Background(), "/status")
	if !errors.Is(err, types.ErrInvalidParam) {
		t.Errorf("err = %v, want wrapping types.ErrInvalidParam", err)
	}
}

func TestShouldRetry_NotSupportedIsPermanent(t *testing.T) {
	h := NewHTTP("http://127.0.0.1")
	if h.shouldRetry(fmt.Errorf("dial: %w", types.ErrNotSupported)) {
		t.Error("ErrNotSupported must not be retried")
	}
}
