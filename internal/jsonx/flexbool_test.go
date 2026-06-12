package jsonx

import (
	"encoding/json"
	"testing"
)

func TestFlexBool_UnmarshalAcceptsLooseEncodings(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"native true", `true`, true},
		{"native false", `false`, false},
		{"string true (old Gen1 firmware)", `"true"`, true},
		{"string false", `"false"`, false},
		{"string one", `"1"`, true},
		{"string zero", `"0"`, false},
		{"number one", `1`, true},
		{"number zero", `0`, false},
		{"null is false", `null`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b FlexBool
			if err := json.Unmarshal([]byte(tt.in), &b); err != nil {
				t.Fatalf("Unmarshal(%s) error: %v", tt.in, err)
			}
			if b.Bool() != tt.want {
				t.Errorf("Unmarshal(%s) = %v, want %v", tt.in, b.Bool(), tt.want)
			}
		})
	}
}

// TestFlexBool_InStruct is the Bug F regression: an is_valid field arriving as a
// quoted string (as an old Bulb Duo reports it) must decode rather than fail the
// whole status unmarshal.
func TestFlexBool_InStruct(t *testing.T) {
	type meter struct {
		IsValid FlexBool `json:"is_valid"`
	}
	var m meter
	if err := json.Unmarshal([]byte(`{"is_valid":"true"}`), &m); err != nil {
		t.Fatalf("unmarshal struct with string is_valid: %v", err)
	}
	if !m.IsValid {
		t.Errorf("IsValid = %v, want true", m.IsValid)
	}
}

func TestFlexBool_MarshalRoundTrip(t *testing.T) {
	out, err := json.Marshal(FlexBool(true))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != "true" {
		t.Errorf("Marshal(FlexBool(true)) = %s, want true", out)
	}
}

func TestFlexBool_RejectsGarbage(t *testing.T) {
	var b FlexBool
	if err := json.Unmarshal([]byte(`"notabool"`), &b); err == nil {
		t.Error("expected error for non-boolean string, got nil")
	}
}
