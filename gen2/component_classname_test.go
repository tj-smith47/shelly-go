package gen2

import "testing"

func TestComponentClassName(t *testing.T) {
	cases := map[string]string{
		"switch":      "Switch",
		"cover":       "Cover",
		"light":       "Light",
		"input":       "Input",
		"wifi":        "WiFi",
		"ble":         "BLE",
		"ws":          "Ws",
		"sys":         "Sys",
		"em":          "EM",
		"em1":         "EM1",
		"pm1":         "PM1",
		"devicepower": "DevicePower",
		"eth":         "Eth",
		"":            "",
	}
	for in, want := range cases {
		if got := ComponentClassName(in); got != want {
			t.Errorf("ComponentClassName(%q) = %q, want %q", in, got, want)
		}
	}
}
