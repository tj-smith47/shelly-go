package rpc

import (
	"context"
	"encoding/json"
	"fmt"
)

// batchRPCRequest wraps multiple RPC requests for batch execution.
// It implements transport.RPCRequest interface.
type batchRPCRequest struct {
	requests []*Request
}

func (b *batchRPCRequest) GetID() any        { return nil }
func (b *batchRPCRequest) GetMethod() string { return "" }
func (b *batchRPCRequest) GetParams() json.RawMessage {
	// Marshal the batch of requests as params
	// This should never fail as we're marshaling known valid Request structs
	data, err := json.Marshal(b.requests)
	if err != nil {
		return nil
	}
	return data
}
func (b *batchRPCRequest) GetAuth() any            { return nil }
func (b *batchRPCRequest) GetJSONRPC() string      { return "" }
func (b *batchRPCRequest) IsREST() bool            { return false }
func (b *batchRPCRequest) IsBatch() bool           { return true }
func (b *batchRPCRequest) GetRequests() []*Request { return b.requests }

// Batch represents a collection of RPC requests to be executed together.
//
// Batch requests allow multiple RPC calls to be sent in a single network
// round-trip, improving performance when multiple operations are needed.
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
// The request will be assigned a unique ID when the batch is executed.
// Requests are executed in the order they are added.
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

// Execute executes the batch and returns the results.
//
// All requests in the batch are sent to the server in a single RPC call.
// The results are returned in the same order as the requests were added.
//
// If the transport fails, an error is returned. Individual request errors
// are available in the returned BatchResult.
func (b *Batch) Execute(ctx context.Context) ([]BatchResult, error) {
	if len(b.requests) == 0 {
		return []BatchResult{}, nil
	}

	// Build RPC requests with sequential IDs
	rpcRequests, err := b.client.builder.BuildBatch(b.requests)
	if err != nil {
		return nil, fmt.Errorf("failed to build batch requests: %w", err)
	}

	// Wrap requests in a batch RPC request
	batchReq := &batchRPCRequest{requests: rpcRequests}

	// Execute batch via transport
	responseData, err := b.client.transport.Call(ctx, batchReq)
	if err != nil {
		return nil, fmt.Errorf("batch request failed: %w", err)
	}

	// Parse batch response
	batchResp, err := ParseBatchResponse([]byte(responseData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse batch response: %w", err)
	}

	// Convert to BatchResults
	results := make([]BatchResult, len(rpcRequests))
	for i, rpcReq := range rpcRequests {
		// Find corresponding response by ID
		var resp *Response
		for _, r := range batchResp.Responses {
			if idsEqual(r.ID, rpcReq.ID) {
				resp = r
				break
			}
		}

		if resp == nil {
			results[i] = BatchResult{
				Request: b.requests[i],
				Err:     fmt.Errorf("no response for request ID %v", rpcReq.ID),
			}
			continue
		}

		if resp.Error != nil {
			results[i] = BatchResult{
				Request: b.requests[i],
				Err:     resp.Error,
			}
		} else {
			results[i] = BatchResult{
				Request: b.requests[i],
				Result:  resp.Result,
			}
		}
	}

	return results, nil
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
