package transport_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/tj-smith47/shelly-go/internal/authtest"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

// frameDevice records every request frame and answers it as a device would:
// error 400 for a frame authtest.FrameError refuses, an empty result otherwise.
type frameDevice struct {
	frames [][]byte
	gets   []string
	mu     sync.Mutex
}

func (d *frameDevice) answer(frame []byte) []byte {
	d.mu.Lock()
	d.frames = append(d.frames, frame)
	d.mu.Unlock()
	var f struct {
		ID any `json:"id"`
	}
	_ = json.Unmarshal(frame, &f)
	reply := map[string]any{"id": f.ID, "src": authtest.Realm, "result": map[string]any{}}
	if e := authtest.FrameError(frame); e != nil {
		reply = map[string]any{"id": f.ID, "src": authtest.Realm, "error": e}
	}
	data, _ := json.Marshal(reply)
	return data
}

func (d *frameDevice) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		d.mu.Lock()
		d.gets = append(d.gets, r.URL.Path)
		d.mu.Unlock()
		_, _ = w.Write([]byte(`{}`))
		return
	}
	body, ok := authtest.ReadHTTP(w, r)
	if !ok {
		d.mu.Lock()
		d.frames = append(d.frames, body)
		d.mu.Unlock()
		return
	}
	_, _ = w.Write(d.answer(body))
}

func (d *frameDevice) serveWS(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		_, frame, err := conn.ReadMessage()
		if err != nil || conn.WriteMessage(websocket.TextMessage, d.answer(frame)) != nil {
			return
		}
	}
}

func (d *frameDevice) recorded() [][]byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([][]byte(nil), d.frames...)
}

// typedNilRequest is a request whose optional values are typed nils: each
// would marshal as JSON null if a transport put it in the frame.
type typedNilRequest struct{}

func (typedNilRequest) GetID() any                 { return (*int64)(nil) }
func (typedNilRequest) GetMethod() string          { return "Shelly.GetStatus" }
func (typedNilRequest) GetParams() json.RawMessage { return json.RawMessage(`null`) }
func (typedNilRequest) GetAuth() any               { return (*rpc.AuthData)(nil) }
func (typedNilRequest) GetJSONRPC() string         { return "" }
func (typedNilRequest) IsREST() bool               { return false }

type frameTransport struct {
	tr     transport.Transport
	frames func() [][]byte
	gets   func() []string
}

func newFrameTransports(t *testing.T) map[string]frameTransport {
	t.Helper()
	httpDev, wsDev := &frameDevice{}, &frameDevice{}
	httpSrv := httptest.NewServer(httpDev)
	wsSrv := httptest.NewServer(http.HandlerFunc(wsDev.serveWS))
	t.Cleanup(httpSrv.Close)
	t.Cleanup(wsSrv.Close)

	ws := transport.NewWebSocket("ws"+strings.TrimPrefix(wsSrv.URL, "http"),
		transport.WithReconnect(false), transport.WithTimeout(5*time.Second))
	mq, mqFrames := transport.NewFakeMQTT(authtest.Realm)
	t.Cleanup(func() { _ = ws.Close(); _ = mq.Close() })

	return map[string]frameTransport{
		"http": {
			tr: transport.NewHTTP(httpSrv.URL, transport.WithRetry(0, 0)), frames: httpDev.recorded,
			gets: func() []string { httpDev.mu.Lock(); defer httpDev.mu.Unlock(); return httpDev.gets },
		},
		"websocket": {tr: ws, frames: wsDev.recorded},
		"mqtt":      {tr: mq, frames: mqFrames},
	}
}

// Requests without credentials must reach the device with no "auth" key and
// no other key set to null: a device with authentication enabled answers
// "auth": null with error 400 "bad request".
func TestTransports_NoNullFields(t *testing.T) {
	calls := map[string]func(context.Context, transport.Transport) error{
		"rpc.NewClient": func(ctx context.Context, tr transport.Transport) error {
			_, err := rpc.NewClient(tr).Call(ctx, "Switch.GetStatus", map[string]any{"id": 0})
			return err
		},
		"rpc.Request with nil Auth": func(ctx context.Context, tr transport.Transport) error {
			_, err := tr.Call(ctx, &rpc.Request{ID: uint64(9), JSONRPC: rpc.JSONRPCVersion, Method: "Shelly.GetStatus"})
			return err
		},
		"rpc.Request without id": func(ctx context.Context, tr transport.Transport) error {
			_, err := tr.Call(ctx, &rpc.Request{JSONRPC: rpc.JSONRPCVersion, Method: "Shelly.GetStatus"})
			return err
		},
		"typed nil id, params and auth": func(ctx context.Context, tr transport.Transport) error {
			_, err := tr.Call(ctx, typedNilRequest{})
			return err
		},
		"SimpleRequest": func(ctx context.Context, tr transport.Transport) error {
			_, err := tr.Call(ctx, transport.NewSimpleRequest("/shelly"))
			return err
		},
		// Shelly documents no batch frame and the fake answers it with one
		// result object, which Execute cannot parse; any other error fails.
		"batch": func(ctx context.Context, tr transport.Transport) error {
			_, err := rpc.NewClient(tr).NewBatch().Add("Switch.GetStatus", map[string]any{"id": 0}).
				Add("Shelly.GetStatus", nil).Execute(ctx)
			if err != nil && !strings.Contains(err.Error(), "failed to parse batch response") {
				return err
			}
			return nil
		},
	}

	for trName, ft := range newFrameTransports(t) {
		for callName, call := range calls {
			t.Run(trName+"/"+callName, func(t *testing.T) {
				before := len(ft.frames())
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := call(ctx, ft.tr); err != nil {
					t.Errorf("Call() error = %v", err)
				}
				frames := ft.frames()[before:]
				if trName == "http" && callName == "SimpleRequest" {
					if len(frames) != 0 || len(ft.gets()) == 0 {
						t.Errorf("SimpleRequest over HTTP: frames %q, GETs %q, want one GET and no frame", frames, ft.gets())
					}
					return
				}
				if len(frames) != 1 {
					t.Fatalf("device got %d frames, want 1: %q", len(frames), frames)
				}
				checkNoNulls(t, frames[0])
			})
		}
	}
}

func checkNoNulls(t *testing.T, frame []byte) {
	t.Helper()
	var f map[string]json.RawMessage
	if err := json.Unmarshal(frame, &f); err != nil {
		t.Fatalf("frame %q: %v", frame, err)
	}
	if _, ok := f["auth"]; ok {
		t.Errorf(`frame %s has an "auth" key`, frame)
	}
	if _, ok := f["id"]; !ok {
		t.Errorf("frame %s has no id", frame)
	}
	for k, v := range f {
		if string(v) == "null" {
			t.Errorf("frame %s sends %q as null", frame, k)
		}
	}
}
