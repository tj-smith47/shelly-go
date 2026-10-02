//go:build linux

package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
)

func TestWithBindInterface_LoopbackSucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	h := NewHTTP(srv.URL, WithBindInterface("lo"), WithRetry(0, 0))
	_, err := h.Get(context.Background(), "/status")
	if errors.Is(err, syscall.EPERM) {
		t.Skip("SO_BINDTODEVICE needs CAP_NET_RAW")
	}
	if err != nil {
		t.Fatalf("request bound to lo failed: %v", err)
	}
}

func TestWithBindInterface_UnknownInterfaceFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	h := NewHTTP(srv.URL, WithBindInterface("nosuch0"), WithRetry(0, 0))
	_, err := h.Get(context.Background(), "/status")
	if errors.Is(err, syscall.EPERM) {
		t.Skip("SO_BINDTODEVICE needs CAP_NET_RAW")
	}
	if !errors.Is(err, syscall.ENODEV) {
		t.Errorf("err = %v, want ENODEV for an interface that does not exist", err)
	}
}
