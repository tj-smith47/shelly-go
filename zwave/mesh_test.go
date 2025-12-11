package zwave

import (
	"testing"
)

func TestCommonAssociationGroups(t *testing.T) {
	groups := CommonAssociationGroups()

	if len(groups) == 0 {
		t.Fatal("expected non-empty association groups")
	}

	// Check that Lifeline group (1) exists
	var lifelineFound bool
	for _, g := range groups {
		if g.ID == 1 && g.Name == "Lifeline" {
			lifelineFound = true
			if g.MaxNodes < 1 {
				t.Errorf("Lifeline MaxNodes = %d, want >= 1", g.MaxNodes)
			}
			if g.Profile == "" {
				t.Error("Lifeline Profile is empty")
			}
			break
		}
	}
	if !lifelineFound {
		t.Error("Lifeline group (ID 1) not found")
	}

	// Check that Basic Set group (2) exists
	var basicSetFound bool
	for _, g := range groups {
		if g.ID == 2 && g.Name == "Basic Set" {
			basicSetFound = true
			if g.MaxNodes < 1 {
				t.Errorf("Basic Set MaxNodes = %d, want >= 1", g.MaxNodes)
			}
			break
		}
	}
	if !basicSetFound {
		t.Error("Basic Set group (ID 2) not found")
	}
}

func TestCommonAssociationGroups_UniqueIDs(t *testing.T) {
	groups := CommonAssociationGroups()

	ids := make(map[int]bool)
	for _, g := range groups {
		if ids[g.ID] {
			t.Errorf("duplicate group ID: %d", g.ID)
		}
		ids[g.ID] = true
	}
}

func TestCommonConfigParameters(t *testing.T) {
	params := CommonConfigParameters()

	if len(params) == 0 {
		t.Fatal("expected non-empty config parameters")
	}

	// Check each parameter has required fields
	for _, p := range params {
		if p.Number < 1 {
			t.Errorf("parameter number = %d, want >= 1", p.Number)
		}
		if p.Name == "" {
			t.Errorf("parameter %d has empty name", p.Number)
		}
		if p.Size < 1 || p.Size > 4 {
			t.Errorf("parameter %d Size = %d, want 1-4", p.Number, p.Size)
		}
		if p.MinValue > p.MaxValue {
			t.Errorf("parameter %d MinValue (%d) > MaxValue (%d)", p.Number, p.MinValue, p.MaxValue)
		}
		if p.DefaultValue < p.MinValue || p.DefaultValue > p.MaxValue {
			t.Errorf("parameter %d DefaultValue (%d) out of range [%d, %d]",
				p.Number, p.DefaultValue, p.MinValue, p.MaxValue)
		}
	}
}

func TestCommonConfigParameters_UniqueNumbers(t *testing.T) {
	params := CommonConfigParameters()

	numbers := make(map[int]bool)
	for _, p := range params {
		if numbers[p.Number] {
			t.Errorf("duplicate parameter number: %d", p.Number)
		}
		numbers[p.Number] = true
	}
}

func TestNodeInfo_Fields(t *testing.T) {
	node := NodeInfo{
		NodeID:      5,
		HomeID:      0xA1B2C3D4,
		Model:       "SNSW-001P16ZW",
		Name:        "Living Room Switch",
		IsListening: true,
		IsRouting:   true,
		Security:    SecurityS2Authenticated,
		Topology:    TopologyMesh,
	}

	if node.NodeID != 5 {
		t.Errorf("NodeID = %d, want 5", node.NodeID)
	}
	if node.HomeID != 0xA1B2C3D4 {
		t.Errorf("HomeID = %x, want A1B2C3D4", node.HomeID)
	}
	if node.Model != "SNSW-001P16ZW" {
		t.Errorf("Model = %q, want %q", node.Model, "SNSW-001P16ZW")
	}
	if !node.IsListening {
		t.Error("IsListening = false, want true")
	}
	if !node.IsRouting {
		t.Error("IsRouting = false, want true")
	}
	if node.Security != SecurityS2Authenticated {
		t.Errorf("Security = %q, want %q", node.Security, SecurityS2Authenticated)
	}
	if node.Topology != TopologyMesh {
		t.Errorf("Topology = %q, want %q", node.Topology, TopologyMesh)
	}
}

func TestNetworkInfo_Fields(t *testing.T) {
	network := NetworkInfo{
		HomeID:           0xA1B2C3D4,
		ControllerNodeID: 1,
		NodeCount:        3,
		Nodes: []NodeInfo{
			{NodeID: 1, Name: "Controller"},
			{NodeID: 5, Name: "Switch 1"},
			{NodeID: 6, Name: "Switch 2"},
		},
	}

	if network.HomeID != 0xA1B2C3D4 {
		t.Errorf("HomeID = %x, want A1B2C3D4", network.HomeID)
	}
	if network.ControllerNodeID != 1 {
		t.Errorf("ControllerNodeID = %d, want 1", network.ControllerNodeID)
	}
	if network.NodeCount != 3 {
		t.Errorf("NodeCount = %d, want 3", network.NodeCount)
	}
	if len(network.Nodes) != 3 {
		t.Errorf("len(Nodes) = %d, want 3", len(network.Nodes))
	}
}

func TestAssociationGroup_Fields(t *testing.T) {
	group := AssociationGroup{
		ID:       2,
		Name:     "Basic Set",
		MaxNodes: 5,
		NodeIDs:  []int{3, 7, 12},
		Profile:  "Sends Basic Set commands when switch state changes",
	}

	if group.ID != 2 {
		t.Errorf("ID = %d, want 2", group.ID)
	}
	if group.Name != "Basic Set" {
		t.Errorf("Name = %q, want %q", group.Name, "Basic Set")
	}
	if group.MaxNodes != 5 {
		t.Errorf("MaxNodes = %d, want 5", group.MaxNodes)
	}
	if len(group.NodeIDs) != 3 {
		t.Errorf("len(NodeIDs) = %d, want 3", len(group.NodeIDs))
	}
	if group.Profile == "" {
		t.Error("Profile is empty")
	}
}

func TestConfigurationParameter_Fields(t *testing.T) {
	currentVal := 50
	param := ConfigurationParameter{
		Number:       36,
		Name:         "Power Report Threshold",
		Description:  "Power change percentage to trigger report",
		Size:         1,
		DefaultValue: 10,
		MinValue:     0,
		MaxValue:     100,
		CurrentValue: &currentVal,
	}

	if param.Number != 36 {
		t.Errorf("Number = %d, want 36", param.Number)
	}
	if param.Size != 1 {
		t.Errorf("Size = %d, want 1", param.Size)
	}
	if param.DefaultValue != 10 {
		t.Errorf("DefaultValue = %d, want 10", param.DefaultValue)
	}
	if param.CurrentValue == nil || *param.CurrentValue != 50 {
		t.Errorf("CurrentValue = %v, want 50", param.CurrentValue)
	}
}
