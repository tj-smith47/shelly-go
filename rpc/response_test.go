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
			name:    "array",
			json:    `[{"jsonrpc":"2.0","id":1,"result":{}}]`,
			wantErr: true,
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
			case *Notification:
				gotType = "*rpc.Notification"
			}

			if gotType != tt.wantType {
				t.Errorf("ParseMessage() type = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}
