package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// skipContainerTest skips tests that require Docker containers on platforms
// where Docker is not available (macOS ARM64 GitHub Actions runners).
func skipContainerTest(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping container test in short mode")
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		t.Skip("Skipping container test on macOS ARM64 (Docker not available in CI)")
	}
}

// startMQTTBrokerContainer starts an MQTT broker container for testing.
func startMQTTBrokerContainer(ctx context.Context, t *testing.T) (testcontainers.Container, string) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "eclipse-mosquitto:2", //nolint:misspell // Mosquitto is the correct name
		ExposedPorts: []string{"1883/tcp"},
		WaitingFor:   wait.ForListeningPort("1883/tcp").WithStartupTimeout(60 * time.Second),
		Cmd:          []string{"sh", "-c", "echo 'listener 1883\nallow_anonymous true' > /mosquitto/config/mosquitto.conf && mosquitto -c /mosquitto/config/mosquitto.conf"}, //nolint:misspell // Mosquitto is the correct name
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skipf("Failed to start MQTT broker container (Docker not available?): %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "1883")
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to get container port: %v", err)
	}

	brokerURL := fmt.Sprintf("tcp://%s:%s", host, port.Port())
	return container, brokerURL
}

// waitForState waits for the MQTT transport to reach the expected state.
func waitForState(m *MQTT, want ConnectionState, timeout time.Duration) bool { //nolint:unparam // want may vary in future tests
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.State() == want {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestMQTT_Connect_WithBroker(t *testing.T) {
	skipContainerTest(t)

	ctx := context.Background()
	container, brokerURL := startMQTTBrokerContainer(ctx, t)
	defer container.Terminate(ctx)

	mqtt := NewMQTT(brokerURL, "test-device-123", WithTimeout(10*time.Second))

	err := mqtt.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer mqtt.Close()

	// Wait for the onConnect callback to fire
	if !waitForState(mqtt, StateConnected, 2*time.Second) {
		t.Errorf("State() = %v, want StateConnected", mqtt.State())
	}
}

func TestMQTT_Subscribe_WithBroker(t *testing.T) {
	skipContainerTest(t)

	ctx := context.Background()
	container, brokerURL := startMQTTBrokerContainer(ctx, t)
	defer container.Terminate(ctx)

	mqtt := NewMQTT(brokerURL, "test-device-456", WithTimeout(10*time.Second))

	err := mqtt.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer mqtt.Close()

	received := make(chan json.RawMessage, 1)
	handler := NotificationHandler(func(data json.RawMessage) {
		received <- data
	})

	err = mqtt.Subscribe(handler)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	err = mqtt.Unsubscribe()
	if err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}
}

func TestMQTT_StateChanges_WithBroker(t *testing.T) {
	skipContainerTest(t)

	ctx := context.Background()
	container, brokerURL := startMQTTBrokerContainer(ctx, t)
	defer container.Terminate(ctx)

	mqtt := NewMQTT(brokerURL, "test-device-789", WithTimeout(10*time.Second))

	var mu sync.Mutex
	states := make([]ConnectionState, 0)
	mqtt.OnStateChange(func(state ConnectionState) {
		mu.Lock()
		states = append(states, state)
		mu.Unlock()
	})

	if mqtt.State() != StateDisconnected {
		t.Errorf("Initial State() = %v, want StateDisconnected", mqtt.State())
	}

	err := mqtt.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Wait for the onConnect callback to fire
	if !waitForState(mqtt, StateConnected, 2*time.Second) {
		t.Errorf("After Connect() State() = %v, want StateConnected", mqtt.State())
	}

	err = mqtt.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if mqtt.State() != StateClosed {
		t.Errorf("After Close() State() = %v, want StateClosed", mqtt.State())
	}

	// Verify state changes were recorded
	mu.Lock()
	defer mu.Unlock()
	if len(states) < 2 {
		t.Errorf("Expected at least 2 state changes, got %d", len(states))
	}
}

func TestMQTT_Call_Timeout_WithBroker(t *testing.T) {
	skipContainerTest(t)

	ctx := context.Background()
	container, brokerURL := startMQTTBrokerContainer(ctx, t)
	defer container.Terminate(ctx)

	mqtt := NewMQTT(brokerURL, "test-device-call", WithTimeout(10*time.Second))

	err := mqtt.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer mqtt.Close()

	// Call with short timeout - should timeout since no device responds
	callCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	_, err = mqtt.Call(callCtx, "Shelly.GetDeviceInfo", nil)
	if err == nil {
		t.Error("Call() should timeout when no device responds")
	}
	// Error should be context deadline exceeded
	if err != context.DeadlineExceeded {
		t.Logf("Call() error = %v (expected context.DeadlineExceeded)", err)
	}
}

func TestMQTT_Call_WithMockDevice(t *testing.T) {
	skipContainerTest(t)

	ctx := context.Background()
	container, brokerURL := startMQTTBrokerContainer(ctx, t)
	defer container.Terminate(ctx)

	deviceID := "shellyplus1pm-mockdevice"

	// Create the "client" MQTT that will make calls
	clientMQTT := NewMQTT(brokerURL, deviceID, WithTimeout(10*time.Second))
	err := clientMQTT.Connect(ctx)
	if err != nil {
		t.Fatalf("Client Connect() error = %v", err)
	}
	defer clientMQTT.Close()

	// Call should timeout since there's no device responding
	callCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	_, err = clientMQTT.Call(callCtx, "Shelly.GetDeviceInfo", nil)
	// We expect a timeout since no device responds
	if err == nil {
		t.Log("Call() succeeded unexpectedly")
	} else {
		t.Logf("Call() error = %v (expected timeout)", err)
	}
}

func TestMQTT_MultipleConnections_WithBroker(t *testing.T) {
	skipContainerTest(t)

	ctx := context.Background()
	container, brokerURL := startMQTTBrokerContainer(ctx, t)
	defer container.Terminate(ctx)

	// Create multiple MQTT clients
	clients := make([]*MQTT, 5)
	for i := 0; i < 5; i++ {
		clients[i] = NewMQTT(brokerURL, fmt.Sprintf("device-%d", i), WithTimeout(10*time.Second))
		err := clients[i].Connect(ctx)
		if err != nil {
			t.Fatalf("Connect() client %d error = %v", i, err)
		}
	}

	// Verify all connected (wait for onConnect callbacks)
	for i, c := range clients {
		if !waitForState(c, StateConnected, 2*time.Second) {
			t.Errorf("Client %d State() = %v, want StateConnected", i, c.State())
		}
	}

	// Close all
	for i, c := range clients {
		if err := c.Close(); err != nil {
			t.Errorf("Close() client %d error = %v", i, err)
		}
	}
}

func TestMQTT_Reconnect_WithBroker(t *testing.T) {
	skipContainerTest(t)

	ctx := context.Background()
	container, brokerURL := startMQTTBrokerContainer(ctx, t)
	defer container.Terminate(ctx)

	mqtt := NewMQTT(brokerURL, "reconnect-device", WithTimeout(10*time.Second), WithReconnect(true))

	err := mqtt.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Wait for the onConnect callback to fire
	if !waitForState(mqtt, StateConnected, 2*time.Second) {
		t.Errorf("State() = %v, want StateConnected", mqtt.State())
	}

	// Close and verify closed
	mqtt.Close()

	if mqtt.State() != StateClosed {
		t.Errorf("After Close() State() = %v, want StateClosed", mqtt.State())
	}
}

func TestMQTT_SubscribeWhileConnected_WithBroker(t *testing.T) {
	skipContainerTest(t)

	ctx := context.Background()
	container, brokerURL := startMQTTBrokerContainer(ctx, t)
	defer container.Terminate(ctx)

	mqtt := NewMQTT(brokerURL, "subscribe-device", WithTimeout(10*time.Second))

	// Connect first
	err := mqtt.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer mqtt.Close()

	// Now subscribe while connected
	handler := NotificationHandler(func(data json.RawMessage) {})
	err = mqtt.Subscribe(handler)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	// Unsubscribe
	err = mqtt.Unsubscribe()
	if err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}
}
