package rpc

import (
	"crypto/md5" //nolint:gosec // G501: MD5 is required by HTTP Digest Authentication (RFC 2617)
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// AuthMethod represents the authentication method to use for RPC requests.
type AuthMethod int

const (
	// AuthMethodNone indicates no authentication.
	AuthMethodNone AuthMethod = iota

	// AuthMethodBasic indicates HTTP Basic authentication.
	// This is handled by the transport layer.
	AuthMethodBasic

	// AuthMethodDigest indicates HTTP Digest authentication.
	// This is handled by the transport layer.
	AuthMethodDigest

	// AuthMethodRPC indicates RPC-level authentication using the "auth" field.
	AuthMethodRPC
)

// String returns the string representation of the auth method.
func (am AuthMethod) String() string {
	switch am {
	case AuthMethodNone:
		return "none"
	case AuthMethodBasic:
		return "basic"
	case AuthMethodDigest:
		return "digest"
	case AuthMethodRPC:
		return "rpc"
	default:
		return "unknown"
	}
}

// BasicAuth creates AuthData for basic authentication.
//
// Basic authentication sends the username and password in plain text
// (base64 encoded). This should only be used over HTTPS.
func BasicAuth(username, password string) *AuthData {
	return &AuthData{
		Username: username,
		Password: password,
	}
}

// DigestAuth creates AuthData for digest authentication.
//
// Digest authentication uses cryptographic hashing to avoid sending
// passwords in plain text. This is more secure than basic auth over
// unencrypted connections.
//
// Parameters:
//   - username: The username
//   - password: The password
//   - realm: The authentication realm (from server challenge)
//   - nonce: The server nonce (from server challenge)
//   - method: The HTTP method (e.g., "POST")
//   - uri: The request URI (e.g., "/rpc")
//   - algorithm: The hash algorithm ("MD5" or "SHA-256")
func DigestAuth(
	username, password, realm, nonce, method, uri, algorithm string,
) (*AuthData, error) {
	// Generate client nonce
	cnonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate cnonce: %w", err)
	}

	// Calculate response hash
	response := calculateDigestResponse(
		username, password, realm, nonce, cnonce, method, uri, algorithm,
	)

	return &AuthData{
		Username:  username,
		Realm:     realm,
		Nonce:     nonce,
		CNonce:    cnonce,
		NC:        1,
		Algorithm: algorithm,
		Response:  response,
	}, nil
}

// calculateDigestResponse calculates the digest authentication response hash.
func calculateDigestResponse(
	username, password, realm, nonce, cnonce, method, uri, algorithm string,
) string {
	// Calculate HA1 = hash(username:realm:password)
	ha1 := calculateHash(fmt.Sprintf("%s:%s:%s", username, realm, password), algorithm)

	// Calculate HA2 = hash(method:uri)
	ha2 := calculateHash(fmt.Sprintf("%s:%s", method, uri), algorithm)

	// Calculate response = hash(HA1:nonce:nc:cnonce:qop:HA2)
	// Note: qop is assumed to be "auth" for Shelly devices, nc is always 1
	const nc = 1
	response := calculateHash(
		fmt.Sprintf("%s:%s:%08x:%s:auth:%s", ha1, nonce, nc, cnonce, ha2),
		algorithm,
	)

	return response
}

// calculateHash calculates a hash using the specified algorithm.
func calculateHash(data, algorithm string) string {
	switch algorithm {
	case "SHA-256":
		hash := sha256.Sum256([]byte(data))
		return hex.EncodeToString(hash[:])
	case "MD5", "":
		// MD5 is the default if no algorithm is specified
		hash := md5.Sum([]byte(data)) //nolint:gosec // G401: MD5 required by HTTP Digest Auth
		return hex.EncodeToString(hash[:])
	default:
		// Fallback to MD5 for unknown algorithms
		hash := md5.Sum([]byte(data)) //nolint:gosec // G401: MD5 required by HTTP Digest Auth
		return hex.EncodeToString(hash[:])
	}
}

// generateNonce generates a random nonce for digest authentication.
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CalculateHA1 calculates the HA1 hash for digest authentication.
//
// HA1 = hash(username:realm:password)
//
// This can be pre-calculated and stored instead of storing the plaintext
// password for improved security.
func CalculateHA1(username, password, realm, algorithm string) string {
	return calculateHash(fmt.Sprintf("%s:%s:%s", username, realm, password), algorithm)
}

// DigestAuthFromHA1 creates AuthData for digest authentication using a
// pre-calculated HA1 hash instead of a plaintext password.
//
// This is more secure as it avoids storing plaintext passwords.
func DigestAuthFromHA1(
	username, ha1, realm, nonce, method, uri, algorithm string,
) (*AuthData, error) {
	// Generate client nonce
	cnonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate cnonce: %w", err)
	}

	// Calculate HA2 = hash(method:uri)
	ha2 := calculateHash(fmt.Sprintf("%s:%s", method, uri), algorithm)

	// Calculate response = hash(HA1:nonce:nc:cnonce:qop:HA2)
	nc := 1
	response := calculateHash(
		fmt.Sprintf("%s:%s:%08x:%s:auth:%s", ha1, nonce, nc, cnonce, ha2),
		algorithm,
	)

	return &AuthData{
		Username:  username,
		Realm:     realm,
		Nonce:     nonce,
		CNonce:    cnonce,
		NC:        nc,
		Algorithm: algorithm,
		Response:  response,
	}, nil
}

// ValidateAuthData validates that the AuthData contains the required fields
// for the authentication method.
func ValidateAuthData(auth *AuthData) error {
	if auth == nil {
		return fmt.Errorf("auth data is nil")
	}

	if auth.Username == "" {
		return fmt.Errorf("username is required")
	}

	// Check if this is digest auth or basic auth
	//nolint:nestif // Digest auth validation requires checking multiple required fields
	if auth.Response != "" {
		// Digest auth requires additional fields
		if auth.Realm == "" {
			return fmt.Errorf("realm is required for digest auth")
		}
		if auth.Nonce == "" {
			return fmt.Errorf("nonce is required for digest auth")
		}
		if auth.CNonce == "" {
			return fmt.Errorf("cnonce is required for digest auth")
		}
		if auth.NC <= 0 {
			return fmt.Errorf("nc must be positive for digest auth")
		}
	} else if auth.Password == "" {
		// Basic auth requires password
		return fmt.Errorf("password is required for basic auth")
	}

	return nil
}
