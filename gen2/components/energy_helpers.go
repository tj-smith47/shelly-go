package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tj-smith47/shelly-go/gen2"
)

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
