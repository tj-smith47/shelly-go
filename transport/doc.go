// Package transport provides communication layer implementations for Shelly devices.
//
// The transport package defines a Transport interface that abstracts different
// communication protocols, allowing the same device code to work over HTTP,
// WebSocket, MQTT, or CoAP/CoIoT.
//
// # Supported Transports
//
// - HTTP: Standard HTTP/HTTPS for Gen1 REST and Gen2+ RPC
// - WebSocket: Bidirectional real-time communication for Gen2+
// - MQTT: Pub/sub messaging for Gen2+ devices
// - CoAP: CoIoT protocol for Gen1 devices
//
// # Usage
//
// Create a transport for your device:
//
//	// HTTP transport (most common)
//	http := transport.NewHTTP("http://192.168.1.100",
//	    transport.WithTimeout(30*time.Second),
//	    transport.WithAuth("admin", "password"))
//
//	// WebSocket for real-time communication
//	ws := transport.NewWebSocket("ws://192.168.1.100/rpc",
//	    transport.WithReconnect(true))
//
//	// MQTT for pub/sub
//	mqtt := transport.NewMQTT("mqtt://broker:1883",
//	    transport.WithMQTTTopic("shellies/device-id"))
//
// # Transport Interface
//
// All transports implement the Transport interface which provides:
//
//   - Call(method, params): Execute an RPC call or REST request
//   - Close(): Close the connection
//
// # Connection Pooling
//
// HTTP transports use connection pooling to improve performance when
// making multiple requests to the same device.
//
// # Authentication
//
// Transports support both digest and basic authentication:
//
//	http := transport.NewHTTP("http://192.168.1.100",
//	    transport.WithDigestAuth("admin", "password"))
//
// # TLS/SSL
//
// HTTPS and secure WebSocket connections are supported:
//
//	https := transport.NewHTTP("https://192.168.1.100",
//	    transport.WithTLS(&tls.Config{InsecureSkipVerify: true}))
//
// # Retry Logic
//
// Transports include automatic retry with exponential backoff for
// transient failures:
//
//	http := transport.NewHTTP("http://192.168.1.100",
//	    transport.WithRetry(3, 1*time.Second))
//
// # Context Support
//
// All transport operations accept context.Context for cancellation
// and timeout control:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	result, err := transport.Call(ctx, "Shelly.GetStatus", nil)
package transport
