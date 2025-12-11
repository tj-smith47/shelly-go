package cloud

import (
	"bytes"
	"crypto/sha1" //nolint:gosec // G505: SHA1 is required by Shelly Cloud API protocol
	"encoding/base64"
	"encoding/hex"
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

// Common authentication errors.
var (
	// ErrInvalidToken indicates the token is invalid or malformed.
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired indicates the token has expired.
	ErrTokenExpired = errors.New("token expired")

	// ErrNoUserAPIURL indicates the token is missing the user_api_url field.
	ErrNoUserAPIURL = errors.New("token missing user_api_url")

	// ErrInvalidCredentials indicates invalid email or password.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// HashPassword returns the SHA1 hash of a password.
// The Shelly Cloud API expects passwords to be hashed with SHA1.
func HashPassword(password string) string {
	hash := sha1.Sum([]byte(password)) //nolint:gosec // G401: SHA1 required by Shelly Cloud API
	return hex.EncodeToString(hash[:])
}

// AuthorizeURL returns the OAuth authorization URL for the given client ID
// and redirect URI.
func AuthorizeURL(clientID, redirectURI string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	return OAuthAuthorizeURL + "?" + params.Encode()
}

// ParseToken parses a JWT token string and extracts the claims.
// The token is a JWT with three parts separated by '.'.
// We decode the middle part (payload) to extract the claims.
func ParseToken(tokenString string) (*Token, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	// Split the JWT into its three parts
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	// Decode the payload (middle part)
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}

	// Parse the claims
	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	// Create the token
	token := &Token{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		UserAPIURL:  claims.UserAPIURL,
	}

	// Set expiry if present
	if claims.ExpiresAt > 0 {
		token.Expiry = time.Unix(claims.ExpiresAt, 0)
	}

	return token, nil
}

// base64URLDecode decodes a base64 URL-encoded string.
// JWT uses base64 URL encoding without padding.
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	// Replace URL-safe characters
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	return base64.StdEncoding.DecodeString(s)
}

// ExtractUserAPIURL extracts the user_api_url from a JWT token.
// This is the designated server for API calls for this user.
func ExtractUserAPIURL(tokenString string) (string, error) {
	token, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}

	if token.UserAPIURL == "" {
		return "", ErrNoUserAPIURL
	}

	return token.UserAPIURL, nil
}

// ValidateToken checks if a token is valid and not expired.
func ValidateToken(tokenString string) error {
	token, err := ParseToken(tokenString)
	if err != nil {
		return err
	}

	if !token.Valid() {
		return ErrTokenExpired
	}

	if token.UserAPIURL == "" {
		return ErrNoUserAPIURL
	}

	return nil
}

// TokenExpiry returns the expiry time of a JWT token.
// Returns the zero time if the token has no expiry.
func TokenExpiry(tokenString string) (time.Time, error) {
	token, err := ParseToken(tokenString)
	if err != nil {
		return time.Time{}, err
	}
	return token.Expiry, nil
}

// IsTokenExpired checks if a JWT token is expired.
func IsTokenExpired(tokenString string) bool {
	expiry, err := TokenExpiry(tokenString)
	if err != nil {
		return true // Invalid tokens are considered expired
	}
	if expiry.IsZero() {
		return false // No expiry set, assume valid
	}
	return time.Now().After(expiry)
}

// TimeUntilExpiry returns the duration until the token expires.
// Returns zero duration if the token is already expired or has no expiry.
func TimeUntilExpiry(tokenString string) time.Duration {
	expiry, err := TokenExpiry(tokenString)
	if err != nil {
		return 0
	}
	if expiry.IsZero() {
		return 0 // No expiry set
	}
	duration := time.Until(expiry)
	if duration < 0 {
		return 0
	}
	return duration
}

// ShouldRefresh returns true if the token should be refreshed.
// A token should be refreshed if it expires within the given threshold.
func ShouldRefresh(tokenString string, threshold time.Duration) bool {
	remaining := TimeUntilExpiry(tokenString)
	if remaining == 0 {
		// Either expired or no expiry
		return IsTokenExpired(tokenString)
	}
	return remaining < threshold
}

// TokenSource is an interface for retrieving tokens.
// This can be used for automatic token refresh.
type TokenSource interface {
	// Token returns a valid token. If the current token is expired
	// or about to expire, it should return a refreshed token.
	Token() (*Token, error)
}

// staticTokenSource is a TokenSource that always returns the same token.
type staticTokenSource struct {
	token *Token
}

// StaticTokenSource returns a TokenSource that always returns the same token.
func StaticTokenSource(token *Token) TokenSource {
	return &staticTokenSource{token: token}
}

// Token returns the static token.
func (s *staticTokenSource) Token() (*Token, error) {
	if s.token == nil {
		return nil, ErrInvalidToken
	}
	return s.token, nil
}

// credentialTokenSource refreshes tokens using email/password credentials.
type credentialTokenSource struct {
	httpClient   *http.Client
	token        *Token
	email        string
	passwordSHA1 string
	clientID     string
	tokenURL     string
	threshold    time.Duration
	mu           sync.Mutex
}

// CredentialTokenSourceOption is an option for configuring a credentialTokenSource.
type CredentialTokenSourceOption func(*credentialTokenSource)

// WithCredentialHTTPClient sets the HTTP client for the token source.
func WithCredentialHTTPClient(client *http.Client) CredentialTokenSourceOption {
	return func(ts *credentialTokenSource) {
		ts.httpClient = client
	}
}

// WithCredentialClientID sets the OAuth client ID.
func WithCredentialClientID(clientID string) CredentialTokenSourceOption {
	return func(ts *credentialTokenSource) {
		ts.clientID = clientID
	}
}

// WithRefreshThreshold sets the threshold for automatic refresh.
// If the token expires within this duration, it will be refreshed.
func WithRefreshThreshold(threshold time.Duration) CredentialTokenSourceOption {
	return func(ts *credentialTokenSource) {
		ts.threshold = threshold
	}
}

// WithTokenURL sets a custom token URL (for testing).
func WithTokenURL(tokenURL string) CredentialTokenSourceOption {
	return func(ts *credentialTokenSource) {
		ts.tokenURL = tokenURL
	}
}

// CredentialTokenSource returns a TokenSource that refreshes tokens using
// email and password credentials. The token will be automatically refreshed
// when it expires or is about to expire (within the threshold).
func CredentialTokenSource(email, passwordSHA1 string, opts ...CredentialTokenSourceOption) TokenSource {
	ts := &credentialTokenSource{
		email:        email,
		passwordSHA1: passwordSHA1,
		clientID:     ClientIDDIY,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		threshold:    5 * time.Minute, // Default: refresh 5 minutes before expiry
	}

	for _, opt := range opts {
		opt(ts)
	}

	return ts
}

// Token returns a valid token, refreshing if necessary.
func (ts *credentialTokenSource) Token() (*Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Check if we have a valid token that doesn't need refresh
	if ts.token != nil && ts.token.Valid() {
		if !ShouldRefresh(ts.token.AccessToken, ts.threshold) {
			return ts.token, nil
		}
	}

	// Need to refresh the token
	token, err := ts.refresh()
	if err != nil {
		return nil, err
	}

	ts.token = token
	return token, nil
}

// refresh obtains a new token using credentials.
func (ts *credentialTokenSource) refresh() (*Token, error) {
	reqBody := LoginRequest{
		Email:    ts.email,
		Password: ts.passwordSHA1,
		ClientID: ts.clientID,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	tokenURL := ts.tokenURL
	if tokenURL == "" {
		tokenURL = OAuthTokenURL
	}

	req, err := http.NewRequest(http.MethodPost, tokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var loginResp LoginResponse
	if unmarshalErr := json.Unmarshal(respBody, &loginResp); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse response: %w", unmarshalErr)
	}

	if !loginResp.IsOK {
		if len(loginResp.Errors) > 0 {
			return nil, fmt.Errorf("%w: %s", ErrInvalidCredentials, strings.Join(loginResp.Errors, ", "))
		}
		return nil, ErrInvalidCredentials
	}

	if loginResp.Data == nil || loginResp.Data.Token == "" {
		return nil, errors.New("no token in response")
	}

	token, err := ParseToken(loginResp.Data.Token)
	if err != nil {
		return nil, err
	}

	if loginResp.Data.UserAPIURL != "" {
		token.UserAPIURL = loginResp.Data.UserAPIURL
	}

	return token, nil
}
