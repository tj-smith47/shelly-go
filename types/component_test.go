package types

import "testing"

func TestComponentType_String(t *testing.T) {
	tests := []struct {
		name string
		ct   ComponentType
		want string
	}{
		{
			name: "switch",
			ct:   ComponentTypeSwitch,
			want: "switch",
		},
		{
			name: "cover",
			ct:   ComponentTypeCover,
			want: "cover",
		},
		{
			name: "light",
			ct:   ComponentTypeLight,
			want: "light",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ct.String(); got != tt.want {
				t.Errorf("ComponentType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseComponentID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "switch:0",
			input: "switch:0",
			want:  "switch:0",
		},
		{
			name:  "cover:1",
			input: "cover:1",
			want:  "cover:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseComponentID(tt.input)
			if got.String() != tt.want {
				t.Errorf("ParseComponentID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComponentID_String(t *testing.T) {
	cid := ComponentID("light:2")
	want := "light:2"
	if got := cid.String(); got != want {
		t.Errorf("ComponentID.String() = %v, want %v", got, want)
	}
}
