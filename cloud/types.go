package cloud

import (
	"encoding/json"
	"time"
)

// ClientID constants for OAuth authentication.
const (
	// ClientIDDIY is the client ID for DIY enthusiasts.
	// Tokens obtained with this client ID may be subject to rate restrictions.
	ClientIDDIY = "shelly-diy"
)

// OAuth endpoints and URLs.
const (
	// OAuthAuthorizeURL is the OAuth authorization endpoint.
	OAuthAuthorizeURL = "https://my.shelly.cloud/oauth_login.html"

	// OAuthTokenURL is the OAuth token exchange endpoint.
	//nolint:gosec // G101: False positive - this is a public API URL, not a credential
	OAuthTokenURL = "https://api.shelly.cloud/oauth/login"

	// DefaultWSPort is the default WebSocket port for real-time events.
	DefaultWSPort = 6113
)

// Token represents an OAuth access token from the Shelly Cloud API.
type Token struct {
	Expiry       time.Time `json:"-"`
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	UserAPIURL   string    `json:"user_api_url,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
}

// Valid reports whether the token is valid and not expired.
func (t *Token) Valid() bool {
	if t == nil || t.AccessToken == "" {
		return false
	}
	if t.Expiry.IsZero() {
		return true // No expiry set, assume valid
	}
	return time.Now().Before(t.Expiry)
}

// JWTClaims represents the claims in a Shelly Cloud JWT token.
type JWTClaims struct {
	UserAPIURL string `json:"user_api_url"`
	Email      string `json:"email,omitempty"`
	Issuer     string `json:"iss,omitempty"`
	UserID     int    `json:"user_id,omitempty"`
	IssuedAt   int64  `json:"iat,omitempty"`
	ExpiresAt  int64  `json:"exp,omitempty"`
}

// Device represents a device in the Shelly Cloud.
type Device struct {
	Extra           map[string]json.RawMessage `json:"extra,omitempty"`
	ID              string                     `json:"id"`
	Name            string                     `json:"name,omitempty"`
	Type            string                     `json:"type,omitempty"`
	Model           string                     `json:"model,omitempty"`
	MAC             string                     `json:"mac,omitempty"`
	FirmwareVersion string                     `json:"fw_ver,omitempty"`
	Generation      int                        `json:"gen,omitempty"`
	LastSeen        int64                      `json:"last_seen,omitempty"`
	Online          bool                       `json:"online"`
	CloudEnabled    bool                       `json:"cloud_enabled,omitempty"`
}

// DeviceStatus represents the status of a device in the Shelly Cloud.
type DeviceStatus struct {
	DevInfo  *DeviceInfo     `json:"_dev_info,omitempty"`
	ID       string          `json:"id"`
	Status   json.RawMessage `json:"status,omitempty"`
	Settings json.RawMessage `json:"settings,omitempty"`
	Online   bool            `json:"online"`
}

// DeviceInfo contains device information from the cloud.
type DeviceInfo struct {
	// Code is the device code/type identifier.
	Code string `json:"code,omitempty"`

	// Generation is the device generation.
	Generation int `json:"gen,omitempty"`

	// Online indicates if the device is online.
	Online bool `json:"online"`
}

// AllDevicesResponse represents the response from the device/all endpoint.
type AllDevicesResponse struct {
	Data   *AllDevicesData `json:"data,omitempty"`
	Errors []string        `json:"errors,omitempty"`
	IsOK   bool            `json:"isok"`
}

// AllDevicesData contains the data from the device/all response.
type AllDevicesData struct {
	// DevicesStatus is a map of device ID to device status.
	DevicesStatus map[string]*DeviceStatus `json:"devices_status,omitempty"`
}

// DeviceStatusResponse represents the response from the device/status endpoint.
type DeviceStatusResponse struct {
	Data   *DeviceStatusData `json:"data,omitempty"`
	Errors []string          `json:"errors,omitempty"`
	IsOK   bool              `json:"isok"`
}

// DeviceStatusData contains the device status data.
type DeviceStatusData struct {
	// DeviceStatus contains the status of the requested device.
	DeviceStatus *DeviceStatus `json:"device_status,omitempty"`

	// Online indicates if the device is online.
	Online bool `json:"online"`
}

// ControlRequest represents a device control request.
type ControlRequest struct {
	Blue       *int   `json:"blue,omitempty"`
	Effect     *int   `json:"effect,omitempty"`
	Timer      *int   `json:"timer,omitempty"`
	Red        *int   `json:"red,omitempty"`
	Position   *int   `json:"pos,omitempty"`
	Brightness *int   `json:"brightness,omitempty"`
	Green      *int   `json:"green,omitempty"`
	White      *int   `json:"white,omitempty"`
	ColorTemp  *int   `json:"color_temp,omitempty"`
	Gain       *int   `json:"gain,omitempty"`
	DeviceID   string `json:"id"`
	Direction  string `json:"direction,omitempty"`
	Turn       string `json:"turn,omitempty"`
	Channel    int    `json:"channel,omitempty"`
}

// ControlResponse represents the response from a control request.
type ControlResponse struct {
	Errors []string `json:"errors,omitempty"`
	IsOK   bool     `json:"isok"`
}

// GroupControlRequest represents a group control request.
type GroupControlRequest struct {
	// Switches is a list of switch control requests.
	Switches []GroupSwitch `json:"switches,omitempty"`

	// Covers is a list of cover control requests.
	Covers []GroupCover `json:"covers,omitempty"`

	// Lights is a list of light control requests.
	Lights []GroupLight `json:"lights,omitempty"`
}

// GroupSwitch represents a switch in a group control request.
type GroupSwitch struct {
	Turn string   `json:"turn"`
	IDs  []string `json:"ids"`
}

// GroupCover represents a cover in a group control request.
type GroupCover struct {
	Position  *int     `json:"pos,omitempty"`
	Direction string   `json:"direction,omitempty"`
	IDs       []string `json:"ids"`
}

// GroupLight represents a light in a group control request.
type GroupLight struct {
	Brightness *int     `json:"brightness,omitempty"`
	White      *int     `json:"white,omitempty"`
	Red        *int     `json:"red,omitempty"`
	Green      *int     `json:"green,omitempty"`
	Blue       *int     `json:"blue,omitempty"`
	Turn       string   `json:"turn,omitempty"`
	IDs        []string `json:"ids"`
}

// V2DevicesRequest represents a request to the v2/devices/api/get endpoint.
type V2DevicesRequest struct {
	// IDs is a list of device IDs to query (max 10).
	IDs []string `json:"ids"`

	// Select specifies additional data to fetch ("status", "settings").
	Select []string `json:"select,omitempty"`

	// Pick specifies properties to pick from the additional data.
	Pick []string `json:"pick,omitempty"`
}

// V2DevicesResponse represents the response from the v2/devices/api/get endpoint.
type V2DevicesResponse struct {
	// Devices is a map of device ID to device data.
	Devices map[string]*V2DeviceData `json:"devices,omitempty"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`
}

// V2DeviceData contains device data from the v2 API.
type V2DeviceData struct {
	ID       string          `json:"id"`
	Status   json.RawMessage `json:"status,omitempty"`
	Settings json.RawMessage `json:"settings,omitempty"`
	Online   bool            `json:"online"`
}

// WebSocketMessage represents a message from the WebSocket connection.
type WebSocketMessage struct {
	Event     string          `json:"event"`
	DeviceID  string          `json:"device_id,omitempty"`
	Status    json.RawMessage `json:"status,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Channel   int             `json:"channel,omitempty"`
	Timestamp int64           `json:"ts,omitempty"`
}

// EventType constants for WebSocket events.
const (
	// EventDeviceOnline is sent when a device comes online.
	EventDeviceOnline = "Shelly:Online"

	// EventDeviceOffline is sent when a device goes offline.
	EventDeviceOffline = "Shelly:Offline"

	// EventDeviceStatusChange is sent when device status changes.
	EventDeviceStatusChange = "Shelly:StatusChange"

	// EventNotifyStatus is sent for Gen2+ device status notifications.
	EventNotifyStatus = "NotifyStatus"

	// EventNotifyFullStatus is sent for Gen2+ full status notifications.
	EventNotifyFullStatus = "NotifyFullStatus"

	// EventNotifyEvent is sent for Gen2+ events (button press, etc.).
	EventNotifyEvent = "NotifyEvent"
)

// LoginRequest represents a login request for OAuth token exchange.
type LoginRequest struct {
	// Email is the user's email address.
	Email string `json:"email"`

	// Password is the SHA1 hash of the user's password.
	Password string `json:"password"`

	// ClientID is the OAuth client ID.
	ClientID string `json:"client_id"`
}

// LoginResponse represents the response from the login endpoint.
type LoginResponse struct {
	Data   *LoginData `json:"data,omitempty"`
	Errors []string   `json:"errors,omitempty"`
	IsOK   bool       `json:"isok"`
}

// LoginData contains the login response data.
type LoginData struct {
	// Token is the access token.
	Token string `json:"token"`

	// UserAPIURL is the designated API server.
	UserAPIURL string `json:"user_api_url,omitempty"`
}
