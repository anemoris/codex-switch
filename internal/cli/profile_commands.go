package cli

import (
	"fmt"
	"strings"

	"codex-switch/internal/config"
	"codex-switch/internal/profile"
)

func (a *App) runAdd(args []string) error {
	name, rest := consumeOptionalName(args)

	fs := a.newFlagSet("add")

	var codexHome string
	var label string
	var shortcut string
	var makeDefault bool
	fs.StringVar(&codexHome, "home", "", "profile CODEX_HOME path")
	fs.StringVar(&label, "label", "", "display label")
	fs.StringVar(&shortcut, "shortcut", "", "shortcut command name")
	fs.BoolVar(&makeDefault, "default", false, "set as default profile")

	if err := fs.Parse(rest); err != nil {
		return err
	}
	if name == "" {
		if fs.NArg() != 1 {
			return usageError("usage: codex-switch add <name> [--home path] [--label text] [--shortcut command] [--default]")
		}
		name = fs.Arg(0)
	} else if fs.NArg() != 0 {
		return usageError("usage: codex-switch add <name> [--home path] [--label text] [--shortcut command] [--default]")
	}

	if err := config.ValidateProfileName(name); err != nil {
		return err
	}
	if shortcut != "" {
		if err := config.ValidateShortcut(shortcut); err != nil {
			return err
		}
	}

	cfg, err := a.store.Load()
	if err != nil {
		return err
	}
	if _, exists := cfg.Profiles[name]; exists {
		return fmt.Errorf("profile %q already exists", name)
	}

	if codexHome == "" {
		codexHome, err = config.DefaultProfileHome(name)
		if err != nil {
			return err
		}
	} else {
		codexHome, err = config.ExpandPath(codexHome)
		if err != nil {
			return err
		}
	}

	cfg.Profiles[name] = config.Profile{
		Label:     label,
		CodexHome: codexHome,
		Shortcut:  shortcut,
	}
	if makeDefault || cfg.DefaultProfile == "" {
		cfg.DefaultProfile = name
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}
	if err := a.store.Save(cfg); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(a.stdout, "Added profile %s\n", name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "CODEX_HOME=%s\n", codexHome); err != nil {
		return err
	}
	if shortcut != "" {
		_, err = fmt.Fprintf(a.stdout, "Shortcut %s is available via `codex-switch aliases`\n", shortcut)
		return err
	}
	return nil
}

func (a *App) runList(args []string) error {
	if len(args) != 0 {
		return usageError("usage: codex-switch list")
	}

	cfg, err := a.store.Load()
	if err != nil {
		return err
	}

	names := sortedProfileNames(cfg)
	if len(names) == 0 {
		_, err := fmt.Fprintln(a.stdout, "No profiles configured.")
		return err
	}

	for _, name := range names {
		p := cfg.Profiles[name]
		markers := make([]string, 0, 3)
		if cfg.DefaultProfile == name {
			markers = append(markers, "default")
		}
		if p.Shortcut != "" {
			markers = append(markers, "shortcut="+p.Shortcut)
		}
		authSummary, authErr := profile.ReadAuthSummary(p.CodexHome)
		if authSummary.Present {
			markers = append(markers, "auth=present")
			if authSummary.Email != "" {
				markers = append(markers, "email="+authSummary.Email)
			}
			if authErr != nil {
				markers = append(markers, "auth_info=unreadable")
			}
		} else {
			markers = append(markers, "auth=missing")
		}

		line := name
		if p.Label != "" {
			line += " (" + p.Label + ")"
		}
		line += " [" + strings.Join(markers, ", ") + "]"
		line += "\n  CODEX_HOME=" + p.CodexHome
		line += "\n  USAGE_SNAPSHOT=" + formatUsageSnapshot(profile.ReadUsageSnapshot(p.CodexHome))
		if _, err := fmt.Fprintln(a.stdout, line); err != nil {
			return err
		}
	}

	return nil
}

func formatUsageSnapshot(snapshot profile.UsageSnapshot) string {
	if !snapshot.Present {
		return "unavailable"
	}
	primaryLabel, ok := formatUsageWindow(snapshot.PrimaryWindowMinutes)
	if !ok {
		return "unavailable"
	}
	secondaryLabel, ok := formatUsageWindow(snapshot.SecondaryWindowMinutes)
	if !ok {
		return "unavailable"
	}

	planType := snapshot.PlanType
	if planType == "" {
		planType = "unknown"
	}

	return fmt.Sprintf("%s=%d%% %s=%d%% plan=%s", primaryLabel, snapshot.PrimaryUsedPercent, secondaryLabel, snapshot.SecondaryUsedPercent, planType)
}

func formatUsageWindow(minutes int) (string, bool) {
	if minutes <= 0 {
		return "", false
	}
	if minutes%1440 == 0 {
		return fmt.Sprintf("%dd", minutes/1440), true
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%dh", minutes/60), true
	}
	return fmt.Sprintf("%dm", minutes), true
}

func (a *App) runShow(args []string) error {
	if len(args) != 1 {
		return usageError("usage: codex-switch show <name>")
	}

	name := args[0]
	cfg, err := a.store.Load()
	if err != nil {
		return err
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	if _, err := fmt.Fprintf(a.stdout, "name=%s\n", name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "codex_home=%s\n", p.CodexHome); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "auth_path=%s\n", profile.AuthPath(p.CodexHome)); err != nil {
		return err
	}
	authSummary, authErr := profile.ReadAuthSummary(p.CodexHome)
	if _, err := fmt.Fprintf(a.stdout, "auth_present=%t\n", authSummary.Present); err != nil {
		return err
	}
	if authSummary.Present {
		if _, err := fmt.Fprintf(a.stdout, "auth_email=%s\n", authSummary.Email); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(a.stdout, "auth_account_id=%s\n", authSummary.AccountID); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(a.stdout, "auth_name=%s\n", authSummary.Name); err != nil {
			return err
		}
		if authErr != nil {
			if _, err := fmt.Fprintf(a.stdout, "auth_info_error=%s\n", authErr); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(a.stdout, "label=%s\n", p.Label); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "shortcut=%s\n", p.Shortcut); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.stdout, "default=%t\n", cfg.DefaultProfile == name)
	return err
}

func (a *App) runRemove(args []string) error {
	if len(args) != 1 {
		return usageError("usage: codex-switch remove <name>")
	}

	name := args[0]
	cfg, err := a.store.Load()
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(cfg.Profiles, name)
	if cfg.DefaultProfile == name {
		cfg.DefaultProfile = ""
		names := sortedProfileNames(cfg)
		if len(names) > 0 {
			cfg.DefaultProfile = names[0]
		}
	}

	if err := a.store.Save(cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.stdout, "Removed profile %s\n", name)
	return err
}

func (a *App) runSetDefault(args []string) error {
	if len(args) != 1 {
		return usageError("usage: codex-switch set-default <name>")
	}

	name := args[0]
	cfg, err := a.store.Load()
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	cfg.DefaultProfile = name
	if err := a.store.Save(cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.stdout, "Default profile set to %s\n", name)
	return err
}
