package types

import "encoding/json"

// RawFields is a map that captures unknown JSON fields for forward compatibility.
// When Shelly releases new firmware with additional fields, existing code won't
// break because unknown fields are preserved in this map.
//
// This allows the library to:
//  1. Parse responses from newer firmware without errors
//  2. Preserve unknown fields when reading and writing configs
//  3. Future-proof applications against firmware updates
//
// Example usage:
//
//	type ComponentStatus struct {
//	    Output     bool      `json:"output"`
//	    Temperature float64  `json:"temperature,omitempty"`
//	    RawFields           // Embeds RawFields to capture unknown fields
//	}
type RawFields map[string]json.RawMessage

// GetString retrieves a string value from RawFields if it exists.
func (r RawFields) GetString(key string) (string, bool) {
	raw, ok := r[key]
	if !ok {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

// GetInt retrieves an int value from RawFields if it exists.
func (r RawFields) GetInt(key string) (int, bool) {
	raw, ok := r[key]
	if !ok {
		return 0, false
	}
	var i int
	if err := json.Unmarshal(raw, &i); err != nil {
		return 0, false
	}
	return i, true
}

// GetFloat retrieves a float64 value from RawFields if it exists.
func (r RawFields) GetFloat(key string) (float64, bool) {
	raw, ok := r[key]
	if !ok {
		return 0, false
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return 0, false
	}
	return f, true
}

// GetBool retrieves a bool value from RawFields if it exists.
func (r RawFields) GetBool(key string) (value, ok bool) {
	raw, ok := r[key]
	if !ok {
		return false, false
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err != nil {
		return false, false
	}
	return b, true
}

// Get retrieves and unmarshals a value from RawFields into v.
// Returns true if the key exists and unmarshaling succeeds.
func (r RawFields) Get(key string, v any) bool {
	raw, ok := r[key]
	if !ok {
		return false
	}
	return json.Unmarshal(raw, v) == nil
}

// Set stores a value in RawFields by marshaling it to JSON.
// Returns an error if marshaling fails.
func (r RawFields) Set(key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r[key] = data
	return nil
}

// Has returns true if the key exists in RawFields.
func (r RawFields) Has(key string) bool {
	_, ok := r[key]
	return ok
}

// Delete removes a key from RawFields.
func (r RawFields) Delete(key string) {
	delete(r, key)
}

// Keys returns all keys in RawFields.
func (r RawFields) Keys() []string {
	keys := make([]string, 0, len(r))
	for k := range r {
		keys = append(keys, k)
	}
	return keys
}

// Clone creates a deep copy of RawFields.
func (r RawFields) Clone() RawFields {
	if r == nil {
		return nil
	}
	clone := make(RawFields, len(r))
	for k, v := range r {
		// Make a copy of the raw message
		vcopy := make(json.RawMessage, len(v))
		copy(vcopy, v)
		clone[k] = vcopy
	}
	return clone
}

// Merge merges another RawFields into this one.
// Values from other will overwrite values in r for duplicate keys.
func (r RawFields) Merge(other RawFields) {
	for k, v := range other {
		r[k] = v
	}
}
