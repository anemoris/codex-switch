package cli

import (
	"fmt"

	"codex-switch/internal/config"
	"codex-switch/internal/profile"
)

func (a *App) runSyncSkills(args []string) error {
	name, rest := consumeOptionalName(args)

	fs := a.newFlagSet("sync-skills")
	var allProfiles bool
	var allSkills bool
	fs.BoolVar(&allProfiles, "all", false, "sync skills to every configured profile")
	fs.BoolVar(&allSkills, "all-skills", false, "sync every skill from ~/.codex/skills")

	if err := fs.Parse(rest); err != nil {
		return err
	}
	skillNames := fs.Args()

	if name != "" && allProfiles {
		return usageError("usage: codex-switch sync-skills [<profile>|--all] [<skill> ...|--all-skills]")
	}
	if name == "" && !allProfiles {
		return usageError("usage: codex-switch sync-skills [<profile>|--all] [<skill> ...|--all-skills]")
	}
	if allSkills && len(skillNames) != 0 {
		return usageError("usage: codex-switch sync-skills [<profile>|--all] [<skill> ...|--all-skills]")
	}
	if !allSkills && len(skillNames) == 0 {
		return usageError("usage: codex-switch sync-skills [<profile>|--all] [<skill> ...|--all-skills]")
	}

	cfg, err := a.store.Load()
	if err != nil {
		return err
	}

	if allProfiles {
		names := sortedProfileNames(cfg)
		if len(names) == 0 {
			return fmt.Errorf("no profiles configured")
		}
		for _, profileName := range names {
			if err := a.syncSkillsForProfile(profileName, cfg.Profiles[profileName], skillNames, allSkills); err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(a.stdout, "profiles_synced=%d\n", len(names))
		return err
	}

	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	return a.syncSkillsForProfile(name, p, skillNames, allSkills)
}

func (a *App) syncSkillsForProfile(name string, p config.Profile, skillNames []string, allSkills bool) error {
	result, err := profile.SyncSkills(p.CodexHome, skillNames, allSkills)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "Synced skills for profile %s\n", name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "skills=%d\n", len(result.Skills)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "source=%s\n", result.SourceRoot); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.stdout, "dest=%s\n", result.DestRoot)
	return err
}
