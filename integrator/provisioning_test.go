package integrator

import (
	"context"
	"testing"
	"time"
)

func TestNewProvisioningManager(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	if pm == nil {
		t.Fatal("NewProvisioningManager() returned nil")
	}
}

func TestProvisioningManager_Templates(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	// Create template
	template := pm.CreateTemplate("t1", "Template 1", map[string]any{
		"name": "Test Name",
	})

	if template.ID != "t1" {
		t.Errorf("ID = %v, want t1", template.ID)
	}
	if template.Name != "Template 1" {
		t.Errorf("Name = %v, want Template 1", template.Name)
	}
	if template.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	// Get template
	got, ok := pm.GetTemplate("t1")
	if !ok {
		t.Error("GetTemplate() returned false")
	}
	if got.Name != "Template 1" {
		t.Errorf("Name = %v, want Template 1", got.Name)
	}

	// List templates
	templates := pm.ListTemplates()
	if len(templates) != 1 {
		t.Errorf("len(ListTemplates()) = %d, want 1", len(templates))
	}

	// Update template
	err := pm.UpdateTemplate("t1", map[string]any{"name": "Updated"})
	if err != nil {
		t.Fatalf("UpdateTemplate() error = %v", err)
	}

	got, _ = pm.GetTemplate("t1")
	if got.Settings["name"] != "Updated" {
		t.Error("template not updated")
	}
	if got.UpdatedAt.Before(got.CreatedAt) {
		t.Error("UpdatedAt not updated")
	}

	// Update nonexistent
	err = pm.UpdateTemplate("nonexistent", nil)
	if err == nil {
		t.Error("UpdateTemplate(nonexistent) should error")
	}

	// Get nonexistent
	_, ok = pm.GetTemplate("nonexistent")
	if ok {
		t.Error("GetTemplate(nonexistent) should return false")
	}

	// Delete template
	deleted := pm.DeleteTemplate("t1")
	if !deleted {
		t.Error("DeleteTemplate() = false, want true")
	}

	deleted = pm.DeleteTemplate("nonexistent")
	if deleted {
		t.Error("DeleteTemplate(nonexistent) = true, want false")
	}
}

func TestConfigTemplate_IsCompatible(t *testing.T) {
	// No restrictions
	template := &ConfigTemplate{}
	if !template.IsCompatible("SHSW-1") {
		t.Error("IsCompatible() = false, want true (no restrictions)")
	}

	// With restrictions
	template.DeviceTypes = []string{"SHSW-1", "SHSW-25"}
	if !template.IsCompatible("SHSW-1") {
		t.Error("IsCompatible(SHSW-1) = false, want true")
	}
	if template.IsCompatible("SHPLG-S") {
		t.Error("IsCompatible(SHPLG-S) = true, want false")
	}
}

func TestProvisioningManager_Tasks(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	pm.CreateTemplate("t1", "Template 1", nil)

	// Create task
	task, err := pm.CreateTask("task1", "Task 1", "t1", []string{"dev1", "dev2"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if task.ID != "task1" {
		t.Errorf("ID = %v, want task1", task.ID)
	}
	if task.Status != TaskStatusPending {
		t.Errorf("Status = %v, want pending", task.Status)
	}

	// Get task
	got, ok := pm.GetTask("task1")
	if !ok {
		t.Error("GetTask() returned false")
	}
	if got.Name != "Task 1" {
		t.Errorf("Name = %v, want Task 1", got.Name)
	}

	// List tasks
	tasks := pm.ListTasks()
	if len(tasks) != 1 {
		t.Errorf("len(ListTasks()) = %d, want 1", len(tasks))
	}

	// Get progress
	progress, ok := pm.GetProgress("task1")
	if !ok {
		t.Error("GetProgress() returned false")
	}
	if progress.TotalDevices != 2 {
		t.Errorf("TotalDevices = %d, want 2", progress.TotalDevices)
	}

	// Get nonexistent
	_, ok = pm.GetTask("nonexistent")
	if ok {
		t.Error("GetTask(nonexistent) should return false")
	}

	// Delete task
	deleted := pm.DeleteTask("task1")
	if !deleted {
		t.Error("DeleteTask() = false, want true")
	}

	// Verify progress also deleted
	_, ok = pm.GetProgress("task1")
	if ok {
		t.Error("progress should be deleted with task")
	}
}

func TestProvisioningManager_CreateTask_NoTemplate(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	_, err := pm.CreateTask("task1", "Task 1", "nonexistent", nil)
	if err == nil {
		t.Error("CreateTask() should error for nonexistent template")
	}
}

func TestProvisioningManager_ExecuteTask(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	// Setup
	_ = fm.accounts.AddDevice("user1", &AccountDevice{
		DeviceID:     "dev1",
		DeviceType:   "SHSW-1",
		AccessGroups: "01",
		Host:         "host1",
	})
	_ = fm.accounts.AddDevice("user1", &AccountDevice{
		DeviceID:     "dev2",
		DeviceType:   "SHSW-1",
		AccessGroups: "00", // Read-only
		Host:         "host1",
	})

	template := pm.CreateTemplate("t1", "Template 1", map[string]any{"name": "Test"})
	template.DeviceTypes = []string{"SHSW-1"}
	template.Actions = []TemplateAction{
		{Type: "relay", Params: map[string]any{"turn": "on"}},
	}

	_, _ = pm.CreateTask("task1", "Task 1", "t1", []string{"dev1", "dev2", "nonexistent"})

	err := pm.ExecuteTask(context.Background(), "task1")
	if err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}

	task, _ := pm.GetTask("task1")
	if task.Status != TaskStatusCompleted && task.Status != TaskStatusFailed {
		t.Errorf("Status = %v, want completed or failed", task.Status)
	}
	if task.StartedAt == nil {
		t.Error("StartedAt is nil")
	}
	if task.CompletedAt == nil {
		t.Error("CompletedAt is nil")
	}

	progress, _ := pm.GetProgress("task1")
	if progress.FailedDevices+progress.SkippedDevices+progress.CompletedDevices != 3 {
		t.Error("not all devices processed")
	}
}

func TestProvisioningManager_ExecuteTask_NotFound(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	err := pm.ExecuteTask(context.Background(), "nonexistent")
	if err == nil {
		t.Error("ExecuteTask(nonexistent) should error")
	}
}

func TestProvisioningManager_ExecuteTask_NotPending(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	pm.CreateTemplate("t1", "Template", nil)
	_, _ = pm.CreateTask("task1", "Task", "t1", nil)
	pm.tasks["task1"].Status = TaskStatusCompleted

	err := pm.ExecuteTask(context.Background(), "task1")
	if err == nil {
		t.Error("ExecuteTask() should error for non-pending task")
	}
}

func TestProvisioningManager_ExecuteTask_Canceled(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", DeviceType: "SHSW-1", AccessGroups: "01"})
	pm.CreateTemplate("t1", "Template", nil)
	_, _ = pm.CreateTask("task1", "Task", "t1", []string{"dev1"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := pm.ExecuteTask(ctx, "task1")
	if err != context.Canceled {
		t.Errorf("ExecuteTask() error = %v, want context.Canceled", err)
	}

	task, _ := pm.GetTask("task1")
	if task.Status != TaskStatusCanceled {
		t.Errorf("Status = %v, want canceled", task.Status)
	}
}

func TestProvisioningManager_CancelTask(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	pm.CreateTemplate("t1", "Template", nil)
	_, _ = pm.CreateTask("task1", "Task", "t1", nil)

	err := pm.CancelTask("task1")
	if err != nil {
		t.Fatalf("CancelTask() error = %v", err)
	}

	task, _ := pm.GetTask("task1")
	if task.Status != TaskStatusCanceled {
		t.Errorf("Status = %v, want canceled", task.Status)
	}

	// Cancel nonexistent
	err = pm.CancelTask("nonexistent")
	if err == nil {
		t.Error("CancelTask(nonexistent) should error")
	}

	// Cancel completed task
	pm.tasks["task1"].Status = TaskStatusCompleted
	err = pm.CancelTask("task1")
	if err == nil {
		t.Error("CancelTask(completed) should error")
	}
}

func TestProvisioningManager_provisionDevice_Incompatible(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	_ = fm.accounts.AddDevice("user1", &AccountDevice{DeviceID: "dev1", DeviceType: "SHPLG-S", AccessGroups: "01"})

	template := &ConfigTemplate{
		DeviceTypes: []string{"SHSW-1"}, // Different type
	}

	result := pm.provisionDevice(context.Background(), "dev1", template)
	if result.Status != "skipped" {
		t.Errorf("Status = %v, want skipped", result.Status)
	}
}

func TestProvisioningManager_RegisterDevices(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	fm.CreateGroup("g1", "Group 1", nil)

	registration := &BulkRegistration{
		Devices: []DeviceRegistration{
			{DeviceID: "dev1", UserID: "user1", DeviceType: "SHSW-1", Name: "Device 1", Host: "host1"},
			{DeviceID: "dev2", UserID: "user2", DeviceType: "SHSW-1", Name: "Device 2", Host: "host1"},
		},
		GroupID: "g1",
	}

	result := pm.RegisterDevices(context.Background(), registration)

	if result.TotalDevices != 2 {
		t.Errorf("TotalDevices = %d, want 2", result.TotalDevices)
	}
	if result.RegisteredDevices != 2 {
		t.Errorf("RegisteredDevices = %d, want 2", result.RegisteredDevices)
	}
	if result.FailedDevices != 0 {
		t.Errorf("FailedDevices = %d, want 0", result.FailedDevices)
	}

	// Verify devices were added
	if fm.accounts.DeviceCount() != 2 {
		t.Errorf("DeviceCount() = %d, want 2", fm.accounts.DeviceCount())
	}

	// Verify added to group
	g, _ := fm.GetGroup("g1")
	if len(g.DeviceIDs) != 2 {
		t.Errorf("group DeviceIDs = %d, want 2", len(g.DeviceIDs))
	}
}

func TestProvisioningManager_ToJSON(t *testing.T) {
	client := New("tag", "token")
	fm := NewFleetManager(client)
	pm := NewProvisioningManager(fm)

	pm.CreateTemplate("t1", "Template 1", map[string]any{"key": "value"})
	_, _ = pm.CreateTask("task1", "Task 1", "t1", []string{"dev1"})

	data, err := pm.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("ToJSON() returned empty data")
	}
}

func TestTaskStatus_Values(t *testing.T) {
	statuses := []TaskStatus{
		TaskStatusPending,
		TaskStatusRunning,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusCanceled,
	}

	// Just verify they exist and are different
	seen := make(map[TaskStatus]bool)
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("duplicate status value: %v", s)
		}
		seen[s] = true
	}
}

func TestDeviceProvisionResult_CompletedAt(t *testing.T) {
	result := &DeviceProvisionResult{
		DeviceID:  "dev1",
		Status:    "success",
		StartedAt: time.Now(),
	}

	if result.CompletedAt != nil {
		t.Error("CompletedAt should be nil initially")
	}

	now := time.Now()
	result.CompletedAt = &now

	if result.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestTemplateAction_DelayAfter(t *testing.T) {
	action := TemplateAction{
		Type:       "relay",
		Params:     map[string]any{"turn": "on"},
		DelayAfter: 100 * time.Millisecond,
	}

	if action.DelayAfter != 100*time.Millisecond {
		t.Errorf("DelayAfter = %v, want 100ms", action.DelayAfter)
	}
}

func TestBulkRegistrationResult_Results(t *testing.T) {
	result := &BulkRegistrationResult{
		TotalDevices:      3,
		RegisteredDevices: 2,
		FailedDevices:     1,
		Results: map[string]string{
			"dev1": "success",
			"dev2": "success",
			"dev3": "failed: error message",
		},
	}

	if result.Results["dev1"] != "success" {
		t.Errorf("Results[dev1] = %v, want success", result.Results["dev1"])
	}
	if result.Results["dev3"] != "failed: error message" {
		t.Errorf("Results[dev3] = %v", result.Results["dev3"])
	}
}
