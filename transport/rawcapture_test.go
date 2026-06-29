package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/tj-smith47/shelly-go/types"
)

// TestRawCapture_REST proves a Gen1 REST response body is captured verbatim.
func TestRawCapture_REST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ison":true}`))
	}))
	defer server.Close()

	var sink RawCapture
	ctx := WithRawCapture(context.Background(), &sink)

	tr := NewHTTP(server.URL)
	if _, err := tr.Call(ctx, NewSimpleRequest("/relay/0")); err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	got := sink.Responses()
	if len(got) != 1 {
		t.Fatalf("captured %d responses, want 1", len(got))
	}
	if strings.TrimSpace(string(got[0])) != `{"ison":true}` {
		t.Errorf("captured body = %q, want %q", got[0], `{"ison":true}`)
	}
}

// TestRawCapture_RPC proves a Gen2 RPC response is captured as the verbatim
// JSON-RPC envelope (the exact bytes the device returned), not just the result.
func TestRawCapture_RPC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(types.Response{
			ID:     1,
			Result: json.RawMessage(`{"was_on":true}`),
		})
	}))
	defer server.Close()

	var sink RawCapture
	ctx := WithRawCapture(context.Background(), &sink)

	tr := NewHTTP(server.URL)
	if _, err := tr.Call(ctx, newTestRPCRequest("Switch.Toggle", nil)); err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	got := sink.Responses()
	if len(got) != 1 {
		t.Fatalf("captured %d responses, want 1", len(got))
	}
	var env types.Response
	if err := json.Unmarshal(got[0], &env); err != nil {
		t.Fatalf("captured body is not a JSON-RPC envelope: %v", err)
	}
	if strings.TrimSpace(string(env.Result)) != `{"was_on":true}` {
		t.Errorf("envelope result = %q, want %q", env.Result, `{"was_on":true}`)
	}
}

// TestRawCapture_NilSinkIsNoOp confirms WithRawCapture(ctx, nil) leaves the
// context untouched and the call still succeeds (callers can wire it blindly).
func TestRawCapture_NilSinkIsNoOp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ison":true}`))
	}))
	defer server.Close()

	ctx := WithRawCapture(context.Background(), nil)
	if ctx != context.Background() {
		t.Error("WithRawCapture(ctx, nil) should return the original context unchanged")
	}

	tr := NewHTTP(server.URL)
	if _, err := tr.Call(ctx, NewSimpleRequest("/relay/0")); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
}

// TestRawCapture_NoSinkInstalled confirms captureRaw is a no-op when no sink is
// present on the context — the common, capture-disabled hot path.
func TestRawCapture_NoSinkInstalled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ison":true}`))
	}))
	defer server.Close()

	var sink RawCapture // created but never installed on the context

	tr := NewHTTP(server.URL)
	if _, err := tr.Call(context.Background(), NewSimpleRequest("/relay/0")); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if got := sink.Responses(); len(got) != 0 {
		t.Errorf("uninstalled sink captured %d responses, want 0", len(got))
	}
}

// TestRawCapture_AccumulatesInOrder proves multiple sequential calls under one
// sink are recorded in call order.
func TestRawCapture_AccumulatesInOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"path":"` + r.URL.Path + `"}`))
	}))
	defer server.Close()

	var sink RawCapture
	ctx := WithRawCapture(context.Background(), &sink)
	tr := NewHTTP(server.URL)

	for _, p := range []string{"/a", "/b", "/c"} {
		if _, err := tr.Call(ctx, NewSimpleRequest(p)); err != nil {
			t.Fatalf("Call(%s) error = %v", p, err)
		}
	}

	got := sink.Responses()
	want := []string{`{"path":"/a"}`, `{"path":"/b"}`, `{"path":"/c"}`}
	if len(got) != len(want) {
		t.Fatalf("captured %d responses, want %d", len(got), len(want))
	}
	for i, w := range want {
		if strings.TrimSpace(string(got[i])) != w {
			t.Errorf("response[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// TestRawCapture_ConcurrentBatch proves one sink shared across concurrent calls
// (the group-batch case) records every response without a data race.
func TestRawCapture_ConcurrentBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var sink RawCapture
	ctx := WithRawCapture(context.Background(), &sink)
	tr := NewHTTP(server.URL)

	const n = 16
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			_, _ = tr.Call(ctx, NewSimpleRequest("/status"))
		})
	}
	wg.Wait()

	if got := sink.Responses(); len(got) != n {
		t.Errorf("captured %d responses, want %d", len(got), n)
	}
}
