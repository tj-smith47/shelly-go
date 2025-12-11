// Package lora provides support for the Shelly LoRa Add-On.
//
// The Shelly LoRa Add-On enables long-range, low-power wireless communication
// between Shelly devices and other LoRa-compatible devices. This allows for
// extended range communication beyond WiFi capabilities, making it ideal for
// outdoor installations, agricultural monitoring, and industrial applications.
//
// # Supported Devices
//
// The LoRa add-on works with Shelly devices that have an add-on connector,
// including:
//   - Shelly Plus 1
//   - Shelly Plus 1PM
//   - Shelly Plus 2PM
//   - Shelly Plus i4
//   - Other Plus/Pro devices with add-on support
//
// # Enabling the Add-On
//
// Before using LoRa functionality, the add-on must be enabled in the device
// configuration. This is done by setting the device.addon_type to "LoRa" in
// the Sys configuration:
//
//	// Enable LoRa add-on via Sys configuration
//	sysClient.SetConfig(ctx, &sys.SetConfigParams{
//	    Device: &sys.DeviceConfig{
//	        AddonType: types.StringPtr("LoRa"),
//	    },
//	})
//	// Reboot required after enabling
//
// # Basic Usage
//
// Once the LoRa add-on is enabled and the device rebooted, create a LoRa
// component instance:
//
//	client := rpc.NewClient(transport)
//	lora := lora.NewLoRa(client, 100) // ID 100 is the default LoRa component ID
//
//	// Get current configuration
//	config, err := lora.GetConfig(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Frequency: %d Hz\n", config.Freq)
//
// # Sending Data
//
// Send data over LoRa RF using SendBytes. Data must be base64 encoded:
//
//	import "encoding/base64"
//
//	data := base64.StdEncoding.EncodeToString([]byte("Hello LoRa!"))
//	err := lora.SendBytes(ctx, data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Receiving Data
//
// Data received over LoRa RF triggers NotifyEvent notifications. Subscribe
// to these events through the RPC client's notification system:
//
//	client.OnNotificationMethod("NotifyEvent", func(method string, params json.RawMessage) {
//	    var event struct {
//	        Component string `json:"component"`
//	        Event     string `json:"event"`
//	        Info      struct {
//	            Data string  `json:"data"` // Base64 encoded
//	            RSSI int     `json:"rssi"`
//	            SNR  float64 `json:"snr"`
//	            TS   float64 `json:"ts"`
//	        } `json:"info"`
//	    }
//	    json.Unmarshal(params, &event)
//
//	    if event.Component == "lora:100" && event.Event == "lora" {
//	        decoded, _ := base64.StdEncoding.DecodeString(event.Info.Data)
//	        fmt.Printf("Received: %s (RSSI: %d, SNR: %.1f)\n",
//	            decoded, event.Info.RSSI, event.Info.SNR)
//	    }
//	})
//
// # Configuration Parameters
//
// The LoRa configuration includes radio parameters:
//   - Freq: RF frequency in Hz (e.g., 865000000 for 865 MHz)
//   - BW: Bandwidth setting (affects range vs data rate)
//   - DR: Data rate (spreading factor, higher = longer range but slower)
//   - Plen: Preamble length for packet detection
//   - TxP: Transmit power in dBm (affects range and power consumption)
//
// # Regional Considerations
//
// LoRa frequencies and parameters are regulated by region:
//   - EU868: 863-870 MHz (Europe)
//   - US915: 902-928 MHz (Americas)
//   - AU915: 915-928 MHz (Australia)
//   - AS923: 923 MHz (Asia)
//
// Configure appropriate frequencies and power levels for your region to
// comply with local radio regulations.
//
// # Add-On Management
//
// The LoRa add-on firmware can be updated using the AddOn methods:
//
//	// Check for updates
//	info, err := lora.GetAddOnInfo(ctx)
//
//	// Check if update is available
//	hasUpdate, err := lora.CheckForUpdate(ctx)
//
//	// Apply update if available
//	err := lora.Update(ctx)
package lora
