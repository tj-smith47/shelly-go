package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/internal/digest"
	"github.com/tj-smith47/shelly-go/types"
)

// AlgorithmSHA256 is the digest algorithm Gen2+ devices use, and the only
// one this package computes.
const AlgorithmSHA256 = digest.Algorithm

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

// BasicAuth creates AuthData holding a username and password.
//
// Deprecated: a Shelly device takes no password in a request frame, and the
// password is never sent. Use transport.WithAuth for HTTP Basic auth, or
// transport.WithDigestAuth to have the HTTP and WebSocket transports answer
// the device's digest challenge.
func BasicAuth(username, password string) *AuthData {
	return &AuthData{
		Username: username,
		Password: password,
	}
}

// DigestAuth builds the auth object of one RPC request frame answering a
// device's digest challenge, as described in Shelly's Gen2 Authentication
// documentation:
//
//	response = SHA256(HA1:nonce:nc:cnonce:auth:HA2)
//	HA1      = SHA256(username:realm:password)
//	HA2      = SHA256("dummy_method:dummy_uri")
//
// username is "admin" when empty. realm is the device id, as the challenge
// names it. nonce is the challenge's nonce and is echoed with its JSON type:
// pass a string (firmware 2.0.0 and later), a number (earlier firmware) or the
// challenge's json.RawMessage. The nonce count is 1 and cnonce is random.
//
// transport.WithDigestAuth does all of this per frame, reusing the nonce; use
// DigestAuth only when building frames by hand.
func DigestAuth(username, password, realm string, nonce any) (*AuthData, error) {
	if username == "" {
		username = digest.User
	}
	return DigestAuthFromHA1(username, digest.HA1(username, realm, password), realm, nonce)
}

// DigestAuthFromHA1 is DigestAuth for a caller holding the HA1 (see
// CalculateHA1) instead of the password.
func DigestAuthFromHA1(username, ha1, realm string, nonce any) (*AuthData, error) {
	if username == "" {
		username = digest.User
	}
	if realm == "" {
		return nil, fmt.Errorf("%w: realm (the device id) is required", types.ErrInvalidParam)
	}
	raw, err := json.Marshal(nonce)
	if err != nil {
		return nil, fmt.Errorf("%w: nonce: %w", types.ErrInvalidParam, err)
	}
	frame, err := digest.NewFrame(username, ha1, realm, raw, 1)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", types.ErrInvalidParam, err)
	}
	return &AuthData{
		Username:  frame.Username,
		Realm:     frame.Realm,
		Nonce:     frame.Nonce,
		CNonce:    frame.CNonce,
		NC:        frame.NC,
		Algorithm: frame.Algorithm,
		Response:  frame.Response,
	}, nil
}

// CalculateHA1 returns SHA256("username:realm:password"), the ha1 parameter
// of Shelly.SetAuth, where username is "admin" and realm is the device id.
func CalculateHA1(username, password, realm string) string {
	return digest.HA1(username, realm, password)
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

	if auth.Response == "" {
		if auth.Password == "" {
			return fmt.Errorf("password is required for basic auth")
		}
		return nil
	}

	switch {
	case auth.Realm == "":
		return fmt.Errorf("realm is required for digest auth")
	case len(auth.Nonce) == 0:
		return fmt.Errorf("nonce is required for digest auth")
	case auth.CNonce == 0:
		return fmt.Errorf("cnonce is required for digest auth")
	case auth.NC == "":
		return fmt.Errorf("nc is required for digest auth")
	case auth.Algorithm != AlgorithmSHA256:
		return fmt.Errorf("algorithm must be %s for digest auth", AlgorithmSHA256)
	}
	return nil
}
