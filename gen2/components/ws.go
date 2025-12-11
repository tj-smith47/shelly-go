package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Ws represents a Shelly Gen2+ Outbound WebSocket component.
//
// Ws configures the device to establish and maintain an outbound WebSocket
// connection to a remote server. This allows:
//   - Remote RPC control through the WebSocket connection
//   - Unsolicited status notifications on connect
//   - Similar features to inbound WS and MQTT channels
//
// Note: Ws component does not use component IDs.
// It is a singleton component accessed via "ws" key.
//
// Example:
//
//	ws := components.NewWs(device.Client())
//	status, err := ws.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("WebSocket connected: %t\n", status.Connected)
//	}
type Ws struct {
	client *rpc.Client
}

// NewWs creates a new Ws (Outbound WebSocket) component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	ws := components.NewWs(device.Client())
func NewWs(client *rpc.Client) *Ws {
	return &Ws{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (w *Ws) Client() *rpc.Client {
	return w.client
}

// WsConfig represents the configuration of the Ws component.
type WsConfig struct {
	// Enable enables or disables the outbound WebSocket connection.
	Enable *bool `json:"enable,omitempty"`

	// Server is the WebSocket server URL.
	// Format: ws://host:port/path or wss://host:port/path
	Server *string `json:"server,omitempty"`

	// SSLCA controls TLS settings:
	//   - null or "": No TLS verification (use ws://)
	//   - "*": TLS without certificate verification
	//   - "ca.pem": TLS with default CA bundle verification
	//   - "user_ca.pem": TLS with user-provided CA
	SSLCA *string `json:"ssl_ca,omitempty"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// WsStatus represents the current status of the Ws component.
type WsStatus struct {
	types.RawFields
	Connected bool `json:"connected"`
}

// GetConfig retrieves the Ws configuration.
//
// Example:
//
//	config, err := ws.GetConfig(ctx)
//	if err == nil && config.Enable != nil && *config.Enable {
//	    fmt.Printf("Outbound WS server: %s\n", *config.Server)
//	}
func (w *Ws) GetConfig(ctx context.Context) (*WsConfig, error) {
	resultJSON, err := w.client.Call(ctx, "Ws.GetConfig", nil)
	if err != nil {
		return nil, err
	}

	var config WsConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the Ws configuration.
//
// Only non-nil fields will be updated.
//
// Example - Connect to a WebSocket server:
//
//	err := ws.SetConfig(ctx, &WsConfig{
//	    Enable: ptr(true),
//	    Server: ptr("ws://myserver.example.com:8080/shelly"),
//	})
//
// Example - Enable TLS with verification:
//
//	err := ws.SetConfig(ctx, &WsConfig{
//	    Enable: ptr(true),
//	    Server: ptr("wss://secure.example.com:8443/shelly"),
//	    SSLCA:  ptr("ca.pem"),
//	})
func (w *Ws) SetConfig(ctx context.Context, config *WsConfig) error {
	params := map[string]any{
		"config": config,
	}

	_, err := w.client.Call(ctx, "Ws.SetConfig", params)
	return err
}

// GetStatus retrieves the current Ws status.
//
// Returns whether the device is currently connected to the WebSocket server.
//
// Example:
//
//	status, err := ws.GetStatus(ctx)
//	if err == nil {
//	    if status.Connected {
//	        fmt.Println("Connected to WebSocket server")
//	    } else {
//	        fmt.Println("Not connected to WebSocket server")
//	    }
//	}
func (w *Ws) GetStatus(ctx context.Context) (*WsStatus, error) {
	resultJSON, err := w.client.Call(ctx, "Ws.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var status WsStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (w *Ws) Type() string {
	return "ws"
}

// Key returns the component key for aggregated status/config responses.
func (w *Ws) Key() string {
	return "ws"
}

// Ensure Ws implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Ws)(nil)
