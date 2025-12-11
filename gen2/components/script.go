package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// scriptComponentType is the type identifier for the Script component.
const scriptComponentType = "script"

// Script represents a Shelly Gen2+ Script component.
//
// Script allows managing JavaScript scripts that run on the device.
// Scripts can respond to events, automate actions, and extend device
// functionality beyond the built-in features.
//
// Limits:
//   - Maximum 3 scripts per device (varies by model)
//   - Script size limited by device flash memory
//   - Scripts run in a sandboxed JavaScript environment
//
// Note: Script component uses numeric IDs starting from 1.
//
// Example:
//
//	script := components.NewScript(device.Client())
//	scripts, err := script.List(ctx)
//	if err == nil {
//	    for _, s := range scripts.Scripts {
//	        fmt.Printf("Script %d: %s (running: %t)\n", s.ID, *s.Name, s.Running)
//	    }
//	}
type Script struct {
	client *rpc.Client
}

// NewScript creates a new Script component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	script := components.NewScript(device.Client())
func NewScript(client *rpc.Client) *Script {
	return &Script{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (s *Script) Client() *rpc.Client {
	return s.client
}

// ScriptConfig represents the configuration of a script.
type ScriptConfig struct {
	Name   *string `json:"name,omitempty"`
	Enable *bool   `json:"enable,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// ScriptStatus represents the status of a script.
type ScriptStatus struct {
	MemUsage *int `json:"mem_usage,omitempty"`
	MemPeak  *int `json:"mem_peak,omitempty"`
	MemFree  *int `json:"mem_free,omitempty"`
	types.RawFields
	Errors  []string `json:"errors,omitempty"`
	ID      int      `json:"id"`
	Running bool     `json:"running"`
}

// ScriptListItem represents a script in the list response.
type ScriptListItem struct {
	Name *string `json:"name,omitempty"`
	types.RawFields
	ID      int  `json:"id"`
	Enable  bool `json:"enable"`
	Running bool `json:"running"`
}

// ScriptListResponse represents the response from Script.List.
type ScriptListResponse struct {
	types.RawFields
	Scripts []ScriptListItem `json:"scripts"`
}

// ScriptCreateResponse represents the response from Script.Create.
type ScriptCreateResponse struct {
	types.RawFields
	ID int `json:"id"`
}

// ScriptGetCodeResponse represents the response from Script.GetCode.
type ScriptGetCodeResponse struct {
	types.RawFields
	Data string `json:"data"`
}

// ScriptEvalResponse represents the response from Script.Eval.
type ScriptEvalResponse struct {
	// Result is the result of the evaluated expression.
	Result any `json:"result"`

	// RawFields captures any additional fields for future compatibility
	types.RawFields
}

// List retrieves all scripts on the device.
//
// Example:
//
//	result, err := script.List(ctx)
//	if err == nil {
//	    for _, s := range result.Scripts {
//	        fmt.Printf("Script %d: %s\n", s.ID, *s.Name)
//	    }
//	}
func (s *Script) List(ctx context.Context) (*ScriptListResponse, error) {
	resultJSON, err := s.client.Call(ctx, "Script.List", nil)
	if err != nil {
		return nil, err
	}

	var result ScriptListResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Create creates a new script slot.
//
// Parameters:
//   - name: Optional name for the script
//
// Example:
//
//	result, err := script.Create(ctx, ptr("My Script"))
//	if err == nil {
//	    fmt.Printf("Created script with ID: %d\n", result.ID)
//	}
func (s *Script) Create(ctx context.Context, name *string) (*ScriptCreateResponse, error) {
	params := map[string]any{}
	if name != nil {
		params["name"] = *name
	}

	resultJSON, err := s.client.Call(ctx, "Script.Create", params)
	if err != nil {
		return nil, err
	}

	var result ScriptCreateResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetConfig retrieves the configuration for a script.
//
// Parameters:
//   - id: Script ID
//
// Example:
//
//	config, err := script.GetConfig(ctx, 1)
//	if err == nil && config.Name != nil {
//	    fmt.Printf("Script name: %s\n", *config.Name)
//	}
func (s *Script) GetConfig(ctx context.Context, id int) (*ScriptConfig, error) {
	params := map[string]any{
		"id": id,
	}

	resultJSON, err := s.client.Call(ctx, "Script.GetConfig", params)
	if err != nil {
		return nil, err
	}

	var config ScriptConfig
	if err := json.Unmarshal(resultJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetConfig updates the configuration for a script.
//
// Parameters:
//   - id: Script ID
//   - config: New configuration
//
// Example:
//
//	err := script.SetConfig(ctx, 1, &ScriptConfig{
//	    Name:   ptr("Updated Script"),
//	    Enable: ptr(true),
//	})
func (s *Script) SetConfig(ctx context.Context, id int, config *ScriptConfig) error {
	params := map[string]any{
		"id":     id,
		"config": config,
	}

	_, err := s.client.Call(ctx, "Script.SetConfig", params)
	return err
}

// GetStatus retrieves the status of a script.
//
// Parameters:
//   - id: Script ID
//
// Example:
//
//	status, err := script.GetStatus(ctx, 1)
//	if err == nil {
//	    fmt.Printf("Running: %t, Memory: %d bytes\n", status.Running, *status.MemUsage)
//	}
func (s *Script) GetStatus(ctx context.Context, id int) (*ScriptStatus, error) {
	params := map[string]any{
		"id": id,
	}

	resultJSON, err := s.client.Call(ctx, "Script.GetStatus", params)
	if err != nil {
		return nil, err
	}

	var status ScriptStatus
	if err := json.Unmarshal(resultJSON, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// GetCode retrieves the source code of a script.
//
// Parameters:
//   - id: Script ID
//
// Example:
//
//	code, err := script.GetCode(ctx, 1)
//	if err == nil {
//	    fmt.Printf("Script code:\n%s\n", code.Data)
//	}
func (s *Script) GetCode(ctx context.Context, id int) (*ScriptGetCodeResponse, error) {
	params := map[string]any{
		"id": id,
	}

	resultJSON, err := s.client.Call(ctx, "Script.GetCode", params)
	if err != nil {
		return nil, err
	}

	var result ScriptGetCodeResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// PutCode uploads source code to a script.
//
// Parameters:
//   - id: Script ID
//   - code: JavaScript source code
//   - appendCode: If true, append to existing code; if false, replace
//
// Example:
//
//	err := script.PutCode(ctx, 1, "print('Hello, World!');", false)
func (s *Script) PutCode(ctx context.Context, id int, code string, appendCode bool) error {
	params := map[string]any{
		"id":     id,
		"code":   code,
		"append": appendCode,
	}

	_, err := s.client.Call(ctx, "Script.PutCode", params)
	return err
}

// Start starts a script.
//
// Parameters:
//   - id: Script ID
//
// Example:
//
//	err := script.Start(ctx, 1)
func (s *Script) Start(ctx context.Context, id int) error {
	params := map[string]any{
		"id": id,
	}

	_, err := s.client.Call(ctx, "Script.Start", params)
	return err
}

// Stop stops a running script.
//
// Parameters:
//   - id: Script ID
//
// Example:
//
//	err := script.Stop(ctx, 1)
func (s *Script) Stop(ctx context.Context, id int) error {
	params := map[string]any{
		"id": id,
	}

	_, err := s.client.Call(ctx, "Script.Stop", params)
	return err
}

// Delete deletes a script.
//
// Parameters:
//   - id: Script ID
//
// Example:
//
//	err := script.Delete(ctx, 1)
func (s *Script) Delete(ctx context.Context, id int) error {
	params := map[string]any{
		"id": id,
	}

	_, err := s.client.Call(ctx, "Script.Delete", params)
	return err
}

// Eval evaluates a JavaScript expression in the context of a running script.
//
// The script must be running for Eval to work.
//
// Parameters:
//   - id: Script ID
//   - code: JavaScript expression to evaluate
//
// Example:
//
//	result, err := script.Eval(ctx, 1, "1 + 2")
//	if err == nil {
//	    fmt.Printf("Result: %v\n", result.Result)
//	}
func (s *Script) Eval(ctx context.Context, id int, code string) (*ScriptEvalResponse, error) {
	params := map[string]any{
		"id":   id,
		"code": code,
	}

	resultJSON, err := s.client.Call(ctx, "Script.Eval", params)
	if err != nil {
		return nil, err
	}

	var result ScriptEvalResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Type returns the component type identifier.
func (s *Script) Type() string {
	return scriptComponentType
}

// Key returns the component key for aggregated status/config responses.
func (s *Script) Key() string {
	return scriptComponentType
}

// Ensure Script implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*Script)(nil)
