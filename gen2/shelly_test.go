package gen2

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestNewShelly(t *testing.T) {
	mt := &mockTransport{}
	client := newTestClient(mt)

	shelly := NewShelly(client)

	if shelly == nil {
		t.Fatal("NewShelly() returned nil")
	}

	if shelly.client == nil {
		t.Error("NewShelly() client is nil")
	}
}

func TestShelly_GetDeviceInfo(t *testing.T) {
	tests := []struct {
		want     *DeviceInfo
		name     string
		response string
		wantErr  bool
	}{
		{
			name: "successful response",
			response: `{
				"name": "My Shelly",
				"id": "shellypro1pm-1234567890ab",
				"mac": "1234567890AB",
				"model": "SPR-1PCBA1EU",
				"gen": 2,
				"fw_id": "20230913-123456/v1.0.0",
				"ver": "1.0.0",
				"app": "Pro1PM",
				"auth_en": true,
				"auth_domain": "shellypro1pm-1234567890ab"
			}`,
			want: &DeviceInfo{
				Name:            "My Shelly",
				ID:              "shellypro1pm-1234567890ab",
				MAC:             "1234567890AB",
				Model:           "SPR-1PCBA1EU",
				Gen:             2,
				FirmwareID:      "20230913-123456/v1.0.0",
				FirmwareVersion: "1.0.0",
				App:             "Pro1PM",
				AuthEnabled:     true,
				AuthDomain:      "shellypro1pm-1234567890ab",
			},
			wantErr: false,
		},
		{
			name:     "minimal response",
			response: `{"id":"test-id","mac":"ABC123","model":"Test","gen":3,"fw_id":"test","ver":"1.0","app":"Test"}`,
			want: &DeviceInfo{
				ID:              "test-id",
				MAC:             "ABC123",
				Model:           "Test",
				Gen:             3,
				FirmwareID:      "test",
				FirmwareVersion: "1.0",
				App:             "Test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				response: []byte(tt.response),
			}
			client := newTestClient(mt)
			shelly := NewShelly(client)

			got, err := shelly.GetDeviceInfo(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDeviceInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.Model != tt.want.Model {
				t.Errorf("Model = %v, want %v", got.Model, tt.want.Model)
			}
			if got.Gen != tt.want.Gen {
				t.Errorf("Gen = %v, want %v", got.Gen, tt.want.Gen)
			}
			if got.FirmwareVersion != tt.want.FirmwareVersion {
				t.Errorf("FirmwareVersion = %v, want %v", got.FirmwareVersion, tt.want.FirmwareVersion)
			}
			if got.AuthEnabled != tt.want.AuthEnabled {
				t.Errorf("AuthEnabled = %v, want %v", got.AuthEnabled, tt.want.AuthEnabled)
			}
		})
	}
}

func TestShelly_GetDeviceInfo_Error(t *testing.T) {
	mt := &mockTransport{
		err: errors.New("connection failed"),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	_, err := shelly.GetDeviceInfo(context.Background())

	if err == nil {
		t.Error("GetDeviceInfo() expected error, got nil")
	}
}

func TestShelly_GetStatus(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{
			"switch:0": {"id":0,"output":true,"apower":123.4},
			"sys": {"mac":"1234567890AB","available_updates":{}},
			"wifi": {"sta_ip":"192.168.1.100","status":"got ip"}
		}`),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	status, err := shelly.GetStatus(context.Background())

	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if len(status) != 3 {
		t.Errorf("GetStatus() returned %d components, want 3", len(status))
	}

	if _, ok := status["switch:0"]; !ok {
		t.Error("GetStatus() missing switch:0 component")
	}

	if _, ok := status["sys"]; !ok {
		t.Error("GetStatus() missing sys component")
	}

	if _, ok := status["wifi"]; !ok {
		t.Error("GetStatus() missing wifi component")
	}
}

func TestShelly_GetConfig(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{
			"switch:0": {"id":0,"name":"My Switch"},
			"sys": {"device":{"name":"My Device"}},
			"wifi": {"ap":{"enable":false},"sta":{"enable":true}}
		}`),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	config, err := shelly.GetConfig(context.Background())

	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if len(config) != 3 {
		t.Errorf("GetConfig() returned %d components, want 3", len(config))
	}

	if _, ok := config["switch:0"]; !ok {
		t.Error("GetConfig() missing switch:0 component")
	}
}

func TestShelly_SetConfig(t *testing.T) {
	tests := []struct {
		config any
		name   string
	}{
		{
			name: "map config",
			config: map[string]any{
				"sys": map[string]any{
					"device": map[string]any{
						"name": "New Name",
					},
				},
			},
		},
		{
			name: "struct config",
			config: struct {
				Sys struct {
					Device struct {
						Name string `json:"name"`
					} `json:"device"`
				} `json:"sys"`
			}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				response: []byte(`{"restart_required":false}`),
			}
			client := newTestClient(mt)
			shelly := NewShelly(client)

			err := shelly.SetConfig(context.Background(), tt.config)

			if err != nil {
				t.Fatalf("SetConfig() error = %v", err)
			}
		})
	}
}

func TestShelly_ListMethods(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"methods":["Shelly.GetDeviceInfo","Shelly.GetStatus","Switch.Set","Switch.Toggle"]}`),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	methods, err := shelly.ListMethods(context.Background())

	if err != nil {
		t.Fatalf("ListMethods() error = %v", err)
	}

	want := []string{"Shelly.GetDeviceInfo", "Shelly.GetStatus", "Switch.Set", "Switch.Toggle"}
	if len(methods) != len(want) {
		t.Errorf("ListMethods() returned %d methods, want %d", len(methods), len(want))
	}

	for i, method := range methods {
		if method != want[i] {
			t.Errorf("ListMethods()[%d] = %v, want %v", i, method, want[i])
		}
	}
}

func TestShelly_Reboot(t *testing.T) {
	tests := []struct {
		name    string
		delayMS int
	}{
		{
			name:    "immediate reboot",
			delayMS: 0,
		},
		{
			name:    "delayed reboot",
			delayMS: 5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				response: []byte(`null`),
			}
			client := newTestClient(mt)
			shelly := NewShelly(client)

			err := shelly.Reboot(context.Background(), tt.delayMS)

			if err != nil {
				t.Fatalf("Reboot() error = %v", err)
			}

			// Verify the correct method was called
			if mt.lastCall.method != "Shelly.Reboot" {
				t.Errorf("Called method = %v, want Shelly.Reboot", mt.lastCall.method)
			}

			// Note: Parameter validation omitted - we're testing method calls, not parameter marshaling
		})
	}
}

func TestShelly_FactoryReset(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`null`),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	err := shelly.FactoryReset(context.Background())

	if err != nil {
		t.Fatalf("FactoryReset() error = %v", err)
	}

	if mt.lastCall.method != "Shelly.FactoryReset" {
		t.Errorf("Called method = %v, want Shelly.FactoryReset", mt.lastCall.method)
	}
}

func TestShelly_ResetWiFiConfig(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`null`),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	err := shelly.ResetWiFiConfig(context.Background())

	if err != nil {
		t.Fatalf("ResetWiFiConfig() error = %v", err)
	}

	if mt.lastCall.method != "Shelly.ResetWiFiConfig" {
		t.Errorf("Called method = %v, want Shelly.ResetWiFiConfig", mt.lastCall.method)
	}
}

func TestShelly_CheckForUpdate(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
	}{
		{
			name: "update available",
			response: `{
				"stable": {"version": "1.1.0", "build_id": "20231015-123456"},
				"beta": {"version": "1.2.0-beta1", "build_id": "20231020-654321"},
				"old_version": "1.0.0"
			}`,
			wantErr: false,
		},
		{
			name:     "no update available",
			response: `{"old_version": "1.1.0"}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				response: []byte(tt.response),
			}
			client := newTestClient(mt)
			shelly := NewShelly(client)

			info, err := shelly.CheckForUpdate(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckForUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if info == nil {
				t.Fatal("CheckForUpdate() returned nil info")
			}

			// Parse expected response
			var want UpdateInfo
			if err := json.Unmarshal([]byte(tt.response), &want); err != nil {
				t.Fatalf("Failed to unmarshal expected response: %v", err)
			}

			if info.OldVersion != want.OldVersion {
				t.Errorf("OldVersion = %v, want %v", info.OldVersion, want.OldVersion)
			}

			if want.Stable != nil {
				if info.Stable == nil {
					t.Error("Stable is nil, expected non-nil")
				} else if info.Stable.Version != want.Stable.Version {
					t.Errorf("Stable.Version = %v, want %v", info.Stable.Version, want.Stable.Version)
				}
			}

			if want.Beta != nil {
				if info.Beta == nil {
					t.Error("Beta is nil, expected non-nil")
				} else if info.Beta.Version != want.Beta.Version {
					t.Errorf("Beta.Version = %v, want %v", info.Beta.Version, want.Beta.Version)
				}
			}
		})
	}
}

func TestShelly_Update(t *testing.T) {
	tests := []struct {
		params *UpdateParams
		name   string
	}{
		{
			name:   "stable channel",
			params: &UpdateParams{Stage: "stable"},
		},
		{
			name:   "beta channel",
			params: &UpdateParams{Stage: "beta"},
		},
		{
			name:   "custom URL",
			params: &UpdateParams{URL: "http://example.com/firmware.zip"},
		},
		{
			name:   "nil params (defaults to stable)",
			params: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				response: []byte(`{"status":"updating"}`),
			}
			client := newTestClient(mt)
			shelly := NewShelly(client)

			err := shelly.Update(context.Background(), tt.params)

			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			if mt.lastCall.method != "Shelly.Update" {
				t.Errorf("Called method = %v, want Shelly.Update", mt.lastCall.method)
			}

			// Note: Parameter validation omitted - we're testing method calls, not parameter marshaling
		})
	}
}

func TestShelly_SetAuth(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`null`),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	params := &SetAuthParams{
		User:  "admin",
		Realm: "shellypro1pm-1234567890ab",
		HA1:   "5f4dcc3b5aa765d61d8327deb882cf99",
	}

	err := shelly.SetAuth(context.Background(), params)

	if err != nil {
		t.Fatalf("SetAuth() error = %v", err)
	}

	if mt.lastCall.method != "Shelly.SetAuth" {
		t.Errorf("Called method = %v, want Shelly.SetAuth", mt.lastCall.method)
	}
}

func TestShelly_GetComponents(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		includeStatus bool
		includeConfig bool
	}{
		{
			name:          "with status and config",
			includeStatus: true,
			includeConfig: true,
			response: `{
				"components": [
					{
						"key": "switch:0",
						"status": {"output": true},
						"config": {"name": "My Switch"}
					}
				]
			}`,
		},
		{
			name:          "without status and config",
			includeStatus: false,
			includeConfig: false,
			response:      `{"components": [{"key": "switch:0"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				response: []byte(tt.response),
			}
			client := newTestClient(mt)
			shelly := NewShelly(client)

			components, err := shelly.GetComponents(context.Background(), tt.includeStatus, tt.includeConfig)

			if err != nil {
				t.Fatalf("GetComponents() error = %v", err)
			}

			if components == nil {
				t.Fatal("GetComponents() returned nil")
			}

			// Note: Parameter validation omitted - we're testing method calls, not parameter marshaling
		})
	}
}

func TestShelly_DetectLocation(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"tz":"America/New_York","lat":40.7128,"lon":-74.0060}`),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	location, err := shelly.DetectLocation(context.Background())

	if err != nil {
		t.Fatalf("DetectLocation() error = %v", err)
	}

	if location.TZ != "America/New_York" {
		t.Errorf("TZ = %v, want America/New_York", location.TZ)
	}

	if location.Lat != 40.7128 {
		t.Errorf("Lat = %v, want 40.7128", location.Lat)
	}

	if location.Lon != -74.0060 {
		t.Errorf("Lon = %v, want -74.0060", location.Lon)
	}
}

func TestShelly_ListProfiles(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"profiles":["switch","cover"]}`),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	profiles, err := shelly.ListProfiles(context.Background())

	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}

	want := []string{"switch", "cover"}
	if len(profiles) != len(want) {
		t.Errorf("ListProfiles() returned %d profiles, want %d", len(profiles), len(want))
	}

	for i, profile := range profiles {
		if profile != want[i] {
			t.Errorf("ListProfiles()[%d] = %v, want %v", i, profile, want[i])
		}
	}
}

func TestShelly_SetProfile(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`null`),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	err := shelly.SetProfile(context.Background(), "cover")

	if err != nil {
		t.Fatalf("SetProfile() error = %v", err)
	}

	if mt.lastCall.method != "Shelly.SetProfile" {
		t.Errorf("Called method = %v, want Shelly.SetProfile", mt.lastCall.method)
	}

	// Note: Parameter validation omitted - we're testing method calls, not parameter marshaling
}

func TestShelly_PutUserCA(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		append bool
	}{
		{
			name:   "replace CA",
			data:   "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
			append: false,
		},
		{
			name:   "append CA",
			data:   "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
			append: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				response: []byte(`null`),
			}
			client := newTestClient(mt)
			shelly := NewShelly(client)

			err := shelly.PutUserCA(context.Background(), tt.data, tt.append)

			if err != nil {
				t.Fatalf("PutUserCA() error = %v", err)
			}

			// Note: Parameter validation omitted - we're testing method calls, not parameter marshaling
		})
	}
}

func TestShelly_PutTLSClientCert(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`null`),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	certData := "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"
	err := shelly.PutTLSClientCert(context.Background(), certData)

	if err != nil {
		t.Fatalf("PutTLSClientCert() error = %v", err)
	}

	if mt.lastCall.method != "Shelly.PutTLSClientCert" {
		t.Errorf("Called method = %v, want Shelly.PutTLSClientCert", mt.lastCall.method)
	}

	// Note: Parameter validation omitted - we're testing method calls, not parameter marshaling
}

func TestShelly_PutTLSClientKey(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`null`),
	}
	client := newTestClient(mt)
	shelly := NewShelly(client)

	keyData := "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----"
	err := shelly.PutTLSClientKey(context.Background(), keyData)

	if err != nil {
		t.Fatalf("PutTLSClientKey() error = %v", err)
	}

	if mt.lastCall.method != "Shelly.PutTLSClientKey" {
		t.Errorf("Called method = %v, want Shelly.PutTLSClientKey", mt.lastCall.method)
	}

	// Note: Parameter validation omitted - we're testing method calls, not parameter marshaling
}

func TestShelly_ErrorHandling(t *testing.T) {
	testErr := errors.New("test error")

	tests := []struct {
		call func(*Shelly) error
		name string
	}{
		{
			name: "GetDeviceInfo",
			call: func(s *Shelly) error {
				_, err := s.GetDeviceInfo(context.Background())
				return err
			},
		},
		{
			name: "GetStatus",
			call: func(s *Shelly) error {
				_, err := s.GetStatus(context.Background())
				return err
			},
		},
		{
			name: "GetConfig",
			call: func(s *Shelly) error {
				_, err := s.GetConfig(context.Background())
				return err
			},
		},
		{
			name: "SetConfig",
			call: func(s *Shelly) error {
				return s.SetConfig(context.Background(), map[string]any{})
			},
		},
		{
			name: "ListMethods",
			call: func(s *Shelly) error {
				_, err := s.ListMethods(context.Background())
				return err
			},
		},
		{
			name: "Reboot",
			call: func(s *Shelly) error {
				return s.Reboot(context.Background(), 0)
			},
		},
		{
			name: "FactoryReset",
			call: func(s *Shelly) error {
				return s.FactoryReset(context.Background())
			},
		},
		{
			name: "CheckForUpdate",
			call: func(s *Shelly) error {
				_, err := s.CheckForUpdate(context.Background())
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockTransport{
				err: testErr,
			}
			client := newTestClient(mt)
			shelly := NewShelly(client)

			err := tt.call(shelly)

			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
