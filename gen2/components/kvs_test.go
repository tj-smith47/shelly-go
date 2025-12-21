package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewKVS(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	kvs := NewKVS(client)

	if kvs == nil {
		t.Fatal("NewKVS returned nil")
	}

	if kvs.Type() != "kvs" {
		t.Errorf("Type() = %q, want %q", kvs.Type(), "kvs")
	}

	if kvs.Key() != "kvs" {
		t.Errorf("Key() = %q, want %q", kvs.Key(), "kvs")
	}

	if kvs.Client() != client {
		t.Error("Client() did not return the expected client")
	}
}

func TestKVS_Set(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    any
		wantEtag string
		wantRev  int
	}{
		{
			name:     "set string",
			key:      "name",
			value:    "test",
			wantEtag: "abc123",
			wantRev:  1,
		},
		{
			name:     "set number",
			key:      "counter",
			value:    42,
			wantEtag: "def456",
			wantRev:  2,
		},
		{
			name:     "set boolean",
			key:      "enabled",
			value:    true,
			wantEtag: "ghi789",
			wantRev:  3,
		},
		{
			name:     "set null",
			key:      "cleared",
			value:    nil,
			wantEtag: "jkl012",
			wantRev:  4,
		},
		{
			name:     "set float",
			key:      "temperature",
			value:    23.5,
			wantEtag: "mno345",
			wantRev:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "KVS.Set" {
						t.Errorf("method = %q, want %q", method, "KVS.Set")
					}
					return jsonrpcResponse(`{"etag": "` + tt.wantEtag + `", "rev": ` + string(rune('0'+tt.wantRev)) + `}`)
				},
			}
			client := rpc.NewClient(tr)
			kvs := NewKVS(client)

			result, err := kvs.Set(context.Background(), tt.key, tt.value)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			if result.Etag != tt.wantEtag {
				t.Errorf("result.Etag = %q, want %q", result.Etag, tt.wantEtag)
			}

			if result.Rev != tt.wantRev {
				t.Errorf("result.Rev = %d, want %d", result.Rev, tt.wantRev)
			}
		})
	}
}

func TestKVS_Set_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	kvs := NewKVS(client)
	testComponentError(t, "Set", func() error {
		_, err := kvs.Set(context.Background(), "key", "value")
		return err
	})
}

func TestKVS_SetWithEtag(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    any
		etag     string
		wantEtag string
	}{
		{
			name:     "update with etag",
			key:      "counter",
			value:    43,
			etag:     "abc123",
			wantEtag: "def456",
		},
		{
			name:     "update string with etag",
			key:      "name",
			value:    "updated",
			etag:     "old_etag",
			wantEtag: "new_etag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "KVS.Set" {
						t.Errorf("method = %q, want %q", method, "KVS.Set")
					}
					return jsonrpcResponse(`{"etag": "` + tt.wantEtag + `", "rev": 10}`)
				},
			}
			client := rpc.NewClient(tr)
			kvs := NewKVS(client)

			result, err := kvs.SetWithEtag(context.Background(), tt.key, tt.value, tt.etag)
			if err != nil {
				t.Fatalf("SetWithEtag() error = %v", err)
			}

			if result.Etag != tt.wantEtag {
				t.Errorf("result.Etag = %q, want %q", result.Etag, tt.wantEtag)
			}
		})
	}
}

func TestKVS_SetWithEtag_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	kvs := NewKVS(client)
	testComponentError(t, "SetWithEtag", func() error {
		_, err := kvs.SetWithEtag(context.Background(), "key", "value", "etag")
		return err
	})
}

func TestKVS_Get(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		result    string
		wantValue any
		wantEtag  string
	}{
		{
			name:      "get string",
			key:       "name",
			result:    `{"etag": "abc123", "value": "test_value"}`,
			wantValue: "test_value",
			wantEtag:  "abc123",
		},
		{
			name:      "get number",
			key:       "counter",
			result:    `{"etag": "def456", "value": 42}`,
			wantValue: float64(42),
			wantEtag:  "def456",
		},
		{
			name:      "get boolean",
			key:       "enabled",
			result:    `{"etag": "ghi789", "value": true}`,
			wantValue: true,
			wantEtag:  "ghi789",
		},
		{
			name:      "get null",
			key:       "empty",
			result:    `{"etag": "jkl012", "value": null}`,
			wantValue: nil,
			wantEtag:  "jkl012",
		},
		{
			name:      "get float",
			key:       "temperature",
			result:    `{"etag": "mno345", "value": 23.5}`,
			wantValue: 23.5,
			wantEtag:  "mno345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "KVS.Get" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			kvs := NewKVS(client)

			result, err := kvs.Get(context.Background(), tt.key)
			if err != nil {
				t.Errorf("Get() error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("Get() returned nil result")
			}

			if result.Value != tt.wantValue {
				t.Errorf("result.Value = %v (%T), want %v (%T)", result.Value, result.Value, tt.wantValue, tt.wantValue)
			}

			if result.Etag != tt.wantEtag {
				t.Errorf("result.Etag = %q, want %q", result.Etag, tt.wantEtag)
			}
		})
	}
}

func TestKVS_Get_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	kvs := NewKVS(client)
	testComponentError(t, "Get", func() error {
		_, err := kvs.Get(context.Background(), "key")
		return err
	})
}

func TestKVS_Get_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	kvs := NewKVS(client)
	testComponentInvalidJSON(t, "Get", func() error {
		_, err := kvs.Get(context.Background(), "key")
		return err
	})
}

func TestKVS_GetMany(t *testing.T) {
	tests := []struct {
		name      string
		match     string
		result    string
		wantCount int
	}{
		{
			name:  "multiple items",
			match: "sensor_*",
			result: `{
				"items": [
					{"key": "sensor_temp", "value": 23.5, "etag": "abc"},
					{"key": "sensor_humidity", "value": 65, "etag": "def"}
				]
			}`,
			wantCount: 2,
		},
		{
			name:      "no items",
			match:     "nonexistent_*",
			result:    `{"items": []}`,
			wantCount: 0,
		},
		{
			name:  "single item",
			match: "counter",
			result: `{
				"items": [
					{"key": "counter", "value": 42, "etag": "ghi"}
				]
			}`,
			wantCount: 1,
		},
		{
			name:  "all items",
			match: "*",
			result: `{
				"items": [
					{"key": "a", "value": 1},
					{"key": "b", "value": 2},
					{"key": "c", "value": 3}
				]
			}`,
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "KVS.GetMany" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			kvs := NewKVS(client)

			result, err := kvs.GetMany(context.Background(), tt.match)
			if err != nil {
				t.Errorf("GetMany() error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("GetMany() returned nil result")
			}

			if len(result.Items) != tt.wantCount {
				t.Errorf("len(Items) = %d, want %d", len(result.Items), tt.wantCount)
			}
		})
	}
}

func TestKVS_GetMany_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	kvs := NewKVS(client)
	testComponentError(t, "GetMany", func() error {
		_, err := kvs.GetMany(context.Background(), "*")
		return err
	})
}

func TestKVS_List(t *testing.T) {
	tests := []struct {
		name      string
		result    string
		wantCount int
		wantRev   int
	}{
		{
			name:      "multiple keys",
			result:    `{"keys": ["key1", "key2", "key3"], "rev": 5}`,
			wantCount: 3,
			wantRev:   5,
		},
		{
			name:      "no keys",
			result:    `{"keys": [], "rev": 0}`,
			wantCount: 0,
			wantRev:   0,
		},
		{
			name:      "single key",
			result:    `{"keys": ["only_key"], "rev": 1}`,
			wantCount: 1,
			wantRev:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "KVS.List" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			kvs := NewKVS(client)

			result, err := kvs.List(context.Background())
			if err != nil {
				t.Errorf("List() error = %v", err)
				return
			}

			if result == nil {
				t.Fatal("List() returned nil result")
			}

			if len(result.Keys) != tt.wantCount {
				t.Errorf("len(Keys) = %d, want %d", len(result.Keys), tt.wantCount)
			}

			if result.Rev != tt.wantRev {
				t.Errorf("Rev = %d, want %d", result.Rev, tt.wantRev)
			}
		})
	}
}

func TestKVS_List_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	kvs := NewKVS(client)
	testComponentError(t, "List", func() error {
		_, err := kvs.List(context.Background())
		return err
	})
}

func TestKVS_List_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	kvs := NewKVS(client)
	testComponentInvalidJSON(t, "List", func() error {
		_, err := kvs.List(context.Background())
		return err
	})
}

func TestKVS_Delete(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "KVS.Delete" {
				t.Errorf("method = %q, want %q", method, "KVS.Delete")
			}
			return jsonrpcResponse(`{"rev": 10}`)
		},
	}
	client := rpc.NewClient(tr)
	kvs := NewKVS(client)

	result, err := kvs.Delete(context.Background(), "old_key")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if result.Rev != 10 {
		t.Errorf("result.Rev = %d, want 10", result.Rev)
	}
}

func TestKVS_Delete_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	kvs := NewKVS(client)
	testComponentError(t, "Delete", func() error {
		_, err := kvs.Delete(context.Background(), "key")
		return err
	})
}

func TestKVSItem_JSONSerialization(t *testing.T) {
	tests := []struct {
		item  KVSItem
		check func(t *testing.T, data map[string]any)
		name  string
	}{
		{
			name: "full item",
			item: KVSItem{
				Key:   "test_key",
				Value: "test_value",
				Etag:  ptr("abc123"),
			},
			check: func(t *testing.T, data map[string]any) {
				if data["key"].(string) != "test_key" {
					t.Errorf("key = %v, want test_key", data["key"])
				}
				if data["value"].(string) != "test_value" {
					t.Errorf("value = %v, want test_value", data["value"])
				}
				if data["etag"].(string) != "abc123" {
					t.Errorf("etag = %v, want abc123", data["etag"])
				}
			},
		},
		{
			name: "item without etag",
			item: KVSItem{
				Key:   "simple_key",
				Value: 42,
			},
			check: func(t *testing.T, data map[string]any) {
				if _, ok := data["etag"]; ok {
					t.Error("etag should not be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.item)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			tt.check(t, parsed)
		})
	}
}

func TestKVS_ContextCancellation(t *testing.T) {
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					_ = req.GetMethod()
					select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return jsonrpcResponse(`{"keys": [], "rev": 0}`)
			}
		},
	}
	client := rpc.NewClient(tr)
	kvs := NewKVS(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := kvs.List(ctx)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
