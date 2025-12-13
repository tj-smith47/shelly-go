package components

import (
	"context"
	"fmt"

	"github.com/tj-smith47/shelly-go/gen2"
	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/types"
)

// EMData represents a Shelly Gen2+ EMData (3-Phase Energy Data) component.
//
// EMData components store historical energy data for 3-phase electrical systems.
// Unlike EM components (which provide real-time monitoring), EMData stores up to
// 60 days of measurements at 1-minute intervals in non-volatile memory.
//
// This component provides:
//   - Historical voltage, current, power measurements per phase
//   - Timestamp-based queries for energy consumption analysis
//   - CSV export via HTTP endpoint
//   - Data management (deletion) capabilities
//
// This component is typically found on devices like:
//   - Shelly Pro 3EM (3-phase energy monitor)
//   - Shelly Pro EM-50 (professional energy monitor)
//
// Example:
//
//	emdata := components.NewEMData(device.Client(), 0)
//
//	// Get data for last 24 hours
//	endTS := time.Now().Unix()
//	startTS := endTS - 86400
//	data, err := emdata.GetData(ctx, &startTS, &endTS)
//	if err == nil {
//	    for _, block := range data.Data {
//	        for _, values := range block.Values {
//	            fmt.Printf("Total Power: %.2fW\n", values.TotalActivePower)
//	        }
//	    }
//	}
type EMData struct {
	*gen2.BaseComponent
}

// NewEMData creates a new EMData component accessor.
//
// Parameters:
//   - client: RPC client for communication
//   - id: Component ID (usually 0)
//
// Example:
//
//	device := gen2.NewDevice(rpcClient)
//	emdata := components.NewEMData(device.Client(), 0)
func NewEMData(client *rpc.Client, id int) *EMData {
	return &EMData{
		BaseComponent: gen2.NewBaseComponent(client, "emdata", id),
	}
}

// EMDataStatus represents the current status of an EMData component.
//
// It contains information about the data collection state, including the
// last record ID and total number of available records.
type EMDataStatus struct {
	LastRecordID     *int `json:"last_record_id,omitempty"`
	AvailableRecords *int `json:"available_records,omitempty"`
	types.RawFields
	Errors []string `json:"errors,omitempty"`
	ID     int      `json:"id"`
}

// EMDataRecordsResult contains the list of available time intervals with stored data.
type EMDataRecordsResult struct {
	types.RawFields
	Records []EMDataRecord `json:"records"`
}

// EMDataRecord represents a time interval containing stored measurements.
type EMDataRecord struct {
	types.RawFields
	ID     int   `json:"id"`
	TS     int64 `json:"ts"`
	Period int   `json:"period"`
	Count  int   `json:"count"`
}

// EMDataGetDataResult contains historical measurement data.
type EMDataGetDataResult struct {
	types.RawFields
	Data []EMDataBlock `json:"data"`
	Keys []string      `json:"keys,omitempty"`
}

// EMDataBlock represents a block of measurements for a specific time period.
//
// Note: The data array may contain multiple blocks if power loss or device
// restarts interrupted the recording sequence.
type EMDataBlock struct {
	types.RawFields
	Values []EMDataValues `json:"values"`
	TS     int64          `json:"ts"`
	Period int            `json:"period"`
}

// EMDataValues represents measurements for all three phases at a single point in time.
type EMDataValues struct {
	BPowerFactor *float64 `json:"b_pf,omitempty"`
	types.RawFields
	TotalActRetEnergy *float64 `json:"total_act_ret_energy,omitempty"`
	TotalActEnergy    *float64 `json:"total_act_energy,omitempty"`
	APowerFactor      *float64 `json:"a_pf,omitempty"`
	AFreq             *float64 `json:"a_freq,omitempty"`
	NCurrent          *float64 `json:"n_current,omitempty"`
	CFreq             *float64 `json:"c_freq,omitempty"`
	CPowerFactor      *float64 `json:"c_pf,omitempty"`
	BFreq             *float64 `json:"b_freq,omitempty"`
	CVoltage          float64  `json:"c_voltage"`
	BCurrent          float64  `json:"b_current"`
	AVoltage          float64  `json:"a_voltage"`
	CCurrent          float64  `json:"c_current"`
	CActivePower      float64  `json:"c_act_power"`
	CApparentPower    float64  `json:"c_aprt_power"`
	BActivePower      float64  `json:"b_act_power"`
	BApparentPower    float64  `json:"b_aprt_power"`
	TotalCurrent      float64  `json:"total_current"`
	TotalActivePower  float64  `json:"total_act_power"`
	TotalAprtPower    float64  `json:"total_aprt_power"`
	BVoltage          float64  `json:"b_voltage"`
	AApparentPower    float64  `json:"a_aprt_power"`
	AActivePower      float64  `json:"a_act_power"`
	ACurrent          float64  `json:"a_current"`
}

// EMDataGetDataParams contains parameters for the GetData method.
type EMDataGetDataParams struct {
	TS    *int64 `json:"ts,omitempty"`
	EndTS *int64 `json:"end_ts,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// EMDataGetRecordsParams contains parameters for the GetRecords method.
type EMDataGetRecordsParams struct {
	TS *int64 `json:"ts,omitempty"`
	types.RawFields
	ID int `json:"id"`
}

// GetStatus retrieves the current EMData status.
//
// Returns information about the data collection state, including the last
// record ID and total number of available records.
//
// Example:
//
//	status, err := emdata.GetStatus(ctx)
//	if err != nil {
//	    return err
//	}
//	if status.LastRecordID != nil {
//	    fmt.Printf("Last record ID: %d\n", *status.LastRecordID)
//	}
//	if status.AvailableRecords != nil {
//	    fmt.Printf("Available records: %d\n", *status.AvailableRecords)
//	}
func (e *EMData) GetStatus(ctx context.Context) (*EMDataStatus, error) {
	return gen2.UnmarshalStatus[EMDataStatus](ctx, e.BaseComponent)
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
//	records, err := emdata.GetRecords(ctx, nil)
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
//	recentRecords, err := emdata.GetRecords(ctx, &startTS)
func (e *EMData) GetRecords(ctx context.Context, fromTS *int64) (*EMDataRecordsResult, error) {
	params := &EMDataGetRecordsParams{
		ID: e.ID(),
		TS: fromTS,
	}

	return callEnergyDataMethod[EMDataRecordsResult](ctx, e.BaseComponent, "GetRecords", params)
}

// GetData retrieves historical measurements for a specified time range.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - startTS: Optional start timestamp (Unix time). If nil, returns from earliest available data
//   - endTS: Optional end timestamp (Unix time). If nil, returns up to latest available data
//
// Note: Timestamps should ideally be multiples of the data period (typically 60 seconds).
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
//	data, err := emdata.GetData(ctx, &startTS, &endTS)
//	if err != nil {
//	    return err
//	}
//
//	var totalEnergy float64
//	for _, block := range data.Data {
//	    for _, values := range block.Values {
//	        // Calculate energy: Power (W) * Time (s) / 3600 = Wh
//	        energy := values.TotalActivePower * float64(block.Period) / 3600.0
//	        totalEnergy += energy
//	    }
//	}
//	fmt.Printf("Total energy consumption: %.2f Wh\n", totalEnergy)
func (e *EMData) GetData(ctx context.Context, startTS, endTS *int64) (*EMDataGetDataResult, error) {
	params := &EMDataGetDataParams{
		ID:    e.ID(),
		TS:    startTS,
		EndTS: endTS,
	}

	result, err := callEnergyDataMethod[EMDataGetDataResult](ctx, e.BaseComponent, "GetData", params)
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
//	err := emdata.DeleteAllData(ctx)
//	if err != nil {
//	    return fmt.Errorf("failed to delete data: %w", err)
//	}
func (e *EMData) DeleteAllData(ctx context.Context) error {
	params := map[string]any{
		"id": e.ID(),
	}

	_, err := e.BaseComponent.Client().Call(ctx, "EMData.DeleteAllData", params)
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
//	url := emdata.GetDataCSVURL("192.168.1.100", &startTS, &endTS, true)
//	fmt.Printf("Download CSV: %s\n", url)
//
//	// Download with curl:
//	// curl -OJ "http://192.168.1.100/emdata/0/data.csv?ts=1234567890&end_ts=1234654290&add_keys=true"
func (e *EMData) GetDataCSVURL(deviceAddr string, startTS, endTS *int64, addKeys bool) string {
	return buildDataCSVURL("emdata", deviceAddr, e.ID(), startTS, endTS, addKeys)
}
