package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRawFields_GetString(t *testing.T) {
	rf := make(RawFields)
	rf["test"] = json.RawMessage(`"hello"`)
	rf["number"] = json.RawMessage(`42`)

	tests := []struct {
		name   string
		key    string
		want   string
		wantOK bool
	}{
		{
			name:   "existing string",
			key:    "test",
			want:   "hello",
			wantOK: true,
		},
		{
			name:   "non-existent key",
			key:    "missing",
			want:   "",
			wantOK: false,
		},
		{
			name:   "wrong type",
			key:    "number",
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := rf.GetString(tt.key)
			if ok != tt.wantOK {
				t.Errorf("RawFields.GetString() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("RawFields.GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRawFields_GetInt(t *testing.T) {
	rf := make(RawFields)
	rf["number"] = json.RawMessage(`42`)
	rf["string"] = json.RawMessage(`"test"`)

	tests := []struct {
		name   string
		key    string
		want   int
		wantOK bool
	}{
		{
			name:   "existing int",
			key:    "number",
			want:   42,
			wantOK: true,
		},
		{
			name:   "non-existent key",
			key:    "missing",
			want:   0,
			wantOK: false,
		},
		{
			name:   "wrong type",
			key:    "string",
			want:   0,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := rf.GetInt(tt.key)
			if ok != tt.wantOK {
				t.Errorf("RawFields.GetInt() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("RawFields.GetInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRawFields_GetFloat(t *testing.T) {
	rf := make(RawFields)
	rf["float"] = json.RawMessage(`3.14`)
	rf["string"] = json.RawMessage(`"test"`)

	tests := []struct {
		name   string
		key    string
		want   float64
		wantOK bool
	}{
		{
			name:   "existing float",
			key:    "float",
			want:   3.14,
			wantOK: true,
		},
		{
			name:   "non-existent key",
			key:    "missing",
			want:   0,
			wantOK: false,
		},
		{
			name:   "wrong type",
			key:    "string",
			want:   0,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := rf.GetFloat(tt.key)
			if ok != tt.wantOK {
				t.Errorf("RawFields.GetFloat() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("RawFields.GetFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRawFields_GetBool(t *testing.T) {
	rf := make(RawFields)
	rf["bool"] = json.RawMessage(`true`)
	rf["string"] = json.RawMessage(`"test"`)

	tests := []struct {
		name   string
		key    string
		want   bool
		wantOK bool
	}{
		{
			name:   "existing bool",
			key:    "bool",
			want:   true,
			wantOK: true,
		},
		{
			name:   "non-existent key",
			key:    "missing",
			want:   false,
			wantOK: false,
		},
		{
			name:   "wrong type",
			key:    "string",
			want:   false,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := rf.GetBool(tt.key)
			if ok != tt.wantOK {
				t.Errorf("RawFields.GetBool() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("RawFields.GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRawFields_Get(t *testing.T) {
	type testStruct struct {
		Value string `json:"value"`
	}

	rf := make(RawFields)
	rf["struct"] = json.RawMessage(`{"value":"test"}`)

	t.Run("get struct", func(t *testing.T) {
		var s testStruct
		ok := rf.Get("struct", &s)
		if !ok {
			t.Error("RawFields.Get() ok = false, want true")
		}
		if s.Value != "test" {
			t.Errorf("struct.Value = %v, want test", s.Value)
		}
	})

	t.Run("get non-existent", func(t *testing.T) {
		var s testStruct
		ok := rf.Get("missing", &s)
		if ok {
			t.Error("RawFields.Get() ok = true, want false")
		}
	})
}

func TestRawFields_Set(t *testing.T) {
	rf := make(RawFields)

	tests := []struct {
		value   any
		name    string
		key     string
		wantErr bool
	}{
		{
			name:  "set string",
			key:   "string",
			value: "test",
		},
		{
			name:  "set int",
			key:   "int",
			value: 42,
		},
		{
			name:  "set bool",
			key:   "bool",
			value: true,
		},
		{
			name:  "set struct",
			key:   "struct",
			value: struct{ Value string }{Value: "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rf.Set(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("RawFields.Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !rf.Has(tt.key) {
				t.Error("key not set in RawFields")
			}
		})
	}
}

func TestRawFields_Has(t *testing.T) {
	rf := make(RawFields)
	rf["exists"] = json.RawMessage(`"value"`)

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "existing key",
			key:  "exists",
			want: true,
		},
		{
			name: "non-existent key",
			key:  "missing",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rf.Has(tt.key); got != tt.want {
				t.Errorf("RawFields.Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRawFields_Delete(t *testing.T) {
	rf := make(RawFields)
	rf["test"] = json.RawMessage(`"value"`)

	if !rf.Has("test") {
		t.Fatal("test key not present before delete")
	}

	rf.Delete("test")

	if rf.Has("test") {
		t.Error("test key still present after delete")
	}
}

func TestRawFields_Keys(t *testing.T) {
	rf := make(RawFields)
	rf["key1"] = json.RawMessage(`"value1"`)
	rf["key2"] = json.RawMessage(`"value2"`)
	rf["key3"] = json.RawMessage(`"value3"`)

	keys := rf.Keys()
	if len(keys) != 3 {
		t.Errorf("RawFields.Keys() length = %v, want 3", len(keys))
	}

	// Check all keys are present (order doesn't matter)
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}
	for _, want := range []string{"key1", "key2", "key3"} {
		if !keyMap[want] {
			t.Errorf("RawFields.Keys() missing key %v", want)
		}
	}
}

func TestRawFields_Clone(t *testing.T) {
	rf := make(RawFields)
	rf["key1"] = json.RawMessage(`"value1"`)
	rf["key2"] = json.RawMessage(`42`)

	clone := rf.Clone()

	if len(clone) != len(rf) {
		t.Errorf("clone length = %v, want %v", len(clone), len(rf))
	}

	// Modify original
	rf["key3"] = json.RawMessage(`true`)

	// Clone should not have new key
	if clone.Has("key3") {
		t.Error("clone was modified when original was modified")
	}

	// Check cloned values
	val, ok := clone.GetString("key1")
	if !ok || val != "value1" {
		t.Errorf("cloned value = %v, want value1", val)
	}
}

func TestRawFields_Clone_Nil(t *testing.T) {
	var rf RawFields
	clone := rf.Clone()
	if clone != nil {
		t.Error("clone of nil RawFields should be nil")
	}
}

func TestRawFields_Merge(t *testing.T) {
	rf1 := make(RawFields)
	rf1["key1"] = json.RawMessage(`"value1"`)
	rf1["key2"] = json.RawMessage(`42`)

	rf2 := make(RawFields)
	rf2["key2"] = json.RawMessage(`99`)   // Overwrite
	rf2["key3"] = json.RawMessage(`true`) // Add new

	rf1.Merge(rf2)

	if !rf1.Has("key3") {
		t.Error("merged RawFields missing key3")
	}

	val, ok := rf1.GetInt("key2")
	if !ok || val != 99 {
		t.Errorf("merged value for key2 = %v, want 99", val)
	}
}

func TestRawFields_Integration(t *testing.T) {
	// Test that RawFields works correctly with embedded structs
	type ComponentStatus struct {
		RawFields RawFields `json:"-"`
		ID        int       `json:"id"`
		Output    bool      `json:"output"`
	}

	input := `{"id":0,"output":true,"temperature":25.5,"future_field":"value"}`

	// Unmarshal with custom handler to capture unknown fields
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	status := ComponentStatus{
		RawFields: make(RawFields),
	}

	// Manually unmarshal known fields
	_ = json.Unmarshal(data["id"], &status.ID)
	_ = json.Unmarshal(data["output"], &status.Output)

	// Store unknown fields
	for k, v := range data {
		if k != "id" && k != "output" {
			status.RawFields[k] = v
		}
	}

	// Verify known fields
	if status.ID != 0 || status.Output != true {
		t.Error("known fields not properly unmarshaled")
	}

	// Verify unknown fields are captured
	temp, ok := status.RawFields.GetFloat("temperature")
	if !ok || temp != 25.5 {
		t.Errorf("temperature = %v, want 25.5", temp)
	}

	future, ok := status.RawFields.GetString("future_field")
	if !ok || future != "value" {
		t.Errorf("future_field = %v, want value", future)
	}
}

func TestRawFields_RoundTrip(t *testing.T) {
	original := make(RawFields)
	_ = original.Set("string", "test")
	_ = original.Set("int", 42)
	_ = original.Set("bool", true)
	_ = original.Set("float", 3.14)

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal back
	var decoded RawFields
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Compare
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round trip failed: got %v, want %v", decoded, original)
	}
}
