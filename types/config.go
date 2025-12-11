package types

// Config represents a component or device configuration.
// This is a marker interface - specific components define their own config types.
type Config interface {
	// Validate validates the configuration and returns an error if invalid.
	Validate() error
}

// CommonConfig contains configuration fields common to many components.
type CommonConfig struct {
	Name      *string `json:"name,omitempty"`
	RawFields `json:"-"`
	ID        int `json:"id"`
}

// Validate implements the Config interface.
func (c *CommonConfig) Validate() error {
	// No validation needed for common config
	return nil
}
