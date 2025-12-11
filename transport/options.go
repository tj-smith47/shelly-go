package transport

import (
	"crypto/tls"
	"net/http"
	"time"
)

// Option is a function that configures a transport.
type Option func(*options)

// options holds common configuration for all transports.
type options struct {
	client        *http.Client
	headers       map[string]string
	tlsConfig     *tls.Config
	mqttTopic     string
	mqttClientID  string
	username      string
	password      string
	retryDelay    time.Duration
	timeout       time.Duration
	retryBackoff  float64
	pingInterval  time.Duration
	pongTimeout   time.Duration
	maxRetries    int
	authType      authType
	coapPort      int
	reconnect     bool
	mqttQoS       byte
	coapMulticast bool
}

// authType represents the type of authentication.
type authType int

const (
	authTypeNone authType = iota
	authTypeBasic
	authTypeDigest
)

// default returns a default options struct.
func defaultOptions() *options {
	return &options{
		timeout:      30 * time.Second,
		headers:      make(map[string]string),
		authType:     authTypeNone,
		maxRetries:   3,
		retryDelay:   1 * time.Second,
		retryBackoff: 2.0,
		reconnect:    true,
		pingInterval: 30 * time.Second,
		pongTimeout:  10 * time.Second,
		mqttQoS:      0,
		coapPort:     5683,
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.timeout = timeout
	}
}

// WithClient sets a custom HTTP client.
func WithClient(client *http.Client) Option {
	return func(o *options) {
		o.client = client
	}
}

// WithHeader adds a custom HTTP header.
func WithHeader(key, value string) Option {
	return func(o *options) {
		o.headers[key] = value
	}
}

// WithAuth sets basic authentication credentials.
func WithAuth(username, password string) Option {
	return func(o *options) {
		o.authType = authTypeBasic
		o.username = username
		o.password = password
	}
}

// WithDigestAuth sets digest authentication credentials.
func WithDigestAuth(username, password string) Option {
	return func(o *options) {
		o.authType = authTypeDigest
		o.username = username
		o.password = password
	}
}

// WithTLS sets the TLS configuration.
func WithTLS(config *tls.Config) Option {
	return func(o *options) {
		o.tlsConfig = config
	}
}

// WithInsecureSkipVerify disables TLS certificate verification.
// Only use this for testing or with self-signed certificates.
func WithInsecureSkipVerify() Option {
	return func(o *options) {
		if o.tlsConfig == nil {
			o.tlsConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}
		o.tlsConfig.InsecureSkipVerify = true
	}
}

// WithRetry sets the retry configuration.
// maxRetries is the maximum number of retry attempts.
// initialDelay is the delay before the first retry.
func WithRetry(maxRetries int, initialDelay time.Duration) Option {
	return func(o *options) {
		o.maxRetries = maxRetries
		o.retryDelay = initialDelay
	}
}

// WithRetryBackoff sets the retry backoff multiplier.
// Default is 2.0 (exponential backoff).
func WithRetryBackoff(multiplier float64) Option {
	return func(o *options) {
		o.retryBackoff = multiplier
	}
}

// WithReconnect enables/disables automatic reconnection for WebSocket.
func WithReconnect(enable bool) Option {
	return func(o *options) {
		o.reconnect = enable
	}
}

// WithPingInterval sets the WebSocket ping interval.
func WithPingInterval(interval time.Duration) Option {
	return func(o *options) {
		o.pingInterval = interval
	}
}

// WithPongTimeout sets the WebSocket pong timeout.
func WithPongTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.pongTimeout = timeout
	}
}

// WithMQTTTopic sets the MQTT topic pattern.
func WithMQTTTopic(topic string) Option {
	return func(o *options) {
		o.mqttTopic = topic
	}
}

// WithMQTTClientID sets the MQTT client ID.
func WithMQTTClientID(clientID string) Option {
	return func(o *options) {
		o.mqttClientID = clientID
	}
}

// WithMQTTQoS sets the MQTT QoS level (0, 1, or 2).
func WithMQTTQoS(qos byte) Option {
	return func(o *options) {
		o.mqttQoS = qos
	}
}

// WithCoAPMulticast enables CoAP multicast mode.
func WithCoAPMulticast() Option {
	return func(o *options) {
		o.coapMulticast = true
	}
}

// WithCoAPPort sets the CoAP port (default: 5683).
func WithCoAPPort(port int) Option {
	return func(o *options) {
		o.coapPort = port
	}
}

// applyOptions applies option functions to an options struct.
func applyOptions(opts *options, options []Option) {
	for _, opt := range options {
		opt(opts)
	}
}
