package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/tj-smith47/shelly-go/internal/authtest"
	"github.com/tj-smith47/shelly-go/internal/digest"
	"github.com/tj-smith47/shelly-go/types"
)

// digestWSDevice is a fake Gen2+ device on a websocket. It answers a frame
// whose auth object authtest rejects with error 401 and a digest challenge,
// and sends one notification after the first authenticated frame of each
// connection, as a device only notifies a peer that authenticated.
type digestWSDevice struct {
	nonce       any
	challenge   string
	conns       []*websocket.Conn
	ncs         []string
	mu          sync.Mutex
	frames      int
	challenges  int
	nonceOnDial bool
	dials       int
}

func (d *digestWSDevice) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	d.mu.Lock()
	d.dials++
	if d.nonceOnDial {
		d.nonce = d.dials * 1000
	}
	d.conns = append(d.conns, conn)
	d.mu.Unlock()

	notified := false
	for {
		var frame struct {
			ID   int64           `json:"id"`
			Src  string          `json:"src"`
			Auth json.RawMessage `json:"auth"`
		}
		if conn.ReadJSON(&frame) != nil {
			return
		}
		reply, notify := d.answer(frame.ID, frame.Src, frame.Auth)
		if conn.WriteJSON(reply) != nil {
			return
		}
		if notify && !notified {
			notified = true
			if conn.WriteJSON(map[string]any{
				"src": authtest.Realm, "dst": frame.Src, "method": "NotifyStatus",
				"params": map[string]any{"switch:0": map[string]any{"output": true}},
			}) != nil {
				return
			}
		}
	}
}

func (d *digestWSDevice) answer(id int64, src string, auth json.RawMessage) (reply map[string]any, authenticated bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.frames++
	if len(auth) > 0 {
		if nc, err := authtest.FrameAuth(auth, authtest.Realm, d.nonce, digestTestPassword); err == nil {
			d.ncs = append(d.ncs, nc)
			return map[string]any{"id": id, "src": authtest.Realm, "dst": src, "result": map[string]any{}}, true
		}
	}
	d.challenges++
	message := d.challenge
	if message == "" {
		message = authtest.FrameChallenge(authtest.Realm, d.nonce)
	}
	return map[string]any{
		"id": id, "src": authtest.Realm, "dst": src,
		"error": map[string]any{"code": 401, "message": message},
	}, false
}

// drop closes every connection from the device side.
func (d *digestWSDevice) drop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.conns {
		_ = c.Close()
	}
	d.conns = nil
}

func (d *digestWSDevice) counts() (frames, challenges int, ncs []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.frames, d.challenges, append([]string(nil), d.ncs...)
}

func newDigestWS(t *testing.T, d *digestWSDevice, opts ...Option) *WebSocket {
	t.Helper()
	srv := httptest.NewServer(d)
	t.Cleanup(srv.Close)
	ws := NewWebSocket(svrWSURL(srv), append([]Option{WithReconnect(false), WithTimeout(5 * time.Second)}, opts...)...)
	t.Cleanup(func() { _ = ws.Close() })
	return ws
}

func callWS(t *testing.T, ws *WebSocket) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := ws.Call(ctx, newTestRPCRequest("Shelly.GetStatus", nil))
	return err
}

func TestWebSocket_DigestAuth(t *testing.T) {
	nonces := map[string]any{
		"string nonce (fw >= 2.0.0)": "AAAAAABnZWVrc2Zvcmdl",
		"numeric nonce (fw < 2.0.0)": 1625214011,
	}
	for name, nonce := range nonces {
		t.Run(name, func(t *testing.T) {
			d := &digestWSDevice{nonce: nonce}
			ws := newDigestWS(t, d, WithDigestAuth("admin", digestTestPassword))
			notified := make(chan json.RawMessage, 1)
			if err := ws.Subscribe(func(msg json.RawMessage) {
				select {
				case notified <- msg:
				default:
				}
			}); err != nil {
				t.Fatal(err)
			}

			for range 3 {
				if err := callWS(t, ws); err != nil {
					t.Fatalf("Call() error = %v", err)
				}
			}

			frames, challenges, ncs := d.counts()
			if frames != 4 || challenges != 1 {
				t.Errorf("frames/challenges = %d/%d, want 4/1", frames, challenges)
			}
			want := []string{"00000001", "00000002", "00000003"}
			for i := range want {
				if i >= len(ncs) || ncs[i] != want[i] {
					t.Errorf("ncs = %v, want %v", ncs, want)
					break
				}
			}
			select {
			case <-notified:
			case <-time.After(5 * time.Second):
				t.Error("no notification after authenticating")
			}
		})
	}
}

func TestWebSocket_DigestAuth_WrongPassword(t *testing.T) {
	d := &digestWSDevice{nonce: "n"}
	ws := newDigestWS(t, d, WithDigestAuth("admin", "wrong"))

	err := callWS(t, ws)
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) || rpcErr.Code != http.StatusUnauthorized || !errors.Is(err, types.ErrAuth) {
		t.Fatalf("Call() error = %v, want *RPCError 401 matching types.ErrAuth", err)
	}
	if frames, _, _ := d.counts(); frames != 2 {
		t.Errorf("frames = %d, want 2 (one retry)", frames)
	}
}

func TestWebSocket_NoCredentials_401(t *testing.T) {
	d := &digestWSDevice{nonce: "n"}
	ws := newDigestWS(t, d)

	err := callWS(t, ws)
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) || rpcErr.Code != http.StatusUnauthorized || !errors.Is(err, types.ErrAuth) {
		t.Fatalf("Call() error = %v, want *RPCError 401 matching types.ErrAuth", err)
	}
	if _, err := digest.ParseFrameChallenge(rpcErr.Message); err != nil {
		t.Errorf("RPCError.Message %q is not the challenge: %v", rpcErr.Message, err)
	}
	if frames, _, _ := d.counts(); frames != 1 {
		t.Errorf("frames = %d, want 1", frames)
	}
}

func TestWebSocket_DigestAuth_UnusableChallenge(t *testing.T) {
	d := &digestWSDevice{nonce: "n", challenge: `{"auth_type":"digest","nonce":"n","realm":"r","algorithm":"MD5"}`}
	ws := newDigestWS(t, d, WithDigestAuth("admin", digestTestPassword))

	err := callWS(t, ws)
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) || !errors.Is(err, types.ErrAuth) || !errors.Is(err, digest.ErrChallenge) {
		t.Fatalf("Call() error = %v, want a 401 *RPCError naming the unusable challenge", err)
	}
}

// TestWebSocket_DigestAuth_Reconnect checks the nonce survives a reconnect,
// and that a device issuing a new nonce per connection is challenged again.
func TestWebSocket_DigestAuth_Reconnect(t *testing.T) {
	t.Run("same nonce", func(t *testing.T) {
		d := &digestWSDevice{nonce: "AAAAAABnZWVrc2Zvcmdl"}
		ws := newDigestWS(t, d, WithDigestAuth("admin", digestTestPassword))
		if err := callWS(t, ws); err != nil {
			t.Fatal(err)
		}
		d.drop()
		waitDisconnected(t, ws)
		if err := callWS(t, ws); err != nil {
			t.Fatalf("Call() after reconnect error = %v", err)
		}
		if _, challenges, ncs := d.counts(); challenges != 1 || len(ncs) != 2 || ncs[1] != "00000002" {
			t.Errorf("challenges = %d, ncs = %v; want 1 and [00000001 00000002]", challenges, ncs)
		}
	})

	t.Run("new nonce per connection", func(t *testing.T) {
		d := &digestWSDevice{nonceOnDial: true}
		ws := newDigestWS(t, d, WithDigestAuth("", digestTestPassword))
		if err := callWS(t, ws); err != nil {
			t.Fatal(err)
		}
		d.drop()
		waitDisconnected(t, ws)
		if err := callWS(t, ws); err != nil {
			t.Fatalf("Call() after reconnect error = %v", err)
		}
		if _, challenges, ncs := d.counts(); challenges != 2 || len(ncs) != 2 || ncs[1] != "00000001" {
			t.Errorf("challenges = %d, ncs = %v; want 2 and [00000001 00000001]", challenges, ncs)
		}
	})
}

func waitDisconnected(t *testing.T, ws *WebSocket) {
	t.Helper()
	disconnected := make(chan struct{})
	var once sync.Once
	ws.OnStateChange(func(s ConnectionState) {
		if s == StateDisconnected {
			once.Do(func() { close(disconnected) })
		}
	})
	if ws.State() == StateDisconnected {
		return
	}
	select {
	case <-disconnected:
	case <-time.After(5 * time.Second):
		t.Fatal("transport never saw the connection drop")
	}
}

func TestWebSocket_CallerAuthSentAsIs(t *testing.T) {
	d := &digestWSDevice{nonce: "n"}
	ws := newDigestWS(t, d, WithDigestAuth("admin", digestTestPassword))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := ws.Call(ctx, newTestRPCRequestWithAuth("Shelly.GetStatus", nil, map[string]any{"username": "admin"}))
	if !errors.Is(err, types.ErrAuth) {
		t.Fatalf("Call() error = %v, want ErrAuth", err)
	}
	if frames, _, _ := d.counts(); frames != 1 {
		t.Errorf("frames = %d, want 1: a caller's own auth object is not answered for it", frames)
	}
}

func TestRPCError_MapsCode(t *testing.T) {
	err := error(&RPCError{Code: -105, Message: "Argument 'id', value 7 not found!"})
	if !errors.Is(err, types.ErrNotFound) || errors.Is(err, types.ErrAuth) {
		t.Errorf("errors.Is mapping wrong for %v", err)
	}
	if err.Error() != "rpc error -105: Argument 'id', value 7 not found!" {
		t.Errorf("Error() = %q", err.Error())
	}
}
