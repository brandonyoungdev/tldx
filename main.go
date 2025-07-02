package main

import (
	"log"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
)

func main() {
	ctx := config.NewTldxContext()

	rootCmd := cmd.NewRootCmd(ctx)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}

}
