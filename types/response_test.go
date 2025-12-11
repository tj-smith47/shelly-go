package types

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "basic error",
			err:  &Error{Code: -105, Message: "not found"},
			want: "shelly error -105: not found",
		},
		{
			name: "auth error",
			err:  &Error{Code: -115, Message: "unauthorized"},
			want: "shelly error -115: unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestError_ToStandardError(t *testing.T) {
	tests := []struct {
		wantErr error
		err     *Error
		name    string
	}{
		{
			name:    "not found error",
			err:     &Error{Code: ErrCodeNotFound, Message: "resource not found"},
			wantErr: ErrNotFound,
		},
		{
			name:    "component not found",
			err:     &Error{Code: ErrCodeComponentNotFound, Message: "component not found"},
			wantErr: ErrNotFound,
		},
		{
			name:    "unauthorized error",
			err:     &Error{Code: ErrCodeUnauthorized, Message: "auth failed"},
			wantErr: ErrAuth,
		},
		{
			name:    "timeout error",
			err:     &Error{Code: ErrCodeDeadlineExceeded, Message: "deadline exceeded"},
			wantErr: ErrTimeout,
		},
		{
			name:    "method not found",
			err:     &Error{Code: ErrCodeMethodNotFound, Message: "method not found"},
			wantErr: ErrNotSupported,
		},
		{
			name:    "invalid argument",
			err:     &Error{Code: ErrCodeInvalidArgument, Message: "bad param"},
			wantErr: ErrInvalidParam,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.ToStandardError()
			if !errors.Is(got, tt.wantErr) {
				t.Errorf("Error.ToStandardError() = %v, want error matching %v", got, tt.wantErr)
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
			name: "response with error",
			resp: &Response{
				Error: &Error{Code: -105, Message: "not found"},
			},
			want: true,
		},
		{
			name: "response without error",
			resp: &Response{
				Result: json.RawMessage(`{"success":true}`),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resp.IsError(); got != tt.want {
				t.Errorf("Response.IsError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResponse_GetError(t *testing.T) {
	tests := []struct {
		wantIs  error
		resp    *Response
		name    string
		wantErr bool
	}{
		{
			name: "response with not found error",
			resp: &Response{
				Error: &Error{Code: ErrCodeNotFound, Message: "not found"},
			},
			wantErr: true,
			wantIs:  ErrNotFound,
		},
		{
			name: "response without error",
			resp: &Response{
				Result: json.RawMessage(`{"success":true}`),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resp.GetError()
			if (err != nil) != tt.wantErr {
				t.Errorf("Response.GetError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("Response.GetError() = %v, want error matching %v", err, tt.wantIs)
			}
		})
	}
}

func TestResponse_UnmarshalResult(t *testing.T) {
	type testResult struct {
		Value  string `json:"value"`
		Number int    `json:"number"`
	}

	tests := []struct {
		resp    *Response
		want    *testResult
		name    string
		wantErr bool
	}{
		{
			name: "successful unmarshal",
			resp: &Response{
				Result: json.RawMessage(`{"value":"test","number":42}`),
			},
			want: &testResult{
				Value:  "test",
				Number: 42,
			},
			wantErr: false,
		},
		{
			name: "response with error",
			resp: &Response{
				Error: &Error{Code: -105, Message: "not found"},
			},
			wantErr: true,
		},
		{
			name: "response with no result",
			resp: &Response{
				ID: 1,
			},
			wantErr: true,
		},
		{
			name: "invalid json",
			resp: &Response{
				Result: json.RawMessage(`invalid`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got testResult
			err := tt.resp.UnmarshalResult(&got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Response.UnmarshalResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Value != tt.want.Value || got.Number != tt.want.Number {
					t.Errorf("Response.UnmarshalResult() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestRPCError(t *testing.T) {
	err := RPCError(ErrCodeNotFound, "test message")
	if err.Code != ErrCodeNotFound {
		t.Errorf("RPCError().Code = %v, want %v", err.Code, ErrCodeNotFound)
	}
	if err.Message != "test message" {
		t.Errorf("RPCError().Message = %v, want %v", err.Message, "test message")
	}
}

func TestResponse_JSON(t *testing.T) {
	t.Run("unmarshal response with result", func(t *testing.T) {
		input := `{"id":1,"result":{"status":"ok"}}`
		var resp Response
		err := json.Unmarshal([]byte(input), &resp)
		if err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if resp.ID != 1 {
			t.Errorf("Response.ID = %v, want 1", resp.ID)
		}
		if resp.Result == nil {
			t.Error("Response.Result is nil")
		}
	})

	t.Run("unmarshal response with error", func(t *testing.T) {
		input := `{"id":2,"error":{"code":-105,"message":"not found"}}`
		var resp Response
		err := json.Unmarshal([]byte(input), &resp)
		if err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if resp.ID != 2 {
			t.Errorf("Response.ID = %v, want 2", resp.ID)
		}
		if resp.Error == nil {
			t.Fatal("Response.Error is nil")
		}
		if resp.Error.Code != -105 {
			t.Errorf("Response.Error.Code = %v, want -105", resp.Error.Code)
		}
	})

	t.Run("marshal response", func(t *testing.T) {
		resp := Response{
			ID:     3,
			Result: json.RawMessage(`{"ok":true}`),
		}
		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		var decoded Response
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if decoded.ID != resp.ID {
			t.Errorf("decoded.ID = %v, want %v", decoded.ID, resp.ID)
		}
	})
}

func TestMapErrorCode(t *testing.T) {
	tests := []struct {
		want error
		name string
		code int
	}{
		{
			name: "not found",
			code: ErrCodeNotFound,
			want: ErrNotFound,
		},
		{
			name: "component not found",
			code: ErrCodeComponentNotFound,
			want: ErrNotFound,
		},
		{
			name: "unauthorized",
			code: ErrCodeUnauthorized,
			want: ErrAuth,
		},
		{
			name: "deadline exceeded",
			code: ErrCodeDeadlineExceeded,
			want: ErrTimeout,
		},
		{
			name: "method not found",
			code: ErrCodeMethodNotFound,
			want: ErrNotSupported,
		},
		{
			name: "component config not set",
			code: ErrCodeComponentConfigNotSet,
			want: ErrNotSupported,
		},
		{
			name: "invalid argument",
			code: ErrCodeInvalidArgument,
			want: ErrInvalidParam,
		},
		{
			name: "invalid method param",
			code: ErrCodeInvalidMethodParam,
			want: ErrInvalidParam,
		},
		{
			name: "unavailable",
			code: ErrCodeUnavailable,
			want: ErrDeviceOffline,
		},
		{
			name: "JSON-RPC invalid request",
			code: -32600,
			want: ErrInvalidParam,
		},
		{
			name: "JSON-RPC method not found",
			code: -32601,
			want: ErrNotSupported,
		},
		{
			name: "JSON-RPC invalid params",
			code: -32602,
			want: ErrInvalidParam,
		},
		{
			name: "JSON-RPC internal error",
			code: -32603,
			want: ErrRPCMethod,
		},
		{
			name: "HTTP not found",
			code: 404,
			want: ErrNotFound,
		},
		{
			name: "HTTP unauthorized",
			code: 401,
			want: ErrAuth,
		},
		{
			name: "HTTP forbidden",
			code: 403,
			want: ErrAuth,
		},
		{
			name: "unknown error code",
			code: 999,
			want: ErrRPCMethod,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapErrorCode(tt.code)
			if !errors.Is(got, tt.want) {
				t.Errorf("MapErrorCode(%v) = %v, want error matching %v", tt.code, got, tt.want)
			}
		})
	}
}
