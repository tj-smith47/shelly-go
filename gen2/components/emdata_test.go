package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewEMData(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	emdata := NewEMData(client, 0)

	if emdata == nil {
		t.Fatal("NewEMData returned nil")
	}

	if emdata.Type() != "emdata" {
		t.Errorf("Type() = %q, want %q", emdata.Type(), "emdata")
	}

	if emdata.ID() != 0 {
		t.Errorf("ID() = %d, want %d", emdata.ID(), 0)
	}

	if emdata.Key() != "emdata:0" {
		t.Errorf("Key() = %q, want %q", emdata.Key(), "emdata:0")
	}
}

func TestEMData_GetStatus(t *testing.T) {
	tests := []struct {
		name                 string
		result               string
		wantID               int
		wantLastRecordID     *int
		wantAvailableRecords *int
		wantErrorsLen        int
	}{
		{
			name: "full status",
			result: `{
				"id": 0,
				"last_record_id": 12345,
				"available_records": 1440
			}`,
			wantID:               0,
			wantLastRecordID:     ptr(12345),
			wantAvailableRecords: ptr(1440),
			wantErrorsLen:        0,
		},
		{
			name: "minimal status",
			result: `{
				"id": 0
			}`,
			wantID:               0,
			wantLastRecordID:     nil,
			wantAvailableRecords: nil,
			wantErrorsLen:        0,
		},
		{
			name: "with errors",
			result: `{
				"id": 0,
				"last_record_id": 100,
				"available_records": 50,
				"errors": ["storage_error", "overflow"]
			}`,
			wantID:               0,
			wantLastRecordID:     ptr(100),
			wantAvailableRecords: ptr(50),
			wantErrorsLen:        2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "EMData.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			emdata := NewEMData(client, 0)

			status, err := emdata.GetStatus(context.Background())
			if err != nil {
				t.Errorf("GetStatus() error = %v", err)
				return
			}

			if status == nil {
				t.Fatal("GetStatus() returned nil status")
			}

			if status.ID != tt.wantID {
				t.Errorf("status.ID = %d, want %d", status.ID, tt.wantID)
			}

			if tt.wantLastRecordID != nil {
				if status.LastRecordID == nil {
					t.Errorf("status.LastRecordID = nil, want %d", *tt.wantLastRecordID)
				} else if *status.LastRecordID != *tt.wantLastRecordID {
					t.Errorf("status.LastRecordID = %d, want %d", *status.LastRecordID, *tt.wantLastRecordID)
				}
			} else if status.LastRecordID != nil {
				t.Errorf("status.LastRecordID = %d, want nil", *status.LastRecordID)
			}

			if tt.wantAvailableRecords != nil {
				if status.AvailableRecords == nil {
					t.Errorf("status.AvailableRecords = nil, want %d", *tt.wantAvailableRecords)
				} else if *status.AvailableRecords != *tt.wantAvailableRecords {
					t.Errorf("status.AvailableRecords = %d, want %d", *status.AvailableRecords, *tt.wantAvailableRecords)
				}
			} else if status.AvailableRecords != nil {
				t.Errorf("status.AvailableRecords = %d, want nil", *status.AvailableRecords)
			}

			if len(status.Errors) != tt.wantErrorsLen {
				t.Errorf("len(status.Errors) = %d, want %d", len(status.Errors), tt.wantErrorsLen)
			}
		})
	}
}

func TestEMData_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	emdata := NewEMData(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := emdata.GetStatus(context.Background())
		return err
	})
}

func TestEMData_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	emdata := NewEMData(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := emdata.GetStatus(context.Background())
		return err
	})
}

func TestEMData_GetRecords(t *testing.T) {
	tests := []struct {
		name            string
		fromTS          *int64
		result          string
		wantRecordsLen  int
		wantFirstRecID  int
		wantFirstRecTS  int64
		wantFirstPeriod int
		wantFirstCount  int
	}{
		{
			name:   "multiple records",
			fromTS: nil,
			result: `{
				"records": [
					{
						"id": 1,
						"ts": 1656356400,
						"period": 60,
						"count": 1440
					},
					{
						"id": 2,
						"ts": 1656442800,
						"period": 60,
						"count": 1440
					}
				]
			}`,
			wantRecordsLen:  2,
			wantFirstRecID:  1,
			wantFirstRecTS:  1656356400,
			wantFirstPeriod: 60,
			wantFirstCount:  1440,
		},
		{
			name:   "with timestamp filter",
			fromTS: ptr(int64(1656356400)),
			result: `{
				"records": [
					{
						"id": 1,
						"ts": 1656356400,
						"period": 60,
						"count": 720
					}
				]
			}`,
			wantRecordsLen:  1,
			wantFirstRecID:  1,
			wantFirstRecTS:  1656356400,
			wantFirstPeriod: 60,
			wantFirstCount:  720,
		},
		{
			name:           "empty records",
			fromTS:         nil,
			result:         `{"records": []}`,
			wantRecordsLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "EMData.GetRecords" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			emdata := NewEMData(client, 0)

			records, err := emdata.GetRecords(context.Background(), tt.fromTS)
			if err != nil {
				t.Errorf("GetRecords() error = %v", err)
				return
			}

			if records == nil {
				t.Fatal("GetRecords() returned nil")
			}

			if len(records.Records) != tt.wantRecordsLen {
				t.Errorf("len(records.Records) = %d, want %d", len(records.Records), tt.wantRecordsLen)
			}

			if tt.wantRecordsLen > 0 {
				firstRec := records.Records[0]
				if firstRec.ID != tt.wantFirstRecID {
					t.Errorf("records.Records[0].ID = %d, want %d", firstRec.ID, tt.wantFirstRecID)
				}
				if firstRec.TS != tt.wantFirstRecTS {
					t.Errorf("records.Records[0].TS = %d, want %d", firstRec.TS, tt.wantFirstRecTS)
				}
				if firstRec.Period != tt.wantFirstPeriod {
					t.Errorf("records.Records[0].Period = %d, want %d", firstRec.Period, tt.wantFirstPeriod)
				}
				if firstRec.Count != tt.wantFirstCount {
					t.Errorf("records.Records[0].Count = %d, want %d", firstRec.Count, tt.wantFirstCount)
				}
			}
		})
	}
}

func TestEMData_GetRecords_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	emdata := NewEMData(client, 0)
	testComponentError(t, "GetRecords", func() error {
		_, err := emdata.GetRecords(context.Background(), nil)
		return err
	})
}

func TestEMData_GetRecords_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	emdata := NewEMData(client, 0)
	testComponentInvalidJSON(t, "GetRecords", func() error {
		_, err := emdata.GetRecords(context.Background(), nil)
		return err
	})
}

func TestEMData_GetData(t *testing.T) {
	tests := []struct {
		name               string
		startTS            *int64
		endTS              *int64
		result             string
		wantDataBlocksLen  int
		wantKeysLen        int
		wantFirstBlockTS   int64
		wantFirstValuesLen int
	}{
		{
			name:    "single block",
			startTS: ptr(int64(1656356400)),
			endTS:   ptr(int64(1656360000)),
			result: `{
				"data": [
					{
						"ts": 1656356400,
						"period": 60,
						"values": [
							{
								"a_voltage": 230.5,
								"a_current": 5.2,
								"a_act_power": 1198.6,
								"a_aprt_power": 1200.0,
								"b_voltage": 229.8,
								"b_current": 4.8,
								"b_act_power": 1103.0,
								"b_aprt_power": 1105.0,
								"c_voltage": 231.2,
								"c_current": 5.5,
								"c_act_power": 1271.6,
								"c_aprt_power": 1273.0,
								"total_current": 15.5,
								"total_act_power": 3573.2,
								"total_aprt_power": 3578.0
							}
						]
					}
				]
			}`,
			wantDataBlocksLen:  1,
			wantKeysLen:        0,
			wantFirstBlockTS:   1656356400,
			wantFirstValuesLen: 1,
		},
		{
			name:    "with optional fields",
			startTS: ptr(int64(1656356400)),
			endTS:   ptr(int64(1656360000)),
			result: `{
				"data": [
					{
						"ts": 1656356400,
						"period": 60,
						"values": [
							{
								"a_voltage": 230.5,
								"a_current": 5.2,
								"a_act_power": 1198.6,
								"a_aprt_power": 1200.0,
								"a_pf": 0.999,
								"a_freq": 50.0,
								"b_voltage": 229.8,
								"b_current": 4.8,
								"b_act_power": 1103.0,
								"b_aprt_power": 1105.0,
								"b_pf": 0.998,
								"b_freq": 50.0,
								"c_voltage": 231.2,
								"c_current": 5.5,
								"c_act_power": 1271.6,
								"c_aprt_power": 1273.0,
								"c_pf": 0.999,
								"c_freq": 50.0,
								"total_current": 15.5,
								"total_act_power": 3573.2,
								"total_aprt_power": 3578.0,
								"n_current": 0.5,
								"total_act_energy": 123456.78,
								"total_act_ret_energy": 9876.54
							}
						]
					}
				],
				"keys": ["a_voltage", "a_current", "a_act_power"]
			}`,
			wantDataBlocksLen:  1,
			wantKeysLen:        3,
			wantFirstBlockTS:   1656356400,
			wantFirstValuesLen: 1,
		},
		{
			name:              "empty data",
			startTS:           nil,
			endTS:             nil,
			result:            `{"data": []}`,
			wantDataBlocksLen: 0,
			wantKeysLen:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "EMData.GetData" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			emdata := NewEMData(client, 0)

			data, err := emdata.GetData(context.Background(), tt.startTS, tt.endTS)
			if err != nil {
				t.Errorf("GetData() error = %v", err)
				return
			}

			if data == nil {
				t.Fatal("GetData() returned nil")
			}

			if len(data.Data) != tt.wantDataBlocksLen {
				t.Errorf("len(data.Data) = %d, want %d", len(data.Data), tt.wantDataBlocksLen)
			}

			if len(data.Keys) != tt.wantKeysLen {
				t.Errorf("len(data.Keys) = %d, want %d", len(data.Keys), tt.wantKeysLen)
			}

			if tt.wantDataBlocksLen > 0 {
				firstBlock := data.Data[0]
				if firstBlock.TS != tt.wantFirstBlockTS {
					t.Errorf("data.Data[0].TS = %d, want %d", firstBlock.TS, tt.wantFirstBlockTS)
				}
				if len(firstBlock.Values) != tt.wantFirstValuesLen {
					t.Errorf("len(data.Data[0].Values) = %d, want %d", len(firstBlock.Values), tt.wantFirstValuesLen)
				}

				if tt.wantFirstValuesLen > 0 {
					values := firstBlock.Values[0]
					// Verify some field values
					if values.AVoltage == 0 {
						t.Error("values.AVoltage should not be 0")
					}
					if values.TotalActivePower == 0 {
						t.Error("values.TotalActivePower should not be 0")
					}
				}
			}
		})
	}
}

func TestEMData_GetData_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	emdata := NewEMData(client, 0)
	testComponentError(t, "GetData", func() error {
		_, err := emdata.GetData(context.Background(), nil, nil)
		return err
	})
}

func TestEMData_GetData_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	emdata := NewEMData(client, 0)
	testComponentInvalidJSON(t, "GetData", func() error {
		_, err := emdata.GetData(context.Background(), nil, nil)
		return err
	})
}

func TestEMData_DeleteAllData(t *testing.T) {
	methodCalled := false
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
			method := req.GetMethod()
			if method != "EMData.DeleteAllData" {
				t.Errorf("unexpected method call: %s", method)
			}
			methodCalled = true
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	emdata := NewEMData(client, 0)

	err := emdata.DeleteAllData(context.Background())
	if err != nil {
		t.Errorf("DeleteAllData() error = %v", err)
	}

	if !methodCalled {
		t.Error("EMData.DeleteAllData was not called")
	}
}

func TestEMData_DeleteAllData_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	emdata := NewEMData(client, 0)
	testComponentError(t, "DeleteAllData", func() error {
		return emdata.DeleteAllData(context.Background())
	})
}

func TestEMData_GetDataCSVURL(t *testing.T) {
	tests := []struct {
		name       string
		deviceAddr string
		startTS    *int64
		endTS      *int64
		addKeys    bool
		wantURL    string
	}{
		{
			name:       "all parameters",
			deviceAddr: "192.168.1.100",
			startTS:    ptr(int64(1656356400)),
			endTS:      ptr(int64(1656442800)),
			addKeys:    true,
			wantURL:    "http://192.168.1.100/emdata/0/data.csv?ts=1656356400&end_ts=1656442800&add_keys=true",
		},
		{
			name:       "without keys",
			deviceAddr: "192.168.1.100",
			startTS:    ptr(int64(1656356400)),
			endTS:      ptr(int64(1656442800)),
			addKeys:    false,
			wantURL:    "http://192.168.1.100/emdata/0/data.csv?ts=1656356400&end_ts=1656442800",
		},
		{
			name:       "only start timestamp",
			deviceAddr: "10.0.0.50",
			startTS:    ptr(int64(1656356400)),
			endTS:      nil,
			addKeys:    true,
			wantURL:    "http://10.0.0.50/emdata/0/data.csv?ts=1656356400&add_keys=true",
		},
		{
			name:       "only end timestamp",
			deviceAddr: "10.0.0.50",
			startTS:    nil,
			endTS:      ptr(int64(1656442800)),
			addKeys:    false,
			wantURL:    "http://10.0.0.50/emdata/0/data.csv?end_ts=1656442800",
		},
		{
			name:       "no optional params",
			deviceAddr: "192.168.1.100",
			startTS:    nil,
			endTS:      nil,
			addKeys:    false,
			wantURL:    "http://192.168.1.100/emdata/0/data.csv?",
		},
		{
			name:       "only add keys",
			deviceAddr: "shelly-em.local",
			startTS:    nil,
			endTS:      nil,
			addKeys:    true,
			wantURL:    "http://shelly-em.local/emdata/0/data.csv?add_keys=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{}
			client := rpc.NewClient(tr)
			emdata := NewEMData(client, 0)

			url := emdata.GetDataCSVURL(tt.deviceAddr, tt.startTS, tt.endTS, tt.addKeys)
			if url != tt.wantURL {
				t.Errorf("GetDataCSVURL() = %q, want %q", url, tt.wantURL)
			}
		})
	}
}

func TestEMData_GetDataCSVURL_DifferentID(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)
	emdata := NewEMData(client, 1) // ID = 1 instead of 0

	url := emdata.GetDataCSVURL("192.168.1.100", nil, nil, false)
	expectedURL := "http://192.168.1.100/emdata/1/data.csv?"

	if url != expectedURL {
		t.Errorf("GetDataCSVURL() for ID=1 = %q, want %q", url, expectedURL)
	}
}
