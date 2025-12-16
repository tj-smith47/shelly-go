package testutil

import (
	"errors"
	"testing"
)

// mockTB implements testing.TB for testing assertion failure branches.
type mockTB struct {
	testing.TB
	failed bool
}

func (m *mockTB) Helper()                        {}
func (m *mockTB) Errorf(string, ...any)          { m.failed = true }
func (m *mockTB) Error(...any)                   { m.failed = true }
func (m *mockTB) Fatalf(string, ...any)          { m.failed = true }
func (m *mockTB) Fatal(...any)                   { m.failed = true }
func (m *mockTB) Fail()                          { m.failed = true }
func (m *mockTB) FailNow()                       { m.failed = true }
func (m *mockTB) Failed() bool                   { return m.failed }
func (m *mockTB) Log(...any)                     {}
func (m *mockTB) Logf(string, ...any)            {}
func (m *mockTB) Name() string                   { return "mock" }
func (m *mockTB) Skip(...any)                    {}
func (m *mockTB) SkipNow()                       {}
func (m *mockTB) Skipf(string, ...any)           {}
func (m *mockTB) Skipped() bool                  { return false }
func (m *mockTB) TempDir() string                { return "" }
func (m *mockTB) Setenv(string, string)          {}
func (m *mockTB) Cleanup(func())                 {}

func newMockTB() *mockTB { return &mockTB{} }

// Fixture tests
func TestLoadFixture_Success(t *testing.T) {
	data, err := LoadFixture("gen2/switch_status.json")
	if err != nil {
		t.Fatalf("LoadFixture failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("LoadFixture returned empty data")
	}
}

func TestLoadFixture_MissingFile(t *testing.T) {
	_, err := LoadFixture("nonexistent.json")
	if err == nil {
		t.Error("LoadFixture should return error for missing file")
	}
}

func TestMustLoadFixture_Success(t *testing.T) {
	data := MustLoadFixture("gen2/switch_status.json")
	if len(data) == 0 {
		t.Error("MustLoadFixture returned empty data")
	}
}

func TestMustLoadFixture_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoadFixture should panic for missing file")
		}
	}()
	MustLoadFixture("nonexistent.json")
}

func TestLoadFixtureJSON_Success(t *testing.T) {
	var result map[string]any
	err := LoadFixtureJSON("gen2/switch_status.json", &result)
	if err != nil {
		t.Fatalf("LoadFixtureJSON failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("LoadFixtureJSON returned empty result")
	}
}

func TestLoadFixtureJSON_NotFound(t *testing.T) {
	var result map[string]any
	err := LoadFixtureJSON("nonexistent.json", &result)
	if err == nil {
		t.Error("LoadFixtureJSON should return error for missing file")
	}
}

// JSON helper tests
func TestMustJSON_Success(t *testing.T) {
	data := MustJSON(map[string]int{"a": 1})
	if string(data) != `{"a":1}` {
		t.Errorf("MustJSON returned unexpected data: %s", data)
	}
}

func TestMustJSON_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustJSON should panic for unmarshalable value")
		}
	}()
	// Channels cannot be marshaled to JSON
	MustJSON(make(chan int))
}

func TestMustJSONRaw_Success(t *testing.T) {
	data := MustJSONRaw(map[string]int{"a": 1})
	if string(data) != `{"a":1}` {
		t.Errorf("MustJSONRaw returned unexpected data: %s", data)
	}
}

func TestJSONEqual_Equal(t *testing.T) {
	if !JSONEqual([]byte(`{"a":1,"b":2}`), []byte(`{"b":2,"a":1}`)) {
		t.Error("JSONEqual should return true for equivalent JSON")
	}
}

func TestJSONEqual_NotEqual(t *testing.T) {
	if JSONEqual([]byte(`{"a":1}`), []byte(`{"a":2}`)) {
		t.Error("JSONEqual should return false for different JSON")
	}
}

func TestJSONEqual_InvalidFirst(t *testing.T) {
	if JSONEqual([]byte(`{invalid`), []byte(`{"a":1}`)) {
		t.Error("JSONEqual should return false for invalid first JSON")
	}
}

func TestJSONEqual_InvalidSecond(t *testing.T) {
	if JSONEqual([]byte(`{"a":1}`), []byte(`{invalid`)) {
		t.Error("JSONEqual should return false for invalid second JSON")
	}
}

// String helper tests
func TestContainsString_Found(t *testing.T) {
	if !containsString("hello world", "world") {
		t.Error("containsString should return true when substring exists")
	}
}

func TestContainsString_NotFound(t *testing.T) {
	if containsString("hello world", "foo") {
		t.Error("containsString should return false when substring doesn't exist")
	}
}

func TestContainsString_Empty(t *testing.T) {
	if !containsString("hello", "") {
		t.Error("containsString should return true for empty substring")
	}
}

func TestContainsString_LongerSubstring(t *testing.T) {
	if containsString("hi", "hello") {
		t.Error("containsString should return false when substring is longer")
	}
}

func TestFindSubstring_Found(t *testing.T) {
	idx := findSubstring("hello world", "world")
	if idx != 6 {
		t.Errorf("findSubstring expected 6, got %d", idx)
	}
}

func TestFindSubstring_NotFound(t *testing.T) {
	idx := findSubstring("hello", "xyz")
	if idx != -1 {
		t.Errorf("findSubstring expected -1, got %d", idx)
	}
}

func TestFindSubstring_AtStart(t *testing.T) {
	idx := findSubstring("hello world", "hello")
	if idx != 0 {
		t.Errorf("findSubstring expected 0, got %d", idx)
	}
}

func TestCreateTestContext_ReturnsMap(t *testing.T) {
	ctx := CreateTestContext()
	if ctx == nil {
		t.Error("CreateTestContext should return non-nil map")
	}
}

// AssertEqual tests
func TestAssertEqual_Pass(t *testing.T) {
	m := newMockTB()
	AssertEqual(m, 42, 42)
	if m.failed {
		t.Error("AssertEqual should not fail for equal values")
	}
}

func TestAssertEqual_Fail(t *testing.T) {
	m := newMockTB()
	AssertEqual(m, 42, 43)
	if !m.failed {
		t.Error("AssertEqual should fail for unequal values")
	}
}

// AssertNotEqual tests
func TestAssertNotEqual_Pass(t *testing.T) {
	m := newMockTB()
	AssertNotEqual(m, 42, 43)
	if m.failed {
		t.Error("AssertNotEqual should not fail for different values")
	}
}

func TestAssertNotEqual_Fail(t *testing.T) {
	m := newMockTB()
	AssertNotEqual(m, 42, 42)
	if !m.failed {
		t.Error("AssertNotEqual should fail for equal values")
	}
}

// AssertNil tests
func TestAssertNil_Pass(t *testing.T) {
	m := newMockTB()
	AssertNil(m, nil)
	if m.failed {
		t.Error("AssertNil should not fail for nil")
	}
}

func TestAssertNil_PassNilPointer(t *testing.T) {
	m := newMockTB()
	var nilPtr *string
	AssertNil(m, nilPtr)
	if m.failed {
		t.Error("AssertNil should not fail for nil pointer")
	}
}

func TestAssertNil_Fail(t *testing.T) {
	m := newMockTB()
	s := "not nil"
	AssertNil(m, &s)
	if !m.failed {
		t.Error("AssertNil should fail for non-nil pointer")
	}
}

// AssertNotNil tests
func TestAssertNotNil_Pass(t *testing.T) {
	m := newMockTB()
	s := "something"
	AssertNotNil(m, &s)
	if m.failed {
		t.Error("AssertNotNil should not fail for non-nil pointer")
	}
}

func TestAssertNotNil_FailNil(t *testing.T) {
	m := newMockTB()
	AssertNotNil(m, nil)
	if !m.failed {
		t.Error("AssertNotNil should fail for nil")
	}
}

func TestAssertNotNil_FailNilPointer(t *testing.T) {
	m := newMockTB()
	var nilPtr *string
	AssertNotNil(m, nilPtr)
	if !m.failed {
		t.Error("AssertNotNil should fail for nil pointer")
	}
}

// AssertNoError tests
func TestAssertNoError_Pass(t *testing.T) {
	m := newMockTB()
	AssertNoError(m, nil)
	if m.failed {
		t.Error("AssertNoError should not fail for nil error")
	}
}

func TestAssertNoError_Fail(t *testing.T) {
	m := newMockTB()
	AssertNoError(m, errors.New("some error"))
	if !m.failed {
		t.Error("AssertNoError should fail for non-nil error")
	}
}

// AssertError tests
func TestAssertError_Pass(t *testing.T) {
	m := newMockTB()
	AssertError(m, errors.New("some error"))
	if m.failed {
		t.Error("AssertError should not fail for non-nil error")
	}
}

func TestAssertError_Fail(t *testing.T) {
	m := newMockTB()
	AssertError(m, nil)
	if !m.failed {
		t.Error("AssertError should fail for nil error")
	}
}

// AssertErrorContains tests
func TestAssertErrorContains_Pass(t *testing.T) {
	m := newMockTB()
	AssertErrorContains(m, errors.New("connection failed"), "failed")
	if m.failed {
		t.Error("AssertErrorContains should not fail when error contains substring")
	}
}

func TestAssertErrorContains_FailNil(t *testing.T) {
	m := newMockTB()
	AssertErrorContains(m, nil, "failed")
	if !m.failed {
		t.Error("AssertErrorContains should fail for nil error")
	}
}

func TestAssertErrorContains_FailNoMatch(t *testing.T) {
	m := newMockTB()
	AssertErrorContains(m, errors.New("connection failed"), "timeout")
	if !m.failed {
		t.Error("AssertErrorContains should fail when error doesn't contain substring")
	}
}

// AssertTrue tests
func TestAssertTrue_Pass(t *testing.T) {
	m := newMockTB()
	AssertTrue(m, true)
	if m.failed {
		t.Error("AssertTrue should not fail for true")
	}
}

func TestAssertTrue_Fail(t *testing.T) {
	m := newMockTB()
	AssertTrue(m, false)
	if !m.failed {
		t.Error("AssertTrue should fail for false")
	}
}

// AssertFalse tests
func TestAssertFalse_Pass(t *testing.T) {
	m := newMockTB()
	AssertFalse(m, false)
	if m.failed {
		t.Error("AssertFalse should not fail for false")
	}
}

func TestAssertFalse_Fail(t *testing.T) {
	m := newMockTB()
	AssertFalse(m, true)
	if !m.failed {
		t.Error("AssertFalse should fail for true")
	}
}

// AssertLen tests
func TestAssertLen_Pass(t *testing.T) {
	m := newMockTB()
	AssertLen(m, []int{1, 2, 3}, 3)
	if m.failed {
		t.Error("AssertLen should not fail for correct length")
	}
}

func TestAssertLen_Fail(t *testing.T) {
	m := newMockTB()
	AssertLen(m, []int{1, 2, 3}, 5)
	if !m.failed {
		t.Error("AssertLen should fail for incorrect length")
	}
}

// AssertContains tests
func TestAssertContains_Pass(t *testing.T) {
	m := newMockTB()
	AssertContains(m, []string{"a", "b", "c"}, "b")
	if m.failed {
		t.Error("AssertContains should not fail when element exists")
	}
}

func TestAssertContains_Fail(t *testing.T) {
	m := newMockTB()
	AssertContains(m, []string{"a", "b", "c"}, "z")
	if !m.failed {
		t.Error("AssertContains should fail when element doesn't exist")
	}
}

// AssertStringContains tests
func TestAssertStringContains_Pass(t *testing.T) {
	m := newMockTB()
	AssertStringContains(m, "hello world", "world")
	if m.failed {
		t.Error("AssertStringContains should not fail when substring exists")
	}
}

func TestAssertStringContains_Fail(t *testing.T) {
	m := newMockTB()
	AssertStringContains(m, "hello world", "foo")
	if !m.failed {
		t.Error("AssertStringContains should fail when substring doesn't exist")
	}
}

// AssertJSONEqual tests
func TestAssertJSONEqual_Pass(t *testing.T) {
	m := newMockTB()
	AssertJSONEqual(m, []byte(`{"a":1}`), []byte(`{"a":1}`))
	if m.failed {
		t.Error("AssertJSONEqual should not fail for equal JSON")
	}
}

func TestAssertJSONEqual_Fail(t *testing.T) {
	m := newMockTB()
	AssertJSONEqual(m, []byte(`{"a":1}`), []byte(`{"a":2}`))
	if !m.failed {
		t.Error("AssertJSONEqual should fail for different JSON")
	}
}
