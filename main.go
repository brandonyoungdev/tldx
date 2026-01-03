package main

import (
	"context"
	"os"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/charmbracelet/fang"
)

func main() {
	app := config.NewTldxContext()

	rootCmd := cmd.NewRootCmd(app)

	// Use Fang for graceful shutdown handling
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithNotifySignal(os.Interrupt),
	); err != nil {
		os.Exit(1)
	}
}
