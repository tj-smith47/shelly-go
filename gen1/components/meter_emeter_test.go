package components

import (
	"context"
	"errors"
	"testing"
)

// ==================== Meter Tests ====================

// TestNewMeter tests meter creation.
func TestNewMeter(t *testing.T) {
	mt := newMockTransport()
	meter := NewMeter(mt, 0)

	if meter == nil {
		t.Fatal("expected meter to be created")
	}

	if meter.ID() != 0 {
		t.Errorf("expected ID 0, got %d", meter.ID())
	}
}

// TestMeterGetStatus tests meter status retrieval.
func TestMeterGetStatus(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/meter/0", &MeterStatus{
		Power:     150.5,
		IsValid:   true,
		Timestamp: 1699300000,
		Counters:  []float64{10.5, 20.3, 15.7},
		Total:     100000,
	})

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	status, err := meter.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Power != 150.5 {
		t.Errorf("expected power 150.5, got %f", status.Power)
	}

	if !status.IsValid {
		t.Error("expected valid reading")
	}

	if status.Total != 100000 {
		t.Errorf("expected total 100000, got %d", status.Total)
	}

	if len(status.Counters) != 3 {
		t.Errorf("expected 3 counters, got %d", len(status.Counters))
	}
}

// TestMeterGetPower tests GetPower method.
func TestMeterGetPower(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/meter/0", &MeterStatus{Power: 250.0})

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	power, err := meter.GetPower(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if power != 250.0 {
		t.Errorf("expected power 250, got %f", power)
	}
}

// TestMeterGetTotal tests GetTotal method.
func TestMeterGetTotal(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/meter/0", &MeterStatus{Total: 50000})

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	total, err := meter.GetTotal(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 50000 {
		t.Errorf("expected total 50000, got %d", total)
	}
}

// TestMeterGetTotalKWh tests GetTotalKWh method.
func TestMeterGetTotalKWh(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/meter/0", &MeterStatus{Total: 60000}) // 60000 watt-minutes = 1 kWh

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	kwh, err := meter.GetTotalKWh(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 60000 watt-minutes / (60 * 1000) = 1 kWh
	if kwh != 1.0 {
		t.Errorf("expected 1 kWh, got %f", kwh)
	}
}

// TestMeterGetCounters tests GetCounters method.
func TestMeterGetCounters(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/meter/0", &MeterStatus{Counters: []float64{1.5, 2.5, 3.5}})

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	counters, err := meter.GetCounters(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(counters) != 3 {
		t.Errorf("expected 3 counters, got %d", len(counters))
	}

	if counters[0] != 1.5 {
		t.Errorf("expected counter 0 = 1.5, got %f", counters[0])
	}
}

// TestMeterResetCounters tests counter reset.
func TestMeterResetCounters(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/meter/0?reset_totals=true", map[string]bool{"ok": true})

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	err := meter.ResetCounters(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestMeterSetPowerLimit tests power limit setting.
func TestMeterSetPowerLimit(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/relay/0?max_power=2000", map[string]bool{"ok": true})

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	err := meter.SetPowerLimit(ctx, 2000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestMeterErrors tests error handling.
func TestMeterErrors(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/meter/0", errors.New("sensor error"))

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	_, err := meter.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// ==================== EMeter Tests ====================

// TestNewEMeter tests emeter creation.
func TestNewEMeter(t *testing.T) {
	mt := newMockTransport()
	emeter := NewEMeter(mt, 0)

	if emeter == nil {
		t.Fatal("expected emeter to be created")
	}

	if emeter.ID() != 0 {
		t.Errorf("expected ID 0, got %d", emeter.ID())
	}
}

// TestEMeterGetStatus tests emeter status retrieval.
func TestEMeterGetStatus(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0", &EMeterStatus{
		Power:         -500.0,
		Reactive:      100.0,
		PF:            0.95,
		Current:       2.5,
		Voltage:       230.0,
		IsValid:       true,
		Total:         50000.0,
		TotalReturned: 10000.0,
	})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	status, err := emeter.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Power != -500.0 {
		t.Errorf("expected power -500, got %f", status.Power)
	}

	if status.Voltage != 230.0 {
		t.Errorf("expected voltage 230, got %f", status.Voltage)
	}

	if status.Current != 2.5 {
		t.Errorf("expected current 2.5, got %f", status.Current)
	}

	if status.PF != 0.95 {
		t.Errorf("expected PF 0.95, got %f", status.PF)
	}
}

// TestEMeterGetPower tests GetPower method.
func TestEMeterGetPower(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0", &EMeterStatus{Power: 1500.0})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	power, err := emeter.GetPower(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if power != 1500.0 {
		t.Errorf("expected power 1500, got %f", power)
	}
}

// TestEMeterGetVoltage tests GetVoltage method.
func TestEMeterGetVoltage(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0", &EMeterStatus{Voltage: 240.5})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	voltage, err := emeter.GetVoltage(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if voltage != 240.5 {
		t.Errorf("expected voltage 240.5, got %f", voltage)
	}
}

// TestEMeterGetCurrent tests GetCurrent method.
func TestEMeterGetCurrent(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0", &EMeterStatus{Current: 10.5})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	current, err := emeter.GetCurrent(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if current != 10.5 {
		t.Errorf("expected current 10.5, got %f", current)
	}
}

// TestEMeterGetPowerFactor tests GetPowerFactor method.
func TestEMeterGetPowerFactor(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0", &EMeterStatus{PF: 0.85})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	pf, err := emeter.GetPowerFactor(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pf != 0.85 {
		t.Errorf("expected PF 0.85, got %f", pf)
	}
}

// TestEMeterGetTotal tests GetTotal method.
func TestEMeterGetTotal(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0", &EMeterStatus{Total: 75000.0})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	total, err := emeter.GetTotal(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 75000.0 {
		t.Errorf("expected total 75000, got %f", total)
	}
}

// TestEMeterGetTotalKWh tests GetTotalKWh method.
func TestEMeterGetTotalKWh(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0", &EMeterStatus{Total: 10000.0}) // 10000 Wh = 10 kWh

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	kwh, err := emeter.GetTotalKWh(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if kwh != 10.0 {
		t.Errorf("expected 10 kWh, got %f", kwh)
	}
}

// TestEMeterGetTotalReturned tests GetTotalReturned method.
func TestEMeterGetTotalReturned(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0", &EMeterStatus{TotalReturned: 5000.0})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	returned, err := emeter.GetTotalReturned(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if returned != 5000.0 {
		t.Errorf("expected returned 5000, got %f", returned)
	}
}

// TestEMeterGetTotalReturnedKWh tests GetTotalReturnedKWh method.
func TestEMeterGetTotalReturnedKWh(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0", &EMeterStatus{TotalReturned: 5000.0}) // 5000 Wh = 5 kWh

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	kwh, err := emeter.GetTotalReturnedKWh(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if kwh != 5.0 {
		t.Errorf("expected 5 kWh, got %f", kwh)
	}
}

// TestEMeterGetNetEnergy tests GetNetEnergy method.
func TestEMeterGetNetEnergy(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0", &EMeterStatus{Total: 10000.0, TotalReturned: 3000.0})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	net, err := emeter.GetNetEnergy(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 10000 - 3000 = 7000
	if net != 7000.0 {
		t.Errorf("expected net 7000, got %f", net)
	}
}

// TestEMeterResetCounters tests counter reset.
func TestEMeterResetCounters(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0?reset_totals=true", map[string]bool{"ok": true})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	err := emeter.ResetCounters(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestEMeterGetConfig tests config retrieval.
func TestEMeterGetConfig(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/emeter/0", &EMeterConfig{CTType: 1})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	config, err := emeter.GetConfig(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.CTType != 1 {
		t.Errorf("expected CT type 1, got %d", config.CTType)
	}
}

// TestEMeterSetCTType tests CT type setting.
func TestEMeterSetCTType(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/settings/emeter/0?cttype=1", map[string]bool{"ok": true})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	err := emeter.SetCTType(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestEMeterGetData tests historical data retrieval.
func TestEMeterGetData(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0/em_data", []EMeterData{
		{Timestamp: 1699300000, Total: 1000, TotalReturned: 100},
		{Timestamp: 1699300060, Total: 1100, TotalReturned: 110},
	})

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	data, err := emeter.GetData(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) != 2 {
		t.Errorf("expected 2 data points, got %d", len(data))
	}
}

// TestEMeterErrors tests error handling.
func TestEMeterErrors(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0", errors.New("CT error"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetStatus(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterMultipleChannels tests multiple emeter channels (3EM).
func TestEMeterMultipleChannels(t *testing.T) {
	mt := newMockTransport()
	mt.SetResponse("/emeter/0", &EMeterStatus{Power: 500, Voltage: 230})
	mt.SetResponse("/emeter/1", &EMeterStatus{Power: 600, Voltage: 231})
	mt.SetResponse("/emeter/2", &EMeterStatus{Power: 700, Voltage: 229})

	ctx := context.Background()

	// Test all three phases
	for i := 0; i < 3; i++ {
		emeter := NewEMeter(mt, i)
		if emeter.ID() != i {
			t.Errorf("expected ID %d, got %d", i, emeter.ID())
		}

		status, err := emeter.GetStatus(ctx)
		if err != nil {
			t.Fatalf("unexpected error for emeter %d: %v", i, err)
		}

		expectedPower := float64((i+1)*100 + 400)
		if status.Power != expectedPower {
			t.Errorf("emeter %d: expected power %f, got %f", i, expectedPower, status.Power)
		}
	}
}

// ==================== Additional Error Tests ====================

// TestMeterGetPowerError tests GetPower error handling.
func TestMeterGetPowerError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/meter/0", errors.New("offline"))

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	_, err := meter.GetPower(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestMeterGetTotalError tests GetTotal error handling.
func TestMeterGetTotalError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/meter/0", errors.New("offline"))

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	_, err := meter.GetTotal(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestMeterGetTotalKWhError tests GetTotalKWh error handling.
func TestMeterGetTotalKWhError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/meter/0", errors.New("offline"))

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	_, err := meter.GetTotalKWh(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestMeterGetCountersError tests GetCounters error handling.
func TestMeterGetCountersError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/meter/0", errors.New("offline"))

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	_, err := meter.GetCounters(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestMeterResetCountersError tests ResetCounters error handling.
func TestMeterResetCountersError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/meter/0?reset_totals=true", errors.New("failed"))

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	err := meter.ResetCounters(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestMeterSetPowerLimitError tests SetPowerLimit error handling.
func TestMeterSetPowerLimitError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/relay/0?max_power=2000", errors.New("failed"))

	meter := NewMeter(mt, 0)
	ctx := context.Background()

	err := meter.SetPowerLimit(ctx, 2000)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterGetPowerError tests GetPower error handling.
func TestEMeterGetPowerError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0", errors.New("offline"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetPower(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterGetVoltageError tests GetVoltage error handling.
func TestEMeterGetVoltageError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0", errors.New("offline"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetVoltage(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterGetCurrentError tests GetCurrent error handling.
func TestEMeterGetCurrentError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0", errors.New("offline"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetCurrent(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterGetPowerFactorError tests GetPowerFactor error handling.
func TestEMeterGetPowerFactorError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0", errors.New("offline"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetPowerFactor(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterGetTotalError tests GetTotal error handling.
func TestEMeterGetTotalError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0", errors.New("offline"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetTotal(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterGetTotalKWhError tests GetTotalKWh error handling.
func TestEMeterGetTotalKWhError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0", errors.New("offline"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetTotalKWh(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterGetTotalReturnedError tests GetTotalReturned error handling.
func TestEMeterGetTotalReturnedError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0", errors.New("offline"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetTotalReturned(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterGetTotalReturnedKWhError tests GetTotalReturnedKWh error handling.
func TestEMeterGetTotalReturnedKWhError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0", errors.New("offline"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetTotalReturnedKWh(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterGetNetEnergyError tests GetNetEnergy error handling.
func TestEMeterGetNetEnergyError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0", errors.New("offline"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetNetEnergy(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterResetCountersError tests ResetCounters error handling.
func TestEMeterResetCountersError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0?reset_totals=true", errors.New("failed"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	err := emeter.ResetCounters(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterGetConfigError tests GetConfig error handling.
func TestEMeterGetConfigError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/emeter/0", errors.New("unauthorized"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterSetCTTypeError tests SetCTType error handling.
func TestEMeterSetCTTypeError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/settings/emeter/0?cttype=1", errors.New("invalid"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	err := emeter.SetCTType(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEMeterGetDataError tests GetData error handling.
func TestEMeterGetDataError(t *testing.T) {
	mt := newMockTransport()
	mt.SetError("/emeter/0/em_data", errors.New("no data"))

	emeter := NewEMeter(mt, 0)
	ctx := context.Background()

	_, err := emeter.GetData(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}
