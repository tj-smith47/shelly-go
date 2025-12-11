package components

import (
	"context"
	"encoding/json"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// kvsComponentType is the type identifier for the KVS component.
const kvsComponentType = "kvs"

// KVS represents a Shelly Gen2+ Key-Value Storage component.
//
// KVS provides persistent key-value storage for device scripts and
// external applications. Values persist across reboots and can store
// strings, numbers, booleans, and null values.
//
// Limits:
//   - Maximum 50 key-value pairs (varies by device)
//   - Key length: up to 42 bytes
//   - Value size: up to 256 bytes (strings)
//
// Note: KVS is commonly used by scripts to persist state between reboots.
//
// Example:
//
//	kvs := components.NewKVS(device.Client())
//	err := kvs.Set(ctx, "my_key", "my_value")
//	if err == nil {
//	    value, _ := kvs.Get(ctx, "my_key")
//	    fmt.Printf("Value: %v\n", value.Value)
//	}
type KVS struct {
	client *rpc.Client
}

// NewKVS creates a new KVS (Key-Value Storage) component accessor.
//
// Parameters:
//   - client: RPC client for communication
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	kvs := components.NewKVS(device.Client())
func NewKVS(client *rpc.Client) *KVS {
	return &KVS{
		client: client,
	}
}

// Client returns the underlying RPC client.
func (k *KVS) Client() *rpc.Client {
	return k.client
}

// KVSItem represents a key-value pair.
type KVSItem struct {
	Value any     `json:"value"`
	Etag  *string `json:"etag,omitempty"`
	types.RawFields
	Key string `json:"key"`
}

// KVSGetResponse represents the response from KVS.Get.
type KVSGetResponse struct {
	Value any `json:"value"`
	types.RawFields
	Etag string `json:"etag"`
}

// KVSGetManyResponse represents the response from KVS.GetMany.
type KVSGetManyResponse struct {
	types.RawFields
	Items []KVSItem `json:"items"`
}

// KVSListResponse represents the response from KVS.List.
type KVSListResponse struct {
	types.RawFields
	Keys []string `json:"keys"`
	Rev  int      `json:"rev"`
}

// KVSSetResponse represents the response from KVS.Set.
type KVSSetResponse struct {
	types.RawFields
	Etag string `json:"etag"`
	Rev  int    `json:"rev"`
}

// KVSDeleteResponse represents the response from KVS.Delete.
type KVSDeleteResponse struct {
	types.RawFields
	Rev int `json:"rev"`
}

// Set stores a value for a key.
//
// Parameters:
//   - key: Key name (up to 42 bytes)
//   - value: Value to store (string, number, boolean, or nil)
//
// Example:
//
//	result, err := kvs.Set(ctx, "counter", 42)
//	if err == nil {
//	    fmt.Printf("Stored with etag: %s\n", result.Etag)
//	}
func (k *KVS) Set(ctx context.Context, key string, value any) (*KVSSetResponse, error) {
	params := map[string]any{
		"key":   key,
		"value": value,
	}

	resultJSON, err := k.client.Call(ctx, "KVS.Set", params)
	if err != nil {
		return nil, err
	}

	var result KVSSetResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SetWithEtag stores a value for a key with optimistic concurrency.
//
// The operation only succeeds if the current etag matches.
// This prevents race conditions when multiple clients update the same key.
//
// Parameters:
//   - key: Key name
//   - value: Value to store
//   - etag: Expected current etag (for optimistic locking)
//
// Example:
//
//	// Get current value and etag
//	current, _ := kvs.Get(ctx, "counter")
//	newValue := current.Value.(float64) + 1
//	// Update only if etag matches
//	result, err := kvs.SetWithEtag(ctx, "counter", newValue, current.Etag)
func (k *KVS) SetWithEtag(ctx context.Context, key string, value any, etag string) (*KVSSetResponse, error) {
	params := map[string]any{
		"key":   key,
		"value": value,
		"etag":  etag,
	}

	resultJSON, err := k.client.Call(ctx, "KVS.Set", params)
	if err != nil {
		return nil, err
	}

	var result KVSSetResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get retrieves a value by key.
//
// Parameters:
//   - key: Key name to retrieve
//
// Example:
//
//	result, err := kvs.Get(ctx, "counter")
//	if err == nil {
//	    fmt.Printf("Value: %v, Etag: %s\n", result.Value, result.Etag)
//	}
func (k *KVS) Get(ctx context.Context, key string) (*KVSGetResponse, error) {
	params := map[string]any{
		"key": key,
	}

	resultJSON, err := k.client.Call(ctx, "KVS.Get", params)
	if err != nil {
		return nil, err
	}

	var result KVSGetResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetMany retrieves multiple values by key pattern.
//
// Parameters:
//   - match: Key pattern (supports * and ? wildcards)
//
// Example:
//
//	// Get all keys starting with "sensor_"
//	result, err := kvs.GetMany(ctx, "sensor_*")
//	if err == nil {
//	    for _, item := range result.Items {
//	        fmt.Printf("%s = %v\n", item.Key, item.Value)
//	    }
//	}
func (k *KVS) GetMany(ctx context.Context, match string) (*KVSGetManyResponse, error) {
	params := map[string]any{
		"match": match,
	}

	resultJSON, err := k.client.Call(ctx, "KVS.GetMany", params)
	if err != nil {
		return nil, err
	}

	var result KVSGetManyResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// List lists all key names.
//
// Example:
//
//	result, err := kvs.List(ctx)
//	if err == nil {
//	    fmt.Printf("Keys (%d):\n", len(result.Keys))
//	    for _, key := range result.Keys {
//	        fmt.Printf("  - %s\n", key)
//	    }
//	}
func (k *KVS) List(ctx context.Context) (*KVSListResponse, error) {
	resultJSON, err := k.client.Call(ctx, "KVS.List", nil)
	if err != nil {
		return nil, err
	}

	var result KVSListResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete removes a key-value pair.
//
// Parameters:
//   - key: Key name to delete
//
// Example:
//
//	result, err := kvs.Delete(ctx, "old_key")
//	if err == nil {
//	    fmt.Printf("Deleted. New revision: %d\n", result.Rev)
//	}
func (k *KVS) Delete(ctx context.Context, key string) (*KVSDeleteResponse, error) {
	params := map[string]any{
		"key": key,
	}

	resultJSON, err := k.client.Call(ctx, "KVS.Delete", params)
	if err != nil {
		return nil, err
	}

	var result KVSDeleteResponse
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Type returns the component type identifier.
func (k *KVS) Type() string {
	return kvsComponentType
}

// Key returns the component key for aggregated status/config responses.
func (k *KVS) Key() string {
	return kvsComponentType
}

// Ensure KVS implements a minimal component-like interface for documentation purposes.
var _ interface {
	Type() string
	Key() string
} = (*KVS)(nil)
