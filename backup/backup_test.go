package backup

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/rpc"
)

// mockTransport implements rpc.Transport for testing.
type mockTransport struct {
	callFunc  func(ctx context.Context, method string, params any) (json.RawMessage, error)
	closeFunc func() error
}

func (m *mockTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if m.callFunc != nil {
		return m.callFunc(ctx, method, params)
	}
	return nil, nil
}

func (m *mockTransport) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// jsonrpcResponse wraps a result in a JSON-RPC response envelope.
func jsonrpcResponse(result string) (json.RawMessage, error) {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  json.RawMessage(result),
	}
	return json.Marshal(response)
}

var errTest = errors.New("test error")

func TestNew(t *testing.T) {
	transport := &mockTransport{}
	client := rpc.NewClient(transport)
	mgr := New(client)

	if mgr == nil {
		t.Fatal("New returned nil")
	}
	if mgr.client != client {
		t.Error("client not set correctly")
	}
}

func TestManager_Export(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"shellyplus1-123456","model":"SNSW-001X16EU","gen":2,"ver":"1.0.0"}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Kitchen"}}}`)
			case "Cloud.GetConfig":
				return jsonrpcResponse(`{"enable":true}`)
			case "BLE.GetConfig":
				return jsonrpcResponse(`{"enable":true}`)
			case "MQTT.GetConfig":
				return jsonrpcResponse(`{"enable":false}`)
			case "Webhook.List":
				return jsonrpcResponse(`{"hooks":[]}`)
			case "Schedule.List":
				return jsonrpcResponse(`{"jobs":[]}`)
			case "Script.List":
				return jsonrpcResponse(`{"scripts":[]}`)
			case "KVS.List":
				return jsonrpcResponse(`{"keys":[]}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	data, err := mgr.Export(context.Background(), nil)
	if err != nil {
		t.Errorf("Export() error = %v", err)
		return
	}

	if len(data) == 0 {
		t.Error("Export() returned empty data")
	}

	// Parse the backup
	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Errorf("Failed to parse backup: %v", err)
		return
	}

	if backup.Version != BackupVersion {
		t.Errorf("Version = %d, want %d", backup.Version, BackupVersion)
	}
	if backup.DeviceInfo == nil {
		t.Error("DeviceInfo is nil")
	}
	if backup.DeviceInfo.ID != "shellyplus1-123456" {
		t.Errorf("DeviceInfo.ID = %s, want shellyplus1-123456", backup.DeviceInfo.ID)
	}
}

func TestManager_Export_WithWiFi(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test"}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{}`)
			case "WiFi.GetConfig":
				return jsonrpcResponse(`{"sta":{"ssid":"TestNetwork"}}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	opts := &ExportOptions{
		IncludeWiFi: true,
	}

	data, err := mgr.Export(context.Background(), opts)
	if err != nil {
		t.Errorf("Export() error = %v", err)
		return
	}

	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Fatalf("Failed to unmarshal backup: %v", err)
	}

	if backup.WiFi == nil {
		t.Error("WiFi should be included in backup")
	}
}

func TestManager_Export_DeviceInfoError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method == "Shelly.GetDeviceInfo" {
				return nil, errTest
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	_, err := mgr.Export(context.Background(), nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestManager_Export_ConfigError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test"}`)
			case "Shelly.GetConfig":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	_, err := mgr.Export(context.Background(), nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestManager_Export_WithScripts(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test"}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{}`)
			case "Script.List":
				return jsonrpcResponse(`{"scripts":[{"id":1,"name":"test","enable":true}]}`)
			case "Script.GetCode":
				return jsonrpcResponse(`{"data":"print('hello')"}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	opts := &ExportOptions{
		IncludeScripts: true,
	}

	data, err := mgr.Export(context.Background(), opts)
	if err != nil {
		t.Errorf("Export() error = %v", err)
		return
	}

	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Fatalf("Failed to unmarshal backup: %v", err)
	}

	if len(backup.Scripts) == 0 {
		t.Error("Scripts should be included in backup")
	}
	if backup.Scripts[0].Code != "print('hello')" {
		t.Errorf("Script code = %s, want print('hello')", backup.Scripts[0].Code)
	}
}

func TestManager_Export_WithKVS(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test"}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{}`)
			case "KVS.List":
				return jsonrpcResponse(`{"keys":["key1","key2"]}`)
			case "KVS.Get":
				return jsonrpcResponse(`{"value":"test"}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	opts := &ExportOptions{
		IncludeKVS: true,
	}

	data, err := mgr.Export(context.Background(), opts)
	if err != nil {
		t.Errorf("Export() error = %v", err)
		return
	}

	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Fatalf("Failed to unmarshal backup: %v", err)
	}

	if len(backup.KVS) != 2 {
		t.Errorf("KVS count = %d, want 2", len(backup.KVS))
	}
}

func TestManager_Restore(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(`{"restart_required":false}`)
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	backup := &Backup{
		Version:   BackupVersion,
		Cloud:     json.RawMessage(`{"enable":true}`),
		BLE:       json.RawMessage(`{"enable":true}`),
		MQTT:      json.RawMessage(`{"enable":false}`),
		Schedules: json.RawMessage(`{"jobs":[]}`),
		Webhooks:  json.RawMessage(`{"hooks":[]}`),
	}

	data, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	result, err := mgr.Restore(context.Background(), data, nil)
	if err != nil {
		t.Errorf("Restore() error = %v", err)
		return
	}

	if !result.Success {
		t.Error("Restore() Success = false, want true")
	}
}

func TestManager_Restore_InvalidBackup(t *testing.T) {
	transport := &mockTransport{}
	client := rpc.NewClient(transport)
	mgr := New(client)

	_, err := mgr.Restore(context.Background(), []byte("invalid json"), nil)
	if !errors.Is(err, ErrInvalidBackup) {
		t.Errorf("Restore() error = %v, want ErrInvalidBackup", err)
	}
}

func TestManager_Restore_VersionMismatch(t *testing.T) {
	transport := &mockTransport{}
	client := rpc.NewClient(transport)
	mgr := New(client)

	backup := &Backup{
		Version: BackupVersion + 1,
	}
	data, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	_, err = mgr.Restore(context.Background(), data, nil)
	if !errors.Is(err, ErrVersionMismatch) {
		t.Errorf("Restore() error = %v, want ErrVersionMismatch", err)
	}
}

func TestManager_Restore_DryRun(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			// Should not be called in dry run
			t.Errorf("unexpected call to %s in dry run", method)
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	backup := &Backup{
		Version: BackupVersion,
		Cloud:   json.RawMessage(`{"enable":true}`),
	}
	data, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	opts := &RestoreOptions{
		DryRun: true,
	}

	result, err := mgr.Restore(context.Background(), data, opts)
	if err != nil {
		t.Errorf("Restore() error = %v", err)
		return
	}

	if !result.Success {
		t.Error("Restore() Success = false, want true")
	}
}

func TestManager_Restore_WithWiFi(t *testing.T) {
	wifiCalled := false
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method == "WiFi.SetConfig" {
				wifiCalled = true
			}
			return jsonrpcResponse(`{"restart_required":true}`)
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	backup := &Backup{
		Version: BackupVersion,
		WiFi:    json.RawMessage(`{"sta":{"ssid":"Test"}}`),
	}
	data, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	opts := &RestoreOptions{
		RestoreWiFi: true,
	}

	result, err := mgr.Restore(context.Background(), data, opts)
	if err != nil {
		t.Errorf("Restore() error = %v", err)
		return
	}

	if !wifiCalled {
		t.Error("WiFi.SetConfig should have been called")
	}
	if !result.RestartRequired {
		t.Error("RestartRequired should be true")
	}
}

func TestManager_Restore_WithScripts(t *testing.T) {
	scriptMethods := make(map[string]bool)
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			scriptMethods[method] = true
			if method == "Script.Create" {
				return jsonrpcResponse(`{"id":1}`)
			}
			if method == "Script.List" {
				return jsonrpcResponse(`{"scripts":[]}`)
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	backup := &Backup{
		Version: BackupVersion,
		Scripts: []*Script{
			{ID: 1, Name: "test", Enable: true, Code: "print('hello')"},
		},
	}
	data, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	opts := &RestoreOptions{
		RestoreScripts: true,
		StopScripts:    true,
	}

	_, err = mgr.Restore(context.Background(), data, opts)
	if err != nil {
		t.Errorf("Restore() error = %v", err)
		return
	}

	if !scriptMethods["Script.Create"] {
		t.Error("Script.Create should have been called")
	}
	if !scriptMethods["Script.PutCode"] {
		t.Error("Script.PutCode should have been called")
	}
}

func TestManager_Restore_WithKVS(t *testing.T) {
	kvsCalled := false
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			if method == "KVS.Set" {
				kvsCalled = true
			}
			return jsonrpcResponse(`null`)
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	backup := &Backup{
		Version: BackupVersion,
		KVS: map[string]json.RawMessage{
			"key1": json.RawMessage(`{"value":"test"}`),
		},
	}
	data, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	opts := &RestoreOptions{
		RestoreKVS: true,
	}

	_, err = mgr.Restore(context.Background(), data, opts)
	if err != nil {
		t.Errorf("Restore() error = %v", err)
	}

	if !kvsCalled {
		t.Error("KVS.Set should have been called")
	}
}

func TestManager_ParseBackup(t *testing.T) {
	mgr := New(nil)

	backup := &Backup{
		Version: BackupVersion,
		DeviceInfo: &DeviceInfo{
			ID:    "test-device",
			Model: "SNSW-001X16EU",
		},
	}
	data, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	parsed, err := mgr.ParseBackup(data)
	if err != nil {
		t.Errorf("ParseBackup() error = %v", err)
		return
	}

	if parsed.DeviceInfo.ID != "test-device" {
		t.Errorf("DeviceInfo.ID = %s, want test-device", parsed.DeviceInfo.ID)
	}
}

func TestManager_ParseBackup_Invalid(t *testing.T) {
	mgr := New(nil)

	_, err := mgr.ParseBackup([]byte("invalid"))
	if !errors.Is(err, ErrInvalidBackup) {
		t.Errorf("ParseBackup() error = %v, want ErrInvalidBackup", err)
	}
}

func TestDefaultExportOptions(t *testing.T) {
	opts := DefaultExportOptions()

	if opts.IncludeWiFi {
		t.Error("IncludeWiFi should be false by default")
	}
	if !opts.IncludeCloud {
		t.Error("IncludeCloud should be true by default")
	}
	if !opts.IncludeAuth {
		t.Error("IncludeAuth should be true by default")
	}
	if !opts.IncludeScripts {
		t.Error("IncludeScripts should be true by default")
	}
	if !opts.IncludeKVS {
		t.Error("IncludeKVS should be true by default")
	}
}

func TestDefaultRestoreOptions(t *testing.T) {
	opts := DefaultRestoreOptions()

	if opts.RestoreWiFi {
		t.Error("RestoreWiFi should be false by default")
	}
	if !opts.RestoreCloud {
		t.Error("RestoreCloud should be true by default")
	}
	if opts.RestoreAuth {
		t.Error("RestoreAuth should be false by default")
	}
	if opts.DryRun {
		t.Error("DryRun should be false by default")
	}
	if !opts.StopScripts {
		t.Error("StopScripts should be true by default")
	}
}

func TestBackupVersion(t *testing.T) {
	if BackupVersion != 1 {
		t.Errorf("BackupVersion = %d, want 1", BackupVersion)
	}
}

// Encryptor Tests

func TestNewEncryptor(t *testing.T) {
	enc := NewEncryptor("testpassword")
	if enc == nil {
		t.Fatal("NewEncryptor returned nil")
	}
	if len(enc.key) != 32 {
		t.Errorf("key length = %d, want 32", len(enc.key))
	}
}

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	enc := NewEncryptor("testpassword")
	original := []byte("Hello, World! This is secret data.")

	encrypted, err := enc.Encrypt(original)
	if err != nil {
		t.Errorf("Encrypt() error = %v", err)
		return
	}

	if len(encrypted) == 0 {
		t.Error("Encrypt() returned empty data")
	}

	// Encrypted data should be different from original
	if string(encrypted) == string(original) {
		t.Error("Encrypted data should not match original")
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Errorf("Decrypt() error = %v", err)
		return
	}

	if string(decrypted) != string(original) {
		t.Errorf("Decrypt() = %s, want %s", string(decrypted), string(original))
	}
}

func TestEncryptor_DecryptWrongPassword(t *testing.T) {
	enc1 := NewEncryptor("password1")
	enc2 := NewEncryptor("password2")

	original := []byte("Secret data")
	encrypted, err := enc1.Encrypt(original)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	_, err = enc2.Decrypt(encrypted)
	if err == nil {
		t.Error("Decrypt() should fail with wrong password")
	}
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("Decrypt() error = %v, want ErrDecryptionFailed", err)
	}
}

func TestEncryptor_DecryptTooShort(t *testing.T) {
	enc := NewEncryptor("testpassword")

	_, err := enc.Decrypt([]byte{1, 2, 3})
	if err == nil {
		t.Error("Decrypt() should fail with short data")
	}
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("Decrypt() error = %v, want ErrDecryptionFailed", err)
	}
}

func TestEncryptor_EncryptToBase64(t *testing.T) {
	enc := NewEncryptor("testpassword")
	original := []byte("Test data")

	encoded, err := enc.EncryptToBase64(original)
	if err != nil {
		t.Errorf("EncryptToBase64() error = %v", err)
		return
	}

	// Should be valid base64
	if len(encoded) == 0 {
		t.Error("EncryptToBase64() returned empty string")
	}

	decrypted, err := enc.DecryptFromBase64(encoded)
	if err != nil {
		t.Errorf("DecryptFromBase64() error = %v", err)
		return
	}

	if string(decrypted) != string(original) {
		t.Errorf("DecryptFromBase64() = %s, want %s", string(decrypted), string(original))
	}
}

func TestEncryptor_DecryptFromBase64_Invalid(t *testing.T) {
	enc := NewEncryptor("testpassword")

	_, err := enc.DecryptFromBase64("not valid base64!!!")
	if err == nil {
		t.Error("DecryptFromBase64() should fail with invalid base64")
	}
}

// Credential Store Tests

func TestNewCredentialStore(t *testing.T) {
	store := NewCredentialStore("password")
	if store == nil {
		t.Fatal("NewCredentialStore returned nil")
	}
	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0", store.Count())
	}
}

func TestCredentialStore_StoreGet(t *testing.T) {
	store := NewCredentialStore("password")

	creds := &SecureCredentials{
		WiFiSSID:     "TestNetwork",
		WiFiPassword: "SecretPassword",
		AuthUser:     "admin",
		AuthPassword: "adminpass",
	}

	store.Store("device1", creds)

	if store.Count() != 1 {
		t.Errorf("Count() = %d, want 1", store.Count())
	}

	got, ok := store.Get("device1")
	if !ok {
		t.Error("Get() returned false for existing device")
		return
	}

	if got.WiFiSSID != "TestNetwork" {
		t.Errorf("WiFiSSID = %s, want TestNetwork", got.WiFiSSID)
	}
	if got.WiFiPassword != "SecretPassword" {
		t.Errorf("WiFiPassword = %s, want SecretPassword", got.WiFiPassword)
	}
}

func TestCredentialStore_GetNotFound(t *testing.T) {
	store := NewCredentialStore("password")

	_, ok := store.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for non-existent device")
	}
}

func TestCredentialStore_Delete(t *testing.T) {
	store := NewCredentialStore("password")

	store.Store("device1", &SecureCredentials{WiFiSSID: "Test"})
	store.Delete("device1")

	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0", store.Count())
	}

	_, ok := store.Get("device1")
	if ok {
		t.Error("Get() should return false after delete")
	}
}

func TestCredentialStore_Clear(t *testing.T) {
	store := NewCredentialStore("password")

	store.Store("device1", &SecureCredentials{WiFiSSID: "Test1"})
	store.Store("device2", &SecureCredentials{WiFiSSID: "Test2"})
	store.Clear()

	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0", store.Count())
	}
}

func TestCredentialStore_ExportImport(t *testing.T) {
	store := NewCredentialStore("password")

	store.Store("device1", &SecureCredentials{
		WiFiSSID:     "Network1",
		WiFiPassword: "Pass1",
	})
	store.Store("device2", &SecureCredentials{
		WiFiSSID:     "Network2",
		WiFiPassword: "Pass2",
	})

	exported, err := store.Export()
	if err != nil {
		t.Errorf("Export() error = %v", err)
		return
	}

	// Create new store and import
	store2 := NewCredentialStore("password")
	err = store2.Import(exported)
	if err != nil {
		t.Errorf("Import() error = %v", err)
		return
	}

	if store2.Count() != 2 {
		t.Errorf("Count() after import = %d, want 2", store2.Count())
	}

	creds, ok := store2.Get("device1")
	if !ok {
		t.Error("Get() returned false for device1 after import")
		return
	}
	if creds.WiFiSSID != "Network1" {
		t.Errorf("WiFiSSID = %s, want Network1", creds.WiFiSSID)
	}
}

func TestCredentialStore_ImportWrongPassword(t *testing.T) {
	store1 := NewCredentialStore("password1")
	store1.Store("device1", &SecureCredentials{WiFiSSID: "Test"})

	exported, err := store1.Export()
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	store2 := NewCredentialStore("password2")
	err = store2.Import(exported)
	if err == nil {
		t.Error("Import() should fail with wrong password")
	}
}

// Migrator Tests

func TestNewMigrator(t *testing.T) {
	srcTransport := &mockTransport{}
	tgtTransport := &mockTransport{}
	srcClient := rpc.NewClient(srcTransport)
	tgtClient := rpc.NewClient(tgtTransport)

	m := NewMigrator(srcClient, tgtClient)
	if m == nil {
		t.Fatal("NewMigrator returned nil")
	}
	if m.SourceClient != srcClient {
		t.Error("SourceClient not set correctly")
	}
	if m.TargetClient != tgtClient {
		t.Error("TargetClient not set correctly")
	}
}

func TestDefaultMigrationOptions(t *testing.T) {
	opts := DefaultMigrationOptions()

	if opts.IncludeWiFi {
		t.Error("IncludeWiFi should be false by default")
	}
	if !opts.IncludeCloud {
		t.Error("IncludeCloud should be true by default")
	}
	if !opts.IncludeScripts {
		t.Error("IncludeScripts should be true by default")
	}
	if !opts.RebootAfter {
		t.Error("RebootAfter should be true by default")
	}
	if opts.DryRun {
		t.Error("DryRun should be false by default")
	}
}

func TestMigrator_ValidateMigration_SameDevice(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	validation, err := m.ValidateMigration(context.Background())
	if err != nil {
		t.Errorf("ValidateMigration() error = %v", err)
		return
	}

	if !validation.Valid {
		t.Error("ValidateMigration() Valid = false, want true")
	}
	if len(validation.Errors) > 0 {
		t.Errorf("ValidateMigration() has errors: %v", validation.Errors)
	}
}

func TestMigrator_ValidateMigration_DifferentModels(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if callCount%2 == 1 {
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			}
			return jsonrpcResponse(`{"id":"test-device","model":"SHSW-PM","gen":1}`)
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	validation, err := m.ValidateMigration(context.Background())
	if err != nil {
		t.Errorf("ValidateMigration() error = %v", err)
		return
	}

	if validation.Valid {
		t.Error("ValidateMigration() Valid = true, want false for different models")
	}
	if len(validation.Errors) == 0 {
		t.Error("ValidateMigration() should have errors for different models")
	}
}

func TestMigrator_ValidateMigration_AllowDifferentModels(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if callCount%2 == 1 {
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			}
			return jsonrpcResponse(`{"id":"test-device","model":"SNSW-002X16EU","gen":2}`)
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)
	m.AllowDifferentModels = true

	validation, err := m.ValidateMigration(context.Background())
	if err != nil {
		t.Errorf("ValidateMigration() error = %v", err)
		return
	}

	if !validation.Valid {
		t.Error("ValidateMigration() Valid = false, want true with AllowDifferentModels")
	}
	if len(validation.Warnings) == 0 {
		t.Error("ValidateMigration() should have warnings for different models")
	}
}

func TestMigrator_ValidateMigration_SourceOffline(t *testing.T) {
	srcTransport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errTest
		},
	}
	tgtTransport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(`{"id":"test"}`)
		},
	}

	srcClient := rpc.NewClient(srcTransport)
	tgtClient := rpc.NewClient(tgtTransport)

	m := NewMigrator(srcClient, tgtClient)

	validation, err := m.ValidateMigration(context.Background())
	if err != nil {
		t.Errorf("ValidateMigration() error = %v", err)
		return
	}

	if validation.Valid {
		t.Error("ValidateMigration() Valid = true, want false when source offline")
	}
}

func TestMigrator_Migrate_DryRun(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	opts := &MigrationOptions{
		DryRun: true,
	}

	result, err := m.Migrate(context.Background(), opts)
	if err != nil {
		t.Errorf("Migrate() error = %v", err)
		return
	}

	if !result.Success {
		t.Error("Migrate() Success = false, want true for dry run")
	}
}

func TestMigrator_Migrate_SourceOffline(t *testing.T) {
	srcTransport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errTest
		},
	}
	tgtTransport := &mockTransport{}

	srcClient := rpc.NewClient(srcTransport)
	tgtClient := rpc.NewClient(tgtTransport)

	m := NewMigrator(srcClient, tgtClient)

	_, err := m.Migrate(context.Background(), nil)
	if !errors.Is(err, ErrSourceDeviceOffline) {
		t.Errorf("Migrate() error = %v, want ErrSourceDeviceOffline", err)
	}
}

func TestMigrator_Migrate_IncompatibleDevices(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if callCount%2 == 1 {
				return jsonrpcResponse(`{"id":"test","model":"MODEL1","gen":2}`)
			}
			return jsonrpcResponse(`{"id":"test","model":"MODEL2","gen":2}`)
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	_, err := m.Migrate(context.Background(), nil)
	if !errors.Is(err, ErrIncompatibleDevices) {
		t.Errorf("Migrate() error = %v, want ErrIncompatibleDevices", err)
	}
}

func TestMigrator_IsInProgress(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(`{"id":"test"}`)
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	if m.IsInProgress() {
		t.Error("IsInProgress() = true, want false initially")
	}
}

func TestMigrator_MigrationAlreadyInProgress(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			// Simulate slow response
			return jsonrpcResponse(`{"id":"test","model":"MODEL","gen":2}`)
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	// Manually set inProgress
	m.mu.Lock()
	m.inProgress = true
	m.mu.Unlock()

	_, err := m.Migrate(context.Background(), nil)
	if !errors.Is(err, ErrMigrationInProgress) {
		t.Errorf("Migrate() error = %v, want ErrMigrationInProgress", err)
	}
}

func TestMigrator_OnProgress(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(`{"id":"test-device","model":"MODEL","gen":2}`)
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	progressUpdates := []string{}
	m.OnProgress = func(step string, progress float64) {
		progressUpdates = append(progressUpdates, step)
	}

	opts := &MigrationOptions{DryRun: true}
	result, err := m.Migrate(context.Background(), opts)
	if err != nil {
		t.Errorf("Migrate() error = %v", err)
	}
	if result == nil {
		t.Error("Migrate() returned nil result")
	}

	if len(progressUpdates) == 0 {
		t.Error("OnProgress callback was not called")
	}
}

func TestMigrationResult_Duration(t *testing.T) {
	start := time.Now()
	result := &MigrationResult{
		StartedAt:   start,
		CompletedAt: start.Add(5 * time.Second),
	}

	if result.Duration() != 5*time.Second {
		t.Errorf("Duration() = %v, want 5s", result.Duration())
	}
}

func TestEncryptedBackupVersion(t *testing.T) {
	if EncryptedBackupVersion != 1 {
		t.Errorf("EncryptedBackupVersion = %d, want 1", EncryptedBackupVersion)
	}
}

func TestManager_ExportEncrypted(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	opts := &ExportOptions{}
	encBackup, err := mgr.ExportEncrypted(context.Background(), "testpassword", opts)
	if err != nil {
		t.Errorf("ExportEncrypted() error = %v", err)
		return
	}

	if encBackup.Version != EncryptedBackupVersion {
		t.Errorf("Version = %d, want %d", encBackup.Version, EncryptedBackupVersion)
	}
	if encBackup.EncryptedData == "" {
		t.Error("EncryptedData is empty")
	}
	if encBackup.DeviceID != "test-device" {
		t.Errorf("DeviceID = %s, want test-device", encBackup.DeviceID)
	}
}

func TestManager_RestoreEncrypted(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			default:
				return jsonrpcResponse(`{"restart_required":false}`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	// First export
	opts := &ExportOptions{}
	encBackup, err := mgr.ExportEncrypted(context.Background(), "testpassword", opts)
	if err != nil {
		t.Fatalf("ExportEncrypted() failed: %v", err)
	}

	// Then restore
	restoreOpts := &RestoreOptions{DryRun: true}
	result, err := mgr.RestoreEncrypted(context.Background(), encBackup, "testpassword", restoreOpts)
	if err != nil {
		t.Errorf("RestoreEncrypted() error = %v", err)
		return
	}

	if !result.Success {
		t.Error("RestoreEncrypted() Success = false, want true")
	}
}

func TestManager_RestoreEncrypted_WrongPassword(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device"}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	// Export with one password
	opts := &ExportOptions{}
	encBackup, err := mgr.ExportEncrypted(context.Background(), "password1", opts)
	if err != nil {
		t.Fatalf("ExportEncrypted() failed: %v", err)
	}

	// Try to restore with different password
	_, err = mgr.RestoreEncrypted(context.Background(), encBackup, "password2", nil)
	if err == nil {
		t.Error("RestoreEncrypted() should fail with wrong password")
	}
}

func TestMigrator_Migrate_FullSuccess(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	opts := &MigrationOptions{
		DryRun:       false,
		IncludeWiFi:  true,
		IncludeCloud: true,
		IncludeMQTT:  true,
		IncludeBLE:   true,
	}
	result, err := m.Migrate(context.Background(), opts)
	if err != nil {
		t.Errorf("Migrate() error = %v, want nil", err)
		return
	}

	if !result.Success {
		t.Error("Migrate() Success = false, want true")
	}
	if result.SourceDevice == nil {
		t.Error("Migrate() SourceDevice is nil")
	}
	if result.TargetDevice == nil {
		t.Error("Migrate() TargetDevice is nil")
	}
	if len(result.ComponentsMigrated) == 0 {
		t.Error("Migrate() ComponentsMigrated is empty")
	}
}

func TestMigrator_Migrate_TargetOffline(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			// First call is source device info (success), second is target (fail)
			if callCount == 1 && method == "Shelly.GetDeviceInfo" {
				return jsonrpcResponse(`{"id":"source","model":"MODEL","gen":2}`)
			}
			if callCount == 2 && method == "Shelly.GetDeviceInfo" {
				return nil, errTest
			}
			return jsonrpcResponse(`null`)
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)
	_, err := m.Migrate(context.Background(), nil)
	if !errors.Is(err, ErrTargetDeviceOffline) {
		t.Errorf("Migrate() error = %v, want ErrTargetDeviceOffline", err)
	}
}

func TestMigrator_Migrate_GenerationMismatch(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if method == "Shelly.GetDeviceInfo" {
				if callCount == 1 {
					return jsonrpcResponse(`{"id":"source","model":"MODEL","gen":2}`)
				}
				return jsonrpcResponse(`{"id":"target","model":"MODEL","gen":3}`)
			}
			return jsonrpcResponse(`null`)
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)
	m.AllowDifferentGenerations = false

	_, err := m.Migrate(context.Background(), nil)
	if !errors.Is(err, ErrIncompatibleDevices) {
		t.Errorf("Migrate() error = %v, want ErrIncompatibleDevices", err)
	}
}

func TestMigrator_Migrate_ExportFailed(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test","model":"MODEL","gen":2}`)
			case "Shelly.GetConfig":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	_, err := m.Migrate(context.Background(), nil)
	if !errors.Is(err, ErrMigrationFailed) {
		t.Errorf("Migrate() error = %v, want ErrMigrationFailed", err)
	}
}

func TestMigrator_Migrate_RestoreFailed(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test","model":"MODEL","gen":2}`)
			case "Shelly.GetConfig":
				// First call succeeds (export), second fails (restore validation)
				if callCount <= 3 {
					return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
				}
				return nil, errTest
			case "Shelly.SetConfig":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	_, err := m.Migrate(context.Background(), nil)
	if !errors.Is(err, ErrMigrationFailed) {
		t.Errorf("Migrate() error = %v, want ErrMigrationFailed", err)
	}
}

func TestMigrator_Migrate_WithReboot(t *testing.T) {
	rebootCalled := false
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test","model":"MODEL","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":true}`)
			case "Shelly.Reboot":
				rebootCalled = true
				return jsonrpcResponse(`null`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	opts := &MigrationOptions{
		RebootAfter: true,
	}
	result, err := m.Migrate(context.Background(), opts)
	if err != nil {
		t.Errorf("Migrate() error = %v", err)
	}

	if result.RestartRequired && !rebootCalled {
		t.Error("Migrate() should have called Shelly.Reboot")
	}
}

func TestMigrator_Migrate_RebootFailed(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test","model":"MODEL","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":true}`)
			case "Shelly.Reboot":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	opts := &MigrationOptions{
		RebootAfter:  true,
		IncludeWiFi:  true, // Need to include something to trigger SetConfig
		IncludeCloud: true,
	}
	result, err := m.Migrate(context.Background(), opts)

	// Should still succeed but have warning
	if err != nil {
		t.Errorf("Migrate() error = %v, want nil", err)
		return
	}
	// The warnings may come from restore result or reboot failure
	// Check that migration still succeeded
	if !result.Success {
		t.Error("Migrate() should succeed even if reboot fails")
	}
}

func TestMigrator_Migrate_WithAllOptions(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test","model":"MODEL","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "Schedule.List":
				return jsonrpcResponse(`{"jobs":[]}`)
			case "Webhook.List":
				return jsonrpcResponse(`{"hooks":[]}`)
			case "Script.List":
				return jsonrpcResponse(`{"scripts":[]}`)
			case "KVS.GetMany":
				return jsonrpcResponse(`{"items":{}}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	opts := &MigrationOptions{
		DryRun:           false,
		IncludeWiFi:      true,
		IncludeCloud:     true,
		IncludeMQTT:      true,
		IncludeBLE:       true,
		IncludeSchedules: true,
		IncludeWebhooks:  true,
		IncludeScripts:   true,
		IncludeKVS:       true,
	}
	result, err := m.Migrate(context.Background(), opts)
	if err != nil {
		t.Errorf("Migrate() error = %v", err)
		return
	}

	expectedComponents := []string{"wifi", "cloud", "mqtt", "ble", "schedules", "webhooks", "scripts", "kvs"}
	if len(result.ComponentsMigrated) != len(expectedComponents) {
		t.Errorf("Migrate() ComponentsMigrated = %v, want %v", result.ComponentsMigrated, expectedComponents)
	}
}

func TestMigrator_Migrate_NilOptions(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test","model":"MODEL","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	// Pass nil options - should use defaults
	result, err := m.Migrate(context.Background(), nil)
	if err != nil {
		t.Errorf("Migrate() error = %v", err)
		return
	}

	if !result.Success {
		t.Error("Migrate() Success = false, want true")
	}
}

func TestManager_Restore_WithSchedules(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "Schedule.DeleteAll":
				return jsonrpcResponse(`null`)
			case "Schedule.Create":
				return jsonrpcResponse(`{"id":1}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	backup := &Backup{
		Version: BackupVersion,
		DeviceInfo: &DeviceInfo{
			ID:         "test-device",
			Model:      "SNSW-001X16EU",
			Generation: 2,
		},
		Config:    json.RawMessage(`{"sys":{"device":{"name":"Test"}}}`),
		Schedules: json.RawMessage(`{"jobs":[{"enable":true,"timespec":"0 0 * * *"}]}`),
	}

	backupData, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	opts := &RestoreOptions{
		RestoreSchedules: true,
	}
	result, err := mgr.Restore(context.Background(), backupData, opts)
	if err != nil {
		t.Errorf("Restore() error = %v", err)
		return
	}

	if !result.Success {
		t.Error("Restore() Success = false, want true")
	}
}

func TestManager_Restore_WithWebhooks(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "Webhook.DeleteAll":
				return jsonrpcResponse(`null`)
			case "Webhook.Create":
				return jsonrpcResponse(`{"id":1}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	backup := &Backup{
		Version: BackupVersion,
		DeviceInfo: &DeviceInfo{
			ID:         "test-device",
			Model:      "SNSW-001X16EU",
			Generation: 2,
		},
		Config:   json.RawMessage(`{"sys":{"device":{"name":"Test"}}}`),
		Webhooks: json.RawMessage(`{"hooks":[{"id":1,"event":"switch.on","urls":["http://example.com"]}]}`),
	}

	backupData, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	opts := &RestoreOptions{
		RestoreWebhooks: true,
	}
	result, err := mgr.Restore(context.Background(), backupData, opts)
	if err != nil {
		t.Errorf("Restore() error = %v", err)
		return
	}

	if !result.Success {
		t.Error("Restore() Success = false, want true")
	}
}

func TestManager_Restore_SchedulesUnmarshalError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	// Create backup JSON directly with invalid nested schedules structure
	// that will fail during restoreSchedules when it tries to unmarshal the jobs array
	backupData := []byte(`{
		"version": 1,
		"device_info": {"id": "test-device", "model": "SNSW-001X16EU", "gen": 2},
		"config": {"sys": {"device": {"name": "Test"}}},
		"schedules": "not an object"
	}`)

	opts := &RestoreOptions{
		RestoreSchedules: true,
	}
	result, err := mgr.Restore(context.Background(), backupData, opts)
	if err != nil {
		t.Errorf("Restore() error = %v, want nil", err)
		return
	}

	// Should still complete but with errors
	if len(result.Errors) == 0 {
		t.Error("Restore() should have errors for invalid schedules JSON")
	}
}

func TestManager_Restore_WebhooksUnmarshalError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	// Create backup JSON directly with invalid nested webhooks structure
	backupData := []byte(`{
		"version": 1,
		"device_info": {"id": "test-device", "model": "SNSW-001X16EU", "gen": 2},
		"config": {"sys": {"device": {"name": "Test"}}},
		"webhooks": "not an object"
	}`)

	opts := &RestoreOptions{
		RestoreWebhooks: true,
	}
	result, err := mgr.Restore(context.Background(), backupData, opts)
	if err != nil {
		t.Errorf("Restore() error = %v, want nil", err)
		return
	}

	// Should still complete but with errors
	if len(result.Errors) == 0 {
		t.Error("Restore() should have errors for invalid webhooks JSON")
	}
}

func TestManager_Restore_StopScriptsWithRunning(t *testing.T) {
	scriptStopCalled := false
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "Script.List":
				return jsonrpcResponse(`{"scripts":[{"id":1,"running":true},{"id":2,"running":false}]}`)
			case "Script.Stop":
				scriptStopCalled = true
				return jsonrpcResponse(`null`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	backup := &Backup{
		Version: BackupVersion,
		DeviceInfo: &DeviceInfo{
			ID:         "test-device",
			Model:      "SNSW-001X16EU",
			Generation: 2,
		},
		Config: json.RawMessage(`{"sys":{"device":{"name":"Test"}}}`),
		// Scripts must be non-empty for stopAllScripts to be called
		Scripts: []*Script{{ID: 1, Name: "test", Enable: false}},
	}

	backupData, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	opts := &RestoreOptions{
		StopScripts: true,
	}
	result, err := mgr.Restore(context.Background(), backupData, opts)
	if err != nil {
		t.Errorf("Restore() error = %v", err)
	}
	if result == nil {
		t.Error("Restore() returned nil result")
	}

	if !scriptStopCalled {
		t.Error("Restore() should have called Script.Stop for running script")
	}
}

func TestManager_Restore_StopScriptsListError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "Script.List":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	backup := &Backup{
		Version: BackupVersion,
		DeviceInfo: &DeviceInfo{
			ID:         "test-device",
			Model:      "SNSW-001X16EU",
			Generation: 2,
		},
		Config: json.RawMessage(`{"sys":{"device":{"name":"Test"}}}`),
	}

	backupData, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	opts := &RestoreOptions{
		StopScripts: true,
	}
	result, err := mgr.Restore(context.Background(), backupData, opts)

	// Should still succeed even if Script.List fails
	if err != nil {
		t.Errorf("Restore() error = %v, want nil", err)
	}
	if !result.Success {
		t.Error("Restore() Success = false, want true")
	}
}

func TestManager_Restore_StopScriptsUnmarshalError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Shelly.SetConfig":
				return jsonrpcResponse(`{"restart_required":false}`)
			case "Script.List":
				return jsonrpcResponse(`invalid json`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	backup := &Backup{
		Version: BackupVersion,
		DeviceInfo: &DeviceInfo{
			ID:         "test-device",
			Model:      "SNSW-001X16EU",
			Generation: 2,
		},
		Config: json.RawMessage(`{"sys":{"device":{"name":"Test"}}}`),
	}

	backupData, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	opts := &RestoreOptions{
		StopScripts: true,
	}
	result, err := mgr.Restore(context.Background(), backupData, opts)

	// Should still succeed even if unmarshal fails
	if err != nil {
		t.Errorf("Restore() error = %v, want nil", err)
	}
	if !result.Success {
		t.Error("Restore() Success = false, want true")
	}
}

func TestManager_Export_ListWebhooksError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Webhook.List":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	opts := &ExportOptions{
		IncludeWebhooks: true,
	}
	backupData, err := mgr.Export(context.Background(), opts)
	// Export should succeed but webhooks will be nil
	if err != nil {
		t.Errorf("Export() error = %v", err)
	}
	if backupData == nil {
		t.Error("Export() returned nil backup")
		return
	}
	var backup Backup
	if err := json.Unmarshal(backupData, &backup); err != nil {
		t.Errorf("Failed to unmarshal backup: %v", err)
		return
	}
	if backup.Webhooks != nil {
		t.Error("Export() Webhooks should be nil when error occurred")
	}
}

func TestManager_Export_ListSchedulesError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			case "Schedule.List":
				return nil, errTest
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	opts := &ExportOptions{
		IncludeSchedules: true,
	}
	backupData, err := mgr.Export(context.Background(), opts)
	// Export should succeed but schedules will be nil
	if err != nil {
		t.Errorf("Export() error = %v", err)
	}
	if backupData == nil {
		t.Error("Export() returned nil backup")
		return
	}
	var backup Backup
	if err := json.Unmarshal(backupData, &backup); err != nil {
		t.Errorf("Failed to unmarshal backup: %v", err)
		return
	}
	if backup.Schedules != nil {
		t.Error("Export() Schedules should be nil when error occurred")
	}
}

func TestManager_getAuthInfo_Error(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errTest
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	authInfo, err := mgr.getAuthInfo(context.Background())
	if err == nil {
		t.Error("getAuthInfo() should return error")
	}
	if authInfo != nil {
		t.Error("getAuthInfo() should return nil on error")
	}
}

func TestManager_getAuthInfo_UnmarshalError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(`invalid json`)
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	authInfo, err := mgr.getAuthInfo(context.Background())
	if err == nil {
		t.Error("getAuthInfo() should return error for invalid json")
	}
	if authInfo != nil {
		t.Error("getAuthInfo() should return nil on unmarshal error")
	}
}

func TestManager_Export_WithAuth(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "Shelly.GetDeviceInfo":
				// Use auth_en which is the correct field name
				return jsonrpcResponse(`{"id":"test-device","model":"SNSW-001X16EU","gen":2,"auth_en":true,"auth_user":"admin"}`)
			case "Shelly.GetConfig":
				return jsonrpcResponse(`{"sys":{"device":{"name":"Test"}}}`)
			default:
				return jsonrpcResponse(`null`)
			}
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	opts := &ExportOptions{
		IncludeAuth: true,
	}
	backupData, err := mgr.Export(context.Background(), opts)
	if err != nil {
		t.Errorf("Export() error = %v", err)
		return
	}

	var backup Backup
	if err := json.Unmarshal(backupData, &backup); err != nil {
		t.Errorf("Failed to unmarshal backup: %v", err)
		return
	}
	if backup.Auth == nil {
		t.Error("Export() Auth is nil when IncludeAuth is true")
	} else if !backup.Auth.Enable {
		t.Error("Export() Auth.Enable should be true")
	}
}

func TestEncryptor_EncryptEmpty(t *testing.T) {
	enc := NewEncryptor("password")
	encrypted, err := enc.Encrypt([]byte{})
	if err != nil {
		t.Errorf("Encrypt() error = %v", err)
	}
	if len(encrypted) == 0 {
		t.Error("Encrypt() returned empty data")
	}

	// Decrypt it back
	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Errorf("Decrypt() error = %v", err)
	}
	if len(decrypted) != 0 {
		t.Errorf("Decrypt() returned non-empty data: %v", decrypted)
	}
}

func TestCredentialStore_Count(t *testing.T) {
	store := NewCredentialStore("password")

	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0", store.Count())
	}

	store.Store("device1", &SecureCredentials{WiFiSSID: "test"})
	store.Store("device2", &SecureCredentials{WiFiSSID: "test2"})

	if store.Count() != 2 {
		t.Errorf("Count() = %d, want 2", store.Count())
	}
}

func TestManager_getDeviceInfo_UnmarshalError(t *testing.T) {
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return jsonrpcResponse(`invalid json`)
		},
	}

	client := rpc.NewClient(transport)
	mgr := New(client)

	info, err := mgr.getDeviceInfo(context.Background())
	if err == nil {
		t.Error("getDeviceInfo() should return error for invalid json")
	}
	if info != nil {
		t.Error("getDeviceInfo() should return nil on unmarshal error")
	}
}

func TestMigrator_ValidateMigration_TargetOffline(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if callCount == 1 && method == "Shelly.GetDeviceInfo" {
				return jsonrpcResponse(`{"id":"source","model":"MODEL","gen":2}`)
			}
			return nil, errTest
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	validation, err := m.ValidateMigration(context.Background())
	// ValidateMigration never returns error - it puts errors in the Errors slice
	if err != nil {
		t.Errorf("ValidateMigration() should not return error, got %v", err)
	}
	if validation == nil {
		t.Error("ValidateMigration() should return validation object")
		return
	}
	// Validation should not be valid when target is unreachable
	if validation.Valid {
		t.Error("ValidateMigration() Valid should be false when target is offline")
	}
	if len(validation.Errors) == 0 {
		t.Error("ValidateMigration() should have errors when target is unreachable")
	}
	if validation.TargetDevice != nil {
		t.Error("ValidateMigration() TargetDevice should be nil when offline")
	}
}

func TestMigrator_ValidateMigration_VersionMismatch(t *testing.T) {
	callCount := 0
	transport := &mockTransport{
		callFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if method == "Shelly.GetDeviceInfo" {
				if callCount == 1 {
					return jsonrpcResponse(`{"id":"source","model":"MODEL","gen":2,"ver":"1.0.0"}`)
				}
				return jsonrpcResponse(`{"id":"target","model":"MODEL","gen":2,"ver":"2.0.0"}`)
			}
			return jsonrpcResponse(`null`)
		},
	}

	srcClient := rpc.NewClient(transport)
	tgtClient := rpc.NewClient(transport)

	m := NewMigrator(srcClient, tgtClient)

	validation, err := m.ValidateMigration(context.Background())
	if err != nil {
		t.Errorf("ValidateMigration() error = %v", err)
	}
	if validation == nil {
		t.Error("ValidateMigration() returned nil")
		return
	}
	// Check that source and target have different versions
	if validation.SourceDevice.Version == validation.TargetDevice.Version {
		t.Error("ValidateMigration() source and target should have different versions")
	}
	// Should still be valid if only version differs (warnings added)
	if !validation.Valid {
		t.Error("ValidateMigration() should be valid when only version differs")
	}
}
