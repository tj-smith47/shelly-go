package gen1

import (
	"encoding/binary"
	"testing"
	"time"
)

// TestNewCoIoTListener tests CoIoT listener creation.
func TestNewCoIoTListener(t *testing.T) {
	listener := NewCoIoTListener()

	if listener == nil {
		t.Fatal("expected listener to be created")
	}

	if listener.multicastAddr != CoIoTMulticastAddr {
		t.Errorf("expected multicast addr %s, got %s", CoIoTMulticastAddr, listener.multicastAddr)
	}

	if listener.port != CoIoTPort {
		t.Errorf("expected port %d, got %d", CoIoTPort, listener.port)
	}

	if listener.bufferSize != 1500 {
		t.Errorf("expected buffer size 1500, got %d", listener.bufferSize)
	}
}

// TestCoIoTListenerOptions tests listener options.
func TestCoIoTListenerOptions(t *testing.T) {
	listener := NewCoIoTListener(
		WithCoIoTMulticastAddr("224.0.0.1"),
		WithCoIoTPort(5684),
		WithCoIoTBufferSize(2000),
	)

	if listener.multicastAddr != "224.0.0.1" {
		t.Errorf("expected multicast addr 224.0.0.1, got %s", listener.multicastAddr)
	}

	if listener.port != 5684 {
		t.Errorf("expected port 5684, got %d", listener.port)
	}

	if listener.bufferSize != 2000 {
		t.Errorf("expected buffer size 2000, got %d", listener.bufferSize)
	}
}

// TestCoIoTListenerOnStatus tests handler registration.
func TestCoIoTListenerOnStatus(t *testing.T) {
	listener := NewCoIoTListener()

	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {
		// Handler registered
	})

	if len(listener.handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(listener.handlers))
	}

	// Multiple handlers
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {})
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {})

	if len(listener.handlers) != 3 {
		t.Errorf("expected 3 handlers, got %d", len(listener.handlers))
	}
}

// TestCoIoTListenerIsRunning tests running state.
func TestCoIoTListenerIsRunning(t *testing.T) {
	listener := NewCoIoTListener()

	if listener.IsRunning() {
		t.Error("expected not running initially")
	}
}

// TestCoIoTListenerStopWithoutStart tests stopping before starting.
func TestCoIoTListenerStopWithoutStart(t *testing.T) {
	listener := NewCoIoTListener()

	err := listener.Stop()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCoIoTStatus tests CoIoTStatus struct.
func TestCoIoTStatus(t *testing.T) {
	status := &CoIoTStatus{
		DeviceID:   "shsw-pm-aabbccddeeff",
		DeviceType: "SHSW-PM",
		DeviceMAC:  "AABBCCDDEEFF",
		Generation: 1,
		Version:    2,
		Serial:     1234,
		Timestamp:  time.Now(),
		URIPath:    "/cit/s",
		SourceAddr: "192.168.1.100",
		CoAPCode:   30,
		CoAPType:   1,
		MessageID:  12345,
		Sensors: map[string]any{
			"0_1101": true,
			"0_4101": 25.5,
		},
		Actuators: map[string]any{
			"0_0": true,
		},
		Raw: []byte(`test`),
	}

	if status.DeviceID != "shsw-pm-aabbccddeeff" {
		t.Errorf("expected device ID shsw-pm-aabbccddeeff, got %s", status.DeviceID)
	}

	if status.DeviceType != "SHSW-PM" {
		t.Errorf("expected device type SHSW-PM, got %s", status.DeviceType)
	}

	if status.DeviceMAC != "AABBCCDDEEFF" {
		t.Errorf("expected MAC AABBCCDDEEFF, got %s", status.DeviceMAC)
	}

	if status.Version != 2 {
		t.Errorf("expected version 2, got %d", status.Version)
	}

	if len(status.Sensors) != 2 {
		t.Errorf("expected 2 sensors, got %d", len(status.Sensors))
	}

	if len(status.Actuators) != 1 {
		t.Errorf("expected 1 actuator, got %d", len(status.Actuators))
	}
}

// buildCoAPMessage builds a CoAP message with the given options and payload.
// nolint:unparam // code parameter is always codeStatus in tests, but kept for flexibility
func buildCoAPMessage(tokenLen int, code int, options map[int][]byte, payload []byte) []byte {
	// Header: Ver=1, Type=1 (NON), Token Length, Code, Message ID
	header := make([]byte, 4)
	header[0] = byte(0x50 | (tokenLen & 0x0F)) // Ver=1, Type=1, TKL
	header[1] = byte(code)
	binary.BigEndian.PutUint16(header[2:4], 1234) // Message ID

	result := header

	// Add token (if any)
	for i := 0; i < tokenLen; i++ {
		result = append(result, byte(i))
	}

	// Add options (sorted by option number)
	prevOption := 0
	for optNum, optVal := range options {
		delta := optNum - prevOption
		length := len(optVal)

		// Encode delta
		var deltaBytes []byte
		var deltaNibble int
		if delta < 13 {
			deltaNibble = delta
		} else if delta < 269 {
			deltaNibble = 13
			deltaBytes = []byte{byte(delta - 13)}
		} else {
			deltaNibble = 14
			deltaBytes = make([]byte, 2)
			// Safe: delta-269 is always < 65536 for CoAP options
			binary.BigEndian.PutUint16(deltaBytes, uint16(delta-269)) //nolint:gosec // test helper with controlled values
		}

		// Encode length
		var lengthBytes []byte
		var lengthNibble int
		if length < 13 {
			lengthNibble = length
		} else if length < 269 {
			lengthNibble = 13
			lengthBytes = []byte{byte(length - 13)}
		} else {
			lengthNibble = 14
			lengthBytes = make([]byte, 2)
			// Safe: length-269 is always < 65536 for test values
			binary.BigEndian.PutUint16(lengthBytes, uint16(length-269)) //nolint:gosec // test helper with controlled values
		}

		// Option byte
		result = append(result, byte((deltaNibble<<4)|lengthNibble))
		result = append(result, deltaBytes...)
		result = append(result, lengthBytes...)
		result = append(result, optVal...)

		prevOption = optNum
	}

	// Add payload marker and payload
	if len(payload) > 0 {
		result = append(result, 0xFF)
		result = append(result, payload...)
	}

	return result
}

// TestParseCoAPMessage_Basic tests basic CoAP message parsing.
func TestParseCoAPMessage_Basic(t *testing.T) {
	// Build message with GlobalDevID option and JSON payload
	options := map[int][]byte{
		optionURIPath:     []byte("cit"),
		optionGlobalDevID: []byte("SHSW-PM#C45BBE6C2D3A#2"),
	}
	payload := []byte(`{"G":[[0,1101,1],[0,4101,25.5]]}`)

	msg := buildCoAPMessage(0, codeStatus, options, payload)

	status, err := ParseCoAPMessage(msg, "192.168.1.100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.DeviceID != "shsw-pm-c45bbe6c2d3a" {
		t.Errorf("expected device ID shsw-pm-c45bbe6c2d3a, got %s", status.DeviceID)
	}

	if status.DeviceType != "SHSW-PM" {
		t.Errorf("expected device type SHSW-PM, got %s", status.DeviceType)
	}

	if status.DeviceMAC != "C45BBE6C2D3A" {
		t.Errorf("expected MAC C45BBE6C2D3A, got %s", status.DeviceMAC)
	}

	if status.Version != 2 {
		t.Errorf("expected version 2, got %d", status.Version)
	}

	if status.SourceAddr != "192.168.1.100" {
		t.Errorf("expected source 192.168.1.100, got %s", status.SourceAddr)
	}

	if len(status.Sensors) != 2 {
		t.Errorf("expected 2 sensors, got %d", len(status.Sensors))
	}

	if v, ok := status.Sensors["0_1101"]; !ok || v != float64(1) {
		t.Errorf("expected sensor 0_1101 = 1, got %v", v)
	}

	if v, ok := status.Sensors["0_4101"]; !ok || v != 25.5 {
		t.Errorf("expected sensor 0_4101 = 25.5, got %v", v)
	}
}

// TestParseCoAPMessage_TooShort tests handling of short messages.
func TestParseCoAPMessage_TooShort(t *testing.T) {
	_, err := ParseCoAPMessage([]byte{0x01, 0x02}, "")
	if err == nil {
		t.Error("expected error for short message")
	}
}

// TestParseCoAPMessage_InvalidTokenLength tests invalid token length.
func TestParseCoAPMessage_InvalidTokenLength(t *testing.T) {
	// Token length > 8
	msg := []byte{0x5F, 0x01, 0x00, 0x01} // TKL = 15 (invalid)
	_, err := ParseCoAPMessage(msg, "")
	if err == nil {
		t.Error("expected error for invalid token length")
	}
}

// TestParseCoAPMessage_MessageTooShortForToken tests token exceeds message.
func TestParseCoAPMessage_MessageTooShortForToken(t *testing.T) {
	// Token length = 8 but message is only 4 bytes
	msg := []byte{0x58, 0x01, 0x00, 0x01}
	_, err := ParseCoAPMessage(msg, "")
	if err == nil {
		t.Error("expected error for truncated token")
	}
}

// TestParseCoAPMessage_WithToken tests message with token.
func TestParseCoAPMessage_WithToken(t *testing.T) {
	options := map[int][]byte{
		optionGlobalDevID: []byte("SHSW-1#AABBCC#1"),
	}
	msg := buildCoAPMessage(4, codeStatus, options, []byte(`{}`))

	status, err := ParseCoAPMessage(msg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.DeviceID != "shsw-1-aabbcc" {
		t.Errorf("expected device ID shsw-1-aabbcc, got %s", status.DeviceID)
	}
}

// TestParseCoAPMessage_NoPayload tests message without payload.
func TestParseCoAPMessage_NoPayload(t *testing.T) {
	// Use buildCoAPMessage but without payload
	options := map[int][]byte{
		optionGlobalDevID: []byte("SHSW-1#ABC#1"), // Short value to use simple length encoding
	}

	// Build message without payload by using buildCoAPMessage and not adding payload
	optVal := options[optionGlobalDevID]

	// Extended delta for 3332: delta nibble = 14, delta-269 = 3063
	deltaExt := make([]byte, 2)
	binary.BigEndian.PutUint16(deltaExt, uint16(optionGlobalDevID-269))
	optBytes := make([]byte, 0, 1+len(deltaExt)+len(optVal))
	optBytes = append(optBytes, 0xE0|byte(len(optVal))) // delta=14, length as nibble (12 bytes fits)
	optBytes = append(optBytes, deltaExt...)
	optBytes = append(optBytes, optVal...)

	header := make([]byte, 0, 4+len(optBytes))
	header = append(header, 0x50, byte(codeStatus), 0x00, 0x01)
	msg := append(header, optBytes...)

	status, err := ParseCoAPMessage(msg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.DeviceID != "shsw-1-abc" {
		t.Errorf("expected device ID shsw-1-abc, got %s", status.DeviceID)
	}
}

// TestParseCoAPMessage_ExtendedDelta13 tests extended delta encoding (13).
func TestParseCoAPMessage_ExtendedDelta13(t *testing.T) {
	// Create two options: first at option 1 (If-Match), second uses delta=13 extended encoding
	// to reach option 14 (delta nibble=13, extended=0 means delta=13+0=13, so 1+13=14)

	// First option: option 1 (delta=1, length=1)
	opt1 := []byte{0x11, 0xAA} // delta=1, length=1, value=0xAA

	// Second option: URI-Path (option 11), delta from 1 is 10 (fits in nibble)
	// Actually let's use extended delta properly: delta nibble = 13, extended = 0 means delta = 13
	// So from option 1, delta 13 gets us to option 14 (Size1)
	// Let's test that extended delta=13 encoding parses correctly
	opt2 := make([]byte, 0, 5)      // 2 bytes header + 3 bytes "cit"
	opt2 = append(opt2, 0xD3, 0x00) // delta nibble=13, extended=0 (delta=13), length=3
	opt2 = append(opt2, []byte("cit")...)

	// Option 1 + delta 13 = option 14 (Size1), not URI-Path
	// The test verifies extended delta parsing works; we won't have URIPath set

	header := make([]byte, 0, 4+len(opt1)+len(opt2))
	header = append(header, 0x50, byte(codeStatus), 0x00, 0x01)
	msg := append(header, opt1...)
	msg = append(msg, opt2...)
	msg = append(msg, 0xFF)
	msg = append(msg, []byte(`{}`)...)

	status, err := ParseCoAPMessage(msg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Since neither option is URI-Path (11), URIPath should be empty
	// This test verifies extended delta encoding parses without error
	if status == nil {
		t.Error("expected status to be returned")
	}
}

// TestParseCoAPMessage_ExtendedLength13 tests extended length encoding (13).
func TestParseCoAPMessage_ExtendedLength13(t *testing.T) {
	// Build message with option value of length 20

	// Option 11 (URI-Path) with length 20
	longValue := make([]byte, 20)
	for i := range longValue {
		longValue[i] = 'a'
	}

	optBytes := make([]byte, 0, 2+len(longValue))
	optBytes = append(optBytes, 0xBD, 0x07) // delta=11, length=13+7=20
	optBytes = append(optBytes, longValue...)

	header := make([]byte, 0, 4+len(optBytes))
	header = append(header, 0x50, byte(codeStatus), 0x00, 0x01)
	msg := append(header, optBytes...)
	msg = append(msg, 0xFF)
	msg = append(msg, []byte(`{}`)...)

	status, err := ParseCoAPMessage(msg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "/" + string(longValue)
	if status.URIPath != expected {
		t.Errorf("expected URI path %s, got %s", expected, status.URIPath)
	}
}

// TestParseCoAPMessage_ExtendedDelta14 tests extended delta encoding (14).
func TestParseCoAPMessage_ExtendedDelta14(t *testing.T) {
	// Option 3332 requires delta=14 encoding (3332 >= 269)
	options := map[int][]byte{
		optionGlobalDevID: []byte("TEST#MAC#1"),
	}
	msg := buildCoAPMessage(0, codeStatus, options, []byte(`{}`))

	status, err := ParseCoAPMessage(msg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.DeviceType != "TEST" {
		t.Errorf("expected device type TEST, got %s", status.DeviceType)
	}
}

// TestParseCoAPMessage_TruncatedExtendedDelta tests truncated extended delta.
func TestParseCoAPMessage_TruncatedExtendedDelta(t *testing.T) {
	// Delta=13 but no extended byte
	msg := []byte{0x50, 0x1E, 0x00, 0x01, 0xD0}
	_, err := ParseCoAPMessage(msg, "")
	if err == nil {
		t.Error("expected error for truncated extended delta")
	}
}

// TestParseCoAPMessage_TruncatedExtendedDelta14 tests truncated delta=14.
func TestParseCoAPMessage_TruncatedExtendedDelta14(t *testing.T) {
	// Delta=14 but only 1 extended byte
	msg := []byte{0x50, 0x1E, 0x00, 0x01, 0xE0, 0x00}
	_, err := ParseCoAPMessage(msg, "")
	if err == nil {
		t.Error("expected error for truncated extended delta")
	}
}

// TestParseCoAPMessage_TruncatedExtendedLength tests truncated extended length.
func TestParseCoAPMessage_TruncatedExtendedLength(t *testing.T) {
	// Length=13 but no extended byte
	msg := []byte{0x50, 0x1E, 0x00, 0x01, 0x0D}
	_, err := ParseCoAPMessage(msg, "")
	if err == nil {
		t.Error("expected error for truncated extended length")
	}
}

// TestParseCoAPMessage_TruncatedExtendedLength14 tests truncated length=14.
func TestParseCoAPMessage_TruncatedExtendedLength14(t *testing.T) {
	// Length=14 but only 1 extended byte
	msg := []byte{0x50, 0x1E, 0x00, 0x01, 0x0E, 0x00}
	_, err := ParseCoAPMessage(msg, "")
	if err == nil {
		t.Error("expected error for truncated extended length")
	}
}

// TestParseCoAPMessage_InvalidLength15 tests invalid length=15.
func TestParseCoAPMessage_InvalidLength15(t *testing.T) {
	// Length=15 is reserved
	msg := []byte{0x50, 0x1E, 0x00, 0x01, 0x0F}
	_, err := ParseCoAPMessage(msg, "")
	if err == nil {
		t.Error("expected error for invalid length 15")
	}
}

// TestParseCoAPMessage_Delta15 tests delta=15 (payload marker or reserved).
func TestParseCoAPMessage_Delta15(t *testing.T) {
	// Delta=15 should be treated as end of options
	msg := []byte{0x50, 0x1E, 0x00, 0x01, 0xF0}
	status, err := ParseCoAPMessage(msg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should parse without error, just no options
	if status == nil {
		t.Error("expected status to be returned")
	}
}

// TestParseCoAPMessage_OptionValueExceedsLength tests option value overflow.
func TestParseCoAPMessage_OptionValueExceedsLength(t *testing.T) {
	// Option claims length 10 but message ends
	msg := []byte{0x50, 0x1E, 0x00, 0x01, 0x0A, 0x01, 0x02}
	_, err := ParseCoAPMessage(msg, "")
	if err == nil {
		t.Error("expected error for option value exceeding message")
	}
}

// TestParseCoAPMessage_MultipleURIPath tests multiple URI-Path options.
func TestParseCoAPMessage_MultipleURIPath(t *testing.T) {
	// Build message with multiple URI-Path segments

	// First URI-Path option (11): "cit"
	opt1 := make([]byte, 0, 4) // 1 byte header + 3 bytes "cit"
	opt1 = append(opt1, 0xB3)  // delta=11, length=3
	opt1 = append(opt1, []byte("cit")...)

	// Second URI-Path option (delta=0): "s"
	opt2 := make([]byte, 0, 2) // 1 byte header + 1 byte 's'
	opt2 = append(opt2, 0x01)  // delta=0, length=1
	opt2 = append(opt2, 's')

	header := make([]byte, 0, 4+len(opt1)+len(opt2))
	header = append(header, 0x50, byte(codeStatus), 0x00, 0x01)
	msg := append(header, opt1...)
	msg = append(msg, opt2...)
	msg = append(msg, 0xFF)
	msg = append(msg, []byte(`{}`)...)

	status, err := ParseCoAPMessage(msg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.URIPath != "/cit/s" {
		t.Errorf("expected URI path /cit/s, got %s", status.URIPath)
	}
}

// TestParseCoAPMessage_InvalidJSON tests invalid JSON payload.
func TestParseCoAPMessage_InvalidJSON(t *testing.T) {
	options := map[int][]byte{
		optionGlobalDevID: []byte("SHSW-1#AABBCC#1"),
	}
	msg := buildCoAPMessage(0, codeStatus, options, []byte(`{invalid}`))

	status, err := ParseCoAPMessage(msg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still parse device info from options
	if status.DeviceID != "shsw-1-aabbcc" {
		t.Errorf("expected device ID shsw-1-aabbcc, got %s", status.DeviceID)
	}

	// But sensors should be empty
	if len(status.Sensors) != 0 {
		t.Errorf("expected 0 sensors for invalid JSON, got %d", len(status.Sensors))
	}
}

// TestParseCoAPMessage_PayloadWithValidity tests V field in payload.
func TestParseCoAPMessage_PayloadWithValidity(t *testing.T) {
	options := map[int][]byte{
		optionGlobalDevID: []byte("SHSW-1#AABBCC#1"),
	}
	payload := []byte(`{"G":[[0,1101,1]],"V":30}`)
	msg := buildCoAPMessage(0, codeStatus, options, payload)

	status, err := ParseCoAPMessage(msg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.ValidityRaw != 30 {
		t.Errorf("expected validity 30, got %d", status.ValidityRaw)
	}
}

// TestParseCoAPMessage_PayloadWithSerial tests S field in payload.
func TestParseCoAPMessage_PayloadWithSerial(t *testing.T) {
	options := map[int][]byte{
		optionGlobalDevID: []byte("SHSW-1#AABBCC#1"),
	}
	payload := []byte(`{"G":[[0,1101,1]],"S":12345}`)
	msg := buildCoAPMessage(0, codeStatus, options, payload)

	status, err := ParseCoAPMessage(msg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Serial != 12345 {
		t.Errorf("expected serial 12345, got %d", status.Serial)
	}
}

// TestParseGlobalDevID tests parsing various GlobalDevID formats.
func TestParseGlobalDevID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantMAC  string
		wantID   string
		wantVer  int
	}{
		{
			name:     "full format",
			input:    "SHSW-PM#C45BBE6C2D3A#2",
			wantType: "SHSW-PM",
			wantMAC:  "C45BBE6C2D3A",
			wantID:   "shsw-pm-c45bbe6c2d3a",
			wantVer:  2,
		},
		{
			name:     "version 1",
			input:    "SHSW-1#AABBCC#1",
			wantType: "SHSW-1",
			wantMAC:  "AABBCC",
			wantID:   "shsw-1-aabbcc",
			wantVer:  1,
		},
		{
			name:     "type only",
			input:    "SHSW-25",
			wantType: "SHSW-25",
			wantMAC:  "",
			wantID:   "",
			wantVer:  0,
		},
		{
			name:     "type and MAC only",
			input:    "SHEM#112233",
			wantType: "SHEM",
			wantMAC:  "112233",
			wantID:   "shem-112233",
			wantVer:  0,
		},
		{
			name:     "invalid version",
			input:    "TEST#MAC#invalid",
			wantType: "TEST",
			wantMAC:  "MAC",
			wantID:   "test-mac",
			wantVer:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &CoIoTStatus{}
			parseGlobalDevID(tt.input, status)

			if status.DeviceType != tt.wantType {
				t.Errorf("DeviceType = %q, want %q", status.DeviceType, tt.wantType)
			}
			if status.DeviceMAC != tt.wantMAC {
				t.Errorf("DeviceMAC = %q, want %q", status.DeviceMAC, tt.wantMAC)
			}
			if status.DeviceID != tt.wantID {
				t.Errorf("DeviceID = %q, want %q", status.DeviceID, tt.wantID)
			}
			if status.Version != tt.wantVer {
				t.Errorf("Version = %d, want %d", status.Version, tt.wantVer)
			}
		})
	}
}

// TestParseCoIoTPayload tests JSON payload parsing.
func TestParseCoIoTPayload(t *testing.T) {
	tests := []struct {
		name        string
		payload     []byte
		wantSensors int
		wantSerial  int
		wantValid   int
	}{
		{
			name:        "with sensors",
			payload:     []byte(`{"G":[[0,1101,1],[0,4101,25.5],[1,1101,0]]}`),
			wantSensors: 3,
		},
		{
			name:        "with serial",
			payload:     []byte(`{"G":[],"S":999}`),
			wantSensors: 0,
			wantSerial:  999,
		},
		{
			name:        "with validity",
			payload:     []byte(`{"G":[],"V":60}`),
			wantSensors: 0,
			wantValid:   60,
		},
		{
			name:        "all fields",
			payload:     []byte(`{"G":[[0,1101,1]],"S":100,"V":30}`),
			wantSensors: 1,
			wantSerial:  100,
			wantValid:   30,
		},
		{
			name:        "invalid JSON",
			payload:     []byte(`{not valid}`),
			wantSensors: 0,
		},
		{
			name:        "empty G array",
			payload:     []byte(`{"G":[]}`),
			wantSensors: 0,
		},
		{
			name:        "malformed sensor entry",
			payload:     []byte(`{"G":[[0,1101]]}`), // missing value
			wantSensors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &CoIoTStatus{
				Sensors:   make(map[string]any),
				Actuators: make(map[string]any),
			}
			parseCoIoTPayload(tt.payload, status)

			if len(status.Sensors) != tt.wantSensors {
				t.Errorf("Sensors count = %d, want %d", len(status.Sensors), tt.wantSensors)
			}
			if status.Serial != tt.wantSerial {
				t.Errorf("Serial = %d, want %d", status.Serial, tt.wantSerial)
			}
			if status.ValidityRaw != tt.wantValid {
				t.Errorf("ValidityRaw = %d, want %d", status.ValidityRaw, tt.wantValid)
			}
		})
	}
}

// TestHandleMessage tests message handling with dispatch.
func TestHandleMessage(t *testing.T) {
	listener := NewCoIoTListener()

	var receivedDeviceID string
	var receivedStatus *CoIoTStatus
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {
		receivedDeviceID = deviceID
		receivedStatus = status
	})

	options := map[int][]byte{
		optionGlobalDevID: []byte("DEVICE#123456#2"),
	}
	msg := buildCoAPMessage(0, codeStatus, options, []byte(`{"G":[[0,1101,1]]}`))

	listener.handleMessage(msg, "10.0.0.1")

	// Give time for handler
	time.Sleep(10 * time.Millisecond)

	if receivedDeviceID != "device-123456" {
		t.Errorf("expected device ID device-123456, got %s", receivedDeviceID)
	}

	if receivedStatus == nil {
		t.Fatal("expected status to be received")
	}

	if receivedStatus.SourceAddr != "10.0.0.1" {
		t.Errorf("expected source 10.0.0.1, got %s", receivedStatus.SourceAddr)
	}
}

// TestHandleMessage_Invalid tests handling of invalid messages.
func TestHandleMessage_Invalid(t *testing.T) {
	listener := NewCoIoTListener()

	called := false
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) {
		called = true
	})

	// Invalid message (too short)
	listener.handleMessage([]byte{0x00, 0x01}, "")

	time.Sleep(10 * time.Millisecond)

	if called {
		t.Error("handler should not be called for invalid message")
	}
}

// TestMultipleHandlers tests multiple status handlers.
func TestMultipleHandlers(t *testing.T) {
	listener := NewCoIoTListener()

	count := 0
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) { count++ })
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) { count++ })
	listener.OnStatus(func(deviceID string, status *CoIoTStatus) { count++ })

	options := map[int][]byte{
		optionGlobalDevID: []byte("TEST#MAC#1"),
	}
	msg := buildCoAPMessage(0, codeStatus, options, []byte(`{}`))

	listener.handleMessage(msg, "")

	time.Sleep(10 * time.Millisecond)

	if count != 3 {
		t.Errorf("expected 3 handler calls, got %d", count)
	}
}

// TestCoIoTDescription tests CoIoTDescription struct.
func TestCoIoTDescription(t *testing.T) {
	desc := &CoIoTDescription{
		DeviceID:   "shellyem-AABBCC",
		DeviceType: "SHEM",
		Blocks: []CoIoTBlock{
			{ID: 0, Description: "Relay 0"},
			{ID: 1, Description: "Relay 1"},
		},
		Sensors: []CoIoTSensor{
			{ID: 3101, Type: "W", Description: "Power", Unit: "W", Block: 0},
			{ID: 3110, Type: "Wh", Description: "Energy", Unit: "Wh", Block: 0, Range: "0/10000000"},
		},
		Actuators: []CoIoTActuator{
			{ID: 1101, Type: "S", Description: "Relay", Block: 0},
		},
	}

	if desc.DeviceID != "shellyem-AABBCC" {
		t.Errorf("expected device ID shellyem-AABBCC, got %s", desc.DeviceID)
	}

	if len(desc.Blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(desc.Blocks))
	}

	if len(desc.Sensors) != 2 {
		t.Errorf("expected 2 sensors, got %d", len(desc.Sensors))
	}

	if len(desc.Actuators) != 1 {
		t.Errorf("expected 1 actuator, got %d", len(desc.Actuators))
	}

	if desc.Sensors[0].Unit != "W" {
		t.Errorf("expected unit W, got %s", desc.Sensors[0].Unit)
	}

	if desc.Sensors[1].Range != "0/10000000" {
		t.Errorf("expected range 0/10000000, got %s", desc.Sensors[1].Range)
	}
}

// TestCoIoTBlock tests CoIoTBlock struct.
func TestCoIoTBlock(t *testing.T) {
	block := CoIoTBlock{
		ID:          0,
		Description: "Relay",
	}

	if block.ID != 0 {
		t.Errorf("expected ID 0, got %d", block.ID)
	}

	if block.Description != "Relay" {
		t.Errorf("expected description Relay, got %s", block.Description)
	}
}

// TestCoIoTSensor tests CoIoTSensor struct.
func TestCoIoTSensor(t *testing.T) {
	sensor := CoIoTSensor{
		ID:          3101,
		Type:        "W",
		Description: "Power",
		Unit:        "W",
		Block:       0,
		Links:       []int{1, 2},
		Range:       "0/10000",
	}

	if sensor.Type != "W" {
		t.Errorf("expected type W, got %s", sensor.Type)
	}

	if sensor.Unit != "W" {
		t.Errorf("expected unit W, got %s", sensor.Unit)
	}

	if len(sensor.Links) != 2 {
		t.Errorf("expected 2 links, got %d", len(sensor.Links))
	}

	if sensor.Range != "0/10000" {
		t.Errorf("expected range 0/10000, got %s", sensor.Range)
	}
}

// TestCoIoTActuator tests CoIoTActuator struct.
func TestCoIoTActuator(t *testing.T) {
	actuator := CoIoTActuator{
		ID:          1101,
		Type:        "S",
		Description: "Relay",
		Block:       0,
		Links:       []int{0},
		Range:       "0/1",
	}

	if actuator.Type != "S" {
		t.Errorf("expected type S, got %s", actuator.Type)
	}

	if actuator.Description != "Relay" {
		t.Errorf("expected description Relay, got %s", actuator.Description)
	}

	if len(actuator.Links) != 1 {
		t.Errorf("expected 1 link, got %d", len(actuator.Links))
	}
}

// TestGetDeviceDescription tests GetDeviceDescription function.
func TestGetDeviceDescription(t *testing.T) {
	// This should return an error since it requires HTTP
	_, err := GetDeviceDescription("192.168.1.100")
	if err == nil {
		t.Error("expected error")
	}
}

// TestParseCoIoTDescription tests parsing description JSON.
func TestParseCoIoTDescription(t *testing.T) {
	jsonData := []byte(`{
		"id": "shellyem-AABBCC",
		"type": "SHEM",
		"blk": [
			{"I": 0, "D": "Channel 0"},
			{"I": 1, "D": "Channel 1"}
		],
		"sen": [
			{"I": 3101, "T": "W", "D": "Power", "U": "W", "B": 0}
		],
		"act": [
			{"I": 1101, "T": "S", "D": "Relay", "B": 0}
		]
	}`)

	desc, err := ParseCoIoTDescription(jsonData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if desc.DeviceID != "shellyem-AABBCC" {
		t.Errorf("expected ID shellyem-AABBCC, got %s", desc.DeviceID)
	}

	if desc.DeviceType != "SHEM" {
		t.Errorf("expected type SHEM, got %s", desc.DeviceType)
	}

	if len(desc.Blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(desc.Blocks))
	}

	if len(desc.Sensors) != 1 {
		t.Errorf("expected 1 sensor, got %d", len(desc.Sensors))
	}

	if len(desc.Actuators) != 1 {
		t.Errorf("expected 1 actuator, got %d", len(desc.Actuators))
	}
}

// TestParseCoIoTDescription_Invalid tests invalid JSON.
func TestParseCoIoTDescription_Invalid(t *testing.T) {
	_, err := ParseCoIoTDescription([]byte(`{invalid}`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestCoIoTConstants tests CoIoT constants.
func TestCoIoTConstants(t *testing.T) {
	if CoIoTMulticastAddr != "224.0.1.187" {
		t.Errorf("expected multicast addr 224.0.1.187, got %s", CoIoTMulticastAddr)
	}

	if CoIoTPort != 5683 {
		t.Errorf("expected port 5683, got %d", CoIoTPort)
	}

	if DefaultCoIoTPeriod != 15 {
		t.Errorf("expected period 15, got %d", DefaultCoIoTPeriod)
	}

	if optionURIPath != 11 {
		t.Errorf("expected URI-Path option 11, got %d", optionURIPath)
	}

	if optionGlobalDevID != 3332 {
		t.Errorf("expected GlobalDevID option 3332, got %d", optionGlobalDevID)
	}

	if codeStatus != 30 {
		t.Errorf("expected status code 30, got %d", codeStatus)
	}
}

// TestSensorIDToDescription tests sensor ID mapping.
func TestSensorIDToDescription(t *testing.T) {
	tests := []struct {
		id   int
		want string
	}{
		{1101, "Relay State"},
		{2101, "Input State"},
		{3101, "Active Power"},
		{3104, "Voltage"},
		{4101, "Power (W)"},
		{6109, "Status (Error)"},
		{9101, "Temperature"},
		{9999, ""}, // Unknown
	}

	for _, tt := range tests {
		got := SensorIDToDescription[tt.id]
		if got != tt.want {
			t.Errorf("SensorIDToDescription[%d] = %q, want %q", tt.id, got, tt.want)
		}
	}
}
