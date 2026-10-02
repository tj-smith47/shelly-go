package backup

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/tj-smith47/shelly-go/types"
)

func TestResolveStaticNetwork(t *testing.T) {
	t.Parallel()
	bkpStatic := StaticNetwork{IP: "10.0.0.5", Gateway: "10.0.0.1", Netmask: "255.255.255.0", DNS: "10.0.0.2"}
	tests := []struct {
		name       string
		override   StaticNetwork
		fromBackup StaticNetwork
		want       StaticNetwork
		wantErr    bool
	}{
		{name: "no override keeps the backup", fromBackup: bkpStatic, want: bkpStatic},
		{name: "dhcp everywhere", want: StaticNetwork{}},
		{
			name:       "ip only takes each field from the backup",
			override:   StaticNetwork{IP: "10.0.0.9"},
			fromBackup: bkpStatic,
			want:       StaticNetwork{IP: "10.0.0.9", Gateway: "10.0.0.1", Netmask: "255.255.255.0", DNS: "10.0.0.2"},
		},
		{
			name:       "fields fall back one at a time",
			override:   StaticNetwork{IP: "10.0.0.9", Gateway: "10.0.0.254"},
			fromBackup: bkpStatic,
			want:       StaticNetwork{IP: "10.0.0.9", Gateway: "10.0.0.254", Netmask: "255.255.255.0", DNS: "10.0.0.2"},
		},
		{name: "ip with dhcp backup is refused", override: StaticNetwork{IP: "10.0.0.9"}, wantErr: true},
		{
			name:     "ip without netmask is refused",
			override: StaticNetwork{IP: "10.0.0.9", Gateway: "10.0.0.1"},
			wantErr:  true,
		},
		{
			name:       "incomplete backup static is refused",
			fromBackup: StaticNetwork{IP: "10.0.0.5", Netmask: "255.255.255.0"},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ResolveStaticNetwork(tt.override, tt.fromBackup)
			if tt.wantErr {
				if !errors.Is(err, ErrIncompleteStaticNetwork) || !errors.Is(err, types.ErrInvalidParam) {
					t.Fatalf("err = %v, want ErrIncompleteStaticNetwork and ErrInvalidParam", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// staticGen1Backup is a Gen1 backup whose primary station is static.
func staticGen1Backup() *Backup {
	return &Backup{
		WiFi: json.RawMessage(`{"sta":{"enabled":true,"ssid":"Home","ipv4_method":"static",` +
			`"ip":"10.0.0.5","gw":"10.0.0.1","mask":"255.255.255.0","dns":"10.0.0.2"}}`),
	}
}

func staWrite(t *testing.T, writes []string) string {
	t.Helper()
	for _, w := range writes {
		if strings.HasPrefix(w, "/settings/sta?") {
			return w
		}
	}
	t.Fatalf("no station write; writes=%v", writes)
	return ""
}

func TestRestoreGen1_StaticOverrideTakesBackupFields(t *testing.T) {
	t.Parallel()
	dev, writes := gen1ColorDevice(t, `{}`)
	_, err := RestoreGen1(context.Background(), dev, staticGen1Backup(), &Gen1RestoreOptions{
		NetworkOnly:     true,
		NetworkOverride: &Gen1NetworkOverride{StaticIP: "10.0.0.9"},
	})
	if err != nil {
		t.Fatalf("RestoreGen1: %v", err)
	}
	w := staWrite(t, *writes)
	for _, want := range []string{"ip=10.0.0.9", "gateway=10.0.0.1", "netmask=255.255.255.0", "dns=10.0.0.2"} {
		if !strings.Contains(w, want) {
			t.Errorf("station write %q lacks %q", w, want)
		}
	}
}

func TestRestoreGen1_IncompleteStaticOverrideRefusedBeforeWriting(t *testing.T) {
	t.Parallel()
	dev, writes := gen1ColorDevice(t, `{}`)
	bkp := &Backup{WiFi: json.RawMessage(`{"sta":{"enabled":true,"ssid":"Home","ipv4_method":"dhcp"}}`)}
	_, err := RestoreGen1(context.Background(), dev, bkp, &Gen1RestoreOptions{
		NetworkOverride: &Gen1NetworkOverride{StaticIP: "10.0.0.9"},
	})
	if !errors.Is(err, ErrIncompleteStaticNetwork) || !errors.Is(err, types.ErrInvalidParam) {
		t.Fatalf("err = %v, want ErrIncompleteStaticNetwork and ErrInvalidParam", err)
	}
	if len(*writes) != 0 {
		t.Errorf("device was called before the refusal: %v", *writes)
	}
}

func TestRestoreGen1_SkipNetworkIgnoresOverride(t *testing.T) {
	t.Parallel()
	dev, _ := gen1ColorDevice(t, `{}`)
	_, err := RestoreGen1(context.Background(), dev, &Backup{}, &Gen1RestoreOptions{
		SkipNetwork:        true,
		ClockDependentOnly: true,
		NetworkOverride:    &Gen1NetworkOverride{StaticIP: "10.0.0.9"},
	})
	if err != nil {
		t.Fatalf("RestoreGen1 with SkipNetwork: %v", err)
	}
}

func TestRestoreGen1_OpenOverride(t *testing.T) {
	t.Parallel()
	t.Run("dhcp clears the key", func(t *testing.T) {
		t.Parallel()
		dev, writes := gen1ColorDevice(t, `{}`)
		bkp := &Backup{WiFi: json.RawMessage(`{"sta":{"enabled":true,"ssid":"Home","key":"old"}}`)}
		_, err := RestoreGen1(context.Background(), dev, bkp, &Gen1RestoreOptions{
			NetworkOnly:     true,
			NetworkOverride: &Gen1NetworkOverride{SSID: "Guest", Open: true},
		})
		if err != nil {
			t.Fatalf("RestoreGen1: %v", err)
		}
		w := staWrite(t, *writes)
		if !strings.Contains(w, "ssid=Guest") || !strings.Contains(w, "key=&") {
			t.Errorf("open station write %q must name Guest with an empty key", w)
		}
	})
	t.Run("static sends an empty key", func(t *testing.T) {
		t.Parallel()
		dev, writes := gen1ColorDevice(t, `{}`)
		_, err := RestoreGen1(context.Background(), dev, staticGen1Backup(), &Gen1RestoreOptions{
			NetworkOnly:     true,
			NetworkOverride: &Gen1NetworkOverride{Open: true, StaticIP: "10.0.0.9"},
		})
		if err != nil {
			t.Fatalf("RestoreGen1: %v", err)
		}
		w := staWrite(t, *writes)
		if !strings.Contains(w, "key=&") || !strings.Contains(w, "ipv4_method=static") {
			t.Errorf("open static write %q must carry an empty key", w)
		}
	})
	t.Run("static without open leaves the key out", func(t *testing.T) {
		t.Parallel()
		dev, writes := gen1ColorDevice(t, `{}`)
		_, err := RestoreGen1(context.Background(), dev, staticGen1Backup(), &Gen1RestoreOptions{
			NetworkOnly:     true,
			NetworkOverride: &Gen1NetworkOverride{StaticIP: "10.0.0.9"},
		})
		if err != nil {
			t.Fatalf("RestoreGen1: %v", err)
		}
		w := staWrite(t, *writes)
		if strings.Contains(w, "key=") || !strings.Contains(w, "ipv4_method=static") {
			t.Errorf("static write %q must not carry a key the backup does not hold", w)
		}
	})
	t.Run("password is refused", func(t *testing.T) {
		t.Parallel()
		dev, writes := gen1ColorDevice(t, `{}`)
		_, err := RestoreGen1(context.Background(), dev, &Backup{}, &Gen1RestoreOptions{
			NetworkOverride: &Gen1NetworkOverride{SSID: "Guest", Open: true, Password: "x"},
		})
		if !errors.Is(err, types.ErrInvalidParam) {
			t.Fatalf("err = %v, want ErrInvalidParam", err)
		}
		if len(*writes) != 0 {
			t.Errorf("device was called before the refusal: %v", *writes)
		}
	})
}

func TestResolveStaticNetwork_ErrorNamesTheCause(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                 string
		override, fromBackup StaticNetwork
		want                 string
	}{
		{
			name:     "dhcp backup",
			override: StaticNetwork{IP: "10.0.0.9"},
			want:     "the backup's station uses DHCP",
		},
		{
			name:       "static backup without a netmask",
			override:   StaticNetwork{IP: "10.0.0.9"},
			fromBackup: StaticNetwork{IP: "10.0.0.5", Gateway: "10.0.0.1"},
			want:       "the backup's static settings lack them",
		},
		{
			name:       "incomplete backup and no override",
			fromBackup: StaticNetwork{IP: "10.0.0.5", Gateway: "10.0.0.1"},
			want:       "the backup's static address 10.0.0.5 has no gateway or no netmask",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ResolveStaticNetwork(tt.override, tt.fromBackup)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want it to say %q", err, tt.want)
			}
		})
	}
}

func TestBackupStation(t *testing.T) {
	t.Parallel()
	const configSta = `{"wifi_sta":{"ssid":"FromConfig","ipv4_method":"dhcp"}}`
	tests := []struct {
		name string
		bkp  *Backup
		want string
	}{
		{
			name: "blob station with an SSID",
			bkp: &Backup{
				WiFi: json.RawMessage(`{"sta":{"ssid":"FromBlob"}}`), Config: json.RawMessage(configSta),
			},
			want: "FromBlob",
		},
		{
			name: "blob station with an empty SSID falls back to the settings",
			bkp: &Backup{
				WiFi: json.RawMessage(`{"sta":{"ssid":""}}`), Config: json.RawMessage(configSta),
			},
			want: "FromConfig",
		},
		{
			name: "gen2 settings are not read",
			bkp: &Backup{
				DeviceInfo: &DeviceInfo{Generation: 2}, Config: json.RawMessage(configSta),
			},
		},
		{name: "settings station without an SSID", bkp: &Backup{Config: json.RawMessage(`{"wifi_sta":{"ssid":null}}`)}},
		{name: "nothing", bkp: &Backup{}},
		{name: "nil backup"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var sta struct {
				SSID string `json:"ssid"`
			}
			if raw := tt.bkp.Station(); raw != nil {
				if err := json.Unmarshal(raw, &sta); err != nil {
					t.Fatalf("decode station: %v", err)
				}
			}
			if sta.SSID != tt.want {
				t.Errorf("station SSID = %q, want %q", sta.SSID, tt.want)
			}
		})
	}
}

func TestRestoreGen1_StaticFromSettingsWhenBlobNamesNoSSID(t *testing.T) {
	t.Parallel()
	dev, writes := gen1ColorDevice(t, `{}`)
	bkp := &Backup{
		WiFi: json.RawMessage(`{"sta":{"enabled":true,"ssid":"","ipv4_method":"dhcp"}}`),
		Config: json.RawMessage(`{"wifi_sta":{"enabled":true,"ssid":"Home","ipv4_method":"static",` +
			`"ip":"10.0.0.5","gw":"10.0.0.1","mask":"255.255.255.0"}}`),
	}
	_, err := RestoreGen1(context.Background(), dev, bkp, &Gen1RestoreOptions{
		NetworkOnly:     true,
		NetworkOverride: &Gen1NetworkOverride{SSID: "Home", Password: "pw", StaticIP: "10.0.0.9"},
	})
	if err != nil {
		t.Fatalf("RestoreGen1: %v", err)
	}
	if w := staWrite(t, *writes); !strings.Contains(w, "gateway=10.0.0.1") || !strings.Contains(w, "netmask=255.255.255.0") {
		t.Errorf("station write %q, want the settings' gateway and netmask", w)
	}
}

func TestRestoreGen1_OpenSSID(t *testing.T) {
	t.Parallel()
	t.Run("no SSID anywhere is refused", func(t *testing.T) {
		t.Parallel()
		dev, writes := gen1ColorDevice(t, `{}`)
		_, err := RestoreGen1(context.Background(), dev, &Backup{}, &Gen1RestoreOptions{
			NetworkOnly:     true,
			NetworkOverride: &Gen1NetworkOverride{Open: true},
		})
		if !errors.Is(err, types.ErrInvalidParam) || !strings.Contains(err.Error(), "Gen1NetworkOverride.SSID") {
			t.Fatalf("err = %v, want ErrInvalidParam naming Gen1NetworkOverride.SSID", err)
		}
		if len(*writes) != 0 {
			t.Errorf("device was called before the refusal: %v", *writes)
		}
	})
	t.Run("SSID from the settings", func(t *testing.T) {
		t.Parallel()
		dev, writes := gen1ColorDevice(t, `{}`)
		bkp := &Backup{Config: json.RawMessage(`{"wifi_sta":{"enabled":true,"ssid":"Guest","ipv4_method":"dhcp"}}`)}
		_, err := RestoreGen1(context.Background(), dev, bkp, &Gen1RestoreOptions{
			NetworkOnly:     true,
			NetworkOverride: &Gen1NetworkOverride{Open: true},
		})
		if err != nil {
			t.Fatalf("RestoreGen1: %v", err)
		}
		if w := staWrite(t, *writes); !strings.Contains(w, "ssid=Guest") || !strings.Contains(w, "key=&") {
			t.Errorf("station write %q, want Guest with an empty key", w)
		}
	})
	t.Run("incomplete static names the override fields", func(t *testing.T) {
		t.Parallel()
		dev, _ := gen1ColorDevice(t, `{}`)
		_, err := RestoreGen1(context.Background(), dev, &Backup{}, &Gen1RestoreOptions{
			NetworkOverride: &Gen1NetworkOverride{StaticIP: "10.0.0.9"},
		})
		if !errors.Is(err, ErrIncompleteStaticNetwork) || !strings.Contains(err.Error(), "Gen1NetworkOverride.Netmask") {
			t.Fatalf("err = %v, want ErrIncompleteStaticNetwork naming Gen1NetworkOverride.Netmask", err)
		}
	})
}
