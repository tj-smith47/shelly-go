package integrator

import (
	"testing"
	"time"
)

func TestNewAnalytics(t *testing.T) {
	a := NewAnalytics()

	if a == nil {
		t.Fatal("NewAnalytics() returned nil")
	}
	if a.apiUsage == nil {
		t.Error("apiUsage is nil")
	}
	if a.devicePatterns == nil {
		t.Error("devicePatterns is nil")
	}
	if a.connectionMetrics == nil {
		t.Error("connectionMetrics is nil")
	}
	if a.errorTracker == nil {
		t.Error("errorTracker is nil")
	}
}

func TestAnalytics_SetRetentionPeriod(t *testing.T) {
	a := NewAnalytics()
	a.SetRetentionPeriod(24 * time.Hour)

	if a.retentionPeriod != 24*time.Hour {
		t.Errorf("retentionPeriod = %v, want 24h", a.retentionPeriod)
	}
}

func TestAnalytics_Accessors(t *testing.T) {
	a := NewAnalytics()

	if a.APIUsage() == nil {
		t.Error("APIUsage() is nil")
	}
	if a.DevicePatterns() == nil {
		t.Error("DevicePatterns() is nil")
	}
	if a.ConnectionMetrics() == nil {
		t.Error("ConnectionMetrics() is nil")
	}
	if a.ErrorTracker() == nil {
		t.Error("ErrorTracker() is nil")
	}
}

func TestAPIUsageTracker_RecordCall(t *testing.T) {
	tracker := NewAPIUsageTracker()

	call := APICall{
		Timestamp:  time.Now(),
		Endpoint:   "/api/v1/devices",
		Method:     "GET",
		StatusCode: 200,
		Latency:    100 * time.Millisecond,
		DeviceID:   "dev1",
	}

	tracker.RecordCall(&call)
	tracker.RecordCall(&call)

	if tracker.TotalCalls() != 2 {
		t.Errorf("TotalCalls() = %d, want 2", tracker.TotalCalls())
	}
}

func TestAPIUsageTracker_CallsByEndpoint(t *testing.T) {
	tracker := NewAPIUsageTracker()

	tracker.RecordCall(&APICall{Timestamp: time.Now(), Endpoint: "/api/v1/devices"})
	tracker.RecordCall(&APICall{Timestamp: time.Now(), Endpoint: "/api/v1/devices"})
	tracker.RecordCall(&APICall{Timestamp: time.Now(), Endpoint: "/api/v1/status"})

	byEndpoint := tracker.CallsByEndpoint()
	if byEndpoint["/api/v1/devices"] != 2 {
		t.Errorf("devices endpoint = %d, want 2", byEndpoint["/api/v1/devices"])
	}
	if byEndpoint["/api/v1/status"] != 1 {
		t.Errorf("status endpoint = %d, want 1", byEndpoint["/api/v1/status"])
	}
}

func TestAPIUsageTracker_CallsInLastHour(t *testing.T) {
	tracker := NewAPIUsageTracker()

	tracker.RecordCall(&APICall{Timestamp: time.Now()})
	tracker.RecordCall(&APICall{Timestamp: time.Now()})

	calls := tracker.CallsInLastHour()
	if calls != 2 {
		t.Errorf("CallsInLastHour() = %d, want 2", calls)
	}
}

func TestAPIUsageTracker_RecentCalls(t *testing.T) {
	tracker := NewAPIUsageTracker()

	for i := 0; i < 10; i++ {
		tracker.RecordCall(&APICall{Timestamp: time.Now(), Endpoint: "/test"})
	}

	recent := tracker.RecentCalls(5)
	if len(recent) != 5 {
		t.Errorf("len(RecentCalls(5)) = %d, want 5", len(recent))
	}

	recent = tracker.RecentCalls(0)
	if len(recent) != 10 {
		t.Errorf("len(RecentCalls(0)) = %d, want 10", len(recent))
	}

	recent = tracker.RecentCalls(100)
	if len(recent) != 10 {
		t.Errorf("len(RecentCalls(100)) = %d, want 10", len(recent))
	}
}

func TestAPIUsageTracker_AverageLatency(t *testing.T) {
	tracker := NewAPIUsageTracker()

	if tracker.AverageLatency() != 0 {
		t.Error("AverageLatency() should be 0 with no calls")
	}

	tracker.RecordCall(&APICall{Timestamp: time.Now(), Latency: 100 * time.Millisecond})
	tracker.RecordCall(&APICall{Timestamp: time.Now(), Latency: 200 * time.Millisecond})

	avg := tracker.AverageLatency()
	if avg != 150*time.Millisecond {
		t.Errorf("AverageLatency() = %v, want 150ms", avg)
	}
}

func TestAPIUsageTracker_MaxRecentCalls(t *testing.T) {
	tracker := NewAPIUsageTracker()
	tracker.maxRecentCalls = 5

	for i := 0; i < 10; i++ {
		tracker.RecordCall(&APICall{Timestamp: time.Now()})
	}

	if len(tracker.recentCalls) != 5 {
		t.Errorf("len(recentCalls) = %d, want 5", len(tracker.recentCalls))
	}
}

func TestDevicePatternTracker_RecordStatusChange(t *testing.T) {
	tracker := NewDevicePatternTracker()

	tracker.RecordStatusChange("dev1", "SHSW-1")
	tracker.RecordStatusChange("dev1", "SHSW-1")
	tracker.RecordStatusChange("dev2", "SHPLG-S")

	stats, ok := tracker.GetDeviceStats("dev1")
	if !ok {
		t.Fatal("GetDeviceStats() returned false")
	}
	if stats.TotalEvents != 2 {
		t.Errorf("TotalEvents = %d, want 2", stats.TotalEvents)
	}
	if stats.StatusChanges != 2 {
		t.Errorf("StatusChanges = %d, want 2", stats.StatusChanges)
	}
}

func TestDevicePatternTracker_RecordOnlineStatus(t *testing.T) {
	tracker := NewDevicePatternTracker()

	tracker.RecordOnlineStatus("dev1", "SHSW-1", true)
	tracker.RecordOnlineStatus("dev1", "SHSW-1", false)
	tracker.RecordOnlineStatus("dev1", "SHSW-1", true)

	stats, _ := tracker.GetDeviceStats("dev1")
	if stats.OnlineEvents != 2 {
		t.Errorf("OnlineEvents = %d, want 2", stats.OnlineEvents)
	}
	if stats.OfflineEvents != 1 {
		t.Errorf("OfflineEvents = %d, want 1", stats.OfflineEvents)
	}
}

func TestDevicePatternTracker_RecordCommand(t *testing.T) {
	tracker := NewDevicePatternTracker()

	tracker.RecordCommand("dev1", "SHSW-1")
	tracker.RecordCommand("dev1", "SHSW-1")

	stats, _ := tracker.GetDeviceStats("dev1")
	if stats.CommandsSent != 2 {
		t.Errorf("CommandsSent = %d, want 2", stats.CommandsSent)
	}
}

func TestDevicePatternTracker_GetMostActiveDevices(t *testing.T) {
	tracker := NewDevicePatternTracker()

	for i := 0; i < 10; i++ {
		tracker.RecordStatusChange("dev1", "SHSW-1")
	}
	for i := 0; i < 5; i++ {
		tracker.RecordStatusChange("dev2", "SHSW-1")
	}
	for i := 0; i < 3; i++ {
		tracker.RecordStatusChange("dev3", "SHSW-1")
	}

	mostActive := tracker.GetMostActiveDevices(2)
	if len(mostActive) != 2 {
		t.Fatalf("len(GetMostActiveDevices(2)) = %d, want 2", len(mostActive))
	}
	if mostActive[0].DeviceID != "dev1" {
		t.Errorf("mostActive[0].DeviceID = %v, want dev1", mostActive[0].DeviceID)
	}
	if mostActive[1].DeviceID != "dev2" {
		t.Errorf("mostActive[1].DeviceID = %v, want dev2", mostActive[1].DeviceID)
	}
}

func TestDevicePatternTracker_GetTypeStats(t *testing.T) {
	tracker := NewDevicePatternTracker()

	tracker.RecordStatusChange("dev1", "SHSW-1")
	tracker.RecordStatusChange("dev2", "SHSW-1")
	tracker.RecordStatusChange("dev3", "SHPLG-S")

	stats := tracker.GetTypeStats()
	if len(stats) != 2 {
		t.Errorf("len(GetTypeStats()) = %d, want 2", len(stats))
	}
}

func TestDevicePatternTracker_GetPeakHours(t *testing.T) {
	tracker := NewDevicePatternTracker()

	tracker.RecordStatusChange("dev1", "SHSW-1")

	peakHours := tracker.GetPeakHours()
	if len(peakHours) == 0 {
		t.Error("GetPeakHours() is empty")
	}

	currentHour := time.Now().Hour()
	if peakHours[currentHour] != 1 {
		t.Errorf("peakHours[%d] = %d, want 1", currentHour, peakHours[currentHour])
	}
}

func TestConnectionMetricsTracker_RecordConnection(t *testing.T) {
	tracker := NewConnectionMetricsTracker()

	tracker.RecordConnection("host1")
	tracker.RecordConnection("host1")
	tracker.RecordConnection("host2")

	stats, ok := tracker.GetHostStats("host1")
	if !ok {
		t.Fatal("GetHostStats() returned false")
	}
	if stats.TotalConnections != 2 {
		t.Errorf("TotalConnections = %d, want 2", stats.TotalConnections)
	}
	if stats.CurrentState != "connected" {
		t.Errorf("CurrentState = %v, want connected", stats.CurrentState)
	}
}

func TestConnectionMetricsTracker_RecordDisconnection(t *testing.T) {
	tracker := NewConnectionMetricsTracker()

	tracker.RecordConnection("host1")
	time.Sleep(10 * time.Millisecond) // Small delay
	tracker.RecordDisconnection("host1")

	stats, _ := tracker.GetHostStats("host1")
	if stats.TotalDisconnections != 1 {
		t.Errorf("TotalDisconnections = %d, want 1", stats.TotalDisconnections)
	}
	if stats.CurrentState != "disconnected" {
		t.Errorf("CurrentState = %v, want disconnected", stats.CurrentState)
	}
	if stats.ConnectedAt != nil {
		t.Error("ConnectedAt should be nil after disconnect")
	}

	// Disconnect without connect should not panic
	tracker.RecordDisconnection("nonexistent")
}

func TestConnectionMetricsTracker_RecordMessages(t *testing.T) {
	tracker := NewConnectionMetricsTracker()

	tracker.RecordConnection("host1")
	tracker.RecordMessageReceived("host1")
	tracker.RecordMessageReceived("host1")
	tracker.RecordMessageSent("host1")

	stats, _ := tracker.GetHostStats("host1")
	if stats.MessagesReceived != 2 {
		t.Errorf("MessagesReceived = %d, want 2", stats.MessagesReceived)
	}
	if stats.MessagesSent != 1 {
		t.Errorf("MessagesSent = %d, want 1", stats.MessagesSent)
	}
	if stats.LastMessageAt == nil {
		t.Error("LastMessageAt is nil")
	}

	// Messages without host should not panic
	tracker.RecordMessageReceived("nonexistent")
	tracker.RecordMessageSent("nonexistent")
}

func TestConnectionMetricsTracker_TotalMessages(t *testing.T) {
	tracker := NewConnectionMetricsTracker()

	tracker.RecordConnection("host1")
	tracker.RecordMessageReceived("host1")
	tracker.RecordMessageReceived("host1")
	tracker.RecordMessageSent("host1")

	if tracker.TotalMessagesReceived() != 2 {
		t.Errorf("TotalMessagesReceived() = %d, want 2", tracker.TotalMessagesReceived())
	}
	if tracker.TotalMessagesSent() != 1 {
		t.Errorf("TotalMessagesSent() = %d, want 1", tracker.TotalMessagesSent())
	}
}

func TestConnectionMetricsTracker_GetAllHostStats(t *testing.T) {
	tracker := NewConnectionMetricsTracker()

	tracker.RecordConnection("host1")
	tracker.RecordConnection("host2")

	stats := tracker.GetAllHostStats()
	if len(stats) != 2 {
		t.Errorf("len(GetAllHostStats()) = %d, want 2", len(stats))
	}
}

func TestErrorTracker_RecordError(t *testing.T) {
	tracker := NewErrorTracker()

	tracker.RecordError("connection", "connection failed", "dev1", "host1")
	tracker.RecordError("connection", "connection failed", "dev2", "host1")
	tracker.RecordError("timeout", "request timed out", "", "host2")

	if tracker.TotalErrors() != 3 {
		t.Errorf("TotalErrors() = %d, want 3", tracker.TotalErrors())
	}
}

func TestErrorTracker_ErrorsByType(t *testing.T) {
	tracker := NewErrorTracker()

	tracker.RecordError("connection", "msg", "", "")
	tracker.RecordError("connection", "msg", "", "")
	tracker.RecordError("timeout", "msg", "", "")

	byType := tracker.ErrorsByType()
	if byType["connection"] != 2 {
		t.Errorf("connection errors = %d, want 2", byType["connection"])
	}
	if byType["timeout"] != 1 {
		t.Errorf("timeout errors = %d, want 1", byType["timeout"])
	}
}

func TestErrorTracker_ErrorsByDevice(t *testing.T) {
	tracker := NewErrorTracker()

	tracker.RecordError("error", "msg", "dev1", "")
	tracker.RecordError("error", "msg", "dev1", "")
	tracker.RecordError("error", "msg", "dev2", "")

	byDevice := tracker.ErrorsByDevice()
	if byDevice["dev1"] != 2 {
		t.Errorf("dev1 errors = %d, want 2", byDevice["dev1"])
	}
}

func TestErrorTracker_RecentErrors(t *testing.T) {
	tracker := NewErrorTracker()

	for i := 0; i < 10; i++ {
		tracker.RecordError("error", "msg", "", "")
	}

	recent := tracker.RecentErrors(5)
	if len(recent) != 5 {
		t.Errorf("len(RecentErrors(5)) = %d, want 5", len(recent))
	}
}

func TestErrorTracker_ErrorRate(t *testing.T) {
	tracker := NewErrorTracker()

	tracker.RecordError("error", "msg", "", "")
	tracker.RecordError("error", "msg", "", "")

	rate := tracker.ErrorRate(100)
	if rate != 0.02 {
		t.Errorf("ErrorRate(100) = %v, want 0.02", rate)
	}

	rate = tracker.ErrorRate(0)
	if rate != 0 {
		t.Errorf("ErrorRate(0) = %v, want 0", rate)
	}
}

func TestErrorTracker_MaxRecentErrors(t *testing.T) {
	tracker := NewErrorTracker()
	tracker.maxRecentErrors = 5

	for i := 0; i < 10; i++ {
		tracker.RecordError("error", "msg", "", "")
	}

	if len(tracker.recentErrors) != 5 {
		t.Errorf("len(recentErrors) = %d, want 5", len(tracker.recentErrors))
	}
}

func TestAnalytics_GetSummary(t *testing.T) {
	a := NewAnalytics()

	// Record some data
	a.apiUsage.RecordCall(&APICall{Timestamp: time.Now(), Endpoint: "/test", Latency: 100 * time.Millisecond})
	a.devicePatterns.RecordStatusChange("dev1", "SHSW-1")
	a.connectionMetrics.RecordConnection("host1")
	a.errorTracker.RecordError("error", "msg", "", "")

	summary := a.GetSummary()

	if summary.GeneratedAt.IsZero() {
		t.Error("GeneratedAt is zero")
	}
	if summary.APIUsage == nil {
		t.Error("APIUsage is nil")
	}
	if summary.APIUsage.TotalCalls != 1 {
		t.Errorf("TotalCalls = %d, want 1", summary.APIUsage.TotalCalls)
	}
	if summary.DevicePatterns == nil {
		t.Error("DevicePatterns is nil")
	}
	if summary.DevicePatterns.TotalDevicesTracked != 1 {
		t.Errorf("TotalDevicesTracked = %d, want 1", summary.DevicePatterns.TotalDevicesTracked)
	}
	if summary.ConnectionMetrics == nil {
		t.Error("ConnectionMetrics is nil")
	}
	if summary.ConnectionMetrics.TotalConnections != 1 {
		t.Errorf("TotalConnections = %d, want 1", summary.ConnectionMetrics.TotalConnections)
	}
	if summary.Errors == nil {
		t.Error("Errors is nil")
	}
	if summary.Errors.TotalErrors != 1 {
		t.Errorf("TotalErrors = %d, want 1", summary.Errors.TotalErrors)
	}
}

func TestAnalytics_ToJSON(t *testing.T) {
	a := NewAnalytics()

	a.apiUsage.RecordCall(&APICall{Timestamp: time.Now(), Endpoint: "/test"})

	data, err := a.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("ToJSON() returned empty data")
	}
}

func TestAnalytics_GetSummary_PeakHours(t *testing.T) {
	a := NewAnalytics()

	// Record activity in multiple hours
	for i := 0; i < 5; i++ {
		a.devicePatterns.RecordStatusChange("dev1", "SHSW-1")
	}

	summary := a.GetSummary()

	if len(summary.DevicePatterns.PeakHours) == 0 {
		t.Error("PeakHours is empty")
	}
}

func TestAnalytics_GetSummary_ConnectedHosts(t *testing.T) {
	a := NewAnalytics()

	a.connectionMetrics.RecordConnection("host1")
	a.connectionMetrics.RecordConnection("host2")
	a.connectionMetrics.RecordDisconnection("host2")

	summary := a.GetSummary()

	if summary.ConnectionMetrics.ConnectedHosts != 1 {
		t.Errorf("ConnectedHosts = %d, want 1", summary.ConnectionMetrics.ConnectedHosts)
	}
}

func TestAPICall_Fields(t *testing.T) {
	call := APICall{
		Timestamp:  time.Now(),
		Endpoint:   "/test",
		Method:     "POST",
		StatusCode: 201,
		Latency:    50 * time.Millisecond,
		DeviceID:   "dev1",
	}

	if call.Method != "POST" {
		t.Errorf("Method = %v, want POST", call.Method)
	}
	if call.StatusCode != 201 {
		t.Errorf("StatusCode = %d, want 201", call.StatusCode)
	}
}

func TestErrorRecord_Fields(t *testing.T) {
	record := ErrorRecord{
		Timestamp: time.Now(),
		ErrorType: "network",
		Message:   "connection refused",
		DeviceID:  "dev1",
		Host:      "host1",
	}

	if record.ErrorType != "network" {
		t.Errorf("ErrorType = %v, want network", record.ErrorType)
	}
	if record.Message != "connection refused" {
		t.Errorf("Message = %v, want connection refused", record.Message)
	}
}

func TestDeviceActivityStats_ActiveHours(t *testing.T) {
	tracker := NewDevicePatternTracker()

	// Record activity
	tracker.RecordStatusChange("dev1", "SHSW-1")

	stats, _ := tracker.GetDeviceStats("dev1")
	currentHour := time.Now().Hour()

	if stats.ActiveHours[currentHour] != 1 {
		t.Errorf("ActiveHours[%d] = %d, want 1", currentHour, stats.ActiveHours[currentHour])
	}
}

func TestHostConnectionStats_AverageConnectionDuration(t *testing.T) {
	tracker := NewConnectionMetricsTracker()

	tracker.RecordConnection("host1")
	time.Sleep(20 * time.Millisecond)
	tracker.RecordDisconnection("host1")

	tracker.RecordConnection("host1")
	time.Sleep(20 * time.Millisecond)
	tracker.RecordDisconnection("host1")

	stats, _ := tracker.GetHostStats("host1")

	if stats.AverageConnectionDuration < 10*time.Millisecond {
		t.Errorf("AverageConnectionDuration = %v, expected > 10ms", stats.AverageConnectionDuration)
	}
}
