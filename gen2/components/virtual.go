package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// Virtual component type constants.
const (
	virtualBooleanType = "boolean"
	virtualNumberType  = "number"
	virtualButtonType  = "button"
)

// Virtual provides methods to manage virtual components on Shelly devices.
//
// Virtual components are a subset of dynamic components that allow users to
// interact with scripts. They are available on Gen3 and Gen2 Pro devices.
//
// Supported virtual component types:
//   - Boolean: True/false values with toggle capability
//   - Number: Numeric values with min/max range
//   - Text: String values with max length
//   - Enum: Enumerated values from a predefined list
//   - Button: Press events (no state)
//   - Group: Container for organizing other virtual components
//
// Component IDs are in the range [200..299] and there is a limit of 10
// instances per device.
//
// Example:
//
//	virtual := components.NewVirtual(client)
//	result, err := virtual.Add(ctx, "boolean", nil, 0)
//	if err == nil {
//	    fmt.Printf("Created virtual component with ID: %d\n", result.ID)
//	}
type Virtual struct {
	client *rpc.Client
}

// NewVirtual creates a new Virtual component manager.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	virtual := components.NewVirtual(rpcClient)
func NewVirtual(client *rpc.Client) *Virtual {
	return &Virtual{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (v *Virtual) Client() *rpc.Client {
	return v.client
}

// VirtualAddResult contains the result of adding a virtual component.
type VirtualAddResult struct {
	// ID is the ID of the newly created component.
	ID int `json:"id"`
}

// Add creates a new virtual component.
//
// Parameters:
//   - componentType: The type of component ("boolean", "text", "number", "enum", "group", "button")
//   - config: Optional configuration for the new component
//   - id: Optional specific ID (200-299). If 0, first free ID is used.
//
// Example - Create a boolean component:
//
//	result, err := virtual.Add(ctx, "boolean", nil, 0)
//
// Example - Create a number component with specific ID:
//
//	result, err := virtual.Add(ctx, "number", map[string]any{
//	    "name": "Temperature Setpoint",
//	    "min": 15.0,
//	    "max": 30.0,
//	}, 201)
func (v *Virtual) Add(
	ctx context.Context, componentType string, config map[string]any, id int,
) (*VirtualAddResult, error) {
	params := map[string]any{
		"type": componentType,
	}
	if config != nil {
		params["config"] = config
	}
	if id > 0 {
		params["id"] = id
	}

	resultJSON, err := v.client.Call(ctx, "Virtual.Add", params)
	if err != nil {
		return nil, err
	}

	var result VirtualAddResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete removes an existing virtual component.
//
// Parameters:
//   - key: Component key in format "<type>:<id>" (e.g., "boolean:200")
//
// Example:
//
//	err := virtual.Delete(ctx, "boolean:200")
func (v *Virtual) Delete(ctx context.Context, key string) error {
	params := map[string]any{
		"key": key,
	}

	_, err := v.client.Call(ctx, "Virtual.Delete", params)
	return err
}

// Type returns the component type identifier.
func (v *Virtual) Type() string {
	return "virtual"
}

// VirtualBoolean represents a virtual Boolean component.
//
// Boolean components store true/false values and support toggle operations.
// They can be used in scripts to track state.
//
// Example:
//
//	vBool := components.NewVirtualBoolean(client, 200)
//	status, err := vBool.GetStatus(ctx)
//	if err == nil {
//	    fmt.Printf("Value: %v\n", *status.Value)
//	}
type VirtualBoolean struct {
	client *rpc.Client
	id     int
}

// NewVirtualBoolean creates a new virtual Boolean component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (200-299)
//
// Example:
//
//	vBool := components.NewVirtualBoolean(rpcClient, 200)
func NewVirtualBoolean(client *rpc.Client, id int) *VirtualBoolean {
	return &VirtualBoolean{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (vb *VirtualBoolean) Client() *rpc.Client {
	return vb.client
}

// ID returns the component ID.
func (vb *VirtualBoolean) ID() int {
	return vb.id
}

// VirtualBooleanConfig represents the configuration of a virtual Boolean component.
type VirtualBooleanConfig struct {
	Name         *string      `json:"name,omitempty"`
	DefaultValue *bool        `json:"default_value,omitempty"`
	Persisted    *bool        `json:"persisted,omitempty"`
	Meta         *VirtualMeta `json:"meta,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// VirtualBooleanStatus represents the status of a virtual Boolean component.
type VirtualBooleanStatus struct {
	Value *bool `json:"value,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// VirtualMeta contains user-defined metadata for virtual components.
type VirtualMeta struct {
	// UI contains UI hints for the component.
	UI *VirtualMetaUI `json:"ui,omitempty"`
}

// VirtualMetaUI contains UI rendering hints.
type VirtualMetaUI struct {
	// View specifies the preferred UI view type.
	View *string `json:"view,omitempty"`

	// Icon specifies the icon to display.
	Icon *string `json:"icon,omitempty"`
}

// GetConfig retrieves the virtual Boolean configuration.
func (vb *VirtualBoolean) GetConfig(ctx context.Context) (*VirtualBooleanConfig, error) {
	params := map[string]any{
		"id": vb.id,
	}

	resultJSON, err := vb.client.Call(ctx, "Boolean.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config VirtualBooleanConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the virtual Boolean configuration.
func (vb *VirtualBoolean) SetConfig(ctx context.Context, config *VirtualBooleanConfig) error {
	configMap := make(map[string]any)

	if config.Name != nil {
		configMap["name"] = *config.Name
	}
	if config.DefaultValue != nil {
		configMap["default_value"] = *config.DefaultValue
	}
	if config.Persisted != nil {
		configMap["persisted"] = *config.Persisted
	}

	params := map[string]any{
		"id":     vb.id,
		"config": configMap,
	}

	_, err := vb.client.Call(ctx, "Boolean.SetConfig", params)
	return err
}

// GetStatus retrieves the current virtual Boolean status.
func (vb *VirtualBoolean) GetStatus(ctx context.Context) (*VirtualBooleanStatus, error) {
	params := map[string]any{
		"id": vb.id,
	}

	resultJSON, err := vb.client.Call(ctx, "Boolean.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status VirtualBooleanStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Set sets the boolean value.
//
// Example:
//
//	err := vBool.Set(ctx, true)
func (vb *VirtualBoolean) Set(ctx context.Context, value bool) error {
	params := map[string]any{
		"id":    vb.id,
		"value": value,
	}

	_, err := vb.client.Call(ctx, "Boolean.Set", params)
	return err
}

// Toggle toggles the boolean value.
//
// Example:
//
//	err := vBool.Toggle(ctx)
func (vb *VirtualBoolean) Toggle(ctx context.Context) error {
	params := map[string]any{
		"id": vb.id,
	}

	_, err := vb.client.Call(ctx, "Boolean.Toggle", params)
	return err
}

// Type returns the component type identifier.
func (vb *VirtualBoolean) Type() string {
	return virtualBooleanType
}

// Key returns the component key.
func (vb *VirtualBoolean) Key() string {
	return fmt.Sprintf("%s:%d", virtualBooleanType, vb.id)
}

// VirtualNumber represents a virtual Number component.
//
// Number components store numeric values within a defined range.
//
// Example:
//
//	vNum := components.NewVirtualNumber(client, 201)
//	status, err := vNum.GetStatus(ctx)
type VirtualNumber struct {
	client *rpc.Client
	id     int
}

// NewVirtualNumber creates a new virtual Number component accessor.
func NewVirtualNumber(client *rpc.Client, id int) *VirtualNumber {
	return &VirtualNumber{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (vn *VirtualNumber) Client() *rpc.Client {
	return vn.client
}

// ID returns the component ID.
func (vn *VirtualNumber) ID() int {
	return vn.id
}

// VirtualNumberConfig represents the configuration of a virtual Number component.
type VirtualNumberConfig struct {
	Name         *string      `json:"name,omitempty"`
	Min          *float64     `json:"min,omitempty"`
	Max          *float64     `json:"max,omitempty"`
	Step         *float64     `json:"step,omitempty"`
	DefaultValue *float64     `json:"default_value,omitempty"`
	Persisted    *bool        `json:"persisted,omitempty"`
	Unit         *string      `json:"unit,omitempty"`
	Meta         *VirtualMeta `json:"meta,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// VirtualNumberStatus represents the status of a virtual Number component.
type VirtualNumberStatus struct {
	Value *float64 `json:"value,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// GetConfig retrieves the virtual Number configuration.
func (vn *VirtualNumber) GetConfig(ctx context.Context) (*VirtualNumberConfig, error) {
	params := map[string]any{
		"id": vn.id,
	}

	resultJSON, err := vn.client.Call(ctx, "Number.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config VirtualNumberConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the virtual Number configuration.
func (vn *VirtualNumber) SetConfig(ctx context.Context, config *VirtualNumberConfig) error {
	configMap := make(map[string]any)

	if config.Name != nil {
		configMap["name"] = *config.Name
	}
	if config.Min != nil {
		configMap["min"] = *config.Min
	}
	if config.Max != nil {
		configMap["max"] = *config.Max
	}
	if config.Step != nil {
		configMap["step"] = *config.Step
	}
	if config.DefaultValue != nil {
		configMap["default_value"] = *config.DefaultValue
	}
	if config.Persisted != nil {
		configMap["persisted"] = *config.Persisted
	}
	if config.Unit != nil {
		configMap["unit"] = *config.Unit
	}

	params := map[string]any{
		"id":     vn.id,
		"config": configMap,
	}

	_, err := vn.client.Call(ctx, "Number.SetConfig", params)
	return err
}

// GetStatus retrieves the current virtual Number status.
func (vn *VirtualNumber) GetStatus(ctx context.Context) (*VirtualNumberStatus, error) {
	params := map[string]any{
		"id": vn.id,
	}

	resultJSON, err := vn.client.Call(ctx, "Number.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status VirtualNumberStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Set sets the numeric value.
//
// Example:
//
//	err := vNum.Set(ctx, 22.5)
func (vn *VirtualNumber) Set(ctx context.Context, value float64) error {
	params := map[string]any{
		"id":    vn.id,
		"value": value,
	}

	_, err := vn.client.Call(ctx, "Number.Set", params)
	return err
}

// Type returns the component type identifier.
func (vn *VirtualNumber) Type() string {
	return virtualNumberType
}

// Key returns the component key.
func (vn *VirtualNumber) Key() string {
	return fmt.Sprintf("%s:%d", virtualNumberType, vn.id)
}

// VirtualText represents a virtual Text component.
//
// Text components store string values.
type VirtualText struct {
	client *rpc.Client
	id     int
}

// NewVirtualText creates a new virtual Text component accessor.
func NewVirtualText(client *rpc.Client, id int) *VirtualText {
	return &VirtualText{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (vt *VirtualText) Client() *rpc.Client {
	return vt.client
}

// ID returns the component ID.
func (vt *VirtualText) ID() int {
	return vt.id
}

// VirtualTextConfig represents the configuration of a virtual Text component.
type VirtualTextConfig struct {
	Name         *string      `json:"name,omitempty"`
	MaxLen       *int         `json:"max_len,omitempty"`
	DefaultValue *string      `json:"default_value,omitempty"`
	Persisted    *bool        `json:"persisted,omitempty"`
	Meta         *VirtualMeta `json:"meta,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// VirtualTextStatus represents the status of a virtual Text component.
type VirtualTextStatus struct {
	Value *string `json:"value,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// GetConfig retrieves the virtual Text configuration.
func (vt *VirtualText) GetConfig(ctx context.Context) (*VirtualTextConfig, error) {
	params := map[string]any{
		"id": vt.id,
	}

	resultJSON, err := vt.client.Call(ctx, "Text.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config VirtualTextConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the virtual Text configuration.
func (vt *VirtualText) SetConfig(ctx context.Context, config *VirtualTextConfig) error {
	configMap := make(map[string]any)

	if config.Name != nil {
		configMap["name"] = *config.Name
	}
	if config.MaxLen != nil {
		configMap["max_len"] = *config.MaxLen
	}
	if config.DefaultValue != nil {
		configMap["default_value"] = *config.DefaultValue
	}
	if config.Persisted != nil {
		configMap["persisted"] = *config.Persisted
	}

	params := map[string]any{
		"id":     vt.id,
		"config": configMap,
	}

	_, err := vt.client.Call(ctx, "Text.SetConfig", params)
	return err
}

// GetStatus retrieves the current virtual Text status.
func (vt *VirtualText) GetStatus(ctx context.Context) (*VirtualTextStatus, error) {
	params := map[string]any{
		"id": vt.id,
	}

	resultJSON, err := vt.client.Call(ctx, "Text.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status VirtualTextStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Set sets the text value.
//
// Example:
//
//	err := vText.Set(ctx, "Hello, World!")
func (vt *VirtualText) Set(ctx context.Context, value string) error {
	params := map[string]any{
		"id":    vt.id,
		"value": value,
	}

	_, err := vt.client.Call(ctx, "Text.Set", params)
	return err
}

// Type returns the component type identifier.
func (vt *VirtualText) Type() string {
	return "text"
}

// Key returns the component key.
func (vt *VirtualText) Key() string {
	return fmt.Sprintf("text:%d", vt.id)
}

// VirtualEnum represents a virtual Enum component.
//
// Enum components store one value from a predefined list of options.
type VirtualEnum struct {
	client *rpc.Client
	id     int
}

// NewVirtualEnum creates a new virtual Enum component accessor.
func NewVirtualEnum(client *rpc.Client, id int) *VirtualEnum {
	return &VirtualEnum{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (ve *VirtualEnum) Client() *rpc.Client {
	return ve.client
}

// ID returns the component ID.
func (ve *VirtualEnum) ID() int {
	return ve.id
}

// VirtualEnumConfig represents the configuration of a virtual Enum component.
type VirtualEnumConfig struct {
	Name         *string      `json:"name,omitempty"`
	DefaultValue *string      `json:"default_value,omitempty"`
	Persisted    *bool        `json:"persisted,omitempty"`
	Meta         *VirtualMeta `json:"meta,omitempty"`
	types.RawFields
	Options []string `json:"options,omitempty"`
	ID      int      `json:"id"`
}

// VirtualEnumStatus represents the status of a virtual Enum component.
type VirtualEnumStatus struct {
	Value *string `json:"value,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// GetConfig retrieves the virtual Enum configuration.
func (ve *VirtualEnum) GetConfig(ctx context.Context) (*VirtualEnumConfig, error) {
	params := map[string]any{
		"id": ve.id,
	}

	resultJSON, err := ve.client.Call(ctx, "Enum.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config VirtualEnumConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the virtual Enum configuration.
func (ve *VirtualEnum) SetConfig(ctx context.Context, config *VirtualEnumConfig) error {
	configMap := make(map[string]any)

	if config.Name != nil {
		configMap["name"] = *config.Name
	}
	if len(config.Options) > 0 {
		configMap["options"] = config.Options
	}
	if config.DefaultValue != nil {
		configMap["default_value"] = *config.DefaultValue
	}
	if config.Persisted != nil {
		configMap["persisted"] = *config.Persisted
	}

	params := map[string]any{
		"id":     ve.id,
		"config": configMap,
	}

	_, err := ve.client.Call(ctx, "Enum.SetConfig", params)
	return err
}

// GetStatus retrieves the current virtual Enum status.
func (ve *VirtualEnum) GetStatus(ctx context.Context) (*VirtualEnumStatus, error) {
	params := map[string]any{
		"id": ve.id,
	}

	resultJSON, err := ve.client.Call(ctx, "Enum.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status VirtualEnumStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Set sets the enum value.
//
// The value must be one of the configured options.
//
// Example:
//
//	err := vEnum.Set(ctx, "option1")
func (ve *VirtualEnum) Set(ctx context.Context, value string) error {
	params := map[string]any{
		"id":    ve.id,
		"value": value,
	}

	_, err := ve.client.Call(ctx, "Enum.Set", params)
	return err
}

// Type returns the component type identifier.
func (ve *VirtualEnum) Type() string {
	return "enum"
}

// Key returns the component key.
func (ve *VirtualEnum) Key() string {
	return fmt.Sprintf("enum:%d", ve.id)
}

// VirtualButton represents a virtual Button component.
//
// Button components emit press events but don't hold state.
type VirtualButton struct {
	client *rpc.Client
	id     int
}

// NewVirtualButton creates a new virtual Button component accessor.
func NewVirtualButton(client *rpc.Client, id int) *VirtualButton {
	return &VirtualButton{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (vb *VirtualButton) Client() *rpc.Client {
	return vb.client
}

// ID returns the component ID.
func (vb *VirtualButton) ID() int {
	return vb.id
}

// VirtualButtonConfig represents the configuration of a virtual Button component.
type VirtualButtonConfig struct {
	Name *string      `json:"name,omitempty"`
	Meta *VirtualMeta `json:"meta,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// VirtualButtonStatus represents the status of a virtual Button component.
type VirtualButtonStatus struct {
	LastPressed *int64 `json:"last_pressed,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// GetConfig retrieves the virtual Button configuration.
func (vb *VirtualButton) GetConfig(ctx context.Context) (*VirtualButtonConfig, error) {
	params := map[string]any{
		"id": vb.id,
	}

	resultJSON, err := vb.client.Call(ctx, "Button.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config VirtualButtonConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the virtual Button configuration.
func (vb *VirtualButton) SetConfig(ctx context.Context, config *VirtualButtonConfig) error {
	configMap := make(map[string]any)

	if config.Name != nil {
		configMap["name"] = *config.Name
	}

	params := map[string]any{
		"id":     vb.id,
		"config": configMap,
	}

	_, err := vb.client.Call(ctx, "Button.SetConfig", params)
	return err
}

// GetStatus retrieves the current virtual Button status.
func (vb *VirtualButton) GetStatus(ctx context.Context) (*VirtualButtonStatus, error) {
	params := map[string]any{
		"id": vb.id,
	}

	resultJSON, err := vb.client.Call(ctx, "Button.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status VirtualButtonStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Trigger simulates a button press.
//
// Example:
//
//	err := vButton.Trigger(ctx)
func (vb *VirtualButton) Trigger(ctx context.Context) error {
	params := map[string]any{
		"id": vb.id,
	}

	_, err := vb.client.Call(ctx, "Button.Trigger", params)
	return err
}

// Type returns the component type identifier.
func (vb *VirtualButton) Type() string {
	return virtualButtonType
}

// Key returns the component key.
func (vb *VirtualButton) Key() string {
	return fmt.Sprintf("%s:%d", virtualButtonType, vb.id)
}

// VirtualGroup represents a virtual Group component.
//
// Group components organize other virtual components.
type VirtualGroup struct {
	client *rpc.Client
	id     int
}

// NewVirtualGroup creates a new virtual Group component accessor.
func NewVirtualGroup(client *rpc.Client, id int) *VirtualGroup {
	return &VirtualGroup{
		client: client,
		id:     id,
	}
}

// Client returns the underlying RPC client.
func (vg *VirtualGroup) Client() *rpc.Client {
	return vg.client
}

// ID returns the component ID.
func (vg *VirtualGroup) ID() int {
	return vg.id
}

// VirtualGroupConfig represents the configuration of a virtual Group component.
type VirtualGroupConfig struct {
	Name *string      `json:"name,omitempty"`
	Meta *VirtualMeta `json:"meta,omitempty"`
	types.RawFields
	Members []string `json:"members,omitempty"`
	ID      int      `json:"id"`
}

// VirtualGroupStatus represents the status of a virtual Group component.
type VirtualGroupStatus struct {
	types.RawFields
	ID int `json:"id"`
}

// GetConfig retrieves the virtual Group configuration.
func (vg *VirtualGroup) GetConfig(ctx context.Context) (*VirtualGroupConfig, error) {
	params := map[string]any{
		"id": vg.id,
	}

	resultJSON, err := vg.client.Call(ctx, "Group.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config VirtualGroupConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the virtual Group configuration.
func (vg *VirtualGroup) SetConfig(ctx context.Context, config *VirtualGroupConfig) error {
	configMap := make(map[string]any)

	if config.Name != nil {
		configMap["name"] = *config.Name
	}
	if len(config.Members) > 0 {
		configMap["members"] = config.Members
	}

	params := map[string]any{
		"id":     vg.id,
		"config": configMap,
	}

	_, err := vg.client.Call(ctx, "Group.SetConfig", params)
	return err
}

// GetStatus retrieves the current virtual Group status.
func (vg *VirtualGroup) GetStatus(ctx context.Context) (*VirtualGroupStatus, error) {
	params := map[string]any{
		"id": vg.id,
	}

	resultJSON, err := vg.client.Call(ctx, "Group.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status VirtualGroupStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Type returns the component type identifier.
func (vg *VirtualGroup) Type() string {
	return "group"
}

// Key returns the component key.
func (vg *VirtualGroup) Key() string {
	return fmt.Sprintf("group:%d", vg.id)
}

// Ensure all virtual components implement a minimal interface.
var (
	_ interface {
		Type() string
		Key() string
		ID() int
	} = (*VirtualBoolean)(nil)
	_ interface {
		Type() string
		Key() string
		ID() int
	} = (*VirtualNumber)(nil)
	_ interface {
		Type() string
		Key() string
		ID() int
	} = (*VirtualText)(nil)
	_ interface {
		Type() string
		Key() string
		ID() int
	} = (*VirtualEnum)(nil)
	_ interface {
		Type() string
		Key() string
		ID() int
	} = (*VirtualButton)(nil)
	_ interface {
		Type() string
		Key() string
		ID() int
	} = (*VirtualGroup)(nil)
)
