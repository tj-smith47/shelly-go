package components

import (
	"context"
	"fmt"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// EM1Data represents a Shelly Gen2+ EM1Data (Single-Phase Energy Data) component.
//
// EM1Data components store historical energy data for single-phase electrical systems.
// Unlike EM1 components (which provide real-time monitoring), EM1Data stores up to
// 60 days of measurements at configurable intervals in non-volatile memory.
//
// This component provides:
//   - Historical voltage, current, power measurements for single phase
//   - Configurable data collection period and retention
//   - Timestamp-based queries for energy consumption analysis
//   - CSV export via HTTP endpoint
//   - Data management (deletion) capabilities
//
// This component is typically found on devices like:
//   - Shelly Pro EM (single/dual-phase energy monitor)
//   - Shelly Pro EM-50 (professional energy monitor)
//
// Example:
//
//	em1data := components.NewEM1Data(device.Client(), 0)
//
//	// Configure data collection
//	err := em1data.SetConfig(ctx, &EM1DataConfig{
//	    DataPeriod:      ptr(60),  // 60-second intervals
//	    DataStorageDays: ptr(30),  // Keep 30 days
//	})
//
//	// Get data for last 24 hours
//	endTS := time.Now().Unix()
//	startTS := endTS - 86400
//	data, err := em1data.GetData(ctx, &startTS, &endTS)
//	if err == nil {
//	    for _, block := range data.Data {
//	        for _, values := range block.Values {
//	            fmt.Printf("Power: %.2fW\n", values.ActivePower)
//	        }
//	    }
//	}
type EM1Data struct {
	*gen2.BaseComponent
}

// NewEM1Data creates a new EM1Data component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (usually 0)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	em1data := components.NewEM1Data(device.Client(), 0)
func NewEM1Data(client *rpc.Client, id int) *EM1Data {
	return &EM1Data{
		BaseComponent: gen2.NewBaseComponent(client, "em1data", id),
	}
}

// EM1DataConfig represents the configuration of an EM1Data component.
type EM1DataConfig struct {
	Name            *string `json:"name,omitempty"`
	DataPeriod      *int    `json:"data_period,omitempty"`
	DataStorageDays *int    `json:"data_storage_days,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// EM1DataStatus represents the current status of an EM1Data component.
//
// It contains information about the data collection state, including the
// last record ID and total number of available records.
type EM1DataStatus struct {
	LastRecordID     *int `json:"last_record_id,omitempty"`
	AvailableRecords *int `json:"available_records,omitempty"`
	types.RawFields
	Errors []string `json:"errors,omitempty"`
	ID     int      `json:"id"`
}

// EM1DataRecordsResult contains the list of available time intervals with stored data.
type EM1DataRecordsResult struct {
	types.RawFields
	Records []EM1DataRecord `json:"records"`
}

// EM1DataRecord represents a time interval containing stored measurements.
type EM1DataRecord struct {
	types.RawFields
	ID     int   `json:"id"`
	TS     int64 `json:"ts"`
	Period int   `json:"period"`
	Count  int   `json:"count"`
}

// EM1DataGetDataResult contains historical measurement data.
type EM1DataGetDataResult struct {
	types.RawFields
	Data []EM1DataBlock `json:"data"`
	Keys []string       `json:"keys,omitempty"`
}

// EM1DataBlock represents a block of measurements for a specific time period.
//
// Note: The data array may contain multiple blocks if power loss or device
// restarts interrupted the recording sequence.
type EM1DataBlock struct {
	types.RawFields
	Values []EM1DataValues `json:"values"`
	TS     int64           `json:"ts"`
	Period int             `json:"period"`
}

// EM1DataValues represents single-phase measurements at a single point in time.
type EM1DataValues struct {
	PowerFactor  *float64 `json:"pf,omitempty"`
	Freq         *float64 `json:"freq,omitempty"`
	ActEnergy    *float64 `json:"act_energy,omitempty"`
	ActRetEnergy *float64 `json:"act_ret_energy,omitempty"`
	types.RawFields
	Voltage       float64 `json:"voltage"`
	Current       float64 `json:"current"`
	ActivePower   float64 `json:"act_power"`
	ApparentPower float64 `json:"aprt_power"`
}

// EM1DataGetDataParams contains parameters for the GetData method.
type EM1DataGetDataParams struct {
	TS    *int64 `json:"ts,omitempty"`
	EndTS *int64 `json:"end_ts,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// EM1DataGetRecordsParams contains parameters for the GetRecords method.
type EM1DataGetRecordsParams struct {
	TS *int64 `json:"ts,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// GetConfig retrieves the EM1Data configuration.
//
// Example:
//
//	config, err := em1data.GetConfig(ctx)
//	if err != nil {
//	    return err
//	}
//	if config.DataPeriod != nil {
//	    fmt.Printf("Collection interval: %d seconds\n", *config.DataPeriod)
//	}
//	if config.DataStorageDays != nil {
//	    fmt.Printf("Retention period: %d days\n", *config.DataStorageDays)
//	}
func (e *EM1Data) GetConfig(ctx context.Context) (*EM1DataConfig, error) {
	return gen2.UnmarshalConfig[EM1DataConfig](ctx, e.BaseComponent)
}

// SetConfig updates the EM1Data configuration.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - config: Configuration to apply
//
// Example:
//
//	// Configure 60-second intervals with 30-day retention
//	err := em1data.SetConfig(ctx, &EM1DataConfig{
//	    Name:            ptr("Main Meter Data"),
//	    DataPeriod:      ptr(60),  // 60 seconds
//	    DataStorageDays: ptr(30),  // 30 days
//	})
func (e *EM1Data) SetConfig(ctx context.Context, config *EM1DataConfig) error {
	return gen2.SetConfigWithID(ctx, e.BaseComponent, config)
}

// GetStatus retrieves the current EM1Data status.
//
// Returns information about the data collection state, including the last
// record ID and total number of available records.
//
// Example:
//
//	status, err := em1data.GetStatus(ctx)
//	if err != nil {
//	    return err
//	}
//	if status.LastRecordID != nil {
//	    fmt.Printf("Last record ID: %d\n", *status.LastRecordID)
//	}
//	if status.AvailableRecords != nil {
//	    fmt.Printf("Available records: %d\n", *status.AvailableRecords)
//	}
func (e *EM1Data) GetStatus(ctx context.Context) (*EM1DataStatus, error) {
	return gen2.UnmarshalStatus[EM1DataStatus](ctx, e.BaseComponent)
}

// GetRecords retrieves available time intervals containing stored data.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - fromTS: Optional Unix timestamp to get records from (nil = all records)
//
// Returns a list of time intervals that contain measurement data. Use these
// intervals to make targeted GetData requests.
//
// Example:
//
//	// Get all available records
//	records, err := em1data.GetRecords(ctx, nil)
//	if err != nil {
//	    return err
//	}
//
//	for _, record := range records.Records {
//	    fmt.Printf("Record %d: %d points from %v\n",
//	        record.ID, record.Count, time.Unix(record.TS, 0))
//	}
//
//	// Get records from specific timestamp
//	startTS := time.Now().Add(-24 * time.Hour).Unix()
//	recentRecords, err := em1data.GetRecords(ctx, &startTS)
func (e *EM1Data) GetRecords(ctx context.Context, fromTS *int64) (*EM1DataRecordsResult, error) {
	params := &EM1DataGetRecordsParams{
		ID: e.ID(),
		TS: fromTS,
	}

	return callEnergyDataMethod[EM1DataRecordsResult](ctx, e.BaseComponent, "GetRecords", params)
}

// GetData retrieves historical measurements for a specified time range.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - startTS: Optional start timestamp (Unix time). If nil, returns from earliest available data
//   - endTS: Optional end timestamp (Unix time). If nil, returns up to latest available data
//
// Note: Timestamps should ideally be multiples of the data period (configured in DataPeriod).
// Invalid timestamps may return an error.
//
// The returned data may contain multiple blocks if power loss or device restarts
// interrupted the recording sequence.
//
// Example:
//
//	// Get last 24 hours of data
//	endTS := time.Now().Unix()
//	startTS := endTS - 86400
//	data, err := em1data.GetData(ctx, &startTS, &endTS)
//	if err != nil {
//	    return err
//	}
//
//	var totalEnergy float64
//	for _, block := range data.Data {
//	    for _, values := range block.Values {
//	        // Calculate energy: Power (W) * Time (s) / 3600 = Wh
//	        energy := values.ActivePower * float64(block.Period) / 3600.0
//	        totalEnergy += energy
//	    }
//	}
//	fmt.Printf("Total energy consumption: %.2f Wh\n", totalEnergy)
func (e *EM1Data) GetData(ctx context.Context, startTS, endTS *int64) (*EM1DataGetDataResult, error) {
	params := &EM1DataGetDataParams{
		ID:    e.ID(),
		TS:    startTS,
		EndTS: endTS,
	}

	result, err := callEnergyDataMethod[EM1DataGetDataResult](ctx, e.BaseComponent, "GetData", params)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteAllData deletes all stored historical data.
//
// Warning: This operation cannot be undone. All measurement history will be lost.
//
// Example:
//
//	err := em1data.DeleteAllData(ctx)
//	if err != nil {
//	    return fmt.Errorf("failed to delete data: %w", err)
//	}
func (e *EM1Data) DeleteAllData(ctx context.Context) error {
	params := map[string]any{
		"id": e.ID(),
	}

	_, err := e.BaseComponent.Client().Call(ctx, "EM1Data.DeleteAllData", params)
	if err != nil {
		return fmt.Errorf("DeleteAllData failed for %s: %w", e.Key(), err)
	}

	return nil
}

// GetDataCSVURL returns the HTTP URL for downloading historical data as CSV.
//
// Parameters:
//   - deviceAddr: Device IP address or hostname (e.g., "192.168.1.100")
//   - startTS: Optional start timestamp. If nil, exports from earliest data
//   - endTS: Optional end timestamp. If nil, exports up to latest data
//   - addKeys: If true, includes column headers in the CSV
//
// Note: This method only generates the URL. You must perform the HTTP GET request
// separately. Authentication may be required depending on device settings.
//
// Example:
//
//	// Generate CSV download URL for last week
//	endTS := time.Now().Unix()
//	startTS := endTS - (7 * 86400)
//	url := em1data.GetDataCSVURL("192.168.1.100", &startTS, &endTS, true)
//	fmt.Printf("Download CSV: %s\n", url)
//
//	// Download with curl:
//	// curl -OJ "http://192.168.1.100/em1data/0/data.csv?ts=1234567890&end_ts=1234654290&add_keys=true"
func (e *EM1Data) GetDataCSVURL(deviceAddr string, startTS, endTS *int64, addKeys bool) string {
	return buildDataCSVURL("em1data", deviceAddr, e.ID(), startTS, endTS, addKeys)
}
