package cli

import (
	"github.com/mdfriday/hugoverse/internal/interfaces/cli/server"
	"github.com/mdfriday/hugoverse/internal/interfaces/cli/sse"
)

// GetCommands returns all available commands
func GetCommands() []Command {
	return []Command{
		&ServeCommand{}, // Legacy: hugoverse serve -env prod -port 1314
		&server.Command{}, // New: hugoverse server (config from ENV)
		&sse.Command{},
	}
}
