package integrator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenManager_NeedsRefresh(t *testing.T) {
	client := New("tag", "token")
	tm := NewTokenManager(client)

	// No auth data - needs refresh
	if !tm.NeedsRefresh() {
		t.Error("NeedsRefresh() = false, want true (no auth data)")
	}

	// Valid token - doesn't need refresh
	client.authData = &AuthData{
		Token:     "valid",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	if tm.NeedsRefresh() {
		t.Error("NeedsRefresh() = true, want false (valid token)")
	}

	// Token expiring soon - needs refresh
	client.authData = &AuthData{
		Token:     "expiring-soon",
		ExpiresAt: time.Now().Add(2 * time.Minute).Unix(),
	}
	if !tm.NeedsRefresh() {
		t.Error("NeedsRefresh() = false, want true (expiring soon)")
	}
}

func TestTokenManager_SetRefreshBuffer(t *testing.T) {
	client := New("tag", "token")
	tm := NewTokenManager(client)

	tm.SetRefreshBuffer(10 * time.Minute)

	if tm.refreshBuffer != 10*time.Minute {
		t.Errorf("refreshBuffer = %v, want 10m", tm.refreshBuffer)
	}
}

func TestTokenManager_EnsureValid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := AuthResponse{
			IsOK: true,
			Data: &AuthData{
				Token:     "new-token",
				ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithOptions("tag", "token", server.URL, nil)
	tm := NewTokenManager(client)

	err := tm.EnsureValid(context.Background())
	if err != nil {
		t.Fatalf("EnsureValid() error = %v", err)
	}

	if !client.IsAuthenticated() {
		t.Error("client not authenticated after EnsureValid()")
	}
}

func TestTokenManager_EnsureValid_AlreadyValid(t *testing.T) {
	client := New("tag", "token")
	client.authData = &AuthData{
		Token:     "valid",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	tm := NewTokenManager(client)

	err := tm.EnsureValid(context.Background())
	if err != nil {
		t.Fatalf("EnsureValid() error = %v", err)
	}

	// Token should be unchanged
	if client.authData.Token != "valid" {
		t.Error("token should not have changed")
	}
}

func TestTokenManager_StartStopAutoRefresh(t *testing.T) {
	client := New("tag", "token")
	client.authData = &AuthData{
		Token:     "valid",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	tm := NewTokenManager(client)

	ctx, cancel := context.WithCancel(context.Background())

	tm.StartAutoRefresh(ctx)

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	if !tm.refreshRunning {
		t.Error("refreshRunning = false, want true")
	}

	// Starting again should be no-op
	tm.StartAutoRefresh(ctx)

	tm.StopAutoRefresh()

	if tm.refreshRunning {
		t.Error("refreshRunning = true after stop, want false")
	}

	// Stopping again should be no-op
	tm.StopAutoRefresh()

	cancel()
}

func TestMultiRegionAuth_AddRegion(t *testing.T) {
	ma := NewMultiRegionAuth("tag", "token")

	ma.AddRegion("eu", "https://eu.api.shelly.cloud", []string{"eu-host-1", "eu-host-2"})

	regions := ma.Regions()
	if len(regions) != 1 {
		t.Errorf("len(Regions()) = %d, want 1", len(regions))
	}
	if regions[0] != "eu" {
		t.Errorf("region = %v, want eu", regions[0])
	}
}

func TestMultiRegionAuth_SetupDefaultRegions(t *testing.T) {
	ma := NewMultiRegionAuth("tag", "token")
	ma.SetupDefaultRegions()

	regions := ma.Regions()
	if len(regions) != 2 {
		t.Errorf("len(Regions()) = %d, want 2", len(regions))
	}

	// Check EU region
	token, err := ma.GetRegionToken("eu")
	if err != ErrNotAuthenticated {
		t.Errorf("GetRegionToken() error = %v, want ErrNotAuthenticated", err)
	}
	if token != "" {
		t.Error("token should be empty before auth")
	}
}

func TestMultiRegionAuth_GetHostRegion(t *testing.T) {
	ma := NewMultiRegionAuth("tag", "token")
	ma.AddRegion("eu", "https://eu.api.shelly.cloud", []string{"shelly-13-eu.shelly.cloud", "shelly-14-eu.shelly.cloud"})
	ma.AddRegion("us", "https://us.api.shelly.cloud", []string{"shelly-13-us.shelly.cloud"})

	region, err := ma.GetHostRegion("shelly-13-eu.shelly.cloud")
	if err != nil {
		t.Fatalf("GetHostRegion() error = %v", err)
	}
	if region != "eu" {
		t.Errorf("region = %v, want eu", region)
	}

	region, err = ma.GetHostRegion("shelly-13-us.shelly.cloud")
	if err != nil {
		t.Fatalf("GetHostRegion() error = %v", err)
	}
	if region != "us" {
		t.Errorf("region = %v, want us", region)
	}

	_, err = ma.GetHostRegion("unknown-host")
	if err == nil {
		t.Error("GetHostRegion() should error for unknown host")
	}
}

func TestMultiRegionAuth_AuthenticateRegion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := AuthResponse{
			IsOK: true,
			Data: &AuthData{
				Token:     "eu-token",
				ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ma := NewMultiRegionAuth("tag", "token")
	ma.AddRegion("eu", server.URL, []string{"eu-host-1"})

	err := ma.AuthenticateRegion(context.Background(), "eu")
	if err != nil {
		t.Fatalf("AuthenticateRegion() error = %v", err)
	}

	token, err := ma.GetRegionToken("eu")
	if err != nil {
		t.Fatalf("GetRegionToken() error = %v", err)
	}
	if token != "eu-token" {
		t.Errorf("token = %v, want eu-token", token)
	}
}

func TestMultiRegionAuth_AuthenticateRegion_NotConfigured(t *testing.T) {
	ma := NewMultiRegionAuth("tag", "token")

	err := ma.AuthenticateRegion(context.Background(), "unknown")
	if err == nil {
		t.Error("AuthenticateRegion() should error for unknown region")
	}
}

func TestMultiRegionAuth_GetRegionToken_Expired(t *testing.T) {
	ma := NewMultiRegionAuth("tag", "token")
	ma.AddRegion("eu", "https://eu.api.shelly.cloud", nil)
	ma.regions["eu"].AuthData = &AuthData{
		Token:     "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}

	_, err := ma.GetRegionToken("eu")
	if err != ErrTokenExpired {
		t.Errorf("GetRegionToken() error = %v, want ErrTokenExpired", err)
	}
}

func TestMultiRegionAuth_AuthenticateAll(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := AuthResponse{
			IsOK: true,
			Data: &AuthData{
				Token:     "token",
				ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ma := NewMultiRegionAuth("tag", "token")
	ma.AddRegion("eu", server.URL, nil)
	ma.AddRegion("us", server.URL, nil)

	errors := ma.AuthenticateAll(context.Background())
	if len(errors) != 0 {
		t.Errorf("AuthenticateAll() errors = %v", errors)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

func TestServiceAccount(t *testing.T) {
	sa := &ServiceAccount{
		Name:          "test-account",
		IntegratorTag: "tag",
		Token:         "token",
		Permissions: ServiceAccountPermissions{
			CanControl:    true,
			CanReadStatus: true,
		},
		CreatedAt: time.Now(),
	}

	if sa.Name != "test-account" {
		t.Errorf("Name = %v, want test-account", sa.Name)
	}
	if !sa.Permissions.CanControl {
		t.Error("CanControl = false, want true")
	}
}

func TestAPIKey_GenerateAndValidate(t *testing.T) {
	perms := APIKeyPermissions{
		Scopes:             []string{APIKeyScopes.DeviceRead, APIKeyScopes.DeviceControl},
		RateLimitPerMinute: 100,
	}

	fullKey, apiKey, err := GenerateAPIKey("test-key", perms)
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}

	if fullKey == "" {
		t.Error("fullKey is empty")
	}
	if apiKey.Name != "test-key" {
		t.Errorf("Name = %v, want test-key", apiKey.Name)
	}
	if len(apiKey.Prefix) != 8 {
		t.Errorf("len(Prefix) = %d, want 8", len(apiKey.Prefix))
	}

	// Validate the key
	if !ValidateAPIKey(fullKey, apiKey.KeyHash) {
		t.Error("ValidateAPIKey() = false, want true")
	}

	// Invalid key should not validate
	if ValidateAPIKey("wrong-key", apiKey.KeyHash) {
		t.Error("ValidateAPIKey() = true for wrong key")
	}
}

func TestAPIKey_HasScope(t *testing.T) {
	_, apiKey, err := GenerateAPIKey("test", APIKeyPermissions{
		Scopes: []string{APIKeyScopes.DeviceRead, APIKeyScopes.DeviceControl},
	})
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}

	if !apiKey.HasScope(APIKeyScopes.DeviceRead) {
		t.Error("HasScope(DeviceRead) = false, want true")
	}
	if !apiKey.HasScope(APIKeyScopes.DeviceControl) {
		t.Error("HasScope(DeviceControl) = false, want true")
	}
	if apiKey.HasScope(APIKeyScopes.AccountManage) {
		t.Error("HasScope(AccountManage) = true, want false")
	}
}

func TestAPIKey_IsExpired(t *testing.T) {
	_, apiKey, err := GenerateAPIKey("test", APIKeyPermissions{})
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}

	// No expiry
	if apiKey.IsExpired() {
		t.Error("IsExpired() = true, want false (no expiry)")
	}

	// Expired
	past := time.Now().Add(-1 * time.Hour)
	apiKey.ExpiresAt = &past
	if !apiKey.IsExpired() {
		t.Error("IsExpired() = false, want true")
	}

	// Not expired
	future := time.Now().Add(1 * time.Hour)
	apiKey.ExpiresAt = &future
	if apiKey.IsExpired() {
		t.Error("IsExpired() = true, want false")
	}
}

func TestParseJWTClaims(t *testing.T) {
	// Create a fake JWT payload
	payload := map[string]any{
		"user_id":      "user123",
		"itg":          "integrator-tag",
		"iat":          time.Now().Unix(),
		"exp":          time.Now().Add(24 * time.Hour).Unix(),
		"user_api_url": "https://api.shelly.cloud",
	}
	payloadJSON, _ := json.Marshal(payload) //nolint:errcheck // test data generation
	payloadB64 := base64.URLEncoding.EncodeToString(payloadJSON)

	// Create fake JWT (header.payload.signature)
	fakeJWT := "eyJhbGciOiJIUzI1NiJ9." + payloadB64 + ".signature"

	claims, err := ParseJWTClaims(fakeJWT)
	if err != nil {
		t.Fatalf("ParseJWTClaims() error = %v", err)
	}

	if claims.UserID != "user123" {
		t.Errorf("UserID = %v, want user123", claims.UserID)
	}
	if claims.IntegratorTag != "integrator-tag" {
		t.Errorf("IntegratorTag = %v, want integrator-tag", claims.IntegratorTag)
	}
	if claims.UserAPIURL != "https://api.shelly.cloud" {
		t.Errorf("UserAPIURL = %v, want https://api.shelly.cloud", claims.UserAPIURL)
	}
}

func TestParseJWTClaims_InvalidFormat(t *testing.T) {
	_, err := ParseJWTClaims("not-a-jwt")
	if err == nil {
		t.Error("ParseJWTClaims() should error for invalid format")
	}

	_, err = ParseJWTClaims("only.two")
	if err == nil {
		t.Error("ParseJWTClaims() should error for only 2 parts")
	}
}

func TestParseJWTClaims_InvalidBase64(t *testing.T) {
	// Invalid base64 in payload part
	_, err := ParseJWTClaims("eyJhbGciOiJIUzI1NiJ9.!!!invalid-base64!!!.signature")
	if err == nil {
		t.Error("ParseJWTClaims() should error for invalid base64")
	}
}

func TestParseJWTClaims_InvalidJSON(t *testing.T) {
	// Valid base64 but invalid JSON
	invalidJSON := base64.URLEncoding.EncodeToString([]byte("{not valid json"))
	_, err := ParseJWTClaims("eyJhbGciOiJIUzI1NiJ9." + invalidJSON + ".signature")
	if err == nil {
		t.Error("ParseJWTClaims() should error for invalid JSON")
	}
}

func TestParseJWTClaims_PaddingVariations(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
	}{
		{
			name: "needs 2 padding chars",
			payload: map[string]any{
				"user_id": "a",
			},
		},
		{
			name: "needs 1 padding char",
			payload: map[string]any{
				"user_id": "ab",
			},
		},
		{
			name: "needs 0 padding chars",
			payload: map[string]any{
				"user_id": "abcd",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadJSON, _ := json.Marshal(tt.payload)
			// Use RawURLEncoding (no padding) to test padding logic
			payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
			fakeJWT := "eyJhbGciOiJIUzI1NiJ9." + payloadB64 + ".signature"

			claims, err := ParseJWTClaims(fakeJWT)
			if err != nil {
				t.Fatalf("ParseJWTClaims() error = %v", err)
			}
			if claims.UserID != tt.payload["user_id"] {
				t.Errorf("UserID = %v, want %v", claims.UserID, tt.payload["user_id"])
			}
		})
	}
}

func TestJWTClaims_IsExpired(t *testing.T) {
	claims := &JWTClaims{
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	if !claims.IsExpired() {
		t.Error("IsExpired() = false, want true")
	}

	claims.ExpiresAt = time.Now().Add(1 * time.Hour).Unix()
	if claims.IsExpired() {
		t.Error("IsExpired() = true, want false")
	}
}

func TestJWTClaims_ExpiresTime(t *testing.T) {
	now := time.Now()
	claims := &JWTClaims{
		ExpiresAt: now.Unix(),
	}

	expires := claims.ExpiresTime()
	if expires.Unix() != now.Unix() {
		t.Errorf("ExpiresTime() = %v, want %v", expires, now)
	}
}

func TestJWTClaims_TimeUntilExpiry(t *testing.T) {
	claims := &JWTClaims{
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}

	ttl := claims.TimeUntilExpiry()
	if ttl < 59*time.Minute || ttl > 61*time.Minute {
		t.Errorf("TimeUntilExpiry() = %v, expected ~1 hour", ttl)
	}
}

func TestCallbackTokenVerifier_VerifyCallbackToken(t *testing.T) {
	// Create a valid callback token
	payload := map[string]any{
		"exp": time.Now().Add(1 * time.Minute).Unix(),
	}
	payloadJSON, _ := json.Marshal(payload) //nolint:errcheck // test data generation
	payloadB64 := base64.URLEncoding.EncodeToString(payloadJSON)
	fakeToken := "eyJhbGciOiJIUzI1NiJ9." + payloadB64 + ".signature"

	verifier := &CallbackTokenVerifier{}
	callback, err := verifier.VerifyCallbackToken(fakeToken)
	if err != nil {
		t.Fatalf("VerifyCallbackToken() error = %v", err)
	}
	if callback.Token != fakeToken {
		t.Error("Token not preserved")
	}
}

func TestCallbackTokenVerifier_VerifyCallbackToken_Expired(t *testing.T) {
	payload := map[string]any{
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	}
	payloadJSON, _ := json.Marshal(payload) //nolint:errcheck // test data generation
	payloadB64 := base64.URLEncoding.EncodeToString(payloadJSON)
	fakeToken := "eyJhbGciOiJIUzI1NiJ9." + payloadB64 + ".signature"

	verifier := &CallbackTokenVerifier{}
	_, err := verifier.VerifyCallbackToken(fakeToken)
	if err == nil {
		t.Error("VerifyCallbackToken() should error for expired token")
	}
}

func TestTokenManager_NilClient(t *testing.T) {
	tm := NewTokenManager(nil)
	if tm.NeedsRefresh() {
		t.Error("NeedsRefresh() = true, want false (nil client)")
	}
}
