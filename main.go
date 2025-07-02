package main

import (
	"log"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
)

func main() {
	app := config.NewTldxContext()

	rootCmd := cmd.NewRootCmd(app)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}

}
