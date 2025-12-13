package components

import "testing"

func TestEMDataCSVURL(t *testing.T) {
	tests := []struct {
		name       string
		deviceAddr string
		id         int
		startTS    *int64
		endTS      *int64
		addKeys    bool
		wantURL    string
	}{
		{
			name:       "all parameters",
			deviceAddr: "192.168.1.100",
			id:         0,
			startTS:    ptr(int64(1656356400)),
			endTS:      ptr(int64(1656442800)),
			addKeys:    true,
			wantURL:    "http://192.168.1.100/emdata/0/data.csv?ts=1656356400&end_ts=1656442800&add_keys=true",
		},
		{
			name:       "no timestamps",
			deviceAddr: "192.168.1.100",
			id:         0,
			startTS:    nil,
			endTS:      nil,
			addKeys:    true,
			wantURL:    "http://192.168.1.100/emdata/0/data.csv?add_keys=true",
		},
		{
			name:       "start timestamp only",
			deviceAddr: "192.168.1.100",
			id:         0,
			startTS:    ptr(int64(1656356400)),
			endTS:      nil,
			addKeys:    false,
			wantURL:    "http://192.168.1.100/emdata/0/data.csv?ts=1656356400",
		},
		{
			name:       "different component ID",
			deviceAddr: "10.0.0.50",
			id:         2,
			startTS:    nil,
			endTS:      nil,
			addKeys:    false,
			wantURL:    "http://10.0.0.50/emdata/2/data.csv?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := EMDataCSVURL(tt.deviceAddr, tt.id, tt.startTS, tt.endTS, tt.addKeys)
			if url != tt.wantURL {
				t.Errorf("EMDataCSVURL() = %q, want %q", url, tt.wantURL)
			}
		})
	}
}

func TestEM1DataCSVURL(t *testing.T) {
	tests := []struct {
		name       string
		deviceAddr string
		id         int
		startTS    *int64
		endTS      *int64
		addKeys    bool
		wantURL    string
	}{
		{
			name:       "all parameters",
			deviceAddr: "192.168.1.100",
			id:         0,
			startTS:    ptr(int64(1656356400)),
			endTS:      ptr(int64(1656442800)),
			addKeys:    true,
			wantURL:    "http://192.168.1.100/em1data/0/data.csv?ts=1656356400&end_ts=1656442800&add_keys=true",
		},
		{
			name:       "no timestamps",
			deviceAddr: "192.168.1.100",
			id:         0,
			startTS:    nil,
			endTS:      nil,
			addKeys:    true,
			wantURL:    "http://192.168.1.100/em1data/0/data.csv?add_keys=true",
		},
		{
			name:       "end timestamp only",
			deviceAddr: "192.168.1.100",
			id:         1,
			startTS:    nil,
			endTS:      ptr(int64(1656442800)),
			addKeys:    false,
			wantURL:    "http://192.168.1.100/em1data/1/data.csv?end_ts=1656442800",
		},
		{
			name:       "hostname instead of IP",
			deviceAddr: "shelly-em.local",
			id:         0,
			startTS:    nil,
			endTS:      nil,
			addKeys:    false,
			wantURL:    "http://shelly-em.local/em1data/0/data.csv?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := EM1DataCSVURL(tt.deviceAddr, tt.id, tt.startTS, tt.endTS, tt.addKeys)
			if url != tt.wantURL {
				t.Errorf("EM1DataCSVURL() = %q, want %q", url, tt.wantURL)
			}
		})
	}
}

func TestBuildDataCSVURL(t *testing.T) {
	tests := []struct {
		name          string
		componentType string
		deviceAddr    string
		id            int
		startTS       *int64
		endTS         *int64
		addKeys       bool
		wantURL       string
	}{
		{
			name:          "custom component type",
			componentType: "pmdata",
			deviceAddr:    "192.168.1.100",
			id:            0,
			startTS:       ptr(int64(1234567890)),
			endTS:         nil,
			addKeys:       true,
			wantURL:       "http://192.168.1.100/pmdata/0/data.csv?ts=1234567890&add_keys=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := buildDataCSVURL(tt.componentType, tt.deviceAddr, tt.id, tt.startTS, tt.endTS, tt.addKeys)
			if url != tt.wantURL {
				t.Errorf("buildDataCSVURL() = %q, want %q", url, tt.wantURL)
			}
		})
	}
}
