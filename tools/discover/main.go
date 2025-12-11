// Standalone Shelly device discovery tool.
//
// This tool scans a network for Shelly devices using multiple methods:
//   - mDNS discovery for Gen2+ devices (fast, but Gen1 doesn't support mDNS)
//   - CoIoT multicast discovery for Gen1 devices
//   - BLE scanning for unconfigured devices (Shelly BLU, Plus devices in BLE mode)
//   - WiFi AP scanning for unconfigured devices broadcasting their AP
//   - HTTP probing for all generations (slower, but finds everything)
//
// By default, network-based methods (mDNS, CoIoT, HTTP probe) run.
// BLE and WiFi scanning must be explicitly enabled as they require
// special permissions and may take longer.
//
// Usage:
//
//	go run tools/discover/main.go [options]
//
// Options:
//
//	-network string   Network CIDR to scan (default "192.168.1.0/24")
//	-timeout duration Timeout per device (default 2s)
//	-json             Output as JSON
//	-mdns             Use mDNS discovery (default true, Gen2+)
//	-coiot            Use CoIoT multicast discovery (default true, Gen1)
//	-probe            HTTP probe all IPs (default true, finds all)
//	-ble              Use BLE scanning (default false, requires bluetooth)
//	-wifi             Scan for Shelly WiFi APs (default false, requires root)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/discovery"
)

// Discovery method constants.
const (
	discoveryMethodMDNS  = "mDNS"
	discoveryMethodHTTP  = "HTTP"
	discoveryMethodCoIoT = "CoIoT"
	discoveryMethodBLE   = "BLE"
	discoveryMethodWiFi  = "WiFi"
)

// gen1ModelNames maps Gen1 model codes to friendly names.
var gen1ModelNames = map[string]string{
	"SHSW-1":   "Shelly 1",
	"SHSW-PM":  "Shelly 1PM",
	"SHSW-25":  "Shelly 2.5",
	"SHSW-21":  "Shelly 2",
	"SHSW-44":  "Shelly 4Pro",
	"SHPLG-1":  "Shelly Plug",
	"SHPLG-S":  "Shelly Plug S",
	"SHPLG-U1": "Shelly Plug US",
	"SHPLG2-1": "Shelly Plug E",
	"SHDM-1":   "Shelly Dimmer",
	"SHDM-2":   "Shelly Dimmer 2",
	"SHRGBW2":  "Shelly RGBW2",
	"SHCB-1":   "Shelly Bulb",
	"SHBLB-1":  "Shelly Bulb RGBW",
	"SHVIN-1":  "Shelly Vintage",
	"SHBDUO-1": "Shelly Duo",
	"SHDW-1":   "Shelly Door/Window",
	"SHDW-2":   "Shelly Door/Window 2",
	"SHHT-1":   "Shelly H&T",
	"SHMOS-01": "Shelly Motion",
	"SHMOS-02": "Shelly Motion 2",
	"SHWT-1":   "Shelly Flood",
	"SHSM-01":  "Shelly Smoke",
	"SHGS-1":   "Shelly Gas",
	"SHEM":     "Shelly EM",
	"SHEM-3":   "Shelly 3EM",
	"SHUNI-1":  "Shelly UNI",
	"SHIX3-1":  "Shelly i3",
	"SHBTN-1":  "Shelly Button1",
	"SHBTN-2":  "Shelly Button1 v2",
	"SHAIR-1":  "Shelly Air",
	"SHTRV-01": "Shelly TRV",
	"SHCL-255": "Shelly Color",
	"SHSPOT-1": "Shelly Spot",
	"SHSPOT-2": "Shelly Spot 2",
}

// DiscoveredDevice represents a discovered Shelly device.
type DiscoveredDevice struct {
	IP              string `json:"ip"`
	MAC             string `json:"mac,omitempty"`
	Model           string `json:"model,omitempty"`
	App             string `json:"app,omitempty"`
	Name            string `json:"name,omitempty"`
	FWVersion       string `json:"fw_version,omitempty"`
	DiscoveryMethod string `json:"discovery_method,omitempty"`
	Generation      int    `json:"generation"`
	AuthNeeded      bool   `json:"auth_needed,omitempty"`
}

func main() {
	network := flag.String("network", "192.168.1.0/24", "Network CIDR to scan")
	timeout := flag.Duration("timeout", 2*time.Second, "Timeout per device")
	jsonOutput := flag.Bool("json", false, "Output as JSON")
	useMDNS := flag.Bool("mdns", true, "Use mDNS discovery (Gen2+)")
	useCoIoT := flag.Bool("coiot", true, "Use CoIoT multicast discovery (Gen1)")
	useProbe := flag.Bool("probe", true, "HTTP probe all IPs in range (fallback)")
	useBLE := flag.Bool("ble", false, "Use BLE scanning (requires bluetooth adapter)")
	useWiFi := flag.Bool("wifi", false, "Scan for Shelly WiFi APs (may require root)")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	var devices []DiscoveredDevice

	if *useBLE {
		fmt.Fprintln(os.Stderr, "Scanning via BLE (Shelly BLU devices)...")
		bleDevices := discoverBLE(ctx, *timeout)
		devices = append(devices, bleDevices...)
	}

	if *useWiFi {
		fmt.Fprintln(os.Stderr, "Scanning for Shelly WiFi APs (unconfigured devices)...")
		wifiDevices := discoverWiFiAPs(ctx)
		devices = append(devices, wifiDevices...)
	}

	if *useMDNS {
		fmt.Fprintln(os.Stderr, "Scanning via mDNS (Gen2+)...")
		mdnsDevices := discoverMDNS(ctx, *timeout)
		devices = append(devices, mdnsDevices...)
	}

	if *useCoIoT {
		fmt.Fprintln(os.Stderr, "Scanning via CoIoT multicast (Gen1)...")
		coiotDevices := discoverCoIoT(ctx, *timeout)
		devices = append(devices, coiotDevices...)
	}

	if *useProbe {
		fmt.Fprintln(os.Stderr, "Probing network range (fallback)...")
		probeDevices := probeNetwork(ctx, *network, *timeout)
		devices = append(devices, probeDevices...)
	}

	// Deduplicate by IP
	devices = deduplicateDevices(devices)

	// Sort by generation (ascending), then IP
	sort.Slice(devices, func(i, j int) bool {
		if devices[i].Generation != devices[j].Generation {
			return devices[i].Generation < devices[j].Generation
		}
		return devices[i].IP < devices[j].IP
	})

	// Cancel before output - discovery is done
	cancel()

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(devices); err != nil {
			fmt.Fprintf(os.Stderr, "JSON encode error: %v\n", err)
			return
		}
	} else {
		printTable(devices)
	}
}

// discoverBLE uses the library's BLE discoverer.
func discoverBLE(ctx context.Context, timeout time.Duration) []DiscoveredDevice {
	discoverer, err := discovery.NewBLEDiscoverer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "BLE: Failed to initialize: %v\n", err)
		return nil
	}
	discoverer.ScanDuration = timeout * 3

	found, err := discoverer.DiscoverWithContext(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "BLE: Discovery error: %v\n", err)
		return nil
	}

	devices := make([]DiscoveredDevice, 0, len(found))
	for i := range found {
		d := &found[i]
		devices = append(devices, DiscoveredDevice{
			IP:              "", // BLE devices don't have IP until provisioned
			MAC:             d.MACAddress,
			Model:           d.Model,
			Name:            d.Name,
			FWVersion:       d.Firmware,
			DiscoveryMethod: discoveryMethodBLE,
			Generation:      int(d.Generation),
		})
	}
	return devices
}

// discoverWiFiAPs uses the library's WiFi discoverer.
func discoverWiFiAPs(ctx context.Context) []DiscoveredDevice {
	discoverer := discovery.NewWiFiDiscoverer()

	found, err := discoverer.DiscoverWithContext(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WiFi: Discovery error: %v\n", err)
		return nil
	}

	devices := make([]DiscoveredDevice, 0, len(found))
	for i := range found {
		d := &found[i]
		ip := ""
		if d.Address != nil {
			ip = d.Address.String()
		}
		devices = append(devices, DiscoveredDevice{
			IP:              ip,
			MAC:             d.MACAddress,
			Model:           d.Model,
			Name:            d.Name,
			FWVersion:       d.Firmware,
			DiscoveryMethod: discoveryMethodWiFi,
			Generation:      int(d.Generation),
		})
	}
	return devices
}

func discoverMDNS(ctx context.Context, timeout time.Duration) []DiscoveredDevice {
	// Simple mDNS query for _shelly._tcp.local
	// This is a simplified implementation - the library has a full one
	var devices []DiscoveredDevice

	// Create UDP socket for mDNS
	addr, err := net.ResolveUDPAddr("udp4", "224.0.0.251:5353")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mDNS resolve error: %v\n", err)
		return devices
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		fmt.Fprintf(os.Stderr, "mDNS listen error: %v\n", err)
		return devices
	}
	defer conn.Close()

	// Build mDNS query for _shelly._tcp.local
	query := buildMDNSQuery("_shelly._tcp.local")
	if _, err := conn.WriteTo(query, addr); err != nil {
		fmt.Fprintf(os.Stderr, "mDNS write error: %v\n", err)
		return devices
	}

	// Read responses
	if err := conn.SetReadDeadline(time.Now().Add(timeout * 2)); err != nil {
		fmt.Fprintf(os.Stderr, "mDNS deadline error: %v\n", err)
		return devices
	}

	seen := make(map[string]bool)
	buf := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			break // Timeout
		}

		ip := remoteAddr.IP.String()
		if seen[ip] {
			continue
		}
		seen[ip] = true

		// Parse the response to extract service info
		name := parseMDNSResponse(buf[:n])

		// HTTP probe to get full device info
		device := probeDevice(ctx, ip, timeout)
		if device != nil {
			if name != "" && device.Name == "" {
				device.Name = name
			}
			device.DiscoveryMethod = discoveryMethodMDNS
			devices = append(devices, *device)
		}
	}

	return devices
}

func discoverCoIoT(ctx context.Context, timeout time.Duration) []DiscoveredDevice {
	// CoIoT uses CoAP multicast on 224.0.1.187:5683 for Gen1 device discovery
	var devices []DiscoveredDevice

	addr, err := net.ResolveUDPAddr("udp4", "224.0.1.187:5683")
	if err != nil {
		fmt.Fprintf(os.Stderr, "CoIoT resolve error: %v\n", err)
		return devices
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		fmt.Fprintf(os.Stderr, "CoIoT listen error: %v\n", err)
		return devices
	}
	defer conn.Close()

	// Send CoAP GET request for /cit/d (device description)
	// CoAP header: Version=1, Type=NON, Code=GET, MsgID=random
	coap := buildCoAPRequest()
	if _, err := conn.WriteTo(coap, addr); err != nil {
		fmt.Fprintf(os.Stderr, "CoIoT write error: %v\n", err)
		return devices
	}

	if err := conn.SetReadDeadline(time.Now().Add(timeout * 2)); err != nil {
		fmt.Fprintf(os.Stderr, "CoIoT deadline error: %v\n", err)
		return devices
	}

	seen := make(map[string]bool)
	buf := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			break // Timeout
		}

		ip := remoteAddr.IP.String()
		if seen[ip] {
			continue
		}
		seen[ip] = true

		// Parse CoAP response to check if it's a Shelly
		if !isCoAPShellyResponse(buf[:n]) {
			continue
		}

		// HTTP probe to get full device info
		device := probeDevice(ctx, ip, timeout)
		if device != nil {
			device.DiscoveryMethod = discoveryMethodCoIoT
			devices = append(devices, *device)
		}
	}

	return devices
}

// buildCoAPRequest builds a CoAP NON GET request for /cit/d.
func buildCoAPRequest() []byte {
	// CoAP Header: Ver=01, Type=01 (NON), TKL=0, Code=GET (0.01), MsgID=0x0001
	// Uri-Path options: /cit/d
	return []byte{
		0x50, 0x01, // Header: NON GET
		0x00, 0x01, // Message ID
		0xB3,          // Uri-Path option: delta=11, length=3
		'c', 'i', 't', // path segment "cit"
		0x01, // Uri-Path option: delta=0, length=1
		'd',  // path segment "d"
	}
}

// isCoAPShellyResponse checks if a CoAP response looks like a Shelly device.
func isCoAPShellyResponse(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// Check for CoAP response (version 1, any type)
	if (data[0] >> 6) != 1 {
		return false
	}
	// Response should have code 2.05 (Content) = 0x45
	if data[1] != 0x45 {
		return false
	}
	// Look for "shelly" or "SHSW" in payload
	payload := string(data)
	return strings.Contains(strings.ToLower(payload), "shelly") ||
		strings.Contains(payload, "SHSW") ||
		strings.Contains(payload, "SHDM") ||
		strings.Contains(payload, "SHPLG") ||
		strings.Contains(payload, "SHRGB")
}

func probeNetwork(ctx context.Context, cidr string, timeout time.Duration) []DiscoveredDevice {
	ips := generateIPs(cidr)
	var devices []DiscoveredDevice
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrency
	sem := make(chan struct{}, 50)

	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()

			device := probeDevice(ctx, ip, timeout)
			if device != nil {
				device.DiscoveryMethod = discoveryMethodHTTP
				mu.Lock()
				devices = append(devices, *device)
				mu.Unlock()
			}
		}(ip)
	}

	wg.Wait()
	return devices
}

func probeDevice(ctx context.Context, ip string, timeout time.Duration) *DiscoveredDevice {
	client := &http.Client{Timeout: timeout}

	// Try Gen2+ endpoint first
	gen2URL := fmt.Sprintf("http://%s/rpc/Shelly.GetDeviceInfo", ip)
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, gen2URL, http.NoBody)
	if err != nil {
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		// Try Gen1 endpoint
		return probeGen1(ctx, client, ip, timeout)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var info struct {
			Name  string `json:"name"`
			ID    string `json:"id"`
			MAC   string `json:"mac"`
			Model string `json:"model"`
			FW    string `json:"fw_id"`
			App   string `json:"app"`
			Gen   int    `json:"gen"`
			Auth  bool   `json:"auth_en"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&info); err == nil {
			return &DiscoveredDevice{
				IP:         ip,
				MAC:        info.MAC,
				Model:      info.Model,
				App:        info.App,
				Name:       info.Name,
				Generation: info.Gen,
				FWVersion:  info.FW,
				AuthNeeded: info.Auth,
			}
		}
	}

	// Try Gen1 as fallback
	return probeGen1(ctx, client, ip, timeout)
}

func probeGen1(ctx context.Context, client *http.Client, ip string, timeout time.Duration) *DiscoveredDevice {
	gen1URL := fmt.Sprintf("http://%s/shelly", ip)
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, gen1URL, http.NoBody)
	if err != nil {
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var info struct {
			Type       string `json:"type"`
			MAC        string `json:"mac"`
			FW         string `json:"fw"`
			Name       string `json:"name,omitempty"`
			Auth       bool   `json:"auth"`
			Discovable bool   `json:"discoverable"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&info); err == nil {
			device := &DiscoveredDevice{
				IP:         ip,
				MAC:        info.MAC,
				Model:      info.Type,
				Name:       info.Name,
				Generation: 1,
				FWVersion:  info.FW,
				AuthNeeded: info.Auth,
			}

			// Gen1 devices don't include name in /shelly response
			// Fetch from /settings endpoint
			if device.Name == "" {
				device.Name = getGen1Name(ctx, client, ip, timeout)
			}

			return device
		}
	}

	return nil
}

// getGen1Name fetches the device name from Gen1 /settings endpoint.
func getGen1Name(ctx context.Context, client *http.Client, ip string, timeout time.Duration) string {
	settingsURL := fmt.Sprintf("http://%s/settings", ip)
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, settingsURL, http.NoBody)
	if err != nil {
		return ""
	}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var settings map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return ""
	}

	// Gen1 stores user-configured name at root level
	if name, ok := settings["name"].(string); ok && name != "" {
		return name
	}

	// Fall back to device hostname
	if device, ok := settings["device"].(map[string]any); ok {
		if hostname, ok := device["hostname"].(string); ok {
			return hostname
		}
	}

	return ""
}

func generateIPs(cidr string) []string {
	var ips []string
	baseIP, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return ips
	}

	for ip := baseIP.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		// Skip network and broadcast addresses
		if ip[len(ip)-1] != 0 && ip[len(ip)-1] != 255 {
			ips = append(ips, ip.String())
		}
	}
	return ips
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func deduplicateDevices(devices []DiscoveredDevice) []DiscoveredDevice {
	seen := make(map[string]*DiscoveredDevice)
	for i := range devices {
		d := &devices[i]
		// Use MAC as key if IP is empty (BLE/WiFi AP devices)
		key := d.IP
		if key == "" {
			key = d.MAC
		}
		if key == "" {
			continue
		}

		existing, ok := seen[key]
		if !ok {
			seen[key] = d
			continue
		}
		// Determine if new device should replace existing
		shouldReplace := shouldReplaceDevice(existing, d)
		if shouldReplace {
			// Preserve mDNS discovery method if it was found that way
			if existing.DiscoveryMethod == discoveryMethodMDNS {
				d.DiscoveryMethod = discoveryMethodMDNS
			}
			seen[key] = d
		}
	}
	result := make([]DiscoveredDevice, 0, len(seen))
	for _, d := range seen {
		result = append(result, *d)
	}
	return result
}

// shouldReplaceDevice determines if the candidate device info should replace existing.
func shouldReplaceDevice(existing, candidate *DiscoveredDevice) bool {
	// Higher generation number usually means correct identification
	if candidate.Generation > existing.Generation {
		return true
	}
	if candidate.Generation < existing.Generation {
		return false
	}
	// Same generation - prefer mDNS discovery method (shows Gen2+ mDNS capability)
	if candidate.DiscoveryMethod == discoveryMethodMDNS && existing.DiscoveryMethod != discoveryMethodMDNS {
		return true
	}
	// Prefer more complete data
	if candidate.Model != "" && existing.Model == "" {
		return true
	}
	if candidate.App != "" && existing.App == "" {
		return true
	}
	return false
}

func printTable(devices []DiscoveredDevice) {
	if len(devices) == 0 {
		fmt.Println("No Shelly devices found.")
		return
	}

	fmt.Printf("\nFound %d Shelly device(s):\n\n", len(devices))
	fmt.Printf("%-16s %-18s %-30s %-5s %-20s %-8s %s\n",
		"IP", "MAC", "Model", "Gen", "Name", "Method", "Auth")
	fmt.Println(strings.Repeat("-", 110))

	for i := range devices {
		d := &devices[i]
		authStr := "No"
		if d.AuthNeeded {
			authStr = "Yes"
		}
		genStr := fmt.Sprintf("%d", d.Generation)
		if d.Generation == 0 {
			genStr = "?"
		}
		name := d.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		method := d.DiscoveryMethod
		if method == "" {
			method = "-"
		}
		ip := d.IP
		if ip == "" {
			ip = "(not on network)"
		}
		// Build model display: "FriendlyName (ModelCode)"
		model := formatModelDisplay(d.Model, d.App, d.Generation)
		fmt.Printf("%-16s %-18s %-30s %-5s %-20s %-8s %s\n",
			ip, d.MAC, model, genStr, name, method, authStr)
	}
	fmt.Println()
}

// formatModelDisplay creates a display string like "Plus1PM (SNSW-001P16EU)" or "Shelly 1PM (SHSW-PM)".
func formatModelDisplay(modelCode, app string, gen int) string {
	var friendlyName string
	if gen == 1 {
		// Look up Gen1 friendly name from model code
		if name, ok := gen1ModelNames[modelCode]; ok {
			friendlyName = name
		} else {
			friendlyName = modelCode // Fallback to code if not in map
		}
	} else {
		// Gen2+ uses app field as friendly name
		friendlyName = app
	}

	if friendlyName == "" {
		return modelCode
	}
	if modelCode == "" {
		return friendlyName
	}
	return fmt.Sprintf("%s (%s)", friendlyName, modelCode)
}

// buildMDNSQuery builds a minimal DNS query packet.
func buildMDNSQuery(name string) []byte {
	// DNS header (12 bytes)
	header := []byte{
		0x00, 0x00, // Transaction ID
		0x00, 0x00, // Flags (standard query)
		0x00, 0x01, // Questions: 1
		0x00, 0x00, // Answer RRs: 0
		0x00, 0x00, // Authority RRs: 0
		0x00, 0x00, // Additional RRs: 0
	}

	// Encode the name
	parts := strings.Split(name, ".")
	// Pre-allocate: each part needs 1 byte for length + part bytes + 1 null terminator
	qnameLen := 1 // null terminator
	for _, part := range parts {
		qnameLen += 1 + len(part)
	}
	qname := make([]byte, 0, qnameLen)
	for _, part := range parts {
		qname = append(qname, byte(len(part)))
		qname = append(qname, []byte(part)...)
	}
	qname = append(qname, 0x00) // Null terminator

	// Question: PTR record (0x000c), IN class (0x0001)
	question := []byte{0x00, 0x0c, 0x00, 0x01}

	packet := make([]byte, 0, len(header)+len(qname)+len(question))
	packet = append(packet, header...)
	packet = append(packet, qname...)
	packet = append(packet, question...)

	return packet
}

// parseMDNSResponse extracts the service name from an mDNS response.
func parseMDNSResponse(data []byte) string {
	// Very basic parsing - just look for readable strings
	for i := 0; i < len(data)-1; i++ {
		if data[i] == 0 {
			continue
		}
		// Look for length-prefixed strings that look like names
		length := int(data[i])
		if length > 0 && length < 64 && i+1+length <= len(data) {
			s := string(data[i+1 : i+1+length])
			if strings.HasPrefix(s, "shelly") || strings.HasPrefix(s, "Shelly") {
				return s
			}
		}
	}
	return ""
}
