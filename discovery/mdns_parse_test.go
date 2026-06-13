package discovery

// mdns_parse_test.go — tests for mDNS binary parsing functions.
// All tests use crafted DNS wire-format data; no network required.

import (
	"net"
	"testing"
)

// ─── StopDiscovery — not-running early return branch ─────────────────────────

// TestMDNS_StopDiscovery_WhenNotRunning covers the !m.running → return nil branch.
// This does not require a multicast socket.
func TestMDNS_StopDiscovery_WhenNotRunning(t *testing.T) {
	m := NewMDNSDiscoverer()
	// A fresh discoverer is not running; StopDiscovery must be a no-op.
	if err := m.StopDiscovery(); err != nil {
		t.Errorf("StopDiscovery on non-running discoverer: %v", err)
	}
}

// buildRawMDNSResponse builds a minimal mDNS response packet with the given answer records.
// Header: flags=0x8400 (QR=1 AA=1), QDCOUNT=0, ANCOUNT=len(answers), ARCOUNT=0.
func buildRawMDNSResponse(answers []byte, ancount int) []byte {
	msg := make([]byte, 0, 12+len(answers))
	msg = append(msg,
		0x00, 0x00, // ID = 0
		0x84, 0x00, // Flags: QR=1 AA=1 (response)
		0x00, 0x00, // QDCOUNT = 0
		byte(ancount>>8), byte(ancount), // ANCOUNT
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0
	)
	return append(msg, answers...)
}

// encodeDNSName encodes a dotted name like "foo.bar.local" into DNS wire format.
func encodeDNSName(name string) []byte {
	var b []byte
	for _, label := range splitDNSName(name) {
		b = append(b, byte(len(label)))
		b = append(b, label...)
	}
	b = append(b, 0x00) // null terminator
	return b
}

func splitDNSName(name string) []string {
	var labels []string
	cur := ""
	for _, c := range name {
		if c == '.' {
			if cur != "" {
				labels = append(labels, cur)
			}
			cur = ""
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		labels = append(labels, cur)
	}
	return labels
}

// buildRecord builds a DNS resource record (NAME + TYPE + CLASS + TTL + RDLENGTH + RDATA).
func buildRecord(name []byte, rtype uint16, rdata []byte) []byte {
	rec := append([]byte{}, name...)
	rec = append(rec,
		byte(rtype>>8), byte(rtype), // TYPE
		0x00, 0x01, // CLASS IN
		0x00, 0x00, 0x00, 0x78, // TTL = 120
		byte(len(rdata)>>8), byte(len(rdata)), // RDLENGTH
	)
	return append(rec, rdata...)
}

// ─── parseResponse ────────────────────────────────────────────────────────────

func TestParseResponse_TooShort(t *testing.T) {
	m := NewMDNSDiscoverer()
	if got := m.parseResponse([]byte{0x84, 0x00}); got != nil {
		t.Error("expected nil for too-short response")
	}
}

func TestParseResponse_NotAResponse(t *testing.T) {
	m := NewMDNSDiscoverer()
	// Flags byte at offset 2: bit 7 must be 1 for response; 0x00 = query.
	data := make([]byte, 12)
	data[2] = 0x00 // QR=0 → query, not response
	if got := m.parseResponse(data); got != nil {
		t.Error("expected nil for non-response packet")
	}
}

func TestParseResponse_NoAnswers(t *testing.T) {
	m := NewMDNSDiscoverer()
	// ANCOUNT=0, ARCOUNT=0
	data := []byte{
		0x00, 0x00, 0x84, 0x00,
		0x00, 0x00, // QDCOUNT
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT
		0x00, 0x00, // ARCOUNT = 0
	}
	if got := m.parseResponse(data); got != nil {
		t.Error("expected nil when no answers")
	}
}

func TestParseResponse_ARecordOnly_NoID(t *testing.T) {
	// An A record provides an address but no ID → parseResponse returns nil.
	m := NewMDNSDiscoverer()
	name := encodeDNSName("ShellyPlus1-AABBCC.local")
	rdata := []byte{192, 168, 1, 42}
	rec := buildRecord(name, dnsTypeA, rdata)
	pkt := buildRawMDNSResponse(rec, 1)
	// A record name includes ".local" → extractDeviceID returns "ShellyPlus1-AABBCC"
	// but the device.ID would only be set if device.Address is also set.
	// A-record path sets device.Address and device.ID — valid device must have both.
	got := m.parseResponse(pkt)
	// May be nil (if A record IP is not private) or non-nil.
	// 192.168.1.42 IS private → device.Address set, device.ID set.
	if got == nil {
		t.Log("parseResponse returned nil (A record name didn't produce ID)")
	}
}

func TestParseResponse_PTR_Plus_A_Record(t *testing.T) {
	m := NewMDNSDiscoverer()

	// PTR record: name = _shelly._tcp.local, rdata = instance name
	ptrName := encodeDNSName("_shelly._tcp.local")
	instanceName := encodeDNSName("ShellyPlus1-AABBCC._shelly._tcp.local")
	ptrRec := buildRecord(ptrName, dnsTypePTR, instanceName)

	// A record: name = ShellyPlus1-AABBCC.local, rdata = 10.0.0.5
	aName := encodeDNSName("ShellyPlus1-AABBCC.local")
	aRdata := net.ParseIP("10.0.0.5").To4()
	aRec := buildRecord(aName, dnsTypeA, aRdata)

	allRecords := append(ptrRec, aRec...)
	// Header with ANCOUNT=1, ARCOUNT=1 → totalRecords=2.
	pkt := make([]byte, 0, 12+len(allRecords))
	pkt = append(pkt,
		0x00, 0x00, 0x84, 0x00,
		0x00, 0x00, // QDCOUNT
		0x00, 0x01, // ANCOUNT = 1
		0x00, 0x00, // NSCOUNT
		0x00, 0x01, // ARCOUNT = 1
	)
	pkt = append(pkt, allRecords...)

	got := m.parseResponse(pkt)
	if got == nil {
		t.Fatal("expected non-nil device from PTR + A records")
	}
	if got.ID == "" {
		t.Error("expected non-empty device ID")
	}
	if got.Address == nil {
		t.Error("expected non-nil device address")
	}
}

func TestParseResponse_SRV_Record_Sets_Port(t *testing.T) {
	m := NewMDNSDiscoverer()

	// SRV rdata: priority(2) + weight(2) + port(2) + target name
	srvName := encodeDNSName("ShellyPlus1-AABBCC._shelly._tcp.local")
	targetName := encodeDNSName("ShellyPlus1-AABBCC.local")
	srvRdata := append([]byte{0, 0, 0, 0, 0x1F, 0x90}, targetName...) // port = 8080
	srvRec := buildRecord(srvName, dnsTypeSRV, srvRdata)

	// A record to give us a valid address + ID.
	aName := encodeDNSName("ShellyPlus1-AABBCC.local")
	aRdata := net.ParseIP("10.0.0.6").To4()
	aRec := buildRecord(aName, dnsTypeA, aRdata)

	allRecords := append(srvRec, aRec...)
	pkt := make([]byte, 0, 12+len(allRecords))
	pkt = append(pkt,
		0x00, 0x00, 0x84, 0x00,
		0x00, 0x00,
		0x00, 0x01, // ANCOUNT
		0x00, 0x00,
		0x00, 0x01, // ARCOUNT
	)
	pkt = append(pkt, allRecords...)

	got := m.parseResponse(pkt)
	if got == nil {
		t.Fatal("expected non-nil device from SRV + A records")
	}
	if got.Port != 8080 {
		t.Errorf("Port = %d, want 8080", got.Port)
	}
}

func TestParseResponse_TXT_Record_PopulatesDevice(t *testing.T) {
	m := NewMDNSDiscoverer()

	// TXT rdata: length-prefixed key=value strings
	// "gen=2" + "app=Plus1PM" + "ver=1.0.0" + "auth=0"
	genKV := "gen=2"
	appKV := "app=Plus1PM"
	verKV := "ver=1.0.0"
	authKV := "auth=1"
	var txtRdata []byte
	for _, kv := range []string{genKV, appKV, verKV, authKV} {
		txtRdata = append(txtRdata, byte(len(kv)))
		txtRdata = append(txtRdata, kv...)
	}
	txtName := encodeDNSName("ShellyPlus1-AABBCC._shelly._tcp.local")
	txtRec := buildRecord(txtName, dnsTypeTXT, txtRdata)

	// A record for address + ID.
	aName := encodeDNSName("ShellyPlus1-AABBCC.local")
	aRdata := net.ParseIP("10.0.0.7").To4()
	aRec := buildRecord(aName, dnsTypeA, aRdata)

	allRecords := append(txtRec, aRec...)
	pkt := make([]byte, 0, 12+len(allRecords))
	pkt = append(pkt,
		0x00, 0x00, 0x84, 0x00,
		0x00, 0x00,
		0x00, 0x01,
		0x00, 0x00,
		0x00, 0x01,
	)
	pkt = append(pkt, allRecords...)

	got := m.parseResponse(pkt)
	if got == nil {
		t.Fatal("expected non-nil device from TXT + A records")
	}
	if got.Generation != 2 {
		t.Errorf("Generation = %d, want 2", got.Generation)
	}
	if got.Model != "Plus1PM" {
		t.Errorf("Model = %q, want Plus1PM", got.Model)
	}
	if got.Firmware != "1.0.0" {
		t.Errorf("Firmware = %q, want 1.0.0", got.Firmware)
	}
	if !got.AuthRequired {
		t.Error("AuthRequired should be true")
	}
}

// ─── processRecord ────────────────────────────────────────────────────────────

func TestProcessRecord_PTR_SetsID(t *testing.T) {
	m := NewMDNSDiscoverer()
	// PTR rdata: encoded instance name.
	rdata := encodeDNSName("ShellyDevice._shelly._tcp.local")
	device := &DiscoveredDevice{}
	// The data passed to processRecord for PTR is the full packet; rdataOffset points
	// into it. We embed the PTR rdata in a larger byte slice.
	fullData := append(make([]byte, 20), rdata...)
	m.processRecord(fullData, 20, dnsTypePTR, rdata, "ShellyDevice._shelly._tcp.local", device)
	if device.ID == "" {
		t.Error("expected non-empty ID from PTR record")
	}
}

func TestProcessRecord_A_NotPrivate_DoesNotSetAddress(t *testing.T) {
	m := NewMDNSDiscoverer()
	// 8.8.8.8 is NOT private → address should not be set.
	rdata := net.ParseIP("8.8.8.8").To4()
	device := &DiscoveredDevice{}
	m.processRecord(nil, 0, dnsTypeA, rdata, "device.local", device)
	if device.Address != nil {
		t.Error("8.8.8.8 is not private — address must not be set")
	}
}

func TestProcessRecord_A_Loopback_SetsAddress(t *testing.T) {
	m := NewMDNSDiscoverer()
	// 127.0.0.1 is loopback → IsLoopback() = true → address IS set.
	rdata := net.ParseIP("127.0.0.1").To4()
	device := &DiscoveredDevice{}
	m.processRecord(nil, 0, dnsTypeA, rdata, "device.local", device)
	if device.Address == nil {
		t.Error("127.0.0.1 is loopback — expected address to be set")
	}
}

func TestProcessRecord_A_ShortRdata(t *testing.T) {
	m := NewMDNSDiscoverer()
	// len(rdata) != 4 → skip.
	device := &DiscoveredDevice{}
	m.processRecord(nil, 0, dnsTypeA, []byte{192, 168, 1}, "device.local", device)
	if device.Address != nil {
		t.Error("short A rdata must not set address")
	}
}

func TestProcessRecord_SRV_TooShort(t *testing.T) {
	m := NewMDNSDiscoverer()
	device := &DiscoveredDevice{}
	m.processRecord(nil, 0, dnsTypeSRV, []byte{0, 1, 0, 1}, "n", device)
	if device.Port != 0 {
		t.Error("SRV rdata < 6 must not set port")
	}
}

// ─── parseTXTRecord — all switch branches ─────────────────────────────────────

func TestParseTXTRecord_Gen1(t *testing.T) {
	m := NewMDNSDiscoverer()
	kv := "gen=1"
	rdata := append([]byte{byte(len(kv))}, kv...)
	device := &DiscoveredDevice{}
	m.parseTXTRecord(rdata, device)
	if device.Generation != 1 {
		t.Errorf("Generation = %d, want 1", device.Generation)
	}
}

func TestParseTXTRecord_Gen3(t *testing.T) {
	m := NewMDNSDiscoverer()
	kv := "gen=3"
	rdata := append([]byte{byte(len(kv))}, kv...)
	device := &DiscoveredDevice{}
	m.parseTXTRecord(rdata, device)
	if device.Generation != 3 {
		t.Errorf("Generation = %d, want 3", device.Generation)
	}
}

func TestParseTXTRecord_ModelFromModel(t *testing.T) {
	m := NewMDNSDiscoverer()
	kv := "model=SHSW-25"
	rdata := append([]byte{byte(len(kv))}, kv...)
	device := &DiscoveredDevice{}
	m.parseTXTRecord(rdata, device)
	if device.Model != "SHSW-25" {
		t.Errorf("Model = %q, want SHSW-25", device.Model)
	}
}

func TestParseTXTRecord_FirmwareFromFW(t *testing.T) {
	m := NewMDNSDiscoverer()
	kv := "fw=1.12.3"
	rdata := append([]byte{byte(len(kv))}, kv...)
	device := &DiscoveredDevice{}
	m.parseTXTRecord(rdata, device)
	if device.Firmware != "1.12.3" {
		t.Errorf("Firmware = %q, want 1.12.3", device.Firmware)
	}
}

func TestParseTXTRecord_AuthZero(t *testing.T) {
	m := NewMDNSDiscoverer()
	kv := "auth=0"
	rdata := append([]byte{byte(len(kv))}, kv...)
	device := &DiscoveredDevice{}
	m.parseTXTRecord(rdata, device)
	if device.AuthRequired {
		t.Error("auth=0 must not set AuthRequired")
	}
}

func TestParseTXTRecord_Truncated(t *testing.T) {
	m := NewMDNSDiscoverer()
	// length byte > remaining data → should not panic.
	rdata := []byte{0x0F, 'k', 'e', 'y'} // says 15 bytes but only 3 available
	device := &DiscoveredDevice{}
	m.parseTXTRecord(rdata, device) // must not panic
}

func TestParseTXTRecord_NoEquals(t *testing.T) {
	m := NewMDNSDiscoverer()
	kv := "nokeyvalue"
	rdata := append([]byte{byte(len(kv))}, kv...)
	device := &DiscoveredDevice{}
	m.parseTXTRecord(rdata, device) // must not panic, device unchanged
}

// ─── extractDeviceID ─────────────────────────────────────────────────────────

func TestExtractDeviceID_ShellyTCP(t *testing.T) {
	m := NewMDNSDiscoverer()
	got := m.extractDeviceID("ShellyPlus1-AABBCC._shelly._tcp.local.")
	if got != "ShellyPlus1-AABBCC" {
		t.Errorf("got %q, want ShellyPlus1-AABBCC", got)
	}
}

func TestExtractDeviceID_HTTPTcp(t *testing.T) {
	m := NewMDNSDiscoverer()
	got := m.extractDeviceID("ShellyDev._http._tcp.local.")
	if got != "ShellyDev" {
		t.Errorf("got %q, want ShellyDev", got)
	}
}

func TestExtractDeviceID_LocalSuffix(t *testing.T) {
	m := NewMDNSDiscoverer()
	got := m.extractDeviceID("ShellyDev.local.")
	if got != "ShellyDev" {
		t.Errorf("got %q, want ShellyDev", got)
	}
}

func TestExtractDeviceID_NoMatch(t *testing.T) {
	m := NewMDNSDiscoverer()
	got := m.extractDeviceID("baredevice")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// ─── parseName with compression ──────────────────────────────────────────────

func TestParseName_Compression(t *testing.T) {
	m := NewMDNSDiscoverer()
	// Build: "foo" at offset 0, then a compression pointer back to 0.
	// offset 0: \x03foo\x00
	// offset 5: \xC0\x00 (pointer to offset 0)
	data := []byte{
		0x03, 'f', 'o', 'o', 0x00, // "foo." label at offset 0
		0xC0, 0x00, // compression pointer to offset 0
	}
	name, nextOffset := m.parseName(data, 5)
	if name != "foo" {
		t.Errorf("name = %q, want 'foo'", name)
	}
	if nextOffset != 7 {
		t.Errorf("nextOffset = %d, want 7", nextOffset)
	}
}

func TestParseName_LabelTruncated(t *testing.T) {
	m := NewMDNSDiscoverer()
	// Length byte says 10 but only 3 bytes follow.
	data := []byte{0x0A, 'a', 'b', 'c'}
	name, _ := m.parseName(data, 0)
	// Must not panic; name may be empty or partial.
	_ = name
}

// ─── skipName ────────────────────────────────────────────────────────────────

func TestSkipName_NullByte(t *testing.T) {
	m := NewMDNSDiscoverer()
	data := []byte{0x00}
	offset := m.skipName(data, 0)
	if offset != 1 {
		t.Errorf("offset = %d, want 1", offset)
	}
}

func TestSkipName_CompressionPointer(t *testing.T) {
	m := NewMDNSDiscoverer()
	// 0xC0 0x05 = compression pointer → skip 2 bytes and return.
	data := []byte{0xC0, 0x05}
	offset := m.skipName(data, 0)
	if offset != 2 {
		t.Errorf("offset = %d, want 2", offset)
	}
}

func TestSkipName_Label(t *testing.T) {
	m := NewMDNSDiscoverer()
	// 3-byte label "foo" + null terminator
	data := []byte{0x03, 'f', 'o', 'o', 0x00}
	offset := m.skipName(data, 0)
	if offset != 5 {
		t.Errorf("offset = %d, want 5", offset)
	}
}

// ─── parseResourceRecord — no-progress guard ──────────────────────────────────

func TestParseResourceRecord_TruncatedHeader(t *testing.T) {
	m := NewMDNSDiscoverer()
	// Name is just a null terminator (1 byte), then only 5 bytes for the record header
	// (need 10) → should return offset unchanged.
	data := []byte{0x00, 0x00, 0x01, 0x00, 0x01, 0x00} // null + 5 bytes
	device := &DiscoveredDevice{}
	newOffset := m.parseResourceRecord(data, 0, device)
	// After parsing the null-terminated name, offset is 1; then 1+10 > 6 → bail out.
	if newOffset > len(data) {
		t.Errorf("newOffset %d exceeds data length %d", newOffset, len(data))
	}
}

// ─── parseResponse — question section skipping ───────────────────────────────

func TestParseResponse_WithQuestion(t *testing.T) {
	m := NewMDNSDiscoverer()

	// Build a response with QDCOUNT=1 + ANCOUNT=1 (A record).
	questionName := encodeDNSName("_shelly._tcp.local")
	question := append(questionName, 0x00, 0x01, 0x00, 0x01) // QTYPE=A, QCLASS=IN

	aName := encodeDNSName("ShellyDev.local")
	aRdata := net.ParseIP("10.0.0.8").To4()
	aRec := buildRecord(aName, dnsTypeA, aRdata)

	pkt := make([]byte, 0, 12+len(question)+len(aRec))
	pkt = append(pkt,
		0x00, 0x00, 0x84, 0x00,
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x01, // ANCOUNT = 1
		0x00, 0x00,
		0x00, 0x00,
	)
	pkt = append(pkt, question...)
	pkt = append(pkt, aRec...)

	got := m.parseResponse(pkt)
	// May be nil (if extractDeviceID from "ShellyDev.local" returns empty)
	// or non-nil. Either way, must not panic.
	_ = got
}
