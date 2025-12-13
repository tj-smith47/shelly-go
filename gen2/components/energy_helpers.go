package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/gen2"
)

// EMDataCSVURL generates a CSV download URL for 3-phase energy monitor data.
//
// This is a convenience function that doesn't require creating an EMData instance.
// Use this when you only need to generate the URL without performing RPC calls.
//
// Parameters:
//   - deviceAddr: Device IP address or hostname (e.g., "192.168.1.100")
//   - id: Component ID (usually 0)
//   - startTS: Optional start timestamp (Unix seconds). If nil, exports from earliest data
//   - endTS: Optional end timestamp (Unix seconds). If nil, exports up to latest data
//   - addKeys: If true, includes column headers in the CSV
//
// Example:
//
//	// Generate CSV download URL for last 24 hours
//	endTS := time.Now().Unix()
//	startTS := endTS - 86400
//	url := components.EMDataCSVURL("192.168.1.100", 0, &startTS, &endTS, true)
//	fmt.Printf("Download with: curl -OJ %q\n", url)
func EMDataCSVURL(deviceAddr string, id int, startTS, endTS *int64, addKeys bool) string {
	return buildDataCSVURL("emdata", deviceAddr, id, startTS, endTS, addKeys)
}

// EM1DataCSVURL generates a CSV download URL for single-phase energy monitor data.
//
// This is a convenience function that doesn't require creating an EM1Data instance.
// Use this when you only need to generate the URL without performing RPC calls.
//
// Parameters:
//   - deviceAddr: Device IP address or hostname (e.g., "192.168.1.100")
//   - id: Component ID (usually 0)
//   - startTS: Optional start timestamp (Unix seconds). If nil, exports from earliest data
//   - endTS: Optional end timestamp (Unix seconds). If nil, exports up to latest data
//   - addKeys: If true, includes column headers in the CSV
//
// Example:
//
//	// Generate CSV download URL for last week
//	endTS := time.Now().Unix()
//	startTS := endTS - (7 * 86400)
//	url := components.EM1DataCSVURL("192.168.1.100", 0, &startTS, &endTS, true)
//	fmt.Printf("Download with: curl -OJ %q\n", url)
func EM1DataCSVURL(deviceAddr string, id int, startTS, endTS *int64, addKeys bool) string {
	return buildDataCSVURL("em1data", deviceAddr, id, startTS, endTS, addKeys)
}

// buildDataCSVURL constructs a CSV export URL for energy data components.
// This is a shared helper for EMData and EM1Data components.
func buildDataCSVURL(componentType, deviceAddr string, id int, startTS, endTS *int64, addKeys bool) string {
	url := fmt.Sprintf("http://%s/%s/%d/data.csv?", deviceAddr, componentType, id)

	params := make([]string, 0, 3)

	if startTS != nil {
		params = append(params, fmt.Sprintf("ts=%d", *startTS))
	}

	if endTS != nil {
		params = append(params, fmt.Sprintf("end_ts=%d", *endTS))
	}

	if addKeys {
		params = append(params, "add_keys=true")
	}

	// Join params with &
	for i, param := range params {
		url += param
		if i < len(params)-1 {
			url += "&"
		}
	}

	return url
}

// callEnergyDataMethod is a generic helper for making RPC calls to energy data components.
// It handles the common pattern of calling a method and unmarshaling the result.
func callEnergyDataMethod[T any](ctx context.Context, c *gen2.BaseComponent, method string, params any) (*T, error) {
	// Build the full method name using proper capitalization
	componentType := c.Type()
	var capitalizedType string
	switch componentType {
	case "emdata":
		capitalizedType = "EMData"
	case "em1data":
		capitalizedType = "EM1Data"
	default:
		capitalizedType = componentType
	}

	fullMethod := fmt.Sprintf("%s.%s", capitalizedType, method)

	resultJSON, err := c.Client().Call(ctx, fullMethod, params)
	if err != nil {
		return nil, fmt.Errorf("%s failed for %s: %w", method, c.Key(), err)
	}

	var result T
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
