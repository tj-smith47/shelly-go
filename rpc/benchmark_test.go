package rpc

import (
	"encoding/json"
	"testing"
)

// BenchmarkRequest_Marshal benchmarks JSON marshaling of RPC requests.
func BenchmarkRequest_Marshal(b *testing.B) {
	rb := NewRequestBuilder()
	req, _ := rb.Build("Shelly.GetDeviceInfo", map[string]any{"key": "value", "number": 42})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

// BenchmarkResponse_Unmarshal benchmarks JSON unmarshaling of RPC responses.
func BenchmarkResponse_Unmarshal(b *testing.B) {
	data := []byte(`{
		"id": 1,
		"jsonrpc": "2.0",
		"result": {
			"name": "Test Device",
			"id": "shellyplus1pm-aabbcc",
			"mac": "AA:BB:CC:DD:EE:FF",
			"model": "SNSW-001P16EU",
			"gen": 2,
			"fw_id": "20230913-114244/v1.0.0-g1234567",
			"ver": "1.0.0",
			"auth_en": false
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resp Response
		_ = json.Unmarshal(data, &resp)
	}
}

// BenchmarkResponse_Unmarshal_Error benchmarks error response unmarshaling.
func BenchmarkResponse_Unmarshal_Error(b *testing.B) {
	data := []byte(`{
		"id": 1,
		"jsonrpc": "2.0",
		"error": {
			"code": -32602,
			"message": "Invalid params"
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resp Response
		_ = json.Unmarshal(data, &resp)
	}
}

// BenchmarkRequestBuilder_Build benchmarks request building.
func BenchmarkRequestBuilder_Build(b *testing.B) {
	rb := NewRequestBuilder()
	params := map[string]any{"id": 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rb.Build("Switch.GetStatus", params)
	}
}

// BenchmarkBatchResponse_Unmarshal benchmarks batch response unmarshaling.
func BenchmarkBatchResponse_Unmarshal(b *testing.B) {
	data := []byte(`[
		{"id":1,"jsonrpc":"2.0","result":{"id":0,"source":"init","output":true}},
		{"id":2,"jsonrpc":"2.0","result":{"id":0,"name":"Switch 0"}},
		{"id":3,"jsonrpc":"2.0","result":{"uptime":12345}},
		{"id":4,"jsonrpc":"2.0","result":{"sta_ip":"192.168.1.100"}},
		{"id":5,"jsonrpc":"2.0","result":{"connected":true}}
	]`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var responses []Response
		_ = json.Unmarshal(data, &responses)
	}
}

// BenchmarkRequest_Large benchmarks large request marshaling.
func BenchmarkRequest_Large(b *testing.B) {
	rb := NewRequestBuilder()
	largeParams := map[string]any{
		"id":     0,
		"name":   "test_script",
		"enable": true,
		"code": `
			let counter = 0;
			Timer.set(1000, true, function() {
				counter++;
				print("Counter: " + counter);
				if (counter > 100) {
					counter = 0;
				}
			});
		`,
	}

	req, _ := rb.Build("Script.PutCode", largeParams)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

// BenchmarkParseResult benchmarks result parsing into struct.
func BenchmarkParseResult(b *testing.B) {
	data := []byte(`{
		"id": 1,
		"jsonrpc": "2.0",
		"result": {
			"name": "Test Device",
			"id": "shellyplus1pm-aabbcc",
			"mac": "AA:BB:CC:DD:EE:FF"
		}
	}`)

	type DeviceInfo struct {
		Name string `json:"name"`
		ID   string `json:"id"`
		MAC  string `json:"mac"`
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resp Response
		_ = json.Unmarshal(data, &resp)
		var info DeviceInfo
		_ = json.Unmarshal(resp.Result, &info)
	}
}
