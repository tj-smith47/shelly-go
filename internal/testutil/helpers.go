package testutil

import (
	"embed"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

//go:embed fixtures
var fixturesFS embed.FS

// LoadFixture loads a JSON fixture file from the fixtures directory.
// Returns the raw JSON bytes.
func LoadFixture(name string) ([]byte, error) {
	return fixturesFS.ReadFile(filepath.Join("fixtures", name))
}

// MustLoadFixture loads a fixture and panics on error.
func MustLoadFixture(name string) []byte {
	data, err := LoadFixture(name)
	if err != nil {
		panic("failed to load fixture " + name + ": " + err.Error())
	}
	return data
}

// LoadFixtureJSON loads a fixture and unmarshals it into the given value.
func LoadFixtureJSON(name string, v any) error {
	data, err := LoadFixture(name)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// AssertEqual asserts that two values are equal.
func AssertEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

// AssertNotEqual asserts that two values are not equal.
func AssertNotEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if reflect.DeepEqual(expected, actual) {
		t.Errorf("expected values to differ, both are %v", expected)
	}
}

// AssertNil asserts that a value is nil.
func AssertNil(t *testing.T, actual any) {
	t.Helper()
	if actual != nil && !reflect.ValueOf(actual).IsNil() {
		t.Errorf("expected nil, got %v", actual)
	}
}

// AssertNotNil asserts that a value is not nil.
func AssertNotNil(t *testing.T, actual any) {
	t.Helper()
	if actual == nil || reflect.ValueOf(actual).IsNil() {
		t.Error("expected non-nil value")
	}
}

// AssertNoError asserts that an error is nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// AssertError asserts that an error is not nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// AssertErrorContains asserts that an error contains a substring.
func AssertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error containing %q, got nil", substr)
		return
	}
	if !containsString(err.Error(), substr) {
		t.Errorf("expected error containing %q, got %v", substr, err)
	}
}

// AssertTrue asserts that a value is true.
func AssertTrue(t *testing.T, actual bool) {
	t.Helper()
	if !actual {
		t.Error("expected true, got false")
	}
}

// AssertFalse asserts that a value is false.
func AssertFalse(t *testing.T, actual bool) {
	t.Helper()
	if actual {
		t.Error("expected false, got true")
	}
}

// AssertLen asserts the length of a slice, map, or string.
func AssertLen(t *testing.T, obj any, length int) {
	t.Helper()
	v := reflect.ValueOf(obj)
	if v.Len() != length {
		t.Errorf("expected length %d, got %d", length, v.Len())
	}
}

// AssertContains asserts that a slice contains an element.
func AssertContains(t *testing.T, slice, element any) {
	t.Helper()
	v := reflect.ValueOf(slice)
	for i := 0; i < v.Len(); i++ {
		if reflect.DeepEqual(v.Index(i).Interface(), element) {
			return
		}
	}
	t.Errorf("slice does not contain %v", element)
}

// AssertStringContains asserts that a string contains a substring.
func AssertStringContains(t *testing.T, s, substr string) {
	t.Helper()
	if !containsString(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}

// MustJSON marshals a value to JSON and panics on error.
func MustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic("failed to marshal JSON: " + err.Error())
	}
	return data
}

// MustJSONRaw marshals a value to json.RawMessage.
func MustJSONRaw(v any) json.RawMessage {
	return json.RawMessage(MustJSON(v))
}

// JSONEqual checks if two JSON values are equal.
func JSONEqual(a, b []byte) bool {
	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		return false
	}
	return reflect.DeepEqual(va, vb)
}

// AssertJSONEqual asserts that two JSON values are equal.
func AssertJSONEqual(t *testing.T, expected, actual []byte) {
	t.Helper()
	if !JSONEqual(expected, actual) {
		t.Errorf("JSON not equal:\nexpected: %s\nactual: %s", expected, actual)
	}
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" ||
		findSubstring(s, substr) >= 0)
}

// findSubstring finds a substring in a string.
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// CreateTestContext creates a context for testing.
// This is a placeholder for potential future enhancements like
// mock time or other test-specific context values.
func CreateTestContext() map[string]any {
	return make(map[string]any)
}
