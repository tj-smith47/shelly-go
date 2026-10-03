// Package authtest checks Gen2+ digest authentication the way a Shelly
// device does, written from Shelly's Authentication documentation
// (https://shelly-api-docs.shelly.cloud/gen2/General/Authentication) and
// independent of the internal/digest code it is used to test. Fake devices
// in tests use it to require a password.
package authtest

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Realm is the device id fake devices use as their realm.
const Realm = "shellyplus2pm-c4d8d5a1b2c3"

const (
	user      = "admin"
	algorithm = "SHA-256"
)

var hexNC = regexp.MustCompile(`^[0-9a-f]{8}$`)

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// HA1 is SHA256("admin:realm:password").
func HA1(realm, password string) string {
	return sha256Hex(user + ":" + realm + ":" + password)
}

// FrameChallenge returns the message of the 401 error a device answers an
// unauthenticated request frame with. nonce keeps its JSON type: a string on
// firmware 2.0.0 and later, a number before.
func FrameChallenge(realm string, nonce any) string {
	data, err := json.Marshal(map[string]any{
		"auth_type": "digest", "nonce": nonce, "realm": realm, "algorithm": algorithm,
	})
	if err != nil {
		return err.Error()
	}
	return string(data)
}

// FrameAuth checks auth, the auth object of a request frame, against the
// challenge (realm, nonce) and password. It returns the frame's nc.
func FrameAuth(auth json.RawMessage, realm string, nonce any, password string) (string, error) {
	var a map[string]json.RawMessage
	if err := json.Unmarshal(auth, &a); err != nil {
		return "", fmt.Errorf("auth object: %w", err)
	}
	str := func(key string) string {
		var s string
		if json.Unmarshal(a[key], &s) != nil {
			return ""
		}
		return s
	}
	wantNonce, err := json.Marshal(nonce)
	if err != nil {
		return "", err
	}
	if string(a["nonce"]) != string(wantNonce) {
		return "", fmt.Errorf("nonce %s, want %s (same JSON type)", a["nonce"], wantNonce)
	}
	cnonce := string(a["cnonce"])
	if cnonce == "" || strings.ContainsAny(cnonce, `".eE-`) {
		return "", fmt.Errorf("cnonce %s is not a positive integer", cnonce)
	}
	if str("username") != user || str("realm") != realm || str("algorithm") != algorithm {
		return "", fmt.Errorf("username/realm/algorithm %q/%q/%q", str("username"), str("realm"), str("algorithm"))
	}
	nc := str("nc")
	if !hexNC.MatchString(nc) {
		return "", fmt.Errorf("nc %q is not 8 hex digits", nc)
	}

	ha2 := sha256Hex("dummy_method:dummy_uri")
	want := sha256Hex(HA1(realm, password) + ":" + fmt.Sprint(nonce) + ":" + nc + ":" + cnonce + ":auth:" + ha2)
	if subtle.ConstantTimeCompare([]byte(str("response")), []byte(want)) != 1 {
		return nc, errors.New("response does not match")
	}
	return nc, nil
}

// HeaderChallenge is the WWW-Authenticate value a device sends with a 401.
func HeaderChallenge(realm, nonce string) string {
	return fmt.Sprintf(`Digest qop="auth", realm=%q, nonce=%q, algorithm=SHA-256`, realm, nonce)
}

// HeaderAuth checks the Authorization header of r against the challenge
// (realm, nonce) and password, with HA2 = SHA256("method:uri"). It returns
// the request's nc.
func HeaderAuth(r *http.Request, realm, nonce, password string) (string, error) {
	header, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Digest ")
	if !ok {
		return "", errors.New("no digest Authorization header")
	}
	p := map[string]string{}
	for part := range strings.SplitSeq(header, ",") {
		k, v, found := strings.Cut(strings.TrimSpace(part), "=")
		if found {
			p[k] = strings.Trim(v, `"`)
		}
	}
	if p["username"] != user || p["realm"] != realm || p["nonce"] != nonce ||
		p["uri"] != r.URL.RequestURI() || p["qop"] != "auth" || p["algorithm"] != algorithm {
		return "", fmt.Errorf("header fields %v", p)
	}
	if !hexNC.MatchString(p["nc"]) {
		return "", fmt.Errorf("nc %q is not 8 hex digits", p["nc"])
	}
	ha2 := sha256Hex(r.Method + ":" + p["uri"])
	want := sha256Hex(strings.Join([]string{HA1(realm, password), nonce, p["nc"], p["cnonce"], "auth", ha2}, ":"))
	if subtle.ConstantTimeCompare([]byte(p["response"]), []byte(want)) != 1 {
		return p["nc"], errors.New("response does not match")
	}
	return p["nc"], nil
}
