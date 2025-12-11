package types

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Standard errors that can be returned by the library.
// Use errors.Is() to check for these errors.
var (
	// ErrNotFound is returned when a resource (device, component, etc.) is not found.
	ErrNotFound = errors.New("shelly: resource not found")

	// ErrAuth is returned when authentication fails or is required.
	ErrAuth = errors.New("shelly: authentication failed")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("shelly: operation timed out")

	// ErrNotSupported is returned when a feature is not supported by the device.
	ErrNotSupported = errors.New("shelly: feature not supported")

	// ErrInvalidParam is returned when a parameter is invalid.
	ErrInvalidParam = errors.New("shelly: invalid parameter")

	// ErrDeviceOffline is returned when a device is offline or unreachable.
	ErrDeviceOffline = errors.New("shelly: device offline")

	// ErrRPCMethod is returned when an RPC method call fails.
	ErrRPCMethod = errors.New("shelly: RPC method error")

	// ErrInvalidResponse is returned when a response cannot be parsed.
	ErrInvalidResponse = errors.New("shelly: invalid response")

	// ErrUnsupportedDevice is returned when a device type is not supported.
	ErrUnsupportedDevice = errors.New("shelly: unsupported device type")

	// ErrNilDevice is returned when a device is nil.
	ErrNilDevice = errors.New("shelly: nil device")
)

// Error represents an error response from a Shelly device.
// Gen2+ devices return structured error responses via RPC.
type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("shelly error %d: %s", e.Code, e.Message)
}

// Common RPC error codes.
const (
	ErrCodeInvalidArgument       = -103
	ErrCodeDeadlineExceeded      = -104
	ErrCodeNotFound              = -105
	ErrCodeResourceExhausted     = -108
	ErrCodeFailedPrecondition    = -109
	ErrCodeUnavailable           = -114
	ErrCodeUnauthorized          = -115
	ErrCodeMethodNotFound        = -106
	ErrCodeInvalidMethodParam    = -107
	ErrCodeComponentNotFound     = -101
	ErrCodeComponentConfigNotSet = -102
)

// ToStandardError converts a Shelly error code to a standard library error.
func (e *Error) ToStandardError() error {
	switch e.Code {
	case ErrCodeNotFound, ErrCodeComponentNotFound:
		return fmt.Errorf("%w: %s", ErrNotFound, e.Message)
	case ErrCodeUnauthorized:
		return fmt.Errorf("%w: %s", ErrAuth, e.Message)
	case ErrCodeDeadlineExceeded:
		return fmt.Errorf("%w: %s", ErrTimeout, e.Message)
	case ErrCodeMethodNotFound, ErrCodeComponentConfigNotSet:
		return fmt.Errorf("%w: %s", ErrNotSupported, e.Message)
	case ErrCodeInvalidArgument, ErrCodeInvalidMethodParam:
		return fmt.Errorf("%w: %s", ErrInvalidParam, e.Message)
	default:
		return e
	}
}

// MapErrorCode maps an RPC error code to a standard error.
// This is used by the RPC package to convert error codes to standard errors.
func MapErrorCode(code int) error {
	switch code {
	case ErrCodeNotFound, ErrCodeComponentNotFound:
		return ErrNotFound
	case ErrCodeUnauthorized:
		return ErrAuth
	case ErrCodeDeadlineExceeded:
		return ErrTimeout
	case ErrCodeMethodNotFound, ErrCodeComponentConfigNotSet:
		return ErrNotSupported
	case ErrCodeInvalidArgument, ErrCodeInvalidMethodParam:
		return ErrInvalidParam
	case ErrCodeUnavailable:
		return ErrDeviceOffline
	case -32600: // Invalid Request
		return ErrInvalidParam
	case -32601: // Method not found
		return ErrNotSupported
	case -32602: // Invalid params
		return ErrInvalidParam
	case -32603: // Internal error
		return ErrRPCMethod
	case 404: // HTTP Not Found
		return ErrNotFound
	case 401, 403: // HTTP Auth errors
		return ErrAuth
	default:
		return ErrRPCMethod
	}
}

// Response represents a generic response from a Shelly device.
// This is primarily used for RPC responses in Gen2+ devices.
type Response struct {
	Error  *Error          `json:"error,omitempty"`
	Src    string          `json:"src,omitempty"`
	Dst    string          `json:"dst,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	ID     int             `json:"id,omitempty"`
}

// IsError returns true if the response contains an error.
func (r *Response) IsError() bool {
	return r.Error != nil
}

// GetError returns the error if present, or nil.
func (r *Response) GetError() error {
	if r.Error == nil {
		return nil
	}
	return r.Error.ToStandardError()
}

// UnmarshalResult unmarshals the result into the provided value.
// Returns an error if the response contains an error or if unmarshaling fails.
func (r *Response) UnmarshalResult(v any) error {
	if r.IsError() {
		return r.GetError()
	}
	if r.Result == nil {
		return fmt.Errorf("%w: no result in response", ErrInvalidResponse)
	}
	if err := json.Unmarshal(r.Result, v); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return nil
}

// RPCError creates a new Error with the given code and message.
func RPCError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}
