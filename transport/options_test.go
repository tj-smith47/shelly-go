package transport

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()

	if opts.timeout != 30*time.Second {
		t.Errorf("default timeout = %v, want 30s", opts.timeout)
	}

	if opts.maxRetries != 3 {
		t.Errorf("default maxRetries = %v, want 3", opts.maxRetries)
	}

	if opts.retryBackoff != 2.0 {
		t.Errorf("default retryBackoff = %v, want 2.0", opts.retryBackoff)
	}

	if !opts.reconnect {
		t.Error("default reconnect = false, want true")
	}
}

func TestWithTimeout(t *testing.T) {
	opts := defaultOptions()
	timeout := 60 * time.Second

	WithTimeout(timeout)(opts)

	if opts.timeout != timeout {
		t.Errorf("timeout = %v, want %v", opts.timeout, timeout)
	}
}

func TestWithClient(t *testing.T) {
	opts := defaultOptions()
	client := &http.Client{Timeout: 45 * time.Second}

	WithClient(client)(opts)

	if opts.client != client {
		t.Error("client not set correctly")
	}
}

func TestWithHeader(t *testing.T) {
	opts := defaultOptions()

	WithHeader("X-Test", "value")(opts)

	if opts.headers["X-Test"] != "value" {
		t.Errorf("header = %v, want value", opts.headers["X-Test"])
	}
}

func TestWithAuth(t *testing.T) {
	opts := defaultOptions()

	WithAuth("user", "pass")(opts)

	if opts.authType != authTypeBasic {
		t.Errorf("authType = %v, want %v", opts.authType, authTypeBasic)
	}

	if opts.username != "user" {
		t.Errorf("username = %v, want user", opts.username)
	}

	if opts.password != "pass" {
		t.Errorf("password = %v, want pass", opts.password)
	}
}

func TestWithDigestAuth(t *testing.T) {
	opts := defaultOptions()

	WithDigestAuth("admin", "secret")(opts)

	if opts.authType != authTypeDigest {
		t.Errorf("authType = %v, want %v", opts.authType, authTypeDigest)
	}

	if opts.username != "admin" {
		t.Errorf("username = %v, want admin", opts.username)
	}
}

func TestWithTLS(t *testing.T) {
	opts := defaultOptions()
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}

	WithTLS(tlsConfig)(opts)

	if opts.tlsConfig != tlsConfig {
		t.Error("tlsConfig not set correctly")
	}
}

func TestWithInsecureSkipVerify(t *testing.T) {
	opts := defaultOptions()

	WithInsecureSkipVerify()(opts)

	if opts.tlsConfig == nil {
		t.Fatal("tlsConfig is nil")
	}

	if !opts.tlsConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify = false, want true")
	}
}

func TestWithInsecureSkipVerify_ExistingTLS(t *testing.T) {
	opts := defaultOptions()
	opts.tlsConfig = &tls.Config{MinVersion: tls.VersionTLS13}

	WithInsecureSkipVerify()(opts)

	if !opts.tlsConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify = false, want true")
	}

	// Should preserve existing config
	if opts.tlsConfig.MinVersion != tls.VersionTLS13 {
		t.Error("MinVersion was changed")
	}
}

func TestWithRetry(t *testing.T) {
	opts := defaultOptions()

	WithRetry(5, 2*time.Second)(opts)

	if opts.maxRetries != 5 {
		t.Errorf("maxRetries = %v, want 5", opts.maxRetries)
	}

	if opts.retryDelay != 2*time.Second {
		t.Errorf("retryDelay = %v, want 2s", opts.retryDelay)
	}
}

func TestWithRetryBackoff(t *testing.T) {
	opts := defaultOptions()

	WithRetryBackoff(1.5)(opts)

	if opts.retryBackoff != 1.5 {
		t.Errorf("retryBackoff = %v, want 1.5", opts.retryBackoff)
	}
}

func TestWithReconnect(t *testing.T) {
	opts := defaultOptions()

	WithReconnect(false)(opts)

	if opts.reconnect {
		t.Error("reconnect = true, want false")
	}
}

func TestWithPingInterval(t *testing.T) {
	opts := defaultOptions()

	WithPingInterval(60 * time.Second)(opts)

	if opts.pingInterval != 60*time.Second {
		t.Errorf("pingInterval = %v, want 60s", opts.pingInterval)
	}
}

func TestWithPongTimeout(t *testing.T) {
	opts := defaultOptions()

	WithPongTimeout(5 * time.Second)(opts)

	if opts.pongTimeout != 5*time.Second {
		t.Errorf("pongTimeout = %v, want 5s", opts.pongTimeout)
	}
}

func TestWithMQTTTopic(t *testing.T) {
	opts := defaultOptions()

	WithMQTTTopic("shellies/test")(opts)

	if opts.mqttTopic != "shellies/test" {
		t.Errorf("mqttTopic = %v, want shellies/test", opts.mqttTopic)
	}
}

func TestWithMQTTClientID(t *testing.T) {
	opts := defaultOptions()

	WithMQTTClientID("test-client")(opts)

	if opts.mqttClientID != "test-client" {
		t.Errorf("mqttClientID = %v, want test-client", opts.mqttClientID)
	}
}

func TestWithMQTTQoS(t *testing.T) {
	opts := defaultOptions()

	WithMQTTQoS(2)(opts)

	if opts.mqttQoS != 2 {
		t.Errorf("mqttQoS = %v, want 2", opts.mqttQoS)
	}
}

func TestWithCoAPMulticast(t *testing.T) {
	opts := defaultOptions()

	WithCoAPMulticast()(opts)

	if !opts.coapMulticast {
		t.Error("coapMulticast = false, want true")
	}
}

func TestWithCoAPPort(t *testing.T) {
	opts := defaultOptions()

	WithCoAPPort(5684)(opts)

	if opts.coapPort != 5684 {
		t.Errorf("coapPort = %v, want 5684", opts.coapPort)
	}
}

func TestApplyOptions(t *testing.T) {
	opts := defaultOptions()

	options := []Option{
		WithTimeout(45 * time.Second),
		WithRetry(10, 500*time.Millisecond),
		WithAuth("user", "pass"),
	}

	applyOptions(opts, options)

	if opts.timeout != 45*time.Second {
		t.Errorf("timeout = %v, want 45s", opts.timeout)
	}

	if opts.maxRetries != 10 {
		t.Errorf("maxRetries = %v, want 10", opts.maxRetries)
	}

	if opts.username != "user" {
		t.Errorf("username = %v, want user", opts.username)
	}
}

func TestMultipleOptions(t *testing.T) {
	opts := defaultOptions()

	options := []Option{
		WithTimeout(30 * time.Second),
		WithAuth("admin", "secret"),
		WithRetry(5, 2*time.Second),
		WithRetryBackoff(1.5),
		WithReconnect(false),
		WithMQTTTopic("test/topic"),
		WithMQTTQoS(1),
	}

	applyOptions(opts, options)

	// Verify all options were applied
	if opts.timeout != 30*time.Second {
		t.Error("timeout not applied")
	}
	if opts.username != "admin" {
		t.Error("username not applied")
	}
	if opts.maxRetries != 5 {
		t.Error("maxRetries not applied")
	}
	if opts.retryBackoff != 1.5 {
		t.Error("retryBackoff not applied")
	}
	if opts.reconnect {
		t.Error("reconnect not applied")
	}
	if opts.mqttTopic != "test/topic" {
		t.Error("mqttTopic not applied")
	}
	if opts.mqttQoS != 1 {
		t.Error("mqttQoS not applied")
	}
}
