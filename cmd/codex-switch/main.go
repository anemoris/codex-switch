package main

import (
	"fmt"
	"os"

	"codex-switch/internal/cli"
	"codex-switch/internal/config"
	"codex-switch/internal/runner"
)

var version = "dev"

func main() {
	store, err := config.NewDefaultStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	app := cli.New(store, runner.RealRunner{}, version, os.Stdin, os.Stdout, os.Stderr)
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
