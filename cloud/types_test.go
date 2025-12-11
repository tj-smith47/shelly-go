package cloud

import (
	"encoding/json"
	"testing"
)

func TestDeviceStatusJSON(t *testing.T) {
	data := `{
		"id": "device123",
		"online": true,
		"_dev_info": {
			"code": "SHSW-1",
			"gen": 1,
			"online": true
		},
		"status": {"relay": true}
	}`

	var status DeviceStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if status.ID != "device123" {
		t.Errorf("ID = %v, want device123", status.ID)
	}
	if !status.Online {
		t.Error("Online = false, want true")
	}
	if status.DevInfo == nil {
		t.Fatal("DevInfo is nil")
	}
	if status.DevInfo.Code != "SHSW-1" {
		t.Errorf("DevInfo.Code = %v, want SHSW-1", status.DevInfo.Code)
	}
	if status.DevInfo.Generation != 1 {
		t.Errorf("DevInfo.Generation = %v, want 1", status.DevInfo.Generation)
	}
}

func TestDeviceJSON(t *testing.T) {
	data := `{
		"id": "device123",
		"name": "Living Room Light",
		"type": "SHSW-1",
		"model": "shelly1",
		"mac": "AA:BB:CC:DD:EE:FF",
		"gen": 1,
		"fw_ver": "1.0.0",
		"online": true,
		"last_seen": 1609459200,
		"cloud_enabled": true
	}`

	var device Device
	if err := json.Unmarshal([]byte(data), &device); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if device.ID != "device123" {
		t.Errorf("ID = %v, want device123", device.ID)
	}
	if device.Name != "Living Room Light" {
		t.Errorf("Name = %v, want Living Room Light", device.Name)
	}
	if device.Type != "SHSW-1" {
		t.Errorf("Type = %v, want SHSW-1", device.Type)
	}
	if device.Generation != 1 {
		t.Errorf("Generation = %v, want 1", device.Generation)
	}
	if !device.Online {
		t.Error("Online = false, want true")
	}
	if !device.CloudEnabled {
		t.Error("CloudEnabled = false, want true")
	}
}

func TestAllDevicesResponseJSON(t *testing.T) {
	data := `{
		"isok": true,
		"data": {
			"devices_status": {
				"device1": {
					"id": "device1",
					"online": true
				},
				"device2": {
					"id": "device2",
					"online": false
				}
			}
		}
	}`

	var resp AllDevicesResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !resp.IsOK {
		t.Error("IsOK = false, want true")
	}
	if resp.Data == nil {
		t.Fatal("Data is nil")
	}
	if len(resp.Data.DevicesStatus) != 2 {
		t.Errorf("DevicesStatus count = %v, want 2", len(resp.Data.DevicesStatus))
	}
	if resp.Data.DevicesStatus["device1"] == nil {
		t.Error("DevicesStatus[device1] is nil")
	}
	if !resp.Data.DevicesStatus["device1"].Online {
		t.Error("DevicesStatus[device1].Online = false, want true")
	}
}

func TestControlRequestJSON(t *testing.T) {
	brightness := 50
	req := ControlRequest{
		DeviceID:   "device123",
		Channel:    0,
		Turn:       "on",
		Brightness: &brightness,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed ControlRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.DeviceID != "device123" {
		t.Errorf("DeviceID = %v, want device123", parsed.DeviceID)
	}
	if parsed.Turn != "on" {
		t.Errorf("Turn = %v, want on", parsed.Turn)
	}
	if parsed.Brightness == nil || *parsed.Brightness != 50 {
		t.Error("Brightness mismatch")
	}
}

func TestGroupControlRequestJSON(t *testing.T) {
	brightness := 75
	req := GroupControlRequest{
		Switches: []GroupSwitch{
			{IDs: []string{"device1_0", "device2_0"}, Turn: "on"},
		},
		Lights: []GroupLight{
			{IDs: []string{"device3_0"}, Turn: "on", Brightness: &brightness},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed GroupControlRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(parsed.Switches) != 1 {
		t.Errorf("Switches count = %v, want 1", len(parsed.Switches))
	}
	if len(parsed.Switches[0].IDs) != 2 {
		t.Errorf("Switches[0].IDs count = %v, want 2", len(parsed.Switches[0].IDs))
	}
	if len(parsed.Lights) != 1 {
		t.Errorf("Lights count = %v, want 1", len(parsed.Lights))
	}
}

func TestV2DevicesRequestJSON(t *testing.T) {
	req := V2DevicesRequest{
		IDs:    []string{"device1", "device2"},
		Select: []string{"status", "settings"},
		Pick:   []string{"relay:0", "input:0"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed V2DevicesRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(parsed.IDs) != 2 {
		t.Errorf("IDs count = %v, want 2", len(parsed.IDs))
	}
	if len(parsed.Select) != 2 {
		t.Errorf("Select count = %v, want 2", len(parsed.Select))
	}
}

func TestWebSocketMessageJSON(t *testing.T) {
	data := `{
		"event": "Shelly:StatusChange",
		"device_id": "device123",
		"channel": 0,
		"status": {"output": true},
		"ts": 1609459200
	}`

	var msg WebSocketMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if msg.Event != EventDeviceStatusChange {
		t.Errorf("Event = %v, want %v", msg.Event, EventDeviceStatusChange)
	}
	if msg.DeviceID != "device123" {
		t.Errorf("DeviceID = %v, want device123", msg.DeviceID)
	}
	if msg.Channel != 0 {
		t.Errorf("Channel = %v, want 0", msg.Channel)
	}
	if msg.Timestamp != 1609459200 {
		t.Errorf("Timestamp = %v, want 1609459200", msg.Timestamp)
	}
}

func TestLoginRequestJSON(t *testing.T) {
	req := LoginRequest{
		Email:    "test@example.com",
		Password: "hashedpassword",
		ClientID: "shelly-diy",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed LoginRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.Email != "test@example.com" {
		t.Errorf("Email = %v, want test@example.com", parsed.Email)
	}
	if parsed.ClientID != "shelly-diy" {
		t.Errorf("ClientID = %v, want shelly-diy", parsed.ClientID)
	}
}

func TestLoginResponseJSON(t *testing.T) {
	data := `{
		"isok": true,
		"data": {
			"token": "jwt-token-here",
			"user_api_url": "https://shelly-49-eu.shelly.cloud"
		}
	}`

	var resp LoginResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !resp.IsOK {
		t.Error("IsOK = false, want true")
	}
	if resp.Data == nil {
		t.Fatal("Data is nil")
	}
	if resp.Data.Token != "jwt-token-here" {
		t.Errorf("Token = %v, want jwt-token-here", resp.Data.Token)
	}
	if resp.Data.UserAPIURL != "https://shelly-49-eu.shelly.cloud" {
		t.Errorf("UserAPIURL = %v, want https://shelly-49-eu.shelly.cloud", resp.Data.UserAPIURL)
	}
}

func TestLoginResponseErrorJSON(t *testing.T) {
	data := `{
		"isok": false,
		"errors": ["Invalid credentials", "Account locked"]
	}`

	var resp LoginResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if resp.IsOK {
		t.Error("IsOK = true, want false")
	}
	if len(resp.Errors) != 2 {
		t.Errorf("Errors count = %v, want 2", len(resp.Errors))
	}
}

func TestEventTypeConstants(t *testing.T) {
	// Verify event type constants are defined
	if EventDeviceOnline != "Shelly:Online" {
		t.Errorf("EventDeviceOnline = %v, want Shelly:Online", EventDeviceOnline)
	}
	if EventDeviceOffline != "Shelly:Offline" {
		t.Errorf("EventDeviceOffline = %v, want Shelly:Offline", EventDeviceOffline)
	}
	if EventDeviceStatusChange != "Shelly:StatusChange" {
		t.Errorf("EventDeviceStatusChange = %v, want Shelly:StatusChange", EventDeviceStatusChange)
	}
	if EventNotifyStatus != "NotifyStatus" {
		t.Errorf("EventNotifyStatus = %v, want NotifyStatus", EventNotifyStatus)
	}
	if EventNotifyFullStatus != "NotifyFullStatus" {
		t.Errorf("EventNotifyFullStatus = %v, want NotifyFullStatus", EventNotifyFullStatus)
	}
	if EventNotifyEvent != "NotifyEvent" {
		t.Errorf("EventNotifyEvent = %v, want NotifyEvent", EventNotifyEvent)
	}
}

func TestConstants(t *testing.T) {
	if ClientIDDIY != "shelly-diy" {
		t.Errorf("ClientIDDIY = %v, want shelly-diy", ClientIDDIY)
	}
	if DefaultWSPort != 6113 {
		t.Errorf("DefaultWSPort = %v, want 6113", DefaultWSPort)
	}
}
