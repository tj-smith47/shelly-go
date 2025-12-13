package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tj-smith47/shelly-go/transport"
)

// ClientOption is a functional option for configuring the RPC client.
type ClientOption func(*clientOptions)

type clientOptions struct {
	username string
	password string
	timeout  time.Duration
}

func defaultClientOptions() *clientOptions {
	return &clientOptions{
		timeout: 10 * time.Second,
	}
}

// WithTimeout sets the request timeout for the HTTP client.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}

// WithBasicAuth sets basic authentication credentials.
func WithBasicAuth(username, password string) ClientOption {
	return func(o *clientOptions) {
		o.username = username
		o.password = password
	}
}

// NewHTTPClient creates a new RPC client with an HTTP transport.
//
// This is a convenience constructor that combines transport.NewHTTP and NewClient.
// For more advanced transport configuration, use transport.NewHTTP directly.
//
// Example:
//
//	client, err := rpc.NewHTTPClient("192.168.1.100",
//	    rpc.WithTimeout(30*time.Second),
//	    rpc.WithBasicAuth("admin", "password"))
func NewHTTPClient(addr string, opts ...ClientOption) (*Client, error) {
	options := defaultClientOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Build transport options
	var transportOpts []transport.Option
	transportOpts = append(transportOpts, transport.WithTimeout(options.timeout))
	if options.username != "" {
		transportOpts = append(transportOpts, transport.WithAuth(options.username, options.password))
	}

	httpTransport := transport.NewHTTP(addr, transportOpts...)
	return NewClient(httpTransport), nil
}

// Client is a JSON-RPC 2.0 client that wraps a transport.
//
// The client is safe for concurrent use by multiple goroutines.
type Client struct {
	transport transport.Transport
	builder   *RequestBuilder
	router    *NotificationRouter
	auth      *AuthData
}

// NewClient creates a new RPC client with the given transport.
func NewClient(t transport.Transport) *Client {
	c := &Client{
		transport: t,
		builder:   NewRequestBuilder(),
		router:    NewNotificationRouter(),
	}

	// If the transport supports notifications, register our handler
	if subscriber, ok := t.(transport.Subscriber); ok {
		//nolint:errcheck // Subscribe fails only if transport is closed, which is caller's issue
		subscriber.Subscribe(func(data json.RawMessage) {
			c.handleNotification([]byte(data))
		})
	}

	return c
}

// NewClientWithAuth creates a new RPC client with authentication.
//
// The auth data will be included in all RPC requests.
func NewClientWithAuth(t transport.Transport, auth *AuthData) *Client {
	c := NewClient(t)
	c.auth = auth
	return c
}

// Call executes a single RPC request and returns the result.
//
// The method parameter specifies the RPC method to call (e.g., "Switch.Set").
// The params parameter contains the method parameters, which will be
// JSON-encoded. It can be a map, struct, or nil.
//
// The returned json.RawMessage contains the raw JSON result, which can be
// unmarshaled into the appropriate type.
func (c *Client) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Build the request
	req, err := c.builder.Build(method, params)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Add authentication if configured
	if c.auth != nil {
		req.WithAuth(c.auth)
	}

	// Execute request via transport (pass the request struct, transport handles JSON encoding)
	responseData, err := c.transport.Call(ctx, method, req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Parse response
	resp, err := ParseResponse([]byte(responseData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for RPC error
	if resp.Error != nil {
		return nil, resp.Error
	}

	return resp.Result, nil
}

// CallResult executes an RPC request and unmarshals the result into the provided value.
//
// This is a convenience method that combines Call and json.Unmarshal.
//
// Example:
//
//	var status SwitchStatus
//	err := client.CallResult(ctx, "Switch.GetStatus", params, &status)
func (c *Client) CallResult(ctx context.Context, method string, params, result any) error {
	data, err := c.Call(ctx, method, params)
	if err != nil {
		return err
	}

	if result != nil && len(data) > 0 {
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return nil
}

// Notify sends a notification (request without response expectation).
//
// Notifications are one-way messages that do not expect a response from
// the server. They are useful for fire-and-forget operations.
func (c *Client) Notify(ctx context.Context, method string, params any) error {
	// Build notification (request without ID)
	req, err := c.builder.BuildNotification(method, params)
	if err != nil {
		return fmt.Errorf("failed to build notification: %w", err)
	}

	// Add authentication if configured
	if c.auth != nil {
		req.WithAuth(c.auth)
	}

	// Send notification via transport (ignore response)
	_, err = c.transport.Call(ctx, method, req)
	if err != nil {
		return fmt.Errorf("notification failed: %w", err)
	}

	return nil
}

// OnNotification registers a global notification handler.
//
// The handler will be called for all notifications received from the server.
// This requires a stateful transport like WebSocket or MQTT.
func (c *Client) OnNotification(handler NotificationHandler) {
	c.router.OnNotification(handler)
}

// OnNotificationMethod registers a method-specific notification handler.
//
// The handler will only be called for notifications with the given method.
// This requires a stateful transport like WebSocket or MQTT.
func (c *Client) OnNotificationMethod(method string, handler MethodNotificationHandler) {
	c.router.OnNotificationMethod(method, handler)
}

// RemoveNotificationHandlers removes all global notification handlers.
func (c *Client) RemoveNotificationHandlers() {
	c.router.RemoveNotificationHandlers()
}

// RemoveMethodHandlers removes all handlers for a specific method.
func (c *Client) RemoveMethodHandlers(method string) {
	c.router.RemoveMethodHandlers(method)
}

// RemoveAllHandlers removes all notification handlers (global and method-specific).
func (c *Client) RemoveAllHandlers() {
	c.router.RemoveAllHandlers()
}

// SetAuth sets the authentication data for all subsequent requests.
func (c *Client) SetAuth(auth *AuthData) {
	c.auth = auth
}

// ClearAuth clears the authentication data.
func (c *Client) ClearAuth() {
	c.auth = nil
}

// Close closes the underlying transport.
//
// For stateful transports like WebSocket or MQTT, this will close the
// connection. For stateless transports like HTTP, this may be a no-op.
func (c *Client) Close() error {
	return c.transport.Close()
}

// Transport returns the underlying transport.
func (c *Client) Transport() transport.Transport {
	return c.transport
}

// handleNotification is called by the transport when a notification is received.
func (c *Client) handleNotification(data []byte) {
	// Parse notification
	notification, err := ParseNotification(data)
	if err != nil {
		// Ignore malformed notifications
		return
	}

	// Route to handlers
	c.router.Route(notification)
}

// RequestBuilder returns the underlying request builder.
//
// This can be used to build custom requests or access request ID management.
func (c *Client) RequestBuilder() *RequestBuilder {
	return c.builder
}

// NotificationRouter returns the underlying notification router.
//
// This can be used for advanced notification handling scenarios.
func (c *Client) NotificationRouter() *NotificationRouter {
	return c.router
}
