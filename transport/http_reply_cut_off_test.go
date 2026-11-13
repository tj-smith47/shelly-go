package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// stallAfterHeaders answers with status and then never sends the promised
// body, as a Gen1 device does when a factory reset wipes it mid-reply.
func stallAfterHeaders(t *testing.T, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "64")
		w.WriteHeader(status)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHTTPReplyCutOffAfterSuccessStatus(t *testing.T) {
	srv := stallAfterHeaders(t, http.StatusOK)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := NewHTTP(srv.URL).Call(ctx, NewSimpleRequest("/settings?reset=true"))
	if !errors.Is(err, types.ErrReplyCutOff) {
		t.Errorf("err = %v, want ErrReplyCutOff", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want the read's own cause kept", err)
	}
}

// A refusal whose body never arrives is not an accepted request.
func TestHTTPStalledErrorStatusIsNotReplyCutOff(t *testing.T) {
	srv := stallAfterHeaders(t, http.StatusInternalServerError)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := NewHTTP(srv.URL).Call(ctx, NewSimpleRequest("/settings?reset=true"))
	if err == nil || errors.Is(err, types.ErrReplyCutOff) {
		t.Errorf("err = %v, want an error that is not ErrReplyCutOff", err)
	}
}
