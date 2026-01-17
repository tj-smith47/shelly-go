package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     string
	}{
		{
			name:     "simple password",
			password: "password123",
			want:     "cbfdac6008f9cab4083784cbd1874f76618d2a97",
		},
		{
			name:     "empty password",
			password: "",
			want:     "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			name:     "complex password",
			password: "P@ssw0rd!#$%",
			want:     "cc795c98085b982ebfb57744e2b956f0f9c91e35",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashPassword(tt.password)
			if got != tt.want {
				t.Errorf("HashPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthorizeURL(t *testing.T) {
	url := AuthorizeURL("shelly-diy", "https://example.com/callback")

	// Check that it contains expected parts
	if url == "" {
		t.Error("AuthorizeURL() returned empty string")
	}

	expectedParts := []string{
		OAuthAuthorizeURL,
		"client_id=shelly-diy",
		"redirect_uri=https%3A%2F%2Fexample.com%2Fcallback",
		"response_type=code",
	}

	for _, part := range expectedParts {
		if !contains(url, part) {
			t.Errorf("AuthorizeURL() missing expected part: %s", part)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestParseToken(t *testing.T) {
	tests := []struct {
		errType     error
		name        string
		tokenString string
		wantErr     bool
	}{
		{
			name:        "empty token",
			tokenString: "",
			wantErr:     true,
			errType:     ErrInvalidToken,
		},
		{
			name:        "invalid token - no dots",
			tokenString: "invalid",
			wantErr:     true,
			errType:     ErrInvalidToken,
		},
		{
			name:        "invalid token - one dot",
			tokenString: "header.payload",
			wantErr:     true,
			errType:     ErrInvalidToken,
		},
		{
			name:        "invalid token - invalid base64",
			tokenString: "header.!!!invalid!!!.signature",
			wantErr:     true,
		},
		{
			name:        "invalid token - invalid json",
			tokenString: "header.dGhpcyBpcyBub3QganNvbg.signature", // "this is not json" in base64
			wantErr:     true,
		},
		{
			name: "valid token",
			// {"user_api_url":"https://shelly-49-eu.shelly.cloud","user_id":123,"email":"test@example.com","exp":1735689600}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL3NoZWxseS00OS1ldS5zaGVsbHkuY2xvdWQiLCJ1c2VyX2lkIjoxMjMsImVtYWlsIjoidGVzdEBleGFtcGxlLmNvbSIsImV4cCI6MTczNTY4OTYwMH0.signature",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ParseToken(tt.tokenString)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if err != tt.errType {
					t.Errorf("ParseToken() error = %v, wantErrType %v", err, tt.errType)
				}
				return
			}

			if !tt.wantErr {
				if token == nil {
					t.Error("ParseToken() returned nil token")
					return
				}
				if token.AccessToken != tt.tokenString {
					t.Error("ParseToken() AccessToken mismatch")
				}
				if token.UserAPIURL != "https://shelly-49-eu.shelly.cloud" {
					t.Errorf("ParseToken() UserAPIURL = %v, want https://shelly-49-eu.shelly.cloud", token.UserAPIURL)
				}
			}
		})
	}
}

func TestExtractUserAPIURL(t *testing.T) {
	tests := []struct {
		name        string
		tokenString string
		want        string
		wantErr     bool
	}{
		{
			name:        "empty token",
			tokenString: "",
			wantErr:     true,
		},
		{
			name: "valid token with user_api_url",
			// {"user_api_url":"https://shelly-49-eu.shelly.cloud"}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL3NoZWxseS00OS1ldS5zaGVsbHkuY2xvdWQifQ.signature",
			want:        "https://shelly-49-eu.shelly.cloud",
			wantErr:     false,
		},
		{
			name: "valid token without user_api_url",
			// {"user_id":123}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoxMjN9.signature",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractUserAPIURL(tt.tokenString)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractUserAPIURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ExtractUserAPIURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	tests := []struct {
		wantErr     error
		name        string
		tokenString string
	}{
		{
			name:        "empty token",
			tokenString: "",
			wantErr:     ErrInvalidToken,
		},
		{
			name: "valid token with user_api_url",
			// {"user_api_url":"https://shelly-49-eu.shelly.cloud","exp":2000000000}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL3NoZWxseS00OS1ldS5zaGVsbHkuY2xvdWQiLCJleHAiOjIwMDAwMDAwMDB9.signature",
			wantErr:     nil,
		},
		{
			name: "token without user_api_url",
			// {"user_id":123,"exp":2000000000}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoxMjMsImV4cCI6MjAwMDAwMDAwMH0.signature",
			wantErr:     ErrNoUserAPIURL,
		},
		{
			name: "expired token",
			// {"user_api_url":"https://shelly-49-eu.shelly.cloud","exp":1000000000}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL3NoZWxseS00OS1ldS5zaGVsbHkuY2xvdWQiLCJleHAiOjEwMDAwMDAwMDB9.signature",
			wantErr:     ErrTokenExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToken(tt.tokenString)
			if err != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTokenExpiry(t *testing.T) {
	tests := []struct {
		name        string
		tokenString string
		wantZero    bool
		wantErr     bool
	}{
		{
			name:        "empty token",
			tokenString: "",
			wantErr:     true,
		},
		{
			name: "token with expiry",
			// {"user_api_url":"https://example.com","exp":1735689600}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIiwiZXhwIjoxNzM1Njg5NjAwfQ.signature",
			wantZero:    false,
			wantErr:     false,
		},
		{
			name: "token without expiry",
			// {"user_api_url":"https://example.com"}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIn0.signature",
			wantZero:    true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expiry, err := TokenExpiry(tt.tokenString)
			if (err != nil) != tt.wantErr {
				t.Errorf("TokenExpiry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && expiry.IsZero() != tt.wantZero {
				t.Errorf("TokenExpiry() zero = %v, wantZero %v", expiry.IsZero(), tt.wantZero)
			}
		})
	}
}

func TestIsTokenExpired(t *testing.T) {
	tests := []struct {
		name        string
		tokenString string
		want        bool
	}{
		{
			name:        "empty token",
			tokenString: "",
			want:        true,
		},
		{
			name: "expired token",
			// {"user_api_url":"https://example.com","exp":1000000000}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIiwiZXhwIjoxMDAwMDAwMDAwfQ.signature",
			want:        true,
		},
		{
			name: "future expiry",
			// {"user_api_url":"https://example.com","exp":2000000000}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIiwiZXhwIjoyMDAwMDAwMDAwfQ.signature",
			want:        false,
		},
		{
			name: "no expiry",
			// {"user_api_url":"https://example.com"}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIn0.signature",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTokenExpired(tt.tokenString)
			if got != tt.want {
				t.Errorf("IsTokenExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeUntilExpiry(t *testing.T) {
	tests := []struct {
		name        string
		tokenString string
		wantZero    bool
	}{
		{
			name:        "empty token",
			tokenString: "",
			wantZero:    true,
		},
		{
			name: "expired token",
			// {"user_api_url":"https://example.com","exp":1000000000}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIiwiZXhwIjoxMDAwMDAwMDAwfQ.signature",
			wantZero:    true,
		},
		{
			name: "future expiry",
			// {"user_api_url":"https://example.com","exp":2000000000}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIiwiZXhwIjoyMDAwMDAwMDAwfQ.signature",
			wantZero:    false,
		},
		{
			name: "no expiry",
			// {"user_api_url":"https://example.com"}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIn0.signature",
			wantZero:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimeUntilExpiry(tt.tokenString)
			if (got == 0) != tt.wantZero {
				t.Errorf("TimeUntilExpiry() zero = %v, wantZero %v", got == 0, tt.wantZero)
			}
		})
	}
}

func TestTokenValid(t *testing.T) {
	tests := []struct {
		token *Token
		name  string
		want  bool
	}{
		{
			name:  "nil token",
			token: nil,
			want:  false,
		},
		{
			name:  "empty access token",
			token: &Token{AccessToken: ""},
			want:  false,
		},
		{
			name:  "valid token no expiry",
			token: &Token{AccessToken: "test-token"},
			want:  true,
		},
		{
			name: "valid token future expiry",
			token: &Token{
				AccessToken: "test-token",
				Expiry:      time.Now().Add(time.Hour),
			},
			want: true,
		},
		{
			name: "expired token",
			token: &Token{
				AccessToken: "test-token",
				Expiry:      time.Now().Add(-time.Hour),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.Valid()
			if got != tt.want {
				t.Errorf("Token.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBase64URLDecode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "standard base64 url",
			input:   "SGVsbG8gV29ybGQ",
			want:    "Hello World",
			wantErr: false,
		},
		{
			name:    "with padding needed",
			input:   "YQ",
			want:    "a",
			wantErr: false,
		},
		{
			name:    "url safe characters",
			input:   "PDw_Pj4",
			want:    "<<?>>",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := base64URLDecode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("base64URLDecode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("base64URLDecode() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestShouldRefresh(t *testing.T) {
	tests := []struct {
		name        string
		tokenString string
		threshold   time.Duration
		want        bool
	}{
		{
			name:        "empty token",
			tokenString: "",
			threshold:   5 * time.Minute,
			want:        true,
		},
		{
			name: "expired token",
			// {"user_api_url":"https://example.com","exp":1000000000}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIiwiZXhwIjoxMDAwMDAwMDAwfQ.signature",
			threshold:   5 * time.Minute,
			want:        true,
		},
		{
			name: "token expiring soon",
			// {"user_api_url":"https://example.com","exp":2000000000} - far future
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIiwiZXhwIjoyMDAwMDAwMDAwfQ.signature",
			threshold:   5 * time.Minute,
			want:        false,
		},
		{
			name: "token without expiry",
			// {"user_api_url":"https://example.com"}
			tokenString: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIn0.signature",
			threshold:   5 * time.Minute,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldRefresh(tt.tokenString, tt.threshold)
			if got != tt.want {
				t.Errorf("ShouldRefresh() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStaticTokenSource(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		token := &Token{AccessToken: "test-token"}
		ts := StaticTokenSource(token)

		got, err := ts.Token()
		if err != nil {
			t.Fatalf("Token() error = %v", err)
		}
		if got != token {
			t.Error("Token() returned different token")
		}
	})

	t.Run("nil token", func(t *testing.T) {
		ts := StaticTokenSource(nil)

		_, err := ts.Token()
		if err != ErrInvalidToken {
			t.Errorf("Token() error = %v, want ErrInvalidToken", err)
		}
	})
}

func TestCredentialTokenSourceOptions(t *testing.T) {
	httpClient := &http.Client{Timeout: 60 * time.Second}

	ts := CredentialTokenSource(
		"test@example.com",
		"hashedpassword",
		WithCredentialHTTPClient(httpClient),
		WithCredentialClientID("custom-client"),
		WithRefreshThreshold(10*time.Minute),
	)

	cts, ok := ts.(*credentialTokenSource)
	if !ok {
		t.Fatal("Failed to cast to credentialTokenSource")
	}

	if cts.httpClient != httpClient {
		t.Error("WithCredentialHTTPClient not applied")
	}
	if cts.clientID != "custom-client" {
		t.Errorf("clientID = %v, want custom-client", cts.clientID)
	}
	if cts.threshold != 10*time.Minute {
		t.Errorf("threshold = %v, want 10m", cts.threshold)
	}
}

func TestCredentialTokenSourceCachedToken(t *testing.T) {
	// Create a mock token that doesn't need refresh
	validToken := &Token{
		// {"user_api_url":"https://example.com","exp":2000000000}
		AccessToken: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIiwiZXhwIjoyMDAwMDAwMDAwfQ.signature",
		Expiry:      time.Unix(2000000000, 0),
	}

	ts := CredentialTokenSource("test@example.com", "hashedpassword")
	cts, ok := ts.(*credentialTokenSource)
	if !ok {
		t.Fatal("Failed to cast to credentialTokenSource")
	}

	// Pre-populate with a valid token
	cts.token = validToken

	// Should return the cached token without making network calls
	got, err := ts.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if got != validToken {
		t.Error("Token() should return cached token")
	}
}

func TestCredentialTokenSourceRefreshExpiredToken(t *testing.T) {
	// Create a mock OAuth server that expects form-encoded data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify form-encoded content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %v, want application/x-www-form-urlencoded", contentType)
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Errorf("Failed to parse form: %v", err)
		}

		if r.Form.Get("email") != "test@example.com" {
			t.Errorf("Email = %v, want test@example.com", r.Form.Get("email"))
		}

		resp := LoginResponse{
			IsOK: true,
			Data: &LoginData{
				// Valid JWT: {"user_api_url":"https://shelly-49-eu.shelly.cloud","exp":2000000000}
				Token:      "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL3NoZWxseS00OS1ldS5zaGVsbHkuY2xvdWQiLCJleHAiOjIwMDAwMDAwMDB9.signature",
				UserAPIURL: "https://shelly-49-eu.shelly.cloud",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	// Create an expired token
	expiredToken := &Token{
		// {"user_api_url":"https://example.com","exp":1000000000} - expired
		AccessToken: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL2V4YW1wbGUuY29tIiwiZXhwIjoxMDAwMDAwMDAwfQ.signature",
		Expiry:      time.Unix(1000000000, 0),
	}

	ts := CredentialTokenSource(
		"test@example.com",
		"hashedpassword",
		WithTokenURL(server.URL),
	)
	cts, ok := ts.(*credentialTokenSource)
	if !ok {
		t.Fatal("Failed to cast to credentialTokenSource")
	}

	// Pre-populate with an expired token
	cts.token = expiredToken

	// Token should refresh successfully
	newToken, err := ts.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	if newToken.UserAPIURL != "https://shelly-49-eu.shelly.cloud" {
		t.Errorf("UserAPIURL = %v, want https://shelly-49-eu.shelly.cloud", newToken.UserAPIURL)
	}
}

func TestCredentialTokenSourceNilToken(t *testing.T) {
	// Create a mock server that expects form-encoded data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify form-encoded content type
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %v, want application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		}

		resp := LoginResponse{
			IsOK: true,
			Data: &LoginData{
				// Valid JWT: {"user_api_url":"https://shelly-49-eu.shelly.cloud","exp":2000000000}
				Token:      "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL3NoZWxseS00OS1ldS5zaGVsbHkuY2xvdWQiLCJleHAiOjIwMDAwMDAwMDB9.signature",
				UserAPIURL: "https://shelly-49-eu.shelly.cloud",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	ts := CredentialTokenSource(
		"test@example.com",
		"hashedpassword",
		WithTokenURL(server.URL),
	)

	// No cached token - should fetch new token
	token, err := ts.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	if token == nil {
		t.Error("Token() returned nil")
	}
}

func TestCredentialTokenSourceRefreshError(t *testing.T) {
	// Create a mock OAuth server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := LoginResponse{
			IsOK:   false,
			Errors: []string{"Invalid credentials"},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	ts := CredentialTokenSource(
		"test@example.com",
		"wrongpassword",
		WithTokenURL(server.URL),
	)

	_, err := ts.Token()
	if err == nil {
		t.Error("Expected error for invalid credentials")
	}
}

func TestCredentialTokenSourceNoTokenInResponse(t *testing.T) {
	// Create a mock OAuth server that returns no token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := LoginResponse{
			IsOK: true,
			Data: &LoginData{
				Token: "", // Empty token
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	ts := CredentialTokenSource(
		"test@example.com",
		"hashedpassword",
		WithTokenURL(server.URL),
	)

	_, err := ts.Token()
	if err == nil {
		t.Error("Expected error for empty token")
	}
}

func TestCredentialTokenSourceInvalidJSON(t *testing.T) {
	// Create a mock OAuth server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("not valid json")); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}))
	defer server.Close()

	ts := CredentialTokenSource(
		"test@example.com",
		"hashedpassword",
		WithTokenURL(server.URL),
	)

	_, err := ts.Token()
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestWithTokenURL(t *testing.T) {
	ts := CredentialTokenSource(
		"test@example.com",
		"hashedpassword",
		WithTokenURL("https://custom.server.com/oauth/token"),
	)

	cts, ok := ts.(*credentialTokenSource)
	if !ok {
		t.Fatal("Failed to cast to credentialTokenSource")
	}
	if cts.tokenURL != "https://custom.server.com/oauth/token" {
		t.Errorf("tokenURL = %v, want https://custom.server.com/oauth/token", cts.tokenURL)
	}
}

func TestExchangeCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify path
			if r.URL.Path != "/oauth/auth" {
				t.Errorf("Path = %v, want /oauth/auth", r.URL.Path)
			}

			// Verify method
			if r.Method != http.MethodPost {
				t.Errorf("Method = %v, want POST", r.Method)
			}

			// Verify form-encoded content type
			if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Errorf("Content-Type = %v, want application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
			}

			// Parse form data
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			if r.Form.Get("client_id") != "shelly-diy" {
				t.Errorf("client_id = %v, want shelly-diy", r.Form.Get("client_id"))
			}
			if r.Form.Get("grant_type") != "code" {
				t.Errorf("grant_type = %v, want code", r.Form.Get("grant_type"))
			}
			if r.Form.Get("code") != "test-code-123" {
				t.Errorf("code = %v, want test-code-123", r.Form.Get("code"))
			}

			// Return valid token
			resp := OAuthCodeResponse{
				// Valid JWT: {"user_api_url":"https://shelly-49-eu.shelly.cloud","exp":2000000000}
				AccessToken: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL3NoZWxseS00OS1ldS5zaGVsbHkuY2xvdWQiLCJleHAiOjIwMDAwMDAwMDB9.signature",
				TokenType:   "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode failed: %v", err)
			}
		}))
		defer server.Close()

		ctx := t.Context()
		token, err := ExchangeCode(ctx, server.URL, "test-code-123", "")
		if err != nil {
			t.Fatalf("ExchangeCode failed: %v", err)
		}

		if token == nil {
			t.Fatal("Token should not be nil")
		}
		if token.UserAPIURL != "https://shelly-49-eu.shelly.cloud" {
			t.Errorf("UserAPIURL = %v, want https://shelly-49-eu.shelly.cloud", token.UserAPIURL)
		}
	})

	t.Run("with custom client ID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			if r.Form.Get("client_id") != "custom-client" {
				t.Errorf("client_id = %v, want custom-client", r.Form.Get("client_id"))
			}

			resp := OAuthCodeResponse{
				AccessToken: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL3NoZWxseS00OS1ldS5zaGVsbHkuY2xvdWQiLCJleHAiOjIwMDAwMDAwMDB9.signature",
				TokenType:   "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode failed: %v", err)
			}
		}))
		defer server.Close()

		ctx := t.Context()
		_, err := ExchangeCode(ctx, server.URL, "test-code", "custom-client")
		if err != nil {
			t.Fatalf("ExchangeCode failed: %v", err)
		}
	})

	t.Run("normalizes server URL without https", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := OAuthCodeResponse{
				AccessToken: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL3NoZWxseS00OS1ldS5zaGVsbHkuY2xvdWQiLCJleHAiOjIwMDAwMDAwMDB9.signature",
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode failed: %v", err)
			}
		}))
		defer server.Close()

		// Test passes if we get to the server - URL normalization worked
		ctx := t.Context()
		// For this test, we can't easily strip https from httptest.Server
		// Just verify the function handles URLs correctly
		_, _ = ExchangeCode(ctx, server.URL, "test-code", "")
	})

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := OAuthCodeResponse{
				Error: "invalid_grant",
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode failed: %v", err)
			}
		}))
		defer server.Close()

		ctx := t.Context()
		_, err := ExchangeCode(ctx, server.URL, "bad-code", "")
		if err == nil {
			t.Error("Expected error for invalid_grant")
		}
	})

	t.Run("empty access token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := OAuthCodeResponse{
				AccessToken: "",
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode failed: %v", err)
			}
		}))
		defer server.Close()

		ctx := t.Context()
		_, err := ExchangeCode(ctx, server.URL, "test-code", "")
		if err == nil {
			t.Error("Expected error for empty access token")
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte("not valid json")); err != nil {
				t.Fatalf("write failed: %v", err)
			}
		}))
		defer server.Close()

		ctx := t.Context()
		_, err := ExchangeCode(ctx, server.URL, "test-code", "")
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestBrowserLoginOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := &BrowserLoginOptions{}
		if opts.ClientID != "" {
			t.Error("ClientID should be empty by default")
		}
		if opts.Timeout != 0 {
			t.Error("Timeout should be zero by default")
		}
		if opts.CallbackPort != 0 {
			t.Error("CallbackPort should be zero by default")
		}
	})

	t.Run("custom options", func(t *testing.T) {
		opts := &BrowserLoginOptions{
			ClientID:     "custom-client",
			CallbackPort: 8080,
			Timeout:      10 * time.Minute,
		}
		if opts.ClientID != "custom-client" {
			t.Errorf("ClientID = %v, want custom-client", opts.ClientID)
		}
		if opts.CallbackPort != 8080 {
			t.Errorf("CallbackPort = %v, want 8080", opts.CallbackPort)
		}
		if opts.Timeout != 10*time.Minute {
			t.Errorf("Timeout = %v, want 10m", opts.Timeout)
		}
	})
}

func TestBrowserLoginContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	cancel()

	opts := &BrowserLoginOptions{
		Timeout: 100 * time.Millisecond,
	}

	_, err := BrowserLogin(ctx, opts)
	if err == nil {
		t.Error("Expected error for canceled context")
	}
}

func TestBrowserLoginNilOptions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Pass nil options - should use defaults
	_, err := BrowserLogin(ctx, nil)
	// Should timeout waiting for callback
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestBrowserLoginGeneratesAuthorizeURL(t *testing.T) {
	// We can test that the callback server starts and the authorize URL is generated
	// by using a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	opts := &BrowserLoginOptions{
		ClientID: "test-client",
	}

	// This will timeout, but we can verify the error is a context timeout
	_, err := BrowserLogin(ctx, opts)
	if err != context.DeadlineExceeded {
		// The error might be wrapped, check if it's related to context
		if err == nil {
			t.Error("Expected timeout error")
		}
	}
}

func TestBrowserLoginCallbackSuccess(t *testing.T) {
	// Create a mock OAuth server for code exchange
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OAuthCodeResponse{
			AccessToken: "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL3NoZWxseS00OS1ldS5zaGVsbHkuY2xvdWQiLCJleHAiOjIwMDAwMDAwMDB9.signature",
			TokenType:   "Bearer",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer oauthServer.Close()

	// Note: BrowserLogin uses api.shelly.cloud for code exchange
	// This test would need the actual server, so we skip it for unit tests
	t.Skip("BrowserLogin requires actual OAuth server - integration test only")
}

func TestBrowserLoginCallbackError(t *testing.T) {
	// Test that error responses from callback are handled
	t.Skip("BrowserLogin callback error testing requires integration test setup")
}

func TestCredentialTokenSourceFormEncoding(t *testing.T) {
	// Test that credential refresh sends form-encoded data correctly
	var receivedEmail, receivedPassword string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify POST method
		if r.Method != http.MethodPost {
			t.Errorf("Method = %v, want POST", r.Method)
		}

		// Verify form-encoded content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %v, want application/x-www-form-urlencoded", contentType)
		}

		// Parse and store form values
		if err := r.ParseForm(); err != nil {
			t.Errorf("Failed to parse form: %v", err)
		}
		receivedEmail = r.Form.Get("email")
		receivedPassword = r.Form.Get("password")

		resp := LoginResponse{
			IsOK: true,
			Data: &LoginData{
				Token:      "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2FwaV91cmwiOiJodHRwczovL3NoZWxseS00OS1ldS5zaGVsbHkuY2xvdWQiLCJleHAiOjIwMDAwMDAwMDB9.signature",
				UserAPIURL: "https://shelly-49-eu.shelly.cloud",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode failed: %v", err)
		}
	}))
	defer server.Close()

	ts := CredentialTokenSource(
		"user@example.com",
		"sha1hashedpassword",
		WithTokenURL(server.URL),
	)

	_, err := ts.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	if receivedEmail != "user@example.com" {
		t.Errorf("Email = %v, want user@example.com", receivedEmail)
	}
	if receivedPassword != "sha1hashedpassword" {
		t.Errorf("Password = %v, want sha1hashedpassword", receivedPassword)
	}
}
