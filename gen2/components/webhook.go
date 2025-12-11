package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// webhookComponentType is the type identifier for the Webhook component.
const webhookComponentType = "webhook"

// Webhook represents a Shelly Gen2+ Webhook component.
//
// Webhook allows the device to send HTTP requests triggered by events.
// Events occur when device components change state (switch toggles,
// button presses, sensor readings, etc.).
//
// Limits:
//   - 20 webhook instances per device (10 for battery-operated devices)
//
// Note: Changing the device profile will delete all webhooks.
//
// Example:
//
//	webhook := components.NewWebhook(device.Client())
//	hooks, err := webhook.List(ctx)
//	if err == nil {
//	    fmt.Printf("Found %d webhooks\n", len(hooks.Hooks))
//	}
type Webhook struct {
	client *rpc.Client
}

// NewWebhook creates a new Webhook component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	webhook := components.NewWebhook(device.Client())
func NewWebhook(client *rpc.Client) *Webhook {
	return &Webhook{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (w *Webhook) Client() *rpc.Client {
	return w.client
}

// WebhookConfig represents a webhook configuration.
type WebhookConfig struct {
	ID           *int    `json:"id,omitempty"`
	Name         *string `json:"name,omitempty"`
	SSLCA        *string `json:"ssl_ca,omitempty"`
	Condition    *string `json:"condition,omitempty"`
	RepeatPeriod *int    `json:"repeat_period,omitempty"`
	types.RawFields
	Event         string   `json:"event"`
	URLs          []string `json:"urls"`
	ActiveBetween []string `json:"active_between,omitempty"`
	Cid           int      `json:"cid"`
	Enable        bool     `json:"enable"`
}

// WebhookListResponse represents the response from Webhook.List.
type WebhookListResponse struct {
	types.RawFields
	Hooks []WebhookConfig `json:"hooks"`
	Rev   int             `json:"rev"`
}

// WebhookCreateResponse represents the response from Webhook.Create.
type WebhookCreateResponse struct {
	types.RawFields
	ID  int `json:"id"`
	Rev int `json:"rev"`
}

// WebhookUpdateResponse represents the response from Webhook.Update.
type WebhookUpdateResponse struct {
	types.RawFields
	Rev int `json:"rev"`
}

// WebhookDeleteResponse represents the response from Webhook.Delete.
type WebhookDeleteResponse struct {
	types.RawFields
	Rev int `json:"rev"`
}

// WebhookSupportedEvent represents a supported webhook event.
type WebhookSupportedEvent struct {
	types.RawFields
	Event string `json:"event"`
}

// WebhookListSupportedResponse represents the response from Webhook.ListSupported.
type WebhookListSupportedResponse struct {
	types.RawFields
	HookTypes []string `json:"hook_types"`
}

// List retrieves all configured webhooks.
//
// Example:
//
//	result, err := webhook.List(ctx)
//	if err == nil {
//	    for _, hook := range result.Hooks {
//	        fmt.Printf("Webhook %d: %s -> %v\n", *hook.ID, hook.Event, hook.URLs)
//	    }
//	}
func (w *Webhook) List(ctx context.Context) (*WebhookListResponse, error) {
	resultJSON, err := w.client.Call(ctx, "Webhook.List", nil)
	if err != nil {
		return nil, err
	}

	var result WebhookListResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Create creates a new webhook.
//
// Parameters:
//   - config: Webhook configuration (ID field is ignored, will be assigned by system)
//
// Example:
//
//	result, err := webhook.Create(ctx, &WebhookConfig{
//	    Cid:    0,
//	    Enable: true,
//	    Event:  "switch.on",
//	    Name:   ptr("Notify on switch"),
//	    URLs:   []string{"http://example.com/webhook"},
//	})
//	if err == nil {
//	    fmt.Printf("Created webhook with ID: %d\n", result.ID)
//	}
func (w *Webhook) Create(ctx context.Context, config *WebhookConfig) (*WebhookCreateResponse, error) {
	// Don't send ID when creating
	params := map[string]any{
		"cid":    config.Cid,
		"enable": config.Enable,
		"event":  config.Event,
		"urls":   config.URLs,
	}

	if config.Name != nil {
		params["name"] = *config.Name
	}
	if config.SSLCA != nil {
		params["ssl_ca"] = *config.SSLCA
	}
	if config.ActiveBetween != nil {
		params["active_between"] = config.ActiveBetween
	}
	if config.Condition != nil {
		params["condition"] = *config.Condition
	}
	if config.RepeatPeriod != nil {
		params["repeat_period"] = *config.RepeatPeriod
	}

	resultJSON, err := w.client.Call(ctx, "Webhook.Create", params)
	if err != nil {
		return nil, err
	}

	var result WebhookCreateResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Update updates an existing webhook.
//
// Parameters:
//   - id: Webhook ID to update
//   - config: New configuration (only non-nil fields are updated)
//
// Example:
//
//	_, err := webhook.Update(ctx, 1, &WebhookConfig{
//	    Enable: false,
//	})
func (w *Webhook) Update(ctx context.Context, id int, config *WebhookConfig) (*WebhookUpdateResponse, error) {
	params := map[string]any{
		"id": id,
	}

	// Only include fields that are set
	if config.Enable {
		params["enable"] = config.Enable
	}
	if config.Event != "" {
		params["event"] = config.Event
	}
	if len(config.URLs) > 0 {
		params["urls"] = config.URLs
	}
	if config.Name != nil {
		params["name"] = *config.Name
	}
	if config.SSLCA != nil {
		params["ssl_ca"] = *config.SSLCA
	}
	if config.ActiveBetween != nil {
		params["active_between"] = config.ActiveBetween
	}
	if config.Condition != nil {
		params["condition"] = *config.Condition
	}
	if config.RepeatPeriod != nil {
		params["repeat_period"] = *config.RepeatPeriod
	}

	resultJSON, err := w.client.Call(ctx, "Webhook.Update", params)
	if err != nil {
		return nil, err
	}

	var result WebhookUpdateResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete deletes a webhook by ID.
//
// Example:
//
//	_, err := webhook.Delete(ctx, 1)
func (w *Webhook) Delete(ctx context.Context, id int) (*WebhookDeleteResponse, error) {
	params := map[string]any{
		"id": id,
	}

	resultJSON, err := w.client.Call(ctx, "Webhook.Delete", params)
	if err != nil {
		return nil, err
	}

	var result WebhookDeleteResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteAll deletes all webhooks.
//
// Example:
//
//	_, err := webhook.DeleteAll(ctx)
func (w *Webhook) DeleteAll(ctx context.Context) (*WebhookDeleteResponse, error) {
	resultJSON, err := w.client.Call(ctx, "Webhook.DeleteAll", nil)
	if err != nil {
		return nil, err
	}

	var result WebhookDeleteResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListSupported lists all supported webhook event types.
//
// Example:
//
//	result, err := webhook.ListSupported(ctx)
//	if err == nil {
//	    fmt.Println("Supported events:")
//	    for _, event := range result.HookTypes {
//	        fmt.Printf("  - %s\n", event)
//	    }
//	}
func (w *Webhook) ListSupported(ctx context.Context) (*WebhookListSupportedResponse, error) {
	resultJSON, err := w.client.Call(ctx, "Webhook.ListSupported", nil)
	if err != nil {
		return nil, err
	}

	var result WebhookListSupportedResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Type returns the component type identifier.
func (w *Webhook) Type() string {
	return webhookComponentType
}

// Key returns the component key for aggregated status/config responses.
func (w *Webhook) Key() string {
	return webhookComponentType
}

// Ensure Webhook implements a minimal component-like interface.
var _ interface {
	Type() string
	Key() string
} = (*Webhook)(nil)
