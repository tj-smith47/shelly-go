package transport

import (
	"context"
	"encoding/json"
	"sync"
)

// rawCaptureKey is the unexported context key under which a [RawCapture] sink
// is stored. A struct{} key avoids collisions with any other context value.
type rawCaptureKey struct{}

// RawCapture accumulates the verbatim response bodies a device returns during
// the calls made under a context. It mirrors the pattern of
// net/http/httptrace: a caller installs a sink with [WithRawCapture], runs one
// or more operations, then reads the collected responses with [RawCapture.Responses].
//
// The same sink may be shared across concurrent device calls (e.g. a batch
// applied to a group), so RawCapture is safe for concurrent use.
type RawCapture struct {
	responses []json.RawMessage
	mu        sync.Mutex
}

// WithRawCapture returns a context that records every successful device
// response body produced by transport calls into sink, in call order. For
// Gen2+ devices this is the verbatim JSON-RPC envelope; for Gen1 it is the
// response body. Passing a nil sink returns ctx unchanged, so callers can wire
// it unconditionally.
func WithRawCapture(ctx context.Context, sink *RawCapture) context.Context {
	if sink == nil {
		return ctx
	}
	return context.WithValue(ctx, rawCaptureKey{}, sink)
}

// Responses returns a copy of the captured response bodies in the order the
// calls completed. The returned slice is owned by the caller.
func (c *RawCapture) Responses() []json.RawMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]json.RawMessage, len(c.responses))
	copy(out, c.responses)
	return out
}

// Record appends a defensive copy of one response body to the sink, in call
// order. The transport choke point calls this for every device response; it is
// exported so other response producers (and tests) can populate a sink
// directly. The copy is required because the caller may reuse the underlying
// buffer after the call.
func (c *RawCapture) Record(body json.RawMessage) {
	cp := make(json.RawMessage, len(body))
	copy(cp, body)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.responses = append(c.responses, cp)
}

// captureRaw records body into the context's RawCapture sink, if one is
// present. It is a no-op when no sink was installed, keeping the hot path free
// for the common case where raw capture is not requested.
func captureRaw(ctx context.Context, body json.RawMessage) {
	if sink, ok := ctx.Value(rawCaptureKey{}).(*RawCapture); ok {
		sink.Record(body)
	}
}
