package reprovision

import (
	"net"
	"reflect"
	"testing"
)

func mustCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("ParseCIDR(%q): %v", cidr, err)
	}
	return n
}

func TestSelectProbeBindInterfaces(t *testing.T) {
	t.Parallel()
	target := net.ParseIP("10.23.47.227")
	tests := []struct {
		name   string
		target net.IP
		ifaces []probeIface
		want   []string
	}{
		{
			name:   "default route first, then wired, then wireless",
			target: target,
			ifaces: []probeIface{
				{Name: "wlp6s0", IsWireless: true, Nets: []*net.IPNet{mustCIDR(t, "10.23.47.10/23")}},
				{Name: "vmbr0", Nets: []*net.IPNet{mustCIDR(t, "10.23.47.11/23")}},
			},
			want: []string{"", "vmbr0", "wlp6s0"},
		},
		{
			name:   "wireless only",
			target: target,
			ifaces: []probeIface{{Name: "wlan0", IsWireless: true, Nets: []*net.IPNet{mustCIDR(t, "10.23.47.10/23")}}},
			want:   []string{"", "wlan0"},
		},
		{
			name:   "wired only",
			target: target,
			ifaces: []probeIface{{Name: "eth0", Nets: []*net.IPNet{mustCIDR(t, "10.23.47.10/23")}}},
			want:   []string{"", "eth0"},
		},
		{
			name:   "no same-subnet interface",
			target: target,
			ifaces: []probeIface{{Name: "eth0", Nets: []*net.IPNet{mustCIDR(t, "192.168.1.10/24")}}},
			want:   []string{""},
		},
		{
			name:   "nil target falls back to default route",
			ifaces: []probeIface{{Name: "eth0", Nets: []*net.IPNet{mustCIDR(t, "10.23.47.10/23")}}},
			want:   []string{""},
		},
		{name: "no interfaces", target: target, want: []string{""}},
		{
			name:   "multiple wired preserve order, deduped",
			target: target,
			ifaces: []probeIface{
				{Name: "eth0", Nets: []*net.IPNet{mustCIDR(t, "10.23.47.10/23")}},
				{Name: "eth0", Nets: []*net.IPNet{mustCIDR(t, "10.23.47.10/23")}},
				{Name: "eth1", Nets: []*net.IPNet{mustCIDR(t, "10.23.46.5/23")}},
			},
			want: []string{"", "eth0", "eth1"},
		},
		{
			name:   "address family must match",
			target: target,
			ifaces: []probeIface{{Name: "eth0", Nets: []*net.IPNet{mustCIDR(t, "fd00::1/64")}}},
			want:   []string{""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := selectProbeBindInterfaces(tt.target, tt.ifaces); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("selectProbeBindInterfaces() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectProbeBindInterfaces_AlwaysOffersDefaultRouteFirst(t *testing.T) {
	t.Parallel()
	cases := [][]probeIface{
		nil,
		{{Name: "eth0", Nets: []*net.IPNet{mustCIDR(t, "10.23.47.1/23")}}},
		{{Name: "wlan0", IsWireless: true, Nets: []*net.IPNet{mustCIDR(t, "10.23.47.1/23")}}},
	}
	for _, ifaces := range cases {
		got := selectProbeBindInterfaces(net.ParseIP("10.23.47.227"), ifaces)
		if len(got) == 0 || got[0] != "" {
			t.Errorf("candidates %v must begin with the default route", got)
		}
	}
}

// TestHostProbeIfaces only reads the host's interface list; it changes nothing.
func TestHostProbeIfaces(t *testing.T) {
	t.Parallel()
	ifaces, err := hostProbeIfaces()
	if err != nil {
		t.Fatalf("hostProbeIfaces: %v", err)
	}
	for _, ifc := range ifaces {
		if ifc.Name == "" || len(ifc.Nets) == 0 {
			t.Errorf("interface %+v has no name or no addresses", ifc)
		}
	}
	if interfaceIsWireless("definitely-not-an-iface") {
		t.Error("an unknown interface reported as wireless")
	}
}

// TestNewHTTP_BindsInterface builds a bound transport without sending anything.
func TestNewHTTP_BindsInterface(t *testing.T) {
	t.Parallel()
	if newHTTP("192.0.2.10", "lo") == nil || newHTTP("192.0.2.10", "") == nil {
		t.Error("newHTTP returned nil")
	}
}
