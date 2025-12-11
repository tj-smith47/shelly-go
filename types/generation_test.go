package types

import (
	"encoding/json"
	"testing"
)

func TestGeneration_String(t *testing.T) {
	tests := []struct {
		name string
		want string
		gen  Generation
	}{
		{
			name: "Generation1",
			gen:  Generation1,
			want: "Gen1",
		},
		{
			name: "Generation2",
			gen:  Generation2,
			want: "Gen2",
		},
		{
			name: "Generation3",
			gen:  Generation3,
			want: "Gen3",
		},
		{
			name: "Generation4",
			gen:  Generation4,
			want: "Gen4",
		},
		{
			name: "GenerationUnknown",
			gen:  GenerationUnknown,
			want: "Unknown",
		},
		{
			name: "invalid generation",
			gen:  Generation(99),
			want: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gen.String(); got != tt.want {
				t.Errorf("Generation.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeneration_IsRPC(t *testing.T) {
	tests := []struct {
		name string
		gen  Generation
		want bool
	}{
		{
			name: "Gen1 is not RPC",
			gen:  Generation1,
			want: false,
		},
		{
			name: "Gen2 is RPC",
			gen:  Generation2,
			want: true,
		},
		{
			name: "Gen3 is RPC",
			gen:  Generation3,
			want: true,
		},
		{
			name: "Gen4 is RPC",
			gen:  Generation4,
			want: true,
		},
		{
			name: "Unknown is not RPC",
			gen:  GenerationUnknown,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gen.IsRPC(); got != tt.want {
				t.Errorf("Generation.IsRPC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeneration_IsREST(t *testing.T) {
	tests := []struct {
		name string
		gen  Generation
		want bool
	}{
		{
			name: "Gen1 is REST",
			gen:  Generation1,
			want: true,
		},
		{
			name: "Gen2 is not REST",
			gen:  Generation2,
			want: false,
		},
		{
			name: "Gen3 is not REST",
			gen:  Generation3,
			want: false,
		},
		{
			name: "Gen4 is not REST",
			gen:  Generation4,
			want: false,
		},
		{
			name: "Unknown is not REST",
			gen:  GenerationUnknown,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gen.IsREST(); got != tt.want {
				t.Errorf("Generation.IsREST() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeneration_MarshalText(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		gen     Generation
		wantErr bool
	}{
		{
			name: "Gen1",
			gen:  Generation1,
			want: "Gen1",
		},
		{
			name: "Gen2",
			gen:  Generation2,
			want: "Gen2",
		},
		{
			name: "Gen3",
			gen:  Generation3,
			want: "Gen3",
		},
		{
			name: "Gen4",
			gen:  Generation4,
			want: "Gen4",
		},
		{
			name: "Unknown",
			gen:  GenerationUnknown,
			want: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.gen.MarshalText()
			if (err != nil) != tt.wantErr {
				t.Errorf("Generation.MarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("Generation.MarshalText() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestGeneration_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    Generation
		wantErr bool
	}{
		{
			name: "Gen1",
			text: "Gen1",
			want: Generation1,
		},
		{
			name: "gen1 lowercase",
			text: "gen1",
			want: Generation1,
		},
		{
			name: "1 numeric",
			text: "1",
			want: Generation1,
		},
		{
			name: "Gen2",
			text: "Gen2",
			want: Generation2,
		},
		{
			name: "Plus variant",
			text: "Plus",
			want: Generation2,
		},
		{
			name: "Pro variant",
			text: "Pro",
			want: Generation2,
		},
		{
			name: "Gen3",
			text: "Gen3",
			want: Generation3,
		},
		{
			name: "Gen4",
			text: "Gen4",
			want: Generation4,
		},
		{
			name: "Unknown",
			text: "Unknown",
			want: GenerationUnknown,
		},
		{
			name: "empty string",
			text: "",
			want: GenerationUnknown,
		},
		{
			name:    "invalid",
			text:    "GenX",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Generation
			err := got.UnmarshalText([]byte(tt.text))
			if (err != nil) != tt.wantErr {
				t.Errorf("Generation.UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Generation.UnmarshalText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeneration_JSON(t *testing.T) {
	type testStruct struct {
		Gen Generation `json:"generation"`
	}

	tests := []struct {
		name    string
		input   string
		want    Generation
		wantErr bool
	}{
		{
			name:  "Gen1 from JSON",
			input: `{"generation":"Gen1"}`,
			want:  Generation1,
		},
		{
			name:  "Gen2 from JSON",
			input: `{"generation":"Gen2"}`,
			want:  Generation2,
		},
		{
			name:  "numeric from JSON",
			input: `{"generation":"2"}`,
			want:  Generation2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s testStruct
			err := json.Unmarshal([]byte(tt.input), &s)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && s.Gen != tt.want {
				t.Errorf("Generation from JSON = %v, want %v", s.Gen, tt.want)
			}
		})
	}

	// Test marshaling
	t.Run("Marshal to JSON", func(t *testing.T) {
		s := testStruct{Gen: Generation2}
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		want := `{"generation":"Gen2"}`
		if string(data) != want {
			t.Errorf("json.Marshal() = %v, want %v", string(data), want)
		}
	})
}
