package cli

import (
	"github.com/mdfriday/hugoverse/internal/interfaces/cli/sse"
)

// GetCommands returns all available commands
func GetCommands() []Command {
	return []Command{
		&sse.Command{},
	}
}
