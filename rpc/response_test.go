package rpc

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/types"
)

func TestResponse_MarshalJSON(t *testing.T) {
	tests := []struct {
		resp *Response
		name string
	}{
		{
			name: "success response",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`{"status":"ok"}`),
			},
		},
		{
			name: "error response",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Error: &ErrorObject{
					Code:    -32600,
					Message: "Invalid Request",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.resp)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}

			var decoded Response
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			if decoded.JSONRPC != tt.resp.JSONRPC {
				t.Errorf("JSONRPC = %v, want %v", decoded.JSONRPC, tt.resp.JSONRPC)
			}
		})
	}
}

func TestResponse_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid success response",
			json:    `{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`,
			wantErr: false,
		},
		{
			name:    "valid error response",
			json:    `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}`,
			wantErr: false,
		},
		{
			name:    "invalid jsonrpc version",
			json:    `{"jsonrpc":"1.0","id":1,"result":{}}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp Response
			err := json.Unmarshal([]byte(tt.json), &resp)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResponse_IsError(t *testing.T) {
	tests := []struct {
		resp *Response
		name string
		want bool
	}{
		{
			name: "success response",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`{}`),
			},
			want: false,
		},
		{
			name: "error response",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Error:   &ErrorObject{Code: -32600, Message: "Error"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resp.IsError(); got != tt.want {
				t.Errorf("IsError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResponse_GetResult(t *testing.T) {
	tests := []struct {
		target  any
		resp    *Response
		name    string
		wantErr bool
	}{
		{
			name: "unmarshal map",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`{"status":"ok"}`),
			},
			target:  &map[string]any{},
			wantErr: false,
		},
		{
			name: "error response",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Error:   &ErrorObject{Code: -32600, Message: "Error"},
			},
			target:  &map[string]any{},
			wantErr: true,
		},
		{
			name: "empty result",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Result:  nil,
			},
			target:  &map[string]any{},
			wantErr: false,
		},
		{
			name: "invalid result JSON",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`{invalid}`),
			},
			target:  &map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resp.GetResult(tt.target)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResponse_GetError(t *testing.T) {
	tests := []struct {
		resp    *Response
		name    string
		wantErr bool
	}{
		{
			name: "success response",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`{}`),
			},
			wantErr: false,
		},
		{
			name: "error response",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Error:   &ErrorObject{Code: -32600, Message: "Error"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resp.GetError()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResponse_String(t *testing.T) {
	tests := []struct {
		name string
		resp *Response
		want string
	}{
		{
			name: "success response",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`{"status":"ok"}`),
			},
			want: `Response{ID: 1, Result: {"status":"ok"}}`,
		},
		{
			name: "error response",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Error:   &ErrorObject{Code: -32600, Message: "Error"},
			},
			want: "Response{ID: 1, Error: RPC error -32600: Error}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.resp.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorObject_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ErrorObject
		want string
	}{
		{
			name: "error without data",
			err:  &ErrorObject{Code: -32600, Message: "Invalid Request"},
			want: "RPC error -32600: Invalid Request",
		},
		{
			name: "error with data",
			err:  &ErrorObject{Code: -32600, Message: "Invalid Request", Data: json.RawMessage(`{"detail":"extra info"}`)},
			want: `RPC error -32600: Invalid Request (data: {"detail":"extra info"})`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorObject_Unwrap(t *testing.T) {
	err := &ErrorObject{Code: 404, Message: "Not found"}
	unwrapped := err.Unwrap()

	if !errors.Is(unwrapped, types.ErrNotFound) {
		t.Errorf("Unwrap() should return ErrNotFound")
	}

	if !errors.Is(err, types.ErrNotFound) {
		t.Errorf("errors.Is() should work with ErrorObject")
	}
}

func TestBatchResponse_MarshalJSON(t *testing.T) {
	batch := &BatchResponse{
		Responses: []*Response{
			{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`{"status":"ok"}`)},
			{JSONRPC: "2.0", ID: 2, Error: &ErrorObject{Code: -32600, Message: "Error"}},
		},
	}

	data, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var decoded BatchResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(decoded.Responses) != len(batch.Responses) {
		t.Errorf("Response count = %v, want %v", len(decoded.Responses), len(batch.Responses))
	}
}

func TestBatchResponse_Get(t *testing.T) {
	batch := &BatchResponse{
		Responses: []*Response{
			{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`{}`)},
			{JSONRPC: "2.0", ID: 2, Result: json.RawMessage(`{}`)},
		},
	}

	tests := []struct {
		name  string
		index int
		want  bool
	}{
		{name: "valid index 0", index: 0, want: true},
		{name: "valid index 1", index: 1, want: true},
		{name: "invalid index -1", index: -1, want: false},
		{name: "invalid index 2", index: 2, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := batch.Get(tt.index)
			if (got != nil) != tt.want {
				t.Errorf("Get(%v) = %v, want nil = %v", tt.index, got, !tt.want)
			}
		})
	}
}

func TestBatchResponse_GetByID(t *testing.T) {
	batch := &BatchResponse{
		Responses: []*Response{
			{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`{}`)},
			{JSONRPC: "2.0", ID: 2, Result: json.RawMessage(`{}`)},
		},
	}

	tests := []struct {
		id   any
		name string
		want bool
	}{
		{name: "valid ID 1", id: 1, want: true},
		{name: "valid ID 2", id: 2, want: true},
		{name: "invalid ID 3", id: 3, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := batch.GetByID(tt.id)
			if (got != nil) != tt.want {
				t.Errorf("GetByID(%v) = %v, want nil = %v", tt.id, got, !tt.want)
			}
		})
	}
}

func TestBatchResponse_Len(t *testing.T) {
	batch := &BatchResponse{
		Responses: []*Response{
			{JSONRPC: "2.0", ID: 1},
			{JSONRPC: "2.0", ID: 2},
			{JSONRPC: "2.0", ID: 3},
		},
	}

	if got := batch.Len(); got != 3 {
		t.Errorf("Len() = %v, want 3", got)
	}
}

func TestBatchResponse_HasErrors(t *testing.T) {
	tests := []struct {
		batch *BatchResponse
		name  string
		want  bool
	}{
		{
			name: "no errors",
			batch: &BatchResponse{
				Responses: []*Response{
					{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`{}`)},
				},
			},
			want: false,
		},
		{
			name: "has errors",
			batch: &BatchResponse{
				Responses: []*Response{
					{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`{}`)},
					{JSONRPC: "2.0", ID: 2, Error: &ErrorObject{Code: -32600, Message: "Error"}},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.batch.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBatchResponse_Errors(t *testing.T) {
	batch := &BatchResponse{
		Responses: []*Response{
			{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`{}`)},
			{JSONRPC: "2.0", ID: 2, Error: &ErrorObject{Code: -32600, Message: "Error 1"}},
			{JSONRPC: "2.0", ID: 3, Error: &ErrorObject{Code: -32601, Message: "Error 2"}},
		},
	}

	errs := batch.Errors()
	if len(errs) != 2 {
		t.Errorf("Errors() count = %v, want 2", len(errs))
	}
}

func TestBatchResponse_String(t *testing.T) {
	batch := &BatchResponse{
		Responses: []*Response{
			{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`{}`)},
			{JSONRPC: "2.0", ID: 2, Error: &ErrorObject{Code: -32600, Message: "Error"}},
		},
	}

	got := batch.String()
	want := "BatchResponse{Total: 2, Success: 1, Errors: 1}"
	if got != want {
		t.Errorf("String() = %v, want %v", got, want)
	}
}

func TestNotification_MarshalJSON(t *testing.T) {
	notif := &Notification{
		JSONRPC: "2.0",
		Method:  "NotifyStatus",
		Params:  json.RawMessage(`{"status":"ok"}`),
	}

	data, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var decoded Notification
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.Method != notif.Method {
		t.Errorf("Method = %v, want %v", decoded.Method, notif.Method)
	}
}

func TestNotification_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid notification",
			json:    `{"jsonrpc":"2.0","method":"NotifyStatus","params":{"status":"ok"}}`,
			wantErr: false,
		},
		{
			name:    "missing method",
			json:    `{"jsonrpc":"2.0","params":{}}`,
			wantErr: true,
		},
		{
			name:    "invalid jsonrpc version",
			json:    `{"jsonrpc":"1.0","method":"Notify"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var notif Notification
			err := json.Unmarshal([]byte(tt.json), &notif)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNotification_GetParams(t *testing.T) {
	notif := &Notification{
		JSONRPC: "2.0",
		Method:  "NotifyStatus",
		Params:  json.RawMessage(`{"status":"ok"}`),
	}

	var params map[string]any
	if err := notif.GetParams(&params); err != nil {
		t.Fatalf("GetParams() error = %v", err)
	}

	if params["status"] != "ok" {
		t.Errorf("status = %v, want ok", params["status"])
	}
}

func TestNotification_GetParams_EmptyParams(t *testing.T) {
	notif := &Notification{
		JSONRPC: "2.0",
		Method:  "NotifyStatus",
		Params:  nil,
	}

	var params map[string]any
	if err := notif.GetParams(&params); err != nil {
		t.Errorf("GetParams() with empty params should return nil, got error: %v", err)
	}
}

func TestNotification_GetParams_InvalidJSON(t *testing.T) {
	notif := &Notification{
		JSONRPC: "2.0",
		Method:  "NotifyStatus",
		Params:  json.RawMessage(`{invalid json}`),
	}

	var params map[string]any
	if err := notif.GetParams(&params); err == nil {
		t.Error("GetParams() with invalid JSON should return error")
	}
}

func TestNotification_String(t *testing.T) {
	notif := &Notification{
		JSONRPC: "2.0",
		Method:  "NotifyStatus",
		Params:  json.RawMessage(`{"status":"ok"}`),
	}

	got := notif.String()
	want := `Notification{Method: NotifyStatus, Params: {"status":"ok"}}`
	if got != want {
		t.Errorf("String() = %v, want %v", got, want)
	}
}

func TestParseResponse(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid response",
			json:    `{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseResponse([]byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseBatchResponse(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid batch",
			json:    `[{"jsonrpc":"2.0","id":1,"result":{}},{"jsonrpc":"2.0","id":2,"result":{}}]`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    `[{invalid}]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBatchResponse([]byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBatchResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseNotification(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid notification",
			json:    `{"jsonrpc":"2.0","method":"Notify","params":{}}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseNotification([]byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseNotification() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantType string
		wantErr  bool
	}{
		{
			name:     "response",
			json:     `{"jsonrpc":"2.0","id":1,"result":{}}`,
			wantErr:  false,
			wantType: "*rpc.Response",
		},
		{
			name:     "batch response",
			json:     `[{"jsonrpc":"2.0","id":1,"result":{}}]`,
			wantErr:  false,
			wantType: "*rpc.BatchResponse",
		},
		{
			name:     "notification",
			json:     `{"jsonrpc":"2.0","method":"Notify","params":{}}`,
			wantErr:  false,
			wantType: "*rpc.Notification",
		},
		{
			name:    "invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
		{
			name:    "unknown message type",
			json:    `{"jsonrpc":"2.0"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseMessage([]byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			gotType := ""
			switch msg.(type) {
			case *Response:
				gotType = "*rpc.Response"
			case *BatchResponse:
				gotType = "*rpc.BatchResponse"
			case *Notification:
				gotType = "*rpc.Notification"
			}

			if gotType != tt.wantType {
				t.Errorf("ParseMessage() type = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

func TestNewBatchResponse(t *testing.T) {
	responses := []*Response{
		{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`{}`)},
		{JSONRPC: "2.0", ID: 2, Result: json.RawMessage(`{}`)},
	}

	batch := NewBatchResponse(responses)

	if len(batch.Responses) != len(responses) {
		t.Errorf("Response count = %v, want %v", len(batch.Responses), len(responses))
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  *float64
	}{
		{"float64", float64(42.5), ptr(42.5)},
		{"float32", float32(42.5), ptr(42.5)},
		{"int", int(42), ptr(42.0)},
		{"int8", int8(42), ptr(42.0)},
		{"int16", int16(42), ptr(42.0)},
		{"int32", int32(42), ptr(42.0)},
		{"int64", int64(42), ptr(42.0)},
		{"uint", uint(42), ptr(42.0)},
		{"uint8", uint8(42), ptr(42.0)},
		{"uint16", uint16(42), ptr(42.0)},
		{"uint32", uint32(42), ptr(42.0)},
		{"uint64", uint64(42), ptr(42.0)},
		{"string", "not a number", nil},
		{"nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toFloat64(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("toFloat64() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Errorf("toFloat64() = nil, want %v", *tt.want)
				} else if *got != *tt.want {
					t.Errorf("toFloat64() = %v, want %v", *got, *tt.want)
				}
			}
		})
	}
}

func ptr(f float64) *float64 {
	return &f
}

func TestIdsEqual(t *testing.T) {
	tests := []struct {
		name string
		a    any
		b    any
		want bool
	}{
		{"same int", 42, 42, true},
		{"same float64", 42.0, 42.0, true},
		{"int vs float64", 42, 42.0, true},
		{"different int", 42, 43, false},
		{"same string", "abc", "abc", true},
		{"different string", "abc", "def", false},
		{"string vs int", "42", 42, false},
		{"nil vs nil", nil, nil, true},
		{"int8 vs int64", int8(42), int64(42), true},
		{"uint vs int", uint(42), int(42), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := idsEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("idsEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
