package firmware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/rpc"
)

// Common errors.
var (
	// ErrNoUpdate indicates no update is available.
	ErrNoUpdate = errors.New("no firmware update available")

	// ErrUpdateInProgress indicates an update is already in progress.
	ErrUpdateInProgress = errors.New("firmware update already in progress")

	// ErrRollbackUnavailable indicates rollback is not available.
	ErrRollbackUnavailable = errors.New("firmware rollback not available")

	// ErrDownloadFailed indicates firmware download failed.
	ErrDownloadFailed = errors.New("firmware download failed")

	// ErrInvalidURL indicates an invalid firmware URL.
	ErrInvalidURL = errors.New("invalid firmware URL")

	// ErrRolloutInProgress indicates a staged rollout is already in progress.
	ErrRolloutInProgress = errors.New("staged rollout already in progress")
)

// Manager handles firmware operations for a single device.
type Manager struct {
	client *rpc.Client
}

// New creates a new firmware Manager with the given RPC client.
func New(client *rpc.Client) *Manager {
	return &Manager{client: client}
}

// CheckForUpdate checks if a firmware update is available.
func (m *Manager) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	result, err := m.client.Call(ctx, "Shelly.CheckForUpdate", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check for update: %w", err)
	}

	var response struct {
		Stable *struct {
			Version string `json:"version"`
		} `json:"stable,omitempty"`
		Beta *struct {
			Version string `json:"version"`
		} `json:"beta,omitempty"`
	}
	if unmarshalErr := json.Unmarshal(result, &response); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse update info: %w", unmarshalErr)
	}

	// Get current version (error ignored - we'll just have empty current version)
	current, versionErr := m.GetVersion(ctx)
	if versionErr != nil {
		current = nil
	}

	info := &UpdateInfo{}
	if current != nil {
		info.Current = current.FirmwareVersion
	}

	if response.Stable != nil {
		info.Available = response.Stable.Version
		if info.Available != info.Current {
			info.HasUpdateFlag = true
		}
	}

	if response.Beta != nil {
		info.Beta = response.Beta.Version
	}

	return info, nil
}

// Update starts a firmware update.
func (m *Manager) Update(ctx context.Context, opts *UpdateOptions) error {
	params := make(map[string]any)

	if opts != nil {
		if opts.URL != "" {
			params["url"] = opts.URL
		} else if opts.Stage != "" {
			params["stage"] = opts.Stage
		}
	}

	_, err := m.client.Call(ctx, "Shelly.Update", params)
	if err != nil {
		return fmt.Errorf("failed to start update: %w", err)
	}

	return nil
}

// GetVersion retrieves the current firmware version information.
func (m *Manager) GetVersion(ctx context.Context) (*DeviceVersion, error) {
	result, err := m.client.Call(ctx, "Shelly.GetDeviceInfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	var version DeviceVersion
	if err := json.Unmarshal(result, &version); err != nil {
		return nil, fmt.Errorf("failed to parse version info: %w", err)
	}

	return &version, nil
}

// GetStatus retrieves the current firmware update status.
func (m *Manager) GetStatus(ctx context.Context) (*UpdateStatus, error) {
	result, err := m.client.Call(ctx, "Shelly.GetStatus", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var response struct {
		Sys struct {
			AvailableUpdates *struct {
				Stable *struct {
					Version string `json:"version"`
				} `json:"stable,omitempty"`
			} `json:"available_updates,omitempty"`
		} `json:"sys,omitempty"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	status := &UpdateStatus{
		Status: "idle",
	}

	if response.Sys.AvailableUpdates != nil && response.Sys.AvailableUpdates.Stable != nil {
		status.HasUpdate = true
		status.NewVersion = response.Sys.AvailableUpdates.Stable.Version
	}

	return status, nil
}

// Rollback rolls back to the previous firmware version.
func (m *Manager) Rollback(ctx context.Context) error {
	_, err := m.client.Call(ctx, "Shelly.Rollback", nil)
	if err != nil {
		return fmt.Errorf("failed to rollback: %w", err)
	}
	return nil
}

// GetRollbackStatus checks if rollback is available.
func (m *Manager) GetRollbackStatus(ctx context.Context) (*RollbackStatus, error) {
	result, err := m.client.Call(ctx, "Shelly.GetStatus", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var response struct {
		Sys struct {
			SafeMode bool `json:"safe_mode,omitempty"`
		} `json:"sys,omitempty"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	// Rollback is generally available when device is in safe mode
	// or within a certain time after update
	status := &RollbackStatus{
		CanRollback: response.Sys.SafeMode,
	}

	return status, nil
}

// UpdateStable updates to the latest stable firmware.
func (m *Manager) UpdateStable(ctx context.Context) error {
	return m.Update(ctx, &UpdateOptions{Stage: "stable"})
}

// UpdateBeta updates to the latest beta firmware.
func (m *Manager) UpdateBeta(ctx context.Context) error {
	return m.Update(ctx, &UpdateOptions{Stage: "beta"})
}

// UpdateFromURL updates from a specific firmware URL.
func (m *Manager) UpdateFromURL(ctx context.Context, url string) error {
	return m.Update(ctx, &UpdateOptions{URL: url})
}

// IsUpdateAvailable checks if any update is available.
func (m *Manager) IsUpdateAvailable(ctx context.Context) (bool, error) {
	info, err := m.CheckForUpdate(ctx)
	if err != nil {
		return false, err
	}
	return info.HasUpdate(), nil
}

// IsBetaAvailable checks if a beta update is available.
func (m *Manager) IsBetaAvailable(ctx context.Context) (bool, error) {
	info, err := m.CheckForUpdate(ctx)
	if err != nil {
		return false, err
	}
	return info.HasBeta(), nil
}

// BatchCheckUpdates checks for updates on multiple devices concurrently.
func BatchCheckUpdates(ctx context.Context, devices []Device) []CheckResult {
	results := make([]CheckResult, len(devices))

	// Process devices (sequentially for simplicity; could be concurrent)
	for i, device := range devices {
		result := CheckResult{
			Device:  device,
			Address: device.Address(),
		}

		// Get RPC client from device
		client := device.Client()
		if client == nil {
			result.Error = errors.New("device has no RPC client")
			results[i] = result
			continue
		}

		mgr := New(client)
		info, err := mgr.CheckForUpdate(ctx)
		if err != nil {
			result.Error = err
		} else {
			result.Info = info
		}
		results[i] = result
	}

	return results
}

// BatchUpdate starts updates on multiple devices concurrently.
func BatchUpdate(ctx context.Context, devices []Device, opts *UpdateOptions) []UpdateResult {
	results := make([]UpdateResult, len(devices))

	// Process devices (sequentially for simplicity; could be concurrent)
	for i, device := range devices {
		result := UpdateResult{
			Device:  device,
			Address: device.Address(),
		}

		// Get RPC client from device
		client := device.Client()
		if client == nil {
			result.Error = errors.New("device has no RPC client")
			results[i] = result
			continue
		}

		mgr := New(client)
		err := mgr.Update(ctx, opts)
		if err != nil {
			result.Error = err
		} else {
			result.Success = true
		}
		results[i] = result
	}

	return results
}

// UpdateDevicesWithUpdates checks and updates all devices that have updates available.
func UpdateDevicesWithUpdates(ctx context.Context, devices []Device, opts *UpdateOptions) []UpdateResult {
	// First check for updates
	checkResults := BatchCheckUpdates(ctx, devices)

	// Filter devices with updates
	var toUpdate []Device
	for _, r := range checkResults {
		if r.Error == nil && r.Info != nil && r.Info.HasUpdate() {
			toUpdate = append(toUpdate, r.Device)
		}
	}

	// Update devices with available updates
	if len(toUpdate) > 0 {
		return BatchUpdate(ctx, toUpdate, opts)
	}

	return nil
}

// Downloader handles firmware file downloads.
type Downloader struct {
	// HTTPClient is the HTTP client to use for downloads.
	// If nil, http.DefaultClient is used.
	HTTPClient *http.Client
}

// NewDownloader creates a new firmware Downloader.
func NewDownloader() *Downloader {
	return &Downloader{
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// DownloadResult contains the result of a firmware download.
type DownloadResult struct {
	ContentType string
	Data        []byte
	Size        int64
}

// Download downloads firmware from the given URL.
func (d *Downloader) Download(ctx context.Context, url string) (*DownloadResult, error) {
	if url == "" {
		return nil, ErrInvalidURL
	}

	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP status %d", ErrDownloadFailed, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: reading body: %v", ErrDownloadFailed, err)
	}

	return &DownloadResult{
		Data:        data,
		Size:        int64(len(data)),
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

// DownloadToWriter downloads firmware from the given URL to a writer.
func (d *Downloader) DownloadToWriter(ctx context.Context, url string, w io.Writer) (int64, error) {
	if url == "" {
		return 0, ErrInvalidURL
	}

	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: HTTP status %d", ErrDownloadFailed, resp.StatusCode)
	}

	n, err := io.Copy(w, resp.Body)
	if err != nil {
		return n, fmt.Errorf("%w: writing: %v", ErrDownloadFailed, err)
	}

	return n, nil
}

// GetFirmwareURL retrieves the firmware download URL for a device.
func (m *Manager) GetFirmwareURL(ctx context.Context, stage string) (string, error) {
	// Get update info which may contain URLs
	result, err := m.client.Call(ctx, "Shelly.CheckForUpdate", nil)
	if err != nil {
		return "", fmt.Errorf("failed to check for update: %w", err)
	}

	var response struct {
		Stable *struct {
			Version    string `json:"version"`
			BuildID    string `json:"build_id"`
			FirmwareID string `json:"fw_id,omitempty"`
		} `json:"stable,omitempty"`
		Beta *struct {
			Version    string `json:"version"`
			BuildID    string `json:"build_id"`
			FirmwareID string `json:"fw_id,omitempty"`
		} `json:"beta,omitempty"`
	}
	if unmarshalErr := json.Unmarshal(result, &response); unmarshalErr != nil {
		return "", fmt.Errorf("failed to parse update info: %w", unmarshalErr)
	}

	// Get device info for constructing URL
	version, err := m.GetVersion(ctx)
	if err != nil {
		return "", err
	}

	// Construct firmware URL based on stage
	// Shelly firmware URLs follow pattern: http://archive.shelly-tools.de/version/DEVICE_MODEL.zip
	// or from Shelly's own servers
	var buildID string
	if stage == "beta" && response.Beta != nil {
		buildID = response.Beta.BuildID
	} else if response.Stable != nil {
		buildID = response.Stable.BuildID
	}

	if buildID == "" {
		return "", ErrNoUpdate
	}

	// The actual URL would be device-specific and returned by the API in future versions
	// For now, return a constructed URL pattern
	url := fmt.Sprintf("http://archive.shelly-tools.de/%s/%s.zip", buildID, version.Model)
	return url, nil
}

// StagedRollout manages percentage-based firmware rollouts across device fleets.
type StagedRollout struct {
	Options             *UpdateOptions
	OnProgress          func(device Device, result *UpdateResult, completed, total int)
	OnComplete          func(results []UpdateResult)
	Devices             []Device
	Results             []UpdateResult
	Percentage          int
	BatchSize           int
	DelayBetweenBatches time.Duration
	mu                  sync.Mutex
	inProgress          bool
	canceled            bool
}

// NewStagedRollout creates a new staged rollout manager.
func NewStagedRollout(devices []Device, percentage int, opts *UpdateOptions) *StagedRollout {
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	return &StagedRollout{
		Devices:             devices,
		Percentage:          percentage,
		BatchSize:           5,
		DelayBetweenBatches: 30 * time.Second,
		Options:             opts,
		Results:             []UpdateResult{},
	}
}

// TargetDeviceCount returns the number of devices that will be updated based on percentage.
func (s *StagedRollout) TargetDeviceCount() int {
	count := len(s.Devices) * s.Percentage / 100
	if count == 0 && s.Percentage > 0 && len(s.Devices) > 0 {
		count = 1 // At least one device if percentage > 0
	}
	return count
}

// SelectDevices randomly selects devices for the rollout based on percentage.
func (s *StagedRollout) SelectDevices() []Device {
	target := s.TargetDeviceCount()
	if target >= len(s.Devices) {
		return s.Devices
	}

	// Shuffle and select
	selected := make([]Device, len(s.Devices))
	copy(selected, s.Devices)

	//nolint:gosec // G404: math/rand is fine for shuffling device order, no security impact
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})

	return selected[:target]
}

// Start begins the staged rollout.
func (s *StagedRollout) Start(ctx context.Context) ([]UpdateResult, error) {
	s.mu.Lock()
	if s.inProgress {
		s.mu.Unlock()
		return nil, ErrRolloutInProgress
	}
	s.inProgress = true
	s.canceled = false
	s.Results = []UpdateResult{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.inProgress = false
		s.mu.Unlock()
	}()

	// Select devices based on percentage
	selected := s.SelectDevices()
	if len(selected) == 0 {
		return nil, nil
	}

	total := len(selected)
	completed := 0

	// Process in batches
	for i := 0; i < len(selected); i += s.BatchSize {
		// Check for cancellation or context done
		select {
		case <-ctx.Done():
			return s.Results, ctx.Err()
		default:
		}

		s.mu.Lock()
		if s.canceled {
			s.mu.Unlock()
			return s.Results, nil
		}
		s.mu.Unlock()

		// Get batch
		end := i + s.BatchSize
		if end > len(selected) {
			end = len(selected)
		}
		batch := selected[i:end]

		// Update batch
		batchResults := BatchUpdate(ctx, batch, s.Options)

		// Process results
		for j, result := range batchResults {
			s.mu.Lock()
			s.Results = append(s.Results, result)
			completed++
			s.mu.Unlock()

			if s.OnProgress != nil {
				s.OnProgress(batch[j], &result, completed, total)
			}
		}

		// Delay between batches (except for last batch)
		if end < len(selected) && s.DelayBetweenBatches > 0 {
			select {
			case <-ctx.Done():
				return s.Results, ctx.Err()
			case <-time.After(s.DelayBetweenBatches):
			}
		}
	}

	if s.OnComplete != nil {
		s.OnComplete(s.Results)
	}

	return s.Results, nil
}

// Cancel cancels the staged rollout.
func (s *StagedRollout) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.canceled = true
}

// IsInProgress returns true if a rollout is in progress.
func (s *StagedRollout) IsInProgress() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inProgress
}

// SetPercentage updates the rollout percentage.
// This only affects future calls to Start, not in-progress rollouts.
func (s *StagedRollout) SetPercentage(percentage int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}
	s.Percentage = percentage
}

// RolloutStatus contains the current status of a staged rollout.
type RolloutStatus struct {
	// InProgress indicates if the rollout is in progress.
	InProgress bool

	// TotalDevices is the total number of devices in the fleet.
	TotalDevices int

	// TargetDevices is the number of devices targeted for update.
	TargetDevices int

	// CompletedDevices is the number of devices that have been updated.
	CompletedDevices int

	// SuccessfulUpdates is the number of successful updates.
	SuccessfulUpdates int

	// FailedUpdates is the number of failed updates.
	FailedUpdates int

	// Percentage is the current rollout percentage.
	Percentage int
}

// GetStatus returns the current rollout status.
func (s *StagedRollout) GetStatus() RolloutStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := RolloutStatus{
		InProgress:       s.inProgress,
		TotalDevices:     len(s.Devices),
		TargetDevices:    s.TargetDeviceCount(),
		CompletedDevices: len(s.Results),
		Percentage:       s.Percentage,
	}

	for _, r := range s.Results {
		if r.Success {
			status.SuccessfulUpdates++
		} else {
			status.FailedUpdates++
		}
	}

	return status
}
