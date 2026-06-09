package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tj-smith47/shelly-go/gen1"
)

// Gen1NetworkOverride replaces the WiFi station settings applied during a Gen1
// restore. It lets one device's configuration be cloned onto another without
// copying the source's IP address, so both devices stay online with distinct
// addresses. SSID and Password are optional: when empty, the backup's own
// station credentials are kept.
type Gen1NetworkOverride struct {
	// SSID overrides the station SSID; empty keeps the backup's SSID.
	SSID string
	// Password overrides the station key; empty keeps the backup's key.
	Password string
	// StaticIP, when set, switches the station to a static IPv4 address.
	// Gateway and Netmask are required alongside it.
	StaticIP string
	// Gateway is the static IPv4 default gateway.
	Gateway string
	// Netmask is the static IPv4 subnet mask.
	Netmask string
	// DNS is the static IPv4 nameserver (optional).
	DNS string
}

// IsStatic reports whether a static IPv4 address was requested.
func (o *Gen1NetworkOverride) IsStatic() bool {
	return o != nil && o.StaticIP != ""
}

// Gen1RestoreOptions configures a Gen1 restore.
type Gen1RestoreOptions struct {
	// NetworkOverride, when non-nil, replaces the backup's WiFi station settings
	// before they are applied. Ignored when SkipNetwork is true.
	NetworkOverride *Gen1NetworkOverride
	// Name overrides the device's stored display name. Empty leaves the name as
	// the backup's. Used so a cloned device is named distinctly from its source.
	Name string
	// SkipNetwork skips WiFi configuration.
	SkipNetwork bool
	// SkipAuth skips authentication configuration.
	SkipAuth bool
	// SkipState skips restoring captured live component state — color temperature
	// and brightness — so a restore leaves the target's current light look intact
	// and applies configuration only.
	SkipState bool
	// SkipMeters skips restoring meter / energy-meter configuration (e.g. Gen1
	// overpower limits), leaving the target's protection settings untouched.
	SkipMeters bool
	// SkipWebhooks skips webhook (action URL) restoration.
	SkipWebhooks bool
}

// gen1IPv4ModeStatic is the value used to request static IPv4 addressing on Gen1
// WiFi station config (ipv4_method).
const gen1IPv4ModeStatic = "static"

// ExportGen1 builds a Backup from a Gen1 device by reading its settings and
// actions. The backup is stored in the same Backup struct used for Gen2, with
// DeviceInfo.Generation=1. DeviceInfo is populated from the device's /shelly
// response; callers may enrich it further afterward.
func ExportGen1(ctx context.Context, dev *gen1.Device) (*Backup, error) {
	bkp := &Backup{
		Version:   BackupVersion,
		CreatedAt: time.Now().UTC(),
	}

	// Get device info
	info, err := dev.GetDeviceInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}
	bkp.DeviceInfo = &DeviceInfo{
		Model:      info.Model,
		MAC:        info.MAC,
		Version:    info.Version,
		Generation: 1,
	}

	// Get full settings (this is Gen1's equivalent of Shelly.GetConfig)
	settings, err := dev.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Store full settings as Config
	configData, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}
	bkp.Config = configData

	// Extract WiFi settings
	bkp.WiFi = marshalGen1WiFi(settings)

	// Extract MQTT settings
	if settings.MQTT != nil {
		bkp.MQTT = mustMarshal(settings.MQTT)
	}

	// Extract Cloud settings
	if settings.Cloud != nil {
		bkp.Cloud = mustMarshal(settings.Cloud)
	}

	// Extract Auth info
	if settings.Login != nil {
		bkp.Auth = &AuthInfo{
			User:   settings.Login.Username,
			Enable: settings.Login.Enabled,
		}
	}

	// Extract component settings
	bkp.Components = marshalGen1Components(settings)

	// Capture live light/color/white state — the brightness, color temperature,
	// RGB and per-channel values the device keeps in /light, /color and /white
	// rather than /settings, which would otherwise be lost on restore.
	captureGen1LiveState(ctx, dev, settings, bkp)

	// Extract schedule rules from relay/light settings
	bkp.Schedules = marshalGen1Schedules(settings)

	// Get action URLs (Gen1's equivalent of webhooks)
	actions, err := dev.GetActions(ctx)
	if err == nil && actions != nil {
		bkp.Webhooks = mustMarshal(actions)
	}

	return bkp, nil
}

// RestoreGen1 restores a Backup to a Gen1 device via individual HTTP settings
// calls. The order of operations mirrors the device's hardware-verified restore
// sequence and must not be reordered.
func RestoreGen1(ctx context.Context, dev *gen1.Device, bkp *Backup, opts *Gen1RestoreOptions) (*RestoreResult, error) {
	result := &RestoreResult{
		Success: true,
	}

	// Parse the full settings from backup Config
	var settings gen1.Settings
	if bkp.Config != nil {
		if err := json.Unmarshal(bkp.Config, &settings); err != nil {
			return nil, fmt.Errorf("failed to parse backup config: %w", err)
		}
	}

	// A name override replaces the backup's stored name (e.g. for a clone).
	if opts.Name != "" {
		settings.Name = opts.Name
	}

	// Restore device-level settings
	restoreGen1DeviceSettings(ctx, dev, &settings, result)

	// Restore WiFi (if not skipped)
	if !opts.SkipNetwork {
		restoreGen1WiFi(ctx, dev, bkp, opts.NetworkOverride, result)
	}

	// Restore MQTT
	restoreGen1MQTT(ctx, dev, &settings, result)

	// Restore Cloud
	restoreGen1Cloud(ctx, dev, &settings, result)

	// Restore CoIoT
	restoreGen1CoIoT(ctx, dev, &settings, result)

	// Restore SNTP
	restoreGen1SNTP(ctx, dev, &settings, result)

	// Restore Auth (if not skipped)
	if !opts.SkipAuth {
		restoreGen1Auth(ctx, dev, bkp, result)
	}

	// Restore component configs (if not skipped via schedules/scripts)
	restoreGen1Components(ctx, dev, &settings, result)

	// Apply captured live light state (color temperature, brightness, color, white
	// channels) — these live in /light, /color and /white, not /settings, so the
	// component config above does not carry them. Each apply is a no-op when its
	// state is absent from the backup.
	if !opts.SkipState {
		applyGen1LightState(ctx, dev, bkp, result)
		applyGen1ColorState(ctx, dev, bkp, result)
		applyGen1WhiteState(ctx, dev, bkp, result)
	}

	// Re-apply per-meter overpower limits, which restoreGen1Components does not
	// cover (the device-level max_power is restored with the device settings).
	if !opts.SkipMeters {
		restoreGen1Meters(ctx, dev, &settings, result)
		restoreGen1EMeters(ctx, dev, &settings, result)
	}

	// Restore action URLs / webhooks (if not skipped)
	if !opts.SkipWebhooks {
		restoreGen1Actions(ctx, dev, bkp, result)
	}

	return result, nil
}

// Component map keys under which captured live state is stored, separate from the
// /settings-derived config which carries only static fields (name, default_state).
const (
	lightStateKey = "light_state"
	colorStateKey = "color_state"
	whiteStateKey = "white_state"
)

// Gen1 device operating modes (settings.mode) for multi-mode RGBW devices.
const (
	gen1ModeColor = "color"
	gen1ModeWhite = "white"
)

// gen1LightState is a light's live, restorable state — the color temperature and
// brightness the Duo keeps in /light rather than /settings.
type gen1LightState struct {
	ID         int `json:"id"`
	Temp       int `json:"temp,omitempty"`
	Brightness int `json:"brightness"`
}

// captureGen1LiveState reads the device's live light/color/white state and stores
// each non-empty result under its Components key. The light state is captured for
// any device with lights (covers white-temp bulbs like the Duo); color and white
// state are gated on the device mode and best-effort, so a device that does not
// expose the matching endpoint contributes nothing.
func captureGen1LiveState(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, bkp *Backup) {
	put := func(key string, data json.RawMessage) {
		if data == nil {
			return
		}
		if bkp.Components == nil {
			bkp.Components = map[string]json.RawMessage{}
		}
		bkp.Components[key] = data
	}

	put(lightStateKey, captureGen1LightState(ctx, dev, len(settings.Lights)))
	if settings.Mode == gen1ModeColor {
		put(colorStateKey, captureGen1ColorState(ctx, dev))
	}
	if settings.Mode == gen1ModeWhite {
		put(whiteStateKey, captureGen1WhiteState(ctx, dev, len(settings.Lights)))
	}
}

// captureGen1LightState reads each light's live status so color temperature and
// brightness survive a backup. Returns nil when there are no lights or none could
// be read; per-light read failures are skipped rather than failing the backup.
func captureGen1LightState(ctx context.Context, dev *gen1.Device, numLights int) json.RawMessage {
	if numLights == 0 {
		return nil
	}
	states := make([]gen1LightState, 0, numLights)
	for i := range numLights {
		status, err := dev.Light(i).GetStatus(ctx)
		if err != nil {
			continue
		}
		states = append(states, gen1LightState{ID: i, Temp: status.Temp, Brightness: status.Brightness})
	}
	if len(states) == 0 {
		return nil
	}
	return mustMarshal(states)
}

// restoreGen1Meters re-applies per-meter overpower limits captured in
// settings.Meters. Only PowerLimit is writable on Gen1 — the API exposes no setter
// for the under/over limits — and a zero limit (unset) is left alone.
func restoreGen1Meters(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	for i := range settings.Meters {
		limit := settings.Meters[i].PowerLimit
		if limit <= 0 {
			continue
		}
		if err := dev.Meter(i).SetPowerLimit(ctx, limit); err != nil {
			addWarningf(result, "set meter %d power limit: %v", i, err)
		}
	}
}

// restoreGen1EMeters re-applies the writable energy-meter settings captured in
// settings.EMeters (Shelly EM / 3EM): the overpower threshold and the over/under
// power action URLs and their thresholds. SetEMeterConfig is a no-op when an
// entry carries none of those, so untouched meters are left alone.
func restoreGen1EMeters(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	for i := range settings.EMeters {
		if err := dev.SetEMeterConfig(ctx, i, settings.EMeters[i]); err != nil {
			addWarningf(result, "set emeter %d config: %v", i, err)
		}
	}
}

// applyGen1LightState re-applies captured color temperature and brightness to the
// device's lights. Absent state (older backups, non-light devices) is a no-op.
// At the clockless factory AP the temp command may be rejected; the LAN re-apply
// pass in RestoreToAP runs this again where the device has a clock.
func applyGen1LightState(ctx context.Context, dev *gen1.Device, bkp *Backup, result *RestoreResult) {
	raw, ok := bkp.Components[lightStateKey]
	if !ok {
		return
	}
	var states []gen1LightState
	if err := json.Unmarshal(raw, &states); err != nil {
		addWarningf(result, "parse light state: %v", err)
		return
	}
	for _, st := range states {
		light := dev.Light(st.ID)
		if st.Temp > 0 {
			if tErr := light.SetColorTemp(ctx, st.Temp); tErr != nil {
				addWarningf(result, "set light %d color temp: %v", st.ID, tErr)
			}
		}
		if st.Brightness > 0 {
			if bErr := light.SetBrightness(ctx, st.Brightness); bErr != nil {
				addWarningf(result, "set light %d brightness: %v", st.ID, bErr)
			}
		}
	}
}

// gen1ColorState is a color-mode light's live, restorable state — the RGB
// channels, white channel, gain (brightness) and effect the Bulb/RGBW2 keep in
// /color rather than /settings.
type gen1ColorState struct {
	Red    int `json:"red"`
	Green  int `json:"green"`
	Blue   int `json:"blue"`
	White  int `json:"white"`
	Gain   int `json:"gain"`
	Effect int `json:"effect"`
}

// gen1WhiteState is a white channel's live, restorable brightness, kept in
// /white on RGBW2 devices running in white mode.
type gen1WhiteState struct {
	ID         int `json:"id"`
	Brightness int `json:"brightness"`
}

// captureGen1ColorState reads the single color output's live status so RGB,
// white, gain and effect survive a backup. Returns nil when the device exposes no
// /color endpoint (non-color device) or the read fails, so callers can store it
// unconditionally without polluting white-only backups.
func captureGen1ColorState(ctx context.Context, dev *gen1.Device) json.RawMessage {
	status, err := dev.Color(0).GetStatus(ctx)
	if err != nil {
		return nil
	}
	return mustMarshal(gen1ColorState{
		Red:    status.Red,
		Green:  status.Green,
		Blue:   status.Blue,
		White:  status.White,
		Gain:   status.Gain,
		Effect: status.Effect,
	})
}

// captureGen1WhiteState reads each white channel's live brightness. Returns nil
// when there are no channels or none could be read (e.g. a single-output white-temp
// bulb, whose state the /light capture already covers); per-channel read failures
// are skipped rather than failing the backup.
func captureGen1WhiteState(ctx context.Context, dev *gen1.Device, numChannels int) json.RawMessage {
	if numChannels == 0 {
		return nil
	}
	states := make([]gen1WhiteState, 0, numChannels)
	for i := range numChannels {
		status, err := dev.White(i).GetStatus(ctx)
		if err != nil {
			continue
		}
		states = append(states, gen1WhiteState{ID: i, Brightness: status.Brightness})
	}
	if len(states) == 0 {
		return nil
	}
	return mustMarshal(states)
}

// applyGen1ColorState re-applies captured RGB, white, gain and effect to the
// device's color output. Absent state (white-only or older backups) is a no-op.
// Like the light state, the color command may be rejected at the clockless factory
// AP; the LAN re-apply pass in RestoreToAP runs this again where there is a clock.
func applyGen1ColorState(ctx context.Context, dev *gen1.Device, bkp *Backup, result *RestoreResult) {
	raw, ok := bkp.Components[colorStateKey]
	if !ok {
		return
	}
	var state gen1ColorState
	if err := json.Unmarshal(raw, &state); err != nil {
		addWarningf(result, "parse color state: %v", err)
		return
	}
	color := dev.Color(0)
	if err := color.SetRGBW(ctx, state.Red, state.Green, state.Blue, state.White); err != nil {
		addWarningf(result, "set color RGBW: %v", err)
	}
	if state.Gain > 0 {
		if err := color.SetGain(ctx, state.Gain); err != nil {
			addWarningf(result, "set color gain: %v", err)
		}
	}
	if state.Effect > 0 {
		if err := color.SetEffect(ctx, state.Effect); err != nil {
			addWarningf(result, "set color effect: %v", err)
		}
	}
}

// applyGen1WhiteState re-applies captured per-channel brightness to the device's
// white channels. Absent state (non-white-mode or older backups) is a no-op.
func applyGen1WhiteState(ctx context.Context, dev *gen1.Device, bkp *Backup, result *RestoreResult) {
	raw, ok := bkp.Components[whiteStateKey]
	if !ok {
		return
	}
	var states []gen1WhiteState
	if err := json.Unmarshal(raw, &states); err != nil {
		addWarningf(result, "parse white state: %v", err)
		return
	}
	for _, st := range states {
		if st.Brightness > 0 {
			if err := dev.White(st.ID).SetBrightness(ctx, st.Brightness); err != nil {
				addWarningf(result, "set white %d brightness: %v", st.ID, err)
			}
		}
	}
}

// marshalGen1WiFi extracts WiFi settings from Gen1 Settings into a JSON blob.
func marshalGen1WiFi(settings *gen1.Settings) json.RawMessage {
	wifi := map[string]any{}
	if settings.WiFiSta != nil {
		wifi["sta"] = settings.WiFiSta
	}
	if settings.WiFiSta1 != nil {
		wifi["sta1"] = settings.WiFiSta1
	}
	if settings.WiFiAp != nil {
		wifi["ap"] = settings.WiFiAp
	}
	if settings.ApRoaming != nil {
		wifi["ap_roaming"] = settings.ApRoaming
	}
	if len(wifi) == 0 {
		return nil
	}
	return mustMarshal(wifi)
}

// marshalGen1Components extracts component settings into the Components map.
func marshalGen1Components(settings *gen1.Settings) map[string]json.RawMessage {
	components := map[string]json.RawMessage{}
	if len(settings.Lights) > 0 {
		components["lights"] = mustMarshal(settings.Lights)
	}
	if len(settings.Relays) > 0 {
		components["relays"] = mustMarshal(settings.Relays)
	}
	if len(settings.Rollers) > 0 {
		components["rollers"] = mustMarshal(settings.Rollers)
	}
	if len(settings.Meters) > 0 {
		components["meters"] = mustMarshal(settings.Meters)
	}
	if len(settings.EMeters) > 0 {
		components["emeters"] = mustMarshal(settings.EMeters)
	}
	if len(components) == 0 {
		return nil
	}
	return components
}

// marshalGen1Schedules extracts schedule rules from relay and light settings.
func marshalGen1Schedules(settings *gen1.Settings) json.RawMessage {
	type scheduleEntry struct {
		Component string   `json:"component"`
		Rules     []string `json:"rules"`
		ID        int      `json:"id"`
		Enabled   bool     `json:"enabled"`
	}

	var entries []scheduleEntry
	for i := range settings.Relays {
		relay := &settings.Relays[i]
		if len(relay.ScheduleRules) > 0 {
			entries = append(entries, scheduleEntry{
				Component: "relay",
				ID:        i,
				Enabled:   relay.Schedule,
				Rules:     relay.ScheduleRules,
			})
		}
	}
	for i := range settings.Lights {
		light := &settings.Lights[i]
		if len(light.ScheduleRules) > 0 {
			entries = append(entries, scheduleEntry{
				Component: "light",
				ID:        i,
				Enabled:   light.Schedule,
				Rules:     light.ScheduleRules,
			})
		}
	}
	if len(entries) == 0 {
		return nil
	}
	return mustMarshal(entries)
}

// restoreGen1DeviceSettings restores device-level settings (name, timezone, etc.).
func restoreGen1DeviceSettings(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	if settings.Name != "" {
		if err := dev.SetName(ctx, settings.Name); err != nil {
			addWarningf(result, "set name: %v", err)
		}
	}
	if settings.Tz != "" {
		if err := dev.SetTimezone(ctx, settings.Tz); err != nil {
			addWarningf(result, "set timezone: %v", err)
		}
	}
	if settings.Lat != 0 || settings.Lng != 0 {
		if err := dev.SetLocation(ctx, settings.Lat, settings.Lng); err != nil {
			addWarningf(result, "set location: %v", err)
		}
	}
	if settings.Mode != "" {
		if err := dev.SetMode(ctx, settings.Mode); err != nil {
			addWarningf(result, "set mode: %v", err)
		}
	}
	if err := dev.SetDiscoverable(ctx, settings.Discoverable); err != nil {
		addWarningf(result, "set discoverable: %v", err)
	}
	if settings.MaxPower > 0 {
		if err := dev.SetMaxPower(ctx, settings.MaxPower); err != nil {
			addWarningf(result, "set max power: %v", err)
		}
	}
}

// restoreGen1WiFi restores WiFi settings from the backup. When override is
// non-nil, its network fields replace the backup's station settings before they
// are applied (used to clone a device's config without copying its IP address).
func restoreGen1WiFi(
	ctx context.Context,
	dev *gen1.Device,
	bkp *Backup,
	override *Gen1NetworkOverride,
	result *RestoreResult,
) {
	if bkp.WiFi == nil && override == nil {
		return
	}
	var wifi struct {
		Sta       *gen1.WiFiStaSettings   `json:"sta,omitempty"`
		Sta1      *gen1.WiFiStaSettings   `json:"sta1,omitempty"`
		Ap        *gen1.WiFiApSettings    `json:"ap,omitempty"`
		ApRoaming *gen1.ApRoamingSettings `json:"ap_roaming,omitempty"`
	}
	if bkp.WiFi != nil {
		if err := json.Unmarshal(bkp.WiFi, &wifi); err != nil {
			addWarningf(result, "parse WiFi config: %v", err)
			return
		}
	}

	if override != nil {
		if wifi.Sta == nil {
			wifi.Sta = &gen1.WiFiStaSettings{Enabled: true}
		}
		applyGen1WiFiOverride(wifi.Sta, override)
	}

	if wifi.Sta != nil {
		restoreGen1WiFiStation(ctx, dev, wifi.Sta, result)
		result.RestartRequired = true
	}
	// Restore the secondary (backup) station so the device keeps its failover
	// network. Only meaningful when the source actually configured an SSID.
	if wifi.Sta1 != nil && wifi.Sta1.SSID != "" {
		restoreGen1WiFiStation1(ctx, dev, wifi.Sta1, result)
		result.RestartRequired = true
	}
	if wifi.Ap != nil {
		if err := dev.SetWiFiAP(ctx, wifi.Ap.Enabled, wifi.Ap.SSID, wifi.Ap.Key); err != nil {
			addWarningf(result, "set WiFi AP: %v", err)
		}
	}
	if wifi.ApRoaming != nil {
		if err := dev.SetApRoaming(ctx, wifi.ApRoaming.Enabled, wifi.ApRoaming.Threshold); err != nil {
			addWarningf(result, "set AP roaming: %v", err)
		}
	}
}

// applyGen1WiFiOverride overlays a Gen1NetworkOverride onto a Gen1 station config.
// SSID and Key are replaced only when explicitly provided; a static IP switches
// the station to static IPv4 addressing.
func applyGen1WiFiOverride(sta *gen1.WiFiStaSettings, ov *Gen1NetworkOverride) {
	sta.Enabled = true
	if ov.SSID != "" {
		sta.SSID = ov.SSID
	}
	if ov.Password != "" {
		sta.Key = ov.Password
	}
	if ov.IsStatic() {
		sta.Ipv4Method = gen1IPv4ModeStatic
		sta.IP = ov.StaticIP
		sta.Gw = ov.Gateway
		sta.Mask = ov.Netmask
		sta.DNS = ov.DNS
	}
}

// restoreGen1WiFiStation restores a WiFi station configuration.
func restoreGen1WiFiStation(ctx context.Context, dev *gen1.Device, sta *gen1.WiFiStaSettings, result *RestoreResult) {
	if sta.Ipv4Method == gen1IPv4ModeStatic {
		err := dev.SetWiFiStationStatic(ctx, sta.SSID, sta.Key, sta.IP, sta.Gw, sta.Mask, sta.DNS)
		if err != nil {
			addWarningf(result, "set WiFi station static: %v", err)
		}
		return
	}
	if err := dev.SetWiFiStation(ctx, sta.Enabled, sta.SSID, sta.Key); err != nil {
		addWarningf(result, "set WiFi station: %v", err)
	}
}

// restoreGen1WiFiStation1 restores the secondary (backup) WiFi station, mirroring
// restoreGen1WiFiStation but targeting the sta1 slot.
func restoreGen1WiFiStation1(ctx context.Context, dev *gen1.Device, sta *gen1.WiFiStaSettings, result *RestoreResult) {
	if sta.Ipv4Method == gen1IPv4ModeStatic {
		err := dev.SetWiFiStation1Static(ctx, sta.SSID, sta.Key, sta.IP, sta.Gw, sta.Mask, sta.DNS)
		if err != nil {
			addWarningf(result, "set WiFi station1 static: %v", err)
		}
		return
	}
	if err := dev.SetWiFiStation1(ctx, sta.Enabled, sta.SSID, sta.Key); err != nil {
		addWarningf(result, "set WiFi station1: %v", err)
	}
}

// restoreGen1MQTT restores MQTT settings.
func restoreGen1MQTT(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	if settings.MQTT == nil {
		return
	}
	cfg := &gen1.MQTTConfig{
		Enable:              settings.MQTT.Enable,
		Server:              settings.MQTT.Server,
		User:                settings.MQTT.User,
		Password:            settings.MQTT.Pass,
		ID:                  settings.MQTT.ID,
		KeepAlive:           settings.MQTT.KeepAlive,
		MaxQos:              settings.MQTT.MaxQos,
		CleanSession:        settings.MQTT.CleanSession,
		Retain:              settings.MQTT.Retain,
		UpdatePeriod:        settings.MQTT.UpdatePeriod,
		ReconnectTimeoutMax: settings.MQTT.ReconnectTimeoutMax,
		ReconnectTimeoutMin: settings.MQTT.ReconnectTimeoutMin,
	}
	if err := dev.SetMQTTConfig(ctx, cfg); err != nil {
		addWarningf(result, "set MQTT: %v", err)
	}
}

// restoreGen1Cloud restores Cloud settings.
func restoreGen1Cloud(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	if settings.Cloud == nil {
		return
	}
	if err := dev.SetCloud(ctx, settings.Cloud.Enabled); err != nil {
		addWarningf(result, "set cloud: %v", err)
	}
}

// restoreGen1CoIoT restores CoIoT protocol settings.
func restoreGen1CoIoT(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	if settings.CoIoT == nil {
		return
	}
	if err := dev.SetCoIoT(ctx, settings.CoIoT.Enabled, settings.CoIoT.UpdatePeriod, settings.CoIoT.Peer); err != nil {
		addWarningf(result, "set CoIoT: %v", err)
	}
}

// restoreGen1SNTP restores time server settings.
func restoreGen1SNTP(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	if settings.SNTP == nil {
		return
	}
	if settings.SNTP.Server != "" {
		if err := dev.SetTimeServer(ctx, settings.SNTP.Server); err != nil {
			addWarningf(result, "set time server: %v", err)
		}
	}
}

// restoreGen1Auth restores authentication settings.
func restoreGen1Auth(ctx context.Context, dev *gen1.Device, bkp *Backup, result *RestoreResult) {
	if bkp.Auth == nil {
		return
	}
	// Note: Gen1 auth restore enables/disables auth but cannot restore the password
	// (passwords are never exported). User must set password separately if needed.
	if err := dev.SetAuth(ctx, bkp.Auth.Enable, bkp.Auth.User, ""); err != nil {
		addWarningf(result, "set auth: %v", err)
	}
}

// restoreGen1Components restores component-specific configurations.
func restoreGen1Components(ctx context.Context, dev *gen1.Device, settings *gen1.Settings, result *RestoreResult) {
	for i := range settings.Lights {
		light := &settings.Lights[i]
		cfg := gen1.LightConfig{
			Name:         light.Name,
			DefaultState: light.DefaultState,
			BtnType:      light.BtnType,
			AutoOn:       light.AutoOn,
			AutoOff:      light.AutoOff,
			BtnReverse:   light.BtnReverse != 0,
			Schedule:     light.Schedule,
			// Carry the rules in the same write as the schedule flag: a separate
			// follow-up schedule_rules call races the config write on the device
			// and is silently dropped (200 OK, rules not persisted).
			ScheduleRules: light.ScheduleRules,
		}
		if err := dev.SetLightConfig(ctx, i, &cfg); err != nil {
			addWarningf(result, "set light %d config: %v", i, err)
		}
	}

	for i := range settings.Relays {
		relay := &settings.Relays[i]
		cfg := &gen1.RelayConfig{
			Name:         relay.Name,
			DefaultState: relay.DefaultState,
			BtnType:      relay.BtnType,
			AutoOn:       relay.AutoOn,
			AutoOff:      relay.AutoOff,
			MaxPower:     relay.MaxPower,
			BtnReverse:   relay.IsBtnReverse(),
			Schedule:     relay.Schedule,
			// As with lights, carry the rules atomically with the config write so
			// the device does not silently drop a separate follow-up call.
			ScheduleRules: relay.ScheduleRules,
		}
		if err := dev.SetRelayConfig(ctx, i, cfg); err != nil {
			addWarningf(result, "set relay %d config: %v", i, err)
		}
	}

	for i := range settings.Rollers {
		roller := &settings.Rollers[i]
		cfg := &gen1.RollerConfig{
			MaxTimeOpen:    roller.MaxTimeOpen,
			MaxTimeClose:   roller.MaxTimeClose,
			DefaultState:   roller.DefaultState,
			Swap:           roller.Swap,
			SwapInputs:     roller.SwapInputs,
			InputMode:      roller.InputMode,
			BtnType:        roller.BtnType,
			BtnReverse:     roller.BtnReverse != 0,
			SafetyMode:     roller.SafetyMode,
			SafetyAction:   roller.SafetyAction,
			ObstacleMode:   roller.ObstacleMode,
			ObstacleAction: roller.ObstacleAction,
			ObstaclePower:  roller.ObstaclePower,
			ObstacleDelay:  roller.ObstacleDelay,
			Positioning:    roller.Positioning,
		}
		if err := dev.SetRollerConfig(ctx, i, cfg); err != nil {
			addWarningf(result, "set roller %d config: %v", i, err)
		}
	}
}

// restoreGen1Actions restores action URLs (Gen1's equivalent of webhooks).
func restoreGen1Actions(ctx context.Context, dev *gen1.Device, bkp *Backup, result *RestoreResult) {
	if bkp.Webhooks == nil {
		return
	}
	var actions gen1.ActionSettings
	if err := json.Unmarshal(bkp.Webhooks, &actions); err != nil {
		addWarningf(result, "parse actions: %v", err)
		return
	}

	for _, action := range actions.Actions {
		if len(action.URLs) > 0 {
			if err := dev.SetAction(ctx, action.Index, action.Event, action.URLs, action.Enabled); err != nil {
				addWarningf(result, "set action %s: %v", action.Event, err)
			}
		}
	}
}

// addWarningf appends a formatted warning message to the restore result.
func addWarningf(result *RestoreResult, format string, args ...any) {
	result.Warnings = append(result.Warnings, fmt.Sprintf(format, args...))
}

// mustMarshal marshals a value to JSON, panicking on error.
// Only used for values that are known to be valid JSON-serializable types.
func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("backup: failed to marshal %T: %v", v, err))
	}
	return data
}
