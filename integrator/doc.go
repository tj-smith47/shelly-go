// Package integrator provides a client for the Shelly Integrator API.
//
// The Integrator API is a Cloud-to-Cloud B2B API for integration, control,
// and device status updates collection from Shelly devices across multiple
// Shelly user accounts. It is intended for B2B and large industrial use cases.
//
// # Features
//
//   - Real-time device status streaming via WebSocket
//   - Centralized monitoring of devices across multiple accounts
//   - Device control (relay, roller, light commands)
//   - Device verification and settings retrieval
//   - JWT-based authentication with auto-refresh
//   - Multi-region authentication support
//   - Service account and API key management
//   - Multi-user account management with consent handling
//   - Fleet-wide operations with device grouping
//   - Health monitoring and aggregate statistics
//   - Bulk device provisioning with templates
//   - Usage analytics and metrics tracking
//
// # Getting Access
//
// An integrator account must be created by contacting Shelly support.
// Licenses for personal use are not provided. Upon approval, you receive:
//   - Integrator Tag (itg)
//   - Integrator Token
//
// # Authentication Flow
//
//  1. Obtain JWT using integrator credentials
//  2. Connect to WebSocket with JWT
//  3. Receive real-time events and send commands
//
// # Basic Usage
//
//	// Create client with credentials
//	client := integrator.New("your-integrator-tag", "your-integrator-token")
//
//	// Authenticate and get JWT
//	if err := client.Authenticate(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Connect to a cloud server
//	conn, err := client.Connect(ctx, "shelly-13-eu.shelly.cloud", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer conn.Close()
//
//	// Subscribe to events
//	conn.OnStatusChange(func(event *StatusChangeEvent) {
//	    fmt.Printf("Device %s status: %v\n", event.DeviceID, event.Status)
//	})
//
//	// Send command
//	err = conn.SendRelayCommand(ctx, deviceID, 0, true)
//
// # Token Management
//
// The TokenManager provides automatic token lifecycle management:
//
//	tm := integrator.NewTokenManager(client)
//	tm.SetRefreshBuffer(10 * time.Minute) // Refresh 10 min before expiry
//	tm.StartAutoRefresh(ctx)              // Start background refresh
//	defer tm.StopAutoRefresh()
//
// # Multi-Region Support
//
// The MultiRegionAuth manages authentication across multiple Shelly cloud regions:
//
//	ma := integrator.NewMultiRegionAuth("tag", "token")
//	ma.SetupDefaultRegions()              // Configure EU and US regions
//	errors := ma.AuthenticateAll(ctx)     // Authenticate to all regions
//	token, _ := ma.GetRegionToken("eu")   // Get region-specific token
//
// # Account Management
//
// The AccountManager tracks user accounts and their devices:
//
//	am := integrator.NewAccountManager()
//	am.OnDeviceAdded(func(userID string, device *AccountDevice) {
//	    fmt.Printf("New device: %s for user %s\n", device.DeviceID, userID)
//	})
//	am.ProcessCallback(callback)          // Handle Shelly consent callbacks
//	stats := am.GetStats()                // Get aggregate statistics
//
// # Fleet Management
//
// The FleetManager provides fleet-wide operations:
//
//	fm := integrator.NewFleetManager(client)
//	fm.SetAccountManager(am)
//
//	// Connect to all hosts with devices
//	fm.ConnectAll(ctx, nil)
//
//	// Create device groups
//	fm.CreateGroup("lights", "Living Room Lights", []string{"dev1", "dev2"})
//
//	// Send commands to entire groups
//	results := fm.GroupRelaysOn(ctx, "lights")
//
//	// Monitor device health
//	unhealthy := fm.HealthMonitor().GetUnhealthyDevices(5 * time.Minute)
//
// # Bulk Provisioning
//
// The ProvisioningManager handles bulk device configuration:
//
//	pm := integrator.NewProvisioningManager(fm)
//
//	// Create configuration template
//	template := pm.CreateTemplate("default", "Default Config", map[string]any{
//	    "name": "My Device",
//	})
//	template.DeviceTypes = []string{"SHSW-1", "SHSW-25"}
//	template.Actions = []TemplateAction{
//	    {Type: "relay", Params: map[string]any{"turn": "off"}},
//	}
//
//	// Create and execute provisioning task
//	task, _ := pm.CreateTask("task1", "Initial Setup", "default", deviceIDs)
//	pm.ExecuteTask(ctx, "task1")
//
// # Analytics
//
// The Analytics system tracks usage metrics:
//
//	analytics := integrator.NewAnalytics()
//	analytics.APIUsage().RecordCall(APICall{Endpoint: "/devices"})
//	analytics.DevicePatterns().RecordStatusChange("dev1", "SHSW-1")
//	analytics.ConnectionMetrics().RecordConnection("host1")
//	analytics.ErrorTracker().RecordError("timeout", "request failed", "", "")
//
//	summary := analytics.GetSummary()
//	fmt.Printf("Total API calls: %d\n", summary.APIUsage.TotalCalls)
//
// # WebSocket Events
//
// The WebSocket connection receives the following event types:
//   - StatusOnChange: Device status updates
//   - Settings: Device settings changes
//   - Online: Device online/offline status changes
//
// # Access Groups
//
// Access levels are controlled by access groups:
//   - 0x00: Read-only (state changes only)
//   - 0x01: Control enabled (can send commands)
//
// # User Consent
//
// Users grant device access through a consent URL:
//
//	url := integrator.GetConsentURL("your-tag", "https://your-callback-url")
//
// When users grant or revoke access, Shelly sends a callback that can be
// processed with AccountManager.ProcessCallback().
//
// For more information, see the official documentation:
// https://shelly-api-docs.shelly.cloud/integrator-api/
package integrator
