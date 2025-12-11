package integrator

import (
	"encoding/json"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// AuthRequest represents a request to obtain a JWT token.
type AuthRequest struct {
	// IntegratorTag is the integrator identifier.
	IntegratorTag string `json:"itg"`

	// Token is the integrator secret token.
	Token string `json:"token"`
}

// AuthResponse contains the JWT token response.
type AuthResponse struct {
	Data *AuthData `json:"data,omitempty"`
	types.RawFields
	Errors json.RawMessage `json:"errors,omitempty"`
	IsOK   bool            `json:"isok"`
}

// AuthData contains the JWT token and expiration info.
type AuthData struct {
	types.RawFields
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

// ExpiresTime returns the expiration time as time.Time.
func (a *AuthData) ExpiresTime() time.Time {
	return time.Unix(a.ExpiresAt, 0)
}

// IsExpired returns true if the token has expired.
func (a *AuthData) IsExpired() bool {
	return time.Now().Unix() > a.ExpiresAt
}

// WSMessage represents a WebSocket message from the Shelly cloud.
type WSMessage struct {
	Online *int `json:"online,omitempty"`
	types.RawFields
	Event        string          `json:"event"`
	DeviceID     string          `json:"device_id,omitempty"`
	Device       string          `json:"device,omitempty"`
	AccessGroups string          `json:"accessGroups,omitempty"`
	Status       json.RawMessage `json:"status,omitempty"`
	Settings     json.RawMessage `json:"settings,omitempty"`
	Timestamp    int64           `json:"ts,omitempty"`
}

// GetDeviceID returns the device ID from either DeviceID or Device field.
func (m *WSMessage) GetDeviceID() string {
	if m.DeviceID != "" {
		return m.DeviceID
	}
	return m.Device
}

// IsOnline returns true if the device is online (Online == 1).
func (m *WSMessage) IsOnline() bool {
	return m.Online != nil && *m.Online == 1
}

// StatusChangeEvent represents a device status change event.
type StatusChangeEvent struct {
	Timestamp time.Time
	DeviceID  string
	Status    json.RawMessage
}

// SettingsChangeEvent represents a device settings change event.
type SettingsChangeEvent struct {
	Timestamp time.Time
	DeviceID  string
	Settings  json.RawMessage
}

// OnlineStatusEvent represents a device online/offline event.
type OnlineStatusEvent struct {
	Timestamp time.Time
	DeviceID  string
	Online    bool
}

// ActionRequest represents an action request to send to a device.
type ActionRequest struct {
	Params   any    `json:"params,omitempty"`
	Event    string `json:"event"`
	DeviceID string `json:"device_id"`
	Action   string `json:"action"`
}

// DeviceCommand represents a command to send to a device.
type DeviceCommand struct {
	Params   any    `json:"params,omitempty"`
	Event    string `json:"event"`
	DeviceID string `json:"device_id"`
	Action   string `json:"action"`
	Channel  int    `json:"channel,omitempty"`
}

// ConnectOptions contains options for WebSocket connection.
type ConnectOptions struct {
	// PingInterval is the interval between ping messages.
	// Default: 30 seconds
	PingInterval time.Duration

	// ReadTimeout is the timeout for reading messages.
	// Default: 60 seconds
	ReadTimeout time.Duration

	// ReconnectDelay is the delay before attempting to reconnect.
	// Default: 5 seconds
	ReconnectDelay time.Duration

	// MaxReconnectAttempts is the maximum number of reconnect attempts.
	// 0 means unlimited. Default: 0
	MaxReconnectAttempts int
}

// DefaultConnectOptions returns default connection options.
func DefaultConnectOptions() *ConnectOptions {
	return &ConnectOptions{
		PingInterval:         30 * time.Second,
		ReadTimeout:          60 * time.Second,
		ReconnectDelay:       5 * time.Second,
		MaxReconnectAttempts: 0,
	}
}

// AccessGroup represents the access level granted to the integrator.
type AccessGroup byte

const (
	// AccessGroupReadOnly allows only receiving status changes.
	AccessGroupReadOnly AccessGroup = 0x00

	// AccessGroupControl allows sending control commands.
	AccessGroupControl AccessGroup = 0x01
)

// String returns a string representation of the access group.
func (a AccessGroup) String() string {
	switch a {
	case AccessGroupReadOnly:
		return "read-only"
	case AccessGroupControl:
		return "control"
	default:
		return "unknown"
	}
}

// CanControl returns true if the access group allows control.
func (a AccessGroup) CanControl() bool {
	return a&AccessGroupControl != 0
}

// EventType represents the type of WebSocket event.
type EventType string

const (
	// EventStatusOnChange is sent when device status changes.
	EventStatusOnChange EventType = "Shelly:StatusOnChange"

	// EventSettings is sent when device settings change.
	EventSettings EventType = "Shelly:Settings"

	// EventOnline is sent when device online status changes.
	EventOnline EventType = "Shelly:Online"
)

// CloudServer represents a Shelly cloud server region.
type CloudServer struct {
	// Host is the server hostname.
	Host string

	// Region is the server region (e.g., "eu", "us").
	Region string

	// WSPort is the WebSocket port (default 6113).
	WSPort int
}

// DefaultCloudServers returns a list of known Shelly cloud servers.
func DefaultCloudServers() []CloudServer {
	return []CloudServer{
		{Host: "shelly-13-eu.shelly.cloud", Region: "eu", WSPort: 6113},
		{Host: "shelly-14-eu.shelly.cloud", Region: "eu", WSPort: 6113},
		{Host: "shelly-15-eu.shelly.cloud", Region: "eu", WSPort: 6113},
		{Host: "shelly-13-us.shelly.cloud", Region: "us", WSPort: 6113},
		{Host: "shelly-14-us.shelly.cloud", Region: "us", WSPort: 6113},
	}
}
