package cli

import (
	"fmt"
	"io"

	"codex-switch/internal/config"
	"codex-switch/internal/runner"
)

type App struct {
	store   *config.Store
	runner  runner.Runner
	version string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

func New(store *config.Store, cmdRunner runner.Runner, version string, stdin io.Reader, stdout, stderr io.Writer) *App {
	return &App{
		store:   store,
		runner:  cmdRunner,
		version: normalizeVersion(version),
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
	}
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		return a.printHelp()
	}

	switch args[0] {
	case "add":
		return a.runAdd(args[1:])
	case "list":
		return a.runList(args[1:])
	case "show":
		return a.runShow(args[1:])
	case "status":
		return a.runStatus(args[1:])
	case "doctor":
		return a.runDoctor(args[1:])
	case "remove":
		return a.runRemove(args[1:])
	case "set-default":
		return a.runSetDefault(args[1:])
	case "run":
		return a.runExec(args[1:])
	case "login":
		return a.runLogin(args[1:])
	case "import-auth":
		return a.runImportAuth(args[1:])
	case "env":
		return a.runEnv(args[1:])
	case "aliases":
		return a.runAliases(args[1:])
	case "init-shell":
		return a.runInitShell(args[1:])
	case "uninit-shell":
		return a.runUninitShell(args[1:])
	case "cleanup":
		return a.runCleanup(args[1:])
	case "help", "-h", "--help":
		return a.printHelp()
	case "version", "--version", "-v":
		_, err := fmt.Fprintln(a.stdout, a.version)
		return err
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
