// Package cloud provides a client for the Shelly Cloud Control API.
//
// The Cloud Control API is a secure API for controlling Shelly devices through
// Shelly Cloud. It supports OAuth-based authentication and provides both HTTP
// API calls for device control and WebSocket connections for real-time events.
//
// # Authentication
//
// The API uses OAuth 2.0 for authentication. DIY users should use "shelly-diy"
// as the client ID. All OAuth tokens are JWT tokens that contain the user_api_url
// field, which specifies the designated server for API calls.
//
// Example authentication flow:
//
//	client := cloud.NewClient(
//	    cloud.WithAccessToken(accessToken),
//	)
//
//	// Or with email/password (for server-side applications)
//	client, err := cloud.NewClientWithCredentials(email, passwordSHA1)
//
// # Device Operations
//
// The client supports listing devices, getting status, and controlling devices:
//
//	// List all devices
//	devices, err := client.GetAllDevices(ctx)
//
//	// Get device status
//	status, err := client.GetDeviceStatus(ctx, deviceID)
//
//	// Control a switch
//	err := client.SetSwitch(ctx, deviceID, channel, true)
//
//	// Control a cover
//	err := client.SetCoverPosition(ctx, deviceID, channel, 50)
//
// # Real-Time Events
//
// WebSocket connections provide real-time device status updates:
//
//	ws, err := client.ConnectWebSocket(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer ws.Close()
//
//	ws.OnDeviceStatus(func(deviceID string, status *DeviceStatus) {
//	    fmt.Printf("Device %s status changed\n", deviceID)
//	})
//
//	// Start receiving events
//	if err := ws.Listen(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
// # Rate Limiting
//
// The Cloud Control API is rate-limited to 1 request per second. The client
// automatically handles rate limiting to avoid exceeding this limit.
//
// # Server Selection
//
// The user_api_url field in the JWT token specifies which server to use for
// API calls. The client automatically extracts this from the access token and
// routes requests to the correct server.
//
// For more information, see the Shelly Cloud Control API documentation:
// https://shelly-api-docs.shelly.cloud/cloud-control-api/
package cloud
