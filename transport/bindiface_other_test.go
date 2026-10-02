//go:build !linux

package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tj-smith47/shelly-go/types"
)

func TestWithBindInterface_NotSupported(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	h := NewHTTP(srv.URL, WithBindInterface("en0"))
	_, err := h.Get(context.Background(), "/status")
	if !errors.Is(err, types.ErrNotSupported) {
		t.Errorf("err = %v, want wrapping types.ErrNotSupported", err)
	}
	if hits != 0 {
		t.Errorf("server saw %d requests, want none", hits)
	}
}
