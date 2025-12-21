package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/tj-smith47/shelly-go/transport"
)

// mockTransport is a mock transport for testing
type mockTransport struct {
	err      error
	lastCall any
	response []byte
	mu       sync.Mutex
}

func (m *mockTransport) Call(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
	m.mu.Lock()
	m.lastCall = req
	m.mu.Unlock()

	// If this is a batch request, dynamically generate response IDs to match request IDs
	if batchReq, ok := req.(transport.BatchRPCRequest); ok && batchReq.IsBatch() {
		// Get the requests from params (which contains the marshaled batch)
		var requests []*Request
		if err := json.Unmarshal(req.GetParams(), &requests); err == nil && len(requests) > 0 {
			return m.generateBatchResponse(requests)
		}
	}

	return m.response, m.err
}

// generateBatchResponse creates a dynamic batch response with matching IDs
func (m *mockTransport) generateBatchResponse(requests []*Request) (json.RawMessage, error) {
	if m.err != nil {
		return nil, m.err
	}

	// If a custom response is set, try to use it but with updated IDs
	if len(m.response) > 0 {
		// Parse the template response
		var responseTemplate []map[string]any
		if err := json.Unmarshal(m.response, &responseTemplate); err != nil {
			// If it's not a batch response template, just return the raw response
			return m.response, nil
		}

		// Check if the template response IDs are already set (for out-of-order testing)
		// If any template has an ID field, we assume it's already set up correctly
		hasIDs := false
		for _, respTmpl := range responseTemplate {
			if _, ok := respTmpl["id"]; ok {
				hasIDs = true
				break
			}
		}

		// If template already has IDs (like in ID correlation tests), just update them to match
		// the generated request IDs while preserving the order
		if hasIDs {
			// Map template IDs (1, 2, 3, etc.) to actual request IDs
			idMap := make(map[float64]any)
			for i, req := range requests {
				// Template uses 1-based numbering (1, 2, 3...)
				idMap[float64(i+1)] = req.ID
			}

			// Update each response's ID from the template ID to the actual request ID
			responses := make([]map[string]any, len(responseTemplate))
			for i, respTmpl := range responseTemplate {
				responses[i] = make(map[string]any)
				for k, v := range respTmpl {
					responses[i][k] = v
				}
				// Map the template ID to the actual request ID
				if templateID, ok := respTmpl["id"].(float64); ok {
					if actualID, ok := idMap[templateID]; ok {
						responses[i]["id"] = actualID
					}
				}
			}
			return json.Marshal(responses)
		}

		// Otherwise, map responses 1:1 with requests in order
		responses := make([]map[string]any, len(responseTemplate))
		for i := range responseTemplate {
			responses[i] = responseTemplate[i]
			// Match this response template to the corresponding request
			if i < len(requests) {
				responses[i]["id"] = requests[i].ID
			}
		}

		return json.Marshal(responses)
	}

	// No template, create default responses for all requests
	responses := make([]map[string]any, len(requests))
	for i, req := range requests {
		responses[i] = map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  map[string]any{},
		}
	}

	return json.Marshal(responses)
}

func (m *mockTransport) Close() error {
	return nil
}

// mockStatefulTransport is a mock transport that supports notifications
type mockStatefulTransport struct {
	notificationHandler transport.NotificationHandler
	mockTransport
	closed bool
}

func (m *mockStatefulTransport) Subscribe(handler transport.NotificationHandler) error {
	m.notificationHandler = handler
	return nil
}

func (m *mockStatefulTransport) Unsubscribe() error {
	m.notificationHandler = nil
	return nil
}

func (m *mockStatefulTransport) Close() error {
	m.closed = true
	return nil
}

func TestNewClient(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.transport != mt {
		t.Error("client transport not set correctly")
	}

	if client.builder == nil {
		t.Error("client builder should be initialized")
	}

	if client.router == nil {
		t.Error("client router should be initialized")
	}
}

func TestNewClient_WithStatefulTransport(t *testing.T) {
	mt := &mockStatefulTransport{}
	client := NewClient(mt)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if mt.notificationHandler == nil {
		t.Error("notification handler should be registered with stateful transport")
	}
}

func TestNewClientWithAuth(t *testing.T) {
	mt := &mockTransport{}
	auth := &AuthData{
		Username: "admin",
		Password: "password",
	}

	client := NewClientWithAuth(mt, auth)

	if client == nil {
		t.Fatal("NewClientWithAuth() returned nil")
	}

	if client.auth != auth {
		t.Error("client auth not set correctly")
	}
}

func TestClient_Call_Success(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"jsonrpc":"2.0","id":1,"result":{"status":"on"}}`),
	}

	client := NewClient(mt)

	result, err := client.Call(context.Background(), "Switch.GetStatus", map[string]any{"id": 0})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	if len(result) == 0 {
		t.Error("result should not be empty")
	}

	var status map[string]any
	if err := json.Unmarshal(result, &status); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if status["status"] != "on" {
		t.Errorf("status = %v, want on", status["status"])
	}
}

func TestClient_Call_Error(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":404,"message":"Not found"}}`),
	}

	client := NewClient(mt)

	_, err := client.Call(context.Background(), "Switch.GetStatus", map[string]any{"id": 999})
	if err == nil {
		t.Error("Call() should return error for RPC error response")
	}

	// Check if it's an ErrorObject
	var errObj *ErrorObject
	if errors.As(err, &errObj) {
		if errObj.Code != 404 {
			t.Errorf("error code = %v, want 404", errObj.Code)
		}
	} else {
		t.Error("error should be ErrorObject")
	}
}

func TestClient_Call_TransportError(t *testing.T) {
	mt := &mockTransport{
		err: errors.New("transport error"),
	}

	client := NewClient(mt)

	_, err := client.Call(context.Background(), "Test", nil)
	if err == nil {
		t.Error("Call() should return error when transport fails")
	}
}

func TestClient_Call_WithAuth(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`),
	}

	auth := &AuthData{
		Username: "admin",
		Password: "password",
	}

	client := NewClientWithAuth(mt, auth)

	_, err := client.Call(context.Background(), "Test", nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	// Verify auth was included in the request
	req, ok := mt.lastCall.(*Request)
	if !ok {
		t.Fatal("lastCall should be a *Request")
	}

	if req.Auth == nil {
		t.Error("auth should be included in request")
	}

	if req.Auth.Username != "admin" {
		t.Errorf("auth username = %v, want admin", req.Auth.Username)
	}
}

func TestClient_CallResult(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"jsonrpc":"2.0","id":1,"result":{"status":"on","brightness":50}}`),
	}

	client := NewClient(mt)

	var result struct {
		Status     string `json:"status"`
		Brightness int    `json:"brightness"`
	}

	err := client.CallResult(context.Background(), "Light.GetStatus", map[string]any{"id": 0}, &result)
	if err != nil {
		t.Fatalf("CallResult() error = %v", err)
	}

	if result.Status != "on" {
		t.Errorf("status = %v, want on", result.Status)
	}

	if result.Brightness != 50 {
		t.Errorf("brightness = %v, want 50", result.Brightness)
	}
}

func TestClient_CallResult_Error(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":404,"message":"Not found"}}`),
	}

	client := NewClient(mt)

	var result map[string]any
	err := client.CallResult(context.Background(), "Test", nil, &result)
	if err == nil {
		t.Error("CallResult() should return error for RPC error")
	}
}

func TestClient_CallResult_InvalidResult(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"jsonrpc":"2.0","id":1,"result":"not a struct"}`),
	}

	client := NewClient(mt)

	var result struct {
		Field string `json:"field"`
	}

	err := client.CallResult(context.Background(), "Test", nil, &result)
	if err == nil {
		t.Error("CallResult() should return error for invalid result type")
	}
}

func TestClient_Notify(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{}`), // Notifications don't expect a response, but return one anyway
	}

	client := NewClient(mt)

	err := client.Notify(context.Background(), "Test.Event", map[string]any{"value": 42})
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	// Verify the request was a notification (no ID)
	req, ok := mt.lastCall.(*Request)
	if !ok {
		t.Fatal("lastCall should be a *Request")
	}

	if !req.IsNotification() {
		t.Error("request should be a notification (no ID)")
	}
}

func TestClient_OnNotification(t *testing.T) {
	mt := &mockStatefulTransport{}
	client := NewClient(mt)

	called := false
	var receivedMethod string

	client.OnNotification(func(method string, params json.RawMessage) {
		called = true
		receivedMethod = method
	})

	// Simulate notification from transport
	notifData := json.RawMessage(`{"jsonrpc":"2.0","method":"NotifyStatus","params":{"status":"ok"}}`)
	mt.notificationHandler(notifData)

	if !called {
		t.Error("notification handler was not called")
	}

	if receivedMethod != "NotifyStatus" {
		t.Errorf("received method = %v, want NotifyStatus", receivedMethod)
	}
}

func TestClient_OnNotificationMethod(t *testing.T) {
	mt := &mockStatefulTransport{}
	client := NewClient(mt)

	called := false
	var receivedParams json.RawMessage

	client.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {
		called = true
		receivedParams = params
	})

	// Simulate notification from transport
	notifData := json.RawMessage(`{"jsonrpc":"2.0","method":"NotifyStatus","params":{"status":"ok"}}`)
	mt.notificationHandler(notifData)

	if !called {
		t.Error("method notification handler was not called")
	}

	if string(receivedParams) != `{"status":"ok"}` {
		t.Errorf("received params = %v, want %v", string(receivedParams), `{"status":"ok"}`)
	}
}

func TestClient_OnNotification_InvalidNotification(t *testing.T) {
	mt := &mockStatefulTransport{}
	client := NewClient(mt)

	called := false
	client.OnNotification(func(method string, params json.RawMessage) {
		called = true
	})

	// Simulate invalid notification from transport
	mt.notificationHandler([]byte(`{invalid}`))

	// Handler should not be called for invalid notification
	if called {
		t.Error("handler should not be called for invalid notification")
	}
}

func TestClient_RemoveNotificationHandlers(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	client.OnNotification(func(method string, params json.RawMessage) {})

	if !client.router.HasHandlers() {
		t.Error("router should have handlers")
	}

	client.RemoveNotificationHandlers()

	// Global handlers should be removed
	// (method handlers would remain if any were added)
}

func TestClient_RemoveMethodHandlers(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	client.OnNotificationMethod("NotifyStatus", func(params json.RawMessage) {})

	if !client.router.HasMethodHandlers("NotifyStatus") {
		t.Error("router should have method handlers")
	}

	client.RemoveMethodHandlers("NotifyStatus")

	if client.router.HasMethodHandlers("NotifyStatus") {
		t.Error("method handlers should be removed")
	}
}

func TestClient_RemoveAllHandlers(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	client.OnNotification(func(method string, params json.RawMessage) {})
	client.OnNotificationMethod("Test", func(params json.RawMessage) {})

	if !client.router.HasHandlers() {
		t.Error("router should have handlers")
	}

	client.RemoveAllHandlers()

	if client.router.HasHandlers() {
		t.Error("all handlers should be removed")
	}
}

func TestClient_SetAuth(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	if client.auth != nil {
		t.Error("client should not have auth initially")
	}

	auth := &AuthData{
		Username: "admin",
		Password: "password",
	}

	client.SetAuth(auth)

	if client.auth != auth {
		t.Error("auth not set correctly")
	}
}

func TestClient_ClearAuth(t *testing.T) {
	mt := &mockTransport{}
	auth := &AuthData{
		Username: "admin",
		Password: "password",
	}
	client := NewClientWithAuth(mt, auth)

	if client.auth == nil {
		t.Error("client should have auth")
	}

	client.ClearAuth()

	if client.auth != nil {
		t.Error("auth should be cleared")
	}
}

func TestClient_Close_Stateful(t *testing.T) {
	mt := &mockStatefulTransport{}
	client := NewClient(mt)

	err := client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if !mt.closed {
		t.Error("transport should be closed")
	}
}

func TestClient_Close_Stateless(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	// Should not error for stateless transport
	err := client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestClient_Transport(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	if client.Transport() != mt {
		t.Error("Transport() should return the underlying transport")
	}
}

func TestClient_RequestBuilder(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	builder := client.RequestBuilder()

	if builder == nil {
		t.Error("RequestBuilder() should not return nil")
	}

	if builder != client.builder {
		t.Error("RequestBuilder() should return the client's builder")
	}
}

func TestClient_NotificationRouter(t *testing.T) {
	mt := &mockTransport{}
	client := NewClient(mt)

	router := client.NotificationRouter()

	if router == nil {
		t.Error("NotificationRouter() should not return nil")
	}

	if router != client.router {
		t.Error("NotificationRouter() should return the client's router")
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`),
	}

	client := NewClient(mt)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The behavior depends on when the transport checks the context
	// We just verify it doesn't panic
	_, _ = client.Call(ctx, "Test", nil)
}

func TestClient_ConcurrentCalls(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`),
	}

	client := NewClient(mt)

	errChan := make(chan error, 10)

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			_, err := client.Call(context.Background(), "Test", nil)
			if err != nil {
				errChan <- err
			}
		})
	}

	// Wait for all calls to complete
	wg.Wait()
	close(errChan)

	// Check for errors after all goroutines complete
	for err := range errChan {
		t.Errorf("Call() error = %v", err)
	}
}

func TestClient_Call_InvalidParams(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`),
	}

	client := NewClient(mt)

	// Params that can't be marshaled to JSON (channels)
	_, err := client.Call(context.Background(), "Test", make(chan int))
	if err == nil {
		t.Error("Call() should return error for invalid params")
	}
}

func TestClient_Call_InvalidResponse(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{invalid}`),
	}

	client := NewClient(mt)

	_, err := client.Call(context.Background(), "Test", nil)
	if err == nil {
		t.Error("Call() should return error for invalid response")
	}
}

func TestClient_Notify_TransportError(t *testing.T) {
	mt := &mockTransport{
		err: errors.New("transport error"),
	}

	client := NewClient(mt)

	err := client.Notify(context.Background(), "Test", nil)
	if err == nil {
		t.Error("Notify() should return error when transport fails")
	}
}

func TestClient_Notify_InvalidParams(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{}`),
	}

	client := NewClient(mt)

	// Params that can't be marshaled to JSON
	err := client.Notify(context.Background(), "Test", make(chan int))
	if err == nil {
		t.Error("Notify() should return error for invalid params")
	}
}

func TestClient_CallResult_NilResult(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`),
	}

	client := NewClient(mt)

	// Passing nil result should not error
	err := client.CallResult(context.Background(), "Test", nil, nil)
	if err != nil {
		t.Errorf("CallResult() with nil result error = %v", err)
	}
}

func TestClient_CallResult_EmptyResult(t *testing.T) {
	mt := &mockTransport{
		response: []byte(`{"jsonrpc":"2.0","id":1,"result":null}`),
	}

	client := NewClient(mt)

	var result map[string]any
	err := client.CallResult(context.Background(), "Test", nil, &result)
	if err != nil {
		t.Errorf("CallResult() with null result error = %v", err)
	}
}

func TestNewHTTPClient(t *testing.T) {
	// Test basic creation - this won't actually connect, just verify creation
	client, err := NewHTTPClient("192.168.1.100")
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewHTTPClient() returned nil")
	}

	if client.transport == nil {
		t.Error("client transport should be initialized")
	}

	if client.builder == nil {
		t.Error("client builder should be initialized")
	}

	if client.router == nil {
		t.Error("client router should be initialized")
	}
}

func TestNewHTTPClient_WithOptions(t *testing.T) {
	client, err := NewHTTPClient("192.168.1.100",
		WithTimeout(30*time.Second),
		WithBasicAuth("admin", "password"),
	)
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewHTTPClient() returned nil")
	}

	// Verify the client was created (we can't easily verify options were applied
	// since they're internal to the transport, but we can verify it doesn't error)
	if client.transport == nil {
		t.Error("client transport should be initialized")
	}
}

func TestNewHTTPClient_WithURLScheme(t *testing.T) {
	// Test with explicit http scheme
	client, err := NewHTTPClient("http://192.168.1.100")
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewHTTPClient() returned nil")
	}

	// Test with explicit https scheme
	clientHTTPS, err := NewHTTPClient("https://192.168.1.100")
	if err != nil {
		t.Fatalf("NewHTTPClient(https) error = %v", err)
	}

	if clientHTTPS == nil {
		t.Fatal("NewHTTPClient(https) returned nil")
	}
}

func TestWithTimeout(t *testing.T) {
	opt := WithTimeout(30 * time.Second)
	options := defaultClientOptions()
	opt(options)

	if options.timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", options.timeout)
	}
}

func TestWithBasicAuth(t *testing.T) {
	opt := WithBasicAuth("admin", "secret")
	options := defaultClientOptions()
	opt(options)

	if options.username != "admin" {
		t.Errorf("username = %v, want admin", options.username)
	}

	if options.password != "secret" {
		t.Errorf("password = %v, want secret", options.password)
	}
}

func TestDefaultClientOptions(t *testing.T) {
	options := defaultClientOptions()

	if options.timeout != 10*time.Second {
		t.Errorf("default timeout = %v, want 10s", options.timeout)
	}

	if options.username != "" {
		t.Error("default username should be empty")
	}

	if options.password != "" {
		t.Error("default password should be empty")
	}
}
