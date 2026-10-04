package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tj-smith47/shelly-go/internal/digest"
	"github.com/tj-smith47/shelly-go/types"
)

// HTTP is an HTTP/HTTPS transport for Shelly devices.
// Supports both Gen1 (REST) and Gen2+ (RPC over HTTP POST).
type HTTP struct {
	client    *http.Client
	opts      *options
	digest    *digest.Session
	baseURL   string
	mu        sync.RWMutex
	requestID atomic.Int64
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
		tr := &http.Transport{
			TLSClientConfig:     options.tlsConfig,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		}
		if options.bindIface != "" {
			tr.DialContext = (&net.Dialer{Control: bindControl(options.bindIface)}).DialContext
		}
		client = &http.Client{Timeout: options.timeout, Transport: tr}
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
		digest:  options.digestSession(),
	}
}

// Call executes an RPC or REST request via HTTP.
//
// For Gen2+ RPC: req contains the RPC method, params, and optional auth.
// For Gen1 REST: req is a SimpleRequest with the URL path.
func (h *HTTP) Call(ctx context.Context, req RPCRequest) (json.RawMessage, error) {
	// A caller-supplied client owns its own dialer, so the bind cannot be applied;
	// sending over the default route would reach the wrong device.
	if h.opts.client != nil && h.opts.bindIface != "" {
		return nil, fmt.Errorf("%w: WithBindInterface cannot be combined with WithClient", types.ErrInvalidParam)
	}

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

// doCall performs a single HTTP call attempt. With digest credentials, a 401
// carrying a challenge (the first request, or a nonce the device no longer
// accepts) is answered and the request sent once more; later requests reuse
// the nonce with a rising nonce count.
func (h *HTTP) doCall(ctx context.Context, rpcReq RPCRequest) (json.RawMessage, error) {
	body, status, challenge, err := h.send(ctx, rpcReq)
	if err != nil {
		return nil, err
	}

	if status == http.StatusUnauthorized && h.digest != nil && challenge != "" {
		ch, chErr := digest.ParseHeader(challenge)
		if chErr != nil {
			return nil, fmt.Errorf("%w: %w", types.ErrAuth, chErr)
		}
		h.digest.Accept(ch)
		body, status, _, err = h.send(ctx, rpcReq)
		if err != nil {
			return nil, err
		}
	}

	if status >= 400 {
		return nil, h.parseHTTPError(status, body)
	}

	// Record the verbatim device response for any caller that installed a
	// RawCapture sink on the context (see WithRawCapture); a no-op otherwise.
	// This is the single choke point for both Gen1 (REST) and Gen2+ (RPC), so
	// one hook here surfaces the exact device response for every call.
	captureRaw(ctx, body)

	// Return raw body for both RPC and REST
	// The RPC client will handle parsing RPC responses
	return body, nil
}

// send makes one request and returns the response body, status and
// WWW-Authenticate header.
func (h *HTTP) send(ctx context.Context, rpcReq RPCRequest) (body []byte, status int, challenge string, err error) {
	var req *http.Request

	// Determine if this is a REST (Gen1) or RPC (Gen2+) call
	if rpcReq.IsREST() {
		req, err = h.buildRESTRequest(ctx, rpcReq.GetMethod())
	} else {
		req, err = h.buildRPCRequest(ctx, rpcReq)
	}
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to build request: %w", err)
	}

	if authErr := h.applyAuth(req); authErr != nil {
		return nil, 0, "", fmt.Errorf("failed to apply auth: %w", authErr)
	}

	// Add custom headers
	h.mu.RLock()
	for k, v := range h.opts.headers {
		req.Header.Set(k, v)
	}
	h.mu.RUnlock()

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, 0, "", fmt.Errorf("HTTP request failed: %w", RedactURLError(err))
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to read response: %w", err)
	}
	return body, resp.StatusCode, resp.Header.Get("WWW-Authenticate"), nil
}

// RedactURLError hides the query values of the URL quoted in a failed request's
// error and returns the same error. Gen1 devices take settings as query
// parameters and the cloud API takes its auth key as one, so the URL of such a
// request carries a secret, and the error text reaches logs and callers'
// messages. Parameter names are kept: they say which request failed. Errors
// that are not a *url.Error, or whose URL has no query, are returned unchanged.
func RedactURLError(err error) error {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return err
	}
	// Split by hand: the URL of a parse failure cannot be parsed again.
	base, query, found := strings.Cut(urlErr.URL, "?")
	if !found || query == "" {
		return err
	}
	query, fragment, hasFragment := strings.Cut(query, "#")
	names := make([]string, 0, strings.Count(query, "&")+1)
	for pair := range strings.SplitSeq(query, "&") {
		name, _, _ := strings.Cut(pair, "=")
		names = append(names, name+"=REDACTED")
	}
	urlErr.URL = base + "?" + strings.Join(names, "&")
	if hasFragment {
		urlErr.URL += "#" + fragment
	}
	return err
}

// buildRPCRequest builds an RPC request (Gen2+).
func (h *HTTP) buildRPCRequest(ctx context.Context, rpcReq RPCRequest) (*http.Request, error) {
	reqBody, err := newFrame(rpcReq)
	if err != nil {
		return nil, err
	}
	// Shelly documents id as required; a request without one, such as a
	// notification, gets a generated id.
	if _, ok := reqBody["id"]; !ok {
		reqBody["id"] = h.requestID.Add(1)
	}
	if v := rpcReq.GetJSONRPC(); v != "" {
		reqBody["jsonrpc"] = v
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
	req, err := http.NewRequestWithContext(ctx, "GET", h.baseURL+path, http.NoBody)
	if err != nil {
		return nil, RedactURLError(err)
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

// applyAuth applies authentication to the request. Digest auth adds a header
// only once the device has issued a nonce; until then the request goes
// without one and the device's 401 supplies the challenge.
func (h *HTTP) applyAuth(req *http.Request) error {
	h.mu.RLock()
	authType := h.opts.authType
	h.mu.RUnlock()

	switch authType {
	case authTypeBasic:
		req.SetBasicAuth(h.opts.username, h.opts.password)
	case authTypeDigest:
		if h.digest == nil || !h.digest.Ready() {
			return nil
		}
		header, err := h.digest.Header(req.Method, req.URL.RequestURI())
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", header)
	}

	return nil
}

// shouldRetry determines if an error should be retried.
func (h *HTTP) shouldRetry(err error) bool {
	// Don't retry auth errors or not found errors (use errors.Is for wrapped errors)
	if errors.Is(err, types.ErrAuth) || errors.Is(err, types.ErrNotFound) {
		return false
	}

	// A platform that cannot bind to an interface will not gain the ability on a retry.
	if errors.Is(err, types.ErrNotSupported) {
		return false
	}

	// Don't retry context errors (use errors.Is for wrapped errors)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Retry timeout and network errors
	return true
}

// Get makes a simple HTTP GET request to the given path.
// This is a convenience method for REST-style API calls that works
// for all device generations.
//
// Example:
//
//	result, err := transport.Get(ctx, "/status")          // Gen1
//	result, err := transport.Get(ctx, "/rpc/Shelly.GetStatus")  // Gen2+
func (h *HTTP) Get(ctx context.Context, path string) (json.RawMessage, error) {
	return h.Call(ctx, NewSimpleRequest(path))
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
