package components

import (
	"testing"
)

// mockLightConfig implements lightConfigParams for testing.
type mockLightConfig struct {
	name         string
	defaultState string
	autoOn       float64
	autoOff      float64
	schedule     bool
}

func (m *mockLightConfig) getName() string       { return m.name }
func (m *mockLightConfig) getDefaultState() string { return m.defaultState }
func (m *mockLightConfig) getAutoOn() float64    { return m.autoOn }
func (m *mockLightConfig) getAutoOff() float64   { return m.autoOff }
func (m *mockLightConfig) getSchedule() bool     { return m.schedule }

func TestBuildLightConfigQuery_Empty(t *testing.T) {
	config := &mockLightConfig{}
	result := buildLightConfigQuery(config)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestBuildLightConfigQuery_Name(t *testing.T) {
	config := &mockLightConfig{name: "TestLight"}
	result := buildLightConfigQuery(config)
	if result != "name=TestLight" {
		t.Errorf("expected 'name=TestLight', got %q", result)
	}
}

func TestBuildLightConfigQuery_DefaultState(t *testing.T) {
	config := &mockLightConfig{defaultState: "on"}
	result := buildLightConfigQuery(config)
	if result != "default_state=on" {
		t.Errorf("expected 'default_state=on', got %q", result)
	}
}

func TestBuildLightConfigQuery_AutoOn(t *testing.T) {
	config := &mockLightConfig{autoOn: 30.5}
	result := buildLightConfigQuery(config)
	if result != "auto_on=30.5" {
		t.Errorf("expected 'auto_on=30.5', got %q", result)
	}
}

func TestBuildLightConfigQuery_AutoOff(t *testing.T) {
	config := &mockLightConfig{autoOff: 60.0}
	result := buildLightConfigQuery(config)
	if result != "auto_off=60" {
		t.Errorf("expected 'auto_off=60', got %q", result)
	}
}

func TestBuildLightConfigQuery_Schedule(t *testing.T) {
	config := &mockLightConfig{schedule: true}
	result := buildLightConfigQuery(config)
	if result != "schedule=true" {
		t.Errorf("expected 'schedule=true', got %q", result)
	}
}

func TestBuildLightConfigQuery_AllFields(t *testing.T) {
	config := &mockLightConfig{
		name:         "Kitchen",
		defaultState: "off",
		autoOn:       10.0,
		autoOff:      20.0,
		schedule:     true,
	}
	result := buildLightConfigQuery(config)
	expected := "name=Kitchen&default_state=off&auto_on=10&auto_off=20&schedule=true"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildLightConfigQuery_PartialFields(t *testing.T) {
	config := &mockLightConfig{
		name:     "Bedroom",
		autoOff:  45.0,
		schedule: true,
	}
	result := buildLightConfigQuery(config)
	expected := "name=Bedroom&auto_off=45&schedule=true"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
