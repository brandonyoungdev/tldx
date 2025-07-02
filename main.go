package main

import (
	"log"
	"os"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
)

func main() {
	ctx := &config.TldxContext{
		Config: &config.TldxConfigOptions{},
	}

	rootCmd := cmd.NewRootCmd(ctx)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

}
