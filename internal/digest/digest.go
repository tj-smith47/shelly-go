// Package digest computes the SHA-256 digest authentication Gen2+ Shelly
// devices use, for both the HTTP Authorization header and the auth object of
// an RPC request frame (WebSocket). It is the single implementation the rpc
// and transport packages share.
//
// See https://shelly-api-docs.shelly.cloud/gen2/General/Authentication.
package digest

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

const (
	// Algorithm is the only digest algorithm Gen2+ devices use.
	Algorithm = "SHA-256"

	// User is the only user a Gen2+ device has.
	User = "admin"

	qop = "auth"

	// frameHA2Input stands in for "method:uri" in a request frame, which has
	// neither.
	frameHA2Input = "dummy_method:dummy_uri"
)

// ErrChallenge is wrapped by every error about a challenge the device sent
// that cannot be answered.
var ErrChallenge = errors.New("unusable digest challenge")

// Hash returns the hex SHA-256 of s.
func Hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// HA1 returns SHA256("user:realm:password"), the value a device stores
// through Shelly.SetAuth.
func HA1(user, realm, password string) string {
	return Hash(user + ":" + realm + ":" + password)
}

// NC formats a nonce count the way the protocol sends it: 8 hex digits.
func NC(nc uint32) string {
	return fmt.Sprintf("%08x", nc)
}

// Response returns SHA256("ha1:nonce:nc:cnonce:auth:ha2").
func Response(ha1, nonce string, nc uint32, cnonce, ha2 string) string {
	return Hash(strings.Join([]string{ha1, nonce, NC(nc), cnonce, qop, ha2}, ":"))
}

// CNonce returns a random, non-zero client nonce.
func CNonce() (uint32, error) {
	var b [4]byte
	for {
		if _, err := rand.Read(b[:]); err != nil {
			return 0, fmt.Errorf("generate client nonce: %w", err)
		}
		if n := binary.BigEndian.Uint32(b[:]); n != 0 {
			return n, nil
		}
	}
}

// NonceText returns the text a nonce contributes to the response hash: the
// string itself for a JSON string, the number's digits for a JSON number.
// Firmware before 2.0.0 sends a number, later firmware a base64 string.
func NonceText(nonce json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(nonce, &s); err == nil {
		if s == "" {
			return "", fmt.Errorf("%w: empty nonce", ErrChallenge)
		}
		return s, nil
	}
	var n json.Number
	if err := json.Unmarshal(nonce, &n); err != nil {
		return "", fmt.Errorf("%w: nonce %s is neither a string nor a number", ErrChallenge, nonce)
	}
	return n.String(), nil
}

// Frame is the auth object of an RPC request frame. Nonce echoes the
// challenge's nonce with its JSON type.
type Frame struct {
	Realm     string          `json:"realm"`
	Username  string          `json:"username"`
	NC        string          `json:"nc"`
	Response  string          `json:"response"`
	Algorithm string          `json:"algorithm"`
	Nonce     json.RawMessage `json:"nonce"`
	CNonce    uint32          `json:"cnonce"`
}

// NewFrame builds the auth object of a request frame for user, ha1 (see HA1),
// realm (the device id), the challenge's nonce and nonce count nc.
func NewFrame(user, ha1, realm string, nonce json.RawMessage, nc uint32) (*Frame, error) {
	text, err := NonceText(nonce)
	if err != nil {
		return nil, err
	}
	cnonce, err := CNonce()
	if err != nil {
		return nil, err
	}
	return &Frame{
		Realm:     realm,
		Username:  user,
		Nonce:     nonce,
		NC:        NC(nc),
		CNonce:    cnonce,
		Response:  Response(ha1, text, nc, fmt.Sprint(cnonce), Hash(frameHA2Input)),
		Algorithm: Algorithm,
	}, nil
}

// Challenge is a digest challenge from a device: the JSON message of a 401
// error answering a request frame, or an HTTP WWW-Authenticate header.
type Challenge struct {
	AuthType  string          `json:"auth_type"`
	Realm     string          `json:"realm"`
	Algorithm string          `json:"algorithm"`
	Nonce     json.RawMessage `json:"nonce"`
}

// ParseFrameChallenge parses the message of a 401 error a device sent in
// reply to a request frame.
func ParseFrameChallenge(message string) (*Challenge, error) {
	var ch Challenge
	if err := json.Unmarshal([]byte(message), &ch); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrChallenge, message)
	}
	if ch.AuthType != "digest" {
		return nil, fmt.Errorf("%w: auth_type %q", ErrChallenge, ch.AuthType)
	}
	return ch.check()
}

// ParseHeader parses a WWW-Authenticate header value.
func ParseHeader(header string) (*Challenge, error) {
	scheme, rest, _ := strings.Cut(strings.TrimSpace(header), " ")
	if !strings.EqualFold(scheme, "Digest") {
		return nil, fmt.Errorf("%w: no digest challenge in %q", ErrChallenge, header)
	}
	params := map[string]string{}
	for part := range strings.SplitSeq(rest, ",") {
		key, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if found {
			params[strings.ToLower(strings.TrimSpace(key))] = strings.Trim(strings.TrimSpace(value), `"`)
		}
	}
	ch := Challenge{AuthType: "digest", Realm: params["realm"], Algorithm: params["algorithm"]}
	if params["nonce"] != "" {
		nonce, err := json.Marshal(params["nonce"])
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrChallenge, err)
		}
		ch.Nonce = nonce
	}
	return ch.check()
}

func (ch *Challenge) check() (*Challenge, error) {
	if ch.Algorithm != Algorithm {
		return nil, fmt.Errorf("%w: algorithm %q, want %s", ErrChallenge, ch.Algorithm, Algorithm)
	}
	if ch.Realm == "" {
		return nil, fmt.Errorf("%w: no realm", ErrChallenge)
	}
	if _, err := NonceText(ch.Nonce); err != nil {
		return nil, err
	}
	return ch, nil
}

// Session holds a user's password and the nonce the device last issued, so
// later requests authenticate without a fresh challenge. A device accepts
// one nonce for many requests as long as the nonce count rises. Session is
// safe for concurrent use.
type Session struct {
	ch       *Challenge
	user     string
	password string
	ha1      string
	mu       sync.Mutex
	nc       uint32
}

// NewSession returns a session for user (User when empty) and password.
func NewSession(user, password string) *Session {
	if user == "" {
		user = User
	}
	return &Session{user: user, password: password}
}

// Accept makes ch the challenge later requests answer, restarting the nonce
// count.
func (s *Session) Accept(ch *Challenge) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ch == nil || s.ch.Realm != ch.Realm {
		s.ha1 = HA1(s.user, ch.Realm, s.password)
	}
	s.ch = ch
	s.nc = 0
}

// Ready reports whether a challenge has been accepted, so Frame and Header
// can answer it.
func (s *Session) Ready() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ch != nil
}

// errNotReady is returned by Frame and Header before the first Accept.
var errNotReady = errors.New("digest: no challenge accepted yet")

// next returns the held challenge, its HA1 and the next nonce count.
func (s *Session) next() (ch *Challenge, ha1 string, nc uint32, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ch == nil {
		return nil, "", 0, errNotReady
	}
	s.nc++
	return s.ch, s.ha1, s.nc, nil
}

// Frame returns the auth object for the next request frame.
func (s *Session) Frame() (*Frame, error) {
	ch, ha1, nc, err := s.next()
	if err != nil {
		return nil, err
	}
	return NewFrame(s.user, ha1, ch.Realm, ch.Nonce, nc)
}

// Header returns the Authorization header value for the next HTTP request
// with method and uri (the request URI).
func (s *Session) Header(method, uri string) (string, error) {
	ch, ha1, nc, err := s.next()
	if err != nil {
		return "", err
	}
	nonce, err := NonceText(ch.Nonce)
	if err != nil {
		return "", err
	}
	cnonce, err := CNonce()
	if err != nil {
		return "", err
	}
	cn := fmt.Sprint(cnonce)
	response := Response(ha1, nonce, nc, cn, Hash(method+":"+uri))
	return fmt.Sprintf(
		`Digest username=%q, realm=%q, nonce=%q, uri=%q, algorithm=%s, qop=%s, nc=%s, cnonce=%q, response=%q`,
		s.user, ch.Realm, nonce, uri, Algorithm, qop, NC(nc), cn, response,
	), nil
}
