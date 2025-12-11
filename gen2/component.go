package gen2

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Component represents a Gen2+ device component.
//
// Components are functional units within a device (e.g., Switch, Cover, Light).
// Each component has a type and ID, and supports GetConfig, SetConfig, and
// GetStatus operations.
type Component interface {
	// Type returns the component type (e.g., "switch", "cover", "light")
	Type() string

	// ID returns the component ID (instance number)
	ID() int

	// Key returns the component key in the format "type:id" (e.g., "switch:0")
	Key() string

	// GetConfig retrieves the component's configuration
	GetConfig(ctx context.Context) (json.RawMessage, error)

	// SetConfig updates the component's configuration
	SetConfig(ctx context.Context, config any) error

	// GetStatus retrieves the component's current status
	GetStatus(ctx context.Context) (json.RawMessage, error)
}

// BaseComponent provides common functionality for all Gen2+ components.
//
// Components should embed this struct and add their specific methods.
type BaseComponent struct {
	client *rpc.Client
	typ    string
	id     int
}

// NewBaseComponent creates a new base component.
func NewBaseComponent(client *rpc.Client, typ string, id int) *BaseComponent {
	return &BaseComponent{
		client: client,
		typ:    typ,
		id:     id,
	}
}

// Type returns the component type.
func (c *BaseComponent) Type() string {
	return c.typ
}

// ID returns the component ID.
func (c *BaseComponent) ID() int {
	return c.id
}

// Client returns the underlying RPC client.
//
// This can be used for advanced operations or custom RPC calls.
func (c *BaseComponent) Client() *rpc.Client {
	return c.client
}

// Key returns the component key in the format "type:id".
func (c *BaseComponent) Key() string {
	return fmt.Sprintf("%s:%d", c.typ, c.id)
}

// GetConfig retrieves the component's configuration.
//
// Returns the raw JSON configuration that can be unmarshaled into a
// component-specific config struct.
//
// Example:
//
//	comp := device.Switch(0)
//	configJSON, err := comp.GetConfig(ctx)
//	if err != nil {
//	    return err
//	}
//
//	var config SwitchConfig
//	json.Unmarshal(configJSON, &config)
func (c *BaseComponent) GetConfig(ctx context.Context) (json.RawMessage, error) {
	method := fmt.Sprintf("%s.GetConfig", c.capitalizedType())
	params := map[string]any{
		"id": c.id,
	}

	result, err := c.client.Call(ctx, method, params)
	if err != nil {
		return nil, fmt.Errorf("GetConfig failed for %s: %w", c.Key(), err)
	}

	return result, nil
}

// SetConfig updates the component's configuration.
//
// The config parameter can be a map or struct containing the configuration
// fields to update. Only provided fields will be updated.
//
// Example:
//
//	config := map[string]any{
//	    "name": "Living Room Light",
//	    "initial_state": "off",
//	}
//	err := comp.SetConfig(ctx, config)
func (c *BaseComponent) SetConfig(ctx context.Context, config any) error {
	method := fmt.Sprintf("%s.SetConfig", c.capitalizedType())

	// Build params with id and config fields
	configMap, ok := config.(map[string]any)
	if !ok {
		// Convert struct to map via JSON marshaling
		configJSON, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		if err := json.Unmarshal(configJSON, &configMap); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	// Add id to config map
	params := make(map[string]any)
	params["id"] = c.id
	for k, v := range configMap {
		params[k] = v
	}

	_, err := c.client.Call(ctx, method, params)
	if err != nil {
		return fmt.Errorf("SetConfig failed for %s: %w", c.Key(), err)
	}

	return nil
}

// GetStatus retrieves the component's current status.
//
// Returns the raw JSON status that can be unmarshaled into a
// component-specific status struct.
//
// Example:
//
//	comp := device.Switch(0)
//	statusJSON, err := comp.GetStatus(ctx)
//	if err != nil {
//	    return err
//	}
//
//	var status SwitchStatus
//	json.Unmarshal(statusJSON, &status)
func (c *BaseComponent) GetStatus(ctx context.Context) (json.RawMessage, error) {
	method := fmt.Sprintf("%s.GetStatus", c.capitalizedType())
	params := map[string]any{
		"id": c.id,
	}

	result, err := c.client.Call(ctx, method, params)
	if err != nil {
		return nil, fmt.Errorf("GetStatus failed for %s: %w", c.Key(), err)
	}

	return result, nil
}

// call is a helper method for calling component-specific RPC methods.
func (c *BaseComponent) call(ctx context.Context, method string, params, result any) error {
	fullMethod := fmt.Sprintf("%s.%s", c.capitalizedType(), method)

	if result != nil {
		return c.client.CallResult(ctx, fullMethod, params, result)
	}

	_, err := c.client.Call(ctx, fullMethod, params)
	return err
}

// componentTypeNames maps lowercase component types to their RPC method names.
// This is used for special cases where the RPC name doesn't follow standard capitalization.
var componentTypeNames = map[string]string{
	"em":           "EM",
	"em1":          "EM1",
	"pm":           "PM",
	"pm1":          "PM1",
	"kvs":          "KVS",
	"wifi":         "WiFi",
	"ble":          "BLE",
	"mqtt":         "Mqtt",
	"ui":           "UI",
	"sys":          "Sys",
	"ws":           "Ws",
	"bthome":       "BTHome",
	"bthomedevice": "BTHomeDevice",
	"bthomesensor": "BTHomeSensor",
	"rgb":          "RGB",
	"rgbw":         "RGBW",
	"ht_ui":        "HT_UI",
	"plugs_ui":     "Plugs_UI",
	"sensoraddon":  "SensorAddon",
	"devicepower":  "DevicePower",
}

// capitalizedType returns the component type with the first letter capitalized.
// This is used for RPC method names (e.g., "switch" -> "Switch").
func (c *BaseComponent) capitalizedType() string {
	if c.typ == "" {
		return ""
	}

	// Check for special cases first
	if name, ok := componentTypeNames[c.typ]; ok {
		return name
	}

	// Standard capitalization: first letter uppercase
	return string(c.typ[0]-32) + c.typ[1:]
}

// ComponentList represents a list of components on a device.
type ComponentList struct {
	Components []ComponentInfo `json:"components"`
}

// ComponentInfo contains information about a component.
type ComponentInfo struct {
	types.RawFields
	Key    string          `json:"key"`
	Status json.RawMessage `json:"status,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

// ParseComponentKey parses a component key into type and ID.
//
// Example: "switch:0" -> ("switch", 0)
func ParseComponentKey(key string) (compType string, compID int, err error) {
	parts := strings.Split(key, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid component key format: %s", key)
	}

	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid component ID in key %s: %w", key, err)
	}

	return parts[0], id, nil
}

// UnmarshalConfig retrieves and unmarshals component configuration.
//
// This is a generic helper that reduces boilerplate in component implementations.
// It calls GetConfig on the component and unmarshals the result into type T.
//
// Example:
//
//	func (s *Switch) GetConfig(ctx context.Context) (*SwitchConfig, error) {
//	    return gen2.UnmarshalConfig[SwitchConfig](ctx, s)
//	}
func UnmarshalConfig[T any](ctx context.Context, c Component) (*T, error) {
	configJSON, err := c.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	var config T
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// UnmarshalStatus retrieves and unmarshals component status.
//
// This is a generic helper that reduces boilerplate in component implementations.
// It calls GetStatus on the component and unmarshals the result into type T.
//
// Example:
//
//	func (s *Switch) GetStatus(ctx context.Context) (*SwitchStatus, error) {
//	    return gen2.UnmarshalStatus[SwitchStatus](ctx, s)
//	}
func UnmarshalStatus[T any](ctx context.Context, c Component) (*T, error) {
	statusJSON, err := c.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	var status T
	if err := json.Unmarshal(statusJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// SetConfigWithID sets configuration, ensuring the ID field matches the component ID.
//
// This helper automatically sets the ID field to match the component's ID if it's zero.
// This is useful for component SetConfig methods where users can omit the ID.
//
// Example:
//
//	func (s *Switch) SetConfig(ctx context.Context, config *SwitchConfig) error {
//	    return gen2.SetConfigWithID(ctx, s, config)
//	}
func SetConfigWithID(ctx context.Context, c Component, config any) error {
	// Use reflection to set ID field if present and zero
	v := reflect.ValueOf(config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		if idField := v.FieldByName("ID"); idField.IsValid() && idField.CanSet() {
			if idField.Int() == 0 {
				idField.SetInt(int64(c.ID()))
			}
		}
	}

	return c.SetConfig(ctx, config)
}

// EnsureIDParam ensures a parameter struct or map has its ID field set to the component's ID.
//
// If params is nil, returns a map with just the ID field.
// If params has an ID field that is zero, sets it to the component's ID.
//
// Example:
//
//	func (s *Switch) Toggle(ctx context.Context, params *SwitchToggleParams) error {
//	    params = gen2.EnsureIDParam(s, params).(*SwitchToggleParams)
//	    _, err := s.Client().Call(ctx, "Switch.Toggle", params)
//	    return err
//	}
func EnsureIDParam(c Component, params any) any {
	if params == nil {
		return map[string]any{"id": c.ID()}
	}

	v := reflect.ValueOf(params)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		if idField := v.FieldByName("ID"); idField.IsValid() && idField.CanSet() {
			if idField.Int() == 0 {
				idField.SetInt(int64(c.ID()))
			}
		}
	}

	return params
}

// EnsureID is a generic version of EnsureIDParam that returns the correct type.
// It ensures a parameter struct has its ID field set to the component's ID.
//
// Example:
//
//	func (s *Switch) Toggle(ctx context.Context, params *SwitchToggleParams) error {
//	    params = gen2.EnsureID(s, params)
//	    _, err := s.Client().Call(ctx, "Switch.Toggle", params)
//	    return err
//	}
func EnsureID[T any](c Component, params *T) *T {
	if params == nil {
		params = new(T)
	}

	v := reflect.ValueOf(params).Elem()

	if v.Kind() == reflect.Struct {
		if idField := v.FieldByName("ID"); idField.IsValid() && idField.CanSet() {
			if idField.Int() == 0 {
				idField.SetInt(int64(c.ID()))
			}
		}
	}

	return params
}
