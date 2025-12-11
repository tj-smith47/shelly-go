//go:build windows

package discovery

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// platformWiFiScanner implements WiFiScanner for Windows using netsh.
type platformWiFiScanner struct {
	*wifiscanScanner
}

// newPlatformWiFiScanner creates a platform-specific WiFi scanner for Windows.
func newPlatformWiFiScanner() WiFiScanner {
	return &platformWiFiScanner{
		wifiscanScanner: &wifiscanScanner{},
	}
}

// Connect connects to a WiFi network on Windows using netsh.
// For open networks (like Shelly AP), this creates a temporary profile.
// Note: This may require administrator privileges.
func (s *platformWiFiScanner) Connect(ctx context.Context, ssid, password string) error {
	// For open networks, we need to create a profile first
	if password == "" {
		if err := s.createOpenProfile(ctx, ssid); err != nil {
			return err
		}
	} else {
		if err := s.createSecureProfile(ctx, ssid, password); err != nil {
			return err
		}
	}

	// Connect using the profile
	cmd := exec.CommandContext(ctx, "netsh", "wlan", "connect", "name="+ssid, "ssid="+ssid)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if strings.Contains(errMsg, "not found") ||
			strings.Contains(errMsg, "could not find") {
			return ErrSSIDNotFound
		}
		return &WiFiError{Message: "netsh connect failed: " + errMsg, Err: err}
	}

	// Wait for connection to establish
	return s.waitForConnection(ctx, ssid, 15*time.Second)
}

// createOpenProfile creates a WLAN profile for an open network.
func (s *platformWiFiScanner) createOpenProfile(ctx context.Context, ssid string) error {
	// Create XML profile for open network
	profile := `<?xml version="1.0"?>
<WLANProfile xmlns="http://www.microsoft.com/networking/WLAN/profile/v1">
	<name>` + escapeXML(ssid) + `</name>
	<SSIDConfig>
		<SSID>
			<name>` + escapeXML(ssid) + `</name>
		</SSID>
	</SSIDConfig>
	<connectionType>ESS</connectionType>
	<connectionMode>manual</connectionMode>
	<MSM>
		<security>
			<authEncryption>
				<authentication>open</authentication>
				<encryption>none</encryption>
				<useOneX>false</useOneX>
			</authEncryption>
		</security>
	</MSM>
</WLANProfile>`

	return s.addProfile(ctx, profile)
}

// createSecureProfile creates a WLAN profile for a WPA2-PSK network.
func (s *platformWiFiScanner) createSecureProfile(ctx context.Context, ssid, password string) error {
	// Create XML profile for WPA2-PSK network
	profile := `<?xml version="1.0"?>
<WLANProfile xmlns="http://www.microsoft.com/networking/WLAN/profile/v1">
	<name>` + escapeXML(ssid) + `</name>
	<SSIDConfig>
		<SSID>
			<name>` + escapeXML(ssid) + `</name>
		</SSID>
	</SSIDConfig>
	<connectionType>ESS</connectionType>
	<connectionMode>manual</connectionMode>
	<MSM>
		<security>
			<authEncryption>
				<authentication>WPA2PSK</authentication>
				<encryption>AES</encryption>
				<useOneX>false</useOneX>
			</authEncryption>
			<sharedKey>
				<keyType>passPhrase</keyType>
				<protected>false</protected>
				<keyMaterial>` + escapeXML(password) + `</keyMaterial>
			</sharedKey>
		</security>
	</MSM>
</WLANProfile>`

	return s.addProfile(ctx, profile)
}

// addProfile adds a WLAN profile using netsh.
func (s *platformWiFiScanner) addProfile(ctx context.Context, profileXML string) error {
	// netsh wlan add profile filename= requires a file, so we use stdin
	cmd := exec.CommandContext(ctx, "netsh", "wlan", "add", "profile", "filename=/dev/stdin")
	cmd.Stdin = strings.NewReader(profileXML)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Profile might already exist, try to delete and recreate
		return &WiFiError{Message: "netsh add profile failed: " + stderr.String(), Err: err}
	}

	return nil
}

// Disconnect disconnects from the current WiFi network on Windows.
func (s *platformWiFiScanner) Disconnect(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "netsh", "wlan", "disconnect")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &WiFiError{Message: "netsh disconnect failed: " + stderr.String(), Err: err}
	}

	return nil
}

// CurrentNetwork returns the currently connected WiFi network on Windows.
func (s *platformWiFiScanner) CurrentNetwork(ctx context.Context) (*WiFiNetwork, error) {
	cmd := exec.CommandContext(ctx, "netsh", "wlan", "show", "interfaces")
	output, err := cmd.Output()
	if err != nil {
		return nil, &WiFiError{Message: "netsh show interfaces failed", Err: err}
	}

	return s.parseNetshInterfaces(string(output))
}

// parseNetshInterfaces parses the output of netsh wlan show interfaces.
func (s *platformWiFiScanner) parseNetshInterfaces(output string) (*WiFiNetwork, error) {
	network := &WiFiNetwork{
		LastSeen: time.Now(),
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse SSID line like "    SSID                   : NetworkName"
		if strings.HasPrefix(line, "SSID") && !strings.HasPrefix(line, "SSID BSSID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				network.SSID = strings.TrimSpace(parts[1])
			}
		}

		// Parse BSSID
		if strings.HasPrefix(line, "BSSID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				network.BSSID = strings.TrimSpace(parts[1])
			}
		}

		// Parse Signal
		if strings.HasPrefix(line, "Signal") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				signalStr := strings.TrimSpace(parts[1])
				signalStr = strings.TrimSuffix(signalStr, "%")
				var signal int
				for _, c := range signalStr {
					if c >= '0' && c <= '9' {
						signal = signal*10 + int(c-'0')
					}
				}
				// Convert percentage to dBm approximation
				network.Signal = signal - 100
			}
		}

		// Parse Channel
		if strings.HasPrefix(line, "Channel") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				channelStr := strings.TrimSpace(parts[1])
				var channel int
				for _, c := range channelStr {
					if c >= '0' && c <= '9' {
						channel = channel*10 + int(c-'0')
					}
				}
				network.Channel = channel
			}
		}

		// Parse Authentication
		if strings.HasPrefix(line, "Authentication") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				network.Security = strings.TrimSpace(parts[1])
			}
		}

		// Parse State to check if connected
		if strings.HasPrefix(line, "State") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				state := strings.TrimSpace(parts[1])
				if state != "connected" {
					return nil, &WiFiError{Message: "not connected to any WiFi network"}
				}
			}
		}
	}

	if network.SSID == "" {
		return nil, &WiFiError{Message: "not connected to any WiFi network"}
	}

	return network, nil
}

// waitForConnection waits for the WiFi connection to be established.
func (s *platformWiFiScanner) waitForConnection(ctx context.Context, ssid string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		network, err := s.CurrentNetwork(ctx)
		if err == nil && network != nil && network.SSID == ssid {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return ErrConnectionTimeout
}

// escapeXML escapes special characters for XML.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
