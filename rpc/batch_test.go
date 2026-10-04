package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/tj-smith47/shelly-go/transport"
)

func TestClient_NewBatch(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	batch := client.NewBatch()

	if batch == nil {
		t.Fatal("NewBatch() returned nil")
	}

	if batch.client != client {
		t.Error("batch client should reference the creating client")
	}

	if len(batch.requests) != 0 {
		t.Error("new batch should be empty")
	}
}

func TestBatch_Add(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)
	batch := client.NewBatch()

	result := batch.Add("Switch.GetStatus", map[string]any{"id": 0})

	// Should return the batch for chaining
	if result != batch {
		t.Error("Add() should return the batch for chaining")
	}

	if batch.Len() != 1 {
		t.Errorf("batch length = %v, want 1", batch.Len())
	}

	// Add more requests
	batch.Add("Switch.GetStatus", map[string]any{"id": 1})
	batch.Add("Light.GetStatus", map[string]any{"id": 0})

	if batch.Len() != 3 {
		t.Errorf("batch length = %v, want 3", batch.Len())
	}
}

func TestBatch_AddRequest(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)
	batch := client.NewBatch()

	req := NewBatchRequest("Switch.Set", map[string]any{"id": 0, "on": true})
	result := batch.AddRequest(req)

	if result != batch {
		t.Error("AddRequest() should return the batch for chaining")
	}

	if batch.Len() != 1 {
		t.Errorf("batch length = %v, want 1", batch.Len())
	}
}

func TestBatch_Len(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)
	batch := client.NewBatch()

	if batch.Len() != 0 {
		t.Errorf("empty batch length = %v, want 0", batch.Len())
	}

	batch.Add("Test", nil)
	if batch.Len() != 1 {
		t.Errorf("batch length = %v, want 1", batch.Len())
	}

	batch.Add("Test", nil)
	batch.Add("Test", nil)
	if batch.Len() != 3 {
		t.Errorf("batch length = %v, want 3", batch.Len())
	}
}

func TestBatch_Clear(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)
	batch := client.NewBatch()

	batch.Add("Test1", nil)
	batch.Add("Test2", nil)
	batch.Add("Test3", nil)

	if batch.Len() != 3 {
		t.Errorf("batch length = %v, want 3", batch.Len())
	}

	result := batch.Clear()

	if result != batch {
		t.Error("Clear() should return the batch for chaining")
	}

	if batch.Len() != 0 {
		t.Errorf("cleared batch length = %v, want 0", batch.Len())
	}
}

func TestBatchResult_IsError(t *testing.T) {
	tests := []struct {
		result BatchResult
		name   string
		want   bool
	}{
		{
			name: "success result",
			result: BatchResult{
				Result: json.RawMessage(`{"status":"ok"}`),
			},
			want: false,
		},
		{
			name: "error result",
			result: BatchResult{
				Err: errors.New("test error"),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsError(); got != tt.want {
				t.Errorf("IsError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBatchResult_Unmarshal(t *testing.T) {
	tests := []struct {
		result  BatchResult
		name    string
		wantErr bool
	}{
		{
			name: "valid result",
			result: BatchResult{
				Result: json.RawMessage(`{"status":"ok"}`),
			},
			wantErr: false,
		},
		{
			name: "error result",
			result: BatchResult{
				Err: errors.New("test error"),
			},
			wantErr: true,
		},
		{
			name: "empty result",
			result: BatchResult{
				Result: nil,
			},
			wantErr: false,
		},
		{
			name: "invalid JSON",
			result: BatchResult{
				Result: json.RawMessage(`{invalid}`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target map[string]any
			err := tt.result.Unmarshal(&target)

			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBatchResult_String(t *testing.T) {
	tests := []struct {
		name   string
		result BatchResult
		want   string
	}{
		{
			name: "success result",
			result: BatchResult{
				Request: BatchRequest{Method: "Test"},
				Result:  json.RawMessage(`{"status":"ok"}`),
			},
			want: `BatchResult{Method: Test, Result: {"status":"ok"}}`,
		},
		{
			name: "error result",
			result: BatchResult{
				Request: BatchRequest{Method: "Test"},
				Err:     errors.New("test error"),
			},
			want: "BatchResult{Method: Test, Error: test error}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBatchBuilder_Add(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	bb := client.Batch()
	result := bb.Add("Test", nil)

	if result != bb {
		t.Error("Add() should return the builder for chaining")
	}

	if bb.Len() != 1 {
		t.Errorf("batch length = %v, want 1", bb.Len())
	}
}

func TestBatchBuilder_AddRequest(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	bb := client.Batch()
	req := NewBatchRequest("Test", nil)
	result := bb.AddRequest(req)

	if result != bb {
		t.Error("AddRequest() should return the builder for chaining")
	}

	if bb.Len() != 1 {
		t.Errorf("batch length = %v, want 1", bb.Len())
	}
}

// frameTransport answers each call with its method as the result, fails the
// method "Fail.Me" with an RPC error, and runs onCall after recording a call.
type frameTransport struct {
	onCall  func(n int)
	methods []string
	ids     []any
}

func (f *frameTransport) Call(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
	f.methods = append(f.methods, req.GetMethod())
	f.ids = append(f.ids, req.GetID())
	if f.onCall != nil {
		f.onCall(len(f.methods))
	}
	reply := map[string]any{"id": req.GetID(), "result": req.GetMethod()}
	if req.GetMethod() == "Fail.Me" {
		reply = map[string]any{"id": req.GetID(), "error": map[string]any{"code": -103, "message": "invalid argument"}}
	}
	return json.Marshal(reply)
}

func (f *frameTransport) Close() error { return nil }

// Shelly devices have no batch frame, so each request is its own call with
// its own id, sent in order, and one failure does not stop the rest.
func TestBatch_Execute_SeparateCalls(t *testing.T) {
	ft := &frameTransport{}
	results, err := NewClient(ft).Batch().
		Add("Switch.GetStatus", map[string]any{"id": 0}).
		Add("Fail.Me", nil).
		Add("Shelly.GetDeviceInfo", nil).
		Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := []string{"Switch.GetStatus", "Fail.Me", "Shelly.GetDeviceInfo"}
	if fmt.Sprint(ft.methods) != fmt.Sprint(want) {
		t.Errorf("calls = %v, want %v in order", ft.methods, want)
	}
	seen := map[any]bool{}
	for _, id := range ft.ids {
		if id == nil || seen[id] {
			t.Errorf("ids %v are not distinct and set", ft.ids)
		}
		seen[id] = true
	}

	if len(results) != len(want) {
		t.Fatalf("got %d results, want %d", len(results), len(want))
	}
	for i, r := range results {
		if r.Request.Method != want[i] {
			t.Errorf("results[%d] is for %s, want %s", i, r.Request.Method, want[i])
		}
	}
	var rpcErr *ErrorObject
	if !errors.As(results[1].Err, &rpcErr) || rpcErr.Code != -103 {
		t.Errorf("results[1].Err = %v, want RPC error -103", results[1].Err)
	}
	var got string
	if err := results[2].Unmarshal(&got); err != nil || got != "Shelly.GetDeviceInfo" {
		t.Errorf("results[2] = %q, %v", got, err)
	}
	if results[0].IsError() || results[2].IsError() {
		t.Error("a request after or before the failed one reports an error")
	}
}

func TestBatch_Execute_Empty(t *testing.T) {
	ft := &frameTransport{}
	results, err := NewClient(ft).NewBatch().Execute(context.Background())
	if err != nil || len(results) != 0 || len(ft.methods) != 0 {
		t.Errorf("Execute() = %v, %v with %d calls, want no results and no calls", results, err, len(ft.methods))
	}
}

// A canceled context stops the batch; the requests not sent carry ctx.Err().
func TestBatch_Execute_Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ft := &frameTransport{onCall: func(n int) {
		if n == 2 {
			cancel()
		}
	}}
	results, err := NewClient(ft).NewBatch().
		Add("A.Get", nil).Add("B.Get", nil).Add("C.Get", nil).Add("D.Get", nil).
		Execute(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Execute() error = %v, want context.Canceled", err)
	}
	if len(ft.methods) != 2 {
		t.Errorf("sent %v, want only the first two", ft.methods)
	}
	if len(results) != 4 || results[0].Err != nil || results[1].Err != nil {
		t.Fatalf("results = %v, want 4 with the first two answered", results)
	}
	for _, r := range results[2:] {
		if !errors.Is(r.Err, context.Canceled) {
			t.Errorf("%s: Err = %v, want context.Canceled", r.Request.Method, r.Err)
		}
	}
}
