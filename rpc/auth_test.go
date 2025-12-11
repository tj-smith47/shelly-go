package rpc

import (
	"crypto/md5" //nolint:gosec // G501: MD5 is required by HTTP Digest Auth RFC 7616
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestAuthMethod_String(t *testing.T) {
	tests := []struct {
		want   string
		method AuthMethod
	}{
		{method: AuthMethodNone, want: "none"},
		{method: AuthMethodBasic, want: "basic"},
		{method: AuthMethodDigest, want: "digest"},
		{method: AuthMethodRPC, want: "rpc"},
		{method: AuthMethod(999), want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.method.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasicAuth(t *testing.T) {
	auth := BasicAuth("admin", "password")

	if auth == nil {
		t.Fatal("BasicAuth() returned nil")
	}

	if auth.Username != "admin" {
		t.Errorf("Username = %v, want admin", auth.Username)
	}

	if auth.Password != "password" {
		t.Errorf("Password = %v, want password", auth.Password)
	}
}

func TestDigestAuth(t *testing.T) {
	auth, err := DigestAuth(
		"admin",
		"password",
		"shelly",
		"abc123",
		"POST",
		"/rpc",
		"MD5",
	)

	if err != nil {
		t.Fatalf("DigestAuth() error = %v", err)
	}

	if auth == nil {
		t.Fatal("DigestAuth() returned nil")
	}

	if auth.Username != "admin" {
		t.Errorf("Username = %v, want admin", auth.Username)
	}

	if auth.Realm != "shelly" {
		t.Errorf("Realm = %v, want shelly", auth.Realm)
	}

	if auth.Nonce != "abc123" {
		t.Errorf("Nonce = %v, want abc123", auth.Nonce)
	}

	if auth.CNonce == "" {
		t.Error("CNonce should not be empty")
	}

	if auth.NC != 1 {
		t.Errorf("NC = %v, want 1", auth.NC)
	}

	if auth.Algorithm != "MD5" {
		t.Errorf("Algorithm = %v, want MD5", auth.Algorithm)
	}

	if auth.Response == "" {
		t.Error("Response should not be empty")
	}
}

func TestDigestAuth_SHA256(t *testing.T) {
	auth, err := DigestAuth(
		"admin",
		"password",
		"shelly",
		"abc123",
		"POST",
		"/rpc",
		"SHA-256",
	)

	if err != nil {
		t.Fatalf("DigestAuth() error = %v", err)
	}

	if auth.Algorithm != "SHA-256" {
		t.Errorf("Algorithm = %v, want SHA-256", auth.Algorithm)
	}

	// SHA-256 response should be 64 characters (256 bits in hex)
	if len(auth.Response) != 64 {
		t.Errorf("Response length = %v, want 64", len(auth.Response))
	}
}

func TestCalculateHA1(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		password  string
		realm     string
		algorithm string
		wantLen   int
	}{
		{
			name:      "MD5",
			username:  "admin",
			password:  "password",
			realm:     "shelly",
			algorithm: "MD5",
			wantLen:   32, // MD5 is 128 bits = 32 hex chars
		},
		{
			name:      "SHA-256",
			username:  "admin",
			password:  "password",
			realm:     "shelly",
			algorithm: "SHA-256",
			wantLen:   64, // SHA-256 is 256 bits = 64 hex chars
		},
		{
			name:      "empty algorithm defaults to MD5",
			username:  "admin",
			password:  "password",
			realm:     "shelly",
			algorithm: "",
			wantLen:   32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ha1 := CalculateHA1(tt.username, tt.password, tt.realm, tt.algorithm)

			if len(ha1) != tt.wantLen {
				t.Errorf("HA1 length = %v, want %v", len(ha1), tt.wantLen)
			}

			// Verify it's valid hex
			if _, err := hex.DecodeString(ha1); err != nil {
				t.Errorf("HA1 is not valid hex: %v", err)
			}
		})
	}
}

func TestCalculateHA1_Deterministic(t *testing.T) {
	// Same inputs should always produce the same HA1
	ha1_1 := CalculateHA1("admin", "password", "shelly", "MD5")
	ha1_2 := CalculateHA1("admin", "password", "shelly", "MD5")

	if ha1_1 != ha1_2 {
		t.Error("HA1 calculation should be deterministic")
	}

	// Different inputs should produce different HA1
	ha1_3 := CalculateHA1("admin", "different", "shelly", "MD5")
	if ha1_1 == ha1_3 {
		t.Error("Different passwords should produce different HA1")
	}
}

func TestDigestAuthFromHA1(t *testing.T) {
	// Pre-calculate HA1
	ha1 := CalculateHA1("admin", "password", "shelly", "MD5")

	auth, err := DigestAuthFromHA1(
		"admin",
		ha1,
		"shelly",
		"abc123",
		"POST",
		"/rpc",
		"MD5",
	)

	if err != nil {
		t.Fatalf("DigestAuthFromHA1() error = %v", err)
	}

	if auth == nil {
		t.Fatal("DigestAuthFromHA1() returned nil")
	}

	if auth.Username != "admin" {
		t.Errorf("Username = %v, want admin", auth.Username)
	}

	if auth.Response == "" {
		t.Error("Response should not be empty")
	}

	// Verify the response is the same as if we used DigestAuth directly
	auth2, err := DigestAuth(
		"admin",
		"password",
		"shelly",
		"abc123",
		"POST",
		"/rpc",
		"MD5",
	)
	if err != nil {
		t.Fatalf("DigestAuth() error = %v", err)
	}

	// The responses won't be exactly the same because cnonce is random,
	// but both should be valid 32-character hex strings for MD5
	if len(auth.Response) != len(auth2.Response) {
		t.Errorf("Response lengths differ: %v vs %v", len(auth.Response), len(auth2.Response))
	}

	if len(auth.Response) != 32 {
		t.Errorf("Response length = %v, want 32", len(auth.Response))
	}
}

func TestValidateAuthData(t *testing.T) {
	tests := []struct {
		auth    *AuthData
		name    string
		wantErr bool
	}{
		{
			name:    "nil auth data",
			auth:    nil,
			wantErr: true,
		},
		{
			name: "basic auth valid",
			auth: &AuthData{
				Username: "admin",
				Password: "password",
			},
			wantErr: false,
		},
		{
			name: "basic auth missing username",
			auth: &AuthData{
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "basic auth missing password",
			auth: &AuthData{
				Username: "admin",
			},
			wantErr: true,
		},
		{
			name: "digest auth valid",
			auth: &AuthData{
				Username:  "admin",
				Realm:     "shelly",
				Nonce:     "abc123",
				CNonce:    "def456",
				NC:        1,
				Algorithm: "MD5",
				Response:  "hash123",
			},
			wantErr: false,
		},
		{
			name: "digest auth missing realm",
			auth: &AuthData{
				Username: "admin",
				Nonce:    "abc123",
				CNonce:   "def456",
				NC:       1,
				Response: "hash123",
			},
			wantErr: true,
		},
		{
			name: "digest auth missing nonce",
			auth: &AuthData{
				Username: "admin",
				Realm:    "shelly",
				CNonce:   "def456",
				NC:       1,
				Response: "hash123",
			},
			wantErr: true,
		},
		{
			name: "digest auth missing cnonce",
			auth: &AuthData{
				Username: "admin",
				Realm:    "shelly",
				Nonce:    "abc123",
				NC:       1,
				Response: "hash123",
			},
			wantErr: true,
		},
		{
			name: "digest auth invalid nc",
			auth: &AuthData{
				Username: "admin",
				Realm:    "shelly",
				Nonce:    "abc123",
				CNonce:   "def456",
				NC:       0,
				Response: "hash123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAuthData(tt.auth)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAuthData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce1, err := generateNonce()
	if err != nil {
		t.Fatalf("generateNonce() error = %v", err)
	}

	// Nonce should be 32 characters (16 bytes in hex)
	if len(nonce1) != 32 {
		t.Errorf("nonce length = %v, want 32", len(nonce1))
	}

	// Verify it's valid hex
	if _, err := hex.DecodeString(nonce1); err != nil {
		t.Errorf("nonce is not valid hex: %v", err)
	}

	// Generate another nonce and verify it's different
	nonce2, err := generateNonce()
	if err != nil {
		t.Fatalf("generateNonce() error = %v", err)
	}

	if nonce1 == nonce2 {
		t.Error("nonces should be unique")
	}
}

func TestCalculateHash(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		algorithm string
		want      string
	}{
		{
			name:      "MD5",
			data:      "test",
			algorithm: "MD5",
			want:      calculateMD5("test"),
		},
		{
			name:      "SHA-256",
			data:      "test",
			algorithm: "SHA-256",
			want:      calculateSHA256("test"),
		},
		{
			name:      "empty algorithm defaults to MD5",
			data:      "test",
			algorithm: "",
			want:      calculateMD5("test"),
		},
		{
			name:      "unknown algorithm defaults to MD5",
			data:      "test",
			algorithm: "UNKNOWN",
			want:      calculateMD5("test"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateHash(tt.data, tt.algorithm)

			if got != tt.want {
				t.Errorf("calculateHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateDigestResponse(t *testing.T) {
	// Test with known values
	response := calculateDigestResponse(
		"admin",
		"password",
		"shelly",
		"abc123",
		"def456",
		"POST",
		"/rpc",
		"MD5",
	)

	// Response should be 32 characters for MD5
	if len(response) != 32 {
		t.Errorf("response length = %v, want 32", len(response))
	}

	// Verify it's valid hex
	if _, err := hex.DecodeString(response); err != nil {
		t.Errorf("response is not valid hex: %v", err)
	}

	// Same inputs should produce same response
	response2 := calculateDigestResponse(
		"admin",
		"password",
		"shelly",
		"abc123",
		"def456",
		"POST",
		"/rpc",
		"MD5",
	)

	if response != response2 {
		t.Error("digest response calculation should be deterministic")
	}
}

func TestCalculateDigestResponse_SHA256(t *testing.T) {
	response := calculateDigestResponse(
		"admin",
		"password",
		"shelly",
		"abc123",
		"def456",
		"POST",
		"/rpc",
		"SHA-256",
	)

	// Response should be 64 characters for SHA-256
	if len(response) != 64 {
		t.Errorf("response length = %v, want 64", len(response))
	}

	// Verify it's valid hex
	if _, err := hex.DecodeString(response); err != nil {
		t.Errorf("response is not valid hex: %v", err)
	}
}

// Helper functions for testing
func calculateMD5(data string) string {
	//nolint:gosec // G401: MD5 is required by HTTP Digest Auth RFC 7616
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func calculateSHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func TestAuthDataJSONMarshaling(t *testing.T) {
	// This test verifies that AuthData can be marshaled/unmarshaled correctly
	// when included in a Request
	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "Test",
		Auth: &AuthData{
			Username: "admin",
			Password: "password",
		},
	}

	data, err := req.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	// Verify the JSON contains the auth field
	jsonStr := string(data)
	if !strings.Contains(jsonStr, "auth") {
		t.Error("marshaled JSON should contain auth field")
	}

	if !strings.Contains(jsonStr, "admin") {
		t.Error("marshaled JSON should contain username")
	}
}

func TestDigestAuth_DifferentMethods(t *testing.T) {
	// Different HTTP methods should produce different responses
	auth1, err := DigestAuth("admin", "password", "shelly", "abc123", "GET", "/rpc", "MD5")
	if err != nil {
		t.Fatalf("DigestAuth() error = %v", err)
	}

	auth2, err := DigestAuth("admin", "password", "shelly", "abc123", "POST", "/rpc", "MD5")
	if err != nil {
		t.Fatalf("DigestAuth() error = %v", err)
	}

	// Responses will differ due to random cnonce, but we can verify both are valid
	if len(auth1.Response) != 32 {
		t.Errorf("auth1 response length = %v, want 32", len(auth1.Response))
	}

	if len(auth2.Response) != 32 {
		t.Errorf("auth2 response length = %v, want 32", len(auth2.Response))
	}
}

func TestDigestAuth_DifferentURIs(t *testing.T) {
	// Different URIs should produce different responses
	auth1, err := DigestAuth("admin", "password", "shelly", "abc123", "POST", "/rpc", "MD5")
	if err != nil {
		t.Fatalf("DigestAuth() error = %v", err)
	}

	auth2, err := DigestAuth("admin", "password", "shelly", "abc123", "POST", "/settings", "MD5")
	if err != nil {
		t.Fatalf("DigestAuth() error = %v", err)
	}

	// Both should be valid
	if len(auth1.Response) != 32 {
		t.Errorf("auth1 response length = %v, want 32", len(auth1.Response))
	}

	if len(auth2.Response) != 32 {
		t.Errorf("auth2 response length = %v, want 32", len(auth2.Response))
	}
}
