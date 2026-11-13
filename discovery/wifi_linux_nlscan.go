//go:build linux

package discovery

import (
	"context"
	"fmt"
	"math"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/wifi"
	"golang.org/x/sys/unix"
)

// nl80211Scan runs one active scan on the WiFi interface with index ifIndex
// and returns once the kernel has published that scan's results.
//
// It replaces wifi.Client.Scan, whose request carries a one-byte SSID (a NUL)
// where a zero-length one is meant. The radio then probes for a network named
// "\x00", no access point answers, and only the beacons that happen to arrive
// during the short active dwell are recorded, so each scan returns a different
// fraction of the networks in range.
func nl80211Scan(ctx context.Context, ifIndex int) error {
	req, err := scanRequest(ifIndex)
	if err != nil {
		return err
	}

	conn, err := genetlink.Dial(&netlink.Config{Strict: true})
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	// Receive takes no context, so the context's end closes the socket to
	// release it.
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	family, err := conn.GetFamily(unix.NL80211_GENL_NAME)
	if err != nil {
		return err
	}
	if err := joinScanGroup(conn, family); err != nil {
		return err
	}

	msg := genetlink.Message{
		Header: genetlink.Header{Command: unix.NL80211_CMD_TRIGGER_SCAN, Version: family.Version},
		Data:   req,
	}
	if _, err := conn.Send(msg, family.ID, netlink.Request|netlink.Acknowledge); err != nil {
		return err
	}

	watch := scanWatch{ifIndex: ifIndex}
	for {
		events, raw, err := conn.Receive()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		for i, ev := range events {
			// The acknowledgement of the request arrives on the same socket and
			// is not an nl80211 message.
			if uint16(raw[i].Header.Type) != family.ID {
				continue
			}
			done, err := watch.done(ev.Header.Command, ev.Data)
			if done || err != nil {
				return err
			}
		}
	}
}

// joinScanGroup subscribes conn to nl80211's scan notifications.
func joinScanGroup(conn *genetlink.Conn, family genetlink.Family) error {
	for _, group := range family.Groups {
		if group.Name == unix.NL80211_MULTICAST_GROUP_SCAN {
			return conn.JoinGroup(group.ID)
		}
	}
	return wifi.ErrScanGroupNotFound
}

// scanRequest encodes a scan trigger for the interface with index ifIndex that
// asks every access point in range to answer.
func scanRequest(ifIndex int) ([]byte, error) {
	if ifIndex < 0 || ifIndex > math.MaxUint32 {
		return nil, fmt.Errorf("nl80211: interface index %d out of range", ifIndex)
	}
	enc := netlink.NewAttributeEncoder()
	enc.Uint32(unix.NL80211_ATTR_IFINDEX, uint32(ifIndex))
	enc.Nested(unix.NL80211_ATTR_SCAN_SSIDS, func(ae *netlink.AttributeEncoder) error {
		// Zero bytes of SSID is the wildcard. Any byte at all, a NUL included,
		// names one network and silences every other access point.
		ae.Bytes(1, nil)
		return nil
	})
	return enc.Encode()
}

// scanWatch follows nl80211 scan notifications for one interface until the
// scan this process started has finished.
type scanWatch struct {
	ifIndex int
	started bool
}

// done reports whether the notification with command cmd and attributes data
// ends the scan. Results published before the scan's own start notification
// belong to an earlier scan (the supplicant's, usually one channel) and are
// ignored.
func (w *scanWatch) done(cmd uint8, data []byte) (bool, error) {
	if !scanEventOn(data, w.ifIndex) {
		return false, nil
	}
	switch cmd {
	case unix.NL80211_CMD_TRIGGER_SCAN:
		w.started = true
	case unix.NL80211_CMD_SCAN_ABORTED:
		if w.started {
			return true, wifi.ErrScanAborted
		}
	case unix.NL80211_CMD_NEW_SCAN_RESULTS:
		return w.started, nil
	}
	return false, nil
}

// scanEventOn reports whether a scan notification's attributes name the
// interface with index ifIndex.
func scanEventOn(data []byte, ifIndex int) bool {
	ad, err := netlink.NewAttributeDecoder(data)
	if err != nil {
		return false
	}
	for ad.Next() {
		if ad.Type() == unix.NL80211_ATTR_IFINDEX {
			return int(ad.Uint32()) == ifIndex
		}
	}
	return false
}
