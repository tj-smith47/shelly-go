package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Common client errors.
var (
	// ErrRateLimited indicates the client has exceeded the rate limit.
	ErrRateLimited = errors.New("rate limited")

	// ErrUnauthorized indicates the request was not authorized.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrDeviceOffline indicates the target device is offline.
	ErrDeviceOffline = errors.New("device offline")

	// ErrDeviceNotFound indicates the device was not found.
	ErrDeviceNotFound = errors.New("device not found")

	// ErrServerError indicates a server error occurred.
	ErrServerError = errors.New("server error")
)

// Client is a client for the Shelly Cloud Control API.
type Client struct {
	httpClient  *http.Client
	rateLimiter *rateLimiter
	accessToken string
	baseURL     string
	clientID    string
	mu          sync.RWMutex
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithAccessToken sets the access token directly.
func WithAccessToken(token string) ClientOption {
	return func(c *Client) {
		c.accessToken = token
		// Try to extract user_api_url from the token
		if userAPIURL, err := ExtractUserAPIURL(token); err == nil {
			c.baseURL = normalizeBaseURL(userAPIURL)
		}
	}
}

// WithBaseURL sets the base URL for API calls.
// This overrides the URL extracted from the token.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = normalizeBaseURL(baseURL)
	}
}

// WithClientID sets the OAuth client ID.
func WithClientID(clientID string) ClientOption {
	return func(c *Client) {
		c.clientID = clientID
	}
}

// WithRateLimit sets the rate limit (requests per second).
// The default is 1 request per second.
func WithRateLimit(requestsPerSecond float64) ClientOption {
	return func(c *Client) {
		c.rateLimiter = newRateLimiter(requestsPerSecond)
	}
}

// NewClient creates a new Cloud API client.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		clientID:    ClientIDDIY,
		rateLimiter: newRateLimiter(1.0), // 1 request per second
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// NewClientWithCredentials creates a new Cloud API client and authenticates
// with the given email and password SHA1 hash.
func NewClientWithCredentials(ctx context.Context, email, passwordSHA1 string, opts ...ClientOption) (*Client, error) {
	c := NewClient(opts...)

	token, err := c.Login(ctx, email, passwordSHA1)
	if err != nil {
		return nil, err
	}

	c.SetToken(token)
	return c, nil
}

// Login authenticates with the Shelly Cloud API and returns a token.
func (c *Client) Login(ctx context.Context, email, passwordSHA1 string) (*Token, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.wait(ctx); err != nil {
		return nil, err
	}

	// Build request body
	reqBody := LoginRequest{
		Email:    email,
		Password: passwordSHA1,
		ClientID: c.clientID,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, OAuthTokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var loginResp LoginResponse
	if unmarshalErr := json.Unmarshal(respBody, &loginResp); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse response: %w", unmarshalErr)
	}

	// Check for errors
	if !loginResp.IsOK {
		if len(loginResp.Errors) > 0 {
			return nil, fmt.Errorf("%w: %s", ErrInvalidCredentials, strings.Join(loginResp.Errors, ", "))
		}
		return nil, ErrInvalidCredentials
	}

	if loginResp.Data == nil || loginResp.Data.Token == "" {
		return nil, errors.New("no token in response")
	}

	// Parse the token
	token, err := ParseToken(loginResp.Data.Token)
	if err != nil {
		return nil, err
	}

	// Override user_api_url if provided in response
	if loginResp.Data.UserAPIURL != "" {
		token.UserAPIURL = loginResp.Data.UserAPIURL
	}

	return token, nil
}

// SetToken sets the access token.
func (c *Client) SetToken(token *Token) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.accessToken = token.AccessToken
	if token.UserAPIURL != "" {
		c.baseURL = normalizeBaseURL(token.UserAPIURL)
	}
}

// GetToken returns the current access token.
func (c *Client) GetToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken
}

// GetBaseURL returns the base URL for API calls.
func (c *Client) GetBaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

// doRequest performs an HTTP request with authentication and rate limiting.
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body any) ([]byte, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.wait(ctx); err != nil {
		return nil, err
	}

	// Get token and base URL
	c.mu.RLock()
	token := c.accessToken
	baseURL := c.baseURL
	c.mu.RUnlock()

	if token == "" {
		return nil, ErrUnauthorized
	}

	if baseURL == "" {
		return nil, ErrNoUserAPIURL
	}

	// Build URL
	reqURL := baseURL + endpoint

	// Build request body
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	switch resp.StatusCode {
	case http.StatusOK:
		return respBody, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return nil, ErrServerError
	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

// doGet performs a GET request with query parameters.
func (c *Client) doGet(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	if len(params) > 0 {
		endpoint = endpoint + "?" + params.Encode()
	}
	return c.doRequest(ctx, http.MethodGet, endpoint, nil)
}

// doPost performs a POST request with a JSON body.
func (c *Client) doPost(ctx context.Context, endpoint string, body any) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, endpoint, body)
}

// normalizeBaseURL ensures the base URL has the correct format.
func normalizeBaseURL(baseURL string) string {
	// Remove trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Ensure https:// prefix
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	return baseURL
}

// rateLimiter implements a simple rate limiter.
type rateLimiter struct {
	lastCall time.Time
	interval time.Duration
	mu       sync.Mutex
}

// newRateLimiter creates a new rate limiter.
func newRateLimiter(requestsPerSecond float64) *rateLimiter {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 1.0
	}
	return &rateLimiter{
		interval: time.Duration(float64(time.Second) / requestsPerSecond),
	}
}

// wait waits until the next request can be made.
func (r *rateLimiter) wait(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	nextCall := r.lastCall.Add(r.interval)

	if now.Before(nextCall) {
		waitDuration := nextCall.Sub(now)
		select {
		case <-time.After(waitDuration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	r.lastCall = time.Now()
	return nil
}
