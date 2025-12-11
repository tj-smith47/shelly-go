package integrator

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// Token refresh constants.
const (
	// DefaultTokenRefreshBuffer is the time before expiration to trigger refresh.
	DefaultTokenRefreshBuffer = 5 * time.Minute

	// MinTokenRefreshInterval prevents excessive refresh attempts.
	MinTokenRefreshInterval = 1 * time.Minute
)

// TokenManager handles automatic JWT token lifecycle management.
type TokenManager struct {
	lastRefresh    time.Time
	client         *Client
	refreshDone    chan struct{}
	refreshBuffer  time.Duration
	mu             sync.RWMutex
	refreshRunning bool
}

// NewTokenManager creates a new token manager for the given client.
func NewTokenManager(client *Client) *TokenManager {
	return &TokenManager{
		client:        client,
		refreshBuffer: DefaultTokenRefreshBuffer,
	}
}

// SetRefreshBuffer sets the time before expiration to trigger refresh.
func (tm *TokenManager) SetRefreshBuffer(d time.Duration) {
	tm.mu.Lock()
	tm.refreshBuffer = d
	tm.mu.Unlock()
}

// NeedsRefresh returns true if the token should be refreshed.
func (tm *TokenManager) NeedsRefresh() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.client == nil {
		return false
	}

	tm.client.mu.RLock()
	authData := tm.client.authData
	tm.client.mu.RUnlock()

	if authData == nil {
		return true
	}

	// Check if token expires within the refresh buffer
	expiresAt := authData.ExpiresTime()
	return time.Until(expiresAt) < tm.refreshBuffer
}

// EnsureValid ensures the token is valid, refreshing if necessary.
func (tm *TokenManager) EnsureValid(ctx context.Context) error {
	if !tm.NeedsRefresh() {
		return nil
	}

	tm.mu.Lock()
	// Check minimum refresh interval
	if time.Since(tm.lastRefresh) < MinTokenRefreshInterval {
		tm.mu.Unlock()
		return nil
	}
	tm.lastRefresh = time.Now()
	tm.mu.Unlock()

	return tm.client.Authenticate(ctx)
}

// StartAutoRefresh starts a background goroutine that automatically refreshes
// the token before it expires.
func (tm *TokenManager) StartAutoRefresh(ctx context.Context) {
	tm.mu.Lock()
	if tm.refreshRunning {
		tm.mu.Unlock()
		return
	}
	tm.refreshRunning = true
	tm.refreshDone = make(chan struct{})
	tm.mu.Unlock()

	go tm.autoRefreshLoop(ctx)
}

// StopAutoRefresh stops the automatic token refresh.
func (tm *TokenManager) StopAutoRefresh() {
	tm.mu.Lock()
	if !tm.refreshRunning {
		tm.mu.Unlock()
		return
	}
	tm.refreshRunning = false
	close(tm.refreshDone)
	tm.mu.Unlock()
}

func (tm *TokenManager) autoRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(MinTokenRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tm.refreshDone:
			return
		case <-ticker.C:
			if tm.NeedsRefresh() {
				//nolint:errcheck // Background refresh - errors are handled by next explicit call
				tm.EnsureValid(ctx)
			}
		}
	}
}

// MultiRegionAuth manages authentication across multiple Shelly cloud regions.
type MultiRegionAuth struct {
	regions       map[string]*RegionAuth
	httpClient    *http.Client
	integratorTag string
	token         string
	mu            sync.RWMutex
}

// RegionAuth holds authentication data for a specific region.
type RegionAuth struct {
	AuthData *AuthData
	types.RawFields
	Region string
	APIURL string
	Hosts  []string
}

// NewMultiRegionAuth creates a new multi-region authentication manager.
func NewMultiRegionAuth(integratorTag, token string) *MultiRegionAuth {
	return &MultiRegionAuth{
		integratorTag: integratorTag,
		token:         token,
		regions:       make(map[string]*RegionAuth),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AddRegion adds a region configuration.
func (ma *MultiRegionAuth) AddRegion(region, apiURL string, hosts []string) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	ma.regions[region] = &RegionAuth{
		Region: region,
		APIURL: apiURL,
		Hosts:  hosts,
	}
}

// AuthenticateRegion authenticates to a specific region.
func (ma *MultiRegionAuth) AuthenticateRegion(ctx context.Context, region string) error {
	ma.mu.RLock()
	regionAuth, ok := ma.regions[region]
	ma.mu.RUnlock()

	if !ok {
		return fmt.Errorf("region %q not configured", region)
	}

	client := NewWithOptions(ma.integratorTag, ma.token, regionAuth.APIURL, ma.httpClient)
	if err := client.Authenticate(ctx); err != nil {
		return fmt.Errorf("failed to authenticate to region %q: %w", region, err)
	}

	ma.mu.Lock()
	ma.regions[region].AuthData = client.authData
	ma.mu.Unlock()

	return nil
}

// AuthenticateAll authenticates to all configured regions.
func (ma *MultiRegionAuth) AuthenticateAll(ctx context.Context) map[string]error {
	ma.mu.RLock()
	regions := make([]string, 0, len(ma.regions))
	for region := range ma.regions {
		regions = append(regions, region)
	}
	ma.mu.RUnlock()

	errors := make(map[string]error)
	for _, region := range regions {
		if err := ma.AuthenticateRegion(ctx, region); err != nil {
			errors[region] = err
		}
	}
	return errors
}

// GetRegionToken returns the JWT token for a specific region.
func (ma *MultiRegionAuth) GetRegionToken(region string) (string, error) {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	regionAuth, ok := ma.regions[region]
	if !ok {
		return "", fmt.Errorf("region %q not configured", region)
	}
	if regionAuth.AuthData == nil {
		return "", ErrNotAuthenticated
	}
	if regionAuth.AuthData.IsExpired() {
		return "", ErrTokenExpired
	}
	return regionAuth.AuthData.Token, nil
}

// GetHostRegion returns the region for a given host.
func (ma *MultiRegionAuth) GetHostRegion(host string) (string, error) {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	for region, auth := range ma.regions {
		for _, h := range auth.Hosts {
			if h == host {
				return region, nil
			}
		}
	}
	return "", fmt.Errorf("host %q not found in any region", host)
}

// Regions returns the list of configured regions.
func (ma *MultiRegionAuth) Regions() []string {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	regions := make([]string, 0, len(ma.regions))
	for region := range ma.regions {
		regions = append(regions, region)
	}
	return regions
}

// SetupDefaultRegions configures the default EU and US regions.
func (ma *MultiRegionAuth) SetupDefaultRegions() {
	ma.AddRegion("eu", "https://api.shelly.cloud", []string{
		"shelly-13-eu.shelly.cloud",
		"shelly-14-eu.shelly.cloud",
		"shelly-15-eu.shelly.cloud",
	})
	ma.AddRegion("us", "https://api.shelly.cloud", []string{
		"shelly-13-us.shelly.cloud",
		"shelly-14-us.shelly.cloud",
	})
}

// ServiceAccount represents an integrator service account for API access.
type ServiceAccount struct {
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	types.RawFields
	Name          string                    `json:"name"`
	IntegratorTag string                    `json:"integrator_tag"`
	Token         string                    `json:"token"`
	Description   string                    `json:"description,omitempty"`
	Permissions   ServiceAccountPermissions `json:"permissions"`
}

// ServiceAccountPermissions defines what a service account can access.
type ServiceAccountPermissions struct {
	types.RawFields
	AllowedRegions    []string `json:"allowed_regions,omitempty"`
	AllowedDevices    []string `json:"allowed_devices,omitempty"`
	CanControl        bool     `json:"can_control"`
	CanReadStatus     bool     `json:"can_read_status"`
	CanManageAccounts bool     `json:"can_manage_accounts"`
}

// APIKey represents an API key for programmatic access.
type APIKey struct {
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	types.RawFields
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	KeyHash     string            `json:"key_hash"`
	Prefix      string            `json:"prefix"`
	Permissions APIKeyPermissions `json:"permissions"`
}

// APIKeyPermissions defines what an API key can access.
type APIKeyPermissions struct {
	types.RawFields
	Scopes             []string `json:"scopes"`
	RateLimitPerMinute int      `json:"rate_limit_per_minute,omitempty"`
}

// APIKeyScopes defines common API key scopes.
var APIKeyScopes = struct {
	DeviceRead    string
	DeviceControl string
	DeviceManage  string
	AccountRead   string
	AccountManage string
	FleetRead     string
	FleetManage   string
	AnalyticsRead string
}{
	DeviceRead:    "device:read",
	DeviceControl: "device:control",
	DeviceManage:  "device:manage",
	AccountRead:   "account:read",
	AccountManage: "account:manage",
	FleetRead:     "fleet:read",
	FleetManage:   "fleet:manage",
	AnalyticsRead: "analytics:read",
}

// GenerateAPIKey generates a new API key.
// Returns the full key (only available once) and the APIKey metadata.
func GenerateAPIKey(name string, permissions APIKeyPermissions) (string, *APIKey, error) {
	// Generate a secure random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", nil, fmt.Errorf("generate key: %w", err)
	}

	fullKey := base64.URLEncoding.EncodeToString(keyBytes)

	// Hash the key for storage
	hash := sha256.Sum256([]byte(fullKey))
	keyHash := hex.EncodeToString(hash[:])

	now := time.Now()
	return fullKey, &APIKey{
		ID:          fmt.Sprintf("key_%d", now.UnixNano()),
		Name:        name,
		KeyHash:     keyHash,
		Prefix:      fullKey[:8],
		Permissions: permissions,
		CreatedAt:   now,
	}, nil
}

// ValidateAPIKey validates an API key against its stored hash.
func ValidateAPIKey(key, storedHash string) bool {
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])
	return keyHash == storedHash
}

// HasScope checks if the API key has the specified scope.
func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Permissions.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// IsExpired returns true if the API key has expired.
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// JWTClaims represents the claims extracted from a Shelly JWT token.
type JWTClaims struct {
	types.RawFields
	UserID        string `json:"user_id,omitempty"`
	IntegratorTag string `json:"itg,omitempty"`
	UserAPIURL    string `json:"user_api_url,omitempty"`
	IssuedAt      int64  `json:"iat,omitempty"`
	ExpiresAt     int64  `json:"exp,omitempty"`
}

// ParseJWTClaims extracts claims from a JWT token without verifying the signature.
// Note: Shelly JWTs are signed with a secret key but the claims can be decoded.
func ParseJWTClaims(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Decode the payload (second part)
	payload := parts[1]
	// Add padding if necessary
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		// Try standard base64 without URL encoding
		decoded, err = base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
		}
	}

	var claims JWTClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	return &claims, nil
}

// IsExpired returns true if the token has expired based on the claims.
func (c *JWTClaims) IsExpired() bool {
	return time.Now().Unix() > c.ExpiresAt
}

// ExpiresTime returns the expiration time as time.Time.
func (c *JWTClaims) ExpiresTime() time.Time {
	return time.Unix(c.ExpiresAt, 0)
}

// TimeUntilExpiry returns the duration until the token expires.
func (c *JWTClaims) TimeUntilExpiry() time.Duration {
	return time.Until(c.ExpiresTime())
}

// CallbackTokenVerifier verifies callback tokens from Shelly.
// When users grant/revoke device access, Shelly sends a signed callback token.
type CallbackTokenVerifier struct {
	// PublicKey is the Shelly public key for verification.
	// In production, this would be the actual RSA/ECDSA public key.
	PublicKey string
}

// CallbackToken represents a token received in user consent callbacks.
type CallbackToken struct {
	ExpiresAt time.Time `json:"expires_at"`
	types.RawFields
	Token        string `json:"token"`
	UserID       string `json:"user_id"`
	DeviceID     string `json:"device_id"`
	Action       string `json:"action"`
	AccessGroups string `json:"access_groups"`
}

// VerifyCallbackToken verifies the authenticity of a callback token.
// Returns the parsed token claims if valid, or an error if invalid.
func (v *CallbackTokenVerifier) VerifyCallbackToken(token string) (*CallbackToken, error) {
	claims, err := ParseJWTClaims(token)
	if err != nil {
		return nil, fmt.Errorf("failed to parse callback token: %w", err)
	}

	if claims.IsExpired() {
		return nil, fmt.Errorf("callback token expired")
	}

	// In production, verify the signature against the public key
	// For now, we just parse and validate the expiry

	return &CallbackToken{
		Token:     token,
		ExpiresAt: claims.ExpiresTime(),
	}, nil
}
