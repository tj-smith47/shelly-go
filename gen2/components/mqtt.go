package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// mqttComponentType is the type identifier for the MQTT component.
const mqttComponentType = "mqtt"

// MQTT represents a Shelly Gen2+ MQTT component.
//
// MQTT enables the device to connect to an MQTT broker for messaging,
// allowing integration with home automation systems like Home Assistant,
// openHAB, and custom applications.
//
// Note: MQTT component does not use component IDs.
// It is a singleton component accessed via "mqtt" key.
//
// Example:
//
//	mqtt := components.NewMQTT(device.Client())
//	status, err := mqtt.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("MQTT connected: %t\n", status.Connected)
//	}
type MQTT struct {
	client *rpc.Client
}

// NewMQTT creates a new MQTT component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	mqtt := components.NewMQTT(device.Client())
func NewMQTT(client *rpc.Client) *MQTT {
	return &MQTT{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (m *MQTT) Client() *rpc.Client {
	return m.client
}

// MQTTConfig represents the configuration of the MQTT component.
type MQTTConfig struct {
	// Enable enables or disables MQTT connection.
	Enable *bool `json:"enable,omitempty"`

	// Server is the MQTT broker hostname, optionally with port (host:port).
	// Default port is 1883 for non-TLS, 8883 for TLS.
	Server *string `json:"server,omitempty"`

	// ClientID is the MQTT client identifier.
	// If not set, defaults to the device ID.
	ClientID *string `json:"client_id,omitempty"`

	// User is the MQTT username for authentication.
	User *string `json:"user,omitempty"`

	// Pass is the MQTT password for authentication.
	// Write-only: not returned in GetConfig responses.
	Pass *string `json:"pass,omitempty"`

	// SSLCA controls TLS settings:
	//   - null or "": No TLS
	//   - "*": TLS without certificate verification
	//   - "ca.pem": TLS with default CA bundle verification
	//   - "user_ca.pem": TLS with user-provided CA (see Shelly.PutUserCA)
	SSLCA *string `json:"ssl_ca,omitempty"`

	// TopicPrefix is the prefix for MQTT topics.
	// Max 300 characters. Cannot start with $ and cannot contain #, +, %, ?
	// If not set, defaults to device ID.
	TopicPrefix *string `json:"topic_prefix,omitempty"`

	// RPCNTF enables RPC notifications (NotifyStatus, NotifyEvent) on
	// <topic_prefix>/events/rpc.
	// Default: true
	RPCNTF *bool `json:"rpc_ntf,omitempty"`

	// StatusNTF enables publishing complete component status on
	// <topic_prefix>/status/<component>:<id>.
	// Default: false
	StatusNTF *bool `json:"status_ntf,omitempty"`

	// UseClientCert enables client certificate authentication.
	// Default: false
	UseClientCert *bool `json:"use_client_cert,omitempty"`

	// EnableControl enables MQTT control feature.
	// Default: true
	EnableControl *bool `json:"enable_control,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// MQTTStatus represents the current status of the MQTT component.
type MQTTStatus struct {
	types.RawFields
	Connected bool `json:"connected"`
}

// GetConfig retrieves the MQTT configuration.
//
// Note: Password field is not returned for security reasons.
//
// Example:
//
//	config, err := mqtt.GetConfig(ctx)
//	if err == nil && config.Enable != nil && *config.Enable {
//	    fmt.Printf("MQTT enabled, server: %s\n", *config.Server)
//	}
func (m *MQTT) GetConfig(ctx context.Context) (*MQTTConfig, error) {
	resultJSON, err := m.client.Call(ctx, "MQTT.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config MQTTConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the MQTT configuration.
//
// Only non-nil fields will be updated.
//
// Example - Connect to MQTT broker:
//
//	err := mqtt.SetConfig(ctx, &MQTTConfig{
//	    Enable: ptr(true),
//	    Server: ptr("mqtt.example.com:1883"),
//	    User:   ptr("user"),
//	    Pass:   ptr("password"),
//	})
//
// Example - Enable TLS with verification:
//
//	err := mqtt.SetConfig(ctx, &MQTTConfig{
//	    Enable: ptr(true),
//	    Server: ptr("mqtt.example.com:8883"),
//	    SSLCA:  ptr("ca.pem"),
//	})
func (m *MQTT) SetConfig(ctx context.Context, config *MQTTConfig) error {
	params := map[string]any{
		"config": config,
	}

	_, err := m.client.Call(ctx, "MQTT.SetConfig", params)
	return err
}

// GetStatus retrieves the current MQTT status.
//
// Returns whether the device is currently connected to the MQTT broker.
//
// Example:
//
//	status, err := mqtt.GetStatus(ctx)
//	if err == nil {
//	    if status.Connected {
//	        fmt.Println("Connected to MQTT broker")
//	    } else {
//	        fmt.Println("Not connected to MQTT broker")
//	    }
//	}
func (m *MQTT) GetStatus(ctx context.Context) (*MQTTStatus, error) {
	resultJSON, err := m.client.Call(ctx, "MQTT.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status MQTTStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (m *MQTT) Type() string {
	return mqttComponentType
}

// Key returns the component key for aggregated status/config responses.
func (m *MQTT) Key() string {
	return mqttComponentType
}

// Ensure MQTT implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*MQTT)(nil)
