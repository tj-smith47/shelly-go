package transport

import (
	"bytes"
	"context"
	"crypto/md5"  //nolint:gosec // MD5 required by RFC 2617 HTTP Digest Authentication
	"crypto/rand"
	"crypto/sha256"
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

// Call executes an RPC or REST request via HTTP.
//
// For Gen2+ RPC: req contains the RPC method, params, and optional auth.
// For Gen1 REST: req is a SimpleRequest with the URL path.
func (h *HTTP) Call(ctx context.Context, req RPCRequest) (json.RawMessage, error) {
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

		result, err := h.doCall(ctx, req)
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
func (h *HTTP) doCall(ctx context.Context, rpcReq RPCRequest) (json.RawMessage, error) {
	var req *http.Request
	var err error

	// Determine if this is a REST (Gen1) or RPC (Gen2+) call
	if rpcReq.IsREST() {
		req, err = h.buildRESTRequest(ctx, rpcReq.GetMethod())
	} else {
		req, err = h.buildRPCRequest(ctx, rpcReq)
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
func (h *HTTP) buildRPCRequest(ctx context.Context, rpcReq RPCRequest) (*http.Request, error) {
	// Build the JSON-RPC 2.0 request body
	reqBody := map[string]any{
		"id":      rpcReq.GetID(),
		"jsonrpc": rpcReq.GetJSONRPC(),
		"method":  rpcReq.GetMethod(),
	}

	// Unmarshal params from json.RawMessage and add to request
	if params := rpcReq.GetParams(); len(params) > 0 {
		var p any
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
		reqBody["params"] = p
	}

	// Add RPC-level authentication if provided
	if auth := rpcReq.GetAuth(); auth != nil {
		reqBody["auth"] = auth
	}

	body, err := json.Marshal(reqBody)
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
func (h *HTTP) buildRESTRequest(ctx context.Context, path string) (*http.Request, error) {
	url := h.baseURL + path

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
	algorithm := challenge["algorithm"]

	// Normalize algorithm - default to MD5 if not specified
	if algorithm == "" {
		algorithm = "MD5"
	}

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
		req.Method, uri, algorithm,
	)

	// Build Authorization header
	authValue := fmt.Sprintf(
		`Digest username=%q, realm=%q, nonce=%q, uri=%q, response=%q`,
		h.opts.username, realm, nonce, uri, response,
	)
	if qop != "" {
		authValue += fmt.Sprintf(`, qop=%s, nc=%s, cnonce=%q`, qop, nc, cnonce)
	}
	// Include algorithm in response if not MD5 (some servers require this)
	if algorithm != "" && algorithm != "MD5" {
		authValue += fmt.Sprintf(`, algorithm=%s`, algorithm)
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
// Supports MD5 (RFC 2617) and SHA-256 (RFC 7616) algorithms.
//
//nolint:gosec // MD5 is required by HTTP Digest Auth spec (RFC 2617)
func calculateDigestResponse(
	username, password, realm, nonce, nc, cnonce, qop, method, uri, algorithm string,
) string {
	// Select hash function based on algorithm
	hashFunc := md5Hash
	if algorithm == "SHA-256" || algorithm == "SHA-256-sess" {
		hashFunc = sha256Hash
	}

	// HA1 = HASH(username:realm:password)
	ha1Input := fmt.Sprintf("%s:%s:%s", username, realm, password)
	ha1 := hashFunc(ha1Input)

	// HA2 = HASH(method:uri)
	ha2Input := fmt.Sprintf("%s:%s", method, uri)
	ha2 := hashFunc(ha2Input)

	// Response calculation depends on qop
	var response string
	if qop == "auth" || qop == "auth-int" {
		// response = HASH(HA1:nonce:nc:cnonce:qop:HA2)
		responseInput := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2)
		response = hashFunc(responseInput)
	} else {
		// response = HASH(HA1:nonce:HA2)
		responseInput := fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2)
		response = hashFunc(responseInput)
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

// sha256Hash returns the hex-encoded SHA-256 hash of the input string.
func sha256Hash(input string) string {
	hash := sha256.Sum256([]byte(input))
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
