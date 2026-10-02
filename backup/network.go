package backup

import (
	"cmp"
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/types"
)

// StaticNetwork is the static IPv4 addressing of a WiFi station. An empty IP
// means the station uses DHCP.
type StaticNetwork struct {
	IP      string
	Gateway string
	Netmask string
	DNS     string
}

// ResolveStaticNetwork decides the static addressing a restore writes when a
// caller overrides the backup's network.
//
// An override with no IP keeps the backup's addressing unchanged. An override
// with an IP replaces the backup's address, and each of its empty Gateway,
// Netmask and DNS fields is taken on its own from the backup's static settings.
// A static address that still has no gateway or no netmask is refused with an
// error matching both ErrIncompleteStaticNetwork and types.ErrInvalidParam,
// because a device written that way cannot be reached again.
//
// Gen1 restores with a Gen1NetworkOverride apply this rule themselves. A caller
// that writes a Gen2+ station override itself calls it before the write.
func ResolveStaticNetwork(override, fromBackup StaticNetwork) (StaticNetwork, error) {
	resolved := fromBackup
	if override.IP != "" {
		resolved = StaticNetwork{
			IP:      override.IP,
			Gateway: cmp.Or(override.Gateway, fromBackup.Gateway),
			Netmask: cmp.Or(override.Netmask, fromBackup.Netmask),
			DNS:     cmp.Or(override.DNS, fromBackup.DNS),
		}
	}
	if resolved.IP != "" && (resolved.Gateway == "" || resolved.Netmask == "") {
		var why string
		switch {
		case override.IP == "":
			why = "the backup's static address " + resolved.IP + " has no gateway or no netmask"
		case fromBackup.IP == "":
			why = resolved.IP + " needs a gateway and a netmask, and the backup's station uses DHCP, " +
				"so it has none to supply"
		default:
			why = resolved.IP + " needs a gateway and a netmask, and the backup's static settings lack them"
		}
		return StaticNetwork{}, fmt.Errorf("%w: %w: %s", types.ErrInvalidParam, ErrIncompleteStaticNetwork, why)
	}
	return resolved, nil
}

// Station returns the primary WiFi station settings recorded in the backup, as
// the raw JSON object the device reported: the WiFi blob's "sta" when it names
// an SSID, otherwise the "wifi_sta" of a Gen1 backup's full settings when that
// names one. A backup with no DeviceInfo is read as possibly Gen1. It returns
// nil when neither names an SSID.
func (b *Backup) Station() json.RawMessage {
	if b == nil {
		return nil
	}
	var blob struct {
		Sta json.RawMessage `json:"sta"`
	}
	if len(b.WiFi) > 0 && json.Unmarshal(b.WiFi, &blob) == nil && stationHasSSID(blob.Sta) {
		return blob.Sta
	}
	if (b.DeviceInfo != nil && b.DeviceInfo.Generation > 1) || len(b.Config) == 0 {
		return nil
	}
	var settings struct {
		Sta json.RawMessage `json:"wifi_sta"`
	}
	if json.Unmarshal(b.Config, &settings) == nil && stationHasSSID(settings.Sta) {
		return settings.Sta
	}
	return nil
}

// stationHasSSID reports whether a raw station object names an SSID.
func stationHasSSID(raw json.RawMessage) bool {
	var sta struct {
		SSID string `json:"ssid"`
	}
	return len(raw) > 0 && json.Unmarshal(raw, &sta) == nil && sta.SSID != ""
}
