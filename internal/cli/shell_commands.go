package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"codex-switch/internal/config"
	"codex-switch/internal/shellinit"
)

func (a *App) runEnv(args []string) error {
	profileName, rest := consumeOptionalName(args)

	fs := a.newFlagSet("env")
	var shell string
	fs.StringVar(&shell, "shell", "sh", "shell format: sh, bash, zsh, fish")

	if err := fs.Parse(rest); err != nil {
		return err
	}
	if profileName == "" && fs.NArg() == 1 {
		profileName = fs.Arg(0)
	} else if fs.NArg() > 0 {
		return usageError("usage: codex-switch env [<profile>] [--shell sh|bash|zsh|fish]")
	}

	p, _, err := a.lookupProfileOrDefault(profileName)
	if err != nil {
		return err
	}
	out, err := renderEnv(shell, p.CodexHome)
	if err != nil {
		return err
	}
	_, err = io.WriteString(a.stdout, out)
	return err
}

func (a *App) runAliases(args []string) error {
	fs := a.newFlagSet("aliases")
	var shell string
	fs.StringVar(&shell, "shell", "zsh", "shell format: bash, zsh, fish")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return usageError("usage: codex-switch aliases [--shell bash|zsh|fish]")
	}

	cfg, err := a.store.Load()
	if err != nil {
		return err
	}
	out, err := renderAliases(shell, cfg)
	if err != nil {
		return err
	}
	_, err = io.WriteString(a.stdout, out)
	return err
}

func (a *App) runInitShell(args []string) error {
	fs := a.newFlagSet("init-shell")

	defaultShell := shellinit.DetectShell(os.Getenv("SHELL"))
	var shell string
	var rcFile string
	fs.StringVar(&shell, "shell", defaultShell, "shell format: bash, zsh, fish")
	fs.StringVar(&rcFile, "rc-file", "", "shell rc file path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return usageError("usage: codex-switch init-shell [--shell bash|zsh|fish] [--rc-file path]")
	}

	rcFile, err := resolveRCPath(shell, rcFile)
	if err != nil {
		return err
	}

	block, err := shellinit.ManagedBlock(shell)
	if err != nil {
		return err
	}
	changed, err := shellinit.Install(rcFile, block)
	if err != nil {
		return err
	}
	if err := a.recordShellInit(shell, rcFile); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(a.stdout, "Shell init installed for %s\n", shell); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "rc_file=%s\n", rcFile); err != nil {
		return err
	}
	if !changed {
		_, err = fmt.Fprintln(a.stdout, "status=unchanged")
		return err
	}
	_, err = fmt.Fprintf(a.stdout, "reload=source %s\n", shellinit.Quote(filepath.Clean(rcFile)))
	return err
}

func (a *App) runUninitShell(args []string) error {
	fs := a.newFlagSet("uninit-shell")

	defaultShell := shellinit.DetectShell(os.Getenv("SHELL"))
	var shell string
	var rcFile string
	fs.StringVar(&shell, "shell", defaultShell, "shell format: bash, zsh, fish")
	fs.StringVar(&rcFile, "rc-file", "", "shell rc file path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return usageError("usage: codex-switch uninit-shell [--shell bash|zsh|fish] [--rc-file path]")
	}

	rcFile, err := resolveRCPath(shell, rcFile)
	if err != nil {
		return err
	}

	changed, err := shellinit.Uninstall(rcFile)
	if err != nil {
		return err
	}
	if err := a.removeRecordedShellInit(shell, rcFile); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "Shell init removed for %s\n", shell); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "rc_file=%s\n", rcFile); err != nil {
		return err
	}
	if !changed {
		_, err = fmt.Fprintln(a.stdout, "status=unchanged")
		return err
	}
	_, err = fmt.Fprintf(a.stdout, "reload=source %s\n", shellinit.Quote(filepath.Clean(rcFile)))
	return err
}

func (a *App) runCleanup(args []string) error {
	fs := a.newFlagSet("cleanup")

	defaultShell := shellinit.DetectShell(os.Getenv("SHELL"))
	var shell string
	var rcFile string
	var purgeData bool
	fs.StringVar(&shell, "shell", defaultShell, "shell format: bash, zsh, fish")
	fs.StringVar(&rcFile, "rc-file", "", "shell rc file path")
	fs.BoolVar(&purgeData, "purge-data", false, "remove the codex-switch data directory")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return usageError("usage: codex-switch cleanup [--shell bash|zsh|fish] [--rc-file path] [--purge-data]")
	}

	rcFile, err := resolveRCPath(shell, rcFile)
	if err != nil {
		return err
	}

	root, err := config.RootDir()
	if err != nil {
		return err
	}
	if purgeData {
		if err := validatePurgeRoot(root); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(a.stdout, "data_root=%s\n", root); err != nil {
		return err
	}
	changed, err := shellinit.Uninstall(rcFile)
	if err != nil {
		return err
	}
	if err := a.removeRecordedShellInit(shell, rcFile); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "shell_init=%s\n", cleanupStatus(changed, "removed", "unchanged")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "shell=%s\n", shell); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "rc_file=%s\n", rcFile); err != nil {
		return err
	}
	if changed {
		if _, err := fmt.Fprintf(a.stdout, "reload=source %s\n", shellinit.Quote(filepath.Clean(rcFile))); err != nil {
			return err
		}
	}

	if purgeData {
		if err := os.RemoveAll(root); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(a.stdout, "data=purged"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(a.stdout, "data=preserved"); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(a.stdout, "binary=remove_manually")
	return err
}

func resolveRCPath(shell, rcFile string) (string, error) {
	if err := validateShell(shell); err != nil {
		return "", err
	}
	if rcFile == "" {
		return shellinit.DefaultRCPath(shell)
	}
	return config.ExpandPath(rcFile)
}

func validateShell(shell string) error {
	switch shell {
	case "bash", "zsh", "fish":
		return nil
	default:
		return fmt.Errorf("unsupported shell %q", shell)
	}
}

func (a *App) recordShellInit(shell, rcFile string) error {
	root := filepath.Dir(a.store.Path())
	state, err := shellinit.LoadState(root)
	if err != nil {
		return err
	}
	state.Add(shell, rcFile)
	return shellinit.SaveState(root, state)
}

func (a *App) removeRecordedShellInit(shell, rcFile string) error {
	root := filepath.Dir(a.store.Path())
	state, err := shellinit.LoadState(root)
	if err != nil {
		return err
	}
	state.Remove(shell, rcFile)
	return shellinit.SaveState(root, state)
}

func cleanupStatus(changed bool, changedValue, unchangedValue string) string {
	if changed {
		return changedValue
	}
	return unchangedValue
}

func validatePurgeRoot(root string) error {
	root = filepath.Clean(root)
	if !filepath.IsAbs(root) {
		return fmt.Errorf("refusing to purge non-absolute data root %q", root)
	}
	if root == string(filepath.Separator) {
		return fmt.Errorf("refusing to purge filesystem root %q", root)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	home = filepath.Clean(home)
	if root == home {
		return fmt.Errorf("refusing to purge home directory %q", root)
	}

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("data root %q is not a directory", root)
	}

	markerPath := config.RootMarkerPath(root)
	if markerInfo, err := os.Stat(markerPath); err == nil && !markerInfo.IsDir() {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	legacyConfigPath := filepath.Join(root, "config.json")
	if legacyInfo, err := os.Stat(legacyConfigPath); err == nil && !legacyInfo.IsDir() {
		if ok, err := config.IsLegacyConfigFile(legacyConfigPath); err != nil {
			return err
		} else if ok {
			return nil
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	return fmt.Errorf("refusing to purge %q because it is missing the %s marker", root, config.RootMarkerName)
}

func renderEnv(shell, codexHome string) (string, error) {
	switch shell {
	case "sh", "bash", "zsh":
		return fmt.Sprintf("export CODEX_HOME=%s\n", shellinit.Quote(codexHome)), nil
	case "fish":
		return fmt.Sprintf("set -gx CODEX_HOME %s\n", shellinit.Quote(codexHome)), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

func renderAliases(shell string, cfg config.Config) (string, error) {
	var b strings.Builder
	for _, name := range sortedProfileNames(cfg) {
		p := cfg.Profiles[name]
		if p.Shortcut == "" {
			continue
		}
		switch shell {
		case "bash", "zsh":
			fmt.Fprintf(&b, "alias %s=\"codex-switch run %s -- codex\"\n", p.Shortcut, shellinit.Quote(name))
		case "fish":
			fmt.Fprintf(&b, "alias %s \"codex-switch run %s -- codex\"\n", p.Shortcut, shellinit.Quote(name))
		default:
			return "", fmt.Errorf("unsupported shell %q", shell)
		}
	}
	return b.String(), nil
}
