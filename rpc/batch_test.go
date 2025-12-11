package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
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

func TestBatch_Execute_Empty(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)
	batch := client.NewBatch()

	results, err := batch.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("empty batch results length = %v, want 0", len(results))
	}
}

func TestBatch_Execute_Success(t *testing.T) {
	// Mock transport that returns successful batch response
	mt := &mockTransport{
		response: []byte(`[
			{"jsonrpc":"2.0","id":1,"result":{"status":"on"}},
			{"jsonrpc":"2.0","id":2,"result":{"status":"off"}},
			{"jsonrpc":"2.0","id":3,"result":{"brightness":50}}
		]`),
	}

	client := NewClient(mt)
	batch := client.NewBatch()

	batch.Add("Switch.GetStatus", map[string]any{"id": 0})
	batch.Add("Switch.GetStatus", map[string]any{"id": 1})
	batch.Add("Light.GetStatus", map[string]any{"id": 0})

	results, err := batch.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("results length = %v, want 3", len(results))
	}

	// Check first result
	if results[0].IsError() {
		t.Error("first result should not be an error")
	}

	var status map[string]any
	if err := results[0].Unmarshal(&status); err != nil {
		t.Errorf("failed to unmarshal first result: %v", err)
	}

	if status["status"] != "on" {
		t.Errorf("first result status = %v, want on", status["status"])
	}
}

func TestBatch_Execute_WithErrors(t *testing.T) {
	// Mock transport that returns some errors
	mt := &mockTransport{
		response: []byte(`[
			{"jsonrpc":"2.0","id":1,"result":{"status":"on"}},
			{"jsonrpc":"2.0","id":2,"error":{"code":404,"message":"Not found"}},
			{"jsonrpc":"2.0","id":3,"result":{"brightness":50}}
		]`),
	}

	client := NewClient(mt)
	batch := client.NewBatch()

	batch.Add("Switch.GetStatus", map[string]any{"id": 0})
	batch.Add("Switch.GetStatus", map[string]any{"id": 999}) // Non-existent
	batch.Add("Light.GetStatus", map[string]any{"id": 0})

	results, err := batch.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("results length = %v, want 3", len(results))
	}

	// First result should succeed
	if results[0].IsError() {
		t.Error("first result should not be an error")
	}

	// Second result should fail
	if !results[1].IsError() {
		t.Error("second result should be an error")
	}

	if results[1].Err == nil {
		t.Error("second result Err should not be nil")
	}

	// Third result should succeed
	if results[2].IsError() {
		t.Error("third result should not be an error")
	}
}

func TestBatch_Execute_TransportError(t *testing.T) {
	// Mock transport that returns an error
	mt := &mockTransport{
		err: errors.New("transport error"),
	}

	client := NewClient(mt)
	batch := client.NewBatch()

	batch.Add("Test", nil)

	_, err := batch.Execute(context.Background())
	if err == nil {
		t.Error("Execute() should return error when transport fails")
	}
}

func TestBatch_Execute_InvalidResponse(t *testing.T) {
	// Mock transport that returns invalid JSON
	mt := &mockTransport{
		response: []byte(`{invalid}`),
	}

	client := NewClient(mt)
	batch := client.NewBatch()

	batch.Add("Test", nil)

	_, err := batch.Execute(context.Background())
	if err == nil {
		t.Error("Execute() should return error for invalid response")
	}
}

func TestBatch_Execute_MissingResponse(t *testing.T) {
	// Mock transport that returns fewer responses than requests
	mt := &mockTransport{
		response: []byte(`[
			{"jsonrpc":"2.0","id":1,"result":{}}
		]`),
	}

	client := NewClient(mt)
	batch := client.NewBatch()

	batch.Add("Test1", nil)
	batch.Add("Test2", nil)

	results, err := batch.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("results length = %v, want 2", len(results))
	}

	// Second result should have an error for missing response
	if !results[1].IsError() {
		t.Error("result with missing response should be an error")
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

func TestClient_Batch(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`[
			{"jsonrpc":"2.0","id":1,"result":{"status":"on"}},
			{"jsonrpc":"2.0","id":2,"result":{"status":"off"}}
		]`),
	}

	client := NewClient(mt)

	// Test fluent interface
	results, err := client.Batch().
		Add("Switch.GetStatus", map[string]any{"id": 0}).
		Add("Switch.GetStatus", map[string]any{"id": 1}).
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Batch().Execute() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("results length = %v, want 2", len(results))
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

func TestBatch_Execute_IDCorrelation(t *testing.T) {
	// Mock transport that returns responses in different order
	mt := &mockTransport{
		response: []byte(`[
			{"jsonrpc":"2.0","id":2,"result":{"second":true}},
			{"jsonrpc":"2.0","id":3,"result":{"third":true}},
			{"jsonrpc":"2.0","id":1,"result":{"first":true}}
		]`),
	}

	client := NewClient(mt)
	batch := client.NewBatch()

	batch.Add("Test1", nil)
	batch.Add("Test2", nil)
	batch.Add("Test3", nil)

	results, err := batch.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Results should be in the original request order, not response order
	var first, second, third map[string]any

	if err := results[0].Unmarshal(&first); err != nil {
		t.Fatalf("failed to unmarshal first result: %v", err)
	}

	if err := results[1].Unmarshal(&second); err != nil {
		t.Fatalf("failed to unmarshal second result: %v", err)
	}

	if err := results[2].Unmarshal(&third); err != nil {
		t.Fatalf("failed to unmarshal third result: %v", err)
	}

	if first["first"] != true {
		t.Error("first result should contain 'first: true'")
	}

	if second["second"] != true {
		t.Error("second result should contain 'second: true'")
	}

	if third["third"] != true {
		t.Error("third result should contain 'third: true'")
	}
}

func TestBatch_ContextCancellation(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`[{"jsonrpc":"2.0","id":1,"result":{}}]`),
	}

	client := NewClient(mt)
	batch := client.NewBatch()
	batch.Add("Test", nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The behavior depends on when the transport checks the context
	// We just verify it doesn't panic
	_, _ = batch.Execute(ctx)
}
