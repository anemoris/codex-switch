package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"codex-switch/internal/config"
	"codex-switch/internal/profile"
)

func (a *App) execProfileCommand(commandName string, args []string, defaultCommand []string) error {
	profileName, commandArgs, err := a.parseProfileCommandArgs(commandName, args, defaultCommand)
	if err != nil {
		return err
	}
	p, err := a.lookupProfile(profileName)
	if err != nil {
		return err
	}
	return a.runCommandInProfile(p, commandArgs)
}

func (a *App) parseProfileCommandArgs(commandName string, args []string, defaultCommand []string) (string, []string, error) {
	return a.parseProfileCommandArgsWithFlags(commandName, args, defaultCommand, nil)
}

// parseProfileCommandArgsWithFlags resolves a profile name and command
// arguments from the given args slice. The profile name can come from three
// sources, checked in this order:
//
//  1. A leading positional arg that doesn't start with "-" (consumed by
//     consumeOptionalName before flag parsing).
//  2. The --profile flag.
//  3. A positional arg that remains after flag parsing (only when no "--"
//     separator was seen before flags).
//
// If no profile name is resolved from any source, it falls back to the
// configured default profile. The command arguments are everything after
// the "--" separator (or defaultCommand if nothing follows).
func (a *App) parseProfileCommandArgsWithFlags(commandName string, args []string, defaultCommand []string, registerFlags func(fs *flag.FlagSet)) (string, []string, error) {
	profileName, rest := consumeOptionalName(args)

	fs := a.newFlagSet(commandName)
	fs.StringVar(&profileName, "profile", profileName, "profile name; falls back to default profile")
	if registerFlags != nil {
		registerFlags(fs)
	}
	startsWithCommandSeparator := len(rest) > 0 && rest[0] == "--"

	if err := fs.Parse(rest); err != nil {
		return "", nil, err
	}

	parsedRest := fs.Args()
	if profileName == "" && !startsWithCommandSeparator && len(parsedRest) > 0 && parsedRest[0] != "--" {
		profileName = parsedRest[0]
		parsedRest = parsedRest[1:]
	}

	commandArgs := normalizeCommandArgs(parsedRest)
	if len(commandArgs) == 0 {
		commandArgs = append([]string(nil), defaultCommand...)
	}

	_, resolvedName, err := a.lookupProfileOrDefault(profileName)
	if err != nil {
		return "", nil, err
	}
	return resolvedName, commandArgs, nil
}

func (a *App) lookupProfile(name string) (config.Profile, error) {
	p, _, err := a.lookupProfileOrDefault(name)
	return p, err
}

func (a *App) lookupProfileOrDefault(name string) (config.Profile, string, error) {
	cfg, err := a.store.Load()
	if err != nil {
		return config.Profile{}, "", err
	}
	if name == "" {
		if cfg.DefaultProfile == "" {
			return config.Profile{}, "", errors.New("no profile specified and no default profile configured")
		}
		name = cfg.DefaultProfile
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		return config.Profile{}, "", fmt.Errorf("profile %q not found", name)
	}
	return p, name, nil
}

func (a *App) runCommandInProfile(p config.Profile, commandArgs []string) error {
	if err := profile.EnsureHome(p.CodexHome); err != nil {
		return err
	}

	if _, err := exec.LookPath(commandArgs[0]); err != nil {
		return fmt.Errorf("command %q not found in PATH", commandArgs[0])
	}

	cmd := exec.Command(commandArgs[0], commandArgs[1:]...)
	cmd.Stdin = a.stdin
	cmd.Stdout = a.stdout
	cmd.Stderr = a.stderr
	cmd.Env = append(filteredEnv(os.Environ(), "CODEX_HOME="), "CODEX_HOME="+p.CodexHome)
	return a.runner.Run(cmd)
}

func filteredEnv(env []string, prefix string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return out
}

func (a *App) newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	return fs
}

func usageError(message string) error {
	return errors.New(message)
}

func sortedProfileNames(cfg config.Config) []string {
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func consumeOptionalName(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return "", args
}

func normalizeCommandArgs(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}
	return args
}
