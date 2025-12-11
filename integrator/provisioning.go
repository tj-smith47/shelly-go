package integrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

// Provisioning status constants.
const (
	provisioningStatusSuccess = "success"
	provisioningStatusSkipped = "skipped"
)

// ProvisioningManager handles bulk device provisioning and configuration deployment.
type ProvisioningManager struct {
	fleet     *FleetManager
	templates map[string]*ConfigTemplate
	tasks     map[string]*ProvisioningTask
	progress  map[string]*ProvisioningProgress
	mu        sync.RWMutex
}

// NewProvisioningManager creates a new provisioning manager.
func NewProvisioningManager(fleet *FleetManager) *ProvisioningManager {
	return &ProvisioningManager{
		fleet:     fleet,
		templates: make(map[string]*ConfigTemplate),
		tasks:     make(map[string]*ProvisioningTask),
		progress:  make(map[string]*ProvisioningProgress),
	}
}

// ConfigTemplate represents a device configuration template.
type ConfigTemplate struct {
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Settings  map[string]any `json:"settings"`
	types.RawFields
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	DeviceTypes []string         `json:"device_types,omitempty"`
	Actions     []TemplateAction `json:"actions,omitempty"`
}

// TemplateAction represents an action to execute after configuration.
type TemplateAction struct {
	Params map[string]any `json:"params,omitempty"`
	types.RawFields
	Type       string        `json:"type"`
	DelayAfter time.Duration `json:"delay_after,omitempty"`
}

// CreateTemplate creates a new configuration template.
func (pm *ProvisioningManager) CreateTemplate(
	id, name string, settings map[string]any,
) *ConfigTemplate {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	template := &ConfigTemplate{
		ID:        id,
		Name:      name,
		Settings:  settings,
		CreatedAt: now,
		UpdatedAt: now,
	}
	pm.templates[id] = template
	return template
}

// GetTemplate returns a template by ID.
func (pm *ProvisioningManager) GetTemplate(id string) (*ConfigTemplate, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	template, ok := pm.templates[id]
	return template, ok
}

// ListTemplates returns all templates.
func (pm *ProvisioningManager) ListTemplates() []*ConfigTemplate {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	templates := make([]*ConfigTemplate, 0, len(pm.templates))
	for _, t := range pm.templates {
		templates = append(templates, t)
	}
	return templates
}

// UpdateTemplate updates an existing template.
func (pm *ProvisioningManager) UpdateTemplate(id string, settings map[string]any) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	template, ok := pm.templates[id]
	if !ok {
		return fmt.Errorf("template %s not found", id)
	}

	template.Settings = settings
	template.UpdatedAt = time.Now()
	return nil
}

// DeleteTemplate deletes a template.
func (pm *ProvisioningManager) DeleteTemplate(id string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	_, ok := pm.templates[id]
	if ok {
		delete(pm.templates, id)
	}
	return ok
}

// IsCompatible checks if a template is compatible with a device type.
func (t *ConfigTemplate) IsCompatible(deviceType string) bool {
	if len(t.DeviceTypes) == 0 {
		return true // No restrictions
	}
	for _, dt := range t.DeviceTypes {
		if dt == deviceType {
			return true
		}
	}
	return false
}

// ProvisioningTask represents a bulk provisioning task.
type ProvisioningTask struct {
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	types.RawFields
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	TemplateID   string     `json:"template_id"`
	Status       TaskStatus `json:"status"`
	ErrorMessage string     `json:"error_message,omitempty"`
	DeviceIDs    []string   `json:"device_ids"`
}

// TaskStatus represents the status of a provisioning task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCanceled  TaskStatus = "canceled"
)

// ProvisioningProgress tracks the progress of a provisioning task.
type ProvisioningProgress struct {
	DeviceResults map[string]*DeviceProvisionResult `json:"device_results"`
	types.RawFields
	TaskID           string `json:"task_id"`
	TotalDevices     int    `json:"total_devices"`
	CompletedDevices int    `json:"completed_devices"`
	FailedDevices    int    `json:"failed_devices"`
	SkippedDevices   int    `json:"skipped_devices"`
}

// DeviceProvisionResult contains the provisioning result for a single device.
type DeviceProvisionResult struct {
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	types.RawFields
	DeviceID string `json:"device_id"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

// CreateTask creates a new provisioning task.
func (pm *ProvisioningManager) CreateTask(id, name, templateID string, deviceIDs []string) (*ProvisioningTask, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Verify template exists
	if _, ok := pm.templates[templateID]; !ok {
		return nil, fmt.Errorf("template %s not found", templateID)
	}

	task := &ProvisioningTask{
		ID:         id,
		Name:       name,
		TemplateID: templateID,
		DeviceIDs:  deviceIDs,
		Status:     TaskStatusPending,
		CreatedAt:  time.Now(),
	}
	pm.tasks[id] = task

	// Initialize progress
	pm.progress[id] = &ProvisioningProgress{
		TaskID:        id,
		TotalDevices:  len(deviceIDs),
		DeviceResults: make(map[string]*DeviceProvisionResult),
	}

	return task, nil
}

// GetTask returns a task by ID.
func (pm *ProvisioningManager) GetTask(id string) (*ProvisioningTask, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	task, ok := pm.tasks[id]
	return task, ok
}

// ListTasks returns all tasks.
func (pm *ProvisioningManager) ListTasks() []*ProvisioningTask {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	tasks := make([]*ProvisioningTask, 0, len(pm.tasks))
	for _, t := range pm.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

// GetProgress returns the progress for a task.
func (pm *ProvisioningManager) GetProgress(taskID string) (*ProvisioningProgress, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	progress, ok := pm.progress[taskID]
	return progress, ok
}

// ExecuteTask executes a provisioning task.
func (pm *ProvisioningManager) ExecuteTask(ctx context.Context, taskID string) error {
	pm.mu.Lock()
	task, ok := pm.tasks[taskID]
	if !ok {
		pm.mu.Unlock()
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != TaskStatusPending {
		pm.mu.Unlock()
		return fmt.Errorf("task %s is not pending (status: %s)", taskID, task.Status)
	}

	template, ok := pm.templates[task.TemplateID]
	if !ok {
		pm.mu.Unlock()
		return fmt.Errorf("template %s not found", task.TemplateID)
	}

	progress := pm.progress[taskID]

	now := time.Now()
	task.Status = TaskStatusRunning
	task.StartedAt = &now
	pm.mu.Unlock()

	// Execute provisioning for each device
	for _, deviceID := range task.DeviceIDs {
		select {
		case <-ctx.Done():
			pm.mu.Lock()
			task.Status = TaskStatusCanceled
			pm.mu.Unlock()
			return ctx.Err()
		default:
		}

		result := pm.provisionDevice(ctx, deviceID, template)

		pm.mu.Lock()
		progress.DeviceResults[deviceID] = result
		switch result.Status {
		case provisioningStatusSuccess:
			progress.CompletedDevices++
		case provisioningStatusSkipped:
			progress.SkippedDevices++
		default:
			progress.FailedDevices++
		}
		pm.mu.Unlock()
	}

	pm.mu.Lock()
	completedAt := time.Now()
	task.CompletedAt = &completedAt
	if progress.FailedDevices > 0 && progress.CompletedDevices == 0 {
		task.Status = TaskStatusFailed
	} else {
		task.Status = TaskStatusCompleted
	}
	pm.mu.Unlock()

	return nil
}

func (pm *ProvisioningManager) provisionDevice(
	ctx context.Context, deviceID string, template *ConfigTemplate,
) *DeviceProvisionResult {
	result := &DeviceProvisionResult{
		DeviceID:  deviceID,
		StartedAt: time.Now(),
	}

	// Get device info
	device, _, ok := pm.fleet.AccountManager().GetDevice(deviceID)
	if !ok {
		result.Status = "failed"
		result.Error = "device not found"
		now := time.Now()
		result.CompletedAt = &now
		return result
	}

	// Check compatibility
	if !template.IsCompatible(device.DeviceType) {
		result.Status = provisioningStatusSkipped
		result.Error = fmt.Sprintf("template not compatible with device type %s", device.DeviceType)
		now := time.Now()
		result.CompletedAt = &now
		return result
	}

	// Check if device can be controlled
	if !device.CanControl() {
		result.Status = provisioningStatusSkipped
		result.Error = "no control access to device"
		now := time.Now()
		result.CompletedAt = &now
		return result
	}

	// Apply configuration settings
	// In a real implementation, this would send settings to the device
	// For now, we simulate success
	for _, action := range template.Actions {
		if err := pm.fleet.SendCommand(ctx, deviceID, action.Type, action.Params); err != nil {
			result.Status = "failed"
			result.Error = fmt.Sprintf("action %s failed: %v", action.Type, err)
			now := time.Now()
			result.CompletedAt = &now
			return result
		}
		if action.DelayAfter > 0 {
			time.Sleep(action.DelayAfter)
		}
	}

	result.Status = provisioningStatusSuccess
	now := time.Now()
	result.CompletedAt = &now
	return result
}

// CancelTask cancels a running task.
func (pm *ProvisioningManager) CancelTask(taskID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	task, ok := pm.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != TaskStatusRunning && task.Status != TaskStatusPending {
		return fmt.Errorf("task %s cannot be canceled (status: %s)", taskID, task.Status)
	}

	task.Status = TaskStatusCanceled
	return nil
}

// DeleteTask deletes a task.
func (pm *ProvisioningManager) DeleteTask(taskID string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	_, ok := pm.tasks[taskID]
	if ok {
		delete(pm.tasks, taskID)
		delete(pm.progress, taskID)
	}
	return ok
}

// BulkRegistration represents a bulk device registration request.
type BulkRegistration struct {
	DefaultTemplate string               `json:"default_template,omitempty"`
	GroupID         string               `json:"group_id,omitempty"`
	Devices         []DeviceRegistration `json:"devices"`
}

// DeviceRegistration represents a single device registration.
type DeviceRegistration struct {
	types.RawFields
	DeviceID     string `json:"device_id"`
	UserID       string `json:"user_id"`
	DeviceType   string `json:"device_type"`
	Name         string `json:"name,omitempty"`
	Host         string `json:"host"`
	AccessGroups string `json:"access_groups,omitempty"`
	TemplateID   string `json:"template_id,omitempty"`
}

// BulkRegistrationResult contains the result of a bulk registration.
type BulkRegistrationResult struct {
	Results map[string]string `json:"results"`
	types.RawFields
	TotalDevices      int `json:"total_devices"`
	RegisteredDevices int `json:"registered_devices"`
	FailedDevices     int `json:"failed_devices"`
}

// RegisterDevices performs bulk device registration.
func (pm *ProvisioningManager) RegisterDevices(
	ctx context.Context, registration *BulkRegistration,
) *BulkRegistrationResult {
	result := &BulkRegistrationResult{
		TotalDevices: len(registration.Devices),
		Results:      make(map[string]string),
	}

	am := pm.fleet.AccountManager()

	for _, dev := range registration.Devices {
		// Create account device
		accountDevice := &AccountDevice{
			DeviceID:     dev.DeviceID,
			DeviceType:   dev.DeviceType,
			Name:         dev.Name,
			Host:         dev.Host,
			AccessGroups: dev.AccessGroups,
			GrantedAt:    time.Now(),
		}

		if err := am.AddDevice(dev.UserID, accountDevice); err != nil {
			result.FailedDevices++
			result.Results[dev.DeviceID] = fmt.Sprintf("failed: %v", err)
			continue
		}

		result.RegisteredDevices++
		result.Results[dev.DeviceID] = provisioningStatusSuccess

		// Add to group if specified (best-effort, group may not exist)
		if registration.GroupID != "" {
			//nolint:errcheck // Best-effort grouping - device is registered even if group fails
			pm.fleet.AddToGroup(registration.GroupID, dev.DeviceID)
		}
	}

	return result
}

// ToJSON serializes the provisioning manager state to JSON.
func (pm *ProvisioningManager) ToJSON() ([]byte, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	state := struct {
		Progress  map[string]*ProvisioningProgress `json:"progress"`
		Templates []*ConfigTemplate                `json:"templates"`
		Tasks     []*ProvisioningTask              `json:"tasks"`
	}{
		Templates: pm.ListTemplates(),
		Tasks:     pm.ListTasks(),
		Progress:  pm.progress,
	}

	return json.Marshal(state)
}
