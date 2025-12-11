package zwave

// NodeInfo contains information about a Z-Wave node in the network.
type NodeInfo struct {
	Model       string
	Name        string
	Security    SecurityLevel
	Topology    NetworkTopology
	NodeID      int
	HomeID      uint32
	IsListening bool
	IsRouting   bool
}

// NetworkInfo contains information about a Z-Wave network.
type NetworkInfo struct {
	Nodes            []NodeInfo
	ControllerNodeID int
	NodeCount        int
	HomeID           uint32
}

// AssociationGroup represents a Z-Wave association group.
//
// Association groups allow devices to directly control other devices
// without going through the gateway. For example, a switch can be
// associated with a light to control it directly.
type AssociationGroup struct {
	Name     string
	Profile  string
	NodeIDs  []int
	ID       int
	MaxNodes int
}

// CommonAssociationGroups returns the standard association groups for Wave devices.
//
// Most Shelly Wave devices support these common groups:
//   - Group 1 (Lifeline): Reports to controller
//   - Group 2 (Basic Set): On/off control to associated devices
//
// Example:
//
//	groups := zwave.CommonAssociationGroups()
//	for _, g := range groups {
//	    fmt.Printf("Group %d: %s (max %d nodes)\n", g.ID, g.Name, g.MaxNodes)
//	}
func CommonAssociationGroups() []AssociationGroup {
	return []AssociationGroup{
		{
			ID:       1,
			Name:     "Lifeline",
			MaxNodes: 1,
			Profile:  "Reports device status to the primary controller",
		},
		{
			ID:       2,
			Name:     "Basic Set",
			MaxNodes: 5,
			Profile:  "Sends Basic Set commands when switch state changes",
		},
		{
			ID:       3,
			Name:     "Multilevel",
			MaxNodes: 5,
			Profile:  "Sends Multilevel Switch commands for dimmer control",
		},
	}
}

// ConfigurationParameter represents a Z-Wave configuration parameter.
//
// Configuration parameters allow fine-tuning device behavior beyond
// what the standard command classes provide.
type ConfigurationParameter struct {
	CurrentValue *int
	Name         string
	Description  string
	Number       int
	Size         int
	DefaultValue int
	MinValue     int
	MaxValue     int
}

// CommonConfigParameters returns common configuration parameters for Wave switches.
//
// Note: Actual parameters vary by device model. Consult the device
// documentation for the complete parameter list.
//
// Example:
//
//	params := zwave.CommonConfigParameters()
//	for _, p := range params {
//	    fmt.Printf("Parameter %d: %s (default: %d)\n", p.Number, p.Name, p.DefaultValue)
//	}
func CommonConfigParameters() []ConfigurationParameter {
	return []ConfigurationParameter{
		{
			Number:       1,
			Name:         "Output Relay State After Power Failure",
			Description:  "Determines switch state after power restoration",
			Size:         1,
			DefaultValue: 0,
			MinValue:     0,
			MaxValue:     2,
		},
		{
			Number:       36,
			Name:         "Power Report Threshold",
			Description:  "Power change percentage to trigger report",
			Size:         1,
			DefaultValue: 10,
			MinValue:     0,
			MaxValue:     100,
		},
		{
			Number:       39,
			Name:         "Minimum Power Report Interval",
			Description:  "Minimum time between power reports (seconds)",
			Size:         2,
			DefaultValue: 30,
			MinValue:     0,
			MaxValue:     32767,
		},
		{
			Number:       91,
			Name:         "Water Alarm Response",
			Description:  "Action when water alarm is received",
			Size:         1,
			DefaultValue: 0,
			MinValue:     0,
			MaxValue:     2,
		},
		{
			Number:       120,
			Name:         "Factory Reset Confirmation",
			Description:  "Enable/disable factory reset via button",
			Size:         1,
			DefaultValue: 1,
			MinValue:     0,
			MaxValue:     1,
		},
	}
}
