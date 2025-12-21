package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/transport"
)

// restCall is a helper to make Gen1 REST API calls.
func restCall(ctx context.Context, t transport.Transport, path string) (json.RawMessage, error) {
	return t.Call(ctx, transport.NewSimpleRequest(path))
}

// Query parameter constants.
const (
	scheduleParam   = "schedule=true&"
	btnReverseParam = "btn_reverse=true&"
	boolTrue        = "true"
	boolFalse       = "false"
)

// lightConfigParams is an interface for building light config query parameters.
// Implemented by ColorConfig and WhiteConfig.
type lightConfigParams interface {
	getName() string
	getDefaultState() string
	getAutoOn() float64
	getAutoOff() float64
	getSchedule() bool
}

// buildLightConfigQuery builds a query parameter string from light config fields.
// Returns empty string if no parameters need to be set.
func buildLightConfigQuery(config lightConfigParams) string {
	params := ""
	if name := config.getName(); name != "" {
		params += fmt.Sprintf("name=%s&", name)
	}
	if ds := config.getDefaultState(); ds != "" {
		params += fmt.Sprintf("default_state=%s&", ds)
	}
	if ao := config.getAutoOn(); ao > 0 {
		params += fmt.Sprintf("auto_on=%v&", ao)
	}
	if aof := config.getAutoOff(); aof > 0 {
		params += fmt.Sprintf("auto_off=%v&", aof)
	}
	if config.getSchedule() {
		params += scheduleParam
	}

	if params == "" {
		return ""
	}

	// Remove trailing &
	return params[:len(params)-1]
}
