//go:build linux

package discovery

import (
	"testing"
)

func TestParseNmcliLine(t *testing.T) {
	s := &platformWiFiScanner{
		wifiscanScanner: &wifiscanScanner{},
		iface:           "wlan0",
	}

	tests := []struct {
		name       string
		line       string
		wantSSID   string
		wantSignal int
		wantNil    bool
	}{
		{
			name:    "not current network",
			line:    " :HomeNetwork:75:WPA2",
			wantNil: true,
		},
		{
			name:    "empty line",
			line:    "",
			wantNil: true,
		},
		{
			name:       "current network with all fields",
			line:       "*:MyNetwork:80:WPA2",
			wantSSID:   "MyNetwork",
			wantSignal: 80,
			wantNil:    false,
		},
		{
			name:       "current network SSID only",
			line:       "*:OnlySSID",
			wantSSID:   "OnlySSID",
			wantSignal: 0,
			wantNil:    false,
		},
		{
			name:       "current network with signal",
			line:       "*:NetworkName:65",
			wantSSID:   "NetworkName",
			wantSignal: 65,
			wantNil:    false,
		},
		{
			name:    "empty SSID after asterisk",
			line:    "*::80:WPA2",
			wantNil: true,
		},
		{
			name:    "only asterisk and colon",
			line:    "*:",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.parseNmcliLine(tt.line)
			if tt.wantNil {
				if result != nil {
					t.Errorf("parseNmcliLine(%q) = %+v, want nil", tt.line, result)
				}
				return
			}

			if result == nil {
				t.Fatalf("parseNmcliLine(%q) = nil, want non-nil", tt.line)
			}

			if result.SSID != tt.wantSSID {
				t.Errorf("parseNmcliLine(%q).SSID = %q, want %q", tt.line, result.SSID, tt.wantSSID)
			}

			if result.Signal != tt.wantSignal {
				t.Errorf("parseNmcliLine(%q).Signal = %d, want %d", tt.line, result.Signal, tt.wantSignal)
			}
		})
	}
}

func TestParseIntFromString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   int
		wantOK bool
	}{
		{
			name:   "simple number",
			input:  "123",
			want:   123,
			wantOK: true,
		},
		{
			name:   "number with trailing text",
			input:  "456abc",
			want:   456,
			wantOK: true,
		},
		{
			name:   "leading zeros",
			input:  "007",
			want:   7,
			wantOK: true,
		},
		{
			name:   "zero",
			input:  "0",
			want:   0,
			wantOK: true,
		},
		{
			name:   "only text",
			input:  "abc",
			want:   0,
			wantOK: true,
		},
		{
			name:   "empty string",
			input:  "",
			want:   0,
			wantOK: false, // EOF error expected
		},
		{
			name:   "negative prefix",
			input:  "-123",
			want:   0,
			wantOK: true,
		},
		{
			name:   "decimal number",
			input:  "12.34",
			want:   12,
			wantOK: true,
		},
		{
			name:   "large number",
			input:  "999999",
			want:   999999,
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result int
			ok, err := parseIntFromString(tt.input, &result)
			if !tt.wantOK {
				// Expecting an error or not-OK result
				if err == nil && ok {
					t.Errorf("parseIntFromString(%q) expected error or !ok, got ok=true, err=nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseIntFromString(%q) error = %v", tt.input, err)
				return
			}
			if result != tt.want {
				t.Errorf("parseIntFromString(%q) = %d, want %d", tt.input, result, tt.want)
			}
		})
	}
}

func TestDetectWiFiInterface(t *testing.T) {
	// This test just verifies the function doesn't panic
	// and returns a non-empty string (even if the default fallback)
	iface := detectWiFiInterface()
	if iface == "" {
		t.Error("detectWiFiInterface() returned empty string")
	}
}

func TestHasCommand(t *testing.T) {
	// Test with a command that should exist on any Linux system
	if !hasCommand("ls") {
		t.Skip("ls command not found, skipping")
	}

	// Test with a command that definitely doesn't exist
	if hasCommand("nonexistent-command-12345") {
		t.Error("hasCommand returned true for nonexistent command")
	}
}
