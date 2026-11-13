//go:build linux

package discovery

import (
	"errors"
	"testing"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/wifi"
	"golang.org/x/sys/unix"
)

// The scan request must carry exactly one zero-length SSID. A single NUL byte
// there is a probe for one network, which makes every scan return a random
// fraction of what is in range.
func TestScanRequestAsksEveryAccessPoint(t *testing.T) {
	data, err := scanRequest(7)
	if err != nil {
		t.Fatalf("scanRequest: %v", err)
	}
	attrs, err := netlink.UnmarshalAttributes(data)
	if err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	var ssids [][]byte
	ifIndex := -1
	for _, a := range attrs {
		switch a.Type &^ (unix.NLA_F_NESTED | unix.NLA_F_NET_BYTEORDER) {
		case unix.NL80211_ATTR_IFINDEX:
			ifIndex = int(netlinkUint32(t, a.Data))
		case unix.NL80211_ATTR_SCAN_SSIDS:
			nested, nerr := netlink.UnmarshalAttributes(a.Data)
			if nerr != nil {
				t.Fatalf("unmarshal SSID list: %v", nerr)
			}
			for _, n := range nested {
				ssids = append(ssids, n.Data)
			}
		}
	}
	if ifIndex != 7 {
		t.Errorf("interface index = %d, want 7", ifIndex)
	}
	if len(ssids) != 1 || len(ssids[0]) != 0 {
		t.Errorf("SSID list = %q, want one zero-length entry", ssids)
	}
}

func TestScanRequestRejectsBadInterfaceIndex(t *testing.T) {
	if _, err := scanRequest(-1); err == nil {
		t.Fatal("scanRequest(-1) succeeded, want an error")
	}
}

func TestScanWatch(t *testing.T) {
	onIface := func(index uint32) []byte {
		enc := netlink.NewAttributeEncoder()
		enc.Uint32(unix.NL80211_ATTR_IFINDEX, index)
		data, err := enc.Encode()
		if err != nil {
			t.Fatalf("encode event: %v", err)
		}
		return data
	}
	type event struct {
		cmd   uint8
		iface uint32
	}
	tests := []struct {
		name     string
		events   []event
		wantDone bool
		wantErr  error
	}{
		{
			name:   "results of an earlier scan do not end this one",
			events: []event{{unix.NL80211_CMD_NEW_SCAN_RESULTS, 3}},
		},
		{
			name: "results after the start notification end the scan",
			events: []event{
				{unix.NL80211_CMD_NEW_SCAN_RESULTS, 3},
				{unix.NL80211_CMD_TRIGGER_SCAN, 3},
				{unix.NL80211_CMD_NEW_SCAN_RESULTS, 3},
			},
			wantDone: true,
		},
		{
			name: "another interface's scan is ignored",
			events: []event{
				{unix.NL80211_CMD_TRIGGER_SCAN, 4},
				{unix.NL80211_CMD_NEW_SCAN_RESULTS, 4},
			},
		},
		{
			name: "an aborted scan is an error",
			events: []event{
				{unix.NL80211_CMD_TRIGGER_SCAN, 3},
				{unix.NL80211_CMD_SCAN_ABORTED, 3},
			},
			wantDone: true,
			wantErr:  wifi.ErrScanAborted,
		},
		{
			name:   "an earlier scan's abort is ignored",
			events: []event{{unix.NL80211_CMD_SCAN_ABORTED, 3}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := scanWatch{ifIndex: 3}
			var done bool
			var err error
			for _, ev := range tt.events {
				done, err = w.done(ev.cmd, onIface(ev.iface))
				if done || err != nil {
					break
				}
			}
			if done != tt.wantDone || !errors.Is(err, tt.wantErr) {
				t.Errorf("done = %v, err = %v; want done = %v, err = %v", done, err, tt.wantDone, tt.wantErr)
			}
		})
	}
}

func TestScanWatchIgnoresUnreadableEvent(t *testing.T) {
	w := scanWatch{ifIndex: 3, started: true}
	if done, err := w.done(unix.NL80211_CMD_NEW_SCAN_RESULTS, []byte{0xff}); done || err != nil {
		t.Errorf("done = %v, err = %v; want the event ignored", done, err)
	}
}

func netlinkUint32(t *testing.T, b []byte) uint32 {
	t.Helper()
	ad, err := netlink.NewAttributeDecoder(append([]byte{8, 0, 1, 0}, b...))
	if err != nil || !ad.Next() {
		t.Fatalf("decode uint32: %v", err)
	}
	return ad.Uint32()
}
