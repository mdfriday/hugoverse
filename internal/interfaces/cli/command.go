package cli

// Command represents a CLI command
type Command interface {
	// Name returns the name of the command
	Name() string
	// Description returns the description of the command
	Description() string
	// Run executes the command
	Run() error
}
