package rpc

import (
	"context"
	"encoding/json"
	"fmt"
)

// Batch collects RPC requests and sends them one after another with Execute,
// each as its own call.
type Batch struct {
	client   *Client
	requests []BatchRequest
}

// NewBatch creates a new empty batch.
func (c *Client) NewBatch() *Batch {
	return &Batch{
		client:   c,
		requests: make([]BatchRequest, 0),
	}
}

// Add adds a new request to the batch.
//
// Each request gets its own ID when it is sent. Requests are sent in the
// order they are added.
func (b *Batch) Add(method string, params any) *Batch {
	b.requests = append(b.requests, NewBatchRequest(method, params))
	return b
}

// AddRequest adds a BatchRequest to the batch.
func (b *Batch) AddRequest(req BatchRequest) *Batch {
	b.requests = append(b.requests, req)
	return b
}

// Len returns the number of requests in the batch.
func (b *Batch) Len() int {
	return len(b.requests)
}

// Clear removes all requests from the batch.
func (b *Batch) Clear() *Batch {
	b.requests = b.requests[:0]
	return b
}

// Execute sends each request as its own call, in the order they were added,
// and returns one result per request in that order.
//
// Shelly devices have no batch frame, so a batch is a convenience for making
// several calls, not one round trip. A request that fails sets its result's
// Err and the rest are still sent. If ctx is canceled, no further request is
// sent, each remaining result's Err is set to ctx.Err(), and Execute returns
// the results with ctx.Err().
func (b *Batch) Execute(ctx context.Context) ([]BatchResult, error) {
	results := make([]BatchResult, len(b.requests))
	for i, req := range b.requests {
		results[i].Request = req
		if err := ctx.Err(); err != nil {
			results[i].Err = err
			continue
		}
		results[i].Result, results[i].Err = b.client.Call(ctx, req.Method, req.Params)
	}
	return results, ctx.Err()
}

// BatchResult represents the result of a single request in a batch.
type BatchResult struct {
	Request BatchRequest
	Err     error
	Result  json.RawMessage
}

// IsError returns true if this result contains an error.
func (br *BatchResult) IsError() bool {
	return br.Err != nil
}

// Unmarshal unmarshals the result into the provided value.
// Returns an error if the result contains an error or if unmarshaling fails.
func (br *BatchResult) Unmarshal(v any) error {
	if br.Err != nil {
		return br.Err
	}

	if len(br.Result) == 0 {
		return nil
	}

	if err := json.Unmarshal(br.Result, v); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return nil
}

// String returns a string representation of the batch result for debugging.
func (br *BatchResult) String() string {
	if br.Err != nil {
		return fmt.Sprintf("BatchResult{Method: %s, Error: %v}",
			br.Request.Method, br.Err)
	}
	return fmt.Sprintf("BatchResult{Method: %s, Result: %s}",
		br.Request.Method, string(br.Result))
}

// BatchBuilder provides a fluent interface for building and executing batches.
//
// Example:
//
//	results, err := client.Batch().
//		Add("Switch.GetStatus", map[string]any{"id": 0}).
//		Add("Switch.GetStatus", map[string]any{"id": 1}).
//		Execute(ctx)
type BatchBuilder struct {
	*Batch
}

// Batch creates a new BatchBuilder for fluent batch construction.
func (c *Client) Batch() *BatchBuilder {
	return &BatchBuilder{
		Batch: c.NewBatch(),
	}
}

// Add adds a request to the batch and returns the builder for chaining.
func (bb *BatchBuilder) Add(method string, params any) *BatchBuilder {
	bb.Batch.Add(method, params)
	return bb
}

// AddRequest adds a BatchRequest to the batch and returns the builder for chaining.
func (bb *BatchBuilder) AddRequest(req BatchRequest) *BatchBuilder {
	bb.Batch.AddRequest(req)
	return bb
}

// Execute executes the batch and returns the results.
func (bb *BatchBuilder) Execute(ctx context.Context) ([]BatchResult, error) {
	return bb.Batch.Execute(ctx)
}
