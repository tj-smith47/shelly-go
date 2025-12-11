package integrator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// API endpoints
const (
	DefaultAPIURL = "https://api.shelly.cloud"
	AuthEndpoint  = "/integrator/get_access_token"
)

// Common errors.
var (
	// ErrNotAuthenticated indicates the client has not authenticated.
	ErrNotAuthenticated = errors.New("not authenticated")

	// ErrTokenExpired indicates the JWT token has expired.
	ErrTokenExpired = errors.New("token expired")

	// ErrAuthFailed indicates authentication failed.
	ErrAuthFailed = errors.New("authentication failed")

	// ErrConnectionClosed indicates the WebSocket connection is closed.
	ErrConnectionClosed = errors.New("connection closed")

	// ErrNoAccess indicates no access to the device.
	ErrNoAccess = errors.New("no access to device")
)

// Client is the Shelly Integrator API client.
type Client struct {
	authData      *AuthData
	httpClient    *http.Client
	connections   map[string]*Connection
	integratorTag string
	token         string
	apiURL        string
	mu            sync.RWMutex
}

// New creates a new Integrator API client.
func New(integratorTag, token string) *Client {
	return &Client{
		integratorTag: integratorTag,
		token:         token,
		apiURL:        DefaultAPIURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		connections: make(map[string]*Connection),
	}
}

// NewWithOptions creates a new client with custom options.
func NewWithOptions(integratorTag, token, apiURL string, httpClient *http.Client) *Client {
	c := New(integratorTag, token)
	if apiURL != "" {
		c.apiURL = apiURL
	}
	if httpClient != nil {
		c.httpClient = httpClient
	}
	return c
}

// Authenticate obtains a JWT token from the Shelly cloud.
func (c *Client) Authenticate(ctx context.Context) error {
	authReq := AuthRequest{
		IntegratorTag: c.integratorTag,
		Token:         c.token,
	}

	body, err := json.Marshal(authReq)
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+AuthEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var authResp AuthResponse
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !authResp.IsOK || authResp.Data == nil {
		return fmt.Errorf("%w: %s", ErrAuthFailed, string(authResp.Errors))
	}

	c.mu.Lock()
	c.authData = authResp.Data
	c.mu.Unlock()

	return nil
}

// IsAuthenticated returns true if the client has a valid, non-expired token.
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.authData == nil {
		return false
	}
	return !c.authData.IsExpired()
}

// GetToken returns the current JWT token, or an error if not authenticated.
func (c *Client) GetToken() (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.authData == nil {
		return "", ErrNotAuthenticated
	}
	if c.authData.IsExpired() {
		return "", ErrTokenExpired
	}
	return c.authData.Token, nil
}

// TokenExpiresAt returns the token expiration time.
func (c *Client) TokenExpiresAt() (time.Time, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.authData == nil {
		return time.Time{}, ErrNotAuthenticated
	}
	return c.authData.ExpiresTime(), nil
}

// RefreshToken refreshes the JWT token if it's expired or about to expire.
func (c *Client) RefreshToken(ctx context.Context) error {
	c.mu.RLock()
	needsRefresh := c.authData == nil || c.authData.IsExpired()
	c.mu.RUnlock()

	if needsRefresh {
		return c.Authenticate(ctx)
	}
	return nil
}

// Connect establishes a WebSocket connection to a Shelly cloud server.
func (c *Client) Connect(ctx context.Context, host string, opts *ConnectOptions) (*Connection, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = DefaultConnectOptions()
	}

	conn, err := newConnection(ctx, host, token, opts)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.connections[host] = conn
	c.mu.Unlock()

	return conn, nil
}

// GetConnection returns an existing connection to a host, or nil if not connected.
func (c *Client) GetConnection(host string) *Connection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connections[host]
}

// Disconnect closes the connection to a specific host.
func (c *Client) Disconnect(host string) error {
	c.mu.Lock()
	conn, ok := c.connections[host]
	if ok {
		delete(c.connections, host)
	}
	c.mu.Unlock()

	if !ok {
		return nil
	}
	return conn.Close()
}

// DisconnectAll closes all active connections.
func (c *Client) DisconnectAll() error {
	c.mu.Lock()
	conns := make([]*Connection, 0, len(c.connections))
	for _, conn := range c.connections {
		conns = append(conns, conn)
	}
	c.connections = make(map[string]*Connection)
	c.mu.Unlock()

	var lastErr error
	for _, conn := range conns {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ActiveConnections returns the list of hosts with active connections.
func (c *Client) ActiveConnections() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hosts := make([]string, 0, len(c.connections))
	for host := range c.connections {
		hosts = append(hosts, host)
	}
	return hosts
}

// Close closes all connections and cleans up resources.
func (c *Client) Close() error {
	return c.DisconnectAll()
}
