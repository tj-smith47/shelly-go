package integrator

import (
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/tj-smith47/shelly-go/types"
)

const (
	stateConnected    = "connected"
	stateDisconnected = "disconnected"
)

// Analytics provides usage analytics and metrics for the integrator.
type Analytics struct {
	apiUsage          *APIUsageTracker
	devicePatterns    *DevicePatternTracker
	connectionMetrics *ConnectionMetricsTracker
	errorTracker      *ErrorTracker
	retentionPeriod   time.Duration
	mu                sync.RWMutex
}

// NewAnalytics creates a new analytics instance.
func NewAnalytics() *Analytics {
	return &Analytics{
		apiUsage:          NewAPIUsageTracker(),
		devicePatterns:    NewDevicePatternTracker(),
		connectionMetrics: NewConnectionMetricsTracker(),
		errorTracker:      NewErrorTracker(),
		retentionPeriod:   7 * 24 * time.Hour, // 7 days default
	}
}

// SetRetentionPeriod sets the data retention period.
func (a *Analytics) SetRetentionPeriod(d time.Duration) {
	a.mu.Lock()
	a.retentionPeriod = d
	a.mu.Unlock()
}

// APIUsage returns the API usage tracker.
func (a *Analytics) APIUsage() *APIUsageTracker {
	return a.apiUsage
}

// DevicePatterns returns the device pattern tracker.
func (a *Analytics) DevicePatterns() *DevicePatternTracker {
	return a.devicePatterns
}

// ConnectionMetrics returns the connection metrics tracker.
func (a *Analytics) ConnectionMetrics() *ConnectionMetricsTracker {
	return a.connectionMetrics
}

// ErrorTracker returns the error tracker.
func (a *Analytics) ErrorTracker() *ErrorTracker {
	return a.errorTracker
}

// APIUsageTracker tracks API usage statistics.
type APIUsageTracker struct {
	callsByEndpoint map[string]int64
	callsByHour     map[string]int64
	recentCalls     []APICall
	totalCalls      int64
	maxRecentCalls  int
	mu              sync.RWMutex
}

// APICall represents a single API call.
type APICall struct {
	Timestamp time.Time `json:"timestamp"`
	types.RawFields
	Endpoint   string        `json:"endpoint"`
	Method     string        `json:"method"`
	DeviceID   string        `json:"device_id,omitempty"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"latency"`
}

// NewAPIUsageTracker creates a new API usage tracker.
func NewAPIUsageTracker() *APIUsageTracker {
	return &APIUsageTracker{
		callsByEndpoint: make(map[string]int64),
		callsByHour:     make(map[string]int64),
		recentCalls:     make([]APICall, 0, 1000),
		maxRecentCalls:  1000,
	}
}

// RecordCall records an API call.
func (t *APIUsageTracker) RecordCall(call *APICall) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalCalls++
	t.callsByEndpoint[call.Endpoint]++

	hourKey := call.Timestamp.Format("2006-01-02-15")
	t.callsByHour[hourKey]++

	// Add to recent calls
	t.recentCalls = append(t.recentCalls, *call)
	if len(t.recentCalls) > t.maxRecentCalls {
		t.recentCalls = t.recentCalls[1:]
	}
}

// TotalCalls returns the total number of API calls.
func (t *APIUsageTracker) TotalCalls() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.totalCalls
}

// CallsByEndpoint returns calls grouped by endpoint.
func (t *APIUsageTracker) CallsByEndpoint() map[string]int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]int64)
	for k, v := range t.callsByEndpoint {
		result[k] = v
	}
	return result
}

// CallsInLastHour returns the number of calls in the last hour.
func (t *APIUsageTracker) CallsInLastHour() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	hourKey := time.Now().Format("2006-01-02-15")
	return t.callsByHour[hourKey]
}

// RecentCalls returns the most recent API calls.
func (t *APIUsageTracker) RecentCalls(limit int) []APICall {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if limit <= 0 || limit > len(t.recentCalls) {
		limit = len(t.recentCalls)
	}

	start := len(t.recentCalls) - limit
	result := make([]APICall, limit)
	copy(result, t.recentCalls[start:])
	return result
}

// AverageLatency returns the average latency of recent calls.
func (t *APIUsageTracker) AverageLatency() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.recentCalls) == 0 {
		return 0
	}

	var total time.Duration
	for _, call := range t.recentCalls {
		total += call.Latency
	}
	return total / time.Duration(len(t.recentCalls))
}

// DevicePatternTracker tracks device activity patterns.
type DevicePatternTracker struct {
	deviceActivity map[string]*DeviceActivityStats
	typeActivity   map[string]*TypeActivityStats
	hourlyActivity map[int]int64
	mu             sync.RWMutex
}

// DeviceActivityStats contains activity statistics for a device.
type DeviceActivityStats struct {
	FirstActivity time.Time     `json:"first_activity"`
	LastActivity  time.Time     `json:"last_activity"`
	ActiveHours   map[int]int64 `json:"active_hours"`
	types.RawFields
	DeviceID      string `json:"device_id"`
	TotalEvents   int64  `json:"total_events"`
	StatusChanges int64  `json:"status_changes"`
	OnlineEvents  int64  `json:"online_events"`
	OfflineEvents int64  `json:"offline_events"`
	CommandsSent  int64  `json:"commands_sent"`
}

// TypeActivityStats contains activity statistics for a device type.
type TypeActivityStats struct {
	types.RawFields
	DeviceType             string  `json:"device_type"`
	DeviceCount            int64   `json:"device_count"`
	TotalEvents            int64   `json:"total_events"`
	AverageEventsPerDevice float64 `json:"average_events_per_device"`
}

// NewDevicePatternTracker creates a new device pattern tracker.
func NewDevicePatternTracker() *DevicePatternTracker {
	return &DevicePatternTracker{
		deviceActivity: make(map[string]*DeviceActivityStats),
		typeActivity:   make(map[string]*TypeActivityStats),
		hourlyActivity: make(map[int]int64),
	}
}

// RecordStatusChange records a status change event.
func (t *DevicePatternTracker) RecordStatusChange(deviceID, deviceType string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	hour := now.Hour()

	// Update device activity
	stats, ok := t.deviceActivity[deviceID]
	if !ok {
		stats = &DeviceActivityStats{
			DeviceID:      deviceID,
			FirstActivity: now,
			ActiveHours:   make(map[int]int64),
		}
		t.deviceActivity[deviceID] = stats
	}

	stats.TotalEvents++
	stats.StatusChanges++
	stats.LastActivity = now
	stats.ActiveHours[hour]++

	// Update type activity
	typeStats, ok := t.typeActivity[deviceType]
	if !ok {
		typeStats = &TypeActivityStats{
			DeviceType: deviceType,
		}
		t.typeActivity[deviceType] = typeStats
	}
	typeStats.TotalEvents++

	// Update hourly activity
	t.hourlyActivity[hour]++
}

// RecordOnlineStatus records an online/offline event.
func (t *DevicePatternTracker) RecordOnlineStatus(deviceID, deviceType string, online bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()

	stats, ok := t.deviceActivity[deviceID]
	if !ok {
		stats = &DeviceActivityStats{
			DeviceID:      deviceID,
			FirstActivity: now,
			ActiveHours:   make(map[int]int64),
		}
		t.deviceActivity[deviceID] = stats
	}

	stats.TotalEvents++
	stats.LastActivity = now
	if online {
		stats.OnlineEvents++
	} else {
		stats.OfflineEvents++
	}
}

// RecordCommand records a command sent to a device.
func (t *DevicePatternTracker) RecordCommand(deviceID, deviceType string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()

	stats, ok := t.deviceActivity[deviceID]
	if !ok {
		stats = &DeviceActivityStats{
			DeviceID:      deviceID,
			FirstActivity: now,
			ActiveHours:   make(map[int]int64),
		}
		t.deviceActivity[deviceID] = stats
	}

	stats.CommandsSent++
	stats.LastActivity = now
}

// GetDeviceStats returns stats for a specific device.
func (t *DevicePatternTracker) GetDeviceStats(deviceID string) (*DeviceActivityStats, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	stats, ok := t.deviceActivity[deviceID]
	return stats, ok
}

// GetMostActiveDevices returns the most active devices.
func (t *DevicePatternTracker) GetMostActiveDevices(limit int) []*DeviceActivityStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	devices := make([]*DeviceActivityStats, 0, len(t.deviceActivity))
	for _, stats := range t.deviceActivity {
		devices = append(devices, stats)
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].TotalEvents > devices[j].TotalEvents
	})

	if limit > len(devices) {
		limit = len(devices)
	}
	return devices[:limit]
}

// GetTypeStats returns stats for all device types.
func (t *DevicePatternTracker) GetTypeStats() []*TypeActivityStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := make([]*TypeActivityStats, 0, len(t.typeActivity))
	for _, s := range t.typeActivity {
		stats = append(stats, s)
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].TotalEvents > stats[j].TotalEvents
	})

	return stats
}

// GetPeakHours returns the hours with most activity.
func (t *DevicePatternTracker) GetPeakHours() map[int]int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[int]int64)
	for k, v := range t.hourlyActivity {
		result[k] = v
	}
	return result
}

// ConnectionMetricsTracker tracks WebSocket connection metrics.
type ConnectionMetricsTracker struct {
	hostStats             map[string]*HostConnectionStats
	totalConnections      int64
	totalDisconnections   int64
	totalMessagesReceived int64
	totalMessagesSent     int64
	mu                    sync.RWMutex
}

// HostConnectionStats contains connection statistics for a host.
type HostConnectionStats struct {
	ConnectedAt   *time.Time `json:"connected_at,omitempty"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	types.RawFields
	Host                      string          `json:"host"`
	CurrentState              string          `json:"current_state"`
	ConnectionDurations       []time.Duration `json:"-"`
	TotalConnections          int64           `json:"total_connections"`
	TotalDisconnections       int64           `json:"total_disconnections"`
	MessagesReceived          int64           `json:"messages_received"`
	MessagesSent              int64           `json:"messages_sent"`
	AverageConnectionDuration time.Duration   `json:"average_connection_duration"`
}

// NewConnectionMetricsTracker creates a new connection metrics tracker.
func NewConnectionMetricsTracker() *ConnectionMetricsTracker {
	return &ConnectionMetricsTracker{
		hostStats: make(map[string]*HostConnectionStats),
	}
}

// RecordConnection records a new connection.
func (t *ConnectionMetricsTracker) RecordConnection(host string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalConnections++

	stats, ok := t.hostStats[host]
	if !ok {
		stats = &HostConnectionStats{
			Host: host,
		}
		t.hostStats[host] = stats
	}

	now := time.Now()
	stats.TotalConnections++
	stats.CurrentState = stateConnected
	stats.ConnectedAt = &now
}

// RecordDisconnection records a disconnection.
func (t *ConnectionMetricsTracker) RecordDisconnection(host string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalDisconnections++

	stats, ok := t.hostStats[host]
	if !ok {
		return
	}

	stats.TotalDisconnections++
	stats.CurrentState = stateDisconnected

	// Calculate connection duration
	if stats.ConnectedAt != nil {
		duration := time.Since(*stats.ConnectedAt)
		stats.ConnectionDurations = append(stats.ConnectionDurations, duration)

		// Update average
		var total time.Duration
		for _, d := range stats.ConnectionDurations {
			total += d
		}
		stats.AverageConnectionDuration = total / time.Duration(len(stats.ConnectionDurations))
	}
	stats.ConnectedAt = nil
}

// RecordMessageReceived records a received message.
func (t *ConnectionMetricsTracker) RecordMessageReceived(host string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalMessagesReceived++

	stats, ok := t.hostStats[host]
	if !ok {
		return
	}

	stats.MessagesReceived++
	now := time.Now()
	stats.LastMessageAt = &now
}

// RecordMessageSent records a sent message.
func (t *ConnectionMetricsTracker) RecordMessageSent(host string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalMessagesSent++

	stats, ok := t.hostStats[host]
	if !ok {
		return
	}

	stats.MessagesSent++
}

// GetHostStats returns stats for a specific host.
func (t *ConnectionMetricsTracker) GetHostStats(host string) (*HostConnectionStats, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	stats, ok := t.hostStats[host]
	return stats, ok
}

// GetAllHostStats returns stats for all hosts.
func (t *ConnectionMetricsTracker) GetAllHostStats() []*HostConnectionStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := make([]*HostConnectionStats, 0, len(t.hostStats))
	for _, s := range t.hostStats {
		stats = append(stats, s)
	}
	return stats
}

// TotalMessagesReceived returns the total messages received.
func (t *ConnectionMetricsTracker) TotalMessagesReceived() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.totalMessagesReceived
}

// TotalMessagesSent returns the total messages sent.
func (t *ConnectionMetricsTracker) TotalMessagesSent() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.totalMessagesSent
}

// ErrorTracker tracks errors and failures.
type ErrorTracker struct {
	errorsByType    map[string]int64
	errorsByDevice  map[string]int64
	recentErrors    []ErrorRecord
	totalErrors     int64
	maxRecentErrors int
	mu              sync.RWMutex
}

// ErrorRecord represents a recorded error.
type ErrorRecord struct {
	Timestamp time.Time `json:"timestamp"`
	types.RawFields
	ErrorType string `json:"error_type"`
	Message   string `json:"message"`
	DeviceID  string `json:"device_id,omitempty"`
	Host      string `json:"host,omitempty"`
}

// NewErrorTracker creates a new error tracker.
func NewErrorTracker() *ErrorTracker {
	return &ErrorTracker{
		errorsByType:    make(map[string]int64),
		errorsByDevice:  make(map[string]int64),
		recentErrors:    make([]ErrorRecord, 0, 100),
		maxRecentErrors: 100,
	}
}

// RecordError records an error.
func (t *ErrorTracker) RecordError(errorType, message, deviceID, host string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalErrors++
	t.errorsByType[errorType]++
	if deviceID != "" {
		t.errorsByDevice[deviceID]++
	}

	record := ErrorRecord{
		Timestamp: time.Now(),
		ErrorType: errorType,
		Message:   message,
		DeviceID:  deviceID,
		Host:      host,
	}

	t.recentErrors = append(t.recentErrors, record)
	if len(t.recentErrors) > t.maxRecentErrors {
		t.recentErrors = t.recentErrors[1:]
	}
}

// TotalErrors returns the total number of errors.
func (t *ErrorTracker) TotalErrors() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.totalErrors
}

// ErrorsByType returns errors grouped by type.
func (t *ErrorTracker) ErrorsByType() map[string]int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]int64)
	for k, v := range t.errorsByType {
		result[k] = v
	}
	return result
}

// ErrorsByDevice returns errors grouped by device.
func (t *ErrorTracker) ErrorsByDevice() map[string]int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]int64)
	for k, v := range t.errorsByDevice {
		result[k] = v
	}
	return result
}

// RecentErrors returns recent errors.
func (t *ErrorTracker) RecentErrors(limit int) []ErrorRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if limit <= 0 || limit > len(t.recentErrors) {
		limit = len(t.recentErrors)
	}

	start := len(t.recentErrors) - limit
	result := make([]ErrorRecord, limit)
	copy(result, t.recentErrors[start:])
	return result
}

// ErrorRate returns the error rate (errors per total operations).
func (t *ErrorTracker) ErrorRate(totalOperations int64) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if totalOperations == 0 {
		return 0
	}
	return float64(t.totalErrors) / float64(totalOperations)
}

// AnalyticsSummary contains a summary of all analytics.
type AnalyticsSummary struct {
	// GeneratedAt is when this summary was generated.
	GeneratedAt time.Time `json:"generated_at"`

	// APIUsage contains API usage statistics.
	APIUsage *APIUsageSummary `json:"api_usage"`

	// DevicePatterns contains device activity patterns.
	DevicePatterns *DevicePatternSummary `json:"device_patterns"`

	// ConnectionMetrics contains connection metrics.
	ConnectionMetrics *ConnectionMetricsSummary `json:"connection_metrics"`

	// Errors contains error statistics.
	Errors *ErrorSummary `json:"errors"`

	// RawFields captures any unknown fields.
	types.RawFields
}

// APIUsageSummary summarizes API usage.
type APIUsageSummary struct {
	TopEndpoints map[string]int64 `json:"top_endpoints"`
	types.RawFields
	TotalCalls     int64         `json:"total_calls"`
	CallsLastHour  int64         `json:"calls_last_hour"`
	AverageLatency time.Duration `json:"average_latency"`
}

// DevicePatternSummary summarizes device patterns.
type DevicePatternSummary struct {
	types.RawFields
	MostActiveDevices   []string `json:"most_active_devices"`
	PeakHours           []int    `json:"peak_hours"`
	TotalDevicesTracked int      `json:"total_devices_tracked"`
}

// ConnectionMetricsSummary summarizes connection metrics.
type ConnectionMetricsSummary struct {
	types.RawFields
	TotalConnections      int64 `json:"total_connections"`
	TotalDisconnections   int64 `json:"total_disconnections"`
	TotalMessagesReceived int64 `json:"total_messages_received"`
	TotalMessagesSent     int64 `json:"total_messages_sent"`
	ConnectedHosts        int   `json:"connected_hosts"`
}

// ErrorSummary summarizes errors.
type ErrorSummary struct {
	ErrorsByType map[string]int64 `json:"errors_by_type"`
	types.RawFields
	TotalErrors int64 `json:"total_errors"`
}

// GetSummary returns a summary of all analytics.
func (a *Analytics) GetSummary() *AnalyticsSummary {
	summary := &AnalyticsSummary{
		GeneratedAt: time.Now(),
	}

	// API Usage
	summary.APIUsage = &APIUsageSummary{
		TotalCalls:     a.apiUsage.TotalCalls(),
		CallsLastHour:  a.apiUsage.CallsInLastHour(),
		AverageLatency: a.apiUsage.AverageLatency(),
		TopEndpoints:   a.apiUsage.CallsByEndpoint(),
	}

	// Device Patterns
	mostActive := a.devicePatterns.GetMostActiveDevices(10)
	mostActiveIDs := make([]string, len(mostActive))
	for i, d := range mostActive {
		mostActiveIDs[i] = d.DeviceID
	}

	peakHours := a.devicePatterns.GetPeakHours()
	peakHoursList := make([]int, 0, len(peakHours))
	for h := range peakHours {
		peakHoursList = append(peakHoursList, h)
	}
	sort.Slice(peakHoursList, func(i, j int) bool {
		return peakHours[peakHoursList[i]] > peakHours[peakHoursList[j]]
	})
	if len(peakHoursList) > 3 {
		peakHoursList = peakHoursList[:3]
	}

	a.devicePatterns.mu.RLock()
	summary.DevicePatterns = &DevicePatternSummary{
		TotalDevicesTracked: len(a.devicePatterns.deviceActivity),
		MostActiveDevices:   mostActiveIDs,
		PeakHours:           peakHoursList,
	}
	a.devicePatterns.mu.RUnlock()

	// Connection Metrics
	connectedHosts := 0
	for _, s := range a.connectionMetrics.GetAllHostStats() {
		if s.CurrentState == stateConnected {
			connectedHosts++
		}
	}

	a.connectionMetrics.mu.RLock()
	summary.ConnectionMetrics = &ConnectionMetricsSummary{
		TotalConnections:      a.connectionMetrics.totalConnections,
		TotalDisconnections:   a.connectionMetrics.totalDisconnections,
		TotalMessagesReceived: a.connectionMetrics.totalMessagesReceived,
		TotalMessagesSent:     a.connectionMetrics.totalMessagesSent,
		ConnectedHosts:        connectedHosts,
	}
	a.connectionMetrics.mu.RUnlock()

	// Errors
	summary.Errors = &ErrorSummary{
		TotalErrors:  a.errorTracker.TotalErrors(),
		ErrorsByType: a.errorTracker.ErrorsByType(),
	}

	return summary
}

// ToJSON serializes the analytics to JSON.
func (a *Analytics) ToJSON() ([]byte, error) {
	return json.Marshal(a.GetSummary())
}
