package lora

import (
	"github.com/tj-smith47/shelly-go/types"
)

// Config represents the LoRa component configuration.
type Config struct {
	types.RawFields
	ID   int   `json:"id"`
	Freq int64 `json:"freq,omitempty"`
	BW   int   `json:"bw,omitempty"`
	DR   int   `json:"dr,omitempty"`
	Plen int   `json:"plen,omitempty"`
	TxP  int   `json:"txp,omitempty"`
}

// Status represents the LoRa component status.
type Status struct {
	types.RawFields
	ID   int     `json:"id"`
	RSSI int     `json:"rssi,omitempty"`
	SNR  float64 `json:"snr,omitempty"`
}

// SetConfigParams represents parameters for setting LoRa configuration.
type SetConfigParams struct {
	// Freq is the RF frequency in Hz.
	Freq *int64 `json:"freq,omitempty"`

	// BW is the bandwidth setting.
	BW *int `json:"bw,omitempty"`

	// DR is the data rate (spreading factor).
	DR *int `json:"dr,omitempty"`

	// Plen is the preamble length.
	Plen *int `json:"plen,omitempty"`

	// TxP is the transmit power in dBm.
	TxP *int `json:"txp,omitempty"`
}

// SendBytesParams represents parameters for sending data over LoRa.
type SendBytesParams struct {
	Data string `json:"data"`
	ID   int    `json:"id"`
}

// AddOnInfo represents information about the LoRa add-on.
type AddOnInfo struct {
	types.RawFields
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
}

// ReceivedData represents data received over LoRa RF.
// This is included in NotifyEvent notifications.
type ReceivedData struct {
	// Data is the base64-encoded received data.
	Data string `json:"data"`

	// RSSI is the Received Signal Strength Indicator.
	RSSI int `json:"rssi"`

	// SNR is the Signal-to-Noise Ratio.
	SNR float64 `json:"snr"`

	// TS is the timestamp of when the data was received.
	TS float64 `json:"ts"`
}

// Event represents a LoRa notification event.
type Event struct {
	// Component is the component identifier (e.g., "lora:100").
	Component string `json:"component"`

	// Event is the event type (e.g., "lora").
	Event string `json:"event"`

	// Info contains the received data information.
	Info ReceivedData `json:"info"`
}

// RegisteredDevice represents a registered LoRa device in the mesh network.
type RegisteredDevice struct {
	Metadata map[string]any `json:"metadata,omitempty"`
	DeviceID string         `json:"device_id"`
	Address  string         `json:"address,omitempty"`
	Name     string         `json:"name,omitempty"`
	Group    string         `json:"group,omitempty"`
	LastSeen float64        `json:"last_seen,omitempty"`
	LastRSSI int            `json:"last_rssi,omitempty"`
	LastSNR  float64        `json:"last_snr,omitempty"`
	Online   bool           `json:"online"`
}

// RoutedMessage represents a message routed from a LoRa device.
type RoutedMessage struct {
	// FromDevice is the device ID that sent the message.
	FromDevice string `json:"from_device"`

	// ToDevice is the target device ID (empty for broadcast).
	ToDevice string `json:"to_device,omitempty"`

	// Group is the target group (for multicast).
	Group string `json:"group,omitempty"`

	// Data is the raw message payload.
	Data []byte `json:"data"`

	// RSSI is the signal strength of the received message.
	RSSI int `json:"rssi"`

	// SNR is the signal-to-noise ratio.
	SNR float64 `json:"snr"`

	// Timestamp is when the message was received.
	Timestamp float64 `json:"timestamp"`
}

// MessageHandler is a function that handles incoming LoRa messages.
type MessageHandler func(msg *RoutedMessage)

// MessageFilter allows filtering messages by device, group, or custom criteria.
type MessageFilter struct {
	Custom     func(msg *RoutedMessage) bool
	FromDevice string
	Group      string
}

// Match returns true if the message matches the filter criteria.
func (f *MessageFilter) Match(msg *RoutedMessage) bool {
	if f.FromDevice != "" && msg.FromDevice != f.FromDevice {
		return false
	}
	if f.Group != "" && msg.Group != f.Group {
		return false
	}
	if f.Custom != nil && !f.Custom(msg) {
		return false
	}
	return true
}
