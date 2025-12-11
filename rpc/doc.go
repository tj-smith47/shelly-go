// Package rpc provides a JSON-RPC 2.0 framework for Shelly device communication.
//
// This package implements a complete RPC client with support for:
//   - Single and batch RPC requests
//   - Request/response correlation via ID management
//   - Notification handling for asynchronous events
//   - Multiple transport protocols (HTTP, WebSocket, MQTT, CoAP)
//   - Context-based cancellation and timeout
//   - Authentication (basic, digest, token)
//
// # Basic Usage
//
// Create an RPC client wrapping any transport:
//
//	httpTransport := transport.NewHTTP("http://192.168.1.100")
//	client := rpc.NewClient(httpTransport)
//
//	// Single RPC call
//	result, err := client.Call(ctx, "Switch.Set", map[string]any{
//		"id": 0,
//		"on": true,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Batch Requests
//
// Batch multiple RPC calls into a single round-trip:
//
//	batch := client.NewBatch()
//	batch.Add("Switch.GetStatus", map[string]any{"id": 0})
//	batch.Add("Switch.GetStatus", map[string]any{"id": 1})
//	batch.Add("Light.GetStatus", map[string]any{"id": 0})
//
//	results, err := batch.Execute(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Process results in order
//	for i, result := range results {
//		if result.Err != nil {
//			log.Printf("Request %d failed: %v", i, result.Err)
//			continue
//		}
//		log.Printf("Request %d result: %s", i, result.Result)
//	}
//
// # Notifications
//
// Subscribe to asynchronous notifications (requires stateful transport like WebSocket):
//
//	// Subscribe to all notifications
//	client.OnNotification(func(method string, params json.RawMessage) {
//		log.Printf("Notification: %s with params: %s", method, params)
//	})
//
//	// Subscribe to specific notification methods
//	client.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {
//		var status struct {
//			Component string `json:"component"`
//			State     bool   `json:"state"`
//		}
//		if err := json.Unmarshal(params, &status); err == nil {
//			log.Printf("Status update: %s = %v", status.Component, status.State)
//		}
//	})
//
// # Request ID Management
//
// The client automatically manages request IDs for correlating responses.
// IDs are generated sequentially starting from 1. For batch requests,
// each request in the batch receives its own unique ID.
//
// # Error Handling
//
// RPC errors are mapped to standard error types from the types package:
//
//	result, err := client.Call(ctx, "Switch.Set", params)
//	if err != nil {
//		switch {
//		case errors.Is(err, types.ErrNotFound):
//			// Component not found
//		case errors.Is(err, types.ErrAuth):
//			// Authentication required
//		case errors.Is(err, types.ErrTimeout):
//			// Request timed out
//		default:
//			// Other error
//		}
//	}
//
// # Authentication
//
// Authentication is handled by the underlying transport. Configure
// authentication options when creating the transport:
//
//	httpTransport := transport.NewHTTP("http://192.168.1.100",
//		transport.WithAuth("admin", "password"),
//		transport.WithDigestAuth("admin", "password"))
//
//	client := rpc.NewClient(httpTransport)
//
// # Context Support
//
// All operations accept context.Context for cancellation and timeout:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	result, err := client.Call(ctx, "Switch.Set", params)
//	if errors.Is(err, context.DeadlineExceeded) {
//		// Request timed out
//	}
//
// # Thread Safety
//
// The RPC client is safe for concurrent use by multiple goroutines.
// Request ID generation and notification handler registration are
// internally synchronized.
//
// # Transport Compatibility
//
// The RPC client works with any transport implementing the
// transport.Transport interface:
//
//   - HTTP: For Gen1 REST and Gen2+ RPC over HTTP
//   - WebSocket: For real-time bidirectional communication
//   - MQTT: For pub/sub messaging
//   - CoAP: For Gen1 CoIoT protocol
//
// For stateful transports (WebSocket, MQTT), the client can handle
// asynchronous notifications via the OnNotification handler.
package rpc
