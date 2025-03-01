package main

import (
	"fmt"
	"os"

	"github.com/mdfriday/hugoverse/internal/interfaces/cli"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Available commands:")
		for _, cmd := range cli.GetCommands() {
			fmt.Printf("  %s\t%s\n", cmd.Name(), cmd.Description())
		}
		os.Exit(1)
	}

	cmdName := os.Args[1]
	for _, cmd := range cli.GetCommands() {
		if cmd.Name() == cmdName {
			if err := cmd.Run(); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	fmt.Printf("Unknown command: %s\n", cmdName)
	os.Exit(1)
}
