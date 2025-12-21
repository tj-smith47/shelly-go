package components

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tj-smith47/shelly-go/rpc"
	"github.com/tj-smith47/shelly-go/transport"
)

func TestNewEM1Data(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)

	em1data := NewEM1Data(client, 0)

	if em1data == nil {
		t.Fatal("NewEM1Data returned nil")
	}

	if em1data.Type() != "em1data" {
		t.Errorf("Type() = %q, want %q", em1data.Type(), "em1data")
	}

	if em1data.ID() != 0 {
		t.Errorf("ID() = %d, want %d", em1data.ID(), 0)
	}

	if em1data.Key() != "em1data:0" {
		t.Errorf("Key() = %q, want %q", em1data.Key(), "em1data:0")
	}
}

func TestEM1Data_GetConfig(t *testing.T) {
	tests := []struct {
		name                string
		result              string
		wantID              int
		wantName            *string
		wantDataPeriod      *int
		wantDataStorageDays *int
	}{
		{
			name: "full config",
			result: `{
				"id": 0,
				"name": "Main Meter Data",
				"data_period": 60,
				"data_storage_days": 30
			}`,
			wantID:              0,
			wantName:            ptr("Main Meter Data"),
			wantDataPeriod:      ptr(60),
			wantDataStorageDays: ptr(30),
		},
		{
			name: "minimal config",
			result: `{
				"id": 0
			}`,
			wantID:              0,
			wantName:            nil,
			wantDataPeriod:      nil,
			wantDataStorageDays: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "EM1Data.GetConfig" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			em1data := NewEM1Data(client, 0)

			config, err := em1data.GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() error = %v", err)
				return
			}

			if config == nil {
				t.Fatal("GetConfig() returned nil config")
			}

			if config.ID != tt.wantID {
				t.Errorf("config.ID = %d, want %d", config.ID, tt.wantID)
			}

			if tt.wantName != nil {
				if config.Name == nil {
					t.Errorf("config.Name = nil, want %q", *tt.wantName)
				} else if *config.Name != *tt.wantName {
					t.Errorf("config.Name = %q, want %q", *config.Name, *tt.wantName)
				}
			} else if config.Name != nil {
				t.Errorf("config.Name = %q, want nil", *config.Name)
			}

			if tt.wantDataPeriod != nil {
				if config.DataPeriod == nil {
					t.Errorf("config.DataPeriod = nil, want %d", *tt.wantDataPeriod)
				} else if *config.DataPeriod != *tt.wantDataPeriod {
					t.Errorf("config.DataPeriod = %d, want %d", *config.DataPeriod, *tt.wantDataPeriod)
				}
			} else if config.DataPeriod != nil {
				t.Errorf("config.DataPeriod = %d, want nil", *config.DataPeriod)
			}

			if tt.wantDataStorageDays != nil {
				if config.DataStorageDays == nil {
					t.Errorf("config.DataStorageDays = nil, want %d", *tt.wantDataStorageDays)
				} else if *config.DataStorageDays != *tt.wantDataStorageDays {
					t.Errorf("config.DataStorageDays = %d, want %d", *config.DataStorageDays, *tt.wantDataStorageDays)
				}
			} else if config.DataStorageDays != nil {
				t.Errorf("config.DataStorageDays = %d, want nil", *config.DataStorageDays)
			}
		})
	}
}

func TestEM1Data_GetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em1data := NewEM1Data(client, 0)
	testComponentError(t, "GetConfig", func() error {
		_, err := em1data.GetConfig(context.Background())
		return err
	})
}

func TestEM1Data_GetConfig_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	em1data := NewEM1Data(client, 0)
	testComponentInvalidJSON(t, "GetConfig", func() error {
		_, err := em1data.GetConfig(context.Background())
		return err
	})
}

func TestEM1Data_SetConfig(t *testing.T) {
	config := &EM1DataConfig{
		ID:              0,
		Name:            ptr("Industrial Meter"),
		DataPeriod:      ptr(60),
		DataStorageDays: ptr(30),
	}

	methodCalled := false
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "EM1Data.SetConfig" {
				t.Errorf("unexpected method call: %s", method)
			}
			methodCalled = true
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	em1data := NewEM1Data(client, 0)

	err := em1data.SetConfig(context.Background(), config)
	if err != nil {
		t.Errorf("SetConfig() error = %v", err)
	}

	if !methodCalled {
		t.Error("EM1Data.SetConfig was not called")
	}
}

func TestEM1Data_SetConfig_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em1data := NewEM1Data(client, 0)
	testComponentError(t, "SetConfig", func() error {
		return em1data.SetConfig(context.Background(), &EM1DataConfig{})
	})
}

func TestEM1Data_GetStatus(t *testing.T) {
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
				"last_record_id": 5000,
				"available_records": 720
			}`,
			wantID:               0,
			wantLastRecordID:     ptr(5000),
			wantAvailableRecords: ptr(720),
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
				"errors": ["storage_error"]
			}`,
			wantID:               0,
			wantLastRecordID:     ptr(100),
			wantAvailableRecords: ptr(50),
			wantErrorsLen:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{
				callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
					if method != "EM1Data.GetStatus" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			em1data := NewEM1Data(client, 0)

			status, err := em1data.GetStatus(context.Background())
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

func TestEM1Data_GetStatus_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em1data := NewEM1Data(client, 0)
	testComponentError(t, "GetStatus", func() error {
		_, err := em1data.GetStatus(context.Background())
		return err
	})
}

func TestEM1Data_GetStatus_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	em1data := NewEM1Data(client, 0)
	testComponentInvalidJSON(t, "GetStatus", func() error {
		_, err := em1data.GetStatus(context.Background())
		return err
	})
}

func TestEM1Data_GetRecords(t *testing.T) {
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
						"count": 720
					},
					{
						"id": 2,
						"ts": 1656399600,
						"period": 60,
						"count": 720
					}
				]
			}`,
			wantRecordsLen:  2,
			wantFirstRecID:  1,
			wantFirstRecTS:  1656356400,
			wantFirstPeriod: 60,
			wantFirstCount:  720,
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
						"count": 360
					}
				]
			}`,
			wantRecordsLen:  1,
			wantFirstRecID:  1,
			wantFirstRecTS:  1656356400,
			wantFirstPeriod: 60,
			wantFirstCount:  360,
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
					if method != "EM1Data.GetRecords" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			em1data := NewEM1Data(client, 0)

			records, err := em1data.GetRecords(context.Background(), tt.fromTS)
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

func TestEM1Data_GetRecords_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em1data := NewEM1Data(client, 0)
	testComponentError(t, "GetRecords", func() error {
		_, err := em1data.GetRecords(context.Background(), nil)
		return err
	})
}

func TestEM1Data_GetRecords_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	em1data := NewEM1Data(client, 0)
	testComponentInvalidJSON(t, "GetRecords", func() error {
		_, err := em1data.GetRecords(context.Background(), nil)
		return err
	})
}

func TestEM1Data_GetData(t *testing.T) {
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
								"voltage": 230.5,
								"current": 5.2,
								"act_power": 1198.6,
								"aprt_power": 1200.0
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
								"voltage": 230.5,
								"current": 5.2,
								"act_power": 1198.6,
								"aprt_power": 1200.0,
								"pf": 0.999,
								"freq": 50.0,
								"act_energy": 12345.67,
								"act_ret_energy": 987.65
							}
						]
					}
				],
				"keys": ["voltage", "current", "act_power"]
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
					if method != "EM1Data.GetData" {
						t.Errorf("unexpected method call: %s", method)
					}
					return jsonrpcResponse(tt.result)
				},
			}
			client := rpc.NewClient(tr)
			em1data := NewEM1Data(client, 0)

			data, err := em1data.GetData(context.Background(), tt.startTS, tt.endTS)
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
					if values.Voltage == 0 {
						t.Error("values.Voltage should not be 0")
					}
					if values.ActivePower == 0 {
						t.Error("values.ActivePower should not be 0")
					}
				}
			}
		})
	}
}

func TestEM1Data_GetData_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em1data := NewEM1Data(client, 0)
	testComponentError(t, "GetData", func() error {
		_, err := em1data.GetData(context.Background(), nil, nil)
		return err
	})
}

func TestEM1Data_GetData_InvalidJSON(t *testing.T) {
	client := rpc.NewClient(invalidJSONTransport())
	em1data := NewEM1Data(client, 0)
	testComponentInvalidJSON(t, "GetData", func() error {
		_, err := em1data.GetData(context.Background(), nil, nil)
		return err
	})
}

func TestEM1Data_DeleteAllData(t *testing.T) {
	methodCalled := false
	tr := &mockTransport{
		callFunc: func(ctx context.Context, req transport.RPCRequest) (json.RawMessage, error) {
					method := req.GetMethod()
			if method != "EM1Data.DeleteAllData" {
				t.Errorf("unexpected method call: %s", method)
			}
			methodCalled = true
			return jsonrpcResponse(`null`)
		},
	}
	client := rpc.NewClient(tr)
	em1data := NewEM1Data(client, 0)

	err := em1data.DeleteAllData(context.Background())
	if err != nil {
		t.Errorf("DeleteAllData() error = %v", err)
	}

	if !methodCalled {
		t.Error("EM1Data.DeleteAllData was not called")
	}
}

func TestEM1Data_DeleteAllData_Error(t *testing.T) {
	client := rpc.NewClient(errorTransport(errors.New("device unreachable")))
	em1data := NewEM1Data(client, 0)
	testComponentError(t, "DeleteAllData", func() error {
		return em1data.DeleteAllData(context.Background())
	})
}

func TestEM1Data_GetDataCSVURL(t *testing.T) {
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
			wantURL:    "http://192.168.1.100/em1data/0/data.csv?ts=1656356400&end_ts=1656442800&add_keys=true",
		},
		{
			name:       "without keys",
			deviceAddr: "192.168.1.100",
			startTS:    ptr(int64(1656356400)),
			endTS:      ptr(int64(1656442800)),
			addKeys:    false,
			wantURL:    "http://192.168.1.100/em1data/0/data.csv?ts=1656356400&end_ts=1656442800",
		},
		{
			name:       "only start timestamp",
			deviceAddr: "10.0.0.50",
			startTS:    ptr(int64(1656356400)),
			endTS:      nil,
			addKeys:    true,
			wantURL:    "http://10.0.0.50/em1data/0/data.csv?ts=1656356400&add_keys=true",
		},
		{
			name:       "only end timestamp",
			deviceAddr: "10.0.0.50",
			startTS:    nil,
			endTS:      ptr(int64(1656442800)),
			addKeys:    false,
			wantURL:    "http://10.0.0.50/em1data/0/data.csv?end_ts=1656442800",
		},
		{
			name:       "no optional params",
			deviceAddr: "192.168.1.100",
			startTS:    nil,
			endTS:      nil,
			addKeys:    false,
			wantURL:    "http://192.168.1.100/em1data/0/data.csv?",
		},
		{
			name:       "only add keys",
			deviceAddr: "shelly-em.local",
			startTS:    nil,
			endTS:      nil,
			addKeys:    true,
			wantURL:    "http://shelly-em.local/em1data/0/data.csv?add_keys=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &mockTransport{}
			client := rpc.NewClient(tr)
			em1data := NewEM1Data(client, 0)

			url := em1data.GetDataCSVURL(tt.deviceAddr, tt.startTS, tt.endTS, tt.addKeys)
			if url != tt.wantURL {
				t.Errorf("GetDataCSVURL() = %q, want %q", url, tt.wantURL)
			}
		})
	}
}

func TestEM1Data_GetDataCSVURL_DifferentID(t *testing.T) {
	tr := &mockTransport{}
	client := rpc.NewClient(tr)
	em1data := NewEM1Data(client, 2) // ID = 2 instead of 0

	url := em1data.GetDataCSVURL("192.168.1.100", nil, nil, false)
	expectedURL := "http://192.168.1.100/em1data/2/data.csv?"

	if url != expectedURL {
		t.Errorf("GetDataCSVURL() for ID=2 = %q, want %q", url, expectedURL)
	}
}
