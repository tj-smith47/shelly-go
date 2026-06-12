// Package jsonx holds small JSON (un)marshaling helpers shared across the Gen1
// device and component types, where older firmware is loose about field types.
package jsonx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// FlexBool is a bool that also unmarshals from the looser encodings older Gen1
// firmware emits for the same field: a JSON string ("true"/"false"/"1"/"0") or a
// number (0 or 1). Early Shelly Bulb Duo builds, for instance, report a meter's
// is_valid as the string "true" rather than the boolean true, which a plain bool
// field rejects with "cannot unmarshal string into ... of type bool". FlexBool
// accepts all of these and marshals back out as a normal JSON boolean.
type FlexBool bool

// Bool returns the plain bool value.
func (b FlexBool) Bool() bool { return bool(b) }

// UnmarshalJSON accepts boolean, numeric, or quoted-string encodings of a bool. A
// JSON null decodes to false.
func (b *FlexBool) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*b = false
		return nil
	}

	switch data[0] {
	case 't', 'f': // native JSON bool
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*b = FlexBool(v)
	case '"': // quoted: "true"/"false"/"1"/"0"
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		v, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("jsonx: cannot parse %q as bool: %w", s, err)
		}
		*b = FlexBool(v)
	default: // numeric: 0 = false, non-zero = true
		var n float64
		if err := json.Unmarshal(data, &n); err != nil {
			return fmt.Errorf("jsonx: cannot parse %s as bool: %w", data, err)
		}
		*b = FlexBool(n != 0)
	}
	return nil
}
