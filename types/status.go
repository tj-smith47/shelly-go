package types

// Status represents the current status of a component or device.
// This is a marker interface - specific components define their own status types.
type Status interface {
	// IsHealthy returns true if the component is operating normally.
	IsHealthy() bool
}

// CommonStatus contains status fields common to many components.
type CommonStatus struct {
	RawFields `json:"-"`
	ID        int `json:"id"`
}

// IsHealthy implements the Status interface.
// Default implementation returns true - components can override.
func (c *CommonStatus) IsHealthy() bool {
	return true
}
