package cli

import (
	"flag"
	"os"
)

// ServeCommand is an adapter for the existing serverCmd to work with the new Command interface
type ServeCommand struct{}

// Name returns the name of the command
func (c *ServeCommand) Name() string {
	return "serve"
}

// Description returns the description of the command
func (c *ServeCommand) Description() string {
	return "Start the Hugoverse API server (supports -env, -port, -https flags)"
}

// Run executes the command
func (c *ServeCommand) Run() error {
	// 创建一个 FlagSet 来模拟 parent
	parent := flag.NewFlagSet("hugoverse", flag.ExitOnError)
	
	// 将剩余参数设置为 parent 的参数
	// os.Args[0] = "hugoverse"
	// os.Args[1] = "serve"
	// os.Args[2:] = actual flags
	parent.Parse(os.Args[1:])
	
	// 使用现有的 NewServeCmd
	cmd, err := NewServeCmd(parent)
	if err != nil {
		return err
	}
	
	return cmd.Run()
}
