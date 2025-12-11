package transport

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // MD5 required by RFC 2617 HTTP Digest Authentication
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// HTTP is an HTTP/HTTPS transport for Shelly devices.
// Supports both Gen1 (REST) and Gen2+ (RPC over HTTP POST).
type HTTP struct {
	client  *http.Client
	opts    *options
	baseURL string
	mu      sync.RWMutex
}

// NewHTTP creates a new HTTP transport.
//
// The baseURL should be the device's base URL (e.g., "http://192.168.1.100").
// Options can be provided to configure timeouts, authentication, retries, etc.
//
// Example:
//
//	transport := NewHTTP("http://192.168.1.100",
//	    WithTimeout(30*time.Second),
//	    WithAuth("admin", "password"))
func NewHTTP(baseURL string, opts ...Option) *HTTP {
	options := defaultOptions()
	applyOptions(options, opts)

	client := options.client
	if client == nil {
		client = &http.Client{
			Timeout: options.timeout,
			Transport: &http.Transport{
				TLSClientConfig:     options.tlsConfig,
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}

	// Normalize baseURL - add http:// if no scheme provided
	normalizedURL := baseURL
	if !strings.HasPrefix(normalizedURL, "http://") && !strings.HasPrefix(normalizedURL, "https://") {
		normalizedURL = "http://" + normalizedURL
	}
	normalizedURL = strings.TrimSuffix(normalizedURL, "/")

	return &HTTP{
		baseURL: normalizedURL,
		client:  client,
		opts:    options,
	}
}

// Call executes a method call via HTTP.
//
// For Gen2+ RPC: method is the RPC method (e.g., "Switch.Set")
// and params is marshaled to JSON for the RPC request body.
//
// For Gen1 REST: method is the URL path (e.g., "/relay/0?turn=on")
// and params can be nil or additional query parameters.
func (h *HTTP) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	var lastErr error
	retries := h.opts.maxRetries
	delay := h.opts.retryDelay

	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				delay = time.Duration(float64(delay) * h.opts.retryBackoff)
			}
		}

		result, err := h.doCall(ctx, method, params)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry certain errors
		if !h.shouldRetry(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doCall performs a single HTTP call attempt.
func (h *HTTP) doCall(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Determine if this is an RPC call or REST call
	// RPC methods contain a dot (e.g., "Shelly.GetStatus")
	// REST paths start with "/" (e.g., "/relay/0")
	isRPC := strings.Contains(method, ".")

	var req *http.Request
	var err error

	if isRPC {
		req, err = h.buildRPCRequest(ctx, method, params)
	} else {
		req, err = h.buildRESTRequest(ctx, method, params)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Apply authentication
	if authErr := h.applyAuth(req); authErr != nil {
		return nil, fmt.Errorf("failed to apply auth: %w", authErr)
	}

	// Add custom headers
	h.mu.RLock()
	for k, v := range h.opts.headers {
		req.Header.Set(k, v)
	}
	h.mu.RUnlock()

	// Execute request
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode >= 400 {
		return nil, h.parseHTTPError(resp.StatusCode, body)
	}

	// Return raw body for both RPC and REST
	// The RPC client will handle parsing RPC responses
	return body, nil
}

// buildRPCRequest builds an RPC request (Gen2+).
func (h *HTTP) buildRPCRequest(ctx context.Context, method string, params any) (*http.Request, error) {
	rpcReq := map[string]any{
		"id":     1,
		"method": method,
	}

	if params != nil {
		rpcReq["params"] = params
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.baseURL+"/rpc", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// buildRESTRequest builds a REST request (Gen1).
func (h *HTTP) buildRESTRequest(ctx context.Context, path string, params any) (*http.Request, error) {
	url := h.baseURL + path

	// Add query parameters if provided
	// params can be a map[string]string or map[string]any
	// Convert to query string
	// For now, assume params is already encoded in the path
	// This can be enhanced later if needed
	_ = params

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// parseHTTPError converts an HTTP error response to a Go error.
func (h *HTTP) parseHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: HTTP %d", types.ErrAuth, statusCode)
	case http.StatusNotFound:
		return fmt.Errorf("%w: HTTP %d", types.ErrNotFound, statusCode)
	case http.StatusRequestTimeout:
		return fmt.Errorf("%w: HTTP %d", types.ErrTimeout, statusCode)
	default:
		return fmt.Errorf("HTTP error %d: %s", statusCode, string(body))
	}
}

// applyAuth applies authentication to the request.
func (h *HTTP) applyAuth(req *http.Request) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	switch h.opts.authType {
	case authTypeBasic:
		req.SetBasicAuth(h.opts.username, h.opts.password)
	case authTypeDigest:
		// Digest auth requires a challenge-response
		// For simplicity, we'll do a pre-emptive request to get the challenge
		// In production, this should cache the challenge
		return h.applyDigestAuth(req)
	}

	return nil
}

// applyDigestAuth applies digest authentication.
// This requires a two-step process:
// 1. Make initial request to get WWW-Authenticate challenge
// 2. Parse challenge and calculate response hash
// 3. Retry request with Authorization header
func (h *HTTP) applyDigestAuth(req *http.Request) error {
	// Make initial request to get the challenge
	challengeReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("create challenge request: %w", err)
	}

	resp, err := h.client.Do(challengeReq)
	if err != nil {
		return fmt.Errorf("challenge request: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		// No auth required or different error
		return nil
	}

	// Parse WWW-Authenticate header
	authHeader := resp.Header.Get("WWW-Authenticate")
	if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "digest ") {
		return fmt.Errorf("no digest challenge in response")
	}

	// Parse challenge parameters
	challenge := parseDigestChallenge(authHeader[7:]) // Skip "Digest "

	realm := challenge["realm"]
	nonce := challenge["nonce"]
	qop := challenge["qop"]

	if realm == "" || nonce == "" {
		return fmt.Errorf("invalid digest challenge: missing realm or nonce")
	}

	// Generate client nonce
	cnonce := generateCNonce()
	nc := "00000001" // nonce count

	// Calculate response
	uri := req.URL.RequestURI()
	response := calculateDigestResponse(
		h.opts.username, h.opts.password,
		realm, nonce, nc, cnonce, qop,
		req.Method, uri,
	)

	// Build Authorization header
	authValue := fmt.Sprintf(
		`Digest username=%q, realm=%q, nonce=%q, uri=%q, response=%q`,
		h.opts.username, realm, nonce, uri, response,
	)
	if qop != "" {
		authValue += fmt.Sprintf(`, qop=%s, nc=%s, cnonce=%q`, qop, nc, cnonce)
	}

	req.Header.Set("Authorization", authValue)
	return nil
}

// parseDigestChallenge parses a digest authentication challenge string.
func parseDigestChallenge(challenge string) map[string]string {
	result := make(map[string]string)

	// Split by comma, handling quoted values
	parts := strings.Split(challenge, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		idx := strings.Index(part, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+1:])

		// Remove quotes if present
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		result[key] = value
	}

	return result
}

// generateCNonce generates a client nonce for digest auth.
func generateCNonce() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based if crypto/rand fails
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// calculateDigestResponse calculates the digest authentication response hash.
//
//nolint:gosec // MD5 is required by HTTP Digest Auth spec (RFC 2617)
func calculateDigestResponse(
	username, password, realm, nonce, nc, cnonce, qop, method, uri string,
) string {
	// HA1 = MD5(username:realm:password)
	ha1Input := fmt.Sprintf("%s:%s:%s", username, realm, password)
	ha1 := md5Hash(ha1Input)

	// HA2 = MD5(method:uri)
	ha2Input := fmt.Sprintf("%s:%s", method, uri)
	ha2 := md5Hash(ha2Input)

	// Response calculation depends on qop
	var response string
	if qop == "auth" || qop == "auth-int" {
		// response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
		responseInput := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2)
		response = md5Hash(responseInput)
	} else {
		// response = MD5(HA1:nonce:HA2)
		responseInput := fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2)
		response = md5Hash(responseInput)
	}

	return response
}

// md5Hash returns the hex-encoded MD5 hash of the input string.
//
//nolint:gosec // MD5 is required by HTTP Digest Auth spec (RFC 2617)
func md5Hash(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// shouldRetry determines if an error should be retried.
func (h *HTTP) shouldRetry(err error) bool {
	// Don't retry auth errors or not found errors
	if err == types.ErrAuth || err == types.ErrNotFound {
		return false
	}

	// Don't retry context errors
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Retry timeout and network errors
	return true
}

// Close closes the HTTP transport.
// This closes idle connections in the HTTP client's connection pool.
func (h *HTTP) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if transport, ok := h.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	return nil
}

// SetTimeout updates the HTTP client timeout.
func (h *HTTP) SetTimeout(timeout time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.opts.timeout = timeout
	h.client.Timeout = timeout
}

// GetTimeout returns the current timeout.
func (h *HTTP) GetTimeout() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.opts.timeout
}

// calculateBackoffDelay calculates the delay for exponential backoff.
func calculateBackoffDelay(baseDelay time.Duration, attempt int, multiplier float64) time.Duration {
	delay := float64(baseDelay) * math.Pow(multiplier, float64(attempt))
	return time.Duration(delay)
}
